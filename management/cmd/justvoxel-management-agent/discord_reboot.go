package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

// This singleton contains verification evidence only. It is not a reboot engine
// or a notification queue; authoritative handlers still own reboot execution.
type discordRebootState struct {
	Generation                                                    int64
	GuildID, UserID, ChannelID, Operation, BootID, ExpectedDigest string
	RequestedAt                                                   time.Time
	State                                                         string
}

func (a discordRebootState) valid(now time.Time) bool {
	return a.Generation >= 0 && validDiscordSnowflake(a.GuildID) && validDiscordSnowflake(a.UserID) && validDiscordSnowflake(a.ChannelID) && (a.Operation == "reboot" || a.Operation == "update_reboot") && bootIDPattern.MatchString(a.BootID) && (a.ExpectedDigest == "" || operationFingerprintPattern.MatchString(a.ExpectedDigest)) && (a.Operation != "update_reboot" || a.ExpectedDigest != "") && a.RequestedAt.Unix() > 0 && !a.RequestedAt.After(now.Add(time.Minute)) && (a.State == "requesting" || a.State == "accepted" || a.State == "unverified")
}
func (s *webUIStore) discordRebootRead() (*discordRebootState, error) {
	if s == nil || s.db == nil {
		return nil, errDiscordState
	}
	var a discordRebootState
	var requested int64
	err := s.db.QueryRow(`SELECT discord_generation,guild_id,user_id,channel_id,operation,boot_id,expected_digest,requested_at,state FROM discord_reboot_state WHERE id=1`).Scan(&a.Generation, &a.GuildID, &a.UserID, &a.ChannelID, &a.Operation, &a.BootID, &a.ExpectedDigest, &requested, &a.State)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, errDiscordState
	}
	a.RequestedAt = time.Unix(requested, 0).UTC()
	if !a.valid(s.now()) {
		return nil, errDiscordState
	}
	return &a, nil
}
func (s *webUIStore) discordRebootCreate(a discordRebootState) error {
	if s == nil || s.db == nil || !a.valid(s.now()) || a.State != "requesting" {
		return errDiscordState
	}
	tx, err := s.db.Begin()
	if err != nil {
		return errDiscordState
	}
	defer tx.Rollback()
	var raw string
	var v discordState
	if tx.QueryRow(`SELECT record_json FROM discord_state WHERE id=1`).Scan(&raw) != nil || json.Unmarshal([]byte(raw), &v) != nil || v.Generation != a.Generation || v.GuildID != a.GuildID || !v.Enabled || !v.Verified || !v.administrator(a.UserID) {
		return errDiscordState
	}
	_, err = tx.Exec(`INSERT INTO discord_reboot_state(id,discord_generation,guild_id,user_id,channel_id,operation,boot_id,expected_digest,requested_at,state) VALUES(1,?,?,?,?,?,?,?,?,?)`, a.Generation, a.GuildID, a.UserID, a.ChannelID, a.Operation, a.BootID, a.ExpectedDigest, a.RequestedAt.Unix(), a.State)
	if err != nil || tx.Commit() != nil {
		return errDiscordState
	}
	return nil
}

// Conditional transitions cannot resurrect evidence cleared by a trust change.
func (s *webUIStore) discordRebootFinish(a discordRebootState, state string) error {
	if s == nil || s.db == nil {
		return errDiscordState
	}
	var result sql.Result
	var err error
	if state == "" {
		result, err = s.db.Exec(`DELETE FROM discord_reboot_state WHERE id=1 AND discord_generation=? AND guild_id=? AND boot_id=? AND requested_at=?`, a.Generation, a.GuildID, a.BootID, a.RequestedAt.Unix())
	} else {
		if state != "accepted" && state != "unverified" {
			return errDiscordState
		}
		result, err = s.db.Exec(`UPDATE discord_reboot_state SET state=? WHERE id=1 AND discord_generation=? AND guild_id=? AND boot_id=? AND requested_at=?`, state, a.Generation, a.GuildID, a.BootID, a.RequestedAt.Unix())
	}
	if err != nil {
		return errDiscordState
	}
	n, err := result.RowsAffected()
	if err != nil || n != 1 {
		return errDiscordState
	}
	return nil
}

