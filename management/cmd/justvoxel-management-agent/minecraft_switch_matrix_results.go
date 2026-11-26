package main

import (
	"errors"
	"os"
	"path/filepath"
	"time"
)

const (
	matrixPending     = "NOT_RUN"
	matrixExecuting   = "EXECUTING"
	matrixPassed      = "PASSED"
	matrixFailed      = "FAILED"
	matrixUnsupported = "UNSUPPORTED"
	matrixBlocked     = "BLOCKED"
	matrixInterrupted = "INTERRUPTED"
	matrixCancelled   = "CANCELLED"
	matrixMaxCases    = 1536
)

type switchMatrixCaseResult struct {
	CaseID         string                     `json:"case_id"`
	Outcome        string                     `json:"outcome"`
	Stage          string                     `json:"failure_stage,omitempty"`
	Reason         string                     `json:"reason,omitempty"`
	ReleaseOutcome switchMatrixReleaseOutcome `json:"observed_release_outcome,omitempty"`
	ChildOperation string                     `json:"switch_operation,omitempty"`
	Runtime        string                     `json:"runtime_verification,omitempty"`
	World          string                     `json:"world_marker_verification,omitempty"`
	Original       string                     `json:"original_source_verification,omitempty"`
	FinalState     string                     `json:"final_state_verification,omitempty"`
	Recovery       string                     `json:"recovery,omitempty"`
	Cleanup        string                     `json:"cleanup,omitempty"`
	StartedAt      time.Time                  `json:"started_at,omitzero"`
	DurationMS     int64                      `json:"duration_ms"`
}

type switchMatrixRun struct {
	Schema          int                      `json:"schema"`
	ID              string                   `json:"operation_id"`
	State           string                   `json:"state"`
	CancelRequested bool                     `json:"cancel_requested"`
	Catalog         map[string][]string      `json:"candidate_versions,omitempty"`
	Cases           []switchMatrixCase       `json:"cases"`
	Results         []switchMatrixCaseResult `json:"results"`
	Progress        switchMatrixProgress     `json:"progress"`
}

type switchMatrixProgress struct {
	ManifestReady      bool              `json:"-"`
	ID                 string            `json:"operation_id"`
	State              string            `json:"state"`
	Current            *switchMatrixCase `json:"current_case,omitempty"`
	Completed          int               `json:"completed"`
	Total              int               `json:"total"`
	Passed             int               `json:"passed"`
	Failed             int               `json:"failed"`
	Unsupported        int               `json:"unsupported"`
	Blocked            int               `json:"blocked"`
	Unfinished         int               `json:"unfinished"`
	CancelRequested    bool              `json:"cancel_requested"`
	DiagnosticCategory string            `json:"diagnostic_category"`
	DiagnosticID       string            `json:"diagnostic_id"`
}

func (r switchMatrixRun) progress() switchMatrixProgress {
	p := switchMatrixProgress{ID: r.ID, State: r.State, Total: len(r.Cases),
		ManifestReady:   r.Cases != nil || r.Catalog != nil || r.State == "running" || r.State == "completed",
		CancelRequested: r.CancelRequested, DiagnosticCategory: "migration", DiagnosticID: r.ID}
	for i, result := range r.Results {
		switch result.Outcome {
		case matrixPassed:
			p.Passed++
		case matrixFailed:
			p.Failed++
		case matrixUnsupported:
			p.Unsupported++
		case matrixBlocked:
			p.Blocked++
		case matrixExecuting:
			c := r.Cases[i]
			p.Current = &c
		}
	}
	p.Completed = p.Passed + p.Failed + p.Unsupported + p.Blocked
	p.Unfinished = p.Total - p.Completed
	return p
}

func (r switchMatrixCaseResult) verifiedPass(c switchMatrixCase) bool {
	return r.Runtime == "verified" && r.World == "verified_overworld_nether_end" &&
		r.FinalState == "verified" && r.Cleanup == "verified" &&
		(c.Action == "clone" && r.Original == "verified" || c.Action == "convert" && r.Original == "not_applicable")
}

