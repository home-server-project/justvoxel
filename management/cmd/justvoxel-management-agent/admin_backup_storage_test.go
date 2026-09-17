package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAdminBackupStorageRequiresAdministrator(t *testing.T) {
	old := runAdminBackupStorageHelper
	defer func() { runAdminBackupStorageHelper = old }()
	called := false
	runAdminBackupStorageHelper = func(_ context.Context, _ string, _ []byte) ([]byte, error) {
		called = true
		return []byte(`{"ok":true,"current":{"type":"system","path":"/var/lib/justvoxel/backups"},"warnings":[]}`), nil
	}

	for _, role := range []principalRole{roleOperator, roleViewer} {
		s := surfaceTestServer(t, role)
		rr := httptest.NewRecorder()
		s.adminBackupStoragePlan(rr, surfaceRequest(http.MethodPost, "/v1/admin/backup-storage/plan", `{}`))
		if rr.Code != http.StatusForbidden {
			t.Fatalf("role %s plan status = %d, want 403", role, rr.Code)
		}
	}
	if called {
		t.Fatal("backup storage helper ran for a non-Administrator")
	}
}

func TestAdminBackupStoragePlanUsesFixedHelperWithoutReturningCredentials(t *testing.T) {
	s := surfaceTestServer(t, roleAdministrator)
	old := runAdminBackupStorageHelper
	defer func() { runAdminBackupStorageHelper = old }()

	runAdminBackupStorageHelper = func(_ context.Context, action string, request []byte) ([]byte, error) {
		if action != "plan" {
			t.Fatalf("action = %q, want plan", action)
		}
		body := string(request)
		for _, want := range []string{`"type":"smb"`, `"source":"//nas/backups"`, `"username":"backup-user"`} {
			if !strings.Contains(body, want) {
				t.Fatalf("helper request missing %s: %s", want, body)
			}
		}
		return []byte(`{"ok":true,"current":{"type":"system","path":"/var/lib/justvoxel/backups"},"proposed":{"type":"smb","path":"/var/mnt/justvoxel-backup/backups","mount_point":"/var/mnt/justvoxel-backup","expected_source":"//nas/backups","credentials_required":true},"warnings":["Network backups depend on the SMB server being reachable."],"changed":true,"applied":false}`), nil
	}

	body := `{"type":"smb","path":"/var/mnt/justvoxel-backup/backups","mount_point":"/var/mnt/justvoxel-backup","source":"//nas/backups","username":"backup-user","password":"do-not-echo","domain":"HOME"}`
	rr := httptest.NewRecorder()
	s.adminBackupStoragePlan(rr, surfaceRequest(http.MethodPost, "/v1/admin/backup-storage/plan", body))
	if rr.Code != http.StatusOK {
		t.Fatalf("plan status = %d: %s", rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), "do-not-echo") || strings.Contains(rr.Body.String(), "backup-user") {
		t.Fatalf("credentials leaked in plan response: %s", rr.Body.String())
	}
	for _, want := range []string{"//nas/backups", `"credentials_required":true`, `"changed":true`} {
		if !strings.Contains(rr.Body.String(), want) {
			t.Fatalf("plan response missing %q: %s", want, rr.Body.String())
		}
	}
}

func TestAdminBackupStorageApplyErrorsAreSanitized(t *testing.T) {
	s := surfaceTestServer(t, roleAdministrator)
	old := runAdminBackupStorageHelper
	defer func() { runAdminBackupStorageHelper = old }()
	runAdminBackupStorageHelper = func(_ context.Context, _ string, _ []byte) ([]byte, error) {
		return []byte("password=do-not-leak"), errors.New("exit status 1")
	}

	rr := httptest.NewRecorder()
	s.adminBackupStorageApply(rr, surfaceRequest(http.MethodPost, "/v1/admin/backup-storage/apply", `{"type":"smb"}`))
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("apply status = %d: %s", rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), "do-not-leak") {
		t.Fatalf("helper output leaked: %s", rr.Body.String())
	}
}

func TestAdminBackupStorageValidationMessageIsStructured(t *testing.T) {
	s := surfaceTestServer(t, roleAdministrator)
	old := runAdminBackupStorageHelper
	defer func() { runAdminBackupStorageHelper = old }()
	runAdminBackupStorageHelper = func(_ context.Context, _ string, _ []byte) ([]byte, error) {
		return []byte(`{"ok":false,"error":"The selected partition has no filesystem. Formatting belongs to Advanced storage provisioning (A3.2).","warnings":[]}`), nil
	}

	rr := httptest.NewRecorder()
	s.adminBackupStoragePlan(rr, surfaceRequest(http.MethodPost, "/v1/admin/backup-storage/plan", `{"type":"partition"}`))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("validation status = %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "A3.2") {
		t.Fatalf("validation message missing: %s", rr.Body.String())
	}
}
