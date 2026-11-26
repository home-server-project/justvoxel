package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

type switchMatrixBackupFile struct {
	Inode  [2]uint64 `json:"inode"`
	Size   int64     `json:"size"`
	SHA256 string    `json:"sha256"`
}

type switchMatrixBackups struct {
	Root      [2]uint64                         `json:"root"`
	Directory [2]uint64                         `json:"directory"`
	Files     map[string]switchMatrixBackupFile `json:"files"`
}

type switchMatrixFixture struct {
	Parent          string                        `json:"parent"`
	ID              string                        `json:"id"`
	Stage           string                        `json:"stage"`
	Name            string                        `json:"name"`
	CandidateSHA256 string                        `json:"candidate_sha256,omitempty"`
	Candidate       *instanceRuntimeRecord        `json:"candidate,omitempty"`
	ConfigSHA256    string                        `json:"config_sha256"`
	UID             uint32                        `json:"uid"`
	GID             uint32                        `json:"gid"`
	DataParent      [2]uint64                     `json:"data_parent"`
	DataIdentity    string                        `json:"data_identity"`
	Storage         instanceRuntimeStorage        `json:"storage"`
	Placement       *instanceRuntimeBackupStorage `json:"placement"`
	Backups         switchMatrixBackups           `json:"backups"`
	ModSHA256       string                        `json:"mod_sha256,omitempty"`
}

func matrixFixturePath(parent, id string) string {
	return filepath.Join(instanceRuntimes.staging, "matrix-"+parent+"-"+id+".json")
}

func readSwitchMatrixFixture(parent, id string) (switchMatrixFixture, error) {
	var f switchMatrixFixture
	if !validOperationID(parent) || !instanceIDPattern.MatchString(id) {
		return f, errInstanceID
	}
	raw, err := instanceRuntimes.readPrivate(matrixFixturePath(parent, id))
	if err != nil {
		return f, err
	}
	if instanceDecode(raw, &f) != nil || f.Parent != parent || f.ID != id || f.Placement == nil ||
		validateInstanceRecordPlacement(f.placementRecord()) != nil {
		return f, errInstanceRecovery
	}
	if f.Stage == "reserved" || f.Stage == "retired" && f.ConfigSHA256 == "" {
		if instanceSHA256Pattern.MatchString(f.CandidateSHA256) && f.ConfigSHA256 == "" &&
			f.DataIdentity == "" && f.DataParent == [2]uint64{} && f.Backups.Directory == [2]uint64{} &&
			(f.Candidate == nil || f.Stage == "reserved" && f.validCandidate()) {
			return f, nil
		}
		return f, errInstanceRecovery
	}
	if f.Candidate != nil {
		return f, errInstanceRecovery
	}
	if f.Name == "" || !instanceSHA256Pattern.MatchString(f.ConfigSHA256) || f.DataIdentity == "" || f.DataParent[1] == 0 {
		return f, errInstanceRecovery
	}
	switch f.Stage {
	case "binding", "owned", "intent", "instance_delete", "retired":
		return f, nil
	default:
		return f, errInstanceRecovery
	}
}

func (f switchMatrixFixture) validCandidate() bool {
	r := f.Candidate
	name, err := validateInstanceName(f.Name)
	return r != nil && r.SchemaVersion == 1 && r.Runtime.MatrixFixture && f.checkRecord(*r) == nil &&
		instanceHash(mustSwitchJSON(r.Runtime)) == f.CandidateSHA256 && len(r.Firewall) == 0 &&
		r.EnvironmentSHA256 == "" && r.QuadletSHA256 == "" && err == nil && name != "" && name == f.Name
}

func (f switchMatrixFixture) save(replace bool) error {
	return instanceRuntimes.writeJSON(matrixFixturePath(f.Parent, f.ID), f, replace)
}

func (f switchMatrixFixture) prepareReceiptDirectory() error {
	dir, err := instanceRuntimes.privateDirectory(instanceRuntimes.staging, true)
	if err != nil {
		return err
	}
	return dir.Close()
}

func (f switchMatrixFixture) placementRecord() instanceRuntimeRecord {
	return instanceRuntimeRecord{Storage: f.Storage, Backups: f.Placement,
		Runtime: minecraftInstanceRuntimeConfig{ID: f.ID, UID: f.UID, GID: f.GID}}
}

