//go:build justvoxel_dev

package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"golang.org/x/sys/unix"
)

func TestSwitchMatrixProductionFixtureCreation(t *testing.T) {
	t.Run("legacy fixture retirement", func(t *testing.T) { testSwitchMatrixProductionFixtureCreation(t, false, false) })
	t.Run("stopped source retirement after reboot", func(t *testing.T) { testSwitchMatrixProductionFixtureCreation(t, true, false) })
	t.Run("transient fixture shutdown recovery", func(t *testing.T) { testSwitchMatrixProductionFixtureCreation(t, false, true) })
}

func testSwitchMatrixProductionFixtureCreation(t *testing.T, reboot, automatic bool) {
	f := newInstanceActivationFixture(t)
	// The running source stays admitted while the second fixture reserves Clone capacity.
	f.snapshot.TotalMemoryMiB = 16384
	b := f.backend
	b.networkRoot = filepath.Join(t.TempDir(), "network")
	o := matrixTestStore(t, b)
	s := &server{operations: o}
	j, err := o.beginSwitchMatrix(switchMatrixRunnerFingerprint())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = o.transition(j.OperationID, operationValidating, "manifest", "Reviewing fixtures."); err != nil {
		t.Fatal(err)
	}
	if _, err = o.transition(j.OperationID, operationRunning, "fixtures", "Preparing fixtures."); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	manifest := matrixRunnerTestManifest(t)
	smoke, reason := selectSwitchMatrixSmoke(manifest)
	if smoke < 0 {
		t.Fatal("smoke case unavailable", reason)
	}
	sourceCase := manifest.Cases[smoke]
	if sourceCase.SourceState != "running" {
		t.Fatal("expected a running source for the smoke Clone")
	}
	targetFixture := slices.IndexFunc(manifest.Cases, func(c switchMatrixCase) bool {
		return c.Source == sourceCase.Target && c.Version == sourceCase.Version && c.Family == switchMatrixModern &&
			c.Action == "clone" && c.SourceState == "stopped" && !c.Crossplay && c.Disposition == ""
	})
	if targetFixture < 0 {
		t.Fatal("expected a stopped fixture at the smoke Clone target version")
	}
	fixtureCases := []switchMatrixCase{sourceCase, manifest.Cases[targetFixture]}
	var fixtureIndex int
	var ids []string
	failFixture := func(step string) {
		t.Helper()
		c := fixtureCases[fixtureIndex]
		requiredMiB := switchMatrixMemoryMiB
		if c.Action == "clone" {
			requiredMiB *= 2
		}
		runningFixtureMiB := 0
		for _, id := range ids {
			if f.states[id] == "active" {
				runningFixtureMiB += switchMatrixMemoryMiB
			}
		}
		t.Fatalf("%s (fixture index=%d software=%s version=%s action=%s expected_state=%s total_mib=%d requested_mib=%d running_fixture_mib=%d system_reserve_mib=%d units=%v containers=%v)",
			step, fixtureIndex, c.Source, c.Version, c.Action, c.SourceState, f.snapshot.TotalMemoryMiB, requiredMiB,
			runningFixtureMiB, minecraftMemoryReserveMiB, f.states, f.containers)
	}

	// Keep the real registry, renderer, activation and receipt files. Replace
	// the host capacity and command boundaries for the isolated filesystem.
	oldStatfs, oldBackup, oldUnits := switchMatrixStatfs, instanceBackupStorageCommand, instanceBackupUnits
	t.Cleanup(func() {
		switchMatrixStatfs, instanceBackupStorageCommand, instanceBackupUnits = oldStatfs, oldBackup, oldUnits
	})
	instanceBackupUnits = t.TempDir()
	switchMatrixStatfs = func(_ string, out *unix.Statfs_t) error {
		out.Bsize, out.Bavail, out.Ffree = 4096, 1<<23, 8192
		return nil
	}
	root := filepath.Join(o.baseDir, "switch-matrix-fixtures")
	inode := func(path string) [2]uint64 {
		info, err := os.Lstat(path)
		if err != nil || !info.IsDir() {
			failFixture("fixture storage identity unavailable")
		}
		st := info.Sys().(*syscall.Stat_t)
		return [2]uint64{uint64(st.Dev), st.Ino}
	}
	dataIdentity := func(path string) string {
		data := inode(path)
		return fmt.Sprintf("%d:%d", data[0], data[1])
	}
	substituted, replacedData := "", ""
	backupReplaced := false
	var shutdownHash string
	var failNextStop bool
	type provisionedFixture struct {
		record instanceRuntimeRecord
		data   switchMatrixDataIdentity
	}
	provisioned := map[string]provisionedFixture{}
	verifiedVersion := map[string]int{}
	b.command = func(ctx context.Context, command string, args ...string) ([]byte, error) {
		switch command {
		case instanceRuntimeHelper:
			if args[0] == "version" {
				verifiedVersion[args[1]+"/"+args[2]]++
			}
		case "podman":
			switch args[0] {
			case "info":
				return []byte(f.snapshot.StorageRoot), nil
			case "image", "inspect":
				if args[len(args)-1] == "{{.Id}}" || args[len(args)-1] == "{{.Image}}" {
					return []byte("exact-fixture-image\n"), nil
				}
			case "kill":
				if failNextStop {
					failNextStop = false
					return nil, errors.New("transient shutdown command failure")
				}
				if shutdownHash != "" {
					id := strings.TrimPrefix(args[len(args)-1], "minecraft-")
					hash, err := switchMatrixConfigHash(id)
					if err != nil || hash != shutdownHash {
						failFixture("graceful shutdown started before the original configuration was restored")
					}
				}
			case "rm":
				id := strings.TrimPrefix(args[len(args)-1], "minecraft-")
				if _, ok := provisioned[id]; !ok {
					failFixture("cleanup attempted to remove a foreign container")
				}
				delete(f.containers, id)
				return nil, nil
			}
		case "systemctl":
			if args[0] == "show" && strings.HasPrefix(args[len(args)-1], "minecraft-backup-") {
				return []byte("LoadState=not-found\nActiveState=inactive\nJob=0\n"), nil
			}
			if args[0] == "show" && args[1] == "--property=Job" {
				return []byte("0\n"), nil
			}
		case "nsenter":
			if len(args) < 12 || args[0] != "--mount=/proc/1/ns/mnt" || args[1] != "--" {
				failFixture("unexpected fixture storage command")
			}
			helper, action, id := args[2], args[3], args[4]
			path := args[6]
			if !instanceIDPattern.MatchString(id) || args[5] != root || path != filepath.Join(root, "minecraft-instances", id, "data") {
				failFixture("host storage operation escaped the fixture")
			}
			switch helper {
			case instanceRuntimeHelper:
				if action == "provision" {
					raw, err := os.ReadFile(filepath.Join(o.operationsDir, j.OperationID+".json"))
					var persisted operationJournal
					receipt, receiptErr := readSwitchMatrixFixture(j.OperationID, id)
					if err != nil || instanceDecode(raw, &persisted) != nil || receiptErr != nil ||
						persisted.OperationID != j.OperationID || persisted.OperationType != operationTypeSwitchMatrix ||
						persisted.PlanFingerprint != j.PlanFingerprint || persisted.State != operationRunning ||
						persisted.Stage != "fixtures" || !slices.Contains(persisted.MatrixFixtures, id) ||
						receipt.Storage.Root != root || receipt.Storage.DataPath != path ||
						!slices.Equal(args[7:], []string{receipt.Storage.MountPoint, receipt.Storage.UUID, receipt.Storage.Source,
							strconv.FormatUint(uint64(receipt.UID), 10), strconv.FormatUint(uint64(receipt.GID), 10)}) {
						failFixture("runtime published ahead of its durable ownership reservation")
					}
					previous, alreadyProvisioned := provisioned[id]
					if alreadyProvisioned {
						o.mu.Lock()
						parent, parentErr := o.switchMatrixParent(j.OperationID, "fixtures", false)
						o.mu.Unlock()
						record, err := b.readRecord(id)
						if parentErr != nil || !slices.Contains(parent.MatrixFixtures, id) ||
							err != nil || receipt.Stage != "owned" || !record.Runtime.MatrixFixture ||
							receipt.checkRecord(previous.record) != nil || receipt.checkRecord(record) != nil ||
							receipt.DataParent != previous.data.Parent || receipt.DataIdentity != previous.data.Data ||
							id == substituted || id == replacedData ||
							inode(filepath.Dir(path)) != previous.data.Parent || dataIdentity(path) != previous.data.Data {
							failFixture("repeat provisioning lacked exact ownership of the existing fixture")
						}
						// The existing data and parent are already provisioned.
						return nil, nil
					}
					if receipt.Stage != "reserved" || !receipt.validCandidate() {
						failFixture("runtime published ahead of its durable ownership reservation")
					}
					if _, err := os.Lstat(filepath.Dir(path)); !errors.Is(err, os.ErrNotExist) {
						failFixture("initial provisioning found an existing fixture parent")
					}
					if _, err := f.command(ctx, command, args...); err != nil {
						return nil, err
					}
					provisioned[id] = provisionedFixture{
						record: *receipt.Candidate,
						data:   switchMatrixDataIdentity{Data: dataIdentity(path), Parent: inode(filepath.Dir(path))},
					}
					return nil, nil
				}
			case instanceDeleteHelper:
				switch action {
				case "matrix-inspect":
					parent := inode(filepath.Dir(path))
					if id == substituted {
						parent[1]++
					}
					data := dataIdentity(path)
					if id == replacedData {
						data = "99:123"
					}
					return mustSwitchJSON(switchMatrixDataIdentity{Data: data, Parent: parent}), nil
				case "inspect":
					return mustSwitchJSON(map[string]string{"identity": dataIdentity(path)}), nil
				case "remove":
					if args[12] != provisioned[id].data.Data {
						failFixture("cleanup attempted to remove data without the provisioned inode")
					}
					if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
						return nil, nil // The production helper accepts an idempotent removal.
					}
					if args[12] != dataIdentity(path) {
						failFixture("cleanup attempted to remove data without the exact inode")
					}
					return nil, os.Remove(path)
				}
			case instanceSwitchRetireHelper:
				switch action {
				case "unrestored":
					return nil, nil
				case "matrix-empty":
					var identity [2]uint64
					if instanceDecode([]byte(args[12]), &identity) != nil || identity != inode(filepath.Dir(path)) {
						failFixture("cleanup attempted to remove a substituted parent")
					}
					return nil, os.Remove(filepath.Dir(path))
				}
			}
		}
		return f.command(ctx, command, args...)
	}
	instanceBackupStorageCommand = func(_ context.Context, command string, args ...string) ([]byte, error) {
		if command != "nsenter" || len(args) != 6 || args[0] != "--mount=/proc/1/ns/mnt" || args[1] != "--" ||
			args[2] != instanceBackupHelper || args[5] != j.OperationID {
			failFixture("backup helper escaped the reserved fixture")
		}
		id := args[4]
		if _, ok := provisioned[id]; !ok {
			failFixture("backup helper targeted an unprovisioned fixture")
		}
		backupRoot := filepath.Join(root, "backups")
		path := filepath.Join(backupRoot, id)
		staged := filepath.Join(backupRoot, ".matrix-"+j.OperationID+"-"+id)
		receipt, err := readSwitchMatrixFixture(j.OperationID, id)
		if err != nil || receipt.Placement.Root != backupRoot || receipt.Placement.Path != path {
			failFixture("backup helper lacked exact fixture placement")
		}
		parent, err := o.get(j.OperationID)
		if err != nil || parent.OperationType != operationTypeSwitchMatrix || !slices.Contains(parent.MatrixFixtures, id) {
			failFixture("backup helper lacked the parent reservation")
		}
		binding := parent.State == operationRunning && (parent.Stage == "fixtures" || parent.Stage == "switch") ||
			parent.State == operationNeedsAttention
		retirement := parent.State == operationRunning && parent.Stage == "retirement" || parent.State == operationNeedsAttention
		record, err := b.readRecord(id)
		if err != nil || receipt.checkRecord(record) != nil {
			failFixture("backup helper lacked the committed fixture placement")
		}
		if args[3] != "matrix-placement" {
			hash, err := switchMatrixConfigHash(id)
			if err != nil || hash != receipt.ConfigSHA256 {
				failFixture("backup helper lacked the committed configuration hash")
			}
		}
		inventory := func(directory string) switchMatrixBackups {
			identity := inode(directory)
			entries, err := os.ReadDir(directory)
			if err != nil || len(entries) != 0 {
				failFixture("expected an empty fixture backup namespace")
			}
			return switchMatrixBackups{Root: inode(backupRoot), Directory: identity, Files: map[string]switchMatrixBackupFile{}}
		}
		switch args[3] {
		case "matrix-prepare":
			if _, err := os.Lstat(path); !binding || receipt.Stage != "binding" || receipt.Backups.Root != [2]uint64{} ||
				receipt.Backups.Directory != [2]uint64{} || !errors.Is(err, os.ErrNotExist) {
				failFixture("backup preparation lacked an unused binding")
			}
			if err := os.Mkdir(staged, 0700); err != nil {
				return nil, err
			}
			return mustSwitchJSON(inventory(staged)), nil
		case "matrix-bind":
			prepared := inventory(staged)
			if _, err := os.Lstat(path); !binding || receipt.Stage != "binding" || receipt.Backups.Root != prepared.Root ||
				receipt.Backups.Directory != prepared.Directory || !errors.Is(err, os.ErrNotExist) {
				failFixture("backup namespace published before its binding was recorded")
			}
			return nil, os.Rename(staged, path)
		case "matrix-inventory", "matrix-retire", "matrix-placement":
			if _, err := os.Lstat(staged); !errors.Is(err, os.ErrNotExist) ||
				receipt.Backups.Root != inode(backupRoot) || receipt.Backups.Directory != inode(path) {
				failFixture("backup inspection or retirement lacked exact ownership")
			}
			switch args[3] {
			case "matrix-inventory":
				if receipt.Stage != "owned" || !(retirement || parent.State == operationRunning && parent.Stage == "switch") {
					failFixture("backup inventory requested outside the owned Switch or retirement phase")
				}
				return mustSwitchJSON(inventory(path)), nil
			case "matrix-retire":
				if !retirement || receipt.Stage != "intent" || len(receipt.Backups.Files) != 0 {
					failFixture("backup retirement lacked the recorded empty inventory")
				}
				return nil, os.Remove(path)
			case "matrix-placement":
				if backupReplaced || parent.State != operationNeedsAttention || receipt.Stage != "owned" {
					return nil, errInstanceRecovery
				}
				return nil, nil
			}
		}
		failFixture("unexpected fixture backup command")
		return nil, errInstanceRecovery
	}

	passwords := map[string]bool{}
	var paths [2][]string
	var originals []instanceRuntimeRecord
	startCommand := pluginInstallCommand
	t.Cleanup(func() { pluginInstallCommand = startCommand })
	pluginInstallCommand = func(_ context.Context, command string, args ...string) ([]byte, error) {
		if len(ids) != 2 || command != "systemctl" || !slices.Equal(args, []string{"start", "minecraft-" + ids[1] + ".service"}) {
			failFixture("unexpected fixture start")
		}
		f.states[ids[1]], f.containers[ids[1]] = "active", "running"
		return nil, nil
	}
	foreignBefore, err := b.readRecord(f.configs[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	for i, c := range fixtureCases {
		fixtureIndex = i
		config, snapshot, err := s.prepareSwitchMatrixSource(ctx, c)
		if err != nil {
			stage, reason := switchMatrixFixtureDiagnostic(err)
			failFixture(fmt.Sprintf("matrix source preparation failed (stage=%s reason=%s)", stage, reason))
		}
		if !instanceRuntimeSecretPattern.MatchString(config.RCONPassword) || passwords[config.RCONPassword] {
			failFixture("matrix credential failed validation or was reused")
		}
		passwords[config.RCONPassword] = true
		name := "Switch fixture " + c.Source
		invalid := config
		invalid.RCONPassword = strings.Repeat("z", 48)
		if id, err := o.createSwitchMatrixFixture(ctx, j.OperationID, invalid, name, snapshot); id != "" || !errors.Is(err, errInstanceRuntimeCredential) {
			failFixture("production creation did not reject an invalid RCON credential")
		}
		id, err := o.createSwitchMatrixFixture(ctx, j.OperationID, config, name, snapshot)
		if err != nil {
			stage, reason := switchMatrixFixtureDiagnostic(err)
			failFixture(fmt.Sprintf("production fixture creation failed (stage=%s reason=%s)", stage, reason))
		}
		record, err := o.switchMatrixOwnedRuntime(ctx, j.OperationID, id)
		_, didProvision := provisioned[id]
		if err != nil || record.Runtime.RCONPassword != config.RCONPassword ||
			!record.Runtime.MatrixFixture || !record.Runtime.PreserveContent || !didProvision ||
			verifiedVersion[c.Source+"/"+c.Version] != 1 {
			failFixture("published runtime does not match the reserved fixture")
		}
		_, envPath, quadletPath := b.paths(id)
		env, envErr := b.readPrivate(envPath)
		quadlet, quadletErr := b.readPrivate(quadletPath)
		status, statusErr := b.observeRaw(ctx, id)
		if envErr != nil || quadletErr != nil || statusErr != nil || status.State != "stopped" ||
			!bytes.Contains(env, []byte("TYPE="+strings.ToUpper(c.Source)+"\n")) ||
			!bytes.Contains(env, []byte("VERSION="+c.Version+"\n")) ||
			!bytes.Contains(env, []byte("RCON_PASSWORD="+config.RCONPassword+"\n")) ||
			bytes.Contains(env, []byte("MODRINTH_PROJECTS=")) ||
			!bytes.Contains(quadlet, []byte("PublishPort=127.0.0.1:"+strconv.Itoa(config.JavaPort)+":25565/tcp")) ||
			bytes.Contains(quadlet, []byte("[Install]")) || record.Firewall == nil || len(record.Firewall) != 0 {
			failFixture("production environment, Quadlet or service contract was not published")
		}
		ids = append(ids, id)
		originals = append(originals, record)
		paths[i] = []string{envPath, quadletPath, record.Storage.DataPath, record.Backups.Path}

		beforeHash, err := switchMatrixConfigHash(id)
		if err != nil {
			failFixture("could not hash the original source configuration")
		}
		if err = b.prepareSwitchSourceCopy(ctx, id, c.Version); err != nil {
			failFixture("prepare source copy")
		}
		state := c.SourceState
		if state == "running" {
			// Production Clone restarts an originally running source before
			// clearing its temporary configuration.
			f.states[id], f.containers[id] = "active", "running"
		}
		e := minecraftSwitchEvidence{OperationID: j.OperationID, Source: id, SourceRecord: &record, SourceContentChanged: true}
		if err = s.clearSwitchSourceContent(ctx, &e); err != nil {
			failFixture("clear temporary source configuration")
		}
		afterHash, err := switchMatrixConfigHash(id)
		if err != nil || afterHash != beforeHash {
			failFixture("source configuration no longer matches its original SHA256")
		}
		if err = s.withSwitchMatrixRuntime(ctx, j.OperationID, id, state, func(ctx context.Context, current instanceRuntimeRecord) error {
			if !bytes.Equal(mustSwitchJSON(current), mustSwitchJSON(record)) {
				return errInstanceRecovery
			}
			status, err := b.observeRaw(ctx, id)
			if err != nil || status.State != "running" {
				return errors.Join(errInstanceRecovery, err)
			}
			return nil
		}); err != nil {
			stage, reason := switchMatrixStartupDiagnostic("original_source", err)
			failFixture(fmt.Sprintf("original source verification failed (stage=%s reason=%s)", stage, reason))
		}
		if state == "stopped" {
			if err = s.stopSwitchMatrixFixtureForRetirement(ctx, j.OperationID, id); err != nil {
				failFixture("could not return verified fixture to stopped state")
			}
		}
		status, err = b.observeRaw(ctx, id)
		if err != nil || status.State != state {
			failFixture("verified fixture did not return to its required state")
		}
	}
	fixtureIndex = 0
	source, cloneID := originals[0], ids[1]
	if automatic {
		run := switchMatrixRun{Schema: 1, ID: j.OperationID, State: "running",
			Cases: []switchMatrixCase{sourceCase, manifest.Cases[targetFixture], manifest.Cases[smoke+1]}}
		run.Results = newSwitchMatrixResults(run.Cases)
		if err := o.saveSwitchMatrixRun(run); err != nil {
			t.Fatal(err)
		}
		admitted := 0
		err := o.runSwitchMatrixCases(j.OperationID, func(c switchMatrixCase) (switchMatrixCaseResult, error) {
			admitted++
			if admitted == 2 {
				failNextStop = true
				child, err := s.closeSwitchMatrixCase(ctx, j.OperationID, 0)
				if _, ok := err.(*switchMatrixRetryableClosure); !ok {
					t.Fatalf("expected a retryable closure failure: %v", err)
				}
				return switchMatrixCaseResult{Outcome: matrixFailed, Stage: "source_world", Reason: "setblock_rejected",
					World: "source_markers_failed", Cleanup: "unresolved", ClosureReason: switchMatrixClosureDiagnostic(child, err)}, err
			}
			if admitted == 3 {
				for _, id := range ids {
					receipt, err := readSwitchMatrixFixture(j.OperationID, id)
					if err != nil || receipt.Stage != "retired" {
						t.Fatalf("case admitted before verified fixture retirement: %v", err)
					}
				}
			}
			return matrixRunnerPassed(c), nil
		}, func() (operationJournal, error) {
			persisted, err := o.readSwitchMatrixRun(j.OperationID)
			if err != nil || persisted.Results[1].ClosureReason != "fixture_shutdown_unverified" {
				t.Fatalf("closure evidence must be durable before retry: %v", err)
			}
			return s.recoverSwitchMatrixClosure(j.OperationID, true)
		})
		run, readErr := o.readSwitchMatrixRun(j.OperationID)
		result := run.Results[1]
		if err != nil || readErr != nil || admitted != 3 || run.State != "completed" || result.Outcome != matrixFailed ||
			result.Stage != "source_world" || result.Reason != "setblock_rejected" ||
			result.ClosureReason != "fixture_shutdown_unverified" || result.ClosureRecovery != "verified" || result.Cleanup != "verified" {
			t.Fatalf("automatic recovery lost diagnostics or did not continue: %+v %v %v", run, err, readErr)
		}
		return
	}
	if _, err = o.transition(j.OperationID, operationRunning, "switch", "Reviewing Switch."); err != nil {
		failFixture("could not enter the Switch phase")
	}
	child, err := o.beginSwitchMatrixChild(j.OperationID, source.Runtime.ID, switchMatrixRunnerFingerprint())
	if err != nil {
		failFixture("could not queue the Clone")
	}
	e := minecraftSwitchEvidence{Schema: 1, OperationID: child.OperationID, Source: source.Runtime.ID, SourceRecord: &source,
		Target: originals[1].Runtime.Software, Version: source.Runtime.Version, Action: "clone", Stage: "complete",
		CloneID: cloneID, CloneRecord: &originals[1], OriginallyRunning: true}
	if reboot {
		e.Stage, e.CloneID, e.CloneRecord = "source_copy", "", nil
		e.SourceStopped = false
		if err = saveSwitchEvidence(e); err != nil {
			failFixture("could not persist the recovered source and interrupted Clone")
		}
		if _, err = o.transition(child.OperationID, operationNeedsAttention, "interrupted", "Switch was interrupted."); err != nil {
			failFixture("could not mark the child interrupted")
		}
		o.holdSwitchMatrixRun(j.OperationID)
		if err = o.saveSwitchMatrixRun(switchMatrixRun{Schema: 1, ID: j.OperationID, State: "needs_attention",
			Cases: []switchMatrixCase{sourceCase}, Results: []switchMatrixCaseResult{
				{Outcome: matrixFailed, ChildOperation: child.OperationID, Original: "failed", Cleanup: "unresolved"},
			}}); err != nil {
			failFixture("could not retain the failed result")
		}
		f.states[e.Source] = "inactive"
		delete(f.containers, e.Source)
		pluginInstallCommand = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			failFixture("recovery tried to start or probe the stopped disposable source")
			return nil, errInstanceRecovery
		}
		if err = s.recoverSwitchMatrixRun(j.OperationID); err != nil {
			t.Fatalf("stopped fixture recovery failed: %v", err)
		}
		resolved, err := o.get(child.OperationID)
		if err != nil || resolved.State != operationResolved {
			failFixture("child recovery did not resolve")
		}
		run, err := o.readSwitchMatrixRun(j.OperationID)
		if err != nil || run.State != "recovered" || len(run.Results) != 1 ||
			run.Results[0].Outcome != matrixFailed || run.Results[0].Original != "failed" ||
			run.Results[0].Recovery != "verified" || run.Results[0].Cleanup != "verified" {
			failFixture("verified closure did not preserve the historical failure")
		}
		for _, id := range ids {
			receipt, err := readSwitchMatrixFixture(j.OperationID, id)
			if err != nil || receipt.Stage != "retired" || f.states[id] != "" || f.containers[id] != "" {
				failFixture("fixture was not retired")
			}
		}
		if active, err := o.currentMinecraftMaintenance(); err != nil || active != nil {
			failFixture("verified closure retained maintenance")
		}
		return
	}
	for _, step := range []struct {
		state operationState
		stage string
	}{{operationValidating, "validating"}, {operationRunning, "converting"}, {operationVerifying, "verifying"}, {operationSucceeded, "complete"}} {
		if step.state == operationVerifying {
			if err = saveSwitchEvidence(e); err != nil {
				failFixture("could not persist completed Clone evidence before verification")
			}
		}
		if _, err = o.transition(child.OperationID, step.state, step.stage, "Fixture Switch transaction."); err != nil {
			failFixture(fmt.Sprintf("could not transition Clone (state=%s stage=%s)", step.state, step.stage))
		}
	}
	fixtureIndex = 1
	substituted = ids[len(ids)-1]
	if _, err := s.closeSwitchMatrixCase(ctx, j.OperationID, 0); !errors.Is(err, errInstanceRecovery) {
		failFixture("cleanup did not refuse substituted fixture storage")
	}
	substituted = ""
	if _, err := s.closeSwitchMatrixCase(ctx, j.OperationID, 1); err != nil {
		failFixture("exact clone fixture cleanup failed")
	}
	fixtureIndex = 0
	o.holdSwitchMatrixRun(j.OperationID)
	parent, err := o.get(j.OperationID)
	if err != nil || parent.State != operationNeedsAttention || parent.Stage != "matrix_recovery" {
		failFixture("parent did not enter matrix recovery")
	}
	cases := []switchMatrixCase{sourceCase}
	if err = o.saveSwitchMatrixRun(switchMatrixRun{Schema: 1, ID: j.OperationID, State: "needs_attention", Cases: cases,
		Results: []switchMatrixCaseResult{{Outcome: matrixFailed, ChildOperation: child.OperationID, Original: "failed",
			Cleanup: "unresolved", ClosureReason: "fixture_closure_unverified"}}}); err != nil {
		failFixture("could not persist the unresolved matrix result")
	}
	dir, envPath, quadletPath := b.paths(source.Runtime.ID)
	configPath := filepath.Join(dir, "config.json")
	legacy := source
	legacy.Firewall = nil
	if err = b.writeJSON(configPath, legacy, true); err != nil {
		failFixture("could not reproduce the legacy firewall encoding")
	}
	reject := func(scenario string) {
		t.Helper()
		before, err := switchMatrixConfigHash(source.Runtime.ID)
		if err != nil {
			failFixture("could not hash the source before recovery for " + scenario)
		}
		if err = s.recoverSwitchMatrixRun(j.OperationID); !errors.Is(err, errInstanceRecovery) {
			failFixture("recovery did not refuse " + scenario)
		}
		after, err := switchMatrixConfigHash(source.Runtime.ID)
		if err != nil || after != before || f.states[source.Runtime.ID] != "active" {
			failFixture("recovery modified a fixture before verifying ownership for " + scenario)
		}
	}
	changed := legacy
	changed.Runtime.MaxPlayers++
	if err = b.writeJSON(configPath, changed, true); err != nil {
		failFixture("could not write changed runtime settings")
	}
	reject("changed runtime settings")
	if err = b.writeJSON(configPath, legacy, true); err != nil {
		failFixture("could not restore the legacy firewall encoding")
	}
	substituted = source.Runtime.ID
	reject("substituted data parent")
	substituted = ""
	replacedData = source.Runtime.ID
	reject("replaced data inode")
	replacedData = ""
	backupReplaced = true
	reject("replaced backup namespace")
	backupReplaced = false
	for _, path := range []string{envPath, quadletPath} {
		raw, err := b.readPrivate(path)
		if err != nil {
			failFixture("could not read " + filepath.Base(path))
		}
		if err = b.writePrivate(path, append(slices.Clone(raw), '\n'), true); err != nil {
			failFixture("could not change " + filepath.Base(path))
		}
		reject("changed " + filepath.Base(path))
		if err = b.writePrivate(path, raw, true); err != nil {
			failFixture("could not restore " + filepath.Base(path))
		}
	}
	unrelated := e
	unrelated.OperationID = j.OperationID
	if err = b.writeJSON(switchEvidencePath(child.OperationID), unrelated, true); err != nil {
		failFixture("could not write unrelated child operation evidence")
	}
	reject("unrelated child operation evidence")
	if err = saveSwitchEvidence(e); err != nil {
		failFixture("could not restore completed child operation evidence")
	}
	wrongSaved := source
	wrongSaved.Firewall = nil
	unrelated = e
	unrelated.SourceRecord = &wrongSaved
	if err = saveSwitchEvidence(unrelated); err != nil {
		failFixture("could not write child evidence with the wrong firewall encoding")
	}
	reject("saved source with the wrong firewall encoding")
	if err = saveSwitchEvidence(e); err != nil {
		failFixture("could not restore completed child operation evidence")
	}
	if _, err = o.switchMatrixOwnedRuntime(ctx, j.OperationID, source.Runtime.ID); !errors.Is(err, errInstanceRecovery) {
		failFixture("legacy fixture passed strict ownership before repair")
	}
	shutdownHash = instanceHash(append(mustSwitchJSON(source), '\n'))
	if err = s.recoverSwitchMatrixRun(j.OperationID); err != nil {
		failFixture("exact legacy source recovery and retirement failed")
	}
	run, err := o.readSwitchMatrixRun(j.OperationID)
	if err != nil || run.State != "recovered" || len(run.Results) != 1 || run.Results[0].Outcome != matrixFailed ||
		run.Results[0].Original != "failed" || run.Results[0].Cleanup != "verified" ||
		run.Results[0].ClosureReason != "fixture_closure_unverified" || run.Results[0].ClosureRecovery != "verified" {
		failFixture("recovery did not preserve the historical failure and verify cleanup")
	}
	completed, err := o.get(child.OperationID)
	if err != nil || completed.State != operationSucceeded {
		failFixture("recovery altered the completed Clone")
	}
	active, err := o.currentMinecraftMaintenance()
	if err != nil || active != nil {
		failFixture("verified closure retained the parent reservation")
	}
	for i, fixturePaths := range paths {
		fixtureIndex = i
		for _, path := range fixturePaths {
			if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
				failFixture("owned fixture resource remained after retirement: " + path)
			}
		}
	}
	foreignAfter, err := b.readRecord(f.configs[0].ID)
	if err != nil || !bytes.Equal(mustSwitchJSON(foreignBefore), mustSwitchJSON(foreignAfter)) {
		t.Fatal("cleanup changed a foreign instance", err)
	}
}

