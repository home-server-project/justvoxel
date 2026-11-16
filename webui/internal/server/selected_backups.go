package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/home-server-project/justvoxel-webui/internal/api"
)

type selectedBackupsAPI interface {
	MinecraftInstanceBackups(context.Context, string, string, string, any) (json.RawMessage, error)
}

// The physical backup destination belongs to the appliance, independently of
// the selected server's backup execution, schedule, and restore operations.
type selectedBackupDestinationAPI interface {
	storageDestinationAPI
	AdminBackupStoragePlan(context.Context, string, api.AdminBackupStorageRequest) (api.AdminBackupStorageResponse, error)
	AdminBackupStorageApply(context.Context, string, api.AdminBackupStorageRequest) (api.AdminBackupStorageResponse, error)
}

type selectedBackupMetadata struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
	Created string `json:"created"`
	Archive string `json:"archive"`
	Size    int64  `json:"size"`
}
type selectedRestorePlan struct {
	PlanID                      string                 `json:"plan_id"`
	Metadata                    selectedBackupMetadata `json:"metadata"`
	CurrentVersion              string                 `json:"current_version"`
	Integrity                   string                 `json:"integrity"`
	RuntimeState                string                 `json:"runtime_state"`
	Players                     api.InstancePlayers    `json:"players"`
	PlayersConfirmationRequired bool                   `json:"players_confirmation_required"`
}

