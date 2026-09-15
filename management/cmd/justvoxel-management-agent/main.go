package main

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/crypto/argon2"
)

const (
	managementAPI = "v1"
	stateDir      = "/var/lib/justvoxel/webui"
	authPath      = stateDir + "/auth.json"
	metadataPath  = "/usr/lib/justvoxel/webui-release.json"
	statusHelper  = "/usr/libexec/justvoxel/mjust/web-status-json"
	passwordChars = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789"
)

const (
	argonMemory      = 64 * 1024
	argonIterations  = 3
	argonParallelism = 2
	argonKeyLength   = 32
)

type credential struct {
	Username    string `json:"username"`
	Salt        string `json:"salt"`
	Hash        string `json:"hash"`
	MemoryKiB   uint32 `json:"memory_kib"`
	Iterations  uint32 `json:"iterations"`
	Parallelism uint8  `json:"parallelism"`
	KeyLength   uint32 `json:"key_length"`
	MustChange  bool   `json:"must_change"`
}

type releaseMetadata struct {
	Version       string `json:"version"`
	SourceCommit  string `json:"source_commit"`
	ManagementAPI string `json:"management_api"`
	BuildDate     string `json:"build_date"`
}

type session struct {
	Created    time.Time
	LastSeen   time.Time
	MustChange bool
}

type server struct {
	webUID uint32

	mu       sync.Mutex
	sessions map[string]session
	failures []time.Time
	lockTill time.Time
}

type peerUIDKey struct{}

func main() {
	log.SetFlags(log.LstdFlags | log.LUTC)
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		fatal("usage: justvoxel-management-agent <serve|bootstrap|password-reset|version>")
	}

	switch args[0] {
	case "bootstrap":
		password, created, err := bootstrap()
		if err != nil {
			fatal("bootstrap: %v", err)
		}
		if created {
			fmt.Println(password)
		}
	case "password-reset":
		password, err := resetPassword()
		if err != nil {
			fatal("password reset: %v", err)
		}
		fmt.Println(password)
	case "serve":
		socket := "/run/justvoxel/management.sock"
		if len(args) > 2 || (len(args) == 2 && args[1] == "") {
			fatal("usage: justvoxel-management-agent serve [socket]")
		}
		if len(args) == 2 {
			socket = args[1]
		}
		if err := serve(socket); err != nil {
			fatal("serve: %v", err)
		}
	case "version":
		meta, _ := readReleaseMetadata()
		fmt.Printf("JustVoxel Management API %s\nWebUI %s\n", managementAPI, valueOr(meta.Version, "unknown"))
	default:
		fatal("unknown command %q", args[0])
	}
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "ERROR: "+format+"\n", args...)
	os.Exit(1)
}

func bootstrap() (string, bool, error) {
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		return "", false, err
	}
	created := false
	password := ""
	if _, err := os.Stat(authPath); errors.Is(err, os.ErrNotExist) {
		password, err = randomPassword(14)
		if err != nil {
			return "", false, err
		}
		cred, err := credentialForPassword(password, true)
		if err != nil {
			return "", false, err
		}
		if err := writeCredential(cred); err != nil {
			return "", false, err
		}
		created = true
	} else if err != nil {
		return "", false, err
	} else if _, err := readCredential(); err != nil {
		return "", false, fmt.Errorf("existing credential is invalid: %w", err)
	}
	return password, created, nil
}

func resetPassword() (string, error) {
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		return "", err
	}
	password, err := randomPassword(14)
	if err != nil {
		return "", err
	}
	cred, err := credentialForPassword(password, true)
	if err != nil {
		return "", err
	}
	if err := writeCredential(cred); err != nil {
		return "", err
	}
	return password, nil
}

