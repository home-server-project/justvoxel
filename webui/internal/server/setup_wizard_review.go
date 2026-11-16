package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/home-server-project/justvoxel-webui/internal/api"
)

const minecraftEULAURL = "https://www.minecraft.net/eula"

var setupPlanFingerprintPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

type setupPlanningAPI interface {
	AdminSetupPlan(ctx context.Context, session string, request api.AdminSetupPlanRequest) (api.AdminSetupPlanResponse, error)
}

type setupReviewState struct {
	AddServer       bool
	DisplayName     string
	StartAfterSetup bool
	StartOnBoot     bool
	BackupAvailable bool
	CreationBlocked bool
	InstanceRuntime api.InstanceRuntimeSettings
	InstanceBackup  api.InstanceBackupSettings
	Request         api.AdminSetupPlanRequest
	Plan            api.AdminSetupPlanResponse
	EULAAccepted    bool
	DraftUpdatedAt  time.Time
	SetupMode       string
	ServerType      string
}

type setupReviewStore struct {
	mu      sync.Mutex
	reviews map[setupDraftKey]setupReviewState
}

var setupReviews = setupReviewStore{reviews: make(map[setupDraftKey]setupReviewState)}

type setupReviewPageData struct {
	AddServer          bool
	BasePath           string
	DisplayName        string
	StartAfterSetup    bool
	StartOnBoot        bool
	CreationBlocked    bool
	Title              string
	SetupMode          string
	ServerTypeLabel    string
	Version            string
	ManagementAPI      string
	CSRF               string
	Identity           api.SessionInfo
	Plan               api.AdminSetupPlanResponse
	Error              string
	EULAAccepted       bool
	EULAURL            string
	StorageLabel       string
	StorageSize        string
	BackupLabel        string
	BackupSize         string
	VersionLabel       string
	VersionPolicyLabel string
	ImageChannelLabel  string
	GameModeLabel      string
}

func (a *App) registerSetupWizardReviewRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /setup/review", a.setupWizardReviewPage)
	mux.HandleFunc("GET /setup/review/configuration", a.setupWizardReviewConfiguration)
	mux.HandleFunc("POST /setup/review/eula", a.setupWizardReviewEULA)
	mux.HandleFunc("POST /setup/review/back", a.setupWizardReviewBack)
	mux.HandleFunc("POST /setup/review/cancel", a.setupWizardCancel)
	a.registerSetupWizardApplyRoutes(mux)
}

func (a *App) setupWizardReviewPage(w http.ResponseWriter, r *http.Request) {
	session, client, identity, ok := a.setupWizardRequest(w, r, false)
	if !ok {
		return
	}
	if a.redirectCurrentSetupOperation(w, r, session, client) {
		return
	}
	draft, exists := setupDrafts.get(a, session)
	if !exists || !setupDraftReadyForReview(draft) {
		http.Redirect(w, r, setupBasePath(r), http.StatusSeeOther)
		return
	}

	state, message, status, ok := a.setupReviewState(r.Context(), session, client, draft)
	state.CreationBlocked = draft.CreationAttempted && !draft.CoreCreated
	if !ok {
		state.AddServer, state.SetupMode, state.DisplayName = draft.AddServer, draft.Mode, draft.DisplayName
		if status == http.StatusUnauthorized || status == http.StatusForbidden {
			a.handleAdminDiscoveryError(w, r, message)
			return
		}
		w.WriteHeader(status)
		a.renderSetupReview(w, identity, state, csrfFromRequest(r), setupReviewErrorMessage(message))
		return
	}
	a.renderSetupReview(w, identity, state, csrfFromRequest(r), draft.CreationError)
}

func (a *App) setupWizardReviewConfiguration(w http.ResponseWriter, r *http.Request) {
	session, client, _, ok := a.setupWizardRequest(w, r, false)
	if !ok {
		return
	}
	draft, exists := setupDrafts.get(a, session)
	if !exists || !setupDraftReadyForReview(draft) {
		http.Redirect(w, r, setupBasePath(r), http.StatusSeeOther)
		return
	}
	state, err, status, ready := a.setupReviewState(r.Context(), session, client, draft)
	if !ready {
		http.Error(w, setupReviewErrorMessage(err), status)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\"justvoxel-setup.txt\"")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(setupConfigurationSnapshot(state.Plan, requestLocale(r))))
}

