const assert = require('node:assert/strict');
const { readFileSync } = require('./helpers/webui-i18n.cjs');
const { test } = require('node:test');
const vm = require('node:vm');
const read = path => readFileSync(`${__dirname}/../${path}`, 'utf8');
const source = read('webui/internal/server/static/system-hardware.js');
const app = read('webui/internal/server/static/app.js');
const powerSource = app.slice(app.indexOf('const systemPower ='), app.indexOf('const initSystemUpdateWorkspace ='));
const header = read('webui/internal/server/templates/header.html');
const css = read('webui/internal/server/static/app.css');
class Element {
  constructor() {
    this.children = []; this.events = {}; this.hidden = true; this.value = '10';
    this.checked = false; this.disabled = false; this.dataset = {}; this.attributes = {};
  }
  addEventListener(name, handler) { (this.events[name] ||= []).push(handler); }
  async emit(name, event = {}) { for (const handler of this.events[name] || []) await handler(event); }
  setAttribute(name, value) { this.attributes[name] = value; }
  append(...nodes) { this.children.push(...nodes); }
  replaceChildren(...nodes) { this.children = nodes; }
  get childElementCount() { return this.children.length; }
  reportValidity() { return /^\d+$/.test(this.value) && Number(this.value) >= 1 && Number(this.value) <= 999; }
  showModal() { this.open = true; }
  close() { this.open = false; }
}
async function settle() { for (let i = 0; i < 80; i++) await Promise.resolve(); }
const deferred = () => { let resolve, reject; const promise = new Promise((yes, no) => { resolve = yes; reject = no; }); return {promise, resolve, reject}; };
function page(hardware, ups, {systemActions = false, development = false, intercept} = {}) {
  let currentHardware = {...hardware};
  const nodes = new Map();
  const node = selector => { if (!nodes.has(selector)) nodes.set(selector, new Element()); return nodes.get(selector); };
  for (const selector of ['[data-display-dialog]', '[data-system-action-dialog]', '[data-system-power]']) node(selector).querySelector = node;
  const buttons = ['reboot', 'firmware-reboot', 'poweroff'].map(action => {
    const button = node(`[data-system-action="${action}"]`); button.dataset.systemAction = action; return button;
  });
  node('[data-system-power]').querySelectorAll = () => buttons;
  node('[data-system-power]').hidden = false;
  for (const button of buttons) {
    const tag = header.match(new RegExp(`<button[^>]*data-system-action="${button.dataset.systemAction}"[^>]*>`))[0];
    button.hidden = /\bhidden\b/.test(tag);
    button.disabled = /\bdisabled\b/.test(tag);
  }
  node('[data-system-actions-csrf]').value = 'csrf-token';
  const requests = [];
  const context = vm.createContext({
    URLSearchParams, controlCenter: node('[data-control-center]'),
    window: {location: {assign() {}}},
    document: {querySelector: selector => selector === "[data-development-profile]" && !development ? null : node(selector), createElement: () => new Element(), addEventListener() {}},
    fetch: async (url, options = {}) => {
      requests.push({url, options});
      const overridden = intercept?.(url, options);
      if (overridden) return overridden;
      if (options.method === 'POST') {
        const action = new URLSearchParams(options.body).get('action');
        if (action.startsWith('lid-')) currentHardware = {...currentHardware, lid_ignored: action === 'lid-enable'};
      }
      const data = url === '/api/ups' ? ups : url === '/api/system-actions' ? {variant: hardware.hws ? 'hws' : 'vm', firmware: {available: hardware.hws}} : currentHardware;
      return {ok: true, json: async () => data};
    },
  });
  if (systemActions) vm.runInContext(powerSource, context);
  vm.runInContext(source, context);
  return {
    node, requests,
    setHardware(value) { currentHardware = value; },
    posts() { return requests.filter(r => r.options.method === 'POST').map(r => new URLSearchParams(r.options.body)); },
    actions() { return this.posts().map(body => body.get('action')); },
    async open() { node('[data-control-center]').open = true; await node('[data-control-center]').emit('toggle'); await settle(); },
    async close() { node('[data-control-center]').open = false; await node('[data-control-center]').emit('toggle'); await settle(); },
    async popup() { await node('[data-display-open]').emit('click'); await settle(); },
    async change(selector, checked) {
      assert.notEqual(node(selector).checked, checked, `${selector} must actually change`);
      node(selector).checked = checked; await node(selector).emit('change'); await settle();
    },
    async apply() { await node('[data-display-apply]').emit('click'); await settle(); },
  };
}
const hws = {hws: true, display_available: true, lid_available: true, lid_ignored: false, batteries: []};
const upsStatus = charge => ({available: true, monitoring: true, source: {mode: 'local'}, devices: [{display_name: 'Monitored UPS', state: 'online', ...(charge === undefined ? {} : {battery_charge: charge})}]});

