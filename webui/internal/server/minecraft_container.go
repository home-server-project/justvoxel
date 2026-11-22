package server

import (
	"context"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/home-server-project/justvoxel-webui/internal/api"
)

type minecraftContainerAPI interface {
	MinecraftInstanceContainerStatus(context.Context, string, string) (api.MinecraftContainerStatus, error)
	MinecraftInstanceContainerUpdate(context.Context, string, string, api.MinecraftContainerUpdateRequest) (api.PersistentOperationResponse, error)
	MinecraftInstanceContainerOperation(context.Context, string, string, string) (api.PersistentOperationResponse, error)
	MinecraftInstanceContainerAcknowledge(context.Context, string, string, string) (api.PersistentOperationResponse, error)
}

func (a *App) minecraftContainerWorkspace(w http.ResponseWriter, r *http.Request) {
	session, _, ok := a.minecraftWorkspaceIdentity(w, r, "administrator")
	if !ok {
		return
	}
	selected, ok := a.selectedMinecraft(w, r, session)
	if !ok {
		return
	}
	client, ok := a.api.(minecraftContainerAPI)
	if !ok {
		writeMinecraftWorkspaceError(w, 503, ui(w, "ui.10e725f8d499"))
		return
	}
	query := r.URL.Query()
	query.Del("server")
	r.URL.RawQuery = query.Encode()
	if r.Method == http.MethodGet && r.URL.Path == "/api/version/container/status" {
		if len(query) != 0 {
			writeMinecraftWorkspaceError(w, 400, ui(w, "ui.b0a61f2de3f3"))
			return
		}
		out, err := client.MinecraftInstanceContainerStatus(r.Context(), session, selected.ID)
		if err != nil {
			a.writeMinecraftWorkspaceAPIError(w, err, ui(w, "ui.ab79f4caf391"))
			return
		}
		if out.InstanceID != selected.ID {
			writeMinecraftWorkspaceError(w, 503, ui(w, "ui.d2997af148a4"))
			return
		}
		writeMinecraftWorkspaceJSON(w, 200, struct {
			api.MinecraftContainerStatus
			ServerName string `json:"server_name"`
		}{out, selected.Name})
		return
	}
	var out api.PersistentOperationResponse
	var err error
	code := 200
	if r.Method == http.MethodGet {
		out, err = client.MinecraftInstanceContainerOperation(r.Context(), session, selected.ID, r.URL.Query().Get("id"))
	} else {
		r.Body = http.MaxBytesReader(w, r.Body, 4096)
		if r.ParseForm() != nil || len(query) != 0 {
			writeMinecraftWorkspaceError(w, 400, ui(w, "ui.b00521c2118b"))
			return
		}
		allowed := []string{"csrf", "plan_fingerprint", "restart_now", "backup_before_restart", "confirm_players", "restore_digest"}
		retention := r.URL.Path == "/api/version/container/retention"
		if retention {
			allowed = []string{"csrf", "keep_previous_images"}
		}
		acknowledge := r.URL.Path == "/api/version/container/acknowledge"
		if acknowledge {
			allowed = []string{"csrf", "operation_id"}
		}
		for key, values := range r.PostForm {
			if !slices.Contains(allowed, key) || len(values) != 1 {
				writeMinecraftWorkspaceError(w, 400, ui(w, "ui.b00521c2118b"))
				return
			}
		}
		for _, key := range []string{"restart_now", "backup_before_restart", "confirm_players"} {
			if value := r.FormValue(key); value != "" && value != "yes" && value != "no" {
				writeMinecraftWorkspaceError(w, 400, ui(w, "ui.3a71f134c8d4"))
				return
			}
		}
		if !a.validCSRF(r) {
			writeMinecraftWorkspaceError(w, 403, ui(w, "ui.74bccc44f472"))
			return
		}
		if retention {
			value := r.FormValue("keep_previous_images")
			if value == "" {
				value = "1"
			}
			if strings.Trim(value, "0123456789") != "" {
				writeMinecraftWorkspaceError(w, 400, ui(w, "ui.3343c4387296"))
				return
			}
			n, parseErr := strconv.Atoi(value)
			if parseErr != nil || n < 0 {
				writeMinecraftWorkspaceError(w, 400, ui(w, "ui.3343c4387296"))
				return
			}
			writer, ok := a.api.(interface {
				MinecraftInstanceContainerRetention(context.Context, string, string, int) error
			})
			if !ok {
				writeMinecraftWorkspaceError(w, 503, ui(w, "ui.e25c8609ecb8"))
				return
			}
			if err := writer.MinecraftInstanceContainerRetention(r.Context(), session, selected.ID, n); err != nil {
				a.writeMinecraftWorkspaceAPIError(w, err, ui(w, "ui.cb2baeb8a0df"))
				return
			}
			writeMinecraftWorkspaceJSON(w, 200, map[string]any{"ok": true, "keep_previous_images": n})
			return
		}

		if acknowledge {
			out, err = client.MinecraftInstanceContainerAcknowledge(r.Context(), session, selected.ID, r.FormValue("operation_id"))
		} else {
			out, err = client.MinecraftInstanceContainerUpdate(r.Context(), session, selected.ID, api.MinecraftContainerUpdateRequest{PlanFingerprint: r.FormValue("plan_fingerprint"), RestartNow: r.FormValue("restart_now") == "yes", BackupBeforeRestart: r.FormValue("backup_before_restart") == "yes", ConfirmPlayers: r.FormValue("confirm_players") == "yes", RestoreDigest: r.FormValue("restore_digest")})
			code = 202
		}
	}
	if err != nil {
		a.writeMinecraftWorkspaceAPIError(w, err, ui(w, "ui.ca1e194a59d9"))
		return
	}
	if out.Operation != nil && (out.Operation.InstanceID != selected.ID || out.Operation.OperationType != "minecraft_container_update") {
		writeMinecraftWorkspaceError(w, 404, ui(w, "ui.805e95ac8928"))
		return
	}
	writeMinecraftWorkspaceJSON(w, code, out)
}
