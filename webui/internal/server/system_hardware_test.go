package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/home-server-project/justvoxel-webui/internal/api"
)

type fakeHardwareAPI struct {
	fakeSystemActionsAPI
	request api.SystemHardwareRequest
	calls   int
}

func (f *fakeHardwareAPI) SystemHardwareStatus(context.Context, string) (api.SystemHardwareStatus, error) {
	return api.SystemHardwareStatus{HWS: true, DisplayAvailable: true, LidAvailable: true}, nil
}
func (f *fakeHardwareAPI) AdminSystemHardwareAction(_ context.Context, _ string, request api.SystemHardwareRequest) (api.SystemHardwareStatus, error) {
	f.request = request
	f.calls++
	return api.SystemHardwareStatus{HWS: true, LidIgnored: request.Action == "lid-enable"}, nil
}

func TestHardwareWebMutationAuthorizationCSRFAndValidation(t *testing.T) {
	for _, tc := range []struct {
		role, body string
		want       int
	}{
		{"operator", "csrf=token&action=display-off", 403},
		{"viewer", "csrf=token&action=lid-enable", 403},
		{"administrator", "action=display-on", 403},
		{"administrator", "csrf=token&action=display-delay&delay_seconds=10%3Breboot", 400},
		{"administrator", "csrf=token&action=display-delay&delay_seconds=0", 400},
		{"administrator", "csrf=token&action=display-delay&delay_seconds=3601", 400},
		{"administrator", "csrf=token&action=shell", 400},
		{"administrator", "csrf=token&action=display-off&delay_seconds=20", 400},
		{"administrator", "csrf=token&action=display-delay&delay_seconds=10", 200},
		{"administrator", "csrf=token&action=display-off", 200},
		{"administrator", "csrf=token&action=display-on", 200},
		{"administrator", "csrf=token&action=lid-enable", 200},
		{"administrator", "csrf=token&action=lid-disable", 200},
	} {
		client := &fakeHardwareAPI{fakeSystemActionsAPI: fakeSystemActionsAPI{role: tc.role}}
		app, err := New(client, Config{ManagementAPI: "v1"})
		if err != nil {
			t.Fatal(err)
		}
		rr := httptest.NewRecorder()
		app.Handler().ServeHTTP(rr, authenticatedSystemActionRequest(http.MethodPost, "http://example/api/system-actions/display", tc.body))
		if rr.Code != tc.want {
			t.Fatalf("%s %s: %d %s", tc.role, tc.body, rr.Code, rr.Body.String())
		}
		if tc.want != 200 && client.calls != 0 {
			t.Fatal("denied mutation reached backend")
		}
		if tc.want == 200 && client.calls != 1 {
			t.Fatal("mutation did not reach backend once")
		}
		if client.actionCalls != 0 {
			t.Fatal("hardware entered host shutdown path")
		}
	}
}

func TestHardwareStatusRemainsReadOnlyForViewers(t *testing.T) {
	client := &fakeHardwareAPI{fakeSystemActionsAPI: fakeSystemActionsAPI{role: "viewer"}}
	app, err := New(client, Config{ManagementAPI: "v1"})
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	app.Handler().ServeHTTP(rr, authenticatedSystemActionRequest(http.MethodGet, "http://example/api/system-hardware", ""))
	if rr.Code != 200 || !strings.Contains(rr.Body.String(), `"hws":true`) || client.calls != 0 {
		t.Fatalf("read-only status: %d %s", rr.Code, rr.Body.String())
	}
}

func TestHardwareDialogTemplateContract(t *testing.T) {
	header, err := assets.ReadFile("templates/header.html")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`data-display-open aria-label="Display" data-tooltip="Display" hidden`, `data-display-dialog`, `Display power`, `min="1" max="3600" step="1" value="10"`, `Turn OFF now`, `Turn ON now`, `data-display-cancel>Cancel`, `data-display-action="display-delay">Apply`, `Ignore laptop lid close`, `role="switch" data-lid-ignore`, `data-lid-option hidden`, `data-power-indicators hidden`, `/static/system-hardware.js`} {
		if !strings.Contains(string(header), want) {
			t.Fatalf("template missing %q", want)
		}
	}
	poweroff := strings.Index(string(header), `data-system-action="poweroff"`)
	display := strings.Index(string(header), `data-display-open`)
	indicators := strings.Index(string(header), `data-power-indicators`)
	logout := strings.Index(string(header), `class="nav-logout"`)
	if !(poweroff < display && display < indicators && indicators < logout) {
		t.Fatal("approved action order lost")
	}
}

func TestHardwareWebPasswordAndAuthenticationBoundaries(t *testing.T) {
	for _, passwordChange := range []bool{false, true} {
		client := &fakeHardwareAPI{}
		app, err := New(client, Config{ManagementAPI: "v1"})
		if err != nil {
			t.Fatal(err)
		}
		req := authenticatedSystemActionRequest(http.MethodPost, "http://example/api/system-actions/display", "csrf=token&action=display-off")
		if passwordChange {
			req.AddCookie(&http.Cookie{Name: mustChangeCookie, Value: "1"})
		} else {
			req.Header.Del("Cookie")
			req.AddCookie(&http.Cookie{Name: csrfCookie, Value: "token"})
		}
		rr := httptest.NewRecorder()
		app.systemHardwareAction(rr, req)
		want := http.StatusUnauthorized
		if passwordChange {
			want = http.StatusForbidden
		}
		if rr.Code != want || client.calls != 0 {
			t.Fatalf("passwordChange=%v: %d %s", passwordChange, rr.Code, rr.Body.String())
		}
	}
}
