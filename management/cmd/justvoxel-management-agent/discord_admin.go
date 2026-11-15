package main

import (
	"context"
	"net/http"
	"strings"
	"time"
)

func registerDiscordRoutes(mux *http.ServeMux, s *server) {
	mux.HandleFunc("GET /v1/admin/discord", s.discordAdmin)
	mux.HandleFunc("POST /v1/admin/discord", s.discordAdmin)
}

func (s *server) discordAdmin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	actor, ok := s.requirePrimaryAdministrator(w, r)
	if !ok {
		return
	}
	if actor.AuthSource != authSourceSystem && actor.AuthSource != authSourceSeparate {
		writeError(w, 403, "primary administrator WebUI session required")
		return
	}
	var input struct {
		Action string `json:"action"`
		Token  string `json:"token,omitempty"`
	}
	if r.Method == http.MethodPost && decodeContainerJSON(r.Body, &input) != nil {
		writeError(w, 400, "Invalid Discord request.")
		return
	}
	s.discordMu.Lock()
	defer s.discordMu.Unlock()
	v, err := s.store.discordRead()
	if err != nil {
		if r.Method != http.MethodPost || input.Action != "disconnect" {
			writeError(w, 503, "Discord settings unavailable. Disconnect to remove Discord state.")
			return
		}
		v = discordState{Schema: 1}
	}
	if r.Method == http.MethodGet {
		writeJSON(w, 200, discordPublicState(v))
		return
	}
	if s.operations != nil {
		current, err := s.operations.currentFactoryReset()
		if err != nil || (current != nil && operationIsCurrent(current.State)) {
			writeError(w, 409, "Complete pending Factory Reset recovery first.")
			return
		}
	}
	switch input.Action {
	case "credential":
		if !validDiscordToken(input.Token) {
			writeError(w, 400, "Enter a valid bot token.")
			return
		}
		v = discordState{Schema: 1, Token: input.Token, Generation: v.Generation + 1, Status: "unverified"}
	case "verify":
		if v.Token == "" {
			writeError(w, 400, "Configure a bot token first.")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		var bot struct {
			ID         string `json:"id"`
			Username   string `json:"username"`
			GlobalName string `json:"global_name"`
			Bot        bool   `json:"bot"`
		}
		var app struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		if discordRequest(ctx, v.Token, "/users/@me", &bot) != nil || !bot.Bot || !validDiscordSnowflake(bot.ID) || strings.TrimSpace(bot.Username) == "" || discordRequest(ctx, v.Token, "/applications/@me", &app) != nil || !validDiscordSnowflake(app.ID) {
			writeError(w, 503, "Bot verification failed. Check the token and connection.")
			return
		}
		v.BotID, v.BotUsername, v.BotName = bot.ID, bot.Username, bot.GlobalName
		if v.BotName == "" {
			v.BotName = bot.Username
		}
		v.ApplicationID, v.ApplicationName = app.ID, app.Name
		v.Verified, v.Status = true, "verified"
	case "disconnect":
		v = discordState{Schema: 1, Generation: v.Generation + 1, Status: "disconnected"}
	default:
		writeError(w, 400, "Unknown Discord settings action.")
		return
	}
	if s.store.discordSave(v, &actor, input.Action) != nil {
		writeError(w, 503, "Discord settings could not be saved.")
		return
	}
	writeJSON(w, 200, discordPublicState(v))
}
