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

## Container image versus Minecraft game version

JustVoxel treats these as two independent administrator choices.

The **container image tag** controls the `itzg/minecraft-server` image and its Java/runtime tooling. The setup wizard defaults to upstream `latest`, matching the proven existing deployment, and also offers upstream `stable` or a custom/exact image tag. Upstream `stable` tracks the newest released container image; `latest` tracks the newest main-branch image. An exact release tag can remain pinned. A normal service restart uses the local image already present for that tag; `mjust update-minecraft` explicitly checks the remote digest and pulls when the configured tag has changed remotely.

The **Minecraft/Paper game version** is controlled separately by the container's `VERSION` setting. The recommended JustVoxel policy resolves the newest Minecraft version that currently has a stable Paper build and pins that exact game version. Administrators can instead select another stable Paper-supported version or deliberately choose `VERSION=LATEST`. `LATEST` is allowed, but mjust warns that a later container start/restart can upgrade the Minecraft game version.

`mjust configure` can change either policy without reinstalling the appliance. It does not restart a running server automatically after a policy change.

## First setup

`mjust setup` refuses to overwrite an existing installation. It collects the Minecraft data path, Java heap, total container-memory limit, ports, timezone, player limit, MOTD, container image policy, Minecraft/Paper version policy, backup target, retention, and schedule. It requires explicit Minecraft EULA acceptance before creating an active runtime.

Memory suggestions are calculated from installed system RAM, but both the Java heap and container limit remain administrator-adjustable. The container limit must remain larger than the Java heap.

For Bedrock cross-play, setup installs Geyser and Floodgate through the container's runtime download mechanism, performs the first start, then applies the verified `auth-type: floodgate` setting to the generated Geyser configuration before final validation.

## Storage model

The Minecraft persistent data path is independently configurable. This is the path to place on faster storage such as NVMe when desired. When that path is on a separate mounted filesystem, setup records its mount identity and the generated service fails closed if that filesystem is missing or replaced. Podman's own image store remains a host concern and is not moved by mjust.

Backups can target a normal system directory or an already-mounted dedicated disk, dedicated partition, NFS share, or SMB/CIFS share. For external mounts, mjust records mount identity and the backup helper fails closed if the expected mount disappears.

Automatic disk partitioning, formatting, fstab generation, and world-data migration are intentionally reserved for the next storage implementation step. `mjust storage-plan` documents the supported model but performs no destructive action.

## Operational recipes

The management surface includes setup, configure, status, start, stop, restart, players, Java and Bedrock whitelist operations, cold backup, Minecraft container update, logs, validation, and the non-destructive storage-plan placeholder.

Stop and restart query RCON first and fail closed if player state is unknown. If players are online, the administrator can cancel or explicitly continue. The configured graceful stop then uses the existing 60-second in-game shutdown warning before the server stops and saves.

## Backups and maintenance

Manual backups, timer backups, and pre-update backups all use the same verified cold-backup helper and the same retention value. For example, when retention is seven, creation of the eighth verified archive removes the oldest archive.

During `mjust update-minecraft`, the updater owns the shared maintenance lock and asks the backup helper to leave Minecraft stopped after the verified archive. This avoids the old stop -> backup -> start -> stop -> update -> start pattern. The update path uses one graceful stop and one final start.

## Minecraft container updates

`mjust update-minecraft` checks the configured image tag's remote registry digest, local image digest, and currently running image. A moving tag such as `stable` or `latest` changes locally only after an explicit pull. A pinned/custom tag remains on that tag; mjust also reports the newest upstream image release when available so the administrator can decide whether to change tags through `mjust configure`.

When the Minecraft game version is pinned, the updater also checks PaperMC and reports when a newer Minecraft version with a stable Paper build is available. It does not silently change the pinned game version. If the administrator selected `VERSION=LATEST`, mjust reports that policy clearly instead.

When Minecraft is active, the update path queries RCON and fails closed if player state is unknown. If players are online, the administrator can explicitly continue; the normal 60-second shutdown announcement remains in effect. Player state is checked again immediately before the graceful stop.

The updater then creates the verified cold backup, pulls the configured image when needed, starts Minecraft once, and verifies RCON, Minecraft, plugins, and Geyser when Bedrock support is enabled.

JustVoxel keeps at most one known previous Minecraft container image as a rollback generation. After a later successful image replacement, the older tracked generation is removed when it is not used by a container. JustVoxel does not run a broad Podman image prune and does not remove unrelated images.
