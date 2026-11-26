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
	"time"
)

func TestSwitchTransitionMatrix(t *testing.T) {
	cases := []struct {
		name, from, to, content         string
		runtime, layout, convert, clone bool
	}{
		{"paper purpur", "paper", "purpur", "", true, true, true, true},
		{"purpur leaf", "purpur", "leaf", "", true, true, true, true},
		{"vanilla paper", "vanilla", "paper", "", true, true, true, true},
		{"vanilla purpur", "vanilla", "purpur", "", true, true, true, true},
		{"vanilla fabric", "vanilla", "fabric", "", true, true, true, true},
		{"vanilla quilt", "vanilla", "quilt", "", true, true, true, true},
		{"vanilla forge", "vanilla", "forge", "", true, true, true, true},
		{"vanilla neoforge", "vanilla", "neoforge", "", true, true, true, true},
		{"paper vanilla", "paper", "vanilla", "", true, true, true, true},
		{"paper missing layout", "paper", "vanilla", "", true, false, false, false},
		{"fabric quilt compatible", "fabric", "quilt", "verified", true, true, true, true},
		{"fabric quilt incompatible", "fabric", "quilt", "incompatible", true, true, false, true},
		{"fabric quilt unknown", "fabric", "quilt", "unknown", true, true, false, true},
		{"quilt only mod", "quilt", "fabric", "incompatible", true, true, false, true},
		{"forge neo compatible", "forge", "neoforge", "verified", true, true, true, true},
		{"forge neo incompatible", "forge", "neoforge", "incompatible", true, true, false, true},
		{"forge neo unknown", "forge", "neoforge", "unknown", true, true, false, true},
		{"neo forge compatible", "neoforge", "forge", "verified", true, true, true, true},
		{"neo forge incompatible", "neoforge", "forge", "incompatible", true, true, false, true},
		{"fabric forge compatible", "fabric", "forge", "verified", true, true, true, true},
		{"fabric forge unknown", "fabric", "forge", "unknown", true, true, false, true},
		{"empty modded world", "fabric", "paper", "", true, true, false, true},
		{"modded vanilla", "neoforge", "vanilla", "", true, true, false, true},
		{"paper loader layout", "paper", "fabric", "", true, true, true, true},
		{"paper loader no layout", "paper", "fabric", "", true, false, false, false},
		{"paper loader plugins", "paper", "fabric", "incompatible", true, true, false, true},
		{"paper spigot", "paper", "spigot", "", true, true, false, false},
		{"spigot paper 26.1+", "spigot", "paper", "", true, true, false, false},
		{"java bedrock", "paper", "bedrock", "", true, true, false, false},
		{"bedrock java", "bedrock", "paper", "", true, true, false, false},
		{"missing exact runtime", "paper", "pufferfish", "", false, true, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var content []switchContent
			if tc.content != "" {
				content = []switchContent{{Name: "Named blocker", Filename: "mod.jar", Compatibility: tc.content}}
			}
			convert, clone := switchTransitionActions(tc.from, tc.to, tc.runtime, tc.layout, content)
			if convert != tc.convert || clone != tc.clone {
				t.Fatalf("got convert=%v clone=%v", convert, clone)
			}
			if tc.clone && !tc.convert && !(clone || convert) {
				t.Fatal("protected clone must remain Available")
			}
		})
	}
}

func switchSourceFixture(t *testing.T, from string) *instanceActivationFixture {
	t.Helper()
	f := newInstanceActivationFixtureFor(t, from, "1.21.4")
	// Configure the source before its first activation; this is fixture creation,
	// not an ordinary Version operation on an existing server.
	record, err := f.backend.readRecord("jv-a1b2c3")
	if err != nil {
		t.Fatal(err)
	}
	record.Runtime.VersionPolicy, record.Runtime.BedrockEnabled = "pinned", false
	if caps, _ := javaServerSoftware(from); caps.ModLoader {
		record.Runtime.LoaderVersion, record.Runtime.ImageTag = "0.16.10", "java21"
	}
	if err = f.backend.writeJSON(filepath.Join(f.backend.root, record.Runtime.ID, "config.json"), record, true); err != nil {
		t.Fatal(err)
	}
	oldBackend := instanceRuntimes
	instanceRuntimes = f.backend
	t.Cleanup(func() { instanceRuntimes = oldBackend })
	if err = f.apply(record.Runtime.ID); err != nil {
		t.Fatal(err)
	}
	return f
}