func (s *server) discordRebootLocked(ctx context.Context, rt *discordRuntime, a discordPendingAction) discordPanel {
	acceptedTrust, authorized := s.discordAdvancedAuthorized(rt, a.Generation, a.GuildID, a.UserID)
	if !authorized {
		return discordFailure()
	}
	pending, err := s.store.discordRebootRead()
	if err != nil || pending != nil {
		return discordPanel{text: "A previous reboot request needs verification, or reboot evidence is unavailable. Review JustVoxel before retrying."}
	}
	boot, err := readSystemUpdateBootID()
	if err != nil || !bootIDPattern.MatchString(boot) {
		return discordPanel{text: "Boot identity unavailable; reboot was not requested."}
	}
	// Collect private staged semantics for ordinary reboot exactly as Telegram does.
	s.discordMu.Unlock()
	status, err := s.collectSystemUpdateStatus(ctx)
	s.discordMu.Lock()
	if err != nil {
		return discordPanel{text: "Deployment status unavailable; reboot was not requested."}
	}
	if _, ok := s.discordAdvancedAuthorized(rt, a.Generation, a.GuildID, a.UserID); !ok {
		return discordFailure()
	}
	expected := ""
	if a.Operation == "update_reboot" {
		if status.Staged == nil || !status.RebootRequired || status.Staged.Digest != a.StagedDigest || !operationFingerprintPattern.MatchString(a.StagedDigest) {
			return discordPanel{text: "Staged update changed. Review reboot options again."}
		}
		expected = a.StagedDigest
	} else if status.Staged != nil && !status.stagedDownloadOnly {
		expected = status.Staged.Digest
	}
	record := discordRebootState{Generation: a.Generation, GuildID: a.GuildID, UserID: a.UserID, ChannelID: a.ChannelID, Operation: a.Operation, BootID: boot, ExpectedDigest: expected, RequestedAt: s.store.now().UTC().Truncate(time.Second), State: "requesting"}
	if s.store.discordRebootCreate(record) != nil {
		return discordPanel{text: "Reboot verification evidence could not be saved; reboot was not requested."}
	}
	code, raw := s.discordAdvancedInvokeLocked(ctx, rt, a.GuildID, a.UserID, a.Operation, a)
	var outcome struct {
		Accepted             bool   `json:"accepted"`
		ConfirmationRequired bool   `json:"confirmation_required"`
		Reason               string `json:"reason"`
		State                string `json:"state"`
	}
	decoded := json.Unmarshal(raw, &outcome) == nil
	state := "unverified"
	if decoded && code < 400 && outcome.Accepted {
		state = "accepted"
	} else if (decoded && (outcome.ConfirmationRequired || outcome.State == "failed")) || (code >= 400 && code < 500) {
		state = ""
	}
	// Enable/disable can fence the interaction while retaining the same durable
	// trust. Finish the accepted attempt against that retained generation too.
	latest, trustErr := s.store.discordRead()
	if trustErr == nil && latest.Token == acceptedTrust.Token && latest.GuildID == record.GuildID {
		record.Generation = latest.Generation
	}
	if s.store.discordRebootFinish(record, state) != nil {
		return discordPanel{text: "Reboot result is not confirmed. Review System Update and appliance health in JustVoxel."}
	}
	if _, ok := s.discordAdvancedAuthorized(rt, a.Generation, a.GuildID, a.UserID); !ok {
		return discordFailure()
	}
	if decoded && outcome.ConfirmationRequired && outcome.Reason == "players_online" && !a.PlayersConfirmed {
		a.PlayersConfirmed = true
		return discordAdvancedConfirm(rt, a, discordAdvancedPreview(a))
	}
	if state == "unverified" {
		return discordPanel{text: "Reboot result will be reconciled from boot and deployment evidence. Open JustVoxel to review current state."}
	}
	if state == "" {
		return discordAdvancedFailure()
	}
	return discordPanel{text: "Reboot request accepted. Return to service and deployment health will be verified after restart; success is not yet confirmed."}
}

