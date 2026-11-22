// Shared test-only English presentation source and the real browser i18n runtime.
'use strict';
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const root = path.resolve(__dirname, '../../webui/internal/server');
const source = JSON.parse(fs.readFileSync(path.join(root, 'locales/en.json'), 'utf8'));
const browserRuntime = fs.readFileSync(path.join(root, 'static/i18n.js'), 'utf8');
const runInNewContext = vm.runInNewContext.bind(vm);
const runInContext = vm.runInContext.bind(vm);
const createContext = vm.createContext.bind(vm);
const runtime = {justVoxelLanguage: {locale: 'en', source, messages: source}, document: {addEventListener() {}}};
runInNewContext(browserRuntime, runtime);
function install(context = globalThis) {
  if (!Object.hasOwn(context, 'justVoxelLanguage')) context.justVoxelLanguage = runtime.justVoxelLanguage;
  for (const name of ['jvT', 'jvS', 'jvH', 'jvC']) {
    if (!Object.hasOwn(context, name)) context[name] = runtime[name];
  }
  return context;
}
// Go preludes and standalone Node fixtures share these VM entry points. Install
// only language and translation globals; retain every fixture's DOM and capability mocks.
vm.createContext = (context = {}, ...options) => createContext(install(context), ...options);
vm.runInNewContext = (code, context = {}, ...options) => runInNewContext(code, install(context), ...options);
vm.runInContext = (code, context, ...options) => runInContext(code, install(context), ...options);
install();
const escapeHTML = text => text.replace(/[&<>"']/g, char => ({'&':'&amp;', '<':'&lt;', '>':'&gt;', '"':'&#34;', "'":'&#39;'}[char]));
function message(key) {
  if (!Object.hasOwn(source, key)) throw new Error(`Unknown English catalog key: ${key}`);
  return source[key];
}
// Resolve only static catalog calls for existing exact source-copy assertions.
// Interpolated calls project to equivalent canonical template strings.
function englishPresentationSource(text, filename) {
  if (filename.endsWith('.html')) {
    return text.replace(/\{\{t "([^"]+)" "([^"]+)" (\.[A-Za-z0-9.]+|\(ts \.[A-Za-z0-9.]+\))\}\}/g, (_, key, name, expression) =>
      escapeHTML(message(key)).replaceAll('{' + name + '}', '{{' + expression.replace(/^\(ts /, '').replace(/\)$/, '') + '}}'))
      .replace(/\{\{t "([^"]+)"\}\}/g, (_, key) => escapeHTML(message(key)))
      .replace(/\{\{ts ([^}]+)\}\}/g, '{{$1}}');
  }
  if (filename.endsWith('.js')) {
    const singleQuotes = (text.match(/'/g) || []).length > (text.match(/"/g) || []).length;
    return text.replace(/\$\{jvH\("([^"]+)"\)\}/g, (_, key) => escapeHTML(message(key)))
      .replace(/jvT\("([^"]+)"\)/g, (_, key) => singleQuotes
        ? "'" + message(key).replace(/\\/g, '\\\\').replace(/'/g, "\\'").replace(/\n/g, '\\n').replace(/\r/g, '\\r') + "'"
        : JSON.stringify(message(key)))
      .replace(/jvT\("([^"]+)", \{([^\n{}]*)\}\)/g, (call, key, args) => {
        const values = [...args.matchAll(/([A-Za-z][A-Za-z0-9_]*): \((.*?)\)(?:, |$)/g)];
        if (!values.length) return call;
        const replacements = new Map(values.map(([, name, expression]) => [name, '${' + expression + '}']));
        return '`' + message(key).replace(/\\/g, '\\\\').replace(/`/g, '\\`')
          .replace(/\{([A-Za-z][A-Za-z0-9_]*)\}/g, (token, name) => replacements.get(name) ?? token) + '`';
      });
  }
  return text;
}
function readFileSync(filename, ...options) {
  const data = fs.readFileSync(filename, ...options);
  // Binary reads, hashes, backend source and external fixtures remain untouched.
  if (typeof data !== 'string' || typeof filename !== 'string') return data;
  const resolved = path.resolve(filename);
  if (!resolved.startsWith(root + path.sep)) return data;
  return englishPresentationSource(data, resolved);
}
module.exports = {...fs, readFileSync, install, englishPresentationSource};
