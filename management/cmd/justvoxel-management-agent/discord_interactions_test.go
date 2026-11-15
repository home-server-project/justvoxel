package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
)

type discordCommandCall struct {
	app, guild snowflake.ID
	commands   []discord.ApplicationCommandCreate
}
type discordFakeREST struct {
	mu               sync.Mutex
	commands         []discordCommandCall
	acknowledgements chan discord.InteractionResponse
	replies          chan discord.MessageUpdate
}

func newDiscordFakeREST() *discordFakeREST {
	return &discordFakeREST{acknowledgements: make(chan discord.InteractionResponse, 32), replies: make(chan discord.MessageUpdate, 32)}
}
func (f *discordFakeREST) SetGuildCommands(app, guild snowflake.ID, c []discord.ApplicationCommandCreate, _ ...rest.RequestOpt) ([]discord.ApplicationCommand, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.commands = append(f.commands, discordCommandCall{app, guild, c})
	return nil, nil
}
func (f *discordFakeREST) CreateInteractionResponse(_ snowflake.ID, _ string, r discord.InteractionResponse, _ ...rest.RequestOpt) error {
	f.acknowledgements <- r
	return nil
}
func (f *discordFakeREST) UpdateInteractionResponse(_ snowflake.ID, _ string, m discord.MessageUpdate, _ ...rest.RequestOpt) (*discord.Message, error) {
	f.replies <- m
	return nil, nil
}
func discordBoundFixture(t *testing.T) (*server, *discordRuntime, discordState) {
	t.Helper()
	s, rt := discordRuntimeFixture(t)
	v, _ := s.store.discordRead()
	v.GuildID = "111"
	v.GuildName = "Family"
	discordSaveTestState(t, s, v)
	rt.guilds["111"] = discordGuild{ID: "111", Name: "Family", Available: true}
	rt.ready = true
	rt.api = newDiscordFakeREST()
	return s, rt, v
}
func discordTestInteraction(t *testing.T, guild, user, op, custom string) discord.Interaction {
	t.Helper()
	data := map[string]any{"id": "555", "name": op, "type": 1}
	kind := 2
	if custom != "" {
		kind = 3
		data = map[string]any{"component_type": 2, "custom_id": custom}
		if op == "select" {
			data["component_type"] = 3
			data["values"] = []string{"jv-a1b2c3"}
		}
	}
	raw := map[string]any{"id": "888", "application_id": "456", "type": kind, "token": "private-interaction-token", "data": data}
	if guild != "" {
		raw["guild_id"] = guild
		raw["member"] = map[string]any{"user": map[string]any{"id": user, "username": "friendly", "global_name": "Friendly"}}
	} else {
		raw["user"] = map[string]any{"id": user, "username": "friendly"}
	}
	encoded, _ := json.Marshal(raw)
	i, err := discord.UnmarshalInteraction(encoded)
	if err != nil {
		t.Fatal(err)
	}
	return i
}
func discordHandleTest(t *testing.T, s *server, rt *discordRuntime, v discordState, user, op, id string) discordPanel {
	t.Helper()
	custom := ""
	name := op
	if op != "jv" && op != "help" {
		custom = discordComponentID(v, op, id)
		name = ""
		if op == "select" {
			name = "select"
		}
	}
	i := discordTestInteraction(t, v.GuildID, user, name, custom)
	s.discordMu.Lock()
	defer s.discordMu.Unlock()
	return s.discordHandleLocked(context.Background(), rt, i, v.GuildID, user)
}
func discordApproveFixture(t *testing.T, s *server, v *discordState, user string) {
	t.Helper()
	v.Administrators = append(v.Administrators, discordIdentity{GuildID: v.GuildID, UserID: user, Name: "Friendly", Username: "friendly", RequestedAt: time.Now()})
	discordSaveTestState(t, s, *v)
}
func discordNonce(t *testing.T, p discordPanel) string {
	t.Helper()
	if len(p.components) != 1 {
		t.Fatal("missing confirmation", p.text)
	}
	raw, _ := json.Marshal(p.components)
	var rows []struct {
		Components []struct {
			ID string `json:"custom_id"`
		} `json:"components"`
	}
	if json.Unmarshal(raw, &rows) != nil || len(rows) == 0 || len(rows[0].Components) != 2 {
		t.Fatal("invalid confirmation")
	}
	parts := strings.Split(rows[0].Components[0].ID, ":")
	if len(parts) != 4 || parts[2] != "confirm" {
		t.Fatal(string(raw))
	}
	return parts[3]
}
func discordMinecraftFixture(t *testing.T) (*server, *discordRuntime, discordState, *instanceActivationFixture) {
	t.Helper()
	s, f, _, _ := instanceRestartFixture(t)
	base, rt, v := discordBoundFixture(t)
	s.store = base.store
	s.discordRuntime = rt
	discordRegistryFixture(t, f.backend.identities())
	return s, rt, v, f
}

