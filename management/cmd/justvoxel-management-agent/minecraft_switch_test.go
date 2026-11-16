package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSwitchStructuralPathsPreservePaperForkFamily(t *testing.T) {
	software := []string{"paper", "purpur", "spigot", "pufferfish", "leaf", "fabric", "neoforge", "forge", "quilt", "vanilla", "bedrock"}
	for _, from := range software {
		for _, to := range software {
			want := from != to && javaServerPaperFamily(from) && javaServerPaperFamily(to)
			if switchStructuralPath(from, to) != want {
				t.Fatalf("unsafe structural path %s -> %s", from, to)
			}
			if want && !javaServerAllowsSwitch(from, to) {
				t.Fatalf("existing maintenance path lost %s -> %s", from, to)
			}
		}
	}
	if switchStructuralPath("spigot", "paper") || switchStructuralPath("paper", "spigot") || switchStructuralPath("bedrock", "paper") || switchStructuralPath("paper", "bedrock") {
		t.Fatal("unimplemented world conversion allowed")
	}
}

func switchPlannerFixture(t *testing.T) *instanceActivationFixture {
	t.Helper()
	f := whitelistSurfaceFixture(t)
	previous := runAdminVersionStatus
	previousClient := pluginCatalogClient
	pluginCatalogClient = &http.Client{Transport: pluginTestTransport(func(*http.Request) (*http.Response, error) { return nil, errors.New("catalogue unavailable") })}
	t.Cleanup(func() { runAdminVersionStatus = previous; pluginCatalogClient = previousClient })
	runAdminVersionStatus = func(ctx context.Context, policy, version, target, image string) ([]byte, error) {
		if id, _ := ctx.Value(instanceVersionKey{}).(string); id != "jv-a1b2c3" {
			t.Fatal("planner lost selected immutable identity")
		}
		if target == "" {
			return json.Marshal(adminVersionStatus{Installed: "1.21.4"})
		}
		if policy != "pinned" || version != "1.21.4" {
			t.Fatalf("Switch changed game version: %s %s", policy, version)
		}
		return []byte(`{"server_type":"` + target + `","installed":"1.21.4","selected_candidate":"1.21.4","image_tag":"stable","server_supported":true,"stack_state":"updates_available","update_available":true,"plan_fingerprint":"` + testMinecraftResetFingerprint + `","managed_image_digest":"` + testMinecraftResetFingerprint + `"}`), nil
	}
	return f
}

func TestSwitchExactVersionAndImmutableSourcePlan(t *testing.T) {
	f := switchPlannerFixture(t)
	before, err := f.backend.readRecord("jv-a1b2c3")
	if err != nil {
		t.Fatal(err)
	}
	p, err := planMinecraftSwitch(context.Background(), "jv-a1b2c3", "purpur")
	if err != nil || !p.Convert || !p.Clone || p.Version != "1.21.4" || p.InstanceID != before.Runtime.ID {
		t.Fatalf("invalid safe plan: %+v %v", p, err)
	}
	after, _ := f.backend.readRecord(before.Runtime.ID)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("planning changed the source")
	}
	current, err := planMinecraftSwitch(context.Background(), before.Runtime.ID, "paper")
	if err != nil || current.Current != "paper" || current.Convert || current.Clone {
		t.Fatalf("current card selectable: %+v %v", current, err)
	}
	old := runAdminVersionStatus
	runAdminVersionStatus = func(ctx context.Context, policy, version, target, image string) ([]byte, error) {
		if target == "" {
			return old(ctx, policy, version, target, image)
		}
		return json.Marshal(adminVersionStatus{ServerType: target, Installed: "1.21.4", SelectedCandidate: "1.21.5", ServerSupported: true, StackState: "updates_available", UpdateAvailable: true, PlanFingerprint: testMinecraftResetFingerprint})
	}
	unsupported, err := planMinecraftSwitch(context.Background(), before.Runtime.ID, "purpur")
	if err != nil || unsupported.Convert || unsupported.Clone || !strings.Contains(unsupported.Reason, "never upgrades or downgrades") {
		t.Fatalf("different game version accepted: %+v %v", unsupported, err)
	}
}

