package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
)

func advancedInteraction(t *testing.T, v discordState, user, op, id string) discord.Interaction {
	t.Helper()
	data := map[string]any{"component_type": 2, "custom_id": discordComponentID(v, op, id)}
	if op == "adv_member" {
		data["component_type"] = 3
		data["custom_id"] = discordComponentID(v, op, "")
		data["values"] = []string{id}
	}
	raw, _ := json.Marshal(map[string]any{"id": "888", "application_id": v.ApplicationID, "type": 3, "token": "never-retain-this-interaction-token", "guild_id": v.GuildID, "channel": map[string]any{"id": "999", "type": 0}, "member": map[string]any{"user": map[string]any{"id": user, "username": "friendly"}}, "data": data})
	i, err := discord.UnmarshalInteraction(raw)
	if err != nil {
		t.Fatal(err)
	}
	return i
}
func advancedHandle(t *testing.T, s *server, rt *discordRuntime, v discordState, user, op, id string) discordPanel {
	t.Helper()
	i := advancedInteraction(t, v, user, op, id)
	s.discordMu.Lock()
	defer s.discordMu.Unlock()
	return s.discordHandleLocked(context.Background(), rt, i, v.GuildID, user)
}
func advancedNonce(t *testing.T, p discordPanel, op string) string {
	t.Helper()
	raw, _ := json.Marshal(p.components)
	var rows []struct {
		Components []struct {
			ID string `json:"custom_id"`
		} `json:"components"`
	}
	if json.Unmarshal(raw, &rows) != nil {
		t.Fatal("invalid components")
	}
	for _, row := range rows {
		for _, c := range row.Components {
			parts := strings.Split(c.ID, ":")
			if len(parts) == 4 && parts[2] == op {
				return parts[3]
			}
		}
	}
	t.Fatalf("missing %s: %s (%s)", op, raw, p.text)
	return ""
}
func advancedInvoke(t *testing.T, s *server, rt *discordRuntime, v discordState, op string, a discordPendingAction) (int, []byte) {
	t.Helper()
	s.discordMu.Lock()
	defer s.discordMu.Unlock()
	return s.discordAdvancedInvokeLocked(context.Background(), rt, v.GuildID, "777", op, a)
}
func TestDiscordAdvancedAuthorityAndScope(t *testing.T) {
	s, rt, v, f := discordMinecraftFixture(t)
	id := f.configs[0].ID
	member := discordReadPanel(v, "777", []discordInstance{{ID: id, Name: "Family"}}, id)
	raw, _ := json.Marshal(member.components)
	for _, label := range []string{"Advanced", "System", "Minecraft Network"} {
		if strings.Contains(string(raw), label) {
			t.Fatal("member sees", label)
		}
	}
	ops := []string{"adv_menu", "adv_instance_details", "adv_backups", "adv_backup_status", "adv_boot", "adv_version_status", "adv_container_status", "adv_network", "adv_system", "adv_download", "adv_reboots"}
	for _, op := range ops {
		target := id
		if op == "adv_network" || op == "adv_system" || op == "adv_download" || op == "adv_reboots" {
			target = ""
		}
		if p := advancedHandle(t, s, rt, v, "777", op, target); p.text != discordFailure().text {
			t.Fatal("member advanced authority", op, p.text)
		}
	}
	discordApproveFixture(t, s, &v, "777")
	admin := discordReadPanel(v, "777", []discordInstance{{ID: id, Name: "Family"}}, id)
	raw, _ = json.Marshal(admin.components)
	for _, label := range []string{"Advanced", "System", "Minecraft Network"} {
		if !strings.Contains(string(raw), label) {
			t.Fatal("admin missing", label)
		}
	}
	if p := advancedHandle(t, s, rt, v, "777", "adv_menu", id); !strings.Contains(p.text, "Advanced") {
		t.Fatal(p.text)
	}
	for _, bad := range []string{"Family", "jv-dead00", "../config"} {
		if p := advancedHandle(t, s, rt, v, "777", "adv_instance_details", bad); p.text != discordFailure().text {
			t.Fatal("nonregistered authority", bad)
		}
	}
	for _, forbidden := range []string{"shell", "poweroff", "firmware-reboot", "restore", "retention", "network_lan", "factory_reset", "/v1/admin/system/reboot"} {
		if code, _ := advancedInvoke(t, s, rt, v, forbidden, discordPendingAction{}); code != 403 {
			t.Fatal("exposed", forbidden)
		}
	}
	reviewed := advancedHandle(t, s, rt, v, "777", "adv_boot_review", id)
	nonce := advancedNonce(t, reviewed, "adv_confirm")
	v.Administrators = nil
	discordSaveTestState(t, s, v)
	if p := advancedHandle(t, s, rt, v, "777", "adv_confirm", nonce); p.text != discordFailure().text {
		t.Fatal("revoked admin", p.text)
	}
	if code, _ := advancedInvoke(t, s, rt, v, "system_updates", discordPendingAction{}); code != 403 {
		t.Fatal("revoked read authority")
	}
}
func TestDiscordAdvancedTrustFences(t *testing.T) {
	for _, change := range []string{"generation", "guild", "factory"} {
		t.Run(change, func(t *testing.T) {
			s, rt, v, _ := discordMinecraftFixture(t)
			discordApproveFixture(t, s, &v, "777")
			old := v
			switch change {
			case "generation":
				v.Generation++
				discordSaveTestState(t, s, v)
			case "guild":
				v.GuildID = "222"
				v.Administrators = nil
				discordSaveTestState(t, s, v)
			case "factory":
				if _, _, err := s.operations.beginFactoryReset(testFactoryResetFingerprint); err != nil {
					t.Fatal(err)
				}
			}
			if p := advancedHandle(t, s, rt, old, "777", "adv_system", ""); p.text != discordFailure().text {
				t.Fatal("stale advanced interaction", change, p.text)
			}
		})
	}
	if discordGatewayIntents != gateway.IntentGuilds || len(discordCommands()) != 2 {
		t.Fatal("commands/intents expanded")
	}
}
func TestDiscordAdvancedDetailsAndStartOnBoot(t *testing.T) {
	s, rt, v, f := discordMinecraftFixture(t)
	discordApproveFixture(t, s, &v, "777")
	id := f.configs[0].ID
	before, err := f.backend.readRecord(id)
	if err != nil {
		t.Fatal(err)
	}
	p := advancedHandle(t, s, rt, v, "777", "adv_instance_details", id)
	if !strings.Contains(p.text, before.Runtime.Software) || strings.Contains(p.text, "RCON") || strings.Contains(p.text, "/var/") {
		t.Fatal("unsafe or missing details", p.text)
	}
	review := advancedHandle(t, s, rt, v, "777", "adv_boot_review", id)
	nonce := advancedNonce(t, review, "adv_confirm")
	a := rt.actions[nonce]
	if a.Operation != "start_on_boot" || !a.EnabledSet || a.PreviousEnabled != before.Runtime.StartOnBoot || a.DesiredEnabled == before.Runtime.StartOnBoot || a.InstanceID != id {
		t.Fatal("unbound boot review", a)
	}
	untouched, _ := f.backend.readRecord(id)
	if !reflect.DeepEqual(untouched, before) {
		t.Fatal("review mutated runtime")
	}
	p = advancedHandle(t, s, rt, v, "777", "adv_confirm", nonce)
	after, err := f.backend.readRecord(id)
	if err != nil {
		t.Fatal(err)
	}
	expected := before.Runtime
	expected.StartOnBoot = !expected.StartOnBoot
	if !reflect.DeepEqual(after.Runtime, expected) {
		t.Fatal("boot changed unrelated settings", p.text)
	}
	if p = advancedHandle(t, s, rt, v, "777", "adv_confirm", nonce); p.text != discordFailure().text {
		t.Fatal("nonce reused")
	}
	// A stale review cannot silently replace a newer preference.
	request := httptest.NewRequest("PUT", "http://discord/internal", nil)
	capture := httptest.NewRecorder()
	stale := before.Runtime.StartOnBoot
	s.setInstanceStartOnBoot(capture, request, id, before.Runtime.StartOnBoot, &stale)
	if capture.Code < 400 || !strings.Contains(capture.Body.String(), "changed") {
		t.Fatal("stale boot accepted", capture.Body.String())
	}
	// Telegram continues through the same helper without Discord's expected-state option.
	body := fmt.Sprintf(`{"enabled":%t}`, before.Runtime.StartOnBoot)
	request = surfaceRequest("PUT", "/v1/telegram/instances/"+id+"/boot", body)
	request.SetPathValue("id", id)
	capture = httptest.NewRecorder()
	request = request.WithContext(context.WithValue(request.Context(), gatewayPrincipalKey{}, telegramActor(456)))
	s.telegramStartOnBoot(capture, request)
	final, err := f.backend.readRecord(id)
	if err != nil || final.Runtime.StartOnBoot != before.Runtime.StartOnBoot {
		t.Fatal("Telegram shared mutation", capture.Code, err)
	}
}
func TestDiscordAdvancedMinecraftReviewAndRevalidation(t *testing.T) {
	s, rt, v, f := discordMinecraftFixture(t)
	discordApproveFixture(t, s, &v, "777")
	id := f.configs[0].ID
	oldStatus, oldWorker := runAdminVersionStatus, startMinecraftStackUpdateWorker
	t.Cleanup(func() { runAdminVersionStatus = oldStatus; startMinecraftStackUpdateWorker = oldWorker })
	fingerprint := "sha256:" + strings.Repeat("a", 64)
	calls := 0
	runAdminVersionStatus = func(ctx context.Context, policy, version, serverType, imageTag string) ([]byte, error) {
		calls++
		if ctx.Value(instanceVersionKey{}) != id || policy != "" || version != "" || serverType != "" || imageTag != "" {
			t.Error("invented selection or wrong ID")
		}
		raw, _ := json.Marshal(adminVersionStatus{Installed: "1.21", Available: "1.22", SelectedCandidate: "1.22", StackState: "updates_available", ServerSupported: true, UpdateAvailable: true, PlanFingerprint: fingerprint})
		return raw, nil
	}
	var applied *minecraftStackUpdateRequest
	startMinecraftStackUpdateWorker = func(_ *server, _ string, a minecraftStackUpdateRequest, actor session) {
		applied = &a
		if actor.Username != "discord:777" || actor.AuthSource != authSourceDiscord || actor.Role != roleAdministrator {
			t.Error("incorrect actor")
		}
	}
	p := advancedHandle(t, s, rt, v, "777", "adv_version_review", id)
	nonce := advancedNonce(t, p, "adv_confirm")
	if strings.Contains(p.text, fingerprint) || strings.Contains(string(mustAdvancedJSON(p.components)), fingerprint) || rt.actions[nonce].Fingerprint != fingerprint || applied != nil {
		t.Fatal("review authority leaked or applied")
	}
	fingerprint = "sha256:" + strings.Repeat("b", 64)
	p = advancedHandle(t, s, rt, v, "777", "adv_confirm", nonce)
	if applied != nil || !strings.Contains(p.text, "Update information changed") {
		t.Fatal("stale plan applied", p.text)
	}
	p = advancedHandle(t, s, rt, v, "777", "adv_version_review", id)
	nonce = advancedNonce(t, p, "adv_confirm")
	p = advancedHandle(t, s, rt, v, "777", "adv_confirm", nonce)
	if applied == nil || applied.PlanFingerprint != fingerprint || applied.ConfirmPlayers || calls < 4 {
		t.Fatal("authoritative update missing", p.text)
	}
	p = advancedHandle(t, s, rt, v, "777", "adv_version_update_status", id)
	if !strings.Contains(p.text, "Update progress") || strings.Contains(p.text, fingerprint) {
		t.Fatal("progress bypass or leak", p.text)
	}
}
func mustAdvancedJSON(value any) []byte { raw, _ := json.Marshal(value); return raw }
func TestDiscordAdvancedContainerReviewDefaults(t *testing.T) {
	s, rt, v, f := discordMinecraftFixture(t)
	discordApproveFixture(t, s, &v, "777")
	id := f.configs[0].ID
	oldStatus, oldWorker := runMinecraftContainerStatus, startMinecraftContainerUpdateWorker
	t.Cleanup(func() { runMinecraftContainerStatus = oldStatus; startMinecraftContainerUpdateWorker = oldWorker })
	fingerprint := "sha256:" + strings.Repeat("c", 64)
	runMinecraftContainerStatus = func(_ context.Context, instance string) (minecraftContainerStatus, error) {
		if instance != id {
			t.Error("wrong instance")
		}
		return minecraftContainerStatus{InstanceID: id, Running: true, State: "update_available", Image: "/private/image", PlanFingerprint: fingerprint}, nil
	}
	var applied *minecraftContainerUpdateRequest
	startMinecraftContainerUpdateWorker = func(_ *server, _ string, a minecraftContainerUpdateRequest, _ session) { applied = &a }
	p := advancedHandle(t, s, rt, v, "777", "adv_container_review", id)
	nonce := advancedNonce(t, p, "adv_confirm")
	if applied != nil || !strings.Contains(p.text, "Backup before restart: yes") || strings.Contains(p.text, "/private") || rt.actions[nonce].Fingerprint != fingerprint {
		t.Fatal("container review", p.text)
	}
	fingerprint = "sha256:" + strings.Repeat("d", 64)
	p = advancedHandle(t, s, rt, v, "777", "adv_confirm", nonce)
	if applied != nil || !strings.Contains(p.text, "changed") {
		t.Fatal("stale container plan", p.text)
	}
	p = advancedHandle(t, s, rt, v, "777", "adv_container_review", id)
	nonce = advancedNonce(t, p, "adv_confirm")
	p = advancedHandle(t, s, rt, v, "777", "adv_confirm", nonce)
	if applied == nil || !applied.RestartNow || !applied.BackupBeforeRestart || applied.ConfirmPlayers || applied.RestoreDigest != "" || applied.PlanFingerprint != fingerprint {
		t.Fatal("container defaults", p.text)
	}
	p = advancedHandle(t, s, rt, v, "777", "adv_container_update_status", id)
	if !strings.Contains(p.text, "Update progress") {
		t.Fatal(p.text)
	}
}
func TestDiscordAdvancedBackupReadsAreBounded(t *testing.T) {
	f := newInstanceBackupFixture(t)
	s, rt, v := discordBoundFixture(t)
	s.operations = f.server.operations
	discordRegistryFixture(t, f.backend.identities())
	discordApproveFixture(t, s, &v, "777")
	id := f.configs[0].ID
	base := f.backend.command
	f.backend.command = func(ctx context.Context, command string, args ...string) ([]byte, error) {
		if command == "nsenter" && len(args) >= 5 && args[2] == instanceBackupHelper && args[3] == "inventory" {
			entries := []instanceBackupMetadata{}
			for n := 0; n < 12; n++ {
				entries = append(entries, instanceBackupMetadata{Name: fmt.Sprintf("Backup %02d", n), Created: fmt.Sprintf("2026-11-%02d", n+1), Archive: "/private/archive", Size: int64(n + 1)})
			}
			return json.Marshal(entries)
		}
		return base(ctx, command, args...)
	}
	p := advancedHandle(t, s, rt, v, "777", "adv_backups", id)
	if strings.Count(p.text, "bytes") != 10 || !strings.Contains(p.text, "Backup 11") || strings.Contains(p.text, "Backup 00") || strings.Contains(p.text, "/private") {
		t.Fatal("inventory bounds", p.text)
	}
	p = advancedHandle(t, s, rt, v, "777", "adv_backup_status", id)
	if !strings.Contains(p.text, "Last success:") {
		t.Fatal("backup status adapter", p.text)
	}
}
func TestDiscordAdvancedNetworkFingerprintAndMutation(t *testing.T) {
	s, rt, v, f := discordMinecraftFixture(t)
	discordApproveFixture(t, s, &v, "777")
	n, err := networkRead(f.backend)
	if err != nil {
		t.Fatal(err)
	}
	legacy := n.Config
	legacy.WhitelistEnforced = nil
	explicit := legacy
	enabled := true
	explicit.WhitelistEnforced = &enabled
	if minecraftNetworkFingerprint(legacy) != telegramNetworkFingerprint(explicit) {
		t.Fatal("legacy whitelist fingerprint diverges")
	}
	p := advancedHandle(t, s, rt, v, "777", "adv_network_enable", "")
	nonce := advancedNonce(t, p, "adv_confirm")
	if rt.actions[nonce].NetworkExpected != minecraftNetworkFingerprint(n.Config) || !rt.actions[nonce].DesiredEnabled || strings.Contains(string(mustAdvancedJSON(p.components)), rt.actions[nonce].NetworkExpected) {
		t.Fatal("network fingerprint binding")
	}
	// The shared helper rejects stale evidence before compatibility/publication.
	if err = s.changeMinecraftNetwork(context.Background(), &enabled, nil, "stale", true); !errors.Is(err, errMinecraftNetworkReviewChanged) {
		t.Fatal("stale config accepted", err)
	}
	// Running affected instances still cannot be changed through either adapter.
	members := []string{f.configs[0].ID}
	if err = s.changeMinecraftNetwork(context.Background(), nil, &members, minecraftNetworkFingerprint(n.Config), false); err == nil || !strings.Contains(err.Error(), "stop affected servers") {
		t.Fatal("Telegram stopped rule lost", err)
	}
	p = advancedHandle(t, s, rt, v, "777", "adv_member", f.configs[0].ID)
	if strings.Contains(p.text, "Add") {
		nonce = advancedNonce(t, p, "adv_confirm")
		a := rt.actions[nonce]
		if a.Operation != "network_change" || len(a.NetworkMembers) != 1 || a.NetworkMembers[0] != f.configs[0].ID {
			t.Fatal("single-member selection")
		}
	}
}

