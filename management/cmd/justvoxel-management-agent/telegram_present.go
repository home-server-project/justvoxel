package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

const telegramUnavailable = "Status is unavailable in Telegram. Open JustVoxel WebUI for details."

// These helpers read explicit public fields only. Unknown structures and errors
// never become messages, and image identities remain internal callback evidence.
var telegramHumanValue = regexp.MustCompile(`^[\p{L}\p{N} _.,+()·:-]{1,100}$`)
var telegramHexValue = regexp.MustCompile(`(?i)[a-f0-9]{32,}`)

func telegramHuman(s string) string {
	if !telegramHumanValue.MatchString(s) || telegramHexValue.MatchString(s) || strings.Contains(strings.ToLower(s), "sha256:") || gatewayCredentialText.MatchString(s) || telegramTokenPattern.MatchString(s) {
		return ""
	}
	return s
}
func telegramEnabled(v bool) string {
	if v {
		return "Enabled"
	}
	return "Disabled"
}
func telegramYes(v bool) string {
	if v {
		return "Yes"
	}
	return "No"
}
func telegramStateLabel(s string) string {
	// Fixed status vocabulary: never relay diagnostic text as a state.
	switch strings.ToLower(s) {
	case "healthy", "ok":
		return "Healthy"
	case "degraded":
		return "Degraded"
	case "running", "active":
		return "Running"
	case "stopped", "inactive":
		return "Stopped"
	case "starting":
		return "Starting"
	case "stopping":
		return "Stopping"
	case "failed", "failure":
		return "Failed"
	case "success", "successful", "succeeded", "completed":
		return "Successful"
	case "idle":
		return "Idle"
	case "pending", "queued":
		return "Queued"
	case "interrupted":
		return "Interrupted"
	case "configured":
		return "Configured"
	case "ready":
		return "Ready"
	case "current", "up_to_date":
		return "Up to date"
	case "update_available", "available":
		return "Update available"
	case "not_running":
		return "Stopped"
	case "unsupported":
		return "Unsupported"
	case "unavailable", "unknown":
		return "Unavailable"
	case "attention needed", "needs_attention":
		return "Needs attention"
	case "in_progress":
		return "In progress"
	case "validating":
		return "Validating"
	case "verifying":
		return "Verifying"
	case "rolling_back":
		return "Rolling back"
	case "rolled_back":
		return "Rolled back"
	case "resolved":
		return "Resolved"
	case "recovery_required":
		return "Recovery required"
	case "invalid_configuration":
		return "Configuration unavailable"
	case "restart_needed":
		return "Restart required"
	case "image_missing":
		return "Download required"
	case "maintenance":
		return "Maintenance"
	case "offline":
		return "Offline"
	}
	return ""
}
func telegramDuration(seconds int64) string {
	if seconds < 60 {
		return fmt.Sprintf("%d seconds", seconds)
	}
	if seconds < 3600 {
		return fmt.Sprintf("%d minutes", seconds/60)
	}
	if seconds < 86400 {
		return fmt.Sprintf("%d hours %d minutes", seconds/3600, seconds%3600/60)
	}
	return fmt.Sprintf("%d days %d hours", seconds/86400, seconds%86400/3600)
}
func telegramHumanTime(value string) string {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return ""
	}
	return t.UTC().Format("2 Jan 2006, 15:04 UTC")
}
func telegramInstanceName(id string) string {
	instances, err := minecraftInstances.listInstances("")
	if err == nil {
		for _, instance := range instances {
			if instance.ID == id {
				if name := telegramHuman(instance.DisplayName); name != "" {
					return name
				}
			}
		}
	}
	if instanceIDPattern.MatchString(id) {
		return id
	}
	return "Minecraft server"
}
func telegramResultText(operation string, code int, raw json.RawMessage) string {
	if code >= 400 {
		return "The operation could not complete. Review its current state and recovery information in JustVoxel."
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil || fields == nil {
		return telegramUnavailable
	}
	text := []string{}
	add := func(label, value string) {
		if value != "" {
			text = append(text, label+": "+value)
		}
	}
	switch operation {
	case "status":
		var v struct {
			System struct {
				Health string `json:"health"`
			} `json:"system"`
			Extended struct {
				System struct {
					State  string `json:"state"`
					Uptime *int64 `json:"uptime_seconds"`
					RAM    *int64 `json:"physical_ram_kib"`
					Failed *int   `json:"failed_count"`
				} `json:"system"`
			} `json:"extended"`
		}
		if json.Unmarshal(raw, &v) != nil {
			return telegramUnavailable
		}
		text = append(text, "JustVoxel appliance")
		add("Health", telegramStateLabel(v.System.Health))
		add("State", telegramStateLabel(v.Extended.System.State))
		if p := v.Extended.System.Uptime; p != nil && *p >= 0 {
			add("Uptime", telegramDuration(*p))
		}
		if p := v.Extended.System.RAM; p != nil && *p > 0 {
			add("Memory", fmt.Sprintf("%.1f GiB", float64(*p)/(1024*1024)))
		}
		if p := v.Extended.System.Failed; p != nil && *p >= 0 {
			add("Failed services", fmt.Sprint(*p))
		}
	case "instance_status":
		var v instanceRuntimeStatus
		if json.Unmarshal(raw, &v) != nil || v.State == "" {
			return telegramUnavailable
		}
		text = append(text, telegramInstanceName(v.ID))
		add("State", telegramStateLabel(v.State))
		add("Start on Boot", telegramEnabled(v.StartOnBoot))
	case "instance_details":
		var v struct {
			ID      string `json:"id"`
			Runtime struct {
				Software string `json:"software"`
				Version  string `json:"version"`
			} `json:"runtime"`
		}
		if json.Unmarshal(raw, &v) != nil {
			return telegramUnavailable
		}
		text = append(text, telegramInstanceName(v.ID))
		add("Software", telegramHuman(v.Runtime.Software))
		add("Minecraft version", telegramHuman(v.Runtime.Version))
	case "players":
		var v struct {
			ID      string `json:"id"`
			Online  *int   `json:"online"`
			Maximum int    `json:"maximum"`
		}
		if json.Unmarshal(raw, &v) != nil || v.Online == nil {
			return telegramUnavailable
		}
		text = append(text, telegramInstanceName(v.ID)+" — Players")
		add("Online", fmt.Sprintf("%d / %d", *v.Online, v.Maximum))
		if *v.Online == 0 {
			text = append(text, "No players online.")
		}
	case "backups":
		var v struct {
			ID      string `json:"id"`
			Backups *[]struct {
				Created string `json:"created"`
			} `json:"backups"`
		}
		if json.Unmarshal(raw, &v) != nil || v.Backups == nil {
			return telegramUnavailable
		}
		text = append(text, telegramInstanceName(v.ID)+" — Backups")
		add("Backups retained", fmt.Sprint(len(*v.Backups)))
		latest := ""
		for _, b := range *v.Backups {
			if b.Created > latest {
				latest = b.Created
			}
		}
		add("Last backup", telegramHumanTime(latest))
	case "backup_status":
		var v instanceBackupStatus
		if json.Unmarshal(raw, &v) != nil || v.State == "" {
			return telegramUnavailable
		}
		text = append(text, "Backups")
		add("Status", telegramStateLabel(v.State))
		add("Last successful backup", telegramHumanTime(v.LastSuccess))
		add("Last attempt", telegramHumanTime(v.LastAttempt))
		add("Restart required", telegramYes(v.RestartRequired))
	case "network_status":
		var v struct {
			Config   minecraftNetworkConfig `json:"config"`
			Ready    bool                   `json:"runtime_ready"`
			Recovery bool                   `json:"recovery_required"`
		}
		if json.Unmarshal(raw, &v) != nil || (fields["config"] == nil || string(fields["config"]) == "null") {
			return telegramUnavailable
		}
		text = append(text, "Minecraft Network")
		add("Velocity", telegramEnabled(v.Config.Enabled))
		ready := "Not ready"
		if v.Ready {
			ready = "Ready"
		}
		add("Runtime", ready)
		members := []string{}
		for _, id := range v.Config.Members {
			members = append(members, telegramInstanceName(id))
		}
		if len(members) == 0 {
			members = append(members, "None")
		}
		add("Members", strings.Join(members, ", "))
		add("Recovery required", telegramYes(v.Recovery))
	case "version_status":
		var v adminVersionStatus
		if json.Unmarshal(raw, &v) != nil {
			return telegramUnavailable
		}
		text = append(text, telegramInstanceName(v.InstanceID)+" — Minecraft update")
		add("Installed", telegramHuman(v.Installed))
		add("Available", telegramHuman(v.Available))
		add("Recommended", telegramHuman(v.Recommended))
		add("Selected", telegramHuman(v.SelectedCandidate))
		add("Status", telegramStateLabel(v.StackState))
	case "container_status":
		var v minecraftContainerStatus
		if json.Unmarshal(raw, &v) != nil || v.State == "" {
			return telegramUnavailable
		}
		text = append(text, telegramInstanceName(v.InstanceID)+" — Container update")
		add("Status", telegramStateLabel(v.State))
	case "version_update_status", "container_update_status":
		var v struct {
			Operation *operationJournal `json:"operation"`
		}
		if json.Unmarshal(raw, &v) != nil {
			return telegramUnavailable
		}
		text = append(text, "Update progress")
		if v.Operation == nil {
			if fields["operation"] == nil {
				return telegramUnavailable
			}
			text = append(text, "No update operation yet.")
		} else {
			add("Status", telegramStateLabel(string(v.Operation.State)))
			add("Updated", telegramHumanTime(v.Operation.UpdatedAt))
		}
	case "system_updates", "system_update_check", "system_download":
		var v adminSystemUpdateStatus
		if json.Unmarshal(raw, &v) != nil || (fields["running"] == nil || string(fields["running"]) == "null") {
			return telegramUnavailable
		}
		if operation == "system_download" {
			if v.Staged != nil {
				return "OS update downloaded.\n" + telegramOSSummary(v)
			}
			return "OS download check completed.\n" + telegramOSSummary(v)
		}
		return telegramOSSummary(v)
	default:
		messages := map[string]string{"instance_start": "Server start", "instance_stop": "Server stop", "instance_restart": "Server restart", "backup_now": "Backup", "start_on_boot": "Start on Boot change", "network_change": "Minecraft Network change", "version_update": "Minecraft update", "container_update": "Container update", "reboot": "Reboot", "update_reboot": "Update reboot"}
		label, ok := messages[operation]
		if !ok {
			return telegramUnavailable
		}
		var v struct {
			OK       bool   `json:"ok"`
			Accepted bool   `json:"accepted"`
			State    string `json:"state"`
		}
		if json.Unmarshal(raw, &v) != nil {
			return telegramUnavailable
		}
		if (code == 202 && (v.OK || fields["operation"] != nil)) || v.Accepted {
			return label + " accepted."
		}
		if v.OK || v.State == "running" || v.State == "stopped" {
			return label + " completed."
		}
		return telegramUnavailable
	}
	if len(text) < 2 {
		return telegramUnavailable
	}
	return strings.Join(text, "\n")
}
func telegramOSSummary(v adminSystemUpdateStatus) string {
	lines := []string{"JustVoxel OS"}
	if version := telegramHuman(v.Running.Version); version != "" {
		lines = append(lines, "Running: "+version)
	}
	if v.Staged != nil {
		label := "Downloaded"
		if v.RebootRequired {
			label = "Ready to install"
		}
		version := telegramHuman(v.Staged.Version)
		if version == "" {
			version = "Yes"
		}
		lines = append(lines, label+": "+version)
	} else if v.Available != nil {
		if version := telegramHuman(v.Available.Version); version != "" {
			lines = append(lines, "Available: "+version)
		}
		lines = append(lines, "Downloaded: No")
	} else if v.CheckState == "update_available" {
		lines = append(lines, "Update: Available", "Downloaded: No")
	} else if v.CheckState == "current" {
		lines = append(lines, "Update: No update available")
	} else {
		lines = append(lines, "Update: Check for updates")
	}
	lines = append(lines, "Reboot required: "+telegramYes(v.RebootRequired))
	return strings.Join(lines, "\n")
}
func telegramRebootText(v adminSystemUpdateStatus) string {
	if v.Staged != nil && v.RebootRequired {
		text := "Reboot JustVoxel?\n\nUpdate ready"
		if version := telegramHuman(v.Staged.Version); version != "" {
			text += ": " + version
		}
		return text + "\nBackup before reboot: recommended"
	}
	return "Reboot JustVoxel?\n\nNo OS update is waiting for activation."
}
