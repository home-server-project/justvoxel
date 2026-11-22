package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"encoding/hex"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

const (
	twoFactorIssuer             = "JustVoxel"
	twoFactorPeriod             = 30
	twoFactorEnrollmentLifetime = 10 * time.Minute
)

var errTwoFactorChanged = errors.New("two-factor authentication state changed")

type twoFactorEnrollment struct {
	ID             string
	UserID         int64
	AccountCreated time.Time
	Secret         string
	Expires        time.Time
}

type twoFactorRequest struct {
	UserID     *int64 `json:"user_id,omitempty"`
	Enrollment string `json:"enrollment,omitempty"`
	Password   string `json:"password,omitempty"`
	Code       string `json:"code,omitempty"`
	Recovery   bool   `json:"recovery,omitempty"`
}

func registerTwoFactorRoutes(mux *http.ServeMux, s *server) {
	mux.HandleFunc("POST /v1/auth/login/two-factor", s.verifyLoginTwoFactor)
	mux.HandleFunc("POST /v1/auth/two-factor/begin", s.beginTwoFactor)
	mux.HandleFunc("POST /v1/auth/two-factor/confirm", s.confirmTwoFactor)
	mux.HandleFunc("POST /v1/auth/two-factor/disable", s.disableTwoFactor)
}

func newTwoFactorKey(username string) (*otp.Key, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer: twoFactorIssuer, AccountName: username, Period: twoFactorPeriod,
		Digits: otp.DigitsSix, Algorithm: otp.AlgorithmSHA1,
	})
	if err != nil {
		return nil, err
	}
	uri, err := url.Parse(key.URL())
	if err != nil {
		return nil, err
	}
	// Account names may contain path separators. Keep the entire label in
	// one segment, with a literal colon delimiting issuer and account name.
	uri.RawPath = "/" + twoFactorIssuer + ":" + strings.ReplaceAll(url.PathEscape(username), ":", "%3A")
	return otp.NewKeyFromURL(uri.String())
}

func twoFactorQR(key *otp.Key) ([]byte, error) {
	const size = 216
	qr, err := key.Image(176, 176)
	if err != nil {
		return nil, err
	}
	canvas := image.NewGray(image.Rect(0, 0, size, size))
	draw.Draw(canvas, canvas.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.Draw(canvas, image.Rect(20, 20, size-20, size-20), qr, qr.Bounds().Min, draw.Src)
	var encoded bytes.Buffer
	err = png.Encode(&encoded, canvas)
	return encoded.Bytes(), err
}

// Return the greatest matching interval so a successful login can rule out
// reuse of the same interval, including codes accepted for clock drift.
func matchingTwoFactorStep(secret, code string, now time.Time) int64 {
	code = strings.TrimSpace(code)
	if len(code) != 6 {
		return 0
	}
	var step int64
	for _, offset := range []time.Duration{-30 * time.Second, 0, 30 * time.Second} {
		at := now.Add(offset)
		expected, err := totp.GenerateCode(secret, at)
		if err == nil && subtle.ConstantTimeCompare([]byte(expected), []byte(code)) == 1 {
			step = at.Unix() / twoFactorPeriod
		}
	}
	return step
}

func recoveryHash(code string) string {
	normalized := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(code), "-", ""))
	digest := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(digest[:])
}

func generateRecoveryCodes() ([]string, []string, error) {
	codes := make([]string, 10)
	hashes := make([]string, len(codes))
	for i := range codes {
		var random [10]byte
		if _, err := rand.Read(random[:]); err != nil {
			return nil, nil, err
		}
		code := base32.StdEncoding.EncodeToString(random[:])
		codes[i] = strings.Join([]string{code[:4], code[4:8], code[8:12], code[12:]}, "-")
		hashes[i] = recoveryHash(code)
	}
	return codes, hashes, nil
}

func (record *twoFactorRecord) recoveryIndex(code string) int {
	hash := recoveryHash(code)
	index := -1
	for i, stored := range record.RecoveryHashes {
		if subtle.ConstantTimeCompare([]byte(hash), []byte(stored)) == 1 {
			index = i
		}
	}
	return index
}

func (s *server) twoFactorTarget(w http.ResponseWriter, actor session, requested *int64) (webUser, bool) {
	if requested == nil {
		if actor.WebUserID == 0 {
			return webUser{Username: actor.Username}, true
		}
		requested = &actor.WebUserID
	}
	if *requested <= 0 || (!actor.Role.isAdministratorLevel() && *requested != actor.WebUserID) {
		writeError(w, http.StatusForbidden, "operation not permitted for this account")
		return webUser{}, false
	}
	user, err := s.store.webUserByID(*requested)
	if err != nil {
		writeAdminUserStoreError(w, err)
		return webUser{}, false
	}
	return user, true
}