func switchPlannerFixture(t *testing.T, software ...string) *instanceActivationFixture {
	t.Helper()
	var f *instanceActivationFixture
	if len(software) == 0 {
		f = whitelistSurfaceFixture(t)
	} else {
		f = switchSourceFixture(t, software[0])
	}
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
		caps, _ := javaServerSoftware(target)
		loader, tag := "", "stable"
		if caps.ModLoader {
			loader, tag = "0.16.10", "java21"
		}
		return []byte(`{"server_type":"` + target + `","installed":"1.21.4","selected_candidate":"1.21.4","image_tag":"` + tag + `","loader_version":"` + loader + `","server_supported":true,"stack_state":"updates_available","update_available":true,"plan_fingerprint":"` + testMinecraftResetFingerprint + `","managed_image_digest":"` + testMinecraftResetFingerprint + `"}`), nil
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
	oldBackup, oldLock, oldDiagnostic := instanceBackupStorageCommand, pluginRuntimeLockPath, runFailedUnitCommand
	t.Cleanup(func() {
		instanceBackupStorageCommand, pluginRuntimeLockPath, runFailedUnitCommand = oldBackup, oldLock, oldDiagnostic
	})
	runFailedUnitCommand = func(context.Context, string, ...string) ([]byte, error) {
		t.Fatal("Switch without a clone requested clone runtime evidence")
		return nil, nil
	}
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
	if err != nil || result.State != operationResolved {
		t.Fatalf("early clone failure was not recovered: %+v %v", result, err)
	}
	after, _ := f.backend.readRecord(plan.InstanceID)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("failed clone modified source")
	}
	log := switchTestDiagnostic(t, s.operations, journal.OperationID)
	if !strings.Contains(log, "Instance backup storage") || !strings.Contains(log, `result="succeeded"`) ||
		strings.Contains(log, "clone_id=") || strings.Contains(result.Status, "Instance backup storage") {
		t.Fatal("early failure detail was lost, exposed in status, or attributed to a clone")
	}
	response := switchTestOperation(t, s, plan.InstanceID, "?operation_id="+journal.OperationID)
	if response["target"] != plan.Target || response["action"] != "clone" || response["version"] != plan.Version {
		t.Fatal("early failed Switch lost its durable metadata", response)
	}
	if _, exists := response["clone_id"]; exists {
		t.Fatal("early failed Switch invented a clone ID")
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

func TestSwitchCloneRuntimeVerificationAndRecovery(t *testing.T) {
	for _, tc := range []struct {
		name, from, target string
		success, interrupt bool
	}{
		{name: "Paper to Purpur failure", from: "paper", target: "purpur"},
		{name: "Paper to Vanilla failure", from: "paper", target: "vanilla"},
		{name: "Paper to Fabric failure", from: "paper", target: "fabric"},
		{name: "Vanilla to Purpur Java only", from: "vanilla", target: "purpur", success: true},
		{name: "retry interrupted cleanup", from: "paper", target: "purpur", interrupt: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			target := tc.target
			var f *instanceActivationFixture
			if tc.from == "paper" {
				f = switchPlannerFixture(t)
			} else {
				f = switchPlannerFixture(t, tc.from)
			}
			oldStorage, oldCommand, oldLock, oldDiagnostic := instanceBackupStorageCommand, pluginInstallCommand, pluginRuntimeLockPath, runFailedUnitCommand
			t.Cleanup(func() {
				instanceBackupStorageCommand = oldStorage
				pluginInstallCommand = oldCommand
				pluginRuntimeLockPath = oldLock
				runFailedUnitCommand = oldDiagnostic
			})
			pluginRuntimeLockPath = filepath.Join(t.TempDir(), "maintenance.lock")
			f.players = "There are 0 of a max of 20 players online:"
			sourceID := "jv-a1b2c3"
			sourceRecord, readErr := f.backend.readRecord(sourceID)
			if readErr != nil {
				t.Fatal(readErr)
			}
			switchWorldFixture(t, sourceRecord.Storage.DataPath, "1.21.4", "Family")
			originalWorld := switchWorldSnapshot(t, sourceRecord.Storage.DataPath)
			cloneID := ""
			cleanupCalls, verificationCalls := 0, 0
			var diagnostics []string
			var rcon string
			runFailedUnitCommand = func(ctx context.Context, command string, args ...string) ([]byte, error) {
				if cloneID == "" || verificationCalls != 1 || cleanupCalls != 0 {
					t.Fatal("runtime evidence must be captured for the allocated failed clone before cleanup")
				}
				deadline, bounded := ctx.Deadline()
				if !bounded || time.Until(deadline) > 10*time.Second {
					t.Fatal("runtime evidence command has no short deadline")
				}
				diagnostics = append(diagnostics, command)
				unit := "minecraft-" + cloneID + ".service"
				switch command {
				case "systemctl":
					if !reflect.DeepEqual(args, []string{"show", "--no-pager", "--property=LoadState,ActiveState,SubState,Result,ExecMainCode,ExecMainStatus", "--", unit}) {
						t.Fatal("unexpected service evidence command", args)
					}
					if target == "vanilla" {
						return nil, errors.New("mock command failure")
					}
					return []byte("LoadState=loaded\nResult=exit-code\n"), nil
				case "journalctl":
					if !reflect.DeepEqual(args, []string{"-b", "-u", unit, "--no-pager", "-n", "80"}) {
						t.Fatal("unexpected journal evidence command", args)
					}
					if target == "vanilla" {
						return []byte(strings.Repeat("oversized output", failedUnitOutputLimit)), nil
					}
					clone, err := f.backend.readRecord(cloneID)
					if err != nil {
						t.Fatal(err)
					}
					rcon = clone.Runtime.RCONPassword
					return []byte("clone startup diagnostic\npassword=journal-secret\nopaque " + rcon + "\n"), nil
				default:
					t.Fatal("unexpected evidence command", command)
					return nil, nil
				}
			}
			base := f.backend.command
			blockCleanup := tc.interrupt
			imageChecks := 0
			f.backend.command = func(ctx context.Context, command string, args ...string) ([]byte, error) {
				if command == "podman" && (len(args) == 5 && args[0] == "image" && args[1] == "inspect" ||
					len(args) == 4 && args[0] == "inspect" && args[3] == "{{.Image}}") {
					if !tc.success || f.states[cloneID] != "active" {
						t.Fatal("unexpected clone image proof", args)
					}
					if args[0] == "image" && args[2] != "docker.io/itzg/minecraft-server@"+testMinecraftResetFingerprint ||
						args[0] == "inspect" && args[1] != "minecraft-"+cloneID {
						t.Fatal("clone image proof used an unexpected binding", args)
					}
					imageChecks++
					return []byte("verified-image-id\n"), nil
				}
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
				if command == "nsenter" && len(args) > 4 {
					if args[2] == instanceDeleteHelper {
						if cloneID == "" || args[4] != cloneID || args[4] == sourceID {
							t.Fatal("storage cleanup did not target the expected failed clone", args)
						}
						out, err := base(ctx, command, args...)
						if err == nil && args[3] == "remove" && os.Geteuid() != 0 {
							// Unprivileged test runs exercise the already-absent parent path.
							parent := filepath.Dir(args[6])
							if _, statErr := os.Lstat(parent); statErr == nil {
								err = os.Remove(parent)
							}
						}
						return out, err
					}
					if args[2] == instanceRuntimeHelper && args[3] == "restore-validate" && args[4] == cloneID {
						if blockCleanup {
							return nil, errors.New("clone storage temporarily unavailable")
						}
						return nil, nil
					}
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
			archive := instanceBackupMetadata{Schema: 1, ID: sourceID, Software: tc.from, Version: "1.21.4", Archive: "Source-backup-" + tc.from + "-2026-03-06-0015.tar.gz"}
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
					if len(args) != 9 || args[8] != sourceID {
						t.Fatal("clone restore omitted its source binding", args)
					}
					return json.Marshal(instanceRestoreInspection{Metadata: archive, CurrentSoftware: target, CurrentVersion: archive.Version, Integrity: "verified", Fingerprint: instanceHash([]byte("clone-copy"))})
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
				if action == "activate" {
					clone, readErr := f.backend.readRecord(id)
					if readErr != nil {
						return nil, readErr
					}
					if copyErr := os.CopyFS(clone.Storage.DataPath, os.DirFS(sourceRecord.Storage.DataPath)); copyErr != nil {
						return nil, copyErr
					}
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
					if cloneID == "" || !reflect.DeepEqual(args, []string{"--instance", cloneID, "--switch-clone", "1.21.4"}) {
						t.Fatal("runtime verification did not target the exact-version clone", args)
					}
					if switchNeedsWorldMove(tc.from, target) {
						clone, readErr := f.backend.readRecord(cloneID)
						if readErr != nil {
							t.Fatal(readErr)
						}
						relocated := filepath.Join(clone.Storage.DataPath, "Family", "DIM-1", "region", "r.0.0.mca")
						raw, readErr := os.ReadFile(relocated)
						if readErr != nil || string(raw) != "nether chunks" {
							t.Fatal("target readiness ran before verified relocation", readErr)
						}
						if writeErr := os.WriteFile(relocated, []byte("failed target changed chunks"), 0644); writeErr != nil {
							t.Fatal(writeErr)
						}
					}
					verificationCalls++
					verified = true
					if tc.success {
						clone, err := f.backend.readRecord(cloneID)
						if err != nil || clone.Runtime.BedrockEnabled || clone.Runtime.Software != "purpur" || clone.Runtime.Version != "1.21.4" || f.states[cloneID] != "active" {
							t.Fatal("Java-only target did not start at the planned version", err)
						}
						return []byte("Minecraft version: Purpur 1.21.4\n"), nil
					}
					return []byte("verification stopped at internal-node.example\ntoken=verification-secret\n"), errors.New("target software failed exact verification")
				}
				t.Fatal("unexpected runtime command", command, args)
				return nil, nil
			}
			plan, err := planMinecraftSwitch(context.Background(), sourceID, target)
			if err != nil {
				t.Fatal(err)
			}
			source, _ := f.backend.readRecord(sourceID)
			sourceContainer := f.containers[sourceID]
			s := surfaceTestServer(t, roleAdministrator)
			s.operations = openTestOperationStore(t)
			request := minecraftSwitchRequest{Target: target, Action: "clone", Fingerprint: plan.Fingerprint, Confirm: true}
			journal, _, err := s.operations.beginMinecraftMaintenance(switchActionFingerprint(request), operationTypeMinecraftSwitch, sourceID)
			if err != nil {
				t.Fatal(err)
			}
			s.runMinecraftSwitch(context.Background(), journal, plan, request)
			result, err := s.operations.get(journal.OperationID)
			if tc.success {
				if err != nil || result.State != operationSucceeded || verificationCalls != 1 || imageChecks != 2 || cleanupCalls != 0 ||
					f.states[cloneID] != "inactive" || len(diagnostics) != 0 {
					t.Fatalf("Java-only clone did not complete and stop cleanly: %+v %v", result, err)
				}
				return
			}
			if err != nil || !verified || result.State == operationSucceeded || result.State == operationRunning {
				t.Fatalf("failed verification reported success: %+v verified=%v err=%v", result, verified, err)
			}
			expectedState := operationResolved
			if tc.interrupt {
				expectedState = operationNeedsAttention
			}
			if cleanupCalls != 1 || verificationCalls != 1 || result.State != expectedState {
				t.Fatalf("failed clone was not cleaned up and recovered exactly once: cleanup=%d verification=%d result=%+v", cleanupCalls, verificationCalls, result)
			}
			evidence, err := readSwitchEvidence(journal)
			namePath := filepath.Join(f.backend.identities().directory, "names", cloneID+".json")
			if tc.interrupt {
				if err != nil || evidence.CloneID != cloneID || evidence.Stage != "verifying_clone" {
					t.Fatalf("unsafe cleanup lost its Switch evidence: %+v %v", evidence, err)
				}
				if _, err := os.Lstat(namePath); err != nil {
					t.Fatal("unsafe cleanup removed the name before absence was proven", err)
				}
				blockCleanup = false
				for i := 0; i < 2; i++ {
					if err := s.recoverMinecraftSwitch(context.Background(), &evidence); err != nil {
						t.Fatal("verified clone cleanup could not be retried safely", err)
					}
				}
			}
			if err != nil || evidence.CloneID != cloneID || evidence.Stage != "recovered" {
				t.Fatalf("clone recovery evidence was not completed: %+v %v", evidence, err)
			}
			if !reflect.DeepEqual(diagnostics, []string{"systemctl", "journalctl"}) {
				t.Fatal("failed clone did not attempt both bounded diagnostic commands", diagnostics)
			}
			log := switchTestDiagnostic(t, s.operations, journal.OperationID)
			recoveryResult := `result="succeeded"`
			if tc.interrupt {
				recoveryResult = `result="failed"`
			}
			for _, want := range []string{"target software failed exact verification", `stage="verifying_clone"`, `target="` + target + `"`,
				`action="clone"`, `version="1.21.4"`, `clone_id="` + cloneID + `"`, recoveryResult, "<HOSTNAME-REDACTED>"} {
				if !strings.Contains(log, want) {
					t.Errorf("migration diagnostic missing %q", want)
				}
			}
			for _, secret := range []string{"internal-node.example", "verification-secret", "journal-secret", "oversized output", rcon} {
				if secret != "" && strings.Contains(log, secret) {
					t.Fatal("migration diagnostic retained private or oversized evidence")
				}
			}
			if target == "vanilla" {
				if strings.Count(log, "Startup evidence command unavailable") != 2 {
					t.Fatal("command errors were not recorded as unavailable")
				}
			} else if !strings.Contains(log, "Result=exit-code") || !strings.Contains(log, "clone startup diagnostic") {
				t.Fatal("service and journal evidence were lost")
			}
			if _, err := os.Lstat(filepath.Dir(evidence.CloneRecord.Storage.DataPath)); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("failed clone parent remains", err)
			}
			dir, env, quadlet := f.backend.paths(cloneID)
			for _, path := range []string{namePath, dir, env, quadlet, f.backend.deletePath(cloneID), filepath.Join(dir, "restore-journal.json")} {
				if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("failed clone artifact remains", path, err)
				}
			}
			if _, err := f.backend.identities().readRecord(cloneID + ".json"); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("failed clone remains registered", err)
			}
			if state, err := f.backend.observeRaw(context.Background(), cloneID); err != nil || state.State != "configured" {
				t.Fatal("failed clone unit or container remains", state, err)
			}
			response := switchTestOperation(t, s, sourceID, "?operation_id="+journal.OperationID)
			if response["clone_id"] != cloneID || response["target"] != target || response["action"] != "clone" || response["version"] != "1.21.4" {
				t.Fatal("operation response lost failed clone identity", response)
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

			if !reflect.DeepEqual(originalWorld, switchWorldSnapshot(t, sourceRecord.Storage.DataPath)) {
				t.Fatal("failed converted clone changed original world paths or bytes")
			}
		})
	}
}

