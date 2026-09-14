# Third-party notices

## AlmaLinux / bootc foundation

JustVoxel composes an AlmaLinux 10 minimal-plus bootc root filesystem using AlmaLinux repositories and the upstream `bootc-base-imagectl` build pattern.

- AlmaLinux: https://almalinux.org/
- bootc: https://github.com/bootc-dev/bootc

## Passive Black Box build reference

The repository/build layout is adapted from the proven Home Server Project / Highway to IT Passive Black Box AlmaLinux bootc build pattern:

- https://github.com/highwaytoit/pasiv-black-box

JustVoxel intentionally removes the monitoring/Cockpit stack and keeps only patterns useful to this Minecraft appliance.

## Universal Blue `ujust` / `ugum`

The planned `mjust` user interaction model is inspired by Universal Blue's `ujust` / `ugum` implementation:

- https://github.com/ublue-os/packages/tree/main/packages/ublue-os-just

The current Step 1 foundation contains no copied `ujust` or `ugum` source code. If future JustVoxel work directly reuses or adapts Universal Blue source, the applicable Apache-2.0 license and attribution will be retained with those files.
