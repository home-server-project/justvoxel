package server

import "net/http"

func (a *App) providerLoginPage(w http.ResponseWriter, r *http.Request) {
	a.clearFactoryResetCompleteCookie(w)
	if c, err := r.Cookie(sessionCookie); err == nil && c.Value != "" {
		if _, err := a.api.Status(r.Context(), c.Value); err == nil {
			if mustChange(r) {
				http.Redirect(w, r, oauthContinuePath("/password", oauthContinuation(r)), http.StatusSeeOther)
			} else {
				http.Redirect(w, r, oauthResumePath(oauthContinuation(r)), http.StatusSeeOther)
			}
			return
		}
	}
	message := ""
	switch r.URL.Query().Get("message") {
	case "password-changed":
		message = ui(w, "ui.49cd9108c32c")
	case "auth-mode-changed":
		message = ui(w, "ui.e707330bcc43")
	}
	a.render(w, "login.html", pageData{
		Title:         ui(w, "ui.bfd402b2f6f3"),
		Version:       a.config.Version,
		ManagementAPI: a.config.ManagementAPI,
		Message:       message,
		OAuthRequest:  oauthContinuation(r),
	})
}
