package main

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"
)

type adminUserView struct {
	ID                int64       `json:"id"`
	Username          string      `json:"username"`
	Role              webUserRole `json:"role"`
	Enabled           bool        `json:"enabled"`
	CreatedAt         string      `json:"created_at"`
	UpdatedAt         string      `json:"updated_at"`
	PasswordChangedAt string      `json:"password_changed_at"`
	RestartUsed       int         `json:"restart_used"`
	RestartLimit      int         `json:"restart_limit"`
	LastRestartAt     *string     `json:"last_restart_at,omitempty"`
	BackupUsed        int         `json:"backup_used"`
	BackupLimit       int         `json:"backup_limit"`
	LastBackupAt      *string     `json:"last_backup_at,omitempty"`
}

func registerAdminUserRoutes(mux *http.ServeMux, s *server) {
	mux.HandleFunc("GET /v1/admin/users", s.adminListUsers)
	mux.HandleFunc("POST /v1/admin/users", s.adminCreateUser)
	mux.HandleFunc("PATCH /v1/admin/users/{id}/role", s.adminSetUserRole)
	mux.HandleFunc("PATCH /v1/admin/users/{id}/enabled", s.adminSetUserEnabled)
	mux.HandleFunc("POST /v1/admin/users/{id}/password", s.adminSetUserPassword)
	mux.HandleFunc("DELETE /v1/admin/users/{id}", s.adminDeleteUser)
	mux.HandleFunc("GET /v1/admin/users/{id}/usage", s.adminUserUsage)
	mux.HandleFunc("POST /v1/admin/users/{id}/restart-allowance/reset", s.adminResetRestartAllowance)
	mux.HandleFunc("POST /v1/admin/users/{id}/backup-allowance/reset", s.adminResetBackupAllowance)
}

func (s *server) adminListUsers(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdministrator(w, r); !ok {
		return
	}
	users, err := s.store.listAdminUserViews()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "WebUI users are unavailable")
		return
	}
	policy, err := administratorPasswordPolicy()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "password policy unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"primary_administrator": map[string]any{
			"username": systemAdminUsername,
			"role":     roleAdministrator,
			"managed":  false,
		},
		"users":                users,
		"minimum_password_len": policy.MinLength,
	})
}

