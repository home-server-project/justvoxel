const assert = require('node:assert/strict');
const { readFileSync } = require('./helpers/webui-i18n.cjs');
const { test } = require('node:test');
const vm = require('node:vm');

const app = readFileSync(`${__dirname}/../webui/internal/server/static/app.js`, 'utf8');
const serverIDValidator = app.match(/^const validServerID = .*;$/m)[0];
const workspace = app.slice(app.indexOf('const migrationOpen ='), app.indexOf('const systemMonitorOpen ='));
const flush = () => new Promise(resolve => setImmediate(resolve));
const origin = 'https://justvoxel.test';
const importPath = '/workspace/migration/import';
const selectionEvent = 'justvoxel:server-selection-changed';

class Element extends EventTarget {
  constructor() {
    super();
    this.children = [];
    this.dataset = {};
    this.disabled = false;
    this.textContent = '';
    const classes = new Set();
    this.classList = {
      add: name => classes.add(name),
      remove: (...names) => names.forEach(name => classes.delete(name)),
      contains: name => classes.has(name),
      toggle: (name, on) => on ? classes.add(name) : classes.delete(name),
    };
  }
  append(...nodes) { this.children.push(...nodes); nodes.forEach(node => { node.parentElement = this; }); }
  appendChild(node) { this.append(node); }
  replaceChildren(...nodes) { this.children.forEach(node => node.remove()); this.append(...nodes); }
  remove() {
    if (!this.parentElement) return;
    this.parentElement.children = this.parentElement.children.filter(node => node !== this);
    this.parentElement = null;
  }
  querySelector() { return null; }
  querySelectorAll() { return []; }
  setAttribute() {}
  showModal() { this.open = true; }
  close() { this.open = false; }
  focus() {}
  click() { this.dispatchEvent(new Event('click')); }
}

function importRoot(server, action = 'review') {
  const root = new Element();
  const submitter = new Element();
  const form = new Element();
  form.method = 'post';
  form.getAttribute = () => `${importPath}/${action}?server=${server}`;
  form.closest = selector => selector === 'form' ? form : null;
  root.dataset.serverId = server;
  root.contains = node => node === form;
  let submit;
  root.addEventListener = (type, listener) => { if (type === 'submit') submit = listener; };
  return { root, submitter, submit: () => submit({ target: form, submitter, preventDefault() {} }) };
}

async function fixture(review) {
  const first = importRoot('jv-first01');
  const next = importRoot('jv-second1');
  const reselected = importRoot('jv-first01');
  const reviewed = importRoot('jv-first01', 'apply');
  const nextReviewed = importRoot('jv-second1', 'apply');
  const roots = {
    first: first.root,
    next: next.root,
    reselected: reselected.root,
    reviewed: reviewed.root,
    nextReviewed: nextReviewed.root,
  };
  const body = new Element();
  const migration = new Element();
  const open = new Element();
  const close = new Element();
  const state = new Element();
  const content = new Element();
  const refresh = new Element();
  const requests = [];
  const loads = [];
  const generation = { migration: 0 };
  migration.querySelector = selector => ({
    '[data-migration-close]': close,
    '[data-migration-refresh]': refresh,
    '[data-migration-state]': state,
    '[data-migration-workspace-content]': content,
  })[selector];

  const window = new EventTarget();
  window.location = new URL('/?workspace=migration&tab=import&server=jv-first01', origin);
  for (const name of ['Export', 'Import', 'Operation', 'Switch']) {
    window['JustVoxelServerMigration' + name] = { init() {} };
  }
  const context = {
    window, URL, URLSearchParams, AbortController, controlCenter: null,
    document: {
      body,
      addEventListener() {},
      querySelector: selector => selector === '[data-migration-open]' ? open : migration,
      createElement(tag) {
        const node = new Element();
        if (tag === 'div') node.querySelector = () => roots[node.innerHTML];
        return node;
      },
    },
    readWorkspaceWindowState: () => ({}),
    setupWorkspaceWindow: (element, { onOpen, onClose }) => ({
      open() { element.open = true; return onOpen(); },
      close() { element.close(); onClose(); },
    }),
    initializeSharedWebUIControls() {},
    justVoxelServerSelection: async () => {},
    justVoxelSelectionGeneration: generation,
    justVoxelSelectedURL: url => url,
    FormData: class {
      forEach(callback) { callback('smb', 'source_kind'); }
    },
    fetch: async (url, options) => {
      if (options.method === 'POST') {
        requests.push({ url, body: new URLSearchParams(options.body), signal: options.signal });
        await review?.();
        const server = new URL(url, origin).searchParams.get('server');
        const markup = server === next.root.dataset.serverId ? 'nextReviewed' : 'reviewed';
        return { ok: true, status: 200, url: origin + url, text: async () => markup };
      }
      loads.push(url);
      const markup = window.location.searchParams.get('server') === next.root.dataset.serverId
        ? 'next' : loads.length === 1 ? 'first' : 'reselected';
      return {
        ok: true, status: 200, url: origin + importPath,
        text: async () => markup,
      };
    },
  };
  vm.runInNewContext(serverIDValidator + '\n' + workspace, context);
  open.click();
  await flush();
  assert.equal(content.children[0], first.root);
  const credentials = () => {
    const dialog = body.children[0];
    assert.equal(dialog.open, true);
    const [, , label, actions] = dialog.children[0].children;
    const [input] = label.children;
    const [cancel, use] = actions.children;
    return { dialog, input, cancel, use };
  };
  const assertIdle = () => {
    assert.equal(migration.classList.contains('is-busy'), false);
    assert.equal(state.classList.contains('is-busy'), false);
    assert.equal(state.textContent, '');
    assert.equal(refresh.disabled, false);
  };
  return { first, next, reselected, reviewed, nextReviewed, content, body, credentials, requests, loads, migration, assertIdle,
    close: () => close.click(),
    invalidate: () => { generation.migration++; },
    changeServer(serverId = next.root.dataset.serverId) {
      generation.migration++;
      const detail = { workspace: 'migration' };
      if (serverId !== null) detail.serverId = serverId;
      if (serverId === first.root.dataset.serverId || serverId === next.root.dataset.serverId) {
        window.location.searchParams.set('server', serverId);
      }
      window.dispatchEvent(Object.assign(new Event(selectionEvent), { detail }));
    },
  };
}

