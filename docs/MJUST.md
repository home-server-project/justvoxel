# mjust management layer

`mjust` is the JustVoxel administrator interface. It is a small wrapper around upstream `just`; recipes stay short and delegate system work to scripts under `/usr/libexec/justvoxel/mjust/`.

The interaction model is inspired by Universal Blue `ujust`, but JustVoxel keeps an independent server-focused implementation.

## Immutable templates versus administrator state

The bootc image owns only immutable source material:

- `/usr/share/justvoxel/templates/`
- `/usr/share/justvoxel/mjust/`
- `/usr/libexec/justvoxel/`

`mjust setup` renders administrator-owned copies into mutable `/etc` locations:

- `/etc/containers/systemd/minecraft.container`
- `/etc/justvoxel/justvoxel.conf`
- `/etc/justvoxel/minecraft.env`
- `/etc/justvoxel/minecraft-backup.env`
- `/etc/systemd/system/minecraft-backup.service`
- `/etc/systemd/system/minecraft-backup.timer`

The active files are copies, not symlinks back into `/usr/share`. A bootc update may update the shipped templates and mjust implementation, but it must not regenerate or overwrite an installed machine's `/etc` configuration automatically.

## First setup

`mjust setup` refuses to overwrite an existing installation. It collects the Minecraft data path, memory, ports, timezone, player limit, MOTD, pinned Minecraft version, backup target, retention, and schedule. It requires explicit Minecraft EULA acceptance before creating an active runtime.

The setup resolves the newest Minecraft version that has a stable Paper build through PaperMC's current downloads service and lets the administrator keep that exact version or enter another exact version. `LATEST` is rejected for the game version so a container-image refresh cannot silently upgrade the world format.

For Bedrock cross-play, setup installs Geyser and Floodgate through the container's runtime download mechanism, performs the first start, then applies the verified `auth-type: floodgate` setting to the generated Geyser configuration before final validation.

## Storage model

The Minecraft persistent data path is independently configurable. This is the path to place on faster storage such as NVMe when desired. When that path is on a separate mounted filesystem, setup records its mount identity and the generated service fails closed if that filesystem is missing or replaced. Podman's own image store remains a host concern and is not moved by mjust.

Backups can target a normal system directory or an already-mounted dedicated disk, dedicated partition, NFS share, or SMB/CIFS share. For external mounts, mjust records mount identity and the backup helper fails closed if the expected mount disappears.

Automatic disk partitioning, formatting, fstab generation, and world-data migration are intentionally reserved for the next storage implementation step. `mjust storage-plan` documents the supported model but performs no destructive action.

## Operational recipes

The initial management surface includes setup, configure, status, start, stop, restart, players, Java and Bedrock whitelist operations, cold backup, Minecraft container update, logs, validation, and the non-destructive storage-plan placeholder.

Stop and restart query RCON first and refuse to interrupt an unknown player state. If players appear online, mjust requires an explicit confirmation.

## Minecraft container updates

`mjust update-minecraft` tracks the remote registry digest, local tag digest, and image currently running. It distinguishes an update that needs to be pulled from a newer image that is already local but not yet running.

When Minecraft is active, the update path queries RCON and fails closed if player state is unknown. It refuses normal updates while players are online, rechecks immediately before maintenance, creates a verified cold backup, takes the shared maintenance lock, checks players again, updates or restarts the container, then verifies RCON, Minecraft version, plugins, and Geyser when Bedrock support is enabled.

The pinned Minecraft game version is not changed by this container-image update operation.
