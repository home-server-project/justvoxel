package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func switchCrossplayPlatform(software string) string {
	if javaServerPaperFamily(software) {
		return "spigot"
	}
	if javaServerCrossplay(software) {
		return software
	}
	return ""
}

// These are the shared Paper-family components, identified by the reviewed
// publisher hashes and their canonical install locations.
func (b *instanceRuntimeBackend) verifySwitchPaperCrossplay(ctx context.Context, record instanceRuntimeRecord, artifacts map[string]switchManagedArtifact) error {
	dir, err := openPluginDirectory(record.Storage.DataPath)
	if err != nil {
		return err
	}
	defer dir.Close()
	for _, component := range []struct{ project, filename string }{
		{"geyser", "Geyser-Spigot.jar"}, {"floodgate", "floodgate-spigot.jar"},
	} {
		if !instanceSHA256Pattern.MatchString(artifacts[component.project].SHA256) {
			return errInstanceRecovery
		}
		jar, err := openPluginAt(dir, component.filename)
		if err != nil {
			return err
		}
		h := sha256.New()
		n, err := io.Copy(h, io.LimitReader(jar, pluginMaxJar+1))
		jar.Close()
		if err != nil || n > pluginMaxJar || hex.EncodeToString(h.Sum(nil)) != artifacts[component.project].SHA256 {
			return errInstanceRecovery
		}
		if _, err := b.command(ctx, "/usr/libexec/justvoxel/mjust/managed-plugin-files", filepath.Join(record.Storage.DataPath, "plugins"), component.project, component.filename, "--check"); err != nil {
			return err
		}
	}
	return nil
}

type switchCrossplayInstall struct {
	Directory string `json:"directory"`
	Filename  string `json:"filename"`
	SHA256    string `json:"sha256,omitempty"`
	SHA512    string `json:"sha512,omitempty"`
}

func switchModCrossplayComponent(data, filename, source, target string) string {
	switch strings.ToLower(filename) {
	case "geyser-fabric.jar", "geyser-neoforge.jar":
		return "geyser"
	case "floodgate-fabric.jar", "floodgate-neoforge.jar":
		return "floodgate"
	case "fabric-api.jar":
		return "fabric-api"
	}
	root, err := openContentDirectory(data, "mods")
	if err != nil {
		return ""
	}
	defer root.Close()
	file, err := openPluginAt(root, filename)
	if err != nil {
		return ""
	}
	defer file.Close()
	for _, loader := range []string{source, target} {
		capability, ok := javaServerSoftware(loader)
		if !ok || !capability.ModLoader {
			continue
		}
		ids, err := modArtifactIdentities(file, filename, loader)
		if err != nil {
			continue
		}
		switch {
		case slices.Contains(ids, "geyser-fabric") || slices.Contains(ids, "geyser_neoforge"):
			return "geyser"
		case slices.Contains(ids, "floodgate"):
			return "floodgate"
		case slices.Contains(ids, "fabric-api"):
			return "fabric-api"
		}
	}
	return ""
}

func reviewSwitchCrossplayInstall(target string, metadata map[string]switchManagedArtifact, content []switchContent) ([]switchCrossplayInstall, error) {
	platform := switchCrossplayPlatform(target)
	kind := javaServerContentKind(target)
	components := []string{"geyser", "floodgate"}
	var names []string
	switch platform {
	case "spigot":
		components = append(components, "viaversion")
		names = []string{"Geyser-Spigot.jar", "floodgate-spigot.jar", "ViaVersion.jar"}
	case "fabric":
		components = append(components, "fabric-api")
		names = []string{"Geyser-Fabric.jar", "floodgate-fabric.jar", "fabric-api.jar"}
	case "neoforge":
		names = []string{"Geyser-NeoForge.jar", "floodgate-neoforge.jar"}
	default:
		return nil, errInstanceRecovery
	}
	if len(metadata) != len(components) {
		return nil, errInstanceRecovery
	}
	out := make([]switchCrossplayInstall, 0, len(components))
	for i, component := range components {
		hash := metadata[component]
		if component == "viaversion" {
			if hash.SHA256 != "" || !pluginHashPattern.MatchString(hash.SHA512) {
				return nil, errInstanceRecovery
			}
		} else if hash.SHA512 != "" || !instanceSHA256Pattern.MatchString(hash.SHA256) {
			return nil, errInstanceRecovery
		}
		reuse := false
		for _, item := range content {
			if item.Directory != kind || (!strings.EqualFold(item.Filename, names[i]) && item.Component != component) {
				continue
			}
			if item.CloneDisposition == "retain" {
				if component != "viaversion" || item.SHA512 != hash.SHA512 || reuse {
					return nil, errInstanceRecovery
				}
				reuse = true
			}
		}
		if !reuse {
			out = append(out, switchCrossplayInstall{Directory: kind, Filename: names[i], SHA256: hash.SHA256, SHA512: hash.SHA512})
		}
	}
	return out, nil
}

