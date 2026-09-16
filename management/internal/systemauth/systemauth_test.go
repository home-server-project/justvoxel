package systemauth

import "testing"

func TestHostPasswordPolicyLoads(t *testing.T) {
	policy, err := Policy()
	if err != nil {
		t.Fatalf("load host password policy: %v", err)
	}
	if policy.MinLength <= 0 {
		t.Fatalf("unexpected minimum password length %d", policy.MinLength)
	}
}

func TestPasswordChangeResponse(t *testing.T) {
	const currentPassword = "old-secret"
	const newPassword = "new-secret"

	tests := []struct {
		name    string
		message string
		index   int
		want    string
	}{
		{name: "current password", message: "Current password:", want: currentPassword},
		{name: "old password", message: "Old password:", want: currentPassword},
		{name: "new password", message: "New password:", want: newPassword},
		{name: "retype password", message: "Retype new password:", index: 1, want: newPassword},
		{name: "generic first prompt", message: "Password:", want: newPassword},
		{name: "empty first prompt", message: "", want: newPassword},
		{name: "generic later prompt", message: "Password:", index: 1, want: newPassword},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := passwordChangeResponse(tt.message, tt.index, currentPassword, newPassword); got != tt.want {
				t.Fatalf("passwordChangeResponse(%q, %d) = %q, want %q", tt.message, tt.index, got, tt.want)
			}
		})
	}
}