func setupConfigurationSnapshot(plan api.AdminSetupPlanResponse, locales ...string) string {
	text := uiText
	if len(locales) > 0 && supportedLocale(locales[0]) {
		text = func(key string, args ...any) string { return translate(locales[0], key, args...) }
	}
	var b strings.Builder
	b.WriteString(text("ui.c50f9c821dea"))
	b.WriteString(text("ui.85e3e51739ba") + shortSetupPlanReference(plan.PlanFingerprint) + "\n\n")

	b.WriteString("Server\n")
	if plan.Normalized.Server.DisplayName != "" {
		b.WriteString(text("ui.485ab5092843") + plan.Normalized.Server.DisplayName + "\n")
	}
	b.WriteString(text("ui.a2c01d240982") + plan.Normalized.Server.MOTD + "\n")
	b.WriteString(text("ui.b46f2b34ae1a") + strconv.Itoa(plan.Normalized.Server.MaxPlayers) + "\n")
	if plan.Normalized.Minecraft.ServerType == "bedrock" {
		b.WriteString(text("ui.e22fdb4f6ff4"))
	} else if plan.Normalized.Server.BedrockEnabled {
		b.WriteString(text("ui.3a5399d89a9c"))
	} else {
		b.WriteString(text("ui.9d0d9670a493"))
	}

	b.WriteString("Minecraft\n")
	b.WriteString(text("ui.dd18be3fa497") + setupServerTypeLabel(plan.Normalized.Minecraft.ServerType) + "\n")
	b.WriteString(text("ui.cceff78a0388") + setupGameModeLabel(plan.Normalized.Minecraft.GameMode) + "\n")
	if plan.Normalized.Minecraft.ServerType == "bedrock" {
		b.WriteString(fmt.Sprintf(text("ui.e2f15a94687a"), plan.Normalized.Minecraft.BedrockTCPPort, plan.Normalized.Minecraft.BedrockUDPStart, plan.Normalized.Minecraft.BedrockUDPEnd, plan.Normalized.Minecraft.ContainerMemory))
	} else {
		b.WriteString(text("ui.1e07f7664304") + plan.Normalized.Minecraft.JavaMemory + "\n")
		b.WriteString(text("ui.4d28004ccd28") + plan.Normalized.Minecraft.ContainerMemory + "\n")
		b.WriteString(text("ui.b48beae6f5b7") + strconv.Itoa(plan.Normalized.Minecraft.JavaPort) + "/TCP\n")
		if plan.Normalized.Server.BedrockEnabled {
			b.WriteString(text("ui.d64e113696b5") + strconv.Itoa(plan.Normalized.Minecraft.BedrockPort) + "/TCP/UDP\n")
		}
		b.WriteString(text("ui.de2fe7222dea") + plan.Normalized.Minecraft.ImageTag + "\n")
	}
	b.WriteString(text("ui.a1feb8978a3e") + plan.Normalized.Minecraft.Version + "\n\n")

	b.WriteString(text("ui.d6cdbb62fee6"))
	b.WriteString(text("ui.9bfe22f3c187") + plan.Normalized.Storage.Path + "\n")
	if plan.Normalized.Storage.Device != "" {
		b.WriteString("Device: " + plan.Normalized.Storage.Device + "\n")
	}
	if plan.Normalized.Storage.Filesystem != "" {
		b.WriteString("Filesystem: " + plan.Normalized.Storage.Filesystem + "\n")
	}
	if plan.Normalized.Storage.UUID != "" {
		b.WriteString("UUID: " + plan.Normalized.Storage.UUID + "\n")
	}
	if plan.Normalized.Storage.MountPoint != "" {
		b.WriteString(text("ui.411db0c42e81") + plan.Normalized.Storage.MountPoint + "\n")
	}
	b.WriteString(text("ui.532372475397"))
	if plan.Normalized.Backups.Automatic {
		b.WriteString(text("ui.c53faa906bb1"))
		b.WriteString(text("ui.ec536ddebc55") + plan.Normalized.Backups.DailyTime + "\n")
	} else {
		b.WriteString(text("ui.b0092fa23991"))
	}
	b.WriteString(text("ui.83ca4e01db58") + strconv.Itoa(plan.Normalized.Backups.Keep) + "\n")
	b.WriteString(text("ui.2d7b920015ab") + plan.Normalized.Backups.Path + "\n")
	if plan.Normalized.Backups.Device != "" {
		b.WriteString("Device: " + plan.Normalized.Backups.Device + "\n")
	}
	if plan.Normalized.Backups.Filesystem != "" {
		b.WriteString("Filesystem: " + plan.Normalized.Backups.Filesystem + "\n")
	}
	if plan.Normalized.Backups.UUID != "" {
		b.WriteString("UUID: " + plan.Normalized.Backups.UUID + "\n")
	}
	if plan.Normalized.Backups.Source != "" {
		b.WriteString(text("ui.5e05d1545763") + plan.Normalized.Backups.Source + "\n")
	}
	if plan.Normalized.Backups.MountPoint != "" {
		b.WriteString(text("ui.411db0c42e81") + plan.Normalized.Backups.MountPoint + "\n")
	}
	if len(plan.Warnings) > 0 {
		b.WriteString("\nWarnings\n")
		for _, warning := range plan.Warnings {
			b.WriteString("- " + warning.Message + "\n")
		}
	}
	b.WriteString(text("ui.12c638e1d2f5"))
	return b.String()
}
func (a *App) setupWizardReviewEULA(w http.ResponseWriter, r *http.Request) {
	session, client, identity, ok := a.setupWizardRequest(w, r, true)
	if !ok {
		return
	}
	if a.redirectCurrentSetupOperation(w, r, session, client) {
		return
	}
	draft, exists := setupDrafts.get(a, session)
	if !exists || !setupDraftReadyForReview(draft) {
		http.Redirect(w, r, setupBasePath(r), http.StatusSeeOther)
		return
	}
	state, planErr, status, ready := a.setupReviewState(r.Context(), session, client, draft)
	if !ready {
		if errors.Is(planErr, api.ErrUnauthorized) || errors.Is(planErr, api.ErrPasswordChangeRequired) {
			a.handleAdminDiscoveryError(w, r, planErr)
			return
		}
		w.WriteHeader(status)
		a.renderSetupReview(w, identity, state, csrfFromRequest(r), setupReviewErrorMessage(planErr))
		return
	}
	if submitted := r.FormValue("plan_fingerprint"); submitted == "" || submitted != state.Plan.PlanFingerprint {
		state.EULAAccepted = false
		setupReviews.save(a, session, state)
		w.WriteHeader(http.StatusConflict)
		a.renderSetupReview(w, identity, state, csrfFromRequest(r), ui(w, "ui.9d942c10f2b5"))
		return
	}
	if r.FormValue("eula_accepted") != "on" {
		state.EULAAccepted = false
		setupReviews.save(a, session, state)
		w.WriteHeader(http.StatusBadRequest)
		a.renderSetupReview(w, identity, state, csrfFromRequest(r), ui(w, "ui.59895b0148d2"))
		return
	}
	state.EULAAccepted = true
	setupReviews.save(a, session, state)
	a.recordSetupDraftDiagnosticBestEffort(r.Context(), client, session, "Minecraft EULA acceptance recorded", draft)
	http.Redirect(w, r, setupBasePath(r)+"/review", http.StatusSeeOther)
}

