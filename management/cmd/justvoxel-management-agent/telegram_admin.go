package main

import (
	"context"
	"encoding/hex"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var telegramTokenPattern = regexp.MustCompile(`^[0-9]{5,20}:[A-Za-z0-9_-]{30,128}$`)

func registerTelegramRoutes(mux *http.ServeMux, s *server) {
	mux.HandleFunc("GET /v1/admin/telegram", s.telegramAdmin)
	mux.HandleFunc("POST /v1/admin/telegram", s.telegramAdmin)
}
func (s *server) telegramAdmin(w http.ResponseWriter, r *http.Request) {
	actor, ok := s.requirePrimaryAdministrator(w, r)
	if !ok {
		return
	}
	// Pairing and credentials require an authenticated human WebUI session.
	if actor.AuthSource != authSourceSystem && actor.AuthSource != authSourceSeparate {
		writeError(w, 403, "primary administrator WebUI session required")
		return
	}
	s.telegramMu.Lock()
	defer s.telegramMu.Unlock()
	var input struct {
		Action        string `json:"action"`
		Token         string `json:"token,omitempty"`
		Enabled       bool   `json:"enabled,omitempty"`
		UserID        int64  `json:"user_id,omitempty"`
		Permission    string `json:"permission,omitempty"`
		Notifications bool   `json:"notifications,omitempty"`
	}
	if r.Method == "POST" && !decodeTelegramJSON(w, r, &input) {
		return
	}
	v, err := s.store.telegramRead()
	if err != nil {
		// Discarding only Telegram-owned credentials/approvals is a safe,
		// explicit disconnect even when its private record is malformed.
		if r.Method != "POST" || input.Action != "disconnect" {
			writeError(w, 503, "Telegram settings unavailable. Use Disconnect completely to remove Telegram-only state.")
			return
		}
		v = telegramState{Schema: 1, Generation: s.store.now().UnixNano()}
	}
	if r.Method == "GET" {
		writeJSON(w, 200, telegramPublicState(v, s.store.now()))
		return
	}
	if !s.telegramResetAllowed() {
		writeError(w, 409, "Complete pending Factory Reset recovery first.")
		return
	}
	now := s.store.now()
	challenge := ""
	switch input.Action {
	case "credential":
		if !telegramTokenPattern.MatchString(input.Token) {
			writeError(w, 400, "Enter a valid bot token.")
			return
		}
		// Replacing a credential disconnects all identities from the old bot.
		v = telegramState{Schema: 1, Token: input.Token, Generation: v.Generation + 1, Status: "unverified", Users: []telegramUser{}, Callbacks: map[string]telegramCallback{}, Seen: map[string]string{}}
	case "verify":
		if v.Token == "" {
			writeError(w, 400, "Configure a bot token first.")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		var bot struct {
			ID       int64  `json:"id"`
			Username string `json:"username"`
			IsBot    bool   `json:"is_bot"`
		}
		var webhook struct {
			URL string `json:"url"`
		}
		if telegramRequest(ctx, v.Token, "getMe", struct{}{}, &bot) != nil || !bot.IsBot || bot.ID <= 0 || telegramRequest(ctx, v.Token, "getWebhookInfo", struct{}{}, &webhook) != nil {
			writeError(w, 503, "Bot verification failed. Check the token and connection.")
			return
		}
		if webhook.URL != "" {
			writeError(w, 409, "This bot has a webhook. Use a dedicated bot without another update consumer.")
			return
		}
		v.BotID, v.BotName, v.Verified, v.Status = bot.ID, bot.Username, true, "verified"
	case "enabled":
		if input.Enabled && (!v.Verified || v.Token == "") {
			writeError(w, 409, "Verify the bot before enabling Telegram.")
			return
		}
		v.Enabled = input.Enabled
		v.invalidate()
		if v.Enabled {
			v.Status = "connecting"
		} else {
			v.Status = "disabled"
		}
	case "disconnect":
		v = telegramState{Schema: 1, Generation: v.Generation + 1, Status: "disconnected", Users: []telegramUser{}}
	case "pair":
		if !v.Enabled || !v.Verified {
			writeError(w, 409, "Enable a verified bot before pairing.")
			return
		}
		challenge, err = gatewayRandom("")
		if err != nil {
			writeError(w, 503, "Pairing unavailable")
			return
		}
		v.Pair = &telegramPair{Digest: hex.EncodeToString(gatewayDigest(challenge)), Expires: now.Add(10 * time.Minute)}
	case "approve":
		if v.Pair == nil || !now.Before(v.Pair.Expires) || v.Pair.UserID <= 0 || v.Pair.UserID != input.UserID || !telegramPermission(input.Permission) {
			writeError(w, 409, "Review a current pairing request and permission before approval.")
			return
		}
		if _, found := v.user(input.UserID); found {
			writeError(w, 409, "User is already approved.")
			return
		}
		if len(v.Users) >= 64 {
			writeError(w, 409, "Telegram user limit reached.")
			return
		}
		v.Users = append(v.Users, telegramUser{ID: input.UserID, Permission: input.Permission, Notifications: input.Notifications})
		v.invalidate()
	case "user", "revoke":
		if !telegramPermission(input.Permission) && input.Action == "user" {
			writeError(w, 400, "Choose Read Only or Server Management.")
			return
		}
		found := false
		users := []telegramUser{}
		for _, user := range v.Users {
			if user.ID == input.UserID {
				found = true
				if input.Action == "revoke" {
					continue
				}
				user.Permission, user.Notifications = input.Permission, input.Notifications
			}
			users = append(users, user)
		}
		if !found {
			writeError(w, 404, "Approved user not found.")
			return
		}
		v.Users = users
		v.invalidate()
	default:
		writeError(w, 400, "Unknown Telegram settings action.")
		return
	}
	if err := s.store.telegramSave(v, &actor, input.Action); err != nil {
		writeError(w, 503, "Telegram settings could not be saved.")
		return
	}
	if s.telegramPollCancel != nil {
		s.telegramPollCancel()
	}
	result := telegramPublicState(v, now)
	if challenge != "" {
		result["pairing_challenge"] = challenge
		result["pairing_command"] = "/start " + challenge
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, result)
}

func telegramClaimPair(v *telegramState, id int64, text string, now time.Time) bool {
	parts := strings.Fields(text)
	if id <= 0 || len(parts) != 2 || parts[0] != "/start" || v.Pair == nil || !now.Before(v.Pair.Expires) || v.Pair.UserID != 0 {
		return false
	}
	if hex.EncodeToString(gatewayDigest(parts[1])) != v.Pair.Digest {
		return false
	}
	v.Pair.UserID = id
	// The challenge is consumed, but the numeric identity still needs WebUI approval.
	v.Pair.Digest = ""
	return true
}

func decodeTelegramJSON(w http.ResponseWriter, r *http.Request, out any) bool {
	if decodeContainerJSON(r.Body, out) != nil {
		writeError(w, 400, "Invalid Telegram request.")
		return false
	}
	return true
}
