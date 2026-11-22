package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
)

type telegramOperation struct {
	Management bool
	Instance   bool
	Method     string
	Path       string
	Handler    http.HandlerFunc
}

// Independent, complete Telegram policy. No gateway level or client-supplied
// Management API path is consulted. No general configuration handler is exposed.
func (s *server) telegramOperations() map[string]telegramOperation {
	return map[string]telegramOperation{
		"status":                  {false, false, "GET", "/v1/status", s.status},
		"instance_status":         {false, true, "GET", "/v1/minecraft/instances/{id}/status", s.instanceRuntimeStatus},
		"instance_details":        {false, true, "GET", "/v1/minecraft/instances/{id}/configuration", s.instanceRuntimeConfiguration},
		"players":                 {false, true, "GET", "/v1/minecraft/instances/{id}/players", s.instanceMaintenanceRead},
		"backups":                 {false, true, "GET", "/v1/minecraft/instances/{id}/backups", s.instanceBackupRead},
		"backup_status":           {false, true, "GET", "/v1/minecraft/instances/{id}/backups/status", s.instanceBackupRead},
		"network_status":          {false, false, "GET", "/v1/minecraft/network", s.minecraftNetworkStatus},
		"version_status":          {false, true, "GET", "/v1/minecraft/instances/{id}/version/status", s.adminVersionStatus},
		"version_update_status":   {false, true, "GET", "/v1/minecraft/instances/{id}/version/update/current", s.adminMinecraftStackUpdateCurrent},
		"container_update_status": {false, true, "GET", "/v1/minecraft/instances/{id}/container/update/current", s.adminMinecraftContainerUpdateCurrent},
		"container_status":        {false, true, "GET", "/v1/minecraft/instances/{id}/container/status", s.adminMinecraftContainerStatus},
		"system_updates":          {false, false, "GET", "/v1/admin/system/updates", s.adminSystemUpdateStatus},
		"system_update_check":     {false, false, "POST", "/v1/admin/system/updates/check", s.adminSystemUpdateCheck},
		"instance_start":          {true, true, "POST", "/v1/minecraft/instances/{id}/start", func(w http.ResponseWriter, r *http.Request) { s.instanceRuntimeAction(w, r, "start") }},
		"instance_stop":           {true, true, "POST", "/v1/minecraft/instances/{id}/stop", func(w http.ResponseWriter, r *http.Request) { s.instanceRuntimeAction(w, r, "stop") }},
		"instance_restart":        {true, true, "POST", "/v1/minecraft/instances/{id}/restart", func(w http.ResponseWriter, r *http.Request) { s.instanceRuntimeAction(w, r, "restart") }},
		"backup_now":              {true, true, "POST", "/v1/minecraft/instances/{id}/backups", s.instanceBackupNow},
		"start_on_boot":           {true, true, "PUT", "/v1/telegram/instances/{id}/boot", s.telegramStartOnBoot},
		"network_change":          {true, false, "PUT", "/v1/telegram/network", s.telegramNetworkChange},
		"version_update":          {true, true, "POST", "/v1/minecraft/instances/{id}/version/update", s.adminMinecraftStackUpdate},
		"container_update":        {true, true, "POST", "/v1/minecraft/instances/{id}/container/update", s.adminMinecraftContainerUpdate},
		"system_download":         {true, false, "POST", "/v1/admin/system/updates", s.adminSystemUpdate},
		"update_reboot":           {true, false, "POST", "/v1/admin/system/update-reboot", s.adminSystemUpdateReboot},
		"reboot":                  {true, false, "POST", "/v1/admin/system/reboot", func(w http.ResponseWriter, r *http.Request) { s.adminSystemAction(w, r, "reboot") }},
	}
}
func telegramManagementOperation(name string) (known, management bool) {
	// Use the same static catalog for execution and callback permission checks.
	if op, ok := (*server)(nil).telegramOperations()[name]; ok {
		return true, op.Management
	}
	switch name {
	case "menu", "appliance_menu", "servers_menu", "backups_menu", "updates_menu", "instance_menu", "network_menu", "os_menu", "notifications", "notifications_toggle":
		return true, false
	case "version_review", "container_review", "reboot_menu":
		return true, true
	}
	return false, false
}
func telegramAllowed(v *telegramState, userID int64, operation string) bool {
	user, approved := v.user(userID)
	known, management := telegramManagementOperation(operation)
	return v.Enabled && approved && known && (!management || user.Permission == telegramManagement)
}
func telegramActor(id int64) session {
	return session{Username: "telegram:" + strconv.FormatInt(id, 10), Role: roleAdministrator, AuthSource: authSourceTelegram}
}

