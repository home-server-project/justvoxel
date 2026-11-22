package main

import (
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func switchWorldFixture(t *testing.T, data, version, level string) map[string]string {
	t.Helper()
	files := map[string]string{"server.properties": "level-name=" + level + "\n", filepath.Join(level, "level.dat"): "world identity", filepath.Join(level, "region", "r.0.0.mca"): "overworld chunks"}
	if java25GameReleasePattern.MatchString(version) {
		files[filepath.Join(level, "dimensions", "minecraft", "overworld", "region", "r.0.0.mca")] = "modern overworld chunks"
		for _, name := range []string{"game_rules.dat", "scheduled_events.dat", "wandering_trader.dat", "weather.dat", "world_gen_settings.dat"} {
			files[filepath.Join(level, "dimensions", "minecraft", "overworld", "data", "minecraft", name)] = "source " + name
		}
	} else {
		files[filepath.Join(level+"_nether", "DIM-1", "region", "r.0.0.mca")] = "nether chunks"
		files[filepath.Join(level+"_the_end", "DIM1", "region", "r.0.0.mca")] = "end chunks"
	}
	for path, content := range files {
		path = filepath.Join(data, path)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return files
}

func switchWorldSnapshot(t *testing.T, data string) map[string]string {
	t.Helper()
	files := map[string]string{}
	if err := filepath.WalkDir(data, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(data, path)
		files[rel] = string(raw)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	return files
}

func TestSwitchWorldRelocations(t *testing.T) {
	for _, version := range []string{"26.1", "26.1.2", "1.21.4"} {
		for _, level := range []string{"world", "Family World"} {
			t.Run(version+"/"+level, func(t *testing.T) {
				data := t.TempDir()
				before := switchWorldFixture(t, data, version, level)
				moves, err := inspectSwitchWorld(data, version)
				if err != nil {
					t.Fatal(err)
				}
				expected := map[string]string{}
				for path, value := range before {
					expected[path] = value
				}
				for _, move := range moves {
					for path, value := range before {
						if path == move.From || strings.HasPrefix(path, move.From+string(filepath.Separator)) {
							delete(expected, path)
							expected[move.To+strings.TrimPrefix(path, move.From)] = value
						}
					}
				}
				if err = transformSwitchWorld(data, version); err != nil {
					t.Fatal(err)
				}
				if got := switchWorldSnapshot(t, data); !reflect.DeepEqual(got, expected) {
					t.Fatalf("paths or content changed: got=%v want=%v", got, expected)
				}
				if java25GameReleasePattern.MatchString(version) {
					if len(moves) != 5 || moves[0].From != filepath.Join(level, "dimensions", "minecraft", "overworld", "data", "minecraft", "game_rules.dat") || moves[0].To != filepath.Join(level, "data", "minecraft", "game_rules.dat") {
						t.Fatal("wrong modern layout")
					}
				} else if len(moves) != 2 || moves[0].From != filepath.Join(level+"_nether", "DIM-1") || moves[0].To != filepath.Join(level, "DIM-1") {
					t.Fatal("wrong legacy layout")
				}
			})
		}
	}
}

func TestSwitchWorldRefusesMissingAmbiguousAndUnsafeLayouts(t *testing.T) {
	for _, kind := range []string{"missing", "destination", "mixed", "duplicate level", "escaped level", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			data := t.TempDir()
			switchWorldFixture(t, data, "26.1", "Family")
			src := filepath.Join(data, "Family", "dimensions", "minecraft", "overworld", "data", "minecraft", "weather.dat")
			switch kind {
			case "missing":
				if err := os.Remove(src); err != nil {
					t.Fatal(err)
				}
			case "destination":
				dir := filepath.Join(data, "Family", "data", "minecraft")
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, "weather.dat"), []byte("conflicting data"), 0644); err != nil {
					t.Fatal(err)
				}
			case "mixed":
				if err := os.Mkdir(filepath.Join(data, "Family_nether"), 0755); err != nil {
					t.Fatal(err)
				}
			case "duplicate level":
				if err := os.WriteFile(filepath.Join(data, "server.properties"), []byte("level-name=Family\nlevel-name=other\n"), 0644); err != nil {
					t.Fatal(err)
				}
			case "escaped level":
				if err := os.WriteFile(filepath.Join(data, "server.properties"), []byte("level-name=../outside\n"), 0644); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Remove(src); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filepath.Join(data, "Family", "level.dat"), src); err != nil {
					t.Fatal(err)
				}
			}
			before := switchWorldSnapshot(t, data)
			if err := transformSwitchWorld(data, "26.1"); err == nil {
				t.Fatal("unsafe layout accepted")
			}
			if !reflect.DeepEqual(before, switchWorldSnapshot(t, data)) {
				t.Fatal("failed preflight changed world data")
			}
		})
	}
}

