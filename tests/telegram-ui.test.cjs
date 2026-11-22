'use strict';
const {test} = require('node:test');
const assert = require('node:assert/strict');
const {readFileSync} = require('./helpers/webui-i18n.cjs');
const vm = require('node:vm');
const base = `${__dirname}/../webui/internal/server/`;
const source = readFileSync(`${base}static/telegram.js`, 'utf8');
const header = readFileSync(`${base}templates/header.html`, 'utf8');
const markup = header.slice(header.indexOf('<section data-gateway-telegram'), header.indexOf('<section data-gateway-discord'));

class Node {
  constructor(tag = 'div', text = '') {
    this.tagName = tag; this.textContent = text; this.children = []; this.listeners = {};
    this.dataset = {}; this.style = {}; this.hidden = false; this.value = ''; this.checked = false;
  }
  append(...children) { this.children.push(...children); }
  prepend(...children) { this.children.unshift(...children); }
  replaceChildren(...children) { this.children = children; }
  setAttribute(name, value) { this[name] = value; }
  addEventListener(name, listener) { this.listeners[name] = listener; }
  select() { this.selected = true; }
  remove() { this.removed = true; }
  focus() { this.focused = true; }
  contains(target) { return target === this || this.children.some(child => child.contains(target)); }
}
const descendants = node => [node, ...node.children.flatMap(descendants)];
const control = (row, label) => descendants(row).find(node => node.tagName === 'button' && node.textContent === label);
const toggle = row => descendants(row).find(node => node.tagName === 'input');
const dropdown = row => descendants(row).find(node => node.tagName === 'select');
const chooseManagement = row => { const select = dropdown(row); select.value = 'server_management'; select.listeners.change(); };
const selected = row => dropdown(row).value === 'server_management';
const pairing = id => ({user_id: id, expires: '2099-01-01T00:00:00Z'});
const user = id => ({id, permission: 'read_only', notifications: false});
const settle = () => new Promise(resolve => setImmediate(resolve));

