package main

import (
	"context"
	"errors"
	"os"
	"regexp"
	"slices"
	"time"
)

type switchMatrixChildKey struct{}

var errSwitchMatrixPolicy = errors.New("production Switch policy blocked this case")
var errSwitchMatrixReleaseChanged = errors.New("reviewed exact runtime changed")

type switchMatrixChildFailure struct{ Stage string }

func (*switchMatrixChildFailure) Error() string { return "production Switch failed" }

func validateSwitchMatrixJournal(j operationJournal) error {
	if j.MatrixParent != "" && (j.OperationType != operationTypeMinecraftSwitch ||
		!validOperationID(j.MatrixParent) || j.MatrixParent == j.OperationID || !instanceIDPattern.MatchString(j.InstanceID)) {
		return errInstanceRecovery
	}
	if j.OperationType != operationTypeSwitchMatrix {
		if len(j.MatrixFixtures) != 0 {
			return errInstanceRecovery
		}
		return nil
	}
	if j.InstanceID != "" || len(j.MatrixFixtures) > 4096 {
		return errInstanceRecovery
	}
	seen := make(map[string]bool, len(j.MatrixFixtures))
	for _, id := range j.MatrixFixtures {
		if !instanceIDPattern.MatchString(id) || seen[id] {
			return errInstanceRecovery
		}
		seen[id] = true
	}
	return nil
}

func (s *operationStore) reconcileSwitchMatrixChildren() error {
	active := false
	for id, j := range s.operations {
		if j.MatrixParent == "" {
			continue
		}
		parent := s.operations[j.MatrixParent]
		if parent.OperationType != operationTypeSwitchMatrix || !slices.Contains(parent.MatrixFixtures, j.InstanceID) {
			return errInstanceRecovery
		}
		if !operationIsCurrent(j.State) {
			continue
		}
		if active || s.currentMinecraftResetID != j.MatrixParent || parent.State != operationNeedsAttention {
			return errInstanceRecovery
		}
		active = true
		if !operationInterruptedByRestart(j.State) {
			continue
		}
		now := s.now().UTC().Format(time.RFC3339Nano)
		j.State, j.Stage = operationNeedsAttention, "interrupted"
		j.Status = "Matrix Switch interrupted. Recovery evidence was preserved."
		j.UpdatedAt, j.InterruptedAt = now, now
		if err := s.persist(j); err != nil {
			return err
		}
		s.operations[id] = j
		s.appendOperationJournalDiagnosticBestEffort(j)
	}
	return nil
}

func (s *operationStore) beginSwitchMatrix(fingerprint string) (operationJournal, error) {
	// An independent backup releases the runtime flock before restarting.
	// Its worker gate still covers restart, retention, and the final status.
	if !instanceBackupWorkerMu.TryLock() {
		return operationJournal{}, errInstanceBusy
	}
	defer instanceBackupWorkerMu.Unlock()
	j, _, err := s.beginMinecraftMaintenance(fingerprint, operationTypeSwitchMatrix)
	return j, err
}

// Caller holds the store mutex. Parent authority is private to the worker and
// is never accepted from a request or propagated through shell environment.
func (s *operationStore) switchMatrixParent(id, stage string, recovery bool) (operationJournal, error) {
	j := s.operations[id]
	if id != s.currentMinecraftResetID || j.OperationType != operationTypeSwitchMatrix || s.switchMatrixExclusion == nil {
		return operationJournal{}, errInstanceBusy
	}
	if (j.State == operationRunning && j.Stage == stage) || (recovery && j.State == operationNeedsAttention) {
		return j, nil
	}
	return operationJournal{}, errInstanceBusy
}

func (s *operationStore) activeSwitchMatrixChild(parent string) bool {
	for _, j := range s.operations {
		if j.MatrixParent == parent && operationIsCurrent(j.State) {
			return true
		}
	}
	return false
}

