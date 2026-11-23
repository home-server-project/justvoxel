package main

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"
)

type reviewedMigrationInstanceKey struct{}

type instanceVersionKey struct{}
type instanceVersionSelectionKey struct{}

func validatedVersionInstance(w http.ResponseWriter, r *http.Request, live ...bool) (string, bool) {
	id, ok := instanceRequestID(w, r)
	if !ok {
		return "", false
	}
	record, err := instanceRuntimes.readRecord(id)
	if err == nil && (len(live) == 0 || live[0]) {
		err = instanceRuntimes.verifyLive(record)
	}
	if err != nil {
		instanceRuntimeError(w, err, id)
		return "", false
	}
	return id, true
}

// These bounded root-only commands are used by the mature update worker while
// it holds appliance maintenance. They expose no generic shell/container action.
func instanceVersionCommand(ctx context.Context, args []string) error {
	if os.Geteuid() != 0 || len(args) < 2 || !instanceIDPattern.MatchString(args[1]) {
		return errInstanceID
	}
	b := instanceRuntimes
	id := args[1]
	containerOnly := args[0] == "instance-container-publish"
	if containerOnly {
		if len(args) != 3 || (args[2] != "" && !operationFingerprintPattern.MatchString(args[2])) {
			return errInstanceID
		}
		pending, err := b.pending(id)
		if err != nil {
			return err
		}
		if pending {
			recoveryCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
			defer cancel()
			if err := b.recover(recoveryCtx, id); err != nil {
				return err
			}
		}
	}
	previous, err := b.readRecord(id)
	if err != nil {
		return err
	}
	if err = b.verifyLive(previous); err != nil {
		return err
	}
	if containerOnly {
		args = []string{"instance-version-publish", id, previous.Runtime.Software, previous.Runtime.ImageTag, previous.Runtime.Version, previous.Runtime.VersionPolicy, args[2]}
	}
	switch args[0] {
	case "instance-switch-content-verify":
		if len(args) != 5 || !validSetupServerType(args[2]) || !instanceRuntimeVersionPattern.MatchString(args[3]) || args[3] == "LATEST" {
			return errInstanceID
		}
		return verifySwitchContent(ctx, id, args[2], args[3], args[4])
	case "instance-version-backup":
		if len(args) != 2 {
			return errInstanceID
		}
		return instanceMaintenanceColdBackup(ctx, id)
	case "instance-version-publish":
		migration := len(args) == 9 && args[7] == "--migration"
		if (len(args) != 7 && !migration) || (args[6] != "" && !operationFingerprintPattern.MatchString(args[6])) || !validSetupServerType(args[2]) || !imageTagInputPattern.MatchString(args[3]) || !(instanceRuntimeVersionPattern.MatchString(args[4]) || migration && args[2] == "bedrock" && bedrockVersionPattern.MatchString(args[4])) || (!containerOnly && args[5] != "recommended" && args[5] != "latest" && args[5] != "pinned") {
			return errInstanceID
		}
		if !javaServerAllowsSwitch(previous.Runtime.Software, args[2]) {
			return instanceRuntimeFailure("Software conversion requires a separate supported operation.")
		}
		if !migration && !containerOnly && !javaServerPaperFamily(previous.Runtime.Software) && previous.Runtime.Software != "vanilla" && (args[2] != previous.Runtime.Software || args[4] != previous.Runtime.Version || args[5] != previous.Runtime.VersionPolicy) {
			return instanceRuntimeFailure("Version updates for this server software require separate compatibility support.")
		}
		snapshot, _, _, err := b.snapshot(ctx)
		if err != nil {
			return err
		}
		records, err := b.records()
		if err != nil {
			return err
		}
		next := previous
		if migration {
			next, err = b.migrationPortableSettings(previous, args[8])
			if err != nil {
				return err
			}
			if args[2] != previous.Runtime.Software || args[3] != previous.Runtime.ImageTag || args[6] != previous.Runtime.ImageDigest || args[5] != "pinned" || next.Runtime.VersionPolicy != "pinned" || args[4] != next.Runtime.Version {
				return errInstanceRecovery
			}
			ctx = context.WithValue(ctx, reviewedMigrationInstanceKey{}, id)
		} else {
			next.Runtime.PreserveContent = os.Getenv("JUSTVOXEL_SWITCH_PRESERVE_CONTENT") == "yes"
		}
		next.Runtime.Software = args[2]
		next.Runtime.ImageTag = args[3]
		next.Runtime.Version = args[4]
		next.Runtime.VersionPolicy = args[5]
		next.Runtime.ImageDigest = args[6]
		if previous.Runtime.Version != next.Runtime.Version || previous.Runtime.Software != next.Runtime.Software {
			channel, err := b.command(ctx, instanceRuntimeHelper, "channel", next.Runtime.Software, next.Runtime.Version, next.Runtime.VersionPolicy)
			if err != nil {
				return instanceRuntimeFailure("Selected release channel could not be verified.")
			}
			next.Runtime.PaperChannel = strings.TrimSpace(string(channel))
		}
		return b.applyVersion(ctx, previous, instanceReplaceRecord(records, next), snapshot, true, containerOnly)
	}
	return errInstanceID
}

// instanceMaintenanceColdBackup leaves only the selected instance stopped for maintenance.
func instanceMaintenanceColdBackup(ctx context.Context, id string) error {
	if !instanceIDPattern.MatchString(id) {
		return errInstanceID
	}
	switchVersion := os.Getenv("JUSTVOXEL_SWITCH_BACKUP_VERSION")
	if switchVersion != "" && (switchVersion == "LATEST" || !instanceRuntimeVersionPattern.MatchString(switchVersion)) {
		return errInstanceID
	}
	b := instanceRuntimes
	record, err := b.readRecord(id)
	if err != nil {
		return err
	}
	if err = b.verifyLive(record); err != nil {
		return err
	}
	if _, err = b.backupStorage(ctx, "preflight", id); err != nil {
		return err
	}
	state, err := b.observe(ctx, id)
	if err != nil {
		return err
	}
	if state.State == "running" {
		if err = b.actionWithWarnings(ctx, id, "stop", true); err != nil {
			return err
		}
	} else if state.State != "stopped" {
		return errInstanceBusy
	}
	// Switch's backup records the verified fixed game version, even when the
	// source normally follows releases. Ordinary version/container maintenance
	// keeps its established backup behavior.
	if switchVersion != "" {
		if err = b.prepareSwitchSourceCopy(ctx, id, switchVersion); err != nil {
			return err
		}
	}
	_, err = b.backupStorage(ctx, "create", id)
	return err
}
