package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

// This capability is bound to one exact publication, never to a Version caller.
type switchRuntimeApplyKey struct{}
type switchRuntimeApplyAuthorization struct {
	previous, next instanceRuntimeRecord
}

func switchRuntimeApplyAuthorized(ctx context.Context, previous, next instanceRuntimeRecord) bool {
	a, ok := ctx.Value(switchRuntimeApplyKey{}).(switchRuntimeApplyAuthorization)
	return ok && reflect.DeepEqual(a.previous, previous) && reflect.DeepEqual(a.next, next) &&
		previous.Runtime.Version == next.Runtime.Version &&
		previous.Runtime.Software != "bedrock" && next.Runtime.Software != "bedrock" &&
		switchStructuralPath(previous.Runtime.Software, next.Runtime.Software)
}

// A nil plan authorizes restoration only to the original source runtime. Both
// directions require the cold rollback authority already persisted by Convert.
func (b *instanceRuntimeBackend) applySwitchRuntime(ctx context.Context, previous instanceRuntimeRecord, records []instanceRuntimeRecord, snapshot minecraftInstanceRuntimeContext, e minecraftSwitchEvidence, plan *minecraftSwitchPlan) error {
	if b != instanceRuntimes || e.Schema != 1 || (e.OperationID == "" || filepath.Base(e.OperationID) != e.OperationID) ||
		!switchStructuralPathFromEvidence(e) || e.Action != "convert" || !e.WorldChanged || !validInstanceRestoreArchive(e.ColdArchive) {
		return errInstanceRecovery
	}
	var durable minecraftSwitchEvidence
	raw, err := b.readPrivate(switchEvidencePath(e.OperationID))
	if err != nil || instanceDecode(raw, &durable) != nil || !reflect.DeepEqual(durable, e) {
		return errInstanceRecovery
	}
	var next instanceRuntimeRecord
	for _, record := range records {
		if record.Runtime.ID == e.Source {
			next = record
		}
	}
	if previous.Runtime.ID != e.Source || next.Runtime.ID != e.Source || previous.Runtime.Version != e.Version || next.Runtime.Version != e.Version ||
		previous.Storage != e.SourceRecord.Storage || next.Storage != e.SourceRecord.Storage {
		return errInstanceRecovery
	}
	if plan != nil {
		p := *plan
		if !p.Convert || p.InstanceID != e.Source || p.Current != e.SourceRecord.Runtime.Software || previous.Runtime.Software != p.Current ||
			p.Target != e.Target || next.Runtime.Software != p.Target || p.Version != e.Version || switchWorldRisk(p.Current, p.Target) ||
			previous.Runtime.PufferfishBuild != e.SourceRecord.Runtime.PufferfishBuild || next.Runtime.PufferfishBuild != p.Runtime.PufferfishBuild ||
			previous.Runtime.LoaderVersion != e.SourceRecord.Runtime.LoaderVersion ||
			!operationFingerprintPattern.MatchString(p.Fingerprint) || !instanceSHA256Pattern.MatchString(p.ContentFingerprint) ||
			p.Runtime.ServerType != p.Target || p.Runtime.Installed != p.Version || p.Runtime.SelectedCandidate != p.Version || !p.Runtime.ServerSupported ||
			!operationFingerprintPattern.MatchString(p.RuntimeDigest) || next.Runtime.ImageDigest != p.RuntimeDigest ||
			next.Runtime.ImageTag != p.Runtime.ImageTag || next.Runtime.LoaderVersion != p.Runtime.LoaderVersion || next.Runtime.VersionPolicy != "pinned" ||
			next.Runtime.BedrockEnabled != p.Runtime.CrossplayEnabled {
			return errInstanceRecovery
		}
		for _, item := range p.Content {
			if item.Compatibility != "verified" {
				return errInstanceRecovery
			}
		}
	} else {
		original := e.SourceRecord.Runtime
		original.Version, original.PreserveContent = e.Version, true
		if previous.Runtime.Software != e.Target || !reflect.DeepEqual(next.Runtime, original) {
			return errInstanceRecovery
		}
	}
	if !switchStructuralPath(previous.Runtime.Software, next.Runtime.Software) {
		return errInstanceRecovery
	}
	ctx = context.WithValue(ctx, switchRuntimeApplyKey{}, switchRuntimeApplyAuthorization{previous: previous, next: next})
	return b.applyVersion(ctx, previous, records, snapshot, true)
}

