package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/disgoorg/disgo/gateway"
)

type discordFakeGateway struct {
	status  atomic.Int32
	closes  atomic.Int32
	handler gateway.EventHandlerFunc
	fail    bool
}

func (c *discordFakeGateway) Open(ctx context.Context) error {
	if c.fail {
		return errors.New("private diagnostic Authorization token session resume sequence")
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	c.status.Store(int32(gateway.StatusReady))
	c.handler(nil, gateway.EventTypeReady, 0, gateway.EventReady{})
	return nil
}
func (c *discordFakeGateway) Close(context.Context) {
	c.closes.Add(1)
	c.status.Store(int32(gateway.StatusDisconnected))
}
func (c *discordFakeGateway) Status() gateway.Status { return gateway.Status(c.status.Load()) }

func discordVerifiedState() discordState {
	return discordState{Schema: 1, Token: discordTestToken, Verified: true, Status: "verified", BotID: "123", BotUsername: "voxel", ApplicationID: "456", Generation: 10}
}
func discordSaveTestState(t *testing.T, s *server, v discordState) {
	t.Helper()
	if err := s.store.discordSave(v, nil, ""); err != nil {
		t.Fatal(err)
	}
}
func discordEventually(t *testing.T, predicate func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if predicate() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("Discord runtime did not reach expected state")
}
func discordLoopForTest(t *testing.T, s *server, fail bool) <-chan *discordFakeGateway {
	t.Helper()
	clients := make(chan *discordFakeGateway, 10)
	s.discordNewGateway = func(ctx context.Context, token string, handler gateway.EventHandlerFunc, _ func()) discordGatewayClient {
		if token != discordTestToken {
			t.Error("unexpected credential")
		}
		c := &discordFakeGateway{handler: handler, fail: fail}
		clients <- c
		// Interrupt the next retry delay so failure bounds need no wall-clock backoff.
		if fail {
			s.discordWakeLocked()
		}
		return c
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); s.discordLoop(ctx) }()
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("runtime did not stop")
		}
	})
	return clients
}
func discordNextClient(t *testing.T, clients <-chan *discordFakeGateway) *discordFakeGateway {
	t.Helper()
	select {
	case c := <-clients:
		return c
	case <-time.After(3 * time.Second):
		t.Fatal("runtime did not start")
		return nil
	}
}
func discordRuntimeFixture(t *testing.T) (*server, *discordRuntime) {
	t.Helper()
	s := discordTestServer(t)
	v := discordVerifiedState()
	v.Enabled = true
	discordSaveTestState(t, s, v)
	c := &discordFakeGateway{}
	c.status.Store(int32(gateway.StatusReady))
	rt := &discordRuntime{generation: v.Generation, status: "connected", guilds: map[string]discordGuild{}, client: c}
	s.discordRuntime = rt
	return s, rt
}
func discordGuildEvent(t *testing.T, s *server, rt *discordRuntime, raw string) {
	t.Helper()
	var event gateway.EventGuildCreate
	if err := json.Unmarshal([]byte(raw), &event); err != nil {
		t.Fatal(err)
	}
	s.discordEvent(rt, gateway.EventTypeGuildCreate, event)
}
func discordSnapshotForTest(t *testing.T, s *server) map[string]any {
	t.Helper()
	s.discordMu.Lock()
	defer s.discordMu.Unlock()
	v, err := s.store.discordRead()
	if err != nil {
		t.Fatal(err)
	}
	return s.discordSnapshot(v)
}
func TestDiscordGatewayMinimumIntent(t *testing.T) {
	// Construction is local; Open is deliberately never called.
	c := newDiscordGateway(context.Background(), discordTestToken, func(gateway.Gateway, gateway.EventType, int, gateway.EventData) {}, func() {})
	g, ok := c.(gateway.Gateway)
	if !ok || g.Intents() != gateway.IntentGuilds {
		t.Fatal("Gateway must request exactly IntentGuilds")
	}
	if g.Intents().Has(gateway.IntentMessageContent) || g.Intents().Has(gateway.IntentGuildMembers) || g.Intents().Has(gateway.IntentGuildPresences) || g.Intents().Has(gateway.IntentGuildMessages) || g.Intents().Has(gateway.IntentDirectMessages) {
		t.Fatal("unrelated intent enabled")
	}
}
func TestDiscordEnableDisableLifecycle(t *testing.T) {
	s := discordTestServer(t)
	configureTestDiscord(t, s)
	actor := administratorSession(time.Now())
	if rr := discordAdminRequest(s, actor, map[string]any{"action": "enabled", "enabled": true}); rr.Code != 409 {
		t.Fatal("enabled unverified bot", rr.Code)
	}
	v := discordVerifiedState()
	v.GuildID = "111"
	v.GuildName = "Saved server"
	discordSaveTestState(t, s, v)
	clients := discordLoopForTest(t, s, false)
	select {
	case <-clients:
		t.Fatal("disabled runtime opened")
	case <-time.After(300 * time.Millisecond):
	}
	rr := discordAdminRequest(s, actor, map[string]any{"action": "enabled", "enabled": true})
	if rr.Code != 200 || !strings.Contains(rr.Body.String(), `"runtime_status":"connecting"`) {
		t.Fatal("enable did not start connecting", rr.Code)
	}
	c := discordNextClient(t, clients)
	discordEventually(t, func() bool { return discordSnapshotForTest(t, s)["runtime_status"] == "connected" })
	rr = discordAdminRequest(s, actor, map[string]any{"action": "enabled", "enabled": false})
	after, err := s.store.discordRead()
	if rr.Code != 200 || err != nil || after.Enabled || !after.Verified || after.Token != v.Token || after.BotID != v.BotID || after.ApplicationID != v.ApplicationID || after.GuildID != v.GuildID || after.GuildName != v.GuildName || after.Generation != v.Generation+2 || c.closes.Load() == 0 {
		t.Fatal("disable failed to retain configuration or stop client", rr.Code, err)
	}
	if discordSnapshotForTest(t, s)["runtime_status"] != "disabled" {
		t.Fatal("disabled status incorrect")
	}
}
func TestDiscordRuntimeRevocationFencesCallbacks(t *testing.T) {
	for _, action := range []string{"credential", "disconnect", "factory"} {
		t.Run(action, func(t *testing.T) {
			s, rt := discordRuntimeFixture(t)
			discordGuildEvent(t, s, rt, `{"id":"111","name":"Old server"}`)
			v, _ := s.store.discordRead()
			v.GuildID = "111"
			v.GuildName = "Old server"
			discordSaveTestState(t, s, v)
			c := rt.client.(*discordFakeGateway)
			lifetime, cancel := context.WithCancel(context.Background())
			defer cancel()
			rt.cancel = cancel
			if action == "factory" {
				if err := s.discordFactoryRevoke(); err != nil {
					t.Fatal(err)
				}
			} else {
				body := map[string]any{"action": action}
				if action == "credential" {
					body["token"] = "replacement"
				}
				if rr := discordAdminRequest(s, administratorSession(time.Now()), body); rr.Code != 200 {
					t.Fatal("revocation failed", rr.Code)
				}
			}
			if c.closes.Load() == 0 || s.discordRuntime != nil || lifetime.Err() == nil {
				t.Fatal("client not stopped")
			}
			// Even if a new runtime exists, old callbacks must not populate its guilds.
			next := &discordRuntime{generation: 99, guilds: map[string]discordGuild{}}
			s.discordRuntime = next
			discordGuildEvent(t, s, rt, `{"id":"222","name":"Stale callback"}`)
			if len(next.guilds) != 0 {
				t.Fatal("stale callback changed runtime")
			}
			after, err := s.store.discordRead()
			if err != nil || after.Enabled || after.Verified || after.GuildID != "" || after.GuildName != "" || after.BotID != "" || after.ApplicationID != "" {
				t.Fatal("revocation retained configuration", err)
			}
			if action == "credential" && (after.Token != "replacement" || after.Generation <= v.Generation) {
				t.Fatal("replacement did not invalidate generation")
			}
			if action != "credential" && after.Token != "" {
				t.Fatal("revocation retained credential")
			}
		})
	}
}
func TestDiscordInitialFailurePausesBoundedly(t *testing.T) {
	s := discordTestServer(t)
	v := discordVerifiedState()
	v.Enabled = true
	discordSaveTestState(t, s, v)
	clients := discordLoopForTest(t, s, true)
	for i := 0; i < 3; i++ {
		discordNextClient(t, clients)
	}
	discordEventually(t, func() bool { return discordSnapshotForTest(t, s)["runtime_status"] == "connection_paused" })
	select {
	case <-clients:
		t.Fatal("paused client retried")
	case <-time.After(300 * time.Millisecond):
	}
	actor := administratorSession(time.Now())
	for _, enabled := range []bool{false, true} {
		if rr := discordAdminRequest(s, actor, map[string]any{"action": "enabled", "enabled": enabled}); rr.Code != 200 {
			t.Fatal(rr.Code)
		}
	}
	discordNextClient(t, clients)
}
func TestDiscordGuildDiscoveryBindingAndAvailability(t *testing.T) {
	s, rt := discordRuntimeFixture(t)
	var ready gateway.EventReady
	if err := json.Unmarshal([]byte(`{"guilds":[{"id":"111"},{"id":"222"}],"session_id":"private-session","resume_gateway_url":"private-resume"}`), &ready); err != nil {
		t.Fatal(err)
	}
	s.discordEvent(rt, gateway.EventTypeReady, ready)
	actor := administratorSession(time.Now())
	bind := func(id string) int {
		return discordAdminRequest(s, actor, map[string]any{"action": "guild", "guild_id": id}).Code
	}
	if rt.ready || bind("111") == 200 {
		t.Fatal("binding before initial discovery")
	}
	discordGuildEvent(t, s, rt, `{"id":"111","name":"Zulu","members":[],"channels":[],"roles":[]}`)
	if rt.ready {
		t.Fatal("discovery finished prematurely")
	}
	discordGuildEvent(t, s, rt, `{"id":"222","name":"Alpha"}`)
	if !rt.ready {
		t.Fatal("discovery not complete")
	}
	for _, id := range []string{"0", "-1", "999", "18446744073709551616"} {
		if bind(id) == 200 {
			t.Fatal("bound invalid or absent guild")
		}
	}
	if rr := discordAdminRequest(s, actor, map[string]any{"action": "guild", "guild_id": "111", "guild_name": "Forged"}); rr.Code != 400 {
		t.Fatal("browser guild_name accepted", rr.Code)
	}
	if bind("111") != 200 {
		t.Fatal("available guild rejected")
	}
	v, _ := s.store.discordRead()
	if v.GuildID != "111" || v.GuildName != "Zulu" {
		t.Fatal("binding not authoritative")
	}
	var update gateway.EventGuildUpdate
	if err := json.Unmarshal([]byte(`{"id":"111","name":"Renamed"}`), &update); err != nil {
		t.Fatal(err)
	}
	s.discordEvent(rt, gateway.EventTypeGuildUpdate, update)
	if discordSnapshotForTest(t, s)["guild_name"] != "Renamed" {
		t.Fatal("rename not reflected")
	}
	var removed gateway.EventGuildDelete
	if err := json.Unmarshal([]byte(`{"id":"111","unavailable":true}`), &removed); err != nil {
		t.Fatal(err)
	}
	s.discordEvent(rt, gateway.EventTypeGuildDelete, removed)
	if bind("111") == 200 || discordSnapshotForTest(t, s)["bound_guild_available"] != false {
		t.Fatal("unavailable guild bindable")
	}
	discordGuildEvent(t, s, rt, `{"id":"111","name":"Returned"}`)
	if discordSnapshotForTest(t, s)["bound_guild_available"] != true {
		t.Fatal("guild availability not restored")
	}
	removed.Unavailable = false
	s.discordEvent(rt, gateway.EventTypeGuildDelete, removed)
	v, _ = s.store.discordRead()
	if v.GuildID != "111" || v.GuildName != "Zulu" || discordSnapshotForTest(t, s)["bound_guild_available"] != false {
		t.Fatal("leave cleared or switched binding")
	}
	guilds := discordSnapshotForTest(t, s)["guilds"].([]discordGuild)
	if len(guilds) != 1 || guilds[0].ID != "222" {
		t.Fatal("leave did not update discovery")
	}
	// A fresh READY rebuilds the authoritative cache after a non-resumed session.
	s.discordEvent(rt, gateway.EventTypeReady, gateway.EventReady{})
	if !rt.ready || len(rt.guilds) != 0 {
		t.Fatal("fresh discovery retained stale guilds")
	}
}
func TestDiscordGuildPublicStateBoundedSortedAndSafe(t *testing.T) {
	s, rt := discordRuntimeFixture(t)
	rt.ready = true
	for i := 0; i < discordGuildLimit+20; i++ {
		id := fmt.Sprint(i + 1)
		rt.guilds[id] = discordGuild{ID: id, Name: fmt.Sprintf("Guild %03d", i), Available: true}
	}
	rt.guilds["999"] = discordGuild{ID: "999", Name: "Unavailable", Available: false}
	snapshot := discordSnapshotForTest(t, s)
	guilds := snapshot["guilds"].([]discordGuild)
	if len(guilds) != discordGuildLimit || guilds[0].Name != "Guild 000" || guilds[len(guilds)-1].Name != "Guild 199" {
		t.Fatal("guild view not sorted and bounded")
	}
	raw, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{discordTestToken, `"token"`, `session`, `resume`, `sequence`, `Authorization`, `members`, `roles`, `channels`, `private diagnostic`, `Available`} {
		if strings.Contains(string(raw), secret) {
			t.Fatal("unsafe public field", secret)
		}
	}
	if got := discordGuildName(strings.Repeat("x", 120) + "\n"); len(got) != 100 {
		t.Fatal("guild name not bounded")
	}
	rt.client.(*discordFakeGateway).status.Store(int32(gateway.StatusDisconnected))
	snapshot = discordSnapshotForTest(t, s)
	if snapshot["runtime_status"] != "connection_retry" || snapshot["guilds_ready"] != false || len(snapshot["guilds"].([]discordGuild)) != 0 {
		t.Fatal("disconnected cache exposed as current")
	}
}
func TestDiscordBindingPersistenceAndVolatileRestart(t *testing.T) {
	store, path := openTestWebUIStore(t)
	v := discordVerifiedState()
	v.Enabled = true
	v.GuildID = "111"
	v.GuildName = "Saved guild"
	s := &server{store: store}
	discordSaveTestState(t, s, v)
	if err := store.close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := openWebUIStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.close()
	s.store = reopened
	after, err := reopened.discordRead()
	if err != nil || after.GuildID != v.GuildID || after.GuildName != v.GuildName || !after.Enabled {
		t.Fatal("binding lost on reopen", err)
	}
	snapshot := discordSnapshotForTest(t, s)
	if snapshot["runtime_status"] != "connecting" || snapshot["guilds_ready"] != false || snapshot["bound_guild_available"] != false {
		t.Fatal("restart inferred connected from disk")
	}
}
func TestDiscordInstallURLLeastPrivilege(t *testing.T) {
	v := discordVerifiedState()
	raw := discordPublicState(v)["install_url"].(string)
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	if u.Scheme != "https" || u.Host != "discord.com" || u.Path != "/oauth2/authorize" || u.User != nil || u.Fragment != "" || len(q) != 3 || q.Get("client_id") != v.ApplicationID || q.Get("scope") != "bot" || q.Get("permissions") != "3072" {
		t.Fatal("unsafe install URL")
	}
	// Administrator is bit 3; the fixed permission set contains only bits 10 and 11.
	permissions, err := strconv.ParseUint(q.Get("permissions"), 10, 64)
	if err != nil || permissions&8 != 0 || permissions != (1024|2048) {
		t.Fatal("unexpected permission set")
	}
	v.Verified = false
	if discordPublicState(v)["install_url"] != "" {
		t.Fatal("unverified install URL")
	}
}

