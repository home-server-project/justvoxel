package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/home-server-project/justvoxel-webui/internal/api"
)

func (a *App) registerRolePages(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/session-info", a.sessionInfo)
	mux.HandleFunc("POST /api/backups/manual", a.quickLookManualBackup)
}

func (a *App) sessionInfo(w http.ResponseWriter, r *http.Request) {
	_, _, identity, ok := a.rolePageRequest(w, r)
	if !ok {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	if err := json.NewEncoder(w).Encode(identity); err != nil {
		http.Error(w, ui(w, "ui.d08b96cb2adf"), http.StatusInternalServerError)
	}
}

func (a *App) quickLookManualBackup(w http.ResponseWriter, r *http.Request) {
	if !a.validCSRF(r) {
		writeMinecraftWorkspaceError(w, 403, "invalid CSRF token")
		return
	}
	session, _, ok := a.minecraftWorkspaceIdentity(w, r, "administrator", "operator")
	if !ok {
		return
	}
	selected, ok := a.selectedMinecraft(w, r, session)
	if !ok {
		return
	}
	client, ok := a.api.(selectedBackupsAPI)
	if !ok {
		writeMinecraftWorkspaceError(w, 503, ui(w, "ui.6ebddf892354"))
		return
	}
	if _, err := client.MinecraftInstanceBackups(r.Context(), session, selected.ID, "now", struct{}{}); err != nil {
		a.writeMinecraftWorkspaceAPIError(w, err, ui(w, "ui.8d2d72a33769"))
		return
	}
	writeMinecraftWorkspaceJSON(w, 200, map[string]any{"ok": true, "message": ui(w, "ui.c05a5abd8eea")})
}

func (a *App) rolePageRequest(w http.ResponseWriter, r *http.Request) (string, sessionIdentityAPI, api.SessionInfo, bool) {
	session, ok := sessionFromRequest(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return "", nil, api.SessionInfo{}, false
	}
	client, ok := a.api.(sessionIdentityAPI)
	if !ok {
		http.Error(w, ui(w, "ui.db238a1fe47d"), http.StatusServiceUnavailable)
		return "", nil, api.SessionInfo{}, false
	}
	identity, err := client.Session(r.Context(), session)
	if err != nil {
		a.handleRolePageError(w, r, err)
		return "", nil, api.SessionInfo{}, false
	}
	return session, client, identity, true
}

func (a *App) handleRolePageError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, api.ErrUnauthorized) {
		a.clearSessionCookies(w)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	var responseErr *api.ResponseError
	if errors.As(err, &responseErr) && responseErr.StatusCode == http.StatusForbidden {
		http.Error(w, ui(w, "ui.cc11d415d932"), http.StatusForbidden)
		return
	}
	http.Error(w, ui(w, "ui.0b8e73a792bb"), http.StatusBadGateway)
}
