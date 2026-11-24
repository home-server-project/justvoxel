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

> **翻訳について：** この JustVoxel の翻訳は、正式な英語原文をもとに AI の支援を受けて作成しました。不正確、わかりにくい、または不自然な表現を見つけた場合は、GitHub で[問題を報告](https://github.com/home-server-project/justvoxel/issues)するか、[修正を提案するプルリクエスト](https://github.com/home-server-project/justvoxel/pulls)を送ってください。

<p align="center">
  <strong>手持ちの機器、レンタル専用サーバー、ホスティング先の VM で、自分の Minecraft サーバーを運用。管理はひとつのコントロールセンターから。</strong>
</p>

<p align="center">
  OS の管理に手をかけずに自分で Minecraft サーバーを運営したい人のための、セルフホスト型サーバーアプライアンスです。
</p>

<p align="center">
  <a href="#-justvoxel-をインストール">インストール</a> •
  <a href="#-minecraft-サーバー運用をもっと手軽に">機能</a> •
  <a href="../docs/INSTALLER.md">ドキュメント</a>
</p>

<p align="center">
  <img src="../docs/images/justvoxel-overview-1.webp" width="100%" alt="JustVoxel コントロールセンターの概要"><img src="../docs/images/justvoxel-overview-2.webp" width="100%" alt=""><img src="../docs/images/justvoxel-overview-3.webp" width="100%" alt="">
</p>

JustVoxel は、Minecraft サーバーのセルフホスティング専用アプライアンスです。対応する PC、ノート PC、物理サーバー、仮想マシン、または独自の OS イメージを使えるホスティングサービスにインストールすれば、ブラウザーから Minecraft を管理できます。

複数のスクリプトを保守したり、Minecraft 用の管理パネルを別途インストールしたりする必要はありません。

使わなくなったノート PC やデスクトップ PC はありませんか？ **JustVoxel HWS なら、まだ使える機器を Minecraft 専用アプライアンスとして再活用できます。** 対応するノート PC では、ふたを閉じてもスリープせずにサーバーを動かし続けるよう設定できます。ローカルコンソールの画面をすぐに、または少し時間をおいて消灯することもできます。

> 🚧 **プレリリース** — JustVoxel は初回の一般公開に向けて準備中です。最終リリースに向けた検証を続けながら、現在のインストール ISO をテストや先行利用向けに提供しています。

## ✨ Minecraft サーバー運用をもっと手軽に

JustVoxel は、Minecraft とその基盤となる OS の管理をひとつにまとめます。

- 🎮 **複数の Minecraft サーバー** — 1 台のアプライアンスで、異なるワールドや種類のサーバーを個別に運用できます。
- 🖥️ **Web コントロールセンター** — PC、タブレット、スマートフォンからサーバーを管理できます。
- 🧩 **プラグインと MOD** — 対応する配布サービスから、互換性のあるコンテンツを探して管理できます。
- 💾 **バックアップと復元** — スケジュール設定による自動バックアップと手動バックアップで、サーバーごとのデータを保護できます。
- 🌐 **リモートアクセス** — プライベートアクセスやゲーム用トンネルを利用できる、4 つの連携サービスから選べます。
- 🤖 **AI と自動化** — MCP と REST を通じて、対応する AI アシスタントやアプリケーションを接続できます。
- 💬 **Telegram と Discord** — 必要に応じて、ボットからの操作、通知、コミュニティ向けの機能を利用できます。
- 🛡️ **システム管理** — Linux コマンドを覚えなくても、ネットワーク、ストレージ、更新、稼働状態、ログ、対応する電源操作を管理できます。

JustVoxel は、家族や友人、小規模な Minecraft コミュニティのほか、従来の Minecraft ホスティングの管理パネルを借りる代わりに、自分のサーバーを持ちたい人に向けて設計されています。

## 🖥️ ひとつのアプライアンス、2 つのエディション

### JustVoxel VM

次のような仮想マシンやホスティング環境向けです。

- Proxmox
- KVM / libvirt
- VMware
- Hyper-V
- VirtualBox
- 対応するカスタム OS イメージを使えるホスティング VM や VPS

VM エディションには Minecraft アプライアンスの全機能が含まれ、仮想マシン内では不要な物理機器向けパッケージが省かれています。

### JustVoxel HWS

次のような物理機器に直接インストールするためのエディションです。

- 使わなくなったノート PC
- 再利用するデスクトップ PC
- ミニ PC
- 自宅サーバー
- 独自のインストールメディアを使用できるレンタルのベアメタルサーバー

HWS には、UPS と NUT の連携、SMART・NVMe ツール、センサー、ファームウェア・UEFI の操作、デバイス用ファームウェア、ハードウェア診断、ノート PC の画面やふたを閉じたときの動作設定など、物理機器向けの機能が追加されています。

両エディションで、JustVoxel コントロールセンターと Management API、Minecraft の操作手順、ストレージの仕組み、バックアップ、ネットワーク、アップデートの手順は共通です。

[VM と HWS で利用できる機能の詳細](../docs/VARIANTS.md)

## 🚀 JustVoxel をインストール

ひとつのインストール ISO が JustVoxel の両エディションに対応しています。

[⬇️ JustVoxel x86_64 ISO をダウンロード](https://sourceforge.net/projects/justvoxel/files/JustVoxel-x86_64.iso/download)

インストール先に合ったエディションを選びます。

- **JustVoxel VM** — 仮想マシンと対応する VM ホスティング環境
- **JustVoxel HWS** — PC、ノート PC、ミニ PC、物理機器を使った自宅サーバー、対応するベアメタルホスティング

インストール、ディスクの準備、初回ログインについては、[インストールガイド](../docs/INSTALLER.md)を参照してください。

## 🏠 Minecraft を中心に、ほかのサービスも運用できる

JustVoxel は Minecraft サーバーアプライアンスとして開発・サポートされています。基盤は、**Podman、Quadlet、systemd、SSH、NetworkManager、標準の Linux 管理ツールを備えた通常の AlmaLinux 10 bootc サーバー**です。

Linux の管理に慣れている方は、同じマシンを汎用の自宅サーバーとしても使い、Minecraft と並行して別のコンテナを実行できます。

たとえば、次のサービスを手動でデプロイできます。

- メディア配信用の **Jellyfin** または **Plex**
- パスワード管理用の **Vaultwarden**
- 写真や動画の管理用の **Immich**
- ホームオートメーション用の **Home Assistant Container**
- ファイル管理や共同作業用の **Nextcloud**
- **Syncthing** または **Navidrome**
- セルフホスト型 AI エージェント／ゲートウェイとしての **OpenClaw**

これらのアプリケーションは **JustVoxel コントロールセンターの管理対象外です**。Podman コンテナ、Quadlet、ストレージ、ポート、アップデート、バックアップ、リバースプロキシの設定、アプリケーション固有の要件は、管理者が管理します。

JustVoxel は Minecraft アプライアンスの管理を担います。ほかの用途には、通常どおり Podman や Quadlet を使えます。

JustVoxel のメンテナンスでも、Minecraft と無関係なコンテナは管理者が運用するものとして扱います。Minecraft の **最初からやり直す** などの操作を行っても、無関係な Podman コンテナは削除されません。

必要に応じて、Minecraft を主な用途としながら、同じマシンをコンテナで他のサービスも動かす自宅サーバーとして利用できます。

## 🎮 好みの Minecraft サーバーを作成

空きメモリと利用可能なポートが十分にあれば、JustVoxel では独立した複数の Minecraft サーバーを同時に実行できます。

各サーバーは固有の ID、設定、実行環境、ポート、データ、バックアップを持ち、起動・停止などの操作や状態確認も個別に行えます。

### 現在対応しているサーバーソフトウェア

| サーバーソフトウェア | コンテンツの種類 | JustVoxel が管理する Java → Bedrock クロスプレイ |
| --- | --- | --- |
| **Paper** | プラグイン | ✅ |
| **Purpur** | プラグイン | ✅ |
| **Spigot** | プラグイン | ✅ |
| **Pufferfish** | プラグイン | ✅ |
| **Leaf** | プラグイン | ✅ |
| **Fabric** | MOD | ✅ 対象の組み合わせで互換性を確認できた場合 |
| **NeoForge** | MOD | ✅ 対象の組み合わせで互換性を確認できた場合 |
| **Forge** | MOD | Java のみ |
| **Quilt** | MOD | Java のみ |
| **Vanilla Java** | プラグイン・MOD 非対応 | Java のみ |
| **ネイティブ Bedrock** | ネイティブ Bedrock サーバー | ネイティブ Bedrock クライアント |

案内に沿って最初のサーバーを作るには **簡易セットアップ** を選びます。メモリ、ポート、ストレージ、バックアップ、バージョンの運用方針、対応するクロスプレイを細かく設定するには **詳細セットアップ** を選びます。

JustVoxel は、選択したサーバーソフトウェアごとに利用できる機能を確認します。

[サーバーソフトウェアと互換性の詳細](../docs/server-software.md)

## 🧩 プラグインと MOD

対象のサーバーを選び、**Minecraft → プラグインと MOD** を開きます。

サーバーの種類に応じて、次の配布サービスを利用できます。

- [Modrinth](https://modrinth.com/)
- [Hangar](https://hangar.papermc.io/)
- [CurseForge](https://www.curseforge.com/minecraft)（設定済みの API キーが必要）

対応するインストール処理では、互換性の確認、ダウンロードファイルの検証、サーバー停止中のバックアップ、再起動後の検証、所有関係の追跡、復旧のための保護を行います。

Paper 系サーバーではプラグインカタログを利用できます。Fabric、Forge、NeoForge、Quilt では MOD カタログを利用し、MOD が対象のローダーと Minecraft のバージョンに正確に対応しているかを確認します。Vanilla Java とネイティブ Bedrock にはプラグインや MOD のカタログは表示されません。

[プラグインと MOD の詳細](../docs/plugin-workspace.md)

## 🌐 オプションの Minecraft Network

複数のサーバーをそれぞれ独立して運用できます。その場合、プロキシは**不要**です。

対応する複数のサーバーに公開の接続先をひとつ用意したい場合は、Velocity を使ったオプション機能 **Minecraft Network** を利用できます。

コントロールセンターから、対応するバックエンドサーバーの選択、Java と Bedrock の公開ポートの設定、仮想ロビーとウェルカムメッセージの管理、ネットワークに参加するサーバーの追加・削除ができます。

Paper、Purpur、Pufferfish、Leaf はネットワークのバックエンドとして利用できます。対応する Fabric / Quilt や Forge / NeoForge のサーバーも、必要なプロキシ互換性を JustVoxel が確認できた場合は参加できます。Spigot、Vanilla、ネイティブ Bedrock は独立したサーバーとして運用します。

[Minecraft Network の詳細](../docs/minecraft-network.md)

## 🖥️ コントロールセンター

コントロールセンターは、Linux の内部構造を意識せずにサーバーを管理できるよう設計されています。

### Minecraft サーバー

複数のサーバーを作成・管理し、起動や停止、状態とプレイヤーの確認、対応する設定の変更、サーバーごとのツールの利用ができます。

### プラグインと MOD

互換性のあるコンテンツを探し、インストール済みのプラグインや MOD を確認して、対応する追加コンテンツを管理できます。変更への対応が必要になったときは、復旧に関する情報を確認できます。

### バックアップ・ストレージ

バックアップの作成、履歴の確認、サーバーデータ全体の復元、保存先の設定、サーバーのインポートとエクスポート、対応するストレージの管理ができます。

### ネットワーク

有線 LAN、Wi-Fi、**接続のトラブルシューティング**、リモートアクセスを管理できます。接続が切れる可能性のある設定変更にはロールバックによる保護があり、IP や Wi-Fi の設定を誤っても、管理画面にアクセスできないままになるのを防げます。

### システム

稼働状態、リソース、ログ、OS アップデート、ユーザー、セキュリティの確認と、対応する電源操作や HWS 専用のハードウェア操作ができます。

### AI・API 接続

対応する AI アプリケーション、MCP クライアント、REST による自動化、Telegram、Discord を接続できます。別の管理システムを用意する必要はありません。

ターミナルを使いたい場合は、アプライアンスと同じ安全基準で動作する対話型のターミナル管理ツール **mjust** も利用できます。

[WebUI ドキュメント](../docs/WEBUI.md) · [mjust ガイド](../docs/MJUST.md)

## 🌐 友人が参加しやすい接続方法を選ぶ

JustVoxel は現在、**ネットワーク → リモートアクセス** で 4 つのサービスと連携できます。

JustVoxel では、サービスに応じてアプライアンス側で状態の確認、有効化・無効化、セットアップを行えます。

| サービス | 向いている用途 | 利用できる機能 |
| --- | --- | --- |
| **[Tailscale](https://tailscale.com/)** | 友人や家族向けのプライベートサーバー | 許可されたデバイス間のプライベートメッシュネットワーク |
| **[NetBird](https://netbird.io/)** | アクセスを管理できるプライベートネットワーク | アカウントとアクセスポリシーで管理できるプライベートネットワーク |
| **[Playit.gg](https://playit.gg/)** | ルーターで通常のポート転送を設定せずに Minecraft を公開 | 公開ゲームトンネルと接続用アドレス |
| **[NordVPN Meshnet](https://meshnet.nordvpn.com/)** | NordVPN ユーザー向けのデバイス間のプライベートアクセス | JustVoxel を商用 VPN クライアントとして使わずに Meshnet で接続 |

### どれを選べばよいですか？

**🎮 友人にプライベート VPN クライアントをインストールせずに参加してもらいたい場合**  
まず **Playit.gg** を検討してください。プレイヤーは、設定したゲームトンネルの公開アドレスから接続できます。

**🔒 信頼できるデバイスだけで使うプライベートサーバーにしたい場合**  
**Tailscale** または **NetBird** を検討してください。

**🌍 すでに NordVPN Meshnet を利用している場合**  
JustVoxel で Meshnet 連携を管理できます。この連携は、アプライアンスを通常の商用 NordVPN の出口サーバーに接続する機能とは別です。

サービスのアカウント、クラウド側のポリシー、トンネルの設定は、提供元のサービス側で管理します。JustVoxel は、対応するアプライアンス側の連携を担当します。

複数の VPN やメッシュネットワークを同時に利用することもできますが、ルーティング、DNS、ファイアウォールルールが競合することがあります。

[ネットワークとリモートアクセスのガイド](../docs/NETWORKING.md)

## 🤖 AI・MCP・REST・自動化

AI アシスタントにサーバーの状態を尋ねたり、ホームオートメーションと連携したり、独自の操作フローを作ったりできます。

JustVoxel の **AI・API 接続** には、**MCP と REST のインターフェース**が組み込まれています。

セットアップ画面では、対応するアプリケーションや接続の種類として、次のような項目を選べます。

- **OpenAI · ChatGPT と Codex**
- **Claude**
- **Grok**
- **Gemini**
- **GitHub Copilot**
- **その他の AI・MCP アプリ**
- **自動化・REST API**

接続ごとに、アクセスできる範囲を設定できます。

権限は **レベル 1：読み取り専用** からサーバー管理まで設定できます。明示的に承認した場合に限り、高度なシステム操作も許可できます。接続先を指定した Minecraft サーバーだけに制限することもできます。

JustVoxel の MCP や REST インターフェースを公開するだけなら、AI モデルの API キーは不要です。クライアントが実際に利用できるかどうかは、アプリケーション、アカウントや契約プランで使える機能、ネットワークアクセスによって異なります。

AI・API ゲートウェイは初期状態では無効です。有効化してから利用できます。

[AI・API 接続の詳細](../docs/AI-API-CONNECTIONS.md)

## 💬 Telegram と Discord

JustVoxel には、WebUI と同じ管理バックエンドを使う Telegram と Discord の連携機能が組み込まれています。必要に応じて利用できます。

### Telegram

専用の Telegram ボットでは、次の操作や情報の確認ができます。

- アプライアンスと Minecraft の状態の確認
- サーバーの選択とプレイヤー情報の確認
- 承認済みの管理ユーザーによる起動、停止、再起動
- 手動バックアップ
- Minecraft とコンテナのアップデート内容の確認
- OS アップデート情報の確認
- 所定の手順によるアプライアンスの再起動
- サーバーの状態、バックアップ結果、アップデートの有無に関する任意の通知

Telegram には専用の **読み取り専用** と **サーバー管理** の権限があり、AI モデルは不要です。

[Telegram の設定と権限](../docs/TELEGRAM.md)

### Discord

Discord は、Minecraft コミュニティの窓口としても管理画面としても利用できます。

明示的に連携した Discord サーバーのコミュニティメンバーは、サーバーの基本的な状態を確認し、Minecraft への参加を申請できます。メンバーに管理権限が自動で付与されることはありません。

承認済みの JustVoxel の Discord 管理者は、サーバーの起動、停止、再起動、手動バックアップ、アップデートの確認、Minecraft Network の操作、そのほかの高度な操作など、対応するサーバー管理機能を利用できます。

Discord サーバーのメンバー資格、Minecraft のホワイトリストによるアクセス許可、JustVoxel の管理権限は、それぞれ個別に管理されます。

[Discord の設定、コミュニティのアクセス、管理](../docs/DISCORD.md)

## 💾 バックアップ、復元、ストレージ、移行

アプライアンスのイメージ以上に大切なのは Minecraft のデータです。

JustVoxel は、Minecraft データとバックアップの保存領域を変更不可の OS デプロイメントの外に置き、それぞれの保存先を個別に選べるようにしています。

次のストレージを利用できます。

- システムのファイルシステム
- 専用の内蔵ディスク
- USB ストレージ
- 既存のローカルパーティションやファイルシステム
- すでに未割り当ての領域に作成する新規パーティション
- NFS
- SMB/CIFS

サーバーごとに自動バックアップと手動バックアップを利用できます。復元すると、対象サーバーの永続データ全体が置き換わります。JustVoxel 上のサーバー ID とアプライアンスの設定は維持されます。

別のマシンへの移行や再インストール時のサーバー移行には、内容を確認しながら進められる **エクスポート・インポート・復旧** の手順も用意されています。

[ストレージ](../docs/STORAGE.md) · [復元](../docs/RESTORE.md) · [移行](../docs/MIGRATION.md)

## 🛡️ 変更を安全に適用する仕組み

JustVoxel では、ブラウザーから実行できる操作を、アプライアンスでサポートするものに限定しています。任意のシェルコマンドを実行できる仕組みではありません。

対応する操作には、アプライアンス全体で次のような安全対策を適用しています。

- プレイに影響する操作の前に、プレイヤーの接続状況を確認
- 適用前に変更内容を確認
- 復旧に備える必要がある変更では、サーバーを停止してバックアップ
- サーバーを正確に識別し、別のサーバーと誤って取り違えるのを防止
- 再起動後の状態やランタイムの動作を検証
- 操作や復旧の状態を永続的に保存
- 対応する操作のロールバック手順を用意
- 接続に影響するおそれがあるネットワーク変更には NetworkManager のチェックポイントを使用
- Minecraft と OS のアップデートを別々の手順で実施

ブラウザーから利用する WebUI は、非特権で動作します。ブラウザーには root シェルを公開せず、特権が必要な変更は、ローカルの JustVoxel Management Agent が定義済みの API を通じて処理します。

## 🔄 システムと Minecraft を個別にアップデート

JustVoxel は、Home Server Base 10 / AlmaLinux 10 を基盤とし、OS をイメージ単位で管理するアプライアンスです。

OS のアップデートは、Minecraft のバージョンや、サーバーソフトウェア、プラグイン、MOD、コンテナの更新とは別に行います。

アプライアンスをアップデートしても Minecraft のバージョンが勝手に変わることはありません。Minecraft サーバーを変更するために、ホスト OS を再ビルドする必要もありません。

[システム管理のドキュメント](../docs/SYSTEM.md)

## 📊 JustVoxel とほかのプロジェクトの比較

JustVoxel は、**OS と Minecraft の管理機能をひとつのアプライアンスにまとめた**プロジェクトです。この点が、従来の Minecraft 管理パネルと異なります。

| 機能 | **JustVoxel** | **MineOS SvelteKit** | **Crafty Controller** | **MineOS Node** |
| --- | :---: | :---: | :---: | :---: |
| Minecraft 専用アプライアンスとして完結する OS | ⭐ | — | — | — |
| 複数の Minecraft サーバー | ✅ | ✅ | ✅ | ✅ |
| Java と Bedrock のサーバーに対応 | ✅ | ✅ | ✅ | 🟡 Java と旧来の PocketMine に対応 |
| 案内に沿ってプラグインや MOD を探す機能 | ✅ Modrinth、Hangar、CurseForge | ✅ Modrinth、CurseForge | — | — |
| ホストのネットワーク、ストレージ、OS の管理 | ⭐ | — | — | — |
| プライベートアクセス・ゲームトンネルの連携 | ⭐ 4 種類 | — | — | — |
| MCP・REST による AI・自動化ゲートウェイ | ⭐ | — | — | — |
| 標準の Telegram・Discord 管理機能 | ⭐ | — | — | — |
| ノート PC のふたと画面の制御 | ✅ HWS | — | — | — |
| イメージ単位で管理する変更不可の OS のアップデート | ✅ | — | — | — |
| ターミナル管理インターフェース | ✅ mjust | ✅ mineos CLI/TUI | — | ✅ CLI |
| インストール形態 | 完結した bootc アプライアンス | Docker / Compose アプリケーション | Docker、Linux、Windows で動作する管理パネル | Linux / BSD で動作する Node.js アプリケーション |

⭐ 特長 &nbsp;•&nbsp; ✅ 対応 &nbsp;•&nbsp; 🟡 一部対応・旧来の機能 &nbsp;•&nbsp; — 確認したプロジェクトの主要ドキュメントに記載なし

**比較の最終確認日：2026 年 11 月 16 日。** この比較は、各プロジェクトの用途の違いを示すためのものです。自分で管理したい OS がすでにある場合、Minecraft 管理アプリケーションの MineOS や Crafty は有力な選択肢です。JustVoxel は、Minecraft の管理パネルと**その基盤となるサーバーアプライアンス**をひとつの製品として管理したい人向けです。

比較には、各プロジェクトの現行ドキュメントとリポジトリを使用しています。

- [MineOS SvelteKit](https://github.com/freeman412/mineos-sveltekit)
- [Crafty Controller](https://craftycontrol.com/) · [ソースコード](https://gitlab.com/crafty-controller/crafty-4)
- [MineOS Node](https://github.com/hexparrot/mineos-node)

比較に古い情報や公平でない記述があれば、Issue を作成してください。JustVoxel に不利になる訂正も歓迎します。

## 📚 ドキュメント

アプライアンスの詳しいガイドは[ドキュメントディレクトリ](../docs/)にあります。

主なガイドは次のとおりです。

- [WebUI](../docs/WEBUI.md)
- [Minecraft ランタイム](../docs/MINECRAFT_RUNTIME.md)
- [サーバーソフトウェア](../docs/server-software.md)
- [プラグインと MOD](../docs/plugin-workspace.md)
- [Minecraft Network](../docs/minecraft-network.md)
- [ネットワークとリモートアクセス](../docs/NETWORKING.md)
- [AI・API 接続](../docs/AI-API-CONNECTIONS.md)
- [Telegram](../docs/TELEGRAM.md)
- [Discord](../docs/DISCORD.md)
- [ストレージ](../docs/STORAGE.md)
- [復元](../docs/RESTORE.md)
- [移行](../docs/MIGRATION.md)
- [システム管理](../docs/SYSTEM.md)
- [mjust](../docs/MJUST.md)

イメージの開発、テスト、リリースについては、[開発者ガイド](../README-DEV.md)を参照してください。

## ⛏️ Minecraft ソフトウェアと EULA

JustVoxel には、Minecraft サーバーのバイナリ、Mojang のサーバーソフトウェア、作成済みのワールドは**含まれていません**。Minecraft EULA への同意も事前には行われません。

サーバーを作成する際に、管理者が Minecraft EULA に同意すると、JustVoxel が選択された対応 Minecraft サーバーの実行環境を用意します。

Minecraft、Mojang のソフトウェア、Paper、Purpur、Spigot、Pufferfish、Leaf、Fabric、NeoForge、Forge、Quilt、プラグイン、MOD、そのほかのサードパーティコンポーネントには、それぞれ固有のライセンスと配布条件が適用されます。

## 📄 ライセンス

JustVoxel は **Apache-2.0** ライセンスで提供されます。[LICENSE](../LICENSE) を参照してください。

サードパーティに関する通知は [THIRD_PARTY_NOTICES.md](../THIRD_PARTY_NOTICES.md) にまとめています。
