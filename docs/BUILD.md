# JustVoxel build model

JustVoxel follows the Home Server Project's shared Home Server Base 10 bootc composition model.

## Go build toolchain

The root `.go-version` selects one exact stable Go patch release for local builds,
standalone WebUI builds, Management Agent builds, and GitHub Actions. Update that
file to change the compiler; the `go 1.25.0` module directives remain the minimum
module requirement.

Development hosts need a modern bootstrap `go` command with native toolchain
switching support. The distro package does not need to match `.go-version`.
Build scripts override inherited `GOTOOLCHAIN` for their process, verify the
selected compiler, and let Go download/cache it when necessary. They do not
modify global Go configuration. Toolchain downloads require network access and
a writable Go cache.

The Management Agent uses AlmaLinux's `golang` package as its bootstrap inside
Home Server Base, then builds and validates through
`build_files/build-management-agent.sh`. Local and testing CI Management Agent
cache identities include `.go-version`, so changing it invalidates cached binaries.

## Parent image

JustVoxel consumes [Home Server Base 10](https://github.com/home-server-project/home-server-base-10) directly from the signed moving channel:

```text
ghcr.io/home-server-project/home-server-base-10:stable
```

The workflow verifies the current `:stable` signature with Cosign and builds from that tag directly. The parent image is not maintained as a manually pinned digest in this repository.

The supported host-management model is bootc. No host-side rpm-ostree layering workflow is supported.

## Image composition

`Containerfile.dev` assembles the shared runtime, WebUI, Management Agent, and VM guest tools. Its runtime variant marker is `justvoxel-vm`. The final `Containerfile` builds JustVoxel VM from that baseline and adds the packages in `build_files/packages-hws.env` for JustVoxel HWS. HWS has its own variant marker, hardware checks, and trust policy.

## Testing

PRs targeting `testing` run source checks and compile the Management Agent from source for a development image build. PR validation uses read-only repository access.

Dispatch **Development testing** from `testing` to publish `ghcr.io/home-server-project/justvoxel-dev:testing`. It reuses the signed `ghcr.io/home-server-project/justvoxel-management-agent:testing` package when the inputs match. Otherwise it rebuilds, signs, and verifies that package.

Dispatch **Product testing** from `testing` to verify the development image and build both products from the same verified digest. It publishes `ghcr.io/home-server-project/justvoxel-vm:testing` and `ghcr.io/home-server-project/justvoxel-hws:testing`. Testing builds also publish immutable tags with the build date, short commit, run ID, and attempt.

## Stable releases

After product validation, open a pull request from this repository’s `testing` branch to `main`. The promotion check verifies both the source branch and repository. Direct-push prevention requires GitHub repository rules protecting `main` and requiring the promotion check.

A push to `main`, a manual dispatch on `main`, or the Saturday 08:00 UTC schedule builds from the validated main revision. The workflow compiles the Management Agent directly from source and assembles the common image locally as `localhost/justvoxel-dev:build`. Both final builds consume that local image. Stable publication and signing are limited to:

- `ghcr.io/home-server-project/justvoxel-vm:stable`
- `ghcr.io/home-server-project/justvoxel-hws:stable`

CI rechunks the common image and the final HWS image with a target of 127 layers and enforces a 128-layer limit. Published image digests are signed and verified with Cosign.

## Local development

The human-run `jv-build.sh` builds and validates `justvoxel-dev:local` on the development VM. It publishes to `192.168.125.100:5000/justvoxel-dev` with an immutable `local-YYYYMMDD-HHMMSS-<short-commit>` tag and moves `local` to the signed digest. The local registry cleanup retains the newest two immutable builds and protects digests referenced by retained tags.
