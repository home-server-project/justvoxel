package main

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/home-server-project/justvoxel/management/internal/systemauth"
)

func withAuthTestDependencies(t *testing.T) {
	t.Helper()
	oldReadMode := readAuthMode
	oldWriteMode := writeAuthMode
	oldWriteSeparate := writeSeparateAdmin
	oldVerifySeparate := verifySeparateAdmin
	oldSystemAuth := systemAuthenticate
	oldSystemChange := systemChangePassword
	oldSystemValidate := systemValidatePass
	oldSystemPolicy := systemPasswordPolicy
	t.Cleanup(func() {
		readAuthMode = oldReadMode
		writeAuthMode = oldWriteMode
		writeSeparateAdmin = oldWriteSeparate
		verifySeparateAdmin = oldVerifySeparate
		systemAuthenticate = oldSystemAuth
		systemChangePassword = oldSystemChange
		systemValidatePass = oldSystemValidate
		systemPasswordPolicy = oldSystemPolicy
	})
}

func TestSystemLoginUsesVoxelPAMCredential(t *testing.T) {
	withAuthTestDependencies(t)
	readAuthMode = func() (authMode, error) { return authModeSystem, nil }
	systemAuthenticate = func(username, password string) (systemauth.AuthResult, error) {
		if username == "voxel" && password == "system-secret" {
			return systemauth.AuthResult{}, nil
		}
		return systemauth.AuthResult{}, systemauth.ErrInvalidCredentials
	}

	s := &server{sessions: make(map[string]session)}
	good := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewBufferString(`{"username":"voxel","password":"system-secret"}`))
	goodRR := httptest.NewRecorder()
	s.providerLogin(goodRR, good)
	if goodRR.Code != http.StatusOK {
		t.Fatalf("valid system credential returned %d: %s", goodRR.Code, goodRR.Body.String())
	}

	for _, body := range []string{
		`{"username":"admin","password":"system-secret"}`,
		`{"username":"voxel","password":"wrong"}`,
	} {
		req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewBufferString(body))
		rr := httptest.NewRecorder()
		s.providerLogin(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("invalid credential returned %d: %s", rr.Code, rr.Body.String())
		}
	}
}

func TestSystemLoginPropagatesPasswordChangeRequired(t *testing.T) {
	withAuthTestDependencies(t)
	readAuthMode = func() (authMode, error) { return authModeSystem, nil }
	systemAuthenticate = func(username, password string) (systemauth.AuthResult, error) {
		return systemauth.AuthResult{PasswordChangeRequired: true}, nil
	}

	s := &server{sessions: make(map[string]session)}
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewBufferString(`{"username":"voxel","password":"bootstrap"}`))
	rr := httptest.NewRecorder()
	s.providerLogin(rr, req)
	if rr.Code != http.StatusOK || !bytes.Contains(rr.Body.Bytes(), []byte(`"must_change":true`)) {
		t.Fatalf("expired-password state not returned: %d %s", rr.Code, rr.Body.String())
	}
}

func TestSystemPasswordChangeInvalidatesSessions(t *testing.T) {
	withAuthTestDependencies(t)
	readAuthMode = func() (authMode, error) { return authModeSystem, nil }
	systemValidatePass = func(username, oldPassword, newPassword string) error { return nil }
	systemChangePassword = func(username, oldPassword, newPassword string) error {
		if username != systemAdminUsername || oldPassword != "old-secret" || newPassword != "new-secret" {
			t.Fatalf("unexpected password change arguments")
		}
		return nil
	}

	now := time.Now()
	s := &server{sessions: map[string]session{
		"active": {Created: now, LastSeen: now, MustChange: true},
		"other":  {Created: now, LastSeen: now},
	}}
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/password", bytes.NewBufferString(`{"current_password":"old-secret","new_password":"new-secret"}`))
	req.Header.Set("Authorization", "Bearer active")
	rr := httptest.NewRecorder()
	s.providerChangePassword(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("password change returned %d: %s", rr.Code, rr.Body.String())
	}
	if len(s.sessions) != 0 {
		t.Fatal("sessions were not invalidated after password change")
	}
}

