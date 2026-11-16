package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/home-server-project/justvoxel-webui/internal/api"
)

type selectedMinecraftAPI interface {
	MinecraftInstanceConfiguration(context.Context, string, string) (api.InstanceRuntimeSettings, error)
	SetMinecraftInstanceConfiguration(context.Context, string, string, api.InstanceRuntimeSettings) error
	MinecraftInstanceStatus(context.Context, string, string) (api.MinecraftInstanceStatus, error)
	MinecraftInstancePlayers(context.Context, string, string) (api.InstancePlayers, error)
	MinecraftInstanceLogs(context.Context, string, string) (string, error)
	RenameMinecraftInstance(context.Context, string, string, string) error
	DeleteMinecraftInstance(context.Context, string, string, string) error
}
type selectedSettingsLifecycleAPI interface {
	MinecraftInstanceActionConfirmed(context.Context, string, string, string, bool) (api.MinecraftActionResponse, error)
}

type selectedServerContext struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Selection resolves only registered immutable identities. Missing context is
// accepted only when the actual inventory contains exactly one server.
func (a *App) selectedMinecraft(w http.ResponseWriter, r *http.Request, session string) (selectedServerContext, bool) {
	values, exists := r.URL.Query()["server"]
	id := ""
	if exists {
		if len(values) != 1 || !addServerIDPattern.MatchString(values[0]) {
			writeMinecraftWorkspaceError(w, 400, ui(w, "ui.abda10a5fa44"))
			return selectedServerContext{}, false
		}
		id = values[0]
	}
	instances, err := a.independentServerInventory(r.Context(), session)
	if err != nil {
		writeMinecraftWorkspaceError(w, 503, ui(w, "ui.23112b254b17"))
		return selectedServerContext{}, false
	}
	if !exists {
		if len(instances) == 0 {
			writeMinecraftWorkspaceError(w, 409, ui(w, "ui.f30501d9bdd9"))
			return selectedServerContext{}, false
		}
		if len(instances) != 1 {
			writeMinecraftWorkspaceError(w, 409, ui(w, "ui.01fabcf55f34"))
			return selectedServerContext{}, false
		}
		id = instances[0].ID
	}
	for _, instance := range instances {
		if instance.ID == id {
			return selectedServerContext{ID: id, Name: instance.DisplayName}, true
		}
	}
	writeMinecraftWorkspaceError(w, 404, ui(w, "ui.eca706fcd65b"))
	return selectedServerContext{}, false
}
func (a *App) minecraftSelection(w http.ResponseWriter, r *http.Request) {
	session, _, ok := a.minecraftWorkspaceIdentity(w, r, "administrator", "operator", "viewer")
	if !ok {
		return
	}
	instances, err := a.independentServerInventory(r.Context(), session)
	if err != nil {
		a.writeMinecraftWorkspaceAPIError(w, err, ui(w, "ui.b681d9a7fc45"))
		return
	}
	selected := selectedServerContext{}
	if _, explicit := r.URL.Query()["server"]; explicit || (len(instances) == 1 && r.URL.Query().Get("selection") != "none") {
		var valid bool
		selected, valid = a.selectedMinecraft(w, r, session)
		if !valid {
			return
		}
	}
	type selectionChoice struct {
		selectedServerContext
		Software     string `json:"software,omitempty"`
		SoftwareIcon string `json:"software_icon,omitempty"`
	}
	choices := []selectionChoice{}
	client, canReadConfiguration := a.api.(interface {
		MinecraftInstanceConfiguration(context.Context, string, string) (api.InstanceRuntimeSettings, error)
	})
	for _, i := range instances {
		choice := selectionChoice{selectedServerContext: selectedServerContext{ID: i.ID, Name: i.DisplayName}}
		if canReadConfiguration {
			if config, err := client.MinecraftInstanceConfiguration(r.Context(), session, i.ID); err == nil {
				choice.SoftwareIcon = serverSoftwareIcon(config.Software)
				if choice.SoftwareIcon != "" {
					choice.Software = serverSoftware(config.Software)
				}
			}
		}
		choices = append(choices, choice)
	}
	writeMinecraftWorkspaceJSON(w, 200, map[string]any{"selected": selected, "servers": choices})
}

