//go:build justvoxel_dev

package main

import (
	"context"
	"slices"
	"testing"
)

const targetedLeafRunning = "fabric.leaf.clone.26_1_plus.java.running"
const targetedLeafStopped = "fabric.leaf.clone.26_1_plus.java.stopped"

func TestSwitchMatrixTargetedManifest(t *testing.T) {
	ids := []string{targetedLeafStopped, targetedLeafRunning, "neoforge.pufferfish.clone.pre_26_1.java.running"}
	for _, bad := range [][]string{nil, {targetedLeafRunning, targetedLeafRunning}, {"../data"}, {""}, make([]string, 17)} {
		if err := validateSwitchMatrixSelection(bad); err == nil {
			t.Fatal("invalid matrix selection admitted", bad)
		}
	}
	if err := validateSwitchMatrixSelection(ids); err != nil {
		t.Fatal(err)
	}
	catalog := map[string][]string{switchMatrixModern: {"26.45.7", "26.3"}, switchMatrixLegacy: {"1.21.11", "1.21.10"}}
	cases := buildSwitchMatrixCases(context.Background(), catalog, func(ctx context.Context, software, version string) switchMatrixRelease {
		if !slices.Contains([]string{"fabric", "leaf", "neoforge", "pufferfish"}, software) {
			t.Fatal("targeted discovery probed an unrelated publisher", software)
		}
		if software == "pufferfish" && version == "1.21.11" {
			return switchMatrixRelease{Outcome: switchMatrixReleaseUnverified, Verification: "publisher_selection_unverified"}
		}
		release := matrixRunnerTestRelease(ctx, software, version)
		if software == "pufferfish" {
			release.Runtime.PufferfishBuild = 39
		}
		return release
	}, func(context.Context, string, string, *switchMatrixRuntime) string {
		t.Fatal("Java-only selection performed a cross-play probe")
		return ""
	}, ids...)
	if len(cases) != len(ids) {
		t.Fatalf("targeted manifest returned %d cases, want %d", len(cases), len(ids))
	}
	cases = orderTargetedSwitchMatrixCases(ids, cases)
	versions := []string{"26.45.7", "26.45.7", "1.21.10"}
	for i, c := range cases {
		if c.ID != ids[i] || c.Version != versions[i] || c.Disposition != "" || c.Crossplay || c.TargetCrossplay {
			t.Fatalf("targeted manifest did not preserve requested case: %+v", c)
		}
	}
}

func TestSwitchMatrixEarlyCrossplayPolicyAndGeyserEvidence(t *testing.T) {
	catalog := map[string][]string{switchMatrixLegacy: {"1.21.11"}}
	ids := []string{
		"fabric.forge.clone.pre_26_1.crossplay.running",
		"fabric.quilt.clone.pre_26_1.crossplay.stopped",
		"fabric.vanilla.clone.pre_26_1.crossplay.running",
		"fabric.forge.convert.pre_26_1.crossplay.running",
		"fabric.quilt.convert.pre_26_1.crossplay.stopped",
		"fabric.vanilla.convert.pre_26_1.crossplay.running",
	}
	for _, check := range []struct {
		name, diagnostic, disposition, reason string
	}{
		{name: "verified"},
		{name: "geyser_incompatible", diagnostic: "geyser_java_target_mismatch", disposition: matrixUnsupported, reason: "managed_crossplay_stack_unsupported"},
		{name: "geyser_failure", diagnostic: "geyser_request_failed", disposition: matrixFailed, reason: "managed_crossplay_stack_unverified"},
		{name: "viaversion_incompatible", diagnostic: "viaversion_exact_release_unsupported", disposition: matrixUnsupported, reason: "managed_crossplay_stack_unsupported"},
		{name: "viaversion_artifact_unverified", diagnostic: "viaversion_release_unverified", disposition: matrixFailed, reason: "managed_crossplay_stack_unverified"},
	} {
		t.Run(check.name, func(t *testing.T) {
			var probes []string
			cases := buildSwitchMatrixCases(context.Background(), catalog, matrixRunnerTestRelease,
				func(_ context.Context, software, _ string, _ *switchMatrixRuntime) string {
					probes = append(probes, software)
					return check.diagnostic
				}, ids...)
			if !slices.Equal(probes, []string{"fabric"}) {
				t.Fatal("Java-only targets should only probe the source cross-play stack", probes)
			}
			if len(cases) != len(ids) {
				t.Fatalf("cross-play manifest returned %d cases, want %d", len(cases), len(ids))
			}
			cases = orderTargetedSwitchMatrixCases(ids, cases)
			for _, c := range cases[:3] {
				if c.Disposition != check.disposition || c.Reason != check.reason || c.SourceCrossplayCheck != check.diagnostic ||
					!c.Crossplay || c.TargetCrossplay || c.TargetCrossplayCheck != "" {
					t.Fatalf("Java-only Clone did not reflect source cross-play evidence: %+v", c)
				}
			}
			for _, c := range cases[3:] {
				if c.Disposition != matrixBlocked || c.Reason != "switch_world_policy" {
					t.Fatalf("cross-play source unexpectedly permits Convert: %+v", c)
				}
			}
		})
	}
}

