package server

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/home-server-project/justvoxel-webui/internal/api"
)

type adminMigrationWorkspaceRecoveryAPI interface {
	adminMigrationWorkspaceAPI
	AdminMigrationRecoveryPlan(ctx context.Context, session string, request api.AdminMigrationRecoveryPlanRequest) (api.AdminMigrationRecoveryPlanResponse, error)
	AdminMigrationRecoveryApply(ctx context.Context, session string, request api.AdminMigrationRecoveryApplyRequest) (api.AdminMigrationApplyResponse, error)
	AdminMigrationImportResolve(ctx context.Context, session string, request api.AdminMigrationImportResolveRequest) (api.AdminMigrationImportResolveResponse, error)
	AdminMigrationRecoveryResolve(ctx context.Context, session string, request api.AdminMigrationRecoveryResolveRequest) (api.AdminMigrationRecoveryResolveResponse, error)
}

type migrationWorkspaceRecoveryTransactionView struct {
	Summary    api.AdminMigrationRecoverySummary
	ModeLabel  string
	PhaseLabel string
}

type migrationWorkspaceRecoveryPageData struct {
	ServerID               string
	Title                  string
	Version                string
	ManagementAPI          string
	CSRF                   string
	Identity               api.SessionInfo
	Discovery              api.AdminMigrationRecoveryDiscoveryResponse
	Transactions           []migrationWorkspaceRecoveryTransactionView
	ImportNeedsAttention   bool
	OperationID            string
	RecoveryNeedsAttention bool
	CanKeepCurrent         bool
	Error                  string
}

type migrationWorkspaceRecoveryReviewPageData struct {
	ServerID      string
	Title         string
	Version       string
	ManagementAPI string
	CSRF          string
	Identity      api.SessionInfo
	Request       api.AdminMigrationRecoveryPlanRequest
	Plan          api.AdminMigrationRecoveryPlanResponse
	ModeLabel     string
	PhaseLabel    string
	Error         string
}

func (a *App) registerAdminMigrationWorkspaceRecoveryPages(mux *http.ServeMux) {
	mux.HandleFunc("GET /workspace/migration/recovery", a.migrationWorkspaceRecoveryPage)
	mux.HandleFunc("POST /workspace/migration/recovery/review", a.migrationWorkspaceRecoveryReview)
	mux.HandleFunc("POST /workspace/migration/recovery/apply", a.migrationWorkspaceRecoveryApply)
	mux.HandleFunc("POST /workspace/migration/recovery/resolve", a.migrationWorkspaceRecoveryResolve)
}

func (a *App) migrationWorkspaceRecoveryRequest(w http.ResponseWriter, r *http.Request, requireCSRF bool) (string, adminMigrationWorkspaceRecoveryAPI, api.SessionInfo, bool, bool) {
	if requireCSRF && !a.validCSRF(r) {
		http.Error(w, "invalid CSRF token", http.StatusForbidden)
		return "", nil, api.SessionInfo{}, false, false
	}
	session, _, identity, ok := a.migrationWorkspaceRequest(w, r)
	if !ok {
		return "", nil, api.SessionInfo{}, false, false
	}
	client, ok := a.api.(adminMigrationWorkspaceRecoveryAPI)
	if !ok {
		http.Error(w, ui(w, "ui.5c0c8938828e"), http.StatusServiceUnavailable)
		return "", nil, api.SessionInfo{}, false, false
	}
	current, err := client.AdminCurrentMigrationOperation(r.Context(), session)
	if err != nil {
		a.handleMigrationWorkspaceRequestError(w, r, err, ui(w, "ui.9ebe3ac32640"))
		return "", nil, api.SessionInfo{}, false, false
	}
	if current.Operation != nil {
		if current.Operation.State == "needs_attention" && (current.Operation.OperationType == "migration_import" || current.Operation.OperationType == "migration_export" || current.Operation.OperationType == "migration_recovery") {
			return session, client, identity, current.Operation.OperationType == "migration_import", true
		}
		if !migrationWorkspaceOperationType(current.Operation.OperationType) {
			http.Error(w, ui(w, "ui.a2ee2f05ca1e"), http.StatusBadGateway)
			return "", nil, api.SessionInfo{}, false, false
		}
		http.Redirect(w, r, migrationSelectedURL(r, "/workspace/migration/progress/"+current.Operation.OperationID), http.StatusSeeOther)
		return "", nil, api.SessionInfo{}, false, false
	}
	return session, client, identity, false, true
}

