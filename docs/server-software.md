# Minecraft server software

Add Server Simple Setup uses three states in the existing card: family, software,
then server name and game mode. It leads to the shared Review page. Advanced Setup
retains Server, Cross-play, Memory, Storage, Backups and Review, with software
grouped by capability. All ten Java server types and native Bedrock are selectable.

Add Server offers three family lanes: Plugins (Paper, Purpur, Spigot, Pufferfish, Leaf), Mods
(Fabric, NeoForge, Forge, Quilt), and Vanilla Java · Bedrock. Advanced Setup uses
these same three software lanes in its Server step.

The Minecraft Servers Add Server action opens or reuses one dedicated wizard tab
and starts fresh. Refreshing that wizard preserves the current step. A submitted
or partially completed creation retains its existing Review/recovery path.
Cancel and Review's Return to Minecraft Servers clear the Add Server draft and
review, focus the original tab when available, and try to close the wizard tab.
If the browser prevents closing, the wizard shows a message and Close tab button.

Setup resolves an exact stable Minecraft release. Spigot derives its JDK image
from that release: java21-jdk for 1.20.5–1.21.11 and java25-jdk for 26.x. Mod loaders
resolve a stable exact loader/build from their publisher metadata and verify the
exact game/loader pair using the existing upstream artifact verifier. The draft
carries these pins through Review/Create. No manual loader control is needed.
New server types use a pinned runtime; this does not enable ongoing game updates.

Leaf Setup and Version use verified JARs from the official Winds-Studio/Leaf
GitHub releases. Recommended selects the newest production Minecraft version.
Latest selects the numerically newest verified version, including early
development builds. Specific lists verified versions with their Leaf status.
Bedrock compatibility is checked against the selected Minecraft version.

Simple Setup enables managed cross-play only when compatibility is proven for its
exact runtime. Missing cross-play evidence leaves Java setup available with
cross-play disabled. Advanced Setup validates an enabled cross-play choice
separately; disabling it permits a valid Java server to continue. Fabric and
NeoForge require exact platform compatibility; Forge, Quilt and Vanilla are Java
only. Existing storage, backup and port safety remain authoritative.

Advanced Setup shows the current managed cross-play Minecraft version even with
cross-play off for other Java software. Leaf checks compatibility when cross-play
is enabled. Other valid Minecraft selections remain available as Java-only;
Latest and Specific are never silently downgraded. Existing Recommended automatic
cross-play fallback remains unchanged. If current Geyser metadata cannot be
verified, Java-only setup remains available. Fabric and NeoForge previews resolve
the exact Java runtime without downloading cross-play artifacts. Enabling their
cross-play toggle verifies the exact runtime and managed artifacts; failed proof
keeps the toggle enabled and checked and blocks Continue until compatibility is
proven or the user disables cross-play.

| Software | Plugins & Mods workspace | Managed Bedrock cross-play |
| --- | --- | --- |
| Paper, Purpur, Pufferfish, Leaf | Plugins: Modrinth, Hangar, configured CurseForge | Existing Spigot-family implementation |
| Spigot | Plugins: Modrinth and configured CurseForge with Bukkit compatibility | Spigot-family Geyser/Floodgate with managed ViaVersion |
| Fabric | Mods: Modrinth and configured CurseForge | Only a proven current Geyser-Fabric game/loader pair, Floodgate-Fabric and exact-version Fabric API |
| NeoForge | Mods: Modrinth and configured CurseForge | Only a proven current Geyser-NeoForge game/loader pair and Floodgate-NeoForge |
| Forge, Quilt | Mods: Modrinth and configured CurseForge | Unavailable |
| Vanilla | Unavailable | Unavailable |
| Native Bedrock | Unavailable | Native Bedrock clients; no Geyser/Floodgate |

Spigot's Hangar provider stays unavailable: the current Hangar platform mapping
only proves Paper compatibility. Modrinth releases must explicitly include
`spigot` or `bukkit` and the exact Minecraft version; installation also requires
a usable Bukkit `plugin.yml` descriptor. CurseForge exact file platform markers
must provide positive evidence for Compatible only; absent evidence stays unknown
and requires artifact validation. Platform inheritance and exact release checks
apply independently of Minecraft release number. Spigot proves installed plugin
activation from live RCON plugin inventory; the Paper family retains strict
PluginInitializerManager startup-log proof.

