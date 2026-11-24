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

> **À propos de la traduction :** Cette traduction de JustVoxel a été réalisée à partir du texte anglais de référence avec l’aide de l’IA. Si vous relevez une formulation inexacte, peu claire ou peu naturelle, merci de proposer une correction en ouvrant un [ticket sur GitHub](https://github.com/home-server-project/justvoxel/issues) ou en soumettant une [demande de fusion](https://github.com/home-server-project/justvoxel/pulls).

<p align="center">
  <strong>Vos serveurs Minecraft sur votre propre matériel, un serveur dédié en location ou une VM hébergée. Un seul Centre de contrôle.</strong>
</p>

<p align="center">
  Une solution tout-en-un pour auto-héberger vos serveurs Minecraft et les gérer sans avoir à administrer le système d’exploitation.
</p>

<p align="center">
  <a href="#-installer-justvoxel">Installation</a> •
  <a href="#-hébergez-minecraft-plus-simplement">Fonctionnalités</a> •
  <a href="../docs/INSTALLER.md">Documentation</a>
</p>

<p align="center">
  <img src="../docs/images/justvoxel-overview-1.webp" width="100%" alt="Aperçu du Centre de contrôle JustVoxel"><img src="../docs/images/justvoxel-overview-2.webp" width="100%" alt=""><img src="../docs/images/justvoxel-overview-3.webp" width="100%" alt="">
</p>

JustVoxel est un système dédié à l’auto-hébergement de serveurs Minecraft. Installez-le sur un PC, un ordinateur portable, un serveur physique, une machine virtuelle ou un service d’hébergement compatible qui vous permet d’utiliser votre propre image système. Vous pourrez ensuite gérer Minecraft depuis un navigateur.

Aucun ensemble de scripts à maintenir, aucun panneau de gestion Minecraft séparé à installer.

Vous avez un ancien ordinateur portable ou PC ? **Avec JustVoxel HWS, vous pouvez le réutiliser comme serveur dédié à Minecraft.** Sur les ordinateurs portables compatibles, JustVoxel peut empêcher la mise en veille à la fermeture du capot pour que le serveur continue de fonctionner. Vous pouvez également éteindre l’écran de la console locale immédiatement ou après un court délai.

> 🚧 **Préversion** — JustVoxel se prépare à sa première publication. L’ISO d’installation actuelle est disponible pour les essais et les premières utilisations pendant la validation finale de la version publique.

## ✨ Hébergez Minecraft plus simplement

JustVoxel réunit la gestion de Minecraft et celle du système d’exploitation.

- 🎮 **Plusieurs serveurs Minecraft** — faites fonctionner indépendamment différents mondes et types de serveurs sur la même machine.
- 🖥️ **Centre de contrôle web** — gérez vos serveurs depuis un ordinateur, une tablette ou un téléphone.
- 🧩 **Plugins et mods** — découvrez et gérez du contenu compatible auprès des fournisseurs pris en charge.
- 💾 **Sauvegarde et restauration** — protégez chaque serveur avec des sauvegardes planifiées et manuelles.
- 🌐 **Accès à distance** — choisissez parmi quatre solutions intégrées d’accès privé ou de tunnel de jeu.
- 🤖 **IA et automatisation** — connectez des assistants IA et des applications compatibles via MCP et REST.
- 💬 **Telegram et Discord** — utilisez, si vous le souhaitez, des commandes de bot, des notifications et des outils pour votre communauté.
- 🛡️ **Gestion du système** — gérez le réseau, le stockage, les mises à jour et les commandes d’alimentation prises en charge, et consultez l’état du système et les journaux sans apprendre les commandes Linux correspondantes.

JustVoxel s’adresse aux familles, aux groupes d’amis, aux petites communautés Minecraft et aux personnes qui souhaitent avoir leur propre serveur sans louer de panneau de gestion Minecraft classique.

## 🖥️ Un système, deux éditions

### JustVoxel VM

L’édition VM est destinée aux machines virtuelles et aux environnements hébergés.

- Proxmox
- KVM / libvirt
- VMware
- Hyper-V
- VirtualBox
- VM hébergées ou VPS qui acceptent une image de système d’exploitation personnalisée et compatible

L’édition VM fournit l’environnement Minecraft complet sans les paquets destinés au matériel physique qui seraient inutiles dans une machine virtuelle.

### JustVoxel HWS

L’édition HWS s’installe directement sur du matériel physique.

- un ancien ordinateur portable
- un ordinateur de bureau reconverti
- un mini-PC
- un serveur domestique
- un serveur physique dédié en location qui accepte un support d’installation personnalisé

HWS ajoute des fonctions destinées aux machines physiques, dont l’intégration des onduleurs via NUT, les outils SMART et NVMe, la prise en charge des capteurs et des micrologiciels de périphériques, les commandes liées au micrologiciel et à l’UEFI, les diagnostics matériels et les réglages de l’écran et du capot des ordinateurs portables.

Les deux éditions partagent le Centre de contrôle JustVoxel, l’API de gestion, les opérations Minecraft, le modèle de stockage, les sauvegardes, les fonctions réseau et le mode de mise à jour.

[Découvrir les fonctionnalités des éditions VM et HWS](../docs/VARIANTS.md)

## 🚀 Installer JustVoxel

Une seule image ISO permet d’installer les deux éditions de JustVoxel.

[⬇️ Télécharger l’ISO JustVoxel x86_64](https://sourceforge.net/projects/justvoxel/files/JustVoxel-x86_64.iso/download)

Choisissez l’édition qui correspond à votre environnement.

- **JustVoxel VM** pour les machines virtuelles et les environnements de VM hébergées compatibles
- **JustVoxel HWS** pour les PC, les ordinateurs portables, les mini-PC, les serveurs domestiques physiques et les serveurs dédiés hébergés compatibles

Le [guide d’installation](../docs/INSTALLER.md) détaille la préparation des disques, l’installation et la première connexion.

## 🏠 Un serveur dédié à Minecraft qui peut aussi héberger d’autres services

JustVoxel est conçu et pris en charge avant tout comme un système dédié à Minecraft. Il repose sur un **serveur AlmaLinux 10 bootc standard avec Podman, Quadlet, systemd, SSH, NetworkManager et les outils d’administration Linux habituels**.

Les utilisateurs expérimentés peuvent aussi utiliser la machine comme serveur domestique polyvalent et exécuter d’autres conteneurs en parallèle de Minecraft.

Vous pouvez par exemple déployer manuellement les services suivants.

- **Jellyfin** ou **Plex** pour les contenus multimédias
- **Vaultwarden** pour la gestion des mots de passe
- **Immich** pour la gestion des photos et des vidéos
- **Home Assistant Container** pour la domotique
- **Nextcloud** pour les fichiers et le travail collaboratif
- **Syncthing** ou **Navidrome**
- **OpenClaw** comme agent ou passerelle IA auto-hébergée

**Ces applications relèvent de l’administration manuelle.** Leurs conteneurs Podman, leurs fichiers Quadlet, leur stockage, leurs ports, leurs mises à jour, leurs sauvegardes, la configuration de leur proxy inverse et leurs besoins propres restent sous le contrôle de l’administrateur.

Le Centre de contrôle gère l’environnement Minecraft. Pour les autres services, les utilisateurs expérimentés peuvent continuer à utiliser Podman et Quadlet comme d’habitude.

Lors des opérations de maintenance, JustVoxel considère les autres conteneurs comme des services sous la responsabilité de l’administrateur. Ils sont préservés lors d’opérations telles que **Recommencer** dans Minecraft.

La même machine peut ainsi rester principalement dédiée à Minecraft tout en faisant office de serveur domestique pour d’autres services conteneurisés si vous en avez besoin.

## 🎮 Créez le serveur Minecraft qui vous convient

JustVoxel peut faire fonctionner plusieurs serveurs Minecraft indépendants en même temps, si la machine dispose de suffisamment de mémoire et de ports libres.

Chaque serveur a sa propre identité, sa configuration, son environnement d’exécution, ses ports, ses données, ses sauvegardes, ses commandes de gestion du cycle de vie et son état.

### Logiciels serveur pris en charge

| Logiciel serveur | Type de contenu | Jeu multiplateforme Java → Bedrock géré par JustVoxel |
| --- | --- | --- |
| **Paper** | Plugins | ✅ |
| **Purpur** | Plugins | ✅ |
| **Spigot** | Plugins | ✅ |
| **Pufferfish** | Plugins | ✅ |
| **Leaf** | Plugins | ✅ |
| **Fabric** | Mods | ✅ Si la compatibilité exacte est confirmée |
| **NeoForge** | Mods | ✅ Si la compatibilité exacte est confirmée |
| **Forge** | Mods | Java uniquement |
| **Quilt** | Mods | Java uniquement |
| **Vanilla Java** | Sans plugins ni mods | Java uniquement |
| **Bedrock natif** | Serveur Bedrock natif | Clients Bedrock natifs |

Choisissez **Configuration simple** pour être guidé dans la création de votre premier serveur, ou **Configuration avancée** pour régler plus finement la mémoire, les ports, le stockage, les sauvegardes, la gestion des versions et le jeu multiplateforme lorsque celui-ci est compatible.

JustVoxel vérifie quelles fonctionnalités sont compatibles avec le logiciel serveur sélectionné.

[Consulter les logiciels serveur et le détail des compatibilités](../docs/server-software.md)

## 🧩 Plugins et mods

Pour le serveur sélectionné, ouvrez **Minecraft → Plugins et mods**.

Selon le type de serveur, JustVoxel peut utiliser les services suivants.

- [Modrinth](https://modrinth.com/)
- [Hangar](https://hangar.papermc.io/)
- [CurseForge](https://www.curseforge.com/minecraft) avec la clé API que vous avez configurée

Les installations prises en charge s’appuient sur des contrôles de compatibilité, des téléchargements vérifiés, des sauvegardes à froid, une vérification du redémarrage, un suivi des éléments gérés par JustVoxel et des mécanismes de récupération.

Les serveurs de la famille Paper peuvent utiliser des catalogues de plugins. Fabric, Forge, NeoForge et Quilt utilisent des catalogues de mods avec une vérification précise du chargeur de mods et de la version de Minecraft. Par choix, aucun catalogue de plugins ou de mods n’est proposé pour Vanilla Java ni pour Bedrock natif.

[En savoir plus sur les plugins et les mods](../docs/plugin-workspace.md)

## 🌐 Réseau Minecraft facultatif

Vous pouvez utiliser plusieurs serveurs autonomes **sans proxy**.

Si vous souhaitez proposer un point d’entrée public unique pour plusieurs serveurs compatibles, JustVoxel intègre également un **réseau Minecraft** facultatif basé sur Velocity.

Depuis le Centre de contrôle, vous pouvez sélectionner les serveurs pris en charge à placer derrière le proxy, configurer les ports publics Java et Bedrock, gérer le lobby virtuel et le message de bienvenue, et choisir les serveurs qui font partie du réseau.

Paper, Purpur, Pufferfish et Leaf sont pris en charge dans le réseau. Les serveurs Fabric/Quilt et Forge/NeoForge compatibles peuvent aussi en faire partie si JustVoxel confirme leur compatibilité avec le proxy. Spigot, Vanilla et Bedrock natif restent autonomes.

[Découvrir le réseau Minecraft](../docs/minecraft-network.md)

## 🖥️ Découvrez le Centre de contrôle

JustVoxel organise son interface autour des tâches de gestion du serveur pour que vous puissiez les effectuer sans connaître le fonctionnement interne de Linux.

### Serveurs Minecraft

Créez et gérez plusieurs serveurs, démarrez-les ou arrêtez-les, consultez leur état et la liste des joueurs, configurez les paramètres disponibles et accédez aux outils propres à chaque serveur.

### Plugins et mods

Découvrez du contenu compatible, consultez les plugins et mods installés, gérez les ajouts pris en charge et accédez aux informations de récupération lorsqu’une modification nécessite votre intervention.

### Sauvegardes et stockage

Créez des sauvegardes, consultez leur historique, restaurez l’intégralité des données d’un serveur, configurez les destinations de sauvegarde, importez ou exportez des serveurs et gérez les espaces de stockage pris en charge.

### Réseau

Gérez Ethernet, le Wi-Fi, le **dépannage de la connexion** et l’accès à distance. Les modifications réseau susceptibles d’interrompre l’accès bénéficient d’un mécanisme de retour arrière qui vous permet de retrouver l’accès en cas de mauvais réglage IP ou Wi-Fi.

### Système

Consultez l’état du système, les ressources, les journaux et les mises à jour du système d’exploitation. Gérez les utilisateurs, les paramètres de sécurité, les commandes d’alimentation prises en charge et les réglages matériels propres à HWS.

### Connexions IA et API

Connectez des applications IA compatibles, des clients MCP, des outils d’automatisation REST, Telegram et Discord sans mettre en place un autre système de gestion.

Vous préférez utiliser un terminal ? JustVoxel inclut aussi **mjust**, une interface d’administration interactive en mode terminal qui applique les mêmes règles de sécurité que le reste du système.

[Documentation de l’interface web](../docs/WEBUI.md) · [Guide de mjust](../docs/MJUST.md)

## 🌐 Choisissez comment vos amis se connectent

JustVoxel intègre actuellement quatre options d’accès à distance dans **Réseau → Accès à distance**.

Pour chaque fournisseur, JustVoxel permet, selon le cas, de consulter l’état de l’intégration, de l’activer, de la désactiver et de la configurer.

| Solution | Pour quel usage ? | Ce qu’elle apporte |
| --- | --- | --- |
| **[Tailscale](https://tailscale.com/)** | Serveurs privés pour vos amis et votre famille | Réseau maillé privé entre les appareils autorisés |
| **[NetBird](https://netbird.io/)** | Réseaux privés avec gestion des accès | Réseau privé avec gestion des comptes et des règles d’accès |
| **[Playit.gg](https://playit.gg/)** | Accès public à Minecraft sans redirection de ports sur le routeur | Tunnels de jeu publics et adresses de connexion |
| **[NordVPN Meshnet](https://meshnet.nordvpn.com/)** | Accès privé entre appareils pour les utilisateurs de NordVPN | Connexion via Meshnet sans utiliser JustVoxel comme client du VPN commercial |

### Quelle solution choisir ?

**🎮 Vos amis veulent se connecter sans installer de client VPN privé ?**  
Commencez par regarder **Playit.gg**. Les joueurs peuvent utiliser l’adresse publique du tunnel de jeu que vous avez configuré.

**🔒 Vous souhaitez réserver l’accès à des appareils de confiance ?**  
Envisagez **Tailscale** ou **NetBird**.

**🌍 Vous utilisez déjà NordVPN Meshnet ?**  
JustVoxel peut gérer son intégration Meshnet. La connexion de la machine à un serveur de sortie du service VPN commercial de NordVPN reste une opération distincte.

Les comptes, les règles d’accès côté cloud et la configuration des tunnels se gèrent chez le fournisseur. JustVoxel prend en charge l’intégration sur la machine.

Il est possible d’utiliser plusieurs solutions VPN ou réseaux maillés en même temps, mais leurs règles de routage, de DNS ou de pare-feu peuvent entrer en conflit.

[Lire le guide du réseau et de l’accès à distance](../docs/NETWORKING.md)

## 🤖 IA, MCP, REST et automatisation

Vous souhaitez demander l’état d’un serveur à un assistant IA, le relier à votre système domotique ou créer vos propres automatisations ?

JustVoxel propose des **interfaces MCP et REST** intégrées, accessibles depuis **Connexions IA et API**.

La configuration propose notamment les applications et types de connexion suivants, sous réserve de compatibilité.

- **OpenAI · ChatGPT et Codex**
- **Claude**
- **Grok**
- **Gemini**
- **GitHub Copilot**
- **Autre application IA / MCP**
- **Automatisation / API REST**

Vous décidez à quoi chaque connexion peut accéder.

Les autorisations vont du **Niveau 1 — Lecture seule** à la gestion des serveurs, puis aux commandes système avancées, qui exigent une approbation explicite. Vous pouvez également limiter une connexion à certains serveurs Minecraft.

L’interface MCP ou REST de JustVoxel est disponible sans clé API de modèle d’IA. La compatibilité côté client dépend de l’application, des fonctionnalités du compte ou de l’abonnement et de l’accès réseau.

La passerelle IA et API est désactivée par défaut. Vous pouvez l’activer lorsque vous en avez besoin.

[Découvrir les connexions IA et API](../docs/AI-API-CONNECTIONS.md)

## 💬 Telegram et Discord

JustVoxel propose des intégrations natives et facultatives pour Telegram et Discord. Elles s’appuient sur le même service de gestion que l’interface web.

### Telegram

Un bot Telegram dédié peut fournir les fonctions suivantes.

- consultation de l’état du système et de Minecraft
- sélection du serveur et consultation des informations sur les joueurs
- démarrage, arrêt et redémarrage pour les utilisateurs habilités à gérer le serveur
- sauvegardes manuelles
- consultation des mises à jour de Minecraft et des conteneurs
- informations sur les mises à jour du système d’exploitation
- redémarrage de la machine selon une procédure encadrée
- notifications facultatives sur l’état des serveurs, le résultat des sauvegardes et les mises à jour disponibles

Telegram utilise ses propres autorisations **Lecture seule** et **Gestion du serveur**. Aucun modèle d’IA n’est nécessaire.

[Configuration et autorisations Telegram](../docs/TELEGRAM.md)

### Discord

Discord peut servir à la fois d’espace communautaire pour Minecraft et d’interface d’administration.

Les membres du serveur Discord associé explicitement à JustVoxel peuvent consulter l’état de base des serveurs et demander l’accès à Minecraft sans recevoir automatiquement de droits d’administration.

Les administrateurs JustVoxel autorisés sur Discord peuvent utiliser les actions de gestion prises en charge, comme le démarrage, l’arrêt, le redémarrage, la sauvegarde manuelle, la consultation des mises à jour, les commandes du réseau Minecraft et d’autres actions avancées.

L’appartenance au serveur Discord, l’inscription sur la liste blanche Minecraft et les droits d’administration JustVoxel relèvent d’autorisations distinctes.

[Configuration de Discord, accès communautaire et administration](../docs/DISCORD.md)

## 💾 Sauvegarde, restauration, stockage et migration

Vos données Minecraft ont plus de valeur que l’image système.

JustVoxel conserve les données Minecraft et les sauvegardes en dehors du déploiement immuable du système d’exploitation. Leurs emplacements se configurent indépendamment.

Plusieurs options de stockage sont prises en charge.

- le système de fichiers du système d’exploitation
- les disques internes dédiés
- le stockage USB
- les partitions et systèmes de fichiers locaux existants
- les nouvelles partitions créées dans de l’espace déjà non alloué
- NFS
- SMB/CIFS

Des sauvegardes automatiques et manuelles sont disponibles pour chaque serveur. La restauration remplace l’ensemble des données persistantes du serveur sélectionné tout en préservant son identité dans JustVoxel et sa configuration dans le système.

Pour déplacer des serveurs d’une machine à l’autre ou les transférer lors d’une réinstallation, JustVoxel propose aussi des procédures vérifiées d’**exportation, d’importation et de récupération**.

[Stockage](../docs/STORAGE.md) · [Restauration](../docs/RESTORE.md) · [Migration](../docs/MIGRATION.md)

## 🛡️ Des modifications plus sûres

L’interface JustVoxel donne accès à des opérations de gestion définies et encadrées.

Les opérations prises en charge suivent des règles de sécurité appliquées à l’échelle du système.

- vérification de la présence de joueurs avant les actions susceptibles d’interrompre le service
- vérification des modifications avant de sélectionner **Appliquer**
- sauvegardes à froid lorsqu’une modification nécessite une possibilité de récupération
- identification exacte de chaque serveur pour éviter de le confondre avec un autre
- vérification du redémarrage et du bon fonctionnement
- conservation de l’état des opérations et de la récupération
- procédures de retour arrière pour les opérations prises en charge
- points de restauration NetworkManager pour les modifications réseau à risque
- procédures distinctes pour les mises à jour de Minecraft et du système d’exploitation

L’interface web fonctionne sans privilèges. Les modifications qui exigent des privilèges passent par l’agent de gestion local de JustVoxel et les opérations prévues par son API. Le navigateur n’a pas accès à un shell root.

## 🔄 Des mises à jour système indépendantes de celles de Minecraft

JustVoxel est un système géré par image, basé sur Home Server Base 10 / AlmaLinux 10.

Les mises à jour du système d’exploitation sont gérées séparément des versions de Minecraft, des logiciels serveur, des plugins, des mods et des mises à jour de conteneurs.

Vous pouvez ainsi mettre à jour JustVoxel sans changement implicite de version de Minecraft, et modifier un serveur Minecraft sans reconstruire le système d’exploitation de la machine hôte.

[Documentation sur la gestion du système](../docs/SYSTEM.md)

## 📊 Comment JustVoxel se compare-t-il aux autres solutions ?

À la différence d’un panneau de gestion Minecraft classique, JustVoxel réunit **le système d’exploitation et les outils de gestion de Minecraft dans un seul produit**.

| Fonctionnalité | **JustVoxel** | **MineOS SvelteKit** | **Crafty Controller** | **MineOS Node** |
| --- | :---: | :---: | :---: | :---: |
| Système d’exploitation complet dédié à Minecraft | ⭐ | — | — | — |
| Plusieurs serveurs Minecraft | ✅ | ✅ | ✅ | ✅ |
| Prise en charge des serveurs Java et Bedrock | ✅ | ✅ | ✅ | 🟡 Java et prise en charge historique de PocketMine |
| Recherche guidée de plugins et de mods | ✅ Modrinth, Hangar, CurseForge | ✅ Modrinth, CurseForge | — | — |
| Gestion du réseau, du stockage et du système d’exploitation de l’hôte | ⭐ | — | — | — |
| Accès privé et tunnels de jeu intégrés | ⭐ 4 options | — | — | — |
| Passerelle MCP et REST pour l’IA et l’automatisation | ⭐ | — | — | — |
| Gestion intégrée via Telegram et Discord | ⭐ | — | — | — |
| Réglages du capot et de l’écran des ordinateurs portables | ✅ HWS | — | — | — |
| Mises à jour par image du système d’exploitation immuable | ✅ | — | — | — |
| Interface d’administration en mode terminal | ✅ mjust | ✅ mineos CLI/TUI | — | ✅ CLI |
| Mode d’installation | Système bootc complet | Application Docker / Compose | Panneau de gestion sous Docker, Linux ou Windows | Application Node.js sous Linux/BSD |

⭐ point fort &nbsp;•&nbsp; ✅ pris en charge &nbsp;•&nbsp; 🟡 prise en charge partielle ou historique &nbsp;•&nbsp; — absent de la documentation principale consultée

**Dernière vérification du comparatif le 16 novembre 2026.** Ce comparatif vise à situer chaque projet selon vos besoins. MineOS et Crafty sont des applications de gestion de Minecraft qui peuvent très bien convenir si vous disposez déjà d’un système d’exploitation que vous souhaitez administrer vous-même. JustVoxel s’adresse aux personnes qui souhaitent gérer **le panneau Minecraft et le système sur lequel il fonctionne** comme un seul produit.

Le comparatif s’appuie sur la documentation et les dépôts des projets consultés à cette date.

- [MineOS SvelteKit](https://github.com/freeman412/mineos-sveltekit)
- [Crafty Controller](https://craftycontrol.com/) · [code source](https://gitlab.com/crafty-controller/crafty-4)
- [MineOS Node](https://github.com/hexparrot/mineos-node)

Si un élément du comparatif est obsolète ou injuste, ouvrez un ticket. Les corrections sont bienvenues même lorsqu’elles sont moins favorables à JustVoxel.

## 📚 Documentation

Les guides détaillés se trouvent dans le [dossier de documentation](../docs/).

Vous pouvez commencer par les guides suivants.

- [Interface web](../docs/WEBUI.md)
- [Environnement d’exécution Minecraft](../docs/MINECRAFT_RUNTIME.md)
- [Logiciels serveur](../docs/server-software.md)
- [Plugins et mods](../docs/plugin-workspace.md)
- [Réseau Minecraft](../docs/minecraft-network.md)
- [Réseau et accès à distance](../docs/NETWORKING.md)
- [Connexions IA et API](../docs/AI-API-CONNECTIONS.md)
- [Telegram](../docs/TELEGRAM.md)
- [Discord](../docs/DISCORD.md)
- [Stockage](../docs/STORAGE.md)
- [Restauration](../docs/RESTORE.md)
- [Migration](../docs/MIGRATION.md)
- [Gestion du système](../docs/SYSTEM.md)
- [mjust](../docs/MJUST.md)

Pour le développement des images, les tests et la publication des versions, consultez le [guide de développement](../README-DEV.md).

## ⛏️ Logiciels Minecraft et CLUF

JustVoxel **ne fournit ni** les fichiers binaires du serveur Minecraft, ni les logiciels serveur de Mojang, ni un monde déjà créé, ni un contrat de licence utilisateur final Minecraft (CLUF, ou EULA) déjà accepté.

Lors de la création d’un serveur, l’administrateur accepte le CLUF de Minecraft et JustVoxel prépare l’environnement d’exécution du serveur Minecraft compatible sélectionné.

Minecraft, les logiciels de Mojang, Paper, Purpur, Spigot, Pufferfish, Leaf, Fabric, NeoForge, Forge, Quilt, les plugins, les mods et les autres composants tiers restent soumis à leurs propres licences et conditions de distribution.

## 📄 Licence

JustVoxel est distribué sous licence **Apache-2.0**. Consultez le fichier [LICENSE](../LICENSE).

Les mentions relatives aux composants tiers figurent dans [THIRD_PARTY_NOTICES.md](../THIRD_PARTY_NOTICES.md).
