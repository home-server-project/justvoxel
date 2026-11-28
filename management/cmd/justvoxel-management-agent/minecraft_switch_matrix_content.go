//go:build justvoxel_dev

package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

// Select exact releases for two fixed fixture projects using the same catalog,
// compatibility, artifact verification and ownership format as Plugins & Mods.
func switchMatrixFixturePlugin(ctx context.Context, server pluginServerContext, slug string, needsDependency bool) (pluginInstallArtifact, error) {
	var project struct {
		ID   string `json:"id"`
		Slug string `json:"slug"`
	}
	if err := modrinthPluginJSON(ctx, "/project/"+slug, nil, &project); err != nil ||
		project.Slug != slug || !modrinthPluginID.MatchString(project.ID) {
		return pluginInstallArtifact{}, errSwitchMatrixProfile
	}
	games, _ := json.Marshal([]string{server.MinecraftVersion})
	loaders, _ := json.Marshal(pluginLoaders(server.ServerType))
	var releases []pluginCatalogVersion
	if err := modrinthPluginJSON(ctx, "/project/"+project.ID+"/version", url.Values{
		"game_versions": {string(games)}, "loaders": {string(loaders)}, "include_changelog": {"false"},
	}, &releases); err != nil {
		return pluginInstallArtifact{}, errSwitchMatrixProfile
	}
	slices.SortFunc(releases, func(a, b pluginCatalogVersion) int {
		if order := strings.Compare(b.Published, a.Published); order != 0 {
			return order
		}
		return strings.Compare(a.ID, b.ID)
	})
	for _, release := range releases {
		if release.ProjectID != project.ID || !modrinthPluginID.MatchString(release.ID) ||
			release.Type != "release" || !pluginVersionCompatible(release, server) ||
			!modrinthContentEnvironment(release.Environment, server.ServerType) || release.Dependencies == nil ||
			release.Number == "" || len(release.Number) > 256 {
			continue
		}
		required := slices.ContainsFunc(release.Dependencies, func(d pluginDependency) bool {
			return d.Type == "required"
		})
		if required != needsDependency {
			continue
		}
		f, err := selectPluginArtifact(release.Files)
		if err == nil {
			return pluginInstallArtifact{Target: "plugins", Source: "Modrinth", ProjectID: project.ID,
				VersionID: release.ID, VersionNumber: release.Number, Filename: f.Filename, Size: f.Size,
				SHA512: f.Hashes.SHA512, URL: f.URL, Requested: true, RequiredBy: []string{}}, nil
		}
	}
	return pluginInstallArtifact{}, errSwitchMatrixProfile
}

func (s *server) seedSwitchMatrixManagedContent(ctx context.Context, parent, id string) (result error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	lock, err := acquirePluginMaintenanceLock()
	if err != nil {
		return wrapSwitchMatrixFixtureError(matrixFixtureOwnership, err)
	}
	defer func() {
		if err := lock.Close(); err != nil {
			result = &switchMatrixFixtureError{stage: matrixFixtureOwnership, cause: errors.Join(result, err)}
		}
	}()
	record, err := s.operations.switchMatrixOwnedRuntime(ctx, parent, id)
	if err != nil {
		return wrapSwitchMatrixFixtureError(matrixFixtureOwnership, err)
	}
	b := instanceRuntimes
	if err := b.deleteStopped(ctx, id, false); err != nil {
		return err
	}
	data := record.Storage.DataPath
	manifest, manifestBytes, err := readPluginManifest(data)
	if err != nil || len(manifest.Artifacts) != 0 {
		return errSwitchMatrixProfile
	}
	plugins, err := openContentDirectory(data, "plugins")
	if err != nil {
		return err
	}
	defer plugins.Close()
	staging, err := os.MkdirTemp(data, ".switch-matrix-content-")
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, os.RemoveAll(staging)) }()
	server := pluginServerContext{ServerType: record.Runtime.Software, MinecraftVersion: record.Runtime.Version}
	plan := pluginInstallPlan{Server: server}
	for _, slug := range []string{"luckperms", "viabackwards"} {
		artifact, err := switchMatrixFixturePlugin(ctx, server, slug, slug == "viabackwards")
		if err != nil {
			return err
		}
		if err = downloadPluginArtifact(ctx, artifact, staging); err != nil {
			return err
		}
		file, err := os.Open(filepath.Join(staging, artifact.Filename))
		if err != nil {
			return err
		}
		identity, inspectErr := inspectServerPlugin(file, artifact.Filename, server.ServerType)
		if err = errors.Join(inspectErr, file.Close()); err != nil {
			return err
		}
		if !strings.EqualFold(identity.Name, slug) {
			return errSwitchMatrixProfile
		}
		artifact.PluginName, artifact.Main = identity.Name, identity.Main
		plan.Artifacts = append(plan.Artifacts, artifact)
	}
	if err = checkPluginInstallCollisions(plugins, plan); err != nil {
		return err
	}
	for _, artifact := range plan.Artifacts {
		if _, err = placePluginArtifact(plugins, staging, parent, artifact); err != nil {
			return err
		}
		manifest.Artifacts = append(manifest.Artifacts, pluginManifestArtifact(artifact))
	}
	// A genuine plugin JAR in mods/ represents foreign executable content.
	// A Paper-family server ignores that directory. Switch still inventories it.
	root, err := openPluginDataDirectory(data)
	if err != nil {
		return err
	}
	defer root.Close()
	if err = unix.Mkdirat(int(root.Fd()), "mods", 0755); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	mods, err := openContentDirectory(data, "mods")
	if err != nil {
		return err
	}
	defer mods.Close()
	foreign := plan.Artifacts[0]
	foreign.Target = "mods"
	if _, err = placePluginArtifact(mods, staging, parent, foreign); err != nil {
		return err
	}
	changed, err := commitPluginManifest(data, manifestBytes, manifest)
	if err != nil {
		return errors.Join(errSwitchEvidencePersistence, err)
	}
	if !changed {
		return errSwitchMatrixProfile
	}
	return nil
}
