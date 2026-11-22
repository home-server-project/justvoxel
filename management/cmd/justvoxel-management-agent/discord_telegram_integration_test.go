package main

import (
	"context"
	"encoding/json"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/gateway"
)

// Both surfaces use this server and its single private identity database. No
// polling loop is started; any accidental outbound request fails locally.
func discordTelegramFixture(t *testing.T) (*server, *discordRuntime, discordState) {
	t.Helper()
	s, rt, v := discordBoundFixture(t)
	discordApproveFixture(t, s, &v, "777")
	discordTelegramConfigure(t, s)
	return s, rt, v
}

func discordTelegramConfigure(t *testing.T, s *server) {
	t.Helper()
	v := telegramTestState()
	v.BotID, v.BotName = 123456789, "voxel_bot"
	v.Offset = 42
	v.Seen["456:online"] = activationOldBoot
	if _, err := v.issueCallback(telegramCallback{UserID: 456, Operation: "instance_status", InstanceID: "jv-a1b2c3"}, s.store.now()); err != nil {
		t.Fatal(err)
	}
	if err := s.store.telegramSave(v, nil, ""); err != nil {
		t.Fatal(err)
	}
	d, err := s.store.discordRead()
	if err != nil {
		t.Fatal(err)
	}
	d.Pending = []discordIdentity{{GuildID: d.GuildID, UserID: "779", Name: "Pending member", Username: "pending", RequestedAt: s.store.now()}}
	discordSaveTestState(t, s, d)
	old := telegramRequest
	telegramRequest = func(context.Context, string, string, any, any) error {
		t.Error("unexpected outbound Telegram request")
		return errors.New("transport disabled by integration fixture")
	}
	t.Cleanup(func() { telegramRequest = old })
	useDiscordTestTransport(t, func(*http.Request) (*http.Response, error) {
		t.Error("unexpected outbound Discord HTTP request")
		return nil, errors.New("transport disabled by integration fixture")
	})
}

func discordTelegramReads(t *testing.T, s *server) (discordState, telegramState) {
	t.Helper()
	d, err := s.store.discordRead()
	if err != nil {
		t.Fatal(err)
	}
	v, err := s.store.telegramRead()
	if err != nil {
		t.Fatal(err)
	}
	return d, v
}

func discordTelegramInvoke(t *testing.T, s *server, user int64, cb telegramCallback) (int, json.RawMessage) {
	t.Helper()
	s.telegramMu.Lock()
	defer s.telegramMu.Unlock()
	_, v := discordTelegramReads(t, s)
	return s.telegramInvoke(context.Background(), &v, user, cb)
}

func discordTelegramMinecraftFixture(t *testing.T) (*server, *discordRuntime, discordState, *instanceActivationFixture) {
	t.Helper()
	s, rt, v, f := discordMinecraftFixture(t)
	discordApproveFixture(t, s, &v, "777")
	discordTelegramConfigure(t, s)
	return s, rt, v, f
}

func TestDiscordTelegramStateIndependence(t *testing.T) {
	for _, surface := range []string{"discord", "telegram"} {
		for _, action := range []string{"toggle", "credential", "disconnect", "identity"} {
			t.Run(surface+"/"+action, func(t *testing.T) {
				s, rt, v := discordTelegramFixture(t)
				accessCreateTest(t, s, v, "778", "jv-a1b2c3", "java")
				beforeD, beforeT := discordTelegramReads(t, s)
				beforeAccess := accessRowsTest(t, s)
				actor := administratorSession(time.Now())
				request := func(body map[string]any) {
					t.Helper()
					var code int
					if surface == "discord" {
						code = discordAdminRequest(s, actor, body).Code
					} else {
						code = telegramAdminRequest(s, actor, body).Code
					}
					if code != 200 {
						t.Fatalf("%s %s rejected: %d", surface, action, code)
					}
					d, telegram := discordTelegramReads(t, s)
					if surface == "discord" && !reflect.DeepEqual(beforeT, telegram) || surface == "telegram" && (!reflect.DeepEqual(beforeD, d) || !reflect.DeepEqual(beforeAccess, accessRowsTest(t, s))) {
						t.Fatal("transition changed the other integration before the next action")
					}
				}
				switch action {
				case "toggle":
					request(map[string]any{"action": "enabled", "enabled": false})
					midD, midT := discordTelegramReads(t, s)
					if surface == "discord" && (midD.Enabled || midD.Generation <= beforeD.Generation) || surface == "telegram" && (midT.Enabled || midT.Generation <= beforeT.Generation) {
						t.Fatal("disable did not change selected integration")
					}
					request(map[string]any{"action": "enabled", "enabled": true})
				case "credential":
					token := discordTestToken + "replacement"
					if surface == "telegram" {
						token = strings.Replace(telegramTestToken, "123456789:", "987654321:", 1)
					}
					request(map[string]any{"action": "credential", "token": token})
				case "disconnect":
					request(map[string]any{"action": "disconnect"})
				case "identity":
					if surface == "discord" {
						rt.guilds["222"] = discordGuild{ID: "222", Name: "Other", Available: true}
						request(map[string]any{"action": "guild", "guild_id": "222"})
					} else {
						request(map[string]any{"action": "pair"})
						request(map[string]any{"action": "user", "user_id": 123, "permission": telegramManagement})
					}
				}
				afterD, afterT := discordTelegramReads(t, s)
				if surface == "discord" {
					if !reflect.DeepEqual(beforeT, afterT) {
						t.Fatal("Discord transition changed persisted Telegram state")
					}
					if action == "toggle" && (!afterD.Enabled || afterD.Generation != beforeD.Generation+2 || !reflect.DeepEqual(afterD.Administrators, beforeD.Administrators)) {
						t.Fatal("Discord toggle lost retained trust")
					}
					if action != "toggle" && (len(afterD.Administrators) != 0 || len(afterD.Pending) != 0 || len(accessRowsTest(t, s)) != 0) {
						t.Fatal("Discord trust transition retained approvals/requests")
					}
					if action == "credential" && (afterD.Verified || afterD.Enabled || afterD.GuildID != "" || afterD.Token == beforeD.Token) || action == "disconnect" && afterD.Token != "" || action == "identity" && afterD.GuildID != "222" {
						t.Fatal("Discord transition did not replace selected state")
					}
				} else {
					if !reflect.DeepEqual(beforeD, afterD) || !reflect.DeepEqual(beforeAccess, accessRowsTest(t, s)) {
						t.Fatal("Telegram transition changed Discord trust/community state")
					}
					if action == "toggle" && (!afterT.Enabled || afterT.Generation != beforeT.Generation+2 || !reflect.DeepEqual(afterT.Users, beforeT.Users)) {
						t.Fatal("Telegram toggle lost retained users")
					}
					if (action == "credential" || action == "disconnect") && (afterT.Verified || afterT.Enabled || len(afterT.Users) != 0 || len(afterT.Callbacks) != 0) {
						t.Fatal("Telegram credential trust survived revocation")
					}
					if action == "credential" && afterT.Token == beforeT.Token || action == "disconnect" && afterT.Token != "" {
						t.Fatal("Telegram credential was not replaced/revoked")
					}
				}
			})
		}
	}
}

func TestDiscordTelegramSimultaneouslyEnabled(t *testing.T) {
	s, _, _ := discordTelegramFixture(t)
	for _, enabled := range []bool{false, true, false} {
		if err := s.store.gatewaySetEnabled(enabled, gatewayTestActor); err != nil {
			t.Fatal(err)
		}
		d, v := discordTelegramReads(t, s)
		if !d.Enabled || !d.Verified || !v.Enabled || !v.Verified || !telegramAllowed(&v, 456, "instance_start") || !d.administrator("777") {
			t.Fatal("independent integration switches depend on gateway")
		}
	}
}

