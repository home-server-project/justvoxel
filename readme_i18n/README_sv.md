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

> **Om översättningen:** Den här översättningen av JustVoxel har tagits fram med hjälp av AI utifrån den engelska originaltexten. Om du hittar en felaktig, otydlig eller onaturlig formulering får du gärna föreslå en rättelse genom att öppna ett [ärende på GitHub](https://github.com/home-server-project/justvoxel/issues) eller skicka ett [ändringsförslag (pull request)](https://github.com/home-server-project/justvoxel/pulls).

<p align="center">
  <strong>Dina Minecraft-servrar på egen hårdvara, en hyrd dedikerad server eller en virtuell maskin hos en leverantör. Allt i ett kontrollcenter.</strong>
</p>

<p align="center">
  Ett komplett system för dig som vill driva egna Minecraft-servrar utan att behöva administrera det underliggande operativsystemet.
</p>

<p align="center">
  <a href="#-installera-justvoxel">Installera</a> •
  <a href="#-enklare-att-drifta-minecraft-servrar">Funktioner</a> •
  <a href="../docs/INSTALLER.md">Dokumentation</a>
</p>

<p align="center">
  <img src="../docs/images/justvoxel-overview-1.webp" width="100%" alt="Översikt över JustVoxels kontrollcenter"><img src="../docs/images/justvoxel-overview-2.webp" width="100%" alt=""><img src="../docs/images/justvoxel-overview-3.webp" width="100%" alt="">
</p>

JustVoxel är ett dedikerat system för att driva egna Minecraft-servrar. Du kan installera det på en kompatibel PC, bärbar dator, fysisk server eller virtuell maskin, eller hos en värdtjänst där du får använda en egen operativsystemsavbildning. Sedan hanterar du Minecraft i webbläsaren.

Du behöver varken underhålla egna skript eller installera en separat kontrollpanel för Minecraft.

Har du en gammal bärbar dator eller PC liggande? **Med JustVoxel HWS kan den få nytt liv som dedikerad Minecraft-server.** På bärbara datorer som stöds kan du stänga av viloläget när locket stängs, så att servern fortsätter att köra med locket nere. Via den lokala konsolen kan du också stänga av skärmen direkt eller efter en kort stund.

> 🚧 **Förhandsversion:** JustVoxel förbereds för sin första offentliga lansering. Den aktuella installationsavbildningen (ISO) finns tillgänglig för testning och tidig användning medan de sista kontrollerna inför lanseringen pågår.

## ✨ Enklare att drifta Minecraft-servrar

JustVoxel samlar Minecraft-hanteringen och det underliggande operativsystemet i ett system.

- 🎮 **Flera Minecraft-servrar** – kör olika världar och servertyper oberoende av varandra på samma system.
- 🖥️ **Webbaserat kontrollcenter** – hantera servrar från en dator, surfplatta eller mobil.
- 🧩 **Insticksprogram och moddar** – hitta och hantera kompatibelt innehåll från innehållskällor som stöds.
- 💾 **Säkerhetskopiering och återställning** – skydda varje server med schemalagda och manuella säkerhetskopior.
- 🌐 **Fjärråtkomst** – välj mellan fyra integrerade lösningar för privat åtkomst och tunnlar för speltrafik.
- 🤖 **AI och automatisering** – anslut kompatibla AI-assistenter och appar via MCP och REST.
- 💬 **Telegram och Discord** – använd botar, aviseringar och funktioner för spelgemenskapen om du vill.
- 🛡️ **Systemhantering** – hantera nätverk, lagring, uppdateringar, systemhälsa, loggar och tillgängliga strömåtgärder utan att behöva lära dig Linux-kommandon.

JustVoxel är gjort för familjer, vänner, mindre Minecraft-gemenskaper och andra som vill äga sin egen server i stället för att hyra en traditionell Minecraft-värdtjänst med kontrollpanel.

## 🖥️ Ett system, två utgåvor

### JustVoxel VM

För virtuella maskiner och värdmiljöer som:

- Proxmox
- KVM / libvirt
- VMware
- Hyper-V
- VirtualBox
- virtuella maskiner eller VPS-tjänster hos leverantörer som tillåter en egen kompatibel operativsystemsavbildning

VM-utgåvan innehåller hela Minecraft-systemet utan paket för fysisk hårdvara som inte behövs i en virtuell maskin.

### JustVoxel HWS

För installation direkt på fysisk hårdvara, till exempel:

- en äldre bärbar dator
- en återanvänd stationär dator
- en mini-PC
- en hemmaserver
- en hyrd fysisk server där eget installationsmedium är tillåtet

HWS har funktioner för fysisk hårdvara, bland annat UPS/NUT-integration, verktyg för SMART och NVMe, sensorer, hantering av fast programvara och UEFI, fast programvara för enheter, hårdvarudiagnostik samt skärm- och lockinställningar för bärbara datorer.

Båda utgåvorna har samma kontrollcenter, Management API, Minecraft-funktioner, lagringsmodell, säkerhetskopiering, nätverksfunktioner och uppdateringsflöde.

[Läs om funktionsstödet i VM och HWS](../docs/VARIANTS.md)

## 🚀 Installera JustVoxel

Samma ISO-avbildning används för att installera båda utgåvorna av JustVoxel.

[⬇️ Ladda ner JustVoxel x86_64 ISO](https://sourceforge.net/projects/justvoxel/files/JustVoxel-x86_64.iso/download)

**Skapa ett startbart USB-minne:** Skriv ISO-avbildningen till ett USB-minne med [balenaEtcher](https://etcher.balena.io/), [Rufus](https://rufus.ie/en/), [Ventoy](https://www.ventoy.net/en/index.html), [Fedora Media Writer](https://fedoraproject.org/workstation/download/) eller ett annat verktyg som kan skriva diskavbildningar direkt. På Linux och andra Unix-liknande system fungerar även den traditionella metoden med `dd`. När en avbildning skrivs direkt till USB-minnet raderas allt som finns på det.

Välj den utgåva som passar din installationsmiljö:

- **JustVoxel VM** för virtuella maskiner och kompatibla VM-miljöer hos leverantörer
- **JustVoxel HWS** för PC, bärbara datorer, mini-PC, fysiska hemmaservrar och kompatibla värdtjänster med fysisk server

I installationsguiden beskrivs hur du installerar JustVoxel, förbereder disken och loggar in första gången. [Installationsguide](../docs/INSTALLER.md)

## 🏠 Minecraft i första hand, med plats för annat

JustVoxel utvecklas och stöds i första hand som ett system för Minecraft-servrar. I grunden är det en vanlig **AlmaLinux 10-server med bootc, Podman, Quadlet, systemd, SSH, NetworkManager och vanliga administrationsverktyg för Linux**.

Avancerade användare kan använda samma maskin som hemmaserver för fler ändamål och köra andra containrar vid sidan av Minecraft.

Du kan till exempel installera följande tjänster manuellt:

- **Jellyfin** eller **Plex** för medieuppspelning
- **Vaultwarden** för lösenordshantering
- **Immich** för hantering av bilder och videor
- **Home Assistant Container** för hemautomation
- **Nextcloud** för filer och samarbete
- **Syncthing** eller **Navidrome**
- **OpenClaw** som egen AI-agent eller gateway

De här programmen **hanteras inte av JustVoxels kontrollcenter**. Administratören ansvarar för programmens Podman-containrar, Quadlet-konfigurationer, lagring, portar, uppdateringar, säkerhetskopior, konfiguration av omvänd proxy och programspecifika krav.

JustVoxel sköter Minecraft-systemet. För andra tjänster kan avancerade användare använda Podman och Quadlet på vanligt sätt.

Vid underhåll hanteras fristående containrar som administratörens egna tjänster. Åtgärder som **Börja om** för Minecraft tar inte bort orelaterade Podman-containrar.

På så sätt kan JustVoxel-maskinen i första hand vara en Minecraft-server och samtidigt fungera som en vanlig containerbaserad hemmaserver när du behöver det.

## 🎮 Skapa den Minecraft-server du vill ha

JustVoxel kan köra flera oberoende Minecraft-servrar samtidigt om maskinen har tillräckligt med minne och lediga portar.

Varje server har en egen identitet, konfiguration, körmiljö, portar, data, säkerhetskopior, livscykelhantering och status.

### Serverprogramvara som stöds

| Serverprogramvara | Typ av innehåll | Spel över plattformsgränser (Java → Bedrock) med stöd i JustVoxel |
| --- | --- | --- |
| **Paper** | Insticksprogram | ✅ |
| **Purpur** | Insticksprogram | ✅ |
| **Spigot** | Insticksprogram | ✅ |
| **Pufferfish** | Insticksprogram | ✅ |
| **Leaf** | Insticksprogram | ✅ |
| **Fabric** | Moddar | ✅ När kompatibiliteten har verifierats för den exakta kombinationen |
| **NeoForge** | Moddar | ✅ När kompatibiliteten har verifierats för den exakta kombinationen |
| **Forge** | Moddar | Endast Java |
| **Quilt** | Moddar | Endast Java |
| **Vanilla Java** | Inga insticksprogram eller moddar | Endast Java |
| **Inbyggd Bedrock** | Inbyggd Bedrock-server | Bedrock-klienter som ansluter direkt |

Välj **Enkel konfiguration** för att få vägledning när du skapar din första server. Med **Avancerad konfiguration** får du mer kontroll över minne, portar, lagring, säkerhetskopiering, versionspolicy och plattformsspel där det stöds.

JustVoxel kontrollerar vilka funktioner den valda serverprogramvaran stöder.

[Läs om serverprogramvara och kompatibilitet](../docs/server-software.md)

## 🧩 Insticksprogram och moddar

Öppna **Minecraft → Insticksprogram och moddar** för den valda servern.

Beroende på servertyp kan JustVoxel använda följande kataloger:

- [Modrinth](https://modrinth.com/)
- [Hangar](https://hangar.papermc.io/)
- [CurseForge](https://www.curseforge.com/minecraft) med den API-nyckel du har ställt in

Vid installationer som stöds kontrollerar JustVoxel kompatibiliteten, verifierar hämtade filer, tar säkerhetskopior när servern är stoppad, verifierar omstarten, håller reda på ägarskapet och vidtar skyddsåtgärder för återställning.

Servrar i Paper-familjen kan använda kataloger med insticksprogram. Fabric, Forge, NeoForge och Quilt använder moddkataloger där den exakta moddladdaren och Minecraft-versionen kontrolleras. Kataloger för insticksprogram och moddar är avsiktligt otillgängliga för Vanilla Java och Inbyggd Bedrock.

[Läs om insticksprogram och moddar](../docs/plugin-workspace.md)

## 🌐 Valfritt Minecraft-nätverk

Du kan köra flera fristående servrar helt utan proxy.

Om du vill ha en gemensam publik ingång till flera kompatibla servrar finns även ett valfritt **Minecraft-nätverk** som bygger på Velocity.

I kontrollcentret kan du välja vilka bakomliggande servrar som ska ingå, ställa in publika portar för Java och Bedrock, hantera den virtuella lobbyn och välkomstmeddelandet och ändra vilka servrar som ingår i nätverket.

Paper, Purpur, Pufferfish och Leaf stöds som bakomliggande servrar i nätverket. Kompatibla servrar med Fabric/Quilt eller Forge/NeoForge kan också ingå när JustVoxel har verifierat att de fungerar med proxyn. Spigot, Vanilla och Inbyggd Bedrock körs fortfarande fristående.

[Läs om Minecraft-nätverk](../docs/minecraft-network.md)

## 🖥️ Det här finns i kontrollcentret

JustVoxel utgår från det du vill göra med servern. Du behöver inte känna till hur Linux fungerar under huven.

### Minecraft-servrar

Skapa och hantera flera servrar, starta och stoppa dem, se status och vilka som spelar, ändra tillgängliga inställningar och öppna verktygen för respektive server.

### Insticksprogram och moddar

Hitta kompatibelt innehåll, se vilka insticksprogram och moddar som är installerade, hantera tillägg som stöds och se återställningsinformation när en ändring behöver åtgärdas.

### Säkerhetskopior och lagring

Skapa säkerhetskopior, se historiken, återställ alla serverdata, välj var säkerhetskopior ska sparas, importera och exportera servrar och hantera lagring som stöds.

### Nätverk

Hantera Ethernet, Wi-Fi och fjärråtkomst och **felsök anslutningen**. Nätverksändringar som kan bryta anslutningen kan återställas så att en felaktig IP- eller Wi-Fi-inställning inte behöver låsa dig ute permanent.

### System

Se systemhälsa, resurser, loggar och uppdateringar av operativsystemet. Här hanterar du också användare, säkerhet, tillgängliga strömåtgärder och de hårdvarufunktioner som bara finns i HWS.

### AI- och API-anslutningar

Anslut AI-appar och MCP-klienter som stöds, REST-baserad automatisering, Telegram och Discord utan att sätta upp ett separat administrationssystem.

Föredrar du terminalen? JustVoxel innehåller också **mjust**, ett interaktivt administrationsgränssnitt i terminalen som följer samma säkerhetsregler som resten av systemet.

[Dokumentation för webbgränssnittet](../docs/WEBUI.md) · [Guide till mjust](../docs/MJUST.md)

## 🌐 Låt vänner ansluta på ett sätt som passar er

JustVoxel har för närvarande fyra integrerade alternativ under **Nätverk → Fjärråtkomst**.

Integrationen omfattar statusvisning, aktivering, inaktivering och vägledning för att komma igång, där det är tillämpligt för leverantören.

| Alternativ | Passar bäst för | Det här får du |
| --- | --- | --- |
| **[Tailscale](https://tailscale.com/)** | Privata servrar för vänner och familj | Privat meshnätverk mellan godkända enheter |
| **[NetBird](https://netbird.io/)** | Privata nätverk med styrd åtkomst | Privat nätverk med kontroll över konton och åtkomstregler |
| **[Playit.gg](https://playit.gg/)** | Publik åtkomst till Minecraft utan vanlig portvidarebefordran i routern | Publika speltunnlar och anslutningsadresser |
| **[NordVPN Meshnet](https://meshnet.nordvpn.com/)** | Privat åtkomst mellan enheter för NordVPN-användare | Anslutning via Meshnet utan att använda JustVoxel som klient för den kommersiella VPN-tjänsten |

### Vilket alternativ ska jag välja?

**🎮 Ska vänner kunna ansluta utan att installera en privat VPN-klient?**  
Titta först på **Playit.gg**. Spelarna kan använda den publika adressen till den speltunnel du har ställt in.

**🔒 Vill du ha en privat server för betrodda enheter?**  
Titta på **Tailscale** eller **NetBird**.

**🌍 Använder du redan NordVPN Meshnet?**  
JustVoxel kan hantera sin Meshnet-integration. Den fungerar oberoende av anslutningar till utgångsservrar i NordVPN:s kommersiella VPN-tjänst.

Du hanterar konton, åtkomstregler i molnet och tunnelinställningar hos respektive leverantör. JustVoxel sköter den del av integrationen som stöds på det lokala systemet.

Du kan använda flera VPN- eller meshprodukter samtidigt, men routning, DNS eller brandväggsregler kan hamna i konflikt.

[Läs guiden om nätverk och fjärråtkomst](../docs/NETWORKING.md)

## 🤖 AI, MCP, REST och automatisering

Vill du fråga en AI-assistent om serverstatus, koppla in hemautomation eller bygga ett eget flöde för att styra servern?

JustVoxel har inbyggda **MCP- och REST-gränssnitt** under **AI- och API-anslutningar**.

När du lägger till en anslutning kan du bland annat välja följande appar och anslutningstyper:

- **OpenAI · ChatGPT och Codex**
- **Claude**
- **Grok**
- **Gemini**
- **GitHub Copilot**
- **Annan AI-/MCP-app**
- **Automatisering / REST API**

Du bestämmer vad varje anslutning får komma åt.

Behörigheterna sträcker sig från **Nivå 1 – Skrivskyddad** till serverhantering och, endast om du uttryckligen godkänner det, avancerad systemstyrning. Du kan också begränsa en anslutning till vissa Minecraft-servrar.

JustVoxel behöver ingen API-nyckel till en AI-modell för att göra sitt MCP- eller REST-gränssnitt tillgängligt. Vilka klienter som faktiskt fungerar beror på appen, funktionerna i ditt konto eller abonnemang och nätverksåtkomsten.

Gatewayen för AI och API är avstängd tills du aktiverar den.

[Läs om AI- och API-anslutningar](../docs/AI-API-CONNECTIONS.md)

## 💬 Telegram och Discord

JustVoxel har inbyggda, valfria integrationer med Telegram och Discord. Båda använder samma centrala hanteringsbackend som webbgränssnittet.

### Telegram

En separat Telegram-bot kan ge dig:

- status för JustVoxel och Minecraft
- serverval och spelarinformation
- start, stopp och omstart för användare med godkänd hanteringsbehörighet
- manuella säkerhetskopior
- granskning av uppdateringar för Minecraft och containrar
- information om uppdateringar av operativsystemet
- kontrollerad omstart av systemet
- valfria aviseringar om serverstatus, resultat av säkerhetskopiering och tillgängliga uppdateringar

Telegram har egna behörigheter för **Skrivskyddad** åtkomst och **Serveradministration** och kräver ingen AI-modell.

[Konfiguration och behörigheter för Telegram](../docs/TELEGRAM.md)

### Discord

Discord kan användas både som mötesplats för Minecraft-spelare och som administrationsgränssnitt.

Medlemmar på den Discord-server som uttryckligen har kopplats till JustVoxel kan se grundläggande serverstatus och begära åtkomst till Minecraft utan att automatiskt få administratörsbehörighet.

Godkända JustVoxel-administratörer i Discord kan utföra serveråtgärder som stöds, till exempel start, stopp, omstart, manuell säkerhetskopiering, granskning av uppdateringar och hantering av Minecraft-nätverket, samt ytterligare avancerade åtgärder.

Medlemskap i Discord, åtkomst via Minecrafts tillåtelselista och administratörsbehörighet i JustVoxel är separata.

[Konfiguration, medlemsåtkomst och administration i Discord](../docs/DISCORD.md)

## 💾 Säkerhetskopiering, återställning, lagring och migrering

Dina Minecraft-data är viktigare än systemavbildningen.

JustVoxel håller Minecraft-data och säkerhetskopior utanför den oföränderliga operativsystemsinstallationen. Du väljer lagring för dem var för sig.

Följande lagringsalternativ stöds:

- systemets filsystem
- dedikerade interna diskar
- USB-lagring
- befintliga lokala partitioner eller filsystem
- nya partitioner som skapas i redan oallokerat utrymme
- NFS
- SMB/CIFS

Varje server kan använda både automatiska och manuella säkerhetskopior. Återställning ersätter alla beständiga data för den valda servern och bevarar serverns identitet i JustVoxel och systemets konfiguration.

JustVoxel har också granskade flöden för **export, import och återställning** när du vill flytta servrar mellan maskiner eller i samband med en ominstallation.

[Lagring](../docs/STORAGE.md) · [Återställning](../docs/RESTORE.md) · [Migrering](../docs/MIGRATION.md)

## 🛡️ Utformat för säkrare ändringar

JustVoxels webbgränssnitt erbjuder avgränsade systemåtgärder. Det kan inte köra godtyckliga skalkommandon.

Åtgärder som stöds omfattas av säkerhetsregler på systemnivå, till exempel:

- kontroll av anslutna spelare före störande åtgärder
- granskning av ändringar innan du väljer Tillämpa
- säkerhetskopiering med servern stoppad när en ändring kan behöva återställas
- kontroll av den exakta serveridentiteten så att servrar inte förväxlas
- verifiering av omstart och drift
- beständig status för åtgärder och återställning
- möjlighet att ångra åtgärder som stöds
- kontrollpunkter i NetworkManager för riskfyllda nätverksändringar
- separata uppdateringsflöden för Minecraft och operativsystemet

Webbgränssnittet körs utan förhöjda behörigheter. Ändringar som kräver sådana behörigheter går via den lokala JustVoxel Management Agent och dess API för avgränsade åtgärder. Webbläsaren får ingen tillgång till ett root-skal.

## 🔄 Separata uppdateringar för systemet och Minecraft

JustVoxel bygger på Home Server Base 10 / AlmaLinux 10 och hanteras via systemavbildningar.

Operativsystemet uppdateras separat från Minecraft-versioner, serverprogramvara, insticksprogram, moddar och containeruppdateringar.

Du kan alltså uppdatera JustVoxel utan att Minecraft-versionen ändras i bakgrunden och ändra en Minecraft-server utan att bygga om värdsystemets operativsystem.

[Dokumentation om systemhantering](../docs/SYSTEM.md)

## 📊 Hur står sig JustVoxel i jämförelse?

En traditionell kontrollpanel för Minecraft är ett separat program. I JustVoxel ingår **operativsystemet och Minecraft-hanteringen i samma system**.

| Funktion | **JustVoxel** | **MineOS SvelteKit** | **Crafty Controller** | **MineOS Node** |
| --- | :---: | :---: | :---: | :---: |
| Komplett operativsystem för Minecraft-servrar | ⭐ | — | — | — |
| Flera Minecraft-servrar | ✅ | ✅ | ✅ | ✅ |
| Stöd för Java- och Bedrock-servrar | ✅ | ✅ | ✅ | 🟡 Java och äldre stöd för PocketMine |
| Hjälp att hitta insticksprogram och moddar | ✅ Modrinth, Hangar, CurseForge | ✅ Modrinth, CurseForge | — | — |
| Hantering av värdens nätverk, lagring och operativsystem | ⭐ | — | — | — |
| Integrerad privat åtkomst och tunnlar för speltrafik | ⭐ 4 alternativ | — | — | — |
| MCP- och REST-gateway för AI och automatisering | ⭐ | — | — | — |
| Inbyggd hantering via Telegram och Discord | ⭐ | — | — | — |
| Lock- och skärminställningar för bärbara datorer | ✅ HWS | — | — | — |
| Uppdatering av oföränderligt operativsystem via systemavbildningar | ✅ | — | — | — |
| Administrationsgränssnitt i terminalen | ✅ mjust | ✅ mineos CLI/TUI | — | ✅ CLI |
| Installationsmodell | Komplett system med bootc | Docker-/Compose-program | Kontrollpanel i Docker, på Linux eller på Windows | Node.js-program på Linux/BSD |

⭐ utmärkande funktion &nbsp;•&nbsp; ✅ stöds &nbsp;•&nbsp; 🟡 delvis eller äldre stöd &nbsp;•&nbsp; — ingår inte i den centrala projektdokumentation som granskats

**Jämförelsen granskades senast den 16 november 2026.** Syftet är att visa skillnaderna mellan projekten utan att döma ut de andra alternativen. MineOS och Crafty är program för Minecraft-hantering som kan passa utmärkt om du redan har ett operativsystem som du vill administrera själv. JustVoxel vänder sig till dig som vill hantera kontrollpanelen för Minecraft **och det underliggande serversystemet** som en och samma produkt.

Jämförelsen bygger på projektens aktuella dokumentation och kodarkiv:

- [MineOS SvelteKit](https://github.com/freeman412/mineos-sveltekit)
- [Crafty Controller](https://craftycontrol.com/) · [källkod](https://gitlab.com/crafty-controller/crafty-4)
- [MineOS Node](https://github.com/hexparrot/mineos-node)

Om något i jämförelsen är inaktuellt eller missvisande får du gärna öppna ett ärende. Vi välkomnar också rättelser som är till JustVoxels nackdel.

## 📚 Dokumentation

De utförliga guiderna för JustVoxel finns samlade i dokumentationskatalogen. [Dokumentationskatalog](../docs/)

Här kan du börja:

- [Webbgränssnitt](../docs/WEBUI.md)
- [Minecrafts körmiljö](../docs/MINECRAFT_RUNTIME.md)
- [Serverprogramvara](../docs/server-software.md)
- [Insticksprogram och moddar](../docs/plugin-workspace.md)
- [Minecraft-nätverk](../docs/minecraft-network.md)
- [Nätverk och fjärråtkomst](../docs/NETWORKING.md)
- [AI- och API-anslutningar](../docs/AI-API-CONNECTIONS.md)
- [Telegram](../docs/TELEGRAM.md)
- [Discord](../docs/DISCORD.md)
- [Lagring](../docs/STORAGE.md)
- [Återställning](../docs/RESTORE.md)
- [Migrering](../docs/MIGRATION.md)
- [Systemhantering](../docs/SYSTEM.md)
- [mjust](../docs/MJUST.md)

Utvecklarguiden beskriver hur du utvecklar och testar systemavbildningar samt hur utgivningen går till. [Utvecklarguide](../README-DEV.md)

## ⛏️ Minecraft-programvara och licensavtal

JustVoxel innehåller **inga** binärfiler för Minecraft-servrar, ingen serverprogramvara från Mojang, ingen förskapad värld och inget på förhand godkänt slutanvändarlicensavtal för Minecraft (EULA).

När en server skapas godkänner administratören Minecrafts slutanvändarlicensavtal. JustVoxel förbereder den valda körmiljön för Minecraft-servern, förutsatt att den stöds.

Minecraft, Mojangs programvara, Paper, Purpur, Spigot, Pufferfish, Leaf, Fabric, NeoForge, Forge, Quilt, insticksprogram, moddar och övriga tredjepartskomponenter omfattas av sina respektive licenser och distributionsvillkor.

## 📄 Licens

JustVoxel licensieras under **Apache-2.0**. [LICENSE](../LICENSE)

Licensmeddelanden för tredjepartskomponenter finns i [THIRD_PARTY_NOTICES.md](../THIRD_PARTY_NOTICES.md).
