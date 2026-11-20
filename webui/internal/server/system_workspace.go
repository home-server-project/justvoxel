package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/home-server-project/justvoxel-webui/internal/api"
)

type systemWorkspaceValidationAPI interface {
	AdminValidation(ctx context.Context, session string) (api.AdminValidationResponse, error)
}

type systemWorkspaceHistoryAPI interface {
	Activity(ctx context.Context, session string, limit int) (api.PublicActivityResponse, error)
	AdminAudit(ctx context.Context, session string, limit int) (api.AuditResponse, error)
	AdminNotifications(ctx context.Context, session string, openOnly bool) (api.NotificationsResponse, error)
	AdminResolveNotification(ctx context.Context, session string, id int64) error
}

type systemWorkspaceUsersAPI interface {
	AdminUsers(ctx context.Context, session string) (api.AdminUsersResponse, error)
	AdminCreateUser(ctx context.Context, session, username, role, password string) (api.AdminUser, error)
	AdminSetUserRole(ctx context.Context, session string, id int64, role string) (api.AdminUser, error)
	AdminSetUserEnabled(ctx context.Context, session string, id int64, enabled bool) (api.AdminUser, error)
	AdminSetUserPassword(ctx context.Context, session string, id int64, password string) error
	AdminDeleteUser(ctx context.Context, session string, id int64) error
	AdminResetRestartAllowance(ctx context.Context, session string, id int64) (api.AdminUser, error)
	AdminResetBackupAllowance(ctx context.Context, session string, id int64) (api.AdminUser, error)
}

type systemWorkspaceResetAPI interface {
	AdminMinecraftResetPlan(ctx context.Context, session string) (api.AdminResetPlanResponse, error)
	AdminFactoryResetPlan(ctx context.Context, session string) (api.AdminResetPlanResponse, error)
	AdminMinecraftResetApply(ctx context.Context, session string, request api.AdminMinecraftResetApplyRequest) (api.AdminResetApplyResponse, error)
	AdminFactoryResetApply(ctx context.Context, session string, request api.AdminFactoryResetApplyRequest) (api.AdminResetApplyResponse, error)
	AdminFactoryResetResolve(ctx context.Context, session, operationID string) (api.PersistentOperationResponse, error)
	AdminCurrentMinecraftResetOperation(ctx context.Context, session string) (api.PersistentOperationResponse, error)
	AdminCurrentFactoryResetOperation(ctx context.Context, session string) (api.PersistentOperationResponse, error)
	AdminOperation(ctx context.Context, session, id string) (api.PersistentOperationResponse, error)
}

type systemWorkspaceHealthResponse struct {
	OK     bool                        `json:"ok"`
	Result api.AdminValidationResponse `json:"result"`
}

type systemWorkspaceHistoryResponse struct {
	OK            bool                      `json:"ok"`
	Role          string                    `json:"role"`
	Username      string                    `json:"username"`
	Notifications []api.Notification        `json:"notifications,omitempty"`
	Audit         []api.AuditEvent          `json:"audit,omitempty"`
	Events        []api.PublicActivityEvent `json:"events,omitempty"`
}

type systemWorkspaceUsersResponse struct {
	OK                   bool                `json:"ok"`
	Username             string              `json:"username"`
	PrimaryAdministrator string              `json:"primary_administrator"`
	PrimaryTwoFactor     api.TwoFactorStatus `json:"primary_two_factor"`
	Users                []api.AdminUser     `json:"users"`
	MinimumPasswordLen   int                 `json:"minimum_password_len"`
	Message              string              `json:"message,omitempty"`
}

type systemWorkspaceSecurityResponse struct {
	OK                 bool   `json:"ok"`
	Mode               string `json:"mode"`
	Username           string `json:"username"`
	MinimumPasswordLen int    `json:"minimum_password_len"`
}

type systemWorkspaceMutationResponse struct {
	OK             bool   `json:"ok"`
	Message        string `json:"message,omitempty"`
	Reauthenticate bool   `json:"reauthenticate,omitempty"`
}

type systemWorkspaceAboutResponse struct {
	OK            bool   `json:"ok"`
	Username      string `json:"username"`
	Role          string `json:"role"`
	Variant       string `json:"variant"`
	JustVoxel     string `json:"justvoxel"`
	WebUI         string `json:"webui"`
	ManagementAPI string `json:"management_api"`
	Commit        string `json:"commit"`
}

type systemWorkspaceErrorResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
}

