# JustVoxel variants

## Shared core

Both VM and Bare Metal receive:

- AlmaLinux 10 minimal-plus / bootc
- Podman and Quadlet support
- NetworkManager + `nmtui`
- OpenSSH
- firewalld
- SELinux container policy and administration tools
- upstream `just`
- `skopeo`
- basic network/file troubleshooting utilities
- Tailscale and NetBird clients, installed but disabled/unconfigured

Neither image includes Cockpit, a virtualization stack, a NAS stack, databases, monitoring suites, or unrelated application servers.

## VM

The VM image intentionally excludes physical-hardware administration packages such as NUT, Btrfs administration, SMART/NVMe tooling, sensors, firmware update tooling, and USB/PCI diagnostics.

## Bare Metal

Bare Metal adds:

- NUT / NUT client
- Btrfs tools
- SMART tools + SELinux policy
- NVMe CLI
- lm_sensors
- ethtool
- USB / PCI utilities
- dmidecode
- fwupd / fwupd-efi

NUT is shipped disabled and unconfigured because UPS hardware and policy are site-specific.