func (f switchMatrixFixture) checkRecord(record instanceRuntimeRecord) error {
	if record.Runtime.ID != f.ID || record.Runtime.StartOnBoot || record.NetworkMember ||
		record.Runtime.UID != f.UID || record.Runtime.GID != f.GID || record.Storage != f.Storage || !reflect.DeepEqual(record.Backups, f.Placement) ||
		validateInstanceRecordPlacement(record) != nil {
		return errInstanceRecovery
	}
	return nil
}

func switchMatrixConfigHash(id string) (string, error) {
	raw, err := instanceRuntimes.readPrivate(filepath.Join(instanceRuntimes.root, id, "config.json"))
	if err != nil {
		return "", err
	}
	return instanceHash(raw), nil
}

type switchMatrixDataIdentity struct {
	Data   string    `json:"identity"`
	Parent [2]uint64 `json:"parent_identity"`
}

func inspectSwitchMatrixData(ctx context.Context, record instanceRuntimeRecord) (switchMatrixDataIdentity, error) {
	var result switchMatrixDataIdentity
	out, err := instanceRuntimes.instanceDeletionStorage(ctx, "matrix-inspect", instanceDeleteJournal{ID: record.Runtime.ID, Record: record})
	if err != nil || instanceDecode(out, &result) != nil || result.Data == "" || result.Parent[1] == 0 {
		return result, errInstanceRecovery
	}
	return result, nil
}

func (f switchMatrixFixture) checkData(ctx context.Context, record instanceRuntimeRecord) error {
	identity, err := inspectSwitchMatrixData(ctx, record)
	if err != nil || identity.Data != f.DataIdentity || identity.Parent != f.DataParent {
		return errInstanceRecovery
	}
	return nil
}

// Recheck the retained directory even when a previous Delete attempt
// already removed its data. Only the journal's post-shutdown stages allow
// an absent world.
func (f switchMatrixFixture) checkDeleteRetry(ctx context.Context, j instanceDeleteJournal) error {
	evidence := struct {
		Parent          [2]uint64 `json:"parent"`
		Data            string    `json:"data"`
		DataMayBeAbsent bool      `json:"data_may_be_absent"`
	}{f.DataParent, f.DataIdentity, j.Stage != "intent"}
	j.DataIdentity = string(mustSwitchJSON(evidence))
	_, err := instanceRuntimes.instanceDeletionStorage(ctx, "matrix-retry", j)
	return err
}

func (f switchMatrixFixture) backupStorage(ctx context.Context, action string) (switchMatrixBackups, error) {
	var result switchMatrixBackups
	args := []string{"--mount=/proc/1/ns/mnt", "--", instanceBackupHelper, action, f.ID, f.Parent}
	out, err := instanceBackupStorageCommand(ctx, "nsenter", args...)
	if err != nil {
		return result, errInstanceRecovery
	}
	if (action == "matrix-prepare" || action == "matrix-inventory") && (instanceDecode(out, &result) != nil || result.Root[1] == 0 ||
		result.Directory[1] == 0 || result.Files == nil) {
		return result, errInstanceRecovery
	}
	return result, nil
}

func (s *operationStore) bindSwitchMatrixFixture(ctx context.Context, parent, id string) error {
	s.mu.Lock()
	p := s.operations[parent]
	owned := parent == s.currentMinecraftResetID && p.OperationType == operationTypeSwitchMatrix &&
		(p.State == operationRunning && (p.Stage == "fixtures" || p.Stage == "switch") ||
			p.State == operationNeedsAttention) && slices.Contains(p.MatrixFixtures, id)
	s.mu.Unlock()
	if !owned {
		return errInstanceRecovery
	}
	f, err := readSwitchMatrixFixture(parent, id)
	if errors.Is(err, os.ErrNotExist) || err == nil && f.Stage == "reserved" {
		f, err = captureSwitchMatrixBinding(ctx, parent, id, err == nil)
	}
	if err != nil || f.Stage != "binding" {
		return errInstanceRecovery
	}
	return f.finishBinding(ctx)
}

