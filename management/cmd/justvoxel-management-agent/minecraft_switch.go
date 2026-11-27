package main

import (
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Structural eligibility is separate from exact runtime, world and content proof.
// Spigot stays unsupported until its own source layout has a verified path.
func switchStructuralPath(from, to string) bool {
	a, aok := javaServerSoftware(from)
	b, bok := javaServerSoftware(to)
	if !aok || !bok || from == to || from == "spigot" || to == "spigot" {
		return false
	}
	return (javaServerPaperFamily(from) || from == "vanilla" || a.ModLoader) &&
		(javaServerPaperFamily(to) || to == "vanilla" || b.ModLoader)
}

func switchNeedsWorldMove(from, to string) bool {
	c, _ := javaServerSoftware(to)
	return javaServerPaperFamily(from) && (to == "vanilla" || c.ModLoader)
}

// An empty mods folder does not prove that a modded world is Vanilla-safe.
func switchWorldRisk(from, to string) bool {
	c, _ := javaServerSoftware(from)
	return c.ModLoader && from != to
}

func switchRejectsCrossplayPlatform(from, to string, enabled bool) bool {
	return enabled && javaServerCrossplay(to) && switchCrossplayPlatform(from) != switchCrossplayPlatform(to)
}

// runtimeAvailable means the exact game release, target loader pin and image
// have all been verified. layoutVerified is required only for leaving Paper.
func switchTransitionActions(from, to string, runtimeAvailable, layoutVerified bool, content []switchContent) (convert, clone bool) {
	if !runtimeAvailable || !switchStructuralPath(from, to) || (switchNeedsWorldMove(from, to) || from == "leaf") && !layoutVerified {
		return false, false
	}
	convert, clone = !switchWorldRisk(from, to), true
	for _, item := range content {
		if item.Compatibility != "verified" {
			convert = false
		}
	}
	return
}

type switchContent struct {
	Name          string `json:"name"`
	Filename      string `json:"filename"`
	Compatibility string `json:"compatibility"`
	Reason        string `json:"reason"`
}
type switchManagedArtifact struct {
	SHA256 string `json:"sha256"`
}
type switchWorkerKey struct{}
type switchContentKey struct{}
type switchPufferfishBuildKey struct{}

type minecraftSwitchPlan struct {
	MatrixPolicyBlocked bool                             `json:"-"`
	MatrixLeafBuild     string                           `json:"-"`
	InstanceID          string                           `json:"instance_id"`
	Current             string                           `json:"current"`
	Target              string                           `json:"target"`
	Version             string                           `json:"version"`
	Convert             bool                             `json:"convert"`
	Clone               bool                             `json:"clone"`
	Reason              string                           `json:"reason"`
	Content             []switchContent                  `json:"content"`
	Warnings            []string                         `json:"warnings"`
	Fingerprint         string                           `json:"plan_fingerprint"`
	Runtime             adminVersionStatus               `json:"-"`
	ManagedArtifacts    map[string]switchManagedArtifact `json:"-"`
	RuntimeDigest       string                           `json:"-"`
	ContentFingerprint  string                           `json:"-"`
	WorldFingerprint    string                           `json:"-"`
}

func switchUnsupportedReason(from, to string) string {
	if from == "bedrock" || to == "bedrock" {
		return "Java and native Bedrock worlds and runtimes cannot be converted by JustVoxel."
	}
	if from == "spigot" || to == "spigot" {
		return "No verified Spigot world transformation is implemented. Paper-family to Spigot is unsafe; direct Spigot to Paper-family is unsupported for Minecraft 26.1 and newer."
	}
	return "No verified direct conversion is implemented for this software pair."
}

// Read the existing inventory and ownership manifest. No files are changed and
// no install resolver is invoked (those resolvers may prepare downloads).
func switchContentEvidence(ctx context.Context, record instanceRuntimeRecord, target, version string, runtimeMetadata ...map[string]switchManagedArtifact) ([]switchContent, string, error) {
	data := record.Storage.DataPath
	plugins, err := readPluginInventory(data)
	if err != nil {
		return nil, "", err
	}
	mods, err := readModInventory(data, record.Runtime.Software)
	if err != nil {
		return nil, "", err
	}
	manifest, raw, err := readPluginManifest(data)
	if err != nil {
		return nil, "", err
	}
	// Entries ignored by the display inventory (for example symlinked JARs)
	// are still unverified executable content and must block in-place Convert.
	for _, kind := range []string{"plugins", "mods"} {
		directory, openErr := openContentDirectory(data, kind)
		if errors.Is(openErr, os.ErrNotExist) {
			continue
		}
		if openErr != nil {
			return nil, "", openErr
		}
		entries, readErr := directory.ReadDir(-1)
		directory.Close()
		if readErr != nil || len(entries) > 10000 {
			return nil, "", errInstanceRecovery
		}
		for _, entry := range entries {
			if !strings.EqualFold(filepath.Ext(entry.Name()), ".jar") || entry.Type().IsRegular() {
				continue
			}
			item := installedPlugin{Name: entry.Name(), Filename: entry.Name(), Management: "manual", MetadataStatus: "Unverified filesystem entry"}
			if kind == "plugins" {
				plugins = append(plugins, item)
			} else {
				mods = append(mods, item)
			}
		}
	}
	sort.Slice(plugins, func(i, j int) bool { return plugins[i].Filename < plugins[j].Filename })
	sort.Slice(mods, func(i, j int) bool { return mods[i].Filename < mods[j].Filename })
	inventory := append(plugins, mods...)
	result := make([]switchContent, 0, len(inventory))
	hashes := []string{instanceHash(raw)}
	owned, err := runtimeOwnedModComponents(pluginServerContext{InstanceID: record.Runtime.ID, ServerType: record.Runtime.Software}, data)
	if err != nil {
		return nil, "", err
	}
	ownedMods := make(map[string]string, len(owned))
	for _, artifact := range owned {
		ownedMods[artifact.Filename] = artifact.SHA512
		hashes = append(hashes, "runtime-owned/"+artifact.Filename+":"+artifact.SHA512)
	}
	runtimeProjects := []string{}
	_, environment, _ := instanceRuntimes.paths(record.Runtime.ID)
	committed, readErr := instanceRuntimes.readPrivate(environment)
	if record.Runtime.PreserveContent && target != "" && javaServerManagedViaVersion(record.Runtime.Software) {
		// The temporary guard omits automatic project downloads. Recover their
		// authoritative normal definition from the same immutable renderer template,
		// rather than classifying a runtime artifact from its filename alone.
		if snapshot, _, _, snapshotErr := instanceRuntimes.snapshot(ctx); snapshotErr == nil {
			committed, readErr = os.ReadFile(filepath.Join(snapshot.TemplateRoot, "config", "minecraft.env.in"))
		}
	}
	if readErr == nil {
		for _, line := range strings.Split(string(committed), "\n") {
			if !strings.HasPrefix(line, "MODRINTH_PROJECTS=") {
				continue
			}
			for _, project := range strings.Split(strings.TrimPrefix(line, "MODRINTH_PROJECTS="), ",") {
				runtimeProjects = append(runtimeProjects, strings.TrimPrefix(project, "paper:"))
			}
		}
	}

	metadata := map[string]switchManagedArtifact{}
	if len(runtimeMetadata) > 0 {
		metadata = runtimeMetadata[0]
	}
	proofs := map[int]pluginCatalogVersion{}
	for index, item := range inventory {
		name := item.Name
		if name == "" {
			name = item.Filename
		}
		evidence := switchContent{Name: name, Filename: item.Filename, Compatibility: "unknown", Reason: "Unmanaged or unverified content. Verify compatibility in Plugins & Mods."}
		kind := "plugins"
		if index >= len(plugins) {
			kind = "mods"
		}
		var entry *pluginManifestEntry
		for i := range manifest.Artifacts {
			if manifest.Artifacts[i].Filename == item.Filename && pluginTarget(manifest.Artifacts[i].Target) == kind {
				entry = &manifest.Artifacts[i]
				break
			}
		}
		// Hash every local JAR, including unowned content, into the reviewed plan.
		root, openErr := openContentDirectory(data, kind)
		hash := ""
		if openErr == nil {
			hash, _ = hashPluginAt(root, item.Filename)
			root.Close()
		}
		hashes = append(hashes, kind+"/"+item.Filename+":"+hash)
		if target == "" {
			result = append(result, evidence)
			continue
		}
		if (item.Management == "justvoxel" || entry == nil && len(runtimeProjects) > 0) && kind == "plugins" && pluginHashPattern.MatchString(hash) {
			// Built-in classification alone is not provenance. Prove exact installed
			// bytes against the existing publisher catalogue before calling compatible.
			var v pluginCatalogVersion
			var project struct {
				ID   string `json:"id"`
				Slug string `json:"slug"`
			}
			verified := false
			if modrinthPluginJSON(ctx, "/version_file/"+hash, map[string][]string{"algorithm": {"sha512"}}, &v) == nil && modrinthPluginID.MatchString(v.ProjectID) && pluginVersionCompatible(v, pluginServerContext{ServerType: target, MinecraftVersion: version}) && modrinthContentEnvironment(v.Environment, target) && v.Dependencies != nil && slices.ContainsFunc(v.Files, func(file pluginArtifactFile) bool { return file.Hashes.SHA512 == hash }) {
				verified = modrinthPluginJSON(ctx, "/project/"+v.ProjectID, nil, &project) == nil && project.ID == v.ProjectID && slices.Contains(runtimeProjects, project.Slug)
			}
			if verified {
				evidence.Compatibility, evidence.Reason = "verified", "Installed managed runtime artifact verified by publisher hash and exact game/plugin API metadata."
				proofs[index] = v
			} else {
				component := managedPluginIdentity(item.Filename, item.Main)
				if artifact, ok := metadata[component]; ok && instanceSHA256Pattern.MatchString(artifact.SHA256) {
					directory, e := openPluginDirectory(data)
					if e == nil {
						jar, e := openPluginAt(directory, item.Filename)
						if e == nil {
							h := sha256.New()
							_, e = io.Copy(h, io.LimitReader(jar, pluginMaxJar+1))
							jar.Close()
							if e == nil && hex.EncodeToString(h.Sum(nil)) == artifact.SHA256 {
								evidence.Compatibility, evidence.Reason = "verified", "Installed official managed artifact matches the exact-version runtime plan."
							}
						}
						directory.Close()
					}
				}
			}
		} else if entry != nil && pluginTarget(entry.Target) == kind && pluginSource(entry.Source) == "Modrinth" && modrinthPluginID.MatchString(entry.VersionID) {
			root, e := openContentDirectory(data, kind)
			if e == nil {
				hash, hashErr := hashPluginAt(root, entry.Filename)
				root.Close()
				if hashErr == nil && hash == entry.SHA512 {
					var v pluginCatalogVersion
					if modrinthPluginJSON(ctx, "/version/"+entry.VersionID, nil, &v) == nil && v.ID == entry.VersionID && v.ProjectID == entry.ProjectID {
						server := pluginServerContext{ServerType: target, MinecraftVersion: version}
						compatible := pluginVersionCompatible(v, server) && modrinthContentEnvironment(v.Environment, target) && slices.ContainsFunc(v.Files, func(file pluginArtifactFile) bool { return file.Hashes.SHA512 == entry.SHA512 })
						if compatible && v.Dependencies == nil {
							evidence.Reason = "Required dependency metadata is unavailable; compatibility is unverified."
						} else if compatible {
							evidence.Compatibility, evidence.Reason = "verified", "Provider declares compatibility for this exact Minecraft version and target plugin API."
							proofs[index] = v
						} else {
							evidence.Compatibility, evidence.Reason = "incompatible", "The installed release or a required dependency does not support the target at this exact Minecraft version."
						}
					}
				}
			}
		}
		if kind == "mods" && entry == nil && pluginHashPattern.MatchString(hash) {
			var v pluginCatalogVersion
			if modrinthPluginJSON(ctx, "/version_file/"+hash, map[string][]string{"algorithm": {"sha512"}}, &v) == nil && modrinthPluginID.MatchString(v.ID) && modrinthPluginID.MatchString(v.ProjectID) && slices.ContainsFunc(v.Files, func(f pluginArtifactFile) bool { return f.Hashes.SHA512 == hash }) {
				if !pluginVersionCompatible(v, pluginServerContext{ServerType: target, MinecraftVersion: version}) || !modrinthContentEnvironment(v.Environment, target) {
					evidence.Compatibility, evidence.Reason = "incompatible", "The exact installed mod release does not support this target loader and Minecraft version."
				} else if v.Dependencies != nil {
					evidence.Compatibility, evidence.Reason = "verified", "Installed mod bytes and target loader/version verified by publisher hash."
					proofs[index] = v
				}
			}
		}
		if entry != nil && pluginTarget(entry.Target) == kind && pluginSource(entry.Source) == "CurseForge" && hash == entry.SHA512 && pluginHashPattern.MatchString(hash) {
			release, lookupErr := exactCurseForgeFile(ctx, entry.ProjectID, entry.VersionID)
			if lookupErr == nil {
				if !curseForgeCompatible(release, pluginServerContext{ServerType: target, MinecraftVersion: version}) {
					evidence.Compatibility, evidence.Reason = "incompatible", "The installed CurseForge release does not support this exact target loader and Minecraft version."
				} else if release.Dependencies != nil && release.Game == 432 && release.Available && len(release.Dependencies) <= 128 {
					directory, openErr := openContentDirectory(data, kind)
					if openErr == nil {
						jar, jarErr := openPluginAt(directory, item.Filename)
						if jarErr == nil {
							digest := sha1.New()
							_, hashErr := io.Copy(digest, io.LimitReader(jar, pluginMaxJar+1))
							jar.Close()
							matches := 0
							for _, published := range release.Hashes {
								if published.Algo == 1 && curseForgeSHA1.MatchString(published.Value) && published.Value == hex.EncodeToString(digest.Sum(nil)) {
									matches++
								}
							}
							if hashErr == nil && matches == 1 {
								evidence.Compatibility, evidence.Reason = "verified", "Installed bytes and exact target compatibility verified by CurseForge."
								proof := pluginCatalogVersion{ID: "curseforge:" + entry.VersionID, ProjectID: "curseforge:" + entry.ProjectID, Dependencies: []pluginDependency{}}
								for _, d := range release.Dependencies {
									if d.Relation == 3 {
										proof.Dependencies = append(proof.Dependencies, pluginDependency{ProjectID: "curseforge:" + strconv.FormatInt(d.ModID, 10), Type: "required"})
									}
								}
								proofs[index] = proof
							}
						}
						directory.Close()
					}
				}
			}
		}
		if kind == "mods" && ownedMods[item.Filename] == hash && hash != "" && record.Runtime.Software != target {
			evidence.Compatibility, evidence.Reason = "incompatible", "The installed managed cross-play component belongs to the source loader."
			delete(proofs, index)
		}
		if javaServerContentKind(target) != kind {
			evidence.Compatibility, evidence.Reason = "incompatible", "This executable content is not supported by the target software. Handle it manually in Plugins & Mods."
			delete(proofs, index)
		}
		result = append(result, evidence)
	}
	// Required dependencies must themselves have verified installed bytes and
	// compatible provider evidence. Propagate blockers through dependency chains.
	for pass := 0; pass < len(proofs); pass++ {
		changed := false
		for index, release := range proofs {
			if result[index].Compatibility != "verified" {
				continue
			}
			for _, dependency := range release.Dependencies {
				if dependency.Type != "required" {
					continue
				}
				found := false
				if dependency.ProjectID != "" || dependency.VersionID != "" {
					for candidate, installed := range proofs {
						if result[candidate].Compatibility == "verified" && (dependency.ProjectID == "" || dependency.ProjectID == installed.ProjectID) && (dependency.VersionID == "" || dependency.VersionID == installed.ID) {
							found = true
							break
						}
					}
				}
				if !found {
					result[index].Compatibility = "unknown"
					result[index].Reason = "A required installed dependency cannot be verified for this exact Minecraft version and target."
					changed = true
					break
				}
			}
		}
		if !changed {
			break
		}
	}
	// Inventory and hashes both participate, including manual file additions.
	payload, err := json.Marshal(struct {
		Items  []installedPlugin
		Hashes []string
	}{inventory, hashes})
	if err != nil {
		return nil, "", err
	}
	return result, instanceHash(payload), nil
}

func planMinecraftSwitch(ctx context.Context, id, target string) (minecraftSwitchPlan, error) {
	p := minecraftSwitchPlan{InstanceID: id, Target: target, Content: []switchContent{}, Warnings: []string{}}
	if !validSetupServerType(target) {
		return p, errInstanceID
	}
	b := instanceRuntimes
	record, err := b.readRecord(id)
	if err != nil {
		return p, err
	}
	p.Current = record.Runtime.Software
	if p.Current == target {
		p.Reason = "Current server software."
		return p, nil
	}
	if err = b.verifyLive(record); err != nil {
		return p, err
	}
	if record.NetworkMember {
		p.Reason = "Remove this server from Minecraft Network before using standalone software conversion."
		return p, nil
	}
	if !switchStructuralPath(p.Current, target) {
		p.Reason = switchUnsupportedReason(p.Current, target)
		return p, nil
	}
	ctx = context.WithValue(ctx, instanceVersionKey{}, id)
	// Installed runtime evidence, rather than a moving Recommended/Latest choice,
	// determines the fixed version. The target plan must return that exact pin.
	raw, err := runAdminVersionStatus(ctx, "", "", "", "")
	var current adminVersionStatus
	if err != nil || json.Unmarshal(raw, &current) != nil || current.Installed == "" || current.Installed == "LATEST" || !instanceRuntimeVersionPattern.MatchString(current.Installed) {
		p.Reason = "The installed Minecraft version could not be verified. No conversion is available."
		return p, nil
	}
	p.Version = current.Installed
	if record.Runtime.Version != "LATEST" && record.Runtime.Version != p.Version {
		p.Reason = "Installed Minecraft evidence disagrees with the configured exact version. Switch cannot safely choose a game version."
		return p, nil
	}
	legacyRuntime := javaServerPaperFamily(p.Current) && javaServerPaperFamily(target)
	legacyBuild := 0
	imageTag := record.Runtime.ImageTag
	if legacyRuntime && (target == "pufferfish" || target == "leaf") {
		bedrock := "no"
		if record.Runtime.BedrockEnabled {
			bedrock = "yes"
		}
		resolved, resolveErr := runSetupVersionPreview(ctx, "pinned", p.Version, bedrock, target)
		var selection adminVersionStatus
		if resolveErr != nil || json.Unmarshal(resolved, &selection) != nil || selection.SelectedCandidate != p.Version || selection.ImageTag == "" || !selection.ServerSupported || !validPufferfishBuild(target, selection.PufferfishBuild) {
			p.Reason = "The target runtime does not support this exact Minecraft version, or its availability could not be verified."
			return p, nil
		}
		imageTag = selection.ImageTag
		legacyBuild = selection.PufferfishBuild
	}
	if switchRejectsCrossplayPlatform(p.Current, target, record.Runtime.BedrockEnabled) {
		p.MatrixPolicyBlocked = true
		p.Reason = "The target requires different managed cross-play artifacts. Switch preserves installed executable files and cannot install a different platform stack."
		return p, nil
	}
	if !legacyRuntime {
		ctx = context.WithValue(ctx, switchRuntimeKey{}, true)
	}
	raw, err = runAdminVersionStatus(ctx, "pinned", p.Version, target, imageTag)
	if err != nil || json.Unmarshal(raw, &p.Runtime) != nil || p.Runtime.ServerType != target || p.Runtime.SelectedCandidate != p.Version || p.Runtime.Installed != p.Version || !p.Runtime.ServerSupported || !operationFingerprintPattern.MatchString(p.Runtime.PlanFingerprint) || p.Runtime.StackState != "updates_available" || !p.Runtime.UpdateAvailable || (p.Runtime.CrossplayEnabled && !p.Runtime.CrossplayCompatible) {
		p.Reason = "The target runtime and managed cross-play could not be verified for Minecraft " + p.Version + ". Switch never upgrades or downgrades Minecraft."
		return p, nil
	}
	if legacyRuntime {
		p.Runtime.PufferfishBuild = legacyBuild
	}
	var managed struct {
		Artifacts map[string]switchManagedArtifact `json:"managed_artifacts"`
		Digest    string                           `json:"managed_image_digest"`
	}
	if json.Unmarshal(raw, &managed) != nil {
		return p, errInstanceRecovery
	}
	if !validPufferfishBuild(target, p.Runtime.PufferfishBuild) ||
		validateJavaServerSelection(target, "pinned", p.Version, p.Runtime.LoaderVersion, p.Runtime.ImageTag, p.Runtime.CrossplayEnabled) != nil {
		p.Reason = "An exact target runtime or loader pin could not be verified."
		return p, nil
	}
	if p.Runtime.CrossplayEnabled != (record.Runtime.BedrockEnabled && javaServerCrossplay(target)) {
		p.Reason = "The target did not return a verified cross-play setting for this exact release."
		return p, nil
	}
	if record.Runtime.BedrockEnabled && !p.Runtime.CrossplayEnabled {
		p.Warnings = append(p.Warnings, "The target uses Java connections. Existing cross-play executable files are preserved and can be reviewed in Plugins & Mods.")
	}
	if target == "pufferfish" && verifySwitchPufferfishCache(record.Storage.DataPath) != nil {
		p.Reason = "A cached Pufferfish server JAR or an unreadable data directory prevents verifying the exact target build."
		return p, nil
	}
	p.ManagedArtifacts = managed.Artifacts
	p.RuntimeDigest = managed.Digest
	if !operationFingerprintPattern.MatchString(p.RuntimeDigest) {
		p.Reason = "The exact target runtime image could not be verified."
		return p, nil
	}
	p.Content, p.ContentFingerprint, err = switchContentEvidence(ctx, record, target, p.Version, managed.Artifacts)
	if err != nil {
		p.Reason = "Plugins & Mods inventory could not be verified. Refresh Switch after resolving inventory access."
		return p, nil
	}
	if p.Runtime.CrossplayEnabled {
		if switchCrossplayPlatform(target) != "spigot" || b.verifySwitchPaperCrossplay(ctx, record, p.ManagedArtifacts) != nil {
			p.MatrixPolicyBlocked = true
			p.Reason = "The installed managed cross-play files do not match the reviewed target stack. Switch preserves the installed files."
			return p, nil
		}
	}
	layoutVerified := true
	if p.Current == "leaf" && verifySwitchLeafWorld(record.Storage.DataPath) != nil {
		layoutVerified = false
		p.Reason = "Leaf world region files could not be verified as Anvil. The target has no reviewed Linear region support."
	}
	if switchNeedsWorldMove(p.Current, target) {
		if moves, layoutErr := inspectSwitchWorld(record.Storage.DataPath, p.Version); layoutErr != nil {
			layoutVerified = false
			p.Reason = "The Paper world layout is missing or ambiguous. A verified world transformation is required before conversion."
		} else {
			p.WorldFingerprint = instanceHash(mustSwitchJSON(moves))
		}
	}
	if switchWorldRisk(p.Current, target) {
		p.Warnings = append(p.Warnings, "Mod-added blocks, items, entities and other world data may not survive. An empty mods folder does not establish world safety; only a protected experimental clone is available.")
	}
	p.Convert, p.Clone = switchTransitionActions(p.Current, target, true, layoutVerified, p.Content)
	if record.Runtime.BedrockEnabled && !p.Runtime.CrossplayEnabled && javaServerContentKind(p.Current) == "mods" && javaServerContentKind(p.Target) == "mods" {
		for _, item := range p.Content {
			if item.Compatibility != "verified" {
				p.Convert, p.Clone = false, false
				p.Reason = "Installed mod files could not be verified for the Java-only target loader. Switch preserves those files and will not launch an incompatible stack."
				break
			}
		}
	}
	p.MatrixPolicyBlocked = !p.Convert || !p.Clone
	p.Warnings = append(p.Warnings, "No plugins or mods are removed, disabled, replaced, updated or downloaded by Switch.")
	if !p.Convert && p.Reason == "" {
		p.Reason = "Convert is blocked by world safety or incompatible or unverified plugins/mods. Review the named items in Plugins & Mods. An experimental clone preserves the original; its runtime must still pass verification."
	}
	raw, err = json.Marshal(struct {
		ID, Target, Version, Runtime, Content, Record, World string
		Convert, Clone                                       bool
		PufferfishBuild                                      int `json:",omitempty"`
		ManagedArtifacts                                     map[string]switchManagedArtifact
	}{id, target, p.Version, p.Runtime.PlanFingerprint, p.ContentFingerprint, instanceHash(mustSwitchJSON(record)), p.WorldFingerprint, p.Convert, p.Clone, p.Runtime.PufferfishBuild, p.ManagedArtifacts})
	if err != nil {
		return p, err
	}
	p.Fingerprint = "sha256:" + instanceHash(raw)
	return p, nil
}
func mustSwitchJSON(value any) []byte { raw, _ := json.Marshal(value); return raw }

func registerMinecraftSwitchRoutes(mux *http.ServeMux, s *server) {
	mux.HandleFunc("GET /v1/minecraft/instances/{id}/switch/plan", s.minecraftSwitchPlan)
	mux.HandleFunc("POST /v1/minecraft/instances/{id}/switch/apply", s.minecraftSwitchApply)
	mux.HandleFunc("GET /v1/minecraft/instances/{id}/switch/operation", s.minecraftSwitchOperation)
	mux.HandleFunc("POST /v1/minecraft/instances/{id}/switch/recover", s.minecraftSwitchRecover)
}
func (s *server) minecraftSwitchPlan(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdministrator(w, r); !ok {
		return
	}
	id, ok := validatedVersionInstance(w, r, false)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()
	p, err := planMinecraftSwitch(ctx, id, r.URL.Query().Get("target"))
	if err != nil {
		instanceRuntimeError(w, err, id)
		return
	}
	writeJSON(w, 200, p)
}

type minecraftSwitchRequest struct {
	Target      string `json:"target"`
	Action      string `json:"action"`
	Fingerprint string `json:"plan_fingerprint"`
	Confirm     bool   `json:"confirm"`
}

// This private orchestration evidence composes existing runtime creation,
// restore and maintenance transactions; it never replaces their journals.
type minecraftSwitchEvidence struct {
	Schema               int                     `json:"schema"`
	OperationID          string                  `json:"operation_id"`
	Source               string                  `json:"source"`
	Target               string                  `json:"target"`
	CloneID              string                  `json:"clone_id,omitempty"`
	CloneRecord          *instanceRuntimeRecord  `json:"clone_record,omitempty"`
	Version              string                  `json:"version"`
	Action               string                  `json:"action"`
	OriginallyRunning    bool                    `json:"originally_running"`
	SourceStopped        bool                    `json:"source_stopped"`
	SourceRecord         *instanceRuntimeRecord  `json:"source_record,omitempty"`
	SourceContentChanged bool                    `json:"source_content_changed,omitempty"`
	SourceRestoreJournal *instanceRestoreJournal `json:"source_restore_journal,omitempty"`
	ColdArchive          string                  `json:"cold_archive,omitempty"`
	WorldChanged         bool                    `json:"world_changed,omitempty"`
	Stage                string                  `json:"stage"`
}

func switchEvidencePath(operation string) string {
	return filepath.Join(instanceRuntimes.staging, "switch-"+operation+".json")
}
func saveSwitchEvidence(e minecraftSwitchEvidence) error {
	directory, err := instanceRuntimes.privateDirectory(instanceRuntimes.staging, true)
	if err != nil {
		return err
	}
	directory.Close()
	return instanceRuntimes.writeJSON(switchEvidencePath(e.OperationID), e, true)
}
func readSwitchEvidence(j operationJournal) (minecraftSwitchEvidence, error) {
	var e minecraftSwitchEvidence
	raw, err := instanceRuntimes.readPrivate(switchEvidencePath(j.OperationID))
	if err != nil {
		return e, err
	}
	if instanceDecode(raw, &e) != nil || e.Schema != 1 || e.OperationID != j.OperationID || e.Source != j.InstanceID || !switchStructuralPathFromEvidence(e) {
		return e, errInstanceRecovery
	}
	return e, nil
}
func switchStructuralPathFromEvidence(e minecraftSwitchEvidence) bool {
	if !instanceIDPattern.MatchString(e.Source) || e.SourceRecord == nil || e.SourceRecord.Runtime.ID != e.Source || !switchStructuralPath(e.SourceRecord.Runtime.Software, e.Target) || e.Version == "LATEST" || !instanceRuntimeVersionPattern.MatchString(e.Version) || (e.Action != "convert" && e.Action != "clone") {
		return false
	}
	if e.CloneID != "" && (e.Action != "clone" || !instanceIDPattern.MatchString(e.CloneID) || e.CloneID == e.Source || e.CloneRecord == nil || e.CloneRecord.Runtime.ID != e.CloneID || e.CloneRecord.Storage.Root != e.SourceRecord.Storage.Root || e.CloneRecord.Storage.DataPath != filepath.Join(e.SourceRecord.Storage.Root, "minecraft-instances", e.CloneID, "data")) {
		return false
	}
	if e.WorldChanged && (e.Action != "convert" || !validInstanceRestoreArchive(e.ColdArchive)) {
		return false
	}
	return true
}

func (s *server) minecraftSwitchApply(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdministrator(w, r); !ok {
		return
	}
	id, ok := validatedVersionInstance(w, r)
	if !ok {
		return
	}
	var request minecraftSwitchRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if !request.Confirm || !operationFingerprintPattern.MatchString(request.Fingerprint) || (request.Action != "convert" && request.Action != "clone") {
		writeError(w, 400, "Review and confirm Switch first.")
		return
	}
	if s.operations == nil {
		writeError(w, 503, "Switch operation tracking is unavailable.")
		return
	}
	// A lost Apply response reconnects to its immutable operation before planning.
	if current, err := s.operations.currentMinecraftMaintenance(); err != nil {
		writeError(w, 503, "Switch operation tracking is unavailable.")
		return
	} else if current != nil {
		if current.InstanceID == id && current.OperationType == operationTypeMinecraftSwitch && current.PlanFingerprint == switchActionFingerprint(request) {
			writeJSON(w, 202, adminOperationResponse{Operation: current})
			return
		}
		writeError(w, 409, "Another maintenance operation requires attention.")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()
	p, err := planMinecraftSwitch(ctx, id, request.Target)
	if err != nil {
		instanceRuntimeError(w, err, id)
		return
	}
	if p.Fingerprint != request.Fingerprint || (request.Action == "convert" && !p.Convert) || (request.Action == "clone" && !p.Clone) {
		writeError(w, 409, "Switch compatibility changed or the action is unavailable. Refresh and review again.")
		return
	}
	j, created, err := s.operations.beginMinecraftMaintenance(switchActionFingerprint(request), operationTypeMinecraftSwitch, id)
	if err != nil {
		instanceRuntimeError(w, err, id)
		return
	}
	if created {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
			defer cancel()
			s.runMinecraftSwitch(ctx, j, p, request)
		}()
	}
	writeJSON(w, 202, adminOperationResponse{Operation: &j})
}
func switchActionFingerprint(r minecraftSwitchRequest) string {
	return "sha256:" + instanceHash([]byte(r.Fingerprint+":"+r.Action+":"+r.Target))
}
func (s *server) minecraftSwitchOperation(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdministrator(w, r); !ok {
		return
	}
	id, ok := validatedVersionInstance(w, r, false)
	if !ok {
		return
	}
	if s.operations == nil {
		writeError(w, 503, "Switch tracking is unavailable.")
		return
	}
	var j *operationJournal
	if requested := r.URL.Query().Get("operation_id"); requested != "" {
		found, err := s.operations.get(requested)
		if err != nil {
			writeError(w, 503, "Switch tracking is unavailable.")
			return
		}
		j = &found
	} else {
		j = s.operations.latestUnresolvedMinecraftSwitch(id)
	}
	if j != nil && (j.InstanceID != id || j.OperationType != operationTypeMinecraftSwitch || j.MatrixParent != "") {
		j = nil
	}
	response := map[string]any{"operation": j}
	if j != nil {
		if evidence, e := readSwitchEvidence(*j); e == nil {
			response["target"] = evidence.Target
			response["action"] = evidence.Action
			response["version"] = evidence.Version
			if evidence.CloneID != "" {
				response["clone_id"] = evidence.CloneID
			}
		}
	}
	writeJSON(w, 200, response)
}

func (s *operationStore) latestUnresolvedMinecraftSwitch(id string) *operationJournal {
	s.mu.Lock()
	defer s.mu.Unlock()
	if current, ok := s.operations[s.currentMinecraftResetID]; ok && current.InstanceID == id && current.OperationType == operationTypeMinecraftSwitch && current.MatrixParent == "" && operationIsCurrent(current.State) {
		return &current
	}
	var latest *operationJournal
	var started time.Time
	for _, j := range s.operations {
		if j.InstanceID != id || j.OperationType != operationTypeMinecraftSwitch || j.MatrixParent != "" || !operationIsCurrent(j.State) {
			continue
		}
		at, err := time.Parse(time.RFC3339Nano, j.StartedAt)
		if err == nil && (latest == nil || at.After(started) || at.Equal(started) && j.OperationID > latest.OperationID) {
			latest, started = &j, at
		}
	}
	return latest
}

func (s *server) runMinecraftSwitch(ctx context.Context, j operationJournal, p minecraftSwitchPlan, r minecraftSwitchRequest) (stage, runtimeDiagnostic string) {
	e := minecraftSwitchEvidence{Schema: 1, OperationID: j.OperationID, Source: j.InstanceID, Target: p.Target, Version: p.Version, Action: r.Action, Stage: "validating"}
	if j.MatrixParent != "" {
		if !s.operations.authorizeQueuedMatrixSwitch(j) {
			return "ownership", ""
		}
		ctx = context.WithValue(ctx, switchMatrixChildKey{}, j)
	}
	lock, err := acquirePluginMaintenanceLock()
	source, readErr := instanceRuntimes.readRecord(j.InstanceID)
	if readErr == nil {
		e.SourceRecord = &source
		if switchStructuralPathFromEvidence(e) {
			readErr = saveSwitchEvidence(e)
		} else {
			readErr = errInstanceRecovery
		}
	}
	err = errors.Join(err, readErr)
	if _, transitionErr := s.operations.transition(j.OperationID, operationValidating, "validating", "Revalidating Switch at the same Minecraft version."); transitionErr != nil {
		_ = lock.Close()
		s.operations.recordSwitchFailure(e, errors.Join(err, transitionErr))
		return "validating", ""
	}
	if lock != nil {
		if err == nil {
			if prior, journalErr := instanceRuntimes.restoreJournal(j.InstanceID); journalErr == nil {
				// Completed journals may predate ordinary configuration edits.
				// Capture their identity; never treat a terminal binding as active recovery.
				if !restoreTerminal(prior.Stage) {
					err = errInstanceRecovery
				} else {
					e.SourceRestoreJournal = &prior
				}
			} else if !errors.Is(journalErr, os.ErrNotExist) {
				err = journalErr
			}
		}
		if err == nil {
			state, stateErr := instanceRuntimes.observe(ctx, j.InstanceID)
			if stateErr != nil || (state.State != "running" && state.State != "stopped") {
				err = errors.Join(errInstanceBusy, stateErr)
			} else {
				e.OriginallyRunning = state.State == "running"
			}
		}
		if err == nil {
			e.Stage = "reviewed"
			err = saveSwitchEvidence(e)
		}
		if err == nil {
			var refreshed minecraftSwitchPlan
			refreshed, err = planMinecraftSwitch(ctx, j.InstanceID, p.Target)
			if err == nil && (refreshed.Fingerprint != p.Fingerprint || r.Action == "convert" && !refreshed.Convert || r.Action == "clone" && !refreshed.Clone) {
				err = errInstanceBusy
			}
		}
		if err == nil {
			_, err = s.operations.transition(j.OperationID, operationRunning, "converting", "Creating a consistent copy or cold backup and verifying the exact target runtime.")
		}
		if err == nil && r.Action == "convert" {
			err = s.convertMinecraftSwitch(ctx, &e, p, lock)
		} else if err == nil {
			err = s.cloneMinecraftSwitch(ctx, &e, p, lock)
		}
		if err == nil && j.MatrixParent != "" {
			if bindErr := s.operations.finalizeSwitchMatrixChild(ctx, j.MatrixParent, e); bindErr != nil {
				_ = lock.Close()
				s.operations.recordSwitchFailure(e, bindErr)
				_, _ = s.operations.transition(j.OperationID, operationNeedsAttention, "fixture_binding", "Verified Switch fixture ownership could not be finalized. Recovery evidence was preserved.")
				return "fixture_binding", ""
			}
		}
		err = errors.Join(err, lock.Close())
	}
	if err != nil {
		failureStage := e.Stage
		s.operations.recordSwitchFailure(e, err)
		s.operations.recordSwitchCloneRuntime(e)
		// Rollback composes the existing creation/restore/runtime recovery paths. It
		// never re-plans or retries the requested conversion.
		recovered := false
		if lock != nil && e.SourceRecord != nil {
			recovery, recoveryCancel := context.WithTimeout(context.Background(), 30*time.Minute)
			if j.MatrixParent != "" {
				recovery = context.WithValue(recovery, switchMatrixChildKey{}, j)
			}
			recoveryErr := s.recoverMinecraftSwitch(recovery, &e)
			if recoveryErr == nil && j.MatrixParent != "" {
				recoveryErr = s.operations.refreshSwitchMatrixFixture(recovery, j.MatrixParent, e, true)
			}
			recoveryCancel()
			s.operations.recordSwitchRecovery(e, recoveryErr)
			recovered = recoveryErr == nil
		}
		_, transitionErr := s.operations.transition(j.OperationID, operationNeedsAttention, "switch_failed", "Switch did not complete. Review Switch recovery; the conversion will not be retried automatically.")
		if recovered && transitionErr == nil {
			message := "Switch failed. The original server was recovered and verified. No conversion was retried."
			if e.Action == "clone" {
				message = "Switch failed. The original server was recovered and verified; the failed clone was cleaned up. No conversion was retried."
			}
			_, _ = s.operations.transition(j.OperationID, operationResolved, "resolved", message)
		}
		return failureStage, switchRuntimeDiagnostic(err)
	}
	_, err = s.operations.transition(j.OperationID, operationVerifying, "verifying", "Exact target Minecraft runtime verified.")
	if err == nil {
		_, err = s.operations.transition(j.OperationID, operationSucceeded, "complete", "Switch completed at Minecraft "+p.Version+".")
	}
	if err != nil {
		return "journal_finalization", ""
	}
	return "", ""
}

func (s *server) minecraftSwitchRecover(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdministrator(w, r); !ok {
		return
	}
	id, ok := validatedVersionInstance(w, r, false)
	if !ok {
		return
	}
	var request struct {
		OperationID string `json:"operation_id"`
		Confirm     bool   `json:"confirm"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	if !request.Confirm || s.operations == nil {
		writeError(w, 400, "Confirm Switch recovery.")
		return
	}
	j, err := s.operations.currentMinecraftMaintenance()
	if err != nil || j == nil || j.InstanceID != id || j.OperationID != request.OperationID || j.OperationType != operationTypeMinecraftSwitch || j.State != operationNeedsAttention {
		writeError(w, 409, "No Switch recovery is available.")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Minute)
	defer cancel()
	lock, err := acquirePluginMaintenanceLock()
	if err != nil {
		s.operations.recordSwitchRecovery(minecraftSwitchEvidence{OperationID: j.OperationID}, err)
		instanceRuntimeError(w, err, id)
		return
	}
	defer lock.Close()
	// Recheck after taking maintenance ownership so a queued duplicate cannot
	// recover evidence from an operation that has since been resolved.
	current, err := s.operations.currentMinecraftMaintenance()
	if err != nil || current == nil || current.OperationID != j.OperationID || current.State != operationNeedsAttention {
		writeError(w, 409, "No Switch recovery is available.")
		return
	}
	e, err := readSwitchEvidence(*j)
	if err != nil {
		e = minecraftSwitchEvidence{OperationID: j.OperationID}
	}
	if errors.Is(err, os.ErrNotExist) {
		// No evidence means no worker mutation was authorized. Verify source only.
		err = instanceRuntimes.verifyCommittedSwitchState(ctx, id)
		if err != nil {
			err = switchRecoveryRefusal("source verification failed")
		}
	} else if err == nil {
		err = s.recoverMinecraftSwitchLocked(ctx, &e, lock)
	} else {
		err = switchRecoveryRefusal("Switch recovery evidence validation failed")
	}
	s.operations.recordSwitchRecovery(e, err)
	if err != nil {
		instanceRuntimeError(w, err, id)
		return
	}
	resolved, err := s.operations.transition(j.OperationID, operationResolved, "resolved", "Switch stopped. Original server recovered and verified; no conversion was retried.")
	if err != nil {
		instanceRuntimeError(w, err, id)
		return
	}
	writeJSON(w, 200, adminOperationResponse{Operation: &resolved})
}

// Rechecked by the existing worker after cold shutdown, before publishing the
// new software. This closes the running-server content race after Apply review.
func verifySwitchContent(ctx context.Context, id, target, version, fingerprint string) error {
	if !instanceSHA256Pattern.MatchString(fingerprint) {
		return errInstanceID
	}
	record, err := instanceRuntimes.readRecord(id)
	if err != nil {
		return err
	}
	if !switchStructuralPath(record.Runtime.Software, target) {
		return errInstanceID
	}
	ctx = context.WithValue(ctx, instanceVersionKey{}, id)
	raw, err := runAdminVersionStatus(ctx, "pinned", version, target, "")
	var runtime struct {
		Artifacts           map[string]switchManagedArtifact `json:"managed_artifacts"`
		CrossplayEnabled    bool                             `json:"crossplay_enabled"`
		CrossplayCompatible bool                             `json:"crossplay_compatible"`
	}
	if err != nil || json.Unmarshal(raw, &runtime) != nil || runtime.CrossplayEnabled && !runtime.CrossplayCompatible {
		return errInstanceRecovery
	}
	content, hash, err := switchContentEvidence(ctx, record, target, version, runtime.Artifacts)
	if err != nil || hash != fingerprint {
		return instanceRuntimeFailure("Plugins & Mods changed; review Switch again.")
	}
	for _, item := range content {
		if item.Compatibility != "verified" {
			return instanceRuntimeFailure("Plugins & Mods compatibility is unverified; review Switch again.")
		}
	}
	return nil
}
