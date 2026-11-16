package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
)

// The REST seam contains guild commands, interaction replies and fixed decision notifications.
// Interaction tokens remain private: immediate replies and bounded runtime-only admin response targets.
type discordREST interface {
	CreateMessage(snowflake.ID, discord.MessageCreate, ...rest.RequestOpt) (*discord.Message, error)
	SetGuildCommands(snowflake.ID, snowflake.ID, []discord.ApplicationCommandCreate, ...rest.RequestOpt) ([]discord.ApplicationCommand, error)
	CreateInteractionResponse(snowflake.ID, string, discord.InteractionResponse, ...rest.RequestOpt) error
	UpdateInteractionResponse(snowflake.ID, string, discord.MessageUpdate, ...rest.RequestOpt) (*discord.Message, error)
}

// DisGo rate limiters own a cleanup goroutine. Reuse one for this appliance
// rather than allocating a new limiter on each enable or credential generation.
var discordRESTLimiter = sync.OnceValue(func() rest.RateLimiter {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return rest.NewRateLimiter(rest.WithRateLimiterLogger(logger))
})

func (s *server) discordRESTClient(token string) discordREST {
	if s.discordNewREST != nil {
		return s.discordNewREST(token)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return rest.New(rest.NewClient(token, rest.WithLogger(logger), rest.WithRateLimiter(discordRESTLimiter())), rest.WithDefaultAllowedMentions(discord.AllowedMentions{}))
}
func discordCommands() []discord.ApplicationCommandCreate {
	return []discord.ApplicationCommandCreate{
		discord.SlashCommandCreate{Name: "jv", Description: "Open the JustVoxel control panel", Contexts: []discord.InteractionContextType{discord.InteractionContextTypeGuild}},
		discord.SlashCommandCreate{Name: "help", Description: "JustVoxel usage and access", Contexts: []discord.InteractionContextType{discord.InteractionContextTypeGuild}},
	}
}

// Called from the existing runtime loop, under discordMu. Retries are bounded in
// frequency and in time; neither callbacks nor the settings endpoint wait on REST.
func (s *server) discordSyncLocked(rt *discordRuntime, v discordState) {
	if rt.api == nil || rt.syncing || rt.syncedGuild == v.GuildID || time.Now().Before(rt.syncAfter) || !s.discordUsableLocked(rt, v, v.GuildID) {
		return
	}
	rt.syncing = true
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		_, err := rt.api.SetGuildCommands(discordID(v.ApplicationID), discordID(v.GuildID), discordCommands(), rest.WithCtx(ctx))
		s.discordMu.Lock()
		defer s.discordMu.Unlock()
		if s.discordRuntime != rt {
			return
		}
		rt.syncing = false
		if rt.generation != v.Generation {
			return
		}
		rt.syncAfter = time.Now().Add(time.Minute)
		if err == nil {
			rt.syncedGuild = v.GuildID
		}
	}()
}
func discordID(id string) snowflake.ID { n, _ := strconv.ParseUint(id, 10, 64); return snowflake.ID(n) }
func (s *server) discordUsableLocked(rt *discordRuntime, v discordState, guild string) bool {
	return rt != nil && s.discordRuntime == rt && v.Generation == rt.generation && v.Enabled && v.Verified && validDiscordSnowflake(guild) && guild == v.GuildID && rt.status == "connected" && rt.client != nil && rt.client.Status() == gateway.StatusReady && rt.guilds[guild].Available && s.discordResetAllowed()
}
func discordActor(v discordState, user string) session {
	role := roleViewer
	if v.administrator(user) {
		role = roleAdministrator
	}
	return session{Username: "discord:" + user, Role: role, AuthSource: authSourceDiscord}
}

