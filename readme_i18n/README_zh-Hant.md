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

> **翻譯說明：** 此 JustVoxel 譯文是以英文原文為依據，借助人工智慧完成的。如果發現用詞不準確、不清楚或不自然，歡迎在 GitHub 上提交[問題回報](https://github.com/home-server-project/justvoxel/issues)或[合併請求（pull request）](https://github.com/home-server-project/justvoxel/pulls)，並附上修正建議。

<p align="center">
  <strong>在自己的硬體、租用的獨立伺服器或託管的虛擬機器上執行 Minecraft 伺服器，透過同一個控制中心管理。</strong>
</p>

<p align="center">
  這套自行架設的 Minecraft 伺服器系統，讓你專心管理遊戲伺服器，無須自行管理底層作業系統。
</p>

<p align="center">
  <a href="#-安裝-justvoxel">安裝</a> •
  <a href="#-輕鬆架設-minecraft-伺服器">功能</a> •
  <a href="../docs/INSTALLER.md">文件</a>
</p>

<p align="center">
  <img src="../docs/images/justvoxel-overview-1.webp" width="100%" alt="JustVoxel 控制中心概覽"><img src="../docs/images/justvoxel-overview-2.webp" width="100%" alt=""><img src="../docs/images/justvoxel-overview-3.webp" width="100%" alt="">
</p>

JustVoxel 是一套專為自行架設 Minecraft 伺服器打造的系統。你可以將它安裝在相容的個人電腦、筆記型電腦、實體伺服器或虛擬機器上，也可以使用允許執行自訂作業系統映像的託管服務。安裝後即可透過瀏覽器管理 Minecraft。

無須維護一堆指令碼，也無須另外安裝 Minecraft 管理面板。

手邊有閒置的筆記型電腦或個人電腦？**JustVoxel HWS 可以把它變成 Minecraft 專用伺服器，讓舊硬體繼續派上用場。** 在支援的筆記型電腦上，可以停用闔蓋睡眠，讓伺服器在螢幕闔上後繼續執行。也可以透過本機主控台立即關閉螢幕，或在短暫延遲後關閉。

> 🚧 **預發行：** JustVoxel 正在為首次公開發行做準備。最終發行驗證仍在進行中，目前的安裝 ISO 可供測試和提早試用。

## ✨ 輕鬆架設 Minecraft 伺服器

JustVoxel 將 Minecraft 管理功能與底層作業系統整合在一起。

- 🎮 **多個 Minecraft 伺服器** — 在同一台設備上獨立執行不同世界和不同類型的伺服器。
- 🖥️ **網頁控制中心** — 透過電腦、平板或手機管理伺服器。
- 🧩 **外掛與模組** — 從支援的供應商探索並管理相容的內容。
- 💾 **備份與還原** — 透過排程備份和手動備份保護每個伺服器。
- 🌐 **遠端存取** — 可選用四種內建的私人存取與遊戲隧道方案。
- 🤖 **AI 與自動化** — 透過 MCP 和 REST 連接相容的 AI 助理與應用程式。
- 💬 **Telegram 與 Discord** — 可選用機器人控制、通知和社群管理功能。
- 🛡️ **系統管理** — 管理網路、儲存空間及更新，查看健康狀態與記錄，並執行支援的電源操作，無須學習底層 Linux 指令。

JustVoxel 適合家庭、朋友、小型 Minecraft 社群，以及希望擁有自己的伺服器、無須租用傳統 Minecraft 託管面板的人。

## 🖥️ 一套系統，兩個版本

### JustVoxel VM

適用於虛擬機器和託管環境，例如：

- Proxmox
- KVM / libvirt
- VMware
- Hyper-V
- VirtualBox
- 允許使用相容自訂作業系統映像的託管虛擬機器或 VPS 方案

VM 版提供完整的 Minecraft 伺服器系統，省去虛擬機器內用不到的實體硬體套件。

### JustVoxel HWS

適合直接安裝在實體硬體上，包括：

- 舊的筆記型電腦
- 改作他用的桌上型電腦
- 迷你電腦
- 家用伺服器
- 允許使用自訂安裝媒體的租用裸機伺服器

HWS 加入實體機器支援，包括 UPS/NUT 整合、SMART 和 NVMe 工具、感測器、韌體／UEFI 控制、裝置韌體、硬體診斷，以及筆記型電腦的螢幕和上蓋控制功能。

兩個版本使用相同的 JustVoxel 控制中心、管理 API、Minecraft 操作流程、儲存架構、備份功能、網路功能與更新流程。

[瞭解 VM 與 HWS 的功能差異](../docs/VARIANTS.md)

## 🚀 安裝 JustVoxel

同一個安裝 ISO 支援兩個 JustVoxel 版本。

[⬇️ 下載 JustVoxel x86_64 ISO](https://sourceforge.net/projects/justvoxel/files/JustVoxel-x86_64.iso/download)

**製作可開機的 USB 隨身碟：** 使用 [balenaEtcher](https://etcher.balena.io/)、[Rufus](https://rufus.ie/en/)、[Ventoy](https://www.ventoy.net/en/index.html)、[Fedora Media Writer](https://fedoraproject.org/workstation/download/) 或其他能直接寫入磁碟映像檔的工具，將 ISO 寫入 USB 隨身碟。在 Linux 和其他類 Unix 系統上，也可以使用傳統的 `dd` 方法。直接將映像檔寫入 USB 隨身碟會清除隨身碟上的所有資料。

依安裝環境選擇適合的版本：

- **JustVoxel VM** 適用於虛擬機器及相容的託管虛擬機器環境
- **JustVoxel HWS** 適用於個人電腦、筆記型電腦、迷你電腦、實體家用伺服器，以及相容的裸機託管環境

請參閱[安裝指南](../docs/INSTALLER.md)，瞭解安裝、磁碟準備和首次登入的步驟。

## 🏠 以 Minecraft 為主，也能執行其他服務

JustVoxel 的設計和支援以 Minecraft 伺服器為主，底層仍是一般的 **AlmaLinux 10 bootc 伺服器，搭配 Podman、Quadlet、systemd、SSH、NetworkManager 和標準 Linux 管理工具**。

進階使用者也可以把同一台機器當作多用途家用伺服器，在執行 Minecraft 的同時執行其他容器。

例如，你可以手動部署下列服務：

- **Jellyfin** 或 **Plex**，用於管理和播放媒體
- **Vaultwarden**，用於管理密碼
- **Immich**，用於管理相片與影片
- **Home Assistant Container**，用於居家自動化
- **Nextcloud**，用於檔案管理與協作
- **Syncthing** 或 **Navidrome**
- **OpenClaw**，作為自行架設的 AI 代理程式或閘道

這些應用程式**不由 JustVoxel 控制中心管理**。其 Podman 容器、Quadlet、儲存空間、連接埠、更新、備份、反向代理設定，以及應用程式各自的需求，仍由管理員負責。

JustVoxel 的分工很單純：它負責管理 Minecraft 伺服器系統，進階使用者仍可用一般的 Podman 和 Quadlet 操作流程管理其他服務。

JustVoxel 在維護時也會將與 Minecraft 無關的容器視為由管理員負責的工作負載。執行 Minecraft 的**重新開始**等操作時，不會移除無關的 Podman 容器。

這樣一來，JustVoxel 主機可以繼續以 Minecraft 為主要用途，也能在需要時當作一般的容器化家用伺服器。

## 🎮 建立你想要的 Minecraft 伺服器

只要主機有足夠的記憶體和可用的連接埠，JustVoxel 就能同時執行多個獨立的 Minecraft 伺服器。

每個伺服器都有自己的識別資訊、設定、執行環境、連接埠、資料、備份、生命週期控制功能和狀態。

### 目前支援的伺服器軟體

| 伺服器軟體 | 可用內容類型 | 由 JustVoxel 管理的 Java → Bedrock 跨平台遊玩 |
| --- | --- | --- |
| **Paper** | 外掛 | ✅ |
| **Purpur** | 外掛 | ✅ |
| **Spigot** | 外掛 | ✅ |
| **Pufferfish** | 外掛 | ✅ |
| **Leaf** | 外掛 | ✅ |
| **Fabric** | 模組 | ✅ 確認所用組合相容時 |
| **NeoForge** | 模組 | ✅ 確認所用組合相容時 |
| **Forge** | 模組 | 僅限 Java |
| **Quilt** | 模組 | 僅限 Java |
| **Vanilla Java** | 無外掛或模組 | 僅限 Java |
| **原生 Bedrock** | 原生 Bedrock 伺服器 | 原生 Bedrock 用戶端 |

選擇**簡易設定**，依引導建立第一個伺服器。如果想自行調整記憶體、連接埠、儲存空間、備份、版本策略和相容的跨平台遊玩選項，請選擇**進階設定**。

JustVoxel 會根據你選擇的伺服器軟體，檢查各項功能是否可用。

[查看伺服器軟體與相容性詳細資料](../docs/server-software.md)

## 🧩 外掛與模組

選取伺服器後，開啟 **Minecraft → 外掛與模組**。

依伺服器類型而定，JustVoxel 可使用下列來源：

- [Modrinth](https://modrinth.com/)
- [Hangar](https://hangar.papermc.io/)
- [CurseForge](https://www.curseforge.com/minecraft)，需使用你設定的 API 金鑰

安裝受支援的內容時，JustVoxel 會檢查相容性、驗證下載內容、執行停機備份、驗證重新啟動是否成功、追蹤內容的歸屬，並提供復原保護。

Paper 系列伺服器可使用外掛目錄。Fabric、Forge、NeoForge 和 Quilt 使用模組目錄，並核對模組載入器和 Minecraft 的確切版本。Vanilla Java 和原生 Bedrock 刻意不提供外掛或模組目錄。

[瞭解外掛與模組](../docs/plugin-workspace.md)

## 🌐 選用的 Minecraft 網路

執行多個獨立伺服器是正常的用法，**無須**使用代理伺服器。

如果你想讓多個相容伺服器共用一個公開入口，JustVoxel 也提供以 Velocity 為基礎的選用功能 **Minecraft 網路**。

你可以在控制中心選擇支援的後端伺服器、設定對外的 Java 和 Bedrock 連接埠、管理虛擬大廳和歡迎訊息，以及管理哪些伺服器加入網路。

Paper、Purpur、Pufferfish 和 Leaf 都可作為網路的後端。相容的 Fabric／Quilt 和 Forge／NeoForge 伺服器，也可以在 JustVoxel 確認它們符合代理伺服器的相容性要求後加入。Spigot、Vanilla 和原生 Bedrock 則維持獨立運作。

[瞭解 Minecraft 網路](../docs/minecraft-network.md)

## 🖥️ 認識控制中心

JustVoxel 的介面以伺服器管理工作為主，將 Linux 的底層細節留在幕後。

### Minecraft 伺服器

建立和管理多個伺服器、啟動或停止伺服器、查看狀態和玩家資訊、調整支援的設定，並使用各伺服器專屬的工具。

### 外掛與模組

探索相容內容、查看已安裝的外掛與模組、管理支援的新增內容，並在變更需要處理時查看復原資訊。

### 備份與儲存

建立備份、查看備份歷程、還原完整的伺服器資料、設定備份目的地、匯入或匯出伺服器，以及管理支援的儲存空間。

### 網路

管理乙太網路、Wi-Fi 和遠端存取，並**排解連線問題**。可能中斷連線的網路變更有回復保護，避免因為 IP 或 Wi-Fi 設定錯誤而永久失去存取能力。

### 系統

查看健康狀態、資源、記錄、作業系統更新、使用者、安全性、支援的電源操作，以及 HWS 專屬的硬體控制功能。

### AI 與 API 連線

連接支援的 AI 應用程式、MCP 用戶端、REST 自動化工具、Telegram 和 Discord，無須另外架設管理系統。

習慣使用終端機？JustVoxel 也提供 **mjust**，這套互動式終端機管理介面採用相同的安全機制。

[WebUI 文件](../docs/WEBUI.md) · [mjust 指南](../docs/MJUST.md)

## 🌐 選擇適合你的方式，讓朋友連線

JustVoxel 目前在**網路 → 遠端存取**中整合了四種選項。

這些選項均已整合到 JustVoxel。依供應商而定，你可以查看支援的狀態資訊、啟用或停用服務，並依照設定流程完成設定。

| 選項 | 適合的用途 | 提供的功能 |
| --- | --- | --- |
| **[Tailscale](https://tailscale.com/)** | 親友專用的私人伺服器 | 讓獲准的裝置透過私人網狀網路互連 |
| **[NetBird](https://netbird.io/)** | 需要管理存取權的私人網路 | 可管理帳號及存取原則的私人網路 |
| **[Playit.gg](https://playit.gg/)** | 對外開放 Minecraft，無須在路由器上進行一般的連接埠轉送 | 公開的遊戲隧道與連線位址 |
| **[NordVPN Meshnet](https://meshnet.nordvpn.com/)** | NordVPN 使用者的裝置對裝置私人存取 | 提供 Meshnet 連線，無須將 JustVoxel 當作商用 VPN 用戶端 |

### 我該選哪一種？

**🎮 希望朋友無須安裝私人 VPN 用戶端就能加入？**  
可以先看看 **Playit.gg**。玩家可以使用你所設定的遊戲隧道提供的公開位址。

**🔒 想讓受信任的裝置存取私人伺服器？**  
可以考慮 **Tailscale** 或 **NetBird**。

**🌍 已經在使用 NordVPN Meshnet？**  
JustVoxel 可以管理與 Meshnet 的整合。這和把主機連到一般的 NordVPN 商用出口伺服器是兩回事。

供應商帳號、雲端政策和隧道設定仍由供應商管理。JustVoxel 負責主機端支援的整合功能。

可以同時使用多種 VPN 或網狀網路產品，但路由、DNS 或防火牆規則可能發生衝突。

[閱讀網路與遠端存取指南](../docs/NETWORKING.md)

## 🤖 AI、MCP、REST 與自動化

想向 AI 助理查詢伺服器狀態、串接居家自動化，或建立自己的控制流程嗎？

JustVoxel 透過 **AI 與 API 連線**提供內建的 **MCP 和 REST 介面**。

設定時可選擇相容的應用程式和連線類型，例如：

- **OpenAI — ChatGPT 與 Codex**
- **Claude**
- **Grok**
- **Gemini**
- **GitHub Copilot**
- **其他 AI／MCP 應用程式**
- **自動化／REST API**

每個連線可以存取哪些內容，由你決定。

權限從**第 1 級 — 唯讀**到伺服器管理功能不等。進階系統控制必須經過明確核准。你也可以將連線限制在指定的 Minecraft 伺服器。

JustVoxel 提供 MCP 或 REST 介面本身無須使用 AI 模型的 API 金鑰。用戶端實際能否使用，取決於應用程式、帳號或訂閱方案提供的功能，以及網路存取條件。

AI／API 閘道預設為停用，直到你手動啟用。

[瞭解 AI 與 API 連線](../docs/AI-API-CONNECTIONS.md)

## 💬 Telegram 與 Discord

JustVoxel 原生支援選用的 Telegram 和 Discord 整合，兩者與 WebUI 共用同一套管理後端，由該後端統一處理操作。

### Telegram

專用的 Telegram 機器人可提供下列功能：

- 主機和 Minecraft 狀態
- 選擇伺服器及查看玩家資訊
- 讓已獲管理權限的使用者啟動、停止或重新啟動伺服器
- 手動備份
- 檢視 Minecraft 和容器更新
- 查看作業系統更新資訊
- 依受控流程重新啟動主機
- 選用通知，包括伺服器狀態、備份結果及可用更新

Telegram 使用自己的**唯讀**和**伺服器管理**權限，無須使用 AI 模型。

[Telegram 設定與權限](../docs/TELEGRAM.md)

### Discord

Discord 既可作為 Minecraft 社群的互動介面，也可作為管理介面。

在明確綁定的 Discord 伺服器中，社群成員可以查看遊戲伺服器的基本狀態並申請 Minecraft 存取權，不會因此自動獲得管理權限。

已獲核准的 JustVoxel Discord 管理員可以執行支援的伺服器操作，例如啟動、停止、重新啟動、手動備份、檢視更新、管理 Minecraft 網路，以及其他進階操作。

Discord 成員資格、Minecraft 白名單存取權和 JustVoxel 管理權限各自獨立。

[Discord 設定、社群存取和管理](../docs/DISCORD.md)

## 💾 備份、還原、儲存與移轉

Minecraft 資料比系統映像更重要。

JustVoxel 將 Minecraft 資料和備份存放在不可變的作業系統部署之外，兩者的存放位置可以分別選擇。

支援的儲存選項包括：

- 系統檔案系統
- 專用的內接磁碟
- USB 儲存裝置
- 現有的本機磁碟分割區或檔案系統
- 使用原本未配置的空間建立的新分割區
- NFS
- SMB/CIFS

每個伺服器都能使用自動和手動備份。還原會取代所選伺服器的完整持久化資料，同時保留該伺服器在 JustVoxel 中的識別資訊及系統層級的設定。

若要在不同主機之間移轉伺服器，或在重新安裝時搬移伺服器，JustVoxel 也提供經過審查的**匯出／匯入／復原**流程。

[儲存](../docs/STORAGE.md) · [還原](../docs/RESTORE.md) · [移轉](../docs/MIGRATION.md)

## 🛡️ 讓變更更安全

JustVoxel 的網頁介面提供受控的管理功能，無法任意執行 shell 指令。

支援的操作遵循系統層級的安全規則，包括：

- 執行可能造成中斷的操作前，檢查是否有玩家在線
- 套用前先檢視變更
- 變更需要復原保護時，先執行停機備份
- 精確核對伺服器的識別資訊，避免把一個伺服器誤當成另一個
- 驗證重新啟動結果及執行狀態
- 持久保存操作與復原狀態
- 為支援的操作提供回復途徑
- 為高風險的網路變更建立 NetworkManager 檢查點
- 將 Minecraft 和作業系統的更新流程分開

面向瀏覽器的 WebUI 以非特權身分執行。需要特權的變更會透過本機的 JustVoxel Management Agent 及其明確定義的 API 執行，瀏覽器無法直接存取 root shell。

## 🔄 系統與 Minecraft 各自更新

JustVoxel 採用映像管理，建構於 Home Server Base 10／AlmaLinux 10 之上。

作業系統的更新與 Minecraft 版本、伺服器軟體、外掛、模組及容器更新分開處理。

更新 JustVoxel 時，Minecraft 版本不會在你不知情的情況下變更。調整 Minecraft 伺服器時，也無須重建主機作業系統。

[系統管理文件](../docs/SYSTEM.md)

## 📊 JustVoxel 和其他專案有何差異？

JustVoxel 與傳統 Minecraft 控制面板是不同類型的專案。它將**作業系統和 Minecraft 管理層整合成一套伺服器系統**。

| 功能 | **JustVoxel** | **MineOS SvelteKit** | **Crafty Controller** | **MineOS Node** |
| --- | :---: | :---: | :---: | :---: |
| 完整的 Minecraft 伺服器作業系統 | ⭐ | — | — | — |
| 多個 Minecraft 伺服器 | ✅ | ✅ | ✅ | ✅ |
| 支援 Java 和 Bedrock 伺服器 | ✅ | ✅ | ✅ | 🟡 支援 Java 和舊版 PocketMine |
| 引導探索外掛與模組 | ✅ Modrinth、Hangar、CurseForge | ✅ Modrinth、CurseForge | — | — |
| 主機網路、儲存與作業系統管理 | ⭐ | — | — | — |
| 整合私人存取與遊戲隧道 | ⭐ 4 種選項 | — | — | — |
| MCP + REST AI 與自動化閘道 | ⭐ | — | — | — |
| 原生 Telegram 與 Discord 管理 | ⭐ | — | — | — |
| 筆記型電腦上蓋和螢幕控制 | ✅ HWS | — | — | — |
| 以映像管理的不可變作業系統更新 | ✅ | — | — | — |
| 終端機管理介面 | ✅ mjust | ✅ mineos CLI/TUI | — | ✅ CLI |
| 安裝方式 | 完整的 bootc 伺服器系統 | Docker / Compose 應用程式 | 在 Docker、Linux 或 Windows 上執行的控制面板 | 在 Linux/BSD 上執行的 Node.js 應用程式 |

⭐ 特色功能 &nbsp;•&nbsp; ✅ 支援 &nbsp;•&nbsp; 🟡 部分支援或舊版支援 &nbsp;•&nbsp; — 查閱的專案核心文件未提及

**上次檢視比較內容：2026 年 11 月 16 日。** 這項比較無意將其他專案一概評為「不好」。如果你已有想自行管理的作業系統，MineOS 和 Crafty 都可以是很好的 Minecraft 管理工具。JustVoxel 適合希望將 Minecraft 面板**連同底層的伺服器系統**一起管理的人。

這項比較以各專案當時的文件和儲存庫為依據：

- [MineOS SvelteKit](https://github.com/freeman412/mineos-sveltekit)
- [Crafty Controller](https://craftycontrol.com/) · [原始碼](https://gitlab.com/crafty-controller/crafty-4)
- [MineOS Node](https://github.com/hexparrot/mineos-node)

如果任何比較項目已過時或有失公允，歡迎提出議題。即使更正後對 JustVoxel 較不利，也歡迎提出。

## 📚 文件

詳細的 JustVoxel 指南收錄在[文件目錄](../docs/)。

可從下列指南開始：

- [WebUI](../docs/WEBUI.md)
- [Minecraft 執行環境](../docs/MINECRAFT_RUNTIME.md)
- [伺服器軟體](../docs/server-software.md)
- [外掛與模組](../docs/plugin-workspace.md)
- [Minecraft 網路](../docs/minecraft-network.md)
- [網路與遠端存取](../docs/NETWORKING.md)
- [AI 與 API 連線](../docs/AI-API-CONNECTIONS.md)
- [Telegram](../docs/TELEGRAM.md)
- [Discord](../docs/DISCORD.md)
- [儲存](../docs/STORAGE.md)
- [還原](../docs/RESTORE.md)
- [移轉](../docs/MIGRATION.md)
- [系統管理](../docs/SYSTEM.md)
- [mjust](../docs/MJUST.md)

關於映像開發、測試和發行，請參閱[開發者指南](../README-DEV.md)。

## ⛏️ Minecraft 軟體與 EULA

JustVoxel **不隨附** Minecraft 伺服器執行檔、Mojang 伺服器軟體或預先建立的世界，也不會預先接受 Minecraft EULA。

建立伺服器時，由管理員接受 Minecraft EULA，JustVoxel 則會準備所選的受支援 Minecraft 伺服器執行環境。

Minecraft、Mojang 軟體、Paper、Purpur、Spigot、Pufferfish、Leaf、Fabric、NeoForge、Forge、Quilt、外掛、模組及其他第三方元件，均保有各自的授權條款和散布條件。

## 📄 授權

JustVoxel 採用 **Apache-2.0** 授權。請參閱 [LICENSE](../LICENSE)。

第三方聲明收錄於 [THIRD_PARTY_NOTICES.md](../THIRD_PARTY_NOTICES.md)。
