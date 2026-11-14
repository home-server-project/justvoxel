package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type telegramButton struct {
	Text string `json:"text"`
	Data string `json:"callback_data"`
}
type telegramChoice struct {
	Label string
	Call  telegramCallback
}

func telegramBody(value any) json.RawMessage { raw, _ := json.Marshal(value); return raw }
func (s *server) telegramSend(ctx context.Context, v *telegramState, id int64, text string, choices []telegramChoice) {
	if _, approved := v.user(id); !approved || !v.Enabled {
		return
	}
	if len(text) > 4000 {
		text = strings.ToValidUTF8(text[:4000], "")
	}
	input := map[string]any{"chat_id": id, "text": text, "protect_content": true}
	rows := [][]telegramButton{}
	for _, choice := range choices {
		if !telegramAllowed(v, id, choice.Call.Operation) {
			continue
		}
		cb := choice.Call
		cb.UserID = id
		key, err := v.issueCallback(cb, s.store.now())
		if err != nil {
			continue
		}
		rows = append(rows, []telegramButton{{Text: choice.Label, Data: key}})
	}
	if len(rows) > 0 {
		if s.store.telegramSave(*v, nil, "") != nil {
			return
		}
		input["reply_markup"] = map[string]any{"inline_keyboard": rows}
	}
	sendCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if telegramRequest(sendCtx, v.Token, "sendMessage", input, nil) != nil {
		v.OperationStatus = "message_delivery_unconfirmed"
		_ = s.store.telegramSave(*v, nil, "")
	}
}
func telegramOption(label, operation, id string, body any) telegramChoice {
	cb := telegramCallback{Operation: operation, InstanceID: id}
	if body != nil {
		cb.Body = telegramBody(body)
	}
	return telegramChoice{label, cb}
}
func (s *server) telegramCommand(ctx context.Context, v *telegramState, id int64, text string) {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return
	}
	command := strings.SplitN(fields[0], "@", 2)[0]
	switch command {
	case "/start", "/menu":
		s.telegramMenu(ctx, v, id)
	case "/status":
		s.telegramDispatch(ctx, v, id, telegramCallback{Operation: "status"})
	case "/notifications":
		s.telegramDispatch(ctx, v, id, telegramCallback{Operation: "notifications"})
	case "/version":
		if len(fields) < 2 || len(fields) > 4 || !instanceIDPattern.MatchString(fields[1]) {
			s.telegramSend(ctx, v, id, "Use /version <instance ID> [recommended|latest|pinned] [version].", nil)
			return
		}
		selection := map[string]string{}
		if len(fields) >= 3 {
			selection["policy"] = fields[2]
		}
		if len(fields) == 4 {
			selection["version"] = fields[3]
		}
		s.telegramDispatch(ctx, v, id, telegramCallback{Operation: "version_review", InstanceID: fields[1], Body: telegramBody(selection)})
	case "/help":
		s.telegramSend(ctx, v, id, "Use /menu for existing Minecraft servers, backups, Network and updates. Server Management actions require confirmation. Access is granted and revoked by the primary administrator in Control Center → AI & API Connections → Add → Telegram. Usernames do not grant access. Notifications are optional.", nil)
	default:
		s.telegramSend(ctx, v, id, "Use /menu or /help.", nil)
	}
}
func (s *server) telegramMenu(ctx context.Context, v *telegramState, id int64) {
	if !telegramAllowed(v, id, "menu") {
		return
	}
	choices := []telegramChoice{telegramOption("Appliance status", "status", "", nil), telegramOption("Minecraft Network / Velocity", "network_menu", "", nil), telegramOption("OS updates", "system_updates", "", nil), telegramOption("Check OS updates", "system_update_check", "", nil), telegramOption("Download OS update", "system_download", "", nil), telegramOption("Reboot options", "reboot_menu", "", nil), telegramOption("Notifications", "notifications", "", nil)}
	instances, err := minecraftInstances.listInstances("")
	if err != nil {
		s.telegramSend(ctx, v, id, "Server inventory unavailable.", nil)
		return
	}
	for _, instance := range instances {
		label := instance.DisplayName
		if label == "" {
			label = instance.ID
		}
		choices = append(choices, telegramOption(label+" ("+instance.ID+")", "instance_menu", instance.ID, nil))
	}
	s.telegramSend(ctx, v, id, "JustVoxel — choose an appliance or Minecraft server action.", choices)
}
func (s *server) telegramInstanceMenu(ctx context.Context, v *telegramState, id int64, instance string) {
	if !instanceIDPattern.MatchString(instance) {
		return
	}
	if _, err := instanceRuntimes.readRecord(instance); err != nil {
		s.telegramSend(ctx, v, id, "That server is unavailable.", nil)
		return
	}
	choices := []telegramChoice{}
	for _, item := range [][2]string{{"Status", "instance_status"}, {"Server details", "instance_details"}, {"Players", "players"}, {"Backup inventory", "backups"}, {"Backup status", "backup_status"}, {"Minecraft updates", "version_status"}, {"Container updates", "container_status"}, {"Minecraft update progress", "version_update_status"}, {"Container update progress", "container_update_status"}, {"Start", "instance_start"}, {"Stop", "instance_stop"}, {"Restart", "instance_restart"}, {"Back up now", "backup_now"}, {"Review Minecraft update", "version_review"}, {"Review container update", "container_review"}} {
		choices = append(choices, telegramOption(item[0], item[1], instance, nil))
	}
	choices = append(choices, telegramOption("Enable Start on Boot", "start_on_boot", instance, map[string]bool{"enabled": true}), telegramOption("Disable Start on Boot", "start_on_boot", instance, map[string]bool{"enabled": false}), telegramOption("Main menu", "menu", "", nil))
	s.telegramSend(ctx, v, id, "Minecraft server "+instance, choices)
}
func (s *server) telegramNetworkMenu(ctx context.Context, v *telegramState, id int64) {
	code, raw := s.telegramInvoke(ctx, v, id, telegramCallback{Operation: "network_status"})
	if code >= 400 {
		s.telegramSend(ctx, v, id, telegramResultText("network_status", code, raw), nil)
		return
	}
	var status struct {
		Config  minecraftNetworkConfig `json:"config"`
		Servers []struct {
			ID       string `json:"id"`
			Eligible bool   `json:"eligible"`
		} `json:"servers"`
	}
	if json.Unmarshal(raw, &status) != nil {
		return
	}
	expected := telegramNetworkFingerprint(status.Config)
	choices := []telegramChoice{telegramOption("Enable Network", "network_change", "", map[string]any{"enabled": true, "expected": expected}), telegramOption("Disable Network", "network_change", "", map[string]any{"enabled": false, "expected": expected})}
	for _, server := range status.Servers {
		selected := networkContains(status.Config.Members, server.ID)
		if !selected && !server.Eligible {
			continue
		}
		members := []string{}
		for _, member := range status.Config.Members {
			if member != server.ID {
				members = append(members, member)
			}
		}
		label := "Remove " + server.ID
		if !selected {
			members = append(members, server.ID)
			label = "Select " + server.ID
		}
		choices = append(choices, telegramOption(label, "network_change", "", map[string]any{"members": members, "expected": expected}))
	}
	choices = append(choices, telegramOption("Main menu", "menu", "", nil))
	s.telegramSend(ctx, v, id, telegramResultText("network_status", code, raw)+"\nTo change membership, disable Network first, then stop affected servers. Existing compatibility and recovery checks still apply.", choices)
}
func (s *server) telegramReviewUpdate(ctx context.Context, v *telegramState, id int64, cb telegramCallback) {
	read := "version_status"
	operation := "version_update"
	if cb.Operation == "container_review" {
		read = "container_status"
		operation = "container_update"
	}
	code, raw := s.telegramInvoke(ctx, v, id, telegramCallback{Operation: read, InstanceID: cb.InstanceID, Body: cb.Body})
	if code >= 400 {
		s.telegramSend(ctx, v, id, telegramResultText(read, code, raw), nil)
		return
	}
	var plan struct {
		Fingerprint string `json:"plan_fingerprint"`
	}
	if json.Unmarshal(raw, &plan) != nil || !operationFingerprintPattern.MatchString(plan.Fingerprint) {
		s.telegramSend(ctx, v, id, "A current update plan could not be verified.", nil)
		return
	}
	body := map[string]any{"plan_fingerprint": plan.Fingerprint, "confirm_players": false}
	if operation == "version_update" {
		var selection map[string]string
		if len(cb.Body) > 0 && json.Unmarshal(cb.Body, &selection) != nil {
			return
		}
		for _, key := range []string{"policy", "version"} {
			if selection[key] != "" {
				body[key] = selection[key]
			}
		}
	} else {
		body["restart_now"] = true
		body["backup_before_restart"] = true
	}
	choices := []telegramChoice{telegramOption("Review and apply update", operation, cb.InstanceID, body), telegramOption("Cancel", "instance_menu", cb.InstanceID, nil)}
	s.telegramSend(ctx, v, id, telegramResultText(read, code, raw)+"\nThe reviewed plan will be validated again before the existing backup and update workflow runs.", choices)
}
func (s *server) telegramRebootMenu(ctx context.Context, v *telegramState, id int64) {
	code, raw := s.telegramInvoke(ctx, v, id, telegramCallback{Operation: "system_updates"})
	if code >= 400 {
		s.telegramSend(ctx, v, id, telegramResultText("system_updates", code, raw), nil)
		return
	}
	var status adminSystemUpdateStatus
	if json.Unmarshal(raw, &status) != nil {
		return
	}
	choices := []telegramChoice{telegramOption("Ordinary reboot", "reboot", "", adminSystemActionRequest{ActionConfirmed: true})}
	if status.Staged != nil && status.RebootRequired {
		for _, seconds := range []int{60, 10} {
			for _, backup := range []bool{true, false} {
				label := fmt.Sprintf("Update reboot · %ds warning", seconds)
				if backup {
					label += " · backup first"
				}
				choices = append(choices, telegramOption(label, "update_reboot", "", adminSystemUpdateRebootRequest{ActionConfirmed: true, WarningSeconds: seconds, BackupMinecraft: backup, StagedDigest: status.Staged.Digest}))
			}
		}
	}
	choices = append(choices, telegramOption("Cancel", "menu", "", nil))
	s.telegramSend(ctx, v, id, telegramResultText("system_updates", code, raw)+"\nReboot needs explicit confirmation. Update reboot options apply only to a verified downloaded or staged update.", choices)
}
func (s *server) telegramDispatch(ctx context.Context, v *telegramState, id int64, cb telegramCallback) {
	if !telegramAllowed(v, id, cb.Operation) {
		actor := telegramActor(id)
		_ = s.store.recordAuditEvent(actor, "telegram_denied", "telegram", false, "permission denied")
		s.telegramSend(ctx, v, id, "Your permission does not allow this action.", nil)
		return
	}
	switch cb.Operation {
	case "menu":
		s.telegramMenu(ctx, v, id)
		return
	case "instance_menu":
		s.telegramInstanceMenu(ctx, v, id, cb.InstanceID)
		return
	case "network_menu":
		s.telegramNetworkMenu(ctx, v, id)
		return
	case "version_review", "container_review":
		s.telegramReviewUpdate(ctx, v, id, cb)
		return
	case "reboot_menu":
		s.telegramRebootMenu(ctx, v, id)
		return
	case "notifications", "notifications_toggle":
		user, _ := v.user(id)
		if cb.Operation == "notifications_toggle" {
			for i := range v.Users {
				if v.Users[i].ID == id {
					v.Users[i].Notifications = !v.Users[i].Notifications
					user = v.Users[i]
				}
			}
			if s.store.telegramSave(*v, nil, "") != nil {
				return
			}
		}
		label := "Enable notifications"
		if user.Notifications {
			label = "Disable notifications"
		}
		s.telegramSend(ctx, v, id, "Notifications: "+strconv.FormatBool(user.Notifications), []telegramChoice{telegramOption(label, "notifications_toggle", "", nil), telegramOption("Main menu", "menu", "", nil)})
		return
	}
	op, exists := s.telegramOperations()[cb.Operation]
	if !exists {
		return
	}
	if op.Management && !cb.Confirmed {
		cb.Confirmed = true
		label := "Confirm " + strings.ReplaceAll(cb.Operation, "_", " ")
		preview := "Confirm this action"
		if cb.InstanceID != "" {
			preview += " for " + cb.InstanceID
		}
		if cb.Operation == "update_reboot" {
			var reboot adminSystemUpdateRebootRequest
			if json.Unmarshal(cb.Body, &reboot) != nil {
				return
			}
			preview += fmt.Sprintf(". Reboot into the reviewed update with a %d-second warning; backup before reboot: %t", reboot.WarningSeconds, reboot.BackupMinecraft)
		}
		preview += ". Existing player and safety checks will run before interruption."
		s.telegramSend(ctx, v, id, preview, []telegramChoice{{label, cb}, telegramOption("Cancel", "menu", "", nil)})
		return
	}
	reboot := cb.Operation == "reboot" || cb.Operation == "update_reboot"
	if reboot {
		if v.Reboot != nil {
			s.telegramSend(ctx, v, id, "A previous reboot request still needs verification. Check JustVoxel before retrying.", nil)
			return
		}
		bootID, err := readSystemUpdateBootID()
		if err != nil || !bootIDPattern.MatchString(bootID) {
			s.telegramSend(ctx, v, id, "Boot identity unavailable; reboot was not requested.", nil)
			return
		}
		pending := &telegramReboot{BootID: bootID, RequestedAt: s.store.now(), State: "requesting", Operation: cb.Operation}
		if cb.Operation == "update_reboot" {
			var request adminSystemUpdateRebootRequest
			if json.Unmarshal(cb.Body, &request) != nil || !operationFingerprintPattern.MatchString(request.StagedDigest) {
				return
			}
			pending.Expected = request.StagedDigest
		} else {
			status, err := s.collectSystemUpdateStatus(ctx)
			if err != nil {
				s.telegramSend(ctx, v, id, "Deployment status unavailable; reboot was not requested.", nil)
				return
			}
			if status.Staged != nil && !status.stagedDownloadOnly {
				pending.Expected = status.Staged.Digest
			}
		}
		for _, user := range v.Users {
			if user.Notifications {
				v.Seen[strconv.FormatInt(user.ID, 10)+":online"] = bootID
			}
		}
		v.Reboot = pending
		if s.store.telegramSave(*v, nil, "") != nil {
			return
		}
	}
	tokenAtAcceptance := v.Token
	code, raw := s.telegramInvoke(ctx, v, id, cb)
	if v.Schema != 1 || v.Token != tokenAtAcceptance {
		return
	}
	var outcome struct {
		Accepted             bool   `json:"accepted"`
		ConfirmationRequired bool   `json:"confirmation_required"`
		Reason               string `json:"reason"`
		State                string `json:"state"`
	}
	_ = json.Unmarshal(raw, &outcome)
	if reboot && v.Reboot != nil {
		if code < 400 && outcome.Accepted {
			v.Reboot.State = "accepted"
			v.OperationStatus = "reboot_requested"
		} else if outcome.ConfirmationRequired || (code >= 400 && code < 500) || outcome.State == "failed" {
			v.Reboot = nil
		} else {
			v.Reboot.State = "unverified"
			v.OperationStatus = "reboot_result_unknown"
		}
		if s.store.telegramSave(*v, nil, "") != nil {
			return
		}
	}
	if outcome.ConfirmationRequired && outcome.Reason == "players_online" {
		body := map[string]any{}
		if len(cb.Body) > 0 && json.Unmarshal(cb.Body, &body) != nil {
			return
		}
		body["confirm_players"] = true
		cb.Body = telegramBody(body)
		cb.Confirmed = true
		s.telegramSend(ctx, v, id, "Players are online. Confirm their interruption to continue with the existing warning and shutdown checks.", []telegramChoice{{"Confirm player interruption", cb}, telegramOption("Cancel", "menu", "", nil)})
		return
	}
	v.OperationStatus = cb.Operation + "_completed"
	if code >= 400 {
		v.OperationStatus = cb.Operation + "_failed"
	} else if code == 202 {
		v.OperationStatus = cb.Operation + "_accepted"
	}
	text := telegramResultText(cb.Operation, code, raw)
	if reboot && v.Reboot != nil && v.Reboot.State == "unverified" {
		v.OperationStatus = "reboot_result_unknown"
		text = "The reboot result could not be verified. The pending request is saved; JustVoxel will verify boot and deployment state before reporting a result."
	}
	if s.store.telegramSave(*v, nil, "") != nil {
		return
	}
	s.telegramSend(ctx, v, id, text, []telegramChoice{telegramOption("Main menu", "menu", "", nil)})
	if reboot && v.Reboot != nil && code < 400 && outcome.Accepted {
		s.telegramNotify(ctx, v, "reboot_started", v.Reboot.RequestedAt.Format(time.RFC3339Nano), "Reboot workflow started. Return to service and any OS update result will be verified after restart.", false)
	}
}
