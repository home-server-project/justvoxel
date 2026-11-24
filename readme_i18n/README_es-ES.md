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

> **Nota sobre la traducción:** Esta traducción de JustVoxel se ha elaborado a partir del texto original en inglés con ayuda de la IA. Si encuentras alguna expresión inexacta, poco clara o poco natural, propón una corrección abriendo una [incidencia en GitHub](https://github.com/home-server-project/justvoxel/issues) o enviando una [solicitud de cambios (pull request)](https://github.com/home-server-project/justvoxel/pulls).

<p align="center">
  <strong>Tus servidores de Minecraft, en tu propio equipo, en un servidor dedicado de alquiler o en una máquina virtual alojada. Un único centro de control.</strong>
</p>

<p align="center">
  Un sistema para alojar tus propios servidores de Minecraft y gestionarlos desde el navegador sin tener que administrar el sistema operativo.
</p>

<p align="center">
  <a href="#-instalar-justvoxel">Instalar</a> •
  <a href="#-alojar-servidores-de-minecraft-es-más-fácil">Funciones</a> •
  <a href="../docs/INSTALLER.md">Documentación</a>
</p>

<p align="center">
  <img src="../docs/images/justvoxel-overview-1.webp" width="100%" alt="Vista general del centro de control de JustVoxel"><img src="../docs/images/justvoxel-overview-2.webp" width="100%" alt=""><img src="../docs/images/justvoxel-overview-3.webp" width="100%" alt="">
</p>

JustVoxel es un sistema dedicado para alojar tus propios servidores de Minecraft. Instálalo en un PC, un portátil, un servidor físico o una máquina virtual compatibles, o en un servicio de alojamiento que te permita ejecutar tu propia imagen del sistema operativo. Después podrás gestionar Minecraft desde el navegador.

No tendrás que mantener una colección de scripts ni instalar un panel de gestión de Minecraft aparte.

¿Tienes un portátil o un PC antiguo? **Con JustVoxel HWS puedes aprovecharlo como servidor dedicado de Minecraft en vez de dejarlo en un cajón.** En los portátiles compatibles puedes desactivar la suspensión al cerrar la tapa para que el servidor siga funcionando. También puedes apagar la pantalla de la consola local al instante o tras un breve intervalo.

> 🚧 **Versión preliminar:** JustVoxel se está preparando para su primer lanzamiento público. Puedes usar la ISO de instalación actual para probarlo y empezar a utilizarlo mientras se completa la validación final.

## ✨ Alojar servidores de Minecraft es más fácil

JustVoxel integra la gestión de Minecraft y la del sistema operativo en el que se ejecuta.

- 🎮 **Varios servidores de Minecraft** — ejecuta distintos mundos y tipos de servidor de forma independiente en un mismo equipo.
- 🖥️ **Centro de control web** — gestiona tus servidores desde un ordenador, una tableta o un móvil.
- 🧩 **Plugins y mods** — busca y gestiona contenido compatible de los proveedores admitidos.
- 💾 **Copias de seguridad y restauración** — protege cada servidor con copias programadas y manuales.
- 🌐 **Acceso remoto** — elige entre cuatro opciones integradas de acceso privado y túneles para juegos.
- 🤖 **IA y automatización** — conecta asistentes de IA y aplicaciones compatibles mediante MCP y REST.
- 💬 **Telegram y Discord** — usa bots para controlar el servidor, recibir notificaciones y gestionar la comunidad de forma opcional.
- 🛡️ **Gestión del sistema** — gestiona la red, el almacenamiento, las actualizaciones, el estado del sistema, los registros y las opciones de encendido y apagado disponibles sin tener que aprender comandos de Linux.

JustVoxel está pensado para familias, grupos de amigos, pequeñas comunidades de Minecraft y quienes prefieren tener su propio servidor en lugar de alquilar un panel de alojamiento de Minecraft tradicional.

## 🖥️ Un sistema, dos ediciones

### JustVoxel VM

Para máquinas virtuales y entornos de alojamiento como los siguientes.

- Proxmox
- KVM / libvirt
- VMware
- Hyper-V
- VirtualBox
- máquinas virtuales alojadas o servicios VPS que admitan una imagen personalizada compatible del sistema operativo

La edición VM incluye el sistema completo para alojar Minecraft y prescinde de los paquetes destinados al hardware físico que resultan innecesarios en una máquina virtual.

### JustVoxel HWS

Para instalar directamente en equipos físicos, como los siguientes.

- un portátil antiguo
- un ordenador de sobremesa reutilizado
- un mini PC
- un servidor doméstico
- un servidor físico dedicado de alquiler que permita usar medios de instalación personalizados

HWS añade funciones para equipos físicos, como la integración con sistemas de alimentación ininterrumpida (SAI) mediante NUT, herramientas SMART y NVMe, sensores, controles de firmware y UEFI, firmware de dispositivos, diagnóstico de hardware y controles de pantalla y tapa para portátiles.

Ambas ediciones comparten el centro de control de JustVoxel, la API de gestión, los procedimientos de Minecraft, el modelo de almacenamiento, las copias de seguridad, la gestión de la red y el proceso de actualización.

[Consulta las prestaciones de las ediciones VM y HWS](../docs/VARIANTS.md)

## 🚀 Instalar JustVoxel

La misma ISO de instalación sirve para las dos ediciones de JustVoxel.

[⬇️ Descargar la ISO de JustVoxel para x86_64](https://sourceforge.net/projects/justvoxel/files/JustVoxel-x86_64.iso/download)

**Crea un USB de arranque:** Graba la imagen ISO en una memoria USB con [balenaEtcher](https://etcher.balena.io/), [Rufus](https://rufus.ie/en/), [Ventoy](https://www.ventoy.net/en/index.html), [Fedora Media Writer](https://fedoraproject.org/workstation/download/) u otra herramienta que permita escribir imágenes directamente en la unidad. En Linux y otros sistemas similares a Unix también puedes usar el método tradicional con `dd`. Al grabar la imagen directamente en la memoria USB, se borra todo su contenido.

Elige la edición que corresponda al equipo o al servicio donde vas a instalar JustVoxel.

- **JustVoxel VM** para máquinas virtuales y servicios compatibles de alojamiento de máquinas virtuales
- **JustVoxel HWS** para PC, portátiles, mini PC, servidores domésticos físicos y servicios compatibles de alquiler de servidores físicos dedicados

Consulta la [guía de instalación](../docs/INSTALLER.md) para saber cómo instalar el sistema, preparar el disco e iniciar sesión por primera vez.

## 🏠 Un servidor para Minecraft que también admite otros usos

JustVoxel se desarrolla y se mantiene principalmente como sistema para alojar servidores de Minecraft. El sistema subyacente es un **servidor AlmaLinux 10 con bootc, Podman, Quadlet, systemd, SSH, NetworkManager y las herramientas habituales de administración de Linux**.

Los usuarios avanzados pueden utilizar el mismo equipo como servidor doméstico de uso más amplio y ejecutar otros contenedores junto a Minecraft.

Puedes desplegar manualmente servicios como los siguientes.

- **Jellyfin** o **Plex** para contenido multimedia
- **Vaultwarden** para gestionar contraseñas
- **Immich** para gestionar fotos y vídeos
- **Home Assistant Container** para domótica
- **Nextcloud** para compartir archivos y colaborar
- **Syncthing** o **Navidrome**
- **OpenClaw** como agente o pasarela de IA en tu propio servidor

Estas aplicaciones quedan **fuera de la gestión del centro de control de JustVoxel**. La persona que administra el equipo se encarga de sus contenedores Podman, Quadlets, almacenamiento, puertos, actualizaciones, copias de seguridad, configuración del proxy inverso y requisitos específicos de cada aplicación.

JustVoxel gestiona el sistema de Minecraft y deja el resto en manos de los usuarios avanzados, que pueden seguir utilizando Podman y Quadlet como de costumbre.

Durante el mantenimiento, JustVoxel trata los demás contenedores como servicios a cargo de quien administra el equipo. Los contenedores Podman ajenos a Minecraft quedan intactos al ejecutar operaciones como **Empezar de nuevo** en Minecraft.

Así, un equipo con JustVoxel puede dedicarse principalmente a Minecraft y funcionar también como servidor doméstico con contenedores cuando lo necesites.

## 🎮 Crea el servidor de Minecraft que quieras

JustVoxel puede ejecutar varios servidores de Minecraft independientes a la vez si el equipo dispone de suficiente memoria y de los puertos necesarios.

Cada servidor tiene su propia identidad, configuración, entorno de ejecución, puertos, datos, copias de seguridad, controles del ciclo de vida y estado.

### Software de servidor disponible

| Software de servidor | Tipo de contenido | Juego cruzado gestionado de Java a Bedrock |
| --- | --- | --- |
| **Paper** | Plugins | ✅ |
| **Purpur** | Plugins | ✅ |
| **Spigot** | Plugins | ✅ |
| **Pufferfish** | Plugins | ✅ |
| **Leaf** | Plugins | ✅ |
| **Fabric** | Mods | ✅ Si se confirma la compatibilidad exacta |
| **NeoForge** | Mods | ✅ Si se confirma la compatibilidad exacta |
| **Forge** | Mods | Solo Java |
| **Quilt** | Mods | Solo Java |
| **Vanilla Java** | Sin plugins ni mods | Solo Java |
| **Bedrock nativo** | Servidor Bedrock nativo | Clientes Bedrock nativos |

Elige **Configuración sencilla** para crear tu primer servidor paso a paso, o **Configuración avanzada** si quieres ajustar la memoria, los puertos, el almacenamiento, las copias de seguridad, la política de versiones y el juego cruzado cuando sea compatible.

JustVoxel comprueba qué funciones admite el software de servidor que has seleccionado.

[Consulta el software de servidor y los detalles de compatibilidad](../docs/server-software.md)

## 🧩 Plugins y mods

Abre **Minecraft → Plugins y mods** en el servidor seleccionado.

Según el tipo de servidor, JustVoxel puede utilizar los siguientes proveedores.

- [Modrinth](https://modrinth.com/)
- [Hangar](https://hangar.papermc.io/)
- [CurseForge](https://www.curseforge.com/minecraft) con la clave de API que hayas configurado

Las instalaciones admitidas incluyen comprobaciones de compatibilidad, descargas verificadas, copias de seguridad con el servidor parado, verificación tras el reinicio, un registro de los archivos que gestiona JustVoxel y medidas de recuperación.

Los servidores de la familia de Paper pueden usar catálogos de plugins. Fabric, Forge, NeoForge y Quilt utilizan catálogos de mods y comprueban la compatibilidad exacta con el cargador de mods y la versión de Minecraft. Por diseño, Vanilla Java y Bedrock nativo carecen de catálogo de plugins y mods.

[Más información sobre plugins y mods](../docs/plugin-workspace.md)

## 🌐 Red de Minecraft opcional

Puedes tener varios servidores independientes. **No necesitan un proxy.**

Si quieres ofrecer un único punto de acceso público a varios servidores compatibles, JustVoxel incluye una **Red de Minecraft** opcional basada en Velocity.

Desde el centro de control puedes seleccionar los servidores de destino compatibles, configurar los puertos públicos de Java y Bedrock, gestionar la sala de espera virtual y el mensaje de bienvenida, y decidir qué servidores forman parte de la red.

Paper, Purpur, Pufferfish y Leaf pueden formar parte de la red como servidores de destino. Los servidores compatibles de Fabric/Quilt y Forge/NeoForge también pueden participar si JustVoxel confirma que cumplen los requisitos de compatibilidad con el proxy. Spigot, Vanilla y Bedrock nativo siguen funcionando de forma independiente.

[Más información sobre la Red de Minecraft](../docs/minecraft-network.md)

## 🖥️ Conoce el centro de control

JustVoxel se organiza en torno a las tareas de gestión del servidor para que puedas realizarlas sin conocer los detalles internos de Linux.

### Servidores de Minecraft

Crea y gestiona varios servidores, inícialos o detenlos, consulta su estado y los jugadores, ajusta las opciones disponibles y accede a las herramientas de cada servidor.

### Plugins y mods

Busca contenido compatible, revisa los plugins y mods instalados, gestiona las incorporaciones admitidas y consulta la información de recuperación cuando un cambio requiera atención.

### Copias de seguridad y almacenamiento

Crea copias de seguridad, consulta el historial, restaura todos los datos de un servidor, configura dónde guardar las copias, importa y exporta servidores, y gestiona el almacenamiento compatible.

### Red

Gestiona Ethernet, Wi-Fi y el acceso remoto, y usa las herramientas de **diagnóstico de la conexión**. Los cambios de red que podrían interrumpir la conexión se pueden revertir para que un ajuste incorrecto de IP o Wi-Fi no te deje sin acceso de forma permanente.

### Sistema

Consulta el estado del sistema, los recursos, los registros, las actualizaciones del sistema operativo, los usuarios, la seguridad, las acciones de encendido y apagado disponibles y los controles de hardware exclusivos de HWS.

### Conexiones de IA y API

Conecta aplicaciones de IA compatibles, clientes MCP, automatizaciones mediante REST, Telegram y Discord sin tener que montar otro sistema de gestión.

¿Prefieres usar la terminal? JustVoxel también incluye **mjust**, una interfaz interactiva de administración desde la terminal que aplica las mismas medidas de seguridad del sistema.

[Documentación de la interfaz web](../docs/WEBUI.md) · [Guía de mjust](../docs/MJUST.md)

## 🌐 Elige cómo se conectan tus amigos

JustVoxel integra actualmente cuatro opciones de acceso remoto en **Red → Acceso remoto**.

Además de incluir los paquetes en la imagen, JustVoxel te permite consultar su estado, activarlos, desactivarlos y configurar la integración con el proveedor cuando corresponda.

| Opción | Ideal para | Qué ofrece |
| --- | --- | --- |
| **[Tailscale](https://tailscale.com/)** | Servidores privados para amigos y familiares | Red privada en malla entre dispositivos autorizados |
| **[NetBird](https://netbird.io/)** | Redes privadas con acceso gestionado | Red privada con control de cuentas y políticas de acceso |
| **[Playit.gg](https://playit.gg/)** | Acceso público a Minecraft sin redirigir puertos en el router | Túneles públicos para juegos y direcciones de conexión |
| **[NordVPN Meshnet](https://meshnet.nordvpn.com/)** | Acceso privado entre dispositivos para usuarios de NordVPN | Conexión por Meshnet sin usar JustVoxel como cliente de una VPN comercial |

### ¿Cuál me conviene?

**🎮 ¿Quieres que tus amigos se conecten sin instalar un cliente de VPN privada?**\
Echa un vistazo a **Playit.gg**. Los jugadores pueden usar la dirección pública del túnel que hayas configurado para el juego.

**🔒 ¿Quieres un servidor privado para dispositivos de confianza?**\
Valora **Tailscale** o **NetBird**.

**🌍 ¿Ya utilizas NordVPN Meshnet?**\
JustVoxel puede gestionar su integración con Meshnet. Es una función independiente de la conexión del equipo a un servidor de salida comercial de NordVPN.

Las cuentas, las políticas en la nube y la configuración del túnel se gestionan con el proveedor. JustVoxel se encarga de la parte de la integración que admite el equipo.

Puedes utilizar varios servicios de VPN o redes en malla a la vez, aunque pueden surgir conflictos de enrutamiento, DNS o reglas del cortafuegos.

[Consulta la guía de red y acceso remoto](../docs/NETWORKING.md)

## 🤖 IA, MCP, REST y automatización

¿Quieres consultar el estado del servidor con un asistente de IA, integrarlo en tu sistema domótico o crear tu propia automatización para controlarlo?

JustVoxel incluye **interfaces MCP y REST** en **Conexiones de IA y API**.

En la configuración puedes elegir entre aplicaciones compatibles y tipos de conexión como los siguientes.

- **OpenAI — ChatGPT y Codex**
- **Claude**
- **Grok**
- **Gemini**
- **GitHub Copilot**
- **Otra aplicación de IA o MCP**
- **Automatización / API REST**

Tú decides a qué puede acceder cada conexión.

Los permisos van desde el **Nivel 1 — Solo lectura** hasta la gestión de servidores y, únicamente con aprobación explícita, el control avanzado del sistema. También puedes limitar cada conexión a los servidores de Minecraft que selecciones.

Para ofrecer su interfaz MCP o REST, JustVoxel no necesita una clave de API de un modelo de IA. La compatibilidad con cada cliente depende de la aplicación, de las prestaciones de la cuenta o suscripción y del acceso a la red.

La pasarela de IA y API está desactivada hasta que la habilites.

[Más información sobre las conexiones de IA y API](../docs/AI-API-CONNECTIONS.md)

## 💬 Telegram y Discord

JustVoxel incluye integraciones nativas y opcionales con Telegram y Discord, que utilizan el mismo servicio central de gestión que la interfaz web.

### Telegram

Un bot dedicado de Telegram ofrece las siguientes funciones.

- consulta del estado de JustVoxel y de Minecraft
- selección del servidor y consulta de información sobre los jugadores
- inicio, parada y reinicio para usuarios con permisos de gestión
- copias de seguridad manuales
- revisión de las actualizaciones de Minecraft y de los contenedores
- información sobre las actualizaciones del sistema operativo
- procedimientos de reinicio controlado del equipo
- notificaciones opcionales sobre el estado del servidor, el resultado de las copias de seguridad y la disponibilidad de actualizaciones

Telegram utiliza sus propios permisos de **Solo lectura** y **Administración de servidores** y funciona sin un modelo de IA.

[Configuración y permisos de Telegram](../docs/TELEGRAM.md)

### Discord

Discord ofrece funciones tanto para la comunidad de Minecraft como para quienes administran el servidor.

Los miembros del servidor de Discord que se haya vinculado de forma explícita pueden consultar el estado básico del servidor y solicitar acceso a Minecraft. El acceso a estas funciones de comunidad no concede permisos de administración.

Los administradores de JustVoxel autorizados en Discord pueden realizar las acciones admitidas, como iniciar, detener o reiniciar servidores, crear copias de seguridad manuales, revisar actualizaciones, gestionar la Red de Minecraft y ejecutar otras acciones avanzadas.

La pertenencia al servidor de Discord, el acceso mediante la lista de jugadores permitidos de Minecraft y la administración de JustVoxel se rigen por permisos independientes.

[Configuración de Discord, acceso de la comunidad y administración](../docs/DISCORD.md)

## 💾 Copias de seguridad, restauración, almacenamiento y migración

Los datos de Minecraft importan más que la imagen de JustVoxel.

JustVoxel mantiene los datos de Minecraft y las copias de seguridad fuera de la instalación inmutable del sistema operativo. Puedes elegir por separado dónde guardar unos y otras.

Entre las opciones de almacenamiento admitidas se incluyen las siguientes.

- el sistema de archivos del equipo
- discos internos dedicados
- almacenamiento USB
- particiones o sistemas de archivos locales ya existentes
- particiones nuevas creadas en espacio que ya está sin asignar
- NFS
- SMB/CIFS

Cada servidor admite copias de seguridad automáticas y manuales. La restauración sustituye todos los datos persistentes del servidor seleccionado y conserva su identidad y configuración en JustVoxel.

Para trasladar servidores entre equipos o después de reinstalar el sistema, JustVoxel también ofrece procedimientos de **Exportación, importación y recuperación** con pasos de revisión.

[Almacenamiento](../docs/STORAGE.md) · [Restauración](../docs/RESTORE.md) · [Migración](../docs/MIGRATION.md)

## 🛡️ Diseñado para aplicar cambios con más seguridad

La interfaz web de JustVoxel ofrece operaciones definidas y evita la ejecución de comandos arbitrarios del intérprete de órdenes.

Las operaciones admitidas aplican medidas de seguridad como las siguientes.

- comprobación de si hay jugadores conectados antes de realizar acciones que puedan interrumpir el servicio
- revisión de los cambios antes de pulsar **Aplicar**
- copias de seguridad con el servidor parado cuando un cambio requiere protección para poder recuperar los datos
- comprobación de la identidad exacta de cada servidor para evitar que se confunda con otro
- verificación del reinicio y del funcionamiento del servidor
- conservación del estado de las operaciones y de la recuperación
- opciones para revertir las operaciones que lo admiten
- puntos de control de NetworkManager para cambios de red que entrañan riesgo
- procedimientos de actualización independientes para Minecraft y el sistema operativo

La interfaz web que se abre en el navegador funciona sin privilegios. Los cambios que requieren privilegios pasan por el agente de gestión local de JustVoxel y por las operaciones que ofrece su API. El navegador carece de acceso a un intérprete de órdenes con privilegios de root.

## 🔄 El sistema y Minecraft se actualizan por separado

JustVoxel se gestiona mediante imágenes y está basado en Home Server Base 10 / AlmaLinux 10.

Las actualizaciones del sistema operativo se gestionan por separado de las versiones de Minecraft, el software de servidor, los plugins, los mods y las actualizaciones de los contenedores.

Así puedes actualizar el equipo sin que cambie tu versión de Minecraft de forma inadvertida y modificar un servidor de Minecraft sin tener que volver a generar el sistema operativo del equipo.

[Documentación sobre la gestión del sistema](../docs/SYSTEM.md)

## 📊 ¿Cómo se compara JustVoxel con otros proyectos?

JustVoxel se diferencia de los paneles de control tradicionales de Minecraft porque **el sistema operativo y la gestión de Minecraft forman un único producto**.

| Función | **JustVoxel** | **MineOS SvelteKit** | **Crafty Controller** | **MineOS Node** |
| --- | :---: | :---: | :---: | :---: |
| Sistema operativo completo para alojar Minecraft | ⭐ | — | — | — |
| Varios servidores de Minecraft | ✅ | ✅ | ✅ | ✅ |
| Compatibilidad con servidores Java y Bedrock | ✅ | ✅ | ✅ | 🟡 Java y compatibilidad con versiones antiguas de PocketMine |
| Búsqueda guiada de plugins y mods | ✅ Modrinth, Hangar, CurseForge | ✅ Modrinth, CurseForge | — | — |
| Gestión de la red, el almacenamiento y el sistema operativo del equipo | ⭐ | — | — | — |
| Acceso privado y túneles para juegos integrados | ⭐ 4 opciones | — | — | — |
| Pasarela MCP y REST para IA y automatización | ⭐ | — | — | — |
| Gestión nativa mediante Telegram y Discord | ⭐ | — | — | — |
| Controles de tapa y pantalla para portátiles | ✅ HWS | — | — | — |
| Actualizaciones del sistema operativo inmutable mediante imágenes | ✅ | — | — | — |
| Interfaz de gestión desde la terminal | ✅ mjust | ✅ mineos CLI/TUI | — | ✅ CLI |
| Modelo de instalación | Sistema completo basado en bootc | Aplicación Docker / Compose | Panel de control en Docker, Linux o Windows | Aplicación Node.js en Linux/BSD |

⭐ destacado &nbsp;•&nbsp; ✅ compatible &nbsp;•&nbsp; 🟡 compatibilidad parcial o con versiones antiguas &nbsp;•&nbsp; — no figura en la documentación principal revisada del proyecto

**Última revisión de la comparativa (16 de noviembre de 2026).** La comparativa busca orientar, sin descalificar a los demás proyectos. MineOS y Crafty son aplicaciones para gestionar Minecraft que pueden ser excelentes opciones si ya tienes un sistema operativo que quieres administrar por tu cuenta. JustVoxel se dirige a quienes quieren gestionar **el panel de Minecraft y el sistema del servidor** como un solo producto.

La comparativa se basa en la documentación y los repositorios actuales de los proyectos.

- [MineOS SvelteKit](https://github.com/freeman412/mineos-sveltekit)
- [Crafty Controller](https://craftycontrol.com/) · [código fuente](https://gitlab.com/crafty-controller/crafty-4)
- [MineOS Node](https://github.com/hexparrot/mineos-node)

Si algún dato de la comparativa está desactualizado o es injusto, abre una incidencia. También agradecemos las correcciones aunque dejen a JustVoxel en peor lugar.

## 📚 Documentación

Las guías detalladas de JustVoxel están en el [directorio de documentación](../docs/).

Puedes empezar por estas guías.

- [Interfaz web](../docs/WEBUI.md)
- [Entorno de ejecución de Minecraft](../docs/MINECRAFT_RUNTIME.md)
- [Software de servidor](../docs/server-software.md)
- [Plugins y mods](../docs/plugin-workspace.md)
- [Red de Minecraft](../docs/minecraft-network.md)
- [Red y acceso remoto](../docs/NETWORKING.md)
- [Conexiones de IA y API](../docs/AI-API-CONNECTIONS.md)
- [Telegram](../docs/TELEGRAM.md)
- [Discord](../docs/DISCORD.md)
- [Almacenamiento](../docs/STORAGE.md)
- [Restauración](../docs/RESTORE.md)
- [Migración](../docs/MIGRATION.md)
- [Gestión del sistema](../docs/SYSTEM.md)
- [mjust](../docs/MJUST.md)

Para obtener información sobre el desarrollo de imágenes, las pruebas y los lanzamientos, consulta la [guía de desarrollo](../README-DEV.md).

## ⛏️ Software de Minecraft y EULA

JustVoxel **no** incluye binarios de servidor de Minecraft, software de servidor de Mojang, un mundo creado de antemano ni un EULA de Minecraft aceptado previamente.

Al crear el servidor, la persona que lo administra acepta el EULA de Minecraft y JustVoxel prepara el entorno de ejecución del software de servidor compatible que se haya seleccionado.

Minecraft, el software de Mojang, Paper, Purpur, Spigot, Pufferfish, Leaf, Fabric, NeoForge, Forge, Quilt, los plugins, los mods y otros componentes de terceros conservan sus propias licencias y condiciones de distribución.

## 📄 Licencia

JustVoxel se distribuye bajo la licencia **Apache-2.0**. Consulta [LICENSE](../LICENSE).

Los avisos de terceros se recogen en [THIRD_PARTY_NOTICES.md](../THIRD_PARTY_NOTICES.md).
