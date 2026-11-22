package server

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/home-server-project/justvoxel-webui/internal/api"
	"net/http"
)

type telegramAdminAPI interface {
	TelegramAdmin(context.Context, string, json.RawMessage) (json.RawMessage, error)
}

func (a *App) registerTelegramRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/telegram", a.telegramAdmin)
	mux.HandleFunc("POST /api/telegram", a.telegramAdmin)
}
func (a *App) telegramAdmin(w http.ResponseWriter, r *http.Request) {
	session, _, ok := a.fileBrowserIdentity(w, r)
	if !ok {
		return
	}
	client, ok := a.api.(telegramAdminAPI)
	if !ok {
		gatewayJSON(w, 503, map[string]string{"error": ui(w, "ui.c2a0120d57f6")})
		return
	}
	var body json.RawMessage
	if r.Method == "POST" {
		r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
		if r.ParseForm() != nil {
			gatewayJSON(w, 400, map[string]string{"error": ui(w, "ui.af92524e4c08")})
			return
		}
		if !a.validCSRF(r) {
			gatewayJSON(w, 403, map[string]string{"error": ui(w, "ui.74bccc44f472")})
			return
		}
		body = json.RawMessage(r.FormValue("request"))
		if len(body) == 0 || !json.Valid(body) {
			gatewayJSON(w, 400, map[string]string{"error": ui(w, "ui.af92524e4c08")})
			return
		}
	}
	result, err := client.TelegramAdmin(r.Context(), session, body)
	if err != nil {
		message := ui(w, "ui.42851165fb3e")
		status := http.StatusBadGateway
		var response *api.ResponseError
		// This explicit settings endpoint returns fixed appliance messages, never
		// raw Telegram/HTTP diagnostics or the submitted credential.
		if errors.As(err, &response) && response.StatusCode >= 400 && response.StatusCode <= 599 {
			status = response.StatusCode
			if response.Message != "" {
				message = response.Message
			}
		}
		gatewayJSON(w, status, map[string]string{"error": message})
		return
	}
	gatewayJSON(w, 200, result)
}
