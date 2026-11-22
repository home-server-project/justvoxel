package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/home-server-project/justvoxel-webui/internal/api"
)

type adminMigrationWorkspaceExportAPI interface {
	adminMigrationWorkspaceAPI
	AdminMigrationExportPlan(ctx context.Context, session string, request api.AdminMigrationExportTargetRequest) (api.AdminMigrationExportPlanResponse, error)
	AdminMigrationExportApply(ctx context.Context, session string, request api.AdminMigrationExportApplyRequest) (api.AdminMigrationApplyResponse, error)
}

type migrationWorkspaceExportDeviceView struct {
	Device api.AdminMigrationExportDevice
	Size   string
}

type migrationWorkspaceExportPageData struct {
	ServerID         string
	Title            string
	Version          string
	ManagementAPI    string
	CSRF             string
	Identity         api.SessionInfo
	Discovery        api.AdminMigrationExportDiscoveryResponse
	LocalAvailable   bool
	BackupAvailable  bool
	DeviceAvailable  bool
	NFSAvailable     bool
	SMBAvailable     bool
	AttachedDevices  []migrationWorkspaceExportDeviceView
	RemovableDevices []migrationWorkspaceExportDeviceView
	Error            string
}

type migrationWorkspaceExportReviewPageData struct {
	ServerID      string
	Title         string
	Version       string
	ManagementAPI string
	CSRF          string
	Identity      api.SessionInfo
	Request       api.AdminMigrationExportTargetRequest
	Plan          api.AdminMigrationExportPlanResponse
	DataSize      string
	AvailableSize string
	Error         string
}

func (a *App) registerAdminMigrationWorkspaceExportPages(mux *http.ServeMux) {
	mux.HandleFunc("GET /workspace/migration/export", a.migrationWorkspaceExportPage)
	mux.HandleFunc("POST /workspace/migration/export/review", a.migrationWorkspaceExportReview)
	mux.HandleFunc("POST /workspace/migration/export/apply", a.migrationWorkspaceExportApply)
	mux.HandleFunc("GET /api/workspace/migration/progress/{id}", a.migrationWorkspaceProgressStatus)
}

func (a *App) migrationWorkspaceExportRequest(w http.ResponseWriter, r *http.Request, requireCSRF bool) (string, adminMigrationWorkspaceExportAPI, api.SessionInfo, bool) {
	if requireCSRF && !a.validCSRF(r) {
		http.Error(w, "invalid CSRF token", http.StatusForbidden)
		return "", nil, api.SessionInfo{}, false
	}
	session, _, identity, ok := a.migrationWorkspaceRequest(w, r)
	if !ok {
		return "", nil, api.SessionInfo{}, false
	}
	client, ok := a.api.(adminMigrationWorkspaceExportAPI)
	if !ok {
		http.Error(w, ui(w, "ui.5b372da0b3ba"), http.StatusServiceUnavailable)
		return "", nil, api.SessionInfo{}, false
	}
	return session, client, identity, true
}

func (a *App) migrationWorkspaceExportPage(w http.ResponseWriter, r *http.Request) {
	session, client, identity, ok := a.migrationWorkspaceExportRequest(w, r, false)
	if !ok {
		return
	}
	if migrationWorkspaceRedirectBrowser(w, r, "/?workspace=migration&tab=export") {
		return
	}
	if a.redirectCurrentMigrationWorkspaceOperation(w, r, session, client) {
		return
	}
	discovery, err := client.AdminMigrationExportDiscovery(r.Context(), session)
	if err != nil {
		a.handleMigrationWorkspaceRequestError(w, r, err, ui(w, "ui.9b0f8d42e43d"))
		return
	}
	a.renderMigrationWorkspaceExportPage(w, http.StatusOK, identity, discovery, csrfFromRequest(r), "", migrationWorkspaceSelectedID(r))
}

