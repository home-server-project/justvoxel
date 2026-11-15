# Discord connection

The primary WebUI administrator configures a dedicated Discord bot in the existing
**AI & API Connections** modal. Save its token, verify it, use **Add bot to Discord
server**, then enable Discord. Refresh to discover servers, choose one, and press
**Use this server**. JustVoxel supports one bound Discord server and never switches
it automatically. If access is lost, restore the bot's access or explicitly bind
another available server.

In the bound Discord server, use **/jv** for an ephemeral interactive control panel
or **/help** for usage and access information. These are guild-scoped commands;
there are no global commands or DM controls. Choose a registered Minecraft server
by its friendly name, then use **Status**, **Players**, or **Refresh**. Player counts
come from the existing Management Agent player endpoint. If more than 25 servers
exist, the select shows the first 25 sorted by friendly name and instance ID;
additional servers remain available in JustVoxel WebUI.

Any bound-server member has read-only access. **Request admin access** records a
pending request. In the existing Discord connection surface, the primary WebUI
administrator can approve or reject requests and remove approved administrators.
Discord user IDs identify people; names are presentation only. Discord roles do
not grant management authority. Pending requests and approved administrators are
each limited to 32.

Approved Discord administrators can **Start**, **Stop**, **Restart**, or create a
manual **Backup** of a selected registered server. Every action requires a
confirmation. Stop and Restart first ask the authoritative lifecycle handler with
player interruption unconfirmed. If it reports players online, a second explicit
confirmation is required. Confirmations expire after two minutes, are single use,
and bind the user, guild, configuration generation, instance, and action. Cancel
consumes the confirmation. Restarting the Management Agent invalidates outstanding
confirmations. Manual backup progress and recovery details remain in WebUI.

Every interaction rechecks current enabled/verified state, runtime, bound guild,
and Factory Reset state. Every management action rechecks current approval and
registered instance identity. Old controls cannot grant authority after removal,
rebinding, credential replacement, disconnect, or Factory Reset. DMs and other
guilds are denied. Factory Reset blocks interactions and revokes Discord state.

Disabling and enabling retains approvals when credentials and binding are
unchanged. Replacing the token, disconnecting, or binding a different Discord
server clears requests and approvals. Disconnect removes all Discord settings.
The private identity database stores the credential, bounded configuration,
administrator approvals, and community access requests; interaction tokens and messages are never persisted.

Discord runs inside the existing Management Agent with **Guilds intent only**.
No Message Content, Guild Members, Presence, or message intents are required.
JustVoxel does not read ordinary messages or channel history. No Discord roles,
message content, or interaction tokens are returned to the WebUI. Failures shown
in Discord use safe fixed messages rather than internal errors or command output.
Browser state is refreshed using the existing Refresh action, without polling.

## Community Minecraft access

In **/jv**, select a Minecraft server and press **Request Minecraft Access**.
Choose **Java Edition** or **Bedrock / Xbox**, then enter the one Minecraft
identity requested by the Discord modal. Cancel leaves no request. Approved
JustVoxel Discord administrators can also request Minecraft access for themselves.

Discord membership, Minecraft whitelist access, and JustVoxel administration are
separate. Submitting a community request does **not** automatically whitelist the
player or grant JustVoxel administration. **Request admin access** remains the
separate primary WebUI administrator approval workflow described above.

Java entry asks for **Minecraft Java username**. Surrounding whitespace is trimmed;
the username must contain 1–16 ASCII letters, numbers, or underscores. This checks
syntax, not ownership. Bedrock entry asks for **Xbox gamertag or Floodgate UUID**.
It trims surrounding whitespace, removes one optional leading `.`, and accepts a
nonempty identity up to 64 characters without control characters or newlines.
Bedrock identities do not use Java username restrictions. No external Mojang,
Microsoft, Xbox, or Geyser identity lookup dependency is added.

Currently approved JustVoxel Discord administrators use **Access requests** in
**/jv**, select a pending request, and choose **Approve** or **Reject**. Review
shows at most 25 requests, oldest first; review those before newer requests when
the list is full. Approval invokes the existing authoritative Management Agent
`whitelistChange` handler for the request's exact immutable server ID, edition,
and stored identity, with `action=add`. That handler owns runtime validation,
idempotency, Java whitelist changes, and Bedrock/Floodgate resolution. An account
already on the whitelist counts as success. Rejection performs no whitelist
operation and never removes existing Minecraft access.