func (a *App) migrationWorkspaceRecoveryPage(w http.ResponseWriter, r *http.Request) {
	session, client, identity, importNeedsAttention, ok := a.migrationWorkspaceRecoveryRequest(w, r, false)
	if !ok {
		return
	}
	if migrationWorkspaceRedirectBrowser(w, r, "/?workspace=migration&tab=recovery") {
		return
	}
	discovery, err := client.AdminMigrationRecoveryDiscovery(r.Context(), session)
	if err != nil {
		a.handleMigrationWorkspaceRequestError(w, r, err, ui(w, "ui.7e7eff5aca52"))
		return
	}
	operationID := ""
	importNeedsAttention = false
	recoveryNeedsAttention := false
	current, currentErr := client.AdminCurrentMigrationOperation(r.Context(), session)
	if currentErr != nil {
		a.handleMigrationWorkspaceRequestError(w, r, currentErr, ui(w, "ui.9ebe3ac32640"))
		return
	}
	if current.Operation != nil && current.Operation.State == "needs_attention" {
		if current.Operation.OperationType == "migration_import" || current.Operation.OperationType == "migration_export" || current.Operation.OperationType == "migration_recovery" {
			operationID = current.Operation.OperationID
			importNeedsAttention = current.Operation.OperationType == "migration_import"
			recoveryNeedsAttention = current.Operation.OperationType == "migration_recovery" || current.Operation.OperationType == "migration_export"
		}
	}
	a.renderMigrationWorkspaceRecoveryPage(w, http.StatusOK, identity, discovery, importNeedsAttention, recoveryNeedsAttention, operationID, csrfFromRequest(r), "", migrationWorkspaceSelectedID(r))
}

func (a *App) migrationWorkspaceRecoveryReview(w http.ResponseWriter, r *http.Request) {
	session, client, identity, _, ok := a.migrationWorkspaceRecoveryRequest(w, r, true)
	if !ok {
		return
	}
	discovery, err := client.AdminMigrationRecoveryDiscovery(r.Context(), session)
	if err != nil {
		a.handleMigrationWorkspaceRequestError(w, r, err, ui(w, "ui.7e7eff5aca52"))
		return
	}
	request, err := parseMigrationWorkspaceRecoveryForm(r, discovery)
	if err != nil {
		a.renderMigrationWorkspaceRecoveryPage(w, http.StatusBadRequest, identity, discovery, false, false, "", csrfFromRequest(r), err.Error(), migrationWorkspaceSelectedID(r))
		return
	}
	plan, err := client.AdminMigrationRecoveryPlan(r.Context(), session, request)
	if err != nil {
		var responseErr *api.ResponseError
		if errors.As(err, &responseErr) && responseErr.StatusCode == http.StatusBadRequest && plan.SchemaVersion != "" {
			a.renderMigrationWorkspaceRecoveryReview(w, http.StatusBadRequest, identity, request, plan, csrfFromRequest(r), apiMessage(err, ui(w, "ui.fa1afa682313")), migrationWorkspaceSelectedID(r))
			return
		}
		a.handleMigrationWorkspaceRequestError(w, r, err, ui(w, "ui.34fcb6a8932c"))
		return
	}
	a.renderMigrationWorkspaceRecoveryReview(w, http.StatusOK, identity, request, plan, csrfFromRequest(r), "", migrationWorkspaceSelectedID(r))
}