async function harness(initial = {}) {
  const nodes = new Map();
  const node = name => {
    const selector = name.startsWith('[') ? name : `[data-telegram-${name}]`;
    if (!nodes.has(selector)) nodes.set(selector, new Node());
    return nodes.get(selector);
  };
  const events = [];
  const dialog = new Node('dialog'); dialog.open = true; dialog.querySelector = node;
  dialog.dispatchEvent = event => { events.push(event); return true; };
  node('content').dataset.telegramCsrf = 'test-csrf';
  let snapshot = {configured: true, verified: true, enabled: true, status: 'connected', bot_name: 'test_bot', users: [], ...initial};
  let poll;
  let fail = false;
  let verifyFail = false;
  const copied = [];
  let clipboardMode = 'modern';
  let timerID = 0;
  const timers = new Map();
  const document = {querySelector: () => dialog, createElement: tag => new Node(tag), body: new Node('body'), execCommand() { if (clipboardMode === 'manual') return false; copied.push(document.body.children.at(-1).value); return true; }, createRange: () => ({selectNodeContents(node) { document.selection = node.textContent; }})};
  document.body.classList = {contains: () => true};
  const requests = [];
  const context = vm.createContext({
    CustomEvent: class { constructor(type, options) { this.type = type; this.detail = options.detail; } },
    document, navigator: {get clipboard() { return clipboardMode === 'unavailable' ? undefined : {async writeText(text) { if (clipboardMode !== 'modern') throw new Error('LAN HTTP'); copied.push(text); }}; }},
    window: Object.assign(new Node(), {getSelection: () => ({removeAllRanges() {}, addRange() {}})}), URLSearchParams, setTimeout(callback, delay) { timers.set(++timerID, {callback, delay}); return timerID; }, clearTimeout(id) { timers.delete(id); }, setInterval(callback, delay) { assert.equal(delay, 5000); poll = callback; return 1; }, clearInterval() { poll = null; },
    async fetch(url, options) {
      assert.equal(url, '/api/telegram'); assert.equal(options.cache, 'no-store');
      if (options.method === 'POST') {
        assert.equal(options.body.get('csrf'), 'test-csrf');
        const payload = JSON.parse(options.body.get('request')); requests.push(payload);
        if (payload.action === 'verify' && verifyFail) return {ok: false, async json() { return {error: 'Verification failed'}; }};
        if (!fail) {
          if (payload.action === 'enabled') snapshot = {...snapshot, enabled: payload.enabled, status: payload.enabled ? 'connected' : 'disabled'};
          if (payload.action === 'verify') snapshot = {...snapshot, verified: true, status: 'verified'};
          if (payload.action === 'approve') { snapshot.users.push({id: payload.user_id, permission: payload.permission, notifications: payload.notifications}); snapshot.pairing = null; }
          if (payload.action === 'user') snapshot.users = snapshot.users.map(item => item.id === payload.user_id ? {...item, permission: payload.permission, notifications: payload.notifications} : item);
          if (payload.action === 'revoke') snapshot.users = snapshot.users.filter(item => item.id !== payload.user_id);
          if (['credential', 'disconnect'].includes(payload.action)) snapshot = {...snapshot, configured: payload.action === 'credential', users: [], pairing: null};
        }
      }
      return {ok: !fail, redirected: false, async json() { return fail ? {error: 'Could not save'} : structuredClone(snapshot); }};
    },
  });
  vm.runInContext(source, context);
  dialog.listeners['gateway-surface-change'](); await settle();
  return {
    node, dialog, events, requests, document, copied, timers, clipboard(mode) { clipboardMode = mode; }, verifyFail(value) { verifyFail = value; }, snapshot: () => snapshot, fail(value) { fail = value; },
    pending: () => node('pending'), row: id => node('users').children.find(row => row.children[1].textContent === `User ${id}`),
    async refresh() { await dialog.listeners['telegram-refresh'](); },
    async poll() { assert.ok(poll); poll(); await settle(); },
    async click(name) { await node(name).listeners.click(); },
    async submit() { node('token-form').listeners.submit({preventDefault() {}}); await settle(); },
    async reopen() { node('[data-gateway-telegram]').hidden = false; dialog.open = true; dialog.listeners['gateway-surface-change'](); await settle(); },
  };
}

test('pending permission and Notifications survive polling and refresh; Approve sends visible drafts', async () => {
  const h = await harness({pairing: pairing(101)});
  assert.equal(dropdown(h.pending()).value, 'read_only');
  assert.deepEqual(dropdown(h.pending()).children.map(option => option.textContent), ['Read Only', 'Server Management']);
  assert.equal(toggle(h.pending()).checked, false);
  chooseManagement(h.pending());
  toggle(h.pending()).checked = true; toggle(h.pending()).listeners.change();
  await h.poll(); await h.refresh();
  assert.equal(selected(h.pending()), true); assert.equal(toggle(h.pending()).checked, true);
  await control(h.pending(), 'Approve').listeners.click();
  assert.deepEqual(h.requests.at(-1), {action: 'approve', user_id: 101, permission: 'server_management', notifications: true});
  assert.equal(h.pending().children.length, 0);
  // The same ID claiming a new challenge must receive a fresh draft.
  h.snapshot().pairing = pairing(101); await h.refresh();
  assert.equal(selected(h.pending()), false); assert.equal(toggle(h.pending()).checked, false);
});

test('pending drafts are bound to numeric ID and cleared on disappearance and expiry', async () => {
  const h = await harness({pairing: pairing(101)});
  const edit = () => { chooseManagement(h.pending()); toggle(h.pending()).checked = true; toggle(h.pending()).listeners.change(); };
  edit(); h.snapshot().pairing = pairing(202); await h.refresh();
  assert.equal(selected(h.pending()), false); assert.equal(toggle(h.pending()).checked, false);
  edit(); h.snapshot().pairing = null; await h.refresh();
  h.snapshot().pairing = pairing(202); await h.refresh();
  assert.equal(selected(h.pending()), false); assert.equal(toggle(h.pending()).checked, false);
  edit(); h.snapshot().pairing.expires = '2000-01-01T00:00:00Z'; await h.refresh();
  assert.equal(h.pending().children.length, 0);
  h.snapshot().pairing = pairing(202); await h.refresh(); assert.equal(selected(h.pending()), false);
});

