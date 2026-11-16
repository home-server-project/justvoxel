package server

import (
	"context"
	"net/http"
	"slices"

	"github.com/home-server-project/justvoxel-webui/internal/api"
)

type migrationSwitchAPI interface {
	MinecraftInstanceSwitchPlan(context.Context, string, string, string) (api.MinecraftSwitchPlan, error)
	MinecraftInstanceSwitchApply(context.Context, string, string, api.MinecraftSwitchRequest) (api.MinecraftSwitchOperation, error)
	MinecraftInstanceSwitchOperation(context.Context, string, string, string) (api.MinecraftSwitchOperation, error)
	MinecraftInstanceSwitchRecover(context.Context, string, string, string) (api.MinecraftSwitchOperation, error)
}

func (a *App) migrationSwitchPage(w http.ResponseWriter, r *http.Request) {
	session, _, ok := a.minecraftWorkspaceIdentity(w, r, "administrator")
	if !ok {
		return
	}
	selected, ok := a.selectedMinecraft(w, r, session)
	if !ok {
		return
	}
	if migrationWorkspaceRedirectBrowser(w, r, "/?workspace=migration&tab=switch") {
		return
	}
	// Share Setup's catalogue, including its order, copy and artwork identities.
	groups := []struct {
		Name    string
		Choices []setupSoftwareChoice
	}{
		{ui(w, "ui.53e5b3cbd2f4"), simpleSoftware("plugins")},
		{ui(w, "ui.2e43190a6798"), simpleSoftware("mods")},
		{"Original", simpleSoftware("original")},
	}
	a.renderMigrationWorkspace(w, "migration_workspace_switch.html", 200, struct {
		ServerID string
		CSRF     string
		Groups   any
	}{selected.ID, csrfFromRequest(r), groups})
}
func (a *App) migrationSwitchAPI(w http.ResponseWriter, r *http.Request) {
	session, _, ok := a.minecraftWorkspaceIdentity(w, r, "administrator")
	if !ok {
		return
	}
	selected, ok := a.selectedMinecraft(w, r, session)
	if !ok {
		return
	}
	client, ok := a.api.(migrationSwitchAPI)
	if !ok {
		writeMinecraftWorkspaceError(w, 503, ui(w, "ui.1eb6b959b7ce"))
		return
	}
	action := r.PathValue("action")
	if r.Method == http.MethodGet {
		if action == "plan" {
			plan, err := client.MinecraftInstanceSwitchPlan(r.Context(), session, selected.ID, r.URL.Query().Get("target"))
			if err != nil {
				a.writeMinecraftWorkspaceAPIError(w, err, ui(w, "ui.a99ea29239db"))
				return
			}
			if plan.InstanceID != selected.ID {
				writeMinecraftWorkspaceError(w, 503, ui(w, "ui.f6a40c5c6742"))
				return
			}
			writeMinecraftWorkspaceJSON(w, 200, plan)
			return
		}
		if action == "operation" {
			operation, err := client.MinecraftInstanceSwitchOperation(r.Context(), session, selected.ID, r.URL.Query().Get("operation_id"))
			if err != nil {
				a.writeMinecraftWorkspaceAPIError(w, err, ui(w, "ui.711570b7f679"))
				return
			}
			if operation.Operation != nil && (operation.Operation.InstanceID != selected.ID || operation.Operation.OperationType != "minecraft_switch") {
				writeMinecraftWorkspaceError(w, 503, ui(w, "ui.f6a40c5c6742"))
				return
			}
			writeMinecraftWorkspaceJSON(w, 200, operation)
			return
		}
		writeMinecraftWorkspaceError(w, 404, ui(w, "ui.097074c74d5b"))
		return
	}
	for key, values := range r.URL.Query() {
		if key != "server" || len(values) != 1 {
			writeMinecraftWorkspaceError(w, 400, ui(w, "ui.35e44579c835"))
			return
		}
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if r.ParseForm() != nil || !a.validCSRF(r) {
		writeMinecraftWorkspaceError(w, 403, ui(w, "ui.ebdf0875cf9f"))
		return
	}
	for key, values := range r.PostForm {
		if !slices.Contains([]string{"csrf", "target", "action", "plan_fingerprint", "confirm", "operation_id"}, key) || len(values) != 1 {
			writeMinecraftWorkspaceError(w, 400, ui(w, "ui.35e44579c835"))
			return
		}
	}
	if r.FormValue("confirm") != "yes" {
		writeMinecraftWorkspaceError(w, 400, ui(w, "ui.51514d844584"))
		return
	}
	var operation api.MinecraftSwitchOperation
	var err error
	switch action {
	case "apply":
		operation, err = client.MinecraftInstanceSwitchApply(r.Context(), session, selected.ID, api.MinecraftSwitchRequest{Target: r.FormValue("target"), Action: r.FormValue("action"), Fingerprint: r.FormValue("plan_fingerprint"), Confirm: true})
	case "recover":
		operation, err = client.MinecraftInstanceSwitchRecover(r.Context(), session, selected.ID, r.FormValue("operation_id"))
	default:
		writeMinecraftWorkspaceError(w, 404, ui(w, "ui.097074c74d5b"))
		return
	}
	if err != nil {
		a.writeMinecraftWorkspaceAPIError(w, err, ui(w, "ui.93325fefd879"))
		return
	}
	if operation.Operation == nil || operation.Operation.InstanceID != selected.ID || operation.Operation.OperationType != "minecraft_switch" {
		writeMinecraftWorkspaceError(w, 503, ui(w, "ui.438b4085ebd4"))
		return
	}
	writeMinecraftWorkspaceJSON(w, 202, operation)
}
