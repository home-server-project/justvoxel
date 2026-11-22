package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/rest"
)

// Only fixed Stage 4 operations enter this adapter. Selection authority lives in
// short-lived server-side actions, never component IDs or arbitrary request data.
func (s *server) discordAdvancedInvokeLocked(ctx context.Context, rt *discordRuntime, guild, user, op string, a discordPendingAction) (int, []byte) {
	v, err := s.store.discordRead()
	if err != nil || !s.discordUsableLocked(rt, v, guild) || !validDiscordSnowflake(user) || !v.administrator(user) || (a.Generation != 0 && a.Generation != v.Generation) {
		return 403, nil
	}
	method, path := "GET", ""
	var handler http.HandlerFunc
	var body any = struct{}{}
	instance := false
	management := false
	switch op {
	case "instance_details":
		instance = true
		path = "configuration"
		handler = s.instanceRuntimeConfiguration
	case "backups":
		instance = true
		path = "backups"
		handler = s.instanceBackupRead
	case "backup_status":
		instance = true
		path = "backups/status"
		handler = s.instanceBackupRead
	case "version_status":
		instance = true
		path = "version/status"
		handler = s.adminVersionStatus
	case "version_update_status":
		instance = true
		path = "version/update/current"
		handler = s.adminMinecraftStackUpdateCurrent
	case "container_status":
		instance = true
		path = "container/status"
		handler = s.adminMinecraftContainerStatus
	case "container_update_status":
		instance = true
		path = "container/update/current"
		handler = s.adminMinecraftContainerUpdateCurrent
	case "network_status":
		path = "/v1/minecraft/network"
		handler = s.minecraftNetworkStatus
	case "system_updates":
		path = "/v1/admin/system/updates"
		handler = s.adminSystemUpdateStatus
	case "system_update_check":
		method = "POST"
		path = "/v1/admin/system/updates/check"
		handler = s.adminSystemUpdateCheck
	case "version_update":
		instance = true
		management = true
		path = "version/update"
		handler = s.adminMinecraftStackUpdate
		if !operationFingerprintPattern.MatchString(a.Fingerprint) {
			return 403, nil
		}
		body = minecraftStackUpdateRequest{Policy: a.Policy, Version: a.Version, PlanFingerprint: a.Fingerprint, ConfirmPlayers: a.PlayersConfirmed}
	case "container_update":
		instance = true
		management = true
		path = "container/update"
		handler = s.adminMinecraftContainerUpdate
		if !operationFingerprintPattern.MatchString(a.Fingerprint) {
			return 403, nil
		}
		body = minecraftContainerUpdateRequest{PlanFingerprint: a.Fingerprint, RestartNow: true, BackupBeforeRestart: true, ConfirmPlayers: a.PlayersConfirmed}
	case "system_download":
		management = true
		path = "/v1/admin/system/updates"
		handler = s.adminSystemUpdate
	case "reboot":
		management = true
		path = "/v1/admin/system/reboot"
		handler = func(w http.ResponseWriter, r *http.Request) { s.adminSystemAction(w, r, "reboot") }
		body = adminSystemActionRequest{ActionConfirmed: true, ConfirmPlayers: a.PlayersConfirmed}
	case "update_reboot":
		management = true
		path = "/v1/admin/system/update-reboot"
		handler = s.adminSystemUpdateReboot
		if !operationFingerprintPattern.MatchString(a.StagedDigest) || (a.WarningSeconds != 10 && a.WarningSeconds != 60) {
			return 403, nil
		}
		body = adminSystemUpdateRebootRequest{ActionConfirmed: true, ConfirmPlayers: a.PlayersConfirmed, BackupMinecraft: a.BackupMinecraft, WarningSeconds: a.WarningSeconds, StagedDigest: a.StagedDigest}
	default:
		return 403, nil
	}
	if instance {
		if !discordRegisteredInstance(a.InstanceID) {
			return 403, nil
		}
		path = "/v1/minecraft/instances/" + a.InstanceID + "/" + path
	} else if a.InstanceID != "" {
		return 403, nil
	}
	if management {
		method = "POST"
		if a.Operation != op || a.UserID != user || a.GuildID != guild || a.Generation != v.Generation {
			return 403, nil
		}
	}
	actor := discordActor(v, user)
	if management && s.store.recordAuditEvent(actor, "discord_attempt_"+op, a.InstanceID, true, "") != nil {
		return 503, nil
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return 503, nil
	}
	request, _ := http.NewRequestWithContext(context.WithValue(ctx, gatewayPrincipalKey{}, actor), method, "http://discord"+path, bytes.NewReader(raw))
	request.Header.Set("Content-Type", "application/json")
	request.SetPathValue("id", a.InstanceID)
	capture := &gatewayResponse{header: make(http.Header), status: 200}
	s.discordMu.Unlock()
	handler(capture, request)
	s.discordMu.Lock()
	if capture.overflow || !json.Valid(capture.body.Bytes()) {
		return 502, nil
	}
	return capture.status, append([]byte(nil), capture.body.Bytes()...)
}