test('approved-user drafts survive refresh, follow immutable IDs, and Save synchronizes state', async () => {
  const h = await harness({users: [user(101), user(202)]});
  chooseManagement(h.row(101));
  toggle(h.row(101)).checked = true; toggle(h.row(101)).listeners.change();
  h.snapshot().users.reverse(); await h.poll(); await h.refresh();
  assert.equal(selected(h.row(101)), true); assert.equal(toggle(h.row(101)).checked, true);
  assert.equal(selected(h.row(202)), false); assert.equal(toggle(h.row(202)).checked, false);
  await control(h.row(101), 'Save').listeners.click();
  assert.deepEqual(h.requests.at(-1), {action: 'user', user_id: 101, permission: 'server_management', notifications: true});
  assert.equal(selected(h.row(101)), true);
  // Once saved, later authoritative changes are no longer shadowed by the old draft.
  Object.assign(h.snapshot().users.find(item => item.id === 101), user(101)); await h.refresh();
  assert.equal(selected(h.row(101)), false); assert.equal(toggle(h.row(101)).checked, false);
});

test('Revoke removes only that user draft; reappearing ID has authoritative defaults', async () => {
  const h = await harness({users: [user(101), user(202)]});
  for (const id of [101, 202]) { chooseManagement(h.row(id)); toggle(h.row(id)).checked = true; toggle(h.row(id)).listeners.change(); }
  await control(h.row(101), 'Revoke').listeners.click(); assert.equal(h.row(101), undefined);
  h.snapshot().users.push(user(101)); await h.refresh();
  assert.equal(selected(h.row(101)), false); assert.equal(toggle(h.row(101)).checked, false);
  assert.equal(selected(h.row(202)), true); assert.equal(toggle(h.row(202)).checked, true);
});

test('failed saves retain drafts for retry', async () => {
  const h = await harness({users: [user(101)], pairing: pairing(202)});
  chooseManagement(h.row(101)); chooseManagement(h.pending());
  h.fail(true); await control(h.row(101), 'Save').listeners.click(); await control(h.pending(), 'Approve').listeners.click();
  h.fail(false); await h.refresh(); assert.equal(selected(h.row(101)), true); assert.equal(selected(h.pending()), true);
});

test('configured Bot lane reveals a replacement only on demand; Cancel clears it', async () => {
  const h = await harness();
  assert.equal(h.node('token-form').hidden, true); assert.equal(h.node('bot-actions').hidden, false);
  await h.click('replace'); assert.equal(h.node('token-form').hidden, false); assert.equal(h.node('bot-actions').hidden, true);
  assert.equal(h.node('token-cancel').hidden, false);
  h.node('token').value = 'typed-test-replacement'; await h.refresh();
  assert.equal(h.node('token').value, 'typed-test-replacement'); assert.equal(h.node('token-form').hidden, false);
  await h.click('token-cancel');
  assert.equal(h.node('token').value, ''); assert.equal(h.node('token-form').hidden, true); assert.equal(h.node('bot-actions').hidden, false);
});

test('unconfigured Bot lane shows only token and Save; credential actions retain existing contract', async () => {
  const h = await harness({configured: false, verified: false, enabled: false});
  assert.equal(h.node('token-form').hidden, false); assert.equal(h.node('token-cancel').hidden, true); assert.equal(h.node('bot-actions').hidden, true);
  h.node('token').value = 'test-only-token'; await h.submit();
  assert.deepEqual(h.requests, [{action: 'credential', token: 'test-only-token'}, {action: 'verify'}]);
  assert.equal(h.node('token').value, ''); assert.equal(h.node('token-form').hidden, true);
  assert.equal(h.node('verify').hidden, true);
  assert.equal(h.requests.some(request => request.action === 'enabled'), false);
});