func (a *App) setupWizardReviewBack(w http.ResponseWriter, r *http.Request) {
	session, client, _, ok := a.setupWizardRequest(w, r, true)
	if !ok {
		return
	}
	if a.redirectCurrentSetupOperation(w, r, session, client) {
		return
	}
	draft, exists := setupDrafts.get(a, session)
	if !exists || !draft.Started {
		http.Redirect(w, r, setupBasePath(r), http.StatusSeeOther)
		return
	}
	setupReviews.delete(a, session)
	if draft.Mode == "recommended" && !draft.CoreCreated {
		draft.Started = false
		draft.SimpleState = 3
		draft.CurrentStep = setupServerStep
	} else {
		draft.CurrentStep = setupBackupsStep
	}
	setupDrafts.save(a, session, draft)
	a.recordSetupDraftDiagnosticBestEffort(r.Context(), client, session, "user returned from Review to Backups", draft)
	http.Redirect(w, r, setupBasePath(r), http.StatusSeeOther)
}

func (a *App) setupReviewState(ctx context.Context, session string, client adminDiscoveryAPI, draft setupDraft) (setupReviewState, error, int, bool) {
	request, err := setupPlanRequestFromDraft(draft)
	if err != nil {
		return setupReviewState{}, err, http.StatusBadRequest, false
	}
	if cached, ok := setupReviews.get(a, session); ok && cached.DraftUpdatedAt.Equal(draft.UpdatedAt) && cached.Request == request && validSetupPlanFingerprint(cached.Plan.PlanFingerprint) {
		return cached, nil, http.StatusOK, true
	}
	planner, ok := client.(setupPlanningAPI)
	if !ok {
		return setupReviewState{}, errors.New("setup planning is unavailable"), http.StatusServiceUnavailable, false
	}
	plan, err := planner.AdminSetupPlan(ctx, session, request)
	state := setupReviewState{
		AddServer: draft.AddServer, StartOnBoot: draft.StartOnBoot, StartAfterSetup: draft.StartAfterSetup, BackupAvailable: true, Request: request, Plan: plan, DraftUpdatedAt: draft.UpdatedAt,
		SetupMode: draft.Mode, ServerType: draft.Minecraft.ServerType, DisplayName: plan.Normalized.Server.DisplayName,
	}
	if err != nil {
		if errors.Is(err, api.ErrUnauthorized) {
			return state, err, http.StatusUnauthorized, false
		}
		if errors.Is(err, api.ErrPasswordChangeRequired) {
			return state, err, http.StatusForbidden, false
		}
		var responseErr *api.ResponseError
		if errors.As(err, &responseErr) && responseErr.StatusCode == http.StatusBadRequest {
			return state, err, http.StatusBadRequest, false
		}
		return state, err, http.StatusBadGateway, false
	}
	if !plan.OK {
		return state, errors.New("setup planning did not return a validated configuration"), http.StatusBadRequest, false
	}
	if !validSetupPlanFingerprint(plan.PlanFingerprint) {
		return state, errors.New("setup planning did not return a valid plan identity"), http.StatusBadGateway, false
	}
	setupReviews.save(a, session, state)
	return state, nil, http.StatusOK, true
}

