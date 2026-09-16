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
	oldPassword := os.Getenv("JUSTVOXEL_PAM_TEST_OLD