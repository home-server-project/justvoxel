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

> **Catatan penerjemahan:** Terjemahan JustVoxel ini dibuat dari teks asli berbahasa Inggris dengan bantuan AI. Jika Anda menemukan ungkapan yang tidak akurat, kurang jelas, atau terdengar tidak alami, silakan usulkan perbaikan dengan membuat [laporan masalah di GitHub](https://github.com/home-server-project/justvoxel/issues) atau mengirim [permintaan perubahan (pull request)](https://github.com/home-server-project/justvoxel/pulls).

<p align="center">
  <strong>Kelola server Minecraft Anda dari satu Pusat Kontrol, baik di perangkat sendiri, server khusus sewaan, maupun VM dari penyedia hosting.</strong>
</p>

<p align="center">
  Sistem siap pakai untuk menjalankan server Minecraft yang Anda host sendiri, tanpa perlu mengurus sistem operasinya.
</p>

<p align="center">
  <a href="#-instal-justvoxel">Instalasi</a> •
  <a href="#-hosting-minecraft-jadi-lebih-mudah">Fitur</a> •
  <a href="../docs/INSTALLER.md">Dokumentasi</a>
</p>

<p align="center">
  <img src="../docs/images/justvoxel-overview-1.webp" width="100%" alt="Tampilan Pusat Kontrol JustVoxel"><img src="../docs/images/justvoxel-overview-2.webp" width="100%" alt=""><img src="../docs/images/justvoxel-overview-3.webp" width="100%" alt="">
</p>

JustVoxel adalah sistem siap pakai khusus untuk menjalankan server Minecraft yang Anda host sendiri. Instal di PC, laptop, server fisik, mesin virtual, atau layanan hosting yang kompatibel dan mengizinkan penggunaan citra OS sendiri, lalu kelola Minecraft dari peramban.

Anda tidak perlu memelihara kumpulan skrip atau memasang panel pengelolaan Minecraft terpisah.

Punya laptop atau PC lama? **JustVoxel HWS bisa mengubahnya menjadi perangkat khusus untuk server Minecraft agar perangkat yang masih berguna tidak teronggok begitu saja.** Pada laptop yang didukung, JustVoxel bisa mencegah laptop masuk ke mode tidur saat penutupnya ditutup, sehingga server tetap berjalan. Kontrol layar konsol lokal juga memungkinkan Anda mematikan layar seketika atau setelah jeda singkat.

> 🚧 **Prarilis:** JustVoxel sedang disiapkan untuk rilis publik pertamanya. ISO instalasi saat ini tersedia untuk pengujian dan penggunaan awal selagi validasi akhir rilis berlangsung.

## ✨ Hosting Minecraft jadi lebih mudah

JustVoxel memadukan pengelolaan Minecraft dengan sistem operasi yang menjalankannya.

- 🎮 **Beberapa server Minecraft** — jalankan berbagai dunia dan jenis server secara mandiri pada satu perangkat.
- 🖥️ **Pusat Kontrol berbasis web** — kelola server dari komputer, tablet, atau ponsel.
- 🧩 **Plugin & Mod** — temukan dan kelola konten yang kompatibel dari penyedia yang didukung.
- 💾 **Pencadangan & Pemulihan** — lindungi setiap server dengan pencadangan terjadwal dan manual.
- 🌐 **Akses Jarak Jauh** — pilih dari empat opsi terintegrasi untuk akses privat dan terowongan game.
- 🤖 **AI & Otomatisasi** — hubungkan asisten AI dan aplikasi yang kompatibel melalui MCP dan REST.
- 💬 **Telegram & Discord** — gunakan bot, notifikasi, dan alur kerja komunitas sesuai kebutuhan.
- 🛡️ **Pengelolaan Sistem** — kelola jaringan, penyimpanan, pembaruan, kesehatan sistem, log, dan kontrol daya yang didukung tanpa perlu mempelajari perintah Linux.

JustVoxel dirancang untuk keluarga, teman, komunitas kecil Minecraft, dan siapa pun yang ingin memiliki server sendiri tanpa menyewa panel hosting Minecraft tradisional.

## 🖥️ Satu sistem, dua edisi

### JustVoxel VM

Untuk mesin virtual dan lingkungan hosting seperti berikut.

- Proxmox
- KVM / libvirt
- VMware
- Hyper-V
- VirtualBox
- VM dari penyedia hosting atau layanan VPS yang mendukung citra OS kustom yang kompatibel

Edisi VM menyediakan sistem Minecraft lengkap tanpa paket untuk perangkat keras fisik yang tidak diperlukan di mesin virtual.

### JustVoxel HWS

Edisi ini dapat diinstal langsung pada perangkat keras fisik berikut.

- laptop lama
- PC desktop yang dialihfungsikan
- PC mini
- server rumahan
- server bare metal sewaan yang mengizinkan media instalasi kustom

HWS menambahkan dukungan untuk perangkat fisik, seperti integrasi UPS/NUT, alat SMART dan NVMe, sensor, kontrol firmware/UEFI, firmware perangkat, diagnostik perangkat keras, serta kontrol layar dan penutup laptop.

Kedua edisi menggunakan Pusat Kontrol JustVoxel, Management API, alur kerja Minecraft, model penyimpanan, sistem pencadangan, pengelolaan jaringan, dan proses pembaruan yang sama.

[Pelajari cakupan fitur edisi VM dan HWS](../docs/VARIANTS.md)

## 🚀 Instal JustVoxel

Satu ISO penginstal mendukung kedua edisi JustVoxel.

[⬇️ Unduh ISO JustVoxel x86_64](https://sourceforge.net/projects/justvoxel/files/JustVoxel-x86_64.iso/download)

**Buat drive USB yang dapat digunakan untuk boot** dengan menulis berkas ISO ke drive USB menggunakan [balenaEtcher](https://etcher.balena.io/), [Rufus](https://rufus.ie/en/), [Ventoy](https://www.ventoy.net/en/index.html), [Fedora Media Writer](https://fedoraproject.org/workstation/download/), atau alat lain yang dapat menulis image secara langsung. Di Linux dan sistem lain yang mirip Unix, metode tradisional dengan `dd` juga dapat digunakan. Menulis image langsung ke drive USB akan menghapus seluruh isinya.

Pilih edisi sesuai dengan tempat Anda menginstalnya.

- **JustVoxel VM** untuk mesin virtual dan lingkungan hosting VM yang kompatibel
- **JustVoxel HWS** untuk PC, laptop, PC mini, server rumahan fisik, dan hosting bare metal yang kompatibel

Lihat [panduan instalasi](../docs/INSTALLER.md) untuk langkah instalasi, persiapan disk, dan login pertama.

## 🏠 Mengutamakan Minecraft, tetap bisa menjalankan layanan lain

JustVoxel dibuat dan didukung terutama sebagai sistem siap pakai untuk server Minecraft. Sistem dasarnya tetap berupa **server AlmaLinux 10 bootc standar dengan Podman, Quadlet, systemd, SSH, NetworkManager, dan alat administrasi Linux standar**.

Pengguna tingkat lanjut dapat memanfaatkan mesin yang sama sebagai server rumahan serbaguna dan menjalankan kontainer tambahan bersama Minecraft.

Anda dapat memasang sendiri layanan seperti berikut.

- **Jellyfin** atau **Plex** untuk media
- **Vaultwarden** untuk mengelola kata sandi
- **Immich** untuk mengelola foto dan video
- **Home Assistant Container** untuk otomatisasi rumah
- **Nextcloud** untuk pengelolaan berkas dan kolaborasi
- **Syncthing** atau **Navidrome**
- **OpenClaw** sebagai agen atau gateway AI yang di-host sendiri

Aplikasi ini **tidak dikelola oleh Pusat Kontrol JustVoxel**. Administrator tetap mengelola kontainer Podman, konfigurasi Quadlet, penyimpanan, port, pembaruan, cadangan, konfigurasi proksi balik, dan kebutuhan khusus tiap aplikasi.

JustVoxel menangani pengelolaan sistem Minecraft. Untuk layanan lain, pengguna tingkat lanjut tetap dapat menggunakan alur kerja Podman dan Quadlet yang biasa mereka gunakan.

Saat melakukan pemeliharaan, JustVoxel memperlakukan kontainer lain sebagai beban kerja milik administrator. Operasi Minecraft seperti **Mulai dari awal** tidak menghapus kontainer Podman di luar Minecraft.

Dengan demikian, mesin JustVoxel tetap dapat berfungsi terutama sebagai server Minecraft sekaligus menjadi server rumahan berbasis kontainer jika diperlukan.

## 🎮 Buat server Minecraft sesuai keinginan Anda

JustVoxel dapat menjalankan beberapa server Minecraft secara mandiri dan bersamaan jika mesin memiliki memori yang cukup dan port yang tersedia.

Setiap server memiliki identitas, konfigurasi, runtime, port, data, cadangan, kontrol siklus hidup, dan status tersendiri.

### Perangkat lunak server yang tersedia

| Perangkat lunak server | Jenis konten | Dukungan terkelola untuk permainan lintas platform Java → Bedrock |
| --- | --- | --- |
| **Paper** | Plugin | ✅ |
| **Purpur** | Plugin | ✅ |
| **Spigot** | Plugin | ✅ |
| **Pufferfish** | Plugin | ✅ |
| **Leaf** | Plugin | ✅ |
| **Fabric** | Mod | ✅ Jika kompatibilitas kombinasi yang digunakan telah diverifikasi |
| **NeoForge** | Mod | ✅ Jika kompatibilitas kombinasi yang digunakan telah diverifikasi |
| **Forge** | Mod | Hanya Java |
| **Quilt** | Mod | Hanya Java |
| **Vanilla Java** | Tanpa plugin atau mod | Hanya Java |
| **Bedrock asli** | Server Bedrock asli | Klien Bedrock asli |

Pilih **Penyiapan sederhana** untuk membuat server pertama dengan panduan, atau **Penyiapan lanjutan** jika Anda ingin mengatur lebih banyak hal, seperti memori, port, penyimpanan, pencadangan, kebijakan versi, dan permainan lintas platform yang kompatibel.

JustVoxel memeriksa fitur yang didukung oleh perangkat lunak server yang Anda pilih.

[Lihat detail perangkat lunak server dan kompatibilitasnya](../docs/server-software.md)

## 🧩 Plugin & Mod

Buka **Minecraft → Plugin & Mod** untuk server yang dipilih.

Bergantung pada jenis servernya, JustVoxel dapat menggunakan penyedia berikut.

- [Modrinth](https://modrinth.com/)
- [Hangar](https://hangar.papermc.io/)
- [CurseForge](https://www.curseforge.com/minecraft) dengan kunci API yang telah Anda konfigurasikan

Untuk instalasi yang didukung, JustVoxel memeriksa kompatibilitas, memverifikasi unduhan, membuat cadangan saat server berhenti, memverifikasi mulai ulang, melacak kepemilikan, dan menyediakan mekanisme pemulihan.

Server keluarga Paper dapat menggunakan katalog plugin. Fabric, Forge, NeoForge, dan Quilt menggunakan katalog mod dengan pemeriksaan kecocokan loader dan versi Minecraft yang tepat. Vanilla Java dan Bedrock asli memang tidak menyediakan katalog plugin atau mod.

[Pelajari Plugin & Mod](../docs/plugin-workspace.md)

## 🌐 Jaringan Minecraft opsional

Beberapa server mandiri dapat berjalan seperti biasa **tanpa proksi**.

Jika Anda ingin menyediakan satu titik masuk publik untuk beberapa server yang kompatibel, JustVoxel juga menyediakan **Jaringan Minecraft** opsional berbasis Velocity.

Dari Pusat Kontrol, Anda dapat memilih server backend yang didukung, mengatur port publik Java dan Bedrock, mengelola lobi virtual dan pesan sambutan, serta menentukan server mana yang tergabung dalam jaringan.

Paper, Purpur, Pufferfish, dan Leaf didukung sebagai backend jaringan. Server Fabric/Quilt dan Forge/NeoForge yang kompatibel juga dapat bergabung setelah JustVoxel memverifikasi kompatibilitas proksi yang diperlukan. Spigot, Vanilla, dan Bedrock asli tetap berjalan secara mandiri.

[Pelajari Jaringan Minecraft](../docs/minecraft-network.md)

## 🖥️ Mengenal Pusat Kontrol

JustVoxel menata antarmukanya berdasarkan tugas pengelolaan server agar Anda tidak perlu memahami cara kerja internal Linux.

### Server Minecraft

Buat dan kelola beberapa server, jalankan atau hentikan server, periksa status dan pemain, sesuaikan pengaturan yang didukung, dan gunakan alat khusus untuk setiap server.

### Plugin & Mod

Temukan konten yang kompatibel, tinjau plugin dan mod yang terinstal, kelola konten tambahan yang didukung, dan lihat informasi pemulihan jika ada perubahan yang perlu ditangani.

### Cadangan & Penyimpanan

Buat cadangan, tinjau riwayat pencadangan, pulihkan seluruh data server, atur tujuan cadangan, impor atau ekspor server, dan kelola penyimpanan yang didukung.

### Jaringan

Kelola Ethernet, Wi-Fi, **pemecahan masalah koneksi**, dan Akses Jarak Jauh. Perubahan jaringan yang berpotensi memutus koneksi dilindungi dengan mekanisme untuk mengembalikan konfigurasi sebelumnya, sehingga perubahan IP atau Wi-Fi yang keliru tidak membuat Anda kehilangan akses secara permanen.

### Sistem

Periksa kondisi sistem, sumber daya, log, pembaruan sistem operasi, pengguna, keamanan, tindakan daya yang didukung, dan kontrol perangkat keras khusus HWS.

### Koneksi AI & API

Hubungkan aplikasi AI yang didukung, klien MCP, otomatisasi REST, Telegram, dan Discord tanpa perlu menyiapkan sistem pengelolaan tambahan.

Lebih suka memakai terminal? JustVoxel juga menyediakan **mjust**, antarmuka administrasi terminal interaktif yang menggunakan aturan pengamanan yang sama.

[Dokumentasi WebUI](../docs/WEBUI.md) · [Panduan mjust](../docs/MJUST.md)

## 🌐 Pilih cara teman Anda terhubung

JustVoxel saat ini mengintegrasikan empat opsi di **Jaringan → Akses Jarak Jauh**.

Integrasi ini mencakup paket yang disertakan dalam citra sistem serta tampilan status, kontrol aktivasi dan penonaktifan, dan alur penyiapan yang didukung untuk masing-masing penyedia jika tersedia.

| Opsi | Cocok untuk | Fitur yang tersedia |
| --- | --- | --- |
| **[Tailscale](https://tailscale.com/)** | Server privat untuk teman dan keluarga | Jaringan mesh privat antarperangkat yang disetujui |
| **[NetBird](https://netbird.io/)** | Jaringan privat dengan akses terkelola | Jaringan privat dengan pengelolaan akun dan kebijakan akses |
| **[Playit.gg](https://playit.gg/)** | Akses publik ke Minecraft tanpa penerusan port biasa di router | Terowongan game publik dan alamat koneksi |
| **[NordVPN Meshnet](https://meshnet.nordvpn.com/)** | Akses privat antarperangkat untuk pengguna NordVPN | Koneksi Meshnet tanpa menjadikan JustVoxel klien VPN komersial |

### Opsi mana yang sebaiknya saya pilih?

**🎮 Ingin teman bisa bergabung tanpa menginstal klien VPN privat?**  
Coba lihat **Playit.gg** terlebih dahulu. Pemain dapat menggunakan alamat publik dari terowongan game yang Anda konfigurasikan.

**🔒 Ingin server privat untuk perangkat tepercaya?**  
Pertimbangkan **Tailscale** atau **NetBird**.

**🌍 Sudah menggunakan NordVPN Meshnet?**  
JustVoxel dapat mengelola integrasi Meshnet. Integrasi ini terpisah dari koneksi perangkat ke server keluar VPN komersial NordVPN.

Akun, kebijakan di sisi cloud, dan konfigurasi terowongan tetap dikelola di layanan penyedia. JustVoxel menangani integrasi yang didukung di sisi perangkat.

Beberapa layanan VPN atau mesh dapat digunakan secara bersamaan, tetapi aturan perutean, DNS, atau firewall dapat bertabrakan.

[Baca panduan jaringan dan Akses Jarak Jauh](../docs/NETWORKING.md)

## 🤖 AI, MCP, REST & Otomatisasi

Ingin menanyakan status server kepada asisten AI, menghubungkan sistem otomatisasi rumah, atau membuat alur pengelolaan sendiri?

JustVoxel menyediakan **antarmuka MCP dan REST** bawaan melalui **Koneksi AI & API**.

Saat menyiapkan koneksi, Anda dapat memilih aplikasi dan jenis koneksi yang kompatibel, seperti berikut.

- **OpenAI — ChatGPT & Codex**
- **Claude**
- **Grok**
- **Gemini**
- **GitHub Copilot**
- **Aplikasi AI / MCP lainnya**
- **Otomatisasi / REST API**

Anda menentukan apa yang dapat diakses oleh setiap koneksi.

Izin dimulai dari **Tingkat 1 — Baca Saja**, mencakup pengelolaan server, dan bisa mencapai kontrol sistem lanjutan hanya dengan persetujuan eksplisit. Koneksi juga dapat dibatasi ke server Minecraft tertentu.

JustVoxel dapat menyediakan antarmuka MCP atau REST tanpa kunci API model AI. Dukungan untuk klien tertentu bergantung pada aplikasi, fitur akun atau langganan, dan akses jaringan.

Gateway AI/API dinonaktifkan secara bawaan sampai Anda mengaktifkannya.

[Pelajari Koneksi AI & API](../docs/AI-API-CONNECTIONS.md)

## 💬 Telegram & Discord

JustVoxel menyediakan integrasi bawaan opsional untuk Telegram dan Discord. Operasi dari kedua integrasi diproses oleh backend pengelolaan yang juga digunakan WebUI.

### Telegram

Bot Telegram khusus dapat menyediakan fungsi berikut.

- melihat status perangkat dan Minecraft
- memilih server dan melihat informasi pemain
- menjalankan, menghentikan, dan memulai ulang server bagi pengguna yang diberi izin pengelolaan
- membuat cadangan manual
- meninjau pembaruan Minecraft dan kontainer
- melihat informasi pembaruan OS
- menjalankan alur mulai ulang perangkat secara terkendali
- menerima notifikasi opsional tentang status server, hasil pencadangan, dan ketersediaan pembaruan

Telegram memiliki izin **Baca Saja** dan **Pengelolaan Server** tersendiri. Integrasi ini dapat berjalan tanpa model AI.

[Penyiapan dan izin Telegram](../docs/TELEGRAM.md)

### Discord

Discord dapat digunakan sebagai sarana untuk komunitas Minecraft sekaligus untuk administrasi.

Anggota komunitas di server Discord yang ditautkan secara eksplisit dapat melihat status dasar server dan meminta akses ke Minecraft tanpa otomatis mendapat hak administrasi.

Administrator JustVoxel di Discord yang telah disetujui dapat menjalankan tindakan yang didukung, seperti menjalankan, menghentikan, dan memulai ulang server, membuat cadangan manual, meninjau pembaruan, mengelola Jaringan Minecraft, serta melakukan tindakan lanjutan lainnya.

Keanggotaan di Discord, akses melalui daftar izin Minecraft, dan hak administrasi JustVoxel merupakan izin yang terpisah.

[Penyiapan Discord, akses komunitas, dan administrasi](../docs/DISCORD.md)

## 💾 Pencadangan, Pemulihan, Penyimpanan & Migrasi

Data Minecraft lebih penting daripada citra sistem perangkat.

JustVoxel menyimpan data Minecraft dan cadangan di luar instalasi sistem operasi yang tidak dapat diubah. Lokasi penyimpanan data dan cadangan dapat dipilih secara terpisah.

Pilihan penyimpanan yang didukung meliputi hal berikut.

- sistem berkas tempat sistem operasi terpasang
- disk internal khusus
- penyimpanan USB
- partisi atau sistem berkas lokal yang sudah ada
- partisi baru yang dibuat dari ruang yang belum dialokasikan
- NFS
- SMB/CIFS

Setiap server mendukung pencadangan otomatis dan manual. Pemulihan menggantikan seluruh data persisten server yang dipilih sambil mempertahankan identitas JustVoxel dan konfigurasi server tersebut di perangkat.

Untuk memindahkan server ke mesin lain atau saat instalasi ulang, JustVoxel juga menyediakan alur **Ekspor / Impor / Pemulihan** dengan tahap peninjauan.

[Penyimpanan](../docs/STORAGE.md) · [Pemulihan](../docs/RESTORE.md) · [Migrasi](../docs/MIGRATION.md)

## 🛡️ Dirancang agar perubahan lebih aman

Antarmuka JustVoxel menjalankan operasi pengelolaan yang telah ditentukan, dengan pengamanan di tingkat perangkat.

Operasi yang didukung mengikuti aturan pengamanan seperti berikut.

- pemeriksaan keberadaan pemain sebelum tindakan yang dapat mengganggu permainan
- peninjauan perubahan sebelum memilih Terapkan
- cadangan saat server berhenti jika perubahan memerlukan perlindungan pemulihan
- pencocokan identitas server secara tepat agar satu server tidak tertukar dengan server lain
- verifikasi mulai ulang dan kondisi runtime
- penyimpanan status operasi dan pemulihan secara persisten
- mekanisme untuk mengembalikan kondisi sebelumnya pada operasi yang didukung
- titik pemulihan NetworkManager untuk perubahan jaringan berisiko
- alur pembaruan Minecraft dan sistem operasi yang terpisah

WebUI yang diakses melalui peramban berjalan tanpa hak istimewa. Perubahan yang memerlukan hak istimewa ditangani oleh JustVoxel Management Agent lokal melalui operasi API yang telah ditentukan, sehingga shell root tidak terekspos ke peramban.

## 🔄 Pembaruan sistem terpisah dari pembaruan Minecraft

JustVoxel dibangun di atas Home Server Base 10 / AlmaLinux 10 dan dikelola melalui citra sistem.

Pembaruan sistem operasi ditangani secara terpisah dari perubahan versi Minecraft, perangkat lunak server, plugin, mod, dan pembaruan kontainer.

Dengan begitu, pembaruan perangkat tidak diam-diam mengubah versi Minecraft. Perubahan pada server Minecraft juga tidak mengharuskan Anda membangun ulang sistem operasi host.

[Dokumentasi pengelolaan sistem](../docs/SYSTEM.md)

## 📊 Bagaimana JustVoxel dibandingkan dengan proyek lain?

JustVoxel memiliki cakupan berbeda dari panel kontrol Minecraft tradisional karena **sistem operasi dan pengelolaan Minecraft terintegrasi dalam satu produk**.

| Kemampuan | **JustVoxel** | **MineOS SvelteKit** | **Crafty Controller** | **MineOS Node** |
| --- | :---: | :---: | :---: | :---: |
| OS lengkap untuk perangkat server Minecraft | ⭐ | — | — | — |
| Beberapa server Minecraft | ✅ | ✅ | ✅ | ✅ |
| Dukungan server Java + Bedrock | ✅ | ✅ | ✅ | 🟡 Dukungan Java + PocketMine versi lama |
| Pencarian plugin dan mod dengan panduan | ✅ Modrinth, Hangar, CurseForge | ✅ Modrinth, CurseForge | — | — |
| Pengelolaan jaringan, penyimpanan, dan OS host | ⭐ | — | — | — |
| Akses privat / terowongan game terintegrasi | ⭐ 4 opsi | — | — | — |
| Gateway MCP + REST untuk AI dan otomatisasi | ⭐ | — | — | — |
| Pengelolaan melalui integrasi bawaan Telegram + Discord | ⭐ | — | — | — |
| Kontrol penutup dan layar laptop | ✅ HWS | — | — | — |
| Pembaruan OS imutabel melalui citra sistem | ✅ | — | — | — |
| Antarmuka pengelolaan terminal | ✅ mjust | ✅ mineos CLI/TUI | — | ✅ CLI |
| Model instalasi | Sistem bootc siap pakai | Aplikasi Docker / Compose | Panel kontrol di Docker, Linux, atau Windows | Aplikasi Node.js di Linux/BSD |

⭐ unggulan &nbsp;•&nbsp; ✅ didukung &nbsp;•&nbsp; 🟡 sebagian/versi lama &nbsp;•&nbsp; — tidak tercakup dalam dokumentasi inti proyek yang ditinjau

**Perbandingan terakhir ditinjau pada 16 November 2026.** Setiap proyek memiliki kegunaannya masing-masing. MineOS dan Crafty dapat menjadi pilihan yang sangat baik untuk mengelola Minecraft jika Anda sudah memiliki sistem operasi yang ingin Anda kelola sendiri. JustVoxel ditujukan bagi pengguna yang ingin panel Minecraft **dan sistem server yang menjalankannya** dikelola sebagai satu produk.

Perbandingan ini mengacu pada dokumentasi dan repositori terbaru dari proyek-proyek berikut.

- [MineOS SvelteKit](https://github.com/freeman412/mineos-sveltekit)
- [Crafty Controller](https://craftycontrol.com/) · [kode sumber](https://gitlab.com/crafty-controller/crafty-4)
- [MineOS Node](https://github.com/hexparrot/mineos-node)

Jika ada bagian perbandingan yang sudah tidak berlaku atau kurang adil, silakan buka isu. Koreksi tetap kami terima meskipun membuat JustVoxel tampak kurang unggul.

## 📚 Dokumentasi

Panduan lengkap JustVoxel tersedia di [direktori dokumentasi](../docs/).

Anda bisa mulai dari panduan berikut.

- [WebUI](../docs/WEBUI.md)
- [Runtime Minecraft](../docs/MINECRAFT_RUNTIME.md)
- [Perangkat lunak server](../docs/server-software.md)
- [Plugin & Mod](../docs/plugin-workspace.md)
- [Jaringan Minecraft](../docs/minecraft-network.md)
- [Jaringan & Akses Jarak Jauh](../docs/NETWORKING.md)
- [Koneksi AI & API](../docs/AI-API-CONNECTIONS.md)
- [Telegram](../docs/TELEGRAM.md)
- [Discord](../docs/DISCORD.md)
- [Penyimpanan](../docs/STORAGE.md)
- [Pemulihan](../docs/RESTORE.md)
- [Migrasi](../docs/MIGRATION.md)
- [Pengelolaan sistem](../docs/SYSTEM.md)
- [mjust](../docs/MJUST.md)

Untuk pengembangan citra sistem, pengujian, dan rilis, lihat [panduan pengembang](../README-DEV.md).

## ⛏️ Perangkat lunak Minecraft dan EULA

JustVoxel **tidak** menyertakan berkas biner server Minecraft, perangkat lunak server Mojang, dunia yang telah dibuat sebelumnya, atau EULA Minecraft yang telah disetujui sebelumnya.

Saat membuat server, administrator menyetujui EULA Minecraft dan JustVoxel menyiapkan runtime server Minecraft yang dipilih dari opsi yang didukung.

Minecraft, perangkat lunak Mojang, Paper, Purpur, Spigot, Pufferfish, Leaf, Fabric, NeoForge, Forge, Quilt, plugin, mod, dan komponen pihak ketiga lainnya tetap tunduk pada lisensi dan ketentuan distribusi masing-masing.

## 📄 Lisensi

JustVoxel menggunakan lisensi **Apache-2.0**. Lihat [LICENSE](../LICENSE).

Pemberitahuan pihak ketiga dicatat dalam [THIRD_PARTY_NOTICES.md](../THIRD_PARTY_NOTICES.md).