func TestDiscordGuildCommandReconciliation(t *testing.T) {
	s, rt, v := discordBoundFixture(t)
	fake := rt.api.(*discordFakeREST)
	s.discordMu.Lock()
	s.discordSyncLocked(rt, v)
	s.discordMu.Unlock()
	discordEventually(t, func() bool { fake.mu.Lock(); defer fake.mu.Unlock(); return len(fake.commands) == 1 })
	fake.mu.Lock()
	call := fake.commands[0]
	fake.mu.Unlock()
	if call.app != 456 || call.guild != 111 || len(call.commands) != 2 {
		t.Fatal("incorrect command target")
	}
	for index, name := range []string{"jv", "help"} {
		c := call.commands[index].(discord.SlashCommandCreate)
		if c.Name != name || !reflect.DeepEqual(c.Contexts, []discord.InteractionContextType{discord.InteractionContextTypeGuild}) {
			t.Fatal("incorrect command", c.Name)
		}
	}
	if discordGatewayIntents != gateway.IntentGuilds {
		t.Fatal("extra intent")
	}
	// No global command method is even available through discordREST.
	typ := reflect.TypeOf((*discordREST)(nil)).Elem()
	for index := 0; index < typ.NumMethod(); index++ {
		if strings.Contains(typ.Method(index).Name, "Global") {
			t.Fatal("global command API exposed")
		}
	}
	for _, c := range discordCommands() {
		if c.CommandName() == "justvoxel" {
			t.Fatal("obsolete command registered")
		}
	}
}
func TestDiscordInteractionBoundary(t *testing.T) {
	for _, tc := range []string{"dm", "guild", "generation", "disabled", "unverified", "runtime", "factory", "disconnected"} {
		t.Run(tc, func(t *testing.T) {
			s, rt, v := discordBoundFixture(t)
			guild := "111"
			fake := rt.api.(*discordFakeREST)
			switch tc {
			case "dm":
				guild = ""
			case "guild":
				guild = "222"
			case "generation":
				v.Generation++
				discordSaveTestState(t, s, v)
			case "disabled":
				v.Enabled = false
				discordSaveTestState(t, s, v)
			case "unverified":
				v.Enabled = false
				v.Verified = false
				discordSaveTestState(t, s, v)
			case "runtime":
				rt.status = "connection_retry"
			case "factory":
				operations := openTestOperationStore(t)
				attachTestOperationStore(t, s, operations)
				if _, _, err := operations.beginFactoryReset(testFactoryResetFingerprint); err != nil {
					t.Fatal(err)
				}
			case "disconnected":
				s.discordRuntime = nil
			}
			s.discordInteraction(rt, discordTestInteraction(t, guild, "777", "help", ""))
			select {
			case <-fake.acknowledgements:
				t.Fatal("denied interaction acknowledged")
			default:
			}
		})
	}
}
func TestDiscordReadOnlyPanelAndAuthoritativeEndpoints(t *testing.T) {
	s, rt, v, f := discordMinecraftFixture(t)
	id := f.configs[0].ID
	f.players = "There are 1 of a max of 20 players online: Alex"
	for _, op := range []string{"jv", "select", "status", "players", "refresh"} {
		target := id
		if op == "jv" || op == "select" {
			target = ""
		}
		panel := discordHandleTest(t, s, rt, v, "777", op, target)
		raw, _ := json.Marshal(panel.components)
		if strings.Contains(panel.text, "unavailable or expired") || !strings.Contains(string(raw), "Request admin access") || strings.Contains(string(raw), `"label":"Start"`) {
			t.Fatal(op, panel.text, string(raw))
		}
		if op == "players" && !strings.Contains(panel.text, "1/20") {
			t.Fatal("authoritative player count missing", panel.text)
		}
	}
	panel := discordHandleTest(t, s, rt, v, "777", "status", "jv-dead00")
	if panel.text != discordFailure().text {
		t.Fatal("unregistered instance accepted")
	}
	for _, op := range []string{"start", "stop", "restart", "backup"} {
		if p := discordHandleTest(t, s, rt, v, "777", op, id); p.text != discordFailure().text || len(rt.actions) != 0 {
			t.Fatal("ordinary member can manage", op)
		}
	}
	actor := discordActor(v, "777")
	if actor.Username != "discord:777" || actor.AuthSource != authSourceDiscord || actor.Role != roleViewer {
		t.Fatal("wrong reader principal")
	}
	discordApproveFixture(t, s, &v, "777")
	if discordActor(v, "777").Role != roleAdministrator {
		t.Fatal("approved principal missing")
	}
	if p := discordHandleTest(t, s, rt, v, "777", "start", "jv-dead00"); p.text != discordFailure().text {
		t.Fatal("approved user accepted unregistered instance")
	}
}
func TestDiscordAcknowledgesBeforeAuthoritativeWork(t *testing.T) {
	s, rt, _, _ := discordMinecraftFixture(t)
	fake := rt.api.(*discordFakeREST)
	checkedAck := false
	runAdminDiscoveryHelper = func(context.Context, string) ([]byte, error) {
		checkedAck = true
		select {
		case ack := <-fake.acknowledgements:
			if ack.Type != discord.InteractionResponseTypeDeferredCreateMessage || ack.Data.(discord.MessageCreate).Flags != discord.MessageFlagEphemeral {
				t.Error("reply was not deferred ephemeral")
			}
		default:
			t.Error("authoritative work began before acknowledgement")
		}
		return []byte(`{"configured":false}`), nil
	}

	s.discordInteraction(rt, discordTestInteraction(t, "111", "777", "jv", ""))
	select {
	case reply := <-fake.replies:
		if reply.Content == nil || strings.Contains(*reply.Content, "private") {
			t.Fatal("unsafe response")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("interaction did not complete")
	}
	if !checkedAck {
		t.Fatal("inventory handler was not called")
	}
	var raw string
	if s.store.db.QueryRow(`SELECT record_json FROM discord_state WHERE id=1`).Scan(&raw) != nil || strings.Contains(raw, "private-interaction-token") {
		t.Fatal("interaction token persisted")
	}
}
func TestDiscordSelectBoundAndSafePresentation(t *testing.T) {
	v := discordVerifiedState()
	instances := []discordInstance{}
	for index := 0; index < 26; index++ {
		instances = append(instances, discordInstance{ID: fmt.Sprintf("jv-%06x", index), Name: fmt.Sprintf("Family %02d", index), State: "stopped"})
	}
	panel := discordReadPanel(v, "777", instances, "")
	raw, _ := json.Marshal(panel.components)
	var rows []struct {
		Components []struct {
			Options []json.RawMessage `json:"options"`
		} `json:"components"`
	}
	_ = json.Unmarshal(raw, &rows)
	if len(rows[0].Components[0].Options) != 25 || !strings.Contains(panel.text, "More servers are available in JustVoxel WebUI") {
		t.Fatal("select not bounded")
	}
	if strings.Contains(discordText("@everyone\n**secret**"), "@everyone") {
		t.Fatal("unsafe mentions")
	}
}
func TestDiscordAdminRequestsAndPrimaryApproval(t *testing.T) {
	s, rt, v := discordBoundFixture(t)
	for index := 0; index < 2; index++ {
		p := discordHandleTest(t, s, rt, v, "777", "request", "")
		if !strings.Contains(p.text, "request") {
			t.Fatal(p.text)
		}
	}
	current, _ := s.store.discordRead()
	if len(current.Pending) != 1 || current.Pending[0].UserID != "777" || current.Pending[0].GuildID != v.GuildID || current.Pending[0].RequestedAt.IsZero() {
		t.Fatal("request not bounded or identified")
	}
	actor := administratorSession(time.Now())
	if rr := discordAdminRequest(s, actor, map[string]any{"action": "admin_approve", "user_id": "777", "name": "Forged"}); rr.Code != 400 {
		t.Fatal("browser metadata trusted")
	}
	other := administratorSession(time.Now())
	other.Username = "other"
	discordSession := administratorSession(time.Now())
	discordSession.AuthSource = authSourceDiscord
	external := administratorSession(time.Now())
	external.ConnectionID = "external"
	for _, identity := range []session{other, discordSession, external} {
		if rr := discordAdminRequest(s, identity, map[string]any{"action": "admin_approve", "user_id": "777"}); rr.Code != 403 {
			t.Fatal("nonprimary approval", rr.Code)
		}
	}
	for _, action := range []string{"admin_reject", "admin_approve", "admin_remove"} {
		if action == "admin_approve" {
			discordHandleTest(t, s, rt, v, "777", "request", "")
		}
		if rr := discordAdminRequest(s, actor, map[string]any{"action": action, "user_id": "777"}); rr.Code != 200 {
			t.Fatal(action, rr.Code)
		}
		current, _ = s.store.discordRead()
		if action == "admin_approve" && (!current.administrator("777") || current.Administrators[0].Name != "Friendly") {
			t.Fatal("pending identity not used")
		}
	}
	current, _ = s.store.discordRead()
	if len(current.Pending) != 0 || len(current.Administrators) != 0 {
		t.Fatal("records not removed")
	}
	for index := 0; index < 33; index++ {
		discordHandleTest(t, s, rt, v, fmt.Sprint(1000+index), "request", "")
	}
	current, _ = s.store.discordRead()
	if len(current.Pending) != 32 {
		t.Fatal("request capacity")
	}
	full := current
	full.Pending = nil
	for index := 0; index < 32; index++ {
		full.Administrators = append(full.Administrators, discordIdentity{GuildID: v.GuildID, UserID: fmt.Sprint(2000 + index)})
	}
	full.Pending = []discordIdentity{{GuildID: v.GuildID, UserID: "777"}}
	if full.adminAction("admin_approve", "777") || len(full.Administrators) != 32 || len(full.Pending) != 1 {
		t.Fatal("administrator capacity")
	}
}
func TestDiscordApprovalRevocationAndRetention(t *testing.T) {
	for _, action := range []string{"credential", "disconnect", "factory", "guild", "enabled"} {
		t.Run(action, func(t *testing.T) {
			s, rt, v := discordBoundFixture(t)
			discordApproveFixture(t, s, &v, "777")
			v.Pending = []discordIdentity{{GuildID: v.GuildID, UserID: "778"}}
			discordSaveTestState(t, s, v)
			if action == "factory" {
				if err := s.discordFactoryRevoke(); err != nil {
					t.Fatal(err)
				}
			} else {
				body := map[string]any{"action": action}
				switch action {
				case "credential":
					body["token"] = "replacement"
				case "guild":
					rt.guilds["222"] = discordGuild{ID: "222", Name: "Other", Available: true}
					body["guild_id"] = "222"
				case "enabled":
					body["enabled"] = false
				}
				if rr := discordAdminRequest(s, administratorSession(time.Now()), body); rr.Code != 200 {
					t.Fatal(action, rr.Code)
				}
			}
			after, _ := s.store.discordRead()
			if action == "enabled" {
				if len(after.Administrators) != 1 || len(after.Pending) != 1 {
					t.Fatal("disable cleared approvals")
				}
				if rr := discordAdminRequest(s, administratorSession(time.Now()), map[string]any{"action": "enabled", "enabled": true}); rr.Code != 200 {
					t.Fatal(rr.Code)
				}
				after, _ = s.store.discordRead()
				if !after.administrator("777") {
					t.Fatal("enable lost approval")
				}
			} else if len(after.Pending) != 0 || len(after.Administrators) != 0 {
				t.Fatal("revocation retained approvals")
			}
		})
	}
}
func TestDiscordConfirmationBindingCapacityExpiryAndSingleUse(t *testing.T) {
	now := time.Now()
	rt := &discordRuntime{generation: 10}
	a := discordPendingAction{Generation: 10, GuildID: "111", UserID: "777", InstanceID: "jv-a1b2c3", Operation: "restart"}
	nonce, ok := discordNewAction(rt, a, now)
	if !ok || len(nonce) != 48 || nonce == a.UserID {
		t.Fatal("nonce not opaque")
	}
	second, _ := discordNewAction(rt, a, now)
	if nonce == second {
		t.Fatal("nonce reused")
	}
	if _, ok := discordTakeAction(rt, nonce, "222", "777", now); ok {
		t.Fatal("wrong guild")
	}
	if _, ok := discordTakeAction(rt, nonce, "111", "778", now); ok {
		t.Fatal("wrong user")
	}
	rt.generation++
	if _, ok := discordTakeAction(rt, nonce, "111", "777", now); ok {
		t.Fatal("wrong generation")
	}
	rt.generation--
	taken, ok := discordTakeAction(rt, nonce, "111", "777", now)
	if !ok || taken.InstanceID != a.InstanceID || taken.Operation != a.Operation || taken.Generation != a.Generation || taken.CreatedAt != now || taken.ExpiresAt.Sub(now) != 2*time.Minute {
		t.Fatal("binding lost")
	}
	if _, ok := discordTakeAction(rt, nonce, "111", "777", now); ok {
		t.Fatal("reused")
	}
	if _, ok := discordTakeAction(rt, second, "111", "777", now.Add(2*time.Minute)); ok {
		t.Fatal("expired")
	}
	rt.actions = nil
	for index := 0; index < 128; index++ {
		if _, ok := discordNewAction(rt, a, now); !ok {
			t.Fatal("early limit")
		}
	}
	if _, ok := discordNewAction(rt, a, now); ok {
		t.Fatal("capacity exceeded")
	}
	if _, ok := discordNewAction(rt, a, now.Add(3*time.Minute)); !ok || len(rt.actions) != 1 {
		t.Fatal("expired records not pruned")
	}
}
func TestDiscordCancelAndRemovedAdministratorInvalidateControls(t *testing.T) {
	s, rt, v, f := discordMinecraftFixture(t)
	discordApproveFixture(t, s, &v, "777")
	id := f.configs[0].ID
	first := discordHandleTest(t, s, rt, v, "777", "restart", id)
	nonce := discordNonce(t, first)
	if p := discordHandleTest(t, s, rt, v, "778", "confirm", nonce); p.text != discordFailure().text {
		t.Fatal("wrong user confirmed")
	}
	if p := discordHandleTest(t, s, rt, v, "777", "cancel", nonce); p.text != "Cancelled. Open /jv to continue." {
		t.Fatal("cancel failed", p.text)
	}
	if _, exists := rt.actions[nonce]; exists {
		t.Fatal("cancel did not consume")
	}
	if p := discordHandleTest(t, s, rt, v, "777", "confirm", nonce); p.text != discordFailure().text {
		t.Fatal("cancelled nonce replay")
	}
	nonce = discordNonce(t, discordHandleTest(t, s, rt, v, "777", "restart", id))
	if rr := discordAdminRequest(s, administratorSession(time.Now()), map[string]any{"action": "admin_remove", "user_id": "777"}); rr.Code != 200 {
		t.Fatal(rr.Code)
	}
	if len(rt.actions) != 0 {
		t.Fatal("removal retained nonce")
	}
	if p := discordHandleTest(t, s, rt, v, "777", "confirm", nonce); p.text != discordFailure().text {
		t.Fatal("removed administrator retained authority")
	}
	if p := discordHandleTest(t, s, rt, v, "777", "restart", id); p.text != discordFailure().text {
		t.Fatal("old management button remained usable")
	}
}
func TestDiscordLifecycleUsesAuthoritativePlayerConfirmation(t *testing.T) {
	for _, op := range []string{"stop", "restart"} {
		t.Run(op, func(t *testing.T) {
			s, rt, v, f := discordMinecraftFixture(t)
			discordApproveFixture(t, s, &v, "777")
			id := f.configs[0].ID
			f.players = "There are 1 of a max of 20 players online: Alex"
			f.calls = nil
			nonce := discordNonce(t, discordHandleTest(t, s, rt, v, "777", op, id))
			if len(f.calls) != 0 {
				t.Fatal("first confirmation unexpectedly touched runtime")
			}
			p := discordHandleTest(t, s, rt, v, "777", "confirm", nonce)
			if !strings.Contains(p.text, "Players are online") || f.states[id] != "active" {
				t.Fatal("unconfirmed player interruption", p.text, f.states[id])
			}
			warned := false
			for _, call := range f.calls {
				if call == "podman exec minecraft-"+id+" rcon-cli list" {
					warned = true
				}
				if strings.Contains(call, "say Server shutting down") || strings.Contains(call, "systemctl --no-block stop") {
					t.Fatal("mutation before player warning", call)
				}
			}
			if !warned {
				t.Fatal("existing handler did not query players")
			}
			second := discordNonce(t, p)
			if second == nonce || !rt.actions[second].PlayersConfirmed {
				t.Fatal("second stage not fresh")
			}
			base := f.backend.command
			f.backend.command = func(ctx context.Context, cmd string, args ...string) ([]byte, error) {
				if cmd == "podman" && len(args) == 4 && strings.HasPrefix(args[3], "say ") {
					return nil, nil
				}
				return base(ctx, cmd, args...)
			}
			result := discordHandleTest(t, s, rt, v, "777", "confirm", second)
			if !strings.Contains(result.text, "Request accepted") {
				t.Fatal("confirmed action failed", result.text)
			}
			if len(rt.actions) != 0 {
				t.Fatal("confirmation not single use")
			}
			var audits int
			if err := s.store.db.QueryRow(`SELECT COUNT(*) FROM audit_events WHERE action=? AND actor_username=?`, "minecraft_"+op, "discord:777").Scan(&audits); err != nil || audits == 0 {
				t.Fatal("authoritative lifecycle audit missing", err)
			}
		})
	}
}
func TestDiscordSourceKeepsOperationsAndSecretsAtBoundary(t *testing.T) {
	raw, err := os.ReadFile("discord_interactions.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(raw)
	for _, required := range []string{"s.minecraftInstancesList", "s.instanceRuntimeStatus", "s.instanceMaintenanceRead", "s.instanceRuntimeAction", "s.instanceBackupNow", "gatewayPrincipalKey{}", "crypto/rand", "confirm_players"} {
		if !strings.Contains(source, required) {
			t.Fatal("missing authoritative boundary", required)
		}
	}
	for _, forbidden := range []string{"exec.Command", "systemctl", "rcon-cli", "podman", "SetGlobalCommands", "slog.Error", "slog.Info"} {
		if strings.Contains(source, forbidden) {
			t.Fatal("forbidden direct execution or disclosure", forbidden)
		}
	}
}

func TestDiscordManualBackupUsesExistingHandler(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprint(fail), func(t *testing.T) {
			f := newInstanceBackupFixture(t)
			discordRegistryFixture(t, f.backend.identities())
			s, rt, v := discordBoundFixture(t)
			s.operations = f.server.operations
			discordApproveFixture(t, s, &v, "777")
			id := f.configs[1].ID
			starts := 0
			base := f.backend.command
			f.backend.command = func(ctx context.Context, command string, args ...string) ([]byte, error) {
				if command == "systemctl" && len(args) > 1 && args[0] == "--no-block" && args[1] == "start" {
					if !reflect.DeepEqual(args, []string{"--no-block", "start", "minecraft-backup-" + id + ".service"}) {
						t.Fatal("wrong backup target", args)
					}
					starts++
					if fail {
						return nil, fmt.Errorf("private-stderr /secret/path rcon-password")
					}
				}
				return base(ctx, command, args...)
			}
			nonce := discordNonce(t, discordHandleTest(t, s, rt, v, "777", "backup", id))
			if starts != 0 {
				t.Fatal("backup before confirmation")
			}
			result := discordHandleTest(t, s, rt, v, "777", "confirm", nonce)
			if starts != 1 || strings.Contains(result.text, "private-stderr") || strings.Contains(result.text, "/secret/path") || strings.Contains(result.text, "rcon-password") {
				t.Fatal("unsafe backup result", result.text)
			}
			if strings.Contains(result.text, "Request accepted") == fail {
				t.Fatal("incorrect backup result", result.text)
			}
			var audits int
			if err := s.store.db.QueryRow(`SELECT COUNT(*) FROM audit_events WHERE action='instance_backup' AND actor_username='discord:777' AND target=?`, id).Scan(&audits); err != nil || (!fail && audits != 1) {
				t.Fatal("backup handler audit missing", err, audits)
			}
		})
	}
}
func TestDiscordRebindingFencesOldControlsAndSynchronizesNewGuild(t *testing.T) {
	s, rt, v := discordBoundFixture(t)
	discordApproveFixture(t, s, &v, "777")
	rt.guilds["222"] = discordGuild{ID: "222", Name: "New", Available: true}
	nonce, _ := discordNewAction(rt, discordPendingAction{Generation: v.Generation, GuildID: v.GuildID, UserID: "777", InstanceID: "jv-a1b2c3", Operation: "restart"}, time.Now())
	fake := rt.api.(*discordFakeREST)
	if rr := discordAdminRequest(s, administratorSession(time.Now()), map[string]any{"action": "guild", "guild_id": "222"}); rr.Code != 200 {
		t.Fatal(rr.Code)
	}
	after, _ := s.store.discordRead()
	if after.Generation == v.Generation || len(rt.actions) != 0 || len(after.Administrators) != 0 {
		t.Fatal("binding did not invalidate authority")
	}
	if p := discordHandleTest(t, s, rt, v, "777", "confirm", nonce); p.text != discordFailure().text {
		t.Fatal("old guild authorized")
	}
	if p := discordHandleTest(t, s, rt, after, "777", "confirm", nonce); p.text != discordFailure().text {
		t.Fatal("stale nonce in new guild")
	}
	discordEventually(t, func() bool { fake.mu.Lock(); defer fake.mu.Unlock(); return len(fake.commands) == 1 })
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if fake.commands[0].guild != 222 {
		t.Fatal("old guild used for synchronization")
	}
}
func TestDiscordFactoryResetInvalidatesOutstandingConfirmations(t *testing.T) {
	s, rt, v, f := discordMinecraftFixture(t)
	discordApproveFixture(t, s, &v, "777")
	nonce := discordNonce(t, discordHandleTest(t, s, rt, v, "777", "restart", f.configs[0].ID))
	if _, _, err := s.operations.beginFactoryReset(testFactoryResetFingerprint); err != nil {
		t.Fatal(err)
	}
	if p := discordHandleTest(t, s, rt, v, "777", "confirm", nonce); p.text != discordFailure().text {
		t.Fatal("reset accepted outstanding action")
	}
	if err := s.discordFactoryRevoke(); err != nil {
		t.Fatal(err)
	}
	after, _ := s.store.discordRead()
	if after.Token != "" || len(after.Pending) != 0 || len(after.Administrators) != 0 || s.discordRuntime != nil {
		t.Fatal("reset retained authority")
	}
}

