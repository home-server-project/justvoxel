(() => {
  const terminal = new Set(["succeeded", "resolved", "needs_attention", "failed"]);
  window.JustVoxelServerMigrationSwitch = {
    init(root) {
      if (!root?.matches("[data-migration-switch]") || root.dataset.switchInitialized) return;
      root.dataset.switchInitialized = "true";
      const server = root.dataset.serverId;
      const generation = globalThis.justVoxelSelectionGeneration?.migration;
      const current = () => root.isConnected && generation === globalThis.justVoxelSelectionGeneration?.migration;
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
        if (!current()) throw new Error("Selected server changed. Refresh Switch.");
        const query = new URLSearchParams({ server, ...(post ? {} : params) });
        const response = await fetch(`/api/migration/switch/${action}?${query}`, {
          method: post ? "POST" : "GET", credentials: "same-origin", cache: "no-store",
          ...(post ? { headers: { "Content-Type": "application/x-www-form-urlencoded" }, body: new URLSearchParams({ csrf, ...params }) } : {}),
        });
        const result = await response.json();
        if (!current()) throw new Error("Selected server changed. Refresh Switch.");
        if (!response.ok) throw new Error(result.error || "Switch is unavailable.");
        return result;
      };
      const paragraph = (text, className = "") => {
        const node = document.createElement("p"); node.textContent = text; node.className = className; return node;
      };
      const showCard = (node, plan) => {
        const active = plan.current === node.dataset.switchTarget;
        const available = !active && Boolean(plan.convert || plan.clone);
        node.classList.toggle("is-current", active);
        node.classList.toggle("is-available", available);
        node.classList.toggle("is-unsupported", !active && !available);
        node.disabled = active || !available || busy;
        node.querySelector("[data-switch-sticker]").textContent = active ? "Current" : available ? "Available" : "Not Supported";
        node.title = plan.reason || (active ? "Current server software" : `Available for Minecraft ${plan.version}`);
        const reason = node.querySelector("[data-switch-reason]");
        if (reason) { reason.textContent = plan.reason || ""; reason.hidden = active || available || !plan.reason; }
      };
      const reconnect = async (id = "") => {
        const result = await request("operation", { operation_id: id });
        operation = result.operation || null;
        busy = Boolean(operation && !["succeeded", "resolved"].includes(operation.state));
        if (!operation) { progress.replaceChildren(); return; }
        progress.replaceChildren(paragraph(operation.status || "Switch is in progress.", "notice"));
        if (operation.state === "succeeded" && result.clone_id) {
          progress.appendChild(paragraph("Clone created and verified. The new server is stopped and can be renamed normally."));
          const link = document.createElement("a"); link.className = "button-link";
          link.href = `/?server=${encodeURIComponent(result.clone_id)}`; link.textContent = "Open cloned server"; progress.appendChild(link);
        }
        if (["needs_attention", "failed"].includes(operation.state)) {
          const label = document.createElement("label"); label.className = "minecraft-native-toggle";
          const consent = document.createElement("input"); consent.type = "checkbox"; consent.setAttribute("role", "switch");
          label.append(paragraph("Allow recovery to restore the original server’s previous state and clean up a failed clone when present."), consent);
          const recover = document.createElement("button"); recover.type = "button"; recover.textContent = "Recover Switch"; recover.disabled = true;
          consent.addEventListener("change", () => { recover.disabled = !consent.checked; });
          recover.addEventListener("click", async () => {
            if (!current() || !consent.checked || recover.disabled) return;
            recover.disabled = true;
            try { await request("recover", { operation_id: operation.operation_id, confirm: "yes" }, true); if (current()) await load(); }
            catch (failure) { if (current()) { state.textContent = failure.message; recover.disabled = !consent.checked; } }
          });
          progress.append(label, recover);
        }
        cards.forEach((node) => { if (busy) node.disabled = true; });
        if (!terminal.has(operation.state) && !polling) {
          polling = true;
          setTimeout(async () => {
            polling = false;
            if (!current()) return;
            try { await reconnect(operation.operation_id); if (terminal.has(operation.state)) await load(operation.operation_id); }
            catch (failure) { if (current()) state.textContent = `${failure.message} Refresh Switch to reconnect. No conversion was retried.`; }
          }, 2000);
        }
      };
      const select = async (target) => {
        if (busy) return;
        const ticket = ++sequence;
        review.hidden = true;
        state.textContent = "Inspecting current server compatibility…";
        try {
          // Selection always re-reads authoritative inventory, even if a prior
          // catalogue response had made this card available.
          const plan = await request("plan", { target });
          if (!current() || ticket !== sequence) return;
          showCard(cards.find((node) => node.dataset.switchTarget === target), plan);
          cards.forEach((node) => node.classList.toggle("is-selected", node.dataset.switchTarget === target));
          review.replaceChildren();
          review.append(paragraph(`Current software: ${plan.current}`), paragraph(`Target software: ${plan.target}`), paragraph(`Minecraft version: ${plan.version || "Unverified"}`),
            paragraph(`Convert existing server: ${plan.convert ? "Available" : "Unavailable"}`), paragraph(`Clone and convert: ${plan.clone ? "Available" : "Unavailable"}`));
          if (plan.reason) review.appendChild(paragraph(plan.reason, "notice warning"));
          for (const item of plan.content || []) {
            const label = { verified: "Verified compatible", incompatible: "Incompatible", unknown: "Unknown / unverified" }[item.compatibility] || "Unknown / unverified";
            review.appendChild(paragraph(`${item.name} (${item.filename}): ${label}. ${item.reason}`, item.compatibility === "verified" ? "muted" : "notice warning"));
          }
          for (const warning of plan.warnings || []) review.appendChild(paragraph(warning, "notice warning"));
          const plugins = document.createElement("a"); plugins.href = `/?workspace=plugins&server=${encodeURIComponent(server)}`; plugins.className = "button-link secondary"; plugins.textContent = "Open Plugins & Mods"; review.appendChild(plugins);
          if (plan.convert || plan.clone) {
            review.appendChild(paragraph("Online players receive the normal shutdown countdown. JustVoxel makes a cold backup/copy and verifies the exact target runtime. Switch keeps this Minecraft version fixed. A clone uses separate storage and new ports automatically; the original is preserved.", "notice"));
            const label = document.createElement("label"); label.className = "minecraft-native-toggle system-ups-shutdown-switch";
            const consent = document.createElement("input"); consent.type = "checkbox"; consent.setAttribute("role", "switch");
            label.append(paragraph("Confirm reviewed conversion and any player interruption."), consent); review.appendChild(label);
            const actions = document.createElement("div"); actions.className = "action-row";
            for (const [action, text, allowed] of [["convert", "Convert existing server", plan.convert], ["clone", "Clone and convert", plan.clone]]) {
              const button = document.createElement("button"); button.type = "button"; button.textContent = text; button.disabled = true;
              if (!allowed) button.title = plan.reason || "Compatibility cannot be verified.";
              consent.addEventListener("change", () => { button.disabled = !allowed || !consent.checked || busy; });
              button.addEventListener("click", async () => {
                if (!current() || !allowed || !consent.checked || busy || button.disabled) return;
                busy = true; actions.querySelectorAll("button").forEach((item) => { item.disabled = true; }); cards.forEach((item) => { item.disabled = true; });
                try {
                  const result = await request("apply", { target, action, plan_fingerprint: plan.plan_fingerprint, confirm: "yes" }, true);
                  if (!result.operation?.operation_id) throw new Error("Switch tracking is unavailable.");
                  review.hidden = true; await reconnect(result.operation.operation_id);
                } catch (failure) {
                  if (!current()) return;
                  state.textContent = failure.message;
                  // A lost Apply response may have started work. Reconnect
                  // before enabling another review or action.
                  try { await reconnect(); if (!operation) { busy = false; await load(); state.textContent = failure.message; } }
                  catch (_) { state.textContent = `${failure.message} Refresh Switch to reconnect before trying again.`; }
                }
              });
              actions.appendChild(button);
            }
            review.appendChild(actions);
          }
          review.hidden = false; state.textContent = "";
        } catch (failure) { if (current() && ticket === sequence) state.textContent = failure.message; }
      };
      const load = async (id = "") => {
        const ticket = ++sequence;
        review.hidden = true;
        cards.forEach((node) => { node.disabled = true; node.classList.toggle("is-selected", false); });
        state.textContent = "Checking the selected server…";
        try {
          await reconnect(id);
          const results = await Promise.allSettled(cards.map(async (node) => ({ node, plan: await request("plan", { target: node.dataset.switchTarget }) })));
          if (!current() || ticket !== sequence) return;
          let failed = false;
          results.forEach((result) => {
            if (result.status === "fulfilled") showCard(result.value.node, result.value.plan);
            else failed = true;
          });
          state.textContent = failed ? "Some compatibility information is unavailable. Refresh Switch to check again." : "";
        } catch (failure) { if (current() && ticket === sequence) state.textContent = failure.message; }
      };
      cards.forEach((node) => node.addEventListener("click", () => { if (!node.disabled) void select(node.dataset.switchTarget); }));
      void load();
    },
  };
})();
