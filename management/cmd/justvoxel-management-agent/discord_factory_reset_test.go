package main

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestFactoryResetDiscordRevocationBoundary(t *testing.T) {
	for _, failRevoke := range []bool{false, true} {
		name := "revoked_before_cleanup"
		if failRevoke {
			name = "revocation_failure_stops_cleanup"
		}
		t.Run(name, func(t *testing.T) {
			useEmptyFactoryResetRuntime(t)
			useTestMinecraftInstances(t)
			s := surfaceTestServer(t, roleAdministrator)
			operations := openTestOperationStore(t)
			attachTestOperationStore(t, s, operations)
			configureTestDiscord(t, s)
			initial := discordVerifiedState()
			initial.Enabled = true
			discordSaveTestState(t, s, initial)
			client := &discordFakeGateway{}
			s.discordRuntime = &discordRuntime{generation: initial.Generation, client: client}
			if err := s.store.telegramSave(telegramTestState(), nil, ""); err != nil {
				t.Fatal(err)
			}
			if failRevoke {
				// Fail only Discord's durable write. Telegram revocation remains real.
				if _, err := s.store.db.Exec(`CREATE TRIGGER fail_discord_revocation BEFORE UPDATE ON discord_state BEGIN SELECT RAISE(ABORT,'private storage failure'); END`); err != nil {
					t.Fatal(err)
				}
			}
			old := runAdminFactoryResetHelper
			defer func() { runAdminFactoryResetHelper = old }()
			cleanupCalls := 0
			runAdminFactoryResetHelper = func(_ context.Context, args ...string) ([]byte, error) {
				if len(args) == 1 && args[0] == "plan" {
					return []byte(validFactoryResetPlanHelper), nil
				}
				cleanupCalls++
				if client.closes.Load() == 0 || s.discordRuntime != nil {
					t.Fatal("cleanup began with a live Discord runtime")
				}
				if failRevoke {
					t.Fatal("destructive cleanup continued after Discord revocation failure")
				}
				v, err := s.store.discordRead()
				if err != nil || v.Token != "" || v.Verified || v.BotID != "" || v.ApplicationID != "" || v.Status != "disconnected" {
					t.Fatal("cleanup began before Discord revocation", err)
				}
				// Deliberately stop later cleanup: revocation must remain durable even if
				// the rest of Factory Reset cannot finish.
				return nil, errors.New("simulated later cleanup failure")
			}
			fingerprint := exactFactoryResetFingerprint(t)
			operation, _, err := operations.beginFactoryReset(fingerprint)
			if err != nil {
				t.Fatal(err)
			}
			if err := executeFactoryReset(context.Background(), s, operation.OperationID, fingerprint); err == nil {
				t.Fatal("expected reset failure")
			}
			if client.closes.Load() == 0 || s.discordRuntime != nil {
				t.Fatal("Factory Reset failed to stop Discord")
			}
			finished, err := operations.get(operation.OperationID)
			if err != nil || finished.State != operationNeedsAttention {
				t.Fatal("reset did not preserve needs-attention recovery", err)
			}
			telegram, err := s.store.telegramRead()
			if err != nil || telegram.Token != "" || telegram.Enabled || telegram.Verified {
				t.Fatal("Telegram revocation weakened", err)
			}
			if failRevoke {
				if cleanupCalls != 0 || finished.Stage != "discord_revocation_failed" {
					t.Fatal("incorrect reset failure boundary", cleanupCalls, finished.Stage)
				}
				v, err := s.store.discordRead()
				if err != nil || v.Token != discordTestToken {
					t.Fatal("unexpected partial Discord revocation", err)
				}
				rr := discordAdminRequest(s, administratorSession(time.Now()), map[string]any{"action": "credential", "token": "replacement"})
				if rr.Code != 409 {
					t.Fatal("pending reset allowed credential replacement", rr.Code)
				}
				if _, err := s.store.db.Exec(`DROP TRIGGER fail_discord_revocation`); err != nil {
					t.Fatal(err)
				}
				// Retry the supported reset revocation after storage becomes writable.
				if err := s.discordFactoryRevoke(); err != nil {
					t.Fatal("revocation retry failed", err)
				}
			} else if cleanupCalls != 1 {
				t.Fatal("cleanup boundary was not reached", cleanupCalls)
			}
			var sequence int
			var dbName, path string
			if err := s.store.db.QueryRow(`PRAGMA database_list`).Scan(&sequence, &dbName, &path); err != nil {
				t.Fatal(err)
			}
			if err := s.store.close(); err != nil {
				t.Fatal(err)
			}
			reopened, err := openWebUIStore(path)
			if err != nil {
				t.Fatal(err)
			}
			defer reopened.close()
			v, err := reopened.discordRead()
			if err != nil || v.Token != "" || v.Verified || v.Status != "disconnected" {
				t.Fatal("Discord revocation was not durable", err)
			}
		})
	}
}

func TestDiscordFactoryIdentityResetDefaults(t *testing.T) {
	s := discordTestServer(t)
	configureTestDiscord(t, s)
	if err := s.store.resetFactoryState(); err != nil {
		t.Fatal(err)
	}
	v, err := s.store.discordRead()
	if err != nil || !reflect.DeepEqual(v, discordState{Schema: 1, Status: "disconnected"}) {
		t.Fatal("identity reset retained Discord state", err)
	}
}
