package server

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/home-server-project/justvoxel-webui/internal/api"
)

type adminMigrationWorkspaceImportAPI interface {
	adminMigrationWorkspaceAPI
	AdminMigrationImportPlan(ctx context.Context, session string, request api.AdminMigrationImportRequest) (api.AdminMigrationImportPlanResponse, error)
	AdminMigrationImportApply(ctx context.Context, session string, request api.AdminMigrationImportApplyRequest) (api.AdminMigrationApplyResponse, error)
	AdminStorage(ctx context.Context, session string) (api.AdminStorageDiscovery, error)
}

type migrationWorkspaceImportPageData struct {
	ServerID      string
	Title         string
	Version       string
	ManagementAPI string
	CSRF          string
	Identity      api.SessionInfo
	Discovery     api.AdminMigrationImportDiscoveryResponse
	Local         bool
	Backup        bool
	Device        bool
	NFS           bool
	SMB           bool
	SourceDevices []migrationWorkspaceExportDeviceView
	Filesystems   []setupFilesystemView
	Error         string
}

type migrationWorkspaceImportReviewPageData struct {
	ServerID            string
	Title               string
	Version             string
	ManagementAPI       string
	CSRF                string
	Identity            api.SessionInfo
	Request             api.AdminMigrationImportRequest
	Plan                api.AdminMigrationImportPlanResponse
	DataSize            string
	StorageSize         string
	BackupSize          string
	Prompt              string
	Error               string
	NeedsInput          bool
	Incompatible        bool
	SourceSoftware      string
	DestinationSoftware string
	NeedsSourcePath     bool
	NeedsRoot           bool
	NeedsVersion        bool
	SourceSMB           bool
	EULAURL             string
}

func (a *App) registerAdminMigrationWorkspaceImportPages(mux *http.ServeMux) {
	mux.HandleFunc("GET /workspace/migration/import", a.migrationWorkspaceImportPage)
	mux.HandleFunc("POST /workspace/migration/import/review", a.migrationWorkspaceImportReview)
	mux.HandleFunc("POST /workspace/migration/import/apply", a.migrationWorkspaceImportApply)
}

func (a *App) migrationWorkspaceImportRequest(w http.ResponseWriter, r *http.Request, requireCSRF bool) (string, adminMigrationWorkspaceImportAPI, api.SessionInfo, bool) {
	if requireCSRF && !a.validCSRF(r) {
		http.Error(w, "invalid CSRF token", http.StatusForbidden)
		return "", nil, api.SessionInfo{}, false
	}
	session, _, identity, ok := a.migrationWorkspaceRequest(w, r)
	if !ok {
		return "", nil, api.SessionInfo{}, false
	}
	client, ok := a.api.(adminMigrationWorkspaceImportAPI)
	if !ok {
		http.Error(w, ui(w, "ui.ac6951e54f48"), http.StatusServiceUnavailable)
		return "", nil, api.SessionInfo{}, false
	}
	return session, client, identity, true
}

func (a *App) migrationWorkspaceImportPage(w http.ResponseWriter, r *http.Request) {
	session, client, identity, ok := a.migrationWorkspaceImportRequest(w, r, false)
	if !ok {
		return
	}
	if migrationWorkspaceRedirectBrowser(w, r, "/?workspace=migration&tab=import") {
		return
	}
	if a.redirectCurrentMigrationWorkspaceOperation(w, r, session, client) {
		return
	}
	discovery, devices, inventory, ok := a.migrationWorkspaceImportDependencies(w, r, session, client)
	if !ok {
		return
	}
	a.renderMigrationWorkspaceImportPage(w, http.StatusOK, identity, discovery, devices, inventory, csrfFromRequest(r), "", migrationWorkspaceSelectedID(r))
}

