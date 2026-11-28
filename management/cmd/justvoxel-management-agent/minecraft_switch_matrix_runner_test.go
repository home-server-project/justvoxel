//go:build justvoxel_dev

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
)

func matrixRunnerTestRelease(_ context.Context, software, version string) switchMatrixRelease {
	r := &switchMatrixRuntime{Game: version, Image: "java21", ImageDigest: "sha256:" + strings.Repeat("d", 64)}
	if java25GameReleasePattern.MatchString(version) {
		r.Image = "java25"
	}
	if capabilities, _ := javaServerSoftware(software); capabilities.ModLoader {
		r.Loader = "21.1.2"
	}
	switch software {
	case "leaf":
		r.Build, r.Digest, r.Channel = "123", "sha256:"+strings.Repeat("c", 64), "PRODUCTION"
	case "pufferfish":
		r.PufferfishBuild = 123
	}
	return switchMatrixRelease{Outcome: switchMatrixReleaseAvailable, Runtime: r}
}

func TestSwitchMatrixManifest(t *testing.T) {
	versions := switchMatrixVersions([]string{"26.3", "26.3", "26.2", "1.21.11", "1.21.8", "1.21", "26.4-rc1", "1.16.5"})
	if !slices.Equal(versions[switchMatrixModern], []string{"26.3", "26.2"}) ||
		!slices.Equal(versions[switchMatrixLegacy], []string{"1.21.11", "1.21.8", "1.21"}) {
		t.Fatalf("unexpected discovery catalog: %v", versions)
	}
	ctx := context.Background()
	stack := func(_ context.Context, _ string, version string, _ *switchMatrixRuntime) string {
		if version == "26.3" {
			return "geyser_java_target_mismatch"
		}
		return ""
	}
	lookups := map[switchMatrixReleaseKey]int{}
	cases := buildSwitchMatrixCases(ctx, versions, func(ctx context.Context, name, version string) switchMatrixRelease {
		lookups[switchMatrixReleaseKey{name, version}]++
		return matrixRunnerTestRelease(ctx, name, version)
	}, stack)
	if !reflect.DeepEqual(cases, buildSwitchMatrixCases(ctx, versions, matrixRunnerTestRelease, stack)) {
		t.Fatal("manifest is not deterministic")
	}
	for k, n := range lookups {
		if n != 1 {
			t.Fatalf("publisher lookup repeated for %v", k)
		}
	}
	ids, directions := map[string]bool{}, map[[2]string][]switchMatrixCase{}
	for _, c := range cases {
		if ids[c.ID] || c.Source == c.Target {
			t.Fatalf("duplicate or same-software case: %+v", c)
		}
		ids[c.ID] = true
		k := [2]string{c.Source, c.Target}
		directions[k] = append(directions[k], c)
	}
	if len(directions) != 110 {
		t.Fatalf("missing ordered directions: %d", len(directions))
	}
	for k, rows := range directions {
		if k[0] == "bedrock" || k[1] == "bedrock" || k[0] == "spigot" || k[1] == "spigot" {
			if len(rows) != 1 || rows[0].Disposition != matrixUnsupported || rows[0].Reason != "structural_direction_unsupported" {
				t.Fatalf("bad structural accounting for %v: %+v", k, rows)
			}
			continue
		}
		for _, family := range []string{switchMatrixLegacy, switchMatrixModern} {
			for _, action := range []string{"clone", "convert"} {
				for _, enabled := range []bool{false, true} {
					if enabled && !javaServerCrossplay(k[0]) {
						continue
					}
					var states []string
					for _, c := range rows {
						if c.Family != family || c.Action != action || c.Crossplay != enabled {
							continue
						}
						states = append(states, c.SourceState)
						wantBlocked := action == "convert" && (switchWorldRisk(k[0], k[1]) ||
							enabled && (switchRejectsCrossplayPlatform(k[0], k[1], true) || switchNeedsVerifiedJavaOnlyMods(k[0], k[1], true)))
						if wantBlocked && c.Disposition != matrixBlocked || !wantBlocked && c.Disposition != "" {
							t.Fatalf("bad policy for %+v", c)
						}
						if !wantBlocked && enabled && family == switchMatrixModern && c.Version != "26.2" {
							t.Fatalf("did not select confirmed managed stack: %+v", c)
						}
					}
					if len(states) == 0 || !slices.Equal(states, []string{""}) && !slices.Equal(states, []string{"running", "stopped"}) {
						t.Fatalf("missing or duplicate source states for %v %s %s %t: %v", k, family, action, enabled, states)
					}
				}
			}
		}
	}
	t.Run("publisher_absence_and_transient_error", func(t *testing.T) {
		lookup := func(source, version string) switchMatrixRelease {
			if source == "paper" {
				return matrixRunnerTestRelease(ctx, source, version)
			}
			if version == "26.2" {
				return switchMatrixRelease{Outcome: switchMatrixReleaseUnverified}
			}
			return switchMatrixRelease{Outcome: switchMatrixReleaseUnsupported}
		}
		version, _, target, reason := selectSwitchMatrixPair([]string{"26.3", "26.2"}, "paper", "purpur", lookup, nil)
		if version != "26.2" || target.Outcome != switchMatrixReleaseUnverified || reason != "exact_release_unverified" {
			t.Fatalf("publisher failure became absence: %q %+v %s", version, target, reason)
		}
		_, _, target, reason = selectSwitchMatrixPair([]string{"26.3"}, "paper", "purpur", lookup, nil)
		if target.Outcome != switchMatrixReleaseUnsupported || reason != "exact_release_unsupported" {
			t.Fatalf("authoritative absence lost: %+v %s", target, reason)
		}
	})
	t.Run("crossplay_evidence_is_logged_per_role", func(t *testing.T) {
		rows := buildSwitchMatrixCases(ctx, map[string][]string{switchMatrixModern: {"26.3"}}, matrixRunnerTestRelease,
			func(_ context.Context, name, _ string, _ *switchMatrixRuntime) string {
				if name == "paper" {
					return "geyser_request_timeout"
				}
				return "floodgate_artifact_unverified"
			})
		for i, c := range rows {
			if c.Source == "paper" && c.Target == "purpur" && c.Family == switchMatrixModern && c.Crossplay && c.Action == "clone" {
				result := newSwitchMatrixResults(rows)[i]
				if result.Outcome != matrixFailed || result.Reason != "managed_crossplay_stack_unverified" ||
					c.SourceCrossplayCheck != "geyser_request_timeout" || c.TargetCrossplayCheck != "floodgate_artifact_unverified" {
					t.Fatalf("lost per-role managed stack evidence: %+v %+v", c, result)
				}
				return
			}
		}
		t.Fatal("expected managed cross-play row")
	})
	t.Run("publisher_protocol_rejects_untrusted_details", func(t *testing.T) {
		for _, scenario := range []struct {
			raw  string
			want switchMatrixReleaseOutcome
		}{
			{`{"outcome":"AVAILABLE","runtime":{"game":"26.3","loader":"","image":"java25"}}`, switchMatrixReleaseAvailable},
			{`{"outcome":"UNSUPPORTED"}`, switchMatrixReleaseUnsupported},
			{`{"outcome":"UNAVAILABLE"}`, switchMatrixReleaseUnavailable},
			{`{"outcome":"UNAVAILABLE","runtime":{"game":"26.3","loader":"","image":"java25"}}`, switchMatrixReleaseUnverified},
			{`{"outcome":"UNVERIFIED"}`, switchMatrixReleaseUnverified},
			{`{"outcome":"UNSUPPORTED","runtime":{"game":"26.3","loader":"","image":"java25"}}`, switchMatrixReleaseUnverified},
			{`{"outcome":"AVAILABLE","runtime":{"game":"26.3","loader":"","image":"java25","reason":"password=private-value"}}`, switchMatrixReleaseUnverified},
			{`{"outcome":"AVAILABLE","runtime":{"game":"26.2","loader":"","image":"java25"}}`, switchMatrixReleaseUnverified},
			{`{"outcome":"AVAILABLE","runtime":{"game":"26.3","loader":"","image":"java21"}}`, switchMatrixReleaseUnverified},
			{`{"outcome":"ERROR"}`, switchMatrixReleaseUnverified},
			{`{"outcome":"UNSUPPORTED","verification":"private-value"}`, switchMatrixReleaseUnverified},
		} {
			r := decodeSwitchMatrixRelease("paper", "26.3", []byte(scenario.raw))
			if r.Outcome != scenario.want || r.Outcome != switchMatrixReleaseAvailable && r.Runtime != nil {
				t.Fatalf("bad publisher classification: %+v", r)
			}
		}
	})
	t.Run("crossplay_helper_codes_are_allowlisted", func(t *testing.T) {
		for raw, want := range map[string]string{
			`{"compatible":true,"reason":""}`:                             "",
			`{"compatible":false,"reason":"viaversion_response_invalid"}`: "viaversion_response_invalid",
			`{"compatible":true,"reason":"unexpected credentials"}`:       "crossplay_response_invalid",
			`{"compatible":false,"reason":"unexpected credentials"}`:      "crossplay_response_invalid",
			`{}`: "crossplay_response_invalid",
		} {
			if reason := decodeSwitchMatrixCrossplayCheck([]byte(raw)); reason != want {
				t.Errorf("got %q, want %q", reason, want)
			}
		}
	})
}

