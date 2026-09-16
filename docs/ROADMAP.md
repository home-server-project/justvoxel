# JustVoxel roadmap

This roadmap tracks product direction and future appliance capabilities. It is priority-based rather than date-based: items move forward when the current layer is validated and stable enough to support the next one.

JustVoxel is intentionally a dedicated Minecraft server appliance. The goal is not to become a general-purpose game-hosting panel or generic container platform.

## Product principles

JustVoxel should remain understandable to someone who can install an operating system, create a VM, write an ISO to a USB drive, and follow normal computer instructions without requiring deep Linux administration knowledge.

The appliance is designed to run either in an isolated VM or on dedicated hardware. Minecraft itself remains a separate containerized workload rather than being baked into the JustVoxel operating-system image.

Normal operations should prefer guided, validated workflows over exposing raw Linux commands. Advanced administrators still retain access to the underlying AlmaLinux system when they need it.

Safety remains more important than automation. If JustVoxel cannot reliably determine the state of players, storage, backups, configuration, or another destructive boundary, the normal behavior should be to fail closed rather than guess.

## Current priority: stabilization and validation

The immediate priority is to finish and harden what already exists before expanding the appliance surface.

Current work includes:

- real-world WebUI testing and bug fixing
- VM validation
- Bare Metal validation
- installer and first-boot validation
- backup and restore validation
- destructive and failure-path storage testing
- network-storage failure testing
- Minecraft/Paper update testing
- operating-system update testing
- documentation reconciliation after behavior is proven

Existing capabilities such as scheduled backups, manual backups, restore, storage provisioning and migration, Minecraft updates, bootc OS update staging, status, validation, resources, reboot, poweroff, and firmware handling are implemented features rather than future roadmap items.

## Near-term: one administrator identity by default

JustVoxel should avoid making a home user manage unrelated system and WebUI administrator passwords by default.

The preferred default is one JustVoxel administrator account, normally `voxel`, with the local AlmaLinux account as the authoritative administrator identity. WebUI authentication should use the supported RHEL/AlmaLinux PAM path through the privileged management agent rather than reading `/etc/shadow` directly or maintaining a second synchronized password copy.

The browser-facing WebUI must remain unprivileged. PAM interaction, password changes, session invalidation, and other privileged authentication work belong behind the existing management-agent boundary.

First-boot behavior should be simple: the bootstrap administrator credential is replaced once, and the resulting system credential is the same credential used for local console access, WebUI administration, and SSH password authentication when SSH password login is enabled.

Password policy should follow the normal supported AlmaLinux/RHEL stack rather than introducing a custom JustVoxel password-complexity framework or manually rewriting authselect-managed PAM files. JustVoxel should accept the platform's practical minimum requirement and explain that stronger passwords are recommended without forcing unnecessary application-specific composition rules.

A later optional authentication mode may allow the WebUI administrator password to be separate from the system account. That should be implemented as an explicit alternate authentication provider, not by copying or synchronizing passwords between databases. Future Operator or Viewer accounts should remain WebUI-only identities and should not automatically become Linux users.

## Near-term: player-aware maintenance shutdowns

Maintenance actions should not make an empty Minecraft server wait through a player warning timer, but active players should receive clear in-game notice before interruption.

A shared player-aware maintenance path should be used by operations that can stop Minecraft, including restart, Minecraft update, appliance reboot or poweroff, restore, migration, and other maintenance workflows where applicable.

Desired behavior:

- if Minecraft is already stopped, continue without a warning delay
- if zero players are online, perform the normal clean stop immediately or after only a very short safety delay
- if players are online, show visible in-game maintenance warnings at useful intervals such as 60, 30, 10, and 5 seconds before the clean stop
- if player state cannot be determined reliably, fail closed rather than assuming the server is empty
- avoid duplicated delays between JustVoxel's player-aware logic and the underlying Minecraft container shutdown mechanism

The goal is to make reboot, shutdown, update, and restart fast when the home server is unused while still giving players enough time to finish what they are doing when the server is active.

## Next: existing-server import and portable export

The highest-priority new capability after stabilization is safe migration into and out of JustVoxel.

### Import an existing Minecraft server

JustVoxel should be able to adopt an existing Minecraft/Paper server without requiring the user to manually rebuild the world, plugins, configuration, whitelist, and related persistent data.

The intended import workflow should support safe sources such as:

- an archive supplied by the administrator
- attached USB or other local storage
- supported network storage

