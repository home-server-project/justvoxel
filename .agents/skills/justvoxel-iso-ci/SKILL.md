---
name: justvoxel-iso-ci
description: Change, review, or diagnose the unified JustVoxel ISO GitHub Actions workflow, stable VM/HWS payload checks, image-builder execution, installer image build, permissions, pinned actions, checksums, artifact retention, manual/main-only triggering, and ISO CI failures.
---

# JustVoxel ISO CI

Keep installer CI narrow, deterministic, and truthful about what it proves.

## Investigation

- Inspect the exact workflow file and exact run/commit SHA that executed.
- Identify the first real failing step before editing workflow logic.
- Distinguish stable-payload verification, installer-image build, image-builder, ISO output, checksum, and artifact-upload failures.
- Do not treat a canceled, skipped, queued, or failed job as green.

## Final ISO Contract

The supported ISO workflow is:

- `.github/workflows/iso.yml`
- visible workflow name `ISO`
- `workflow_dispatch` only
- valid only when `github.ref == refs/heads/main`
- one ISO supporting VM and HWS
- VM payload `ghcr.io/home-server-project/justvoxel-vm:stable`
- HWS payload `ghcr.io/home-server-project/justvoxel-hws:stable`
- no testing/development/custom product-channel input
- ISO and SHA256 checksum uploaded together
- Actions artifact retention `90` days
- no GitHub Release or SourceForge publication unless explicitly approved later

A manual dispatch against another ref must fail rather than produce installation media.

## Workflow Rules

- Pin third-party Actions to full commit SHAs with useful version comments.
- Grant only the permissions required by the workflow/jobs.
- Preserve stable VM/HWS signature verification with the repository `cosign.pub`.
- Keep artifact naming predictable and non-testing.
- Preserve `compression-level: 0` for ISO artifacts unless a demonstrated need justifies changing it.
- Reuse existing working installer/build logic rather than creating parallel pipelines.

## Main Protection

Installer source changes develop on `testing` or another explicitly approved development branch.

Stable installer source reaches `main` through the protected:

`testing -> pull request -> main`

promotion path.

Do not patch `main` directly for ISO troubleshooting.

Do not bypass, disable, weaken, or rewrite the `Protect main` ruleset or required `promotion` check as part of CI repair.

Repository-protection changes require explicit human approval for that exact settings change.

## Evidence Boundaries

CI can prove that:

- the selected stable payload checks passed;
- the configured installer image built;
- image-builder produced the expected ISO;
- the checksum was generated;
- the Actions artifact was uploaded.

CI alone does not prove:

- the ISO booted;
- Anaconda started correctly;
- VM/HWS selection worked at runtime;
- storage behaved correctly;
- payload installation completed;
- the installed system booted;
- the installed system actually tracks the expected stable channel;
- JustVoxel runtime worked.

Keep those claims for the corresponding real install-test stages.