func TestDiscordTelegramRestartPersistence(t *testing.T) {
	s, rt, v := discordTelegramFixture(t)
	accessCreateTest(t, s, v, "778", "jv-a1b2c3", "java")
	if err := s.store.discordRebootCreate(rebootEvidence(v, s.store.now())); err != nil {
		t.Fatal(err)
	}
	_, telegram := discordTelegramReads(t, s)
	telegram.Reboot = &telegramReboot{BootID: activationOldBoot, Expected: "sha256:" + strings.Repeat("c", 64), RequestedAt: s.store.now(), State: "requesting"}
	if err := s.store.telegramSave(telegram, nil, ""); err != nil {
		t.Fatal(err)
	}
	discordNewAction(rt, discordPendingAction{Generation: v.Generation, GuildID: v.GuildID, UserID: "777", Operation: "reboot"}, s.store.now())
	beforeD, beforeT := discordTelegramReads(t, s)
	beforeAccess := accessRowsTest(t, s)
	beforeReboot, err := s.store.discordRebootRead()
	if err != nil {
		t.Fatal(err)
	}
	var path string
	if err = s.store.db.QueryRow(`SELECT file FROM pragma_database_list WHERE name='main'`).Scan(&path); err != nil {
		t.Fatal(err)
	}
	if err = s.store.close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := openWebUIStore(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { reopened.close() })
	// Deliberately construct a fresh server, never copy volatile runtime fields.
	fresh := &server{store: reopened, sessions: map[string]session{}}
	afterD, afterT := discordTelegramReads(t, fresh)
	afterReboot, err := reopened.discordRebootRead()
	if err != nil || !reflect.DeepEqual(beforeD, afterD) || !reflect.DeepEqual(beforeT, afterT) || !reflect.DeepEqual(beforeAccess, accessRowsTest(t, fresh)) || !reflect.DeepEqual(beforeReboot, afterReboot) {
		t.Fatal("durable integration state changed on reopen", err)
	}
	public := discordSnapshotForTest(t, fresh)
	if fresh.discordRuntime != nil || public["runtime_status"] != "connecting" || public["guilds_ready"] != false || public["bound_guild_available"] != false {
		t.Fatal("disk state incorrectly implies a connected Discord runtime")
	}
	if p := advancedHandle(t, fresh, rt, v, "777", "adv_system", ""); p.text != discordFailure().text {
		t.Fatal("previous server runtime accepted after restart")
	}
	for _, table := range []string{"discord_state", "telegram_state"} {
		var raw string
		if err := reopened.db.QueryRow("SELECT record_json FROM " + table + " WHERE id=1").Scan(&raw); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(raw, "interaction-token") || strings.Contains(raw, "never-retain") {
			t.Fatal("interaction token persisted")
		}
	}
}

func TestDiscordReconnectPreservesTelegramAndDurableState(t *testing.T) {
	s, rt, v := discordTelegramFixture(t)
	accessCreateTest(t, s, v, "778", "jv-a1b2c3", "java")
	beforeD, beforeT := discordTelegramReads(t, s)
	beforeAccess := accessRowsTest(t, s)
	// The Gateway client owns reconnection. A fresh READY must discard the old
	// guild cache; no configuration generation or saved guild is changed.
	client := rt.client.(*discordFakeGateway)
	client.status.Store(int32(gateway.StatusDisconnected))
	if discordSnapshotForTest(t, s)["runtime_status"] != "connection_retry" {
		t.Fatal("disconnect presented as connected")
	}
	if p := advancedHandle(t, s, rt, v, "777", "adv_system", ""); p.text != discordFailure().text {
		t.Fatal("disconnected runtime executed interaction")
	}
	// Start the existing runtime loop to create a replacement client locally.
	s.discordMu.Lock()
	s.discordStopLocked()
	s.discordMu.Unlock()
	clients := discordLoopForTest(t, s, false)
	freshClient := discordNextClient(t, clients)
	discordEventually(t, func() bool { return discordSnapshotForTest(t, s)["runtime_status"] == "connected" })
	s.discordMu.Lock()
	fresh := s.discordRuntime
	s.discordMu.Unlock()
	if fresh == rt || freshClient == client || discordSnapshotForTest(t, s)["bound_guild_available"] != false {
		t.Fatal("replacement reused volatile discovery")
	}
	var ready gateway.EventReady
	if err := json.Unmarshal([]byte(`{"guilds":[{"id":"111","unavailable":true}]}`), &ready); err != nil {
		t.Fatal(err)
	}
	s.discordEvent(fresh, gateway.EventTypeReady, ready)
	if discordSnapshotForTest(t, s)["guilds_ready"] != false {
		t.Fatal("READY alone implied bound guild discovery completed")
	}
	discordGuildEvent(t, s, rt, `{"id":"111","name":"stale callback"}`)
	discordGuildEvent(t, s, fresh, `{"id":"222","name":"Other"}`)
	if discordSnapshotForTest(t, s)["bound_guild_available"] != false {
		t.Fatal("wrong guild or stale callback restored usability")
	}
	if p := advancedHandle(t, s, fresh, v, "777", "adv_system", ""); p.text != discordFailure().text {
		t.Fatal("saved guild absent but interaction executed")
	}
	discordGuildEvent(t, s, fresh, `{"id":"111","name":"Rediscovered"}`)
	if discordSnapshotForTest(t, s)["bound_guild_available"] != true {
		t.Fatal("fresh saved guild discovery did not restore availability")
	}
	afterD, afterT := discordTelegramReads(t, s)
	if !reflect.DeepEqual(beforeD, afterD) || !reflect.DeepEqual(beforeT, afterT) || !reflect.DeepEqual(beforeAccess, accessRowsTest(t, s)) {
		t.Fatal("reconnect changed durable identities or Telegram")
	}
}

func TestDiscordTelegramStartOnBootSharedState(t *testing.T) {
	s, rt, v, f := discordTelegramMinecraftFixture(t)
	id := f.configs[0].ID
	before, err := f.backend.readRecord(id)
	if err != nil {
		t.Fatal(err)
	}
	backupSettings := instanceBackupSettings{Time: "06:15", Retention: 19}
	if err := f.backend.writeJSON(f.backend.backupPath(id, "settings"), backupSettings, true); err != nil {
		t.Fatal(err)
	}
	code, raw := discordTelegramInvoke(t, s, 456, telegramCallback{Operation: "start_on_boot", InstanceID: id, Confirmed: true, Body: telegramBody(map[string]bool{"enabled": !before.Runtime.StartOnBoot})})
	if code != 200 {
		t.Fatalf("Telegram boot preference: %d %s", code, raw)
	}
	p := advancedHandle(t, s, rt, v, "777", "adv_boot_review", id)
	nonce := advancedNonce(t, p, "adv_confirm")
	if rt.actions[nonce].PreviousEnabled != !before.Runtime.StartOnBoot || rt.actions[nonce].DesiredEnabled != before.Runtime.StartOnBoot {
		t.Fatal("Discord did not review Telegram's authoritative value")
	}
	advancedHandle(t, s, rt, v, "777", "adv_confirm", nonce)
	code, raw = discordTelegramInvoke(t, s, 456, telegramCallback{Operation: "instance_details", InstanceID: id})
	var details struct {
		ID      string `json:"id"`
		Runtime struct {
			StartOnBoot bool `json:"start_on_boot"`
		} `json:"runtime"`
	}
	if code != 200 || json.Unmarshal(raw, &details) != nil || details.ID != id || details.Runtime.StartOnBoot != before.Runtime.StartOnBoot {
		t.Fatalf("Telegram read after Discord mutation: %d %s", code, raw)
	}
	after, err := f.backend.readRecord(id)
	if err != nil || !reflect.DeepEqual(before.Runtime, after.Runtime) || !reflect.DeepEqual(before.Storage, after.Storage) || !reflect.DeepEqual(before.Backups, after.Backups) || !reflect.DeepEqual(before.Firewall, after.Firewall) {
		t.Fatal("boot round trip changed unrelated runtime/storage/backup settings", err)
	}
	if afterBackups, err := f.backend.backupSettings(id); err != nil || !reflect.DeepEqual(backupSettings, afterBackups) {
		t.Fatal("Start on Boot changed unrelated backup settings", err)
	}
}

func discordTelegramLifecycleInvoke(t *testing.T, s *server, rt *discordRuntime, v discordState, op, id string) (int, []byte) {
	t.Helper()
	s.discordMu.Lock()
	defer s.discordMu.Unlock()
	return s.discordInvokeLocked(context.Background(), rt, v.GuildID, "777", op, id, false)
}

func TestDiscordTelegramSharedLifecycleAuthority(t *testing.T) {
	s, rt, v, f := discordTelegramMinecraftFixture(t)
	id := f.configs[1].ID
	if code, raw := discordTelegramLifecycleInvoke(t, s, rt, v, "start", id); code != 200 || f.states[id] != "active" {
		t.Fatalf("Discord start did not enter shared runtime: %d %s", code, raw)
	}
	code, raw := discordTelegramInvoke(t, s, 123, telegramCallback{Operation: "instance_status", InstanceID: id})
	var status instanceRuntimeStatus
	if code != 200 || json.Unmarshal(raw, &status) != nil || status.State != "running" {
		t.Fatalf("Telegram cannot see Discord start: %d %s", code, raw)
	}
	f.players = "There are 1 of a max of 20 players online: Alex"
	for _, op := range []string{"stop", "restart"} {
		codeD, rawD := discordTelegramLifecycleInvoke(t, s, rt, v, op, id)
		codeT, rawT := discordTelegramInvoke(t, s, 456, telegramCallback{Operation: "instance_" + op, InstanceID: id, Confirmed: true})
		for _, result := range []struct {
			code int
			raw  []byte
		}{{codeD, rawD}, {codeT, rawT}} {
			var safety struct {
				ConfirmationRequired bool   `json:"confirmation_required"`
				Reason               string `json:"reason"`
			}
			if result.code != 200 || json.Unmarshal(result.raw, &safety) != nil || !safety.ConfirmationRequired || safety.Reason != "players_online" || f.states[id] != "active" {
				t.Fatalf("surface bypassed shared player safety: %d %s", result.code, result.raw)
			}
		}
	}
	f.players = "There are 0 of a max of 20 players online:"
	if code, raw := discordTelegramInvoke(t, s, 456, telegramCallback{Operation: "instance_stop", InstanceID: id, Confirmed: true}); code != 200 || f.states[id] != "inactive" {
		t.Fatalf("Telegram stop did not change shared runtime: %d %s", code, raw)
	}
	code, raw = discordTelegramLifecycleInvoke(t, s, rt, v, "status", id)
	if code != 200 || json.Unmarshal(raw, &status) != nil || status.State != "stopped" {
		t.Fatalf("Discord cannot see Telegram stop: %d %s", code, raw)
	}
}

