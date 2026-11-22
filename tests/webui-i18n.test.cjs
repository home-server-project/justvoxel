'use strict';
const {test} = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const {install, englishPresentationSource} = require('./helpers/webui-i18n.cjs');
const root = path.resolve(__dirname, '../webui/internal/server');
const english = JSON.parse(fs.readFileSync(path.join(root, 'locales/en.json'), 'utf8'));
const keyFor = text => {
  const key = Object.keys(english).find(key => english[key] === text);
  assert.ok(key, `missing canonical English copy: ${text}`);
  return key;
};
test('shared VM and eval runtime uses canonical messages, opaque interpolation and source lookup', () => {
  const context = vm.createContext({key: keyFor('Delete {v0}'), values: {v0: 'Close {v1}<world>'}});
  assert.equal(vm.runInContext('jvT(key, values)', context), 'Delete Close {v1}<world>');
  assert.equal(vm.runInNewContext('jvS("Close")'), 'Close');
  assert.equal(vm.runInContext('jvS("unlisted provider description")', context), 'unlisted provider description');
  assert.equal(vm.runInContext('jvC("Close")', context), 'Close');
  assert.equal(vm.runInContext('jvH(key, {v0: "&lt;world&gt;"})', context), 'Delete &lt;world&gt;');
  assert.equal(vm.runInContext('eval("jvT(key, values)")', context), 'Delete Close {v1}<world>');
  assert.equal(new Function('key', 'return jvT(key);')(keyFor('Close')), 'Close');
  assert.equal(install(context), context, 'install is safe to repeat');
});
test('source assertions resolve the referenced catalog entry and reject missing keys', () => {
  const key = keyFor('Close');
  assert.equal(englishPresentationSource(`aria-label="{{t "${key}"}}"`, 'fixture.html'), 'aria-label="Close"');
  assert.throws(() => englishPresentationSource('{{t "missing.key"}}', 'fixture.html'), /Unknown English catalog key/);
});
test('production runtime retains source translation and safe HTML interpolation', () => {
  const messages = {...english, [keyFor('Delete {v0}')]: 'Supprimer {v0}'};
  for (const [key, source] of Object.entries(english)) {
    if (source === 'Close') messages[key] = 'Fermer';
  }
  const context = {justVoxelLanguage: {locale: 'fr-CA', source: english, messages}, document: {addEventListener() {}}};
  // Explicit globals prevent the English test prelude from replacing this locale.
  for (const name of ['jvT','jvS','jvH','jvC']) Object.defineProperty(context, name, {configurable: true, value: undefined});
  vm.runInNewContext(fs.readFileSync(path.join(root, 'static/i18n.js'), 'utf8'), context);
  assert.equal(context.jvS('Close'), 'Fermer');
  assert.equal(context.jvC('Fermer'), 'Close');
  assert.equal(context.jvT(keyFor('Delete {v0}'), {v0: 'Close {v1}'}), 'Supprimer Close {v1}');
  assert.equal(context.jvH(keyFor('Delete {v0}'), {v0: '&lt;world&gt;'}), 'Supprimer &lt;world&gt;');
});

test('VM prelude exposes English language and preserves fixture language objects', () => {
  const context = vm.createContext({});
  assert.equal(context.justVoxelLanguage.locale, 'en');
  assert.deepEqual(context.justVoxelLanguage.source, english);
  assert.deepEqual(context.justVoxelLanguage.messages, english);
  const language = {locale: 'fr-CA', source: english, messages: {}};
  assert.equal(install({justVoxelLanguage: language}).justVoxelLanguage, language);
});
test('duplicate English sources have identical translations in each locale', () => {
  for (const filename of fs.readdirSync(path.join(root, 'locales')).filter(name => name.endsWith('.json'))) {
    const messages = JSON.parse(fs.readFileSync(path.join(root, 'locales', filename), 'utf8'));
    const translations = new Map();
    for (const [key, source] of Object.entries(english)) {
      if (translations.has(source)) assert.equal(messages[key], translations.get(source), `${filename}: conflicting translation for ${JSON.stringify(source)} (${key})`);
      translations.set(source, messages[key]);
    }
  }
});
