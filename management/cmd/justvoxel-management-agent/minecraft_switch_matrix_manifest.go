package main

import (
	"context"
	"slices"
	"strings"
	"time"
)

const (
	switchMatrixLegacy         = "pre_26_1"
	switchMatrixModern         = "26_1_plus"
	switchMatrixCatalogLimit   = 8 // per layout family
	switchMatrixVersionHelper  = "/usr/libexec/justvoxel/mjust/admin-version-status-json"
	switchMatrixInitialSource  = "uninitialized"
	switchMatrixManagedContent = "managed_content"
)

type switchMatrixCase struct {
	ID                   string              `json:"id"`
	Source               string              `json:"source"`
	Target               string              `json:"target"`
	Family               string              `json:"world_layout_family,omitempty"`
	Version              string              `json:"minecraft_version,omitempty"`
	Action               string              `json:"action,omitempty"`
	Crossplay            bool                `json:"source_crossplay"`
	TargetCrossplay      bool                `json:"target_crossplay"`
	SourceState          string              `json:"original_state,omitempty"`
	SourceRelease        switchMatrixRelease `json:"source_release,omitzero"`
	TargetRelease        switchMatrixRelease `json:"target_release,omitzero"`
	Disposition          string              `json:"disposition,omitempty"`
	Reason               string              `json:"reason,omitempty"`
	ReleaseReason        string              `json:"release_reason,omitempty"`
	SourceCrossplayCheck string              `json:"source_crossplay_verification,omitempty"`
	TargetCrossplayCheck string              `json:"target_crossplay_verification,omitempty"`
	Profile              string              `json:"fixture_profile,omitempty"`
}

// Add two representative cases while keeping every existing runtime case.
// Use the same exact releases and image selection as the stopped Java rows.
func addSwitchMatrixCoverage(cases []switchMatrixCase) []switchMatrixCase {
	for _, pair := range []struct{ source, target, profile string }{
		{"vanilla", "purpur", switchMatrixInitialSource},
		{"paper", "purpur", switchMatrixManagedContent},
	} {
		for _, c := range cases {
			if c.Source != pair.source || c.Target != pair.target || c.Action != "clone" ||
				c.Crossplay || c.Family != switchMatrixModern || c.SourceState == "running" {
				continue
			}
			c.Profile = pair.profile
			c.SourceState = "stopped"
			c.ID = c.canonicalID() + "." + pair.profile
			cases = append(cases, c)
			break
		}
	}
	return cases
}

func switchMatrixVersions(catalog []string) map[string][]string {
	out := map[string][]string{}
	for _, version := range catalog {
		var family string
		switch {
		case java21GameReleasePattern.MatchString(version):
			family = switchMatrixLegacy
		case java25GameReleasePattern.MatchString(version):
			family = switchMatrixModern
		default:
			continue
		}
		if len(out[family]) < switchMatrixCatalogLimit && !slices.Contains(out[family], version) {
			out[family] = append(out[family], version)
		}
	}
	return out
}

func discoverSwitchMatrixVersions(ctx context.Context) (map[string][]string, string) {
	probe, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	raw, err := instanceRuntimes.command(probe, switchMatrixVersionHelper, "matrix-catalog")
	var catalog []string
	switch {
	case probe.Err() != nil:
		return nil, "publisher_catalog_timeout"
	case err != nil:
		return nil, "publisher_catalog_request_failed"
	case len(raw) > 256<<10 || instanceDecode(raw, &catalog) != nil || len(catalog) == 0:
		return nil, "publisher_catalog_response_invalid"
	}
	return switchMatrixVersions(catalog), ""
}

type switchMatrixReleaseKey struct{ Software, Version string }

// An empty diagnostic means the exact managed selection was verified.
func verifySwitchMatrixCrossplaySelection(ctx context.Context, software, version string, r *switchMatrixRuntime) string {
	probe, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	raw, err := instanceRuntimeCommand(probe, switchMatrixVersionHelper, "matrix-crossplay", software, version, r.Loader)
	if probe.Err() != nil {
		return "crossplay_probe_timeout"
	}
	if err != nil {
		return "crossplay_helper_failed"
	}
	return decodeSwitchMatrixCrossplayCheck(raw)
}

