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

> **Çeviri notu:** Bu JustVoxel çevirisi, esas alınan İngilizce metinden yapay zekâ yardımıyla hazırlanmıştır. Hatalı, anlaşılması güç veya kulağa doğal gelmeyen bir ifadeyle karşılaşırsanız lütfen bir [GitHub sorun kaydı](https://github.com/home-server-project/justvoxel/issues) açarak ya da bir [değişiklik isteği (pull request)](https://github.com/home-server-project/justvoxel/pulls) göndererek düzeltme önerin.

<p align="center">
  <strong>Minecraft sunucularınız. Kendi donanımınızda, kiralık bir fiziksel sunucuda veya barındırılan bir sanal makinede. Tek bir Kontrol Merkezi.</strong>
</p>

<p align="center">
  Amacı işletim sistemi yönetmekten ziyade Minecraft sunucusu çalıştırmak olanlar için, kendi ortamınızda barındırabileceğiniz hazır bir Minecraft sunucu sistemi.
</p>

<p align="center">
  <a href="#-justvoxelu-kurun">Kurulum</a> •
  <a href="#-minecraft-sunucusu-barındırmak-artık-daha-kolay">Özellikler</a> •
  <a href="../docs/INSTALLER.md">Belgeler</a>
</p>

<p align="center">
  <img src="../docs/images/justvoxel-overview-1.webp" width="100%" alt="JustVoxel Kontrol Merkezi'ne genel bakış"><img src="../docs/images/justvoxel-overview-2.webp" width="100%" alt=""><img src="../docs/images/justvoxel-overview-3.webp" width="100%" alt="">
</p>

JustVoxel, kendi ortamınızda barındırabileceğiniz, Minecraft'a özel hazır bir sunucu sistemidir. Uyumlu bir PC'ye, dizüstü bilgisayara, fiziksel sunucuya, sanal makineye veya kendi işletim sistemi imajınızı çalıştırmanıza izin veren bir barındırma hizmetine kurun. Ardından Minecraft'ı tarayıcıdan yönetin.

Bakımını yapmanız gereken bir betik koleksiyonu veya ayrıca kurmanız gereken bir Minecraft yönetim paneli yok.

Eski bir dizüstü bilgisayarınız veya PC'niz mi var? **JustVoxel HWS ile bu donanımı bir çekmecede bekletmek yerine Minecraft'a özel bir sunucuya dönüştürebilirsiniz.** Desteklenen dizüstü bilgisayarlarda, kapak kapatıldığında uykuya geçişi engelleyerek sunucunun çalışmaya devam etmesini sağlayabilirsiniz. Ayrıca yerel konsol ekranını hemen veya kısa bir gecikmenin ardından kapatabilirsiniz.

> 🚧 **Ön sürüm:** JustVoxel ilk genel sürümüne hazırlanıyor. Son sürüm doğrulaması devam ederken mevcut kurulum ISO'sunu test edebilir ve erkenden kullanmaya başlayabilirsiniz.

## ✨ Minecraft sunucusu barındırmak artık daha kolay

JustVoxel, Minecraft yönetimini sunucunun çalıştığı işletim sistemiyle bir araya getirir.

- 🎮 **Birden fazla Minecraft sunucusu** — farklı dünyaları ve sunucu türlerini tek bir sistemde birbirinden bağımsız çalıştırın.
- 🖥️ **Web Kontrol Merkezi** — sunucuları bilgisayar, tablet veya telefondan yönetin.
- 🧩 **Eklentiler ve Modlar** — desteklenen sağlayıcılardaki uyumlu içerikleri keşfedin ve yönetin.
- 💾 **Yedekleme ve Geri Yükleme** — zamanlanmış ve manuel yedeklerle her sunucuyu koruyun.
- 🌐 **Uzaktan Erişim** — tümleşik dört özel erişim ve oyun tüneli seçeneği arasından seçim yapın.
- 🤖 **Yapay Zekâ ve Otomasyon** — uyumlu yapay zekâ asistanlarını ve uygulamalarını MCP ve REST üzerinden bağlayın.
- 💬 **Telegram ve Discord** — isteğe bağlı bot komutları, bildirimler ve topluluk iş akışları.
- 🛡️ **Sistem Yönetimi** — Linux komutlarını öğrenmek zorunda kalmadan ağı, depolamayı, güncellemeleri, sistem sağlığını, günlükleri ve desteklenen güç işlemlerini yönetin.

JustVoxel aileler, arkadaş grupları, küçük Minecraft toplulukları ve geleneksel bir Minecraft barındırma paneli kiralamak yerine kendi sunucusuna sahip olmak isteyenler için tasarlanmıştır.

## 🖥️ Tek sistem, iki sürüm

### JustVoxel VM

Şu sanal makine ve barındırma ortamları için:

- Proxmox
- KVM / libvirt
- VMware
- Hyper-V
- VirtualBox
- Uyumlu bir özel işletim sistemi imajına izin veren barındırılan sanal makineler veya VPS hizmetleri

VM sürümü, sanal makinede gereksiz olan fiziksel donanım paketleri hariç Minecraft sunucu sisteminin tamamını içerir.

### JustVoxel HWS

Doğrudan fiziksel donanıma kurulum için:

- Eski bir dizüstü bilgisayar
- Yeniden değerlendirilen bir masaüstü bilgisayar
- Mini PC
- Ev sunucusu
- Özel kurulum ortamına izin veren kiralık bir fiziksel sunucu (bare metal)

HWS, fiziksel makineler için UPS/NUT entegrasyonu, SMART ve NVMe araçları, sensörler, ürün yazılımı ve UEFI denetimleri, aygıt yazılımları, donanım tanılama araçları ve dizüstü bilgisayarların ekran ve kapak denetimlerini ekler.

Her iki sürüm de aynı JustVoxel Kontrol Merkezi'ni, Management API'yi, Minecraft iş akışlarını, depolama modelini, yedekleme ve ağ özelliklerini ve güncelleme deneyimini sunar.

[VM ve HWS sürümlerinin desteklediği özellikleri inceleyin](../docs/VARIANTS.md)

## 🚀 JustVoxel'u kurun

Tek bir kurulum ISO'su iki JustVoxel sürümünü de destekler.

[⬇️ JustVoxel x86_64 ISO dosyasını indirin](https://sourceforge.net/projects/justvoxel/files/JustVoxel-x86_64.iso/download)

**Önyüklenebilir bir USB bellek hazırlayın:** ISO dosyasını [balenaEtcher](https://etcher.balena.io/), [Rufus](https://rufus.ie/en/), [Ventoy](https://www.ventoy.net/en/index.html), [Fedora Media Writer](https://fedoraproject.org/workstation/download/) veya disk kalıplarını doğrudan yazabilen başka bir araçla USB belleğe yazın. Linux ve diğer Unix benzeri sistemlerde geleneksel `dd` yöntemi de işe yarar. Bir disk kalıbını doğrudan USB belleğe yazmak, bellekteki tüm verileri siler.

Kurulum yapacağınız ortama uygun sürümü seçin:

- **JustVoxel VM** — sanal makineler ve uyumlu sanal makine barındırma ortamları için
- **JustVoxel HWS** — PC'ler, dizüstü bilgisayarlar, mini PC'ler, fiziksel ev sunucuları ve uyumlu fiziksel sunucu barındırma hizmetleri için

Kurulum, disk hazırlığı ve ilk oturum açma için [kurulum kılavuzuna](../docs/INSTALLER.md) bakın.

## 🏠 Öncelik Minecraft, kullanım alanı daha geniş

JustVoxel öncelikle bir Minecraft sunucu sistemi olarak geliştirilir ve desteklenir. Temelinde ise **Podman, Quadlet, systemd, SSH, NetworkManager ve standart Linux yönetim araçlarını içeren bir AlmaLinux 10 bootc sunucusu** bulunur.

İleri düzey kullanıcılar aynı makineyi daha kapsamlı bir ev sunucusu olarak kullanabilir ve Minecraft'ın yanında başka konteynerler de çalıştırabilir.

Örneğin şu hizmetleri elle kurabilirsiniz:

- Medya için **Jellyfin** veya **Plex**
- Parola yönetimi için **Vaultwarden**
- Fotoğraf ve video yönetimi için **Immich**
- Ev otomasyonu için **Home Assistant Container**
- Dosyalar ve iş birliği için **Nextcloud**
- **Syncthing** veya **Navidrome**
- Kendi ortamınızda barındırabileceğiniz bir yapay zekâ aracısı veya ağ geçidi olarak **OpenClaw**

Bu uygulamalar **JustVoxel Kontrol Merkezi tarafından yönetilmez**. Bu uygulamaların Podman konteynerleri, Quadlet tanımları, depolaması, bağlantı noktaları, güncellemeleri, yedekleri, ters proxy yapılandırması ve uygulamaya özgü gereksinimleri yöneticinin sorumluluğundadır.

JustVoxel bu sınırı bilinçli olarak basit tutar. Minecraft sunucu sistemini kendisi yönetir. İleri düzey kullanıcılar diğer işlerde standart Podman ve Quadlet iş akışlarını kullanmaya devam edebilir.

JustVoxel bakım işlemleri de Minecraft'la ilgisi olmayan konteynerleri yöneticinin kendi iş yükleri olarak ele alır. Minecraft'taki **Baştan başla** gibi işlemler, Minecraft'la ilgisi olmayan Podman konteynerlerini kaldırmaz.

Böylece JustVoxel makinenizi öncelikle Minecraft sunucusu olarak kullanabilir, ihtiyaç duyduğunuzda aynı makinede konteyner tabanlı bir ev sunucusu da çalıştırabilirsiniz.

## 🎮 İstediğiniz Minecraft sunucusunu oluşturun

Makinede yeterli bellek ve boş bağlantı noktası varsa JustVoxel aynı anda birden fazla bağımsız Minecraft sunucusu çalıştırabilir.

Her sunucunun kendine ait kimliği, yapılandırması, çalışma ortamı, bağlantı noktaları, verileri, yedekleri, yaşam döngüsü denetimleri ve durum bilgisi vardır.

### Mevcut sunucu yazılımları

| Sunucu yazılımı | İçerik modeli | Yönetilen Java → Bedrock çapraz oyun desteği |
| --- | --- | --- |
| **Paper** | Eklentiler | ✅ |
| **Purpur** | Eklentiler | ✅ |
| **Spigot** | Eklentiler | ✅ |
| **Pufferfish** | Eklentiler | ✅ |
| **Leaf** | Eklentiler | ✅ |
| **Fabric** | Modlar | ✅ Tam uyumluluk doğrulandığında |
| **NeoForge** | Modlar | ✅ Tam uyumluluk doğrulandığında |
| **Forge** | Modlar | Yalnızca Java |
| **Quilt** | Modlar | Yalnızca Java |
| **Vanilla Java** | Eklenti veya mod yok | Yalnızca Java |
| **Yerel Bedrock** | Yerel Bedrock sunucusu | Yerel Bedrock istemcileri |

İlk sunucunuzu adım adım oluşturmak için **Basit kurulum** seçeneğini kullanın. Bellek, bağlantı noktaları, depolama, yedekler, sürüm politikası ve uyumlu çapraz oyun seçenekleri üzerinde daha fazla denetim istiyorsanız **Gelişmiş kurulum** seçeneğini kullanın.

JustVoxel, hangi özelliklerin kullanılabileceğini seçtiğiniz sunucu yazılımına göre doğrular.

[Sunucu yazılımlarını ve uyumluluk ayrıntılarını inceleyin](../docs/server-software.md)

## 🧩 Eklentiler ve Modlar

Seçtiğiniz sunucu için **Minecraft → Eklentiler ve Modlar** bölümünü açın.

JustVoxel, sunucu türüne bağlı olarak şu sağlayıcıları kullanabilir:

- [Modrinth](https://modrinth.com/)
- [Hangar](https://hangar.papermc.io/)
- Yapılandırdığınız API anahtarıyla [CurseForge](https://www.curseforge.com/minecraft)

Desteklenen kurulumlarda uyumluluk denetimleri, doğrulanmış indirmeler, sunucu durdurularak alınan yedekler, yeniden başlatma doğrulaması, sahiplik takibi ve kurtarma önlemleri kullanılır.

Paper ailesindeki sunucular eklenti kataloglarını kullanabilir. Fabric, Forge, NeoForge ve Quilt, yükleyici ve Minecraft sürümünün tam uyumluluğu kontrol edilerek mod kataloglarını kullanır. Vanilla Java ve yerel Bedrock için eklenti veya mod kataloğu bilinçli olarak sunulmaz.

[Eklentiler ve Modlar hakkında bilgi edinin](../docs/plugin-workspace.md)

## 🌐 İsteğe bağlı Minecraft Ağı

Birden fazla bağımsız sunucu çalıştırmak olağan bir kullanım şeklidir. Bunun için proxy **gerekmez**.

Uyumlu birden fazla sunucuya tek bir genel giriş noktası üzerinden erişim sağlamak isterseniz JustVoxel, Velocity tabanlı isteğe bağlı bir **Minecraft Ağı** sunar.

Kontrol Merkezi'nden desteklenen arka uç sunucularını seçebilir, herkese açık Java ve Bedrock bağlantı noktalarını yapılandırabilir, sanal lobiyi ve karşılama mesajını yönetebilir ve hangi sunucuların Ağa katılacağını belirleyebilirsiniz.

Paper, Purpur, Pufferfish ve Leaf, Ağın arka uç sunucuları olarak desteklenir. JustVoxel gerekli proxy uyumluluğunu doğruladığında uyumlu Fabric/Quilt ve Forge/NeoForge sunucuları da katılabilir. Spigot, Vanilla ve yerel Bedrock bağımsız sunucular olarak kalır.

[Minecraft Ağı hakkında bilgi edinin](../docs/minecraft-network.md)

## 🖥️ Kontrol Merkezi ile tanışın

JustVoxel'in arayüzü, sunucunuzda yapmak istediğiniz işlemler etrafında tasarlanmıştır.

### Minecraft Sunucuları

Birden fazla sunucu oluşturup yönetin, sunucuları başlatın veya durdurun, durum ve oyuncu bilgilerini inceleyin, desteklenen ayarları yapılandırın ve sunucuya özgü araçlara erişin.

### Eklentiler ve Modlar

Uyumlu içerikleri keşfedin, kurulu eklentileri ve modları inceleyin, desteklenen eklemeleri yönetin ve bir değişiklik müdahale gerektirdiğinde kurtarma bilgilerini görüntüleyin.

### Yedekleme ve Depolama

Yedekler oluşturun, yedekleme geçmişini inceleyin, sunucu verilerinin tamamını geri yükleyin, yedekleme hedeflerini yapılandırın, sunucuları içe veya dışa aktarın ve desteklenen depolama alanlarını yönetin.

### Ağ

Ethernet, Wi-Fi, **bağlantı sorunlarını giderme** ve Uzaktan Erişim seçeneklerini yönetin. Bağlantıyı kesebilecek ağ değişikliklerinde geri alma koruması kullanılır. Böylece hatalı bir IP veya Wi-Fi ayarı nedeniyle erişiminizi kalıcı olarak kaybetmezsiniz.

### Sistem

Sistem sağlığını, kaynakları, günlükleri, işletim sistemi güncellemelerini, kullanıcıları, güvenliği, desteklenen güç işlemlerini ve HWS'ye özgü donanım denetimlerini inceleyin.

### Yapay Zekâ ve API Bağlantıları

Desteklenen yapay zekâ uygulamalarını, MCP istemcilerini, REST otomasyonlarını, Telegram'ı ve Discord'u ayrı bir yönetim altyapısı kurmadan bağlayın.

Terminali mi tercih edersiniz? JustVoxel, aynı güvenlik kurallarına tabi etkileşimli terminal yönetim arayüzü **mjust**'ı da içerir.

[WebUI belgeleri](../docs/WEBUI.md) · [mjust kılavuzu](../docs/MJUST.md)

## 🌐 Arkadaşlarınız sunucunuza dilediğiniz şekilde bağlansın

JustVoxel, **Ağ → Uzaktan Erişim** bölümünde şu anda dört tümleşik uzaktan erişim seçeneği sunar.

Desteklenen sağlayıcılar için JustVoxel durum bilgilerini gösterir ve sağlayıcıya göre etkinleştirme, devre dışı bırakma ve kurulum adımları sunar.

| Seçenek | En uygun kullanım | Sunduğu olanaklar |
| --- | --- | --- |
| **[Tailscale](https://tailscale.com/)** | Aile ve arkadaşlar için özel sunucular | Onaylı cihazlar arasında özel örgü ağ |
| **[NetBird](https://netbird.io/)** | Erişimi yönetilen özel ağlar | Hesap ve erişim politikalarının yönetilebildiği özel ağ |
| **[Playit.gg](https://playit.gg/)** | Yönlendiricide standart bağlantı noktası yönlendirmesi yapmadan Minecraft sunucusuna genel erişim | Herkese açık oyun tünelleri ve bağlantı adresleri |
| **[NordVPN Meshnet](https://meshnet.nordvpn.com/)** | NordVPN kullanıcıları için cihazlar arası özel erişim | JustVoxel'u ticari bir VPN istemcisi olarak kullanmadan Meshnet bağlantısı |

### Hangisini seçmeliyim?

**🎮 Arkadaşlarınız özel bir VPN istemcisi kurmadan mı katılsın?**  
İlk olarak **Playit.gg**'ye göz atın. Oyuncular yapılandırdığınız oyun tünelinin genel adresini kullanabilir.

**🔒 Yalnızca güvenilen cihazların erişebileceği özel bir sunucu mu istiyorsunuz?**  
**Tailscale** veya **NetBird**'ü değerlendirin.

**🌍 Zaten NordVPN Meshnet kullanıyor musunuz?**  
JustVoxel, Meshnet entegrasyonunu yönetebilir. Bu özellik, sistemi standart bir ticari NordVPN çıkış sunucusuna bağlamaktan ayrıdır.

Sağlayıcı hesapları, bulut tarafındaki politikalar ve tünel yapılandırması ilgili sağlayıcının hizmetinde yönetilir. JustVoxel, kendi tarafında desteklenen entegrasyonu yönetir.

Birden fazla VPN veya örgü ağ ürününü aynı anda kullanabilirsiniz, ancak yönlendirme, DNS veya güvenlik duvarı kuralları çakışabilir.

[Ağ ve Uzaktan Erişim kılavuzunu okuyun](../docs/NETWORKING.md)

## 🤖 Yapay Zekâ, MCP, REST ve Otomasyon

Bir yapay zekâ asistanına sunucu durumunu sormak, ev otomasyonunuzu bağlamak veya kendi yönetim iş akışınızı oluşturmak ister misiniz?

JustVoxel, **Yapay Zekâ ve API Bağlantıları** üzerinden yerleşik **MCP ve REST arayüzleri** sunar.

Kurulum ekranında uyumlu uygulamalar ve bağlantı türleri için şu seçenekler bulunur:

- **OpenAI — ChatGPT ve Codex**
- **Claude**
- **Grok**
- **Gemini**
- **GitHub Copilot**
- **Diğer yapay zekâ / MCP uygulamaları**
- **Otomasyon / REST API**

Her bağlantının nelere erişebileceğine siz karar verirsiniz.

İzinler **Düzey 1 — Salt okunur** erişimden sunucu yönetimine, oradan da yalnızca açık onayla verilebilen gelişmiş sistem denetimine kadar uzanır. Bağlantıları belirli Minecraft sunucularıyla da sınırlayabilirsiniz.

JustVoxel'in MCP veya REST arayüzünü sunabilmesi için yapay zekâ modeline ait bir API anahtarı gerekmez. İstemcinin bu arayüzleri kullanabilmesi uygulamaya, hesap veya aboneliğin sunduğu özelliklere ve ağ erişimine bağlıdır.

Yapay zekâ ve API ağ geçidi, siz etkinleştirene kadar kapalıdır.

[Yapay Zekâ ve API Bağlantıları hakkında bilgi edinin](../docs/AI-API-CONNECTIONS.md)

## 💬 Telegram ve Discord

JustVoxel, WebUI ile aynı merkezi yönetim altyapısını kullanan, yerleşik ve isteğe bağlı Telegram ve Discord entegrasyonları içerir.

### Telegram

Bu iş için kullanılan bir Telegram botu şunları sunabilir:

- JustVoxel sisteminin ve Minecraft'ın durumu
- Sunucu seçimi ve oyuncu bilgileri
- Yetkili yöneticiler için sunucu başlatma, durdurma ve yeniden başlatma
- Manuel yedekleme
- Minecraft ve konteyner güncellemelerini inceleme
- İşletim sistemi güncelleme bilgileri
- Denetimli sistem yeniden başlatma işlemleri
- Sunucu durumu, yedekleme sonuçları ve güncelleme olup olmadığı hakkında isteğe bağlı bildirimler

Telegram kendi **Salt okunur** ve **Sunucu yönetimi** izinlerini kullanır ve bir yapay zekâ modeli gerektirmez.

[Telegram kurulumu ve izinleri](../docs/TELEGRAM.md)

### Discord

Discord, hem topluluk üyelerinin Minecraft sunucusuyla etkileşim kurduğu hem de yöneticilerin sunucuyu yönettiği bir arayüz olabilir.

Açıkça bağlanan Discord sunucusundaki topluluk üyeleri temel sunucu durumunu görebilir ve Minecraft erişimi isteyebilir. Bu işlemler üyelere otomatik olarak yönetim yetkisi vermez.

Yetkili JustVoxel Discord yöneticileri sunucuyu başlatabilir, durdurabilir ve yeniden başlatabilir, manuel yedek alabilir, güncellemeleri inceleyebilir, Minecraft Ağı'nı yönetebilir ve desteklenen diğer gelişmiş işlemleri yapabilir.

Discord üyeliği, Minecraft izin listesi üzerinden erişim ve JustVoxel yönetimi ayrı izinler olarak kalır.

[Discord kurulumu, topluluk erişimi ve yönetimi](../docs/DISCORD.md)

## 💾 Yedekleme, Geri Yükleme, Depolama ve Taşıma

Minecraft verileri, sistem imajından daha önemlidir.

JustVoxel, Minecraft verilerini ve yedekleri değişmez işletim sistemi kurulumunun dışında tutar. Verilerin ve yedeklerin nerede saklanacağını ayrı ayrı seçebilirsiniz.

Desteklenen depolama seçenekleri şunlardır:

- Sistem dosya sistemi
- Bu işe ayrılmış dahili diskler
- USB depolama
- Mevcut yerel bölümler ve dosya sistemleri
- Diskteki mevcut ayrılmamış alandan oluşturulan yeni bölümler
- NFS
- SMB/CIFS

Her sunucu için otomatik ve manuel yedekleme kullanılabilir. Geri yükleme, seçilen sunucunun kalıcı verilerinin tamamını yedekteki verilerle değiştirir. Sunucunun JustVoxel kimliği ve sistem yapılandırması korunur.

Sunucuları başka bir makineye veya yeniden kurulan bir sisteme taşımak için JustVoxel, gözden geçirme adımları içeren **Dışa Aktarma / İçe Aktarma / Kurtarma** iş akışları da sunar.

[Depolama](../docs/STORAGE.md) · [Geri yükleme](../docs/RESTORE.md) · [Taşıma](../docs/MIGRATION.md)

## 🛡️ Değişiklikleri daha güvenli yapın

JustVoxel, rastgele kabuk komutları çalıştırmak yerine desteklenen işlemleri sistem genelinde geçerli güvenlik kurallarıyla yürütür.

Bu kurallar şunları içerir:

- Sunucunun çalışmasını aksatabilecek işlemlerden önce oyuncu kontrolü
- **Uygula** seçeneğini kullanmadan önce değişiklikleri gözden geçirme
- Kurtarma güvencesi gereken değişikliklerde sunucuyu durdurarak yedek alma
- Bir sunucunun yanlışlıkla başka bir sunucunun yerini almasını önleyen sunucu kimliğini birebir eşleştirme
- Yeniden başlatma ve çalışma durumu doğrulaması
- İşlem ve kurtarma durumunun kalıcı olarak saklanması
- Desteklenen işlemler için geri alma yolları
- Riskli ağ değişiklikleri için NetworkManager kontrol noktaları
- Minecraft ve işletim sistemi için ayrı güncelleme iş akışları

Tarayıcıdan erişilen WebUI ayrıcalıksız çalışır. Tarayıcıya bir root kabuğu açmak yerine ayrıcalık gerektiren değişiklikler, yerel JustVoxel Management Agent ve açıkça tanımlanmış API'si üzerinden yapılır.

## 🔄 Sistem ve Minecraft güncellemeleri ayrı yürütülür

JustVoxel, Home Server Base 10 / AlmaLinux 10 üzerine kurulu, imajla yönetilen bir Minecraft sunucu sistemidir.

İşletim sistemi güncellemeleri Minecraft sürümlerinden, sunucu yazılımlarından, eklentilerden, modlardan ve konteyner güncellemelerinden ayrı yürütülür.

Böylece JustVoxel'i güncellemek Minecraft sürümünüzü siz fark etmeden değiştirmez. Bir Minecraft sunucusunda değişiklik yapmak da ana işletim sistemini yeniden derlemeyi gerektirmez.

[Sistem yönetimi belgeleri](../docs/SYSTEM.md)

## 📊 JustVoxel diğer projelerle nasıl karşılaştırılır?

JustVoxel'i geleneksel Minecraft kontrol panellerinden ayıran özellik, **işletim sistemi ile Minecraft yönetim katmanını tek bir hazır sunucu sisteminde birleştirmesidir**.

| Özellik | **JustVoxel** | **MineOS SvelteKit** | **Crafty Controller** | **MineOS Node** |
| --- | :---: | :---: | :---: | :---: |
| Minecraft'a özel eksiksiz sunucu işletim sistemi | ⭐ | — | — | — |
| Birden fazla Minecraft sunucusu | ✅ | ✅ | ✅ | ✅ |
| Java ve Bedrock sunucu desteği | ✅ | ✅ | ✅ | 🟡 Java ve eski PocketMine desteği |
| Rehberli eklenti ve mod keşfi | ✅ Modrinth, Hangar, CurseForge | ✅ Modrinth, CurseForge | — | — |
| Ana sistemin ağ, depolama ve işletim sistemi yönetimi | ⭐ | — | — | — |
| Tümleşik özel erişim ve oyun tünelleri | ⭐ 4 seçenek | — | — | — |
| MCP ve REST üzerinden yapay zekâ ve otomasyon ağ geçidi | ⭐ | — | — | — |
| Yerleşik Telegram ve Discord yönetimi | ⭐ | — | — | — |
| Dizüstü bilgisayar kapağı ve ekranı için sistem denetimleri | ✅ HWS | — | — | — |
| İmajla yönetilen değişmez işletim sistemi güncellemeleri | ✅ | — | — | — |
| Terminal yönetim arayüzü | ✅ mjust | ✅ mineos CLI/TUI | — | ✅ CLI |
| Kurulum modeli | Eksiksiz bootc sunucu sistemi | Docker / Compose uygulaması | Docker, Linux veya Windows üzerinde kontrol paneli | Linux/BSD üzerinde Node.js uygulaması |

⭐ öne çıkan &nbsp;•&nbsp; ✅ destekleniyor &nbsp;•&nbsp; 🟡 kısmi veya eski destek &nbsp;•&nbsp; — incelenen temel proje belgelerinde yer almıyor

**Karşılaştırmanın son inceleme tarihi 16 Kasım 2026.** Amaç diğer projeleri “kötü” ilan etmek değil. MineOS ve Crafty, kendiniz yönetmek istediğiniz bir işletim sistemine zaten sahipseniz çok iyi seçenekler olabilecek Minecraft yönetim uygulamalarıdır. JustVoxel, Minecraft panelinin **ve onu çalıştıran sunucu sisteminin** tek bir ürün olarak yönetilmesini isteyen kullanıcılara yöneliktir.

Karşılaştırma, projelerin güncel belgelerine ve depolarına dayanır:

- [MineOS SvelteKit](https://github.com/freeman412/mineos-sveltekit)
- [Crafty Controller](https://craftycontrol.com/) · [Kaynak kodu](https://gitlab.com/crafty-controller/crafty-4)
- [MineOS Node](https://github.com/hexparrot/mineos-node)

Karşılaştırmada güncelliğini yitirmiş veya haksız bir değerlendirme varsa lütfen bir GitHub issue'su açın. JustVoxel'i daha olumsuz gösteren düzeltmelere de açığız.

## 📚 Belgeler

Ayrıntılı JustVoxel kılavuzlarını [belgeler dizininde](../docs/) bulabilirsiniz.

Başlangıç için şu kılavuzlara bakabilirsiniz:

- [WebUI](../docs/WEBUI.md)
- [Minecraft çalışma ortamı](../docs/MINECRAFT_RUNTIME.md)
- [Sunucu yazılımları](../docs/server-software.md)
- [Eklentiler ve Modlar](../docs/plugin-workspace.md)
- [Minecraft Ağı](../docs/minecraft-network.md)
- [Ağ ve Uzaktan Erişim](../docs/NETWORKING.md)
- [Yapay Zekâ ve API Bağlantıları](../docs/AI-API-CONNECTIONS.md)
- [Telegram](../docs/TELEGRAM.md)
- [Discord](../docs/DISCORD.md)
- [Depolama](../docs/STORAGE.md)
- [Geri yükleme](../docs/RESTORE.md)
- [Taşıma](../docs/MIGRATION.md)
- [Sistem yönetimi](../docs/SYSTEM.md)
- [mjust](../docs/MJUST.md)

İmaj geliştirme, test ve sürüm yayımlama için [geliştirici kılavuzuna](../README-DEV.md) bakın.

## ⛏️ Minecraft yazılımı ve EULA

JustVoxel, Minecraft sunucusunun ikili dosyalarını, Mojang sunucu yazılımını, önceden oluşturulmuş bir dünyayı veya önceden kabul edilmiş bir Minecraft EULA'sını **içermez**.

Sunucu oluşturulurken yönetici, Minecraft Son Kullanıcı Lisans Sözleşmesi'ni (EULA) kabul eder. JustVoxel de seçilen ve desteklenen Minecraft sunucusunun çalışma ortamını hazırlar.

Minecraft, Mojang yazılımları, Paper, Purpur, Spigot, Pufferfish, Leaf, Fabric, NeoForge, Forge, Quilt, eklentiler, modlar ve diğer üçüncü taraf bileşenleri kendi lisans ve dağıtım koşullarına tabidir.

## 📄 Lisans

JustVoxel, **Apache-2.0** lisansı altında sunulur. Ayrıntılar [LICENSE](../LICENSE) dosyasında yer alır.

Üçüncü taraf bildirimleri [THIRD_PARTY_NOTICES.md](../THIRD_PARTY_NOTICES.md) dosyasında tutulur.
