---
name: justvoxel-iso-review
description: Perform an independent read-only review and verification of completed JustVoxel installer and ISO changes by inspecting the real diff, approved scope, installer architecture, stable VM/HWS payload contract, workflow protection, relevant checks, upstream patch risks, and actual ISO/install evidence.
---

# JustVoxel ISO Review

Review the implementation that actually exists, not the Codex completion message.

Review is read-only by default.

## Review Process

1. Inspect branch, `git status`, diff/stat, and exact changed files.
2. Compare the changes with the approved scope.
3. Read affected installer configuration, callers, patch context, workflow logic, skills, and docs as needed.
4. Run only the checks appropriate to the changed area.
5. Separate source evidence from ISO, boot, install, and runtime evidence.
6. Report concrete findings. Do not modify code unless the human separately approves a fix.

## What to Look For

Prioritize:

- stale `:testing` or obsolete `:10` product references in active installer paths;
- wrong VM/HWS payload or edition selection;
- missing `main` guard or any automatic ISO workflow trigger;
- non-90-day artifact retention or missing checksum;
- stable-image signature verification weakening;
- installer-only behavior leaking into the installed appliance;
- Anaconda/bootc/image-builder architecture drift;
- patch drift against the pinned upstream WebUI source;
- storage or destructive-action regressions;
- login/bootstrap credential mistakes;
- networking profiles that fail after first boot;
- workflow permission mistakes;
- legacy/pre-v2 installer implementation accidentally reintroduced;
- unrelated cleanup or scope expansion;
- documentation or skills claiming behavior no longer implemented.

Protect known-working installer behavior. Do not restructure working paths merely for style.

## Evidence Ladder

Keep these stages explicit:

1. source/configuration review;
2. ISO successfully built;
3. ISO successfully booted;
4. Anaconda successfully started;
5. VM/HWS edition selection behaved correctly;
6. storage configuration succeeded;
7. selected stable payload installation succeeded;
8. installed system successfully booted;
9. installed system tracks the expected `:stable` product channel;
10. installed JustVoxel health/runtime verified.

A later stage must never be inferred from an earlier one.

If a stage was not exercised, say so.

## Findings

Keep findings specific:

- exact file/path and line/range when practical;
- what is wrong;
- why it matters;
- safest intended correction.

Do not invent findings to make the review look thorough.

If no meaningful finding exists, say so and identify any important stage that remains unverified.

Discovery is not permission to fix an out-of-scope issue.