func discordRegisteredInstance(id string) bool {
	if !instanceIDPattern.MatchString(id) {
		return false
	}
	identities, err := minecraftInstances.listInstances("")
	if err != nil {
		return false
	}
	for _, i := range identities {
		if i.ID == id {
			return true
		}
	}
	return false
}
func (s *server) discordAdvancedAuthorized(rt *discordRuntime, generation int64, guild, user string) (discordState, bool) {
	v, err := s.store.discordRead()
	return v, err == nil && v.Generation == generation && validDiscordSnowflake(user) && v.administrator(user) && s.discordUsableLocked(rt, v, guild)
}
func discordAdvancedFailure() discordPanel {
	return discordPanel{text: "The operation could not complete. Review current status and recovery information in JustVoxel WebUI."}
}

// Even public presentation strings are bounded and stripped of private paths and
// credential-shaped content. Raw errors and free-form backend messages are omitted.
func discordAdvancedText(value string) string {
	value = gatewayCredentialText.ReplaceAllString(gatewayPrivatePath.ReplaceAllString(value, "[private]"), "[private]")
	return discordText(value)
}
func discordEnabled(b bool) string {
	if b {
		return "Enabled"
	}
	return "Disabled"
}
func discordAdvancedButtons(v discordState, id string, items ...[2]string) []discord.LayoutComponent {
	rows := []discord.LayoutComponent{}
	for len(items) > 0 {
		n := min(5, len(items))
		buttons := []discord.InteractiveComponent{}
		for _, item := range items[:n] {
			buttons = append(buttons, discordButton(v, item[0], "adv_"+item[1], id))
		}
		rows = append(rows, discord.NewActionRow(buttons...))
		items = items[n:]
	}
	return rows
}
func discordAdvancedConfirm(rt *discordRuntime, a discordPendingAction, text string) discordPanel {
	nonce, ok := discordNewAction(rt, a, time.Now())
	if !ok {
		return discordFailure()
	}
	if a.PlayersConfirmed {
		text = "Players are online.\nConfirm player interruption to continue.\n" + text
	}
	prefix := fmt.Sprintf("jv:%d:", a.Generation)
	return discordPanel{text: text, components: []discord.LayoutComponent{discord.NewActionRow(discord.NewDangerButton("Confirm", prefix+"adv_confirm:"+nonce), discord.NewSecondaryButton("Cancel", prefix+"adv_cancel:"+nonce))}}
}
func discordAdvancedReviewChoice(rt *discordRuntime, a discordPendingAction, label string) (discord.InteractiveComponent, bool) {
	a.ReviewOnly = true
	nonce, ok := discordNewAction(rt, a, time.Now())
	if !ok {
		return nil, false
	}
	return discord.NewSecondaryButton(label, fmt.Sprintf("jv:%d:adv_review:%s", a.Generation, nonce)), true
}
func discordAdvancedPreview(a discordPendingAction) string {
	switch a.Operation {
	case "start_on_boot":
		return "Change Start on Boot to " + discordEnabled(a.DesiredEnabled) + "?"
	case "network_change":
		if a.EnabledSet {
			return "Change Minecraft Network to " + discordEnabled(a.DesiredEnabled) + "?"
		}
		return "Apply this reviewed Minecraft Network member change?"
	case "version_update":
		return "Apply the reviewed Minecraft update? The plan will be checked again."
	case "container_update":
		return "Update container?\nRestart after update: yes\nBackup before restart: yes"
	case "system_download":
		return "Download & stage the JustVoxel OS update? Reboot is a separate action."
	case "reboot":
		return "Ordinary reboot? The appliance will restart."
	case "update_reboot":
		backup := "no backup"
		if a.BackupMinecraft {
			backup = "backup first"
		}
		return fmt.Sprintf("Update reboot · %ds warning · %s?", a.WarningSeconds, backup)
	}
	return "Confirm operation?"
}

