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
The private identity database stores only the credential and bounded configuration
and approval records; interaction tokens and messages are never persisted.

Discord runs inside the existing Management Agent with **Guilds intent only**.
No Message Content, Guild Members, Presence, or message intents are required.
JustVoxel does not read ordinary messages or channel history. No Discord roles,
message content, or interaction tokens are returned to the WebUI. Failures shown
in Discord use safe fixed messages rather than internal errors or command output.
Browser state is refreshed using the existing Refresh action, without polling.
