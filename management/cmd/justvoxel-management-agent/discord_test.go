package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

const discordTestToken = "opaque-discord-credential.without.required-segmentation"
const discordTestBot = `{"id":"123456789012345678","username":"voxel_bot","global_name":"Voxel Bot","bot":true}`
const discordTestApp = `{"id":"987654321098765432","name":"Voxel"}`

func discordTestServer(t *testing.T) *server {
	t.Helper()
	store, _ := openTestWebUIStore(t)
	return &server{store: store, sessions: map[string]session{}}
}

func discordAdminRequest(s *server, actor session, body any) *httptest.ResponseRecorder {
	raw, _ := json.Marshal(body)
	method := http.MethodPost
	if body == nil {
		method, raw = http.MethodGet, nil
	}
	s.sessions["discord-browser-session"] = actor
	r := httptest.NewRequest(method, "/v1/admin/discord", bytes.NewReader(raw))
	r.Header.Set("Authorization", "Bearer discord-browser-session")
	mux := http.NewServeMux()
	registerDiscordRoutes(mux, s)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, r)
	return rr
}

func configureTestDiscord(t *testing.T, s *server) {
	t.Helper()
	rr := discordAdminRequest(s, administratorSession(time.Now()), map[string]any{"action": "credential", "token": discordTestToken})
	if rr.Code != 200 || strings.Contains(rr.Body.String(), discordTestToken) || rr.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("credential was rejected or disclosed", rr.Code)
	}
}

func useDiscordTestTransport(t *testing.T, f func(*http.Request) (*http.Response, error)) {
	t.Helper()
	old := discordHTTP
	client := *discordHTTP
	client.Transport = telegramRoundTrip(f)
	discordHTTP = &client
	t.Cleanup(func() { discordHTTP = old })
}

func discordTestResponse(r *http.Request, status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}
}

func TestDiscordMigrationAndPrivatePersistence(t *testing.T) {
	store, path := openTestWebUIStore(t)
	fresh, err := store.discordRead()
	if err != nil || fresh.Schema != 1 || fresh.Token != "" || fresh.Verified || fresh.Status != "disconnected" {
		t.Fatal("fresh Discord state is not disconnected", err)
	}
	var name string
	if err := store.db.QueryRow(`SELECT name FROM schema_migrations WHERE version=8`).Scan(&name); err != nil || name != "native_discord" {
		t.Fatal("Discord migration missing", err)
	}
	s := &server{store: store, sessions: map[string]session{}}
	configureTestDiscord(t, s)
	var raw string
	if err := store.db.QueryRow(`SELECT record_json FROM discord_state WHERE id=1`).Scan(&raw); err != nil || !strings.Contains(raw, discordTestToken) {
		t.Fatal("private credential not persisted", err)
	}
	before, err := store.discordRead()
	if err != nil {
		t.Fatal(err)
	}
	if err = store.close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := openWebUIStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.close()
	after, err := reopened.discordRead()
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("Discord state did not survive reopen", err)
	}
	s.store = reopened
	rr := discordAdminRequest(s, administratorSession(time.Now()), nil)
	if rr.Code != 200 || rr.Header().Get("Cache-Control") != "no-store" || strings.Contains(rr.Body.String(), discordTestToken) || strings.Contains(rr.Body.String(), `"token"`) {
		t.Fatal("admin state exposes credential", rr.Code)
	}
	public, err := json.Marshal(discordPublicState(after))
	if err != nil || bytes.Contains(public, []byte(discordTestToken)) || bytes.Contains(public, []byte(`"token"`)) {
		t.Fatal("public state exposes credential", err)
	}
	var leaks int
	if err := reopened.db.QueryRow(`SELECT count(*) FROM audit_events WHERE context LIKE ? OR target LIKE ? OR action LIKE ?`, "%"+discordTestToken+"%", "%"+discordTestToken+"%", "%"+discordTestToken+"%").Scan(&leaks); err != nil || leaks != 0 {
		t.Fatal("credential in audit", err)
	}
}

