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

> **Nota sulla traduzione:** Questa traduzione di JustVoxel è stata realizzata con l’aiuto dell’IA a partire dal testo originale inglese. Se trovi un passaggio impreciso, poco chiaro o poco naturale, proponi una correzione con una [segnalazione su GitHub](https://github.com/home-server-project/justvoxel/issues) o una [richiesta di modifica (pull request)](https://github.com/home-server-project/justvoxel/pulls).

<p align="center">
  <strong>I tuoi server Minecraft sul tuo hardware, su un server dedicato a noleggio o su una VM in hosting. Un unico Centro di controllo.</strong>
</p>

<p align="center">
  Un’appliance per ospitare i tuoi server Minecraft e gestirli senza dover amministrare anche il sistema operativo.
</p>

<p align="center">
  <a href="#-installa-justvoxel">Installazione</a> •
  <a href="#-ospitare-minecraft-diventa-più-semplice">Funzionalità</a> •
  <a href="../docs/INSTALLER.md">Documentazione</a>
</p>

<p align="center">
  <img src="../docs/images/justvoxel-overview-1.webp" width="100%" alt="Panoramica del Centro di controllo di JustVoxel"><img src="../docs/images/justvoxel-overview-2.webp" width="100%" alt=""><img src="../docs/images/justvoxel-overview-3.webp" width="100%" alt="">
</p>

JustVoxel è un’appliance dedicata al self-hosting di server Minecraft. Installala su un PC, un portatile, un server fisico, una macchina virtuale o un servizio di hosting compatibile che ti permetta di usare una tua immagine del sistema operativo. Poi gestisci Minecraft dal browser.

Nessuna raccolta di script da mantenere. Nessun pannello separato da installare per gestire Minecraft.

Hai un vecchio portatile o PC? **Con JustVoxel HWS puoi riutilizzarlo come appliance dedicata a Minecraft, invece di lasciarlo in un cassetto.** Sui portatili compatibili puoi disattivare la sospensione alla chiusura del coperchio, così il server continua a funzionare anche da chiuso. Puoi anche spegnere lo schermo della console locale subito o dopo un breve intervallo.

> 🚧 **Versione preliminare:** JustVoxel si sta preparando alla prima versione pubblica. La ISO di installazione attuale è disponibile per le prove e per chi vuole iniziare a usarla mentre proseguono le verifiche finali per il rilascio.

## ✨ Ospitare Minecraft diventa più semplice

JustVoxel integra la gestione di Minecraft con il sistema operativo su cui funziona.

- 🎮 **Più server Minecraft** — esegui mondi e tipi di server diversi in modo indipendente sulla stessa appliance.
- 🖥️ **Centro di controllo web** — gestisci i server da computer, tablet o telefono.
- 🧩 **Plugin e mod** — trova e gestisci contenuti compatibili dai fornitori supportati.
- 💾 **Backup e ripristino** — proteggi ogni server con backup pianificati e manuali.
- 🌐 **Accesso remoto** — scegli tra quattro opzioni integrate per l’accesso privato e i tunnel di gioco.
- 🤖 **IA e automazione** — collega assistenti IA e applicazioni compatibili tramite MCP e REST.
- 💬 **Telegram e Discord** — comandi dei bot, notifiche e funzionalità per la community, tutti facoltativi.
- 🛡️ **Gestione del sistema** — gestisci rete, archiviazione, aggiornamenti, stato del sistema, log e comandi di alimentazione supportati senza dover imparare i relativi comandi Linux.

JustVoxel è pensato per famiglie, gruppi di amici, piccole community di Minecraft e per chi vuole avere un server proprio invece di noleggiare un tradizionale servizio di hosting Minecraft con pannello di controllo.

## 🖥️ Una sola appliance, due edizioni

### JustVoxel VM

L’edizione VM è pensata per macchine virtuali e ambienti in hosting come quelli elencati di seguito.

- Proxmox
- KVM / libvirt
- VMware
- Hyper-V
- VirtualBox
- VM in hosting o VPS che consentono di usare un’immagine personalizzata compatibile del sistema operativo

L’edizione VM include l’appliance Minecraft completa ed esclude i pacchetti per l’hardware fisico che non servono in una macchina virtuale.

### JustVoxel HWS

L’edizione HWS è pensata per l’installazione diretta su hardware fisico.

- un vecchio portatile
- un PC fisso riconvertito
- un mini PC
- un server domestico
- un server bare metal a noleggio che ammette supporti di installazione personalizzati

HWS aggiunge funzionalità per le macchine fisiche, tra cui l’integrazione con i gruppi di continuità (UPS) tramite NUT, gli strumenti SMART e NVMe, i sensori, i controlli del firmware e di UEFI, il firmware dei dispositivi, la diagnostica hardware e i controlli dello schermo e del coperchio dei portatili.

Entrambe le edizioni condividono il Centro di controllo di JustVoxel, la Management API, le procedure di gestione di Minecraft, il modello di archiviazione, i backup e la gestione della rete e degli aggiornamenti.

[Scopri le funzionalità delle edizioni VM e HWS](../docs/VARIANTS.md)

## 🚀 Installa JustVoxel

Una sola ISO di installazione supporta entrambe le edizioni di JustVoxel.

[⬇️ Scarica la ISO x86_64 di JustVoxel](https://sourceforge.net/projects/justvoxel/files/JustVoxel-x86_64.iso/download)

Scegli l’edizione adatta all’ambiente di installazione.

- **JustVoxel VM** per macchine virtuali e ambienti di hosting di VM compatibili
- **JustVoxel HWS** per PC, portatili, mini PC, server domestici fisici e hosting bare metal compatibile

Consulta la [guida all’installazione](../docs/INSTALLER.md) per installare il sistema, preparare i dischi ed effettuare il primo accesso.

## 🏠 Pensato per Minecraft, aperto ad altri usi

JustVoxel è sviluppato e supportato principalmente come appliance per server Minecraft. Alla base c’è un normale **server AlmaLinux 10 bootc con Podman, Quadlet, systemd, SSH, NetworkManager e gli strumenti standard di amministrazione Linux**.

Gli utenti esperti possono usare la stessa macchina come server domestico per altri servizi ed eseguire altri container accanto a Minecraft.

Puoi installare manualmente servizi come questi.

- **Jellyfin** o **Plex** per i contenuti multimediali
- **Vaultwarden** per la gestione delle password
- **Immich** per la gestione di foto e video
- **Home Assistant Container** per la domotica
- **Nextcloud** per i file e la collaborazione
- **Syncthing** o **Navidrome**
- **OpenClaw** come agente o gateway IA self-hosted

Queste applicazioni **non sono gestite dal Centro di controllo di JustVoxel**. L’amministratore gestisce direttamente i relativi container Podman, le unità Quadlet, l’archiviazione, le porte, gli aggiornamenti, i backup, la configurazione del reverse proxy e i requisiti specifici di ciascuna applicazione.

JustVoxel gestisce l’appliance Minecraft. Per gli altri servizi, gli utenti esperti possono continuare a usare le normali procedure di Podman e Quadlet.

Anche durante la manutenzione, JustVoxel considera i container estranei a Minecraft di competenza dell’amministratore. Operazioni come **Ricomincia** in Minecraft non rimuovono i container Podman estranei a Minecraft.

La macchina può quindi continuare a funzionare principalmente come appliance Minecraft e, quando serve, fare anche da normale server domestico per applicazioni in container.

## 🎮 Crea il server Minecraft che vuoi

JustVoxel può eseguire contemporaneamente più server Minecraft indipendenti, purché la macchina disponga di memoria sufficiente e di porte libere.

Ogni server ha una propria identità, configurazione, ambiente di esecuzione, porte, dati, backup, comandi per il ciclo di vita e stato.

### Software server attualmente supportato

| Software server | Tipo di contenuti | Cross-play Java → Bedrock gestito |
| --- | --- | --- |
| **Paper** | Plugin | ✅ |
| **Purpur** | Plugin | ✅ |
| **Spigot** | Plugin | ✅ |
| **Pufferfish** | Plugin | ✅ |
| **Leaf** | Plugin | ✅ |
| **Fabric** | Mod | ✅ Se viene verificata la compatibilità esatta |
| **NeoForge** | Mod | ✅ Se viene verificata la compatibilità esatta |
| **Forge** | Mod | Solo Java |
| **Quilt** | Mod | Solo Java |
| **Vanilla Java** | Nessun plugin o mod | Solo Java |
| **Bedrock nativo** | Server Bedrock nativo | Client Bedrock nativi |

Scegli **Configurazione semplice** per creare il primo server con una procedura guidata, oppure **Configurazione avanzata** per avere più controllo su memoria, porte, archiviazione, backup, criteri di gestione delle versioni e opzioni di cross-play compatibili.

JustVoxel verifica la compatibilità di ogni funzionalità con il software server selezionato.

[Consulta i dettagli sul software server e sulla compatibilità](../docs/server-software.md)

## 🧩 Plugin e mod

Apri **Minecraft → Plugin e mod** per il server selezionato.

A seconda del tipo di server, JustVoxel può usare i seguenti cataloghi.

- [Modrinth](https://modrinth.com/)
- [Hangar](https://hangar.papermc.io/)
- [CurseForge](https://www.curseforge.com/minecraft) con la chiave API che hai configurato

Le installazioni supportate prevedono controlli di compatibilità, download verificati, backup a server spento, verifica del riavvio, registrazione della titolarità dei contenuti installati e misure di protezione per il recupero.

I server della famiglia Paper possono usare i cataloghi di plugin. Fabric, Forge, NeoForge e Quilt usano cataloghi di mod con controlli puntuali sul loader e sulla versione di Minecraft. Per scelta progettuale, Vanilla Java e Bedrock nativo non offrono un catalogo di plugin o mod.

[Scopri come gestire plugin e mod](../docs/plugin-workspace.md)

## 🌐 Rete Minecraft facoltativa

Puoi usare più server autonomi. **Non** serve un proxy.

Se vuoi un unico punto di accesso pubblico per più server compatibili, JustVoxel offre anche una **Rete Minecraft** facoltativa basata su Velocity.

Dal Centro di controllo puoi selezionare i server di backend supportati, configurare le porte pubbliche per Java e Bedrock, gestire la lobby virtuale e il messaggio di benvenuto e scegliere quali server fanno parte della rete.

Paper, Purpur, Pufferfish e Leaf sono supportati come backend della rete. Possono partecipare anche server Fabric/Quilt e Forge/NeoForge compatibili, se JustVoxel verifica la compatibilità richiesta con il proxy. Spigot, Vanilla e Bedrock nativo continuano a funzionare come server autonomi.

[Scopri la Rete Minecraft](../docs/minecraft-network.md)

## 🖥️ Il Centro di controllo

JustVoxel organizza il Centro di controllo intorno alle operazioni che puoi eseguire sui server, senza richiedere conoscenze dei meccanismi interni di Linux.

### Server Minecraft

Crea e gestisci più server, avviali o arrestali, controlla lo stato e i giocatori, modifica le impostazioni supportate e accedi agli strumenti specifici di ogni server.

### Plugin e mod

Trova contenuti compatibili, controlla i plugin e le mod installati, gestisci le aggiunte supportate e consulta le informazioni sul recupero quando una modifica richiede attenzione.

### Backup e archiviazione

Crea backup, consulta la cronologia, ripristina tutti i dati di un server, configura le destinazioni dei backup, importa ed esporta server e gestisci le risorse di archiviazione supportate.

### Rete

Gestisci Ethernet, Wi-Fi, **la risoluzione dei problemi di connessione** e l’accesso remoto. Le modifiche di rete che potrebbero interrompere la connessione prevedono il ripristino della configurazione precedente, così un’impostazione IP o Wi-Fi errata non ti lascia definitivamente senza accesso.

### Sistema

Controlla lo stato del sistema, le risorse, i log, gli aggiornamenti del sistema operativo, gli utenti e le impostazioni di sicurezza. Puoi anche eseguire le operazioni di alimentazione supportate e, solo su HWS, accedere ai controlli hardware.

### Connessioni IA e API

Collega applicazioni IA supportate, client MCP, automazioni via REST, Telegram e Discord senza dover predisporre un altro sistema di gestione.

Preferisci il terminale? JustVoxel include anche **mjust**, un’interfaccia interattiva di amministrazione da terminale che adotta lo stesso modello di sicurezza dell’appliance.

[Documentazione della WebUI](../docs/WEBUI.md) · [Guida a mjust](../docs/MJUST.md)

## 🌐 Scegli come far collegare i tuoi amici

JustVoxel integra attualmente quattro opzioni in **Rete → Accesso remoto**.

L’integrazione consente di consultare lo stato e offre procedure supportate per attivare, disattivare e, dove previsto, configurare il servizio.

| Opzione | Ideale per | Cosa offre |
| --- | --- | --- |
| **[Tailscale](https://tailscale.com/)** | Server privati per amici e familiari | Rete mesh privata tra dispositivi autorizzati |
| **[NetBird](https://netbird.io/)** | Reti private con accesso gestito | Rete privata con controllo degli account e delle regole di accesso |
| **[Playit.gg](https://playit.gg/)** | Accesso pubblico a Minecraft senza il consueto inoltro delle porte sul router | Tunnel di gioco pubblici e indirizzi di connessione |
| **[NordVPN Meshnet](https://meshnet.nordvpn.com/)** | Accesso privato tra dispositivi per gli utenti di NordVPN | Connettività Meshnet senza usare JustVoxel come client VPN commerciale |

### Quale opzione scegliere?

**🎮 I tuoi amici vogliono collegarsi senza installare un client VPN privato?**  
Parti da **Playit.gg**. I giocatori possono usare l’indirizzo pubblico del tunnel di gioco che hai configurato.

**🔒 Vuoi un server privato per dispositivi fidati?**  
Valuta **Tailscale** o **NetBird**.

**🌍 Usi già NordVPN Meshnet?**  
JustVoxel può gestire l’integrazione con Meshnet. Questa funzione è distinta dal collegamento dell’appliance a un normale server di uscita del servizio VPN commerciale di NordVPN.

Gli account, le regole configurate nel cloud e la configurazione dei tunnel restano di competenza del fornitore. JustVoxel gestisce l’integrazione supportata sul lato dell’appliance.

È possibile usare più servizi VPN o mesh contemporaneamente, ma le impostazioni di routing, DNS o firewall possono entrare in conflitto.

[Leggi la guida alla rete e all’accesso remoto](../docs/NETWORKING.md)

## 🤖 IA, MCP, REST e automazione

Vuoi chiedere lo stato del server a un assistente IA, collegare il tuo sistema di domotica o creare una procedura di controllo personalizzata?

JustVoxel offre **interfacce MCP e REST** integrate, disponibili in **Connessioni IA e API**.

Nella configurazione puoi scegliere tra applicazioni compatibili e tipi di connessione come questi.

- **OpenAI — ChatGPT e Codex**
- **Claude**
- **Grok**
- **Gemini**
- **GitHub Copilot**
- **Altra app IA / MCP**
- **Automazione / API REST**

Decidi tu a cosa può accedere ogni connessione.

I permessi vanno dal **Livello 1 — Sola lettura** alla gestione dei server e, solo con approvazione esplicita, al controllo avanzato del sistema. Puoi anche limitare le connessioni a determinati server Minecraft.

Per esporre l’interfaccia MCP o REST, JustVoxel non ha bisogno di una chiave API per un modello di IA. La compatibilità effettiva dipende dall’applicazione, dalle funzionalità dell’account o dell’abbonamento e dall’accesso alla rete.

Il gateway IA e API è disattivato per impostazione predefinita. Puoi attivarlo quando ti serve.

[Scopri le Connessioni IA e API](../docs/AI-API-CONNECTIONS.md)

## 💬 Telegram e Discord

JustVoxel include integrazioni native e facoltative con Telegram e Discord. Entrambe si appoggiano allo stesso backend di gestione della WebUI, che resta il riferimento per tutte le operazioni.

### Telegram

Un bot Telegram dedicato può offrire queste funzioni.

- stato dell’appliance e di Minecraft
- selezione del server e informazioni sui giocatori
- avvio, arresto e riavvio per gli utenti autorizzati alla gestione
- backup manuali
- verifica degli aggiornamenti di Minecraft e dei container
- informazioni sugli aggiornamenti del sistema operativo
- procedure di riavvio controllato
- notifiche facoltative sullo stato del server, sull’esito dei backup e sulla disponibilità di aggiornamenti

Telegram usa i propri permessi **Sola lettura** e **Gestione server** e non richiede un modello di IA.

[Configurazione e permessi di Telegram](../docs/TELEGRAM.md)

### Discord

Discord può offrire sia funzionalità Minecraft per la community sia strumenti di amministrazione.

I membri della community sul server Discord associato esplicitamente possono vedere le informazioni di base sullo stato del server e richiedere l’accesso a Minecraft, senza ricevere automaticamente i privilegi di amministrazione.

Gli amministratori di JustVoxel autorizzati su Discord possono eseguire le operazioni supportate sui server, tra cui avvio, arresto, riavvio, backup manuale, verifica degli aggiornamenti, gestione della Rete Minecraft e altre operazioni avanzate.

L’appartenenza al server Discord, l’accesso tramite la lista degli utenti autorizzati di Minecraft e l’amministrazione di JustVoxel restano permessi distinti.

[Configurazione di Discord, accesso per la community e amministrazione](../docs/DISCORD.md)

## 💾 Backup, ripristino, archiviazione e migrazione

I dati di Minecraft contano più dell’immagine dell’appliance.

JustVoxel conserva i dati di Minecraft e i backup al di fuori dell’installazione immutabile del sistema operativo e ti permette di scegliere separatamente dove salvarli.

Puoi scegliere tra queste opzioni di archiviazione.

- il file system di sistema
- dischi interni dedicati
- dispositivi di archiviazione USB
- partizioni o file system locali esistenti
- nuove partizioni create nello spazio non ancora allocato
- NFS
- SMB/CIFS

Ogni server può usare backup automatici e manuali. Il ripristino sostituisce tutti i dati persistenti del server selezionato e ne conserva l’identità in JustVoxel e la configurazione nell’appliance.

Per trasferire i server tra macchine o in caso di reinstallazione, JustVoxel offre anche procedure di **esportazione, importazione e recupero** sottoposte a revisione.

[Archiviazione](../docs/STORAGE.md) · [Ripristino](../docs/RESTORE.md) · [Migrazione](../docs/MIGRATION.md)

## 🛡️ Misure di sicurezza per le modifiche

La WebUI di JustVoxel gestisce le operazioni tramite l’appliance, senza eseguire comandi di shell arbitrari.

Le operazioni supportate adottano misure come queste.

- verifica della presenza di giocatori prima delle operazioni che possono interrompere il servizio
- revisione delle modifiche prima di selezionare Applica
- backup a server spento quando serve la possibilità di recuperare i dati dopo una modifica
- identificazione univoca dei server per evitare di confonderli
- verifica del riavvio e del funzionamento
- conservazione dello stato delle operazioni e del recupero
- possibilità di annullare le modifiche per le operazioni supportate
- checkpoint di NetworkManager per le modifiche di rete rischiose
- procedure distinte per gli aggiornamenti di Minecraft e del sistema operativo

La WebUI a cui accedi dal browser resta priva di privilegi. Le modifiche che richiedono privilegi passano attraverso l’agente di gestione locale di JustVoxel, tramite la sua API dedicata. Il browser non ha accesso a una shell di root.

## 🔄 Aggiornamenti di sistema separati da quelli di Minecraft

JustVoxel è un’appliance gestita tramite immagini, basata su Home Server Base 10 / AlmaLinux 10.

Gli aggiornamenti del sistema operativo sono gestiti separatamente dalle versioni di Minecraft e dagli aggiornamenti del software server, dei plugin, delle mod e dei container.

Aggiornare l’appliance non cambia la versione di Minecraft senza avvisarti. Modificare un server Minecraft non richiede di ricreare l’immagine del sistema operativo host.

[Documentazione sulla gestione del sistema](../docs/SYSTEM.md)

## 📊 Come si confronta JustVoxel con gli altri progetti?

A differenza di un pannello di controllo tradizionale per Minecraft, JustVoxel riunisce **il sistema operativo e gli strumenti di gestione di Minecraft in un’unica appliance**.

| Funzionalità | **JustVoxel** | **MineOS SvelteKit** | **Crafty Controller** | **MineOS Node** |
| --- | :---: | :---: | :---: | :---: |
| Sistema operativo completo per un’appliance Minecraft | ⭐ | — | — | — |
| Più server Minecraft | ✅ | ✅ | ✅ | ✅ |
| Supporto per server Java e Bedrock | ✅ | ✅ | ✅ | 🟡 Java e supporto legacy per PocketMine |
| Ricerca guidata di plugin e mod | ✅ Modrinth, Hangar, CurseForge | ✅ Modrinth, CurseForge | — | — |
| Gestione della rete, dell’archiviazione e del sistema operativo host | ⭐ | — | — | — |
| Accesso privato e tunnel di gioco integrati | ⭐ 4 opzioni | — | — | — |
| Gateway MCP e REST per IA e automazione | ⭐ | — | — | — |
| Gestione nativa tramite Telegram e Discord | ⭐ | — | — | — |
| Controlli del coperchio e dello schermo dei portatili | ✅ HWS | — | — | — |
| Aggiornamenti del sistema operativo immutabile tramite immagini | ✅ | — | — | — |
| Interfaccia di gestione da terminale | ✅ mjust | ✅ mineos CLI/TUI | — | ✅ CLI |
| Modalità di installazione | Appliance bootc completa | Applicazione Docker / Compose | Pannello di controllo su Docker, Linux o Windows | Applicazione Node.js su Linux/BSD |

⭐ punto di forza &nbsp;•&nbsp; ✅ supportato &nbsp;•&nbsp; 🟡 parziale o legacy &nbsp;•&nbsp; — assente dalla documentazione principale del progetto consultata per il confronto

**Ultima revisione del confronto il 16 novembre 2026.** Il confronto mette in evidenza a chi si rivolge ciascun progetto. MineOS e Crafty sono applicazioni per la gestione di Minecraft che possono essere ottime scelte se hai già un sistema operativo che vuoi amministrare in autonomia. JustVoxel si rivolge a chi vuole gestire il pannello di Minecraft **e l’appliance su cui funziona** come un unico prodotto.

Il confronto si basa sulla documentazione e sui repository attuali dei progetti.

- [MineOS SvelteKit](https://github.com/freeman412/mineos-sveltekit)
- [Crafty Controller](https://craftycontrol.com/) · [codice sorgente](https://gitlab.com/crafty-controller/crafty-4)
- [MineOS Node](https://github.com/hexparrot/mineos-node)

Se una voce del confronto è obsoleta o poco imparziale, apri una issue. Sono benvenute anche le correzioni a sfavore di JustVoxel.

## 📚 Documentazione

Le guide dettagliate dell’appliance si trovano nella [cartella della documentazione](../docs/).

Puoi partire da queste guide.

- [WebUI](../docs/WEBUI.md)
- [Ambiente di esecuzione di Minecraft](../docs/MINECRAFT_RUNTIME.md)
- [Software server](../docs/server-software.md)
- [Plugin e mod](../docs/plugin-workspace.md)
- [Rete Minecraft](../docs/minecraft-network.md)
- [Rete e accesso remoto](../docs/NETWORKING.md)
- [Connessioni IA e API](../docs/AI-API-CONNECTIONS.md)
- [Telegram](../docs/TELEGRAM.md)
- [Discord](../docs/DISCORD.md)
- [Archiviazione](../docs/STORAGE.md)
- [Ripristino](../docs/RESTORE.md)
- [Migrazione](../docs/MIGRATION.md)
- [Gestione del sistema](../docs/SYSTEM.md)
- [mjust](../docs/MJUST.md)

Per lo sviluppo e i test delle immagini e per i rilasci, consulta la [guida per gli sviluppatori](../README-DEV.md).

## ⛏️ Software Minecraft ed EULA

JustVoxel **non** distribuisce i binari del server Minecraft, il software server di Mojang, un mondo già creato o un’EULA di Minecraft accettata in anticipo.

Durante la creazione del server, l’amministratore accetta l’EULA di Minecraft e JustVoxel prepara l’ambiente di esecuzione supportato per il server selezionato.

Minecraft, il software di Mojang, Paper, Purpur, Spigot, Pufferfish, Leaf, Fabric, NeoForge, Forge, Quilt, i plugin, le mod e gli altri componenti di terze parti restano soggetti alle rispettive licenze e condizioni di distribuzione.

## 📄 Licenza

JustVoxel è distribuito con licenza **Apache-2.0**. Consulta [LICENSE](../LICENSE).

Le note relative ai componenti di terze parti sono raccolte in [THIRD_PARTY_NOTICES.md](../THIRD_PARTY_NOTICES.md).