type advancedREST struct {
	*discordFakeREST
	messages chan discord.MessageCreate
	fail     bool
}

func (f *advancedREST) CreateMessage(_ snowflake.ID, m discord.MessageCreate, _ ...rest.RequestOpt) (*discord.Message, error) {
	f.messages <- m
	if f.fail {
		return nil, errors.New("private REST error")
	}
	return nil, nil
}
func advancedOSFixture(t *testing.T) (*server, *discordRuntime, discordState, *advancedREST) {
	t.Helper()
	s, rt, v := discordBoundFixture(t)
	discordApproveFixture(t, s, &v, "777")
	isolateSystemUpdateLock(t)
	old := runBootcStatus
	t.Cleanup(func() { runBootcStatus = old })
	runBootcStatus = func(context.Context) ([]byte, error) { return []byte(bootcStatusStagedFixture), nil }
	fake := &advancedREST{discordFakeREST: newDiscordFakeREST(), messages: make(chan discord.MessageCreate, 8)}
	rt.api = fake
	return s, rt, v, fake
}
func TestDiscordAdvancedOSReadsAndRebootChoices(t *testing.T) {
	s, rt, v, _ := advancedOSFixture(t)
	old := runBootcUpgradeCheck
	t.Cleanup(func() { runBootcUpgradeCheck = old })
	checks := 0
	runBootcUpgradeCheck = func(context.Context) ([]byte, error) { checks++; return []byte("private registry diagnostics"), nil }
	for _, op := range []string{"adv_system_updates", "adv_system_update_check"} {
		p := advancedHandle(t, s, rt, v, "777", op, "")
		if !strings.Contains(p.text, "Running: 10") || strings.Contains(p.text, "private") {
			t.Fatal(op, p.text)
		}
	}
	if checks != 1 {
		t.Fatal("authoritative check not called")
	}
	p := advancedHandle(t, s, rt, v, "777", "adv_reboots", "")
	choices := map[string]bool{}
	ordinary := 0
	for _, a := range rt.actions {
		switch a.Operation {
		case "reboot":
			ordinary++
		case "update_reboot":
			choices[fmt.Sprintf("%d:%t", a.WarningSeconds, a.BackupMinecraft)] = true
			if a.StagedDigest != "sha256:"+strings.Repeat("b", 64) || a.PlayersConfirmed {
				t.Fatal("unbound reboot")
			}
		}
	}
	if ordinary != 1 || len(choices) != 4 {
		t.Fatal("reboot choices", p.text, choices)
	}
	raw := string(mustAdvancedJSON(p.components))
	for _, bad := range []string{"sha256:", "poweroff", "firmware", "Factory Reset"} {
		if strings.Contains(raw, bad) {
			t.Fatal("unsafe reboot option", bad)
		}
	}
	runBootcStatus = func(context.Context) ([]byte, error) { return []byte(bootcStatusCurrentFixture), nil }
	rt.actions = nil
	p = advancedHandle(t, s, rt, v, "777", "adv_reboots", "")
	for _, a := range rt.actions {
		if a.Operation == "update_reboot" {
			t.Fatal("unstaged update reboot")
		}
	}
}
func TestDiscordAdvancedDownloadReturnsWithoutInteractionToken(t *testing.T) {
	s, rt, v, fake := advancedOSFixture(t)
	old := runBootcUpgrade
	t.Cleanup(func() { runBootcUpgrade = old })
	started, release := make(chan struct{}), make(chan struct{})
	runBootcUpgrade = func(ctx context.Context) ([]byte, error) {
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) > 15*time.Minute {
			t.Error("unbounded update")
		}
		close(started)
		select {
		case <-release:
			return []byte("PRIVATE BOOTC OUTPUT /private/path"), nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	p := advancedHandle(t, s, rt, v, "777", "adv_download", "")
	nonce := advancedNonce(t, p, "adv_confirm")
	select {
	case <-started:
		t.Fatal("download before confirmation")
	default:
	}
	returned := make(chan discordPanel, 1)
	go func() { returned <- advancedHandle(t, s, rt, v, "777", "adv_confirm", nonce) }()
	select {
	case p = <-returned:
		if !strings.Contains(p.text, "OS update download started") {
			t.Fatal(p.text)
		}
	case <-time.After(time.Second):
		close(release)
		t.Fatal("interaction blocked on download")
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		close(release)
		t.Fatal("handler not invoked")
	}
	close(release)
	select {
	case message := <-fake.messages:
		if message.AllowedMentions == nil || !reflect.DeepEqual(*message.AllowedMentions, discord.AllowedMentions{}) || strings.Contains(message.Content, "PRIVATE") {
			t.Fatal("unsafe completion", message)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("missing best-effort result")
	}
	var raw string
	_ = s.store.db.QueryRow(`SELECT record_json FROM discord_state`).Scan(&raw)
	if strings.Contains(raw, "never-retain") {
		t.Fatal("interaction token persisted")
	}
}
func TestDiscordAdvancedSourceBoundary(t *testing.T) {
	for _, name := range []string{"discord_advanced.go", "discord_reboot.go"} {
		path := filepath.Join(repositoryRoot(t), "management/cmd/justvoxel-management-agent", name)
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range file.Imports {
			if imp.Path.Value == `"os/exec"` {
				t.Fatal("direct process execution")
			}
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
				switch sel.Sel.Name {
				case "Command", "CommandContext", "telegramInvoke", "telegramStartOnBoot", "telegramNetworkChange":
					t.Error("forbidden Stage 4 call", sel.Sel.Name)
				}
			}
			return true
		})
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, bad := range []string{"systemctl", "podman", "bootc upgrade", "/v1/telegram/", "RestoreDigest:", "image-retention"} {
			if strings.Contains(string(raw), bad) {
				t.Fatal("forbidden advanced surface", name, bad)
			}
		}
	}
}

