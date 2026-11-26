const assert = require('node:assert/strict');
const { readFileSync } = require('node:fs');
const path = require('node:path');
const { test } = require('node:test');
const vm = require('node:vm');

const source = readFileSync(path.join(__dirname, '../webui/internal/server/static/server-migration-switch.js'), 'utf8');
const english = JSON.parse(readFileSync(path.join(__dirname, '../webui/internal/server/locales/en.json'), 'utf8'));
const flush = () => new Promise(resolve => setImmediate(resolve));
const snapshot = (state, target = 'leaf') => ({
  target, operation: { operation_id: 'switch-1', state, status: state },
});
const plan = target => ({ current: 'paper', target, version: '1.21.4', clone: true, plan_fingerprint: 'reviewed' });

class Element {
  constructor(tag) {
    this.tag = tag;
    this.dataset = {};
    this.events = {};
    this.children = [];
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
  querySelectorAll(tag) { return this.children.filter(node => node.tag === tag); }
  addEventListener(event, listener) { this.events[event] = listener; }
  setAttribute() {}
}

function fixture(operation, readPlan = plan, apply, recover) {
  const state = new Element('p');
  const progress = new Element('div');
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
  const generation = { migration: 1 };
  const root = new Element('section');
  root.dataset.serverId = 'jv-server';
  root.isConnected = true;
  root.matches = selector => selector === '[data-migration-switch]';
  root.querySelectorAll = () => cards;
  root.querySelector = selector => ({
    '[data-switch-state]': state, '[data-switch-operation]': progress,
    '[data-switch-review]': review, '[data-switch-csrf]': { value: 'csrf' },
  })[selector];
  const context = {
    window: {}, URLSearchParams, justVoxelSelectionGeneration: generation, jvT: key => english[key] ?? key,
    document: { createElement: tag => new Element(tag) },
    setTimeout: callback => timers.push(callback),
    fetch: async (request, options) => {
      const url = new URL(request, 'https://justvoxel.example');
      const action = url.pathname.split('/').at(-1);
      let result;
      if (action === 'plan') {
        result = await readPlan(url.searchParams.get('target'));
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
  vm.runInNewContext(source, context);
  context.window.JustVoxelServerMigrationSwitch.init(root);
  const selected = () => cards.filter(card => card.classList.contains('is-selected')).map(card => card.dataset.switchTarget);
  return { cards, root, state, review, progress, timers, operations, generation, selected };
}

test('durable Switch target stays selected through polling and a terminal reload', async () => {
  let state = 'validating';
  const f = fixture(() => snapshot(state));
  // The server's result must clear a previously selected unrelated card.
  f.cards[2].classList.toggle('is-selected', true);
  await flush();
  assert.deepEqual(f.selected(), ['leaf']);
  for (state of ['running', 'verifying', 'resolved']) {
    assert.equal(f.timers.length, 1);
    await f.timers.shift()();
    assert.deepEqual(f.selected(), ['leaf']);
    assert.equal(f.progress.children[0].textContent, state);
    assert.equal(f.review.hidden, true);
  }
  assert.deepEqual(f.operations, ['', 'switch-1', 'switch-1', 'switch-1', 'switch-1']);
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
      assert.equal(row.className, 'action-row');
      assert.deepEqual(row.children.map(node => node.tag), ['span', 'button']);
      const [description, button] = row.children;
      assert.equal(description.textContent, 'Restore original server state and clean up failed clone.');
      assert.equal(button.textContent, 'Recover Switch');
      assert.equal(button.disabled, false);
      const request = button.events.click();
      assert.equal(button.disabled, true);
      await button.events.click();
      assert.equal(recoveryRequests, 1);
      state = 'resolved';
      finishRecovery(snapshot(state));
      await request;
      assert.deepEqual(f.operations, ['', 'switch-1']);
      assert.deepEqual(f.selected(), ['leaf']);
      assert.equal(f.progress.children[0].textContent, 'resolved');
      assert.equal(f.review.hidden, true);
    }
  }
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
