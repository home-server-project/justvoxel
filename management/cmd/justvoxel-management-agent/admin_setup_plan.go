package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

const adminSetupPlanHelper = "/usr/libexec/justvoxel/mjust/admin-setup-plan-json"

var adminSetupPlanTimeout = 25 * time.Second

var runAdminSetupPlanHelper = func(ctx context.Context, request []byte) ([]byte, error) {
	cmd := exec.CommandContext(ctx, adminSetupPlanHelper)
	cmd.Stdin = bytes.NewReader(request)
	return cmd.CombinedOutput()
}

type adminSetupPlanServerRequest struct {
	MOTD           string `json:"motd"`
	MaxPlayers     int    `json:"max_players"`
	BedrockEnabled bool   `json:"bedrock_enabled"`
	Timezone       string `json:"timezone"`
}

type adminSetupPlanMinecraftRequest struct {
	JavaMemory      string `json:"java_memory"`
	ContainerMemory string `json:"container_memory"`
	JavaPort        int    `json:"java_port"`
	BedrockPort     int    `json:"bedrock_port"`
	ImageTag        string `json:"image_tag"`
	VersionPolicy   string `json:"version_policy"`
	Version         string `json:"version"`
}

type adminSetupPlanStorageRequest struct {
	Type       string `json:"type"`
	Path       string `json:"path"`
	Device     string `json:"device"`
	MountPoint string `json:"mount_point"`
}

type adminSetupPlanBackupsRequest struct {
	Automatic  bool   `json:"automatic"`
	DailyTime  string `json:"daily_time"`
	Keep       int    `json:"keep"`
	Type       string `json:"type"`
	Path       string `json:"path"`
	Device     string `json:"device"`
	MountPoint string `json:"mount_point"`
	Source     string `json:"source"`
	Username   string `json:"username"`
	Domain     string `json:"domain"`
}

type adminSetupPlanRequest struct {
	Server    adminSetupPlanServerRequest    `json:"server"`
	Minecraft adminSetupPlanMinecraftRequest `json:"minecraft"`
	Storage   adminSetupPlanStorageRequest   `json:"storage"`
	Backups   adminSetupPlanBackupsRequest   `json:"backups"`
}

type adminSetupPlanWarning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type adminSetupPlanServer struct {
	MOTD           string `json:"motd"`
	MaxPlayers     int    `json:"max_players"`
	BedrockEnabled bool   `json:"bedrock_enabled"`
	Timezone       string `json:"timezone"`
}

type adminSetupPlanMinecraft struct {
	JavaMemory             string `json:"java_memory"`
	ContainerMemory        string `json:"container_memory"`
	JavaPort               int    `json:"java_port"`
	BedrockPort            int    `json:"bedrock_port"`
	ImageTag               string `json:"image_tag"`
	RequestedVersionPolicy string `json:"requested_version_policy"`
	VersionPolicy          string `json:"version_policy"`
	Version                string `json:"version"`
	SystemMemoryMiB        int    `json:"system_memory_mib"`
	SystemReserveMiB       int    `json:"system_reserve_mib"`
}

type adminSetupPlanStorage struct {
	Type           string `json:"type"`
	Path           string `json:"path"`
	Device         string `json:"device,omitempty"`
	ParentDisk     string `json:"parent_disk,omitempty"`
	Model          string `json:"model,omitempty"`
	Transport      string `json:"transport,omitempty"`
	SizeBytes      uint64 `json:"size_bytes,omitempty"`
	Filesystem     string `json:"filesystem,omitempty"`
	UUID           string `json:"uuid,omitempty"`
	MountPoint     string `json:"mount_point,omitempty"`
	ExpectedUUID   string `json:"expected_uuid,omitempty"`
	ExpectedSource string `json:"expected_source,omitempty"`
	MountedAt      string `json:"mounted_at,omitempty"`
	Source         string `json:"source,omitempty"`
	SystemDisk     bool   `json:"system_disk"`
	Purpose        string `json:"purpose,omitempty"`
}

