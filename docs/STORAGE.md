# JustVoxel storage model

JustVoxel does not hardcode an active Minecraft data directory or backup directory into the image. Shipped paths are only setup defaults; the administrator chooses the actual paths before `mjust setup` renders the runtime.

## Minecraft data

Minecraft worlds, configuration, plugins, whitelist data, Geyser/Floodgate state, and other server state live outside the immutable OS deployment and are bind-mounted to `/data` in the Minecraft container.

The data path can be on the system filesystem or on another already-mounted filesystem. This is the normal way to place the Minecraft world on a faster NVMe while the bootc OS lives on a SATA SSD. JustVoxel does not relocate Podman's global image store as part of Minecraft setup.

When the data path uses a separate mounted filesystem, `mjust setup` records that mount's UUID and/or source. The generated Minecraft service validates the mount before startup so a missing NVMe or other data filesystem cannot silently redirect the server into an empty directory on the system disk.

## Backup targets

The backup engine supports a configurable local path and optional mount validation. A configured mount can represent a dedicated disk, a dedicated partition, an NFS share, an SMB/CIFS share, or another mounted filesystem.

If no backup mount is configured, the backup path is treated as a normal directory on the system filesystem. If a backup mount is configured, the helper validates that the mount is actually present before it creates a backup directory or stops Minecraft. This prevents a missing external disk, partition, or network share from silently redirecting archives onto the system filesystem.

For local dedicated storage, mjust records the expected filesystem UUID. For network storage, it records the expected source reported by `findmnt`. The backup helper verifies those values on every run.

The backup helper performs a cold backup: it gracefully stops Minecraft if it was running, archives the complete data directory to a `.partial` file, verifies the archive, atomically publishes it, applies retention, and starts Minecraft again only if the helper stopped it. A shared `flock` lock prevents backup and Minecraft update maintenance from overlapping.

## Step 3 limits

`mjust setup` can use existing mounted filesystems. It does not partition a disk, format a filesystem, write `/etc/fstab`, or migrate an existing world between storage devices.

A later storage step will add guarded discovery and provisioning for dedicated disks and dedicated partitions. That work must identify the system disk, display the exact destructive target, require explicit confirmation, use filesystem UUIDs, mount and validate the target, and only then update JustVoxel configuration.
