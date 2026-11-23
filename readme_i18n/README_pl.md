<p align="center">
  <img src="../docs/images/justvoxel-logo.png" width="260" alt="JustVoxel">
</p>

<p align="center">
  <a href="../README.md">English</a> •
  <a href="README_fr-CA.md">Français (Canada)</a> •
  <a href="README_de.md">Deutsch</a> •
  <a href="README_pl.md">Polski</a> •
  <a href="README_pt-BR.md">Português (Brasil)</a> •
  <a href="README_es-419.md">Español (Latinoamérica)</a> •
  <a href="README_uk.md">Українська</a> •
  <a href="README_ru.md">Русский</a> •
  <a href="README_zh-Hans.md">简体中文</a>
</p>

<p align="center">
  <strong>Twoje serwery Minecraft. Własny sprzęt, wynajęty serwer dedykowany lub maszyna wirtualna u dostawcy hostingu. Jedno centrum sterowania.</strong>
</p>

<p align="center">
  Kompletny system do samodzielnego hostowania serwerów Minecraft dla osób, które chcą prowadzić serwer gry bez konieczności administrowania systemem operacyjnym.
</p>

<p align="center">
  <a href="#-instalacja-justvoxel">Instalacja</a> •
  <a href="#-prostszy-hosting-serwerów-minecraft">Funkcje</a> •
  <a href="../docs/INSTALLER.md">Dokumentacja</a>
</p>

<p align="center">
  <img src="../docs/images/justvoxel-overview-1.webp" width="100%" alt="Widok centrum sterowania JustVoxel"><img src="../docs/images/justvoxel-overview-2.webp" width="100%" alt=""><img src="../docs/images/justvoxel-overview-3.webp" width="100%" alt="">
</p>

JustVoxel to system przeznaczony do prowadzenia własnych serwerów Minecraft. Zainstaluj go na zgodnym komputerze stacjonarnym, laptopie, serwerze fizycznym, maszynie wirtualnej lub u dostawcy hostingu, który pozwala uruchomić własny obraz systemu operacyjnego. Potem możesz zarządzać serwerami Minecraft z przeglądarki.

Nie musisz utrzymywać zestawu skryptów ani instalować osobnego panelu do zarządzania serwerami Minecraft.

Masz stary laptop lub komputer stacjonarny? **Dzięki JustVoxel HWS sprzęt, który leży w szufladzie, może stać się dedykowanym serwerem Minecraft.** Na obsługiwanych laptopach można wyłączyć usypianie po zamknięciu pokrywy, aby serwer nadal działał. Można też wygasić ekran lokalnej konsoli od razu lub po krótkiej zwłoce.

> 🚧 **Wersja przedpremierowa:** Trwają przygotowania do pierwszego publicznego wydania JustVoxel. Obecny obraz instalacyjny ISO jest dostępny do testów i wczesnego użytkowania. Końcowa weryfikacja wydania nadal trwa.

## ✨ Prostszy hosting serwerów Minecraft

JustVoxel łączy zarządzanie serwerami Minecraft i systemem operacyjnym, na którym działają.

- 🎮 **Wiele serwerów Minecraft** — uruchamiaj niezależnie różne światy i typy serwerów na jednym urządzeniu.
- 🖥️ **Centrum sterowania w przeglądarce** — zarządzaj serwerami z komputera, tabletu lub telefonu.
- 🧩 **Wtyczki i mody** — wyszukuj zgodne dodatki u obsługiwanych dostawców i zarządzaj nimi.
- 💾 **Kopie zapasowe i przywracanie** — chroń dane każdego serwera, tworząc kopie ręcznie lub według harmonogramu.
- 🌐 **Dostęp zdalny** — wybierz jedno z czterech zintegrowanych rozwiązań do prywatnego dostępu lub tunelowania połączeń z grą.
- 🤖 **AI i automatyzacja** — podłącz zgodnych asystentów AI i aplikacje przez MCP i REST.
- 💬 **Telegram i Discord** — opcjonalnie korzystaj ze sterowania przez boty, powiadomień i funkcji dla społeczności.
- 🛡️ **Zarządzanie systemem** — konfiguruj sieć i pamięć masową, zarządzaj aktualizacjami, sprawdzaj stan systemu i dzienniki oraz korzystaj z obsługiwanych funkcji zasilania bez nauki poleceń Linuksa.

