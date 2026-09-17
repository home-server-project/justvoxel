package main

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
)

type auditEventView struct {
	ID            int64         `json:"id"`
	OccurredAt    string        `json:"occurred_at"`
	ActorUsername string        `json:"actor_username"`
	ActorRole     principalRole `json:"actor_role"`
	Action        string        `json:"action"`
	Target        string        `json:"target,omitempty"`
	Success       bool          `json:"success"`
	Context       string        `json:"context,omitempty"`
}

type notificationView struct {
	ID         int64   `json:"id"`
	CreatedAt  string  `json:"created_at"`
	Kind       string  `json:"kind"`
	UserID     *int64  `json:"user_id,omitempty"`
	Title      string  `json:"title"`
	Message    string  `json:"message"`
	ResolvedAt *string `json:"resolved_at,omitempty"`
	ResolvedBy *string `json:"resolved_by,omitempty"`
}

func registerAdminActivityRoutes(mux *http.ServeMux, s *server) {
	mux.HandleFunc("GET /v1/admin/audit", s.adminAudit)
	mux.HandleFunc("GET /v1/admin/notifications", s.adminNotifications)
	mux.HandleFunc("POST /v1/admin/notifications/{id}/resolve", s.adminResolveNotification)
}

func (s *server) adminAudit(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdministrator(w, r); !ok {
		return
	}
	limit := boundedQueryLimit(r, 100, 200)
	events, err := s.store.listAuditEvents(limit)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "audit history is unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": events})
}

func (s *server) adminNotifications(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdministrator(w, r); !ok {
		return
	}
	limit := boundedQueryLimit(r, 50, 200)
	openOnly := r.URL.Query().Get("open") != "false"
	notifications, err := s.store.listNotifications(limit, openOnly)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "notifications are unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"notifications": notifications})
}

func (s *server) adminResolveNotification(w http.ResponseWriter, r *http.Request) {
	actor, ok := s.requireAdministrator(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid notification id")
		return
	}
	if err := s.store.resolveNotification(id, actor); err != nil {
		if errors.Is(err, errWebUserNotFound) {
			writeError(w, http.StatusNotFound, "notification not found")
			return
		}
		writeError(w, http.StatusServiceUnavailable, "notification could not be resolved")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func boundedQueryLimit(r *http.Request, fallback, maximum int) int {
	value, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || value <= 0 {
		return fallback
	}
	if value > maximum {
		return maximum
	}
	return value
}

func (s *webUIStore) listAuditEvents(limit int) ([]auditEventView, error) {
	rows, err := s.db.Query(`SELECT id, occurred_at, actor_username, actor_role, action, COALESCE(target, ''), success, context
		FROM audit_events ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("list audit events: %w", err)
	}
	defer rows.Close()
	events := make([]auditEventView, 0)
	for rows.Next() {
		var event auditEventView
		var success int
		if err := rows.Scan(&event.ID, &event.OccurredAt, &event.ActorUsername, &event.ActorRole, &event.Action, &event.Target, &success, &event.Context); err != nil {
			return nil, fmt.Errorf("scan audit event: %w", err)
		}
		event.Success = success == 1
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list audit events: %w", err)
	}
	return events, nil
}

func (s *webUIStore) listNotifications(limit int, openOnly bool) ([]notificationView, error) {
	query := `SELECT id, created_at, kind, user_id, title, message, resolved_at, resolved_by FROM notifications`
	if openOnly {
		query += ` WHERE resolved_at IS NULL`
	}
	query += ` ORDER BY id DESC LIMIT ?`
	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()
	items := make([]notificationView, 0)
	for rows.Next() {
		var item notificationView
		var userID sql.NullInt64
		var resolvedAt, resolvedBy sql.NullString
		if err := rows.Scan(&item.ID, &item.CreatedAt, &item.Kind, &userID, &item.Title, &item.Message, &resolvedAt, &resolvedBy); err != nil {
			return nil, fmt.Errorf("scan notification: %w", err)
		}
		if userID.Valid {
			value := userID.Int64
			item.UserID = &value
		}
		if resolvedAt.Valid {
			value := resolvedAt.String
			item.ResolvedAt = &value
		}
		if resolvedBy.Valid {
			value := resolvedBy.String
			item.ResolvedBy = &value
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}
	return items, nil
}

func (s *webUIStore) resolveNotification(id int64, actor session) error {
	if actor.Role != roleAdministrator {
		return errors.New("administrator role is required")
	}
	now := formatWebUIStoreTime(s.now())
	result, err := s.db.Exec(`UPDATE notifications SET resolved_at = ?, resolved_by = ? WHERE id = ? AND resolved_at IS NULL`, now, actor.Username, id)
	if err != nil {
		return fmt.Errorf("resolve notification: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read notification resolve result: %w", err)
	}
	if count == 0 {
		return errWebUserNotFound
	}
	return s.recordAuditEvent(actor, "resolve_notification", strconv.FormatInt(id, 10), true, "")
}