// Called only by the existing runtime lifecycle, with discordMu held. Frequency
// and concurrency are appliance-wide so generation changes cannot overlap work.
func (s *server) discordReconcileRebootLocked(rt *discordRuntime, v discordState) {
	if s.discordRebootReconciling || time.Now().Before(s.discordRebootAfter) || !s.discordUsableLocked(rt, v, v.GuildID) {
		return
	}
	s.discordRebootAfter = time.Now().Add(time.Minute)
	pending, err := s.store.discordRebootRead()
	if err != nil || pending == nil || pending.Generation != v.Generation || pending.GuildID != v.GuildID {
		return
	}
	s.discordRebootReconciling = true
	go func(a discordRebootState) {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		s.discordMu.Lock()
		defer s.discordMu.Unlock()
		defer func() { s.discordRebootReconciling = false }()
		current, readErr := s.store.discordRead()
		if readErr != nil || current.Generation != a.Generation || !s.discordUsableLocked(rt, current, a.GuildID) {
			return
		}
		// All potentially slow reads release the Discord settings lock. The final
		// conditional delete plus trust fence protects a replacement trust context.
		s.discordMu.Unlock()
		boot, bootErr := readSystemUpdateBootID()
		var status adminSystemUpdateStatus
		var statusErr error
		if bootErr == nil && bootIDPattern.MatchString(boot) && boot != a.BootID {
			status, statusErr = s.collectSystemUpdateStatus(ctx)
		}
		s.discordMu.Lock()
		current, readErr = s.store.discordRead()
		if readErr != nil || current.Generation != a.Generation || !s.discordUsableLocked(rt, current, a.GuildID) {
			return
		}
		message := ""
		done := false
		overdue := s.store.now().Sub(a.RequestedAt) > 20*time.Minute
		if bootErr != nil || !bootIDPattern.MatchString(boot) {
			if overdue {
				message = "Reboot could not be verified: boot identity is unavailable."
				done = true
			}
		} else if boot == a.BootID {
			if a.Operation == "update_reboot" && current.administrator(a.UserID) {
				code, raw := s.discordRebootObservationLocked(ctx, rt, a, true)
				var workflow adminSystemUpdateRebootResponse
				if code == 200 && json.Unmarshal(raw, &workflow) == nil && workflow.State == "failed" {
					message = "The update reboot workflow failed before restarting. Review System Update in JustVoxel; update success has not been confirmed."
					done = true
				}
			}
			if !done && overdue {
				message = "Reboot could not be verified: appliance is still on the previous boot."
				done = true
			}
		} else if statusErr != nil {
			if overdue {
				message = "Reboot could not be verified: deployment status is unavailable. Review System Update in JustVoxel."
				done = true
			}
		} else {
			healthy := false
			if current.administrator(a.UserID) {
				code, raw := s.discordRebootObservationLocked(ctx, rt, a, false)
				var health struct {
					OK bool `json:"ok"`
				}
				healthy = code == 200 && json.Unmarshal(raw, &health) == nil && health.OK
			}
			message, done = discordRebootResult(a, boot, status, healthy, s.store.now())
		}
		if done && s.store.discordRebootFinish(a, "") == nil {
			s.discordAdvancedNotifyLocked(a.Generation, a.GuildID, a.ChannelID, message)
		}
	}(*pending)
}

// Internal authenticated observations used only by reboot verification. No
// component can choose a handler, path or request body here.
func (s *server) discordRebootObservationLocked(ctx context.Context, rt *discordRuntime, a discordRebootState, workflow bool) (int, []byte) {
	v, ok := s.discordAdvancedAuthorized(rt, a.Generation, a.GuildID, a.UserID)
	if !ok {
		return 403, nil
	}
	path := "/v1/admin/validation"
	handler := s.adminValidation
	if workflow {
		path = "/v1/admin/system/update-reboot"
		handler = s.adminSystemUpdateRebootStatus
	}
	request, _ := http.NewRequestWithContext(context.WithValue(ctx, gatewayPrincipalKey{}, discordActor(v, a.UserID)), "GET", "http://discord"+path, nil)
	capture := &gatewayResponse{header: make(http.Header), status: 200}
	s.discordMu.Unlock()
	handler(capture, request)
	s.discordMu.Lock()
	if capture.overflow || !json.Valid(capture.body.Bytes()) {
		return 502, nil
	}
	return capture.status, append([]byte(nil), capture.body.Bytes()...)
}
func discordRebootResult(a discordRebootState, boot string, status adminSystemUpdateStatus, healthy bool, now time.Time) (string, bool) {
	if boot == a.BootID {
		if now.Sub(a.RequestedAt) > 20*time.Minute {
			return "Reboot could not be verified: appliance is still on the previous boot.", true
		}
		return "", false
	}
	if a.ExpectedDigest != "" && status.Running.Digest != a.ExpectedDigest {
		return "Reboot verification failed: the expected OS deployment did not boot. Review System Update in JustVoxel.", true
	}
	if a.ExpectedDigest != "" && status.ActivationFailed {
		return "OS update requires attention: activation failed. Update success has not been confirmed. Review System Update in JustVoxel.", true
	}
	if !healthy {
		if now.Sub(a.RequestedAt) > 20*time.Minute {
			return "Reboot could not be verified: appliance health checks did not pass. Review JustVoxel health and recovery.", true
		}
		return "", false
	}
	if a.ExpectedDigest != "" {
		return "OS update confirmed: the expected deployment is booted and appliance health checks passed.", true
	}
	return "Reboot confirmed: the appliance returned on a new boot and appliance health checks passed.", true
}