// Return true when selected handling consumed the request, including refusals.
func (a *App) selectedMinecraftWorkspace(w http.ResponseWriter, r *http.Request, session string, apply bool) bool {
	selected, ok := a.selectedMinecraft(w, r, session)
	if !ok {
		return true
	}
	client, ok := a.api.(selectedMinecraftAPI)
	if !ok {
		writeMinecraftWorkspaceError(w, 503, ui(w, "ui.9ffd18fb5953"))
		return true
	}
	path := r.URL.Path
	if strings.Contains(path, "/whitelist") {
		a.selectedWhitelist(w, r, session, selected, client)
		return true
	}
	if strings.HasSuffix(path, "/logs") {
		logs, err := client.MinecraftInstanceLogs(r.Context(), session, selected.ID)
		if err != nil {
			a.writeMinecraftWorkspaceAPIError(w, err, ui(w, "ui.e1049525c721"))
		} else {
			writeMinecraftWorkspaceJSON(w, 200, minecraftWorkspaceLogsResponse{OK: true, Lines: strings.Split(logs, "\n")})
		}
		return true
	}
	current, err := client.MinecraftInstanceConfiguration(r.Context(), session, selected.ID)
	if err != nil {
		a.writeMinecraftWorkspaceAPIError(w, err, ui(w, "ui.bbd7c341c39d"))
		return true
	}
	if r.Method == http.MethodGet {
		// Reuse exactly the existing controls' public presentation contract.
		raw, _ := json.Marshal(current)
		var minecraft map[string]any
		_ = json.Unmarshal(raw, &minecraft)
		if connections, ok := a.api.(interface {
			MinecraftInstanceConnection(context.Context, string, string) (api.MinecraftConnection, error)
		}); ok {
			if connection, err := connections.MinecraftInstanceConnection(r.Context(), session, selected.ID); err == nil && connection.ID == selected.ID {
				minecraft["network_member"] = connection.NetworkMember
			}
		}
		minecraft["server_type"] = current.Software
		minecraft["version_mode"] = current.VersionPolicy
		if current.VersionPolicy == "" {
			minecraft["version_mode"] = "pinned"
			if current.Version == "LATEST" {
				minecraft["version_mode"] = "latest"
			}
		}
		defaults := map[string]any{}
		if discovery, ok := a.api.(interface {
			AdminSetupDefaults(context.Context, string) (api.AdminSetupDefaults, error)
		}); ok {
			if v, e := discovery.AdminSetupDefaults(r.Context(), session); e == nil {
				defaults["system_memory_mib"] = v.SystemMemoryMiB
				defaults["system_reserve_minimum_mib"] = v.SystemReserveMinimumMiB
				defaults["system_reserve_recommended_mib"] = v.SystemReserveRecommendedMiB
			}
		}
		writeMinecraftWorkspaceJSON(w, 200, map[string]any{"ok": true, "configured": true, "minecraft": minecraft, "defaults": defaults})
		return true
	}
	if r.FormValue("tab") == "version" {
		a.selectedVersionReview(w, r, session, selected, current, apply)
		return true
	}
	next, err := selectedSettingsOverlay(r, current)
	if err != nil {
		writeMinecraftWorkspaceError(w, 400, err.Error())
		return true
	}
	before, _ := json.Marshal(current)
	after, _ := json.Marshal(next)
	digest := sha256.Sum256(append(append([]byte(selected.ID+":"), before...), after...))
	review := hex.EncodeToString(digest[:])
	if apply && r.FormValue("review") != review {
		writeMinecraftWorkspaceError(w, 409, ui(w, "ui.39f443367304"))
		return true
	}
	// Compare the entire definition so only Start on Boot avoids a restart.
	bootComparison := current
	bootComparison.StartOnBoot = next.StartOnBoot
	bootOnly := current.StartOnBoot != next.StartOnBoot && reflect.DeepEqual(bootComparison, next)
	var oldFields, newFields map[string]any
	_ = json.Unmarshal(before, &oldFields)
	_ = json.Unmarshal(after, &newFields)
	changes := []api.AdminConfigurationChange{}
	for _, field := range []string{"java_memory", "container_memory", "motd", "max_players", "game_mode", "java_port", "bedrock_enabled", "bedrock_port", "start_on_boot"} {
		if fmt.Sprint(oldFields[field]) != fmt.Sprint(newFields[field]) {
			changes = append(changes, api.AdminConfigurationChange{Field: field, Label: strings.ReplaceAll(field, "_", " "), Before: fmt.Sprint(oldFields[field]), After: fmt.Sprint(newFields[field])})
		}
	}
	// Runtime state deliberately stays outside the configuration fingerprint.
	// A server stopped after Review must never be started by Apply.
	state, stateErr := client.MinecraftInstanceStatus(r.Context(), session, selected.ID)
	plan := map[string]any{"ok": true, "applied": false, "review": review, "changes": changes,
		"server_name": selected.Name, "runtime_state": state.State, "reviewed_settings": true}
	warnings := []string{ui(w, "ui.f18fa5927b35")}
	blocked := stateErr != nil || state.ID != selected.ID || (state.State != "stopped" && state.State != "running")
	if blocked && r.FormValue("tab") == "overview" {
		writeMinecraftWorkspaceError(w, 409, ui(w, "ui.c4a78abd3102"))
		return true
	}
	restart := !bootOnly && state.State == "running" && len(changes) > 0
	confirm := r.FormValue("confirm_players") == "yes"
	if bootOnly {
		warnings = []string{ui(w, "ui.db2c8b15eebb")}
	} else if restart && !blocked {
		players, playerErr := client.MinecraftInstancePlayers(r.Context(), session, selected.ID)
		if playerErr != nil || players.ID != selected.ID || players.State != "online" || players.Online < 0 {
			blocked = true
			warnings = []string{selected.Name + ui(w, "ui.f6da1ffb6f13")}
		} else {
			plan["online"] = players.Online
			plan["confirmation_required"] = players.Online > 0
			warnings = []string{fmt.Sprintf(ui(w, "ui.56ef3335a65a"), selected.Name, players.Online, selected.Name)}
		}
	}
	if stateErr != nil || state.ID != selected.ID || (state.State != "stopped" && state.State != "running") {
		warnings = []string{ui(w, "ui.40f689b1f685")}
	}
	plan["warnings"], plan["apply_blocked"], plan["restart_required"] = warnings, blocked, restart
	if !apply || blocked || (plan["confirmation_required"] == true && !confirm) {
		writeMinecraftWorkspaceJSON(w, 200, plan)
		return true
	}
	var lifecycle selectedSettingsLifecycleAPI
	if restart {
		lifecycle, ok = a.api.(selectedSettingsLifecycleAPI)
		if !ok {
			writeMinecraftWorkspaceError(w, 503, ui(w, "ui.91d30945c2b6"))
			return true
		}
		// The normal lifecycle rechecks players under its mutation lock and uses
		// the established player-aware graceful stop, including warning countdown.
		stopped, stopErr := lifecycle.MinecraftInstanceActionConfirmed(r.Context(), session, selected.ID, "stop", confirm)
		if stopErr != nil {
			a.writeMinecraftWorkspaceAPIError(w, stopErr, ui(w, "ui.451d978a6d73"))
			return true
		}
		if stopped.ConfirmationRequired {
			plan["confirmation_required"], plan["online"] = true, stopped.Online
			plan["warnings"] = []string{fmt.Sprintf(ui(w, "ui.744ef70c54e1"), selected.Name, stopped.Online)}
			writeMinecraftWorkspaceJSON(w, 200, plan)
			return true
		}
		state, err = client.MinecraftInstanceStatus(r.Context(), session, selected.ID)
		if !stopped.OK || err != nil || state.ID != selected.ID || state.State != "stopped" {
			writeMinecraftWorkspaceError(w, 409, ui(w, "ui.396b953359d0"))
			return true
		}
	}
	if err = client.SetMinecraftInstanceConfiguration(r.Context(), session, selected.ID, next); err != nil {
		// The existing configuration operation owns rollback/recovery. Never
		// start a server whose configuration application failed.
		a.writeMinecraftWorkspaceAPIError(w, err, ui(w, "ui.21f3e4d97eaf"))
		return true
	}
	if restart {
		started, startErr := lifecycle.MinecraftInstanceActionConfirmed(r.Context(), session, selected.ID, "start", false)
		if startErr != nil {
			a.writeMinecraftWorkspaceAPIError(w, startErr, ui(w, "ui.b035da86f445"))
			return true
		}
		state, err = client.MinecraftInstanceStatus(r.Context(), session, selected.ID)
		if !started.OK || err != nil || state.ID != selected.ID || state.State != "running" {
			writeMinecraftWorkspaceError(w, 409, ui(w, "ui.e04e61e4bded"))
			return true
		}
	}
	plan["applied"], plan["message"] = true, ui(w, "ui.5620878f3da7")
	if restart {
		plan["restarted"] = true
	}
	writeMinecraftWorkspaceJSON(w, 200, plan)
	return true
}
func selectedSettingsOverlay(r *http.Request, current api.InstanceRuntimeSettings) (api.InstanceRuntimeSettings, error) {
	// This parser already limits sections and validates user-friendly safe fields.
	if r.FormValue("tab") == "overview" {
		v, e := strconv.ParseBool(r.FormValue("start_on_boot"))
		if e != nil {
			return current, errors.New(uiText("ui.e47e26c6a03c"))
		}
		current.StartOnBoot = v
		return current, nil
	}
	request, err := minecraftWorkspaceConfigurationRequest(r, api.AdminConfigurationDiscovery{})
	if err != nil {
		return current, err
	}
	switch r.FormValue("tab") {
	case "memory":
		current.JavaMemory = request.JavaMemory
		current.ContainerMemory = request.ContainerMemory
	case "gameplay":
		current.MOTD = request.MOTD
		current.MaxPlayers = request.MaxPlayers
		current.GameMode = request.GameMode
	case "crossplay":
		current.JavaPort = request.JavaPort
		current.BedrockEnabled = request.BedrockEnabled
		current.BedrockPort = request.BedrockPort
	default:
		return current, errors.New(uiText("ui.40feea1fd913"))
	}
	return current, nil
}
func (a *App) selectedMinecraftSnapshot(w http.ResponseWriter, r *http.Request) {
	session, _, ok := a.minecraftWorkspaceIdentity(w, r, "administrator", "operator", "viewer")
	if !ok {
		return
	}
	selected, ok := a.selectedMinecraft(w, r, session)
	if !ok {
		return
	}
	client, ok := a.api.(selectedMinecraftAPI)
	if !ok {
		writeMinecraftWorkspaceError(w, 503, ui(w, "ui.a95d80be4cce"))
		return
	}
	c, err := client.MinecraftInstanceConfiguration(r.Context(), session, selected.ID)
	if err != nil {
		a.writeMinecraftWorkspaceAPIError(w, err, ui(w, "ui.e7b087268b8e"))
		return
	}
	state, err := client.MinecraftInstanceStatus(r.Context(), session, selected.ID)
	if err != nil || state.ID != selected.ID {
		a.writeMinecraftWorkspaceAPIError(w, err, ui(w, "ui.a95d80be4cce"))
		return
	}
	players, err := client.MinecraftInstancePlayers(r.Context(), session, selected.ID)
	if err != nil || players.ID != selected.ID {
		players.State = "unknown"
	}
	playerState := "unavailable"
	var online any = ui(w, "ui.ca1844969742")
	if players.State == "online" {
		playerState, online = "online", players.Online
	} else if players.State == "offline" {
		playerState, online = "stopped", 0
	}
	writeMinecraftWorkspaceJSON(w, 200, map[string]any{"status": map[string]any{"minecraft": map[string]any{"configured": true, "state": state.State, "version": c.Version, "max_players": c.MaxPlayers, "start_on_boot": c.StartOnBoot}}, "players": map[string]any{"state": playerState, "online": online, "max": c.MaxPlayers}})
}
func (a *App) selectedMinecraftPresentation(w http.ResponseWriter, r *http.Request) {
	if !a.validCSRF(r) {
		writeMinecraftWorkspaceError(w, 403, ui(w, "ui.74bccc44f472"))
		return
	}
	session, _, ok := a.minecraftWorkspaceIdentity(w, r, "administrator")
	if !ok {
		return
	}
	selected, ok := a.selectedMinecraft(w, r, session)
	if !ok {
		return
	}
	client, ok := a.api.(selectedMinecraftAPI)
	if !ok {
		writeMinecraftWorkspaceError(w, 503, ui(w, "ui.5d561696dd33"))
		return
	}
	var err error
	if strings.HasSuffix(r.URL.Path, "/rename") {
		err = client.RenameMinecraftInstance(r.Context(), session, selected.ID, r.FormValue("display_name"))
	} else {
		err = client.DeleteMinecraftInstance(r.Context(), session, selected.ID, r.FormValue("confirm_name"))
	}
	if err != nil {
		a.writeMinecraftWorkspaceAPIError(w, err, ui(w, "ui.d7769f5853f3"))
		return
	}
	writeMinecraftWorkspaceJSON(w, 200, map[string]any{"ok": true, "redirect": "/minecraft/servers?deleted=yes"})
}

