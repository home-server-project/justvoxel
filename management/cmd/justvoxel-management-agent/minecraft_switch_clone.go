package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Use the same immutable reservations, dual-stack binding and cross-play port
// checks as instance creation. Stopped servers reserve ports too.
func switchClonePorts(records []instanceRuntimeRecord, c *minecraftInstanceRuntimeConfig) error {
	used := map[int]bool{}
	for _, r := range records {
		used[r.Runtime.JavaPort] = true
		if r.Runtime.BedrockEnabled {
			used[r.Runtime.BedrockPort] = true
		}
		if r.Runtime.Software == "bedrock" {
			used[r.Runtime.BedrockTCPPort] = true
			for port := r.Runtime.BedrockUDPStart; port <= r.Runtime.BedrockUDPEnd && port > 0; port++ {
				used[port] = true
			}
		}
	}
	allocate := func(start int, crossplay bool) (int, error) {
		for port := start; port <= 65535; port++ {
			if used[port] || port == 25575 {
				continue
			}
			listeners, err := instanceHoldPort(port)
			if err != nil {
				continue
			}
			for _, listener := range listeners {
				listener.Close()
			}
			if crossplay {
				ports, err := instanceHoldCrossplayPort(port)
				if err != nil {
					continue
				}
				ports.Close()
			}
			used[port] = true
			return port, nil
		}
		return 0, instanceRuntimeFailure("No free server ports could be allocated.")
	}
	var err error
	c.BedrockPort = 0
	c.JavaPort, err = allocate(25565, false)
	if err != nil {
		return err
	}
	if c.BedrockEnabled {
		c.BedrockPort, err = allocate(19132, true)
	}
	return err
}

func switchCloneName(instances []minecraftRegisteredInstance, source, target string) string {
	base := "Minecraft"
	for _, instance := range instances {
		if instance.ID == source {
			base = instance.DisplayName
			break
		}
	}
	software, _ := javaServerSoftware(target)
	// Leave room for the software name and a collision suffix, in Unicode runes.
	runes := []rune(base)
	if len(runes) > 35 {
		base = string(runes[:35])
	}
	base += " " + software.Name
	for n := 1; ; n++ {
		name := base
		if n > 1 {
			name += " " + strconv.Itoa(n)
		}
		if !instanceNameUsed(instances, "", name) {
			return name
		}
	}
}