func TestSwitchFailedCloneTransformationPreservesOriginal(t *testing.T) {
	source, clone := t.TempDir(), filepath.Join(t.TempDir(), "clone")
	original := switchWorldFixture(t, source, "26.1", "Family")
	if err := os.CopyFS(clone, os.DirFS(source)); err != nil {
		t.Fatal(err)
	}
	if err := transformSwitchWorld(clone, "26.1"); err != nil {
		t.Fatal(err)
	}
	// A target can modify the transformed copy and then fail readiness.
	if err := os.WriteFile(filepath.Join(clone, "Family", "data", "minecraft", "weather.dat"), []byte("failed target runtime"), 0644); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(original, switchWorldSnapshot(t, source)) {
		t.Fatal("failed target modified source paths or bytes")
	}
	if _, err := os.Stat(filepath.Join(clone, "Family", "data", "minecraft", "world_gen_settings.dat")); err != nil {
		t.Fatal("clone transformation was not exercised", err)
	}
}

func TestSwitchWorldRollbackRestoresColdBackupAfterFailedConversion(t *testing.T) {
	f := switchPlannerFixture(t)
	source, err := f.backend.readRecord("jv-a1b2c3")
	if err != nil {
		t.Fatal(err)
	}
	data := source.Storage.DataPath
	switchWorldFixture(t, data, "1.21.4", "Family")
	before := switchWorldSnapshot(t, data)
	cold := filepath.Join(t.TempDir(), "cold")
	if err = os.CopyFS(cold, os.DirFS(data)); err != nil {
		t.Fatal(err)
	}
	if err = transformSwitchWorld(data, "1.21.4"); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(data, "Family", "DIM-1", "region", "r.0.0.mca"), []byte("failed runtime wrote chunks"), 0644); err != nil {
		t.Fatal(err)
	}
	oldCommand, oldLock := instanceBackupStorageCommand, pluginRuntimeLockPath
	t.Cleanup(func() { instanceBackupStorageCommand = oldCommand; pluginRuntimeLockPath = oldLock })
	pluginRuntimeLockPath = filepath.Join(t.TempDir(), "maintenance.lock")
	archive := "Family-backup-paper-2026-11-16-1200.tar.gz"
	actions := []string{}
	instanceBackupStorageCommand = func(_ context.Context, command string, args ...string) ([]byte, error) {
		if command != "nsenter" || len(args) != 8 || args[2] != instanceRestoreHelper || args[4] != source.Runtime.ID || args[5] != archive {
			t.Fatal("rollback lost source/archive authority", command, args)
		}
		action := args[3]
		actions = append(actions, action)
		if action == "plan" {
			return json.Marshal(instanceRestoreInspection{Metadata: instanceBackupMetadata{ID: source.Runtime.ID, Software: "paper", Version: "1.21.4", Archive: archive}, Integrity: "verified", Fingerprint: instanceHash([]byte("cold backup"))})
		}
		if action == "activate" {
			// Model the existing restore engine's inode-preserving directory
			// activation, including the failed target copy retained for rollback.
			if err := os.Rename(data, data+".failed"); err != nil {
				return nil, err
			}
			if err := os.CopyFS(data, os.DirFS(cold)); err != nil {
				return nil, err
			}
		}
		return nil, nil
	}
	e := minecraftSwitchEvidence{Schema: 1, OperationID: strings.Repeat("a", 32), Source: source.Runtime.ID, SourceRecord: &source, Target: "vanilla", Action: "convert", Version: "1.21.4", ColdArchive: archive, WorldChanged: true}
	lock, err := acquirePluginMaintenanceLock()
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	s := surfaceTestServer(t, roleAdministrator)
	if err = s.restoreSwitchWorld(context.Background(), &e, lock); err != nil {
		t.Fatal(err)
	}
	if e.WorldChanged || !reflect.DeepEqual(before, switchWorldSnapshot(t, data)) {
		t.Fatal("cold backup paths/bytes were not restored")
	}
	expected := []string{"plan", "prepare", "shutdown", "activate", "verifying", "completed"}
	if !reflect.DeepEqual(actions, expected) {
		t.Fatalf("did not use existing restore transaction: %v", actions)
	}
	if _, err = os.Stat(filepath.Join(data+".failed", "Family", "DIM-1", "region", "r.0.0.mca")); err != nil {
		t.Fatal("failed target evidence was discarded", err)
	}
}

