# mjust administrator guide

`mjust` is the JustVoxel administrator interface. It is a small wrapper around upstream `just`; recipes stay short and delegate appliance work to scripts under `/usr/libexec/justvoxel/mjust/`.

The interaction model is inspired by Universal Blue `ujust`, but JustVoxel uses its own server-focused implementation.

This document is the current operational reference for the `testing` implementation. It records what mjust can do now, the supported storage/partition models, the safety rules, and the operations that are intentionally not automated.

> JustVoxel is still under development. VM and Bare Metal validation are required before the project is treated as production-ready.

## Configuration ownership

The bootc image owns immutable implementation and templates under:

- `/usr/share/justvoxel/`
- `/usr/libexec/justvoxel/`

The administrator-owned active configuration lives under `/etc`, including:

- `/etc/justvoxel/justvoxel.conf`
- `/etc/justvoxel/minecraft.env`
- `/etc/justvoxel/minecraft-backup.env`
- `/etc/containers/systemd/minecraft.container`
- `/etc/systemd/system/minecraft-backup.service`
- `/etc/systemd/system/minecraft-backup.timer`

Bootc image updates may update mjust and its immutable templates, but they do not silently overwrite an installed machine's active `/etc` configuration.

Required runtime directory state under `/var` is treated differently. `/var` persists across bootc deployments, so image updates cannot rely on package-owned `/var` directories being copied into an already-installed machine. JustVoxel keeps the image `/var` skeleton minimal and uses declarative `tmpfiles.d`/systemd mechanisms for required runtime directories. The image finalization path runs fatal bootc lint before the final `/var` cleanup so missing reconstruction rules are not hidden by the cleanup step.

## Current mjust commands

The current command surface is:

- `mjust` — interactive menu
- `mjust setup` — normal first-time appliance configuration
- `mjust setup-advanced` — the same setup implementation with advanced options enabled
- `mjust configure` — safe changes to an installed configuration
- `mjust status` — Minecraft service status
- `mjust start`
- `mjust stop`
- `mjust restart`
- `mjust players` — query players through RCON
- `mjust whitelist-list`
- `mjust whitelist-add-java <name>`
- `mjust whitelist-remove-java <name>`
- `mjust whitelist-add-bedrock <gamertag>`
- `mjust whitelist-remove-bedrock <gamertag>`
- `mjust backup` — verified cold backup
- `mjust update-minecraft` — container-image maintenance
- `mjust logs`
- `mjust validate`
- `mjust storage` — storage-management menu
- `mjust storage-plan` — non-destructive storage/device overview
- `mjust storage-disk` — dedicated whole-disk provisioning
- `mjust storage-partition` — adopt/use an existing partition
- `mjust storage-free-space` — create a partition only in existing unallocated space
- `mjust storage-network` — NFS or SMB/CIFS backup target
- `mjust storage-migrate` — guarded Minecraft data migration

## First setup

`mjust setup` refuses to overwrite an existing JustVoxel installation.

The normal setup wizard currently collects:

- Minecraft persistent-data location
- Minecraft Java heap
- maximum total Minecraft container memory
- Minecraft Java host TCP port
- optional Bedrock UDP port
- timezone
- maximum players
- server welcome message (MOTD)
- Minecraft container-image policy
- Minecraft/Paper game-version policy
- backup target
- backup retention count
- systemd backup schedule
- whether the backup timer is enabled
- explicit Minecraft EULA acceptance

Memory values are suggested from installed system RAM, but the administrator can change both the Java heap and the maximum total Minecraft memory. The Java heap is contained inside the container memory limit, so the total limit must be larger than the heap.

Normal setup does not ask for UID/GID. It selects the non-root IDs used for persistent Minecraft data in this order:

1. the administrator who invoked setup through `sudo` (`SUDO_UID:SUDO_GID`)
2. the local `voxel` account when available
3. `1000:1000` as the final fallback

UID or GID `0` is rejected. Setup prints the selected ownership source briefly so the administrator knows what will own the world data.

`mjust setup-advanced` calls the same setup implementation with an advanced flag. It keeps the same normal flow but allows the administrator to override the suggested Minecraft data UID/GID. This is intentionally not duplicated into a second wizard.

Normal `mjust configure` does not expose UID/GID changes for an existing world. Changing ownership of an already-populated world requires a separately designed guarded operation rather than an ordinary configuration edit.

## Container image policy

The Minecraft container image and Minecraft/Paper game version are separate settings.

The container is based on `docker.io/itzg/minecraft-server`.

New setup uses these image-tag choices:

- `stable` — moving upstream released image; recommended/default for new installations
- `latest` — moving upstream main-branch image
- custom/exact tag — administrator-selected tag validated against the upstream registry

mjust rejects upstream tags that are explicitly marked deprecated by the upstream image metadata when that metadata is available.

A normal service restart does not itself pull a newer OCI image. A moving image tag changes locally only after an explicit image pull. `mjust update-minecraft` performs that remote/local comparison and pull operation.

Existing JustVoxel configurations created before the setting existed retain the compatibility default of `latest`; setup does not silently rewrite an installed machine's existing policy.

For a custom or exact tag, mjust keeps using that tag until the administrator changes it.

## Minecraft/Paper version policy

Minecraft/Paper versioning is independent from the container-image tag.

The administrator can choose:

1. Pin the newest Minecraft version that currently has a stable Paper build. This is the recommended/default choice.
2. Use `VERSION=LATEST`. The already-installed itzg container resolves the newest Minecraft/Paper release when Minecraft starts, so the game/Paper files under persistent data may update after a start or restart even when no newer Podman image was pulled.
3. Pin another exact Minecraft version, provided PaperMC reports a stable Paper build for it.

When a pinned Minecraft version is configured, `mjust update-minecraft` may report that a newer stable Paper-supported Minecraft version exists, but it does not silently change the pinned game version.

`mjust configure` can change both the container image policy and the Minecraft version policy. It does not automatically restart Minecraft after the change.

## Player-aware service control

Stop, restart, update, backup maintenance, and data migration use RCON player checks when Minecraft is active.

If player status cannot be confirmed, mjust fails closed instead of interrupting the server.

If players are online, the administrator can cancel or explicitly continue. When maintenance continues, the normal configured Minecraft graceful-stop path is used. The server announces shutdown to connected players and waits 60 seconds before the final stop/save sequence.

## Backup model

Minecraft backups are full cold backups of the persistent Minecraft data directory.

The backup helper:

1. validates the configured backup target
2. verifies that the destination is writable
3. acquires the shared maintenance lock
4. gracefully stops Minecraft if it is running
5. writes the archive to a `.partial` file
6. verifies the archive
7. atomically publishes the completed archive
8. applies retention
9. restarts Minecraft only when the backup operation itself stopped it, unless another maintenance operation requested that it remain stopped

Manual backups, scheduled backups, pre-update backups, and pre-migration backups all use the same retention count. If retention is seven, publishing the eighth verified archive removes the oldest completed archive.

A missing or incorrect external/network mount fails closed before Minecraft is stopped, so a backup cannot silently fall back onto the root filesystem.

## Minecraft container updates

`mjust update-minecraft` compares:

- the configured remote image digest
- the local image digest for that tag
- the image currently used by the running Minecraft container

The update flow is designed as one maintenance cycle:

1. inspect image/update state
2. report current container policy and available upstream information
3. query players if the server is active
4. require explicit administrator approval
5. re-check player state immediately before maintenance
6. acquire the maintenance lock
7. create a verified cold backup and leave Minecraft stopped
8. pull the configured image if required
9. start Minecraft once
10. verify RCON, Minecraft version, plugins, Geyser when enabled, and the running image identity

JustVoxel tracks one previous Minecraft image generation for rollback and does not run a general Podman image prune that could affect unrelated containers.

## Storage model overview

Minecraft data and backup storage are independent choices.

Podman's global image store is not moved by mjust.

Typical layouts are:

### VM

Recommended:

- primary virtual disk — JustVoxel OS and Minecraft data
- second virtual disk — backups

NFS and SMB/CIFS backup targets are also supported. The VM image includes `nfs-utils` and `cifs-utils`.

If no safe secondary virtual disk is available, the storage flow tells the administrator to add another disk in the hypervisor and rerun the operation.

### Bare Metal

Supported backup/storage targets include:

- dedicated internal disk
- external USB disk or USB stick
- existing local partition/filesystem
- a new partition created only from already-unallocated disk space
- NFS share
- SMB/CIFS share
- normal directory on the system filesystem

USB storage is treated as ordinary block storage. mjust shows device path, size, model, transport, filesystem, UUID, and mount state so the administrator can identify the intended device.

## `mjust storage-plan`

`mjust storage-plan` is non-destructive. It is intended to be the first inspection command before changing storage.

It reports the current JustVoxel variant, visible block storage, protected system disks, and the recommended VM/Bare Metal storage model.

## Whole-disk provisioning

`mjust storage-disk` provisions an entire dedicated disk.

The flow:

1. identifies the disks backing `/`, `/boot`, `/boot/efi`, and `/var`
2. excludes those system disks from whole-disk erase candidates
3. excludes disks that already have mounted child filesystems
4. shows safe candidates with device, size, transport, and model
5. requires the administrator to select the exact disk
6. shows the selected device layout again
7. requires the exact destructive phrase `ERASE /dev/...`
8. wipes existing signatures
9. creates GPT
10. creates one XFS partition
11. mounts it persistently by filesystem UUID

A simple `y`/`yes` is not enough for whole-disk destruction.

This path works for a second VM disk, internal SATA/NVMe storage, or an external USB disk/stick.

## Existing partition adoption

`mjust storage-partition` can use an existing writable partition.

Supported existing local filesystems are:

- XFS
- ext4
- Btrfs

If the partition is already mounted at a safe, non-critical mount point, JustVoxel adopts and validates the existing mount without rewriting the administrator's current mount configuration.

If the partition is not mounted, mjust asks for a mount point and creates a persistent UUID-based mount.

If the selected partition contains no recognized filesystem, mjust may format only that selected partition as XFS after requiring the exact phrase `FORMAT /dev/...`.

mjust does not silently reformat a recognized filesystem.

## Partition creation from unallocated space

`mjust storage-free-space` can create a new partition only in space that is already unallocated.

This is intended for cases such as a system disk or secondary disk where installation intentionally left unused space.

The flow shows the disk and the largest unallocated segment, then lets the administrator select either all available space or a specific size in GiB.

Before writing the partition table, mjust requires the exact phrase `CREATE PARTITION /dev/...`.

The new partition is formatted as XFS and mounted by UUID.

### Important limitation

JustVoxel does **not** automatically shrink or resize an existing filesystem or partition.

If a disk is fully allocated, mjust refuses the free-space operation rather than trying to make space by shrinking an existing filesystem.

## System-disk rules

Whole-disk erase of the detected system disk is not offered.

Using already-unallocated space on the system disk is allowed because the operation does not resize or overwrite existing partitions. The exact free-space range and device are displayed before the new partition is created.

Critical mount points such as `/`, `/boot`, `/boot/efi`, and `/var` are protected from adoption as JustVoxel storage targets.

## Persistent local mounts

New local mounts created by mjust use filesystem UUIDs rather than transient Linux names such as `/dev/sdb1`.

That is especially important for USB storage because device names can change after reboot or when hardware is reconnected.

The runtime records the expected filesystem identity and validates it before Minecraft data or backup operations continue.

## NFS backup storage

NFS is supported for both VM and Bare Metal backup targets.

mjust can create a persistent NFS mount using `_netdev` and `nofail`, or adopt an already-mounted matching NFS share without rewriting its existing mount configuration.

Before a backup uses the share, the backup helper confirms:

- the mount exists
- the source matches the configured source
- the target is actually writable

This allows NFS servers that use root-squash or server-side ownership rules; JustVoxel verifies usable write access rather than requiring local `chown` semantics.

## SMB/CIFS backup storage

SMB/CIFS is supported for both VM and Bare Metal backup targets.

When mjust creates an SMB mount, credentials are stored in:

`/etc/justvoxel/smb-backup.credentials`

The file is root-owned with root-only permissions and is referenced from the mount configuration. Credentials are not written directly into `/etc/fstab`.

Like NFS, the mount uses network-aware boot options and the backup helper verifies the actual mounted source and writability before stopping Minecraft.

## Minecraft data migration

`mjust storage-migrate` moves the active Minecraft persistent-data directory to provisioned local storage.

Supported migration destinations are:

- dedicated whole disk
- existing local partition/filesystem
- new partition created in already-unallocated space

The migration process:

1. provision or adopt the destination storage
2. refuse a non-empty destination Minecraft directory
3. query player state and fail closed if unknown
4. require confirmation if players are online
5. acquire the shared maintenance lock
6. create a verified cold backup and leave Minecraft stopped
7. copy the full persistent data tree with `rsync`
8. run a dry-run `rsync` verification
9. update the administrator-owned data path and mount identity
10. apply SELinux labels
11. regenerate the local Quadlet/runtime configuration
12. start and verify Minecraft when it was running before migration

If the new runtime fails, mjust restores the previous data-path configuration and attempts to return to the old deployment.

The old Minecraft data directory is not automatically deleted. The administrator removes it only after validating normal gameplay and backups.

## Backup target strength

JustVoxel permits a backup partition on the same physical disk as the OS/data because it can still protect against some configuration or reinstall mistakes.

It should not be treated as protection against physical disk failure.

Stronger backup separation comes from:

- a separate internal disk
- a separate virtual disk with independent hypervisor backup/replication policy
- an external USB disk/stick
- NFS/SMB storage on another machine

