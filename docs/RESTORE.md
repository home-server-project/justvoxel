# Minecraft restore

JustVoxel provides two deliberately separate restore operations.

## Restore world

```text
mjust restore
```

This is the normal rollback operation when gameplay or world state needs to be
returned to an earlier JustVoxel backup.

It restores the primary Minecraft world and its standard Nether/End world
directories. Normal world-based player data such as inventories, positions,
advancements and statistics rolls back with the world.

It does **not** replace current plugins, plugin data stored outside the world
directories, `server.properties`, whitelist/ops, JustVoxel configuration, the
Minecraft Quadlet, container image policy, Minecraft version policy, or the
bootc operating-system deployment.

## Restore full Minecraft data

```text
mjust restore-full
```

This replaces the complete backed-up Minecraft persistent data directory,
including worlds, plugins, plugin data and Minecraft-side configuration.

It still does **not** restore `/etc/justvoxel`, the Minecraft Quadlet, the
container channel/tag, the current Minecraft version policy, or the JustVoxel
OS deployment.

## Safety model

Both restore modes:

- validate the configured backup mount/source identity before discovering files;
- offer only completed `minecraft-*.tar.gz` archives and ignore `.partial` files;
- verify gzip integrity and reject unsafe tar paths/special files before live data is touched;
- prepare restore data on the same filesystem as the current Minecraft data;
- use the shared JustVoxel maintenance lock;
- check players through RCON and use the normal graceful Minecraft shutdown path;
- require the exact typed confirmation `RESTORE`;
- preserve the affected current data as a pre-restore safety copy;
- apply the currently configured Minecraft UID/GID and SELinux labeling;
- start Minecraft and validate RCON, Paper/version reporting, ViaVersion and, when enabled, Geyser/Floodgate;
- automatically roll back to the preserved pre-restore state if the restored server fails validation;
- remove temporary staging and safety-copy state only after successful validation;
- never modify or delete the selected backup archive during restore.

A previous incomplete restore transaction blocks another restore until an
administrator reviews the preserved recovery state.

## Backup metadata

New backups keep the existing archive format and add an optional sidecar:

```text
minecraft-YYYY-MM-DD-HHMMSS.tar.gz
minecraft-YYYY-MM-DD-HHMMSS.tar.gz.meta.json
```

The metadata records restore-useful information such as configured Minecraft
version policy, reported server version when available, container image/digest,
UID/GID, Bedrock/Geyser/Floodgate state and JustVoxel bootc image identity.

Old backups without metadata remain supported.

When metadata shows that a backup comes from an older Minecraft version,
JustVoxel warns that the restored data will still start with the currently
configured version. A backup from a newer pinned Minecraft version is refused
when the current pinned version is older, because opening newer world data with
an older Minecraft version is an unsafe downgrade direction.

## Configuration recovery is separate

Current `minecraft-*.tar.gz` backups contain Minecraft persistent data only.
They do not contain `/etc/justvoxel` or the Minecraft Quadlet. A future
configuration/disaster-recovery feature may snapshot and restore those files
separately; world restore and full Minecraft-data restore intentionally do not.