func TestDiscordTelegramSharedManualBackupAuthority(t *testing.T) {
	f := newInstanceBackupFixture(t)
	s, rt, v := discordTelegramFixture(t)
	s.operations = f.server.operations
	discordRegistryFixture(t, f.backend.identities())
	id := f.configs[1].ID
	starts := 0
	base := f.backend.command
	f.backend.command = func(ctx context.Context, command string, args ...string) ([]byte, error) {
		if command == "systemctl" && reflect.DeepEqual(args, []string{"--no-block", "start", "minecraft-backup-" + id + ".service"}) {
			starts++
		}
		return base(ctx, command, args...)
	}
	if code, raw := discordTelegramLifecycleInvoke(t, s, rt, v, "backup", id); code != 202 {
		t.Fatalf("Discord backup: %d %s", code, raw)
	}
	if code, raw := discordTelegramInvoke(t, s, 456, telegramCallback{Operation: "backup_now", InstanceID: id, Confirmed: true}); code != 202 {
		t.Fatalf("Telegram backup: %d %s", code, raw)
	}
	if starts != 2 {
		t.Fatal("surfaces did not use same backup service", starts)
	}
	for _, actor := range []string{"discord:777", "telegram:456"} {
		var count int
		if err := s.store.db.QueryRow(`SELECT count(*) FROM audit_events WHERE action='instance_backup' AND actor_username=? AND target=? AND success=1`, actor, id).Scan(&count); err != nil || count != 1 {
			t.Fatal("shared backup handler audit missing", actor, err)
		}
	}
}

func TestDiscordTelegramMultipleInstanceIsolation(t *testing.T) {
	s, rt, v, f := discordTelegramMinecraftFixture(t)
	a, b := f.configs[0].ID, f.configs[1].ID
	const c = "jv-abcdef"
	registerLogicalForTest(t, f.backend.identities(), "abcdef", "paper", "Same display name")
	third := f.configs[0]
	third.ID, third.JavaPort = c, instanceTestPort(t)
	for third.JavaPort == f.configs[0].JavaPort || third.JavaPort == f.configs[1].JavaPort {
		third.JavaPort = instanceTestPort(t)
	}
	third.StartOnBoot = false
	dir, _, _ := f.backend.paths(c)
	d, err := f.backend.privateDirectory(dir, true)
	if err != nil {
		t.Fatal(err)
	}
	d.Close()
	if err := f.backend.writeJSON(filepath.Join(dir, "config.json"), instanceRuntimeRecord{SchemaVersion: 1, Runtime: third, Storage: instanceStorageFor(f.snapshot, c), Firewall: []instanceFirewallRule{}}, false); err != nil {
		t.Fatal(err)
	}
	request := accessCreateTest(t, s, v, "778", c, "java")
	if err := f.backend.action(context.Background(), a, "stop"); err != nil {
		t.Fatal(err)
	}
	nonce := discordNonce(t, discordHandleTest(t, s, rt, v, "777", "start", a))
	if rt.actions[nonce].InstanceID != a {
		t.Fatal("Discord selected action did not bind immutable A")
	}
	code, raw := discordTelegramInvoke(t, s, 456, telegramCallback{UserID: 456, Operation: "instance_start", InstanceID: b, Confirmed: true})
	if code != 200 || f.states[b] != "active" || f.states[a] != "inactive" {
		t.Fatalf("Telegram B start: %d %s", code, raw)
	}
	// A forged component cannot replace the server-side A confirmation with B.
	if p := discordHandleTest(t, s, rt, v, "777", "confirm", nonce+":"+b); p.text != discordFailure().text {
		t.Fatal("component target substitution accepted")
	}
	discordHandleTest(t, s, rt, v, "777", "status", b)
	if rt.actions[nonce].InstanceID != a {
		t.Fatal("selecting B retargeted the pending A confirmation")
	}
	f.calls = nil
	if p := discordHandleTest(t, s, rt, v, "777", "confirm", nonce); !strings.Contains(p.text, "Request accepted") || f.states[a] != "active" {
		t.Fatal("Discord A confirmation did not execute A", p.text)
	}
	for _, call := range f.calls {
		if strings.Contains(call, "--no-block start minecraft-"+b+".service") {
			t.Fatal("Discord A action redirected to B")
		}
	}
	_, telegram := discordTelegramReads(t, s)
	key, err := telegram.issueCallback(telegramCallback{UserID: 456, Operation: "instance_stop", InstanceID: b, Confirmed: true}, s.store.now())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.store.telegramSave(telegram, nil, ""); err != nil {
		t.Fatal(err)
	}
	telegramRequest = func(context.Context, string, string, any, any) error { return nil }
	var update telegramUpdate
	encoded, err := json.Marshal(map[string]any{"update_id": 100, "callback_query": map[string]any{"id": "query-B", "from": map[string]any{"id": 456}, "message": map[string]any{"chat": map[string]any{"id": 456, "type": "private"}}, "data": key}})
	if err != nil || json.Unmarshal(encoded, &update) != nil {
		t.Fatal("invalid Telegram callback fixture", err)
	}
	s.telegramProcessUpdate(context.Background(), telegram.Generation, update)
	if f.states[b] != "inactive" || f.states[a] != "active" {
		t.Fatal("persisted Telegram B callback affected A or failed to stop B")
	}
	_, telegram = discordTelegramReads(t, s)
	if _, exists := telegram.Callbacks[key]; exists {
		t.Fatal("Telegram B callback was not consumed")
	}
	rows := accessRowsTest(t, s)
	if len(rows) != 1 || rows[0].RequestID != request.RequestID || rows[0].InstanceID != c || rows[0].State != "pending" {
		t.Fatal("community request C redirected or mutated")
	}
	code, raw = discordTelegramLifecycleInvoke(t, s, rt, v, "status", a)
	var status instanceRuntimeStatus
	if code != 200 || json.Unmarshal(raw, &status) != nil || status.ID != a {
		t.Fatalf("status lost immutable A: %d %s", code, raw)
	}
}

func TestDiscordTelegramVersionReviewCrossSurfaceFence(t *testing.T) {
	s, rt, v, f := discordTelegramMinecraftFixture(t)
	a, b := f.configs[0].ID, f.configs[1].ID
	oldStatus, oldWorker := runAdminVersionStatus, startMinecraftStackUpdateWorker
	t.Cleanup(func() { runAdminVersionStatus = oldStatus; startMinecraftStackUpdateWorker = oldWorker })
	fingerprint := "sha256:" + strings.Repeat("a", 64)
	runAdminVersionStatus = func(ctx context.Context, _, _, _, _ string) ([]byte, error) {
		fp := fingerprint
		if ctx.Value(instanceVersionKey{}) == b {
			fp = "sha256:" + strings.Repeat("c", 64)
		}
		return json.Marshal(adminVersionStatus{Installed: "1.21", Available: "1.22", SelectedCandidate: "1.22", StackState: "updates_available", ServerSupported: true, UpdateAvailable: true, PlanFingerprint: fp})
	}
	accepted := 0
	startMinecraftStackUpdateWorker = func(*server, string, minecraftStackUpdateRequest, session) { accepted++ }
	review := advancedHandle(t, s, rt, v, "777", "adv_version_review", a)
	nonce := advancedNonce(t, review, "adv_confirm")
	// Telegram reads the same plan; neither integration owns a shared plan cache.
	code, raw := discordTelegramInvoke(t, s, 456, telegramCallback{Operation: "version_status", InstanceID: a})
	if code != 200 || !strings.Contains(string(raw), fingerprint) || rt.actions[nonce].Fingerprint != fingerprint {
		t.Fatal("surfaces reviewed different authoritative plans")
	}
	oldFingerprint := fingerprint
	fingerprint = "sha256:" + strings.Repeat("b", 64) // authoritative status seam changes
	p := advancedHandle(t, s, rt, v, "777", "adv_confirm", nonce)
	if accepted != 0 || !strings.Contains(p.text, "changed") {
		t.Fatal("Discord obsolete review accepted", p.text)
	}
	for _, id := range []string{a, b} {
		code, _ := discordTelegramInvoke(t, s, 456, telegramCallback{Operation: "version_update", InstanceID: id, Confirmed: true, Body: telegramBody(minecraftStackUpdateRequest{PlanFingerprint: oldFingerprint})})
		if code != 409 || accepted != 0 {
			t.Fatal("Telegram obsolete/cross-instance fingerprint accepted", id, code)
		}
	}
	// Bind an actual Telegram review callback, then change backend truth again.
	telegramRequest = func(context.Context, string, string, any, any) error { return nil }
	_, telegram := discordTelegramReads(t, s)
	s.telegramMu.Lock()
	s.telegramReviewUpdate(context.Background(), &telegram, 456, telegramCallback{Operation: "version_review", InstanceID: a})
	s.telegramMu.Unlock()
	_, telegram = discordTelegramReads(t, s)
	var callback telegramCallback
	for _, cb := range telegram.Callbacks {
		if cb.Operation == "version_update" {
			callback = cb
		}
	}
	if callback.InstanceID != a || !strings.Contains(string(callback.Body), fingerprint) {
		t.Fatal("Telegram review did not bind authoritative A plan")
	}
	fingerprint = "sha256:" + strings.Repeat("d", 64)
	callback.Confirmed = true
	if code, _ := discordTelegramInvoke(t, s, 456, callback); code != 409 || accepted != 0 {
		t.Fatal("Telegram reviewed callback bypassed stale plan fence", code)
	}
}

