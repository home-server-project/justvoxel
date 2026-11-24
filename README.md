<p align="center">
  <img src="docs/images/justvoxel-logo.png" width="260" alt="JustVoxel">
</p>

<p align="center">
  <a href="README.md">English</a> •
  <a href="readme_i18n/README_fr-CA.md">Français (Canada)</a> •
  <a href="readme_i18n/README_fr-FR.md">Français (France)</a> •
  <a href="readme_i18n/README_de.md">Deutsch</a> •
  <a href="readme_i18n/README_pl.md">Polski</a> •
  <a href="readme_i18n/README_pt-BR.md">Português (Brasil)</a> •
  <a href="readme_i18n/README_es-419.md">Español (Latinoamérica)</a> •
  <a href="readme_i18n/README_es-ES.md">Español (España)</a> •
  <a href="readme_i18n/README_it.md">Italiano</a> •
  <a href="readme_i18n/README_nl.md">Nederlands</a> •
  <a href="readme_i18n/README_sv.md">Svenska</a> •
  <a href="readme_i18n/README_nb.md">Norsk Bokmål</a> •
  <a href="readme_i18n/README_da.md">Dansk</a> •
  <a href="readme_i18n/README_tr.md">Türkçe</a> •
  <a href="readme_i18n/README_id.md">Bahasa Indonesia</a> •
  <a href="readme_i18n/README_uk.md">Українська</a> •
  <a href="readme_i18n/README_ru.md">Русский</a> •
  <a href="readme_i18n/README_zh-Hans.md">简体中文</a> •
  <a href="readme_i18n/README_zh-Hant.md">繁體中文</a> •
  <a href="readme_i18n/README_ja.md">日本語</a> •
  <a href="readme_i18n/README_ko.md">한국어</a>
</p>

<!-- justvoxel-translation-notice -->

