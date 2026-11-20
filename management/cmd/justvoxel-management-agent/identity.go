package main

import (
	"net/http"
	"strings"
	"time"

	"github.com/home-server-project/justvoxel/management/internal/systemauth"
)

type principalRole string

type authSource string

const (
	roleAdministrator principalRole = "administrator"
	roleOperator      principalRole = "operator"
	roleViewer        principalRole = "viewer"

	authSourceSystem    authSource = "system"
	authSourceSeparate  authSource = "separate"
	authSourceWebUI     authSource = "webui"
	authSourceLocalRoot authSource = "local-root"
	authSourceExternal  authSource = "external"
	authSourceTelegram  authSource = "telegram"
	authSourceDiscord   authSource = "discord"
)

type identityAuthResult struct {
	Username               string
	Role                   principalRole
	AuthSource             authSource
	WebUserID              int64
	PasswordChangeRequired bool
	AdminMode              authMode
	AccountCreated         time.Time
	PasswordChanged        time.Time
}

func (s *server) authenticateIdentity(username, password string) (identityAuthResult, error) {
	username = strings.TrimSpace(username)
	if username == systemAdminUsername {
		result, err := authenticateAdministrator(username, password)
		if err != nil {
			return identityAuthResult{}, err
		}
		source := authSourceSystem
		if result.Mode == authModeSeparate {
			source = authSourceSeparate
		}
		return identityAuthResult{
			Username:               systemAdminUsername,
			Role:                   roleAdministrator,
			AuthSource:             source,
			PasswordChangeRequired: result.PasswordChangeRequired,
			AdminMode:              result.Mode,
		}, nil
	}

	if s.store == nil {
		return identityAuthResult{}, systemauth.ErrInvalidCredentials
	}
	user, ok, err := s.store.checkWebUserPassword(username, password)
	if err != nil {
		return identityAuthResult{}, err
	}
	if !ok || !user.Enabled {
		return identityAuthResult{}, systemauth.ErrInvalidCredentials
	}

	role := principalRole(user.Role)
	if role != roleOperator && role != roleViewer {
		return identityAuthResult{}, systemauth.ErrInvalidCredentials
	}
	return identityAuthResult{
		Username:        user.Username,
		Role:            role,
		AuthSource:      authSourceWebUI,
		WebUserID:       user.ID,
		AccountCreated:  user.CreatedAt,
		PasswordChanged: user.PasswordChangedAt,
	}, nil
}

func (s *server) sessionStatus(w http.ResponseWriter, r *http.Request) {
	_, sess, ok := s.authorize(r, true)
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid session")
		return
	}
	profile, err := s.store.readDateTimeProfile()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "Date and time display settings are unavailable")
		return
	}
	var factor twoFactorStatus
	if sess.AuthSource == authSourceSystem || sess.AuthSource == authSourceSeparate || sess.AuthSource == authSourceWebUI {
		factor, err = s.accountTwoFactorStatus(sess.WebUserID)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "two-factor authentication unavailable")
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"time_format":        profile.TimeFormat,
		"date_format":        profile.DateFormat,
		"username":           sess.Username,
		"role":               sess.Role,
		"auth_source":        sess.AuthSource,
		"must_change":        sess.MustChange,
		"two_factor_enabled": factor.Enabled,
		"two_factor_source":  factor.Source,
	})
}