func TestDiscordTelegramContainerReviewCrossSurfaceFence(t *testing.T) {
	s, rt, v, f := discordTelegramMinecraftFixture(t)
	id := f.configs[0].ID
	oldStatus, oldWorker := runMinecraftContainerStatus, startMinecraftContainerUpdateWorker
	t.Cleanup(func() { runMinecraftContainerStatus = oldStatus; startMinecraftContainerUpdateWorker = oldWorker })
	fingerprint := "sha256:" + strings.Repeat("a", 64)
	runMinecraftContainerStatus = func(_ context.Context, target string) (minecraftContainerStatus, error) {
		if target != id {
			t.Error("container plan target changed")
		}
		return minecraftContainerStatus{InstanceID: id, Running: true, State: "update_available", PlanFingerprint: fingerprint}, nil
	}
	var accepted *minecraftContainerUpdateRequest
	startMinecraftContainerUpdateWorker = func(_ *server, _ string, req minecraftContainerUpdateRequest, _ session) { accepted = &req }
	p := advancedHandle(t, s, rt, v, "777", "adv_container_review", id)
	nonce := advancedNonce(t, p, "adv_confirm")
	oldFingerprint := fingerprint
	if code, raw := discordTelegramInvoke(t, s, 456, telegramCallback{Operation: "container_status", InstanceID: id}); code != 200 || !strings.Contains(string(raw), fingerprint) {
		t.Fatal("Telegram cannot see shared container plan")
	}
	fingerprint = "sha256:" + strings.Repeat("b", 64)
	p = advancedHandle(t, s, rt, v, "777", "adv_confirm", nonce)
	if accepted != nil || !strings.Contains(p.text, "changed") {
		t.Fatal("stale Discord container review accepted")
	}
	if code, _ := discordTelegramInvoke(t, s, 456, telegramCallback{Operation: "container_update", InstanceID: id, Confirmed: true, Body: telegramBody(minecraftContainerUpdateRequest{PlanFingerprint: oldFingerprint, RestartNow: true, BackupBeforeRestart: true})}); code != 409 || accepted != nil {
		t.Fatal("stale Telegram container plan accepted", code)
	}
	p = advancedHandle(t, s, rt, v, "777", "adv_container_review", id)
	p = advancedHandle(t, s, rt, v, "777", "adv_confirm", advancedNonce(t, p, "adv_confirm"))
	if accepted == nil || !accepted.RestartNow || !accepted.BackupBeforeRestart || accepted.RestoreDigest != "" || accepted.PlanFingerprint != fingerprint {
		t.Fatal("fresh Discord container review lost fixed safe options", p.text)
	}
}

func TestDiscordTelegramNetworkCrossSurfaceFence(t *testing.T) {
	f, config := networkTransactionFixture(t)
	s, _ := networkLockFixture(t, f)
	bound, rt, v := discordTelegramFixture(t)
	s.store, s.sessions, s.discordRuntime = bound.store, bound.sessions, rt
	discordRegistryFixture(t, f.backend.identities())
	config.Enabled = false
	if err := networkApply(context.Background(), f.backend, config); err != nil {
		t.Fatal(err)
	}
	p := advancedHandle(t, s, rt, v, "777", "adv_network_enable", "")
	nonce := advancedNonce(t, p, "adv_confirm")
	current, err := networkRead(f.backend)
	if err != nil {
		t.Fatal(err)
	}
	old := minecraftNetworkFingerprint(current.Config)
	if rt.actions[nonce].NetworkExpected != old {
		t.Fatal("Discord network review did not bind config A")
	}
	code, raw := discordTelegramInvoke(t, s, 456, telegramCallback{Operation: "network_change", Confirmed: true, Body: telegramBody(map[string]any{"enabled": true, "expected": old})})
	if code != 200 {
		t.Fatalf("Telegram network mutation: %d %s", code, raw)
	}
	current, err = networkRead(f.backend)
	if err != nil || !current.Config.Enabled || minecraftNetworkFingerprint(current.Config) == old || !reflect.DeepEqual(current.Config.Members, config.Members) {
		t.Fatal("shared network state did not change with immutable members", err)
	}
	p = advancedHandle(t, s, rt, v, "777", "adv_confirm", nonce)
	if !strings.Contains(p.text, "changed") {
		t.Fatal("obsolete Discord network review accepted", p.text)
	}
	// A fresh Discord review sees Telegram's enabled config and can disable it.
	p = advancedHandle(t, s, rt, v, "777", "adv_network_disable", "")
	advancedHandle(t, s, rt, v, "777", "adv_confirm", advancedNonce(t, p, "adv_confirm"))
	code, raw = discordTelegramInvoke(t, s, 123, telegramCallback{Operation: "network_status"})
	var observed struct {
		Config minecraftNetworkConfig `json:"config"`
	}
	if code != 200 || json.Unmarshal(raw, &observed) != nil || observed.Config.Enabled || !reflect.DeepEqual(observed.Config.Members, config.Members) {
		t.Fatalf("Telegram did not observe Discord's network config: %d %s", code, raw)
	}
	id := config.Members[0]
	f.states[id], f.containers[id] = "active", "running"
	current, err = networkRead(f.backend)
	if err != nil {
		t.Fatal(err)
	}
	code, raw = discordTelegramInvoke(t, s, 456, telegramCallback{Operation: "network_change", Confirmed: true, Body: telegramBody(map[string]any{"members": []string{}, "expected": minecraftNetworkFingerprint(current.Config)})})
	if code < 400 || !strings.Contains(string(raw), "stop affected servers") {
		t.Fatalf("Telegram bypassed authoritative stopped-member rule: %d %s", code, raw)
	}
	p = advancedHandle(t, s, rt, v, "777", "adv_member", id)
	p = advancedHandle(t, s, rt, v, "777", "adv_confirm", advancedNonce(t, p, "adv_confirm"))
	if !strings.Contains(p.text, "stop affected servers") {
		t.Fatal("Discord bypassed shared stopped-member rule", p.text)
	}
	final, err := networkRead(f.backend)
	if err != nil || !reflect.DeepEqual(current.Config, final.Config) {
		t.Fatal("failed membership mutation changed network", err)
	}
}