func (a *App) migrationWorkspaceExportReview(w http.ResponseWriter, r *http.Request) {
	session, client, identity, ok := a.migrationWorkspaceExportRequest(w, r, true)
	if !ok {
		return
	}
	if a.redirectCurrentMigrationWorkspaceOperation(w, r, session, client) {
		return
	}
	discovery, err := client.AdminMigrationExportDiscovery(r.Context(), session)
	if err != nil {
		a.handleMigrationWorkspaceRequestError(w, r, err, ui(w, "ui.9b0f8d42e43d"))
		return
	}
	request, err := parseMigrationWorkspaceExportForm(r, discovery)
	if err != nil {
		a.renderMigrationWorkspaceExportPage(w, http.StatusBadRequest, identity, discovery, csrfFromRequest(r), err.Error(), migrationWorkspaceSelectedID(r))
		return
	}
	plan, err := client.AdminMigrationExportPlan(r.Context(), session, request)
	if err != nil {
		var responseErr *api.ResponseError
		if errors.As(err, &responseErr) && responseErr.StatusCode == http.StatusBadRequest && plan.SchemaVersion != "" {
			a.renderMigrationWorkspaceExportReview(w, http.StatusBadRequest, identity, request, plan, csrfFromRequest(r), apiMessage(err, ui(w, "ui.99606fd06234")), migrationWorkspaceSelectedID(r))
			return
		}
		a.handleMigrationWorkspaceRequestError(w, r, err, ui(w, "ui.f1b7caca8e5d"))
		return
	}
	a.renderMigrationWorkspaceExportReview(w, http.StatusOK, identity, request, plan, csrfFromRequest(r), "", migrationWorkspaceSelectedID(r))
}

func (a *App) migrationWorkspaceExportApply(w http.ResponseWriter, r *http.Request) {
	session, client, identity, ok := a.migrationWorkspaceExportRequest(w, r, true)
	if !ok {
		return
	}
	if a.redirectCurrentMigrationWorkspaceOperation(w, r, session, client) {
		return
	}
	discovery, err := client.AdminMigrationExportDiscovery(r.Context(), session)
	if err != nil {
		a.handleMigrationWorkspaceRequestError(w, r, err, ui(w, "ui.9b0f8d42e43d"))
		return
	}
	request, err := parseMigrationWorkspaceExportForm(r, discovery)
	if err != nil {
		a.renderMigrationWorkspaceExportPage(w, http.StatusBadRequest, identity, discovery, csrfFromRequest(r), err.Error(), migrationWorkspaceSelectedID(r))
		return
	}
	plan, err := client.AdminMigrationExportPlan(r.Context(), session, request)
	if err != nil {
		var responseErr *api.ResponseError
		if errors.As(err, &responseErr) && responseErr.StatusCode == http.StatusBadRequest && plan.SchemaVersion != "" {
			a.renderMigrationWorkspaceExportReview(w, http.StatusBadRequest, identity, request, plan, csrfFromRequest(r), apiMessage(err, ui(w, "ui.99606fd06234")), migrationWorkspaceSelectedID(r))
			return
		}
		a.handleMigrationWorkspaceRequestError(w, r, err, ui(w, "ui.f1b7caca8e5d"))
		return
	}

	submittedFingerprint := strings.TrimSpace(r.FormValue("plan_fingerprint"))
	if submittedFingerprint == "" || submittedFingerprint != plan.PlanFingerprint {
		a.renderMigrationWorkspaceExportReview(w, http.StatusConflict, identity, request, plan, csrfFromRequest(r), ui(w, "ui.301807232647"), migrationWorkspaceSelectedID(r))
		return
	}
	if strings.TrimSpace(r.FormValue("export_confirmation")) != "EXPORT" {
		a.renderMigrationWorkspaceExportReview(w, http.StatusBadRequest, identity, request, plan, csrfFromRequest(r), ui(w, "ui.fc008692fded"), migrationWorkspaceSelectedID(r))
		return
	}
	playersConfirmed := r.FormValue("players_confirmed") == "yes"
	if plan.Requirements != nil && plan.Requirements.PlayersConfirmationRequired && !playersConfirmed {
		a.renderMigrationWorkspaceExportReview(w, http.StatusBadRequest, identity, request, plan, csrfFromRequest(r), ui(w, "ui.47e685e28464"), migrationWorkspaceSelectedID(r))
		return
	}

	smbPassword := ""
	if plan.Requirements != nil && plan.Requirements.SMBPasswordRequired {
		smbPassword = r.FormValue("smb_password")
		if smbPassword == "" {
			a.renderMigrationWorkspaceExportReview(w, http.StatusBadRequest, identity, request, plan, csrfFromRequest(r), ui(w, "ui.219faeffd07d"), migrationWorkspaceSelectedID(r))
			return
		}
	}

	result, err := client.AdminMigrationExportApply(r.Context(), session, api.AdminMigrationExportApplyRequest{
		PlanFingerprint:  plan.PlanFingerprint,
		Request:          request,
		ExportConfirmed:  true,
		PlayersConfirmed: playersConfirmed,
		SMBPassword:      smbPassword,
	})
	smbPassword = ""
	if err != nil {
		a.renderMigrationWorkspaceExportReview(w, http.StatusBadRequest, identity, request, plan, csrfFromRequest(r), apiMessage(err, ui(w, "ui.e65e1f03e013")), migrationWorkspaceSelectedID(r))
		return
	}
	if !result.OK || result.Operation == nil || result.Operation.OperationType != "migration_export" {
		a.renderMigrationWorkspaceExportReview(w, http.StatusBadGateway, identity, request, plan, csrfFromRequest(r), ui(w, "ui.85689180d9a0"), migrationWorkspaceSelectedID(r))
		return
	}
	http.Redirect(w, r, migrationSelectedURL(r, "/workspace/migration/progress/"+result.Operation.OperationID), http.StatusSeeOther)
}

