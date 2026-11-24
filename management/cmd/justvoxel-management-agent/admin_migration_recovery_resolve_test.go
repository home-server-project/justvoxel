package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

const testMigrationRecoveryFingerprint = "sha256:9999999999999999999999999999999999999999999999999999999999999999"

func beginAttentionMigrationRecovery(t *testing.T, store *operationStore) operationJournal {
	t.Helper()
	operation, _, err := store.beginMigrationRecovery("jv-a1b2c3", testMigrationRecoveryFingerprint)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.transition(operation.OperationID, operationValidating, "recovery_preflight", "Checking recovery."); err != nil {
		t.Fatal(err)
	}
	if _, err := store.transition(operation.OperationID, operationNeedsAttention, "recovery_backend_invalid", "Recovery backend returned invalid progress data."); err != nil {
		t.Fatal(err)
	}
	return operation
}

func TestAdminMigrationRecoveryResolvePersistedOperation(t *testing.T) {
	base := filepath.Join(t.TempDir(), "management")
	first, err := openOperationStore(base)
	if err != nil {
		t.Fatal(err)
	}
	operation := beginAttentionMigrationRecovery(t, first)
	if err := first.close(); err != nil {
		t.Fatal(err)
	}
	store, err := openOperationStore(base)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.close() })
	s := migrationSurfaceTestServer(t, roleAdministrator)
	attachTestOperationStore(t, s, store)
	previous := runAdminMigrationRecoveryHelper
	runAdminMigrationRecoveryHelper = func(_ context.Context, action string, _ []byte) ([]byte, error) {
		if action != "resolve-check" {
			t.Fatalf("unexpected helper action %q", action)
		}
		return []byte(`{"ok":true,"schema_version":"v1","safe_to_resolve":true}`), nil
	}
	t.Cleanup(func() { runAdminMigrationRecoveryHelper = previous })

	body, _ := json.Marshal(adminMigrationRecoveryResolveRequest{OperationID: operation.OperationID, KeepCurrentState: true})
	rr := httptest.NewRecorder()
	s.adminMigrationRecoveryResolve(rr, migrationSurfaceRequest(http.MethodPost, "/v1/minecraft/instances/jv-a1b2c3/migration/recovery/resolve", string(body)))
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), `"state":"resolved"`) {
		t.Fatalf("resolve status=%d: %s", rr.Code, rr.Body.String())
	}
	journal, err := store.get(operation.OperationID)
	if err != nil || journal.State != operationResolved || journal.FinishedAt == "" {
		t.Fatalf("resolved journal = %#v, err=%v", journal, err)
	}
	persisted, err := readOperationJournal(filepath.Join(store.operationsDir, operation.OperationID+".json"))
	if err != nil || persisted.State != operationResolved || persisted.FinishedAt == "" {
		t.Fatalf("persisted journal = %#v, err=%v", persisted, err)
	}
	if store.currentMigrationID != "" || store.migrationLockHeld {
		t.Fatalf("terminal transition retained migration ownership: id=%q lock=%t", store.currentMigrationID, store.migrationLockHeld)
	}
	if current, err := store.currentMigration(); err != nil || current != nil {
		t.Fatalf("recovery remained current: %#v err=%v", current, err)
	}
	if _, _, err := store.beginMigrationExport("jv-a1b2c3", testMigrationExportFingerprint); err != nil {
		t.Fatalf("migration lock remained held: %v", err)
	}
}