type discordNetworkStatus struct {
	Config  minecraftNetworkConfig `json:"config"`
	Servers []struct {
		ID       string `json:"id"`
		Eligible bool   `json:"eligible"`
	} `json:"servers"`
	RecoveryRequired bool `json:"recovery_required"`
	RuntimeReady     bool `json:"runtime_ready"`
}

// Omit absent values instead of showing blank diagnostic labels.
func discordSummaryField(text, label, value string) string {
	if strings.TrimSpace(value) == "" {
		return text
	}
	return text + "\n" + label + ": " + discordAdvancedText(value)
}
func discordHumanState(value string) string {
	value = strings.ReplaceAll(strings.TrimSpace(value), "_", " ")
	if value == "" {
		return "Unavailable"
	}
	return strings.ToUpper(value[:1]) + value[1:]
}
func discordUpdateAvailability(available bool) string {
	if available {
		return "Available"
	}
	return "Up to date"
}
func discordConnections(java, bedrock bool) string {
	if java && bedrock {
		return "Java + Bedrock"
	}
	if java {
		return "Java"
	}
	if bedrock {
		return "Bedrock"
	}
	return "Unavailable"
}
func discordSoftware(value string) string {
	switch value {
	case "paper":
		return "Paper"
	case "purpur":
		return "Purpur"
	case "fabric":
		return "Fabric"
	case "forge":
		return "Forge"
	case "neoforge":
		return "NeoForge"
	case "vanilla":
		return "Vanilla"
	case "bedrock":
		return "Bedrock"
	case "folia":
		return "Folia"
	case "quilt":
		return "Quilt"
	case "spigot":
		return "Spigot"
	}
	return discordHumanState(value)
}
func discordOSSummary(status adminSystemUpdateStatus) string {
	text := discordSummaryField("JustVoxel OS", "Running", status.Running.Version)
	if status.Staged != nil {
		text = discordSummaryField(text, "Staged", status.Staged.Version)
	}
	if status.Available != nil {
		text = discordSummaryField(text, "Update available", status.Available.Version)
	}
	switch status.CheckState {
	case "current":
		if !status.RebootRequired && !status.ActivationPending && !status.ActivationFailed {
			text += "\nStatus: No update action required"
		}
	case "staged_current":
		text += "\nStatus: Update downloaded and staged"
	case "update_available":
		text += "\nStatus: Update available"
	case "unknown":
		text += "\nStatus: Check for updates to see availability"
	}
	if status.RebootRequired {
		text += "\n⚠ Reboot required"
	}
	if status.ActivationPending {
		text += "\n⚠ Update activation pending"
	}
	if status.ActivationFailed {
		text += "\n⚠ Update activation failed\nReview System Update in JustVoxel."
	}
	return text
}
func discordNetworkSummary(status discordNetworkStatus, instances []discordInstance) string {
	text := fmt.Sprintf("Minecraft Network / Velocity · %s\n\nJava: %d · Bedrock: %d", discordEnabled(status.Config.Enabled), status.Config.JavaPort, status.Config.BedrockPort)
	selected := map[string]bool{}
	for _, id := range status.Config.Members {
		selected[id] = true
	}
	names := map[string]int{}
	for _, server := range instances {
		if selected[server.ID] {
			names[server.Name]++
		}
	}
	members := []string{}
	count := 0
	for _, server := range instances {
		if !selected[server.ID] {
			continue
		}
		count++
		if len(members) >= 10 {
			continue
		}
		name := discordAdvancedText(server.Name)
		if name == "" {
			name = "Unnamed server"
		}
		if names[server.Name] > 1 {
			name += " (" + server.ID + ")"
		}
		members = append(members, name)
	}
	if len(members) > 0 {
		text += "\nServers: " + strings.Join(members, ", ")
	}
	if count > 10 {
		text += "\nMore selected servers are available in JustVoxel WebUI."
	}
	if status.RuntimeReady {
		text += "\nStatus: Ready"
	} else {
		text += "\nStatus: Not ready"
	}
	if status.RecoveryRequired {
		text += "\n⚠ Recovery required"
	}
	return text
}

