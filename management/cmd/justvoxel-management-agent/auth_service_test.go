package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/home-server-project/justvoxel/management/internal/systemauth"
)

func TestAuthenticateAdministratorRejectsOtherUsername(t *testing.T) {
	_, err := authenticateAdministrator("admin", "anything")
	if !errors.Is(err, systemauth.ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials, got %v", err)
	}
}

func TestMissingAuthModeDefaultsToSystem(t *testing.T) {
	if _, err := os.Stat(authModePath); err == nil {
		t.Skip("auth mode state exists on test host")
	}
	mode, err := currentAuthMode()
	if err != nil {
		t.Fatal(err)
	}
	if mode != authModeSystem {
		t.Fatalf("expected system mode, got %q", mode)
	}
}

func TestLocalAccountPasswordVerification(t *testing.T) {
	account, err := localAccountForPassword(systemAdminUsername, "administrator", "separate-secret")
	if err != nil {
		t.Fatal(err)
	}
	if !verifyLocalAccount(account, "separate-secret") {
		t.Fatal("correct separate password did not verify")
	}
	if verifyLocalAccount(account, "wrong") {
		t.Fatal("wrong separate password verified")
	}
}

func TestAuthModeStateRoundTripHelpers(t *testing.T) {
	state := authModeState{Version: authModeVersion, Mode: authModeSeparate}
	if state.Version != 1 || state.Mode != authModeSeparate {
		t.Fatal("unexpected auth mode state")
	}

	store := localAuthStore{Version: localAuthVersion, Accounts: []localAccount{{Username: systemAdminUsername, Role: "administrator"}}}
	if filepath.Base(localAuthPath) != "local-auth.json" || len(store.Accounts) != 1 {
		t.Fatal("unexpected local auth store shape")
	}
}