func (a *App) migrationWorkspaceImportReview(w http.ResponseWriter, r *http.Request) {
	session, client, identity, ok := a.migrationWorkspaceImportRequest(w, r, true)
	if !ok {
		return
	}
	if a.redirectCurrentMigrationWorkspaceOperation(w, r, session, client) {
		return
	}
	discovery, devices, inventory, ok := a.migrationWorkspaceImportDependencies(w, r, session, client)
	if !ok {
		return
	}
	request, err := parseMigrationWorkspaceImportForm(r, discovery, devices, inventory)
	if err != nil {
		a.renderMigrationWorkspaceImportPage(w, http.StatusBadRequest, identity, discovery, devices, inventory, csrfFromRequest(r), err.Error(), migrationWorkspaceSelectedID(r))
		return
	}
	plan, err := client.AdminMigrationImportPlan(r.Context(), session, request)
	if err != nil {
		var responseErr *api.ResponseError
		if errors.As(err, &responseErr) && responseErr.StatusCode == http.StatusBadRequest && plan.SchemaVersion != "" {
			if migrationWorkspaceImportPlanNeedsInput(plan.Code) {
				a.renderMigrationWorkspaceImportReview(w, http.StatusOK, identity, scrubMigrationWorkspaceImportSecrets(request), plan, csrfFromRequest(r), migrationWorkspaceImportPrompt(plan), "", migrationWorkspaceSelectedID(r))
				return
			}
			a.renderMigrationWorkspaceImportReview(w, http.StatusBadRequest, identity, scrubMigrationWorkspaceImportSecrets(request), plan, csrfFromRequest(r), "", migrationWorkspaceImportPlanError(plan, err, ui(w, "ui.a422bb1f8713")), migrationWorkspaceSelectedID(r))
			return
		}
		a.handleMigrationWorkspaceRequestError(w, r, err, ui(w, "ui.9f6ea0843d43"))
		return
	}
	a.renderMigrationWorkspaceImportReview(w, http.StatusOK, identity, scrubMigrationWorkspaceImportSecrets(request), plan, csrfFromRequest(r), "", "", migrationWorkspaceSelectedID(r))
}

