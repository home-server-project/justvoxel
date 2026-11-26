package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
)

func switchTestDiagnostic(t *testing.T, store *operationStore, id string) string {
	t.Helper()
	data, err := store.readDiagnosticLog("migration", id)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func switchTestOperation(t *testing.T, s *server, sourceID, query string) map[string]any {
	t.Helper()
	w := httptest.NewRecorder()
	s.minecraftSwitchOperation(w, surfaceRequest(http.MethodGet, "/v1/minecraft/instances/"+sourceID+"/switch/operation"+query, ""))
	var result map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &result); w.Code != http.StatusOK || err != nil {
		t.Fatalf("Switch operation response unavailable: %d %v", w.Code, err)
	}
	return result
}

func TestSwitchOperationReconnectAndExplicitLookup(t *testing.T) {
	f := newInstanceActivationFixture(t)
	original := instanceRuntimes
	instanceRuntimes = f.backend
	t.Cleanup(func() { instanceRuntimes = original })
	source, other := f.configs[0].ID, f.configs[1].ID
	for _, tc := range []struct {
		states    []operationState
		reconnect bool
	}{
		{[]operationState{operationNeedsAttention, operationResolved}, false},
		{[]operationState{operationValidating, operationRunning, operationVerifying, operationSucceeded}, false},
		{[]operationState{operationValidating}, true},
		{[]operationState{operationValidating, operationRunning}, true},
		{[]operationState{operationNeedsAttention}, true},
		{[]operationState{operationFailed}, true},
	} {
		state := tc.states[len(tc.states)-1]
		t.Run(string(state), func(t *testing.T) {
			s := surfaceTestServer(t, roleAdministrator)
			s.operations = openTestOperationStore(t)
			journal, _, err := s.operations.beginMinecraftMaintenance(testMinecraftResetFingerprint, operationTypeMinecraftSwitch, source)
			if err != nil {
				t.Fatal(err)
			}
			for _, next := range tc.states {
				if _, err := s.operations.transition(journal.OperationID, next, string(next), "Switch "+string(next)+"."); err != nil {
					t.Fatal(err)
				}
			}
			assertResult := func(response map[string]any, wantOperation bool) {
				t.Helper()
				operation := response["operation"]
				if !wantOperation {
					if operation != nil {
						t.Fatalf("unexpected Switch operation: %v", operation)
					}
					return
				}
				result, ok := operation.(map[string]any)
				if !ok || result["operation_id"] != journal.OperationID || result["state"] != string(state) {
					t.Fatalf("wrong Switch operation: %v", operation)
				}
			}
			query := "?operation_id=" + journal.OperationID
			assertResult(switchTestOperation(t, s, source, ""), tc.reconnect)
			assertResult(switchTestOperation(t, s, source, query), true)
			assertResult(switchTestOperation(t, s, other, ""), false)
			assertResult(switchTestOperation(t, s, other, query), false)
		})
	}
}

func TestSwitchDiagnosticCategoryAndRecoveryFailure(t *testing.T) {
	store := openTestOperationStore(t)
	if !validDiagnosticOperationType(operationTypeMinecraftSwitch) || diagnosticCategoryForOperationType(operationTypeMinecraftSwitch) != "migration" {
		t.Fatal("Switch must be a Migration diagnostic operation")
	}
	journal, _, err := store.beginMinecraftMaintenance(testMinecraftResetFingerprint, operationTypeMinecraftSwitch, "jv-a1b2c3")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.transition(journal.OperationID, operationValidating, "validating", "Checking Switch."); err != nil {
		t.Fatal(err)
	}
	e := minecraftSwitchEvidence{OperationID: journal.OperationID}
	store.recordSwitchRecovery(e, errors.New("recovery could not verify clone on private-node.example\napi key=private-value"))
	log := switchTestDiagnostic(t, store, journal.OperationID)
	for _, want := range []string{`state="queued"`, `state="validating"`, "switch_recovery", `result="failed"`, "recovery could not verify clone", "<HOSTNAME-REDACTED>"} {
		if !strings.Contains(log, want) {
			t.Errorf("Migration diagnostic missing %q", want)
		}
	}
	for _, secret := range []string{"private-node.example", "private-value"} {
		if strings.Contains(log, secret) {
			t.Fatal("Migration diagnostic leaked private recovery details")
		}
	}
	path := filepath.Join(store.logsDir, "migration-logs", journal.OperationID+".log")
	for file, mode := range map[string]os.FileMode{path: 0o600, filepath.Dir(path): 0o700} {
		info, err := os.Stat(file)
		if err != nil || info.Mode().Perm() != mode {
			t.Fatalf("unexpected private log permissions for %s: %v %v", file, info, err)
		}
	}
	entries, err := store.listDiagnosticLogs()
	if err != nil || len(entries) != 1 || entries[0].Category != "migration" || entries[0].OperationType != operationTypeMinecraftSwitch {
		t.Fatalf("Switch missing from Migration log index: %+v %v", entries, err)
	}
}

