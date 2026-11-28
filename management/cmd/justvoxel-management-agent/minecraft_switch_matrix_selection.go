//go:build justvoxel_dev

package main

import (
	"errors"
	"slices"
	"strings"
)

const maxSwitchMatrixSelection = 16

func validateSwitchMatrixSelection(ids []string) error {
	if len(ids) == 0 || len(ids) > maxSwitchMatrixSelection {
		return errors.New("Select 1 to 16 canonical Switch matrix cases.")
	}
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		if len(id) > 96 {
			return errors.New("Invalid canonical Switch matrix case ID.")
		}
		fields := strings.Split(id, ".")
		if len(fields) != 6 {
			return errors.New("Invalid canonical Switch matrix case ID.")
		}
		source, target, action, family, mode, state := fields[0], fields[1], fields[2], fields[3], fields[4], fields[5]
		if _, ok := javaServerSoftware(source); !ok {
			return errors.New("Invalid canonical Switch matrix source software.")
		}
		if _, ok := javaServerSoftware(target); !ok || source == target {
			return errors.New("Invalid canonical Switch matrix target software.")
		}
		if !slices.Contains([]string{"clone", "convert"}, action) ||
			!slices.Contains([]string{switchMatrixLegacy, switchMatrixModern}, family) ||
			!slices.Contains([]string{"java", "crossplay"}, mode) ||
			!slices.Contains([]string{"running", "stopped"}, state) {
			return errors.New("Invalid canonical Switch matrix case ID.")
		}
		if seen[id] {
			return errors.New("Duplicate Switch matrix case ID.")
		}
		seen[id] = true
	}
	return nil
}

func orderTargetedSwitchMatrixCases(ids []string, cases []switchMatrixCase) []switchMatrixCase {
	found := make(map[string]switchMatrixCase, len(cases))
	for _, c := range cases {
		found[c.ID] = c
	}
	ordered := make([]switchMatrixCase, 0, len(ids))
	for _, id := range ids {
		c, ok := found[id]
		if !ok {
			fields := strings.Split(id, ".")
			c = switchMatrixCase{ID: id, Source: fields[0], Target: fields[1], Action: fields[2], Family: fields[3],
				Crossplay: fields[4] == "crossplay", SourceState: fields[5],
				Disposition: matrixUnsupported, Reason: "canonical_case_unavailable"}
		}
		ordered = append(ordered, c)
	}
	return ordered
}
