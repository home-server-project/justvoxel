//go:build justvoxel_dev

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

const (
	switchMatrixMemoryMiB   = 2560
	switchMatrixMinFree     = 12 << 30 // two worlds, staged Restore, cold archive and local image pulls
	switchMatrixWorldBudget = 1 << 30
)

// The test fixture supplies filesystem statistics at the host syscall boundary.
var switchMatrixStatfs = unix.Statfs

var errSwitchMatrixWorldMarkers = errors.New("Minecraft world marker verification failed")
var errSwitchMatrixStartup = errors.New("Minecraft startup failed")

func bindSwitchMatrixImages(ctx context.Context, cases []switchMatrixCase) {
	images := map[string]string{}
	for i := range cases {
		c := &cases[i]
		for _, release := range []*switchMatrixRelease{&c.SourceRelease, &c.TargetRelease} {
			if release.Runtime == nil {
				continue
			}
			r := *release.Runtime
			digest, ok := images[r.Image]
			if !ok {
				probe, cancel := context.WithTimeout(ctx, 30*time.Second)
				raw, err := instanceRuntimeCommand(probe, switchMatrixVersionHelper, "matrix-image", r.Image)
				cancel()
				if err == nil && operationFingerprintPattern.MatchString(strings.TrimSpace(string(raw))) {
					digest = strings.TrimSpace(string(raw))
				}
				images[r.Image] = digest
			}
			r.ImageDigest = digest
			release.Runtime = &r
		}
	}
}

func checkSwitchMatrixFreeSpace(path string) error {
	var fs unix.Statfs_t
	if err := switchMatrixStatfs(path, &fs); err != nil || fs.Bsize <= 0 ||
		fs.Bavail < uint64(switchMatrixMinFree)/uint64(fs.Bsize) || fs.Ffree < 4096 {
		return errInstanceBusy
	}
	return nil
}

func (s *server) prepareSwitchMatrixSource(ctx context.Context, c switchMatrixCase) (_ minecraftInstanceRuntimeConfig, _ minecraftInstanceRuntimeContext, result error) {
	stage := matrixFixtureStorage
	defer func() { result = wrapSwitchMatrixFixtureError(stage, result) }()
	b := instanceRuntimes
	snapshot, uid, gid, err := b.snapshot(ctx)
	if err != nil {
		return minecraftInstanceRuntimeConfig{}, snapshot, err
	}
	required := switchMatrixMemoryMiB
	if c.Action == "clone" {
		required *= 2
	}
	stage = matrixFixtureOwnership
	records, err := b.records()
	if err != nil {
		return minecraftInstanceRuntimeConfig{}, snapshot, err
	}
	if err = b.admitMemory(ctx, snapshot.TotalMemoryMiB, records,
		minecraftInstanceRuntimeConfig{ContainerMemory: strconv.Itoa(required) + "M"}, false); err != nil {
		return minecraftInstanceRuntimeConfig{}, snapshot, wrapSwitchMatrixFixtureError("resource_admission", err)
	}
	stage = matrixFixtureStorage
	root := filepath.Join(s.operations.baseDir, "switch-matrix-fixtures")
	backups := filepath.Join(root, "backups")
	graph, err := b.command(ctx, "podman", "info", "--format", "{{.Store.GraphRoot}}")
	graphRoot := strings.TrimSpace(string(graph))
	if err != nil || validateInstanceRuntimePath(graphRoot) != nil {
		return minecraftInstanceRuntimeConfig{}, snapshot, errInstanceRecovery
	}
	for _, path := range []string{s.operations.baseDir, graphRoot} {
		if err = checkSwitchMatrixFreeSpace(path); err != nil {
			return minecraftInstanceRuntimeConfig{}, snapshot, err
		}
	}
	for _, path := range []string{root, backups} {
		fd, e := b.privateDirectory(path, true)
		if e != nil {
			return minecraftInstanceRuntimeConfig{}, snapshot, e
		}
		if e = fd.Close(); e != nil {
			return minecraftInstanceRuntimeConfig{}, snapshot, e
		}
		if e = checkSwitchMatrixFreeSpace(path); e != nil {
			return minecraftInstanceRuntimeConfig{}, snapshot, e
		}
	}
	snapshot.StorageRoot, snapshot.MountPoint, snapshot.ExpectedUUID, snapshot.ExpectedSource = root, "", "", ""
	snapshot.BackupPath = backups
	snapshot.BackupPlacement = &instanceRuntimeBackupStorage{Type: "system", Root: backups}
	stage = matrixFixtureCredential
	secret, err := newInstanceRCONPassword()
	if err != nil {
		return minecraftInstanceRuntimeConfig{}, snapshot, err
	}
	stage = matrixFixtureSoftware
	channel, err := b.command(ctx, instanceRuntimeHelper, "channel", c.Source, c.Version, "pinned")
	if err != nil {
		return minecraftInstanceRuntimeConfig{}, snapshot, err
	}
	r := c.SourceRelease.Runtime
	if r == nil {
		return minecraftInstanceRuntimeConfig{}, snapshot, errInstanceRecovery
	}
	config := minecraftInstanceRuntimeConfig{MatrixFixture: true, PreserveContent: !c.Crossplay,
		Software: c.Source, Version: c.Version, VersionPolicy: "pinned", LoaderVersion: r.Loader,
		ImageTag: r.Image, ImageDigest: r.ImageDigest, PufferfishBuild: r.PufferfishBuild, MatrixLeafBuild: r.Build, PaperChannel: strings.TrimSpace(string(channel)),
		JavaMemory: switchMatrixHeap, ContainerMemory: switchMatrixMemory, UID: uid, GID: gid, BedrockEnabled: c.Crossplay,
		EULAAccepted: true, GameMode: "creative", Difficulty: "peaceful", MaxPlayers: 1, Whitelist: true, EnforceWhitelist: true,
		MOTD: "JustVoxel Switch test fixture", RCONPassword: secret}
	stage = matrixFixturePorts
	err = switchClonePorts(records, &config)
	return config, snapshot, err
}

