package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/home-server-project/justvoxel-webui/internal/api"
)

type systemWorkspaceLogsAPI interface {
	AdminDiagnosticLogs(context.Context, string) ([]api.DiagnosticLogEntry, error)
	AdminDiagnosticLog(context.Context, string, string, string) ([]byte, error)
}

type systemWorkspaceFailedUnitsAPI interface {
	AdminFailedUnits(context.Context, string) ([]api.FailedUnit, error)
	AdminFailedUnitJournal(context.Context, string, string) ([]byte, error)
}

func (a *App) systemWorkspaceLogs(w http.ResponseWriter, r *http.Request) {
	session, _, ok := a.systemWorkspaceIdentity(w, r, "administrator")
	if !ok {
		return
	}
	client, ok := a.api.(systemWorkspaceLogsAPI)
	if !ok {
		writeSystemWorkspaceError(w, http.StatusServiceUnavailable, ui(w, "ui.a7bab9c87013"))
		return
	}
	logs, err := client.AdminDiagnosticLogs(r.Context(), session)
	if err != nil {
		a.writeSystemWorkspaceAPIError(w, err, ui(w, "ui.a7bab9c87013"))
		return
	}
	units := make([]api.FailedUnit, 0)
	unavailable := ""
	if live, supported := a.api.(systemWorkspaceFailedUnitsAPI); supported {
		units, err = live.AdminFailedUnits(r.Context(), session)
		if errors.Is(err, api.ErrUnauthorized) || errors.Is(err, api.ErrPasswordChangeRequired) {
			a.writeSystemWorkspaceAPIError(w, err, ui(w, "ui.138f547a3261"))
			return
		}
		if err != nil {
			unavailable = ui(w, "ui.138f547a3261")
			units = nil
		}
	} else {
		unavailable = ui(w, "ui.138f547a3261")
	}
	writeSystemWorkspaceJSON(w, http.StatusOK, map[string]any{"ok": true, "logs": logs, "failed_units": units, "failed_units_error": unavailable})
}

