const assert = require('node:assert/strict');
const { readFileSync } = require('node:fs');
const path = require('node:path');
const { test } = require('node:test');
const vm = require('node:vm');

const read = file => readFileSync(path.join(__dirname, '../webui/internal/server', file), 'utf8');
const source = read('static/server-migration-switch.js');
const app = read('static/app.js');
const english = JSON.parse(read('locales/en.json'));
const flush = () => new Promise(resolve => setImmediate(resolve));
const sourceID = 'jv-source1';
const cloneID = 'jv-clone01';
const snapshot = (state, target = 'leaf') => ({
  target, operation: { operation_id: 'switch-1', state, status: state },
});
const plan = target => ({ current: 'paper', target, version: '1.21.4', clone: true, plan_fingerprint: 'reviewed' });
const completedClone = (id = cloneID) => ({
  ...snapshot('succeeded'), action: 'clone', clone_id: id,
  operation: { ...snapshot('succeeded').operation, status: 'Switch completed at Minecraft 26.3.' },
});
const actionClass = 'action-row migration-switch-action';

class Element {
  constructor(tag) {
    this.tag = tag;
    this.dataset = {};
    this.events = {};
    this.children = [];
    this.style = { setProperty() {} };
    this.attributes = {};
    this.disabled = false;
    const classes = new Set();
    this.classList = {
      contains: name => classes.has(name),
      toggle: (name, enabled) => enabled ? classes.add(name) : classes.delete(name),
    };
  }
  append(...nodes) { this.children.push(...nodes); }
  appendChild(node) { this.append(node); }
  replaceChildren(...nodes) { this.children = nodes; }
  insertBefore(node, before) {
    const index = this.children.indexOf(before);
    assert.notEqual(index, -1);
    this.children.splice(index, 0, node);
    node.remove = () => { this.children = this.children.filter(child => child !== node); };
  }
  querySelectorAll(tag) { return this.children.filter(node => node.tag === tag); }
  addEventListener(event, listener) { this.events[event] = listener; }
  click() { return this.events.click?.(); }
  setAttribute(name, value) { this.attributes[name] = value; }
}

function fixture(operation, readPlan = plan, apply, recover, setupNavigation, matrix) {
  const state = new Element('p');
  const progress = new Element('div');
  const matrixRun = new Element('button');
  const matrixStatus = new Element('div');
  matrixRun.hidden = matrixRun.disabled = true;
  const review = new Element('div');
  const cards = ['paper', 'leaf', 'fabric'].map(target => {
    const card = new Element('button');
    card.dataset.switchTarget = target;
    const marker = new Element('span');
    card.querySelector = () => marker;
    return card;
  });
  const timers = [];
  const operations = [];
  const root = new Element('section');
  root.dataset.serverId = matrix?.server ?? sourceID;
  root.isConnected = true;
  const workspace = {
    open: true,
    activeTab: 'switch',
    querySelector(selector) {
      assert.equal(selector, '[data-migration-tab="switch"][aria-selected="true"]');
      return this.activeTab === 'switch' ? {} : null;
    },
  };
  root.closest = selector => selector === '[data-migration-workspace-dialog]' ? workspace : null;
  root.matches = selector => selector === '[data-migration-switch]';
  root.querySelectorAll = () => cards;
  root.querySelector = selector => ({
    '[data-switch-state]': state, '[data-switch-operation]': progress,
    '[data-switch-review]': review, '[data-switch-csrf]': { value: 'csrf' },
    ...(matrix && {'[data-switch-matrix-run]': matrixRun, '[data-switch-matrix-status]': matrixStatus}),
  })[selector];
  const context = {
    window: {}, URLSearchParams, justVoxelSelectionGeneration: { migration: 1 },
    jvT: (key, values = {}) => (english[key] ?? key).replace(/\{(\w+)\}/g, (token, name) => values[name] ?? token),
    document: { createElement: tag => new Element(tag) },
    setTimeout: callback => timers.push(callback),
    fetch: async (request, options) => {
      const url = new URL(request, 'https://justvoxel.example');
      if (url.pathname.startsWith('/api/migration/switch-matrix')) {
        assert.ok(matrix);
        assert.equal(url.search, '');
        if (options.method === 'POST') assert.equal(options.body.get('csrf'), 'csrf');
        return matrix.request(url.pathname, options);
      }
      assert.equal(url.searchParams.get('server'), sourceID);
      const action = url.pathname.split('/').at(-1);
      let result;
      if (action === 'plan') {
        result = await readPlan(url.searchParams.get('target'), url.searchParams.get('crossplay'));
      } else if (action === 'operation') {
        operations.push(url.searchParams.get('operation_id'));
        result = await operation(url.searchParams.get('operation_id'));
      } else if (action === 'recover') {
        assert.equal(options.body.get('operation_id'), 'switch-1');
        assert.equal(options.body.get('confirm'), 'yes');
        assert.equal(options.body.get('csrf'), 'csrf');
        result = recover ? await recover() : snapshot('resolved');
      } else if (action === 'apply' && apply) {
        return apply(options.body);
      } else {
        assert.fail(`Unexpected action ${action}`);
      }
      return { ok: true, json: async () => result };
    },
  };
  const navigation = setupNavigation?.(context);
  vm.runInNewContext(source, context);
  context.window.JustVoxelServerMigrationSwitch.init(root);
  const selected = () => cards.filter(card => card.classList.contains('is-selected')).map(card => card.dataset.switchTarget);
  return { cards, root, workspace, state, review, progress, matrixRun, matrixStatus, timers, operations,
    generation: context.justVoxelSelectionGeneration, selected, navigation };
}