// Only a bound receipt can authorize a fixture start or a graceful shutdown.
func (s *operationStore) switchMatrixOwnedRuntime(ctx context.Context, parent, id string) (instanceRuntimeRecord, error) {
	var zero instanceRuntimeRecord
	s.mu.Lock()
	j := s.operations[parent]
	owned := j.OperationType == operationTypeSwitchMatrix && s.currentMinecraftResetID == parent &&
		s.switchMatrixExclusion != nil && slices.Contains(j.MatrixFixtures, id)
	s.mu.Unlock()
	if !owned {
		return zero, errInstanceID
	}
	f, err := readSwitchMatrixFixture(parent, id)
	if err != nil || f.Stage != "owned" {
		return zero, errInstanceRecovery
	}
	b := instanceRuntimes
	record, err := b.readRecord(id)
	if err != nil || !record.Runtime.MatrixFixture || !operationFingerprintPattern.MatchString(record.Runtime.ImageDigest) ||
		len(record.Firewall) != 0 || f.checkRecord(record) != nil || b.verifyLive(record) != nil || f.checkData(ctx, record) != nil {
		return zero, errInstanceRecovery
	}
	hash, err := switchMatrixConfigHash(id)
	if err != nil || hash != f.ConfigSHA256 {
		return zero, errInstanceRecovery
	}
	return record, nil
}

func (s *server) withSwitchMatrixRuntime(ctx context.Context, parent, id, requiredState string,
	verify func(context.Context, instanceRuntimeRecord) error) (err error) {
	o, b := s.operations, instanceRuntimes
	lock, err := acquirePluginMaintenanceLock()
	if err != nil {
		return wrapSwitchMatrixFixtureError(matrixFixtureOwnership, err)
	}
	defer func() { err = errors.Join(err, lock.Close()) }()
	o.mu.Lock()
	_, fixturePhase := o.switchMatrixParent(parent, "fixtures", false)
	_, switchPhase := o.switchMatrixParent(parent, "switch", false)
	childActive := o.activeSwitchMatrixChild(parent)
	o.mu.Unlock()
	if fixturePhase != nil && switchPhase != nil || childActive {
		return wrapSwitchMatrixFixtureError(matrixFixtureOwnership, errInstanceRecovery)
	}
	record, err := o.switchMatrixOwnedRuntime(ctx, parent, id)
	if err != nil {
		return wrapSwitchMatrixFixtureError(matrixFixtureOwnership, err)
	}
	state, err := b.observeRaw(ctx, id)
	if err != nil || state.State != requiredState {
		return errors.Join(errSwitchMatrixStartup, errInstanceRecovery, err)
	}
	if requiredState == "stopped" {
		if err = lock.start(ctx, id); err != nil {
			return errors.Join(errSwitchMatrixStartup, err)
		}
		if _, err = o.switchMatrixOwnedRuntime(ctx, parent, id); err != nil {
			return wrapSwitchMatrixFixtureError(matrixFixtureOwnership, err)
		}
	}
	return verify(ctx, record)
}

