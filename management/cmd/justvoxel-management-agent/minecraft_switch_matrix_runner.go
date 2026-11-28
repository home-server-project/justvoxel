//go:build justvoxel_dev

package main

import (
	"context"
	"errors"
	"os"
	"slices"
	"time"
)

// The Management API authorizes this appliance-wide operation. No request
// context, server selection, runtime options, or resource IDs enter the worker.
func (s *server) startSwitchMatrixRun(selected ...string) (operationJournal, error) {
	targeted := len(selected) != 0
	if targeted {
		if err := validateSwitchMatrixSelection(selected); err != nil {
			return operationJournal{}, err
		}
		selected = slices.Clone(selected)
	}
	o := s.operations
	if o == nil || !o.switchMatrixWorkerMu.TryLock() {
		return operationJournal{}, errInstanceBusy
	}
	j, err := o.beginSwitchMatrix(switchMatrixRunnerFingerprint())
	if err != nil {
		o.switchMatrixWorkerMu.Unlock()
		return j, err
	}
	if err = o.saveSwitchMatrixRun(switchMatrixRun{Schema: 1, ID: j.OperationID, State: "preparing", Targeted: targeted}); err != nil {
		o.holdSwitchMatrixRun(j.OperationID)
		o.switchMatrixWorkerMu.Unlock()
		return j, err
	}
	go func() {
		defer o.switchMatrixWorkerMu.Unlock()
		if err := s.runSwitchMatrix(j, selected...); err != nil {
			o.holdSwitchMatrixRun(j.OperationID)
		}
	}()
	return j, nil
}

func (s *server) runSwitchMatrix(j operationJournal, selected ...string) error {
	o := s.operations
	id := j.OperationID
	if _, err := o.transition(id, operationValidating, "manifest", "Discovering exact Switch matrix releases."); err != nil {
		return err
	}
	discovery, cancel := context.WithTimeout(context.Background(), 45*time.Minute)
	defer cancel()
	catalog, catalogIssue := discoverSwitchMatrixVersions(discovery)
	cases := buildSwitchMatrixCases(discovery, catalog, resolveSwitchMatrixRelease, verifySwitchMatrixCrossplaySelection, selected...)
	if len(selected) > 0 {
		cases = orderTargetedSwitchMatrixCases(selected, cases)
	} else {
		cases = addSwitchMatrixCoverage(cases)
	}
	bindSwitchMatrixImages(discovery, cases)
	cancel()

	if err := o.updateSwitchMatrixRun(id, func(r *switchMatrixRun) error {
		r.Catalog, r.Cases, r.State = catalog, cases, "running"
		r.CatalogIssue = catalogIssue
		r.Results = newSwitchMatrixResults(cases)
		return nil
	}); err != nil {
		return err
	}
	if _, err := o.transition(id, operationRunning, "fixtures", "Switch matrix manifest recorded."); err != nil {
		return err
	}
	return o.runSwitchMatrixCases(id, func(c switchMatrixCase) (switchMatrixCaseResult, error) {
		return s.executeSwitchMatrixCase(id, c)
	}, func() (operationJournal, error) {
		return s.recoverSwitchMatrixClosure(id, true)
	})
}

type switchMatrixCaseWorker func(switchMatrixCase) (switchMatrixCaseResult, error)

var errSwitchMatrixStopAdmission = errors.New("Switch matrix cancellation requested")

