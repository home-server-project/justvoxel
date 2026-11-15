(() => {
  'use strict';
  const dialog = document.querySelector('[data-agent-gateway-dialog]');
  const surface = dialog?.querySelector('[data-gateway-discord]');
  if (!surface) return;
  const find = selector => surface.querySelector(selector);
  const active = () => dialog.open && !surface.hidden;
  const primary = () => document.body.classList.contains('primary-administrator');
  const token = find('[data-discord-token]');
  const feedback = find('[data-discord-feedback]');
  const identity = find('[data-discord-identity]');
  const status = find('[data-discord-status]');
  const labels = {disconnected: 'Disconnected', unverified: 'Token saved — verify the bot', verified: 'Bot verified'};
  const enabled = find('[data-discord-enabled]');
  const guild = find('[data-discord-guild]');
  const bind = find('[data-discord-bind]');
  const install = find('[data-discord-install]');
  const runtimeLabels = {disabled: 'Disabled', connecting: 'Connecting', connected: 'Connected', connection_retry: 'Retrying connection', connection_paused: 'Connection paused — disable and enable Discord to retry'};
  let availableGuilds = new Set();
  let busy = false;
  let generation = 0;
  const clearToken = () => { token.value = ''; };
  const clearIdentity = () => { identity.replaceChildren(); identity.hidden = true; };
  const unavailable = () => {
    status.textContent = 'Connection unavailable'; clearIdentity();
    enabled.disabled = true; enabled.checked = false;
    guild.disabled = true; guild.replaceChildren(); availableGuilds.clear(); bind.disabled = true;
    install.hidden = true; install.href = '';
    find('[data-discord-runtime]').textContent = 'Connection unavailable';
    find('[data-discord-bound]').textContent = 'Unavailable';
    find('[data-discord-empty]').hidden = true;
    find('[data-discord-bound-unavailable]').hidden = true;
  };
  const node = (tag, text) => { const element = document.createElement(tag); element.textContent = text; return element; };
  const render = snapshot => {
    clearIdentity();
    const previous = guild.value;
    availableGuilds = new Set();
    guild.replaceChildren();
    const placeholder = node('option', 'Choose a Discord server'); placeholder.value = ''; guild.append(placeholder);
    const guilds = Array.isArray(snapshot?.guilds) ? snapshot.guilds.slice(0, 200) : [];
    for (const entry of guilds) {
      if (typeof entry?.id !== 'string' || !/^[1-9][0-9]{0,19}$/.test(entry.id) || typeof entry.name !== 'string') continue;
      const option = node('option', entry.name); option.value = entry.id; guild.append(option); availableGuilds.add(entry.id);
    }
    guild.value = availableGuilds.has(previous) ? previous : '';
    enabled.checked = snapshot?.enabled === true;
    enabled.disabled = snapshot?.verified !== true;
    guild.disabled = snapshot?.enabled !== true || snapshot?.guilds_ready !== true || availableGuilds.size === 0;
    bind.disabled = guild.disabled || !availableGuilds.has(guild.value);
    const runtime = snapshot?.runtime_status;
    find('[data-discord-runtime]').textContent = typeof runtime === 'string' && Object.hasOwn(runtimeLabels, runtime) ? runtimeLabels[runtime] : 'Connection unavailable';
    find('[data-discord-empty]').hidden = snapshot?.guilds_ready !== true || availableGuilds.size !== 0;
    find('[data-discord-bound]').textContent = typeof snapshot?.guild_name === 'string' && snapshot.guild_name ? snapshot.guild_name : 'None';
    find('[data-discord-bound-unavailable]').hidden = !snapshot?.guild_id || snapshot?.enabled !== true || snapshot?.guilds_ready !== true || snapshot?.bound_guild_available === true;
    install.hidden = true; install.href = '';
    if (snapshot?.verified === true && typeof snapshot.application_id === 'string' && /^[1-9][0-9]{0,19}$/.test(snapshot.application_id)) {
      install.href = `https://discord.com/oauth2/authorize?client_id=${snapshot.application_id}&scope=bot&permissions=3072`;
      install.hidden = false;
    }
    const state = snapshot && typeof snapshot.status === 'string' && Object.hasOwn(labels, snapshot.status) ? snapshot.status : '';
    status.textContent = labels[state] || 'Connection unavailable';
    find('[data-discord-verify]').disabled = snapshot?.configured !== true;
    if (state !== 'verified' || snapshot.verified !== true) return;
    const safe = value => typeof value === 'string' ? value : '';
    identity.append(node('h4', 'Bot'), node('p', safe(snapshot.bot_name)));
    if (safe(snapshot.bot_username)) identity.append(node('p', `@${snapshot.bot_username}`));
    identity.append(node('h4', 'Discord application'), node('p', safe(snapshot.application_name)));
    identity.hidden = false;
  };
  const send = async payload => {
    const options = {credentials: 'same-origin', cache: 'no-store', headers: {Accept: 'application/json'}};
    if (payload) {
      options.method = 'POST';
      options.body = new URLSearchParams({csrf: find('[data-discord-content]').dataset.discordCsrf, request: JSON.stringify(payload)});
    }
    const response = await fetch('/api/discord', options);
    if (!response.ok || response.redirected) throw new Error('Discord settings could not be loaded or saved. Check the bot token, session, and connection.');
    return response.json();
  };
  const request = async payload => {
    if (busy || !primary() || !active()) { if (payload && Object.hasOwn(payload, 'token')) payload.token = ''; return; }
    busy = true;
    const current = generation;
    feedback.textContent = payload ? 'Saving…' : '';
    try {
      const snapshot = await send(payload);
      if (current === generation && active()) {
        render(snapshot); feedback.textContent = payload ? 'Saved.' : '';
      }
    } catch (_) {
      if (current === generation && active()) {
        unavailable(); feedback.textContent = 'Discord settings could not be loaded or saved. Check the bot token, session, and connection.';
      }
    } finally {
      if (payload && Object.hasOwn(payload, 'token')) payload.token = '';
      clearToken(); busy = false;
      if (current !== generation && active()) request();
    }
  };
  const stop = () => { generation++; clearToken(); };
  dialog.addEventListener('gateway-surface-change', () => {
    stop();
    if (active()) { unavailable(); request(); }
  });
  dialog.addEventListener('close', stop);
  dialog.addEventListener('cancel', stop);
  dialog.addEventListener('discord-refresh', () => request());
  window.addEventListener('pagehide', stop);
  find('[data-discord-token-form]').addEventListener('submit', event => {
    event.preventDefault();
    const value = token.value; clearToken();
    request({action: 'credential', token: value});
  });
  find('[data-discord-verify]').addEventListener('click', () => request({action: 'verify'}));
  enabled.addEventListener('change', () => request({action: 'enabled', enabled: enabled.checked}));
  guild.addEventListener('change', () => { bind.disabled = busy || guild.disabled || !availableGuilds.has(guild.value); });
  bind.addEventListener('click', () => { if (!guild.disabled && availableGuilds.has(guild.value)) request({action: 'guild', guild_id: guild.value}); });
  find('[data-discord-disconnect]').addEventListener('click', () => request({action: 'disconnect'}));
})();