func (a *App) registerSystemWorkspaceRoutes(mux *http.ServeMux) {
	a.registerTerminalRoutes(mux)
	mux.HandleFunc("GET /api/system/workspace/date-time", a.systemWorkspaceDateTime)
	mux.HandleFunc("POST /api/system/workspace/date-time", a.systemWorkspaceDateTimeApply)
	mux.HandleFunc("GET /api/system/workspace/health", a.systemWorkspaceHealth)
	mux.HandleFunc("POST /api/system/workspace/logs/support", a.systemWorkspaceSupportReport)
	mux.HandleFunc("POST /api/system/workspace/logs/network", a.systemWorkspaceNetworkReport)
	mux.HandleFunc("GET /api/system/workspace/logs", a.systemWorkspaceLogs)
	mux.HandleFunc("GET /api/system/workspace/logs/failed/{unit}", a.systemWorkspaceFailedUnitJournal)
	mux.HandleFunc("GET /api/system/workspace/logs/{category}/{id}", a.systemWorkspaceLog)
	mux.HandleFunc("GET /api/system/workspace/history", a.systemWorkspaceHistory)
	mux.HandleFunc("POST /api/system/workspace/history/notifications/{id}/resolve", a.systemWorkspaceResolveNotification)
	mux.HandleFunc("GET /api/system/workspace/users", a.systemWorkspaceUsers)
	mux.HandleFunc("POST /api/system/workspace/users/create", a.systemWorkspaceCreateUser)
	mux.HandleFunc("POST /api/system/workspace/users/{id}/role", a.systemWorkspaceSetUserRole)
	mux.HandleFunc("POST /api/system/workspace/users/{id}/enabled", a.systemWorkspaceSetUserEnabled)
	mux.HandleFunc("POST /api/system/workspace/users/{id}/password", a.systemWorkspaceSetUserPassword)
	mux.HandleFunc("POST /api/system/workspace/users/{id}/delete", a.systemWorkspaceDeleteUser)
	mux.HandleFunc("POST /api/system/workspace/users/{id}/restart-allowance/reset", a.systemWorkspaceResetRestartAllowance)
	mux.HandleFunc("POST /api/system/workspace/users/{id}/backup-allowance/reset", a.systemWorkspaceResetBackupAllowance)
	mux.HandleFunc("GET /api/system/workspace/security", a.systemWorkspaceSecurity)
	mux.HandleFunc("POST /api/system/workspace/security/authentication", a.systemWorkspaceAuthenticationChange)
	mux.HandleFunc("POST /api/system/workspace/security/password", a.systemWorkspacePasswordChange)
	mux.HandleFunc("POST /api/system/workspace/reset/minecraft/plan", a.systemWorkspaceMinecraftResetPlan)
	mux.HandleFunc("POST /api/system/workspace/reset/minecraft/apply", a.systemWorkspaceMinecraftResetApply)
	mux.HandleFunc("GET /api/system/workspace/reset/minecraft/current", a.systemWorkspaceMinecraftResetCurrent)
	mux.HandleFunc("POST /api/system/workspace/reset/factory/plan", a.systemWorkspaceFactoryResetPlan)
	mux.HandleFunc("POST /api/system/workspace/reset/factory/apply", a.systemWorkspaceFactoryResetApply)
	mux.HandleFunc("POST /api/system/workspace/reset/factory/resolve", a.systemWorkspaceFactoryResetResolve)
	mux.HandleFunc("GET /api/system/workspace/reset/factory/current", a.systemWorkspaceFactoryResetCurrent)
	mux.HandleFunc("GET /api/system/workspace/reset/operations/{id}", a.systemWorkspaceResetOperation)
	mux.HandleFunc("GET /api/operations/{id}/trace", a.operationTrace)
	mux.HandleFunc("GET /api/system/workspace/about", a.systemWorkspaceAbout)
}

func (a *App) systemWorkspaceIdentity(w http.ResponseWriter, r *http.Request, allowed ...string) (string, api.SessionInfo, bool) {
	if mustChange(r) {
		writeSystemWorkspaceError(w, http.StatusForbidden, ui(w, "ui.fe5ad66e02ea"))
		return "", api.SessionInfo{}, false
	}
	session, ok := sessionFromRequest(r)
	if !ok {
		writeSystemWorkspaceError(w, http.StatusUnauthorized, ui(w, "ui.45cceda8377a"))
		return "", api.SessionInfo{}, false
	}
	client, ok := a.api.(sessionIdentityAPI)
	if !ok {
		writeSystemWorkspaceError(w, http.StatusServiceUnavailable, ui(w, "ui.98a9fb5b71d1"))
		return "", api.SessionInfo{}, false
	}
	identity, err := client.Session(r.Context(), session)
	if err != nil {
		a.writeSystemWorkspaceAPIError(w, err, ui(w, "ui.98a9fb5b71d1"))
		return "", api.SessionInfo{}, false
	}
	for _, role := range allowed {
		if identity.Role == role {
			return session, identity, true
		}
	}
	writeSystemWorkspaceError(w, http.StatusForbidden, ui(w, "ui.765e141d59df"))
	return "", api.SessionInfo{}, false
}