func (s *server) stopSwitchMatrixFixtureForRetirement(ctx context.Context, parent, id string) error {
	lock, err := acquirePluginMaintenanceLock()
	if err != nil {
		return err
	}
	defer lock.Close()
	f, err := readSwitchMatrixFixture(parent, id)
	// The retirement contract interprets non-owned receipt stages and checks
	// terminal deletion, absent resources and partial creation evidence.
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if f.Stage == "reserved" {
		return s.recoverSwitchMatrixFixturePublication(ctx, f)
	}
	if f.Stage != "owned" {
		return nil
	}
	b := instanceRuntimes
	r, err := s.operations.switchMatrixOwnedRuntime(ctx, parent, id)
	if err != nil {
		return err
	}
	state, err := b.observeRaw(ctx, id)
	if err != nil {
		return err
	}
	switch state.State {
	case "stopped":
		return nil
	case "running":
		return b.actionWithVerifiedRecordWarnings(ctx, r, "stop", false)
	case "starting":
		// A timed-out systemctl start may have left a queued startup. There
		// is no Switch worker active when retirement is entered.
		if err = pluginService(ctx, id, "stop"); err != nil {
			return err
		}
		return b.deleteStopped(ctx, id, false)
	case "failed":
		return b.normalizeStoppedFailure(ctx, state)
	default:
		return errInstanceRecovery
	}
}

// A source can be interrupted after its initial configuration is durable but
// before activation commits. The reservation's exact candidate binds the
// existing definition for the normal activation rollback and publication.
func (s *server) recoverSwitchMatrixFixturePublication(ctx context.Context, f switchMatrixFixture) error {
	o, b := s.operations, instanceRuntimes
	o.mu.Lock()
	j, err := o.switchMatrixParent(f.Parent, "retirement", true)
	o.mu.Unlock()
	if err != nil || !slices.Contains(j.MatrixFixtures, f.ID) {
		return errInstanceRecovery
	}
	record, err := b.readRecord(f.ID)
	if errors.Is(err, os.ErrNotExist) && f.Candidate != nil {
		snapshot, _, _, err := b.snapshot(ctx)
		if err != nil {
			return err
		}
		return b.configureRegisteredMatrixFixture(ctx, f, snapshot)
	}
	if errors.Is(err, os.ErrNotExist) || errors.Is(err, errInstanceNotFound) {
		return nil // The retirement receipt has stricter absence checks.
	}
	matches := func(r instanceRuntimeRecord) bool {
		return r.Runtime.MatrixFixture && f.checkRecord(r) == nil && len(r.Firewall) == 0 &&
			instanceHash(mustSwitchJSON(r.Runtime)) == f.CandidateSHA256
	}
	if err != nil || !matches(record) {
		return errInstanceRecovery
	}
	pending, err := b.pending(f.ID)
	if err != nil {
		return err
	}
	if pending {
		dir, _, _ := b.paths(f.ID)
		raw, readErr := b.readPrivate(filepath.Join(dir, "pending.json"))
		var evidence instanceActivationEvidence
		if readErr != nil || instanceDecode(raw, &evidence) != nil || !matches(evidence.Previous) ||
			!matches(evidence.Target) || len(evidence.Changes) != 0 || b.recover(ctx, f.ID) != nil {
			return errInstanceRecovery
		}
		record, err = b.readRecord(f.ID)
		if err != nil || !matches(record) {
			return errInstanceRecovery
		}
	}
	if record.EnvironmentSHA256 != "" {
		return nil
	}
	snapshot, _, _, err := b.snapshot(ctx)
	if err != nil {
		return err
	}
	records, err := b.records()
	if err != nil {
		return err
	}
	// apply uses the record's pinned placement, does not start the unit, and
	// preserves the existing journaled provisioning safeguards.
	return b.apply(ctx, record, records, snapshot)
}