func TestDiscordTelegramUpdateRebootCrossSurfaceFence(t *testing.T) {
	s, rt, v, _ := advancedOSFixture(t)
	discordTelegramConfigure(t, s)
	oldBoot, oldHelper := readSystemUpdateBootID, runAdminSystemUpdateRebootHelper
	t.Cleanup(func() { readSystemUpdateBootID = oldBoot; runAdminSystemUpdateRebootHelper = oldHelper })
	readSystemUpdateBootID = func() (string, error) { return activationOldBoot, nil }
	rebootCalled := false
	runAdminSystemUpdateRebootHelper = func(context.Context, ...string) ([]byte, int, error) {
		rebootCalled = true
		return nil, 0, errors.New("unexpected reboot helper invocation")
	}
	p := advancedHandle(t, s, rt, v, "777", "adv_reboots", "")
	var reviewed discordPendingAction
	var nonce string
	for key, a := range rt.actions {
		if a.Operation == "update_reboot" && a.WarningSeconds == 60 && a.BackupMinecraft {
			reviewed, nonce = a, key
			break
		}
	}
	if nonce == "" || reviewed.StagedDigest != "sha256:"+strings.Repeat("b", 64) {
		t.Fatal("missing staged A review", p.text)
	}
	p = advancedHandle(t, s, rt, v, "777", "adv_review", nonce)
	nonce = advancedNonce(t, p, "adv_confirm")
	final, ok := rt.actions[nonce]
	if !ok || final.ReviewOnly || final.Operation != "update_reboot" || final.StagedDigest != reviewed.StagedDigest || final.WarningSeconds != reviewed.WarningSeconds || final.BackupMinecraft != reviewed.BackupMinecraft || final.GuildID != v.GuildID || final.UserID != "777" || final.Generation != v.Generation {
		t.Fatal("final confirmation did not preserve the reviewed staged A action", p.text)
	}
	if code, raw := discordTelegramInvoke(t, s, 456, telegramCallback{Operation: "system_updates"}); code != 200 || !strings.Contains(string(raw), reviewed.StagedDigest) {
		t.Fatal("Telegram does not observe same staged digest")
	}
	telegramRequest = func(context.Context, string, string, any, any) error { return nil }
	_, telegram := discordTelegramReads(t, s)
	s.telegramMu.Lock()
	s.telegramRebootMenu(context.Background(), &telegram, 456)
	s.telegramMu.Unlock()
	_, telegram = discordTelegramReads(t, s)
	var oldCallback telegramCallback
	for _, cb := range telegram.Callbacks {
		if cb.Operation == "update_reboot" {
			oldCallback = cb
			break
		}
	}
	if !strings.Contains(string(oldCallback.Body), reviewed.StagedDigest) {
		t.Fatal("Telegram review did not bind same staged A")
	}
	_, beforeTelegram := discordTelegramReads(t, s)
	// Existing authoritative bootc status seam, with no actual bootc invocation.
	runBootcStatus = func(context.Context) ([]byte, error) {
		return []byte(strings.Replace(bootcStatusStagedFixture, reviewed.StagedDigest, "sha256:"+strings.Repeat("c", 64), 1)), nil
	}
	p = advancedHandle(t, s, rt, v, "777", "adv_confirm", nonce)
	if !strings.Contains(p.text, "Staged update changed. Review reboot options again.") {
		t.Fatal("Discord did not reject the stale reviewed deployment", p.text)
	}
	if rebootCalled {
		t.Fatal("Discord invoked the reboot helper for an obsolete staged deployment")
	}
	if pending, err := s.store.discordRebootRead(); err != nil || pending != nil {
		t.Fatal("obsolete review created reboot evidence", err)
	}
	_, afterTelegram := discordTelegramReads(t, s)
	if !reflect.DeepEqual(beforeTelegram, afterTelegram) {
		t.Fatal("refused Discord reboot changed persisted Telegram state")
	}
	// Telegram forwards its bound digest to the existing update-reboot helper,
	// which owns its staged-deployment fence. Model only that helper's refusal;
	// this test makes no claim that a real reboot/backend was exercised.
	telegramHelperCalled := false
	runAdminSystemUpdateRebootHelper = func(_ context.Context, args ...string) ([]byte, int, error) {
		telegramHelperCalled = true
		if !strings.Contains(strings.Join(args, " "), "--staged-digest="+reviewed.StagedDigest) {
			t.Error("Telegram silently replaced reviewed digest")
		}
		return []byte(`{"ok":false,"state":"failed","reason":"staged_deployment_changed","accepted":false}`), 2, errors.New("staged deployment changed")
	}
	oldCallback.Confirmed = true
	code, raw := discordTelegramInvoke(t, s, 456, oldCallback)
	if code != 409 || !telegramHelperCalled {
		t.Fatalf("Telegram ignored authoritative staged-digest refusal: %d %s", code, raw)
	}
	var result adminSystemUpdateRebootResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("decode Telegram staged-digest refusal: %v", err)
	}
	if result.OK || result.Accepted || result.State != "failed" || result.Reason != "staged_deployment_changed" {
		t.Fatalf("Telegram ignored authoritative staged-digest refusal: %+v", result)
	}
}

func TestDiscordTelegramSharedConcurrencyAuthority(t *testing.T) {
	for _, first := range []string{"telegram", "discord"} {
		t.Run(first, func(t *testing.T) {
			s, rt, v, fake := advancedOSFixture(t)
			discordTelegramConfigure(t, s)
			old := runBootcUpgrade
			t.Cleanup(func() { runBootcUpgrade = old })
			started, release, finished := make(chan struct{}), make(chan struct{}), make(chan struct{})
			joined := false
			defer func() {
				select {
				case <-release:
				default:
					close(release)
				}
				if !joined {
					if first == "telegram" {
						select {
						case <-finished:
						case <-time.After(3 * time.Second):
							t.Error("Telegram worker cleanup timed out")
						}
					} else {
						select {
						case <-fake.messages:
						case <-time.After(3 * time.Second):
							t.Error("Discord worker cleanup timed out")
						}
					}
				}
			}()
			var calls atomic.Int32
			runBootcUpgrade = func(context.Context) ([]byte, error) {
				if calls.Add(1) != 1 {
					t.Error("second surface bypassed shared update lock")
					return nil, errors.New("unexpected second invocation")
				}
				close(started)
				<-release
				return nil, nil
			}
			discordDownload := func() {
				t.Helper()
				p := advancedHandle(t, s, rt, v, "777", "adv_download", "")
				advancedHandle(t, s, rt, v, "777", "adv_confirm", advancedNonce(t, p, "adv_confirm"))
			}
			if first == "telegram" {
				go func() {
					defer close(finished)
					discordTelegramInvoke(t, s, 456, telegramCallback{Operation: "system_download", Confirmed: true})
				}()
			} else {
				discordDownload()
			}
			select {
			case <-started:
			case <-time.After(3 * time.Second):
				close(release)
				t.Fatal("first surface did not reach authoritative update")
			}
			// Always release a blocked worker before restoring global test seams.
			if first == "telegram" {
				discordDownload()
				select {
				case message := <-fake.messages:
					if strings.Contains(message.Content, "completed.") {
						t.Error("lock conflict claimed completion")
					}
				case <-time.After(3 * time.Second):
					t.Error("Discord lock conflict did not return")
				}
			} else {
				code, _ := discordTelegramInvoke(t, s, 456, telegramCallback{Operation: "system_download", Confirmed: true})
				if code != 409 {
					t.Error("Telegram bypassed Discord-held authoritative lock", code)
				}
			}
			close(release)
			if first == "telegram" {
				select {
				case <-finished:
				case <-time.After(3 * time.Second):
					t.Fatal("Telegram worker did not finish")
				}
			} else {
				select {
				case <-fake.messages:
				case <-time.After(3 * time.Second):
					t.Fatal("Discord worker did not finish")
				}
			}
			joined = true
			if calls.Load() != 1 {
				t.Fatal("surfaces used separate concurrency authority", calls.Load())
			}
		})
	}
}

func TestDiscordTelegramNotificationIsolation(t *testing.T) {
	for _, fail := range []bool{false, true} {
		name := "delivered"
		if fail {
			name = "failed_delivery"
		}
		t.Run(name, func(t *testing.T) {
			s, rt, v, f := discordTelegramMinecraftFixture(t)
			fake := &advancedREST{discordFakeREST: newDiscordFakeREST(), messages: make(chan discord.MessageCreate, 8), fail: fail}
			rt.api = fake
			isolateSystemUpdateLock(t)
			oldStatus, oldUpgrade, oldBoot := runBootcStatus, runBootcUpgrade, readSystemUpdateBootID
			t.Cleanup(func() {
				runBootcStatus, runBootcUpgrade, readSystemUpdateBootID = oldStatus, oldUpgrade, oldBoot
			})
			runBootcStatus = func(context.Context) ([]byte, error) { return []byte(bootcStatusStagedFixture), nil }
			runBootcUpgrade = func(context.Context) ([]byte, error) { return []byte("PRIVATE BACKEND /private/path"), nil }
			readSystemUpdateBootID = func() (string, error) { return activationOldBoot, nil }
			telegramCalls := 0
			telegramRequest = func(_ context.Context, token, method string, input, _ any) error {
				telegramCalls++
				if token != telegramTestToken || method != "sendMessage" {
					t.Error("wrong outbound Telegram route")
				}
				body, err := json.Marshal(input)
				if err != nil {
					t.Error(err)
				}
				for _, bad := range []string{discordTestToken, telegramTestToken, "interaction-token", "never-retain", "PRIVATE", "/private/path"} {
					if strings.Contains(string(body), bad) {
						t.Error("private data in Telegram notification")
					}
				}
				if fail {
					return errors.New("PRIVATE BACKEND /private/path " + discordTestToken)
				}
				return nil
			}
			beforeD, beforeT := discordTelegramReads(t, s)
			request := accessCreateTest(t, s, v, "778", f.configs[0].ID, "java")
			p := discordHandleTest(t, s, rt, v, "777", "access_reject", request.RequestID)
			if !strings.Contains(p.text, "rejected") {
				t.Fatal("community decision failed", p.text)
			}
			// Exercise real completion/reconciliation entry points, with only their
			// authoritative system backends stubbed. No reboot is requested.
			s.discordDownload(discordActor(v, "777"), discordPendingAction{Generation: v.Generation, GuildID: v.GuildID, ChannelID: "999"})
			evidence := rebootEvidence(v, s.store.now().Add(-21*time.Minute))
			if err := s.store.discordRebootCreate(evidence); err != nil {
				t.Fatal(err)
			}
			s.discordMu.Lock()
			s.discordReconcileRebootLocked(rt, v)
			s.discordMu.Unlock()
			discordEventually(t, func() bool {
				s.discordMu.Lock()
				defer s.discordMu.Unlock()
				return !s.discordRebootReconciling
			})
			if pending, err := s.store.discordRebootRead(); err != nil || pending != nil {
				t.Fatal("reboot result did not remove terminal evidence", err)
			}
			for i := 0; i < 3; i++ {
				select {
				case message := <-fake.messages:
					if message.AllowedMentions == nil || !reflect.DeepEqual(*message.AllowedMentions, discord.AllowedMentions{}) {
						t.Fatal("Discord notification permits mentions")
					}
					for _, bad := range []string{"@everyone", "@here", discordTestToken, telegramTestToken, "interaction-token", "never-retain", "PRIVATE", "/private/path"} {
						if strings.Contains(message.Content, bad) {
							t.Fatal("unsafe outbound Discord content")
						}
					}
				default:
					t.Fatal("missing Discord notification")
				}
			}
			_, afterT := discordTelegramReads(t, s)
			if telegramCalls != 0 || !reflect.DeepEqual(beforeT, afterT) {
				t.Fatal("Discord delivery touched Telegram")
			}
			s.telegramMu.Lock()
			s.telegramNotify(context.Background(), &afterT, "backup:"+f.configs[1].ID, "attempt:failed", "Backup failed. Review JustVoxel.", false)
			s.telegramMu.Unlock()
			afterD, _ := discordTelegramReads(t, s)
			if telegramCalls != 1 || !reflect.DeepEqual(beforeD, afterD) {
				t.Fatal("Telegram delivery touched Discord")
			}
			select {
			case <-fake.messages:
				t.Fatal("Telegram notification crossed into Discord")
			default:
			}
		})
	}
}