// Every entry is already a native Discord component with generation fencing.
// This function reauthorizes all reads as well as mutations for Stage 4.
func (s *server) discordAdvancedLocked(ctx context.Context, rt *discordRuntime, v discordState, i discord.Interaction, guild, user, op, id string) discordPanel {
	var ok bool
	v, ok = s.discordAdvancedAuthorized(rt, v.Generation, guild, user)
	if !ok {
		return discordFailure()
	}
	generation := v.Generation
	channel := ""
	if i.Channel().MessageChannel != nil {
		channel = i.Channel().ID().String()
	}
	action := discordPendingAction{Generation: generation, GuildID: guild, UserID: user, InstanceID: id, ChannelID: channel}
	if op == "adv_confirm" || op == "adv_cancel" || op == "adv_review" {
		a, found := discordTakeAction(rt, id, guild, user, time.Now())
		if !found {
			return discordFailure()
		}
		if op == "adv_cancel" {
			return discordPanel{text: "Cancelled. Open /jv to continue."}
		}
		if a.InstanceID != "" && !discordRegisteredInstance(a.InstanceID) {
			return discordFailure()
		}
		if op == "adv_review" {
			if !a.ReviewOnly {
				return discordFailure()
			}
			a.ReviewOnly = false
			return discordAdvancedConfirm(rt, a, discordAdvancedPreview(a))
		}
		if a.ReviewOnly {
			return discordFailure()
		}
		return s.discordAdvancedApplyLocked(ctx, rt, a)
	}
	switch op {
	case "adv_system":
		if id != "" {
			return discordFailure()
		}
		return discordPanel{text: "System", components: discordAdvancedButtons(v, "", [2]string{"OS updates", "system_updates"}, [2]string{"Check OS updates", "system_update_check"}, [2]string{"Download & stage update", "download"}, [2]string{"Reboot options", "reboots"}, [2]string{"Back", "back"})}
	case "adv_back":
		instances, found := s.discordInstancesLocked(ctx, rt, guild, user)
		if !found {
			return discordFailure()
		}
		current, authorized := s.discordAdvancedAuthorized(rt, generation, guild, user)
		if !authorized {
			return discordFailure()
		}
		return discordReadPanel(current, user, instances, id)
	case "adv_download":
		if id != "" || !validDiscordSnowflake(channel) {
			return discordFailure()
		}
		action.Operation = "system_download"
		return discordAdvancedConfirm(rt, action, discordAdvancedPreview(action))
	case "adv_system_updates", "adv_system_update_check", "adv_reboots":
		if id != "" {
			return discordFailure()
		}
		read := "system_updates"
		if op == "adv_system_update_check" {
			read = "system_update_check"
		}
		code, raw := s.discordAdvancedInvokeLocked(ctx, rt, guild, user, read, action)
		var status adminSystemUpdateStatus
		if code >= 400 || json.Unmarshal(raw, &status) != nil {
			return discordAdvancedFailure()
		}
		if _, ok = s.discordAdvancedAuthorized(rt, generation, guild, user); !ok {
			return discordFailure()
		}
		panel := discordPanel{text: discordOSSummary(status)}
		if op == "adv_reboots" {
			a := action
			a.Operation = "reboot"
			if button, yes := discordAdvancedReviewChoice(rt, a, "Ordinary reboot"); yes {
				panel.components = append(panel.components, discord.NewActionRow(button))
			}
			if status.Staged != nil && status.RebootRequired && operationFingerprintPattern.MatchString(status.Staged.Digest) {
				for _, seconds := range []int{60, 10} {
					buttons := []discord.InteractiveComponent{}
					for _, backup := range []bool{true, false} {
						a := action
						a.Operation = "update_reboot"
						a.WarningSeconds = seconds
						a.BackupMinecraft = backup
						a.StagedDigest = status.Staged.Digest
						label := fmt.Sprintf("Update reboot · %ds warning · no backup", seconds)
						if backup {
							label = fmt.Sprintf("Update reboot · %ds warning · backup first", seconds)
						}
						if b, yes := discordAdvancedReviewChoice(rt, a, label); yes {
							buttons = append(buttons, b)
						}
					}
					if len(buttons) > 0 {
						panel.components = append(panel.components, discord.NewActionRow(buttons...))
					}
				}
			}
		}
		panel.components = append(panel.components, discordAdvancedButtons(v, "", [2]string{"Back", "system"})...)
		return panel
	case "adv_network", "adv_members", "adv_member", "adv_network_enable", "adv_network_disable":
		if op != "adv_member" && id != "" {
			return discordFailure()
		}
		action.InstanceID = ""
		code, raw := s.discordAdvancedInvokeLocked(ctx, rt, guild, user, "network_status", action)
		var status discordNetworkStatus
		if code >= 400 || json.Unmarshal(raw, &status) != nil {
			return discordAdvancedFailure()
		}
		instances, found := s.discordInstancesLocked(ctx, rt, guild, user)
		if !found {
			return discordFailure()
		}
		if _, ok = s.discordAdvancedAuthorized(rt, generation, guild, user); !ok {
			return discordFailure()
		}
		action.NetworkExpected = minecraftNetworkFingerprint(status.Config)
		selected := map[string]bool{}
		for _, member := range status.Config.Members {
			selected[member] = true
		}
		if op == "adv_network_enable" || op == "adv_network_disable" {
			action.Operation = "network_change"
			action.EnabledSet = true
			action.DesiredEnabled = op == "adv_network_enable"
			return discordAdvancedConfirm(rt, action, discordAdvancedPreview(action))
		}
		eligible := map[string]bool{}
		for _, server := range status.Servers {
			eligible[server.ID] = server.Eligible || selected[server.ID]
		}
		if op == "adv_member" {
			if !eligible[id] || !discordRegisteredInstance(id) {
				return discordFailure()
			}
			name := ""
			for _, server := range instances {
				if server.ID == id {
					name = server.Name
				}
			}
			if name == "" {
				name = id
			}
			members := []string{}
			for _, member := range status.Config.Members {
				if member != id {
					members = append(members, member)
				}
			}
			label := "Remove "
			if !selected[id] {
				members = append(members, id)
				label = "Add "
			}
			action.Operation = "network_change"
			action.NetworkMembers = members
			return discordAdvancedConfirm(rt, action, label+discordAdvancedText(name)+"?\nDisable Minecraft Network and stop affected servers before changing membership.")
		}
		panel := discordPanel{text: discordNetworkSummary(status, instances)}
		if op == "adv_members" {
			options, total := discordNetworkMemberOptions(instances, eligible)
			if len(options) > 0 {
				panel.components = append(panel.components, discord.NewActionRow(discord.NewStringSelectMenu(discordComponentID(v, "adv_member", ""), "Choose one member to add or remove", options...)))
			}
			if total > 25 {
				panel.text += "\nAdditional membership management is available in JustVoxel WebUI."
			}
			panel.components = append(panel.components, discordAdvancedButtons(v, "", [2]string{"Back", "network"})...)
		} else {
			toggle := [2]string{"Enable Network", "network_enable"}
			if status.Config.Enabled {
				toggle = [2]string{"Disable Network", "network_disable"}
			}
			panel.components = discordAdvancedButtons(v, "", toggle, [2]string{"Change members", "members"}, [2]string{"Back", "back"})
		}
		return panel
	}
	if !discordRegisteredInstance(id) {
		return discordFailure()
	}
	if op == "adv_menu" {
		return discordPanel{text: "Advanced Minecraft server administration", components: discordAdvancedButtons(v, id, [2]string{"Server details", "instance_details"}, [2]string{"Backup inventory", "backups"}, [2]string{"Backup status", "backup_status"}, [2]string{"Start on Boot", "boot"}, [2]string{"Minecraft updates", "version_status"}, [2]string{"Container updates", "container_status"}, [2]string{"Back", "back"})}
	}
	read := strings.TrimPrefix(op, "adv_")
	if op == "adv_boot" || op == "adv_boot_review" {
		read = "instance_details"
	}
	if op == "adv_version_review" {
		read = "version_status"
	}
	if op == "adv_container_review" {
		read = "container_status"
	}
	switch read {
	case "instance_details", "backups", "backup_status", "version_status", "container_status", "version_update_status", "container_update_status":
	default:
		return discordFailure()
	}
	code, raw := s.discordAdvancedInvokeLocked(ctx, rt, guild, user, read, action)
	if code >= 400 {
		return discordAdvancedFailure()
	}
	if _, ok = s.discordAdvancedAuthorized(rt, generation, guild, user); !ok {
		return discordFailure()
	}
	panel := discordPanel{components: discordAdvancedButtons(v, id, [2]string{"Back", "menu"})}
	switch read {
	case "instance_details":
		var details struct {
			ID      string                  `json:"id"`
			Runtime instanceRuntimeSettings `json:"runtime"`
		}
		if json.Unmarshal(raw, &details) != nil || details.ID != id {
			return discordAdvancedFailure()
		}
		if op == "adv_boot_review" {
			action.Operation = "start_on_boot"
			action.EnabledSet = true
			action.PreviousEnabled = details.Runtime.StartOnBoot
			action.DesiredEnabled = !details.Runtime.StartOnBoot
			return discordAdvancedConfirm(rt, action, discordAdvancedPreview(action))
		}
		if op == "adv_boot" {
			panel.text = "Start on Boot\nCurrent: " + discordEnabled(details.Runtime.StartOnBoot)
			label := "Enable"
			if details.Runtime.StartOnBoot {
				label = "Disable"
			}
			panel.components = discordAdvancedButtons(v, id, [2]string{label, "boot_review"}, [2]string{"Back", "menu"})
			return panel
		}
		name := id
		instances, found := s.discordInstancesLocked(ctx, rt, guild, user)
		if !found {
			return discordFailure()
		}
		for _, server := range instances {
			if server.ID == id {
				name = server.Name
			}
		}
		panel.text = fmt.Sprintf("%s\n\nSoftware: %s\nMinecraft: %s\nStart on Boot: %s\nConnections: %s", discordAdvancedText(name), discordAdvancedText(discordSoftware(details.Runtime.Software)), discordAdvancedText(details.Runtime.Version), discordEnabled(details.Runtime.StartOnBoot), discordConnections(details.Runtime.JavaPort > 0, details.Runtime.BedrockEnabled || details.Runtime.Software == "bedrock"))
	case "backups":
		var inventory struct {
			Backups []instanceBackupMetadata `json:"backups"`
		}
		if json.Unmarshal(raw, &inventory) != nil {
			return discordAdvancedFailure()
		}
		sort.SliceStable(inventory.Backups, func(i, j int) bool { return inventory.Backups[i].Created > inventory.Backups[j].Created })
		panel.text = "Backup inventory"
		if len(inventory.Backups) == 0 {
			panel.text += "\nNo backups are available."
		}
		for index, entry := range inventory.Backups {
			if index >= 10 {
				break
			}
			panel.text += fmt.Sprintf("\n%s · %s · %d bytes", discordAdvancedText(entry.Name), discordAdvancedText(entry.Created), entry.Size)
		}
	case "backup_status":
		var status instanceBackupStatus
		if json.Unmarshal(raw, &status) != nil {
			return discordAdvancedFailure()
		}
		panel.text = discordBackupSummary(status)
	case "version_status":
		var status adminVersionStatus
		if json.Unmarshal(raw, &status) != nil {
			return discordAdvancedFailure()
		}
		panel.text = discordVersionSummary(status)
		safe := status.UpdateAvailable && status.ServerSupported && status.StackState == "updates_available" && (!status.CrossplayEnabled || status.CrossplayCompatible) && operationFingerprintPattern.MatchString(status.PlanFingerprint)
		if op == "adv_version_review" {
			if !safe {
				return panel
			}
			action.Operation = "version_update"
			action.Fingerprint = status.PlanFingerprint
			// Empty selection reproduces the authoritative current-config review exactly.
			return discordAdvancedConfirm(rt, action, panel.text+"\n"+discordAdvancedPreview(action))
		}
		items := [][2]string{{"Update progress", "version_update_status"}, {"Back", "menu"}}
		if safe {
			items = append([][2]string{{"Review Minecraft update", "version_review"}}, items...)
		}
		panel.components = discordAdvancedButtons(v, id, items...)
	case "container_status":
		var status minecraftContainerStatus
		if json.Unmarshal(raw, &status) != nil {
			return discordAdvancedFailure()
		}
		safe := status.InstanceID == id && status.Running && (status.State == "update_available" || status.State == "image_missing" || status.State == "restart_needed") && operationFingerprintPattern.MatchString(status.PlanFingerprint)
		panel.text = discordContainerSummary(status)
		if op == "adv_container_review" {
			if !safe {
				return panel
			}
			action.Operation = "container_update"
			action.Fingerprint = status.PlanFingerprint
			return discordAdvancedConfirm(rt, action, panel.text+"\n"+discordAdvancedPreview(action))
		}
		items := [][2]string{{"Update progress", "container_update_status"}, {"Back", "menu"}}
		if safe {
			items = append([][2]string{{"Review container update", "container_review"}}, items...)
		}
		panel.components = discordAdvancedButtons(v, id, items...)
	case "version_update_status", "container_update_status":
		var status struct {
			Operation *struct {
				State      string `json:"state"`
				Stage      string `json:"stage"`
				Status     string `json:"status"`
				FinishedAt string `json:"finished_at"`
			} `json:"operation"`
		}
		if json.Unmarshal(raw, &status) != nil {
			return discordAdvancedFailure()
		}
		panel.text = "No current update operation."
		if status.Operation != nil {
			panel.text = discordProgressSummary(status.Operation.State, status.Operation.Stage, status.Operation.Status, status.Operation.FinishedAt != "")
		}
	}
	if _, ok = s.discordAdvancedAuthorized(rt, generation, guild, user); !ok {
		return discordFailure()
	}
	return panel
}

