(() => {
  'use strict';

  const initialized = new WeakSet();
  globalThis.justVoxelMinecraftServers = (root = document) => {
  const list = root.querySelector('.minecraft-server-entries');
  const arrange = root.querySelector('[data-server-arrange]');
  const done = root.querySelector('[data-server-arrange-done]');
  if (!list || !arrange || !done || initialized.has(list)) return;
  initialized.add(list);

  const showPortForwarding = async (card, id) => {
    const dialog = document.createElement('dialog');
    dialog.className = 'minecraft-server-invitation';
    const title = document.createElement('h2');
    title.textContent = jvT("network.router.port_forwarding");
    dialog.setAttribute('aria-label', title.textContent);
    const name = document.createElement('p');
    name.textContent = card.dataset.serverName;
    const state = document.createElement('p');
    state.setAttribute('role', 'status');
    state.textContent = jvT("ui.ec963ffc911b");
    const summary = document.createElement('div');
    const actions = document.createElement('div');
    actions.className = 'action-row';
    const close = document.createElement('button');
    close.type = 'button';
    close.className = 'secondary';
    close.textContent = jvT("ui.7d9eb7acb13e");
    actions.appendChild(close);
    dialog.append(title, name, state, summary, actions);
    const controller = new AbortController();
    close.addEventListener('click', () => dialog.close());
    dialog.addEventListener('close', () => {
      controller.abort();
      dialog.remove();
      card.querySelector('summary')?.focus();
    });
    document.body.appendChild(dialog);
    dialog.showModal();
    const paragraph = (text) => {
      const p = document.createElement('p');
      p.textContent = text;
      summary.appendChild(p);
    };
    const render = (data) => {
      summary.replaceChildren();
      switch (data.status) {
        case 'not_configured': {
          state.textContent = jvT("network.router.not_configured");
          const router = document.createElement('a');
          router.className = 'button-link secondary';
          router.href = '/?workspace=network&tab=Router';
          router.textContent = jvT("network.router.open_workspace");
          actions.prepend(router);
          break;
        }
        case 'connected': state.textContent = jvT("ui.22965568d22a"); break;
        case 'configuration_error': state.textContent = jvT("network.router.configuration_error"); break;
        default: state.textContent = jvT("network.router.unavailable"); break;
      }
      if (data.forwarding_state === 'needs_repair') state.textContent = jvT("network.router.repair_state");
      if (data.forwarding_state === 'conflict') state.textContent = jvT("network.router.conflict_detail");
      if (data.forwarding_state === 'unknown') state.textContent = jvT("network.router.unknown_detail");
      if (data.network_member) paragraph(jvT("network.router.managed_by_network"));
      if (data.inventory_error || !data.endpoints?.length) paragraph(jvT("network.router.inventory_error"));
      for (const endpoint of data.endpoints || []) {
        const edition = endpoint.edition === 'java' ? jvT("network.router.java") : jvT("network.router.bedrock");
        const ports = endpoint.last_port > endpoint.port ? endpoint.port + '–' + endpoint.last_port : endpoint.port;
        let label, detail;
        switch (endpoint.state) {
          case 'open':
            label = jvT("network.router.open");
            detail = endpoint.source === 'managed' ? jvT("network.router.managed") : jvT("network.router.open_detail");
            break;
          case 'needs_repair':
            label = jvT("network.router.repair_state");
            detail = jvT("network.router.repair_detail");
            break;
          case 'closed':
            label = jvT("network.router.closed");
            detail = jvT("network.router.closed_detail");
            break;
          case 'conflict':
            label = jvT("network.router.conflict");
            detail = jvT("network.router.conflict_detail");
            break;
          default:
            label = jvT("ui.b764cdc0eab7");
            detail = jvT("network.router.unknown_detail");
            break;
        }
        const heading = document.createElement('h3');
        heading.textContent = [edition, endpoint.protocols.join('+').toUpperCase(), ports, label].join(' · ');
        summary.appendChild(heading);
        paragraph(detail);
      }
      const labels = {
        enable: jvT("network.router.enable"),
        repair: jvT("network.router.repair"),
        disable: jvT("network.router.disable"),
      };
      const action = data.forwarding_action;
      if (!Object.hasOwn(labels, action) || !data.forwarding_scope || data.status !== 'connected') return;
      const button = document.createElement('button');
      button.type = 'button';
      button.textContent = labels[action];
      actions.prepend(button);
      button.addEventListener('click', async () => {
        button.remove();
        state.textContent = jvT("ui.ec963ffc911b");
        try {
          const response = await fetch('/api/network/router/mikrotik/forwarding', {
            method: 'POST', credentials: 'same-origin', signal: controller.signal,
            headers: {'X-CSRF-Token': root.querySelector('[data-server-csrf]').dataset.serverCsrf},
            body: new URLSearchParams({scope: data.forwarding_scope, action}),
          });
          const result = await response.json();
          if (response.ok) render(result);
          else {
            summary.replaceChildren();
            state.textContent = result.error || jvT("network.router.forwarding_unknown");
          }
        } catch (_) {
          summary.replaceChildren();
          state.textContent = jvT("network.router.forwarding_unknown");
        }
      }, {once: true});
    };
    try {
      const response = await fetch('/api/network/router/mikrotik/instances/' + encodeURIComponent(id), {
        credentials: 'same-origin', cache: 'no-store', signal: controller.signal,
      });
      if (!response.ok) throw new Error();
      render(await response.json());
    } catch (_) {
      if (!controller.signal.aborted) state.textContent = jvT("network.router.unavailable");
    }
  };

  const key = 'justvoxel:minecraft-server-order:v1';
  const idPattern = /^jv-[a-z0-9]{6,12}$/;
  const cards = () => Array.from(list.querySelectorAll('[data-minecraft-server-card]'));
  const rendered = cards();
  const byID = new Map(rendered.map(card => [card.dataset.serverId, card]));
  let saved;
  try {
    saved = JSON.parse(window.localStorage.getItem(key));
  } catch (_) {
    // Browser storage is optional. Preserve the server-rendered order.
  }
  if (Array.isArray(saved)) {
    const seen = new Set();
    const ordered = [];
    for (const id of saved) {
      if (typeof id !== 'string' || !idPattern.test(id) || seen.has(id) || !byID.has(id)) continue;
      seen.add(id);
      ordered.push(byID.get(id));
    }
    for (const card of rendered) {
      if (!seen.has(card.dataset.serverId)) ordered.push(card);
    }
    // Ordering contains only real server cards, identified by immutable ID.
    for (const card of ordered) list.appendChild(card);
  }

  let arranging = false;
  const updateBounds = () => {
    const current = cards();
    current.forEach((card, index) => {
      card.querySelector('[data-server-move-earlier]').disabled = index === 0;
      card.querySelector('[data-server-move-later]').disabled = index === current.length - 1;
    });
  };
  const setArranging = active => {
    arranging = active;
    arrange.hidden = active;
    done.hidden = !active;
    for (const card of cards()) {
      for (const normal of card.querySelectorAll('[data-server-normal-actions]')) {
        normal.hidden = active;
        normal.inert = active;
      }
      card.querySelector('[data-server-arrange-actions]').hidden = !active;
      if (active) card.querySelector('details').open = false;
    }
    updateBounds();
    (active ? done : arrange).focus();
  };
  const persist = () => {
    try {
      window.localStorage.setItem(key, JSON.stringify(cards().map(card => card.dataset.serverId).filter(id => idPattern.test(id))));
    } catch (_) {
      // Keep this page's presentation usable when storage is unavailable.
    }
  };
  arrange.hidden = rendered.length === 0;
  arrange.addEventListener('click', () => setArranging(true));
  done.addEventListener('click', () => setArranging(false));
  for (const card of rendered) {
    for (const direction of ['earlier', 'later']) {
      const button = card.querySelector(`[data-server-move-${direction}]`);
      button.addEventListener('click', () => {
        if (!arranging || button.disabled) return;
        const current = cards();
        const index = current.indexOf(card);
        const neighbor = current[index + (direction === 'earlier' ? -1 : 1)];
        if (!neighbor) return;
        if (direction === 'earlier') list.insertBefore(card, neighbor);
        else list.insertBefore(neighbor, card);
        updateBounds();
        persist();
        // Moving to a boundary disables this button; keep keyboard focus usable.
        (button.disabled ? card.querySelector(`[data-server-move-${direction === 'earlier' ? 'later' : 'earlier'}]`) : button).focus();
      });
    }
    const suppress = event => {
      if (!arranging) return;
      event.preventDefault();
      event.stopImmediatePropagation();
    };
    for (const normal of card.querySelectorAll('[data-server-normal-actions]')) {
      normal.addEventListener('click', suppress, true);
      normal.addEventListener('submit', suppress, true);
    }
    initializeCard(card, root, () => arranging);
  }
  updateBounds();
  initializeLiveMemory(list, cards);
  };
  const matchesRecovery = (response, payload, id, operationID) =>
    response.status === 409 && !response.redirected && payload?.recovery_required === true &&
    payload.recovery_kind === 'plugin_install' && payload.id === id &&
    typeof payload.recovery_operation_id === 'string' &&
    /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/.test(payload.recovery_operation_id) &&
    (!operationID || payload.recovery_operation_id === operationID) &&
    (payload.recovery_action === 'start' || payload.recovery_action === 'stop');

  const openRecovery = ({recovery, id, csrf, selected, request, refresh, isCurrent = () => true}) => {
    const operationID = recovery.recovery_operation_id;
    const dialog = document.createElement('dialog');
    dialog.className = 'minecraft-server-invitation';
    dialog.setAttribute('aria-label', jvT("plugins.recovery.title"));
    const title = document.createElement('h2');
    title.textContent = jvT("plugins.recovery.title");
    const description = document.createElement('p');
    const status = document.createElement('p');
    status.setAttribute('role', 'status');
    const actions = document.createElement('div');
    actions.className = 'action-row';
    const apply = document.createElement('button');
    apply.type = 'button';
    const details = document.createElement('a');
    details.className = 'button-link secondary';
    details.href = selected('/?workspace=plugins');
    details.textContent = jvT("plugins.recovery.details");
    const close = document.createElement('button');
    close.type = 'button';
    close.className = 'secondary';
    close.textContent = jvT("ui.7d9eb7acb13e");
    close.addEventListener('click', () => dialog.close());
    const render = () => {
      const start = recovery.recovery_action === 'start';
      description.textContent = start ? jvT("plugins.recovery.card_start") : jvT("plugins.recovery.card_stop");
      const confirmPlayers = !start && recovery.confirmation_required === true;
      apply.textContent = start ? jvT("plugins.recovery.start") :
        confirmPlayers ? jvT("plugins.recovery.confirm_stop") : jvT("plugins.recovery.stop");
      status.textContent = confirmPlayers
        ? jvT("plugins.recovery.players", {count: (recovery.online)}) + (recovery.players?.length ? ' ' + recovery.players.join(', ') : '')
        : '';
    };
    const dismissIfStale = () => {
      if (isCurrent()) return false;
      if (dialog.open) dialog.close();
      return true;
    };
    let busy = false;
    apply.addEventListener('click', async () => {
      if (!dialog.open || busy || dismissIfStale()) return;
      busy = apply.disabled = close.disabled = true;
      status.textContent = jvT("ui.76811ca2ebdb");
      try {
        const body = new URLSearchParams({csrf, operation_id: operationID, restore_runtime: 'true'});
        if (recovery.recovery_action === 'stop' && recovery.confirmation_required === true) body.set('confirm_players', 'true');
        const result = await request('/api/plugins/workspace/install/restore-runtime', {method: "POST", body}, operationID);
        if (dismissIfStale()) return;
        if (result?.recovery) {
          recovery = result.recovery;
          render();
          return;
        }
        if (result?.operation?.operation_id !== operationID || result.operation.instance_id !== id || result.operation.state !== 'resolved') {
          throw new Error(jvT("ui.1f3344e99b38"));
        }
        await refresh();
        if (dialog.open) dialog.close();
      } catch (error) {
        if (!dismissIfStale()) status.textContent = error.message;
      } finally {
        busy = apply.disabled = close.disabled = false;
      }
    });
    details.addEventListener('click', event => { if (busy || dismissIfStale()) event.preventDefault(); });
    dialog.addEventListener('cancel', event => { if (busy) event.preventDefault(); });
    dialog.addEventListener('close', () => {
      window.removeEventListener('justvoxel:server-selection-changed', dismissIfStale);
      dialog.remove();
    });
    window.addEventListener('justvoxel:server-selection-changed', dismissIfStale);
    actions.append(apply, details, close);
    dialog.append(title, description, status, actions);
    render();
    document.body.appendChild(dialog);
    dialog.showModal();
  };
  globalThis.justVoxelPluginRuntimeRecovery = {matches: matchesRecovery, open: openRecovery};

  function initializeLiveMemory(list, cards) {
    const history = new Map();
    const validBytes = value => typeof value === 'number' && Number.isFinite(value) && value >= 0;
    const format = bytes => bytes >= 1024 ** 3
      ? `${(bytes / 1024 ** 3).toFixed(2)} GB`
      : `${Math.round(bytes / 1024 ** 2)} MB`;
    const update = rows => {
      const byID = new Map(rows.filter(row => row && /^jv-[a-z0-9]{6,12}$/.test(row.id)).map(row => [row.id, row]));
      const now = Date.now();
      for (const card of cards()) {
        const row = byID.get(card.dataset.serverId);
        const tile = card.querySelector('[data-server-ram]');
        if (row && typeof row.state === 'string' && typeof row.state_class === 'string'
            && typeof row.running === 'boolean' && typeof row.can_switch === 'boolean') {
          if (['neutral', 'success', 'warning', 'error'].includes(row.card_health)) {
            for (const health of ['neutral', 'success', 'warning', 'error']) card.classList.toggle(`minecraft-server-health-${health}`, health === row.card_health);
          }
          const status = card.querySelector('.minecraft-server-state');
          if (status) {
            status.textContent = jvS(row.state);
            status.className = `minecraft-server-state ${row.state_class}`;
          }
          const running = card.querySelector('[data-server-running]');
          if (running) {
            running.checked = row.running;
            running.disabled = !(row.can_switch || (card.dataset.networkMember === 'true' && row.state === 'Failed'));
          }
          if (tile) tile.dataset.serverState = row.state;
        }
        if (!tile) continue;
        const value = tile.querySelector('[data-server-ram-value]');
        const graph = tile.querySelector('[data-server-ram-graph]');
        const id = card.dataset.serverId;
        graph.replaceChildren();
        // Only authoritative lifecycle state can declare a stop.
        if (tile.dataset.serverState === 'Stopped') {
          value.textContent = '---';
          history.delete(id);
          continue;
        }
        if (!row || !validBytes(row.usage_bytes)) {
          value.textContent = jvT("ui.ca1844969742");
          history.delete(id);
          continue;
        }
        value.textContent = format(row.usage_bytes);
        const samples = (history.get(id) || []).filter(sample => now - sample.at < 60000);
        samples.push({usage: row.usage_bytes, at: now});
        if (samples.length > 30) samples.splice(0, samples.length - 30);
        history.set(id, samples);
        if (!validBytes(row.limit_bytes) || row.limit_bytes <= 0 || samples.length < 2) continue;
        const line = document.createElementNS('http://www.w3.org/2000/svg', 'polyline');
        line.setAttribute('points', samples.map((sample, index) => {
          const x = (30 - samples.length + index) * 4;
          const y = 12 * (1 - Math.min(sample.usage, row.limit_bytes) / row.limit_bytes);
          return `${x},${y}`;
        }).join(' '));
        graph.appendChild(line);
      }
    };
    let pending = false;
    const poll = async () => {
      if (!list.isConnected) { clearInterval(timer); return; }
      if (pending) return;
      pending = true;
      try {
        const response = await fetch('/api/minecraft/servers/live-memory', {
          credentials: 'same-origin', cache: 'no-store', headers: {Accept: 'application/json'},
        });
        if (!response.ok || response.redirected) throw new Error(jvT("ui.dbcb2af9750b"));
        const rows = await response.json();
        if (!Array.isArray(rows)) throw new Error(jvT("ui.a0910cc5d920"));
        if (list.isConnected) update(rows);
      } catch (_) {
        if (list.isConnected) update([]);
      } finally {
        pending = false;
        if (!list.isConnected) clearInterval(timer);
      }
    };
    const timer = setInterval(poll, 2000);
    poll();
  }
  function initializeCard(card, root, isArranging) {
    const id = card.dataset.serverId;
    const inventory = root.querySelector('[data-server-csrf]');
    if (!inventory || !/^jv-[a-z0-9]{6,12}$/.test(id)) return;
    const csrf = inventory.dataset.serverCsrf;
    const selected = path => path + (path.includes('?') ? '&' : '?') + new URLSearchParams({server: id});
    const settingsURL = selected('/?workspace=minecraft');
    const connectURL = selected('/minecraft/how-to-connect');
    const feedback = (message, connect = false, review = false, warning = false) => {
      const node = root.querySelector('[data-server-feedback]');
      if (!node) return;
      node.className = warning ? 'notice warning' : 'notice error';
      node.textContent = message;
      node.hidden = !message;
      if (connect || review) {
        const link = document.createElement('a');
        link.href = connect ? connectURL : settingsURL;
        link.textContent = connect ? jvT("ui.3d912f7f913c") : jvT("ui.62fe28d6ca7f");
        node.appendChild(link);
      }
    };
    const request = async (path, options = {}, recoveryOperationID = null) => {
      const response = await fetch(selected(path), {credentials: 'same-origin', cache: 'no-store', ...options, headers: {Accept: 'application/json'}});
      let payload;
      try { payload = await response.json(); } catch (_) { throw new Error(jvT("ui.a3a82b87d0eb")); }
      if (recoveryOperationID !== null && matchesRecovery(response, payload, id, recoveryOperationID)) return {recovery: payload};
      if (!response.ok || response.redirected) throw new Error(payload.error || (recoveryOperationID !== null && payload.message) || jvT("ui.4d5161e70e99"));
      return payload;
    };
    const refresh = async () => {
      if (root.justVoxelRefreshServers) await root.justVoxelRefreshServers();
    };
    let pending = false;
    const controls = Array.from(card.querySelectorAll('input, button'));
    const originalDisabled = new Map(controls.map(control => [control, control.disabled]));
    const operation = async (action, {connect = false} = {}) => {
      if (pending || isArranging()) return;
      pending = true;
      controls.forEach(control => { control.disabled = true; });
      feedback('');
      try { await action(); }
      catch (error) { feedback(error.message, connect); }
      finally {
        pending = false;
        controls.forEach(control => { control.disabled = originalDisabled.get(control); });
      }
    };
    const mutate = async action => {
      let failure;
      try { await action(); } catch (error) { failure = error; }
      // Even a refused/failed request may have changed state. Reload authoritative cards.
      await refresh();
      if (failure) throw failure;
    };
    const running = card.querySelector('[data-server-running]');
    running?.addEventListener('change', () => {
      const requested = running.checked;
      running.checked = !requested;
      if (!requested && card.dataset.networkMember === 'true') {
        feedback(jvT("ui.ceadef34e4b9"), false, false, true);
        return;
      }
      operation(async () => {
        let recovery;
        await mutate(async () => {
          const status = card.querySelector('.minecraft-server-state');
          if (status) {
            status.textContent = requested ? jvT("ui.aeed4d26bb5f") : jvT("ui.a71ee1d4ac5c");
            status.className = 'minecraft-server-state warning';
          }
          const result = await request('/api/minecraft/servers/lifecycle', {method: "POST", body: new URLSearchParams({csrf, action: requested ? 'start' : 'stop'})}, '');
          if (result.recovery) recovery = result.recovery;
          else if (!result.ok) throw new Error(jvT("ui.bd44764996e3"));
        });
        if (recovery) openRecovery({recovery, id, csrf, selected, request, refresh});
      });
    });
    const requiresReview = plan => Object.entries(plan).some(([key, value]) => key.endsWith('_required') && value === true);
    const boot = card.querySelector('[data-server-boot]');
    boot?.addEventListener('change', () => {
      const requested = boot.checked;
      boot.checked = !requested;
      operation(async () => {
        let review = false;
        await mutate(async () => {
          const body = new URLSearchParams({csrf, tab: 'overview', start_on_boot: String(requested)});
          const plan = await request('/api/minecraft/workspace/settings/plan', {method: "POST", body});
          if (requiresReview(plan)) { review = true; return; }
          if (!plan.review) throw new Error(jvT("ui.257b560c22e1"));
          body.set('review', plan.review);
          const result = await request('/api/minecraft/workspace/settings/apply', {method: "POST", body});
          if (requiresReview(result) && !result.applied) { review = true; return; }
          if (!result.applied) throw new Error(result.error || jvT("ui.59f7401b3446"));
        });
        if (review) feedback(jvT("ui.35de87951c7e"), false, true);
      });
    });
    card.querySelector('[data-server-restart]')?.addEventListener('click', () => {
      closeMenu();
      operation(() => mutate(async () => {
        const result = await request('/minecraft/servers/view/' + encodeURIComponent(id) + '/restart', {method: "POST", body: new URLSearchParams({csrf})});
        if (!result.ok) throw new Error(jvT("ui.a5a019d89baa"));
      }));
    });
    card.querySelector('[data-server-delete]')?.addEventListener('click', () => {
      if (pending || isArranging()) return;
      closeMenu();
      globalThis.justVoxelOpenDeleteServer({name: card.dataset.serverName,
        description: jvT("ui.e9d6a58ae67a"),
        csrf, request, dismissOnDeleted: true, isCurrent: () => card.isConnected && !isArranging(),
        onError: error => feedback(error.message), onDeleted: () => refresh(),
      });
    });
    card.querySelector('[data-server-port-forwarding]')?.addEventListener('click', () => {
      if (isArranging()) return;
      closeMenu();
      showPortForwarding(card, id);
    });
    card.querySelector('[data-server-invite]')?.addEventListener('click', () => {
      closeMenu();
      operation(async () => {
        const payload = await request('/api/minecraft/servers/invitation');
        if (!card.isConnected || isArranging()) return;
        if (!payload.invitation) throw new Error(jvT("ui.619158e764df"));
        const dialog = document.createElement('dialog');
        dialog.className = 'minecraft-server-invitation';
        dialog.setAttribute('aria-label', jvT("ui.2614b42d8452"));
        const title = document.createElement('h2'); title.textContent = jvT("ui.2614b42d8452");
        const name = document.createElement('p'); name.textContent = payload.name;
        const label = document.createElement('label'); label.textContent = jvT("ui.af4ea02852eb");
        const text = document.createElement('textarea'); text.readOnly = true; text.rows = 10; text.spellcheck = false; text.value = payload.invitation;
        label.appendChild(text);
        const status = document.createElement('p'); status.setAttribute('role', 'status');
        const actions = document.createElement('div'); actions.className = 'action-row';
        const close = document.createElement('button'); close.type = 'button'; close.className = 'secondary'; close.textContent = jvT("ui.7d9eb7acb13e");
        const copy = document.createElement('button'); copy.type = 'button'; copy.textContent = jvT("ui.74554e40cdde");
        close.addEventListener('click', () => dialog.close());
        copy.addEventListener('click', async () => {
          try { await navigator.clipboard.writeText(text.value); status.textContent = jvT("ui.d249818eccab"); }
          catch (_) { status.textContent = jvT("ui.e30e3a48556a"); }
        });
        actions.append(close, copy); dialog.append(title, name, label, status, actions);
        dialog.addEventListener('click', event => {
          if (event.target !== dialog) return;
          const rect = dialog.getBoundingClientRect();
          if (event.clientX < rect.left || event.clientX > rect.right || event.clientY < rect.top || event.clientY > rect.bottom) dialog.close();
        });
        dialog.addEventListener('close', () => dialog.remove());
        document.body.appendChild(dialog); dialog.showModal();
      }, {connect: true});
    });
    for (const link of card.querySelectorAll('a[data-server-workspace]')) {
      link.addEventListener('click', event => {
        if (event.defaultPrevented || event.button !== 0 || event.ctrlKey || event.metaKey || event.shiftKey || event.altKey || isArranging()) return;
        const workspace = link.dataset.serverWorkspace;
        const selector = {minecraft: '[data-minecraft-open]', backups: '[data-backups-open]', version: '[data-version-open]', plugins: '[data-plugins-open]', migration: '[data-migration-open]'}[workspace];
        const launcher = selector && document.querySelector(selector);
        if (!launcher || typeof globalThis.justVoxelSelectServer !== 'function') return;
        event.preventDefault();
        closeMenu();
        if (globalThis.justVoxelSelectServer(id, workspace)) launcher.click();
      });
    }
    const menu = card.querySelector('details');
    const choices = menu?.querySelector('[popover]');
    const trigger = menu?.querySelector('summary');
    function closeMenu() {
      if (choices?.matches(':popover-open')) choices.hidePopover();
      if (menu) menu.open = false;
    }
    if (menu && choices && trigger) {
      menu.addEventListener('toggle', () => {
        if (!menu.isConnected) return;
        if (!menu.open || isArranging()) { closeMenu(); return; }
        choices.showPopover();
        const rect = trigger.getBoundingClientRect();
        const bounds = choices.getBoundingClientRect();
        choices.style.left = Math.max(8, Math.min(rect.right - bounds.width, window.innerWidth - bounds.width - 8)) + 'px';
        choices.style.top = Math.max(8, Math.min(rect.bottom + 4, window.innerHeight - bounds.height - 8)) + 'px';
      });
      choices.addEventListener('toggle', event => { if (event.newState === 'closed') menu.open = false; });
      choices.addEventListener('click', event => { if (event.target.closest('a')) closeMenu(); });
    }
  }
  window.addEventListener?.('resize', () => {
    document.querySelectorAll('.minecraft-server-more[open]').forEach(menu => { menu.open = false; });
  });
  globalThis.justVoxelMinecraftServers();
})();