func captureSwitchMatrixBinding(ctx context.Context, parent, id string, replace bool) (switchMatrixFixture, error) {
	var f switchMatrixFixture
	b := instanceRuntimes
	record, err := b.readRecord(id)
	if err != nil || record.Runtime.ID != id || record.Runtime.StartOnBoot || record.NetworkMember ||
		record.Backups == nil || validateInstanceRecordPlacement(record) != nil || b.verifyLive(record) != nil {
		return f, errInstanceRecovery
	}
	identity, err := inspectSwitchMatrixData(ctx, record)
	if err != nil {
		return f, err
	}
	hash, err := switchMatrixConfigHash(id)
	if err != nil {
		return f, err
	}
	names, err := b.identities().readNames()
	if err != nil || names[id].DisplayName == "" {
		return f, errInstanceRecovery
	}
	f = switchMatrixFixture{Parent: parent, ID: id, Stage: "binding", Name: names[id].DisplayName,
		ConfigSHA256: hash, DataIdentity: identity.Data, DataParent: identity.Parent, UID: record.Runtime.UID, GID: record.Runtime.GID, Storage: record.Storage, Placement: record.Backups}
	if err = f.prepareReceiptDirectory(); err != nil {
		return f, err
	}
	return f, f.save(replace)
}

func (f switchMatrixFixture) finishBinding(ctx context.Context) error {
	b := instanceRuntimes
	record, err := b.readRecord(f.ID)
	if err != nil || f.checkRecord(record) != nil || b.verifyLive(record) != nil || f.checkData(ctx, record) != nil {
		return errInstanceRecovery
	}
	hash, err := switchMatrixConfigHash(f.ID)
	if err != nil || hash != f.ConfigSHA256 {
		return errInstanceRecovery
	}
	if f.Backups.Directory[1] == 0 {
		f.Backups, err = f.backupStorage(ctx, "matrix-prepare")
		if err != nil {
			return err
		}
		// Persist the staged inode before it can become the live namespace.
		if err = f.save(true); err != nil {
			return err
		}
	}
	if _, err = f.backupStorage(ctx, "matrix-bind"); err != nil {
		return err
	}
	f.Stage = "owned"
	return f.save(true)
}

func (s *operationStore) checkSwitchMatrixFixture(ctx context.Context, parent, id string) error {
	f, err := readSwitchMatrixFixture(parent, id)
	if err != nil || f.Stage != "owned" {
		return errInstanceRecovery
	}
	record, err := instanceRuntimes.readRecord(id)
	if err != nil || f.checkRecord(record) != nil {
		return errInstanceRecovery
	}
	hash, err := switchMatrixConfigHash(id)
	if err != nil || hash != f.ConfigSHA256 {
		return errInstanceRecovery
	}
	if err = f.checkData(ctx, record); err != nil {
		return err
	}
	_, err = f.backupStorage(ctx, "matrix-inventory")
	return err
}

func (s *operationStore) createSwitchMatrixFixture(ctx context.Context, parent string, c minecraftInstanceRuntimeConfig, name string, snapshot minecraftInstanceRuntimeContext) (string, error) {
	if !s.switchMatrixMu.TryLock() {
		return "", errInstanceBusy
	}
	defer s.switchMatrixMu.Unlock()
	s.mu.Lock()
	_, err := s.switchMatrixParent(parent, "fixtures", false)
	busy := s.activeSwitchMatrixChild(parent)
	s.mu.Unlock()
	if err != nil || busy || c.StartOnBoot || c.ID != "" || snapshot.BackupPlacement == nil {
		return "", errInstanceRecovery
	}
	lock, err := acquirePluginMaintenanceLock()
	if err != nil {
		return "", err
	}
	defer lock.Close()
	id, err := instanceRuntimes.createTrackedCandidate(ctx, c, name, snapshot, func(record instanceRuntimeRecord) error {
		return s.reserveSwitchMatrixFixture(parent, record, operationJournal{}, name)
	})
	if err != nil {
		return id, err
	}
	return id, s.bindSwitchMatrixFixture(ctx, parent, id)
}