func verifySwitchMatrixRuntime(ctx context.Context, record instanceRuntimeRecord, software string, release *switchMatrixRuntime, crossplay bool) error {
	c := record.Runtime
	if release == nil || c.Software != software || c.Version != release.Game || c.VersionPolicy != "pinned" ||
		c.LoaderVersion != release.Loader || c.ImageTag != release.Image || c.ImageDigest != release.ImageDigest ||
		c.PufferfishBuild != release.PufferfishBuild || c.MatrixLeafBuild != release.Build ||
		c.BedrockEnabled != crossplay || c.StartOnBoot || !c.MatrixFixture {
		return &pluginRuntimeVerificationError{switchStatus: switchVerificationConfiguration}
	}
	if err := verifyPluginRuntimeMode(ctx, c.ID, c.Version, "--switch"); err != nil {
		return err
	}
	if err := verifySwitchMatrixPorts(ctx, c); err != nil {
		return wrapSwitchMatrixFixtureError(matrixFixturePorts, err)
	}
	if software == "leaf" {
		if err := verifySwitchMatrixLeafArtifact(record.Storage.DataPath, release); err != nil {
			return &pluginRuntimeVerificationError{switchStatus: switchVerificationSoftware}
		}
	}
	return nil
}

func verifySwitchMatrixPorts(ctx context.Context, c minecraftInstanceRuntimeConfig) error {
	raw, err := instanceRuntimes.command(ctx, "podman", "port", "minecraft-"+c.ID)
	if err != nil || len(raw) > 512 {
		return errInstanceRecovery
	}
	want := []string{fmt.Sprintf("25565/tcp -> 127.0.0.1:%d", c.JavaPort)}
	if c.BedrockEnabled {
		for _, protocol := range []string{"tcp", "udp"} {
			want = append(want, fmt.Sprintf("19132/%s -> 127.0.0.1:%d", protocol, c.BedrockPort))
		}
	}
	got := strings.Split(strings.TrimSpace(string(raw)), "\n")
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		return errInstanceRecovery
	}
	return nil
}

func verifySwitchMatrixLeafArtifact(path string, release *switchMatrixRuntime) error {
	root, err := os.OpenRoot(path)
	if err != nil {
		return err
	}
	defer root.Close()
	name := "leaf-" + release.Game + "-" + release.Build + ".jar"
	info, err := root.Lstat(name)
	if err != nil || !info.Mode().IsRegular() || info.Size() > 256<<20 {
		return errInstanceRecovery
	}
	file, err := root.Open(name)
	if err != nil {
		return err
	}
	hash := sha256.New()
	n, copyErr := io.Copy(hash, io.LimitReader(file, (256<<20)+1))
	if errors.Join(copyErr, file.Close()) != nil || n != info.Size() ||
		"sha256:"+hex.EncodeToString(hash.Sum(nil)) != release.Digest {
		return errInstanceRecovery
	}
	return nil
}

type switchMatrixMarker struct{ Dimension, Position, Block string }

var switchMatrixMarkers = []switchMatrixMarker{
	{"overworld", "8 90 8", "diamond_block"},
	{"the_nether", "9 90 9", "gold_block"},
	{"the_end", "10 90 10", "emerald_block"},
}