func parseMigrationWorkspaceExportForm(r *http.Request, discovery api.AdminMigrationExportDiscoveryResponse) (api.AdminMigrationExportTargetRequest, error) {
	if err := r.ParseForm(); err != nil {
		return api.AdminMigrationExportTargetRequest{}, errors.New("invalid Server Export request")
	}
	kind := strings.TrimSpace(r.FormValue("kind"))
	filename := strings.TrimSpace(r.FormValue("filename"))
	if !migrationWorkspaceExportKindAvailable(discovery.TargetKinds, kind) {
		return api.AdminMigrationExportTargetRequest{}, errors.New("choose an available Server Export destination type")
	}
	if filename == "" || len(filename) > 160 || strings.ContainsAny(filename, "\r\n") {
		return api.AdminMigrationExportTargetRequest{}, errors.New("enter a valid migration bundle filename")
	}
	request := api.AdminMigrationExportTargetRequest{Kind: kind, Filename: filename}
	switch kind {
	case "local":
		request.Path = strings.TrimSpace(r.FormValue("path"))
		if !strings.HasPrefix(request.Path, "/") {
			return api.AdminMigrationExportTargetRequest{}, errors.New("enter an absolute local/server destination path")
		}
	case "backup":
		if !discovery.ConfiguredBackupAvailable {
			return api.AdminMigrationExportTargetRequest{}, errors.New("configured backup storage is not currently available for Server Export")
		}
	case "device":
		selected := strings.TrimSpace(r.FormValue("device"))
		found := false
		for _, device := range discovery.Devices {
			if device.Path == selected {
				request.Device = device.Path
				request.Removable = device.Removable
				found = true
				break
			}
		}
		if !found {
			return api.AdminMigrationExportTargetRequest{}, errors.New("choose an attached disk or removable-media destination reported by JustVoxel")
		}
	case "nfs":
		request.Source = strings.TrimSpace(r.FormValue("source"))
		if request.Source == "" || !strings.Contains(request.Source, ":") {
			return api.AdminMigrationExportTargetRequest{}, errors.New("enter the temporary NFS source as server:/export/path")
		}
	case "smb":
		rawLocation := strings.TrimSpace(r.FormValue("source"))
		shareRoot, embeddedFolder, splitErr := splitMigrationSMBLocation(rawLocation)
		if splitErr != nil {
			return api.AdminMigrationExportTargetRequest{}, splitErr
		}
		reviewedFolder := strings.TrimSpace(r.FormValue("path"))
		if embeddedFolder != "" && reviewedFolder != "" && embeddedFolder != reviewedFolder {
			return api.AdminMigrationExportTargetRequest{}, errors.New("the SMB destination folder changed; review the destination again")
		}
		if reviewedFolder == "" {
			reviewedFolder = embeddedFolder
		}
		request.Source = shareRoot
		request.Path = reviewedFolder
		request.Username = strings.TrimSpace(r.FormValue("username"))
		request.Domain = strings.TrimSpace(r.FormValue("domain"))
		if request.Username == "" {
			return api.AdminMigrationExportTargetRequest{}, errors.New("enter the SMB username")
		}
	default:
		return api.AdminMigrationExportTargetRequest{}, errors.New("unsupported Server Export destination type")
	}
	if strings.ContainsAny(request.Path+request.Device+request.Source+request.Username+request.Domain, "\r\n") {
		return api.AdminMigrationExportTargetRequest{}, errors.New(uiText("ui.a931435e8e31"))
	}
	return request, nil
}

func migrationWorkspaceExportKindAvailable(kinds []string, wanted string) bool {
	for _, kind := range kinds {
		if kind == wanted {
			return true
		}
	}
	return false
}

