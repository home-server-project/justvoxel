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

func operatorServerForStore(store *webUIStore, user webUser) *server {
	now := time.Now()
	return &server{
		store: store,
		sessions: map[string]session{
			"token": {
				Username:   user.Username,
				Role:       roleOperator,
				AuthSource: authSourceWebUI,
				WebUserID:  user.ID,
				Created:    now,
				LastSeen:   now,
			},
		},
	}
}

func TestOperatorRestartConsumesAllowanceAndCreatesNotification(t *testing.T) {
	store, _ := openTestWebUIStore(t)
	base := time.Now().UTC().Truncate(time.Second)
	store.now = func() time.Time { return base }
	user, err := store.createWebUser("Ilya", webUserRoleOperator, "operator-password")
	if err != nil {
		t.Fatal(err)
	}
	s := operatorServerForStore(store, user)

	calls := 0
	oldRunner := runWebHelper
	runWebHelper = func(_ context.Context, args ...string) ([]byte, int, error) {
		calls++
		if len(args) != 1 || args[0] != "restart" {
			t.Fatalf("unexpected helper args: %#v", args)
		}
		return []byte(`{"ok":true,"action":"restart","message":"Minecraft restart requested."}`), 0, nil
	}
	defer func() { runWebHelper = oldRunner }()

	rr := httptest.NewRecorder()
	s.minecraftRestart(rr, authorizedRequest(http.MethodPost, "/v1/minecraft/restart", `{}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("first restart returned %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"restart_used":1`) {
		t.Fatalf("first restart response missing usage: %s", rr.Body.String())
	}
	used, _, err := store.operatorRestartUsage(user.ID)
	if err != nil || used != 1 {
		t.Fatalf("usage after first restart = %d, err=%v", used, err)
	}

	rr = httptest.NewRecorder()
	s.minecraftRestart(rr, authorizedRequest(http.MethodPost, "/v1/minecraft/restart", `{}`))
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("cooldown restart returned %d: %s", rr.Code, rr.Body.String())
	}
	if calls != 1 {
		t.Fatalf("helper calls during cooldown = %d, want 1", calls)
	}

	store.now = func() time.Time { return base.Add(16 * time.Minute) }
	rr = httptest.NewRecorder()
	s.minecraftRestart(rr, authorizedRequest(http.MethodPost, "/v1/minecraft/restart", `{}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("second accepted restart returned %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"restart_used":2`) {
		t.Fatalf("second restart response missing 2/2 usage: %s", rr.Body.String())
	}
	if calls != 2 {
		t.Fatalf("accepted helper calls = %d, want 2", calls)
	}

	var openNotifications int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE kind = 'operator_restart_limit' AND user_id = ? AND resolved_at IS NULL`, user.ID).Scan(&openNotifications); err != nil {
		t.Fatal(err)
	}
	if openNotifications != 1 {
		t.Fatalf("open restart-limit notifications = %d, want 1", openNotifications)
	}

	store.now = func() time.Time { return base.Add(32 * time.Minute) }
	rr = httptest.NewRecorder()
	s.minecraftRestart(rr, authorizedRequest(http.MethodPost, "/v1/minecraft/restart", `{}`))
	if rr.Code != http.StatusConflict {
		t.Fatalf("exhausted restart returned %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"administrator_reset_required":true`) {
		t.Fatalf("exhausted response missing reset requirement: %s", rr.Body.String())
	}
	if calls != 2 {
		t.Fatalf("helper ran after allowance exhaustion: %d calls", calls)
	}
}

func TestOperatorRestartConfirmationDoesNotConsumeAllowance(t *testing.T) {
	store, _ := openTestWebUIStore(t)
	base := time.Now().UTC().Truncate(time.Second)
	store.now = func() time.Time { return base }
	user, err := store.createWebUser("Alex", webUserRoleOperator, "operator-password")
	if err != nil {
		t.Fatal(err)
	}
	s := operatorServerForStore(store, user)

	oldRunner := runWebHelper
	runWebHelper = func(_ context.Context, args ...string) ([]byte, int, error) {
		return []byte(`{"ok":false,"action":"restart","confirmation_required":true,"reason":"players_online","players":["Steve"],"online":1}`), 10, errors.New("confirmation required")
	}
	defer func() { runWebHelper = oldRunner }()

	rr := httptest.NewRecorder()
	s.minecraftRestart(rr, authorizedRequest(http.MethodPost, "/v1/minecraft/restart", `{}`))
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), `"confirmation_required":true`) {
		t.Fatalf("confirmation response = %d %s", rr.Code, rr.Body.String())
	}
	used, last, err := store.operatorRestartUsage(user.ID)
	if err != nil || used != 0 || last != nil {
		t.Fatalf("confirmation consumed allowance: used=%d last=%v err=%v", used, last, err)
	}
}

