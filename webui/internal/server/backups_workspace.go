package server

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/home-server-project/justvoxel-webui/internal/api"
)

type backupsWorkspaceAPI interface {
	Session(ctx context.Context, session string) (api.SessionInfo, error)
}

type backupsWorkspaceAutomaticForm struct {
	Enabled   bool
	DailyTime string
	Keep      int
}

type backupsWorkspaceBackupView struct {
	ID             string
	DeleteRef      string
	DisplayName    string
	SizeBytes      uint64
	CreatedLabel   string
	Size           string
	MetadataStatus string
	Version        string
	Bedrock        bool
	Variant        string
}

type backupsWorkspacePartitionView struct {
	Path       string
	Size       string
	Filesystem string
	Mountpoint string
	Model      string
}

type backupsWorkspaceData struct {
	SelectedServer               string
	SelectedName                 string
	SelectedBackupStatus         string
	SelectedRestoreStatus        string
	SelectedRestoreRecovery      bool
	SelectedRestoreProgress      *selectedRestoreProgress
	DestinationWorkspace         string
	CSRF                         string
	Identity                     api.SessionInfo
	Backups                      []backupsWorkspaceBackupView
	BackupCount                  int
	TotalSize                    string
	Message                      string
	Error                        string
	Automatic                    backupsWorkspaceAutomaticForm
	AutomaticPlan                *api.AdminConfigurationChangeResponse
	Timezone                     string
	Destination                  api.AdminBackupStorageResponse
	DestinationForm              api.AdminBackupStorageRequest
	DestinationPlan              *api.AdminBackupStorageResponse
	DestinationPartitions        []backupsWorkspacePartitionView
	DestinationAvailable         string
	DestinationFilesystemSize    string
	ProposedDestinationAvailable string
	ProposedDestinationSize      string
	RestoreRequest               backupsRestoreRequest
	RestorePlan                  *backupsRestorePlan
	RestoreSize                  string
	RestoreModeTitle             string
	RestoreScope                 string
	RestoreError                 string
}

func (a *App) registerBackupsWorkspaceRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /workspace/backups/restore/recover", func(w http.ResponseWriter, r *http.Request) {
		if !a.selectedBackupsWorkspace(w, r) {
			http.NotFound(w, r)
		}
	})
	mux.HandleFunc("GET /workspace/backups/restore/status", a.selectedRestoreStatus)
	mux.HandleFunc("GET /workspace/backups", a.backupsWorkspacePage)
	mux.HandleFunc("POST /workspace/backups/backup", a.backupsWorkspaceBackupNow)
	mux.HandleFunc("POST /workspace/backups/automatic/plan", a.backupsWorkspaceAutomaticPlan)
	mux.HandleFunc("POST /workspace/backups/automatic/apply", a.backupsWorkspaceAutomaticApply)
	mux.HandleFunc("POST /workspace/backups/destination/plan", a.backupsWorkspaceDestinationPlan)
	mux.HandleFunc("POST /workspace/backups/destination/apply", a.backupsWorkspaceDestinationApply)
	mux.HandleFunc("POST /workspace/backups/restore/plan", a.backupsWorkspaceRestorePlan)
	mux.HandleFunc("POST /workspace/backups/restore/apply", a.backupsWorkspaceRestoreApply)
}

func (a *App) backupsWorkspaceRequest(w http.ResponseWriter, r *http.Request, requireCSRF bool) (string, backupsWorkspaceAPI, api.SessionInfo, bool) {
	if mustChange(r) {
		http.Error(w, ui(w, "ui.fe5ad66e02ea"), http.StatusForbidden)
		return "", nil, api.SessionInfo{}, false
	}
	if requireCSRF && !a.validCSRF(r) {
		http.Error(w, "invalid CSRF token", http.StatusForbidden)
		return "", nil, api.SessionInfo{}, false
	}
	session, ok := sessionFromRequest(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return "", nil, api.SessionInfo{}, false
	}
	client, ok := a.api.(backupsWorkspaceAPI)
	if !ok {
		http.Error(w, ui(w, "ui.e151226cccf3"), http.StatusServiceUnavailable)
		return "", nil, api.SessionInfo{}, false
	}
	identity, err := client.Session(r.Context(), session)
	if err != nil {
		a.handleBackupsWorkspaceError(w, r, err)
		return "", nil, api.SessionInfo{}, false
	}
	if !isAdministratorLevel(identity.Role) {
		http.Error(w, ui(w, "ui.031cc7a3ec82"), http.StatusForbidden)
		return "", nil, api.SessionInfo{}, false
	}
	return session, client, identity, true
}

func (a *App) backupsWorkspacePage(w http.ResponseWriter, r *http.Request) {
	a.selectedBackupsWorkspace(w, r)
}

func (a *App) backupsWorkspaceBackupNow(w http.ResponseWriter, r *http.Request) {
	if !a.selectedBackupsWorkspace(w, r) {
		http.NotFound(w, r)
	}
}

func (a *App) backupsWorkspaceAutomaticPlan(w http.ResponseWriter, r *http.Request) {
	a.selectedBackupsWorkspace(w, r)
}

