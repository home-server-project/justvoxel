package main

import (
	"bytes"
	"encoding/base32"
	"encoding/json"
	"image/png"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/home-server-project/justvoxel/management/internal/systemauth"
	"github.com/pquerna/otp/totp"
)

func TestTwoFactorTOTPCompatibility(t *testing.T) {
	secret := base32.StdEncoding.EncodeToString([]byte("12345678901234567890"))
	at := time.Unix(59, 0)
	// RFC 6238 Appendix B, SHA-1 at Unix time 59, reduced to six digits.
	for _, delta := range []int64{-30, 0, 30} {
		if got := matchingTwoFactorStep(secret, "287082", at.Add(time.Duration(delta)*time.Second)); got != 1 {
			t.Errorf("RFC code with clock offset %d returned interval %d", delta, got)
		}
	}
	for _, delta := range []int64{60, 90} {
		if matchingTwoFactorStep(secret, "287082", at.Add(time.Duration(delta)*time.Second)) != 0 {
			t.Errorf("accepted RFC code with clock offset %d", delta)
		}
	}
	if matchingTwoFactorStep(secret, "000000", at) != 0 {
		t.Error("accepted incorrect code")
	}
	key, err := newTwoFactorKey("Test-Operator")
	if err != nil {
		t.Fatal(err)
	}
	uri, err := url.Parse(key.URL())
	if err != nil {
		t.Fatal("invalid enrollment URI")
	}
	if uri.Scheme != "otpauth" || uri.Host != "totp" || uri.Path != "/JustVoxel:Test-Operator" {
		t.Error("incorrect enrollment URI type, issuer or account label")
	}
	for parameter, expected := range map[string]string{
		"issuer": "JustVoxel", "algorithm": "SHA1", "digits": "6", "period": "30", "secret": key.Secret(),
	} {
		if uri.Query().Get(parameter) != expected {
			t.Errorf("incorrect enrollment URI parameter %s", parameter)
		}
	}
	escapedKey, err := newTwoFactorKey("A/B: C?#")
	if err != nil {
		t.Fatal("failed to generate key for username with reserved characters")
	}
	escaped, err := url.Parse(escapedKey.URL())
	if err != nil || escaped.EscapedPath() != "/JustVoxel:A%2FB%3A%20C%3F%23" || escapedKey.AccountName() != "A/B: C?#" {
		t.Fatal("username was not encoded as a single account label")
	}

}

type twoFactorFixture struct {
	server *server
	mux    *http.ServeMux
}

type twoFactorTestResponse struct {
	Session       string   `json:"session"`
	Challenge     string   `json:"challenge"`
	SetupKey      string   `json:"setup_key"`
	Enrollment    string   `json:"enrollment"`
	QRPNG         []byte   `json:"qr_png"`
	RecoveryCodes []string `json:"recovery_codes"`
}

func newTwoFactorFixture(t *testing.T) *twoFactorFixture {
	t.Helper()
	store, _ := openTestWebUIStore(t)
	s := &server{
		store: store, adminFactorPath: filepath.Join(t.TempDir(), "local-auth.json"),
		sessions: map[string]session{"admin": administratorSession(time.Now())},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/auth/login", s.providerLogin)
	mux.HandleFunc("GET /v1/session", s.sessionStatus)
	registerTwoFactorRoutes(mux, s)
	registerAdminUserRoutes(mux, s)
	return &twoFactorFixture{server: s, mux: mux}
}

func (f *twoFactorFixture) request(t *testing.T, path, token string, body any, status int) *httptest.ResponseRecorder {
	t.Helper()
	method := http.MethodGet
	var input []byte
	if body != nil {
		method = http.MethodPost
		var err error
		input, err = json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(input))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	response := httptest.NewRecorder()
	f.mux.ServeHTTP(response, req)
	if response.Code != status {
		t.Fatalf("%s %s returned %d, want %d", method, path, response.Code, status)
	}
	return response
}

func decodeTwoFactorResponse(t *testing.T, response *httptest.ResponseRecorder) twoFactorTestResponse {
	t.Helper()
	var decoded twoFactorTestResponse
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatal("invalid authentication response")
	}
	return decoded
}

func (f *twoFactorFixture) login(t *testing.T, username, password string) twoFactorTestResponse {
	t.Helper()
	return decodeTwoFactorResponse(t, f.request(t, "/v1/auth/login", "", map[string]string{"username": username, "password": password}, http.StatusOK))
}

