package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestTelegramAdministrationUsesExistingAuthenticatedUnixClient(t *testing.T) {
	calls := 0
	c := &Client{http: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Path != "/v1/admin/telegram" || r.Header.Get("Authorization") != "Bearer session-token" {
			t.Fatal("wrong Telegram transport boundary")
		}
		if calls == 1 && r.Method != "GET" || calls == 2 && r.Method != "POST" {
			t.Fatal("wrong settings method")
		}
		if calls == 2 {
			raw, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(raw), `"action":"pair"`) {
				t.Fatal("typed action missing")
			}
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"enabled":false,"users":[]}`))}, nil
	})}}
	if _, err := c.TelegramAdmin(context.Background(), "session-token", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := c.TelegramAdmin(context.Background(), "session-token", json.RawMessage(`{"action":"pair"}`)); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatal("missing settings roundtrip")
	}
}
