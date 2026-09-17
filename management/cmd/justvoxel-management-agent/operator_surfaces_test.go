package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func surfaceTestServer(t *testing.T, role principalRole) *server {
	t.Helper()
	store, _ := openTestWebUIStore(t)
	now := time.Now()
	return &server{store: store, sessions: map[string]session{
		"token": {Username: "tester", Role: role, AuthSource: authSourceWebUI, WebUserID: 1, Created: now, LastSeen: now},
	}}
}

func surfaceRequest(method, path, body string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer token")
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

func TestViewerCannotAccessWhitelistOrRawMinecraftLogs(t *testing.T) {
	s := surfaceTestServer(t, roleViewer)

	rr := httptest.NewRecorder()
	s.whitelistList(rr, surfaceRequest(http.MethodGet, "/v1/whitelist", ""))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("viewer whitelist status = %d, want 403", rr.Code)
	}

	rr = httptest.NewRecorder()
	s.minecraftLogs(rr, surfaceRequest(http.MethodGet, "/v1/logs/minecraft", ""))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("viewer logs status = %d, want 403", rr.Code)
	}
}

func TestOperatorWhitelistUsesOnlyFixedHelperActions(t *testing.T) {
	s := surfaceTestServer(t, roleOperator)
	old := runWhitelistHelper
	defer func() { runWhitelistHelper = old }()
	var got []string
	runWhitelistHelper = func(_ context.Context, args ...string) ([]byte, error) {
		got = append([]string(nil), args...)
		return []byte("Added Alex\n"), nil
	}

	rr := httptest.NewRecorder()
	s.whitelistChange(rr, surfaceRequest(http.MethodPost, "/v1/whitelist", `{"platform":"java","action":"add","name":"Alex"}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("operator whitelist status = %d: %s", rr.Code, rr.Body.String())
	}
	if len(got) != 2 || got[0] != "add-java" || got[1] != "Alex" {
		t.Fatalf("helper args = %#v", got)
	}
}

func TestViewerGetsSanitizedActivityOnly(t *testing.T) {
	store, _ := openTestWebUIStore(t)
	now := time.Now()
	s := &server{store: store, sessions: map[string]session{
		"token": {Username: "viewer", Role: roleViewer, AuthSource: authSourceWebUI, WebUserID: 1, Created: now, LastSeen: now},
	}}
	actor := session{Username: "operator-one", Role: roleOperator, AuthSource: authSourceWebUI, WebUserID: 7}
	if err := store.recordAuditEvent(actor, "minecraft_restart", "minecraft.service", false, "restart_cooldown private context"); err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	s.publicActivity(rr, surfaceRequest(http.MethodGet, "/v1/activity", ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("activity status = %d: %s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, "minecraft_restart") {
		t.Fatalf("activity missing action: %s", body)
	}
	for _, secret := range []string{"operator-one", "minecraft.service", "private context"} {
		if strings.Contains(body, secret) {
			t.Fatalf("sanitized activity leaked %q: %s", secret, body)
		}
	}
}
