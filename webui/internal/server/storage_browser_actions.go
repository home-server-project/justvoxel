package server

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/home-server-project/justvoxel-webui/internal/api"
)

type adminStorageActionAPI interface {
	adminDiscoveryAPI
	AdminStorageActionPlan(ctx context.Context, session string, request api.AdminStorageActionRequest) (api.AdminStorageActionResponse, error)
	AdminStorageActionApply(ctx context.Context, session string, request api.AdminStorageActionRequest) (api.AdminStorageActionResponse, error)
}

type adminStorageMountAPI interface {
	AdminStorageMountStatus(ctx context.Context, session, device string) (api.AdminStorageMountResponse, error)
	AdminStorageMountPlan(ctx context.Context, session string, request api.AdminStorageMountRequest) (api.AdminStorageMountResponse, error)
	AdminStorageMountApply(ctx context.Context, session string, request api.AdminStorageMountRequest) (api.AdminStorageMountResponse, error)
}

type adminStorageWholeDiskProvisionAPI interface {
	AdminStorageProvisionPlan(ctx context.Context, session string, request api.AdminStorageProvisionRequest) (api.AdminStorageProvisionResponse, error)
	AdminStorageProvisionApply(ctx context.Context, session string, request api.AdminStorageProvisionRequest) (api.AdminStorageProvisionResponse, error)
}

type adminStorageWholeDiskMigrationAPI interface {
	dataMigrationPlanApplyAPI
}

type storageBrowserWholeDiskResponse struct {
	OK                          bool     `json:"ok"`
	Error                       string   `json:"error,omitempty"`
	Purpose                     string   `json:"purpose,omitempty"`
	Device                      string   `json:"device,omitempty"`
	Model                       string   `json:"model,omitempty"`
	Transport                   string   `json:"transport,omitempty"`
	SizeBytes                   uint64   `json:"size_bytes,omitempty"`
	MountPoint                  string   `json:"mount_point,omitempty"`
	Path                        string   `json:"path,omitempty"`
	Fingerprint                 string   `json:"fingerprint,omitempty"`
	Confirmation                string   `json:"confirmation,omitempty"`
	Warnings                    []string `json:"warnings"`
	PlayersConfirmationRequired bool     `json:"players_confirmation_required,omitempty"`
	Online                      int      `json:"online,omitempty"`
	Players                     []string `json:"players,omitempty"`
	Applied                     bool     `json:"applied,omitempty"`
	OperationID                 string   `json:"operation_id,omitempty"`
}

func (a *App) registerStorageBrowserActionRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/new-storage/backup-partition/plan", a.storageBrowserBackupPartitionPlan)
	mux.HandleFunc("POST /api/new-storage/backup-partition/apply", a.storageBrowserBackupPartitionApply)
	mux.HandleFunc("POST /api/new-storage/actions/plan", a.storageBrowserActionPlan)
	mux.HandleFunc("POST /api/new-storage/actions/apply", a.storageBrowserActionApply)
	mux.HandleFunc("GET /api/new-storage/mounts/status", a.storageBrowserMountStatus)
	mux.HandleFunc("POST /api/new-storage/mounts/plan", a.storageBrowserMountPlan)
	mux.HandleFunc("POST /api/new-storage/mounts/apply", a.storageBrowserMountApply)
	mux.HandleFunc("POST /api/new-storage/whole-disk/plan", a.storageBrowserWholeDiskPlan)
	mux.HandleFunc("POST /api/new-storage/whole-disk/apply", a.storageBrowserWholeDiskApply)
	mux.HandleFunc("GET /api/new-storage/minecraft-data/current", a.storageMinecraftMigrationCurrent)
	mux.HandleFunc("POST /api/new-storage/minecraft-data/plan", a.storageMinecraftMigrationPlan)
	mux.HandleFunc("POST /api/new-storage/minecraft-data/apply", a.storageMinecraftMigrationApply)
	mux.HandleFunc("GET /api/new-storage/minecraft-data/progress/{id}", a.storageMinecraftMigrationProgress)
}

