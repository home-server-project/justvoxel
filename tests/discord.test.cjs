'use strict';
const {test} = require('node:test');
const assert = require('node:assert/strict');
const {readFileSync} = require('node:fs');
const vm = require('node:vm');
const source = readFileSync(`${__dirname}/../webui/internal/server/static/discord.js`, 'utf8');
const header = readFileSync(`${__dirname}/../webui/internal/server/templates/header.html`, 'utf8');
const flush = () => new Promise(resolve => setImmediate(resolve));

function harness(primary = true, initiallyOpen = false) {
  class Element {
    constructor(tag = 'div') { this.tagName = tag; this.value = ''; this.textContent = ''; this.hidden = false; this.children = []; this.listeners = {}; this.dataset = {}; }
    focus() { this.focused = true; }
    addEventListener(type, callback) { this.listeners[type] = callback; }
    replaceChildren(...children) { this.children = children; }
    append(...children) { this.children.push(...children); }
  }
  const nodes = new Map();
  const find = selector => {
    if (!nodes.has(selector)) nodes.set(selector, new Element());
    return nodes.get(selector);
  };
  find('[data-discord-content]').dataset.discordCsrf = 'browser-csrf';
  const surface = new Element(); surface.hidden = true; surface.querySelector = find;
  const dialog = new Element(); dialog.open = initiallyOpen; dialog.querySelector = selector => selector === '[data-gateway-discord]' ? surface : find(selector);
  const events = [];
  dialog.dispatchEvent = event => { events.push(event); };
  const window = new Element();
  const calls = [];
  const replies = [];
  const context = vm.createContext({
    URLSearchParams, window, CustomEvent: class { constructor(type, options) { this.type = type; this.detail = options.detail; } },
    document: {querySelector: () => dialog, createElement: tag => new Element(tag), body: {classList: {contains: () => primary}}},
    fetch: async (url, options) => {
      calls.push({url, options});
      const reply = replies.shift() || {configured: false, verified: false, status: 'disconnected'};
      if (reply instanceof Error) throw reply;
      if (typeof reply === 'function') return reply();
      return {ok: true, redirected: false, json: async () => reply};
    },
  });
  vm.runInContext(source, context);
  const dispatch = type => dialog.listeners[type]();
  return {find, surface, dialog, window, calls, replies, events, dispatch,
    open() { dialog.open = true; surface.hidden = false; dispatch('gateway-surface-change'); },
    submit() { find('[data-discord-token-form]').listeners.submit({preventDefault() {}}); },
    click(control) { find(`[data-discord-${control}]`).listeners.click(); },
  };
}

const verified = {configured: true, verified: true, status: 'verified', bot_name: 'Voxel Bot', bot_username: 'voxel_bot', application_name: 'Voxel'};

test('Discord loads on selection and Refresh using only its authenticated endpoint', async () => {
  const h = harness();
  assert.equal(h.calls.length, 0);
  h.open(); await flush();
  h.dispatch('discord-refresh'); await flush();
  assert.equal(h.calls.length, 2);
  for (const {url, options} of h.calls) {
    assert.equal(url, '/api/discord');
    assert.equal(options.credentials, 'same-origin');
    assert.equal(options.cache, 'no-store');
    assert.equal(options.headers.Accept, 'application/json');
    assert.equal(options.method, undefined);
    assert.equal(options.body, undefined);
  }
  assert.equal(h.find('[data-discord-status]').textContent, '● Disconnected');
  const denied = harness(false); denied.open(); await flush();
  assert.equal(denied.calls.length, 0);
  h.surface.hidden = true; h.dispatch('gateway-surface-change');
  h.dispatch('discord-refresh'); await flush();
  assert.equal(h.calls.length, 4);
});