func (b *instanceRuntimeBackend) installSwitchCloneCrossplay(ctx context.Context, e minecraftSwitchEvidence, p minecraftSwitchPlan) error {
	if e.CloneRecord == nil || !e.CloneCrossplayInstall || !e.CloneRecord.Runtime.BedrockEnabled ||
		e.CloneID != e.CloneRecord.Runtime.ID || e.CloneID == e.Source || p.CrossplayChoice != "preserve" {
		return errInstanceRecovery
	}
	artifacts := maps.Clone(p.ManagedArtifacts)
	if !slices.ContainsFunc(p.CrossplayInstall, func(item switchCrossplayInstall) bool { return item.SHA512 != "" }) {
		delete(artifacts, "viaversion")
	}
	_, err := b.command(ctx, "nsenter", "--mount=/proc/1/ns/mnt", "--", "env", "JUSTVOXEL_MAINTENANCE_LOCK_HELD=1",
		"/usr/libexec/justvoxel/mjust/instance-crossplay-start", e.CloneID, "--switch-reviewed", string(mustSwitchJSON(artifacts)))
	if err != nil {
		return errors.Join(errInstanceRecovery, &pluginRuntimeVerificationError{switchStatus: switchVerificationCrossplay})
	}
	return nil
}

// Only a failed, owned clone can reach this path. Its complete data tree is
// about to be retired, so an interrupted installer need not restore its JARs.
func (s *server) retireSwitchCloneCrossplay(ctx context.Context, e minecraftSwitchEvidence, record instanceRuntimeRecord) error {
	if !e.CloneCrossplayInstall {
		return nil
	}
	b := instanceRuntimes
	dir, _, _ := b.paths(e.CloneID)
	stagingPath := filepath.Join(dir, "crossplay-staging")
	if _, err := os.Lstat(stagingPath); err == nil {
		staging, err := b.privateDirectory(stagingPath, false)
		if err != nil {
			return errInstanceRecovery
		}
		staging.Close()
		root, err := os.OpenRoot(dir)
		if err != nil {
			return errInstanceRecovery
		}
		err = root.RemoveAll("crossplay-staging")
		root.Close()
		if err != nil {
			return errInstanceRecovery
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return errInstanceRecovery
	}
	if child, ok := ctx.Value(switchMatrixChildKey{}).(operationJournal); ok {
		return s.operations.retireFailedMatrixCloneMods(ctx, child, e, record)
	}
	ownership := filepath.Join(dir, "crossplay-mods.json")
	if _, err := os.Lstat(ownership); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return errInstanceRecovery
	}
	owned, err := runtimeOwnedModComponents(pluginServerContext{InstanceID: e.CloneID, ServerType: record.Runtime.Software}, record.Storage.DataPath)
	platform := switchCrossplayPlatform(record.Runtime.Software)
	if err != nil || !(platform == "fabric" && len(owned) == 3 || platform == "neoforge" && len(owned) == 2) {
		return errInstanceRecovery
	}
	return b.removePrivate(ownership)
}
