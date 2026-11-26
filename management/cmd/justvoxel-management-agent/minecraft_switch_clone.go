package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
	if _, err = b.backupStorage(ctx, "preflight", e.Source); err != nil {
		return err
	}
	state, err := b.observe(ctx, e.Source)
	if err != nil || (state.State != "running" && state.State != "stopped") {
		return errInstanceBusy
	}
	e.OriginallyRunning = state.State == "running"
	e.Stage = "source_copy"
	// Intent is durable BEFORE shutdown, including power loss during countdown.
	e.SourceStopped = e.OriginallyRunning
	e.SourceContentChanged = true
	if err = saveSwitchEvidence(*e); err != nil {
		return err
	}
	if err = b.prepareSwitchSourceCopy(ctx, e.Source, p.Version); err != nil {
		return err
	}
	if e.OriginallyRunning {
		if err = b.actionWithWarnings(ctx, e.Source, "stop", true); err != nil {
			return err
		}
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
			if switchNeedsWorldMove(p.Current, p.Target) {
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
	snapshot, _, _, err := b.snapshot(ctx)
	if err != nil {
		return err
	}
	snapshot.StorageRoot, snapshot.MountPoint, snapshot.ExpectedUUID, snapshot.ExpectedSource = source.Storage.Root, source.Storage.MountPoint, source.Storage.UUID, source.Storage.Source
	if source.Backups != nil {
		snapshot.BackupPlacement = source.Backups
		snapshot.BackupPath = source.Backups.Root
	}
	records, err := b.records()
	if err != nil {
		return err
	}
	c := source.Runtime
	c.ID = ""
	c.PreserveContent = true
	c.Software = p.Target
	c.Version = p.Version
	c.PufferfishBuild = p.Runtime.PufferfishBuild
	c.LoaderVersion = p.Runtime.LoaderVersion
	if !(javaServerPaperFamily(p.Current) && javaServerPaperFamily(p.Target)) {
		c.BedrockEnabled = p.Runtime.CrossplayEnabled
	}
	c.VersionPolicy = "pinned"
	c.StartOnBoot = false
	channel, channelErr := b.command(ctx, instanceRuntimeHelper, "channel", p.Target, p.Version, "pinned")
	if channelErr != nil {
		return channelErr
	}
	c.PaperChannel = strings.TrimSpace(string(channel))
	c.ImageTag = p.Runtime.ImageTag
	c.ImageDigest = p.RuntimeDigest
	if err = switchClonePorts(records, &c); err != nil {
		return err
	}
	var secret [24]byte
	if _, err = rand.Read(secret[:]); err != nil {
		return err
	}
	c.RCONPassword = hex.EncodeToString(secret[:])
	names, err := b.identities().listInstances("")
	if err != nil {
		return err
	}
	name := switchCloneName(names, e.Source, p.Target)
	_, err = b.createTracked(ctx, c, name, snapshot, func(record instanceRuntimeRecord) error {
		if record.Runtime.ID == e.Source {
			return errInstanceID
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
	out, err := b.switchCloneRestoreStorage(ctx, "plan", e.CloneID, e.Source, archive.Archive, "")
	if err != nil {
		return err
	}
	var inspection instanceRestoreInspection
	if instanceDecode(out, &inspection) != nil || inspection.Metadata != archive || inspection.CurrentSoftware != p.Target || inspection.CurrentVersion != p.Version ||
		inspection.Integrity != "verified" || !instanceSHA256Pattern.MatchString(inspection.Fingerprint) {
		return errInstanceRecovery
	}
	plan := instanceRestorePlan{instanceRestoreInspection: inspection, RuntimeState: "stopped", CloneSourceID: e.Source}
	claim := &instanceRestoreClaim{s: s, maintenance: lock.file}
	err = s.runInstanceRestore(ctx, e.CloneID, plan, claim)
	// runInstanceRestore owns this descriptor during rollback/start handoffs.
	lock.file = claim.maintenance
	if err != nil {
		return err
	}
	if switchNeedsWorldMove(p.Current, p.Target) {
		clone, readErr := b.readRecord(e.CloneID)
		if readErr != nil {
			return readErr
		}
		if err = transformSwitchWorld(clone.Storage.DataPath, p.Version); err != nil {
			return err
		}
	}
	e.Stage = "verifying_clone"
	if err = saveSwitchEvidence(*e); err != nil {
		return err
	}
	if err = lock.start(ctx, e.CloneID); err != nil {
		return err
	}
	if err = b.verifySwitchCloneRuntime(ctx, *e, p); err != nil {
		return err
	}

	// A new clone is left stopped after meaningful runtime verification, avoiding
	// accidental simultaneous public servers or an unreviewed boot preference.
	if err = b.actionWithWarnings(ctx, e.CloneID, "stop", true); err != nil {
		return err
	}
	if javaServerPaperFamily(p.Current) && javaServerPaperFamily(p.Target) {
		if err = b.finishSwitchClone(ctx, e.CloneID); err != nil {
			return err
		}
	}
	if err = s.clearSwitchSourceContent(ctx, e); err != nil {
		return err
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
		return errInstanceRecovery
	}
	if err := b.verifyLive(record); err != nil {
		return err
	}
	if err := verifyPluginRuntimeMode(ctx, e.CloneID, p.Version, "--switch-clone"); err != nil {
		return err
	}
	state, err := b.observeRaw(ctx, e.CloneID)
	if err != nil || state.State != "running" {
		return errInstanceRecovery
	}
	if _, err := b.players(ctx, e.CloneID); err != nil {
		return err
	}
	expected, imageErr := b.command(ctx, "podman", "image", "inspect", instanceRuntimeImage(record.Runtime), "--format", "{{.Id}}")
	actual, runningErr := b.command(ctx, "podman", "inspect", "minecraft-"+e.CloneID, "--format", "{{.Image}}")
	if imageErr != nil || runningErr != nil || strings.TrimSpace(string(expected)) == "" || strings.TrimSpace(string(expected)) != strings.TrimSpace(string(actual)) {
		return errInstanceRecovery
	}
	return nil
}

func (b *instanceRuntimeBackend) switchCloneRestoreStorage(ctx context.Context, action, id, source, archive, fingerprint string) ([]byte, error) {
	if !instanceIDPattern.MatchString(id) || !instanceIDPattern.MatchString(source) || id == source || (action != "plan" && action != "prepare") {
		return nil, errInstanceID
	}
	out, err := instanceBackupStorageCommand(ctx, "nsenter", "--mount=/proc/1/ns/mnt", "--", instanceRestoreHelper, action, id, archive, fingerprint, "false", source)
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
	if !e.SourceStopped {
		return nil
	}
	b := instanceRuntimes
	state, err := b.observe(ctx, e.Source)
	if err != nil {
		return err
	}
	if state.State == "stopped" {
		if err = lock.start(ctx, e.Source); err != nil {
			return err
		}
	} else if state.State != "running" {
		return errInstanceRecovery
	}
	if err = verifyPluginRuntime(ctx, e.Source, e.Version); err != nil {
		return err
	}
	e.SourceStopped = false
	return saveSwitchEvidence(*e)
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
		return b.verifyCommittedSwitchState(ctx, e.Source)
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
		if e.WorldChanged || current.Runtime.Software != e.SourceRecord.Runtime.Software || current.Runtime.PreserveContent || state.State == "stopped" && e.OriginallyRunning {
			state, err := b.observeRaw(ctx, e.Source)
			if err != nil {
				return err
			}
			if state.State == "running" {
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
		e.SourceStopped = e.OriginallyRunning
		if err = saveSwitchEvidence(*e); err != nil {
			return err
		}
		if err = s.restoreSwitchSource(ctx, e, lock); err != nil {
			return err
		}
		if !e.OriginallyRunning {
			state, err := b.observeRaw(ctx, e.Source)
			if err != nil {
				return err
			}
			if state.State == "running" {
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
		return b.verifyCommittedSwitchState(ctx, e.Source)
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
	if err = s.restoreSwitchSource(ctx, e, lock); err != nil {
		return switchRecoveryRefusal("original source server state could not be restored")
	}
	if err = s.clearSwitchSourceContent(ctx, e); err != nil {
		return switchRecoveryRefusal("original source configuration could not be restored")
	}
	if e.CloneID != "" {
		if err := b.verifySwitchCloneSource(ctx, *e); err != nil {
			return err
		}
		if err := s.retireFailedSwitchClone(ctx, *e, lock); err != nil {
			return err
		}
	} else if err := b.verifyCommittedSwitchState(ctx, e.Source); err != nil {
		return switchRecoveryRefusal("source verification failed")
	}
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

func (b *instanceRuntimeBackend) verifySwitchCloneSource(ctx context.Context, e minecraftSwitchEvidence) error {
	source, err := b.readRecord(e.Source)
	saved := e.SourceRecord.Runtime
	if err != nil || source.Storage != e.SourceRecord.Storage ||
		source.Runtime.Software != saved.Software || source.Runtime.Version != saved.Version ||
		source.Runtime.VersionPolicy != saved.VersionPolicy || source.Runtime.PaperChannel != saved.PaperChannel ||
		source.Runtime.PreserveContent != saved.PreserveContent || source.Runtime.StartOnBoot != saved.StartOnBoot ||
		source.Runtime.UID != saved.UID || source.Runtime.GID != saved.GID {
		return switchRecoveryRefusal("source verification failed")
	}
	state, err := b.observeRaw(ctx, e.Source)
	if err != nil || (e.OriginallyRunning && state.State != "running") || (!e.OriginallyRunning && state.State != "stopped") {
		return switchRecoveryRefusal("original source server state could not be verified")
	}
	if err := b.verifyCommittedSwitchState(ctx, e.Source); err != nil {
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
		if err = b.configureRegistered(ctx, id, instancePublicSettings(e.CloneRecord.Runtime), snapshot, uid, gid); err != nil {
			return runtimeIncomplete
		}
		record, err = b.readRecord(id)
	}
	if err != nil || !switchCloneRecordMatches(record, *e.CloneRecord) {
		return runtimeIncomplete
	}
	state, err := b.observeRaw(ctx, id)
	if err != nil {
		return runtimeIncomplete
	}
	if state.State == "running" {
		if err := b.actionWithWarnings(ctx, id, "stop", true); err != nil {
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
	state, err = b.observeRaw(ctx, id)
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
	if err := b.retireSwitchCloneRestore(ctx, e, record, ""); err != nil {
		return err
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

func (b *instanceRuntimeBackend) verifyCommittedSwitchState(ctx context.Context, id string) error {
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
		return verifyPluginRuntime(verify, id, record.Runtime.Version)
	}
	if state.State != "stopped" {
		return errInstanceRecovery
	}
	return b.storage(ctx, "validate", record)
}

func (b *instanceRuntimeBackend) setSwitchPreserveContent(ctx context.Context, id string, preserve bool) error {
	previous, err := b.readRecord(id)
	if err != nil {
		return err
	}
	if previous.Runtime.PreserveContent == preserve {
		return nil
	}
	next := previous
	next.Runtime.PreserveContent = preserve
	snapshot, _, _, err := b.snapshot(ctx)
	if err != nil {
		return err
	}
	records, err := b.records()
	if err != nil {
		return err
	}
	return b.applyVersion(ctx, previous, instanceReplaceRecord(records, next), snapshot, true)
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
	if err = b.applyVersion(ctx, current, instanceReplaceRecord(records, next), snapshot, true); err != nil {
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

// Only a verified terminal clone restore may have its configuration binding
// advanced past the temporary startup guard. Active/deployed restore evidence
// keeps its original binding and recovery semantics.
func (b *instanceRuntimeBackend) finishSwitchClone(ctx context.Context, id string) error {
	journal, err := b.restoreJournal(id)
	if err != nil || journal.Stage != "completed" {
		return errInstanceRecovery
	}
	raw, err := b.readPrivate(filepath.Join(b.root, id, "config.json"))
	if err != nil || instanceHash(raw) != journal.ConfigSHA256 {
		return errInstanceRecovery
	}
	if err = b.setSwitchPreserveContent(ctx, id, false); err != nil {
		return err
	}
	raw, err = b.readPrivate(filepath.Join(b.root, id, "config.json"))
	if err != nil {
		return err
	}
	journal.ConfigSHA256 = instanceHash(raw)
	return b.writeJSON(filepath.Join(b.root, id, "restore-journal.json"), journal, true)
}
