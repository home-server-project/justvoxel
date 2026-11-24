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

> **À propos de la traduction :** Cette traduction de JustVoxel a été réalisée à partir du texte anglais de référence avec l’aide de l’IA. Si vous repérez une formulation inexacte, peu claire ou peu naturelle, veuillez proposer une correction en ouvrant un [ticket sur GitHub](https://github.com/home-server-project/justvoxel/issues) ou en soumettant une [demande de modification](https://github.com/home-server-project/justvoxel/pulls).

<p align="center">
  <strong>Vos serveurs Minecraft. Votre propre matériel, un serveur dédié loué ou une machine virtuelle hébergée. Un seul centre de contrôle.</strong>
</p>

<p align="center">
  Un système Minecraft auto-hébergé pour les personnes qui veulent s’occuper de leur serveur de jeu sans avoir à administrer son système d’exploitation.
</p>

<p align="center">
  <a href="#-installer-justvoxel">Installation</a> •
  <a href="#-héberger-des-serveurs-minecraft-plus-facilement">Fonctionnalités</a> •
  <a href="../docs/INSTALLER.md">Documentation</a>
</p>

<p align="center">
  <img src="../docs/images/justvoxel-overview-1.webp" width="100%" alt="Aperçu du centre de contrôle JustVoxel"><img src="../docs/images/justvoxel-overview-2.webp" width="100%" alt=""><img src="../docs/images/justvoxel-overview-3.webp" width="100%" alt="">
</p>

JustVoxel est un système auto-hébergé conçu pour les serveurs Minecraft. Installez-le sur un ordinateur de bureau, un portable, un serveur physique ou une machine virtuelle compatible, ou encore chez un hébergeur qui accepte votre propre image de système d’exploitation. Vous pourrez ensuite gérer Minecraft à partir d’un navigateur.

Vous n’avez aucune collection de scripts à maintenir ni aucun panneau de gestion Minecraft distinct à installer.

Vous avez un vieux portable ou un ordinateur de bureau qui ne sert plus? **Avec JustVoxel HWS, vous pouvez en faire un serveur dédié à Minecraft au lieu de laisser du matériel encore utile dans un tiroir.** Sur les portables compatibles, vous pouvez empêcher la mise en veille à la fermeture du couvercle pour que le serveur continue de fonctionner. Les commandes d’affichage de la console locale permettent aussi d’éteindre l’écran immédiatement ou après un court délai.

> 🚧 **Préversion** — JustVoxel se prépare à sa première version publique. L’image ISO d’installation actuelle est offerte pour les essais et pour les personnes qui souhaitent commencer à l’utiliser pendant que la validation finale se poursuit.

## ✨ Héberger des serveurs Minecraft plus facilement

JustVoxel réunit la gestion de Minecraft et celle du système d’exploitation qui le fait fonctionner.

- 🎮 **Plusieurs serveurs Minecraft** — Faites fonctionner différents mondes et types de serveurs indépendamment sur la même machine.
- 🖥️ **Centre de contrôle Web** — Gérez vos serveurs à partir d’un ordinateur, d’une tablette ou d’un téléphone.
- 🧩 **Extensions et mods** — Trouvez et gérez du contenu compatible auprès des fournisseurs pris en charge.
- 💾 **Sauvegarde et restauration** — Protégez chaque serveur grâce aux sauvegardes planifiées et manuelles.
- 🌐 **Accès à distance** — Choisissez parmi quatre options intégrées d’accès privé ou de tunnel de jeu.
- 🤖 **IA et automatisation** — Connectez des assistants d’IA et des applications compatibles au moyen de MCP et de REST.
- 💬 **Telegram et Discord** — Utilisez les robots, les notifications et les fonctions communautaires si vous le souhaitez.
- 🛡️ **Gestion du système** — Gérez le réseau, le stockage et les mises à jour, consultez l’état du système et les journaux, et utilisez les commandes d’alimentation prises en charge sans avoir à apprendre les commandes Linux correspondantes.

JustVoxel s’adresse aux familles, aux groupes d’amis, aux petites communautés Minecraft et aux personnes qui préfèrent avoir leur propre serveur plutôt que de louer un service d’hébergement Minecraft classique avec panneau de gestion.

## 🖥️ Un système, deux éditions

### JustVoxel VM

Cette édition convient aux machines virtuelles et aux environnements hébergés, notamment :

- Proxmox
- KVM / libvirt
- VMware
- Hyper-V
- VirtualBox
- les machines virtuelles hébergées et les serveurs privés virtuels (VPS) qui acceptent une image de système d’exploitation personnalisée compatible

L’édition VM offre le système Minecraft complet, sans les paquets destinés au matériel physique qui seraient inutiles dans une machine virtuelle.

### JustVoxel HWS

Cette édition s’installe directement sur du matériel physique, par exemple :

- un vieux portable
- un ordinateur de bureau réaffecté
- un mini-PC
- un serveur domestique
- un serveur physique dédié loué chez un hébergeur qui accepte les supports d’installation personnalisés

HWS ajoute des fonctions propres aux machines physiques, dont la prise en charge des blocs d’alimentation sans coupure (UPS) avec NUT, les outils SMART et NVMe, les capteurs, les commandes du micrologiciel et de l’UEFI, les micrologiciels de périphériques, les diagnostics matériels ainsi que les commandes d’écran et de couvercle des portables.

Les deux éditions offrent le même centre de contrôle JustVoxel, la même API de gestion et les mêmes fonctions de gestion de Minecraft, de stockage, de sauvegarde, de réseautique et de mise à jour.

[Consulter les fonctionnalités des éditions VM et HWS](../docs/VARIANTS.md)

## 🚀 Installer JustVoxel

Une seule image ISO permet d’installer l’une ou l’autre des éditions de JustVoxel.

[⬇️ Télécharger l’image ISO JustVoxel x86_64](https://sourceforge.net/projects/justvoxel/files/JustVoxel-x86_64.iso/download)

Choisissez l’édition qui correspond à votre environnement :

- **JustVoxel VM** pour les machines virtuelles, y compris celles des hébergeurs compatibles
- **JustVoxel HWS** pour les ordinateurs de bureau, les portables, les mini-PC, les serveurs domestiques physiques et les serveurs physiques dédiés des hébergeurs compatibles

Le [guide d’installation](../docs/INSTALLER.md) explique comment préparer le disque, installer JustVoxel et ouvrir une première session.

## 🏠 Minecraft d’abord, avec de la place pour d’autres services

JustVoxel est conçu et pris en charge avant tout comme système dédié aux serveurs Minecraft. Il repose sur un **serveur AlmaLinux 10 bootc standard avec Podman, Quadlet, systemd, SSH, NetworkManager et les outils d’administration Linux habituels**.

Les utilisateurs avancés peuvent aussi se servir de la machine comme serveur domestique pour d’autres usages et y faire fonctionner des conteneurs en parallèle avec Minecraft.

Vous pouvez notamment déployer manuellement les services suivants :

- **Jellyfin** ou **Plex** pour vos médias
- **Vaultwarden** pour la gestion des mots de passe
- **Immich** pour la gestion des photos et des vidéos
- **Home Assistant Container** pour la domotique
- **Nextcloud** pour les fichiers et la collaboration
- **Syncthing** ou **Navidrome**
- **OpenClaw** comme agent ou passerelle d’IA auto-hébergé

**Le centre de contrôle JustVoxel ne gère pas ces applications.** L’administrateur garde la main sur leurs conteneurs Podman, leurs Quadlets, leur stockage, leurs ports, leurs mises à jour, leurs sauvegardes, la configuration de leur mandataire inverse et les exigences propres à chacune.

JustVoxel garde cette répartition simple. Il gère le système Minecraft, tandis que les utilisateurs avancés peuvent employer Podman et Quadlet comme d’habitude pour leurs autres services.

Les opérations de maintenance de JustVoxel traitent aussi les conteneurs sans lien avec Minecraft comme des services qui relèvent de l’administrateur. Les opérations Minecraft comme **Recommencer** ne suppriment pas ces conteneurs Podman.

Votre machine peut donc rester dédiée en priorité à Minecraft tout en jouant le rôle d’un serveur domestique à conteneurs si vous le souhaitez.

## 🎮 Créez le serveur Minecraft qui vous convient

JustVoxel peut faire fonctionner plusieurs serveurs Minecraft indépendants en même temps si la machine dispose de suffisamment de mémoire et de ports libres.

Chaque serveur a sa propre identité, sa configuration, son environnement d’exécution, ses ports, ses données, ses sauvegardes, ses commandes de démarrage et d’arrêt ainsi que son état.

### Logiciels serveurs actuellement pris en charge

| Logiciel serveur | Contenu pris en charge | Jeu multiplateforme Java → Bedrock géré par JustVoxel |
| --- | --- | --- |
| **Paper** | Extensions | ✅ |
| **Purpur** | Extensions | ✅ |
| **Spigot** | Extensions | ✅ |
| **Pufferfish** | Extensions | ✅ |
| **Leaf** | Extensions | ✅ |
| **Fabric** | Mods | ✅ Si la compatibilité exacte est confirmée |
| **NeoForge** | Mods | ✅ Si la compatibilité exacte est confirmée |
| **Forge** | Mods | Java seulement |
| **Quilt** | Mods | Java seulement |
| **Vanilla Java** | Aucune extension ni aucun mod | Java seulement |
| **Bedrock natif** | Serveur Bedrock natif | Clients Bedrock natifs |

Choisissez la **configuration simple** pour être guidé lors de la création de votre premier serveur. La **configuration avancée** vous donne plus de contrôle sur la mémoire, les ports, le stockage, les sauvegardes, la gestion des versions et les options de jeu multiplateforme compatibles.

JustVoxel vérifie quelles fonctionnalités sont compatibles avec le logiciel serveur que vous choisissez.

[Consulter les détails sur les logiciels serveurs et leur compatibilité](../docs/server-software.md)

## 🧩 Extensions et mods

Ouvrez **Minecraft → Extensions et mods** pour le serveur sélectionné.

Selon le type de serveur, JustVoxel peut utiliser les catalogues suivants :

- [Modrinth](https://modrinth.com/)
- [Hangar](https://hangar.papermc.io/)
- [CurseForge](https://www.curseforge.com/minecraft) avec la clé d’API que vous avez configurée

Pour les installations prises en charge, JustVoxel vérifie la compatibilité, les téléchargements et le redémarrage. Il effectue des sauvegardes à froid, garde une trace des éléments qu’il gère et prévoit des mécanismes de récupération.

Les serveurs de la famille Paper peuvent utiliser les catalogues d’extensions. Fabric, Forge, NeoForge et Quilt utilisent les catalogues de mods et vérifient que le chargeur de mods et la version de Minecraft correspondent exactement. Par choix de conception, Vanilla Java et Bedrock natif ne proposent aucun catalogue d’extensions ou de mods.

[En savoir plus sur les extensions et les mods](../docs/plugin-workspace.md)

## 🌐 Réseau Minecraft facultatif

Vous pouvez très bien exploiter plusieurs serveurs autonomes. Ils **n’ont pas besoin** d’un serveur mandataire.

Si vous voulez offrir un point d’entrée public unique pour plusieurs serveurs compatibles, JustVoxel propose aussi un **réseau Minecraft** facultatif qui s’appuie sur Velocity.

Dans le centre de contrôle, vous pouvez sélectionner les serveurs d’arrière-plan pris en charge, configurer les ports publics Java et Bedrock, gérer le salon d’accueil virtuel et son message de bienvenue, et choisir les serveurs qui font partie du réseau.

Paper, Purpur, Pufferfish et Leaf sont pris en charge comme serveurs d’arrière-plan. Les serveurs Fabric/Quilt et Forge/NeoForge compatibles peuvent aussi se joindre au réseau si JustVoxel confirme la compatibilité requise avec le mandataire. Spigot, Vanilla et Bedrock natif restent autonomes.

[En savoir plus sur le réseau Minecraft](../docs/minecraft-network.md)

## 🖥️ Découvrez le centre de contrôle

JustVoxel organise ses outils autour des tâches que vous accomplissez sur vos serveurs, sans vous obliger à connaître le fonctionnement interne de Linux.

### Serveurs Minecraft

Créez et gérez plusieurs serveurs, démarrez-les ou arrêtez-les, consultez leur état et la liste des joueurs, modifiez les paramètres pris en charge et accédez aux outils propres à chacun.

### Extensions et mods

Trouvez du contenu compatible, consultez les extensions et les mods installés, gérez les ajouts pris en charge et consultez les renseignements sur la récupération lorsqu’une modification nécessite votre intervention.

### Sauvegardes et stockage

Créez des sauvegardes, consultez leur historique, restaurez toutes les données d’un serveur, configurez les destinations de sauvegarde, importez ou exportez des serveurs et gérez le stockage pris en charge.

### Réseau

Gérez les connexions Ethernet et Wi-Fi, le **dépannage des connexions** et l’accès à distance. Pour les modifications réseau qui risquent de vous déconnecter, un mécanisme de retour à la configuration précédente évite qu’une erreur de configuration IP ou Wi-Fi vous fasse perdre l’accès de façon permanente.

### Système

Consultez l’état du système, les ressources et les journaux. Gérez les mises à jour du système d’exploitation, les utilisateurs, la sécurité, les commandes d’alimentation prises en charge et les commandes matérielles propres à l’édition HWS.

### Connexions IA et API

Connectez les applications d’IA prises en charge, les clients MCP, les outils d’automatisation REST, Telegram et Discord sans avoir à déployer un autre système de gestion.

Vous préférez le terminal? JustVoxel comprend aussi **mjust**, une interface d’administration interactive en mode texte qui applique les mêmes règles de sécurité.

[Documentation de l’interface Web](../docs/WEBUI.md) · [Guide de mjust](../docs/MJUST.md)

## 🌐 Choisissez comment vos amis se connectent

JustVoxel intègre actuellement quatre options d’accès à distance sous **Réseau → Accès à distance**.

Pour chaque fournisseur, JustVoxel va au-delà de l’inclusion des paquets dans l’image. Selon le service, vous pouvez consulter son état, l’activer, le désactiver et le configurer.

| Option | Usage recommandé | Ce que vous obtenez |
| --- | --- | --- |
| **[Tailscale](https://tailscale.com/)** | Serveurs privés pour la famille et les amis | Réseau maillé privé entre les appareils autorisés |
| **[NetBird](https://netbird.io/)** | Réseaux privés avec gestion des accès | Réseau privé avec gestion des comptes et des politiques d’accès |
| **[Playit.gg](https://playit.gg/)** | Accès public à Minecraft sans redirection de ports sur le routeur | Tunnels de jeu publics et adresses de connexion |
| **[NordVPN Meshnet](https://meshnet.nordvpn.com/)** | Accès privé entre les appareils des utilisateurs de NordVPN | Connexion à Meshnet sans utiliser JustVoxel comme client VPN commercial |

### Quelle option choisir?

**🎮 Vos amis veulent se joindre à la partie sans installer de client VPN privé?**  
Commencez par **Playit.gg**. Les joueurs peuvent utiliser l’adresse publique du tunnel de jeu que vous avez configuré.

**🔒 Vous voulez un serveur privé réservé à des appareils de confiance?**  
Envisagez **Tailscale** ou **NetBird**.

**🌍 Vous utilisez déjà NordVPN Meshnet?**  
JustVoxel peut gérer son intégration à Meshnet. Cette fonction est distincte de la connexion à un serveur de sortie du service VPN commercial de NordVPN.

Les comptes, les politiques du service infonuagique et la configuration des tunnels continuent de relever du fournisseur. JustVoxel gère les fonctions d’intégration prises en charge sur votre machine.

Vous pouvez utiliser plusieurs VPN ou réseaux maillés à la fois, mais leurs routes, leurs paramètres DNS ou leurs règles de pare-feu peuvent entrer en conflit.

[Consulter le guide du réseau et de l’accès à distance](../docs/NETWORKING.md)

## 🤖 IA, MCP, REST et automatisation

Vous voulez demander l’état du serveur à un assistant d’IA, connecter votre système domotique ou créer vos propres commandes automatisées?

JustVoxel propose des **interfaces MCP et REST intégrées** dans **Connexions IA et API**.

Lors de la configuration, vous pouvez choisir parmi les applications compatibles et les types de connexion suivants :

- **OpenAI — ChatGPT et Codex**
- **Claude**
- **Grok**
- **Gemini**
- **GitHub Copilot**
- **Autre application d’IA ou client MCP**
- **Automatisation / API REST**

Vous décidez à quoi chaque connexion a accès.

Les autorisations vont du **niveau 1 — Lecture seule** à la gestion des serveurs, puis aux commandes système avancées, qui exigent votre approbation explicite. Vous pouvez aussi limiter une connexion à certains serveurs Minecraft.

JustVoxel n’a besoin d’aucune clé d’API de modèle d’IA pour rendre ses interfaces MCP ou REST accessibles. La prise en charge du côté client dépend de l’application, des fonctions associées au compte ou à l’abonnement et de l’accès réseau.

La passerelle IA/API est désactivée par défaut. Vous choisissez quand l’activer.

[En savoir plus sur les connexions IA et API](../docs/AI-API-CONNECTIONS.md)

## 💬 Telegram et Discord

JustVoxel offre des intégrations natives facultatives pour Telegram et Discord. Elles passent par le même service de gestion que l’interface Web, qui fait autorité pour les opérations sur le système.

### Telegram

Un robot Telegram dédié vous permet de :

- consulter l’état du système et de Minecraft
- sélectionner un serveur et consulter les renseignements sur les joueurs
- démarrer, arrêter et redémarrer un serveur si vous avez les droits de gestion requis
- effectuer des sauvegardes manuelles
- consulter les mises à jour de Minecraft et des conteneurs
- obtenir des renseignements sur les mises à jour du système d’exploitation
- redémarrer le système de façon contrôlée
- recevoir, si vous le souhaitez, des notifications sur l’état des serveurs, le résultat des sauvegardes et les mises à jour disponibles

Telegram utilise ses propres autorisations de **lecture seule** et de **gestion des serveurs** et ne nécessite aucun modèle d’IA.

[Configuration et autorisations de Telegram](../docs/TELEGRAM.md)

### Discord

Discord peut servir à la fois de point de rencontre pour votre communauté Minecraft et d’interface d’administration.

Les membres du serveur Discord que vous avez explicitement associé à JustVoxel peuvent consulter les renseignements de base sur l’état du serveur et demander l’accès à Minecraft, sans obtenir automatiquement de droits d’administration.

Les administrateurs Discord autorisés dans JustVoxel peuvent démarrer, arrêter et redémarrer les serveurs, lancer une sauvegarde manuelle, consulter les mises à jour, utiliser les commandes du réseau Minecraft et effectuer d’autres opérations avancées prises en charge.

L’adhésion au serveur Discord, l’inscription à la liste des joueurs autorisés dans Minecraft et l’administration de JustVoxel relèvent de permissions distinctes.

[Configuration de Discord, accès de la communauté et administration](../docs/DISCORD.md)

## 💾 Sauvegarde, restauration, stockage et migration

Les données Minecraft comptent davantage que l’image du système.

JustVoxel conserve les données Minecraft et les sauvegardes à l’extérieur du déploiement immuable du système d’exploitation. Vous choisissez séparément où stocker les unes et les autres.

Les options de stockage prises en charge comprennent :

- le système de fichiers du système d’exploitation
- les disques internes dédiés
- le stockage USB
- les partitions et systèmes de fichiers locaux existants
- les nouvelles partitions créées dans de l’espace déjà non alloué
- NFS
- SMB/CIFS

Chaque serveur peut avoir ses sauvegardes automatiques et manuelles. La restauration remplace toutes les données persistantes du serveur sélectionné tout en conservant son identité JustVoxel et sa configuration dans le système.

Pour transférer des serveurs entre deux machines ou les récupérer après une réinstallation, JustVoxel propose aussi des procédures d’**exportation, d’importation et de récupération** qui comprennent une étape de vérification.

[Stockage](../docs/STORAGE.md) · [Restauration](../docs/RESTORE.md) · [Migration](../docs/MIGRATION.md)

## 🛡️ Des mesures de sécurité pour vos modifications

Derrière l’interface Web, JustVoxel encadre les commandes qui peuvent être exécutées sur la machine.

Les opérations prises en charge appliquent notamment les mesures de sécurité suivantes :

- vérification de la présence de joueurs avant une opération susceptible d’interrompre le jeu
- révision des modifications avant de les appliquer
- sauvegardes à froid lorsqu’une modification exige une protection pour permettre la récupération
- identification précise de chaque serveur pour éviter toute confusion avec un autre
- vérification du redémarrage et du bon fonctionnement du serveur
- conservation de l’état des opérations et de la récupération
- possibilité de revenir en arrière pour les opérations prises en charge
- points de contrôle NetworkManager pour les modifications réseau à risque
- procédures distinctes pour la mise à jour de Minecraft et celle du système d’exploitation

L’interface Web accessible dans le navigateur fonctionne sans privilèges élevés. Les modifications qui exigent de tels privilèges passent par l’agent de gestion local de JustVoxel et les opérations explicites de son API. Le navigateur n’a aucun accès à un interpréteur de commandes root.

## 🔄 Des mises à jour système distinctes de celles de Minecraft

JustVoxel est un système géré par images qui repose sur Home Server Base 10 / AlmaLinux 10.

Les mises à jour du système d’exploitation sont gérées séparément des versions de Minecraft et des mises à jour des logiciels serveurs, des extensions, des mods et des conteneurs.

Une mise à jour de JustVoxel ne change donc pas votre version de Minecraft à votre insu. De même, modifier un serveur Minecraft ne nécessite pas de reconstruire le système d’exploitation hôte.

[Documentation sur la gestion du système](../docs/SYSTEM.md)

## 📊 Comment JustVoxel se compare-t-il aux autres solutions?

JustVoxel se distingue d’un panneau de gestion Minecraft classique : **le système d’exploitation et les outils de gestion de Minecraft forment un seul produit**.

| Fonctionnalité | **JustVoxel** | **MineOS SvelteKit** | **Crafty Controller** | **MineOS Node** |
| --- | :---: | :---: | :---: | :---: |
| Système d’exploitation complet dédié à Minecraft | ⭐ | — | — | — |
| Plusieurs serveurs Minecraft | ✅ | ✅ | ✅ | ✅ |
| Prise en charge des serveurs Java et Bedrock | ✅ | ✅ | ✅ | 🟡 Java et ancienne prise en charge de PocketMine |
| Recherche guidée d’extensions et de mods | ✅ Modrinth, Hangar, CurseForge | ✅ Modrinth, CurseForge | — | — |
| Gestion du réseau, du stockage et du système d’exploitation hôtes | ⭐ | — | — | — |
| Accès privé et tunnels de jeu intégrés | ⭐ 4 options | — | — | — |
| Passerelle MCP et REST pour l’IA et l’automatisation | ⭐ | — | — | — |
| Gestion intégrée par Telegram et Discord | ⭐ | — | — | — |
| Commandes de couvercle et d’écran des portables | ✅ HWS | — | — | — |
| Mises à jour par image du système d’exploitation immuable | ✅ | — | — | — |
| Interface de gestion en mode terminal | ✅ mjust | ✅ mineos CLI/TUI | — | ✅ CLI |
| Mode d’installation | Système bootc complet | Application Docker / Compose | Panneau de gestion sous Docker, Linux ou Windows | Application Node.js sous Linux/BSD |

⭐ Point distinctif &nbsp;•&nbsp; ✅ Pris en charge &nbsp;•&nbsp; 🟡 Prise en charge partielle ou ancienne &nbsp;•&nbsp; — Fonction absente de la documentation de base consultée pour le projet

**Dernière révision du comparatif : 16 novembre 2026.** Le but de ce tableau n’est pas de dénigrer les autres projets. MineOS et Crafty sont des applications de gestion Minecraft qui peuvent être d’excellents choix si vous avez déjà un système d’exploitation que vous souhaitez administrer vous-même. JustVoxel s’adresse aux personnes qui veulent gérer le panneau Minecraft **et le système serveur qui le fait fonctionner** comme un seul produit.

Le comparatif s’appuie sur la documentation et les dépôts actuels des projets :

- [MineOS SvelteKit](https://github.com/freeman412/mineos-sveltekit)
- [Crafty Controller](https://craftycontrol.com/) · [code source](https://gitlab.com/crafty-controller/crafty-4)
- [MineOS Node](https://github.com/hexparrot/mineos-node)

Si une entrée du tableau est dépassée ou vous semble injuste, veuillez ouvrir un ticket. Les corrections sont les bienvenues, même si elles sont moins favorables à JustVoxel.

## 📚 Documentation

Les guides détaillés de JustVoxel se trouvent dans le [répertoire de documentation](../docs/).

Voici quelques points de départ :

- [Interface Web](../docs/WEBUI.md)
- [Environnement d’exécution Minecraft](../docs/MINECRAFT_RUNTIME.md)
- [Logiciels serveurs](../docs/server-software.md)
- [Extensions et mods](../docs/plugin-workspace.md)
- [Réseau Minecraft](../docs/minecraft-network.md)
- [Réseautique et accès à distance](../docs/NETWORKING.md)
- [Connexions IA et API](../docs/AI-API-CONNECTIONS.md)
- [Telegram](../docs/TELEGRAM.md)
- [Discord](../docs/DISCORD.md)
- [Stockage](../docs/STORAGE.md)
- [Restauration](../docs/RESTORE.md)
- [Migration](../docs/MIGRATION.md)
- [Gestion du système](../docs/SYSTEM.md)
- [mjust](../docs/MJUST.md)

Pour le développement des images, les essais et la publication des versions, consultez le [guide de développement](../README-DEV.md).

## ⛏️ Logiciels Minecraft et contrat de licence (EULA)

JustVoxel **ne fournit aucun** binaire de serveur Minecraft, logiciel serveur de Mojang, monde créé à l’avance ou contrat de licence Minecraft (EULA) déjà accepté.

Lors de la création du serveur, l’administrateur accepte le contrat de licence Minecraft, puis JustVoxel prépare l’environnement d’exécution pris en charge qui correspond au serveur choisi.

Minecraft, les logiciels de Mojang, Paper, Purpur, Spigot, Pufferfish, Leaf, Fabric, NeoForge, Forge, Quilt, les extensions, les mods et les autres composants tiers conservent leurs propres licences et conditions de distribution.

## 📄 Licence

JustVoxel est distribué sous licence **Apache-2.0**. Consultez le fichier [LICENSE](../LICENSE).

Les avis concernant les composants tiers figurent dans le fichier [THIRD_PARTY_NOTICES.md](../THIRD_PARTY_NOTICES.md).