func TestSwitchUnknownContentBlocksConvertAndPreservesFiles(t *testing.T) {
	f := switchPlannerFixture(t)
	record, err := f.backend.readRecord("jv-a1b2c3")
	if err != nil {
		t.Fatal(err)
	}
	plugins := filepath.Join(record.Storage.DataPath, "plugins")
	if err = os.Mkdir(plugins, 0700); err != nil {
		t.Fatal(err)
	}
	jar := filepath.Join(plugins, "example.jar")
	pluginTestJar(t, jar, map[string]string{"plugin.yml": "name: FamilyTools\nversion: 1.0\nmain: example.FamilyTools\n"})
	before, err := os.ReadFile(jar)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := planMinecraftSwitch(context.Background(), record.Runtime.ID, "purpur")
	if err != nil || plan.Convert || !plan.Clone || len(plan.Content) != 1 || plan.Content[0].Compatibility != "unknown" || plan.Content[0].Name != "FamilyTools" {
		t.Fatalf("unmanaged content accepted: %+v %v", plan, err)
	}
	after, _ := os.ReadFile(jar)
	if string(before) != string(after) {
		t.Fatal("Switch changed or removed installed content")
	}
	// Re-reading authoritative inventory changes the reviewed fingerprint.
	if err = os.WriteFile(jar, append(before, 0), 0600); err != nil {
		t.Fatal(err)
	}
	changed, err := planMinecraftSwitch(context.Background(), record.Runtime.ID, "purpur")
	if err != nil || changed.Fingerprint == plan.Fingerprint {
		t.Fatal("modified JAR did not invalidate review", err)
	}
}