func decodeSwitchMatrixCrossplayCheck(raw []byte) string {
	var reply struct {
		Compatible *bool  `json:"compatible"`
		Reason     string `json:"reason"`
	}
	const invalid = "crossplay_response_invalid"
	if len(raw) > 256 || instanceDecode(raw, &reply) != nil || reply.Compatible == nil {
		return invalid
	}
	if *reply.Compatible {
		if reply.Reason == "" {
			return ""
		}
		return invalid
	}
	switch reply.Reason {
	case "crossplay_helper_failed", "crossplay_selection_invalid", "crossplay_loader_missing", "geyser_java_target_mismatch",
		"geyser_request_timeout", "geyser_request_failed", "geyser_response_invalid",
		"viaversion_request_timeout", "viaversion_request_failed", "viaversion_response_invalid",
		"viaversion_release_unverified", "viaversion_exact_release_unsupported", "crossplay_mod_artifacts_unverified", "crossplay_mod_request_timeout",
		"geyser_artifact_unverified", "floodgate_artifact_unverified":
		return reply.Reason
	default:
		return invalid
	}
}

func switchMatrixCrossplayPairReason(source, target string) string {
	unsupported := false
	for _, check := range []string{source, target} {
		switch check {
		case "":
		case "geyser_java_target_mismatch", "viaversion_exact_release_unsupported":
			unsupported = true
		default:
			return "managed_crossplay_stack_unverified"
		}
	}
	if unsupported {
		return "managed_crossplay_stack_unsupported"
	}
	return ""
}

func (c switchMatrixCase) canonicalID() string {
	mode := "java"
	if c.Crossplay {
		mode = "crossplay"
	}
	return strings.Join([]string{c.Source, c.Target, c.Action, c.Family, mode, c.SourceState}, ".")
}

func buildSwitchMatrixCases(ctx context.Context, versions map[string][]string, lookup func(context.Context, string, string) switchMatrixRelease,
	managed func(context.Context, string, string, *switchMatrixRuntime) string, selected ...string) []switchMatrixCase {
	software := []string{"bedrock"}
	for name := range javaServerCapabilityTable {
		software = append(software, name)
	}
	slices.Sort(software)
	cached := map[switchMatrixReleaseKey]switchMatrixRelease{}
	stacks := map[switchMatrixReleaseKey]string{}
	get := func(name, version string) switchMatrixRelease {
		key := switchMatrixReleaseKey{name, version}
		if result, ok := cached[key]; ok {
			return result
		}
		result := switchMatrixRelease{Outcome: switchMatrixReleaseUnverified, Verification: "publisher_discovery_deadline"}
		if ctx.Err() == nil {
			result = lookup(ctx, name, version)
		}
		cached[key] = result
		return result
	}
	stackCheck := func(name, version string, r *switchMatrixRuntime) string {
		key := switchMatrixReleaseKey{name, version}
		if diagnostic, known := stacks[key]; known {
			return diagnostic
		}
		diagnostic := "crossplay_discovery_deadline"
		if ctx.Err() == nil {
			diagnostic = managed(ctx, name, version, r)
		}
		stacks[key] = diagnostic
		return diagnostic
	}
	var cases []switchMatrixCase
	add := func(c switchMatrixCase) {
		c.ID = c.canonicalID()
		if len(selected) == 0 || slices.Contains(selected, c.ID) {
			cases = append(cases, c)
		}
	}
	requested := func(c switchMatrixCase, inMode bool) bool {
		if len(selected) == 0 {
			return true
		}
		for _, id := range selected {
			fields := strings.Split(id, ".")
			if len(fields) == 6 && fields[0] == c.Source && fields[1] == c.Target && fields[3] == c.Family &&
				(!inMode || (fields[4] == "crossplay") == c.Crossplay) {
				return true
			}
		}
		return false
	}
	for _, source := range software {
		for _, target := range software {
			if source == target {
				continue
			}
			base := switchMatrixCase{Source: source, Target: target}
			if !switchStructuralPath(source, target) {
				base.Disposition, base.Reason = matrixUnsupported, "structural_direction_unsupported"
				add(base)
				continue
			}
			for _, family := range []string{switchMatrixLegacy, switchMatrixModern} {
				pair := base
				pair.Family = family
				if !requested(pair, false) {
					continue
				}
				pair.Version, pair.SourceRelease, pair.TargetRelease, pair.ReleaseReason =
					selectSwitchMatrixPair(versions[family], source, target, get, nil)
				for _, crossplay := range []bool{false, true} {
					if crossplay && !javaServerCrossplay(source) {
						continue
					}
					selection := pair
					selection.Crossplay = crossplay
					if !requested(selection, true) {
						continue
					}
					platformBlocked := switchRejectsCrossplayPlatform(source, target, crossplay)
					modsBlocked := switchNeedsVerifiedJavaOnlyMods(source, target, crossplay)
					for _, action := range []string{"clone", "convert"} {
						c := selection
						c.Action, c.Crossplay = action, crossplay
						c.TargetCrossplay = crossplay && javaServerCrossplay(target)
						convertPolicyBlocked := action == "convert" && (switchWorldRisk(source, target) || platformBlocked || modsBlocked)
						if action == "clone" && platformBlocked && switchCrossplayPlatform(target) == "spigot" {
							// Exercise Java-only Clone from cross-play loaders to Paper-family targets.
							// Loader destinations exercise installing a new cross-play stack.
							c.TargetCrossplay = false
						}
						if crossplay && pair.ReleaseReason == "" && !convertPolicyBlocked {
							c.Version, c.SourceRelease, c.TargetRelease, c.ReleaseReason =
								selectSwitchMatrixPair(versions[family], source, target, get, func(version string, a, b *switchMatrixRuntime) string {
									from := stackCheck(source, version, a)
									to := ""
									if c.TargetCrossplay {
										to = stackCheck(target, version, b)
									}
									return switchMatrixCrossplayPairReason(from, to)
								})
							if c.ReleaseReason == "managed_crossplay_stack_unverified" || c.ReleaseReason == "managed_crossplay_stack_unsupported" {
								c.SourceCrossplayCheck = stacks[switchMatrixReleaseKey{source, c.Version}]
								if c.TargetCrossplay {
									c.TargetCrossplayCheck = stacks[switchMatrixReleaseKey{target, c.Version}]
								}
							}
						}
						c.Reason = c.ReleaseReason
						convert, clone := switchTransitionActions(source, target, true, true, nil)
						switch {
						case action == "convert" && !convert || action == "clone" && !clone:
							c.Disposition, c.Reason = matrixBlocked, "switch_world_policy"
						case action == "convert" && platformBlocked:
							c.Disposition, c.Reason = matrixBlocked, "switch_crossplay_platform_policy"
						case action == "convert" && modsBlocked:
							c.Disposition, c.Reason = matrixBlocked, "switch_crossplay_target_loader_policy"
						case c.Reason == "exact_release_unsupported" || c.Reason == "managed_crossplay_stack_unsupported":
							c.Disposition = matrixUnsupported
						case c.Reason == "exact_release_unavailable":
							c.Disposition = matrixBlocked
						case c.Reason != "":
							c.Disposition = matrixFailed
						}
						if c.Disposition == "" || len(selected) != 0 {
							c.SourceState = "running"
							add(c)
							c.SourceState = "stopped"
						}
						add(c)
					}
				}
			}
		}
	}
	return cases
}

