# JustVoxel

JustVoxel is a purpose-built immutable Minecraft server appliance from the Home Server Project.

> **Development status:** active implementation is on the `testing` branch. `mjust` now includes guarded local/network storage provisioning and Minecraft data migration, but VM and hardware validation are still required before production use.

## Images

JustVoxel is built from one AlmaLinux 10 minimal-plus bootc source tree with two thin variants:

- **JustVoxel VM** — for KVM/libvirt, Proxmox, VMware, VirtualBox, Hyper-V, and similar hypervisors.
- **JustVoxel Bare Metal** — the same appliance core plus physical-machine administration tools.

Planned image repositories:

```text
ghcr.io/home-server-project/justvoxel-vm:10
ghcr.io/home-server-project/justvoxel-baremetal:10
```

Development images use the `:testing` tag.

## Installation

The JustVoxel operating-system repository intentionally stays focused on the appliance image and runtime.

For installer ISO creation, installation layout, disk-size guidance, VM recommendations, and hardware/system requirements, use the dedicated [JustVoxel ISO Builder](https://github.com/home-server-project/justvoxel-iso) repository.

The ISO builder is being developed as a lightweight unattended bootc installer rather than a desktop-based interactive installer.

## Memory guidance

JustVoxel does not hard-block lower-memory systems. Users are free to try the appliance on any compatible hardware.

- **Minimum recommended:** 8 GiB RAM
- **Recommended for normal performance:** 12–16 GiB RAM

Systems below 8 GiB may still run JustVoxel, but Minecraft performance can be limited depending on world size, plugins, player count, view/simulation distance, Geyser/Floodgate cross-play, and other workload characteristics.

JustVoxel enables zram as a memory-pressure safety buffer using `zram-generator`'s built-in sizing policy: `min(RAM / 2, 4096 MiB)`. Zram is not additional physical RAM and is not counted when `mjust setup` calculates Minecraft memory recommendations. No disk swap is configured by JustVoxel by default.

The setup wizard remains permissive and uses conservative recommendations based on physical `MemTotal` only.

## Foundation

Both variants include bootc, Podman/Quadlet support, NetworkManager with systemd-resolved, OpenSSH, SELinux tooling, firewalld, upstream `just`, Tailscale and NetBird clients, NFS/SMB client support, XFS/partition tooling, and lightweight QEMU/Proxmox, VMware, and Hyper-V guest integration. Tailscale and NetBird remain disabled and unconfigured by default.

The Bare Metal variant additionally includes NUT, Btrfs tools, SMART/NVMe tooling, sensors, ethtool, USB/PCI diagnostics, DMI information, firmware tooling, Wi-Fi support/firmware, CPU microcode support, hdparm, and powertop.

The VM variant deliberately excludes that physical-hardware administration set. Linux kernel support provides the basic VirtualBox guest drivers; JustVoxel does not add out-of-tree VirtualBox Guest Additions or DKMS machinery.

## Minecraft runtime

JustVoxel ships immutable templates for Paper with Geyser, Floodgate, and ViaVersion. The image itself does not accept the Minecraft EULA, generate a world, or create an active Quadlet.

`mjust setup` creates administrator-owned runtime files under `/etc`. The container image tag and Minecraft game version are independently configurable. Minecraft data location is independent from backup storage.

## Storage

For VM, the recommended layout is one primary virtual disk for the appliance/Minecraft data and a second virtual disk for backups. NFS and SMB/CIFS backup targets are also supported and the required clients are included in the VM image.

Bare Metal supports dedicated internal disks, external USB disks/sticks, existing local partitions, new partitions created only in already-unallocated space, NFS, SMB/CIFS, and normal local directories.

Whole-disk erase excludes system disks and mounted disks. Destructive operations require typing the exact device phrase. Local persistent mounts use UUIDs. JustVoxel does not automatically shrink existing filesystems.

`mjust storage-migrate` can move Minecraft data to provisioned local storage after a verified cold backup; the old data is retained until the administrator removes it manually.

## mjust

The management layer includes first setup, safe configuration changes, status/start/stop/restart, online player checks, Java and Floodgate whitelist operations, verified cold backup, player-aware Minecraft maintenance, storage provisioning/migration, logs, and appliance validation.

Moving container tags are refreshed only by an explicit pull. Pre-update backups use the same retention policy as timer/manual backups. JustVoxel tracks one previous Minecraft image for rollback and never performs a broad Podman image prune.

See `docs/MJUST.md` and `docs/STORAGE.md`.

## Branch and release model

- `testing` — active development and `:testing` images.
- `main` — validated promotions and stable `:10` images.
- GitHub Releases are created from `main` only.
- Image rechunking uses the Home Server Project / Pasiv Black Box limits: 127 RPM chunks, 128 OCI layers maximum.

No Minecraft server JAR or Mojang server binary is baked into the bootc image.

## Credits

The `mjust` interaction model is inspired by Universal Blue's `ujust` / `ugum` work in [ublue-os/packages](https://github.com/ublue-os/packages). JustVoxel uses its own small server-focused implementation. Any future direct source reuse will retain the applicable Apache-2.0 attribution and notices.

## License

Apache-2.0. See [LICENSE](LICENSE).