func (s *server) cloneMinecraftSwitch(ctx context.Context, e *minecraftSwitchEvidence, p minecraftSwitchPlan, lock *pluginMaintenanceLock) (result error) {
	b := instanceRuntimes
	source, err := b.readRecord(e.Source)
	if err != nil {
		return err
	}
	if source.NetworkMember {
		return instanceRuntimeFailure("Remove the source from Minecraft Network before cloning its standalone settings.")
	}
	target, snapshot, err := b.prepareSwitchTarget(ctx, source, p, true)
	if err != nil {
		return err
	}
	if _, err = b.backupStorage(ctx, "preflight", e.Source); err != nil {
		return err
	}
	state, err := b.observe(ctx, e.Source)
	if err != nil || (state.State != "running" && state.State != "stopped") || p.SourceUninitialized && state.State != "stopped" {
		return errInstanceBusy
	}
	e.OriginallyRunning = state.State == "running"
	e.Stage = "source_copy"
	// Intent is durable BEFORE shutdown, including power loss during countdown.
	e.SourceStopped = e.OriginallyRunning
	e.SourceContentChanged = !p.SourceUninitialized
	if err = saveSwitchEvidence(*e); err != nil {
		return err
	}
	if e.OriginallyRunning {
		if err = b.actionWithWarnings(ctx, e.Source, "stop", true); err != nil {
			return err
		}
	}
	if p.SourceUninitialized {
		err = b.requireSwitchInitialization(ctx, source, p)
	} else {
		err = b.prepareSwitchSourceCopy(ctx, e.Source, p.Version)
	}
	if err != nil {
		return err
	}
	before, inventoryErr := b.backupStorage(ctx, "inventory", e.Source)
	var existing []instanceBackupMetadata
	if inventoryErr != nil || json.Unmarshal(before, &existing) != nil {
		return instanceRuntimeFailure("Source backup inventory could not be verified.")
	}
	_, backupErr := b.backupStorage(ctx, "create", e.Source)
	var archive instanceBackupMetadata
	if backupErr == nil {
		after, readErr := b.backupStorage(ctx, "inventory", e.Source)
		var inventory []instanceBackupMetadata
		if readErr != nil || json.Unmarshal(after, &inventory) != nil {
			backupErr = errInstanceRecovery
		} else {
			added := 0
			for _, candidate := range inventory {
				known := false
				for _, old := range existing {
					if old.Archive == candidate.Archive {
						known = true
						break
					}
				}
				if !known {
					archive = candidate
					added++
				}
			}
			if added != 1 {
				backupErr = errInstanceRecovery
			}
		}
	}
	if backupErr == nil {
		stoppedRecord, readErr := b.readRecord(e.Source)
		if readErr != nil {
			backupErr = readErr
		} else {
			_, contentHash, contentErr := switchContentEvidence(ctx, stoppedRecord, "", p.Version)
			if !p.SourceUninitialized && switchNeedsWorldMove(p.Current, p.Target) {
				moves, layoutErr := inspectSwitchWorld(stoppedRecord.Storage.DataPath, p.Version)
				if layoutErr != nil || instanceHash(mustSwitchJSON(moves)) != p.WorldFingerprint {
					contentErr = errInstanceRecovery
				}
			}
			if contentErr != nil || contentHash != p.ContentFingerprint {
				backupErr = instanceRuntimeFailure("Plugins & Mods changed before the consistent copy. Review Switch again.")
			}
		}
	}
	if p.SourceUninitialized && backupErr == nil {
		backupErr = b.requireSwitchInitialization(ctx, source, p)
	}
	// Restore the original promptly, even when backup creation failed. Persisted
	// intent remains set until restart AND Minecraft readiness have been verified.
	if err = s.restoreSwitchSource(ctx, e, lock); err != nil {
		return errors.Join(backupErr, err)
	}
	if backupErr != nil {
		return backupErr
	}
	if archive.ID != e.Source || archive.Version != p.Version || archive.Software != source.Runtime.Software || !validInstanceRestoreArchive(archive.Archive) {
		return instanceRuntimeFailure("A consistent exact-version source backup could not be verified.")
	}
	names, err := b.identities().listInstances("")
	if err != nil {
		return err
	}
	name := switchCloneName(names, e.Source, p.Target)
	_, err = b.createTrackedCandidate(ctx, target.Runtime, name, snapshot, func(record instanceRuntimeRecord) error {
		if record.Runtime.ID != target.Runtime.ID || record.Runtime.ID == e.Source {
			return errInstanceID
		}
		if child, ok := ctx.Value(switchMatrixChildKey{}).(operationJournal); ok {
			if err := s.operations.reserveSwitchMatrixFixture(child.MatrixParent, record, child, name); err != nil {
				return err
			}
		}
		e.CloneID = record.Runtime.ID
		e.CloneRecord = &record
		e.Stage = "creating_clone"
		return saveSwitchEvidence(*e)
	})
	if err != nil {
		return err
	}
	e.Stage = "copying_clone"
	if err = saveSwitchEvidence(*e); err != nil {
		return err
	}
	// Reuse the existing restore engine's archive validation, bounded extraction,
	// capacity checks, inode-bound activation and rollback. Source metadata keeps
	// its true immutable ID; this internal mode never rewrites archive ownership.
	out, err := b.switchCloneRestoreStorage(ctx, "plan", e.CloneID, e.Source, archive.Archive, "", p.InitializationFingerprint)
	if err != nil {
		return err
	}
	var inspection instanceRestoreInspection
	if instanceDecode(out, &inspection) != nil || inspection.Metadata != archive || inspection.CurrentSoftware != p.Target || inspection.CurrentVersion != p.Version ||
		inspection.Integrity != "verified" || !instanceSHA256Pattern.MatchString(inspection.Fingerprint) {
		return errInstanceRecovery
	}
	plan := instanceRestorePlan{instanceRestoreInspection: inspection, RuntimeState: "stopped", CloneSourceID: e.Source, CloneInitialization: p.InitializationFingerprint}
	claim := &instanceRestoreClaim{s: s, maintenance: lock.file}
	err = s.runInstanceRestore(ctx, e.CloneID, plan, claim)
	// runInstanceRestore owns this descriptor during rollback/start handoffs.
	lock.file = claim.maintenance
	if err != nil {
		return err
	}
	cloneData, err := b.switchCloneContentRoot(ctx, *e, archive.Archive)
	if err != nil {
		return err
	}
	expectedContent, manifestHash, contentErr := applySwitchCloneContent(cloneData, e.OperationID, p)
	cloneData.Close()
	if contentErr != nil {
		return contentErr
	}
	s.operations.recordSwitchCloneExclusions(*e, p)
	if len(p.CrossplayInstall) != 0 {
		e.CloneCrossplayInstall = true
		if err = saveSwitchEvidence(*e); err != nil {
			return err
		}
		if err = b.installSwitchCloneCrossplay(ctx, *e, p); err != nil {
			return err
		}
	}
	if !p.SourceUninitialized && p.Current == "leaf" {
		if err = verifySwitchLeafWorld(target.Storage.DataPath); err != nil {
			return err
		}
	}
	if !p.SourceUninitialized && switchNeedsWorldMove(p.Current, p.Target) {
		if err = transformSwitchWorld(target.Storage.DataPath, p.Version); err != nil {
			return err
		}
	}
	if p.Target == "pufferfish" {
		if err = verifySwitchPufferfishCache(target.Storage.DataPath); err != nil {
			return err
		}
	}
	if _, err = verifySwitchCloneContent(target.Storage.DataPath, manifestHash, expectedContent, p.CrossplayInstall); err != nil {
		return err
	}
	e.Stage = "verifying_clone"
	if err = saveSwitchEvidence(*e); err != nil {
		return err
	}
	launched := time.Now()
	if err = lock.start(ctx, e.CloneID); err != nil {
		return err
	}
	session, err := s.confirmSwitchStartupMigration(ctx, *e, p, *e.CloneRecord, archive.Archive, launched)
	if err != nil {
		return err
	}
	if err = b.verifySwitchCloneRuntime(ctx, *e, p); err != nil {
		return err
	}

	if err = b.verifySwitchStartupSession(ctx, *e.CloneRecord, session); err != nil {
		return err
	}

	// A new clone is left stopped after meaningful runtime verification, avoiding
	// accidental simultaneous public servers or an unreviewed boot preference.
	if err = b.actionWithWarnings(ctx, e.CloneID, "stop", true); err != nil {
		return err
	}
	if p.SourceUninitialized {
		if err = verifySwitchInitializedWorld(target.Storage.DataPath); err != nil {
			return instanceRuntimeFailure("The new destination world could not be verified after initialization.")
		}
	}
	if e.CloneContentFingerprint, err = verifySwitchCloneContent(target.Storage.DataPath, manifestHash, expectedContent, p.CrossplayInstall); err != nil {
		return err
	}
	if target.Runtime.BedrockEnabled && switchCrossplayPlatform(p.Target) == "spigot" {
		if err = b.verifySwitchPaperCrossplay(ctx, target, p.ManagedArtifacts); err != nil {
			return errors.Join(err, &pluginRuntimeVerificationError{switchStatus: switchVerificationCrossplay})
		}
	}
	if err = s.clearSwitchSourceContent(ctx, e); err != nil {
		return err
	}
	restoredSource, err := b.readRecord(e.Source)
	if err != nil || restoredSource.Runtime != source.Runtime || restoredSource.Storage != source.Storage ||
		!reflect.DeepEqual(restoredSource.Backups, source.Backups) || !slices.Equal(restoredSource.Firewall, source.Firewall) {
		return instanceRuntimeFailure("The original source configuration could not be verified after copying.")
	}
	e.Stage = "complete"
	return saveSwitchEvidence(*e)
}