func (s *operationStore) reserveSwitchMatrixFixture(parent string, candidate instanceRuntimeRecord, child operationJournal, name ...string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	stage := "fixtures"
	if child.OperationID != "" {
		stage = "switch"
		actual := s.operations[child.OperationID]
		if actual.MatrixParent != parent || actual.InstanceID != child.InstanceID ||
			actual.State != operationRunning || actual.Stage != "converting" {
			return errInstanceBusy
		}
	}
	j, err := s.switchMatrixParent(parent, stage, false)
	if err != nil {
		return err
	}
	id := candidate.Runtime.ID
	if !instanceIDPattern.MatchString(id) || slices.Contains(j.MatrixFixtures, id) {
		return errInstanceID
	}
	identity, err := instanceRuntimes.identities().read()
	if err != nil || identity.ID == id {
		return errInstanceID
	}
	if _, err := instanceRuntimes.identities().readRecord(id + ".json"); !errors.Is(err, os.ErrNotExist) {
		return errInstanceID
	}
	dir, env, quadlet := instanceRuntimes.paths(id)
	for _, path := range []string{dir, env, quadlet, instanceRuntimes.deletePath(id)} {
		if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
			return errInstanceID
		}
	}
	names, err := instanceRuntimes.identities().readNames()
	if err != nil || names[id].ID != "" {
		return errInstanceID
	}
	if candidate.Runtime.StartOnBoot || candidate.NetworkMember ||
		candidate.Backups == nil || validateInstanceRecordPlacement(candidate) != nil {
		return errInstanceRecovery
	}
	f := switchMatrixFixture{Parent: parent, ID: id, Stage: "reserved",
		CandidateSHA256: instanceHash(mustSwitchJSON(candidate.Runtime)),
		Storage:         candidate.Storage, Placement: candidate.Backups, UID: candidate.Runtime.UID, GID: candidate.Runtime.GID}
	if candidate.Runtime.MatrixFixture {
		if len(name) != 1 {
			return errInstanceRecovery
		}
		validated, err := validateInstanceName(name[0])
		if err != nil || validated != name[0] || validated == "" {
			return errInstanceRecovery
		}
		f.Name, f.Candidate = validated, &candidate
		if !f.validCandidate() {
			return errInstanceRecovery
		}
	}
	// No creation can follow until both this receipt and the reservation
	// are durable. A receipt alone does not authorize any resource access.
	if err = f.prepareReceiptDirectory(); err != nil {
		return err
	}
	if err = f.save(false); err != nil {
		return err
	}
	j.MatrixFixtures = append(slices.Clone(j.MatrixFixtures), id)
	j.UpdatedAt = s.now().UTC().Format(time.RFC3339Nano)
	if err = s.persist(j); err != nil {
		return err
	}
	s.operations[parent] = j
	return nil
}

func (s *operationStore) beginSwitchMatrixChild(parent, id, fingerprint string) (operationJournal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.switchMatrixParent(parent, "switch", false)
	if err != nil || s.activeSwitchMatrixChild(parent) {
		return operationJournal{}, errInstanceBusy
	}
	if !slices.Contains(p.MatrixFixtures, id) {
		return operationJournal{}, errInstanceID
	}
	child, err := newOperationID()
	if err != nil {
		return operationJournal{}, err
	}
	now := s.now().UTC().Format(time.RFC3339Nano)
	j := operationJournal{
		SchemaVersion: operationSchemaVersion, OperationID: child, OperationType: operationTypeMinecraftSwitch,
		MatrixParent: parent, InstanceID: id, PlanFingerprint: fingerprint, State: operationQueued,
		Stage: "queued", Status: "Matrix Switch queued.", StartedAt: now, UpdatedAt: now,
		Rollback: operationRollback{State: "not_started"},
	}
	if err = s.persist(j); err != nil {
		return operationJournal{}, err
	}
	s.operations[child] = j
	s.appendOperationJournalDiagnosticBestEffort(j)
	return j, nil
}

