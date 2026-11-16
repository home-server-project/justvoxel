package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/home-server-project/justvoxel-webui/internal/api"
)

type adminSystemActionsAPI interface {
	Session(ctx context.Context, session string) (api.SessionInfo, error)
	AdminSystemActionsStatus(ctx context.Context, session string) (api.AdminSystemActionsStatus, error)
	AdminSystemAction(ctx context.Context, session, action string, confirmPlayers bool) (api.AdminSystemActionResponse, error)
}

func (a *App) registerAdminSystemActionPages(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/system-actions", a.systemActionsStatus)
	mux.HandleFunc("GET /api/system-hardware", a.systemHardwareStatus)
	mux.HandleFunc("POST /api/system-actions/display", a.systemHardwareAction)
	mux.HandleFunc("POST /api/system-actions/{action}", a.systemAction)
}

func (a *App) systemActionsStatus(w http.ResponseWriter, r *http.Request) {
	session, client, ok := a.systemActionsRequest(w, r, false)
	if !ok {
		return
	}
	status, err := client.AdminSystemActionsStatus(r.Context(), session)
	if err != nil {
		a.handleSystemActionsAPIError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	if err := json.NewEncoder(w).Encode(status); err != nil {
		http.Error(w, ui(w, "ui.77f9ad942b89"), http.StatusInternalServerError)
	}
}

func (a *App) systemAction(w http.ResponseWriter, r *http.Request) {
	session, client, ok := a.systemActionsRequest(w, r, true)
	if !ok {
		return
	}
	action := r.PathValue("action")
	switch action {
	case "reboot", "poweroff", "firmware-reboot":
	default:
		http.Error(w, ui(w, "ui.0aa2aff32ba8"), http.StatusBadRequest)
		return
	}
	result, err := client.AdminSystemAction(r.Context(), session, action, r.FormValue("confirm_players") == "yes")
	if err != nil {
		if errors.Is(err, api.ErrUnauthorized) {
			a.clearSessionCookies(w)
			http.Error(w, ui(w, "ui.cbac8bee7536"), http.StatusUnauthorized)
			return
		}
		if errors.Is(err, api.ErrPasswordChangeRequired) {
			http.Error(w, ui(w, "ui.63bae92440ec"), http.StatusForbidden)
			return
		}
		var responseErr *api.ResponseError
		if errors.As(err, &responseErr) {
			status := responseErr.StatusCode
			if status < 400 || status > 599 {
				status = http.StatusBadGateway
			}
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "no-store")
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(result)
			return
		}
		http.Error(w, ui(w, "ui.56fbe7d555c8"), http.StatusBadGateway)
		return
	}
	status := http.StatusOK
	if result.Accepted {
		status = http.StatusAccepted
	} else if !result.ConfirmationRequired {
		http.Error(w, ui(w, "ui.477c8f8976a1"), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(result); err != nil {
		return
	}
}

func (a *App) systemActionsRequest(w http.ResponseWriter, r *http.Request, requireCSRF bool) (string, adminSystemActionsAPI, bool) {
	if requireCSRF && !a.validCSRF(r) {
		http.Error(w, "invalid CSRF token", http.StatusForbidden)
		return "", nil, false
	}
	if mustChange(r) {
		http.Error(w, ui(w, "ui.63bae92440ec"), http.StatusForbidden)
		return "", nil, false
	}
	session, ok := sessionFromRequest(r)
	if !ok {
		http.Error(w, ui(w, "ui.cbac8bee7536"), http.StatusUnauthorized)
		return "", nil, false
	}
	client, ok := a.api.(adminSystemActionsAPI)
	if !ok {
		http.Error(w, ui(w, "ui.4432793e595c"), http.StatusServiceUnavailable)
		return "", nil, false
	}
	identity, err := client.Session(r.Context(), session)
	if err != nil {
		a.handleSystemActionsAPIError(w, err)
		return "", nil, false
	}
	if identity.Role != "administrator" {
		http.Error(w, ui(w, "ui.031cc7a3ec82"), http.StatusForbidden)
		return "", nil, false
	}
	return session, client, true
}

func (a *App) handleSystemActionsAPIError(w http.ResponseWriter, err error) {
	if errors.Is(err, api.ErrUnauthorized) {
		a.clearSessionCookies(w)
		http.Error(w, ui(w, "ui.cbac8bee7536"), http.StatusUnauthorized)
		return
	}
	if errors.Is(err, api.ErrPasswordChangeRequired) {
		http.Error(w, ui(w, "ui.63bae92440ec"), http.StatusForbidden)
		return
	}
	var responseErr *api.ResponseError
	if errors.As(err, &responseErr) && responseErr.StatusCode == http.StatusForbidden {
		http.Error(w, ui(w, "ui.031cc7a3ec82"), http.StatusForbidden)
		return
	}
	http.Error(w, ui(w, "ui.56fbe7d555c8"), http.StatusBadGateway)
}
