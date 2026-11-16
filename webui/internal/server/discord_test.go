package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/home-server-project/justvoxel-webui/internal/api"
)

type discordTestAPI struct {
	filesTestAPI
	calls int
	last  json.RawMessage
	err   error
}

func (f *discordTestAPI) DiscordAdmin(_ context.Context, _ string, body json.RawMessage) (json.RawMessage, error) {
	f.calls++
	f.last = append(json.RawMessage{}, body...)
	return json.RawMessage(`{"configured":true,"verified":false,"status":"unverified"}`), f.err
}
func TestDiscordWorkspacePrimaryAdministratorBoundary(t *testing.T) {
	for _, tc := range []struct {
		role, name string
		allowed    bool
	}{{"viewer", "Viewer", false}, {"operator", "Operator", false}, {"administrator", "other", false}, {"administrator", "voxel", true}} {
		f := &discordTestAPI{filesTestAPI: filesTestAPI{role: tc.role, username: tc.name}}
		a, err := New(f, Config{})
		if err != nil {
			t.Fatal(err)
		}
		rr := httptest.NewRecorder()
		a.Handler().ServeHTTP(rr, authenticatedSystemActionRequest("GET", "/api/discord", ""))
		if (rr.Code == 200) != tc.allowed || (!tc.allowed && f.calls > 0) {
			t.Fatal("Discord primary boundary", tc.role, tc.name, rr.Code)
		}
	}
}
func TestDiscordWorkspaceCSRFAndTokenSubmission(t *testing.T) {
	f := &discordTestAPI{filesTestAPI: filesTestAPI{role: "administrator", username: "voxel"}}
	a, err := New(f, Config{})
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	a.Handler().ServeHTTP(rr, authenticatedSystemActionRequest("POST", "/api/discord", "request=%7B%7D"))
	if rr.Code != 403 || f.calls != 0 {
		t.Fatal("CSRF bypass", rr.Code)
	}
	token := "discord-credential-" + strings.Repeat("a", 40)
	request, _ := json.Marshal(map[string]any{"action": "credential", "token": token})
	body := url.Values{"csrf": {"token"}, "request": {string(request)}}.Encode()
	rr = httptest.NewRecorder()
	a.Handler().ServeHTTP(rr, authenticatedSystemActionRequest("POST", "/api/discord", body))
	if rr.Code != 200 || f.calls != 1 || !strings.Contains(string(f.last), token) || strings.Contains(rr.Body.String(), token) || rr.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("credential handling", rr.Code)
	}
	rr = httptest.NewRecorder()
	a.Handler().ServeHTTP(rr, externalGatewayRequest("POST", "/api/discord", `{"action":"verify"}`))
	if rr.Code == 200 || f.calls != 1 {
		t.Fatal("external credential authenticated Discord administration")
	}
}

func TestDiscordWorkspaceDoesNotReturnTransportDiagnostics(t *testing.T) {
	for _, failure := range []error{errors.New("private transport detail submitted-token"), &api.ResponseError{StatusCode: 503, Message: "private upstream detail submitted-token"}} {
		f := &discordTestAPI{filesTestAPI: filesTestAPI{role: "administrator", username: "voxel"}, err: failure}
		a, err := New(f, Config{})
		if err != nil {
			t.Fatal(err)
		}
		rr := httptest.NewRecorder()
		a.Handler().ServeHTTP(rr, authenticatedSystemActionRequest("POST", "/api/discord", url.Values{"csrf": {"token"}, "request": {`{"action":"credential","token":"submitted-token"}`}}.Encode()))
		if rr.Code != 502 || strings.Contains(rr.Body.String(), "private") || strings.Contains(rr.Body.String(), "submitted-token") || rr.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("unsafe Discord error response", rr.Code)
		}
	}
}

func TestDiscordWorkspaceRejectsInvalidOrOversizedRequests(t *testing.T) {
	f := &discordTestAPI{filesTestAPI: filesTestAPI{role: "administrator", username: "voxel"}}
	a, err := New(f, Config{})
	if err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{
		url.Values{"csrf": {"wrong"}, "request": {`{"action":"verify"}`}}.Encode(),
		url.Values{"csrf": {"token"}, "request": {"invalid JSON"}}.Encode(),
		url.Values{"csrf": {"token"}, "request": {""}}.Encode(),
		url.Values{"csrf": {"token"}, "request": {strings.Repeat("x", 17<<10)}}.Encode(),
	} {
		rr := httptest.NewRecorder()
		a.Handler().ServeHTTP(rr, authenticatedSystemActionRequest("POST", "/api/discord", body))
		if rr.Code < 400 || f.calls != 0 {
			t.Fatal("invalid request reached Discord", rr.Code)
		}
	}
}

