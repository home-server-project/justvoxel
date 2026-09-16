package systemauth

import (
	"errors"
	"os"
	"os/exec"
	"testing"
)

func TestPAMSystemAccountLifecycleIntegration(t *testing.T) {
	if os.Getenv("JUSTVOXEL_PAM_INTEGRATION") != "1" {
		t.Skip("set JUSTVOXEL_PAM_INTEGRATION=1 inside an AlmaLinux test environment")
	}
	username := os.Getenv("JUSTVOXEL_PAM_TEST_USER")
	oldPassword := os.Getenv("JUSTVOXEL_PAM_TEST_OLD_PASSWORD")
	newPassword := os.Getenv("JUSTVOXEL_PAM_TEST_NEW_PASSWORD")
	if username == "" || oldPassword == "" || newPassword == "" {
		t.Fatal("PAM integration test credentials are not configured")
	}

	policy, err := Policy()
	if err != nil {
		t.Fatalf("read password policy: %v", err)
	}
	if policy.MinLength <= 0 {
		t.Fatalf("invalid password minimum length %d", policy.MinLength)
	}
	t.Logf("effective AlmaLinux libpwquality minimum length: %d", policy.MinLength)

	result, err := Authenticate(username, oldPassword)
	if err != nil {
		t.Fatalf("correct system password rejected before expiration: %v", err)
	}
	if result.PasswordChangeRequired {
		t.Fatal("fresh test password unexpectedly requires a change")
	}
	if _, err := Authenticate(username, "definitely-wrong-password"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("wrong system password returned %v", err)
	}

	if output, err := exec.Command("chage", "-d", "0", username).CombinedOutput(); err != nil {
		t.Fatalf("expire test password: %v: %s", err, output)
	}
	result, err = Authenticate(username, oldPassword)
	if err != nil {
		t.Fatalf("expired system password could not authenticate for change flow: %v", err)
	}
	if !result.PasswordChangeRequired {
		t.Fatal("expired system password was not reported as requiring change")
	}

	if err := ChangePassword(username, oldPassword, newPassword); err != nil {
		t.Fatalf("PAM password change failed: %v", err)
	}
	if _, err := Authenticate(username, oldPassword); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("old password still authenticates after change: %v", err)
	}
	result, err = Authenticate(username, newPassword)
	if err != nil {
		t.Fatalf("new system password rejected after PAM change: %v", err)
	}
	if result.PasswordChangeRequired {
		t.Fatal("new password still marked as expired")
	}
}