func TestSwitchCloneRestoreRefusalIsBounded(t *testing.T) {
	generic := switchCloneRestoreFailure(nil).Error()
	for _, tc := range []struct {
		output string
		reason string
	}{
		{`{"error":"invalid_loader_evidence"}`, "The backup loader version could not be verified against the source server."},
		{`{"error":"invalid_archive_integrity"}`, "The backup archive or world metadata failed integrity validation."},
		{`{"error":"stale_plan"}`, "The clone restore plan changed before staging."},
		{`{"error":"recovery_required"}`, "An interrupted restore requires recovery before staging."},
		{`{"error":"/private/archive-controlled"}`, ""},
		{`{"error":"invalid_metadata_binding","detail":"private-runtime-secret"}`, ""},
		{`{"error":"stale_plan"}` + strings.Repeat(" ", 256), ""},
		{`{"error":"stale_plan"}{"error":"invalid_loader_evidence"}`, ""},
	} {
		want := generic
		if tc.reason != "" {
			want += " " + tc.reason
		}
		if got := switchCloneRestoreFailure([]byte(tc.output)).Error(); got != want || !strings.Contains(got, "The original server is preserved.") {
			t.Errorf("unexpected clone restore refusal %q, want %q", got, want)
		}
	}
}

func TestSwitchPlannerResolvesApprovedDirections(t *testing.T) {
	cases := []struct {
		from, to                                 string
		plugins, layout, runtime, convert, clone bool
	}{
		{"vanilla", "paper", false, false, true, true, true},
		{"vanilla", "purpur", false, false, true, true, true},
		{"vanilla", "fabric", false, false, true, true, true},
		{"paper", "vanilla", false, true, true, true, true},
		{"paper", "fabric", false, true, true, true, true},
		{"paper", "fabric", true, true, true, false, true},
		{"paper", "fabric", false, false, true, false, false},
		{"fabric", "paper", false, false, true, false, true},
		{"fabric", "quilt", false, false, true, true, true},
		{"forge", "neoforge", false, false, true, true, true},
		{"neoforge", "forge", false, false, true, true, true},
		{"vanilla", "fabric", false, false, false, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.from+"/"+tc.to, func(t *testing.T) {
			f := switchPlannerFixture(t, tc.from)
			record, err := f.backend.readRecord("jv-a1b2c3")
			if err != nil {
				t.Fatal(err)
			}
			if tc.layout {
				switchWorldFixture(t, record.Storage.DataPath, "1.21.4", "Family")
			}
			if tc.plugins {
				if err = os.MkdirAll(filepath.Join(record.Storage.DataPath, "plugins"), 0755); err != nil {
					t.Fatal(err)
				}
				pluginTestJar(t, filepath.Join(record.Storage.DataPath, "plugins", "Family.jar"), map[string]string{"plugin.yml": "name: Family\nversion: 1\nmain: example.Family\n"})
			}
			resolver := runAdminVersionStatus
			runAdminVersionStatus = func(ctx context.Context, policy, version, target, image string) ([]byte, error) {
				if target != "" {
					exact, _ := ctx.Value(switchRuntimeKey{}).(bool)
					if !exact {
						t.Fatal("new direction did not use exact runtime resolver")
					}
					if !tc.runtime {
						return nil, errors.New("no exact target release")
					}
				}
				return resolver(ctx, policy, version, target, image)
			}
			plan, err := planMinecraftSwitch(context.Background(), record.Runtime.ID, tc.to)
			if err != nil || plan.Convert != tc.convert || plan.Clone != tc.clone {
				t.Fatalf("wrong approved plan: %+v %v", plan, err)
			}
			if tc.clone && plan.Fingerprint == "" {
				t.Fatal("Available target has no reviewed fingerprint")
			}
			if tc.plugins && (len(plan.Content) != 1 || plan.Content[0].Filename != "Family.jar" || plan.Content[0].Compatibility == "verified") {
				t.Fatal("plugin blocker not named", plan.Content)
			}
		})
	}
}