func matrixRunnerPassed(c switchMatrixCase) switchMatrixCaseResult {
	original := "verified"
	if c.Action == "convert" {
		original = "not_applicable"
	}
	return switchMatrixCaseResult{Outcome: matrixPassed, Runtime: "verified", World: "verified_overworld_nether_end",
		Original: original, FinalState: "verified", Cleanup: "verified"}
}

func matrixRunnerTestManifest(t *testing.T) switchMatrixRun {
	t.Helper()
	catalog := switchMatrixVersions([]string{"26.2", "1.21.11"})
	cases := buildSwitchMatrixCases(context.Background(), catalog, matrixRunnerTestRelease,
		func(context.Context, string, string, *switchMatrixRuntime) string { return "" })
	return switchMatrixRun{Catalog: catalog, Cases: cases, Results: newSwitchMatrixResults(cases)}
}

func newMatrixRunnerTest(t *testing.T, n int) (*server, operationJournal, switchMatrixRun) {
	t.Helper()
	f := newInstanceActivationFixture(t)
	o := matrixTestStore(t, f.backend)
	j, err := o.beginSwitchMatrix(switchMatrixRunnerFingerprint())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = o.transition(j.OperationID, operationValidating, "manifest", "Discovering releases."); err != nil {
		t.Fatal(err)
	}
	r := switchMatrixRun{Schema: 1, ID: j.OperationID, State: "running"}
	manifest := matrixRunnerTestManifest(t)
	smoke, reason := selectSwitchMatrixSmoke(manifest)
	if smoke < 0 {
		t.Fatal("test manifest missing smoke case", reason)
	}
	r.Cases = append(r.Cases, manifest.Cases[smoke])
	r.Results = append(r.Results, switchMatrixCaseResult{Outcome: matrixPending})
	for i, c := range manifest.Cases {
		if len(r.Cases) == n {
			break
		}
		if i != smoke && c.Disposition == "" {
			r.Cases = append(r.Cases, c)
			r.Results = append(r.Results, switchMatrixCaseResult{Outcome: matrixPending})
		}
	}
	if len(r.Cases) != n {
		t.Fatal("insufficient test cases")
	}
	if err = o.saveSwitchMatrixRun(r); err != nil {
		t.Fatal(err)
	}
	if _, err = o.transition(j.OperationID, operationRunning, "fixtures", "Working on fixtures."); err != nil {
		t.Fatal(err)
	}
	return &server{operations: o}, j, r
}

