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
  let busy = false;
  let generation = 0;
  const clearToken = () => { token.value = ''; };
  const clearIdentity = () => { identity.replaceChildren(); identity.hidden = true; };
  const unavailable = () => { status.textContent = 'Connection unavailable'; clearIdentity(); };
  const node = (tag, text) => { const element = document.createElement(tag); element.textContent = text; return element; };
  const render = snapshot => {
    clearIdentity();
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
  find('[data-discord-disconnect]').addEventListener('click', () => request({action: 'disconnect'}));
})();
