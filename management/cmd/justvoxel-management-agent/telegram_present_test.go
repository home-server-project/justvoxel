package main

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestTelegramHumanSummaries(t *testing.T) {
	fixtures := []struct{ operation, raw, want string }{
		{"status", `{"system":{"health":"Healthy"},"extended":{"system":{"state":"running","uptime_seconds":1020,"physical_ram_kib":24536678,"failed_count":0}}}`, "Uptime: 17 minutes"},
		{"instance_status", `{"id":"jv-abcdef","state":"running","start_on_boot":true}`, "Start on Boot: Enabled"},
		{"instance_details", `{"id":"jv-abcdef","runtime":{"software":"paper","version":"1.21.5","rcon_password":"private-rcon"}}`, "Minecraft version: 1.21.5"},
		{"players", `{"id":"jv-abcdef","state":"running","online":2,"maximum":20}`, "Online: 2 / 20"},
		{"players", `{"id":"jv-abcdef","online":0,"maximum":20}`, "No players online."},
		{"backups", `{"id":"jv-abcdef","backups":[{"created":"2026-11-15T12:30:00Z","archive":"/private/backups/world.tar.gz","sha256":"sha256:private-digest"}]}`, "Backups retained: 1"},
		{"backup_status", `{"state":"succeeded","last_success":"2026-11-15T12:30:00Z"}`, "Status: Successful"},
		{"network_status", `{"config":{"enabled":true,"members":["jv-abcdef"],"forwarding_secret":"private-forwarding"},"runtime_ready":true,"recovery_required":false}`, "Recovery required: No"},
		{"version_status", `{"installed":"1.21.4","available":"1.21.5","recommended":"1.21.5","plan_fingerprint":"sha256:private-digest"}`, "Recommended: 1.21.5"},
		{"container_status", `{"state":"update_available","image":"ghcr.io/private/image@sha256:private-digest","plan_fingerprint":"sha256:private-digest"}`, "Status: Update available"},
		{"version_update_status", `{"operation":{"state":"running","updated_at":"2026-11-15T12:30:00Z","status":"private-log"}}`, "Status: Running"},
		{"container_update_status", `{"operation":{"state":"succeeded","updated_at":"2026-11-15T12:30:00Z"}}`, "Status: Successful"},
		{"system_updates", `{"running":{"version":"11.0.0-rc.1","digest":"sha256:private-digest","image":"ghcr.io/private/image"},"check_state":"current","reboot_required":false}`, "Update: No update available"},
		{"system_update_check", `{"running":{"version":"11.0.0-rc.1"},"available":{"version":"11.0.0","digest":"sha256:private-digest"}}`, "Available: 11.0.0"},
		{"system_updates", `{"running":{"version":"11.0.0-rc.1"},"staged":{"version":"11.0.0"},"reboot_required":true}`, "Ready to install: 11.0.0"},
		{"system_update_check", `{"running":{},"check_state":"update_available"}`, "Update: Available"},
		{"system_download", `{"ok":true,"running":{"version":"11.0.0-rc.1"},"staged":{"version":"11.0.0"},"reboot_required":true}`, "OS update downloaded."},
	}
	for _, fixture := range fixtures {
		t.Run(fixture.operation+fixture.want, func(t *testing.T) {
			var payload map[string]any
			if err := json.Unmarshal([]byte(fixture.raw), &payload); err != nil {
				t.Fatal(err)
			}
			for _, key := range []string{"token", "credential", "password", "rcon_password", "logs", "path", "forwarding_secret", "sha256"} {
				payload[key] = "private-" + key + " /private/hidden ghcr.io/private/image sha256:hidden"
			}
			raw, _ := json.Marshal(payload)
			text := telegramResultText(fixture.operation, 200, raw)
			if !strings.Contains(text, fixture.want) {
				t.Fatalf("missing human summary %q: %s", fixture.want, text)
			}
			for _, secret := range []string{"{", "}", `"extended":`, `"system":`, `"start_on_boot":`, "sha256:", "ghcr.io/", "/private/", "private-", "forwarding_secret", "rcon_password"} {
				if strings.Contains(text, secret) {
					t.Fatalf("private or JSON output %q: %s", secret, text)
				}
			}
		})
	}
}
func TestTelegramSummaryFailuresNeverDumpPayload(t *testing.T) {
	for name := range (*server)(nil).telegramOperations() {
		for _, raw := range []string{`not json private-log`, `{"state":["private-log"],"token":"private-token"}`, `[]`, `null`, `{}`} {
			text := telegramResultText(name, 200, json.RawMessage(raw))
			if text != telegramUnavailable {
				t.Fatalf("%s malformed response became %q", name, text)
			}
		}
		if text := telegramResultText(name, 503, json.RawMessage(`{"error":"private-log /private/path"}`)); strings.Contains(text, "private") {
			t.Fatal(text)
		}
	}
	for _, value := range []string{"sha256:" + strings.Repeat("a", 64), strings.Repeat("a", 64), "/private/path", "ghcr.io/private/image", "jvapi_privatecredential", telegramTestToken} {
		if text := telegramHuman(value); text != "" {
			t.Fatalf("unsafe public value: %s", text)
		}
	}
}
func TestTelegramActionSummaries(t *testing.T) {
	for _, operation := range []string{"instance_start", "instance_stop", "instance_restart", "backup_now", "start_on_boot", "network_change", "version_update", "container_update", "reboot", "update_reboot"} {
		for _, code := range []int{200, 202} {
			text := telegramResultText(operation, code, json.RawMessage(`{"ok":true,"token":"private-token","path":"/private/hidden","logs":"private-log"}`))
			want := "completed."
			if code == 202 {
				want = "accepted."
			}
			if !strings.HasSuffix(text, want) || strings.Contains(text, "private") || strings.ContainsAny(text, "{}") {
				t.Fatalf("%s: %s", operation, text)
			}
		}
	}
}
func TestTelegramMenuHierarchyAndPermissions(t *testing.T) {
	main := telegramMainChoices()
	labels := []string{}
	for _, choice := range main {
		labels = append(labels, choice.Label)
		if choice.Call.InstanceID != "" {
			t.Fatal("main menu lists server")
		}
	}
	if !reflect.DeepEqual(labels, []string{"Appliance", "Minecraft servers", "Minecraft Network", "OS updates", "Notifications"}) {
		t.Fatal(labels)
	}
	for _, state := range []string{"running", "stopped"} {
		for _, boot := range []bool{true, false} {
			choices := telegramServerChoices("jv-abcdef", instanceRuntimeStatus{State: state, StartOnBoot: boot})
			counts := map[string]int{}
			v := telegramTestState()
			for _, choice := range choices {
				counts[choice.Call.Operation]++
				known, management := telegramManagementOperation(choice.Call.Operation)
				if !known || !telegramAllowed(&v, 456, choice.Call.Operation) || (management && telegramAllowed(&v, 123, choice.Call.Operation)) {
					t.Fatal("incorrect menu authorization", choice)
				}
				if choice.Call.Operation == "start_on_boot" {
					var body struct {
						Enabled bool `json:"enabled"`
					}
					if json.Unmarshal(choice.Call.Body, &body) != nil || body.Enabled == boot || choice.Label != "Start on Boot: "+telegramEnabled(boot) {
						t.Fatal("incorrect boot toggle", choice)
					}
				}
			}
			if counts["start_on_boot"] != 1 {
				t.Fatal("duplicate boot toggle")
			}
			if state == "running" && (counts["instance_start"] != 0 || counts["instance_stop"] != 1 || counts["instance_restart"] != 1) {
				t.Fatal(counts)
			}
			if state == "stopped" && (counts["instance_start"] != 1 || counts["instance_stop"] != 0 || counts["instance_restart"] != 0) {
				t.Fatal(counts)
			}
		}
	}
	for _, name := range []string{"appliance_menu", "servers_menu", "backups_menu", "updates_menu", "os_menu"} {
		known, management := telegramManagementOperation(name)
		v := telegramTestState()
		if !known || management || !telegramAllowed(&v, 123, name) {
			t.Fatal("navigation grants management or denies reader", name)
		}
	}
}
func TestTelegramRebootPresentationKeepsEvidencePrivate(t *testing.T) {
	for _, staged := range []bool{false, true} {
		v := adminSystemUpdateStatus{RebootRequired: staged}
		if staged {
			v.Staged = &adminSystemUpdateDeployment{Version: "11.0.0", Digest: "sha256:private-digest", Image: "ghcr.io/private/image"}
		}
		text := telegramRebootText(v)
		if !strings.HasPrefix(text, "Reboot JustVoxel?") || strings.Contains(text, "sha256:") || strings.Contains(text, "ghcr.io") || strings.ContainsAny(text, "{}") {
			t.Fatal(text)
		}
		if staged && !strings.Contains(text, "Update ready: 11.0.0") {
			t.Fatal(text)
		}
		if !staged && !strings.Contains(text, "No OS update is waiting for activation.") {
			t.Fatal(text)
		}
	}
}
func TestTelegramNotificationsUseHumanState(t *testing.T) {
	s := telegramTestServer(t)
	v := telegramTestState()
	old := telegramRequest
	t.Cleanup(func() { telegramRequest = old })
	for _, enabled := range []bool{false, true} {
		v.Users[0].Notifications = enabled
		var text string
		telegramRequest = func(_ context.Context, _ string, method string, input any, _ any) error {
			if method == "sendMessage" {
				text = input.(map[string]any)["text"].(string)
			}
			return nil
		}
		s.telegramDispatch(context.Background(), &v, 123, telegramCallback{Operation: "notifications"})
		if text != "Notifications: "+telegramEnabled(enabled) {
			t.Fatal(text)
		}
	}
}

