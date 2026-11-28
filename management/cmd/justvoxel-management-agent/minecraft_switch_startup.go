package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

var switchContainerID = regexp.MustCompile(`^[a-f0-9]{64}$`)
var switchStartupDone = regexp.MustCompile(`^Done \([0-9.]+s\)! For help`)
var switchStartupClock = regexp.MustCompile(`^\[[0-9]{2}:[0-9]{2}:[0-9]{2}(?: (?:INFO|WARN|ERROR)\]|\] \[[A-Za-z0-9 ._#-]+/(?:INFO|WARN|ERROR)\]): \s*`)

const (
	switchStartupWindow        = 15 * time.Minute
	switchStartupLogLines      = 256
	switchMigrationLogger      = "[WorldFolderMigration] "
	switchMigrationInstruction = "Type 'confirm' to continue with startup world migration, or 'cancel' to stop the server."
	switchMigrationReminder    = "World migration is still waiting for confirmation. Type 'confirm' to continue, or 'cancel' to stop the server."
)

type switchStartupSession struct {
	id      string
	started time.Time
}

func (s switchStartupSession) equal(other switchStartupSession) bool {
	return s.id == other.id && s.started.Equal(other.started)
}

func switchLeafStartupMigration(software, version string) bool {
	return software == "leaf" && java25GameReleasePattern.MatchString(version)
}

func (b *instanceRuntimeBackend) switchStartupSession(ctx context.Context, record instanceRuntimeRecord) (switchStartupSession, error) {
	name := "minecraft-" + record.Runtime.ID
	raw, err := b.command(ctx, "podman", "container", "inspect", name)
	var containers []struct {
		ID    string `json:"Id"`
		Name  string
		Image string
		State struct {
			Running   bool
			StartedAt time.Time
		}
		Mounts []struct {
			Destination string
			Type        string
			Source      string
		}
	}
	if err != nil || json.Unmarshal(raw, &containers) != nil || len(containers) != 1 {
		return switchStartupSession{}, errInstanceRecovery
	}
	container := containers[0]
	started := container.State.StartedAt
	if !switchContainerID.MatchString(container.ID) || strings.TrimPrefix(container.Name, "/") != name ||
		!container.State.Running || started.IsZero() || started.After(time.Now()) {
		return switchStartupSession{}, errInstanceRecovery
	}
	dataMounted := false
	for _, mount := range container.Mounts {
		if mount.Destination != "/data" {
			continue
		}
		if dataMounted || mount.Type != "bind" || mount.Source != record.Storage.DataPath {
			return switchStartupSession{}, errInstanceRecovery
		}
		dataMounted = true
	}
	if !dataMounted || container.Image == "" {
		return switchStartupSession{}, errInstanceRecovery
	}
	image, err := b.command(ctx, "podman", "image", "inspect", instanceRuntimeImage(record.Runtime), "--format", "{{.Id}}")
	if err != nil || strings.TrimSpace(string(image)) != container.Image {
		return switchStartupSession{}, errInstanceRecovery
	}
	return switchStartupSession{container.ID, started}, nil
}

type switchStartupLog struct {
	prompt, ready, starting bool
}

