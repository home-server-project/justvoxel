package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestPasswordHashVerification(t *testing.T) {
	cred, err := credentialForPassword("correct horse battery staple", true)
	if err != nil {
		t.Fatal(err)
	}
	if !cred.MustChange {
		t.Fatal("expected bootstrap credential to require password change")
	}
	if !verifyPassword(cred, "correct horse battery staple") {
		t.Fatal("correct password did not verify")
	}
	if verifyPassword(cred, "wrong password") {
		t.Fatal("wrong password verified")
	}
}

func TestRandomPasswordPolicy(t *testing.T) {
	password, err := randomPassword(14)
	if err != nil {
		t.Fatal(err)
	}
	if len(password) != 14 {
		t.Fatalf("got password length %d", len(password))
	}
	for _, char := range password {
		if !strings.ContainsRune(passwordChars, char) {
			t.Fatalf("password contains unsupported character %q", char)
		}
	}
}

func TestMustChangeSessionCannotAccessAdminEndpoint(t *testing.T) {
	now := time.Now()
	s := &server{sessions: map[string]session{
		"token": {Created: now, LastSeen: now, MustChange: true},
	}}
	req := httptest.NewRequest(http.MethodGet, "http://unix/v1/status", nil)
	req.Header.Set("Authorization", "Bearer token")
	_, sess, ok := s.authorize(req, false)
	if ok || !sess.MustChange {
		t.Fatal("must-change session was allowed through normal authorization")
	}
	if _, _, ok := s.authorize(req, true); !ok {
		t.Fatal("must-change session was not allowed for credential update path")
	}
}

func TestLoginThrottleActivates(t *testing.T) {
	s := &server{}
	for i := 0; i < 5; i++ {
		s.recordFailure()
	}
	if delay := s.loginDelay(); delay <= 0 {
		t.Fatal("expected login throttling after repeated failures")
	}
}