func TestDiscordVerifyAndCredentialReplacement(t *testing.T) {
	s := telegramTestServer(t)
	telegramBefore, err := s.store.telegramRead()
	if err != nil {
		t.Fatal(err)
	}
	configureTestDiscord(t, s)
	calls := []string{}
	useDiscordTestTransport(t, func(r *http.Request) (*http.Response, error) {
		if r.Method != "GET" || r.URL.Scheme != "https" || r.URL.Host != "discord.com" || r.URL.RawQuery != "" || r.Header.Get("Authorization") != "Bot "+discordTestToken || strings.Contains(r.URL.String(), discordTestToken) {
			t.Fatal("unsafe Discord verification request")
		}
		if r.Header.Get("User-Agent") != "DiscordBot (https://github.com/home-server-project/justvoxel, 1.0)" {
			t.Fatal("incorrect Discord User-Agent")
		}
		if _, ok := r.Context().Deadline(); !ok {
			t.Fatal("verification has no deadline")
		}
		calls = append(calls, r.URL.Path)
		switch r.URL.Path {
		case "/api/v10/users/@me":
			return discordTestResponse(r, 200, discordTestBot), nil
		case "/api/v10/applications/@me":
			return discordTestResponse(r, 200, discordTestApp), nil
		default:
			t.Fatal("unexpected verification endpoint")
			return nil, errors.New("denied")
		}
	})
	actor := administratorSession(time.Now())
	rr := discordAdminRequest(s, actor, map[string]any{"action": "verify"})
	v, err := s.store.discordRead()
	if rr.Code != 200 || err != nil || !v.Verified || v.Status != "verified" || v.BotID != "123456789012345678" || v.BotUsername != "voxel_bot" || v.BotName != "Voxel Bot" || v.ApplicationID != "987654321098765432" || v.ApplicationName != "Voxel" || strings.Contains(rr.Body.String(), discordTestToken) {
		t.Fatal("authoritative identity not saved safely", rr.Code, err)
	}
	if !reflect.DeepEqual(calls, []string{"/api/v10/users/@me", "/api/v10/applications/@me"}) {
		t.Fatal("verification did not use both v10 endpoints", calls)
	}
	var public struct {
		BotID         string `json:"bot_id"`
		ApplicationID string `json:"application_id"`
	}
	if json.Unmarshal(rr.Body.Bytes(), &public) != nil || public.BotID != v.BotID || public.ApplicationID != v.ApplicationID {
		t.Fatal("snowflakes not returned as strings")
	}
	generation := v.Generation
	rr = discordAdminRequest(s, actor, map[string]any{"action": "credential", "token": "replacement-opaque-token"})
	v, err = s.store.discordRead()
	if rr.Code != 200 || err != nil || v.Token != "replacement-opaque-token" || v.Generation <= generation || v.Verified || v.BotID != "" || v.BotUsername != "" || v.BotName != "" || v.ApplicationID != "" || v.ApplicationName != "" || v.Status != "unverified" {
		t.Fatal("credential replacement retained identity", rr.Code, err)
	}
	telegramAfter, err := s.store.telegramRead()
	if err != nil || !reflect.DeepEqual(telegramBefore, telegramAfter) {
		t.Fatal("Discord configuration or verification changed Telegram state", err)
	}
}

func TestDiscordVerificationRejectsInvalidIdentityAndHTTPFailures(t *testing.T) {
	for _, tc := range []struct {
		name, bot, app       string
		botStatus, appStatus int
	}{
		{"non_bot", strings.Replace(discordTestBot, `"bot":true`, `"bot":false`, 1), discordTestApp, 200, 200},
		{"missing_bot", `{"id":"123","username":"voxel"}`, discordTestApp, 200, 200},
		{"empty_id", `{"id":"","username":"voxel","bot":true}`, discordTestApp, 200, 200},
		{"numeric_json_id", `{"id":123,"username":"voxel","bot":true}`, discordTestApp, 200, 200},
		{"non_numeric_bot", `{"id":"123x","username":"voxel","bot":true}`, discordTestApp, 200, 200},
		{"zero_bot", `{"id":"0","username":"voxel","bot":true}`, discordTestApp, 200, 200},
		{"overflow_bot", `{"id":"18446744073709551616","username":"voxel","bot":true}`, discordTestApp, 200, 200},
		{"missing_username", `{"id":"123","bot":true}`, discordTestApp, 200, 200},
		{"non_numeric_app", discordTestBot, `{"id":"app"}`, 200, 200},
		{"missing_app_id", discordTestBot, `{"name":"Voxel"}`, 200, 200},
		{"numeric_json_app", discordTestBot, `{"id":123}`, 200, 200},
		{"user_http_failure", discordTestBot, discordTestApp, 401, 200},
		{"app_http_failure", discordTestBot, discordTestApp, 200, 503},
		{"rate_limited", discordTestBot, discordTestApp, 200, 429},
		{"malformed_app", discordTestBot, `not json`, 200, 200},
		{"oversized_app", discordTestBot, strings.Repeat("x", (1<<20)+1), 200, 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := discordTestServer(t)
			configureTestDiscord(t, s)
			useDiscordTestTransport(t, func(r *http.Request) (*http.Response, error) {
				if r.URL.Path == "/api/v10/users/@me" {
					return discordTestResponse(r, tc.botStatus, tc.bot), nil
				}
				return discordTestResponse(r, tc.appStatus, tc.app), nil
			})
			rr := discordAdminRequest(s, administratorSession(time.Now()), map[string]any{"action": "verify"})
			v, err := s.store.discordRead()
			if rr.Code != 503 || err != nil || v.Verified || v.BotID != "" || v.ApplicationID != "" || v.Status != "unverified" {
				t.Fatal("failed verification saved identity", rr.Code, err)
			}
			if strings.Contains(rr.Body.String(), discordTestToken) {
				t.Fatal("verification error disclosed credential")
			}
		})
	}
}

