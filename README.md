# 🎮 JustVoxel

**Your Minecraft servers. Your own hardware, rented dedicated server, or hosted VM. One Control Center.**

Run your own Minecraft servers without becoming a Linux administrator.

JustVoxel is a dedicated, self-hosted Minecraft server appliance. Install it on a compatible PC, laptop, physical server, virtual machine, or hosting service that lets you run your own OS image, then manage Minecraft from a browser.

No collection of scripts to maintain. No separate Minecraft management panel to install.

Have an old laptop or PC? **JustVoxel HWS can turn it into a dedicated Minecraft appliance instead of leaving useful hardware in a drawer.** On supported laptops, JustVoxel includes controls to ignore lid-close sleep so the server keeps running with the lid closed, plus local console display controls that can turn the screen off immediately or after a short delay.

> 🚧 **Pre-release:** JustVoxel is being prepared for its first public release. The features described here reflect the current development implementation. Stable installation media and final release instructions will be published after release validation.

## ✨ Minecraft hosting, made simpler

JustVoxel combines Minecraft management with the operating system underneath it.

- 🎮 **Multiple Minecraft servers** — run different worlds and server types independently on one appliance.
- 🖥️ **Web Control Center** — manage servers from a computer, tablet, or phone.
- 🧩 **Plugins & Mods** — discover and manage compatible content from supported providers.
- 💾 **Backups & Restore** — protect each server with scheduled and manual backups.
- 🌐 **Remote Access** — choose from four integrated private-access and game-tunneling options.
- 🤖 **AI & Automation** — connect compatible AI assistants and applications through MCP and REST.
- 💬 **Telegram & Discord** — optional bot controls, notifications, and community workflows.
- 🛡️ **System Management** — manage networking, storage, updates, health, logs, and supported power controls without learning the Linux commands underneath.

JustVoxel is designed for families, friends, small Minecraft communities, and people who want to own their server instead of renting a traditional Minecraft hosting panel.

## 🖥️ One appliance, two editions

### JustVoxel VM

For virtual machines and hosted environments such as:

- Proxmox
- KVM / libvirt
- VMware
- Hyper-V
- VirtualBox
- hosted VMs or VPS products that allow a compatible custom OS image

The VM edition contains the complete Minecraft appliance without physical-hardware packages that do not make sense inside a VM.

### JustVoxel HWS

For installation directly on physical hardware, including:

- an old laptop
- a repurposed desktop
- a mini PC
- a home server
- a rented bare-metal server where custom installation media is allowed

HWS adds physical-machine support such as UPS/NUT integration, SMART and NVMe tools, sensors, firmware/UEFI controls, device firmware, hardware diagnostics, and laptop display/lid controls.

Both editions use the same JustVoxel Control Center, Management API, Minecraft workflows, storage model, backups, networking, and update experience.

