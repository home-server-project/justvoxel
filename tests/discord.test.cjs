'use strict';
const {test} = require('node:test');
const assert = require('node:assert/strict');
const {readFileSync} = require('node:fs');
const vm = require('node:vm');
const source = readFileSync(`${__dirname}/../webui/internal/server/static/discord.js`, 'utf8');
const header = readFileSync(`${__dirname}/../webui/internal/server/templates/header.html`, 'utf8');
const flush = () => new Promise(resolve => setImmediate(resolve));

function harness(primary = true) {
  class Element {
    constructor(tag = 'div') { this.tagName = tag; this.value = ''; this.textContent = ''; this.hidden = false; this.children = []; this.listeners = {}; this.dataset = {}; }
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
  const dialog = new Element(); dialog.open = true; dialog.querySelector = () => surface;
  const window = new Element();
  const calls = [];
  const replies = [];
  const context = vm.createContext({
    URLSearchParams, window,
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
  return {find, surface, dialog, window, calls, replies, dispatch,
    open() { surface.hidden = false; dispatch('gateway-surface-change'); },
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
  assert.equal(h.find('[data-discord-status]').textContent, 'Disconnected');
  const denied = harness(false); denied.open(); await flush();
  assert.equal(denied.calls.length, 0);
  h.surface.hidden = true; h.dispatch('gateway-surface-change');
  h.dispatch('discord-refresh'); await flush();
  assert.equal(h.calls.length, 2);
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
  assert.equal(h.find('[data-discord-status]').textContent, 'Token saved — verify the bot');
  assert.equal(h.find('[data-discord-verify]').disabled, false);
  h.replies.push(verified); h.click('verify'); await flush();
  assert.deepEqual(JSON.parse(h.calls.at(-1).options.body.get('request')), {action: 'verify'});
  assert.equal(h.find('[data-discord-status]').textContent, 'Bot verified');
  const identity = h.find('[data-discord-identity]');
  assert.equal(identity.hidden, false);
  assert.deepEqual(identity.children.map(node => node.textContent), ['Bot', 'Voxel Bot', '@voxel_bot', 'Discord application', 'Voxel']);
  h.click('disconnect'); await flush();
  assert.deepEqual(JSON.parse(h.calls.at(-1).options.body.get('request')), {action: 'disconnect'});
  assert.equal(identity.hidden, true);
  assert.equal(identity.children.length, 0);
  assert.equal(h.find('[data-discord-verify]').disabled, true);
});

test('unknown or malformed state hides stale identity and uses the unavailable label', async () => {
  const h = harness();
  h.replies.push(verified); h.open(); await flush();
  for (const snapshot of [null, {}, [], {status: 'constructor'}, {status: 'connected'}, {status: {}}]) {
    h.replies.push(snapshot === null ? () => ({ok: true, json: async () => null}) : snapshot);
    h.dispatch('discord-refresh'); await flush();
    assert.equal(h.find('[data-discord-status]').textContent, 'Connection unavailable');
    assert.equal(h.find('[data-discord-identity]').hidden, true);
    assert.equal(h.find('[data-discord-identity]').children.length, 0);
  }
});

test('identity is text and token clears whenever the surface closes or changes away', async () => {
  const h = harness();
  h.replies.push({...verified, bot_name: '<img src=x onerror=bad()>', application_name: '<script>bad()</script>'});
  h.open(); await flush();
  assert.equal(h.find('[data-discord-identity]').children[1].textContent, '<img src=x onerror=bad()>');
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
  assert.equal(h.find('[data-discord-status]').textContent, 'Connection unavailable');
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
  finish({ok: true, json: async () => verified}); await flush();
  assert.equal(h.find('[data-discord-identity]').hidden, true);
  h.dialog.open = true; h.open(); await flush();
  assert.equal(h.find('[data-discord-status]').textContent, 'Disconnected');
});

test('Discord stays within Stage 2A and reuses existing static styles', () => {
  assert.doesNotMatch(source, /localStorage|sessionStorage|console\.|innerHTML|insertAdjacentHTML|setInterval|setTimeout|\.style\.|cssText|insertRule|<style/);
  assert.deepEqual([...source.matchAll(/action: '([^']+)'/g)].map(match => match[1]), ['credential', 'verify', 'enabled', 'guild', 'disconnect']);
  assert.match(header, /<script src="\/static\/discord.js" defer><\/script>/);
  assert.match(header, /type="password" name="token" autocomplete="off" data-password-ui="off" data-discord-token/);
  const surface = header.slice(header.indexOf('<section data-gateway-discord'), header.indexOf('<section data-gateway-help'));
  assert.match(surface, /class="panel"/);
  assert.match(surface, /class="form-actions"/);
  assert.doesNotMatch(surface, /style=|<style|data-discord-pair|data-discord-channel|notifications/);
  const icon = readFileSync(`${__dirname}/../webui/internal/server/static/ai-providers/discord.svg`, 'utf8');
  assert.match(icon, /<svg/);
  assert.doesNotMatch(icon, /<script|<style|<animate|<foreignObject|\bhref=|\bsrc=|\bon\w+=|<text|font/);
});

test('Stage 2A enable and guild binding are explicit and readiness gated', async () => {
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
  assert.equal(h.find('[data-discord-runtime]').textContent, 'Connecting');
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

test('Stage 2A install link has a fixed origin and minimum bot permissions', async () => {
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

test('Stage 2A reports empty discovery and lost bound access without changing binding', async () => {
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
  assert.match(h.find('[data-discord-runtime]').textContent, /disable and enable Discord/);
});