func serve(socket string) error {
	if _, _, err := bootstrap(); err != nil {
		return err
	}
	webAccount, err := user.Lookup("justvoxel-web")
	if err != nil {
		return fmt.Errorf("lookup justvoxel-web: %w", err)
	}
	uid64, err := strconv.ParseUint(webAccount.Uid, 10, 32)
	if err != nil {
		return fmt.Errorf("parse justvoxel-web uid: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(socket), 0o750); err != nil {
		return err
	}
	_ = os.Remove(socket)
	listener, err := net.Listen("unix", socket)
	if err != nil {
		return err
	}
	defer listener.Close()
	if err := os.Chmod(socket, 0o660); err != nil {
		return err
	}

	s := &server{webUID: uint32(uid64), sessions: make(map[string]session)}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/auth/login", s.login)
	mux.HandleFunc("POST /v1/auth/logout", s.logout)
	mux.HandleFunc("POST /v1/auth/password", s.changePassword)
	mux.HandleFunc("GET /v1/info", s.info)
	mux.HandleFunc("GET /v1/status", s.status)

	httpServer := &http.Server{
		Handler:           s.requirePeer(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		ConnContext: func(ctx context.Context, c net.Conn) context.Context {
			uid, err := peerUID(c)
			if err != nil {
				uid = ^uint32(0)
			}
			return context.WithValue(ctx, peerUIDKey{}, uid)
		},
	}
	log.Printf("JustVoxel Management API %s listening on %s", managementAPI, socket)
	return httpServer.Serve(listener)
}

func (s *server) requirePeer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uid, ok := r.Context().Value(peerUIDKey{}).(uint32)
		if !ok || (uid != 0 && uid != s.webUID) {
			writeError(w, http.StatusForbidden, "peer not permitted")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func peerUID(conn net.Conn) (uint32, error) {
	unixConn, ok := conn.(*net.UnixConn)
	if !ok {
		return 0, errors.New("not a Unix connection")
	}
	raw, err := unixConn.SyscallConn()
	if err != nil {
		return 0, err
	}
	var cred *syscall.Ucred
	var sockErr error
	if err := raw.Control(func(fd uintptr) {
		cred, sockErr = syscall.GetsockoptUcred(int(fd), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
	}); err != nil {
		return 0, err
	}
	if sockErr != nil {
		return 0, sockErr
	}
	return cred.Uid, nil
}

func (s *server) login(w http.ResponseWriter, r *http.Request) {
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
	cred, err := readCredential()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "credential unavailable")
		return
	}
	if request.Username != cred.Username || !verifyPassword(cred, request.Password) {
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
	s.sessions[token] = session{Created: now, LastSeen: now, MustChange: cred.MustChange}
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"session": token, "must_change": cred.MustChange, "management_api": managementAPI})
}

func (s *server) logout(w http.ResponseWriter, r *http.Request) {
	token, _, ok := s.authorize(r, true)
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid session")
		return
	}
	s.mu.Lock()
	delete(s.sessions, token)
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *server) changePassword(w http.ResponseWriter, r *http.Request) {
	_, _, ok := s.authorize(r, true)
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid session")
		return
	}
	var request struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	if len(request.NewPassword) < 12 || len(request.NewPassword) > 128 {
		writeError(w, http.StatusBadRequest, "new password must be 12 to 128 characters")
		return
	}
	cred, err := readCredential()
	if err != nil || !verifyPassword(cred, request.CurrentPassword) {
		writeError(w, http.StatusUnauthorized, "current password is incorrect")
		return
	}
	newCred, err := credentialForPassword(request.NewPassword, false)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "password update failed")
		return
	}
	if err := writeCredential(newCred); err != nil {
		writeError(w, http.StatusInternalServerError, "password update failed")
		return
	}
	s.mu.Lock()
	s.sessions = make(map[string]session)
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *server) info(w http.ResponseWriter, _ *http.Request) {
	meta, _ := readReleaseMetadata()
	variant, _ := os.ReadFile("/usr/lib/justvoxel/variant")
	writeJSON(w, http.StatusOK, map[string]string{
		"management_api": managementAPI,
		"webui_version":  valueOr(meta.Version, "unknown"),
		"variant":        strings.TrimSpace(string(variant)),
	})
}