func validSetupPlanFingerprint(value string) bool {
	return setupPlanFingerprintPattern.MatchString(value)
}

func shortSetupPlanReference(value string) string {
	if !validSetupPlanFingerprint(value) {
		return ""
	}
	hexValue := strings.TrimPrefix(value, "sha256:")
	return hexValue[:8] + "…"
}

func setupDraftReadyForReview(draft setupDraft) bool {
	return draft.Started && draft.CurrentStep == setupReviewStep && draft.Server.Complete && draft.Minecraft.ConnectionsComplete && draft.Minecraft.ResourcesComplete && draft.Minecraft.Complete && draft.Storage.Complete && draft.Backups.Complete
}

func setupPlanRequestFromDraft(draft setupDraft) (api.AdminSetupPlanRequest, error) {
	draft.Minecraft = effectiveSetupMinecraft(draft.Minecraft)
	maxPlayers, err := strconv.Atoi(draft.Server.MaxPlayers)
	if err != nil || maxPlayers < 1 {
		return api.AdminSetupPlanRequest{}, errors.New(uiText("ui.781dc8534fc6"))
	}
	javaPort, err := strconv.Atoi(draft.Minecraft.JavaPort)
	if err != nil {
		return api.AdminSetupPlanRequest{}, errors.New(uiText("ui.00ed659686cd"))
	}
	bedrockPort, err := strconv.Atoi(draft.Minecraft.BedrockPort)
	if err != nil {
		return api.AdminSetupPlanRequest{}, errors.New(uiText("ui.28b17d10f4c4"))
	}
	backupKeep, err := strconv.Atoi(draft.Backups.Keep)
	if err != nil || backupKeep < 1 {
		return api.AdminSetupPlanRequest{}, errors.New(uiText("ui.a7d2c45bfca1"))
	}
	return api.AdminSetupPlanRequest{
		AddServer: draft.AddServer, StartOnBoot: draft.StartOnBoot, StartAfterSetup: draft.StartAfterSetup, MountedOnly: draft.Mode == "recommended", DiagnosticSessionID: draft.DiagnosticSessionID,
		Server: api.AdminSetupPlanServerRequest{
			DisplayName: draft.DisplayName, MOTD: draft.Server.MOTD, MOTDAutomatic: draft.Server.MOTDAutomatic, MaxPlayers: maxPlayers,
			BedrockEnabled: draft.Server.BedrockEnabled,
		},
		Minecraft: api.AdminSetupPlanMinecraftRequest{
			LoaderVersion: draft.Minecraft.LoaderVersion, ServerType: draft.Minecraft.ServerType, JavaMemory: draft.Minecraft.JavaMemory, ContainerMemory: draft.Minecraft.ContainerMemory,
			JavaPort: javaPort, BedrockPort: bedrockPort, BedrockTCPPort: draft.Minecraft.BedrockTCPPort, BedrockUDPStart: draft.Minecraft.BedrockUDPStart, BedrockUDPEnd: draft.Minecraft.BedrockUDPEnd, ImageTag: draft.Minecraft.ImageTag,
			VersionPolicy: draft.Minecraft.VersionPolicy, Version: draft.Minecraft.Version, GameMode: draft.Minecraft.GameMode,
		},
		Storage: api.AdminSetupPlanStorageRequest{
			Type: draft.Storage.Type, Path: draft.Storage.Path, Device: draft.Storage.Device, MountPoint: draft.Storage.MountPoint,
		},
		Backups: api.AdminSetupPlanBackupsRequest{
			Automatic: draft.Backups.Automatic, DailyTime: draft.Backups.DailyTime, Keep: backupKeep,
			Type: draft.Backups.Type, Path: draft.Backups.Path, Device: draft.Backups.Device,
			MountPoint: draft.Backups.MountPoint, Source: draft.Backups.Source,
			Username: draft.Backups.Username, Domain: draft.Backups.Domain,
		},
	}, nil
}