func TestDiscordGenerationFenceRejectsCurrentPointerWithOldGeneration(t *testing.T) {
	s, rt := discordRuntimeFixture(t)
	v, _ := s.store.discordRead()
	v.Generation++
	discordSaveTestState(t, s, v)
	discordGuildEvent(t, s, rt, `{"id":"111","name":"Old generation"}`)
	if len(rt.guilds) != 0 || rt.ready {
		t.Fatal("old generation event mutated runtime")
	}
	if snapshot := discordSnapshotForTest(t, s); snapshot["runtime_status"] != "connecting" || snapshot["guilds_ready"] != false {
		t.Fatal("old generation shown as connected")
	}
	if rr := discordAdminRequest(s, administratorSession(time.Now()), map[string]any{"action": "guild", "guild_id": "111"}); rr.Code == 200 {
		t.Fatal("old generation allowed binding")
	}
}

func TestDiscordRuntimeInactiveDuringFactoryReset(t *testing.T) {
	s := discordTestServer(t)
	operations := openTestOperationStore(t)
	attachTestOperationStore(t, s, operations)
	if _, _, err := operations.beginFactoryReset(testFactoryResetFingerprint); err != nil {
		t.Fatal(err)
	}
	v := discordVerifiedState()
	v.Enabled = true
	discordSaveTestState(t, s, v)
	clients := discordLoopForTest(t, s, false)
	select {
	case <-clients:
		t.Fatal("runtime started during Factory Reset")
	case <-time.After(300 * time.Millisecond):
	}
	if rr := discordAdminRequest(s, administratorSession(time.Now()), map[string]any{"action": "enabled", "enabled": true}); rr.Code != 409 {
		t.Fatal("Factory Reset allowed enabling", rr.Code)
	}
}
