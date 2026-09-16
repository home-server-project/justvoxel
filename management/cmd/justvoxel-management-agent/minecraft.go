package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os/exec"
	"time"
)

type minecraftActionRequest struct {
	ConfirmPlayers bool `json:"confirm_players"`
}

type minecraftActionResponse struct {
	OK                   bool     `json:"ok"`
	Action               string   `json:"action,omitempty"`
	Message              string   `json:"message,omitempty"`
	ConfirmationRequired bool     `json:"confirmation_required,omitempty"`
	Reason               string   `json:"reason,omitempty"`
	Players              []string `json:"players,omitempty"`
	Online               int      `json:"online,omitempty"`
}

var runWebHelper = func(ctx context.Context, args ...string) ([]byte, int, error) {
	cmd := exec.CommandContext(ctx, statusHelper, args...)
	output, err := cmd.Output()
	if err == nil {
		return output, 0, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return output, exitErr.ExitCode(), err
	}
	return output, -1, err
}

func registerMinecraftRoutes(mux *http.ServeMux, s *server) {
	mux.HandleFunc("GET /v1/players", s.players)
	mux.HandleFunc("POST /v1/minecraft/start", s.minecraftStart)
	mux.HandleFunc("POST /v1/minecraft/stop", s.minecraftStop)
	mux.HandleFunc("POST /v1/minecraft/restart", s.minecraftRestart)
}

func (s *server) players(w http.ResponseWriter, r *http.Request) {
	if !s.authorizeAdminRequest(w, r) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	output, exitCode, err := runWebHelper(ctx, "players")
	if err != nil || exitCode != 0 {
		writeError(w, http.StatusServiceUnavailable, "player status collection failed")
		return
	}
	writeHelperJSON(w, http.StatusOK, output, "player status collector returned invalid data")
}

func (s *server) minecraftStart(w http.ResponseWriter, r *http.Request) {
	s.minecraftAction(w, r, "start")
}

func (s *server) minecraftStop(w http.ResponseWriter, r *http.Request) {
	s.minecraftAction(w, r, "stop")
}

func (s *server) minecraftRestart(w http.ResponseWriter, r *http.Request) {
	s.minecraftAction(w, r, "restart")
}

func (s *server) minecraftAction(w http.ResponseWriter, r *http.Request, action string) {
	if !s.authorizeAdminRequest(w, r) {
		return
	}
	var request minecraftActionRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	args := []string{action}
	if request.ConfirmPlayers {
		args = append(args, "--confirm-players")
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	output, exitCode, err := runWebHelper(ctx, args...)
	if !json.Valid(output) {
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "Minecraft control helper failed")
		} else {
			writeError(w, http.StatusInternalServerError, "Minecraft control helper returned invalid data")
		}
		return
	}
	var result minecraftActionResponse
	if err := json.Unmarshal(output, &result); err != nil {
		writeError(w, http.StatusInternalServerError, "Minecraft control helper returned invalid data")
		return
	}
	if err == nil && exitCode == 0 {
		writeHelperJSON(w, http.StatusOK, output, "Minecraft control helper returned invalid data")
		return
	}
	if exitCode == 10 && result.ConfirmationRequired {
		writeHelperJSON(w, http.StatusOK, output, "Minecraft control helper returned invalid data")
		return
	}
	status := http.StatusInternalServerError
	switch exitCode {
	case 2:
		status = http.StatusConflict
	case 3:
		status = http.StatusServiceUnavailable
	case 4:
		status = http.StatusInternalServerError
	}
	writeHelperJSON(w, status, output, "Minecraft control helper returned invalid data")
}

func (s *server) authorizeAdminRequest(w http.ResponseWriter, r *http.Request) bool {
	_, sess, ok := s.authorize(r, false)
	if ok {
		return true
	}
	if sess.MustChange {
		writeError(w, http.StatusForbidden, "password change required")
	} else {
		writeError(w, http.StatusUnauthorized, "invalid session")
	}
	return false
}

func writeHelperJSON(w http.ResponseWriter, status int, output []byte, invalidMessage string) {
	if !json.Valid(output) {
		writeError(w, http.StatusInternalServerError, invalidMessage)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_, _ = w.Write(output)
}