func (s *operationStore) refreshSwitchMatrixFixture(ctx context.Context, parent string, e minecraftSwitchEvidence, recovered bool) error {
	f, err := readSwitchMatrixFixture(parent, e.Source)
	if err != nil || f.Stage != "owned" {
		return errInstanceRecovery
	}
	b := instanceRuntimes
	record, err := b.readRecord(f.ID)
	if err != nil || f.checkRecord(record) != nil || b.verifyLive(record) != nil {
		return errInstanceRecovery
	}
	hash, err := switchMatrixConfigHash(f.ID)
	if err != nil {
		return err
	}
	data, err := inspectSwitchMatrixData(ctx, record)
	if err != nil || data.Parent != f.DataParent {
		return errInstanceRecovery
	}
	if data.Data != f.DataIdentity {
		// The only matrix transaction allowed to replace the source world is
		// verified cold-archive rollback. Bind both inodes and the restored
		// configuration before accepting the replacement as test-owned.
		j, err := b.restoreJournal(f.ID)
		token := func(pair []uint64) string {
			if len(pair) != 2 {
				return ""
			}
			return strconv.FormatUint(pair[0], 10) + ":" + strconv.FormatUint(pair[1], 10)
		}
		if !recovered || e.Action != "convert" || e.ColdArchive == "" || err != nil ||
			j.Archive != e.ColdArchive || j.Stage != "completed" || j.ConfigSHA256 != hash ||
			!slices.Equal(j.Parent, f.DataParent[:]) || token(j.Original) != f.DataIdentity || token(j.Replacement) != data.Data {
			return errInstanceRecovery
		}
		f.DataIdentity = data.Data
	}
	f.ConfigSHA256 = hash
	return f.save(true)
}

const matrixCrossplayOwnershipHelper = "/usr/libexec/justvoxel/mjust/managed-crossplay-mod-ownership"

// Verify the exact canonical mod evidence in the host mount namespace. Empty
// evidence is meaningful only where the existing managed stack permits it.
func (f switchMatrixFixture) crossplayOwnership(ctx context.Context, record instanceRuntimeRecord) (string, error) {
	b := instanceRuntimes
	path := filepath.Join(b.root, f.ID, "crossplay-mods.json")
	raw, err := b.readPrivate(path)
	isModPlatform := record.Runtime.Software == "fabric" || record.Runtime.Software == "neoforge"
	if errors.Is(err, os.ErrNotExist) && (!isModPlatform || !record.Runtime.BedrockEnabled) {
		return "", nil
	}
	if err != nil || !isModPlatform {
		return "", errInstanceRecovery
	}
	_, err = b.command(ctx, "nsenter", "--mount=/proc/1/ns/mnt", "--", matrixCrossplayOwnershipHelper,
		"retire-verify", record.Runtime.Software, record.Storage.DataPath, path, strconv.FormatBool(record.Runtime.BedrockEnabled))
	if err != nil {
		return "", errInstanceRecovery
	}
	after, err := b.readPrivate(path)
	if err != nil || instanceHash(raw) != instanceHash(after) {
		return "", errInstanceRecovery
	}
	return instanceHash(raw), nil
}

// The verified Switch recovery path may retire a failed clone before the
// fixture receipt exists. Only that child may retire its loader metadata.
func (s *operationStore) retireFailedMatrixCloneMods(ctx context.Context, child operationJournal, e minecraftSwitchEvidence, record instanceRuntimeRecord) error {
	s.mu.Lock()
	p, err := s.switchMatrixParent(child.MatrixParent, "switch", true)
	j := s.operations[child.OperationID]
	s.mu.Unlock()
	if err != nil || j.OperationID != e.OperationID || j.InstanceID != e.Source || j.MatrixParent != p.OperationID ||
		(j.State != operationRunning && j.State != operationNeedsAttention) ||
		e.CloneID != record.Runtime.ID || !slices.Contains(p.MatrixFixtures, e.CloneID) {
		return errInstanceRecovery
	}
	b := instanceRuntimes
	mod := filepath.Join(b.root, e.CloneID, "crossplay-mods.json")
	if _, err = os.Lstat(mod); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	f := switchMatrixFixture{Parent: p.OperationID, ID: e.CloneID}
	mods, err := f.crossplayOwnership(ctx, record)
	if err != nil || mods == "" {
		return errInstanceRecovery
	}
	config, err := switchMatrixConfigHash(e.CloneID)
	if err != nil {
		return err
	}
	data, err := inspectSwitchMatrixData(ctx, record)
	if err != nil {
		return err
	}
	intent := struct {
		Child, Clone, Config, Mods string
		Data                       switchMatrixDataIdentity
	}{j.OperationID, e.CloneID, config, mods, data}
	path := filepath.Join(b.staging, "matrix-mod-retirement-"+j.OperationID+".json")
	raw, err := b.readPrivate(path)
	if errors.Is(err, os.ErrNotExist) {
		err = b.writeJSON(path, intent, false)
	} else if err == nil {
		var saved = intent
		if instanceDecode(raw, &saved) != nil || saved != intent {
			return errInstanceRecovery
		}
	}
	if err != nil {
		return err
	}
	return b.removePrivate(mod)
}