test('credential, verify and disconnect use form transport and never redisplay a token', async () => {
  const h = harness(); h.open(); await flush();
  h.find('[data-discord-token]').value = 'private-bot-token';
  h.replies.push({configured: true, verified: false, status: 'unverified', token: 'must-not-redisplay'});
  h.submit();
  assert.equal(h.find('[data-discord-token]').value, '');
  await flush();
  const save = h.calls.at(-1);
  assert.equal(save.url, '/api/discord');
  assert.equal(save.options.method, 'POST');
  assert.equal(save.options.body.get('csrf'), 'browser-csrf');
  assert.deepEqual(JSON.parse(save.options.body.get('request')), {action: 'credential', token: 'private-bot-token'});
  assert.equal(h.find('[data-discord-token]').value, '');
  assert.equal(h.find('[data-discord-status]').textContent, '● Needs verification');
  assert.equal(h.find('[data-discord-verify]').disabled, false);
  h.replies.push(verified); h.click('verify'); await flush();
  assert.deepEqual(JSON.parse(h.calls.at(-1).options.body.get('request')), {action: 'verify'});
  assert.equal(h.find('[data-discord-status]').textContent, '@voxel_bot  ● Disabled');
  assert.equal(h.find('[data-discord-bot-actions]').hidden, false);
  assert.equal(h.find('[data-discord-token-form]').hidden, true);
  h.click('disconnect'); await flush();
  assert.deepEqual(JSON.parse(h.calls.at(-1).options.body.get('request')), {action: 'disconnect'});
  assert.equal(h.find('[data-discord-status]').textContent, '● Disconnected');
  assert.equal(h.find('[data-discord-verify]').disabled, true);
});

test('unknown or malformed state hides stale identity and uses the unavailable label', async () => {
  const h = harness();
  h.replies.push(verified); h.open(); await flush();
  for (const snapshot of [null, {}, [], {status: 'constructor'}, {status: 'connected'}, {status: {}}]) {
    h.replies.push(snapshot === null ? () => ({ok: true, json: async () => null}) : snapshot);
    h.dispatch('discord-refresh'); await flush();
    assert.equal(h.find('[data-discord-status]').textContent, '● Status unavailable');
    assert.equal(h.find('[data-discord-global]').hidden, true);
  }
});

test('identity is text and token clears whenever the surface closes or changes away', async () => {
  const h = harness();
  h.replies.push({...verified, bot_username: '<img src=x onerror=bad()>', application_name: '<script>bad()</script>'});
  h.open(); await flush();
  assert.equal(h.find('[data-discord-status]').textContent, '@<img src=x onerror=bad()>  ● Disabled');
  for (const event of ['close', 'cancel', 'gateway-surface-change']) {
    h.find('[data-discord-token]').value = 'unsaved-secret';
    h.surface.hidden = true; h.dispatch(event);
    assert.equal(h.find('[data-discord-token]').value, '');
  }
  h.find('[data-discord-token]').value = 'unsaved-secret';
  h.window.listeners.pagehide();
  assert.equal(h.find('[data-discord-token]').value, '');
});

test('errors are fixed and token is cleared after failed submission', async () => {
  const h = harness(); h.open(); await flush();
  h.replies.push(new Error('raw diagnostic private-bot-token'));
  h.find('[data-discord-token]').value = 'private-bot-token'; h.submit(); await flush();
  assert.equal(h.find('[data-discord-token]').value, '');
  assert.equal(h.find('[data-discord-status]').textContent, '● Disconnected');
  assert.doesNotMatch(h.find('[data-discord-feedback]').textContent, /raw diagnostic|private-bot-token/);
  h.replies.push(() => ({ok: false, json: async () => ({error: 'raw diagnostic private-bot-token'})}));
  h.dispatch('discord-refresh'); await flush();
  assert.doesNotMatch(h.find('[data-discord-feedback]').textContent, /raw diagnostic|private-bot-token/);
});

test('a response after closing cannot restore stale identity', async () => {
  const h = harness();
  let finish;
  h.replies.push(() => new Promise(resolve => { finish = resolve; }));
  h.open();
  h.dialog.open = false; h.dispatch('close');
  const count = h.events.length;
  finish({ok: true, json: async () => verified}); await flush();
  assert.equal(h.events.length, count);
  assert.equal(h.events.at(-1).detail.configured, false);
  h.dialog.open = true; h.open(); await flush();
  assert.equal(h.find('[data-discord-status]').textContent, '● Disconnected');
});