func TestTelegramServersSubmenuUsesDisplayNames(t *testing.T) {
	store, _, _ := instanceRuntimeFixture(t)
	oldInstances, oldRequest := minecraftInstances, telegramRequest
	minecraftInstances = store
	t.Cleanup(func() { minecraftInstances = oldInstances; telegramRequest = oldRequest })
	s := telegramTestServer(t)
	v := telegramTestState()
	var labels []string
	telegramRequest = func(_ context.Context, _ string, method string, input any, _ any) error {
		if method == "sendMessage" {
			for _, row := range input.(map[string]any)["reply_markup"].(map[string]any)["inline_keyboard"].([][]telegramButton) {
				labels = append(labels, row[0].Text)
			}
		}
		return nil
	}
	s.telegramServersMenu(context.Background(), &v, 123)
	for _, want := range []string{"JV Paper", "JV Vanilla", "Main menu"} {
		found := false
		for _, label := range labels {
			if label == want {
				found = true
			}
		}
		if !found {
			t.Fatal("missing display name", want, labels)
		}
	}
	for _, label := range labels {
		if strings.Contains(label, "(jv-") {
			t.Fatal("unnecessary ID disambiguation", label)
		}
	}
}
func TestTelegramServerMenuLoadsAuthoritativeStateAndFiltersButtons(t *testing.T) {
	f := newInstanceBackupFixture(t)
	oldInstances, oldRequest := minecraftInstances, telegramRequest
	minecraftInstances = f.backend.store
	t.Cleanup(func() { minecraftInstances = oldInstances; telegramRequest = oldRequest })
	s := telegramTestServer(t)
	id := f.configs[0].ID
	for _, state := range []string{"running", "stopped"} {
		action := "start"
		if state == "stopped" {
			action = "stop"
		}
		if err := f.backend.action(context.Background(), id, action); err != nil {
			t.Fatal(err)
		}
		for _, userID := range []int64{123, 456} {
			labels := map[string]bool{}
			telegramRequest = func(_ context.Context, _ string, method string, input any, _ any) error {
				if method == "sendMessage" {
					for _, row := range input.(map[string]any)["reply_markup"].(map[string]any)["inline_keyboard"].([][]telegramButton) {
						labels[row[0].Text] = true
					}
				}
				return nil
			}
			v, _ := s.store.telegramRead()
			s.telegramMu.Lock()
			s.telegramInstanceMenu(context.Background(), &v, userID, id)
			s.telegramMu.Unlock()
			if !labels["Status"] || !labels["Backups"] || !labels["Minecraft servers"] {
				t.Fatal(labels)
			}
			if userID == 123 {
				if labels["Start"] || labels["Stop"] || labels["Restart"] || labels["Start on Boot: Enabled"] {
					t.Fatal("reader received management buttons", labels)
				}
			} else if state == "running" {
				if !labels["Stop"] || !labels["Restart"] || labels["Start"] {
					t.Fatal(labels)
				}
			} else if !labels["Start"] || labels["Stop"] || labels["Restart"] {
				t.Fatal(labels)
			}
		}
	}
}