func (s *operationStore) authorizeQueuedMatrixSwitch(j operationJournal) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.switchMatrixParent(j.MatrixParent, "switch", false)
	actual := s.operations[j.OperationID]
	return err == nil && slices.Contains(p.MatrixFixtures, j.InstanceID) &&
		actual.MatrixParent == j.MatrixParent && actual.OperationType == operationTypeMinecraftSwitch &&
		actual.InstanceID == j.InstanceID && actual.PlanFingerprint == j.PlanFingerprint && actual.State == operationQueued && actual.Stage == "queued"
}

// Called by the matrix orchestrator after classification. The normal plan,
// Switch worker, startup handoff and recovery remain authoritative.
func (s *server) switchMatrixApply(ctx context.Context, parent, id, target, version, action string, reviewed ...switchMatrixCase) (operationJournal, switchMatrixReleaseOutcome, error) {
	o := s.operations
	if !o.switchMatrixMu.TryLock() {
		return operationJournal{}, "", errInstanceBusy
	}
	defer o.switchMatrixMu.Unlock()
	if action != "clone" && action != "convert" {
		return operationJournal{}, "", errInstanceRecovery
	}
	o.mu.Lock()
	p, err := o.switchMatrixParent(parent, "switch", false)
	o.mu.Unlock()
	if err != nil || !slices.Contains(p.MatrixFixtures, id) {
		return operationJournal{}, "", errInstanceID
	}
	if err = o.checkSwitchMatrixFixture(ctx, parent, id); err != nil {
		return operationJournal{}, "", err
	}
	release := resolveSwitchMatrixRelease(ctx, target, version)
	if p.PlanFingerprint != switchMatrixRunnerFingerprint() {
		_ = o.appendOperationDiagnostic(parent, operationTypeSwitchMatrix, "release_lookup", "Exact release lookup completed",
			map[string]string{"software": target, "version": version, "outcome": string(release.Outcome)})
	}
	if release.Outcome != switchMatrixReleaseAvailable {
		return operationJournal{}, release.Outcome, nil
	}
	if len(reviewed) == 1 && !sameSwitchMatrixRuntime(release.Runtime, reviewed[0].TargetRelease.Runtime) {
		return operationJournal{}, switchMatrixReleaseUnverified, errSwitchMatrixReleaseChanged
	}
	plan, err := planMinecraftSwitch(ctx, id, target)
	if err != nil {
		return operationJournal{}, release.Outcome, err
	}
	if (action == "clone" && !plan.Clone || action == "convert" && !plan.Convert) && plan.MatrixPolicyBlocked {
		return operationJournal{}, release.Outcome, errSwitchMatrixPolicy
	}
	if plan.Version != release.Runtime.Game || plan.Runtime.LoaderVersion != release.Runtime.Loader ||
		plan.Runtime.ImageTag != release.Runtime.Image || plan.Runtime.PufferfishBuild != release.Runtime.PufferfishBuild ||
		(action == "clone" && !plan.Clone) || (action == "convert" && !plan.Convert) {
		return operationJournal{}, release.Outcome, errInstanceRecovery
	}
	if len(reviewed) == 1 && (plan.RuntimeDigest != reviewed[0].TargetRelease.Runtime.ImageDigest ||
		plan.Current != reviewed[0].Source || plan.Runtime.CrossplayEnabled != reviewed[0].TargetCrossplay) {
		return operationJournal{}, switchMatrixReleaseUnverified, errSwitchMatrixReleaseChanged
	}
	if len(reviewed) == 1 {
		plan.MatrixLeafBuild = reviewed[0].TargetRelease.Runtime.Build
	}
	req := minecraftSwitchRequest{Target: target, Action: action, Fingerprint: plan.Fingerprint, Confirm: true}
	j, err := o.beginSwitchMatrixChild(parent, id, switchActionFingerprint(req))
	if err != nil {
		return operationJournal{}, release.Outcome, err
	}
	stage := s.runMinecraftSwitch(ctx, j, plan, req)
	o.mu.Lock()
	result := o.operations[j.OperationID]
	o.mu.Unlock()
	if stage != "" {
		return result, release.Outcome, &switchMatrixChildFailure{Stage: stage}
	}
	return result, release.Outcome, nil
}