Geyser-Fabric and Geyser-NeoForge downloads come only from the fixed official
GeyserMC download API. Their publisher SHA-256 hashes are checked before replacement.
Floodgate for both loaders comes only from Modrinth project `bWrNNfkb`, selecting
the newest listed release for the exact game and loader. Its publisher SHA-512
is verified before recording a local SHA-256 for installed-file ownership checks.
Compatibility requires agreement between Geyser's published current Java release
and dependency metadata inside the verified platform artifacts, including the
selected loader pin. Missing, unsupported or contradictory evidence blocks
cross-play. Older releases never gain a Bukkit ViaVersion plugin or another proxy.

Fabric API uses fixed Modrinth project `P7dR8mSH`, explicit Fabric loader metadata,
and the exact game release. Only a publisher release with one primary artifact
and its SHA-512 hash can be selected. Geyser, Floodgate and Fabric API are staged
as one set before replacement in `mods`; no mod is placed in `plugins`.

Managed mods have root-private per-instance ownership evidence. Unowned canonical
files, modified owned files and duplicate managed mod identities block replacement.
Interrupted replacement retains rollback evidence; disabling cross-play removes
only verified owned Geyser/Floodgate binaries and preserves configuration, identity
material and Fabric API, which other mods may also need.
The existing Podman data-mount labeling model remains unchanged.

The configured Geyser numeric port is reserved and published over **both TCP and
UDP**, with firewall ownership tracked for both protocols. It cannot share a TCP
port with a Java server. The primary Geyser transport is NetherNet; this change
does not allocate another WebRTC port or install a signaling service.

Source checks do not establish runtime compatibility or visual acceptance. Verify
these changes in the development VM before release.

Publisher references used by this implementation:

