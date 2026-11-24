package main

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestMigrationPortableDataCannotReplaceDestinationIdentityOrRuntimeOwnership(t *testing.T) {
	f := newInstanceActivationFixture(t)
	b := f.backend
	previous, err := b.readRecord(f.configs[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	dir, _, _ := b.paths(previous.Runtime.ID)
	config, err := b.readPrivate(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(previous.Storage.DataPath), 0700); err != nil {
		t.Fatal(err)
	}
	parent, err := b.privateDirectory(filepath.Dir(previous.Storage.DataPath), true)
	if err != nil {
		t.Fatal(err)
	}
	parent.Close()
	tx := filepath.Join(filepath.Dir(previous.Storage.DataPath), ".justvoxel-import-test")
	d, err := b.privateDirectory(tx, true)
	if err != nil {
		t.Fatal(err)
	}
	d.Close()
	evidence := map[string]any{"schema": 1, "instance_id": previous.Runtime.ID, "data_path": previous.Storage.DataPath, "transaction": tx, "config_identity": instanceHash(config), "data_identity": "1:2", "staged_identity": "verified", "source_identity": "sourceInstanceID=jv-d4e5f6", "was_running": true}
	if err = b.writeJSON(filepath.Join(tx, "evidence.json"), evidence, false); err != nil {
		t.Fatal(err)
	}
	portable := map[string]any{"instance_id": previous.Runtime.ID, "game_mode": "creative", "difficulty": "hard", "whitelist": false, "enforce_whitelist": false, "max_players": 25, "motd": "Imported server"}
	if err = b.writeJSON(filepath.Join(tx, "portable.json"), portable, false); err != nil {
		t.Fatal(err)
	}
	next, err := b.migrationPortableSettings(previous, tx)
	if err != nil {
		t.Fatal(err)
	}
	expected := previous
	expected.Runtime.GameMode = "creative"
	expected.Runtime.Difficulty = "hard"
	expected.Runtime.Whitelist = false
	expected.Runtime.EnforceWhitelist = false
	expected.Runtime.MaxPlayers = 25
	expected.Runtime.MOTD = "Imported server"
	if !reflect.DeepEqual(next, expected) {
		t.Fatal("portable source changed destination identity, ports, limits, storage, boot policy or runtime ownership")
	}
	for _, software := range []string{"fabric", "forge", "neoforge", "quilt"} {
		destination := previous
		destination.Runtime.Software = software
		destination.Runtime.BedrockEnabled = false
		destination.Runtime.ImageTag = "java21"
		destination.Runtime.Version = "1.21.3"
		destination.Runtime.LoaderVersion = "1.2.1"
		portable["source_software"], portable["version"], portable["loader_version"] = software, "1.21.2", "1.2.3"
		if err = b.writeJSON(filepath.Join(tx, "portable.json"), portable, true); err != nil {
			t.Fatal(err)
		}
		next, err := b.migrationPortableSettings(destination, tx)
		if err != nil || next.Runtime.Version != "1.21.2" || next.Runtime.LoaderVersion != "1.2.3" || next.Runtime.VersionPolicy != "pinned" {
			t.Fatalf("%s reviewed release lost: %v", software, err)
		}
	}
	destination := previous
	destination.Runtime.Software, destination.Runtime.PufferfishBuild = "pufferfish", 38
	portable["source_software"], portable["version"], portable["loader_version"], portable["pufferfish_build"] = "pufferfish", "1.21.10", "", 39
	if err = b.writeJSON(filepath.Join(tx, "portable.json"), portable, true); err != nil {
		t.Fatal(err)
	}
	next, err = b.migrationPortableSettings(destination, tx)
	if err != nil || next.Runtime.PufferfishBuild != 39 {
		t.Fatal("native Pufferfish build lost", err)
	}
	portable["instance_id"] = "jv-d4e5f6"
	if err = b.writeJSON(filepath.Join(tx, "portable.json"), portable, true); err != nil {
		t.Fatal(err)
	}
	if _, err = b.migrationPortableSettings(previous, tx); err == nil {
		t.Fatal("archive source identity replaced selected destination")
	}
	delete(evidence, "instance_id")
	if err = b.writeJSON(filepath.Join(tx, "evidence.json"), evidence, true); err != nil {
		t.Fatal(err)
	}
	if _, err = b.migrationPortableSettings(previous, tx); err == nil {
		t.Fatal("missing durable identity was guessed")
	}
}

func TestMigrationPreflightLeavesRunningDestinationIntact(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("exercises root-only version commands")
	}
	f := newInstanceActivationFixture(t)
	f.activateBoth()
	b := f.backend
	original := instanceRuntimes
	instanceRuntimes = b
	t.Cleanup(func() { instanceRuntimes = original })
	id := f.configs[0].ID
	previous, err := b.readRecord(id)
	if err != nil {
		t.Fatal(err)
	}
	dir, _, _ := b.paths(id)
	config, err := b.readPrivate(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	tx := filepath.Join(filepath.Dir(previous.Storage.DataPath), ".justvoxel-import-test")
	if err := os.Mkdir(tx, 0700); err != nil {
		t.Fatal(err)
	}
	world := filepath.Join(previous.Storage.DataPath, "level.dat")
	if err := os.WriteFile(world, []byte("destination"), 0600); err != nil {
		t.Fatal(err)
	}
	evidence := map[string]any{"schema": 1, "instance_id": id, "transaction": tx, "data_path": previous.Storage.DataPath, "config_identity": instanceHash(config)}
	if err := b.writeJSON(filepath.Join(tx, "evidence.json"), evidence, false); err != nil {
		t.Fatal(err)
	}
	c := previous.Runtime
	args := []string{"instance-version-migration-preflight", id, c.Software, c.ImageTag, "1.21.4", "pinned", c.ImageDigest, "--migration", tx}
	f.states[id], f.containers[id] = "active", "running"
	for _, ready := range []bool{false, true} {
		if ready {
			portable := map[string]any{"instance_id": id, "source_software": c.Software, "version": args[4], "game_mode": c.GameMode, "difficulty": c.Difficulty, "max_players": c.MaxPlayers, "motd": c.MOTD}
			if err := b.writeJSON(filepath.Join(tx, "portable.json"), portable, false); err != nil {
				t.Fatal(err)
			}
		}
		err := instanceVersionCommand(context.Background(), args)
		if (err == nil) != ready {
			t.Fatalf("portable settings ready=%t, preflight returned %v", ready, err)
		}
		after, err := b.readRecord(id)
		if err != nil || !reflect.DeepEqual(after, previous) || b.verifyLive(previous) != nil {
			t.Fatal("preflight changed the committed runtime", err)
		}
		data, err := os.ReadFile(world)
		if err != nil || string(data) != "destination" || f.states[id] != "active" || f.containers[id] != "running" {
			t.Fatal("preflight disrupted the destination", err)
		}
	}
}