func (b *instanceRuntimeBackend) verifySwitchCloneRuntime(ctx context.Context, e minecraftSwitchEvidence, p minecraftSwitchPlan) error {
	record, err := b.readRecord(e.CloneID)
	if err != nil || e.CloneRecord == nil || e.CloneID == e.Source || record.Runtime.ID != e.CloneID ||
		record.Runtime.Software != p.Target || record.Runtime.Version != p.Version || record.Runtime.VersionPolicy != "pinned" ||
		record.Runtime.ImageTag != p.Runtime.ImageTag || record.Runtime.ImageDigest != p.RuntimeDigest ||
		record.Runtime.LoaderVersion != p.Runtime.LoaderVersion || record.Runtime.PufferfishBuild != p.Runtime.PufferfishBuild ||
		record.Runtime.BedrockEnabled != e.CloneRecord.Runtime.BedrockEnabled || !record.Runtime.PreserveContent ||
		record.NetworkMember || record.Storage != e.CloneRecord.Storage {
		return errors.Join(errInstanceRecovery, &pluginRuntimeVerificationError{switchStatus: switchVerificationConfiguration})
	}
	if err := b.verifyLive(record); err != nil {
		return errors.Join(err, &pluginRuntimeVerificationError{switchStatus: switchVerificationConfiguration})
	}
	if err := verifyPluginRuntimeMode(ctx, e.CloneID, p.Version, "--switch"); err != nil {
		return err
	}
	state, err := b.observeRaw(ctx, e.CloneID)
	if err != nil {
		return errors.Join(errInstanceRecovery, &pluginRuntimeVerificationError{switchStatus: switchVerificationInspection})
	}
	if state.State != "running" {
		return errors.Join(errInstanceRecovery, &pluginRuntimeVerificationError{switchStatus: switchVerificationSession})
	}
	if _, err := b.players(ctx, e.CloneID); err != nil {
		return errors.Join(err, &pluginRuntimeVerificationError{switchStatus: switchVerificationRCON})
	}
	expected, imageErr := b.command(ctx, "podman", "image", "inspect", instanceRuntimeImage(record.Runtime), "--format", "{{.Id}}")
	actual, runningErr := b.command(ctx, "podman", "inspect", "minecraft-"+e.CloneID, "--format", "{{.Image}}")
	if imageErr != nil || runningErr != nil || strings.TrimSpace(string(expected)) == "" || strings.TrimSpace(string(expected)) != strings.TrimSpace(string(actual)) {
		return errors.Join(errInstanceRecovery, &pluginRuntimeVerificationError{switchStatus: switchVerificationImage})
	}
	return nil
}

func (b *instanceRuntimeBackend) switchCloneRestoreStorage(ctx context.Context, action, id, source, archive, fingerprint, initialization string) ([]byte, error) {
	if !instanceIDPattern.MatchString(id) || !instanceIDPattern.MatchString(source) || id == source || (action != "plan" && action != "prepare") {
		return nil, errInstanceID
	}
	args := []string{"--mount=/proc/1/ns/mnt", "--", instanceRestoreHelper, action, id, archive, fingerprint, "false", source}
	if initialization != "" {
		if !instanceSHA256Pattern.MatchString(initialization) {
			return nil, errInstanceRecovery
		}
		args = append(args, initialization)
	}
	out, err := instanceBackupStorageCommand(ctx, "nsenter", args...)
	if err != nil {
		return nil, switchCloneRestoreFailure(out)
	}
	return out, nil
}

func switchCloneRestoreFailure(out []byte) error {
	const message = "The consistent source archive could not be safely staged into the new clone. The original server is preserved."
	var refusal struct {
		Error string `json:"error"`
	}
	if len(out) <= 256 && instanceDecode(out, &refusal) == nil {
		if reason := restoreValidationMessage(refusal.Error); reason != "" {
			return &instanceRestoreValidationError{code: refusal.Error, cause: instanceRuntimeFailure(message + " " + reason)}
		}
		var reason string
		switch refusal.Error {
		case "invalid_clone_identity":
			reason = "The source and clone runtime identities or storage bindings could not be verified."
		case "invalid_metadata_binding":
			reason = "The backup metadata does not match the source server or archive."
		case "invalid_archive_integrity":
			reason = "The backup archive or world metadata failed integrity validation."
		case "invalid_loader_evidence":
			reason = "The backup loader version could not be verified against the source server."
		case "incompatible_software_version", "unverified_version":
			reason = "The source and target could not be verified at the same exact Minecraft version."
		case "insufficient_staging_capacity":
			reason = "Clone storage has insufficient capacity for staging and rollback."
		case "stale_plan":
			reason = "The clone restore plan changed before staging."
		case "recovery_required":
			reason = "An interrupted restore requires recovery before staging."
		}
		if reason != "" {
			return instanceRuntimeFailure(message + " " + reason)
		}
	}
	return instanceRuntimeFailure(message)
}

