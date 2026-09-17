package main

import (
	"net/http"
	"strconv"
	"time"
)

func (s *server) providerLogin(w http.ResponseWriter, r *http.Request) {
	if wait := s.loginDelay(); wait > 0 {
		w.Header().Set("Retry-After", strconv.Itoa(int(wait.Seconds())+1))
		writeError(w, http.StatusTooManyRequests, "too many failed login attempts")
		return
	}

	var request struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}

	result, err := s.authenticateIdentity(request.Username, request.Password)
	if err != nil {
		s.recordFailure()
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	s.clearFailures()
	token, err := randomToken(32)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "session creation failed")
		return
	}
	now := time.Now()
	s.mu.Lock()
	s.sessions[token] = session{
		Username:   result.Username,
		Role:       result.Role,
		AuthSource: result.AuthSource,
		WebUserID:  result.WebUserID,
		Created:    now,
		LastSeen:   now,
		MustChange: result.PasswordChangeRequired,
	}
	s.mu.Unlock()

	response := map[string]any{
		"session":        token,
		"must_change":    result.PasswordChangeRequired,
		"management_api": managementAPI,
		"username":       result.Username,
		"role":           result.Role,
		"auth_source":    result.AuthSource,
	}
	if result.Role == roleAdministrator {
		response["auth_mode"] = result.AdminMode
	}
	writeJSON(w, http.StatusOK, response)
}
