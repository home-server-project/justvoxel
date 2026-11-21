package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/home-server-project/justvoxel-webui/internal/api"
)

type routerProxyFixture struct {
	*fakeAPI
	username string
	role     string
	received string
}

func (f *routerProxyFixture) Session(context.Context, string) (api.SessionInfo, error) {
	return api.SessionInfo{Role: f.role, Username: f.username}, nil
}

func (f *routerProxyFixture) RouterOSStatus(_ context.Context, _, id string) (api.RouterOSStatus, error) {
	f.received = "status/" + id
	return api.RouterOSStatus{Status: "not_configured"}, nil
}

func (f *routerProxyFixture) RouterOSTest(_ context.Context, _ string, input api.RouterOSCredentials) (api.RouterOSStatus, error) {
	f.received = "test/" + input.Endpoint
	return api.RouterOSStatus{Status: "connected"}, nil
}

func (f *routerProxyFixture) RouterOSSave(_ context.Context, _ string, input api.RouterOSConfiguration) error {
	f.received = "save/" + input.Ingress.Name
	return nil
}

func (f *routerProxyFixture) RouterOSForget(context.Context, string) error {
	f.received = "forget"
	return nil
}

func TestRouterOSWebAuthorizationCSRFAndProxy(t *testing.T) {
	for _, tc := range []struct {
		method, suffix, actor, role, csrf, forwarded string
		want                                         int
	}{
		{"GET", "/instances/jv-family1", "admin-voxel", "administrator", "", "status/jv-family1", 200},
		{"GET", "?port=22", "admin-voxel", "administrator", "", "", 400},
		{"GET", "", "viewer", "viewer", "", "", 403},
		{"POST", "/test", "admin-voxel", "administrator", "csrf", "", 403},
		{"PUT", "", "admin-voxel", "administrator", "csrf", "", 403},
		{"DELETE", "", "admin-voxel", "administrator", "csrf", "", 403},
		{"POST", "/test", "voxel", "administrator", "", "", 403},
		{"PUT", "", "voxel", "administrator", "wrong", "", 403},
		{"DELETE", "", "voxel", "administrator", "", "", 403},
		{"POST", "/test", "voxel", "administrator", "csrf", "test/https://router.example", 200},
		{"PUT", "", "voxel", "administrator", "csrf", "save/uplinks", 200},
		{"DELETE", "", "voxel", "administrator", "csrf", "forget", 200},
	} {
		t.Run(tc.method+"/"+tc.actor+"/"+tc.suffix+"/"+tc.csrf, func(t *testing.T) {
			fixture := &routerProxyFixture{fakeAPI: &fakeAPI{}, username: tc.actor, role: tc.role}
			app, err := New(fixture, Config{ExternalScheme: "http"})
			if err != nil {
				t.Fatal(err)
			}
			form := url.Values{}
			if tc.method == "POST" || tc.method == "PUT" {
				form.Set("endpoint", "https://router.example")
				form.Set("username", "router-user")
				form.Set("password", "private-router-password")
			}
			if tc.method == "PUT" {
				form.Set("target", "192.0.2.10")
				form.Set("ingress_type", "list")
				form.Set("ingress_name", "uplinks")
			}
			request := httptest.NewRequest(tc.method, routerOSWebPath+tc.suffix, strings.NewReader(form.Encode()))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			request.Header.Set("X-CSRF-Token", tc.csrf)
			request.AddCookie(&http.Cookie{Name: sessionCookie, Value: "session-token"})
			request.AddCookie(&http.Cookie{Name: csrfCookie, Value: "csrf"})
			response := httptest.NewRecorder()
			app.Handler().ServeHTTP(response, request)
			if response.Code != tc.want || fixture.received != tc.forwarded || strings.Contains(response.Body.String(), "private-router-password") {
				t.Errorf("status %d, proxied %s", response.Code, fixture.received)
			}
		})
	}
}

func TestPortForwardingDialogUsesOnlySelectedImmutableID(t *testing.T) {
	source, err := assets.ReadFile("static/minecraft-servers.js")
	if err != nil {
		t.Fatal(err)
	}
	js := string(source)
	start, end := strings.Index(js, "  const showPortForwarding ="), strings.Index(js, "  const key =")
	if start < 0 || end <= start {
		t.Fatal("missing port forwarding dialog")
	}
	dialog := js[start:end]
	for _, want := range []string{
		"'/api/network/router/mikrotik/instances/' + encodeURIComponent(id)",
		`data.network_member) paragraph(jvT("network.router.managed_by_network"))`,
		`data.endpoints || []`,
		`/?workspace=network&tab=Router`,
		`className = 'minecraft-server-invitation'`,
	} {
		if !strings.Contains(dialog, want) {
			t.Errorf("dialog missing %q", want)
		}
	}
	if regexp.MustCompile(`createElement\('(input|form|select|textarea)'\)|\b(POST|PUT|PATCH|DELETE)\b|\d+px\b`).MatchString(dialog) ||
		strings.Count(dialog, "createElement('button')") != 1 {
		t.Fatal("port forwarding dialog must contain only display and navigation")
	}
	if !strings.Contains(js, "const id = card.dataset.serverId;") ||
		!strings.Contains(js, "showPortForwarding(card, id);") {
		t.Fatal("port forwarding dialog did not use the selected card ID")
	}
}
