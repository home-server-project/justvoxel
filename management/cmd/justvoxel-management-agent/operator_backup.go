package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	operatorBackupLimit    = 2
	operatorBackupCooldown = 15 * time.Minute
	manualBackupService    = "minecraft-backup.service"
)

var operatorBackupMu sync.Mutex

type operatorBackupAvailability struct {
	Allowed    bool
	Used       int
	Limit      int
	Reason     string
	RetryAfter time.Duration
}

type operatorBackupGuard struct {
	store  *webUIStore
	tx     *sql.Tx
	userID int64
}

type operatorBackupUsageView struct {
	BackupUsed      int `json:"backup_used"`
	BackupLimit     int `json:"backup_limit"`
	CooldownSeconds int `json:"cooldown_seconds"`
}

type manualBackupResponse struct {
	OK            bool                     `json:"ok"`
	Message       string                   `json:"message"`
	OperatorUsage *operatorBackupUsageView `json:"operator_usage,omitempty"`
}

var backupServiceState = func(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "systemctl", "show", "--property=ActiveState", "--value", manualBackupService)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

var requestManualBackup = func(ctx context.Context) error {
	return exec.CommandContext(ctx, "systemctl", "--no-block", "start", manualBackupService).Run()
}

func (s *server) manualBackup(w http.ResponseWriter, r *http.Request) {
	sess, ok := s.requireRoles(w, r, roleAdministrator, roleOperator)
	if !ok {
		return
	}

	operatorBackupMu.Lock()
	defer operatorBackupMu.Unlock()

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	state, err := backupServiceState(ctx)
	if err != nil {
		if s.store != nil {
			_ = s.store.recordAuditEvent(sess, "manual_backup", manualBackupService, false, "backup service state unavailable")
		}
		writeError(w, http.StatusServiceUnavailable, "backup service is unavailable")
		return
	}
	if state == "active" || state == "activating" || state == "deactivating" {
		if s.store != nil {
			_ = s.store.recordAuditEvent(sess, "manual_backup", manualBackupService, false, "backup already running")
		}
		writeJSON(w, http.StatusConflict, map[string]any{
			"error":  "backup already running",
			"reason": "backup_in_progress",
		})
		return
	}

	if sess.Role == roleAdministrator {
		if err := requestManualBackup(ctx); err != nil {
			if s.store != nil {
				_ = s.store.recordAuditEvent(sess, "manual_backup", manualBackupService, false, "systemd rejected manual backup request")
			}
			writeError(w, http.StatusServiceUnavailable, "manual backup could not be started")
			return
		}
		if s.store != nil {
			_ = s.store.recordAuditEvent(sess, "manual_backup", manualBackupService, true, "administrator manual backup accepted")
		}
		writeJSON(w, http.StatusAccepted, manualBackupResponse{OK: true, Message: "Manual backup requested."})
		return
	}

	if s.store == nil {
		writeError(w, http.StatusForbidden, "operator backup controls are unavailable")
		return
	}
	guard, availability, err := s.store.beginOperatorBackup(sess.WebUserID)
	if err != nil {
		_ = s.store.recordAuditEvent(sess, "manual_backup", manualBackupService, false, "operator backup eligibility check failed")
		writeError(w, http.StatusServiceUnavailable, "operator backup allowance is unavailable")
		return
	}
	if !availability.Allowed {
		_ = s.store.recordAuditEvent(sess, "manual_backup", manualBackupService, false, availability.Reason)
		switch availability.Reason {
		case "backup_limit_reached":
			writeJSON(w, http.StatusConflict, map[string]any{
				"error":                        "backup allowance exhausted",
				"reason":                       availability.Reason,
				"backup_used":                  availability.Used,
				"backup_limit":                 availability.Limit,
				"administrator_reset_required": true,
			})
		case "backup_cooldown":
			seconds := int(math.Ceil(availability.RetryAfter.Seconds()))
			if seconds < 1 {
				seconds = 1
			}
			w.Header().Set("Retry-After", strconv.Itoa(seconds))
			writeJSON(w, http.StatusTooManyRequests, map[string]any{
				"error":               "backup cooldown active",
				"reason":              availability.Reason,
				"backup_used":         availability.Used,
				"backup_limit":        availability.Limit,
				"retry_after_seconds": seconds,
			})
		default:
			writeError(w, http.StatusConflict, "operator backup is not currently available")
		}
		return
	}

	if err := requestManualBackup(ctx); err != nil {
		guard.cancel()
		_ = s.store.recordAuditEvent(sess, "manual_backup", manualBackupService, false, "systemd rejected manual backup request")
		writeError(w, http.StatusServiceUnavailable, "manual backup could not be started")
		return
	}
	used, err := guard.accept(sess)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "Manual backup was requested, but backup allowance accounting failed")
		return
	}
	writeJSON(w, http.StatusAccepted, manualBackupResponse{
		OK:      true,
		Message: "Manual backup requested.",
		OperatorUsage: &operatorBackupUsageView{
			BackupUsed:      used,
			BackupLimit:     operatorBackupLimit,
			CooldownSeconds: int(operatorBackupCooldown.Seconds()),
		},
	})
}