func (a *App) systemWorkspaceHealth(w http.ResponseWriter, r *http.Request) {
	session, _, ok := a.systemWorkspaceIdentity(w, r, "administrator")
	if !ok {
		return
	}
	client, ok := a.api.(systemWorkspaceValidationAPI)
	if !ok {
		writeSystemWorkspaceError(w, http.StatusServiceUnavailable, ui(w, "ui.1c13e84c5086"))
		return
	}
	result, err := client.AdminValidation(r.Context(), session)
	if err != nil {
		a.writeSystemWorkspaceAPIError(w, err, ui(w, "ui.1c13e84c5086"))
		return
	}
	writeSystemWorkspaceJSON(w, http.StatusOK, systemWorkspaceHealthResponse{OK: true, Result: result})
}

func (a *App) systemWorkspaceHistory(w http.ResponseWriter, r *http.Request) {
	session, identity, ok := a.systemWorkspaceIdentity(w, r, "administrator", "operator", "viewer")
	if !ok {
		return
	}
	client, ok := a.api.(systemWorkspaceHistoryAPI)
	if !ok {
		writeSystemWorkspaceError(w, http.StatusServiceUnavailable, ui(w, "ui.99d46563a71b"))
		return
	}
	response := systemWorkspaceHistoryResponse{OK: true, Role: identity.Role, Username: identity.Username}
	if identity.Role == "administrator" {
		audit, err := client.AdminAudit(r.Context(), session, 100)
		if err != nil {
			a.writeSystemWorkspaceAPIError(w, err, ui(w, "ui.fbb800527085"))
			return
		}
		notifications, err := client.AdminNotifications(r.Context(), session, true)
		if err != nil {
			a.writeSystemWorkspaceAPIError(w, err, ui(w, "ui.7925452cdfcb"))
			return
		}
		response.Audit = audit.Events
		response.Notifications = notifications.Notifications
	} else {
		activity, err := client.Activity(r.Context(), session, 100)
		if err != nil {
			a.writeSystemWorkspaceAPIError(w, err, ui(w, "ui.99d46563a71b"))
			return
		}
		response.Events = activity.Events
	}
	writeSystemWorkspaceJSON(w, http.StatusOK, response)
}

func (a *App) systemWorkspaceResolveNotification(w http.ResponseWriter, r *http.Request) {
	if !a.validCSRF(r) {
		writeSystemWorkspaceError(w, http.StatusForbidden, ui(w, "ui.74bccc44f472"))
		return
	}
	session, _, ok := a.systemWorkspaceIdentity(w, r, "administrator")
	if !ok {
		return
	}
	client, ok := a.api.(systemWorkspaceHistoryAPI)
	if !ok {
		writeSystemWorkspaceError(w, http.StatusServiceUnavailable, ui(w, "ui.7925452cdfcb"))
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeSystemWorkspaceError(w, http.StatusBadRequest, ui(w, "ui.b780fcec4a5d"))
		return
	}
	if err := client.AdminResolveNotification(r.Context(), session, id); err != nil {
		a.writeSystemWorkspaceAPIError(w, err, ui(w, "ui.a63ba84b3034"))
		return
	}
	writeSystemWorkspaceJSON(w, http.StatusOK, systemWorkspaceMutationResponse{OK: true, Message: ui(w, "ui.4f376daea4c4")})
}

