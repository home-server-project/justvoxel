# JustVoxel build model

JustVoxel follows the Home Server Project's shared Home Server Base 10 bootc composition model.

## Base

Both variants consume [Home Server Base 10](https://github.com/home-server-project/home-server-base-10) as their direct bootc parent.

Home Server Base 10 owns the shared AlmaLinux 10 Minimal Plus rootfs composition, including the standard system-configuration, persistent-journal, and generic-growfs fragments. AlmaLinux 10 remains the upstream Enterprise Linux source for the kernel and core operating-system packages.

JustVoxel does not maintain a second local Minimal Plus manifest or rootfs-builder path. CI resolves the Home Server Base `:stable` image to one exact digest, verifies that digest with Cosign, and supplies the same verified parent to both the VM and Bare Metal image builds.

The supported host-management model is bootc. No host-side rpm-ostree layering workflow is supported.

## Variants

The `Containerfile` has two final targets:

- `justvoxel-vm`
- `justvoxel-baremetal`

Both derive from the same `justvoxel-common` stage. Bare Metal adds only the physical-machine administration package delta.

## Rechunk limits

CI rechunks each completed image with:

- RPM chunk target: `127`
- OCI layer hard limit: `128`

This matches the Passive Black Box build pattern.

## Branches

`testing` builds on push, manual dispatch, and daily at 14:40 UTC. It publishes moving `:testing` tags plus immutable `testing-YYYYMMDD-<git-sha>` tags for both variants and never creates GitHub Releases. Testing immutable images older than 45 days are eligible for cleanup while at least seven recent tagged builds per variant are retained. `main` publishes stable `:10` images and is the only branch allowed to create GitHub Releases.

## Signing

Published images are signed with the Home Server Project Cosign key. The image installs scoped `containers/image` trust for its own GHCR repository.