func setupReviewErrorMessage(err error) string {
	if err == nil {
		return uiText("ui.9fabd6a957c3")
	}
	if message, ok := api.ErrorMessage(err); ok {
		return message
	}
	if err.Error() == "setup planning is unavailable" || err.Error() == "setup planning did not return a valid plan identity" {
		return uiText("ui.9fabd6a957c3")
	}
	return err.Error()
}

func (a *App) renderSetupReview(w http.ResponseWriter, identity api.SessionInfo, state setupReviewState, csrf, errorMessage string) {
	plan := state.Plan
	basePath := "/setup"
	if state.AddServer {
		basePath = addServerBasePath
	}
	data := setupReviewPageData{
		AddServer: state.AddServer, BasePath: basePath, DisplayName: state.DisplayName,
		StartOnBoot: state.StartOnBoot, StartAfterSetup: state.StartAfterSetup, CreationBlocked: state.CreationBlocked,
		Title: ui(w, "ui.5a16e3d43a6a"), SetupMode: state.SetupMode, ServerTypeLabel: setupServerTypeLabel(plan.Normalized.Minecraft.ServerType),
		Version: a.config.Version, ManagementAPI: a.config.ManagementAPI,
		CSRF: csrf, Identity: identity, Plan: plan, Error: errorMessage,
		EULAAccepted: state.EULAAccepted, EULAURL: minecraftEULAURL,
		StorageLabel:       setupStorageTypeLabel(plan.Normalized.Storage.Type, plan.Normalized.Storage.Transport),
		BackupLabel:        setupBackupTypeLabel(plan.Normalized.Backups.Type, plan.Normalized.Backups.Transport),
		VersionLabel:       setupVersionPolicyLabel(plan.Normalized.Minecraft.RequestedVersionPolicy),
		VersionPolicyLabel: strings.TrimPrefix(setupVersionPolicyLabel(plan.Normalized.Minecraft.RequestedVersionPolicy), "· "),
		ImageChannelLabel:  setupImageChannelLabel(plan.Normalized.Minecraft.ImageTag),
		GameModeLabel:      setupGameModeLabel(plan.Normalized.Minecraft.GameMode),
	}
	if validSetupServerType(plan.Normalized.Minecraft.ServerType) && setupNeedsExactRuntime(plan.Normalized.Minecraft.ServerType) {
		data.ImageChannelLabel = ui(w, "ui.d461a493a375")
	}
	if plan.Normalized.Storage.SizeBytes > 0 {
		data.StorageSize = humanBytes(plan.Normalized.Storage.SizeBytes)
	}
	if plan.Normalized.Backups.SizeBytes > 0 {
		data.BackupSize = humanBytes(plan.Normalized.Backups.SizeBytes)
	}
	if state.AddServer || state.SetupMode == "recommended" {
		data.StorageLabel += " · " + plan.Normalized.Storage.Path
		data.BackupLabel += " · " + plan.Normalized.Backups.Path
	}

	a.renderAdminDiscovery(w, "setup_review.html", data)
}

