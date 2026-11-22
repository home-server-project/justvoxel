---
name: justvoxel-iso-webui
description: Change or review the custom JustVoxel Anaconda WebUI layer, including upstream source pinning, patching, wizard composition, JustVoxel pages, branding, VM/HWS edition presentation, login presentation, storage UI, and Firefox/installer visual behavior.
---

# JustVoxel ISO WebUI

Keep Anaconda WebUI as the installer foundation. Customize only the pieces JustVoxel actually needs.

## Upstream First

The installer builds a pinned upstream Anaconda WebUI and applies narrow JustVoxel overrides.

Preserve this model:

- keep upstream source/version pins intentional and reviewable;
- do not vendor the entire upstream project;
- do not fork or reimplement Anaconda installer logic unnecessarily;
- keep distro/runtime integration supplied by packaged Anaconda/Cockpit where possible;
- replace only the compiled/source pieces required by the approved JustVoxel customization.

When changing the patch, inspect the pinned upstream source that the patch applies to.

Do not assume an upstream component/API still looks the same after changing pins.

## Wizard Scope

The WebUI should simplify installation, not become a second installer engine.

Use JustVoxel pages and wording for appliance-specific choices such as VM/HWS edition and login information.

Preserve the existing virtualization-based recommendation behavior.

Do not duplicate storage, bootloader, network, or account engines in React when Anaconda already provides those capabilities.

Keep unavailable editions explicitly unavailable. Never make a VM selection secretly install HWS or vice versa.

Payload/channel semantics belong to `justvoxel-iso-installer`. The final installer uses the VM/HWS `:stable` channels only.

## Storage UI

Preserve upstream storage validation, warnings, and editor behavior.

JustVoxel may expose a smaller set of scenarios, but those choices should invoke upstream storage mechanisms.

Automatic appliance installation may use the approved whole-disk path.

Manual partitioning should reuse the upstream storage editor and its confirmation, validation, and return flow.

Do not invent duplicate technical storage scenarios as JustVoxel radio options merely because upstream has them internally.

## Human Experience

Use plain installer language.

The user should understand:

- which JustVoxel edition is being installed;
- which installation method is selected;
- whether a disk may be erased;
- how they will log in after install;
- what will happen when they start installation.

Avoid Linux implementation jargon unless necessary for an advanced/manual storage screen.

Branding should remain restrained and readable. Do not sacrifice installer clarity for visual customization.

## Files to Inspect

Typical installer WebUI work touches:

- `installer/webui-overrides/justvoxel.patch`
- `installer/webui-overrides/src/`
- `installer/webui-overrides/README.md`
- `installer/branding.css`
- `installer/firefox-userChrome.css`
- installer branding/logo/Plymouth assets where relevant

Use `justvoxel-iso-installer` as a supporting skill when a UI change also changes real installer behavior or payload/storage semantics.
