package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func adminServerForTest() *server {
	now := time.Now()
	return &server{sessions: map[string]session{
		"token": {Created: now, LastSeen: now},
	}}
}

func authorizedRequest(method, target, body string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestPlayersRequiresAuthentication(t *testing.T) {
	s := adminServerForTest()
	called := false
	oldRunner := runWebHelper
	runWebHelper = func(context.Context, ...string) ([]byte, int, error) {
		called = true
		return nil, 0, nil
	}
	defer func() { runWebHelper = oldRunner }()

	rr := httptest.NewRecorder()
	s.players(rr, httptest.NewRequest(http.MethodGet, "http://unix/v1/players", nil))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
	if called {
		t.Fatal("helper executed for unauthenticated request")
	}
}

func TestMinecraftStartUsesAllowlistedAction(t *testing.T) {
	s := adminServerForTest()
	oldRunner := runWebHelper
	runWebHelper = func(_ context.Context, args ...string) ([]byte, int, error) {
		if len(args) != 1 || args[0] != "start" {
			t.Fatalf("unexpected helper args: %#v", args)
		}
		return []byte(`{"ok":true,"action":"start","message":"Minecraft start requested."}`), 0, nil
	}
	defer func() { runWebHelper = oldRunner }()

	rr := httptest.NewRecorder()
	s.minecraftStart(rr, authorizedRequest(http.MethodPost, "http://unix/v1/minecraft/start", `{}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestMinecraftRestartRequiresExplicitPlayerConfirmation(t *testing.T) {
	s := adminServerForTest()
	oldRunner := runWebHelper
	runWebHelper = func(_ context.Context, args ...string) ([]byte, int, error) {
		if len(args) != 1 || args[0] != "restart" {
			t.Fatalf("unexpected helper args: %#v", args)
		}
		return []byte(`{"ok":false,"action":"restart","confirmation_required":true,"reason":"players_online","players":["Alex","Steve"],"online":2}`), 10, errors.New("confirmation required")
	}
	defer func() { runWebHelper = oldRunner }()

	rr := httptest.NewRecorder()
	s.minecraftRestart(rr, authorizedRequest(http.MethodPost, "http://unix/v1/minecraft/restart", `{}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 confirmation response, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `"confirmation_required":true`) {
		t.Fatalf("missing confirmation response: %s", rr.Body.String())
	}
}

func TestMinecraftConfirmedRestartUsesOnlyConfirmationFlag(t *testing.T) {
	s := adminServerForTest()
	oldRunner := runWebHelper
	runWebHelper = func(_ context.Context, args ...string) ([]byte, int, error) {
		if len(args) != 2 || args[0] != "restart" || args[1] != "--confirm-players" {
			t.Fatalf("unexpected helper args: %#v", args)
		}
		return []byte(`{"ok":true,"action":"restart","message":"Minecraft restarted."}`), 0, nil
	}
	defer func() { runWebHelper = oldRunner }()

	rr := httptest.NewRecorder()
	s.minecraftRestart(rr, authorizedRequest(http.MethodPost, "http://unix/v1/minecraft/restart", `{"confirm_players":true}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestMinecraftPlayerStatusFailureFailsClosed(t *testing.T) {
	s := adminServerForTest()
	oldRunner := runWebHelper
	runWebHelper = func(_ context.Context, _ ...string) ([]byte, int, error) {
		return []byte(`{"ok":false,"reason":"player_status_unavailable","message":"Could not confirm player status through RCON."}`), 3, errors.New("player status unavailable")
	}
	defer func() { runWebHelper = oldRunner }()

	rr := httptest.NewRecorder()
	s.minecraftStop(rr, authorizedRequest(http.MethodPost, "http://unix/v1/minecraft/stop", `{}`))
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestMinecraftRoutesDoNotExposeArbitraryActions(t *testing.T) {
	s := adminServerForTest()
	mux := http.NewServeMux()
	registerMinecraftRoutes(mux, s)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, authorizedRequest(http.MethodPost, "http://unix/v1/minecraft/shell", `{}`))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unregistered action, got %d", rr.Code)
	}
}