func TestSwitchValidationFailureKeepsDurableMetadataWhenDiagnosticsUnavailable(t *testing.T) {
	f := switchPlannerFixture(t)
	source := "jv-a1b2c3"
	plan, err := planMinecraftSwitch(context.Background(), source, "purpur")
	if err != nil {
		t.Fatal(err)
	}
	s := surfaceTestServer(t, roleAdministrator)
	s.operations = openTestOperationStore(t)
	oldLock := pluginRuntimeLockPath
	pluginRuntimeLockPath = filepath.Join(t.TempDir(), "maintenance.lock")
	t.Cleanup(func() { pluginRuntimeLockPath = oldLock })
	request := minecraftSwitchRequest{Target: plan.Target, Action: "clone", Fingerprint: plan.Fingerprint, Confirm: true}
	journal, _, err := s.operations.beginMinecraftMaintenance(switchActionFingerprint(request), operationTypeMinecraftSwitch, source)
	if err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(s.operations.logsDir, "migration-logs", journal.OperationID+".log")
	if err = os.Remove(logPath); err != nil {
		t.Fatal(err)
	}
	if err = os.Symlink(filepath.Join(t.TempDir(), "unavailable"), logPath); err != nil {
		t.Fatal(err)
	}
	originalCommand := f.backend.command
	failed := false
	f.backend.command = func(ctx context.Context, command string, args ...string) ([]byte, error) {
		if !failed && command == "systemctl" && len(args) != 0 && args[0] == "show" && args[len(args)-1] == "minecraft-"+source+".service" {
			failed = true
			return nil, errors.New("transient validation failure")
		}
		return originalCommand(ctx, command, args...)
	}
	s.runMinecraftSwitch(context.Background(), journal, plan, request)
	result, err := s.operations.get(journal.OperationID)
	if !failed || err != nil || result.State != operationResolved {
		t.Fatalf("diagnostic writes interfered with early recovery: %v %+v %v", failed, result, err)
	}
	path := "?operation_id=" + journal.OperationID + "&target=fabric&action=convert&version=LATEST&clone_id=jv-fakeid"
	response := switchTestOperation(t, s, source, path)
	if response["target"] != plan.Target || response["action"] != "clone" || response["version"] != plan.Version || response["clone_id"] != nil {
		t.Fatal("validating Switch lost its saved metadata", response)
	}
	e, err := readSwitchEvidence(journal)
	if err != nil {
		t.Fatal(err)
	}
	e.Source = "jv-wrong1"
	if err := saveSwitchEvidence(e); err != nil {
		t.Fatal(err)
	}
	response = switchTestOperation(t, s, source, path)
	for _, key := range []string{"target", "action", "version", "clone_id"} {
		if _, exists := response[key]; exists {
			t.Fatal("invalid evidence fabricated operation metadata", key)
		}
	}
}