func (a *App) renderMigrationWorkspaceExportPage(w http.ResponseWriter, status int, identity api.SessionInfo, discovery api.AdminMigrationExportDiscoveryResponse, csrf, errorMessage string, serverID string) {
	attached := []migrationWorkspaceExportDeviceView{}
	removable := []migrationWorkspaceExportDeviceView{}
	for _, device := range discovery.Devices {
		view := migrationWorkspaceExportDeviceView{Device: device, Size: humanBytes(device.SizeBytes)}
		if device.Removable {
			removable = append(removable, view)
		} else {
			attached = append(attached, view)
		}
	}
	a.renderMigrationWorkspace(w, "migration_workspace_export.html", status, migrationWorkspaceExportPageData{
		ServerID:         serverID,
		Title:            ui(w, "ui.d1f081749243"),
		Version:          a.config.Version,
		ManagementAPI:    a.config.ManagementAPI,
		CSRF:             csrf,
		Identity:         identity,
		Discovery:        discovery,
		LocalAvailable:   migrationWorkspaceExportKindAvailable(discovery.TargetKinds, "local"),
		BackupAvailable:  migrationWorkspaceExportKindAvailable(discovery.TargetKinds, "backup") && discovery.ConfiguredBackupAvailable,
		DeviceAvailable:  migrationWorkspaceExportKindAvailable(discovery.TargetKinds, "device"),
		NFSAvailable:     migrationWorkspaceExportKindAvailable(discovery.TargetKinds, "nfs"),
		SMBAvailable:     migrationWorkspaceExportKindAvailable(discovery.TargetKinds, "smb"),
		AttachedDevices:  attached,
		RemovableDevices: removable,
		Error:            errorMessage,
	})
}

func (a *App) renderMigrationWorkspaceExportReview(w http.ResponseWriter, status int, identity api.SessionInfo, request api.AdminMigrationExportTargetRequest, plan api.AdminMigrationExportPlanResponse, csrf, errorMessage string, serverID string) {
	dataSize := ""
	availableSize := ""
	if plan.Normalized != nil {
		dataSize = humanBytes(plan.Normalized.DataBytes)
		if plan.Normalized.TargetAvailableBytes > 0 {
			availableSize = humanBytes(plan.Normalized.TargetAvailableBytes)
		}
	}
	a.renderMigrationWorkspace(w, "migration_workspace_export_review.html", status, migrationWorkspaceExportReviewPageData{
		ServerID:      serverID,
		Title:         ui(w, "ui.805afef53d4c"),
		Version:       a.config.Version,
		ManagementAPI: a.config.ManagementAPI,
		CSRF:          csrf,
		Identity:      identity,
		Request:       request,
		Plan:          plan,
		DataSize:      dataSize,
		AvailableSize: availableSize,
		Error:         errorMessage,
	})
}

func (a *App) redirectCurrentMigrationWorkspaceOperation(w http.ResponseWriter, r *http.Request, session string, client adminMigrationWorkspaceAPI) bool {
	response, err := client.AdminCurrentMigrationOperation(r.Context(), session)
	if err != nil {
		a.handleMigrationWorkspaceRequestError(w, r, err, ui(w, "ui.9ebe3ac32640"))
		return true
	}
	if response.Operation == nil {
		return false
	}
	if response.Operation.InstanceID != migrationWorkspaceSelectedID(r) || !migrationWorkspaceOperationType(response.Operation.OperationType) {
		http.Error(w, ui(w, "ui.a2ee2f05ca1e"), http.StatusBadGateway)
		return true
	}
	http.Redirect(w, r, migrationSelectedURL(r, "/workspace/migration/progress/"+response.Operation.OperationID), http.StatusSeeOther)
	return true
}