func verifySwitchMatrixMarkers(ctx context.Context, rcon func(context.Context, string) ([]byte, error), seed bool) error {
	for _, marker := range switchMatrixMarkers {
		prefix := "execute in minecraft:" + marker.Dimension + " "
		x, _, _ := strings.Cut(marker.Position, " ")
		_, z, _ := strings.Cut(strings.TrimPrefix(marker.Position, x+" "), " ")
		if _, err := rcon(ctx, prefix+"run forceload add "+x+" "+z); err != nil {
			return errSwitchMatrixWorldMarkers
		}
		found := false
		for attempt := 0; attempt < 12; attempt++ {
			if seed {
				_, _ = rcon(ctx, prefix+"run setblock "+marker.Position+" minecraft:"+marker.Block+" replace")
			}
			out, err := rcon(ctx, prefix+"if block "+marker.Position+" minecraft:"+marker.Block)
			if err == nil && normalizeInstanceJavaPlayerList(out) == "Test passed" {
				found = true
				break
			}
			if instancePause(ctx, time.Second) != nil {
				break
			}
		}
		if !found {
			return errSwitchMatrixWorldMarkers
		}
	}
	if seed {
		out, err := rcon(ctx, "save-all flush")
		if err != nil || !strings.Contains(normalizeInstanceJavaPlayerList(out), "Saved the game") {
			return errSwitchMatrixWorldMarkers
		}
		return verifySwitchMatrixMarkers(ctx, rcon, false)
	}
	return nil
}

func switchMatrixCheckWorldBudget(ctx context.Context, path string) error {
	if err := checkSwitchMatrixFreeSpace(path); err != nil {
		return err
	}
	var size int64
	return filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
		if err != nil || ctx.Err() != nil {
			return errors.Join(err, ctx.Err())
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			size += info.Size()
		}
		if size > switchMatrixWorldBudget {
			return errInstanceBusy
		}
		return nil
	})
}

func switchMatrixRCON(id string) func(context.Context, string) ([]byte, error) {
	return func(ctx context.Context, command string) ([]byte, error) {
		return instanceRuntimes.rcon(ctx, id, command)
	}
}

