# mjust management layer

`mjust` is the JustVoxel administrator interface. It is a small wrapper around upstream `just`; recipes stay short and delegate system work to scripts under `/usr/libexec/justvoxel/mjust/`.

The interaction model is inspired by Universal Blue `ujust`, but JustVoxel keeps an independent server-focused implementation.

## Immutable templates versus administrator state

The bootc image owns immutable source material under `/usr/share/justvoxel/` and `/usr/libexec/justvoxel/`. `mjust setup` renders administrator-owned runtime files into `/etc`. Bootc image updates do not silently replace those local settings.

## First setup

`mjust setup` collects Minecraft data location, Java heap, container memory limit, Java/Bedrock ports, timezone, player limit, MOTD, container image policy, Minecraft/Paper version policy, backup target, retention, schedule, and explicit EULA acceptance.

Memory suggestions are based on installed RAM but remain adjustable.

## Container image and game version

The container image tag and Minecraft/Paper game version are independent administrator choices.

The itzg container can use `latest`, `stable`, or a validated custom/exact tag. Moving tags are refreshed only when mjust explicitly pulls them. Minecraft can remain pinned to an exact stable Paper-supported version or deliberately use `VERSION=LATEST`.

`mjust configure` can change either policy without reinstalling the appliance and does not automatically restart a running server.

## Storage management

`mjust storage-plan` shows the current variant, block devices, protected system disks, and the recommended storage model.

`mjust storage` opens backup storage management. VM recommends a second virtual disk while still supporting NFS/SMB. Bare Metal supports dedicated internal/USB disks, existing partitions, free-space partition creation, NFS, SMB, and a normal system directory.

`mjust storage-disk`, `storage-partition`, `storage-free-space`, and `storage-network` expose the same storage operations directly. `mjust storage-migrate` performs a guarded Minecraft data migration to local provisioned storage.

Whole-disk erase and partition formatting require exact typed target phrases. Whole-disk operations exclude system disks and mounted disks. New partition creation uses only already-unallocated space and never shrinks an existing filesystem.

Local persistent mounts use UUIDs. Network shares are mounted persistently with `_netdev,nofail`; backup execution validates the expected source before stopping Minecraft.

See `docs/STORAGE.md` for the full safety model.

## Operational recipes

The management surface includes setup, configure, status, start, stop, restart, players, Java and Bedrock whitelist operations, cold backup, Minecraft container update, logs, validation, storage management, and data migration.

Stop/restart and maintenance operations query RCON first. Unknown player state fails closed. If players are online, the administrator can cancel or explicitly continue; the existing 60-second in-game shutdown warning is then used before the server stops.

## Backups and updates

Manual, timer, pre-update, and pre-migration backups share the same verified cold-backup helper and retention count.

`mjust update-minecraft` checks the configured image tag's remote digest, local digest, and running image, creates a cold backup before replacement, keeps one known previous Minecraft image for rollback, and verifies RCON/plugins after restart. It never runs a broad Podman image prune.
