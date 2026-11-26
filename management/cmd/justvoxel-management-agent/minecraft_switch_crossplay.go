package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"path/filepath"
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
