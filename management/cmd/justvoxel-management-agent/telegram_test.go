package main

import (
	"bytes"
	"context"
	"encoding/hex"
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

const telegramTestToken = "123456789:" + "abcdefghijklmnopqrstuvwxyzABCDEFG1234567890"

func telegramTestState() telegramState {
	return telegramState{Schema: 1, Token: telegramTestToken, Enabled: true, Verified: true, Generation: 1, Status: "connected", Users: []telegramUser{{ID: 123, Permission: telegramReadOnly}, {ID: 456, Permission: telegramManagement, Notifications: true}}, Callbacks: map[string]telegramCallback{}, Seen: map[string]string{}}
}
func telegramTestServer(t *testing.T) *server {
	t.Helper()
	store, _ := openTestWebUIStore(t)
	s := &server{store: store, sessions: map[string]session{}}
	if err := store.telegramSave(telegramTestState(), nil, ""); err != nil {
		t.Fatal(err)
	}
	return s
}
func telegramAdminRequest(s *server, actor session, body any) *httptest.ResponseRecorder {
	raw, _ := json.Marshal(body)
	method := "POST"
	if body == nil {
		method = "GET"
		raw = nil
	}
	request := httptest.NewRequest(method, "/v1/admin/telegram", bytes.NewReader(raw))
	request = request.WithContext(context.WithValue(request.Context(), gatewayPrincipalKey{}, actor))
	rr := httptest.NewRecorder()
	s.telegramAdmin(rr, request)
	return rr
}
func TestTelegramIndependentPermissionAllowlist(t *testing.T) {
	v := telegramTestState()
	reads := []string{"status", "instance_status", "instance_details", "players", "backups", "backup_status", "network_status", "version_status", "container_status", "version_update_status", "container_update_status", "system_updates", "system_update_check", "menu", "instance_menu", "network_menu", "notifications", "notifications_toggle"}
	writes := []string{"instance_start", "instance_stop", "instance_restart", "backup_now", "start_on_boot", "network_change", "version_update", "container_update", "system_download", "update_reboot", "reboot", "version_review", "container_review", "reboot_menu"}
	for _, name := range reads {
		if !telegramAllowed(&v, 123, name) || !telegramAllowed(&v, 456, name) {
			t.Fatal("read denied", name)
		}
	}
	for _, name := range writes {
		if telegramAllowed(&v, 123, name) || !telegramAllowed(&v, 456, name) {
			t.Fatal("write permission mismatch", name)
		}
	}
	for _, name := range append(append([]string{}, reads...), writes...) {
		if telegramAllowed(&v, 999, name) {
			t.Fatal("unapproved identity allowed", name)
		}
	}
	for _, name := range []string{"systemd", "shell", "poweroff", "instance_create", "instance_delete", "instance_configure", "network", "network_wifi_connect", "storage", "restore_apply", "migration_recovery_apply", "plugin_install", "users", "factory_reset"} {
		if telegramAllowed(&v, 456, name) {
			t.Fatal("forbidden capability", name)
		}
	}
	v.Enabled = false
	if telegramAllowed(&v, 456, "status") {
		t.Fatal("disabled Telegram authorized")
	}
}
func TestTelegramPairingRequiresChallengeAndExplicitHumanApproval(t *testing.T) {
	s := telegramTestServer(t)
	actor := administratorSession(time.Now())
	rr := telegramAdminRequest(s, actor, map[string]any{"action": "pair"})
	if rr.Code != 200 {
		t.Fatal(rr.Code, rr.Body.String())
	}
	var result struct {
		Challenge string `json:"pairing_challenge"`
	}
	if json.Unmarshal(rr.Body.Bytes(), &result) != nil || len(result.Challenge) < 40 {
		t.Fatal("missing challenge")
	}
	v, err := s.store.telegramRead()
	if err != nil {
		t.Fatal(err)
	}
	if telegramClaimPair(&v, 999, "/start", s.store.now()) || telegramClaimPair(&v, 999, "/start incorrect", s.store.now()) {
		t.Fatal("unverified pairing claimed")
	}
	old := telegramRequest
	defer func() { telegramRequest = old }()
	sends := 0
	telegramRequest = func(_ context.Context, _ string, method string, input, out any) error {
		if method != "sendMessage" {
			t.Fatal("unexpected Telegram request", method)
		}
		sends++
		body := input.(map[string]any)
		if body["chat_id"] != int64(999) || body["text"] != "Pairing request received. The primary administrator must approve your numeric Telegram ID in JustVoxel." {
			t.Fatal("unexpected pairing acknowledgement")
		}
		return nil
	}
	var update telegramUpdate
	if json.Unmarshal([]byte(`{"update_id":20,"message":{"from":{"id":999},"chat":{"id":999,"type":"private"}}}`), &update) != nil {
		t.Fatal("fixture")
	}
	update.Message.Text = "/start " + result.Challenge
	s.telegramProcessUpdate(context.Background(), v.Generation, update)
	s.telegramProcessUpdate(context.Background(), v.Generation, update)
	v, err = s.store.telegramRead()
	if err != nil || sends != 1 || v.Offset != 21 || v.Pair == nil || v.Pair.UserID != 999 || v.Pair.Digest != "" {
		t.Fatal("challenge not consumed exactly once", err)
	}
	if _, approved := v.user(999); approved || telegramAllowed(&v, 999, "status") {
		t.Fatal("Start self-authorized")
	}
	update.ID++
	update.Message.From.ID, update.Message.Chat.ID = 888, 888
	s.telegramProcessUpdate(context.Background(), v.Generation, update)
	v, err = s.store.telegramRead()
	if err != nil || sends != 1 || v.Pair == nil || v.Pair.UserID != 999 {
		t.Fatal("challenge reused", err)
	}
	if _, approved := v.user(888); approved {
		t.Fatal("different numeric ID self-authorized")
	}
	rr = telegramAdminRequest(s, actor, map[string]any{"action": "approve", "user_id": 888, "permission": telegramManagement})
	if rr.Code != 409 {
		t.Fatal("different numeric ID approved", rr.Code)
	}
	rr = telegramAdminRequest(s, actor, map[string]any{"action": "approve", "user_id": 999, "permission": telegramReadOnly})
	if rr.Code != 200 {
		t.Fatal(rr.Code, rr.Body.String())
	}
	v, _ = s.store.telegramRead()
	if _, approved := v.user(999); !approved || v.Pair != nil {
		t.Fatal("approval not persisted or challenge retained")
	}
}
func TestTelegramPairChallengeExpiresAndIsNotRedisplayed(t *testing.T) {
	now := time.Now()
	v := telegramTestState()
	challenge := "one-time-challenge"
	v.Pair = &telegramPair{Digest: hex.EncodeToString(gatewayDigest(challenge)), Expires: now}
	if telegramClaimPair(&v, 999, "/start "+challenge, now) {
		t.Fatal("expired pairing accepted")
	}
	raw, _ := json.Marshal(telegramPublicState(v, now.Add(-time.Second)))
	if bytes.Contains(raw, []byte(challenge)) || bytes.Contains(raw, []byte(v.Pair.Digest)) || bytes.Contains(raw, []byte(telegramTestToken)) {
		t.Fatal("private state redisplayed")
	}
	s := telegramTestServer(t)
	if err := s.store.telegramSave(v, nil, ""); err != nil {
		t.Fatal(err)
	}
	old := telegramRequest
	defer func() { telegramRequest = old }()
	telegramRequest = func(context.Context, string, string, any, any) error {
		t.Fatal("expired challenge received a reply")
		return nil
	}
	var update telegramUpdate
	if json.Unmarshal([]byte(`{"update_id":20,"message":{"from":{"id":999},"chat":{"id":999,"type":"private"}}}`), &update) != nil {
		t.Fatal("fixture")
	}
	update.Message.Text = "/start " + challenge
	s.telegramProcessUpdate(context.Background(), v.Generation, update)
	v, err := s.store.telegramRead()
	if err != nil || v.Pair == nil || v.Pair.UserID != 0 {
		t.Fatal("expired challenge claimed", err)
	}
	if _, approved := v.user(999); approved || telegramAllowed(&v, 999, "status") {
		t.Fatal("expired challenge authorized user")
	}
}
func TestTelegramAdministrationRejectsNonHumanAndNonPrimaryPrincipals(t *testing.T) {
	s := telegramTestServer(t)
	for _, actor := range []session{{Username: "viewer", Role: roleViewer, AuthSource: authSourceWebUI}, {Username: "other", Role: roleAdministrator, AuthSource: authSourceSystem}, {Username: "voxel", Role: roleAdministrator, AuthSource: authSourceExternal}, {Username: "voxel", Role: roleAdministrator, AuthSource: authSourceLocalRoot}, telegramActor(456)} {
		rr := telegramAdminRequest(s, actor, map[string]any{"action": "pair"})
		if rr.Code == 200 {
			t.Fatal("non-primary/human admin accepted", actor)
		}
	}
}
func TestTelegramCallbackBindsIdentityOperationInstanceAndExpires(t *testing.T) {
	now := time.Now()
	v := telegramTestState()
	original := telegramCallback{UserID: 456, Operation: "instance_stop", InstanceID: "jv-abcdef", Body: json.RawMessage(`{"confirm_players":false}`), Confirmed: true}
	key, err := v.issueCallback(original, now)
	if err != nil || len(key) > 64 {
		t.Fatal(key, err)
	}
	if _, allowed := v.consumeCallback(key, 123, now); allowed {
		t.Fatal("another identity consumed button")
	}
	cb, allowed := v.consumeCallback(key, 456, now)
	if !allowed || cb.Operation != original.Operation || cb.InstanceID != original.InstanceID || !bytes.Equal(cb.Body, original.Body) || !cb.Confirmed {
		t.Fatal("button binding changed", cb)
	}
	if _, allowed = v.consumeCallback(key, 456, now); allowed {
		t.Fatal("callback replayed")
	}
	key, _ = v.issueCallback(original, now)
	if _, allowed = v.consumeCallback(key, 456, now.Add(5*time.Minute)); allowed {
		t.Fatal("expired callback accepted")
	}
}
func TestTelegramRevocationAndDowngradeInvalidateOldButtons(t *testing.T) {
	for _, action := range []string{"revoke", "user"} {
		t.Run(action, func(t *testing.T) {
			s := telegramTestServer(t)
			v, _ := s.store.telegramRead()
			key, _ := v.issueCallback(telegramCallback{UserID: 456, Operation: "instance_restart", InstanceID: "jv-abcdef", Confirmed: true}, s.store.now())
			if err := s.store.telegramSave(v, nil, ""); err != nil {
				t.Fatal(err)
			}
			rr := telegramAdminRequest(s, administratorSession(time.Now()), map[string]any{"action": action, "user_id": 456, "permission": telegramReadOnly})
			if rr.Code != 200 {
				t.Fatal(rr.Code)
			}
			v, _ = s.store.telegramRead()
			if _, allowed := v.consumeCallback(key, 456, s.store.now()); allowed || telegramAllowed(&v, 456, "instance_restart") {
				t.Fatal("old management access survived")
			}
		})
	}
}
func TestTelegramCredentialReplacementAndCompleteDisconnect(t *testing.T) {
	s := telegramTestServer(t)
	actor := administratorSession(time.Now())
	replacement := "987654321:" + strings.Repeat("X", 40)
	rr := telegramAdminRequest(s, actor, map[string]any{"action": "credential", "token": replacement})
	if rr.Code != 200 || bytes.Contains(rr.Body.Bytes(), []byte(replacement)) {
		t.Fatal("replacement failed or exposed token", rr.Code)
	}
	v, _ := s.store.telegramRead()
	if v.Token != replacement || v.Enabled || v.Verified || len(v.Users) != 0 {
		t.Fatal("old bot approvals retained")
	}
	rr = telegramAdminRequest(s, actor, map[string]any{"action": "disconnect"})
	v, _ = s.store.telegramRead()
	if rr.Code != 200 || v.Token != "" || v.Pair != nil || len(v.Users) != 0 || v.Enabled {
		t.Fatal("incomplete disconnection")
	}
}
func TestTelegramVerificationRejectsWebhookAndNeverReturnsCredential(t *testing.T) {
	s := telegramTestServer(t)
	old := telegramRequest
	defer func() { telegramRequest = old }()
	webhook := "https://another-consumer.example/hook"
	telegramRequest = func(_ context.Context, _ string, method string, _ any, out any) error {
		var raw string
		switch method {
		case "getMe":
			raw = `{"id":777,"username":"example_bot","is_bot":true}`
		case "getWebhookInfo":
			raw = `{"url":"` + webhook + `"}`
		default:
			t.Fatal("unexpected method", method)
		}
		return json.Unmarshal([]byte(raw), out)
	}
	actor := administratorSession(time.Now())
	rr := telegramAdminRequest(s, actor, map[string]any{"action": "verify"})
	if rr.Code != 409 || strings.Contains(rr.Body.String(), webhook) || strings.Contains(rr.Body.String(), telegramTestToken) {
		t.Fatal("webhook accepted or credential leaked", rr.Code)
	}
	webhook = ""
	rr = telegramAdminRequest(s, actor, map[string]any{"action": "verify"})
	if rr.Code != 200 || strings.Contains(rr.Body.String(), telegramTestToken) {
		t.Fatal("verification failed or token exposed", rr.Code)
	}
}
func TestTelegramUpdateConsumptionAndUnknownStartCannotAuthorize(t *testing.T) {
	s := telegramTestServer(t)
	old := telegramRequest
	defer func() { telegramRequest = old }()
	sends := 0
	telegramRequest = func(_ context.Context, _ string, method string, input, out any) error {
		if method == "sendMessage" {
			sends++
		} else {
			t.Fatal("unexpected Telegram request", method)
		}
		return nil
	}
	var update telegramUpdate
	if json.Unmarshal([]byte(`{"update_id":20,"message":{"from":{"id":999},"chat":{"id":999,"type":"private"},"text":"/start"}}`), &update) != nil {
		t.Fatal("fixture")
	}
	for index, text := range []string{"/start", "/menu", "/status", "/help", "random text", "/start incorrect"} {
		update.ID = int64(20 + index)
		update.Message.Text = text
		s.telegramProcessUpdate(context.Background(), 1, update)
		s.telegramProcessUpdate(context.Background(), 1, update)
		v, err := s.store.telegramRead()
		if err != nil || sends != 0 || v.Offset != update.ID+1 || v.Pair != nil {
			t.Fatal("unknown message replied, replayed or created pairing", text, sends, err)
		}
		if _, approved := v.user(999); approved || telegramAllowed(&v, 999, "status") {
			t.Fatal("unknown account self-authorized")
		}
	}
	var denied int
	if err := s.store.db.QueryRow(`SELECT count(*) FROM audit_events WHERE action='telegram_denied'`).Scan(&denied); err != nil || denied != 0 {
		t.Fatal("ordinary unknown messages generated denial noise", denied, err)
	}
}

func TestTelegramUnapprovedCallbacksAreSilent(t *testing.T) {
	for _, revoked := range []bool{false, true} {
		name := "unknown"
		if revoked {
			name = "revoked"
		}
		t.Run(name, func(t *testing.T) {
			s := telegramTestServer(t)
			v, _ := s.store.telegramRead()
			id := int64(999)
			if revoked {
				id = 456
			}
			key, err := v.issueCallback(telegramCallback{UserID: id, Operation: "instance_restart", InstanceID: "jv-abcdef", Confirmed: true}, s.store.now())
			if err != nil {
				t.Fatal(err)
			}
			if revoked {
				v.Users = v.Users[:1]
			}
			challenge := "active-challenge"
			v.Pair = &telegramPair{Digest: hex.EncodeToString(gatewayDigest(challenge)), Expires: s.store.now().Add(time.Minute)}
			if err := s.store.telegramSave(v, nil, ""); err != nil {
				t.Fatal(err)
			}
			old := telegramRequest
			defer func() { telegramRequest = old }()
			acks := 0
			telegramRequest = func(_ context.Context, _ string, method string, input, out any) error {
				body := input.(map[string]any)
				if method != "answerCallbackQuery" || len(body) != 1 || body["callback_query_id"] != "q" {
					t.Fatal("unapproved callback received informational response", method)
				}
				acks++
				return nil
			}
			var update telegramUpdate
			if json.Unmarshal([]byte(`{"update_id":20,"callback_query":{"id":"q","from":{"id":999},"message":{"chat":{"id":999,"type":"private"}}}}`), &update) != nil {
				t.Fatal("fixture")
			}
			update.Callback.From.ID, update.Callback.Message.Chat.ID = id, id
			update.Callback.Message.Text = "/start " + challenge
			// Even an empty callback must not claim a challenge from message text.
			for index, data := range []string{key, ""} {
				update.ID = int64(20 + index)
				update.Callback.Data = data
				s.telegramProcessUpdate(context.Background(), v.Generation, update)
				s.telegramProcessUpdate(context.Background(), v.Generation, update)
			}
			v, err = s.store.telegramRead()
			if err != nil || acks != 2 || v.Pair == nil || v.Pair.UserID != 0 || len(v.Callbacks) != 1 {
				t.Fatal("unapproved callback dispatched, replayed or claimed pairing", err)
			}
			if _, approved := v.user(id); approved {
				t.Fatal("callback authorized user")
			}
			var attempts int
			if err := s.store.db.QueryRow(`SELECT count(*) FROM audit_events WHERE action='telegram_attempt_instance_restart'`).Scan(&attempts); err != nil || attempts != 0 {
				t.Fatal("unapproved callback attempted management action", attempts, err)
			}
		})
	}
}
func TestTelegramGroupAndForwardedCallbacksDoNotExposeInformation(t *testing.T) {
	s := telegramTestServer(t)
	v, _ := s.store.telegramRead()
	v.Pair = &telegramPair{Digest: hex.EncodeToString(gatewayDigest("active-challenge")), Expires: s.store.now().Add(time.Minute)}
	if err := s.store.telegramSave(v, nil, ""); err != nil {
		t.Fatal(err)
	}
	old := telegramRequest
	defer func() { telegramRequest = old }()
	calls := 0
	telegramRequest = func(context.Context, string, string, any, any) error { calls++; return nil }
	for index, raw := range []string{`{"message":{"from":{"id":456},"chat":{"id":-555,"type":"group"},"text":"/menu"}}`, `{"callback_query":{"id":"q","from":{"id":456},"message":{"chat":{"id":123,"type":"private"}},"data":"old"}}`, `{"message":{"from":{"id":999},"chat":{"id":-555,"type":"group"},"text":"/start active-challenge"}}`, `{"message":{"from":{"id":999},"chat":{"id":888,"type":"private"},"text":"/start active-challenge"}}`} {
		var update telegramUpdate
		if json.Unmarshal([]byte(raw), &update) != nil {
			t.Fatal("fixture")
		}
		update.ID = int64(index + 100)
		s.telegramProcessUpdate(context.Background(), 1, update)
	}
	if calls != 0 {
		t.Fatal("group or different private chat reached Telegram reply")
	}
	v, err := s.store.telegramRead()
	if err != nil || v.Pair == nil || v.Pair.UserID != 0 {
		t.Fatal("group or different private chat claimed pairing", err)
	}
}
func TestTelegramNotificationOptInDedupAndRestartPersistence(t *testing.T) {
	s := telegramTestServer(t)
	old := telegramRequest
	defer func() { telegramRequest = old }()
	targets := []int64{}
	telegramRequest = func(_ context.Context, _ string, method string, input, out any) error {
		if method == "sendMessage" {
			targets = append(targets, input.(map[string]any)["chat_id"].(int64))
		}
		return nil
	}
	v, _ := s.store.telegramRead()
	s.telegramNotify(context.Background(), &v, "backup:jv-abcdef", "attempt-1:failed", "Backup failed.", false)
	v, _ = s.store.telegramRead()
	s.telegramNotify(context.Background(), &v, "backup:jv-abcdef", "attempt-1:failed", "Backup failed.", false)
	if !reflect.DeepEqual(targets, []int64{456}) {
		t.Fatal("notification duplicate or opt-in bypass", targets)
	}
}
func TestTelegramRebootResultRequiresNewBootExpectedDeploymentAndHealth(t *testing.T) {
	now := time.Now()
	p := telegramReboot{BootID: "old", Expected: "sha256:" + strings.Repeat("a", 64), RequestedAt: now}
	good := adminSystemUpdateStatus{Running: adminSystemUpdateDeployment{Digest: p.Expected}}
	if _, done := telegramRebootResult(p, "old", good, true, now); done {
		t.Fatal("agent restart reported update success")
	}
	if _, done := telegramRebootResult(p, "new", good, false, now); done {
		t.Fatal("unhealthy appliance reported update success")
	}
	text, done := telegramRebootResult(p, "new", good, true, now)
	if !done || !strings.Contains(text, "OS update confirmed") {
		t.Fatal("verified update not reported", text)
	}
	bad := good
	bad.Running.Digest = "sha256:" + strings.Repeat("b", 64)
	text, done = telegramRebootResult(p, "new", bad, true, now)
	if !done || strings.Contains(text, "OS update confirmed") {
		t.Fatal("wrong deployment reported success", text)
	}
	text, done = telegramRebootResult(p, "old", good, true, now.Add(21*time.Minute))
	if !done || !strings.Contains(text, "could not be verified") {
		t.Fatal("cancelled reboot not reported", text)
	}
	p.Expected = ""
	text, done = telegramRebootResult(p, "new", good, true, now)
	if !done || !strings.Contains(text, "Reboot confirmed") {
		t.Fatal("ordinary reboot not verified", text)
	}
}
func TestTelegramPendingRebootSurvivesStoreReload(t *testing.T) {
	s := telegramTestServer(t)
	v, _ := s.store.telegramRead()
	v.Reboot = &telegramReboot{BootID: "c388d25b-504f-4e70-9cdb-5f7f62757a62", Expected: "sha256:" + strings.Repeat("c", 64), RequestedAt: s.store.now(), State: "requesting"}
	if err := s.store.telegramSave(v, nil, ""); err != nil {
		t.Fatal(err)
	}
	reloaded, err := s.store.telegramRead()
	if err != nil || (reloaded.Reboot == nil || v.Reboot.BootID != reloaded.Reboot.BootID || v.Reboot.Expected != reloaded.Reboot.Expected || !v.Reboot.RequestedAt.Equal(reloaded.Reboot.RequestedAt) || v.Reboot.State != reloaded.Reboot.State) {
		t.Fatal("reboot evidence lost", err)
	}
}
func TestTelegramSanitizationDropsSecretsPathsAndRawLogs(t *testing.T) {
	raw := json.RawMessage(`{"state":"running","rcon_password":"secret-rcon","token":"secret-token","floodgate_key":"secret-key","logs":"raw-sensitive-log","path":"/etc/private","digest":"sha256:private-digest","image":"ghcr.io/private/image","config":{"members":["jv-abcdef"],"forwarding_secret":"private-forwarding"}}`)
	text := telegramResultText("network_status", 200, raw)
	for _, secret := range []string{"secret-rcon", "secret-token", "secret-key", "raw-sensitive-log", "/etc/private", "private-forwarding", "sha256:", "ghcr.io/private/image", `"config":`, "{"} {
		if strings.Contains(text, secret) {
			t.Fatal("private payload leaked", secret)
		}
	}
	if !strings.Contains(text, "jv-abcdef") {
		t.Fatal("sanitization removed authorized member")
	}
}
func TestTelegramFactoryRevocationIsDurableAndIndependentOfGateway(t *testing.T) {
	s := telegramTestServer(t)
	if err := s.telegramFactoryRevoke(); err != nil {
		t.Fatal(err)
	}
	v, err := s.store.telegramRead()
	if err != nil || v.Token != "" || v.Enabled || len(v.Users) > 0 {
		t.Fatal("factory revocation incomplete", err)
	}
	gateway, err := s.store.gatewaySnapshot()
	if err != nil || gateway.Enabled {
		t.Fatal("Telegram required enabled gateway", err)
	}
}

type telegramRoundTrip func(*http.Request) (*http.Response, error)

func (f telegramRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestTelegramTransportNeverReturnsTokenBearingErrors(t *testing.T) {
	old := telegramHTTP
	defer func() { telegramHTTP = old }()
	telegramHTTP = &http.Client{Transport: telegramRoundTrip(func(r *http.Request) (*http.Response, error) {
		return nil, errors.New("private failure " + r.URL.String())
	})}
	err := telegramRequest(context.Background(), telegramTestToken, "getMe", struct{}{}, nil)
	if err == nil || strings.Contains(err.Error(), telegramTestToken) || strings.Contains(err.Error(), "https://") {
		t.Fatal("URL-bearing error leaked", err)
	}
}
func TestTelegramTransportRejectsUnlistedMethodsAndConsumerConflict(t *testing.T) {
	old := telegramHTTP
	defer func() { telegramHTTP = old }()
	calls := 0
	telegramHTTP = &http.Client{Transport: telegramRoundTrip(func(r *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 409, Body: io.NopCloser(strings.NewReader(`{"ok":false,"error_code":409,"description":"private detail"}`)), Header: make(http.Header)}, nil
	})}
	if telegramRequest(context.Background(), telegramTestToken, "deleteWebhook", nil, nil) == nil || calls != 0 {
		t.Fatal("unlisted API method accepted")
	}
	err := telegramRequest(context.Background(), telegramTestToken, "getUpdates", map[string]any{"timeout": 25}, nil)
	var conflict *telegramAPIError
	if !errors.As(err, &conflict) || conflict.Code != 409 || strings.Contains(err.Error(), "private detail") {
		t.Fatal("consumer conflict not safely identified", err)
	}
}

func TestTelegramDisableCancelsPollingAndInvalidatesCallbacks(t *testing.T) {
	s := telegramTestServer(t)
	cancelled := false
	s.telegramPollCancel = func() { cancelled = true }
	rr := telegramAdminRequest(s, administratorSession(time.Now()), map[string]any{"action": "enabled", "enabled": false})
	v, _ := s.store.telegramRead()
	if rr.Code != 200 || !cancelled || v.Enabled || len(v.Callbacks) > 0 || v.Pair != nil {
		t.Fatal("poll cancellation incomplete")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if telegramPause(ctx, time.Hour) {
		t.Fatal("cancelled poller kept waiting")
	}
}
func TestTelegramManagementRequiresConfirmationBeforeDispatch(t *testing.T) {
	s := telegramTestServer(t)
	old := telegramRequest
	defer func() { telegramRequest = old }()
	telegramRequest = func(context.Context, string, string, any, any) error { return nil }
	v, _ := s.store.telegramRead()
	s.telegramMu.Lock()
	s.telegramDispatch(context.Background(), &v, 456, telegramCallback{Operation: "instance_restart", InstanceID: "jv-abcdef"})
	s.telegramMu.Unlock()
	v, _ = s.store.telegramRead()
	found := false
	for _, cb := range v.Callbacks {
		if cb.Operation == "instance_restart" {
			found = cb.Confirmed && cb.UserID == 456 && cb.InstanceID == "jv-abcdef"
		}
	}
	if !found {
		t.Fatal("management confirmation not bound to exact user and instance")
	}
	var attempts int
	if err := s.store.db.QueryRow(`SELECT count(*) FROM audit_events WHERE action='telegram_attempt_instance_restart'`).Scan(&attempts); err != nil || attempts != 0 {
		t.Fatal("operation attempted before confirmation", err)
	}
}
func TestTelegramRebootMenuBindsStagedDigestWarningAndBackupChoices(t *testing.T) {
	s := telegramTestServer(t)
	oldRequest, oldBootc := telegramRequest, runBootcStatus
	defer func() { telegramRequest = oldRequest; runBootcStatus = oldBootc }()
	telegramRequest = func(context.Context, string, string, any, any) error { return nil }
	digest := "sha256:" + strings.Repeat("a", 64)
	runBootcStatus = func(context.Context) ([]byte, error) {
		return []byte(`{"apiVersion":"org.containers.bootc/v1","status":{"booted":{"image":{"imageDigest":"sha256:` + strings.Repeat("b", 64) + `"}},"staged":{"image":{"imageDigest":"` + digest + `"},"downloadOnly":true}}}`), nil
	}
	v, _ := s.store.telegramRead()
	s.telegramMu.Lock()
	s.telegramRebootMenu(context.Background(), &v, 456)
	s.telegramMu.Unlock()
	v, _ = s.store.telegramRead()
	choices := map[string]bool{}
	for _, cb := range v.Callbacks {
		if cb.Operation != "update_reboot" {
			continue
		}
		var input adminSystemUpdateRebootRequest
		if json.Unmarshal(cb.Body, &input) != nil {
			t.Fatal("invalid reboot body")
		}
		if cb.Confirmed || input.ConfirmPlayers || !input.ActionConfirmed || input.StagedDigest != digest {
			t.Fatal("reboot bypassed confirmation or deployment binding")
		}
		key := time.Duration(input.WarningSeconds).String()
		if input.BackupMinecraft {
			key += "backup"
		}
		choices[key] = true
	}
	if len(choices) != 4 {
		t.Fatal("missing 10/60 warning or backup choices", choices)
	}
}

func TestTelegramAgentDeniesUnconfirmedOrForbiddenManagementCalls(t *testing.T) {
	s := telegramTestServer(t)
	v, _ := s.store.telegramRead()
	for _, tc := range []struct {
		id   int64
		call telegramCallback
	}{
		{123, telegramCallback{Operation: "instance_stop", InstanceID: "jv-abcdef", Confirmed: true}},
		{456, telegramCallback{Operation: "system_download"}},
		{456, telegramCallback{Operation: "shell", Confirmed: true}},
		{456, telegramCallback{Operation: "instance_restart", InstanceID: "../../other", Confirmed: true}},
	} {
		s.telegramMu.Lock()
		code, _ := s.telegramInvoke(context.Background(), &v, tc.id, tc.call)
		s.telegramMu.Unlock()
		if code != 403 {
			t.Fatal("Agent accepted denied capability", tc.call.Operation, code)
		}
	}
	var denied int
	if err := s.store.db.QueryRow(`SELECT count(*) FROM audit_events WHERE action='telegram_denied' AND success=0`).Scan(&denied); err != nil || denied != 4 {
		t.Fatal("denied management actions not audited", denied, err)
	}
}
func TestTelegramTransportRefusesRedirects(t *testing.T) {
	old := telegramHTTP
	defer func() { telegramHTTP = old }()
	client := *telegramHTTP
	calls := 0
	client.Transport = telegramRoundTrip(func(r *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 302, Header: http.Header{"Location": []string{"https://another-host.example/copy"}}, Body: io.NopCloser(strings.NewReader(`{"ok":false,"error_code":302}`)), Request: r}, nil
	})
	telegramHTTP = &client
	if err := telegramRequest(context.Background(), telegramTestToken, "getMe", struct{}{}, nil); err == nil || calls != 1 {
		t.Fatal("credential-bearing request followed redirect", calls, err)
	}
}

func TestTelegramCorruptStateCanDisconnectWithoutStrandingReset(t *testing.T) {
	for _, factory := range []bool{false, true} {
		s := telegramTestServer(t)
		if _, err := s.store.db.Exec(`UPDATE telegram_state SET record_json='{"schema":999,"token":"malformed-private-state"}' WHERE id=1`); err != nil {
			t.Fatal(err)
		}
		if factory {
			if err := s.telegramFactoryRevoke(); err != nil {
				t.Fatal("Telegram corruption stranded reset", err)
			}
		} else {
			rr := telegramAdminRequest(s, administratorSession(time.Now()), map[string]any{"action": "disconnect"})
			if rr.Code != 200 {
				t.Fatal("Telegram corruption stranded disconnection", rr.Code)
			}
		}
		v, err := s.store.telegramRead()
		if err != nil || v.Token != "" || v.Enabled || len(v.Users) > 0 {
			t.Fatal("corrupt Telegram state not safely revoked", err)
		}
	}
}
func TestTelegramSettingsRejectDuplicateKeysAndTrailingJSON(t *testing.T) {
	s := telegramTestServer(t)
	for _, body := range []string{`{"action":"pair","action":"disconnect"}`, `{"action":"disconnect"} {"action":"pair"}`, `{"action":"pair","username":"voxel"}`} {
		request := httptest.NewRequest("POST", "/v1/admin/telegram", strings.NewReader(body))
		request = request.WithContext(context.WithValue(request.Context(), gatewayPrincipalKey{}, administratorSession(time.Now())))
		rr := httptest.NewRecorder()
		s.telegramAdmin(rr, request)
		if rr.Code != 400 {
			t.Fatal("ambiguous settings accepted", rr.Code)
		}
	}
}

func TestTelegramEnableIsIndependentOfDisabledAIGateway(t *testing.T) {
	s := telegramTestServer(t)
	if err := s.store.gatewaySetEnabled(false, gatewayTestActor); err != nil {
		t.Fatal(err)
	}
	v, err := s.store.telegramRead()
	if err != nil || !v.Enabled || !telegramAllowed(&v, 456, "instance_start") {
		t.Fatal("gateway disable changed independent Telegram permission", err)
	}
}

func TestTelegramUncertainRebootRetainsPendingEvidence(t *testing.T) {
	s := telegramTestServer(t)
	oldAPI, oldBoot, oldHelper := telegramRequest, readSystemUpdateBootID, runAdminSystemUpdateRebootHelper
	defer func() {
		telegramRequest = oldAPI
		readSystemUpdateBootID = oldBoot
		runAdminSystemUpdateRebootHelper = oldHelper
	}()
	telegramRequest = func(context.Context, string, string, any, any) error { return nil }
	readSystemUpdateBootID = func() (string, error) { return "c388d25b-504f-4e70-9cdb-5f7f62757a62", nil }
	runAdminSystemUpdateRebootHelper = func(context.Context, ...string) ([]byte, int, error) { return []byte("unsupported output"), 0, nil }
	v, _ := s.store.telegramRead()
	cb := telegramCallback{Operation: "update_reboot", Confirmed: true, Body: telegramBody(adminSystemUpdateRebootRequest{ActionConfirmed: true, WarningSeconds: 60, StagedDigest: "sha256:" + strings.Repeat("d", 64)})}
	s.telegramMu.Lock()
	s.telegramDispatch(context.Background(), &v, 456, cb)
	s.telegramMu.Unlock()
	saved, err := s.store.telegramRead()
	if err != nil || saved.Reboot == nil || saved.Reboot.State != "unverified" || saved.OperationStatus != "reboot_result_unknown" {
		t.Fatal("ambiguous reboot lost evidence or reported a definite result", err)
	}
}
func TestTelegramUnavailableDeploymentEventuallyReportsUnverifiedResult(t *testing.T) {
	s := telegramTestServer(t)
	oldAPI, oldBootc := telegramRequest, runBootcStatus
	defer func() { telegramRequest = oldAPI; runBootcStatus = oldBootc }()
	telegramRequest = func(context.Context, string, string, any, any) error { return nil }
	runBootcStatus = func(context.Context) ([]byte, error) { return nil, errors.New("private deployment error") }
	v, _ := s.store.telegramRead()
	v.Reboot = &telegramReboot{BootID: "old", RequestedAt: s.store.now().Add(-21 * time.Minute), Expected: "sha256:" + strings.Repeat("d", 64), State: "unverified"}
	if err := s.store.telegramSave(v, nil, ""); err != nil {
		t.Fatal(err)
	}
	s.telegramMu.Lock()
	s.telegramReconcileReboot(context.Background(), &v, 456, "new")
	s.telegramMu.Unlock()
	saved, err := s.store.telegramRead()
	if err != nil || saved.Reboot != nil || !strings.Contains(saved.OperationStatus, "could not be verified") || strings.Contains(saved.OperationStatus, "private deployment error") {
		t.Fatal("unavailable deployment stranded Telegram or exposed diagnostics", err)
	}
}