test('credential replacement and Disconnect clear invalidated drafts', async () => {
  for (const action of ['credential', 'disconnect']) {
    const h = await harness({users: [user(101)], pairing: pairing(202)});
    chooseManagement(h.row(101)); chooseManagement(h.pending());
    if (action === 'credential') { await h.click('replace'); h.node('token').value = 'test-only-replacement'; await h.submit(); }
    else await h.click('disconnect');
    assert.equal(h.node('users').children.length, 0); assert.equal(h.pending().children.length, 0);
    h.snapshot().users = [user(101)]; h.snapshot().pairing = pairing(202); await h.refresh();
    assert.equal(selected(h.row(101)), false); assert.equal(selected(h.pending()), false);
  }
});

test('leaving or closing Telegram clears drafts, typed tokens, and pairing challenges', async () => {
  for (const event of ['close', 'cancel', 'gateway-surface-change']) {
    const h = await harness({users: [user(101)], pairing: pairing(202)});
    chooseManagement(h.row(101)); chooseManagement(h.pending());
    await h.click('replace'); h.node('token').value = 'test-only-replacement';
    if (event === 'gateway-surface-change') h.node('[data-gateway-telegram]').hidden = true;
    else h.dialog.open = false;
    h.dialog.listeners[event]();
    assert.equal(h.node('token').value, ''); assert.equal(h.node('challenge').textContent, '');
    await h.reopen();
    assert.equal(selected(h.row(101)), false); assert.equal(selected(h.pending()), false); assert.equal(h.node('token-form').hidden, true);
  }
});

test('one-time pairing challenge survives refresh only while awaiting a claim', async () => {
  const h = await harness({pairing: pairing(0), pairing_command: '/start test-only-challenge'});
  await h.click('pair');
  assert.equal(h.node('challenge').textContent, '/start test-only-challenge');
  await h.click('copy'); assert.deepEqual(h.copied, ['/start test-only-challenge']);
  delete h.snapshot().pairing_command; await h.refresh();
  assert.match(h.node('challenge').textContent, /test-only-challenge/);
  h.snapshot().pairing = pairing(101); await h.refresh();
  assert.equal(h.node('challenge').textContent, '');
  assert.ok(control(h.pending(), 'Approve'));
});

test('status snapshots never redisplay a stored token', async () => {
  const h = await harness({token: 'must-never-be-displayed'});
  await h.refresh(); await h.click('replace');
  assert.equal(h.node('token').value, '');
  for (const name of ['status', 'feedback', 'challenge', 'operation']) assert.equal(h.node(name).textContent.includes('must-never-be-displayed'), false);
  assert.equal(h.requests.length, 0);
});

test('compact template, exact action labels, accessible choices, and secret opt-out', () => {
  for (const obsolete of ['data-telegram-back', 'Back to Add', 'Bot connection', 'Approved Telegram users', 'Operation status', 'class="panel"', 'Gateway switch']) assert.equal(markup.includes(obsolete), false);
  assert.equal(source.includes('data-telegram-back'), false);
  const gateway = readFileSync(`${base}static/agent-gateway.js`, 'utf8'); assert.equal(gateway.includes('data-telegram-back'), false);
  const labels = [...markup.matchAll(/<button\b[^>]*>([^<]+)<\/button>/g)].map(match => match[1]);
  assert.deepEqual(labels, ['Save', 'Cancel', 'Retry', 'Replace token', 'Disconnect', 'Pair user', 'Copy']);
  for (const label of ['Approve', 'Save', 'Revoke']) assert.ok(source.includes(`button('${label}'`));
  assert.match(markup, /type="password" name="token" aria-label="Bot Token" autocomplete="off" data-password-ui="off" spellcheck="false" autocapitalize="off" data-telegram-token required/);
  assert.doesNotMatch(markup.match(/<input[^>]*data-telegram-token[^>]*>/)[0], /value=/);
  assert.match(markup, /\/static\/ai-providers\/telegram\.svg/);
  assert.match(markup, /Status: <span data-telegram-operation>/);
  assert.match(source, /createElement\('select'\)/); assert.match(source, /setAttribute\('role', 'switch'\)/);
  assert.doesNotMatch(source, /localStorage|sessionStorage|console\.|innerHTML/);
});