func (a *App) migrationWorkspaceRecoveryApply(w http.ResponseWriter, r *http.Request) {
	session, client, identity, _, ok := a.migrationWorkspaceRecoveryRequest(w, r, true)
	if !ok {
		return
	}
	discovery, err := client.AdminMigrationRecoveryDiscovery(r.Context(), session)
	if err != nil {
		a.handleMigrationWorkspaceRequestError(w, r, err, ui(w, "ui.7e7eff5aca52"))
		return
	}
	request, err := parseMigrationWorkspaceRecoveryForm(r, discovery)
	if err != nil {
		a.renderMigrationWorkspaceRecoveryPage(w, http.StatusBadRequest, identity, discovery, false, false, "", csrfFromRequest(r), err.Error(), migrationWorkspaceSelectedID(r))
		return
	}
	plan, err := client.AdminMigrationRecoveryPlan(r.Context(), session, request)
	if err != nil {
		var responseErr *api.ResponseError
		if errors.As(err, &responseErr) && responseErr.StatusCode == http.StatusBadRequest && plan.SchemaVersion != "" {
			a.renderMigrationWorkspaceRecoveryReview(w, http.StatusBadRequest, identity, request, plan, csrfFromRequest(r), apiMessage(err, ui(w, "ui.fb60580e832d")), migrationWorkspaceSelectedID(r))
			return
		}
		a.handleMigrationWorkspaceRequestError(w, r, err, ui(w, "ui.34fcb6a8932c"))
		return
	}
	if submitted := strings.TrimSpace(r.FormValue("plan_fingerprint")); submitted == "" || submitted != plan.PlanFingerprint {
		a.renderMigrationWorkspaceRecoveryReview(w, http.StatusConflict, identity, request, plan, csrfFromRequest(r), ui(w, "ui.e2c61487aca2"), migrationWorkspaceSelectedID(r))
		return
	}
	if r.FormValue("cleanup_confirmed") != "yes" {
		a.renderMigrationWorkspaceRecoveryReview(w, http.StatusBadRequest, identity, request, plan, csrfFromRequest(r), ui(w, "ui.33aa4339001a"), migrationWorkspaceSelectedID(r))
		return
	}
	if plan.Requirements == nil || !plan.Requirements.FinalizeConfirmationRequired {
		a.renderMigrationWorkspaceRecoveryReview(w, http.StatusBadGateway, identity, request, plan, csrfFromRequest(r), ui(w, "ui.e6d4985788e7"), migrationWorkspaceSelectedID(r))
		return
	}
	result, err := client.AdminMigrationRecoveryApply(r.Context(), session, api.AdminMigrationRecoveryApplyRequest{
		PlanFingerprint:   plan.PlanFingerprint,
		Transaction:       request.Transaction,
		FinalizeConfirmed: true,
	})
	if err != nil {
		a.renderMigrationWorkspaceRecoveryReview(w, http.StatusBadRequest, identity, request, plan, csrfFromRequest(r), apiMessage(err, ui(w, "ui.36eb315ebd2b")), migrationWorkspaceSelectedID(r))
		return
	}
	if !result.OK || result.Operation == nil || result.Operation.OperationType != "migration_recovery" {
		a.renderMigrationWorkspaceRecoveryReview(w, http.StatusBadGateway, identity, request, plan, csrfFromRequest(r), ui(w, "ui.94f54cabc6c4"), migrationWorkspaceSelectedID(r))
		return
	}
	http.Redirect(w, r, migrationSelectedURL(r, "/workspace/migration/progress/"+result.Operation.OperationID), http.StatusSeeOther)
}

func (a *App) migrationWorkspaceRecoveryResolve(w http.ResponseWriter, r *http.Request) {
	session, client, identity, importNeedsAttention, ok := a.migrationWorkspaceRecoveryRequest(w, r, true)
	if !ok {
		return
	}
	discovery, err := client.AdminMigrationRecoveryDiscovery(r.Context(), session)
	if err != nil {
		a.handleMigrationWorkspaceRequestError(w, r, err, ui(w, "ui.7e7eff5aca52"))
		return
	}
	current, err := client.AdminCurrentMigrationOperation(r.Context(), session)
	if err != nil {
		a.handleMigrationWorkspaceRequestError(w, r, err, ui(w, "ui.9ebe3ac32640"))
		return
	}
	if current.Operation == nil || current.Operation.State != "needs_attention" ||
		(current.Operation.OperationType != "migration_import" && current.Operation.OperationType != "migration_recovery" && current.Operation.OperationType != "migration_export") {
		http.Error(w, ui(w, "ui.544589e2ce76"), http.StatusConflict)
		return
	}
	recoveryNeedsAttention := current.Operation.OperationType == "migration_recovery" || current.Operation.OperationType == "migration_export"
	if len(discovery.Transactions) != 0 {
		message := ui(w, "ui.5f88b4671375")
		if recoveryNeedsAttention {
			message = ui(w, "ui.6cf897de299a")
		}
		a.renderMigrationWorkspaceRecoveryPage(w, http.StatusConflict, identity, discovery, importNeedsAttention, recoveryNeedsAttention, "", csrfFromRequest(r), message, migrationWorkspaceSelectedID(r))
		return
	}
	if strings.TrimSpace(r.FormValue("operation_id")) != current.Operation.OperationID || r.FormValue("keep_current_state") != "yes" {
		a.renderMigrationWorkspaceRecoveryPage(w, http.StatusBadRequest, identity, discovery, importNeedsAttention, recoveryNeedsAttention, current.Operation.OperationID, csrfFromRequest(r), ui(w, "ui.3c364622d2cc"), migrationWorkspaceSelectedID(r))
		return
	}
	if recoveryNeedsAttention {
		_, err = client.AdminMigrationRecoveryResolve(r.Context(), session, api.AdminMigrationRecoveryResolveRequest{
			OperationID: current.Operation.OperationID, KeepCurrentState: true,
		})
	} else {
		_, err = client.AdminMigrationImportResolve(r.Context(), session, api.AdminMigrationImportResolveRequest{
			OperationID: current.Operation.OperationID, KeepCurrentState: true,
		})
	}
	if err != nil {
		message := ui(w, "ui.6f3b6ff4590a")
		if recoveryNeedsAttention {
			message = ui(w, "ui.399682ac12ba")
		}
		a.renderMigrationWorkspaceRecoveryPage(w, http.StatusConflict, identity, discovery, importNeedsAttention, recoveryNeedsAttention, current.Operation.OperationID, csrfFromRequest(r), apiMessage(err, message), migrationWorkspaceSelectedID(r))
		return
	}
	http.Redirect(w, r, migrationSelectedURL(r, "/workspace/migration"), http.StatusSeeOther)
}

