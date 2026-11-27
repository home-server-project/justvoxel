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
	f := newInstanceActivationFixture(t)
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
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal("fixture storage identity unavailable", err)
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
	provisioned := map[string]bool{}
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
				if shutdownHash != "" {
					id := strings.TrimPrefix(args[len(args)-1], "minecraft-")
					hash, err := switchMatrixConfigHash(id)
					if err != nil || hash != shutdownHash {
						t.Fatal("graceful shutdown started before the original configuration was restored", err)
					}
				}
			case "rm":
				id := strings.TrimPrefix(args[len(args)-1], "minecraft-")
				if !provisioned[id] {
					t.Fatal("cleanup attempted to remove a foreign container")
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
			helper, action, id := args[2], args[3], args[4]
			path := args[6]
			if !instanceIDPattern.MatchString(id) || path != filepath.Join(root, "minecraft-instances", id, "data") {
				t.Fatal("host storage operation escaped the fixture")
			}
			switch helper {
			case instanceRuntimeHelper:
				if action == "provision" {
					raw, err := os.ReadFile(filepath.Join(o.operationsDir, j.OperationID+".json"))
					var persisted operationJournal
					receipt, receiptErr := readSwitchMatrixFixture(j.OperationID, id)
					if err != nil || instanceDecode(raw, &persisted) != nil || receiptErr != nil ||
						!slices.Contains(persisted.MatrixFixtures, id) || receipt.Stage != "reserved" ||
						!receipt.validCandidate() || receipt.Storage.DataPath != path {
						t.Fatal("runtime published ahead of its durable ownership reservation", err, receiptErr)
					}
					provisioned[id] = true
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
					if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
						return nil, nil // The production helper accepts an idempotent removal.
					}
					if args[12] != dataIdentity(path) {
						t.Fatal("cleanup attempted to remove data without the exact inode")
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
						t.Fatal("cleanup attempted to remove a substituted parent")
					}
					return nil, os.Remove(filepath.Dir(path))
				}
			}
		}
		return f.command(ctx, command, args...)
	}
	instanceBackupStorageCommand = func(_ context.Context, command string, args ...string) ([]byte, error) {
		if command != "nsenter" || len(args) != 6 || args[2] != instanceBackupHelper || args[5] != j.OperationID || !provisioned[args[4]] {
			t.Fatal("backup helper escaped the reserved fixture")
		}
		id := args[4]
		backupRoot := filepath.Join(root, "backups")
		path := filepath.Join(backupRoot, id)
		switch args[3] {
		case "matrix-prepare":
			if err := os.Mkdir(path, 0700); err != nil {
				return nil, err
			}
			fallthrough
		case "matrix-inventory":
			return mustSwitchJSON(switchMatrixBackups{Root: inode(backupRoot), Directory: inode(path), Files: map[string]switchMatrixBackupFile{}}), nil
		case "matrix-bind", "matrix-retire", "matrix-placement":
			receipt, err := readSwitchMatrixFixture(j.OperationID, id)
			if err != nil || receipt.Backups.Root != inode(backupRoot) || receipt.Backups.Directory != inode(path) {
				t.Fatal("backup publication or retirement lacked exact ownership", err)
			}
			if args[3] == "matrix-retire" {
				return nil, os.Remove(path)
			}
			if args[3] == "matrix-placement" {
				if backupReplaced || receipt.Stage != "owned" {
					return nil, errInstanceRecovery
				}
				return nil, nil
			}
			if receipt.Stage != "binding" {
				t.Fatal("backup namespace published before its binding was recorded")
			}
			return nil, nil
		}
		t.Fatal("unexpected fixture backup command", args[3])
		return nil, errInstanceRecovery
	}

	manifest := matrixRunnerTestManifest(t)
	smoke, reason := selectSwitchMatrixSmoke(manifest)
	if smoke < 0 {
		t.Fatal("smoke case unavailable", reason)
	}
	targetFixture := slices.IndexFunc(manifest.Cases, func(c switchMatrixCase) bool {
		return c.Source == manifest.Cases[smoke].Target && c.Family == switchMatrixModern && !c.Crossplay && c.Disposition == ""
	})
	if targetFixture < 0 {
		t.Fatal("expected a fixture for the smoke case target")
	}
	passwords := map[string]bool{}
	var ids []string
	var paths []string
	var originals []instanceRuntimeRecord
	startCommand := pluginInstallCommand
	t.Cleanup(func() { pluginInstallCommand = startCommand })
	pluginInstallCommand = func(_ context.Context, command string, args ...string) ([]byte, error) {
		if len(ids) != 2 || command != "systemctl" || !slices.Equal(args, []string{"start", "minecraft-" + ids[1] + ".service"}) {
			t.Fatal("unexpected fixture start", command, args)
		}
		f.states[ids[1]], f.containers[ids[1]] = "active", "running"
		return nil, nil
	}
	foreignBefore, err := b.readRecord(f.configs[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []switchMatrixCase{manifest.Cases[smoke], manifest.Cases[targetFixture]} {
		config, snapshot, err := s.prepareSwitchMatrixSource(ctx, c)
		if err != nil {
			t.Fatal("matrix source preparation failed", err)
		}
		if !instanceRuntimeSecretPattern.MatchString(config.RCONPassword) || passwords[config.RCONPassword] {
			t.Fatal("matrix credential failed validation or was reused")
		}
		passwords[config.RCONPassword] = true
		name := "Switch fixture " + c.Source
		invalid := config
		invalid.RCONPassword = strings.Repeat("z", 48)
		if id, err := o.createSwitchMatrixFixture(ctx, j.OperationID, invalid, name, snapshot); id != "" || !errors.Is(err, errInstanceRuntimeCredential) {
			t.Fatal("production creation accepted an invalid RCON credential", err)
		}
		id, err := o.createSwitchMatrixFixture(ctx, j.OperationID, config, name, snapshot)
		if err != nil {
			t.Fatal("production fixture creation failed", err)
		}
		record, err := o.switchMatrixOwnedRuntime(ctx, j.OperationID, id)
		if err != nil || record.Runtime.RCONPassword != config.RCONPassword ||
			!record.Runtime.MatrixFixture || !record.Runtime.PreserveContent || !provisioned[id] ||
			verifiedVersion[c.Source+"/"+c.Version] != 1 {
			t.Fatal("published runtime does not match the reserved fixture", err)
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
			t.Fatal("production environment, Quadlet or service contract was not published", envErr, quadletErr, statusErr)
		}
		ids = append(ids, id)
		originals = append(originals, record)
		paths = append(paths, envPath, quadletPath, record.Storage.DataPath, record.Backups.Path)

		beforeHash, err := switchMatrixConfigHash(id)
		if err != nil {
			t.Fatal(err)
		}
		if err = b.prepareSwitchSourceCopy(ctx, id, c.Version); err != nil {
			t.Fatal("prepare source copy", err)
		}
		state := "stopped"
		if id == ids[0] {
			// Production Clone restarts an originally running source before
			// clearing its temporary configuration.
			f.states[id], f.containers[id], state = "active", "running", "running"
		}
		e := minecraftSwitchEvidence{OperationID: j.OperationID, Source: id, SourceRecord: &record, SourceContentChanged: true}
		if err = s.clearSwitchSourceContent(ctx, &e); err != nil {
			t.Fatal("clear temporary source configuration", err)
		}
		afterHash, err := switchMatrixConfigHash(id)
		if err != nil || afterHash != beforeHash {
			t.Fatal("source configuration no longer matches its original SHA256", err)
		}
		if err = s.withSwitchMatrixRuntime(ctx, j.OperationID, id, state, func(_ context.Context, current instanceRuntimeRecord) error {
			if !bytes.Equal(mustSwitchJSON(current), mustSwitchJSON(record)) {
				return errInstanceRecovery
			}
			return nil
		}); err != nil {
			t.Fatal("original source verification refused its committed receipt", err)
		}
		if state == "stopped" {
			if err = s.stopSwitchMatrixFixtureForRetirement(ctx, j.OperationID, id); err != nil {
				t.Fatal("could not return verified fixture to stopped state", err)
			}
		}
	}
	source, cloneID := originals[0], ids[1]
	if _, err = o.transition(j.OperationID, operationRunning, "switch", "Reviewing Switch."); err != nil {
		t.Fatal(err)
	}
	child, err := o.beginSwitchMatrixChild(j.OperationID, source.Runtime.ID, switchMatrixRunnerFingerprint())
	if err != nil {
		t.Fatal(err)
	}
	e := minecraftSwitchEvidence{Schema: 1, OperationID: child.OperationID, Source: source.Runtime.ID, SourceRecord: &source,
		Target: originals[1].Runtime.Software, Version: source.Runtime.Version, Action: "clone", Stage: "complete",
		CloneID: cloneID, CloneRecord: &originals[1], OriginallyRunning: true}
	if err = saveSwitchEvidence(e); err != nil {
		t.Fatal(err)
	}
	for _, step := range []struct {
		state operationState
		stage string
	}{{operationValidating, "validating"}, {operationRunning, "converting"}, {operationVerifying, "verifying"}, {operationSucceeded, "complete"}} {
		if _, err = o.transition(child.OperationID, step.state, step.stage, "Fixture Switch transaction."); err != nil {
			t.Fatal(err)
		}
	}
	substituted = ids[len(ids)-1]
	if _, err := s.closeSwitchMatrixCase(ctx, j.OperationID, 0); !errors.Is(err, errInstanceRecovery) {
		t.Fatal("cleanup accepted substituted fixture storage", err)
	}
	substituted = ""
	if _, err := s.closeSwitchMatrixCase(ctx, j.OperationID, 1); err != nil {
		t.Fatal("exact clone fixture cleanup failed", err)
	}
	o.holdSwitchMatrixRun(j.OperationID)
	cases := []switchMatrixCase{manifest.Cases[smoke]}
	if err = o.saveSwitchMatrixRun(switchMatrixRun{Schema: 1, ID: j.OperationID, State: "needs_attention", Cases: cases,
		Results: []switchMatrixCaseResult{{Outcome: matrixFailed, ChildOperation: child.OperationID, Original: "failed", Cleanup: "unresolved"}}}); err != nil {
		t.Fatal(err)
	}
	dir, envPath, quadletPath := b.paths(source.Runtime.ID)
	configPath := filepath.Join(dir, "config.json")
	legacy := source
	legacy.Firewall = nil
	if err = b.writeJSON(configPath, legacy, true); err != nil {
		t.Fatal(err)
	}
	reject := func() {
		t.Helper()
		before, err := switchMatrixConfigHash(source.Runtime.ID)
		if err != nil {
			t.Fatal(err)
		}
		if err = s.recoverSwitchMatrixRun(j.OperationID); !errors.Is(err, errInstanceRecovery) {
			t.Fatal("recovery accepted unverified evidence", err)
		}
		after, err := switchMatrixConfigHash(source.Runtime.ID)
		if err != nil || after != before || f.states[source.Runtime.ID] != "active" {
			t.Fatal("recovery modified a fixture before verifying ownership", err)
		}
	}
	changed := legacy
	changed.Runtime.MaxPlayers++
	if err = b.writeJSON(configPath, changed, true); err != nil {
		t.Fatal(err)
	}
	reject()
	if err = b.writeJSON(configPath, legacy, true); err != nil {
		t.Fatal(err)
	}
	substituted = source.Runtime.ID
	reject()
	substituted = ""
	replacedData = source.Runtime.ID
	reject()
	replacedData = ""
	backupReplaced = true
	reject()
	backupReplaced = false
	for _, path := range []string{envPath, quadletPath} {
		raw, err := b.readPrivate(path)
		if err != nil {
			t.Fatal(err)
		}
		if err = b.writePrivate(path, append(slices.Clone(raw), '\n'), true); err != nil {
			t.Fatal(err)
		}
		reject()
		if err = b.writePrivate(path, raw, true); err != nil {
			t.Fatal(err)
		}
	}
	unrelated := e
	unrelated.OperationID = j.OperationID
	if err = b.writeJSON(switchEvidencePath(child.OperationID), unrelated, true); err != nil {
		t.Fatal(err)
	}
	reject()
	if err = saveSwitchEvidence(e); err != nil {
		t.Fatal(err)
	}
	wrongSaved := source
	wrongSaved.Firewall = nil
	unrelated = e
	unrelated.SourceRecord = &wrongSaved
	if err = saveSwitchEvidence(unrelated); err != nil {
		t.Fatal(err)
	}
	reject()
	if err = saveSwitchEvidence(e); err != nil {
		t.Fatal(err)
	}
	if _, err = o.switchMatrixOwnedRuntime(ctx, j.OperationID, source.Runtime.ID); !errors.Is(err, errInstanceRecovery) {
		t.Fatal("legacy fixture passed strict ownership before repair", err)
	}
	shutdownHash = instanceHash(append(mustSwitchJSON(source), '\n'))
	if err = s.recoverSwitchMatrixRun(j.OperationID); err != nil {
		t.Fatal("exact legacy source recovery and retirement failed", err)
	}
	run, err := o.readSwitchMatrixRun(j.OperationID)
	if err != nil || run.State != "recovered" || run.Results[0].Outcome != matrixFailed ||
		run.Results[0].Original != "failed" || run.Results[0].Cleanup != "verified" {
		t.Fatal("recovery did not preserve the historical failure and verify cleanup", err)
	}
	completed, err := o.get(child.OperationID)
	if err != nil || completed.State != operationSucceeded {
		t.Fatal("recovery altered the completed Clone", err)
	}
	active, err := o.currentMinecraftMaintenance()
	if err != nil || active != nil {
		t.Fatal("verified closure retained the parent reservation", err)
	}
	for _, path := range paths {
		if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("owned fixture resource remained after retirement", err)
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