- [Geyser supported versions](https://geysermc.org/wiki/geyser/supported-versions/)
- [Geyser setup, platform configuration and TCP/UDP requirements](https://geysermc.org/wiki/geyser/setup/)
- [Official Geyser download API](https://geysermc.org/wiki/api/downloads.geysermc.org/build-latest/)
- [Floodgate-Modded publisher project](https://modrinth.com/mod/floodgate)
- [Modrinth project-version API](https://docs.modrinth.com/api/operations/getprojectversions/)
- [Fabric API publisher project](https://modrinth.com/mod/fabric-api)

## Native Bedrock

Bedrock is a separate server runtime using
`docker.io/itzg/minecraft-bedrock-server:stable`, with per-instance persistent
storage mounted at `/data`. It has no Java image, Java heap, plugins, mod loaders,
or managed Geyser/Floodgate cross-play. Survival, Creative and Adventure are
available as initial game modes. Servers remain open to authenticated Bedrock
players; JustVoxel does not currently provide a Bedrock allow-list workflow.

Setup verifies the official Minecraft Services `serverBedrockLinux` release URL
and pins its exact numeric version. Preview releases and moving `LATEST` versions
are refused. An unavailable or changed release requires reviewing setup again.
Restarting a Bedrock container keeps the selected game version.

NetherNet uses one unique TCP signaling port (first candidate 19132) and a unique
contiguous UDP gameplay range (first candidate 19140), with at least one UDP port
per maximum player. Both active listeners and stopped instance reservations are
checked. Native containers use host networking so Bedrock advertises appliance
addresses without a guessed NAT address. LAN broadcast discovery is disabled;
connect using the appliance address and the server's TCP port. Routers and
external firewalls must allow the TCP port and complete UDP range as well.
JustVoxel tracks ownership of each firewall port, preserves pre-existing rules,
and removes only operation/instance-owned rules during rollback or deletion.

Simple Setup selects Bedrock through family, software, then name/game mode and
uses the shared Review. Advanced Setup retains six steps; its Cross-play step
shows the native connection and exact release, and Memory offers only a container
limit. The existing identity, lifecycle, boot preference, backup inventory and
cold-backup/transactional restore paths apply. Restore requires the same instance,
Bedrock software and exact version, and validates Bedrock world headers without
using Java NBT metadata. Java/Bedrock conversion is unsupported.

## Appliance timezone

Minecraft setup and Review have no independent timezone setting. New instance
configuration does not store one. Java and Bedrock containers inherit appliance
time through a read-only `/etc/localtime` mount; change the authoritative timezone
under System → Date & Time. Legacy timezone fields remain readable for deployed
records and recovery journals but are not used to generate new runtimes.

Upstream contracts:
[Bedrock image configuration](https://github.com/itzg/docker-minecraft-bedrock-server#nethernet),
[official release metadata](https://net.web.minecraft-services.net/api/v1.0/download/links).


## Software conversion in Server Migration

**Server Migration → Switch** replaces Minecraft Update's former Switch tab.
Minecraft Update contains Container and Version only. Switch shares Setup's
catalogue and artwork, keeping all eleven cards visible: Paper, Purpur, Spigot,
Pufferfish, Leaf, Fabric, NeoForge, Forge, Quilt, Vanilla Java and native Bedrock.
Current is yellow and not selectable; Available offers a review of at least one
implemented action; Not Supported is dimmed, disabled and remains visible.

The supported Paper family is **Paper / Purpur / Pufferfish / Leaf**, in all
directions where the exact game/runtime and required cross-play can be verified.
Vanilla-to-Paper-family and Vanilla-to-loader use the target runtime's normal
world loading. Paper-to-Vanilla and Paper-to-loader use verified, version-aware
layout relocation under the configured `level-name`: Overworld data files on
26.1+, and Nether/End directories on supported pre-26.1 releases. Unverifiable
or conflicting layouts block those actions.
Software conversion never upgrades or downgrades Minecraft. An unavailable exact
target build blocks conversion. The target remains at the reviewed exact pin;
Minecraft Version is the separate operation for release and policy changes.

Convert preserves the same immutable instance and its normal placement, ports
and backup destination. Clone and convert creates a separate immutable instance
with a new data directory under the same root, the same backup destination and
safe new ports allocated automatically. It preserves the original, uses a
consistent cold source copy and meaningful exact-runtime verification, and leaves
a successful clone stopped. Clone names derive from source name and target
software and can be renamed normally. Failures use existing transaction recovery
and cannot become successful merely because a container started.

Incompatible and unknown/unverified content blocks in-place Convert. Review
lists named blockers and links to the selected server's Plugins & Mods workspace.
Provider/catalogue failure does not mean compatible. No plugins or mods are
automatically removed, disabled, replaced, updated or downloaded by Switch.
Reopening or refreshing Switch re-evaluates current inventory. An experimental
clone may remain available when Convert is blocked, while preserving the original.
A target is Available whenever Convert or protected Clone is available.

Fabric/Quilt, Forge/NeoForge and other Java loader transitions require verified
installed bytes, exact game/target-loader metadata and verified required
dependencies for Convert. Unknown or incompatible content blocks Convert but
leaves a protected experimental clone available when its exact runtime exists.
Plugins never become mods, and mods never become plugins. Blockers must be handled
manually in Plugins & Mods. Leaving a mod loader for Vanilla or Paper offers only
a clone: historic mod-added blocks/items/entities may not survive, even if the
current mods folder is empty. Meaningful target runtime verification is required.

Spigot remains unsupported in this scope: Paper-to-Spigot is unsafe, and direct
Spigot-to-Paper from 26.1 has a different world layout. No intermediary conversion
is attempted. Java and native Bedrock remain unsupported in both directions.

The new conversion directions retain the startup content guard to avoid
unrequested downloads. Supported target cross-play settings are preserved; review warns when the
target requires Java connections. Existing same-Paper-family cross-play behavior is
preserved. Failed in-place conversions restore their cold archive through the
existing restore transaction before restarting the original. Interrupted recovery
retains the same archive authority and never retries conversion.

Upstream world-layout reference:
[Paper migration guidance](https://docs.papermc.io/paper/migration/).
