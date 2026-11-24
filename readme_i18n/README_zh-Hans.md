<p align="center">
  <img src="../docs/images/justvoxel-logo.png" width="260" alt="JustVoxel">
</p>

<p align="center">
  <a href="../README.md">English</a> •
  <a href="README_fr-CA.md">Français (Canada)</a> •
  <a href="README_fr-FR.md">Français (France)</a> •
  <a href="README_de.md">Deutsch</a> •
  <a href="README_pl.md">Polski</a> •
  <a href="README_pt-BR.md">Português (Brasil)</a> •
  <a href="README_es-419.md">Español (Latinoamérica)</a> •
  <a href="README_es-ES.md">Español (España)</a> •
  <a href="README_it.md">Italiano</a> •
  <a href="README_nl.md">Nederlands</a> •
  <a href="README_sv.md">Svenska</a> •
  <a href="README_nb.md">Norsk Bokmål</a> •
  <a href="README_da.md">Dansk</a> •
  <a href="README_tr.md">Türkçe</a> •
  <a href="README_id.md">Bahasa Indonesia</a> •
  <a href="README_uk.md">Українська</a> •
  <a href="README_ru.md">Русский</a> •
  <a href="README_zh-Hans.md">简体中文</a> •
  <a href="README_zh-Hant.md">繁體中文</a> •
  <a href="README_ja.md">日本語</a> •
  <a href="README_ko.md">한국어</a>
</p>

<!-- justvoxel-translation-notice -->

