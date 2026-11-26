package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func matrixTestStore(t *testing.T, backend *instanceRuntimeBackend) *operationStore {
	t.Helper()
	oldBackend, oldLock, oldExclusion := instanceRuntimes, pluginRuntimeLockPath, instanceRestoreExclusionPath
	instanceRuntimes = backend
	pluginRuntimeLockPath = filepath.Join(t.TempDir(), "runtime.lock")
	instanceRestoreExclusionPath = filepath.Join(t.TempDir(), "restore-exclusion.lock")
	t.Cleanup(func() {
		instanceRuntimes, pluginRuntimeLockPath, instanceRestoreExclusionPath = oldBackend, oldLock, oldExclusion
	})
	return openTestOperationStore(t)
}

func matrixTestBegin(t *testing.T, o *operationStore, stage string) operationJournal {
	t.Helper()
	j, err := o.beginSwitchMatrix("sha256:" + strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = o.transition(j.OperationID, operationValidating, "validating", "Validating matrix fixtures."); err != nil {
		t.Fatal(err)
	}
	j, err = o.transition(j.OperationID, operationRunning, stage, "Working on matrix fixtures.")
	if err != nil {
		t.Fatal(err)
	}
	return j
}

func TestSwitchMatrixOperationOwnership(t *testing.T) {
	f := newInstanceActivationFixture(t)
	o := matrixTestStore(t, f.backend)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// The shell exclusion lock and the operation-store Restore lock are
	// separate objects, and an already-running shell backup blocks admission.
	shell := exec.CommandContext(ctx, "bash", "-c", `
umask 077
exec 8>>"$1"
flock -n 8 || exit 1
echo ready
read -r release
`, "legacy", instanceRestoreExclusionPath)
	release := matrixTestHoldProcess(t, shell)
	fingerprint := "sha256:" + strings.Repeat("a", 64)
	if _, err := o.beginSwitchMatrix(fingerprint); err == nil {
		t.Fatal("matrix entered while a separate legacy process owned exclusion")
	}
	release()
	j := matrixTestBegin(t, o, "fixtures")
	if _, err := o.beginSwitchMatrix(fingerprint); err == nil {
		t.Fatal("duplicate matrix admission")
	}
	if _, _, err := o.beginSetup(fingerprint); err == nil {
		t.Fatal("Setup admitted while matrix owns operation reservations")
	}
	candidate := instanceRuntimeRecord{Runtime: f.configs[0]}
	if err := o.reserveSwitchMatrixFixture(j.OperationID, candidate, operationJournal{}); !errors.Is(err, errInstanceID) {
		t.Fatal("matrix acquired an existing user instance")
	}
	id := "jv-matrixown"
	snapshot := f.snapshot
	snapshot.BackupPlacement = &instanceRuntimeBackupStorage{Type: "system", Root: snapshot.BackupPath}
	candidate.Runtime.ID, candidate.Runtime.StartOnBoot = id, false
	candidate.Storage, candidate.Backups = instanceStorageFor(snapshot, id), instanceBackupsFor(snapshot, id)
	if err := o.reserveSwitchMatrixFixture(j.OperationID, candidate, operationJournal{}); err != nil {
		t.Fatal(err)
	}
	if _, err := o.beginSwitchMatrixChild(j.OperationID, id, fingerprint); err == nil {
		t.Fatal("Switch admitted at fixture creation stage")
	}
	if _, err := o.transition(j.OperationID, operationRunning, "switch", "Switching matrix fixture."); err != nil {
		t.Fatal(err)
	}
	if _, err := o.beginSwitchMatrixChild(j.OperationID, f.configs[0].ID, fingerprint); !errors.Is(err, errInstanceID) {
		t.Fatalf("foreign instance result: %v", err)
	}
	child, err := o.beginSwitchMatrixChild(j.OperationID, id, fingerprint)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = o.beginSwitchMatrixChild(j.OperationID, id, fingerprint); err == nil {
		t.Fatal("duplicate child admission")
	}
	if err = o.close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := openOperationStore(o.baseDir)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.close()
	interrupted := reopened.operations[child.OperationID]
	if interrupted.State != operationNeedsAttention || interrupted.Stage != "interrupted" ||
		interrupted.InterruptedAt == "" || interrupted.MatrixParent != j.OperationID {
		t.Fatalf("child interruption evidence was lost: %+v", interrupted)
	}
	if reopened.authorizeQueuedMatrixSwitch(child) {
		t.Fatal("interrupted child was authorized to rerun")
	}
	if _, err = reopened.transition(child.OperationID, operationResolved, "resolved", "Fixture recovery verified."); err != nil {
		t.Fatal(err)
	}
	if current, err := reopened.currentMinecraftMaintenance(); err != nil || current == nil || current.OperationID != j.OperationID {
		t.Fatalf("child terminal transition released parent ownership: %+v, %v", current, err)
	}
	if _, err = reopened.transition(j.OperationID, operationResolved, "resolved", "Matrix cleanup verified."); err == nil {
		t.Fatal("parent closed with unretired fixture")
	}
}

func matrixTestHoldProcess(t *testing.T, cmd *exec.Cmd) func() {
	t.Helper()
	in, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	waited := false
	t.Cleanup(func() {
		_ = in.Close()
		if !waited {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	})
	lines := bufio.NewScanner(out)
	if !lines.Scan() || lines.Text() != "ready" {
		t.Fatalf("separate flock process did not reach checkpoint: %q", lines.Text())
	}
	return func() {
		_, writeErr := fmt.Fprintln(in, "release")
		_ = in.Close()
		waitErr := cmd.Wait()
		waited = true
		if err := errors.Join(writeErr, waitErr); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSwitchMatrixStartupLockHandoff(t *testing.T) {
	f := newInstanceActivationFixture(t)
	o := matrixTestStore(t, f.backend)
	s := &server{operations: o}
	j := matrixTestBegin(t, o, "switch")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	checkBackups := matrixTestBackupEntrypoints(t, ctx, o)
	lock, err := acquirePluginMaintenanceLock()
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	if err = lock.startHandoff(func() error {
		startup := exec.CommandContext(ctx, "bash", "-c", `
exec 9>>"$1"
flock -n 9 || exit 1
echo ready
read -r release
`, "ExecStartPre", pluginRuntimeLockPath)
		release := matrixTestHoldProcess(t, startup)
		checkBackups()
		if _, err := s.lockInstanceBackup(); err == nil {
			t.Fatal("scheduled Agent backup entered the handoff")
		}
		if _, err := s.claimMinecraftMutation(); err == nil {
			t.Fatal("ordinary Agent mutation entered the handoff")
		}
		release()
		if exec.CommandContext(ctx, "flock", "-n", pluginRuntimeLockPath, "true").Run() != nil {
			t.Fatal("test never reached the open runtime flock interval")
		}
		checkBackups()
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err = lock.Close(); err != nil {
		t.Fatal(err)
	}
	// Simulate loss of the Agent with the on-disk parent evidence intact.
	if err = o.close(); err != nil {
		t.Fatal(err)
	}
	if exec.CommandContext(ctx, "flock", "-n", instanceRestoreExclusionPath, "true").Run() != nil {
		t.Fatal("exclusion descriptor did not close")
	}
	checkBackups()
	reopened, err := openOperationStore(o.baseDir)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.close()
	current, err := reopened.currentMinecraftMaintenance()
	if err != nil || current == nil || current.OperationID != j.OperationID ||
		current.State != operationNeedsAttention || current.Stage != "interrupted" || current.InterruptedAt == "" {
		t.Fatalf("interruption evidence was lost: %+v, %v", current, err)
	}
	checkBackups()
	if _, err = reopened.transition(j.OperationID, operationResolved, "resolved", "No fixture resources were created."); err != nil {
		t.Fatal(err)
	}
	if exec.CommandContext(ctx, "flock", "-n", instanceRestoreExclusionPath, "true").Run() != nil {
		t.Fatal("verified terminal parent left exclusion locked")
	}
}

func matrixTestBackupEntrypoints(t *testing.T, ctx context.Context, o *operationStore) func() {
	t.Helper()
	repo, dir := repositoryRoot(t), t.TempDir()
	copyScript := func(relative string, replacements ...string) string {
		t.Helper()
		source, err := os.ReadFile(filepath.Join(repo, relative))
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(dir, filepath.Base(relative))
		if err = os.WriteFile(path, []byte(strings.NewReplacer(replacements...).Replace(string(source))), 0600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	trace := filepath.Join(dir, "touched")
	stub := filepath.Join(dir, "backup-stub.sh")
	if err := os.WriteFile(stub, []byte("jv_backup_load_config() { touch '"+trace+"'; return 71; }\n"+
		"jv_backup_prepare_write_target() { touch '"+trace+"'; return 72; }\n"), 0600); err != nil {
		t.Fatal(err)
	}
	reservation := copyScript("mjust/libexec/switch-matrix-reservation.sh",
		"/var/lib/justvoxel/management/logs/operations", o.operationsDir)
	const prefix = "/usr/libexec/justvoxel/mjust/"
	legacy := copyScript("runtime/minecraft-backup",
		prefix+"backup-common.sh", stub, prefix+"interrupt-safety.sh", stub,
		prefix+"backup-instance-storage.sh", stub, prefix+"switch-matrix-reservation.sh", reservation,
		"/run/justvoxel-minecraft-restore-exclusion.lock", instanceRestoreExclusionPath,
		"/run/justvoxel-minecraft-maintenance.lock", pluginRuntimeLockPath,
		"/var/lib/justvoxel/management/setup-in-progress", filepath.Join(dir, "setup"),
		"${EUID} -ne 0", "${EUID} -ne "+strconv.Itoa(os.Geteuid()))
	apply := copyScript("mjust/libexec/admin-backup-delete-apply.sh",
		prefix+"switch-matrix-reservation.sh", reservation,
		"/run/justvoxel-minecraft-restore-exclusion.lock", instanceRestoreExclusionPath)
	return func() {
		t.Helper()
		cmd := exec.CommandContext(ctx, "bash", "-c", `JUSTVOXEL_MAINTENANCE_LOCK_HELD=1 bash "$1"`, "backup", legacy)
		if output, err := cmd.CombinedOutput(); err == nil ||
			(!strings.Contains(string(output), "Restore operation") && !strings.Contains(string(output), "Switch matrix maintenance")) {
			t.Fatalf("legacy backup refusal: %s, %v", output, err)
		}
		cmd = exec.CommandContext(ctx, "bash", "-c", `
backup_delete_json_error() { printf '%s\n' "$1"; }
trace=$3
backup_delete_collect_plan() { touch "$trace"; return 1; }
JV_MAINTENANCE_LOCK=$2
source "$1"
backup_delete_apply <<< '{}'
`, "delete", apply, pluginRuntimeLockPath, trace)
		if output, err := cmd.CombinedOutput(); err != nil || !strings.Contains(string(output), "Nothing was deleted") {
			t.Fatalf("backup deletion refusal: %s, %v", output, err)
		}
		if _, err := os.Stat(trace); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("backup path started storage work before exclusion")
		}
	}
}