function nativeNavigation(context) {
  const otherID = 'jv-other01';
  const nav = {
    servers: [sourceID, otherID, cloneID],
    requests: [],
    selectionEvents: [],
    pluginsOpened: 0,
  };
  const launcher = new Element('button');
  const plugins = new Element('button');
  const dialog = new Element('dialog');
  const content = new Element('div');
  const state = new Element('p');
  dialog.dataset.workspaceWindow = 'minecraft-servers';
  dialog.open = false;
  dialog.show = () => { dialog.open = true; };
  const slots = {
    '[data-minecraft-servers-content]': content,
    '[data-minecraft-servers-state]': state,
    '[data-minecraft-servers-actions]': new Element('div'),
    '[data-minecraft-servers-close]': new Element('button'),
    '[data-minecraft-servers-refresh]': new Element('button'),
  };
  dialog.querySelector = selector => slots[selector];
  content.querySelector = () => null;
  content.replaceChildren = (fragment = []) => { content.children = fragment; };
  plugins.addEventListener('click', () => { nav.pluginsOpened++; });

  let address = new URL(`https://justvoxel.example/?workspace=migration&tab=switch&server=${sourceID}`);
  const location = {
    get href() { return address.href; },
    set href(_) { assert.fail('Switch must not navigate the document'); },
    get origin() { return address.origin; },
    get pathname() { return address.pathname; },
    get search() { return address.search; },
    assign() { assert.fail('Switch must not assign location'); },
    reload() { assert.fail('Switch must not reload the document'); },
  };
  const launchers = {
    '[data-minecraft-servers-open]': launcher,
    '[data-plugins-open]': plugins,
    '[data-minecraft-servers-workspace-dialog]': dialog,
  };
  Object.assign(context.document, {
    body: { classList: { contains: name => name === 'role-administrator' } },
    querySelector: selector => launchers[selector] ?? null,
    querySelectorAll: () => [],
    createRange: () => ({
      selectNodeContents: node => assert.equal(node, content),
      createContextualFragment: ids => ids.map(id => {
        const card = new Element('article');
        card.dataset.serverId = id;
        return card;
      }),
    }),
  });
  const switchFetch = context.fetch;
  Object.assign(context, {
    URL, location, controlCenter: { open: false },
    ResizeObserver: class { observe() {} },
    CustomEvent: class { constructor(type, {detail}) { this.type = type; this.detail = detail; } },
    justVoxelMinecraftServers: root => assert.equal(root, content),
    fetch: async (url, options) => {
      if (url === '/api/minecraft/workspace/selection?selection=none') {
        nav.requests.push('selection inventory');
        const servers = nav.readInventory ? await nav.readInventory() : nav.servers;
        return { ok: true, json: async () => ({ servers: servers.map(id => ({id})) }) };
      }
      if (url === '/workspace/minecraft-servers?') {
        nav.requests.push('workspace list');
        if (nav.listUnavailable) return { ok: false };
        return { ok: true, text: async () => nav.servers };
      }
      return switchFetch(url, options);
    },
  });
  Object.assign(context.window, {
    location, history: { replaceState(_state, _unused, target) { address = new URL(target, address); } },
    matchMedia: () => ({ matches: true }),
    open() { assert.fail('Switch must not open a new browser tab'); },
    dispatchEvent: event => nav.selectionEvents.push(event.detail),
  });
  const selection = app.slice(0, app.indexOf('// Identity fallback'));
  const windows = app.slice(app.indexOf('const workspaceWindows ='), app.indexOf('// One explicit-open diagnostic viewer'));
  const servers = app.slice(app.indexOf('const minecraftServersDialog ='), app.indexOf('// Both Minecraft Network launchers'));
  vm.runInNewContext(selection + windows + servers, context);
  const selections = { minecraft: otherID, migration: sourceID, plugins: otherID, backups: otherID, version: sourceID };
  for (const [workspace, id] of Object.entries(selections)) assert.equal(context.justVoxelSelectServer(id, workspace), true);
  address = new URL(`https://justvoxel.example/?workspace=migration&tab=switch&server=${sourceID}`);
  nav.selectionEvents.length = 0;
  return Object.assign(nav, {
    launcher, dialog, content, state, location,
    inventory: refresh => context.justVoxelLoadServerSelections(refresh),
    selections: () => Object.fromEntries(Object.keys(selections).map(workspace => [
      workspace, new URL(context.justVoxelSelectedURL('/api/read', workspace), address).searchParams.get('server'),
    ])),
  });
}

test('durable Switch target stays selected through polling and a terminal reload', async () => {
  let state = 'queued';
  const f = fixture(() => snapshot(state));
  // The server's result must clear a previously selected unrelated card.
  f.cards[2].classList.toggle('is-selected', true);
  await flush();
  assert.deepEqual(f.selected(), ['leaf']);
  assert.equal(f.progress.children[0].textContent, 'queued');
  for (state of ['validating', 'running', 'verifying', 'resolved']) {
    assert.equal(f.timers.length, 1);
    await f.timers.shift()();
    assert.deepEqual(f.selected(), ['leaf']);
    assert.equal(f.progress.children[0].textContent, state);
    assert.equal(f.review.hidden, true);
  }
  assert.deepEqual(f.operations, ['', 'switch-1', 'switch-1', 'switch-1', 'switch-1', 'switch-1']);
});

test('fresh open has no selection or progress for terminal history', async () => {
  for (const state of ['resolved', 'succeeded']) {
    const f = fixture(id => id === 'switch-1' ? snapshot(state) : { operation: null });
    await flush();
    assert.deepEqual(f.operations, ['']);
    assert.deepEqual(f.selected(), []);
    assert.deepEqual(f.progress.children, []);
  }
});

