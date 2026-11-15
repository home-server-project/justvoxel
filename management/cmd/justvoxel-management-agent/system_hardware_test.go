package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

type hardwareTestInfo struct{ mode os.FileMode }

func (i hardwareTestInfo) Name() string       { return "hardware" }
func (i hardwareTestInfo) Size() int64        { return 0 }
func (i hardwareTestInfo) Mode() os.FileMode  { return i.mode }
func (i hardwareTestInfo) ModTime() time.Time { return time.Time{} }
func (i hardwareTestInfo) IsDir() bool        { return i.mode.IsDir() }
func (i hardwareTestInfo) Sys() any           { return nil }

func hardwareFixture(t *testing.T) {
	t.Helper()
	variant, supply, input, lid, stat, run := hardwareVariantPath, hardwareSupplyPath, hardwareInputPath, hardwareLidPath, hardwareStat, hardwareRun
	t.Cleanup(func() {
		hardwareVariantPath, hardwareSupplyPath, hardwareInputPath, hardwareLidPath, hardwareStat, hardwareRun = variant, supply, input, lid, stat, run
	})
	dir := t.TempDir()
	hardwareVariantPath = filepath.Join(dir, "variant")
	hardwareSupplyPath = filepath.Join(dir, "power_supply")
	hardwareInputPath = filepath.Join(dir, "input")
	hardwareLidPath = filepath.Join(dir, "logind.conf.d", "90-justvoxel-lid.conf")
	if err := os.WriteFile(hardwareVariantPath, []byte("justvoxel-hws\n"), 0600); err != nil {
		t.Fatal(err)
	}
	hardwareStat = func(path string) (os.FileInfo, error) {
		switch path {
		case hardwareConsolePath:
			return hardwareTestInfo{os.ModeDevice | os.ModeCharDevice | 0600}, nil
		case hardwareTTYPath:
			return hardwareTestInfo{os.ModeDir | 0755}, nil
		case "/usr/bin/setterm", "/usr/bin/systemd-run", "/usr/bin/systemctl", systemDisplayHelper:
			return hardwareTestInfo{0755}, nil
		default:
			return nil, os.ErrNotExist
		}
	}
	hardwareRun = func(context.Context, string, ...string) error { t.Fatal("unexpected hardware command"); return nil }
}

