package main

import (
	"crypto/sha256"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const twoFactorChallengeLifetime = 5 * time.Minute

type twoFactorChallenge struct {
	Identity     identityAuthResult
	SecretDigest [32]byte
}

func rateLimited(w http.ResponseWriter, wait time.Duration) bool {
	if wait <= 0 {
		return false
	}
	w.Header().Set("Retry-After", strconv.Itoa(int(wait.Seconds())+1))
	writeError(w, http.StatusTooManyRequests, "too many failed login attempts")
	return true
}

func (s *server) providerLogin(w http.ResponseWriter, r *http.Request) {
	if rateLimited(w, s.loginDelay()) {
		return
	}
	var request struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}

	s.twoFactorMu.Lock()
	defer s.twoFactorMu.Unlock()
	if rateLimited(w, s.loginDelay()) {
		return
	}
	result, err := s.authenticateIdentity(request.Username, request.Password)
	if err != nil {
		s.recordFailure()
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	s.clearFailures()
	factor, err := s.readTwoFactor(result.WebUserID)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "two-factor authentication unavailable")
		return
	}
	token, err := randomToken(32)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "session creation failed")
		return
	}
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	if result.WebUserID != 0 && !s.currentChallengeIdentity(result) {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if factor != nil {
		for token, entry := range s.sessions {
			if entry.Challenge != nil && now.Sub(entry.Created) >= twoFactorChallengeLifetime {
				delete(s.sessions, token)
			}
		}
		s.sessions[token] = session{
			WebUserID: result.WebUserID, AuthSource: result.AuthSource, Created: now,
			Challenge: &twoFactorChallenge{Identity: result, SecretDigest: sha256.Sum256([]byte(factor.Secret))},
		}
		writeJSON(w, http.StatusOK, map[string]string{"challenge": token, "management_api": managementAPI})
		return
	}
	s.completeLogin(w, token, result, now)
}

// Called with the session lock held. A challenge token is never used as
// the bearer token for a completed login.
func (s *server) completeLogin(w http.ResponseWriter, token string, result identityAuthResult, now time.Time) {
	s.sessions[token] = session{
		Username: result.Username, Role: result.Role, AuthSource: result.AuthSource,
		WebUserID: result.WebUserID, Created: now, LastSeen: now, MustChange: result.PasswordChangeRequired,
	}
	response := map[string]any{
		"session": token, "must_change": result.PasswordChangeRequired,
		"management_api": managementAPI, "username": result.Username,
		"role": result.Role, "auth_source": result.AuthSource,
	}
	if result.Role == roleAdministrator {
		response["auth_mode"] = result.AdminMode
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *server) verifyLoginTwoFactor(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Code     string `json:"code"`
		Recovery bool   `json:"recovery"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	token, _ := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	s.twoFactorMu.Lock()
	defer s.twoFactorMu.Unlock()
	if rateLimited(w, s.secondFactorAttempts.delay()) {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entry := s.sessions[token]
	now := time.Now()
	if entry.Challenge == nil || now.Sub(entry.Created) >= twoFactorChallengeLifetime {
		if entry.Challenge != nil {
			delete(s.sessions, token)
		}
		writeError(w, http.StatusGone, "two-factor login expired")
		return
	}
	identity := entry.Challenge.Identity
	record, err := s.readTwoFactor(identity.WebUserID)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "two-factor authentication unavailable")
		return
	}
	if record == nil || sha256.Sum256([]byte(record.Secret)) != entry.Challenge.SecretDigest || !s.currentChallengeIdentity(identity) {
		delete(s.sessions, token)
		writeError(w, http.StatusGone, "two-factor login expired")
		return
	}
	next := *record
	valid := false
	if request.Recovery {
		if index := record.recoveryIndex(request.Code); index >= 0 {
			next.RecoveryHashes = append(append([]string{}, record.RecoveryHashes[:index]...), record.RecoveryHashes[index+1:]...)
			valid = true
		}
	} else if step := matchingTwoFactorStep(record.Secret, request.Code, now); step > record.LastStep {
		next.LastStep, valid = step, true
	}
	if !valid {
		s.secondFactorAttempts.recordFailure()
		writeError(w, http.StatusUnauthorized, "invalid verification code")
		return
	}
	sessionToken, err := randomToken(32)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "session creation failed")
		return
	}
	if err := s.changeTwoFactor(identity.WebUserID, record, &next); err != nil {
		writeError(w, http.StatusServiceUnavailable, "two-factor authentication unavailable")
		return
	}
	delete(s.sessions, token)
	s.completeLogin(w, sessionToken, identity, now)
}

func (s *server) currentChallengeIdentity(identity identityAuthResult) bool {
	if identity.WebUserID == 0 {
		mode, err := readAuthMode()
		return err == nil && mode == identity.AdminMode
	}
	user, err := s.store.webUserByID(identity.WebUserID)
	return err == nil && user.Enabled && principalRole(user.Role) == identity.Role &&
		user.Username == identity.Username && user.CreatedAt.Equal(identity.AccountCreated) &&
		user.PasswordChangedAt.Equal(identity.PasswordChanged)
}