func TestSwitchMatrixExecution(t *testing.T) {
	t.Run("real_worker_refuses_foreign_ids_and_insufficient_capacity", func(t *testing.T) {
		s, j, r := newMatrixRunnerTest(t, 1)
		o := s.operations
		if _, err := s.startSwitchMatrixRun(); err == nil {
			t.Fatal("duplicate runner admitted")
		}
		if _, err := o.switchMatrixOwnedRuntime(context.Background(), j.OperationID, "jv-foreign"); !errors.Is(err, errInstanceID) {
			t.Fatalf("foreign runtime result: %v", err)
		}
		old := instanceRuntimes.snapshot
		instanceRuntimes.snapshot = func(ctx context.Context) (minecraftInstanceRuntimeContext, uint32, uint32, error) {
			data, uid, gid, err := old(ctx)
			data.TotalMemoryMiB = 2048
			return data, uid, gid, err
		}
		defer func() { instanceRuntimes.snapshot = old }()
		result, unresolved := s.executeSwitchMatrixCase(j.OperationID, r.Cases[0])
		if unresolved != nil || result.Outcome != matrixFailed || result.Stage != "resource_admission" || result.Cleanup != "verified" {
			t.Fatalf("bad capacity refusal: %+v %v", result, unresolved)
		}
		current, err := o.get(j.OperationID)
		if err != nil || len(current.MatrixFixtures) != 0 {
			t.Fatalf("fixture reserved before capacity passed: %+v %v", current, err)
		}
		if _, err = o.transition(j.OperationID, operationRunning, "switch", "Reviewing a fixture."); err != nil {
			t.Fatal(err)
		}
		if _, _, err = s.switchMatrixApply(context.Background(), j.OperationID, "jv-a1b2c3", "vanilla", "1.21.11", "clone"); !errors.Is(err, errInstanceID) {
			t.Fatalf("production Switch accepted an ordinary instance: %v", err)
		}
	})
	t.Run("marker_probe_requires_all_dimensions_and_save", func(t *testing.T) {
		written, forced := map[string]bool{}, map[string]bool{}
		var saved bool
		rcon := func(ctx context.Context, command string) ([]byte, error) {
			for _, m := range switchMatrixMarkers {
				prefix := "execute in minecraft:" + m.Dimension + " "
				switch command {
				case prefix + "run forceload add 8 8", prefix + "run forceload add 9 9", prefix + "run forceload add 10 10":
					if forced[m.Dimension] {
						return []byte("No chunks were marked for force loading"), nil
					}
					forced[m.Dimension] = true
					return []byte("Marked chunk [0, 0] in minecraft:" + m.Dimension + " to be force loaded"), nil
				case prefix + "run forceload query 8 8", prefix + "run forceload query 9 9", prefix + "run forceload query 10 10":
					return []byte("Chunk at [0, 0] in minecraft:" + m.Dimension + " is marked for force loading"), nil
				case prefix + "if loaded " + m.Position:
					return []byte("Test passed"), nil
				case prefix + "run setblock " + m.Position + " minecraft:" + m.Block + " replace":
					written[m.Dimension] = true
					return []byte("Changed the block at " + strings.ReplaceAll(m.Position, " ", ", ")), nil
				case prefix + "if block " + m.Position + " minecraft:" + m.Block:
					if written[m.Dimension] {
						return []byte("Test passed"), nil
					}
					return []byte("Test failed"), nil
				}
			}
			if command == "save-all flush" {
				saved = true
				return []byte("Saving the game (this may take a moment!)Saved the game"), nil
			}
			t.Fatalf("unexpected RCON command: %q", command)
			return nil, nil
		}
		if err := verifySwitchMatrixMarkers(context.Background(), rcon, true); err != nil || !saved || len(written) != 3 {
			t.Fatalf("failed real marker command contract: %v", err)
		}
		delete(written, "the_end")
		var failure *switchMatrixMarkerFailure
		if err := verifySwitchMatrixMarkers(context.Background(), rcon, false); !errors.As(err, &failure) ||
			failure.Stage != "compare" || failure.Dimension != "the_end" || failure.Outcome != "block_mismatch" {
			t.Fatalf("missing End marker passed verification or lost evidence: %v", err)
		}
	})
	t.Run("recovered_failure_continues", func(t *testing.T) {
		s, j, _ := newMatrixRunnerTest(t, 4)
		count := 0
		err := s.operations.runSwitchMatrixCases(j.OperationID, func(c switchMatrixCase) (switchMatrixCaseResult, error) {
			count++
			if count == 2 {
				return switchMatrixCaseResult{Outcome: matrixFailed, Stage: "source_world", Reason: "setblock_rejected",
					World: "source_markers_failed", Cleanup: "verified"}, nil
			}
			if count == 3 {
				return switchMatrixCaseResult{Outcome: matrixFailed, Stage: string(matrixFixtureSoftware),
					Reason: "fixture_software_version_unverified", Recovery: "verified", Cleanup: "verified"}, nil
			}
			return matrixRunnerPassed(c), nil
		})
		p, readErr := s.operations.switchMatrixProgress(j.OperationID)
		if err != nil || readErr != nil || count != 4 || p.Failed != 2 || p.Passed != 2 {
			t.Fatalf("recovered failure stranded the next case: %+v %v %v", p, err, readErr)
		}
	})
	t.Run("runtime_binding_contains_local_only_ports_and_limits", func(t *testing.T) {
		f := newInstanceActivationFixture(t)
		c := f.configs[0]
		c.MatrixFixture, c.StartOnBoot = true, false
		c.JavaMemory, c.ContainerMemory = switchMatrixHeap, switchMatrixMemory
		files, err := f.backend.identities().renderInstanceRuntimes(f.snapshot, []minecraftInstanceRuntimeConfig{c})
		if err != nil || len(files) != 1 {
			t.Fatalf("fixture runtime render: %v", err)
		}
		quadlet := string(files[0].Quadlet)
		for _, key := range []string{"PublishPort=127.0.0.1:", "PodmanArgs=--cpus=2", "Restart=no", "Memory=2560M"} {
			if !strings.Contains(quadlet, key) {
				t.Fatalf("missing fixture restriction %q", key)
			}
		}
		if strings.Contains(quadlet, "[Install]") {
			t.Fatal("fixture enabled at boot")
		}
		next := instanceRuntimeRecord{Runtime: c}
		changes, rules, err := f.backend.planFirewall(context.Background(), next, next)
		if err != nil || len(changes) != 0 || len(rules) != 0 {
			t.Fatalf("fixture made firewall changes: %v %v %v", changes, rules, err)
		}
	})
	t.Run("unresolved_cleanup_halts", func(t *testing.T) {
		s, j, _ := newMatrixRunnerTest(t, 2)
		count := 0
		err := s.operations.runSwitchMatrixCases(j.OperationID, func(switchMatrixCase) (switchMatrixCaseResult, error) {
			count++
			return switchMatrixCaseResult{Outcome: matrixFailed, Cleanup: "unresolved", Reason: "fixture_closure_unverified"}, errInstanceRecovery
		})
		s.operations.holdSwitchMatrixRun(j.OperationID)
		p, readErr := s.operations.switchMatrixProgress(j.OperationID)
		active, activeErr := s.operations.currentMinecraftMaintenance()
		if err == nil || readErr != nil || activeErr != nil || active == nil || active.OperationID != j.OperationID ||
			count != 1 || p.Unfinished != 1 || p.State != "needs_attention" {
			t.Fatalf("unresolved cleanup advanced or released the run: %+v %v %v %v", p, err, readErr, activeErr)
		}
	})
}