func TestSwitchIncompatibleManagedReleaseNamesBlocker(t *testing.T) {
	f := switchPlannerFixture(t)
	record, _ := f.backend.readRecord("jv-a1b2c3")
	if err := os.Mkdir(filepath.Join(record.Storage.DataPath, "plugins"), 0700); err != nil {
		t.Fatal(err)
	}
	pluginTestJar(t, filepath.Join(record.Storage.DataPath, "plugins", "family.jar"), map[string]string{"plugin.yml": "name: FamilyTools\nversion: 1.0\nmain: example.FamilyTools\n"})
	root, err := openPluginDirectory(record.Storage.DataPath)
	if err != nil {
		t.Fatal(err)
	}
	hash, err := hashPluginAt(root, "family.jar")
	root.Close()
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Mkdir(filepath.Join(record.Storage.DataPath, ".justvoxel"), 0700); err != nil {
		t.Fatal(err)
	}
	manifest := pluginManifest{SchemaVersion: "v1", Artifacts: []pluginManifestEntry{{Source: "Modrinth", ProjectID: "abcdefgh", VersionID: "ijklmnop", VersionNumber: "1.0", Filename: "family.jar", SHA512: hash, PluginName: "FamilyTools", Main: "example.FamilyTools", Requested: true, RequiredBy: []string{}}}}
	raw, _ := json.Marshal(manifest)
	if err = os.WriteFile(filepath.Join(record.Storage.DataPath, ".justvoxel", "plugins.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	oldClient := pluginCatalogClient
	t.Cleanup(func() { pluginCatalogClient = oldClient })
	pluginCatalogClient = &http.Client{Transport: pluginTestTransport(func(r *http.Request) (*http.Response, error) {
		if !strings.Contains(r.URL.Path, "/version/ijklmnop") {
			t.Fatalf("unexpected provider lookup: %s", r.URL)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"id":"ijklmnop","project_id":"abcdefgh","game_versions":["1.21.5"],"loaders":["paper"],"files":[]}`)), Header: make(http.Header)}, nil
	})}
	plan, err := planMinecraftSwitch(context.Background(), record.Runtime.ID, "purpur")
	if err != nil || plan.Convert || len(plan.Content) != 1 || plan.Content[0].Compatibility != "incompatible" || plan.Content[0].Name != "FamilyTools" {
		t.Fatalf("missing incompatible blocker: %+v %v", plan, err)
	}
	_, copyFingerprint, copyErr := switchContentEvidence(context.Background(), record, "", plan.Version)
	if copyErr != nil || copyFingerprint != plan.ContentFingerprint {
		t.Fatal("read-only clone recheck changed inventory fingerprint", copyErr)
	}
}

func TestSwitchCloneCreationUsesNewIdentityPlacementAndPorts(t *testing.T) {
	f := whitelistSurfaceFixture(t)
	source, err := f.backend.readRecord("jv-a1b2c3")
	if err != nil {
		t.Fatal(err)
	}
	records, err := f.backend.records()
	if err != nil {
		t.Fatal(err)
	}
	candidate := source.Runtime
	candidate.ID = ""
	candidate.Software = "purpur"
	candidate.StartOnBoot = false
	candidate.PreserveContent = true
	if err = switchClonePorts(records, &candidate); err != nil {
		t.Fatal(err)
	}
	if candidate.JavaPort == source.Runtime.JavaPort {
		t.Fatal("source Java port reused")
	}
	id, err := f.backend.createTracked(context.Background(), candidate, "JV Paper Purpur", f.snapshot, func(record instanceRuntimeRecord) error {
		if record.Runtime.ID == source.Runtime.ID || !instanceIDPattern.MatchString(record.Runtime.ID) {
			t.Fatal("clone reused source ID")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	clone, err := f.backend.readRecord(id)
	if err != nil {
		t.Fatal(err)
	}
	after, _ := f.backend.readRecord(source.Runtime.ID)
	if !reflect.DeepEqual(source, after) || clone.Storage.Root != source.Storage.Root || clone.Storage.DataPath == source.Storage.DataPath || clone.Runtime.ID == source.Runtime.ID || clone.Runtime.Version != source.Runtime.Version {
		t.Fatal("clone changed source identity/placement/game version")
	}
	if candidate.PreserveContent != clone.Runtime.PreserveContent {
		t.Fatal("clone startup can refresh plugins")
	}
}

func TestSwitchCloneFailureNeverReportsSuccess(t *testing.T) {
	f := switchPlannerFixture(t)
	oldBackup, oldLock := instanceBackupStorageCommand, pluginRuntimeLockPath
	t.Cleanup(func() { instanceBackupStorageCommand = oldBackup; pluginRuntimeLockPath = oldLock })
	pluginRuntimeLockPath = filepath.Join(t.TempDir(), "maintenance.lock")
	instanceBackupStorageCommand = func(context.Context, string, ...string) ([]byte, error) { return nil, errors.New("unavailable backup") }
	plan, err := planMinecraftSwitch(context.Background(), "jv-a1b2c3", "purpur")
	if err != nil {
		t.Fatal(err)
	}
	s := surfaceTestServer(t, roleAdministrator)
	s.operations = openTestOperationStore(t)
	request := minecraftSwitchRequest{Target: plan.Target, Action: "clone", Fingerprint: plan.Fingerprint, Confirm: true}
	journal, _, err := s.operations.beginMinecraftMaintenance(switchActionFingerprint(request), operationTypeMinecraftSwitch, plan.InstanceID)
	if err != nil {
		t.Fatal(err)
	}
	before, _ := f.backend.readRecord(plan.InstanceID)
	s.runMinecraftSwitch(context.Background(), journal, plan, request)
	result, err := s.operations.get(journal.OperationID)
	if err != nil || result.State == operationSucceeded || result.State == operationRunning {
		t.Fatalf("failed clone reported healthy: %+v %v", result, err)
	}
	after, _ := f.backend.readRecord(plan.InstanceID)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("failed clone modified source")
	}
}

func TestVersionUpdateCannotBypassSwitchPlanner(t *testing.T) {
	whitelistSurfaceFixture(t)
	s := surfaceTestServer(t, roleAdministrator)
	s.operations = openTestOperationStore(t)
	body := `{"server_type":"purpur","plan_fingerprint":"` + testMinecraftResetFingerprint + `"}`
	response := httptest.NewRecorder()
	s.adminMinecraftStackUpdate(response, surfaceRequest(http.MethodPost, "/v1/minecraft/instances/jv-a1b2c3/version/update", body))
	if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), "Switch") {
		t.Fatalf("legacy update bypassed Switch: %d %s", response.Code, response.Body.String())
	}
}

func TestSwitchCloneAllocatesSeparateCrossplayPorts(t *testing.T) {
	source := minecraftInstanceRuntimeConfig{ID: "jv-a1b2c3", JavaPort: 25565, BedrockEnabled: true, BedrockPort: 19132}
	records := []instanceRuntimeRecord{{Runtime: source}, {Runtime: minecraftInstanceRuntimeConfig{ID: "jv-d4e5f6", JavaPort: 25566, BedrockEnabled: true, BedrockPort: 19133}}}
	clone := source
	if err := switchClonePorts(records, &clone); err != nil {
		t.Fatal(err)
	}
	if clone.JavaPort == 25565 || clone.JavaPort == 25566 || clone.BedrockPort == 19132 || clone.BedrockPort == 19133 || clone.JavaPort == clone.BedrockPort {
		t.Fatalf("unsafe clone reservations: %+v", clone)
	}
	if records[0].Runtime != source {
		t.Fatal("allocation changed source runtime")
	}
}

func TestSwitchOperationSurvivesRestartWithoutAutomaticRetry(t *testing.T) {
	store := openTestOperationStore(t)
	journal, _, err := store.beginMinecraftMaintenance(testMinecraftResetFingerprint, operationTypeMinecraftSwitch, "jv-a1b2c3")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.transition(journal.OperationID, operationValidating, "validating", "Checking Switch."); err != nil {
		t.Fatal(err)
	}
	if _, err = store.transition(journal.OperationID, operationRunning, "converting", "Converting server."); err != nil {
		t.Fatal(err)
	}
	base := store.baseDir
	if err = store.close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := openOperationStore(base)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopened.close() })
	store = reopened
	current, err := store.currentMinecraftMaintenance()
	if err != nil || current == nil || current.OperationID != journal.OperationID || current.InstanceID != "jv-a1b2c3" || current.OperationType != operationTypeMinecraftSwitch || current.State != operationNeedsAttention || current.Stage != "interrupted" {
		t.Fatalf("Switch was lost or retried after restart: %+v %v", current, err)
	}
	if operationTransitionAllowed(operationNeedsAttention, operationRunning) {
		t.Fatal("recovery can retry the requested conversion")
	}
	if _, _, err = store.beginMinecraftMaintenance("sha256:"+strings.Repeat("d", 64), operationTypeMinecraftUpdate, "jv-a1b2c3"); err == nil {
		t.Fatal("version update bypassed unresolved Switch")
	}
}

func TestSwitchRecognizesConfiguredRuntimeArtifactByPublisherHash(t *testing.T) {
	f := switchPlannerFixture(t)
	record, _ := f.backend.readRecord("jv-a1b2c3")
	if err := os.Mkdir(filepath.Join(record.Storage.DataPath, "plugins"), 0700); err != nil {
		t.Fatal(err)
	}
	pluginTestJar(t, filepath.Join(record.Storage.DataPath, "plugins", "ViaBackwards.jar"), map[string]string{"plugin.yml": "name: ViaBackwards\nversion: 1.0\nmain: com.viaversion.viabackwards.BukkitPlugin\n"})
	root, err := openPluginDirectory(record.Storage.DataPath)
	if err != nil {
		t.Fatal(err)
	}
	hash, err := hashPluginAt(root, "ViaBackwards.jar")
	root.Close()
	if err != nil {
		t.Fatal(err)
	}
	old := pluginCatalogClient
	t.Cleanup(func() { pluginCatalogClient = old })
	pluginCatalogClient = &http.Client{Transport: pluginTestTransport(func(r *http.Request) (*http.Response, error) {
		body := `{"id":"ijklmnop","project_id":"abcdefgh","game_versions":["1.21.4"],"loaders":["paper"],"environment":"server_only","dependencies":[],"files":[{"hashes":{"sha512":"` + hash + `"}}]}`
		if strings.Contains(r.URL.Path, "/project/") {
			body = `{"id":"abcdefgh","slug":"viabackwards"}`
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}
	plan, err := planMinecraftSwitch(context.Background(), record.Runtime.ID, "purpur")
	if err != nil || !plan.Convert || !plan.Clone || len(plan.Content) != 1 || plan.Content[0].Compatibility != "verified" {
		t.Fatalf("configured verified runtime artifact became unknown: %+v %v", plan, err)
	}
}

func TestSwitchRechecksContentBeforeDestructivePublication(t *testing.T) {
	f := switchPlannerFixture(t)
	plan, err := planMinecraftSwitch(context.Background(), "jv-a1b2c3", "purpur")
	if err != nil {
		t.Fatal(err)
	}
	record, _ := f.backend.readRecord(plan.InstanceID)
	if err = os.Mkdir(filepath.Join(record.Storage.DataPath, "mods"), 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(record.Storage.DataPath, "mods", "unidentified.jar")
	original := []byte("unknown loader-specific content")
	if err = os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	if err = verifySwitchContent(context.Background(), plan.InstanceID, plan.Target, plan.Version, plan.ContentFingerprint); err == nil {
		t.Fatal("content changed after review but publication was allowed")
	}
	content, err := os.ReadFile(path)
	if err != nil || string(content) != string(original) {
		t.Fatal("Switch removed or modified blocking mod", err)
	}
}

func TestSwitchSymlinkedExecutableContentRemainsUnverified(t *testing.T) {
	f := switchPlannerFixture(t)
	record, _ := f.backend.readRecord("jv-a1b2c3")
	plugins := filepath.Join(record.Storage.DataPath, "plugins")
	if err := os.Mkdir(plugins, 0700); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside.jar")
	pluginTestJar(t, outside, map[string]string{"plugin.yml": "name: Outside\nversion: 1\nmain: example.Outside\n"})
	if err := os.Symlink(outside, filepath.Join(plugins, "outside.jar")); err != nil {
		t.Fatal(err)
	}
	plan, err := planMinecraftSwitch(context.Background(), record.Runtime.ID, "purpur")
	if err != nil || plan.Convert || len(plan.Content) != 1 || plan.Content[0].Compatibility != "unknown" {
		t.Fatalf("symlinked executable content was ignored: %+v %v", plan, err)
	}
}

func TestSwitchRuntimeGuardPreservesNormalRendering(t *testing.T) {
	store, snapshot, configs := instanceRuntimeFixture(t)
	normal, err := store.renderInstanceRuntimes(snapshot, configs)
	if err != nil {
		t.Fatal(err)
	}
	configs[0].PreserveContent = true
	guarded, err := store.renderInstanceRuntimes(snapshot, configs)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(normal[0].Environment), "MODRINTH_PROJECTS=") || strings.Contains(string(guarded[0].Environment), "MODRINTH_PROJECTS=") {
		t.Fatal("Switch can download plugins at startup")
	}
	if guarded[0].ID != normal[0].ID || guarded[0].DataPath != normal[0].DataPath || string(guarded[0].Quadlet) != string(normal[0].Quadlet) {
		t.Fatal("guard changed identity, storage, ports or mount labeling")
	}
	configs[0].PreserveContent = false
	restored, err := store.renderInstanceRuntimes(snapshot, configs)
	if err != nil || string(restored[0].Environment) != string(normal[0].Environment) {
		t.Fatal("normal Version/Container startup rendering changed", err)
	}
}

func TestSwitchFailedCloneRuntimeVerificationPreservesOriginal(t *testing.T) {
	f := switchPlannerFixture(t)
	oldStorage, oldCommand, oldLock := instanceBackupStorageCommand, pluginInstallCommand, pluginRuntimeLockPath
	t.Cleanup(func() {
		instanceBackupStorageCommand = oldStorage
		pluginInstallCommand = oldCommand
		pluginRuntimeLockPath = oldLock
	})
	pluginRuntimeLockPath = filepath.Join(t.TempDir(), "maintenance.lock")
	f.players = "There are 0 of a max of 20 players online:"
	sourceID := "jv-a1b2c3"
	cloneID := ""
	cleanupCalls, verificationCalls := 0, 0
	base := f.backend.command
	f.backend.command = func(ctx context.Context, command string, args ...string) ([]byte, error) {
		if command == "podman" && len(args) > 0 && args[0] == "rm" {
			if cloneID == "" || cloneID == sourceID || !reflect.DeepEqual(args, []string{"rm", "--ignore", "--", "minecraft-" + cloneID}) {
				t.Fatal("cleanup did not target the expected failed clone", args)
			}
			if verificationCalls != 1 || f.states[cloneID] != "inactive" {
				t.Fatal("clone cleanup occurred before failed verification and normal shutdown")
			}
			cleanupCalls++
			delete(f.containers, cloneID)
			return nil, nil
		}
		if command == "nsenter" && len(args) > 4 && args[2] == instanceDeleteHelper && (cloneID == "" || args[4] != cloneID || args[4] == sourceID) {
			t.Fatal("storage cleanup did not target the expected failed clone", args)
		}
		if command == "systemctl" && len(args) > 0 {
			if args[0] == "show" && strings.Contains(strings.Join(args, " "), "--property=Job") {
				return nil, nil
			}
			if strings.Contains(strings.Join(args, " "), "backup-") {
				if args[0] == "show" {
					return []byte("LoadState=not-found\nActiveState=inactive\nJob=\n"), nil
				}
				return nil, nil
			}
		}
		return base(ctx, command, args...)
	}
	created := false
	archive := instanceBackupMetadata{Schema: 1, ID: sourceID, Software: "paper", Version: "1.21.4", Archive: "JV Paper-backup-paper-2026-03-06-0015.tar.gz"}
	instanceBackupStorageCommand = func(ctx context.Context, command string, args ...string) ([]byte, error) {
		if command != "nsenter" || len(args) < 5 {
			t.Fatal("unexpected storage call", command, args)
		}
		action, id := args[3], args[4]
		if args[2] == instanceBackupHelper {
			if id != sourceID {
				t.Fatal("source backup identity changed")
			}
			switch action {
			case "preflight":
				return nil, nil
			case "create":
				created = true
				return nil, nil
			case "inventory":
				if created {
					return json.Marshal([]instanceBackupMetadata{archive})
				}
				return []byte("[]"), nil
			}
		}
		if args[2] != instanceRestoreHelper || id == sourceID {
			t.Fatal("clone restore touched the source", args)
		}
		if action == "plan" {
			if cloneID != "" || !instanceIDPattern.MatchString(id) {
				t.Fatal("unexpected clone identity or automatic retry", id)
			}
			cloneID = id
			return json.Marshal(instanceRestoreInspection{Metadata: archive, Integrity: "verified", Fingerprint: instanceHash([]byte("clone-copy"))})
		}
		if action == "prepare" {
			raw, err := f.backend.readPrivate(filepath.Join(f.backend.root, id, "config.json"))
			if err != nil {
				return nil, err
			}
			journal := instanceRestoreJournal{Schema: 1, ID: id, Transaction: strings.Repeat("b", 32), ConfigSHA256: instanceHash(raw), Parent: []uint64{1, 2}, Original: []uint64{1, 3}, Replacement: []uint64{1, 4}, Stage: "ready_to_apply", Archive: archive.Archive, Fingerprint: args[6]}
			return nil, f.backend.writeJSON(filepath.Join(f.backend.root, id, "restore-journal.json"), journal, true)
		}
		journal, err := f.backend.restoreJournal(id)
		if err != nil {
			return nil, err
		}
		journal.Stage = map[string]string{"shutdown": "shutdown", "activate": "replacement_activated", "verifying": "runtime_verification", "completed": "completed"}[action]
		if journal.Stage == "" {
			t.Fatal("unexpected restore stage", action)
		}
		return nil, f.backend.writeJSON(filepath.Join(f.backend.root, id, "restore-journal.json"), journal, true)
	}
	verified := false
	pluginInstallCommand = func(_ context.Context, command string, args ...string) ([]byte, error) {
		if command == "systemctl" {
			id := strings.TrimSuffix(strings.TrimPrefix(args[len(args)-1], "minecraft-"), ".service")
			if args[0] == "start" {
				if id == sourceID {
					t.Fatal("stopped source started for clone")
				}
				f.states[id] = "active"
				f.containers[id] = "running"
				return nil, nil
			}
			if args[0] == "is-active" {
				return []byte(f.states[id]), nil
			}
		}
		if command == "podman" {
			if cloneID == "" || !reflect.DeepEqual(args, []string{"exec", "minecraft-" + cloneID, "rcon-cli", "list"}) {
				t.Fatal("unexpected clone readiness command", command, args)
			}
			return []byte(f.players), nil
		}
		if strings.HasSuffix(command, "verify-minecraft-stack") {
			if cloneID == "" || !reflect.DeepEqual(args, []string{"--instance", cloneID, "1.21.4"}) {
				t.Fatal("runtime verification did not target the exact-version clone", args)
			}
			verificationCalls++
			verified = true
			return nil, errors.New("target software failed exact verification")
		}
		t.Fatal("unexpected runtime command", command, args)
		return nil, nil
	}
	plan, err := planMinecraftSwitch(context.Background(), sourceID, "purpur")
	if err != nil {
		t.Fatal(err)
	}
	source, _ := f.backend.readRecord(sourceID)
	sourceContainer := f.containers[sourceID]
	s := surfaceTestServer(t, roleAdministrator)
	s.operations = openTestOperationStore(t)
	request := minecraftSwitchRequest{Target: "purpur", Action: "clone", Fingerprint: plan.Fingerprint, Confirm: true}
	journal, _, err := s.operations.beginMinecraftMaintenance(switchActionFingerprint(request), operationTypeMinecraftSwitch, sourceID)
	if err != nil {
		t.Fatal(err)
	}
	s.runMinecraftSwitch(context.Background(), journal, plan, request)
	result, err := s.operations.get(journal.OperationID)
	if err != nil || !verified || result.State == operationSucceeded || result.State == operationRunning {
		t.Fatalf("failed verification reported success: %+v verified=%v err=%v", result, verified, err)
	}
	if cleanupCalls != 1 || verificationCalls != 1 || result.State != operationResolved {
		t.Fatalf("failed clone was not cleaned up and recovered exactly once: cleanup=%d verification=%d result=%+v", cleanupCalls, verificationCalls, result)
	}
	evidence, err := readSwitchEvidence(journal)
	if err != nil || evidence.CloneID != cloneID || evidence.Stage != "recovered" {
		t.Fatalf("clone recovery evidence was not completed: %+v %v", evidence, err)
	}
	if err := f.backend.verifyCommittedSwitchState(context.Background(), sourceID); err != nil {
		t.Fatal("original was not left verified", err)
	}
	if _, err := f.backend.readRecord(cloneID); !errors.Is(err, os.ErrNotExist) && !errors.Is(err, errInstanceNotFound) {
		t.Fatal("failed clone runtime record remains", err)
	}
	after, _ := f.backend.readRecord(sourceID)
	if !reflect.DeepEqual(source, after) || f.states[sourceID] != "inactive" || f.containers[sourceID] != sourceContainer {
		t.Fatal("failed clone changed original configuration or stopped state")
	}
}
