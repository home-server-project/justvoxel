package main

import (
	"encoding/json"
	"errors"
	"time"
)

const telegramReadOnly = "read_only"
const telegramManagement = "server_management"

var errTelegramState = errors.New("Telegram state unavailable")

type telegramUser struct {
	ID            int64  `json:"id"`
	Permission    string `json:"permission"`
	Notifications bool   `json:"notifications"`
}
type telegramPair struct {
	Digest  string    `json:"digest"`
	Expires time.Time `json:"expires"`
	UserID  int64     `json:"user_id"`
}
type telegramCallback struct {
	UserID     int64           `json:"user_id"`
	Operation  string          `json:"operation"`
	InstanceID string          `json:"instance_id"`
	Body       json.RawMessage `json:"body,omitempty"`
	Expires    time.Time       `json:"expires"`
	Confirmed  bool            `json:"confirmed"`
}
type telegramReboot struct {
	Operation   string    `json:"operation,omitempty"`
	BootID      string    `json:"boot_id"`
	Expected    string    `json:"expected"`
	RequestedAt time.Time `json:"requested_at"`
	State       string    `json:"state"`
}

// This record is stored only in the existing root-owned mode-0600 identity DB.
// It must never be used as an HTTP response. telegramPublicState is the DTO.
type telegramState struct {
	Schema          int                         `json:"schema"`
	Token           string                      `json:"token,omitempty"`
	BotID           int64                       `json:"bot_id"`
	BotName         string                      `json:"bot_name"`
	Enabled         bool                        `json:"enabled"`
	Verified        bool                        `json:"verified"`
	Generation      int64                       `json:"generation"`
	Offset          int64                       `json:"offset"`
	Status          string                      `json:"status"`
	OperationStatus string                      `json:"operation_status,omitempty"`
	Users           []telegramUser              `json:"users"`
	Pair            *telegramPair               `json:"pair,omitempty"`
	Callbacks       map[string]telegramCallback `json:"callbacks,omitempty"`
	Seen            map[string]string           `json:"seen,omitempty"`
	Reboot          *telegramReboot             `json:"reboot,omitempty"`
}

func telegramPermission(p string) bool { return p == telegramReadOnly || p == telegramManagement }
func (v *telegramState) user(id int64) (telegramUser, bool) {
	if id <= 0 {
		return telegramUser{}, false
	}
	for _, user := range v.Users {
		if user.ID == id && telegramPermission(user.Permission) {
			return user, true
		}
	}
	return telegramUser{}, false
}
func (s *webUIStore) telegramRead() (telegramState, error) {
	var v telegramState
	var raw string
	if s == nil || s.db == nil {
		return v, errTelegramState
	}
	if err := s.db.QueryRow(`SELECT record_json FROM telegram_state WHERE id=1`).Scan(&raw); err != nil {
		return v, errTelegramState
	}
	if len(raw) > 2<<20 || json.Unmarshal([]byte(raw), &v) != nil || v.Schema != 1 {
		return v, errTelegramState
	}
	ids := map[int64]bool{}
	for _, user := range v.Users {
		if user.ID <= 0 || ids[user.ID] || !telegramPermission(user.Permission) {
			return v, errTelegramState
		}
		ids[user.ID] = true
	}
	if v.Callbacks == nil {
		v.Callbacks = map[string]telegramCallback{}
	}
	if v.Seen == nil {
		v.Seen = map[string]string{}
	}
	if v.Users == nil {
		v.Users = []telegramUser{}
	}
	return v, nil
}
func (s *webUIStore) telegramSave(v telegramState, actor *session, action string) error {
	if v.Schema != 1 {
		return errTelegramState
	}
	raw, err := json.Marshal(v)
	if err != nil || len(raw) > 2<<20 {
		return errTelegramState
	}
	tx, err := s.db.Begin()
	if err != nil {
		return errTelegramState
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`INSERT INTO telegram_state(id,record_json) VALUES(1,?) ON CONFLICT(id) DO UPDATE SET record_json=excluded.record_json`, string(raw)); err != nil {
		return errTelegramState
	}
	if actor != nil {
		if err = insertAuditEventTx(tx, s.now(), *actor, "telegram_"+action, "telegram", true, ""); err != nil {
			return errTelegramState
		}
	}
	return tx.Commit()
}
func telegramPublicState(v telegramState, now time.Time) map[string]any {
	out := map[string]any{"configured": v.Token != "", "enabled": v.Enabled, "verified": v.Verified, "bot_name": v.BotName, "status": v.Status, "operation_status": v.OperationStatus, "users": v.Users}
	if v.Pair != nil && now.Before(v.Pair.Expires) {
		out["pairing"] = map[string]any{"user_id": v.Pair.UserID, "expires": v.Pair.Expires}
	}
	return out
}
func (v *telegramState) invalidate() {
	v.Generation++
	v.Pair = nil
	v.Callbacks = map[string]telegramCallback{}
}
func (v *telegramState) consumeCallback(key string, userID int64, now time.Time) (telegramCallback, bool) {
	cb, found := v.Callbacks[key]
	if !found || cb.UserID != userID || !now.Before(cb.Expires) {
		return telegramCallback{}, false
	}
	delete(v.Callbacks, key)
	_, approved := v.user(userID)
	return cb, v.Enabled && approved && telegramAllowed(v, userID, cb.Operation)
}
func (v *telegramState) issueCallback(cb telegramCallback, now time.Time) (string, error) {
	for key, item := range v.Callbacks {
		if !now.Before(item.Expires) {
			delete(v.Callbacks, key)
		}
	}
	if len(v.Callbacks) >= 512 {
		return "", errors.New("Telegram menu capacity reached; retry shortly")
	}
	key, err := gatewayRandom("")
	if err != nil {
		return "", err
	}
	cb.Expires = now.Add(5 * time.Minute)
	v.Callbacks[key] = cb
	return key, nil
}

func (s *server) telegramFactoryRevoke() error {
	s.telegramExecutionMu.Lock()
	defer s.telegramExecutionMu.Unlock()
	s.telegramMu.Lock()
	defer s.telegramMu.Unlock()
	// Revocation never needs to trust or parse the credentials it removes. Keep
	// malformed Telegram-only state from stranding supported reset recovery.
	actor := session{Username: systemAdminUsername, Role: roleAdministrator}
	empty := telegramState{Schema: 1, Generation: s.store.now().UnixNano(), Status: "disconnected", Users: []telegramUser{}}
	if err := s.store.telegramSave(empty, &actor, "factory_revocation"); err != nil {
		return err
	}
	if s.telegramPollCancel != nil {
		s.telegramPollCancel()
	}
	return nil
}