func TestAdminMigrationRecoveryResolveFailsClosed(t *testing.T) {
	for _, tc := range []struct {
		name       string
		output     string
		wrongID    bool
		wrongType  bool
		running    bool
		confirmed  bool
		wantStatus int
	}{
		{name: "unsafe", output: `{"ok":true,"schema_version":"v1","safe_to_resolve":false}`, confirmed: true, wantStatus: http.StatusConflict},
		{name: "malformed", output: `{`, confirmed: true, wantStatus: http.StatusInternalServerError},
		{name: "wrong schema", output: `{"ok":true,"schema_version":"v2","safe_to_resolve":true}`, confirmed: true, wantStatus: http.StatusInternalServerError},
		{name: "wrong id", output: `{"ok":true,"schema_version":"v1","safe_to_resolve":true}`, wrongID: true, confirmed: true, wantStatus: http.StatusNotFound},
		{name: "wrong type", output: `{"ok":true,"schema_version":"v1","safe_to_resolve":true}`, wrongType: true, confirmed: true, wantStatus: http.StatusConflict},
		{name: "running", output: `{"ok":true,"schema_version":"v1","safe_to_resolve":true}`, running: true, confirmed: true, wantStatus: http.StatusConflict},
		{name: "confirmation missing", output: `{"ok":true,"schema_version":"v1","safe_to_resolve":true}`, wantStatus: http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := migrationSurfaceTestServer(t, roleAdministrator)
			store := openTestOperationStore(t)
			attachTestOperationStore(t, s, store)
			var operation operationJournal
			var err error
			if tc.wrongType {
				operation, _, err = store.beginMigrationImport("jv-a1b2c3", testMigrationImportFingerprint)
			} else {
				operation, _, err = store.beginMigrationRecovery("jv-a1b2c3", testMigrationRecoveryFingerprint)
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err := store.transition(operation.OperationID, operationValidating, "recovery_preflight", "Checking recovery."); err != nil {
				t.Fatal(err)
			}
			if tc.running {
				if _, err := store.transition(operation.OperationID, operationRunning, "recovering", "Recovering."); err != nil {
					t.Fatal(err)
				}
			} else {
				if _, err := store.transition(operation.OperationID, operationNeedsAttention, "recovery_backend_invalid", "Recovery needs attention."); err != nil {
					t.Fatal(err)
				}
			}
			previous := runAdminMigrationRecoveryHelper
			runAdminMigrationRecoveryHelper = func(_ context.Context, action string, _ []byte) ([]byte, error) {
				if action != "resolve-check" {
					t.Fatalf("unexpected helper action %q", action)
				}
				return []byte(tc.output), nil
			}
			t.Cleanup(func() { runAdminMigrationRecoveryHelper = previous })
			id := operation.OperationID
			if tc.wrongID {
				id = "12345678-1234-4123-8123-123456789abc"
			}
			body, _ := json.Marshal(adminMigrationRecoveryResolveRequest{OperationID: id, KeepCurrentState: tc.confirmed})
			rr := httptest.NewRecorder()
			s.adminMigrationRecoveryResolve(rr, migrationSurfaceRequest(http.MethodPost, "/v1/minecraft/instances/jv-a1b2c3/migration/recovery/resolve", string(body)))
			if rr.Code != tc.wantStatus {
				t.Fatalf("status=%d want=%d: %s", rr.Code, tc.wantStatus, rr.Body.String())
			}
			current, err := store.currentMigration()
			if err != nil || current == nil || current.OperationID != operation.OperationID {
				t.Fatalf("current = %#v, err=%v", current, err)
			}
			if _, _, err := store.beginMigrationExport("jv-a1b2c3", testMigrationExportFingerprint); !errors.Is(err, errMigrationOperationBusy) {
				t.Fatalf("migration lock released after refusal: %v", err)
			}
		})
	}
}