func (o *operationStore) runSwitchMatrixCases(id string, execute switchMatrixCaseWorker, recoverClosure ...func() (operationJournal, error)) error {
	manifest, err := o.readSwitchMatrixRun(id)
	if err != nil {
		return err
	}
	smoke, haltReason := -1, ""
	var order []int
	if manifest.Targeted {
		for i, result := range manifest.Results {
			if result.Outcome != matrixPending {
				haltReason = "targeted_selection_unavailable"
				order = nil
				break
			}
			order = append(order, i)
		}
	} else {
		smoke, haltReason = selectSwitchMatrixSmoke(manifest)
		if smoke >= 0 {
			if err := o.updateSwitchMatrixRun(id, func(r *switchMatrixRun) error {
				r.SmokeCase = r.Cases[smoke].ID
				return nil
			}); err != nil {
				return err
			}
			order = append(order, smoke)
			for i, result := range manifest.Results {
				if manifest.Cases[i].Profile != "" && result.Outcome == matrixPending {
					order = append(order, i)
				}
			}
			for i, result := range manifest.Results {
				if i != smoke && manifest.Cases[i].Profile == "" && result.Outcome == matrixPending {
					order = append(order, i)
				}
			}
		}
	}
	for _, i := range order {
		c := manifest.Cases[i]
		// This durable update serializes admission with Stop Test. Once
		// admitted, the case owns its transaction through verified closure.
		started := o.now()
		err = o.updateSwitchMatrixRun(id, func(r *switchMatrixRun) error {
			if r.CancelRequested {
				return errSwitchMatrixStopAdmission
			}
			r.Results[i] = switchMatrixCaseResult{Outcome: matrixExecuting, StartedAt: started}
			return nil
		})
		if errors.Is(err, errSwitchMatrixStopAdmission) {
			break
		}
		if err != nil {
			return err
		}
		result, unresolved := execute(c)
		if result.Outcome == matrixPassed && (unresolved != nil || !result.verifiedPass(c)) {
			result.Outcome, result.Stage, result.Reason = matrixFailed, "verification", "case_verification_incomplete"
		}
		if unresolved == nil && result.Cleanup != "verified" {
			unresolved = errInstanceRecovery
			result.Outcome, result.Cleanup = matrixFailed, "unresolved"
			if result.Stage == "" {
				result.Stage, result.Reason = "cleanup", "fixture_closure_unverified"
			}
		}
		switch {
		case i == smoke && result.Outcome != matrixPassed:
			haltReason = "smoke_case_failed"
		case switchMatrixCommonFixtureFailure(result):
			haltReason = "common_fixture_infrastructure_failed"
		}
		_, retryable := unresolved.(*switchMatrixRetryableClosure)
		autoRecover := len(recoverClosure) == 1 && retryable && !switchMatrixCommonFixtureFailure(result)
		result.StartedAt, result.DurationMS = started, max(0, o.now().Sub(started).Milliseconds())
		err = o.updateSwitchMatrixRun(id, func(r *switchMatrixRun) error {
			r.Results[i] = result
			r.HaltReason = haltReason
			if unresolved != nil && !autoRecover {
				r.State = "needs_attention"
			}
			return nil
		})
		if err != nil {
			return errors.Join(err, unresolved)
		}
		if autoRecover {
			// The first failure is already durable. Retry closure once under
			// the same worker and maintenance reservation.
			child, recoveryErr := recoverClosure[0]()
			result.recordClosureRecovery(child, recoveryErr)
			result.DurationMS = max(0, o.now().Sub(started).Milliseconds())
			if err = o.updateSwitchMatrixRun(id, func(r *switchMatrixRun) error {
				r.Results[i] = result
				if recoveryErr != nil {
					r.State = "needs_attention"
				}
				return nil
			}); err != nil {
				return errors.Join(err, recoveryErr)
			}
			unresolved = recoveryErr
		}
		if unresolved != nil {
			return unresolved
		}
		if haltReason != "" {
			break
		}
	}
	if err := o.checkSwitchMatrixRunClosure(id); err != nil {
		return err
	}
	if _, err = o.transition(id, operationVerifying, "closure", "All Switch matrix resources were retired."); err != nil {
		return err
	}
	halted := false
	if err = o.updateSwitchMatrixRun(id, func(r *switchMatrixRun) error {
		r.State = "completed"
		if haltReason != "" && (!r.CancelRequested || r.HaltReason != "") {
			r.State, r.HaltReason = "halted", haltReason
			halted = true
		} else if r.CancelRequested {
			for i := range r.Results {
				if r.Results[i].Outcome == matrixPending {
					r.Results[i] = switchMatrixCaseResult{Outcome: matrixCancelled, Reason: "stop_requested"}
					r.State = "cancelled"
				}
			}
		}
		return nil
	}); err != nil {
		return err
	}
	stage, status := "complete", "Switch matrix closed. See the Migration log for the complete result."
	if halted {
		stage, status = "halted", "Switch matrix stopped early. All test resources were retired. See the Migration log for the reason."
	}
	if manifest.Targeted {
		status = "Targeted Switch test closed. See the Migration log for the selected case results."
		if halted {
			status = "Targeted Switch test stopped early. All test resources were retired. See the Migration log for the reason."
		}
	}
	_, err = o.transition(id, operationSucceeded, stage, status)
	return err
}

