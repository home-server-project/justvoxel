package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func assertPluginRuntimeRecovery(t *testing.T, rr *httptest.ResponseRecorder, journal operationJournal, action string) pluginRuntimeRecoveryResponse {
	t.Helper()
	var out pluginRuntimeRecoveryResponse
	if rr.Code != http.StatusConflict || json.Unmarshal(rr.Body.Bytes(), &out) != nil ||
		!out.RecoveryRequired || out.RecoveryKind != "plugin_install" || out.ID != journal.InstanceID ||
		out.RecoveryOperationID != journal.OperationID || out.RecoveryAction != action {
		t.Fatalf("wrong recovery conflict: %d %s", rr.Code, rr.Body.String())
	}
	return out
}

func TestPluginRuntimeRecoveryStartAndRecheck(t *testing.T) {
	for _, outcome := range []string{"resolved", "start fails", "stack fails", "file changes", "installed startup fails"} {
		t.Run(outcome, func(t *testing.T) {
			f := newPluginInstallFixture(t)
			journal, tx := pluginAttentionFixture(t, f)
			f.active = false
			if outcome == "installed startup fails" {
				if err := os.WriteFile(filepath.Join(f.data, "plugins", "root.jar"), f.jars["root.jar"], 0600); err != nil {
					t.Fatal(err)
				}
				if _, err := commitPluginManifest(f.data, nil, pluginManifest{SchemaVersion: "v1", Artifacts: tx.Artifacts}); err != nil {
					t.Fatal(err)
				}
				writePluginStartup(t, f.data, pluginFixtureStartup+"[Root] Error occurred while enabling Root v1.0\n")
			}

			rr := httptest.NewRecorder()
			f.s.adminPluginInstallAcknowledge(rr, pluginTestRequest(http.MethodPost, "/", `{"operation_id":"`+journal.OperationID+`"}`))
			assertPluginRuntimeRecovery(t, rr, journal, "start")
			if f.active || !slices.Equal(f.commands, []string{"systemctl is-active minecraft-jv-a1b2c3.service"}) {
				t.Fatalf("re-check changed or probed the stopped runtime: %v", f.commands)
			}

			f.failRollbackVerify = outcome == "stack fails"
			f.mutateOnVerify = outcome == "file changes"
			before := pluginInstallCommand
			var started, verified bool
			pluginInstallCommand = func(ctx context.Context, command string, args ...string) ([]byte, error) {
				if command == "systemctl" && args[0] == "start" {
					if started || !slices.Equal(args, []string{"start", "minecraft-jv-a1b2c3.service"}) {
						t.Fatalf("wrong recovery start: %v", args)
					}
					started = true
					testPluginStartupPrerequisite(t)
					if release, err := f.s.lockInstanceRuntime(); !errors.Is(err, errInstanceBusy) {
						if release != nil {
							_ = release()
						}
						t.Fatalf("normal lifecycle gained access during handoff: %v", err)
					}
					other := httptest.NewRecorder()
					f.s.adminPluginInstallAcknowledge(other, pluginTestRequest("POST", "/", `{"operation_id":"`+journal.OperationID+`"}`))
					if other.Code != http.StatusConflict || !strings.Contains(other.Body.String(), "maintenance is busy") {
						t.Fatal("another recovery entered during handoff", other.Body.String())
					}
					if outcome == "start fails" {
						return []byte("private start detail"), errors.New("start failure")
					}
				}
				if strings.HasSuffix(command, "/verify-minecraft-stack") {
					assertPluginMaintenanceBlocked(t)
					current, _ := f.s.operations.get(journal.OperationID)
					if !f.active || !started || current.State != operationNeedsAttention ||
						!slices.Contains(f.commands, "podman exec minecraft-"+tx.InstanceID+" rcon-cli list") ||
						!slices.Equal(args, []string{"--instance", tx.InstanceID, tx.MinecraftVersion}) {
						t.Fatal("verification did not run on the original instance before resolution", args, current.State)
					}
					verified = true
				}
				return before(ctx, command, args...)
			}

			rr = httptest.NewRecorder()
			f.s.adminPluginInstallRestoreRuntime(rr, pluginTestRequest("POST", "/", `{"operation_id":"`+journal.OperationID+`","restore_runtime":true}`))
			if !started || verified != (outcome != "start fails") {
				t.Fatal("wrong recovery command sequence", f.commands)
			}
			current, err := f.s.operations.get(journal.OperationID)
			if err != nil {
				t.Fatal(err)
			}
			if outcome == "resolved" {
				if rr.Code != 200 || current.State != operationResolved {
					t.Fatal(rr.Code, rr.Body.String(), current.State)
				}
				assertPluginStaging(t, f, current, false)
			} else {
				if rr.Code != 409 || current.State != operationNeedsAttention || strings.Contains(rr.Body.String(), "private start detail") {
					t.Fatal(rr.Code, rr.Body.String(), current.State)
				}
				assertPluginStaging(t, f, current, true)
			}
			for _, command := range f.commands {
				if strings.Contains(command, "backup") || strings.HasPrefix(command, "systemctl stop") {
					t.Fatal("start recovery retried installation or stopped the server", command)
				}
			}
		})
	}
}

