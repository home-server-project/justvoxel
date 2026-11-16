package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/home-server-project/justvoxel-webui/internal/api"
)

type storageMinecraftMigrationAPI interface {
	dataMigrationOrchestrationAPI
	Session(ctx context.Context, session string) (api.SessionInfo, error)
}

type storageMinecraftMigrationResponse struct {
	OK        bool                                `json:"ok"`
	Error     string                              `json:"error,omitempty"`
	Plan      *api.AdminDataMigrationPlanResponse `json:"plan,omitempty"`
	Operation *api.PersistentOperation            `json:"operation,omitempty"`
}

func (a *App) storageMinecraftMigrationIdentity(w http.ResponseWriter, r *http.Request) (string, storageMinecraftMigrationAPI, bool) {
	if mustChange(r) {
		writeStorageMinecraftMigrationJSON(w, http.StatusForbidden, storageMinecraftMigrationResponse{OK: false, Error: ui(w, "ui.fe5ad66e02ea")})
		return "", nil, false
	}
	session, ok := sessionFromRequest(r)
	if !ok {
		writeStorageMinecraftMigrationJSON(w, http.StatusUnauthorized, storageMinecraftMigrationResponse{OK: false, Error: ui(w, "ui.45cceda8377a")})
		return "", nil, false
	}
	client, ok := a.api.(storageMinecraftMigrationAPI)
	if !ok {
		writeStorageMinecraftMigrationJSON(w, http.StatusServiceUnavailable, storageMinecraftMigrationResponse{OK: false, Error: ui(w, "ui.11d8b513d533")})
		return "", nil, false
	}
	identity, err := client.Session(r.Context(), session)
	if err != nil {
		a.writeStorageMinecraftMigrationError(w, err, ui(w, "ui.11d8b513d533"))
		return "", nil, false
	}
	if identity.Role != "administrator" {
		writeStorageMinecraftMigrationJSON(w, http.StatusForbidden, storageMinecraftMigrationResponse{OK: false, Error: ui(w, "ui.2d3d0b30bde9")})
		return "", nil, false
	}
	return session, client, true
}

func (a *App) storageMinecraftMigrationCurrent(w http.ResponseWriter, r *http.Request) {
	session, client, ok := a.storageMinecraftMigrationIdentity(w, r)
	if !ok {
		return
	}
	operation, err := dataMigrationCurrentOperation(r.Context(), client, session)
	if err != nil {
		a.writeStorageMinecraftMigrationError(w, err, ui(w, "ui.35e0f0ae3c44"))
		return
	}
	writeStorageMinecraftMigrationJSON(w, http.StatusOK, storageMinecraftMigrationResponse{OK: true, Operation: operation})
}

func (a *App) storageMinecraftMigrationPlan(w http.ResponseWriter, r *http.Request) {
	if !a.validCSRF(r) {
		writeStorageMinecraftMigrationJSON(w, http.StatusForbidden, storageMinecraftMigrationResponse{OK: false, Error: ui(w, "ui.74bccc44f472")})
		return
	}
	session, client, ok := a.storageMinecraftMigrationIdentity(w, r)
	if !ok {
		return
	}
	if current, err := dataMigrationCurrentOperation(r.Context(), client, session); err != nil {
		a.writeStorageMinecraftMigrationError(w, err, ui(w, "ui.35e0f0ae3c44"))
		return
	} else if current != nil {
		writeStorageMinecraftMigrationJSON(w, http.StatusConflict, storageMinecraftMigrationResponse{
			OK: false, Error: ui(w, "ui.fd4bc8218042"), Operation: current,
		})
		return
	}

	request, err := storageMinecraftMigrationRequest(r)
	if err != nil {
		writeStorageMinecraftMigrationJSON(w, http.StatusBadRequest, storageMinecraftMigrationResponse{OK: false, Error: err.Error()})
		return
	}
	plan, err := dataMigrationPlanReviewed(r.Context(), client, session, request)
	if err != nil {
		writeStorageMinecraftMigrationJSON(w, dataMigrationErrorStatus(err), storageMinecraftMigrationResponse{
			OK: false, Error: dataMigrationErrorMessage(err, ui(w, "ui.e9179b85561f")), Plan: &plan,
		})
		return
	}
	writeStorageMinecraftMigrationJSON(w, http.StatusOK, storageMinecraftMigrationResponse{OK: true, Plan: &plan})
}

