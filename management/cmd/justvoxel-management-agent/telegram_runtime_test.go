package main

import (
	"context"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"testing"
)

func TestTelegramStartOnBootPreservesUnrelatedRuntimeSettings(t *testing.T) {
	f := newInstanceBackupFixture(t)
	f.backend.networkRoot = filepath.Join(t.TempDir(), "network")
	id := f.configs[0].ID
	before, err := f.backend.readRecord(id)
	if err != nil {
		t.Fatal(err)
	}
	request := authorizedRequest("PUT", "/v1/telegram/instances/"+id+"/boot", string(telegramBody(map[string]bool{"enabled": !before.Runtime.StartOnBoot})))
	request.SetPathValue("id", id)
	rr := httptest.NewRecorder()
	f.server.telegramStartOnBoot(rr, request)
	if rr.Code != 200 {
		t.Fatal(rr.Code, rr.Body.String())
	}
	after, err := f.backend.readRecord(id)
	if err != nil {
		t.Fatal(err)
	}
	before.Runtime.StartOnBoot = !before.Runtime.StartOnBoot
	if !reflect.DeepEqual(before.Runtime, after.Runtime) || !reflect.DeepEqual(before.Storage, after.Storage) || !reflect.DeepEqual(before.Backups, after.Backups) {
		t.Fatal("boot action modified unrelated configuration")
	}
}
func TestTelegramNarrowSettingsRejectGeneralConfiguration(t *testing.T) {
	s := roleServerForTest(roleAdministrator)
	rr := httptest.NewRecorder()
	request := authorizedRequest("PUT", "/v1/telegram/instances/jv-abcdef/boot", `{"enabled":true,"java_port":25566}`)
	request.SetPathValue("id", "jv-abcdef")
	s.telegramStartOnBoot(rr, request)
	if rr.Code != 400 {
		t.Fatal("general runtime mutation accepted", rr.Code)
	}
	rr = httptest.NewRecorder()
	s.telegramNetworkChange(rr, authorizedRequest("PUT", "/v1/telegram/network", `{"enabled":true,"java_port":25566}`))
	if rr.Code != 400 {
		t.Fatal("general Network configuration accepted", rr.Code)
	}
}
func TestTelegramNetworkMembershipRequiresStoppedAffectedServers(t *testing.T) {
	f, c := networkTransactionFixture(t)
	oldBackend, oldLock := instanceRuntimes, pluginRuntimeLockPath
	instanceRuntimes = f.backend
	pluginRuntimeLockPath = filepath.Join(t.TempDir(), "maintenance.lock")
	t.Cleanup(func() { instanceRuntimes = oldBackend; pluginRuntimeLockPath = oldLock })
	s := roleServerForTest(roleAdministrator)
	s.operations = openTestOperationStore(t)
	id := f.configs[0].ID
	if err := f.backend.action(context.Background(), id, "start"); err != nil {
		t.Fatal(err)
	}
	previous, err := networkRead(f.backend)
	if err != nil {
		t.Fatal(err)
	}
	raw := telegramBody(map[string]any{"members": c.Members, "expected": telegramNetworkFingerprint(previous.Config)})
	rr := httptest.NewRecorder()
	s.telegramNetworkChange(rr, authorizedRequest("PUT", "/v1/telegram/network", string(raw)))
	if rr.Code != 409 {
		t.Fatal("running member accepted", rr.Code, rr.Body.String())
	}
	after, err := networkRead(f.backend)
	if err != nil || !reflect.DeepEqual(previous, after) {
		t.Fatal("failed precondition changed network state", err)
	}
	state, err := f.backend.observe(context.Background(), id)
	if err != nil || state.State != "running" {
		t.Fatal("precondition stopped server", err)
	}
}
func TestTelegramNetworkFingerprintPreservesLegacyWhitelistDefault(t *testing.T) {
	legacy := networkDefaultState().Config
	explicit := legacy
	enforced := true
	explicit.WhitelistEnforced = &enforced
	if telegramNetworkFingerprint(legacy) != telegramNetworkFingerprint(explicit) {
		t.Fatal("legacy enforced setting strands current menu")
	}
	disabled := false
	explicit.WhitelistEnforced = &disabled
	if telegramNetworkFingerprint(legacy) == telegramNetworkFingerprint(explicit) {
		t.Fatal("changed Network setting not detected")
	}
}
