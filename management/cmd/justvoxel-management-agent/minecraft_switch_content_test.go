package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
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
				if !clone || convert != (status == "verified") {
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
