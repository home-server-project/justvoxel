# Native Telegram

Telegram is optional and configured inside **AI & API Connections**. Its backend,
authorization and enable switch are independent of the AI/API gateway. It uses the
[Telegram Bot API](https://core.telegram.org/bots/api) over outbound HTTPS long
polling in the existing Management Agent. It needs no AI model, inbound port,
webhook, additional container, or separate daemon. The WebUI continues to use the
existing local Unix-socket Management API.

**Connections** shows the configured Telegram integration and its current status.
Telegram remains independent of **AI & API Gateway**; either switch can be changed
without changing the other.

A short setup guide is available in **AI & API Connections → Help → Provider reference → Telegram**.

Discord and Telegram are independent interaction surfaces over the same
authoritative JustVoxel management operations and may be enabled simultaneously.
Telegram approval does not grant Discord approval, and Discord approval does not
grant Telegram approval. Shared backend operation locks and reviewed
fingerprints/digests remain authoritative regardless of which surface changed
backend state.

## Connect and pair

Create a dedicated bot using Telegram's BotFather. Do not run another polling
app or webhook with the same bot. Sign in as the primary `voxel` administrator
and open **Control Center → AI & API Connections → Add → Telegram**.

1. Enter **Bot Token**.
2. Choose **Save**. JustVoxel automatically verifies the bot identity and checks
   that it has no configured webhook. If verification fails, the token stays
   saved; choose **Retry** or **Replace token**.
3. After configuration, use the **Telegram** enable/disable switch in the
   **AI & API Connections** top navigation. **Bot verified** and **Connected** are
   distinct states.
4. Choose **Pair user**.
5. Choose **Copy** to copy the exact `/start <challenge>` command, then send it in
   a **private chat** with the bot within ten minutes. On LAN HTTP, Copy uses a
   clipboard fallback; if copying is unavailable, the command is selected for Ctrl+C.
6. Telegram acknowledges the pairing request, which still requires approval.
7. WebUI automatically shows the numeric Telegram user ID. Check it with that person.
8. Choose **Read Only** or **Server Management** from the permission dropdown and
   choose whether to enable Notifications. These choices survive automatic refresh.
9. Choose **Approve**.

The command is shown separately from prose and kept only in browser memory while
waiting for a claim. It disappears on claim, expiry, credential replacement,
disconnect, or leaving Telegram. Settings refresh cannot retrieve it again.
Configured bots show **Replace token** and **Disconnect**; a verified bot needs no
manual verification step. Replacement reveals a temporary token field with
**Save** and **Cancel**. Saving a replacement automatically verifies it without
enabling Telegram.

Pressing Start alone never grants access. A pairing challenge can be claimed
only once, and claiming it does not approve the user. Usernames and display names
are never authority. Telegram private-chat user IDs are the identities used for
all requests and notifications. Groups and mismatched/forwarded button users are
rejected. Unapproved users are silently ignored. Only an exact active, unexpired,
unclaimed pairing challenge receives the acknowledgement that the request was
received and still requires primary-administrator approval in JustVoxel.

Tokens are stored in the existing root-owned private WebUI SQLite database,
whose directory is mode 0700 and database is mode 0600. The token input is cleared
after submission and when the workspace closes. The token is never returned in
settings, retained in browser storage, or written to audit/notification output.
Telegram's HTTPS protocol necessarily includes the token in requests to its fixed
API host; redirects are refused and URL-bearing transport errors are discarded.

Replacing the credential disables Telegram and removes old approvals and
pairings; after automatic verification, enable and pair again. **Disconnect** removes the
saved credential, approvals, challenges, buttons, observations, and pending
Telegram notification state. It does not cancel an already accepted appliance
operation. Full Factory Reset revokes Telegram access before destructive cleanup;
a later reset failure does not restore it.

## Permissions

There are exactly two Telegram permissions. They do not use AI/API Levels 1–4.
Approving a Telegram account grants access to the appliance's existing servers;
Telegram does not have a separate per-instance scope setting. Immutable server
IDs, rather than friendly names, bind every instance action.

| Permission | Allowed operations |
| --- | --- |
| Read Only | Appliance and instance status; sanitized server details; player counts; backup inventory/status; Minecraft Network/Velocity status and eligibility; Minecraft/container and OS update information; registry update checks; own opt-in notification preference |
| Server Management | All Read Only operations; existing instance Start/Stop/Restart and manual backup; Start on Boot; reviewed supported Minecraft/container updates; Network enable/disable and member selection; download OS updates; confirmed safe update reboot or ordinary reboot |

Server Management does **not** grant server creation/deletion, restore, migration,
reset, storage administration, LAN/Wi-Fi/firewall/remote-access changes, shell,
arbitrary systemd/container control, user/security administration, file/plugin
operations, unrestricted settings, or power off. The Management Agent's explicit
Telegram allowlist enforces these boundaries; buttons are not authorization.
There is no external Telegram execute endpoint or generic privileged proxy.

Use `/menu`, `/status`, `/notifications`, or `/help`. The main menu contains
**Appliance**, **Minecraft servers**, **Minecraft Network**, **OS updates**, and
**Notifications**. Minecraft servers lists display names; selecting one loads its
current state and shows Status, Players, Backups and Updates. Running servers
have Stop and Restart; stopped servers have Start. A single **Start on Boot**
button shows the current preference. Management actions are visible only to
Server Management users.

Backups contains Status, History and Back up now. Updates contains Minecraft
update and Container update, with review/apply and progress actions. Each submenu
has a route back to its parent and the main menu. Status responses are concise
human summaries, without raw JSON, private paths, logs, fingerprints, image
references or digests. Missing or unreadable status directs the user to WebUI.
Notifications shows **Enabled** or **Disabled**.

`/version <instance ID> [recommended|latest|pinned] [version]` reviews a supported
Minecraft update selection; the existing exact version/loader/software
compatibility policy still applies. Container updates use the reviewed existing
workflow with a backup and restart. Existing update tracking and recovery remain
authoritative.

Every management action has a deliberate confirmation. Stop/restart first use
explicit player-check mode; when players are online, a separate button confirms
interruption. Existing authoritative checks run again and can still refuse an
operation. A queued operation is reported as accepted, not completed.

Start on Boot changes only the saved boot preference, under the existing runtime
locks and publication checks. Network-managed members retain their protected boot
lifecycle. Network actions preserve ports, welcome text, whitelist settings and
unrelated configuration. Membership menus bind the reviewed configuration and
refuse stale selections. To change membership, **disable Network, stop affected
servers, then select/remove members**. The existing Network transaction owns
compatibility preparation, memory checks, publication, locking and recovery.
Enabling Network starts its selected members according to the established
Network lifecycle. See [Minecraft Network](minecraft-network.md).

Permissions are rechecked for every message and callback. Buttons bind the
numeric user, operation, immutable instance ID, exact reviewed request and a
five-minute expiration. They are consumed durably before dispatch and cannot be
replayed. Revocation, permission changes, disabling and credential replacement
invalidate saved buttons. Already accepted appliance work may finish; subsequent
requests use the current permission. Accepted, attempted and denied management
actions are audited with numeric Telegram identity and fixed operation names,
without request bodies, challenge values or credentials.

## Updates and reboot

OS update information distinguishes the running image, registry availability and
the downloaded/staged deployment. Downloading an update does not reboot.
**Reboot options** offers ordinary reboot and, when a verified update is staged,
update reboot with a **10-second or 60-second** warning and optional
**backup before reboot**. Player-interruption approval and an explicit final
Telegram confirmation are required. All work uses the existing bootc and reboot
handlers/helpers; Telegram does not add another activation engine.

The pending boot identity, expected image digest, request time and request state
are saved before requesting reboot. An Agent restart or accepted helper response
is not update success. After a new boot, Telegram checks the actual booted digest,
existing activation evidence, authoritative appliance/instance validation, and
Management/WebUI service health. A different image, failed activation or failed
health check is reported accurately. Failed update-reboot workers are reported
while the appliance is still online. A request that has not produced a new boot,
or whose health cannot be verified within twenty minutes, is reported as
unverified; it is never assumed successful. Use existing **System Update**
recovery and logs when an operation fails. Telegram's own pending verification
state does not replace or modify those operation journals.

## Notifications and connection recovery

Approved users can opt in in Telegram or the administrator can set their
preference in the workspace. Notifications include observed server failures/state
changes, backup completion/failure, relevant Minecraft/container/OS update
availability, downloaded OS updates, reboot start, return online and a verified
OS update result. Instance/backup observations are about once a minute while the
poller is available; update provider checks are spaced by thirty minutes.
Long-running appliance actions can delay observations. This is an observation
feed, not a complete event log. Messages contain bounded public status only,
never raw sensitive logs or configuration files.

Per-user observation markers survive Agent restarts. Markers are saved before
sending, and ambiguous deliveries are not retried: this avoids duplicate
notifications but means a network interruption can lose an individual message.
The workspace reports unconfirmed delivery; check authoritative state in
JustVoxel. Notifications disabled for a user are never sent to that user.

Polling uses a local process lock. A Telegram competing-consumer conflict or
invalid credential pauses the connection. Other network failures receive up to
eight bounded attempts, with backoff capped at sixty-four seconds, then pause.
Verify the token, Internet access, and that no other app/webhook uses the bot;
verify and enable again in the workspace. Disable cancels an outstanding poll.
Configuration and cursor state survive a normal Agent restart. Updates are
recorded as consumed before dispatch, so interruption can leave a command result
unconfirmed; review appliance status rather than blindly retrying it. Existing
operation locks, fingerprints and recovery evidence remain authoritative.

If Telegram-only settings become unreadable, **Disconnect** remains available to
the authenticated primary administrator. It removes only Telegram credentials
and approvals; it does not edit appliance operation journals. Factory Reset
revocation also removes malformed Telegram-only settings without trusting them,
so they cannot permanently block reset recovery.