func TestDiscordOptionalNamesAndUsernameFallback(t *testing.T) {
	s := discordTestServer(t)
	configureTestDiscord(t, s)
	useDiscordTestTransport(t, func(r *http.Request) (*http.Response, error) {
		body := `{"id":"123","username":"voxel_bot","global_name":null,"bot":true}`
		if r.URL.Path == "/api/v10/applications/@me" {
			body = `{"id":"456"}`
		}
		return discordTestResponse(r, 200, body), nil
	})
	rr := discordAdminRequest(s, administratorSession(time.Now()), map[string]any{"action": "verify"})
	v, err := s.store.discordRead()
	if rr.Code != 200 || err != nil || !v.Verified || v.BotName != "voxel_bot" || v.ApplicationName != "" {
		t.Fatal("optional identity names rejected", rr.Code, err)
	}
}

func TestDiscordTransportRefusesRedirectsAndUnapprovedEndpoints(t *testing.T) {
	calls := 0
	useDiscordTestTransport(t, func(r *http.Request) (*http.Response, error) {
		calls++
		response := discordTestResponse(r, 302, discordTestBot)
		response.Header.Set("Location", "https://another-host.example/stolen")
		return response, nil
	})
	var out any
	if err := discordRequest(context.Background(), discordTestToken, "/users/@me", &out); err == nil || calls != 1 {
		t.Fatal("redirect not refused", calls, err)
	}
	for _, path := range []string{"https://another-host.example", "/users/@me?token=secret", "/gateway/bot", "/applications/@me/commands"} {
		if err := discordRequest(context.Background(), discordTestToken, path, &out); err == nil {
			t.Fatal("unapproved endpoint accepted")
		}
	}
	if calls != 1 {
		t.Fatal("unapproved endpoint reached transport")
	}
	if discordHTTP.Timeout <= 0 || discordHTTP.Timeout > 20*time.Second {
		t.Fatal("unbounded transport timeout")
	}
}

func TestDiscordErrorsNeverExposePrivateTransportOrResponse(t *testing.T) {
	for _, transportFailure := range []bool{true, false} {
		t.Run(map[bool]string{true: "transport", false: "http"}[transportFailure], func(t *testing.T) {
			s := discordTestServer(t)
			configureTestDiscord(t, s)
			useDiscordTestTransport(t, func(r *http.Request) (*http.Response, error) {
				if transportFailure {
					return nil, errors.New("private diagnostic Authorization: Bot " + discordTestToken)
				}
				return discordTestResponse(r, 401, `{"message":"private diagnostic `+discordTestToken+`"}`), nil
			})
			var out any
			err := discordRequest(context.Background(), discordTestToken, "/users/@me", &out)
			if err == nil || strings.Contains(err.Error(), discordTestToken) || strings.Contains(err.Error(), "Authorization") || strings.Contains(err.Error(), "private diagnostic") {
				t.Fatal("unsafe transport error")
			}
			rr := discordAdminRequest(s, administratorSession(time.Now()), map[string]any{"action": "verify"})
			if rr.Code != 503 || strings.Contains(rr.Body.String(), discordTestToken) || strings.Contains(rr.Body.String(), "Authorization") || strings.Contains(rr.Body.String(), "private diagnostic") {
				t.Fatal("unsafe admin error", rr.Code)
			}
		})
	}
}

