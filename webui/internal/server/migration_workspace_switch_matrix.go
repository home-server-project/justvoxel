package server

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/home-server-project/justvoxel-webui/internal/api"
)

type migrationSwitchMatrixAPI interface {
	SwitchMatrixStatus(context.Context, string, string) (api.SwitchMatrixStatus, error)
	StartSwitchMatrix(context.Context, string) (api.SwitchMatrixStatus, error)
	StopSwitchMatrix(context.Context, string, string) (api.SwitchMatrixStatus, error)
	RecoverSwitchMatrix(context.Context, string, string) (api.SwitchMatrixStatus, error)
}

func (a *App) migrationSwitchMatrix(w http.ResponseWriter, r *http.Request) {
	session, _, ok := a.minecraftWorkspaceIdentity(w, r, "administrator")
	if !ok {
		return
	}
	client, ok := a.api.(migrationSwitchMatrixAPI)
	if !ok {
		writeMinecraftWorkspaceError(w, http.StatusServiceUnavailable, ui(w, "switch_matrix.status_failed"))
		return
	}
	badRequest := func() {
		writeMinecraftWorkspaceError(w, http.StatusBadRequest, ui(w, "ui.35e44579c835"))
	}
	ctx := r.Context()
	var result api.SwitchMatrixStatus
	var err error
	targeted := false
	action := r.PathValue("matrixAction")
	if r.Method == http.MethodGet {
		id := ""
		if r.URL.RawQuery != "" {
			query, parseErr := url.ParseQuery(r.URL.RawQuery)
			values := query["operation_id"]
			if parseErr != nil || len(query) != 1 || len(values) != 1 || !validWorkspaceLogID("migration", values[0]) {
				badRequest()
				return
			}
			id = values[0]
		}
		result, err = client.SwitchMatrixStatus(ctx, session, id)
	} else {
		if r.URL.RawQuery != "" {
			badRequest()
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 4096)
		if r.ParseForm() != nil || !a.validCSRF(r) {
			writeMinecraftWorkspaceError(w, http.StatusForbidden, ui(w, "ui.ebdf0875cf9f"))
			return
		}
		for key, values := range r.PostForm {
			if key == "case_id" && action == "start" && len(values) > 0 && len(values) <= 16 {
				continue
			}
			if (key != "csrf" && (key != "operation_id" || action == "start")) || len(values) != 1 {
				badRequest()
				return
			}
		}
		id := r.PostForm.Get("operation_id")
		if action != "start" && !validWorkspaceLogID("migration", id) {
			badRequest()
			return
		}
		capability, statusErr := client.SwitchMatrixStatus(ctx, session, "")
		if statusErr != nil {
			a.writeSwitchMatrixAPIError(w, statusErr, "switch_matrix.status_failed")
			return
		}
		if !capability.Enabled {
			writeMinecraftWorkspaceError(w, http.StatusNotFound, ui(w, "ui.097074c74d5b"))
			return
		}
		switch action {
		case "start":
			ids := r.PostForm["case_id"]
			targeted = len(ids) != 0
			if targeted {
				selection, ok := a.api.(interface {
					StartTargetedSwitchMatrix(context.Context, string, []string) (api.SwitchMatrixStatus, error)
				})
				if !ok {
					http.NotFound(w, r)
					return
				}
				result, err = selection.StartTargetedSwitchMatrix(ctx, session, ids)
			} else {
				result, err = client.StartSwitchMatrix(ctx, session)
			}
		case "stop":
			result, err = client.StopSwitchMatrix(ctx, session, id)
		case "recover":
			_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(37 * time.Minute))
			result, err = client.RecoverSwitchMatrix(ctx, session, id)
		default:
			http.NotFound(w, r)
			return
		}
	}
	if err != nil {
		message := "switch_matrix.action_failed"
		if action == "" {
			message = "switch_matrix.status_failed"
		}
		var response *api.ResponseError
		if targeted && errors.As(err, &response) && response.StatusCode == http.StatusBadRequest {
			a.writeMinecraftWorkspaceAPIError(w, err, ui(w, message))
		} else {
			a.writeSwitchMatrixAPIError(w, err, message)
		}
		return
	}
	code := http.StatusOK
	if action == "start" || action == "stop" {
		code = http.StatusAccepted
	}
	writeMinecraftWorkspaceJSON(w, code, result)
}

func (a *App) writeSwitchMatrixAPIError(w http.ResponseWriter, err error, key string) {
	var response *api.ResponseError
	if errors.As(err, &response) {
		err = &api.ResponseError{StatusCode: response.StatusCode}
	}
	a.writeMinecraftWorkspaceAPIError(w, err, ui(w, key))
}