func (s *server) restoreSwitchSource(ctx context.Context, e *minecraftSwitchEvidence, lock *pluginMaintenanceLock) error {
	if !e.SourceStopped && !e.OriginallyRunning {
		return nil
	}
	b := instanceRuntimes
	record, err := b.readRecord(e.Source)
	if err != nil || e.SourceRecord == nil || record.Runtime.ID != e.SourceRecord.Runtime.ID ||
		record.Storage != e.SourceRecord.Storage || record.Runtime.StartOnBoot != e.SourceRecord.Runtime.StartOnBoot ||
		record.Runtime.UID != e.SourceRecord.Runtime.UID || record.Runtime.GID != e.SourceRecord.Runtime.GID ||
		(record.Runtime.Version != e.Version && record.Runtime.Version != "LATEST") ||
		record.Runtime.Software != e.SourceRecord.Runtime.Software ||
		record.Runtime.LoaderVersion != e.SourceRecord.Runtime.LoaderVersion || record.Runtime.PufferfishBuild != e.SourceRecord.Runtime.PufferfishBuild ||
		record.Runtime.ImageTag != e.SourceRecord.Runtime.ImageTag || record.Runtime.ImageDigest != e.SourceRecord.Runtime.ImageDigest {
		return errInstanceRecovery
	}
	state, err := b.observe(ctx, e.Source)
	if err != nil {
		return err
	}
	if state.State == "stopped" || state.State == "failed" && state.resettableFailure {
		// A reboot can stop the source even after the original restart was
		// recorded. Pin moving definitions to the transaction's exact game
		// before starting, then restore the saved policy through normal cleanup.
		if record.Runtime.Version == "LATEST" {
			e.SourceStopped, e.SourceContentChanged = true, true
			if err = saveSwitchEvidence(*e); err != nil {
				return err
			}
			if err = b.prepareSwitchSourceCopy(ctx, e.Source, e.Version); err != nil {
				return err
			}
		}
		if err = lock.start(ctx, e.Source); err != nil {
			return err
		}
	} else if state.State != "running" && state.State != "starting" {
		return errInstanceRecovery
	} else if state.State == "running" && !e.SourceStopped {
		return nil
	}
	if err = verifyPluginRuntimeMode(ctx, e.Source, e.Version, "--switch"); err != nil {
		return err
	}
	e.SourceStopped = false
	return saveSwitchEvidence(*e)
}

// A stopped source can stay stopped during recovery only when the matrix
// still owns the fixture and its retirement. Receipt refresh and retirement
// still verify the committed config, storage, and any Restore replacement.
func (s *server) switchRecoveryLeavesFixtureStopped(ctx context.Context, e minecraftSwitchEvidence) (bool, error) {
	child, ok := ctx.Value(switchMatrixChildKey{}).(operationJournal)
	if !ok || !e.OriginallyRunning {
		return false, nil
	}
	if child.MatrixParent == "" || child.OperationID != e.OperationID || child.InstanceID != e.Source ||
		e.SourceRecord == nil || !e.SourceRecord.Runtime.MatrixFixture || s.operations == nil {
		return false, errInstanceRecovery
	}
	o := s.operations
	o.mu.Lock()
	parent, err := o.switchMatrixParent(child.MatrixParent, "switch", true)
	current := o.operations[child.OperationID]
	o.mu.Unlock()
	if err != nil || current.OperationType != operationTypeMinecraftSwitch || current.MatrixParent != child.MatrixParent ||
		current.InstanceID != e.Source || !slices.Contains(parent.MatrixFixtures, e.Source) {
		return false, errInstanceRecovery
	}
	f, err := readSwitchMatrixFixture(child.MatrixParent, e.Source)
	if err != nil || f.Stage != "owned" || f.checkRecord(*e.SourceRecord) != nil ||
		f.ConfigSHA256 != instanceHash(append(mustSwitchJSON(*e.SourceRecord), '\n')) {
		return false, errInstanceRecovery
	}
	state, err := instanceRuntimes.observe(ctx, e.Source)
	return err == nil && state.State == "stopped", err
}

func (s *server) recoverMinecraftSwitch(ctx context.Context, e *minecraftSwitchEvidence) error {
	lock, err := acquirePluginMaintenanceLock()
	if err != nil {
		return err
	}
	defer lock.Close()
	return s.recoverMinecraftSwitchLocked(ctx, e, lock)
}

