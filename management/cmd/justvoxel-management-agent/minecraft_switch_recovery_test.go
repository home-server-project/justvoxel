package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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
			source := instanceRuntimeRecord{Runtime: minecraftInstanceRuntimeConfig{ID: sourceID, Software: "paper"}, Storage: instanceRuntimeStorage{Root: root, DataPath: filepath.Join(sourcePath, "data")}}
			clone := instanceRuntimeRecord{Runtime: minecraftInstanceRuntimeConfig{ID: cloneID, Software: "purpur"}, Storage: instanceRuntimeStorage{Root: root, DataPath: filepath.Join(clonePath, "data")}}
			e := minecraftSwitchEvidence{Source: sourceID, Target: "purpur", Version: "1.21.4", Action: "clone", SourceRecord: &source, CloneID: cloneID, CloneRecord: &clone}
			if os.Geteuid() != 0 {
				switch name {
				case "empty", "wrong mode", "data", "provision marker", "unexpected entry":
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
			if err := removeEmptySwitchCloneParent(e); (err == nil) != wantRemoved {
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