func TestSwitchKeepsTerminalRestoreBindingThroughTemporaryConfig(t *testing.T) {
	for _, stage := range []string{"completed", "rolled_back", "runtime_verification"} {
		t.Run(stage, func(t *testing.T) {
			f := switchPlannerFixture(t)
			source, err := f.backend.readRecord("jv-a1b2c3")
			if err != nil {
				t.Fatal(err)
			}
			raw, err := f.backend.readPrivate(filepath.Join(f.backend.root, source.Runtime.ID, "config.json"))
			if err != nil {
				t.Fatal(err)
			}
			journal := instanceRestoreJournal{Schema: 1, ID: source.Runtime.ID, Transaction: strings.Repeat("b", 32), ConfigSHA256: instanceHash(raw), Parent: []uint64{1, 2}, Original: []uint64{1, 3}, Replacement: []uint64{1, 4}, Archive: restoreTestArchive, Fingerprint: instanceHash([]byte("restore evidence")), Stage: stage}
			if err = f.backend.writeJSON(filepath.Join(f.backend.root, source.Runtime.ID, "restore-journal.json"), journal, true); err != nil {
				t.Fatal(err)
			}
			e := minecraftSwitchEvidence{Source: source.Runtime.ID, SourceRecord: &source, SourceRestoreJournal: &journal, Target: "vanilla", Version: "1.21.4"}
			if err = f.backend.prepareSwitchSourceCopy(context.Background(), source.Runtime.ID, "1.21.4"); err != nil {
				t.Fatal(err)
			}
			if err = rebindSwitchSourceRestore(context.Background(), &e); err != nil {
				t.Fatal(err)
			}
			after, err := f.backend.restoreJournal(source.Runtime.ID)
			if err != nil {
				t.Fatal(err)
			}
			raw, err = f.backend.readPrivate(filepath.Join(f.backend.root, source.Runtime.ID, "config.json"))
			if err != nil {
				t.Fatal(err)
			}
			if restoreTerminal(stage) {
				if after.ConfigSHA256 != instanceHash(raw) {
					t.Fatal("terminal evidence stranded by Switch config change")
				}
				after.ConfigSHA256 = journal.ConfigSHA256
				if !reflect.DeepEqual(after, journal) {
					t.Fatal("Switch rewrote restore identity or stage")
				}
			} else if !reflect.DeepEqual(after, journal) {
				t.Fatal("active recovery evidence was modified")
			}
		})
	}
}

