package main

import (
	"context"
	"errors"
	"os"
	"regexp"
	"strings"
	"time"
)

var switchStartupHistory = regexp.MustCompile(`(?i)starting minecraft server version|loading minecraft [0-9]|preparing (?:level|start region|spawn area)|done \([0-9.]+s\)! for help|world storage migration|\[server thread[ /\]]`)

func (b *instanceRuntimeBackend) switchInitializationEvidence(ctx context.Context, record instanceRuntimeRecord) (string, error) {
	id := record.Runtime.ID
	if !instanceIDPattern.MatchString(id) || record.Runtime.VersionPolicy != "pinned" || record.Runtime.Version == "LATEST" ||
		!instanceRuntimeVersionPattern.MatchString(record.Runtime.Version) {
		return "", errInstanceRecovery
	}
	state, err := b.observe(ctx, id)
	if err != nil || state.State != "stopped" {
		return "", errInstanceBusy
	}
	if _, err := b.restoreJournal(id); !errors.Is(err, os.ErrNotExist) {
		return "", errInstanceRecovery
	}
	probe, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	container, err := b.command(probe, "podman", "ps", "--all", "--filter", "name=^minecraft-"+id+"$", "--format", "{{.Names}}")
	if err != nil || strings.TrimSpace(string(container)) != "" {
		return "", errInstanceRecovery
	}
	// Query retained history across boots. A failed or truncated query cannot
	// establish the absence of Minecraft startup or world-generation markers.
	history, err := b.command(probe, "journalctl", "--no-pager", "--quiet", "--output=cat", "--unit", "minecraft-"+id+".service")
	if err != nil || len(history) >= 64<<10 || switchStartupHistory.Match(history) {
		return "", errInstanceRecovery
	}
	out, err := instanceBackupStorageCommand(ctx, "nsenter", "--mount=/proc/1/ns/mnt", "--", instanceRestoreHelper, "switch-initial-data", id, "", "", "false")
	var evidence struct {
		Fingerprint string `json:"fingerprint"`
	}
	if err != nil || len(out) > 256 || instanceDecode(out, &evidence) != nil || !instanceSHA256Pattern.MatchString(evidence.Fingerprint) {
		return "", errInstanceRecovery
	}
	state, err = b.observeRaw(ctx, id)
	if err != nil || state.State != "stopped" {
		return "", errInstanceBusy
	}
	return evidence.Fingerprint, nil
}

func (b *instanceRuntimeBackend) requireSwitchInitialization(ctx context.Context, source instanceRuntimeRecord, p minecraftSwitchPlan) error {
	current, err := b.readRecord(source.Runtime.ID)
	if err == nil && current.Runtime == source.Runtime && current.Storage == source.Storage {
		proof, proofErr := b.switchInitializationEvidence(ctx, current)
		if proofErr == nil && proof == p.InitializationFingerprint {
			return nil
		}
	}
	return instanceRuntimeFailure("The source initialization evidence changed. Review Switch again.")
}

func verifySwitchInitializedWorld(data string) error {
	root, err := os.OpenRoot(data)
	if err != nil {
		return errInstanceRecovery
	}
	defer root.Close()
	level, err := switchLevelName(root)
	if err != nil {
		return errInstanceRecovery
	}
	info, err := switchWorldInfo(root, level+"/level.dat")
	if err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
		return errInstanceRecovery
	}
	return nil
}
