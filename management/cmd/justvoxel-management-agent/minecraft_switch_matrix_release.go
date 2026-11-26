package main

import (
	"context"
	"time"
)

type switchMatrixReleaseOutcome string

const (
	switchMatrixReleaseAvailable   switchMatrixReleaseOutcome = "AVAILABLE"
	switchMatrixReleaseUnsupported switchMatrixReleaseOutcome = "UNSUPPORTED"
	switchMatrixReleaseUnverified  switchMatrixReleaseOutcome = "UNVERIFIED"
)

type switchMatrixRuntime struct {
	Game            string `json:"game"`
	Loader          string `json:"loader"`
	Image           string `json:"image"`
	PufferfishBuild int    `json:"pufferfish_build,omitempty"`
	Build           string `json:"build,omitempty"`
	Digest          string `json:"digest,omitempty"`
	ImageDigest     string `json:"image_digest,omitempty"`
	Channel         string `json:"channel,omitempty"`
	Reason          string `json:"reason,omitempty"`
}

type switchMatrixRelease struct {
	Outcome switchMatrixReleaseOutcome `json:"outcome"`
	Runtime *switchMatrixRuntime       `json:"runtime,omitempty"`
}

func resolveSwitchMatrixRelease(ctx context.Context, software, version string) switchMatrixRelease {
	unverified := switchMatrixRelease{Outcome: switchMatrixReleaseUnverified}
	if _, ok := javaServerSoftware(software); !ok || !instanceRuntimeVersionPattern.MatchString(version) || version == "LATEST" {
		return unverified
	}
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	raw, err := instanceRuntimeCommand(ctx, switchMatrixVersionHelper, "matrix-release", software, version)
	if err != nil {
		return unverified
	}
	return decodeSwitchMatrixRelease(software, version, raw)
}

func decodeSwitchMatrixRelease(software, version string, raw []byte) switchMatrixRelease {
	unverified := switchMatrixRelease{Outcome: switchMatrixReleaseUnverified}
	if _, ok := javaServerSoftware(software); !ok ||
		!java21GameReleasePattern.MatchString(version) && !java25GameReleasePattern.MatchString(version) {
		return unverified
	}
	var result switchMatrixRelease
	if len(raw) > 1024 || instanceDecode(raw, &result) != nil {
		return unverified
	}
	switch result.Outcome {
	case switchMatrixReleaseUnverified:
		return unverified
	case switchMatrixReleaseUnsupported:
		if result.Runtime == nil {
			return result
		}
	case switchMatrixReleaseAvailable:
		r := result.Runtime
		if r != nil && r.Game == version && (r.Image == "java21" || r.Image == "java25") &&
			r.ImageDigest == "" && r.Reason == "" && switchMatrixPublisherIdentity(software, r) &&
			validPufferfishBuild(software, r.PufferfishBuild) &&
			validateJavaServerSelection(software, "pinned", r.Game, r.Loader, r.Image, false) == nil {
			return result
		}
	}
	return unverified
}

func switchMatrixPublisherIdentity(software string, r *switchMatrixRuntime) bool {
	if software == "leaf" {
		return switchMatrixBuildPattern.MatchString(r.Build) && operationFingerprintPattern.MatchString(r.Digest) &&
			(r.Channel == "PRODUCTION" || r.Channel == "TESTING" || r.Channel == "UNSUPPORTED")
	}
	return r.Build == "" && r.Digest == "" && r.Channel == ""
}

func sameSwitchMatrixRuntime(current, reviewed *switchMatrixRuntime) bool {
	if current == nil || reviewed == nil {
		return false
	}
	expected := *reviewed
	expected.ImageDigest = ""
	return *current == expected
}
