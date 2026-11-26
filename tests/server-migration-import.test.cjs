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

const mutationObservers = new Set();

class MutationObserver {
  constructor(callback) {
    this.callback = callback;
    this.records = [];
  }
  observe(target, options) {
    this.target = target;
    this.options = options;
    mutationObservers.add(this);
  }
  disconnect() {
    mutationObservers.delete(this);
    this.takeRecords();
  }
  takeRecords() { return this.records.splice(0); }
  recordRemoval(target, removedNode) {
    if (!this.options.childList) return;
    if (this.target !== target && !(this.options.subtree && this.target.contains(target))) return;
    this.records.push({ removedNodes: [removedNode] });
    if (this.records.length === 1) {
      queueMicrotask(() => {
        const records = this.takeRecords();
        if (records.length) this.callback(records);
      });
    }
  }
}

class Element extends EventTarget {
  constructor() {
    super();
    this.children = [];
    this.attributes = new Map();
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
  append(...nodes) {
    for (const node of nodes) {
      if (node.parentElement) node.remove();
      this.children.push(node);
      node.parentElement = this;
    }
  }
  appendChild(node) { this.append(node); }
  replaceChildren(...nodes) { this.children.forEach(node => node.remove()); this.append(...nodes); }
  remove() {
    if (!this.parentElement) return;
    const parent = this.parentElement;
    parent.children = parent.children.filter(node => node !== this);
    this.parentElement = null;
    for (const observer of mutationObservers) observer.recordRemoval(parent, this);
  }
  contains(node) { return this === node || this.children.some(child => child.contains(node)); }
  get isConnected() { return this.isDocument || Boolean(this.parentElement?.isConnected); }
  querySelector() { return null; }
  querySelectorAll() { return []; }
  setAttribute(name, value) { this.attributes.set(name, value); }
  getAttribute(name) { return this.attributes.get(name) ?? null; }
  showModal() { this.open = true; }
  close() {
    if (!this.open) return;
    this.open = false;
    setImmediate(() => this.dispatchEvent(new Event('close')));
  }
  focus() {}
  click() { this.dispatchEvent(new Event('click')); }
}

function migrationRoot(server, action = 'review', path = importPath) {
  const root = new Element();
  const submitter = new Element();
  const form = new Element();
  form.method = 'post';
  form.getAttribute = () => `${path}/${action}?server=${server}`;
  form.closest = selector => selector === 'form' ? form : null;
  root.dataset.serverId = server;
  root.contains = node => node === form;
  let submit;
  root.addEventListener = (type, listener) => { if (type === 'submit') submit = listener; };
  return { root, submitter, submit: () => submit({ target: form, submitter, preventDefault() {} }) };
}

async function fixture(review, tab = 'import') {
  const path = '/workspace/migration/' + tab;
  const action = tab === 'import' ? 'review' : 'apply';
  const first = migrationRoot('jv-first01', action, path);
  const next = migrationRoot('jv-second1', action, path);
  const reselected = migrationRoot('jv-first01', action, path);
  const reviewed = migrationRoot('jv-first01', 'apply', path);
  const nextReviewed = migrationRoot('jv-second1', 'apply', path);
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
  const document = Object.assign(new Element(), {
    body,
    isDocument: true,
    querySelector: selector => selector === '[data-migration-open]' ? open : migration,
    createElement(tag) {
      const node = new Element();
      if (tag === 'div') node.querySelector = () => roots[node.innerHTML];
      return node;
    },
  });
  document.appendChild(body);
  const requests = [];
  const loads = [];
  const controllers = [];
  let abortNextController = false;
  const sharedControlsCalls = [];
  const generation = { migration: 0 };
  migration.querySelector = selector => ({
    '[data-migration-close]': close,
    '[data-migration-refresh]': refresh,
    '[data-migration-state]': state,
    '[data-migration-workspace-content]': content,
  })[selector];

  const window = new EventTarget();
  window.location = new URL(`/?workspace=migration&tab=${tab}&server=jv-first01`, origin);
  for (const name of ['Export', 'Import', 'Operation', 'Switch']) {
    window['JustVoxelServerMigration' + name] = { init() {} };
  }
  const context = {
    window, URL, URLSearchParams, document, MutationObserver, controlCenter: null,
    AbortController: class extends AbortController {
      constructor() {
        super();
        controllers.push(this);
        if (abortNextController) {
          abortNextController = false;
          this.abort();
        }
      }
    },
    readWorkspaceWindowState: () => ({}),
    setupWorkspaceWindow: (element, { onOpen, onClose }) => ({
      open() { element.open = true; return onOpen(); },
      close() { element.close(); onClose(); },
    }),
    initializeSharedWebUIControls(root) {
      sharedControlsCalls.push(root);
      assert.fail('The transient SMB prompt must skip shared WebUI controls');
    },
    justVoxelServerSelection: async () => {},
    justVoxelSelectionGeneration: generation,
    justVoxelSelectedURL: url => url,
    FormData: class {
      forEach(callback) { callback('smb', tab === 'import' ? 'source_kind' : 'kind'); }
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
        ok: true, status: 200, url: origin + path,
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
  return { first, next, reselected, reviewed, nextReviewed, content, body, document, credentials, requests, loads, migration, assertIdle, sharedControlsCalls,
    submitSignal: () => controllers.at(-1).signal,
    abortNextSubmit: () => { abortNextController = true; },
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

test('the transient SMB password input is ready at attachment and stays in its label', async () => {
  const f = await fixture();
  let attachedInput;
  let attachedLabel;
  let inputRemovals = 0;
  const appendChild = f.body.appendChild.bind(f.body);
  f.body.appendChild = dialog => {
    const [, , label, actions] = dialog.children[0].children;
    const [input] = label.children;
    assert.equal(actions.children.length, 2);
    assert.equal(input.parentElement, label);
    assert.equal(input.type, 'password');
    assert.equal(input.autocomplete, 'off');
    assert.equal(input.getAttribute('data-bwignore'), 'true');
    assert.equal(input.getAttribute('data-password-ui'), 'off');
    attachedInput = input;
    attachedLabel = label;
    const remove = input.remove.bind(input);
    input.remove = () => {
      inputRemovals++;
      remove();
    };
    appendChild(dialog);
  };

  const submitted = f.first.submit();
  await flush();
  const prompt = f.credentials();
  assert.equal(prompt.input, attachedInput);
  assert.equal(prompt.input.parentElement, attachedLabel);
  prompt.cancel.click();
  await submitted;
  assert.equal(inputRemovals, 0);
  assert.deepEqual(f.sharedControlsCalls, []);
  assert.equal(f.body.children.length, 0);
});

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
  assert.equal(f.credentials().input, prompt.input);
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
  assert.equal(firstPrompt.dialog.open, false);
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

for (const [reason, dismiss] of [
  ['unexpected removal', (_, { dialog }) => dialog.remove()],
  ['immediate reinsertion', (f, { dialog }) => {
    dialog.remove();
    f.body.appendChild(dialog);
    assert.equal(dialog.isConnected, true);
    assert.equal(dialog.open, true);
  }],
  ['immediate subtree reinsertion', f => {
    f.body.remove();
    f.document.appendChild(f.body);
  }],
  ['external close', (_, { dialog }) => dialog.close()],
  ['reinsertion followed by a click before mutation delivery', (f, { dialog, use }) => {
    dialog.remove();
    f.body.appendChild(dialog);
    use.click();
  }],
]) {
  test(`SMB Import cancels after ${reason} and allows a fresh review`, async () => {
    const f = await fixture();
    let resolved = false;
    const submitted = f.first.submit().then(() => { resolved = true; });
    const signal = f.submitSignal();
    const prompt = f.credentials();
    prompt.input.value = 'discarded';
    assert.equal(f.first.submitter.disabled, true);

    dismiss(f, prompt);
    await flush();
    assert.equal(resolved, true);
    await submitted;
    assert.equal(f.requests.length, 0);
    assert.equal(f.first.submitter.disabled, false);
    assert.equal(f.content.children[0], f.first.root);
    assert.equal(prompt.dialog.open, false);
    assert.equal(prompt.dialog.isConnected, false);
    assert.equal(prompt.input.value, '');
    assert.equal([...mutationObservers].some(observer => observer.target === f.document), false);
    f.assertIdle();

    const retry = f.first.submit();
    // Starting another submit would abort the previous controller if it still owned the slot.
    assert.equal(signal.aborted, false);
    const { input, use } = f.credentials();
    input.value = ' exact review password ';
    use.click();
    await retry;
    assert.equal(f.requests.length, 1);
    assert.equal(f.requests[0].url, importPath + '/review?server=jv-first01');
    assert.equal(f.requests[0].body.get('source_smb_password'), ' exact review password ');
    assert.equal(f.content.children[0], f.reviewed.root);
    assert.equal(f.first.submitter.disabled, false);
    f.assertIdle();
  });
}

test('an already-aborted SMB Import submit settles without attaching a prompt', async () => {
  const f = await fixture();
  f.body.appendChild = () => assert.fail('An aborted prompt must not attach');
  f.abortNextSubmit();
  await f.first.submit();
  assert.equal(f.submitSignal().aborted, true);
  assert.equal(f.first.submitter.disabled, false);
  assert.equal(f.requests.length, 0);
  f.assertIdle();
});

test('SMB Export cancels on prompt removal and sends only a fresh temporary password on retry', async () => {
  const f = await fixture(undefined, 'export');
  const submitted = f.first.submit();
  const prompt = f.credentials();
  prompt.input.value = 'discarded';
  prompt.dialog.remove();
  await submitted;
  assert.equal(f.first.submitter.disabled, false);
  assert.equal(f.requests.length, 0);
  f.assertIdle();

  const retry = f.first.submit();
  const { input, use } = f.credentials();
  input.value = 'export-password';
  use.click();
  await retry;
  assert.equal(f.requests.length, 1);
  assert.equal(f.requests[0].url, '/workspace/migration/export/apply?server=jv-first01');
  assert.equal(f.requests[0].body.get('smb_password'), 'export-password');
  assert.equal(f.content.children[0], f.reviewed.root);
  assert.equal(f.first.submitter.disabled, false);
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
    assert.equal(prompt.dialog.open, false);
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