func (a *App) migrationWorkspaceImportApply(w http.ResponseWriter, r *http.Request) {
	session, client, identity, ok := a.migrationWorkspaceImportRequest(w, r, true)
	if !ok {
		return
	}
	if a.redirectCurrentMigrationWorkspaceOperation(w, r, session, client) {
		return
	}
	discovery, devices, inventory, ok := a.migrationWorkspaceImportDependencies(w, r, session, client)
	if !ok {
		return
	}
	request, err := parseMigrationWorkspaceImportForm(r, discovery, devices, inventory)
	if err != nil {
		a.renderMigrationWorkspaceImportPage(w, http.StatusBadRequest, identity, discovery, devices, inventory, csrfFromRequest(r), err.Error(), migrationWorkspaceSelectedID(r))
		return
	}
	plan, err := client.AdminMigrationImportPlan(r.Context(), session, request)
	if err != nil {
		var responseErr *api.ResponseError
		if errors.As(err, &responseErr) && responseErr.StatusCode == http.StatusBadRequest && plan.SchemaVersion != "" {
			prompt := ""
			if migrationWorkspaceImportPlanNeedsInput(plan.Code) {
				prompt = migrationWorkspaceImportPrompt(plan)
			}
			a.renderMigrationWorkspaceImportReview(w, http.StatusBadRequest, identity, scrubMigrationWorkspaceImportSecrets(request), plan, csrfFromRequest(r), prompt, migrationWorkspaceImportPlanError(plan, err, ui(w, "ui.15bcb6a29318")), migrationWorkspaceSelectedID(r))
			return
		}
		a.handleMigrationWorkspaceRequestError(w, r, err, ui(w, "ui.9f6ea0843d43"))
		return
	}

	publicRequest := scrubMigrationWorkspaceImportSecrets(request)
	if submitted := strings.TrimSpace(r.FormValue("plan_fingerprint")); submitted == "" || submitted != plan.PlanFingerprint {
		a.renderMigrationWorkspaceImportReview(w, http.StatusConflict, identity, publicRequest, plan, csrfFromRequest(r), "", ui(w, "ui.a64e64b1034c"), migrationWorkspaceSelectedID(r))
		return
	}
	if strings.TrimSpace(r.FormValue("import_confirmation")) != "IMPORT" {
		a.renderMigrationWorkspaceImportReview(w, http.StatusBadRequest, identity, publicRequest, plan, csrfFromRequest(r), "", ui(w, "ui.7e2d11808806"), migrationWorkspaceSelectedID(r))
		return
	}
	requirements := plan.Requirements
	if requirements == nil {
		a.renderMigrationWorkspaceImportReview(w, http.StatusBadGateway, identity, publicRequest, plan, csrfFromRequest(r), "", ui(w, "ui.6751c946f2c5"), migrationWorkspaceSelectedID(r))
		return
	}
	playersConfirmed := r.FormValue("players_confirmed") == "yes"
	eulaAccepted := r.FormValue("eula_accepted") == "yes"
	vanillaConfirmed := r.FormValue("vanilla_confirmed") == "yes"
	pluginsConfirmed := r.FormValue("plugins_confirmed") == "yes"
	onlineModeConfirmed := r.FormValue("online_mode_confirmed") == "yes"

	if requirements.PlayersConfirmationRequired && !playersConfirmed {
		a.renderMigrationWorkspaceImportReview(w, http.StatusBadRequest, identity, publicRequest, plan, csrfFromRequest(r), "", ui(w, "ui.6ed85a980b54"), migrationWorkspaceSelectedID(r))
		return
	}
	if requirements.VanillaConfirmationRequired && !vanillaConfirmed {
		a.renderMigrationWorkspaceImportReview(w, http.StatusBadRequest, identity, publicRequest, plan, csrfFromRequest(r), "", ui(w, "ui.c1463b7ff9e4"), migrationWorkspaceSelectedID(r))
		return
	}
	if requirements.PluginsConfirmationRequired && !pluginsConfirmed {
		a.renderMigrationWorkspaceImportReview(w, http.StatusBadRequest, identity, publicRequest, plan, csrfFromRequest(r), "", ui(w, "ui.87c0882ca808"), migrationWorkspaceSelectedID(r))
		return
	}
	if requirements.OnlineModeConfirmationRequired && !onlineModeConfirmed {
		a.renderMigrationWorkspaceImportReview(w, http.StatusBadRequest, identity, publicRequest, plan, csrfFromRequest(r), "", "Confirm that the source used online-mode=true before starting Server Import.", migrationWorkspaceSelectedID(r))
		return
	}

	backupSMBPassword := ""

	result, err := client.AdminMigrationImportApply(r.Context(), session, api.AdminMigrationImportApplyRequest{
		PlanFingerprint:     plan.PlanFingerprint,
		Request:             request,
		ImportConfirmed:     true,
		PlayersConfirmed:    playersConfirmed,
		EULAAccepted:        eulaAccepted,
		VanillaConfirmed:    vanillaConfirmed,
		PluginsConfirmed:    pluginsConfirmed,
		OnlineModeConfirmed: onlineModeConfirmed,
		BackupSMBPassword:   backupSMBPassword,
	})
	request.Source.SMBPassword = ""
	backupSMBPassword = ""
	if err != nil {
		a.renderMigrationWorkspaceImportReview(w, http.StatusBadRequest, identity, publicRequest, plan, csrfFromRequest(r), "", apiMessage(err, ui(w, "ui.f9adf77c9ecf")), migrationWorkspaceSelectedID(r))
		return
	}
	if !result.OK || result.Operation == nil || result.Operation.OperationType != "migration_import" {
		a.renderMigrationWorkspaceImportReview(w, http.StatusBadGateway, identity, publicRequest, plan, csrfFromRequest(r), "", ui(w, "ui.a84fcf28dfb4"), migrationWorkspaceSelectedID(r))
		return
	}
	http.Redirect(w, r, migrationSelectedURL(r, "/workspace/migration/progress/"+result.Operation.OperationID), http.StatusSeeOther)
}