func (a *App) storageBrowserWholeDiskPlan(w http.ResponseWriter, r *http.Request) {
	a.storageBrowserWholeDiskChange(w, r, false)
}

func (a *App) storageBrowserWholeDiskApply(w http.ResponseWriter, r *http.Request) {
	a.storageBrowserWholeDiskChange(w, r, true)
}

func (a *App) storageBrowserWholeDiskChange(w http.ResponseWriter, r *http.Request, apply bool) {
	if !a.validCSRF(r) {
		writeStorageBrowserWholeDiskJSON(w, http.StatusForbidden, storageBrowserWholeDiskResponse{OK: false, Error: "invalid CSRF token"})
		return
	}
	session, discovery, _, ok := a.adminDiscoveryRequest(w, r)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		writeStorageBrowserWholeDiskJSON(w, http.StatusBadRequest, storageBrowserWholeDiskResponse{OK: false, Error: ui(w, "ui.c3701980b10d")})
		return
	}
	purpose := strings.TrimSpace(r.FormValue("purpose"))
	device := strings.TrimSpace(r.FormValue("device"))
	if device == "" || (purpose != "minecraft" && purpose != "backups") {
		writeStorageBrowserWholeDiskJSON(w, http.StatusBadRequest, storageBrowserWholeDiskResponse{OK: false, Error: ui(w, "ui.202907bbca95")})
		return
	}

	if purpose == "minecraft" {
		storage, err := discovery.AdminStorage(r.Context(), session)
		if err != nil {
			writeStorageBrowserWholeDiskJSON(w, storageBrowserErrorStatus(err), storageBrowserWholeDiskResponse{OK: false, Error: apiMessage(err, ui(w, "ui.246f08b71730"))})
			return
		}
		for _, candidate := range storage.Devices {
			if candidate.Path == device && strings.EqualFold(strings.TrimSpace(candidate.Transport), "usb") {
				writeStorageBrowserWholeDiskJSON(w, http.StatusBadRequest, storageBrowserWholeDiskResponse{OK: false, Error: ui(w, "ui.667dd1dda399")})
				return
			}
		}
	}

	if purpose == "backups" {
		client, ok := a.api.(adminStorageWholeDiskProvisionAPI)
		if !ok {
			writeStorageBrowserWholeDiskJSON(w, http.StatusServiceUnavailable, storageBrowserWholeDiskResponse{OK: false, Error: ui(w, "ui.bae0196de135")})
			return
		}
		request := api.AdminStorageProvisionRequest{
			Operation:  "erase_disk",
			Device:     device,
			MountPoint: "/var/mnt/justvoxel-backup",
			Path:       "/var/mnt/justvoxel-backup/backups",
			SizeGiB:    "all",
		}
		if apply {
			request.Fingerprint = strings.TrimSpace(r.FormValue("fingerprint"))
			request.Confirmation = r.FormValue("confirmation")
		}
		var result api.AdminStorageProvisionResponse
		var err error
		if apply {
			result, err = client.AdminStorageProvisionApply(r.Context(), session, request)
		} else {
			result, err = client.AdminStorageProvisionPlan(r.Context(), session, request)
		}
		if err != nil {
			writeStorageBrowserWholeDiskJSON(w, storageBrowserErrorStatus(err), storageBrowserWholeDiskResponse{OK: false, Error: apiMessage(err, ui(w, "ui.ce779051582d"))})
			return
		}
		if apply && !result.Applied {
			writeStorageBrowserWholeDiskJSON(w, http.StatusConflict, storageBrowserWholeDiskResponse{OK: false, Error: ui(w, "ui.e4e1e8b434d0")})
			return
		}
		writeStorageBrowserWholeDiskJSON(w, http.StatusOK, storageBrowserWholeDiskResponse{
			OK: true, Purpose: purpose, Device: result.Proposed.Device, Model: result.Proposed.Model,
			Transport: result.Proposed.Transport, SizeBytes: result.Proposed.SizeBytes,
			MountPoint: result.Proposed.MountPoint, Path: result.Proposed.Path,
			Fingerprint: result.Proposed.Fingerprint, Confirmation: result.Proposed.Confirmation,
			Warnings: result.Warnings, Applied: result.Applied,
		})
		return
	}

	client, ok := a.api.(adminStorageWholeDiskMigrationAPI)
	if !ok {
		writeStorageBrowserWholeDiskJSON(w, http.StatusServiceUnavailable, storageBrowserWholeDiskResponse{OK: false, Error: ui(w, "ui.58ee50dc5f62")})
		return
	}
	request := api.AdminDataMigrationPlanRequest{
		Operation:  "erase_disk",
		Device:     device,
		MountPoint: "/var/mnt/justvoxel-data",
		Path:       "/var/mnt/justvoxel-data/minecraft",
		SizeGiB:    "all",
	}

	var plan api.AdminDataMigrationPlanResponse
	var operation *api.PersistentOperation
	var err error
	if apply {
		plan, operation, err = dataMigrationApplyReviewed(r.Context(), client, session, request, dataMigrationApplyProof{
			PlanFingerprint:    strings.TrimSpace(r.FormValue("fingerprint")),
			MigrationConfirmed: true,
			Confirmation:       r.FormValue("confirmation"),
			PlayersConfirmed:   r.FormValue("players_confirmed") == "yes",
		})
	} else {
		plan, err = dataMigrationPlanReviewed(r.Context(), client, session, request)
	}
	if err != nil {
		writeStorageBrowserWholeDiskJSON(w, dataMigrationErrorStatus(err), storageBrowserWholeDiskResponse{
			OK: false, Error: dataMigrationErrorMessage(err, ui(w, "ui.fd32d3d2fecb")),
		})
		return
	}

	warnings := make([]string, 0, len(plan.Warnings))
	for _, warning := range plan.Warnings {
		warnings = append(warnings, warning.Message)
	}
	response := storageBrowserWholeDiskResponse{
		OK: true, Purpose: purpose, Device: plan.Normalized.Device, Model: plan.Normalized.Model,
		Transport: plan.Normalized.Transport, SizeBytes: plan.Normalized.SizeBytes,
		MountPoint: plan.Normalized.MountPoint, Path: plan.Normalized.Path,
		Fingerprint: plan.PlanFingerprint, Warnings: warnings,
	}
	if plan.Requirements != nil {
		response.Confirmation = plan.Requirements.ConfirmationPhrase
		response.PlayersConfirmationRequired = plan.Requirements.PlayersConfirmationRequired
		response.Online = plan.Requirements.Online
		response.Players = plan.Requirements.Players
	}
	if apply {
		response.Applied = true
		response.OperationID = operation.OperationID
	}
	writeStorageBrowserWholeDiskJSON(w, http.StatusOK, response)
}