func (s *operationStore) retireSwitchMatrixFixture(ctx context.Context, parent, id string) error {
	if !s.switchMatrixMu.TryLock() {
		return errInstanceBusy
	}
	defer s.switchMatrixMu.Unlock()
	s.mu.Lock()
	p, err := s.switchMatrixParent(parent, "retirement", true)
	busy := s.activeSwitchMatrixChild(parent)
	s.mu.Unlock()
	if err != nil || busy || !slices.Contains(p.MatrixFixtures, id) {
		return errInstanceRecovery
	}
	lock, err := acquirePluginMaintenanceLock()
	if err != nil {
		return err
	}
	defer lock.Close()
	err = retireSwitchMatrixFixtureLocked(ctx, parent, id)
	status := "Fixture retirement verified"
	if err != nil {
		status = "Fixture retirement incomplete; evidence preserved"
		if p.State == operationRunning {
			_, transitionErr := s.transition(parent, operationNeedsAttention, "fixture_retirement", status)
			err = errors.Join(err, transitionErr)
		}
	}
	if p.PlanFingerprint != switchMatrixRunnerFingerprint() {
		_ = s.appendOperationDiagnostic(parent, operationTypeSwitchMatrix, "fixture_retirement", status, map[string]string{"instance_id": id})
	}
	return err
}

