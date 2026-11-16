(() => {
  'use strict';
  const dialog = document.querySelector('[data-agent-gateway-dialog]');
  const surface = dialog?.querySelector('[data-gateway-discord]');
  if (!surface) return;
  const find = selector => dialog.querySelector(selector);
  const workspaceOpen = () => dialog.open;
  const active = () => workspaceOpen() && !surface.hidden;
  const primary = () => document.body.classList.contains('primary-administrator');
  const token = find('[data-discord-token]');
  const feedback = find('[data-discord-feedback]');
  const status = find('[data-discord-status]');
  const enabled = find('[data-discord-enabled]');
  const guild = find('[data-discord-guild]');
  const bind = find('[data-discord-bind]');
  const install = find('[data-discord-install]');
  const labels = {disconnected: 'Disconnected', unverified: 'Needs verification', verified: 'Bot verified', disabled: 'Disabled', connecting: 'Connecting', connected: 'Connected', connection_retry: 'Retrying', connection_paused: 'Needs attention'};
  const failure = 'Discord settings could not be loaded or saved. Check the bot token, session, and connection.';
  let availableGuilds = new Set();
  let busy = false;
  let configured = false;
  let replacing = false;
  let generation = 0;
  let surfaceGeneration = 0;
  let refreshQueued = false;
  // Only allowlisted, non-secret presentation fields leave this controller.
  let presentation = {configured: false, verified: false, enabled: false, status: '', bot_username: ''};
  const clearToken = () => { token.value = ''; };
  const renderHeader = () => {
    find('[data-discord-global]').hidden = !presentation.configured;
    enabled.checked = presentation.enabled;
    enabled.disabled = busy || !presentation.verified;
  };
  const renderBot = () => {
    find('[data-discord-token-form]').hidden = configured && !replacing;
    find('[data-discord-bot-actions]').hidden = !configured || replacing;
    find('[data-discord-token-cancel]').hidden = !configured || !replacing;
  };
  const report = message => {
    feedback.textContent = message;
    if (!active()) find('[data-gateway-feedback]').textContent = message;
  };
  const node = (tag, text) => { const element = document.createElement(tag); element.textContent = text; return element; };
  const displayStatus = snapshot => {
    if (snapshot?.configured !== true) return snapshot?.status === 'disconnected' ? 'disconnected' : '';
    if (snapshot.verified !== true) return 'unverified';
    if (snapshot.enabled !== true) return 'disabled';
    if (typeof snapshot.runtime_status === 'string') return Object.hasOwn(labels, snapshot.runtime_status) ? snapshot.runtime_status : '';
    return ['disconnected', 'unverified', 'verified'].includes(snapshot.status) ? snapshot.status : '';
  };
  const render = snapshot => {
    presentation = {
      configured: snapshot?.configured === true, verified: snapshot?.verified === true,
      enabled: snapshot?.enabled === true, status: displayStatus(snapshot),
      bot_username: snapshot?.verified === true && typeof snapshot.bot_username === 'string' ? snapshot.bot_username : '',
    };
    configured = presentation.configured;
    renderHeader();
    dialog.dispatchEvent(new CustomEvent('discord-state', {detail: {...presentation}}));
    if (!active()) return;
    status.textContent = `${presentation.bot_username ? `@${presentation.bot_username}  ` : ''}● ${labels[presentation.status] || 'Status unavailable'}`;
    status.className = `gateway-connection-state${['verified', 'connected'].includes(presentation.status) ? ' success' : ['connection_retry', 'connection_paused'].includes(presentation.status) ? ' error' : presentation.status === 'unverified' ? ' warning' : ''}`;
    renderBot();
    find('[data-discord-verify]').hidden = presentation.verified;
    find('[data-discord-verify]').disabled = !configured;
    const renderAdministrators = (selector, entries, pending) => {
      const list = find(selector); list.replaceChildren();
      for (const entry of (Array.isArray(entries) ? entries.slice(0, 32) : [])) {
        if (typeof entry?.user_id !== 'string' || !/^[1-9][0-9]{0,19}$/.test(entry.user_id)) continue;
        const row = node('div', ''); row.className = 'integration-lane';
        row.append(node('strong', 'Access'), node('span', (typeof entry.name === 'string' && entry.name) || (typeof entry.username === 'string' && entry.username) || 'Discord user'));
        const secondary = node('span', `User ${entry.user_id}`); secondary.className = 'muted'; row.append(secondary);
        for (const [label, action] of (pending ? [['Approve', 'admin_approve'], ['Reject', 'admin_reject']] : [['Remove', 'admin_remove']])) {
          const button = node('button', label); button.type = 'button'; button.className = 'button';
          button.addEventListener('click', () => request({action, user_id: entry.user_id})); row.append(button);
        }
        list.append(row);
      }
    };
    renderAdministrators('[data-discord-pending]', snapshot?.pending, true);
    renderAdministrators('[data-discord-administrators]', snapshot?.administrators, false);
    const previous = guild.value;
    availableGuilds = new Set(); guild.replaceChildren();
    const placeholder = node('option', 'Choose a Discord server'); placeholder.value = ''; guild.append(placeholder);
    for (const entry of (Array.isArray(snapshot?.guilds) ? snapshot.guilds.slice(0, 200) : [])) {
      if (typeof entry?.id !== 'string' || !/^[1-9][0-9]{0,19}$/.test(entry.id) || typeof entry.name !== 'string') continue;
      const option = node('option', entry.name); option.value = entry.id; guild.append(option); availableGuilds.add(entry.id);
    }
    guild.value = availableGuilds.has(previous) ? previous : '';
    guild.disabled = !presentation.verified || !presentation.enabled || snapshot?.guilds_ready !== true || availableGuilds.size === 0;
    bind.disabled = guild.disabled || !availableGuilds.has(guild.value);
    find('[data-discord-empty]').hidden = snapshot?.guilds_ready !== true || availableGuilds.size !== 0;
    find('[data-discord-bound]').textContent = typeof snapshot?.guild_name === 'string' && snapshot.guild_name ? snapshot.guild_name : 'None';
    find('[data-discord-bound-unavailable]').hidden = !snapshot?.guild_id || !presentation.enabled || snapshot?.guilds_ready !== true || snapshot?.bound_guild_available === true;
    install.hidden = true; install.href = '';
    if (presentation.verified && typeof snapshot.application_id === 'string' && /^[1-9][0-9]{0,19}$/.test(snapshot.application_id)) {
      install.href = `https://discord.com/oauth2/authorize?client_id=${snapshot.application_id}&scope=bot&permissions=3072`;
      install.hidden = false;
    }
  };
  const send = async payload => {
    const options = {credentials: 'same-origin', cache: 'no-store', headers: {Accept: 'application/json'}};
    if (payload) {
      options.method = 'POST';
      options.body = new URLSearchParams({csrf: find('[data-discord-content]').dataset.discordCsrf, request: JSON.stringify(payload)});
    }
    const response = await fetch('/api/discord', options);
    if (!response.ok || response.redirected) throw new Error(failure);
    return response.json();
  };
  const request = async payload => {
    const globalChange = payload?.action === 'enabled';
    if (busy || !primary() || !workspaceOpen() || (payload && !globalChange && !active())) {
      if (!payload && busy && workspaceOpen()) refreshQueued = true;
      if (payload && Object.hasOwn(payload, 'token')) payload.token = '';
      renderHeader(); return;
    }
    const currentGeneration = generation;
    const currentSurface = surfaceGeneration;
    const current = () => currentGeneration === generation && workspaceOpen() && (globalChange || currentSurface === surfaceGeneration);
    busy = true; renderHeader();
    report(payload ? 'Saving…' : '');
    try {
      const snapshot = await send(payload);
      if (current()) {
        if (['credential', 'disconnect'].includes(payload?.action)) replacing = false;
        render(snapshot); report(payload ? 'Saved.' : '');
      }
    } catch (_) {
      if (current()) {
        // A failed mutation retains the last authoritative enable state.
        if (!payload) render(null);
        report(failure);
      }
    } finally {
      if (payload && Object.hasOwn(payload, 'token')) payload.token = '';
      busy = false; renderHeader();
      if (refreshQueued && workspaceOpen()) { refreshQueued = false; request(); }
    }
  };
  const leaveSurface = () => {
    surfaceGeneration++; clearToken(); replacing = false; renderBot();
    guild.disabled = true; bind.disabled = true;
    install.hidden = true; install.href = '';
  };
  const stop = () => {
    generation++; leaveSurface(); refreshQueued = false;
    presentation = {configured: false, verified: false, enabled: false, status: '', bot_username: ''};
    configured = false; renderBot(); renderHeader();
    dialog.dispatchEvent(new CustomEvent('discord-state', {detail: {...presentation}}));
  };
  dialog.addEventListener('gateway-surface-change', () => { leaveSurface(); if (workspaceOpen()) request(); });
  dialog.addEventListener('close', stop);
  dialog.addEventListener('cancel', stop);
  dialog.addEventListener('discord-refresh', () => request());
  window.addEventListener('pagehide', stop);
  find('[data-discord-token-form]').addEventListener('submit', event => {
    event.preventDefault();
    const value = token.value; clearToken();
    request({action: 'credential', token: value});
  });
  find('[data-discord-replace]').addEventListener('click', () => { if (!active() || busy) return; replacing = true; clearToken(); renderBot(); token.focus(); });
  find('[data-discord-token-cancel]').addEventListener('click', () => { replacing = false; clearToken(); renderBot(); find('[data-discord-replace]').focus(); });
  find('[data-discord-verify]').addEventListener('click', () => request({action: 'verify'}));
  enabled.addEventListener('change', () => request({action: 'enabled', enabled: enabled.checked}));
  guild.addEventListener('change', () => { bind.disabled = busy || guild.disabled || !availableGuilds.has(guild.value); });
  bind.addEventListener('click', () => { if (!guild.disabled && availableGuilds.has(guild.value)) request({action: 'guild', guild_id: guild.value}); });
  find('[data-discord-disconnect]').addEventListener('click', () => request({action: 'disconnect'}));
  renderHeader();
  if (workspaceOpen()) request();
})();