func (a *App) selectedWhitelist(w http.ResponseWriter, r *http.Request, session string, selected selectedServerContext, settings selectedMinecraftAPI) {
	client, ok := a.api.(interface {
		MinecraftInstanceWhitelist(context.Context, string, string, any) (api.TextOutputResponse, error)
	})
	if !ok {
		writeMinecraftWorkspaceError(w, 503, ui(w, "ui.8249e3a5a00e"))
		return
	}
	if strings.HasSuffix(r.URL.Path, "/state") {
		enabled, err := strconv.ParseBool(r.FormValue("enabled"))
		if err != nil {
			writeMinecraftWorkspaceError(w, 400, ui(w, "ui.ede858c00b87"))
			return
		}
		current, err := settings.MinecraftInstanceConfiguration(r.Context(), session, selected.ID)
		if err != nil {
			a.writeMinecraftWorkspaceAPIError(w, err, ui(w, "ui.8249e3a5a00e"))
			return
		}
		current.Whitelist = enabled
		if err = settings.SetMinecraftInstanceConfiguration(r.Context(), session, selected.ID, current); err != nil {
			a.writeMinecraftWorkspaceAPIError(w, err, ui(w, "ui.fa3b35c1d480"))
			return
		}
		writeMinecraftWorkspaceJSON(w, 200, minecraftWorkspaceTextResponse{OK: true, Message: ui(w, "ui.f3ac5dffc52f")})
		return
	}
	var request any
	if r.Method != http.MethodGet {
		request = map[string]string{"platform": r.FormValue("platform"), "action": r.FormValue("action"), "name": r.FormValue("name")}
	}
	result, err := client.MinecraftInstanceWhitelist(r.Context(), session, selected.ID, request)
	if err != nil {
		a.writeMinecraftWorkspaceAPIError(w, err, ui(w, "ui.8249e3a5a00e"))
		return
	}
	if r.Method != http.MethodGet {
		writeMinecraftWorkspaceJSON(w, 200, minecraftWorkspaceTextResponse{OK: true, Output: result.Output, Message: ui(w, "ui.f3ac5dffc52f")})
		return
	}
	entries, parsed := parseMinecraftWorkspaceWhitelist(result.Output)
	_, identity, ok := a.minecraftWorkspaceIdentity(w, r, "administrator", "operator")
	if !ok {
		return
	}
	writeMinecraftWorkspaceJSON(w, 200, minecraftWorkspaceWhitelistResponse{OK: true, Entries: entries, Parsed: parsed, Output: result.Output, WhitelistEnabled: result.WhitelistEnabled, CanToggle: identity.Role == "administrator"})
}