[Learn about the VM/HWS capability model](https://github.com/home-server-project/justvoxel-base/blob/testing/docs/VARIANTS.md)

## 🎮 Create the Minecraft server you want

JustVoxel can run several independent Minecraft servers at the same time when the machine has enough memory and available ports.

Each server gets its own identity, configuration, runtime, ports, data, backups, lifecycle controls, and status.

### Current server software

| Server software | Content model | Managed Java → Bedrock cross-play |
| --- | --- | --- |
| **Paper** | Plugins | ✅ |
| **Purpur** | Plugins | ✅ |
| **Spigot** | Plugins | ✅ |
| **Pufferfish** | Plugins | ✅ |
| **Leaf** | Plugins | ✅ |
| **Fabric** | Mods | ✅ When exact compatibility is proven |
| **NeoForge** | Mods | ✅ When exact compatibility is proven |
| **Forge** | Mods | Java only |
| **Quilt** | Mods | Java only |
| **Vanilla Java** | No plugins or mods | Java only |
| **Native Bedrock** | Native Bedrock server | Native Bedrock clients |

Choose **Simple Setup** for a guided first server, or **Advanced Setup** when you want more control over memory, ports, storage, backups, version policy, and compatible cross-play.

JustVoxel checks capability against the selected server software instead of pretending every feature works everywhere.

[See the server software and compatibility details](https://github.com/home-server-project/justvoxel-base/blob/testing/docs/server-software.md)

## 🧩 Plugins & Mods

Open **Minecraft → Plugins & Mods** for the selected server.

Depending on the server type, JustVoxel can use:

- [Modrinth](https://modrinth.com/)
- [Hangar](https://hangar.papermc.io/)
- [CurseForge](https://www.curseforge.com/minecraft) with your configured API key

Supported installations use compatibility checks, verified downloads, cold backups, restart verification, ownership tracking, and recovery safeguards.

Paper-family servers can use plugin catalogs. Fabric, Forge, NeoForge, and Quilt use mod catalogs with exact loader and Minecraft-version checks. Vanilla Java and native Bedrock intentionally do not expose a plugin/mod catalog.

[Learn about Plugins & Mods](https://github.com/home-server-project/justvoxel-base/blob/testing/docs/plugin-workspace.md)

## 🌐 Optional Minecraft Network

Several standalone servers are perfectly normal. They do **not** require a proxy.

When you want one public entrance to several compatible servers, JustVoxel also includes an optional **Minecraft Network** built around Velocity.

From the Control Center you can select supported backend servers, configure public Java and Bedrock ports, manage the virtual lobby and welcome message, and control Network membership.

Paper, Purpur, Pufferfish and Leaf are supported Network backends. Compatible Fabric/Quilt and Forge/NeoForge servers can also participate when JustVoxel proves the required proxy compatibility. Spigot, Vanilla and native Bedrock remain standalone.

[Learn about Minecraft Network](https://github.com/home-server-project/justvoxel-base/blob/testing/docs/minecraft-network.md)

## 🖥️ Meet the Control Center

JustVoxel is designed around server tasks instead of Linux internals.

### Minecraft Servers

Create and manage multiple servers, start or stop them, inspect status and players, configure supported settings, and access server-specific tools.

### Plugins & Mods

Discover compatible content, review installed plugins/mods, manage supported additions, and see recovery information when a change needs attention.

### Backups & Storage

Create backups, review backup history, restore complete server data, configure backup destinations, import/export servers, and manage supported storage.

### Network

Manage Ethernet, Wi-Fi, **connection troubleshooting**, and Remote Access. Potentially disruptive network changes use rollback protection so a bad IP or Wi-Fi change does not have to permanently lock you out.

### System

Review health, resources, logs, operating-system updates, users, security, supported power actions, and HWS-only hardware controls.

### AI & API Connections

Connect supported AI applications, MCP clients, REST automation, Telegram, and Discord without creating another management stack.

Prefer a terminal? JustVoxel also includes **mjust**, an interactive terminal administration interface using the same appliance safety model.

[WebUI documentation](https://github.com/home-server-project/justvoxel-base/blob/testing/docs/WEBUI.md) · [mjust guide](https://github.com/home-server-project/justvoxel-base/blob/testing/docs/MJUST.md)

## 🌐 Let friends connect — your way

JustVoxel currently integrates four Remote Access options in **Network → Remote Access**.

These are not simply packages sitting unused in the image. JustVoxel provides supported status, activation/deactivation, and setup flows for the provider where applicable.

| Option | Best for | What it gives you |
| --- | --- | --- |
| **[Tailscale](https://tailscale.com/)** | Private servers for friends and family | Private mesh networking between approved devices |
| **[NetBird](https://netbird.io/)** | Private networks with managed access | Private networking with account and access-policy control |
| **[Playit.gg](https://playit.gg/)** | Public Minecraft access without normal router port forwarding | Public game tunnels and connection addresses |
| **[NordVPN Meshnet](https://meshnet.nordvpn.com/)** | Private device-to-device access for NordVPN users | Meshnet connectivity without using JustVoxel as a commercial VPN client |

### Which one should I choose?

**🎮 Friends should join without installing a private VPN client?**  
Start by looking at **Playit.gg**. Players can use the public address from your configured game tunnel.

**🔒 Want a private server for trusted devices?**  
Consider **Tailscale** or **NetBird**.

**🌍 Already use NordVPN Meshnet?**  
JustVoxel can manage its Meshnet integration. This is separate from connecting the appliance to a normal commercial NordVPN exit server.

Provider accounts, cloud-side policies, and tunnel configuration still belong to the provider. JustVoxel handles the supported appliance-side integration.

Using several VPN/mesh products at the same time is possible, but routing, DNS, or firewall rules can conflict.

[Read the networking and Remote Access guide](https://github.com/home-server-project/justvoxel-base/blob/testing/docs/NETWORKING.md)

## 🤖 AI, MCP, REST & Automation

Want to ask an AI assistant for server status, connect home automation, or build your own control workflow?

JustVoxel provides built-in **MCP and REST interfaces** through **AI & API Connections**.

The setup surface includes choices for compatible applications and connection types such as:

- **OpenAI — ChatGPT & Codex**
- **Claude**
- **Grok**
- **Gemini**
- **GitHub Copilot**
- **Other AI / MCP app**
- **Automation / REST API**

You decide what each connection can access.

Permissions range from **Level 1 — Read Only** through server-management capabilities and, only with explicit approval, advanced system control. Connections can also be restricted to selected Minecraft servers.

JustVoxel does not need an AI model API key merely to expose its MCP or REST interface. Actual client support depends on the application, account/subscription capabilities, and network access.

The AI/API gateway starts disabled until you enable it.

[Learn about AI & API Connections](https://github.com/home-server-project/justvoxel-base/blob/testing/docs/AI-API-CONNECTIONS.md)

## 💬 Telegram & Discord

JustVoxel includes native optional Telegram and Discord integrations built on the same authoritative management backend as the WebUI.

### Telegram

A dedicated Telegram bot can provide:

- appliance and Minecraft status
- server selection and player information
- start, stop, and restart for approved management users
- manual backups
- Minecraft and container update review
- OS update information
- controlled reboot workflows
- optional notifications for server state, backup results, and update availability

Telegram uses its own **Read Only** and **Server Management** permissions and does not require an AI model.

[Telegram setup and permissions](https://github.com/home-server-project/justvoxel-base/blob/testing/docs/TELEGRAM.md)

### Discord

Discord can act as both a community-facing Minecraft surface and an administration surface.

Community members in the explicitly bound Discord server can see basic server status and request Minecraft access without automatically receiving administration rights.

Approved JustVoxel Discord administrators can manage supported server actions such as start, stop, restart, manual backup, update review, Minecraft Network controls, and additional advanced actions.

Discord membership, Minecraft whitelist access, and JustVoxel administration remain separate permissions.

[Discord setup, community access, and administration](https://github.com/home-server-project/justvoxel-base/blob/testing/docs/DISCORD.md)

## 💾 Backups, Restore, Storage & Migration

Minecraft data matters more than the appliance image.

JustVoxel keeps Minecraft data and backup storage outside the immutable operating-system deployment and treats them as separate choices.

Supported storage workflows include:

- the system filesystem
- dedicated internal disks
- USB storage
- existing local partitions/filesystems
- new partitions created from already-unallocated space
- NFS
- SMB/CIFS

Each server can use automatic and manual backups. Restore replaces the complete persistent data of the selected server while preserving its JustVoxel identity and appliance configuration.

For moving servers between machines or reinstallations, JustVoxel also provides reviewed **Export / Import / Recovery** workflows.

[Storage](https://github.com/home-server-project/justvoxel-base/blob/testing/docs/STORAGE.md) · [Restore](https://github.com/home-server-project/justvoxel-base/blob/testing/docs/RESTORE.md) · [Migration](https://github.com/home-server-project/justvoxel-base/blob/testing/docs/MIGRATION.md)

## 🛡️ Built for safer changes

JustVoxel is not just a pretty page that runs arbitrary shell commands.

Supported operations use appliance-level safety rules such as:

- player checks before disruptive actions
- reviewed changes before Apply
- cold backups where a change needs recovery protection
- exact server identity so one server cannot accidentally become another
- restart/runtime verification
- persistent operation and recovery state
- rollback paths for supported operations
- NetworkManager checkpoints for risky network changes
- separate Minecraft and operating-system update workflows

The browser-facing WebUI remains unprivileged. Privileged changes go through the local JustVoxel Management Agent and its explicit API instead of exposing a root shell to the browser.

## 🔄 System updates without mixing them with Minecraft updates

JustVoxel is an image-managed appliance built on Home Server Base 10 / AlmaLinux 10.

Operating-system updates are handled separately from Minecraft versions, server software, plugins, mods, and container updates.

That means updating the appliance does not silently mean changing your Minecraft version, and changing a Minecraft server does not mean rebuilding the host operating system.

[System management documentation](https://github.com/home-server-project/justvoxel-base/blob/testing/docs/SYSTEM.md)

## 📊 How does JustVoxel compare?

JustVoxel is a different type of project from a traditional Minecraft control panel: the **operating system and the Minecraft management layer are one appliance**.

| Capability | **JustVoxel** | **MineOS SvelteKit** | **Crafty Controller** | **MineOS Node** |
| --- | :---: | :---: | :---: | :---: |
| Complete Minecraft appliance OS | ⭐ | — | — | — |
| Multiple Minecraft servers | ✅ | ✅ | ✅ | ✅ |
| Java + Bedrock server support | ✅ | ✅ | ✅ | 🟡 Java + legacy PocketMine support |
| Guided plugin/mod discovery | ✅ Modrinth, Hangar, CurseForge | ✅ Modrinth, CurseForge | — | — |
| Host network, storage and OS management | ⭐ | — | — | — |
| Integrated private access / game tunneling | ⭐ 4 options | — | — | — |
| MCP + REST AI/automation gateway | ⭐ | — | — | — |
| Native Telegram + Discord management | ⭐ | — | — | — |
| Laptop lid/display appliance controls | ✅ HWS | — | — | — |
| Image-managed immutable OS updates | ✅ | — | — | — |
| Terminal management interface | ✅ mjust | ✅ mineos CLI/TUI | — | ✅ CLI |
| Installation model | Complete bootc appliance | Docker / Compose application | Control panel on Docker, Linux or Windows | Node.js application on Linux/BSD |

⭐ standout &nbsp;•&nbsp; ✅ supported &nbsp;•&nbsp; 🟡 partial/legacy &nbsp;•&nbsp; — not part of the reviewed core project documentation

**Comparison last reviewed: November 16, 2026.** The goal is not to declare every other project “bad.” MineOS and Crafty are Minecraft management applications that can be excellent choices when you already have an operating system you want to administer yourself. JustVoxel is aimed at users who want the Minecraft panel **and the server appliance underneath it** managed as one product.

The comparison uses the current project documentation and repositories:

- [MineOS SvelteKit](https://github.com/freeman412/mineos-sveltekit)
- [Crafty Controller](https://craftycontrol.com/) · [source](https://gitlab.com/crafty-controller/crafty-4)
- [MineOS Node](https://github.com/hexparrot/mineos-node)

If a comparison entry is outdated or unfair, please open an issue. Corrections that make JustVoxel look worse are welcome too.

## 🚀 Install JustVoxel

JustVoxel is still in pre-release validation.

The release installation path is the **[JustVoxel ISO Builder](https://github.com/home-server-project/justvoxel-iso)**, which is being prepared as the installation-media factory for VM and physical-hardware deployments.

The intended choices are simple:

- **JustVoxel VM** for virtual machines and compatible hosted VM environments
- **JustVoxel HWS** for PCs, laptops, mini PCs, physical home servers, and compatible bare-metal hosting

Final stable installation instructions and release artifacts will be published when release validation is complete.

## 📚 Documentation

The detailed appliance documentation lives in **[JustVoxel Base](https://github.com/home-server-project/justvoxel-base)**.

Useful starting points:

- [WebUI](https://github.com/home-server-project/justvoxel-base/blob/testing/docs/WEBUI.md)
- [Minecraft runtime](https://github.com/home-server-project/justvoxel-base/blob/testing/docs/MINECRAFT_RUNTIME.md)
- [Server software](https://github.com/home-server-project/justvoxel-base/blob/testing/docs/server-software.md)
- [Plugins & Mods](https://github.com/home-server-project/justvoxel-base/blob/testing/docs/plugin-workspace.md)
- [Minecraft Network](https://github.com/home-server-project/justvoxel-base/blob/testing/docs/minecraft-network.md)
- [Networking & Remote Access](https://github.com/home-server-project/justvoxel-base/blob/testing/docs/NETWORKING.md)
- [AI & API Connections](https://github.com/home-server-project/justvoxel-base/blob/testing/docs/AI-API-CONNECTIONS.md)
- [Telegram](https://github.com/home-server-project/justvoxel-base/blob/testing/docs/TELEGRAM.md)
- [Discord](https://github.com/home-server-project/justvoxel-base/blob/testing/docs/DISCORD.md)
- [Storage](https://github.com/home-server-project/justvoxel-base/blob/testing/docs/STORAGE.md)
- [Restore](https://github.com/home-server-project/justvoxel-base/blob/testing/docs/RESTORE.md)
- [Migration](https://github.com/home-server-project/justvoxel-base/blob/testing/docs/MIGRATION.md)
- [System management](https://github.com/home-server-project/justvoxel-base/blob/testing/docs/SYSTEM.md)
- [mjust](https://github.com/home-server-project/justvoxel-base/blob/testing/docs/MJUST.md)

The **JustVoxel product repository** contains the release-facing product layer. **JustVoxel Base** remains the development repository for the common appliance implementation and its technical documentation.

## ⛏️ Minecraft software and EULA

JustVoxel does **not** ship Minecraft server binaries, Mojang server software, a pre-created world, or a pre-accepted Minecraft EULA.

During server creation, the administrator accepts the Minecraft EULA and JustVoxel prepares the selected supported Minecraft server runtime.

Minecraft, Mojang software, Paper, Purpur, Spigot, Pufferfish, Leaf, Fabric, NeoForge, Forge, Quilt, plugins, mods, and other third-party components retain their own licenses and distribution terms.

## 📄 License

JustVoxel is licensed under **Apache-2.0**. See [LICENSE](LICENSE).

Third-party notices are maintained in [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).