test('fresh open reconnects to unresolved Switch and recovery keeps the same card', async () => {
  for (let state of ['needs_attention', 'failed']) {
    let finishRecovery;
    let recoveryRequests = 0;
    const f = fixture(id => state === 'resolved' && !id ? { operation: null } : snapshot(state), plan, undefined, () => {
      recoveryRequests++;
      return new Promise(resolve => { finishRecovery = resolve; });
    });
    await flush();
    assert.deepEqual(f.selected(), ['leaf']);
    assert.equal(f.review.hidden, true);
    assert.equal(f.progress.children[0].textContent, state);
    f.cards[2].events.click();
    assert.deepEqual(f.selected(), ['leaf']);
    assert.equal(f.progress.children[0].textContent, state);
    if (state === 'needs_attention') {
      const row = f.progress.children[1];
      assert.equal(row.className, actionClass);
      assert.deepEqual(row.children.map(node => node.tag), ['p', 'button']);
      const [description, button] = row.children;
      assert.equal(description.textContent, 'Restore original server state and clean up failed clone.');
      assert.equal(button.textContent, 'Recover Switch');
      assert.equal(button.disabled, false);
      const request = button.events.click();
      assert.equal(button.disabled, true);
      assert.equal(button.textContent, 'Recovering');
      await button.events.click();
      assert.equal(recoveryRequests, 1);
      state = 'resolved';
      finishRecovery(snapshot(state));
      await request;
      assert.deepEqual(f.operations, ['', 'switch-1']);
      assert.deepEqual(f.selected(), ['leaf']);
      assert.deepEqual(f.progress.children.map(node => node.textContent), ['resolved']);
      assert.equal(f.review.hidden, true);
      assert.equal(f.cards[1].disabled, false);
      assert.equal(f.cards[2].disabled, false);
    }
  }
});

test('Switch recovery keeps errors between status and the single action until retry resolves', async () => {
  const status = 'Switch did not complete. Review Switch recovery; the conversion will not be retried automatically.';
  const error = 'Instance runtime recovery is required. Retry the instance recovery operation.';
  let requests = 0;
  let finish;
  const f = fixture(() => {
    const pending = snapshot('needs_attention');
    pending.operation.status = status;
    return pending;
  }, plan, undefined, () => {
    if (++requests === 1) throw new Error(error);
    return new Promise(resolve => { finish = resolve; });
  });
  await flush();
  const [statusRow, actionRow] = f.progress.children;
  const [description, recover] = actionRow.children;
  assert.equal(statusRow.textContent, status);
  assert.equal(f.progress.children.length, 2);
  assert.equal(actionRow.className, actionClass);
  assert.deepEqual(actionRow.children.map(node => node.tag), ['p', 'button']);
  assert.equal(description.textContent, 'Restore original server state and clean up failed clone.');
  assert.equal(recover.type, 'button');
  await recover.events.click();
  assert.deepEqual(f.progress.children.map(node => node.tag), ['p', 'p', 'div']);
  assert.deepEqual(f.progress.children.slice(0, 2).map(node => node.textContent), [status, error]);
  assert.equal(f.progress.children[1].className, 'migration-switch-error');
  assert.equal(f.progress.children[2], actionRow);
  assert.equal(f.state.textContent, '');
  assert.equal(recover.disabled, false);
  const retry = recover.events.click();
  assert.deepEqual(f.progress.children, [statusRow, actionRow]);
  assert.equal(recover.textContent, 'Recovering');
  assert.equal(recover.disabled, true);
  await recover.events.click();
  assert.equal(requests, 2);
  finish(snapshot('resolved'));
  await retry;
  assert.deepEqual(f.progress.children.map(node => node.textContent), ['resolved']);
  assert.equal(f.cards[2].disabled, false);
  f.cards[2].events.click();
  await flush();
  assert.deepEqual(f.progress.children, []);
  assert.equal(f.state.textContent, '');
});

test('a failed recovery response from a prior server selection cannot render an error', async () => {
  let fail;
  const f = fixture(() => snapshot('needs_attention'), plan, undefined, () =>
    new Promise((_, reject) => { fail = reject; }));
  await flush();
  const button = f.progress.children.at(-1).children.at(-1);
  const recovery = button.events.click();
  f.generation.migration++;
  f.root.isConnected = false;
  fail(new Error('old server error'));
  await recovery;
  assert.equal(f.progress.children.length, 2);
  assert.equal(f.state.textContent, '');
});

test('a new card clears terminal progress immediately and stale reviews cannot restore earlier selections', async () => {
  const pending = new Map();
  let deferPlans = false;
  const f = fixture(id => snapshot(id ? 'resolved' : 'running'), target =>
    deferPlans ? new Promise(resolve => pending.set(target, resolve)) : plan(target));
  await flush();
  await f.timers.shift()();
  assert.deepEqual(f.selected(), ['leaf']);
  assert.equal(f.progress.children[0].textContent, 'resolved');
  deferPlans = true;
  f.cards[2].events.click();
  assert.deepEqual(f.selected(), ['fabric']);
  assert.deepEqual(f.progress.children, []);
  assert.equal(f.review.hidden, true);
  assert.equal(f.state.textContent, 'Checking the selected server…');
  f.cards[1].events.click();
  assert.deepEqual(f.selected(), ['leaf']);
  pending.get('leaf')(plan('leaf'));
  await flush();
  const review = f.review.children.slice();
  assert.equal(f.review.hidden, false);
  pending.get('fabric')(plan('fabric'));
  await flush();
  assert.deepEqual(f.selected(), ['leaf']);
  assert.deepEqual(f.review.children, review);
  assert.deepEqual(f.progress.children, []);
});

test('a rejected Apply re-enables the catalogue when there is no unresolved Switch', async () => {
  let applied = false;
  const f = fixture(() => ({ operation: null }), plan, body => {
    applied = true;
    assert.equal(body.get('target'), 'fabric');
    assert.equal(body.get('plan_fingerprint'), 'reviewed');
    assert.equal(body.get('action'), 'clone');
    assert.equal(body.get('confirm'), 'yes');
    return { ok: false, json: async () => ({ error: 'Apply rejected' }) };
  });
  await flush();
  f.cards[2].events.click();
  await flush();
  const consent = f.review.children.find(node => node.tag === 'label').children[1];
  const clone = f.review.children.at(-1).children.at(-1);
  consent.checked = true;
  consent.events.change();
  await clone.events.click();
  assert.equal(applied, true);
  assert.equal(f.cards[2].disabled, false);
  assert.equal(f.review.hidden, true);
  assert.deepEqual(f.selected(), []);
  assert.deepEqual(f.progress.children, []);
});

