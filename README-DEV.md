# JustVoxel developer guide

This repository contains the shared appliance source, build definitions, documentation, and final JustVoxel VM and HWS images. Development takes place on `testing`. Reviewed changes reach `main` after appliance validation.

For installation and a product overview, see the [JustVoxel README](README.md).

## What JustVoxel is

JustVoxel is a complete server appliance, not an RPM, a shell script, or a container bundle that is installed on top of an arbitrary existing Linux system.

The operating system, management layer, update model, storage safety rules, backup/recovery logic, and container runtime integration are built and versioned together. The goal is to give users a predictable system that can be installed, configured, operated, updated, and recovered without requiring deep knowledge of the technologies underneath it.

Under the hood, JustVoxel is built on [Home Server Base 10](https://github.com/home-server-project/home-server-base-10), which provides the shared AlmaLinux 10 Minimal Plus bootc foundation. AlmaLinux 10 remains the upstream Enterprise Linux source for the kernel and core operating-system packages. JustVoxel uses standard Linux components such as Podman, systemd, NetworkManager, firewalld, and SELinux, but normal users are not expected to manage those pieces directly.

For the technical design and the reasons behind it, see [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md).

## Minecraft workloads are separate from the OS

JustVoxel does **not** ship Minecraft server binaries, Mojang server software, a pre-created world, or pre-accepted Minecraft EULA state inside the operating-system image.

The JustVoxel image provides the appliance operating system, management tools,
runtime templates, and safety mechanisms. During first setup, the administrator
accepts the Minecraft EULA and JustVoxel creates the first independent Minecraft
server instance. More instances can be added later, and several instances can run
at the same time when host memory and ports allow it.

Current setup supports Paper, Purpur, Spigot, Fabric, NeoForge, Forge, Quilt,
Vanilla Java, and native Bedrock. Capability checks remain software-specific:
plugins, mods, managed Java-to-Bedrock cross-play, version selection, and safe
software changes are offered only where the implementation proves support.

Minecraft runtime maintenance is intentionally separate from JustVoxel
operating-system maintenance.

For the runtime boundary and implementation details, see [`docs/MINECRAFT_RUNTIME.md`](docs/MINECRAFT_RUNTIME.md).

## Who JustVoxel is for

The main audience is an advanced home user rather than a professional Linux administrator.

If you can install Windows or Linux yourself, create a VM, write an ISO to a USB drive, follow installation instructions, and understand basic ideas such as an IP address and a disk, JustVoxel is intended to handle the deeper appliance work for you.

Experienced administrators are not locked out. JustVoxel remains a normal immutable EL10 server built on Home Server Base 10 / AlmaLinux 10, and standard Linux administration tools remain available when deeper control or troubleshooting is wanted.

## Image composition

`Containerfile.dev` builds the Development showcase. `Containerfile` builds VM and HWS from a private common stage based on signed Home Server Base 10. Both products receive the current checkout’s WebUI and shared Management Agent. HWS adds physical-hardware support before each product receives its own identity and image trust configuration.

See [package composition](docs/PACKAGES.md) for the common dependencies and [the HWS package list](build_files/packages-hws.env) for the hardware delta.

## Operate the appliance with mjust

`mjust` is JustVoxel's built-in administration interface.

Run `mjust` with no arguments to open the interactive terminal interface and choose what you want to do. It is designed so common appliance operations can be completed without knowing the Linux commands underneath them.

The interface covers first setup, server selection, Minecraft configuration and
lifecycle control, players and whitelists, backups and restore, server migration,
storage, supported Minecraft and container updates, Minecraft Network, appliance
health/status, operating-system maintenance, UPS, system resources, validation,
and recovery tasks. Additional-server creation is currently provided by the
WebUI's **Add Minecraft Server** workflow.

The underlying safety checks remain active whether an operation is started from the menu or from a direct `mjust` command.

The interaction model is inspired by Universal Blue's `ujust` / `ugum` work in [ublue-os/packages](https://github.com/ublue-os/packages), while JustVoxel uses its own server-focused implementation.

For the complete interface and command reference, see [`docs/MJUST.md`](docs/MJUST.md).

## Web management

The WebUI source and the privileged Management Agent source are maintained in this repository and built/tested from the same appliance source commit. That source consolidation does not collapse the runtime security boundary: the browser-facing WebUI remains unprivileged and reaches approved privileged operations only through Management API v1 over the local Unix socket.

JustVoxel WebUI is intended for administration from a trusted local network. By default, Web management uses plain HTTP on TCP port `8099`, allowing direct access from the appliance LAN address without a self-signed certificate warning.

The default administrator is `voxel`. In the recommended/default **System account** authentication mode, the same real Linux `voxel` password is used by the WebUI, local console, and SSH password login when SSH password authentication is enabled. Authentication is performed through the AlmaLinux/RHEL PAM stack by the privileged JustVoxel Management Agent; the unprivileged WebUI does not keep a synchronized copy of that password.

An optional **Separate WebUI password** mode is available for administrators who intentionally want browser authentication to differ from the Linux/SSH password.

Because the default local WebUI does not use TLS, administrator credentials and sessions should only be used on a network you trust. Do not forward TCP port `8099` directly to the public Internet.

For the local-access, authentication, and security model, see [`docs/WEBUI.md`](docs/WEBUI.md).

**AI Agents & API Connections** provides a primary-administrator **AI & API Connections** workspace to manage independent AI agent and automation connections through the standard MCP and REST interfaces on the existing WebUI listener. Each connection has a live access level and instance scope; tokens and external OAuth bindings stay authoritative in the Management Agent. The local default remains unencrypted; use trusted HTTPS or private VPN access for protected remote connections. See [`docs/AI-API-CONNECTIONS.md`](docs/AI-API-CONNECTIONS.md).

## Storage, backups, and recovery

JustVoxel separates Minecraft data from backup storage so users can choose a layout that fits their machine or hypervisor.

The appliance can use supported local storage or network storage for backups, and Minecraft data can remain on the system filesystem or be migrated to supported local storage later.

Storage operations are guarded with device validation and exact typed confirmations for destructive actions. Backup and restore workflows use their own validation and safety checks rather than relying on the user to assemble commands manually.

See:

- [`docs/STORAGE.md`](docs/STORAGE.md) for storage and migration
- [`docs/RESTORE.md`](docs/RESTORE.md) for selected-instance full-data recovery

## Updates

JustVoxel keeps operating-system updates and Minecraft workload updates separate.

The appliance operating system is updated through bootc and can be managed through `mjust` system-management commands.

JustVoxel disables rpm-ostree package layering by default to keep deployed systems aligned with the tested appliance image. This does not affect system updates. For more details, see [rpm-ostree package layering](docs/SYSTEM.md#rpm-ostree-package-layering).

Each Minecraft workload has its own supported update flow, backup safeguards,
version policy, and rollback handling.

See [`docs/SYSTEM.md`](docs/SYSTEM.md) for operating-system maintenance and [`docs/MJUST.md`](docs/MJUST.md) for the Minecraft update workflow.

## Documentation

Start with the document that matches what you are trying to do:

- [Installation guide](docs/INSTALLER.md) and [installer design](docs/INSTALLER-DESIGN.md) — install JustVoxel on a VM or physical machine
- [`docs/MJUST.md`](docs/MJUST.md) — operate the appliance
- [`docs/MINECRAFT_RUNTIME.md`](docs/MINECRAFT_RUNTIME.md) — current multi-instance runtime design and implementation history
- [`docs/server-software.md`](docs/server-software.md) — supported Java and Bedrock server software and capability rules
- [`docs/plugin-workspace.md`](docs/plugin-workspace.md) — plugin discovery, installation, lifecycle, add-ons, and provider configuration
- [`docs/minecraft-network.md`](docs/minecraft-network.md) — optional Velocity entrance, lobby, membership, and network whitelist
- [`docs/STATUS.md`](docs/STATUS.md) — understand the appliance health dashboard
- [`docs/SYSTEM.md`](docs/SYSTEM.md) — OS status, updates, resources, reboot, poweroff, and firmware controls
- [`docs/NETWORKING.md`](docs/NETWORKING.md) — LAN management and supported remote-access providers
- [`docs/UPS.md`](docs/UPS.md) — conditional NUT/UPS monitoring, protection, and sharing
- [`docs/STORAGE.md`](docs/STORAGE.md) — storage choices, provisioning, mounts, and migration
- [`docs/RESTORE.md`](docs/RESTORE.md) — selected-instance full-data recovery
- [`docs/MIGRATION.md`](docs/MIGRATION.md) — import, export, and server migration
- [`docs/START_OVER.md`](docs/START_OVER.md) — retire all managed servers while preserving data and backups
- [`docs/AI-API-CONNECTIONS.md`](docs/AI-API-CONNECTIONS.md) — MCP/REST connections, tokens, OAuth, access levels and revocation
- [`docs/WEBUI.md`](docs/WEBUI.md) — WebUI authentication, access, and security model
- [`docs/MANAGEMENT.md`](docs/MANAGEMENT.md) — management layers and native Linux administration
- [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) — why JustVoxel is built as an immutable appliance
- [`docs/PACKAGES.md`](docs/PACKAGES.md) — package groups included in the common image and their purpose
- [`docs/BUILD.md`](docs/BUILD.md) — image composition, signing, CI, branches, and release mechanics
- [`docs/ROADMAP.md`](docs/ROADMAP.md) — current priorities, future features, and project non-goals

## Branches and builds

Pull requests into `testing` validate source and build Development locally in CI. Dispatch **Development** to publish the full WebUI showcase or **Testing** to build VM and HWS from the current checkout and Home Server Base stable. Both workflows share the verified testing-channel Management Agent.

The protected `main` branch is reserved for stable promotion. Development takes place on `testing`. After validation, promote only by pull request from this repository’s `testing` branch to `main`. The required `promotion` check rejects other branches and forks. The **Protect main** ruleset blocks force pushes and deletion and has no bypass actors. Stable builds compile both products from the promoted revision. The [build guide](docs/BUILD.md) covers tags, local development, and release triggers.

## License

Apache-2.0. See [LICENSE](LICENSE).

Minecraft, Mojang software, supported server projects, plugins, and other
third-party components retain their own licenses and distribution terms.
JustVoxel does not embed upstream Minecraft server binaries or Mojang server
software in its bootc images.