func TestDiscordTelegramRebootEvidenceIndependent(t *testing.T) {
	s, rt, v := discordTelegramFixture(t)
	discordEvidence := rebootEvidence(v, s.store.now())
	discordEvidence.Operation = "update_reboot"
	discordEvidence.ExpectedDigest = "sha256:" + strings.Repeat("a", 64)
	_, beforeT := discordTelegramReads(t, s)
	if err := s.store.discordRebootCreate(discordEvidence); err != nil {
		t.Fatal(err)
	}
	_, telegram := discordTelegramReads(t, s)
	if !reflect.DeepEqual(beforeT, telegram) {
		t.Fatal("Discord reboot evidence overwrote Telegram")
	}
	beforeD, err := s.store.discordRebootRead()
	if err != nil {
		t.Fatal(err)
	}
	telegram.Reboot = &telegramReboot{Operation: "update_reboot", BootID: activationOldBoot, Expected: "sha256:" + strings.Repeat("b", 64), RequestedAt: s.store.now(), State: "requesting"}
	if err := s.store.telegramSave(telegram, nil, ""); err != nil {
		t.Fatal(err)
	}
	afterD, err := s.store.discordRebootRead()
	if err != nil || !reflect.DeepEqual(beforeD, afterD) {
		t.Fatal("Telegram reboot evidence overwrote Discord", err)
	}
	_, telegram = discordTelegramReads(t, s)
	// Both presentations consume the same boot/deployment truth but must check
	// their own expected digest. Neither can bless the other's requested image.
	for _, boot := range []string{activationOldBoot, activationNewBoot} {
		for _, running := range []string{discordEvidence.ExpectedDigest, telegram.Reboot.Expected, "sha256:" + strings.Repeat("c", 64)} {
			truth := adminSystemUpdateStatus{Running: adminSystemUpdateDeployment{Digest: running}}
			dText, _ := discordRebootResult(*afterD, boot, truth, true, s.store.now())
			tText, _ := telegramRebootResult(*telegram.Reboot, boot, truth, true, s.store.now())
			if strings.Contains(dText, "OS update confirmed:") != (boot != activationOldBoot && running == discordEvidence.ExpectedDigest) || strings.Contains(tText, "OS update confirmed:") != (boot != activationOldBoot && running == telegram.Reboot.Expected) {
				t.Fatal("one integration confirmed the other's reboot evidence")
			}
		}
	}
	// Finishing either presentation only clears its own durable record.
	if err := s.store.discordRebootFinish(*afterD, ""); err != nil {
		t.Fatal(err)
	}
	_, unchanged := discordTelegramReads(t, s)
	if !reflect.DeepEqual(telegram, unchanged) {
		t.Fatal("Discord result reconciliation changed Telegram evidence")
	}
	if err := s.store.discordRebootCreate(discordEvidence); err != nil {
		t.Fatal(err)
	}
	telegramRequest = func(context.Context, string, string, any, any) error { return nil }
	s.telegramMu.Lock()
	s.telegramFinishReboot(context.Background(), &unchanged, "Update result could not be verified. Review JustVoxel.")
	s.telegramMu.Unlock()
	persisted, err := s.store.discordRebootRead()
	if err != nil || !reflect.DeepEqual(persisted, afterD) || s.discordRuntime != rt {
		t.Fatal("Telegram result reconciliation changed Discord evidence/runtime", err)
	}
}

func TestDiscordTelegramRebootPlayerSafety(t *testing.T) {
	s, rt, v, _ := advancedOSFixture(t)
	discordTelegramConfigure(t, s)
	oldBoot, oldAction := readSystemUpdateBootID, runAdminSystemActionHelper
	t.Cleanup(func() { readSystemUpdateBootID = oldBoot; runAdminSystemActionHelper = oldAction })
	readSystemUpdateBootID = func() (string, error) { return activationOldBoot, nil }
	calls := 0
	runAdminSystemActionHelper = func(_ context.Context, args ...string) ([]byte, int, error) {
		calls++
		if strings.Contains(strings.Join(args, " "), "--confirm-players") {
			t.Error("surface invented player interruption consent")
		}
		return []byte(`{"ok":false,"action":"reboot","state":"confirmation_required","confirmation_required":true,"reason":"players_online"}`), 10, errors.New("players")
	}
	a := discordPendingAction{Generation: v.Generation, GuildID: v.GuildID, UserID: "777", ChannelID: "999", Operation: "reboot"}
	s.discordMu.Lock()
	p := s.discordRebootLocked(context.Background(), rt, a)
	s.discordMu.Unlock()
	if !strings.Contains(p.text, "Players are online") {
		t.Fatal("Discord bypassed authoritative player warning", p.text)
	}
	code, raw := discordTelegramInvoke(t, s, 456, telegramCallback{Operation: "reboot", Confirmed: true, Body: telegramBody(adminSystemActionRequest{ActionConfirmed: true})})
	if code != 200 || !strings.Contains(string(raw), `"confirmation_required":true`) || calls != 2 {
		t.Fatalf("Telegram bypassed same reboot player safety: %d %s", code, raw)
	}
}