func (s *server) recoverMinecraftSwitchLocked(ctx context.Context, e *minecraftSwitchEvidence, lock *pluginMaintenanceLock) error {
	b := instanceRuntimes
	var err error
	// This evidence was saved before any runtime changes were authorized.
	if e.Stage == "validating" {
		if e.CloneID != "" || e.SourceStopped || e.SourceContentChanged || e.WorldChanged {
			return errInstanceRecovery
		}
		return b.verifyCommittedSwitchState(ctx, e.Source, e.Version)
	}
	if e.Action == "convert" {
		if pending, pendingErr := b.pending(e.Source); pendingErr != nil {
			return pendingErr
		} else if pending {
			if err = b.recover(ctx, e.Source); err != nil {
				return err
			}
		}
		if e.SourceRecord == nil || e.SourceRecord.Runtime.ID != e.Source {
			return errInstanceRecovery
		}
		current, err := b.readRecord(e.Source)
		if err != nil {
			return err
		}
		state, observeErr := b.observeRaw(ctx, e.Source)
		if observeErr != nil {
			return observeErr
		}
		if e.WorldChanged || current.Runtime.Software != e.SourceRecord.Runtime.Software || current.Runtime.PreserveContent != e.SourceRecord.Runtime.PreserveContent || (state.State == "stopped" || state.State == "failed") && e.OriginallyRunning {
			if e.WorldChanged && current.Runtime.Software == e.Target && switchLeafStartupMigration(e.Target, e.Version) &&
				(state.State == "running" || state.State == "starting") {
				if err = s.stopFailedSwitchStartup(ctx, *e, current); err != nil {
					return err
				}
			} else if state.State == "running" || state.State == "failed" {
				if err = b.actionWithWarnings(ctx, e.Source, "stop", true); err != nil {
					return err
				}
			}
			previous := *e.SourceRecord
			previous.Runtime.Version = e.Version
			previous.Runtime.PreserveContent = true
			snapshot, _, _, err := b.snapshot(ctx)
			if err != nil {
				return err
			}
			records, err := b.records()
			if err != nil {
				return err
			}
			if e.WorldChanged && current.Runtime.Software != previous.Runtime.Software {
				err = b.applySwitchRuntime(ctx, current, instanceReplaceRecord(records, previous), snapshot, *e, nil)
			} else {
				err = b.applyVersion(ctx, current, instanceReplaceRecord(records, previous), snapshot, true)
			}
			if err != nil {
				return err
			}
		}
		if err = s.restoreSwitchWorld(ctx, e, lock); err != nil {
			return err
		}
		leaveFixtureStopped, err := s.switchRecoveryLeavesFixtureStopped(ctx, *e)
		if err != nil {
			return err
		}
		e.SourceStopped = e.OriginallyRunning && !leaveFixtureStopped
		if err = saveSwitchEvidence(*e); err != nil {
			return err
		}
		if !leaveFixtureStopped {
			if err = s.restoreSwitchSource(ctx, e, lock); err != nil {
				return err
			}
		}
		if !e.OriginallyRunning {
			state, err := b.observeRaw(ctx, e.Source)
			if err != nil {
				return err
			}
			if state.State == "running" || state.State == "failed" {
				if err = b.actionWithWarnings(ctx, e.Source, "stop", true); err != nil {
					return err
				}
			}
		}
		e.SourceContentChanged = true
		if err = saveSwitchEvidence(*e); err != nil {
			return err
		}
		if err = s.clearSwitchSourceContent(ctx, e); err != nil {
			return err
		}
		return b.verifyCommittedSwitchState(ctx, e.Source, e.Version)
	}
	if e.CloneID != "" && !validSwitchClonePlacement(*e) {
		return switchRecoveryRefusal("source and clone storage binding failed")
	}
	if pending, pendingErr := b.pending(e.Source); pendingErr != nil {
		return switchRecoveryRefusal("source verification failed")
	} else if pending {
		if err = b.recover(ctx, e.Source); err != nil {
			return switchRecoveryRefusal("source runtime recovery failed")
		}
	}
	leaveFixtureStopped, err := s.switchRecoveryLeavesFixtureStopped(ctx, *e)
	if err != nil {
		return switchRecoveryRefusal("source fixture ownership or state could not be verified")
	}
	if !leaveFixtureStopped {
		if err = s.restoreSwitchSource(ctx, e, lock); err != nil {
			return switchRecoveryRefusal("original source server state could not be restored")
		}
	}
	if err = s.clearSwitchSourceContent(ctx, e); err != nil {
		return switchRecoveryRefusal("original source configuration could not be restored")
	}
	if err := b.verifySwitchCloneSource(ctx, *e, leaveFixtureStopped); err != nil {
		return err
	}
	if leaveFixtureStopped {
		child := ctx.Value(switchMatrixChildKey{}).(operationJournal)
		if err := s.operations.checkSwitchMatrixFixture(ctx, child.MatrixParent, e.Source); err != nil {
			return switchRecoveryRefusal("source fixture configuration or storage could not be verified")
		}
	}
	if e.CloneID != "" {
		if err := s.retireFailedSwitchClone(ctx, *e, lock); err != nil {
			return err
		}
	}
	e.SourceStopped = false
	return b.finishSwitchCloneRecovery(ctx, e)
}

func switchRecoveryRefusal(reason string) error {
	return fmt.Errorf("%w: %s", errInstanceRecovery, reason)
}

func validSwitchClonePlacement(e minecraftSwitchEvidence) bool {
	if e.CloneID == "" || !switchStructuralPathFromEvidence(e) ||
		validateInstanceRecordPlacement(*e.SourceRecord) != nil || validateInstanceRecordPlacement(*e.CloneRecord) != nil {
		return false
	}
	clone, source := e.CloneRecord, e.SourceRecord
	storage := source.Storage
	storage.DataPath = clone.Storage.DataPath
	return clone.Storage == storage && clone.Runtime.Software == e.Target &&
		clone.Runtime.UID == source.Runtime.UID && clone.Runtime.GID == source.Runtime.GID &&
		!clone.Runtime.StartOnBoot && !clone.NetworkMember
}