// New Switch directions compose the same cold backup, runtime publication and
// archive restore operations. Ordinary Version maintenance is unchanged.
func (s *server) convertMinecraftSwitch(ctx context.Context, e *minecraftSwitchEvidence, p minecraftSwitchPlan, lock *pluginMaintenanceLock) error {
	b := instanceRuntimes
	if !p.Convert || switchWorldRisk(p.Current, p.Target) {
		return errInstanceID
	}
	if _, err := b.backupStorage(ctx, "preflight", e.Source); err != nil {
		return err
	}
	e.SourceStopped, e.SourceContentChanged = e.OriginallyRunning, true
	if err := saveSwitchEvidence(*e); err != nil {
		return err
	}
	if e.OriginallyRunning {
		if err := b.actionWithWarnings(ctx, e.Source, "stop", true); err != nil {
			return err
		}
	}
	if err := b.prepareSwitchSourceCopy(ctx, e.Source, p.Version); err != nil {
		return err
	}
	before, err := b.backupStorage(ctx, "inventory", e.Source)
	var old []instanceBackupMetadata
	if err != nil || json.Unmarshal(before, &old) != nil {
		return errInstanceRecovery
	}
	if _, err = b.backupStorage(ctx, "create", e.Source); err != nil {
		return err
	}
	after, err := b.backupStorage(ctx, "inventory", e.Source)
	var inventory []instanceBackupMetadata
	if err != nil || json.Unmarshal(after, &inventory) != nil {
		return errInstanceRecovery
	}
	added := 0
	for _, candidate := range inventory {
		known := false
		for _, previous := range old {
			if candidate.Archive == previous.Archive {
				known = true
				break
			}
		}
		if !known {
			if candidate.ID != e.Source || candidate.Software != p.Current || candidate.Version != p.Version || !validInstanceRestoreArchive(candidate.Archive) {
				return errInstanceRecovery
			}
			e.ColdArchive = candidate.Archive
			added++
		}
	}
	if added != 1 {
		return errInstanceRecovery
	}
	record, err := b.readRecord(e.Source)
	if err != nil {
		return err
	}
	content, hash, err := switchContentEvidence(ctx, record, p.Target, p.Version)
	if err != nil || hash != p.ContentFingerprint {
		return instanceRuntimeFailure("Plugins & Mods changed; review Switch again.")
	}
	for _, item := range content {
		if item.Compatibility != "verified" {
			return errInstanceRecovery
		}
	}
	if switchNeedsWorldMove(p.Current, p.Target) {
		moves, layoutErr := inspectSwitchWorld(record.Storage.DataPath, p.Version)
		if layoutErr != nil || instanceHash(mustSwitchJSON(moves)) != p.WorldFingerprint {
			return errInstanceRecovery
		}
	}
	// Persist rollback authority BEFORE any target can modify world data.
	e.WorldChanged, e.Stage = true, "converting_world"
	if err = saveSwitchEvidence(*e); err != nil {
		return err
	}
	if switchNeedsWorldMove(p.Current, p.Target) {
		if err = transformSwitchWorld(record.Storage.DataPath, p.Version); err != nil {
			return err
		}
	}
	next := record
	next.Runtime.Software, next.Runtime.VersionPolicy, next.Runtime.Version = p.Target, "pinned", p.Version
	next.Runtime.PufferfishBuild = p.Runtime.PufferfishBuild
	next.Runtime.LoaderVersion, next.Runtime.ImageTag, next.Runtime.ImageDigest = p.Runtime.LoaderVersion, p.Runtime.ImageTag, p.RuntimeDigest
	next.Runtime.BedrockEnabled = p.Runtime.CrossplayEnabled
	channel, err := b.command(ctx, instanceRuntimeHelper, "channel", p.Target, p.Version, "pinned")
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
	if err = b.applySwitchRuntime(ctx, record, instanceReplaceRecord(records, next), snapshot, *e, &p); err != nil {
		return err
	}
	if err = lock.start(ctx, e.Source); err != nil {
		return err
	}
	if err = verifyPluginRuntime(ctx, e.Source, p.Version); err != nil {
		return err
	}
	if err = b.verifySwitchTargetImage(ctx, e.Source, p); err != nil {
		return err
	}
	if !e.OriginallyRunning {
		if err = b.actionWithWarnings(ctx, e.Source, "stop", true); err != nil {
			return err
		}
	}
	if err = rebindSwitchSourceRestore(ctx, e); err != nil {
		return err
	}
	// Keep the no-download guard: clearing it could install target plugins on
	// the next start without the user's consent. Content stays byte-for-byte.
	completed := *e
	completed.WorldChanged, completed.SourceStopped, completed.Stage = false, false, "complete"
	if err = saveSwitchEvidence(completed); err != nil {
		return err
	}
	*e = completed
	return nil
}