func TestSwitchMatrixFixtureFailureDiagnostics(t *testing.T) {
	s, j, r := newMatrixRunnerTest(t, 2)
	o, b := s.operations, instanceRuntimes
	record, err := b.readRecord("jv-a1b2c3")
	if err != nil {
		t.Fatal(err)
	}
	secret := record.Runtime.RCONPassword
	cause := fmt.Errorf("publisher response included credential %s: %w", secret, errInstanceBusy)
	if secret == "" || !strings.Contains(cause.Error(), secret) {
		t.Fatal("test did not inject a private fixture credential")
	}
	for _, scenario := range []struct {
		stage  switchMatrixFixtureStage
		reason string
	}{
		{matrixFixtureConfiguration, "fixture_runtime_configuration_invalid"},
		{matrixFixtureCredential, "fixture_rcon_generation_failed"},
		{matrixFixtureStorage, "fixture_storage_placement_unverified"},
		{matrixFixtureSoftware, "fixture_software_version_unverified"},
		{matrixFixturePorts, "fixture_port_reservation_failed"},
		{matrixFixtureOwnership, "fixture_ownership_reservation_failed"},
		{matrixFixturePublication, "fixture_runtime_publication_failed"},
		{matrixFixtureBinding, "fixture_ownership_binding_failed"},
	} {
		wrapped := wrapSwitchMatrixFixtureError(scenario.stage, cause)
		stage, reason := switchMatrixFixtureDiagnostic(wrapped)
		var failure *switchMatrixFixtureError
		if !errors.Is(wrapped, errInstanceBusy) || !errors.As(wrapped, &failure) ||
			stage != string(scenario.stage) || reason != scenario.reason || strings.Contains(wrapped.Error(), secret) {
			t.Fatal("fixture classification lost its cause or leaked private details", stage, reason)
		}
	}
	stage, reason := switchMatrixFixtureDiagnostic(wrapSwitchMatrixFixtureError(matrixFixtureConfiguration, errInstanceRuntimeCredential))
	if stage != "credential_validation" || reason != "fixture_rcon_credential_invalid" {
		t.Fatal("renderer credential rejection was misclassified")
	}

	stage, reason = switchMatrixStartupDiagnostic("source", errors.Join(errSwitchMatrixStartup, cause))
	if stage != "source_start" || reason != "runtime_start_unverified" {
		t.Fatal("startup failure was misclassified", stage, reason)
	}
	stage, reason = switchMatrixStartupDiagnostic("source", cause)
	if stage != "source_readiness" || reason != "runtime_readiness_unverified" {
		t.Fatal("readiness failure was misclassified", stage, reason)
	}
	if switchMatrixClosureDiagnostic(operationJournal{}, cause) != "fixture_closure_unverified" ||
		switchMatrixClosureDiagnostic(operationJournal{OperationID: j.OperationID, State: operationNeedsAttention}, cause) != "switch_child_recovery_unverified" {
		t.Fatal("cleanup and child recovery failures were not distinguished")
	}

	// A host-command failure enters the production case worker during
	// preparation. The runner publishes the same result used for download.
	base := b.command
	oldStatfs := switchMatrixStatfs
	t.Cleanup(func() { switchMatrixStatfs = oldStatfs })
	switchMatrixStatfs = func(_ string, out *unix.Statfs_t) error {
		out.Bsize, out.Bavail, out.Ffree = 4096, 1<<23, 8192
		return nil
	}
	b.command = func(ctx context.Context, command string, args ...string) ([]byte, error) {
		switch {
		case command == "podman" && args[0] == "info":
			return []byte(o.baseDir), nil
		case command == instanceRuntimeHelper && args[0] == "channel":
			return []byte(cause.Error()), cause
		default:
			return base(ctx, command, args...)
		}
	}
	if err := o.runSwitchMatrixCases(j.OperationID, func(c switchMatrixCase) (switchMatrixCaseResult, error) {
		return s.executeSwitchMatrixCase(j.OperationID, c)
	}); err != nil {
		t.Fatal(err)
	}
	raw, err := o.readDiagnosticLog("migration", j.OperationID)
	if err != nil || !bytes.Contains(raw, []byte("fixture_software_version_unverified")) ||
		bytes.Contains(raw, []byte(secret)) || bytes.Contains(raw, []byte("rcon_password")) {
		t.Fatal("download omitted attribution or exposed private runtime state", err)
	}
	w := httptest.NewRecorder()
	s.writeSwitchMatrixStatus(w, http.StatusOK, j.OperationID)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"state":"halted"`) ||
		strings.Contains(w.Body.String(), secret) || strings.Contains(w.Body.String(), "rcon_password") {
		t.Fatal("public progress omitted halt state or exposed private runtime state")
	}
	run, err := o.readSwitchMatrixRun(j.OperationID)
	if err != nil || run.Results[0].CaseID != r.Cases[0].ID || run.Results[0].Stage != "software_version" ||
		run.Results[0].Cleanup != "verified" || run.Results[1].Outcome != matrixPending {
		t.Fatal("production failure lost its stage, closure or unfinished case", err)
	}
}