type adminSetupPlanBackups struct {
	Type                string `json:"type"`
	Path                string `json:"path"`
	Device              string `json:"device,omitempty"`
	ParentDisk          string `json:"parent_disk,omitempty"`
	Model               string `json:"model,omitempty"`
	Transport           string `json:"transport,omitempty"`
	SizeBytes           uint64 `json:"size_bytes,omitempty"`
	Filesystem          string `json:"filesystem,omitempty"`
	UUID                string `json:"uuid,omitempty"`
	MountPoint          string `json:"mount_point,omitempty"`
	ExpectedUUID        string `json:"expected_uuid,omitempty"`
	ExpectedSource      string `json:"expected_source,omitempty"`
	MountedAt           string `json:"mounted_at,omitempty"`
	Source              string `json:"source,omitempty"`
	SystemDisk          bool   `json:"system_disk"`
	Purpose             string `json:"purpose,omitempty"`
	Username            string `json:"username,omitempty"`
	Domain              string `json:"domain,omitempty"`
	CredentialsRequired bool   `json:"credentials_required"`
	Automatic           bool   `json:"automatic"`
	DailyTime           string `json:"daily_time"`
	Schedule            string `json:"schedule"`
	Keep                int    `json:"keep"`
}

type adminSetupNormalizedPlan struct {
	Server    adminSetupPlanServer    `json:"server"`
	Minecraft adminSetupPlanMinecraft `json:"minecraft"`
	Storage   adminSetupPlanStorage   `json:"storage"`
	Backups   adminSetupPlanBackups   `json:"backups"`
}

type adminSetupPlanRequirements struct {
	SMBPasswordRequired            bool `json:"smb_password_required"`
	NetworkBackupValidationOnApply bool `json:"network_backup_validation_on_apply"`
}

type adminSetupPlanResponse struct {
	OK            bool                        `json:"ok"`
	SchemaVersion string                      `json:"schema_version"`
	Code          string                      `json:"code,omitempty"`
	Error         string                      `json:"error,omitempty"`
	Normalized    *adminSetupNormalizedPlan   `json:"normalized,omitempty"`
	Warnings      []adminSetupPlanWarning     `json:"warnings"`
	Requirements  *adminSetupPlanRequirements `json:"requirements,omitempty"`
}

func registerAdminSetupPlanRoutes(mux *http.ServeMux, s *server) {
	mux.HandleFunc("POST /v1/admin/setup/plan", s.adminSetupPlan)
}

func (s *server) adminSetupPlan(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdministrator(w, r); !ok {
		return
	}

	var request adminSetupPlanRequest
	if !decodeAdminSetupPlanRequest(w, r, &request) {
		return
	}
	payload, err := json.Marshal(request)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "first-run setup request could not be prepared")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), adminSetupPlanTimeout)
	defer cancel()
	output, err := runAdminSetupPlanHelper(ctx, payload)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "first-run setup planning is unavailable")
		return
	}

	var out adminSetupPlanResponse
	if err := decodeAdminSetupPlanResponse(output, &out); err != nil {
		writeError(w, http.StatusInternalServerError, "first-run setup planner returned invalid data")
		return
	}
	if out.SchemaVersion != "v1" {
		writeError(w, http.StatusInternalServerError, "first-run setup planner returned an unsupported schema")
		return
	}
	if out.Warnings == nil {
		out.Warnings = []adminSetupPlanWarning{}
	}

	if !out.OK {
		out.Error = boundedSetupPlanError(out.Error)
		writeJSON(w, http.StatusBadRequest, out)
		return
	}
	if out.Normalized == nil || out.Requirements == nil || out.Normalized.Minecraft.VersionPolicy == "" || out.Normalized.Storage.Type == "" || out.Normalized.Backups.Type == "" {
		writeError(w, http.StatusInternalServerError, "first-run setup planner returned incomplete data")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func decodeAdminSetupPlanRequest(w http.ResponseWriter, r *http.Request, target *adminSetupPlanRequest) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16*1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return false
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return false
	}
	return true
}

func decodeAdminSetupPlanResponse(data []byte, target *adminSetupPlanResponse) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return errors.New("unexpected trailing JSON value")
		}
		return err
	}
	return nil
}

func boundedSetupPlanError(message string) string {
	const fallback = "first-run setup could not be validated"
	if message == "" || len(message) > 512 || strings.ContainsAny(message, "\r\n") {
		return fallback
	}
	return message
}
