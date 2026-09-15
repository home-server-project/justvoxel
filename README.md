# JustVoxel

JustVoxel is a purpose-built immutable server appliance for running and managing a Minecraft server workload.

It is designed for people who are comfortable installing an operating system and following normal computer instructions, but who do not want to become Linux, container, systemd, firewall, SELinux, or Minecraft-server administrators just to run a reliable family or small-community server.

> **Development status:** active implementation is on the `testing` branch. The project is still being validated in VMs and on physical hardware before stable promotion.

## What JustVoxel is

JustVoxel is a complete server appliance, not an RPM, a shell script, or a container bundle that is installed on top of an arbitrary existing Linux system.

The operating system, management layer, update model, storage safety rules, backup/recovery logic, and container runtime integration are built and versioned together. The goal is to give users a predictable system that can be installed, configured, operated, updated, and recovered without requiring deep knowledge of the technologies underneath it.

Under the hood, JustVoxel is built on AlmaLinux 10 and bootc. It uses standard Linux components such as Podman, systemd, NetworkManager, firewalld, and SELinux, but normal users are not expected to manage those pieces directly.

For the technical design and the reasons behind it, see [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md).

## Minecraft is a separate workload

JustVoxel does **not** ship Minecraft server binaries, Mojang server software, a pre-created world, or pre-accepted Minecraft EULA state inside the operating-system image.

The JustVoxel image provides the appliance operating system, management tools, runtime templates, and safety mechanisms. During first setup, the administrator accepts the Minecraft EULA and JustVoxel creates the active configuration for a separate containerized Paper-based server workload.

Minecraft/Paper maintenance is intentionally separate from JustVoxel operating-system maintenance.

For the runtime boundary and implementation details, see [`docs/MINECRAFT_RUNTIME.md`](docs/MINECRAFT_RUNTIME.md).

## Who JustVoxel is for

The main audience is an advanced home user rather than a professional Linux administrator.

If you can install Windows or Linux yourself, create a VM, write an ISO to a USB drive, follow installation instructions, and understand basic ideas such as an IP address and a disk, JustVoxel is intended to handle the deeper appliance work for you.

Experienced administrators are not locked out. JustVoxel remains a normal immutable AlmaLinux server and standard Linux administration tools remain available when deeper control or troubleshooting is wanted.

## Install JustVoxel

For a fresh installation, use the dedicated [JustVoxel ISO Builder](https://github.com/home-server-project/justvoxel-iso).

The ISO project owns the installation side of JustVoxel, including:

- VM versus Bare Metal installation
- CPU, memory, and disk guidance
- installer-media creation
- installation-disk safety
- SSH and first-access guidance
- installer options
- first boot

The normal user path is to install JustVoxel from its installer media rather than manually rebasing another operating system to the JustVoxel image.

Advanced bootc users can still work directly with the published images, but that is not the primary installation path documented for normal users.

## VM or Bare Metal

JustVoxel is built in two variants from the same appliance core.

**JustVoxel VM** is intended for KVM/libvirt, Proxmox, VMware, Hyper-V, VirtualBox, and similar hypervisors.

**JustVoxel Bare Metal** is intended for installation directly on physical hardware and adds physical-machine administration support that does not make sense inside a VM.

The normal JustVoxel and Minecraft management experience is the same on both.

For the exact technical differences, see [`docs/VARIANTS.md`](docs/VARIANTS.md).

## Operate the appliance with mjust

`mjust` is JustVoxel's built-in administration interface.

Run `mjust` with no arguments to open the interactive terminal interface and choose what you want to do. It is designed so common appliance operations can be completed without knowing the Linux commands underneath them.

The interface covers first setup, Minecraft configuration and service control, players and whitelist management, backups and restore, storage, Minecraft updates, appliance health/status, operating-system maintenance, system resources, validation, and other appliance tasks.

The underlying safety checks remain active whether an operation is started from the menu or from a direct `mjust` command.

The interaction model is inspired by Universal Blue's `ujust` / `ugum` work in [ublue-os/packages](https://github.com/ublue-os/packages), while JustVoxel uses its own server-focused implementation.

For the complete interface and command reference, see [`docs/MJUST.md`](docs/MJUST.md).

## Web management

JustVoxel WebUI is intended for administration from a trusted local network. By default, Web management uses plain HTTP on TCP port `8099`, allowing direct access from the appliance LAN address without a self-signed certificate warning.

Because the default local WebUI does not use TLS, administrator credentials and sessions should only be used on a network you trust. Do not forward TCP port `8099` directly to the public Internet.

For the local-access and security model, see [`docs/WEBUI.md`](docs/WEBUI.md).

## Storage, backups, and recovery

JustVoxel separates Minecraft data from backup storage so users can choose a layout that fits their machine or hypervisor.

The appliance can use supported local storage or network storage for backups, and Minecraft data can remain on the system filesystem or be migrated to supported local storage later.

Storage operations are guarded with device validation and exact typed confirmations for destructive actions. Backup and restore workflows use their own validation and safety checks rather than relying on the user to assemble commands manually.

See:

- [`docs/STORAGE.md`](docs/STORAGE.md) for storage and migration
- [`docs/RESTORE.md`](docs/RESTORE.md) for world and full-data recovery

## Updates

JustVoxel keeps operating-system updates and Minecraft workload updates separate.

The appliance operating system is updated through bootc and can be managed through `mjust` system-management commands.

The separate Minecraft/Paper workload has its own update flow, backup safeguards, version policy, and rollback handling.

See [`docs/SYSTEM.md`](docs/SYSTEM.md) for operating-system maintenance and [`docs/MJUST.md`](docs/MJUST.md) for the Minecraft update workflow.

## Documentation

Start with the document that matches what you are trying to do:

- [JustVoxel ISO Builder](https://github.com/home-server-project/justvoxel-iso) — install JustVoxel on a VM or physical machine
- [`docs/MJUST.md`](docs/MJUST.md) — operate the appliance
- [`docs/STATUS.md`](docs/STATUS.md) — understand the appliance health dashboard
- [`docs/SYSTEM.md`](docs/SYSTEM.md) — OS status, updates, resources, reboot, poweroff, and firmware controls
- [`docs/STORAGE.md`](docs/STORAGE.md) — storage choices, provisioning, mounts, and migration
- [`docs/RESTORE.md`](docs/RESTORE.md) — world and full Minecraft-data recovery
- [`docs/MANAGEMENT.md`](docs/MANAGEMENT.md) — management layers and native Linux administration
- [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) — why JustVoxel is built as an immutable appliance
- [`docs/VARIANTS.md`](docs/VARIANTS.md) — VM versus Bare Metal technical differences
- [`docs/BUILD.md`](docs/BUILD.md) — image composition, signing, CI, branches, and release mechanics
- [`docs/ROADMAP.md`](docs/ROADMAP.md) — current priorities, future features, and project non-goals

## Branch and release model

- `testing` — active development and `:testing` images
- `main` — validated promotions and stable `:10` images
- GitHub Releases are created from `main` only

Stable image targets are intended to be:

```text
ghcr.io/home-server-project/justvoxel-vm:10
ghcr.io/home-server-project/justvoxel-baremetal:10
```

Development images use the corresponding `:testing` tag.

## License

Apache-2.0. See [LICENSE](LICENSE).

Minecraft, Mojang software, Paper, plugins, and other third-party components retain their own licenses and distribution terms. JustVoxel does not embed Minecraft server binaries or Mojang server software in its bootc images.
