//go:build justvoxel_dev

package main

import (
	"bytes"
	"context"
	"errors"
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
	stack := func(_ context.Context, _ string, version string, _ *switchMatrixRuntime) bool {
		return version != "26.3"
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
						wantBlocked := action == "convert" && switchWorldRisk(k[0], k[1]) ||
							enabled && switchRejectsCrossplayPlatform(k[0], k[1], true)
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
	t.Run("publisher_protocol_rejects_untrusted_details", func(t *testing.T) {
		for _, scenario := range []struct {
			raw  string
			want switchMatrixReleaseOutcome
		}{
			{`{"outcome":"AVAILABLE","runtime":{"game":"26.3","loader":"","image":"java25"}}`, switchMatrixReleaseAvailable},
			{`{"outcome":"UNSUPPORTED"}`, switchMatrixReleaseUnsupported},
			{`{"outcome":"UNVERIFIED"}`, switchMatrixReleaseUnverified},
			{`{"outcome":"UNSUPPORTED","runtime":{"game":"26.3","loader":"","image":"java25"}}`, switchMatrixReleaseUnverified},
			{`{"outcome":"AVAILABLE","runtime":{"game":"26.3","loader":"","image":"java25","reason":"password=private-value"}}`, switchMatrixReleaseUnverified},
			{`{"outcome":"AVAILABLE","runtime":{"game":"26.2","loader":"","image":"java25"}}`, switchMatrixReleaseUnverified},
			{`{"outcome":"ERROR"}`, switchMatrixReleaseUnverified},
		} {
			r := decodeSwitchMatrixRelease("paper", "26.3", []byte(scenario.raw))
			if r.Outcome != scenario.want || r.Outcome != switchMatrixReleaseAvailable && r.Runtime != nil {
				t.Fatalf("bad publisher classification: %+v", r)
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
	for _, c := range buildSwitchMatrixCases(context.Background(), map[string][]string{switchMatrixLegacy: {"1.21.11"}}, matrixRunnerTestRelease,
		func(context.Context, string, string, *switchMatrixRuntime) bool { return true }) {
		if c.Disposition != "" || len(r.Cases) == n {
			continue
		}
		r.Cases = append(r.Cases, c)
		r.Results = append(r.Results, switchMatrixCaseResult{Outcome: matrixPending})
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
		written := map[string]bool{}
		var saved bool
		rcon := func(ctx context.Context, command string) ([]byte, error) {
			for _, m := range switchMatrixMarkers {
				prefix := "execute in minecraft:" + m.Dimension + " "
				switch command {
				case prefix + "run forceload add 8 8", prefix + "run forceload add 9 9", prefix + "run forceload add 10 10":
					return []byte("Marked chunk"), nil
				case prefix + "run setblock " + m.Position + " minecraft:" + m.Block + " replace":
					written[m.Dimension] = true
					return []byte("Changed the block"), nil
				case prefix + "if block " + m.Position + " minecraft:" + m.Block:
					if written[m.Dimension] {
						return []byte("Test passed"), nil
					}
					return []byte("Test failed"), nil
				}
			}
			if command == "save-all flush" {
				saved = true
				return []byte("Saved the game"), nil
			}
			t.Fatalf("unexpected RCON command: %q", command)
			return nil, nil
		}
		if err := verifySwitchMatrixMarkers(context.Background(), rcon, true); err != nil || !saved || len(written) != 3 {
			t.Fatalf("failed real marker command contract: %v", err)
		}
		delete(written, "the_end")
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := verifySwitchMatrixMarkers(ctx, rcon, false); !errors.Is(err, errSwitchMatrixWorldMarkers) {
			t.Fatalf("missing End marker passed verification: %v", err)
		}
	})
	t.Run("recovered_failure_continues", func(t *testing.T) {
		s, j, _ := newMatrixRunnerTest(t, 2)
		count := 0
		err := s.operations.runSwitchMatrixCases(j.OperationID, func(c switchMatrixCase) (switchMatrixCaseResult, error) {
			count++
			if count == 1 {
				return switchMatrixCaseResult{Outcome: matrixFailed, Recovery: "verified", Cleanup: "verified"}, nil
			}
			return matrixRunnerPassed(c), nil
		})
		p, readErr := s.operations.switchMatrixProgress(j.OperationID)
		if err != nil || readErr != nil || count != 2 || p.Failed != 1 || p.Passed != 1 {
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
		matrixRunnerTestRelease, func(context.Context, string, string, *switchMatrixRuntime) bool { return true })
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