type matrixRuntimeTestExit struct {
	status int
	error
}

func (e matrixRuntimeTestExit) ExitCode() int { return e.status }

func TestSwitchMatrixRuntimeVerificationDiagnostics(t *testing.T) {
	f := newInstanceActivationFixture(t)
	f.activateBoth()
	o := matrixTestStore(t, f.backend)
	j, err := o.beginSwitchMatrix(switchMatrixRunnerFingerprint())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = o.transition(j.OperationID, operationValidating, "manifest", "Discovering releases."); err != nil {
		t.Fatal(err)
	}
	if _, err = o.transition(j.OperationID, operationRunning, "fixtures", "Working on fixtures."); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	record, err := f.backend.readRecord(f.configs[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	credential := record.Runtime.RCONPassword
	if credential == "" {
		t.Fatal("fixture does not have a credential")
	}
	// Put a misleading code and a real fixture credential in both channels.
	raw := "image_identity_mismatch: subprocess detail " + credential
	private := errors.New(raw)
	originalCommand := pluginInstallCommand
	t.Cleanup(func() { pluginInstallCommand = originalCommand })
	var commandErr error
	pluginInstallCommand = func(_ context.Context, command string, args ...string) ([]byte, error) {
		switch {
		case command == "/usr/libexec/justvoxel/mjust/verify-minecraft-stack" &&
			reflect.DeepEqual(args, []string{"--instance", record.Runtime.ID, "--switch", record.Runtime.Version}):
			return []byte(raw), commandErr
		default:
			t.Fatalf("unexpected runtime verifier command: %s %v", command, args)
			return nil, nil
		}
	}

	for _, tc := range []struct {
		status int
		reason string
	}{
		{40, "committed_runtime_configuration_mismatch"},
		{41, "container_session_inspection_failed"},
		{42, "container_session_identity_invalid"},
		{43, "container_session_changed"},
		{44, "current_session_startup_evidence_missing"},
		{45, "image_identity_mismatch"},
		{46, "minecraft_status_probe_failed"},
		{47, "minecraft_game_version_mismatch"},
		{48, "minecraft_software_or_loader_mismatch"},
		{49, "runtime_rcon_failed"},
		{50, "managed_crossplay_verification_failed"},
		{51, "runtime_startup_terminated"},
		{99, "exact_runtime_unverified"},
	} {
		commandErr = matrixRuntimeTestExit{status: tc.status, error: private}
		err = verifyPluginRuntimeMode(ctx, record.Runtime.ID, record.Runtime.Version, "--switch")
		stage, reason := switchMatrixStartupDiagnostic("source", errors.Join(err, private))
		if stage != "source_runtime" || reason != tc.reason {
			t.Errorf("exit %d classified as %s/%s, want source_runtime/%s", tc.status, stage, reason, tc.reason)
		}
	}
	backendCommand := f.backend.command
	t.Cleanup(func() { f.backend.command = backendCommand })
	publishedHost := "127.0.0.1"
	f.backend.command = func(ctx context.Context, command string, args ...string) ([]byte, error) {
		if command == "podman" && reflect.DeepEqual(args, []string{"port", "minecraft-" + record.Runtime.ID}) {
			return fmt.Appendf(nil, "25565/tcp -> %s:%d\n", publishedHost, record.Runtime.JavaPort), nil
		}
		return backendCommand(ctx, command, args...)
	}
	if err := verifySwitchMatrixPorts(ctx, record.Runtime); err != nil {
		t.Fatal("Podman loopback port mapping was rejected", err)
	}
	publishedHost = "0.0.0.0"
	portErr := verifySwitchMatrixPorts(ctx, record.Runtime)
	if portErr == nil {
		t.Fatal("non-loopback port mapping passed verification")
	}
	stage, reason := switchMatrixStartupDiagnostic("target", wrapSwitchMatrixFixtureError(matrixFixturePorts, portErr))
	if stage != "target_runtime" || reason != "runtime_port_binding_unverified" {
		t.Fatalf("published port failure classified as %s/%s", stage, reason)
	}

	manifest := matrixRunnerTestManifest(t)
	smoke, missing := selectSwitchMatrixSmoke(manifest)
	if smoke < 0 {
		t.Fatal("test manifest missing smoke case", missing)
	}
	cases := []switchMatrixCase{manifest.Cases[smoke]}
	if err := o.saveSwitchMatrixRun(switchMatrixRun{
		Schema: 1, ID: j.OperationID, State: "running", Cases: cases, Results: newSwitchMatrixResults(cases),
	}); err != nil {
		t.Fatal(err)
	}
	commandErr = matrixRuntimeTestExit{status: 42, error: private}
	if err := o.runSwitchMatrixCases(j.OperationID, func(switchMatrixCase) (switchMatrixCaseResult, error) {
		err := verifyPluginRuntimeMode(ctx, record.Runtime.ID, record.Runtime.Version, "--switch")
		var failure *pluginRuntimeVerificationError
		if !errors.As(err, &failure) || failure.output != "" {
			t.Fatal("Switch retained raw runtime verification output or lost the typed failure")
		}
		stage, reason := switchMatrixStartupDiagnostic("source", errors.Join(err, private))
		return switchMatrixCaseResult{Outcome: matrixFailed, Stage: stage, Reason: reason, Cleanup: "verified"}, nil
	}); err != nil {
		t.Fatal(err)
	}

	s := adminServerForTest()
	s.operations = o
	mux := http.NewServeMux()
	registerSwitchMatrixRoutes(mux, s)
	registerAdminDiagnosticLogRoutes(mux, s)
	request := func(path string) *httptest.ResponseRecorder {
		t.Helper()
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, authorizedRequest(http.MethodGet, path, ""))
		if w.Code != http.StatusOK || strings.Contains(w.Body.String(), credential) ||
			strings.Contains(w.Body.String(), "subprocess detail") {
			t.Fatalf("unsafe or unavailable response for %s (HTTP %d)", path, w.Code)
		}
		return w
	}
	report := request("/v1/admin/logs/migration/" + j.OperationID + "?download=1")
	var run switchMatrixRun
	if err := json.Unmarshal(report.Body.Bytes(), &run); err != nil {
		t.Fatal(err)
	}
	if report.Header().Get("Content-Disposition") == "" || len(run.Results) != 1 ||
		run.Results[0].Stage != "source_runtime" || run.Results[0].Reason != "container_session_identity_invalid" {
		t.Fatal("download did not preserve the verified diagnostic classification")
	}
	progress := request(switchMatrixAPIPath + "?operation_id=" + j.OperationID)
	var response switchMatrixAPIResponse
	if err := json.Unmarshal(progress.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Operation == nil || response.Operation.State != "halted" ||
		response.Operation.Progress == nil || response.Operation.Progress.Failed != 1 {
		t.Fatal("public progress did not reflect the halted verification failure")
	}
}

func TestSwitchMatrixCancellation(t *testing.T) {
	s, j, _ := newMatrixRunnerTest(t, 3)
	o := s.operations
	count := 0
	err := o.runSwitchMatrixCases(j.OperationID, func(c switchMatrixCase) (switchMatrixCaseResult, error) {
		count++
		if err := o.stopSwitchMatrixRun(j.OperationID); err != nil {
			t.Fatal(err)
		}
		p, err := o.switchMatrixProgress(j.OperationID)
		if err != nil || !p.CancelRequested || p.Current == nil || p.Current.ID != c.ID {
			t.Fatalf("stop did not keep current case running: %+v %v", p, err)
		}
		return matrixRunnerPassed(c), nil
	})
	r, readErr := o.readSwitchMatrixRun(j.OperationID)
	if err != nil || readErr != nil || count != 1 || r.Progress.State != "cancelled" ||
		r.Progress.Completed != 1 || r.Progress.Unfinished != 2 || r.Results[1].Outcome != matrixCancelled || r.Results[2].Outcome != matrixCancelled {
		t.Fatalf("bad cancellation at transaction boundary: %+v %v %v", r.Progress, err, readErr)
	}
}

func TestSwitchMatrixInterruptedRecovery(t *testing.T) {
	s, j, r := newMatrixRunnerTest(t, 3)
	o := s.operations
	r.Results[0] = matrixRunnerPassed(r.Cases[0])
	r.Results[1] = switchMatrixCaseResult{Outcome: matrixExecuting, StartedAt: time.Now()}
	r.CancelRequested = true
	if err := o.saveSwitchMatrixRun(r); err != nil {
		t.Fatal(err)
	}
	if err := o.close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := openOperationStore(o.baseDir)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.close()
	r, err = reopened.readSwitchMatrixRun(j.OperationID)
	if err != nil || r.State != "interrupted" || !r.CancelRequested || r.Results[0].Outcome != matrixPassed ||
		r.Results[1].Outcome != matrixInterrupted || r.Results[2].Outcome != matrixPending ||
		r.Progress.Passed != 1 || r.Progress.Unfinished != 2 {
		t.Fatalf("lost interrupted results: %+v %v", r, err)
	}
	s.operations = reopened
	if _, err = s.startSwitchMatrixRun(); err == nil {
		t.Fatal("restarted matrix admitted another run")
	}
	if err = s.recoverSwitchMatrixRun(j.OperationID); err != nil {
		t.Fatal(err)
	}
	r, err = reopened.readSwitchMatrixRun(j.OperationID)
	active, activeErr := reopened.currentMinecraftMaintenance()
	if err != nil || activeErr != nil || active != nil || r.State != "recovered" || r.Progress.Unfinished != 2 ||
		r.Results[1].Outcome != matrixInterrupted || r.Results[1].Cleanup != "verified" || r.Results[2].Outcome != matrixPending {
		t.Fatalf("explicit recovery reran cases or lost closure: %+v %v %v", r, err, activeErr)
	}
}

func TestSwitchMatrixResultAccounting(t *testing.T) {
	s, j, r := newMatrixRunnerTest(t, 1)
	o := s.operations
	r.Cases = buildSwitchMatrixCases(context.Background(), map[string][]string{switchMatrixLegacy: {"1.21.11"}, switchMatrixModern: {"26.3"}},
		matrixRunnerTestRelease, func(context.Context, string, string, *switchMatrixRuntime) string { return "" })
	r.Results = make([]switchMatrixCaseResult, len(r.Cases))
	outcomes := []string{matrixPassed, matrixFailed, matrixUnsupported, matrixBlocked, matrixInterrupted, matrixCancelled, matrixPending}
	expected := map[string]int{}
	for i := range r.Results {
		outcome := outcomes[i%len(outcomes)]
		expected[outcome]++
		r.Results[i] = switchMatrixCaseResult{Outcome: outcome, Runtime: "source_verified", World: "target_markers_failed",
			Original: "verified", Recovery: "verified", Cleanup: "verified", Stage: "target_world", Reason: "world_markers_unverified",
			DurationMS: 60000, StartedAt: time.Now()}
	}
	if err := o.saveSwitchMatrixRun(r); err != nil {
		t.Fatal(err)
	}
	before, err := o.readDiagnosticLog("migration", j.OperationID)
	if err != nil || len(before) == 0 || len(before) >= operationDiagnosticLimit {
		t.Fatalf("complete manifest exceeded diagnostic capacity: %d %v", len(before), err)
	}
	entries, err := o.listDiagnosticLogs()
	if err != nil || !slices.ContainsFunc(entries, func(e diagnosticLogEntry) bool {
		return e.LogID == j.OperationID && e.Category == "migration" && e.OperationType == operationTypeSwitchMatrix
	}) {
		t.Fatalf("run missing from Migration Logs: %v %v", entries, err)
	}
	read, err := o.readSwitchMatrixRun(j.OperationID)
	p := read.Progress
	if err != nil || len(read.Results) != len(r.Cases) || p.Passed != expected[matrixPassed] ||
		p.Failed != expected[matrixFailed] || p.Unsupported != expected[matrixUnsupported] || p.Blocked != expected[matrixBlocked] ||
		p.Unfinished != expected[matrixInterrupted]+expected[matrixCancelled]+expected[matrixPending] || p.Completed+p.Unfinished != len(r.Cases) {
		t.Fatalf("bad persisted accounting: %+v %v", read.Progress, err)
	}
	for i, result := range read.Results {
		if result.CaseID != r.Cases[i].ID {
			t.Fatalf("download lost case identity at %d", i)
		}
	}
	records, err := instanceRuntimes.records()
	if err != nil || len(records) == 0 {
		t.Fatalf("fixture credentials could not be inspected: %v", err)
	}
	mockLeaf := matrixRunnerTestRelease(context.Background(), "leaf", "1.21.11").Runtime
	diagnostic := string(before)
	for _, record := range records {
		password := record.Runtime.RCONPassword
		if password == "" {
			t.Fatal("fixture runtime credential is empty")
		}
		containsPassword := func(document string) bool {
			return strings.Contains(document, password)
		}
		if containsPassword(mockLeaf.Digest) || containsPassword(mockLeaf.ImageDigest) {
			t.Fatal("fixture runtime credential collides with a mock release checksum")
		}
		if !containsPassword(`{"detail":"unexpected credential ` + password + `"}`) {
			t.Fatal("credential substring check missed an intentional fixture runtime credential leak")
		}
		if containsPassword(diagnostic) {
			t.Fatal("fixture runtime credential leaked")
		}
	}
	raw, err := os.ReadFile(o.diagnosticCategoryDirectory("migration") + "/" + j.OperationID + ".log")
	if err != nil || !bytes.Equal(raw, before) || bytes.Contains(raw, []byte("rcon_password")) {
		t.Fatalf("unexpected downloadable diagnostic: %v", err)
	}
}

func TestSwitchMatrixSmokeGate(t *testing.T) {
	for _, scenario := range []struct {
		name, state, reason string
	}{
		{"passes", "completed", ""},
		{"fails", "halted", "smoke_case_failed"},
		{"unverified_pass", "halted", "smoke_case_failed"},
		{"publisher_unavailable", "halted", "smoke_exact_release_unverified"},
		{"cleanup_unresolved", "needs_attention", "smoke_case_failed"},
		{"cancelled", "cancelled", ""},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			s, j, _ := newMatrixRunnerTest(t, 2)
			o := s.operations
			manifest := matrixRunnerTestManifest(t)
			smoke, reason := selectSwitchMatrixSmoke(manifest)
			if smoke <= 0 || reason != "" {
				t.Fatal("expected a verified smoke case later in the generated manifest")
			}
			smokeCase := manifest.Cases[smoke]
			if scenario.name == "publisher_unavailable" {
				catalog := manifest.Catalog
				manifest.Cases = buildSwitchMatrixCases(context.Background(), catalog,
					func(ctx context.Context, software, version string) switchMatrixRelease {
						if software == "purpur" && java25GameReleasePattern.MatchString(version) {
							return switchMatrixRelease{Outcome: switchMatrixReleaseUnverified, Verification: "publisher_request_timeout"}
						}
						return matrixRunnerTestRelease(ctx, software, version)
					}, func(context.Context, string, string, *switchMatrixRuntime) string { return "" })
				manifest.Results = newSwitchMatrixResults(manifest.Cases)
			}
			manifest.Schema, manifest.ID, manifest.State = 1, j.OperationID, "running"
			if err := o.saveSwitchMatrixRun(manifest); err != nil {
				t.Fatal(err)
			}
			seen := map[string]bool{}
			err := o.runSwitchMatrixCases(j.OperationID, func(c switchMatrixCase) (switchMatrixCaseResult, error) {
				if seen[c.ID] {
					t.Fatal("case admitted twice", c.ID)
				}
				if len(seen) == 0 && c.ID != smokeCase.ID {
					t.Fatal("first case was not the selected smoke case", c.ID)
				}
				seen[c.ID] = true
				switch scenario.name {
				case "fails":
					return switchMatrixCaseResult{Outcome: matrixFailed, Stage: "target_world",
						Reason: "world_markers_unverified", Cleanup: "verified"}, nil
				case "unverified_pass":
					result := matrixRunnerPassed(c)
					result.Original = ""
					return result, nil
				case "cleanup_unresolved":
					return switchMatrixCaseResult{Outcome: matrixFailed, Stage: "cleanup",
						Reason: "fixture_closure_unverified", Cleanup: "unresolved"}, errInstanceRecovery
				case "cancelled":
					if err := o.stopSwitchMatrixRun(j.OperationID); err != nil {
						t.Fatal(err)
					}
					p, err := o.switchMatrixProgress(j.OperationID)
					if err != nil || p.Current == nil || p.Current.ID != smokeCase.ID || !p.CancelRequested {
						t.Fatal("cancellation interrupted the admitted smoke case", err)
					}
				}
				return matrixRunnerPassed(c), nil
			})
			wantRecovery := scenario.name == "cleanup_unresolved"
			if (err != nil) != wantRecovery {
				t.Fatalf("unexpected runner error: %v", err)
			}
			if wantRecovery {
				o.holdSwitchMatrixRun(j.OperationID)
			}
			run, err := o.readSwitchMatrixRun(j.OperationID)
			if err != nil || run.State != scenario.state || run.HaltReason != scenario.reason ||
				!reflect.DeepEqual(run.Cases, manifest.Cases) || run.Progress.Total != len(manifest.Cases) {
				t.Fatalf("manifest or halt state changed: %s %s %v", run.State, run.HaltReason, err)
			}
			active, err := o.currentMinecraftMaintenance()
			if err != nil || (active != nil) != wantRecovery {
				t.Fatal("incorrect maintenance reservation", err)
			}
			wantAdmissions := 1
			switch scenario.name {
			case "passes":
				wantAdmissions = len(manifest.Cases) - manifest.progress().Completed
			case "publisher_unavailable":
				wantAdmissions = 0
			}
			if len(seen) != wantAdmissions || (wantAdmissions != 0 && run.SmokeCase != smokeCase.ID) ||
				(wantAdmissions == 0 && (run.SmokeCase != "" || run.Progress.Passed != 0)) {
				t.Fatal("unexpected smoke admission or result", len(seen), run.Progress)
			}
			if scenario.name == "publisher_unavailable" && !slices.ContainsFunc(run.Cases, func(c switchMatrixCase) bool {
				return c.Source == "vanilla" && c.Target == "purpur" && c.Action == "clone" &&
					c.Family == switchMatrixModern && c.TargetRelease.Verification == "publisher_request_timeout"
			}) {
				t.Fatal("publisher verification cause was lost from the manifest")
			}
			if scenario.name == "fails" && (run.Results[smoke].Reason != "world_markers_unverified" ||
				run.Results[smoke].Cleanup != "verified") {
				t.Fatal("smoke failure evidence was lost")
			}
			if scenario.name == "unverified_pass" && (run.Progress.Passed != 0 ||
				run.Results[smoke].Reason != "case_verification_incomplete") {
				t.Fatal("incomplete evidence was accepted as a smoke pass")
			}
			if scenario.name == "cancelled" && run.Results[smoke].Outcome != matrixPassed {
				t.Fatal("the completed smoke result was lost on cancellation")
			}
			for i, c := range run.Cases {
				if !seen[c.ID] && manifest.Results[i].Outcome == matrixPending {
					want := matrixPending
					if scenario.name == "cancelled" {
						want = matrixCancelled
					}
					if run.Results[i].Outcome != want {
						t.Fatalf("case %s was never admitted and has outcome %s", c.ID, run.Results[i].Outcome)
					}
				}
			}
		})
	}
}

