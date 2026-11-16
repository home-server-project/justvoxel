package main

import (
	"context"
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
	"strings"
	"time"
)

// Switch deliberately supports only implemented world/runtime paths. Loader
// names, an empty mods directory, or Bukkit support never prove world safety.
func switchStructuralPath(from, to string) bool {
	return from != to && javaServerPaperFamily(from) && javaServerPaperFamily(to)
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

type minecraftSwitchPlan struct {
	InstanceID         string             `json:"instance_id"`
	Current            string             `json:"current"`
	Target             string             `json:"target"`
	Version            string             `json:"version"`
	Convert            bool               `json:"convert"`
	Clone              bool               `json:"clone"`
	Reason             string             `json:"reason"`
	Content            []switchContent    `json:"content"`
	Warnings           []string           `json:"warnings"`
	Fingerprint        string             `json:"plan_fingerprint"`
	Runtime            adminVersionStatus `json:"-"`
	RuntimeDigest      string             `json:"-"`
	ContentFingerprint string             `json:"-"`
}

func switchUnsupportedReason(from, to string) string {
	if from == "bedrock" || to == "bedrock" {
		return "Java and native Bedrock worlds and runtimes cannot be converted by JustVoxel."
	}
	if from == "spigot" || to == "spigot" {
		return "No verified Spigot world transformation is implemented. Paper-family to Spigot is unsafe; direct Spigot to Paper-family is unsupported for Minecraft 26.1 and newer."
	}
	if from == "vanilla" || to == "vanilla" {
		return "The required Vanilla/Paper world-layout transformation is not implemented and verified by JustVoxel."
	}
	return "Loader compatibility and historic modded world data cannot be verified for this conversion. Removing current mods does not prove world safety."
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
		} else if kind == "plugins" && entry != nil && pluginTarget(entry.Target) == "plugins" && pluginSource(entry.Source) == "Modrinth" && modrinthPluginID.MatchString(entry.VersionID) {
			root, e := openPluginDirectory(data)
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
	imageTag := record.Runtime.ImageTag
	if target == "pufferfish" || target == "leaf" {
		bedrock := "no"
		if record.Runtime.BedrockEnabled {
			bedrock = "yes"
		}
		resolved, resolveErr := runSetupVersionPreview(ctx, "pinned", p.Version, bedrock, target)
		var selection adminVersionStatus
		if resolveErr != nil || json.Unmarshal(resolved, &selection) != nil || selection.SelectedCandidate != p.Version || selection.ImageTag == "" || !selection.ServerSupported {
			p.Reason = "The target runtime does not support this exact Minecraft version, or its availability could not be verified."
			return p, nil
		}
		imageTag = selection.ImageTag
	}
	raw, err = runAdminVersionStatus(ctx, "pinned", p.Version, target, imageTag)
	if err != nil || json.Unmarshal(raw, &p.Runtime) != nil || p.Runtime.ServerType != target || p.Runtime.SelectedCandidate != p.Version || p.Runtime.Installed != p.Version || !p.Runtime.ServerSupported || !operationFingerprintPattern.MatchString(p.Runtime.PlanFingerprint) || p.Runtime.StackState != "updates_available" || !p.Runtime.UpdateAvailable || (p.Runtime.CrossplayEnabled && !p.Runtime.CrossplayCompatible) {
		p.Reason = "The target runtime and managed cross-play could not be verified for Minecraft " + p.Version + ". Switch never upgrades or downgrades Minecraft."
		return p, nil
	}
	var managed struct {
		Artifacts map[string]switchManagedArtifact `json:"managed_artifacts"`
		Digest    string                           `json:"managed_image_digest"`
	}
	if json.Unmarshal(raw, &managed) != nil {
		return p, errInstanceRecovery
	}
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
	p.Clone, p.Convert = true, true
	if record.NetworkMember {
		p.Clone = false
		p.Reason = "Remove this server from Minecraft Network before cloning its standalone settings."
	}
	for _, content := range p.Content {
		if content.Compatibility != "verified" {
			p.Convert = false
		}
	}
	p.Warnings = append(p.Warnings, "No plugins or mods are removed, disabled, replaced, updated or downloaded by Switch.")
	if !p.Convert {
		p.Reason = "Convert is blocked by incompatible or unverified plugins/mods. Review the named items in Plugins & Mods. An experimental clone preserves the original; its runtime must still pass verification."
	}
	raw, err = json.Marshal(struct {
		ID, Target, Version, Runtime, Content, Record string
		Convert, Clone                                bool
	}{id, target, p.Version, p.Runtime.PlanFingerprint, p.ContentFingerprint, instanceHash(mustSwitchJSON(record)), p.Convert, p.Clone})
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
	Schema               int                    `json:"schema"`
	OperationID          string                 `json:"operation_id"`
	Source               string                 `json:"source"`
	Target               string                 `json:"target"`
	CloneID              string                 `json:"clone_id,omitempty"`
	CloneRecord          *instanceRuntimeRecord `json:"clone_record,omitempty"`
	Version              string                 `json:"version"`
	Action               string                 `json:"action"`
	OriginallyRunning    bool                   `json:"originally_running"`
	SourceStopped        bool                   `json:"source_stopped"`
	SourceRecord         *instanceRuntimeRecord `json:"source_record,omitempty"`
	SourceContentChanged bool                   `json:"source_content_changed,omitempty"`
	Stage                string                 `json:"stage"`
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
	if e.CloneID != "" && (!instanceIDPattern.MatchString(e.CloneID) || e.CloneID == e.Source || e.CloneRecord == nil || e.CloneRecord.Runtime.ID != e.CloneID || e.CloneRecord.Storage.Root != e.SourceRecord.Storage.Root || e.CloneRecord.Storage.DataPath != filepath.Join(e.SourceRecord.Storage.Root, "minecraft-instances", e.CloneID, "data")) {
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
	j, err := s.operations.currentMinecraftMaintenance()
	if requested := r.URL.Query().Get("operation_id"); requested != "" {
		found, e := s.operations.get(requested)
		err = e
		j = &found
	}
	if err != nil {
		writeError(w, 503, "Switch tracking is unavailable.")
		return
	}
	if j != nil && (j.InstanceID != id || j.OperationType != operationTypeMinecraftSwitch) {
		j = nil
	}
	response := map[string]any{"operation": j}
	if j != nil {
		if evidence, e := readSwitchEvidence(*j); e == nil {
			response["clone_id"] = evidence.CloneID
		}
	}
	writeJSON(w, 200, response)
}

// Retain ordinary error details on the privileged side; the persistent operation
// and original transaction journals are the authoritative recovery evidence.
func (s *server) runMinecraftSwitch(ctx context.Context, j operationJournal, p minecraftSwitchPlan, r minecraftSwitchRequest) {
	_, err := s.operations.transition(j.OperationID, operationValidating, "validating", "Revalidating Switch at the same Minecraft version.")
	if err != nil {
		return
	}
	e := minecraftSwitchEvidence{Schema: 1, OperationID: j.OperationID, Source: j.InstanceID, Target: p.Target, Version: p.Version, Action: r.Action, Stage: "reviewed"}
	lock, err := acquirePluginMaintenanceLock()
	if err == nil {
		source, readErr := instanceRuntimes.readRecord(j.InstanceID)
		if readErr != nil {
			err = readErr
		} else {
			e.SourceRecord = &source
		}
		if err == nil {
			state, stateErr := instanceRuntimes.observe(ctx, j.InstanceID)
			if stateErr != nil || (state.State != "running" && state.State != "stopped") {
				err = errInstanceBusy
			} else {
				e.OriginallyRunning = state.State == "running"
			}
		}
		if err == nil {
			err = saveSwitchEvidence(e)
		}
		if err == nil {
			var refreshed minecraftSwitchPlan
			refreshed, err = planMinecraftSwitch(ctx, j.InstanceID, p.Target)
			if err == nil && (refreshed.Fingerprint != p.Fingerprint || (r.Action == "convert" && !refreshed.Convert)) {
				err = errInstanceBusy
			}
		}
		if err == nil {
			_, err = s.operations.transition(j.OperationID, operationRunning, "converting", "Creating a consistent copy or cold backup and verifying the exact target runtime.")
		}
		if err == nil && r.Action == "convert" {
			// The mature worker owns the maintenance flock itself and rechecks the
			// underlying runtime fingerprint immediately before destructive work.
			err = lock.Close()
			if err == nil {
				worker := context.WithValue(ctx, switchWorkerKey{}, true)
				worker = context.WithValue(worker, switchContentKey{}, p.ContentFingerprint)
				worker = context.WithValue(worker, instanceVersionKey{}, j.InstanceID)
				worker = context.WithValue(worker, instanceVersionSelectionKey{}, [2]string{"pinned", p.Version})
				err = runMinecraftStackUpdate(worker, p.Runtime.PlanFingerprint, true, p.Target, p.Runtime.ImageTag)
				if err == nil {
					e.Stage = "complete"
					err = saveSwitchEvidence(e)
				}
			}
		} else if err == nil {
			err = s.cloneMinecraftSwitch(ctx, &e, p, lock)
		}
		err = errors.Join(err, lock.Close())
	}
	if err != nil {
		// Rollback composes the existing creation/restore/runtime recovery paths. It
		// never re-plans or retries the requested conversion.
		recovered := false
		if e.SourceRecord != nil {
			recovery, recoveryCancel := context.WithTimeout(context.Background(), 30*time.Minute)
			recovered = s.recoverMinecraftSwitch(recovery, &e) == nil
			recoveryCancel()
		}
		_, transitionErr := s.operations.transition(j.OperationID, operationNeedsAttention, "switch_failed", "Switch did not complete. Review Switch recovery; the conversion will not be retried automatically.")
		if recovered && transitionErr == nil {
			message := "Switch failed. The original server was recovered and verified. No conversion was retried."
			if e.Action == "clone" {
				message = "Switch failed. The original server was recovered and verified; the failed clone was cleaned up. No conversion was retried."
			}
			_, _ = s.operations.transition(j.OperationID, operationResolved, "resolved", message)
		}
		return
	}
	_, err = s.operations.transition(j.OperationID, operationVerifying, "verifying", "Exact target Minecraft runtime verified.")
	if err == nil {
		_, _ = s.operations.transition(j.OperationID, operationSucceeded, "complete", "Switch completed at Minecraft "+p.Version+".")
	}
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
	e, err := readSwitchEvidence(*j)
	if errors.Is(err, os.ErrNotExist) {
		// No evidence means no worker mutation was authorized. Verify source only.
		lock, lockErr := acquirePluginMaintenanceLock()
		if lockErr != nil {
			err = lockErr
		} else {
			err = instanceRuntimes.verifyCommittedSwitchState(ctx, id)
			err = errors.Join(err, lock.Close())
		}
	} else if err == nil {
		err = s.recoverMinecraftSwitch(ctx, &e)
	}
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
