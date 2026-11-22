const versionLauncher = document.querySelector("[data-version-open]");
const versionDialog = document.querySelector("[data-version-workspace-dialog]");
if (versionLauncher && versionDialog) {
  const content = versionDialog.querySelector("[data-version-content]");
  const state = versionDialog.querySelector("[data-version-state]");
  const tabs = Array.from(versionDialog.querySelectorAll("[data-version-tab]"));
  const csrf = document.querySelector("[data-minecraft-workspace-csrf]")?.value || "";
  let currentTab = "container";
  let sequence = 0;
  let updateOperation = "";
  const operationKinds = new Map();
  let completedOperation = null;
  const request = async (url, options = {}) => {
    const selection = globalThis.justVoxelSelectionGeneration.version;
    const boundURL = options.method && options.method !== "GET" ? (globalThis.justVoxelSelectedURL?.(url, "version") ?? url) : null;
    try {
      if (globalThis.justVoxelSelectionReady) await globalThis.justVoxelSelectionReady;
      if (selection !== globalThis.justVoxelSelectionGeneration.version) throw globalThis.justVoxelStaleSelection();
      const response = await fetch(boundURL || (globalThis.justVoxelSelectedURL?.(url, "version") ?? url), { credentials: "same-origin", cache: "no-store", ...options });
      const payload = await response.json();
      if (selection !== globalThis.justVoxelSelectionGeneration.version) throw globalThis.justVoxelStaleSelection();
      if (!response.ok) throw new Error(payload.error || jvT("ui.f0bee120b3cf"));
      return payload;
    } catch (failure) {
      if (selection !== globalThis.justVoxelSelectionGeneration.version) throw globalThis.justVoxelStaleSelection();
      throw failure;
    }
  };
  const statusURL = (policy = "", version = "", server_type = "", image_tag = "") => "/api/version/workspace/status?" + new URLSearchParams({ policy, version, server_type, image_tag });
  const policyLabel = (policy) => ({ recommended: jvT("ui.d70604e84304"), latest: jvT("ui.8730d3c2022a"), pinned: jvT("ui.c36f2cb11a3a") })[policy] || jvT("ui.b764cdc0eab7");
  const releaseLabel = (channel) => ({ STABLE: jvT("ui.90ee305714d7"), BETA: jvT("ui.703390318bd5"), ALPHA: jvT("ui.b1a96dd646bc"), RELEASE: jvT("ui.e020e3c67bd0") })[channel] || jvT("ui.a90d2f735a49");
  const line = (label, value) => {
    const row = document.createElement("div");
    const name = document.createElement("span");
    const detail = document.createElement("strong");
    row.className = "version-detail-line";
    name.textContent = label;
    detail.textContent = value || jvT("ui.ca1844969742");
    row.append(name, detail);
    return row;
  };
  const card = (title) => {
    const panel = document.createElement("section");
    panel.className = "panel details";
    if (title) {
      const heading = document.createElement("h2");
      heading.textContent = title;
      panel.appendChild(heading);
    }
    return panel;
  };
  const stackLabel = (value) => ({ up_to_date: jvT("ui.ce29b7f85b9e"), updates_available: jvT("ui.8d3e019d3824"), waiting_for_compatibility: jvT("ui.bb624633b852"), unavailable: jvT("ui.ca1844969742"), managed_automatically: jvT("ui.f4a2e2623d12"), disabled: jvT("ui.0ef3eea6c57d"), not_applicable: jvT("ui.5237c96ac1a2") })[value] || jvT("ui.ca1844969742");
  const operationFrame = (operation) => {
    const succeeded = operation.state === "succeeded";
    const failed = ["failed", "needs_attention"].includes(operation.state);
    const resolved = operation.state === "resolved";
    const kind = operationKinds.get(operation.operation_id);
    if (kind === "container") {
      const warning = succeeded && operation.status?.includes("previous-image history needs attention");
      const text = succeeded ? warning ? jvT("ui.bc08501e054c") : jvT("ui.d5573785cee5")
        : resolved ? jvT("ui.d2b9b8d0885a")
        : failed ? jvT("ui.cad0261e7933") : operation.status || jvT("ui.d17ed1a46e05");
      const progress = card("");
      progress.className += " version-operation-status " + (warning ? "notice warning" : succeeded || resolved ? "is-succeeded" : failed ? "is-failed" : "is-running");
      progress.dataset.versionOperation = "";
      progress.setAttribute("role", "status");
      const notice = document.createElement("p");
      notice.textContent = text;
      progress.appendChild(notice);
      return progress;
    }
    const heading = succeeded ? kind === "switch" ? jvT("ui.2cf89fea227d") : kind === "container" ? jvT("ui.2dfed4440124") : jvT("ui.20aaf566fb86")
      : failed ? kind === "container" ? jvT("ui.6e9be83ab512") : jvT("ui.6f8f3eb32fa4") : resolved ? jvT("ui.9ca3ce3d4866") : jvT("ui.b852972dcfed");
    const progress = card(heading);
    progress.className += " version-operation-status " + (succeeded || resolved ? "is-succeeded" : failed ? "is-failed" : "is-running");
    progress.dataset.versionOperation = "";
    progress.setAttribute("role", "status");
    const notice = document.createElement("p");
    notice.textContent = kind === "container" && failed ? jvT("ui.797a6281f280") : operation.status;
    progress.appendChild(notice);
    return progress;
  };
  const watchUpdate = async (id, selection = globalThis.justVoxelSelectionGeneration.version) => {
    if (selection !== globalThis.justVoxelSelectionGeneration.version) return;
    updateOperation = id;
    try {
      const result = await request("/api/version/workspace/update-operation?" + new URLSearchParams({ id }));
      const operation = result.operation;
      if (!operation) throw new Error(jvT("ui.13f772ac9bb2"));
      const progress = content.querySelector("[data-version-operation]");
      if (progress) {
        const refreshed = operationFrame(operation);
        progress.className = refreshed.className;
        progress.replaceChildren(...refreshed.children);
      }
      state.textContent = "";
      if (["succeeded", "resolved", "needs_attention", "failed"].includes(operation.state)) {
        updateOperation = "";
        completedOperation = operation;
        await render("", operation);
      } else {
        setTimeout(() => watchUpdate(id, selection), 2000);
      }
    } catch (failure) { if (failure.staleSelection) return; state.textContent = failure.message; updateOperation = ""; }
  };
  // Let the native select change event settle before changing form geometry.
  const settleSelectChange = () => new Promise((resolve) => setTimeout(resolve, 0));
  const requiresRuntimeUpdate = (plan, status) => Boolean(status?.selected_candidate && status.installed && (
    status.installed !== status.selected_candidate ||
    (!plan.changes?.length && status.update_available && status.stack_state === "updates_available")
  ));
  const showReview = (root, form, plan, params, status, players, lifecycle = {}) => {
    const selection = globalThis.justVoxelSelectionGeneration.version;
    const current = lifecycle.current || (() => selection === globalThis.justVoxelSelectionGeneration.version);
    const versionReview = Boolean(form.elements.policy);
    const channelChange = !versionReview && Boolean(plan.changes?.length);
    const needsUpdate = versionReview ? requiresRuntimeUpdate(plan, status) : status?.image_state === "updates_available";
    const runtimePossible = needsUpdate || channelChange;
    const blocked = (lifecycle.current && (!status?.plan_fingerprint || plan.review !== status.plan_fingerprint || status.image_tag !== params.get("image_tag") || !status.server_supported || !status.installed || status.stack_state === "unavailable")) || plan.compatibility_blocked || (status?.crossplay_enabled && !status.crossplay_compatible) || (versionReview && (!(status.server_supported ?? status.paper_supported) || !status.installed)) || (needsUpdate && !channelChange && status.stack_state !== "updates_available");
    root.querySelector("[data-version-review]")?.remove();
    const review = card(jvT("ui.6a47708f4e14"));
    review.dataset.versionReview = "";
    if (plan.changes?.length) {
      plan.changes.forEach((change) => {
        if (versionReview && change.field === "version") return;
        const item = document.createElement("p");
        const before = change.before === "LATEST" ? jvT("ui.e2a7672e6e42") : change.before || "—";
        const after = change.field === "version" && change.after === "LATEST" ? status?.selected_candidate || jvT("ui.ca1844969742") : change.after || "—";
        item.textContent = versionReview && change.field === "version_policy"
          ? jvT("ui.bc2067f98771", {v0: (policyLabel(change.before)), v1: (policyLabel(change.after))})
          : `${change.label || change.field}: ${before} → ${after}`;
        review.appendChild(item);
      });
    } else if (!needsUpdate) {
      const empty = document.createElement("p");
      empty.textContent = jvT("ui.65d6c045ab91");
      review.appendChild(empty);
    }
    if (channelChange) {
      const note = document.createElement("p");
      note.textContent = jvT("ui.2d937b71f50f");
      review.appendChild(note);
    }
    if (runtimePossible) {
      const grid = document.createElement("div");
      grid.className = "version-status-grid";
      if (versionReview) grid.append(line("Minecraft", `${status.installed} → ${status.selected_candidate}`), line("Geyser", stackLabel(status.geyser_state)), line("Floodgate", stackLabel(status.floodgate_state)), line("ViaVersion", stackLabel(status.viaversion_state)));
      else grid.append(line(jvT("ui.2fb5567b1a57"), needsUpdate ? jvT("ui.ff8b555d818f") : jvT("ui.47da28dcffcf")));
      review.appendChild(grid);
      const notice = document.createElement("p");
      notice.className = "notice warning";
      notice.textContent = (channelChange ? jvT("ui.0c56591be7ad") : "JustVoxel") + jvT("ui.7e912d22fe21");
      review.appendChild(notice);
    } else if (!versionReview && plan.restart_required) {
      const notice = document.createElement("p");
      notice.className = "notice warning";
      notice.textContent = jvT("ui.2e6a36ac6175");
      review.appendChild(notice);
    }
    (plan.warnings || []).forEach((warning) => {
      const notice = document.createElement("p");
      notice.className = "notice warning";
      notice.textContent = warning;
      review.appendChild(notice);
    });
    if (status?.crossplay_enabled && !status.crossplay_compatible) {
      const warning = document.createElement("p");
      warning.className = "notice warning";
      warning.textContent = (status.reason || jvT("ui.baf664e9a7b0")) + (versionReview ? jvT("ui.a812d7cc5fc2") : "");
      review.appendChild(warning);
    }
    if ((versionReview || lifecycle.current) && blocked && !(status.crossplay_enabled && !status.crossplay_compatible)) {
      const warning = document.createElement("p");
      warning.className = "notice warning";
      warning.textContent = status.reason || jvT("ui.743e15023cc9");
      review.appendChild(warning);
    }
    const confirmationRequired = versionReview ? plan.confirmation_required || (needsUpdate && Number(players?.online || 0) > 0) : runtimePossible && Number(players?.online || 0) > 0;
    const checkbox = document.createElement("input");
    checkbox.type = "checkbox";
    checkbox.setAttribute("role", "switch");
    checkbox.checked = false;
    if (!versionReview && confirmationRequired) {
      const warning = document.createElement("p");
      warning.className = "notice warning";
      warning.textContent = jvT("ui.a9c6363f22c2", {v0: (players?.online || 0)});
      review.appendChild(warning);
    }
    if (confirmationRequired && !blocked) {
      const consent = document.createElement("label");
      consent.className = "minecraft-native-toggle system-ups-shutdown-switch";
      const label = document.createElement("span");
      label.textContent = jvT("ui.6003d12bdbd2");
      consent.append(label, checkbox);
      review.appendChild(consent);
    }
    const actions = document.createElement("div");
    actions.className = "action-row";
    const cancel = document.createElement("button");
    cancel.type = "button";
    cancel.className = "secondary";
    cancel.textContent = jvT("ui.87a8ca7f6bd8");
    cancel.addEventListener("click", () => { if (!current() || cancel.disabled) return; review.remove(); form.hidden = false; lifecycle.onCancel?.(); });
    actions.appendChild(cancel);
    if ((plan.changes?.length || needsUpdate) && !blocked) {
      const apply = document.createElement("button");
      apply.type = "button";
      apply.textContent = !versionReview && plan.confirmation_required ? jvT("ui.50978b7c144d") : jvT("ui.85045ccc0567");
      apply.disabled = confirmationRequired && !checkbox.checked;
      checkbox.addEventListener("change", () => { apply.disabled = confirmationRequired && !checkbox.checked; });
      apply.addEventListener("click", async () => {
        if (!current() || apply.disabled) return;
        apply.disabled = true;
        cancel.disabled = true;
        checkbox.disabled = true;
        state.textContent = jvT("ui.6604fb3144ab");
        let saved = false;
        let updateSent = false;
        try {
          if (confirmationRequired && !checkbox.checked) throw new Error(jvT("ui.c93679dc6a8f"));
          await lifecycle.beforeApply?.();
          if (!current()) return;
          // Both policy and channel reviews apply one immutable-ID operation.
          // The reviewed fingerprint is revalidated by the Agent; no re-plan loop.
          const body = new URLSearchParams({ csrf, plan_fingerprint: status.plan_fingerprint, confirm_players: checkbox.checked ? "yes" : "no", policy: params.get("version_policy") || "", version: params.get("version") || "", image_tag: params.get("image_tag") || "" });
          updateSent = true;
          const result = await request("/api/version/workspace/update", { method: "POST", headers: { "Content-Type": "application/x-www-form-urlencoded" }, body });
          if (!result.operation?.operation_id) throw new Error(jvT("ui.95e6869063f9"));
          review.remove();
          updateOperation = result.operation.operation_id;
          await render(jvT("ui.4c22f03fc5d1"));
          watchUpdate(result.operation.operation_id, selection);
          return;
        } catch (failure) {
          if (!current() || failure.staleSelection) return;
          if (lifecycle.current && updateSent) { await render(failure.message); return; }
          // A saved configuration is never automatically reapplied or retried.
          if (saved) await render(jvT("ui.33f2ce647b3a") + failure.message);
          else {
            state.textContent = failure.message;
            apply.disabled = confirmationRequired && !checkbox.checked;
            cancel.disabled = false;
            checkbox.disabled = false;
          }
        }
      });
      actions.appendChild(apply);
    }
    review.appendChild(actions);
    root.appendChild(review);
    form.hidden = true;
  };
  const wireForm = (root, form, minecraft) => {
    form.addEventListener("submit", async (event) => {
      event.preventDefault();
      state.textContent = jvT("ui.089810a04644");
      const channel = form.elements.channel?.value || minecraft.image_tag || "stable";
      const imageTag = channel === "custom" ? form.elements.custom_tag.value.trim() : channel;
      const policy = form.elements.policy?.value || minecraft.version_mode || "pinned";
      const version = policy === "pinned" ? (form.elements.version?.value.trim() || minecraft.version || "") : "";
      const params = new URLSearchParams({ csrf, tab: "version", image_tag: imageTag, version_policy: policy, version });
      try {
        const [plan, candidate, snapshot] = await Promise.all([
          request("/api/minecraft/workspace/settings/plan", { method: "POST", headers: { "Content-Type": "application/x-www-form-urlencoded" }, body: params }),
          request(statusURL(policy, version, "", imageTag)),
          request("/api/minecraft/workspace/snapshot"),
        ]);
        showReview(root, form, plan, params, candidate, snapshot?.players);
        state.textContent = "";
      } catch (failure) { if (failure.staleSelection) return; state.textContent = failure.message; }
    });
  };
  let containerCompleted = null;
  let containerSubmitted = null;
  const containerLabel = (value) => ({ up_to_date: jvT("ui.ce29b7f85b9e"), update_available: jvT("ui.ff8b555d818f"), image_missing: jvT("ui.146af0ad069a"), restart_needed: jvT("ui.b46871152667"), unavailable: jvT("ui.9f52572c6702") })[value] || jvT("ui.9f52572c6702");
  const renderContainer = async (ticket, selection, message) => {
    const current = () => ticket === sequence && selection === globalThis.justVoxelSelectionGeneration.version && currentTab === "container";
    const root = document.createElement("div"); root.className = "version-container-workspace";
    const panel = card(jvT("ui.05026dd1de3c"));
    panel.className += " version-container-panel";
    const identity = document.createElement("div");
    identity.className = "version-status-grid version-container-row version-container-summary";
    identity.dataset.containerRow = "identity";
    const selector = versionDialog.querySelector("[data-selected-server] select");
    const serverLine = line(jvT("ui.aef7de28d529"), selector?.selectedOptions?.[0]?.textContent || jvT("ui.ca1844969742"));
    const imageLine = line(jvT("ui.1aa4cb0bcca7"), jvT("ui.ec963ffc911b"));
    identity.append(serverLine, imageLine);
    const configuration = document.createElement("section"); configuration.className = "version-container-configuration";
    const configurationHeading = document.createElement("h2"); configurationHeading.textContent = jvT("ui.eeb8fa44bb6c");
    const controls = document.createElement("div");
    controls.className = "version-container-row version-container-controls";
    controls.dataset.containerRow = "controls";
    const toggle = (text) => {
      const label = document.createElement("label");
      label.className = "minecraft-native-toggle system-ups-shutdown-switch";
      const title = document.createElement("span"); title.textContent = text;
      const input = document.createElement("input"); input.type = "checkbox"; input.setAttribute("role", "switch");
      input.checked = false; input.disabled = true;
      label.append(title, input); controls.appendChild(label); return input;
    };
    const statusLine = line(jvT("ui.920e413c7d41"), jvT("ui.ec963ffc911b"));
    const update = document.createElement("button"); update.type = "button"; update.textContent = jvT("ui.c1c1009d3f37"); update.disabled = true;
    const actionLane = document.createElement("div"); actionLane.className = "version-container-action-lane";
    actionLane.appendChild(update); identity.appendChild(statusLine);
    const channelCell = document.createElement("label"); channelCell.className = "version-container-channel";
    const channelTitle = document.createElement("span"); channelTitle.textContent = jvT("ui.2b33abc68c21");
    const channel = document.createElement("select"); channel.name = "channel"; channel.setAttribute("aria-label", jvT("ui.e0e5ffeb622e")); channel.disabled = true;
    for (const [value, text] of [["stable", jvT("ui.cbb8e35b3719")], ["latest", jvT("ui.8730d3c2022a")], ["custom", jvT("ui.494ca78f7374")]]) {
      const option = document.createElement("option"); option.value = value; option.textContent = text; channel.appendChild(option);
    }
    const customTag = document.createElement("input"); customTag.name = "custom_tag"; customTag.autocomplete = "off"; customTag.setAttribute("aria-label", jvT("ui.64887ce5c3b0")); customTag.hidden = true;
    const runtimeTag = document.createElement("strong"); runtimeTag.textContent = jvT("ui.ec963ffc911b");
    const runtimeOwner = document.createElement("small"); runtimeOwner.className = "muted";
    channelCell.append(channelTitle, runtimeTag); controls.appendChild(channelCell);
    const retentionLabel = document.createElement("label"); retentionLabel.className = "version-container-retention";
    const retentionTitle = document.createElement("span"); retentionTitle.textContent = jvT("ui.0ece2ba780d7");
    const retention = document.createElement("input"); retention.type = "number"; retention.min = "1"; retention.max = "2"; retention.step = "1"; retention.value = "1"; retention.disabled = true;
    retentionLabel.append(retentionTitle, retention);
    controls.appendChild(retentionLabel);
    const restart = toggle(jvT("ui.78d2efb70368"));
    const backup = toggle(jvT("ui.61764fc034a9"));

    const library = document.createElement("section"); library.className = "version-container-history";
    const historyHeading = document.createElement("h2"); historyHeading.textContent = jvT("ui.5653ac0584a0");
    library.appendChild(historyHeading);
    const images = document.createElement("div"); images.className = "backup-file-grid";
    const footer = document.createElement("div"); footer.className = "backup-command-bar version-container-footer";
    const summary = document.createElement("div"); summary.className = "backup-library-summary";
    const metric = (label, value) => {
      const cell = document.createElement("div");
      const title = document.createElement("span"); title.textContent = label;
      const detail = document.createElement("strong"); detail.textContent = value;
      cell.append(title, detail); summary.appendChild(cell); return detail;
    };
    const imageCount = metric(jvT("ui.be7e2f201293"), "0");
    const totalSizeValue = metric(jvT("ui.51cdf6b25609"), "0 MB");
    const selectedCount = metric(jvT("ui.57fd7a0cf33f"), "0");
    const operationState = metric(jvT("ui.920e413c7d41"), jvT("ui.ec963ffc911b"));
    const historyActions = document.createElement("div"); historyActions.className = "backup-command-actions";
    const restore = document.createElement("button"); restore.type = "button"; restore.textContent = jvT("ui.536d8b1e30eb"); restore.disabled = true;
    historyActions.appendChild(restore); footer.append(summary, historyActions); library.append(images, footer);
    let selectedDigest = "";
    const imageSelectors = [];
    const sizeLabel = (bytes) => bytes > 0 ? bytes >= 1073741824 ? `${(bytes / 1073741824).toFixed(1)} GB` : `${Math.ceil(bytes / 1048576)} MB` : "";
    const helper = document.createElement("p"); helper.className = "muted compact"; helper.hidden = true;
    const progress = document.createElement("div");
    const terminal = document.createElement("pre"); terminal.className = "log-box"; terminal.dataset.containerTerminal = ""; terminal.hidden = true;
    terminal.textContent = jvT("ui.52f09fcd83c9");
    configuration.append(configurationHeading, controls, helper, actionLane);
    panel.append(identity, configuration, library);
    root.append(panel, progress, terminal);
    if (!current()) return;
    content.replaceChildren(root);
    let plan = { state: "unavailable", running: false };
    let operation;
    let busy = true;
    let persistedRetention = "1";
    let runtimeStatus;
    let movingChannel = false;
    let configuredTag = "";
    const proposedTag = () => channel.value === "custom" ? customTag.value.trim() : channel.value;
    const channelDirty = () => movingChannel && proposedTag() !== configuredTag;
    const channelValid = () => !movingChannel || Boolean(proposedTag());
    const retentionValid = () => /^[12]$/.test(retention.value);
    const retentionDirty = () => retention.value !== persistedRetention;
    const containerAction = () => ["update_available", "image_missing", "restart_needed"].includes(plan.state) && Boolean(plan.plan_fingerprint) && (plan.state !== "restart_needed" || restart.checked);
    const sync = () => {
      selectedCount.textContent = selectedDigest ? "1" : "0";
      operationState.textContent = operation?.state || containerLabel(plan.state);
      imageSelectors.forEach((input) => { input.disabled = busy; });
      const actionable = ["update_available", "image_missing", "restart_needed"].includes(plan.state) && Boolean(plan.plan_fingerprint);
      restart.disabled = busy || channelDirty() || !actionable || !plan.running;
      backup.disabled = busy || channelDirty() || !plan.running || (!actionable && !(plan.previous_images || []).length);
      channel.disabled = customTag.disabled = busy;
      customTag.hidden = channel.value !== "custom";
      retention.disabled = busy;
      restore.disabled = busy || !selectedDigest || !plan.restore_fingerprint;
      if (!plan.running && !containerSubmitted) restart.checked = backup.checked = false;
      update.disabled = busy || !channelValid() || !retentionValid() || (!channelDirty() && !containerAction() && !retentionDirty());
    };
    const channelChanged = () => {
      if (!current() || busy) return;
      state.textContent = channelValid() ? "" : jvT("ui.d764eaefaa91");
      sync();
    };
    channel.addEventListener("change", channelChanged);
    customTag.addEventListener("input", channelChanged);
    const saveRetention = async () => {
      if (!current() || !retentionValid()) throw new Error(jvT("ui.ac92e115ad98"));
      if (!retentionDirty()) return;
      const value = retention.value;
      const body = new URLSearchParams({ csrf, keep_previous_images: value });
      await request("/api/version/container/retention", { method: "POST", headers: { "Content-Type": "application/x-www-form-urlencoded" }, body });
      if (!current()) throw globalThis.justVoxelStaleSelection();
      persistedRetention = value;
    };
    retention.addEventListener("input", () => {
      if (!current() || busy) return;
      state.textContent = retentionValid() ? "" : jvT("ui.ac92e115ad98");
      sync();
    });
    backup.addEventListener("change", () => { if (backup.checked) restart.checked = true; sync(); });
    restart.addEventListener("change", () => { if (!restart.checked) backup.checked = false; sync(); });
    const trace = async (id) => {
      const response = await fetch(globalThis.justVoxelSelectedURL?.(`/api/operations/${encodeURIComponent(id)}/trace`, "version") ?? `/api/operations/${encodeURIComponent(id)}/trace`, { credentials: "same-origin", cache: "no-store" });
      const text = await response.text();
      if (!current()) return;
      if (!response.ok) throw new Error(jvT("ui.f2fdc0ec6e96"));
      terminal.textContent = text;
    };
    const watch = async (id) => {
      if (!current()) return;
      try {
        const result = await request("/api/version/container/operation?" + new URLSearchParams({ id }));
        if (!current()) return;
        operation = result.operation;
        if (!operation) throw new Error(jvT("ui.ae60d9f17bc7"));
        terminal.hidden = false;
        // Trace failure must not stop operation-state polling.
        try { await trace(id); } catch (failure) { if (current()) state.textContent = failure.message; }
        if (!current()) return;
        operationKinds.set(id, "container");
        progress.replaceChildren(operationFrame(operation));
        operationState.textContent = operation.state;
        if (["succeeded", "resolved", "needs_attention", "failed"].includes(operation.state)) {
          containerCompleted = operation;
          await render();
        } else setTimeout(() => watch(id), 2000);
      } catch (failure) { if (!current() || failure.staleSelection) return; state.textContent = failure.message; }
    };
    const submit = async (restoring) => {
      if (!current() || (restoring ? restore.disabled : update.disabled)) return;
      busy = true; sync();
      try {
        let confirmPlayers = false;
        const restartNow = restoring ? Boolean(plan.running) : restart.checked;
        if (restartNow) {
          const snapshot = await request("/api/minecraft/workspace/snapshot");
          if (!current()) return;
          const online = snapshot?.players?.online;
          if (!Number.isInteger(online) || online < 0) throw new Error(jvT("ui.7e92de910616"));
          if (online > 0) {
            confirmPlayers = window.confirm(jvT("ui.e5bfbb559bda"));
            if (!confirmPlayers) { busy = false; sync(); return; }
          }
        }
        const body = new URLSearchParams({ csrf, plan_fingerprint: restoring ? plan.restore_fingerprint : plan.plan_fingerprint, restart_now: restartNow ? "yes" : "no", backup_before_restart: backup.checked && restartNow ? "yes" : "no", confirm_players: confirmPlayers ? "yes" : "no" });
        if (restoring) body.set("restore_digest", selectedDigest);
        const result = await request("/api/version/container/update", { method: "POST", headers: { "Content-Type": "application/x-www-form-urlencoded" }, body });
        if (!current()) return;
        if (!result.operation?.operation_id) throw new Error(jvT("ui.9582b3fa3ee2"));
        containerCompleted = null;
        containerSubmitted = { selection, operationID: result.operation.operation_id, restart: restartNow, backup: body.get("backup_before_restart") === "yes" };
        operation = result.operation;
        terminal.hidden = false;
        operationKinds.set(operation.operation_id, "container");
        progress.replaceChildren(operationFrame(operation));
        operationState.textContent = operation.state;
        state.textContent = "";
        void watch(operation.operation_id);
      } catch (failure) {
        if (!current() || failure.staleSelection) return;
        // A lost POST response may still have started the operation.
        await render(failure.message);
      }
    };
    update.addEventListener("click", async () => {
      if (!current() || busy) return;
      if (!retentionValid()) { state.textContent = jvT("ui.ac92e115ad98"); return; }
      if (update.disabled) return;
      if (channelDirty()) {
        if (!channelValid()) return;
        busy = true; sync();
        state.textContent = jvT("ui.089810a04644");
        const imageTag = proposedTag();
        const policy = runtimeStatus.policy;
        const version = policy === "pinned" ? runtimeStatus.configured_version : "";
        const params = new URLSearchParams({ csrf, tab: "version", image_tag: imageTag, version_policy: policy, version });
        try {
          const [settingsPlan, candidate, snapshot] = await Promise.all([
            request("/api/minecraft/workspace/settings/plan", { method: "POST", headers: { "Content-Type": "application/x-www-form-urlencoded" }, body: params }),
            request(statusURL(policy, version, "", imageTag)),
            request("/api/minecraft/workspace/snapshot"),
          ]);
          if (!current()) return;
          // The existing review owns confirmation and the fingerprint-bound operation.
          showReview(root, { elements: {} }, settingsPlan, params, candidate, snapshot?.players, {
            current, beforeApply: saveRetention,
            onCancel: () => { busy = false; sync(); },
          });
          state.textContent = "";
        } catch (failure) {
          if (!current() || failure.staleSelection) return;
          state.textContent = failure.message; busy = false; sync();
        }
        return;
      }
      if (retentionDirty()) {
        busy = true; sync();
        try {
          await saveRetention();
          if (!current()) return;
          if (!containerAction()) { await render(); return; }
          busy = false; sync();
        } catch (failure) {
          if (!current() || failure.staleSelection) return;
          state.textContent = failure.message; busy = false; sync(); return;
        }
      }
      await submit(false);
    });
    restore.addEventListener("click", () => submit(true));
    try {
      const [status, pending, runtime, reviewedPending] = await Promise.all([
        request("/api/version/container/status").catch((failure) => { if (failure.staleSelection) throw failure; return { state: "unavailable", running: false }; }),
        request("/api/version/container/operation"),
        request(statusURL()).catch((failure) => { if (failure.staleSelection) throw failure; return null; }),
        request("/api/version/workspace/update-operation"),
      ]);
      if (!current()) return;
      plan = status;
      runtimeStatus = runtime;
      configuredTag = runtime?.configured_image_tag || runtime?.image_tag || "";
      movingChannel = Boolean(runtime && ["paper", "purpur", "vanilla"].includes(runtime.configured_server_type || runtime.server_type));
      if (movingChannel) {
        channelTitle.textContent = jvT("ui.e0e5ffeb622e");
        channel.value = ["stable", "latest"].includes(configuredTag) ? configuredTag : "custom";
        customTag.value = channel.value === "custom" ? configuredTag : "";
        channelCell.replaceChildren(channelTitle, channel, customTag);
      } else {
        runtimeTag.textContent = runtime?.image_tag || configuredTag || jvT("ui.ca1844969742");
        runtimeOwner.textContent = runtime ? jvT("ui.96419c5b7421", {v0: (runtime.server_software || runtime.server_type)}) : jvT("ui.b523bf4b424f");
        channelCell.appendChild(runtimeOwner);
      }
      persistedRetention = String(status.keep_previous_images ?? 1);
      retention.value = persistedRetention;
      const retained = Array.isArray(status.previous_images) ? status.previous_images : [];
      let totalSize = 0;
      for (const image of retained) {
        const entry = document.createElement("label"); entry.className = "backup-file-card backup-file-selectable";
        const select = document.createElement("input"); select.type = "checkbox"; select.setAttribute("role", "switch"); select.className = "backup-file-checkbox"; select.name = "previous-container-image"; select.value = image.digest; select.setAttribute("aria-label", jvT("ui.048ca469b1f2") + image.digest);
        imageSelectors.push(select);
        select.addEventListener("change", () => {
          if (!current() || busy) return;
          selectedDigest = select.checked ? image.digest : "";
          imageSelectors.forEach((input) => { input.checked = input === select && selectedDigest !== ""; });
          sync();
        });
        const icon = document.createElement("div"); icon.className = "backup-file-icon"; icon.setAttribute("aria-hidden", "true");
        // Reuse the Backups file icon, with explicit bounds even before styles load.
        const svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
        svg.setAttribute("viewBox", "0 0 24 24"); svg.setAttribute("width", "18"); svg.setAttribute("height", "18");
        svg.setAttribute("fill", "none"); svg.setAttribute("stroke", "currentColor");
        for (const geometry of ["M4 4h11l5 5v11H4z", "M15 4v5h5", "M8 14h8M8 17h6"]) {
          const path = document.createElementNS("http://www.w3.org/2000/svg", "path");
          path.setAttribute("d", geometry); svg.appendChild(path);
        }
        icon.appendChild(svg);
        const details = document.createElement("div"); details.className = "backup-file-copy";
        const name = document.createElement("strong"); name.textContent = image.image;
        const metadata = document.createElement("span"); metadata.className = "muted";
        const size = sizeLabel(image.size); totalSize += image.size || 0;
        metadata.textContent = `${image.digest.slice(0, 19)}… · ${new Date(image.retained_at).toLocaleString()}${size ? " · " + size : ""}`;
        metadata.title = image.digest;
        details.append(name, metadata); entry.append(select, icon, details); images.appendChild(entry);
      }
      imageCount.textContent = String(retained.length);
      totalSizeValue.textContent = sizeLabel(totalSize) || "0 MB";
      if (!retained.length) { const empty = document.createElement("p"); empty.className = "backup-library-empty"; empty.textContent = "No previous container images."; images.appendChild(empty); }
      if (status.server_name) serverLine.replaceChildren(...line(jvT("ui.aef7de28d529"), status.server_name).children);
      imageLine.replaceChildren(...line(jvT("ui.1aa4cb0bcca7"), status.image).children);
      statusLine.replaceChildren(...line(jvT("ui.920e413c7d41"), containerLabel(status.state)).children);
      helper.hidden = status.running || status.state === "unavailable";
      helper.textContent = jvT("ui.f933cfc15a2f");
      operation = pending.operation || containerCompleted;
      busy = Boolean(operation && !["succeeded", "resolved"].includes(operation.state));
      if (containerSubmitted && (containerSubmitted.selection !== selection || containerSubmitted.operationID !== operation?.operation_id || ["succeeded", "resolved"].includes(operation?.state))) containerSubmitted = null;
      if (containerSubmitted) {
        restart.checked = containerSubmitted.restart;
        backup.checked = containerSubmitted.backup;
      }
      terminal.textContent = status.state === "unavailable" ? jvT("ui.a44ca9960a1b") : `Checking ${status.image}...\n${containerLabel(status.state)}.`;
      if (operation) {
        terminal.hidden = false;
        operationKinds.set(operation.operation_id, "container");
        progress.replaceChildren(operationFrame(operation));
        operationState.textContent = operation.state;
        try { await trace(operation.operation_id); } catch (failure) { if (current()) state.textContent = failure.message; }
        if (!current()) return;
        if (["needs_attention", "failed"].includes(operation.state)) {
          const acknowledge = document.createElement("button"); acknowledge.type = "button";
          acknowledge.textContent = jvT("ui.3ae61697a3ce");
          acknowledge.addEventListener("click", async () => {
            if (!current()) return;
            acknowledge.disabled = true;
            try {
              const body = new URLSearchParams({ csrf, operation_id: operation.operation_id });
              await request("/api/version/container/acknowledge", { method: "POST", headers: { "Content-Type": "application/x-www-form-urlencoded" }, body });
              if (!current()) return;
              containerCompleted = null; containerSubmitted = null; await render();
            } catch (failure) { if (!current() || failure.staleSelection) return; state.textContent = failure.message; acknowledge.disabled = false; }
          });
          historyActions.replaceChildren(acknowledge);
        } else if (!["succeeded", "resolved"].includes(operation.state)) void watch(operation.operation_id);
      }
      const reviewedOperation = reviewedPending.operation || completedOperation;
      if (reviewedOperation) {
        const reviewedProgress = operationFrame(reviewedOperation);
        progress.appendChild(reviewedProgress);
        const unresolved = !["succeeded", "resolved"].includes(reviewedOperation.state);
        busy = busy || unresolved;
        if (["needs_attention", "failed"].includes(reviewedOperation.state)) {
          const acknowledge = document.createElement("button"); acknowledge.type = "button";
          acknowledge.textContent = jvT("ui.3ae61697a3ce");
          acknowledge.addEventListener("click", async () => {
            if (!current() || acknowledge.disabled) return;
            acknowledge.disabled = true;
            try {
              const body = new URLSearchParams({ csrf, acknowledge: "yes", operation_id: reviewedOperation.operation_id });
              await request("/api/version/workspace/update", { method: "POST", headers: { "Content-Type": "application/x-www-form-urlencoded" }, body });
              if (!current()) return;
              completedOperation = null; await render();
            } catch (failure) { if (!current() || failure.staleSelection) return; state.textContent = failure.message; acknowledge.disabled = false; }
          });
          reviewedProgress.appendChild(acknowledge);
        } else if (unresolved && !updateOperation) void watchUpdate(reviewedOperation.operation_id, selection);
      }
      sync();
      state.textContent = message || "";
    } catch (failure) { if (!current() || failure.staleSelection) return; state.textContent = failure.message; statusLine.replaceChildren(...line(jvT("ui.920e413c7d41"), jvT("ui.9f52572c6702")).children); }
  };

  const render = async (message = "", confirmedOperation = completedOperation) => {
    const ticket = ++sequence;
    const selection = globalThis.justVoxelSelectionGeneration.version;
    await settleSelectChange();
    if (ticket !== sequence) return;
    if (currentTab === "container") return renderContainer(ticket, selection, message);
    state.textContent = jvT("ui.a7d51be33cff");
    try {
      const [settings, status, pending] = await Promise.all([
        request("/api/minecraft/workspace/settings"), request(statusURL()),
        request("/api/version/workspace/update-operation").catch((failure) => {
          return {};
        }),
      ]);
      if (ticket !== sequence) return;
      const minecraft = settings.minecraft || {};
      const root = document.createElement("div");
      let initializePreview;
      const operation = pending.operation || confirmedOperation;
      if (operation) {
        const progress = operationFrame(operation);
        if (["needs_attention", "failed"].includes(operation.state)) {
          updateOperation = "";
          const acknowledge = document.createElement("button");
          acknowledge.type = "button";
          acknowledge.textContent = jvT("ui.3ae61697a3ce");
          acknowledge.addEventListener("click", async () => {
            acknowledge.disabled = true;
            try {
              const body = new URLSearchParams({ csrf, acknowledge: "yes", operation_id: operation.operation_id });
              await request("/api/version/workspace/update", { method: "POST", headers: { "Content-Type": "application/x-www-form-urlencoded" }, body });
              completedOperation = null;
              await render(jvT("ui.ba2e340be87a"));
            } catch (failure) { if (failure.staleSelection) return; state.textContent = failure.message; acknowledge.disabled = false; }
          });
          progress.appendChild(acknowledge);
        } else if (!["succeeded", "resolved"].includes(operation.state) && !updateOperation) {
          watchUpdate(operation.operation_id);
        }
        root.appendChild(progress);
      }

      {
        const panel = card(jvT("ui.63285c4b1570"));
        const grid = document.createElement("div");
        grid.className = "version-status-grid";
        grid.append(line(jvT("ui.f8b32f4e92bd"), status.installed || jvT("ui.9f4a106271aa")), line(jvT("ui.edf73927136a"), status.available ? `${status.available} · ${releaseLabel(status.available_channel)}` : jvT("ui.ca1844969742")), line(jvT("ui.d70604e84304"), status.recommended), line(jvT("ui.84aebc69a1bf"), status.configured_version === "LATEST" ? jvT("ui.e2a7672e6e42") : status.configured_version), line(jvT("ui.8b8ee2755acb"), status.server_software || "Paper"), line(jvT("ui.e85847baf044"), status.image_tag), line(jvT("ui.dd167905de0d"), policyLabel(minecraft.version_mode)), line(jvT("ui.715b1458abbe"), stackLabel(status.stack_state)));
        grid.append(line(jvT("ui.2e189ad510ee"), stackLabel(status.minecraft_state)), line("Geyser", stackLabel(status.geyser_state)), line("Floodgate", stackLabel(status.floodgate_state)), line("ViaVersion", stackLabel(status.viaversion_state)));
        if (status.crossplay_enabled && status.policy === "recommended" && status.available && status.recommended && status.available !== status.recommended) {
          const newVersion = line(jvT("ui.a2918038ff27"), jvT("ui.473fc6e92641", {v0: (status.available)}));
          newVersion.className += " version-new-minecraft";
          grid.appendChild(newVersion);
        }
        panel.appendChild(grid);
        const compatibility = document.createElement("p");
        compatibility.className = status.crossplay_enabled && !status.crossplay_compatible ? "notice warning" : "muted compact";
        compatibility.textContent = status.reason || (status.stack_state === "up_to_date" ? jvT("ui.91a81585ce55") : status.stack_state === "updates_available" ? jvT("ui.a538aa9e9da3") : jvT("ui.24be41e9e99d"));
        panel.appendChild(compatibility);
        root.appendChild(panel);
        const policyPanel = card(jvT("ui.d6299af88c6c"));
        const form = document.createElement("form");
        form.innerHTML = `<div class="version-status-grid" data-version-preview></div><div class="version-selection-row"><label data-version-select><select name="policy" aria-label="${jvH("ui.dd167905de0d")}"><option value="recommended">${jvH("ui.d70604e84304")}</option><option value="latest">${jvH("ui.8730d3c2022a")}</option><option value="pinned">${jvH("ui.c36f2cb11a3a")}</option></select></label><button type="submit">${jvH("ui.6a47708f4e14")}</button></div><label data-version-specific>${jvH("ui.eda536a544fd")}<input name="version" autocomplete="off"></label>`;
        form.elements.policy.value = ["recommended", "latest", "pinned"].includes(minecraft.version_mode) ? minecraft.version_mode : "recommended";
        form.elements.version.value = minecraft.version === "LATEST" ? "" : minecraft.version || "";
        let previewSequence = 0;
        const sync = async () => {
          const previewTicket = ++previewSequence;
          await settleSelectChange();
          if (previewTicket !== previewSequence || !form.isConnected) return;
          const policy = form.elements.policy.value;
          form.querySelector("[data-version-specific]").hidden = policy !== "pinned";
          form.elements.version.required = policy === "pinned";
          const preview = form.querySelector("[data-version-preview]");
          preview.textContent = jvT("ui.01c5de9a31ec");
          try {
            const candidate = await request(statusURL(policy, policy === "pinned" ? form.elements.version.value.trim() : ""));
            await settleSelectChange();
            if (!preview.isConnected || previewTicket !== previewSequence) return;
            preview.replaceChildren(line(jvT("ui.edf73927136a"), candidate.available ? `${candidate.available} · ${releaseLabel(candidate.available_channel)}` : jvT("ui.ca1844969742")), line(policy === "latest" ? jvT("ui.cf1d521078ca") : jvT("ui.a504aecbe3a4"), candidate.selected_candidate), line(jvT("ui.a2f670a61d4d"), candidate.candidate_channel ? releaseLabel(candidate.candidate_channel) : jvT("ui.b764cdc0eab7")), line("Geyser/Floodgate", candidate.crossplay_enabled ? candidate.crossplay_compatible ? jvT("ui.53dacc76f111", {v0: (candidate.selected_candidate)}) : jvT("ui.8b6f0c63e3fb") : jvT("ui.0ef3eea6c57d")));
          } catch (_) {
            await settleSelectChange();
            if (preview.isConnected && previewTicket === previewSequence) preview.textContent = jvT("ui.86231411354e");
          }
        };
        form.elements.policy.addEventListener("change", sync);
        form.elements.version.addEventListener("change", sync);
        policyPanel.appendChild(form);
        root.appendChild(policyPanel);
        wireForm(root, form, minecraft);
        initializePreview = sync;
        form.querySelector("[data-version-specific]").hidden = form.elements.policy.value !== "pinned";
        form.elements.version.required = form.elements.policy.value === "pinned";
      }
      if (pending.operation) {
        root.querySelectorAll("form input, form select, form button").forEach((control) => { control.disabled = true; });
      }
      await settleSelectChange();
      if (ticket !== sequence) return;
      content.replaceChildren(root);
      initializePreview?.();
      state.textContent = operation && message === operation.status ? "" : message;
    } catch (failure) { if (failure.staleSelection) return; if (ticket === sequence) state.textContent = failure.message; }
  };
  window.addEventListener("justvoxel:server-selection-changed", (event) => {
    if (event.detail?.workspace !== "version") return;
    ++sequence;
    updateOperation = "";
    completedOperation = null;
    containerCompleted = null;
    containerSubmitted = null;
    operationKinds.clear();
    content.replaceChildren();
    if (versionDialog.open) void render();
  });
  const refreshWorkspace = () => globalThis.justVoxelServerSelection(versionDialog.querySelector("[data-server-selector-slot]"), true)
    .then(() => render()).catch(error => { state.textContent = error.message; });
  const windowControl = setupWorkspaceWindow(versionDialog, { onOpen: refreshWorkspace });
  versionLauncher.addEventListener("click", () => { document.querySelector("[data-control-center]")?.removeAttribute("open"); windowControl?.open(); });
  versionDialog.querySelector("[data-version-close]").addEventListener("click", () => windowControl?.close());
  versionDialog.querySelector("[data-version-refresh]").addEventListener("click", refreshWorkspace);
  tabs.forEach((tab) => tab.addEventListener("click", () => { currentTab = tab.dataset.versionTab; tabs.forEach((item) => item.setAttribute("aria-selected", String(item === tab))); render(); }));
  if (readWorkspaceWindowState("version").open) windowControl?.open();
}