test('verification failure keeps saved configuration with Retry and replacement; retry verifies', async () => {
  const h = await harness({configured: false, verified: false, enabled: false, status: 'disconnected'});
  h.verifyFail(true); h.node('token').value = 'test-only-token'; await h.submit();
  assert.equal(h.node('token').value, '');
  assert.equal(h.node('token-form').hidden, true);
  assert.equal(h.node('verify').hidden, false);
  assert.equal(h.node('feedback').textContent, 'Token saved, but verification failed. Check the token or connection.');
  await h.poll(); assert.match(h.node('feedback').textContent, /Token saved/);
  h.verifyFail(false); await h.click('verify');
  assert.equal(h.node('verify').hidden, true);
});

test('polling with focused Telegram controls claims pairing and preserves typed replacement', async () => {
  for (const name of ['pair', 'pending', 'enabled', 'token', 'replace']) {
    const h = await harness({pairing: pairing(0), pairing_command: '/start test-only-challenge'});
    await h.click('pair'); h.document.activeElement = h.node(name);
    h.node('content').append(h.node(name));
    h.snapshot().pairing = pairing(609038312); delete h.snapshot().pairing_command;
    await h.poll();
    assert.equal(h.node('challenge').textContent, '');
    assert.match(h.pending().children[0].textContent, /User 609038312/);
    chooseManagement(h.pending()); await h.poll(); assert.equal(selected(h.pending()), true);
    await h.click('replace'); h.node('token').value = 'typed-replacement'; await h.poll();
    assert.equal(h.node('token').value, 'typed-replacement');
  }
});

test('LAN clipboard fallback copies exactly, or selects the command for manual copying', async () => {
  for (const mode of ['fallback', 'unavailable', 'manual']) {
    const h = await harness({pairing: pairing(0), pairing_command: '/start test-only-challenge'});
    await h.click('pair'); h.clipboard(mode); await h.click('copy');
    if (mode !== 'manual') assert.deepEqual(h.copied, ['/start test-only-challenge']);
    else { assert.equal(h.document.selection, '/start test-only-challenge'); assert.match(h.node('feedback').textContent, /Ctrl\+C/); }
    assert.equal(h.document.body.children.at(-1).removed, true);
  }
});

test('header zones and scoped reveal suppression leave other passwords alone', async () => {
  const css = readFileSync(`${base}static/agent-gateway.css`, 'utf8');
  assert.match(markup, /integration-header/); assert.match(markup, /integration-title/);
  assert.match(css, /:is\(\[data-gateway-telegram\], \[data-gateway-discord\]\) \.integration-header.*grid-template-columns: 1fr auto/);
  for (const integration of ['telegram', 'discord']) {
    for (const pseudo of ['::-ms-reveal', '::-ms-clear', '::-webkit-credentials-auto-fill-button']) {
      assert.ok(css.includes(`[data-gateway-${integration}] [data-${integration}-token]${pseudo}`));
    }
  }
  for (const line of css.split('\n').filter(line => /::-ms-reveal|::-ms-clear|::-webkit-credentials-auto-fill-button/.test(line))) {
    for (const selector of line.split('{')[0].split(',').map(selector => selector.trim()).filter(Boolean)) {
      assert.match(selector, /^\[data-gateway-(telegram|discord)\] \[data-\1-token\]::-(?:ms-reveal|ms-clear|webkit-credentials-auto-fill-button)$/);
    }
  }
  assert.match(markup, /<strong>Bot Token<\/strong>/); assert.doesNotMatch(markup, /<strong>Bot<\/strong>/);
  for (const status of ['verified', 'connected']) {
    const h = await harness({status}); assert.match(h.node('status').className, /success/);
  }
});


