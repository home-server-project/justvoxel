package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/home-server-project/justvoxel-webui/internal/api"
)

type twoFactorWebAPI struct {
	*fakeAPI
	verified  int
	mutations int
}

func (f *twoFactorWebAPI) Login(_ context.Context, username, password string) (api.LoginResponse, error) {
	if username == "voxel" && password == "test-password" {
		return api.LoginResponse{Challenge: "challenge-token", ManagementAPI: "v1"}, nil
	}
	return api.LoginResponse{}, api.ErrUnauthorized
}

func (f *twoFactorWebAPI) VerifyTwoFactor(_ context.Context, token, code string, recovery bool) (api.LoginResponse, error) {
	f.verified++
	if token == "challenge-token" && ((!recovery && code == "123456") || (recovery && code == "TEST-ONLY")) {
		return api.LoginResponse{Session: "session-token", ManagementAPI: "v1"}, nil
	}
	return api.LoginResponse{}, api.ErrUnauthorized
}

func (f *twoFactorWebAPI) TwoFactor(_ context.Context, _, _ string, _ api.TwoFactorRequest) (api.TwoFactorResponse, error) {
	f.mutations++
	return api.TwoFactorResponse{OK: true}, nil
}

func twoFactorForm(path string, values url.Values, cookies ...*http.Cookie) *http.Request {
	request := httptest.NewRequest(http.MethodPost, "https://voxel.example"+path, strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Origin", "https://voxel.example")
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}
	return request
}

func responseCookie(response *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == name && cookie.MaxAge >= 0 {
			return cookie
		}
	}
	return nil
}

func TestWebUITwoFactorChallengeUsesOnlyPreAuthenticationCookie(t *testing.T) {
	for _, recovery := range []bool{false, true} {
		name, code, label := "authenticator", "123456", "Authenticator code"
		if recovery {
			name, code, label = "recovery", "TEST-ONLY", "Recovery code"
		}
		t.Run(name, func(t *testing.T) {
			client := &twoFactorWebAPI{fakeAPI: &fakeAPI{}}
			app, err := New(client, Config{ManagementAPI: "v1", ExternalScheme: "https", SecureCookies: true})
			if err != nil {
				t.Fatal(err)
			}
			handler := app.Handler()
			first := httptest.NewRecorder()
			handler.ServeHTTP(first, twoFactorForm("/login", url.Values{"username": {"voxel"}, "password": {"test-password"}}))
			if first.Code != http.StatusSeeOther || first.Header().Get("Location") != twoFactorLoginPath || responseCookie(first, sessionCookie) != nil {
				t.Fatal("password step did not redirect exclusively to the verification challenge")
			}
			challenge, csrf := responseCookie(first, twoFactorCookie), responseCookie(first, csrfCookie)
			if challenge == nil || csrf == nil || !challenge.HttpOnly || !challenge.Secure ||
				challenge.SameSite != http.SameSiteStrictMode || challenge.MaxAge != 300 || challenge.Path != twoFactorLoginPath {
				t.Fatal("challenge cookie is not short-lived and scoped to verification")
			}

			path := twoFactorLoginPath
			if recovery {
				path += "?recovery=true"
			}
			page := httptest.NewRequest(http.MethodGet, "https://voxel.example"+path, nil)
			page.AddCookie(challenge)
			page.AddCookie(csrf)
			shown := httptest.NewRecorder()
			handler.ServeHTTP(shown, page)
			if shown.Code != http.StatusOK || !strings.Contains(shown.Body.String(), label) ||
				strings.Contains(shown.Body.String(), challenge.Value) || shown.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("challenge page omitted the input or exposed the challenge")
			}

			verify := url.Values{"csrf": {csrf.Value}, "code": {code}}
			if recovery {
				verify.Set("recovery", "true")
			}
			blocked := httptest.NewRecorder()
			handler.ServeHTTP(blocked, twoFactorForm(twoFactorLoginPath, verify, csrf))
			if blocked.Code != http.StatusSeeOther || client.verified != 0 {
				t.Fatal("verification was reachable without a challenge cookie")
			}
			blocked = httptest.NewRecorder()
			handler.ServeHTTP(blocked, twoFactorForm(twoFactorLoginPath, verify, challenge))
			if blocked.Code != http.StatusForbidden || client.verified != 0 {
				t.Fatal("verification was reachable without the CSRF cookie")
			}

			wrong := httptest.NewRecorder()
			verify.Set("code", "invalid")
			handler.ServeHTTP(wrong, twoFactorForm(twoFactorLoginPath, verify, csrf, challenge))
			if wrong.Code != http.StatusUnauthorized || responseCookie(wrong, sessionCookie) != nil {
				t.Fatal("failed verification issued a session")
			}

			verified := httptest.NewRecorder()
			verify.Set("code", code)
			handler.ServeHTTP(verified, twoFactorForm(twoFactorLoginPath, verify, csrf, challenge))
			if verified.Code != http.StatusSeeOther || responseCookie(verified, sessionCookie) == nil ||
				responseCookie(verified, twoFactorCookie) != nil {
				t.Fatal("successful verification failed to establish the normal session")
			}
		})
	}
}