test('a new Apply follows its own operation after a terminal result', async () => {
  let nextState = 'validating';
  const nextOperation = () => ({
    target: 'fabric', operation: { operation_id: 'switch-2', state: nextState, status: nextState },
  });
  const f = fixture(id => {
    if (!id) return snapshot('running');
    if (id === 'switch-1') return snapshot('resolved');
    assert.equal(id, 'switch-2');
    return nextOperation();
  }, plan, body => {
    assert.equal(body.get('target'), 'fabric');
    return { ok: true, json: async () => nextOperation() };
  });
  await flush();
  await f.timers.shift()();
  f.cards[2].events.click();
  await flush();
  const consent = f.review.children.find(node => node.tag === 'label').children[1];
  const clone = f.review.children.at(-1).children.at(-1);
  consent.checked = true;
  consent.events.change();
  await clone.events.click();
  assert.equal(f.progress.children[0].textContent, 'validating');
  nextState = 'succeeded';
  await f.timers.shift()();
  assert.deepEqual(f.operations, ['', 'switch-1', 'switch-1', 'switch-2', 'switch-2', 'switch-2']);
  assert.deepEqual(f.selected(), ['fabric']);
  assert.equal(f.progress.children[0].textContent, 'succeeded');
});

test('operation response from an old workspace generation is ignored', async () => {
  let resolveOperation;
  const f = fixture(() => new Promise(resolve => { resolveOperation = resolve; }));
  f.generation.migration++;
  resolveOperation(snapshot('running'));
  await flush();
  assert.deepEqual(f.selected(), []);
  assert.deepEqual(f.progress.children, []);
});

test('Switch uses one aligned stack below the catalogue for operation, checking, errors and review', () => {
  const markup = read('templates/migration_workspace_switch.html');
  assert.match(markup, /<\/fieldset>{{end}}\s*<\/div>\s*<div class="migration-switch-content migration-switch-stack">\s*<div class="migration-switch-stack" data-switch-operation role="status"><\/div>\s*<div class="migration-switch-stack" data-switch-matrix-status role="status"><\/div>\s*<p data-switch-state role="status">{{if \.ServerID}}{{t "ui.e29d2712155f"}}{{else}}{{t "ui.389e87e83148"}}{{end}}<\/p>\s*<section class="migration-switch-stack" data-switch-review hidden><\/section>\s*<\/div>/);
  const css = read('static/server-migration-switch.css');
  const rule = selector => {
    const start = css.indexOf(selector + '{');
    assert.notEqual(start, -1, selector);
    return css.slice(start + selector.length + 1, css.indexOf('}', start));
  };
  const stack = rule('.migration-switch .migration-switch-stack');
  assert.match(stack, /display:grid/);
  assert.match(stack, /grid-template-columns:minmax\(0,1fr\)/);
  assert.match(stack, /gap:var\(--jv-space-sm\)/);
  assert.match(rule('.migration-switch .migration-switch-content'), /text-align:start/);
  assert.match(rule('.migration-switch .migration-switch-content>:empty'), /position:absolute/);
  const text = rule('.migration-switch .migration-switch-content :is(p,label)');
  for (const property of ['margin:0', 'padding:0', 'font-size:inherit', 'line-height:inherit']) {
    assert.ok(text.includes(property), property);
  }
  const actions = rule('.migration-switch .migration-switch-content .migration-switch-action');
  for (const property of ['display:flex', 'flex-wrap:wrap', 'align-items:center', 'justify-content:flex-end', 'margin:0']) {
    assert.ok(actions.includes(property), property);
  }
  assert.match(rule('.migration-switch .migration-switch-action>p'), /flex:1 1 14rem;min-width:0/);
  assert.match(rule('.migration-switch .migration-switch-action>button'), /max-width:100%;white-space:normal/);
  assert.doesNotMatch(css, /migration-switch-recovery/);
});

test('catalogue loading, refresh and network errors use the shared state row', async () => {
  let finish;
  const f = fixture(() => ({operation: null}), target => target === 'fabric'
    ? new Promise((_, reject) => { finish = reject; }) : plan(target));
  assert.equal(f.state.textContent, 'Checking the selected server…');
  await flush();
  assert.equal(f.state.textContent, 'Checking the selected server…');
  finish(new Error('Unavailable'));
  await flush();
  assert.equal(f.state.textContent, english['ui.f2d442c804c0']);
  assert.equal(f.state.classList.contains('migration-switch-error'), true);
  assert.equal(f.review.hidden, true);

  const failed = fixture(() => { throw new Error('Operation lookup failed'); });
  await flush();
  assert.equal(failed.state.textContent, 'Operation lookup failed');
  assert.equal(failed.state.classList.contains('migration-switch-error'), true);
});