func TestSwitchMatrixInfrastructureFailureStopsRun(t *testing.T) {
	s, j, manifest := newMatrixRunnerTest(t, 4)
	o := s.operations
	var admitted []string
	err := o.runSwitchMatrixCases(j.OperationID, func(c switchMatrixCase) (switchMatrixCaseResult, error) {
		admitted = append(admitted, c.ID)
		if len(admitted) == 1 {
			return matrixRunnerPassed(c), nil
		}
		stage, reason := switchMatrixFixtureDiagnostic(wrapSwitchMatrixFixtureError(matrixFixturePublication, errInstanceRecovery))
		return switchMatrixCaseResult{Outcome: matrixFailed, Stage: stage, Reason: reason, Cleanup: "verified"}, nil
	})
	run, readErr := o.readSwitchMatrixRun(j.OperationID)
	summary, statusErr := o.switchMatrixSummary(j.OperationID)
	if err != nil || readErr != nil || statusErr != nil || len(admitted) != 2 || !reflect.DeepEqual(run.Cases, manifest.Cases) ||
		run.State != "halted" || run.HaltReason != "common_fixture_infrastructure_failed" ||
		run.Progress.Passed != 1 || run.Progress.Failed != 1 || run.Progress.Unfinished != 2 ||
		run.Results[1].Stage != "runtime_publication" || run.Results[1].Reason != "fixture_runtime_publication_failed" ||
		run.Results[2].Outcome != matrixPending || run.Results[3].Outcome != matrixPending ||
		summary == nil || summary.State != "halted" || summary.ReservationActive || summary.RecoveryRequired {
		t.Fatalf("common failure did not close with the complete partial result: %+v %v %v %v", run.Progress, err, readErr, statusErr)
	}
}