> **翻译说明：** 此 JustVoxel 译文以英文原文为依据，借助人工智能完成。如果发现不准确、不清楚或不自然的表述，欢迎在 GitHub 上提交[问题报告](https://github.com/home-server-project/justvoxel/issues)或[拉取请求](https://github.com/home-server-project/justvoxel/pulls)，并附上修改建议。

<p align="center">
  <strong>在自己的硬件、租用的独立服务器或托管虚拟机上运行 Minecraft 服务器，通过同一个控制中心管理。</strong>
</p>

<p align="center">
  这套自托管 Minecraft 服务器系统让你专注于运行游戏服务器，无需操心底层操作系统的日常管理。
</p>

<p align="center">
  <a href="#-安装-justvoxel">安装</a> •
  <a href="#-轻松托管-minecraft-服务器">功能</a> •
  <a href="../docs/INSTALLER.md">文档</a>
</p>

<p align="center">
  <img src="../docs/images/justvoxel-overview-1.webp" width="100%" alt="JustVoxel 控制中心概览"><img src="../docs/images/justvoxel-overview-2.webp" width="100%" alt=""><img src="../docs/images/justvoxel-overview-3.webp" width="100%" alt="">
</p>

JustVoxel 是一套专为自托管 Minecraft 服务器设计的系统。你可以将它安装在兼容的台式机、笔记本电脑、物理服务器或虚拟机上，也可以使用允许自定义操作系统镜像的托管服务。安装后即可通过浏览器管理 Minecraft。

无需维护一堆脚本，也无需另行安装 Minecraft 管理面板。

手头有闲置的笔记本或台式机？**用 JustVoxel HWS 就能把它变成 Minecraft 专用服务器，让旧硬件继续派上用场。** 在支持的笔记本上，可以关闭合盖休眠，让服务器在合盖后继续运行。也可以通过本地控制台立即关闭屏幕，或让屏幕在短暂延时后关闭。

> 🚧 **预发布提示：** JustVoxel 正在为首次公开发布做准备。最终发布验证仍在进行，当前的安装 ISO 可供测试和早期试用。

## ✨ 轻松托管 Minecraft 服务器

JustVoxel 将 Minecraft 管理功能与底层操作系统整合在一起。

- 🎮 **多台 Minecraft 服务器** — 在同一台设备上独立运行不同世界和不同类型的服务器。
- 🖥️ **网页控制中心** — 通过电脑、平板或手机管理服务器。
- 🧩 **插件与模组** — 从支持的平台查找并管理兼容的内容。
- 💾 **备份与还原** — 通过定时备份和手动备份保护每台服务器。
- 🌐 **远程访问** — 四种内置方案可供选择，支持私有网络访问或游戏隧道。
- 🤖 **AI 与自动化** — 通过 MCP 和 REST 连接兼容的 AI 助手和应用。
- 💬 **Telegram 与 Discord** — 可选的机器人控制、通知和社区管理功能。
- 🛡️ **系统管理** — 管理网络、存储和更新，查看系统健康状况和日志，执行支持的电源操作，无需学习底层 Linux 命令。

JustVoxel 适合家庭、好友、小型 Minecraft 社区，以及希望拥有自己的服务器、无需租用传统 Minecraft 托管面板的用户。

## 🖥️ 一套系统，两个版本

### JustVoxel VM

适用于虚拟机和托管环境，例如：

- Proxmox
- KVM / libvirt
- VMware
- Hyper-V
- VirtualBox
- 允许使用兼容自定义操作系统镜像的托管虚拟机或 VPS

VM 版提供完整的 Minecraft 专用系统，省去了仅在物理机上才有用的软件包。

### JustVoxel HWS

适合直接安装在物理硬件上，包括：

- 旧笔记本电脑
- 改作服务器的台式机
- 迷你电脑
- 家用服务器
- 允许使用自定义安装介质的租用裸机服务器

HWS 增加了物理机支持，包括 UPS/NUT 集成、SMART 和 NVMe 工具、传感器、固件/UEFI 控制、设备固件、硬件诊断，以及笔记本屏幕和合盖行为设置。

两个版本的 JustVoxel 控制中心、管理 API、Minecraft 操作流程、存储模型、备份、网络功能和更新方式均相同。

[了解 VM 与 HWS 的功能差异](../docs/VARIANTS.md)

## 🚀 安装 JustVoxel

同一个安装 ISO 支持 JustVoxel 的两个版本。

[⬇️ 下载 JustVoxel x86_64 ISO](https://sourceforge.net/projects/justvoxel/files/JustVoxel-x86_64.iso/download)

根据安装环境选择版本：

- **JustVoxel VM** 适用于虚拟机及兼容的托管虚拟机环境
- **JustVoxel HWS** 适用于台式机、笔记本、迷你电脑、家用物理服务器及兼容的裸机托管环境

有关安装、磁盘准备和首次登录的说明，请参阅[安装指南](../docs/INSTALLER.md)。

## 🏠 以 Minecraft 为主，也能运行其他服务

JustVoxel 的设计和支持重点是 Minecraft 专用服务器。底层仍是标准的 **AlmaLinux 10 bootc 服务器，配有 Podman、Quadlet、systemd、SSH、NetworkManager 和标准 Linux 管理工具**。

高级用户可以把同一台机器用作多用途家用服务器，在运行 Minecraft 的同时运行其他容器。

例如，你可以手动部署以下服务：

- **Jellyfin** 或 **Plex**，用于管理和播放媒体
- **Vaultwarden**，用于管理密码
- **Immich**，用于管理照片和视频
- **Home Assistant Container**，用于家庭自动化
- **Nextcloud**，用于文件管理与协作
- **Syncthing** 或 **Navidrome**
- **OpenClaw**，用作自托管 AI 智能体或网关

这些应用**不由 JustVoxel 控制中心管理**。管理员自行负责其 Podman 容器、Quadlet、存储、端口、更新、备份、反向代理配置和各应用的特定要求。

JustVoxel 的管理范围很明确：它负责 Minecraft 专用系统，高级用户仍可按常规 Podman 和 Quadlet 流程管理其他服务。

维护时，JustVoxel 也会将无关容器视为管理员自己的服务，与 Minecraft 资源分开处理。Minecraft 的**重新开始**等操作不会删除无关的 Podman 容器。

因此，这台机器可以主要用于运行 Minecraft，也可以按需兼作常规的容器化家用服务器。

## 🎮 创建适合你的 Minecraft 服务器

只要机器有足够的内存和可用端口，JustVoxel 就能同时运行多台独立的 Minecraft 服务器。

每台服务器都有独立的标识、配置、运行环境、端口、数据、备份、启停控制和状态。

### 当前支持的服务器软件

| 服务器软件 | 扩展内容 | 由系统管理的 Java → 基岩版跨版本联机 |
| --- | --- | --- |
| **Paper** | 插件 | ✅ |
| **Purpur** | 插件 | ✅ |
| **Spigot** | 插件 | ✅ |
| **Pufferfish** | 插件 | ✅ |
| **Leaf** | 插件 | ✅ |
| **Fabric** | 模组 | ✅ 仅在确认所选版本完全兼容时支持 |
| **NeoForge** | 模组 | ✅ 仅在确认所选版本完全兼容时支持 |
| **Forge** | 模组 | 仅 Java 版 |
| **Quilt** | 模组 | 仅 Java 版 |
| **Vanilla Java** | 无插件或模组 | 仅 Java 版 |
| **Native Bedrock** | 原生基岩版服务器 | 原生基岩版客户端 |

选择**简易设置**，即可按引导创建第一台服务器。若希望自行配置内存、端口、存储、备份、版本策略及兼容的跨版本联机功能，请选择**高级设置**。

JustVoxel 会根据所选服务器软件核实各项功能是否可用。

[查看服务器软件及兼容性详情](../docs/server-software.md)

## 🧩 插件与模组

选中服务器后，打开 **Minecraft → 插件与模组**。

根据服务器类型，JustVoxel 可以使用以下平台：

- [Modrinth](https://modrinth.com/)
- [Hangar](https://hangar.papermc.io/)
- [CurseForge](https://www.curseforge.com/minecraft)（需配置自己的 API 密钥）

对于支持的安装操作，系统会检查兼容性、验证下载文件、执行停服备份、验证重启结果、跟踪内容归属，并提供恢复保护。

Paper 系列服务器可使用插件目录。Fabric、Forge、NeoForge 和 Quilt 使用模组目录，系统会严格核对加载器和 Minecraft 版本。按照设计，原版 Java 服务器和原生基岩版服务器不提供插件或模组目录。

[了解插件与模组](../docs/plugin-workspace.md)

## 🌐 可选的 Minecraft Network

多台服务器可以各自独立运行，**无需**代理。

如果希望为多台兼容服务器提供统一的公共入口，可使用 JustVoxel 内置的 **Minecraft Network**。这项可选功能基于 Velocity。

在控制中心，你可以选择支持的后端服务器，配置对外开放的 Java 版和基岩版端口，管理虚拟大厅和欢迎消息，并决定哪些服务器加入网络。

Minecraft Network 支持将 Paper、Purpur、Pufferfish 和 Leaf 用作后端。对于 Fabric/Quilt 和 Forge/NeoForge，JustVoxel 确认服务器满足代理兼容性要求后，也允许其加入。Spigot、Vanilla 和原生基岩版服务器仍然独立运行。

[了解 Minecraft Network](../docs/minecraft-network.md)

## 🖥️ 认识控制中心

JustVoxel 按日常服务器管理任务组织功能，无需熟悉 Linux 内部机制。

### Minecraft 服务器

创建和管理多台服务器，启动或停止服务器，查看运行状态和玩家信息，调整支持的设置，并使用各台服务器专属的工具。

### 插件与模组

查找兼容内容，查看已安装的插件和模组，管理支持的扩展内容。若某次更改需要处理，可查看相应的恢复信息。

### 备份与存储

创建备份，查看备份历史，还原完整的服务器数据，配置备份位置，导入或导出服务器，并管理支持的存储。

### 网络

管理以太网、Wi-Fi 和远程访问，进行**连接故障排查**。对于可能中断连接的网络更改，系统提供回滚保护，避免因 IP 或 Wi-Fi 设置错误而长期无法访问设备。

### 系统

查看系统健康状况、资源和日志，管理操作系统更新、用户和安全设置，执行支持的电源操作，并使用 HWS 专属的硬件控制功能。

### AI 与 API 连接

连接支持的 AI 应用、MCP 客户端、REST 自动化工具、Telegram 和 Discord，无需另建一套管理系统。

更喜欢终端？JustVoxel 还提供交互式终端管理界面 **mjust**，沿用同一套设备安全机制。

[WebUI 文档](../docs/WEBUI.md) · [mjust 指南](../docs/MJUST.md)

## 🌐 选择适合好友的连接方式

JustVoxel 目前在**网络 → 远程访问**中集成了四种方案。

除了预装软件包，JustVoxel 还根据各服务的特点提供状态查看、启用、停用及设置流程。

| 方案 | 适用场景 | 提供的功能 |
| --- | --- | --- |
| **[Tailscale](https://tailscale.com/)** | 亲友使用的私有服务器 | 在获准设备之间建立私有网状网络 |
| **[NetBird](https://netbird.io/)** | 需要管理访问权限的私有网络 | 通过账户和访问策略控制私有网络访问 |
| **[Playit.gg](https://playit.gg/)** | 无需在路由器上设置常规端口转发，即可对外开放 Minecraft | 公共游戏隧道和连接地址 |
| **[NordVPN Meshnet](https://meshnet.nordvpn.com/)** | NordVPN 用户在设备间进行私有访问 | 通过 Meshnet 互联，无需将 JustVoxel 用作商业 VPN 客户端 |

### 应该选哪一种？

**🎮 希望好友无需安装专用 VPN 客户端就能加入游戏？**  
可以先看看 **Playit.gg**。配置好游戏隧道后，玩家即可通过其公共地址连接。

**🔒 想让私有服务器仅供受信任的设备访问？**  
可以考虑 **Tailscale** 或 **NetBird**。

**🌍 已经在使用 NordVPN Meshnet？**  
JustVoxel 可以管理设备上的 Meshnet 集成。这与将设备连接到常规商业 NordVPN 出口服务器是两项独立的功能。

服务账户、云端策略和隧道配置仍由相应的服务商管理。JustVoxel 负责设备端支持的集成功能。

可以同时使用多种 VPN 或网状网络产品，但其路由、DNS 或防火墙规则可能发生冲突。

[阅读网络与远程访问指南](../docs/NETWORKING.md)

## 🤖 AI、MCP、REST 与自动化

想请 AI 助手查看服务器状态，接入家庭自动化系统，或搭建自己的管理流程？

JustVoxel 通过 **AI 与 API 连接**提供内置的 **MCP 和 REST 接口**。

设置时可以选择兼容的应用与连接类型，例如：

- **OpenAI — ChatGPT 与 Codex**
- **Claude**
- **Grok**
- **Gemini**
- **GitHub Copilot**
- **其他 AI / MCP 应用**
- **自动化 / REST API**

你可以决定每个连接能够访问哪些资源。

权限从**级别 1 — 只读**起，可扩展到服务器管理。只有获得明确批准，连接才可执行高级系统控制操作。你也可以将连接的访问范围限定为选定的 Minecraft 服务器。

JustVoxel 开放 MCP 或 REST 接口本身无需 AI 模型 API 密钥。客户端实际可用的功能取决于应用、账户或订阅权限以及网络连接条件。

AI/API 网关默认关闭，需手动启用。

[了解 AI 与 API 连接](../docs/AI-API-CONNECTIONS.md)

## 💬 Telegram 与 Discord

JustVoxel 内置可选的 Telegram 和 Discord 集成，两者均与 WebUI 共用同一个负责执行管理操作的后端。

### Telegram

专用 Telegram 机器人可以提供以下功能：

- 查看设备和 Minecraft 的状态
- 选择服务器并查看玩家信息
- 允许获准的管理用户启动、停止和重启服务器
- 手动备份
- 查看 Minecraft 和容器更新
- 查看操作系统更新信息
- 按受控流程重启设备
- 按需接收服务器状态、备份结果和可用更新通知

Telegram 单独设置**只读**和**服务器管理**权限，无需使用 AI 模型。

[Telegram 设置与权限](../docs/TELEGRAM.md)

### Discord

Discord 既可以为 Minecraft 社区成员提供服务，也可以作为管理员的操作入口。

在已明确绑定的 Discord 服务器中，社区成员可以查看基本的服务器状态并申请 Minecraft 访问权限，但不会因此自动获得管理权限。

经 JustVoxel 批准的 Discord 管理员可以执行支持的操作，包括启动、停止和重启服务器，手动备份，查看更新，管理 Minecraft Network，以及执行其他高级操作。

Discord 成员资格、Minecraft 白名单准入和 JustVoxel 管理权限彼此独立。

[Discord 设置、社区访问与管理](../docs/DISCORD.md)

## 💾 备份、还原、存储与迁移

Minecraft 数据比设备的系统镜像更重要。

JustVoxel 将 Minecraft 数据和备份存储在不可变操作系统部署之外，两者的存储位置可以分别选择。

支持以下存储方式：

- 系统文件系统
- 专用内置磁盘
- USB 存储设备
- 现有本地分区或文件系统
- 在已有的未分配空间上创建的新分区
- NFS
- SMB/CIFS

每台服务器都支持自动备份和手动备份。还原会替换所选服务器的全部持久化数据，同时保留其 JustVoxel 标识和设备配置。

如需在不同机器间迁移服务器或在重装后迁回服务器，JustVoxel 还提供经过审查的**导出、导入与恢复**流程。

[存储](../docs/STORAGE.md) · [还原](../docs/RESTORE.md) · [迁移](../docs/MIGRATION.md)

## 🛡️ 安全地更改配置

JustVoxel 的网页界面不会任意执行 shell 命令。

系统为支持的操作提供以下安全保障：

- 在可能中断游戏的操作前检查在线玩家
- 应用更改前先核对更改内容
- 在需要恢复保障的更改前执行停服备份
- 严格核对服务器标识，防止误将一台服务器当成另一台
- 验证重启结果和运行状态
- 持久保存操作与恢复状态
- 为支持的操作提供回滚途径
- 对有风险的网络更改使用 NetworkManager 检查点
- 分别处理 Minecraft 和操作系统更新

浏览器访问的 WebUI 以非特权身份运行。需要特权的更改由本机 JustVoxel Management Agent 通过专门定义的 API 执行，浏览器无法直接访问 root shell。

## 🔄 分别管理系统更新和 Minecraft 更新

JustVoxel 是一套通过镜像管理的专用系统，基于 Home Server Base 10 / AlmaLinux 10 构建。

操作系统更新与 Minecraft 版本、服务器软件、插件、模组和容器更新分别处理。

更新设备系统不会在你不知情的情况下更改 Minecraft 版本，调整 Minecraft 服务器也无需重新构建宿主操作系统。

[系统管理文档](../docs/SYSTEM.md)

## 📊 JustVoxel 与其他项目有何不同？

JustVoxel 与传统 Minecraft 控制面板的定位不同：**操作系统与 Minecraft 管理层共同组成一套专用系统**。

| 功能 | **JustVoxel** | **MineOS SvelteKit** | **Crafty Controller** | **MineOS Node** |
| --- | :---: | :---: | :---: | :---: |
| 完整的 Minecraft 专用操作系统 | ⭐ | — | — | — |
| 多台 Minecraft 服务器 | ✅ | ✅ | ✅ | ✅ |
| Java 版和基岩版服务器支持 | ✅ | ✅ | ✅ | 🟡 支持 Java 版及旧版 PocketMine |
| 引导式插件和模组查找 | ✅ Modrinth、Hangar、CurseForge | ✅ Modrinth、CurseForge | — | — |
| 宿主机网络、存储与操作系统管理 | ⭐ | — | — | — |
| 集成私有访问和游戏隧道 | ⭐ 4 种方案 | — | — | — |
| MCP + REST AI 与自动化网关 | ⭐ | — | — | — |
| 原生 Telegram 和 Discord 管理功能 | ⭐ | — | — | — |
| 笔记本合盖行为和屏幕控制 | ✅ HWS | — | — | — |
| 基于镜像的不可变操作系统更新 | ✅ | — | — | — |
| 终端管理界面 | ✅ mjust | ✅ mineos CLI/TUI | — | ✅ CLI |
| 安装方式 | 完整的 bootc 专用系统 | Docker / Compose 应用 | 运行在 Docker、Linux 或 Windows 上的控制面板 | 运行在 Linux/BSD 上的 Node.js 应用 |

⭐ 特色功能 &nbsp;•&nbsp; ✅ 支持 &nbsp;•&nbsp; 🟡 部分支持或仅支持旧版 &nbsp;•&nbsp; — 所查阅的项目核心文档未涵盖

**对比信息最后核对日期：2026 年 11 月 16 日。** 这份对比并非要否定其他项目。如果你已有操作系统，并希望自行管理，MineOS 和 Crafty 都可能是很合适的 Minecraft 管理应用。JustVoxel 面向希望将 Minecraft 面板**连同底层服务器系统**作为一个产品统一管理的用户。

对比依据为各项目当前的文档和代码仓库：

- [MineOS SvelteKit](https://github.com/freeman412/mineos-sveltekit)
- [Crafty Controller](https://craftycontrol.com/) · [源代码](https://gitlab.com/crafty-controller/crafty-4)
- [MineOS Node](https://github.com/hexparrot/mineos-node)

如果对比内容已经过时或有失公允，请提交 issue。即使更正后的对比对 JustVoxel 不利，我们也同样欢迎。

## 📚 文档

详细的系统使用指南位于[文档目录](../docs/)中。

可以从以下主题入手：

- [WebUI](../docs/WEBUI.md)
- [Minecraft 运行环境](../docs/MINECRAFT_RUNTIME.md)
- [服务器软件](../docs/server-software.md)
- [插件与模组](../docs/plugin-workspace.md)
- [Minecraft Network](../docs/minecraft-network.md)
- [网络与远程访问](../docs/NETWORKING.md)
- [AI 与 API 连接](../docs/AI-API-CONNECTIONS.md)
- [Telegram](../docs/TELEGRAM.md)
- [Discord](../docs/DISCORD.md)
- [存储](../docs/STORAGE.md)
- [还原](../docs/RESTORE.md)
- [迁移](../docs/MIGRATION.md)
- [系统管理](../docs/SYSTEM.md)
- [mjust](../docs/MJUST.md)

有关镜像开发、测试和发布的信息，请参阅[开发者指南](../README-DEV.md)。

## ⛏️ Minecraft 软件与 EULA

JustVoxel **不附带** Minecraft 服务器二进制文件、Mojang 服务器软件或预先创建的世界，也不会预先代为接受 Minecraft 最终用户许可协议（EULA）。

创建服务器时，管理员需接受 Minecraft EULA，然后由 JustVoxel 准备所选的受支持 Minecraft 服务器运行环境。

Minecraft、Mojang 软件、Paper、Purpur、Spigot、Pufferfish、Leaf、Fabric、NeoForge、Forge、Quilt、插件、模组及其他第三方组件均适用各自的许可证和分发条款。

## 📄 许可证

JustVoxel 采用 **Apache-2.0** 许可证。详见 [LICENSE](../LICENSE)。

第三方声明见 [THIRD_PARTY_NOTICES.md](../THIRD_PARTY_NOTICES.md)。