func TestPluginRuntimeRecoveryStopRequiresPlayerConfirmationAndFinalProof(t *testing.T) {
	for _, changed := range []bool{false, true} {
		t.Run(map[bool]string{false: "resolved", true: "file changes during stop"}[changed], func(t *testing.T) {
			f := newPluginInstallFixture(t)
			f.active = false
			journal, _ := pluginAttentionFixture(t, f)
			f.active = true
			var warnings int
			before := instanceRuntimes.command
			instanceRuntimes.command = func(ctx context.Context, command string, args ...string) ([]byte, error) {
				if command == "podman" && len(args) > 3 && args[0] == "exec" && args[2] == "rcon-cli" {
					if args[1] != "minecraft-jv-a1b2c3" {
						t.Fatal("wrong player target", args)
					}
					if args[3] == "list" {
						return []byte("There are 2 of a max of 20 players online: Alex, Steve"), nil
					}
					if strings.HasPrefix(args[3], "say ") {
						warnings++
						return nil, nil
					}
				}
				if changed && command == "systemctl" && slices.Equal(args, []string{"--no-block", "stop", "--", "minecraft-jv-a1b2c3.service"}) {
					if err := os.WriteFile(filepath.Join(f.data, "plugins", "root.jar"), []byte("unowned file"), 0600); err != nil {
						t.Fatal(err)
					}
				}
				return before(ctx, command, args...)
			}
			rr := httptest.NewRecorder()
			f.s.adminPluginInstallAcknowledge(rr, pluginTestRequest("POST", "/", `{"operation_id":"`+journal.OperationID+`"}`))
			if out := assertPluginRuntimeRecovery(t, rr, journal, "stop"); !f.active || out.ConfirmationRequired {
				t.Fatal("read-only re-check changed the runtime or confirmed players")
			}
			rr = httptest.NewRecorder()
			f.s.adminPluginInstallRestoreRuntime(rr, pluginTestRequest("POST", "/", `{"operation_id":"`+journal.OperationID+`","restore_runtime":true}`))
			out := assertPluginRuntimeRecovery(t, rr, journal, "stop")
			if !out.ConfirmationRequired || out.Online != 2 || !slices.Equal(out.Players, []string{"Alex", "Steve"}) || !f.active || warnings != 0 {
				t.Fatal("player presence was not reported before disruption", out)
			}
			rr = httptest.NewRecorder()
			f.s.adminPluginInstallRestoreRuntime(rr, pluginTestRequest("POST", "/", `{"operation_id":"`+journal.OperationID+`","restore_runtime":true,"confirm_players":true}`))
			if f.active || warnings == 0 || !slices.Contains(f.commands, "systemctl --no-block stop -- minecraft-jv-a1b2c3.service") {
				t.Fatal("confirmed stop did not use the selected graceful shutdown with warnings", f.commands)
			}
			current, _ := f.s.operations.get(journal.OperationID)
			if changed {
				if rr.Code != 409 || current.State != operationNeedsAttention {
					t.Fatal("changed files permitted resolution", current.State, rr.Body.String())
				}
			} else if rr.Code != 200 || current.State != operationResolved {
				t.Fatal("safe stopped state did not resolve", current.State, rr.Body.String())
			}
			assertPluginStaging(t, f, current, changed)
		})
	}
}

