package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/rest"
)

var discordJavaIdentity = regexp.MustCompile(`^[A-Za-z0-9_]{1,16}$`)
var discordAccessID = regexp.MustCompile(`^[a-f0-9]{48}$`)
var errDiscordAccessLimit = errors.New("pending request limit")

type discordAccessRequest struct {
	RequestID                                                                         string
	Generation                                                                        int64
	GuildID, UserID, UserName, ChannelID, InstanceID, Platform, PlayerIdentity, State string
	RequestedAt, DecidedAt, ExpiresAt, ResultExpiresAt                                int64
}

func discordAccessIdentity(platform, value string) (string, bool) {
	if platform == "java" {
		value = strings.TrimSpace(value)
		return value, discordJavaIdentity.MatchString(value)
	}
	// Bedrock controls are rejected before trimming so a newline cannot disappear.
	if !utf8.ValidString(value) || strings.ContainsFunc(value, unicode.IsControl) {
		return "", false
	}
	value = strings.TrimSpace(value)
	if platform != "bedrock" {
		return "", false
	}
	value = strings.TrimPrefix(value, ".")
	return value, strings.TrimSpace(value) != "" && utf8.RuneCountInString(value) <= 64
}
func (a discordAccessRequest) valid() bool {
	identity, ok := discordAccessIdentity(a.Platform, a.PlayerIdentity)
	// A second leading dot is legitimate submitted Bedrock data, not normalized twice.
	if a.Platform == "bedrock" {
		ok = utf8.ValidString(a.PlayerIdentity) && strings.TrimSpace(a.PlayerIdentity) != "" && utf8.RuneCountInString(a.PlayerIdentity) <= 64 && !strings.ContainsFunc(a.PlayerIdentity, unicode.IsControl)
		identity = a.PlayerIdentity
	}
	terminal := (a.State == "approved" || a.State == "rejected") && a.DecidedAt >= a.RequestedAt && a.ResultExpiresAt == a.DecidedAt+86400
	return discordAccessID.MatchString(a.RequestID) && a.Generation >= 0 &&
		validDiscordSnowflake(a.GuildID) && validDiscordSnowflake(a.UserID) &&
		validDiscordSnowflake(a.ChannelID) && instanceIDPattern.MatchString(a.InstanceID) &&
		a.UserName == discordText(a.UserName) && len(a.UserName) <= 400 &&
		ok && identity == a.PlayerIdentity && a.RequestedAt > 0 &&
		a.ExpiresAt == a.RequestedAt+7*86400 &&
		((a.State == "pending" && a.DecidedAt == 0 && a.ResultExpiresAt == 0) || terminal)
}
func discordAccessPrune(tx *sql.Tx, now int64) error {
	_, err := tx.Exec(`DELETE FROM discord_access_requests WHERE (state='pending' AND expires_at<=?) OR (state IN ('approved','rejected') AND result_expires_at<=?)`, now, now)
	return err
}

const discordAccessColumns = `request_id,discord_generation,guild_id,user_id,user_name,channel_id,instance_id,platform,player_identity,state,requested_at,decided_at,expires_at,result_expires_at`