func (s *webUIStore) beginOperatorBackup(userID int64) (*operatorBackupGuard, operatorBackupAvailability, error) {
	availability := operatorBackupAvailability{Limit: operatorBackupLimit}
	if s == nil || s.db == nil || userID <= 0 {
		return nil, availability, errOperatorUnavailable
	}

	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return nil, availability, fmt.Errorf("begin operator backup check: %w", err)
	}
	rollback := func(err error) (*operatorBackupGuard, operatorBackupAvailability, error) {
		_ = tx.Rollback()
		return nil, availability, err
	}

	var role webUserRole
	var enabled int
	var lastPersonal sql.NullString
	if err := tx.QueryRow(`SELECT u.role, u.enabled, o.backup_used, o.last_backup_at
		FROM web_users AS u
		JOIN operator_usage AS o ON o.user_id = u.id
		WHERE u.id = ?`, userID).Scan(&role, &enabled, &availability.Used, &lastPersonal); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return rollback(errOperatorUnavailable)
		}
		return rollback(fmt.Errorf("read operator backup allowance: %w", err))
	}
	if role != webUserRoleOperator || enabled != 1 {
		return rollback(errOperatorUnavailable)
	}
	if availability.Used >= operatorBackupLimit {
		availability.Reason = "backup_limit_reached"
		_ = tx.Rollback()
		return nil, availability, nil
	}

	var lastGlobal sql.NullString
	if err := tx.QueryRow(`SELECT last_backup_at FROM operator_global_state WHERE id = 1`).Scan(&lastGlobal); err != nil {
		return rollback(fmt.Errorf("read global operator backup state: %w", err))
	}

	now := s.now().UTC()
	nextAllowed := time.Time{}
	for _, raw := range []sql.NullString{lastPersonal, lastGlobal} {
		if !raw.Valid || raw.String == "" {
			continue
		}
		parsed, err := parseWebUIStoreTime(raw.String)
		if err != nil {
			return rollback(err)
		}
		candidate := parsed.Add(operatorBackupCooldown)
		if candidate.After(nextAllowed) {
			nextAllowed = candidate
		}
	}
	if !nextAllowed.IsZero() && now.Before(nextAllowed) {
		availability.Reason = "backup_cooldown"
		availability.RetryAfter = nextAllowed.Sub(now)
		_ = tx.Rollback()
		return nil, availability, nil
	}

	availability.Allowed = true
	return &operatorBackupGuard{store: s, tx: tx, userID: userID}, availability, nil
}

func (g *operatorBackupGuard) cancel() {
	if g == nil || g.tx == nil {
		return
	}
	_ = g.tx.Rollback()
	g.tx = nil
}

