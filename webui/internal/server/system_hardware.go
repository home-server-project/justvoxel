package server

import (
	"context"
	"net/http"
	"strconv"

	"github.com/home-server-project/justvoxel-webui/internal/api"
)

type systemHardwareAPI interface {
	SystemHardwareStatus(context.Context, string) (api.SystemHardwareStatus, error)
	AdminSystemHardwareAction(context.Context, string, api.SystemHardwareRequest) (api.SystemHardwareStatus, error)
}

func (a *App) systemHardwareStatus(w http.ResponseWriter, r *http.Request) {
	session, _, ok := a.upsIdentity(w, r)
	if !ok {
		return
	}
	client, ok := a.api.(systemHardwareAPI)
	if !ok {
		http.Error(w, ui(w, "ui.03a58db108c0"), http.StatusServiceUnavailable)
		return
	}
	status, err := client.SystemHardwareStatus(r.Context(), session)
	if !a.handleUPSAPIError(w, r, err) {
		return
	}
	writeUPSWebJSON(w, http.StatusOK, status)
}
func (a *App) systemHardwareAction(w http.ResponseWriter, r *http.Request) {
	session, _, ok := a.systemActionsRequest(w, r, true)
	if !ok {
		return
	}
	client, ok := a.api.(systemHardwareAPI)
	if !ok {
		http.Error(w, ui(w, "ui.aa3a4a59018d"), http.StatusServiceUnavailable)
		return
	}
	request := api.SystemHardwareRequest{Action: r.FormValue("action")}
	switch request.Action {
	case "display-delay":
		delay, err := strconv.Atoi(r.FormValue("delay_seconds"))
		if err != nil || delay < 1 || delay > 999 {
			http.Error(w, ui(w, "ui.801a39b5de44"), http.StatusBadRequest)
			return
		}
		request.DelaySeconds = delay
	case "display-off", "display-on", "lid-enable", "lid-disable":
		if r.FormValue("delay_seconds") != "" {
			http.Error(w, ui(w, "ui.0e1c08a2f807"), http.StatusBadRequest)
			return
		}
	default:
		http.Error(w, ui(w, "ui.83a5c22fadc8"), http.StatusBadRequest)
		return
	}
	status, err := client.AdminSystemHardwareAction(r.Context(), session, request)
	if !a.handleUPSAPIError(w, r, err) {
		return
	}
	writeUPSWebJSON(w, http.StatusOK, status)
}