// The upstream Leaf prompt identifies the console input wait. Legacy runs
// can flood the journal with the exact EOF reminder, so a tagged reminder
// remains useful even when the initial alert has aged out of the log tail.
func parseSwitchStartupLog(raw []byte, started time.Time, version string) (switchStartupLog, error) {
	var result switchStartupLog
	// pluginInstallCommand truncates at 64 KiB. A partial final line cannot
	// establish that a previously observed prompt is still pending.
	if len(raw) >= 64<<10 {
		return result, errInstanceRecovery
	}
	log := strings.TrimRight(string(raw), "\n")
	complete := strings.Count(log, "\n") < switchStartupLogLines-1
	alerted := false
	for _, line := range strings.Split(log, "\n") {
		if line == "" {
			continue
		}
		stamp, message, ok := strings.Cut(line, " ")
		at, err := time.Parse(time.RFC3339Nano, stamp)
		if !ok || err != nil {
			return switchStartupLog{}, errInstanceRecovery
		}
		if at.Before(started) {
			continue
		}
		message = strings.TrimSpace(message)
		if strings.HasPrefix(message, "[init] ") {
			result.starting = true
		}
		prefix := switchStartupClock.FindStringIndex(message)
		if prefix == nil {
			continue
		}
		message = strings.TrimSpace(message[prefix[1]:])
		if switchStartupDone.MatchString(message) {
			result.ready = true
		}
		message, tagged := strings.CutPrefix(message, switchMigrationLogger)
		switch message {
		case "Starting minecraft server version " + version:
			result.starting = true
		case "World storage migration is required during startup.":
			alerted = true
		case switchMigrationInstruction:
			result.prompt = tagged || alerted && complete
		case switchMigrationReminder:
			result.prompt = tagged || alerted && complete
		case "Continuing with startup world migration.", "World migration cancelled by console input.":
			result.prompt, alerted = false, false
		}
	}
	result.starting = result.starting && complete
	result.prompt = result.prompt && !result.ready
	return result, nil
}

func switchStartupLogs(ctx context.Context, session switchStartupSession, version string) (switchStartupLog, error) {
	raw, err := pluginInstallCommand(ctx, "podman", "logs", "--timestamps", "--since", session.started.Format(time.RFC3339Nano), "--tail", strconv.Itoa(switchStartupLogLines), session.id)
	if err != nil {
		return switchStartupLog{}, errInstanceRecovery
	}
	return parseSwitchStartupLog(raw, session.started, version)
}

func (b *instanceRuntimeBackend) sendSwitchStartupAnswer(ctx context.Context, record instanceRuntimeRecord, session switchStartupSession, confirm bool) error {
	command, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	current, err := b.switchStartupSession(command, record)
	if err != nil || !current.equal(session) {
		return errInstanceRecovery
	}
	log, err := switchStartupLogs(command, session, record.Runtime.Version)
	if err != nil || !log.prompt {
		return errInstanceRecovery
	}
	answer := "cancel"
	if confirm {
		answer = "confirm"
	}
	// Bound the writer inside the container as well as the Podman client.
	_, err = b.command(command, "podman", "exec", "--user", strconv.FormatUint(uint64(record.Runtime.UID), 10), session.id,
		"timeout", "--kill-after=1s", "2s", "mc-send-to-console", answer)
	if err != nil {
		return errInstanceRecovery
	}
	return nil
}

func switchStartupAuthorized(j operationJournal, e minecraftSwitchEvidence, p minecraftSwitchPlan) bool {
	return j.OperationID == e.OperationID && j.InstanceID == e.Source && j.OperationType == operationTypeMinecraftSwitch &&
		j.State == operationRunning && j.Stage == "converting" && p.InstanceID == e.Source && p.Target == e.Target && p.Version == e.Version &&
		operationFingerprintPattern.MatchString(p.Fingerprint) &&
		j.PlanFingerprint == switchActionFingerprint(minecraftSwitchRequest{Fingerprint: p.Fingerprint, Action: e.Action, Target: e.Target})
}