func (a *App) storageMinecraftMigrationApply(w http.ResponseWriter, r *http.Request) {
	if !a.validCSRF(r) {
		writeStorageMinecraftMigrationJSON(w, http.StatusForbidden, storageMinecraftMigrationResponse{OK: false, Error: ui(w, "ui.74bccc44f472")})
		return
	}
	session, client, ok := a.storageMinecraftMigrationIdentity(w, r)
	if !ok {
		return
	}
	request, err := storageMinecraftMigrationRequest(r)
	if err != nil {
		writeStorageMinecraftMigrationJSON(w, http.StatusBadRequest, storageMinecraftMigrationResponse{OK: false, Error: err.Error()})
		return
	}

	plan, operation, err := dataMigrationApplyReviewed(r.Context(), client, session, request, dataMigrationApplyProof{
		PlanFingerprint:    strings.TrimSpace(r.FormValue("plan_fingerprint")),
		MigrationConfirmed: strings.TrimSpace(r.FormValue("migration_confirmation")) == "MIGRATE",
		Confirmation:       strings.TrimSpace(r.FormValue("destructive_confirmation")),
		PlayersConfirmed:   r.FormValue("players_confirmed") == "yes",
	})
	if err != nil {
		writeStorageMinecraftMigrationJSON(w, dataMigrationErrorStatus(err), storageMinecraftMigrationResponse{
			OK: false, Error: dataMigrationErrorMessage(err, ui(w, "ui.3bc4fd9254b3")), Plan: &plan,
		})
		return
	}
	writeStorageMinecraftMigrationJSON(w, http.StatusOK, storageMinecraftMigrationResponse{OK: true, Operation: operation})
}

func (a *App) storageMinecraftMigrationProgress(w http.ResponseWriter, r *http.Request) {
	session, client, ok := a.storageMinecraftMigrationIdentity(w, r)
	if !ok {
		return
	}
	operation, err := dataMigrationOperation(r.Context(), client, session, r.PathValue("id"))
	if err != nil {
		a.writeStorageMinecraftMigrationError(w, err, ui(w, "ui.35e0f0ae3c44"))
		return
	}
	writeStorageMinecraftMigrationJSON(w, http.StatusOK, storageMinecraftMigrationResponse{OK: true, Operation: operation})
}

func storageMinecraftMigrationRequest(r *http.Request) (api.AdminDataMigrationPlanRequest, error) {
	if err := r.ParseForm(); err != nil {
		return api.AdminDataMigrationPlanRequest{}, errors.New(uiText("ui.6c892561219a"))
	}
	request := api.AdminDataMigrationPlanRequest{
		Operation:  strings.TrimSpace(r.FormValue("operation")),
		Device:     strings.TrimSpace(r.FormValue("device")),
		MountPoint: strings.TrimSpace(r.FormValue("mount_point")),
		Path:       strings.TrimSpace(r.FormValue("path")),
		SizeGiB:    strings.TrimSpace(r.FormValue("size_gib")),
	}
	if request.SizeGiB == "" {
		request.SizeGiB = "all"
	}
	if request.Operation != "use_partition" {
		return request, errors.New(uiText("ui.404b7cf5c5c3"))
	}
	if !strings.HasPrefix(request.Device, "/dev/") || !strings.HasPrefix(request.MountPoint, "/") || !strings.HasPrefix(request.Path, "/") {
		return request, errors.New(uiText("ui.01237fb5c94f"))
	}
	return request, nil
}

func (a *App) writeStorageMinecraftMigrationError(w http.ResponseWriter, err error, fallback string) {
	status := storageBrowserErrorStatus(err)
	message := apiMessage(err, fallback)
	if errors.Is(err, api.ErrUnauthorized) {
		status = http.StatusUnauthorized
		message = ui(w, "ui.45cceda8377a")
	} else if errors.Is(err, api.ErrPasswordChangeRequired) {
		status = http.StatusForbidden
		message = ui(w, "ui.fe5ad66e02ea")
	}
	writeStorageMinecraftMigrationJSON(w, status, storageMinecraftMigrationResponse{OK: false, Error: message})
}

func writeStorageMinecraftMigrationJSON(w http.ResponseWriter, status int, result storageMinecraftMigrationResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(result)
}
