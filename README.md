# JustVoxel

JustVoxel is a purpose-built immutable Minecraft server appliance from the Home Server Project.

> **Development status:** active implementation is on the `testing` branch. The Step 2 runtime is still template-only; do not use the current images as a production Minecraft server yet.

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

Both variants include bootc, Podman/Quadlet support, NetworkManager with systemd-resolved, OpenSSH, SELinux tooling, firewalld, upstream `just`, basic troubleshooting tools, Tailscale and NetBird clients, NFS/SMB client support, XFS tooling, and lightweight QEMU/Proxmox, VMware, and Hyper-V guest integration. Tailscale and NetBird remain disabled and unconfigured by default.

The Bare Metal variant additionally includes NUT, Btrfs tools, SMART/NVMe tooling, sensors, ethtool, USB/PCI diagnostics, DMI information, firmware tooling, Wi-Fi support/firmware, CPU microcode support, hdparm, and powertop.

The VM variant deliberately excludes that physical-hardware administration set. Linux kernel support provides the basic VirtualBox guest drivers; JustVoxel does not add out-of-tree VirtualBox Guest Additions or DKMS machinery.

The interactive Bash prompt uses a bright green `user@hostname` / prompt marker as a small Minecraft-style identity cue.

## Minecraft runtime — Step 2

The image now ships inactive templates for a Paper server with Geyser, Floodgate, and ViaVersion, based on the previously hardware-verified deployment. The image does not accept the Minecraft EULA, generate a world, or activate the Quadlet automatically.

A generic cold-backup helper is included. Storage paths are not hardcoded. The helper can later be configured for the system filesystem, a dedicated disk, a dedicated partition, or a mounted NFS/SMB share and can fail closed when an expected mount is absent.

## Branch and release model

- `testing` — active development and `:testing` images.
- `main` — validated promotions and stable `:10` images.
- GitHub Releases are created from `main` only.
- Image rechunking uses the Home Server Project / Pasiv Black Box limits: 127 RPM chunks, 128 OCI layers maximum.

## Implementation sequence

1. Repository and bootc image foundation — complete on `testing`.
2. Minecraft runtime and backup foundation — current implementation stage.
3. `mjust` management layer — render setup, EULA acceptance, ports, storage, whitelist, service control, backups, updates, and validation.

No Minecraft server JAR or Mojang server binary is baked into the bootc image.

## Credits

The future `mjust` interaction model is intentionally inspired by Universal Blue's `ujust` / `ugum` work in `ublue-os/packages`. The current implementation does not copy their source code. Any future direct reuse or adaptation will retain the applicable Apache-2.0 attribution and notices.

## License

Apache-2.0. See [LICENSE](LICENSE).