test('successful clone shares the recovery row layout and opens or raises the native Servers workspace', async () => {
  let operation = snapshot('running');
  let finishRefresh;
  const readPlan = target => operation.operation.state === 'succeeded' && target === 'fabric'
    ? new Promise(resolve => { finishRefresh = resolve; }) : plan(target);
  const f = fixture(() => operation, readPlan, undefined, undefined, nativeNavigation);
  const nav = f.navigation;
  const selections = nav.selections();
  const address = nav.location.href;
  await flush();
  operation = completedClone();
  const refresh = f.timers.shift()();
  await flush();
  assert.equal(f.state.textContent, 'Checking the selected server…');
  const [status, actions] = f.progress.children;
  finishRefresh(plan('fabric'));
  await refresh;
  assert.equal(f.state.textContent, '');
  assert.deepEqual(f.progress.children, [status, actions]);
  assert.equal(status.textContent, 'Switch completed at Minecraft 26.3.');
  assert.equal(actions.className, actionClass);
  assert.deepEqual(actions.children.map(node => node.tag), ['p', 'button']);
  const [description, open] = actions.children;
  assert.equal(description.textContent, english['ui.b62c04b44caf']);
  assert.equal(open.textContent, english['ui.10c040af618e']);
  assert.equal(open.type, 'button');
  assert.equal(open.href, undefined);

  assert.equal(nav.dialog.open, false);
  const opening = open.click();
  await open.click();
  await opening;
  await flush();
  assert.equal(nav.dialog.open, true);
  assert.deepEqual(nav.requests, ['selection inventory', 'workspace list']);
  assert.ok(nav.content.children.some(card => card.dataset.serverId === cloneID));
  const list = nav.content.children;
  const zIndex = Number(nav.dialog.style.zIndex);

  await open.click();
  await flush();
  assert.ok(Number(nav.dialog.style.zIndex) > zIndex);
  assert.notEqual(nav.content.children, list);
  assert.ok(nav.content.children.some(card => card.dataset.serverId === cloneID));
  assert.deepEqual(nav.requests, ['selection inventory', 'workspace list', 'selection inventory', 'workspace list']);
  assert.deepEqual(nav.selections(), selections);
  assert.deepEqual(nav.selectionEvents, []);
  assert.equal(nav.location.href, address);
});

test('only the exact, valid clone ID of a successful clone offers the action', async () => {
  for (const result of [
    {...completedClone(), action: 'convert'},
    completedClone(''), completedClone('jv-../../x'), completedClone(sourceID),
    {...completedClone(), operation: snapshot('running').operation},
  ]) {
    const f = fixture(() => result);
    await flush();
    assert.equal(f.progress.children.length, 1);
  }
});

test('missing clone and failed inventory refresh preserve the successful result and allow retry', async () => {
  const f = fixture(() => completedClone(), plan, undefined, undefined, nativeNavigation);
  const nav = f.navigation;
  const initialInventory = nav.inventory(true);
  await initialInventory;
  await flush();
  nav.requests.length = 0;
  const [status, actions] = f.progress.children;
  const open = actions.children[1];
  nav.readInventory = () => { throw new Error('Inventory unavailable'); };
  await open.click();
  assert.equal(nav.inventory(), initialInventory);
  assert.equal(f.progress.children[0], status);
  assert.equal(f.progress.children[1].textContent, english['ui.c3ba26d72a06']);
  assert.equal(f.progress.children[2], actions);
  assert.equal(open.disabled, false);
  assert.equal(nav.dialog.open, false);

  nav.readInventory = () => [sourceID, 'jv-other01'];
  await open.click();
  assert.equal(nav.dialog.open, false);
  assert.deepEqual(nav.requests, ['selection inventory', 'selection inventory']);
  assert.equal(f.progress.children.length, 3);
  assert.equal(f.progress.children[2], actions);

  nav.readInventory = null;
  nav.listUnavailable = true;
  await open.click();
  await flush();
  assert.equal(nav.dialog.open, true);
  assert.equal(nav.state.textContent, english['ui.c3ba26d72a06']);
  assert.deepEqual(f.progress.children, [status, actions]);
  nav.listUnavailable = false;
  await open.click();
  await flush();
  assert.equal(nav.state.textContent, '');
  assert.ok(nav.content.children.some(card => card.dataset.serverId === cloneID));
  assert.deepEqual(f.progress.children, [status, actions]);
});

test('a stale clone callback cannot launch Servers after changing review, source, tab or window', async () => {
  for (const invalidate of [
    f => f.cards[2].click(),
    f => { f.generation.migration++; },
    f => { f.workspace.activeTab = 'export'; },
    f => { f.workspace.open = false; },
  ]) {
    const f = fixture(() => completedClone(), plan, undefined, undefined, nativeNavigation);
    const nav = f.navigation;
    let finish;
    nav.readInventory = () => new Promise(resolve => { finish = resolve; });
    await flush();
    const open = f.progress.children.at(-1).children.at(-1);
    const opening = open.click();
    invalidate(f);
    finish(nav.servers);
    await opening;
    await open.click();
    assert.equal(nav.dialog.open, false);
    assert.deepEqual(nav.requests, ['selection inventory']);
  }
});

test('compatibility review opens native Plugins for its exact source and keeps review and other selections', async () => {
  const reviewPlan = target => ({
    ...plan(target), reason: 'Convert is unavailable', warnings: ['Review installed content'],
    content: [
      {name: 'Plugin A', filename: 'a.jar', compatibility: 'verified', reason: 'Compatible'},
      {name: 'Plugin B', filename: 'b.jar', compatibility: 'unknown', reason: 'Unknown origin'},
    ],
  });
  const f = fixture(() => ({operation: null}), reviewPlan, undefined, undefined, nativeNavigation);
  const nav = f.navigation;
  const selections = nav.selections();
  await flush();
  f.cards[2].click();
  assert.equal(f.state.textContent, 'Checking the selected server…');
  await flush();
  assert.equal(f.review.hidden, false);
  assert.equal(f.state.textContent, '');
  const before = f.review.children.slice();
  const [current, target, version, convertStatus, cloneStatus] = before;
  assert.deepEqual([current, target, version, convertStatus, cloneStatus].map(row => row.textContent), [
    'Current software: paper', 'Target software: fabric', 'Minecraft version: 1.21.4',
    'Convert existing server: Unavailable', 'Clone and convert: Available',
  ]);
  assert.equal(before.find(row => row.textContent.includes('Plugin A')).className, 'muted');
  assert.equal(before.find(row => row.textContent.includes('Plugin B')).className, 'notice warning');
  assert.equal(before.find(row => row.textContent === 'Review installed content').className, 'notice warning');
  const actions = before.at(-1);
  assert.equal(actions.className, actionClass);
  const [plugins, convert, clone] = actions.children;
  assert.equal(plugins.type, 'button');
  assert.equal(plugins.href, undefined);
  assert.equal(plugins.textContent, english['ui.70171ec7a356']);
  const consent = before.find(row => row.tag === 'label').children[1];
  assert.equal(consent.type, 'checkbox');
  assert.equal(clone.disabled, true);
  consent.checked = true;
  consent.events.change();
  assert.equal(convert.disabled, true);
  assert.equal(clone.disabled, false);

  plugins.click();
  assert.equal(nav.pluginsOpened, 1);
  assert.deepEqual(nav.selections(), {...selections, plugins: sourceID});
  assert.deepEqual(nav.selectionEvents.map(({workspace, serverId}) => [workspace, serverId]), [['plugins', sourceID]]);
  assert.equal(f.review.hidden, false);
  assert.deepEqual(f.review.children, before);
  assert.deepEqual(f.selected(), ['fabric']);
  f.cards[1].click();
  plugins.click();
  assert.equal(nav.pluginsOpened, 1);
  await flush();
});