func TestDiscordAdministrationRequiresPrimaryHumanWebUISession(t *testing.T) {
	s := discordTestServer(t)
	for _, actor := range []session{
		{Username: "viewer", Role: roleViewer, AuthSource: authSourceSeparate},
		{Username: "operator", Role: roleOperator, AuthSource: authSourceSystem},
		{Username: "other", Role: roleAdministrator, AuthSource: authSourceSystem},
		{Username: "voxel", Role: roleAdministrator, AuthSource: authSourceExternal},
		{Username: "voxel", Role: roleAdministrator, AuthSource: authSourceLocalRoot},
		{Username: "voxel", Role: roleAdministrator, AuthSource: authSourceWebUI},
		{Username: "voxel", Role: roleAdministrator, AuthSource: authSourceSystem, ConnectionID: "api-principal"},
		{Username: "voxel", Role: roleAdministrator, AuthSource: authSourceSystem, MustChange: true},
		telegramActor(456),
	} {
		actor.Created, actor.LastSeen = time.Now(), time.Now()
		for _, body := range []any{nil, map[string]any{"action": "credential", "token": discordTestToken}, map[string]any{"action": "verify"}, map[string]any{"action": "disconnect"}} {
			rr := discordAdminRequest(s, actor, body)
			if rr.Code != 401 && rr.Code != 403 {
				t.Fatal("non-primary/non-human principal accepted", rr.Code, actor.AuthSource)
			}
		}
	}
	for _, source := range []authSource{authSourceSystem, authSourceSeparate} {
		actor := administratorSession(time.Now())
		actor.AuthSource = source
		rr := discordAdminRequest(s, actor, map[string]any{"action": "credential", "token": discordTestToken})
		if rr.Code != 200 {
			t.Fatal("primary human session rejected", source, rr.Code)
		}
	}
	// The real local root socket path also must not configure Discord.
	r := httptest.NewRequest("GET", "/v1/admin/discord", nil)
	r = r.WithContext(context.WithValue(r.Context(), peerUIDKey{}, uint32(0)))
	rr := httptest.NewRecorder()
	s.discordAdmin(rr, r)
	if rr.Code != 403 {
		t.Fatal("root principal accepted", rr.Code)
	}
}

func TestDiscordCredentialInputAndStrictActions(t *testing.T) {
	s := discordTestServer(t)
	actor := administratorSession(time.Now())
	for _, token := range []string{"", " ", "a b", "a\tb", "a\nb", "a\rb", "a\x00b", "a\x7fb", "a\u00a0b", strings.Repeat("x", 2049)} {
		rr := discordAdminRequest(s, actor, map[string]any{"action": "credential", "token": token})
		if rr.Code != 400 {
			t.Fatal("invalid credential accepted", rr.Code)
		}
	}
	for _, token := range []string{"opaque-token", "no.segments.required.here", strings.Repeat("x", 2048)} {
		rr := discordAdminRequest(s, actor, map[string]any{"action": "credential", "token": token})
		if rr.Code != 200 {
			t.Fatal("opaque credential rejected", rr.Code)
		}
	}
	for _, action := range []string{"enabled", "pair", "guild", "notify", "commands", ""} {
		if rr := discordAdminRequest(s, actor, map[string]any{"action": action}); rr.Code != 400 {
			t.Fatal("unsupported action accepted", rr.Code)
		}
	}
	for _, body := range []string{`{"action":"disconnect","action":"verify"}`, `{"action":"disconnect"} {}`, `{"action":"credential","token":"secret","url":"https://example.com"}`} {
		r := httptest.NewRequest("POST", "/v1/admin/discord", strings.NewReader(body))
		r.Header.Set("Authorization", "Bearer discord-browser-session")
		rr := httptest.NewRecorder()
		s.discordAdmin(rr, r)
		if rr.Code != 400 {
			t.Fatal("ambiguous or unknown input accepted", rr.Code)
		}
	}
	fresh := discordTestServer(t)
	if rr := discordAdminRequest(fresh, actor, map[string]any{"action": "verify"}); rr.Code != 400 {
		t.Fatal("verification without credential accepted", rr.Code)
	}
}