func freshTOTP(t *testing.T, key string) string {
	t.Helper()
	code, err := totp.GenerateCode(key, time.Now())
	if err != nil {
		t.Fatal("generating test TOTP failed")
	}
	return code
}

func (f *twoFactorFixture) enroll(t *testing.T, actor string, target *int64, userID int64) (twoFactorTestResponse, []string) {
	t.Helper()
	begin := decodeTwoFactorResponse(t, f.request(t, "/v1/auth/two-factor/begin", actor, twoFactorRequest{UserID: target}, http.StatusOK))
	if begin.SetupKey == "" || begin.Enrollment == "" || len(begin.RecoveryCodes) != 0 {
		t.Fatal("unexpected initialization response")
	}
	img, err := png.DecodeConfig(bytes.NewReader(begin.QRPNG))
	if err != nil || img.Width != 216 || img.Height != 216 {
		t.Fatal("missing local QR PNG")
	}
	if factor, err := f.server.readTwoFactor(userID); err != nil || factor != nil {
		t.Fatal("unverified enrollment became persistent")
	}
	change := twoFactorRequest{UserID: target, Enrollment: begin.Enrollment, Code: freshTOTP(t, begin.SetupKey)}
	confirmed := decodeTwoFactorResponse(t, f.request(t, "/v1/auth/two-factor/confirm", actor, change, http.StatusOK))
	if len(confirmed.RecoveryCodes) != 10 || confirmed.SetupKey != "" {
		t.Fatal("unexpected confirmation response")
	}
	f.request(t, "/v1/auth/two-factor/confirm", actor, change, http.StatusConflict)
	return begin, confirmed.RecoveryCodes
}

func TestTwoFactorLoginAndOneTimeRecovery(t *testing.T) {
	f := newTwoFactorFixture(t)
	user, err := f.server.store.createWebUser("Example", webUserRoleOperator, "operator-password")
	if err != nil {
		t.Fatal(err)
	}
	login := f.login(t, user.Username, "operator-password")
	if login.Session == "" || login.Challenge != "" {
		t.Fatal("password-only sign-in did not return a session")
	}
	begin, codes := f.enroll(t, login.Session, nil, user.ID)
	var hashes string
	err = f.server.store.db.QueryRow(`SELECT recovery_hashes FROM web_user_two_factor WHERE user_id = ?`, user.ID).Scan(&hashes)
	if err != nil {
		t.Fatal(err)
	}
	seen := make(map[string]bool)
	for _, code := range codes {
		if seen[code] || strings.Contains(hashes, code) || !strings.Contains(hashes, recoveryHash(code)) {
			t.Fatal("recovery code missing, duplicated or stored as plaintext")
		}
		seen[code] = true
	}
	status := f.request(t, "/v1/session", login.Session, nil, http.StatusOK)
	if strings.Contains(status.Body.String(), begin.SetupKey) || strings.Contains(status.Body.String(), recoveryHash(codes[0])) ||
		!strings.Contains(status.Body.String(), `"two_factor_enabled":true`) || !strings.Contains(status.Body.String(), `"two_factor_source":"self"`) {
		t.Fatal("identity response did not expose only two-factor status")
	}
	challenge := f.login(t, user.Username, "operator-password")
	if challenge.Session != "" || challenge.Challenge == "" {
		t.Fatal("first factor issued a regular session")
	}
	f.request(t, "/v1/session", challenge.Challenge, nil, http.StatusUnauthorized)
	f.request(t, "/v1/auth/two-factor/begin", challenge.Challenge, twoFactorRequest{}, http.StatusUnauthorized)
	f.request(t, "/v1/auth/login/two-factor", login.Session, twoFactorRequest{Code: freshTOTP(t, begin.SetupKey)}, http.StatusGone)
	invalid := f.request(t, "/v1/auth/login/two-factor", challenge.Challenge, twoFactorRequest{Code: "invalid"}, http.StatusUnauthorized)
	if decodeTwoFactorResponse(t, invalid).Session != "" {
		t.Fatal("invalid second factor issued a session")
	}
	result := decodeTwoFactorResponse(t, f.request(t, "/v1/auth/login/two-factor", challenge.Challenge,
		twoFactorRequest{Code: freshTOTP(t, begin.SetupKey)}, http.StatusOK))
	if result.Session == "" || result.Session == challenge.Challenge {
		t.Fatal("second factor failed to create a distinct session")
	}
	f.request(t, "/v1/session", result.Session, nil, http.StatusOK)
	f.request(t, "/v1/auth/login/two-factor", challenge.Challenge, twoFactorRequest{Code: codes[0], Recovery: true}, http.StatusGone)
	challenge = f.login(t, user.Username, "operator-password")
	result = decodeTwoFactorResponse(t, f.request(t, "/v1/auth/login/two-factor", challenge.Challenge,
		twoFactorRequest{Code: codes[0], Recovery: true}, http.StatusOK))
	f.request(t, "/v1/session", result.Session, nil, http.StatusOK)
	record, err := f.server.readTwoFactor(user.ID)
	if err != nil || record == nil || len(record.RecoveryHashes) != 9 || record.recoveryIndex(codes[0]) >= 0 {
		t.Fatal("successful recovery did not consume exactly one code")
	}
	challenge = f.login(t, user.Username, "operator-password")
	f.request(t, "/v1/auth/login/two-factor", challenge.Challenge, twoFactorRequest{Code: codes[0], Recovery: true}, http.StatusUnauthorized)
}