func storageBrowserErrorStatus(err error) int {
	status := http.StatusBadGateway
	var responseErr *api.ResponseError
	if errors.As(err, &responseErr) && responseErr.StatusCode >= 400 && responseErr.StatusCode < 500 {
		status = responseErr.StatusCode
	}
	return status
}

func writeStorageBrowserWholeDiskJSON(w http.ResponseWriter, status int, result storageBrowserWholeDiskResponse) {
	if result.Warnings == nil {
		result.Warnings = []string{}
	}
	if result.Players == nil {
		result.Players = []string{}
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(result)
}

func (a *App) storageBrowserActionPlan(w http.ResponseWriter, r *http.Request) {
	a.storageBrowserAction(w, r, false)
}

func (a *App) storageBrowserActionApply(w http.ResponseWriter, r *http.Request) {
	a.storageBrowserAction(w, r, true)
}

func (a *App) storageBrowserAction(w http.ResponseWriter, r *http.Request, apply bool) {
	if !a.validCSRF(r) {
		writeStorageBrowserActionJSON(w, http.StatusForbidden, api.AdminStorageActionResponse{OK: false, Error: "invalid CSRF token"})
		return
	}
	session, _, _, ok := a.adminDiscoveryRequest(w, r)
	if !ok {
		return
	}
	client, ok := a.api.(adminStorageActionAPI)
	if !ok {
		writeStorageBrowserActionJSON(w, http.StatusServiceUnavailable, api.AdminStorageActionResponse{OK: false, Error: ui(w, "ui.36ba49211d45")})
		return
	}
	if err := r.ParseForm(); err != nil {
		writeStorageBrowserActionJSON(w, http.StatusBadRequest, api.AdminStorageActionResponse{OK: false, Error: ui(w, "ui.2cbc7c67f318")})
		return
	}
	request := api.AdminStorageActionRequest{
		Operation:    strings.TrimSpace(r.FormValue("operation")),
		Device:       strings.TrimSpace(r.FormValue("device")),
		MountPoint:   strings.TrimSpace(r.FormValue("mount_point")),
		SizeGiB:      strings.TrimSpace(r.FormValue("size_gib")),
		FreeStart:    strings.TrimSpace(r.FormValue("free_start")),
		Confirmation: r.FormValue("confirmation"),
		Fingerprint:  strings.TrimSpace(r.FormValue("fingerprint")),
	}
	if request.Operation == "" || request.Device == "" {
		writeStorageBrowserActionJSON(w, http.StatusBadRequest, api.AdminStorageActionResponse{OK: false, Error: ui(w, "ui.d1a862daecff")})
		return
	}

	var result api.AdminStorageActionResponse
	var err error
	if apply {
		result, err = client.AdminStorageActionApply(r.Context(), session, request)
	} else {
		result, err = client.AdminStorageActionPlan(r.Context(), session, request)
	}
	if err != nil {
		status := http.StatusBadGateway
		var responseErr *api.ResponseError
		if errors.As(err, &responseErr) && responseErr.StatusCode >= 400 && responseErr.StatusCode < 500 {
			status = responseErr.StatusCode
		}
		writeStorageBrowserActionJSON(w, status, api.AdminStorageActionResponse{OK: false, Error: apiMessage(err, "storage action failed")})
		return
	}
	writeStorageBrowserActionJSON(w, http.StatusOK, result)
}

func writeStorageBrowserActionJSON(w http.ResponseWriter, status int, result api.AdminStorageActionResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(result)
}

func (a *App) storageBrowserMountStatus(w http.ResponseWriter, r *http.Request) {
	session, _, _, ok := a.adminDiscoveryRequest(w, r)
	if !ok {
		return
	}
	client, ok := a.api.(adminStorageMountAPI)
	if !ok {
		writeStorageBrowserMountJSON(w, http.StatusServiceUnavailable, api.AdminStorageMountResponse{OK: false, Error: ui(w, "ui.83daf29184b3")})
		return
	}
	device := strings.TrimSpace(r.URL.Query().Get("device"))
	if device == "" {
		writeStorageBrowserMountJSON(w, http.StatusBadRequest, api.AdminStorageMountResponse{OK: false, Error: ui(w, "ui.9323924f73df")})
		return
	}
	result, err := client.AdminStorageMountStatus(r.Context(), session, device)
	if err != nil {
		status := http.StatusBadGateway
		var responseErr *api.ResponseError
		if errors.As(err, &responseErr) && responseErr.StatusCode >= 400 && responseErr.StatusCode < 500 {
			status = responseErr.StatusCode
		}
		writeStorageBrowserMountJSON(w, status, api.AdminStorageMountResponse{OK: false, Error: apiMessage(err, ui(w, "ui.e70aedb5d61d"))})
		return
	}
	writeStorageBrowserMountJSON(w, http.StatusOK, result)
}

func (a *App) storageBrowserMountPlan(w http.ResponseWriter, r *http.Request) {
	a.storageBrowserMountChange(w, r, false)
}

func (a *App) storageBrowserMountApply(w http.ResponseWriter, r *http.Request) {
	a.storageBrowserMountChange(w, r, true)
}

func (a *App) storageBrowserMountChange(w http.ResponseWriter, r *http.Request, apply bool) {
	if !a.validCSRF(r) {
		writeStorageBrowserMountJSON(w, http.StatusForbidden, api.AdminStorageMountResponse{OK: false, Error: "invalid CSRF token"})
		return
	}
	session, _, _, ok := a.adminDiscoveryRequest(w, r)
	if !ok {
		return
	}
	client, ok := a.api.(adminStorageMountAPI)
	if !ok {
		writeStorageBrowserMountJSON(w, http.StatusServiceUnavailable, api.AdminStorageMountResponse{OK: false, Error: ui(w, "ui.83daf29184b3")})
		return
	}
	if err := r.ParseForm(); err != nil {
		writeStorageBrowserMountJSON(w, http.StatusBadRequest, api.AdminStorageMountResponse{OK: false, Error: ui(w, "ui.4047305379d6")})
		return
	}
	request := api.AdminStorageMountRequest{
		Operation:   strings.TrimSpace(r.FormValue("operation")),
		Device:      strings.TrimSpace(r.FormValue("device")),
		MountPoint:  strings.TrimSpace(r.FormValue("mount_point")),
		Fingerprint: strings.TrimSpace(r.FormValue("fingerprint")),
	}
	if request.Device == "" || (request.Operation != "persist" && request.Operation != "remove") {
		writeStorageBrowserMountJSON(w, http.StatusBadRequest, api.AdminStorageMountResponse{OK: false, Error: ui(w, "ui.6fc5a9d13746")})
		return
	}

	var result api.AdminStorageMountResponse
	var err error
	if apply {
		result, err = client.AdminStorageMountApply(r.Context(), session, request)
	} else {
		result, err = client.AdminStorageMountPlan(r.Context(), session, request)
	}
	if err != nil {
		status := http.StatusBadGateway
		var responseErr *api.ResponseError
		if errors.As(err, &responseErr) && responseErr.StatusCode >= 400 && responseErr.StatusCode < 500 {
			status = responseErr.StatusCode
		}
		writeStorageBrowserMountJSON(w, status, api.AdminStorageMountResponse{OK: false, Error: apiMessage(err, ui(w, "ui.25a1ba832e04"))})
		return
	}
	writeStorageBrowserMountJSON(w, http.StatusOK, result)
}

func writeStorageBrowserMountJSON(w http.ResponseWriter, status int, result api.AdminStorageMountResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(result)
}

// Partition adoption uses the existing backup destination backend, never provisioning.
type adminStorageBackupPartitionAPI interface {
	AdminStorageMountStatus(ctx context.Context, session, device string) (api.AdminStorageMountResponse, error)
	AdminBackupStoragePlan(ctx context.Context, session string, request api.AdminBackupStorageRequest) (api.AdminBackupStorageResponse, error)
	AdminBackupStorageApply(ctx context.Context, session string, request api.AdminBackupStorageRequest) (api.AdminBackupStorageResponse, error)
}

func (a *App) storageBrowserBackupPartitionPlan(w http.ResponseWriter, r *http.Request) {
	a.storageBrowserBackupPartitionChange(w, r, false)
}

func (a *App) storageBrowserBackupPartitionApply(w http.ResponseWriter, r *http.Request) {
	a.storageBrowserBackupPartitionChange(w, r, true)
}

func (a *App) storageBrowserBackupPartitionChange(w http.ResponseWriter, r *http.Request, apply bool) {
	fail := func(status int, message string) {
		writeStorageBrowserActionJSON(w, status, api.AdminStorageActionResponse{OK: false, Error: message})
	}
	if !a.validCSRF(r) {
		fail(http.StatusForbidden, "invalid CSRF token")
		return
	}
	session, discovery, _, ok := a.adminDiscoveryRequest(w, r)
	if !ok {
		return
	}
	client, ok := a.api.(adminStorageBackupPartitionAPI)
	if !ok {
		fail(http.StatusServiceUnavailable, ui(w, "ui.6b782ed6a265"))
		return
	}
	if err := r.ParseForm(); err != nil {
		fail(http.StatusBadRequest, ui(w, "ui.bef236c6e073"))
		return
	}
	device := strings.TrimSpace(r.FormValue("device"))
	storage, err := discovery.AdminStorage(r.Context(), session)
	if err != nil {
		fail(storageBrowserErrorStatus(err), apiMessage(err, ui(w, "ui.246f08b71730")))
		return
	}
	eligible := false
	for _, candidate := range storage.Devices {
		if candidate.Path == device && candidate.Type == "part" && !candidate.System && !candidate.ReadOnly {
			eligible = true
			break
		}
	}
	if !eligible {
		fail(http.StatusBadRequest, ui(w, "ui.7000cc72f10d"))
		return
	}
	status, err := client.AdminStorageMountStatus(r.Context(), session, device)
	if err != nil {
		fail(storageBrowserErrorStatus(err), apiMessage(err, ui(w, "ui.9c607c4e7c78")))
		return
	}
	mount := status.Proposed
	if !status.OK || mount.Device != device || !mount.Mounted || mount.CurrentMountPoint == "" ||
		(mount.Persistence != "justvoxel" && mount.Persistence != "external") || mount.CurrentMountPoint != mount.MountPoint {
		fail(http.StatusBadRequest, ui(w, "ui.7f5cca698896"))
		return
	}
	request := api.AdminBackupStorageRequest{
		Type: "partition", Device: device, MountPoint: mount.CurrentMountPoint,
		Path: filepath.Join(mount.CurrentMountPoint, "backups"),
	}
	// Apply must use the reviewed mount, rather than adopting a new location silently.
	if apply && (r.FormValue("mount_point") != request.MountPoint || r.FormValue("path") != request.Path) {
		fail(http.StatusConflict, ui(w, "ui.f2d711b90422"))
		return
	}
	plan, err := client.AdminBackupStoragePlan(r.Context(), session, request)
	if err != nil {
		fail(storageBrowserErrorStatus(err), apiMessage(err, ui(w, "ui.a8e3fe8ffc80")))
		return
	}
	if !plan.OK {
		fail(http.StatusBadRequest, ui(w, "ui.a8e3fe8ffc80"))
		return
	}
	// Bind review to the backend's validated destination identity and warnings.
	evidence, err := json.Marshal(struct {
		Request    api.AdminBackupStorageRequest
		UUID       string
		Source     string
		Filesystem string
		Warnings   []string
	}{request, plan.Proposed.ExpectedUUID, plan.Proposed.ExpectedSource, plan.Proposed.Filesystem, plan.Warnings})
	if err != nil {
		fail(http.StatusInternalServerError, ui(w, "ui.a2882cb4fca1"))
		return
	}
	fingerprint := fmt.Sprintf("sha256:%x", sha256.Sum256(evidence))
	if apply {
		if r.FormValue("fingerprint") != fingerprint {
			fail(http.StatusConflict, ui(w, "ui.c1d04601394c"))
			return
		}
		result, err := client.AdminBackupStorageApply(r.Context(), session, request)
		if err != nil {
			fail(storageBrowserErrorStatus(err), apiMessage(err, ui(w, "ui.4514d5e2d5d0")))
			return
		}
		if !result.OK || !result.Applied {
			fail(http.StatusConflict, ui(w, "ui.411d2e160936"))
			return
		}
	}
	warnings := plan.Warnings
	if warnings == nil {
		warnings = []string{}
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(struct {
		OK          bool                         `json:"ok"`
		Proposed    api.AdminBackupStorageTarget `json:"proposed"`
		Warnings    []string                     `json:"warnings"`
		Fingerprint string                       `json:"fingerprint"`
		Applied     bool                         `json:"applied"`
	}{true, plan.Proposed, warnings, fingerprint, apply})
}