test('Discord uses compact lanes and one independent global switch', () => {
  assert.doesNotMatch(source, /localStorage|sessionStorage|console\.|innerHTML|insertAdjacentHTML|setInterval|setTimeout|\.style\.|cssText|insertRule|<style/);
  assert.deepEqual([...source.matchAll(/action: '([^']+)'/g)].map(match => match[1]), ['credential', 'verify', 'enabled', 'guild', 'disconnect']);
  assert.match(header, /<script src="\/static\/discord.js" defer><\/script>/);
  assert.match(header, /type="password" name="token" aria-label="Bot Token" autocomplete="off" data-password-ui="off" spellcheck="false" autocapitalize="off" data-discord-token/);
  const surface = header.slice(header.indexOf('<section data-gateway-discord'), header.indexOf('<section data-gateway-help'));
  assert.match(surface, /class="integration-lane integration-header"/);
  assert.match(surface, /<strong>Bot Token<\/strong>/);
  assert.match(surface, /<strong>Discord server<\/strong>/);
  assert.doesNotMatch(surface, /class="panel"|data-discord-back|Back to Add|data-discord-enabled|Bot connection|Discord administrators|features will be added/);
  assert.equal((header.match(/data-discord-enabled/g) || []).length, 1);
  const nav = header.slice(header.indexOf('<nav class="minecraft-workspace-tabs gateway-navigation"'), header.indexOf('</nav>', header.indexOf('<nav class="minecraft-workspace-tabs gateway-navigation"')));
  assert.match(nav, /data-telegram-global[\s\S]*data-discord-global hidden[\s\S]*data-discord-enabled disabled/);
  assert.doesNotMatch(surface, /style=|<style|data-discord-pair|data-discord-channel|notifications/);
  const icon = readFileSync(`${__dirname}/../webui/internal/server/static/ai-providers/discord.svg`, 'utf8');
  assert.match(icon, /<svg/);
  assert.doesNotMatch(icon, /<script|<style|<animate|<foreignObject|\bhref=|\bsrc=|\bon\w+=|<text|font/);
});

test('enable and guild binding are explicit and readiness gated', async () => {
  const h = harness(); h.open(); await flush();
  const enabled = h.find('[data-discord-enabled]');
  const guild = h.find('[data-discord-guild]');
  const bind = h.find('[data-discord-bind]');
  assert.equal(enabled.disabled, true);
  const waiting = {...verified, application_id: '456', enabled: true, runtime_status: 'connecting', guilds_ready: false, guilds: []};
  h.replies.push(waiting);
  h.click('verify'); await flush();
  assert.equal(enabled.disabled, false);
  assert.equal(guild.disabled, true);
  assert.equal(bind.disabled, true);
  assert.equal(h.find('[data-discord-status]').textContent, '@voxel_bot  ● Connecting');
  const ready = {...waiting, runtime_status: 'connected', guilds_ready: true, guilds: [{id: '111', name: '<b>My server</b>'}]};
  h.replies.push(ready); h.dispatch('discord-refresh'); await flush();
  assert.equal(guild.disabled, false);
  assert.equal(bind.disabled, true);
  assert.equal(guild.children[1].textContent, '<b>My server</b>');
  const count = h.calls.length;
  guild.value = '111'; guild.listeners.change();
  assert.equal(bind.disabled, false);
  assert.equal(h.calls.length, count);
  h.replies.push({...ready, guild_id: '111', guild_name: '<b>My server</b>', bound_guild_available: true});
  h.click('bind'); await flush();
  assert.deepEqual(JSON.parse(h.calls.at(-1).options.body.get('request')), {action: 'guild', guild_id: '111'});
  assert.equal(h.find('[data-discord-bound]').textContent, '<b>My server</b>');
  h.replies.push({...ready, enabled: false, runtime_status: 'disabled', guilds_ready: false, guilds: []});
  enabled.checked = false; enabled.listeners.change(); await flush();
  assert.deepEqual(JSON.parse(h.calls.at(-1).options.body.get('request')), {action: 'enabled', enabled: false});
  assert.equal(guild.disabled, true);
  assert.equal(bind.disabled, true);
});

test('install link has a fixed origin and minimum bot permissions', async () => {
  const h = harness();
  h.replies.push({...verified, application_id: '456', install_url: 'https://untrusted.example/'});
  h.open(); await flush();
  const install = h.find('[data-discord-install]');
  const url = new URL(install.href);
  assert.equal(install.hidden, false);
  assert.equal(url.origin, 'https://discord.com');
  assert.equal(url.pathname, '/oauth2/authorize');
  assert.deepEqual(Object.fromEntries(url.searchParams), {client_id: '456', scope: 'bot', permissions: '3072'});
  assert.match(header, /data-discord-install hidden target="_blank" rel="noopener noreferrer"/);
  h.replies.push({...verified, application_id: 'https://untrusted.example/'});
  h.dispatch('discord-refresh'); await flush();
  assert.equal(install.hidden, true);
});

test('reports empty discovery and lost bound access without changing binding', async () => {
  const h = harness();
  const empty = {...verified, enabled: true, runtime_status: 'connected', guilds_ready: true, guilds: [], guild_id: '111', guild_name: 'Saved server', bound_guild_available: false};
  h.replies.push(empty); h.open(); await flush();
  assert.equal(h.find('[data-discord-empty]').hidden, false);
  assert.equal(h.find('[data-discord-bound-unavailable]').hidden, false);
  assert.equal(h.find('[data-discord-bound]').textContent, 'Saved server');
  assert.equal(h.find('[data-discord-guild]').disabled, true);
  assert.equal(h.find('[data-discord-bind]').disabled, true);
  assert.equal(h.calls.length, 1);
  h.replies.push({...empty, runtime_status: 'connection_paused', guilds_ready: false});
  h.dispatch('discord-refresh'); await flush();
  assert.equal(h.find('[data-discord-status]').textContent, '@voxel_bot  ● Needs attention');
});

test('Discord administrator approval stays in the existing provider and submits only fixed action and user ID', async () => {
  for (const [pending, label, action] of [[true, 'Approve', 'admin_approve'], [true, 'Reject', 'admin_reject'], [false, 'Remove', 'admin_remove']]) {
    const h = harness();
    const identity = {user_id: '777', name: '<img src=x onerror=bad()>', username: 'friendly'};
    h.replies.push({...verified, pending: pending ? [identity] : [], administrators: pending ? [] : [identity]});
    h.open(); await flush();
    const list = h.find(pending ? '[data-discord-pending]' : '[data-discord-administrators]');
    assert.equal(list.children.length, 1);
    const row = list.children[0];
    assert.equal(row.className, 'integration-lane');
    assert.equal(row.children[0].textContent, 'Access');
    assert.equal(row.children[1].textContent, identity.name);
    assert.equal(row.children[2].textContent, 'User 777');
    assert.equal(row.children[2].className, 'muted');
    assert.deepEqual(row.children.slice(3).map(button => button.tagName), pending ? ['button', 'button'] : ['button']);
    const button = row.children.find(button => button.textContent === label);
    assert.equal(button.className, 'button');
    button.listeners.click(); await flush();
    const payload = JSON.parse(h.calls[1].options.body.get('request'));
    assert.deepEqual(payload, {action, user_id: '777'});
    assert.equal(h.calls[1].url, '/api/discord');
  }
  const surface = header.slice(header.indexOf('<section data-gateway-discord'), header.indexOf('<section data-gateway-help'));
  assert.match(surface, /data-discord-pending/);
  assert.match(surface, /data-discord-administrators/);
  assert.doesNotMatch(surface, /Pending requests|Approved administrators|class="panel"/);
  assert.doesNotMatch(surface, /<style|style=|<dialog|sidebar|workspace/);
});

test('Discord administrator rows are bounded and cleared after connection failure', async () => {
  const h = harness();
  h.replies.push({...verified, pending: Array.from({length: 40}, (_, index) => ({user_id: String(index + 1), name: 'Person'})), administrators: [{user_id: 'invalid', name: 'Rejected'}]});
  h.open(); await flush();
  assert.equal(h.find('[data-discord-pending]').children.length, 32);
  assert.equal(h.find('[data-discord-administrators]').children.length, 0);
  h.replies.push(new Error('private failure'));
  h.dispatch('discord-refresh'); await flush();
  assert.equal(h.find('[data-discord-pending]').children.length, 0);
  assert.equal(h.find('[data-discord-administrators]').children.length, 0);
});

test('global enable is authoritative and independent on Connections, Add, Help and Discord', async () => {
  for (const surface of ['connections', 'add', 'help', 'discord']) {
    const h = harness();
    const enabled = h.find('[data-discord-enabled]');
    assert.equal(h.find('[data-discord-global]').hidden, true);
    assert.equal(enabled.disabled, true);
    h.dialog.open = true; h.surface.hidden = surface !== 'discord';
    h.replies.push({configured: true, verified: false, enabled: false, status: 'unverified'});
    h.dispatch('gateway-surface-change'); await flush();
    assert.equal(h.find('[data-discord-global]').hidden, false);
    assert.equal(enabled.disabled, true);
    h.replies.push({...verified, enabled: false}); h.dispatch('discord-refresh'); await flush();
    assert.equal(enabled.disabled, false); assert.equal(enabled.checked, false);
    h.replies.push({...verified, enabled: true, runtime_status: 'connecting'});
    enabled.checked = true; enabled.listeners.change();
    assert.equal(enabled.checked, false, 'no stale optimistic state');
    await flush();
    assert.equal(enabled.checked, true);
    assert.deepEqual(JSON.parse(h.calls.at(-1).options.body.get('request')), {action: 'enabled', enabled: true});
    h.replies.push(new Error('private failure'));
    enabled.checked = false; enabled.listeners.change(); await flush();
    assert.equal(enabled.checked, true); assert.equal(enabled.disabled, false);
    assert.doesNotMatch(h.find('[data-discord-feedback]').textContent, /private failure/);
    assert.ok(h.calls.every(call => call.url === '/api/discord'));
  }
});

test('presentation events contain only safe allowlisted identity and human status inputs', async () => {
  const h = harness();
  h.replies.push({...verified, enabled: true, runtime_status: 'connected', token: 'secret', interaction_token: 'secret', error: 'private error', application_id: '456'});
  h.open(); await flush();
  const event = h.events.at(-1);
  assert.equal(event.type, 'discord-state');
  assert.deepEqual(JSON.parse(JSON.stringify(event.detail)), {configured: true, verified: true, enabled: true, status: 'connected', bot_username: 'voxel_bot'});
  assert.doesNotMatch(JSON.stringify(event.detail), /secret|private error|application_id/);
  const cases = [
    [{...verified, verified: false, enabled: true}, 'Needs verification'],
    [{...verified, enabled: false, runtime_status: 'connected'}, 'Disabled'],
    [{...verified, enabled: true}, 'Bot verified'],
    ...Object.entries({connecting: 'Connecting', connected: 'Connected', connection_retry: 'Retrying', connection_paused: 'Needs attention', disconnected: 'Disconnected', unknown: 'Status unavailable'}).map(([runtime_status, label]) => [{...verified, enabled: true, runtime_status}, label]),
  ];
  for (const [snapshot, label] of cases) {
    h.replies.push(snapshot); h.dispatch('discord-refresh'); await flush();
    assert.ok(h.find('[data-discord-status]').textContent.endsWith(`● ${label}`));
  }
});

test('Replace token and Cancel never redisplay credentials and leaving discards drafts', async () => {
  const h = harness(); h.replies.push(verified); h.open(); await flush();
  assert.equal(h.find('[data-discord-token-form]').hidden, true);
  assert.equal(h.find('[data-discord-verify]').hidden, true);
  h.click('replace');
  assert.equal(h.find('[data-discord-token-form]').hidden, false);
  assert.equal(h.find('[data-discord-token-cancel]').hidden, false);
  assert.equal(h.find('[data-discord-token]').focused, true);
  h.find('[data-discord-token]').value = 'draft'; h.click('token-cancel');
  assert.equal(h.find('[data-discord-token]').value, '');
  assert.equal(h.find('[data-discord-token-form]').hidden, true);
  h.click('replace'); h.find('[data-discord-token]').value = 'draft';
  h.replies.push(verified); h.surface.hidden = true; h.dispatch('gateway-surface-change'); await flush();
  assert.equal(h.find('[data-discord-token]').value, '');
  h.replies.push(verified); h.open(); await flush();
  assert.equal(h.find('[data-discord-token-form]').hidden, true);
  assert.equal(h.calls.filter(call => call.options.method === 'POST').length, 0);
});

test('leaving a surface fences a pending credential response and refreshes authoritative state', async () => {
  const h = harness(); h.open(); await flush();
  let finish;
  h.replies.push(() => new Promise(resolve => { finish = resolve; }));
  h.find('[data-discord-token]').value = 'private-token'; h.submit();
  h.surface.hidden = true; h.dispatch('gateway-surface-change');
  h.replies.push({...verified, enabled: false});
  const count = h.events.length;
  finish({ok: true, json: async () => ({...verified, enabled: true, bot_username: 'stale'})});
  await flush();
  assert.equal(h.find('[data-discord-token]').value, '');
  assert.equal(h.events.length, count + 1);
  assert.equal(h.events.at(-1).detail.bot_username, 'voxel_bot');
  assert.equal(h.find('[data-discord-enabled]').checked, false);
});


test('a restored open workspace loads Discord presentation without opening its surface', async () => {
  const h = harness(true, true);
  await flush();
  assert.equal(h.surface.hidden, true);
  assert.equal(h.calls.length, 1);
  assert.equal(h.calls[0].url, '/api/discord');
  assert.equal(h.events.at(-1).type, 'discord-state');
  assert.equal(h.events.at(-1).detail.configured, false);
});

test('both providers use one shared compact layout with intrinsic actions and bounded controls', () => {
  const css = readFileSync(`${__dirname}/../webui/internal/server/static/agent-gateway.css`, 'utf8');
  const shared = ':is([data-gateway-telegram], [data-gateway-discord])';
  const rules = [...css.matchAll(/([^{}]+)\{([^{}]*)\}/g)];
  const declarations = selector => rules.filter(([, selectors]) => selectors.includes(selector)).map(([, , body]) => body).join(' ');
  for (const provider of ['telegram', 'discord']) {
    const surface = header.match(new RegExp(`<section data-gateway-${provider}[\\s\\S]*?</section>`))[0];
    for (const classes of ['integration-lane', 'integration-controls', 'integration-lane integration-header', 'integration-title']) {
      assert.ok(surface.includes(`class="${classes}"`), `${provider}: ${classes}`);
    }
    assert.match(surface, new RegExp(`<form class="integration-controls" data-${provider}-token-form`));
    assert.doesNotMatch(surface, /style=|class="(?:telegram|discord)-(?:lane|controls|header|title)"/);
  }
  // Provider selectors must never regain independent lane/control/header/title layout rules.
  assert.doesNotMatch(css, /\.(?:telegram|discord)-(?:lane|controls|header|title)\b/);
  for (const [, selectors] of rules.filter(([, selectors]) => /\.integration-(?:lane|controls|header|title)\b/.test(selectors))) {
    if (selectors.includes('[data-discord-guild]')) continue;
    assert.ok(selectors.includes(shared), `layout must be shared: ${selectors.trim()}`);
  }
  for (const kind of ['lane', 'controls']) {
    const body = declarations(`${shared} .integration-${kind}`);
    assert.match(body, /display: flex/);
    assert.match(body, /flex-wrap: wrap/);
    assert.match(body, /align-items: center/);
    assert.match(body, /min-width: 0/);
    assert.doesNotMatch(body, /(?:^|[;\s])(?:width|flex-basis): 100%/);
  }
  assert.match(declarations(`${shared} .integration-header`), /grid-template-columns: 1fr auto/);
  assert.match(declarations(`${shared} .integration-title`), /display: flex; align-items: center/);
  assert.match(css, /@media \(max-width: 600px\)\s*\{\s*:is\(\[data-gateway-telegram\], \[data-gateway-discord\]\) \.integration-header \{ display: flex; flex-wrap: wrap; justify-content: space-between/);
  const margins = declarations(`${shared} .integration-lane :is(input, select, button, .button)`);
  assert.match(margins, /margin: 0/);
  assert.match(declarations(`${shared} .integration-lane :is(button, .button)`), /flex: 0 0 auto/);
  // These scoped selectors override the global input/select width and top margin.
  // Check width separately from max-width:100%, which is needed on small windows.
  for (const [selector, size, grow] of [
    [`${shared} .integration-controls input[data-password-ui="off"]`, '18rem', 0],
    ['[data-gateway-discord] .integration-lane [data-discord-guild]', '14rem', 1],
  ]) {
    const body = declarations(selector);
    assert.ok(body.includes(`width: ${size};`));
    assert.ok(body.includes(`flex: ${grow} 1 ${size};`));
    assert.match(body, /min-width: 0/);
    assert.match(body, /max-width: 100%/);
    assert.match(body, /box-sizing: border-box/);
    assert.doesNotMatch(body, /(?:^|[;\s])width: 100%/);
  }
});

test('global password enhancement excludes both bot tokens while keeping normal reveal controls', () => {
  const app = readFileSync(`${__dirname}/../webui/internal/server/static/app.js`, 'utf8');
  const enhancer = app.slice(app.indexOf('const enhancePasswordFields ='), app.indexOf('const initDestructiveConfirmations ='));
  const selector = enhancer.match(/querySelectorAll\('([^']+)'\)/)[1];
  assert.equal(selector, 'input[type="password"]:not([data-password-ui="off"]):not([data-password-reveal-ready])');
  const fields = ['telegram', 'discord'].map(provider => {
    const markup = header.match(new RegExp(`<input[^>]*data-${provider}-token[^>]*>`))[0];
    assert.match(markup, /data-password-ui="off"/);
    return {type: 'password', dataset: {passwordUi: 'off'}};
  });
  const normal = {type: 'password', name: 'password', autocomplete: 'current-password', dataset: {}};
  fields.push(normal);
  const shells = [];
  for (const field of fields) field.parentNode = {insertBefore(shell, input) { shells.push(shell); input.parentNode = shell; }};
  const document = {
    // Model the attribute selector against actual token attributes and ordinary passwords.
    querySelectorAll(query) {
      assert.equal(query, selector);
      const excluded = [...query.matchAll(/:not\(\[([^=\]]+)(?:="([^"]*)")?\]\)/g)];
      return fields.filter(field => field.type === 'password' && excluded.every(([, attribute, value]) => {
        const key = attribute.slice(5).replace(/-([a-z])/g, (_, letter) => letter.toUpperCase());
        return value === undefined ? !(key in field.dataset) : field.dataset[key] !== value;
      }));
    },
    createElement(tag) {
      return {tagName: tag, children: [], classList: {toggle() {}}, appendChild(child) { this.children.push(child); }, setAttribute(key, value) { this[key] = value; }, addEventListener(type, callback) { this[type] = callback; }};
    },
  };
  vm.runInNewContext(`${enhancer}\nenhancePasswordFields(); enhancePasswordFields();`, {document});
  assert.equal(shells.length, 1, 'normal password enhanced exactly once');
  assert.equal(shells[0].children[0], normal);
  for (const token of fields.slice(0, 2)) assert.equal(token.dataset.passwordRevealReady, undefined);
  const reveal = shells[0].children[1];
  assert.equal(reveal['aria-label'], 'Show password');
  reveal.click(); assert.equal(normal.type, 'text');
  reveal.click(); assert.equal(normal.type, 'password');
});