func (g *operatorBackupGuard) accept(actor session) (int, error) {
	if g == nil || g.tx == nil || g.store == nil {
		return 0, errors.New("operator backup guard is not active")
	}
	now := g.store.now().UTC()
	nowText := formatWebUIStoreTime(now)
	if _, err := g.tx.Exec(`UPDATE operator_usage
		SET backup_used = backup_used + 1, last_backup_at = ?, updated_at = ?
		WHERE user_id = ?`, nowText, nowText, g.userID); err != nil {
		g.cancel()
		return 0, fmt.Errorf("consume operator backup allowance: %w", err)
	}
	var used int
	if err := g.tx.QueryRow(`SELECT backup_used FROM operator_usage WHERE user_id = ?`, g.userID).Scan(&used); err != nil {
		g.cancel()
		return 0, fmt.Errorf("read consumed operator backup allowance: %w", err)
	}
	if _, err := g.tx.Exec(`UPDATE operator_global_state
		SET last_backup_at = ?, updated_at = ? WHERE id = 1`, nowText, nowText); err != nil {
		g.cancel()
		return 0, fmt.Errorf("update global operator backup state: %w", err)
	}
	if err := insertAuditEventTx(g.tx, now, actor, "manual_backup", manualBackupService, true, fmt.Sprintf("operator backup allowance %d/%d", used, operatorBackupLimit)); err != nil {
		g.cancel()
		return 0, err
	}
	if used == operatorBackupLimit {
		title := "Operator backup limit reached"
		message := fmt.Sprintf("%s has used %d of %d manual backup actions. Administrator reset is required before another Operator backup.", actor.Username, used, operatorBackupLimit)
		if _, err := g.tx.Exec(`INSERT INTO notifications(created_at, kind, user_id, title, message)
			VALUES(?, 'operator_backup_limit', ?, ?, ?)`, nowText, g.userID, title, message); err != nil {
			g.cancel()
			return 0, fmt.Errorf("create operator backup-limit notification: %w", err)
		}
		if err := insertAuditEventTx(g.tx, now, actor, "operator_backup_limit_reached", actor.Username, true, fmt.Sprintf("backup allowance %d/%d", used, operatorBackupLimit)); err != nil {
			g.cancel()
			return 0, err
		}
	}
	if err := g.tx.Commit(); err != nil {
		g.tx = nil
		return 0, fmt.Errorf("commit operator backup accounting: %w", err)
	}
	g.tx = nil
	return used, nil
}

func (s *webUIStore) resetOperatorBackupAllowance(userID int64, actor session) error {
	if actor.Role != roleAdministrator {
		return errors.New("administrator role is required")
	}
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("begin backup allowance reset: %w", err)
	}
	now := s.now().UTC()
	nowText := formatWebUIStoreTime(now)
	var username string
	if err := tx.QueryRow(`SELECT username FROM web_users WHERE id = ?`, userID).Scan(&username); err != nil {
		_ = tx.Rollback()
		if errors.Is(err, sql.ErrNoRows) {
			return errWebUserNotFound
		}
		return fmt.Errorf("read operator for backup allowance reset: %w", err)
	}
	result, err := tx.Exec(`UPDATE operator_usage SET backup_used = 0, updated_at = ? WHERE user_id = ?`, nowText, userID)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("reset operator backup allowance: %w", err)
	}
	if count, err := result.RowsAffected(); err != nil || count != 1 {
		_ = tx.Rollback()
		if err != nil {
			return fmt.Errorf("read backup allowance reset result: %w", err)
		}
		return errWebUserNotFound
	}
	if _, err := tx.Exec(`UPDATE notifications
		SET resolved_at = ?, resolved_by = ?
		WHERE kind = 'operator_backup_limit' AND user_id = ? AND resolved_at IS NULL`, nowText, actor.Username, userID); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("resolve backup-limit notification: %w", err)
	}
	if err := insertAuditEventTx(tx, now, actor, "reset_backup_allowance", username, true, "operator backup allowance reset to 0/2"); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit backup allowance reset: %w", err)
	}
	return nil
}

func (s *webUIStore) operatorBackupUsage(userID int64) (int, *time.Time, error) {
	var used int
	var last sql.NullString
	if err := s.db.QueryRow(`SELECT backup_used, last_backup_at FROM operator_usage WHERE user_id = ?`, userID).Scan(&used, &last); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil, errWebUserNotFound
		}
		return 0, nil, fmt.Errorf("read operator backup usage: %w", err)
	}
	if !last.Valid || last.String == "" {
		return used, nil, nil
	}
	parsed, err := parseWebUIStoreTime(last.String)
	if err != nil {
		return 0, nil, err
	}
	return used, &parsed, nil
}
