//go:build justvoxel_dev

package main

import (
	"bytes"
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
		if c.Disposition == "" &&
			(c.SourceRelease.Runtime == nil || c.SourceRelease.Runtime.ImageDigest == "" ||
				c.TargetRelease.Runtime == nil || c.TargetRelease.Runtime.ImageDigest == "") {
			c.Disposition, c.Reason = matrixFailed, "runtime_image_unverified"
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
	defer func() {
		if releaseErr := lock.Close(); releaseErr != nil {
			err = &switchMatrixFixtureError{stage: matrixFixtureOwnership, cause: errors.Join(err, releaseErr)}
		}
	}()
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
			if !lock.protected() {
				return wrapSwitchMatrixFixtureError(matrixFixtureOwnership, err)
			}
			return errors.Join(errSwitchMatrixStartup, err)
		}
		if _, err = o.switchMatrixOwnedRuntime(ctx, parent, id); err != nil {
			return wrapSwitchMatrixFixtureError(matrixFixtureOwnership, err)
		}
	}
	return verify(ctx, record)
}

func (s *server) stopSwitchMatrixFixtureForRetirement(ctx context.Context, parent, id string) (result error) {
	lock, err := acquirePluginMaintenanceLock()
	if err != nil {
		return wrapSwitchMatrixFixtureError(matrixFixtureOwnership, err)
	}
	defer func() {
		if err := lock.Close(); err != nil {
			result = &switchMatrixFixtureError{stage: matrixFixtureOwnership, cause: errors.Join(result, err)}
		}
	}()
	f, err := readSwitchMatrixFixture(parent, id)
	// The retirement contract interprets non-owned receipt stages and checks
	// terminal deletion, absent resources and partial creation evidence.
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return wrapSwitchMatrixFixtureError(matrixFixtureOwnership, err)
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
		if err = s.operations.restoreSwitchMatrixFirewallEncoding(ctx, f); err != nil {
			return wrapSwitchMatrixFixtureError(matrixFixtureOwnership, err)
		}
		r, err = s.operations.switchMatrixOwnedRuntime(ctx, parent, id)
		if err != nil {
			return wrapSwitchMatrixFixtureError(matrixFixtureOwnership, err)
		}
	}
	state, err := b.observeShutdown(ctx, id)
	if err != nil {
		if state.shutdownObservationTransient {
			return &switchMatrixRetryableClosure{reason: "fixture_shutdown_unverified", cause: err}
		}
		return err
	}
	switch state.State {
	case "stopped":
		return nil
	case "running":
		err = b.actionWithVerifiedRecordWarnings(ctx, r, "stop", false)
		if errors.Is(err, errInstanceRuntimeStorageIdentity) {
			return wrapSwitchMatrixFixtureError(matrixFixtureOwnership, err)
		}
	case "stopping":
		err = b.waitForGracefulShutdown(ctx, id)
	case "starting":
		// A timed-out systemctl start may have left a queued startup. There
		// is no Switch worker active when retirement is entered.
		if err = pluginService(ctx, id, "stop"); err != nil {
			return err
		}
		return b.deleteStopped(ctx, id, false)
	case "failed":
		err = b.normalizeStoppedFailure(ctx, state)
	default:
		return errInstanceRecovery
	}
	var runtimeError *instanceRuntimeSafeError
	if errors.As(err, &runtimeError) || errors.Is(err, context.DeadlineExceeded) {
		return &switchMatrixRetryableClosure{reason: "fixture_shutdown_unverified", cause: err}
	}
	return err
}