// Only the live reviewed worker calls this after verifying its Restore or cold
// backup. A recovery worker has no path to the confirmation branch.
func (s *server) confirmSwitchStartupMigration(ctx context.Context, e minecraftSwitchEvidence, p minecraftSwitchPlan, record instanceRuntimeRecord, archive string, launched time.Time) (switchStartupSession, error) {
	if !switchLeafStartupMigration(p.Target, p.Version) {
		return switchStartupSession{}, nil
	}
	b := instanceRuntimes
	failure := errors.Join(errInstanceRecovery, &pluginRuntimeVerificationError{switchStatus: switchVerificationSession})
	if s.operations == nil || launched.IsZero() || !validInstanceRestoreArchive(archive) {
		return switchStartupSession{}, failure
	}
	j, err := s.operations.get(e.OperationID)
	if err != nil || !switchStartupAuthorized(j, e, p) {
		return switchStartupSession{}, failure
	}
	saved, err := readSwitchEvidence(j)
	if err != nil || saved.Stage != e.Stage || saved.Action != e.Action || saved.Target != e.Target || saved.Version != e.Version ||
		saved.SourceRecord.Runtime.Software != p.Current {
		return switchStartupSession{}, failure
	}
	id := e.Source
	switch e.Action {
	case "clone":
		id = e.CloneID
		if !p.Clone || saved.CloneID != id || saved.Stage != "verifying_clone" || !validSwitchClonePlacement(saved) ||
			record.Runtime != saved.CloneRecord.Runtime || record.Storage != saved.CloneRecord.Storage ||
			b.verifySwitchCloneStartupRestore(ctx, record, archive) != nil {
			return switchStartupSession{}, failure
		}
	case "convert":
		if !p.Convert || saved.Stage != "converting_world" || !saved.WorldChanged || saved.ColdArchive != archive ||
			record.Storage != saved.SourceRecord.Storage {
			return switchStartupSession{}, failure
		}
	default:
		return switchStartupSession{}, failure
	}
	current, err := b.readRecord(id)
	if err != nil || current.Runtime != record.Runtime || current.Storage != record.Storage || current.NetworkMember ||
		current.Runtime.ID != id || current.Runtime.Software != p.Target || current.Runtime.Version != p.Version || current.Runtime.VersionPolicy != "pinned" ||
		current.Runtime.ImageDigest != p.RuntimeDigest || current.Runtime.ImageTag != p.Runtime.ImageTag || !current.Runtime.PreserveContent ||
		b.verifyLive(current) != nil || b.storage(ctx, "validate", current) != nil {
		return switchStartupSession{}, failure
	}
	return b.waitForSwitchStartupMigration(ctx, current, launched)
}

func (b *instanceRuntimeBackend) waitForSwitchStartupMigration(ctx context.Context, record instanceRuntimeRecord, launched time.Time) (switchStartupSession, error) {
	failure := errors.Join(errInstanceRecovery, &pluginRuntimeVerificationError{switchStatus: switchVerificationSession})
	wait, cancel := context.WithTimeout(ctx, switchStartupWindow)
	defer cancel()
	var bound switchStartupSession
	for wait.Err() == nil {
		probe, cancelProbe := context.WithTimeout(wait, 5*time.Second)
		session, err := b.switchStartupSession(probe, record)
		if err == nil {
			if session.started.Before(launched) || bound.id != "" && !bound.equal(session) {
				cancelProbe()
				return switchStartupSession{}, failure
			}
			bound = session
			log, logErr := switchStartupLogs(probe, session, record.Runtime.Version)
			if logErr == nil && log.prompt {
				cancelProbe()
				// Return after the first attempt, including an uncertain delivery.
				if err := b.sendSwitchStartupAnswer(wait, record, session, true); err != nil {
					return session, errors.Join(failure, err)
				}
				return session, nil
			}
			// A ready JVM can proceed to the existing verifier even if the
			// early log is unavailable or uses another readiness format.
			ready := logErr == nil && log.ready
			if !ready {
				_, playersErr := b.switchStartupPlayers(probe, record, session)
				ready = playersErr == nil
			}
			cancelProbe()
			if ready {
				return session, nil
			}
		} else {
			cancelProbe()
		}
		if b.waitMaintenance(wait, time.Second) != nil {
			break
		}
	}
	return switchStartupSession{}, failure
}

func (b *instanceRuntimeBackend) verifySwitchStartupSession(ctx context.Context, record instanceRuntimeRecord, expected switchStartupSession) error {
	if expected.id == "" {
		return nil
	}
	probe, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	current, err := b.switchStartupSession(probe, record)
	if err != nil || !current.equal(expected) {
		return errors.Join(errInstanceRecovery, &pluginRuntimeVerificationError{switchStatus: switchVerificationSession})
	}
	return nil
}