func (a *App) selectedBackupsWorkspace(w http.ResponseWriter, r *http.Request) bool {
	session, _, ok := a.minecraftWorkspaceIdentity(w, r, "administrator")
	if !ok {
		return true
	}
	if r.Method != http.MethodGet && !a.validCSRF(r) {
		writeMinecraftWorkspaceError(w, 403, ui(w, "ui.74bccc44f472"))
		return true
	}
	selected, ok := a.selectedMinecraft(w, r, session)
	if !ok {
		return true
	}
	client, ok := a.api.(selectedBackupsAPI)
	if !ok {
		writeMinecraftWorkspaceError(w, 503, ui(w, "ui.b12b0e686878"))
		return true
	}
	identityClient, ok := a.api.(sessionIdentityAPI)
	if !ok {
		writeMinecraftWorkspaceError(w, 503, ui(w, "ui.c5306e266aa5"))
		return true
	}
	identity, err := identityClient.Session(r.Context(), session)
	if err != nil {
		a.writeMinecraftWorkspaceAPIError(w, err, ui(w, "ui.c5306e266aa5"))
		return true
	}
	data := backupsWorkspaceData{CSRF: csrfFromRequest(r), Identity: identity, SelectedServer: selected.ID, SelectedName: selected.Name, RestoreScope: ui(w, "ui.984fe20477cb"), RestoreModeTitle: ui(w, "ui.c797d4d8982e")}
	destinationClient, hasDestination := a.api.(selectedBackupDestinationAPI)
	if hasDestination {
		destination, err := a.storageDestinationData(r, session, identity, data.CSRF, nil, nil)
		if err != nil {
			data.Error = apiMessage(err, ui(w, "ui.9b481e923a7c"))
		} else {
			data = destination
			data.SelectedServer, data.SelectedName = selected.ID, selected.Name
			data.RestoreScope = ui(w, "ui.984fe20477cb")
			data.RestoreModeTitle = ui(w, "ui.c797d4d8982e")
		}
		data.DestinationWorkspace = "/workspace/backups"
	}
	if r.URL.Query().Get("result") == "destination" {
		data.Message = ui(w, "ui.d972c62939f4")
	}
	call := func(action string, request any, target any) bool {
		raw, e := client.MinecraftInstanceBackups(r.Context(), session, selected.ID, action, request)
		if e != nil {
			data.Error = apiMessage(e, ui(w, "ui.4ec620f98e1a"))
			return false
		}
		if target != nil && json.Unmarshal(raw, target) != nil {
			data.Error = ui(w, "ui.26e5f7a588d1")
			return false
		}
		return true
	}
	path := r.URL.Path
	if r.Method == http.MethodPost {
		switch path {
		case "/workspace/backups/destination/plan", "/workspace/backups/destination/apply":
			if !hasDestination {
				data.Error = ui(w, "ui.b82d33f56e34")
				break
			}
			apply := strings.HasSuffix(path, "/apply")
			request, err := parseBackupsWorkspaceStorageForm(r, apply)
			data.DestinationForm = request
			if err != nil {
				data.Error = err.Error()
				break
			}
			var result api.AdminBackupStorageResponse
			if apply {
				result, err = destinationClient.AdminBackupStorageApply(r.Context(), session, request)
			} else {
				result, err = destinationClient.AdminBackupStoragePlan(r.Context(), session, request)
			}
			if err != nil {
				data.Error = apiMessage(err, ui(w, "ui.241f98a9f9e6"))
				break
			}
			if apply && result.Applied {
				target := "/workspace/backups?result=destination"
				if r.URL.Query().Has("server") {
					target += "&server=" + selected.ID
				}
				http.Redirect(w, r, target, http.StatusSeeOther)
				return true
			}
			data.DestinationPlan = &result
			data.ProposedDestinationAvailable = backupsWorkspaceFormatOptionalBytes(result.Proposed.AvailableBytes)
			data.ProposedDestinationSize = backupsWorkspaceFormatOptionalBytes(result.Proposed.FilesystemBytes)
			if apply {
				data.Error = ui(w, "ui.08c30ba25266")
			}

		case "/workspace/backups/backup":
			if call("now", struct{}{}, nil) {
				data.Message = ui(w, "ui.0cd9df837d54")
			}
		case "/workspace/backups/automatic/plan", "/workspace/backups/automatic/apply":
			form, e := parseBackupsWorkspaceAutomaticForm(r)
			if e != nil {
				data.Error = e.Error()
				break
			}
			data.Automatic = form
			if strings.HasSuffix(path, "/apply") {
				if call("configure", api.InstanceBackupSettings{Enabled: form.Enabled, Time: form.DailyTime, Retention: form.Keep}, nil) {
					data.Message = ui(w, "ui.093c109f2a79")
				}
			} else {
				data.AutomaticPlan = &api.AdminConfigurationChangeResponse{OK: true, Changes: []api.AdminConfigurationChange{{Field: "backup"}}}
			}
		case "/workspace/backups/restore/plan":
			var plan selectedRestorePlan
			archive := r.FormValue("backup_id")
			if call("restore/plan", map[string]any{"archive": archive}, &plan) {
				if plan.Metadata.ID != selected.ID || plan.Metadata.Archive != archive || plan.Integrity != "verified" {
					data.Error = ui(w, "ui.5937bf51d420")
					break
				}
				data.RestoreRequest = backupsRestoreRequest{BackupID: archive, Mode: "full"}
				data.RestorePlan = &backupsRestorePlan{OK: true, PlanFingerprint: plan.PlanID, Normalized: &backupsRestoreNormalized{BackupID: archive, CreatedAt: plan.Metadata.Created, Current: backupsRestoreCurrent{Version: plan.CurrentVersion}, VersionRelation: ui(w, "ui.2facc6b9b206")}, Requirements: &backupsRestoreRequirements{DestructiveConfirmationRequired: true, PlayersConfirmationRequired: plan.PlayersConfirmationRequired, Online: plan.Players.Online}}
			}
		case "/workspace/backups/restore/apply":
			if r.FormValue("confirmation") != "RESTORE" {
				data.Error = ui(w, "ui.983972bd982d")
				break
			}
			if call("restore/apply", map[string]any{"archive": r.FormValue("backup_id"), "plan_id": r.FormValue("plan_fingerprint"), "confirm_destructive": true, "confirm_players": r.FormValue("players_confirmed") == "yes"}, nil) {
				data.Message = ui(w, "ui.4f73405f1c2f")
				progress := presentSelectedRestore(selectedRestoreProgress{ID: selected.ID, State: "validating"})
				data.SelectedRestoreProgress = &progress
			}
		case "/workspace/backups/restore/recover":
			if r.FormValue("confirm_players") != "yes" {
				data.Error = ui(w, "ui.2685eb20bcaf")
				break
			}
			if call("restore/recover", map[string]any{"confirm_players": true}, nil) {
				data.Message = ui(w, "ui.52e5331f961e")
				progress := presentSelectedRestore(selectedRestoreProgress{ID: selected.ID, State: "recovering"})
				data.SelectedRestoreProgress = &progress
			}
		default:
			data.Error = ui(w, "ui.b82d33f56e34")
		}
	}
	var settings api.InstanceBackupSettings
	if call("settings", nil, &settings) && data.AutomaticPlan == nil {
		data.Automatic = backupsWorkspaceAutomaticForm{Enabled: settings.Enabled, DailyTime: settings.Time, Keep: settings.Retention}
	}
	var inventory struct {
		ID      string                   `json:"id"`
		Backups []selectedBackupMetadata `json:"backups"`
	}
	if call("inventory", nil, &inventory) {
		if inventory.ID != selected.ID {
			data.Error = ui(w, "ui.133b3cbf299e")
		} else {
			var total uint64
			for _, item := range inventory.Backups {
				if item.ID != selected.ID || item.Size < 0 {
					data.Error = ui(w, "ui.133b3cbf299e")
					data.Backups = nil
					break
				}
				size := uint64(item.Size)
				total += size
				data.Backups = append(data.Backups, backupsWorkspaceBackupView{ID: item.Archive, DeleteRef: selected.ID + "/" + item.Archive, DisplayName: item.Archive, SizeBytes: size, Size: humanBytes(size), CreatedLabel: item.Created, Version: item.Version})
			}
			data.BackupCount = len(data.Backups)
			data.TotalSize = humanBytes(total)
		}
	}
	var status struct {
		State   string `json:"state"`
		Message string `json:"message"`
	}
	if call("status", nil, &status) {
		data.SelectedBackupStatus = status.State
	}
	if progress, err := readSelectedRestore(r.Context(), client, session, selected.ID); err != nil {
		data.Error = ui(w, "ui.afadfb0b5986")
	} else {
		data.SelectedRestoreStatus = progress.StateLabel
		data.SelectedRestoreRecovery = progress.State != "idle" && progress.RecoveryRequired
		if progress.State != "idle" {
			data.SelectedRestoreProgress = &progress
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if err := a.executeTemplate(w, "backups_workspace.html", data); err != nil {
		http.Error(w, fmt.Sprint(ui(w, "ui.f6010c89978c")), 500)
	}
	return true
}
