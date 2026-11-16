package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
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
	if instanceDecode(out, &inspection) != nil || inspection.Metadata.ID != e.Source || inspection.Metadata.Archive != archive.Archive || inspection.Integrity != "verified" || !instanceSHA256Pattern.MatchString(inspection.Fingerprint) {
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
	e.Stage = "verifying_clone"
	if err = saveSwitchEvidence(*e); err != nil {
		return err
	}
	if err = lock.start(ctx, e.CloneID); err != nil {
		return err
	}
	if err = verifyPluginRuntime(ctx, e.CloneID, p.Version); err != nil {
		return err
	}
	record, err := b.readRecord(e.CloneID)
	if err != nil || record.Runtime.Software != p.Target || record.Runtime.Version != p.Version || record.Runtime.ID == e.Source {
		return errInstanceRecovery
	}
	expected, imageErr := b.command(ctx, "podman", "image", "inspect", instanceRuntimeImage(record.Runtime), "--format", "{{.Id}}")
	actual, runningErr := b.command(ctx, "podman", "inspect", "minecraft-"+e.CloneID, "--format", "{{.Image}}")
	if imageErr != nil || runningErr != nil || strings.TrimSpace(string(expected)) == "" || strings.TrimSpace(string(expected)) != strings.TrimSpace(string(actual)) {
		return errInstanceRecovery
	}

	// A new clone is left stopped after meaningful runtime verification, avoiding
	// accidental simultaneous public servers or an unreviewed boot preference.
	if err = b.actionWithWarnings(ctx, e.CloneID, "stop", true); err != nil {
		return err
	}
	if err = b.finishSwitchClone(ctx, e.CloneID); err != nil {
		return err
	}
	if err = s.clearSwitchSourceContent(ctx, e); err != nil {
		return err
	}
	e.Stage = "complete"
	return saveSwitchEvidence(*e)
}

func (b *instanceRuntimeBackend) switchCloneRestoreStorage(ctx context.Context, action, id, source, archive, fingerprint string) ([]byte, error) {
	if !instanceIDPattern.MatchString(id) || !instanceIDPattern.MatchString(source) || id == source || (action != "plan" && action != "prepare") {
		return nil, errInstanceID
	}
	out, err := instanceBackupStorageCommand(ctx, "nsenter", "--mount=/proc/1/ns/mnt", "--", instanceRestoreHelper, action, id, archive, fingerprint, "false", source)
	if err != nil {
		return nil, instanceRuntimeFailure("The consistent source archive could not be safely staged into the new clone. The original server is preserved.")
	}
	return out, nil
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
	b := instanceRuntimes
	if e.Action == "convert" {
		lock, err := acquirePluginMaintenanceLock()
		if err != nil {
			return err
		}
		defer lock.Close()
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
		if current.Runtime.Software != e.SourceRecord.Runtime.Software || current.Runtime.PreserveContent || state.State == "stopped" && e.OriginallyRunning {
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
			if err = b.applyVersion(ctx, current, instanceReplaceRecord(records, previous), snapshot, true); err != nil {
				return err
			}
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
	lock, err := acquirePluginMaintenanceLock()
	if err != nil {
		return err
	}
	defer lock.Close()
	if pending, pendingErr := b.pending(e.Source); pendingErr != nil {
		return pendingErr
	} else if pending {
		if err = b.recover(ctx, e.Source); err != nil {
			return err
		}
	}
	if err = s.restoreSwitchSource(ctx, e, lock); err != nil {
		return err
	}
	if err = s.clearSwitchSourceContent(ctx, e); err != nil {
		return err
	}
	if e.CloneID != "" {
		// Creation/activation and restore recovery retain their own evidence. Finish
		// rollback first, then use normal instance deletion for this failed clone.
		record, readErr := b.readRecord(e.CloneID)
		if errors.Is(readErr, os.ErrNotExist) || errors.Is(readErr, errInstanceNotFound) {
			registered, regErr := b.identities().readRecord(e.CloneID + ".json")
			if errors.Is(regErr, os.ErrNotExist) {
				e.Stage = "recovered"
				return saveSwitchEvidence(*e)
			}
			if regErr != nil || registered != e.CloneID || e.CloneRecord == nil {
				return errInstanceRecovery
			}
			snapshot, uid, gid, snapshotErr := b.snapshot(ctx)
			if snapshotErr != nil {
				return snapshotErr
			}
			snapshot.StorageRoot = e.CloneRecord.Storage.Root
			snapshot.MountPoint = e.CloneRecord.Storage.MountPoint
			snapshot.ExpectedUUID = e.CloneRecord.Storage.UUID
			snapshot.ExpectedSource = e.CloneRecord.Storage.Source
			if e.CloneRecord.Backups != nil {
				snapshot.BackupPlacement = e.CloneRecord.Backups
				snapshot.BackupPath = e.CloneRecord.Backups.Root
			}
			if err = b.configureRegistered(ctx, e.CloneID, instancePublicSettings(e.CloneRecord.Runtime), snapshot, uid, gid); err != nil {
				return err
			}
			record, readErr = b.readRecord(e.CloneID)
		}
		if readErr != nil || record.Runtime.ID == e.Source {
			return errInstanceRecovery
		}
		state, stateErr := b.observeRaw(ctx, e.CloneID)
		if stateErr != nil {
			return stateErr
		}
		if state.State == "running" {
			if err = b.actionWithWarnings(ctx, e.CloneID, "stop", true); err != nil {
				return err
			}
		}
		if pending, pendingErr := b.pending(e.CloneID); pendingErr != nil {
			return pendingErr
		} else if pending {
			if err = b.recover(ctx, e.CloneID); err != nil {
				return err
			}
		}
		recoveredRecord, recordErr := b.readRecord(e.CloneID)
		if recordErr != nil {
			return recordErr
		}
		if recoveredRecord.EnvironmentSHA256 == "" {
			// An interrupted initial activation is a configured reservation. Complete
			// its existing stopped-only activation path so normal deletion can verify
			// ownership; never start it or retry conversion.
			snapshot, _, _, snapshotErr := b.snapshot(ctx)
			if snapshotErr != nil {
				return snapshotErr
			}
			records, recordsErr := b.records()
			if recordsErr != nil {
				return recordsErr
			}
			if err = b.apply(ctx, recoveredRecord, records, snapshot); err != nil {
				return err
			}
		}
		if j, journalErr := b.restoreJournal(e.CloneID); journalErr == nil && !restoreTerminal(j.Stage) {
			claim := &instanceRestoreClaim{s: s, maintenance: lock.file}
			err = claim.recover(ctx, e.CloneID)
			lock.file = claim.maintenance
			if err != nil {
				return err
			}
		} else if journalErr != nil && !errors.Is(journalErr, os.ErrNotExist) {
			return journalErr
		}
		// Normal deletion is journaled and safe to retry; no source paths are used.
		names, nameErr := b.identities().readNames()
		if nameErr != nil {
			return nameErr
		}
		if err = b.deleteInstance(ctx, e.CloneID, names[e.CloneID].DisplayName); err != nil {
			return err
		}
	}
	if err = b.verifyCommittedSwitchState(ctx, e.Source); err != nil {
		return err
	}
	e.Stage = "recovered"
	return saveSwitchEvidence(*e)
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