type discordPendingAction struct {
	Generation                                  int64
	GuildID, UserID, InstanceID, Operation      string
	CreatedAt, ExpiresAt                        time.Time
	PlayersConfirmed                            bool
	ReviewOnly                                  bool
	DesiredEnabled, EnabledSet, PreviousEnabled bool
	Fingerprint, Policy, Version                string
	NetworkExpected                             string
	NetworkMembers                              []string
	WarningSeconds                              int
	BackupMinecraft                             bool
	StagedDigest, ChannelID                     string
}

func (s *server) discordInvalidateUserLocked(user string) {
	if s.discordRuntime != nil {
		for nonce, a := range s.discordRuntime.actions {
			if a.UserID == user {
				delete(s.discordRuntime.actions, nonce)
			}
		}
	}
}
func discordNewAction(rt *discordRuntime, a discordPendingAction, now time.Time) (string, bool) {
	if rt.actions == nil {
		rt.actions = map[string]discordPendingAction{}
	}
	for nonce, p := range rt.actions {
		if !now.Before(p.ExpiresAt) {
			delete(rt.actions, nonce)
		}
	}
	if len(rt.actions) >= 128 {
		return "", false
	}
	var random [24]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", false
	}
	nonce := hex.EncodeToString(random[:])
	a.CreatedAt = now
	a.ExpiresAt = now.Add(2 * time.Minute)
	rt.actions[nonce] = a
	return nonce, true
}
func discordTakeAction(rt *discordRuntime, nonce, guild, user string, now time.Time) (discordPendingAction, bool) {
	a, ok := rt.actions[nonce]
	if !ok || a.Generation != rt.generation || a.GuildID != guild || a.UserID != user || !now.Before(a.ExpiresAt) {
		return discordPendingAction{}, false
	}
	delete(rt.actions, nonce) // Consume before any authoritative operation, including cancel.
	return a, true
}

type discordPanel struct {
	text       string
	components []discord.LayoutComponent
}

func discordFailure() discordPanel {
	return discordPanel{text: "This control is unavailable or expired. Open /jv again in the bound Discord server."}
}
func discordComponentID(v discordState, op, id string) string {
	return "jv:" + strconv.FormatInt(v.Generation, 10) + ":" + op + ":" + id
}
func discordButton(v discordState, label, op, id string) discord.InteractiveComponent {
	return discord.NewSecondaryButton(label, discordComponentID(v, op, id))
}
func discordConfirmation(rt *discordRuntime, a discordPendingAction, name string) discordPanel {
	nonce, ok := discordNewAction(rt, a, time.Now())
	if !ok {
		return discordFailure()
	}
	text := fmt.Sprintf("%s %q?", strings.ToUpper(a.Operation[:1])+a.Operation[1:], discordText(name))
	if a.PlayersConfirmed {
		text = "Players are online. Confirm that you want to interrupt them.\n" + text
	}
	return discordPanel{text: text, components: []discord.LayoutComponent{discord.NewActionRow(discord.NewDangerButton("Confirm "+a.Operation, "jv:"+strconv.FormatInt(a.Generation, 10)+":confirm:"+nonce), discord.NewSecondaryButton("Cancel", "jv:"+strconv.FormatInt(a.Generation, 10)+":cancel:"+nonce))}}
}
func discordText(s string) string {
	s = discordGuildName(s)
	return strings.NewReplacer("@", "＠", "`", "ˋ", "*", "＊", "_", "＿", "~", "～", "|", "｜", ">", "›", "[", "［", "]", "］").Replace(s)
}
func discordRuntimeState(state string) string {
	switch state {
	case "running", "stopped", "configured", "failed", "starting", "stopping", "unconfigured", "offline":
		return state
	}
	return "unavailable"
}

