package server

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/home-server-project/justvoxel-webui/internal/api"
)

const (
	twoFactorLoginPath = "/login/two-factor"
	twoFactorCookie    = "jv_two_factor"
)

type twoFactorLoginAPI interface {
	VerifyTwoFactor(ctx context.Context, challenge, code string, recovery bool) (api.LoginResponse, error)
}

type twoFactorAccountAPI interface {
	TwoFactor(ctx context.Context, session, action string, request api.TwoFactorRequest) (api.TwoFactorResponse, error)
}

type twoFactorLoginData struct {
	Title        string
	Error        string
	CSRF         string
	OAuthRequest string
	Recovery     bool
	SwitchURL    string
	LoginURL     string
}

func (a *App) registerTwoFactorRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET "+twoFactorLoginPath, a.twoFactorLogin)
	mux.HandleFunc("POST "+twoFactorLoginPath, a.twoFactorLogin)
	for _, action := range []string{"begin", "confirm", "disable"} {
		mux.HandleFunc("POST /api/system/workspace/users/two-factor/"+action, a.twoFactorAccount(action))
	}
}

func (a *App) setTwoFactorCookie(w http.ResponseWriter, value string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name: twoFactorCookie, Value: value, Path: twoFactorLoginPath, Secure: a.config.SecureCookies,
		HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: maxAge,
	})
}

func (a *App) startTwoFactorChallenge(w http.ResponseWriter, r *http.Request, challenge string) {
	csrf, err := randomToken(32)
	if err != nil {
		http.Error(w, ui(w, "ui.955ec9c5d5c3"), http.StatusInternalServerError)
		return
	}
	a.clearCookie(w, sessionCookie, true)
	a.clearCookie(w, mustChangeCookie, true)
	a.setCookie(w, csrfCookie, csrf, false)
	a.setTwoFactorCookie(w, challenge, 300)
	http.Redirect(w, r, oauthContinuePath(twoFactorLoginPath, oauthContinuation(r)), http.StatusSeeOther)
}

func (a *App) twoFactorLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if r.Method == http.MethodPost {
		r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
		if r.ParseForm() != nil || r.URL.RawQuery != "" {
			http.Error(w, ui(w, "ui.8daf3a877523"), http.StatusBadRequest)
			return
		}
	}
	oauth := oauthContinuation(r)
	loginURL := oauthContinuePath("/login", oauth)
	challenge, err := r.Cookie(twoFactorCookie)
	if err != nil || challenge.Value == "" {
		http.Redirect(w, r, loginURL, http.StatusSeeOther)
		return
	}
	recovery := r.FormValue("recovery") == "true"
	switchPath := twoFactorLoginPath + "?recovery=true"
	if recovery {
		switchPath = twoFactorLoginPath
	}
	data := twoFactorLoginData{
		Title: ui(w, "two_factor.title"), CSRF: csrfFromRequest(r), OAuthRequest: oauth,
		Recovery: recovery, SwitchURL: oauthContinuePath(switchPath, oauth), LoginURL: loginURL,
	}
	status := http.StatusOK
	if r.Method == http.MethodPost {
		validCSRF := a.validCSRF(r)
		if oauth != "" {
			validCSRF = a.oauthValidCSRF(r)
		}
		if !validCSRF {
			http.Error(w, ui(w, "ui.74bccc44f472"), http.StatusForbidden)
			return
		}
		client, ok := a.api.(twoFactorLoginAPI)
		if !ok {
			http.Error(w, ui(w, "two_factor.unavailable"), http.StatusServiceUnavailable)
			return
		}
		result, err := client.VerifyTwoFactor(r.Context(), challenge.Value, r.PostForm.Get("code"), recovery)
		if err == nil {
			a.finishLogin(w, r, result)
			return
		}
		var responseErr *api.ResponseError
		switch {
		case errors.Is(err, api.ErrUnauthorized):
			status, data.Error = http.StatusUnauthorized, ui(w, "two_factor.invalid_code")
		case errors.As(err, &responseErr) && responseErr.StatusCode == http.StatusGone:
			a.setTwoFactorCookie(w, "", -1)
			http.Redirect(w, r, oauthContinuePath("/login?message=two-factor-expired", oauth), http.StatusSeeOther)
			return
		case responseErr != nil && responseErr.StatusCode == http.StatusTooManyRequests:
			status, data.Error = http.StatusTooManyRequests, ui(w, "two_factor.rate_limited")
		default:
			status, data.Error = http.StatusBadGateway, ui(w, "two_factor.unavailable")
		}
	}
	w.WriteHeader(status)
	if err := a.executeTemplate(w, "two-factor-login.html", data); err != nil {
		http.Error(w, ui(w, "ui.4a9d9119358b"), http.StatusInternalServerError)
	}
}

func (a *App) twoFactorAccount(action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
		if r.ParseForm() != nil || r.URL.RawQuery != "" {
			writeSystemWorkspaceError(w, http.StatusBadRequest, ui(w, "ui.8daf3a877523"))
			return
		}
		if !a.validCSRF(r) {
			writeSystemWorkspaceError(w, http.StatusForbidden, ui(w, "ui.74bccc44f472"))
			return
		}
		session, identity, ok := a.systemWorkspaceIdentity(w, r, "administrator", "operator", "viewer")
		if !ok {
			return
		}
		client, ok := a.api.(twoFactorAccountAPI)
		if !ok {
			writeSystemWorkspaceError(w, http.StatusServiceUnavailable, ui(w, "two_factor.unavailable"))
			return
		}
		request := api.TwoFactorRequest{
			Enrollment: r.PostForm.Get("enrollment"), Password: r.PostForm.Get("current_password"),
			Code: r.PostForm.Get("code"), Recovery: r.PostForm.Get("recovery") == "true",
		}
		if rawID := r.PostForm.Get("user_id"); rawID != "" {
			id, err := strconv.ParseInt(rawID, 10, 64)
			if err != nil || id <= 0 {
				writeSystemWorkspaceError(w, http.StatusBadRequest, ui(w, "two_factor.unavailable"))
				return
			}
			request.UserID = &id
		}
		result, err := client.TwoFactor(r.Context(), session, action, request)
		if err == nil {
			writeSystemWorkspaceJSON(w, http.StatusOK, result)
			return
		}
		var responseErr *api.ResponseError
		switch {
		case errors.Is(err, api.ErrUnauthorized):
			message := "two_factor.invalid_code"
			if action == "disable" {
				message = "two_factor.invalid_credentials"
				if isAdministratorLevel(identity.Role) && request.UserID != nil {
					message = "ui.d383cd72b8ec"
				}
			}
			writeSystemWorkspaceError(w, http.StatusBadRequest, ui(w, message))
		case errors.Is(err, api.ErrPasswordChangeRequired):
			writeSystemWorkspaceError(w, http.StatusForbidden, ui(w, "two_factor.forbidden"))
		case errors.As(err, &responseErr) && responseErr.StatusCode == http.StatusTooManyRequests:
			writeSystemWorkspaceError(w, http.StatusTooManyRequests, ui(w, "two_factor.rate_limited"))
		case action == "confirm" && responseErr != nil && responseErr.StatusCode == http.StatusConflict:
			writeSystemWorkspaceError(w, http.StatusConflict, ui(w, "two_factor.restart_setup"))
		default:
			writeSystemWorkspaceError(w, http.StatusBadGateway, ui(w, "two_factor.unavailable"))
		}
	}
}
