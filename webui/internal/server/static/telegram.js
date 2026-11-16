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
  // Human edits live outside render; polling remains authoritative for unedited users.
  let pendingDraft = null;
  const userDrafts = new Map();
  let configured = false;
  let replacing = false;
  // Ignore responses from a surface session that has already been left.
  let generation = 0;
  const renderBot = () => {
    find('[data-telegram-token-form]').hidden = configured && !replacing;
    find('[data-telegram-bot-actions]').hidden = !configured || replacing;
    find('[data-telegram-token-cancel]').hidden = !configured || !replacing;
  };
  const permission = (draft, edited) => {
    const group = node('div', ''); group.className = 'telegram-permission';
    group.setAttribute('role', 'group'); group.setAttribute('aria-label', 'Permission');
    const choices = [];
    for (const [key, label] of [['read_only', 'Read Only'], ['server_management', 'Server Management']]) {
      const choice = button(label, () => {
        draft.permission = key; edited();
        for (const [value, control] of choices) control.setAttribute('aria-pressed', String(value === key));
      });
      choice.setAttribute('aria-pressed', String(draft.permission === key));
      choices.push([key, choice]); group.append(choice);
    }
    return group;
  };
  const notifications = (draft, edited) => {
    const label = node('label', ' Notifications'); label.className = 'gateway-switch';
    const input = document.createElement('input'); input.type = 'checkbox'; input.checked = Boolean(draft.notifications);
    input.setAttribute('role', 'switch');
    input.addEventListener('change', () => { draft.notifications = input.checked; edited(); });
    label.prepend(input); return label;
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
    const status = find('[data-telegram-status]');
    status.textContent = `${snapshot.bot_name ? `@${snapshot.bot_name}  ` : ''}● ${labels[snapshot.status] || 'Connection unavailable'}`;
    status.className = `gateway-connection-state${snapshot.status === 'connected' ? ' success' : ['connection_retry', 'connection_paused'].includes(snapshot.status) ? ' error' : ''}`;
    configured = Boolean(snapshot.configured); renderBot();
    enabled.checked = snapshot.enabled; enabled.disabled = !snapshot.verified;
    find('[data-telegram-verify]').disabled = !snapshot.configured;
    find('[data-telegram-pair]').disabled = !snapshot.enabled;
    find('[data-telegram-operation]').textContent = snapshot.operation_status ? snapshot.operation_status.replaceAll('_', ' ') : 'No Telegram operations yet.';
    const users = find('[data-telegram-users]'); users.replaceChildren();
    const userIDs = new Set((snapshot.users || []).map(user => user.id));
    for (const id of userDrafts.keys()) if (!userIDs.has(id)) userDrafts.delete(id);
    for (const user of snapshot.users || []) {
      const row = node('div', ''); row.className = 'telegram-lane';
      const draft = userDrafts.get(user.id) || {permission: user.permission, notifications: user.notifications};
      const edited = () => userDrafts.set(user.id, draft);
      row.append(node('strong', 'Access'), node('span', `User ${user.id}`), permission(draft, edited), notifications(draft, edited),
        button('Save', () => change({action: 'user', user_id: user.id, permission: draft.permission, notifications: draft.notifications})),
        button('Revoke', () => change({action: 'revoke', user_id: user.id})));
      users.append(row);
    }
    const pending = find('[data-telegram-pending]'); pending.replaceChildren();
    const pairing = snapshot.pairing && new Date(snapshot.pairing.expires).getTime() > Date.now() ? snapshot.pairing : null;
    const pendingID = pairing?.user_id > 0 ? pairing.user_id : null;
    if (!pendingID || pendingDraft?.id !== pendingID) pendingDraft = pendingID ? {id: pendingID, permission: 'read_only', notifications: false} : null;
    if (pairing) {
      if (pendingDraft) {
        challenge.textContent = '';
        const draft = pendingDraft;
        pending.append(node('span', `User ${draft.id}`), permission(draft, () => {}), notifications(draft, () => {}),
          button('Approve', () => change({action: 'approve', user_id: draft.id, permission: draft.permission, notifications: draft.notifications})));
      }
      const expiry = node('small', `Expires ${new Date(pairing.expires).toLocaleString()}`); expiry.className = 'muted'; pending.append(expiry);
    } else challenge.textContent = '';
    if (pairing && !pendingID && snapshot.pairing_command) challenge.textContent = `Send once in a private chat with the bot: ${snapshot.pairing_command}. Do not share this challenge. Verify the numeric user ID before approval.`;
  };
  const change = async payload => {
    if (busy || !active()) return;
    const requestGeneration = generation;
    busy = true; feedback.textContent = 'Saving…';
    try {
      const snapshot = await send(payload);
      if (requestGeneration !== generation || !active()) return;
      if (['approve', 'credential', 'disconnect'].includes(payload.action)) pendingDraft = null;
      const savedDraft = userDrafts.get(payload.user_id);
      if (payload.action === 'revoke' || (payload.action === 'user' && savedDraft?.permission === payload.permission && savedDraft?.notifications === payload.notifications)) userDrafts.delete(payload.user_id);
      if (['credential', 'disconnect'].includes(payload.action)) { userDrafts.clear(); replacing = false; }
      render(snapshot); feedback.textContent = 'Saved.';
    } catch (error) { if (requestGeneration === generation && active()) feedback.textContent = error.message; }
    finally { if (payload && Object.hasOwn(payload, 'token')) payload.token = ''; token.value = ''; busy = false; if (!active()) clearSecrets(); }
  };
  const load = async () => {
    if (busy || !primary() || !active()) return;
    const requestGeneration = generation;
    busy = true;
    try {
      const snapshot = await send();
      if (requestGeneration === generation && active()) { render(snapshot); feedback.textContent = ''; }
    }
    catch (error) { if (requestGeneration === generation && active()) feedback.textContent = error.message; }
    finally { busy = false; if (!active()) clearSecrets(); }
  };
  const clearSecrets = () => { token.value = ''; challenge.textContent = ''; };
  const stop = () => { generation++; clearSecrets(); pendingDraft = null; userDrafts.clear(); replacing = false; renderBot(); clearInterval(refreshTimer); };
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
  find('[data-telegram-replace]').addEventListener('click', () => { replacing = true; token.value = ''; renderBot(); token.focus(); });
  find('[data-telegram-token-cancel]').addEventListener('click', () => { replacing = false; token.value = ''; renderBot(); find('[data-telegram-replace]').focus(); });
  find('[data-telegram-verify]').addEventListener('click', () => change({action: 'verify'}));
  enabled.addEventListener('change', () => change({action: 'enabled', enabled: enabled.checked}));
  find('[data-telegram-pair]').addEventListener('click', () => change({action: 'pair'}));
  find('[data-telegram-disconnect]').addEventListener('click', () => change({action: 'disconnect'}));
})();