Import should inspect and stage the source before activation. It should preserve the original source, identify the important Minecraft data that was found, normalize ownership and SELinux state where required, validate the staged server, and only activate it after the import passes the required checks.

A failed import must not silently replace a working JustVoxel installation.

A real existing Minecraft server should be used as an acceptance case in a disposable JustVoxel VM before this feature is considered proven.

### Export / migration package

JustVoxel should also be able to produce a portable, validated export suitable for migration, disaster recovery, or transfer to another JustVoxel installation.

The export format should make clear what is included and should not depend on the original appliance continuing to exist.

Import and export should share as much validation and archive-handling logic as practical so the appliance has one understandable migration model rather than several incompatible ones.

## After migration: notifications and event history

JustVoxel should provide useful appliance-level notifications without becoming a full monitoring platform.

Examples include:

- backup failure
- low storage space
- repeated Minecraft service failures
- OS update availability
- Minecraft/Paper update availability
- import, export, or restore completion/failure

Initial notification targets should stay simple, such as Discord and a generic webhook mechanism.

Event history should complement notifications by keeping short appliance-relevant records such as repeated Minecraft restarts, the last failure time, whether automatic recovery succeeded, and concise diagnostic context.

Systemd should remain the normal service-recovery mechanism rather than adding a second custom watchdog solely for Minecraft restarts.

## Later: local documentation in CLI and WebUI

JustVoxel documentation should remain available on the appliance even when the Internet is unavailable.

The repository Markdown documentation should remain the single source of truth and continue to ship with the matching JustVoxel system image. A future `mjust docs` experience should make the local documentation easy to browse or search from the terminal.

The WebUI should provide a Help or Documentation area that renders those same bundled Markdown documents as a readable local documentation site with normal navigation, headings, links, tables, notes, and warnings rather than exposing raw Markdown text.

Documentation should follow the installed bootc generation so an operating-system update brings the matching documentation and an operating-system rollback returns to the matching documentation. A separate PDF documentation set is not required unless a later use case justifies one.

## Later: operator access

A later WebUI capability should allow limited non-administrator access without turning JustVoxel into a hosting platform.

A simple model is preferred:

- **Administrator** — full appliance control
- **Operator** — safe Minecraft-level operations such as status, players, whitelist, start/restart, and backups

Operator access should not allow storage reconfiguration, OS administration, appliance reset, firmware operations, or host power controls unless explicitly expanded in a future design.

## Later: safe scheduling improvements

Scheduled backups already exist and are not a future feature.

Future scheduling work should focus only on safe appliance-defined maintenance operations where they add clear value, for example:

- scheduled Minecraft restart
- optional maintenance reboot
- update checks
- defined maintenance windows

The normal user interface should stay focused on predefined appliance operations rather than becoming a general-purpose task scheduler.

## Later: plugin management

Plugin management may eventually provide a guided way to discover, install, update, disable, or remove compatible Paper plugins from trusted sources.

Any implementation should preserve the appliance model by checking compatibility where possible, creating appropriate safety backups before risky changes, and avoiding unrelated host-level software installation behavior.

This is intentionally later work because plugin compatibility creates a much larger support and validation surface than the core appliance.

## Nice to have: JustVoxel-aware system rollback

JustVoxel already provides bootc operating-system status and update staging through `mjust`.

A future `mjust` rollback workflow may provide a safer appliance-aware way to return to the previous JustVoxel bootc deployment instead of requiring the user to work directly with low-level bootc rollback operations.

This is deliberately not a near-term priority. A correct rollback design must consider more than selecting an older OS deployment. It must also define how deployment-specific `/etc` state, active JustVoxel configuration, Minecraft data, and recovery/backup boundaries interact so that rolling the operating system backward does not create an inconsistent appliance state.

Until that design exists, bootc-level rollback remains an advanced native-system operation rather than an automated JustVoxel workflow.

## Non-goals

JustVoxel is not intended to become:

- a multi-game hosting panel
- a generic Podman or Docker manager
- a billing or commercial hosting platform
- a multi-node game-server orchestration system
- a general Linux management panel
- a network infrastructure management system

Supporting more features is not automatically an improvement. New capabilities should be added only when they make a dedicated Minecraft appliance safer, easier to operate, easier to migrate, or easier to recover.

## Roadmap rule

Implemented behavior belongs in the operational documentation. Future behavior belongs here.

When a roadmap item becomes implemented and validated, its detailed behavior should move to the appropriate operational document rather than leaving duplicate or stale future descriptions in this file.
