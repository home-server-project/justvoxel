package main

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/disgoorg/disgo/discord"
)

func rebootEvidence(v discordState, now time.Time) discordRebootState {
	return discordRebootState{Generation: v.Generation, GuildID: v.GuildID, UserID: "777", ChannelID: "999", Operation: "reboot", BootID: activationOldBoot, RequestedAt: now.UTC().Truncate(time.Second), State: "requesting"}
}
func TestDiscordRebootMigrationV10AndPersistence(t *testing.T) {
	s, _, v := discordBoundFixture(t)
	discordApproveFixture(t, s, &v, "777")
	user, err := s.store.createWebUser("ExistingUser", webUserRoleOperator, "existing-user-password")
	if err != nil {
		t.Fatal(err)
	}
	access := accessCreateTest(t, s, v, "778", "jv-a1b2c3", "java")
	var path string
	if err = s.store.db.QueryRow(`SELECT file FROM pragma_database_list WHERE name='main'`).Scan(&path); err != nil {
		t.Fatal(err)
	}
	before, err := s.store.discordRead()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.store.db.Exec(`DROP TABLE web_user_two_factor`); err != nil {
		t.Fatal(err)
	}
	if _, err = s.store.db.Exec(`DROP TABLE discord_reboot_state`); err != nil {
		t.Fatal(err)
	}
	if _, err = s.store.db.Exec(`DROP TABLE routeros_state`); err != nil {
		t.Fatal(err)
	}
	if _, err = s.store.db.Exec(`DELETE FROM schema_migrations WHERE version>=10`); err != nil {
		t.Fatal(err)
	}
	if err = s.store.close(); err != nil {
		t.Fatal(err)
	}
	upgraded, err := openWebUIStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer upgraded.close()
	s.store = upgraded
	var name string
	if err = upgraded.db.QueryRow(`SELECT name FROM schema_migrations WHERE version=10`).Scan(&name); err != nil || name != "discord_reboot_state" {
		t.Fatal("v10 migration", name, err)
	}
	after, err := upgraded.discordRead()
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("existing Discord state changed", err)
	}
	users, err := upgraded.listWebUsers()
	if err != nil || len(users) != 1 || users[0].ID != user.ID {
		t.Fatal("v1-v9 identity lost", err)
	}
	requests, err := upgraded.discordAccessRead()
	if err != nil || len(requests) != 1 || requests[0].RequestID != access.RequestID {
		t.Fatal("v9 access lost", err)
	}
	a := rebootEvidence(v, upgraded.now())
	if err = upgraded.discordRebootCreate(a); err != nil {
		t.Fatal(err)
	}
	if err = upgraded.discordRebootCreate(a); err == nil {
		t.Fatal("multiple pending appliance reboots")
	}
	if err = upgraded.close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := openWebUIStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.close()
	saved, err := reopened.discordRebootRead()
	if err != nil || saved == nil || !reflect.DeepEqual(a, *saved) {
		t.Fatal("reboot did not survive restart", saved, err)
	}
	columns, err := reopened.db.Query(`SELECT name FROM pragma_table_info('discord_reboot_state')`)
	if err != nil {
		t.Fatal(err)
	}
	defer columns.Close()
	count := 0
	for columns.Next() {
		var column string
		if columns.Scan(&column) != nil {
			t.Fatal("column")
		}
		count++
		if strings.Contains(column, "token") || strings.Contains(column, "message") || strings.Contains(column, "output") {
			t.Fatal("private data column", column)
		}
	}
	if count != 10 {
		t.Fatal("unexpected reboot schema", count)
	}
}
func TestDiscordRebootMalformedEvidenceFailsClosed(t *testing.T) {
	for _, bad := range []string{"boot_id='invalid'", "expected_digest='invalid'", "guild_id='invalid'", "user_id='invalid'", "channel_id='invalid'", "requested_at=-1", "discord_generation=-1"} {
		t.Run(bad, func(t *testing.T) {
			s, _, v := discordBoundFixture(t)
			discordApproveFixture(t, s, &v, "777")
			a := rebootEvidence(v, s.store.now())
			if err := s.store.discordRebootCreate(a); err != nil {
				t.Fatal(err)
			}
			if _, err := s.store.db.Exec(`UPDATE discord_reboot_state SET ` + bad); err != nil {
				t.Fatal(err)
			}
			if _, err := s.store.discordRebootRead(); err == nil {
				t.Fatal("malformed record accepted")
			}
		})
	}
}
func TestDiscordRebootTrustRetentionAndAtomicClearing(t *testing.T) {
	for _, change := range []string{"disable", "enable", "same_guild", "credential", "different_guild", "disconnect", "factory_revocation"} {
		t.Run(change, func(t *testing.T) {
			s, _, v := discordBoundFixture(t)
			discordApproveFixture(t, s, &v, "777")
			a := rebootEvidence(v, s.store.now())
			if err := s.store.discordRebootCreate(a); err != nil {
				t.Fatal(err)
			}
			action := "guild"
			retain := false
			switch change {
			case "disable":
				v.Enabled = false
				v.Generation++
				action = "enabled"
				retain = true
			case "enable":
				v.Generation++
				action = "enabled"
				retain = true
			case "same_guild":
				retain = true
			case "credential":
				v.Token = "replacement"
				v.Generation++
				action = "credential"
			case "different_guild":
				v.GuildID = "222"
				v.Administrators = nil
				v.Generation++
			case "disconnect":
				v = discordState{Schema: 1, Status: "disconnected"}
				action = "disconnect"
			case "factory_revocation":
				v = discordState{Schema: 1, Status: "disconnected"}
				action = "factory_revocation"
			}
			if err := s.store.discordSave(v, nil, action); err != nil {
				t.Fatal(err)
			}
			saved, err := s.store.discordRebootRead()
			if err != nil {
				t.Fatal(err)
			}
			if retain {
				if saved == nil || saved.Generation != v.Generation {
					t.Fatal("trusted evidence lost")
				}
			} else if saved != nil {
				t.Fatal("evidence crossed trust boundary")
			}
		})
	}
	for _, action := range []string{"credential", "disconnect", "guild", "factory_revocation"} {
		t.Run("clear_failure_"+action, func(t *testing.T) {
			s, _, v := discordBoundFixture(t)
			discordApproveFixture(t, s, &v, "777")
			a := rebootEvidence(v, s.store.now())
			if err := s.store.discordRebootCreate(a); err != nil {
				t.Fatal(err)
			}
			before, err := s.store.discordRead()
			if err != nil {
				t.Fatal(err)
			}
			if _, err := s.store.db.Exec(`CREATE TRIGGER reject_reboot_clear BEFORE DELETE ON discord_reboot_state BEGIN SELECT RAISE(ABORT,'blocked'); END`); err != nil {
				t.Fatal(err)
			}
			changed := v
			if action == "guild" {
				changed.GuildID = "222"
				changed.Administrators = nil
			} else {
				changed.Token = "replacement"
			}
			if err := s.store.discordSave(changed, nil, action); err == nil {
				t.Fatal("trust changed without clearing evidence")
			}
			current, err := s.store.discordRead()
			if err != nil || !reflect.DeepEqual(current, before) {
				t.Fatal("non-atomic trust transition")
			}
		})
	}
}
func TestDiscordRebootResultEvidence(t *testing.T) {
	now := time.Now()
	v := discordVerifiedState()
	v.GuildID = "111"
	a := rebootEvidence(v, now)
	expected := "sha256:" + strings.Repeat("a", 64)
	cases := []struct {
		name, boot, expected string
		age                  time.Duration
		healthy, activation  bool
		running              string
		done                 bool
		contains             string
	}{
		{"same_under", activationOldBoot, "", time.Minute, false, false, "", false, ""},
		{"same_over", activationOldBoot, "", 21 * time.Minute, false, false, "", true, "previous boot"},
		{"new_healthy", activationNewBoot, "", time.Minute, true, false, "", true, "Reboot confirmed"},
		{"update_healthy", activationNewBoot, expected, time.Minute, true, false, expected, true, "OS update confirmed"},
		{"wrong_digest", activationNewBoot, expected, time.Minute, true, false, "sha256:" + strings.Repeat("b", 64), true, "did not boot"},
		{"activation_failed", activationNewBoot, expected, time.Minute, true, true, expected, true, "requires attention"},
		{"health_wait", activationNewBoot, "", time.Minute, false, false, "", false, ""},
		{"health_expired", activationNewBoot, "", 21 * time.Minute, false, false, "", true, "could not be verified"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a.ExpectedDigest = tc.expected
			a.RequestedAt = now.Add(-tc.age)
			status := adminSystemUpdateStatus{Running: adminSystemUpdateDeployment{Digest: tc.running}, ActivationFailed: tc.activation}
			message, done := discordRebootResult(a, tc.boot, status, tc.healthy, now)
			if done != tc.done || !strings.Contains(message, tc.contains) {
				t.Fatal(message, done)
			}
		})
	}
}
func TestDiscordRebootAcceptanceAndSecondPlayerConfirmation(t *testing.T) {
	s, rt, v, _ := advancedOSFixture(t)
	oldBoot, oldAction := readSystemUpdateBootID, runAdminSystemActionHelper
	t.Cleanup(func() { readSystemUpdateBootID = oldBoot; runAdminSystemActionHelper = oldAction })
	readSystemUpdateBootID = func() (string, error) { return activationOldBoot, nil }
	calls := 0
	runAdminSystemActionHelper = func(_ context.Context, args ...string) ([]byte, int, error) {
		calls++
		pending, err := s.store.discordRebootRead()
		if err != nil || pending == nil || pending.State != "requesting" {
			t.Error("reboot invoked before durable save", err)
		}
		confirmed := false
		for _, arg := range args {
			if arg == "--confirm-players" {
				confirmed = true
			}
		}
		if calls == 1 {
			if confirmed {
				t.Error("first attempt already interrupted players")
			}
			return []byte(`{"ok":false,"action":"reboot","state":"confirmation_required","confirmation_required":true,"reason":"players_online"}`), 10, errors.New("players")
		}
		if !confirmed {
			t.Error("second confirmation not forwarded")
		}
		return []byte(`{"ok":true,"action":"reboot","state":"requested","accepted":true}`), 0, nil
	}
	a := discordPendingAction{Generation: v.Generation, GuildID: v.GuildID, UserID: "777", ChannelID: "999", Operation: "reboot"}
	s.discordMu.Lock()
	review := discordAdvancedConfirm(rt, a, discordAdvancedPreview(a))
	s.discordMu.Unlock()
	first := advancedNonce(t, review, "adv_confirm")
	p := advancedHandle(t, s, rt, v, "777", "adv_confirm", first)
	second := advancedNonce(t, p, "adv_confirm")
	if !strings.Contains(p.text, "Players are online") || !rt.actions[second].PlayersConfirmed {
		t.Fatal("second confirmation missing", p.text)
	}
	pending, err := s.store.discordRebootRead()
	if err != nil || pending != nil {
		t.Fatal("unaccepted reboot evidence retained", err)
	}
	p = advancedHandle(t, s, rt, v, "777", "adv_confirm", second)
	pending, err = s.store.discordRebootRead()
	if err != nil || pending == nil || pending.State != "accepted" || pending.ExpectedDigest != "" || calls != 2 || strings.Contains(p.text, "Reboot confirmed") {
		t.Fatal("acceptance is not verified success", p.text, pending, err)
	}
}
func TestDiscordRebootEvidenceSaveFailurePreventsHandler(t *testing.T) {
	s, rt, v, _ := advancedOSFixture(t)
	oldBoot, oldHelper := readSystemUpdateBootID, runAdminSystemActionHelper
	t.Cleanup(func() { readSystemUpdateBootID = oldBoot; runAdminSystemActionHelper = oldHelper })
	readSystemUpdateBootID = func() (string, error) { return activationOldBoot, nil }
	called := false
	runAdminSystemActionHelper = func(context.Context, ...string) ([]byte, int, error) { called = true; return nil, 0, nil }
	if _, err := s.store.db.Exec(`CREATE TRIGGER reject_reboot_save BEFORE INSERT ON discord_reboot_state BEGIN SELECT RAISE(ABORT,'blocked'); END`); err != nil {
		t.Fatal(err)
	}
	a := discordPendingAction{Generation: v.Generation, GuildID: v.GuildID, UserID: "777", ChannelID: "999", Operation: "reboot"}
	s.discordMu.Lock()
	p := s.discordRebootLocked(context.Background(), rt, a)
	s.discordMu.Unlock()
	if called || !strings.Contains(p.text, "was not requested") {
		t.Fatal("unsafe reboot", p.text)
	}
}
func TestDiscordUpdateRebootDigestAndOptions(t *testing.T) {
	s, rt, v, _ := advancedOSFixture(t)
	oldBoot, oldHelper := readSystemUpdateBootID, runAdminSystemUpdateRebootHelper
	t.Cleanup(func() { readSystemUpdateBootID = oldBoot; runAdminSystemUpdateRebootHelper = oldHelper })
	readSystemUpdateBootID = func() (string, error) { return activationOldBoot, nil }
	called := false
	runAdminSystemUpdateRebootHelper = func(_ context.Context, args ...string) ([]byte, int, error) {
		called = true
		joined := strings.Join(args, " ")
		for _, want := range []string{"--warning-seconds=10", "--backup-minecraft", "--staged-digest=sha256:" + strings.Repeat("b", 64)} {
			if !strings.Contains(joined, want) {
				t.Error("lost reviewed option", want)
			}
		}
		if strings.Contains(joined, "--confirm-players") {
			t.Error("unconfirmed players")
		}
		return []byte(`{"ok":true,"state":"accepted","accepted":true}`), 0, nil
	}
	a := discordPendingAction{Generation: v.Generation, GuildID: v.GuildID, UserID: "777", ChannelID: "999", Operation: "update_reboot", WarningSeconds: 10, BackupMinecraft: true, StagedDigest: "sha256:" + strings.Repeat("c", 64)}
	s.discordMu.Lock()
	p := s.discordRebootLocked(context.Background(), rt, a)
	s.discordMu.Unlock()
	if called || !strings.Contains(p.text, "changed") {
		t.Fatal("changed staged digest accepted", p.text)
	}
	a.StagedDigest = "sha256:" + strings.Repeat("b", 64)
	s.discordMu.Lock()
	p = s.discordRebootLocked(context.Background(), rt, a)
	s.discordMu.Unlock()
	pending, err := s.store.discordRebootRead()
	if !called || err != nil || pending == nil || pending.ExpectedDigest != a.StagedDigest {
		t.Fatal("update reboot binding", p.text, err)
	}
}
func TestDiscordRebootAmbiguousResponseRetainsEvidence(t *testing.T) {
	s, rt, v, _ := advancedOSFixture(t)
	oldBoot, oldHelper := readSystemUpdateBootID, runAdminSystemActionHelper
	t.Cleanup(func() { readSystemUpdateBootID = oldBoot; runAdminSystemActionHelper = oldHelper })
	readSystemUpdateBootID = func() (string, error) { return activationOldBoot, nil }
	runAdminSystemActionHelper = func(context.Context, ...string) ([]byte, int, error) {
		return []byte(`not JSON private /path`), -1, errors.New("transport")
	}
	a := discordPendingAction{Generation: v.Generation, GuildID: v.GuildID, UserID: "777", ChannelID: "999", Operation: "reboot"}
	s.discordMu.Lock()
	p := s.discordRebootLocked(context.Background(), rt, a)
	s.discordMu.Unlock()
	saved, err := s.store.discordRebootRead()
	if err != nil || saved == nil || saved.State != "unverified" || !strings.Contains(p.text, "reconciled") || strings.Contains(p.text, "/path") {
		t.Fatal("ambiguous reboot lost", p.text, err)
	}
}
func TestDiscordRebootRuntimeReconciliation(t *testing.T) {
	for _, test := range []string{"pending", "expired", "workflow_failed", "notification_failed", "trust_changed"} {
		t.Run(test, func(t *testing.T) {
			s, rt, v, fake := advancedOSFixture(t)
			oldBoot, oldHelper := readSystemUpdateBootID, runAdminSystemUpdateRebootHelper
			t.Cleanup(func() { readSystemUpdateBootID = oldBoot; runAdminSystemUpdateRebootHelper = oldHelper })
			readSystemUpdateBootID = func() (string, error) { return activationOldBoot, nil }
			runAdminSystemUpdateRebootHelper = func(context.Context, ...string) ([]byte, int, error) {
				return []byte(`{"ok":false,"state":"failed","message":"PRIVATE"}`), 0, nil
			}
			a := rebootEvidence(v, s.store.now())
			if test != "pending" {
				a.RequestedAt = a.RequestedAt.Add(-21 * time.Minute)
			}
			if test == "workflow_failed" {
				a.Operation = "update_reboot"
				a.ExpectedDigest = "sha256:" + strings.Repeat("b", 64)
				a.RequestedAt = s.store.now().Truncate(time.Second)
			}
			if err := s.store.discordRebootCreate(a); err != nil {
				t.Fatal(err)
			}
			if test == "trust_changed" {
				v.Generation++
				discordSaveTestState(t, s, v)
			}
			fake.fail = test == "notification_failed"
			s.discordMu.Lock()
			s.discordReconcileRebootLocked(rt, v)
			s.discordMu.Unlock()
			discordEventually(t, func() bool { s.discordMu.Lock(); defer s.discordMu.Unlock(); return !s.discordRebootReconciling })
			saved, err := s.store.discordRebootRead()
			if err != nil {
				t.Fatal(err)
			}
			if test == "pending" || test == "trust_changed" {
				if saved == nil {
					t.Fatal("pending evidence removed")
				}
				select {
				case <-fake.messages:
					t.Fatal("unverified or stale notification")
				default:
				}
				return
			}
			if saved != nil {
				t.Fatal("terminal row retained")
			}
			select {
			case m := <-fake.messages:
				if m.AllowedMentions == nil || !reflect.DeepEqual(*m.AllowedMentions, discord.AllowedMentions{}) || strings.Contains(m.Content, "PRIVATE") {
					t.Fatal("unsafe reboot notification")
				}
			default:
				t.Fatal("missing terminal notification")
			}
			s.discordMu.Lock()
			s.discordRebootAfter = time.Time{}
			s.discordReconcileRebootLocked(rt, v)
			s.discordMu.Unlock()
			select {
			case <-fake.messages:
				t.Fatal("duplicate notification")
			default:
			}
		})
	}
}
func TestDiscordRebootNotificationTrustFence(t *testing.T) {
	s, _, v, fake := advancedOSFixture(t)
	for _, change := range []string{"generation", "guild", "disabled"} {
		changed := v
		switch change {
		case "generation":
			changed.Generation++
		case "guild":
			changed.GuildID = "222"
			changed.Administrators = nil
		case "disabled":
			changed.Enabled = false
		}
		discordSaveTestState(t, s, changed)
		s.discordMu.Lock()
		s.discordAdvancedNotifyLocked(v.Generation, v.GuildID, "999", "Fixed result")
		s.discordMu.Unlock()
		select {
		case <-fake.messages:
			t.Fatal("notification crossed", change)
		default:
		}
		discordSaveTestState(t, s, v)
	}
}
func TestDiscordRebootBoundedObservationAndTerminalDelete(t *testing.T) {
	s, _, v := discordBoundFixture(t)
	discordApproveFixture(t, s, &v, "777")
	a := rebootEvidence(v, s.store.now())
	if err := s.store.discordRebootCreate(a); err != nil {
		t.Fatal(err)
	}
	for _, state := range []string{"accepted", "unverified"} {
		if err := s.store.discordRebootFinish(a, state); err != nil {
			t.Fatal(err)
		}
		saved, err := s.store.discordRebootRead()
		if err != nil || saved == nil || saved.State != state {
			t.Fatal(state, err)
		}
	}
	raw, _ := json.Marshal(a)
	if strings.Contains(string(raw), "token") {
		t.Fatal("secret in evidence")
	}
	if err := s.store.discordRebootFinish(a, ""); err != nil {
		t.Fatal(err)
	}
	if err := s.store.discordRebootFinish(a, "accepted"); err == nil {
		t.Fatal("deleted evidence resurrected")
	}
	if a.valid(s.store.now().Add(-2 * time.Minute)) {
		t.Fatal("future timestamp accepted")
	}
}