If Floodgate cannot resolve a Bedrock identity, the request remains pending. Ask
the player to join through Geyser once or provide the Floodgate UUID, then try
Approve again. Other whitelist failures also leave requests pending for retry;
Discord does not display raw backend errors or output. A deleted server cannot
be approved or redirected to another server.

A user may have at most **3 pending requests**, with **64 pending requests** across
the appliance. A second pending submission for the same user, guild, immutable
server ID, and edition returns the existing request. Pending requests expire after
**7 days**. Approved and rejected results remain visible for **24 hours** after
the decision and do not count against pending limits. Expired records are removed
opportunistically on store access; there is no cleanup timer or background worker.

Opening **/jv** shows up to five of your own recent requests, newest first, with
Pending, Approved, or Rejected status. After a decision, JustVoxel attempts one
sanitized message in the originating Discord channel with mentions disabled. If
that send fails, the saved decision still stands and remains visible in **/jv**;
there is no notification queue or background retry.

Requests persist in a dedicated table in the existing private root-owned SQLite
database across Management Agent restarts and Discord disable/enable. Replacing
the credential, disconnecting completely, binding a different guild, or Factory
Reset clears community requests as part of the durable transition. Saving the
same guild retains them. Generation and guild checks independently fence stale
controls and records; cleanup failure prevents a successful trust-change result.
Factory Reset blocks request submission, review, decisions, and notifications.
No interaction token, Discord role, message content, or raw backend failure is
stored with a request. This remains Guilds intent only, with no Message Content
intent, ordinary message reading, or channel-history reading.


## Advanced administration

Only currently approved JustVoxel Discord administrators can open **Advanced**
for a selected Minecraft server, or **Minecraft Network** and **System** from
**/jv**. Ordinary community members keep the same status, player, Minecraft access,
and admin access request controls. Advanced reads and actions recheck current
approval, enabled and verified state, bound guild, runtime generation, Factory
Reset exclusion, and the exact registered immutable instance ID where applicable.
Old buttons fail closed. Discord remains an interaction surface over the existing
Management Agent operations, with no second management backend or WebUI changes.

**Advanced** provides **Server details**, **Backup inventory**, **Backup status**,
**Start on Boot**, **Minecraft updates**, **Container updates**, and **Back**.
Details show a small public configuration summary. Backup inventory displays at
most ten entries, newest first, without archive paths. Backup status shows recent
attempt/success and restart requirements. Manual Backup retains its existing
behavior. Start on Boot shows the current preference and requires confirmation
of the exact desired state; a newer preference cannot be overwritten by a stale
review.

Minecraft and container update status use the existing authoritative status
handlers. **Review Minecraft update** reviews the server's current configured
selection; Discord cannot choose arbitrary versions, images, URLs, or policies.
**Review container update** always requests a restart after update and a backup
before restart. Both require explicit confirmation, store the reviewed fingerprint
only on the server, and let the existing update handler revalidate it before
acceptance. Changed update information requires a fresh review. No restore image
selection or retention changes are exposed. **Update progress** reads the existing
operation journal's bounded status; detailed diagnostics and recovery remain in
JustVoxel WebUI. Discord does not retry updates or acknowledge failed operations.

**Minecraft Network** means only the JustVoxel **Minecraft Network / Velocity**
entrance and its member servers. It does **not** configure LAN, Ethernet, Wi-Fi,
firewall, DNS, routes, VPNs, or remote access. The panel shows enabled state,
Java/Bedrock ports, selected member count, bounded server names/IDs, recovery
requirements, and runtime readiness. Enable and Disable require confirmation.
**Change members** presents at most 25 eligible/current servers sorted by friendly
name and immutable ID; additional membership management is available in WebUI.
Each reviewed action adds or removes one member, with the complete resulting
membership and configuration fingerprint stored server-side. Disable Network and
stop affected servers before changing membership. The shared backend transaction
retains compatibility, publication, maintenance, and recovery checks. If the
configuration changed, review it again.

