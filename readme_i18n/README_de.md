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

> **Hinweis zur Übersetzung:** Diese JustVoxel-Übersetzung wurde mithilfe von KI aus dem maßgeblichen englischen Originaltext erstellt. Wenn dir eine ungenaue, unklare oder unnatürliche Formulierung auffällt, schlage bitte eine Korrektur über eine [Fehlermeldung auf GitHub](https://github.com/home-server-project/justvoxel/issues) oder einen [Änderungsvorschlag (Pull Request)](https://github.com/home-server-project/justvoxel/pulls) vor.

<p align="center">
  <strong>Deine Minecraft-Server. Eigene Hardware, ein gemieteter dedizierter Server oder eine gehostete VM. Ein Control Center.</strong>
</p>

<p align="center">
  Eine selbst gehostete Minecraft-Server-Appliance für alle, die ihren Server betreiben möchten, ohne sich um die Verwaltung des Betriebssystems kümmern zu müssen.
</p>

<p align="center">
  <a href="#-justvoxel-installieren">Installation</a> •
  <a href="#-minecraft-hosting-leicht-gemacht">Funktionen</a> •
  <a href="../docs/INSTALLER.md">Dokumentation</a>
</p>

<p align="center">
  <img src="../docs/images/justvoxel-overview-1.webp" width="100%" alt="Übersicht des JustVoxel Control Center"><img src="../docs/images/justvoxel-overview-2.webp" width="100%" alt=""><img src="../docs/images/justvoxel-overview-3.webp" width="100%" alt="">
</p>

JustVoxel ist eine selbst gehostete Appliance speziell für Minecraft-Server. Installiere JustVoxel auf einem kompatiblen PC, Laptop, physischen Server, in einer virtuellen Maschine oder bei einem Hoster, der eigene Betriebssystem-Images erlaubt. Anschließend verwaltest du Minecraft im Browser.

Du musst weder eine Sammlung von Skripten pflegen noch ein separates Verwaltungspanel für Minecraft installieren.

Hast du noch einen alten Laptop oder PC? **Mit JustVoxel HWS wird daraus eine dedizierte Minecraft-Appliance, bevor die Hardware in der Schublade verstaubt.** Auf unterstützten Laptops kannst du den Ruhezustand beim Zuklappen unterbinden, damit der Server bei geschlossenem Deckel weiterläuft. Den Bildschirm der lokalen Konsole kannst du sofort oder nach einer kurzen Verzögerung ausschalten.

> 🚧 **Vorabversion:** Die erste öffentliche Veröffentlichung von JustVoxel ist in Vorbereitung. Das aktuelle Installations-ISO steht zum Testen und für den frühen Einsatz bereit, während die abschließende Prüfung für die Freigabe läuft.

## ✨ Minecraft-Hosting leicht gemacht

JustVoxel verbindet die Minecraft-Verwaltung mit dem zugrunde liegenden Betriebssystem.

- 🎮 **Mehrere Minecraft-Server** – Betreibe unterschiedliche Welten und Servertypen unabhängig voneinander auf einer Appliance.
- 🖥️ **Control Center im Browser** – Verwalte Server am Computer, Tablet oder Smartphone.
- 🧩 **Plugins und Mods** – Finde und verwalte kompatible Inhalte von unterstützten Anbietern.
- 💾 **Sicherung und Wiederherstellung** – Schütze jeden Server mit geplanten und manuellen Sicherungen.
- 🌐 **Fernzugriff** – Wähle zwischen vier integrierten Lösungen für privaten Zugriff und Spieltunnel.
- 🤖 **KI und Automatisierung** – Binde kompatible KI-Assistenten und Anwendungen über MCP und REST an.
- 💬 **Telegram und Discord** – Nutze bei Bedarf Bots zur Steuerung, für Benachrichtigungen und für Abläufe in deiner Community.
- 🛡️ **Systemverwaltung** – Verwalte Netzwerk, Speicher, Updates, Systemzustand, Protokolle und unterstützte Energieoptionen, ohne die zugehörigen Linux-Befehle lernen zu müssen.

JustVoxel richtet sich an Familien, Freundeskreise, kleine Minecraft-Communitys und alle, die einen eigenen Server betreiben möchten, statt ein klassisches Minecraft-Hostingpanel zu mieten.

## 🖥️ Eine Appliance, zwei Editionen

### JustVoxel VM

Für virtuelle Maschinen und Hostingumgebungen wie

- Proxmox
- KVM / libvirt
- VMware
- Hyper-V
- VirtualBox
- gehostete VMs oder VPS-Angebote, die ein eigenes kompatibles Betriebssystem-Image zulassen

Die VM-Edition enthält die vollständige Minecraft-Appliance. Pakete für physische Hardware, die in einer VM keinen Zweck erfüllen, entfallen.

### JustVoxel HWS

Für die direkte Installation auf physischer Hardware, zum Beispiel auf

- einem alten Laptop
- einem umfunktionierten Desktop-PC
- einem Mini-PC
- einem Heimserver
- einem gemieteten Bare-Metal-Server, der eigene Installationsmedien zulässt

HWS ergänzt Funktionen für physische Geräte, darunter USV-Anbindung über NUT, SMART- und NVMe-Werkzeuge, Sensoren, Firmware- und UEFI-Einstellungen, Gerätefirmware, Hardwarediagnose sowie Einstellungen für Laptopbildschirm und Deckel.

Beide Editionen bieten dasselbe JustVoxel Control Center, dieselbe Management API und dieselben Abläufe für Minecraft, Speicher, Sicherungen, Netzwerk und Updates.

[Funktionsumfang von VM und HWS](../docs/VARIANTS.md)

## 🚀 JustVoxel installieren

Ein gemeinsames Installations-ISO unterstützt beide JustVoxel-Editionen.

[⬇️ JustVoxel-ISO für x86_64 herunterladen](https://sourceforge.net/projects/justvoxel/files/JustVoxel-x86_64.iso/download)

Wähle die Edition passend zu deiner Umgebung:

- **JustVoxel VM** für virtuelle Maschinen und kompatible gehostete VM-Umgebungen
- **JustVoxel HWS** für PCs, Laptops, Mini-PCs, physische Heimserver und kompatibles Bare-Metal-Hosting

Die [Installationsanleitung](../docs/INSTALLER.md) beschreibt die Installation, die Vorbereitung der Datenträger und die erste Anmeldung.

## 🏠 Minecraft im Mittelpunkt, mit Platz für weitere Dienste

JustVoxel wird in erster Linie als Minecraft-Server-Appliance entwickelt und unterstützt. Darunter läuft ein regulärer **AlmaLinux-10-Server auf bootc-Basis mit Podman, Quadlet, systemd, SSH, NetworkManager und gängigen Linux-Verwaltungswerkzeugen**.

Erfahrene Nutzer können denselben Rechner als vielseitigen Heimserver nutzen und neben Minecraft weitere Container betreiben.

Du kannst beispielsweise folgende Dienste manuell einrichten:

- **Jellyfin** oder **Plex** für Medien
- **Vaultwarden** zur Passwortverwaltung
- **Immich** zur Foto- und Videoverwaltung
- **Home Assistant Container** zur Heimautomatisierung
- **Nextcloud** für Dateien und Zusammenarbeit
- **Syncthing** oder **Navidrome**
- **OpenClaw** als selbst gehosteten KI-Agenten bzw. als Gateway

Diese Anwendungen werden **nicht vom JustVoxel Control Center verwaltet**. Für ihre Podman-Container, Quadlets, Speicherorte, Ports, Updates, Sicherungen, Reverse-Proxy-Konfiguration und anwendungsspezifischen Anforderungen bist du als Administrator selbst zuständig.

Die Zuständigkeiten sind bewusst einfach gehalten. JustVoxel verwaltet die Minecraft-Appliance. Für alles Weitere stehen erfahrenen Nutzern die üblichen Podman- und Quadlet-Werkzeuge zur Verfügung.

Auch bei der Wartung behandelt JustVoxel fremde Container als eigenständig verwaltete Dienste. Minecraft-Funktionen wie **Start Over** entfernen keine unbeteiligten Podman-Container.

So kann dein JustVoxel-Rechner in erster Linie eine Minecraft-Appliance bleiben und bei Bedarf zugleich als gewöhnlicher Heimserver für Container dienen.

## 🎮 Erstelle deinen eigenen Minecraft-Server

JustVoxel kann mehrere unabhängige Minecraft-Server gleichzeitig betreiben, wenn genügend Arbeitsspeicher und freie Ports verfügbar sind.

Jeder Server hat seine eigene Kennung, Konfiguration und Laufzeitumgebung sowie eigene Ports, Daten, Sicherungen, Betriebssteuerung und Statusanzeige.

### Aktuell unterstützte Serversoftware

| Serversoftware | Erweiterungen | Verwaltetes Crossplay Java → Bedrock |
| --- | --- | --- |
| **Paper** | Plugins | ✅ |
| **Purpur** | Plugins | ✅ |
| **Spigot** | Plugins | ✅ |
| **Pufferfish** | Plugins | ✅ |
| **Leaf** | Plugins | ✅ |
| **Fabric** | Mods | ✅ Bei nachgewiesener Kompatibilität der konkreten Kombination |
| **NeoForge** | Mods | ✅ Bei nachgewiesener Kompatibilität der konkreten Kombination |
| **Forge** | Mods | Nur Java |
| **Quilt** | Mods | Nur Java |
| **Vanilla Java** | Keine Plugins oder Mods | Nur Java |
| **Natives Bedrock** | Nativer Bedrock-Server | Native Bedrock-Clients |

Die **einfache Einrichtung** führt dich durch die Erstellung deines ersten Servers. Wähle die **erweiterte Einrichtung**, wenn du Arbeitsspeicher, Ports, Speicherorte, Sicherungen, Versionsvorgaben und kompatibles Crossplay genauer festlegen möchtest.

JustVoxel prüft anhand der gewählten Serversoftware, welche Funktionen tatsächlich unterstützt werden.

[Details zu Serversoftware und Kompatibilität](../docs/server-software.md)

## 🧩 Plugins und Mods

Öffne für den ausgewählten Server **Minecraft → Plugins & Mods**.

Je nach Servertyp kann JustVoxel auf folgende Quellen zugreifen:

- [Modrinth](https://modrinth.com/)
- [Hangar](https://hangar.papermc.io/)
- [CurseForge](https://www.curseforge.com/minecraft) mit deinem eingerichteten API-Schlüssel

Bei unterstützten Installationen prüft JustVoxel Kompatibilität und Downloads, erstellt Sicherungen bei gestopptem Server, überprüft den Neustart, erfasst die Zuordnung installierter Inhalte und sichert die Wiederherstellung ab.

Server der Paper-Familie können Plugin-Kataloge nutzen. Fabric, Forge, NeoForge und Quilt verwenden Mod-Kataloge mit genauer Prüfung von Loader und Minecraft-Version. Für Vanilla Java und natives Bedrock ist bewusst kein Plugin- oder Mod-Katalog vorgesehen.

[Weitere Informationen zu Plugins und Mods](../docs/plugin-workspace.md)

## 🌐 Optionales Minecraft Network

Du kannst problemlos mehrere eigenständige Server betreiben. Sie benötigen **keinen** Proxy.

Wenn mehrere kompatible Server über einen gemeinsamen öffentlichen Zugang erreichbar sein sollen, bietet JustVoxel das optionale **Minecraft Network** auf Basis von Velocity.

Im Control Center kannst du unterstützte Backend-Server auswählen, öffentliche Java- und Bedrock-Ports festlegen, die virtuelle Lobby und die Begrüßungsnachricht verwalten und bestimmen, welche Server zum Network gehören.

Paper, Purpur, Pufferfish und Leaf werden als Network-Backends unterstützt. Kompatible Fabric-/Quilt- und Forge-/NeoForge-Server können ebenfalls teilnehmen, wenn JustVoxel die erforderliche Proxy-Kompatibilität nachweist. Spigot, Vanilla und natives Bedrock bleiben eigenständig.

[Weitere Informationen zu Minecraft Network](../docs/minecraft-network.md)

## 🖥️ Das Control Center im Überblick

Im Mittelpunkt von JustVoxel stehen die Aufgaben rund um deinen Server. Linux-Interna musst du dafür nicht kennen.

### Minecraft-Server

Erstelle und verwalte mehrere Server, starte oder stoppe sie, prüfe ihren Status und die Spielerliste, ändere unterstützte Einstellungen und nutze serverspezifische Werkzeuge.

### Plugins und Mods

Entdecke kompatible Inhalte, sieh dir installierte Plugins und Mods an, verwalte unterstützte Erweiterungen und finde Hinweise zur Wiederherstellung, wenn eine Änderung einen Eingriff erfordert.

### Sicherungen und Speicher

Erstelle Sicherungen, prüfe den Sicherungsverlauf, stelle vollständige Serverdaten wieder her, lege Sicherungsziele fest, importiere oder exportiere Server und verwalte unterstützte Speicherorte.

### Netzwerk

Verwalte Ethernet, WLAN und Fernzugriff und nutze die **Fehlerdiagnose für Verbindungen**. Bei Änderungen, die deine Netzwerkverbindung unterbrechen könnten, schützt dich eine Rücknahmefunktion davor, durch eine falsche IP- oder WLAN-Einstellung dauerhaft den Zugriff zu verlieren.

### System

Prüfe Systemzustand, Ressourcen, Protokolle, Betriebssystem-Updates, Benutzer und Sicherheit. Hier findest du auch unterstützte Energieoptionen und die Hardwaresteuerung der HWS-Edition.

### KI- und API-Verbindungen

Verbinde unterstützte KI-Anwendungen, MCP-Clients, REST-Automatisierungen, Telegram und Discord, ohne ein weiteres Verwaltungssystem einrichten zu müssen.

Du arbeitest lieber im Terminal? JustVoxel enthält mit **mjust** eine interaktive Verwaltungsoberfläche für das Terminal. Es gelten dieselben Sicherheitsregeln wie für die übrige Appliance.

[WebUI-Dokumentation](../docs/WEBUI.md) · [mjust-Anleitung](../docs/MJUST.md)

## 🌐 So verbinden sich deine Freunde

Unter **Netzwerk → Fernzugriff** bietet JustVoxel derzeit vier Möglichkeiten für den Fernzugriff.

Die Integration umfasst je nach Anbieter die Statusanzeige, das Aktivieren und Deaktivieren sowie die Einrichtung direkt in JustVoxel.

| Option | Geeignet für | Funktionsumfang |
| --- | --- | --- |
| **[Tailscale](https://tailscale.com/)** | Private Server für Freunde und Familie | Privates Mesh-Netz zwischen zugelassenen Geräten |
| **[NetBird](https://netbird.io/)** | Private Netze mit verwaltetem Zugriff | Privates Netz mit Steuerung über Konten und Zugriffsrichtlinien |
| **[Playit.gg](https://playit.gg/)** | Öffentlichen Minecraft-Zugang ohne klassische Portweiterleitung am Router | Öffentliche Spieltunnel und Verbindungsadressen |
| **[NordVPN Meshnet](https://meshnet.nordvpn.com/)** | Privaten Direktzugriff zwischen Geräten für NordVPN-Nutzer | Meshnet-Verbindung, ohne JustVoxel als kommerziellen VPN-Client zu nutzen |

### Welche Option passt zu mir?

**🎮 Sollen Freunde ohne privaten VPN-Client mitspielen können?**  
Schau dir zuerst **Playit.gg** an. Deine Mitspieler können die öffentliche Adresse deines eingerichteten Spieltunnels verwenden.

**🔒 Möchtest du einen privaten Server für vertrauenswürdige Geräte?**  
Dafür kommen **Tailscale** oder **NetBird** infrage.

**🌍 Nutzt du bereits NordVPN Meshnet?**  
JustVoxel kann die Meshnet-Anbindung verwalten. Eine Verbindung zu einem regulären kommerziellen NordVPN-Ausgangsserver ist davon unabhängig.

Konten, cloudseitige Richtlinien und Tunnelkonfiguration verwaltest du weiterhin beim jeweiligen Anbieter. JustVoxel übernimmt die unterstützte Anbindung auf der Appliance.

Du kannst mehrere VPN- oder Mesh-Produkte gleichzeitig nutzen. Dabei können jedoch Konflikte bei Routing, DNS oder Firewallregeln auftreten.

[Anleitung zu Netzwerk und Fernzugriff](../docs/NETWORKING.md)

## 🤖 KI, MCP, REST und Automatisierung

Möchtest du einen KI-Assistenten nach dem Serverstatus fragen, deine Heimautomatisierung anbinden oder eigene Steuerungsabläufe erstellen?

Unter **KI- und API-Verbindungen** stellt JustVoxel integrierte **MCP- und REST-Schnittstellen** bereit.

Bei der Einrichtung stehen kompatible Anwendungen und Verbindungstypen zur Auswahl, darunter:

- **OpenAI — ChatGPT & Codex**
- **Claude**
- **Grok**
- **Gemini**
- **GitHub Copilot**
- **Andere KI- oder MCP-Anwendung**
- **Automatisierung / REST-API**

Du entscheidest, worauf jede Verbindung zugreifen darf.

Die Berechtigungen reichen von **Stufe 1 – Nur lesen** über die Serververwaltung bis zur erweiterten Systemsteuerung, die eine ausdrückliche Freigabe erfordert. Du kannst Verbindungen auch auf ausgewählte Minecraft-Server beschränken.

Für die Bereitstellung der MCP- oder REST-Schnittstelle benötigt JustVoxel keinen API-Schlüssel für ein KI-Modell. Ob ein Client die Verbindung unterstützt, hängt von der Anwendung, den Funktionen deines Kontos bzw. Abos und dem Netzwerkzugriff ab.

Das KI-/API-Gateway ist anfangs deaktiviert, bis du es einschaltest.

[Weitere Informationen zu KI- und API-Verbindungen](../docs/AI-API-CONNECTIONS.md)

## 💬 Telegram und Discord

JustVoxel bietet optionale, integrierte Anbindungen für Telegram und Discord. Beide verwenden dasselbe zentrale Verwaltungsbackend wie die WebUI.

### Telegram

Ein eigener Telegram-Bot bietet folgende Möglichkeiten:

- Status der Appliance und der Minecraft-Server abrufen
- Server auswählen und Spielerinformationen anzeigen
- Server mit entsprechender Verwaltungsberechtigung starten, stoppen und neu starten
- manuelle Sicherungen erstellen
- Minecraft- und Container-Updates prüfen
- Informationen zu Betriebssystem-Updates abrufen
- die Appliance kontrolliert neu starten
- auf Wunsch Benachrichtigungen zu Serverzustand, Sicherungsergebnissen und verfügbaren Updates erhalten

Telegram verwendet eigene Berechtigungen für **Lesezugriff** und **Serververwaltung** und benötigt kein KI-Modell.

[Telegram einrichten und Berechtigungen vergeben](../docs/TELEGRAM.md)

### Discord

Über Discord kann deine Community auf Minecraft-Funktionen zugreifen. Zugleich lässt sich Discord für die Verwaltung nutzen.

Mitglieder des ausdrücklich verknüpften Discord-Servers können den grundlegenden Serverstatus einsehen und Minecraft-Zugang beantragen. Sie erhalten dadurch keine automatischen Verwaltungsrechte.

Für JustVoxel freigeschaltete Discord-Administratoren können unterstützte Serveraktionen ausführen, etwa Server starten, stoppen oder neu starten, manuelle Sicherungen erstellen, Updates prüfen, das Minecraft Network steuern und weitere erweiterte Aktionen ausführen.

Discord-Mitgliedschaft, Zugang über die Minecraft-Whitelist und JustVoxel-Verwaltung bleiben getrennte Berechtigungen.

[Discord einrichten, Community-Zugriff und Verwaltung](../docs/DISCORD.md)

## 💾 Sicherungen, Wiederherstellung, Speicher und Migration

Die Minecraft-Daten sind wichtiger als das Appliance-Image.

JustVoxel speichert Minecraft-Daten und Sicherungen außerhalb des unveränderlichen Betriebssystems. Ihre Speicherorte lassen sich unabhängig voneinander wählen.

Unterstützt werden:

- das Systemdateisystem
- dedizierte interne Datenträger
- USB-Speicher
- vorhandene lokale Partitionen und Dateisysteme
- neue Partitionen in bereits unzugeordnetem Speicherplatz
- NFS
- SMB/CIFS

Für jeden Server sind automatische und manuelle Sicherungen möglich. Bei der Wiederherstellung werden sämtliche dauerhaft gespeicherten Daten des ausgewählten Servers ersetzt. Seine JustVoxel-Kennung und seine Konfiguration auf der Appliance bleiben erhalten.

Für den Umzug auf einen anderen Rechner oder die Übernahme nach einer Neuinstallation bietet JustVoxel außerdem geprüfte Abläufe für **Export, Import und Wiederherstellung**.

[Speicher](../docs/STORAGE.md) · [Wiederherstellung](../docs/RESTORE.md) · [Migration](../docs/MIGRATION.md)

## 🛡️ Schutz bei Änderungen

Die JustVoxel-Oberfläche führt keine beliebigen Shell-Befehle aus.

Für unterstützte Vorgänge gelten auf der Appliance unter anderem folgende Sicherheitsmaßnahmen:

- Vor Aktionen, die das Spiel unterbrechen, wird geprüft, ob Spieler verbunden sind.
- Änderungen lassen sich vor dem Anwenden prüfen.
- Bei Änderungen, die eine Absicherung für die Wiederherstellung brauchen, wird der Server zur Sicherung gestoppt.
- Jeder Server hat eine eindeutige Kennung, damit Server nicht versehentlich verwechselt werden.
- Neustart und laufender Betrieb werden überprüft.
- Der Status laufender Vorgänge und der Wiederherstellung wird dauerhaft gespeichert.
- Für unterstützte Vorgänge gibt es Möglichkeiten zur Rücknahme.
- NetworkManager-Prüfpunkte sichern riskante Netzwerkänderungen ab.
- Minecraft- und Betriebssystem-Updates haben getrennte Abläufe.

Die WebUI im Browser läuft ohne erhöhte Rechte. Privilegierte Änderungen erfolgen über den lokalen JustVoxel Management Agent und dessen ausdrücklich definierte API. Der Browser erhält keinen Zugriff auf eine Root-Shell.

## 🔄 System und Minecraft getrennt aktualisieren

JustVoxel ist eine Image-basierte Appliance auf Grundlage von Home Server Base 10 / AlmaLinux 10.

Betriebssystem-Updates erfolgen unabhängig von Minecraft-Versionen, Serversoftware, Plugins, Mods und Container-Updates.

Wenn du die Appliance aktualisierst, ändert sich deine Minecraft-Version also nicht stillschweigend mit. Wenn du einen Minecraft-Server änderst, musst du das Betriebssystem des Hosts nicht neu bauen.

[Dokumentation zur Systemverwaltung](../docs/SYSTEM.md)

## 📊 Wie schneidet JustVoxel im Vergleich ab?

JustVoxel verfolgt einen anderen Ansatz als klassische Minecraft-Verwaltungspanels. **Betriebssystem und Minecraft-Verwaltung bilden eine gemeinsame Appliance.**

| Funktion | **JustVoxel** | **MineOS SvelteKit** | **Crafty Controller** | **MineOS Node** |
| --- | :---: | :---: | :---: | :---: |
| Vollständiges Betriebssystem für eine Minecraft-Appliance | ⭐ | — | — | — |
| Mehrere Minecraft-Server | ✅ | ✅ | ✅ | ✅ |
| Unterstützung für Java- und Bedrock-Server | ✅ | ✅ | ✅ | 🟡 Java und ältere PocketMine-Unterstützung |
| Geführte Suche nach Plugins und Mods | ✅ Modrinth, Hangar, CurseForge | ✅ Modrinth, CurseForge | — | — |
| Verwaltung von Hostnetzwerk, Speicher und Betriebssystem | ⭐ | — | — | — |
| Integrierter privater Zugriff / Spieltunnel | ⭐ 4 Optionen | — | — | — |
| MCP- und REST-Gateway für KI und Automatisierung | ⭐ | — | — | — |
| Integrierte Verwaltung über Telegram und Discord | ⭐ | — | — | — |
| Steuerung von Laptopdeckel und Bildschirm | ✅ HWS | — | — | — |
| Image-basierte Updates für ein unveränderliches Betriebssystem | ✅ | — | — | — |
| Verwaltungsoberfläche im Terminal | ✅ mjust | ✅ mineos CLI/TUI | — | ✅ CLI |
| Installationsmodell | Vollständige bootc-Appliance | Docker-/Compose-Anwendung | Verwaltungspanel unter Docker, Linux oder Windows | Node.js-Anwendung unter Linux/BSD |

⭐ Besonderheit &nbsp;•&nbsp; ✅ unterstützt &nbsp;•&nbsp; 🟡 teilweise oder nur für ältere Software unterstützt &nbsp;•&nbsp; — in der geprüften Kerndokumentation des Projekts nicht aufgeführt

**Vergleich zuletzt geprüft am 16. November 2026.** Dieser Vergleich soll andere Projekte nicht schlechtreden. MineOS und Crafty sind Minecraft-Verwaltungsanwendungen, die sich besonders gut eignen können, wenn du bereits ein Betriebssystem hast, das du selbst verwalten möchtest. JustVoxel richtet sich an alle, die das Minecraft-Panel **und die zugrunde liegende Server-Appliance** als ein Produkt verwalten möchten.

Der Vergleich stützt sich auf die aktuelle Dokumentation und die Repositorys der Projekte:

- [MineOS SvelteKit](https://github.com/freeman412/mineos-sveltekit)
- [Crafty Controller](https://craftycontrol.com/) · [Quellcode](https://gitlab.com/crafty-controller/crafty-4)
- [MineOS Node](https://github.com/hexparrot/mineos-node)

Ist ein Eintrag veraltet oder unfair, eröffne bitte ein Issue. Korrekturen sind auch dann willkommen, wenn JustVoxel dadurch schlechter abschneidet.

## 📚 Dokumentation

Die ausführlichen Anleitungen zur Appliance findest du im [Dokumentationsverzeichnis](../docs/).

Nützliche Einstiegspunkte:

- [WebUI](../docs/WEBUI.md)
- [Minecraft-Laufzeitumgebung](../docs/MINECRAFT_RUNTIME.md)
- [Serversoftware](../docs/server-software.md)
- [Plugins und Mods](../docs/plugin-workspace.md)
- [Minecraft Network](../docs/minecraft-network.md)
- [Netzwerk und Fernzugriff](../docs/NETWORKING.md)
- [KI- und API-Verbindungen](../docs/AI-API-CONNECTIONS.md)
- [Telegram](../docs/TELEGRAM.md)
- [Discord](../docs/DISCORD.md)
- [Speicher](../docs/STORAGE.md)
- [Wiederherstellung](../docs/RESTORE.md)
- [Migration](../docs/MIGRATION.md)
- [Systemverwaltung](../docs/SYSTEM.md)
- [mjust](../docs/MJUST.md)

Informationen zur Image-Entwicklung, zu Tests und zu Veröffentlichungen findest du im [Entwicklerhandbuch](../README-DEV.md).

## ⛏️ Minecraft-Software und EULA

JustVoxel liefert **weder Minecraft-Serverbinärdateien noch Serversoftware von Mojang, eine vorab erstellte Welt oder eine bereits akzeptierte Minecraft-EULA** mit.

Beim Erstellen eines Servers akzeptiert der Administrator die Minecraft-EULA. JustVoxel bereitet die ausgewählte unterstützte Minecraft-Serverlaufzeitumgebung vor.

Für Minecraft, Software von Mojang, Paper, Purpur, Spigot, Pufferfish, Leaf, Fabric, NeoForge, Forge, Quilt, Plugins, Mods und andere Komponenten von Drittanbietern gelten weiterhin die jeweiligen Lizenzen und Vertriebsbedingungen.

## 📄 Lizenz

JustVoxel steht unter der Lizenz **Apache-2.0**. Siehe [LICENSE](../LICENSE).

Hinweise zu Drittanbietern findest du in [THIRD_PARTY_NOTICES.md](../THIRD_PARTY_NOTICES.md).