test('same-server synchronization preserves the SMB Import prompt and review ownership', async () => {
  let finishReview;
  const f = await fixture(() => new Promise(resolve => { finishReview = resolve; }));
  const serverId = f.first.root.dataset.serverId;
  const submitted = f.first.submit();
  const prompt = f.credentials();
  prompt.input.value = 'partial-password';

  f.changeServer(serverId);
  await flush();
  assert.equal(f.credentials().dialog, prompt.dialog);
  assert.equal(prompt.input.value, 'partial-password');
  assert.equal(f.first.submitter.disabled, true);
  assert.equal(f.requests.length, 0);
  assert.equal(f.content.children[0], f.first.root);
  assert.equal(f.loads.length, 1);

  prompt.input.value = 'review-password';
  prompt.use.click();
  await flush();
  assert.equal(f.requests.length, 1);
  assert.equal(f.requests[0].url, importPath + '/review?server=' + serverId);
  assert.equal(f.requests[0].body.get('source_smb_password'), 'review-password');
  assert.equal(prompt.input.value, '');
  assert.equal(f.body.children.length, 0);

  f.changeServer(serverId);
  await flush();
  assert.equal(f.first.submitter.disabled, true);
  assert.equal(f.requests[0].signal.aborted, false);
  assert.equal(f.migration.classList.contains('is-busy'), true);
  assert.equal(f.content.children[0], f.first.root);
  assert.equal(f.loads.length, 1);

  finishReview();
  await submitted;
  assert.equal(f.requests.length, 1);
  assert.equal(f.first.submitter.disabled, false);
  assert.equal(f.content.children[0], f.reviewed.root);
  f.assertIdle();

  f.changeServer(serverId);
  await flush();
  assert.equal(f.content.children[0], f.reviewed.root);
  assert.equal(f.loads.length, 1);
  const continued = f.reviewed.submit();
  assert.equal(f.body.children.length, 0);
  assert.equal(f.requests.length, 2);
  assert.equal(f.requests[1].url, importPath + '/apply?server=' + serverId);
  assert.equal(f.requests[1].body.get('source_smb_password'), 'review-password');
  finishReview();
  await continued;
  f.assertIdle();
});