func writeHardwareSupply(t *testing.T, name, kind, capacity, status string) {
	t.Helper()
	dir := filepath.Join(hardwareSupplyPath, name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	for file, data := range map[string]string{"type": kind, "capacity": capacity, "status": status} {
		if err := os.WriteFile(filepath.Join(dir, file), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func writeHardwareInput(t *testing.T, input, name string) {
	t.Helper()
	dir := filepath.Join(hardwareInputPath, input)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "name"), []byte(name), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestLidCapabilityRequiresLidSwitchAndSystemctl(t *testing.T) {
	for _, tc := range []struct {
		name   string
		inputs map[string]string
		want   bool
	}{
		{"missing input class", nil, false},
		{"no lid", map[string]string{"input0": "Power Button", "input3": "Sleep Button", "input12": "AT Translated Set 2 keyboard"}, false},
		{"lid", map[string]string{"input7": "Lid Switch"}, true},
		{"trimmed lid", map[string]string{"input23": " \tLid Switch\n"}, true},
		{"inexact names", map[string]string{"input1": "lid switch", "input2": "Lid Switch extra", "input3": "Lid Switch\nextra"}, false},
		{"unrelated entry", map[string]string{"event0": "Lid Switch"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			hardwareFixture(t)
			for input, name := range tc.inputs {
				writeHardwareInput(t, input, name)
			}
			if got := collectSystemHardware().LidAvailable; got != tc.want {
				t.Fatalf("lid available = %t, want %t", got, tc.want)
			}
		})
	}
	t.Run("unreadable name", func(t *testing.T) {
		hardwareFixture(t)
		// A directory produces a read error even when tests run as root.
		if err := os.MkdirAll(filepath.Join(hardwareInputPath, "input4", "name"), 0755); err != nil {
			t.Fatal(err)
		}
		if collectSystemHardware().LidAvailable {
			t.Fatal("unreadable name counted as lid")
		}
		writeHardwareInput(t, "input9", "Lid Switch")
		if !collectSystemHardware().LidAvailable {
			t.Fatal("unreadable entry hid valid lid")
		}
	})
	t.Run("missing systemctl", func(t *testing.T) {
		hardwareFixture(t)
		writeHardwareInput(t, "input5", "Lid Switch")
		complete := hardwareStat
		hardwareStat = func(path string) (os.FileInfo, error) {
			if path == "/usr/bin/systemctl" {
				return nil, os.ErrNotExist
			}
			return complete(path)
		}
		if collectSystemHardware().LidAvailable {
			t.Fatal("lid available without systemctl")
		}
	})
}

func TestHardwareVMHasNoControlsOrBatteryAndRejectsMutations(t *testing.T) {
	hardwareFixture(t)
	writeHardwareInput(t, "input8", "Lid Switch")
	if err := os.WriteFile(hardwareVariantPath, []byte("justvoxel-vm"), 0600); err != nil {
		t.Fatal(err)
	}
	writeHardwareSupply(t, "not-BAT0", "Battery", "62", "Full")
	view := collectSystemHardware()
	if view.HWS || view.DisplayAvailable || view.LidAvailable || view.LidIgnored || len(view.Batteries) != 0 {
		t.Fatalf("VM hardware exposed: %+v", view)
	}
	for _, action := range []string{"display-off", "display-on", "display-delay", "lid-enable", "lid-disable"} {
		request := systemHardwareRequest{Action: action}
		if action == "display-delay" {
			request.DelaySeconds = 10
		}
		if err := applySystemHardware(context.Background(), request); err == nil {
			t.Fatalf("VM allowed %s", action)
		}
	}
}

func TestDisplayCapabilityRequiresConsoleTTYAndTools(t *testing.T) {
	hardwareFixture(t)
	if !collectSystemHardware().DisplayAvailable {
		t.Fatal("complete HWS console unavailable")
	}
	complete := hardwareStat
	for _, missing := range []string{hardwareConsolePath, hardwareTTYPath, "/usr/bin/setterm", "/usr/bin/systemd-run", systemDisplayHelper} {
		hardwareStat = func(path string) (os.FileInfo, error) {
			if path == missing {
				return nil, os.ErrNotExist
			}
			return complete(path)
		}
		if collectSystemHardware().DisplayAvailable {
			t.Fatalf("display available without %s", missing)
		}
	}
	hardwareStat = func(path string) (os.FileInfo, error) {
		if path == hardwareConsolePath {
			return hardwareTestInfo{0600}, nil
		}
		return complete(path)
	}
	if collectSystemHardware().DisplayAvailable {
		t.Fatal("regular file trusted as console")
	}
}

func TestOnboardBatteryAbsenceAndParsing(t *testing.T) {
	hardwareFixture(t)
	if got := collectSystemHardware().Batteries; len(got) != 0 {
		t.Fatalf("absence: %+v", got)
	}
	writeHardwareSupply(t, "AC", "Mains\n", "100", "Unknown")
	writeHardwareSupply(t, "internal-pack", "Battery\n", "73\n", "Discharging\n")
	got := collectSystemHardware().Batteries
	if len(got) != 1 || got[0].Capacity == nil || *got[0].Capacity != 73 || got[0].Status != "Discharging" {
		t.Fatalf("battery: %+v", got)
	}
	for _, malformed := range []string{"", "-1", "101", "1.5", "NaN", "73;touch /tmp/no", "+73", "9999999999999999999999999"} {
		writeHardwareSupply(t, "internal-pack", "Battery", malformed, "invented")
		got = collectSystemHardware().Batteries
		if len(got) != 1 || got[0].Capacity != nil || got[0].Status != "" {
			t.Fatalf("trusted malformed battery %q: %+v", malformed, got)
		}
	}
	for _, value := range []string{"0", "100"} {
		writeHardwareSupply(t, "internal-pack", "Battery", value, "Charging")
		if collectSystemHardware().Batteries[0].Capacity == nil {
			t.Fatalf("valid percentage %s rejected", value)
		}
	}
	if err := os.Remove(filepath.Join(hardwareSupplyPath, "internal-pack", "capacity")); err != nil {
		t.Fatal(err)
	}
	if got := collectSystemHardware().Batteries; len(got) != 1 || got[0].Capacity != nil {
		t.Fatalf("missing capacity: %+v", got)
	}
}

func TestDisplayOnlyUsesFixedCommands(t *testing.T) {
	hardwareFixture(t)
	for _, tc := range []struct {
		action  string
		delay   int
		command string
		args    []string
	}{
		{"display-off", 0, systemDisplayHelper, []string{"off"}},
		{"display-on", 0, systemDisplayHelper, []string{"on"}},
		{"display-delay", 10, "/usr/bin/systemd-run", []string{"--quiet", "--collect", "--timer-property=AccuracySec=1s", "--on-active=10s", systemDisplayHelper, "off"}},
	} {
		calls := 0
		hardwareRun = func(_ context.Context, command string, args ...string) error {
			calls++
			if command != tc.command || !reflect.DeepEqual(args, tc.args) {
				t.Fatalf("unexpected command %s %v", command, args)
			}
			return nil
		}
		if err := applySystemHardware(context.Background(), systemHardwareRequest{tc.action, tc.delay}); err != nil {
			t.Fatal(err)
		}
		if calls != 1 {
			t.Fatalf("calls = %d", calls)
		}
	}
}

func TestHardwareDelayAndActionValidation(t *testing.T) {
	hardwareFixture(t)
	for _, request := range []systemHardwareRequest{{"display-delay", 0}, {"display-delay", -1}, {"display-delay", 3601}, {"display-off;reboot", 0}, {"display-on", 10}, {"lid-enable", 10}} {
		if err := applySystemHardware(context.Background(), request); err == nil {
			t.Fatalf("accepted invalid request %+v", request)
		}
	}
	for _, delay := range []int{1, 3600} {
		if !validHardwareRequest(systemHardwareRequest{"display-delay", delay}) {
			t.Fatalf("rejected valid delay %d", delay)
		}
	}
}

func TestLidPolicyOwnsOnlyOneFileAndReportsStatus(t *testing.T) {
	hardwareFixture(t)
	writeHardwareInput(t, "input6", "Lid Switch")
	if err := os.MkdirAll(filepath.Dir(hardwareLidPath), 0755); err != nil {
		t.Fatal(err)
	}
	unrelated := filepath.Join(filepath.Dir(hardwareLidPath), "99-admin.conf")
	if err := os.WriteFile(unrelated, []byte("administrator config"), 0644); err != nil {
		t.Fatal(err)
	}
	reloads := 0
	hardwareRun = func(_ context.Context, command string, args ...string) error {
		if command != "/usr/bin/systemctl" || !reflect.DeepEqual(args, []string{"reload", "systemd-logind.service"}) {
			t.Fatalf("unexpected reload: %s %v", command, args)
		}
		reloads++
		return nil
	}
	if collectSystemHardware().LidIgnored {
		t.Fatal("absent policy enabled")
	}
	if err := applySystemHardware(context.Background(), systemHardwareRequest{Action: "lid-enable"}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(hardwareLidPath)
	if err != nil || string(data) != lidIgnorePolicy {
		t.Fatalf("wrong policy: %s %v", data, err)
	}
	info, err := os.Stat(hardwareLidPath)
	if err != nil || info.Mode().Perm() != 0644 {
		t.Fatalf("policy mode: %v %v", info, err)
	}
	if !collectSystemHardware().LidIgnored {
		t.Fatal("owned policy not reported")
	}
	if err := applySystemHardware(context.Background(), systemHardwareRequest{Action: "lid-disable"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(hardwareLidPath); !os.IsNotExist(err) {
		t.Fatalf("owned policy remains: %v", err)
	}
	if collectSystemHardware().LidIgnored || reloads != 2 {
		t.Fatal("disable status/reload incorrect")
	}
	data, err = os.ReadFile(unrelated)
	if err != nil || string(data) != "administrator config" {
		t.Fatal("administrator policy changed")
	}
}

func TestLidReloadFailureRestoresOwnedPolicy(t *testing.T) {
	hardwareFixture(t)
	writeHardwareInput(t, "input6", "Lid Switch")
	hardwareRun = func(context.Context, string, ...string) error { return errors.New("reload failed") }
	if err := applySystemHardware(context.Background(), systemHardwareRequest{Action: "lid-enable"}); err == nil {
		t.Fatal("reload failure hidden")
	}
	if _, err := os.Stat(hardwareLidPath); !os.IsNotExist(err) {
		t.Fatal("failed enable persisted")
	}
	if err := os.WriteFile(hardwareLidPath, []byte(lidIgnorePolicy), 0644); err != nil {
		t.Fatal(err)
	}
	if err := applySystemHardware(context.Background(), systemHardwareRequest{Action: "lid-disable"}); err == nil {
		t.Fatal("reload failure hidden")
	}
	if !collectSystemHardware().LidIgnored {
		t.Fatal("failed disable did not restore policy")
	}
}

func TestHardwareAPIRejectsUnauthorizedAndInvalidInputs(t *testing.T) {
	hardwareFixture(t)
	for _, role := range []principalRole{roleViewer, roleOperator} {
		s := roleServerForTest(role)
		rr := httptest.NewRecorder()
		s.systemHardwareAction(rr, authorizedRequest(http.MethodPost, "http://unix/v1/admin/system/display", `{"action":"lid-enable"}`))
		if rr.Code != http.StatusForbidden {
			t.Fatalf("%s mutation = %d", role, rr.Code)
		}
	}
	s := adminServerForTest()
	for _, body := range []string{`{"action":"display-delay","delay_seconds":"10;reboot"}`, `{"action":"display-off","path":"/dev/tty1"}`, `{"action":"display-delay","delay_seconds":1.5}`, `{"action":"display-delay","delay_seconds":3601}`} {
		rr := httptest.NewRecorder()
		s.systemHardwareAction(rr, authorizedRequest(http.MethodPost, "http://unix/v1/admin/system/display", body))
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("invalid body %s: %d %s", body, rr.Code, rr.Body.String())
		}
	}
}

func TestSystemDisplayHelperContract(t *testing.T) {
	data, err := os.ReadFile("../../../mjust/libexec/system-display")
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	for _, want := range []string{"${EUID} -eq 0 && $# -eq 1", "justvoxel-hws", "*) exit 2", "/usr/bin/setterm --term linux --blank force --powersave powerdown < /dev/tty0 > /dev/tty0", "/usr/bin/setterm --term linux --blank poke < /dev/tty0 > /dev/tty0"} {
		if !strings.Contains(source, want) {
			t.Fatalf("helper missing %q", want)
		}
	}
	for _, forbidden := range []string{"eval", "sh -c", "bl_power", "brightness", "minecraft", "reboot", "poweroff"} {
		if strings.Contains(source, forbidden) {
			t.Fatalf("helper contains %q", forbidden)
		}
	}
}
