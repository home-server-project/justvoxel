# JustVoxel

JustVoxel is a purpose-built immutable Minecraft server appliance from the Home Server Project.

> **Development status:** active implementation is on the `testing` branch. `mjust` now provides the first management skeleton, but VM and hardware validation are still required before production use.

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

## Foundation

Both variants include bootc, Podman/Quadlet support, NetworkManager with systemd-resolved, OpenSSH, SELinux tooling, firewalld, upstream `just`, Tailscale and NetBird clients, NFS/SMB client support, XFS tooling, and lightweight QEMU/Proxmox, VMware, and Hyper-V guest integration. Tailscale and NetBird remain disabled and unconfigured by default.

The Bare Metal variant additionally includes NUT, Btrfs tools, SMART/NVMe tooling, sensors, ethtool, USB/PCI diagnostics, DMI information, firmware tooling, Wi-Fi support/firmware, CPU microcode support, hdparm, and powertop.

The VM variant deliberately excludes that physical-hardware administration set. Linux kernel support provides the basic VirtualBox guest drivers; JustVoxel does not add out-of-tree VirtualBox Guest Additions or DKMS machinery.

## Minecraft runtime

JustVoxel ships immutable templates for Paper with Geyser, Floodgate, and ViaVersion. The image itself does not accept the Minecraft EULA, generate a world, or create an active Quadlet.

`mjust setup` creates administrator-owned runtime files under `/etc` from those templates. The active Quadlet and environment files are local copies and are never executed directly from `/usr/share/justvoxel/templates`, so a future bootc update cannot silently replace a working machine's configuration.

The Minecraft data path is configurable independently from backups. This supports, for example, a SATA SSD for the bootc OS, an NVMe filesystem for the Minecraft world, and a different local partition, disk, NFS share, or SMB share for backups.

## mjust

The management layer includes first setup, safe configuration changes, status/start/stop/restart, online player checks, Java and Floodgate whitelist operations, verified cold backup, player-aware Minecraft maintenance, logs, and appliance validation.

The container image tag and Minecraft/Paper game version are independent administrator choices. The container can use upstream `stable`, upstream `latest`, or a validated custom/exact tag. Minecraft can stay pinned to an exact stable Paper-supported version or deliberately follow `VERSION=LATEST`.

Moving container tags are refreshed only when mjust explicitly pulls them. Pre-update backups use the same retention policy as timer/manual backups. JustVoxel tracks one previous Minecraft image for rollback and never performs a broad Podman image prune.

Disk partitioning, formatting, fstab generation, and storage migration are represented in the design but intentionally perform no destructive action yet. Those safeguards belong to the next storage implementation step.

See `docs/MJUST.md` and `docs/STORAGE.md`.

## Branch and release model

- `testing` — active development and `:testing` images.
- `main` — validated promotions and stable `:10` images.
- GitHub Releases are created from `main` only.
- Image rechunking uses the Home Server Project / Pasiv Black Box limits: 127 RPM chunks, 128 OCI layers maximum.

No Minecraft server JAR or Mojang server binary is baked into the bootc image.

## Credits

The `mjust` interaction model is inspired by Universal Blue's `ujust` / `ugum` work in `ublue-os/packages`. JustVoxel uses its own small server-focused implementation. Any future direct source reuse will retain the applicable Apache-2.0 attribution and notices.

## License

Apache-2.0. See [LICENSE](LICENSE).