test('challenge is cleared by expiry, changed pairing, replacement, disconnect, and surface exit', async () => {
  for (const reason of ['expiry', 'changed-pairing', 'replace', 'disconnect', 'close']) {
    const h = await harness({pairing: pairing(0), pairing_command: '/start test-only-challenge'});
    await h.click('pair');
    assert.equal(h.node('challenge').textContent, '/start test-only-challenge');
    if (reason === 'expiry') [...h.timers.values()].find(timer => timer.delay > 2000).callback();
    else if (reason === 'changed-pairing') { h.snapshot().pairing = {...pairing(0), expires: '2099-01-02T00:00:00Z'}; await h.poll(); }
    else if (reason === 'close') { h.dialog.open = false; h.dialog.listeners.close(); }
    else await h.click(reason);
    assert.equal(h.node('challenge').textContent, '');
    assert.equal(h.node('copy').hidden, true);
  }
});

test('Telegram has exactly one enable control in the workspace navigation', () => {
  const lane = header.match(/<nav class="minecraft-workspace-tabs gateway-navigation"[^>]*>([\s\S]*?)<\/nav>/)[1];
  assert.match(lane, /<span>AI &amp; API Gateway<\/span>/);
  assert.doesNotMatch(lane, /<span>Gateway<\/span>/);
  assert.match(lane, /data-telegram-global hidden><img[^>]*src="\/static\/ai-providers\/telegram.svg"[^>]*><span>Telegram<\/span>/);
  assert.equal((header.match(/data-telegram-enabled/g) || []).length, 1);
  assert.doesNotMatch(markup, /data-telegram-enabled|Enable Telegram/);
});

test('workspace Telegram header follows authoritative configuration, verification and enabled state', async () => {
  for (const configured of [false, true]) for (const verified of [false, true]) for (const enabled of [false, true]) {
    const h = await harness({configured, verified, enabled});
    assert.equal(h.node('global').hidden, !configured);
    assert.equal(h.node('enabled').disabled, !verified);
    assert.equal(h.node('enabled').checked, enabled);
  }
});

test('Telegram header mutates on Connections, Add and Help without changing Gateway', async () => {
  for (const surface of ['overview', 'settings', 'help']) {
    const h = await harness();
    h.node('[data-gateway-telegram]').hidden = true;
    h.dialog.listeners['gateway-surface-change']({detail: surface}); await settle();
    const gateway = h.node('[data-gateway-enabled]'); gateway.checked = false;
    h.node('enabled').checked = false;
    await h.node('enabled').listeners.change();
    assert.deepEqual(h.requests.at(-1), {action: 'enabled', enabled: false});
    assert.equal(h.node('enabled').checked, false);
    assert.equal(h.events.at(-1).detail.enabled, false);
    assert.equal(h.events.at(-1).detail.status, 'disabled');
    assert.equal(gateway.checked, false);
    h.node('enabled').checked = true;
    await h.node('enabled').listeners.change();
    assert.equal(h.node('enabled').checked, true);
    assert.equal(gateway.checked, false);
    h.fail(true); h.node('enabled').checked = false;
    await h.node('enabled').listeners.change();
    assert.equal(h.node('enabled').checked, true);
    assert.equal(h.node('[data-gateway-feedback]').textContent, 'Could not save');
    assert.equal(gateway.checked, false);
  }
});

test('workspace refresh publishes only safe Telegram presentation state while settings are hidden', async () => {
  const h = await harness({token: 'private', pairing: pairing(101), pairing_command: '/start private', users: [user(101)], callback_data: 'private'});
  h.node('[data-gateway-telegram]').hidden = true;
  await h.refresh();
  const event = h.events.at(-1);
  assert.equal(event.type, 'telegram-state');
  assert.deepEqual(JSON.parse(JSON.stringify(event.detail)), {configured: true, verified: true, enabled: true, status: 'connected', bot_name: 'test_bot'});
  assert.deepEqual(Object.keys(event.detail).sort(), ['bot_name', 'configured', 'enabled', 'status', 'verified']);
  assert.doesNotMatch(JSON.stringify(event.detail), /private|pairing|users|callback|token/);
});


test('Telegram generated user rows retain the shared compact lane', async () => {
  const h = await harness({users: [user(12345)]});
  assert.equal(h.row(12345).className, 'integration-lane');
});
