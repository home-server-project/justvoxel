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

> **Om oversættelsen:** Denne oversættelse af JustVoxel er lavet med hjælp fra AI ud fra den engelske originaltekst. Hvis du finder en upræcis, uklar eller unaturlig formulering, kan du foreslå en rettelse ved at oprette en [sag på GitHub](https://github.com/home-server-project/justvoxel/issues) eller indsende et [ændringsforslag (pull request)](https://github.com/home-server-project/justvoxel/pulls).

<p align="center">
  <strong>Dine Minecraft-servere. Din egen hardware, en lejet dedikeret server eller en hostet VM. Ét kontrolcenter.</strong>
</p>

<p align="center">
  Et komplet system til dig, der vil drive dine egne Minecraft-servere uden at skulle administrere operativsystemet.
</p>

<p align="center">
  <a href="#-installer-justvoxel">Installation</a> •
  <a href="#-nemmere-drift-af-minecraft-servere">Funktioner</a> •
  <a href="../docs/INSTALLER.md">Dokumentation</a>
</p>

<p align="center">
  <img src="../docs/images/justvoxel-overview-1.webp" width="100%" alt="Oversigt over JustVoxels kontrolcenter"><img src="../docs/images/justvoxel-overview-2.webp" width="100%" alt=""><img src="../docs/images/justvoxel-overview-3.webp" width="100%" alt="">
</p>

JustVoxel er et dedikeret system til at hoste dine egne Minecraft-servere. Installer det på en kompatibel pc, bærbar computer, fysisk server eller virtuel maskine, eller brug en hostingtjeneste, hvor du kan køre dit eget operativsystemimage. Derefter administrerer du Minecraft fra en browser.

Ingen scriptsamling at vedligeholde. Intet separat Minecraft-kontrolpanel at installere.

Har du en gammel bærbar eller stationær pc? **Med JustVoxel HWS kan du bruge den som dedikeret Minecraft-server og få nytte af hardware, der ellers ligger i skuffen.** På understøttede bærbare kan du indstille maskinen til at blive ved med at køre, når du lukker låget. Du kan også slukke skærmen fra den lokale konsol, enten straks eller efter en kort forsinkelse.

> 🚧 **Forhåndsudgave:** JustVoxel er ved at blive gjort klar til sin første offentlige udgivelse. Den nuværende installations-ISO er tilgængelig til test og tidlig brug, mens de sidste kontroller inden udgivelsen er i gang.

## ✨ Nemmere drift af Minecraft-servere

JustVoxel samler administration af Minecraft og det underliggende operativsystem i ét system.

- 🎮 **Flere Minecraft-servere** – kør forskellige verdener og servertyper uafhængigt af hinanden på samme maskine.
- 🖥️ **Kontrolcenter i browseren** – administrer servere fra en computer, tablet eller telefon.
- 🧩 **Plugins og mods** – find og administrer kompatibelt indhold fra understøttede udbydere.
- 💾 **Sikkerhedskopiering og gendannelse** – beskyt hver server med planlagte og manuelle sikkerhedskopier.
- 🌐 **Ekstern adgang** – vælg mellem fire integrerede løsninger til privat adgang og spiltunneler.
- 🤖 **AI og automatisering** – tilslut kompatible AI-assistenter og programmer via MCP og REST.
- 💬 **Telegram og Discord** – brug botkommandoer, notifikationer og funktioner til fællesskabet efter behov.
- 🛡️ **Systemadministration** – administrer netværk, lager, opdateringer, systemtilstand, logfiler og understøttede strømfunktioner uden at lære de bagvedliggende Linux-kommandoer.

JustVoxel er udviklet til familier, venner, mindre Minecraft-fællesskaber og andre, der hellere vil eje deres egen server end leje sig ind på en traditionel Minecraft-hostingtjeneste med kontrolpanel.

## 🖥️ Ét serversystem, to udgaver

### JustVoxel VM

Til virtuelle maskiner og hostede miljøer, f.eks.

- Proxmox
- KVM / libvirt
- VMware
- Hyper-V
- VirtualBox
- hostede VM'er eller VPS-tjenester, der understøtter et kompatibelt, brugerdefineret operativsystemimage

VM-udgaven indeholder det komplette Minecraft-serversystem og udelader pakker til fysisk hardware, som ikke er relevante i en VM.

### JustVoxel HWS

Til installation direkte på fysisk hardware, f.eks.

- en gammel bærbar computer
- en stationær computer, der får nyt liv som server
- en mini-pc
- en hjemmeserver
- en lejet bare metal-server, hvor du må bruge dit eget installationsmedie

HWS tilføjer funktioner til fysiske maskiner, herunder UPS/NUT-integration, SMART- og NVMe-værktøjer, sensorer, firmware- og UEFI-indstillinger, enhedsfirmware, hardwarediagnostik og indstillinger for skærm og låg på bærbare computere.

Begge udgaver bruger samme JustVoxel-kontrolcenter, Management API, arbejdsgange til Minecraft, lagermodel, sikkerhedskopiering, netværksfunktioner og opdateringsproces.

[Se forskellene mellem VM og HWS](../docs/VARIANTS.md)

## 🚀 Installer JustVoxel

Begge JustVoxel-udgaver bruger samme installations-ISO.

[⬇️ Download JustVoxel x86_64 ISO](https://sourceforge.net/projects/justvoxel/files/JustVoxel-x86_64.iso/download)

Vælg den udgave, der passer til det miljø, du installerer i.

- **JustVoxel VM** til virtuelle maskiner og kompatible hostede VM-miljøer
- **JustVoxel HWS** til pc'er, bærbare computere, mini-pc'er, fysiske hjemmeservere og kompatibel bare metal-hosting

Se [installationsvejledningen](../docs/INSTALLER.md) for at få hjælp til installation, klargøring af diske og første login.

## 🏠 Udviklet til Minecraft med plads til mere

JustVoxel er først og fremmest udviklet og understøttet som et Minecraft-serversystem. Under overfladen kører en almindelig **AlmaLinux 10 bootc-server med Podman, Quadlet, systemd, SSH, NetworkManager og almindelige Linux-administrationsværktøjer**.

Erfarne brugere kan bruge samme maskine som en mere alsidig hjemmeserver og køre ekstra containere ved siden af Minecraft.

Du kan f.eks. installere disse tjenester manuelt.

- **Jellyfin** eller **Plex** til medier
- **Vaultwarden** til administration af adgangskoder
- **Immich** til organisering af billeder og videoer
- **Home Assistant Container** til hjemmeautomatisering
- **Nextcloud** til filer og samarbejde
- **Syncthing** eller **Navidrome**
- **OpenClaw** som selvhostet AI-agent eller -gateway

Disse programmer **administreres ikke via JustVoxels kontrolcenter**. Administratoren har selv ansvaret for deres Podman-containere, Quadlets, lager, porte, opdateringer, sikkerhedskopier, konfiguration af omvendt proxy og programspecifikke krav.

JustVoxel holder ansvarsfordelingen enkel. JustVoxel administrerer Minecraft-serversystemet, og erfarne brugere kan fortsat bruge Podman og Quadlet som normalt til andre tjenester.

Ved vedligeholdelse behandler JustVoxel containere uden for Minecraft som tjenester under administratorens ansvar. Minecraft-handlinger som **Start Over** fjerner derfor ingen af disse Podman-containere.

På den måde kan maskinen først og fremmest fungere som Minecraft-server og samtidig bruges som en almindelig hjemmeserver med containere.

## 🎮 Opret den Minecraft-server, du vil have

JustVoxel kan køre flere uafhængige Minecraft-servere samtidig, når maskinen har nok hukommelse og ledige porte.

Hver server har separat identitet, konfiguration, kørselsmiljø, porte, data og sikkerhedskopier. Du kan styre hver servers livscyklus og se dens status individuelt.

### Understøttet serversoftware

| Serversoftware | Udvidelser | Administreret spil på tværs af udgaver (Java → Bedrock) |
| --- | --- | --- |
| **Paper** | Plugins | ✅ |
| **Purpur** | Plugins | ✅ |
| **Spigot** | Plugins | ✅ |
| **Pufferfish** | Plugins | ✅ |
| **Leaf** | Plugins | ✅ |
| **Fabric** | Mods | ✅ Når kompatibiliteten er bekræftet for den konkrete kombination |
| **NeoForge** | Mods | ✅ Når kompatibiliteten er bekræftet for den konkrete kombination |
| **Forge** | Mods | Kun Java |
| **Quilt** | Mods | Kun Java |
| **Vanilla Java** | Ingen plugins eller mods | Kun Java |
| **Native Bedrock** | Dedikeret Bedrock-server | Bedrock-klienter |

Vælg **Enkel opsætning** for at få hjælp til at oprette din første server. Med **Avanceret opsætning** får du mere kontrol over hukommelse, porte, lager, sikkerhedskopier, versionspolitik og spil på tværs af kompatible udgaver.

JustVoxel kontrollerer, hvilke funktioner den valgte serversoftware understøtter.

[Se serversoftware og kompatibilitet](../docs/server-software.md)

## 🧩 Plugins og mods

Åbn **Minecraft → Plugins og mods** for den valgte server.

Afhængigt af servertypen kan JustVoxel bruge følgende kataloger.

- [Modrinth](https://modrinth.com/)
- [Hangar](https://hangar.papermc.io/)
- [CurseForge](https://www.curseforge.com/minecraft) med den API-nøgle, du har konfigureret

Ved understøttede installationer kontrollerer JustVoxel kompatibilitet, verificerer downloads, tager sikkerhedskopier, mens serveren er stoppet, kontrollerer genstarten, registrerer ejerskab og sikrer mulighed for gendannelse.

Servere i Paper-familien kan bruge pluginkataloger. Fabric, Forge, NeoForge og Quilt bruger modkataloger, hvor JustVoxel kontrollerer den præcise modloader og Minecraft-version. Vanilla Java og Native Bedrock har bevidst intet katalog til plugins eller mods.

[Læs om plugins og mods](../docs/plugin-workspace.md)

## 🌐 Valgfrit Minecraft-netværk

Du kan sagtens køre flere selvstændige servere. Det kræver **ingen proxy**.

Hvis du vil give spillere ét offentligt adgangspunkt til flere kompatible servere, kan du bruge JustVoxels valgfrie **Minecraft-netværk**, som bygger på Velocity.

I kontrolcenteret kan du vælge understøttede backendservere, konfigurere offentlige Java- og Bedrock-porte, administrere den virtuelle lobby og velkomstbeskeden og bestemme, hvilke servere der indgår i netværket.

Paper, Purpur, Pufferfish og Leaf understøttes som backendservere. Kompatible Fabric-, Quilt-, Forge- og NeoForge-servere kan også deltage, når JustVoxel har bekræftet den nødvendige proxykompatibilitet. Spigot, Vanilla og Native Bedrock kører fortsat som selvstændige servere.

[Læs om Minecraft-netværk](../docs/minecraft-network.md)

## 🖥️ Lær kontrolcenteret at kende

JustVoxel tager udgangspunkt i de opgaver, du skal løse for at drive din server, så du kan klare dem uden at kende Linux i detaljer.

### Minecraft-servere

Opret og administrer flere servere, start og stop dem, se status og spilleroplysninger, tilpas understøttede indstillinger, og åbn værktøjer til den enkelte server.

### Plugins og mods

Find kompatibelt indhold, se installerede plugins og mods, administrer understøttede udvidelser, og se oplysninger om gendannelse, hvis en ændring kræver din opmærksomhed.

### Sikkerhedskopier og lager

Opret sikkerhedskopier, se historikken, gendan alle data for en server, vælg, hvor sikkerhedskopier gemmes, importer og eksporter servere, og administrer understøttet lager.

### Netværk

Administrer Ethernet, wi-fi og ekstern adgang, og **find og løs forbindelsesproblemer**. Netværksændringer, der kan afbryde forbindelsen, kan rulles tilbage, så du kan få adgang igen, hvis en IP- eller wi-fi-indstilling går galt.

### System

Se systemtilstand, ressourcer, logfiler, opdateringer af operativsystemet, brugere, sikkerhedsindstillinger og understøttede strømfunktioner samt de hardwareindstillinger, der kun findes i HWS.

### AI- og API-forbindelser

Tilslut understøttede AI-programmer, MCP-klienter, REST-automatisering, Telegram og Discord uden at skulle sætte et ekstra administrationssystem op.

Foretrækker du terminalen? JustVoxel indeholder også **mjust**, en interaktiv terminalgrænseflade til administration, der bruger de samme sikkerhedsregler som resten af systemet.

[Dokumentation til WebUI](../docs/WEBUI.md) · [Vejledning til mjust](../docs/MJUST.md)

## 🌐 Lad vennerne få adgang på dine vilkår

JustVoxel har i øjeblikket fire integrerede løsninger under **Netværk → Ekstern adgang**.

Løsningerne er integreret i JustVoxel, så du alt efter udbyder kan se status, aktivere eller deaktivere forbindelsen og gennemføre den understøttede opsætning.

| Løsning | Velegnet til | Det får du |
| --- | --- | --- |
| **[Tailscale](https://tailscale.com/)** | Private servere for venner og familie | Privat mesh-netværk mellem godkendte enheder |
| **[NetBird](https://netbird.io/)** | Private netværk med styret adgang | Privat netværk med styring af konti og adgangspolitikker |
| **[Playit.gg](https://playit.gg/)** | Offentlig Minecraft-adgang uden almindelig portvideresendelse på routeren | Offentlige spiltunneler og adresser, spillerne kan forbinde til |
| **[NordVPN Meshnet](https://meshnet.nordvpn.com/)** | Privat adgang direkte mellem enheder for NordVPN-brugere | Meshnet-forbindelse uden at bruge JustVoxel som kommerciel VPN-klient |

### Hvilken løsning skal jeg vælge?

**🎮 Skal venner kunne spille med uden at installere en privat VPN-klient?**\
Se først på **Playit.gg**. Spillerne kan bruge den offentlige adresse fra den spiltunnel, du har konfigureret.

**🔒 Vil du have en privat server til enheder, du har tillid til?**\
Overvej **Tailscale** eller **NetBird**.

**🌍 Bruger du allerede NordVPN Meshnet?**\
JustVoxel kan administrere sin Meshnet-integration. Det er en separat funktion fra at forbinde serveren til en af NordVPNs almindelige kommercielle VPN-udgangsservere.

Konti, politikker i skyen og tunnelkonfiguration administreres fortsat hos udbyderen. JustVoxel håndterer den understøttede integration på selve serveren.

Du kan bruge flere VPN- og mesh-løsninger samtidig, men der kan opstå konflikter med routing, DNS eller firewallregler.

[Læs vejledningen til netværk og ekstern adgang](../docs/NETWORKING.md)

## 🤖 AI, MCP, REST og automatisering

Vil du spørge en AI-assistent om serverstatus, forbinde din hjemmeautomatisering eller bygge din egen arbejdsgang til at styre serveren?

JustVoxel har indbyggede **MCP- og REST-grænseflader** under **AI- og API-forbindelser**.

Under opsætningen kan du vælge mellem kompatible programmer og forbindelsestyper som disse.

- **OpenAI – ChatGPT og Codex**
- **Claude**
- **Grok**
- **Gemini**
- **GitHub Copilot**
- **Anden AI-/MCP-app**
- **Automatisering / REST API**

Du bestemmer, hvad hver forbindelse har adgang til.

Tilladelserne spænder fra **Niveau 1 – Skrivebeskyttet** over serveradministration til avanceret systemstyring, som kun er tilgængelig efter udtrykkelig godkendelse. Du kan også begrænse en forbindelse til udvalgte Minecraft-servere.

JustVoxel kan stille MCP- og REST-grænsefladerne til rådighed uden en API-nøgle til en AI-model. Om en bestemt klient kan bruges, afhænger af programmet, de funktioner kontoen eller abonnementet giver adgang til, og netværksadgangen.

AI- og API-gatewayen er deaktiveret, indtil du selv aktiverer den.

[Læs om AI- og API-forbindelser](../docs/AI-API-CONNECTIONS.md)

## 💬 Telegram og Discord

JustVoxel har valgfrie, indbyggede integrationer til Telegram og Discord. De bruger samme centrale administrationsbackend som WebUI.

### Telegram

Med en dedikeret Telegram-bot kan du få adgang til følgende.

- status for systemet og Minecraft
- servervalg og spilleroplysninger
- start, stop og genstart for godkendte administratorer
- manuelle sikkerhedskopier
- gennemgang af opdateringer til Minecraft og containere
- oplysninger om opdateringer til operativsystemet
- kontrolleret genstart af systemet
- valgfrie notifikationer om serverstatus, resultatet af sikkerhedskopiering og tilgængelige opdateringer

Telegram bruger sine egne tilladelser, **Skrivebeskyttet** og **Serveradministration**, og kræver ingen AI-model.

[Opsætning af Telegram og tilladelser](../docs/TELEGRAM.md)

### Discord

Discord kan bruges både til at koordinere Minecraft-fællesskabet og til administration.

Medlemmer af den Discord-server, du udtrykkeligt har knyttet til JustVoxel, kan se grundlæggende serverstatus og anmode om adgang til Minecraft uden automatisk at få administratorrettigheder.

Godkendte JustVoxel-administratorer på Discord kan udføre understøttede handlinger som at starte, stoppe og genstarte servere, tage manuelle sikkerhedskopier, gennemgå opdateringer og styre Minecraft-netværket. De kan også udføre andre avancerede handlinger.

Medlemskab af Discord-serveren, adgang via Minecrafts tilladelsesliste og administratorrettigheder i JustVoxel håndteres hver for sig.

[Opsætning af Discord, adgang for fællesskabet og administration](../docs/DISCORD.md)

## 💾 Sikkerhedskopiering, gendannelse, lager og migrering

Dine Minecraft-data er vigtigere end systemimaget.

JustVoxel holder både Minecraft-data og lageret til sikkerhedskopier adskilt fra det uforanderlige operativsystem. Du vælger lagerplacering for hver del for sig.

Du kan bruge følgende lagermuligheder.

- systemets filsystem
- dedikerede interne diske
- USB-lager
- eksisterende lokale partitioner eller filsystemer
- nye partitioner oprettet på diskplads, der endnu ikke er tildelt en partition
- NFS
- SMB/CIFS

Hver server kan bruge automatisk og manuel sikkerhedskopiering. Ved gendannelse erstattes alle den valgte servers vedvarende data, mens dens JustVoxel-identitet og systemkonfiguration bevares.

Når du flytter servere mellem maskiner eller skal geninstallere, tilbyder JustVoxel også arbejdsgange til **eksport, import og gendannelse**, hvor du kan gennemgå handlingerne.

[Lager](../docs/STORAGE.md) · [Gendannelse](../docs/RESTORE.md) · [Migrering](../docs/MIGRATION.md)

## 🛡️ Bygget til sikrere ændringer

Kontrolcenteret giver kun adgang til de handlinger, JustVoxel understøtter. Hver handling følger faste regler, så du ikke kan køre vilkårlige shellkommandoer fra kontrolcenteret.

Sikkerhedsreglerne omfatter blandt andet følgende.

- kontrol af, om spillere er online, før handlinger, der kan afbryde spillet
- gennemgang af ændringer, før du vælger **Anvend**
- sikkerhedskopiering, mens serveren er stoppet, hvis der skal kunne gendannes efter en ændring
- entydig identifikation af servere, så de ikke kan forveksles
- kontrol af genstart og efterfølgende drift
- vedvarende registrering af status for handlinger og gendannelse
- mulighed for at rulle understøttede handlinger tilbage
- kontrolpunkter i NetworkManager ved risikable netværksændringer
- separate arbejdsgange til opdatering af Minecraft og operativsystemet

WebUI kører uden særlige systemrettigheder. Ændringer, der kræver højere rettigheder, går gennem den lokale JustVoxel Management Agent og de handlinger, dens API stiller til rådighed. Browseren har dermed ingen adgang til en root-shell.

## 🔄 Separate opdateringer til operativsystemet og Minecraft

JustVoxel er et imagebaseret serversystem, der bygger på Home Server Base 10 / AlmaLinux 10.

Operativsystemet opdateres uafhængigt af Minecraft-versioner, serversoftware, plugins, mods og containeropdateringer.

Du kan derfor opdatere JustVoxel og beholde din Minecraft-version. Du kan også ændre en Minecraft-server uden at genopbygge værtens operativsystem.

[Dokumentation til systemadministration](../docs/SYSTEM.md)

## 📊 Hvordan klarer JustVoxel sig i sammenligningen?

JustVoxel adskiller sig fra traditionelle Minecraft-kontrolpaneler ved at samle **operativsystemet og administrationen af Minecraft i ét serversystem**.

| Funktion | **JustVoxel** | **MineOS SvelteKit** | **Crafty Controller** | **MineOS Node** |
| --- | :---: | :---: | :---: | :---: |
| Komplet operativsystem til Minecraft-servere | ⭐ | — | — | — |
| Flere Minecraft-servere | ✅ | ✅ | ✅ | ✅ |
| Understøttelse af Java- og Bedrock-servere | ✅ | ✅ | ✅ | 🟡 Java og ældre PocketMine-understøttelse |
| Vejledt søgning efter plugins og mods | ✅ Modrinth, Hangar, CurseForge | ✅ Modrinth, CurseForge | — | — |
| Administration af værtens netværk, lager og operativsystem | ⭐ | — | — | — |
| Integreret privat adgang og spiltunneler | ⭐ 4 muligheder | — | — | — |
| MCP- og REST-gateway til AI og automatisering | ⭐ | — | — | — |
| Indbygget administration via Telegram og Discord | ⭐ | — | — | — |
| Indstillinger for låg og skærm på bærbare | ✅ HWS | — | — | — |
| Imagebaserede opdateringer af et uforanderligt operativsystem | ✅ | — | — | — |
| Administration fra terminalen | ✅ mjust | ✅ mineos CLI/TUI | — | ✅ CLI |
| Installationsmodel | Komplet bootc-serversystem | Docker-/Compose-program | Kontrolpanel i Docker eller på Linux eller Windows | Node.js-program på Linux/BSD |

⭐ særlig styrke &nbsp;•&nbsp; ✅ understøttet &nbsp;•&nbsp; 🟡 delvis eller ældre understøttelse &nbsp;•&nbsp; — fremgår ikke af den gennemgåede dokumentation for projektets kernefunktioner

**Sammenligningen blev senest gennemgået den 16. november 2026.** Formålet er at vise, hvad de forskellige projekter tilbyder. MineOS og Crafty er programmer til Minecraft-administration og kan være gode valg, hvis du allerede har et operativsystem, du selv vil administrere. JustVoxel henvender sig til dig, der vil administrere Minecraft-kontrolpanelet **og det underliggende serversystem** som ét produkt.

Sammenligningen bygger på projekternes aktuelle dokumentation og kildekodearkiver.

- [MineOS SvelteKit](https://github.com/freeman412/mineos-sveltekit)
- [Crafty Controller](https://craftycontrol.com/) · [Kildekode](https://gitlab.com/crafty-controller/crafty-4)
- [MineOS Node](https://github.com/hexparrot/mineos-node)

Opret gerne en sag, hvis en oplysning i sammenligningen er forældet eller giver et skævt billede. Rettelser er også velkomne, hvis de stiller JustVoxel dårligere i sammenligningen.

## 📚 Dokumentation

De detaljerede vejledninger til JustVoxel ligger i [dokumentationsmappen](../docs/).

Her er nogle gode steder at begynde.

- [WebUI](../docs/WEBUI.md)
- [Kørselsmiljø til Minecraft](../docs/MINECRAFT_RUNTIME.md)
- [Serversoftware](../docs/server-software.md)
- [Plugins og mods](../docs/plugin-workspace.md)
- [Minecraft-netværk](../docs/minecraft-network.md)
- [Netværk og ekstern adgang](../docs/NETWORKING.md)
- [AI- og API-forbindelser](../docs/AI-API-CONNECTIONS.md)
- [Telegram](../docs/TELEGRAM.md)
- [Discord](../docs/DISCORD.md)
- [Lager](../docs/STORAGE.md)
- [Gendannelse](../docs/RESTORE.md)
- [Migrering](../docs/MIGRATION.md)
- [Systemadministration](../docs/SYSTEM.md)
- [mjust](../docs/MJUST.md)

Se [udviklervejledningen](../README-DEV.md) for udvikling og test af systemimages samt udgivelser.

## ⛏️ Minecraft-software og EULA

JustVoxel leveres **ikke** med binærfiler til Minecraft-serveren, Mojangs serversoftware, en færdigoprettet verden eller en Minecraft-slutbrugerlicensaftale (EULA), der er accepteret på forhånd.

Når en server oprettes, accepterer administratoren Minecrafts EULA, og JustVoxel gør det valgte understøttede kørselsmiljø til Minecraft-serveren klar.

Minecraft, Mojang-software, Paper, Purpur, Spigot, Pufferfish, Leaf, Fabric, NeoForge, Forge, Quilt, plugins, mods og andre tredjepartskomponenter beholder deres egne licenser og distributionsvilkår.

## 📄 Licens

JustVoxel er licenseret under **Apache-2.0**. Se [LICENSE](../LICENSE).

Oplysninger om tredjepartskomponenter vedligeholdes i [THIRD_PARTY_NOTICES.md](../THIRD_PARTY_NOTICES.md).