func TestDiscordDisconnectAndFactoryRevocationAreIndependentAndRecoverMalformedState(t *testing.T) {
	for _, factory := range []bool{false, true} {
		for _, malformed := range []bool{false, true} {
			s := telegramTestServer(t)
			before, err := s.store.telegramRead()
			if err != nil {
				t.Fatal(err)
			}
			gatewayBefore, err := s.store.gatewaySnapshot()
			if err != nil {
				t.Fatal(err)
			}
			v := discordState{Schema: 1, Token: discordTestToken, BotID: "123", BotUsername: "voxel", BotName: "Voxel", ApplicationID: "456", ApplicationName: "App", Verified: true, Status: "verified", Generation: 10}
			if err := s.store.discordSave(v, nil, ""); err != nil {
				t.Fatal(err)
			}
			if malformed {
				if _, err := s.store.db.Exec(`UPDATE discord_state SET record_json='{"schema":999,"token":"malformed-private-state"}' WHERE id=1`); err != nil {
					t.Fatal(err)
				}
			}
			for i := 0; i < 2; i++ {
				if factory {
					if err := s.discordFactoryRevoke(); err != nil {
						t.Fatal("Discord corruption stranded Factory Reset", err)
					}
				} else {
					rr := discordAdminRequest(s, administratorSession(time.Now()), map[string]any{"action": "disconnect"})
					if rr.Code != 200 {
						t.Fatal("Discord corruption stranded disconnect", rr.Code)
					}
				}
				after, err := s.store.discordRead()
				if err != nil || after.Token != "" || after.Verified || after.BotID != "" || after.BotUsername != "" || after.BotName != "" || after.ApplicationID != "" || after.ApplicationName != "" || after.Status != "disconnected" {
					t.Fatal("incomplete Discord revocation", err)
				}
			}
			telegramAfter, err := s.store.telegramRead()
			if err != nil || !reflect.DeepEqual(before, telegramAfter) {
				t.Fatal("Discord changed Telegram", err)
			}
			gatewayAfter, err := s.store.gatewaySnapshot()
			if err != nil || !reflect.DeepEqual(gatewayBefore, gatewayAfter) {
				t.Fatal("Discord changed gateway", err)
			}
		}
	}
}

func TestDiscordMigrationFromTelegramDatabasePreservesExistingState(t *testing.T) {
	store, path := openTestWebUIStore(t)
	if err := store.telegramSave(telegramTestState(), nil, ""); err != nil {
		t.Fatal(err)
	}
	_, gatewayToken := createTestGatewayConnection(t, store, 4)
	telegramBefore, err := store.telegramRead()
	if err != nil {
		t.Fatal(err)
	}
	gatewayBefore, err := store.gatewaySnapshot()
	if err != nil {
		t.Fatal(err)
	}
	// Reconstruct the deployed v7 schema by removing all post-v7 migration
	// objects and history. Reopen must apply v8-v12 without altering existing state.
	if _, err := store.db.Exec(`DROP TABLE routeros_state`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`DROP TABLE web_user_two_factor`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`DROP TABLE discord_reboot_state`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`DROP TABLE discord_access_requests`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`DROP TABLE discord_state`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`DELETE FROM schema_migrations WHERE version>=8`); err != nil {
		t.Fatal(err)
	}
	if err := store.close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := openWebUIStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.close()
	discord, err := reopened.discordRead()
	if err != nil || !reflect.DeepEqual(discord, discordState{Schema: 1, Status: "disconnected"}) {
		t.Fatal("upgrade did not add disconnected Discord state", err)
	}
	telegramAfter, err := reopened.telegramRead()
	if err != nil || !reflect.DeepEqual(telegramBefore, telegramAfter) {
		t.Fatal("Discord migration changed Telegram", err)
	}
	gatewayAfter, err := reopened.gatewaySnapshot()
	if err != nil || !reflect.DeepEqual(gatewayBefore, gatewayAfter) {
		t.Fatal("Discord migration changed gateway", err)
	}
	if _, err := reopened.gatewayAuthenticate(gatewayToken); err != nil {
		t.Fatal("Discord migration revoked gateway credential", err)
	}
}