func TestSwitchInPlaceRuntimeFailureRollsBackWorldAndSoftware(t *testing.T) {
	f := switchPlannerFixture(t)
	source, err := f.backend.readRecord("jv-a1b2c3")
	if err != nil {
		t.Fatal(err)
	}
	switchWorldFixture(t, source.Storage.DataPath, "1.21.4", "Family")
	original := switchWorldSnapshot(t, source.Storage.DataPath)
	plan, err := planMinecraftSwitch(context.Background(), source.Runtime.ID, "vanilla")
	if err != nil || !plan.Convert {
		t.Fatal("missing reviewed conversion", plan, err)
	}
	oldStorage, oldRuntime, oldLock := instanceBackupStorageCommand, pluginInstallCommand, pluginRuntimeLockPath
	t.Cleanup(func() {
		instanceBackupStorageCommand = oldStorage
		pluginInstallCommand = oldRuntime
		pluginRuntimeLockPath = oldLock
	})
	pluginRuntimeLockPath = filepath.Join(t.TempDir(), "maintenance.lock")
	cold := filepath.Join(t.TempDir(), "cold")
	archive := instanceBackupMetadata{Schema: 1, ID: source.Runtime.ID, Software: "paper", Version: "1.21.4", Archive: "Family-backup-paper-2026-11-16-1200.tar.gz"}
	backedUp, restored, verified := false, false, false
	instanceBackupStorageCommand = func(_ context.Context, command string, args ...string) ([]byte, error) {
		if command != "nsenter" || len(args) < 5 || args[4] != source.Runtime.ID {
			t.Fatal("foreign rollback storage", command, args)
		}
		action := args[3]
		if args[2] == instanceBackupHelper {
			switch action {
			case "preflight":
				return nil, nil
			case "inventory":
				if backedUp {
					return json.Marshal([]instanceBackupMetadata{archive})
				}
				return []byte("[]"), nil
			case "create":
				if err := os.CopyFS(cold, os.DirFS(source.Storage.DataPath)); err != nil {
					return nil, err
				}
				backedUp = true
				return nil, nil
			}
		}
		if args[2] != instanceRestoreHelper || args[5] != archive.Archive {
			t.Fatal("incorrect cold archive", args)
		}
		if action == "plan" {
			return json.Marshal(instanceRestoreInspection{Metadata: archive, Integrity: "verified", Fingerprint: instanceHash([]byte("cold rollback"))})
		}
		if action == "activate" {
			if !verified {
				t.Fatal("rollback was not triggered by failed target verification")
			}
			if err := os.Rename(source.Storage.DataPath, source.Storage.DataPath+".failed"); err != nil {
				return nil, err
			}
			if err := os.CopyFS(source.Storage.DataPath, os.DirFS(cold)); err != nil {
				return nil, err
			}
			restored = true
		}
		return nil, nil
	}
	pluginInstallCommand = func(_ context.Context, command string, args ...string) ([]byte, error) {
		if command == "systemctl" {
			switch args[0] {
			case "start":
				f.states[source.Runtime.ID] = "active"
				f.containers[source.Runtime.ID] = "running"
				return nil, nil
			case "is-active":
				return []byte(f.states[source.Runtime.ID]), nil
			}
		}
		if command == "podman" {
			return []byte("There are 0 of a max of 20 players online:"), nil
		}
		if strings.HasSuffix(command, "verify-minecraft-stack") {
			current, readErr := f.backend.readRecord(source.Runtime.ID)
			if readErr != nil || current.Runtime.Software != "vanilla" {
				t.Fatal("failure did not exercise target runtime", readErr)
			}
			path := filepath.Join(source.Storage.DataPath, "Family", "DIM1", "region", "r.0.0.mca")
			raw, readErr := os.ReadFile(path)
			if readErr != nil || string(raw) != "end chunks" {
				t.Fatal("runtime started before relocation", readErr)
			}
			if err := os.WriteFile(path, []byte("target runtime changed End chunks"), 0644); err != nil {
				t.Fatal(err)
			}
			verified = true
			return nil, os.ErrInvalid
		}
		t.Fatal("unexpected runtime command", command, args)
		return nil, os.ErrInvalid
	}
	s := surfaceTestServer(t, roleAdministrator)
	e := minecraftSwitchEvidence{Schema: 1, OperationID: strings.Repeat("a", 32), Source: source.Runtime.ID, SourceRecord: &source, Target: "vanilla", Version: "1.21.4", Action: "convert"}
	lock, err := acquirePluginMaintenanceLock()
	if err != nil {
		t.Fatal(err)
	}
	result := s.convertMinecraftSwitch(context.Background(), &e, plan, lock)
	if closeErr := lock.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}
	if result == nil || !verified || !e.WorldChanged || e.ColdArchive != archive.Archive {
		t.Fatal("failure lost durable cold rollback authority", result, e)
	}
	if err = s.recoverMinecraftSwitch(context.Background(), &e); err != nil {
		t.Fatal(err)
	}
	after, err := f.backend.readRecord(source.Runtime.ID)
	if err != nil || !restored || e.WorldChanged || after.Runtime.Software != "paper" || f.states[source.Runtime.ID] != "inactive" {
		t.Fatal("original runtime was not recovered", after, e, err)
	}
	if !reflect.DeepEqual(original, switchWorldSnapshot(t, source.Storage.DataPath)) {
		t.Fatal("failed in-place conversion changed original world paths or bytes")
	}
}
