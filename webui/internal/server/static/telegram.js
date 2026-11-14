(() => {
  'use strict';
  const dialog = document.querySelector('[data-agent-gateway-dialog]');
  const surface = dialog?.querySelector('[data-gateway-telegram]');
  if (!surface) return;
  const active = () => dialog.open && !surface.hidden;
  const find = selector => dialog.querySelector(selector);
  const primary = () => document.body.classList.contains('primary-administrator');
  const token = find('[data-telegram-token]');
  const feedback = find('[data-telegram-feedback]');
  const challenge = find('[data-telegram-challenge]');
  const enabled = find('[data-telegram-enabled]');
  let busy = false;
  let refreshTimer;
  const node = (tag, text) => { const element = document.createElement(tag); element.textContent = text; return element; };
  const button = (label, action) => { const element = node('button', label); element.type = 'button'; element.className = 'button'; element.addEventListener('click', action); return element; };
  const permission = value => {
    const select = document.createElement('select');
    for (const [key, label] of [['read_only', 'Read Only'], ['server_management', 'Server Management']]) {
      const option = node('option', label); option.value = key; select.append(option);
    }
    select.value = value || 'read_only'; select.setAttribute('aria-label', 'Permission'); return select;
  };
  const notifications = value => {
    const label = node('label', ' Notifications ');
    const input = document.createElement('input'); input.type = 'checkbox'; input.checked = Boolean(value); label.prepend(input);
    return {label, input};
  };
  const send = async payload => {
    if (!primary()) throw new Error('Sign in as the primary voxel administrator.');
    const options = {credentials: 'same-origin', cache: 'no-store', headers: {Accept: 'application/json'}};
    if (payload) {
      options.method = 'POST';
      options.body = new URLSearchParams({csrf: find('[data-telegram-content]').dataset.telegramCsrf, request: JSON.stringify(payload)});
    }
    const response = await fetch('/api/telegram', options);
    if (!response.ok || response.redirected) {
      let message = 'Telegram settings could not be loaded or saved. Verify the bot token and use a dedicated bot without another polling app or webhook.';
      try { const result = await response.json(); if (typeof result.error === 'string' && result.error.length <= 300) message = result.error; } catch (_) { /* Keep the fixed fallback. */ }
      throw new Error(message);
    }
    return response.json();
  };
  const render = snapshot => {
    const labels = {disconnected: 'Disconnected', unverified: 'Token saved — verify the bot', verified: 'Bot verified', connecting: 'Connecting', connected: 'Connected', disabled: 'Disabled', connection_retry: 'Connection interrupted — retrying', connection_paused: 'Connection paused — verify the bot and enable again'};
    find('[data-telegram-status]').textContent = `${snapshot.bot_name ? `@${snapshot.bot_name} · ` : ''}${labels[snapshot.status] || 'Connection unavailable'}`;
    enabled.checked = snapshot.enabled; enabled.disabled = !snapshot.verified;
    find('[data-telegram-verify]').disabled = !snapshot.configured;
    find('[data-telegram-pair]').disabled = !snapshot.enabled;
    find('[data-telegram-operation]').textContent = snapshot.operation_status ? snapshot.operation_status.replaceAll('_', ' ') : 'No Telegram operations yet.';
    const users = find('[data-telegram-users]'); users.replaceChildren();
    for (const user of snapshot.users || []) {
      const row = document.createElement('section'); row.className = 'panel';
      row.append(node('h4', `Telegram user ${user.id}`));
      const select = permission(user.permission);
      const notify = notifications(user.notifications);
      row.append(select, notify.label,
        button('Save user', () => change({action: 'user', user_id: user.id, permission: select.value, notifications: notify.input.checked})),
        button('Revoke access', () => change({action: 'revoke', user_id: user.id})));
      users.append(row);
    }
    if (!snapshot.users?.length) users.append(node('p', 'No approved users.'));
    const pending = find('[data-telegram-pending]'); pending.replaceChildren();
    if (snapshot.pairing) {
      pending.append(node('p', `Pairing expires ${new Date(snapshot.pairing.expires).toLocaleString()}.`));
      if (snapshot.pairing.user_id > 0) {
        challenge.textContent = '';
        pending.append(node('p', `Review numeric Telegram user ID ${snapshot.pairing.user_id} with the person you intend to approve.`));
        const select = permission(); const notify = notifications(false);
        pending.append(select, notify.label, button('Approve this user', () => change({action: 'approve', user_id: snapshot.pairing.user_id, permission: select.value, notifications: notify.input.checked})));
      }
    } else challenge.textContent = '';
    if (snapshot.pairing_command) challenge.textContent = `Send this once in a private chat with the bot: ${snapshot.pairing_command}. Then Refresh and explicitly approve the numeric user ID. Do not share this challenge with anyone else.`;
  };
  const change = async payload => {
    if (busy || !active()) return;
    busy = true; feedback.textContent = 'Saving…';
    try {
      const snapshot = await send(payload);
      render(snapshot); feedback.textContent = 'Saved.';
    } catch (error) { feedback.textContent = error.message; }
    finally { if (payload && Object.hasOwn(payload, 'token')) payload.token = ''; token.value = ''; busy = false; if (!active()) clearSecrets(); }
  };
  const load = async () => {
    if (busy || !primary() || !active()) return;
    busy = true;
    try { render(await send()); feedback.textContent = ''; }
    catch (error) { feedback.textContent = error.message; }
    finally { busy = false; if (!active()) clearSecrets(); }
  };
  const clearSecrets = () => { token.value = ''; challenge.textContent = ''; };
  const stop = () => { clearSecrets(); clearInterval(refreshTimer); };
  dialog.addEventListener('gateway-surface-change', () => {
    stop();
    if (!active()) return;
    load();
    refreshTimer = setInterval(() => {
      if (active() && !find('[data-telegram-content]').contains(document.activeElement)) load();
    }, 5000);
  });
  dialog.addEventListener('close', stop);
  dialog.addEventListener('cancel', stop);
  dialog.addEventListener('telegram-refresh', load);
  window.addEventListener('pagehide', stop);
  find('[data-telegram-token-form]').addEventListener('submit', event => { event.preventDefault(); const value = token.value; token.value = ''; change({action: 'credential', token: value}); });
  find('[data-telegram-verify]').addEventListener('click', () => change({action: 'verify'}));
  enabled.addEventListener('change', () => change({action: 'enabled', enabled: enabled.checked}));
  find('[data-telegram-pair]').addEventListener('click', () => change({action: 'pair'}));
  find('[data-telegram-disconnect]').addEventListener('click', () => change({action: 'disconnect'}));
})();