func (a *App) systemWorkspaceLog(w http.ResponseWriter, r *http.Request) {
	session, _, ok := a.systemWorkspaceIdentity(w, r, "administrator")
	if !ok {
		return
	}
	category, id := r.PathValue("category"), r.PathValue("id")
	if !validWorkspaceLogID(category, id) {
		writeSystemWorkspaceError(w, http.StatusBadRequest, ui(w, "ui.b4e5c635a06e"))
		return
	}
	client, ok := a.api.(systemWorkspaceLogsAPI)
	if !ok {
		writeSystemWorkspaceError(w, http.StatusServiceUnavailable, ui(w, "ui.a7bab9c87013"))
		return
	}
	data, err := client.AdminDiagnosticLog(r.Context(), session, category, id)
	if err != nil {
		a.writeSystemWorkspaceAPIError(w, err, ui(w, "ui.bde5c83597fc"))
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.URL.Query().Get("download") == "1" {
		w.Header().Set("Content-Disposition", "attachment; filename=justvoxel-"+category+"-"+id+".log")
	}
	_, _ = w.Write(data)
}

func validWorkspaceLogID(category, id string) bool {
	switch category {
	case "setup", "migration", "restore", "plugins", "reset", "network", "update", "support", "operation":
	default:
		return false
	}
	if len(id) != 36 {
		return false
	}
	for index, char := range id {
		if index == 8 || index == 13 || index == 18 || index == 23 {
			if char != '-' {
				return false
			}
			continue
		}
		if char < '0' || char > '9' {
			if char < 'a' || char > 'f' {
				return false
			}
		}
	}
	return id[14] == '4' && (id[19] == '8' || id[19] == '9' || id[19] == 'a' || id[19] == 'b')
}

func (a *App) systemWorkspaceFailedUnitJournal(w http.ResponseWriter, r *http.Request) {
	session, _, ok := a.systemWorkspaceIdentity(w, r, "administrator")
	if !ok {
		return
	}
	unit := r.PathValue("unit")
	if !api.ValidFailedUnitName(unit) {
		writeSystemWorkspaceError(w, http.StatusBadRequest, ui(w, "ui.d6b7ea342232"))
		return
	}
	client, ok := a.api.(systemWorkspaceFailedUnitsAPI)
	if !ok {
		writeSystemWorkspaceError(w, http.StatusServiceUnavailable, ui(w, "ui.c651952bfc8b"))
		return
	}
	data, err := client.AdminFailedUnitJournal(r.Context(), session, unit)
	if err != nil {
		status := http.StatusBadGateway
		if errors.Is(err, api.ErrUnauthorized) {
			status = http.StatusUnauthorized
		}
		if errors.Is(err, api.ErrPasswordChangeRequired) {
			status = http.StatusForbidden
		}
		var responseErr *api.ResponseError
		if errors.As(err, &responseErr) && responseErr.StatusCode >= 400 && responseErr.StatusCode <= 599 {
			status = responseErr.StatusCode
		}
		writeSystemWorkspaceError(w, status, ui(w, "ui.c651952bfc8b"))
		return
	}
	if len(data) > 256*1024 {
		writeSystemWorkspaceError(w, http.StatusBadGateway, ui(w, "ui.c651952bfc8b"))
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.URL.Query().Get("download") == "1" {
		w.Header().Set("Content-Disposition", `attachment; filename="justvoxel-failed-unit.log"`)
	}
	_, _ = w.Write(data)
}

type systemWorkspaceSupportAPI interface {
	AdminGenerateSupportReport(context.Context, string) (api.DiagnosticLogEntry, error)
}

func (a *App) systemWorkspaceSupportReport(w http.ResponseWriter, r *http.Request) {
	a.systemWorkspaceGenerateReport(w, r, "support")
}

func (a *App) systemWorkspaceNetworkReport(w http.ResponseWriter, r *http.Request) {
	a.systemWorkspaceGenerateReport(w, r, "network")
}

type systemWorkspaceNetworkReportAPI interface {
	AdminGenerateNetworkReport(context.Context, string) (api.DiagnosticLogEntry, error)
}

func (a *App) systemWorkspaceGenerateReport(w http.ResponseWriter, r *http.Request, category string) {
	reportName := "Support"
	if category == "network" {
		reportName = ui(w, "ui.1744b96470b5")
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if !a.validCSRF(r) {
		writeSystemWorkspaceError(w, http.StatusForbidden, ui(w, "ui.74bccc44f472"))
		return
	}
	session, _, ok := a.systemWorkspaceIdentity(w, r, "administrator")
	if !ok {
		return
	}
	if r.URL.RawQuery != "" || r.ParseForm() != nil || len(r.PostForm) > 1 || len(r.PostForm["csrf"]) > 1 {
		writeSystemWorkspaceError(w, http.StatusBadRequest, reportName+ui(w, "ui.147fea70fc4f"))
		return
	}
	for key := range r.PostForm {
		if key != "csrf" {
			writeSystemWorkspaceError(w, http.StatusBadRequest, reportName+ui(w, "ui.147fea70fc4f"))
			return
		}
	}
	var entry api.DiagnosticLogEntry
	var err error
	if category == "network" {
		client, supported := a.api.(systemWorkspaceNetworkReportAPI)
		if !supported {
			writeSystemWorkspaceError(w, http.StatusServiceUnavailable, ui(w, "ui.87900632a243"))
			return
		}
		entry, err = client.AdminGenerateNetworkReport(r.Context(), session)
	} else {
		client, supported := a.api.(systemWorkspaceSupportAPI)
		if !supported {
			writeSystemWorkspaceError(w, http.StatusServiceUnavailable, reportName+" report is unavailable.")
			return
		}
		entry, err = client.AdminGenerateSupportReport(r.Context(), session)
	}
	if err != nil {
		a.writeSystemWorkspaceAPIError(w, err, reportName+ui(w, "ui.f574d2cdfe12"))
		return
	}
	_, timestampErr := time.Parse(time.RFC3339Nano, entry.Timestamp)
	if entry.Category != category || !validWorkspaceLogID(entry.Category, entry.LogID) || timestampErr != nil || entry.OperationID != "" || entry.OperationType != "" || entry.State != "" {
		writeSystemWorkspaceError(w, http.StatusBadGateway, reportName+" report is unavailable.")
		return
	}
	writeSystemWorkspaceJSON(w, http.StatusOK, map[string]any{"ok": true, "log": entry})
}
