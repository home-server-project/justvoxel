package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestSystemHardwareAPIPathsAndFields(t *testing.T) {
	calls := 0
	client := &Client{http: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Header.Get("Authorization") != "Bearer session-token" {
			t.Fatal("missing authentication")
		}
		if calls == 1 {
			if r.Method != http.MethodGet || r.URL.Path != "/v1/system/hardware" {
				t.Fatalf("status path: %s %s", r.Method, r.URL.Path)
			}
		} else {
			if r.Method != http.MethodPost || r.URL.Path != "/v1/admin/system/display" {
				t.Fatalf("action path: %s %s", r.Method, r.URL.Path)
			}
			var request SystemHardwareRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatal(err)
			}
			if request.Action != "display-delay" || request.DelaySeconds != 10 {
				t.Fatalf("request: %+v", request)
			}
		}
		body := `{"hws":true,"display_available":true,"lid_available":true,"lid_ignored":true,"batteries":[{"capacity":0,"status":"Discharging"},{"status":"Unknown"}]}`
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})}}
	status, err := client.SystemHardwareStatus(context.Background(), "session-token")
	if err != nil {
		t.Fatal(err)
	}
	if !status.HWS || !status.DisplayAvailable || !status.LidAvailable || !status.LidIgnored || len(status.Batteries) != 2 || status.Batteries[0].Capacity == nil || *status.Batteries[0].Capacity != 0 || status.Batteries[1].Capacity != nil {
		t.Fatalf("status: %+v", status)
	}
	if _, err := client.AdminSystemHardwareAction(context.Background(), "session-token", SystemHardwareRequest{"display-delay", 10}); err != nil {
		t.Fatal(err)
	}
}