> **Translation notice:** Localized JustVoxel documentation is translated from the canonical English text with AI assistance. If you find an inaccurate, unclear, or unnatural translation, please submit a [GitHub issue](https://github.com/home-server-project/justvoxel/issues) or [pull request](https://github.com/home-server-project/justvoxel/pulls) with the correction.

<p align="center">
  <strong>Your Minecraft servers. Your own hardware, rented dedicated server, or hosted VM. One Control Center.</strong>
</p>

<p align="center">
  A self-hosted Minecraft server appliance for people who want to run the server — not administer the operating system underneath it.
</p>

<p align="center">
  <a href="#-install-justvoxel">Install</a> •
  <a href="#-minecraft-hosting-made-simpler">Features</a> •
  <a href="docs/INSTALLER.md">Documentation</a>
</p>

<p align="center">
  <img src="docs/images/justvoxel-overview-1.webp" width="100%" alt="JustVoxel Control Center overview"><img src="docs/images/justvoxel-overview-2.webp" width="100%" alt=""><img src="docs/images/justvoxel-overview-3.webp" width="100%" alt="">
</p>

JustVoxel is a dedicated, self-hosted Minecraft server appliance. Install it on a compatible PC, laptop, physical server, virtual machine, or hosting service that lets you run your own OS image, then manage Minecraft from a browser.

No collection of scripts to maintain. No separate Minecraft management panel to install.

Have an old laptop or PC? **JustVoxel HWS can turn it into a dedicated Minecraft appliance instead of leaving useful hardware in a drawer.** On supported laptops, JustVoxel includes controls to ignore lid-close sleep so the server keeps running with the lid closed, plus local console display controls that can turn the screen off immediately or after a short delay.

> 🚧 **Pre-release:** JustVoxel is being prepared for its first public release. The current installation ISO is available for testing and early use while final release validation continues.

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

[Learn about the VM/HWS capability model](docs/VARIANTS.md)

## 🚀 Install JustVoxel

One installer ISO supports both JustVoxel editions.

[⬇️ Download JustVoxel x86_64 ISO](https://sourceforge.net/projects/justvoxel/files/JustVoxel-x86_64.iso/download)

**Create a bootable USB:** Write the ISO to a USB drive using [balenaEtcher](https://etcher.balena.io/), [Rufus](https://rufus.ie/en/), [Ventoy](https://www.ventoy.net/en/index.html), [Fedora Media Writer](https://fedoraproject.org/workstation/download/), or another raw-image writing tool. On Linux and other Unix-like systems, the traditional `dd` method also works. Writing an image directly to a USB drive erases the drive.

Choose the edition that matches where you are installing:

- **JustVoxel VM** for virtual machines and compatible hosted VM environments
- **JustVoxel HWS** for PCs, laptops, mini PCs, physical home servers, and compatible bare-metal hosting

See the [installation guide](docs/INSTALLER.md) for installation, disk preparation, and first login.

## 🏠 Minecraft-first, not locked to Minecraft

JustVoxel is built and supported first as a Minecraft server appliance, but the system underneath it is still a normal **AlmaLinux 10 bootc server with Podman, Quadlet, systemd, SSH, NetworkManager and standard Linux administration tools**.

Advanced users are free to use the same machine as a broader home server and run additional containers alongside Minecraft.

For example, you can manually deploy services such as:

- **Jellyfin** or **Plex** for media
- **Vaultwarden** for password management
- **Immich** for photo and video management
- **Home Assistant Container** for home automation
- **Nextcloud** for files and collaboration
- **Syncthing** or **Navidrome**
- **OpenClaw** as a self-hosted AI agent/gateway

These applications are **not managed by the JustVoxel Control Center**. Their Podman containers, Quadlets, storage, ports, updates, backups, reverse-proxy configuration, and application-specific requirements remain under the administrator's control.

JustVoxel deliberately keeps this boundary simple: it manages the Minecraft appliance, while advanced users can still use normal Podman and Quadlet workflows for anything else.

JustVoxel maintenance also treats unrelated containers as administrator-owned workloads rather than Minecraft resources. Operations such as Minecraft **Start Over** do not remove unrelated Podman containers.

So a JustVoxel machine can remain primarily a Minecraft appliance while also serving as a normal containerized home server when you want it to.

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

[See the server software and compatibility details](docs/server-software.md)

## 🧩 Plugins & Mods

Open **Minecraft → Plugins & Mods** for the selected server.

Depending on the server type, JustVoxel can use:

- [Modrinth](https://modrinth.com/)
- [Hangar](https://hangar.papermc.io/)
- [CurseForge](https://www.curseforge.com/minecraft) with your configured API key

Supported installations use compatibility checks, verified downloads, cold backups, restart verification, ownership tracking, and recovery safeguards.

Paper-family servers can use plugin catalogs. Fabric, Forge, NeoForge, and Quilt use mod catalogs with exact loader and Minecraft-version checks. Vanilla Java and native Bedrock intentionally do not expose a plugin/mod catalog.

[Learn about Plugins & Mods](docs/plugin-workspace.md)

## 🌐 Optional Minecraft Network

Several standalone servers are perfectly normal. They do **not** require a proxy.

When you want one public entrance to several compatible servers, JustVoxel also includes an optional **Minecraft Network** built around Velocity.

From the Control Center you can select supported backend servers, configure public Java and Bedrock ports, manage the virtual lobby and welcome message, and control Network membership.

Paper, Purpur, Pufferfish and Leaf are supported Network backends. Compatible Fabric/Quilt and Forge/NeoForge servers can also participate when JustVoxel proves the required proxy compatibility. Spigot, Vanilla and native Bedrock remain standalone.

[Learn about Minecraft Network](docs/minecraft-network.md)

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

[WebUI documentation](docs/WEBUI.md) · [mjust guide](docs/MJUST.md)

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

[Read the networking and Remote Access guide](docs/NETWORKING.md)

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

[Learn about AI & API Connections](docs/AI-API-CONNECTIONS.md)

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

[Telegram setup and permissions](docs/TELEGRAM.md)

### Discord

Discord can act as both a community-facing Minecraft surface and an administration surface.

Community members in the explicitly bound Discord server can see basic server status and request Minecraft access without automatically receiving administration rights.

Approved JustVoxel Discord administrators can manage supported server actions such as start, stop, restart, manual backup, update review, Minecraft Network controls, and additional advanced actions.

Discord membership, Minecraft whitelist access, and JustVoxel administration remain separate permissions.

[Discord setup, community access, and administration](docs/DISCORD.md)

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

[Storage](docs/STORAGE.md) · [Restore](docs/RESTORE.md) · [Migration](docs/MIGRATION.md)

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

[System management documentation](docs/SYSTEM.md)

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

## 📚 Documentation

The detailed appliance guides live in the [documentation directory](docs/).

Useful starting points:

- [WebUI](docs/WEBUI.md)
- [Minecraft runtime](docs/MINECRAFT_RUNTIME.md)
- [Server software](docs/server-software.md)
- [Plugins & Mods](docs/plugin-workspace.md)
- [Minecraft Network](docs/minecraft-network.md)
- [Networking & Remote Access](docs/NETWORKING.md)
- [AI & API Connections](docs/AI-API-CONNECTIONS.md)
- [Telegram](docs/TELEGRAM.md)
- [Discord](docs/DISCORD.md)
- [Storage](docs/STORAGE.md)
- [Restore](docs/RESTORE.md)
- [Migration](docs/MIGRATION.md)
- [System management](docs/SYSTEM.md)
- [mjust](docs/MJUST.md)

For image development, testing, and releases, see the [developer guide](README-DEV.md).

## ⛏️ Minecraft software and EULA

JustVoxel does **not** ship Minecraft server binaries, Mojang server software, a pre-created world, or a pre-accepted Minecraft EULA.

During server creation, the administrator accepts the Minecraft EULA and JustVoxel prepares the selected supported Minecraft server runtime.

Minecraft, Mojang software, Paper, Purpur, Spigot, Pufferfish, Leaf, Fabric, NeoForge, Forge, Quilt, plugins, mods, and other third-party components retain their own licenses and distribution terms.

## 📄 License

JustVoxel is licensed under **Apache-2.0**. See [LICENSE](LICENSE).

Third-party notices are maintained in [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).
