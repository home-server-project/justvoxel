package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWhitelistAddAlreadyPresentIsIdempotent(t *testing.T) {
	cases := []struct {
		name     string
		role     principalRole
		platform string
		player   string
		list     string
	}{
		{name: "administrator java", role: roleAdministrator, platform: "java", player: "aaaa", list: "There are 3 whitelisted player(s): aaaa, hhh, .CatchaLlama\n"},
		{name: "operator bedrock prefixed", role: roleOperator, platform: "bedrock", player: ".CatchaLlama", list: "There are 3 whitelisted player(s): aaaa, hhh, .CatchaLlama\n"},
		{name: "operator bedrock plain", role: roleOperator, platform: "bedrock", player: "CatchaLlama", list: "There are 3 whitelisted player(s): aaaa, hhh, .CatchaLlama\n"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := surfaceTestServer(t, tc.role)
			old := runWhitelistHelper
			defer func() { runWhitelistHelper = old }()
			calls := 0
			runWhitelistHelper = func(_ context.Context, args ...string) ([]byte, error) {
				calls++
				if len(args) != 1 || args[0] != "list" {
					t.Fatalf("unexpected mutation helper call: %#v", args)
				}
				return []byte(tc.list), nil
			}

			rr := httptest.NewRecorder()
			body := `{"platform":"` + tc.platform + `","action":"add","name":"` + tc.player + `"}`
			s.whitelistChange(rr, surfaceRequest(http.MethodPost, "/v1/whitelist", body))
			if rr.Code != http.StatusOK {
				t.Fatalf("status = %d: %s", rr.Code, rr.Body.String())
			}
			if calls != 1 {
				t.Fatalf("helper calls = %d, want list only", calls)
			}
			if !strings.Contains(rr.Body.String(), "already on the whitelist") {
				t.Fatalf("missing idempotent message: %s", rr.Body.String())
			}
		})
	}
}

func TestWhitelistRemoveAbsentIsIdempotent(t *testing.T) {
	s := surfaceTestServer(t, roleOperator)
	old := runWhitelistHelper
	defer func() { runWhitelistHelper = old }()
	calls := 0
	runWhitelistHelper = func(_ context.Context, args ...string) ([]byte, error) {
		calls++
		if len(args) != 1 || args[0] != "list" {
			t.Fatalf("unexpected mutation helper call: %#v", args)
		}
		return []byte("There are 2 whitelisted player(s): aaaa, .CatchaLlama\n"), nil
	}

	rr := httptest.NewRecorder()
	s.whitelistChange(rr, surfaceRequest(http.MethodPost, "/v1/whitelist", `{"platform":"java","action":"remove","name":"MissingPlayer"}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rr.Code, rr.Body.String())
	}
	if calls != 1 {
		t.Fatalf("helper calls = %d, want list only", calls)
	}
	if !strings.Contains(rr.Body.String(), "not currently on the whitelist") {
		t.Fatalf("missing idempotent message: %s", rr.Body.String())
	}
}
