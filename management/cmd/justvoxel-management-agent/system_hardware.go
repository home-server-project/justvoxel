package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const systemDisplayHelper = "/usr/libexec/justvoxel/mjust/system-display"
const lidIgnorePolicy = "# JustVoxel-owned lid policy\n[Login]\nHandleLidSwitch=ignore\nHandleLidSwitchExternalPower=ignore\nHandleLidSwitchDocked=ignore\n"

var hardwareVariantPath = "/usr/lib/justvoxel/variant"
var hardwareSupplyPath = "/sys/class/power_supply"
var hardwareInputPath = "/sys/class/input"
var hardwareConsolePath = "/dev/tty0"
var hardwareTTYPath = "/sys/class/tty/tty0"
var hardwareLidPath = "/etc/systemd/logind.conf.d/90-justvoxel-lid.conf"
var hardwareStat = os.Stat
var hardwareRun = func(ctx context.Context, command string, args ...string) error {
	return exec.CommandContext(ctx, command, args...).Run()
}
var hardwareMu sync.Mutex

type onboardBattery struct {
	Capacity *int   `json:"capacity,omitempty"`
	Status   string `json:"status,omitempty"`
}
type systemHardwareStatus struct {
	HWS              bool             `json:"hws"`
	DisplayAvailable bool             `json:"display_available"`
	LidAvailable     bool             `json:"lid_available"`
	LidIgnored       bool             `json:"lid_ignored"`
	Batteries        []onboardBattery `json:"batteries"`
}
type systemHardwareRequest struct {
	Action       string `json:"action"`
	DelaySeconds int    `json:"delay_seconds"`
}

func hardwareHWS() bool {
	data, err := os.ReadFile(hardwareVariantPath)
	return err == nil && strings.TrimSpace(string(data)) == "justvoxel-hws"
}
func hardwareExecutable(path string) bool {
	info, err := hardwareStat(path)
	return err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0111 != 0
}
func hardwareHasLidSwitch() bool {
	names, err := filepath.Glob(filepath.Join(hardwareInputPath, "input*", "name"))
	if err != nil {
		return false
	}
	for _, name := range names {
		data, err := os.ReadFile(name)
		if err == nil && strings.TrimSpace(string(data)) == "Lid Switch" {
			return true
		}
	}
	return false
}
func collectSystemHardware() systemHardwareStatus {
	view := systemHardwareStatus{HWS: hardwareHWS(), Batteries: []onboardBattery{}}
	if !view.HWS {
		return view
	}
	console, err := hardwareStat(hardwareConsolePath)
	tty, ttyErr := hardwareStat(hardwareTTYPath)
	view.DisplayAvailable = err == nil && console.Mode()&os.ModeCharDevice != 0 && ttyErr == nil && tty.IsDir() && hardwareExecutable("/usr/bin/setterm") && hardwareExecutable("/usr/bin/systemd-run") && hardwareExecutable(systemDisplayHelper)
	view.LidAvailable = hardwareExecutable("/usr/bin/systemctl") && hardwareHasLidSwitch()
	hardwareMu.Lock()
	data, _ := os.ReadFile(hardwareLidPath)
	view.LidIgnored = string(data) == lidIgnorePolicy
	hardwareMu.Unlock()
	entries, _ := os.ReadDir(hardwareSupplyPath)
	for _, entry := range entries {
		base := filepath.Join(hardwareSupplyPath, entry.Name())
		kind, err := os.ReadFile(filepath.Join(base, "type"))
		if err != nil || strings.TrimSpace(string(kind)) != "Battery" {
			continue
		}
		battery := onboardBattery{}
		data, err := os.ReadFile(filepath.Join(base, "capacity"))
		if err == nil {
			raw := strings.TrimSpace(string(data))
			value, parseErr := strconv.Atoi(raw)
			if parseErr == nil && value >= 0 && value <= 100 && raw == strconv.Itoa(value) {
				battery.Capacity = &value
			}
		}
		data, err = os.ReadFile(filepath.Join(base, "status"))
		if err == nil {
			switch status := strings.TrimSpace(string(data)); status {
			case "Charging", "Discharging", "Not charging", "Full", "Unknown":
				battery.Status = status
			}
		}
		view.Batteries = append(view.Batteries, battery)
	}
	return view
}

