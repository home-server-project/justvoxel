# JustVoxel storage model

JustVoxel does not hardcode a Minecraft data directory or backup directory into the active runtime. Paths are administrator choices rendered later by mjust.

## Minecraft data

Minecraft worlds, configuration, plugins, whitelist data, Geyser/Floodgate state, and other server state must live outside the immutable OS deployment and be bind-mounted to `/data` in the Minecraft container.

## Backup targets

The Step 2 backup helper supports a configurable local path and optional mount validation. A configured mount can be a dedicated disk, a dedicated partition, an NFS share, an SMB/CIFS share, or another mounted filesystem.

If no backup mount is configured, the backup path is treated as a normal directory on the system filesystem. If a backup mount is configured, the helper validates that the mount is actually present before it creates the backup directory or stops Minecraft. This prevents a missing external disk or network share from silently redirecting backup archives onto the system filesystem.

For local dedicated storage, mjust can later record an expected filesystem UUID. For network storage, mjust can later record the expected source such as an NFS export or SMB share. These identity checks are optional but supported by the backup helper.

The backup helper performs a cold backup: it gracefully stops Minecraft if it was running, archives the complete data directory to a `.partial` file, verifies the archive, atomically publishes it, applies retention, and starts Minecraft again only if the helper stopped it. A shared flock lock prevents backup and future update maintenance from overlapping.

## Future mjust storage choices

The planned setup flow will expose at least four choices without changing the backup engine: use a directory on the system filesystem, use a dedicated disk, use a dedicated partition, or use a mounted network share such as NFS or SMB/CIFS. Destructive disk or partition provisioning is intentionally not part of Step 2.