func (b *instanceRuntimeBackend) verifySwitchCloneSource(ctx context.Context, e minecraftSwitchEvidence, leaveFixtureStopped bool) error {
	source, err := b.readRecord(e.Source)
	if err != nil || e.SourceRecord == nil {
		return switchRecoveryRefusal("source verification failed")
	}
	saved := e.SourceRecord.Runtime
	if source.Storage != e.SourceRecord.Storage ||
		source.Runtime.Software != saved.Software || source.Runtime.Version != saved.Version ||
		source.Runtime.VersionPolicy != saved.VersionPolicy || source.Runtime.PaperChannel != saved.PaperChannel ||
		source.Runtime.PreserveContent != saved.PreserveContent || source.Runtime.StartOnBoot != saved.StartOnBoot ||
		source.Runtime.LoaderVersion != saved.LoaderVersion || source.Runtime.PufferfishBuild != saved.PufferfishBuild ||
		source.Runtime.ImageTag != saved.ImageTag || source.Runtime.ImageDigest != saved.ImageDigest ||
		source.Runtime.MatrixFixture != saved.MatrixFixture ||
		source.Runtime.UID != saved.UID || source.Runtime.GID != saved.GID {
		return switchRecoveryRefusal("source verification failed")
	}
	state, err := b.observe(ctx, e.Source)
	if err != nil || state.State == "invalid_configuration" || state.State == "recovery_required" {
		return switchRecoveryRefusal("source verification failed")
	}
	required := "stopped"
	if e.OriginallyRunning && !leaveFixtureStopped {
		required = "running"
	}
	// A previously uninitialized source was never stopped or changed by the
	// worker. It can start independently on reboot under its existing policy.
	if state.State != required && !(e.SourceUninitialized && saved.StartOnBoot && state.State == "running") {
		return switchRecoveryRefusal("original source server state could not be verified")
	}
	if err := b.verifyCommittedSwitchState(ctx, e.Source, e.Version); err != nil {
		return switchRecoveryRefusal("source verification failed")
	}
	return nil
}

func (s *server) retireFailedSwitchClone(ctx context.Context, e minecraftSwitchEvidence, lock *pluginMaintenanceLock) error {
	b, id := instanceRuntimes, e.CloneID
	runtimeIncomplete := switchRecoveryRefusal("clone runtime retirement incomplete")
	// Older workers may have started ordinary deletion before retiring the
	// Restore directory. Use its journal while it still exists.
	if deletion, err := b.deleteJournal(id); err == nil {
		if deletion.ResetPolicy != "" || !switchCloneRecordMatches(deletion.Record, *e.CloneRecord) {
			return runtimeIncomplete
		}
		if err := b.deleteStopped(ctx, id, true); err != nil {
			return runtimeIncomplete
		}
		if err := b.retireSwitchCloneRestore(ctx, e, deletion.Record, deletion.Stage); err != nil {
			return err
		}
		if err := b.deleteInstance(ctx, id, deletion.Name); err != nil {
			return runtimeIncomplete
		}
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return runtimeIncomplete
	}

	// Finish interrupted creation/activation and Restore using their existing
	// paths before ordinary deletion. Already retired clones go to finalization.
	record, err := b.readRecord(id)
	if errors.Is(err, os.ErrNotExist) || errors.Is(err, errInstanceNotFound) {
		registered, regErr := b.identities().readRecord(id + ".json")
		if errors.Is(regErr, os.ErrNotExist) {
			return nil
		}
		// Only interrupted initial creation can have a valid reservation but
		// no runtime config. Never reconstruct a formerly committed clone.
		if regErr != nil || registered != id || !errors.Is(err, os.ErrNotExist) || e.Stage != "creating_clone" {
			return runtimeIncomplete
		}
		snapshot, uid, gid, snapshotErr := b.snapshot(ctx)
		if snapshotErr != nil {
			return runtimeIncomplete
		}
		snapshot.StorageRoot = e.CloneRecord.Storage.Root
		snapshot.MountPoint = e.CloneRecord.Storage.MountPoint
		snapshot.ExpectedUUID = e.CloneRecord.Storage.UUID
		snapshot.ExpectedSource = e.CloneRecord.Storage.Source
		if e.CloneRecord.Backups != nil {
			snapshot.BackupPlacement = e.CloneRecord.Backups
			snapshot.BackupPath = e.CloneRecord.Backups.Root
		}
		if e.CloneRecord.Runtime.MatrixFixture {
			child, ok := ctx.Value(switchMatrixChildKey{}).(operationJournal)
			if !ok || child.OperationID != e.OperationID {
				return runtimeIncomplete
			}
			receipt, receiptErr := readSwitchMatrixFixture(child.MatrixParent, id)
			if receiptErr != nil || receipt.Stage != "reserved" || receipt.Candidate == nil ||
				!reflect.DeepEqual(*receipt.Candidate, *e.CloneRecord) {
				return runtimeIncomplete
			}
			err = b.configureRegisteredMatrixFixture(ctx, receipt, snapshot)
		} else {
			err = b.configureRegistered(ctx, id, instancePublicSettings(e.CloneRecord.Runtime), snapshot, uid, gid)
		}
		if err != nil {
			return runtimeIncomplete
		}
		record, err = b.readRecord(id)
	}
	if err != nil || !switchCloneRecordMatches(record, *e.CloneRecord) {
		return runtimeIncomplete
	}
	state, err := b.observeShutdown(ctx, id)
	if err != nil {
		return runtimeIncomplete
	}
	if state.State == "running" || state.State == "starting" {
		if err := s.stopFailedSwitchStartup(ctx, e, record); err != nil {
			return runtimeIncomplete
		}
	}
	if state.State == "stopping" {
		if err := b.waitForGracefulShutdown(ctx, id); err != nil {
			return runtimeIncomplete
		}
	}
	if pending, err := b.pending(id); err != nil {
		return runtimeIncomplete
	} else if pending {
		if err := b.recover(ctx, id); err != nil {
			return runtimeIncomplete
		}
	}
	record, err = b.readRecord(id)
	if err != nil || !switchCloneRecordMatches(record, *e.CloneRecord) {
		return runtimeIncomplete
	}
	if record.EnvironmentSHA256 == "" {
		// Complete an interrupted initial activation without starting the server.
		snapshot, _, _, snapshotErr := b.snapshot(ctx)
		records, recordsErr := b.records()
		if snapshotErr != nil || recordsErr != nil || b.apply(ctx, record, records, snapshot) != nil {
			return runtimeIncomplete
		}
		record, err = b.readRecord(id)
		if err != nil || !switchCloneRecordMatches(record, *e.CloneRecord) {
			return runtimeIncomplete
		}
	}
	if j, err := b.restoreJournal(id); err == nil && !restoreTerminal(j.Stage) {
		if j.OriginallyRunning {
			return switchRecoveryRefusal("retained Restore transaction validation failed")
		}
		claim := &instanceRestoreClaim{s: s, maintenance: lock.file}
		restoreErr := claim.recover(ctx, id)
		lock.file = claim.maintenance
		if restoreErr != nil {
			return switchRecoveryRefusal("clone Restore recovery incomplete")
		}
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return switchRecoveryRefusal("retained Restore transaction validation failed")
	}
	state, err = b.observeShutdown(ctx, id)
	if err != nil || (state.State == "failed" && b.normalizeStoppedFailure(ctx, state) != nil) {
		return runtimeIncomplete
	}
	if b.verifyLive(record) != nil || b.deleteStopped(ctx, id, false) != nil {
		return runtimeIncomplete
	}
	names, err := b.identities().readNames()
	if err != nil || names[id].Software != e.CloneRecord.Runtime.Software {
		return switchRecoveryRefusal("clone name metadata validation failed")
	}
	if err := s.retireSwitchCloneCrossplay(ctx, e, record); err != nil {
		return err
	}
	if err := b.retireSwitchCloneRestore(ctx, e, record, ""); err != nil {
		return err
	}
	if child, ok := ctx.Value(switchMatrixChildKey{}).(operationJournal); ok && !e.CloneCrossplayInstall {
		if err := s.operations.retireFailedMatrixCloneMods(ctx, child, e, record); err != nil {
			return err
		}
	}
	if err := b.deleteInstance(ctx, id, names[id].DisplayName); err != nil {
		return runtimeIncomplete
	}
	return nil
}

