package main

import (
	"errors"
	"net/http"
	"net/url"
	"time"
)

const switchMatrixAPIPath = "/v1/minecraft/switch-matrix"
const switchMatrixProductionDiagnostic = "Switch matrix cleanup is unresolved. Maintenance remains reserved. Return to a JustVoxel development image to recover the operation from Server Migration → Switch."

type switchMatrixSummary struct {
	Targeted          bool                     `json:"targeted,omitempty"`
	OperationID       string                   `json:"operation_id"`
	State             string                   `json:"state"`
	ReservationActive bool                     `json:"reservation_active"`
	RecoveryRequired  bool                     `json:"recovery_required"`
	Progress          *switchMatrixAPICounters `json:"progress,omitempty"`
	DiagnosticID      string                   `json:"diagnostic_id"`
}

type switchMatrixAPICounters struct {
	ManifestReady   bool   `json:"manifest_ready"`
	Completed       int    `json:"completed"`
	Total           int    `json:"total"`
	Passed          int    `json:"passed"`
	Failed          int    `json:"failed"`
	Unsupported     int    `json:"unsupported"`
	Blocked         int    `json:"blocked"`
	Unfinished      int    `json:"unfinished"`
	CancelRequested bool   `json:"cancel_requested"`
	Source          string `json:"source,omitempty"`
	Target          string `json:"target,omitempty"`
	Action          string `json:"action,omitempty"`
}

type switchMatrixAPIResponse struct {
	Enabled   bool                 `json:"enabled"`
	Operation *switchMatrixSummary `json:"operation"`
}

func registerSwitchMatrixRoutes(mux *http.ServeMux, s *server) {
	mux.HandleFunc("GET "+switchMatrixAPIPath, s.switchMatrixStatus)
	registerSwitchMatrixExecutionRoutes(mux, s)
}

func (s *server) switchMatrixStatus(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdministrator(w, r); !ok {
		return
	}
	id := ""
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid Switch matrix status request")
		return
	}
	if values, exists := query["operation_id"]; exists && len(query) == 1 && len(values) == 1 && validOperationID(values[0]) {
		id = values[0]
	} else if len(query) != 0 || r.URL.RawQuery != "" {
		writeError(w, http.StatusBadRequest, "invalid Switch matrix status request")
		return
	}
	s.writeSwitchMatrixStatus(w, http.StatusOK, id)
}

func (s *server) writeSwitchMatrixStatus(w http.ResponseWriter, code int, id string) {
	w.Header().Set("Cache-Control", "no-store")
	if s.operations == nil {
		writeError(w, http.StatusServiceUnavailable, "Switch matrix status is unavailable")
		return
	}
	summary, err := s.operations.switchMatrixSummary(id)
	if errors.Is(err, errOperationNotFound) {
		writeError(w, http.StatusNotFound, "Switch matrix operation not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "Switch matrix status is unavailable")
		return
	}
	writeJSON(w, code, switchMatrixAPIResponse{Enabled: switchMatrixDevelopment, Operation: summary})
}

func (o *operationStore) switchMatrixSummary(id string) (*switchMatrixSummary, error) {
	// Read the report under the same lock used to publish updates. The journal
	// is authoritative for the reservation even when the report is unavailable.
	o.switchMatrixRunMu.Lock()
	defer o.switchMatrixRunMu.Unlock()
	j := o.switchMatrixStatusJournal(id)
	if j.OperationID == "" {
		if id != "" {
			return nil, errOperationNotFound
		}
		return nil, nil
	}
	if !switchMatrixDevelopment && !operationIsCurrent(j.State) {
		return nil, nil
	}
	progress, err := o.switchMatrixProgress(j.OperationID)
	j, journalErr := o.get(j.OperationID)
	if journalErr != nil {
		return nil, journalErr
	}
	active := operationIsCurrent(j.State)
	summary := switchMatrixSummary{
		OperationID: j.OperationID, DiagnosticID: j.OperationID, State: "status_unavailable",
		ReservationActive: active, RecoveryRequired: j.State == operationNeedsAttention,
	}
	if err == nil {
		summary.Targeted = progress.Targeted
		switch progress.State {
		case "preparing", "running", "completed", "cancelled", "halted", "interrupted", "needs_attention", "recovered":
			summary.State = progress.State
		}
		summary.Progress = &switchMatrixAPICounters{
			ManifestReady: progress.ManifestReady, Completed: progress.Completed, Total: progress.Total, Passed: progress.Passed,
			Failed: progress.Failed, Unsupported: progress.Unsupported, Blocked: progress.Blocked,
			Unfinished: progress.Unfinished, CancelRequested: progress.CancelRequested,
		}
		if c := progress.Current; c != nil && (c.Action == "clone" || c.Action == "convert") {
			_, source := javaServerSoftware(c.Source)
			_, target := javaServerSoftware(c.Target)
			if source && target {
				summary.Progress.Source, summary.Progress.Target, summary.Progress.Action = c.Source, c.Target, c.Action
			}
		}
	}
	switch {
	case summary.RecoveryRequired && summary.State != "interrupted":
		summary.State = "needs_attention"
	case active && j.State == operationQueued:
		summary.State = "preparing"
	case active && j.State == operationVerifying && err == nil:
		summary.State = "closing"
	}
	return &summary, nil
}

func (s *operationStore) switchMatrixStatusJournal(id string) operationJournal {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id != "" {
		if j := s.operations[id]; j.OperationType == operationTypeSwitchMatrix {
			return j
		}
		return operationJournal{}
	}
	if j := s.operations[s.currentMinecraftResetID]; j.OperationType == operationTypeSwitchMatrix && operationIsCurrent(j.State) {
		return j
	}
	if !switchMatrixDevelopment {
		return operationJournal{}
	}
	var latest operationJournal
	var started time.Time
	for _, j := range s.operations {
		if j.OperationType != operationTypeSwitchMatrix {
			continue
		}
		at, err := time.Parse(time.RFC3339Nano, j.StartedAt)
		if err == nil && (latest.OperationID == "" || at.After(started) || (at.Equal(started) && j.OperationID > latest.OperationID)) {
			latest, started = j, at
		}
	}
	return latest
}