**System** offers **OS updates**, **Check OS updates**, **Download & stage update**,
and **Reboot options**. OS status summarizes running, staged, and available
versions, check state, reboot requirements, and activation state. Checking
availability may take around two minutes and needs no destructive confirmation.
Downloading and staging requires confirmation and can take around 15 minutes.
Discord returns immediately after accepting the download invocation: use
**System → OS updates** to refresh authoritative state. The bounded background
invocation uses the existing system update handler and locks, retains no
interaction token, and attempts one fixed completion message in the originating
channel only while the same Discord generation and guild remain usable. Mentions
are disabled; send failure changes nothing. There is no download retry queue,
second update engine, service, or separate durable download journal.

Reboot options always offer **Ordinary reboot**. When an update is staged and a
reboot is required, the exact four additional choices are:

- **Update reboot · 60s warning · backup first**
- **Update reboot · 60s warning · no backup**
- **Update reboot · 10s warning · backup first**
- **Update reboot · 10s warning · no backup**

Every reboot requires deliberate confirmation. The reviewed staged digest,
warning, and backup choice stay server-side; a changed staged deployment requires
a fresh review. For reboot and update operations, when the authoritative handler
returns a players-online confirmation requirement, Discord asks a **second explicit
player-interruption confirmation**, bound to the same user, guild, generation,
instance or system operation, fingerprint/digest, and reviewed options. Discord
does not discover players independently or bypass the existing safety contract.
Existing Minecraft/container handlers may accept an operation whose player safety
is handled by their authoritative worker rather than a preflight confirmation.

Before invoking a reboot, JustVoxel saves a singleton verification record in the
existing protected SQLite database (migration 10, `discord_reboot_state`). If that
save fails, Discord does not request a reboot. The record stores boot identity,
expected deployment where applicable, Discord trust identity, originating channel,
request time, operation, and acceptance state. It contains no tokens or backend
output. A players-online response clears the unaccepted record before the second
confirmation. Ambiguous results retain evidence for reconciliation.

Reboot evidence survives Management Agent restart and disable/enable with the same
credential and guild; generation changes update the evidence. Saving the same
guild retains it. Token replacement, disconnect, binding a different guild, and
Factory Reset clear it atomically, and a clear failure prevents the trust change.
The existing connected Discord runtime opportunistically reconciles at most once
per minute, with one reconciliation at a time. A changed boot, the expected
running digest when recorded, activation state, and the existing authenticated
appliance health validation determine success. **HTTP acceptance alone never
confirms reboot or OS update success.** An unchanged boot or unavailable health /
deployment evidence is bounded by a 20-minute verification window. An explicitly
failed update-reboot workflow can finish verification earlier. Terminal results
remove the record before one best-effort sanitized channel message with mentions
disabled. Changed trust prevents delivery; failed delivery never restores evidence
or creates a retry queue.

Discord supports only **/jv** and **/help**, **Guilds intent only**, and no ordinary
message ingestion or channel-history reading. It does not expose poweroff,
firmware reboot, Factory Reset, shell, terminal, arbitrary commands/REST/RCON,
files or logs, storage administration, server creation/deletion, restore/migration,
plugin/mod installation, version conversion, container rollback selection, image
retention mutation, or host network / remote access configuration.


## Telegram interoperability

Discord and Telegram can be enabled simultaneously, independently of the AI/API
Gateway switch. They retain independent identities, credentials, and permissions:
Telegram approval grants no Discord approval, and Discord approval grants no
Telegram approval. Community Minecraft access requests do not grant management
permissions in either integration.

Both surfaces use the same authoritative JustVoxel Minecraft and system
operations. Reviewed plans, configuration fingerprints, and staged deployment
digests are revalidated against current backend state, regardless of which
integration changed it. Existing backend operation locks remain the concurrency
authority; enabling a second integration does not permit conflicting work or
bypass player-interruption confirmations.

Notification delivery remains integration-specific. A failed Discord send does
not change Telegram configuration or delivery, and vice versa. Durable credentials,
approvals, community requests, and each integration's separate reboot evidence
survive Management Agent restart according to their existing retention rules.
Telegram also retains its polling cursor and notification observations. Discord
confirmation actions, interaction tokens, runtime pointers, and guild discovery
are not persisted: after restart it starts connecting with the bound guild
unavailable until fresh READY and guild rediscovery. Reconnect retains the saved
binding and requires fresh availability; it never silently selects another guild.

Automated source and integration tests use local fake Discord REST/Gateway and
Telegram transports and authoritative backend test fixtures. They do not establish
real Discord or Telegram network acceptance, appliance runtime acceptance, or
successful real reboots. Those require separate human verification.