## `mjust configure`

The current configuration menu can:

- show the current configuration
- change Java/container memory limits
- change Java/Bedrock host ports
- change MOTD and maximum players
- change backup schedule and timer state
- enter the storage/migration menu
- change the container-image tag policy
- change the Minecraft/Paper version policy

Configuration changes that require a Minecraft restart are not applied by an automatic disruptive restart. mjust tells the administrator to restart when appropriate.

## Whitelist management

mjust provides separate Java and Bedrock/Floodgate whitelist operations.

Java names are validated before the normal Minecraft whitelist command is run. Bedrock whitelist operations use Floodgate's whitelist command when Bedrock support is enabled.

RCON remains internal to the Minecraft container and is not published as a host port.

## Validation

`mjust validate` checks the active JustVoxel deployment, including important runtime files and permissions, SELinux labeling, firewall state, storage identity, backup target availability, generated service state, RCON response, Minecraft version, Bedrock/Geyser state when enabled, and the systemd failed-unit set.

A clean appliance is expected to report no failed systemd units. If `systemctl --failed` is non-empty, `mjust validate` shows the failed units and returns failure. This makes first-boot service regressions such as the gssproxy state-directory failure visible during preserved-VM validation.

The goal is to fail visibly when the appliance does not match its recorded configuration rather than continuing with an unexpected mount or incomplete runtime.

## Operations intentionally not automated

The current implementation deliberately does **not**:

- shrink an existing partition
- shrink an existing filesystem
- resize an existing filesystem to create free space
- erase the detected system disk as a whole-disk target
- silently reformat a recognized filesystem
- silently overwrite active `/etc` configuration during a bootc update
- automatically delete the old Minecraft data directory after migration
- expose RCON publicly
- relocate Podman's global image store
- run a broad Podman image prune
- automatically change a pinned Minecraft/Paper version

These are intentional safety boundaries, not missing automatic steps.

## Current development status

The management and storage flows are implemented on `testing`, but they still require destructive VM testing and physical-hardware validation before stable promotion.

The safest validation order is VM first: one system disk, one disposable secondary virtual disk, then dedicated-disk provisioning, partition adoption, free-space partition creation, network backup targets, backup/retention, and Minecraft data migration. Bare Metal validation should follow only after the VM paths are proven.

# Future mjust system-management roadmap

The next major mjust expansion is intended to make JustVoxel easier to administer for people who do not want to interpret raw Linux command output.

The planned direction is a new `mjust system` area that presents short, readable appliance information and wraps common bootc/systemd/power operations with the same safety model already used for Minecraft maintenance.

These features are roadmap items only. They are **not implemented yet**.

## Future `mjust system` overview

The system area should provide a compact appliance summary rather than dumping raw command output.

A future overview should include information such as:

- hostname
- uptime
- JustVoxel variant and image/version
- CPU and memory summary
- Minecraft service health
- latest backup status and age
- storage capacity/free-space summary
- network state and active addresses
- failed systemd services
- current bootc deployment state
- whether an OS update is staged

The goal is output that can be understood without Linux-administration experience.

A future convenience command such as `mjust health` may provide the same information as a short health dashboard, for example:

- Minecraft: healthy
- Backups: healthy / last successful backup age
- Storage: healthy / free space
- Network: connected / active address
- Failed services: none or summarized failures
- OS deployment: current / staged update available

## Future network status

The planned network view should summarize useful information instead of reproducing the full output of `ip address` or NetworkManager diagnostics.

Useful fields include:

- detected Ethernet/Wi-Fi interfaces
- interface up/down state
- link/carrier state
- current IPv4 and IPv6 addresses
- default-route interface and gateway
- active DNS configuration
- current hostname

The output should make common problems obvious, such as an interface being down, no address being assigned, or no default route being present.

## Future storage health

Storage health should be variant-aware.

### Bare Metal

When supported by the hardware, mjust may summarize:

- physical disk identity
- mounted filesystem health/state
- capacity and free space
- SMART overall health
- NVMe health status
- media/data-integrity error counters
- device temperature
- remaining-life/wear indicators when the drive exposes them

The user-facing result should be concise: healthy, warning, or problem, with the important reason shown when attention is required.

Raw SMART/NVMe reports should remain available for troubleshooting but should not be the normal user interface.

### VM

The VM variant cannot reliably report the health of the physical device behind a virtual disk.

It should report virtual-disk capacity, filesystem/mount status, and local I/O/filesystem problems, while clearly stating that physical storage health is managed by the hypervisor/storage platform.

## Future failed-service view

A future system-health option should summarize `systemctl --failed` in a readable form.

