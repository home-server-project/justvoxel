# JustVoxel build model

JustVoxel follows the Home Server Project's proven AlmaLinux bootc composition pattern.

## Base

Both variants are composed from AlmaLinux 10 repositories using the minimal-plus content tier plus the standard system-configuration, persistent-journal, and generic-growfs fragments.

The root filesystem is built with `bootc-base-imagectl`. `rpm-ostree` exists only in the ephemeral builder stage as upstream build plumbing; it is not the JustVoxel host management model and no rpm-ostree layering workflow is supported.

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

`testing` publishes only development `:testing` images. `main` publishes stable `:10` plus immutable dated/commit tags and is the only branch allowed to create GitHub Releases.

## Signing

Published images are signed with the Home Server Project Cosign key. The image installs scoped `containers/image` trust for its own GHCR repository.