func TestSystemToSeparateModeSwitch(t *testing.T) {
	withAuthTestDependencies(t)
	mode := authModeSystem
	var separatePassword string
	readAuthMode = func() (authMode, error) { return mode, nil }
	writeAuthMode = func(next authMode) error { mode = next; return nil }
	writeSeparateAdmin = func(password string) error { separatePassword = password; return nil }
	systemAuthenticate = func(username, password string) (systemauth.AuthResult, error) {
		if username == "voxel" && password == "system-secret" {
			return systemauth.AuthResult{}, nil
		}
		return systemauth.AuthResult{}, systemauth.ErrInvalidCredentials
	}
	systemValidatePass = func(username, oldPassword, newPassword string) error { return nil }

	s := &server{sessions: map[string]session{
		"active": {Created: time.Now(), LastSeen: time.Now()},
		"other":  {Created: time.Now(), LastSeen: time.Now()},
	}}
	body := `{"mode":"separate","system_password":"system-secret","new_web_password":"web-secret","confirm_web_password":"web-secret"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/mode", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer active")
	rr := httptest.NewRecorder()
	s.changeAuthMode(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("mode switch returned %d: %s", rr.Code, rr.Body.String())
	}
	if mode != authModeSeparate || separatePassword != "web-secret" {
		t.Fatalf("separate provider not created correctly: mode=%q password=%q", mode, separatePassword)
	}
	if len(s.sessions) != 0 {
		t.Fatal("sessions were not invalidated after mode change")
	}
}

func TestSeparateToSystemRequiresSystemCredential(t *testing.T) {
	withAuthTestDependencies(t)
	mode := authModeSeparate
	readAuthMode = func() (authMode, error) { return mode, nil }
	writeAuthMode = func(next authMode) error { mode = next; return nil }
	systemAuthenticate = func(username, password string) (systemauth.AuthResult, error) {
		if password != "system-secret" {
			return systemauth.AuthResult{}, systemauth.ErrInvalidCredentials
		}
		return systemauth.AuthResult{}, nil
	}

	newServer := func() *server {
		return &server{sessions: map[string]session{"active": {Created: time.Now(), LastSeen: time.Now()}}}
	}

	s := newServer()
	bad := httptest.NewRequest(http.MethodPost, "/v1/auth/mode", bytes.NewBufferString(`{"mode":"system","system_password":"wrong"}`))
	bad.Header.Set("Authorization", "Bearer active")
	badRR := httptest.NewRecorder()
	s.changeAuthMode(badRR, bad)
	if badRR.Code != http.StatusUnauthorized || mode != authModeSeparate {
		t.Fatalf("bad system credential changed mode: %d mode=%q", badRR.Code, mode)
	}

	s = newServer()
	good := httptest.NewRequest(http.MethodPost, "/v1/auth/mode", bytes.NewBufferString(`{"mode":"system","system_password":"system-secret"}`))
	good.Header.Set("Authorization", "Bearer active")
	goodRR := httptest.NewRecorder()
	s.changeAuthMode(goodRR, good)
	if goodRR.Code != http.StatusOK || mode != authModeSystem {
		t.Fatalf("valid system credential did not restore system mode: %d mode=%q body=%s", goodRR.Code, mode, goodRR.Body.String())
	}
	if len(s.sessions) != 0 {
		t.Fatal("sessions were not invalidated after switching back to system mode")
	}
}

func TestSeparateModeIgnoresSystemCredentialForWebLogin(t *testing.T) {
	withAuthTestDependencies(t)
	readAuthMode = func() (authMode, error) { return authModeSeparate, nil }
	verifySeparateAdmin = func(username, password string) bool {
		return username == "voxel" && password == "web-secret"
	}
	systemAuthenticate = func(username, password string) (systemauth.AuthResult, error) {
		return systemauth.AuthResult{}, errors.New("system authentication should not be used in separate mode")
	}

	if _, err := authenticateAdministrator("voxel", "system-secret"); !errors.Is(err, systemauth.ErrInvalidCredentials) {
		t.Fatalf("system password unexpectedly authenticated in separate mode: %v", err)
	}
	if result, err := authenticateAdministrator("voxel", "web-secret"); err != nil || result.Mode != authModeSeparate {
		t.Fatalf("separate credential failed: result=%#v err=%v", result, err)
	}
}