// Called under the runtime maintenance lock during retirement. The receipt
// stays authoritative even when a completed legacy Clone changed [] to null.
func (o *operationStore) restoreSwitchMatrixFirewallEncoding(ctx context.Context, f switchMatrixFixture) error {
	o.mu.Lock()
	parent, err := o.switchMatrixParent(f.Parent, "retirement", true)
	var child operationJournal
	for _, j := range o.operations {
		if j.MatrixParent == f.Parent && j.InstanceID == f.ID && j.OperationType == operationTypeMinecraftSwitch {
			if child.OperationID != "" {
				err = errInstanceRecovery
				break
			}
			child = j
		}
	}
	busy := o.activeSwitchMatrixChild(f.Parent)
	o.mu.Unlock()
	if err != nil || busy || parent.State != operationNeedsAttention || parent.PlanFingerprint != switchMatrixRunnerFingerprint() ||
		f.Stage != "owned" || !slices.Contains(parent.MatrixFixtures, f.ID) ||
		child.State != operationSucceeded || child.Stage != "complete" {
		return errInstanceRecovery
	}
	e, err := readSwitchEvidence(child)
	if err != nil || e.Action != "clone" || e.Stage != "complete" || e.CloneID == "" ||
		e.SourceStopped || e.SourceContentChanged || e.WorldChanged || !slices.Contains(parent.MatrixFixtures, e.CloneID) {
		return errInstanceRecovery
	}
	clone, err := readSwitchMatrixFixture(f.Parent, e.CloneID)
	if err != nil || clone.Stage != "retired" || clone.ConfigSHA256 == "" {
		return errInstanceRecovery
	}
	b := instanceRuntimes
	saved := *e.SourceRecord
	current, err := b.readRecord(f.ID)
	if err != nil || !current.Runtime.MatrixFixture || !operationFingerprintPattern.MatchString(current.Runtime.ImageDigest) ||
		f.checkRecord(current) != nil || len(current.Firewall) != 0 || len(saved.Firewall) != 0 ||
		(current.Firewall == nil) == (saved.Firewall == nil) {
		return errInstanceRecovery
	}
	savedBytes := append(mustSwitchJSON(saved), '\n')
	if instanceHash(savedBytes) != f.ConfigSHA256 {
		return errInstanceRecovery
	}
	dir, _, _ := b.paths(f.ID)
	path := filepath.Join(dir, "config.json")
	raw, err := b.readPrivate(path)
	// Require the actual on-disk bytes to be exactly the known serialization
	// with the alternate encoding, including every other field and newline.
	alternate := saved
	alternate.Firewall = current.Firewall
	if err != nil || !bytes.Equal(raw, append(mustSwitchJSON(alternate), '\n')) ||
		b.verifyLive(current) != nil || f.checkData(ctx, current) != nil {
		return errInstanceRecovery
	}
	if _, err = f.backupStorage(ctx, "matrix-placement"); err != nil {
		return err
	}
	return b.writeJSON(path, saved, true)
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
	if crossplay {
		probe, cancel := context.WithTimeout(ctx, 25*time.Second)
		defer cancel()
		if _, err := instanceRuntimes.command(probe, "podman", "exec", "minecraft-"+c.ID,
			"mc-monitor", "status-bedrock", "--host", "127.0.0.1", "--port", "19132"); err != nil {
			return &pluginRuntimeVerificationError{switchStatus: switchVerificationCrossplay}
		}
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

func (e *switchMatrixMarkerFailure) Error() string { return e.Reason }
func (e *switchMatrixMarkerFailure) Unwrap() error { return errSwitchMatrixWorldMarkers }

func switchMatrixMarkerCommand(ctx context.Context, rcon func(context.Context, string) ([]byte, error),
	dimension, stage, command string) (string, *switchMatrixMarkerFailure) {
	failure := &switchMatrixMarkerFailure{Stage: stage, Dimension: dimension}
	if err := ctx.Err(); err != nil {
		failure.Reason, failure.Outcome = "timeout", "context_done"
		return "", failure
	}
	out, err := rcon(ctx, command)
	switch {
	case errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil:
		failure.Reason, failure.Outcome = "timeout", "command_deadline"
	case errors.Is(err, errInstanceConsoleResponseTooLarge) || len(out) > 8192:
		failure.Reason, failure.Outcome = "command_response_unverified", "oversized_response"
	case err != nil:
		failure.Reason, failure.Outcome = "rcon_transport_failure", "transport_error"
	default:
		response := normalizeInstanceJavaPlayerList(out)
		if dimension != "" && strings.HasPrefix(response, "Unknown dimension ") {
			failure.Reason, failure.Outcome = "dimension_unavailable", "unknown_dimension"
			return "", failure
		}
		if response == "That position is not loaded" {
			failure.Reason, failure.Outcome = "chunk_loading_failed", "position_unloaded"
			return "", failure
		}
		return response, nil
	}
	return "", failure
}

func verifySwitchMatrixMarkers(ctx context.Context, rcon func(context.Context, string) ([]byte, error), seed bool) error {
	for _, marker := range switchMatrixMarkers {
		if err := verifySwitchMatrixMarker(ctx, rcon, marker, seed); err != nil {
			return err
		}
	}
	if seed {
		out, failure := switchMatrixMarkerCommand(ctx, rcon, "", "save_flush", "save-all flush")
		if failure != nil {
			return failure
		}
		const saving = "Saving the game (this may take a moment!)"
		switch strings.TrimSpace(strings.TrimPrefix(out, saving)) {
		case "Saved the game":
		case "Unable to save the game (is there enough disk space?)":
			return &switchMatrixMarkerFailure{Stage: "save_flush", Reason: "save_flush_failed", Outcome: "save_rejected"}
		default:
			return &switchMatrixMarkerFailure{Stage: "save_flush", Reason: "save_flush_unverified", Outcome: "save_not_confirmed"}
		}
		for _, marker := range switchMatrixMarkers {
			if err := verifySwitchMatrixMarker(ctx, rcon, marker, false); err != nil {
				var diagnostic *switchMatrixMarkerFailure
				if errors.As(err, &diagnostic) {
					diagnostic.Stage = "post_save_" + diagnostic.Stage
				}
				return err
			}
		}
	}
	return nil
}

func verifySwitchMatrixMarker(ctx context.Context, rcon func(context.Context, string) ([]byte, error), marker switchMatrixMarker, seed bool) error {
	world := "minecraft:" + marker.Dimension
	prefix := "execute in " + world + " "
	coordinates := strings.Fields(marker.Position)
	chunk := coordinates[0] + " " + coordinates[2]
	command := func(stage, command string) (string, *switchMatrixMarkerFailure) {
		return switchMatrixMarkerCommand(ctx, rcon, marker.Dimension, stage, prefix+command)
	}
	out, failure := command("forceload_add", "run forceload add "+chunk)
	if failure != nil {
		return failure
	}
	switch out {
	case "Marked chunk [0, 0] in " + world + " to be force loaded":
	case "No chunks were marked for force loading":
		out, failure = command("forceload_query", "run forceload query "+chunk)
		if failure != nil {
			return failure
		}
		if out != "Chunk at [0, 0] in "+world+" is marked for force loading" {
			return &switchMatrixMarkerFailure{Stage: "forceload_query", Dimension: marker.Dimension, Reason: "chunk_loading_failed", Outcome: "ticket_unverified"}
		}
	default:
		return &switchMatrixMarkerFailure{Stage: "forceload_add", Dimension: marker.Dimension, Reason: "chunk_loading_failed", Outcome: "ticket_unverified"}
	}

	load, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	for {
		out, failure = switchMatrixMarkerCommand(load, rcon, marker.Dimension, "chunk_ready", prefix+"if loaded "+marker.Position)
		if failure != nil {
			return failure
		}
		if out == "Test passed" {
			break
		}
		if out != "Test failed" {
			return &switchMatrixMarkerFailure{Stage: "chunk_ready", Dimension: marker.Dimension, Reason: "chunk_loading_failed", Outcome: "unexpected_response"}
		}
		if instancePause(load, time.Second) != nil {
			return &switchMatrixMarkerFailure{Stage: "chunk_ready", Dimension: marker.Dimension, Reason: "timeout", Outcome: "chunk_not_loaded"}
		}
	}
	if seed {
		out, failure = command("setblock", "run setblock "+marker.Position+" minecraft:"+marker.Block+" replace")
		if failure != nil {
			return failure
		}
		switch out {
		case "Changed the block at " + strings.Join(coordinates, ", "):
		case "Could not set the block":
			return &switchMatrixMarkerFailure{Stage: "setblock", Dimension: marker.Dimension, Reason: "setblock_rejected", Outcome: "block_change_unconfirmed"}
		default:
			return &switchMatrixMarkerFailure{Stage: "setblock", Dimension: marker.Dimension, Reason: "setblock_unverified", Outcome: "unexpected_response"}
		}
	}
	out, failure = command("compare", "if block "+marker.Position+" minecraft:"+marker.Block)
	if failure != nil {
		return failure
	}
	switch out {
	case "Test passed":
		return nil
	case "Test failed":
		return &switchMatrixMarkerFailure{Stage: "compare", Dimension: marker.Dimension, Reason: "marker_comparison_failed", Outcome: "block_mismatch"}
	default:
		return &switchMatrixMarkerFailure{Stage: "compare", Dimension: marker.Dimension, Reason: "marker_comparison_failed", Outcome: "unexpected_response"}
	}
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
		// Preserve the existing RCON limit and retain a deadline classification
		// when the transport wrapper returns its public-safe error.
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		out, err := instanceRuntimes.rcon(ctx, id, command)
		return out, errors.Join(err, ctx.Err())
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
	failOwnership := func(role string, err error) bool {
		var failure *switchMatrixFixtureError
		if !errors.As(err, &failure) || failure.stage != matrixFixtureOwnership {
			return false
		}
		fail(role+"_ownership", "bound_fixture_ownership_unverified")
		return true
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
			if unresolved == nil {
				unresolved = err
			} else {
				unresolved = errors.Join(unresolved, err)
			}
			return
		}
		result.Cleanup = "verified"
	}()

	if _, err = o.transition(parent, operationRunning, "fixtures", "Preparing a new Switch matrix fixture."); err != nil {
		fail(string(matrixFixtureOwnership), "parent_reservation_unverified")
		unresolved = err
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
	var state instanceRuntimeStatus
	var initialization string
	if c.Profile == switchMatrixInitialSource {
		original, err = o.switchMatrixOwnedRuntime(ctx, parent, id)
		if err != nil {
			fail("source_ownership", "bound_fixture_ownership_unverified")
			return
		}
		initialization, err = b.switchInitializationEvidence(ctx, original)
		if err != nil {
			fail("source_initialization", "uninitialized_source_unverified")
			return
		}
		result.Runtime = "source_uninitialized_verified"
	} else {
		sourceProbe, done := context.WithTimeout(ctx, 17*time.Minute)
		err = s.withSwitchMatrixRuntime(sourceProbe, parent, id, "stopped", func(ctx context.Context, record instanceRuntimeRecord) error {
			original = record
			if err := verifySwitchMatrixRuntime(ctx, record, c.Source, c.SourceRelease.Runtime, c.Crossplay); err != nil {
				return err
			}
			return verifySwitchMatrixMarkers(ctx, switchMatrixRCON(id), true)
		})
		done()
		if errors.As(err, &result.MarkerFailure) {
			result.World = "source_markers_failed"
			if !failOwnership("source", err) {
				fail("source_world", result.MarkerFailure.Reason)
			}
			return
		}
		if err != nil {
			result.Runtime = "source_failed"
			result.Stage, result.Reason = switchMatrixStartupDiagnostic("source", err)
			return
		}
		result.Runtime = "source_verified"
		if c.SourceState == "stopped" {
			if err = s.stopSwitchMatrixFixtureForRetirement(ctx, parent, id); err != nil {
				if !failOwnership("source", err) {
					fail("source_state", "required_source_state_unverified")
				}
				return
			}
			state, err = b.observeShutdown(ctx, id)
		} else {
			state, err = b.observeRaw(ctx, id)
		}
		if err != nil || state.State != c.SourceState {
			fail("source_state", "required_source_state_unverified")
			return
		}
	}
	if c.Profile == switchMatrixManagedContent {
		if err = s.seedSwitchMatrixManagedContent(ctx, parent, id); err != nil {
			if !failOwnership("source", err) {
				fail("source_content", "managed_fixture_content_unverified")
			}
			if errors.Is(err, errSwitchEvidencePersistence) {
				result.Reason = "fixture_manifest_persistence_failed"
				unresolved = err
			}
			return
		}
	}
	if err = switchMatrixCheckWorldBudget(ctx, original.Storage.DataPath); err != nil {
		fail("resource_admission", "world_budget_unverified")
		return
	}
	var originalContent string
	if c.Action == "clone" {
		_, originalContent, err = switchContentEvidence(ctx, original, "", c.Version)
		if err != nil {
			fail("source_content", "original_executables_unverified")
			return
		}
	}
	if _, err = o.transition(parent, operationRunning, "switch", "Reviewing and applying production Switch."); err != nil {
		fail(string(matrixFixtureOwnership), "parent_reservation_unverified")
		unresolved = err
		return
	}
	applyCtx, done := context.WithTimeout(ctx, 45*time.Minute)
	if c.Profile != "" {
		child, release, err := s.switchMatrixApply(applyCtx, parent, id, c.Target, c.Version, "convert", c)
		if !errors.Is(err, errSwitchMatrixPolicy) || child.OperationID != "" {
			done()
			result.ChildOperation, result.ReleaseOutcome, result.ReleaseVerification = child.OperationID, release.Outcome, release.Verification
			if !failOwnership("switch", err) {
				fail("switch_review", "fixture_convert_refusal_unverified")
			}
			return
		}
	}
	child, release, err := s.switchMatrixApply(applyCtx, parent, id, c.Target, c.Version, c.Action, c)
	done()
	result.ChildOperation, result.ReleaseOutcome, result.ReleaseVerification = child.OperationID, release.Outcome, release.Verification
	if failOwnership("switch", err) {
		if errors.Is(err, errSwitchEvidencePersistence) {
			unresolved = err
			result.Reason = "switch_journal_persistence_failed"
		}
		return
	}
	switch {
	case errors.Is(err, errSwitchMatrixProfile):
		fail("switch_review", "fixture_profile_expectations_unverified")
		return
	case errors.Is(err, errSwitchMatrixPolicy):
		result.Outcome = matrixBlocked
		fail("switch_review", "production_switch_policy")
		return
	case release.Outcome == switchMatrixReleaseUnsupported:
		result.Outcome = matrixUnsupported
		fail("switch_review", "exact_release_unsupported")
		return
	case release.Outcome == switchMatrixReleaseUnavailable:
		result.Outcome = matrixBlocked
		fail("switch_review", "exact_release_unavailable")
		return
	case release.Outcome == switchMatrixReleaseUnverified:
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
			if failure.Stage == "journal_finalization" || failure.RuntimeDiagnostic == "switch_journal_persistence_failed" {
				unresolved = err
			}
		}
		return
	}

	target, targetState := id, c.SourceState
	cloneContent := ""
	if c.Action == "clone" {
		e, err := readSwitchEvidence(child)
		if err != nil || e.Stage != "complete" || e.Source != id || e.CloneID == id || e.CloneID == "" || !instanceSHA256Pattern.MatchString(e.CloneContentFingerprint) {
			fail("ownership", "clone_identity_unverified")
			return
		}
		target, targetState, cloneContent = e.CloneID, "stopped", e.CloneContentFingerprint
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
		if c.Profile == switchMatrixInitialSource && verifySwitchInitializedWorld(record.Storage.DataPath) != nil {
			return &switchMatrixMarkerFailure{Stage: "target_initialization", Reason: "initialized_world_unverified", Outcome: "world_evidence_missing"}
		}
		return verifySwitchMatrixMarkers(ctx, switchMatrixRCON(target), c.Profile == switchMatrixInitialSource)
	})
	done()
	if errors.As(err, &result.MarkerFailure) {
		result.World = "target_markers_failed"
		if !failOwnership("target", err) {
			fail("target_world", result.MarkerFailure.Reason)
		}
		return
	}
	if err != nil {
		result.Runtime = "target_failed"
		result.Stage, result.Reason = switchMatrixStartupDiagnostic("target", err)
		return
	}
	result.World = "verified_overworld_nether_end"
	if targetState == "stopped" {
		if err = s.stopSwitchMatrixFixtureForRetirement(ctx, parent, target); err != nil {
			if !failOwnership("target", err) {
				fail("target_state", "required_target_state_unverified")
			}
			return
		}
		state, err = b.observeShutdown(ctx, target)
	} else {
		state, err = b.observeRaw(ctx, target)
	}
	if err != nil || state.State != targetState {
		fail("target_state", "required_target_state_unverified")
		return
	}
	if c.Action == "clone" {
		record, readErr := b.readRecord(target)
		if readErr != nil {
			fail("target_content", "destination_executables_unverified")
			return
		}
		inventory, err := currentSwitchCloneContentFingerprint(record.Storage.DataPath)
		if err != nil || inventory != cloneContent {
			fail("target_content", "destination_executables_changed")
			return
		}
	}
	result.FinalState = "verified"

	if c.Action == "clone" {
		result.Original = "failed"
		if c.Profile == switchMatrixInitialSource {
			record, err := o.switchMatrixOwnedRuntime(ctx, parent, id)
			if err != nil {
				fail("source_ownership", "bound_fixture_ownership_unverified")
				return
			}
			if !reflect.DeepEqual(original, record) {
				fail("original_source", "original_configuration_unverified")
				return
			}
			if err = b.requireSwitchInitialization(ctx, original, minecraftSwitchPlan{InitializationFingerprint: initialization}); err != nil {
				fail("original_source", "original_uninitialized_state_changed")
				return
			}
			result.Original = "verified_uninitialized"
		} else {
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
			if err == nil {
				if c.SourceState == "stopped" {
					state, err = b.observeShutdown(ctx, id)
				} else {
					state, err = b.observeRaw(ctx, id)
				}
			}
			if err != nil || state.State != c.SourceState {
				if failOwnership("original", err) {
					errors.As(err, &result.MarkerFailure)
					return
				}
				fail("original_source", "original_configuration_world_or_state_unverified")
				if errors.As(err, &result.MarkerFailure) {
					result.Reason = result.MarkerFailure.Reason
				} else if diagnostic := switchMatrixRuntimeDiagnostic(err); diagnostic != "" {
					result.Reason = diagnostic
				}
				return
			}
			result.Original = "verified"
		}
		_, sourceContent, err := switchContentEvidence(ctx, original, "", c.Version)
		if err != nil || sourceContent != originalContent {
			result.Original = "failed"
			fail("original_source", "original_executables_changed")
			return
		}
		if c.Profile == switchMatrixManagedContent {
			result.Content = "verified_retention_exclusion_dependency_and_convert_refusal"
		}
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
	if errors.As(err, &failure) && failure.stage == matrixFixtureOwnership {
		return role + "_ownership", "bound_fixture_ownership_unverified"
	}
	if code := switchMatrixRuntimeDiagnostic(err); code != "" {
		return role + "_runtime", code
	}
	switch {
	case errors.As(err, &failure) && failure.stage == "resource_admission":
		return "resource_admission", "world_budget_unverified"
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
	var fixture *switchMatrixFixtureError
	if errors.As(err, &fixture) && fixture.stage == matrixFixtureOwnership {
		return "fixture_closure_ownership_unverified"
	}
	var retryable *switchMatrixRetryableClosure
	if errors.As(err, &retryable) {
		return retryable.reason
	}
	return "fixture_closure_unverified"
}