func (s *server) adminCreateUser(w http.ResponseWriter, r *http.Request) {
	actor, ok := s.requireAdministrator(w, r)
	if !ok {
		return
	}
	var request struct {
		Username string      `json:"username"`
		Role     webUserRole `json:"role"`
		Password string      `json:"password"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	username, err := normalizeWebUsername(request.Username)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if !validWebUserRole(request.Role) {
		writeError(w, http.StatusBadRequest, errWebUserInvalidRole.Error())
		return
	}
	if request.Password == "" {
		writeError(w, http.StatusBadRequest, errWebUserPasswordEmpty.Error())
		return
	}
	if err := systemValidatePass(username, "", request.Password); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if _, err := s.store.webUserByUsername(username); err == nil {
		writeError(w, http.StatusConflict, "WebUI username already exists")
		return
	} else if !errors.Is(err, errWebUserNotFound) {
		writeError(w, http.StatusServiceUnavailable, "WebUI users are unavailable")
		return
	}
	user, err := s.store.createWebUser(username, request.Role, request.Password)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			writeError(w, http.StatusConflict, "WebUI username already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create WebUI user")
		return
	}
	_ = s.store.recordAuditEvent(actor, "create_webui_user", user.Username, true, "role="+string(user.Role))
	view, err := s.store.adminUserViewByID(user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "WebUI user was created but could not be read")
		return
	}
	writeJSON(w, http.StatusCreated, view)
}

func (s *server) adminSetUserRole(w http.ResponseWriter, r *http.Request) {
	actor, ok := s.requireAdministrator(w, r)
	if !ok {
		return
	}
	id, ok := adminPathUserID(w, r)
	if !ok {
		return
	}
	var request struct {
		Role webUserRole `json:"role"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	if !validWebUserRole(request.Role) {
		writeError(w, http.StatusBadRequest, errWebUserInvalidRole.Error())
		return
	}
	user, err := s.store.webUserByID(id)
	if err != nil {
		writeAdminUserStoreError(w, err)
		return
	}
	if err := s.store.setWebUserRole(id, request.Role); err != nil {
		writeAdminUserStoreError(w, err)
		return
	}
	s.invalidateWebUserSessions(id)
	_ = s.store.recordAuditEvent(actor, "change_webui_user_role", user.Username, true, "role="+string(request.Role))
	view, err := s.store.adminUserViewByID(id)
	if err != nil {
		writeAdminUserStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *server) adminSetUserEnabled(w http.ResponseWriter, r *http.Request) {
	actor, ok := s.requireAdministrator(w, r)
	if !ok {
		return
	}
	id, ok := adminPathUserID(w, r)
	if !ok {
		return
	}
	var request struct {
		Enabled *bool `json:"enabled"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	if request.Enabled == nil {
		writeError(w, http.StatusBadRequest, "enabled is required")
		return
	}
	user, err := s.store.webUserByID(id)
	if err != nil {
		writeAdminUserStoreError(w, err)
		return
	}
	if err := s.store.setWebUserEnabled(id, *request.Enabled); err != nil {
		writeAdminUserStoreError(w, err)
		return
	}
	if !*request.Enabled {
		s.invalidateWebUserSessions(id)
	}
	_ = s.store.recordAuditEvent(actor, "set_webui_user_enabled", user.Username, true, strconv.FormatBool(*request.Enabled))
	view, err := s.store.adminUserViewByID(id)
	if err != nil {
		writeAdminUserStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *server) adminSetUserPassword(w http.ResponseWriter, r *http.Request) {
	actor, ok := s.requireAdministrator(w, r)
	if !ok {
		return
	}
	id, ok := adminPathUserID(w, r)
	if !ok {
		return
	}
	var request struct {
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	user, err := s.store.webUserByID(id)
	if err != nil {
		writeAdminUserStoreError(w, err)
		return
	}
	if request.Password == "" {
		writeError(w, http.StatusBadRequest, errWebUserPasswordEmpty.Error())
		return
	}
	if err := systemValidatePass(user.Username, "", request.Password); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.store.setWebUserPassword(id, request.Password); err != nil {
		writeAdminUserStoreError(w, err)
		return
	}
	s.invalidateWebUserSessions(id)
	_ = s.store.recordAuditEvent(actor, "reset_webui_user_password", user.Username, true, "")
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *server) adminDeleteUser(w http.ResponseWriter, r *http.Request) {
	actor, ok := s.requireAdministrator(w, r)
	if !ok {
		return
	}
	id, ok := adminPathUserID(w, r)
	if !ok {
		return
	}
	user, err := s.store.webUserByID(id)
	if err != nil {
		writeAdminUserStoreError(w, err)
		return
	}
	if err := s.store.deleteWebUser(id); err != nil {
		writeAdminUserStoreError(w, err)
		return
	}
	s.invalidateWebUserSessions(id)
	_ = s.store.recordAuditEvent(actor, "delete_webui_user", user.Username, true, "role="+string(user.Role))
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *server) adminUserUsage(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdministrator(w, r); !ok {
		return
	}
	id, ok := adminPathUserID(w, r)
	if !ok {
		return
	}
	view, err := s.store.adminUserViewByID(id)
	if err != nil {
		writeAdminUserStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *server) adminResetRestartAllowance(w http.ResponseWriter, r *http.Request) {
	actor, ok := s.requireAdministrator(w, r)
	if !ok {
		return
	}
	id, ok := adminPathUserID(w, r)
	if !ok {
		return
	}
	if err := s.store.resetOperatorRestartAllowance(id, actor); err != nil {
		writeAdminUserStoreError(w, err)
		return
	}
	view, err := s.store.adminUserViewByID(id)
	if err != nil {
		writeAdminUserStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *server) adminResetBackupAllowance(w http.ResponseWriter, r *http.Request) {
	actor, ok := s.requireAdministrator(w, r)
	if !ok {
		return
	}
	id, ok := adminPathUserID(w, r)
	if !ok {
		return
	}
	if err := s.store.resetOperatorBackupAllowance(id, actor); err != nil {
		writeAdminUserStoreError(w, err)
		return
	}
	view, err := s.store.adminUserViewByID(id)
	if err != nil {
		writeAdminUserStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *server) invalidateWebUserSessions(userID int64) {
	if userID <= 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for token, sess := range s.sessions {
		if sess.AuthSource == authSourceWebUI && sess.WebUserID == userID {
			delete(s.sessions, token)
		}
	}
}

func adminPathUserID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid WebUI user id")
		return 0, false
	}
	return id, true
}

func writeAdminUserStoreError(w http.ResponseWriter, err error) {
	if errors.Is(err, errWebUserNotFound) || errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "WebUI user not found")
		return
	}
	writeError(w, http.StatusServiceUnavailable, "WebUI user data is unavailable")
}

func (s *webUIStore) listAdminUserViews() ([]adminUserView, error) {
	rows, err := s.db.Query(`SELECT
		u.id, u.username, u.role, u.enabled, u.created_at, u.updated_at, u.password_changed_at,
		o.restart_used, o.last_restart_at, o.backup_used, o.last_backup_at
		FROM web_users AS u
		JOIN operator_usage AS o ON o.user_id = u.id
		ORDER BY u.username COLLATE NOCASE, u.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	users := make([]adminUserView, 0)
	for rows.Next() {
		view, err := scanAdminUserView(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, view)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return users, nil
}

func (s *webUIStore) adminUserViewByID(id int64) (adminUserView, error) {
	return scanAdminUserView(s.db.QueryRow(`SELECT
		u.id, u.username, u.role, u.enabled, u.created_at, u.updated_at, u.password_changed_at,
		o.restart_used, o.last_restart_at, o.backup_used, o.last_backup_at
		FROM web_users AS u
		JOIN operator_usage AS o ON o.user_id = u.id
		WHERE u.id = ?`, id))
}

type adminUserScanner interface {
	Scan(dest ...any) error
}

func scanAdminUserView(scanner adminUserScanner) (adminUserView, error) {
	var view adminUserView
	var enabled int
	var lastRestart, lastBackup sql.NullString
	if err := scanner.Scan(
		&view.ID,
		&view.Username,
		&view.Role,
		&enabled,
		&view.CreatedAt,
		&view.UpdatedAt,
		&view.PasswordChangedAt,
		&view.RestartUsed,
		&lastRestart,
		&view.BackupUsed,
		&lastBackup,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return adminUserView{}, errWebUserNotFound
		}
		return adminUserView{}, err
	}
	view.Enabled = enabled == 1
	view.RestartLimit = operatorRestartLimit
	view.BackupLimit = operatorBackupLimit
	if lastRestart.Valid && lastRestart.String != "" {
		value := lastRestart.String
		view.LastRestartAt = &value
	}
	if lastBackup.Valid && lastBackup.String != "" {
		value := lastBackup.String
		view.LastBackupAt = &value
	}
	return view, nil
}