When nothing has failed, it should simply report that there are no failed system services.

When failures exist, it should show the affected unit name, description, and a short state/result summary rather than forcing the administrator to interpret the full systemd output immediately.

Detailed journal output can remain a separate troubleshooting action.

## Future bootc deployment status

JustVoxel should use bootc's machine-readable status output for program logic rather than parsing the human-formatted command output.

The mjust view should translate deployment state into simple concepts such as:

- **Running image** — deployment currently booted
- **Staged image** — deployment already downloaded and ready for the next boot
- **Rollback image** — previous deployment retained by bootc

The user should not need to understand bootc deployment internals to know which image is running and whether an update is ready.

## Future OS update flow

The OS-update flow should never assume that an already-staged deployment is still the newest available image.

Even when an update is staged, the administrator should still be offered **Check for newer update**.

The intended flow is:

1. show the current bootc deployment status
2. show any already-staged deployment
3. offer **Check for newer update** regardless of whether a staged deployment exists
4. perform a non-disruptive update check
5. if the staged deployment is still current, report that it remains the newest available image
6. if something newer exists, offer to download/stage the newer deployment
7. show the resulting staged deployment
8. offer a controlled reboot into that deployment

This matters for appliances that may remain running for weeks or months without rebooting: an older update may already be staged while a newer image has since become available.

A normal check should not reboot the system.

## Future reboot into an OS update

Rebooting into a staged JustVoxel update should use Minecraft-aware safety checks rather than immediately restarting the host.

The planned flow is:

1. confirm that a staged deployment exists
2. query Minecraft player state when the server is running
3. fail closed when player state is unknown
4. if players are online, show the current player state and require explicit administrator confirmation
5. create a verified cold Minecraft backup before the OS transition
6. use the normal graceful Minecraft shutdown path
7. reboot into the staged bootc deployment

This provides the same family-server protection model used by Minecraft container maintenance.

## Future power controls

The `mjust system` area may also expose common appliance power operations:

- reboot
- power off

These should use the Minecraft-aware interruption checks before shutting down a running appliance.

An ordinary reboot or poweroff does not necessarily need to force a new full backup every time; the important baseline is player awareness and a graceful Minecraft shutdown. OS update/reboot operations may use the stronger pre-update backup policy described above.

## Future Bare-Metal firmware reboot

A future Bare Metal-only option may expose reboot into UEFI/firmware setup when the running hardware and firmware support it.

This option should **not** be exposed by the JustVoxel VM variant. Firmware configuration for a VM is considered a hypervisor responsibility.

For Bare Metal, the planned behavior is:

1. verify that firmware-setup reboot is supported by the running system
2. perform a best-effort check for an attached display/DRM connector
3. if a display appears connected, allow normal confirmation
4. if no display appears connected, warn clearly and default to cancel
5. if display state cannot be determined reliably, explain that the check is uncertain and require explicit confirmation
6. perform the same Minecraft player/graceful-stop checks before rebooting

Display detection is advisory only. Hardware, KVM switches, EDID behavior, and firmware can make Linux display detection imperfect, so it should not be treated as an absolute guarantee.

## Much-later bootc rollback work

A JustVoxel-aware OS rollback is intentionally **not** part of the near-term system-management work.

Bootc rollback can restore the previous deployment together with the `/etc` state associated with that deployment. JustVoxel keeps active administrator configuration under `/etc`, including Minecraft and systemd/Quadlet configuration, so a safe appliance-level rollback needs a separate design for preserving and reconciling current administrator state.

A future rollback implementation would likely need to address at least:

- preservation of current JustVoxel `/etc` configuration outside the deployment-specific state
- a verified Minecraft backup before rollback
- player-aware graceful shutdown
- controlled bootc rollback/reboot
- post-boot detection of the rollback event
- comparison/reconciliation of current versus restored JustVoxel configuration
- clear recovery behavior if the rolled-back OS and newer local configuration are incompatible

Because of those requirements, rollback should be treated as a separate later project after normal deployment, update, storage, backup, VM, and Bare Metal flows are proven stable.

## Near-term implementation order

The current priority remains validation rather than immediately adding the roadmap features above.

Recommended order:

1. destructive VM testing of the existing setup/storage/migration flows
2. fix and harden anything found during VM testing
3. validate backup retention and failure behavior
4. validate NFS/SMB failure handling
5. validate Minecraft container/version update behavior
6. build and validate the JustVoxel ISO/installation/first-boot experience
7. perform Bare Metal validation
8. implement the nearer `mjust system` health/status/update/power features
9. leave JustVoxel-aware OS rollback for a later dedicated design cycle