func switchCloneRecordMatches(current, saved instanceRuntimeRecord) bool {
	return current.Runtime.ID == saved.Runtime.ID && current.Storage == saved.Storage &&
		current.Runtime.Software == saved.Runtime.Software && current.Runtime.UID == saved.Runtime.UID &&
		current.Runtime.GID == saved.Runtime.GID && !current.Runtime.StartOnBoot && !current.NetworkMember
}

func (b *instanceRuntimeBackend) retireSwitchCloneRestore(ctx context.Context, e minecraftSwitchEvidence, record instanceRuntimeRecord, deleteStage string) error {
	id := record.Runtime.ID
	journal, err := b.restoreJournal(id)
	if errors.Is(err, os.ErrNotExist) {
		// These older deletion stages may already have removed the journal.
		// Only the fully retired clone can use journal-less cleanup afterward.
		if deleteStage == "data_removed" || deleteStage == "identity_retired" {
			return nil
		}
		return b.switchCloneRetirementStorage(ctx, "unrestored", record, "-")
	}
	refusal := switchRecoveryRefusal("retained Restore transaction validation failed")
	if err != nil || !restoreTerminal(journal.Stage) || journal.OriginallyRunning ||
		!instanceSHA256Pattern.MatchString(journal.Fingerprint) ||
		(journal.Stage == "completed" && len(journal.Replacement) != 2) ||
		(len(journal.Replacement) != 0 && len(journal.Replacement) != 2) {
		return refusal
	}
	raw, err := b.readPrivate(filepath.Join(b.root, id, "config.json"))
	if err != nil {
		return refusal
	}
	if instanceHash(raw) != journal.ConfigSHA256 {
		// A Paper-family worker may have committed removal of its startup
		// guard just before rebinding the terminal journal. Accept only that
		// exact transition on the captured clone. The helper still checks the
		// journal's parent and world inodes before retiring anything.
		prior := record.Runtime
		prior.PreserveContent = true
		if (e.Stage != "verifying_clone" && e.Stage != "complete") || journal.Stage != "completed" ||
			!javaServerPaperFamily(e.SourceRecord.Runtime.Software) || !javaServerPaperFamily(e.Target) ||
			record.Runtime.PreserveContent || prior != e.CloneRecord.Runtime {
			return refusal
		}
	}
	proof, err := json.Marshal(struct {
		ID              string   `json:"id"`
		Stage           string   `json:"stage"`
		Transaction     string   `json:"transaction"`
		Parent          []uint64 `json:"parent"`
		Original        []uint64 `json:"original"`
		Replacement     []uint64 `json:"replacement,omitempty"`
		DataMayBeAbsent bool     `json:"data_may_be_absent"`
	}{
		ID: id, Stage: journal.Stage, Transaction: journal.Transaction,
		Parent: journal.Parent, Original: journal.Original, Replacement: journal.Replacement,
		DataMayBeAbsent: deleteStage != "" && deleteStage != "intent",
	})
	if err != nil {
		return refusal
	}
	return b.switchCloneRetirementStorage(ctx, "journal", record, string(proof))
}

const instanceSwitchRetireHelper = "/usr/libexec/justvoxel/mjust/instance-switch-retire"

func (b *instanceRuntimeBackend) switchCloneRetirementStorage(ctx context.Context, action string, record instanceRuntimeRecord, evidence string) error {
	c, st := record.Runtime, record.Storage
	out, err := b.command(ctx, "nsenter", "--mount=/proc/1/ns/mnt", "--", instanceSwitchRetireHelper, action, c.ID, st.Root, st.DataPath, st.MountPoint, st.UUID, st.Source, strconv.FormatUint(uint64(c.UID), 10), strconv.FormatUint(uint64(c.GID), 10), evidence)
	if err == nil {
		return nil
	}
	switch strings.TrimSpace(string(out)) {
	case "mount_identity":
		return switchRecoveryRefusal("clone storage mount identity mismatch")
	case "unexpected_entry":
		return switchRecoveryRefusal("unexpected clone storage entry")
	default:
		return switchRecoveryRefusal("retained Restore transaction validation or retirement failed")
	}
}