func TestPluginRecoveryLifecycleRefusalAndEvidenceBinding(t *testing.T) {
	for _, scenario := range []string{"matching", "other instance", "other operation", "preflight only", "unsafe evidence", "missing original state", "wrong fingerprint", "wrong transaction instance"} {
		t.Run(scenario, func(t *testing.T) {
			f := newPluginInstallFixture(t)
			var journal operationJournal
			if scenario == "other operation" {
				var err error
				journal, _, err = f.s.operations.beginMinecraftMaintenance("sha256:"+strings.Repeat("a", 64), operationTypeMinecraftUpdate, "jv-a1b2c3")
				if err != nil {
					t.Fatal(err)
				}
				journal, err = f.s.operations.transition(journal.OperationID, operationNeedsAttention, "interrupted", "Update needs attention.")
				if err != nil {
					t.Fatal(err)
				}
			} else {
				var tx pluginTransaction
				journal, tx = pluginAttentionFixture(t, f)
				path := filepath.Join(f.s.operations.baseDir, "plugin-install", journal.OperationID, "transaction.json")
				switch scenario {
				case "preflight only":
					if err := os.Remove(path); err != nil {
						t.Fatal(err)
					}
					dir, err := openPluginEvidenceDirectory(filepath.Dir(path))
					if err != nil {
						t.Fatal(err)
					}
					defer dir.Close()
					preflight := pluginPreflight{InstanceID: tx.InstanceID, DataPath: tx.DataPath, MinecraftVersion: tx.MinecraftVersion,
						SchemaVersion: "v1", OperationID: tx.OperationID, PlanFingerprint: tx.PlanFingerprint, ProjectID: "root", VersionID: "release"}
					if err := publishPluginEvidence(dir, "preflight.json", preflight); err != nil {
						t.Fatal(err)
					}
				case "unsafe evidence":
					if err := os.Chmod(path, 0644); err != nil {
						t.Fatal(err)
					}
				case "missing original state":
					raw, _ := json.Marshal(tx)
					raw = []byte(strings.Replace(string(raw), `"originally_running":true,`, "", 1))
					if err := os.WriteFile(path, raw, 0600); err != nil {
						t.Fatal(err)
					}
				case "wrong fingerprint", "wrong transaction instance":
					if scenario == "wrong fingerprint" {
						tx.PlanFingerprint = "sha256:" + strings.Repeat("b", 64)
					} else {
						tx.InstanceID, tx.DataPath = "jv-d4e5f6", ""
					}
					raw, _ := json.Marshal(tx)
					if err := os.WriteFile(path, raw, 0600); err != nil {
						t.Fatal(err)
					}
				}
			}
			f.active = false
			id := "jv-a1b2c3"
			if scenario == "other instance" {
				id = "jv-d4e5f6"
			}
			r := pluginTestRequest("POST", "/", "{}")
			r.SetPathValue("id", id)
			rr := httptest.NewRecorder()
			f.s.instanceRuntimeAction(rr, r, "start")
			if scenario == "matching" {
				assertPluginRuntimeRecovery(t, rr, journal, "start")
				var fields map[string]any
				if err := json.Unmarshal(rr.Body.Bytes(), &fields); err != nil || len(fields) != 6 {
					t.Fatal("unexpected lifecycle recovery metadata", fields, err)
				}
			} else if rr.Code != 409 || strings.Contains(rr.Body.String(), "recovery_action") || strings.Contains(rr.Body.String(), journal.OperationID) ||
				!strings.Contains(rr.Body.String(), "maintenance requires attention") {
				t.Fatal("ordinary maintenance refusal leaked another recovery", rr.Code, rr.Body.String())
			}
			if f.active {
				t.Fatal("normal lifecycle bypassed maintenance")
			}
			if scenario != "matching" && scenario != "other operation" {
				rr = httptest.NewRecorder()
				r = pluginTestRequest("POST", "/", `{"operation_id":"`+journal.OperationID+`","restore_runtime":true}`)
				r.SetPathValue("id", id)
				f.s.selectedPluginHandler(f.s.adminPluginInstallRestoreRuntime)(rr, r)
				if rr.Code != 409 || f.active || strings.Contains(rr.Body.String(), "recovery_action") {
					t.Fatal("explicit recovery accepted unbound or unproven evidence", rr.Code, rr.Body.String())
				}
				current, _ := f.s.operations.get(journal.OperationID)
				assertPluginStaging(t, f, current, true)
			}
		})
	}
}

