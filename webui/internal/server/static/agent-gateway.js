(() => {
  'use strict';
  const dialog = document.querySelector('[data-agent-gateway-dialog]');
  const launcher = document.querySelector('[data-agent-gateway-open]');
  if (!dialog || !launcher) return;
  const find = selector => dialog.querySelector(selector);
  const content = find('[data-gateway-content]');
  const feedback = find('[data-gateway-feedback]');
  const cards = find('[data-gateway-cards]');
  const details = find('[data-gateway-details]');
  const gatewaySwitch = find('[data-gateway-enabled]');
  const tokenDialog = document.querySelector('[data-gateway-token-dialog]');
  const tokenNode = tokenDialog.querySelector('[data-gateway-token]');
  const levelDialog = document.querySelector('[data-gateway-level4-dialog]');
  let snapshot = {enabled: false, connections: []};
  let instances = [];
  let selectedID = '';
  let surface = 'overview';
  let discordOrigin = 'add';
  let discordState = {configured: false};
  let telegramOrigin = 'add';
  let telegramState = {configured: false};
  const showSurface = name => {
    surface = name;
    find('[data-gateway-overview]').hidden = name !== 'overview';
    details.hidden = name !== 'settings';
    find('[data-gateway-help]').hidden = name !== 'help';
    find('[data-gateway-telegram]').hidden = name !== 'telegram';
    find('[data-gateway-discord]').hidden = name !== 'discord';
    feedback.hidden = name === 'telegram' || name === 'discord';
    find('[data-gateway-connections]').setAttribute('aria-selected', String(name === 'overview' || (name === 'telegram' && telegramOrigin === 'connections') || (name === 'discord' && discordOrigin === 'connections') || (name === 'settings' && Boolean(selectedID))));
    find('[data-gateway-help-open]').setAttribute('aria-selected', String(name === 'help'));
    find('[data-gateway-add]').setAttribute('aria-selected', String((name === 'settings' && !selectedID) || (name === 'telegram' && telegramOrigin === 'add') || (name === 'discord' && discordOrigin === 'add')));
    content.scrollTop = 0;
    dialog.dispatchEvent(new CustomEvent('gateway-surface-change', {detail: name}));
  };
  const showOverview = () => { if (busy) return; selectedID = ''; details.replaceChildren(); showSurface('overview'); };
  const levelLabel = level => ({1: jvT("ui.b75cf05d6d2a"), 2: jvT("ui.29cc4eb6673b"), 3: jvT("ui.42b03421cde7"), 4: jvT("ui.3d275ef75669")}[level] || jvT("ui.8347689b46f2"));
  const providers = new Map([
    ['OpenAI', {icon: 'openai', label: jvT("ui.3f182492f5bd")}],
    ['Claude', {icon: 'claude', label: 'Claude'}],
    ['Grok', {icon: 'grok', label: 'Grok'}],
    ['Gemini', {icon: 'googlegemini', label: 'Gemini'}],
    ["GitHub Copilot", {icon: 'githubcopilot', label: jvT("ui.ed249d417dcd")}],
    ['OpenClaw', {icon: 'openclaw', label: 'OpenClaw'}],
    ["Other AI / MCP app", {label: jvT("ui.7658e93fa9fd")}],
    ["Automation / REST API", {label: jvT("ui.1e94ea6baaaf")}],
  ]);
  const providerAuthMode = purpose => providers.get(purpose)?.icon ? 'oauth_local' : 'token';
  const providerCompatibility = new Map([
    ['OpenAI', jvT("ui.5d9dbab55845")],
    ['Claude', jvT("ui.b7d115720c81")],
    ['Grok', jvT("ui.3c9d80d152e4")],
    ['Gemini', jvT("ui.afa1380f26a9")],
    ["GitHub Copilot", jvT("ui.a1b9e51763a8")],
    ['OpenClaw', jvT("ui.a065a025921b")],
    ["Other AI / MCP app", jvT("ui.14342c819449")],
    ["Automation / REST API", jvT("ui.d7ff96ebe204")],
  ]);
  const legacyPurposes = new Map([["AI Agent", jvT("ui.ab6eb4ac63b5")], ["API Client", jvT("ui.50b8f3892055")], ["Other", jvT("ui.f97e9da0e3b8")]]);
  const purposeLabel = purpose => providers.get(purpose)?.label || legacyPurposes.get(purpose) || purpose || jvT("ui.7658e93fa9fd");
  const providerIcon = purpose => {
    const icon = document.createElement('img');
    const provider = providers.get(purpose);
    icon.className = 'gateway-provider-icon';
    icon.alt = ''; // The adjacent provider name supplies the accessible identity.
    icon.src = provider?.icon ? `/static/ai-providers/${provider.icon}.svg` : '/static/justvoxel-logo.png';
    return icon;
  };
  const activityLabel = value => value ? new Date(value).toLocaleString() : jvT("ui.0a12b92ce789");
  const scopeLabel = connection => connection.all_instances ? jvT("ui.3205261a6537") : (connection.instance_ids || []).map(id => {
    const instance = instances.find(item => item.id === id);
    return instance ? (instance.display_name || instance.name || jvT("ui.fc80df490dea")) : jvT("ui.2a17a7baf7f8");
  }).join(', ') || jvT("ui.8da895ce9e50");
  let pendingLevel4 = null;
  let busy = false;
  let generation = 0;
  const primary = () => document.body.classList.contains('primary-administrator');
  const message = text => { feedback.textContent = text; };
  const text = (tag, value) => { const element = document.createElement(tag); element.textContent = value; return element; };
  const send = async payload => {
    if (!primary()) throw new Error(jvT("ui.220fabe28e2a"));
    const options = {credentials: 'same-origin', cache: 'no-store', headers: {Accept: 'application/json'}};
    if (payload) {
      options.method = "POST";
      options.body = new URLSearchParams({csrf: content.dataset.gatewayCsrf, request: JSON.stringify(payload)});
    }
    const response = await fetch('/api/agent-gateway', options);
    if (!response.ok || response.redirected) throw new Error(jvT("ui.8e688688533a"));
    return response.json();
  };
  const clearToken = () => { tokenNode.textContent = ''; tokenDialog.querySelector('[data-gateway-copy-status]').textContent = ''; };
  tokenDialog.addEventListener('close', clearToken);
  tokenDialog.addEventListener('cancel', clearToken);
  window.addEventListener('pagehide', clearToken);
  tokenDialog.querySelector('[data-gateway-token-done]').addEventListener('click', () => tokenDialog.close());
  tokenDialog.querySelector('[data-gateway-token-copy]').addEventListener('click', async () => {
    try { await navigator.clipboard.writeText(tokenNode.textContent); tokenDialog.querySelector('[data-gateway-copy-status]').textContent = jvT("ui.d47682fb2296"); }
    catch (_) { tokenDialog.querySelector('[data-gateway-copy-status]').textContent = jvT("ui.cdc06c2ea312"); }
  });
  const showTokenOnce = result => {
    if (result.connection?.auth_mode === 'oauth_local' || !result.token) return;
    clearToken();
    tokenNode.textContent = result.token;
    result.token = ''; // Never retain a credential in the connection registry or browser storage.
    tokenDialog.showModal();
  };
  const closeLevel4 = () => { pendingLevel4 = null; };
  levelDialog.addEventListener('close', closeLevel4);
  levelDialog.addEventListener('cancel', closeLevel4);
  levelDialog.querySelector('[data-gateway-level4-cancel]').addEventListener('click', () => levelDialog.close());
  initDestructiveConfirmations(levelDialog);
  const levelSlider = levelDialog.querySelector('[data-destructive-slider]');
  const levelToggle = levelDialog.querySelector('[data-destructive-toggle]');
  const requestLevel4 = payload => {
    pendingLevel4 = payload;
    levelDialog.querySelector('[data-gateway-level4-name]').textContent = payload.connection.name;
    levelSlider.value = '0'; levelToggle.checked = false;
    levelDialog.querySelector('[data-destructive-confirmation]')._justVoxelDestructiveSync();
    levelDialog.showModal();
  };
  const load = async () => {
    if (!primary()) return;
    const ticket = ++generation;
    message(jvT("ui.8aa3f4352701"));
    try {
      const next = await send();
      // Use the established inventory presentation endpoint; only immutable IDs
      // are submitted as privileged selectors.
      const response = await fetch('/api/minecraft/workspace/selection?selection=none', {credentials: 'same-origin', cache: 'no-store'});
      if (!response.ok || response.redirected) throw new Error(jvT("ui.5f9a2524ea77"));
      const inventory = await response.json();
      if (ticket !== generation) return;
      snapshot = next; instances = inventory.servers || [];
      showGatewayState(snapshot.enabled);
      const selected = snapshot.connections.find(c => c.id === selectedID);
      if (surface === 'settings' && selected) renderSettings(selected);
      else if (surface === 'settings' && selectedID) { selectedID = ''; details.replaceChildren(); showSurface('overview'); }
      message('');
    } catch (error) { message(error.message); }
  };
  const mutate = async (payload, onFailure) => {
    if (busy) return;
    busy = true;
    const controls = [...dialog.querySelectorAll('button, input, select')];
    const disabled = controls.map(control => control.disabled);
    controls.forEach(control => { control.disabled = true; });
    try {
      const result = await send(payload);
      if (result.connection?.id) { selectedID = result.connection.id; showSurface('settings'); }
      if (result.connection?.auth_mode !== 'oauth_local') showTokenOnce(result);
      await load();
    } catch (error) { if (onFailure) onFailure(); message(error.message); }
    finally { busy = false; controls.forEach((control, index) => { control.disabled = disabled[index]; }); }
  };
  levelDialog.querySelector('form').addEventListener('submit', event => {
    event.preventDefault();
    if (busy || !pendingLevel4 || Number(levelSlider.value) !== 100 || !levelToggle.checked || !primary()) return;
    const payload = {...pendingLevel4, acknowledged: true, slider: 100};
    levelDialog.close();
    mutate(payload);
  });
  const showGatewayState = enabled => {
    gatewaySwitch.checked = enabled;
    find('[data-gateway-state]').textContent = jvT("ui.c88bd6a57a8e", {v0: (enabled ? jvT("ui.92c1cdfdf4cb") : jvT("ui.75081b593d15"))});
    renderCards(enabled);
  };
  gatewaySwitch.addEventListener('change', () => {
    if (busy) return;
    const enabled = gatewaySwitch.checked;
    const previous = snapshot.enabled;
    showGatewayState(enabled);
    mutate({action: 'enable', enabled}, () => showGatewayState(previous));
  });
  const renderCards = (gatewayEnabled = snapshot.enabled) => {
    cards.replaceChildren();
    if (!snapshot.connections.length && !telegramState.configured && !discordState.configured) cards.append(text('p', jvT("ui.e9136943e7ea")));
    for (const connection of snapshot.connections) {
      const active = gatewayEnabled && connection.enabled;
      const card = text('article', ''); card.className = `panel minecraft-server-card minecraft-server-health-${active ? 'success' : 'neutral'}`; card.dataset.gatewayConnection = connection.id;
      const heading = text('div', ''); heading.className = 'minecraft-server-heading';
      const state = text('strong', active ? jvT("ui.92c1cdfdf4cb") : jvT("ui.75081b593d15")); state.className = `minecraft-server-state ${active ? 'success' : 'neutral'}`;
      heading.append(text('h2', connection.name), state); card.append(heading);
      const identity = text('div', ''); identity.className = 'minecraft-server-software-row';
      const provider = text('p', ''); provider.className = 'muted';
      provider.append(providerIcon(connection.purpose), text('span', purposeLabel(connection.purpose)));
      const actions = text('details', ''); actions.className = 'storage-action-menu minecraft-server-more';
      const button = text('summary', '⋯'); button.setAttribute('aria-label', jvT("ui.34ae6f7b58c5", {v0: (connection.name)}));
      button.addEventListener('click', event => {
        event.preventDefault();
        if (busy) return;
        selectedID = connection.id; renderSettings(connection);
      });
      actions.append(button); identity.append(provider, actions); card.append(identity);
      const metrics = text('div', ''); metrics.className = 'minecraft-server-metrics';
      const auth = connection.auth_mode === 'oauth' || connection.auth_mode === 'oauth_local' ? 'OAuth' : jvT("ui.d2089be67295");
      for (const [label, value] of [[jvT("ui.ec5ba0abb717"), levelLabel(connection.level)], [jvT("ui.b073f6c68ef8"), scopeLabel(connection)], [jvT("ui.8eb3ea9bbde6"), auth]]) {
        const metric = text('div', ''); metric.append(text('span', label), text('strong', value)); metrics.append(metric);
      }
      card.append(metrics);
      const activity = text('div', ''); activity.className = 'gateway-card-activity';
      activity.append(text('span', jvT("ui.06475633ed3e")), text('p', activityLabel(connection.last_activity))); card.append(activity);
      cards.append(card);
    }
    if (telegramState.configured) {
      const statuses = {
        connected: [jvT("ui.22965568d22a"), 'good'], verified: [jvT("ui.f724fb6f802d"), 'good'],
        disabled: [jvT("ui.75081b593d15"), 'muted'], unverified: [jvT("ui.3dddf7d417a7"), 'muted'],
        verification_failed: [jvT("ui.f0f2383f8af8"), 'danger'], connecting: [jvT("ui.d403c686f6a1"), 'muted'],
        connection_retry: [jvT("ui.f8fe6f863109"), 'danger'], connection_paused: [jvT("ui.c1ebc7817870"), 'danger'],
        disconnected: [jvT("ui.04dfac3671b4"), 'muted'],
      };
      const [label, tone] = Object.hasOwn(statuses, telegramState.status) ? statuses[telegramState.status] : [jvT("ui.7eb5af92e49c"), 'muted'];
      const card = text('article', ''); card.className = 'panel minecraft-server-card';
      card.setAttribute('data-telegram-integration', '');
      card.setAttribute('role', 'button'); card.setAttribute('tabindex', '0');
      card.setAttribute('aria-label', jvT("ui.e643538cd7f0", {v0: (label)}));
      const heading = text('div', ''); heading.className = 'minecraft-server-heading';
      const identity = text('div', ''); identity.className = 'gateway-connection-identity';
      identity.append(telegramHelpIcon(), text('h2', 'Telegram'));
      const badge = text('span', label); badge.className = `badge telegram-integration-badge ${tone}`;
      heading.append(identity, badge); card.append(heading);
      if (telegramState.bot_name) card.append(text('p', `@${telegramState.bot_name}`));
      const open = () => { if (busy) return; telegramOrigin = 'connections'; showSurface('telegram'); find('[data-telegram-replace]').focus(); };
      card.addEventListener('click', open);
      card.addEventListener('keydown', event => { if (event.key === 'Enter' || event.key === ' ') { event.preventDefault(); open(); } });
      cards.append(card);
    }
    if (discordState.configured) {
      const statuses = {
        connected: [jvT("ui.22965568d22a"), 'good'], verified: [jvT("ui.f724fb6f802d"), 'good'],
        disabled: [jvT("ui.75081b593d15"), 'muted'], unverified: [jvT("ui.3dddf7d417a7"), 'muted'],
        connecting: [jvT("ui.d403c686f6a1"), 'muted'],
        connection_retry: [jvT("ui.f8fe6f863109"), 'danger'], connection_paused: [jvT("ui.c1ebc7817870"), 'danger'],
        disconnected: [jvT("ui.04dfac3671b4"), 'muted'],
      };
      const [label, tone] = Object.hasOwn(statuses, discordState.status) ? statuses[discordState.status] : [jvT("ui.7eb5af92e49c"), 'muted'];
      const card = text('article', ''); card.className = 'panel minecraft-server-card';
      card.setAttribute('data-discord-integration', '');
      card.setAttribute('role', 'button'); card.setAttribute('tabindex', '0');
      card.setAttribute('aria-label', jvT("ui.4dff51bb17fb", {v0: (label)}));
      const heading = text('div', ''); heading.className = 'minecraft-server-heading';
      const identity = text('div', ''); identity.className = 'gateway-connection-identity';
      identity.append(discordHelpIcon(), text('h2', 'Discord'));
      const badge = text('span', label); badge.className = `badge telegram-integration-badge ${tone}`;
      heading.append(identity, badge); card.append(heading);
      if (discordState.bot_username) card.append(text('p', `@${discordState.bot_username}`));
      const open = () => { if (busy) return; discordOrigin = 'connections'; showSurface('discord'); find('[data-discord-replace]').focus(); };
      card.addEventListener('click', open);
      card.addEventListener('keydown', event => { if (event.key === 'Enter' || event.key === ' ') { event.preventDefault(); open(); } });
      cards.append(card);
    }
  };
  // Presentation only: neither provider previews nor Help submit connection changes.
  const copyText = async (value, field, status, button) => {
    const originalLabel = button.gatewayCopyLabel || button.textContent;
    button.gatewayCopyLabel = originalLabel;
    const feedback = copied => {
      clearTimeout(button.gatewayCopyTimer);
      button.textContent = copied ? jvT("ui.8d525e5f158b") : jvT("ui.57fd7a0cf33f");
      button.title = copied ? jvT("ui.8d525e5f158b") : jvT("ui.6c46b62dc3f2");
      status.textContent = button.title;
      button.gatewayCopyTimer = setTimeout(() => {
        button.textContent = originalLabel;
        button.title = '';
        status.textContent = '';
      }, 1800);
    };
    if (navigator.clipboard?.writeText) {
      try {
        await navigator.clipboard.writeText(value);
        feedback(true);
        return;
      } catch (_) { /* LAN HTTP and browser permissions can prevent clipboard access. */ }
    }
    const temporary = document.createElement('textarea');
    temporary.value = value;
    temporary.readOnly = true;
    temporary.className = 'gateway-copy-buffer';
    // Keep the fallback inside the active dialog so modal focus stays usable.
    field.parentNode.append(temporary);
    let copied = false;
    try {
      temporary.focus();
      temporary.select();
      copied = Boolean(document.execCommand && document.execCommand('copy'));
    } catch (_) { /* Leave the original field selected for keyboard copying below. */ }
    finally { temporary.remove(); }
    if (copied) { feedback(true); return; }
    field.focus();
    field.select();
    field.setSelectionRange(0, value.length);
    feedback(false);
  };
  const settingsDate = value => {
    if (!value) return jvT("ui.ca1844969742");
    const date = new Date(value);
    return Number.isNaN(date.getTime()) ? value : date.toLocaleString(undefined, {
      year: 'numeric', month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit',
    });
  };
  const localAddress = address => {
    const url = new URL(address);
    const host = url.hostname.toLowerCase().replace(/^\[|\]$/g, '');
    const octets = host.split('.').map(Number);
    return host === 'localhost' || host.endsWith('.localhost') || host === '::1' ||
      host.includes(':') && (/^f[cd][0-9a-f]{2}:/.test(host) || /^fe[89ab][0-9a-f]:/.test(host)) ||
      (octets.length === 4 && (octets[0] === 127 || octets[0] === 10 ||
        octets[0] === 192 && octets[1] === 168 || octets[0] === 172 && octets[1] >= 16 && octets[1] <= 31));
  };
  const secureAddress = address => new URL(address).protocol === 'https:';
  // Modes affect instructions only; connection purpose and authentication stay unchanged.
  const clientModes = {
    OpenAI: {'ChatGPT': 'cloud', 'Codex': 'local'},
    Claude: {'Claude': 'cloud', 'Claude Code': 'local'},
    'GitHub Copilot': {'CLI': 'local', 'VS Code': 'local'},
    OpenClaw: {'Control UI': 'local', 'CLI': 'local'},
    Grok: {'Grok': 'cloud'}, Gemini: {'Gemini': 'cloud', 'Gemini CLI': 'local'},
    'Other AI / MCP app': {'Other AI / MCP app': 'client'},
    'Automation / REST API': {'Automation / REST API': 'rest'},
  };
  const renderProviderGuide = (host, purpose, address, options = {}) => {
    const modes = Object.keys(clientModes[purpose] || {});
    const mode = modes.includes(options.mode) ? options.mode : modes[0];
    const reachability = clientModes[purpose]?.[mode];
    const automation = purpose === 'Automation / REST API';
    const hostname = new URL(address).hostname.toLowerCase();
    const octets = hostname.split('.').map(Number);
    const geminiPrivateHost = purpose === 'Gemini' && (/\.(local|internal)$/.test(hostname) ||
      octets.length === 4 && octets[0] === 100 && octets[1] >= 64 && octets[1] <= 127);
    const privateEndpoint = localAddress(address) || geminiPrivateHost;
    const httpsEndpoint = secureAddress(address);
    const definitelyPrivate = privateEndpoint === true;
    const cloudCandidate = privateEndpoint === false && httpsEndpoint === true;
    const geminiCLI = purpose === 'Gemini' && mode === 'Gemini CLI';
    const prerequisite = (reachability === 'cloud' && !cloudCandidate) || (geminiCLI && !httpsEndpoint);
    host.replaceChildren();
    if (!options.settings) {
      const identity = text('div', ''); identity.className = 'gateway-connection-identity';
      const titles = text('div', '');
      const subtitle = {OpenAI: jvT("ui.27f0f7393c73"), Claude: jvT("ui.46bbfcd46443"), Grok: jvT("ui.8d1a3156faa2"), Gemini: jvT("ui.d1f40ecdf296"), 'GitHub Copilot': jvT("ui.a46c86c8dfe5"), OpenClaw: jvT("ui.e6f8e976a215")}[purpose];
      titles.append(text('h3', purpose));
      if (subtitle) titles.append(text('p', subtitle));
      identity.append(providerIcon(purpose), titles); host.append(identity);
    }
    const copyBlock = (value, label, caption) => {
      const row = text('div', ''); row.className = 'gateway-address';
      const copy = text('button', label); copy.type = 'button'; copy.className = 'secondary';
      copy.setAttribute('data-gateway-copy-label', label === 'Copy' ? 'Selected' : label);
      const status = text('p', ''); status.className = 'gateway-visually-hidden'; status.setAttribute('role', 'status');
      const field = document.createElement('input');
      field.type = 'text'; field.readOnly = true; field.value = value;
      field.className = 'gateway-copy-field'; field.setAttribute('aria-label', caption || (label === jvT("ui.9a01feecae67") ? 'Command' : 'Address'));
      field.addEventListener('click', () => field.select());
      copy.addEventListener('click', () => copyText(value, field, status, copy));
      if (caption) row.append(text('span', caption));
      row.append(field, copy); host.append(row, status);
    };
    const addressLabel = automation ? 'REST API address' : jvT("ui.973480a1f339");
    if (!prerequisite) {
      if (options.help && !automation && providers.get(purpose)?.icon) host.append(text('p', jvT("ui.d2bdb3fe418d")));
      else copyBlock(address, jvT("ui.e21f935f11d7"), addressLabel);
    }
    const toolbar = options.settings ? text('div', '') : host;
    if (options.settings) { toolbar.className = 'gateway-guide-toolbar'; host.append(toolbar); }
    toolbar.append(text('h4', jvT("ui.2198ec8ff357")));
    if (modes.length > 1) {
      const tabs = text('div', ''); tabs.className = 'gateway-guide-modes'; tabs.setAttribute('role', 'group'); tabs.setAttribute('aria-label', purpose + ' app');
      for (const choice of modes) {
        const button = text('button', choice); button.type = 'button'; button.className = 'secondary'; button.setAttribute('aria-pressed', String(choice === mode));
        button.addEventListener('click', () => renderProviderGuide(host, purpose, address, {...options, mode: choice})); tabs.append(button);
      }
      toolbar.append(tabs);
    }
    const steps = values => {
      const list = text('ol', ''); list.className = 'gateway-help-steps';
      for (const value of values) list.append(text('li', value)); host.append(list);
    };
    const inline = value => {
      const line = text('p', value); line.className = 'gateway-inline-note'; host.append(line);
    };
    const link = (label, url, parent = host) => {
      const anchor = text('a', label); anchor.href = url; anchor.target = '_blank'; anchor.rel = 'noopener noreferrer'; parent.append(anchor);
    };
    const details = entries => {
      const line = text('p', '');
      line.append(text('span', jvT("ui.9b98da0a3bc7")));
      entries.forEach(([label, url], index) => {
        if (index) line.append(text('span', jvT("ui.e3ee915a8e8c")));
        link(label, url, line);
      });
      line.append(text('span', '.')); host.append(line);
    };
    const tunnelGuide = 'https://developers.openai.com/api/docs/guides/secure-mcp-tunnels';
    const official = {
      OpenAI: mode === 'Codex' ? 'https://developers.openai.com/learn/docs-mcp' : 'https://help.openai.com/en/articles/12584461-connecting-custom-mcp-servers-in-chatgpt',
      Claude: mode === 'Claude Code' ? 'https://code.claude.com/docs/en/mcp' : 'https://support.claude.com/en/articles/11175166-get-started-with-custom-connectors-using-remote-mcp',
      Grok: 'https://docs.x.ai/grok/connectors', Gemini: geminiCLI ? 'https://github.com/google-gemini/gemini-cli/blob/main/docs/tools/mcp-server.md' : 'https://support.google.com/gemini/answer/17209137',
      'GitHub Copilot': mode === 'VS Code' ? 'https://code.visualstudio.com/docs/agent-customization/mcp-servers' : 'https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/add-mcp-servers',
      OpenClaw: mode === 'CLI' ? 'https://docs.openclaw.ai/cli/mcp/transports' : 'https://docs.openclaw.ai/tools/mcp',
    }[purpose];
    const providerDetails = () => {
      if (purpose === 'OpenAI' && mode === 'ChatGPT') details([[jvT("ui.0f4d4965e00e"), tunnelGuide], [jvT("ui.50b01c1a90ce"), official]]);
      else if (purpose === 'Claude' && mode === 'Claude') details([[jvT("ui.8686d2209b63"), official]]);
      else if (purpose === 'Grok') details([[jvT("ui.1665e9ed18fa"), 'https://docs.x.ai/grok/connectors/custom-mcp-tunneling'], [jvT("ui.8686d2209b63"), official]]);
      else if (purpose === 'Gemini') details(geminiCLI ? [[jvT("ui.a5e3a09cd572"), official]] : [[jvT("ui.8686d2209b63"), official], [jvT("ui.38011af023e6"), 'https://developers.google.com/connected-apps/gpc/prerequisites']]);
    };
    const compactProviderLinks = purpose === 'OpenAI' && mode === 'ChatGPT' || purpose === 'Claude' && mode === 'Claude' || purpose === 'Grok' || purpose === 'Gemini';
    if (official && !prerequisite && !compactProviderLinks) link(jvT("ui.4c52f781cbd4"), official, toolbar);
    if (purpose === 'Claude' && mode === 'Claude' && options.help) {
      inline(jvT("ui.4d29ef581a56"));
    }
    if (purpose === 'Gemini' && options.help) {
      host.append(text('h4', jvT("ui.12f67f8539c4")));
      inline(jvT("ui.aa142e7dbe0a"));
      host.append(text('h4', jvT("ui.66880d2d8216")));
      inline(jvT("ui.8c41d90837b7"));
      if (geminiCLI) {
        host.append(text('h4', jvT("ui.731732835cd7")));
        inline(jvT("ui.6bbd7cca433b"));
        host.append(text('code', 'gemini mcp add --transport http justvoxel <MCP_URL>'), text('code', 'gemini mcp list'));
        inline(jvT("ui.c643925e0906"));
        host.append(text('code', '/mcp auth justvoxel'));
        inline(jvT("ui.3cae2c5ae515"));
        inline(jvT("ui.f99058a7900c"));
        const config = text('pre', ''); config.className = 'gateway-cli-config';
        config.append(text('code', JSON.stringify({mcpServers: {justvoxel: {httpUrl: 'https://<justvoxel-lan-host>/mcp/<connection-id>', oauth: {allowPrivateNetwork: true}}}}, null, 2)));
        host.append(config);
        details([[jvT("ui.ff429b184a80"), 'https://support.google.com/gemini/answer/17209137']]);
      }
    }
    if (options.help) {
      if (purpose === 'OpenAI' && mode === 'ChatGPT') {
        host.append(text('h4', jvT("ui.1cae5f09b2a7")));
        inline(jvT("ui.8da98ebfa7fb"));
        inline(jvT("ui.5fbb403c1897"));
        inline(jvT("ui.47d54dc7e91d"));
        host.append(text('h4', jvT("ui.9de420f9b3f0")));
        inline(jvT("ui.d892b800e7de"));
      } else if (purpose === 'Claude' && mode === 'Claude') {
        inline(jvT("ui.9ae76da02655"));
      } else if (purpose === 'OpenAI' && mode === 'Codex') {
        inline(jvT("ui.72b6516f5b67"));
      } else if (purpose === 'Claude' && mode === 'Claude Code') {
        inline(jvT("ui.769a8a38165c"));
      } else if (purpose === 'Grok') {
        inline(jvT("ui.22023905f6d8"));
      } else if (purpose === 'Gemini' && !geminiCLI) {
        inline(jvT("ui.10d79491fd38"));
      } else if (purpose === 'GitHub Copilot') {
        inline(jvT("ui.e3a2c1d9ce95"));
      } else if (purpose === 'OpenClaw') {
        inline(jvT("ui.3283a0d8a123"));
      }
    }
    if (prerequisite) {
      if (geminiCLI) {
        host.append(text('strong', definitelyPrivate ? jvT("ui.baa2c510fb25") : jvT("ui.920e0b40b908")));
        host.append(text('p', jvT("ui.57eaa790bf77")));
        host.append(text('p', definitelyPrivate ? jvT("ui.c2da5c36980e") : jvT("ui.44d059432f21")));
        if (definitelyPrivate) host.append(text('p', jvT("ui.4a64b7f0c0e8")));
      } else if (purpose === 'OpenAI') {
        host.append(text('strong', jvT("ui.5f39c0c78ca3")));
        host.append(text('p', definitelyPrivate
          ? jvT("ui.e594370ec456", {v0: (new URL(address).hostname)})
          : jvT("ui.ad43e5b285e1")));
        if (!options.help) {
          host.append(text('p', jvT("ui.9e16d2323fb5")));
          host.append(text('p', jvT("ui.c15595ce8165")));
        }
        if (definitelyPrivate) {
          const reference = text('div', ''); reference.className = 'gateway-endpoint-reference';
          reference.append(text('p', jvT("ui.06e5fd1dff3b")), text('code', address));
          host.append(reference);
        }
      } else if (purpose === 'Claude') {
        host.append(text('strong', jvT("ui.0f6f73e442af")));
        host.append(text('p', definitelyPrivate
          ? jvT("ui.753f9218ab90")
          : jvT("ui.811c3776af4a")));
        host.append(text('p', jvT("ui.d7a464b25923")));
      } else if (purpose === 'Grok') {
        host.append(text('strong', jvT("ui.cef453ff6619")));
        host.append(text('p', definitelyPrivate
          ? jvT("ui.c08b843a5d9a")
          : jvT("ui.6d084dec6cb2")));
        host.append(text('p', jvT("ui.fd8f19bdd769")));
      } else if (purpose === 'Gemini') {
        host.append(text('strong', jvT("ui.205c94ebdd0e")));
        host.append(text('p', jvT("ui.ae27db11d5c3")));
      }
      providerDetails();
      return;
    }
    if (geminiCLI && options.help) { providerDetails(); return; }
    const commandAddress = options.help ? '<MCP_URL>' : address;
    if (reachability === 'cloud') inline(jvT("ui.64bd807e67bf", {v0: (purpose === 'OpenAI' ? 'ChatGPT' : purpose === 'Claude' ? 'Anthropic infrastructure' : purpose)}));
    if (purpose === 'OpenAI' && mode === 'ChatGPT') {
      steps([jvT("ui.bbac82f70a21"), jvT("ui.05e7b1465503"), jvT("ui.8163358a2d01"), jvT("ui.5e0fa2869342"), jvT("ui.73559c6fae27"), jvT("ui.730a3af10dba"), jvT("ui.8bdebdac44d1")]);
      inline(jvT("ui.4d7040789ec8"));
      if (options.help) details([[jvT("ui.47e6400594ba"), 'https://developers.openai.com/plugins/deploy/connect-chatgpt']]);
    } else if (purpose === 'OpenAI') {
      copyBlock(`codex mcp add justvoxel --url ${commandAddress}`, jvT("ui.9a01feecae67"));
      host.append(text('code', 'codex mcp list'));
      inline(jvT("ui.88d7a61b8655"));
      if (definitelyPrivate) inline(jvT("ui.9d65cf3dbb3d"));
    } else if (purpose === 'Claude' && mode === 'Claude Code') {
      copyBlock(`claude mcp add --transport http justvoxel ${commandAddress}`, jvT("ui.9a01feecae67"));
      for (const command of ['claude mcp login justvoxel', 'claude mcp get justvoxel', 'claude mcp list']) host.append(text('code', command));
      inline(jvT("ui.56790abea4f9"));
      inline(jvT("ui.b71c1a46d49c"));
    } else if (purpose === 'Claude') {
      steps([jvT("ui.0361d12887d4"), jvT("ui.0be48ada4d63"), jvT("ui.1665228725ed"), jvT("ui.5cecddd740e1"), jvT("ui.77c9d1209ce7"), jvT("ui.cdfd6440459a"), jvT("ui.663ae13654cf"), jvT("ui.9804e20eb2d4")]);
      inline(jvT("ui.fb8db7be2981"));
    } else if (purpose === 'Grok') {
      steps([jvT("ui.cfbf3799fff2"), jvT("ui.f31d41c5e683"), jvT("ui.1b78ee00cdbc"), jvT("ui.4f06da02c5cc")]);
    } else if (geminiCLI) {
      copyBlock(`gemini mcp add --transport http justvoxel ${commandAddress}`, jvT("ui.9a01feecae67"));
      host.append(text('code', 'gemini mcp list'));
      host.append(text('p', jvT("ui.7012c6127cf6")));
      host.append(text('code', '/mcp auth justvoxel'));
      host.append(text('p', jvT("ui.07ebd13c7581")));
      if (definitelyPrivate) {
        host.append(text('p', jvT("ui.ca2397a2ad99")));
      }
      inline(jvT("ui.0618d504a654"));
    } else if (purpose === 'Gemini') {
      steps([jvT("ui.657e24a6a3c2"), jvT("ui.acc10a4930c3"), jvT("ui.d624281d038f"), jvT("ui.1f882f15e00d"), jvT("ui.eb7a04d06f1e"), jvT("ui.0afa94b05e7f")]);
    } else if (purpose === 'GitHub Copilot') {
      if (mode === 'VS Code') {
        steps([jvT("ui.e76d0db0f762"), jvT("ui.c0c62652afba"), jvT("ui.4e39a71c8fab"), jvT("ui.907bc967e9c3"), jvT("ui.e3e67e965f6f"), jvT("ui.edd32adb6c38"), jvT("ui.de8ca5b08c4f"), jvT("ui.dc5d00befc86")]);
        inline(jvT("ui.6cb885cdc7c1"));
        if (options.help) inline(jvT("ui.e950c2733608"));
      } else {
        copyBlock(`copilot mcp add --transport http justvoxel ${commandAddress}`, jvT("ui.9a01feecae67"));
        host.append(text('code', '/mcp auth justvoxel'), text('code', 'copilot mcp list'));
        inline(jvT("ui.d06dcf719ff6"));
      }
      if (definitelyPrivate) inline(jvT("ui.42ee8afce10f", {v0: (mode === 'VS Code' ? 'VS Code' : "Copilot CLI")}));
      inline(jvT("ui.842ecc61fdec"));
    } else if (purpose === 'OpenClaw') {
      if (mode === 'CLI') {
        copyBlock(`openclaw mcp add justvoxel --url ${commandAddress} --transport streamable-http`, jvT("ui.9a01feecae67"));
        const config = JSON.stringify({url: commandAddress, transport: 'streamable-http', auth: 'oauth'}).replace(/'/g, "'\\''");
        inline(jvT("ui.5c8d0e398c84"));
        copyBlock(`openclaw mcp set justvoxel '${config}'`, jvT("ui.9a01feecae67"));
        host.append(text('code', 'openclaw mcp login justvoxel'));
        inline(jvT("ui.17b83c455562"));
      } else {
        steps([jvT("ui.00d61f589221"), jvT("ui.22f394ca9d26"), jvT("ui.6922b2371dc7"), jvT("ui.0a965eeb509f"), jvT("ui.cf8725a33727"), jvT("ui.049cd9917979"), jvT("ui.f428fd9ee935"), jvT("ui.9b40e07142b1")]);
        if (options.help) inline(jvT("ui.d22735ff6812"));
      }
      host.append(text('code', 'openclaw mcp doctor justvoxel --probe'), text('code', 'openclaw mcp status --verbose'));
      if (definitelyPrivate) inline(jvT("ui.01ad6a584216"));
    } else if (automation) {
      steps([jvT("ui.e56b3bed7a9a"), jvT("ui.e298d6354244"), jvT("ui.feb357d1f7c0"), jvT("ui.24d248e94a3f")]);
    } else {
      inline(jvT("ui.cd50e192ff3b"));
      steps([jvT("ui.c4c82d168f27"), jvT("ui.6c0f90a23960"), jvT("ui.4a7744e45640"), jvT("ui.3bd948eee224")]);
      inline(jvT("ui.3e9c731a1ead"));
    }
    providerDetails();
  };
  // Both form modes keep individual draft selections while All servers is on.
  const renderMinecraftScope = (form, c) => {
    const field = name => form.elements.namedItem(name);
    const allServers = form.querySelector('[data-gateway-all-servers]');
    allServers.checked = Boolean(c.all_instances);
    const instanceOptions = form.querySelector('[data-gateway-instance-options]');
    const known = new Map(instances.map(instance => [instance.id, instance.display_name || instance.name || jvT("ui.fc80df490dea")]));
    for (const id of c.instance_ids || []) if (!known.has(id)) known.set(id, jvT("ui.2a17a7baf7f8"));
    for (const [id, name] of known) {
      const label = text('label', ''); label.className = 'gateway-server-item';
      const checkbox = document.createElement('input'); checkbox.type = 'checkbox'; checkbox.name = 'instance_ids'; checkbox.value = id; checkbox.checked = (c.instance_ids || []).includes(id);
      checkbox.className = 'date-time-switch'; checkbox.setAttribute('role', 'switch');
      const icon = document.createElement('img'); icon.className = 'minecraft-server-software-icon'; icon.alt = '';
      icon.src = instances.find(instance => instance.id === id)?.software_icon || '/static/justvoxel-logo.png';
      label.append(checkbox, icon, text('span', name));
      instanceOptions.append(label);
    }
    if (!known.size) instanceOptions.append(text('p', jvT("ui.5a86fd3107d4")));
    const sync = () => {
      const all = allServers.checked;
      field('server_scope').value = all ? 'all' : 'selected';
      form.querySelector('[data-gateway-scope]').hidden = false;
      instanceOptions.querySelectorAll('input[name="instance_ids"]').forEach(input => { input.disabled = all; });
    };
    allServers.addEventListener('change', sync); sync();
  };
  const telegramHelpIcon = () => {
    const icon = document.createElement('img');
    icon.className = 'gateway-provider-icon'; icon.alt = '';
    icon.src = '/static/ai-providers/telegram.svg';
    return icon;
  };
  const renderTelegramHelpGuide = host => {
    host.replaceChildren();
    const identity = text('div', ''); identity.className = 'gateway-connection-identity';
    const titles = text('div', ''); titles.append(text('h3', 'Telegram'));
    identity.append(telegramHelpIcon(), titles); host.append(identity);
    host.append(text('h4', jvT("ui.2198ec8ff357")));
    const steps = text('ol', '');
    for (const step of [
      jvT("ui.dc6f20d1b7b1"),
      jvT("ui.fbf2622d98c6"),
      jvT("ui.76a8923ba81b"),
      jvT("ui.06e2146f5b3b"),
      'Copy the exact one-time /start <challenge> command and send it in a private chat with the bot.',
      jvT("ui.330ec92ebd47"),
    ]) steps.append(text('li', step));
    host.append(steps,
      text('p', jvT("ui.7f2ddc531083")),
      text('p', jvT("ui.a6e52c72a38a")),
      text('p', jvT("ui.d0cd89af1978")));
  };
  const discordHelpIcon = () => {
    const icon = document.createElement('img');
    icon.className = 'gateway-provider-icon'; icon.alt = '';
    icon.src = '/static/ai-providers/discord.svg';
    return icon;
  };
  const renderDiscordHelpGuide = host => {
    host.replaceChildren();
    const identity = text('div', ''); identity.className = 'gateway-connection-identity';
    const titles = text('div', ''); titles.append(text('h3', 'Discord'));
    identity.append(discordHelpIcon(), titles); host.append(identity);
    host.append(text('h4', jvT("ui.2198ec8ff357")));
    const steps = text('ol', '');
    const portalStep = text('li', jvT("ui.a43fde646a4d"));
    const portal = text('a', jvT("ui.689ec8e0baf2"));
    portal.href = 'https://discord.com/developers/home';
    portal.target = '_blank';
    portal.rel = 'noopener noreferrer';
    portalStep.append(portal, document.createTextNode('.'));
    for (const step of [
      jvT("ui.19a2d1f17ee4"),
      portalStep,
      jvT("ui.234203edaa69"),
      jvT("ui.a75fafa34d9e"),
      jvT("ui.8e97edcbbc90"),
      jvT("ui.156fe4156207"),
      jvT("ui.2e78c6f1cedb"),
      jvT("ui.01cd932c2b9d"),
    ]) steps.append(typeof step === 'string' ? text('li', step) : step);
    host.append(steps,
      text('p', jvT("ui.a293e37ede76")),
      text('p', jvT("ui.1e61ed689cc2")),
      text('p', jvT("ui.1ac2b9fd2317")),
      text('p', jvT("ui.98a3ee076e13")));
  };
  const resetHelpGuide = () => {
    for (const button of find('[data-gateway-help-providers]').querySelectorAll('button')) button.setAttribute('aria-pressed', 'false');
    find('[data-gateway-help-guide]').replaceChildren();
  };
  const setupHelpGuide = () => {
    const selector = find('[data-gateway-help-providers]');
    const guide = find('[data-gateway-help-guide]');
    const buttons = [];
    const select = purpose => {
      buttons.forEach(([name, button]) => button.setAttribute('aria-pressed', String(name === purpose)));
      if (purpose === 'Telegram') renderTelegramHelpGuide(guide);
      else if (purpose === 'Discord') renderDiscordHelpGuide(guide);
      else renderProviderGuide(guide, purpose, location.origin + (purpose === 'Automation / REST API' ? '/api/v1' : '/mcp'), {help: true});
    };
    for (const purpose of ['Telegram', 'Discord', ...providers.keys()]) {
      const button = text('button', ''); button.type = 'button'; button.className = 'gateway-provider-card';
      button.setAttribute('aria-pressed', 'false');
      button.append(purpose === 'Telegram' ? telegramHelpIcon() : purpose === 'Discord' ? discordHelpIcon() : providerIcon(purpose), text('span', purpose));
      button.addEventListener('click', () => select(purpose)); selector.append(button); buttons.push([purpose, button]);
    }
    resetHelpGuide();
  };
  const renderSettings = connection => {
    const c = structuredClone(connection);
    details.replaceChildren(document.querySelector('[data-gateway-settings-template]').content.cloneNode(true));
    const form = details.querySelector('form');
    const telegramCard = form.querySelector('[data-telegram-card]');
    if (c.id) telegramCard.remove();
    else form.querySelector('[data-telegram-select]').addEventListener('change', () => {
      if (busy) return;
      details.replaceChildren();
      telegramOrigin = 'add';
      showSurface('telegram');
      find(telegramState.configured ? '[data-telegram-replace]' : '[data-telegram-token]').focus();
    });
    const discordCard = form.querySelector('[data-discord-card]');
    if (c.id) discordCard.remove();
    else form.querySelector('[data-discord-select]').addEventListener('change', () => {
      if (busy) return;
      details.replaceChildren();
      discordOrigin = 'add';
      showSurface('discord');
      find(discordState.configured ? '[data-discord-replace]' : '[data-discord-token]').focus();
    });
    if (c.id) {
      form.querySelector('[data-gateway-native-integrations]').remove();
      form.querySelector('[data-gateway-connection-title]').remove();
    }
    const field = name => form.elements.namedItem(name);
    form.className = c.id ? 'gateway-edit-settings' : 'gateway-add-settings';
    for (const name of ['name', 'level']) field(name).value = c[name] ?? '';
    // Keep custom purposes from deployed connections without interpreting them as markup or asset paths.
    if (c.purpose && !providers.has(c.purpose)) {
      const label = text('label', ''); label.className = 'gateway-provider-card';
      const input = document.createElement('input'); input.type = 'radio'; input.name = 'purpose'; input.value = c.purpose;
      label.append(input, providerIcon(c.purpose), text('strong', purposeLabel(c.purpose)));
      form.querySelector('[data-gateway-providers]').append(label);
    }
    field('purpose').value = c.purpose || 'OpenAI';
    const compatibility = form.querySelector('[data-gateway-compatibility-summary]');
    compatibility.hidden = Boolean(c.id);
    if (!c.id) {
      const previewCompatibility = () => {
        compatibility.textContent = providerCompatibility.get(field('purpose').value) || providerCompatibility.get(jvT("ui.7658e93fa9fd"));
      };
      form.querySelectorAll('input[name="purpose"]').forEach(input => input.addEventListener('change', previewCompatibility));
      previewCompatibility();
    }
    if (c.id) {
      form.querySelector('[data-gateway-purpose-label]').textContent = jvT("ui.472590ae974d");
      form.querySelector('[data-gateway-level-label]').textContent = jvT("ui.86da9c960cf9");
    }
    form.querySelector('[data-gateway-save]').textContent = c.id ? jvT("ui.31e392d1c037") : jvT("ui.4759498ac2a7");
    form.querySelector('[data-gateway-advanced]').hidden = !c.id;
    for (const selector of ['[data-gateway-delete]', '[data-gateway-live-note]']) form.querySelector(selector).hidden = !c.id;
    field('enabled').checked = Boolean(c.enabled);
    const syncEnabled = () => {
      const enabled = field('enabled').checked;
      const state = form.querySelector('[data-gateway-connection-state]');
      state.textContent = enabled ? 'Enable' : jvT("ui.b7e3e4aa4257");
      state.className = enabled ? 'gateway-connection-state success' : 'gateway-connection-state error';
    };
    field('enabled').addEventListener('change', syncEnabled);
    syncEnabled();
    if (c.expires_at) {
      const date = new Date(c.expires_at);
      field('expires_at').value = new Date(date.getTime() - date.getTimezoneOffset() * 60000).toISOString().slice(0, 16);
    }
    const oauthMode = Boolean(c.id) && c.auth_mode === 'oauth';
    const localOAuth = Boolean(c.id) && c.auth_mode === 'oauth_local';
    const oauthFields = form.querySelector('[data-gateway-oauth]');
    if (oauthMode) {
      form.querySelector('[data-gateway-advanced]').append(oauthFields);
      oauthFields.hidden = false;
      for (const name of ['issuer', 'audience', 'jwks_uri', 'scope', 'subject', 'client_id']) field(name).value = c.oauth?.[name] || '';
      for (const name of ['issuer', 'audience', 'jwks_uri']) field(name).required = true;
    } else oauthFields.remove();
    renderMinecraftScope(form, c);
    form.querySelector('[data-gateway-actions]').hidden = !c.id || c.auth_mode !== 'token';
    for (const action of ['rotate', 'revoke']) form.querySelector(`[data-gateway-${action}]`).hidden = !c.id || c.auth_mode !== 'token';
    if (c.id) {
      const onboarding = text('section', ''); onboarding.className = 'gateway-guide gateway-settings-guide';
      onboarding.setAttribute('aria-live', 'polite');
      form.querySelector('[data-gateway-status]').after(onboarding);
      const previewProvider = () => {
        const purpose = field('purpose').value;
        const automation = purpose === 'Automation / REST API';
        const address = location.origin + (automation ? '/api/v1' : localOAuth ? '/mcp/' + encodeURIComponent(c.id) : '/mcp');
        renderProviderGuide(onboarding, purpose, address, {settings: true});
      };
      form.querySelectorAll('input[name="purpose"]').forEach(input => input.addEventListener('change', previewProvider));
      previewProvider();
    }
    const status = form.querySelector('[data-gateway-status-details]');
    form.querySelector('[data-gateway-status]').hidden = !c.id;
    status.className = 'minecraft-server-metrics gateway-status-metrics';
    const expiration = text('div', '');
    if (c.id) {
      expiration.append(text('span', "EXPIRATION"), field('expires_at')); status.append(expiration);
    }
    const safeDetails = form.querySelector('[data-gateway-safe-details]');
    if (c.id) {
      for (const [name, value] of [[jvT("ui.f8ca8ad99b78"), c.id], [jvT("ui.97db311936e4"), c.fingerprint || jvT("ui.dc937b598926")], [jvT("ui.d70b9e24bca2"), settingsDate(c.created_at)], [jvT("ui.3a5ecca188c0"), settingsDate(c.updated_at)]]) {
        safeDetails.append(text('dt', name), text('dd', value || jvT("ui.ca1844969742")));
      }
      const state = !c.enabled ? jvT("ui.75081b593d15") : !snapshot.enabled ? jvT("ui.9300c6044988") : jvT("ui.5fa7aac5375c");
      const auth = c.auth_mode === 'oauth_local' || c.auth_mode === 'oauth' ? 'OAuth' : jvT("ui.d2089be67295");
      for (const [label, value] of [[jvT("ui.8c2e4a035f5f"), state], [jvT("ui.69b1d8bd3ab7"), activityLabel(c.last_activity)], ["AUTH", auth], [jvT("ui.f035065747e2"), c.all_instances ? jvT("ui.3205261a6537") : `${(c.instance_ids || []).length} servers`]]) {
        const metric = text('div', ''); metric.append(text('span', label), text('strong', value)); status.append(metric);
      }
    }
    const setting = action => form.querySelector(`[data-gateway-${action}]`);
    for (const action of ['rotate', 'revoke', 'delete']) {
      const button = setting(action);
      button.addEventListener('click', () => {
        if (['delete', 'revoke'].includes(action) && !window.confirm(jvT("ui.de572b7a9876", {v0: (action === 'delete' ? 'Delete' : jvT("ui.d4f88304623c")), v1: (c.name)}))) return;
        mutate({action, id: c.id});
      });
    }
    form.addEventListener('submit', event => {
      event.preventDefault();
      if (busy || !form.reportValidity()) return;
      const updated = {...c, name: field('name').value.trim(), purpose: field('purpose').value.trim(), auth_mode: c.id ? c.auth_mode : providerAuthMode(field('purpose').value.trim()), level: Number(field('level').value), enabled: field('enabled').checked, all_instances: (field('server_scope').value === 'all')};
      updated.instance_ids = updated.all_instances ? [] : [...form.querySelectorAll('input[name="instance_ids"]:checked')].map(input => input.value);
      if (c.id && field('expires_at').value) updated.expires_at = new Date(field('expires_at').value).toISOString();
      else delete updated.expires_at;
      if (updated.auth_mode === 'oauth') updated.oauth = Object.fromEntries(['issuer', 'audience', 'jwks_uri', 'scope', 'subject', 'client_id'].map(name => [name, field(name).value.trim()]));
      else delete updated.oauth;
      const payload = {action: c.id ? 'apply' : 'create', connection: updated};
      if (updated.level === 4 && c.level !== 4) requestLevel4(payload);
      else mutate(payload);
    });
    showSurface('settings');
  };
  const showAdd = () => {
    if (busy) return;
    selectedID = '';
    renderSettings({name: '', purpose: 'OpenAI', auth_mode: 'oauth_local', level: 1, enabled: true, all_instances: true, instance_ids: []});
  };
  find('[data-gateway-add]').addEventListener('click', showAdd);
  find('[data-gateway-connections]').addEventListener('click', showOverview);
  find('[data-gateway-help-open]').addEventListener('click', () => { if (!busy) showSurface('help'); });
  for (const button of dialog.querySelectorAll('[data-gateway-copy-address]')) {
    button.addEventListener('click', async () => {
      const status = find('[data-gateway-address-status]');
      const field = find(`[data-gateway-${button.dataset.gatewayCopyAddress}]`);
      await copyText(field.value, field, status, button);
    });
  }
  for (const [selector, path] of [['[data-gateway-mcp]', '/mcp'], ['[data-gateway-rest]', '/api/v1'], ['[data-gateway-openapi]', '/api/v1/openapi.json'], ['[data-gateway-metadata]', '/.well-known/oauth-protected-resource/mcp']]) {
    const element = find(selector);
    if (element.tagName === 'INPUT') {
      element.value = location.origin + path;
      element.addEventListener('click', () => element.select());
    } else element.textContent = location.origin + path;
    if (element.tagName === 'A') element.href = path;
  }
  setupHelpGuide();
  dialog.addEventListener('telegram-state', event => {
    telegramState = event.detail;
    renderCards();
  });
  dialog.addEventListener('discord-state', event => {
    discordState = event.detail;
    renderCards();
  });
  const workspace = setupWorkspaceWindow(dialog, {onOpen: () => { resetHelpGuide(); load(); showSurface(surface); }, onClose: resetHelpGuide});
  launcher.addEventListener('click', () => {
    if (!primary()) return;
    const center = document.querySelector('[data-control-center]'); if (center) center.open = false;
    workspace.open();
  });
  find('[data-gateway-close]').addEventListener('click', () => workspace.close());
  find('[data-gateway-refresh]').addEventListener('click', () => {
    if (surface === 'telegram') dialog.dispatchEvent(new Event('telegram-refresh'));
    else if (surface === 'discord') dialog.dispatchEvent(new Event('discord-refresh'));
    else {
      if (surface === 'overview') {
        dialog.dispatchEvent(new Event('telegram-refresh'));
        dialog.dispatchEvent(new Event('discord-refresh'));
      }
      if (!busy) load();
    }
  });
  const restore = () => {
    if (document.body.classList.contains('role-pending')) return false;
    if (primary() && readWorkspaceWindowState('agent-gateway').open) workspace.open();
    return true;
  };
  if (!restore()) {
    const observer = new MutationObserver(() => { if (restore()) observer.disconnect(); });
    observer.observe(document.body, {attributes: true, attributeFilter: ['class']});
  }
})();