func (s *server) systemHardwareStatus(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireReadAccess(w, r); !ok {
		return
	}
	writeJSON(w, http.StatusOK, collectSystemHardware())
}

func validHardwareRequest(request systemHardwareRequest) bool {
	switch request.Action {
	case "display-off", "display-on", "lid-enable", "lid-disable":
		return request.DelaySeconds == 0
	case "display-delay":
		return request.DelaySeconds >= 1 && request.DelaySeconds <= 3600
	default:
		return false
	}
}

func applySystemHardware(ctx context.Context, request systemHardwareRequest) error {
	if !validHardwareRequest(request) {
		return errors.New("invalid display or lid action")
	}
	status := collectSystemHardware()
	if !status.HWS {
		return errors.New("hardware controls require HWS")
	}
	switch request.Action {
	case "display-off", "display-on", "display-delay":
		if !status.DisplayAvailable {
			return errors.New("local console display is unavailable")
		}
		switch request.Action {
		case "display-off":
			return hardwareRun(ctx, systemDisplayHelper, "off")
		case "display-on":
			return hardwareRun(ctx, systemDisplayHelper, "on")
		default:
			// systemd chooses a unique transient unit name; only the bounded integer varies.
			return hardwareRun(ctx, "/usr/bin/systemd-run", "--quiet", "--collect", "--timer-property=AccuracySec=1s", "--on-active="+strconv.Itoa(request.DelaySeconds)+"s", systemDisplayHelper, "off")
		}
	case "lid-enable", "lid-disable":
		if !status.LidAvailable {
			return errors.New("lid policy control is unavailable")
		}
		return applyLidPolicy(ctx, request.Action == "lid-enable")
	}
	return errors.New("unsupported hardware action")
}

func applyLidPolicy(ctx context.Context, enabled bool) error {
	hardwareMu.Lock()
	defer hardwareMu.Unlock()
	old, err := os.ReadFile(hardwareLidPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	existed := err == nil
	if enabled {
		if err := os.MkdirAll(filepath.Dir(hardwareLidPath), 0755); err != nil {
			return err
		}
		if err := atomicWrite(hardwareLidPath, []byte(lidIgnorePolicy), 0644); err != nil {
			return err
		}
	} else if err := os.Remove(hardwareLidPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	// Type=notify-reload logind reloads and acknowledges without restarting sessions.
	if err := hardwareRun(ctx, "/usr/bin/systemctl", "reload", "systemd-logind.service"); err != nil {
		var rollbackErr error
		if existed {
			rollbackErr = atomicWrite(hardwareLidPath, old, 0644)
		} else {
			rollbackErr = os.Remove(hardwareLidPath)
			if os.IsNotExist(rollbackErr) {
				rollbackErr = nil
			}
		}
		return errors.Join(err, rollbackErr)
	}
	return nil
}

func (s *server) systemHardwareAction(w http.ResponseWriter, r *http.Request) {
	actor, ok := s.requireAdministrator(w, r)
	if !ok {
		return
	}
	var request systemHardwareRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if !validHardwareRequest(request) {
		writeError(w, http.StatusBadRequest, "invalid display or lid action; delay must be 1–3600 seconds")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	err := applySystemHardware(ctx, request)
	if s.store != nil {
		_ = s.store.recordAuditEvent(actor, "system_"+request.Action, "host", err == nil, "display or lid action")
	}
	if err != nil {
		writeError(w, http.StatusConflict, "display or lid action could not be applied")
		return
	}
	writeJSON(w, http.StatusOK, collectSystemHardware())
}