func (a *App) systemWorkspaceUsers(w http.ResponseWriter, r *http.Request) {
	session, identity, ok := a.systemWorkspaceIdentity(w, r, "administrator")
	if !ok {
		return
	}
	client, ok := a.api.(systemWorkspaceUsersAPI)
	if !ok {
		writeSystemWorkspaceError(w, http.StatusServiceUnavailable, ui(w, "ui.658d4d48c261"))
		return
	}
	payload, err := client.AdminUsers(r.Context(), session)
	if err != nil {
		a.writeSystemWorkspaceAPIError(w, err, ui(w, "ui.658d4d48c261"))
		return
	}
	if payload.Users == nil {
		payload.Users = []api.AdminUser{}
	}
	writeSystemWorkspaceJSON(w, http.StatusOK, systemWorkspaceUsersResponse{
		OK: true, Username: identity.Username, PrimaryAdministrator: payload.PrimaryAdministrator.Username,
		PrimaryTwoFactor: payload.PrimaryAdministrator.TwoFactorStatus,
		Users:            payload.Users, MinimumPasswordLen: payload.MinimumPasswordLen,
	})
}

func (a *App) systemWorkspaceCreateUser(w http.ResponseWriter, r *http.Request) {
	session, client, ok := a.systemWorkspaceUsersMutation(w, r)
	if !ok {
		return
	}
	password := r.FormValue("password")
	if password == "" || password != r.FormValue("confirm_password") {
		writeSystemWorkspaceError(w, http.StatusBadRequest, ui(w, "ui.290c87f0a273"))
		return
	}
	if _, err := client.AdminCreateUser(r.Context(), session, strings.TrimSpace(r.FormValue("username")), r.FormValue("role"), password); err != nil {
		a.writeSystemWorkspaceAPIError(w, err, ui(w, "ui.969cadf867fb"))
		return
	}
	writeSystemWorkspaceJSON(w, http.StatusCreated, systemWorkspaceMutationResponse{OK: true, Message: ui(w, "ui.8b71b6651c39")})
}

func (a *App) systemWorkspaceSetUserRole(w http.ResponseWriter, r *http.Request) {
	session, client, id, ok := a.systemWorkspaceUserMutation(w, r)
	if !ok {
		return
	}
	if _, err := client.AdminSetUserRole(r.Context(), session, id, r.FormValue("role")); err != nil {
		a.writeSystemWorkspaceAPIError(w, err, ui(w, "ui.81294a2aebf6"))
		return
	}
	writeSystemWorkspaceJSON(w, http.StatusOK, systemWorkspaceMutationResponse{OK: true, Message: ui(w, "ui.6107d3a04c7b")})
}

func (a *App) systemWorkspaceSetUserEnabled(w http.ResponseWriter, r *http.Request) {
	session, client, id, ok := a.systemWorkspaceUserMutation(w, r)
	if !ok {
		return
	}
	enabled := r.FormValue("enabled") == "true"
	if _, err := client.AdminSetUserEnabled(r.Context(), session, id, enabled); err != nil {
		a.writeSystemWorkspaceAPIError(w, err, ui(w, "ui.2645111884d5"))
		return
	}
	writeSystemWorkspaceJSON(w, http.StatusOK, systemWorkspaceMutationResponse{OK: true, Message: ui(w, "ui.6107d3a04c7b")})
}

func (a *App) systemWorkspaceSetUserPassword(w http.ResponseWriter, r *http.Request) {
	session, client, id, ok := a.systemWorkspaceUserMutation(w, r)
	if !ok {
		return
	}
	password := r.FormValue("password")
	if password == "" || password != r.FormValue("confirm_password") {
		writeSystemWorkspaceError(w, http.StatusBadRequest, ui(w, "ui.290c87f0a273"))
		return
	}
	if err := client.AdminSetUserPassword(r.Context(), session, id, password); err != nil {
		a.writeSystemWorkspaceAPIError(w, err, ui(w, "ui.523a96f45acf"))
		return
	}
	writeSystemWorkspaceJSON(w, http.StatusOK, systemWorkspaceMutationResponse{OK: true, Message: ui(w, "ui.aec6d3cda8a6")})
}

func (a *App) systemWorkspaceDeleteUser(w http.ResponseWriter, r *http.Request) {
	session, client, id, ok := a.systemWorkspaceUserMutation(w, r)
	if !ok {
		return
	}
	if err := client.AdminDeleteUser(r.Context(), session, id); err != nil {
		a.writeSystemWorkspaceAPIError(w, err, ui(w, "ui.3b3d1ca80546"))
		return
	}
	writeSystemWorkspaceJSON(w, http.StatusOK, systemWorkspaceMutationResponse{OK: true, Message: ui(w, "ui.aaf26cfe7c36")})
}

