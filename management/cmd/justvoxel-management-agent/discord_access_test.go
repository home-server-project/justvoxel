package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
)

func accessTestRequest(v discordState, user, instance, platform string) discordAccessRequest {
	return discordAccessRequest{Generation: v.Generation, GuildID: v.GuildID, UserID: user, UserName: "Friendly", ChannelID: "999", InstanceID: instance, Platform: platform, PlayerIdentity: "Alex123"}
}
func accessCreateTest(t *testing.T, s *server, v discordState, user, id, platform string) discordAccessRequest {
	t.Helper()
	a, err := s.store.discordAccessCreate(accessTestRequest(v, user, id, platform), discordActor(v, user))
	if err != nil {
		t.Fatal(err)
	}
	return a
}
func accessRowsTest(t *testing.T, s *server) []discordAccessRequest {
	t.Helper()
	rows, err := s.store.discordAccessRead()
	if err != nil {
		t.Fatal(err)
	}
	return rows
}
func accessComponent(t *testing.T, p discordPanel, label string) string {
	t.Helper()
	raw, _ := json.Marshal(p.components)
	var rows []struct {
		Components []struct {
			Label string `json:"label"`
			ID    string `json:"custom_id"`
		} `json:"components"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		for _, c := range row.Components {
			if c.Label == label {
				return c.ID
			}
		}
	}
	t.Fatalf("missing %s: %s", label, raw)
	return ""
}
func accessDetailTest(t *testing.T, s *server, rt *discordRuntime, v discordState, user, id string) discordPanel {
	t.Helper()
	raw := map[string]any{
		"id":             "888",
		"application_id": "456",
		"type":           3,
		"token":          "private-interaction-token",
		"guild_id":       v.GuildID,
		"member":         map[string]any{"user": map[string]any{"id": user, "username": "friendly", "global_name": "Friendly"}},
		"data":           map[string]any{"component_type": 3, "custom_id": discordComponentID(v, "access_detail", ""), "values": []string{id}},
	}
	encoded, _ := json.Marshal(raw)
	i, err := discord.UnmarshalInteraction(encoded)
	if err != nil {
		t.Fatal(err)
	}
	s.discordMu.Lock()
	defer s.discordMu.Unlock()
	return s.discordHandleLocked(context.Background(), rt, i, v.GuildID, user)
}
func accessModalInteraction(t *testing.T, v discordState, user, custom, value string) discord.Interaction {
	t.Helper()
	raw, _ := json.Marshal(map[string]any{"id": "888", "application_id": "456", "type": 5, "token": "private-interaction-token", "guild_id": v.GuildID, "channel": map[string]any{"id": "999", "type": 0, "guild_id": v.GuildID}, "member": map[string]any{"user": map[string]any{"id": user, "username": "authoritative", "global_name": "Actual @everyone"}}, "data": map[string]any{"custom_id": custom, "components": []any{map[string]any{"type": 1, "components": []any{map[string]any{"type": 4, "custom_id": "identity", "value": value}}}}}})
	i, err := discord.UnmarshalInteraction(raw)
	if err != nil {
		t.Fatal(err)
	}
	return i
}
func accessOpenModal(t *testing.T, s *server, rt *discordRuntime, v discordState, user, id, edition string) discord.ModalCreate {
	t.Helper()
	p := discordHandleTest(t, s, rt, v, user, "access", id)
	accessComponent(t, p, "Java Edition")
	accessComponent(t, p, "Bedrock / Xbox")
	accessComponent(t, p, "Cancel")
	custom := accessComponent(t, p, edition)
	s.discordInteraction(rt, discordTestInteraction(t, v.GuildID, user, "", custom))
	fake := rt.api.(*discordFakeREST)
	select {
	case response := <-fake.acknowledgements:
		if response.Type != discord.InteractionResponseTypeModal {
			t.Fatalf("edition was deferred: %v", response.Type)
		}
		modal, ok := response.Data.(discord.ModalCreate)
		if !ok {
			t.Fatal("not a modal")
		}
		return modal
	case <-time.After(3 * time.Second):
		t.Fatal("no initial modal response")
	}
	return discord.ModalCreate{}
}
func accessSubmitTest(t *testing.T, s *server, rt *discordRuntime, v discordState, user string, modal discord.ModalCreate, value string) discordPanel {
	t.Helper()
	i := accessModalInteraction(t, v, user, modal.CustomID, value)
	s.discordMu.Lock()
	defer s.discordMu.Unlock()
	return s.discordHandleLocked(context.Background(), rt, i, v.GuildID, user)
}

func TestDiscordAccessMigrationV9AndRestart(t *testing.T) {
	store, path := openTestWebUIStore(t)
	before := discordVerifiedState()
	if err := store.discordSave(before, nil, ""); err != nil {
		t.Fatal(err)
	}
	rawBefore := ""
	if err := store.db.QueryRow(`SELECT record_json FROM discord_state`).Scan(&rawBefore); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`DROP TABLE discord_access_requests`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`DELETE FROM schema_migrations WHERE version=9`); err != nil {
		t.Fatal(err)
	}
	if err := store.close(); err != nil {
		t.Fatal(err)
	}
	upgraded, err := openWebUIStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer upgraded.close()
	var name, rawAfter string
	if err = upgraded.db.QueryRow(`SELECT name FROM schema_migrations WHERE version=9`).Scan(&name); err != nil || name != "discord_access_requests" {
		t.Fatal("migration v9", err)
	}
	if err = upgraded.db.QueryRow(`SELECT record_json FROM discord_state`).Scan(&rawAfter); err != nil || rawBefore != rawAfter {
		t.Fatal("v8 state changed", err)
	}
	before.GuildID = "111"
	a, err := upgraded.discordAccessCreate(accessTestRequest(before, "777", "jv-a1b2c3", "java"), discordActor(before, "777"))
	if err != nil {
		t.Fatal(err)
	}
	if err = upgraded.close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := openWebUIStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.close()
	rows, err := reopened.discordAccessRead()
	if err != nil || len(rows) != 1 || !reflect.DeepEqual(rows[0], a) {
		t.Fatal("request did not survive restart", err)
	}
	// Check the private row and audit schema data; no interaction token is a field.
	rowJSON, _ := json.Marshal(rows)
	if strings.Contains(string(rowJSON), "private-interaction-token") || strings.Contains(string(rowJSON), discordTestToken) {
		t.Fatal("secret persisted")
	}
	if _, err = reopened.db.Exec(`UPDATE discord_access_requests SET player_identity='bad Java name'`); err != nil {
		t.Fatal(err)
	}
	if _, err = reopened.discordAccessRead(); err == nil {
		t.Fatal("malformed row accepted")
	}
}
func TestDiscordAccessIdentityValidation(t *testing.T) {
	for _, tc := range []struct {
		platform, input, want string
		valid                 bool
	}{
		{"java", " Alex_123 ", "Alex_123", true}, {"java", "a", "a", true}, {"java", "1234567890123456", "1234567890123456", true},
		{"java", "12345678901234567", "", false}, {"java", "a.b", "", false}, {"java", "A B", "", false}, {"java", "", "", false}, {"java", "é", "", false},
		{"bedrock", " .Example Player ", "Example Player", true}, {"bedrock", "00000000-0000-0000-0009-01f123456789", "00000000-0000-0000-0009-01f123456789", true},
		{"bedrock", strings.Repeat("a", 64), strings.Repeat("a", 64), true}, {"bedrock", strings.Repeat("a", 65), "", false}, {"bedrock", ".", "", false}, {"bedrock", " . ", "", false}, {"bedrock", "\nAlex", "", false}, {"bedrock", "Alex\x00", "", false}, {"bedrock", "Alex\r\nName", "", false}, {"bedrock", "\t", "", false}, {"other", "Alex", "", false},
	} {
		t.Run(tc.platform+tc.input, func(t *testing.T) {
			value, ok := discordAccessIdentity(tc.platform, tc.input)
			if ok != tc.valid || (ok && value != tc.want) {
				t.Fatalf("got %q %v", value, ok)
			}
		})
	}
}
func TestDiscordAccessPendingDedupeLimitsAndRetention(t *testing.T) {
	s, _, v := discordBoundFixture(t)
	now := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	s.store.now = func() time.Time { return now }
	first := accessCreateTest(t, s, v, "777", "jv-a1b2c3", "java")
	duplicate := accessTestRequest(v, "777", "jv-a1b2c3", "java")
	duplicate.PlayerIdentity = "Other"
	got, err := s.store.discordAccessCreate(duplicate, discordActor(v, "777"))
	if err != nil || got.RequestID != first.RequestID || got.PlayerIdentity != "Alex123" {
		t.Fatal("dedupe", err)
	}
	accessCreateTest(t, s, v, "777", "jv-a1b2c3", "bedrock")
	accessCreateTest(t, s, v, "777", "jv-a1b2c4", "java")
	if _, err = s.store.discordAccessCreate(accessTestRequest(v, "777", "jv-a1b2c5", "java"), discordActor(v, "777")); !errors.Is(err, errDiscordAccessLimit) {
		t.Fatal("fourth request accepted")
	}
	for n := 0; n < 61; n++ {
		accessCreateTest(t, s, v, fmt.Sprint(1000+n), "jv-a1b2c3", "java")
	}
	if len(accessRowsTest(t, s)) != 64 {
		t.Fatal("64 pending not accepted")
	}
	if _, err = s.store.discordAccessCreate(accessTestRequest(v, "9999", "jv-a1b2c3", "java"), discordActor(v, "9999")); !errors.Is(err, errDiscordAccessLimit) {
		t.Fatal("65th accepted")
	}
	if err = s.store.discordAccessDecide(first, "rejected", discordActor(v, "888")); err != nil {
		t.Fatal(err)
	}
	accessCreateTest(t, s, v, "777", "jv-a1b2c5", "java") // terminal excluded from both caps
	if len(accessRowsTest(t, s)) != 65 {
		t.Fatal("terminal record not retained")
	}
	now = now.Add(24*time.Hour - time.Second)
	if len(accessRowsTest(t, s)) != 65 {
		t.Fatal("terminal expired early")
	}
	now = now.Add(time.Second)
	if len(accessRowsTest(t, s)) != 64 {
		t.Fatal("terminal not pruned")
	}
	now = now.Add(6 * 24 * time.Hour)
	if len(accessRowsTest(t, s)) != 0 {
		t.Fatal("pending not pruned at seven days")
	}
	if err = s.store.discordAccessDecide(first, "approved", discordActor(v, "888")); err == nil {
		t.Fatal("expired request decided")
	}
}
func TestDiscordAccessPanelAndModalSubmission(t *testing.T) {
	s, rt, v, f := discordMinecraftFixture(t)
	id := f.configs[0].ID
	old := runWhitelistHelper
	defer func() { runWhitelistHelper = old }()
	runWhitelistHelper = func(context.Context, ...string) ([]byte, error) {
		t.Fatal("submission mutated whitelist")
		return nil, nil
	}
	for _, user := range []string{"777", "888"} {
		if user == "888" {
			discordApproveFixture(t, s, &v, user)
		}
		p := discordHandleTest(t, s, rt, v, user, "refresh", id)
		accessComponent(t, p, "Request Minecraft Access")
		if user == "777" {
			accessComponent(t, p, "Request admin access")
		} else {
			accessComponent(t, p, "Access requests")
		}
	}
	for _, tc := range []struct{ user, edition, label, value, platform, identity string }{
		{"777", "Java Edition", "Minecraft Java username", " Alex123 ", "java", "Alex123"},
		{"888", "Bedrock / Xbox", "Xbox gamertag or Floodgate UUID", " .Example Player ", "bedrock", "Example Player"},
	} {
		modal := accessOpenModal(t, s, rt, v, tc.user, id, tc.edition)
		encoded, _ := json.Marshal(modal)
		if !strings.Contains(string(encoded), tc.label) {
			t.Fatal("incorrect modal label", string(encoded))
		}
		p := accessSubmitTest(t, s, rt, v, tc.user, modal, tc.value)
		if !strings.Contains(p.text, "Pending") {
			t.Fatal(p.text)
		}
		rows := accessRowsTest(t, s)
		found := false
		for _, a := range rows {
			if a.UserID == tc.user {
				found = true
				if a.UserName != "Actual ＠everyone" || a.ChannelID != "999" || a.Platform != tc.platform || a.PlayerIdentity != tc.identity {
					t.Fatal("identity not authoritative", a)
				}
			}
		}
		if !found {
			t.Fatal("missing request")
		}
	}
	current, _ := s.store.discordRead()
	if current.administrator("777") || len(current.Pending) != 0 {
		t.Fatal("community request granted admin approval")
	}
	// Normal Stage 2 responses retain their deferred ephemeral acknowledgement.
	fake := rt.api.(*discordFakeREST)
	s.discordInteraction(rt, discordTestInteraction(t, v.GuildID, "777", "help", ""))
	select {
	case ack := <-fake.acknowledgements:
		if ack.Type != discord.InteractionResponseTypeDeferredCreateMessage || ack.Data.(discord.MessageCreate).Flags != discord.MessageFlagEphemeral {
			t.Fatal("normal flow changed")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no defer")
	}
	select {
	case <-fake.replies:
	case <-time.After(3 * time.Second):
		t.Fatal("no normal reply")
	}
	modal := accessOpenModal(t, s, rt, v, "777", id, "Java Edition")
	s.discordInteraction(rt, accessModalInteraction(t, v, "777", modal.CustomID, "invalid name"))
	select {
	case ack := <-fake.acknowledgements:
		if ack.Type != discord.InteractionResponseTypeDeferredCreateMessage {
			t.Fatal("modal submit not deferred")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no submit defer")
	}
	select {
	case p := <-fake.replies:
		if p.Content == nil || !strings.Contains(*p.Content, "valid Minecraft identity") {
			t.Fatal("invalid submission accepted")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no submit reply")
	}
}

type accessNotificationREST struct {
	*discordFakeREST
	messages []discord.MessageCreate
	channels []snowflake.ID
	fail     bool
}

func (f *accessNotificationREST) CreateMessage(id snowflake.ID, m discord.MessageCreate, _ ...rest.RequestOpt) (*discord.Message, error) {
	f.channels = append(f.channels, id)
	f.messages = append(f.messages, m)
	if f.fail {
		return nil, errors.New("private notification error")
	}
	return nil, nil
}
func TestDiscordAccessReviewDecisionsAuthoritativeWhitelist(t *testing.T) {
	for _, mode := range []string{"approve", "already", "failure", "bedrock", "reject", "notify_failure"} {
		t.Run(mode, func(t *testing.T) {
			f := whitelistSurfaceFixture(t)
			_ = f
			s, rt, v := discordBoundFixture(t)
			discordRegistryFixture(t, f.backend.identities())
			discordApproveFixture(t, s, &v, "888")
			platform := "java"
			if mode == "bedrock" {
				platform = "bedrock"
			}
			a := accessCreateTest(t, s, v, "777", "jv-a1b2c3", platform)
			fake := &accessNotificationREST{discordFakeREST: newDiscordFakeREST(), fail: mode == "notify_failure"}
			rt.api = fake
			calls := [][]string{}
			old := runWhitelistHelper
			defer func() { runWhitelistHelper = old }()
			runWhitelistHelper = func(_ context.Context, args ...string) ([]byte, error) {
				calls = append(calls, append([]string(nil), args...))
				args = selectedWhitelistTestArgs(t, args)
				if args[0] == "list" {
					if mode == "already" {
						return []byte("Whitelisted: Alex123"), nil
					}
					return []byte("Whitelisted: nobody"), nil
				}
				if !reflect.DeepEqual(args, []string{"add-" + platform, "Alex123"}) {
					t.Fatal("wrong durable body", args)
				}
				if mode == "bedrock" {
					return []byte("floodgate could not resolve"), errors.New("private error")
				}
				if mode == "failure" {
					return []byte("private backend output"), errors.New("private error")
				}
				return []byte("private success output"), nil
			}
			p := accessDetailTest(t, s, rt, v, "888", a.RequestID)
			if !strings.Contains(p.text, "Friendly wants to join") || !strings.Contains(p.text, "Alex123") {
				t.Fatal(p.text)
			}
			op := "access_approve"
			if mode == "reject" {
				op = "access_reject"
			}
			p = discordHandleTest(t, s, rt, v, "888", op, a.RequestID)
			rows := accessRowsTest(t, s)
			if len(rows) != 1 {
				t.Fatal("lost request")
			}
			expected := "approved"
			if mode == "reject" {
				expected = "rejected"
			}
			if mode == "failure" || mode == "bedrock" {
				expected = "pending"
			}
			if rows[0].State != expected {
				t.Fatal(mode, p.text, rows[0].State)
			}
			if mode == "reject" && len(calls) != 0 {
				t.Fatal("reject called whitelist")
			}
			if mode == "already" && len(calls) != 1 {
				t.Fatal("idempotency not authoritative")
			}
			if mode == "approve" && len(calls) != 2 {
				t.Fatal("authoritative handler not invoked")
			}
			if expected == "pending" {
				if len(fake.messages) != 0 || !strings.Contains(p.text, "remains pending") {
					t.Fatal("failure terminal or notified")
				}
			} else {
				if len(fake.messages) != 1 || fake.channels[0] != 999 {
					t.Fatal("notification missing")
				}
				m := fake.messages[0]
				if m.AllowedMentions == nil || !reflect.DeepEqual(*m.AllowedMentions, discord.AllowedMentions{}) {
					t.Fatal("unrestricted mentions")
				}
				if rows[0].ResultExpiresAt-rows[0].DecidedAt != 86400 {
					t.Fatal("incorrect result retention")
				}
			}
			text := p.text
			for _, m := range fake.messages {
				text += m.Content
			}
			for _, secret := range []string{"private", "@everyone", "@here", a.RequestID} {
				if strings.Contains(text, secret) {
					t.Fatal("unsafe result", text)
				}
			}
			if mode == "bedrock" && !strings.Contains(p.text, "join through Geyser once") {
				t.Fatal("missing safe Floodgate guidance")
			}
		})
	}
}
func TestDiscordAccessAuthorityAndStaleControls(t *testing.T) {
	for _, mode := range []string{"member", "removed_admin", "wrong_guild", "wrong_generation", "disabled", "unverified", "runtime", "factory", "missing_instance", "forged"} {
		t.Run(mode, func(t *testing.T) {
			s, rt, v, f := discordMinecraftFixture(t)
			discordApproveFixture(t, s, &v, "888")
			a := accessCreateTest(t, s, v, "777", f.configs[0].ID, "java")
			user := "888"
			interactionState := v
			id := a.RequestID
			switch mode {
			case "member":
				user = "777"
			case "removed_admin":
				v.Administrators = nil
				discordSaveTestState(t, s, v)
			case "wrong_guild":
				interactionState.GuildID = "222"
			case "wrong_generation":
				interactionState.Generation--
			case "disabled":
				v.Enabled = false
				discordSaveTestState(t, s, v)
			case "unverified":
				v.Verified = false
				v.Enabled = false
				discordSaveTestState(t, s, v)
			case "runtime":
				rt.status = "connection_retry"
			case "factory":
				operations := openTestOperationStore(t)
				attachTestOperationStore(t, s, operations)
				if _, _, err := operations.beginFactoryReset(testFactoryResetFingerprint); err != nil {
					t.Fatal(err)
				}
			case "missing_instance":
				discordRegistryFixture(t, nil)
			case "forged":
				id = strings.Repeat("a", 48)
			}
			old := runWhitelistHelper
			defer func() { runWhitelistHelper = old }()
			runWhitelistHelper = func(context.Context, ...string) ([]byte, error) {
				t.Fatal("denied control invoked whitelist")
				return nil, nil
			}
			fake := &accessNotificationREST{discordFakeREST: newDiscordFakeREST()}
			rt.api = fake
			for _, op := range []string{"access_approve", "access_reject"} {
				p := discordHandleTest(t, s, rt, interactionState, user, op, id)
				if p.text != discordFailure().text && p.text != "This Minecraft server is no longer available." {
					t.Fatal(mode, op, p.text)
				}
			}
			if len(fake.messages) != 0 || accessRowsTest(t, s)[0].State != "pending" {
				t.Fatal("denied control decided or notified")
			}
		})
	}
	s, rt, v := discordBoundFixture(t)
	if p := discordHandleTest(t, s, rt, v, "777", "access_review", ""); p.text != discordFailure().text {
		t.Fatal("member opened review")
	}
}
func TestDiscordAccessReviewOrderingOverflowAndOwnVisibility(t *testing.T) {
	s, rt, v, f := discordMinecraftFixture(t)
	discordApproveFixture(t, s, &v, "888")
	id := f.configs[0].ID
	now := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	s.store.now = func() time.Time { return now }
	requests := []discordAccessRequest{}
	for n := 0; n < 26; n++ {
		requests = append(requests, accessCreateTest(t, s, v, fmt.Sprint(1000+n), id, "java"))
	}
	// Same timestamp ties must be ordered by opaque request ID.
	p := discordHandleTest(t, s, rt, v, "888", "access_review", "")
	raw, _ := json.Marshal(p.components)
	var rows []struct {
		Components []struct {
			Options []struct {
				Value string `json:"value"`
			} `json:"options"`
		} `json:"components"`
	}
	if json.Unmarshal(raw, &rows) != nil || len(rows) != 1 || len(rows[0].Components[0].Options) != 25 {
		t.Fatal("review not bounded", string(raw))
	}
	previous := ""
	for _, option := range rows[0].Components[0].Options {
		if option.Value <= previous {
			t.Fatal("review order not deterministic")
		}
		previous = option.Value
	}
	if !strings.Contains(p.text, "More requests are pending. Review the oldest requests first.") {
		t.Fatal("overflow missing")
	}
	for n := 0; n < 7; n++ {
		now = now.Add(time.Second)
		a := accessCreateTest(t, s, v, "777", id, "java")
		if err := s.store.discordAccessDecide(a, "approved", discordActor(v, "888")); err != nil {
			t.Fatal(err)
		}
	}
	own := s.discordAccessRecent(v, "777", discordPanel{})
	if strings.Count(own.text, "Approved") != 5 {
		t.Fatal("recent not bounded", own.text)
	}
	other := s.discordAccessRecent(v, "9999", discordPanel{})
	if other.text != "" {
		t.Fatal("other user saw records")
	}
	for _, hidden := range []string{"999", requests[0].RequestID, "discord_generation", "channel_id"} {
		if strings.Contains(own.text, hidden) {
			t.Fatal("internal data visible", own.text)
		}
	}
	// Use distinct approved identities to make newest-first ordering observable.
	if _, err := s.store.db.Exec(`UPDATE discord_access_requests SET player_identity='Newest' WHERE user_id='777' AND requested_at=(SELECT MAX(requested_at) FROM discord_access_requests WHERE user_id='777')`); err != nil {
		t.Fatal(err)
	}
	own = s.discordAccessRecent(v, "777", discordPanel{})
	if strings.Index(own.text, "Newest") > strings.Index(own.text, "Alex123") {
		t.Fatal("newest not first")
	}
	stale := v
	stale.Generation++
	if p := s.discordAccessRecent(stale, "777", discordPanel{}); p.text != "" {
		t.Fatal("stale generation shown")
	}
	stale = v
	stale.GuildID = "222"
	if p := s.discordAccessRecent(stale, "777", discordPanel{}); p.text != "" {
		t.Fatal("other guild shown")
	}
}
func TestDiscordAccessModalFencingAndMissingInstance(t *testing.T) {
	for _, mode := range []string{"user", "guild", "generation", "application", "disabled", "factory", "missing"} {
		t.Run(mode, func(t *testing.T) {
			s, rt, v, f := discordMinecraftFixture(t)
			modal := accessOpenModal(t, s, rt, v, "777", f.configs[0].ID, "Java Edition")
			user := "777"
			interactionState := v
			switch mode {
			case "user":
				user = "999"
			case "guild":
				interactionState.GuildID = "222"
			case "generation":
				v.Generation++
				discordSaveTestState(t, s, v)
			case "application":
				v.ApplicationID = "999"
				discordSaveTestState(t, s, v)
			case "disabled":
				v.Enabled = false
				discordSaveTestState(t, s, v)
			case "factory":
				operations := openTestOperationStore(t)
				attachTestOperationStore(t, s, operations)
				if _, _, err := operations.beginFactoryReset(testFactoryResetFingerprint); err != nil {
					t.Fatal(err)
				}
			case "missing":
				discordRegistryFixture(t, nil)
			}
			p := accessSubmitTest(t, s, rt, interactionState, user, modal, "Alex123")
			if p.text != discordFailure().text && p.text != "This Minecraft server is no longer available." {
				t.Fatal("stale modal accepted", p.text)
			}
			if len(accessRowsTest(t, s)) != 0 {
				t.Fatal("stale submission persisted")
			}
		})
	}
}
func TestDiscordAccessTrustChangesAtomicAndDurable(t *testing.T) {
	for _, action := range []string{"disable_enable", "same_guild", "different_guild", "credential", "disconnect", "factory"} {
		t.Run(action, func(t *testing.T) {
			s, rt, v := discordBoundFixture(t)
			accessCreateTest(t, s, v, "777", "jv-a1b2c3", "java")
			retain := action == "disable_enable" || action == "same_guild"
			switch action {
			case "disable_enable":
				for _, enabled := range []bool{false, true} {
					response := discordAdminRequest(s, administratorSession(time.Now()), map[string]any{"action": "enabled", "enabled": enabled})
					if response.Code != 200 {
						t.Fatal(response.Code, response.Body.String())
					}
				}
			case "same_guild", "different_guild":
				guild := "111"
				if action == "different_guild" {
					guild = "222"
					rt.guilds[guild] = discordGuild{ID: guild, Name: "Other", Available: true}
				}
				response := discordAdminRequest(s, administratorSession(time.Now()), map[string]any{"action": "guild", "guild_id": guild})
				if response.Code != 200 {
					t.Fatal(response.Code, response.Body.String())
				}
			case "credential", "disconnect":
				input := map[string]any{"action": action}
				if action == "credential" {
					input["token"] = discordTestToken
				}
				response := discordAdminRequest(s, administratorSession(time.Now()), input)
				if response.Code != 200 {
					t.Fatal(response.Code, response.Body.String())
				}
			case "factory":
				if err := s.discordFactoryRevoke(); err != nil {
					t.Fatal(err)
				}
			}
			rows := accessRowsTest(t, s)
			if retain {
				current, _ := s.store.discordRead()
				if len(rows) != 1 || rows[0].Generation != current.Generation {
					t.Fatal("request not retained across generation fence")
				}
			} else if len(rows) != 0 {
				t.Fatal("trust change retained requests")
			}
		})
	}
	s, _, v := discordBoundFixture(t)
	accessCreateTest(t, s, v, "777", "jv-a1b2c3", "java")
	if _, err := s.store.db.Exec(`CREATE TRIGGER fail_access_clear BEFORE DELETE ON discord_access_requests BEGIN SELECT RAISE(ABORT,'private failure'); END`); err != nil {
		t.Fatal(err)
	}
	response := discordAdminRequest(s, administratorSession(time.Now()), map[string]any{"action": "disconnect"})
	if response.Code != 503 {
		t.Fatal("cleanup failure reported success")
	}
	current, err := s.store.discordRead()
	if err != nil || !reflect.DeepEqual(current, v) {
		t.Fatal("failed clear changed trust", err)
	}
}
func TestDiscordAccessSourceSecurityBoundary(t *testing.T) {
	source, err := os.ReadFile("discord_access.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"runWhitelistHelper(", "exec.Command", "podman", "systemctl", "time.NewTicker", "time.NewTimer", "http.Get(", "interaction.Token()"} {
		if strings.Contains(string(source), forbidden) {
			t.Fatal("forbidden secondary backend, worker or token storage", forbidden)
		}
	}
	runtime, err := os.ReadFile("discord_runtime.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(runtime), "const discordGatewayIntents = gateway.IntentGuilds") {
		t.Fatal("intent changed")
	}
	for _, intent := range []string{"IntentMessageContent", "IntentGuildMembers", "IntentGuildPresences", "IntentGuildMessages", "IntentDirectMessages"} {
		if strings.Contains(string(runtime), intent) {
			t.Fatal("forbidden intent", intent)
		}
	}
	if len(discordCommands()) != 2 {
		t.Fatal("extra application command")
	}
}

func TestDiscordAccessInFlightDecisionAndRevokedResult(t *testing.T) {
	for _, mode := range []string{"in_flight", "generation", "guild", "reset"} {
		t.Run(mode, func(t *testing.T) {
			f := whitelistSurfaceFixture(t)
			s, rt, v := discordBoundFixture(t)
			discordRegistryFixture(t, f.backend.identities())
			discordApproveFixture(t, s, &v, "888")
			a := accessCreateTest(t, s, v, "777", "jv-a1b2c3", "java")
			fake := &accessNotificationREST{discordFakeREST: newDiscordFakeREST()}
			rt.api = fake
			old := runWhitelistHelper
			defer func() { runWhitelistHelper = old }()
			runWhitelistHelper = func(_ context.Context, args ...string) ([]byte, error) {
				args = selectedWhitelistTestArgs(t, args)
				if args[0] == "list" {
					return []byte("Whitelisted: nobody"), nil
				}
				switch mode {
				case "in_flight":
					// Authoritative handler runs with discordMu released; another reviewer
					// must not reject this request while its whitelist operation is running.
					p := discordHandleTest(t, s, rt, v, "888", "access_reject", a.RequestID)
					if !strings.Contains(p.text, "being reviewed") {
						t.Fatal("concurrent decision accepted", p.text)
					}
				case "generation":
					v.Generation++
					discordSaveTestState(t, s, v)
				case "guild":
					v.GuildID = "222"
					v.Administrators = nil
					discordSaveTestState(t, s, v)
				case "reset":
					operations := openTestOperationStore(t)
					attachTestOperationStore(t, s, operations)
					if _, _, err := operations.beginFactoryReset(testFactoryResetFingerprint); err != nil {
						t.Fatal(err)
					}
				}
				return []byte("Added player"), nil
			}
			p := discordHandleTest(t, s, rt, v, "888", "access_approve", a.RequestID)
			if mode == "in_flight" {
				if accessRowsTest(t, s)[0].State != "approved" {
					t.Fatal("approved result lost", p.text)
				}
			} else if len(fake.messages) != 0 {
				t.Fatal("revoked context sent result notification")
			}
		})
	}
}
func TestDiscordAccessPendingRecordsFencedAndResetClears(t *testing.T) {
	s, rt, v, f := discordMinecraftFixture(t)
	discordApproveFixture(t, s, &v, "888")
	a := accessCreateTest(t, s, v, "777", f.configs[0].ID, "java")
	if _, err := s.store.db.Exec(`UPDATE discord_access_requests SET discord_generation=discord_generation-1`); err != nil {
		t.Fatal(err)
	}
	p := discordHandleTest(t, s, rt, v, "888", "access_review", "")
	if !strings.Contains(p.text, "No pending") {
		t.Fatal("stale generation listed")
	}
	if p = discordHandleTest(t, s, rt, v, "888", "access_approve", a.RequestID); p.text != discordFailure().text {
		t.Fatal("stale generation approved")
	}
	if _, err := s.store.db.Exec(`UPDATE discord_access_requests SET discord_generation=?,guild_id='222'`, v.Generation); err != nil {
		t.Fatal(err)
	}
	p = discordHandleTest(t, s, rt, v, "888", "access_review", "")
	if !strings.Contains(p.text, "No pending") {
		t.Fatal("other guild listed")
	}
	if p = discordHandleTest(t, s, rt, v, "888", "access_reject", a.RequestID); p.text != discordFailure().text {
		t.Fatal("other guild rejected")
	}
	if err := s.store.resetFactoryState(); err != nil {
		t.Fatal(err)
	}
	if len(accessRowsTest(t, s)) != 0 {
		t.Fatal("Factory Reset retained community rows")
	}
}
