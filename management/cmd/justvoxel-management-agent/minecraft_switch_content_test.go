package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSwitchLoaderEvidenceUsesExactBytesAndDependencies(t *testing.T) {
	for _, pair := range [][2]string{{"fabric", "quilt"}, {"quilt", "fabric"}, {"forge", "neoforge"}, {"neoforge", "forge"}, {"fabric", "forge"}} {
		for _, status := range []string{"verified", "incompatible", "unknown", "missing dependency", "wrong hash", "wrong version"} {
			t.Run(pair[0]+"/"+pair[1]+"/"+status, func(t *testing.T) {
				f := switchPlannerFixture(t)
				record, err := f.backend.readRecord("jv-a1b2c3")
				if err != nil {
					t.Fatal(err)
				}
				record.Runtime.Software = pair[0]
				directory := filepath.Join(record.Storage.DataPath, "mods")
				if err = os.MkdirAll(directory, 0755); err != nil {
					t.Fatal(err)
				}
				pluginTestJar(t, filepath.Join(directory, "named-mod.jar"), map[string]string{
					"fabric.mod.json":             `{"schemaVersion":1,"id":"family_mod","name":"Family Mod","version":"1.0","environment":"server"}`,
					"quilt.mod.json":              `{"quilt_loader":{"id":"family_mod","version":"1.0","metadata":{"name":"Family Mod"}}}`,
					"META-INF/mods.toml":          "[[mods]]\nmodId=\"family_mod\"\nversion=\"1.0\"\n",
					"META-INF/neoforge.mods.toml": "[[mods]]\nmodId=\"family_mod\"\nversion=\"1.0\"\n",
				})
				root, err := openContentDirectory(record.Storage.DataPath, "mods")
				if err != nil {
					t.Fatal(err)
				}
				hash, err := hashPluginAt(root, "named-mod.jar")
				root.Close()
				if err != nil {
					t.Fatal(err)
				}
				previous := pluginCatalogClient
				t.Cleanup(func() { pluginCatalogClient = previous })
				pluginCatalogClient = &http.Client{Transport: pluginTestTransport(func(*http.Request) (*http.Response, error) {
					if status == "unknown" {
						return nil, errors.New("provider unavailable")
					}
					loader, version, publishedHash := pair[1], "1.21.4", hash
					if status == "incompatible" {
						loader = pair[0]
					}
					if status == "wrong version" {
						version = "1.21.5"
					}
					if status == "wrong hash" {
						publishedHash = "other bytes"
					}
					release := pluginCatalogVersion{ID: "ijklmnop", ProjectID: "abcdefgh", GameVersions: []string{version}, Loaders: []string{loader}, Environment: "server_only", Dependencies: []pluginDependency{}, Files: []pluginArtifactFile{{}}}
					release.Files[0].Hashes.SHA512 = publishedHash
					if status == "missing dependency" {
						release.Dependencies = []pluginDependency{{ProjectID: "missing1", Type: "required"}}
					}
					raw, err := json.Marshal(release)
					if err != nil {
						t.Fatal(err)
					}
					return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(raw)), Header: make(http.Header)}, nil
				})}
				before := switchWorldSnapshot(t, record.Storage.DataPath)
				content, _, err := switchContentEvidence(context.Background(), record, pair[1], "1.21.4")
				if err != nil || len(content) != 1 {
					t.Fatalf("inventory missing: %+v %v", content, err)
				}
				want := status
				if status == "missing dependency" || status == "wrong hash" {
					want = "unknown"
				}
				if status == "wrong version" {
					want = "incompatible"
				}
				if content[0].Compatibility != want || content[0].Filename != "named-mod.jar" {
					t.Fatalf("wrong target evidence: %+v want %s", content, want)
				}
				convert, clone := switchTransitionActions(pair[0], pair[1], true, true, content)
				if !clone || convert {
					t.Fatalf("wrong actions convert=%v clone=%v", convert, clone)
				}
				after := switchWorldSnapshot(t, record.Storage.DataPath)
				if len(before) != len(after) {
					t.Fatal("planner changed inventory")
				}
				for path, raw := range before {
					if after[path] != raw {
						t.Fatal("planner modified executable content", path)
					}
				}
			})
		}
	}
}

