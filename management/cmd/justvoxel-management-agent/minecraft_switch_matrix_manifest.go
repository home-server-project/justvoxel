package main

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"time"
)

const (
	switchMatrixLegacy        = "pre_26_1"
	switchMatrixModern        = "26_1_plus"
	switchMatrixCatalogLimit  = 8 // per layout family
	switchMatrixVersionHelper = "/usr/libexec/justvoxel/mjust/admin-version-status-json"
)

type switchMatrixCase struct {
	ID              string              `json:"id"`
	Source          string              `json:"source"`
	Target          string              `json:"target"`
	Family          string              `json:"world_layout_family,omitempty"`
	Version         string              `json:"minecraft_version,omitempty"`
	Action          string              `json:"action,omitempty"`
	Crossplay       bool                `json:"source_crossplay"`
	TargetCrossplay bool                `json:"target_crossplay"`
	SourceState     string              `json:"original_state,omitempty"`
	SourceRelease   switchMatrixRelease `json:"source_release,omitzero"`
	TargetRelease   switchMatrixRelease `json:"target_release,omitzero"`
	Disposition     string              `json:"disposition,omitempty"`
	Reason          string              `json:"reason,omitempty"`
	ReleaseReason   string              `json:"release_reason,omitempty"`
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

func verifySwitchMatrixCrossplaySelection(ctx context.Context, software, version string, r *switchMatrixRuntime) bool {
	probe, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	raw, err := instanceRuntimeCommand(probe, switchMatrixVersionHelper, "--server-type", software, "pinned", version, "yes")
	var selection adminVersionStatus
	return err == nil && len(raw) <= 256<<10 && json.Unmarshal(raw, &selection) == nil &&
		selection.SelectedCandidate == version && selection.ServerType == software &&
		selection.LoaderVersion == r.Loader && selection.PufferfishBuild == r.PufferfishBuild &&
		selection.ImageTag == r.Image && selection.ServerSupported &&
		selection.CrossplayEnabled && selection.CrossplayCompatible
}

func buildSwitchMatrixCases(ctx context.Context, versions map[string][]string, lookup func(context.Context, string, string) switchMatrixRelease,
	managed func(context.Context, string, string, *switchMatrixRuntime) bool) []switchMatrixCase {
	software := []string{"bedrock"}
	for name := range javaServerCapabilityTable {
		software = append(software, name)
	}
	slices.Sort(software)
	cached := map[switchMatrixReleaseKey]switchMatrixRelease{}
	stacks := map[switchMatrixReleaseKey]bool{}
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
	stackOK := func(name, version string, r *switchMatrixRuntime) bool {
		key := switchMatrixReleaseKey{name, version}
		if ok, known := stacks[key]; known {
			return ok
		}
		ok := ctx.Err() == nil && managed(ctx, name, version, r)
		stacks[key] = ok
		return ok
	}
	var cases []switchMatrixCase
	add := func(c switchMatrixCase) {
		mode := "java"
		if c.Crossplay {
			mode = "crossplay"
		}
		c.ID = strings.Join([]string{c.Source, c.Target, c.Action, c.Family, mode, c.SourceState}, ".")
		cases = append(cases, c)
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
				pair.Version, pair.SourceRelease, pair.TargetRelease, pair.ReleaseReason =
					selectSwitchMatrixPair(versions[family], source, target, get, nil)
				for _, crossplay := range []bool{false, true} {
					if crossplay && !javaServerCrossplay(source) {
						continue
					}
					selection := pair
					if crossplay && pair.ReleaseReason == "" && !switchRejectsCrossplayPlatform(source, target, true) {
						selection.Version, selection.SourceRelease, selection.TargetRelease, selection.ReleaseReason =
							selectSwitchMatrixPair(versions[family], source, target, get, func(version string, a, b *switchMatrixRuntime) bool {
								return stackOK(source, version, a) && (!javaServerCrossplay(target) || stackOK(target, version, b))
							})
					}
					for _, action := range []string{"clone", "convert"} {
						c := selection
						c.Action, c.Crossplay = action, crossplay
						c.TargetCrossplay = crossplay && javaServerCrossplay(target)
						c.Reason = c.ReleaseReason
						convert, clone := switchTransitionActions(source, target, true, true, nil)
						switch {
						case action == "convert" && !convert || action == "clone" && !clone:
							c.Disposition, c.Reason = matrixBlocked, "switch_world_policy"
						case switchRejectsCrossplayPlatform(source, target, crossplay):
							c.Disposition, c.Reason = matrixBlocked, "switch_crossplay_platform_policy"
						case c.Reason == "exact_release_unsupported":
							c.Disposition = matrixUnsupported
						case c.Reason != "":
							c.Disposition = matrixFailed
						default:
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
	compatible func(string, *switchMatrixRuntime, *switchMatrixRuntime) bool) (string, switchMatrixRelease, switchMatrixRelease, string) {
	unverified := switchMatrixRelease{Outcome: switchMatrixReleaseUnverified}
	if len(versions) == 0 {
		return "", unverified, unverified, "publisher_catalog_unverified"
	}
	var selected string
	var from, to switchMatrixRelease
	var stackVersion string
	var stackFrom, stackTo switchMatrixRelease
	reason := "exact_release_unsupported"
	for _, version := range versions {
		a, b := get(source, version), get(target, version)
		if a.Outcome == switchMatrixReleaseAvailable && b.Outcome == switchMatrixReleaseAvailable {
			if compatible == nil || compatible(version, a.Runtime, b.Runtime) {
				return version, a, b, ""
			}
			if stackVersion == "" {
				stackVersion, stackFrom, stackTo = version, a, b
			}
			continue
		}
		if selected == "" || reason == "exact_release_unsupported" &&
			(a.Outcome == switchMatrixReleaseUnverified || b.Outcome == switchMatrixReleaseUnverified) {
			selected, from, to = version, a, b
			if a.Outcome == switchMatrixReleaseUnverified || b.Outcome == switchMatrixReleaseUnverified {
				reason = "exact_release_unverified"
			}
		}
	}
	if stackVersion != "" {
		return stackVersion, stackFrom, stackTo, "managed_crossplay_stack_unverified"
	}
	return selected, from, to, reason
}