func TestDiscordAdvancedNetworkCandidateBound(t *testing.T) {
	instances := []discordInstance{}
	eligible := map[string]bool{}
	for index := 0; index < 27; index++ {
		id := fmt.Sprintf("jv-%06x", index)
		instances = append(instances, discordInstance{ID: id, Name: fmt.Sprintf("Family %02d", index)})
		eligible[id] = true
	}
	options, total := discordNetworkMemberOptions(instances, eligible)
	if total != 27 || len(options) != 25 {
		t.Fatal("network candidates unbounded", total, len(options))
	}
	raw, _ := json.Marshal(options)
	var visible []struct {
		Value string `json:"value"`
	}
	_ = json.Unmarshal(raw, &visible)
	for index, item := range visible {
		if item.Value != instances[index].ID {
			t.Fatal("candidate order changed")
		}
	}
}
func TestDiscordAdvancedDownloadKeepsExistingConcurrencyAuthority(t *testing.T) {
	s, rt, v, fake := advancedOSFixture(t)
	old := runBootcUpgrade
	t.Cleanup(func() { runBootcUpgrade = old })
	called := false
	runBootcUpgrade = func(context.Context) ([]byte, error) { called = true; return nil, nil }
	s.systemUpdateMu.Lock()
	p := advancedHandle(t, s, rt, v, "777", "adv_download", "")
	p = advancedHandle(t, s, rt, v, "777", "adv_confirm", advancedNonce(t, p, "adv_confirm"))
	select {
	case m := <-fake.messages:
		if strings.Contains(m.Content, "completed.") {
			t.Error("conflict presented as success")
		}
	case <-time.After(time.Second):
		t.Error("existing update lock bypassed or blocked")
	}
	s.systemUpdateMu.Unlock()
	if called {
		t.Fatal("Discord bypassed authoritative update lock")
	}
}

