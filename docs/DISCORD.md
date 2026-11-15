# Discord connection

The primary WebUI administrator can configure Discord in the existing AI & API
Connections workspace. Save a dedicated bot token, verify the bot, then use
**Add bot to Discord server** to install it. The install link requests only View
Channels and Send Messages permissions.

Enable Discord, then use **Refresh** to view its connection and discovered
Discord servers. Choose a server and press **Use this server** to save the binding.
JustVoxel supports one bound Discord server and never selects or switches it
automatically. If the bot loses access, the saved binding remains visible; restore
the bot's access or explicitly choose another available server.

Disabling Discord closes its connection while retaining the credential, verified
identity, and saved server. Disconnect completely removes Discord configuration.
Replacing a token disables the connection and clears the old identity and binding.
If the connection pauses after repeated initial failures, check the credential and
network connection, then disable and enable Discord to retry.

The Gateway runs inside the existing Management Agent with only the Guilds intent.
Connection state and discovered servers exist only in memory. No message reading,
Discord commands, interactive control panel, or Minecraft operations are provided
in this stage. Browser updates use the existing Refresh action, without polling.
