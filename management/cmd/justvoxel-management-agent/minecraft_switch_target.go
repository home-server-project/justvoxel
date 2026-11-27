package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"strings"
)

type switchDefinitionVerificationKey struct{}

func (b *instanceRuntimeBackend) switchTargetConfig(ctx context.Context, source minecraftInstanceRuntimeConfig, p minecraftSwitchPlan) (minecraftInstanceRuntimeConfig, error) {
	c := source
	c.Software, c.Version, c.VersionPolicy = p.Target, p.Version, "pinned"
	c.LoaderVersion, c.PufferfishBuild = p.Runtime.LoaderVersion, p.Runtime.PufferfishBuild
	c.MatrixLeafBuild = p.MatrixLeafBuild
	c.ImageTag, c.ImageDigest = p.Runtime.ImageTag, p.RuntimeDigest
	c.PreserveContent = true
	c.BedrockEnabled = p.Runtime.CrossplayEnabled
	c.BedrockTCPPort, c.BedrockUDPStart, c.BedrockUDPEnd = 0, 0, 0
	if !c.BedrockEnabled {
		c.BedrockPort = 0
	}
	if !switchStructuralPath(source.Software, c.Software) || c.Version == "LATEST" ||
		c.BedrockEnabled != (source.BedrockEnabled && javaServerCrossplay(c.Software)) ||
		!operationFingerprintPattern.MatchString(c.ImageDigest) || !validPufferfishBuild(c.Software, c.PufferfishBuild) {
		return c, errInstanceRecovery
	}
	channel, err := b.command(ctx, instanceRuntimeHelper, "channel", c.Software, c.Version, c.VersionPolicy)
	if err != nil {
		return c, err
	}
	c.PaperChannel = strings.TrimSpace(string(channel))
	return c, nil
}

func (b *instanceRuntimeBackend) prepareSwitchTarget(ctx context.Context, source instanceRuntimeRecord, p minecraftSwitchPlan, clone bool) (instanceRuntimeRecord, minecraftInstanceRuntimeContext, error) {
	snapshot, _, _, err := b.snapshot(ctx)
	if err != nil {
		return instanceRuntimeRecord{}, snapshot, err
	}
	if p.Target == "pufferfish" {
		if err := verifySwitchPufferfishCache(source.Storage.DataPath); err != nil {
			return instanceRuntimeRecord{}, snapshot, err
		}
	}
	records, err := b.records()
	if err != nil {
		return instanceRuntimeRecord{}, snapshot, err
	}
	next := source
	next.Runtime, err = b.switchTargetConfig(ctx, source.Runtime, p)
	if err != nil {
		return next, snapshot, err
	}
	candidate, storageCheck := "", "preflight"
	if clone {
		var id [6]byte
		if _, err = rand.Read(id[:]); err != nil {
			return next, snapshot, err
		}
		secret, err := newInstanceRCONPassword()
		if err != nil {
			return next, snapshot, err
		}
		candidate, storageCheck = "jv-"+hex.EncodeToString(id[:]), "preflight-new"
		c := &next.Runtime
		c.ID, c.RCONPassword, c.StartOnBoot = candidate, secret, false
		if err = switchClonePorts(records, c); err != nil {
			return next, snapshot, err
		}
		snapshot.StorageRoot, snapshot.MountPoint, snapshot.ExpectedUUID, snapshot.ExpectedSource = source.Storage.Root, source.Storage.MountPoint, source.Storage.UUID, source.Storage.Source
		if source.Backups != nil {
			snapshot.BackupPlacement = source.Backups
			snapshot.BackupPath = source.Backups.Root
		}
		next = instanceRuntimeRecord{SchemaVersion: 1, Runtime: *c, Storage: instanceStorageFor(snapshot, candidate),
			Backups: instanceBackupsFor(snapshot, candidate), Firewall: []instanceFirewallRule{}}
	}
	if _, err = b.renderCandidate(snapshot, instanceReplaceRecord(records, next), candidate); err != nil {
		return next, snapshot, err
	}
	if err = b.storage(ctx, storageCheck, next); err != nil {
		return next, snapshot, err
	}
	return next, snapshot, b.validateSoftware(ctx, next.Runtime)
}

// itzg can reuse a Pufferfish jar by filename even when the selected CI build
// changed. The filename contains the Minecraft release, not the CI build.
func verifySwitchPufferfishCache(data string) error {
	root, err := os.OpenRoot(data)
	if err != nil {
		return err
	}
	defer root.Close()
	dir, err := root.Open(".")
	if err != nil {
		return err
	}
	defer dir.Close()
	entries, err := dir.ReadDir(10001)
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	if len(entries) > 10000 {
		return errInstanceRecovery
	}
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, "pufferfish-") && strings.HasSuffix(name, ".jar") {
			return instanceRuntimeFailure("A cached Pufferfish server JAR prevents verifying the exact target build.")
		}
	}
	return nil
}