func TestDiscordApprovalActionsUseExistingPrimarySurface(t *testing.T) {
	for _, action := range []string{"admin_approve", "admin_reject", "admin_remove"} {
		f := &discordTestAPI{filesTestAPI: filesTestAPI{role: "administrator", username: "voxel"}}
		a, err := New(f, Config{})
		if err != nil {
			t.Fatal(err)
		}
		payload := `{"action":"` + action + `","user_id":"777"}`
		rr := httptest.NewRecorder()
		a.Handler().ServeHTTP(rr, authenticatedSystemActionRequest("POST", "/api/discord", url.Values{"csrf": {"token"}, "request": {payload}}.Encode()))
		if rr.Code != 200 || string(f.last) != payload {
			t.Fatal("approval action did not reach existing endpoint", action, rr.Code)
		}
	}
}

func TestDiscordAdministratorAccessUsesCompactExistingSurface(t *testing.T) {
	header, err := os.ReadFile("templates/header.html")
	if err != nil {
		t.Fatal(err)
	}
	markup := string(header)
	workspaceStart := strings.Index(markup, `<dialog class="workspace-window system-workspace-dialog" data-workspace-window="agent-gateway"`)
	if workspaceStart < 0 {
		t.Fatal("existing connections workspace missing")
	}
	workspaceEnd := strings.Index(markup[workspaceStart:], "</dialog>")
	if workspaceEnd < 0 {
		t.Fatal("connections workspace closing tag missing")
	}
	workspace := markup[workspaceStart : workspaceStart+workspaceEnd]
	if !strings.Contains(workspace, `<h2 id="agent-gateway-title">AI &amp; API Connections</h2>`) {
		t.Fatal("existing connections workspace title missing")
	}
	start := strings.Index(workspace, "<section data-gateway-discord")
	if start < 0 {
		t.Fatal("existing provider missing")
	}
	end := strings.Index(workspace[start:], "</section>")
	if end < 0 {
		t.Fatal("Discord surface closing tag missing")
	}
	surface := workspace[start : start+end]
	for _, marker := range []string{`class="integration-lane integration-header"`, "data-discord-pending", "data-discord-administrators", "data-discord-bind"} {
		if !strings.Contains(surface, marker) {
			t.Fatal("existing provider control missing", marker)
		}
	}
	for _, marker := range []string{`<h3>Discord administrators</h3>`, `<section class="panel">`, "Discord connection", "data-discord-enabled"} {
		if strings.Contains(surface, marker) {
			t.Fatal("obsolete Discord surface structure present", marker)
		}
	}
	if strings.Count(markup, "data-discord-enabled") != 1 {
		t.Fatal("Discord must have exactly one enable control")
	}
	navigationStart := strings.Index(workspace, `<nav class="minecraft-workspace-tabs gateway-navigation"`)
	if navigationStart < 0 {
		t.Fatal("gateway navigation missing")
	}
	navigationEnd := strings.Index(workspace[navigationStart:], "</nav>")
	if navigationEnd < 0 {
		t.Fatal("gateway navigation closing tag missing")
	}
	navigation := workspace[navigationStart : navigationStart+navigationEnd]
	if !strings.Contains(navigation, "data-discord-enabled") {
		t.Fatal("Discord enable control missing from gateway navigation")
	}
	telegramGlobal := strings.Index(navigation, "data-telegram-global")
	discordGlobal := strings.Index(navigation, "data-discord-global")
	if telegramGlobal < 0 || discordGlobal < 0 || discordGlobal <= telegramGlobal {
		t.Fatal("Discord global control must follow Telegram global control")
	}
	script, err := os.ReadFile("static/discord.js")
	if err != nil {
		t.Fatal(err)
	}
	for _, action := range []string{"admin_approve", "admin_reject", "admin_remove"} {
		if !strings.Contains(string(script), action) {
			t.Fatal("fixed action missing", action)
		}
	}
	if strings.Contains(string(script), "innerHTML") || strings.Contains(string(script), "createElement('style')") {
		t.Fatal("unsafe identity rendering or new styles")
	}
}