func parseMigrationWorkspaceRecoveryForm(r *http.Request, discovery api.AdminMigrationRecoveryDiscoveryResponse) (api.AdminMigrationRecoveryPlanRequest, error) {
	if err := r.ParseForm(); err != nil {
		return api.AdminMigrationRecoveryPlanRequest{}, errors.New("invalid Migration Recovery request")
	}
	transaction := strings.TrimSpace(r.FormValue("transaction"))
	for _, summary := range discovery.Transactions {
		if summary.Transaction != transaction {
			continue
		}
		if !summary.Finalizable {
			return api.AdminMigrationRecoveryPlanRequest{}, errors.New(uiText("ui.f276e8ceb36b"))
		}
		return api.AdminMigrationRecoveryPlanRequest{Transaction: transaction}, nil
	}
	return api.AdminMigrationRecoveryPlanRequest{}, errors.New(uiText("ui.6c626bb4963d"))
}

func (a *App) renderMigrationWorkspaceRecoveryPage(w http.ResponseWriter, status int, identity api.SessionInfo, discovery api.AdminMigrationRecoveryDiscoveryResponse, importNeedsAttention, recoveryNeedsAttention bool, operationID, csrf, errorMessage string, serverID string) {
	transactions := make([]migrationWorkspaceRecoveryTransactionView, 0, len(discovery.Transactions))
	for _, summary := range discovery.Transactions {
		transactions = append(transactions, migrationWorkspaceRecoveryTransactionView{
			Summary: summary, ModeLabel: migrationWorkspaceRecoveryModeLabel(summary.Mode), PhaseLabel: migrationWorkspaceRecoveryPhaseLabel(summary.Phase),
		})
	}
	a.renderMigrationWorkspace(w, "migration_workspace_recovery.html", status, migrationWorkspaceRecoveryPageData{
		ServerID:               serverID,
		Title:                  ui(w, "ui.504ca73ba883"),
		Version:                a.config.Version,
		ManagementAPI:          a.config.ManagementAPI,
		CSRF:                   csrf,
		Identity:               identity,
		Discovery:              discovery,
		Transactions:           transactions,
		ImportNeedsAttention:   importNeedsAttention,
		OperationID:            operationID,
		RecoveryNeedsAttention: recoveryNeedsAttention,
		CanKeepCurrent:         (importNeedsAttention || recoveryNeedsAttention) && operationID != "" && len(discovery.Transactions) == 0,
		Error:                  errorMessage,
	})
}

func (a *App) renderMigrationWorkspaceRecoveryReview(w http.ResponseWriter, status int, identity api.SessionInfo, request api.AdminMigrationRecoveryPlanRequest, plan api.AdminMigrationRecoveryPlanResponse, csrf, errorMessage string, serverID string) {
	modeLabel := ui(w, "ui.c7d2c34a5578")
	phaseLabel := ui(w, "ui.b764cdc0eab7")
	if plan.Normalized != nil {
		modeLabel = migrationWorkspaceRecoveryModeLabel(plan.Normalized.Mode)
		phaseLabel = migrationWorkspaceRecoveryPhaseLabel(plan.Normalized.Phase)
	}
	a.renderMigrationWorkspace(w, "migration_workspace_recovery_review.html", status, migrationWorkspaceRecoveryReviewPageData{
		ServerID:      serverID,
		Title:         ui(w, "ui.e1cc448edd18"),
		Version:       a.config.Version,
		ManagementAPI: a.config.ManagementAPI,
		CSRF:          csrf,
		Identity:      identity,
		Request:       request,
		Plan:          plan,
		ModeLabel:     modeLabel,
		PhaseLabel:    phaseLabel,
		Error:         errorMessage,
	})
}

func migrationWorkspaceRecoveryModeLabel(value string) string {
	switch value {
	case "configured":
		return uiText("ui.5be9693f1680")
	case "configured-attention":
		return uiText("ui.da4a608c56b9")
	case "fresh-unconfigured":
		return uiText("ui.ca0a18636910")
	default:
		return migrationWorkspaceFriendlyToken(value)
	}
}

func migrationWorkspaceRecoveryPhaseLabel(value string) string {
	switch value {
	case "rolled-back":
		return uiText("ui.71592236d852")
	case "critical-rollback":
		return uiText("ui.bc75bdf6f78d")
	case "rolled-back-fresh":
		return uiText("ui.5375bbeb9452")
	default:
		return migrationWorkspaceFriendlyToken(value)
	}
}