func TestTwoFactorOwnershipForOperatorAndViewer(t *testing.T) {
	withAuthTestDependencies(t)
	readAuthMode = func() (authMode, error) { return authModeSeparate, nil }
	verifySeparateAdmin = func(username, password string) bool { return username == "voxel" && password == "admin-password" }
	systemPasswordPolicy = func() (systemauth.PasswordPolicy, error) { return systemauth.PasswordPolicy{MinLength: 8}, nil }
	for _, role := range []webUserRole{webUserRoleOperator, webUserRoleViewer} {
		t.Run(string(role), func(t *testing.T) {
			f := newTwoFactorFixture(t)
			user, err := f.server.store.createWebUser("Target", role, "user-password")
			if err != nil {
				t.Fatal(err)
			}
			other, err := f.server.store.createWebUser("Other", webUserRoleViewer, "other-password")
			if err != nil {
				t.Fatal(err)
			}
			userSession := f.login(t, user.Username, "user-password").Session
			for _, action := range []string{"begin", "confirm", "disable"} {
				f.request(t, "/v1/auth/two-factor/"+action, userSession, twoFactorRequest{UserID: &other.ID}, http.StatusForbidden)
			}
			_, codes := f.enroll(t, "admin", &user.ID, user.ID)
			assertSource := func(expected string) {
				t.Helper()
				record, err := f.server.readTwoFactor(user.ID)
				if err != nil || record == nil || record.Source != expected {
					t.Fatalf("expected enrollment source %q", expected)
				}
			}
			assertSource(twoFactorAdministrator)
			users := f.request(t, "/v1/admin/users", "admin", nil, http.StatusOK)
			if !bytes.Contains(users.Body.Bytes(), []byte(`"two_factor_source":"administrator"`)) || bytes.Contains(users.Body.Bytes(), []byte(codes[0])) {
				t.Fatal("Users view failed to report enrollment source safely")
			}
			challenge := f.login(t, user.Username, "user-password").Challenge
			userSession = decodeTwoFactorResponse(t, f.request(t, "/v1/auth/login/two-factor", challenge,
				twoFactorRequest{Code: codes[0], Recovery: true}, http.StatusOK)).Session
			f.request(t, "/v1/auth/two-factor/disable", userSession,
				twoFactorRequest{Password: "user-password", Code: codes[1], Recovery: true}, http.StatusForbidden)
			f.request(t, "/v1/auth/two-factor/disable", "admin", twoFactorRequest{UserID: &user.ID, Password: "wrong"}, http.StatusUnauthorized)
			f.request(t, "/v1/auth/two-factor/disable", "admin", twoFactorRequest{UserID: &user.ID, Password: "admin-password"}, http.StatusOK)
			_, codes = f.enroll(t, userSession, nil, user.ID)
			assertSource(twoFactorSelf)
			f.request(t, "/v1/auth/two-factor/disable", userSession,
				twoFactorRequest{Password: "user-password", Code: codes[0], Recovery: true}, http.StatusOK)
			f.enroll(t, userSession, nil, user.ID)
			f.request(t, "/v1/auth/two-factor/disable", "admin", twoFactorRequest{UserID: &user.ID, Password: "admin-password"}, http.StatusOK)
			if factor, err := f.server.readTwoFactor(user.ID); err != nil || factor != nil {
				t.Fatal("disable left a persistent factor")
			}
		})
	}
}

