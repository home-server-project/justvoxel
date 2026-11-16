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
		http.Error(w, "hardware status unavailable", http.StatusServiceUnavailable)
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
		http.Error(w, "hardware controls unavailable", http.StatusServiceUnavailable)
		return
	}
	request := api.SystemHardwareRequest{Action: r.FormValue("action")}
	switch request.Action {
	case "display-delay":
		delay, err := strconv.Atoi(r.FormValue("delay_seconds"))
		if err != nil || delay < 1 || delay > 999 {
			http.Error(w, "Delay must be 1–999 seconds", http.StatusBadRequest)
			return
		}
		request.DelaySeconds = delay
	case "display-off", "display-on", "lid-enable", "lid-disable":
		if r.FormValue("delay_seconds") != "" {
			http.Error(w, "unexpected delay", http.StatusBadRequest)
			return
		}
	default:
		http.Error(w, "unsupported hardware action", http.StatusBadRequest)
		return
	}
	status, err := client.AdminSystemHardwareAction(r.Context(), session, request)
	if !a.handleUPSAPIError(w, r, err) {
		return
	}
	writeUPSWebJSON(w, http.StatusOK, status)
}