func (a *App) systemWorkspaceResetRestartAllowance(w http.ResponseWriter, r *http.Request) {
	session, client, id, ok := a.systemWorkspaceUserMutation(w, r)
	if !ok {
		return
	}
	if _, err := client.AdminResetRestartAllowance(r.Context(), session, id); err != nil {
		a.writeSystemWorkspaceAPIError(w, err, ui(w, "ui.af5c8748fe90"))
		return
	}
	writeSystemWorkspaceJSON(w, http.StatusOK, systemWorkspaceMutationResponse{OK: true, Message: ui(w, "ui.b0e9fe90f7e1")})
}

func (a *App) systemWorkspaceResetBackupAllowance(w http.ResponseWriter, r *http.Request) {
	session, client, id, ok := a.systemWorkspaceUserMutation(w, r)
	if !ok {
		return
	}
	if _, err := client.AdminResetBackupAllowance(r.Context(), session, id); err != nil {
		a.writeSystemWorkspaceAPIError(w, err, ui(w, "ui.21c981f614e1"))
		return
	}
	writeSystemWorkspaceJSON(w, http.StatusOK, systemWorkspaceMutationResponse{OK: true, Message: ui(w, "ui.d14c123e4ea5")})
}

func (a *App) systemWorkspaceUsersMutation(w http.ResponseWriter, r *http.Request) (string, systemWorkspaceUsersAPI, bool) {
	if !a.validCSRF(r) {
		writeSystemWorkspaceError(w, http.StatusForbidden, ui(w, "ui.74bccc44f472"))
		return "", nil, false
	}
	session, _, ok := a.systemWorkspaceIdentity(w, r, "administrator")
	if !ok {
		return "", nil, false
	}
	client, ok := a.api.(systemWorkspaceUsersAPI)
	if !ok {
		writeSystemWorkspaceError(w, http.StatusServiceUnavailable, ui(w, "ui.658d4d48c261"))
		return "", nil, false
	}
	return session, client, true
}

func (a *App) systemWorkspaceUserMutation(w http.ResponseWriter, r *http.Request) (string, systemWorkspaceUsersAPI, int64, bool) {
	session, client, ok := a.systemWorkspaceUsersMutation(w, r)
	if !ok {
		return "", nil, 0, false
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeSystemWorkspaceError(w, http.StatusBadRequest, ui(w, "ui.4e48e87ed787"))
		return "", nil, 0, false
	}
	return session, client, id, true
}

func (a *App) systemWorkspaceSecurity(w http.ResponseWriter, r *http.Request) {
	session, _, ok := a.systemWorkspaceIdentity(w, r, "administrator")
	if !ok {
		return
	}
	client, ok := a.authAPI()
	if !ok {
		writeSystemWorkspaceError(w, http.StatusServiceUnavailable, ui(w, "ui.626618e6da03"))
		return
	}
	status, err := client.AuthStatus(r.Context(), session)
	if err != nil {
		a.writeSystemWorkspaceAPIError(w, err, ui(w, "ui.626618e6da03"))
		return
	}
	writeSystemWorkspaceJSON(w, http.StatusOK, systemWorkspaceSecurityResponse{
		OK: true, Mode: status.Mode, Username: status.Username, MinimumPasswordLen: status.MinimumPasswordLen,
	})
}

func (a *App) systemWorkspaceAuthenticationChange(w http.ResponseWriter, r *http.Request) {
	if !a.validCSRF(r) {
		writeSystemWorkspaceError(w, http.StatusForbidden, ui(w, "ui.74bccc44f472"))
		return
	}
	session, _, ok := a.systemWorkspaceIdentity(w, r, "administrator")
	if !ok {
		return
	}
	client, ok := a.authAPI()
	if !ok {
		writeSystemWorkspaceError(w, http.StatusServiceUnavailable, ui(w, "ui.626618e6da03"))
		return
	}
	current, err := client.AuthStatus(r.Context(), session)
	if err != nil {
		a.writeSystemWorkspaceAPIError(w, err, ui(w, "ui.626618e6da03"))
		return
	}
	change := api.AuthModeChange{
		Mode:               r.FormValue("mode"),
		SystemPassword:     r.FormValue("system_password"),
		NewWebPassword:     r.FormValue("new_web_password"),
		ConfirmWebPassword: r.FormValue("confirm_web_password"),
	}
	if change.Mode == "separate" && (change.NewWebPassword != change.ConfirmWebPassword || len(change.NewWebPassword) < current.MinimumPasswordLen) {
		writeSystemWorkspaceError(w, http.StatusBadRequest, ui(w, "ui.d3b7ff3d0ebe"))
		return
	}
	result, err := client.ChangeAuthMode(r.Context(), session, change)
	if errors.Is(err, api.ErrUnauthorized) {
		writeSystemWorkspaceError(w, http.StatusBadRequest, ui(w, "ui.bab9f9e9fd0b"))
		return
	}
	if err != nil {
		a.writeSystemWorkspaceAPIError(w, err, ui(w, "ui.f5adad2c4d22"))
		return
	}
	if result.Reauthenticate {
		a.clearSessionCookies(w)
	}
	writeSystemWorkspaceJSON(w, http.StatusOK, systemWorkspaceMutationResponse{OK: true, Message: ui(w, "ui.db5611ae62d9"), Reauthenticate: result.Reauthenticate})
}

