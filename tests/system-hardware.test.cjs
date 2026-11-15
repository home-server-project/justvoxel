const assert = require('node:assert/strict');
const { readFileSync } = require('node:fs');
const { test } = require('node:test');
const vm = require('node:vm');
const source = readFileSync(`${__dirname}/../webui/internal/server/static/system-hardware.js`, 'utf8');
class Element {
  constructor() { this.children = []; this.events = {}; this.hidden = true; this.value = '10'; this.dataset = {}; this.attributes = {}; }
  addEventListener(name, handler) { this.events[name] = handler; }
  setAttribute(name, value) { this.attributes[name] = value; }
  append(...nodes) { this.children.push(...nodes); }
  replaceChildren(...nodes) { this.children = nodes; }
  get childElementCount() { return this.children.length; }
  reportValidity() { return true; }
  showModal() { this.open = true; }
  close() { this.open = false; }
}
async function settle() { for (let i = 0; i < 40; i++) await Promise.resolve(); }
function page(hardware, ups) {
  const nodes = new Map();
  const node = selector => { if (!nodes.has(selector)) nodes.set(selector, new Element()); return nodes.get(selector); };
  const dialog = node('[data-display-dialog]');
  dialog.querySelector = node;
  const buttons = ['display-off', 'display-on', 'display-delay'].map(action => { const button = new Element(); button.dataset.displayAction = action; return button; });
  dialog.querySelectorAll = () => buttons;
  node('[data-system-actions-csrf]').value = 'csrf-token';
  const requests = [];
  vm.runInNewContext(source, {
    URLSearchParams,
    document: { querySelector: node, createElement: () => new Element() },
    fetch: async (url, options = {}) => {
      requests.push({url, options});
      return {ok: true, json: async () => url === '/api/ups' ? ups : hardware};
    },
  });
  return { node, buttons, requests, async open() { node('[data-control-center]').open = true; node('[data-control-center]').events.toggle(); await settle(); } };
}
const hws = {hws: true, display_available: true, lid_available: true, lid_ignored: false, batteries: []};
const upsStatus = charge => ({available: true, monitoring: true, source: {mode: 'local'}, devices: [{display_name: 'Monitored UPS', state: 'online', ...(charge === undefined ? {} : {battery_charge: charge})}]});

test('VM hides display, lid and indicators and never requests UPS', async () => {
  const p = page({hws: false, display_available: false, lid_available: false, batteries: []}, upsStatus(50));
  await p.open();
  assert.equal(p.node('[data-display-open]').hidden, true);
  assert.equal(p.node('[data-lid-option]').hidden, true);
  assert.equal(p.node('[data-power-indicators]').hidden, true);
  assert.deepEqual(p.requests.map(r => r.url), ['/api/system-hardware']);
});
test('HWS requires backend console capability before showing Display', async () => {
  const p = page({...hws, display_available: false}, null); await p.open();
  assert.equal(p.node('[data-display-open]').hidden, true);
});
test('onboard and existing UPS data appear simultaneously with percentages inside battery icons', async () => {
  const p = page({...hws, batteries: [{capacity: 73, status: 'Discharging'}]}, upsStatus(86)); await p.open();
  const indicators = p.node('[data-power-indicators]');
  assert.equal(indicators.hidden, false);
  assert.equal(indicators.children.length, 2);
  assert.deepEqual(indicators.children.map(item => item.children.map(child => child.textContent)), [['Onboard', '73%'], ['UPS', '86%']]);
  assert.match(indicators.children[0].children[1].className, /system-ups-status-on_battery/);
  assert.match(indicators.children[1].children[1].className, /system-ups-status-online/);
});
test('detected batteries without percentages have icons without invented data', async () => {
  const p = page({...hws, batteries: [{status: 'Charging'}]}, upsStatus()); await p.open();
  const children = p.node('[data-power-indicators]').children;
  assert.equal(children.length, 2);
  for (const item of children) {
    assert.equal(item.children[1].textContent, '');
    assert.match(item.children[1].attributes['aria-label'], /unavailable/);
  }
  assert.match(children[1].children[1].className, /system-ups-status-online/);
});
test('no configured or monitored UPS produces no UPS indicator', async () => {
  const p = page(hws, {available:true, source:{mode:'local'}, monitoring:false, devices:[]}); await p.open();
  assert.equal(p.node('[data-power-indicators]').hidden, true);
});
test('UPS states reuse existing classes and invalid percentage is omitted', async () => {
  for (const state of ['online', 'on_battery', 'low_battery', 'unknown']) {
    const ups = upsStatus(101); ups.devices[0].state = state;
    const p = page(hws, ups); await p.open();
    const icon = p.node('[data-power-indicators]').children[0].children[1];
    assert.match(icon.className, new RegExp('system-ups-status-' + state));
    assert.equal(icon.textContent, '');
  }
});
test('popup defaults to ten seconds, cancel makes no mutation, on remains available', async () => {
  const p = page({...hws, lid_ignored:true}, null); await p.open();
  p.node('[data-display-open]').events.click();
  assert.equal(p.node('[data-display-delay]').value, '10');
  assert.equal(p.node('[data-lid-ignore]').checked, true);
  p.node('[data-display-cancel]').events.click();
  assert.equal(p.node('[data-display-dialog]').open, false);
  assert.equal(p.requests.filter(r => r.options.method === 'POST').length, 0);
  for (const button of p.buttons) { button.events.click(); await settle(); }
  const bodies = p.requests.filter(r => r.options.method === 'POST').map(r => new URLSearchParams(r.options.body));
  assert.deepEqual(bodies.map(body => body.get('action')), ['display-off', 'display-on', 'display-delay']);
  assert.equal(bodies[2].get('delay_seconds'), '10');
  assert.ok(bodies.every(body => body.get('csrf') === 'csrf-token'));
  assert.ok(p.buttons.every(button => button.disabled === false));
});
test('lid toggle uses fixed actions', async () => {
  const p = page(hws, null); await p.open();
  const lid = p.node('[data-lid-ignore]');
  for (const value of [true, false]) { lid.checked = value; lid.events.change(); await settle(); }
  assert.deepEqual(p.requests.filter(r => r.options.method === 'POST').map(r => new URLSearchParams(r.options.body).get('action')), ['lid-enable', 'lid-disable']);
});