func (s *server) discordAdvancedApplyLocked(ctx context.Context, rt *discordRuntime, a discordPendingAction) discordPanel {
	v, ok := s.discordAdvancedAuthorized(rt, a.Generation, a.GuildID, a.UserID)
	if !ok {
		return discordFailure()
	}
	if a.InstanceID != "" && !discordRegisteredInstance(a.InstanceID) {
		return discordFailure()
	}
	if a.Operation == "system_download" {
		// No interaction or token is captured. This one bounded invocation uses the
		// accepted actor; the existing handler locks remain the concurrency authority.
		if !validDiscordSnowflake(a.ChannelID) {
			return discordFailure()
		}
		actor := discordActor(v, a.UserID)
		if s.store.recordAuditEvent(actor, "discord_attempt_system_download", "", true, "") != nil {
			return discordAdvancedFailure()
		}
		go s.discordDownload(actor, a)
		return discordPanel{text: "OS update download started.\nUse System → OS updates to refresh the authoritative state."}
	}
	if a.Operation == "reboot" || a.Operation == "update_reboot" {
		return s.discordRebootLocked(ctx, rt, a)
	}
	var code int
	var raw []byte
	networkChanged := false
	switch a.Operation {
	case "start_on_boot", "network_change":
		if !a.EnabledSet && a.Operation == "start_on_boot" {
			return discordFailure()
		}
		actor := discordActor(v, a.UserID)
		if s.store.recordAuditEvent(actor, "discord_attempt_"+a.Operation, a.InstanceID, true, "") != nil {
			return discordAdvancedFailure()
		}
		request, _ := http.NewRequestWithContext(context.WithValue(ctx, gatewayPrincipalKey{}, actor), "PUT", "http://discord/internal", nil)
		capture := &gatewayResponse{header: make(http.Header), status: 200}
		s.discordMu.Unlock()
		if a.Operation == "start_on_boot" {
			s.setInstanceStartOnBoot(capture, request, a.InstanceID, a.DesiredEnabled, &a.PreviousEnabled)
		} else {
			var enabled *bool
			var members *[]string
			if a.EnabledSet {
				enabled = &a.DesiredEnabled
			} else {
				members = &a.NetworkMembers
			}
			err := s.changeMinecraftNetwork(ctx, enabled, members, a.NetworkExpected, true)
			networkChanged = errors.Is(err, errMinecraftNetworkReviewChanged)
			if err != nil {
				instanceRuntimeError(capture, err, "")
			} else {
				writeJSON(capture, 200, map[string]bool{"ok": true})
			}
		}
		s.discordMu.Lock()
		code = capture.status
		raw = capture.body.Bytes()
		if _, ok = s.discordAdvancedAuthorized(rt, a.Generation, a.GuildID, a.UserID); !ok {
			return discordFailure()
		}
		if code >= 400 {
			if a.Operation == "network_change" {
				if networkChanged {
					return discordPanel{text: "Minecraft Network changed. Review it again."}
				}
				return discordPanel{text: "Minecraft Network could not change. Review it again; disable Network and stop affected servers before changing membership. Existing compatibility and recovery checks still apply."}
			}
			return discordPanel{text: "Start on Boot could not change. Review its current state again in /jv or JustVoxel WebUI."}
		}
		return discordPanel{text: "Change accepted. Open /jv to review current status."}
	default:
		code, raw = s.discordAdvancedInvokeLocked(ctx, rt, a.GuildID, a.UserID, a.Operation, a)
	}
	if _, ok = s.discordAdvancedAuthorized(rt, a.Generation, a.GuildID, a.UserID); !ok {
		return discordFailure()
	}
	var result struct {
		ConfirmationRequired bool   `json:"confirmation_required"`
		Reason               string `json:"reason"`
	}
	_ = json.Unmarshal(raw, &result)
	if result.ConfirmationRequired && result.Reason == "players_online" && !a.PlayersConfirmed && (a.Operation == "version_update" || a.Operation == "container_update") {
		a.PlayersConfirmed = true
		return discordAdvancedConfirm(rt, a, discordAdvancedPreview(a))
	}
	if code == 409 {
		return discordPanel{text: "Update information changed. Review again."}
	}
	if code >= 400 || result.ConfirmationRequired {
		return discordAdvancedFailure()
	}
	label := "Minecraft"
	progress := "version_update_status"
	if a.Operation == "container_update" {
		label = "Container"
		progress = "container_update_status"
	}
	if a.Operation != "container_update" && a.Operation != "version_update" {
		return discordFailure()
	}
	return discordPanel{text: label + " update accepted.\nUse Update progress to review current state.", components: discordAdvancedButtons(v, a.InstanceID, [2]string{"Update progress", progress}, [2]string{"Back", "menu"})}
}