func (a *App) systemWorkspacePasswordChange(w http.ResponseWriter, r *http.Request) {
	if !a.validCSRF(r) {
		writeSystemWorkspaceError(w, http.StatusForbidden, ui(w, "ui.74bccc44f472"))
		return
	}
	session, _, ok := a.systemWorkspaceIdentity(w, r, "administrator")
	if !ok {
		return
	}
	client, ok := a.authAPI()
	if !ok {
		writeSystemWorkspaceError(w, http.StatusServiceUnavailable, ui(w, "ui.626618e6da03"))
		return
	}
	status, err := client.AuthStatus(r.Context(), session)
	if err != nil {
		a.writeSystemWorkspaceAPIError(w, err, ui(w, "ui.626618e6da03"))
		return
	}
	currentPassword := r.FormValue("current_password")
	newPassword := r.FormValue("new_password")
	confirmPassword := r.FormValue("confirm_password")
	if newPassword != confirmPassword || len(newPassword) < status.MinimumPasswordLen {
		writeSystemWorkspaceError(w, http.StatusBadRequest, ui(w, "ui.bc857e7fc12d"))
		return
	}
	if err := a.api.ChangePassword(r.Context(), session, currentPassword, newPassword); err != nil {
		a.writeSystemWorkspaceAPIError(w, err, ui(w, "ui.8f9cdd4f0888"))
		return
	}
	a.clearSessionCookies(w)
	writeSystemWorkspaceJSON(w, http.StatusOK, systemWorkspaceMutationResponse{OK: true, Message: ui(w, "ui.a768c77be9dd"), Reauthenticate: true})
}

func (a *App) systemWorkspaceResetClient(w http.ResponseWriter, r *http.Request, mutation bool) (string, systemWorkspaceResetAPI, bool) {
	if mutation && !a.validCSRF(r) {
		writeSystemWorkspaceError(w, http.StatusForbidden, ui(w, "ui.74bccc44f472"))
		return "", nil, false
	}
	session, _, ok := a.systemWorkspaceIdentity(w, r, "administrator")
	if !ok {
		return "", nil, false
	}
	client, ok := a.api.(systemWorkspaceResetAPI)
	if !ok {
		writeSystemWorkspaceError(w, http.StatusServiceUnavailable, ui(w, "ui.df79081df425"))
		return "", nil, false
	}
	return session, client, true
}

func (a *App) systemWorkspaceMinecraftResetPlan(w http.ResponseWriter, r *http.Request) {
	session, client, ok := a.systemWorkspaceResetClient(w, r, true)
	if !ok {
		return
	}
	current, err := client.AdminCurrentMinecraftResetOperation(r.Context(), session)
	if err != nil {
		a.writeSystemWorkspaceAPIError(w, err, ui(w, "ui.dbd4bd0bb184"))
		return
	}
	var result api.AdminResetPlanResponse
	if current.Operation != nil && current.Operation.InstanceID != "" {
		recovery, ok := client.(interface {
			HistoricalMinecraftResetPlan(context.Context, string) (api.AdminResetPlanResponse, error)
		})
		if !ok {
			writeSystemWorkspaceError(w, 503, ui(w, "ui.54458d309cb5"))
			return
		}
		result, err = recovery.HistoricalMinecraftResetPlan(r.Context(), session)
	} else {
		result, err = client.AdminMinecraftResetPlan(r.Context(), session)
	}
	if err != nil {
		a.writeSystemWorkspaceAPIError(w, err, ui(w, "ui.6fc175508844"))
		return
	}
	writeSystemWorkspaceJSON(w, http.StatusOK, result)
}