func (s *operationStore) readSwitchMatrixRun(id string) (switchMatrixRun, error) {
	r := switchMatrixRun{}
	j, err := s.get(id)
	if err != nil {
		return r, err
	}
	if j.OperationType != operationTypeSwitchMatrix {
		return r, errOperationNotFound
	}
	raw, err := s.readDiagnosticLog("migration", id)
	if err != nil {
		return r, err
	}
	if instanceDecode(raw, &r) != nil || r.Schema != 1 || r.ID != id ||
		len(r.Cases) != len(r.Results) || len(r.Cases) > matrixMaxCases {
		return switchMatrixRun{}, errInstanceRecovery
	}
	return r, nil
}

// The matrix document owns this Migration Logs slot. A compact, atomically
// replaced snapshot keeps all per-case results inside the normal download limit.
func (s *operationStore) saveSwitchMatrixRun(r switchMatrixRun) error {
	if !validOperationID(r.ID) || len(r.Cases) != len(r.Results) || len(r.Cases) > matrixMaxCases {
		return errInstanceRecovery
	}
	// Bound each row at admission. Even a maximally populated run fits inside
	// its diagnostic slot with room for the catalog and progress summary.
	for i, c := range r.Cases {
		r.Results[i].CaseID = c.ID
		if len(mustSwitchJSON(c)) > 1024 || len(mustSwitchJSON(r.Results[i])) > 1024 {
			return errInstanceRecovery
		}
	}
	r.Progress = r.progress()
	data := append(mustSwitchJSON(r), '\n')
	if len(data) <= 1 || len(data) > operationDiagnosticLimit {
		return errInstanceRecovery
	}
	s.diagnosticMu.Lock()
	defer s.diagnosticMu.Unlock()
	dir := s.diagnosticCategoryDirectory("migration")
	if err := validatePrivateDiagnosticDirectory(dir); err != nil {
		return err
	}
	if err := atomicWrite(filepath.Join(dir, r.ID+".log"), data, 0o600); err != nil {
		return err
	}
	fd, err := os.Open(dir)
	if err != nil {
		return err
	}
	return errors.Join(fd.Sync(), fd.Close())
}

func (s *operationStore) updateSwitchMatrixRun(id string, update func(*switchMatrixRun) error) error {
	s.switchMatrixRunMu.Lock()
	defer s.switchMatrixRunMu.Unlock()
	r, err := s.readSwitchMatrixRun(id)
	if err != nil {
		return err
	}
	if err = update(&r); err != nil {
		return err
	}
	return s.saveSwitchMatrixRun(r)
}

func (s *operationStore) switchMatrixProgress(id string) (switchMatrixProgress, error) {
	r, err := s.readSwitchMatrixRun(id)
	if err != nil {
		return switchMatrixProgress{}, err
	}
	j, err := s.get(id)
	if err != nil {
		return switchMatrixProgress{}, err
	}
	if j.State == operationNeedsAttention && r.State != "interrupted" {
		r.State = "needs_attention"
	}
	return r.progress(), nil
}

func (s *operationStore) stopSwitchMatrixRun(id string) error {
	return s.updateSwitchMatrixRun(id, func(r *switchMatrixRun) error {
		if r.CancelRequested {
			return nil
		}
		if r.State != "preparing" && r.State != "running" {
			return errInstanceBusy
		}
		j, err := s.get(id)
		if err != nil || j.State == operationNeedsAttention || !operationIsCurrent(j.State) {
			return errInstanceBusy
		}
		r.CancelRequested = true
		return nil
	})
}

func (r *switchMatrixRun) interrupt(now time.Time) {
	r.State = "interrupted"
	for i := range r.Results {
		x := &r.Results[i]
		if x.Outcome == matrixExecuting {
			x.Outcome, x.Reason = matrixInterrupted, "agent_interrupted"
			x.Stage = "interrupted"
			x.DurationMS = max(0, now.Sub(x.StartedAt).Milliseconds())
		}
	}
}

func (s *operationStore) reconcileSwitchMatrixRun(j operationJournal) error {
	if j.OperationType != operationTypeSwitchMatrix || j.State != operationNeedsAttention ||
		j.PlanFingerprint != switchMatrixRunnerFingerprint() {
		return nil
	}
	err := s.updateSwitchMatrixRun(j.OperationID, func(r *switchMatrixRun) error {
		if j.Stage == "interrupted" || r.progress().Current != nil {
			r.interrupt(s.now())
		}
		return nil
	})
	// Stage 2 journals and a crash before the first write can have no document.
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
