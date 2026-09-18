# JustVoxel

JustVoxel is a purpose-built immutable Minecraft server appliance for home and small-community servers.

It is designed for people who are comfortable installing an operating system and following normal computer instructions, but who do not want to become Linux, container, systemd, firewall, SELinux, or Minecraft-server administrators just to run a reliable server.

JustVoxel provides the same appliance experience through two editions:

- **JustVoxel VM** — for Proxmox, KVM/libvirt, VMware, Hyper-V, VirtualBox, and other supported hypervisors.
- **JustVoxel HWE** — for installation directly on physical hardware where additional hardware-support packages are useful.

Both editions use the same JustVoxel management experience, including the WebUI, `mjust`, Minecraft management, backups and restore, storage management, migration, updates, validation, and recovery tools.

## Which edition should I choose?

Choose **JustVoxel VM** when the server will run as a virtual machine.

Choose **JustVoxel HWE** when JustVoxel will run directly on a physical computer and you want the additional hardware-oriented package set.

**HWE is simply JustVoxel's name for the edition with extra physical-hardware packages.** It does not mean that the underlying Base is a separate "hardware-enabled" operating system.

<details>
<summary><strong>What does HWE add?</strong></summary>

HWE adds packages useful on physical machines, including:

- UPS support through Network UPS Tools (NUT)
- SMART, NVMe, Btrfs, and disk-management utilities
- hardware sensors and diagnostics
- firmware and UEFI update tooling
- CPU microcode support
- additional Wi-Fi and device firmware
- USB, PCI, Ethernet, and hardware-identification tools
- power-management and disk-tuning utilities

The normal JustVoxel appliance functionality remains the same. These packages extend what the system can work with when physical hardware is present.

For the exact current package list, see [`build_files/packages.env`](build_files/packages.env).

</details>

## What JustVoxel provides

JustVoxel is installed as a complete server appliance rather than as a package or setup script on top of an arbitrary Linux installation.

The normal interface is designed around server tasks instead of Linux internals. Common operations include:

- first-time Minecraft setup
- Java and optional Bedrock cross-play
- player and whitelist management
- start, stop, and restart
- automatic and manual backups
- world and full-data restore
- import and export of existing Minecraft servers
- local and network storage management
- Minecraft/Paper updates
- operating-system updates
- health/status and validation
- Web-based management
- terminal management through `mjust`

Advanced administrators can still use the normal underlying Linux tools when needed.

## Minecraft software and EULA

JustVoxel does **not** ship Minecraft server binaries, Mojang server software, a pre-created world, or a pre-accepted Minecraft EULA.

During setup, the administrator accepts the Minecraft EULA and JustVoxel configures a separate containerized Paper-based Minecraft workload.

Minecraft/Paper updates remain separate from JustVoxel operating-system updates.

## Install JustVoxel

For normal installation, use the [JustVoxel ISO Builder](https://github.com/home-server-project/justvoxel-iso).

The installer project contains the installation-media workflow, VM and physical-machine installation guidance, disk requirements, first-boot information, and SSH/access guidance.

## Documentation

Detailed appliance documentation is maintained with the JustVoxel source in [JustVoxel Base](https://github.com/home-server-project/justvoxel-base).

That documentation covers `mjust`, WebUI management, Minecraft runtime behavior, backups and restore, migration, storage, system management, validation, architecture, and the project roadmap.

For installation-media documentation, use the [JustVoxel ISO Builder](https://github.com/home-server-project/justvoxel-iso).

## Source

The common JustVoxel appliance implementation is developed in [JustVoxel Base](https://github.com/home-server-project/justvoxel-base).

This repository contains the final JustVoxel product layer and release-facing project files.

## License

Apache-2.0. See [LICENSE](LICENSE).

Minecraft, Mojang software, Paper, plugins, and other third-party components retain their own licenses and distribution terms. See [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).
