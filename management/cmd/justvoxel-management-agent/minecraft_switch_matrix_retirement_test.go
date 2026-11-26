package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestSwitchMatrixFixtureRetirement(t *testing.T) {
	for _, scenario := range []string{"managed_fabric", "foreign_instance", "interrupted_metadata", "interrupted_delete", "unsafe_metadata", "substituted_storage", "unresolved_switch", "unresolved_restore", "unexpected_file",
		"created_before_binding", "candidate_mismatch", "binding_before_backup", "binding_after_backup", "binding_publish_retry", "unproven_backup"} {
		t.Run(scenario, func(t *testing.T) {
			d := newDeleteFixtureWithPlacement(t, true)
			b := d.f.backend
			id, sibling := d.f.configs[0].ID, d.f.configs[1].ID
			o := matrixTestStore(t, b)
			ctx := context.Background()
			j := matrixTestBegin(t, o, "retirement")
			if _, err := b.identities().renameInstance(id, "Matrix Fixture", "paper"); err != nil {
				t.Fatal(err)
			}
			record, err := b.readRecord(id)
			if err != nil {
				t.Fatal(err)
			}
			record.Runtime.Software = "fabric"
			record.Runtime.BedrockEnabled = true
			record.Runtime.BedrockPort = 19132
			record.Runtime.LoaderVersion = "0.16.9"
			record.Runtime.VersionPolicy = "pinned"
			record.Runtime.ImageTag = "java21"
			record.Runtime.StartOnBoot = false
			dir, _, _ := b.paths(id)
			if err = b.writeJSON(filepath.Join(dir, "config.json"), record, true); err != nil {
				t.Fatal(err)
			}
			// Persist fixture ownership in the test's private operation store.
			// These runtime/storage command stubs only exercise the cleanup
			// protocol. The backup and mod workers have separate shell fixtures.
			o.mu.Lock()
			j.MatrixFixtures = []string{id}
			err = o.persist(j)
			o.operations[j.OperationID] = j
			o.mu.Unlock()
			if err != nil {
				t.Fatal(err)
			}
			if staged, err := b.privateDirectory(b.staging, true); err != nil {
				t.Fatal(err)
			} else {
				staged.Close()
			}
			hash, err := switchMatrixConfigHash(id)
			if err != nil {
				t.Fatal(err)
			}
			placement := switchMatrixBackups{Root: [2]uint64{1, 10}, Directory: [2]uint64{1, 11}, Files: map[string]switchMatrixBackupFile{}}
			owned := switchMatrixFixture{Parent: j.OperationID, ID: id, Stage: "owned", Name: "Matrix Fixture", ConfigSHA256: hash,
				DataIdentity: "1:2", DataParent: [2]uint64{1, 5}, Storage: record.Storage, Placement: record.Backups, Backups: placement,
				UID: record.Runtime.UID, GID: record.Runtime.GID}
			switch scenario {
			case "created_before_binding", "candidate_mismatch":
				owned.Stage, owned.ConfigSHA256 = "reserved", ""
				owned.DataIdentity, owned.DataParent = "", [2]uint64{}
				owned.CandidateSHA256 = instanceHash(mustSwitchJSON(record.Runtime))
				if scenario == "candidate_mismatch" {
					owned.CandidateSHA256 = strings.Repeat("a", 64)
				}
				owned.Backups = switchMatrixBackups{}
			case "binding_before_backup", "binding_publish_retry", "unproven_backup":
				owned.Stage, owned.Backups = "binding", switchMatrixBackups{}
			case "binding_after_backup":
				owned.Stage = "binding"
			}
			if err = owned.save(false); err != nil {
				t.Fatal(err)
			}
			modPath := filepath.Join(dir, "crossplay-mods.json")
			fresh := scenario == "created_before_binding" || scenario == "binding_before_backup"
			if !fresh {
				if err = os.WriteFile(modPath, []byte(`{"schema":1,"platform":"fabric","artifacts":{"fabric-api.jar":"fixture"}}`), 0600); err != nil {
					t.Fatal(err)
				}
			}
			unrelated := filepath.Join(t.TempDir(), sibling+".backup")
			if err = os.WriteFile(unrelated, []byte("preserve unrelated archive"), 0600); err != nil {
				t.Fatal(err)
			}
			verifications, backupRetirements, absenceChecks := 0, 0, 0
			preparations, publications := 0, 0
			base := b.command
			b.command = func(ctx context.Context, command string, args ...string) ([]byte, error) {
				if command == "nsenter" && len(args) >= 4 {
					switch args[2] {
					case matrixCrossplayOwnershipHelper:
						verifications++
						if len(args) != 8 || args[3] != "retire-verify" || args[4] != "fabric" ||
							args[5] != record.Storage.DataPath || args[6] != modPath || args[7] != "true" {
							t.Fatal("mod verification escaped the owned fixture", args)
						}
						if scenario == "unsafe_metadata" {
							return nil, errInstanceRecovery
						}
						return nil, nil
					case instanceSwitchRetireHelper:
						if args[4] != id || (args[3] != "unrestored" && args[3] != "matrix-empty") {
							t.Fatal("unexpected Restore storage cleanup", args)
						}
						return nil, nil
					case instanceDeleteHelper:
						if args[3] == "matrix-retry" {
							return nil, nil
						}
						if args[3] == "matrix-inspect" {
							if scenario == "substituted_storage" {
								return []byte(`{"identity":"1:2","parent_identity":[1,6]}`), nil
							}
							return []byte(`{"identity":"1:2","parent_identity":[1,5]}`), nil
						}
					}
				}
				return base(ctx, command, args...)
			}
			oldBackup := instanceBackupStorageCommand
			t.Cleanup(func() { instanceBackupStorageCommand = oldBackup })
			instanceBackupStorageCommand = func(_ context.Context, command string, args ...string) ([]byte, error) {
				if command != "nsenter" || len(args) != 6 || args[2] != instanceBackupHelper || args[4] != id {
					t.Fatal("backup operation escaped owned fixture", args)
				}
				switch args[3] {
				case "matrix-prepare":
					preparations++
					return mustSwitchJSON(placement), nil
				case "matrix-bind":
					publications++
					receipt, err := readSwitchMatrixFixture(j.OperationID, id)
					if err != nil || receipt.Stage != "binding" || receipt.Backups.Directory != placement.Directory {
						t.Fatal("backup inode was not persisted before publication", err)
					}
					if publications == 1 {
						return nil, errInstanceRecovery
					}
					return nil, nil
				case "matrix-absent":
					absenceChecks++
					if scenario == "unproven_backup" {
						return nil, errInstanceRecovery
					}
					return nil, nil
				case "matrix-inventory":
					return mustSwitchJSON(placement), nil
				case "matrix-check":
					return nil, nil
				case "matrix-retire":
					backupRetirements++
					if scenario == "binding_after_backup" {
						receipt, err := readSwitchMatrixFixture(j.OperationID, id)
						if err != nil || receipt.Stage != "intent" || receipt.Backups.Directory != placement.Directory {
							t.Fatal("prepared inode was lost from retirement intent", err)
						}
					}
					if (scenario == "interrupted_metadata" || scenario == "binding_after_backup") && backupRetirements == 1 {
						// This failure occurs after the metadata unlink, but
						// before the next durable retirement checkpoint.
						return nil, errInstanceRecovery
					}
					return nil, nil
				}
				t.Fatal("unexpected backup action", args)
				return nil, errInstanceRecovery
			}
			switch scenario {
			case "foreign_instance":
				if err = o.retireSwitchMatrixFixture(ctx, j.OperationID, sibling); err == nil || verifications != 0 {
					t.Fatal("foreign instance accepted")
				}
				return
			case "interrupted_delete":
				d.failRemove = true
			case "binding_publish_retry":
				if _, err = o.transition(j.OperationID, operationNeedsAttention, "interrupted", "Fixture binding interrupted."); err != nil {
					t.Fatal(err)
				}
				if err = o.bindSwitchMatrixFixture(ctx, j.OperationID, id); err == nil || preparations != 1 || publications != 1 {
					t.Fatal("publication interruption did not retain the prepared namespace", err)
				}
				if err = o.bindSwitchMatrixFixture(ctx, j.OperationID, id); err != nil || preparations != 1 || publications != 2 {
					t.Fatal("backup binding did not retry the persisted namespace", err)
				}
			case "unresolved_switch":
				if _, err = o.transition(j.OperationID, operationRunning, "switch", "Switching fixture."); err != nil {
					t.Fatal(err)
				}
				if _, err = o.beginSwitchMatrixChild(j.OperationID, id, "sha256:"+strings.Repeat("a", 64)); err != nil {
					t.Fatal(err)
				}
				if _, err = o.transition(j.OperationID, operationNeedsAttention, "interrupted", "Switch recovery required."); err != nil {
					t.Fatal(err)
				}
			case "unresolved_restore":
				r := instanceRestoreJournal{Schema: 1, ID: id, Stage: "swapped", ConfigSHA256: hash,
					Transaction: strings.Repeat("a", 32), Original: []uint64{1, 2}, Parent: []uint64{1, 5}, Archive: "Matrix Fixture-backup-fabric-2026-01-01-0000.tar.gz"}
				if err = b.writeJSON(filepath.Join(dir, "restore-journal.json"), r, false); err != nil {
					t.Fatal(err)
				}
			case "unexpected_file":
				if err = os.WriteFile(filepath.Join(dir, "foreign.json"), []byte("preserve"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			err = o.retireSwitchMatrixFixture(ctx, j.OperationID, id)
			if slices.Contains([]string{"unsafe_metadata", "substituted_storage", "unresolved_switch", "unresolved_restore", "unexpected_file", "candidate_mismatch", "unproven_backup"}, scenario) {
				if err == nil || backupRetirements != 0 || d.removed {
					t.Fatal("uncertain retirement was accepted")
				}
				if _, err = os.Stat(modPath); err != nil {
					t.Fatal("metadata removed before validation", err)
				}
				if scenario == "candidate_mismatch" || scenario == "unproven_backup" {
					current, err := o.get(j.OperationID)
					if err != nil || current.State != operationNeedsAttention {
						t.Fatal("ambiguous interruption released attention", current.State, err)
					}
				}
				return
			}
			if (scenario == "binding_before_backup" || scenario == "created_before_binding") && (absenceChecks != 2 || backupRetirements != 0) {
				t.Fatal("retirement should verify absent backups without creating a namespace")
			}
			if scenario == "interrupted_metadata" || scenario == "interrupted_delete" || scenario == "binding_after_backup" {
				if err == nil || d.removed {
					t.Fatal("interruption did not preserve recovery evidence")
				}
				if _, e := os.Stat(modPath); !errors.Is(e, os.ErrNotExist) {
					t.Fatalf("interruption checkpoint missed metadata unlink: %v", e)
				}
				d.failRemove = false
				err = o.retireSwitchMatrixFixture(ctx, j.OperationID, id)
			}
			if err != nil || (!fresh && verifications < 2) || !d.removed {
				t.Fatalf("retirement did not use verified normal deletion: %v", err)
			}
			if err = o.retireSwitchMatrixFixture(ctx, j.OperationID, id); err != nil {
				t.Fatal("completed retirement was not idempotent", err)
			}
			if got, err := os.ReadFile(unrelated); err != nil || string(got) != "preserve unrelated archive" {
				t.Fatalf("unrelated archive changed: %q %v", got, err)
			}
		})
	}
}

func TestSwitchMatrixUnusedReservationCanCloseAfterRestart(t *testing.T) {
	d := newDeleteFixtureWithPlacement(t, true)
	b := d.f.backend
	o := matrixTestStore(t, b)
	j := matrixTestBegin(t, o, "fixtures")
	id := "jv-unused"
	template, err := b.readRecord(d.f.configs[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	candidate := template
	candidate.Runtime.ID, candidate.Runtime.StartOnBoot = id, false
	candidate.Storage.DataPath = filepath.Join(candidate.Storage.Root, "minecraft-instances", id, "data")
	placement := *template.Backups
	placement.Path = filepath.Join(placement.Root, id)
	candidate.Backups = &placement
	if err := o.reserveSwitchMatrixFixture(j.OperationID, candidate, operationJournal{}); err != nil {
		t.Fatal(err)
	}
	if err := o.close(); err != nil {
		t.Fatal(err)
	}
	o, err = openOperationStore(o.baseDir)
	if err != nil {
		t.Fatal(err)
	}
	defer o.close()
	current, err := o.get(j.OperationID)
	if err != nil || current.State != operationNeedsAttention {
		t.Fatal("missing interrupted reservation", current.State, err)
	}
	base := b.command
	dataChecks := 0
	b.command = func(ctx context.Context, command string, args ...string) ([]byte, error) {
		if command == "systemctl" && len(args) == 5 && args[1] == "--property=LoadState" &&
			(args[4] == instanceBackupUnit(id, "timer") || args[4] == instanceBackupUnit(id, "service")) {
			return []byte("not-found\n"), nil
		}
		if command == "nsenter" && len(args) >= 7 && args[2] == instanceSwitchRetireHelper {
			dataChecks++
			if args[3] != "matrix-absent" || args[4] != id || args[6] != candidate.Storage.DataPath {
				t.Fatal("unexpected reserved data check", args)
			}
			return nil, nil
		}
		return base(ctx, command, args...)
	}
	oldBackup := instanceBackupStorageCommand
	t.Cleanup(func() { instanceBackupStorageCommand = oldBackup })
	backupChecks := 0
	instanceBackupStorageCommand = func(_ context.Context, command string, args ...string) ([]byte, error) {
		backupChecks++
		if command != "nsenter" || len(args) != 6 || args[2] != instanceBackupHelper || args[3] != "matrix-absent" ||
			args[4] != id || args[5] != j.OperationID {
			t.Fatal("unexpected reserved backup check", args)
		}
		if backupChecks == 1 {
			return nil, errInstanceRecovery
		}
		return nil, nil
	}
	ctx := context.Background()
	if err = o.retireSwitchMatrixFixture(ctx, j.OperationID, id); err == nil {
		t.Fatal("retired without checking both storage locations")
	}
	if _, err = o.transition(j.OperationID, operationResolved, "resolved", "Verified interruption."); err == nil {
		t.Fatal("closed with unverified storage")
	}
	if err = o.retireSwitchMatrixFixture(ctx, j.OperationID, id); err != nil {
		t.Fatal(err)
	}
	if err = o.retireSwitchMatrixFixture(ctx, j.OperationID, id); err != nil || dataChecks != 2 || backupChecks != 2 || d.removed {
		t.Fatal("verified absence did not produce an idempotent retirement", err)
	}
	if _, err = o.transition(j.OperationID, operationResolved, "resolved", "Verified interruption."); err != nil {
		t.Fatal(err)
	}
	if current, err := o.currentMinecraftMaintenance(); err != nil || current != nil || o.switchMatrixExclusion != nil {
		t.Fatal("verified interruption retained the matrix reservation", current, err)
	}
}
