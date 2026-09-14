# JustVoxel

JustVoxel is a purpose-built immutable Minecraft server appliance from the Home Server Project.

> **Development status:** foundation work is in progress on the `testing` branch. Do not use the current images as a production Minecraft server yet.

## Images

JustVoxel is built from one AlmaLinux 10 minimal-plus bootc source tree with two thin variants:

- **JustVoxel VM** — for KVM/libvirt, Proxmox, VMware, VirtualBox, and similar hypervisors.
- **JustVoxel Bare Metal** — the same appliance core plus physical-machine administration tools.

Planned image repositories:

```text
ghcr.io/home-server-project/justvoxel-vm:10
ghcr.io/home-server-project/justvoxel-baremetal:10
```

Development images use the `:testing` tag.

## Foundation included in Step 1

Both variants include bootc, Podman/Quadlet support, NetworkManager, OpenSSH, SELinux tooling, firewalld, upstream `just`, basic troubleshooting tools, and optional-use Tailscale and NetBird clients. Tailscale and NetBird are installed but disabled and unconfigured by default.

The Bare Metal variant additionally includes NUT, Btrfs tools, SMART/NVMe tooling, sensors, ethtool, USB/PCI diagnostics, DMI information, and firmware tooling.

The VM variant deliberately excludes that physical-hardware administration set.

The interactive Bash prompt uses a bright green `user@hostname` / prompt marker as a small Minecraft-style identity cue.

## Branch and release model

- `testing` — active development and `:testing` images.
- `main` — validated promotions and stable `:10` images.
- GitHub Releases are created from `main` only.
- Image rechunking uses the Home Server Project / Pasiv Black Box limits: 127 RPM chunks, 128 OCI layers maximum.

## Implementation sequence

1. Repository and bootc image foundation — current step.
2. Minecraft Quadlet/runtime — adapted from a proven running Paper/Geyser/Floodgate deployment.
3. `mjust` management layer — built on upstream `just` after the runtime exists.

No Minecraft server JAR or Mojang server binary is baked into the bootc image.

## Credits

The future `mjust` interaction model is intentionally inspired by Universal Blue's `ujust` / `ugum` work in `ublue-os/packages`. The current foundation does not copy their source code. Any future direct reuse or adaptation will retain the applicable Apache-2.0 attribution and notices.

## License

Apache-2.0. See [LICENSE](LICENSE).