func TestMigrationRecoveryRetryAfterRestart(t *testing.T) {
	first := openTestOperationStore(t)
	original := beginAttentionMigrationRecovery(t, first)
	if err := first.close(); err != nil {
		t.Fatal(err)
	}
	store, err := openOperationStore(first.baseDir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.close()
	s := migrationSurfaceTestServer(t, roleAdministrator)
	s.operations = store
	previousHelper, previousWorker := runAdminMigrationRecoveryHelper, startMigrationRecoveryWorker
	defer func() {
		runAdminMigrationRecoveryHelper, startMigrationRecoveryWorker = previousHelper, previousWorker
	}()
	plan := adminMigrationRecoveryHelperResponse{
		OK: true, SchemaVersion: "v1",
		Normalized:   &adminMigrationRecoverySummary{Transaction: "/data/.justvoxel-import-test", Finalizable: true},
		Context:      &adminMigrationRecoveryContext{StateIdentity: strings.Repeat("a", 64)},
		Requirements: &adminMigrationRecoveryRequirements{FinalizeConfirmationRequired: true},
	}
	fingerprint, err := adminMigrationRecoveryFingerprint("v1", plan.Normalized, plan.Requirements, plan.Context)
	if err != nil {
		t.Fatal(err)
	}
	runAdminMigrationRecoveryHelper = func(ctx context.Context, action string, payload []byte) ([]byte, error) {
		if action != "plan" || migrationInstanceID(ctx) != original.InstanceID {
			return nil, errors.New("unexpected recovery planner invocation")
		}
		return json.Marshal(plan)
	}
	workers := make(chan string, 8)
	startMigrationRecoveryWorker = func(_ *server, id string, _ migrationRecoveryExecutionPlan) { workers <- id }
	apply := func(id, reviewed string) int {
		body, _ := json.Marshal(adminMigrationRecoveryApplyRequest{reviewed, plan.Normalized.Transaction, true})
		req := surfaceRequest("POST", "/v1/minecraft/instances/"+id+"/migration/recovery/apply", string(body))
		rr := httptest.NewRecorder()
		s.adminMigrationRecoveryApply(rr, req)
		return rr.Code
	}
	before, err := store.get(original.OperationID)
	if err != nil {
		t.Fatal(err)
	}
	if apply("jv-d4e5f6", fingerprint) < 400 || apply(original.InstanceID, testMigrationRecoveryFingerprint) < 400 {
		t.Fatal("wrong destination or stale plan admitted")
	}
	plan.Normalized.Finalizable = false
	invalid, err := adminMigrationRecoveryFingerprint("v1", plan.Normalized, plan.Requirements, plan.Context)
	if err != nil || apply(original.InstanceID, invalid) < 400 {
		t.Fatal("nonfinalizable plan admitted", err)
	}
	plan.Normalized.Finalizable = true
	if after, err := store.get(original.OperationID); err != nil || after != before || len(workers) != 0 {
		t.Fatal("rejected review altered recovery", after, err)
	}
	results := make(chan int, 2)
	for i := 0; i < 2; i++ {
		go func() { results <- apply(original.InstanceID, fingerprint) }()
	}
	a, b := <-results, <-results
	if !((a == 200 && b == 202) || (a == 202 && b == 200)) || len(workers) != 1 || <-workers != original.OperationID {
		t.Fatalf("concurrent retry responses %d and %d", a, b)
	}
	if _, err := store.transition(original.OperationID, operationNeedsAttention, "retry_failed", "Recovery failed again."); err != nil {
		t.Fatal(err)
	}
	if apply(original.InstanceID, fingerprint) != 202 || len(workers) != 1 || <-workers != original.OperationID {
		t.Fatal("freshly reviewed second retry did not start")
	}
	for _, state := range []operationState{operationQueued, operationValidating, operationRunning, operationVerifying, operationSucceeded} {
		if state != operationQueued {
			if _, err := store.transition(original.OperationID, state, "retry_progress", "Recovery progress."); err != nil {
				t.Fatal(err)
			}
		}
		if state == operationSucceeded {
			if store.migrationLockHeld || store.currentMigrationID != "" {
				t.Fatal("successful recovery retained lock")
			}
		} else if !store.migrationLockHeld || store.currentMigrationID != original.OperationID ||
			apply(original.InstanceID, fingerprint) != 200 || len(workers) != 0 {
			t.Fatal("lost ownership or duplicated worker", state)
		}
	}
}

func TestMigrationRetryExcludesOtherOperations(t *testing.T) {
	store := openTestOperationStore(t)
	op, _, err := store.beginMigrationImport("jv-a1b2c3", testMigrationImportFingerprint)
	if err != nil {
		t.Fatal(err)
	}
	op, err = store.transition(op.OperationID, operationNeedsAttention, "failed", "Import requires recovery.")
	if err != nil {
		t.Fatal(err)
	}
	if _, start, err := store.retryMigrationRecovery(op, testMigrationRecoveryFingerprint); err == nil || start {
		t.Fatal("Import requeued as Recovery")
	}
	if _, _, err := store.beginSetup(testSetupFingerprint); err == nil {
		t.Fatal("Setup admitted while migration owns exclusion")
	}
	for _, begin := range []func(string) (operationJournal, bool, error){store.beginRestore, store.beginDataMigration, store.beginMinecraftReset} {
		if _, _, err := begin(testSetupFingerprint); err == nil {
			t.Fatal("competing operation admitted while migration owns exclusion")
		}
	}
}