test('SMB Import waits for credentials, cancels on server change, and can review the new selection', async () => {
  const f = await fixture();
  const firstSubmit = f.first.submit();
  const firstPrompt = f.credentials();
  firstPrompt.input.value = 'discarded';
  f.assertIdle();
  assert.equal(f.requests.length, 0);

  f.changeServer();
  await firstSubmit;
  await flush();
  f.assertIdle();
  assert.equal(f.requests.length, 0);
  assert.equal(f.body.children.length, 0);
  assert.equal(firstPrompt.input.value, '');
  assert.equal(f.first.submitter.disabled, false);
  assert.equal(f.content.children[0], f.next.root);
  assert.equal(f.next.submitter.disabled, false);

  const nextSubmit = f.next.submit();
  const { input, use } = f.credentials();
  f.assertIdle();
  assert.equal(f.requests.length, 0);
  input.value = 'review-password';
  use.click();
  await Promise.resolve();
  assert.equal(f.migration.classList.contains('is-busy'), true);
  await nextSubmit;
  assert.equal(f.requests.length, 1);
  assert.equal(f.requests[0].url, importPath + '/review?server=jv-second1');
  assert.equal(f.requests[0].body.get('source_smb_password'), 'review-password');
  assert.equal(input.value, '');
  assert.equal(f.content.children[0], f.nextReviewed.root);
  f.assertIdle();
});

for (const [reason, cancel] of [
  ['Cancel', (_, prompt) => prompt.cancel.click()],
  ['Escape', (_, prompt) => prompt.dialog.dispatchEvent(new Event('cancel', { cancelable: true }))],
  ['workspace close', f => f.close()],
  ['missing server identity', f => f.changeServer(null)],
  ['invalid server identity', f => f.changeServer('invalid')],
  ['stale credentials', (f, prompt) => { f.invalidate(); prompt.use.click(); }],
]) {
  test(`SMB Import releases its submit control after ${reason}`, async () => {
    const f = await fixture();
    const submitted = f.first.submit();
    const prompt = f.credentials();
    prompt.input.value = 'discarded';
    cancel(f, prompt);
    await submitted;
    await flush();
    f.assertIdle();
    assert.equal(f.requests.length, 0);
    assert.equal(f.body.children.length, 0);
    assert.equal(prompt.input.value, '');
    assert.equal(f.first.submitter.disabled, false);
  });
}

test('an aborted SMB Import review releases its busy state and submit control', async () => {
  const f = await fixture(() => { throw new DOMException('Review aborted', 'AbortError'); });
  const submitted = f.first.submit();
  const { input, use } = f.credentials();
  input.value = 'review-password';
  use.click();
  await submitted;
  assert.equal(f.requests.length, 1);
  assert.equal(f.first.submitter.disabled, false);
  f.assertIdle();
});

test('a stale SMB review response leaves the new server and its verification state alone', async () => {
  const pending = [];
  const f = await fixture(() => new Promise(resolve => pending.push(resolve)));
  const beginReview = (form) => {
    const submitted = form.submit();
    const { input, use } = f.credentials();
    input.value = 'review-password';
    use.click();
    return submitted;
  };

  const first = beginReview(f.first);
  await Promise.resolve();
  f.changeServer();
  await flush();
  const next = beginReview(f.next);
  await Promise.resolve();
  assert.equal(pending.length, 2);

  pending[0]();
  await first;
  assert.equal(f.content.children[0], f.next.root);
  assert.equal(f.migration.classList.contains('is-busy'), true);
  assert.equal(f.next.submitter.disabled, true);

  pending[1]();
  await next;
  assert.equal(f.content.children[0], f.nextReviewed.root);
  f.assertIdle();
});

test('an old SMB review response stays stale after switching A to B and back to A', async () => {
  let finishReview;
  const f = await fixture(() => new Promise(resolve => { finishReview = resolve; }));
  const submitted = f.first.submit();
  const { input, use } = f.credentials();
  input.value = 'review-password';
  use.click();
  await flush();
  assert.equal(f.requests.length, 1);

  f.changeServer();
  await flush();
  assert.equal(f.content.children[0], f.next.root);
  f.changeServer(f.first.root.dataset.serverId);
  await flush();
  assert.equal(f.content.children[0], f.reselected.root);
  assert.equal(f.reselected.root.dataset.serverId, f.first.root.dataset.serverId);

  finishReview();
  await submitted;
  assert.equal(f.content.children[0], f.reselected.root);
  assert.equal(f.requests.length, 1);
  f.assertIdle();
});
