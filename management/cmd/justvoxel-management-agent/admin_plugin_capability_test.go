package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestExactCatalogPlatformCapabilitiesAcrossReleases(t *testing.T) {
	// The same platform contract must apply to old, current and future releases.
	accepted := map[string][]string{
		"spigot": {"spigot", "bukkit"}, "paper": {"paper", "spigot", "bukkit"},
		"purpur": {"purpur", "paper", "spigot", "bukkit"}, "pufferfish": {"paper", "spigot", "bukkit"}, "leaf": {"paper", "spigot", "bukkit"},
		"fabric": {"fabric"}, "forge": {"forge"}, "neoforge": {"neoforge"}, "quilt": {"quilt"},
	}
	for _, game := range []string{"1.21.8", "26.3", "99.1-test"} {
		for _, software := range []string{"spigot", "paper", "purpur", "pufferfish", "leaf", "fabric", "forge", "neoforge", "quilt"} {
			server := pluginContentContext(software)
			server.MinecraftVersion = game
			for _, platform := range []string{"bukkit", "spigot", "paper", "purpur", "fabric", "forge", "neoforge", "quilt"} {
				want := slices.Contains(accepted[software], platform)
				for _, exactGame := range []string{game, "wrong-game"} {
					expected := want && exactGame == game
					// Project identities and titles are deliberately irrelevant to this contract.
					for index, project := range []string{"exampleA", "unrelatedB"} {
						v := pluginCatalogVersion{ProjectID: project, GameVersions: []string{exactGame}, Loaders: []string{platform}}
						if pluginVersionCompatible(v, server) != expected {
							t.Fatal("Modrinth", game, software, platform, exactGame)
						}
						f := curseForgeFile{ID: int64(100 + index), ModID: int64(200 + index), GameVersions: []string{exactGame, strings.ToUpper(platform)}}
						if curseForgeArtifactEligible(f, server) != expected {
							t.Fatal("CurseForge artifact eligibility", game, software, platform, exactGame)
						}
						if curseForgeCompatible(f, server) != expected {
							t.Fatal("CurseForge", game, software, platform, exactGame)
						}
					}
				}
			}
			if software == "spigot" && !pluginVersionCompatible(pluginCatalogVersion{GameVersions: []string{game}, Loaders: []string{"bukkit", "paper"}}, server) {
				t.Fatal("explicit Bukkit evidence ignored")
			}
			for _, generic := range []string{"", "Bukkit Plugins", "Server", "Paper-compatible", "some words"} {
				if curseForgeCompatible(curseForgeFile{GameVersions: []string{game, generic}}, server) {
					t.Fatal("generic metadata became positive evidence", software, generic)
				}
			}
		}
	}
}