func (s *server) executeSwitchMatrixCase(parent string, c switchMatrixCase) (result switchMatrixCaseResult, unresolved error) {
	ctx, cancel := context.WithTimeout(context.Background(), 85*time.Minute)
	defer cancel()
	o, b := s.operations, instanceRuntimes
	j, err := o.get(parent)
	if err != nil {
		return switchMatrixCaseResult{Outcome: matrixFailed, Stage: "ownership", Reason: "parent_unavailable"}, err
	}
	result.Outcome = matrixFailed
	fail := func(stage, reason string) {
		result.Stage, result.Reason = stage, reason
	}
	defer func() {
		cleanup, stop := context.WithTimeout(context.Background(), 35*time.Minute)
		defer stop()
		child, err := s.closeSwitchMatrixCase(cleanup, parent, len(j.MatrixFixtures))
		if child.OperationID != "" {
			result.ChildOperation = child.OperationID
			result.Recovery = "unresolved"
			if child.State == operationResolved {
				result.Recovery = "verified"
			}
		}
		if err != nil {
			result.Cleanup = "unresolved"
			result.Outcome = matrixFailed
			result.ClosureReason = switchMatrixClosureDiagnostic(child, err)
			if result.Stage == "" {
				fail("cleanup", result.ClosureReason)
			}
			unresolved = err
			return
		}
		result.Cleanup = "verified"
	}()

	if _, err = o.transition(parent, operationRunning, "fixtures", "Preparing a new Switch matrix fixture."); err != nil {
		fail(string(matrixFixtureOwnership), "parent_reservation_unverified")
		return
	}
	if c.SourceRelease.Runtime == nil || c.TargetRelease.Runtime == nil ||
		c.SourceRelease.Runtime.ImageDigest == "" || c.TargetRelease.Runtime.ImageDigest == "" {
		fail("release", "runtime_image_unverified")
		return
	}
	config, snapshot, err := s.prepareSwitchMatrixSource(ctx, c)
	if err != nil {
		result.Stage, result.Reason = switchMatrixFixtureDiagnostic(err)
		return
	}
	id, err := o.createSwitchMatrixFixture(ctx, parent, config,
		fmt.Sprintf("Switch test %s %04d", parent[:8], len(j.MatrixFixtures)+1), snapshot)
	if err != nil {
		result.Stage, result.Reason = switchMatrixFixtureDiagnostic(err)
		return
	}
	var original instanceRuntimeRecord
	sourceProbe, done := context.WithTimeout(ctx, 17*time.Minute)
	err = s.withSwitchMatrixRuntime(sourceProbe, parent, id, "stopped", func(ctx context.Context, record instanceRuntimeRecord) error {
		original = record
		if err := verifySwitchMatrixRuntime(ctx, record, c.Source, c.SourceRelease.Runtime, c.Crossplay); err != nil {
			return err
		}
		return verifySwitchMatrixMarkers(ctx, switchMatrixRCON(id), true)
	})
	done()
	if errors.Is(err, errSwitchMatrixWorldMarkers) {
		result.World = "source_markers_failed"
		fail("source_world", "world_markers_unverified")
		return
	}
	if err != nil {
		result.Runtime = "source_failed"
		result.Stage, result.Reason = switchMatrixStartupDiagnostic("source", err)
		return
	}
	result.Runtime = "source_verified"
	if c.SourceState == "stopped" && s.stopSwitchMatrixFixtureForRetirement(ctx, parent, id) != nil {
		fail("source_state", "required_source_state_unverified")
		return
	}
	state, err := b.observeRaw(ctx, id)
	if err != nil || state.State != c.SourceState {
		fail("source_state", "required_source_state_unverified")
		return
	}
	if err = switchMatrixCheckWorldBudget(ctx, original.Storage.DataPath); err != nil {
		fail("resource_admission", "world_budget_unverified")
		return
	}
	if _, err = o.transition(parent, operationRunning, "switch", "Reviewing and applying production Switch."); err != nil {
		fail(string(matrixFixtureOwnership), "parent_reservation_unverified")
		return
	}
	applyCtx, done := context.WithTimeout(ctx, 45*time.Minute)
	child, release, err := s.switchMatrixApply(applyCtx, parent, id, c.Target, c.Version, c.Action, c)
	done()
	result.ChildOperation, result.ReleaseOutcome = child.OperationID, release
	switch {
	case errors.Is(err, errSwitchMatrixPolicy):
		result.Outcome = matrixBlocked
		fail("switch_review", "production_switch_policy")
		return
	case release == switchMatrixReleaseUnsupported:
		result.Outcome = matrixUnsupported
		fail("switch_review", "exact_release_unsupported")
		return
	case release == switchMatrixReleaseUnverified:
		fail("switch_review", "exact_release_unverified")
		return
	case child.OperationID == "":
		fail("switch_review", "production_review_unverified")
		return
	case child.State != operationSucceeded || err != nil:
		if child.State == operationResolved {
			result.Recovery = "verified"
		}
		fail("switch_apply", "production_switch_failed")
		var failure *switchMatrixChildFailure
		if errors.As(err, &failure) && len(failure.Stage) <= 48 && operationStagePattern.MatchString(failure.Stage) {
			result.Stage = "switch_" + failure.Stage
			if failure.RuntimeDiagnostic != "" {
				result.Reason = failure.RuntimeDiagnostic
			}
		}
		return
	}

	target, targetState := id, c.SourceState
	if c.Action == "clone" {
		e, err := readSwitchEvidence(child)
		if err != nil || e.Stage != "complete" || e.Source != id || e.CloneID == id || e.CloneID == "" {
			fail("ownership", "clone_identity_unverified")
			return
		}
		target, targetState = e.CloneID, "stopped"
	}
	targetProbe, done := context.WithTimeout(ctx, 17*time.Minute)
	err = s.withSwitchMatrixRuntime(targetProbe, parent, target, targetState, func(ctx context.Context, record instanceRuntimeRecord) error {
		if err := verifySwitchMatrixRuntime(ctx, record, c.Target, c.TargetRelease.Runtime, c.TargetCrossplay); err != nil {
			return err
		}
		result.Runtime = "verified"
		if err := switchMatrixCheckWorldBudget(ctx, record.Storage.DataPath); err != nil {
			return wrapSwitchMatrixFixtureError("resource_admission", err)
		}
		return verifySwitchMatrixMarkers(ctx, switchMatrixRCON(target), false)
	})
	done()
	if errors.Is(err, errSwitchMatrixWorldMarkers) {
		result.World = "target_markers_failed"
		fail("target_world", "world_markers_unverified")
		return
	}
	if err != nil {
		result.Runtime = "target_failed"
		result.Stage, result.Reason = switchMatrixStartupDiagnostic("target", err)
		return
	}
	result.World = "verified_overworld_nether_end"
	if targetState == "stopped" && s.stopSwitchMatrixFixtureForRetirement(ctx, parent, target) != nil {
		fail("target_state", "required_target_state_unverified")
		return
	}
	state, err = b.observeRaw(ctx, target)
	if err != nil || state.State != targetState {
		fail("target_state", "required_target_state_unverified")
		return
	}
	result.FinalState = "verified"

	if c.Action == "clone" {
		result.Original = "failed"
		originalProbe, done := context.WithTimeout(ctx, 17*time.Minute)
		err = s.withSwitchMatrixRuntime(originalProbe, parent, id, c.SourceState, func(ctx context.Context, record instanceRuntimeRecord) error {
			if !reflect.DeepEqual(original, record) {
				return &pluginRuntimeVerificationError{switchStatus: switchVerificationConfiguration}
			}
			if err := verifySwitchMatrixRuntime(ctx, record, c.Source, c.SourceRelease.Runtime, c.Crossplay); err != nil {
				return err
			}
			return verifySwitchMatrixMarkers(ctx, switchMatrixRCON(id), false)
		})
		done()
		if err == nil && c.SourceState == "stopped" {
			err = s.stopSwitchMatrixFixtureForRetirement(ctx, parent, id)
		}
		state, stateErr := b.observeRaw(ctx, id)
		if err != nil || stateErr != nil || state.State != c.SourceState {
			fail("original_source", "original_configuration_world_or_state_unverified")
			if diagnostic := switchMatrixRuntimeDiagnostic(err); diagnostic != "" {
				result.Reason = diagnostic
			}
			return
		}
		result.Original = "verified"
	} else {
		result.Original = "not_applicable"
	}
	result.Outcome = matrixPassed
	return
}

