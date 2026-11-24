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

> **Over deze vertaling:** Deze vertaling van JustVoxel is met hulp van AI gemaakt op basis van de oorspronkelijke Engelse tekst. Vind je een onjuiste, onduidelijke of onnatuurlijke formulering? Stel dan een correctie voor via een [probleemmelding op GitHub](https://github.com/home-server-project/justvoxel/issues) of een [wijzigingsvoorstel (pull request)](https://github.com/home-server-project/justvoxel/pulls).

<p align="center">
  <strong>Jouw Minecraft-servers op eigen hardware, een gehuurde dedicated server of een gehoste VM. Alles vanuit één beheercentrum.</strong>
</p>

<p align="center">
  Een appliance waarmee je zelf Minecraft-servers host en beheert zonder het onderliggende besturingssysteem te hoeven beheren.
</p>

<p align="center">
  <a href="#-justvoxel-installeren">Installeren</a> •
  <a href="#-eenvoudiger-minecraft-hosten">Functies</a> •
  <a href="../docs/INSTALLER.md">Documentatie</a>
</p>

<p align="center">
  <img src="../docs/images/justvoxel-overview-1.webp" width="100%" alt="Overzicht van het JustVoxel-beheercentrum"><img src="../docs/images/justvoxel-overview-2.webp" width="100%" alt=""><img src="../docs/images/justvoxel-overview-3.webp" width="100%" alt="">
</p>

JustVoxel is een appliance die speciaal is gemaakt om zelf Minecraft-servers te hosten. Installeer JustVoxel op een compatibele pc, laptop, fysieke server of virtuele machine, of bij een hostingdienst die een eigen image van het besturingssysteem toestaat. Daarna beheer je Minecraft vanuit je browser.

Je hoeft geen verzameling scripts te onderhouden of een apart Minecraft-beheerpaneel te installeren.

Heb je nog een oude laptop of pc liggen? **Met JustVoxel HWS maak je er een speciale Minecraft-appliance van en geef je bruikbare hardware een tweede leven.** Op ondersteunde laptops kan JustVoxel de slaapstand bij het sluiten van de klep negeren, zodat de server blijft draaien als je de laptop dichtklapt. Via de lokale console kun je het scherm direct of na een korte vertraging uitschakelen.

> 🚧 **Pre-release:** JustVoxel wordt voorbereid op de eerste openbare release. De huidige installatie-ISO is beschikbaar om te testen en alvast te gebruiken terwijl de laatste releasecontroles nog lopen.

## ✨ Eenvoudiger Minecraft hosten

JustVoxel combineert Minecraft-beheer met het onderliggende besturingssysteem.

- 🎮 **Meerdere Minecraft-servers** — draai verschillende werelden en servertypen onafhankelijk van elkaar op één appliance.
- 🖥️ **Online beheercentrum** — beheer servers vanaf een computer, tablet of telefoon.
- 🧩 **Plug-ins en mods** — ontdek en beheer compatibele uitbreidingen van ondersteunde aanbieders.
- 💾 **Back-ups en herstel** — bescherm elke server met geplande en handmatige back-ups.
- 🌐 **Toegang op afstand** — kies uit vier geïntegreerde opties voor privétoegang en gametunnels.
- 🤖 **AI en automatisering** — koppel compatibele AI-assistenten en applicaties via MCP en REST.
- 💬 **Telegram en Discord** — bedien je server via een bot, ontvang meldingen en regel zaken voor je community. Deze functies zijn optioneel.
- 🛡️ **Systeembeheer** — beheer netwerk, opslag, updates, systeemstatus, logboeken en ondersteunde stroomfuncties zonder Linux-commando’s te hoeven leren.

JustVoxel is bedoeld voor gezinnen, vrienden, kleine Minecraft-community’s en mensen die hun eigen server willen hebben zonder een traditioneel Minecraft-hostingpaneel te huren.

## 🖥️ Eén appliance, twee edities

### JustVoxel VM

Voor virtuele machines en gehoste omgevingen zoals:

- Proxmox
- KVM / libvirt
- VMware
- Hyper-V
- VirtualBox
- gehoste VM’s of VPS-diensten die een compatibele, eigen image van het besturingssysteem toestaan

De VM-editie bevat de volledige Minecraft-appliance zonder de pakketten voor fysieke hardware die in een VM overbodig zijn.

### JustVoxel HWS

Voor installatie rechtstreeks op fysieke hardware, zoals:

- een oude laptop
- een desktop die je een nieuwe bestemming geeft
- een mini-pc
- een thuisserver
- een gehuurde bare-metalserver waarop eigen installatiemedia zijn toegestaan

HWS voegt ondersteuning voor fysieke hardware toe, waaronder UPS/NUT-integratie, SMART- en NVMe-tools, sensoren, firmware- en UEFI-instellingen, apparaatfirmware, hardwarediagnostiek en bediening van het laptopscherm en de klep.

Beide edities gebruiken hetzelfde JustVoxel-beheercentrum en dezelfde Management API. Ook de Minecraft-beheerfuncties, het opslagmodel, back-ups, netwerkbeheer en de manier waarop je updates uitvoert zijn gelijk.

[Lees welke functies VM en HWS ondersteunen](../docs/VARIANTS.md)

## 🚀 JustVoxel installeren

Eén installatie-ISO ondersteunt beide JustVoxel-edities.

[⬇️ Download de JustVoxel x86_64-ISO](https://sourceforge.net/projects/justvoxel/files/JustVoxel-x86_64.iso/download)

Kies de editie die past bij de omgeving waarin je JustVoxel installeert:

- **JustVoxel VM** voor virtuele machines en compatibele gehoste VM-omgevingen
- **JustVoxel HWS** voor pc’s, laptops, mini-pc’s, fysieke thuisservers en compatibele bare-metalhosting

De [installatiehandleiding](../docs/INSTALLER.md) beschrijft de installatie, de schijfvoorbereiding en de eerste keer inloggen.

## 🏠 Minecraft staat voorop, met ruimte voor andere toepassingen

JustVoxel is in de eerste plaats gebouwd en ondersteund als Minecraft-serverappliance. Het onderliggende systeem is een gewone **AlmaLinux 10 bootc-server met Podman, Quadlet, systemd, SSH, NetworkManager en standaard Linux-beheertools**.

Gevorderde gebruikers kunnen dezelfde machine ook als bredere thuisserver gebruiken en naast Minecraft extra containers draaien.

Je kunt bijvoorbeeld zelf de volgende diensten installeren:

- **Jellyfin** of **Plex** voor media
- **Vaultwarden** voor wachtwoordbeheer
- **Immich** voor foto- en videobeheer
- **Home Assistant Container** voor domotica
- **Nextcloud** voor bestanden en samenwerking
- **Syncthing** of **Navidrome**
- **OpenClaw** als zelfgehoste AI-agent en gateway

Deze applicaties vallen **buiten het beheer van het JustVoxel-beheercentrum**. De beheerder blijft verantwoordelijk voor hun Podman-containers, Quadlets, opslag, poorten, updates, back-ups, reverseproxyconfiguratie en toepassingsspecifieke vereisten.

JustVoxel houdt die taakverdeling bewust eenvoudig. Het beheert de Minecraft-appliance. Gevorderde gebruikers kunnen voor andere toepassingen de gebruikelijke werkwijze met Podman en Quadlet volgen.

Ook bij onderhoud blijven andere containers onder beheer van de beheerder en vallen ze buiten de Minecraft-resources. Een Minecraft-bewerking zoals **Opnieuw beginnen** verwijdert geen andere Podman-containers.

Zo kan een JustVoxel-machine in de eerste plaats een Minecraft-appliance blijven en desgewenst ook dienstdoen als gewone thuisserver met containers.

## 🎮 Maak de Minecraft-server die bij je past

JustVoxel kan meerdere onafhankelijke Minecraft-servers tegelijk draaien als de machine genoeg geheugen en vrije poorten heeft.

Elke server heeft een eigen identiteit, configuratie, runtime, poorten, gegevens, back-ups, bediening voor starten en stoppen, en status.

### Huidige serversoftware

| Serversoftware | Uitbreidingen | Beheerde Java → Bedrock-crossplay |
| --- | --- | --- |
| **Paper** | Plug-ins | ✅ |
| **Purpur** | Plug-ins | ✅ |
| **Spigot** | Plug-ins | ✅ |
| **Pufferfish** | Plug-ins | ✅ |
| **Leaf** | Plug-ins | ✅ |
| **Fabric** | Mods | ✅ Als de exacte compatibiliteit is aangetoond |
| **NeoForge** | Mods | ✅ Als de exacte compatibiliteit is aangetoond |
| **Forge** | Mods | Alleen Java |
| **Quilt** | Mods | Alleen Java |
| **Vanilla Java** | Geen plug-ins of mods | Alleen Java |
| **Native Bedrock** | Native Bedrock-server | Native Bedrock-clients |

Kies **Eenvoudige configuratie** als je begeleiding wilt bij het aanmaken van je eerste server, of **Geavanceerde configuratie** als je meer wilt bepalen over geheugen, poorten, opslag, back-ups, versiebeleid en ondersteunde crossplay.

JustVoxel controleert per functie of de gekozen serversoftware die ondersteunt.

[Bekijk de serversoftware en compatibiliteitsdetails](../docs/server-software.md)

## 🧩 Plug-ins en mods

Open **Minecraft → Plug-ins en mods** voor de geselecteerde server.

Afhankelijk van het servertype kan JustVoxel de volgende aanbieders gebruiken:

- [Modrinth](https://modrinth.com/)
- [Hangar](https://hangar.papermc.io/)
- [CurseForge](https://www.curseforge.com/minecraft) met de API-sleutel die je hebt ingesteld

Bij ondersteunde installaties controleert JustVoxel de compatibiliteit en de downloads, maakt het back-ups terwijl de server uitstaat, controleert het of de server na een herstart werkt, houdt het bij welke bestanden het beheert en biedt het herstelmogelijkheden.

Servers uit de Paper-familie kunnen plug-incatalogi gebruiken. Fabric, Forge, NeoForge en Quilt gebruiken modcatalogi waarbij JustVoxel de exacte modloader en Minecraft-versie controleert. Vanilla Java en native Bedrock bieden bewust geen catalogus voor plug-ins of mods.

[Meer over plug-ins en mods](../docs/plugin-workspace.md)

## 🌐 Optioneel Minecraft-netwerk

Je kunt prima meerdere zelfstandige servers draaien. Daarvoor heb je **geen** proxy nodig.

Wil je één openbaar adres voor meerdere compatibele servers? JustVoxel biedt ook een optioneel **Minecraft-netwerk** op basis van Velocity.

In het beheercentrum kun je ondersteunde achterliggende servers selecteren, openbare poorten voor Java en Bedrock instellen, de virtuele lobby en het welkomstbericht beheren en bepalen welke servers aan het netwerk deelnemen.

Paper, Purpur, Pufferfish en Leaf worden ondersteund als achterliggende servers in het netwerk. Compatibele Fabric-, Quilt-, Forge- en NeoForge-servers kunnen ook deelnemen als JustVoxel kan vaststellen dat ze aan de compatibiliteitseisen van de proxy voldoen. Spigot, Vanilla en native Bedrock blijven zelfstandig.

[Meer over het Minecraft-netwerk](../docs/minecraft-network.md)

## 🖥️ Maak kennis met het beheercentrum

In JustVoxel staan de taken die je op je server wilt uitvoeren centraal. Kennis van de interne werking van Linux is daarvoor niet nodig.

### Minecraft-servers

Maak en beheer meerdere servers, start of stop ze, bekijk hun status en spelers, pas ondersteunde instellingen aan en gebruik de tools per server.

### Plug-ins en mods

Ontdek compatibele uitbreidingen, bekijk geïnstalleerde plug-ins en mods, beheer ondersteunde uitbreidingen en raadpleeg herstelinformatie als een wijziging aandacht nodig heeft.

### Back-ups en opslag

Maak back-ups, bekijk de back-upgeschiedenis, herstel alle gegevens van een server, stel back-uplocaties in, importeer en exporteer servers en beheer ondersteunde opslag.

### Netwerk

Beheer ethernet, wifi en toegang op afstand en **los verbindingsproblemen op**. Netwerkwijzigingen die de verbinding kunnen verbreken, kunnen worden teruggedraaid. Zo hoef je bij een verkeerde IP- of wifi-instelling de toegang niet voorgoed te verliezen.

### Systeem

Bekijk de systeemstatus, het resourcegebruik, logboeken, updates van het besturingssysteem en instellingen voor gebruikers en beveiliging. Voer ondersteunde acties uit zoals herstarten of uitschakelen, en bedien op HWS de hardwarefuncties.

### AI- en API-verbindingen

Koppel ondersteunde AI-applicaties, MCP-clients, REST-automatisering, Telegram en Discord zonder een apart beheersysteem op te zetten.

Werk je liever in een terminal? JustVoxel bevat ook **mjust**, een interactieve beheerinterface voor de terminal met dezelfde veiligheidsregels.

[WebUI-documentatie](../docs/WEBUI.md) · [mjust-handleiding](../docs/MJUST.md)

## 🌐 Laat vrienden verbinden op jouw manier

JustVoxel biedt momenteel vier geïntegreerde opties onder **Netwerk → Toegang op afstand**.

Deze opties zijn rechtstreeks vanuit JustVoxel te beheren. Waar van toepassing kun je de status bekijken, de dienst in- of uitschakelen en de configuratie doorlopen.

| Optie | Geschikt voor | Wat je krijgt |
| --- | --- | --- |
| **[Tailscale](https://tailscale.com/)** | Privéservers voor vrienden en familie | Een privémeshnetwerk tussen goedgekeurde apparaten |
| **[NetBird](https://netbird.io/)** | Privénetwerken met beheerde toegang | Een privénetwerk met controle over accounts en toegangsbeleid |
| **[Playit.gg](https://playit.gg/)** | Openbare toegang tot Minecraft zonder poortdoorschakeling op de router | Openbare gametunnels en verbindingsadressen |
| **[NordVPN Meshnet](https://meshnet.nordvpn.com/)** | Privéverbindingen tussen apparaten van NordVPN-gebruikers | Meshnet-verbindingen zonder JustVoxel als commerciële VPN-client te gebruiken |

### Welke optie past bij mij?

**🎮 Wil je dat vrienden kunnen spelen zonder een privé-VPN-client te installeren?**  
Kijk eerst naar **Playit.gg**. Spelers kunnen het openbare adres van je ingestelde gametunnel gebruiken.

**🔒 Wil je een privéserver voor vertrouwde apparaten?**  
Overweeg **Tailscale** of **NetBird**.

**🌍 Gebruik je al NordVPN Meshnet?**  
JustVoxel kan de Meshnet-integratie beheren. Dat staat los van een verbinding tussen de appliance en een reguliere commerciële NordVPN-uitgangsserver.

Accounts, cloudbeleid en tunnelconfiguratie regel je bij de aanbieder. JustVoxel verzorgt de ondersteunde integratie op de appliance.

Je kunt meerdere VPN- of meshproducten tegelijk gebruiken, maar routering, DNS of firewallregels kunnen elkaar in de weg zitten.

[Lees de handleiding voor netwerk en toegang op afstand](../docs/NETWORKING.md)

## 🤖 AI, MCP, REST en automatisering

Wil je een AI-assistent naar de serverstatus vragen, domotica koppelen of je eigen beheertaken automatiseren?

JustVoxel biedt ingebouwde **MCP- en REST-interfaces** via **AI- en API-verbindingen**.

Bij het instellen kun je kiezen uit compatibele applicaties en verbindingstypen, waaronder:

- **OpenAI — ChatGPT en Codex**
- **Claude**
- **Grok**
- **Gemini**
- **GitHub Copilot**
- **Andere AI- of MCP-app**
- **Automatisering / REST API**

Je bepaalt zelf waartoe elke verbinding toegang krijgt.

De rechten lopen van **Niveau 1 — Alleen lezen** via serverbeheer tot geavanceerd systeembeheer, waarvoor uitdrukkelijke toestemming nodig is. Je kunt verbindingen ook beperken tot geselecteerde Minecraft-servers.

JustVoxel heeft geen API-sleutel voor een AI-model nodig om de MCP- of REST-interface beschikbaar te stellen. Of een client de verbinding kan gebruiken, hangt af van de applicatie, de mogelijkheden van het account of abonnement en de netwerktoegang.

De AI/API-gateway staat standaard uit totdat je die inschakelt.

[Meer over AI- en API-verbindingen](../docs/AI-API-CONNECTIONS.md)

## 💬 Telegram en Discord

JustVoxel biedt ingebouwde, optionele integraties voor Telegram en Discord die dezelfde centrale beheerbackend gebruiken als de WebUI.

### Telegram

Met een aparte Telegram-bot kun je:

- de status van de appliance en Minecraft bekijken
- een server selecteren en spelersinformatie bekijken
- servers starten, stoppen en herstarten als je daarvoor beheertoegang hebt
- handmatige back-ups maken
- updates voor Minecraft en containers bekijken
- informatie over updates van het besturingssysteem bekijken
- de appliance gecontroleerd herstarten
- optionele meldingen ontvangen over serverstatus, back-upresultaten en beschikbare updates

Telegram gebruikt eigen rechten voor **Alleen lezen** en **Serverbeheer** en heeft geen AI-model nodig.

[Telegram instellen en rechten toekennen](../docs/TELEGRAM.md)

### Discord

Discord kan zowel je Minecraft-community als de beheerders toegang geven tot JustVoxel-functies.

Leden van de expliciet gekoppelde Discord-server kunnen de basisstatus van de Minecraft-server bekijken en toegang tot Minecraft aanvragen, zonder automatisch beheerrechten te krijgen.

Goedgekeurde Discord-beheerders van JustVoxel kunnen ondersteunde acties uitvoeren, zoals servers starten, stoppen of herstarten, handmatig back-ups maken, updates beoordelen, het Minecraft-netwerk beheren en andere geavanceerde acties uitvoeren.

Discord-lidmaatschap, toegang via de Minecraft-toelatingslijst en JustVoxel-beheerrechten staan los van elkaar.

[Discord instellen, communitytoegang en beheer](../docs/DISCORD.md)

## 💾 Back-ups, herstel, opslag en migratie

Je Minecraft-gegevens zijn belangrijker dan de image van de appliance.

JustVoxel bewaart Minecraft-gegevens en back-ups buiten de onveranderlijke installatie van het besturingssysteem. Je kiest voor beide apart waar je ze opslaat.

Ondersteunde opslagmogelijkheden zijn:

- het systeembestandssysteem
- aparte interne schijven
- USB-opslag
- bestaande lokale partities of bestandssystemen
- nieuwe partities in ruimte die nog niet was toegewezen
- NFS
- SMB/CIFS

Elke server ondersteunt automatische en handmatige back-ups. Bij herstel worden alle blijvend opgeslagen gegevens van de geselecteerde server vervangen, met behoud van de JustVoxel-identiteit en de applianceconfiguratie van die server.

Wil je servers naar een andere machine verplaatsen of meenemen bij een herinstallatie? JustVoxel biedt daarvoor gecontroleerde procedures voor **exporteren, importeren en herstellen**.

[Opslag](../docs/STORAGE.md) · [Herstel](../docs/RESTORE.md) · [Migratie](../docs/MIGRATION.md)

## 🛡️ Ontworpen voor veiligere wijzigingen

Achter de webinterface voert JustVoxel afgebakende beheertaken uit. De interface kan geen willekeurige shellcommando’s uitvoeren.

Voor ondersteunde bewerkingen gelden veiligheidsregels voor de hele appliance, zoals:

- controleren of er spelers online zijn voordat een actie het spel kan onderbreken
- wijzigingen nakijken voordat je op **Toepassen** klikt
- back-ups maken met de server uit als herstel nodig kan zijn
- de exacte serveridentiteit bewaken, zodat servers niet per ongeluk worden verwisseld
- controleren of de server na een herstart goed draait
- de status van bewerkingen en herstel blijvend opslaan
- ondersteunde bewerkingen kunnen terugdraaien
- NetworkManager-controlepunten gebruiken voor riskante netwerkwijzigingen
- Minecraft en het besturingssysteem afzonderlijk bijwerken

De WebUI die je vanuit de browser gebruikt, draait zonder verhoogde rechten. Wijzigingen die verhoogde rechten vereisen gaan via de lokale JustVoxel Management Agent en zijn expliciete API. De browser krijgt geen toegang tot een root-shell.

## 🔄 Systeemupdates staan los van Minecraft-updates

JustVoxel is een appliance op basis van Home Server Base 10 / AlmaLinux 10 en wordt beheerd via systeemimages.

Updates van het besturingssysteem staan los van Minecraft-versies, serversoftware, plug-ins, mods en containerupdates.

Bij een update van de appliance verandert je Minecraft-versie dus niet ongemerkt. Een wijziging aan een Minecraft-server vereist geen nieuwe build van het besturingssysteem van de host.

[Documentatie voor systeembeheer](../docs/SYSTEM.md)

## 📊 Hoe verhoudt JustVoxel zich tot andere projecten?

JustVoxel verschilt van een traditioneel Minecraft-beheerpaneel doordat **het besturingssysteem en de Minecraft-beheerlaag samen één appliance vormen**.

| Functie | **JustVoxel** | **MineOS SvelteKit** | **Crafty Controller** | **MineOS Node** |
| --- | :---: | :---: | :---: | :---: |
| Compleet besturingssysteem voor een Minecraft-appliance | ⭐ | — | — | — |
| Meerdere Minecraft-servers | ✅ | ✅ | ✅ | ✅ |
| Ondersteuning voor Java- en Bedrock-servers | ✅ | ✅ | ✅ | 🟡 Java en oudere PocketMine-ondersteuning |
| Hulp bij het vinden van plug-ins en mods | ✅ Modrinth, Hangar, CurseForge | ✅ Modrinth, CurseForge | — | — |
| Beheer van hostnetwerk, opslag en besturingssysteem | ⭐ | — | — | — |
| Geïntegreerde privétoegang en gametunnels | ⭐ 4 opties | — | — | — |
| AI- en automatiseringsgateway met MCP en REST | ⭐ | — | — | — |
| Ingebouwd beheer via Telegram en Discord | ⭐ | — | — | — |
| Bediening van laptopklep en -scherm | ✅ HWS | — | — | — |
| Updates van het onveranderlijke besturingssysteem via images | ✅ | — | — | — |
| Beheerinterface voor de terminal | ✅ mjust | ✅ mineos CLI/TUI | — | ✅ CLI |
| Installatiemodel | Complete bootc-appliance | Applicatie voor Docker / Compose | Beheerpaneel op Docker, Linux of Windows | Node.js-applicatie op Linux/BSD |

⭐ onderscheidend &nbsp;•&nbsp; ✅ ondersteund &nbsp;•&nbsp; 🟡 gedeeltelijke ondersteuning of ondersteuning voor oudere software &nbsp;•&nbsp; — komt niet voor in de geraadpleegde hoofddocumentatie van het project

**Vergelijking voor het laatst gecontroleerd op 16 november 2026.** Deze vergelijking is niet bedoeld om andere projecten als slecht af te schilderen. MineOS en Crafty zijn Minecraft-beheerapplicaties die uitstekend kunnen passen als je al een besturingssysteem hebt dat je zelf wilt beheren. JustVoxel is bedoeld voor gebruikers die het Minecraft-paneel **én de onderliggende serverappliance** als één product willen beheren.

Voor deze vergelijking zijn de huidige projectdocumentatie en repositories gebruikt:

- [MineOS SvelteKit](https://github.com/freeman412/mineos-sveltekit)
- [Crafty Controller](https://craftycontrol.com/) · [broncode](https://gitlab.com/crafty-controller/crafty-4)
- [MineOS Node](https://github.com/hexparrot/mineos-node)

Klopt iets in de vergelijking niet meer of vind je een vermelding oneerlijk? Open dan een issue. Ook correcties die minder gunstig uitpakken voor JustVoxel zijn welkom.

## 📚 Documentatie

De uitgebreide handleidingen voor de appliance staan in de [documentatiemap](../docs/).

Hier kun je beginnen:

- [WebUI](../docs/WEBUI.md)
- [Minecraft-runtime](../docs/MINECRAFT_RUNTIME.md)
- [Serversoftware](../docs/server-software.md)
- [Plug-ins en mods](../docs/plugin-workspace.md)
- [Minecraft-netwerk](../docs/minecraft-network.md)
- [Netwerk en toegang op afstand](../docs/NETWORKING.md)
- [AI- en API-verbindingen](../docs/AI-API-CONNECTIONS.md)
- [Telegram](../docs/TELEGRAM.md)
- [Discord](../docs/DISCORD.md)
- [Opslag](../docs/STORAGE.md)
- [Herstel](../docs/RESTORE.md)
- [Migratie](../docs/MIGRATION.md)
- [Systeembeheer](../docs/SYSTEM.md)
- [mjust](../docs/MJUST.md)

Zie de [ontwikkelaarshandleiding](../README-DEV.md) voor het ontwikkelen en testen van images en het uitbrengen van releases.

## ⛏️ Minecraft-software en EULA

JustVoxel levert **geen** uitvoerbare Minecraft-serverbestanden, Mojang-serversoftware of vooraf aangemaakte wereld mee. De Minecraft-EULA is ook niet vooraf geaccepteerd.

Bij het aanmaken van een server accepteert de beheerder de Minecraft-EULA en bereidt JustVoxel de geselecteerde ondersteunde Minecraft-serverruntime voor.

Voor Minecraft, Mojang-software, Paper, Purpur, Spigot, Pufferfish, Leaf, Fabric, NeoForge, Forge, Quilt, plug-ins, mods en andere externe onderdelen blijven hun eigen licenties en distributievoorwaarden gelden.

## 📄 Licentie

JustVoxel valt onder de **Apache-2.0**-licentie. Zie [LICENSE](../LICENSE).

Juridische vermeldingen voor onderdelen van derden staan in [THIRD_PARTY_NOTICES.md](../THIRD_PARTY_NOTICES.md).