func discordAccessRows(tx *sql.Tx) ([]discordAccessRequest, error) {
	rows, err := tx.Query(`SELECT ` + discordAccessColumns + ` FROM discord_access_requests`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []discordAccessRequest
	for rows.Next() {
		var a discordAccessRequest
		if err = rows.Scan(&a.RequestID, &a.Generation, &a.GuildID, &a.UserID, &a.UserName, &a.ChannelID, &a.InstanceID, &a.Platform, &a.PlayerIdentity, &a.State, &a.RequestedAt, &a.DecidedAt, &a.ExpiresAt, &a.ResultExpiresAt); err != nil {
			return nil, err
		}
		if !a.valid() {
			return nil, errDiscordState
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
func (s *webUIStore) discordAccessRead() ([]discordAccessRequest, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if err = discordAccessPrune(tx, s.now().Unix()); err != nil {
		return nil, err
	}
	out, err := discordAccessRows(tx)
	if err != nil {
		return nil, err
	}
	return out, tx.Commit()
}
func (s *webUIStore) discordAccessCreate(a discordAccessRequest, actor session) (discordAccessRequest, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return a, err
	}
	defer tx.Rollback()
	now := s.now().Unix()
	if err = discordAccessPrune(tx, now); err != nil {
		return a, err
	}
	all, err := discordAccessRows(tx)
	if err != nil {
		return a, err
	}
	total, own := 0, 0
	for _, p := range all {
		if p.State != "pending" {
			continue
		}
		total++
		if p.UserID == a.UserID {
			own++
		}
		if p.GuildID == a.GuildID && p.Generation == a.Generation && p.UserID == a.UserID && p.InstanceID == a.InstanceID && p.Platform == a.Platform {
			return p, tx.Commit()
		}
	}
	if total >= 64 || own >= 3 {
		return a, errDiscordAccessLimit
	}
	var random [24]byte
	if _, err = rand.Read(random[:]); err != nil {
		return a, err
	}
	a.RequestID = hex.EncodeToString(random[:])
	a.State = "pending"
	a.RequestedAt = now
	a.ExpiresAt = now + 7*86400
	if !a.valid() {
		return a, errDiscordState
	}
	_, err = tx.Exec(`INSERT INTO discord_access_requests (`+discordAccessColumns+`) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, a.RequestID, a.Generation, a.GuildID, a.UserID, a.UserName, a.ChannelID, a.InstanceID, a.Platform, a.PlayerIdentity, a.State, a.RequestedAt, 0, a.ExpiresAt, 0)
	if err != nil {
		return a, err
	}
	if err = insertAuditEventTx(tx, s.now(), actor, "discord_minecraft_access_request", a.InstanceID, true, ""); err != nil {
		return a, err
	}
	return a, tx.Commit()
}
func (s *webUIStore) discordAccessDecide(a discordAccessRequest, state string, actor session) error {
	if state != "approved" && state != "rejected" {
		return errDiscordState
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := s.now().Unix()
	if err = discordAccessPrune(tx, now); err != nil {
		return err
	}
	result, err := tx.Exec(`UPDATE discord_access_requests SET state=?,decided_at=?,result_expires_at=? WHERE request_id=? AND state='pending' AND discord_generation=? AND guild_id=? AND expires_at>?`, state, now, now+86400, a.RequestID, a.Generation, a.GuildID, now)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil || n != 1 {
		return errDiscordState
	}
	if state == "rejected" {
		if err = insertAuditEventTx(tx, s.now(), actor, "discord_minecraft_access_reject", a.InstanceID, true, ""); err != nil {
			return err
		}
	}
	return tx.Commit()
}
func discordAccessInstance(id string) (string, bool) {
	if minecraftInstances == nil || !instanceIDPattern.MatchString(id) {
		return "", false
	}
	instances, err := minecraftInstances.listInstances("")
	if err != nil {
		return "", false
	}
	for _, i := range instances {
		if i.ID == id {
			return discordText(i.DisplayName), true
		}
	}
	return "", false
}
func discordAccessEdition(platform string) string {
	if platform == "java" {
		return "Java Edition"
	}
	return "Bedrock / Xbox"
}
func (s *server) discordAccessRecent(v discordState, user string, p discordPanel) discordPanel {
	rows, err := s.store.discordAccessRead()
	if err != nil {
		p.text += "\nMinecraft access requests are unavailable."
		return p
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].RequestedAt == rows[j].RequestedAt {
			return rows[i].RequestID > rows[j].RequestID
		}
		return rows[i].RequestedAt > rows[j].RequestedAt
	})
	count := 0
	for _, a := range rows {
		if a.Generation != v.Generation || a.GuildID != v.GuildID || a.UserID != user {
			continue
		}
		if count == 0 {
			p.text += "\n\nMinecraft access"
		}
		if count == 5 {
			break
		}
		count++
		name, ok := discordAccessInstance(a.InstanceID)
		if !ok {
			name = "Unavailable server"
		}
		p.text += fmt.Sprintf("\n%s — %s — %s — %s", name, discordAccessEdition(a.Platform), discordText(a.PlayerIdentity), strings.ToUpper(a.State[:1])+a.State[1:])
	}
	return p
}

// Only edition buttons bypass defer. The opaque, short-lived action binds the
// initial modal response to the current user, guild, generation and instance.
func (s *server) discordAccessModalLocked(rt *discordRuntime, v discordState, i discord.Interaction, guild, user string) (discord.ModalCreate, bool) {
	c, ok := i.(discord.ComponentInteraction)
	if !ok || c.Data == nil || c.Data.Type() != discord.ComponentTypeButton {
		return discord.ModalCreate{}, false
	}
	if len(c.Data.CustomID()) > 100 {
		return discord.ModalCreate{}, false
	}
	parts := strings.Split(c.Data.CustomID(), ":")
	if len(parts) != 4 || parts[0] != "jv" || parts[1] != strconv.FormatInt(v.Generation, 10) || (parts[2] != "access_java" && parts[2] != "access_bedrock") {
		return discord.ModalCreate{}, false
	}
	a, ok := discordTakeAction(rt, parts[3], guild, user, time.Now())
	if !ok || a.Operation != "access_edition" {
		return discord.ModalCreate{}, false
	}
	if _, ok = discordAccessInstance(a.InstanceID); !ok {
		return discord.ModalCreate{}, false
	}
	platform := strings.TrimPrefix(parts[2], "access_")
	a.Operation = "access_submit_" + platform
	nonce, ok := discordNewAction(rt, a, time.Now())
	if !ok {
		return discord.ModalCreate{}, false
	}
	label := "Minecraft Java username"
	limit := 16
	if platform == "bedrock" {
		label = "Xbox gamertag or Floodgate UUID"
		limit = 65
	}
	modal := discord.NewModalCreate(discordComponentID(v, "access_submit", nonce), "Request Minecraft Access").AddLabel(label, discord.NewShortTextInput("identity").WithRequired(true).WithMaxLength(limit))
	return modal, true
}
func (s *server) discordAccessSubmitLocked(rt *discordRuntime, v discordState, i discord.ModalSubmitInteraction, guild, user string) discordPanel {
	if len(i.Data.CustomID) > 100 {
		return discordFailure()
	}
	parts := strings.Split(i.Data.CustomID, ":")
	if len(parts) != 4 || parts[0] != "jv" || parts[1] != strconv.FormatInt(v.Generation, 10) || parts[2] != "access_submit" {
		return discordFailure()
	}
	a, ok := discordTakeAction(rt, parts[3], guild, user, time.Now())
	if !ok || (a.Operation != "access_submit_java" && a.Operation != "access_submit_bedrock") {
		return discordFailure()
	}
	name, ok := discordAccessInstance(a.InstanceID)
	if !ok {
		return discordPanel{text: "This Minecraft server is no longer available."}
	}
	platform := strings.TrimPrefix(a.Operation, "access_submit_")
	identity, ok := discordAccessIdentity(platform, i.Data.Text("identity"))
	if !ok {
		return discordPanel{text: "Enter a valid Minecraft identity. Java usernames use 1–16 letters, numbers or underscores. Bedrock identities use up to 64 characters without control characters."}
	}
	if i.Channel().MessageChannel == nil {
		return discordFailure()
	}
	request, err := s.store.discordAccessCreate(discordAccessRequest{Generation: v.Generation, GuildID: guild, UserID: user, UserName: discordText(i.User().EffectiveName()), ChannelID: i.Channel().ID().String(), InstanceID: a.InstanceID, Platform: platform, PlayerIdentity: identity}, discordActor(v, user))
	if errors.Is(err, errDiscordAccessLimit) {
		return discordPanel{text: "The pending request limit has been reached (3 per user, 64 for JustVoxel). Wait for a decision or expiry."}
	}
	if err != nil {
		return discordFailure()
	}
	return discordPanel{text: fmt.Sprintf("Minecraft access request submitted for %q.\n%s — %s — Pending", name, discordAccessEdition(request.Platform), discordText(request.PlayerIdentity))}
}

func (s *server) discordAccessHandleLocked(ctx context.Context, rt *discordRuntime, v discordState, i discord.Interaction, guild, user, op, id string) discordPanel {
	if op == "access" {
		if _, ok := discordAccessInstance(id); !ok {
			return discordPanel{text: "This Minecraft server is no longer available."}
		}
		nonce, ok := discordNewAction(rt, discordPendingAction{Generation: v.Generation, GuildID: guild, UserID: user, InstanceID: id, Operation: "access_edition"}, time.Now())
		if !ok {
			return discordFailure()
		}
		return discordPanel{text: "Choose your Minecraft edition. This requests Minecraft access only; an administrator must approve it.", components: []discord.LayoutComponent{discord.NewActionRow(discordButton(v, "Java Edition", "access_java", nonce), discordButton(v, "Bedrock / Xbox", "access_bedrock", nonce), discordButton(v, "Cancel", "access_cancel", nonce))}}
	}
	if op == "access_cancel" {
		a, ok := discordTakeAction(rt, id, guild, user, time.Now())
		if !ok || a.Operation != "access_edition" {
			return discordFailure()
		}
		return discordPanel{text: "Cancelled. Open /jv to continue."}
	}
	if !v.administrator(user) {
		return discordFailure()
	}
	rows, err := s.store.discordAccessRead()
	if err != nil {
		return discordFailure()
	}
	pending := []discordAccessRequest{}
	for _, a := range rows {
		if a.Generation == v.Generation && a.GuildID == guild && a.State == "pending" {
			pending = append(pending, a)
		}
	}
	sort.Slice(pending, func(i, j int) bool {
		if pending[i].RequestedAt == pending[j].RequestedAt {
			return pending[i].RequestID < pending[j].RequestID
		}
		return pending[i].RequestedAt < pending[j].RequestedAt
	})
	if op == "access_review" {
		if id != "" {
			return discordFailure()
		}
		p := discordPanel{text: "Pending Minecraft access requests"}
		options := []discord.StringSelectMenuOption{}
		for n, a := range pending {
			if n == 25 {
				break
			}
			name, ok := discordAccessInstance(a.InstanceID)
			if !ok {
				name = "Unavailable server"
			}
			label := discordGuildName(a.UserName + " — " + name + " — " + discordAccessEdition(a.Platform))
			options = append(options, discord.NewStringSelectMenuOption(label, a.RequestID))
		}
		if len(options) == 0 {
			p.text = "No pending Minecraft access requests."
		} else {
			p.components = []discord.LayoutComponent{discord.NewActionRow(discord.NewStringSelectMenu(discordComponentID(v, "access_detail", ""), "Access request", options...))}
		}
		if len(pending) > 25 {
			p.text += "\nMore requests are pending. Review the oldest requests first."
		}
		return p
	}
	var a discordAccessRequest
	for _, candidate := range pending {
		if candidate.RequestID == id {
			a = candidate
			break
		}
	}
	if a.RequestID == "" {
		return discordFailure()
	}
	name, ok := discordAccessInstance(a.InstanceID)
	if !ok {
		return discordPanel{text: "This Minecraft server is no longer available."}
	}
	if op == "access_detail" {
		label := "Java username"
		if a.Platform == "bedrock" {
			label = "Bedrock / Xbox identity"
		}
		return discordPanel{text: fmt.Sprintf("%s wants to join %s.\n%s: %s", a.UserName, name, label, discordText(a.PlayerIdentity)), components: []discord.LayoutComponent{discord.NewActionRow(discordButton(v, "Approve", "access_approve", a.RequestID), discordButton(v, "Reject", "access_reject", a.RequestID), discordButton(v, "Back", "access_review", ""))}}
	}
	if op != "access_approve" && op != "access_reject" {
		return discordFailure()
	}
	if rt.accessApprovals[a.RequestID] {
		return discordPanel{text: "This request is being reviewed. Open Minecraft access requests again shortly."}
	}
	state := "rejected"
	if op == "access_approve" {
		code, raw := s.discordAccessWhitelistLocked(ctx, rt, v, guild, user, a)
		if code < 200 || code >= 300 {
			var failure struct {
				Error string `json:"error"`
			}
			_ = json.Unmarshal(raw, &failure)
			if a.Platform == "bedrock" && strings.HasPrefix(failure.Error, "Floodgate could not resolve") {
				return discordPanel{text: "Bedrock identity could not be resolved yet. The request remains pending. Ask the player to join through Geyser once or provide the Floodgate UUID, then try Approve again."}
			}
			return discordPanel{text: "Minecraft access could not be granted. The request remains pending. Check the server and player identity in JustVoxel, then try again."}
		}
		state = "approved"
	}
	latest, err := s.store.discordRead()
	if err != nil || latest.Generation != a.Generation || latest.GuildID != a.GuildID || !latest.administrator(user) || !s.discordUsableLocked(rt, latest, guild) {
		return discordFailure()
	}
	if err = s.store.discordAccessDecide(a, state, discordActor(v, user)); err != nil {
		return discordFailure()
	}
	// Hold discordMu through this bounded send: revocation cannot race notification.
	latest, err = s.store.discordRead()
	if err == nil && latest.Generation == a.Generation && latest.GuildID == a.GuildID && s.discordUsableLocked(rt, latest, guild) {
		result := "was approved"
		if state == "rejected" {
			result = "was not approved"
		}
		sendCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		_, _ = rt.api.CreateMessage(discordID(a.ChannelID), discord.MessageCreate{Content: fmt.Sprintf("%s — Minecraft access to %q %s.", a.UserName, name, result), AllowedMentions: &discord.AllowedMentions{}}, rest.WithCtx(sendCtx))
		cancel()
	}
	return discordPanel{text: "Minecraft access request " + state + ". The requester can see the result in /jv."}
}
func (s *server) discordAccessWhitelistLocked(ctx context.Context, rt *discordRuntime, v discordState, guild, user string, a discordAccessRequest) (int, []byte) {
	latest, err := s.store.discordRead()
	if err != nil || latest.Generation != v.Generation || !latest.administrator(user) || !s.discordUsableLocked(rt, latest, guild) || a.Generation != latest.Generation || a.GuildID != guild || a.State != "pending" || s.store.now().Unix() >= a.ExpiresAt || !a.valid() {
		return 403, nil
	}
	if _, ok := discordAccessInstance(a.InstanceID); !ok {
		return 403, nil
	}
	body, _ := json.Marshal(struct {
		Platform string `json:"platform"`
		Action   string `json:"action"`
		Name     string `json:"name"`
	}{a.Platform, "add", a.PlayerIdentity})
	r, _ := http.NewRequestWithContext(context.WithValue(ctx, gatewayPrincipalKey{}, discordActor(latest, user)), http.MethodPost, "http://discord/v1/minecraft/instances/"+a.InstanceID+"/whitelist", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.SetPathValue("id", a.InstanceID)
	capture := &gatewayResponse{header: make(http.Header), status: 200}
	// Fence concurrent decisions for this request while keeping normal Gateway
	// acknowledgements and trust revocation responsive during the handler call.
	if rt.accessApprovals == nil {
		rt.accessApprovals = map[string]bool{}
	}
	rt.accessApprovals[a.RequestID] = true
	s.discordMu.Unlock()
	s.whitelistChange(capture, r)
	s.discordMu.Lock()
	delete(rt.accessApprovals, a.RequestID)
	if capture.overflow || !json.Valid(capture.body.Bytes()) {
		return 502, nil
	}
	return capture.status, append([]byte(nil), capture.body.Bytes()...)
}
