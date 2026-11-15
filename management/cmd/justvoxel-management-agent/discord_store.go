package main

import (
	"encoding/json"
	"errors"
)

var errDiscordState = errors.New("Discord state unavailable")

// Private record in the existing root-owned identity DB. Never return this as
// an HTTP response; discordPublicState deliberately excludes the credential.
type discordState struct {
	Schema          int    `json:"schema"`
	Token           string `json:"token,omitempty"`
	BotID           string `json:"bot_id"`
	BotUsername     string `json:"bot_username"`
	BotName         string `json:"bot_name"`
	ApplicationID   string `json:"application_id"`
	ApplicationName string `json:"application_name"`
	Verified        bool   `json:"verified"`
	Status          string `json:"status"`
	Generation      int64  `json:"generation"`
}

func (s *webUIStore) discordRead() (discordState, error) {
	var v discordState
	var raw string
	if s == nil || s.db == nil {
		return v, errDiscordState
	}
	if s.db.QueryRow(`SELECT record_json FROM discord_state WHERE id=1`).Scan(&raw) != nil {
		return v, errDiscordState
	}
	if len(raw) > 16<<10 || json.Unmarshal([]byte(raw), &v) != nil || v.Schema != 1 {
		return discordState{}, errDiscordState
	}
	if v.Token != "" && !validDiscordToken(v.Token) {
		return discordState{}, errDiscordState
	}
	if v.Verified && (v.Token == "" || !validDiscordSnowflake(v.BotID) || !validDiscordSnowflake(v.ApplicationID) || v.BotUsername == "" || v.Status != "verified") {
		return discordState{}, errDiscordState
	}
	return v, nil
}

func (s *webUIStore) discordSave(v discordState, actor *session, action string) error {
	if s == nil || s.db == nil || v.Schema != 1 {
		return errDiscordState
	}
	raw, err := json.Marshal(v)
	if err != nil || len(raw) > 16<<10 {
		return errDiscordState
	}
	tx, err := s.db.Begin()
	if err != nil {
		return errDiscordState
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`INSERT INTO discord_state(id,record_json) VALUES(1,?) ON CONFLICT(id) DO UPDATE SET record_json=excluded.record_json`, string(raw)); err != nil {
		return errDiscordState
	}
	if actor != nil {
		if insertAuditEventTx(tx, s.now(), *actor, "discord_"+action, "discord", true, "") != nil {
			return errDiscordState
		}
	}
	if tx.Commit() != nil {
		return errDiscordState
	}
	return nil
}

func discordPublicState(v discordState) map[string]any {
	return map[string]any{"configured": v.Token != "", "verified": v.Verified, "status": v.Status, "bot_id": v.BotID, "bot_username": v.BotUsername, "bot_name": v.BotName, "application_id": v.ApplicationID, "application_name": v.ApplicationName}
}

func (s *server) discordFactoryRevoke() error {
	s.discordMu.Lock()
	defer s.discordMu.Unlock()
	// Do not parse state being revoked: a corrupt Discord record must remain
	// removable through supported disconnect and Factory Reset recovery.
	actor := session{Username: systemAdminUsername, Role: roleAdministrator}
	return s.store.discordSave(discordState{Schema: 1, Status: "disconnected"}, &actor, "factory_revocation")
}