func (a *App) selectedMinecraftConsole(w http.ResponseWriter, r *http.Request) {
	if !a.validCSRF(r) {
		writeMinecraftWorkspaceError(w, http.StatusForbidden, ui(w, "ui.74bccc44f472"))
		return
	}
	session, _, ok := a.minecraftWorkspaceIdentity(w, r, "administrator")
	if !ok {
		return
	}
	// RCON never infers a target, even for a single-server inventory.
	if len(r.URL.Query()["server"]) != 1 || r.FormValue("server_id") != r.URL.Query().Get("server") {
		writeMinecraftWorkspaceError(w, http.StatusConflict, ui(w, "ui.2abb7427913d"))
		return
	}
	selected, ok := a.selectedMinecraft(w, r, session)
	if !ok {
		return
	}
	client, ok := a.api.(interface {
		MinecraftInstanceConsole(context.Context, string, string, api.InstanceConsoleRequest) (api.InstanceConsoleResponse, error)
	})
	if !ok {
		writeMinecraftWorkspaceError(w, 503, "RCON is unavailable.")
		return
	}
	result, err := client.MinecraftInstanceConsole(r.Context(), session, selected.ID, api.InstanceConsoleRequest{
		Command: r.FormValue("command"), Discover: r.FormValue("discover") == "yes",
	})
	if err != nil {
		a.writeMinecraftWorkspaceAPIError(w, err, "RCON is unavailable.")
		return
	}
	if result.ID != selected.ID || !result.OK {
		writeMinecraftWorkspaceError(w, 502, ui(w, "ui.1e57161e8be3"))
		return
	}
	writeMinecraftWorkspaceJSON(w, 200, result)
}
