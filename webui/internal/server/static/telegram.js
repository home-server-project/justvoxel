(() => {
  'use strict';
  const dialog = document.querySelector('[data-agent-gateway-dialog]');
  const surface = dialog?.querySelector('[data-gateway-telegram]');
  if (!surface) return;
  const workspaceOpen = () => dialog.open;
  const active = () => workspaceOpen() && !surface.hidden;
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
  let pairingCommand = '';
  let pairingExpires = '';
  let challengeTimer;
  let verificationFailed = false;
  // Ignore responses from a surface session that has already been left.
  let generation = 0;
  let surfaceGeneration = 0;
  let presentation = {configured: false, verified: false, enabled: false, status: '', bot_name: ''};
  const renderHeader = () => {
    find('[data-telegram-global]').hidden = !presentation.configured;
    enabled.checked = presentation.enabled;
    enabled.disabled = busy || !presentation.verified;
  };
  const report = message => {
    feedback.textContent = message;
    if (!active()) find('[data-gateway-feedback]').textContent = message;
  };
  const clearPairingCommand = () => { pairingCommand = ''; pairingExpires = ''; clearTimeout(challengeTimer); challenge.textContent = ''; find('[data-telegram-copy]').hidden = true; find('[data-telegram-expiry]').textContent = ''; };
  const renderBot = () => {
    find('[data-telegram-token-form]').hidden = configured && !replacing;
    find('[data-telegram-bot-actions]').hidden = !configured || replacing;
    find('[data-telegram-token-cancel]').hidden = !configured || !replacing;
  };
  const permission = (draft, edited) => {
    const group = node('label', jvT("ui.c0a479d66f82")); group.className = 'telegram-permission';
    const select = document.createElement('select'); select.setAttribute('aria-label', jvT("ui.229efc8f5263"));
    for (const [key, label] of [['read_only', jvT("ui.ca1048507ded")], ['server_management', jvT("ui.5754cb826093")]]) {
      const option = node('option', label); option.value = key; select.append(option);
    }
    select.value = draft.permission;
    select.addEventListener('change', () => { draft.permission = select.value; edited(); });
    group.append(select);
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
    if (!primary()) throw new Error(jvT("ui.220fabe28e2a"));
    const options = {credentials: 'same-origin', cache: 'no-store', headers: {Accept: 'application/json'}};
    if (payload) {
      options.method = "POST";
      options.body = new URLSearchParams({csrf: find('[data-telegram-content]').dataset.telegramCsrf, request: JSON.stringify(payload)});
    }
    const response = await fetch('/api/telegram', options);
    if (!response.ok || response.redirected) {
      let message = jvT("ui.136c99c24ae8");
      try { const result = await response.json(); if (typeof result.error === 'string' && result.error.length <= 300) message = result.error; } catch (_) { /* Keep the fixed fallback. */ }
      throw new Error(message);
    }
    return response.json();
  };
  const render = snapshot => {
    // Only allowlisted presentation fields leave this controller.
    presentation = {
      configured: Boolean(snapshot.configured), verified: Boolean(snapshot.verified),
      enabled: Boolean(snapshot.enabled), status: typeof snapshot.status === 'string' ? snapshot.status : '',
      bot_name: typeof snapshot.bot_name === 'string' ? snapshot.bot_name : '',
    };
    configured = presentation.configured;
    renderHeader();
    dialog.dispatchEvent(new CustomEvent('telegram-state', {detail: {...presentation}}));
    if (!active()) return;
    const labels = {disconnected: jvT("ui.04dfac3671b4"), unverified: jvT("ui.2137ad12923b"), verification_failed: jvT("ui.f0f2383f8af8"), verified: jvT("ui.f724fb6f802d"), connecting: jvT("ui.d403c686f6a1"), connected: jvT("ui.22965568d22a"), disabled: jvT("ui.75081b593d15"), connection_retry: jvT("ui.d863e61f0f53"), connection_paused: jvT("ui.bc896a6b4afe")};
    const status = find('[data-telegram-status]');
    status.textContent = `${snapshot.bot_name ? `@${snapshot.bot_name}  ` : ''}● ${labels[snapshot.status] || 'Connection unavailable'}`;
    status.className = `gateway-connection-state${['verified', 'connected'].includes(snapshot.status) ? ' success' : ['verification_failed', 'connection_retry', 'connection_paused'].includes(snapshot.status) ? ' error' : snapshot.status === 'unverified' ? ' warning' : ''}`;
    configured = Boolean(snapshot.configured); renderBot();
    find('[data-telegram-verify]').hidden = snapshot.verified;
    find('[data-telegram-verify]').disabled = !snapshot.configured;
    find('[data-telegram-pair]').disabled = !snapshot.enabled;
    find('[data-telegram-operation]').textContent = snapshot.operation_status ? snapshot.operation_status.replaceAll('_', ' ') : jvT("ui.d54609390f7a");
    const users = find('[data-telegram-users]'); users.replaceChildren();
    const userIDs = new Set((snapshot.users || []).map(user => user.id));
    for (const id of userDrafts.keys()) if (!userIDs.has(id)) userDrafts.delete(id);
    for (const user of snapshot.users || []) {
      const row = node('div', ''); row.className = 'integration-lane';
      const draft = userDrafts.get(user.id) || {permission: user.permission, notifications: user.notifications};
      const edited = () => userDrafts.set(user.id, draft);
      row.append(node('strong', jvT("ui.ec5ba0abb717")), node('span', jvT("ui.19de1acb0e29", {v0: (user.id)})), permission(draft, edited), notifications(draft, edited),
        button(jvT("ui.1509f561f241"), () => change({action: 'user', user_id: user.id, permission: draft.permission, notifications: draft.notifications})),
        button(jvT("ui.87e6d00bbf53"), () => change({action: 'revoke', user_id: user.id})));
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
        pending.append(node('span', jvT("ui.19de1acb0e29", {v0: (draft.id)})), permission(draft, () => {}), notifications(draft, () => {}),
          button(jvT("ui.6007acbe30b2"), () => change({action: 'approve', user_id: draft.id, permission: draft.permission, notifications: draft.notifications})));
      }
    }
    if (!pairing || pendingID || !snapshot.configured || pairing.expires !== pairingExpires) clearPairingCommand();
    challenge.textContent = pairingCommand;
    find('[data-telegram-copy]').hidden = !pairingCommand;
    find('[data-telegram-expiry]').textContent = pairing && !pendingID ? jvT("ui.5503bbe0d985", {v0: (new Date(pairing.expires).toLocaleTimeString())}) : '';
  };
  const change = async payload => {
    const globalChange = payload.action === 'enabled';
    if (busy || !workspaceOpen() || (!globalChange && !active())) { renderHeader(); return; }
    const requestGeneration = generation;
    const requestSurface = surfaceGeneration;
    const current = () => requestGeneration === generation && workspaceOpen() && (globalChange || (requestSurface === surfaceGeneration && active()));
    busy = true; renderHeader(); report(jvT("ui.23e39291d613"));
    try {
      let snapshot = await send(payload);
      if (!current()) return;
      if (['approve', 'credential', 'disconnect'].includes(payload.action)) pendingDraft = null;
      const savedDraft = userDrafts.get(payload.user_id);
      if (payload.action === 'revoke' || (payload.action === 'user' && savedDraft?.permission === payload.permission && savedDraft?.notifications === payload.notifications)) userDrafts.delete(payload.user_id);
      if (['credential', 'disconnect'].includes(payload.action)) { userDrafts.clear(); replacing = false; }
      if (payload.action === 'pair') {
        clearPairingCommand();
        pairingCommand = snapshot.pairing_command || '';
        pairingExpires = snapshot.pairing?.expires || '';
        const remaining = new Date(pairingExpires).getTime() - Date.now();
        if (pairingCommand && remaining > 0) challengeTimer = setTimeout(clearPairingCommand, remaining);
        else clearPairingCommand();
      }
      if (['credential', 'disconnect'].includes(payload.action)) { clearPairingCommand(); verificationFailed = false; }
      if (payload.action === 'credential') {
        verificationFailed = false;
        render(snapshot);
        try { snapshot = await send({action: 'verify'}); }
        catch (_) { verificationFailed = true; }
        if (!current()) return;
      } else if (payload.action === 'verify') verificationFailed = false;
      render(snapshot);
      report(verificationFailed ? jvT("ui.714d3e2f50db") : jvT("ui.fca046ccd586"));
    } catch (error) { if (current()) report(error.message); }
    finally { if (payload && Object.hasOwn(payload, 'token')) payload.token = ''; busy = false; renderHeader(); if (!active()) clearSecrets(); }
  };
  const load = async () => {
    if (busy || !primary() || !workspaceOpen()) return;
    const requestGeneration = generation;
    busy = true;
    try {
      const snapshot = await send();
      if (requestGeneration === generation && workspaceOpen()) { render(snapshot); if (!verificationFailed) feedback.textContent = ''; }
    }
    catch (error) { if (requestGeneration === generation && active()) feedback.textContent = error.message; }
    finally { busy = false; renderHeader(); if (!active()) clearSecrets(); }
  };
  const clearSecrets = () => { token.value = ''; clearPairingCommand(); };
  const leaveSurface = () => { surfaceGeneration++; verificationFailed = false; clearSecrets(); pendingDraft = null; userDrafts.clear(); replacing = false; renderBot(); clearInterval(refreshTimer); };
  const stop = () => { generation++; leaveSurface(); };
  dialog.addEventListener('gateway-surface-change', () => {
    leaveSurface();
    if (!workspaceOpen()) return;
    load();
    if (active()) refreshTimer = setInterval(() => {
      if (active()) load();
    }, 5000);
  });
  dialog.addEventListener('close', stop);
  dialog.addEventListener('cancel', stop);
  dialog.addEventListener('telegram-refresh', load);
  window.addEventListener('pagehide', stop);
  find('[data-telegram-token-form]').addEventListener('submit', event => { event.preventDefault(); const value = token.value; token.value = ''; change({action: 'credential', token: value}); });
  find('[data-telegram-replace]').addEventListener('click', () => { replacing = true; clearPairingCommand(); token.value = ''; renderBot(); token.focus(); });
  find('[data-telegram-token-cancel]').addEventListener('click', () => { replacing = false; token.value = ''; renderBot(); find('[data-telegram-replace]').focus(); });
  find('[data-telegram-verify]').addEventListener('click', () => change({action: 'verify'}));
  enabled.addEventListener('change', () => change({action: 'enabled', enabled: enabled.checked}));
  find('[data-telegram-pair]').addEventListener('click', () => change({action: 'pair'}));
  find('[data-telegram-copy]').addEventListener('click', async () => {
    const command = pairingCommand;
    if (!command) return;
    let copied = false;
    try { if (navigator.clipboard?.writeText) { await navigator.clipboard.writeText(command); copied = true; } } catch (_) { /* LAN HTTP fallback below. */ }
    if (!copied) {
      const textarea = document.createElement('textarea'); textarea.value = command;
      textarea.setAttribute('readonly', ''); textarea.style.position = 'fixed'; textarea.style.opacity = '0';
      document.body.append(textarea); textarea.select();
      try { copied = document.execCommand('copy'); } catch (_) { /* Select the command below. */ }
      textarea.remove();
    }
    if (command !== pairingCommand || !active()) return;
    if (copied) { find('[data-telegram-copy]').textContent = jvT("ui.8d525e5f158b"); setTimeout(() => { find('[data-telegram-copy]').textContent = jvT("ui.e21f935f11d7"); }, 2000); }
    else {
      const range = document.createRange(); range.selectNodeContents(challenge);
      const selection = window.getSelection(); selection.removeAllRanges(); selection.addRange(range);
      feedback.textContent = jvT("ui.ce0d876e5715");
    }
  });
  find('[data-telegram-disconnect]').addEventListener('click', () => change({action: 'disconnect'}));
})();