func selectSwitchMatrixSmoke(r switchMatrixRun) (int, string) {
	if r.CatalogIssue != "" {
		return -1, r.CatalogIssue
	}
	reason := "smoke_case_missing"
	for i, c := range r.Cases {
		if c.Source != "vanilla" || c.Target != "purpur" || c.Action != "clone" ||
			c.Family != switchMatrixModern || c.Crossplay || c.TargetCrossplay || c.Profile != "" {
			continue
		}
		switch {
		case c.ReleaseReason == "publisher_catalog_unverified":
			reason = "smoke_modern_catalog_empty"
		case c.ReleaseReason == "exact_release_unsupported":
			reason = "smoke_exact_release_unsupported"
		case c.ReleaseReason == "exact_release_unavailable":
			reason = "smoke_exact_release_unavailable"
		case c.ReleaseReason == "exact_release_unverified":
			reason = "smoke_exact_release_unverified"
		case c.Reason == "runtime_image_unverified":
			reason = "smoke_runtime_image_unverified"
		case c.Disposition != "" || r.Results[i].Outcome != matrixPending:
			reason = "smoke_case_unavailable"
		case !java25GameReleasePattern.MatchString(c.Version) ||
			!switchMatrixSmokeReleaseMatches(c.SourceRelease, c.Version) ||
			!switchMatrixSmokeReleaseMatches(c.TargetRelease, c.Version):
			reason = "smoke_exact_release_unverified"
		case !operationFingerprintPattern.MatchString(c.SourceRelease.Runtime.ImageDigest) ||
			!operationFingerprintPattern.MatchString(c.TargetRelease.Runtime.ImageDigest):
			reason = "smoke_runtime_image_unverified"
		default:
			return i, ""
		}
	}
	return -1, reason
}

func switchMatrixSmokeReleaseMatches(release switchMatrixRelease, version string) bool {
	return release.Outcome == switchMatrixReleaseAvailable && release.Runtime != nil && release.Runtime.Game == version
}

func switchMatrixCommonFixtureFailure(result switchMatrixCaseResult) bool {
	if result.Outcome != matrixFailed {
		return false
	}
	switch result.Stage {
	case "ownership", "source_ownership", "target_ownership", "original_ownership",
		"switch_ownership", "switch_fixture_binding", "switch_journal_finalization":
		return true
	}
	if result.Reason == "switch_journal_persistence_failed" {
		return true
	}
	if result.Stage == "resource_admission" && result.Reason == "fixture_capacity_unverified" {
		return true
	}
	switch switchMatrixFixtureStage(result.Stage) {
	case matrixFixtureConfiguration, matrixFixtureCredential, matrixFixtureStorage,
		matrixFixtureOwnership, matrixFixturePublication, matrixFixtureBinding:
		return true
	}
	return false
}

func (o *operationStore) holdSwitchMatrixRun(id string) {
	_, _ = o.transition(id, operationNeedsAttention, "matrix_recovery", "Switch matrix requires verified recovery. Its reservation is retained.")
}

// Explicit recovery never calls the case runner or the publisher discovery.
// The operation ID is the only input. IDs of fixtures and children come from
// the parent and child evidence under the existing reservation.
func (s *server) recoverSwitchMatrixRun(id string) error {
	o := s.operations
	if o == nil || !o.switchMatrixWorkerMu.TryLock() {
		return errInstanceBusy
	}
	defer o.switchMatrixWorkerMu.Unlock()
	o.mu.Lock()
	j, err := o.switchMatrixParent(id, "retirement", true)
	o.mu.Unlock()
	if err != nil || j.State != operationNeedsAttention || j.PlanFingerprint != switchMatrixRunnerFingerprint() {
		return errInstanceRecovery
	}
	r, err := o.readSwitchMatrixRun(id)
	if errors.Is(err, os.ErrNotExist) && len(j.MatrixFixtures) == 0 {
		err = o.saveSwitchMatrixRun(switchMatrixRun{Schema: 1, ID: id, State: "interrupted"})
	}
	if err != nil {
		return err
	}
	child, cleanupErr := s.recoverSwitchMatrixClosure(id, false)
	if err = o.updateSwitchMatrixRun(id, func(run *switchMatrixRun) error {
		for i := len(run.Results) - 1; i >= 0; i-- {
			result := &run.Results[i]
			if result.Outcome != matrixExecuting && result.Outcome != matrixInterrupted &&
				result.Cleanup != "unresolved" {
				continue
			}
			if result.Outcome == matrixExecuting {
				result.Outcome, result.Reason = matrixInterrupted, "worker_interrupted"
				result.DurationMS = max(0, o.now().Sub(result.StartedAt).Milliseconds())
			}
			result.recordClosureRecovery(child, cleanupErr)
			break
		}
		if cleanupErr == nil {
			run.State = "recovered"
		} else if r.State != "interrupted" {
			run.State = "needs_attention"
		}
		return nil
	}); err != nil || cleanupErr != nil {
		return errors.Join(err, cleanupErr)
	}
	_, err = o.transition(id, operationResolved, "resolved", "Switch matrix resources were recovered and retired. No cases were rerun.")
	return err
}