func (s *server) status(w http.ResponseWriter, r *http.Request) {
	_, sess, ok := s.authorize(r, false)
	if !ok {
		if sess.MustChange {
			writeError(w, http.StatusForbidden, "password change required")
		} else {
			writeError(w, http.StatusUnauthorized, "invalid session")
		}
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, statusHelper)
	output, err := cmd.Output()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "status collection failed")
		return
	}
	if !json.Valid(output) {
		writeError(w, http.StatusInternalServerError, "status collector returned invalid data")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(output)
}

func (s *server) authorize(r *http.Request, allowMustChange bool) (string, session, bool) {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return "", session{}, false
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	if token == "" {
		return "", session{}, false
	}
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[token]
	if !ok {
		return token, session{}, false
	}
	if now.Sub(sess.Created) > 12*time.Hour || now.Sub(sess.LastSeen) > 30*time.Minute {
		delete(s.sessions, token)
		return token, session{}, false
	}
	if sess.MustChange && !allowMustChange {
		return token, sess, false
	}
	sess.LastSeen = now
	s.sessions[token] = sess
	return token, sess, true
}

func (s *server) loginDelay() time.Duration {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	if now.Before(s.lockTill) {
		return time.Until(s.lockTill)
	}
	cutoff := now.Add(-5 * time.Minute)
	kept := s.failures[:0]
	for _, failure := range s.failures {
		if failure.After(cutoff) {
			kept = append(kept, failure)
		}
	}
	s.failures = kept
	return 0
}

func (s *server) recordFailure() {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failures = append(s.failures, now)
	cutoff := now.Add(-5 * time.Minute)
	count := 0
	for _, failure := range s.failures {
		if failure.After(cutoff) {
			count++
		}
	}
	if count >= 5 {
		s.lockTill = now.Add(30 * time.Second)
	}
}

func (s *server) clearFailures() {
	s.mu.Lock()
	s.failures = nil
	s.lockTill = time.Time{}
	s.mu.Unlock()
}

func credentialForPassword(password string, mustChange bool) (credential, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return credential{}, err
	}
	hash := argon2.IDKey([]byte(password), salt, argonIterations, argonMemory, argonParallelism, argonKeyLength)
	return credential{
		Username:    "admin",
		Salt:        base64.RawStdEncoding.EncodeToString(salt),
		Hash:        base64.RawStdEncoding.EncodeToString(hash),
		MemoryKiB:   argonMemory,
		Iterations:  argonIterations,
		Parallelism: argonParallelism,
		KeyLength:   argonKeyLength,
		MustChange:  mustChange,
	}, nil
}

func verifyPassword(cred credential, password string) bool {
	if cred.Username != "admin" || cred.MemoryKiB == 0 || cred.Iterations == 0 || cred.Parallelism == 0 || cred.KeyLength == 0 {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(cred.Salt)
	if err != nil {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(cred.Hash)
	if err != nil || len(expected) != int(cred.KeyLength) {
		return false
	}
	actual := argon2.IDKey([]byte(password), salt, cred.Iterations, cred.MemoryKiB, cred.Parallelism, cred.KeyLength)
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

func readCredential() (credential, error) {
	data, err := os.ReadFile(authPath)
	if err != nil {
		return credential{}, err
	}
	var cred credential
	if err := json.Unmarshal(data, &cred); err != nil {
		return credential{}, err
	}
	return cred, nil
}

func writeCredential(cred credential) error {
	data, err := json.MarshalIndent(cred, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return atomicWrite(authPath, data, 0o600)
}

func atomicWrite(path string, data []byte, mode os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func randomPassword(length int) (string, error) {
	out := make([]byte, length)
	limit := byte(256 - (256 % len(passwordChars)))
	for i := range out {
		for {
			var b [1]byte
			if _, err := rand.Read(b[:]); err != nil {
				return "", err
			}
			if b[0] < limit {
				out[i] = passwordChars[int(b[0])%len(passwordChars)]
				break
			}
		}
	}
	return string(out), nil
}

func randomToken(bytes int) (string, error) {
	buf := make([]byte, bytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func readReleaseMetadata() (releaseMetadata, error) {
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return releaseMetadata{}, err
	}
	var meta releaseMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return releaseMetadata{}, err
	}
	return meta, nil
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16*1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func valueOr(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