func (b *instanceRuntimeBackend) finishSwitchCloneRecovery(ctx context.Context, e *minecraftSwitchEvidence) error {
	if id := e.CloneID; id != "" {
		runtimeIncomplete := switchRecoveryRefusal("clone runtime retirement incomplete")
		if _, err := b.identities().readRecord(id + ".json"); !errors.Is(err, os.ErrNotExist) {
			return runtimeIncomplete
		}
		dir, env, quadlet := b.paths(id)
		for _, path := range []string{dir, env, quadlet, b.deletePath(id)} {
			if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
				return runtimeIncomplete
			}
		}
		// Configured from observeRaw means systemd did not find a loaded
		// unit and Podman did not find a container, even a stopped one.
		state, err := b.observeRaw(ctx, id)
		if err != nil || state.State != "configured" || b.deleteStopped(ctx, id, true) != nil {
			return runtimeIncomplete
		}
		action := "empty"
		switch e.Stage {
		case "copying_clone", "verifying_clone", "complete", "recovered":
			action = "retired"
		}
		if err := b.switchCloneRetirementStorage(ctx, action, *e.CloneRecord, "-"); err != nil {
			return err
		}
		namesPath := filepath.Join(b.identities().directory, "names")
		name := filepath.Join(namesPath, id+".json")
		nameInvalid := switchRecoveryRefusal("clone name metadata validation failed")
		names, err := b.privateDirectory(namesPath, false)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nameInvalid
		}
		if names != nil {
			defer names.Close()
		}
		raw, err := b.readPrivate(name)
		if err == nil {
			var record minecraftInstanceNameRecord
			if len(raw) > 4096 || instanceDecode(raw, &record) != nil || record.ID != id || record.Software != e.CloneRecord.Runtime.Software {
				return nameInvalid
			}
			displayName, err := validateInstanceName(record.DisplayName)
			if err != nil || displayName == "" || displayName != record.DisplayName {
				return nameInvalid
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return nameInvalid
		}
		if err := b.removePrivate(name); err != nil {
			return nameInvalid
		}
		if _, err := os.Lstat(name); !errors.Is(err, os.ErrNotExist) {
			return nameInvalid
		}
		// Also sync on retry after an interrupted unlink.
		if names != nil && names.Sync() != nil {
			return nameInvalid
		}
	}
	e.Stage = "recovered"
	if err := saveSwitchEvidence(*e); err != nil {
		return switchRecoveryRefusal("recovered Switch evidence could not be persisted")
	}
	return nil
}

func (b *instanceRuntimeBackend) verifyCommittedSwitchState(ctx context.Context, id string, exact ...string) error {
	record, err := b.readRecord(id)
	if err != nil {
		return err
	}
	if err = b.verifyLive(record); err != nil {
		return err
	}
	state, err := b.observe(ctx, id)
	if err != nil {
		return err
	}
	if state.State == "running" {
		verify, cancel := context.WithTimeout(ctx, 15*time.Minute)
		defer cancel()
		version := record.Runtime.Version
		if len(exact) == 1 {
			version = exact[0]
		}
		return verifyPluginRuntimeMode(verify, id, version, "--switch")
	}
	if state.State != "stopped" {
		return errInstanceRecovery
	}
	return b.storage(ctx, "validate", record)
}

func (s *server) clearSwitchSourceContent(ctx context.Context, e *minecraftSwitchEvidence) error {
	if !e.SourceContentChanged {
		return nil
	}
	if e.SourceRecord == nil || e.SourceRecord.Runtime.ID != e.Source {
		return errInstanceRecovery
	}
	b := instanceRuntimes
	current, err := b.readRecord(e.Source)
	if err != nil {
		return err
	}
	next := current
	next.Runtime.PreserveContent = e.SourceRecord.Runtime.PreserveContent
	next.Runtime.Version = e.SourceRecord.Runtime.Version
	next.Runtime.VersionPolicy = e.SourceRecord.Runtime.VersionPolicy
	next.Runtime.PaperChannel = e.SourceRecord.Runtime.PaperChannel
	snapshot, _, _, err := b.snapshot(ctx)
	if err != nil {
		return err
	}
	records, err := b.records()
	if err != nil {
		return err
	}
	if err = b.applyVersion(context.WithValue(ctx, switchDefinitionVerificationKey{}, true), current, instanceReplaceRecord(records, next), snapshot, true); err != nil {
		return err
	}
	if err = rebindSwitchSourceRestore(ctx, e); err != nil {
		return err
	}
	e.SourceContentChanged = false
	return saveSwitchEvidence(*e)
}
func (b *instanceRuntimeBackend) prepareSwitchSourceCopy(ctx context.Context, id, version string) error {
	current, err := b.readRecord(id)
	if err != nil {
		return err
	}
	next := current
	next.Runtime.PreserveContent = true
	next.Runtime.Version = version
	// The operation is fixed to the installed game version even if the source
	// normally follows releases. Normal policy is restored after the copy.
	next.Runtime.VersionPolicy = "pinned"
	channel, err := b.command(ctx, instanceRuntimeHelper, "channel", next.Runtime.Software, version, "pinned")
	if err != nil {
		return err
	}
	next.Runtime.PaperChannel = strings.TrimSpace(string(channel))
	snapshot, _, _, err := b.snapshot(ctx)
	if err != nil {
		return err
	}
	records, err := b.records()
	if err != nil {
		return err
	}
	return b.applyVersion(ctx, current, instanceReplaceRecord(records, next), snapshot, true)
}
