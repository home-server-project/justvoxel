package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Save the observation before sending. Ambiguous Telegram deliveries are not
// retried, preventing duplicate notifications across network failures/restarts.
// A missing first observation establishes a baseline, except explicit failures.
func (s *server) telegramNotify(ctx context.Context, v *telegramState, event, value, message string, baseline bool) {
	for _, user := range v.Users {
		if !user.Notifications {
			continue
		}
		key := strconv.FormatInt(user.ID, 10) + ":" + event
		previous, seen := v.Seen[key]
		if seen && previous == value {
			continue
		}
		v.Seen[key] = value
		if s.store.telegramSave(*v, nil, "") != nil {
			return
		}
		if !seen && baseline {
			continue
		}
		s.telegramSend(ctx, v, user.ID, message, nil)
	}
}
func telegramSignature(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
func (s *server) telegramObserve(ctx context.Context, generation int64) {
	s.telegramExecutionMu.RLock()
	defer s.telegramExecutionMu.RUnlock()
	s.telegramMu.Lock()
	defer s.telegramMu.Unlock()
	v, err := s.store.telegramRead()
	if err != nil || !v.Enabled || v.Generation != generation || !s.telegramResetAllowed() {
		return
	}
	// Synthetic calls remain behind the same allowlist and approved numeric user.
	var reader int64
	for _, user := range v.Users {
		if user.Notifications {
			reader = user.ID
			break
		}
	}
	if reader == 0 && v.Reboot != nil && len(v.Users) > 0 {
		reader = v.Users[0].ID
	}
	if reader == 0 {
		return
	}
	observationCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	bootID, bootErr := readSystemUpdateBootID()
	if bootErr == nil && bootIDPattern.MatchString(bootID) {
		s.telegramNotify(observationCtx, &v, "online", bootID, "JustVoxel has returned online after a restart.", true)
		if v.Reboot != nil {
			s.telegramReconcileReboot(observationCtx, &v, reader, bootID)
			if v.Schema != 1 || !v.Enabled || v.Generation != generation {
				return
			}
		}
	} else if v.Reboot != nil && s.store.now().Sub(v.Reboot.RequestedAt) > 20*time.Minute {
		s.telegramFinishReboot(observationCtx, &v, "Reboot result could not be verified because boot identity is unavailable. Check System Update and appliance health in JustVoxel.")
	}
	records, err := instanceRuntimes.records()
	if err != nil {
		return
	}
	if s.operations != nil {
		journal, readErr := s.operations.currentMinecraftMaintenance()
		if readErr == nil && journal != nil && (journal.OperationType == operationTypeMinecraftUpdate || journal.OperationType == operationTypeMinecraftContainerUpdate) {
			for _, record := range records {
				if record.Runtime.ID == journal.InstanceID {
					text := "Minecraft update for " + journal.InstanceID + ": " + string(journal.State) + ". Review update progress and recovery in JustVoxel."
					s.telegramNotify(observationCtx, &v, "minecraft_update_operation:"+journal.InstanceID, journal.OperationID+":"+string(journal.State), text, false)
					break
				}
			}
		}
	}
	validKeys := map[string]bool{}
	for _, record := range records {
		id := record.Runtime.ID
		for _, user := range v.Users {
			for _, event := range []string{"server:", "backup:", "minecraft_update:", "container_update:", "minecraft_update_operation:"} {
				validKeys[strconv.FormatInt(user.ID, 10)+":"+event+id] = true
			}
		}
		state, err := instanceRuntimes.observe(observationCtx, id)
		if err == nil {
			s.telegramNotify(observationCtx, &v, "server:"+id, state.State, "Minecraft server "+id+": "+state.State, state.State != "failed")
		}
		code, raw := s.telegramInvoke(observationCtx, &v, reader, telegramCallback{Operation: "backup_status", InstanceID: id})
		if v.Schema != 1 || !v.Enabled || v.Generation != generation {
			return
		}
		if code < 400 {
			var backup instanceBackupStatus
			if json.Unmarshal(raw, &backup) == nil {
				signature := telegramSignature(telegramBody([]string{backup.State, backup.LastAttempt, backup.LastSuccess}))
				failure := backup.State == "failed" || backup.State == "interrupted" || backup.RestartRequired
				s.telegramNotify(observationCtx, &v, "backup:"+id, signature, "Backup for "+id+": "+backup.State+". Check JustVoxel for details.", !failure && backup.State != "succeeded")
			}
		}
		// Provider checks are spaced separately from minute-level lifecycle checks.
		key := "update_check:" + id
		last, _ := time.Parse(time.RFC3339, v.Seen[key])
		if s.store.now().Sub(last) < 30*time.Minute {
			continue
		}
		v.Seen[key] = s.store.now().UTC().Format(time.RFC3339)
		if s.store.telegramSave(v, nil, "") != nil {
			return
		}
		code, raw = s.telegramInvoke(observationCtx, &v, reader, telegramCallback{Operation: "version_status", InstanceID: id})
		if v.Schema != 1 || !v.Enabled || v.Generation != generation {
			return
		}
		if code < 400 {
			var version adminVersionStatus
			if json.Unmarshal(raw, &version) == nil && version.UpdateAvailable {
				value := version.Available + ":" + version.SelectedCandidate + ":" + version.ConfiguredVersion
				s.telegramNotify(observationCtx, &v, "minecraft_update:"+id, value, "Minecraft updates are available for "+id+". Review updates before applying.", false)
			}
		}
		code, raw = s.telegramInvoke(observationCtx, &v, reader, telegramCallback{Operation: "container_status", InstanceID: id})
		if v.Schema != 1 || !v.Enabled || v.Generation != generation {
			return
		}
		if code < 400 {
			var container minecraftContainerStatus
			if json.Unmarshal(raw, &container) == nil && container.State == "update_available" {
				s.telegramNotify(observationCtx, &v, "container_update:"+id, container.PlanFingerprint, "A Minecraft container update is available for "+id+".", false)
			}
		}
	}
	osRead := "system_updates"
	lastOSCheck, _ := time.Parse(time.RFC3339, v.Seen["os_update_check"])
	if s.store.now().Sub(lastOSCheck) >= 30*time.Minute {
		v.Seen["os_update_check"] = s.store.now().UTC().Format(time.RFC3339)
		if s.store.telegramSave(v, nil, "") != nil {
			return
		}
		osRead = "system_update_check"
	}
	code, raw := s.telegramInvoke(observationCtx, &v, reader, telegramCallback{Operation: osRead})
	if v.Schema != 1 || !v.Enabled || v.Generation != generation {
		return
	}
	if code < 400 {
		var status adminSystemUpdateStatus
		if json.Unmarshal(raw, &status) == nil {
			if status.Available != nil && status.Available.Digest != status.Running.Digest {
				s.telegramNotify(observationCtx, &v, "os_update", status.Available.Digest+":"+status.Available.Version, "A JustVoxel OS update is available. Downloading it and rebooting are separate actions.", false)
			} else if status.CheckState == "update_available" {
				// composefs registry diffs can prove availability without publishing
				// an exact version/digest in the check output.
				s.telegramNotify(observationCtx, &v, "os_update", "registry-diff:"+status.Running.Digest, "A JustVoxel OS update is available. Downloading it and rebooting are separate actions.", false)
			}
			if status.Staged != nil && status.RebootRequired {
				s.telegramNotify(observationCtx, &v, "os_staged", status.Staged.Digest, "A JustVoxel OS update is downloaded or staged. Review reboot options to activate it.", false)
			}
		}
	}
	// Bound historical observations for removed/revoked identities and servers.
	for key := range v.Seen {
		if before, _, found := strings.Cut(key, ":"); found {
			if uid, err := strconv.ParseInt(before, 10, 64); err == nil {
				if _, approved := v.user(uid); !approved {
					delete(v.Seen, key)
					continue
				}
			}
		}
		if strings.Contains(key, ":server:") || strings.Contains(key, ":backup:") || strings.Contains(key, ":minecraft_update:") || strings.Contains(key, ":container_update:") || strings.Contains(key, ":minecraft_update_operation:") {
			if !validKeys[key] {
				delete(v.Seen, key)
			}
		} else if strings.HasPrefix(key, "update_check:") {
			exists := false
			for _, record := range records {
				if key == "update_check:"+record.Runtime.ID {
					exists = true
					break
				}
			}
			if !exists {
				delete(v.Seen, key)
			}
		}
	}
	_ = s.store.telegramSave(v, nil, "")
}

func telegramRebootResult(p telegramReboot, bootID string, status adminSystemUpdateStatus, healthy bool, now time.Time) (string, bool) {
	if bootID == p.BootID {
		if now.Sub(p.RequestedAt) > 20*time.Minute {
			return "Reboot could not be verified: the appliance is still on the previous boot. Review the reboot workflow in JustVoxel.", true
		}
		return "", false
	}
	if p.Expected != "" && status.Running.Digest != p.Expected {
		return "The appliance returned online, but the requested OS update did not boot successfully. Review System Update and its recovery options in JustVoxel.", true
	}
	if p.Expected != "" && status.ActivationFailed {
		return "The expected OS image is booted, but update activation evidence requires attention. Update success has not been confirmed; review System Update recovery in JustVoxel.", true
	}
	if !healthy {
		if now.Sub(p.RequestedAt) > 20*time.Minute {
			return "The appliance returned online, but health verification did not pass. OS update success has not been confirmed. Review JustVoxel health and recovery.", true
		}
		return "", false
	}
	if p.Expected != "" {
		return "OS update confirmed: the expected deployment is booted and appliance health checks passed.", true
	}
	return "Reboot confirmed: the appliance returned on a new boot and health checks passed.", true
}
func (s *server) telegramReconcileReboot(ctx context.Context, v *telegramState, reader int64, bootID string) {
	pending := v.Reboot
	if pending == nil {
		return
	}
	status, err := s.collectSystemUpdateStatus(ctx)
	if err != nil {
		if s.store.now().Sub(pending.RequestedAt) > 20*time.Minute {
			s.telegramFinishReboot(ctx, v, "Reboot and OS update result could not be verified because deployment status is unavailable. Check System Update and appliance health in JustVoxel.")
		}
		return
	}
	if pending.Operation == "update_reboot" && bootID == pending.BootID {
		checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		raw, code, checkErr := runAdminSystemUpdateRebootHelper(checkCtx, "status")
		cancel()
		workflow, valid := decodeAdminSystemUpdateRebootResponse(raw)
		if checkErr == nil && code == 0 && valid && workflow.State == "failed" {
			message := "The update reboot workflow failed before restarting. Review System Update and backup status in JustVoxel; update success has not been confirmed."
			s.telegramFinishReboot(ctx, v, message)
			return
		}
	}
	healthy := false
	if bootID != pending.BootID {
		// Reuse authoritative appliance/instance validation, discard its raw report.
		actor := telegramActor(reader)
		request, _ := http.NewRequestWithContext(context.WithValue(ctx, gatewayPrincipalKey{}, actor), "GET", "http://telegram/v1/admin/validation", nil)
		capture := &gatewayResponse{header: make(http.Header), status: 200}
		s.telegramMu.Unlock()
		s.adminValidation(capture, request)
		s.telegramMu.Lock()
		latest, stateErr := s.store.telegramRead()
		if stateErr != nil {
			*v = telegramState{}
			return
		}
		*v = latest
		if v.Reboot == nil || !v.Reboot.RequestedAt.Equal(pending.RequestedAt) {
			return
		}
		var health struct {
			OK bool `json:"ok"`
		}
		healthy = capture.status == 200 && !capture.overflow && json.Unmarshal(capture.body.Bytes(), &health) == nil && health.OK
		if healthy {
			_, err := instanceRuntimes.command(ctx, "systemctl", "is-active", "--quiet", "justvoxel-management.service", "justvoxel-webui.service")
			healthy = err == nil
		}
	}
	if result, done := telegramRebootResult(*pending, bootID, status, healthy, s.store.now()); done {
		s.telegramFinishReboot(ctx, v, result)
	}
}

func (s *server) telegramFinishReboot(ctx context.Context, v *telegramState, message string) {
	if v.Reboot == nil {
		return
	}
	event := v.Reboot.RequestedAt.UTC().Format(time.RFC3339Nano)
	s.telegramNotify(ctx, v, "reboot_result", event, message, false)
	v.OperationStatus = message
	v.Reboot = nil
	_ = s.store.telegramSave(*v, nil, "")
}
