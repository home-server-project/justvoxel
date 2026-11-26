package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strings"
	"testing"
)

func switchMatrixSelection(software, version string, bedrock bool) adminVersionStatus {
	c, _ := javaServerSoftware(software)
	p := adminVersionStatus{ServerType: software, Installed: version, SelectedCandidate: version, ServerSupported: true,
		StackState: "updates_available", UpdateAvailable: true, PlanFingerprint: testMinecraftResetFingerprint,
		CrossplayEnabled: bedrock && c.ManagedCrossplay, CrossplayCompatible: true, ImageTag: "stable"}
	if c.ModLoader {
		p.LoaderVersion, p.ImageTag = "1.2.3", "java21"
		if java25GameReleasePattern.MatchString(version) {
			p.ImageTag = "java25"
		}
	}
	if software == "pufferfish" {
		p.PufferfishBuild = 39
	}
	return p
}

func TestSwitchProductionTargetConfigurationMatrix(t *testing.T) {
	var software []string
	for name := range javaServerCapabilityTable {
		if switchStructuralPath(name, "paper") || switchStructuralPath("paper", name) || name == "paper" {
			software = append(software, name)
		}
	}
	sort.Strings(software)
	for _, from := range software {
		for _, version := range []string{"1.21.4", "26.3"} {
			for _, bedrock := range []bool{false, true} {
				if bedrock && !javaServerCrossplay(from) {
					continue
				}
				t.Run(fmt.Sprintf("%s/%s/bedrock-%t", from, version, bedrock), func(t *testing.T) {
					f := newInstanceActivationFixtureFor(t, from, version)
					oldBackend, oldStatus, oldPreview := instanceRuntimes, runAdminVersionStatus, runSetupVersionPreview
					instanceRuntimes = f.backend
					t.Cleanup(func() {
						instanceRuntimes, runAdminVersionStatus, runSetupVersionPreview = oldBackend, oldStatus, oldPreview
					})
					ctx := context.Background()
					source, err := f.backend.readRecord("jv-a1b2c3")
					if err != nil {
						t.Fatal(err)
					}
					selection := switchMatrixSelection(from, version, bedrock)
					source.Runtime.VersionPolicy, source.Runtime.LoaderVersion, source.Runtime.ImageTag = "pinned", selection.LoaderVersion, selection.ImageTag
					source.Runtime.BedrockEnabled = bedrock
					if bedrock {
						for source.Runtime.BedrockPort == 0 || source.Runtime.BedrockPort == f.configs[0].JavaPort || source.Runtime.BedrockPort == f.configs[1].JavaPort {
							source.Runtime.BedrockPort = instanceTestPort(t)
						}
					}
					source.Runtime.PreserveContent = !bedrock && from == "purpur" // A prior Switch result.
					if err = f.backend.writeJSON(filepath.Join(f.backend.root, source.Runtime.ID, "config.json"), source, true); err != nil {
						t.Fatal(err)
					}
					command := f.backend.command
					f.backend.command = func(ctx context.Context, program string, args ...string) ([]byte, error) {
						if program == instanceRuntimeHelper && args[0] == "crossplay" || strings.HasSuffix(program, "/managed-plugin-files") {
							return nil, nil // External publisher/component helper boundary.
						}
						return command(ctx, program, args...)
					}
					if err = f.apply(source.Runtime.ID); err != nil {
						t.Fatal(err)
					}
					source, err = f.backend.readRecord(source.Runtime.ID)
					if err != nil {
						t.Fatal(err)
					}
					if javaServerPaperFamily(from) {
						switchWorldFixture(t, source.Storage.DataPath, version, "Custom World")
					}
					artifacts := map[string]switchManagedArtifact{}
					if bedrock && javaServerPaperFamily(from) {
						plugins := filepath.Join(source.Storage.DataPath, "plugins")
						if err := os.Mkdir(plugins, 0755); err != nil {
							t.Fatal(err)
						}
						for _, jar := range []struct{ project, name, main string }{
							{"geyser", "Geyser-Spigot", "org.geysermc.geyser.platform.spigot.GeyserSpigotPlugin"},
							{"floodgate", "floodgate-spigot", "org.geysermc.floodgate.SpigotPlugin"},
						} {
							path := filepath.Join(plugins, jar.name+".jar")
							pluginTestJar(t, path, map[string]string{"plugin.yml": "name: " + jar.name + "\nversion: 1\nmain: " + jar.main + "\n"})
							raw, err := os.ReadFile(path)
							if err != nil {
								t.Fatal(err)
							}
							hash := sha256.Sum256(raw)
							artifacts[jar.project] = switchManagedArtifact{SHA256: hex.EncodeToString(hash[:])}
						}
					}
					oldClient := pluginCatalogClient
					pluginCatalogClient = &http.Client{Transport: pluginTestTransport(func(*http.Request) (*http.Response, error) {
						return nil, errors.New("no external catalog in fixture")
					})}
					t.Cleanup(func() { pluginCatalogClient = oldClient })
					resolver := func(target string) ([]byte, error) {
						p := switchMatrixSelection(target, version, bedrock)
						return json.Marshal(struct {
							adminVersionStatus
							Digest    string                           `json:"managed_image_digest"`
							Artifacts map[string]switchManagedArtifact `json:"managed_artifacts"`
						}{p, testMinecraftResetFingerprint, artifacts})
					}
					runAdminVersionStatus = func(_ context.Context, policy, selected, target, image string) ([]byte, error) {
						if target == "" {
							return json.Marshal(adminVersionStatus{Installed: version})
						}
						if policy != "pinned" || selected != version {
							t.Fatal("planner changed the exact game version")
						}
						return resolver(target)
					}
					runSetupVersionPreview = func(_ context.Context, policy, selected, enabled, target string) ([]byte, error) {
						return resolver(target)
					}
					for _, running := range []bool{false, true} {
						state := "inactive"
						if running {
							state = "active"
						}
						f.states[source.Runtime.ID] = state
						for _, to := range software {
							if to == from {
								continue
							}
							t.Run(fmt.Sprintf("%s/running-%t", to, running), func(t *testing.T) {
								plan, err := planMinecraftSwitch(ctx, source.Runtime.ID, to)
								if err != nil {
									t.Fatal(err)
								}
								blockedPlatform := bedrock && javaServerCrossplay(to) && switchCrossplayPlatform(from) != switchCrossplayPlatform(to)
								if blockedPlatform {
									if plan.Clone || plan.Convert || !strings.Contains(plan.Reason, "different managed cross-play artifacts") {
										t.Fatalf("invalid cross-play platform review: %+v", plan)
									}
									return
								}
								if !plan.Clone || plan.Version != version || plan.Fingerprint == "" {
									t.Fatalf("missing exact fixture plan: %+v", plan)
								}
								for _, clone := range []bool{true, false} {
									// Render both target contracts even when content or world policy forbids Convert.
									target, snapshot, err := f.backend.prepareSwitchTarget(ctx, source, plan, clone)
									if err != nil {
										t.Fatal(err)
									}
									c := target.Runtime
									want := switchMatrixSelection(to, version, bedrock)
									if c.VersionPolicy != "pinned" || !c.PreserveContent || c.Software != to || c.Version != version ||
										c.BedrockEnabled != want.CrossplayEnabled || c.LoaderVersion != want.LoaderVersion ||
										c.PufferfishBuild != want.PufferfishBuild || c.ImageTag != want.ImageTag || c.ImageDigest != testMinecraftResetFingerprint {
										t.Fatal("prospective target lost reviewed configuration")
									}
									if !c.BedrockEnabled && c.BedrockPort != 0 || c.BedrockUDPStart != 0 || c.BedrockUDPEnd != 0 || c.BedrockTCPPort != 0 {
										t.Fatal("prospective target inherited an invalid port")
									}
									candidate := ""
									if clone {
										candidate = c.ID
										if c.ID == source.Runtime.ID || c.RCONPassword == source.Runtime.RCONPassword || c.StartOnBoot ||
											c.JavaPort == source.Runtime.JavaPort || c.BedrockEnabled && c.BedrockPort == source.Runtime.BedrockPort || len(target.Firewall) != 0 {
											t.Fatal("clone inherited identity, credentials or reservations")
										}
									} else if c.ID != source.Runtime.ID || c.RCONPassword != source.Runtime.RCONPassword || c.JavaPort != source.Runtime.JavaPort ||
										c.StartOnBoot != source.Runtime.StartOnBoot || c.BedrockEnabled && c.BedrockPort != source.Runtime.BedrockPort ||
										!slices.Equal(source.Firewall, target.Firewall) {
										t.Fatal("convert changed the source ownership contract before publication")
									}
									records, err := f.backend.records()
									if err != nil {
										t.Fatal(err)
									}
									files, err := f.backend.renderCandidate(snapshot, instanceReplaceRecord(records, target), candidate)
									if err != nil {
										t.Fatal(err)
									}
									for _, bundle := range files {
										if bundle.ID != c.ID {
											continue
										}
										q, env := string(bundle.Quadlet), string(bundle.Environment)
										if !strings.Contains(q, "@"+c.ImageDigest) || strings.Contains(q, ":19132/udp") != c.BedrockEnabled ||
											!strings.Contains(env, "VERSION="+version+"\n") || strings.Contains(env, "MODRINTH_PROJECTS=") {
											t.Fatal("rendered target did not match reviewed image, ports, game or content policy")
										}
									}
								}
								after, err := f.backend.readRecord(source.Runtime.ID)
								if err != nil || !reflect.DeepEqual(source, after) || f.states[source.Runtime.ID] != state {
									t.Fatal("target preparation changed the committed source or service state", err)
								}
							})
						}
					}
				})
			}
		}
	}
}