func (a *App) systemWorkspaceFactoryResetPlan(w http.ResponseWriter, r *http.Request) {
	session, client, ok := a.systemWorkspaceResetClient(w, r, true)
	if !ok {
		return
	}
	result, err := client.AdminFactoryResetPlan(r.Context(), session)
	if err != nil {
		a.writeSystemWorkspaceAPIError(w, err, ui(w, "ui.2998c5d0dbd7"))
		return
	}
	writeSystemWorkspaceJSON(w, http.StatusOK, result)
}

func (a *App) systemWorkspaceMinecraftResetApply(w http.ResponseWriter, r *http.Request) {
	session, client, ok := a.systemWorkspaceResetClient(w, r, true)
	if !ok {
		return
	}
	current, err := client.AdminCurrentMinecraftResetOperation(r.Context(), session)
	if err != nil {
		a.writeSystemWorkspaceAPIError(w, err, ui(w, "ui.dbd4bd0bb184"))
		return
	}
	request := api.AdminMinecraftResetApplyRequest{PlanFingerprint: r.FormValue("plan_fingerprint"), ConfirmPlayers: r.FormValue("confirm_players") == "yes"}
	var result api.AdminResetApplyResponse
	if current.Operation != nil && current.Operation.InstanceID != "" {
		recovery, ok := client.(interface {
			HistoricalMinecraftResetApply(context.Context, string, api.AdminMinecraftResetApplyRequest) (api.AdminResetApplyResponse, error)
		})
		if !ok {
			writeSystemWorkspaceError(w, 503, ui(w, "ui.54458d309cb5"))
			return
		}
		result, err = recovery.HistoricalMinecraftResetApply(r.Context(), session, request)
	} else {
		result, err = client.AdminMinecraftResetApply(r.Context(), session, request)
	}
	if err != nil {
		a.writeSystemWorkspaceAPIError(w, err, ui(w, "ui.5ee3ade8e7a6"))
		return
	}
	writeSystemWorkspaceJSON(w, http.StatusAccepted, result)
}

func (a *App) systemWorkspaceFactoryResetApply(w http.ResponseWriter, r *http.Request) {
	session, client, ok := a.systemWorkspaceResetClient(w, r, true)
	if !ok {
		return
	}
	password := r.FormValue("system_password")
	result, err := client.AdminFactoryResetApply(r.Context(), session, api.AdminFactoryResetApplyRequest{
		PlanFingerprint: r.FormValue("plan_fingerprint"),
		ConfirmPlayers:  r.FormValue("confirm_players") == "yes",
		SystemPassword:  password,
	})
	password = ""
	if err != nil {
		a.writeSystemWorkspaceAPIError(w, err, ui(w, "ui.a80e0aec453f"))
		return
	}
	writeSystemWorkspaceJSON(w, http.StatusAccepted, result)
}

func (a *App) systemWorkspaceFactoryResetResolve(w http.ResponseWriter, r *http.Request) {
	if !a.validCSRF(r) {
		writeSystemWorkspaceError(w, http.StatusForbidden, ui(w, "ui.74bccc44f472"))
		return
	}
	session, client, ok := a.systemWorkspaceResetClient(w, r, true)
	if !ok {
		return
	}
	operationID := strings.TrimSpace(r.FormValue("operation_id"))
	if operationID == "" || r.FormValue("keep_current_state") != "yes" {
		writeSystemWorkspaceError(w, http.StatusBadRequest, ui(w, "ui.dde3dc2d803f"))
		return
	}
	result, err := client.AdminFactoryResetResolve(r.Context(), session, operationID)
	if err != nil {
		a.writeSystemWorkspaceAPIError(w, err, ui(w, "ui.7cfc47e3f260"))
		return
	}
	writeSystemWorkspaceJSON(w, http.StatusOK, result)
}

func (a *App) systemWorkspaceMinecraftResetCurrent(w http.ResponseWriter, r *http.Request) {
	session, client, ok := a.systemWorkspaceResetClient(w, r, false)
	if !ok {
		return
	}
	result, err := client.AdminCurrentMinecraftResetOperation(r.Context(), session)
	if err != nil {
		a.writeSystemWorkspaceAPIError(w, err, ui(w, "ui.dbd4bd0bb184"))
		return
	}
	writeSystemWorkspaceJSON(w, http.StatusOK, result)
}

func (a *App) systemWorkspaceFactoryResetCurrent(w http.ResponseWriter, r *http.Request) {
	session, client, ok := a.systemWorkspaceResetClient(w, r, false)
	if !ok {
		return
	}
	result, err := client.AdminCurrentFactoryResetOperation(r.Context(), session)
	if err != nil {
		a.writeSystemWorkspaceAPIError(w, err, ui(w, "ui.ad1fb4da0cd8"))
		return
	}
	writeSystemWorkspaceJSON(w, http.StatusOK, result)
}

