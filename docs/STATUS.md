# JustVoxel status dashboard

`mjust status` is the normal user-facing health dashboard for a JustVoxel appliance.

JustVoxel intentionally does not provide a second `mjust health` dashboard. Health information belongs in `mjust status`, while `mjust validate` remains the stricter correctness check used to prove that the appliance matches its expected configuration and runtime state.

## Normal status

Normal `mjust status` uses short appliance-oriented sections instead of raw Linux command output:

- overall health
- host identity, variant, uptime, physical RAM, and failed-service summary
- Minecraft state, players, Paper version, Bedrock cross-play, addresses, memory and uptime
- primary network interface, IPv4 address, gateway and DNS
- system/container storage, Minecraft data storage, backup storage and available capacity
- automatic backup state, most recent completed backup and next scheduled backup
- configured Minecraft container channel and image

Unavailable optional information is shown as `Unknown` instead of dumping command errors.

A deliberately stopped Minecraft server is reported as `Stopped` and does not by itself make the appliance unhealthy.

## Overall health

`Overall: Healthy` is a lightweight operational summary. It does not run `mjust validate` internally.

The normal status view uses cheap local observations such as:

- failed system services
- failed Minecraft service state
- basic network/default-route availability
- configured data/backup storage availability
- configured backup timer state

Problems are summarized as `Attention needed`. When the system state cannot be determined reliably, status uses `Unknown`.

## Failed services

Failed services are part of the normal status dashboard rather than a separate normal-user command.

When no service has failed, status reports:

`System services: Healthy - no failed services`

When failures exist, status shows the number of failed services and a short summary for up to three units: unit name, description and result. Additional failures are left to `mjust status --details` so the normal dashboard stays concise.

## Storage presentation

Status is purpose-aware rather than a raw `df` listing.

It identifies system/container storage, Minecraft data and backup storage. If Minecraft data shares the system/container filesystem, that relationship is stated instead of repeating capacity rows.

If backups share the Minecraft-data filesystem, capacity is not printed twice. Status explains that same-filesystem backups help with world/configuration recovery but do not protect against physical disk failure.

A separate disk, NFS or SMB filesystem is shown as its own backup target with its own available capacity.

## Administrator detail

`mjust status --details` prints the same friendly dashboard first and then adds administrator-oriented information including:

- complete failed-service output
- Minecraft systemd status
- Podman container and memory detail
- raw mount/filesystem detail
- interface, route and resolver detail

`mjust logs` remains the dedicated Minecraft log viewer.

`mjust validate` remains separate from both status modes. Validation is allowed to be stricter, perform more checks, return failure, and explain configuration/runtime mismatches.

## Roadmap boundary

The earlier ideas for a separate `mjust health`, a separate friendly failed-service screen, and a duplicate `mjust system` health dashboard are consolidated into `mjust status`.

The next system-management block remains separate and is intended to cover:

1. bootc running/staged/rollback deployment status
2. non-disruptive OS update check and staging
3. player-aware backup/graceful reboot into a staged OS update
4. player-aware reboot and power-off controls
5. Bare-Metal-only reboot into firmware setup when supported

JustVoxel-aware bootc rollback remains a later dedicated design because it also requires safe handling of deployment-specific `/etc` state.
