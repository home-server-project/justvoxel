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
	caPEM    string
	err      error
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
	f.caPEM = input.CAPEM
	return api.RouterOSStatus{Status: "connected"}, f.err
}

func (f *routerProxyFixture) RouterOSSave(_ context.Context, _ string, input api.RouterOSConfiguration) error {
	f.received = "save/" + input.Ingress.Name
	f.caPEM = input.CAPEM
	return nil
}

func (f *routerProxyFixture) RouterOSForwarding(_ context.Context, _ string, input api.RouterForwardingRequest) (api.RouterOSStatus, error) {
	f.received = input.Action + "/" + input.Scope
	return api.RouterOSStatus{}, f.err
}

func (f *routerProxyFixture) RouterOSForget(context.Context, string) error {
	f.received = "forget"
	return nil
}

func (f *routerProxyFixture) RouterOSPreviewCA(_ context.Context, _ string, caPEM string) (api.RouterOSCAMetadata, error) {
	f.received, f.caPEM = "preview-ca", caPEM
	return api.RouterOSCAMetadata{}, f.err
}

func (f *routerProxyFixture) RouterOSReplaceCA(_ context.Context, _ string, caPEM string) error {
	f.received = "replace-ca"
	f.caPEM = caPEM
	return f.err
}

func (f *routerProxyFixture) RouterOSRemoveCA(context.Context, string) error {
	f.received = "remove-ca"
	return f.err
}

func TestRouterOSWebAuthorizationCSRFAndProxy(t *testing.T) {
	for _, tc := range []struct {
		method, suffix, actor, role, csrf, forwarded string
		want                                         int
	}{
		{"POST", "/forwarding", "voxel", "administrator", "csrf", "enable/jv-family1", 200},
		{"POST", "/forwarding", "voxel", "administrator", "", "", 403},
		{"POST", "/forwarding", "admin-voxel", "administrator", "csrf", "", 403},
		{"POST", "/forwarding", "voxel", "administrator", "csrf", "", 400},
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
		{"POST", "/ca", "voxel", "administrator", "csrf", "preview-ca", 200},
		{"POST", "/ca", "voxel", "administrator", "", "", 403},
		{"PUT", "/ca", "admin-voxel", "administrator", "csrf", "", 403},
		{"DELETE", "/ca", "admin-voxel", "administrator", "csrf", "", 403},
		{"PUT", "/ca", "voxel", "administrator", "", "", 403},
		{"DELETE", "/ca", "voxel", "administrator", "wrong", "", 403},
		{"PUT", "/ca", "voxel", "administrator", "csrf", "replace-ca", 200},
		{"DELETE", "/ca", "voxel", "administrator", "csrf", "remove-ca", 200},
	} {
		t.Run(tc.method+"/"+tc.actor+"/"+tc.suffix+"/"+tc.csrf, func(t *testing.T) {
			fixture := &routerProxyFixture{fakeAPI: &fakeAPI{}, username: tc.actor, role: tc.role}
			app, err := New(fixture, Config{ExternalScheme: "http"})
			if err != nil {
				t.Fatal(err)
			}
			form := url.Values{}
			const pendingCA = "-----BEGIN CERTIFICATE-----\npublic-ca-file\n-----END CERTIFICATE-----\n"
			if tc.method == "POST" || tc.method == "PUT" {
				form.Set("ca_pem", pendingCA)
				if tc.suffix != "/ca" {
					form.Set("endpoint", "https://router.example")
					form.Set("username", "router-user")
					form.Set("password", "private-router-password")
				}
			}
			if tc.method == "PUT" && tc.suffix != "/ca" {
				form.Set("target", "192.0.2.10")
				form.Set("ingress_type", "list")
				form.Set("ingress_name", "uplinks")
			}
			if tc.suffix == "/forwarding" {
				form = url.Values{"scope": {"jv-family1"}, "action": {"enable"}}
				if tc.want == 400 {
					form.Set("target", "192.0.2.1")
				}
			}
			request := httptest.NewRequest(tc.method, routerOSWebPath+tc.suffix, strings.NewReader(form.Encode()))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			request.Header.Set("X-CSRF-Token", tc.csrf)
			request.AddCookie(&http.Cookie{Name: sessionCookie, Value: "session-token"})
			request.AddCookie(&http.Cookie{Name: csrfCookie, Value: "csrf"})
			response := httptest.NewRecorder()
			app.Handler().ServeHTTP(response, request)
			if response.Code != tc.want || fixture.received != tc.forwarded || strings.Contains(response.Body.String(), "private-router-password") ||
				strings.Contains(response.Body.String(), "BEGIN CERTIFICATE") || strings.Contains(response.Body.String(), "ca_pem") {
				t.Errorf("status %d, proxied %s", response.Code, fixture.received)
			}
			expectedPEM := ""
			if tc.suffix != "/forwarding" && tc.want == 200 && (tc.method == "POST" || tc.method == "PUT") {
				expectedPEM = pendingCA
			}
			if fixture.caPEM != expectedPEM {
				t.Error("unexpected CA forwarded to management API")
			}
		})
	}
}

func TestRouterOSWebConnectionDiagnostics(t *testing.T) {
	for _, tc := range []struct {
		code, key string
		status    int
		removing  bool
	}{
		{"tls_error", "tls_error", 502, false},
		{"tls_error", "ca_remove_tls_error", 502, true},
		{"auth_error", "auth_error", 502, false},
		{"network_error", "network_error", 502, false},
		{"response_error", "response_error", 502, false},
		{"invalid_ca", "invalid_ca", 400, false},
		{"sensitive-remote-content", "unavailable", 502, false},
	} {
		t.Run(tc.key, func(t *testing.T) {
			response := httptest.NewRecorder()
			w := &localeWriter{ResponseWriter: response, locale: "fr-CA"}
			(&App{}).writeRouterOSError(w, &api.ResponseError{StatusCode: tc.status, Message: tc.code}, tc.removing)
			if response.Code != tc.status || !strings.Contains(response.Body.String(), translate("fr-CA", "network.router."+tc.key)) {
				t.Fatalf("unexpected localized diagnostic: status %d, %s", response.Code, response.Body.String())
			}
			if strings.Contains(response.Body.String(), "sensitive-remote-content") {
				t.Fatal("raw diagnostic was exposed")
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
	if regexp.MustCompile(`createElement\('(input|form|select|textarea)'\)|\b(PUT|PATCH|DELETE)\b|innerHTML|\d+px\b`).MatchString(dialog) ||
		strings.Count(dialog, "createElement('button')") != 2 || !strings.Contains(dialog, "new URLSearchParams({scope: data.forwarding_scope, action})") {
		t.Fatal("unexpected port forwarding dialog control or request")
	}
	if !strings.Contains(js, "initializeCard(card, root, () => arranging, showPortForwarding);") ||
		!strings.Contains(js, "function initializeCard(card, root, isArranging, showPortForwarding) {") {
		t.Fatal("initializeCard must receive the port forwarding dialog launcher")
	}
	if !strings.Contains(js, "const id = card.dataset.serverId;") ||
		!strings.Contains(js, "showPortForwarding(card, id);") {
		t.Fatal("port forwarding dialog did not use the selected card ID")
	}
}