func TestSwitchMatrixMarkerDiagnostics(t *testing.T) {
	for _, scenario := range []struct {
		stage, outcome, reason, reply string
		err                           error
	}{
		{"forceload_add", "transport_error", "rcon_transport_failure", "", errors.New("private transport output")},
		{"forceload_add", "unknown_dimension", "dimension_unavailable", "Unknown dimension 'minecraft:overworld'", nil},
		{"forceload_add", "ticket_unverified", "chunk_loading_failed", "", nil},
		{"chunk_ready", "command_deadline", "timeout", "", context.DeadlineExceeded},
		{"setblock", "block_change_unconfirmed", "setblock_rejected", "Could not set the block", nil},
		{"setblock", "unexpected_response", "setblock_unverified", "", nil},
		{"compare", "block_mismatch", "marker_comparison_failed", "Test failed", nil},
		{"compare", "position_unloaded", "chunk_loading_failed", "That position is not loaded", nil},
		{"save_flush", "save_rejected", "save_flush_failed", "Unable to save the game (is there enough disk space?)", nil},
		{"save_flush", "save_not_confirmed", "save_flush_unverified", "Saving the game (this may take a moment!)", nil},
		{"post_save_compare", "block_mismatch", "marker_comparison_failed", "Test failed", nil},
	} {
		t.Run(scenario.stage+"/"+scenario.outcome, func(t *testing.T) {
			saved := false
			rcon := func(_ context.Context, cmd string) ([]byte, error) {
				if cmd == "save-all flush" {
					saved = true
					if scenario.stage == "save_flush" {
						return []byte(scenario.reply), scenario.err
					}
					return []byte("Saved the game"), nil
				}
				for _, m := range switchMatrixMarkers {
					prefix := "execute in minecraft:" + m.Dimension + " "
					if !strings.HasPrefix(cmd, prefix) {
						continue
					}
					var stage, reply string
					switch {
					case strings.Contains(cmd, "forceload add"):
						stage, reply = "forceload_add", "Marked chunk [0, 0] in minecraft:"+m.Dimension+" to be force loaded"
					case strings.Contains(cmd, "if loaded"):
						stage, reply = "chunk_ready", "Test passed"
					case strings.Contains(cmd, "setblock"):
						stage, reply = "setblock", "Changed the block at "+strings.ReplaceAll(m.Position, " ", ", ")
					case strings.Contains(cmd, "if block"):
						stage, reply = "compare", "Test passed"
					default:
						t.Fatal("unexpected marker command", cmd)
					}
					if saved {
						stage = "post_save_" + stage
					}
					if stage == scenario.stage {
						return []byte(scenario.reply), scenario.err
					}
					return []byte(reply), nil
				}
				t.Fatal("unexpected marker command", cmd)
				return nil, nil
			}
			err := verifySwitchMatrixMarkers(context.Background(), rcon, true)
			var detail *switchMatrixMarkerFailure
			if !errors.Is(err, errSwitchMatrixWorldMarkers) || !errors.As(err, &detail) ||
				detail.Stage != scenario.stage || detail.Reason != scenario.reason || detail.Outcome != scenario.outcome {
				t.Fatalf("incorrect marker diagnostic: %+v (%v)", detail, err)
			}
			wantDimension := "overworld"
			if scenario.stage == "save_flush" {
				wantDimension = ""
			}
			if detail.Dimension != wantDimension || bytes.Contains(mustSwitchJSON(detail), []byte("private transport output")) {
				t.Fatalf("invalid or unsafe marker diagnostic: %+v", detail)
			}
		})
	}
}