func TestDiscordRejectsStaleAndMalformedComponentIDs(t *testing.T) {
	s, rt, v := discordBoundFixture(t)
	for _, custom := range []string{"jv:9:request:", "jv:10:request:777", "jv:10:confirm:missing", "jv:10:unknown:", "jv:10:confirm:x:restart", strings.Repeat("x", 101)} {
		i := discordTestInteraction(t, v.GuildID, "777", "", custom)
		s.discordMu.Lock()
		p := s.discordHandleLocked(context.Background(), rt, i, v.GuildID, "777")
		s.discordMu.Unlock()
		if p.text != discordFailure().text {
			t.Fatal("malformed component accepted", custom)
		}
	}
}
func TestDiscordApprovedStartRequiresConfirmation(t *testing.T) {
	s, rt, v, f := discordMinecraftFixture(t)
	discordApproveFixture(t, s, &v, "777")
	id := f.configs[1].ID
	p := discordHandleTest(t, s, rt, v, "777", "start", id)
	nonce := discordNonce(t, p)
	if f.states[id] == "active" {
		t.Fatal("started before confirmation")
	}
	p = discordHandleTest(t, s, rt, v, "777", "confirm", nonce)
	if !strings.Contains(p.text, "Request accepted") || f.states[id] != "active" {
		t.Fatal("confirmed Start failed", p.text)
	}
	if p = discordHandleTest(t, s, rt, v, "777", "confirm", nonce); p.text != discordFailure().text {
		t.Fatal("Start nonce reused")
	}
}

func discordRegistryFixture(t *testing.T, registry *minecraftInstanceStore) {
	t.Helper()
	previousRegistry, previousDiscovery := minecraftInstances, runAdminDiscoveryHelper
	minecraftInstances = registry
	runAdminDiscoveryHelper = func(context.Context, string) ([]byte, error) { return []byte(`{"configured":false}`), nil }
	t.Cleanup(func() { minecraftInstances = previousRegistry; runAdminDiscoveryHelper = previousDiscovery })
}
