# JustVoxel networking architecture

JustVoxel provides network management through two independent user interfaces:

- `nm-hsp` is the friendly local terminal network manager shipped as a Home Server Package.
- The JustVoxel WebUI uses the privileged JustVoxel Management Agent.

These interfaces may provide similar user-facing capabilities, but they are deliberately not dependencies of one another.

## Independence rule

The WebUI networking implementation must not:

- import `nm-hsp` packages
- execute the `nm-hsp` binary
- parse `nm-hsp` terminal output or `--snapshot` output
- require a particular `nm-hsp` version to be installed

Likewise, `nm-hsp` must not require the JustVoxel WebUI or Management Agent.

This allows `nm-hsp` to be updated, redesigned, or replaced independently without changing the WebUI networking backend. The two implementations share NetworkManager as the system authority, not an application-level protocol between each other.

## Runtime boundary

The supported WebUI path is:

Browser → unprivileged JustVoxel WebUI → Management API over the local Unix socket → privileged JustVoxel Management Agent → NetworkManager system D-Bus

The browser never talks to D-Bus directly and never receives raw D-Bus object paths as identifiers.

The Management Agent owns the JustVoxel networking backend under `management/internal/networking`. That package talks directly to `org.freedesktop.NetworkManager` on the system D-Bus using `github.com/godbus/dbus/v5`.

## Network Workspace

The draggable, resizable Network window provides Refresh and these administrator tabs.

- **Overview**: connectivity, NetworkManager state/version, networking enabled state, and compact Ethernet/Wi-Fi summaries with addresses. No configuration controls.
- **Ethernet**: active profile, carrier, speed, IPv4, gateway, DNS, MTU and autoconnect, with automatic/DHCP or manual IPv4 controls.
- **Wi-Fi** (when Wi-Fi hardware is present): radio, nearby networks, scan, saved-profile activation, new/hidden networks, disconnect and forget.
- **Troubleshoot**: NetworkManager connectivity, interface/carrier state, addresses, gateway, DNS and active profile. Administrators can request a fresh connectivity check or reconnect a selected active Ethernet/Wi-Fi profile behind a checkpoint.
- **Remote Access**: Tailscale, NetBird, Playit.gg and NordVPN Meshnet availability, enabled/running state, configuration evidence, lifecycle controls and external dashboard links.
- **Router**: router provider cards, the MikroTik RouterOS setup guide, and live forwarding status.

Operators and Viewers receive Overview only. Management Agent authorization remains authoritative for all mutations; WebUI POST requests retain CSRF protection.

### Router workspace

The Router tab has a setup guide and a live connection for MikroTik RouterOS. TP-Link Omada, UniFi, OpenWrt, OPNsense and Cisco are marked **Not supported yet**. Select the Router tab again to return from the guide or connection details to the provider cards.

#### MikroTik RouterOS preparation

The router administrator makes these changes on the router.