func (a *App) migrationWorkspaceProgressStatus(w http.ResponseWriter, r *http.Request) {
	session, ok := sessionFromRequest(r)
	if !ok {
		http.Error(w, ui(w, "ui.cbac8bee7536"), http.StatusUnauthorized)
		return
	}
	client, ok := a.api.(adminMigrationWorkspaceAPI)
	if !ok {
		http.Error(w, ui(w, "ui.0af96830990b"), http.StatusServiceUnavailable)
		return
	}
	identity, err := client.Session(r.Context(), session)
	if err != nil {
		if errors.Is(err, api.ErrUnauthorized) {
			a.clearSessionCookies(w)
			http.Error(w, ui(w, "ui.cbac8bee7536"), http.StatusUnauthorized)
			return
		}
		if errors.Is(err, api.ErrPasswordChangeRequired) {
			http.Error(w, ui(w, "ui.63bae92440ec"), http.StatusForbidden)
			return
		}
		http.Error(w, ui(w, "ui.0b8e73a792bb"), http.StatusBadGateway)
		return
	}
	if !isAdministratorLevel(identity.Role) {
		http.Error(w, ui(w, "ui.031cc7a3ec82"), http.StatusForbidden)
		return
	}
	selected, ok := a.selectedMinecraft(w, r, session)
	if !ok {
		return
	}
	*r = *r.WithContext(api.WithMigrationInstance(r.Context(), selected.ID))
	q := r.URL.Query()
	q.Set("server", selected.ID)
	r.URL.RawQuery = q.Encode()
	response, err := client.AdminOperation(r.Context(), session, r.PathValue("id"))
	if err != nil {
		var responseErr *api.ResponseError
		if errors.Is(err, api.ErrUnauthorized) {
			a.clearSessionCookies(w)
			http.Error(w, ui(w, "ui.cbac8bee7536"), http.StatusUnauthorized)
			return
		}
		if errors.Is(err, api.ErrPasswordChangeRequired) {
			http.Error(w, ui(w, "ui.63bae92440ec"), http.StatusForbidden)
			return
		}
		if errors.As(err, &responseErr) && responseErr.StatusCode == http.StatusNotFound {
			http.Error(w, ui(w, "ui.da2ab6e39c29"), http.StatusNotFound)
			return
		}
		http.Error(w, ui(w, "ui.1c7066d01f2d"), http.StatusBadGateway)
		return
	}
	if response.Operation == nil || response.Operation.InstanceID != migrationWorkspaceSelectedID(r) || !migrationWorkspaceOperationType(response.Operation.OperationType) {
		http.Error(w, ui(w, "ui.da2ab6e39c29"), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, ui(w, "ui.d9b0da01ddf3"), http.StatusInternalServerError)
	}
}

func migrationWorkspaceStageLabel(operationType, value string) string {
	if value == "queued" {
		return uiText("ui.f5e5f81fd256")
	}
	if value == "completed" {
		return "Complete"
	}
	if value == "player_recheck" {
		return uiText("ui.ab2d5bb1e79d")
	}
	if value == "minecraft_stop" {
		return uiText("ui.995beb2af122")
	}
	if value == "invalid_execution_plan" || value == "interrupted" {
		return uiText("ui.1e1b96ec8fd9")
	}
	switch operationType {
	case "migration_export":
		switch value {
		case "export_preflight":
			return uiText("ui.60117bda3d60")
		case "archive_integrity", "integrity_verification":
			return uiText("ui.78c93c0ea236")
		case "export_failed":
			return uiText("ui.a689980fba1b")
		case "rollback":
			return uiText("ui.d0d1b11758c9")
		case "export_rolled_back":
			return uiText("ui.39aec9840b2d")
		case "export_needs_attention", "export_backend_interrupted", "export_backend_incomplete":
			return uiText("ui.1e1b96ec8fd9")
		}
	case "migration_import":
		switch value {
		case "recovery_handoff":
			return uiText("ui.8c5a07566067")
		case "import_preflight":
			return uiText("ui.1db8eecfbe9b")
		case "import_storage":
			return uiText("ui.1b553da67100")
		case "import_execute":
			return uiText("ui.44c03884fea1")
		case "import_verify":
			return uiText("ui.70c4b99c7c87")
		case "import_failed":
			return uiText("ui.44d2868f2a28")
		case "rollback":
			return uiText("ui.f6e0f42acf8e")
		case "import_rolled_back":
			return uiText("ui.3d9d723dfd19")
		case "import_needs_attention", "import_backend_interrupted", "import_backend_incomplete":
			return uiText("ui.1e1b96ec8fd9")
		}
	case "migration_recovery":
		switch value {
		case "recovery_preflight":
			return uiText("ui.8e649c951902")
		case "recovery_finalize":
			return uiText("ui.665aaa8d5d59")
		case "recovery_verify":
			return uiText("ui.0cc638ce0ff6")
		case "recovery_backend_failed", "recovery_backend_invalid", "recovery_backend_incomplete":
			return uiText("ui.1e1b96ec8fd9")
		}
	}
	return migrationWorkspaceFriendlyToken(value)
}