test('VM hides display, lid and indicators and never requests UPS', async () => {
  const p = page({hws: false, display_available: false, lid_available: false, batteries: []}, upsStatus(50));
  await p.open(); await p.popup();
  for (const selector of ['[data-display-open]', '[data-lid-option]', '[data-power-indicators]']) assert.equal(p.node(selector).hidden, true);
  assert.deepEqual(p.requests.map(r => r.url), ['/api/system-hardware']);
});
test('Development shows hardware controls while unavailable actions remain disabled', async () => {
  const p = page({hws: false, display_available: false, lid_available: false, batteries: []},
    null, {systemActions: true, development: true});
  await p.open(); await p.popup();
  assert.equal(p.node('[data-display-open]').hidden, false);
  assert.equal(p.node('[data-lid-option]').hidden, false);
  for (const selector of ['[data-system-action="firmware-reboot"]', '[data-display-apply]', '[data-lid-ignore]']) {
    assert.equal(p.node(selector).disabled, true);
  }
  await p.apply();
  assert.deepEqual(p.posts(), []);
});
test('HWS requires backend console and actual lid capabilities', async () => {
  const p = page({...hws, display_available: false}, null); await p.open();
  assert.equal(p.node('[data-display-open]').hidden, true);
  const desktop = page({...hws, lid_available: false}, null); await desktop.open(); await desktop.popup();
  assert.equal(desktop.node('[data-lid-option]').hidden, true);
  await desktop.change('[data-lid-ignore]', true); await desktop.apply();
  assert.deepEqual(desktop.actions(), []);
});
test('onboard and existing UPS data appear together as text without battery artwork', async () => {
  const p = page({...hws, batteries: [{capacity: 73, status: 'Discharging'}]}, upsStatus(86)); await p.open();
  const indicators = p.node('[data-power-indicators]');
  assert.equal(indicators.hidden, false);
  assert.deepEqual(indicators.children.map(item => item.children.map(child => child.textContent)), [['Onboard', '73%'], ['UPS', '86%']]);
  assert.match(indicators.children[0].children[1].className, /system-ups-status-on_battery/);
  assert.match(indicators.children[1].children[1].className, /system-ups-status-online/);
  for (const text of [source, css, header]) assert.doesNotMatch(text, /control-battery-icon/);
});
test('missing or invalid percentages never invent data', async () => {
  for (const charge of [undefined, null, -1, 101, NaN, Infinity, '50']) {
    const p = page({...hws, batteries: [{capacity: charge, status: 'Charging'}]}, upsStatus(charge)); await p.open();
    const children = p.node('[data-power-indicators]').children;
    assert.equal(children.length, 2);
    for (const item of children) assert.equal(item.children[1].textContent, '');
  }
  const p = page({...hws, batteries: [{capacity: 0}]}, upsStatus(100)); await p.open();
  assert.deepEqual(p.node('[data-power-indicators]').children.map(item => item.children[1].textContent), ['0%', '100%']);
});
test('no configured or monitored UPS produces no UPS indicator', async () => {
  const p = page(hws, {available:true, source:{mode:'local'}, monitoring:false, devices:[]}); await p.open();
  assert.equal(p.node('[data-power-indicators]').hidden, true);
});
test('UPS failure preserves independent onboard status', async () => {
  const p = page({...hws, batteries:[{capacity:42, status:'Discharging'}]}, null, {
    intercept: url => url === '/api/ups' ? Promise.reject(new Error('NUT unavailable')) : undefined,
  });
  await p.open();
  const indicators = p.node('[data-power-indicators]');
  assert.equal(indicators.hidden, false);
  assert.deepEqual(indicators.children.map(item => item.children.map(child => child.textContent)), [['Onboard', '42%']]);
});
test('UPS states reuse existing classes', async () => {
  for (const state of ['online', 'on_battery', 'low_battery', 'bypass', 'unknown']) {
    const ups = upsStatus(101); ups.devices[0].state = state;
    const p = page(hws, ups); await p.open();
    assert.match(p.node('[data-power-indicators]').children[0].children[1].className, new RegExp('system-ups-status-' + state));
  }
});
test('popup has three matching rows, compact footer and reserved errors; network styles stay scoped', () => {
  const popup = header.match(/<dialog[^>]*data-display-dialog[\s\S]*?<\/dialog>/)[0];
  assert.equal((popup.match(/class="display-setting-row"/g) || []).length, 3);
  assert.equal((popup.match(/<button /g) || []).length, 2);
  assert.match(popup, /data-display-cancel>Cancel/); assert.match(popup, /data-display-apply>Apply/);
  assert.match(popup, /<h2 id="display-power-title">Display power<\/h2>/);
  assert.match(popup, /title="Desired display command, sent only when Apply is pressed" data-display-command/);
  assert.match(popup, /min="1" max="999" step="1" value="10" data-display-delay disabled/);
  assert.doesNotMatch(popup, /data-display-timer[^>]*checked|Turn OFF now|Turn ON now/);
  assert.doesNotMatch(source, /Applied\.|Display off scheduled/);
  assert.match(css, /\[data-display-delay\]\{[^}]*width:3ch/);
  assert.match(css, /\[data-display-status\]\{height:3em;overflow:auto/);
  assert.match(css, /\[data-display-dialog\] \.action-row button\{min-height:34px;padding:\.38rem \.7rem/);
  const session = header.slice(header.indexOf('<section class="control-section control-session">'), header.indexOf('<nav class="control-navigation"'));
  assert.ok(session.includes('<span class="control-section-label">Session &amp; power</span>'));
  assert.match(session, /<div class="control-session-heading">\s*<span class="control-section-label">Session &amp; power<\/span>\s*<span class="control-battery-indicators control-section-label" data-power-indicators hidden><\/span>\s*<\/div>/);
  assert.match(css, /\.control-session-heading\{[^}]*display:flex;[^}]*align-items:center;[^}]*justify-content:space-between/);
  assert.match(css, /\.control-session-heading>\.control-section-label\{margin-bottom:0\}/);
  assert.match(css, /\.control-battery-indicators\{[^}]*justify-content:flex-end/);
  assert.ok(session.indexOf('data-power-indicators') < session.indexOf('control-quick-actions'));
  const order = ['data-system-action="reboot"', 'data-system-action="firmware-reboot"', 'data-system-action="poweroff"', 'data-display-open'].map(text => session.indexOf(text));
  assert.ok(order.every((index, i) => index >= 0 && (i === 0 || index > order[i - 1])));
  const network = css.split('/* Compact Minecraft Network presentation; shared Players rules stay unchanged. */')[1];
  assert.ok(network);
  for (const rule of network.matchAll(/([^{}]+)\{/g)) {
    const selector = rule[1].trim();
    if (selector.startsWith('@media') || selector.startsWith('@container')) continue;
    for (const part of selector.split(',')) assert.ok(part.trim() === '.minecraft-network' || part.trim().startsWith('.minecraft-network '), part);
  }
});
test('timer defaults OFF with disabled ten-second field; Apply unchanged sends no actions', async () => {
  const p = page({...hws, lid_ignored:true}, null); await p.open(); await p.popup();
  assert.equal(p.node('[data-display-command]').checked, true);
  assert.equal(p.node('[data-display-delay]').value, '10');
  assert.equal(p.node('[data-display-timer]').checked, false);
  assert.equal(p.node('[data-display-delay]').disabled, true);
  assert.equal(p.node('[data-lid-ignore]').checked, true);
  await p.apply(); assert.deepEqual(p.actions(), []);
  await p.change('[data-display-timer]', true); assert.equal(p.node('[data-display-delay]').disabled, false);
  await p.change('[data-display-timer]', false); assert.equal(p.node('[data-display-delay]').disabled, true);
});
test('display and lid switches stage commands; Cancel discards all changes and next open reloads lid', async () => {
  const p = page(hws, null); await p.open(); await p.popup();
  await p.change('[data-display-command]', false); await p.change('[data-lid-ignore]', true); await p.change('[data-display-timer]', true);
  assert.deepEqual(p.actions(), []);
  await p.node('[data-display-cancel]').emit('click');
  assert.equal(p.node('[data-display-dialog]').open, false); assert.deepEqual(p.actions(), []);
  p.setHardware({...hws, lid_ignored: true});
  await p.popup();
  assert.equal(p.node('[data-lid-ignore]').checked, true);
  assert.equal(p.node('[data-display-command]').checked, true);
  assert.equal(p.node('[data-display-timer]').checked, false);
  await p.apply(); assert.deepEqual(p.actions(), []);
});
test('one toggle sends display-off; successful session selections survive reopen and Cancel', async () => {
  const p = page(hws, null); await p.open(); await p.popup();
  assert.equal(p.node('[data-display-command]').checked, true);
  await p.apply(); assert.deepEqual(p.actions(), []);
  for (const selection of [false, true]) {
    await p.change('[data-display-command]', selection); await p.apply();
    const expected = selection ? ['display-off', 'display-on'] : ['display-off'];
    assert.deepEqual(p.actions(), expected);
    await p.node('[data-display-cancel]').emit('click'); await p.popup();
    assert.equal(p.node('[data-display-command]').checked, selection);
    await p.apply(); assert.deepEqual(p.actions(), expected);
    await p.change('[data-display-command]', !selection);
    await p.node('[data-display-cancel]').emit('click'); await p.popup();
    assert.equal(p.node('[data-display-command]').checked, selection);
    await p.apply(); assert.deepEqual(p.actions(), expected);
  }
});
test('failed popup status read disables mutation and can be retried on next open', async () => {
  let fail = false;
  const p = page(hws, null, {intercept: url => url === '/api/system-hardware' && fail ? Promise.reject(new Error('unavailable')) : undefined});
  await p.open(); fail = true; await p.popup();
  assert.equal(p.node('[data-display-apply]').disabled, true);
  assert.match(p.node('[data-display-status]').textContent, /unavailable/);
  await p.apply(); assert.deepEqual(p.actions(), []);
  await p.node('[data-display-cancel]').emit('click');
  fail = false; await p.popup();
  assert.equal(p.node('[data-display-apply]').disabled, false);
  assert.equal(p.node('[data-display-status]').textContent, '');
});
test('Apply sends only requested fixed lid actions and preserves CSRF', async () => {
  const p = page(hws, null); await p.open(); await p.popup();
  for (const checked of [true, false]) { await p.change('[data-lid-ignore]', checked); await p.apply(); }
  assert.deepEqual(p.actions(), ['lid-enable', 'lid-disable']);
  assert.ok(p.posts().every(body => body.get('csrf') === 'csrf-token'));
});
test('display-on plus timer applies sequentially; selection survives success; timer is one-shot', async () => {
  const p = page(hws, null); await p.open(); await p.popup();
  await p.change('[data-display-command]', false); await p.apply();
  await p.change('[data-display-command]', true); await p.change('[data-display-timer]', true); await p.change('[data-lid-ignore]', true);
  p.node('[data-display-delay]').value = '123'; await p.apply();
  assert.deepEqual(p.actions(), ['display-off', 'display-on', 'display-delay', 'lid-enable']);
  assert.equal(p.posts()[2].get('delay_seconds'), '123');
  assert.equal(p.node('[data-display-status]').textContent, '');
  assert.equal(p.node('[data-display-dialog]').open, true);
  assert.equal(p.node('[data-display-timer]').checked, false);
  assert.equal(p.node('[data-display-delay]').disabled, true);
  await p.apply(); assert.equal(p.actions().length, 4);
  await p.node('[data-display-cancel]').emit('click'); await p.popup();
  assert.equal(p.node('[data-display-command]').checked, true);
  await p.apply(); assert.equal(p.actions().length, 4);
});
test('immediate display-off suppresses redundant timer but still applies lid', async () => {
  const p = page(hws, null); await p.open(); await p.popup();
  await p.change('[data-display-command]', false);
  await p.change('[data-display-timer]', true); await p.change('[data-lid-ignore]', true); await p.apply();
  assert.deepEqual(p.actions(), ['display-off', 'lid-enable']);
  assert.equal(p.node('[data-display-timer]').checked, false);
});
test('timer validates 1–999 before sending any staged action; disabled invalid timer is ignored', async () => {
  for (const value of ['0', '-1', '1000', '3600', '1.5', '']) {
    const p = page(hws, null); await p.open(); await p.popup();
    await p.change('[data-display-command]', false); await p.change('[data-display-timer]', true);
    p.node('[data-display-delay]').value = value; await p.apply(); assert.deepEqual(p.actions(), []);
    await p.change('[data-display-timer]', false); await p.apply(); assert.deepEqual(p.actions(), ['display-off']);
  }
  for (const value of ['1', '999']) {
    const p = page(hws, null); await p.open(); await p.popup(); await p.change('[data-display-timer]', true);
    p.node('[data-display-delay]').value = value; await p.apply();
    assert.deepEqual(p.actions(), ['display-delay']); assert.equal(p.posts()[0].get('delay_seconds'), value);
  }
});
test('partial failure shows error, stops subsequent actions, and retry does not repeat successful commands', async () => {
  let fail = true;
  const p = page(hws, null, {intercept(url, options) {
    if (options.method === 'POST' && new URLSearchParams(options.body).get('action') === 'display-delay' && fail) {
      fail = false; return Promise.resolve({ok: false});
    }
  }});
  await p.open(); await p.popup();
  await p.change('[data-display-command]', false); await p.apply();
  await p.change('[data-display-command]', true);
  await p.change('[data-display-timer]', true); await p.change('[data-lid-ignore]', true); await p.apply();
  assert.deepEqual(p.actions(), ['display-off', 'display-on', 'display-delay']);
  assert.match(p.node('[data-display-status]').textContent, /could not be applied/);
  assert.equal(p.node('[data-display-command]').checked, true);
  assert.equal(p.node('[data-lid-ignore]').checked, false);
  assert.equal(p.node('[data-display-apply]').disabled, false);
  await p.apply(); assert.deepEqual(p.actions(), ['display-off', 'display-on', 'display-delay', 'display-delay']);
});
test('Apply blocks edits, duplicate submissions and cancellation while commands are in flight', async () => {
  const pending = deferred();
  const p = page(hws, null, {intercept: (_, options) => options.method === 'POST' ? pending.promise : undefined});
  await p.open(); await p.popup(); await p.change('[data-display-command]', false);
  const applying = p.apply(); await settle();
  for (const selector of ['[data-display-command]', '[data-lid-ignore]', '[data-display-timer]', '[data-display-apply]', '[data-display-cancel]']) assert.equal(p.node(selector).disabled, true);
  await p.apply(); await p.node('[data-display-cancel]').emit('click');
  let prevented = false; await p.node('[data-display-dialog]').emit('cancel', {preventDefault() { prevented = true; }});
  assert.equal(prevented, true); assert.equal(p.node('[data-display-dialog]').open, true); assert.deepEqual(p.actions(), ['display-off']);
  pending.resolve({ok: true, json: async () => hws}); await applying;
});
const assertPowerAvailable = p => {
  assert.equal(p.node('[data-system-power]').hidden, false);
  for (const action of ['reboot', 'poweroff']) {
    const button = p.node(`[data-system-action="${action}"]`);
    assert.equal(button.hidden, false, action);
    assert.equal(button.disabled, false, action);
  }
};
const capabilityRequests = p => p.requests.filter(r => ['/api/system-actions', '/api/system-hardware', '/api/ups'].includes(r.url)).map(r => r.url);
const repeatMenu = async p => {
  for (let i = 0; i < 3; i++) { await p.open(); await p.close(); }
  await p.open();
};

test('template starts with available power actions and hidden dynamic controls without a settled gate', () => {
  for (const action of ['reboot', 'poweroff']) {
    const tag = header.match(new RegExp(`<button[^>]*data-system-action="${action}"[^>]*>`))[0];
    assert.doesNotMatch(tag, /\b(disabled|hidden)\b/);
  }
  assert.match(header, /<button[^>]*data-system-action="firmware-reboot"[^>]*\bhidden\b/);
  assert.match(header, /<button[^>]*data-display-open[^>]*\bhidden\b/);
  assert.match(header, /data-power-indicators hidden/);
  for (const text of [header, css, source, powerSource]) {
    assert.doesNotMatch(text, /data-actions-settled|data-hardware-settled|actionsSettled|hardwareSettled/);
  }
  assert.doesNotMatch(css, /\.control-power-actions[^{}]*\{[^}]*visibility\s*:\s*hidden/);
});

test('initial discovery keeps power available in either completion order, including failures, without menu retries', async () => {
  for (const first of ['/api/system-actions', '/api/system-hardware']) {
    for (const failed of [null, '/api/system-actions', '/api/system-hardware']) {
      const pending = {'/api/system-actions': deferred(), '/api/system-hardware': deferred()};
      const p = page(hws, null, {systemActions: true, intercept: url => pending[url]?.promise});
      assert.deepEqual(capabilityRequests(p), ['/api/system-actions', '/api/system-hardware']);
      assertPowerAvailable(p);
      assert.equal(p.node('[data-system-action="firmware-reboot"]').hidden, true);
      assert.equal(p.node('[data-display-open]').hidden, true);
      await repeatMenu(p);
      assert.deepEqual(capabilityRequests(p), ['/api/system-actions', '/api/system-hardware']);
      assertPowerAvailable(p);
      const complete = path => {
        if (failed === path) pending[path].reject(new Error('unavailable'));
        else pending[path].resolve({ok: true, json: async () => path === '/api/system-hardware' ? hws : {variant:'hws', firmware:{available:true}}});
      };
      complete(first); await settle(); assertPowerAvailable(p);
      complete(first === '/api/system-actions' ? '/api/system-hardware' : '/api/system-actions'); await settle();
      assertPowerAvailable(p);
      const display = p.node('[data-display-open]');
      const firmware = p.node('[data-system-action="firmware-reboot"]');
      assert.equal(display.hidden, failed === '/api/system-hardware');
      assert.equal(firmware.hidden, failed === '/api/system-actions');
      assert.equal(firmware.disabled, failed === '/api/system-actions');
      if (failed === '/api/system-hardware') assert.equal(p.node('[data-power-indicators]').hidden, true);
      const requests = capabilityRequests(p);
      assert.deepEqual(requests, failed === '/api/system-hardware'
        ? ['/api/system-actions', '/api/system-hardware']
        : ['/api/system-actions', '/api/system-hardware', '/api/ups']);
      // Detect even a temporary hide/reveal or disable/enable on menu toggle.
      for (const [element, property] of [[firmware, 'hidden'], [firmware, 'disabled'], [display, 'hidden']]) {
        const value = element[property];
        Object.defineProperty(element, property, {get: () => value, set: () => assert.fail(`menu changed ${property}`)});
      }
      await repeatMenu(p);
      assert.deepEqual(capabilityRequests(p), requests);
      assertPowerAvailable(p);
    }
  }
});

test('VM and unsupported capabilities remain hidden after initial discovery and repeated menu opens', async () => {
  for (const hardware of [{hws:false, display_available:false}, {...hws, display_available:false}]) {
    const p = page(hardware, null, {systemActions:true, intercept: url => url === '/api/system-actions'
      ? Promise.resolve({ok:true, json:async () => ({variant:hardware.hws ? 'hws' : 'vm', firmware:{available:!hardware.hws}})}) : undefined});
    assertPowerAvailable(p);
    await settle();
    const requests = capabilityRequests(p);
    assert.deepEqual(requests, hardware.hws ? ['/api/system-actions', '/api/system-hardware', '/api/ups'] : ['/api/system-actions', '/api/system-hardware']);
    await repeatMenu(p);
    assert.deepEqual(capabilityRequests(p), requests);
    assert.equal(p.node('[data-display-open]').hidden, true);
    assert.equal(p.node('[data-system-action="firmware-reboot"]').hidden, true);
    assertPowerAvailable(p);
  }
});

test('initial battery and UPS indicators survive menu toggles without clearing or rebuilding', async () => {
  for (const failUPS of [false, true]) {
    const p = page({...hws, batteries:[{capacity:42, status:'Discharging'}]}, upsStatus(86), {
      systemActions:true,
      intercept: url => failUPS && url === '/api/ups' ? Promise.reject(new Error('NUT unavailable')) : undefined,
    });
    await settle();
    const indicators = p.node('[data-power-indicators]');
    assert.equal(indicators.hidden, false);
    assert.deepEqual(indicators.children.map(item => item.children.map(child => child.textContent)),
      failUPS ? [['Onboard', '42%']] : [['Onboard', '42%'], ['UPS', '86%']]);
    const children = [...indicators.children];
    Object.defineProperty(indicators, 'hidden', {get: () => false, set: () => assert.fail('menu changed indicators visibility')});
    indicators.replaceChildren = () => assert.fail('menu cleared indicators');
    indicators.append = () => assert.fail('menu rebuilt indicators');
    await repeatMenu(p);
    children.forEach((child, i) => assert.equal(indicators.children[i], child));
    assert.deepEqual(capabilityRequests(p), ['/api/system-actions', '/api/system-hardware', '/api/ups']);
    assertPowerAvailable(p);
  }
});

test('Display popup performs a fresh authoritative hardware read each time', async () => {
  const p = page(hws, null, {systemActions:true});
  await settle();
  await repeatMenu(p);
  assert.deepEqual(capabilityRequests(p), ['/api/system-actions', '/api/system-hardware', '/api/ups']);
  p.setHardware({...hws, lid_ignored:true});
  await p.popup();
  assert.equal(p.node('[data-lid-ignore]').checked, true);
  await p.node('[data-display-cancel]').emit('click');
  p.setHardware({...hws, lid_ignored:false});
  await p.popup();
  assert.equal(p.node('[data-lid-ignore]').checked, false);
  assert.deepEqual(capabilityRequests(p), ['/api/system-actions', '/api/system-hardware', '/api/ups', '/api/system-hardware', '/api/system-hardware']);
  assert.deepEqual(p.actions(), []);
});