func TestSwitchCloneParentCleanupRequiresExactEmptyPrivateDirectory(t *testing.T) {
	for _, name := range []string{"empty", "missing", "foreign owner", "wrong mode", "symlink", "symlink ancestor", "data", "provision marker", "unexpected entry", "source ID", "source path", "no clone record", "invalid clone ID"} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			base := filepath.Join(root, "minecraft-instances")
			sourceID, cloneID := "jv-source", "jv-clone1"
			sourcePath, clonePath := filepath.Join(base, sourceID), filepath.Join(base, cloneID)
			source := instanceRuntimeRecord{Runtime: minecraftInstanceRuntimeConfig{ID: sourceID, Software: "paper", UID: 1000, GID: 1000}, Storage: instanceRuntimeStorage{Root: root, DataPath: filepath.Join(sourcePath, "data")}}
			clone := instanceRuntimeRecord{Runtime: minecraftInstanceRuntimeConfig{ID: cloneID, Software: "purpur", UID: 1000, GID: 1000}, Storage: instanceRuntimeStorage{Root: root, DataPath: filepath.Join(clonePath, "data")}}
			e := minecraftSwitchEvidence{Source: sourceID, Target: "purpur", Version: "1.21.4", Action: "clone", SourceRecord: &source, CloneID: cloneID, CloneRecord: &clone}
			if os.Geteuid() != 0 {
				switch name {
				case "empty", "missing", "foreign owner", "wrong mode", "symlink", "symlink ancestor", "data", "provision marker", "unexpected entry":
					t.Skip("this case requires a root-owned fixture")
				}
			}
			if err := os.MkdirAll(sourcePath, 0o700); err != nil {
				t.Fatal(err)
			}
			if name != "missing" && name != "symlink" {
				if err := os.Mkdir(clonePath, 0o700); err != nil {
					t.Fatal(err)
				}
			}
			var err error
			switch name {
			case "foreign owner":
				if os.Geteuid() == 0 {
					err = os.Chown(clonePath, 1000, 0)
				}
			case "wrong mode":
				err = os.Chmod(clonePath, 0o750)
			case "symlink":
				err = os.Symlink(sourcePath, clonePath)
			case "symlink ancestor":
				realBase := base + "-real"
				if err = os.Rename(base, realBase); err == nil {
					err = os.Symlink(realBase, base)
				}
			case "data":
				err = os.Mkdir(filepath.Join(clonePath, "data"), 0o700)
			case "provision marker":
				err = os.WriteFile(filepath.Join(clonePath, ".provision.json"), []byte("{}"), 0o600)
			case "unexpected entry":
				err = os.WriteFile(filepath.Join(clonePath, "unexpected"), []byte("keep"), 0o600)
			case "source ID":
				e.CloneID, e.CloneRecord = sourceID, &source
			case "source path":
				source.Storage.DataPath = clone.Storage.DataPath
			case "no clone record":
				e.CloneRecord = nil
			case "invalid clone ID":
				e.CloneID = "../" + sourceID
			}
			if err != nil {
				t.Fatal(err)
			}
			wantRemoved := name == "empty" || name == "missing"
			err = errInstanceRecovery
			if validSwitchClonePlacement(e) {
				storage := newSwitchRetirementTestStorage(t)
				b := &instanceRuntimeBackend{command: storage.command}
				err = b.switchCloneRetirementStorage(context.Background(), "empty", *e.CloneRecord, "-")
			}
			if (err == nil) != wantRemoved {
				t.Fatalf("cleanup success = %v, want %v: %v", err == nil, wantRemoved, err)
			}
			if _, err := os.Lstat(clonePath); (wantRemoved && !errors.Is(err, os.ErrNotExist)) || (!wantRemoved && err != nil) {
				t.Fatal("clone path was retained or removed unexpectedly", err)
			}
			if _, err := os.Lstat(sourcePath); err != nil {
				t.Fatal("cleanup touched the source", err)
			}
		})
	}
}

type switchRetirementTestStorage struct {
	t         *testing.T
	helpers   map[string]string
	mountInfo string
}

func newSwitchRetirementTestStorage(t *testing.T) *switchRetirementTestStorage {
	t.Helper()
	dir := t.TempDir()
	copyHelper := func(name, oldPath, fixturePath string) string {
		t.Helper()
		contents, err := os.ReadFile(filepath.Join("../../../mjust/libexec", name))
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(strings.ReplaceAll(string(contents), oldPath, fixturePath)), 0700); err != nil {
			t.Fatal(err)
		}
		return path
	}
	common := filepath.Join(dir, "common.sh")
	// The local-storage paths under test only call require_root from common.sh.
	if err := os.WriteFile(common, []byte("require_root() { [[ $EUID == 0 ]]; }\n"), 0600); err != nil {
		t.Fatal(err)
	}
	prerequisites := copyHelper("instance-runtime-prerequisites", "/usr/libexec/justvoxel/mjust/common.sh", common)
	retire := copyHelper("instance-switch-retire", instanceRuntimeHelper, prerequisites)
	remove := copyHelper("instance-delete", instanceRuntimeHelper, prerequisites)
	return &switchRetirementTestStorage{t: t, helpers: map[string]string{
		instanceRuntimeHelper: prerequisites, instanceSwitchRetireHelper: retire, instanceDeleteHelper: remove,
	}}
}

func (s *switchRetirementTestStorage) command(ctx context.Context, name string, args ...string) ([]byte, error) {
	s.t.Helper()
	if name != "nsenter" || len(args) < 4 || args[0] != "--mount=/proc/1/ns/mnt" || args[1] != "--" || s.helpers[args[2]] == "" {
		s.t.Fatalf("unexpected storage invocation: %s %v", name, args)
	}
	path := s.helpers[args[2]]
	if args[2] == instanceSwitchRetireHelper && s.mountInfo != "" {
		// Inject mount discovery only. The target and all checked entries are
		// real, isolated directories. This also models same-device bind mounts.
		program, err := os.ReadFile(path)
		if err != nil {
			s.t.Fatal(err)
		}
		path = filepath.Join(filepath.Dir(path), "retire-with-mount-fixture")
		program = []byte(strings.ReplaceAll(string(program), "/proc/self/mountinfo", s.mountInfo))
		if err := os.WriteFile(path, program, 0700); err != nil {
			s.t.Fatal(err)
		}
	}
	return exec.CommandContext(ctx, "bash", append([]string{path}, args[3:]...)...).Output()
}