func TestSwitchCloneExcludesReviewedRelativeIdentities(t *testing.T) {
	source, clone := t.TempDir(), t.TempDir()
	plan := minecraftSwitchPlan{}
	for _, item := range []struct {
		directory, filename, compatibility, action string
	}{
		{"plugins", "Shared.jar", "verified", "retain"},
		{"mods", "Shared.jar", "incompatible", "exclude"},
		{"plugins", "Manual.jar", "unknown", "exclude"},
	} {
		for _, data := range []string{source, clone} {
			dir := filepath.Join(data, item.directory)
			if err := os.MkdirAll(dir, 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, item.filename), []byte(item.directory+item.filename), 0644); err != nil {
				t.Fatal(err)
			}
		}
		dir, err := openContentDirectory(source, item.directory)
		if err != nil {
			t.Fatal(err)
		}
		hash, err := hashSwitchExecutable(dir, item.filename)
		dir.Close()
		if err != nil {
			t.Fatal(err)
		}
		plan.Content = append(plan.Content, switchContent{Directory: item.directory, Filename: item.filename,
			Compatibility: item.compatibility, CloneDisposition: item.action, SHA512: hash.SHA512, SHA256: hash.SHA256})
	}
	manifest := pluginManifest{SchemaVersion: "v1", Artifacts: []pluginManifestEntry{{
		Target: "mods", Source: "Modrinth", ProjectID: "abcdefgh", VersionID: "ijklmnop",
		VersionNumber: "1.0", PluginName: "Shared", Filename: "Shared.jar", SHA512: plan.Content[1].SHA512,
		Requested: true, RequiredBy: []string{},
	}}}
	for _, data := range []string{source, clone} {
		if changed, err := commitPluginManifest(data, nil, manifest); err != nil || !changed {
			t.Fatal("fixture manifest", err)
		}
	}
	_, manifestBytes, err := readPluginManifest(source)
	if err != nil {
		t.Fatal(err)
	}
	plan.ManifestHash = instanceHash(manifestBytes)
	original := switchWorldSnapshot(t, source)
	root, err := openPluginDataDirectory(clone)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	const op = "12345678-1234-4123-8123-123456789abc"
	expected, manifestHash, err := applySwitchCloneContent(root, op, plan)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := verifySwitchCloneContent(clone, manifestHash, expected, nil); err != nil || !instanceSHA256Pattern.MatchString(got) {
		t.Fatal("verified new content", err)
	}
	if len(expected) != 1 || expected[switchExecutableIdentity{"plugins", "Shared.jar"}].SHA512 != plan.Content[0].SHA512 {
		t.Fatal("same filename in the other directory changed the result", expected)
	}
	if updated, _, err := readPluginManifest(clone); err != nil || len(updated.Artifacts) != 0 {
		t.Fatal("excluded mod ownership was left active", err, updated)
	}
	for _, item := range plan.Content[1:] {
		path := filepath.Join(clone, ".justvoxel", "plugin-transactions", op, item.Directory, item.Filename)
		if data, err := os.ReadFile(path); err != nil || string(data) != item.Directory+item.Filename {
			t.Fatal("excluded bytes were not retained", err)
		}
	}
	if after := switchWorldSnapshot(t, source); !reflect.DeepEqual(original, after) {
		t.Fatal("source files or manifest changed")
	}
}

