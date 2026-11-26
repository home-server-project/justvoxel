(() => {
  const terminal = new Set(["succeeded", "resolved", "needs_attention", "failed"]);
  window.JustVoxelServerMigrationSwitch = {
    init(root) {
      if (!root?.matches("[data-migration-switch]") || root.dataset.switchInitialized) return;
      root.dataset.switchInitialized = "true";
      const server = root.dataset.serverId;
      const generation = globalThis.justVoxelSelectionGeneration?.migration;
      const current = () => root.isConnected && generation === globalThis.justVoxelSelectionGeneration?.migration;
      const activeSwitch = () => {
        const workspace = root.closest("[data-migration-workspace-dialog]");
        return current() && workspace?.open && workspace.querySelector('[data-migration-tab="switch"][aria-selected="true"]');
      };
      const state = root.querySelector("[data-switch-state]");
      const review = root.querySelector("[data-switch-review]");
      const progress = root.querySelector("[data-switch-operation]");
      const cards = [...root.querySelectorAll("[data-switch-target]")];
      const csrf = root.querySelector("[data-switch-csrf]").value;
      let sequence = 0;
      let operation = null;
      let busy = false;
      let polling = false;
      const request = async (action, params = {}, post = false) => {
        if (!current()) throw new Error(jvT("ui.bed53cc46da9"));
        const query = new URLSearchParams({ server, ...(post ? {} : params) });
        const response = await fetch(`/api/migration/switch/${action}?${query}`, {
          method: post ? "POST" : "GET", credentials: "same-origin", cache: "no-store",
          ...(post ? { headers: { "Content-Type": "application/x-www-form-urlencoded" }, body: new URLSearchParams({ csrf, ...params }) } : {}),
        });
        const result = await response.json();
        if (!current()) throw new Error(jvT("ui.bed53cc46da9"));
        if (!response.ok) throw new Error(result.error || jvT("ui.1eb6b959b7ce"));
        return result;
      };
      const paragraph = (text, className = "") => {
        const node = document.createElement("p"); node.textContent = text; node.className = className; return node;
      };
      const actionRow = (...children) => {
        const row = document.createElement("div"); row.className = "action-row migration-switch-action";
        row.append(...children);
        return row;
      };
      const matrixRun = root.querySelector("[data-switch-matrix-run]");
      const matrixStatus = root.querySelector("[data-switch-matrix-status]");
      let matrix = null;
      let matrixAction = "";
      let matrixError = "";
      let matrixStatusFailed = false;
      let matrixTicket = 0;
      let syncReview = null;
      const availableCards = new Set();
      const matrixBlocksSwitch = () => matrixAction === "start" || Boolean(matrix?.operation?.reservation_active);
      const syncControls = () => {
        if (matrixRun) {
          matrixRun.hidden = matrix?.enabled !== true;
          matrixRun.disabled = busy || Boolean(matrixAction || matrixStatusFailed || matrix?.operation?.reservation_active);
        }
        cards.forEach((card) => { card.disabled = busy || matrixBlocksSwitch() || !availableCards.has(card); });
        syncReview?.();
      };
      const setBusy = (value) => { busy = value; syncControls(); };
      const matrixButton = (text, handler) => {
        const button = document.createElement("button");
        button.type = "button";
        button.textContent = text;
        button.addEventListener("click", handler);
        return button;
      };
      const renderMatrix = () => {
        syncControls();
        matrixStatus.replaceChildren();
        if (matrixError) matrixStatus.append(paragraph(matrixError, "migration-switch-error"));
        const run = matrix?.operation;
        if (!run) return;
        if (!matrix.enabled) {
          if (run.reservation_active) matrixStatus.append(paragraph(jvT("switch_matrix.development_recovery"), "notice warning"));
          return;
        }

        const status = run.progress;
        const messages = {
          preparing: "switch_matrix.discovering",
          running: "switch_matrix.running",
          closing: "switch_matrix.closing",
          completed: status?.failed ? "switch_matrix.completed_failures" : "switch_matrix.completed",
          cancelled: "switch_matrix.cancelled",
          interrupted: "switch_matrix.interrupted",
          needs_attention: "switch_matrix.needs_attention",
          recovered: "switch_matrix.recovered",
          status_unavailable: "switch_matrix.status_failed",
        };
        matrixStatus.append(paragraph(jvT(messages[run.state] || "switch_matrix.status_failed")));
        if (status?.cancel_requested && run.reservation_active && !run.recovery_required) {
          matrixStatus.append(paragraph(jvT("switch_matrix.cancelling")));
        }
        if (status?.manifest_ready && run.state !== "preparing") {
          if (run.reservation_active && !run.recovery_required && status.total > 0) {
            const bar = document.createElement("progress");
            bar.max = status.total;
            bar.value = status.completed;
            bar.setAttribute("aria-label", jvT("switch_matrix.count", {v0: status.completed, v1: status.total}));
            matrixStatus.append(bar);
          }
          matrixStatus.append(paragraph(jvT("switch_matrix.count", {v0: status.completed, v1: status.total})));
          if (status.source && status.target && ["clone", "convert"].includes(status.action)) {
            const action = status.action === "clone" ? jvT("switch_matrix.clone") : jvT("switch_matrix.convert");
            matrixStatus.append(paragraph(jvT("switch_matrix.current", {v0: status.source, v1: status.target, v2: action})));
          }
          matrixStatus.append(paragraph(jvT("switch_matrix.results", {
            v0: status.passed, v1: status.failed, v2: status.unsupported, v3: status.blocked,
          })));
          if (status.unfinished && (!run.reservation_active || run.recovery_required)) {
            matrixStatus.append(paragraph(jvT("switch_matrix.unfinished", {v0: status.unfinished})));
          }
        }

        if (run.recovery_required) {
          const recover = matrixButton(jvT(matrixAction === "recover" ? "ui.d90064dbdb11" : "switch_matrix.recover"),
            () => void submitMatrix("recover", run.operation_id));
          recover.disabled = Boolean(matrixAction);
          matrixStatus.append(actionRow(paragraph(jvT("switch_matrix.reservation_held")), recover));
        }
        if (run.reservation_active && !run.recovery_required) {
          if (run.state === "closing") return;
          const stop = matrixButton(jvT("switch_matrix.stop"), () => void submitMatrix("stop", run.operation_id));
          stop.disabled = Boolean(matrixAction || status?.cancel_requested);
          matrixStatus.append(actionRow(paragraph(jvT("switch_matrix.stop_help")), stop));
        } else {
          const logs = matrixButton(jvT("switch_matrix.open_logs"), () => {
            if (activeSwitch()) void window.JustVoxelSystem?.openMigrationLogs(run.diagnostic_id);
          });
          matrixStatus.append(actionRow(paragraph(jvT("switch_matrix.logs_help")), logs));
        }
      };
      const matrixRequest = async (action = "", id = "") => {
        const response = await fetch("/api/migration/switch-matrix" + (action ? "/" + action : ""), {
          method: action ? "POST" : "GET", credentials: "same-origin", cache: "no-store",
          ...(action ? {
            headers: { "Content-Type": "application/x-www-form-urlencoded" },
            body: new URLSearchParams(id ? {csrf, operation_id: id} : {csrf}),
          } : {}),
        });
        if (!response.ok) throw new Error();
        return response.json();
      };
      const matrixPoll = (ticket) => {
        if (!matrix?.operation?.reservation_active || matrix.operation.recovery_required || !matrix.enabled) return;
        setTimeout(() => {
          if (activeSwitch() && ticket === matrixTicket && !matrixAction) void refreshMatrix();
        }, 2000);
      };
      const refreshMatrix = async (actionFailed = false) => {
        const ticket = ++matrixTicket;
        try {
          const status = await matrixRequest();
          if (!activeSwitch() || ticket !== matrixTicket) return;
          matrix = status;
          matrixStatusFailed = false;
          matrixError = actionFailed ? jvT("switch_matrix.action_failed") : "";
        } catch (_) {
          if (!activeSwitch() || ticket !== matrixTicket) return;
          matrixStatusFailed = true;
          matrixError = jvT("switch_matrix.status_failed");
        }
        renderMatrix();
        matrixPoll(ticket);
      };
      const submitMatrix = async (action, id = "") => {
        if (!activeSwitch() || matrix?.enabled !== true || matrixAction) return;
        if (action === "start" && matrixRun.disabled) return;
        if (action === "stop" && (matrix.operation?.operation_id !== id || !matrix.operation.reservation_active ||
            matrix.operation.recovery_required || matrix.operation.state === "closing" || matrix.operation.progress?.cancel_requested)) return;
        if (action === "recover" && (matrix.operation?.operation_id !== id || !matrix.operation.recovery_required)) return;
        const ticket = ++matrixTicket;
        matrixAction = action;
        matrixError = "";
        renderMatrix();
        try {
          const status = await matrixRequest(action, id);
          if (!activeSwitch() || ticket !== matrixTicket) return;
          matrix = status;
          matrixStatusFailed = false;
        } catch (_) {
          // A lost response may still have changed the durable operation.
          if (activeSwitch() && ticket === matrixTicket) await refreshMatrix(true);
          return;
        } finally {
          matrixAction = "";
          if (activeSwitch()) renderMatrix();
        }
        matrixPoll(ticket);
      };
      if (matrixRun && matrixStatus) {
        matrixRun.addEventListener("click", () => void submitMatrix("start"));
        void refreshMatrix();
      }
      const setState = (message, error = false) => {
        state.classList.toggle("migration-switch-error", error);
        state.textContent = message;
      };
      const showCard = (node, plan) => {
        const active = plan.current === node.dataset.switchTarget;
        const available = !active && Boolean(plan.convert || plan.clone);
        if (available) availableCards.add(node);
        else availableCards.delete(node);
        node.classList.toggle("is-pending", false);
        node.classList.toggle("is-current", active);
        node.classList.toggle("is-available", available);
        node.classList.toggle("is-unsupported", !active && !available);
        node.disabled = active || !available || busy || matrixBlocksSwitch();
        const marker = node.querySelector("[data-switch-status]");
        marker.textContent = active ? "C" : available ? "A" : "N";
        marker.classList.toggle("plugins-badge-disabled", active);
        marker.classList.toggle("plugins-badge-enabled", available);
        node.title = plan.reason || (active ? jvT("ui.7d4f12e4405b") : jvT("ui.a195a363286b", {v0: (plan.version)}));
      };
      const reconnect = async (id = "", resolvedRecovery = null) => {
        const ticket = sequence;
        const result = await request("operation", { operation_id: id });
        if (!current() || ticket !== sequence) return;
        if (resolvedRecovery && (result.operation?.operation_id !== id || result.operation.state !== "resolved")) return;
        operation = result.operation || null;
        setBusy(Boolean(operation && !["succeeded", "resolved"].includes(operation.state)));
        const recovery = ["needs_attention", "failed"].includes(operation?.state);
        if (!operation) { progress.replaceChildren(); return; }
        if (cards.some((node) => node.dataset.switchTarget === result.target)) {
          cards.forEach((node) => node.classList.toggle("is-selected", node.dataset.switchTarget === result.target));
        }
        progress.replaceChildren(paragraph(operation.status || jvT("ui.934767ef0bcc")));
        const cloneID = result.clone_id;
        if (operation.state === "succeeded" && result.action === "clone" &&
            typeof cloneID === "string" && /^jv-[a-z0-9]{6,12}$/.test(cloneID) && cloneID !== server) {
          const completed = operation;
          const open = document.createElement("button"); open.type = "button"; open.textContent = jvT("ui.10c040af618e");
          const actions = actionRow(paragraph(jvT("ui.b62c04b44caf")), open);
          const canOpen = () => activeSwitch() && ticket === sequence && operation === completed;
          let errorRow;
          open.addEventListener("click", async () => {
            if (!canOpen() || open.disabled) return;
            open.disabled = true;
            errorRow?.remove();
            try {
              const launcher = document.querySelector("[data-minecraft-servers-open]");
              if (!launcher) throw new Error();
              const inventory = await globalThis.justVoxelLoadServerSelections(true);
              if (!canOpen()) return;
              if (!inventory.servers.some(({id}) => id === cloneID)) throw new Error();
              // Opening or raising this window also runs its server list loader.
              launcher.click();
            } catch (_) {
              if (!canOpen()) return;
              errorRow = paragraph(jvT("ui.c3ba26d72a06"), "migration-switch-error");
              progress.insertBefore(errorRow, actions);
            } finally {
              open.disabled = false;
            }
          });
          progress.appendChild(actions);
        }
        if (recovery) {
          const recover = document.createElement("button"); recover.type = "button"; recover.textContent = jvT("ui.b8e7e9e5f031");
          const actions = actionRow(paragraph(jvT("ui.75ea388b9a14")), recover);
          const operationID = operation.operation_id;
          let errorRow;
          recover.addEventListener("click", async () => {
            if (!current() || ticket !== sequence || operation?.operation_id !== operationID || recover.disabled) return;
            recover.disabled = true;
            recover.textContent = jvT("ui.d90064dbdb11");
            errorRow?.remove();
            try {
              const result = await request("recover", { operation_id: operationID, confirm: "yes" }, true);
              if (!current() || ticket !== sequence) return;
              if (result.operation?.operation_id !== operationID || result.operation.state !== "resolved") {
                throw new Error(jvT("ui.1eb6b959b7ce"));
              }
              operation = result.operation;
              setBusy(false);
              progress.replaceChildren(paragraph(operation.status || jvT("ui.934767ef0bcc")));
              await load(operationID, operation);
            } catch (failure) {
              if (!current() || ticket !== sequence) return;
              errorRow = paragraph(failure.message, "migration-switch-error");
              progress.insertBefore(errorRow, actions);
              recover.disabled = false;
              recover.textContent = jvT("ui.b8e7e9e5f031");
            }
          });
          progress.appendChild(actions);
        }
        cards.forEach((node) => { if (busy) node.disabled = true; });
        if (!terminal.has(operation.state) && !polling) {
          polling = true;
          const operationID = operation.operation_id;
          setTimeout(async () => {
            polling = false;
            if (!current() || ticket !== sequence || operation?.operation_id !== operationID) return;
            try {
              await reconnect(operationID);
              if (current() && ticket === sequence && terminal.has(operation?.state)) await load(operationID);
            }
            catch (failure) { if (current() && ticket === sequence) setState(jvT("ui.796b21621486", {v0: (failure.message)}), true); }
          }, 2000);
        }
      };
      const select = async (target) => {
        if (busy || matrixBlocksSwitch()) return;
        const ticket = ++sequence;
        cards.forEach((node) => node.classList.toggle("is-selected", node.dataset.switchTarget === target));
        review.hidden = true;
        syncReview = null;
        if (operation && terminal.has(operation.state)) {
          operation = null;
          progress.replaceChildren();
        }
        setState(jvT("ui.e29d2712155f"));
        try {
          // Selection always re-reads authoritative inventory, even if a prior
          // catalogue response had made this card available.
          const plan = await request("plan", { target });
          if (!current() || ticket !== sequence) return;
          showCard(cards.find((node) => node.dataset.switchTarget === target), plan);
          review.replaceChildren();
          review.append(paragraph(jvT("ui.3101457ed8e3", {v0: (plan.current)})), paragraph(jvT("ui.740cb666e20a", {v0: (plan.target)})), paragraph(jvT("ui.6a5ef5b0f8c1", {v0: (plan.version || jvT("ui.33c8e8de41c3"))})),
            paragraph(jvT("ui.9ccd64d18f1e", {v0: (plan.convert ? jvT("ui.e674447337e8") : jvT("ui.ca1844969742"))})), paragraph(jvT("ui.b985dbadc276", {v0: (plan.clone ? jvT("ui.e674447337e8") : jvT("ui.ca1844969742"))})));
          if (plan.reason) review.appendChild(paragraph(plan.reason, "notice warning"));
          for (const item of plan.content || []) {
            const label = { verified: jvT("ui.630d4989d81c"), incompatible: jvT("ui.8777e3a9e3ec"), unknown: jvT("ui.1a31290e493f") }[item.compatibility] || jvT("ui.1a31290e493f");
            review.appendChild(paragraph(jvT("ui.31d29868e4e3", {v0: (item.name), v1: (item.filename), v2: (label), v3: (item.reason)}), item.compatibility === "verified" ? "muted" : "notice warning"));
          }
          for (const warning of plan.warnings || []) review.appendChild(paragraph(warning, "notice warning"));
          const plugins = document.createElement("button"); plugins.type = "button"; plugins.className = "secondary"; plugins.textContent = jvT("ui.70171ec7a356");
          plugins.addEventListener("click", () => {
            if (!activeSwitch() || ticket !== sequence || review.hidden) return;
            const launcher = document.querySelector("[data-plugins-open]");
            if (launcher && globalThis.justVoxelSelectServer(server, "plugins")) launcher.click();
          });
          const actions = actionRow(plugins);
          if (plan.convert || plan.clone) {
            review.appendChild(paragraph(jvT("ui.6b2df59bf4f2")));
            const label = document.createElement("label"); label.className = "minecraft-native-toggle system-ups-shutdown-switch";
            const consent = document.createElement("input"); consent.type = "checkbox"; consent.setAttribute("role", "switch");
            const confirmation = document.createElement("span"); confirmation.textContent = jvT("ui.1ee40e71974c");
            label.append(confirmation, consent); review.appendChild(label);
            const updateApplyButtons = [];
            syncReview = () => updateApplyButtons.forEach((update) => update());
            consent.addEventListener("change", syncReview);
            for (const [action, text, allowed] of [["convert", jvT("ui.5abb6e80374e"), plan.convert], ["clone", jvT("ui.90b908765b19"), plan.clone]]) {
              const button = document.createElement("button"); button.type = "button"; button.textContent = text; button.disabled = true;
              if (!allowed) button.title = plan.reason || jvT("ui.f4239cda81ee");
              updateApplyButtons.push(() => { button.disabled = !allowed || !consent.checked || busy || matrixBlocksSwitch(); });
              button.addEventListener("click", async () => {
                if (!current() || ticket !== sequence || !allowed || !consent.checked || busy || matrixBlocksSwitch() || button.disabled) return;
                setBusy(true);
                try {
                  const result = await request("apply", { target, action, plan_fingerprint: plan.plan_fingerprint, confirm: "yes" }, true);
                  if (!current() || ticket !== sequence) return;
                  if (!result.operation?.operation_id) throw new Error(jvT("ui.d8bcc288b463"));
                  review.hidden = true; await reconnect(result.operation.operation_id);
                  if (current() && ticket === sequence && terminal.has(operation?.state)) await load(result.operation.operation_id);
                } catch (failure) {
                  if (!current() || ticket !== sequence) return;
                  setState(failure.message, true);
                  // A lost Apply response may have started work. Reconnect
                  // before enabling another review or action.
                  try {
                    await reconnect();
                    if (!current() || ticket !== sequence) return;
                    if (!operation) {
                      await load();
                      if (current()) setState(failure.message, true);
                    } else if (terminal.has(operation.state)) {
                      await load(operation.operation_id);
                    }
                  }
                  catch (_) { if (current() && ticket === sequence) setState(jvT("ui.683b73665685", {v0: (failure.message)}), true); }
                }
              });
              actions.appendChild(button);
            }
          }
          review.appendChild(actions);
          review.hidden = false; setState("");
        } catch (failure) { if (current() && ticket === sequence) setState(failure.message, true); }
      };
      const load = async (id = "", resolvedRecovery = null) => {
        const ticket = ++sequence;
        review.hidden = true;
        syncReview = null;
        availableCards.clear();
        cards.forEach((node) => { node.disabled = true; });
        setState(jvT("ui.e29d2712155f"));
        try {
          try {
            await reconnect(id, resolvedRecovery);
          } catch (failure) {
            // Recovery already returned its authoritative terminal state.
            // Continue to refresh cards if the follow-up snapshot is unavailable.
            if (!resolvedRecovery) throw failure;
            setBusy(false);
          }
          if (!current() || ticket !== sequence) return;
          if (!operation) cards.forEach((node) => { node.classList.toggle("is-selected", false); });
          const results = await Promise.allSettled(cards.map(async (node) => ({ node, plan: await request("plan", { target: node.dataset.switchTarget }) })));
          if (!current() || ticket !== sequence) return;
          let failed = false;
          results.forEach((result) => {
            if (result.status === "fulfilled") showCard(result.value.node, result.value.plan);
            else failed = true;
          });
          setState(failed ? jvT("ui.f2d442c804c0") : "", failed);
        } catch (failure) { if (current() && ticket === sequence) setState(failure.message, true); }
      };
      cards.forEach((node) => node.addEventListener("click", () => { if (!node.disabled) void select(node.dataset.switchTarget); }));
      if (server) void load();
    },
  };
})();
