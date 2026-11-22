package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestPluginManageExplicitClientContracts(t *testing.T) {
	for _, action := range []string{"enable", "disable", "remove"} {
		t.Run(action, func(t *testing.T) {
			calls := []string{}
			client := &Client{http: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls = append(calls, r.URL.Path)
				if r.Header.Get("Authorization") != "Bearer session" {
					t.Fatal("administrator session not forwarded")
				}
				payload := `{"operation":{"instance_id":"jv-family1","operation_id":"` + resetTestOperationID + `","operation_type":"plugin_` + action + `","state":"running"}}`
				if r.URL.Path == "/v1/minecraft/instances/jv-family1/plugins/manage/plan" {
					var request PluginManageRequest
					if json.NewDecoder(r.Body).Decode(&request) != nil || request.Action != action || request.ProjectID != "root" {
						t.Fatal("plan identity changed")
					}
					payload = `{"server":{"instance_id":"jv-family1"},"action":"` + action + `","project_id":"root","state":"enabled","plan_fingerprint":"` + resetTestFingerprint + `"}`
				}
				if r.URL.Path == "/v1/minecraft/instances/jv-family1/plugins/manage" {
					var request PluginManageApplyRequest
					if json.NewDecoder(r.Body).Decode(&request) != nil || request.Action != action || request.PlanFingerprint != resetTestFingerprint || !request.ConfirmPlayers {
						t.Fatal("apply confirmation changed")
					}
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(payload)), Header: make(http.Header)}, nil
			})}}
			if _, err := client.AdminPluginManagePlan(WithMinecraftMaintenanceInstance(context.Background(), "jv-family1", "plugins"), "session", PluginManageRequest{Action: action, ProjectID: "root"}); err != nil {
				t.Fatal(err)
			}
			if _, err := client.AdminPluginManage(WithMinecraftMaintenanceInstance(context.Background(), "jv-family1", "plugins"), "session", PluginManageApplyRequest{Action: action, ProjectID: "root", PlanFingerprint: resetTestFingerprint, ConfirmPlayers: true}); err != nil {
				t.Fatal(err)
			}
			if _, err := client.AdminPluginManageAcknowledge(WithMinecraftMaintenanceInstance(context.Background(), "jv-family1", "plugins"), "session", PluginInstallAcknowledgeRequest{resetTestOperationID}); err != nil {
				t.Fatal(err)
			}
			if _, err := client.AdminCurrentPluginOperation(WithMinecraftMaintenanceInstance(context.Background(), "jv-family1", "plugins"), "session"); err != nil {
				t.Fatal(err)
			}
			if strings.Join(calls, ",") != "/v1/minecraft/instances/jv-family1/plugins/manage/plan,/v1/minecraft/instances/jv-family1/plugins/manage,/v1/minecraft/instances/jv-family1/plugins/manage/acknowledge,/v1/minecraft/instances/jv-family1/plugins/operation" {
				t.Fatal(calls)
			}
		})
	}
}
func TestPluginManageRejectsArbitraryActionsAndInvalidIdentity(t *testing.T) {
	client := &Client{http: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { t.Fatal("invalid request reached Agent"); return nil, nil })}}
	for _, req := range []PluginManageRequest{{Action: "update", ProjectID: "root"}, {Action: "delete", ProjectID: "root"}, {Action: "enable", ProjectID: "../root"}, {Action: "disable"}} {
		if _, err := client.AdminPluginManagePlan(WithMinecraftMaintenanceInstance(context.Background(), "jv-family1", "plugins"), "session", req); err == nil {
			t.Fatal("invalid request accepted", req)
		}
	}
	if _, err := client.AdminPluginManage(WithMinecraftMaintenanceInstance(context.Background(), "jv-family1", "plugins"), "session", PluginManageApplyRequest{Action: "remove", ProjectID: "root", PlanFingerprint: "invalid"}); err == nil {
		t.Fatal("invalid fingerprint accepted")
	}
	if _, err := client.AdminPluginManageAcknowledge(WithMinecraftMaintenanceInstance(context.Background(), "jv-family1", "plugins"), "session", PluginInstallAcknowledgeRequest{OperationID: "../operation"}); err == nil {
		t.Fatal("invalid acknowledgement accepted")
	}
}

func TestPluginInstallRestoreRuntimeUsesSelectedIdentityAndIntent(t *testing.T) {
	const operationID = "12345678-1234-4123-8123-123456789abc"
	ctx := WithMinecraftMaintenanceInstance(context.Background(), "jv-family1", "plugins")
	for _, confirm := range []bool{false, true} {
		client := &Client{http: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/v1/minecraft/instances/jv-family1/plugins/install/restore-runtime" ||
				r.Method != "POST" || r.Header.Get("Authorization") != "Bearer session-token" {
				t.Fatal("wrong recovery target or authorization", r.Method, r.URL.Path)
			}
			raw, err := io.ReadAll(r.Body)
			want := `{"operation_id":"` + operationID + `","restore_runtime":true`
			if confirm {
				want += `,"confirm_players":true`
			}
			if err != nil || string(raw) != want+"}" {
				t.Fatal("unexpected recovery authority", string(raw), err)
			}
			code, body := 200, `{"operation":{"operation_id":"`+operationID+`","instance_id":"jv-family1","operation_type":"plugin_install","state":"resolved"}}`
			if !confirm {
				code, body = 409, `{"id":"jv-family1","recovery_required":true,"recovery_kind":"plugin_install","recovery_operation_id":"`+operationID+`","recovery_action":"stop","message":"Players are online","confirmation_required":true,"online":1,"players":["Alex"]}`
			}
			return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body))}, nil
		})}}
		out, err := client.AdminPluginInstallRestoreRuntime(ctx, "session-token", PluginInstallRestoreRuntimeRequest{
			OperationID: operationID, RestoreRuntime: true, ConfirmPlayers: confirm,
		})
		if confirm {
			if err != nil || out.Operation == nil || out.Operation.InstanceID != "jv-family1" {
				t.Fatal("selected resolution discarded", out, err)
			}
		} else {
			var refusal *ResponseError
			if !errors.As(err, &refusal) || refusal.PluginRecovery == nil || !refusal.PluginRecovery.ConfirmationRequired ||
				refusal.PluginRecovery.Online != 1 || len(refusal.PluginRecovery.Players) != 1 || refusal.PluginRecovery.Players[0] != "Alex" {
				t.Fatal("player confirmation discarded", err)
			}
		}
	}
}
