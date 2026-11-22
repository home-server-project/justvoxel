package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"time"
)

const adminBackupStorageHelper = "/usr/libexec/justvoxel/mjust/admin-backup-storage-json"

type adminBackupStorageRequest struct {
	Type       string `json:"type"`
	Path       string `json:"path"`
	Device     string `json:"device,omitempty"`
	MountPoint string `json:"mount_point,omitempty"`
	Source     string `json:"source,omitempty"`
	Username   string `json:"username,omitempty"`
	Password   string `json:"password,omitempty"`
	Domain     string `json:"domain,omitempty"`
}

type adminBackupStorageTarget struct {
	Status            string `json:"status,omitempty"`
	StatusDetail      string `json:"status_detail,omitempty"`
	Type              string `json:"type"`
	Path              string `json:"path"`
	Device            string `json:"device,omitempty"`
	MountPoint        string `json:"mount_point,omitempty"`
	ExpectedUUID      string `json:"expected_uuid,omitempty"`
	ExpectedSource    string `json:"expected_source,omitempty"`
	Filesystem        string `json:"filesystem,omitempty"`
	Model             string `json:"model,omitempty"`
	SizeBytes         uint64 `json:"size_bytes,omitempty"`
	AvailableBytes    uint64 `json:"available_bytes,omitempty"`
	FilesystemBytes   uint64 `json:"filesystem_bytes,omitempty"`
	SamePhysicalDisk  bool   `json:"same_physical_disk,omitempty"`
	AlreadyMounted    bool   `json:"already_mounted,omitempty"`
	CredentialsNeeded bool   `json:"credentials_required,omitempty"`
}

type adminBackupStorageResponse struct {
	OK       bool                     `json:"ok"`
	Error    string                   `json:"error,omitempty"`
	Current  adminBackupStorageTarget `json:"current"`
	Proposed adminBackupStorageTarget `json:"proposed"`
	Warnings []string                 `json:"warnings"`
	Changed  bool                     `json:"changed"`
	Applied  bool                     `json:"applied"`
}

var runAdminBackupStorageHelper = func(ctx context.Context, action string, request []byte) ([]byte, error) {
	cmd := exec.CommandContext(ctx, adminBackupStorageHelper, action)
	if len(request) > 0 {
		cmd.Stdin = bytes.NewReader(request)
	}
	return cmd.CombinedOutput()
}

func registerAdminBackupStorageRoutes(mux *http.ServeMux, s *server) {
	mux.HandleFunc("GET /v1/admin/backup-storage", s.adminBackupStorageStatus)
	mux.HandleFunc("POST /v1/admin/backup-storage/plan", s.adminBackupStoragePlan)
	mux.HandleFunc("POST /v1/admin/backup-storage/apply", s.adminBackupStorageApply)
}

func (s *server) adminBackupStorageStatus(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdministrator(w, r); !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	output, err := runAdminBackupStorageHelper(ctx, "status", nil)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "backup storage status is unavailable")
		return
	}
	var out adminBackupStorageResponse
	if err := json.Unmarshal(output, &out); err != nil || !out.OK {
		writeError(w, http.StatusInternalServerError, "backup storage status returned invalid data")
		return
	}
	if out.Warnings == nil {
		out.Warnings = []string{}
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *server) adminBackupStoragePlan(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdministrator(w, r); !ok {
		return
	}
	s.runAdminBackupStorageChange(w, r, "plan", session{})
}

func (s *server) adminBackupStorageApply(w http.ResponseWriter, r *http.Request) {
	actor, ok := s.requireAdministrator(w, r)
	if !ok {
		return
	}
	s.runAdminBackupStorageChange(w, r, "apply", actor)
}

func (s *server) runAdminBackupStorageChange(w http.ResponseWriter, r *http.Request, action string, actor session) {
	var request adminBackupStorageRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	payload, err := json.Marshal(request)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "backup storage request could not be prepared")
		return
	}
	timeout := 12 * time.Second
	if action == "apply" {
		timeout = 28 * time.Second
	}
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()
	output, err := runAdminBackupStorageHelper(ctx, action, payload)
	if err != nil {
		if action == "apply" {
			_ = s.store.recordAuditEvent(actor, "update_backup_storage", request.Type, false, "backup storage apply failed")
		}
		writeError(w, http.StatusServiceUnavailable, "backup storage operation failed; previous settings were preserved")
		return
	}
	var out adminBackupStorageResponse
	if err := json.Unmarshal(output, &out); err != nil {
		writeError(w, http.StatusInternalServerError, "backup storage operation returned invalid data")
		return
	}
	if out.Warnings == nil {
		out.Warnings = []string{}
	}
	if !out.OK {
		if out.Error == "" {
			out.Error = "backup storage could not be validated"
		}
		if action == "apply" {
			_ = s.store.recordAuditEvent(actor, "update_backup_storage", request.Type, false, "backup storage validation or apply failed")
		}
		writeJSON(w, http.StatusBadRequest, out)
		return
	}
	if action == "apply" {
		context := fmt.Sprintf("type=%s changed=%t", out.Proposed.Type, out.Changed)
		_ = s.store.recordAuditEvent(actor, "update_backup_storage", out.Proposed.Path, true, context)
	}
	writeJSON(w, http.StatusOK, out)
}