JustVoxel powstał z myślą o rodzinach, grupach znajomych, małych społecznościach Minecraft i osobach, które wolą mieć własny serwer zamiast wynajmować tradycyjny hosting Minecraft z panelem zarządzania.

## 🖥️ Jeden system, dwie edycje

### JustVoxel VM

Do maszyn wirtualnych i środowisk hostingowych, takich jak:

- Proxmox
- KVM / libvirt
- VMware
- Hyper-V
- VirtualBox
- hostowane maszyny wirtualne lub serwery VPS, na których można uruchomić własny zgodny obraz systemu operacyjnego

Edycja VM zawiera kompletny system serwerowy Minecraft bez pakietów do obsługi sprzętu fizycznego, które są zbędne w maszynie wirtualnej.

### JustVoxel HWS

Do instalacji bezpośrednio na sprzęcie, na przykład na:

- starym laptopie
- komputerze stacjonarnym przeznaczonym do nowego zastosowania
- minikomputerze PC
- domowym serwerze
- wynajętym serwerze fizycznym, jeśli dostawca pozwala używać własnych nośników instalacyjnych

Edycja HWS dodaje obsługę sprzętu fizycznego, w tym integrację z zasilaczami UPS przez NUT, narzędzia SMART i NVMe, obsługę czujników, funkcje sterowania firmware/UEFI, oprogramowanie układowe urządzeń, diagnostykę sprzętu oraz ustawienia ekranu i pokrywy laptopa.

Obie edycje mają to samo centrum sterowania JustVoxel, interfejs Management API, funkcje obsługi serwerów Minecraft, model przechowywania danych, kopie zapasowe, obsługę sieci i sposób aktualizacji.

[Poznaj możliwości edycji VM i HWS](../docs/VARIANTS.md)

## 🚀 Instalacja JustVoxel

Jeden obraz instalacyjny ISO obsługuje obie edycje JustVoxel.