func TestSwitchJavaOnlyPublicationReleasesOwnedCrossplayPorts(t *testing.T) {
	ctx := context.Background()
	f := newInstanceActivationFixtureFor(t, "paper", "1.21.4")
	oldBackend := instanceRuntimes
	instanceRuntimes = f.backend
	t.Cleanup(func() { instanceRuntimes = oldBackend })
	id := f.configs[0].ID
	secondID := f.configs[1].ID
	source, err := f.backend.readRecord(id)
	if err != nil {
		t.Fatal(err)
	}
	second, err := f.backend.readRecord(secondID)
	if err != nil {
		t.Fatal(err)
	}
	if source.Runtime != f.configs[0] || second.Runtime != f.configs[1] || secondID == id {
		t.Fatal("unexpected original fixture reservations")
	}
	port := 0
	for port == 0 || port == f.configs[0].JavaPort || port == f.configs[1].JavaPort {
		port = instanceTestPort(t)
	}
	source.Runtime.BedrockEnabled, source.Runtime.BedrockPort = true, port
	source.Runtime.VersionPolicy = "pinned"
	// A pre-existing UDP opening must survive even as owned TCP rules retire.
	unrelated := instanceFirewallRule{Zone: "public", Port: port, Protocol: "udp", Permanent: true}
	f.firewall[unrelated] = true
	if err = f.backend.writeJSON(filepath.Join(f.backend.root, id, "config.json"), source, true); err != nil {
		t.Fatal(err)
	}
	if err = f.apply(id); err != nil {
		t.Fatal(err)
	}
	source, err = f.backend.readRecord(id)
	if err != nil {
		t.Fatal(err)
	}
	ownedTCP, ownedUDP := false, false
	for _, rule := range source.Firewall {
		if rule.Port == port {
			ownedTCP = ownedTCP || rule.Protocol == "tcp"
			ownedUDP = ownedUDP || rule.Protocol == "udp"
		}
	}
	if !ownedTCP || !ownedUDP || instanceRuleContains(source.Firewall, unrelated) {
		t.Fatal("wrong initial cross-play firewall ownership")
	}
	p := minecraftSwitchPlan{Current: "paper", Target: "vanilla", InstanceID: id, Version: source.Runtime.Version, Convert: true,
		Fingerprint: testMinecraftResetFingerprint, ContentFingerprint: strings.TrimPrefix(testMinecraftResetFingerprint, "sha256:"),
		RuntimeDigest: testMinecraftResetFingerprint, Runtime: switchMatrixSelection("vanilla", source.Runtime.Version, false)}
	next, snapshot, err := f.backend.prepareSwitchTarget(ctx, source, p, false)
	if err != nil {
		t.Fatal(err)
	}
	records, err := f.backend.records()
	if err != nil {
		t.Fatal(err)
	}
	secondJava := second
	secondJava.Runtime.JavaPort = port
	// Native Bedrock can test the UDP reservation without also claiming the
	// cross-play TCP number. The TCP check uses the second Java server.
	secondUDP := second
	secondUDP.Runtime = minecraftInstanceRuntimeConfig{
		ID: secondID, Software: "bedrock", Version: "1.21.50",
		VersionPolicy: "pinned", ImageTag: "stable", BedrockTCPPort: second.Runtime.JavaPort,
		BedrockUDPStart: port, BedrockUDPEnd: port, ContainerMemory: second.Runtime.ContainerMemory,
		UID: second.Runtime.UID, GID: second.Runtime.GID, Timezone: second.Runtime.Timezone,
		EULAAccepted: true, MOTD: "Reservation fixture", GameMode: "survival", Difficulty: "normal",
		MaxPlayers: 1, PaperChannel: second.Runtime.PaperChannel, RCONPassword: second.Runtime.RCONPassword,
	}
	for _, candidate := range []instanceRuntimeRecord{secondJava, secondUDP} {
		if _, err = f.backend.render(snapshot, instanceReplaceRecord(records, candidate)); err == nil || !strings.Contains(err.Error(), "conflicting") {
			t.Fatal("renderer did not reject the occupied port", candidate.Runtime.Software, err)
		}
	}
	e := minecraftSwitchEvidence{Schema: 1, OperationID: strings.Repeat("d", 32), Source: id, Target: p.Target, Version: p.Version,
		Action: "convert", Stage: "converting_world", SourceRecord: &source, WorldChanged: true, ColdArchive: "Source-backup-paper-2026-11-25-1200.tar.gz"}
	if err = saveSwitchEvidence(e); err != nil {
		t.Fatal(err)
	}
	if err = f.backend.applySwitchRuntime(ctx, source, instanceReplaceRecord(records, next), snapshot, e, &p); err != nil {
		t.Fatal(err)
	}
	final, err := f.backend.readRecord(id)
	if err != nil || final.Runtime.BedrockEnabled || final.Runtime.BedrockPort != 0 || final.Runtime.JavaPort != source.Runtime.JavaPort {
		t.Fatal("wrong published Java-only reservation", err)
	}
	if !f.firewall[unrelated] {
		t.Fatal("activation removed firewall access it did not own")
	}
	for _, rule := range source.Firewall {
		if rule.Port == port {
			present, queryErr := f.backend.queryFirewall(ctx, rule)
			if queryErr != nil || present {
				t.Fatal("activation retained a previously owned cross-play firewall rule", rule, queryErr)
			}
		}
	}
	records, err = f.backend.records()
	if err != nil {
		t.Fatal(err)
	}
	untouched, err := f.backend.readRecord(secondID)
	if err != nil || untouched.Runtime != second.Runtime {
		t.Fatal("conversion did not preserve the second instance configuration", err)
	}
	for _, candidate := range []instanceRuntimeRecord{secondJava, secondUDP} {
		if _, err = f.backend.render(snapshot, instanceReplaceRecord(records, candidate)); err != nil {
			t.Fatal("old cross-play reservation was not released", candidate.Runtime.Software, err)
		}
	}
	if err = f.backend.apply(ctx, second, instanceReplaceRecord(records, secondJava), snapshot); err != nil {
		t.Fatal("second instance could not commit the released TCP reservation", err)
	}
	reassigned, err := f.backend.readRecord(secondID)
	if err != nil || reassigned.Runtime.JavaPort != port || f.backend.verifyLive(reassigned) != nil {
		t.Fatal("second instance did not commit the released port", err)
	}
	if !f.firewall[unrelated] {
		t.Fatal("TCP reassignment removed the unrelated UDP firewall opening")
	}
}