func TestDiscordTelegramRebootAcceptanceCoexistence(t *testing.T) {
	for _, first := range []string{"discord", "telegram"} {
		t.Run(first, func(t *testing.T) {
			s, rt, v, _ := advancedOSFixture(t)
			discordTelegramConfigure(t, s)
			oldBoot, oldAction := readSystemUpdateBootID, runAdminSystemActionHelper
			t.Cleanup(func() { readSystemUpdateBootID = oldBoot; runAdminSystemActionHelper = oldAction })
			readSystemUpdateBootID = func() (string, error) { return activationOldBoot, nil }
			actors := []string{}
			runAdminSystemActionHelper = func(ctx context.Context, args ...string) ([]byte, int, error) {
				actor, ok := ctx.Value(gatewayPrincipalKey{}).(session)
				if !ok || actor.Role != roleAdministrator || strings.Contains(strings.Join(args, " "), "--confirm-players") {
					t.Error("invalid authoritative reboot actor or invented player consent")
				}
				actors = append(actors, actor.Username)
				return []byte(`{"ok":true,"action":"reboot","state":"requested","accepted":true}`), 0, nil
			}
			telegramRequest = func(context.Context, string, string, any, any) error { return nil }
			discordRequest := func() {
				a := discordPendingAction{Generation: v.Generation, GuildID: v.GuildID, UserID: "777", ChannelID: "999", Operation: "reboot"}
				s.discordMu.Lock()
				p := s.discordRebootLocked(context.Background(), rt, a)
				s.discordMu.Unlock()
				if strings.Contains(p.text, "Reboot confirmed") {
					t.Fatal("acceptance falsely claimed verified reboot success")
				}
			}
			telegramRequestReboot := func() {
				_, state := discordTelegramReads(t, s)
				s.telegramMu.Lock()
				s.telegramDispatch(context.Background(), &state, 456, telegramCallback{Operation: "reboot", Confirmed: true, Body: telegramBody(adminSystemActionRequest{ActionConfirmed: true})})
				s.telegramMu.Unlock()
			}
			if first == "discord" {
				discordRequest()
				before, err := s.store.discordRebootRead()
				if err != nil || before == nil {
					t.Fatal("Discord acceptance did not persist evidence", err)
				}
				telegramRequestReboot()
				after, err := s.store.discordRebootRead()
				if err != nil || !reflect.DeepEqual(before, after) {
					t.Fatal("Telegram acceptance replaced Discord evidence", err)
				}
			} else {
				telegramRequestReboot()
				_, before := discordTelegramReads(t, s)
				discordRequest()
				_, after := discordTelegramReads(t, s)
				if !reflect.DeepEqual(before, after) {
					t.Fatal("Discord acceptance replaced Telegram evidence")
				}
			}
			d, err := s.store.discordRebootRead()
			_, telegram := discordTelegramReads(t, s)
			if err != nil || d == nil || telegram.Reboot == nil || d.State != "accepted" || telegram.Reboot.State != "accepted" || d.BootID != activationOldBoot || telegram.Reboot.BootID != activationOldBoot || d.ExpectedDigest != telegram.Reboot.Expected || d.ExpectedDigest != "" {
				t.Fatal("independent reboot acceptance lost shared boot/download-only truth", err)
			}
			sort.Strings(actors)
			if !reflect.DeepEqual(actors, []string{"discord:777", "telegram:456"}) {
				t.Fatal("surfaces did not enter same authoritative reboot handler", actors)
			}
		})
	}
}

func TestDiscordTelegramFactoryResetRevokesBoth(t *testing.T) {
	useEmptyFactoryResetRuntime(t)
	useTestMinecraftInstances(t)
	s, rt, v := discordTelegramFixture(t)
	s.operations = openTestOperationStore(t)
	accessCreateTest(t, s, v, "778", "jv-a1b2c3", "java")
	if err := s.store.discordRebootCreate(rebootEvidence(v, s.store.now())); err != nil {
		t.Fatal(err)
	}
	_, telegram := discordTelegramReads(t, s)
	telegram.Pair = &telegramPair{UserID: 999, Expires: s.store.now().Add(time.Minute)}
	telegram.Reboot = &telegramReboot{BootID: activationOldBoot, RequestedAt: s.store.now(), State: "requesting"}
	if err := s.store.telegramSave(telegram, nil, ""); err != nil {
		t.Fatal(err)
	}
	nonce, ok := discordNewAction(rt, discordPendingAction{Generation: v.Generation, GuildID: v.GuildID, UserID: "777", Operation: "reboot"}, s.store.now())
	if !ok {
		t.Fatal("pending action fixture")
	}
	old := runAdminFactoryResetHelper
	t.Cleanup(func() { runAdminFactoryResetHelper = old })
	cleanupCalls := 0
	runAdminFactoryResetHelper = func(_ context.Context, args ...string) ([]byte, error) {
		if reflect.DeepEqual(args, []string{"plan"}) {
			return []byte(validFactoryResetPlanHelper), nil
		}
		cleanupCalls++
		d, telegram := discordTelegramReads(t, s)
		if d.Token != "" || telegram.Token != "" || s.discordRuntime != nil {
			t.Error("destructive stage began before both revocations")
		}
		return nil, errors.New("simulated later destructive reset failure")
	}
	fingerprint := exactFactoryResetFingerprint(t)
	op, _, err := s.operations.beginFactoryReset(fingerprint)
	if err != nil {
		t.Fatal(err)
	}
	if err := executeFactoryReset(context.Background(), s, op.OperationID, fingerprint); err == nil || cleanupCalls != 1 {
		t.Fatal("expected later reset failure", err, cleanupCalls)
	}
	d, telegram := discordTelegramReads(t, s)
	if d.Token != "" || d.Enabled || d.Verified || d.GuildID != "" || len(d.Administrators) != 0 || len(d.Pending) != 0 || telegram.Token != "" || telegram.Enabled || telegram.Verified || len(telegram.Users) != 0 || len(telegram.Callbacks) != 0 || telegram.Pair != nil || len(telegram.Seen) != 0 || telegram.Reboot != nil {
		t.Fatal("failed reset restored integration trust or pending state")
	}
	if len(accessRowsTest(t, s)) != 0 {
		t.Fatal("reset retained community requests")
	}
	if reboot, err := s.store.discordRebootRead(); err != nil || reboot != nil {
		t.Fatal("reset retained Discord reboot evidence", err)
	}
	if rt.client.(*discordFakeGateway).closes.Load() == 0 || s.discordRuntime != nil {
		t.Fatal("reset retained Discord runtime")
	}
	if p := advancedHandle(t, s, rt, v, "777", "adv_confirm", nonce); p.text != discordFailure().text {
		t.Fatal("pre-reset volatile action executed")
	}
	clients := discordLoopForTest(t, s, false)
	select {
	case <-clients:
		t.Fatal("runtime reconnected during Factory Reset recovery")
	case <-time.After(300 * time.Millisecond):
	}
	finished, err := s.operations.get(op.OperationID)
	if err != nil || finished.State != operationNeedsAttention {
		t.Fatal("reset recovery state lost", err)
	}
}

func TestDiscordTelegramPermissionsRemainIndependent(t *testing.T) {
	s, rt, v, f := discordTelegramMinecraftFixture(t)
	id := f.configs[0].ID
	for _, op := range []string{"instance_start", "instance_stop", "backup_now", "version_update", "system_download", "reboot"} {
		instance := id
		if op == "system_download" || op == "reboot" {
			instance = ""
		}
		for _, user := range []int64{123, 777} { // Read Only and Discord-only admin
			if code, _ := discordTelegramInvoke(t, s, user, telegramCallback{Operation: op, InstanceID: instance, Confirmed: true}); code != 403 {
				t.Fatal("Discord approval granted Telegram management", user, op, code)
			}
		}
	}
	for _, user := range []string{"778", "456"} { // guild member and Telegram manager
		if p := discordHandleTest(t, s, rt, v, user, "start", id); p.text != discordFailure().text {
			t.Fatal("Telegram permission granted Discord lifecycle authority", user)
		}
		if p := advancedHandle(t, s, rt, v, user, "adv_menu", id); p.text != discordFailure().text {
			t.Fatal("Telegram permission granted Discord advanced authority", user)
		}
	}
	beforeD, beforeT := discordTelegramReads(t, s)
	accessCreateTest(t, s, v, "456", id, "java")
	_, afterT := discordTelegramReads(t, s)
	if !reflect.DeepEqual(beforeT, afterT) {
		t.Fatal("community request changed Telegram permissions")
	}
	requests := accessRowsTest(t, s)
	afterT.Pair = &telegramPair{UserID: 777, Expires: s.store.now().Add(time.Minute)}
	if err := s.store.telegramSave(afterT, nil, ""); err != nil {
		t.Fatal(err)
	}
	if rr := telegramAdminRequest(s, administratorSession(time.Now()), map[string]any{"action": "approve", "user_id": 777, "permission": telegramReadOnly}); rr.Code != 200 {
		t.Fatal("explicit Telegram approval fixture failed", rr.Code)
	}
	afterD, afterT := discordTelegramReads(t, s)
	if !reflect.DeepEqual(beforeD, afterD) || !reflect.DeepEqual(requests, accessRowsTest(t, s)) || telegramAllowed(&afterT, 777, "instance_start") || afterD.administrator("456") {
		t.Fatal("explicit Telegram pairing changed Discord trust or Minecraft access")
	}
}

