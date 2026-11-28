package main

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

const switchTestMigrationPrompt = "[WorldFolderMigration] World storage migration is required during startup.\n" +
	"[WorldFolderMigration] Type 'confirm' to continue with startup world migration, or 'cancel' to stop the server."

func switchTestSessionLog(at time.Time, messages string) []byte {
	var lines []string
	for _, message := range strings.Split(messages, "\n") {
		lines = append(lines, at.Format(time.RFC3339Nano)+" [12:00:00 WARN]: "+message)
	}
	return []byte(strings.Join(lines, "\n") + "\n")
}

func switchTestSessionInspect(id, name string, started time.Time, path string) []byte {
	return fmt.Appendf(nil, `[{"Id":%q,"Name":%q,"Image":"image","State":{"Running":true,"StartedAt":%q},"Mounts":[{"Destination":"/data","Type":"bind","Source":%q}]}]`,
		id, name, started.Format(time.RFC3339Nano), path)
}

func TestSwitchStartupPromptSession(t *testing.T) {
	start := time.Now()
	after := start.Add(time.Second)
	for _, tt := range []struct {
		name  string
		raw   []byte
		want  switchStartupLog
		error bool
	}{
		{"pipe_wait", switchTestSessionLog(after, switchTestMigrationPrompt), switchStartupLog{prompt: true}, false},
		{"legacy_eof_tail", switchTestSessionLog(after, strings.Repeat(switchMigrationLogger+switchMigrationReminder+"\n", switchStartupLogLines-1)+switchMigrationLogger+switchMigrationReminder), switchStartupLog{prompt: true}, false},
		{"truncated_startup", switchTestSessionLog(after, "Starting minecraft server version 26.3"+strings.Repeat("\nLoading library", switchStartupLogLines-1)), switchStartupLog{}, false},
		{"already_confirmed", switchTestSessionLog(after, switchTestMigrationPrompt+"\n[WorldFolderMigration] Continuing with startup world migration."), switchStartupLog{}, false},
		{"stale", switchTestSessionLog(start.Add(-time.Second), switchTestMigrationPrompt), switchStartupLog{}, false},
		{"partial", switchTestSessionLog(after, "World migration is still waiting for confirmation."), switchStartupLog{}, false},
		{"unrelated", switchTestSessionLog(after, "Type confirm to change a plugin setting."), switchStartupLog{}, false},
		{"ready", switchTestSessionLog(after, switchTestMigrationPrompt+"\nDone (2.0s)! For help, type \"help\""), switchStartupLog{ready: true}, false},
		{"startup", switchTestSessionLog(after, "Starting minecraft server version 26.3"), switchStartupLog{starting: true}, false},
		{"unbound", []byte(switchTestMigrationPrompt), switchStartupLog{}, true},
		{"truncated", []byte(strings.Repeat("x", 64<<10)), switchStartupLog{}, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSwitchStartupLog(tt.raw, start, "26.3")
			if got != tt.want || (err != nil) != tt.error {
				t.Fatalf("startup evidence = %+v, err = %v", got, err)
			}
		})
	}
}

func TestSwitchStartupConfirmationRequiresLiveReview(t *testing.T) {
	e := minecraftSwitchEvidence{OperationID: "child", Source: "jv-a1b2c3", Action: "clone", Target: "leaf", Version: "26.3"}
	p := minecraftSwitchPlan{InstanceID: e.Source, Target: e.Target, Version: e.Version, Fingerprint: testMinecraftResetFingerprint, Clone: true}
	j := operationJournal{OperationID: e.OperationID, InstanceID: e.Source, OperationType: operationTypeMinecraftSwitch,
		State: operationRunning, Stage: "converting",
		PlanFingerprint: switchActionFingerprint(minecraftSwitchRequest{Action: e.Action, Target: e.Target, Fingerprint: p.Fingerprint})}
	if !switchStartupAuthorized(j, e, p) {
		t.Fatal("the reviewed live Switch did not authorize startup")
	}
	for _, change := range []func(*operationJournal){
		func(j *operationJournal) { j.State = operationNeedsAttention },
		func(j *operationJournal) { j.Stage = "queued" },
		func(j *operationJournal) { j.InstanceID = "jv-other1" },
		func(j *operationJournal) { j.PlanFingerprint = testMinecraftResetFingerprint },
	} {
		foreign := j
		change(&foreign)
		if switchStartupAuthorized(foreign, e, p) {
			t.Fatal("startup accepted missing or foreign review authority")
		}
	}
	s := &server{}
	if _, err := s.confirmSwitchStartupMigration(context.Background(), e, p, instanceRuntimeRecord{}, "Survival-backup-vanilla-2026-03-06-0015.tar.gz", time.Now()); err == nil {
		t.Fatal("startup accepted missing operation evidence")
	}
}