func setupGameModeLabel(mode string) string {
	switch mode {
	case "survival":
		return uiText("ui.8302496de0ac")
	case "creative":
		return uiText("ui.2c7e63313045")
	case "adventure":
		return uiText("ui.9a3c16424750")
	case "spectator":
		return uiText("ui.5ccda7f08fd1")
	}
	return mode
}

func setupServerTypeLabel(value string) string {
	switch value {
	case "paper", "":
		return "Paper"
	case "purpur":
		return "Purpur"
	case "spigot":
		return "Spigot"
	case "pufferfish", "leaf":
		return serverSoftware(value)
	case "vanilla":
		return "Vanilla"
	case "fabric":
		return "Fabric"
	case "forge":
		return "Forge"
	case "neoforge":
		return "NeoForge"
	case "bedrock":
		return "Bedrock"
	case "quilt":
		return "Quilt"
	default:
		return value
	}
}

func setupStorageTypeLabel(value, transport string) string {
	switch value {
	case "system":
		return uiText("ui.78157120c120")
	case "partition":
		if strings.EqualFold(strings.TrimSpace(transport), "usb") {
			return uiText("ui.7c7ddfe3f00a")
		}
		return uiText("ui.d62f03d8ff00")
	default:
		return value
	}
}

func setupBackupTypeLabel(value, transport string) string {
	switch value {
	case "system":
		return uiText("ui.78157120c120")
	case "partition":
		if strings.EqualFold(strings.TrimSpace(transport), "usb") {
			return uiText("ui.7c7ddfe3f00a")
		}
		return uiText("ui.d62f03d8ff00")
	case "nfs":
		return "NFS share"
	case "smb":
		return uiText("ui.d9e817cf067e")
	default:
		return value
	}
}

func setupVersionPolicyLabel(value string) string {
	switch value {
	case "recommended":
		return "· Recommended"
	case "latest":
		return "· Latest"
	case "pinned":
		return "· Specific"
	default:
		return value
	}
}

func setupImageChannelLabel(tag string) string {
	switch tag {
	case "stable":
		return uiText("ui.90ee305714d7")
	case "latest":
		return uiText("ui.8730d3c2022a")
	default:
		return uiText("ui.494ca78f7374")
	}
}

func (s *setupReviewStore) pruneLocked(now time.Time) {
	for key, state := range s.reviews {
		if now.Sub(state.DraftUpdatedAt) > setupDraftLifetime {
			delete(s.reviews, key)
		}
	}
}

func (s *setupReviewStore) get(app *App, session string) (setupReviewState, bool) {
	now := time.Now()
	key := setupKey(app, session)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneLocked(now)
	state, ok := s.reviews[key]
	return state, ok
}

func (s *setupReviewStore) save(app *App, session string, state setupReviewState) {
	now := time.Now()
	key := setupKey(app, session)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneLocked(now)
	s.reviews[key] = state
}

func (s *setupReviewStore) delete(app *App, session string) {
	key := setupKey(app, session)
	s.mu.Lock()
	delete(s.reviews, key)
	s.mu.Unlock()
}