func retireSwitchMatrixFixtureLocked(ctx context.Context, parent, id string) error {
	b := instanceRuntimes
	f, err := readSwitchMatrixFixture(parent, id)
	if err != nil {
		return err
	}
	if f.Stage == "retired" {
		return nil
	}
	if f.Stage == "reserved" {
		record, err := b.readRecord(id)
		if errors.Is(err, errInstanceNotFound) || errors.Is(err, os.ErrNotExist) {
			return f.retireUnusedReservation(ctx)
		}
		// Creation is the only operation permitted before binding. Require
		// the exact candidate and a stopped, still-empty world.
		if err != nil || f.checkRecord(record) != nil ||
			instanceHash(mustSwitchJSON(record.Runtime)) != f.CandidateSHA256 ||
			b.deleteStopped(ctx, id, false) != nil || b.storage(ctx, "version-empty", record) != nil {
			return errInstanceRecovery
		}
		f, err = captureSwitchMatrixBinding(ctx, parent, id, true)
		if err != nil {
			return err
		}
	}
	if f.Stage == "instance_delete" {
		if restore, err := b.restoreJournal(id); err == nil && !restoreTerminal(restore.Stage) ||
			err != nil && !errors.Is(err, os.ErrNotExist) {
			return errInstanceRecovery
		}
		return f.finishDelete(ctx)
	}
	if err = b.requireNoInstanceRestore(); err != nil {
		return err
	}
	record, err := b.readRecord(id)
	if err != nil || f.checkRecord(record) != nil || b.verifyLive(record) != nil || f.checkData(ctx, record) != nil {
		return errInstanceRecovery
	}
	hash, err := switchMatrixConfigHash(id)
	if err != nil || hash != f.ConfigSHA256 || b.deleteStopped(ctx, id, false) != nil {
		return errInstanceRecovery
	}
	// Refuse unknown private state before changing anything. Ordinary deletion
	// keeps its original allowlist and never consumes cross-play ownership.
	dir, _, _ := b.paths(id)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.Name() != "crossplay-mods.json" && !instanceDeleteKnownState(entry.Name()) {
			return errInstanceRecovery
		}
	}
	if _, err = b.readPrivate(b.backupPath(id, "schedule-pending")); !errors.Is(err, os.ErrNotExist) {
		return errInstanceRecovery
	}
	status, err := b.backupStatus(id)
	if err != nil || status.State == "running" || status.RestartRequired {
		return errInstanceRecovery
	}
	checkBackups, retireBackups := "matrix-check", "matrix-retire"
	if f.Backups.Directory[1] == 0 {
		checkBackups, retireBackups = "matrix-absent", "matrix-absent"
	}
	modPath := filepath.Join(dir, "crossplay-mods.json")
	if f.Stage == "owned" || f.Stage == "binding" {
		f.ModSHA256, err = f.crossplayOwnership(ctx, record)
		if err != nil {
			// A loader fixture can fail before installing managed mods. Missing
			// ownership is safe only with a positively verified empty data tree.
			_, statErr := os.Lstat(modPath)
			if (f.Stage != "binding" && !record.Runtime.MatrixFixture) ||
				!errors.Is(statErr, os.ErrNotExist) || b.storage(ctx, "version-empty", record) != nil {
				return err
			}
		}
		if f.Stage == "owned" {
			f.Backups, err = f.backupStorage(ctx, "matrix-inventory")
		} else {
			_, err = f.backupStorage(ctx, checkBackups)
		}
		if err != nil {
			return err
		}
		f.Stage = "intent"
		if err = f.save(true); err != nil {
			return err
		}
	} else if _, err = f.backupStorage(ctx, checkBackups); err != nil {
		return err
	}
	_, err = os.Lstat(modPath)
	hasModOwnership := err == nil
	if hasModOwnership {
		modHash, verifyErr := f.crossplayOwnership(ctx, record)
		if verifyErr != nil || modHash == "" || modHash != f.ModSHA256 {
			return errInstanceRecovery
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	// Retire verified Restore storage only after the record, backup inventory
	// and mod ownership checks pass. The terminal journal stays until Delete.
	if err = f.retireRestore(ctx, record); err != nil {
		return err
	}
	if hasModOwnership {
		if err = b.removePrivate(modPath); err != nil {
			return err
		}
	}
	privateDir, err := b.privateDirectory(dir, false)
	if err != nil {
		return err
	}
	err = errors.Join(privateDir.Sync(), privateDir.Close())
	if err != nil {
		return err
	}
	if _, err = f.backupStorage(ctx, retireBackups); err != nil {
		return err
	}
	f.Stage = "instance_delete"
	if err = f.save(true); err != nil {
		return err
	}
	return f.finishDelete(ctx)
}

func (f switchMatrixFixture) retireUnusedReservation(ctx context.Context) error {
	b := instanceRuntimes
	id := f.ID
	registry := b.identities()
	legacy, err := registry.read()
	if err != nil || legacy.ID == id {
		return errInstanceRecovery
	}
	dir, _, quadlet := b.paths(id)
	for _, path := range []string{
		filepath.Join(registry.directory, id+".json"),
		dir, quadlet, b.deletePath(id),
		filepath.Join(instanceBackupUnits, instanceBackupUnit(id, "timer")),
		filepath.Join(instanceBackupUnits, instanceBackupUnit(id, "service")),
	} {
		file, err := instanceOpen(path, unix.O_PATH)
		if file != nil {
			file.Close()
		}
		if !errors.Is(err, os.ErrNotExist) {
			return errInstanceRecovery
		}
	}
	names, err := registry.readNames()
	if err != nil {
		return err
	}
	name, nameExists := names[id]
	if nameExists && (!f.validCandidate() || name.DisplayName != f.Name) {
		return errInstanceRecovery
	}
	state, err := b.observeRaw(ctx, id)
	if err != nil || state.State != "configured" || b.deleteStopped(ctx, id, true) != nil {
		return errInstanceRecovery
	}
	for _, suffix := range []string{"timer", "service"} {
		out, err := b.command(ctx, "systemctl", "show", "--property=LoadState", "--value", "--", instanceBackupUnit(id, suffix))
		if err != nil || strings.TrimSpace(string(out)) != "not-found" {
			return errInstanceRecovery
		}
	}
	if err = b.switchCloneRetirementStorage(ctx, "matrix-absent", f.placementRecord(), "-"); err != nil {
		return err
	}
	if _, err = f.backupStorage(ctx, "matrix-absent"); err != nil {
		return err
	}
	if nameExists {
		if err = b.removePrivate(filepath.Join(registry.directory, "names", id+".json")); err != nil {
			return err
		}
	}
	f.Stage = "retired"
	f.Candidate = nil
	return f.save(true)
}

func (f switchMatrixFixture) retireRestore(ctx context.Context, record instanceRuntimeRecord) error {
	b := instanceRuntimes
	j, err := b.restoreJournal(f.ID)
	if errors.Is(err, os.ErrNotExist) {
		return b.switchCloneRetirementStorage(ctx, "unrestored", record, "-")
	}
	if err != nil || !restoreTerminal(j.Stage) || j.ConfigSHA256 != f.ConfigSHA256 || len(j.Parent) != 2 ||
		len(j.Original) != 2 || (j.Stage == "completed" && len(j.Replacement) != 2) {
		return errInstanceRecovery
	}
	raw, err := json.Marshal(map[string]any{"id": f.ID, "stage": j.Stage, "transaction": j.Transaction,
		"parent": j.Parent, "original": j.Original, "replacement": j.Replacement, "data_may_be_absent": false})
	if err != nil {
		return err
	}
	return b.switchCloneRetirementStorage(ctx, "journal", record, string(raw))
}

func (f switchMatrixFixture) finishDelete(ctx context.Context) error {
	b := instanceRuntimes
	if j, err := b.deleteJournal(f.ID); err == nil {
		if j.Name != f.Name || j.DataIdentity != f.DataIdentity || f.checkRecord(j.Record) != nil ||
			instanceHash(append(mustSwitchJSON(j.Record), '\n')) != f.ConfigSHA256 {
			return errInstanceRecovery
		}
		if err = f.checkDeleteRetry(ctx, j); err != nil {
			return err
		}
		if err = b.retireSelectedInstanceOwned(ctx, f.ID, f.Name, "", f.DataIdentity); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	} else if record, err := b.readRecord(f.ID); err == nil {
		hash, hashErr := switchMatrixConfigHash(f.ID)
		if f.checkRecord(record) != nil || hashErr != nil || hash != f.ConfigSHA256 ||
			f.checkData(ctx, record) != nil {
			return errInstanceRecovery
		}
		names, err := b.identities().readNames()
		if err != nil || names[f.ID].DisplayName == "" {
			return errInstanceRecovery
		}
		// Presentation may change before Delete writes its own journal. The
		// immutable ID, committed configuration and storage are already bound.
		if name := names[f.ID].DisplayName; name != f.Name {
			f.Name = name
			if err = f.save(true); err != nil {
				return err
			}
		}
		if err = b.retireSelectedInstanceOwned(ctx, f.ID, f.Name, "", f.DataIdentity); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) && !errors.Is(err, errInstanceNotFound) {
		return err
	}
	if _, err := b.identities().readRecord(f.ID + ".json"); !errors.Is(err, os.ErrNotExist) {
		return errInstanceRecovery
	}
	dir, env, quadlet := b.paths(f.ID)
	for _, path := range []string{dir, env, quadlet, b.deletePath(f.ID)} {
		if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
			return errInstanceRecovery
		}
	}
	committedPlacement := instanceRuntimeRecord{Storage: f.Storage,
		Runtime: minecraftInstanceRuntimeConfig{ID: f.ID, UID: f.UID, GID: f.GID}}
	if err := b.switchCloneRetirementStorage(ctx, "matrix-empty", committedPlacement, string(mustSwitchJSON(f.DataParent))); err != nil {
		return err
	}
	names, err := b.identities().readNames()
	if err != nil {
		return err
	}
	if name, ok := names[f.ID]; ok {
		if name.DisplayName != f.Name {
			return errInstanceRecovery
		}
		if err = b.removePrivate(filepath.Join(b.identities().directory, "names", f.ID+".json")); err != nil {
			return err
		}
	}
	f.Stage = "retired"
	return f.save(true)
}

func (s *operationStore) verifySwitchMatrixClosure(j operationJournal) error {
	if s.activeSwitchMatrixChild(j.OperationID) {
		return errInstanceRecovery
	}
	for _, id := range j.MatrixFixtures {
		f, err := readSwitchMatrixFixture(j.OperationID, id)
		if errors.Is(err, os.ErrNotExist) && s.switchMatrixRecoveredClone(j.OperationID, id) {
			continue
		}
		if err != nil || f.Stage != "retired" {
			return errInstanceRecovery
		}
	}
	return nil
}

// A failed clone can be fully retired by ordinary Switch recovery before it
// ever receives an owned-fixture receipt or creates a backup directory.
func (s *operationStore) switchMatrixRecoveredClone(parent, id string) bool {
	for _, j := range s.operations {
		if j.MatrixParent != parent || j.State != operationResolved {
			continue
		}
		e, err := readSwitchEvidence(j)
		if err == nil && e.CloneID == id && e.Stage == "recovered" {
			return true
		}
	}
	return false
}