func (a *App) migrationWorkspaceImportDependencies(w http.ResponseWriter, r *http.Request, session string, client adminMigrationWorkspaceImportAPI) (api.AdminMigrationImportDiscoveryResponse, api.AdminMigrationExportDiscoveryResponse, api.AdminStorageDiscovery, bool) {
	discovery, err := client.AdminMigrationImportDiscovery(r.Context(), session)
	if err != nil {
		a.handleMigrationWorkspaceRequestError(w, r, err, ui(w, "ui.11ba32d5bbba"))
		return api.AdminMigrationImportDiscoveryResponse{}, api.AdminMigrationExportDiscoveryResponse{}, api.AdminStorageDiscovery{}, false
	}
	devices, err := client.AdminMigrationExportDiscovery(r.Context(), session)
	if err != nil {
		a.handleMigrationWorkspaceRequestError(w, r, err, ui(w, "ui.c5cf2c28d72c"))
		return api.AdminMigrationImportDiscoveryResponse{}, api.AdminMigrationExportDiscoveryResponse{}, api.AdminStorageDiscovery{}, false
	}
	inventory := api.AdminStorageDiscovery{}
	if !discovery.Configured {
		http.Error(w, ui(w, "ui.63fd18644fde"), http.StatusConflict)
		return api.AdminMigrationImportDiscoveryResponse{}, api.AdminMigrationExportDiscoveryResponse{}, inventory, false
	}

	return discovery, devices, inventory, true
}

func parseMigrationWorkspaceImportForm(r *http.Request, discovery api.AdminMigrationImportDiscoveryResponse, devices api.AdminMigrationExportDiscoveryResponse, inventory api.AdminStorageDiscovery) (api.AdminMigrationImportRequest, error) {
	if err := r.ParseForm(); err != nil {
		return api.AdminMigrationImportRequest{}, errors.New("invalid Server Import request")
	}
	kind := strings.TrimSpace(r.FormValue("source_kind"))
	if !migrationWorkspaceExportKindAvailable(discovery.SourceKinds, kind) {
		return api.AdminMigrationImportRequest{}, errors.New("choose an available Server Import source type")
	}

	source := api.AdminMigrationImportSourceRequest{
		Kind:          kind,
		SelectedRoot:  strings.TrimSpace(r.FormValue("selected_root")),
		SourceVersion: strings.TrimSpace(r.FormValue("source_version")),
	}
	manualPath := strings.TrimSpace(r.FormValue("manual_path"))
	relativePath := strings.TrimSpace(r.FormValue("source_path"))
	if manualPath != "" {
		relativePath = manualPath
	}

	switch kind {
	case "local":
		source.Path = strings.TrimSpace(r.FormValue("local_path"))
		if !strings.HasPrefix(source.Path, "/") {
			return api.AdminMigrationImportRequest{}, errors.New("enter an absolute local/server Import source path")
		}
	case "backup":
		source.Path = relativePath
	case "device":
		source.Path = relativePath
		selected := strings.TrimSpace(r.FormValue("source_device"))
		found := false
		for _, device := range devices.Devices {
			if device.Path == selected {
				source.Device = device.Path
				source.Removable = device.Removable
				found = true
				break
			}
		}
		if !found {
			return api.AdminMigrationImportRequest{}, errors.New("choose an attached disk, partition, or removable source reported by JustVoxel")
		}
	case "nfs":
		source.Path = relativePath
		source.Source = strings.TrimSpace(r.FormValue("source_remote"))
		if source.Source == "" || !strings.Contains(source.Source, ":") {
			return api.AdminMigrationImportRequest{}, errors.New("enter the temporary NFS source as server:/export/path")
		}
	case "smb":
		rawLocation := strings.TrimSpace(r.FormValue("source_remote"))
		shareRoot, embeddedPath, splitErr := splitMigrationSMBLocation(rawLocation)
		if splitErr != nil {
			return api.AdminMigrationImportRequest{}, splitErr
		}
		if relativePath == "" {
			relativePath = embeddedPath
		}
		source.Path = relativePath
		source.Source = shareRoot
		source.Username = strings.TrimSpace(r.FormValue("source_username"))
		source.Domain = strings.TrimSpace(r.FormValue("source_domain"))
		source.SMBPassword = r.FormValue("source_smb_password")
		if source.Username == "" {
			return api.AdminMigrationImportRequest{}, errors.New("enter the SMB username")
		}
		if source.SMBPassword == "" {
			return api.AdminMigrationImportRequest{}, errors.New("enter the SMB password so JustVoxel can inspect this location")
		}
	default:
		return api.AdminMigrationImportRequest{}, errors.New("unsupported Server Import source type")
	}
	if strings.ContainsAny(source.Path+source.Device+source.Source+source.Username+source.Domain+source.SelectedRoot+source.SourceVersion, "\r\n") {
		return api.AdminMigrationImportRequest{}, errors.New(uiText("ui.b1663c5a507d"))
	}

	destination, err := parseMigrationWorkspaceImportDestination(r, discovery, inventory)
	if err != nil {
		return api.AdminMigrationImportRequest{}, err
	}
	return api.AdminMigrationImportRequest{Source: source, Destination: destination}, nil
}