func switchMatrixFixtureDiagnostic(err error) (stage, reason string) {
	var failure *switchMatrixFixtureError
	if errors.As(err, &failure) {
		switch failure.stage {
		case matrixFixtureConfiguration:
			return string(failure.stage), "fixture_runtime_configuration_invalid"
		case matrixFixtureCredential:
			if errors.Is(err, errInstanceRuntimeCredential) {
				return string(failure.stage), "fixture_rcon_credential_invalid"
			}
			return string(failure.stage), "fixture_rcon_generation_failed"
		case matrixFixtureStorage:
			return string(failure.stage), "fixture_storage_placement_unverified"
		case matrixFixtureSoftware:
			return string(failure.stage), "fixture_software_version_unverified"
		case matrixFixturePorts:
			return string(failure.stage), "fixture_port_reservation_failed"
		case matrixFixtureOwnership:
			return string(failure.stage), "fixture_ownership_reservation_failed"
		case matrixFixturePublication:
			return string(failure.stage), "fixture_runtime_publication_failed"
		case matrixFixtureBinding:
			return string(failure.stage), "fixture_ownership_binding_failed"
		case "resource_admission":
			return string(failure.stage), "fixture_capacity_unverified"
		}
	}
	return "fixture_creation", "fixture_creation_unclassified"
}

func switchMatrixRuntimeDiagnostic(err error) string {
	var failure *switchMatrixFixtureError
	if errors.As(err, &failure) && failure.stage == matrixFixturePorts {
		return "runtime_port_binding_unverified"
	}
	return switchRuntimeDiagnostic(err)
}

func switchMatrixStartupDiagnostic(role string, err error) (stage, reason string) {
	var failure *switchMatrixFixtureError
	if code := switchMatrixRuntimeDiagnostic(err); code != "" {
		return role + "_runtime", code
	}
	switch {
	case errors.As(err, &failure) && failure.stage == "resource_admission":
		return "resource_admission", "world_budget_unverified"
	case errors.As(err, &failure) && failure.stage == matrixFixtureOwnership:
		return role + "_ownership", "bound_fixture_ownership_unverified"
	case errors.Is(err, errSwitchMatrixStartup):
		return role + "_start", "runtime_start_unverified"
	default:
		return role + "_readiness", "runtime_readiness_unverified"
	}
}

func switchMatrixClosureDiagnostic(child operationJournal, err error) string {
	if err == nil {
		return ""
	}
	if child.OperationID != "" && child.State != operationResolved {
		return "switch_child_recovery_unverified"
	}
	return "fixture_closure_unverified"
}
