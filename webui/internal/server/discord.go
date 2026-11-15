package server

import (
	"context"
	"encoding/json"
	"net/http"
)

type discordAdminAPI interface {
	DiscordAdmin(context.Context, string, json.RawMessage) (json.RawMessage, error)
}

func (a *App) registerDiscordRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/discord", a.discordAdmin)
	mux.HandleFunc("POST /api/discord", a.discordAdmin)
}
func (a *App) discordAdmin(w http.ResponseWriter, r *http.Request) {
	session, _, ok := a.fileBrowserIdentity(w, r)
	if !ok {
		return
	}
	client, ok := a.api.(discordAdminAPI)
	if !ok {
		gatewayJSON(w, 503, map[string]string{"error": "Discord unavailable."})
		return
	}
	var body json.RawMessage
	if r.Method == "POST" {
		r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
		if r.ParseForm() != nil {
			gatewayJSON(w, 400, map[string]string{"error": "Invalid request."})
			return
		}
		if !a.validCSRF(r) {
			gatewayJSON(w, 403, map[string]string{"error": "Invalid CSRF token."})
			return
		}
		body = json.RawMessage(r.FormValue("request"))
		if len(body) == 0 || !json.Valid(body) {
			gatewayJSON(w, 400, map[string]string{"error": "Invalid request."})
			return
		}
	}
	result, err := client.DiscordAdmin(r.Context(), session, body)
	if err != nil {
		message := "Discord settings could not be loaded or saved. Check the bot, session, and any pending Factory Reset."
		status := http.StatusBadGateway
		gatewayJSON(w, status, map[string]string{"error": message})
		return
	}
	gatewayJSON(w, 200, result)
}
