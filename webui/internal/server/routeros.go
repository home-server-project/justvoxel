package server

import (
	"context"
	"errors"
	"mime"
	"net/http"
	"net/url"

	"github.com/home-server-project/justvoxel-webui/internal/api"
)

const routerOSWebPath = "/api/network/router/mikrotik"

type routerOSAPI interface {
	RouterOSStatus(context.Context, string, string) (api.RouterOSStatus, error)
	RouterOSTest(context.Context, string, api.RouterOSCredentials) (api.RouterOSStatus, error)
	RouterOSSave(context.Context, string, api.RouterOSConfiguration) error
	RouterOSForget(context.Context, string) error
}

func (a *App) registerRouterOSRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET "+routerOSWebPath, a.routerOS)
	mux.HandleFunc("GET "+routerOSWebPath+"/instances/{id}", a.routerOS)
	mux.HandleFunc("POST "+routerOSWebPath+"/test", a.routerOS)
	mux.HandleFunc("PUT "+routerOSWebPath, a.routerOS)
	mux.HandleFunc("DELETE "+routerOSWebPath, a.routerOS)
}

func (a *App) routerOS(w http.ResponseWriter, r *http.Request) {
	session, identity, ok := a.systemMonitorIdentity(w, r)
	if !ok {
		return
	}
	if identity.Role != "administrator" || (r.Method != http.MethodGet && identity.Username != "voxel") {
		writeNetworkWebError(w, 403, ui(w, "ui.031cc7a3ec82"))
		return
	}
	client, ok := a.api.(routerOSAPI)
	if !ok {
		writeNetworkWebError(w, 503, ui(w, "network.router.unavailable"))
		return
	}
	if r.URL.ForceQuery || r.URL.RawQuery != "" {
		writeNetworkWebError(w, 400, ui(w, "network.router.invalid_request"))
		return
	}
	var out api.RouterOSStatus
	var err error
	if r.Method == http.MethodGet {
		out, err = client.RouterOSStatus(r.Context(), session, r.PathValue("id"))
	} else {
		r.Form = url.Values{"csrf": {r.Header.Get("X-CSRF-Token")}}
		if !a.validCSRF(r) {
			writeNetworkWebError(w, 403, ui(w, "ui.74bccc44f472"))
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 8192)
		if r.Method == http.MethodDelete {
			if r.ContentLength != 0 {
				writeNetworkWebError(w, 400, ui(w, "network.router.invalid_request"))
				return
			}
		} else {
			media, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if media != "application/x-www-form-urlencoded" || r.ParseForm() != nil {
				writeNetworkWebError(w, 400, ui(w, "network.router.invalid_request"))
				return
			}
		}
		for key, values := range r.PostForm {
			allowed := false
			if r.Method != http.MethodDelete {
				allowed = key == "endpoint" || key == "username" || key == "password"
			}
			if r.Method == http.MethodPut {
				allowed = allowed || key == "target" || key == "ingress_type" || key == "ingress_name"
			}
			if !allowed || len(values) != 1 {
				writeNetworkWebError(w, 400, ui(w, "network.router.invalid_request"))
				return
			}
		}
		credentials := api.RouterOSCredentials{
			Endpoint: r.PostForm.Get("endpoint"), Username: r.PostForm.Get("username"), Password: r.PostForm.Get("password"),
		}
		switch r.Method {
		case http.MethodPost:
			out, err = client.RouterOSTest(r.Context(), session, credentials)
		case http.MethodPut:
			err = client.RouterOSSave(r.Context(), session, api.RouterOSConfiguration{
				RouterOSCredentials: credentials, Target: r.PostForm.Get("target"),
				Ingress: api.RouterOSSelector{Type: r.PostForm.Get("ingress_type"), Name: r.PostForm.Get("ingress_name")},
			})
		case http.MethodDelete:
			err = client.RouterOSForget(r.Context(), session)
		}
	}
	if err != nil {
		a.writeRouterOSError(w, err)
		return
	}
	if r.Method == http.MethodPut || r.Method == http.MethodDelete {
		writeNetworkWebJSON(w, 200, map[string]bool{"ok": true})
		return
	}
	writeNetworkWebJSON(w, 200, out)
}

func (a *App) writeRouterOSError(w http.ResponseWriter, err error) {
	if errors.Is(err, api.ErrUnauthorized) || errors.Is(err, api.ErrPasswordChangeRequired) {
		a.handleNetworkAPIError(w, err)
		return
	}
	status, key := http.StatusBadGateway, "network.router.unavailable"
	var response *api.ResponseError
	if errors.As(err, &response) {
		switch response.Message {
		case "invalid_request":
			status, key = 400, "network.router.invalid_request"
		case "invalid_credentials", "invalid_configuration":
			status, key = 400, "network.router.invalid_configuration"
		case "target_unavailable":
			status, key = response.StatusCode, "network.router.target_unavailable"
		case "ingress_unavailable":
			status, key = 400, "network.router.ingress_unavailable"
		case "inventory_error":
			status, key = response.StatusCode, "network.router.inventory_error"
		case "configuration_error":
			status, key = 503, "network.router.configuration_error"
		case "factory_reset_pending":
			status, key = 409, "network.router.factory_reset_pending"
		}
	}
	writeNetworkWebError(w, status, ui(w, key))
}