func TestOperatorRestartGlobalCooldownAppliesAcrossUsers(t *testing.T) {
	store, _ := openTestWebUIStore(t)
	base := time.Now().UTC().Truncate(time.Second)
	store.now = func() time.Time { return base }
	first, err := store.createWebUser("FirstOperator", webUserRoleOperator, "first-password")
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.createWebUser("SecondOperator", webUserRoleOperator, "second-password")
	if err != nil {
		t.Fatal(err)
	}
	firstServer := operatorServerForStore(store, first)
	secondServer := operatorServerForStore(store, second)

	calls := 0
	oldRunner := runWebHelper
	runWebHelper = func(_ context.Context, args ...string) ([]byte, int, error) {
		calls++
		return []byte(`{"ok":true,"action":"restart","message":"Minecraft restart requested."}`), 0, nil
	}
	defer func() { runWebHelper = oldRunner }()

	rr := httptest.NewRecorder()
	firstServer.minecraftRestart(rr, authorizedRequest(http.MethodPost, "/v1/minecraft/restart", `{}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("first operator restart returned %d: %s", rr.Code, rr.Body.String())
	}

	store.now = func() time.Time { return base.Add(5 * time.Minute) }
	rr = httptest.NewRecorder()
	secondServer.minecraftRestart(rr, authorizedRequest(http.MethodPost, "/v1/minecraft/restart", `{}`))
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("second operator bypassed global cooldown: %d %s", rr.Code, rr.Body.String())
	}
	if calls != 1 {
		t.Fatalf("helper calls = %d, want 1", calls)
	}
	used, _, err := store.operatorRestartUsage(second.ID)
	if err != nil || used != 0 {
		t.Fatalf("global cooldown consumed second user's allowance: used=%d err=%v", used, err)
	}
}

func TestAdministratorResetRestoresAllowanceButNotCooldown(t *testing.T) {
	store, _ := openTestWebUIStore(t)
	base := time.Now().UTC().Truncate(time.Second)
	store.now = func() time.Time { return base }
	user, err := store.createWebUser("Maya", webUserRoleOperator, "operator-password")
	if err != nil {
		t.Fatal(err)
	}
	operator := session{Username: user.Username, Role: roleOperator, AuthSource: authSourceWebUI, WebUserID: user.ID}
	admin := session{Username: systemAdminUsername, Role: roleAdministrator, AuthSource: authSourceSystem}

	guard, availability, err := store.beginOperatorRestart(user.ID)
	if err != nil || !availability.Allowed {
		t.Fatalf("first allowance unavailable: %#v err=%v", availability, err)
	}
	if _, err := guard.accept(operator); err != nil {
		t.Fatal(err)
	}
	store.now = func() time.Time { return base.Add(16 * time.Minute) }
	guard, availability, err = store.beginOperatorRestart(user.ID)
	if err != nil || !availability.Allowed {
		t.Fatalf("second allowance unavailable: %#v err=%v", availability, err)
	}
	if _, err := guard.accept(operator); err != nil {
		t.Fatal(err)
	}
	_, lastBefore, err := store.operatorRestartUsage(user.ID)
	if err != nil || lastBefore == nil {
		t.Fatalf("missing last restart before reset: %v %v", lastBefore, err)
	}

	store.now = func() time.Time { return base.Add(17 * time.Minute) }
	if err := store.resetOperatorRestartAllowance(user.ID, admin); err != nil {
		t.Fatal(err)
	}
	used, lastAfter, err := store.operatorRestartUsage(user.ID)
	if err != nil || used != 0 || lastAfter == nil || !lastAfter.Equal(*lastBefore) {
		t.Fatalf("reset state used=%d lastBefore=%v lastAfter=%v err=%v", used, lastBefore, lastAfter, err)
	}
	var openNotifications int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE kind = 'operator_restart_limit' AND user_id = ? AND resolved_at IS NULL`, user.ID).Scan(&openNotifications); err != nil {
		t.Fatal(err)
	}
	if openNotifications != 0 {
		t.Fatalf("restart-limit notification remained open after reset: %d", openNotifications)
	}

	guard, availability, err = store.beginOperatorRestart(user.ID)
	if guard != nil {
		guard.cancel()
	}
	if err != nil || availability.Reason != "restart_cooldown" {
		t.Fatalf("reset unexpectedly cleared cooldown: %#v err=%v", availability, err)
	}
}