func (s *server) discordDownload(actor session, a discordPendingAction) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute+30*time.Second)
	defer cancel()
	request, _ := http.NewRequestWithContext(context.WithValue(ctx, gatewayPrincipalKey{}, actor), "POST", "http://discord/v1/admin/system/updates", strings.NewReader(`{}`))
	request.Header.Set("Content-Type", "application/json")
	capture := &gatewayResponse{header: make(http.Header), status: 200}
	s.adminSystemUpdate(capture, request)
	var result adminSystemUpdateStatus
	success := capture.status < 400 && !capture.overflow && json.Unmarshal(capture.body.Bytes(), &result) == nil && result.OK
	message := "JustVoxel OS update download did not complete. Review System Update in JustVoxel."
	if success {
		message = "JustVoxel OS update download completed. Open /jv → System → OS updates to review the staged deployment."
	}
	s.discordMu.Lock()
	defer s.discordMu.Unlock()
	s.discordAdvancedNotifyLocked(a.Generation, a.GuildID, a.ChannelID, message)
}
func (s *server) discordAdvancedNotifyLocked(generation int64, guild, channel, message string) {
	v, err := s.store.discordRead()
	rt := s.discordRuntime
	if err != nil || v.Generation != generation || !validDiscordSnowflake(channel) || !s.discordUsableLocked(rt, v, guild) || rt.api == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// Keep the trust fence locked until the best-effort send has completed.
	_, _ = rt.api.CreateMessage(discordID(channel), discord.MessageCreate{Content: message, AllowedMentions: &discord.AllowedMentions{}}, rest.WithCtx(ctx))
}

