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