func (s *server) telegramInvoke(ctx context.Context, v *telegramState, userID int64, cb telegramCallback) (int, json.RawMessage) {
	op, found := s.telegramOperations()[cb.Operation]
	actor := telegramActor(userID)
	deny := func() (int, json.RawMessage) {
		_ = s.store.recordAuditEvent(actor, "telegram_denied", "telegram", false, "operation or instance denied")
		return 403, json.RawMessage(`{"ok":false}`)
	}
	if !found || (cb.UserID != 0 && cb.UserID != userID) || !telegramAllowed(v, userID, cb.Operation) || (op.Instance && !instanceIDPattern.MatchString(cb.InstanceID)) || (!op.Instance && cb.InstanceID != "") {
		return deny()
	}
	if op.Management && !cb.Confirmed {
		return deny()
	}
	if !s.telegramResetAllowed() {
		return deny()
	}
	body := cb.Body
	if len(body) == 0 {
		body = json.RawMessage(`{}`)
		if cb.Operation == "instance_stop" || cb.Operation == "instance_restart" {
			body = json.RawMessage(`{"confirm_players":false}`)
		}
	}
	if len(body) > 4096 || !json.Valid(body) {
		return deny()
	}
	if op.Management && s.store.recordAuditEvent(actor, "telegram_attempt_"+cb.Operation, cb.InstanceID, true, "") != nil {
		return 503, json.RawMessage(`{"ok":false}`)
	}
	path := strings.ReplaceAll(op.Path, "{id}", cb.InstanceID)
	request, _ := http.NewRequestWithContext(context.WithValue(ctx, gatewayPrincipalKey{}, actor), op.Method, "http://telegram"+path, bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if op.Instance {
		request.SetPathValue("id", cb.InstanceID)
	}
	if cb.Operation == "version_status" {
		var selection struct {
			Policy  string `json:"policy"`
			Version string `json:"version"`
		}
		if json.Unmarshal(body, &selection) != nil {
			return deny()
		}
		q := request.URL.Query()
		q.Set("policy", selection.Policy)
		q.Set("version", selection.Version)
		request.URL.RawQuery = q.Encode()
	}
	capture := &gatewayResponse{header: make(http.Header), status: 200}
	// Calls enter with telegramMu held. Let primary-administrator controls remain
	// available during the existing long-running operation. Authorization above is
	// the acceptance point; every subsequent request reads current state again.
	s.telegramMu.Unlock()
	op.Handler(capture, request)
	s.telegramMu.Lock()
	latest, stateErr := s.store.telegramRead()
	if stateErr != nil {
		*v = telegramState{}
		return 503, json.RawMessage(`{"ok":false}`)
	}
	*v = latest
	if op.Management {
		var result struct {
			ConfirmationRequired bool `json:"confirmation_required"`
		}
		_ = json.Unmarshal(capture.body.Bytes(), &result)
		if s.store.recordAuditEvent(actor, "telegram_"+cb.Operation, cb.InstanceID, capture.status < 400 && !result.ConfirmationRequired, "") != nil {
			return 503, json.RawMessage(`{"ok":false}`)
		}
	}
	if capture.overflow || !json.Valid(capture.body.Bytes()) {
		return 502, json.RawMessage(`{"ok":false}`)
	}
	return capture.status, append(json.RawMessage{}, capture.body.Bytes()...)
}

// Only the existing boot preference changes, under the usual mutation locks and
// runtime publication/recovery checks. No stale whole-settings round trip.
func (s *server) telegramStartOnBoot(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdministrator(w, r); !ok {
		return
	}
	id, ok := instanceRequestID(w, r)
	if !ok {
		return
	}
	var input struct {
		Enabled *bool `json:"enabled"`
	}
	if !decodeTelegramJSON(w, r, &input) {
		return
	}
	if input.Enabled == nil {
		writeError(w, 400, "Choose Start on Boot.")
		return
	}
	s.setInstanceStartOnBoot(w, r, id, *input.Enabled, nil)
}

// Shared narrow mutation; expected is optional for existing Telegram behavior.
func (s *server) setInstanceStartOnBoot(w http.ResponseWriter, r *http.Request, id string, enabled bool, expected *bool) {
	s.withInstanceMutation(w, r, id, func(ctx context.Context) error {
		old, err := instanceRuntimes.readRecord(id)
		if err != nil {
			return err
		}
		if expected != nil && old.Runtime.StartOnBoot != *expected {
			return instanceRuntimeFailure("Start on Boot changed. Review it again.")
		}
		if err = networkProtectBootPreference(old, enabled); err != nil {
			return err
		}
		snapshot, _, _, err := instanceRuntimes.snapshot(ctx)
		if err != nil {
			return err
		}
		records, err := instanceRuntimes.records()
		if err != nil {
			return err
		}
		next := old
		next.Runtime.StartOnBoot = enabled
		return instanceRuntimes.apply(ctx, old, instanceReplaceRecord(records, next), snapshot)
	})
}

func (s *server) telegramNetworkChange(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdministrator(w, r); !ok {
		return
	}
	var input struct {
		Enabled  *bool     `json:"enabled,omitempty"`
		Members  *[]string `json:"members,omitempty"`
		Expected string    `json:"expected"`
	}
	if !decodeTelegramJSON(w, r, &input) {
		return
	}
	if (input.Enabled == nil) == (input.Members == nil) {
		writeError(w, 400, "Choose enabled or membership.")
		return
	}
	if err := s.changeMinecraftNetwork(r.Context(), input.Enabled, input.Members, input.Expected, false); err != nil {
		instanceRuntimeError(w, err, "")
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

var errMinecraftNetworkReviewChanged = instanceRuntimeFailure("Minecraft Network changed; review the current menu again.")

// Both interaction surfaces use the same transaction and stopped-member checks.
func (s *server) changeMinecraftNetwork(ctx context.Context, enabled *bool, members *[]string, expected string, requireDisabledMembership bool) error {
	release, unlock, err := s.lockPhasedInstanceRuntime()
	if err != nil {
		return err
	}
	err = func() error {
		n, err := networkRead(instanceRuntimes)
		if err != nil {
			return err
		}
		if expected != minecraftNetworkFingerprint(n.Config) {
			return errMinecraftNetworkReviewChanged
		}
		if members != nil && requireDisabledMembership && n.Config.Enabled {
			return instanceRuntimeFailure("Disable Minecraft Network, then stop affected servers before changing membership.")
		}
		c := n.Config
		if enabled != nil {
			c.Enabled = *enabled
		}
		if members != nil {
			c.Members = append([]string{}, (*members)...)
		}
		if err = networkValidateConfig(c); err != nil {
			return err
		}
		// Telegram requires deliberate stopped membership changes. The normal Network
		// transaction still owns compatibility, reservations, publication and recovery.
		affected := map[string]bool{}
		for _, id := range n.Config.Members {
			affected[id] = true
		}
		for _, id := range c.Members {
			affected[id] = true
		}
		for id := range affected {
			changed := members != nil
			if !changed {
				continue
			}
			state, err := instanceRuntimes.observe(ctx, id)
			if err != nil || state.State != "stopped" {
				return instanceRuntimeFailure("Disable Minecraft Network, then stop affected servers before changing membership.")
			}
		}
		return networkApplyWithMaintenanceRelease(ctx, instanceRuntimes, c, release)
	}()
	return errors.Join(err, unlock())
}

func telegramNetworkFingerprint(config minecraftNetworkConfig) string {
	return minecraftNetworkFingerprint(config)
}