func TestSwitchNeoForgeManagedArtifactsStayBoundToSourceLoader(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("managed mod ownership fixtures require root-private files")
	}
	ctx := context.Background()
	f := switchPlannerFixture(t, "neoforge")
	source, err := f.backend.readRecord("jv-a1b2c3")
	if err != nil {
		t.Fatal(err)
	}
	port := 0
	for port == 0 || port == f.configs[0].JavaPort || port == f.configs[1].JavaPort {
		port = instanceTestPort(t)
	}
	withCrossplay := source
	withCrossplay.Runtime.BedrockEnabled, withCrossplay.Runtime.BedrockPort = true, port
	base := f.backend.command
	f.backend.command = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		if name == instanceRuntimeHelper && args[0] == "crossplay" {
			return nil, nil
		}
		return base(ctx, name, args...)
	}
	records, err := f.backend.records()
	if err != nil {
		t.Fatal(err)
	}
	if err = f.backend.apply(ctx, source, instanceReplaceRecord(records, withCrossplay), f.snapshot); err != nil {
		t.Fatal(err)
	}
	source, err = f.backend.readRecord(source.Runtime.ID)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := planMinecraftSwitch(ctx, source.Runtime.ID, "quilt")
	if err != nil || !plan.Clone || plan.Convert {
		t.Fatal("Java-only target with no incompatible jars must allow protected Clone", err, plan.Reason)
	}
	clone, _, err := f.backend.prepareSwitchTarget(ctx, source, plan, true)
	if err != nil || clone.Runtime.BedrockEnabled || clone.Runtime.BedrockPort != 0 || clone.Runtime.LoaderVersion != plan.Runtime.LoaderVersion {
		t.Fatal("Java-only Quilt candidate inherited NeoForge runtime state", err)
	}

	mods := filepath.Join(source.Storage.DataPath, "mods")
	if err = os.Mkdir(mods, 0755); err != nil {
		t.Fatal(err)
	}
	artifacts := map[string]string{}
	for _, artifact := range []struct{ file, mod string }{
		{"Geyser-NeoForge.jar", "geyser_neoforge"},
		{"floodgate-neoforge.jar", "floodgate"},
	} {
		raw := componentModJar(t, "neoforge", artifact.mod)
		if err = os.WriteFile(filepath.Join(mods, artifact.file), raw, 0644); err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(raw)
		artifacts[artifact.file] = hex.EncodeToString(sum[:])
	}
	ownership := filepath.Join(f.backend.root, source.Runtime.ID, "crossplay-mods.json")
	raw, err := json.Marshal(map[string]any{"schema": 1, "platform": "neoforge", "artifacts": artifacts})
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(ownership, raw, 0600); err != nil {
		t.Fatal(err)
	}
	blocked, err := planMinecraftSwitch(ctx, source.Runtime.ID, "quilt")
	if err != nil || blocked.Clone || blocked.Convert || len(blocked.Content) != len(artifacts) ||
		!strings.Contains(blocked.Reason, "Java-only target loader") {
		t.Fatalf("NeoForge-owned artifacts offered for launch on Quilt: %+v %v", blocked, err)
	}
	for _, artifact := range blocked.Content {
		if artifact.Compatibility != "incompatible" || !strings.Contains(artifact.Reason, "source loader") {
			t.Fatalf("missing source ownership in review: %+v", artifact)
		}
	}
	if _, err = os.Lstat(filepath.Join(f.backend.root, clone.Runtime.ID, "crossplay-mods.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("planning published target mod ownership", err)
	}
	current, err := os.ReadFile(ownership)
	if err != nil || string(current) != string(raw) {
		t.Fatal("planning changed private source ownership", err)
	}
}
