'use strict';
const {test} = require('node:test');
const assert = require('node:assert/strict');
const {readFileSync} = require('node:fs');
const vm = require('node:vm');
const base = `${__dirname}/../webui/internal/server/`;
const source = readFileSync(`${base}static/telegram.js`, 'utf8');
const header = readFileSync(`${base}templates/header.html`, 'utf8');
const markup = header.slice(header.indexOf('<section data-gateway-telegram'), header.indexOf('<section data-gateway-discord'));

class Node {
  constructor(tag = 'div', text = '') {
    this.tagName = tag; this.textContent = text; this.children = []; this.listeners = {};
    this.dataset = {}; this.hidden = false; this.value = ''; this.checked = false;
  }
  append(...children) { this.children.push(...children); }
  prepend(...children) { this.children.unshift(...children); }
  replaceChildren(...children) { this.children = children; }
  setAttribute(name, value) { this[name] = value; }
  addEventListener(name, listener) { this.listeners[name] = listener; }
  focus() { this.focused = true; }
  contains(target) { return target === this || this.children.some(child => child.contains(target)); }
}
const descendants = node => [node, ...node.children.flatMap(descendants)];
const control = (row, label) => descendants(row).find(node => node.tagName === 'button' && node.textContent === label);
const toggle = row => descendants(row).find(node => node.tagName === 'input');
const selected = row => control(row, 'Server Management')['aria-pressed'] === 'true';
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
  const dialog = new Node('dialog'); dialog.open = true; dialog.querySelector = node;
  node('content').dataset.telegramCsrf = 'test-csrf';
  let snapshot = {configured: true, verified: true, enabled: true, status: 'connected', bot_name: 'test_bot', users: [], ...initial};
  let poll;
  let fail = false;
  const requests = [];
  const context = vm.createContext({
    document: {querySelector: () => dialog, createElement: tag => new Node(tag), body: {classList: {contains: () => true}}},
    window: new Node(), URLSearchParams, setInterval(callback, delay) { assert.equal(delay, 5000); poll = callback; return 1; }, clearInterval() { poll = null; },
    async fetch(url, options) {
      assert.equal(url, '/api/telegram'); assert.equal(options.cache, 'no-store');
      if (options.method === 'POST') {
        assert.equal(options.body.get('csrf'), 'test-csrf');
        const payload = JSON.parse(options.body.get('request')); requests.push(payload);
        if (!fail) {
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
    node, dialog, requests, snapshot: () => snapshot, fail(value) { fail = value; },
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
  assert.equal(control(h.pending(), 'Read Only')['aria-pressed'], 'true');
  assert.equal(toggle(h.pending()).checked, false);
  control(h.pending(), 'Server Management').listeners.click();
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
  const edit = () => { control(h.pending(), 'Server Management').listeners.click(); toggle(h.pending()).checked = true; toggle(h.pending()).listeners.change(); };
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
  control(h.row(101), 'Server Management').listeners.click();
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
  for (const id of [101, 202]) { control(h.row(id), 'Server Management').listeners.click(); toggle(h.row(id)).checked = true; toggle(h.row(id)).listeners.change(); }
  await control(h.row(101), 'Revoke').listeners.click(); assert.equal(h.row(101), undefined);
  h.snapshot().users.push(user(101)); await h.refresh();
  assert.equal(selected(h.row(101)), false); assert.equal(toggle(h.row(101)).checked, false);
  assert.equal(selected(h.row(202)), true); assert.equal(toggle(h.row(202)).checked, true);
});

test('failed saves retain drafts for retry', async () => {
  const h = await harness({users: [user(101)], pairing: pairing(202)});
  control(h.row(101), 'Server Management').listeners.click(); control(h.pending(), 'Server Management').listeners.click();
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
  assert.deepEqual(h.requests.at(-1), {action: 'credential', token: 'test-only-token'});
  assert.equal(h.node('token').value, ''); assert.equal(h.node('token-form').hidden, true);
  await h.click('verify'); assert.deepEqual(h.requests.at(-1), {action: 'verify'});
});

test('credential replacement and Disconnect clear invalidated drafts', async () => {
  for (const action of ['credential', 'disconnect']) {
    const h = await harness({users: [user(101)], pairing: pairing(202)});
    control(h.row(101), 'Server Management').listeners.click(); control(h.pending(), 'Server Management').listeners.click();
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
    control(h.row(101), 'Server Management').listeners.click(); control(h.pending(), 'Server Management').listeners.click();
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
  assert.match(h.node('challenge').textContent, /Send once in a private chat/);
  assert.match(h.node('challenge').textContent, /Do not share/);
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
  assert.deepEqual(labels, ['Save', 'Cancel', 'Verify', 'Replace token', 'Disconnect', 'Pair user']);
  for (const label of ['Approve', 'Save', 'Revoke']) assert.ok(source.includes(`button('${label}'`));
  assert.match(markup, /type="password" name="token" aria-label="Bot token" autocomplete="off" data-password-ui="off" data-telegram-token required/);
  assert.doesNotMatch(markup.match(/<input[^>]*data-telegram-token[^>]*>/)[0], /value=/);
  assert.match(markup, /\/static\/ai-providers\/telegram\.svg/);
  assert.match(markup, /Status: <span data-telegram-operation>/);
  assert.match(source, /setAttribute\('aria-pressed'/); assert.match(source, /setAttribute\('role', 'switch'\)/);
  assert.doesNotMatch(source, /localStorage|sessionStorage|console\.|innerHTML|createElement\('select'\)/);
});
