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

Publishing workflows resolve the moving stable tag and verify its exact digest with Cosign. The same digest supplies Management Agent validation and appliance assembly.

The supported host-management model is bootc. No host-side rpm-ostree layering workflow is supported.

## Image composition

`Containerfile.dev` builds Development (`justvoxel-dev`), the full WebUI and integration showcase. VM and HWS build independently of Development. Their `Containerfile` runs the shared appliance build once in an internal Home Server Base stage with the current checkout’s WebUI binary, release metadata and verified Management Agent. HWS adds the hardware packages and configuration before finalization.

## Testing

PRs targeting `testing` run source checks and compile the Management Agent from source for a development image build. PR validation uses read-only repository access.

Dispatch **Development** from `testing` to publish `ghcr.io/home-server-project/justvoxel-dev:testing`. It reuses the signed `ghcr.io/home-server-project/justvoxel-management-agent:testing` package when the inputs match. Otherwise it rebuilds, signs, and verifies that package.

Dispatch **Testing** from `testing` to build both products from the checked-out source. Both testing-channel workflows reuse the same internal Management Agent package after verifying the input identity, base digest, API version and exact package digest signature. It publishes `ghcr.io/home-server-project/justvoxel-vm:testing` and `ghcr.io/home-server-project/justvoxel-hws:testing`. Testing builds also publish immutable tags with the build date, short commit, run ID, and attempt.

## Stable releases

After product validation, open a pull request from this repository’s `testing` branch to `main`. The promotion check verifies both the source branch and repository. Direct-push prevention requires GitHub repository rules protecting `main` and requiring the promotion check.

A push to `main`, a manual dispatch on `main`, or the Saturday 08:00 UTC schedule builds from the validated main revision. Stable uses its own internal verified `ghcr.io/home-server-project/justvoxel-management-agent:stable` package, rebuilding it when inputs change. Development has no stable tag. The stable appliance repositories are

- `ghcr.io/home-server-project/justvoxel-vm:stable`
- `ghcr.io/home-server-project/justvoxel-hws:stable`

CI rechunks Development and both products with a target of 127 layers and enforces a 128-layer limit. Published image digests are signed and verified with Cosign.

## Local development

The human-run `jv-build.sh` builds and validates `justvoxel-dev:local` on the development VM. It publishes to `192.168.125.100:5000/justvoxel-dev` with an immutable `local-YYYYMMDD-HHMMSS-<short-commit>` tag and moves `local` to the signed digest. The local registry cleanup retains the newest two immutable builds and protects digests referenced by retained tags.