func TestDiscordRebootReconciliationUsesAuthoritativeHealth(t *testing.T) {
	for _, outcome := range []string{"healthy", "wrong_digest", "health_wait", "health_expired", "status_expired"} {
		t.Run(outcome, func(t *testing.T) {
			s, rt, v, fake := advancedOSFixture(t)
			validationFixture(t, 0)
			oldBoot := readSystemUpdateBootID
			t.Cleanup(func() { readSystemUpdateBootID = oldBoot })
			readSystemUpdateBootID = func() (string, error) { return activationNewBoot, nil }
			healthCalls := 0
			runAdminValidationHelper = func(ctx context.Context) ([]byte, error) {
				healthCalls++
				actor, ok := ctx.Value(gatewayPrincipalKey{}).(session)
				if !ok || actor.Username != "discord:777" || actor.Role != roleAdministrator || actor.AuthSource != authSourceDiscord {
					t.Error("health not authenticated as Discord admin")
				}
				deadline, ok := ctx.Deadline()
				if !ok || time.Until(deadline) > 90*time.Second {
					t.Error("unbounded reconciliation")
				}
				if outcome == "health_wait" || outcome == "health_expired" {
					return []byte("PRIVATE health output"), validationTestExitError{code: 1}
				}
				return []byte("PRIVATE healthy output"), nil
			}
			a := rebootEvidence(v, s.store.now())
			a.Operation = "update_reboot"
			a.ExpectedDigest = "sha256:" + strings.Repeat("a", 64)
			if outcome == "wrong_digest" {
				a.ExpectedDigest = "sha256:" + strings.Repeat("b", 64)
			}
			if outcome == "health_expired" || outcome == "status_expired" {
				a.RequestedAt = a.RequestedAt.Add(-21 * time.Minute)
			}
			if outcome == "status_expired" {
				runBootcStatus = func(context.Context) ([]byte, error) { return nil, errors.New("private status") }
			}
			if err := s.store.discordRebootCreate(a); err != nil {
				t.Fatal(err)
			}
			s.discordMu.Lock()
			s.discordReconcileRebootLocked(rt, v)
			s.discordMu.Unlock()
			discordEventually(t, func() bool { s.discordMu.Lock(); defer s.discordMu.Unlock(); return !s.discordRebootReconciling })
			saved, err := s.store.discordRebootRead()
			if err != nil {
				t.Fatal(err)
			}
			if outcome == "health_wait" {
				if saved == nil {
					t.Fatal("unhealthy appliance confirmed")
				}
				return
			}
			if saved != nil {
				t.Fatal("terminal evidence retained")
			}
			select {
			case m := <-fake.messages:
				if strings.Contains(m.Content, "PRIVATE") || (strings.Contains(m.Content, "confirmed:") && outcome != "healthy") {
					t.Fatal("false confirmation or private output", m.Content)
				}
				if outcome == "healthy" && (!strings.Contains(m.Content, "OS update confirmed:") || healthCalls != 1) {
					t.Fatal("health success not verified", m.Content, healthCalls)
				}
			default:
				t.Fatal("no terminal result")
			}
		})
	}
}
func TestDiscordUpdateRebootSecondConfirmationKeepsReview(t *testing.T) {
	s, rt, v, _ := advancedOSFixture(t)
	oldBoot, oldHelper := readSystemUpdateBootID, runAdminSystemUpdateRebootHelper
	t.Cleanup(func() { readSystemUpdateBootID = oldBoot; runAdminSystemUpdateRebootHelper = oldHelper })
	readSystemUpdateBootID = func() (string, error) { return activationOldBoot, nil }
	calls := 0
	runAdminSystemUpdateRebootHelper = func(_ context.Context, args ...string) ([]byte, int, error) {
		calls++
		joined := strings.Join(args, " ")
		if !strings.Contains(joined, "--staged-digest=sha256:"+strings.Repeat("b", 64)) || !strings.Contains(joined, "--warning-seconds=60") || !strings.Contains(joined, "--backup-minecraft") {
			t.Error("reboot selection changed")
		}
		confirmed := strings.Contains(joined, "--confirm-players")
		if calls == 1 {
			if confirmed {
				t.Error("players already confirmed")
			}
			return []byte(`{"ok":false,"state":"confirmation_required","confirmation_required":true,"reason":"players_online"}`), 10, errors.New("players")
		}
		if !confirmed {
			t.Error("second player consent missing")
		}
		return []byte(`{"ok":true,"state":"accepted","accepted":true}`), 0, nil
	}
	a := discordPendingAction{Generation: v.Generation, GuildID: v.GuildID, UserID: "777", ChannelID: "999", Operation: "update_reboot", StagedDigest: "sha256:" + strings.Repeat("b", 64), WarningSeconds: 60, BackupMinecraft: true}
	s.discordMu.Lock()
	p := discordAdvancedConfirm(rt, a, discordAdvancedPreview(a))
	s.discordMu.Unlock()
	p = advancedHandle(t, s, rt, v, "777", "adv_confirm", advancedNonce(t, p, "adv_confirm"))
	nonce := advancedNonce(t, p, "adv_confirm")
	second := rt.actions[nonce]
	if second.StagedDigest != a.StagedDigest || second.WarningSeconds != a.WarningSeconds || second.BackupMinecraft != a.BackupMinecraft || second.UserID != a.UserID || second.Generation != a.Generation || second.GuildID != a.GuildID || second.Operation != a.Operation || !second.PlayersConfirmed {
		t.Fatal("second confirmation changed authority")
	}
	p = advancedHandle(t, s, rt, v, "777", "adv_confirm", nonce)
	saved, err := s.store.discordRebootRead()
	if err != nil || saved == nil || saved.State != "accepted" || calls != 2 {
		t.Fatal("second reboot attempt", p.text, err)
	}
}
func TestDiscordRebootGenerationUpdateFailureIsAtomic(t *testing.T) {
	s, _, v := discordBoundFixture(t)
	discordApproveFixture(t, s, &v, "777")
	if err := s.store.discordRebootCreate(rebootEvidence(v, s.store.now())); err != nil {
		t.Fatal(err)
	}
	before, err := s.store.discordRead()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.store.db.Exec(`CREATE TRIGGER reject_reboot_generation BEFORE UPDATE ON discord_reboot_state BEGIN SELECT RAISE(ABORT,'blocked'); END`); err != nil {
		t.Fatal(err)
	}
	changed := v
	changed.Enabled = false
	changed.Generation++
	if err := s.store.discordSave(changed, nil, "enabled"); err == nil {
		t.Fatal("disable crossed failed durable generation update")
	}
	current, err := s.store.discordRead()
	if err != nil || !reflect.DeepEqual(current, before) {
		t.Fatal("non-atomic generation change")
	}
}

func TestDiscordRebootMenuNonceCannotSkipConfirmation(t *testing.T) {
	s, rt, v, _ := advancedOSFixture(t)
	p := advancedHandle(t, s, rt, v, "777", "adv_reboots", "")
	nonce := advancedNonce(t, p, "adv_review")
	if !rt.actions[nonce].ReviewOnly {
		t.Fatal("reboot choice already executable")
	}
	if p = advancedHandle(t, s, rt, v, "777", "adv_confirm", nonce); p.text != discordFailure().text {
		t.Fatal("review nonce skipped confirmation", p.text)
	}
	saved, err := s.store.discordRebootRead()
	if err != nil || saved != nil {
		t.Fatal("reboot evidence saved before confirmation", err)
	}
	p = advancedHandle(t, s, rt, v, "777", "adv_reboots", "")
	nonce = advancedNonce(t, p, "adv_review")
	p = advancedHandle(t, s, rt, v, "777", "adv_review", nonce)
	nonce = advancedNonce(t, p, "adv_confirm")
	if rt.actions[nonce].ReviewOnly {
		t.Fatal("explicit confirmation missing execution authority")
	}
}