func TestSwitchStartupCancelUsesOnlyCurrentContainer(t *testing.T) {
	f := newInstanceActivationFixture(t)
	b := f.backend
	record, err := b.readRecord(f.configs[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	record.Runtime.Software, record.Runtime.Version = "leaf", "26.45.7"
	id := record.Runtime.ID
	f.states[id], f.containers[id] = "active", "running"
	session := switchStartupSession{strings.Repeat("a", 64), time.Now().In(time.FixedZone("", 5*3600+45*60)).Add(-time.Second)}
	oldCommand, oldLogs := b.command, pluginInstallCommand
	t.Cleanup(func() { pluginInstallCommand = oldLogs })
	pluginInstallCommand = func(_ context.Context, command string, args ...string) ([]byte, error) {
		if command != "podman" || !slices.Equal(args, []string{"logs", "--timestamps", "--since", session.started.Format(time.RFC3339Nano), "--tail", "256", session.id}) {
			t.Fatal("logs queried for the wrong session", args)
		}
		return switchTestSessionLog(session.started, switchTestMigrationPrompt), nil
	}
	var inspection []byte
	var answerErr error
	var answers []string
	b.command = func(ctx context.Context, command string, args ...string) ([]byte, error) {
		if command == "podman" && slices.Equal(args, []string{"container", "inspect", "minecraft-" + id}) {
			if inspection != nil {
				return inspection, nil
			}
			return switchTestSessionInspect(session.id, args[2], session.started, record.Storage.DataPath), nil
		}
		if command == "podman" && slices.Contains(args, "image") {
			return []byte("image"), nil
		}
		if command == "podman" && slices.Contains(args, "mc-send-to-console") {
			deadline, ok := ctx.Deadline()
			left := time.Until(deadline)
			if !ok || left <= 0 || left > 5*time.Second || args[3] != session.id ||
				!slices.Equal(args[4:8], []string{"timeout", "--kill-after=1s", "2s", "mc-send-to-console"}) {
				t.Fatal("unsafe console command", args)
			}
			answers = append(answers, args[len(args)-1])
			return nil, answerErr
		}
		if command == "podman" && slices.Contains(args, "rcon-cli") {
			return nil, errors.New("RCON unavailable before startup")
		}
		return oldCommand(ctx, command, args...)
	}
	ctx := context.Background()
	for _, rejected := range [][]byte{
		switchTestSessionInspect(session.id, "minecraft-jv-other1", session.started, record.Storage.DataPath),
		switchTestSessionInspect(session.id, "minecraft-"+id, session.started, "/foreign"),
		switchTestSessionInspect(session.id, "minecraft-"+id, time.Time{}, record.Storage.DataPath),
		append(switchTestSessionInspect(session.id, "minecraft-"+id, session.started, record.Storage.DataPath), []byte(" []")...),
	} {
		inspection = rejected
		if err = b.sendSwitchStartupAnswer(ctx, record, session, true); err == nil || len(answers) != 0 {
			t.Fatal("invalid session inspection authorized confirmation", err, answers)
		}
	}
	inspection = nil
	if _, err = b.waitForSwitchStartupMigration(ctx, record, session.started.Add(time.Nanosecond)); err == nil || len(answers) != 0 {
		t.Fatal("a session predating this Switch authorized confirmation", err, answers)
	}
	answerErr = context.DeadlineExceeded
	if _, err = b.waitForSwitchStartupMigration(ctx, record, session.started); err == nil || !slices.Equal(answers, []string{"confirm"}) {
		t.Fatal("uncertain confirmation was repeated or accepted as verified", err, answers)
	}
	answers, answerErr = nil, nil
	if err = b.stopVerifiedSwitchStartup(ctx, record); err != nil || !slices.Equal(answers, []string{"cancel"}) {
		t.Fatal("recovery did not send one bounded cancel", err, answers)
	}
}

func TestSwitchCloneFailedStartRecoveryRetainsRestoreOwnership(t *testing.T) {
	fx := newSwitchRetirementFixture(t, false)
	b, id := fx.f.backend, fx.e.CloneID
	record := *fx.e.CloneRecord
	path := filepath.Join(b.root, id, "restore-journal.json")
	hash, err := switchMatrixConfigHash(id)
	if err != nil {
		t.Fatal(err)
	}
	restore := instanceRestoreJournal{Schema: 1, ID: id, Stage: "completed", ConfigSHA256: hash,
		Transaction: strings.TrimPrefix(filepath.Base(fx.tx), ".restore-"),
		Parent:      switchRetirementInode(t, fx.parent), Original: switchRetirementInode(t, filepath.Join(fx.tx, "original")),
		Replacement: switchRetirementInode(t, record.Storage.DataPath), Archive: "Survival-backup-vanilla-2026-03-06-0015.tar.gz",
		Fingerprint: instanceHash([]byte("verified fixture archive"))}
	foreign := restore
	foreign.Replacement = slices.Clone(restore.Replacement)
	foreign.Replacement[1]++
	if err = b.writeJSON(path, foreign, false); err != nil {
		t.Fatal(err)
	}
	fx.f.states[id], fx.f.containers[id], fx.f.failPlayers = "active", "running", true
	started, container := time.Now().UTC().Add(-time.Second), strings.Repeat("b", 64)
	base, oldLogs := b.command, pluginInstallCommand
	t.Cleanup(func() { pluginInstallCommand = oldLogs })
	pluginInstallCommand = func(_ context.Context, command string, args ...string) ([]byte, error) {
		if command != "podman" || args[0] != "logs" || args[len(args)-1] != container {
			t.Fatal("wrong startup logs requested", command, args)
		}
		return switchTestSessionLog(started, "Starting minecraft server version "+record.Runtime.Version), nil
	}
	stops := 0
	b.command = func(ctx context.Context, command string, args ...string) ([]byte, error) {
		switch {
		case command == "podman" && slices.Equal(args, []string{"container", "inspect", "minecraft-" + id}):
			return switchTestSessionInspect(container, args[2], started, record.Storage.DataPath), nil
		case command == "podman" && slices.Contains(args, "image"):
			return []byte("image"), nil
		case command == "podman" && args[0] == "exec" && slices.Contains(args, "timeout"):
			return nil, errors.New("no RCON listener")
		case command == "systemctl" && slices.Equal(args, []string{"--no-block", "stop", "--", "minecraft-" + id + ".service"}):
			stops++
			if stops == 1 {
				return nil, errors.New("failed to enqueue stop")
			}
		}
		return base(ctx, command, args...)
	}
	fx.assertBlocked(t, "clone runtime retirement incomplete")
	if stops != 0 {
		t.Fatal("recovery stopped a foreign Restore inode")
	}
	if err = b.writeJSON(path, restore, true); err != nil {
		t.Fatal(err)
	}
	fx.assertBlocked(t, "clone runtime retirement incomplete")
	fx.assertRecovered(t)
	if stops != 2 {
		t.Fatalf("expected one retried stop, got %d", stops)
	}
}

func TestSwitchReadyCloneKeepsPlayerStopProtection(t *testing.T) {
	f := newInstanceActivationFixture(t)
	b := f.backend
	old := instanceRuntimes
	instanceRuntimes = b
	t.Cleanup(func() { instanceRuntimes = old })
	id := f.configs[0].ID
	if err := f.apply(id); err != nil {
		t.Fatal(err)
	}
	record, err := b.readRecord(id)
	if err != nil {
		t.Fatal(err)
	}
	f.states[id], f.containers[id] = "active", "running"
	base := b.command
	lists := 0
	b.command = func(ctx context.Context, command string, args ...string) ([]byte, error) {
		if command == "podman" && slices.Equal(args, []string{"exec", "minecraft-" + id, "rcon-cli", "list"}) {
			lists++
			if lists == 1 {
				return []byte("There are 1 of a max of 20 players online: Alex"), nil
			}
			return []byte("indeterminate player list"), nil
		}
		if (command == "systemctl" && slices.Contains(args, "stop")) ||
			(command == "podman" && (args[0] == "kill" || slices.Contains(args, "mc-send-to-console"))) {
			t.Fatal("ready clone bypassed ordinary shutdown protection", command, args)
		}
		return base(ctx, command, args...)
	}
	s := &server{}
	if err = s.stopFailedSwitchStartup(context.Background(), minecraftSwitchEvidence{}, record); err == nil || lists != 2 {
		t.Fatal("ready clone did not recheck players using normal shutdown", err, lists)
	}
}

func TestSwitchFailedStartupAcceptsReservedMatrixCloneWithCompletedRestore(t *testing.T) {
	fx := newSwitchRetirementFixture(t, false)
	b, id := fx.f.backend, fx.e.CloneID
	o := matrixTestStore(t, b)
	parent := matrixTestBegin(t, o, "switch")
	parent.MatrixFixtures = []string{fx.e.Source, id}
	if err := o.persist(parent); err != nil {
		t.Fatal(err)
	}
	o.operations[parent.OperationID] = parent
	child, err := o.beginSwitchMatrixChild(parent.OperationID, fx.e.Source, testMinecraftResetFingerprint)
	if err != nil {
		t.Fatal(err)
	}
	child, err = o.transition(child.OperationID, operationNeedsAttention, "switch_failed", "Switch startup failed.")
	if err != nil {
		t.Fatal(err)
	}
	record := *fx.e.CloneRecord
	backupRoot := t.TempDir()
	record.Backups = &instanceRuntimeBackupStorage{Type: "system", Root: backupRoot, Path: filepath.Join(backupRoot, id)}
	if err = b.writeJSON(filepath.Join(b.root, id, "config.json"), record, true); err != nil {
		t.Fatal(err)
	}
	e := fx.e
	e.OperationID, e.CloneRecord = child.OperationID, &record
	if err = saveSwitchEvidence(e); err != nil {
		t.Fatal(err)
	}
	receipt := switchMatrixFixture{
		Parent: parent.OperationID, ID: id, Stage: "reserved", Storage: record.Storage, Placement: record.Backups,
		UID: record.Runtime.UID, GID: record.Runtime.GID, CandidateSHA256: instanceHash(mustSwitchJSON(record.Runtime)),
	}
	if err = receipt.save(false); err != nil {
		t.Fatal(err)
	}
	hash, err := switchMatrixConfigHash(id)
	if err != nil {
		t.Fatal(err)
	}
	restore := instanceRestoreJournal{Schema: 1, ID: id, Stage: "completed", ConfigSHA256: hash,
		Transaction: strings.TrimPrefix(filepath.Base(fx.tx), ".restore-"),
		Parent:      switchRetirementInode(t, fx.parent), Original: switchRetirementInode(t, filepath.Join(fx.tx, "original")),
		Replacement: switchRetirementInode(t, record.Storage.DataPath), Archive: "Survival-backup-vanilla-2026-03-06-0015.tar.gz",
		Fingerprint: instanceHash([]byte("fixture archive"))}
	fx.f.states[id], fx.f.containers[id], fx.f.failPlayers = "active", "running", true
	started, container := time.Now().UTC().Add(-time.Second), strings.Repeat("c", 64)
	base, oldLogs := b.command, pluginInstallCommand
	t.Cleanup(func() { pluginInstallCommand = oldLogs })
	pluginInstallCommand = func(_ context.Context, command string, args ...string) ([]byte, error) {
		if command != "podman" || args[0] != "logs" || args[len(args)-1] != container {
			t.Fatal("wrong startup logs requested", command, args)
		}
		return switchTestSessionLog(started, "Starting minecraft server version "+record.Runtime.Version), nil
	}
	b.command = func(ctx context.Context, command string, args ...string) ([]byte, error) {
		switch {
		case command == "podman" && slices.Equal(args, []string{"container", "inspect", "minecraft-" + id}):
			return switchTestSessionInspect(container, args[2], started, record.Storage.DataPath), nil
		case command == "podman" && slices.Contains(args, "image"):
			return []byte("image"), nil
		case command == "podman" && args[0] == "exec" && slices.Contains(args, "timeout"):
			return nil, errors.New("no RCON listener")
		}
		return base(ctx, command, args...)
	}
	ctx := context.Background()
	s := &server{operations: o}
	if err = s.stopFailedSwitchStartup(ctx, e, record); err == nil || fx.f.states[id] != "active" {
		t.Fatal("reservation alone authorized the stop", err)
	}
	if err = b.writeJSON(filepath.Join(b.root, id, "restore-journal.json"), restore, false); err != nil {
		t.Fatal(err)
	}
	if err = s.stopFailedSwitchStartup(ctx, e, record); err != nil || fx.f.states[id] != "inactive" {
		t.Fatal("reserved clone with completed Restore could not be stopped", err)
	}
	if f, err := readSwitchMatrixFixture(parent.OperationID, id); err != nil || f.Stage != "reserved" {
		t.Fatal("stop discarded the retirement receipt", err)
	}
}