func TestSwitchMatrixDoesNotContinueAfterUnverifiedRecovery(t *testing.T) {
	for _, scenario := range []struct {
		name, stage string
		cause       error
		wantRetries int
	}{
		{"closure", "source_world", errInstanceRecovery, 0},
		{"ownership", "source_ownership", &switchMatrixRetryableClosure{reason: "fixture_shutdown_unverified", cause: errInstanceBusy}, 0},
		{"retry_cannot_verify_closure", "source_world", &switchMatrixRetryableClosure{reason: "fixture_shutdown_unverified", cause: errInstanceBusy}, 1},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			s, j, _ := newMatrixRunnerTest(t, 3)
			o := s.operations
			admitted, retries := 0, 0
			err := o.runSwitchMatrixCases(j.OperationID, func(c switchMatrixCase) (switchMatrixCaseResult, error) {
				admitted++
				if admitted == 1 {
					return matrixRunnerPassed(c), nil
				}
				return switchMatrixCaseResult{Outcome: matrixFailed, Stage: scenario.stage, Cleanup: "unresolved",
					ClosureReason: "fixture_shutdown_unverified"}, scenario.cause
			}, func() (operationJournal, error) {
				retries++
				return operationJournal{}, errInstanceRecovery
			})
			o.holdSwitchMatrixRun(j.OperationID)
			run, readErr := o.readSwitchMatrixRun(j.OperationID)
			active, activeErr := o.currentMinecraftMaintenance()
			if err == nil || readErr != nil || activeErr != nil || active == nil || admitted != 2 || retries != scenario.wantRetries ||
				run.State != "needs_attention" || run.Results[1].ClosureReason != "fixture_shutdown_unverified" ||
				run.Results[1].Cleanup != "unresolved" || run.Results[2].Outcome != matrixPending {
				t.Fatalf("unsafe continuation or lost diagnostics: %+v admissions=%d retries=%d errors=%v %v %v", run, admitted, retries, err, readErr, activeErr)
			}
		})
	}
}