func (s *server) twoFactorAction(w http.ResponseWriter, r *http.Request) (string, session, webUser, twoFactorRequest, bool) {
	token, actor, ok := s.authorize(r, false)
	if !ok || token == "" {
		writeError(w, http.StatusUnauthorized, "invalid session")
		return "", session{}, webUser{}, twoFactorRequest{}, false
	}
	var request twoFactorRequest
	if !decodeJSON(w, r, &request) {
		return "", session{}, webUser{}, twoFactorRequest{}, false
	}
	target, ok := s.twoFactorTarget(w, actor, request.UserID)
	return token, actor, target, request, ok
}

func (s *server) beginTwoFactor(w http.ResponseWriter, r *http.Request) {
	s.twoFactorMu.Lock()
	defer s.twoFactorMu.Unlock()
	token, _, target, _, ok := s.twoFactorAction(w, r)
	if !ok {
		return
	}
	record, err := s.readTwoFactor(target.ID)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "two-factor authentication unavailable")
		return
	}
	if record != nil {
		writeError(w, http.StatusConflict, "two-factor authentication is already enabled")
		return
	}
	key, err := newTwoFactorKey(target.Username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "two-factor setup failed")
		return
	}
	qr, err := twoFactorQR(key)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "two-factor setup failed")
		return
	}
	id, err := randomToken(32)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "two-factor setup failed")
		return
	}
	s.mu.Lock()
	current, ok := s.sessions[token]
	if ok {
		current.Enrollment = &twoFactorEnrollment{
			ID: id, UserID: target.ID, AccountCreated: target.CreatedAt,
			Secret: key.Secret(), Expires: time.Now().Add(twoFactorEnrollmentLifetime),
		}
		s.sessions[token] = current
	}
	s.mu.Unlock()
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid session")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"enrollment": id, "setup_key": key.Secret(), "qr_png": qr})
}

func (s *server) confirmTwoFactor(w http.ResponseWriter, r *http.Request) {
	s.twoFactorMu.Lock()
	defer s.twoFactorMu.Unlock()
	token, actor, target, request, ok := s.twoFactorAction(w, r)
	if !ok || rateLimited(w, s.secondFactorAttempts.delay()) {
		return
	}
	pending := actor.Enrollment
	now := time.Now()
	if pending == nil || !now.Before(pending.Expires) || pending.UserID != target.ID ||
		!pending.AccountCreated.Equal(target.CreatedAt) ||
		subtle.ConstantTimeCompare([]byte(pending.ID), []byte(request.Enrollment)) != 1 {
		writeError(w, http.StatusConflict, "two-factor setup expired")
		return
	}
	if matchingTwoFactorStep(pending.Secret, request.Code, now) == 0 {
		s.secondFactorAttempts.recordFailure()
		writeError(w, http.StatusUnauthorized, "invalid verification code")
		return
	}
	codes, hashes, err := generateRecoveryCodes()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "two-factor setup failed")
		return
	}
	source := twoFactorSelf
	if actor.WebUserID != target.ID {
		source = twoFactorAdministrator
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.sessions[token]
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid session")
		return
	}
	record := &twoFactorRecord{
		Secret: pending.Secret, Source: source, RecoveryHashes: hashes, accountCreated: target.CreatedAt,
	}
	if err := s.changeTwoFactor(target.ID, nil, record); err != nil {
		writeError(w, http.StatusConflict, "two-factor setup failed")
		return
	}
	current.Enrollment = nil
	s.sessions[token] = current
	for otherToken, other := range s.sessions {
		if otherToken != token && other.WebUserID == target.ID {
			delete(s.sessions, otherToken)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"recovery_codes": codes})
}

func (s *server) disableTwoFactor(w http.ResponseWriter, r *http.Request) {
	s.twoFactorMu.Lock()
	defer s.twoFactorMu.Unlock()
	token, actor, target, request, ok := s.twoFactorAction(w, r)
	if !ok {
		return
	}
	self := actor.WebUserID == target.ID
	record, err := s.readTwoFactor(target.ID)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "two-factor authentication unavailable")
		return
	}
	if record == nil {
		writeError(w, http.StatusConflict, "two-factor authentication is already disabled")
		return
	}
	if self && record.Source != twoFactorSelf {
		writeError(w, http.StatusForbidden, "two-factor authentication is managed by the administrator")
		return
	}
	if rateLimited(w, s.secondFactorAttempts.delay()) {
		return
	}
	if !s.verifySessionPassword(actor, request.Password) {
		s.secondFactorAttempts.recordFailure()
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if self {
		valid := false
		if request.Recovery {
			valid = record.recoveryIndex(request.Code) >= 0
		} else {
			valid = matchingTwoFactorStep(record.Secret, request.Code, time.Now()) != 0
		}
		if !valid {
			s.secondFactorAttempts.recordFailure()
			writeError(w, http.StatusUnauthorized, "invalid verification code")
			return
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.sessions[token]; !ok {
		writeError(w, http.StatusUnauthorized, "invalid session")
		return
	}
	if err := s.changeTwoFactor(target.ID, record, nil); err != nil {
		writeError(w, http.StatusServiceUnavailable, "two-factor authentication could not be disabled")
		return
	}
	for otherToken, other := range s.sessions {
		if other.WebUserID == target.ID && other.Challenge != nil {
			delete(s.sessions, otherToken)
		}
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