func TestSwitchCloneRefusesChangedOrUnsafeRestoredContent(t *testing.T) {
	for _, change := range []string{"bytes", "symlink", "hardlink", "directory", "unreviewed"} {
		t.Run(change, func(t *testing.T) {
			clone := t.TempDir()
			dir := filepath.Join(clone, "plugins")
			if err := os.Mkdir(dir, 0755); err != nil {
				t.Fatal(err)
			}
			jar := filepath.Join(dir, "Manual.jar")
			if err := os.WriteFile(jar, []byte("original"), 0644); err != nil {
				t.Fatal(err)
			}
			found, err := readSwitchExecutableInventory(clone)
			if err != nil {
				t.Fatal(err)
			}
			hash := found[switchExecutableIdentity{"plugins", "Manual.jar"}]
			var changed error
			switch change {
			case "bytes":
				changed = os.WriteFile(jar, []byte("new bytes"), 0644)
			case "symlink":
				other := filepath.Join(t.TempDir(), "Outside.jar")
				if err := os.Rename(jar, other); err != nil {
					t.Fatal(err)
				}
				changed = os.Symlink(other, jar)
			case "hardlink":
				changed = os.Link(jar, filepath.Join(clone, "Hardlink.jar"))
			case "directory":
				if err := os.Rename(dir, filepath.Join(clone, "other")); err != nil {
					t.Fatal(err)
				}
				changed = os.Symlink(filepath.Join(clone, "other"), dir)
			case "unreviewed":
				changed = os.WriteFile(filepath.Join(dir, "New.jar"), []byte("new file"), 0644)
			}
			if changed != nil {
				t.Fatal(changed)
			}
			root, err := openPluginDataDirectory(clone)
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			plan := minecraftSwitchPlan{ManifestHash: instanceHash(nil), Content: []switchContent{{
				Directory: "plugins", Filename: "Manual.jar", Compatibility: "unknown", CloneDisposition: "exclude",
				SHA512: hash.SHA512, SHA256: hash.SHA256,
			}}}
			if _, _, err := applySwitchCloneContent(root, "12345678-1234-4123-8123-123456789abc", plan); err == nil {
				t.Fatalf("Clone accepted %s", change)
			}
			if _, err := os.Lstat(filepath.Join(clone, ".justvoxel")); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("refused Clone mutated the restored content", err)
			}
		})
	}
}

func TestSwitchRequiredDependencyExclusionPropagates(t *testing.T) {
	f := switchPlannerFixture(t, "fabric")
	record, err := f.backend.readRecord("jv-a1b2c3")
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(record.Storage.DataPath, "mods")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	releases := map[string]pluginCatalogVersion{}
	for index, name := range []string{"first", "second", "third"} {
		file := name + ".jar"
		pluginTestJar(t, filepath.Join(dir, file), map[string]string{"fabric.mod.json": fmt.Sprintf(
			`{"schemaVersion":1,"id":"%s","name":"%s","version":"1.0","environment":"server"}`, name, name)})
		root, err := openContentDirectory(record.Storage.DataPath, "mods")
		if err != nil {
			t.Fatal(err)
		}
		hash, err := hashPluginAt(root, file)
		root.Close()
		if err != nil {
			t.Fatal(err)
		}
		release := pluginCatalogVersion{ID: fmt.Sprintf("version%d", index), ProjectID: fmt.Sprintf("project%d", index),
			GameVersions: []string{"1.21.4"}, Loaders: []string{"quilt"}, Environment: "server_only",
			Dependencies: []pluginDependency{}, Files: []pluginArtifactFile{{}}}
		release.Files[0].Hashes.SHA512 = hash
		if index < 2 {
			release.Dependencies = append(release.Dependencies, pluginDependency{ProjectID: fmt.Sprintf("project%d", index+1), Type: "required"})
		} else {
			release.Loaders = []string{"fabric"}
		}
		releases[hash] = release
	}
	previous := pluginCatalogClient
	t.Cleanup(func() { pluginCatalogClient = previous })
	pluginCatalogClient = &http.Client{Transport: pluginTestTransport(func(r *http.Request) (*http.Response, error) {
		raw, err := json.Marshal(releases[filepath.Base(r.URL.Path)])
		if err != nil {
			return nil, err
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(raw)), Header: make(http.Header)}, nil
	})}
	plan, err := planMinecraftSwitch(context.Background(), record.Runtime.ID, "quilt")
	if err != nil || !plan.Clone || plan.Convert || !plan.RequiresExclusionConsent || len(plan.Content) != 3 {
		t.Fatalf("unexpected dependency and world review: %+v %v", plan, err)
	}
	for i, item := range plan.Content {
		if item.CloneDisposition != "exclude" || i < 2 && !strings.Contains(item.Reason, "Required dependency") {
			t.Fatalf("dependency chain did not exclude %s: %+v", item.Name, item)
		}
	}
}