func (s *server) recoverSwitchMatrixClosure(id string, continuing bool) (operationJournal, error) {
	cleanup, cancel := context.WithTimeout(context.Background(), 35*time.Minute)
	defer cancel()
	child, err := s.closeSwitchMatrixCase(cleanup, id, 0)
	if err != nil || !continuing {
		return child, err
	}
	lock, err := acquirePluginMaintenanceLock()
	if err != nil {
		return child, err
	}
	err = s.operations.checkSwitchMatrixRunClosure(id)
	if err == nil {
		err = instanceRuntimes.requireNoInstanceRestore()
	}
	return child, errors.Join(err, lock.Close())
}

func (r *switchMatrixCaseResult) recordClosureRecovery(child operationJournal, err error) {
	if child.OperationID != "" {
		r.ChildOperation, r.Recovery = child.OperationID, "unresolved"
		if child.State == operationResolved {
			r.Recovery = "verified"
		}
	}
	r.Cleanup, r.ClosureRecovery, r.RecoveryReason = "verified", "verified", ""
	if err != nil {
		r.Cleanup, r.ClosureRecovery = "unresolved", "unresolved"
		r.RecoveryReason = switchMatrixClosureDiagnostic(child, err)
	}
}

func (o *operationStore) switchMatrixCurrentChild(parent string) operationJournal {
	o.mu.Lock()
	defer o.mu.Unlock()
	for _, j := range o.operations {
		if j.MatrixParent == parent && operationIsCurrent(j.State) {
			return j
		}
	}
	return operationJournal{}
}

func (o *operationStore) checkSwitchMatrixRunClosure(id string) error {
	return o.checkSwitchMatrixClosureFrom(id, 0)
}

func (o *operationStore) checkSwitchMatrixClosureFrom(id string, offset int) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	j := o.operations[id]
	if o.currentMinecraftResetID != id || j.OperationType != operationTypeSwitchMatrix ||
		o.switchMatrixExclusion == nil || offset < 0 || offset > len(j.MatrixFixtures) {
		return errInstanceRecovery
	}
	j.MatrixFixtures = j.MatrixFixtures[offset:]
	return o.verifySwitchMatrixClosure(j)
}

func (s *server) closeSwitchMatrixCase(ctx context.Context, id string, offset int) (operationJournal, error) {
	o := s.operations
	child := o.switchMatrixCurrentChild(id)
	if child.OperationID != "" {
		if child.State != operationNeedsAttention {
			// The synchronous Switch worker has returned, or explicit recovery
			// owns the run. A failed journal write may have left an active phase.
			interrupted, err := o.transition(child.OperationID, operationNeedsAttention,
				"interrupted", "Switch worker exited before recording a terminal state.")
			if err != nil {
				return child, err
			}
			child = interrupted
		}
		if err := s.recoverSwitchMatrixChild(ctx, id, child.OperationID); err != nil {
			return child, err
		}
		var err error
		child, err = o.get(child.OperationID)
		if err != nil {
			return child, err
		}
	}
	j, err := o.get(id)
	if err != nil || offset < 0 || offset > len(j.MatrixFixtures) {
		return child, errInstanceRecovery
	}
	if j.State == operationRunning {
		if _, err = o.transition(id, operationRunning, "retirement", "Retiring verified Switch matrix fixtures."); err != nil {
			return child, err
		}
	}
	fixtures := slices.Clone(j.MatrixFixtures[offset:])
	slices.Reverse(fixtures)
	for _, fixture := range fixtures {
		if err = s.stopSwitchMatrixFixtureForRetirement(ctx, id, fixture); err != nil {
			return child, err
		}
		_, err = readSwitchMatrixFixture(id, fixture)
		if errors.Is(err, os.ErrNotExist) {
			o.mu.Lock()
			recovered := o.switchMatrixRecoveredClone(id, fixture)
			o.mu.Unlock()
			if recovered {
				continue
			}
		}
		if err = o.retireSwitchMatrixFixture(ctx, id, fixture); err != nil {
			return child, err
		}
	}
	return child, o.checkSwitchMatrixClosureFrom(id, offset)
}