func TestVoxelTwoFactorUsesCurrentPasswordMode(t *testing.T) {
	withAuthTestDependencies(t)
	for _, mode := range []authMode{authModeSystem, authModeSeparate} {
		t.Run(string(mode), func(t *testing.T) {
			readAuthMode = func() (authMode, error) { return mode, nil }
			systemAuthenticate = func(username, password string) (systemauth.AuthResult, error) {
				if username == "voxel" && password == "system-password" {
					return systemauth.AuthResult{}, nil
				}
				return systemauth.AuthResult{}, systemauth.ErrInvalidCredentials
			}
			f := newTwoFactorFixture(t)
			password, wrongPassword := "system-password", "separate-password"
			if mode == authModeSeparate {
				password, wrongPassword = wrongPassword, password
				account, err := localAccountForPassword("voxel", "administrator", password)
				if err != nil {
					t.Fatal(err)
				}
				if err := writeLocalAuthStoreAt(f.server.adminFactorPath, localAuthStore{Version: localAuthVersion, Accounts: []localAccount{account}}); err != nil {
					t.Fatal(err)
				}
				verifySeparateAdmin = func(username, password string) bool {
					store, err := readLocalAuthStoreAt(f.server.adminFactorPath)
					return err == nil && len(store.Accounts) == 1 && username == "voxel" && verifyLocalAccount(store.Accounts[0], password)
				}
			}
			login := f.login(t, "voxel", password)
			begin, codes := f.enroll(t, login.Session, nil, 0)
			factor, err := f.server.readTwoFactor(0)
			if err != nil || factor == nil || factor.Source != twoFactorSelf {
				t.Fatal("voxel's enrollment was not self-managed")
			}
			file, err := os.ReadFile(f.server.adminFactorPath)
			if err != nil || bytes.Contains(file, []byte(codes[0])) {
				t.Fatal("administrator recovery codes persisted as plaintext")
			}
			info, err := os.Stat(f.server.adminFactorPath)
			if err != nil || info.Mode().Perm() != 0o600 {
				t.Fatal("administrator factor file is not private")
			}
			challenge := f.login(t, "voxel", password)
			if challenge.Session != "" || challenge.Challenge == "" {
				t.Fatal("voxel received a session after password alone")
			}
			code := freshTOTP(t, begin.SetupKey)
			login = decodeTwoFactorResponse(t, f.request(t, "/v1/auth/login/two-factor", challenge.Challenge,
				twoFactorRequest{Code: code}, http.StatusOK))
			f.request(t, "/v1/session", login.Session, nil, http.StatusOK)
			f.request(t, "/v1/auth/two-factor/disable", login.Session, twoFactorRequest{Password: wrongPassword, Code: code}, http.StatusUnauthorized)
			f.request(t, "/v1/auth/two-factor/disable", login.Session, twoFactorRequest{Password: password, Code: code}, http.StatusOK)
			if f.login(t, "voxel", password).Session == "" {
				t.Fatal("voxel could not use password-only sign-in after disable")
			}
		})
	}
}

func TestTwoFactorThrottlingSurvivesPasswordSuccess(t *testing.T) {
	f := newTwoFactorFixture(t)
	user, err := f.server.store.createWebUser("Operator", webUserRoleOperator, "operator-password")
	if err != nil {
		t.Fatal(err)
	}
	login := f.login(t, user.Username, "operator-password")
	begin, _ := f.enroll(t, login.Session, nil, user.ID)
	for range 5 {
		challenge := f.login(t, user.Username, "operator-password").Challenge
		f.request(t, "/v1/auth/login/two-factor", challenge, twoFactorRequest{Code: "invalid"}, http.StatusUnauthorized)
	}
	challenge := f.login(t, user.Username, "operator-password").Challenge
	f.request(t, "/v1/auth/login/two-factor", challenge, twoFactorRequest{Code: freshTOTP(t, begin.SetupKey)}, http.StatusTooManyRequests)
}