[⬇️ Pobierz obraz ISO JustVoxel x86_64](https://sourceforge.net/projects/justvoxel/files/JustVoxel-x86_64.iso/download)

Wybierz edycję odpowiednią dla swojego środowiska:

- **JustVoxel VM** do maszyn wirtualnych, w tym zgodnych maszyn wirtualnych u dostawców hostingu
- **JustVoxel HWS** do komputerów stacjonarnych, laptopów, minikomputerów PC, fizycznych serwerów domowych i zgodnych serwerów fizycznych u dostawców hostingu

[Instrukcja instalacji](../docs/INSTALLER.md) opisuje instalację, przygotowanie dysku i pierwsze logowanie.

## 🏠 Przede wszystkim Minecraft, ale nie tylko

JustVoxel powstał i jest wspierany przede wszystkim jako system do prowadzenia serwerów Minecraft. U jego podstaw nadal działa standardowy **serwer AlmaLinux 10 bootc z narzędziami Podman, Quadlet, systemd, SSH, NetworkManager i typowymi narzędziami administracyjnymi Linuksa**.

Zaawansowani użytkownicy mogą wykorzystać tę samą maszynę jako serwer domowy o szerszym zastosowaniu i uruchamiać dodatkowe kontenery obok serwerów Minecraft.

Przykładowe usługi, które można wdrożyć ręcznie:

- **Jellyfin** lub **Plex** do multimediów
- **Vaultwarden** do zarządzania hasłami
- **Immich** do zarządzania zdjęciami i filmami
- **Home Assistant Container** do automatyki domowej
- **Nextcloud** do przechowywania plików i współpracy
- **Syncthing** lub **Navidrome**
- **OpenClaw** jako samodzielnie hostowany agent AI lub brama AI

**Centrum sterowania JustVoxel nie zarządza tymi aplikacjami**. Za ich kontenery Podman, jednostki Quadlet, przestrzeń dyskową, porty, aktualizacje, kopie zapasowe, konfigurację odwrotnego proxy i spełnienie wymagań poszczególnych aplikacji odpowiada administrator.

Podział obowiązków jest celowo prosty: JustVoxel zarządza systemem serwerowym Minecraft, a do innych zastosowań zaawansowani użytkownicy mogą korzystać ze standardowych narzędzi Podman i Quadlet.

Również podczas prac konserwacyjnych JustVoxel traktuje niezwiązane z grą kontenery jako usługi administratora, odrębne od zasobów Minecraft. Operacje takie jak **Start Over** w sekcji Minecraft nie usuwają niezwiązanych z grą kontenerów Podman.

Maszyna z JustVoxel może więc służyć przede wszystkim do hostowania serwerów Minecraft, a w razie potrzeby pełnić też funkcję zwykłego serwera domowego z usługami w kontenerach.

## 🎮 Stwórz własny serwer Minecraft

JustVoxel może jednocześnie uruchamiać kilka niezależnych serwerów Minecraft, jeśli maszyna ma wystarczająco dużo pamięci i wolnych portów.

Każdy serwer ma własny identyfikator, konfigurację, środowisko uruchomieniowe, porty, dane, kopie zapasowe, funkcje sterowania jego pracą oraz informacje o stanie.

### Obecnie obsługiwane oprogramowanie serwerowe

| Oprogramowanie serwera | Rodzaj dodatków | Zarządzana rozgrywka międzyplatformowa Java → Bedrock |
| --- | --- | --- |
| **Paper** | Wtyczki | ✅ |
| **Purpur** | Wtyczki | ✅ |
| **Spigot** | Wtyczki | ✅ |
| **Pufferfish** | Wtyczki | ✅ |
| **Leaf** | Wtyczki | ✅ |
| **Fabric** | Mody | ✅ Po potwierdzeniu zgodności konkretnej konfiguracji |
| **NeoForge** | Mody | ✅ Po potwierdzeniu zgodności konkretnej konfiguracji |
| **Forge** | Mody | Tylko Java |
| **Quilt** | Mody | Tylko Java |
| **Vanilla Java** | Bez wtyczek i modów | Tylko Java |
| **Native Bedrock** | Natywny serwer Bedrock | Natywne klienty Bedrock |

Wybierz **konfigurację prostą (Simple Setup)**, aby krok po kroku utworzyć pierwszy serwer. **Konfiguracja zaawansowana (Advanced Setup)** daje większą kontrolę nad pamięcią, portami, lokalizacją danych, kopiami zapasowymi, zasadami wyboru wersji i obsługą zgodnej rozgrywki międzyplatformowej.

JustVoxel sprawdza dostępność funkcji dla wybranego oprogramowania serwerowego. Nie zakłada, że każda funkcja działa na każdym serwerze.

[Szczegóły dotyczące oprogramowania serwerowego i zgodności](../docs/server-software.md)

## 🧩 Wtyczki i mody

Dla wybranego serwera otwórz **Minecraft → Wtyczki i mody**.

W zależności od typu serwera JustVoxel może korzystać z następujących katalogów:

- [Modrinth](https://modrinth.com/)
- [Hangar](https://hangar.papermc.io/)
- [CurseForge](https://www.curseforge.com/minecraft) po skonfigurowaniu własnego klucza API

Przy instalowaniu obsługiwanych dodatków JustVoxel sprawdza zgodność, weryfikuje pobrane pliki, tworzy kopie zapasowe przy zatrzymanym serwerze, sprawdza ponowne uruchomienie, śledzi przynależność plików i stosuje zabezpieczenia umożliwiające odzyskiwanie.

Serwery z rodziny Paper mogą korzystać z katalogów wtyczek. Fabric, Forge, NeoForge i Quilt korzystają z katalogów modów ze ścisłą weryfikacją programu ładującego mody i wersji Minecraft. Dla serwerów Vanilla Java i natywnych serwerów Bedrock katalog wtyczek i modów celowo nie jest dostępny.

[Więcej o wtyczkach i modach](../docs/plugin-workspace.md)

## 🌐 Opcjonalna sieć Minecraft Network

Możesz prowadzić kilka niezależnych serwerów. **Nie** wymagają one serwera proxy.

Jeśli chcesz udostępniać kilka zgodnych serwerów pod jednym publicznym adresem, JustVoxel oferuje również opcjonalną funkcję **Minecraft Network** opartą na Velocity.

W centrum sterowania możesz wybrać obsługiwane serwery zaplecza, skonfigurować publiczne porty Java i Bedrock, zarządzać wirtualnym lobby i wiadomością powitalną oraz decydować, które serwery należą do sieci.

Paper, Purpur, Pufferfish i Leaf są obsługiwane jako serwery zaplecza Minecraft Network. Serwery Fabric/Quilt i Forge/NeoForge również mogą dołączyć, jeśli JustVoxel potwierdzi wymaganą zgodność z proxy. Spigot, Vanilla i natywny Bedrock działają samodzielnie.

[Więcej o Minecraft Network](../docs/minecraft-network.md)

## 🖥️ Poznaj centrum sterowania

JustVoxel skupia się na zadaniach związanych z obsługą serwera. Znajomość wewnętrznych mechanizmów Linuksa nie jest potrzebna.

### Serwery Minecraft

Twórz serwery i zarządzaj nimi. Możesz je uruchamiać i zatrzymywać, sprawdzać ich stan i listę graczy, zmieniać obsługiwane ustawienia oraz korzystać z narzędzi dla poszczególnych serwerów.

### Wtyczki i mody

Wyszukuj zgodne dodatki, przeglądaj zainstalowane wtyczki i mody oraz zarządzaj obsługiwanymi dodatkami. Jeśli zmiana wymaga interwencji, znajdziesz tu informacje o odzyskiwaniu.

### Kopie zapasowe i pamięć masowa

Twórz kopie zapasowe, przeglądaj ich historię, przywracaj komplet danych serwera i konfiguruj miejsca zapisu kopii. Możesz też importować i eksportować serwery oraz zarządzać obsługiwaną pamięcią masową.

### Sieć

Zarządzaj połączeniami Ethernet i Wi-Fi oraz dostępem zdalnym. Korzystaj z narzędzi do **diagnozowania problemów z połączeniem**. Zmiany, które mogą przerwać połączenie, są chronione mechanizmem wycofywania, dzięki czemu błędna zmiana adresu IP lub ustawień Wi-Fi nie musi oznaczać trwałej utraty dostępu.

### System

Sprawdzaj stan systemu, wykorzystanie zasobów, dzienniki i aktualizacje systemu operacyjnego. Masz też dostęp do ustawień użytkowników i bezpieczeństwa, obsługiwanych operacji zasilania oraz funkcji sprzętowych dostępnych tylko w edycji HWS.

### Połączenia AI i API

Podłącz obsługiwane aplikacje AI, klientów MCP, automatyzację przez REST, Telegram i Discord bez tworzenia osobnego systemu zarządzania.

Wolisz terminal? JustVoxel zawiera też **mjust**, interaktywny terminalowy interfejs administracyjny korzystający z tych samych zabezpieczeń.

[Dokumentacja WebUI](../docs/WEBUI.md) · [Przewodnik po mjust](../docs/MJUST.md)

## 🌐 Wybierz, jak znajomi połączą się z serwerem

JustVoxel udostępnia obecnie cztery rozwiązania w sekcji **Sieć → Dostęp zdalny**.

Integracja wykracza poza dołączenie pakietów do obrazu. W zależności od usługi JustVoxel umożliwia sprawdzanie stanu, włączanie i wyłączanie integracji oraz jej konfigurację.

| Rozwiązanie | Kiedy się sprawdzi | Co oferuje |
| --- | --- | --- |
| **[Tailscale](https://tailscale.com/)** | Prywatne serwery dla znajomych i rodziny | Prywatna sieć typu mesh między zatwierdzonymi urządzeniami |
| **[NetBird](https://netbird.io/)** | Prywatne sieci z zarządzaniem dostępem | Prywatna sieć z kontrolą kont i zasad dostępu |
| **[Playit.gg](https://playit.gg/)** | Publiczny dostęp do serwera Minecraft bez typowego przekierowania portów na routerze | Publiczne tunele do gry i adresy połączeń |
| **[NordVPN Meshnet](https://meshnet.nordvpn.com/)** | Prywatne połączenia między urządzeniami użytkowników NordVPN | Łączność przez Meshnet bez używania JustVoxel jako klienta komercyjnej usługi VPN |

### Które rozwiązanie wybrać?

**🎮 Chcesz, aby znajomi dołączali bez instalowania klienta prywatnej sieci VPN?**  
Zacznij od **Playit.gg**. Gracze mogą korzystać z publicznego adresu skonfigurowanego tunelu do gry.

**🔒 Chcesz mieć prywatny serwer dla zaufanych urządzeń?**  
Rozważ **Tailscale** lub **NetBird**.

**🌍 Korzystasz już z NordVPN Meshnet?**  
JustVoxel może zarządzać integracją z Meshnet. Jest to funkcja odrębna od łączenia urządzenia ze standardowym komercyjnym serwerem wyjściowym NordVPN.

Kontami, zasadami w chmurze i konfiguracją tuneli nadal zarządza się po stronie dostawcy. JustVoxel odpowiada za obsługiwane funkcje integracji po stronie urządzenia.

Można korzystać równocześnie z kilku rozwiązań VPN lub mesh, ale trasy sieciowe, ustawienia DNS lub reguły zapory mogą ze sobą kolidować.

[Przewodnik po sieci i dostępie zdalnym](../docs/NETWORKING.md)

## 🤖 AI, MCP, REST i automatyzacja

Chcesz zapytać asystenta AI o stan serwera, podłączyć automatykę domową lub opracować własny sposób sterowania?

JustVoxel udostępnia wbudowane **interfejsy MCP i REST** w sekcji **Połączenia AI i API**.

Podczas konfiguracji możesz wybrać zgodną aplikację lub typ połączenia, na przykład:

- **OpenAI — ChatGPT i Codex**
- **Claude**
- **Grok**
- **Gemini**
- **GitHub Copilot**
- **Inna aplikacja AI / MCP**
- **Automatyzacja / REST API**

Ty decydujesz, do czego ma dostęp każde połączenie.

Zakres uprawnień obejmuje **poziom 1 — tylko do odczytu**, funkcje zarządzania serwerami, a także zaawansowane sterowanie systemem, które wymaga wyraźnej zgody. Dostęp danego połączenia można również ograniczyć do wybranych serwerów Minecraft.

JustVoxel nie potrzebuje klucza API modelu AI do samego udostępnienia interfejsu MCP lub REST. Możliwość połączenia po stronie klienta zależy od aplikacji, możliwości konta lub subskrypcji oraz dostępu do sieci.

Brama AI/API jest domyślnie wyłączona. Aby z niej korzystać, trzeba ją włączyć.

[Więcej o połączeniach AI i API](../docs/AI-API-CONNECTIONS.md)

## 💬 Telegram i Discord

JustVoxel oferuje opcjonalne, wbudowane integracje z usługami Telegram i Discord. Korzystają z tego samego centralnego mechanizmu zarządzania co WebUI.

### Telegram

Dedykowany bot na Telegramie może udostępniać:

- informacje o stanie urządzenia i serwerów Minecraft
- wybór serwera i informacje o graczach
- uruchamianie, zatrzymywanie i ponowne uruchamianie serwera przez użytkowników upoważnionych do zarządzania
- ręczne tworzenie kopii zapasowych
- przegląd aktualizacji Minecraft i kontenerów
- informacje o aktualizacjach systemu operacyjnego
- kontrolowane ponowne uruchamianie urządzenia
- opcjonalne powiadomienia o stanie serwera, wynikach tworzenia kopii zapasowych i dostępnych aktualizacjach

Integracja z Telegramem ma własne uprawnienia **Tylko do odczytu** i **Zarządzanie serwerem** i nie wymaga modelu AI.

[Konfiguracja Telegrama i uprawnień](../docs/TELEGRAM.md)

### Discord

Discord może służyć zarówno społeczności graczy Minecraft, jak i administratorom serwera.

Członkowie społeczności na wskazanym i powiązanym serwerze Discord mogą sprawdzać podstawowe informacje o stanie serwera i prosić o dostęp do gry bez automatycznego uzyskiwania uprawnień administracyjnych.

Zatwierdzeni administratorzy integracji JustVoxel z Discordem mogą uruchamiać, zatrzymywać i ponownie uruchamiać serwery, ręcznie tworzyć kopie zapasowe, przeglądać aktualizacje, zarządzać Minecraft Network i wykonywać dodatkowe zaawansowane operacje.

Członkostwo na serwerze Discord, wpis na liście graczy dopuszczonych do serwera Minecraft i uprawnienia administracyjne JustVoxel są od siebie niezależne.

[Konfiguracja Discorda, dostęp społeczności i administracja](../docs/DISCORD.md)

## 💾 Kopie zapasowe, przywracanie, pamięć masowa i migracja

Dane serwerów Minecraft są ważniejsze od obrazu systemu.

JustVoxel przechowuje dane Minecraft i kopie zapasowe poza niezmiennym obrazem zainstalowanego systemu operacyjnego. Miejsca zapisu danych i kopii wybiera się niezależnie.

Obsługiwane miejsca przechowywania danych to:

- system plików systemu operacyjnego
- dedykowane dyski wewnętrzne
- pamięć masowa USB
- istniejące lokalne partycje lub systemy plików
- nowe partycje tworzone z miejsca, które już jest nieprzydzielone
- NFS
- SMB/CIFS

Dla każdego serwera można tworzyć kopie zapasowe automatycznie i ręcznie. Przywrócenie kopii zastępuje wszystkie trwałe dane wybranego serwera. Jego tożsamość w JustVoxel i konfiguracja w systemie pozostają zachowane.

Do przenoszenia serwerów między maszynami lub odtwarzania ich po ponownej instalacji JustVoxel udostępnia też procedury **eksportu, importu i odzyskiwania** z etapem weryfikacji.

[Pamięć masowa](../docs/STORAGE.md) · [Przywracanie](../docs/RESTORE.md) · [Migracja](../docs/MIGRATION.md)

## 🛡️ Bezpieczniejsze wprowadzanie zmian

JustVoxel to więcej niż atrakcyjna strona, która uruchamia dowolne polecenia powłoki.

Obsługiwane operacje korzystają z zabezpieczeń na poziomie całego urządzenia, takich jak:

- sprawdzanie obecności graczy przed operacjami, które mogą przerwać grę
- przegląd zmian przed ich zastosowaniem
- tworzenie kopii przy zatrzymanym serwerze, gdy zmiana wymaga możliwości odzyskania danych
- jednoznaczna identyfikacja serwerów, która zapobiega przypadkowemu zastąpieniu jednego serwera innym
- weryfikacja ponownego uruchomienia i działania serwera
- trwały zapis stanu operacji i odzyskiwania
- możliwość wycofania obsługiwanych operacji
- punkty kontrolne NetworkManager przy ryzykownych zmianach ustawień sieci
- osobne procedury aktualizacji Minecraft i systemu operacyjnego

Interfejs WebUI dostępny w przeglądarce działa bez podwyższonych uprawnień. Zmiany wymagające takich uprawnień przechodzą przez lokalny JustVoxel Management Agent i jego API z określonym zestawem operacji. Przeglądarka nie ma dostępu do powłoki użytkownika root.

## 🔄 Aktualizacje systemu niezależne od aktualizacji Minecraft

JustVoxel to system zarządzany za pomocą obrazów, oparty na Home Server Base 10 / AlmaLinux 10.

Aktualizacje systemu operacyjnego są obsługiwane niezależnie od wersji Minecraft oraz aktualizacji oprogramowania serwerowego, wtyczek, modów i kontenerów.

Aktualizacja urządzenia nie zmienia więc po cichu wersji Minecraft, a wprowadzenie zmian na serwerze gry nie wymaga przebudowy systemu operacyjnego hosta.

[Dokumentacja zarządzania systemem](../docs/SYSTEM.md)

## 📊 Jak JustVoxel wypada na tle innych projektów?

JustVoxel różni się od tradycyjnych paneli do zarządzania serwerami Minecraft: **system operacyjny i warstwa zarządzania serwerami Minecraft tworzą jedno rozwiązanie**.

| Funkcja | **JustVoxel** | **MineOS SvelteKit** | **Crafty Controller** | **MineOS Node** |
| --- | :---: | :---: | :---: | :---: |
| Kompletny system operacyjny do hostowania serwerów Minecraft | ⭐ | — | — | — |
| Wiele serwerów Minecraft | ✅ | ✅ | ✅ | ✅ |
| Obsługa serwerów Java i Bedrock | ✅ | ✅ | ✅ | 🟡 Java i obsługa starszego PocketMine |
| Wyszukiwanie wtyczek i modów z pomocą panelu | ✅ Modrinth, Hangar, CurseForge | ✅ Modrinth, CurseForge | — | — |
| Zarządzanie siecią, pamięcią masową i systemem operacyjnym hosta | ⭐ | — | — | — |
| Zintegrowany dostęp prywatny i tunelowanie połączeń z grą | ⭐ 4 opcje | — | — | — |
| Brama MCP + REST dla AI i automatyzacji | ⭐ | — | — | — |
| Wbudowane zarządzanie przez Telegram i Discord | ⭐ | — | — | — |
| Sterowanie reakcją na zamknięcie pokrywy i ekranem laptopa | ✅ HWS | — | — | — |
| Aktualizacje niezmiennego systemu operacyjnego za pomocą obrazów | ✅ | — | — | — |
| Terminalowy interfejs zarządzania | ✅ mjust | ✅ mineos CLI/TUI | — | ✅ CLI |
| Sposób instalacji | Kompletny system oparty na bootc | Aplikacja Docker / Compose | Panel zarządzania w Docker, Linux lub Windows | Aplikacja Node.js w systemie Linux/BSD |

⭐ cecha wyróżniająca &nbsp;•&nbsp; ✅ funkcja obsługiwana &nbsp;•&nbsp; 🟡 obsługa częściowa lub starszych rozwiązań &nbsp;•&nbsp; — funkcji nie opisano w przejrzanej podstawowej dokumentacji projektu

**Ostatnia weryfikacja porównania: 16 listopada 2026 r.** Celem zestawienia nie jest przedstawienie innych projektów w złym świetle. MineOS i Crafty to aplikacje do zarządzania serwerami Minecraft, które mogą być świetnym wyborem, jeśli masz już system operacyjny i chcesz samodzielnie nim administrować. JustVoxel jest przeznaczony dla osób, które chcą zarządzać panelem Minecraft **wraz z systemem serwerowym, na którym ten panel działa**, jako jednym produktem.

Porównanie opiera się na aktualnej dokumentacji i repozytoriach projektów:

- [MineOS SvelteKit](https://github.com/freeman412/mineos-sveltekit)
- [Crafty Controller](https://craftycontrol.com/) · [kod źródłowy](https://gitlab.com/crafty-controller/crafty-4)
- [MineOS Node](https://github.com/hexparrot/mineos-node)

Jeśli któraś pozycja jest nieaktualna lub niesprawiedliwa, otwórz zgłoszenie w repozytorium. Mile widziane są również poprawki, po których JustVoxel wypadnie mniej korzystnie.

## 📚 Dokumentacja

Szczegółowe przewodniki znajdziesz w [katalogu dokumentacji](../docs/).

Na początek zajrzyj do następujących przewodników:

- [WebUI](../docs/WEBUI.md)
- [Środowisko uruchomieniowe Minecraft](../docs/MINECRAFT_RUNTIME.md)
- [Oprogramowanie serwerowe](../docs/server-software.md)
- [Wtyczki i mody](../docs/plugin-workspace.md)
- [Minecraft Network](../docs/minecraft-network.md)
- [Sieć i dostęp zdalny](../docs/NETWORKING.md)
- [Połączenia AI i API](../docs/AI-API-CONNECTIONS.md)
- [Telegram](../docs/TELEGRAM.md)
- [Discord](../docs/DISCORD.md)
- [Pamięć masowa](../docs/STORAGE.md)
- [Przywracanie](../docs/RESTORE.md)
- [Migracja](../docs/MIGRATION.md)
- [Zarządzanie systemem](../docs/SYSTEM.md)
- [mjust](../docs/MJUST.md)

Informacje o tworzeniu obrazów, testowaniu i wydaniach zawiera [przewodnik dla deweloperów](../README-DEV.md).

## ⛏️ Oprogramowanie Minecraft i umowa EULA

JustVoxel **nie zawiera** plików binarnych serwera Minecraft, oprogramowania serwerowego Mojang, gotowego świata ani uprzednio zaakceptowanej umowy licencyjnej Minecraft (EULA).

Podczas tworzenia serwera administrator akceptuje umowę EULA Minecraft, a JustVoxel przygotowuje wybrane, obsługiwane środowisko uruchomieniowe serwera.

Minecraft, oprogramowanie Mojang, Paper, Purpur, Spigot, Pufferfish, Leaf, Fabric, NeoForge, Forge, Quilt, wtyczki, mody i pozostałe składniki innych dostawców podlegają własnym licencjom i warunkom dystrybucji.

## 📄 Licencja

JustVoxel jest udostępniany na licencji **Apache-2.0**. Szczegóły zawiera plik [LICENSE](../LICENSE).

Informacje prawne dotyczące składników innych dostawców znajdują się w pliku [THIRD_PARTY_NOTICES.md](../THIRD_PARTY_NOTICES.md).