func (b *instanceRuntimeBackend) verifySwitchCloneStartupRestore(ctx context.Context, record instanceRuntimeRecord, archive string) error {
	id := record.Runtime.ID
	j, err := b.restoreJournal(id)
	if err != nil || j.Stage != "completed" || j.OriginallyRunning || len(j.Parent) != 2 || len(j.Replacement) != 2 ||
		!instanceSHA256Pattern.MatchString(j.Fingerprint) ||
		(archive != "" && j.Archive != archive) || !validInstanceRestoreArchive(j.Archive) {
		return errInstanceRecovery
	}
	raw, err := b.readPrivate(filepath.Join(b.root, id, "config.json"))
	if err != nil || instanceHash(raw) != j.ConfigSHA256 {
		return errInstanceRecovery
	}
	proof := struct {
		Parent          []uint64 `json:"parent"`
		Data            string   `json:"data"`
		DataMayBeAbsent bool     `json:"data_may_be_absent"`
	}{j.Parent, fmt.Sprintf("%d:%d", j.Replacement[0], j.Replacement[1]), false}
	_, err = b.instanceDeletionStorage(ctx, "matrix-retry", instanceDeleteJournal{ID: id, Record: record, DataIdentity: string(mustSwitchJSON(proof))})
	return err
}

// The failed-start exception is available only while recovering a durable
// Switch operation. Normal stops still use the ordinary player check.
func (s *server) stopFailedSwitchStartup(ctx context.Context, e minecraftSwitchEvidence, record instanceRuntimeRecord) error {
	b, id := instanceRuntimes, record.Runtime.ID
	probe, cancel := context.WithTimeout(ctx, 5*time.Second)
	_, playersErr := b.players(probe, id)
	cancel()
	if playersErr == nil {
		return b.actionWithWarnings(ctx, id, "stop", true)
	}
	if s.operations == nil {
		return errInstanceRecovery
	}
	j, err := s.operations.get(e.OperationID)
	if err != nil || j.OperationType != operationTypeMinecraftSwitch || j.InstanceID != e.Source ||
		(j.State != operationNeedsAttention && (j.State != operationRunning || j.Stage != "converting")) {
		return errInstanceRecovery
	}
	saved, err := readSwitchEvidence(j)
	if err != nil || saved.Action != e.Action || saved.Target != e.Target || saved.Version != e.Version ||
		record.Runtime.Software != e.Target || record.Runtime.Version != e.Version || record.Runtime.VersionPolicy != "pinned" ||
		record.NetworkMember || b.verifyLive(record) != nil || b.storage(ctx, "validate", record) != nil {
		return errInstanceRecovery
	}
	switch saved.Action {
	case "clone":
		if saved.Stage != "verifying_clone" && saved.Stage != "complete" ||
			!validSwitchClonePlacement(saved) || id != saved.CloneID || !switchCloneRecordMatches(record, *saved.CloneRecord) ||
			b.verifySwitchCloneStartupRestore(ctx, record, "") != nil {
			return errInstanceRecovery
		}
		expected := saved.CloneRecord.Runtime
		// Older Paper-family workers could clear only this startup guard.
		if !record.Runtime.PreserveContent && javaServerPaperFamily(saved.SourceRecord.Runtime.Software) && javaServerPaperFamily(saved.Target) {
			expected.PreserveContent = false
		}
		if record.Runtime != expected {
			return errInstanceRecovery
		}
		if j.MatrixParent != "" || record.Runtime.MatrixFixture {
			s.operations.mu.Lock()
			parent, parentErr := s.operations.switchMatrixParent(j.MatrixParent, "switch", true)
			s.operations.mu.Unlock()
			if parentErr != nil || !slices.Contains(parent.MatrixFixtures, id) {
				return errInstanceRecovery
			}
			receipt, receiptErr := readSwitchMatrixFixture(j.MatrixParent, id)
			if receiptErr != nil || receipt.checkRecord(record) != nil ||
				(receipt.Stage == "reserved" && receipt.CandidateSHA256 != instanceHash(mustSwitchJSON(saved.CloneRecord.Runtime))) ||
				(receipt.Stage != "reserved" && receipt.Stage != "owned" && receipt.Stage != "binding") {
				return errInstanceRecovery
			}
			if receipt.Stage != "reserved" && receipt.checkData(ctx, record) != nil {
				return errInstanceRecovery
			}
		}
	case "convert":
		if id != saved.Source || !saved.WorldChanged || saved.Stage != "converting_world" ||
			!switchLeafStartupMigration(saved.Target, saved.Version) || !record.Runtime.PreserveContent ||
			record.Storage != saved.SourceRecord.Storage || record.Runtime.UID != saved.SourceRecord.Runtime.UID ||
			record.Runtime.GID != saved.SourceRecord.Runtime.GID {
			return errInstanceRecovery
		}
		// Convert persists WorldChanged only after validating the cold archive
		// against the source definition. Restore revalidates it after the
		// original definition has been republished.
	default:
		return errInstanceRecovery
	}
	return b.stopVerifiedSwitchStartup(ctx, record)
}