func (a *App) systemWorkspaceResetOperation(w http.ResponseWriter, r *http.Request) {
	session, client, ok := a.systemWorkspaceResetClient(w, r, false)
	if !ok {
		return
	}
	result, err := client.AdminOperation(r.Context(), session, r.PathValue("id"))
	if err != nil {
		a.writeSystemWorkspaceAPIError(w, err, ui(w, "ui.63cdddd4666a"))
		return
	}
	if result.Operation != nil && result.Operation.OperationType != "minecraft_reset" && result.Operation.OperationType != "factory_reset" {
		writeSystemWorkspaceError(w, http.StatusNotFound, ui(w, "ui.882e439dfa4a"))
		return
	}
	if result.Operation != nil && result.Operation.OperationType == "minecraft_reset" && result.Operation.InstanceID != "" {
		values, explicit := r.URL.Query()["server"]
		requested := ""
		if explicit {
			if len(values) != 1 || !addServerIDPattern.MatchString(values[0]) {
				writeSystemWorkspaceError(w, 400, ui(w, "ui.abda10a5fa44"))
				return
			}
			requested = values[0]
		}
		if !explicit {
			selected, ok := a.selectedMinecraft(w, r, session)
			if !ok {
				return
			}
			requested = selected.ID
		}
		if !addServerIDPattern.MatchString(requested) || result.Operation.InstanceID != requested {
			writeSystemWorkspaceError(w, 409, ui(w, "ui.0306cb441b23"))
			return
		}
	}
	writeSystemWorkspaceJSON(w, http.StatusOK, result)
}

func (a *App) systemWorkspaceAbout(w http.ResponseWriter, r *http.Request) {
	session, identity, ok := a.systemWorkspaceIdentity(w, r, "administrator", "operator", "viewer")
	if !ok {
		return
	}
	status, err := a.api.Status(r.Context(), session)
	if err != nil {
		a.writeSystemWorkspaceAPIError(w, err, ui(w, "ui.89be1d273a28"))
		return
	}
	writeSystemWorkspaceJSON(w, http.StatusOK, systemWorkspaceAboutResponse{
		OK: true, Username: identity.Username, Role: identity.Role, Variant: status.System.Variant,
		JustVoxel: status.JustVoxel.Version, WebUI: a.config.Version, ManagementAPI: a.config.ManagementAPI, Commit: a.config.Commit,
	})
}

func (a *App) writeSystemWorkspaceAPIError(w http.ResponseWriter, err error, fallback string) {
	status := http.StatusBadGateway
	message := fallback
	if errors.Is(err, api.ErrUnauthorized) {
		status = http.StatusUnauthorized
		message = ui(w, "ui.45cceda8377a")
	} else if errors.Is(err, api.ErrPasswordChangeRequired) {
		status = http.StatusForbidden
		message = ui(w, "ui.fe5ad66e02ea")
	} else {
		var responseErr *api.ResponseError
		if errors.As(err, &responseErr) {
			if responseErr.StatusCode >= 400 && responseErr.StatusCode <= 599 {
				status = responseErr.StatusCode
			}
			if responseErr.Message != "" {
				message = responseErr.Message
			}
		}
	}
	writeSystemWorkspaceError(w, status, message)
}

func writeSystemWorkspaceError(w http.ResponseWriter, status int, message string) {
	writeSystemWorkspaceJSON(w, status, systemWorkspaceErrorResponse{OK: false, Error: message})
}

func writeSystemWorkspaceJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func (a *App) selectedResetRequest(w http.ResponseWriter, r *http.Request, session string, client systemWorkspaceResetAPI) (*http.Request, bool) {
	values, explicit := r.URL.Query()["server"]
	if explicit && len(values) == 1 && addServerIDPattern.MatchString(values[0]) {
		ctx := api.WithMinecraftMaintenanceInstance(r.Context(), values[0], "reset/minecraft")
		current, err := client.AdminCurrentMinecraftResetOperation(ctx, session)
		if err == nil && current.Operation != nil && current.Operation.InstanceID == values[0] {
			return r.WithContext(ctx), true
		}
	}
	selected, ok := a.selectedMinecraft(w, r, session)
	if !ok {
		return r, false
	}
	return r.WithContext(api.WithMinecraftMaintenanceInstance(r.Context(), selected.ID, "reset/minecraft")), true
}
