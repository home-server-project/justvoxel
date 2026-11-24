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

> **Nota sobre la traducción:** Esta traducción de JustVoxel se hizo a partir del texto original en inglés con ayuda de IA. Si encuentras una frase imprecisa, poco clara o que suene poco natural, propón una corrección en un [reporte de problema en GitHub](https://github.com/home-server-project/justvoxel/issues) o en una [solicitud de cambios (pull request)](https://github.com/home-server-project/justvoxel/pulls).

<p align="center">
  <strong>Tus servidores de Minecraft. Tu propio equipo, un servidor dedicado de alquiler o una máquina virtual en la nube. Un solo centro de control.</strong>
</p>

<p align="center">
  Un sistema dedicado para alojar Minecraft por tu cuenta y concentrarte en tu servidor sin tener que administrar el sistema operativo.
</p>

<p align="center">
  <a href="#-instala-justvoxel">Instalación</a> •
  <a href="#-alojar-servidores-de-minecraft-es-más-fácil">Funciones</a> •
  <a href="../docs/INSTALLER.md">Documentación</a>
</p>

<p align="center">
  <img src="../docs/images/justvoxel-overview-1.webp" width="100%" alt="Vista general del centro de control de JustVoxel"><img src="../docs/images/justvoxel-overview-2.webp" width="100%" alt=""><img src="../docs/images/justvoxel-overview-3.webp" width="100%" alt="">
</p>

JustVoxel es un sistema dedicado para alojar tus propios servidores de Minecraft. Instálalo en una PC, una laptop, un servidor físico o una máquina virtual compatibles, o en un servicio de alojamiento que te permita usar tu propia imagen del sistema operativo. Después podrás administrar Minecraft desde el navegador.

No necesitas mantener una colección de scripts ni instalar un panel de administración de Minecraft por separado.

¿Tienes una laptop o una PC vieja? **Con JustVoxel HWS puedes convertirla en un equipo dedicado a Minecraft y aprovecharla en vez de dejarla guardada en un cajón.** En las laptops compatibles puedes evitar que el equipo entre en suspensión al cerrar la tapa para que el servidor siga funcionando. También puedes apagar la pantalla de la consola local de inmediato o después de un breve intervalo.

> 🚧 **Versión preliminar:** JustVoxel se prepara para su primer lanzamiento público. La imagen ISO de instalación actual está disponible para pruebas y uso anticipado mientras continúa la validación final.

## ✨ Alojar servidores de Minecraft es más fácil

JustVoxel integra la administración de Minecraft y la del sistema operativo que lo ejecuta.

- 🎮 **Varios servidores de Minecraft** — ejecuta distintos mundos y tipos de servidor de forma independiente en un mismo equipo.
- 🖥️ **Centro de control web** — administra tus servidores desde una computadora, una tableta o un teléfono.
- 🧩 **Plugins y mods** — encuentra y administra contenido compatible de los proveedores admitidos.
- 💾 **Copias de seguridad y restauración** — protege cada servidor con copias programadas y manuales.
- 🌐 **Acceso remoto** — elige entre cuatro opciones integradas de acceso privado y túneles para juegos.
- 🤖 **IA y automatización** — conecta asistentes de IA y aplicaciones compatibles mediante MCP y REST.
- 💬 **Telegram y Discord** — usa controles opcionales mediante bots, recibe notificaciones y gestiona solicitudes de la comunidad.
- 🛡️ **Administración del sistema** — gestiona la red, el almacenamiento, las actualizaciones, el estado del sistema, los registros y las opciones de energía disponibles sin tener que aprender los comandos de Linux.

JustVoxel está pensado para familias, grupos de amigos, pequeñas comunidades de Minecraft y quienes prefieren tener su propio servidor en vez de alquilar un panel de alojamiento tradicional de Minecraft.

## 🖥️ Un sistema, dos ediciones

### JustVoxel VM

Para máquinas virtuales y entornos de alojamiento como los siguientes:

- Proxmox
- KVM / libvirt
- VMware
- Hyper-V
- VirtualBox
- máquinas virtuales alojadas o servicios VPS que admitan una imagen personalizada compatible del sistema operativo

La edición VM incluye el sistema completo para Minecraft sin los paquetes de hardware físico que no tienen utilidad dentro de una máquina virtual.

### JustVoxel HWS

Para instalar directamente en equipos físicos, por ejemplo:

- una laptop vieja
- una computadora de escritorio que quieras reutilizar
- una mini PC
- un servidor doméstico
- un servidor físico dedicado de alquiler que permita usar medios de instalación personalizados

HWS agrega funciones para equipos físicos, como integración con UPS/NUT, herramientas SMART y NVMe, sensores, controles de firmware/UEFI, firmware para dispositivos, diagnóstico de hardware y controles de pantalla y tapa para laptops.

Ambas ediciones comparten el centro de control de JustVoxel, la API de administración, los procedimientos de Minecraft, el modelo de almacenamiento, las copias de seguridad, las funciones de red y el proceso de actualización.

[Consulta las funciones de las ediciones VM y HWS](../docs/VARIANTS.md)

## 🚀 Instala JustVoxel

Una sola imagen ISO de instalación sirve para ambas ediciones de JustVoxel.

[⬇️ Descargar la ISO de JustVoxel para x86_64](https://sourceforge.net/projects/justvoxel/files/JustVoxel-x86_64.iso/download)

Elige la edición según el equipo donde vayas a instalarla:

- **JustVoxel VM** para máquinas virtuales y entornos compatibles de alojamiento de máquinas virtuales
- **JustVoxel HWS** para PC, laptops, mini PC, servidores domésticos físicos y servicios compatibles de alojamiento en servidores físicos dedicados

Consulta la [guía de instalación](../docs/INSTALLER.md) para preparar el disco, instalar JustVoxel e iniciar sesión por primera vez.

## 🏠 Minecraft es la prioridad, pero hay espacio para más

El diseño y el soporte de JustVoxel se centran en su función de servidor de Minecraft. La base sigue siendo un **servidor AlmaLinux 10 bootc con Podman, Quadlet, systemd, SSH, NetworkManager y las herramientas habituales de administración de Linux**.

Los usuarios avanzados pueden aprovechar el mismo equipo como servidor doméstico de uso más amplio y ejecutar otros contenedores junto con Minecraft.

Por ejemplo, puedes instalar manualmente servicios como estos:

- **Jellyfin** o **Plex** para contenido multimedia
- **Vaultwarden** para administrar contraseñas
- **Immich** para organizar fotos y videos
- **Home Assistant Container** para automatizar tu hogar
- **Nextcloud** para trabajar con archivos y colaborar
- **Syncthing** o **Navidrome**
- **OpenClaw** como agente o puerta de enlace de IA en tu propio equipo

El **centro de control de JustVoxel no administra estas aplicaciones**. El administrador se encarga de sus contenedores Podman, Quadlets, almacenamiento, puertos, actualizaciones, copias de seguridad, configuración del proxy inverso y requisitos específicos.

JustVoxel mantiene esta división de forma deliberada. Administra el sistema dedicado a Minecraft y permite que los usuarios avanzados utilicen Podman y Quadlet de la forma habitual para todo lo demás.

Las tareas de mantenimiento de JustVoxel también reconocen que los contenedores ajenos a Minecraft están a cargo del administrador. Operaciones de Minecraft como **Volver a empezar** no eliminan esos contenedores Podman.

Así, tu equipo puede seguir dedicado principalmente a Minecraft y, si lo deseas, servir también como servidor doméstico convencional con contenedores.

## 🎮 Crea el servidor de Minecraft que quieras

JustVoxel puede ejecutar varios servidores de Minecraft independientes al mismo tiempo si el equipo tiene suficiente memoria y puertos disponibles.

Cada servidor tiene su propia identidad, configuración, entorno de ejecución, puertos, datos, copias de seguridad, controles de inicio y detención, y estado.

### Software de servidor disponible

| Software de servidor | Tipo de contenido | Juego cruzado Java → Bedrock administrado por JustVoxel |
| --- | --- | --- |
| **Paper** | Plugins | ✅ |
| **Purpur** | Plugins | ✅ |
| **Spigot** | Plugins | ✅ |
| **Pufferfish** | Plugins | ✅ |
| **Leaf** | Plugins | ✅ |
| **Fabric** | Mods | ✅ Si se verifica la compatibilidad exacta |
| **NeoForge** | Mods | ✅ Si se verifica la compatibilidad exacta |
| **Forge** | Mods | Solo Java |
| **Quilt** | Mods | Solo Java |
| **Vanilla Java** | Sin plugins ni mods | Solo Java |
| **Bedrock nativo** | Servidor Bedrock nativo | Clientes Bedrock nativos |

Elige **Configuración simple** para crear tu primer servidor paso a paso, o **Configuración avanzada** si quieres más control sobre la memoria, los puertos, el almacenamiento, las copias de seguridad, la política de versiones y las opciones compatibles de juego cruzado.

JustVoxel comprueba qué funciones admite el software de servidor seleccionado, ya que la compatibilidad varía según el servidor.

[Consulta los detalles sobre el software de servidor y su compatibilidad](../docs/server-software.md)

## 🧩 Plugins y mods

Abre **Minecraft → Plugins y mods** para el servidor seleccionado.

Según el tipo de servidor, JustVoxel puede usar:

- [Modrinth](https://modrinth.com/)
- [Hangar](https://hangar.papermc.io/)
- [CurseForge](https://www.curseforge.com/minecraft) con la clave de API que hayas configurado

Las instalaciones admitidas incluyen comprobaciones de compatibilidad, descargas verificadas, copias de seguridad con el servidor detenido, verificación del reinicio, seguimiento de la propiedad de los componentes instalados y medidas de recuperación.

Los servidores de la familia Paper pueden usar catálogos de plugins. Fabric, Forge, NeoForge y Quilt usan catálogos de mods y verifican que el cargador y la versión de Minecraft coincidan exactamente. Por diseño, Vanilla Java y Bedrock nativo no muestran catálogos de plugins ni de mods.

[Más información sobre plugins y mods](../docs/plugin-workspace.md)

## 🌐 Red de Minecraft opcional

Puedes tener varios servidores independientes. **No** necesitan un proxy.

Si quieres ofrecer un único punto de acceso público a varios servidores compatibles, JustVoxel incluye una **red de Minecraft** opcional basada en Velocity.

Desde el centro de control puedes seleccionar los servidores de destino compatibles, configurar los puertos públicos de Java y Bedrock, administrar el vestíbulo virtual y el mensaje de bienvenida, y controlar qué servidores forman parte de la red.

Paper, Purpur, Pufferfish y Leaf pueden funcionar como servidores de destino en la red. Los servidores compatibles de Fabric/Quilt y Forge/NeoForge también pueden participar cuando JustVoxel verifica la compatibilidad necesaria con el proxy. Spigot, Vanilla y Bedrock nativo funcionan de forma independiente.

[Más información sobre la red de Minecraft](../docs/minecraft-network.md)

## 🖥️ Conoce el centro de control

JustVoxel está organizado según las tareas que realizas en el servidor, sin que tengas que conocer el funcionamiento interno de Linux.

### Servidores de Minecraft

Crea y administra varios servidores, inícialos o detenlos, consulta su estado y quién está jugando, ajusta la configuración admitida y accede a las herramientas propias de cada servidor.

### Plugins y mods

Descubre contenido compatible, revisa los plugins y mods instalados, administra los que puedes agregar y consulta la información de recuperación cuando un cambio requiera atención.

### Copias de seguridad y almacenamiento

Crea copias de seguridad, consulta su historial, restaura todos los datos de un servidor, configura los destinos de las copias, importa o exporta servidores y administra el almacenamiento compatible.

### Red

Administra Ethernet, Wi-Fi, la **solución de problemas de conexión** y el acceso remoto. Los cambios de red que podrían interrumpir la conexión cuentan con protección para revertirlos y evitar que una configuración incorrecta de IP o Wi-Fi te deje permanentemente sin acceso.

### Sistema

Consulta el estado del sistema, los recursos, los registros, las actualizaciones del sistema operativo, los usuarios, la seguridad, las acciones de energía disponibles y los controles de hardware exclusivos de HWS.

### Conexiones de IA y API

Conecta aplicaciones de IA compatibles, clientes MCP, automatizaciones REST, Telegram y Discord sin tener que montar otro sistema de administración.

¿Prefieres la terminal? JustVoxel también incluye **mjust**, una interfaz de administración interactiva para la terminal que aplica las mismas medidas de seguridad del equipo.

[Documentación de la interfaz web](../docs/WEBUI.md) · [Guía de mjust](../docs/MJUST.md)

## 🌐 Elige cómo se conectan tus amigos

JustVoxel integra cuatro opciones en **Red → Acceso remoto**.

Los servicios vienen integrados en la imagen. JustVoxel te permite consultar su estado, activarlos o desactivarlos y configurarlos según las funciones admitidas por cada proveedor.

| Opción | Ideal para | Qué ofrece |
| --- | --- | --- |
| **[Tailscale](https://tailscale.com/)** | Servidores privados para amigos y familiares | Red privada en malla entre dispositivos autorizados |
| **[NetBird](https://netbird.io/)** | Redes privadas con acceso administrado | Red privada con control de cuentas y políticas de acceso |
| **[Playit.gg](https://playit.gg/)** | Acceso público a Minecraft sin configurar el reenvío de puertos habitual en el router | Túneles públicos para juegos y direcciones de conexión |
| **[NordVPN Meshnet](https://meshnet.nordvpn.com/)** | Acceso privado entre dispositivos para usuarios de NordVPN | Conexión mediante Meshnet sin usar JustVoxel como cliente de VPN comercial |

### ¿Cuál me conviene?

**🎮 ¿Quieres que tus amigos entren sin instalar un cliente de VPN privada?**  
Empieza por **Playit.gg**. Los jugadores pueden usar la dirección pública del túnel de juego que configures.

**🔒 ¿Quieres un servidor privado para dispositivos de confianza?**  
Considera **Tailscale** o **NetBird**.

**🌍 ¿Ya usas NordVPN Meshnet?**  
JustVoxel puede administrar su integración con Meshnet. Esta función es independiente de la conexión del equipo a un servidor de salida convencional de la VPN comercial de NordVPN.

Las cuentas, las políticas en la nube y la configuración de los túneles se gestionan con el proveedor. JustVoxel se encarga de las funciones de integración admitidas en el equipo.

Puedes usar varios servicios de VPN o de red en malla al mismo tiempo, aunque pueden surgir conflictos de enrutamiento, DNS o reglas del firewall.

[Consulta la guía de redes y acceso remoto](../docs/NETWORKING.md)

## 🤖 IA, MCP, REST y automatización

¿Quieres preguntarle a un asistente de IA por el estado del servidor, conectar la automatización de tu hogar o crear tu propio flujo de control?

JustVoxel ofrece **interfaces MCP y REST** integradas en **Conexiones de IA y API**.

Al configurar una conexión puedes elegir entre aplicaciones compatibles y tipos de conexión como los siguientes:

- **OpenAI — ChatGPT y Codex**
- **Claude**
- **Grok**
- **Gemini**
- **GitHub Copilot**
- **Otra aplicación de IA / MCP**
- **Automatización / API REST**

Tú decides a qué puede acceder cada conexión.

Los permisos parten del **Nivel 1 — Solo lectura** e incluyen funciones de administración de servidores y, únicamente con aprobación explícita, control avanzado del sistema. También puedes limitar una conexión a determinados servidores de Minecraft.

JustVoxel no necesita una clave de API de un modelo de IA para ofrecer sus interfaces MCP o REST. La compatibilidad de cada cliente depende de la aplicación, las funciones de la cuenta o suscripción y el acceso a la red.

La puerta de enlace de IA y API está desactivada hasta que la habilites.

[Más información sobre las conexiones de IA y API](../docs/AI-API-CONNECTIONS.md)

## 💬 Telegram y Discord

JustVoxel incluye integraciones nativas opcionales para Telegram y Discord. Ambas utilizan el mismo servicio central de administración que la interfaz web.

### Telegram

Un bot dedicado de Telegram puede ofrecerte:

- el estado del equipo y de Minecraft
- la selección de servidores e información sobre los jugadores
- opciones de inicio, detención y reinicio para usuarios autorizados a administrar servidores
- copias de seguridad manuales
- revisión de actualizaciones de Minecraft y de contenedores
- información sobre las actualizaciones del sistema operativo
- procedimientos de reinicio controlado del equipo
- notificaciones opcionales sobre el estado del servidor, los resultados de las copias de seguridad y las actualizaciones disponibles

Telegram tiene sus propios permisos de **Solo lectura** y **Administración de servidores** y no requiere un modelo de IA.

[Configuración y permisos de Telegram](../docs/TELEGRAM.md)

### Discord

Discord puede servir como punto de encuentro para la comunidad de Minecraft y como interfaz de administración.

Los miembros del servidor de Discord vinculado explícitamente pueden consultar el estado básico del servidor y solicitar acceso a Minecraft sin recibir permisos de administración automáticamente.

Los administradores autorizados de JustVoxel en Discord pueden iniciar, detener y reiniciar servidores, crear copias de seguridad manuales, revisar actualizaciones, controlar la red de Minecraft y realizar otras acciones avanzadas admitidas.

La pertenencia al servidor de Discord, el acceso mediante la lista de jugadores permitidos de Minecraft y la administración de JustVoxel se gestionan con permisos independientes.

[Configuración de Discord, acceso de la comunidad y administración](../docs/DISCORD.md)

## 💾 Copias de seguridad, restauración, almacenamiento y migración

Los datos de Minecraft son más importantes que la imagen del sistema.

JustVoxel mantiene los datos de Minecraft y las copias de seguridad fuera de la instalación inmutable del sistema operativo. Puedes elegir por separado dónde almacenar los datos y dónde guardar sus copias.

Las opciones de almacenamiento admitidas incluyen:

- el sistema de archivos del sistema operativo
- discos internos dedicados
- almacenamiento USB
- particiones o sistemas de archivos locales existentes
- particiones nuevas creadas en espacio que ya estaba sin asignar
- NFS
- SMB/CIFS

Cada servidor admite copias de seguridad automáticas y manuales. La restauración reemplaza **todos los datos persistentes** del servidor seleccionado y conserva su identidad en JustVoxel y su configuración en el equipo.

Para trasladar servidores entre equipos o recuperarlos tras una reinstalación, JustVoxel también ofrece procedimientos de **exportación, importación y recuperación** que incluyen una revisión.

[Almacenamiento](../docs/STORAGE.md) · [Restauración](../docs/RESTORE.md) · [Migración](../docs/MIGRATION.md)

## 🛡️ Cambios con mayor seguridad

JustVoxel va más allá de una interfaz atractiva que ejecuta comandos arbitrarios de shell.

Las operaciones admitidas aplican medidas de seguridad en el propio equipo, entre ellas:

- comprobar si hay jugadores conectados antes de realizar acciones que puedan interrumpir el juego
- revisar los cambios antes de aplicarlos
- crear copias de seguridad con el servidor detenido cuando un cambio necesita protección para poder recuperarse
- verificar la identidad exacta de cada servidor para evitar confundirlo con otro
- verificar el reinicio y el funcionamiento del servidor
- conservar el estado de las operaciones y de la recuperación de forma persistente
- disponer de mecanismos para revertir las operaciones admitidas
- usar puntos de control de NetworkManager para cambios de red que puedan interrumpir el acceso
- actualizar Minecraft y el sistema operativo mediante procedimientos independientes

La interfaz web que se abre en el navegador funciona sin privilegios. Los cambios que requieren privilegios pasan por el agente de administración local de JustVoxel y su API explícita, sin exponer al navegador una consola con acceso root.

## 🔄 Actualizaciones del sistema independientes de las de Minecraft

JustVoxel se administra mediante imágenes del sistema y se basa en Home Server Base 10 / AlmaLinux 10.

Las actualizaciones del sistema operativo se gestionan por separado de las versiones de Minecraft y de las actualizaciones del software de servidor, los plugins, los mods y los contenedores.

Así, actualizar el equipo no cambia silenciosamente tu versión de Minecraft, y modificar un servidor de Minecraft no exige reconstruir el sistema operativo del equipo.

[Documentación de administración del sistema](../docs/SYSTEM.md)

## 📊 ¿Cómo se compara JustVoxel con otros proyectos?

JustVoxel se diferencia de un panel de control tradicional de Minecraft porque **el sistema operativo y las herramientas de administración de Minecraft forman un solo producto**.

| Función | **JustVoxel** | **MineOS SvelteKit** | **Crafty Controller** | **MineOS Node** |
| --- | :---: | :---: | :---: | :---: |
| Sistema operativo completo dedicado a Minecraft | ⭐ | — | — | — |
| Varios servidores de Minecraft | ✅ | ✅ | ✅ | ✅ |
| Compatibilidad con servidores Java y Bedrock | ✅ | ✅ | ✅ | 🟡 Java y compatibilidad heredada con PocketMine |
| Búsqueda guiada de plugins y mods | ✅ Modrinth, Hangar, CurseForge | ✅ Modrinth, CurseForge | — | — |
| Administración de la red, el almacenamiento y el sistema operativo del equipo | ⭐ | — | — | — |
| Acceso privado y túneles para juegos integrados | ⭐ 4 opciones | — | — | — |
| Puerta de enlace MCP y REST para IA y automatización | ⭐ | — | — | — |
| Administración integrada mediante Telegram y Discord | ⭐ | — | — | — |
| Controles de tapa y pantalla para laptops | ✅ HWS | — | — | — |
| Actualizaciones del sistema operativo inmutable mediante imágenes | ✅ | — | — | — |
| Interfaz de administración para la terminal | ✅ mjust | ✅ CLI/TUI de mineos | — | ✅ CLI |
| Modelo de instalación | Sistema dedicado completo con bootc | Aplicación Docker / Compose | Panel de control en Docker, Linux o Windows | Aplicación Node.js en Linux/BSD |

⭐ función destacada &nbsp;•&nbsp; ✅ compatible &nbsp;•&nbsp; 🟡 compatibilidad parcial o heredada &nbsp;•&nbsp; — no figura en la documentación principal del proyecto que se revisó

**Última revisión de la comparación: 16 de noviembre de 2026.** El objetivo no es descalificar a los demás proyectos. MineOS y Crafty son aplicaciones de administración de Minecraft que pueden ser excelentes opciones si ya tienes un sistema operativo que quieres administrar por tu cuenta. JustVoxel está pensado para quienes quieren administrar el panel de Minecraft **y el sistema que lo ejecuta** como un solo producto.

La comparación se basa en la documentación y los repositorios actuales de los proyectos:

- [MineOS SvelteKit](https://github.com/freeman412/mineos-sveltekit)
- [Crafty Controller](https://craftycontrol.com/) · [código fuente](https://gitlab.com/crafty-controller/crafty-4)
- [MineOS Node](https://github.com/hexparrot/mineos-node)

Si algún dato de la comparación está desactualizado o te parece injusto, abre una incidencia. También son bienvenidas las correcciones que dejen a JustVoxel en una posición menos favorable.

## 📚 Documentación

Las guías detalladas de JustVoxel están en el [directorio de documentación](../docs/).

Puedes empezar por estas guías:

- [Interfaz web](../docs/WEBUI.md)
- [Entorno de ejecución de Minecraft](../docs/MINECRAFT_RUNTIME.md)
- [Software de servidor](../docs/server-software.md)
- [Plugins y mods](../docs/plugin-workspace.md)
- [Red de Minecraft](../docs/minecraft-network.md)
- [Redes y acceso remoto](../docs/NETWORKING.md)
- [Conexiones de IA y API](../docs/AI-API-CONNECTIONS.md)
- [Telegram](../docs/TELEGRAM.md)
- [Discord](../docs/DISCORD.md)
- [Almacenamiento](../docs/STORAGE.md)
- [Restauración](../docs/RESTORE.md)
- [Migración](../docs/MIGRATION.md)
- [Administración del sistema](../docs/SYSTEM.md)
- [mjust](../docs/MJUST.md)

Para obtener información sobre el desarrollo de imágenes, las pruebas y los lanzamientos, consulta la [guía para desarrolladores](../README-DEV.md).

## ⛏️ Software de Minecraft y EULA

JustVoxel **no** distribuye binarios del servidor de Minecraft, software de servidor de Mojang, mundos creados de antemano ni una copia del contrato de licencia de usuario final (EULA) de Minecraft aceptada de antemano.

Al crear el servidor, el administrador acepta el EULA de Minecraft y JustVoxel prepara el entorno de ejecución compatible que se haya seleccionado para el servidor.

Minecraft, el software de Mojang, Paper, Purpur, Spigot, Pufferfish, Leaf, Fabric, NeoForge, Forge, Quilt, los plugins, los mods y los demás componentes de terceros conservan sus propias licencias y condiciones de distribución.

## 📄 Licencia

JustVoxel se distribuye bajo la licencia **Apache-2.0**. Consulta el archivo [LICENSE](../LICENSE).

Los avisos sobre componentes de terceros se mantienen en [THIRD_PARTY_NOTICES.md](../THIRD_PARTY_NOTICES.md).
