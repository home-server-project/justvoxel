package main

import (
	"encoding/json"
	"errors"
	"net/url"
	"time"
)

var errDiscordState = errors.New("Discord state unavailable")

// Private record in the existing root-owned identity DB. Never return this as
// an HTTP response; discordPublicState deliberately excludes the credential.
type discordIdentity struct {
	GuildID     string    `json:"guild_id"`
	UserID      string    `json:"user_id"`
	Name        string    `json:"name"`
	Username    string    `json:"username"`
	RequestedAt time.Time `json:"requested_at"`
}

type discordState struct {
	Pending         []discordIdentity `json:"pending,omitempty"`
	Administrators  []discordIdentity `json:"administrators,omitempty"`
	Enabled         bool              `json:"enabled"`
	GuildID         string            `json:"guild_id"`
	GuildName       string            `json:"guild_name"`
	Schema          int               `json:"schema"`
	Token           string            `json:"token,omitempty"`
	BotID           string            `json:"bot_id"`
	BotUsername     string            `json:"bot_username"`
	BotName         string            `json:"bot_name"`
	ApplicationID   string            `json:"application_id"`
	ApplicationName string            `json:"application_name"`
	Verified        bool              `json:"verified"`
	Status          string            `json:"status"`
	Generation      int64             `json:"generation"`
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
	if len(raw) > 64<<10 || json.Unmarshal([]byte(raw), &v) != nil || v.Schema != 1 || !discordValidIdentities(v) {
		return discordState{}, errDiscordState
	}
	if v.Token != "" && !validDiscordToken(v.Token) {
		return discordState{}, errDiscordState
	}
	if v.Verified && (v.Token == "" || !validDiscordSnowflake(v.BotID) || !validDiscordSnowflake(v.ApplicationID) || v.BotUsername == "" || v.Status != "verified") {
		return discordState{}, errDiscordState
	}
	if (v.Enabled && !v.Verified) || (v.GuildID != "" && !validDiscordSnowflake(v.GuildID)) {
		return discordState{}, errDiscordState
	}
	return v, nil
}

func (s *webUIStore) discordSave(v discordState, actor *session, action string) error {
	if s == nil || s.db == nil || v.Schema != 1 || !discordValidIdentities(v) {
		return errDiscordState
	}
	raw, err := json.Marshal(v)
	if err != nil || len(raw) > 64<<10 {
		return errDiscordState
	}
	tx, err := s.db.Begin()
	if err != nil {
		return errDiscordState
	}
	defer tx.Rollback()
	// Change the credential/binding and its community trust records atomically.
	var previousRaw string
	var previous discordState
	readErr := tx.QueryRow(`SELECT record_json FROM discord_state WHERE id=1`).Scan(&previousRaw)
	validPrevious := readErr == nil && json.Unmarshal([]byte(previousRaw), &previous) == nil
	clear := action == "credential" || action == "disconnect" || action == "factory_revocation" || !validPrevious || previous.Token != v.Token || previous.GuildID != v.GuildID
	if clear {
		if _, err = tx.Exec(`DELETE FROM discord_access_requests`); err != nil {
			return errDiscordState
		}
	} else if action == "enabled" && previous.Generation != v.Generation {
		if err = discordAccessPrune(tx, s.now().Unix()); err != nil {
			return errDiscordState
		}
		// Stage 2 fences transient controls on enable/disable. Durable community
		// requests retain the same credential/guild trust across that fence.
		if _, err = tx.Exec(`UPDATE discord_access_requests SET discord_generation=? WHERE discord_generation=? AND guild_id=?`, v.Generation, previous.Generation, v.GuildID); err != nil {
			return errDiscordState
		}
	}
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
	status := "disabled"
	if v.Enabled {
		status = "connecting"
	}
	install := ""
	if v.Verified && validDiscordSnowflake(v.ApplicationID) {
		install = "https://discord.com/oauth2/authorize?" + url.Values{"client_id": {v.ApplicationID}, "scope": {"bot"}, "permissions": {"3072"}}.Encode()
	}
	return map[string]any{"pending": v.Pending, "administrators": v.Administrators, "enabled": v.Enabled, "runtime_status": status, "guilds_ready": false, "guild_id": v.GuildID, "guild_name": v.GuildName, "bound_guild_available": false, "guilds": []discordGuild{}, "install_url": install, "configured": v.Token != "", "verified": v.Verified, "status": v.Status, "bot_id": v.BotID, "bot_username": v.BotUsername, "bot_name": v.BotName, "application_id": v.ApplicationID, "application_name": v.ApplicationName}
}

func (s *server) discordFactoryRevoke() error {
	s.discordMu.Lock()
	defer s.discordMu.Unlock()
	// Do not parse state being revoked: a corrupt Discord record must remain
	// removable through supported disconnect and Factory Reset recovery.
	s.discordStopLocked()
	actor := session{Username: systemAdminUsername, Role: roleAdministrator}
	return s.store.discordSave(discordState{Schema: 1, Status: "disconnected"}, &actor, "factory_revocation")
}

func discordValidIdentities(v discordState) bool {
	if len(v.Pending) > 32 || len(v.Administrators) > 32 {
		return false
	}
	seen := map[string]bool{}
	for _, list := range [][]discordIdentity{v.Pending, v.Administrators} {
		for _, u := range list {
			if !validDiscordSnowflake(u.UserID) || u.GuildID != v.GuildID || !validDiscordSnowflake(u.GuildID) || seen[u.UserID] || u.Name != discordGuildName(u.Name) || u.Username != discordGuildName(u.Username) {
				return false
			}
			seen[u.UserID] = true
		}
	}
	return true
}
func (v discordState) administrator(id string) bool {
	for _, u := range v.Administrators {
		if u.UserID == id && u.GuildID == v.GuildID {
			return true
		}
	}
	return false
}
func (v *discordState) adminAction(action, id string) bool {
	if !validDiscordSnowflake(id) || v.GuildID == "" {
		return false
	}
	if action == "admin_remove" {
		for i, u := range v.Administrators {
			if u.UserID == id {
				v.Administrators = append(v.Administrators[:i], v.Administrators[i+1:]...)
				return true
			}
		}
		return false
	}
	for i, u := range v.Pending {
		if u.UserID == id {
			if action == "admin_approve" {
				if len(v.Administrators) >= 32 {
					return false
				}
				v.Administrators = append(v.Administrators, u)
			}
			v.Pending = append(v.Pending[:i], v.Pending[i+1:]...)
			return true
		}
	}
	return false
}