func TestPluginRecoveryDistinguishesLockFailures(t *testing.T) {
	for _, busy := range []bool{true, false} {
		t.Run(map[bool]string{true: "contention", false: "unsafe lock object"}[busy], func(t *testing.T) {
			f := newPluginInstallFixture(t)
			journal, _ := pluginAttentionFixture(t, f)
			want := "Minecraft maintenance is busy"
			if busy {
				testPluginMaintenanceLock(t)
			} else {
				if err := os.Mkdir(pluginRuntimeLockPath, 0700); err != nil {
					t.Fatal(err)
				}
				want = "Minecraft maintenance state could not be verified"
			}
			rr := httptest.NewRecorder()
			f.s.adminPluginInstallAcknowledge(rr, pluginTestRequest("POST", "/", `{"operation_id":"`+journal.OperationID+`"}`))
			if rr.Code != 409 || !strings.Contains(rr.Body.String(), want) || strings.Contains(rr.Body.String(), pluginRuntimeLockPath) {
				t.Fatal("wrong safe lock refusal", rr.Code, rr.Body.String())
			}
			assertPluginStaging(t, f, journal, true)
		})
	}
}

func TestPluginRuntimeRecoveryRequiresExplicitIntentAndAdministrator(t *testing.T) {
	f := newPluginInstallFixture(t)
	journal, _ := pluginAttentionFixture(t, f)
	f.active = false
	operation := `{"operation_id":"` + journal.OperationID + `"`
	for _, suffix := range []string{
		`}`, `,"restore_runtime":false}`, `,"restore_runtime":true,"action":"start"}`,
		`,"restore_runtime":true,"unit":"minecraft-jv-a1b2c3.service"}`,
		`,"restore_runtime":true,"container":"minecraft-jv-a1b2c3"}`,
		`,"restore_runtime":true,"originally_running":true}`,
		`,"restore_runtime":true,"data_path":"/private"}`,
	} {
		rr := httptest.NewRecorder()
		f.s.adminPluginInstallRestoreRuntime(rr, pluginTestRequest("POST", "/", operation+suffix))
		if rr.Code != http.StatusBadRequest || f.active {
			t.Fatal("invalid recovery intent accepted", suffix, rr.Code)
		}
	}
	for _, role := range []principalRole{roleOperator, roleViewer} {
		rr := httptest.NewRecorder()
		surfaceTestServer(t, role).adminPluginInstallRestoreRuntime(rr, pluginTestRequest("POST", "/", operation+`,"restore_runtime":true}`))
		if rr.Code != http.StatusForbidden {
			t.Fatal("unauthorized runtime recovery", role, rr.Code)
		}
	}
	assertPluginStaging(t, f, journal, true)
}