func TestCurseForgeUnknownPlatformDiscoveryAndReview(t *testing.T) {
	for _, game := range []string{"1.21.8", "99.1-test"} {
		t.Run(game, func(t *testing.T) {
			f, server := curseForgeCatalogFixture(t, "spigot")
			server.MinecraftVersion = game
			original := curseForgeClient.Transport
			curseForgeClient.Transport = pluginTestTransport(func(r *http.Request) (*http.Response, error) {
				response, err := original.RoundTrip(r)
				if err != nil {
					return response, err
				}
				if strings.Contains(r.URL.Path, "/files") {
					var payload map[string]json.RawMessage
					if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
						t.Fatal(err)
					}
					response.Body.Close()
					if strings.HasSuffix(r.URL.Path, "/files") {
						var files []curseForgeFile
						if err := json.Unmarshal(payload["data"], &files); err != nil {
							t.Fatal(err)
						}
						for i := range files {
							files[i].GameVersions = []string{game}
						}
						payload["data"], _ = json.Marshal(files)
					} else {
						var file curseForgeFile
						if err := json.Unmarshal(payload["data"], &file); err != nil {
							t.Fatal(err)
						}
						file.GameVersions = []string{game}
						payload["data"], _ = json.Marshal(file)
					}
					raw, _ := json.Marshal(payload)
					response.Body = io.NopCloser(strings.NewReader(string(raw)))
				}
				return response, nil
			})
			for _, compatible := range []bool{false, true} {
				page, err := discoverCurseForgePlugins(context.Background(), server, pluginDiscoverQuery{Limit: 8, Compatible: compatible, Release: true})
				if err != nil {
					t.Fatal(err)
				}
				if compatible {
					if len(page.Projects) != 0 {
						t.Fatal("unproven file in compatible-only discovery", page)
					}
				} else {
					if len(page.Projects) != 1 || page.Projects[0].Compatible || !strings.Contains(page.Projects[0].Compatibility, "unknown") || !page.Projects[0].InstallAvailable {
						t.Fatal(page)
					}
				}
			}
			plan, err := resolveCurseForgeInstallPlan(context.Background(), server, f.data, "10", "20")
			if err != nil || !strings.Contains(plan.Compatibility, "unknown") {
				t.Fatal(plan, err)
			}
			if err := reviewRootPluginArtifact(context.Background(), plan); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestSpigotExactArtifactStructure(t *testing.T) {
	for _, descriptor := range []string{"plugin.yml", "paper-plugin.yml"} {
		path := filepath.Join(t.TempDir(), "example.jar")
		pluginTestJar(t, path, map[string]string{descriptor: "name: ExamplePlugin\nmain: example.Plugin\nversion: 1\n"})
		jar, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		_, err = inspectServerPlugin(jar, "example.jar", "spigot")
		jar.Close()
		if (err == nil) != (descriptor == "plugin.yml") {
			t.Fatal(descriptor, err)
		}
	}
}

func TestHangarExactPlatformAcrossReleases(t *testing.T) {
	for _, game := range []string{"1.21.8", "26.3", "99.1-test"} {
		for _, software := range []string{"paper", "purpur", "pufferfish", "leaf", "spigot"} {
			for _, exact := range []string{game, "wrong-game"} {
				t.Run(game+"/"+software+"/"+exact, func(t *testing.T) {
					f := newHangarInstallFixture(t, software)
					f.minecraftVersion = game
					v := f.releases["1"]
					v.PlatformDependencies["PAPER"] = []string{exact}
					f.releases["1"] = v
					server := pluginContentContext(software)
					server.InstanceID, server.MinecraftVersion = "jv-a1b2c3", game
					want := software != "spigot" && exact == game
					selected, matches, err := selectHangarVersion(context.Background(), 1, server, true, true)
					if (err == nil && selected != nil && matches) != want {
						t.Fatal("discovery", selected, matches, err)
					}
					_, err = resolveHangarPluginInstallPlan(context.Background(), server, f.data, "1", "10")
					if (err == nil) != want {
						t.Fatal("install", err)
					}
				})
			}
		}
	}
}

func TestModrinthExactReleaseRevalidatedAcrossReleases(t *testing.T) {
	for _, game := range []string{"1.21.8", "26.3", "99.1-test"} {
		for _, platform := range []string{"bukkit", "spigot", "paper", "fabric", "forge", "neoforge", "quilt"} {
			for _, exact := range []string{game, "wrong-game"} {
				t.Run(game+"/"+platform+"/"+exact, func(t *testing.T) {
					f := newPluginInstallFixture(t)
					server := pluginContentContext("spigot")
					server.InstanceID, server.MinecraftVersion = "jv-a1b2c3", game
					v := f.versions["release"]
					v.Loaders, v.GameVersions = []string{platform}, []string{exact}
					f.versions["release"] = v
					_, err := resolvePluginInstallPlan(context.Background(), server, f.data, "root", "release")
					want := (platform == "bukkit" || platform == "spigot") && exact == game
					if (err == nil) != want {
						t.Fatal(game, platform, exact, err)
					}
				})
			}
		}
	}
	for _, environment := range []string{"client_only", "singleplayer_only", "", "unknown"} {
		if modrinthServerEnvironment(environment) {
			t.Fatal("client/unknown eligibility accepted", environment)
		}
	}
}

func TestModrinthContentEnvironmentCatalogAndPlans(t *testing.T) {
	for _, game := range []string{"1.21.8", "26.3", "26.99"} {
		for _, software := range []string{"spigot", "fabric", "forge", "neoforge", "quilt"} {
			platforms := []string{software}
			if software == "spigot" {
				platforms = []string{"bukkit", "spigot"}
			}
			for _, platform := range platforms {
				for _, environment := range []string{"", "unknown", "server_only", "client_and_server", "client_only", "singleplayer_only", "future_environment"} {
					for _, evidence := range []string{"exact", "wrong loader", "wrong game"} {
						// Exercise missing/unknown plugin environments and recognized mod
						// environments against independently incompatible platform evidence.
						if evidence != "exact" && environment != "unknown" && environment != "server_only" {
							continue
						}
						t.Run(strings.Join([]string{game, software, platform, environment, evidence}, "/"), func(t *testing.T) {
							f := newPluginInstallFixtureFor(t, software, game)
							server := pluginContentContext(software)
							server.InstanceID, server.MinecraftVersion = "jv-a1b2c3", game
							if server.ContentKind == "mods" {
								if err := os.Mkdir(filepath.Join(f.data, "mods"), 0700); err != nil {
									t.Fatal(err)
								}
							}
							root := f.versions["release"]
							root.Loaders, root.GameVersions = []string{platform}, []string{game}
							v := root
							v.Environment = environment
							if evidence == "wrong loader" {
								v.Loaders = []string{"paper"} // Paper-only is never Spigot/mod evidence.
							}
							if evidence == "wrong game" {
								v.GameVersions = []string{"different-game"}
							}
							f.versions["release"] = v
							wantEnvironment := environment == "server_only" || environment == "client_and_server" || server.ContentKind == "plugins" && (environment == "" || environment == "unknown")
							want := wantEnvironment && evidence == "exact"
							original := pluginCatalogClient.Transport
							pluginCatalogClient.Transport = pluginTestTransport(func(r *http.Request) (*http.Response, error) {
								if r.URL.Path == "/v2/search" {
									return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"hits":[{"project_id":"root","title":"ExamplePlugin"}],"total_hits":1}`))}, nil
								}
								return original.RoundTrip(r)
							})
							page, err := discoverModrinthPlugins(context.Background(), server, "", true, true, 0, 8)
							if err != nil {
								t.Fatal(err)
							}
							if !want {
								if len(page.Projects) != 0 {
									t.Fatal("ineligible release in Discover", page)
								}
							} else if len(page.Projects) != 1 || !page.Projects[0].Compatible || page.Projects[0].InstallAvailable != (server.ContentKind == "plugins") || page.Projects[0].ReviewAvailable != (server.ContentKind == "mods") || strings.Contains(page.Projects[0].Compatibility, "does not declare") {
								t.Fatal("eligible release unavailable", page)
							}
							plan, err := resolvePluginInstallPlan(context.Background(), server, f.data, "root", "release")
							if (err == nil) != want || err == nil && len(plan.Artifacts) != 1 {
								t.Fatal("exact plan", plan, err)
							}

							// The same release policy applies to pinned and selected required
							// dependencies. An incompatible plugin loader may remain display-only,
							// but must never become an install artifact.
							f.addJar(t, "dependency", "depversion", "dependency.jar", "Dependency", "example.Dependency")
							dep := f.versions["depversion"]
							dep.Environment, dep.Loaders, dep.GameVersions = v.Environment, v.Loaders, v.GameVersions
							f.versions["depversion"] = dep
							for _, pinned := range []bool{false, true} {
								d := pluginDependency{ProjectID: "dependency", Type: "required"}
								if pinned {
									d.VersionID = "depversion"
								}
								root.Dependencies = []pluginDependency{d}
								f.versions["release"] = root
								plan, err := resolvePluginInstallPlan(context.Background(), server, f.data, "root", "release")
								if want {
									if err != nil || len(plan.Artifacts) != 2 {
										t.Fatal("eligible dependency rejected", pinned, plan, err)
									}
								} else if err == nil && (evidence != "wrong loader" || server.ContentKind != "plugins" || !wantEnvironment || len(plan.Artifacts) != 1) {
									t.Fatal("ineligible dependency installed", pinned, plan)
								}
							}
						})
					}
				}
			}
		}
	}
}

func TestModrinthContentEnvironmentPolicy(t *testing.T) {
	for _, software := range []string{"spigot", "paper", "purpur", "pufferfish", "leaf", "fabric", "forge", "neoforge", "quilt", "vanilla", "unsupported"} {
		for _, environment := range []string{"client_and_server", "client_only_server_optional", "server_only", "server_only_client_optional", "dedicated_server_only", "client_or_server", "client_or_server_prefers_both", "", "unknown", "client_only", "singleplayer_only", "future_environment"} {
			plugins := slices.Contains([]string{"spigot", "paper", "purpur", "pufferfish", "leaf"}, software)
			mods := slices.Contains([]string{"fabric", "forge", "neoforge", "quilt"}, software)
			knownServer := slices.Contains([]string{"client_and_server", "client_only_server_optional", "server_only", "server_only_client_optional", "dedicated_server_only", "client_or_server", "client_or_server_prefers_both"}, environment)
			want := (plugins || mods) && knownServer || plugins && (environment == "" || environment == "unknown")
			if modrinthContentEnvironment(environment, software) != want {
				t.Fatal(software, environment, want)
			}
		}
	}
}