func (s *server) recoverSwitchMatrixChild(ctx context.Context, parent, child string) error {
	o := s.operations
	if !o.switchMatrixMu.TryLock() {
		return errInstanceBusy
	}
	defer o.switchMatrixMu.Unlock()
	o.mu.Lock()
	p, err := o.switchMatrixParent(parent, "switch", true)
	j := o.operations[child]
	o.mu.Unlock()
	if err != nil || j.MatrixParent != parent || j.State != operationNeedsAttention || !slices.Contains(p.MatrixFixtures, j.InstanceID) {
		return errInstanceRecovery
	}
	lock, err := acquirePluginMaintenanceLock()
	if err != nil {
		return err
	}
	defer lock.Close()
	e, err := readSwitchEvidence(j)
	if errors.Is(err, os.ErrNotExist) {
		err = instanceRuntimes.verifyCommittedSwitchState(ctx, j.InstanceID)
	} else if err == nil {
		if e.CloneID != "" && !slices.Contains(p.MatrixFixtures, e.CloneID) {
			return errInstanceRecovery
		}
		if e.Stage == "complete" {
			// Production Switch had already saved successful verification.
			// Finish binding its test-only artifacts, then retire them through
			// the matrix contract. The interrupted child is still resolved.
			err = instanceRuntimes.verifyCommittedSwitchState(ctx, e.Source, e.Version)
			if err == nil && e.CloneID != "" {
				err = instanceRuntimes.verifyCommittedSwitchState(ctx, e.CloneID, e.Version)
			}
			if err == nil {
				err = o.finalizeSwitchMatrixChild(ctx, parent, e)
			}
		} else {
			ctx = context.WithValue(ctx, switchMatrixChildKey{}, j)
			err = s.recoverMinecraftSwitchLocked(ctx, &e, lock)
			if err == nil {
				err = o.refreshSwitchMatrixFixture(ctx, parent, e, true)
			}
		}
	}
	e.OperationID = j.OperationID
	o.recordSwitchRecovery(e, err)
	if err != nil {
		return err
	}
	_, err = o.transition(child, operationResolved, "resolved", "Matrix Switch recovery verified.")
	return err
}

func (s *operationStore) finalizeSwitchMatrixChild(ctx context.Context, parent string, e minecraftSwitchEvidence) error {
	if e.Stage != "complete" {
		return errInstanceRecovery
	}
	if e.CloneID != "" {
		if f, err := readSwitchMatrixFixture(parent, e.CloneID); errors.Is(err, os.ErrNotExist) ||
			err == nil && (f.Stage == "reserved" || f.Stage == "binding") {
			return s.bindSwitchMatrixFixture(ctx, parent, e.CloneID)
		} else if err != nil {
			return err
		}
		return s.checkSwitchMatrixFixture(ctx, parent, e.CloneID)
	}
	return s.refreshSwitchMatrixFixture(ctx, parent, e, false)
}

const (
	switchMatrixHeap   = "2G"
	switchMatrixMemory = "2560M"
)

var switchMatrixBuildPattern = regexp.MustCompile(`^[1-9][0-9]{0,9}$`)

func switchMatrixRunnerFingerprint() string {
	return "sha256:" + instanceHash([]byte("justvoxel-switch-matrix-runner-v1"))
}

func (b *instanceRuntimeBackend) configureRegisteredMatrixFixture(ctx context.Context, f switchMatrixFixture, snapshot minecraftInstanceRuntimeContext) error {
	if f.Stage != "reserved" || !f.validCandidate() {
		return errInstanceRecovery
	}
	names, err := b.identities().readNames()
	if err != nil || names[f.ID].DisplayName != f.Name {
		return errInstanceRecovery
	}
	return b.configureRegisteredCandidate(ctx, f.ID, snapshot, func() (instanceRuntimeRecord, error) {
		return *f.Candidate, nil
	})
}