func (b *instanceRuntimeBackend) switchStartupPlayers(ctx context.Context, record instanceRuntimeRecord, session switchStartupSession) (instancePlayers, error) {
	out, err := b.command(ctx, "podman", "exec", session.id, "timeout", "--kill-after=1s", "2s", "rcon-cli", "list")
	if err != nil {
		return instancePlayers{}, err
	}
	return parseInstancePlayers(record.Runtime.ID, out)
}

func (b *instanceRuntimeBackend) stopVerifiedSwitchStartup(ctx context.Context, record instanceRuntimeRecord) error {
	id := record.Runtime.ID
	probe, cancel := context.WithTimeout(ctx, 10*time.Second)
	session, err := b.switchStartupSession(probe, record)
	if err != nil {
		cancel()
		return err
	}
	log, logErr := switchStartupLogs(probe, session, record.Runtime.Version)
	_, playersErr := b.switchStartupPlayers(probe, record, session)
	cancel()
	if playersErr == nil || log.ready {
		return b.actionWithWarnings(ctx, id, "stop", true)
	}
	if logErr != nil || !log.starting && !log.prompt {
		return errInstanceRecovery
	}

	stop, cancelStop := context.WithTimeout(ctx, 4*time.Minute+15*time.Second)
	defer cancelStop()
	// Recheck the session and readiness immediately before the exceptional
	// stop. A server that reached readiness goes through the usual warnings.
	if err := b.verifySwitchStartupSession(stop, record, session); err != nil {
		return err
	}
	probe, cancel = context.WithTimeout(stop, 5*time.Second)
	log, logErr = switchStartupLogs(probe, session, record.Runtime.Version)
	_, playersErr = b.switchStartupPlayers(probe, record, session)
	probeErr := probe.Err()
	cancel()
	if logErr != nil || !log.starting && !log.prompt && !log.ready {
		return errInstanceRecovery
	}
	if log.ready || playersErr == nil {
		return b.actionWithWarnings(ctx, id, "stop", true)
	}
	if probeErr != nil {
		return errInstanceRecovery
	}
	cancelled := false
	if log.prompt && switchLeafStartupMigration(record.Runtime.Software, record.Runtime.Version) {
		cancelled = b.sendSwitchStartupAnswer(stop, record, session, false) == nil
	}
	if !cancelled {
		// A legacy clone may have no input pipe. Recheck that the fallback
		// still applies to the same session before queuing the service stop.
		if err := b.verifySwitchStartupSession(stop, record, session); err != nil {
			state, stateErr := b.observeRaw(stop, id)
			if stateErr == nil && state.State == "stopped" {
				return nil
			}
			return err
		}
	}
	// Also queue Stop after a delivered cancel to suppress service restart.
	if _, err := b.command(stop, "systemctl", "--no-block", "stop", "--", "minecraft-"+id+".service"); err != nil {
		return err
	}
	for {
		state, err := b.observeRaw(stop, id)
		if err != nil {
			return err
		}
		switch state.State {
		case "stopped":
			return nil
		case "failed":
			if err := b.normalizeStoppedFailure(stop, state); err != nil {
				return err
			}
		}
		if err := b.waitMaintenance(stop, time.Second); err != nil {
			return err
		}
	}
}