func TestWebUITwoFactorAccountMutationsRequireCSRFAndAuthenticatedSession(t *testing.T) {
	client := &twoFactorWebAPI{fakeAPI: &fakeAPI{}}
	app, err := New(client, Config{ExternalScheme: "https"})
	if err != nil {
		t.Fatal(err)
	}
	handler := app.Handler()
	csrf := &http.Cookie{Name: csrfCookie, Value: "csrf-token"}
	session := &http.Cookie{Name: sessionCookie, Value: "session-token"}
	challengeAsSession := &http.Cookie{Name: sessionCookie, Value: "challenge-token"}
	for i, action := range []string{"begin", "confirm", "disable"} {
		path := "/api/system/workspace/users/two-factor/" + action
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, twoFactorForm(path, url.Values{"csrf": {"wrong"}}, session, csrf))
		if response.Code != http.StatusForbidden || client.mutations != i {
			t.Fatalf("%s accepted invalid CSRF", action)
		}
		response = httptest.NewRecorder()
		handler.ServeHTTP(response, twoFactorForm(path, url.Values{"csrf": {csrf.Value}}, challengeAsSession, csrf))
		if response.Code != http.StatusUnauthorized || client.mutations != i {
			t.Fatalf("%s accepted a challenge as a session", action)
		}
		response = httptest.NewRecorder()
		handler.ServeHTTP(response, twoFactorForm(path, url.Values{"csrf": {csrf.Value}}, session, csrf))
		if response.Code != http.StatusOK || client.mutations != i+1 {
			t.Fatalf("%s failed to forward authenticated mutation", action)
		}
	}
}

func TestTwoFactorManageDialogAndLocaleContract(t *testing.T) {
	source, err := assets.ReadFile("static/app.js")
	if err != nil {
		t.Fatal(err)
	}
	_, users, _ := strings.Cut(string(source), "  const renderUsers =")
	menu, dialogs, ok := strings.Cut(users, "  const openTwoFactorDialog =")
	dialog, account, ended := strings.Cut(dialogs, "  const renderOwnAccount =")
	if !ok || !ended {
		t.Fatal("two-factor Manage and self-service dialogs are missing")
	}
	first := strings.Index(menu, "choices.appendChild(twoFactor)")
	next := strings.Index(menu, `addAction(jvT("ui.a43a8d8cfd29")`)
	if first < 0 || next <= first || !strings.Contains(menu, `user.two_factor_enabled ? "two_factor.disable" : "two_factor.enable"`) {
		t.Fatal("two-factor action is not first or does not track account status")
	}
	for _, expected := range []string{
		`"system-action-dialog system-user-dialog"`, `"system-action-dialog-title"`, `"action-row"`,
		`disabling ? "danger" : ""`, `"two-factor-qr"`, `216`, `"two_factor.copy_key"`, `"two_factor.copy_all"`,
		`"two_factor.download_codes"`, `"justvoxel-recovery-codes.txt"`, `URL.revokeObjectURL`,
	} {
		if !strings.Contains(dialog, expected) {
			t.Errorf("two-factor dialog missing %q", expected)
		}
	}
	if strings.Contains(dialog, "destructive-confirm") || strings.Contains(dialog, "localStorage") {
		t.Error("two-factor dialog uses a destructive slider or persistent browser storage")
	}
	css := correctionSource(t, "static/app.css")
	if !strings.Contains(css, `.two-factor-fields{text-align:right}`) ||
		!strings.Contains(dialog, `fields.append(copy, status)`) || !strings.Contains(dialog, `fields.appendChild(toggle)`) {
		t.Error("two-factor setup key and code toggle buttons must use the right-aligned field container")
	}
	if !strings.Contains(account, `user.two_factor_source === "administrator"`) || !strings.Contains(account, "if (!managed)") {
		t.Error("administrator-managed account exposes self-disable")
	}
	for key, value := range catalogs["en"] {
		if !strings.HasPrefix(key, "two_factor.") {
			continue
		}
		for _, locale := range supportedLocales {
			if locale != "en" && (catalogs[locale][key] == "" || catalogs[locale][key] == value) {
				t.Errorf("%s lacks a translation for %s", locale, key)
			}
		}
	}
}
