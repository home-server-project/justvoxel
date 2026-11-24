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

> **Nota sobre a tradução:** Esta tradução do JustVoxel foi produzida com auxílio de IA a partir do texto original em inglês. Se encontrar algum trecho impreciso, pouco claro ou que soe estranho, proponha uma correção em um [relato de problema no GitHub](https://github.com/home-server-project/justvoxel/issues) ou em uma [solicitação de alteração (pull request)](https://github.com/home-server-project/justvoxel/pulls).

<p align="center">
  <strong>Seus servidores de Minecraft. No seu próprio equipamento, em um servidor dedicado alugado ou em uma VM hospedada. Tudo em uma única Central de Controle.</strong>
</p>

<p align="center">
  Um sistema dedicado de hospedagem própria para quem quer cuidar do servidor de Minecraft sem precisar administrar o sistema operacional por trás dele.
</p>

<p align="center">
  <a href="#-instale-o-justvoxel">Instalação</a> •
  <a href="#-hospedar-minecraft-ficou-mais-simples">Recursos</a> •
  <a href="../docs/INSTALLER.md">Documentação</a>
</p>

<p align="center">
  <img src="../docs/images/justvoxel-overview-1.webp" width="100%" alt="Visão geral da Central de Controle do JustVoxel"><img src="../docs/images/justvoxel-overview-2.webp" width="100%" alt=""><img src="../docs/images/justvoxel-overview-3.webp" width="100%" alt="">
</p>

O JustVoxel é um sistema dedicado para hospedar seus próprios servidores de Minecraft. Instale-o em um PC, notebook, servidor físico, máquina virtual ou serviço de hospedagem compatível que permita usar sua própria imagem de sistema operacional. Depois, gerencie o Minecraft pelo navegador.

Você não precisa manter uma coleção de scripts nem instalar um painel de gerenciamento de Minecraft à parte.

Tem um notebook ou PC antigo? **Com o JustVoxel HWS, você pode transformar esse equipamento em um servidor dedicado de Minecraft em vez de deixá-lo parado na gaveta.** Em notebooks compatíveis, você pode desativar a suspensão ao fechar a tampa para manter o servidor em funcionamento. Os controles de tela do console local também permitem apagar a tela imediatamente ou após um breve intervalo.

> 🚧 **Pré-lançamento:** O JustVoxel está em preparação para seu primeiro lançamento público. A ISO de instalação atual está disponível para testes e uso antecipado enquanto a validação final da versão continua.

## ✨ Hospedar Minecraft ficou mais simples

O JustVoxel reúne o gerenciamento do Minecraft e o do sistema operacional que o executa.

- 🎮 **Vários servidores de Minecraft** — execute mundos e tipos de servidor diferentes de forma independente no mesmo equipamento.
- 🖥️ **Central de Controle na web** — gerencie os servidores pelo computador, tablet ou celular.
- 🧩 **Plugins e mods** — encontre e gerencie conteúdo compatível dos provedores disponíveis.
- 💾 **Backups e restauração** — proteja cada servidor com backups agendados e manuais.
- 🌐 **Acesso remoto** — escolha entre quatro opções integradas de acesso privado e túneis para jogos.
- 🤖 **IA e automação** — conecte assistentes de IA e aplicativos compatíveis via MCP e REST.
- 💬 **Telegram e Discord** — use controles opcionais por bot, notificações e recursos para a comunidade.
- 🛡️ **Gerenciamento do sistema** — gerencie rede, armazenamento, atualizações, integridade do sistema, logs e opções de energia compatíveis sem precisar aprender os comandos do Linux.

O JustVoxel foi pensado para famílias, amigos, pequenas comunidades de Minecraft e quem quer ter seu próprio servidor em vez de alugar uma hospedagem tradicional de Minecraft com painel.

## 🖥️ Um sistema, duas edições

### JustVoxel VM

Para máquinas virtuais e ambientes de hospedagem como:

- Proxmox
- KVM / libvirt
- VMware
- Hyper-V
- VirtualBox
- VMs hospedadas ou serviços de VPS que aceitem uma imagem de sistema operacional personalizada e compatível

A edição VM inclui o sistema dedicado de Minecraft completo, sem os pacotes de hardware físico que não têm utilidade dentro de uma VM.

### JustVoxel HWS

Para instalação diretamente em equipamentos físicos, como:

- um notebook antigo
- um computador de mesa reaproveitado
- um mini PC
- um servidor doméstico
- um servidor físico dedicado (bare metal) alugado que aceite mídia de instalação personalizada

A edição HWS acrescenta recursos para máquinas físicas, como integração com nobreaks via NUT, ferramentas SMART e NVMe, sensores, controles de firmware/UEFI, firmware de dispositivos, diagnóstico de hardware e controles da tela e da tampa de notebooks.

As duas edições usam a mesma Central de Controle do JustVoxel, a mesma API de gerenciamento e os mesmos recursos de Minecraft, armazenamento, backup, rede e atualização.

[Conheça os recursos das edições VM e HWS](../docs/VARIANTS.md)

## 🚀 Instale o JustVoxel

Uma única ISO de instalação atende às duas edições do JustVoxel.

[⬇️ Baixe a ISO x86_64 do JustVoxel](https://sourceforge.net/projects/justvoxel/files/JustVoxel-x86_64.iso/download)

**Crie um pendrive inicializável:** Grave a imagem ISO em um pendrive com [balenaEtcher](https://etcher.balena.io/), [Rufus](https://rufus.ie/en/), [Ventoy](https://www.ventoy.net/en/index.html), [Fedora Media Writer](https://fedoraproject.org/workstation/download/) ou outra ferramenta capaz de gravar imagens de disco diretamente. No Linux e em outros sistemas semelhantes ao Unix, o método tradicional com `dd` também funciona. Gravar uma imagem diretamente no pendrive apaga todo o seu conteúdo.

Escolha a edição de acordo com o local de instalação:

- **JustVoxel VM** para máquinas virtuais e ambientes compatíveis de hospedagem de VMs
- **JustVoxel HWS** para PCs, notebooks, mini PCs, servidores domésticos físicos e hospedagem bare metal compatível

Consulte o [guia de instalação](../docs/INSTALLER.md) para preparar o disco, instalar o sistema e fazer o primeiro login.

## 🏠 Minecraft em primeiro lugar, com espaço para outros serviços

O foco do desenvolvimento e do suporte do JustVoxel é oferecer um sistema dedicado a servidores de Minecraft. Por trás dele há um **servidor AlmaLinux 10 bootc convencional, com Podman, Quadlet, systemd, SSH, NetworkManager e as ferramentas padrão de administração do Linux**.

Usuários avançados também podem usar a máquina como servidor doméstico para outras finalidades e executar contêineres adicionais ao lado do Minecraft.

Por exemplo, você pode instalar manualmente serviços como:

- **Jellyfin** ou **Plex** para mídia
- **Vaultwarden** para gerenciar senhas
- **Immich** para gerenciar fotos e vídeos
- **Home Assistant Container** para automação residencial
- **Nextcloud** para arquivos e colaboração
- **Syncthing** ou **Navidrome**
- **OpenClaw** como agente ou gateway de IA com hospedagem própria

Esses aplicativos **não são gerenciados pela Central de Controle do JustVoxel**. O administrador continua responsável pelos contêineres Podman, Quadlets, armazenamento, portas, atualizações, backups, configuração de proxy reverso e requisitos específicos de cada aplicativo.

Essa divisão é intencional. O JustVoxel gerencia o sistema dedicado ao Minecraft, e usuários avançados podem usar os procedimentos habituais do Podman e do Quadlet para as demais tarefas.

Durante a manutenção, o JustVoxel trata os contêineres que não fazem parte do Minecraft como serviços sob responsabilidade do administrador. Operações como **Recomeçar** no Minecraft não removem esses outros contêineres Podman.

Assim, a máquina pode continuar dedicada principalmente ao Minecraft e também funcionar como um servidor doméstico convencional com contêineres, se você quiser.

## 🎮 Crie o servidor de Minecraft que você quer

O JustVoxel pode executar vários servidores de Minecraft independentes ao mesmo tempo, desde que a máquina tenha memória suficiente e portas disponíveis.

Cada servidor tem sua própria identidade, configuração, ambiente de execução, portas, dados, backups, controles de ciclo de vida e status.

### Softwares de servidor disponíveis

| Software de servidor | Tipo de conteúdo | Jogo entre edições Java → Bedrock gerenciado pelo JustVoxel |
| --- | --- | --- |
| **Paper** | Plugins | ✅ |
| **Purpur** | Plugins | ✅ |
| **Spigot** | Plugins | ✅ |
| **Pufferfish** | Plugins | ✅ |
| **Leaf** | Plugins | ✅ |
| **Fabric** | Mods | ✅ Quando há confirmação de compatibilidade exata |
| **NeoForge** | Mods | ✅ Quando há confirmação de compatibilidade exata |
| **Forge** | Mods | Somente Java |
| **Quilt** | Mods | Somente Java |
| **Vanilla Java** | Sem plugins nem mods | Somente Java |
| **Bedrock nativo** | Servidor Bedrock nativo | Clientes Bedrock nativos |

Escolha **Configuração simples** para criar seu primeiro servidor passo a passo ou **Configuração avançada** para ter mais controle sobre memória, portas, armazenamento, backups, política de versões e jogo entre edições compatíveis.

O JustVoxel verifica quais recursos o software de servidor escolhido realmente oferece.

[Veja os detalhes sobre softwares de servidor e compatibilidade](../docs/server-software.md)

## 🧩 Plugins e mods

Abra **Minecraft → Plugins e mods** para o servidor selecionado.

Dependendo do tipo de servidor, o JustVoxel pode usar:

- [Modrinth](https://modrinth.com/)
- [Hangar](https://hangar.papermc.io/)
- [CurseForge](https://www.curseforge.com/minecraft) com sua chave de API configurada

Nas instalações compatíveis, o JustVoxel verifica a compatibilidade e os downloads, faz backups com o servidor parado, verifica a reinicialização, registra quais itens gerencia e oferece mecanismos de recuperação.

Servidores da família Paper podem usar catálogos de plugins. Fabric, Forge, NeoForge e Quilt usam catálogos de mods com verificação exata do carregador e da versão do Minecraft. Por decisão de projeto, o Vanilla Java e o Bedrock nativo não oferecem catálogo de plugins ou mods.

[Saiba mais sobre plugins e mods](../docs/plugin-workspace.md)

## 🌐 Rede Minecraft opcional

É perfeitamente normal manter vários servidores independentes. Eles **não** precisam de um proxy.

Se você quiser uma única entrada pública para vários servidores compatíveis, o JustVoxel também oferece uma **Rede Minecraft** opcional baseada no Velocity.

Na Central de Controle, você pode selecionar os servidores de destino compatíveis, configurar as portas públicas de Java e Bedrock, gerenciar o lobby virtual e a mensagem de boas-vindas e escolher quais servidores participam da rede.

Paper, Purpur, Pufferfish e Leaf podem atuar como servidores de destino na rede. Servidores Fabric/Quilt e Forge/NeoForge também podem participar quando o JustVoxel confirma a compatibilidade necessária com o proxy. Spigot, Vanilla e Bedrock nativo continuam independentes.

[Saiba mais sobre a Rede Minecraft](../docs/minecraft-network.md)

## 🖥️ Conheça a Central de Controle

O JustVoxel organiza os controles em torno das tarefas do servidor, sem exigir que você conheça o funcionamento interno do Linux.

### Servidores de Minecraft

Crie e gerencie vários servidores, inicie ou pare cada um, consulte o status e a lista de jogadores, ajuste as configurações disponíveis e acesse as ferramentas específicas de cada servidor.

### Plugins e mods

Encontre conteúdo compatível, confira os plugins e mods instalados, gerencie os itens adicionais compatíveis e consulte as informações de recuperação quando uma alteração exigir atenção.

### Backups e armazenamento

Crie backups, consulte o histórico, restaure todos os dados de um servidor, configure destinos de backup, importe ou exporte servidores e gerencie o armazenamento compatível.

### Rede

Gerencie Ethernet, Wi-Fi, **diagnóstico de problemas de conexão** e acesso remoto. Alterações de rede que podem interromper a conexão contam com proteção para reverter a mudança, evitando que uma configuração incorreta de IP ou Wi-Fi bloqueie seu acesso de vez.

### Sistema

Consulte a integridade do sistema, os recursos, os logs, as atualizações do sistema operacional, os usuários, as opções de segurança, as ações de energia disponíveis e os controles de hardware exclusivos da edição HWS.

### Conexões de IA e API

Conecte aplicativos de IA compatíveis, clientes MCP, automações REST, Telegram e Discord sem precisar montar outra estrutura de gerenciamento.

Prefere o terminal? O JustVoxel também inclui o **mjust**, uma interface de administração interativa no terminal que segue o mesmo modelo de segurança do sistema.

[Documentação da WebUI](../docs/WEBUI.md) · [Guia do mjust](../docs/MJUST.md)

## 🌐 Escolha como seus amigos vão se conectar

O JustVoxel integra quatro opções de acesso remoto em **Rede → Acesso remoto**.

Além de incluir os pacotes na imagem, o JustVoxel permite consultar o status, ativar, desativar e configurar a integração conforme os recursos de cada provedor.

| Opção | Indicado para | O que oferece |
| --- | --- | --- |
| **[Tailscale](https://tailscale.com/)** | Servidores privados para amigos e familiares | Rede mesh privada entre dispositivos autorizados |
| **[NetBird](https://netbird.io/)** | Redes privadas com acesso gerenciado | Rede privada com controle de contas e políticas de acesso |
| **[Playit.gg](https://playit.gg/)** | Acesso público ao Minecraft sem o encaminhamento de portas convencional no roteador | Túneis públicos para jogos e endereços de conexão |
| **[NordVPN Meshnet](https://meshnet.nordvpn.com/)** | Acesso privado entre dispositivos de usuários da NordVPN | Conectividade Meshnet sem usar o JustVoxel como cliente de VPN comercial |

### Qual opção devo escolher?

**🎮 Quer que seus amigos entrem sem instalar um cliente de VPN privada?**  
Comece pelo **Playit.gg**. Os jogadores podem usar o endereço público do túnel que você configurar para o jogo.

**🔒 Quer um servidor privado para dispositivos de confiança?**  
Considere o **Tailscale** ou o **NetBird**.

**🌍 Já usa o NordVPN Meshnet?**  
O JustVoxel pode gerenciar a integração com o Meshnet. Isso é independente de conectar a máquina a um servidor de saída convencional da VPN comercial da NordVPN.

As contas, as políticas na nuvem e a configuração dos túneis continuam a cargo do provedor. O JustVoxel cuida da parte da integração que ele oferece na máquina.

É possível usar vários serviços de VPN ou rede mesh ao mesmo tempo, mas pode haver conflitos de roteamento, DNS ou regras de firewall.

[Leia o guia de rede e acesso remoto](../docs/NETWORKING.md)

## 🤖 IA, MCP, REST e automação

Quer perguntar a um assistente de IA como está o servidor, conectar sua automação residencial ou criar sua própria rotina de controle?

O JustVoxel oferece **interfaces MCP e REST** integradas em **Conexões de IA e API**.

Na configuração, você pode escolher aplicativos e tipos de conexão compatíveis, como:

- **OpenAI — ChatGPT e Codex**
- **Claude**
- **Grok**
- **Gemini**
- **GitHub Copilot**
- **Outro aplicativo de IA / MCP**
- **Automação / API REST**

Você decide o que cada conexão pode acessar.

As permissões vão do **Nível 1 — Somente leitura** ao gerenciamento de servidores e, apenas com aprovação explícita, ao controle avançado do sistema. Você também pode restringir cada conexão a determinados servidores de Minecraft.

O JustVoxel dispensa uma chave de API de modelo de IA para disponibilizar suas interfaces MCP e REST. A compatibilidade do cliente depende do aplicativo, dos recursos da conta ou assinatura e do acesso à rede.

O gateway de IA/API vem desativado e só entra em funcionamento quando você o ativa.

[Saiba mais sobre conexões de IA e API](../docs/AI-API-CONNECTIONS.md)

## 💬 Telegram e Discord

O JustVoxel oferece integrações nativas e opcionais com o Telegram e o Discord. Elas usam o mesmo serviço central de gerenciamento da WebUI.

### Telegram

Um bot dedicado do Telegram pode oferecer:

- status do sistema e do Minecraft
- seleção de servidor e informações sobre jogadores
- comandos para iniciar, parar e reiniciar servidores, disponíveis a usuários autorizados a gerenciá-los
- backups manuais
- consulta a atualizações do Minecraft e dos contêineres
- informações sobre atualizações do sistema operacional
- procedimentos controlados para reiniciar a máquina
- notificações opcionais sobre o estado do servidor, os resultados de backups e a disponibilidade de atualizações

O Telegram usa permissões próprias de **Somente leitura** e **Gerenciamento de servidores** e dispensa um modelo de IA.

[Configuração e permissões do Telegram](../docs/TELEGRAM.md)

### Discord

O Discord pode servir tanto como ponto de contato da comunidade de Minecraft quanto como interface de administração.

No servidor do Discord vinculado explicitamente ao JustVoxel, os membros da comunidade podem consultar o status básico do servidor e solicitar acesso ao Minecraft sem receber direitos de administração automaticamente.

Os administradores autorizados do JustVoxel no Discord podem iniciar, parar e reiniciar servidores, fazer backups manuais, consultar atualizações, usar os controles da Rede Minecraft e executar outras ações avançadas disponíveis.

Participar do servidor do Discord, ter o nome na lista de jogadores autorizados do Minecraft e administrar o JustVoxel são permissões independentes.

[Configuração do Discord, acesso da comunidade e administração](../docs/DISCORD.md)

## 💾 Backups, restauração, armazenamento e migração

Os dados do Minecraft são mais importantes que a imagem do sistema.

O JustVoxel mantém os dados do Minecraft e os backups fora da instalação imutável do sistema operacional. Você pode escolher separadamente onde armazenar cada um.

As opções de armazenamento incluem:

- sistema de arquivos do sistema operacional
- discos internos dedicados
- armazenamento USB
- partições ou sistemas de arquivos locais existentes
- novas partições criadas em espaço que já estava não alocado
- NFS
- SMB/CIFS

Cada servidor pode usar backups automáticos e manuais. A restauração substitui **todos os dados persistentes** do servidor selecionado e preserva sua identidade no JustVoxel e sua configuração no sistema.

Para transferir servidores entre máquinas ou recuperá-los após uma reinstalação, o JustVoxel também oferece procedimentos revisados de **exportação, importação e recuperação**.

[Armazenamento](../docs/STORAGE.md) · [Restauração](../docs/RESTORE.md) · [Migração](../docs/MIGRATION.md)

## 🛡️ Mais segurança ao fazer alterações

O JustVoxel vai além de uma interface gráfica que executa comandos arbitrários no shell.

As operações disponíveis seguem regras de segurança do próprio sistema, como:

- verificar se há jogadores conectados antes de ações que possam interromper o jogo
- revisar as alterações antes de aplicá-las
- fazer backups com o servidor parado quando for preciso garantir a recuperação após uma mudança
- identificar cada servidor com precisão para impedir que ele seja confundido com outro
- verificar a reinicialização e o funcionamento do servidor
- manter registros persistentes do estado das operações e da recuperação
- oferecer formas de reverter operações compatíveis
- criar pontos de restauração do NetworkManager para mudanças de rede arriscadas
- separar os processos de atualização do Minecraft e do sistema operacional

A WebUI acessada pelo navegador funciona sem privilégios elevados. As alterações que exigem privilégios passam pelo agente de gerenciamento local do JustVoxel e por sua API explícita, sem expor um shell de root ao navegador.

## 🔄 Atualizações do sistema separadas das atualizações do Minecraft

O JustVoxel é um sistema dedicado gerenciado por imagens, baseado no Home Server Base 10 / AlmaLinux 10.

As atualizações do sistema operacional são independentes das versões do Minecraft e das atualizações de software de servidor, plugins, mods e contêineres.

Assim, atualizar o sistema não altera silenciosamente sua versão do Minecraft, e modificar um servidor de Minecraft não exige reconstruir o sistema operacional da máquina.

[Documentação de gerenciamento do sistema](../docs/SYSTEM.md)

## 📊 Como o JustVoxel se compara a outros projetos?

O JustVoxel é um tipo de projeto diferente de um painel de controle tradicional de Minecraft. O **sistema operacional e a camada de gerenciamento do Minecraft formam um único sistema dedicado**.

| Recurso | **JustVoxel** | **MineOS SvelteKit** | **Crafty Controller** | **MineOS Node** |
| --- | :---: | :---: | :---: | :---: |
| Sistema operacional completo dedicado ao Minecraft | ⭐ | — | — | — |
| Vários servidores de Minecraft | ✅ | ✅ | ✅ | ✅ |
| Suporte a servidores Java e Bedrock | ✅ | ✅ | ✅ | 🟡 Java e suporte legado ao PocketMine |
| Busca guiada de plugins e mods | ✅ Modrinth, Hangar, CurseForge | ✅ Modrinth, CurseForge | — | — |
| Gerenciamento de rede, armazenamento e sistema operacional da máquina | ⭐ | — | — | — |
| Acesso privado e túneis para jogos integrados | ⭐ 4 opções | — | — | — |
| Gateway MCP + REST para IA e automação | ⭐ | — | — | — |
| Gerenciamento nativo via Telegram e Discord | ⭐ | — | — | — |
| Controles de tampa e tela de notebooks | ✅ HWS | — | — | — |
| Atualizações por imagem do sistema operacional imutável | ✅ | — | — | — |
| Interface de gerenciamento pelo terminal | ✅ mjust | ✅ CLI/TUI do mineos | — | ✅ CLI |
| Modelo de instalação | Sistema dedicado bootc completo | Aplicativo Docker / Compose | Painel de controle no Docker, Linux ou Windows | Aplicativo Node.js no Linux/BSD |

⭐ destaque &nbsp;•&nbsp; ✅ compatível &nbsp;•&nbsp; 🟡 parcial/legado &nbsp;•&nbsp; — não consta na documentação principal do projeto consultada para a comparação

**Última revisão da comparação: 16 de novembro de 2026.** A intenção não é classificar os outros projetos como “ruins”. O MineOS e o Crafty são aplicativos de gerenciamento de Minecraft que podem ser excelentes escolhas para quem já tem um sistema operacional e quer administrá-lo por conta própria. O JustVoxel atende a quem quer gerenciar o painel de Minecraft **e o sistema dedicado que o sustenta** como um único produto.

A comparação se baseia na documentação atual e nos repositórios dos projetos:

- [MineOS SvelteKit](https://github.com/freeman412/mineos-sveltekit)
- [Crafty Controller](https://craftycontrol.com/) · [código-fonte](https://gitlab.com/crafty-controller/crafty-4)
- [MineOS Node](https://github.com/hexparrot/mineos-node)

Se algum item da comparação estiver desatualizado ou for injusto, abra uma issue. Correções que deixem o JustVoxel em desvantagem também são bem-vindas.

## 📚 Documentação

Os guias detalhados do sistema estão no [diretório de documentação](../docs/).

Por onde começar:

- [WebUI](../docs/WEBUI.md)
- [Ambiente de execução do Minecraft](../docs/MINECRAFT_RUNTIME.md)
- [Softwares de servidor](../docs/server-software.md)
- [Plugins e mods](../docs/plugin-workspace.md)
- [Rede Minecraft](../docs/minecraft-network.md)
- [Rede e acesso remoto](../docs/NETWORKING.md)
- [Conexões de IA e API](../docs/AI-API-CONNECTIONS.md)
- [Telegram](../docs/TELEGRAM.md)
- [Discord](../docs/DISCORD.md)
- [Armazenamento](../docs/STORAGE.md)
- [Restauração](../docs/RESTORE.md)
- [Migração](../docs/MIGRATION.md)
- [Gerenciamento do sistema](../docs/SYSTEM.md)
- [mjust](../docs/MJUST.md)

Para informações sobre desenvolvimento de imagens, testes e lançamentos, consulte o [guia de desenvolvimento](../README-DEV.md).

## ⛏️ Software do Minecraft e EULA

O JustVoxel **não** distribui binários de servidor do Minecraft, software de servidor da Mojang, mundos pré-criados nem uma cópia do EULA (Contrato de Licença de Usuário Final) do Minecraft com aceite prévio.

Durante a criação do servidor, o administrador aceita o EULA do Minecraft e o JustVoxel prepara o ambiente de execução do servidor Minecraft compatível escolhido.

Minecraft, os softwares da Mojang, Paper, Purpur, Spigot, Pufferfish, Leaf, Fabric, NeoForge, Forge, Quilt, plugins, mods e outros componentes de terceiros mantêm suas próprias licenças e condições de distribuição.

## 📄 Licença

O JustVoxel é licenciado sob a **Apache-2.0**. Consulte [LICENSE](../LICENSE).

Os avisos de terceiros estão em [THIRD_PARTY_NOTICES.md](../THIRD_PARTY_NOTICES.md).