func TestSwitchMatrixTargetedCancellationAndUnavailableSelection(t *testing.T) {
	for _, unavailable := range []bool{false, true} {
		t.Run(map[bool]string{false: "cancel", true: "unavailable"}[unavailable], func(t *testing.T) {
			s, j, run := newMatrixRunnerTest(t, 2)
			o := s.operations
			run.Targeted = true
			catalog := map[string][]string{switchMatrixModern: {"26.3"}}
			run.Cases = orderTargetedSwitchMatrixCases([]string{targetedLeafRunning, targetedLeafStopped},
				buildSwitchMatrixCases(context.Background(), catalog, matrixRunnerTestRelease, nil, targetedLeafRunning, targetedLeafStopped))
			if unavailable {
				run.Cases[1].Disposition, run.Cases[1].Reason = matrixUnsupported, "exact_release_unsupported"
			}
			run.Results = newSwitchMatrixResults(run.Cases)
			if err := o.saveSwitchMatrixRun(run); err != nil {
				t.Fatal(err)
			}
			entered := 0
			if err := o.runSwitchMatrixCases(j.OperationID, func(c switchMatrixCase) (switchMatrixCaseResult, error) {
				entered++
				if err := o.stopSwitchMatrixRun(j.OperationID); err != nil {
					t.Fatal(err)
				}
				return matrixRunnerPassed(c), nil
			}); err != nil {
				t.Fatal(err)
			}
			got, err := o.readSwitchMatrixRun(j.OperationID)
			active, activeErr := o.currentMinecraftMaintenance()
			if err != nil || activeErr != nil || active != nil || !got.Targeted || got.SmokeCase != "" {
				t.Fatal("targeted run lost its identity or closure", err, activeErr)
			}
			if unavailable {
				if entered != 0 || got.HaltReason != "targeted_selection_unavailable" || got.Results[0].Outcome != matrixPending {
					t.Fatal("unavailable selection admitted work", entered, got.HaltReason, got.Results)
				}
			} else if entered != 1 || got.State != "cancelled" || got.Results[0].Cleanup != "verified" ||
				got.Results[0].Outcome != matrixPassed || got.Results[1].Outcome != matrixCancelled {
				t.Fatal("targeted cancellation lost the admitted case's closure", entered, got.State, got.Results)
			}
		})
	}
}

func TestSwitchMatrixCrossplayCandidateClassifications(t *testing.T) {
	const id = "paper.fabric.clone.pre_26_1.crossplay.running"
	ctx := context.Background()
	for _, check := range []struct {
		name, older, newer, wantVersion, disposition, reason string
	}{
		{"exact_older", "", "geyser_java_target_mismatch", "1.21.10", "", ""},
		{"no_compatible_stack", "geyser_java_target_mismatch", "geyser_java_target_mismatch", "1.21.11", matrixUnsupported, "managed_crossplay_stack_unsupported"},
		{"older_lookup_inconclusive", "geyser_artifact_unverified", "geyser_java_target_mismatch", "1.21.10", matrixFailed, "managed_crossplay_stack_unverified"},
	} {
		t.Run(check.name, func(t *testing.T) {
			catalog := map[string][]string{switchMatrixLegacy: {"1.21.11", "1.21.10"}}
			cases := buildSwitchMatrixCases(ctx, catalog, matrixRunnerTestRelease,
				func(_ context.Context, name, version string, _ *switchMatrixRuntime) string {
					if name == "paper" {
						return ""
					}
					if version == "1.21.11" {
						return check.newer
					}
					return check.older
				}, id)
			if len(cases) != 1 {
				t.Fatalf("missing matrix case: %+v", cases)
			}
			c := cases[0]
			if c.Version != check.wantVersion || c.Disposition != check.disposition || c.Reason != check.reason ||
				!c.Crossplay || !c.TargetCrossplay || c.SourceCrossplayCheck != "" {
				t.Fatalf("incorrect exact selection or cross-play policy: %+v", c)
			}
		})
	}
}

func TestSwitchMatrixRepresentativeProfiles(t *testing.T) {
	cases := matrixRunnerTestManifest(t).Cases
	full := addSwitchMatrixCoverage(slices.Clone(cases))
	if len(full) != len(cases)+2 || !slices.Equal(full[:len(cases)], cases) ||
		full[len(cases)].Profile != switchMatrixInitialSource || full[len(cases)+1].Profile != switchMatrixManagedContent {
		t.Fatal("coverage replaced an existing full-matrix runtime case")
	}
	initial := minecraftSwitchPlan{SourceUninitialized: true, Clone: true, InitializationFingerprint: instanceHash([]byte("initial"))}
	if !verifySwitchMatrixFixturePlan(switchMatrixInitialSource, initial) {
		t.Fatal("verified uninitialized Clone rejected")
	}
	initial.Convert = true
	if verifySwitchMatrixFixturePlan(switchMatrixInitialSource, initial) {
		t.Fatal("uninitialized Convert was admitted")
	}
	content := minecraftSwitchPlan{Clone: true, RequiresExclusionConsent: true, Content: []switchContent{
		{Directory: "plugins", Name: "LuckPerms", Compatibility: "verified", CloneDisposition: "retain"},
		{Directory: "mods", Name: "LuckPerms", Compatibility: "incompatible", CloneDisposition: "exclude"},
		{Directory: "plugins", Name: "ViaBackwards", Compatibility: "unknown", CloneDisposition: "exclude",
			DependencyProof: []switchDependencyDecision{{Project: "missing"}}},
	}}
	if !verifySwitchMatrixFixturePlan(switchMatrixManagedContent, content) {
		t.Fatal("verified mixed-content Clone rejected")
	}
	content.Convert = true
	if verifySwitchMatrixFixturePlan(switchMatrixManagedContent, content) {
		t.Fatal("Convert admitted with content exclusions")
	}
}
