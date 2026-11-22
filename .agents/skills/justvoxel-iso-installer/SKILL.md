---
name: justvoxel-iso-installer
description: Change or review the unified JustVoxel installer architecture, installer environment, image-builder/bootc integration, Anaconda configuration, stable VM/HWS payload selection, storage defaults, networking handoff, accounts, and installer behavior outside the custom WebUI itself.
---

# JustVoxel ISO Installer

Keep the installer small, upstream-based, and separate from the installed JustVoxel appliance.

## Current Direction

Use the supported installer path:

image-builder -> bootc-generic-iso -> Anaconda WebUI -> selected JustVoxel bootc payload.

Installer source lives under `installer/` in the unified JustVoxel repository.

One installer ISO supports two editions:

- JustVoxel VM
- JustVoxel HWS

The ISO workflow is manually dispatched from `main` only.

Do not reintroduce the legacy pre-v2 installer implementation or create parallel installer pipelines without explicit human approval.

## Responsibility Boundary

The installer exists to place the selected JustVoxel image onto the target disk.

Do not:

- turn the installer into another OS-image source;
- move installer-only packages into the JustVoxel VM/HWS images merely because the installer needs them;
- create a custom partitioning engine, bootloader installer, network stack, or general-purpose installer framework when Anaconda already owns that responsibility;
- add Minecraft server binaries, worlds, or EULA state to the installer.

Prefer upstream mechanisms and make JustVoxel-specific changes only where the appliance actually needs them.

## Editions

The supported editions are:

- JustVoxel VM
- JustVoxel HWS

Preserve the existing virtualization detection/recommendation behavior.

An unavailable edition must be shown as unavailable. Never silently map one edition selection to another payload.

Keep edition image references centralized rather than scattering them through unrelated code.

## Stable Payload Contract

The final installer uses only:

- `ghcr.io/home-server-project/justvoxel-vm:stable`
- `ghcr.io/home-server-project/justvoxel-hws:stable`

Do not use `:testing`, obsolete generation tags such as `:10`, or an arbitrary user-selected product tag in the final ISO path.

The online installer must leave the installed system tracking the selected `:stable` channel.

A registry transport prefix may be required for the Anaconda source reference, while the installed bootc target reference remains a normal registry image reference. Preserve that distinction.

Build-time digest/signature checks may prove the current stable payload identity. Do not replace the product channel with a custom digest-resolution or update-selection layer.

Home Server Packages keep their own channel convention: stable packages use `:stable`.

## Storage Defaults

Treat current source as authoritative and verify it before changing values.

The current appliance-oriented defaults include:

- GPT;
- 128 MiB EFI System Partition;
- 896 MiB `/boot`;
- remaining supported system space for JustVoxel root;
- no disk swap;
- installed JustVoxel owns zram policy.

Do not inflate these to generic workstation/server defaults without a demonstrated JustVoxel requirement.

Do not reduce them without validating bootloader, kernel/initramfs, update, and rollback needs.

Prefer Anaconda storage capabilities when they work correctly.

A reliable automatic appliance layout is more important than recreating every general-purpose storage option.

Any destructive storage action must remain obvious to the user.

## Networking

Installer networking exists to complete installation and leave a valid normal machine-specific network configuration.

Prefer Anaconda and NetworkManager behavior.

Do not add permanent appliance networking policy that belongs in the JustVoxel product.

If installer-generated profiles need narrow normalization for reliable first boot, keep it targeted and validate it against the current installer stack.

## Accounts and Access

Keep installer account behavior explicit.

Bootstrap credentials must be clearly treated as installer/first-access behavior.

Prefer installer-native account handling where practical.

Never embed private SSH keys, leak credentials, or silently create undocumented privileged accounts.

## Source of Truth

For current behavior inspect:

- `installer/Containerfile`
- `installer/90-justvoxel.conf`
- `installer/interactive-defaults.ks`
- `installer/iso.yaml`
- `installer/anaconda-overrides/`
- `installer/webui-overrides/`
- `.github/workflows/iso.yml`

Use `justvoxel-iso-webui` when the task concerns the custom Anaconda WebUI itself.
Use `justvoxel-iso-ci` when the task concerns ISO Actions/build behavior.