func TestDiscordTelegramMinecraftConcurrencyAuthority(t *testing.T) {
	s, rt, v, f := discordTelegramMinecraftFixture(t)
	a, b := f.configs[0].ID, f.configs[1].ID
	oldVersion, oldContainer := runAdminVersionStatus, runMinecraftContainerStatus
	oldVersionWorker, oldContainerWorker := startMinecraftStackUpdateWorker, startMinecraftContainerUpdateWorker
	t.Cleanup(func() {
		runAdminVersionStatus, runMinecraftContainerStatus = oldVersion, oldContainer
		startMinecraftStackUpdateWorker, startMinecraftContainerUpdateWorker = oldVersionWorker, oldContainerWorker
	})
	versionFP, containerFP := "sha256:"+strings.Repeat("a", 64), "sha256:"+strings.Repeat("b", 64)
	runAdminVersionStatus = func(context.Context, string, string, string, string) ([]byte, error) {
		return json.Marshal(adminVersionStatus{StackState: "updates_available", ServerSupported: true, UpdateAvailable: true, PlanFingerprint: versionFP})
	}
	runMinecraftContainerStatus = func(_ context.Context, id string) (minecraftContainerStatus, error) {
		return minecraftContainerStatus{InstanceID: id, Running: true, State: "update_available", PlanFingerprint: containerFP}, nil
	}
	versionWorkers, containerWorkers := 0, 0
	// Deliberately hold the accepted durable operation before any real worker.
	startMinecraftStackUpdateWorker = func(*server, string, minecraftStackUpdateRequest, session) { versionWorkers++ }
	startMinecraftContainerUpdateWorker = func(*server, string, minecraftContainerUpdateRequest, session) { containerWorkers++ }
	code, raw := discordTelegramInvoke(t, s, 456, telegramCallback{Operation: "version_update", InstanceID: a, Confirmed: true, Body: telegramBody(minecraftStackUpdateRequest{PlanFingerprint: versionFP})})
	if code != 202 || versionWorkers != 1 {
		t.Fatalf("Telegram did not acquire authoritative maintenance operation: %d %s", code, raw)
	}
	code, _ = advancedInvoke(t, s, rt, v, "container_update", discordPendingAction{Generation: v.Generation, GuildID: v.GuildID, UserID: "777", Operation: "container_update", InstanceID: b, Fingerprint: containerFP})
	if code != 409 || containerWorkers != 0 {
		t.Fatal("Discord bypassed shared Minecraft operation authority", code)
	}
	journal, err := s.operations.currentMinecraftMaintenance()
	if err != nil || journal == nil || journal.InstanceID != a || journal.PlanFingerprint != versionFP || journal.OperationType != operationTypeMinecraftUpdate {
		t.Fatal("second surface replaced the first authoritative operation", err)
	}
}

func TestDiscordTelegramSourceSecurityBoundary(t *testing.T) {
	commands := discordCommands()
	if len(commands) != 2 || discordGatewayIntents != gateway.IntentGuilds {
		t.Fatal("Discord command or intent scope expanded")
	}
	for i, name := range []string{"jv", "help"} {
		command, ok := commands[i].(discord.SlashCommandCreate)
		if !ok || command.Name != name || !reflect.DeepEqual(command.Contexts, []discord.InteractionContextType{discord.InteractionContextTypeGuild}) {
			t.Fatal("Discord must retain exactly guild-only /jv and /help")
		}
	}
	restType := reflect.TypeOf((*discordREST)(nil)).Elem()
	methods := []string{}
	for i := 0; i < restType.NumMethod(); i++ {
		methods = append(methods, restType.Method(i).Name)
	}
	sort.Strings(methods)
	if !reflect.DeepEqual(methods, []string{"CreateInteractionResponse", "CreateMessage", "SetGuildCommands", "UpdateInteractionResponse"}) {
		t.Fatal("Discord REST seam expanded beyond fixed guild/interaction/message operations", methods)
	}
	actual := []string{}
	for name := range (*server)(nil).telegramOperations() {
		actual = append(actual, name)
	}
	expected := strings.Fields("status instance_status instance_details players backups backup_status network_status version_status version_update_status container_update_status container_status system_updates system_update_check instance_start instance_stop instance_restart backup_now start_on_boot network_change version_update container_update system_download update_reboot reboot")
	sort.Strings(actual)
	sort.Strings(expected)
	if !reflect.DeepEqual(actual, expected) {
		t.Fatal("Telegram explicit operation catalog changed", actual)
	}
	// Inspect executable Go syntax, not documentation or harmless comments.
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	selectors := map[string]map[string]bool{"discord": {}, "telegram": {}}
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		surface := ""
		for _, prefix := range []string{"discord", "telegram"} {
			if strings.HasPrefix(path, prefix+"_") {
				surface = prefix
			}
		}
		for _, decl := range file.Decls {
			if f, ok := decl.(*ast.FuncDecl); ok && strings.Contains(strings.ToLower(f.Name.Name), "messenger") {
				t.Error("generic messenger dispatcher added", path, f.Name.Name)
			}
		}
		if surface == "" {
			continue
		}
		for _, imp := range file.Imports {
			if imp.Path.Value == `"os/exec"` {
				t.Error("integration directly executes processes", path)
			}
		}
		ast.Inspect(file, func(node ast.Node) bool {
			switch n := node.(type) {
			case *ast.SelectorExpr:
				selectors[surface][n.Sel.Name] = true
				if surface == "discord" && strings.HasPrefix(n.Sel.Name, "telegram") || surface == "telegram" && strings.HasPrefix(n.Sel.Name, "discord") {
					t.Error("interaction surface dispatches through the other integration", path, n.Sel.Name)
				}
				if surface == "discord" {
					for _, bad := range []string{"IntentMessageContent", "IntentGuildMembers", "IntentGuildPresences", "IntentGuildMessages", "IntentDirectMessages", "EventMessageCreate", "EventMessageUpdate", "EventMessageDelete", "Roles", "Permissions"} {
						if n.Sel.Name == bad {
							t.Error("Discord ingestion/role authority expanded", path, bad)
						}
					}
				}
			case *ast.CallExpr:
				if sel, ok := n.Fun.(*ast.SelectorExpr); ok {
					for _, bad := range []string{"Command", "CommandContext", "runWhitelistHelper", "gatewayExecute"} {
						if sel.Sel.Name == bad {
							t.Error("integration bypassed authoritative handler", path, bad)
						}
					}
					// Token arguments go only to immediate REST responses. The sole
					// retention exception is the private discordAdminResponse literal below.
					for _, arg := range n.Args {
						call, ok := arg.(*ast.CallExpr)
						if !ok {
							continue
						}
						if tokenCall, ok := call.Fun.(*ast.SelectorExpr); ok && tokenCall.Sel.Name == "Token" && sel.Sel.Name != "CreateInteractionResponse" && sel.Sel.Name != "UpdateInteractionResponse" {
							t.Error("interaction token escaped immediate response", path)
						}
					}
				}
			case *ast.CompositeLit:
				for _, element := range n.Elts {
					field, ok := element.(*ast.KeyValueExpr)
					if !ok {
						continue
					}
					call, ok := field.Value.(*ast.CallExpr)
					if !ok {
						continue
					}
					selector, ok := call.Fun.(*ast.SelectorExpr)
					if !ok || selector.Sel.Name != "Token" {
						continue
					}
					typ, typeOK := n.Type.(*ast.Ident)
					key, keyOK := field.Key.(*ast.Ident)
					if !typeOK || !keyOK || typ.Name != "discordAdminResponse" || key.Name != "token" || path != "discord_interactions.go" {
						t.Error("interaction token retained outside private runtime response target", path)
					}
				}
			case *ast.AssignStmt:
				for _, rhs := range n.Rhs {
					if call, ok := rhs.(*ast.CallExpr); ok {
						if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Token" {
							t.Error("interaction token assigned outside immediate REST response", path)
						}
					}
				}
			case *ast.BasicLit:
				if n.Kind == token.STRING {
					value, err := strconv.Unquote(n.Value)
					if err != nil {
						t.Error(err)
					}
					for _, bad := range []string{"api.mojang.com", "sessionserver.mojang.com", "xboxlive.com", "api.geysermc.org"} {
						if strings.Contains(value, bad) {
							t.Error("integration introduced direct backend/external identity lookup", path, bad)
						}
					}
					if surface == "discord" {
						for _, bad := range []string{"systemctl", "podman", "rcon-cli", "bootc upgrade", "nmcli", "firewall-cmd"} {
							if strings.Contains(value, bad) {
								t.Error("Discord introduced direct backend commands", path, bad)
							}
						}
					}
				}
			}
			return true
		})
	}
	for _, surface := range []string{"discord", "telegram"} {
		for _, handler := range []string{"instanceRuntimeStatus", "instanceRuntimeAction", "instanceBackupNow", "adminMinecraftStackUpdate", "adminMinecraftContainerUpdate", "adminSystemUpdate", "adminSystemUpdateReboot", "adminSystemAction", "setInstanceStartOnBoot", "changeMinecraftNetwork"} {
			if !selectors[surface][handler] {
				t.Error("surface no longer enters existing authoritative handler", surface, handler)
			}
		}
	}
	for _, name := range []string{"discordState", "discordRebootState", "discordAccessRequest", "discordPendingAction"} {
		// These types are checked structurally below so a new token field cannot
		// hide behind a different JSON key or an omitted field tag.
		var typ reflect.Type
		switch name {
		case "discordState":
			typ = reflect.TypeOf(discordState{})
		case "discordRebootState":
			typ = reflect.TypeOf(discordRebootState{})
		case "discordAccessRequest":
			typ = reflect.TypeOf(discordAccessRequest{})
		case "discordPendingAction":
			typ = reflect.TypeOf(discordPendingAction{})
		}
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			if strings.Contains(strings.ToLower(field.Name), "token") && !(name == "discordState" && field.Name == "Token") {
				t.Error("interaction token persistence/action field added", name, field.Name)
			}
		}
	}
}