// instances are already ordered by the authoritative Discord instance listing.
func discordNetworkMemberOptions(instances []discordInstance, eligible map[string]bool) ([]discord.StringSelectMenuOption, int) {
	options := []discord.StringSelectMenuOption{}
	total := 0
	for _, server := range instances {
		if !eligible[server.ID] {
			continue
		}
		total++
		if len(options) >= 25 {
			continue
		}
		label := discordGuildName(server.Name)
		if label == "" {
			label = server.ID
		}
		options = append(options, discord.NewStringSelectMenuOption(label, server.ID))
	}
	return options, total
}

func discordBackupSummary(status instanceBackupStatus) string {
	text := "Backup status: " + discordAdvancedText(discordHumanState(status.State))
	text = discordSummaryField(text, "Last attempt", status.LastAttempt)
	text = discordSummaryField(text, "Last success", status.LastSuccess)
	if status.RestartRequired {
		text += "\n⚠ Restart required"
	}
	return text
}
func discordVersionSummary(status adminVersionStatus) string {
	text := "Minecraft updates\nUpdate: " + discordUpdateAvailability(status.UpdateAvailable)
	for _, field := range [][2]string{{"Installed", status.Installed}, {"Available", status.Available}, {"Recommended", status.Recommended}, {"Selected candidate", status.SelectedCandidate}, {"Policy", status.Policy}, {"Configured version", status.ConfiguredVersion}} {
		text = discordSummaryField(text, field[0], field[1])
	}
	if status.StackState != "" {
		text = discordSummaryField(text, "Status", discordHumanState(status.StackState))
	}
	if !status.ServerSupported {
		text += "\n⚠ Server software is unsupported"
	}
	if status.CrossplayEnabled && !status.CrossplayCompatible {
		text += "\n⚠ Cross-play is incompatible with this update"
	}
	return text
}
func discordContainerSummary(status minecraftContainerStatus) string {
	state := "Stopped"
	if status.Running {
		state = "Running"
	}
	available := status.State == "update_available" || status.State == "image_missing" || status.State == "restart_needed"
	text := "Container updates\nStatus: " + state + "\nUpdate: " + discordUpdateAvailability(available)
	if status.KeepPreviousImages > 0 {
		text += fmt.Sprintf("\nPrevious images retained: %d", status.KeepPreviousImages)
	}
	return text
}
func discordProgressSummary(state, stage, result string, completed bool) string {
	status := "In progress"
	if completed {
		status = "Completed"
	}
	text := "Update progress\nStatus: " + status
	if state != "" {
		text = discordSummaryField(text, "State", discordHumanState(state))
	}
	if stage != "" {
		text = discordSummaryField(text, "Stage", discordHumanState(stage))
	}
	return discordSummaryField(text, "Result", result)
}
