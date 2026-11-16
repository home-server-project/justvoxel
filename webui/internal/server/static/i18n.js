// One embedded catalog contract for server-rendered and authored browser messages.
(() => {
  'use strict';
  const language = globalThis.justVoxelLanguage;
  Object.freeze(language.messages);
  Object.freeze(language.source);
  const sourceKeys = new Map(Object.entries(language.source).map(([key, value]) => [value, key]));
  const canonicalMessages = new Map(Object.entries(language.messages).map(([key, value]) => [value, language.source[key]]));
  Object.defineProperty(globalThis, "jvC", {value: value => canonicalMessages.get(value) ?? value});
  Object.defineProperty(globalThis, "jvS", {value: value => sourceKeys.has(value) ? jvT(sourceKeys.get(value)) : value});
  Object.defineProperty(globalThis, 'jvT', {value: (key, values = {}) => {
    const message = language.messages[key] ?? key;
    return message.replace(/\{([A-Za-z][A-Za-z0-9_]*)\}/g, (token, name) =>
      Object.prototype.hasOwnProperty.call(values, name) ? String(values[name]) : token);
  }});
  // Escape catalog text before inserting existing escaped HTML placeholders.
  Object.defineProperty(globalThis, 'jvH', {value: (key, values = {}) => {
    const text = (language.messages[key] ?? language.source[key] ?? key)
      .replace(/[&<>"']/g, char => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[char]));
    return text.replace(/\{([A-Za-z][A-Za-z0-9_]*)\}/g, (token, name) =>
      Object.prototype.hasOwnProperty.call(values, name) ? String(values[name]) : token);
  }});
  document.addEventListener('click', event => {
    if (event.target.closest('[data-language-open]')) {
      const dialog = document.querySelector('[data-language-dialog]');
      if (!dialog) return;
      // Capture local query state as it is now, including workspace navigation.
      dialog.querySelector('[name="return"]').value = location.pathname + location.search;
      dialog.showModal();
    }
    if (event.target.closest('[data-language-close]')) event.target.closest('dialog').close();
  });
})();