type switchRetirementFixture struct {
	f       *instanceActivationFixture
	server  *server
	storage *switchRetirementTestStorage
	e       minecraftSwitchEvidence
	j       operationJournal
	parent  string
	tx      string
	name    string
	before  map[string]string
	world   map[string]string
}

func newSwitchRetirementFixture(t *testing.T, historical bool) *switchRetirementFixture {
	t.Helper()
	if os.Geteuid() != 0 {
		t.Skip("requires root for isolated Restore ownership fixtures")
	}
	f := switchPlannerFixture(t, "vanilla")
	b := f.backend
	oldLock, oldUnits := pluginRuntimeLockPath, instanceBackupUnits
	pluginRuntimeLockPath = filepath.Join(t.TempDir(), "maintenance.lock")
	instanceBackupUnits = t.TempDir()
	t.Cleanup(func() {
		pluginRuntimeLockPath, instanceBackupUnits = oldLock, oldUnits
	})
	sourceID, cloneID := f.configs[0].ID, f.configs[1].ID
	source, err := b.readRecord(sourceID)
	if err != nil {
		t.Fatal(err)
	}
	switchWorldFixture(t, source.Storage.DataPath, source.Runtime.Version, "Survival")
	if err := os.Chown(source.Storage.DataPath, int(source.Runtime.UID), int(source.Runtime.GID)); err != nil {
		t.Fatal(err)
	}
	registry := b.identities().directory
	sourceName := filepath.Join(registry, "names", sourceID+".json")
	if err := b.writeJSON(sourceName, minecraftInstanceNameRecord{ID: sourceID, Software: "vanilla", DisplayName: "Survival"}, true); err != nil {
		t.Fatal(err)
	}
	clone, err := b.readRecord(cloneID)
	if err != nil {
		t.Fatal(err)
	}
	port, secret := clone.Runtime.JavaPort, clone.Runtime.RCONPassword
	clone.Runtime = source.Runtime
	clone.Runtime.ID, clone.Runtime.JavaPort, clone.Runtime.RCONPassword = cloneID, port, secret
	clone.Runtime.Software, clone.Runtime.StartOnBoot, clone.Runtime.PreserveContent = "purpur", false, true
	cloneDir, _, _ := b.paths(cloneID)
	name := filepath.Join(registry, "names", cloneID+".json")
	if err := b.writeJSON(name, minecraftInstanceNameRecord{ID: cloneID, Software: "purpur", DisplayName: "Survival Purpur"}, true); err != nil {
		t.Fatal(err)
	}
	parent := filepath.Dir(clone.Storage.DataPath)
	if historical {
		for _, path := range []string{filepath.Join(cloneDir, "config.json"), cloneDir, filepath.Join(registry, cloneID+".json")} {
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.Mkdir(parent, 0700); err != nil {
			t.Fatal(err)
		}
	} else {
		if err := b.writeJSON(filepath.Join(cloneDir, "config.json"), clone, true); err != nil {
			t.Fatal(err)
		}
		if err := f.apply(cloneID); err != nil {
			t.Fatal(err)
		}
		clone, err = b.readRecord(cloneID)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Chown(clone.Storage.DataPath, int(clone.Runtime.UID), int(clone.Runtime.GID)); err != nil {
			t.Fatal(err)
		}
	}
	tx := filepath.Join(parent, ".restore-"+strings.Repeat("a", 32))
	if err := os.MkdirAll(filepath.Join(tx, "replacement"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{tx, filepath.Join(tx, "replacement")} {
		// Represents the effective justvoxel-web group of the production Agent.
		if err := os.Chown(path, 0, 1001); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(tx, "archive.tar.gz"), []byte("retained staging archive"), 0600); err != nil {
		t.Fatal(err)
	}
	original := filepath.Join(tx, "original")
	if err := os.Mkdir(original, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.Chown(original, int(clone.Runtime.UID), int(clone.Runtime.GID)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(original, "clone-only"), []byte("old clone data"), 0600); err != nil {
		t.Fatal(err)
	}
	s := surfaceTestServer(t, roleAdministrator)
	s.operations = openTestOperationStore(t)
	j, _, err := s.operations.beginMinecraftMaintenance(testMinecraftResetFingerprint, operationTypeMinecraftSwitch, sourceID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.operations.transition(j.OperationID, operationNeedsAttention, "switch_failed", "Switch did not complete."); err != nil {
		t.Fatal(err)
	}
	e := minecraftSwitchEvidence{
		Schema: 1, OperationID: j.OperationID, Source: sourceID, Target: "purpur", Version: source.Runtime.Version,
		Action: "clone", Stage: "verifying_clone", SourceRecord: &source, CloneID: cloneID, CloneRecord: &clone,
	}
	if err := saveSwitchEvidence(e); err != nil {
		t.Fatal(err)
	}
	storage := newSwitchRetirementTestStorage(t)
	base := b.command
	b.command = func(ctx context.Context, command string, args ...string) ([]byte, error) {
		if command == "nsenter" && len(args) > 3 && storage.helpers[args[2]] != "" {
			// Production path and inode checks run against temporary directories.
			// The test already runs in the namespace containing those directories.
			return storage.command(ctx, command, args...)
		}
		if command == "systemctl" && args[0] == "show" {
			if args[1] == "--property=Job" {
				return nil, nil
			}
			if strings.Contains(args[len(args)-1], "-backup-") || args[len(args)-1] == instanceBackupUnit(cloneID, "service") {
				return []byte("LoadState=not-found\nActiveState=inactive\nJob=0\n"), nil
			}
		}
		if command == "podman" && args[0] == "rm" {
			if args[len(args)-1] != "minecraft-"+cloneID {
				t.Fatal("attempted to remove a container other than the clone")
			}
			delete(f.containers, cloneID)
			return nil, nil
		}
		return base(ctx, command, args...)
	}
	_, sourceEnv, sourceQuadlet := b.paths(sourceID)
	before := map[string]string{}
	for _, path := range []string{filepath.Join(registry, sourceID+".json"), sourceName, filepath.Join(b.root, sourceID, "config.json"), sourceEnv, sourceQuadlet} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		before[path] = string(raw)
	}
	return &switchRetirementFixture{
		f: f, server: s, storage: storage, e: e, j: j, parent: parent, tx: tx, name: name,
		before: before, world: switchWorldSnapshot(t, source.Storage.DataPath),
	}
}

func (fx *switchRetirementFixture) recover() *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	path := "/v1/minecraft/instances/" + fx.e.Source + "/switch/recover"
	body := fmt.Sprintf(`{"operation_id":%q,"confirm":true}`, fx.j.OperationID)
	fx.server.minecraftSwitchRecover(response, surfaceRequest(http.MethodPost, path, body))
	return response
}

func (fx *switchRetirementFixture) assertBlocked(t *testing.T, diagnostic string) {
	t.Helper()
	if response := fx.recover(); response.Code != http.StatusConflict {
		t.Fatalf("unexpected recovery response: %d %s", response.Code, response.Body)
	}
	current, err := fx.server.operations.currentMinecraftMaintenance()
	if err != nil || current == nil || current.OperationID != fx.j.OperationID || current.State != operationNeedsAttention {
		t.Fatalf("recovery released maintenance on refusal: %+v %v", current, err)
	}
	if log := switchTestDiagnostic(t, fx.server.operations, fx.j.OperationID); !strings.Contains(log, diagnostic) || strings.Contains(log, fx.parent) {
		t.Fatalf("missing or unsafe diagnostic: %s", log)
	}
}

func (fx *switchRetirementFixture) assertRecovered(t *testing.T) {
	t.Helper()
	if response := fx.recover(); response.Code != http.StatusOK {
		t.Fatalf("recovery failed: %d %s\n%s", response.Code, response.Body, switchTestDiagnostic(t, fx.server.operations, fx.j.OperationID))
	}
	for _, path := range []string{fx.parent, fx.tx, fx.name, filepath.Join(fx.f.backend.root, fx.e.CloneID), fx.f.backend.deletePath(fx.e.CloneID)} {
		if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("clone artifact remains: %s (%v)", path, err)
		}
	}
	for path, want := range fx.before {
		if raw, err := os.ReadFile(path); err != nil || string(raw) != want {
			t.Fatalf("source file changed: %s (%v)", path, err)
		}
	}
	data := fx.e.SourceRecord.Storage.DataPath
	if state, err := fx.f.backend.observe(context.Background(), fx.e.Source); err != nil || state.State != "stopped" ||
		!reflect.DeepEqual(fx.world, switchWorldSnapshot(t, data)) {
		t.Fatalf("source state or data changed: %+v %v", state, err)
	}
	j, err := fx.server.operations.get(fx.j.OperationID)
	if err != nil || j.State != operationResolved {
		t.Fatalf("Switch is not resolved: %+v %v", j, err)
	}
	e, err := readSwitchEvidence(j)
	if err != nil || e.Stage != "recovered" {
		t.Fatalf("recovered Switch evidence not persisted: %v, stage %q", err, e.Stage)
	}
	if current, err := fx.server.operations.currentMinecraftMaintenance(); err != nil || current != nil {
		t.Fatalf("maintenance gate remains held: %+v %v", current, err)
	}
	if log := switchTestDiagnostic(t, fx.server.operations, fx.j.OperationID); !strings.Contains(log, "recovery finalization completed") {
		t.Fatal("finalization stage is missing from Migration diagnostics")
	}
	plan, err := planMinecraftSwitch(context.Background(), fx.e.Source, "purpur")
	if err != nil || !plan.Clone {
		t.Fatalf("another Switch cannot be reviewed: %+v %v", plan, err)
	}
	if response := fx.recover(); response.Code != http.StatusConflict {
		t.Fatalf("resolved Switch accepted another recovery: %d %s", response.Code, response.Body)
	}
	if err := fx.server.recoverMinecraftSwitch(context.Background(), &e); err != nil {
		t.Fatal("re-running finalization after an interruption is unsafe", err)
	}
}

func TestSwitchHistoricalCloneRecovery(t *testing.T) {
	t.Run("retired clone", func(t *testing.T) {
		newSwitchRetirementFixture(t, true).assertRecovered(t)
	})
	t.Run("interrupted after storage retirement", func(t *testing.T) {
		fx := newSwitchRetirementFixture(t, true)
		// Partial removal of a recognized directory is a valid retry input.
		if err := os.Remove(filepath.Join(fx.tx, "archive.tar.gz")); err != nil {
			t.Fatal(err)
		}
		base := fx.f.backend.command
		interrupt := true
		fx.f.backend.command = func(ctx context.Context, command string, args ...string) ([]byte, error) {
			out, err := base(ctx, command, args...)
			if err == nil && interrupt && command == "nsenter" && args[2] == instanceSwitchRetireHelper {
				interrupt = false
				return nil, errors.New("interrupted after verified storage cleanup")
			}
			return out, err
		}
		fx.assertBlocked(t, "retained Restore transaction validation or retirement failed")
		if _, err := os.Lstat(fx.parent); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("production helper did not remove the clone parent", err)
		}
		if _, err := os.Lstat(fx.name); err != nil {
			t.Fatal("name must survive until finalization", err)
		}
		fx.assertRecovered(t)
	})
	t.Run("interrupted after name retirement", func(t *testing.T) {
		fx := newSwitchRetirementFixture(t, true)
		b := fx.f.backend
		if err := b.switchCloneRetirementStorage(context.Background(), "retired", *fx.e.CloneRecord, "-"); err != nil {
			t.Fatal(err)
		}
		if err := b.removePrivate(fx.name); err != nil {
			t.Fatal(err)
		}
		fx.assertRecovered(t)
	})
	t.Run("real bind mount in private namespace", func(t *testing.T) {
		namespace := []string{"--mount", "--propagation", "private", "--"}
		if err := exec.Command("unshare", append(namespace, "true")...).Run(); err != nil {
			t.Skip("requires a private mount namespace; mount discovery is also covered with a fixture")
		}
		fx := newSwitchRetirementFixture(t, true)
		base := fx.f.backend.command
		fx.f.backend.command = func(ctx context.Context, command string, args ...string) ([]byte, error) {
			if command != "nsenter" || args[2] != instanceSwitchRetireHelper {
				return base(ctx, command, args...)
			}
			mountAndRun := `mount --bind -- "$1" "$2" || exit 125; shift 2; exec bash "$@"`
			argv := append(namespace, "bash", "-c", mountAndRun, "mount-test",
				fx.e.SourceRecord.Storage.DataPath, filepath.Join(fx.tx, "original"), fx.storage.helpers[args[2]])
			out, err := exec.CommandContext(ctx, "unshare", append(argv, args[3:]...)...).Output()
			var exit *exec.ExitError
			if errors.As(err, &exit) && exit.ExitCode() == 125 {
				t.Skip("cannot bind mount fixture directories in the private namespace")
			}
			return out, err
		}
		fx.assertBlocked(t, "clone storage mount identity mismatch")
		if !reflect.DeepEqual(fx.world, switchWorldSnapshot(t, fx.e.SourceRecord.Storage.DataPath)) {
			t.Fatal("nested mount refusal touched the source")
		}
	})
	for _, refusal := range []struct {
		name, diagnostic string
		change           func(*testing.T, *switchRetirementFixture) error
	}{
		{"unknown entry", "unexpected clone storage entry", func(_ *testing.T, fx *switchRetirementFixture) error {
			return os.Mkdir(filepath.Join(fx.parent, "unknown"), 0700)
		}},
		{"invalid structure", "retained Restore transaction validation", func(_ *testing.T, fx *switchRetirementFixture) error {
			return os.Mkdir(filepath.Join(fx.tx, "unknown"), 0700)
		}},
		{"unsafe ownership", "retained Restore transaction validation", func(_ *testing.T, fx *switchRetirementFixture) error {
			return os.Chown(fx.tx, 1000, 0)
		}},
		{"unsafe permissions", "retained Restore transaction validation", func(_ *testing.T, fx *switchRetirementFixture) error {
			return os.Chmod(fx.tx, 0770)
		}},
		{"symlink escape", "retained Restore transaction validation", func(_ *testing.T, fx *switchRetirementFixture) error {
			return os.Symlink(fx.e.SourceRecord.Storage.DataPath, filepath.Join(fx.tx, "original", "source-link"))
		}},
		{"nested bind mount", "clone storage mount identity mismatch", func(t *testing.T, fx *switchRetirementFixture) error {
			fx.storage.mountInfo = filepath.Join(t.TempDir(), "mountinfo")
			mountInfo := "900 1 0:1 /other " + filepath.Join(fx.tx, "original") + " rw - tmpfs none rw\n"
			return os.WriteFile(fx.storage.mountInfo, []byte(mountInfo), 0600)
		}},
		{"invalid source binding", "source verification failed", func(_ *testing.T, fx *switchRetirementFixture) error {
			// A structurally valid, distinct path pair on another placement must
			// still fail verification against the source's committed record.
			root := filepath.Join(fx.e.SourceRecord.Storage.Root, "other")
			for _, record := range []*instanceRuntimeRecord{fx.e.SourceRecord, fx.e.CloneRecord} {
				record.Storage.Root = root
				record.Storage.DataPath = filepath.Join(root, "minecraft-instances", record.Runtime.ID, "data")
			}
			return saveSwitchEvidence(fx.e)
		}},
		{"invalid clone binding", "Switch recovery evidence validation failed", func(_ *testing.T, fx *switchRetirementFixture) error {
			fx.e.CloneID = fx.e.Source
			return saveSwitchEvidence(fx.e)
		}},
		{"live clone", "clone runtime retirement incomplete", func(_ *testing.T, fx *switchRetirementFixture) error {
			fx.f.states[fx.e.CloneID], fx.f.containers[fx.e.CloneID] = "active", "running"
			return nil
		}},
		{"registered clone", "clone runtime retirement incomplete", func(_ *testing.T, fx *switchRetirementFixture) error {
			b := fx.f.backend
			return b.writeJSON(filepath.Join(b.identities().directory, fx.e.CloneID+".json"), minecraftInstanceRecord{ID: fx.e.CloneID}, false)
		}},
		{"unverified source", "source verification failed", func(_ *testing.T, fx *switchRetirementFixture) error {
			_, _, q := fx.f.backend.paths(fx.e.Source)
			return os.WriteFile(q, []byte("uncommitted Quadlet"), 0600)
		}},
		{"source no longer stopped", "original source server state could not be verified", func(_ *testing.T, fx *switchRetirementFixture) error {
			fx.f.states[fx.e.Source], fx.f.containers[fx.e.Source] = "active", "running"
			return nil
		}},
	} {
		t.Run(refusal.name, func(t *testing.T) {
			fx := newSwitchRetirementFixture(t, true)
			world := switchWorldSnapshot(t, fx.e.SourceRecord.Storage.DataPath)
			sourceData := fx.e.SourceRecord.Storage.DataPath
			if err := refusal.change(t, fx); err != nil {
				t.Fatal(err)
			}
			fx.assertBlocked(t, refusal.diagnostic)
			for _, path := range []string{fx.tx, fx.name} {
				if _, err := os.Lstat(path); err != nil {
					t.Fatal("refused recovery erased evidence", err)
				}
			}
			if !reflect.DeepEqual(world, switchWorldSnapshot(t, sourceData)) {
				t.Fatal("refused recovery touched source data")
			}
		})
	}
}

func switchRetirementInode(t *testing.T, path string) []uint64 {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	st := info.Sys().(*syscall.Stat_t)
	return []uint64{uint64(st.Dev), st.Ino}
}

func TestSwitchCloneRestoreRetirement(t *testing.T) {
	t.Run("registered clone with missing Restore journal", func(t *testing.T) {
		fx := newSwitchRetirementFixture(t, false)
		fx.assertBlocked(t, "unexpected clone storage entry")
		for _, path := range []string{fx.tx, fx.e.CloneRecord.Storage.DataPath, fx.name} {
			if _, err := os.Lstat(path); err != nil {
				t.Fatal("unbound Restore storage was removed", err)
			}
		}
	})
	for _, stage := range []string{"completed", "rolled_back", "unresolved", "foreign inode", "foreign config digest", "interrupted before deletion"} {
		t.Run(stage, func(t *testing.T) {
			fx := newSwitchRetirementFixture(t, false)
			b := fx.f.backend
			path := filepath.Join(b.root, fx.e.CloneID, "restore-journal.json")
			config, err := b.readPrivate(filepath.Join(b.root, fx.e.CloneID, "config.json"))
			if err != nil {
				t.Fatal(err)
			}
			previous := filepath.Join(fx.tx, "original")
			data := fx.e.CloneRecord.Storage.DataPath
			journal := instanceRestoreJournal{
				Schema: 1, ID: fx.e.CloneID, Stage: "completed", ConfigSHA256: instanceHash(config),
				Transaction: strings.TrimPrefix(filepath.Base(fx.tx), ".restore-"),
				Parent:      switchRetirementInode(t, fx.parent), Original: switchRetirementInode(t, previous),
				Replacement: switchRetirementInode(t, data), Archive: "Survival-backup-vanilla-2026-03-06-0015.tar.gz",
				Fingerprint: instanceHash([]byte("fixture")),
			}
			switch stage {
			case "rolled_back":
				journal.Stage = stage
				journal.Original, journal.Replacement = journal.Replacement, journal.Original
				if err := os.Rename(previous, filepath.Join(fx.tx, "failed-replacement")); err != nil {
					t.Fatal(err)
				}
			case "unresolved":
				journal.Stage = "recovery_required"
			case "foreign inode":
				journal.Original[1]++
			case "foreign config digest":
				journal.ConfigSHA256 = instanceHash([]byte("another configuration"))
			}
			if err := b.writeJSON(path, journal, false); err != nil {
				t.Fatal(err)
			}
			if stage == "unresolved" {
				old := instanceBackupStorageCommand
				t.Cleanup(func() { instanceBackupStorageCommand = old })
				var calls []string
				instanceBackupStorageCommand = func(_ context.Context, command string, args ...string) ([]byte, error) {
					if command != "nsenter" || args[2] != instanceRestoreHelper || args[4] != fx.e.CloneID {
						t.Fatal("Restore recovery selected the wrong target", args)
					}
					calls = append(calls, args[3])
					return nil, errors.New("the existing Restore engine could not verify rollback")
				}
				fx.assertBlocked(t, "clone Restore recovery incomplete")
				if !reflect.DeepEqual(calls, []string{"rollback", "recovery-required"}) {
					t.Fatal("existing Restore recovery was bypassed", calls)
				}
			}
			if stage == "foreign inode" || stage == "foreign config digest" {
				fx.assertBlocked(t, "retained Restore transaction validation")
			}
			if stage == "unresolved" || stage == "foreign inode" || stage == "foreign config digest" {
				for _, kept := range []string{path, fx.tx, data, fx.name} {
					if _, err := os.Lstat(kept); err != nil {
						t.Fatal("failed verification discarded evidence", err)
					}
				}
				return
			}
			base := b.command
			interrupt := stage == "interrupted before deletion"
			checkOrdering := true
			b.command = func(ctx context.Context, command string, args ...string) ([]byte, error) {
				if command == "nsenter" && args[2] == instanceDeleteHelper && args[3] == "inspect" && checkOrdering {
					if _, err := os.Lstat(fx.tx); !errors.Is(err, os.ErrNotExist) {
						t.Fatal("ordinary deletion started before Restore storage was retired", err)
					}
					if _, err := b.restoreJournal(fx.e.CloneID); err != nil {
						t.Fatal("Restore journal lost before ordinary deletion", err)
					}
					if interrupt {
						interrupt = false
						return nil, errors.New("interrupted at start of ordinary deletion")
					}
					checkOrdering = false
				}
				return base(ctx, command, args...)
			}
			if interrupt {
				fx.assertBlocked(t, "clone runtime retirement incomplete")
				if _, err := b.readRecord(fx.e.CloneID); err != nil {
					t.Fatal("clone configuration and identity must survive interrupted retirement", err)
				}
			}
			fx.assertRecovered(t)
			if checkOrdering {
				t.Fatal("ordinary deletion did not run")
			}
		})
	}
}