func parseMigrationWorkspaceImportDestination(r *http.Request, discovery api.AdminMigrationImportDiscoveryResponse, inventory api.AdminStorageDiscovery) (api.AdminMigrationImportDestinationRequest, error) {
	defaults := discovery.Defaults
	if discovery.Configured {
		return api.AdminMigrationImportDestinationRequest{
			JavaPort:        defaults.JavaPort,
			BedrockPort:     defaults.BedrockPort,
			JavaMemory:      defaults.JavaMemory,
			ContainerMemory: defaults.ContainerMemory,
			Timezone:        defaults.Timezone,
			BackupKeep:      defaults.BackupKeep,
			BackupDailyTime: defaults.BackupDailyTime,
			BackupAutomatic: defaults.BackupAutomatic,
			Storage:         api.AdminSetupPlanStorageRequest{},
			Backups:         api.AdminSetupPlanBackupsRequest{},
		}, nil
	}

	return api.AdminMigrationImportDestinationRequest{}, errors.New(uiText("ui.63fd18644fde"))
}

func importServerSoftwareLabel(serverType string) string {
	switch serverType {
	case "paper", "itzg-paper":
		return "Paper"
	case "purpur":
		return "Purpur"
	case "vanilla":
		return "Vanilla"
	default:
		return serverType
	}
}

func scrubMigrationWorkspaceImportSecrets(request api.AdminMigrationImportRequest) api.AdminMigrationImportRequest {
	request.Source.SMBPassword = ""
	return request
}

func migrationWorkspaceImportPlanNeedsInput(code string) bool {
	switch code {
	case "source_selection_required", "multiple_roots", "source_version_required", "stale_root":
		return true
	default:
		return false
	}
}

func migrationWorkspaceImportPlanError(plan api.AdminMigrationImportPlanResponse, err error, fallback string) string {
	if plan.Error != "" {
		return plan.Error
	}
	return apiMessage(err, fallback)
}

func migrationWorkspaceImportPrompt(plan api.AdminMigrationImportPlanResponse) string {
	switch plan.Code {
	case "source_selection_required":
		return uiText("ui.e84d0fa2997d")
	case "multiple_roots", "stale_root":
		return uiText("ui.ca95db728c63")
	case "source_version_required":
		return uiText("ui.56a9a391159d")
	default:
		return ""
	}
}