func (b *instanceRuntimeBackend) verifySwitchTargetImage(ctx context.Context, id string, p minecraftSwitchPlan) error {
	record, err := b.readRecord(id)
	if err != nil || record.Runtime.Software != p.Target || record.Runtime.Version != p.Version || record.Runtime.ImageDigest != p.RuntimeDigest {
		return errInstanceRecovery
	}
	expected, e1 := b.command(ctx, "podman", "image", "inspect", instanceRuntimeImage(record.Runtime), "--format", "{{.Id}}")
	actual, e2 := b.command(ctx, "podman", "inspect", "minecraft-"+id, "--format", "{{.Image}}")
	if e1 != nil || e2 != nil || strings.TrimSpace(string(expected)) == "" || strings.TrimSpace(string(expected)) != strings.TrimSpace(string(actual)) {
		return errInstanceRecovery
	}
	return nil
}

// Retry recovery restores the durable cold archive, never retries conversion.
// The existing restore engine validates archive ownership and keeps its own
// inode-bound activation/rollback journal across interruptions.
func (s *server) restoreSwitchWorld(ctx context.Context, e *minecraftSwitchEvidence, lock *pluginMaintenanceLock) error {
	if !e.WorldChanged {
		return nil
	}
	if err := rebindSwitchSourceRestore(ctx, e); err != nil {
		return err
	}
	if !validInstanceRestoreArchive(e.ColdArchive) {
		return errInstanceRecovery
	}
	b := instanceRuntimes
	claim := &instanceRestoreClaim{s: s, maintenance: lock.file}
	defer func() { lock.file = claim.maintenance }()
	if journal, err := b.restoreJournal(e.Source); err == nil && journal.Archive == e.ColdArchive {
		if journal.Stage == "completed" {
			e.WorldChanged = false
			return saveSwitchEvidence(*e)
		}
		if !restoreTerminal(journal.Stage) {
			if err = claim.recover(ctx, e.Source); err != nil {
				return err
			}
		}
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	out, err := b.restoreStorage(ctx, "plan", e.Source, e.ColdArchive, "", false)
	var inspection instanceRestoreInspection
	if err != nil || instanceDecode(out, &inspection) != nil || inspection.Metadata.ID != e.Source || inspection.Metadata.Software != e.SourceRecord.Runtime.Software || inspection.Metadata.Version != e.Version || inspection.Metadata.Archive != e.ColdArchive || inspection.Integrity != "verified" || !instanceSHA256Pattern.MatchString(inspection.Fingerprint) {
		return errInstanceRecovery
	}
	plan := instanceRestorePlan{instanceRestoreInspection: inspection, RuntimeState: "stopped"}
	if err = s.runInstanceRestore(ctx, e.Source, plan, claim); err != nil {
		return err
	}
	e.WorldChanged = false
	return saveSwitchEvidence(*e)
}

// Only terminal evidence belonging to the journal captured before this Switch,
// or its successfully restored cold archive, may follow temporary config edits.
// Active restore journals keep their exact binding and existing recovery path.
func rebindSwitchSourceRestore(ctx context.Context, e *minecraftSwitchEvidence) error {
	// Deployed Switch evidence has neither field; preserve its recovery contract.
	if e.SourceRestoreJournal == nil && e.ColdArchive == "" {
		return nil
	}
	b := instanceRuntimes
	journal, err := b.restoreJournal(e.Source)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !restoreTerminal(journal.Stage) {
		return nil
	}
	raw, err := b.readPrivate(filepath.Join(b.root, e.Source, "config.json"))
	if err != nil {
		return err
	}
	if journal.ConfigSHA256 == instanceHash(raw) {
		return nil
	}
	authorized := !e.WorldChanged && journal.Archive == e.ColdArchive && journal.Stage == "completed"
	if e.SourceRestoreJournal != nil {
		previous := journal
		previous.ConfigSHA256 = e.SourceRestoreJournal.ConfigSHA256
		authorized = authorized || reflect.DeepEqual(previous, *e.SourceRestoreJournal)
	}
	if !authorized {
		return errInstanceRecovery
	}
	record, err := b.readRecord(e.Source)
	if err != nil || e.SourceRecord == nil || (record.Runtime.Software != e.SourceRecord.Runtime.Software && (record.Runtime.Software != e.Target || record.Runtime.Version != e.Version)) || record.Storage != e.SourceRecord.Storage {
		return errInstanceRecovery
	}
	if err = b.verifyLive(record); err != nil {
		return err
	}
	journal.ConfigSHA256 = instanceHash(raw)
	return b.writeJSON(filepath.Join(b.root, e.Source, "restore-journal.json"), journal, true)
}