1. Create a dedicated RouterOS group, for example `jv`, with the `read`, `write`, `api` and `rest-api` policies. REST authorization requires the `api` policy. The native RouterOS API service on TCP 8728 can stay disabled unless you use it for another purpose. Create a dedicated user, for example `justvoxel`, and assign it to the group. Choose a strong, unique password and limit **Allowed Address** to `<JUSTVOXEL_IP>/32` where appropriate. See [RouterOS users and groups](https://help.mikrotik.com/docs/spaces/ROS/pages/8978504/User).
2. If JustVoxel already trusts the router’s HTTPS certificate, skip creating certificates. Otherwise, open **System → Certificates** and create a private CA. Use a name and **Common Name** such as `justvoxel-ca`, RSA 2048 and a long validity such as 3650 days. Enable **Trusted** and select only `key-cert-sign` and `crl-sign` for **Key Usage**. Sign it as a self-signed root CA.
3. Create the router’s HTTPS server certificate, for example `www-ssl`. Set the **Common Name** and **Subject Alt Name** (SAN) to match the IP address or DNS name you will enter in JustVoxel. For an IP SAN in WebFig, select **IP** as the SAN type and enter only the address in the value field, without `IP:` or `::`. Use RSA 2048, a validity such as 825 days, and the `digital-signature`, `key-encipherment` and `tls-server` key usages. Sign this certificate with the private CA you created. See [RouterOS certificates](https://help.mikrotik.com/docs/spaces/ROS/pages/2555969/Certificates).

   If WebFig refuses to sign the HTTPS server certificate, for example by showing **Selection expected!**, the router administrator can run this command in RouterOS Terminal.

   ```text
   /certificate sign <SERVER_CERTIFICATE_NAME> ca=<CA_CERTIFICATE_NAME>
   ```

   Replace both placeholders with your certificate names. `www-ssl` and `justvoxel-ca` are example certificate names.

4. Assign the signed server certificate to `www-ssl` and enable the HTTPS service. Set `rest-secure=yes` and `rest-plain=no`. WebFig, index and graphs can stay enabled if you use them, or you can disable them to reserve HTTPS for REST. Allow JustVoxel to reach the HTTPS REST endpoint, normally `<JUSTVOXEL_IP> → <ROUTER_IP>:443/TCP`. The router administrator owns service address restrictions and firewall input policy. JustVoxel leaves the router’s input and firewall filter rules alone. See [RouterOS Services](https://help.mikrotik.com/docs/spaces/ROS/pages/103841820/Services) and [RouterOS REST API](https://help.mikrotik.com/docs/spaces/ROS/pages/47579162/REST%2BAPI).
5. If you created a private CA, export only its public certificate as PEM and leave the export passphrase blank. Download the `.crt` file from RouterOS. Upload it in JustVoxel under **System Network → Router → Connection**, then enter the HTTPS endpoint and credentials and test the connection.
6. Ensure the firewall allows legitimate destination-NAT traffic to the JustVoxel host. Some configurations already allow legitimate `dstnat` traffic and need no additional rule.

The following filter settings are an example only, not a universal command. Adapt the WAN selection, destination, chains and rule order to your firewall. Put the allowance before any rule that would otherwise block the forwarded traffic. JustVoxel will never create, move, enable, disable or delete this filter rule.

```text
chain=forward
action=accept
connection-nat-state=dstnat
in-interface-list=<WAN_INTERFACE_LIST>
dst-address=<JUSTVOXEL_IP>
```

#### Cycle 2 connection and forwarding status

In **System Network → Router → Connect**, the primary administrator enters the HTTPS address, username and password, then selects **Test connection**. The test uses any selected CA. Under **Port forwarding**, JustVoxel selects the destination when one active managed appliance address shares the router IPv4 endpoint’s subnet. It selects Internet ingress from consistent Minecraft forwarding rules or a unique interface list named WAN. Internet ingress is where public traffic enters the router before forwarding to JustVoxel. Choose only settings JustVoxel could not detect, then select **Save**. JustVoxel verifies the connection and selections again and saves the selected CA with the connector.

The Management Agent reads connector status over HTTPS REST. It verifies the router’s certificate and endpoint identity using system trust plus the optional MikroTik CA. The CA is used only for this connector, and uploads take effect without changing system trust or restarting services. Credentials and any uploaded CA are stored in the root-owned database at `/var/lib/justvoxel/webui/webui.db`, with directory mode `0700` and database mode `0600`. The connection view shows the CA common name and expiration date. Only the primary administrator can test, save, change the trusted CA or forget the connection.

Renewing the HTTPS server certificate with the same CA requires no change in JustVoxel. If the signing CA changes, use **Replace CA** to select the new public CA, then select **Save**. Open **Edit connection** and use **Remove CA** when the router uses an HTTPS certificate that JustVoxel already trusts through its system roots. Both actions verify the live connection with the proposed trust before saving the change. **Forget connection** and Factory Reset erase the credentials and CA from the local connector state.

Administrators can view live router identity and version and the summarized NAT status for authoritative Minecraft ports. Open means an existing rule matches the selected ingress, protocol, port and JustVoxel destination. It describes the NAT rule and does not establish Internet reachability. Closed means no applicable enabled mapping, Conflict means a competing mapping, and Unknown means the router or rule state could not be verified. Use **Port forwarding** in an individual server’s menu to see its ports and status. Minecraft Network members use the shared Velocity public endpoints. JustVoxel exposes no arbitrary port controls and does not send the raw router NAT table to the WebUI.

#### Minecraft port forwarding

In a server’s **Port forwarding** dialog, the primary administrator can enable missing mappings, repair changed mappings and remove obsolete or duplicate managed rules. **Turn off port forwarding** removes only managed rules for that scope. A matching existing router rule stays open and is never adopted. A conflicting rule or an unknown state blocks the entire action before any write.

The exact ownership comment is `JustVoxel | <scope> | <role>`, including spaces and capitalization. The scope is the immutable instance ID or `minecraft-network` for the shared Velocity entrance. The role is `java`, `bedrock-tcp` or `bedrock-udp`. Changing a display name preserves ownership.

Each action reloads connector settings, validates the appliance destination and router ingress, derives the current Minecraft endpoints and reads the whole NAT table. JustVoxel serializes its own actions. Administrators and other tools must leave the NAT table alone until the action finishes. Immediately before each PATCH or DELETE, JustVoxel reads the exact record ID and checks its ID, ownership comment, scope, role, static status and `dstnat` chain. RouterOS REST has no atomic conditional write, so an external edit between this check and the write could still affect the wrong record.

A failed check or write stops the operation. Earlier changes may have taken effect. JustVoxel attempts a fresh table read and requires verified readback before reporting success. Changes to server ports, membership or configuration can show **Needs repair**. Only an explicit administrator action changes the router. Forgetting the connector or running Factory Reset clears local connector state and leaves external NAT rules in place.

Firewall filter rules, router input rules, interfaces, VLANs, routes, DNS, DHCP, RouterOS users and groups, certificates, `/ip service` settings, packages and updates, router reboots and arbitrary RouterOS commands remain outside the connector’s scope.

### Checkpoint safety layer

The Management Agent now implements the NetworkManager checkpoint transaction layer used by disruptive network mutations.

The Management API exposes administrator-only checkpoint lifecycle endpoints for:

- creating a checkpoint for explicit interface names
- reading active checkpoint metadata
- confirming the new state by destroying the checkpoint
- rolling back immediately and reporting per-interface rollback results

The unprivileged WebUI uses matching proxy endpoints and the existing Keep settings / Revert now banner.

Checkpoint behavior is deliberately conservative:

- the default automatic rollback timeout is 90 seconds
- accepted timeouts are 30 through 300 seconds
- a zero timeout from the WebUI/API means use the 90-second JustVoxel default, never an infinite checkpoint
- at least one explicit interface is required; JustVoxel does not create an all-device checkpoint
- JustVoxel uses NetworkManager checkpoint flags value 0, so it does not destroy external checkpoints or opt into overlapping checkpoints
- overlapping JustVoxel transactions on the same interface are rejected
- confirm and rollback terminal actions are serialized so concurrent requests cannot race
- NetworkManager D-Bus checkpoint object paths remain inside the Management Agent; browser/API clients receive an opaque random transaction ID
- transaction creation, confirmation, and rollback are written to the existing audit log

The in-memory JustVoxel transaction map is intentionally not authoritative for rollback safety. NetworkManager owns the actual checkpoint and its automatic rollback timer. If the Management Agent restarts while a checkpoint is pending, the opaque JustVoxel transaction ID is lost and cannot be confirmed afterward, but NetworkManager still retains the checkpoint and automatically rolls it back when its timeout expires. This fails toward restoring connectivity rather than keeping an unconfirmed network change.

### Wi-Fi management

The Wi-Fi tab provides administrator-only Wi-Fi mutations through the same independent JustVoxel backend:

- turn the NetworkManager Wi-Fi radio on
- turn the Wi-Fi radio off behind a checkpoint that covers all Wi-Fi interfaces
- connect an existing saved Wi-Fi profile by NetworkManager UUID
- create and activate a new Open, OWE, WPA/WPA2 Personal, or WPA3 Personal profile
- join a hidden Open, OWE, WPA/WPA2 Personal, or WPA3 Personal network
- disconnect a Wi-Fi interface
- forget an inactive saved Wi-Fi profile

New WEP and Enterprise credential entry are intentionally unsupported. A previously saved Enterprise profile can still be activated because no new credential is collected by JustVoxel.

Potentially disruptive Wi-Fi actions use this sequence:

1. The browser asks the Management Agent to create the checkpoint first.
2. The browser stores the opaque JustVoxel transaction ID in session storage.
3. Only after the checkpoint is armed does the browser request the Wi-Fi mutation.
4. The workspace shows a "Keep settings" / "Revert now" banner with the rollback countdown.
5. The pending transaction is recovered after a page reload while the browser session remains available.
6. "Keep settings" destroys the checkpoint.
7. "Revert now" rolls back immediately.
8. If the browser loses connectivity and cannot confirm, NetworkManager rolls back automatically at the timeout.

For a newly created Wi-Fi profile, JustVoxel records the new profile UUID in the pending transaction. An explicit JustVoxel rollback first asks NetworkManager to restore connectivity and then deletes that newly created profile. The conservative NetworkManager checkpoint flags remain value 0. Therefore an unattended NetworkManager timeout can restore connectivity while leaving the newly created saved profile behind; JustVoxel deliberately accepts that non-disruptive residue rather than enabling a broad checkpoint flag that could delete unrelated profiles created by another tool during the same window.

An active profile cannot be forgotten directly. The administrator must disconnect it, confirm the disconnected state, and then forget it. This prevents a "forget" action from deleting the profile that NetworkManager would need for rollback.

## Stable identifiers

D-Bus object paths are runtime implementation details and can change.

The WebUI/API boundary should use stable appliance identifiers instead:

- network devices: interface name, for example `enp1s0` or `wlp2s0`
- saved connection profiles: NetworkManager UUID

The backend resolves those identifiers to current NetworkManager objects immediately before an operation.

## Secret handling

Read-only status uses NetworkManager's normal settings and runtime interfaces and does not call `GetSecrets`.

A submitted Wi-Fi password is treated as short-lived request data:

- never place it in command-line arguments
- never log it
- never return it in API responses
- never store an application-side copy
- pass it directly to NetworkManager over D-Bus
- clear the mutable JustVoxel secret buffer after the D-Bus call and remove the PSK from the temporary settings map

NetworkManager requires a Go string at the D-Bus boundary, so JustVoxel does not claim that every runtime string copy can be cryptographically erased.

The existing WebUI transport is plain HTTP on the trusted LAN by default. That is a separate transport-security decision: keeping secrets out of the Management Agent does not encrypt the browser-to-WebUI hop.

## Authorization

The existing Management Agent remains the security boundary.

Read-only networking endpoints should use the same read-access policy as other status surfaces. Configuration-changing actions should initially require the administrator role unless a narrower permission is deliberately approved later.

No browser Polkit agent, sudo bridge, shell wrapper, or additional privileged daemon is required.

## Safe configuration changes

Potentially disruptive network changes must use the implemented NetworkManager checkpoint layer before they are exposed through the WebUI.

The intended mutation transaction is:

1. Create a NetworkManager checkpoint for the affected device or devices.
2. Apply the requested configuration.
3. Let the browser reconnect using the new configuration.
4. Require an explicit confirmation that connectivity is healthy.
5. Destroy the checkpoint after confirmation.
6. If confirmation does not arrive before the timeout, allow NetworkManager to roll back automatically.

This is the safety mechanism for changes such as static IP, gateway, DNS, or other settings that could otherwise lock a remote user out of the appliance.

## Ethernet confirmation and persistence

Ethernet editing requires an active, normal Ethernet profile. Devices use interface names; profiles use NetworkManager UUIDs. The agent validates IPv4 addresses, prefixes, gateways, DNS servers, MTU and identifiers. Advanced profiles remain a job for `nmtui`.

The browser creates the existing 90-second checkpoint for the explicit Ethernet interface before applying a candidate. The agent reads the device's applied connection and stages the candidate with NetworkManager `Reapply`. It does **not** update the saved profile before confirmation. Autoconnect is a future activation policy and is saved only upon confirmation.

If NetworkManager cannot reapply the requested settings safely, the operation reports an error and requests rollback; it never falls back to saving first. Revert or automatic timeout restores live connectivity while the previous persistent profile remains unchanged. An agent restart discards the candidate; NetworkManager still owns the automatic rollback.

After reconnecting (and signing in at the new address when necessary), administrator status recovers pending opaque checkpoints without transferring credentials between browser origins. **Keep settings** must successfully destroy the checkpoint before the agent updates the persistent profile. The agent rejects concurrent changes to either the saved or applied connection. If persistence fails after acceptance, the UI reports that live settings were accepted but saving failed. Review the saved configuration before rebooting; a failed or timed-out save is not reported as successful persistence.

See NetworkManager's [Device applied-connection and Reapply API](https://networkmanager.dev/docs/api/latest/gdbus-org.freedesktop.NetworkManager.Device.html) for the live connection semantics.

## Remote Access providers

Tailscale, NetBird, Playit and NordVPN Meshnet are separate providers outside `management/internal/networking`; NetworkManager remains the Ethernet/Wi-Fi authority. A fixed Management Agent provider layer controls only `tailscaled.service`, `netbird.service`, `playit.service`, `nordvpnd.service` and `nordvpnd.socket` with bounded systemctl operations. Arbitrary provider or unit names are rejected.

**Activate** enables and starts the fixed service. **Deactivate** stops and disables it, so the administrator's choice survives reboot. Successful and failed lifecycle changes are audited. Status is read-access; mutations require administrator authorization.

Tailscale configuration and connection evidence comes from its live JSON status. NetBird and Playit may also use non-empty regular configuration files as evidence, without reading their contents. A configuration file or running service alone does not prove a VPN connection or an available tunnel. Connected VPN IPv4 addresses are displayed without CIDR suffixes; the underlying networking information is preserved. The shared welcome and CLI status helper uses live JSON status, so normal user shells do not need sudo or access to VPN credentials. If status or filesystem evidence is inaccessible, it reports an unavailable state rather than treating an old address as a connection.

Playit tunnel information uses one bounded, read-only local `get_state` IPC request after checking the protocol handshake. Home Server Packages `:stable` currently packages Playit 2.0.2 ([upstream commit](https://github.com/playit-cloud/playit-agent/tree/89638d480be8d26cdbbdfcb14bc80b356d4e8492)), which uses IPC 3. JustVoxel accepts only IPC 2 and 3 for appliance upgrade compatibility and sends the server's accepted version in its request; the package channel remains the version source of truth. Only public addresses, local destinations and disabled flags are returned through the Management API. Disabled tunnels are shown in the existing Network Workspace card but excluded from available counts. Quick Look shows a count; the welcome shows the public address for one available tunnel and a count for multiple tunnels. A confirmed running agent with an empty available list shows **No tunnels**, regardless of active client connections. Missing, inaccessible or incompatible IPC reports **Tunnels: Unavailable** for an active configured service. A running service alone never proves an available tunnel. All-disabled lists report **No tunnels** and are not connected; disabled entries remain visible. The socket permissions and setup workflow remain unchanged.

The Playit card's administrator **Account** action is available while installed and idle, including after **Deactivate**. For an existing configuration it opens **Playit.gg account** with **Cancel** and **Relink account**. Cancel makes no request. Relink runs only the official bounded `/usr/bin/playit reset`, which removes the local saved agent connection for re-claim, verifies live waiting-for-secret state, and starts the same ten-minute setup/claim workflow used for initial setup. It exposes only a validated `https://playit.gg/claim/<10 hex chars>` link. Existing tunnels are not migrated between accounts automatically. JustVoxel does not claim that cloud tunnels are deleted. Without existing configuration, Account starts normal setup without reset or destructive confirmation. Normal Activate uses the current identity; normal Deactivate preserves the account link. Reset success alone is not setup success: the existing setup command must finish successfully with configuration evidence.

Provider cloud configuration remains external:

- [Tailscale admin console](https://console.tailscale.com/admin/)
- [NetBird dashboard](https://app.netbird.io/)
- [Playit.gg account](https://playit.gg/account/)
- [Nord Account](https://my.nordaccount.com/)

JustVoxel does not collect provider API keys or implement cloud account, policy or tunnel management. Playit is enabled during image composition, without starting it or creating a secret. On the first deployed boot, a running unconfigured Playit service is a normal waiting-for-secret state, not a failure. Initial account/agent setup is available in `mjust net` option three, using the packaged `/usr/bin/playit setup`. The packaged CLI owns the setup interaction and persistent configuration; further tunnel/account management belongs at Playit's account page.

### NordVPN Meshnet

NordVPN uses the official native RPM, `/usr/bin/nordvpn`, `/usr/sbin/nordvpnd`, and the packaged daemon/socket units. It is not a container. JustVoxel controls Meshnet only; it never selects or connects to a commercial NordVPN server. Activate enables/starts the fixed units, checks the account and turns Meshnet on, verifying the setting. Deactivate turns Meshnet off, verifies it, then stops/disables both units. It does not log out or remove account state.

**Activate** first offers **Use current login** or **Update Token**, with **Cancel** making no request. Use current login starts/enables NordVPN and checks the saved account before enabling Meshnet. A missing or Unknown account opens **NordVPN access token** for credential attention. The administrator **Token** button beside Activate is available whenever NordVPN is installed and no operation is in progress, including while inactive, after Deactivate or a failed Activate, and when Account is Unknown. It opens explicit credential replacement directly, so revoked, expired or stale saved logins can be repaired without CLI work. Select **Open NordVPN dashboard** to open [NordVPN dashboard](https://my.nordaccount.com/dashboard/nordvpn/) and sign in. Select **NordVPN** from the left menu, scroll down to **Advanced settings**, then select **Access token -> Get Access token -> Generate new token**. NordVPN may ask you to verify your email. Copy the token and paste it into JustVoxel's **Access token** field, then select **Reconnect with new token**. The same dialog offers **Create NordVPN account** through the fixed [JustVoxel creator’s referral link](https://refer-nordvpn.com/pZGMpGTrBrO). Links open only when selected; JustVoxel does not automatically open a tab.

Explicit token replacement checks the account and attempts the official local `nordvpn logout --persist-token` for an existing or Unknown login to reset saved login state without intentionally expiring a still-valid old access token, then performs the official headless `nordvpn login --token <token>` through direct execution. A failed logout does not block the new login attempt. The new login command must succeed, the account must verify as Logged in, and Meshnet must verify as Enabled before success. Normal Activate never logs out. NordVPN access tokens may expire or be revoked externally; JustVoxel does not predict their lifetime or know their expiration date. JustVoxel never persists or logs the token, includes it in audit records, or returns raw CLI output. The token is transiently passed as one process argument required by NordVPN. Optional analytics consent is declined during login. Deactivate disables/verifies Meshnet and stops/disables NordVPN without logging out or removing Nord account state. **Open NordVPN account** on the provider card remains available independently.

Status reads bounded CLI output without opening credential files. Only recognized account and Meshnet states are displayed; unfamiliar or unavailable output is Unknown. Local Nord name/IP rows appear only when the CLI identifies this device. Service activity alone does not mean Meshnet is enabled.

Using multiple VPN options at the same time is possible but not recommended. Routing, DNS or firewall rules can conflict. If something breaks, report a reproducible issue on GitHub; it may be possible to fix. JustVoxel neither blocks simultaneous activation nor automatically disables another provider. Playit remains independent.

Remote Access cards use their workspace container width: one column by default, two from 580px, three from 860px, and four from 1140px. Resizing reflows the cards without changing workspace geometry or card styling.

### Local network diagnostics

Administrators can open **System -> Logs -> Network** and select **Generate network report**. The fixed, bounded report includes release/source metadata, device/address/routes/DNS summaries, failed units, NetworkManager and remote-access service states/current-boot journals, and sanitized provider summaries. Missing commands/providers produce unavailable sections. It does not run NordVPN’s broad diagnostics bundle or read configuration/credential files.

Reports reuse private diagnostic storage in `network-logs` (root-owned 0700 directory and 0600 files), the existing sanitizer and 256 KiB report/64 KiB section limits. Sensitive network lines, authentication URLs, callbacks and key material are redacted. Reports use the existing allowlisted list/read/download flow. They stay local and are never uploaded automatically; review a report before attaching it to an issue.

## Development rule

Behavior may be compared with `nm-hsp` as a user-experience reference, but source-level or release-level coupling is not allowed. Each implementation must remain independently buildable, testable, releasable, and replaceable.
