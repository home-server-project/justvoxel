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

## Community Minecraft access (Stage 3)

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
intent, ordinary message reading, or channel-history reading. Stage 4 advanced
administration is not implemented.