func TestSwitchPublicationKeepsOrdinaryVersionSoftwareGuard(t *testing.T) {
	for _, pair := range [][2]string{{"paper", "vanilla"}, {"vanilla", "fabric"}, {"fabric", "quilt"}} {
		t.Run(pair[0]+"/"+pair[1], func(t *testing.T) {
			f := switchPlannerFixture(t, pair[0])
			ctx := context.Background()
			source, err := f.backend.readRecord("jv-a1b2c3")
			if err != nil {
				t.Fatal(err)
			}
			next := source
			next.Runtime.Software, next.Runtime.LoaderVersion, next.Runtime.ImageTag = pair[1], "", "stable"
			if caps, _ := javaServerSoftware(pair[1]); caps.ModLoader {
				next.Runtime.LoaderVersion, next.Runtime.ImageTag = "0.16.11", "java21"
			}
			records, err := f.backend.records()
			if err != nil {
				t.Fatal(err)
			}
			for _, maintenance := range []bool{false, true} {
				err = f.backend.applyVersion(ctx, source, instanceReplaceRecord(records, next), f.snapshot, maintenance)
				if err == nil || !strings.Contains(err.Error(), "Changing software on an existing instance is outside Phase 3A.") {
					t.Fatalf("ordinary Version accepted software change (maintenance=%v): %v", maintenance, err)
				}
			}
			// Keep existing data nonempty and change loader pins too: only the
			// reviewed Switch publication may cross either generic restriction.
			if pair[0] == "paper" {
				switchWorldFixture(t, source.Storage.DataPath, "1.21.4", "Family")
			} else if err = os.WriteFile(filepath.Join(source.Storage.DataPath, "level.dat"), []byte("original world"), 0644); err != nil {
				t.Fatal(err)
			}
			if pair[0] == "fabric" {
				loaderOnly := source
				loaderOnly.Runtime.LoaderVersion = "0.16.11"
				err = f.backend.applyVersion(ctx, source, instanceReplaceRecord(records, loaderOnly), f.snapshot, true)
				if err == nil || !strings.Contains(err.Error(), "Loader changes on existing data require separate compatibility support.") {
					t.Fatal("ordinary Version bypassed the existing-data loader protection", err)
				}
			}
			resolver := runAdminVersionStatus
			runAdminVersionStatus = func(ctx context.Context, policy, version, target, image string) ([]byte, error) {
				raw, err := resolver(ctx, policy, version, target, image)
				if target != "" && err == nil {
					raw = []byte(strings.ReplaceAll(string(raw), "0.16.10", "0.16.11"))
				}
				return raw, err
			}
			plan, err := planMinecraftSwitch(ctx, source.Runtime.ID, pair[1])
			if err != nil || !plan.Convert {
				t.Fatal("fixture did not reach reviewed Convert", plan, err)
			}
			if err = f.backend.prepareSwitchSourceCopy(ctx, source.Runtime.ID, plan.Version); err != nil {
				t.Fatal(err)
			}
			previous, err := f.backend.readRecord(source.Runtime.ID)
			if err != nil {
				t.Fatal(err)
			}
			next = previous
			next.Runtime.Software, next.Runtime.VersionPolicy, next.Runtime.Version = plan.Target, "pinned", plan.Version
			next.Runtime.LoaderVersion, next.Runtime.ImageTag, next.Runtime.ImageDigest = plan.Runtime.LoaderVersion, plan.Runtime.ImageTag, plan.RuntimeDigest
			next.Runtime.BedrockEnabled = plan.Runtime.CrossplayEnabled
			records, err = f.backend.records()
			if err != nil {
				t.Fatal(err)
			}
			e := minecraftSwitchEvidence{Schema: 1, OperationID: "12345678-1234-4123-8123-123456789abc", Source: source.Runtime.ID, SourceRecord: &source, Target: plan.Target, Version: plan.Version, Action: "convert", WorldChanged: true, ColdArchive: "Family-backup-" + pair[0] + "-2026-11-16-1200.tar.gz", Stage: "converting_world"}
			publish := func(evidence minecraftSwitchEvidence, reviewed minecraftSwitchPlan, target instanceRuntimeRecord) error {
				return f.backend.applySwitchRuntime(ctx, previous, instanceReplaceRecord(records, target), f.snapshot, evidence, &reviewed)
			}
			if err = publish(e, plan, next); err == nil {
				t.Fatal("publication accepted missing durable Switch evidence")
			}
			if err = saveSwitchEvidence(e); err != nil {
				t.Fatal(err)
			}
			for _, mutation := range []string{"version", "loader", "software", "convert", "backup", "content"} {
				target, reviewed, evidence := next, plan, e
				switch mutation {
				case "version":
					target.Runtime.Version = "1.21.5"
				case "loader":
					target.Runtime.LoaderVersion = "0.16.12"
				case "software":
					target.Runtime.Software = "purpur"
				case "convert":
					reviewed.Convert = false
				case "backup":
					evidence.ColdArchive = ""
				case "content":
					reviewed.Content = []switchContent{{Compatibility: "unknown"}}
				}
				if err = publish(evidence, reviewed, target); err == nil {
					t.Fatal("Switch publication accepted unreviewed mutation", mutation)
				}
			}
			// The private capability remains maintenance-only,
			// and binds the exact pair instead of opening subsequent Version calls.
			authorized := context.WithValue(ctx, switchRuntimeApplyKey{}, switchRuntimeApplyAuthorization{previous: previous, next: next})
			if switchRuntimeApplyAuthorized(authorized, previous, source) {
				t.Fatal("private authorization accepted a different publication")
			}
			if err = f.backend.applyVersion(authorized, previous, instanceReplaceRecord(records, next), f.snapshot, false); err == nil {
				t.Fatal("private authorization bypassed maintenance requirement")
			}
			if err = publish(e, plan, next); err != nil {
				t.Fatal("reviewed Switch publication failed", err)
			}
			published, err := f.backend.readRecord(source.Runtime.ID)
			if err != nil || published.Runtime.Software != plan.Target || published.Runtime.LoaderVersion != plan.Runtime.LoaderVersion || published.Runtime.Version != source.Runtime.Version {
				t.Fatal("Switch did not publish exact reviewed runtime", published, err)
			}
			rollback := source
			rollback.Runtime.Version, rollback.Runtime.PreserveContent = e.Version, true
			records, err = f.backend.records()
			if err != nil {
				t.Fatal(err)
			}
			restore := func(evidence minecraftSwitchEvidence, original instanceRuntimeRecord) error {
				return f.backend.applySwitchRuntime(ctx, published, instanceReplaceRecord(records, original), f.snapshot, evidence, nil)
			}
			missing := e
			missing.WorldChanged = false
			if err = restore(missing, rollback); err == nil {
				t.Fatal("recovery accepted missing cold rollback authority")
			}
			wrongSource := rollback
			wrongSource.Runtime.LoaderVersion = "0.16.12"
			if err = restore(e, wrongSource); err == nil {
				t.Fatal("recovery accepted an unrecorded source runtime")
			}
			if err = restore(e, rollback); err != nil {
				t.Fatal("durable Switch evidence could not restore original software/loader", err)
			}
			recovered, err := f.backend.readRecord(source.Runtime.ID)
			if err != nil || !reflect.DeepEqual(recovered.Runtime, rollback.Runtime) {
				t.Fatal("Switch recovery did not restore exact original runtime", recovered, err)
			}
		})
	}
}