func selectSwitchMatrixPair(versions []string, source, target string, get func(string, string) switchMatrixRelease,
	compatible func(string, *switchMatrixRuntime, *switchMatrixRuntime) string) (string, switchMatrixRelease, switchMatrixRelease, string) {
	unverified := switchMatrixRelease{Outcome: switchMatrixReleaseUnverified}
	if len(versions) == 0 {
		return "", unverified, unverified, "publisher_catalog_unverified"
	}
	var selected string
	var from, to switchMatrixRelease
	var reason string
	priority := -1
	for _, version := range versions {
		a, b := get(source, version), get(target, version)
		nextReason, nextPriority := "", 0
		switch {
		case a.Outcome == switchMatrixReleaseUnsupported || b.Outcome == switchMatrixReleaseUnsupported:
			nextReason = "exact_release_unsupported"
		case a.Outcome == switchMatrixReleaseUnavailable || b.Outcome == switchMatrixReleaseUnavailable:
			nextReason, nextPriority = "exact_release_unavailable", 2
		case a.Outcome != switchMatrixReleaseAvailable || b.Outcome != switchMatrixReleaseAvailable || a.Runtime == nil || b.Runtime == nil:
			nextReason, nextPriority = "exact_release_unverified", 3
		default:
			if compatible != nil {
				nextReason = compatible(version, a.Runtime, b.Runtime)
			}
			if nextReason == "" {
				return version, a, b, ""
			}
			nextPriority = 1
			if nextReason == "managed_crossplay_stack_unverified" {
				nextPriority = 4
			}
		}
		if nextPriority > priority {
			selected, from, to = version, a, b
			reason, priority = nextReason, nextPriority
		}
	}
	return selected, from, to, reason
}