const matrixID = 'a32d4725-8951-44cb-8b41-ace226d184a5';
const matrixProgress = {
  manifest_ready: true, completed: 7, total: 23, passed: 2, failed: 1,
  unsupported: 3, blocked: 1, unfinished: 16,
  source: 'paper', target: 'purpur', action: 'convert', cancel_requested: false,
};
const matrixSnapshot = (state, changes = {}) => ({
  enabled: true,
  operation: state ? {
    operation_id: matrixID, diagnostic_id: matrixID, state,
    reservation_active: ['preparing', 'running', 'closing', 'interrupted', 'needs_attention'].includes(state),
    recovery_required: ['interrupted', 'needs_attention'].includes(state),
    progress: state === 'preparing' ? {manifest_ready: false} : matrixProgress,
    ...changes,
  } : null,
});
const matrixOK = result => ({ ok: true, json: async () => result });
const matrixText = f => f.matrixStatus.children.filter(node => node.tag === 'p').map(node => node.textContent);
const matrixAction = (f, label) => f.matrixStatus.children
  .flatMap(node => node.children).find(node => node.tag === 'button' && node.textContent === label);

test('Switch matrix is capability-gated and uses standard buttons at the right of the existing legend', async () => {
  const html = read('templates/migration_workspace_switch.html');
  const css = read('static/server-migration-switch.css');
  assert.match(html, /class="migration-switch-legend">[\s\S]*?Current[\s\S]*?120d1307d24e[\s\S]*?e674447337e8[\s\S]*?<button type="button" data-switch-matrix-run hidden disabled>[\s\S]*?<\/button>\s*<\/div>/);
  assert.match(css, /\[data-switch-matrix-run\]\{margin-inline-start:auto;align-self:center\}/);
  assert.doesNotMatch(css, /\[data-switch-matrix-(?:run|status)\][^{]*\{[^}]*\b(?:font-size|padding|border-radius|min-height):/);
  assert.match(html, /migration-switch-content[^>]*>[\s\S]*data-switch-matrix-status/);

  for (const enabled of [false, true]) {
    let posts = 0;
    const f = fixture(() => ({operation: null}), plan, undefined, undefined, undefined, {
      request: (_url, options) => {
        if (options.method === 'POST') posts++;
        return matrixOK({ enabled, operation: null });
      },
    });
    await flush();
    assert.equal(f.matrixRun.hidden, !enabled);
    assert.deepEqual(matrixText(f), []);
    assert.equal(f.cards[1].disabled, false);
    if (!enabled) f.matrixRun.click();
    assert.equal(posts, 0);
  }
  const unresolved = fixture(() => ({operation: null}), plan, undefined, undefined, undefined, {
    request: () => matrixOK({...matrixSnapshot('interrupted'), enabled: false}),
  });
  await flush();
  assert.ok(unresolved.matrixRun.hidden);
  assert.ok(unresolved.cards.every(card => card.disabled));
  assert.match(matrixText(unresolved).join(' '), /Return to a JustVoxel development image/);
  assert.equal(matrixAction(unresolved, 'Recover Switch Test'), undefined);
  const failed = fixture(() => ({operation: null}), plan, undefined, undefined, undefined, {
    request: () => ({ok: false}),
  });
  await flush();
  assert.ok(failed.matrixRun.hidden);
  assert.deepEqual(matrixText(failed), ['Switch matrix status could not be read.']);
});

test('Switch matrix discovery, live progress, and graceful cancellation use persisted appliance status', async () => {
  let result = matrixSnapshot();
  let starts = 0;
  let stops = 0;
  let finishStop;
  const f = fixture(() => ({operation: null}), plan, undefined, undefined, undefined, {
    request: (url, options) => {
      if (url.endsWith('/start')) {
        starts++;
        assert.deepEqual([...options.body.keys()], ['csrf']);
        result = matrixSnapshot('preparing');
      }
      if (url.endsWith('/stop')) {
        stops++;
        assert.equal(options.body.get('operation_id'), matrixID);
        return new Promise(resolve => { finishStop = resolve; });
      }
      return matrixOK(result);
    },
  });
  await flush();
  assert.equal(f.matrixRun.disabled, false);
  await f.cards[1].click();
  await flush();
  const consent = f.review.children.find(node => node.tag === 'label').children[1];
  consent.checked = true;
  consent.events.change();
  const apply = f.review.children.at(-1).children.at(-1);
  assert.equal(apply.disabled, false);
  f.matrixRun.click();
  f.matrixRun.click();
  await flush();
  assert.equal(starts, 1);
  assert.equal(apply.disabled, true);
  assert.ok(f.cards.every(card => card.disabled));
  assert.deepEqual(matrixText(f), ['Discovering available releases from software publishers.']);
  assert.equal(f.matrixStatus.children.find(node => node.tag === 'progress'), undefined);

  result = matrixSnapshot('running');
  await f.timers.shift()();
  await flush();
  assert.deepEqual(matrixText(f), [
    'Switch matrix test in progress.', '7 / 23 completed', 'paper → purpur (Convert)',
    'Passed: 2 · Failed: 1 · Unsupported: 3 · Blocked: 1',
  ]);
  const bar = f.matrixStatus.children.find(node => node.tag === 'progress');
  assert.equal(bar.max, 23);
  assert.equal(bar.value, 7);
  const stop = matrixAction(f, 'Stop Test');
  assert.equal(stop.className, undefined);
  stop.click();
  stop.click();
  assert.equal(stops, 1);
  assert.equal(matrixAction(f, 'Stop Test').disabled, true);
  assert.doesNotMatch(matrixText(f).join(' '), /Cancellation requested/);
  result = matrixSnapshot('running', {progress: {...matrixProgress, cancel_requested: true}});
  finishStop(matrixOK(result));
  await flush();
  assert.match(matrixText(f).join(' '), /Cancellation requested\./);
  assert.equal(matrixAction(f, 'Stop Test').disabled, true);
  assert.equal(f.matrixRun.disabled, true);
  result = matrixSnapshot('cancelled', {progress: {...matrixProgress, source: '', target: '', cancel_requested: true}});
  while (f.timers.length) {
    await f.timers.shift()();
    await flush();
  }
  assert.match(matrixText(f).join(' '), /test cancelled.*Unfinished: 16/);
  assert.equal(f.matrixRun.disabled, false);
  assert.equal(apply.disabled, false);
  assert.equal(f.cards[1].disabled, false);
  assert.equal(matrixAction(f, 'Open Logs').className, undefined);
});

test('Switch matrix reconnects without an ordinary server and shows completed and recovery outcomes', async () => {
  for (const state of ['completed', 'interrupted', 'needs_attention', 'recovered']) {
    let requests = 0;
    const f = fixture(() => assert.fail('matrix should not request an ordinary server operation'), plan, undefined, undefined, undefined, {
      server: '',
      request: (url, options) => {
        requests++;
        if (url.endsWith('/recover')) {
          assert.equal(options.body.get('operation_id'), matrixID);
          return matrixOK(matrixSnapshot('recovered'));
        }
        assert.equal(options.method, 'GET');
        return matrixOK(matrixSnapshot(state));
      },
    });
    await flush();
    assert.deepEqual(f.operations, []);
    assert.equal(requests, 1);
    assert.deepEqual(f.timers, []);
    assert.ok(matrixAction(f, 'Open Logs'));
    if (state === 'completed') {
      assert.match(matrixText(f).join(' '), /completed with failures/);
      assert.equal(f.matrixRun.disabled, false);
    }
    if (state === 'interrupted' || state === 'needs_attention') {
      assert.match(matrixText(f).join(' '), /needs attention.*Unfinished: 16/);
      assert.equal(f.matrixRun.disabled, true);
      const recover = matrixAction(f, 'Recover Switch Test');
      assert.equal(recover.className, undefined);
      recover.click();
      await flush();
      assert.equal(requests, 2);
    }
    if (state !== 'completed') {
      assert.match(matrixText(f).join(' '), /cleanup recovered.*Unfinished: 16/);
      assert.equal(f.matrixRun.disabled, false);
    }
  }
  let reads = 0;
  const active = fixture(() => assert.fail('no ordinary server'), plan, undefined, undefined, undefined, {
    server: '',
    request: (_url, options) => {
      assert.equal(options.method, 'GET');
      reads++;
      return matrixOK(matrixSnapshot('running'));
    },
  });
  await flush();
  active.workspace.open = false;
  await active.timers.shift()();
  assert.equal(reads, 1);
  assert.equal(active.matrixRun.disabled, true);
});

function nativeMatrixLogs(context) {
  const nav = nativeNavigation(context);
  const system = new Element('dialog');
  system.dataset.workspaceWindow = 'system';
  system.open = false;
  system.show = () => { system.open = true; };
  const content = new Element('div');
  const tabs = ['health', 'logs'].map(name => {
    const tab = new Element('button');
    tab.dataset.systemTab = name;
    return tab;
  });
  system.querySelector = selector => ({
    '[data-system-workspace-content]': content,
    '[data-system-state]': new Element('p'),
  })[selector];
  system.querySelectorAll = selector => selector === '[data-system-tab]' ? tabs : [];
  const launcher = new Element('button');
  const find = context.document.querySelector;
  context.document.querySelector = selector => ({
    '[data-system-open]': launcher,
    '[data-system-workspace-dialog]': system,
  })[selector] ?? find(selector);
  Object.assign(context.document.body.classList, { remove() {}, toggle() {}, add() {} });
  context.jvS = value => value;
  context.formatDiagnosticLogTimestamp = () => '';
  const requests = [];
  const fetch = context.fetch;
  context.fetch = (url, options) => {
    if (url === '/api/session-info') return matrixOK({username: 'voxel', role: 'administrator'});
    if (url === '/api/ups') return matrixOK({available: false});
    if (url === '/api/system/workspace/logs') {
      requests.push(url);
      return matrixOK({logs: [{category: 'migration', log_id: matrixID, operation_type: 'switch_matrix'}]});
    }
    if (url === '/api/system/workspace/logs/migration/' + matrixID) {
      requests.push(url);
      return {ok: true, text: async () => 'structured matrix report'};
    }
    return fetch(url, options);
  };
  const systemSource = app.slice(app.indexOf('const systemWorkspaceOpen ='), app.indexOf('// Plugin workspace. All local inventory'));
  vm.runInNewContext(systemSource, context);
  return Object.assign(nav, {system, content, tabs, logRequests: requests});
}

test('Switch matrix Open Logs raises native System Settings, refreshes Migration, and preserves other workspaces', async () => {
  const f = fixture(() => ({operation: null}), plan, undefined, undefined, nativeMatrixLogs, {
    request: () => matrixOK(matrixSnapshot('completed')),
  });
  const nav = f.navigation;
  await flush();
  await nav.launcher.click();
  await flush();
  const selections = nav.selections();
  const url = nav.location.href;
  assert.ok(nav.dialog.open);
  matrixAction(f, 'Open Logs').click();
  await flush();
  assert.ok(nav.system.open);
  assert.equal(nav.tabs[1].attributes['aria-selected'], 'true');
  const group = nav.content.children[0].children[0].children.find(button => button.textContent === 'Migration');
  assert.equal(group.attributes['aria-pressed'], 'true');
  const requests = ['/api/system/workspace/logs', '/api/system/workspace/logs/migration/' + matrixID];
  assert.deepEqual(nav.logRequests, requests);
  const previousZ = Number(nav.system.style.zIndex);
  matrixAction(f, 'Open Logs').click();
  await flush();
  assert.ok(Number(nav.system.style.zIndex) > previousZ);
  assert.deepEqual(nav.logRequests, requests.concat(requests));
  assert.ok(f.workspace.open);
  assert.ok(nav.dialog.open);
  assert.deepEqual(nav.selections(), selections);
  assert.equal(nav.location.href, url);
});

test('Clone-only cards review exact exclusions and bind consent to the selected cross-play option', async () => {
  const submissions = [];
  const preserveRequests = [];
  const reviewPlan = (target, choice) => ({
    ...plan(target),
    crossplay_choice: choice || 'java',
    crossplay_options: ['java', 'preserve'],
    crossplay_install: choice === 'preserve' ? [{ directory: 'mods', filename: 'Geyser-Fabric.jar' }] : [],
    requires_exclusion_consent: true,
    plan_fingerprint: choice === 'preserve' ? 'reviewed-preserve' : 'reviewed-java',
    content: [
      { name: 'Working mod', directory: 'mods', filename: 'shared.jar', compatibility: 'verified', clone_disposition: 'retain', reason: 'Publisher verified' },
      { name: 'Old plugin', directory: 'plugins', filename: 'shared.jar', compatibility: 'unknown', clone_disposition: 'exclude', reason: 'Dependency missing' },
    ],
  });
  const f = fixture(() => ({ operation: null }), (target, choice) => {
    if (choice === 'preserve') {
      return new Promise(resolve => preserveRequests.push({ target, resolve }));
    }
    return reviewPlan(target, choice);
  }, body => {
    submissions.push(body);
    return { ok: false, json: async () => ({ error: 'Apply rejected' }) };
  });
  await flush();
  assert.equal(f.cards[2].disabled, false);
  await f.cards[2].click();
  await flush();
  assert.ok(f.review.children.some(node => node.textContent?.includes('plugins/shared.jar') && node.textContent.includes('Dependency missing')));
  assert.ok(f.review.children.some(node => node.textContent?.includes('mods/shared.jar') && node.textContent.includes('Retained in clone')));
  const reviewCheckboxes = () => f.review.children.filter(node => node.tag === 'label').map(node => node.children[1]);
  const preserveChoice = () => f.review.children.flatMap(node => node.children).find(node => node.textContent === 'Preserve cross-play');
  const [previousConfirm, previousExclusions] = reviewCheckboxes();
  const previousClone = f.review.children.at(-1).children.at(-1);
  previousConfirm.checked = true;
  previousConfirm.events.change();
  previousExclusions.checked = true;
  previousExclusions.events.change();
  assert.equal(previousClone.disabled, false);

  const selectCrossplay = preserveChoice();
  assert.ok(selectCrossplay);
  selectCrossplay.click();
  assert.deepEqual(preserveRequests.map(request => request.target), ['fabric']);
  assert.equal(f.review.hidden, true);
  await previousClone.click();
  assert.equal(submissions.length, 0);

  // Start a newer review for the same target and cross-play choice.
  f.cards[2].click();
  await flush();
  preserveChoice().click();
  assert.deepEqual(preserveRequests.map(request => request.target), ['fabric', 'fabric']);
  preserveRequests[1].resolve(reviewPlan('fabric', 'preserve'));
  await flush();
  assert.ok(f.review.children.some(node => node.textContent?.includes('Install in clone · mods/Geyser-Fabric.jar')));
  const currentReview = f.review.children.slice();
  preserveRequests[0].resolve({ ...reviewPlan('fabric', 'preserve'), plan_fingerprint: 'stale-preserve', warnings: ['Stale review'] });
  await flush();
  assert.equal(f.review.hidden, false);
  assert.deepEqual(f.review.children, currentReview);
  assert.deepEqual(f.selected(), ['fabric']);
  assert.equal(f.cards[2].disabled, false);

  const [confirm, exclusions] = reviewCheckboxes();
  assert.notEqual(exclusions, previousExclusions);
  assert.ok(!confirm.checked);
  assert.ok(!exclusions.checked);
  const [convert, clone] = f.review.children.at(-1).children.slice(-2);
  confirm.checked = true;
  confirm.events.change();
  assert.equal(convert.disabled, true);
  assert.equal(clone.disabled, true);
  await clone.click();
  assert.equal(submissions.length, 0);
  exclusions.checked = true;
  exclusions.events.change();
  assert.equal(clone.disabled, false);
  assert.equal(convert.disabled, true);
  await convert.click();
  assert.equal(submissions.length, 0);
  await previousClone.click();
  assert.equal(submissions.length, 0);
  await clone.click();
  assert.equal(submissions.length, 1);
  const [body] = submissions;
  assert.equal(body.get('target'), 'fabric');
  assert.equal(body.get('action'), 'clone');
  assert.equal(body.get('plan_fingerprint'), 'reviewed-preserve');
  assert.equal(body.get('crossplay'), 'preserve');
  assert.equal(body.get('accept_exclusions'), 'yes');
});
