//go:build justvoxel_dev

package main

import (
	"encoding/json"
	"net/http"
	"time"
)

const switchMatrixDevelopment = true

func registerSwitchMatrixExecutionRoutes(mux *http.ServeMux, s *server) {
	mux.HandleFunc("POST "+switchMatrixAPIPath+"/start", s.switchMatrixStart)
	mux.HandleFunc("POST "+switchMatrixAPIPath+"/stop", s.switchMatrixStop)
	mux.HandleFunc("POST "+switchMatrixAPIPath+"/recover", s.switchMatrixRecover)
}

func (s *server) switchMatrixStart(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdministrator(w, r); !ok {
		return
	}
	var request struct {
		CaseIDs json.RawMessage `json:"case_ids"`
	}
	if r.URL.RawQuery != "" || decodeContainerJSON(r.Body, &request) != nil {
		writeError(w, http.StatusBadRequest, "invalid Switch matrix start request")
		return
	}
	if s.operations == nil {
		writeError(w, http.StatusServiceUnavailable, "Switch matrix operations are unavailable")
		return
	}
	var selected []string
	if request.CaseIDs != nil {
		if json.Unmarshal(request.CaseIDs, &selected) != nil {
			writeError(w, http.StatusBadRequest, "Expected a list of canonical Switch matrix case IDs.")
			return
		}
		if err := validateSwitchMatrixSelection(selected); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	j, err := s.startSwitchMatrixRun(selected...)
	if err != nil {
		writeError(w, http.StatusConflict, "Switch matrix could not start. Check current maintenance and Switch matrix status.")
		return
	}
	// Return only the durable operation identity. Status reads the persisted
	// progress independently of this request or the browser's selected server.
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusAccepted, switchMatrixAPIResponse{Enabled: true, Operation: &switchMatrixSummary{
		Targeted: len(selected) != 0, OperationID: j.OperationID, State: "preparing", ReservationActive: true, DiagnosticID: j.OperationID,
	}})
}

func (s *server) switchMatrixActionID(w http.ResponseWriter, r *http.Request) (string, bool) {
	if _, ok := s.requireAdministrator(w, r); !ok {
		return "", false
	}
	var request struct {
		OperationID string `json:"operation_id"`
	}
	if r.URL.RawQuery != "" || decodeContainerJSON(r.Body, &request) != nil || !validOperationID(request.OperationID) {
		writeError(w, http.StatusBadRequest, "invalid Switch matrix operation id")
		return "", false
	}
	if s.operations == nil {
		writeError(w, http.StatusServiceUnavailable, "Switch matrix status is unavailable")
		return "", false
	}
	if s.operations.switchMatrixStatusJournal(request.OperationID).OperationID == "" {
		writeError(w, http.StatusNotFound, "Switch matrix operation not found")
		return "", false
	}
	return request.OperationID, true
}

func (s *server) switchMatrixStop(w http.ResponseWriter, r *http.Request) {
	id, ok := s.switchMatrixActionID(w, r)
	if !ok {
		return
	}
	if err := s.operations.stopSwitchMatrixRun(id); err != nil {
		writeError(w, http.StatusConflict, "Switch matrix cancellation could not be recorded. Check current status.")
		return
	}
	s.writeSwitchMatrixStatus(w, http.StatusAccepted, id)
}

func (s *server) switchMatrixRecover(w http.ResponseWriter, r *http.Request) {
	id, ok := s.switchMatrixActionID(w, r)
	if !ok {
		return
	}
	// The existing recovery worker uses a bounded background context and can
	// outlast the normal response deadline while retiring owned fixtures.
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(36 * time.Minute))
	if err := s.recoverSwitchMatrixRun(id); err != nil {
		writeError(w, http.StatusConflict, "Switch matrix recovery could not be completed. Check the current status and Migration Logs.")
		return
	}
	s.writeSwitchMatrixStatus(w, http.StatusOK, id)
}