func TestDiscordAdvancedNetworkSharedTransactionAndBootProtection(t *testing.T) {
	f, config := networkTransactionFixture(t)
	s, _ := networkLockFixture(t, f)
	bound, rt, v := discordBoundFixture(t)
	s.store = bound.store
	s.discordRuntime = rt
	discordRegistryFixture(t, f.backend.identities())
	discordApproveFixture(t, s, &v, "777")
	config.Enabled = false
	if err := networkApply(context.Background(), f.backend, config); err != nil {
		t.Fatal(err)
	}
	// Enable through a reviewed Discord action and the existing Network transaction.
	p := advancedHandle(t, s, rt, v, "777", "adv_network_enable", "")
	p = advancedHandle(t, s, rt, v, "777", "adv_confirm", advancedNonce(t, p, "adv_confirm"))
	current, err := networkRead(f.backend)
	if err != nil || !current.Config.Enabled {
		t.Fatal("Discord enable", p.text, err)
	}
	id := config.Members[0]
	review := advancedHandle(t, s, rt, v, "777", "adv_boot_review", id)
	result := advancedHandle(t, s, rt, v, "777", "adv_confirm", advancedNonce(t, review, "adv_confirm"))
	if strings.Contains(result.text, "Change accepted") {
		t.Fatal("Network boot ownership bypassed")
	}
	// Discord adds its explicit disabled admission under the shared mutation lock.
	members := []string{}
	if err = s.changeMinecraftNetwork(context.Background(), nil, &members, minecraftNetworkFingerprint(current.Config), true); err == nil || !strings.Contains(err.Error(), "Disable Minecraft Network") {
		t.Fatal("enabled membership accepted", err)
	}
	p = advancedHandle(t, s, rt, v, "777", "adv_network_disable", "")
	p = advancedHandle(t, s, rt, v, "777", "adv_confirm", advancedNonce(t, p, "adv_confirm"))
	current, err = networkRead(f.backend)
	if err != nil || current.Config.Enabled {
		t.Fatal("Discord disable", p.text, err)
	}
	f.states[id], f.containers[id] = "inactive", "exited"
	if err = s.changeMinecraftNetwork(context.Background(), nil, &members, minecraftNetworkFingerprint(current.Config), true); err != nil {
		t.Fatal("member removal", err)
	}
	current, err = networkRead(f.backend)
	if err != nil || len(current.Config.Members) != 0 {
		t.Fatal("member removal not committed", err)
	}
	members = []string{id}
	body, _ := json.Marshal(map[string]any{"members": members, "expected": telegramNetworkFingerprint(current.Config)})
	request := surfaceRequest("PUT", "/v1/telegram/network", string(body))
	request = request.WithContext(context.WithValue(request.Context(), gatewayPrincipalKey{}, telegramActor(456)))
	capture := httptest.NewRecorder()
	s.telegramNetworkChange(capture, request)
	current, err = networkRead(f.backend)
	if err != nil || !reflect.DeepEqual(current.Config.Members, members) || capture.Code != 200 {
		t.Fatal("Telegram shared member mutation", capture.Code, capture.Body.String(), err)
	}
}
