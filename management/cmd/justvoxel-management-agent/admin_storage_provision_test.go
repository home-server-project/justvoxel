package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAdminStorageProvisionRequiresAdministrator(t *testing.T) {
	old := runAdminStorageProvisionHelper
	defer func() { runAdminStorageProvisionHelper = old }()
	called := false
	runAdminStorageProvisionHelper = func(_ context.Context, _ string, _ []byte) ([]byte, error) {
		called = true
		return []byte(`{"ok":true,"warnings":[],"whole_disks":[],"blank_partitions":[],"free_space_disks":[]}`), nil
	}

	for _, role := range []principalRole{roleOperator, roleViewer} {
		s := surfaceTestServer(t, role)
		rr := httptest.NewRecorder()
		s.adminStorageProvisionDiscover(rr, surfaceRequest(http.MethodGet, "/v1/admin/storage-provision", ""))
		if rr.Code != http.StatusForbidden {
			t.Fatalf("role %s discover status = %d, want 403", role, rr.Code)
		}
	}
	if called {
		t.Fatal("advanced storage helper ran for non-Administrator")
	}
}

func TestAdminStorageProvisionPlanReturnsExactConfirmation(t *testing.T) {
	s := surfaceTestServer(t, roleAdministrator)
	old := runAdminStorageProvisionHelper
	defer func() { runAdminStorageProvisionHelper = old }()

	runAdminStorageProvisionHelper = func(_ context.Context, action string, request []byte) ([]byte, error) {
		if action != "plan" {
			t.Fatalf("action = %q, want plan", action)
		}
		if !strings.Contains(string(request), `"device":"/dev/vdb"`) {
			t.Fatalf("request missing selected device: %s", request)
		}
		return []byte(`{"ok":true,"warnings":["destructive"],"proposed":{"operation":"erase_disk","device":"/dev/vdb","model":"Virtual Disk","size_bytes":10737418240,"mount_point":"/var/mnt/justvoxel-backup","path":"/var/mnt/justvoxel-backup/backups","confirmation":"ERASE /dev/vdb","fingerprint":"abc123"},"applied":false}`), nil
	}

	body := `{"operation":"erase_disk","device":"/dev/vdb","mount_point":"/var/mnt/justvoxel-backup","path":"/var/mnt/justvoxel-backup/backups"}`
	rr := httptest.NewRecorder()
	s.adminStorageProvisionPlan(rr, surfaceRequest(http.MethodPost, "/v1/admin/storage-provision/plan", body))
	if rr.Code != http.StatusOK {
		t.Fatalf("plan status = %d: %s", rr.Code, rr.Body.String())
	}
	for _, want := range []string{"ERASE /dev/vdb", "abc123", "destructive"} {
		if !strings.Contains(rr.Body.String(), want) {
			t.Fatalf("plan response missing %q: %s", want, rr.Body.String())
		}
	}
}

func TestAdminStorageProvisionApplyReturnsBoundedFailure(t *testing.T) {
	s := surfaceTestServer(t, roleAdministrator)
	old := runAdminStorageProvisionHelper
	defer func() { runAdminStorageProvisionHelper = old }()

	runAdminStorageProvisionHelper = func(_ context.Context, action string, _ []byte) ([]byte, error) {
		if action != "apply" {
			t.Fatalf("action = %q, want apply", action)
		}
		return []byte(`{"ok":false,"error":"The selected device or disk layout changed after Review. Nothing was changed. Review the storage operation again.","warnings":[],"applied":false}`), nil
	}

	body := `{"operation":"erase_disk","device":"/dev/vdb","mount_point":"/var/mnt/justvoxel-backup","path":"/var/mnt/justvoxel-backup/backups","confirmation":"ERASE /dev/vdb","fingerprint":"old"}`
	rr := httptest.NewRecorder()
	s.adminStorageProvisionApply(rr, surfaceRequest(http.MethodPost, "/v1/admin/storage-provision/apply", body))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("apply status = %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "changed after Review") {
		t.Fatalf("bounded safety failure missing: %s", rr.Body.String())
	}
}
