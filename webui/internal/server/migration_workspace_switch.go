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
		{"Plugins / Paper family", simpleSoftware("plugins")},
		{"Mod loaders", simpleSoftware("mods")},
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
		writeMinecraftWorkspaceError(w, 503, "Switch is unavailable.")
		return
	}
	action := r.PathValue("action")
	if r.Method == http.MethodGet {
		if action == "plan" {
			plan, err := client.MinecraftInstanceSwitchPlan(r.Context(), session, selected.ID, r.URL.Query().Get("target"))
			if err != nil {
				a.writeMinecraftWorkspaceAPIError(w, err, "Switch compatibility could not be verified.")
				return
			}
			if plan.InstanceID != selected.ID {
				writeMinecraftWorkspaceError(w, 503, "Switch server identity could not be verified.")
				return
			}
			writeMinecraftWorkspaceJSON(w, 200, plan)
			return
		}
		if action == "operation" {
			operation, err := client.MinecraftInstanceSwitchOperation(r.Context(), session, selected.ID, r.URL.Query().Get("operation_id"))
			if err != nil {
				a.writeMinecraftWorkspaceAPIError(w, err, "Switch progress is unavailable.")
				return
			}
			if operation.Operation != nil && (operation.Operation.InstanceID != selected.ID || operation.Operation.OperationType != "minecraft_switch") {
				writeMinecraftWorkspaceError(w, 503, "Switch server identity could not be verified.")
				return
			}
			writeMinecraftWorkspaceJSON(w, 200, operation)
			return
		}
		writeMinecraftWorkspaceError(w, 404, "Switch action not found.")
		return
	}
	for key, values := range r.URL.Query() {
		if key != "server" || len(values) != 1 {
			writeMinecraftWorkspaceError(w, 400, "Invalid Switch request.")
			return
		}
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if r.ParseForm() != nil || !a.validCSRF(r) {
		writeMinecraftWorkspaceError(w, 403, "Invalid Switch confirmation.")
		return
	}
	for key, values := range r.PostForm {
		if !slices.Contains([]string{"csrf", "target", "action", "plan_fingerprint", "confirm", "operation_id"}, key) || len(values) != 1 {
			writeMinecraftWorkspaceError(w, 400, "Invalid Switch request.")
			return
		}
	}
	if r.FormValue("confirm") != "yes" {
		writeMinecraftWorkspaceError(w, 400, "Confirm Switch first.")
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
		writeMinecraftWorkspaceError(w, 404, "Switch action not found.")
		return
	}
	if err != nil {
		a.writeMinecraftWorkspaceAPIError(w, err, "Switch could not complete. Refresh to reconnect to its operation.")
		return
	}
	if operation.Operation == nil || operation.Operation.InstanceID != selected.ID || operation.Operation.OperationType != "minecraft_switch" {
		writeMinecraftWorkspaceError(w, 503, "Switch operation identity could not be verified.")
		return
	}
	writeMinecraftWorkspaceJSON(w, 202, operation)
}