func (a *App) backupsWorkspaceAutomaticApply(w http.ResponseWriter, r *http.Request) {
	a.selectedBackupsWorkspace(w, r)
}

func (a *App) backupsWorkspaceDestinationPlan(w http.ResponseWriter, r *http.Request) {
	a.selectedBackupsWorkspace(w, r)
}

func (a *App) backupsWorkspaceDestinationApply(w http.ResponseWriter, r *http.Request) {
	a.selectedBackupsWorkspace(w, r)
}

func (a *App) backupsWorkspaceRestorePlan(w http.ResponseWriter, r *http.Request) {
	a.selectedBackupsWorkspace(w, r)
}

func (a *App) backupsWorkspaceRestoreApply(w http.ResponseWriter, r *http.Request) {
	a.selectedBackupsWorkspace(w, r)
}

func parseBackupsWorkspaceAutomaticForm(r *http.Request) (backupsWorkspaceAutomaticForm, error) {
	var form backupsWorkspaceAutomaticForm
	if err := r.ParseForm(); err != nil {
		return form, errors.New(uiText("ui.b76081cac01a"))
	}
	form.Enabled = r.FormValue("automatic_enabled") == "on"
	form.DailyTime = strings.TrimSpace(r.FormValue("daily_time"))
	form.Keep, _ = strconv.Atoi(strings.TrimSpace(r.FormValue("backup_keep")))
	if form.Keep <= 0 {
		return form, errors.New(uiText("ui.4060f8d382a9"))
	}
	if _, err := time.Parse("15:04", form.DailyTime); err != nil {
		return form, errors.New(uiText("ui.eddcab5fe7fa"))
	}
	return form, nil
}

func parseBackupsWorkspaceStorageForm(r *http.Request, includePassword bool) (api.AdminBackupStorageRequest, error) {
	var out api.AdminBackupStorageRequest
	if err := r.ParseForm(); err != nil {
		return out, errors.New(uiText("ui.215a8205bfef"))
	}
	out.Type = strings.TrimSpace(r.FormValue("type"))
	out.Path = strings.TrimSpace(r.FormValue("path"))
	out.Device = strings.TrimSpace(r.FormValue("device"))
	out.MountPoint = strings.TrimSpace(r.FormValue("mount_point"))
	out.Source = strings.TrimSpace(r.FormValue("source"))
	out.Username = strings.TrimSpace(r.FormValue("username"))
	out.Domain = strings.TrimSpace(r.FormValue("domain"))
	if includePassword {
		out.Password = r.FormValue("password")
	}
	if out.Type == "" || out.Path == "" {
		return out, errors.New(uiText("ui.a5dd0537770c"))
	}
	return out, nil
}

func backupsWorkspaceStorageFormFromCurrent(current api.AdminBackupStorageTarget) api.AdminBackupStorageRequest {
	targetType := current.Type
	switch targetType {
	case "system", "partition", "nfs", "smb":
	default:
		targetType = "system"
	}
	request := api.AdminBackupStorageRequest{
		Type: targetType, Path: current.Path, MountPoint: current.MountPoint, Source: current.ExpectedSource,
	}
	if targetType == "partition" && strings.HasPrefix(current.ExpectedSource, "/dev/") {
		request.Device = current.ExpectedSource
	}
	return request
}

func backupsWorkspaceSafePartitions(storage api.AdminStorageDiscovery) []backupsWorkspacePartitionView {
	out := make([]backupsWorkspacePartitionView, 0)
	for _, device := range storage.Devices {
		if device.Type != "part" || device.ReadOnly {
			continue
		}
		switch device.Filesystem {
		case "xfs", "ext4", "btrfs":
		default:
			continue
		}
		critical := false
		for _, mountpoint := range device.Mountpoints {
			switch mountpoint {
			case "/", "/boot", "/boot/efi", "/var":
				critical = true
			}
		}
		if critical {
			continue
		}
		mountpoint := ""
		if len(device.Mountpoints) > 0 {
			mountpoint = device.Mountpoints[0]
		}
		out = append(out, backupsWorkspacePartitionView{
			Path: device.Path, Size: humanBytes(device.SizeBytes), Filesystem: device.Filesystem,
			Mountpoint: mountpoint, Model: device.Model,
		})
	}
	return out
}

func backupsWorkspaceFormatOptionalBytes(value uint64) string {
	if value == 0 {
		return uiText("ui.b764cdc0eab7")
	}
	return humanBytes(value)
}

func (a *App) handleBackupsWorkspaceError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, api.ErrUnauthorized) {
		a.clearSessionCookies(w)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if errors.Is(err, api.ErrPasswordChangeRequired) {
		http.Error(w, ui(w, "ui.fe5ad66e02ea"), http.StatusForbidden)
		return
	}
	var responseErr *api.ResponseError
	if errors.As(err, &responseErr) && responseErr.StatusCode == http.StatusForbidden {
		http.Error(w, ui(w, "ui.031cc7a3ec82"), http.StatusForbidden)
		return
	}
	http.Error(w, ui(w, "ui.9829d799550b"), http.StatusBadGateway)
}