// Gateway delivery must return promptly. A small concurrency limit bounds both
// temporary interaction tokens and operation work. No token reaches a logger.
func (s *server) discordInteraction(rt *discordRuntime, i discord.Interaction) {
	if i == nil || i.GuildID() == nil || i.Member() == nil {
		return
	}
	guild, user := i.GuildID().String(), i.User().ID.String()
	s.discordMu.Lock()
	v, err := s.store.discordRead()
	if err != nil || !validDiscordSnowflake(user) || i.ApplicationID().String() != v.ApplicationID || !s.discordUsableLocked(rt, v, guild) || rt.api == nil || rt.workers >= 16 {
		s.discordMu.Unlock()
		return
	}
	modal, hasModal := s.discordAccessModalLocked(rt, v, i, guild, user)
	acceptedGeneration := v.Generation
	rt.workers++
	s.discordMu.Unlock()
	go func() {
		defer func() { s.discordMu.Lock(); rt.workers--; s.discordMu.Unlock() }()
		ackCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		response := discord.InteractionResponse{Type: discord.InteractionResponseTypeDeferredCreateMessage, Data: discord.MessageCreate{Flags: discord.MessageFlagEphemeral}}
		if hasModal {
			response = discord.InteractionResponse{Type: discord.InteractionResponseTypeModal, Data: modal}
		}
		err := rt.api.CreateInteractionResponse(i.ID(), i.Token(), response, rest.WithCtx(ackCtx))
		cancel()
		if err != nil || hasModal {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 330*time.Second)
		defer cancel()
		s.discordMu.Lock()
		current, stateErr := s.store.discordRead()
		panel := discordFailure()
		if stateErr == nil && current.Generation == acceptedGeneration {
			panel = s.discordHandleLocked(ctx, rt, i, guild, user)
		}
		// Never send results from a revoked credential, binding or reset generation.
		latest, readErr := s.store.discordRead()
		usable := readErr == nil && latest.Generation == acceptedGeneration && s.discordUsableLocked(rt, latest, guild)
		target, adminResponse := rt.adminResponses[user]
		// Serialize the initial request result with its decision update.
		adminResponse = adminResponse && target.token == i.Token()
		if !adminResponse {
			s.discordMu.Unlock()
		} else {
			defer s.discordMu.Unlock()
		}
		if !usable {
			return
		}
		mentions := discord.AllowedMentions{}
		replyCtx, replyCancel := context.WithTimeout(ctx, 10*time.Second)
		defer replyCancel()
		_, _ = rt.api.UpdateInteractionResponse(i.ApplicationID(), i.Token(), discord.MessageUpdate{Content: &panel.text, Components: &panel.components, AllowedMentions: &mentions}, rest.WithCtx(replyCtx))
	}()
}

