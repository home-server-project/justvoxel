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
	j := matrixTestBegin(t, o, "fixtures")
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
	substituted := ""
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
					return mustSwitchJSON(switchMatrixDataIdentity{Data: dataIdentity(path), Parent: parent}), nil
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
		case "matrix-bind", "matrix-retire":
			receipt, err := readSwitchMatrixFixture(j.OperationID, id)
			if err != nil || receipt.Backups.Root != inode(backupRoot) || receipt.Backups.Directory != inode(path) {
				t.Fatal("backup publication or retirement lacked exact ownership", err)
			}
			if args[3] == "matrix-retire" {
				return nil, os.Remove(path)
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
	paper := slices.IndexFunc(manifest.Cases, func(c switchMatrixCase) bool {
		return c.Source == "paper" && c.Family == switchMatrixModern && !c.Crossplay && c.Disposition == ""
	})
	if paper < 0 {
		t.Fatal("expected an executable Paper source")
	}
	passwords := map[string]bool{}
	var ids []string
	var paths []string
	foreignBefore, err := b.readRecord(f.configs[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []switchMatrixCase{manifest.Cases[smoke], manifest.Cases[paper]} {
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
			bytes.Contains(quadlet, []byte("[Install]")) || len(record.Firewall) != 0 {
			t.Fatal("production environment, Quadlet or service contract was not published", envErr, quadletErr, statusErr)
		}
		ids = append(ids, id)
		paths = append(paths, envPath, quadletPath, record.Storage.DataPath, record.Backups.Path)
	}
	substituted = ids[len(ids)-1]
	if _, err := s.closeSwitchMatrixCase(ctx, j.OperationID, 0); !errors.Is(err, errInstanceRecovery) {
		t.Fatal("cleanup accepted substituted fixture storage", err)
	}
	substituted = ""
	if _, err := s.closeSwitchMatrixCase(ctx, j.OperationID, 0); err != nil {
		t.Fatal("exact fixture cleanup failed", err)
	}
	if err := o.checkSwitchMatrixRunClosure(j.OperationID); err != nil {
		t.Fatal("parent closure could not be verified", err)
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