func (a *App) renderMigrationWorkspaceImportPage(w http.ResponseWriter, status int, identity api.SessionInfo, discovery api.AdminMigrationImportDiscoveryResponse, devices api.AdminMigrationExportDiscoveryResponse, inventory api.AdminStorageDiscovery, csrf, errorMessage string, serverID string) {
	deviceViews := make([]migrationWorkspaceExportDeviceView, 0, len(devices.Devices))
	for _, device := range devices.Devices {
		deviceViews = append(deviceViews, migrationWorkspaceExportDeviceView{Device: device, Size: humanBytes(device.SizeBytes)})
	}
	a.renderMigrationWorkspace(w, "migration_workspace_import.html", status, migrationWorkspaceImportPageData{
		ServerID:      serverID,
		Title:         ui(w, "ui.ffe624a69618"),
		Version:       a.config.Version,
		ManagementAPI: a.config.ManagementAPI,
		CSRF:          csrf,
		Identity:      identity,
		Discovery:     discovery,
		Local:         migrationWorkspaceExportKindAvailable(discovery.SourceKinds, "local"),
		Backup:        migrationWorkspaceExportKindAvailable(discovery.SourceKinds, "backup"),
		Device:        migrationWorkspaceExportKindAvailable(discovery.SourceKinds, "device"),
		NFS:           migrationWorkspaceExportKindAvailable(discovery.SourceKinds, "nfs"),
		SMB:           migrationWorkspaceExportKindAvailable(discovery.SourceKinds, "smb"),
		SourceDevices: deviceViews,
		Filesystems:   setupFilesystemViews(inventory),
		Error:         errorMessage,
	})
}

func (a *App) renderMigrationWorkspaceImportReview(w http.ResponseWriter, status int, identity api.SessionInfo, request api.AdminMigrationImportRequest, plan api.AdminMigrationImportPlanResponse, csrf, prompt, errorMessage string, serverID string) {
	incompatible := !plan.OK && plan.Code == "unsupported_server_type" && plan.Normalized != nil
	sourceSoftware, destinationSoftware := "", ""
	if plan.Normalized != nil {
		sourceType := plan.Normalized.Source.ServerType
		if sourceType == "" {
			sourceType = plan.Normalized.Source.CandidateType
		}
		sourceSoftware = importServerSoftwareLabel(sourceType)
		destinationSoftware = importServerSoftwareLabel(plan.Normalized.Destination.ServerType)
	}
	if incompatible {
		errorMessage = ""
		prompt = ""
	}
	dataSize := ""
	storageSize := ""
	backupSize := ""
	if plan.Normalized != nil {
		dataSize = humanBytes(plan.Normalized.Source.ExpandedBytes)
		if plan.Normalized.Destination.Storage != nil && plan.Normalized.Destination.Storage.SizeBytes > 0 {
			storageSize = humanBytes(plan.Normalized.Destination.Storage.SizeBytes)
		}
		if plan.Normalized.Destination.Backups != nil && plan.Normalized.Destination.Backups.SizeBytes > 0 {
			backupSize = humanBytes(plan.Normalized.Destination.Backups.SizeBytes)
		}
	}
	a.renderMigrationWorkspace(w, "migration_workspace_import_review.html", status, migrationWorkspaceImportReviewPageData{
		ServerID:            serverID,
		Title:               ui(w, "ui.7ee41abefcae"),
		Version:             a.config.Version,
		ManagementAPI:       a.config.ManagementAPI,
		CSRF:                csrf,
		Identity:            identity,
		Request:             request,
		Plan:                plan,
		DataSize:            dataSize,
		StorageSize:         storageSize,
		BackupSize:          backupSize,
		Prompt:              prompt,
		Error:               errorMessage,
		NeedsInput:          migrationWorkspaceImportPlanNeedsInput(plan.Code),
		Incompatible:        incompatible,
		SourceSoftware:      sourceSoftware,
		DestinationSoftware: destinationSoftware,
		NeedsSourcePath:     plan.Code == "source_selection_required",
		NeedsRoot:           plan.Code == "multiple_roots" || plan.Code == "stale_root",
		NeedsVersion:        plan.Code == "source_version_required",
		SourceSMB:           request.Source.Kind == "smb",
		EULAURL:             minecraftEULAURL,
	})
}