// Caller holds discordMu. Every invocation reloads state before accepting work.
// Only these fixed handlers can run; no Discord-supplied HTTP path is accepted.
func (s *server) discordInvokeLocked(ctx context.Context, rt *discordRuntime, guild, user, op, id string, players bool) (int, []byte) {
	v, err := s.store.discordRead()
	if err != nil || !s.discordUsableLocked(rt, v, guild) {
		return 403, nil
	}
	management := op == "start" || op == "stop" || op == "restart" || op == "backup"
	if management && !v.administrator(user) {
		return 403, nil
	}
	method, path := "GET", "/v1/minecraft/instances"
	var handler http.HandlerFunc
	switch op {
	case "list":
		handler = s.minecraftInstancesList
	case "status":
		handler = s.instanceRuntimeStatus
		path += "/" + id + "/status"
	case "players":
		handler = s.instanceMaintenanceRead
		path += "/" + id + "/players"
	case "start", "stop", "restart":
		method = "POST"
		path += "/" + id + "/" + op
		handler = func(w http.ResponseWriter, r *http.Request) { s.instanceRuntimeAction(w, r, op) }
	case "backup":
		method = "POST"
		path += "/" + id + "/backups"
		handler = s.instanceBackupNow
	default:
		return 403, nil
	}
	if op != "list" {
		if !instanceIDPattern.MatchString(id) {
			return 403, nil
		}
		identities, err := minecraftInstances.listInstances("")
		if err != nil {
			return 503, nil
		}
		found := false
		for _, instance := range identities {
			if instance.ID == id {
				found = true
				break
			}
		}
		if !found {
			return 403, nil
		}
	}
	actor := discordActor(v, user)
	if management && s.store.recordAuditEvent(actor, "discord_attempt_"+op, id, true, "") != nil {
		return 503, nil
	}
	body := []byte(`{}`)
	if op == "start" || op == "stop" || op == "restart" {
		body = []byte(fmt.Sprintf(`{"confirm_players":%t}`, players))
	}
	request, _ := http.NewRequestWithContext(context.WithValue(ctx, gatewayPrincipalKey{}, actor), method, "http://discord"+path, bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.SetPathValue("id", id)
	capture := &gatewayResponse{header: make(http.Header), status: 200}
	// This is the acceptance point, just as for Telegram: settings stay usable
	// during an accepted long operation. Subsequent interactions reauthorize.
	s.discordMu.Unlock()
	handler(capture, request)
	s.discordMu.Lock()
	if capture.overflow || !json.Valid(capture.body.Bytes()) {
		return 502, nil
	}
	return capture.status, append([]byte(nil), capture.body.Bytes()...)
}

type discordInstance struct{ ID, Name, State string }

func (s *server) discordInstancesLocked(ctx context.Context, rt *discordRuntime, guild, user string) ([]discordInstance, bool) {
	code, raw := s.discordInvokeLocked(ctx, rt, guild, user, "list", "", false)
	var data struct {
		Instances []minecraftRegisteredInstance `json:"instances"`
	}
	if code >= 400 || json.Unmarshal(raw, &data) != nil {
		return nil, false
	}
	out := []discordInstance{}
	for _, i := range data.Instances {
		if instanceIDPattern.MatchString(i.ID) {
			out = append(out, discordInstance{ID: i.ID, Name: i.DisplayName, State: "unavailable"})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := strings.ToLower(out[i].Name), strings.ToLower(out[j].Name)
		if a == b {
			return out[i].ID < out[j].ID
		}
		return a < b
	})
	return out, true
}
func discordReadPanel(v discordState, user string, instances []discordInstance, id string) discordPanel {
	panel := discordPanel{text: "JustVoxel — choose a Minecraft server."}
	options := []discord.StringSelectMenuOption{}
	for index, instance := range instances {
		if index >= 25 {
			break
		}
		label := discordGuildName(instance.Name)
		if label == "" {
			label = instance.ID
		}
		options = append(options, discord.NewStringSelectMenuOption(label, instance.ID))
	}
	if len(options) > 0 {
		panel.components = append(panel.components, discord.NewActionRow(discord.NewStringSelectMenu(discordComponentID(v, "select", ""), "Minecraft server", options...)))
	} else {
		panel.text = "No registered Minecraft servers are available."
	}
	if len(instances) > 25 {
		panel.text += "\nMore servers are available in JustVoxel WebUI."
	}
	if id != "" {
		found := false
		for _, instance := range instances {
			if instance.ID == id {
				panel.text = fmt.Sprintf("%s\nState: %s", discordText(instance.Name), instance.State)
				if len(instances) > 25 {
					panel.text += "\nMore servers are available in JustVoxel WebUI."
				}
				found = true
				break
			}
		}
		if !found {
			return discordFailure()
		}
		panel.components = append(panel.components, discord.NewActionRow(discordButton(v, "Status", "status", id), discordButton(v, "Players", "players", id), discordButton(v, "Refresh", "refresh", id), discordButton(v, "Request Minecraft Access", "access", id)))
		if v.administrator(user) {
			panel.components = append(panel.components, discord.NewActionRow(discordButton(v, "Start", "start", id), discordButton(v, "Stop", "stop", id), discordButton(v, "Restart", "restart", id), discordButton(v, "Backup", "backup", id), discordButton(v, "Advanced", "adv_menu", id)))
		}
	}
	if v.administrator(user) {
		panel.components = append(panel.components, discord.NewActionRow(discordButton(v, "Minecraft access requests", "access_review", ""), discordButton(v, "Minecraft Network", "adv_network", ""), discordButton(v, "System", "adv_system", "")))
	}
	if !v.administrator(user) {
		panel.components = append(panel.components, discord.NewActionRow(discordButton(v, "Request admin access", "request", "")))
	}
	return panel
}

func (s *server) discordHandleLocked(ctx context.Context, rt *discordRuntime, i discord.Interaction, guild, user string) discordPanel {
	v, err := s.store.discordRead()
	if err != nil || i == nil || i.GuildID() == nil || i.Member() == nil || i.GuildID().String() != guild || i.User().ID.String() != user || !validDiscordSnowflake(user) || i.ApplicationID().String() != v.ApplicationID || !s.discordUsableLocked(rt, v, guild) {
		return discordFailure()
	}
	acceptedGeneration := v.Generation
	op, id := "panel", ""
	switch interaction := i.(type) {
	case discord.ApplicationCommandInteraction:
		if interaction.Data == nil || interaction.Data.Type() != discord.ApplicationCommandTypeSlash {
			return discordFailure()
		}
		switch interaction.Data.CommandName() {
		case "jv":
		case "help":
			return discordPanel{text: "Use /jv to select a Minecraft server, view Status or Players, and Refresh. Members of the bound Discord server can request Minecraft access: choose Request Minecraft Access, select an edition, and enter a Minecraft identity. A JustVoxel Discord administrator reviews it through Minecraft access requests; it does not automatically whitelist you or grant administration. Request admin access, then ask the primary JustVoxel WebUI administrator to approve it. Approved Discord administrators can Start, Stop, Restart, or create a manual Backup, with confirmation."}
		default:
			return discordFailure()
		}
	case discord.ModalSubmitInteraction:
		return s.discordAccessSubmitLocked(rt, v, interaction, guild, user)
	case discord.ComponentInteraction:
		if interaction.Data == nil {
			return discordFailure()
		}
		custom := interaction.Data.CustomID()
		if len(custom) > 100 {
			return discordFailure()
		}
		parts := strings.Split(custom, ":")
		if len(parts) != 4 || parts[0] != "jv" || parts[1] != strconv.FormatInt(v.Generation, 10) {
			return discordFailure()
		}
		op, id = parts[2], parts[3]
		if op == "select" || op == "access_detail" || op == "adv_member" {
			data, ok := interaction.Data.(discord.StringSelectMenuInteractionData)
			if !ok || len(data.Values) != 1 || id != "" {
				return discordFailure()
			}
			id = data.Values[0]
		} else if interaction.Data.Type() != discord.ComponentTypeButton {
			return discordFailure()
		}
	default:
		return discordFailure()
	}
	if strings.HasPrefix(op, "adv_") {
		return s.discordAdvancedLocked(ctx, rt, v, i, guild, user, op, id)
	}
	if strings.HasPrefix(op, "access") {
		return s.discordAccessHandleLocked(ctx, rt, v, i, guild, user, op, id)
	}
	if op == "request" {
		if id != "" {
			return discordFailure()
		}
		if v.administrator(user) {
			return discordPanel{text: "You are already an approved Discord administrator."}
		}
		for _, u := range v.Pending {
			if u.UserID == user {
				discordRememberAdminResponse(rt, v, i, guild, user, time.Now())
				return discordPanel{text: "Your admin access request is pending in JustVoxel WebUI."}
			}
		}
		if len(v.Pending) >= 32 {
			return discordPanel{text: "The request list is full. Contact the primary JustVoxel administrator."}
		}
		identity := i.User()
		v.Pending = append(v.Pending, discordIdentity{GuildID: guild, UserID: user, Name: discordGuildName(identity.EffectiveName()), Username: discordGuildName(identity.Username), RequestedAt: time.Now().UTC()})
		actor := discordActor(v, user)
		if s.store.discordSave(v, &actor, "admin_request") != nil {
			return discordFailure()
		}
		discordRememberAdminResponse(rt, v, i, guild, user, time.Now())
		return discordPanel{text: "Admin access requested. The primary JustVoxel WebUI administrator must approve it."}
	}
	switch op {
	case "panel", "select", "refresh", "status", "players", "start", "stop", "restart", "backup", "confirm", "cancel":
	default:
		return discordFailure()
	}
	if (op == "start" || op == "stop" || op == "restart" || op == "backup") && !v.administrator(user) {
		return discordFailure()
	}
	var action discordPendingAction
	if op == "confirm" || op == "cancel" {
		if !v.administrator(user) {
			return discordFailure()
		}
		var ok bool
		action, ok = discordTakeAction(rt, id, guild, user, time.Now())
		if !ok {
			return discordFailure()
		}
		if op == "cancel" {
			return discordPanel{text: "Cancelled. Open /jv to continue."}
		}
		id = action.InstanceID
	}
	instances, ok := s.discordInstancesLocked(ctx, rt, guild, user)
	if !ok {
		return discordFailure()
	}
	// The list may have taken time. Reload approval and binding before any result
	// or confirmation is generated; invoke also checks them at execution.
	v, err = s.store.discordRead()
	if err != nil || v.Generation != acceptedGeneration || !s.discordUsableLocked(rt, v, guild) {
		return discordFailure()
	}
	var selected discordInstance
	if id != "" {
		for _, instance := range instances {
			if instance.ID == id {
				selected = instance
				break
			}
		}
		if selected.ID == "" {
			return discordFailure()
		}
	}
	switch op {
	case "panel":
		if id != "" {
			return discordFailure()
		}
		return s.discordAccessRecent(v, user, discordReadPanel(v, user, instances, ""))
	case "select", "refresh", "status", "players":
		if id == "" {
			return discordFailure()
		}
		code, raw := s.discordInvokeLocked(ctx, rt, guild, user, "status", id, false)
		var status instanceRuntimeStatus
		if code >= 400 || json.Unmarshal(raw, &status) != nil {
			return discordFailure()
		}
		selected.State = discordRuntimeState(status.State)
		for index := range instances {
			if instances[index].ID == id {
				instances[index].State = selected.State
			}
		}
		panel := s.discordAccessRecent(v, user, discordReadPanel(v, user, instances, id))
		if op == "players" {
			code, raw = s.discordInvokeLocked(ctx, rt, guild, user, "players", id, false)
			var p instancePlayers
			if code >= 400 || json.Unmarshal(raw, &p) != nil {
				panel.text += "\nPlayer status unavailable."
			} else if p.State == "online" {
				panel.text += fmt.Sprintf("\nPlayers: %d/%d", p.Online, p.Maximum)
			} else if p.State == "offline" {
				panel.text += "\nPlayers: offline (0 online)"
			} else {
				panel.text += "\nPlayer status unavailable."
			}
		}

		return panel
	case "start", "stop", "restart", "backup":
		if !v.administrator(user) || id == "" {
			return discordFailure()
		}
		return discordConfirmation(rt, discordPendingAction{Generation: v.Generation, GuildID: guild, UserID: user, InstanceID: id, Operation: op}, selected.Name)
	case "confirm":
		if !v.administrator(user) {
			return discordFailure()
		}
		code, raw := s.discordInvokeLocked(ctx, rt, guild, user, action.Operation, id, action.PlayersConfirmed)
		var result struct {
			ConfirmationRequired bool   `json:"confirmation_required"`
			Reason               string `json:"reason"`
		}
		_ = json.Unmarshal(raw, &result)
		if code < 400 && result.ConfirmationRequired && result.Reason == "players_online" && (action.Operation == "stop" || action.Operation == "restart") && !action.PlayersConfirmed {
			latest, e := s.store.discordRead()
			if e != nil || latest.Generation != action.Generation || !s.discordUsableLocked(rt, latest, guild) || !latest.administrator(user) {
				return discordFailure()
			}
			action.PlayersConfirmed = true
			return discordConfirmation(rt, action, selected.Name)
		}
		if code >= 400 || result.ConfirmationRequired {
			return discordPanel{text: "The operation could not complete. Review current status and recovery information in JustVoxel WebUI."}
		}
		return discordPanel{text: "Request accepted for " + discordText(selected.Name) + ". Open /jv to refresh status. Manual backup progress is available in JustVoxel WebUI."}
	default:
		return discordFailure()
	}
}

// This target never enters discordState, a journal, audit, or a WebUI payload.
// Ten minutes leaves margin within Discord's interaction response lifetime.
type discordAdminResponse struct {
	generation                            int64
	guildID, userID, applicationID, token string
	expiresAt                             time.Time
}

func discordPruneAdminResponses(rt *discordRuntime, now time.Time) {
	for user, target := range rt.adminResponses {
		if target.generation != rt.generation || !now.Before(target.expiresAt) {
			delete(rt.adminResponses, user)
		}
	}
}

func discordRememberAdminResponse(rt *discordRuntime, v discordState, i discord.Interaction, guild, user string, now time.Time) {
	discordPruneAdminResponses(rt, now)
	if rt.adminResponses == nil {
		rt.adminResponses = map[string]discordAdminResponse{}
	}
	if _, exists := rt.adminResponses[user]; !exists && len(rt.adminResponses) >= 32 {
		return
	}
	rt.adminResponses[user] = discordAdminResponse{generation: v.Generation, guildID: guild, userID: user, applicationID: i.ApplicationID().String(), token: i.Token(), expiresAt: now.Add(10 * time.Minute)}
}

// Caller holds discordMu, after the authoritative decision has been saved.
// Consume once, then make one bounded best-effort send without a retry queue.
func (s *server) discordAdminResponseLocked(v discordState, user string, approved bool) {
	rt := s.discordRuntime
	if rt == nil {
		return
	}
	discordPruneAdminResponses(rt, time.Now())
	target, ok := rt.adminResponses[user]
	delete(rt.adminResponses, user)
	if !ok || target.generation != v.Generation || target.guildID != v.GuildID || target.applicationID != v.ApplicationID || target.userID != user || !s.discordUsableLocked(rt, v, target.guildID) || rt.api == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		s.discordMu.Lock()
		defer s.discordMu.Unlock()
		latest, err := s.store.discordRead()
		if err != nil || latest.Generation != target.generation || latest.ApplicationID != target.applicationID || !time.Now().Before(target.expiresAt) || !s.discordUsableLocked(rt, latest, target.guildID) || latest.administrator(user) != approved {
			return
		}
		panel := discordPanel{text: "Administrator access was not approved. Contact the primary JustVoxel WebUI administrator."}
		if approved {
			instances, found := s.discordInstancesLocked(ctx, rt, target.guildID, user)
			if !found {
				return
			}
			latest, err = s.store.discordRead()
			if err != nil || latest.Generation != target.generation || !time.Now().Before(target.expiresAt) || !s.discordUsableLocked(rt, latest, target.guildID) || !latest.administrator(user) {
				return
			}
			panel = s.discordAccessRecent(latest, user, discordReadPanel(latest, user, instances, ""))
		}
		mentions := discord.AllowedMentions{}
		_, _ = rt.api.UpdateInteractionResponse(discordID(target.applicationID), target.token, discord.MessageUpdate{Content: &panel.text, Components: &panel.components, AllowedMentions: &mentions}, rest.WithCtx(ctx))
	}()
}
