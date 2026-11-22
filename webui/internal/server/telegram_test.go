package server

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type telegramTestAPI struct {
	filesTestAPI
	calls int
	last  json.RawMessage
}

func (f *telegramTestAPI) TelegramAdmin(_ context.Context, _ string, body json.RawMessage) (json.RawMessage, error) {
	f.calls++
	f.last = append(json.RawMessage{}, body...)
	return json.RawMessage(`{"configured":true,"enabled":false,"verified":false,"status":"unverified","users":[]}`), nil
}
func TestTelegramWorkspacePrimaryAdministratorBoundary(t *testing.T) {
	for _, tc := range []struct {
		role, name string
		allowed    bool
	}{{"viewer", "Viewer", false}, {"operator", "Operator", false}, {"administrator", "other", false}, {"administrator", "voxel", true}} {
		f := &telegramTestAPI{filesTestAPI: filesTestAPI{role: tc.role, username: tc.name}}
		a, err := New(f, Config{})
		if err != nil {
			t.Fatal(err)
		}
		rr := httptest.NewRecorder()
		a.Handler().ServeHTTP(rr, authenticatedSystemActionRequest("GET", "/api/telegram", ""))
		if (rr.Code == 200) != tc.allowed || (!tc.allowed && f.calls > 0) {
			t.Fatal("Telegram primary boundary", tc.role, tc.name, rr.Code)
		}
	}
}
func TestTelegramWorkspaceCSRFAndTokenSubmission(t *testing.T) {
	f := &telegramTestAPI{filesTestAPI: filesTestAPI{role: "administrator", username: "voxel"}}
	a, err := New(f, Config{})
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	a.Handler().ServeHTTP(rr, authenticatedSystemActionRequest("POST", "/api/telegram", "request=%7B%7D"))
	if rr.Code != 403 || f.calls != 0 {
		t.Fatal("CSRF bypass", rr.Code)
	}
	token := "123456:" + strings.Repeat("a", 40)
	request, _ := json.Marshal(map[string]any{"action": "credential", "token": token})
	body := url.Values{"csrf": {"token"}, "request": {string(request)}}.Encode()
	rr = httptest.NewRecorder()
	a.Handler().ServeHTTP(rr, authenticatedSystemActionRequest("POST", "/api/telegram", body))
	if rr.Code != 200 || f.calls != 1 || !strings.Contains(string(f.last), token) || strings.Contains(rr.Body.String(), token) || rr.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("credential handling", rr.Code)
	}
	rr = httptest.NewRecorder()
	a.Handler().ServeHTTP(rr, externalGatewayRequest("POST", "/api/telegram", `{"action":"pair"}`))
	if rr.Code == 200 || f.calls != 1 {
		t.Fatal("external credential authenticated Telegram administration")
	}
}
