// Normalize retired dashboard tab URLs before reading explicit workspace links.
const selectionDashboard = typeof document !== "undefined" && typeof document.querySelector === "function"
  ? document.querySelector("#dashboard[data-dashboard-status]") : null;
const dashboardURL = selectionDashboard?.dataset?.dashboardUrl;
if (dashboardURL) window.history.replaceState(window.history.state, "", dashboardURL + window.location.hash);

const workspaceUsername = selectionDashboard?.dataset.workspaceUsername || "";
const workspaceWindowStateKey = (id) => workspaceUsername
  ? `justvoxel-workspace-window-${id === "system-monitor" ? "v2" : "v1"}:${encodeURIComponent(workspaceUsername)}:${id}`
  : null;

const workspaceRestoreAllowed = () => !["/login", "/password"].includes(window.location.pathname) && !document.querySelector("[data-credential-boundary]");

const readWorkspaceWindowState = (id) => {
  const key = workspaceWindowStateKey(id);
  if (!key) return {};
  try {
    const saved = JSON.parse(window.localStorage.getItem(key) || "{}");
    return workspaceRestoreAllowed() ? saved : { ...saved, open: false };
  } catch (_) {
    return {};
  }
};

const writeWorkspaceWindowState = (id, patch) => {
  const key = workspaceWindowStateKey(id);
  if (!key) return;
  try {
    const current = readWorkspaceWindowState(id);
    window.localStorage.setItem(key, JSON.stringify({ ...current, ...patch }));
  } catch (_) {
    // Layout persistence is optional. The window still works without browser storage.
  }
};

// Browser presentation state only; operation URLs retain their explicit targets.
const serverWorkspaces = ["minecraft", "version", "plugins", "backups", "migration"];
const workspaceServerSelections = Object.create(null);
globalThis.justVoxelSelectionGeneration = Object.fromEntries(serverWorkspaces.map(workspace => [workspace, 0]));
globalThis.justVoxelStaleSelection = () => Object.assign(new Error(jvT("ui.fcf68c31fea8")), { staleSelection: true });
const initialServerDestination = new URLSearchParams(globalThis.location?.search || "");
const initialServerWorkspace = initialServerDestination.get("workspace") || "minecraft";
const validServerID = (id) => typeof id === "string" && /^jv-[a-z0-9]{6,12}$/.test(id);

let serverSelectionInventoryLoading = false;
globalThis.justVoxelLoadServerSelections = (refresh = false) => {
  if (globalThis.justVoxelSelectionReady && (!refresh || serverSelectionInventoryLoading)) return globalThis.justVoxelSelectionReady;
  serverSelectionInventoryLoading = true;
  const inventoryTickets = {...globalThis.justVoxelSelectionGeneration};
  const slotTickets = new Map(Array.from(document.querySelectorAll("[data-server-selector-slot][data-server-workspace]"), slot => [slot, slot.dataset.inventoryGeneration]));
  globalThis.justVoxelSelectionReady = (async () => {
    // Never send a saved or URL server ID to the inventory endpoint.
    const response = await fetch("/api/minecraft/workspace/selection?selection=none", {credentials:"same-origin",cache:"no-store"});
    const data = await response.json();
    if (!response.ok) throw new Error(data.error || jvT("ui.ca05718b247c"));
    for (const workspace of serverWorkspaces) {
      if (inventoryTickets[workspace] !== globalThis.justVoxelSelectionGeneration[workspace]) continue;
      const previous = workspaceServerSelections[workspace];
      const saved = workspaceRestoreAllowed() ? readWorkspaceWindowState(workspace).selectedServer : "";
      const exists = (id) => validServerID(id) && data.servers.some(server => server.id === id);
      const explicit = initialServerWorkspace === workspace && initialServerDestination.getAll("server").length === 1 && exists(initialServerDestination.get("server"));
      const candidate = previous !== undefined ? previous : explicit ? initialServerDestination.get("server") : saved;
      if (saved && !exists(saved)) writeWorkspaceWindowState(workspace, {selectedServer: ""});
      const selectedServer = exists(candidate) ? candidate : data.servers.length === 1 ? data.servers[0].id : "";
      workspaceServerSelections[workspace] = selectedServer;
      if (selectedServer && workspaceRestoreAllowed()) writeWorkspaceWindowState(workspace, {selectedServer});
      if (workspace === "minecraft" && selectionDashboard) selectionDashboard.dataset.selectedMinecraftId = selectedServer;
      document.querySelectorAll(`[data-server-selector-slot][data-server-workspace="${workspace}"]`).forEach(slot => {
        if (slotTickets.get(slot) === slot.dataset.inventoryGeneration) renderServerSelector(slot, data.servers);
      });
      if (previous !== undefined && previous !== selectedServer) {
        ++globalThis.justVoxelSelectionGeneration[workspace];
        window.dispatchEvent(new CustomEvent("justvoxel:server-selection-changed", {detail: {serverId: selectedServer, workspace, inventoryEmpty: data.servers.length === 0}}));
      }
    }
    return data;
  })().finally(() => { serverSelectionInventoryLoading = false; });
  return globalThis.justVoxelSelectionReady;
};
if (selectionDashboard && workspaceRestoreAllowed()) {
  // Consumers await the same inventory validation before their first request.
  globalThis.justVoxelLoadServerSelections().catch(() => { globalThis.justVoxelSelectionReady = null; });
}
globalThis.justVoxelSelectedURL = (value, workspace = "minecraft") => {
  const location = globalThis.location;
  if (!location?.origin) return value;
  const target = new URL(value, location.origin);
  if (!target.searchParams.has("server")) {
    const selected = workspaceServerSelections[workspace];
    if (selected) target.searchParams.set("server", selected);
    else if (selectionDashboard) {
      if (target.pathname === "/api/dashboard-status") target.searchParams.set("selection", "none");
      else target.searchParams.set("server", "");
    } else if (initialServerWorkspace === workspace) {
      // Standalone selected-server pages retain their native explicit context.
      initialServerDestination.getAll("server").forEach(id => target.searchParams.append("server", id));
    }
  }
  return target.pathname + target.search;
};
globalThis.justVoxelSelectServer = (id, workspace, tabName, navigate = true) => {
  if (!validServerID(id) || !serverWorkspaces.includes(workspace)) return false;
  const target = new URL(window.location.href);
  const previousId = workspaceServerSelections[workspace];
  if (navigate) {
    target.searchParams.set("server", id);
    target.searchParams.set("workspace", workspace);
    if (tabName) target.searchParams.set("tab", tabName);
    else target.searchParams.delete("tab");
    target.searchParams.delete("operation");
    window.history.replaceState(window.history.state, "", target);
  } else if (target.searchParams.get("workspace") === "minecraft" || (!target.searchParams.has("workspace") && target.searchParams.has("server"))) {
    target.searchParams.set("server", id);
    window.history.replaceState(window.history.state, "", target);
  }
  workspaceServerSelections[workspace] = id;
  document.querySelectorAll(`[data-server-selector-slot][data-server-workspace="${workspace}"]`).forEach(slot => slot.syncServerSelection?.(id));
  if (workspaceRestoreAllowed()) writeWorkspaceWindowState(workspace, {selectedServer: id});
  if (workspace === "minecraft" && selectionDashboard) selectionDashboard.dataset.selectedMinecraftId = id;
  if (previousId !== id) {
    ++globalThis.justVoxelSelectionGeneration[workspace];
    window.dispatchEvent(new CustomEvent("justvoxel:server-selection-changed", {
      detail: { serverId: id, workspace },
    }));
  }
  return true;
};
// One presentation for every workspace and the compact Quick Look slot.
const renderServerSelector = (slot, servers) => {
  const details = document.createElement("details");
  details.className = "server-selector";
  details.dataset.selectedServer = "";
  const summary = document.createElement("summary");
  summary.setAttribute("aria-label", jvT("ui.286524914b43"));
  const menu = document.createElement("div");
  menu.className = "server-selector-menu";
  // The top layer keeps the list outside scrolling workspace headers and dialogs.
  menu.setAttribute("popover", "auto");
  details.addEventListener("toggle", () => {
    if (!details.isConnected) return;
    if (details.open) {
      menu.showPopover();
      const bounds = summary.getBoundingClientRect();
      menu.style.width = `${bounds.width}px`;
      menu.style.left = `${Math.max(0, Math.min(bounds.left, window.innerWidth - bounds.width))}px`;
      menu.style.top = `${bounds.bottom + 3}px`;
      menu.style.maxHeight = `${Math.max(60, Math.min(240, window.innerHeight - bounds.bottom - 10))}px`;
    } else menu.hidePopover();
  });
  menu.addEventListener("toggle", () => {
    if (!menu.matches(":popover-open")) details.open = false;
  });
  const appendName = (element, server, fallback) => {
    if (server?.software_icon) {
      const icon = document.createElement("img");
      icon.className = "server-selector-icon";
      icon.src = server.software_icon;
      icon.alt = "";
      element.append(icon);
    }
    const name = document.createElement("span");
    name.textContent = server ? server.name : fallback;
    element.append(name);
  };
  const options = servers.map(server => {
    const option = document.createElement("button");
    option.type = "button";
    option.className = "server-selector-option";
    option.dataset.serverId = server.id;
    appendName(option, server);
    option.addEventListener("click", () => {
      if (!validServerID(server.id)) return;
      const dialog = slot.closest("[data-workspace-window]");
      const tab = dialog?.querySelector('[aria-selected="true"]');
      const tabName = tab?.getAttribute(`data-${slot.dataset.serverWorkspace}-tab`);
      globalThis.justVoxelSelectServer(server.id, slot.dataset.serverWorkspace, tabName, !Object.hasOwn(slot.dataset, "serverSelectorCompact"));
      details.open = false;
      summary.focus();
    });
    menu.append(option);
    return option;
  });
  slot.syncServerSelection = (id) => {
    summary.replaceChildren();
    appendName(summary, servers.find(server => server.id === id), servers.length ? jvT("ui.389e87e83148") : jvT("ui.5659061bb090"));
    options.forEach(option => option.setAttribute("aria-pressed", option.dataset.serverId === id ? "true" : "false"));
  };
  details.addEventListener("keydown", event => {
    if (event.key === "Escape") { details.open = false; summary.focus(); }
  });
  details.append(summary);
  details.append(menu);
  slot.syncServerSelection(workspaceServerSelections[slot.dataset.serverWorkspace] || "");
  slot.replaceChildren(details);
};
globalThis.justVoxelServerSelection = async (slot, refresh = false) => {
  if (!slot || slot.dataset.selectionLoading || (!refresh && slot.querySelector("[data-selected-server]"))) return;
  slot.dataset.selectionLoading = "true";
  try {
    await globalThis.justVoxelLoadServerSelections(true);
    // Inventory loading renders all slots together, including duplicate Minecraft slots.
  } finally {
    delete slot.dataset.selectionLoading;
  }
};
// Identity fallback belongs to the active reviewed workflow or administrator session.
const saveRequiredMinecraftIdentity = (csrf) => new Promise((resolve) => {
  const dialog = document.createElement("dialog");
  dialog.className = "system-action-dialog instance-id-dialog";
  const content = document.createElement("div");
  content.className = "system-action-dialog-content";
  const title = document.createElement("h2");
  title.textContent = jvT("ui.ff528c379e66");
  const form = document.createElement("form");
  form.className = "instance-id-form";
  const label = document.createElement("label");
  label.textContent = "jv- ";
  const input = document.createElement("input");
  input.required = true;
  input.minLength = 6;
  input.maxLength = 12;
  input.pattern = "[A-Za-z0-9]{6,12}";
  input.autocomplete = "off";
  input.setAttribute("aria-label", jvT("ui.54a29a10f39d"));
  input.title = jvT("ui.9c8cd5e227d7");
  const save = document.createElement("button");
  save.type = "submit";
  save.textContent = jvT("ui.1509f561f241");
  const feedback = document.createElement("p");
  feedback.textContent = jvT("ui.f2163827c1a2");
  feedback.setAttribute("role", "status");
  label.append(input);
  form.append(label, save);
  content.append(title, form, feedback);
  dialog.append(content);
  document.body.append(dialog);
  dialog.addEventListener("cancel", (event) => event.preventDefault());
  form.addEventListener("submit", async (event) => {
    event.preventDefault();
    if (!form.reportValidity()) return;
    save.disabled = true;
    try {
      const response = await fetch("/api/minecraft/workspace/identity/manual", {
        method: "POST", credentials: "same-origin", cache: "no-store",
        headers: {Accept: "application/json", "Content-Type": "application/x-www-form-urlencoded"},
        body: new URLSearchParams({csrf, suffix: input.value}).toString(),
      });
      const result = await response.json();
      if (!response.ok || result.status !== "available" || !result.id) {
        throw new Error(result.error || jvT("ui.3dca8c62e971"));
      }
      dialog.close();
      dialog.remove();
      resolve();
    } catch (error) {
      feedback.textContent = error?.message || jvT("ui.62e15a84129f");
      save.disabled = false;
    }
  });
  dialog.showModal();
  input.focus();
});
const resumeReviewedIdentitySubmission = async (send, csrf) => {
  const response = await send();
  if (response.status !== 409 || !response.headers.get("Content-Type")?.includes("application/json")) return response;
  const result = await response.clone().json();
  if (result.code !== "identity_required") return response;
  await saveRequiredMinecraftIdentity(csrf);
  return send();
};
const identityDashboard = document.querySelector('[data-instance-required="true"]');
if (identityDashboard) {
  void saveRequiredMinecraftIdentity(document.querySelector('input[name="csrf"]')?.value || "");
}


const enhancePasswordFields = (root = document) => {
  root.querySelectorAll('input[type="password"]:not([data-password-ui="off"]):not([data-password-reveal-ready])').forEach((input) => {
    input.dataset.passwordRevealReady = "true";
    const shell = document.createElement("span");
    shell.className = "password-field-shell";
    input.parentNode.insertBefore(shell, input);
    shell.appendChild(input);
    const button = document.createElement("button");
    button.type = "button";
    button.className = "password-reveal-button";
    button.setAttribute("aria-label", jvT("ui.6aeaa6a53d09"));
    button.innerHTML = '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M2.5 12s3.5-6 9.5-6 9.5 6 9.5 6-3.5 6-9.5 6-9.5-6-9.5-6Z"></path><circle cx="12" cy="12" r="2.8"></circle><path class="password-reveal-slash" d="M4 4l16 16"></path></svg>';
    button.addEventListener("click", () => {
      const showing = input.type === "text";
      input.type = showing ? "password" : "text";
      button.classList.toggle("is-showing", !showing);
      button.setAttribute("aria-label", showing ? jvT("ui.6aeaa6a53d09") : jvT("ui.a60a56c584b3"));
    });
    shell.appendChild(button);
    if (input.autocomplete === "new-password" && !/confirm/i.test(input.name)) {
      const strength = document.createElement("span");
      strength.className = "password-strength is-unacceptable";
      strength.setAttribute("role", "status");
      strength.textContent = jvT("ui.ff964aa9da4d");
      shell.after(strength);
      input.addEventListener("input", () => {
        const value = input.value;
        const classes = [/[a-z]/, /[A-Z]/, /[0-9]/, /[^a-zA-Z0-9]/].filter((pattern) => pattern.test(value)).length;
        const minimum = Math.max(8, input.minLength > 0 ? input.minLength : 8);
        const level = value.length < minimum || classes < 2 ? "unacceptable" : value.length >= 12 && classes >= 3 ? "strong" : "acceptable";
        strength.className = "password-strength is-" + level;
        strength.textContent = jvT("ui.2d81f08557a7") + (level === "unacceptable" ? jvT("ui.5e08fb869bdf") : level === "strong" ? jvT("ui.30a04fdcb6bb") : jvT("ui.4b9cfa1c7da0"));
      });
    }
  });
};

const initDestructiveConfirmations = (root = document) => {
  root.querySelectorAll("[data-destructive-confirmation]:not([data-destructive-ready])").forEach((control) => {
    control.dataset.destructiveReady = "true";
    const slider = control.querySelector("[data-destructive-slider]");
    const shell = control.querySelector("[data-destructive-slider-shell]");
    const text = control.querySelector("[data-destructive-slider-text]");
    const toggleRow = control.querySelector("[data-destructive-toggle-row]");
    const toggle = control.querySelector("[data-destructive-toggle]");
    const value = control.querySelector("[data-destructive-value]");
    const button = control.closest("form")?.querySelector("[data-destructive-submit]") || control.parentElement?.querySelector("[data-destructive-submit]");
    const idleText = control.dataset.destructiveIdle || jvT("ui.656bae9bfca6");

    const sync = () => {
      const progress = Math.max(0, Math.min(100, Number(slider?.value || 0)));
      shell?.style.setProperty("--confirm-progress", String(progress / 100));
      const armed = progress >= 100;
      shell?.classList.toggle("is-armed", armed);
      if (text) text.textContent = armed ? (control.dataset.destructiveArmed || jvT("ui.49670aa6ec95")) : idleText;
      if (toggleRow) toggleRow.hidden = !armed;
      if (!armed && toggle) toggle.checked = false;
      const confirmed = armed && Boolean(toggle?.checked);
      if (value) value.value = confirmed ? (control.dataset.confirmValue || "") : "";
      if (button) button.disabled = !confirmed;
      slider?.setAttribute("aria-valuetext", armed ? jvT("ui.f457978a1756") : progress + " percent");
    };
    slider?.addEventListener("input", sync);
    slider?.addEventListener("change", sync);
    toggle?.addEventListener("change", sync);
    control._justVoxelDestructiveSync = sync;
    sync();
  });
};

const initializeSharedWebUIControls = (root = document) => {
  enhancePasswordFields(root);
  initDestructiveConfirmations(root);
};
initializeSharedWebUIControls(document);
new MutationObserver((mutations) => {
  mutations.forEach((mutation) => mutation.addedNodes.forEach((node) => {
    if (node.nodeType !== Node.ELEMENT_NODE) return;
    initializeSharedWebUIControls(node);
  }));
}).observe(document.documentElement, { childList: true, subtree: true });

document.addEventListener("submit", (event) => {
  const form = event.target.closest(".action-form");
  if (!form) return;

  if (form.dataset.minecraftAction) form.action = globalThis.justVoxelSelectedURL(form.action, "minecraft");
  const submitted = form.querySelector('button[type="submit"]');
  document.querySelectorAll(".action-form button").forEach((button) => {
    button.disabled = true;
  });

  if (submitted && form.dataset.progress) {
    submitted.textContent = form.dataset.progress;
  }
});

const navigationGroups = Array.from(document.querySelectorAll(".nav-group"));
if (navigationGroups.length > 0) {
  navigationGroups.forEach((group) => {
    group.addEventListener("toggle", () => {
      if (!group.open) return;
      navigationGroups.forEach((other) => {
        if (other !== group) other.open = false;
      });
    });
  });

  document.addEventListener("click", (event) => {
    if (event.target.closest(".primary-nav")) return;
    navigationGroups.forEach((group) => {
      group.open = false;
    });
  });

  document.addEventListener("keydown", (event) => {
    if (event.key !== "Escape") return;
    navigationGroups.forEach((group) => {
      group.open = false;
    });
    const active = document.activeElement;
    if (active && active.closest && active.closest(".nav-menu")) {
      const group = active.closest(".nav-group");
      const trigger = group && group.querySelector(".nav-trigger");
      if (trigger) trigger.focus();
    }
  });

  const path = window.location.pathname;
  const brand = document.querySelector(".brand-link");
  if (brand) {
    if (path === "/") brand.setAttribute("aria-current", "page");
    else brand.removeAttribute("aria-current");
  }

  let currentGroup = "";
  let currentHref = "";
  if (path === "/password") {
    currentGroup = "administration";
    currentHref = "/password";
  }

  if (currentGroup) {
    const group = document.querySelector(`[data-nav-group="${currentGroup}"]`);
    const trigger = group && group.querySelector(".nav-trigger");
    if (trigger) trigger.setAttribute("aria-current", "page");
  }
  if (currentHref) {
    document.querySelectorAll(".nav-menu a").forEach((link) => {
      if (link.getAttribute("href") === currentHref) {
        link.setAttribute("aria-current", "page");
      }
    });
  }
}

const dashboard = document.querySelector("#dashboard[data-dashboard-status]");
if (dashboard) {
  const statusURL = "/api/dashboard-status";
  const sessionURL = dashboard.dataset.sessionInfo;
  let pendingAction = dashboard.dataset.pendingAction || "";
  let restartSawTransition = false;

  const text = (id, value) => {
    const el = document.getElementById(id);
    if (el) el.textContent = value;
  };

  const loadRole = async () => {
    try {
      const response = await fetch(sessionURL, {
        method: "GET",
        credentials: "same-origin",
        headers: { Accept: "application/json" },
        cache: "no-store",
      });
      if (response.status === 401) {
        window.location.assign("/login");
        return;
      }
      if (!response.ok) return;
      const identity = await response.json();
      const role = String(identity.role || "").toLowerCase();
      const username = String(identity.username || "");
      if (!["administrator", "operator", "viewer"].includes(role)) return;
      document.body.classList.remove("role-pending");
      document.body.classList.toggle("primary-administrator", username === "voxel" && role === "administrator");
      document.body.classList.add(`role-${role}`);
      text("session-role", role);
    } catch (_) {
      // Fail closed: role-gated controls remain hidden until identity is known.
    }
  };

  const actionProgress = (action) => {
    if (action === "start") return jvT("ui.bbe5fc3b9ef3");
    if (action === "stop") return jvT("ui.bbe8574175ab");
    if (action === "restart") return jvT("ui.75d0f1469d16");
    return jvT("ui.5474eef8d0f1");
  };

  const setButtons = (status) => {
    const running = status.minecraft.state === "Running";
    document.querySelectorAll("[data-minecraft-action]").forEach((form) => {
      const button = form.querySelector('button[type="submit"]');
      if (!button) return;
      if (pendingAction) {
        button.disabled = true;
        return;
      }
      const action = form.dataset.minecraftAction;
      if (action === "start") {
        button.disabled = !status.minecraft.configured || running;
      } else {
        button.disabled = !running;
      }
    });
  };

  const updatePendingState = (status, players) => {
    if (!pendingAction) return;

    if (status.minecraft.state === "Failed") {
      pendingAction = "";
      return;
    }

    if (pendingAction === "start") {
      if (status.minecraft.state === "Running" && players.state === "running") {
        pendingAction = "";
      }
      return;
    }

    if (pendingAction === "stop") {
      if (status.minecraft.state === "Stopped") {
        pendingAction = "";
      }
      return;
    }

    if (pendingAction === "restart") {
      if (status.minecraft.state !== "Running" || players.state !== "running") {
        restartSawTransition = true;
      }
      if (restartSawTransition && status.minecraft.state === "Running" && players.state === "running") {
        pendingAction = "";
      }
    }
  };

  const renderDashboardAttention = (attention) => {
    const panel = document.getElementById("dashboard-attention");
    const title = document.getElementById("dashboard-attention-title");
    const status = document.getElementById("dashboard-attention-status");
    const action = document.getElementById("dashboard-attention-action");
    if (!panel) return;
    panel.hidden = !attention;
    if (!attention) return;
    if (title) title.textContent = jvS(attention.title) || jvT("ui.1e1b96ec8fd9");
    if (status) status.textContent = jvS(attention.status) || "";
    if (action) {
      action.hidden = !attention.action;
      action.href = attention.action || "#";
      action.textContent = jvS(attention.label) || jvT("ui.aff0766a5290");
    }
  };

  const renderDashboard = (snapshot) => {
    const { status, players, attention } = snapshot;
    updatePendingState(status, players);
    renderDashboardAttention(attention || null);
    const setupArea = document.querySelector("[data-dashboard-setup-area]");
    if (setupArea) setupArea.hidden = Boolean(status.minecraft.configured);
    dashboard.dataset.configured = String(Boolean(status.minecraft.configured));

    const displayedState = pendingAction ? actionProgress(pendingAction) : status.minecraft.state;
    text("minecraft-control-state", displayedState);

    setButtons(status);

    if (!pendingAction && window.location.search.includes("result=")) {
      const target = new URL(window.location.href);
      target.searchParams.delete("result");
      window.history.replaceState(window.history.state, "", target);
    }
  };

  const refreshDashboard = async () => {
    try {
      if (globalThis.justVoxelSelectionReady) await globalThis.justVoxelSelectionReady;
      const selection = globalThis.justVoxelSelectionGeneration.minecraft;
      const response = await fetch(globalThis.justVoxelSelectedURL(statusURL, "minecraft"), {
        method: "GET",
        credentials: "same-origin",
        headers: { Accept: "application/json" },
        cache: "no-store",
      });
      if (response.status === 401) {
        window.location.assign("/login");
        return;
      }
      if (response.status === 403) {
        window.location.assign("/password");
        return;
      }
      if (!response.ok) return;
      const snapshot = await response.json();
      if (selection !== globalThis.justVoxelSelectionGeneration.minecraft) return;
      renderDashboard(snapshot);
    } catch (_) {
      // Keep the last known dashboard state and try again on the next poll.
    }
  };

  if (pendingAction) {
    const progress = actionProgress(pendingAction);
    text("minecraft-state-summary", progress);
    text("minecraft-control-state", progress);
    document.querySelectorAll("[data-minecraft-action] button").forEach((button) => {
      button.disabled = true;
    });
  }

  window.addEventListener("justvoxel:server-selection-changed", (event) => {
    if (event.detail?.workspace !== "minecraft") return;
    pendingAction = "";
    restartSawTransition = false;
    document.querySelectorAll("[data-minecraft-action]").forEach((form) => {
      const action = new URL(form.action, window.location.href);
      action.searchParams.set("server", dashboard.dataset.selectedMinecraftId);
      form.action = action.pathname + action.search;
    });
    document.querySelectorAll('a[href^="/minecraft/how-to-connect"]').forEach((link) => {
      const target = new URL(link.href, window.location.href);
      target.searchParams.set("server", dashboard.dataset.selectedMinecraftId);
      link.href = target.pathname + target.search;
    });
    void refreshDashboard();
  });
  globalThis.justVoxelSelectionReady?.then(() => {
    document.querySelectorAll("[data-minecraft-action]").forEach((form) => {
      const target = new URL(form.action, window.location.href);
      target.searchParams.set("server", dashboard.dataset.selectedMinecraftId);
      form.action = target.pathname + target.search;
    });
    document.querySelectorAll('a[href^="/minecraft/how-to-connect"]').forEach((link) => {
      const target = new URL(link.href, window.location.href);
      target.searchParams.set("server", dashboard.dataset.selectedMinecraftId);
      link.href = target.pathname + target.search;
    });
  }).catch(() => {});
  loadRole();
  refreshDashboard();
  window.setInterval(refreshDashboard, 5000);
  window.addEventListener("justvoxel:minecraft-reset-complete", () => void refreshDashboard());
}




document.querySelectorAll('a[href^="/minecraft/how-to-connect"]').forEach((link) => {
  link.href = globalThis.justVoxelSelectedURL(link.getAttribute("href"), "minecraft");
});

const quickLook = document.querySelector("[data-quick-look]");
const quickLookToggle = document.querySelector("[data-quick-look-toggle]");
if (quickLook && quickLookToggle) {
  const identityURL = "/api/session-info";
  const statusURL = "/api/dashboard-status";
  const actionMenus = Array.from(quickLook.querySelectorAll("[data-quick-look-actions]"));
  let identity = null;
  let refreshTimer = null;
  let lastMonitorData = null;
  const topbar = document.querySelector(".topbar");
  const quickLookPanel = quickLook.querySelector("[data-quick-look-panel]");

  const syncQuickLookTop = () => {
    const bounds = topbar?.getBoundingClientRect();
    quickLook.style.top = Math.round(bounds?.bottom || 48) + "px";
    if (bounds) {
      quickLook.style.left = bounds.left + "px";
      quickLook.style.width = bounds.width + "px";
      quickLook.style.right = "auto";
    }
  };
  syncQuickLookTop();
  window.addEventListener("resize", syncQuickLookTop);

  const stateKey = (username) => `justvoxel-quick-look-v1:${encodeURIComponent(username)}`;

  const setOpen = (open, persist = true) => {
    quickLook.classList.toggle("is-open", open);
    topbar?.classList.toggle("quick-look-open", open);
    quickLookToggle.setAttribute("aria-expanded", open ? "true" : "false");
    if (quickLookPanel) {
      quickLookPanel.inert = !open;
      quickLookPanel.setAttribute("aria-hidden", open ? "false" : "true");
    }
    if (identity?.role === "administrator" && persist) {
      try {
        window.localStorage.setItem(stateKey(identity.username), open ? "open" : "closed");
      } catch (_) {
        // Drawer state persistence is optional.
      }
    }
    if (open) {
      void globalThis.justVoxelServerSelection(quickLook.querySelector("[data-server-selector-slot]"), true)
        .then(() => refreshQuickLook()).catch(() => {});
      if (!refreshTimer) refreshTimer = window.setInterval(refreshQuickLook, 5000);
    } else if (refreshTimer) {
      window.clearInterval(refreshTimer);
      refreshTimer = null;
    }
  };

  const setValue = (selector, value) => {
    const node = quickLook.querySelector(selector);
    if (node) node.textContent = value || "—";
  };

  const backupCreate = quickLook.querySelector("[data-quick-look-backup-create]");
  const backupCSRF = quickLook.querySelector("[data-quick-look-backup-csrf]");
  backupCreate?.addEventListener("click", async () => {
    if (backupCreate.disabled) return;
    const selection = globalThis.justVoxelSelectionGeneration.minecraft;
    backupCreate.disabled = true;
    backupCreate.textContent = jvT("ui.bbe5fc3b9ef3");
    try {
      const response = await fetch(globalThis.justVoxelSelectedURL("/api/backups/manual", "minecraft"), {
        method: "POST",
        credentials: "same-origin",
        headers: { Accept: "application/json" },
        body: new URLSearchParams({ csrf: backupCSRF?.value || "" }),
      });
      if (response.status === 401) {
        window.location.assign("/login");
        return;
      }
      const result = await response.json();
      if (selection !== globalThis.justVoxelSelectionGeneration.minecraft) return;
      setValue("[data-quick-look-backup]", response.ok && result.ok
        ? result.message
        : result.error || jvT("ui.3c90992a4536"));
    } catch (_) {
      if (selection !== globalThis.justVoxelSelectionGeneration.minecraft) return;
      setValue("[data-quick-look-backup]", jvT("ui.3c90992a4536"));
    } finally {
      backupCreate.disabled = false;
      backupCreate.textContent = jvT("ui.85fb487f030a");
    }
  });

  const setActionAvailability = (status) => {
    const running = status?.minecraft?.state === "Running";
    quickLook.querySelectorAll("[data-minecraft-action]").forEach((form) => {
      const button = form.querySelector('button[type="submit"]');
      if (!button) return;
      const action = form.dataset.minecraftAction;
      button.disabled = action === "start"
        ? !status.minecraft.configured || running
        : !running;
    });
  };

  const renderQuickLook = (snapshot, providers = []) => {
    const status = snapshot?.status || {};
    const minecraft = status.minecraft || {};
    const players = snapshot?.players || {};
    const system = status.system || {};
    const backup = status.backup || {};

    setValue("[data-quick-look-minecraft]", jvS(minecraft.state));
    setValue("[data-quick-look-version]", minecraft.version);
    setValue(
      "[data-quick-look-players]",
      minecraft.configured ? `${players.online ?? 0} / ${players.max ?? minecraft.max_players ?? "—"}` : jvT("ui.dd1841d29502"),
    );
    setValue("[data-quick-look-system]", jvS(system.health));
    setValue("[data-quick-look-ipv4]", system.ipv4);
    const overlays = quickLook.querySelector("[data-quick-look-overlays]");
    if (overlays) {
      overlays.replaceChildren();
      [["Tailscale", "tailscale"], ["NetBird", "netbird"], ["Playit", "playit"], ["NordVPN", "nordvpn"]].forEach(([label, id]) => {
        const provider = providers.find((item) => item.id === id);
        if (provider?.connected !== true) return;
        let value = "";
        if (id === "playit") {
          const count = Array.isArray(provider.tunnels) ? provider.tunnels.filter((tunnel) => tunnel.is_disabled === false).length : 0;
          if (!count) return;
          value = `${count} tunnel${count === 1 ? "" : "s"}`;
        } else {
          const ip = typeof provider.ip === "string" ? provider.ip.split("/")[0] : "";
          if (!/^\d{1,3}(?:\.\d{1,3}){3}$/.test(ip) || ip.split(".").some((part) => Number(part) > 255)) return;
          value = ip;
        }
        const line = document.createElement("span");
        const name = document.createElement("span");
        name.textContent = label;
        const address = document.createElement("strong");
        address.textContent = value;
        line.append(name, address);
        overlays.appendChild(line);
      });
    }
    setValue("[data-quick-look-backup]", backup.enabled ? jvT("ui.92c1cdfdf4cb") : jvT("ui.75081b593d15"));
    setActionAvailability(status);
  };

  window.addEventListener("justvoxel:server-selection-changed", (event) => {
    if (event.detail?.workspace !== "minecraft") return;
    lastMonitorData = null;
    void refreshQuickLook();
  });
  async function refreshQuickLook() {
    try {
      if (globalThis.justVoxelSelectionReady) await globalThis.justVoxelSelectionReady;
      const selection = globalThis.justVoxelSelectionGeneration.minecraft;
      const response = await fetch(globalThis.justVoxelSelectedURL(statusURL, "minecraft"), {
        method: "GET",
        credentials: "same-origin",
        headers: { Accept: "application/json" },
        cache: "no-store",
      });
      if (response.status === 401) {
        window.location.assign("/login");
        return;
      }
      if (response.status === 403) {
        window.location.assign("/password");
        return;
      }
      if (!response.ok) return;
      const snapshot = await response.json();
      let providers = [];
      try {
        const remote = await fetch("/api/network/remote-access", {
          credentials: "same-origin", headers: { Accept: "application/json" }, cache: "no-store",
        });
        if (remote.ok) providers = (await remote.json()).providers || [];
      } catch (_) {
        // Missing provider evidence is presented conservatively.
      }
      if (selection !== globalThis.justVoxelSelectionGeneration.minecraft) return;
      renderQuickLook(snapshot, providers);
    } catch (_) {
      // Keep the last values and retry on the next refresh.
    }
  }

  const loadQuickLookIdentity = async () => {
    try {
      const response = await fetch(identityURL, {
        method: "GET",
        credentials: "same-origin",
        headers: { Accept: "application/json" },
        cache: "no-store",
      });
      if (response.status === 401) {
        window.location.assign("/login");
        return;
      }
      if (!response.ok) return;

      const result = await response.json();
      const role = String(result.role || "").toLowerCase();
      const username = String(result.username || "");
      if (!username || !["administrator", "operator", "viewer"].includes(role)) return;

      identity = { username, role };
      document.body.classList.remove("role-pending");
      document.body.classList.toggle("primary-administrator", username === "voxel" && role === "administrator");
      document.body.classList.add(`role-${role}`);

      if (role === "viewer" || role === "operator") {
        setOpen(true, false);
        return;
      }

      let saved = "closed";
      try {
        saved = window.localStorage.getItem(stateKey(username)) || "closed";
      } catch (_) {
        saved = "closed";
      }
      setOpen(saved === "open", false);
    } catch (_) {
      // Keep role-gated controls closed if identity cannot be loaded.
    }
  };

  quickLookToggle.addEventListener("click", () => {
    setOpen(!quickLook.classList.contains("is-open"));
  });

  document.addEventListener("click", (event) => {
    actionMenus.forEach((menu) => {
      if (menu.open && !menu.contains(event.target)) menu.open = false;
    });
  });

  loadQuickLookIdentity();
}

const controlCenter = document.querySelector("[data-control-center]");
let clockPreferences = { time: "12", date: "mdy" };
window.JustVoxelClockDisplay = {
  update: (profile) => {
    if (["12", "24"].includes(profile.time_format)) clockPreferences.time = profile.time_format;
    if (["mdy", "dmy"].includes(profile.date_format)) clockPreferences.date = profile.date_format;
    window.dispatchEvent(new Event("justvoxel-clock-display"));
  },
};

const topbarClock = document.querySelector("[data-topbar-clock]");
function formatDiagnosticLogTimestamp(value) {
  if (!value) return "";
  try {
    const instant = new Date(value);
    if (!Number.isFinite(instant.getTime())) return value;
    const timeZone = topbarClock?.dataset.systemTimezone || undefined;
    const parts = Object.fromEntries(new Intl.DateTimeFormat("en-US", { month: "short", day: "numeric", year: "numeric", timeZone }).formatToParts(instant).map(({ type, value }) => [type, value]));
    const date = clockPreferences.date === "dmy" ? `${parts.day} ${parts.month} ${parts.year}` : `${parts.month} ${parts.day} ${parts.year}`;
    const time = new Intl.DateTimeFormat("en-US", { hour: "2-digit", minute: "2-digit", ...(clockPreferences.time === "12" ? { hour12: true } : { hourCycle: "h23" }), timeZone }).format(instant);
    return `${date} · ${time}`;
  } catch (_) {
    return value;
  }
}

if (topbarClock) {
  const initialSystemTime = Number(topbarClock.dataset.systemNow);
  let systemClockOffset = Number.isFinite(initialSystemTime) && initialSystemTime > 0 ? initialSystemTime - Date.now() : 0;
  const updateTopbarClock = () => {
    const now = new Date(Date.now() + systemClockOffset);
    const zone = topbarClock.dataset.systemTimezone || undefined;
    const clockTime = new Intl.DateTimeFormat("en-US", { hour: "2-digit", minute: "2-digit", ...(clockPreferences.time === "12" ? { hour12: true } : { hourCycle: "h23" }), timeZone: zone }).format(now);
    const parts = Object.fromEntries(new Intl.DateTimeFormat("en-US", { month: "short", day: "numeric", year: "numeric", timeZone: zone }).formatToParts(now).map(({ type, value }) => [type, value]));
    const clockDate = clockPreferences.date === "dmy" ? `${parts.day} ${parts.month} ${parts.year}` : `${parts.month} ${parts.day} ${parts.year}`;
    const time = document.createElement("span");
    time.textContent = clockTime;
    const date = document.createElement("span");
    date.className = "topbar-clock-date";
    date.textContent = clockDate;
    const separator = document.createElement("span");
    separator.className = "topbar-clock-separator";
    separator.textContent = " · ";
    topbarClock.replaceChildren(time, separator, date);
    topbarClock.setAttribute("aria-label", `${clockTime} · ${clockDate}`);
    topbarClock.title = new Intl.DateTimeFormat([], {
      ...(clockPreferences.time === "12" ? { hour12: true } : { hourCycle: "h23" }),
      weekday: "long",
      year: "numeric",
      month: "long",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
      timeZone: topbarClock.dataset.systemTimezone || undefined,
    }).format(now);
  };
  const refreshClockDisplay = async () => {
    try {
      const response = await fetch("/api/session-info", { credentials: "same-origin", cache: "no-store" });
      if (response.ok) window.JustVoxelClockDisplay.update(await response.json());
    } catch (_) { /* Retry shared settings on the next refresh. */ }
  };
  refreshClockDisplay();
  window.addEventListener("focus", refreshClockDisplay);
  updateTopbarClock();
  window.setInterval(updateTopbarClock, 30000);
  window.addEventListener("justvoxel-clock-display", updateTopbarClock);
  window.addEventListener("justvoxel-timezone", (event) => {
    const { zone, localDate, localTime } = event.detail || {};
    if (/^\d{4}-\d{2}-\d{2}$/.test(localDate || "") && /^\d{2}:\d{2}$/.test(localTime || "") && zone) {
      const now = new Date(Date.now() + systemClockOffset);
      const parts = Object.fromEntries(new Intl.DateTimeFormat("en-US", {
        timeZone: zone, year: "numeric", month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit", hourCycle: "h23",
      }).formatToParts(now).map(({ type, value }) => [type, value]));
      const displayedMinute = Date.parse(`${parts.year}-${parts.month}-${parts.day}T${parts.hour}:${parts.minute}:00Z`);
      const systemMinute = Date.parse(`${localDate}T${localTime}:00Z`);
      if (Number.isFinite(systemMinute) && Number.isFinite(displayedMinute)) systemClockOffset += systemMinute - displayedMinute;
    }
    updateTopbarClock();
  });
}

if (controlCenter) {
  const brand = document.querySelector(".brand-link");
  if (brand) {
    if (window.location.pathname === "/") brand.setAttribute("aria-current", "page");
    else brand.removeAttribute("aria-current");
  }

  document.querySelectorAll(".control-tile").forEach((link) => {
    const target = new URL(link.href, window.location.origin);
    const samePath = target.pathname === window.location.pathname;
    const sameHash = !target.hash || target.hash === window.location.hash;
    if (samePath && sameHash) link.setAttribute("aria-current", "page");
  });

  document.addEventListener("click", (event) => {
    if (!controlCenter.open || event.target.closest("[data-control-center]")) return;
    controlCenter.open = false;
  });

  document.addEventListener("keydown", (event) => {
    if (event.key !== "Escape" || !controlCenter.open) return;
    controlCenter.open = false;
    const trigger = controlCenter.querySelector(".control-center-trigger");
    if (trigger) trigger.focus();
  });
}

const systemPower = document.querySelector("[data-system-power]");
const systemDialog = document.querySelector("[data-system-action-dialog]");
if (systemPower && systemDialog) {
  const csrfInput = document.querySelector("[data-system-actions-csrf]");
  const menuError = systemPower.querySelector("[data-system-action-error]");
  const firmwareButton = systemPower.querySelector('[data-system-action="firmware-reboot"]');
  const actionButtons = Array.from(systemPower.querySelectorAll("[data-system-action]"));
  const dialogTitle = systemDialog.querySelector("[data-system-action-title]");
  const dialogMessage = systemDialog.querySelector("[data-system-action-message]");
  const dialogPlayers = systemDialog.querySelector("[data-system-action-players]");
  const dialogUpdate = systemDialog.querySelector("[data-system-action-update]");
  const dialogError = systemDialog.querySelector("[data-system-action-dialog-error]");
  const cancelButton = systemDialog.querySelector("[data-system-action-cancel]");
  const confirmButton = systemDialog.querySelector("[data-system-action-confirm]");

  let selectedAction = "";
  let confirmPlayers = false;
  let latestStatus = null;

  const actionCopy = {
    reboot: {
      title: jvT("ui.8fc4170e4f2d"),
      message: jvT("ui.7237ce2bb0a8"),
      confirm: jvT("ui.fba023ca78eb"),
      working: jvT("ui.79235042a9f8"),
    },
    poweroff: {
      title: jvT("ui.3b667455abd0"),
      message: jvT("ui.cd21e88bf525"),
      confirm: jvT("ui.2089aa6cd044"),
      working: jvT("ui.f3245b2b3c7f"),
    },
    "firmware-reboot": {
      title: jvT("ui.0fe5d03a5b38"),
      message: jvT("ui.812c3ecff7ff"),
      confirm: jvT("ui.b098f557fbb4"),
      working: jvT("ui.8a5a61e2dfcd"),
    },
  };

  const loadSystemActionsStatus = async () => {
    if (menuError) menuError.hidden = true;
    if (firmwareButton) firmwareButton.hidden = true;
    try {
      const response = await fetch("/api/system-actions", {
        method: "GET",
        credentials: "same-origin",
        headers: { Accept: "application/json" },
        cache: "no-store",
      });
      if (response.status === 401) {
        window.location.assign("/login");
        return false;
      }
      if (response.status === 403) {
        if (controlCenter) controlCenter.open = false;
        return false;
      }
      if (!response.ok) throw new Error(jvT("ui.180119be630b"));
      const status = await response.json();
      latestStatus = status;
      const firmwareAvailable =
        latestStatus.variant === "hws" &&
        latestStatus.firmware &&
        latestStatus.firmware.available === true;
      if (firmwareButton) {
        firmwareButton.hidden = !firmwareAvailable;
        firmwareButton.disabled = !firmwareAvailable;
      }
      return true;
    } catch (_) {
      if (menuError) menuError.hidden = false;
      return false;
    }
  };

  const resetDialog = () => {
    confirmPlayers = false;
    if (dialogPlayers) {
      dialogPlayers.hidden = true;
      dialogPlayers.replaceChildren();
    }
    if (dialogError) {
      dialogError.hidden = true;
      dialogError.textContent = "";
    }
    if (dialogUpdate) dialogUpdate.hidden = true;
    if (cancelButton) cancelButton.hidden = false;
    if (confirmButton) confirmButton.hidden = false;
  };

  const openConfirmation = (action) => {
    const copy = actionCopy[action];
    if (!copy) return;
    selectedAction = action;
    resetDialog();
    if (dialogTitle) dialogTitle.textContent = copy.title;
    if (dialogMessage) dialogMessage.textContent = copy.message;
    if (confirmButton) {
      confirmButton.textContent = copy.confirm;
      confirmButton.disabled = false;
    }
    if (dialogUpdate) {
      dialogUpdate.hidden = !(latestStatus && latestStatus.staged_update && action !== "poweroff");
    }
    if (controlCenter) controlCenter.open = false;
    systemDialog.showModal();
  };

  const showPlayerConfirmation = (result) => {
    confirmPlayers = true;
    const copy = actionCopy[selectedAction];
    if (dialogTitle) dialogTitle.textContent = jvT("ui.9c9a3156ead1");
    if (dialogMessage) {
      const count = Number(result.online || 0);
      const noun = count === 1 ? "player is" : "players are";
      dialogMessage.textContent =
        String(count) + " " + noun + jvT("ui.4cb4a59b1780") + copy.confirm.toLowerCase() + "?";
    }
    if (dialogPlayers) {
      dialogPlayers.replaceChildren();
      (Array.isArray(result.players) ? result.players : []).forEach((name) => {
        const chip = document.createElement("span");
        chip.className = "player-chip";
        chip.textContent = name;
        dialogPlayers.appendChild(chip);
      });
      dialogPlayers.hidden = dialogPlayers.childElementCount === 0;
    }
    if (confirmButton) {
      confirmButton.textContent = selectedAction === "poweroff" ? jvT("ui.903044546444") : jvT("ui.27015f91e79c");
      confirmButton.disabled = false;
    }
  };

  const showActionError = (message) => {
    if (dialogError) {
      dialogError.textContent = message || jvT("ui.c68490537f5e");
      dialogError.hidden = false;
    }
    if (cancelButton) cancelButton.hidden = false;
    if (confirmButton) confirmButton.disabled = false;
  };

  const submitSystemAction = async () => {
    const copy = actionCopy[selectedAction];
    if (!copy || !csrfInput) return;
    if (confirmButton) {
      confirmButton.disabled = true;
      confirmButton.textContent = jvT("ui.5474eef8d0f1");
    }
    if (dialogError) dialogError.hidden = true;

    const body = new URLSearchParams();
    body.set("csrf", csrfInput.value);
    if (confirmPlayers) body.set("confirm_players", "yes");

    try {
      const response = await fetch("/api/system-actions/" + selectedAction, {
        method: "POST",
        credentials: "same-origin",
        headers: {
          Accept: "application/json",
          "Content-Type": "application/x-www-form-urlencoded",
        },
        body: body.toString(),
      });
      if (response.status === 401) {
        window.location.assign("/login");
        return;
      }
      if (response.status === 403) {
        window.location.assign("/password");
        return;
      }

      let result = {};
      try {
        result = await response.json();
      } catch (_) {
        // A structured error is preferred, but the UI still fails safely.
      }

      if (response.ok && result.confirmation_required && result.reason === "players_online") {
        showPlayerConfirmation(result);
        return;
      }
      if (response.status === 202 && result.accepted) {
        if (dialogTitle) dialogTitle.textContent = copy.working;
        if (dialogMessage) dialogMessage.textContent = result.message || jvT("ui.279acc4bb281");
        if (dialogPlayers) dialogPlayers.hidden = true;
        if (dialogUpdate) dialogUpdate.hidden = true;
        if (cancelButton) cancelButton.hidden = true;
        if (confirmButton) confirmButton.hidden = true;
        return;
      }

      showActionError(result.message || result.reason || jvT("ui.c68490537f5e"));
    } catch (_) {
      showActionError(jvT("ui.705c81356a5c"));
    }
  };

  loadSystemActionsStatus();

  actionButtons.forEach((button) => {
    button.addEventListener("click", () => openConfirmation(button.dataset.systemAction));
  });

  if (cancelButton) {
    cancelButton.addEventListener("click", () => systemDialog.close());
  }
  if (confirmButton) {
    confirmButton.addEventListener("click", submitSystemAction);
  }

  document.addEventListener("click", (event) => {
    if (!systemPower.open || event.target.closest("[data-system-power]")) return;
    if (controlCenter) controlCenter.open = false;
  });
}


const initSystemUpdateWorkspace = () => {
  const systemUpdateOpen = document.querySelector("[data-system-update-open]");
  const systemUpdateDialog = document.querySelector("[data-system-update-dialog]");
  if (!systemUpdateOpen || !systemUpdateDialog) return;

  const csrfInput = document.querySelector("[data-system-update-csrf]");
  const closeButton = systemUpdateDialog.querySelector("[data-system-update-close]");
  const refreshButton = systemUpdateDialog.querySelector("[data-system-update-refresh]");
  const runningValue = systemUpdateDialog.querySelector("[data-system-update-running]");
  const runningImage = systemUpdateDialog.querySelector("[data-system-update-running-image]");
  const stagedValue = systemUpdateDialog.querySelector("[data-system-update-staged]");
  const stagedImage = systemUpdateDialog.querySelector("[data-system-update-staged-image]");
  const availableValue = systemUpdateDialog.querySelector("[data-system-update-available]");
  const availableImage = systemUpdateDialog.querySelector("[data-system-update-available-image]");
  const state = systemUpdateDialog.querySelector("[data-system-update-state]");
  const error = systemUpdateDialog.querySelector("[data-system-update-error]");
  const updateButton = systemUpdateDialog.querySelector("[data-system-update-button]");
  const forceButton = systemUpdateDialog.querySelector("[data-system-update-force]");
  const rebootPanel = systemUpdateDialog.querySelector("[data-system-update-reboot]");
  const rebootTitle = systemUpdateDialog.querySelector("[data-system-update-reboot-title]");
  const backupToggle = systemUpdateDialog.querySelector("[data-system-update-backup]");
  const quickToggle = systemUpdateDialog.querySelector("[data-system-update-quick]");
  const warningText = systemUpdateDialog.querySelector("[data-system-update-warning]");
  const rebootPlayers = systemUpdateDialog.querySelector("[data-system-update-reboot-players]");
  const rebootProgress = systemUpdateDialog.querySelector("[data-system-update-reboot-progress]");
  const rebootCountdown = systemUpdateDialog.querySelector("[data-system-update-countdown]");
  const rebootMessage = systemUpdateDialog.querySelector("[data-system-update-reboot-message]");
  const rebootButton = systemUpdateDialog.querySelector("[data-system-update-reboot-button]");

  let updating = false;
  let checking = false;
  let latestStatus = null;
  let pendingPlayerConfirmation = false;
  let rebootWorkflowActive = false;
  let rebootPollTimer = null;
  let countdownTimer = null;
  let checkSequence = 0;
  let minecraftConfigured = dashboard?.dataset.configured === "true";

  if (dashboard) {
    new MutationObserver(() => {
      minecraftConfigured = dashboard.dataset.configured === "true";
      syncActionAvailability();
    }).observe(dashboard, { attributes: true, attributeFilter: ["data-configured"] });
  }

  const refreshMinecraftAvailability = async () => {
    if (dashboard) return;
    minecraftConfigured = false;
    syncActionAvailability();
    try {
      const response = await fetch(globalThis.justVoxelSelectedURL("/api/dashboard-status", "minecraft"), {
        method: "GET",
        credentials: "same-origin",
        headers: { Accept: "application/json" },
        cache: "no-store",
      });
      if (handleAuthResponse(response) || !response.ok) return;
      const snapshot = await response.json();
      minecraftConfigured = snapshot?.status?.minecraft?.configured === true;
      syncActionAvailability();
    } catch (_) {
      // Keep backup unavailable until Minecraft configuration can be confirmed.
    }
  };

  const deploymentVersion = (deployment, fallback) => {
    if (!deployment) return fallback;
    return deployment.version || fallback || jvT("ui.870a8ffd98f4");
  };

  const deploymentDetail = (deployment) => {
    if (!deployment) return "";
    const parts = [];
    if (deployment.image) parts.push(deployment.image);
    if (deployment.digest) {
      const digest = String(deployment.digest);
      parts.push(digest.length > 24 ? digest.slice(0, 19) + "…" : digest);
    }
    return parts.join(" · ");
  };

  const clearRebootTimers = () => {
    if (rebootPollTimer) {
      window.clearTimeout(rebootPollTimer);
      rebootPollTimer = null;
    }
    if (countdownTimer) {
      window.clearInterval(countdownTimer);
      countdownTimer = null;
    }
  };

  const updateWarningText = () => {
    if (!warningText) return;
    warningText.textContent = quickToggle && quickToggle.checked
      ? jvT("ui.f3ab88d5b2cc")
      : jvT("ui.b2d3efa1f961");
  };

  const freshUpdateAvailable = () => latestStatus?.check_state === "update_available";

  const baseRebootLabel = () => !latestStatus?.activation_failed && freshUpdateAvailable() ? jvT("ui.fea3ff98725c") : jvT("ui.71fb283ae756");

  const showRebootPlayers = (result) => {
    if (!rebootPlayers) return;
    const players = Array.isArray(result.players) ? result.players.filter(Boolean) : [];
    const count = Number(result.online || players.length || 0);
    const names = players.length ? " — " + players.join(", ") : "";
    rebootPlayers.textContent = count + " player" + (count === 1 ? "" : "s") + " online" + names + ". Press again to confirm.";
    rebootPlayers.hidden = false;
  };

  const hideRebootPlayers = () => {
    if (!rebootPlayers) return;
    rebootPlayers.hidden = true;
    rebootPlayers.textContent = "";
  };

  const syncActionAvailability = () => {
    const readOnly = Boolean(latestStatus?.read_only);
    const checkState = latestStatus?.check_state || "";
    const canStage = !readOnly && !checking && !updating && !rebootWorkflowActive &&
      (latestStatus?.activation_failed || checkState === "update_available" || checkState === "unknown");
    if (forceButton) {
      forceButton.hidden = latestStatus?.activation_failed !== true || latestStatus?.activation_recoverable !== true;
      forceButton.disabled = readOnly || checking || updating || rebootWorkflowActive;
    }
    if (updateButton) updateButton.disabled = !canStage;
    if (refreshButton) refreshButton.disabled = checking || updating || rebootWorkflowActive;
    if (backupToggle) {
      if (!minecraftConfigured) backupToggle.checked = false;
      backupToggle.disabled = !minecraftConfigured || checking || updating || rebootWorkflowActive;
    }
    if (quickToggle) quickToggle.disabled = checking || updating || rebootWorkflowActive;
    if (rebootButton) {
      rebootButton.disabled = checking || updating || rebootWorkflowActive ||
        (!freshUpdateAvailable() && !Boolean(latestStatus?.reboot_required));
    }
  };

  const resetPlayerConfirmation = () => {
    if (rebootWorkflowActive) return;
    pendingPlayerConfirmation = false;
    hideRebootPlayers();
    if (rebootButton) rebootButton.textContent = baseRebootLabel();
  };

  const startVisibleCountdown = (deadlineUnix) => {
    if (!rebootCountdown) return;
    if (countdownTimer) window.clearInterval(countdownTimer);
    const tick = () => {
      const remaining = Math.max(0, Number(deadlineUnix || 0) - Math.floor(Date.now() / 1000));
      rebootCountdown.textContent = String(remaining);
      if (remaining <= 0 && countdownTimer) {
        window.clearInterval(countdownTimer);
        countdownTimer = null;
      }
    };
    tick();
    countdownTimer = window.setInterval(tick, 250);
  };

  const renderRebootStatus = (result) => {
    if (!result || !result.state) return;

    if (result.confirmation_required) {
      rebootWorkflowActive = false;
      pendingPlayerConfirmation = true;
      showRebootPlayers(result);
      if (rebootProgress) rebootProgress.hidden = true;
      syncActionAvailability();
      if (rebootButton) {
        rebootButton.disabled = false;
        rebootButton.textContent = jvT("ui.eef42d5c5b7e");
      }
      return;
    }

    const activeStates = new Set(["queued", "countdown", "stopping", "backup", "rebooting"]);
    rebootWorkflowActive = activeStates.has(result.state);

    if (result.state === "idle") {
      pendingPlayerConfirmation = false;
      if (rebootProgress) rebootProgress.hidden = true;
      hideRebootPlayers();
      if (rebootButton) rebootButton.textContent = baseRebootLabel();
      syncActionAvailability();
      return;
    }

    if (result.state === "backup_failed" || result.state === "failed") {
      rebootWorkflowActive = false;
      pendingPlayerConfirmation = false;
      clearRebootTimers();
      hideRebootPlayers();
      if (rebootProgress) rebootProgress.hidden = true;
      if (rebootButton) rebootButton.textContent = baseRebootLabel();
      if (error) {
        error.textContent = result.message || jvT("ui.f57e85681cfe");
        error.hidden = false;
      }
      syncActionAvailability();
      return;
    }

    if (rebootWorkflowActive) {
      pendingPlayerConfirmation = false;
      hideRebootPlayers();
      syncActionAvailability();
      if (rebootProgress) rebootProgress.hidden = false;
      if (rebootButton) rebootButton.textContent = jvT("ui.568681c5c2b8");

      if (result.state === "countdown") {
        if (rebootMessage) {
          const warning = Number(result.warning_seconds || 60);
          rebootMessage.textContent = jvT("ui.1ee0168d311b") + warning + " seconds).";
        }
        startVisibleCountdown(result.deadline_unix);
      } else {
        if (countdownTimer) {
          window.clearInterval(countdownTimer);
          countdownTimer = null;
        }
        if (rebootCountdown) rebootCountdown.textContent = "";
        if (rebootMessage) {
          if (result.state === "queued") rebootMessage.textContent = jvT("ui.5bc83aba3458");
          if (result.state === "stopping") rebootMessage.textContent = jvT("ui.00ec00d5b651");
          if (result.state === "backup") rebootMessage.textContent = jvT("ui.f47aad1860f5");
          if (result.state === "rebooting") rebootMessage.textContent = jvT("ui.79235042a9f8");
        }
      }
    }
  };

  const renderSystemUpdate = (status) => {
    latestStatus = status;
    if (runningValue) runningValue.textContent = deploymentVersion(status.running, jvT("ui.e4ffce61e120"));
    if (runningImage) runningImage.textContent = deploymentDetail(status.running);
    if (stagedValue) stagedValue.textContent = status.staged ? deploymentVersion(status.staged, jvT("ui.83c4e2f49cc0")) : jvT("ui.dc937b598926");
    if (stagedImage) stagedImage.textContent = deploymentDetail(status.staged);

    if (availableValue) {
      if (!status.checked) {
        availableValue.textContent = checking ? jvT("ui.ec963ffc911b") : jvT("ui.d16948e73a68");
      } else if (status.check_state === "update_available") {
        availableValue.textContent = deploymentVersion(status.available, jvT("ui.d87349fb3b49"));
      } else if (status.check_state === "staged_current") {
        availableValue.textContent = jvT("ui.0e297fffef8a");
      } else if (status.check_state === "current") {
        availableValue.textContent = jvT("ui.e67082c9b94f");
      } else {
        availableValue.textContent = jvT("ui.b764cdc0eab7");
      }
    }
    if (availableImage) {
      availableImage.textContent = status.checked && status.check_state === "update_available"
        ? deploymentDetail(status.available)
        : "";
    }

    if (state) {
      if (status.activation_failed) {
        state.textContent = status.recovery_message || jvT("ui.38176da8c559");
      } else if (status.read_only) {
        state.textContent = jvT("ui.5b7ffa3abf12");
      } else if (status.checked) {
        state.textContent = status.message || jvT("ui.4217335b2719");
      } else if (status.reboot_required) {
        state.textContent = jvT("ui.05c1ec7b6cd1");
      } else {
        state.textContent = checking ? jvT("ui.a8d707f462eb") : jvT("ui.1c97a3febb96");
      }
    }

    const showReboot = Boolean(status.reboot_required) || status.check_state === "update_available";
    if (rebootPanel) rebootPanel.hidden = !showReboot;
    if (rebootTitle) rebootTitle.textContent = freshUpdateAvailable() ? jvT("ui.585fb06e9520") : jvT("ui.f5fd3fb1a07f");
    if (!status.reboot_required && !freshUpdateAvailable()) {
      rebootWorkflowActive = false;
      pendingPlayerConfirmation = false;
      clearRebootTimers();
      hideRebootPlayers();
      if (rebootProgress) rebootProgress.hidden = true;
    }
    if (rebootButton && !pendingPlayerConfirmation && !rebootWorkflowActive) {
      rebootButton.textContent = baseRebootLabel();
    }

    if (error) {
      error.hidden = true;
      error.textContent = "";
    }
    syncActionAvailability();
  };

  const showSystemUpdateError = (message, preserveState = false) => {
    if (error) {
      error.textContent = message || jvT("ui.3eb4a4922fd4");
      error.hidden = false;
    }
    if (state && !preserveState) state.textContent = jvT("ui.8aea8af4c3a7");
    syncActionAvailability();
  };

  const readResponse = async (response) => {
    try {
      return await response.json();
    } catch (_) {
      return {};
    }
  };

  const handleAuthResponse = (response) => {
    if (response.status === 401) {
      window.location.assign("/login");
      return true;
    }
    if (response.status === 403) {
      window.location.assign("/password");
      return true;
    }
    return false;
  };

  const pollRebootStatus = async () => {
    if (!latestStatus || !latestStatus.reboot_required || !systemUpdateDialog.open) return;
    try {
      const response = await fetch("/api/system-updates/reboot", {
        method: "GET",
        credentials: "same-origin",
        headers: { Accept: "application/json" },
        cache: "no-store",
      });
      if (handleAuthResponse(response)) return;
      const result = await readResponse(response);
      if (response.ok) {
        renderRebootStatus(result);
        if (rebootWorkflowActive) {
          rebootPollTimer = window.setTimeout(pollRebootStatus, 1000);
        }
      }
    } catch (_) {
      if (rebootWorkflowActive) {
        rebootPollTimer = window.setTimeout(pollRebootStatus, 1000);
      }
    }
  };

  const loadLocalSystemUpdateStatus = async () => {
    const response = await fetch("/api/system-updates", {
      method: "GET",
      credentials: "same-origin",
      headers: { Accept: "application/json" },
      cache: "no-store",
    });
    if (handleAuthResponse(response)) return null;
    const result = await readResponse(response);
    if (!response.ok) throw new Error(result.error || jvT("ui.c87dd5a532c0"));
    renderSystemUpdate(result);
    if (result.reboot_required) await pollRebootStatus();
    return result;
  };

  const checkSystemUpdate = async () => {
    const sequence = ++checkSequence;
    checking = true;
    if (error) error.hidden = true;
    if (availableValue) availableValue.textContent = jvT("ui.ec963ffc911b");
    if (availableImage) availableImage.textContent = "";
    if (state) state.textContent = jvT("ui.a8d707f462eb");
    syncActionAvailability();

    try {
      const local = await loadLocalSystemUpdateStatus();
      if (!local || sequence !== checkSequence) return;
      if (local.read_only) {
        if (availableValue) availableValue.textContent = jvT("ui.ca1844969742");
        return;
      }
      if (!csrfInput) throw new Error(jvT("ui.a6257738e943"));

      if (state) {
        state.textContent = local.activation_failed ? local.recovery_message : local.reboot_required
          ? jvT("ui.05c1ec7b6cd1")
          : jvT("ui.a8d707f462eb");
      }
      if (availableValue) availableValue.textContent = jvT("ui.ec963ffc911b");

      const body = new URLSearchParams({ csrf: csrfInput.value });
      const response = await fetch("/api/system-updates/check", {
        method: "POST",
        credentials: "same-origin",
        headers: {
          Accept: "application/json",
          "Content-Type": "application/x-www-form-urlencoded",
        },
        body: body.toString(),
        cache: "no-store",
      });
      if (handleAuthResponse(response)) return;
      const result = await readResponse(response);
      if (!response.ok) throw new Error(result.error || jvT("ui.fded81ded2ed"));
      if (sequence !== checkSequence) return;
      renderSystemUpdate(result);
      if (result.reboot_required) await pollRebootStatus();
    } catch (caught) {
      if (sequence !== checkSequence) return;
      if (availableValue) availableValue.textContent = jvT("ui.c8813c1128d5");
      if (availableImage) availableImage.textContent = "";
      if (state) {
        state.textContent = latestStatus?.activation_failed ? latestStatus.recovery_message : latestStatus?.reboot_required
          ? jvT("ui.91f978bbee4c")
          : jvT("ui.6809eed29996");
      }
      showSystemUpdateError(caught?.message || jvT("ui.62083bd54303"), true);
    } finally {
      if (sequence === checkSequence) {
        checking = false;
        syncActionAvailability();
      }
    }
  };

  const requestUpdateReboot = async () => {
    if (freshUpdateAvailable() && !latestStatus?.reboot_required && !updating) {
      await applySystemUpdate(true);
      return;
    }
    if (rebootWorkflowActive || updating || checking || !csrfInput || !rebootButton) return;

    rebootButton.disabled = true;
    rebootButton.textContent = jvT("ui.ec963ffc911b");
    if (error) error.hidden = true;

    const body = new URLSearchParams();
    body.set("csrf", csrfInput.value);
    if (latestStatus?.staged?.digest) body.set("staged_digest", latestStatus.staged.digest);
    if (backupToggle && backupToggle.checked) body.set("backup_minecraft", "yes");
    if (quickToggle && quickToggle.checked) body.set("quick_reboot", "yes");
    if (pendingPlayerConfirmation) body.set("confirm_players", "yes");

    try {
      const response = await fetch("/api/system-updates/reboot", {
        method: "POST",
        credentials: "same-origin",
        headers: {
          Accept: "application/json",
          "Content-Type": "application/x-www-form-urlencoded",
        },
        body: body.toString(),
      });
      if (handleAuthResponse(response)) return;
      const result = await readResponse(response);
      if (result.confirmation_required) {
        renderRebootStatus(result);
        return;
      }
      if (!response.ok) {
        renderRebootStatus(result);
        if (error && error.hidden) {
          error.textContent = result.message || result.error || jvT("ui.c6dc8963b640");
          error.hidden = false;
        }
        return;
      }
      renderRebootStatus(result);
      if (result.accepted) rebootPollTimer = window.setTimeout(pollRebootStatus, 250);
    } catch (_) {
      rebootWorkflowActive = false;
      if (rebootButton) rebootButton.textContent = baseRebootLabel();
      if (error) {
        error.textContent = jvT("ui.c6dc8963b640");
        error.hidden = false;
      }
      syncActionAvailability();
    }
  };

  const applySystemUpdate = async (rebootAfter = false) => {
    if (updating || checking || rebootWorkflowActive || !csrfInput || !updateButton) return;
    updating = true;
    updateButton.disabled = true;
    updateButton.classList.add("is-busy");
    updateButton.setAttribute("aria-busy", "true");
    updateButton.textContent = rebootAfter ? jvT("ui.a778d3492af2") : jvT("ui.a778d3492af2");
    if (rebootButton) rebootButton.disabled = true;
    if (state) state.textContent = jvT("ui.a732f56a855b");
    if (error) error.hidden = true;
    syncActionAvailability();

    const body = new URLSearchParams({ csrf: csrfInput.value });
    try {
      const response = await fetch("/api/system-updates", {
        method: "POST",
        credentials: "same-origin",
        headers: {
          Accept: "application/json",
          "Content-Type": "application/x-www-form-urlencoded",
        },
        body: body.toString(),
      });
      if (handleAuthResponse(response)) return;
      const result = await readResponse(response);
      if (!response.ok) {
        showSystemUpdateError(result.error || jvT("ui.ad183bccc3c3"));
        return;
      }
      renderSystemUpdate(result);
      if (result.reboot_required) {
        await pollRebootStatus();
        if (rebootAfter) {
          updating = false;
          syncActionAvailability();
          await requestUpdateReboot();
        }
      }
    } catch (_) {
      showSystemUpdateError(jvT("ui.31f827ccdce3"));
    } finally {
      updating = false;
      updateButton.classList.remove("is-busy");
      updateButton.removeAttribute("aria-busy");
      updateButton.textContent = jvT("ui.2f0480a1438d");
      syncActionAvailability();
    }
  };

  const forceSystemUpdate = async () => {
    if (updating || checking || rebootWorkflowActive || !csrfInput || latestStatus?.activation_recoverable !== true) return;
    if (quickToggle) quickToggle.checked = true;
    updateWarningText();
    resetPlayerConfirmation();
    updating = true;
    syncActionAvailability();
    if (state) state.textContent = jvT("ui.273af1640c6a");
    if (error) error.hidden = true;
    try {
      const response = await fetch("/api/system-updates/force", {
        method: "POST", credentials: "same-origin",
        headers: { Accept: "application/json", "Content-Type": "application/x-www-form-urlencoded" },
        body: new URLSearchParams({ csrf: csrfInput.value }).toString(),
      });
      if (handleAuthResponse(response)) return;
      const result = await readResponse(response);
      if (!response.ok) {
        // The failed command may have changed deployments. Refresh authority
        // before deciding whether recovery can still be offered.
        await loadLocalSystemUpdateStatus();
        showSystemUpdateError(result.error || jvT("ui.d42b64c5b7ff"), true);
        return;
      }
      renderSystemUpdate(result);
    } catch (_) {
      latestStatus = null;
      showSystemUpdateError(jvT("ui.5a7bebeab16a"));
    } finally {
      updating = false;
      syncActionAvailability();
    }
  };

  const workspaceWindow = setupWorkspaceWindow(systemUpdateDialog, {
    onOpen: () => {
      void refreshMinecraftAvailability();
      updateWarningText();
      checkSystemUpdate();
    },
    onClose: () => {
      checkSequence += 1;
      checking = false;
      clearRebootTimers();
    },
  });

  systemUpdateOpen.addEventListener("click", () => {
    if (controlCenter) controlCenter.open = false;
    workspaceWindow?.open();
  });

  closeButton?.addEventListener("click", () => workspaceWindow?.close());
  forceButton?.addEventListener("click", forceSystemUpdate);
  refreshButton?.addEventListener("click", checkSystemUpdate);
  updateButton?.addEventListener("click", () => applySystemUpdate(false));
  rebootButton?.addEventListener("click", requestUpdateReboot);
  quickToggle?.addEventListener("change", () => {
    updateWarningText();
    resetPlayerConfirmation();
  });
  backupToggle?.addEventListener("change", resetPlayerConfirmation);

  const restoreSystemUpdateWhenRoleKnown = () => {
    if (!readWorkspaceWindowState("system-update").open) return;
    if (document.body.classList.contains("role-administrator")) {
      workspaceWindow?.open();
      return;
    }
    if (!document.body.classList.contains("role-pending")) return;
    const observer = new MutationObserver(() => {
      if (document.body.classList.contains("role-pending")) return;
      observer.disconnect();
      if (document.body.classList.contains("role-administrator")) workspaceWindow?.open();
    });
    observer.observe(document.body, { attributes: true, attributeFilter: ["class"] });
  };
  restoreSystemUpdateWhenRoleKnown();
};

const workspaceWindows = new Set();
let workspaceWindowZ = 120;
// Keep workspace windows below the Control Center's reserved topbar layer (1000).
const workspaceWindowZMax = 900;
const workspaceCompactQuery = window.matchMedia("(max-width: 700px)");

const workspaceTopInset = () => {
  const anchor = document.querySelector(".app-header.topbar");
  const rootStyle = getComputedStyle(document.documentElement);
  const gap = parseFloat(rootStyle.getPropertyValue("--jv-panel-gap")) || 4;
  return Math.max(8, Math.round((anchor?.getBoundingClientRect().bottom || 0) + gap));
};

const clampWorkspaceWindow = (element) => {
  if (!element.open || workspaceCompactQuery.matches) return;
  const minTop = workspaceTopInset();
  const width = Math.max(1, document.documentElement.clientWidth - 16);
  const height = Math.max(1, window.innerHeight - minTop - 8);
  // First fit the whole window, including saved dimensions from a larger screen.
  element.style.setProperty("--workspace-available-width", width + "px");
  element.style.setProperty("--workspace-available-height", height + "px");
  const rect = element.getBoundingClientRect();
  const left = Math.min(Math.max(rect.left, 8), Math.max(8, document.documentElement.clientWidth - rect.width - 8));
  const top = Math.min(Math.max(rect.top, minTop), Math.max(minTop, window.innerHeight - rect.height - 8));
  element.style.left = Math.round(left) + "px";
  element.style.top = Math.round(top) + "px";
  // Native resizing must stop at the remaining space, before it can overflow.
  element.style.setProperty("--workspace-available-width", Math.max(1, document.documentElement.clientWidth - Math.round(left) - 8) + "px");
  element.style.setProperty("--workspace-available-height", Math.max(1, window.innerHeight - Math.round(top) - 8) + "px");
};

const setupWorkspaceWindow = (element, options = {}) => {
  const id = element.dataset.workspaceWindow;
  if (!id) return null;

  workspaceWindows.add(element);
  const dragHandle = element.querySelector("[data-workspace-drag-handle]");
  let resizeSaveTimer = null;

  const bringToFront = () => {
    if (workspaceWindowZ >= workspaceWindowZMax) {
      const ordered = Array.from(workspaceWindows).filter((window) => window.open)
        .sort((a, b) => Number(a.style.zIndex || 0) - Number(b.style.zIndex || 0));
      workspaceWindowZ = 120;
      ordered.forEach((window) => { window.style.zIndex = String(++workspaceWindowZ); });
    }
    workspaceWindowZ += 1;
    element.style.zIndex = String(workspaceWindowZ);
  };

  const persistGeometry = () => {
    if (!element.open || workspaceCompactQuery.matches) return;
    const rect = element.getBoundingClientRect();
    writeWorkspaceWindowState(id, {
      left: Math.round(rect.left),
      top: Math.round(rect.top),
      width: Math.round(rect.width),
      height: Math.round(rect.height),
    });
  };

  const applySavedGeometry = () => {
    if (workspaceCompactQuery.matches) return;
    const saved = readWorkspaceWindowState(id);
    if (Number.isFinite(saved.width) && saved.width > 0) element.style.width = saved.width + "px";
    if (Number.isFinite(saved.height) && saved.height > 0) element.style.height = saved.height + "px";
    if (Number.isFinite(saved.left)) element.style.left = saved.left + "px";
    if (Number.isFinite(saved.top)) element.style.top = saved.top + "px";

    // Bound restored geometry synchronously; an off-screen saved window must
    // never contribute document overflow for an animation frame.
    clampWorkspaceWindow(element);
    if (!Number.isFinite(saved.left) || !Number.isFinite(saved.top)) {
      const rect = element.getBoundingClientRect();
      element.style.left = Math.max(8, Math.round((document.documentElement.clientWidth - rect.width) / 2)) + "px";
      element.style.top = workspaceTopInset() + "px";
    }
    clampWorkspaceWindow(element);
    persistGeometry();
  };

  const open = () => {
    if (!element.open) element.show();
    bringToFront();
    applySavedGeometry();
    writeWorkspaceWindowState(id, { open: true });
    options.onOpen?.();
  };

  const close = () => {
    if (element.open) element.close();
  };

  element.addEventListener("pointerdown", bringToFront);

  if (dragHandle) {
    dragHandle.addEventListener("pointerdown", (event) => {
      if (workspaceCompactQuery.matches) return;
      if (event.button !== 0) return;
      if (event.target.closest("button,a,input,select,label,.server-selector")) return;

      bringToFront();
      const rect = element.getBoundingClientRect();
      const startX = event.clientX;
      const startY = event.clientY;
      const startLeft = rect.left;
      const startTop = rect.top;
      const minTop = workspaceTopInset();

      event.preventDefault();
      dragHandle.setPointerCapture(event.pointerId);

      const move = (moveEvent) => {
        const left = Math.min(
          Math.max(startLeft + moveEvent.clientX - startX, 8),
          Math.max(8, document.documentElement.clientWidth - rect.width - 8),
        );
        const top = Math.min(
          Math.max(startTop + moveEvent.clientY - startY, minTop),
          Math.max(minTop, window.innerHeight - rect.height - 8),
        );
        element.style.left = Math.round(left) + "px";
        element.style.top = Math.round(top) + "px";
        clampWorkspaceWindow(element);
      };

      const finish = () => {
        dragHandle.removeEventListener("pointermove", move);
        dragHandle.removeEventListener("pointerup", finish);
        dragHandle.removeEventListener("pointercancel", finish);
        persistGeometry();
      };

      dragHandle.addEventListener("pointermove", move);
      dragHandle.addEventListener("pointerup", finish);
      dragHandle.addEventListener("pointercancel", finish);
    });
  }

  const resizeObserver = new ResizeObserver(() => {
    if (!element.open || workspaceCompactQuery.matches) return;
    clampWorkspaceWindow(element);
    window.clearTimeout(resizeSaveTimer);
    resizeSaveTimer = window.setTimeout(() => {
      persistGeometry();
    }, 120);
  });
  resizeObserver.observe(element);

  element.addEventListener("close", () => {
    writeWorkspaceWindowState(id, { open: false });
    options.onClose?.();
  });

  return { open, close, bringToFront, id };
};

// One explicit-open diagnostic viewer shared by setup and persistent resets.
window.JustVoxelLiveConsole = (() => {
  let dialog, viewer, label, indicator, workspace;
  let selected = null;
  let generation = 0;
  let timer = null;
  let controller = null;
  let reconnecting = false;
  const labels = { setup: jvT("ui.a2d1b5fba925"), minecraft_reset: jvT("ui.979e2ada0bd5"), factory_reset: jvT("ui.20b6891d2ddb") };
  const stageLabels = {
    storage_preflight: jvT("ui.a69c4dece144"), storage_snapshot: jvT("ui.a69c4dece144"), storage_verified: jvT("ui.a69c4dece144"),
    runtime_preflight: jvT("ui.109311589787"), runtime_config: jvT("ui.109311589787"), minecraft_verify: "Minecraft",
    final_validation: jvT("ui.7140f4f19dec"), verifying: jvT("ui.7140f4f19dec"), validating: jvT("ui.68e1ca575295"),
    runtime_rollback: jvT("ui.48f6a8d5688b"), storage_rollback: jvT("ui.48f6a8d5688b"), runtime_recovery: jvT("ui.48f6a8d5688b"), storage_recovery: jvT("ui.48f6a8d5688b"),
    resetting: jvT("ui.a16f0e9fe92d"), resetting_runtime: jvT("ui.35f218878fbc"), resetting_identity: jvT("ui.a647374682e9"),
    safety_backup: jvT("ui.31d98a860687"),
  };
  const stateLabel = (state) => ({
    succeeded: jvT("ui.143b270a3260"), resolved: jvT("ui.143b270a3260"), rolled_back: jvT("ui.71592236d852"),
    needs_attention: jvT("ui.c1ebc7817870"), running: jvT("ui.f4ccae29e1bb"), verifying: jvT("ui.a7d4a9f301ed"),
    queued: jvT("ui.661ff40a07e0"), validating: jvT("ui.5a1a167e3d90"), rolling_back: jvT("ui.8b9e964b606b"),
  }[state] || jvT("ui.b64ac05f17e6"));
  const showState = () => {
    const stage = Object.prototype.hasOwnProperty.call(stageLabels, selected.stage) ? stageLabels[selected.stage] : "";
    label.textContent = labels[selected.kind] + (stage ? ` · ${stage}` : "");
    indicator.textContent = reconnecting ? jvT("ui.27b80374e115") : stateLabel(selected.state);
  };
  const stop = () => {
    generation += 1;
    window.clearTimeout(timer);
    timer = null;
    controller?.abort();
    controller = null;
  };
  const close = () => {
    stop();
    workspace?.close();
  };
  const ensureWindow = () => {
    if (dialog) return;
    const make = (tag, className, text) => {
      const node = document.createElement(tag);
      node.className = className;
      if (text !== undefined) node.textContent = text;
      return node;
    };
    dialog = make("dialog", "workspace-window live-console-dialog");
    dialog.dataset.workspaceWindow = "live-console";
    dialog.setAttribute("aria-labelledby", "live-console-title");
    const shell = make("div", "live-console-shell");
    const header = make("header", "system-workspace-header live-console-header");
    header.setAttribute("data-workspace-drag-handle", "");
    const heading = make("div", "");
    const title = make("h2", "", jvT("ui.7cac54a8cfc3"));
    title.id = "live-console-title";
    heading.append(make("p", "eyebrow", "JustVoxel"), title);
    const dismiss = make("button", "secondary system-monitor-header-button workspace-header-close");
    const svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
    svg.setAttribute("viewBox", "0 0 24 24");
    svg.setAttribute("aria-hidden", "true");
    svg.setAttribute("fill", "none");
    svg.setAttribute("stroke", "currentColor");
    svg.setAttribute("stroke-width", "2");
    svg.setAttribute("stroke-linecap", "round");
    const path = document.createElementNS("http://www.w3.org/2000/svg", "path");
    path.setAttribute("d", "M5 5l14 14M19 5L5 19");
    svg.append(path);
    dismiss.append(svg);
    dismiss.type = "button";
    dismiss.setAttribute("aria-label", jvT("ui.001c794ce986"));
    dismiss.addEventListener("click", close);
    header.append(heading, dismiss);
    const context = make("div", "live-console-context");
    label = make("span", "");
    indicator = make("span", "badge");
    indicator.setAttribute("role", "status");
    context.append(label, indicator);
    viewer = make("pre", "live-console-transcript", "");
    viewer.tabIndex = 0;
    viewer.setAttribute("aria-label", jvT("ui.d690a8f6ded8"));
    shell.append(header, context, viewer);
    dialog.append(shell);
    document.body.append(dialog);
    workspace = setupWorkspaceWindow(dialog, { onClose: () => { if (!dialog.open) stop(); } });
    // Never restore open state: each page begins with the console closed.
  };
  const read = async (token) => {
    const current = () => token === generation && dialog.open;
    if (!current()) return;
    const source = selected;
    controller = new AbortController();
    try {
      const response = await fetch(source.url, {
        credentials: "same-origin", cache: "no-store",
        headers: { Accept: "text/plain" }, signal: controller.signal,
      });
      if (!current()) return;
      if (response.status === 401) {
        close();
        if (source.onUnauthorized) source.onUnauthorized();
        else window.location.assign("/login");
        return;
      }
      if (response.redirected) {
        const target = new URL(response.url, window.location.href);
        if (target.origin === window.location.origin && ["/login", "/password"].includes(target.pathname)) {
          close();
          window.location.assign(target.pathname + target.search);
          return;
        }
      }
      if (response.status === 403) {
        if (source.kind === "setup") {
          close();
          window.location.assign("/password");
          return;
        }
        const payload = await response.json().catch(() => ({}));
        if (!current()) return;
        if (String(jvC(payload.error || "")).toLowerCase().includes("password change")) {
          close();
          window.location.assign("/password");
          return;
        }
      }
      if (!response.ok || !/^text\/plain(?:;|$)/i.test(response.headers.get("Content-Type") || "")) {
        throw new Error(jvT("ui.1fb1cbf2b811"));
      }
      const transcript = await response.text();
      if (!current()) return;
      const follow = viewer.scrollHeight - viewer.clientHeight - viewer.scrollTop <= 40;
      viewer.textContent = transcript;
      if (follow) viewer.scrollTop = viewer.scrollHeight;
      reconnecting = false;
      showState();
    } catch (_) {
      if (!current()) return;
      reconnecting = true;
      showState();
    }
    if (current()) timer = window.setTimeout(() => read(token), 2500);
  };
  const open = (config) => {
    if (!["setup", "minecraft_reset", "factory_reset"].includes(config.kind) || !/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/.test(config.operationID)) return;
    ensureWindow();
    if (dialog.open && selected?.operationID === config.operationID && selected.kind === config.kind) {
      selected.state = config.state;
      selected.stage = config.stage;
      showState();
      workspace.bringToFront();
      return;
    }
    stop();
    const id = encodeURIComponent(config.operationID);
    selected = {
      kind: config.kind, operationID: config.operationID, state: config.state, stage: config.stage,
      onUnauthorized: config.onUnauthorized,
      url: `/api/operations/${id}/trace`,
    };
    reconnecting = false;
    label.textContent = labels[config.kind];
    viewer.textContent = "";
    viewer.scrollTop = 0;
    showState();
    workspace.open();
    void read(generation);
  };
  const updateState = (operationID, state, stage) => {
    if (selected?.operationID !== operationID || !dialog?.open) return;
    selected.state = state;
    selected.stage = stage;
    showState();
  };
  window.addEventListener("pagehide", close);
  window.addEventListener("beforeunload", close);
  return { open, close, updateState };
})();

document.addEventListener("keydown", (event) => {
  if (event.key !== "Escape") return;
  if (document.querySelector("dialog:modal")) return;
  const openWindows = Array.from(workspaceWindows).filter((element) => element.open);
  if (openWindows.length === 0) return;
  openWindows.sort((a, b) => Number(b.style.zIndex || 0) - Number(a.style.zIndex || 0));
  if (openWindows[0] === systemWorkspaceDialog && window.JustVoxelWallpaper?.canLeave() === false) return;
  openWindows[0].close();
});

window.addEventListener("resize", () => {
  workspaceWindows.forEach((element) => clampWorkspaceWindow(element));
});

initSystemUpdateWorkspace();

const storageOpen = document.querySelector("[data-storage-open]");
const storageDialog = document.querySelector("[data-storage-workspace-dialog]");
if (storageOpen && storageDialog) {
  const closeButton = storageDialog.querySelector("[data-storage-close]");
  const refreshButton = storageDialog.querySelector("[data-storage-refresh]");
  const state = storageDialog.querySelector("[data-storage-state]");
  const content = storageDialog.querySelector("[data-storage-workspace-content]");
  let loadSequence = 0;

  const ensureStorageBrowserAssets = async () => {
    let stylesheet = document.querySelector('link[href="/static/storage-browser.css"]');
    if (!stylesheet) {
      stylesheet = document.createElement("link");
      stylesheet.rel = "stylesheet";
      stylesheet.href = "/static/storage-browser.css";
      document.head.appendChild(stylesheet);
    }

    if (!document.querySelector('link[href="/static/new-backups.css"]')) {
      const css = document.createElement("link"); css.rel = "stylesheet"; css.href = "/static/new-backups.css"; document.head.appendChild(css);
    }
    if (!window.JustVoxelNewBackups?.init) {
      await new Promise((resolve, reject) => {
        let script = document.querySelector('script[src="/static/new-backups.js"]');
        if (!script) { script = document.createElement("script"); script.src = "/static/new-backups.js"; document.head.appendChild(script); }
        script.addEventListener("load", resolve, { once: true });
        script.addEventListener("error", reject, { once: true });
      });
    }
    if (window.JustVoxelStorageBrowser?.init) return;
    let script = document.querySelector('script[src="/static/storage-browser.js"]');
    await new Promise((resolve, reject) => {
      if (window.JustVoxelStorageBrowser?.init) {
        resolve();
        return;
      }
      if (!script) {
        script = document.createElement("script");
        script.src = "/static/storage-browser.js";
        script.defer = true;
        document.head.appendChild(script);
      }
      script.addEventListener("load", resolve, { once: true });
      script.addEventListener("error", reject, { once: true });
    });
  };

  const loadStorage = async (url = "/workspace/storage", request = {}) => {
    const sequence = ++loadSequence;
    if (state) state.textContent = jvT("ui.dd427d4c1995");
    if (refreshButton) refreshButton.disabled = true;
    try {
      await ensureStorageBrowserAssets();
      const response = await fetch(url, {
        ...request,
        method: request.method || "GET",
        credentials: "same-origin",
        headers: { Accept: "text/html" },
        cache: "no-store",
      });
      if (response.redirected && new URL(response.url).pathname === "/login") {
        window.location.assign("/login");
        return;
      }
      if (response.status === 401) {
        window.location.assign("/login");
        return;
      }
      if (response.status === 403) throw new Error(jvT("ui.2d3d0b30bde9"));
      if (!response.ok) throw new Error(jvT("ui.6225995eb81b"));
      const markup = await response.text();
      if (sequence !== loadSequence || !content) return;
      content.innerHTML = markup;
      const root = content.querySelector("[data-storage-browser-root]");
      if (!root || !window.JustVoxelStorageBrowser?.init) throw new Error(jvT("ui.ecfe1801cd30"));
      window.JustVoxelStorageBrowser.init(root);
      window.JustVoxelNewBackups?.init(root);
      root.querySelectorAll('form[action^="/workspace/storage/destination/"]').forEach(form => {
        form.addEventListener("submit", async event => {
          event.preventDefault();
          const submitter = event.submitter;
          if (submitter) submitter.disabled = true;
          await loadStorage(form.action, { method: "POST", body: new URLSearchParams(new FormData(form)) });
        });
      });
      root.querySelectorAll('a[href="/workspace/storage"]').forEach(link => link.addEventListener("click", event => { event.preventDefault(); loadStorage(); }));
      if (state) state.textContent = "";
    } catch (error) {
      if (sequence !== loadSequence) return;
      if (content) content.replaceChildren();
      if (state) state.textContent = error?.message || jvT("ui.6225995eb81b");
    } finally {
      if (sequence === loadSequence && refreshButton) refreshButton.disabled = false;
    }
  };

  const workspaceWindow = setupWorkspaceWindow(storageDialog, { onOpen: () => loadStorage() });

  storageOpen.addEventListener("click", () => {
    if (controlCenter) controlCenter.open = false;
    workspaceWindow?.open();
  });
  closeButton?.addEventListener("click", () => workspaceWindow?.close());
  refreshButton?.addEventListener("click", () => loadStorage());

  const restoreStorageWhenRoleKnown = () => {
    if (!readWorkspaceWindowState("storage").open) return;
    if (document.body.classList.contains("role-administrator")) {
      workspaceWindow?.open();
      return;
    }
    if (!document.body.classList.contains("role-pending")) return;
    const observer = new MutationObserver(() => {
      if (document.body.classList.contains("role-pending")) return;
      observer.disconnect();
      if (document.body.classList.contains("role-administrator")) workspaceWindow?.open();
    });
    observer.observe(document.body, { attributes: true, attributeFilter: ["class"] });
  };
  restoreStorageWhenRoleKnown();
}

const backupsOpen = document.querySelector("[data-backups-open]");
const backupsDialog = document.querySelector("[data-backups-workspace-dialog]");
if (backupsOpen && backupsDialog) {
  const closeButton = backupsDialog.querySelector("[data-backups-close]");
  const refreshButton = backupsDialog.querySelector("[data-backups-refresh]");
  const state = backupsDialog.querySelector("[data-backups-state]");
  const content = backupsDialog.querySelector("[data-backups-workspace-content]");
  const backupsDestination = new URLSearchParams(window.location.search);
  const backupParams = new URLSearchParams();
  if (backupsDestination.get("workspace") === "backups") {
    for (const key of ["result", "count"]) {
      if (backupsDestination.has(key)) backupParams.set(key, backupsDestination.get(key));
    }
  }
  globalThis.justVoxelServerSelection(backupsDialog.querySelector("[data-server-selector-slot]")).catch((error)=>{if(state)state.textContent=error.message;});
  let currentURL = "/workspace/backups" + (backupParams.size ? "?" + backupParams : "");
  let loadSequence = 0;

  const loadBackupsScript = async (src, ready) => {
    if (ready()) return;
    let script = document.querySelector(`script[src="${src}"]`);
    await new Promise((resolve, reject) => {
      if (ready()) {
        resolve();
        return;
      }
      if (!script) {
        script = document.createElement("script");
        script.src = src;
        script.defer = true;
        document.head.appendChild(script);
      }
      script.addEventListener("load", resolve, { once: true });
      script.addEventListener("error", reject, { once: true });
    });
  };

  const ensureBackupsAssets = async () => {
    if (!document.querySelector('link[href="/static/new-backups.css"]')) {
      const stylesheet = document.createElement("link");
      stylesheet.rel = "stylesheet";
      stylesheet.href = "/static/new-backups.css";
      document.head.appendChild(stylesheet);
    }
    await loadBackupsScript("/static/new-backups.js", () => Boolean(window.JustVoxelNewBackups?.init));
    await loadBackupsScript("/static/restore-operation.js", () => Boolean(window.JustVoxelRestoreOperation?.init));
  };

  const renderBackupsMarkup = (markup) => {
    if (!content) throw new Error(jvT("ui.ca1e384883f7"));
    content.innerHTML = markup;
    const root = content.querySelector("[data-backups-workspace-root]");
    if (!root) throw new Error(jvT("ui.bb2b17baf73c"));

    const handleSubmit = async (event) => {
      const form = event.target.closest("form");
      if (!form || !root.contains(form)) return;
      const action = new URL(form.getAttribute("action") || currentURL, window.location.href);
      if (action.origin !== window.location.origin || !action.pathname.startsWith("/workspace/backups")) return;

      event.preventDefault();
      const selection = globalThis.justVoxelSelectionGeneration.backups;
      const sequence = ++loadSequence;
      const reviewDialog = form.closest("[data-backup-review-dialog]");
      const reviewClose = reviewDialog?.querySelector("[data-backup-workflow-review-close]");
      const submitter = event.submitter;
      if (reviewDialog) reviewDialog.dataset.busy = "true";
      if (reviewClose) reviewClose.disabled = true;
      if (submitter) submitter.disabled = true;
      if (state) state.textContent = jvT("ui.5474eef8d0f1");

      try {
        const body = new URLSearchParams();
        new FormData(form).forEach((value, key) => body.append(key, String(value)));
        const response = await fetch(globalThis.justVoxelSelectedURL(action.pathname + action.search, "backups"), {
          method: (form.method || "POST").toUpperCase(),
          credentials: "same-origin",
          headers: {
            Accept: "text/html",
            "Content-Type": "application/x-www-form-urlencoded",
          },
          body: body.toString(),
          cache: "no-store",
          redirect: "follow",
        });

        if (sequence !== loadSequence || selection !== globalThis.justVoxelSelectionGeneration.backups) return;
        if (response.redirected) {
          const target = new URL(response.url);
          if (target.pathname === "/login" || target.pathname === "/password") {
            window.location.assign(target.pathname + target.search);
            return;
          }
          if (target.origin === window.location.origin && target.pathname === "/workspace/backups") {
            currentURL = target.pathname + target.search;
          }
        }

        if (response.status === 401) {
          window.location.assign("/login");
          return;
        }
        if (response.status === 403) {
          const message = await response.text();
          if (jvC(message).toLowerCase().includes("password change")) {
            window.location.assign("/password");
            return;
          }
          throw new Error(jvT("ui.2d3d0b30bde9"));
        }
        if (!response.ok) {
          const message = await response.text();
          throw new Error(message.trim() || jvT("ui.90b05e7eb0f5"));
        }

        const responseMarkup = await response.text();
        if (sequence !== loadSequence || selection !== globalThis.justVoxelSelectionGeneration.backups) return;
        renderBackupsMarkup(responseMarkup);
        if (state) state.textContent = "";
      } catch (error) {
        if (sequence !== loadSequence || selection !== globalThis.justVoxelSelectionGeneration.backups) return;
        if (reviewDialog) reviewDialog.dataset.busy = "false";
        if (reviewClose) reviewClose.disabled = false;
        if (submitter) submitter.disabled = false;
        if (state) state.textContent = error?.message || jvT("ui.90b05e7eb0f5");
      }
    };

    const handleClick = async (event) => {
      const link = event.target.closest("a");
      if (!link || !root.contains(link)) return;
      const href = link.getAttribute("href") || "";

      if (href === "#backup-library") {
        event.preventDefault();
        root.querySelector("#backup-library")?.scrollIntoView({ behavior: "smooth", block: "start" });
        return;
      }
      if (href.startsWith("/workspace/backups")) {
        event.preventDefault();
        await loadBackups(href);
      }
    };

    root.addEventListener("submit", handleSubmit);
    root.addEventListener("click", handleClick);
    window.JustVoxelNewBackups?.init(root);
    window.JustVoxelRestoreOperation?.init(root);
  };

  const loadBackups = async (url = currentURL) => {
    if (globalThis.justVoxelSelectionReady) await globalThis.justVoxelSelectionReady;
    const selection = globalThis.justVoxelSelectionGeneration.backups;
    const sequence = ++loadSequence;
    if (state) state.textContent = jvT("ui.20af4b11362c");
    if (refreshButton) refreshButton.disabled = true;
    try {
      await ensureBackupsAssets();
      const response = await fetch(globalThis.justVoxelSelectedURL(url, "backups"), {
        method: "GET",
        credentials: "same-origin",
        headers: { Accept: "text/html" },
        cache: "no-store",
        redirect: "follow",
      });

      if (response.redirected) {
        const target = new URL(response.url);
        if (target.pathname === "/login" || target.pathname === "/password") {
          window.location.assign(target.pathname + target.search);
          return;
        }
      }
      if (response.status === 401) {
        window.location.assign("/login");
        return;
      }
      if (response.status === 403) {
        const message = await response.text();
        if (jvC(message).toLowerCase().includes("password change")) {
          window.location.assign("/password");
          return;
        }
        throw new Error(jvT("ui.2d3d0b30bde9"));
      }
      if (!response.ok) throw new Error(jvT("ui.1e30d919bf9e"));

      const markup = await response.text();
      if (sequence !== loadSequence || selection !== globalThis.justVoxelSelectionGeneration.backups) return;
      const responseURL = new URL(response.url);
      if (responseURL.origin === window.location.origin && responseURL.pathname === "/workspace/backups") {
        currentURL = responseURL.pathname + responseURL.search;
      } else {
        currentURL = url;
      }
      renderBackupsMarkup(markup);
      if (state) state.textContent = "";
    } catch (error) {
      if (sequence !== loadSequence || selection !== globalThis.justVoxelSelectionGeneration.backups) return;
      if (content) content.replaceChildren();
      if (state) state.textContent = error?.message || jvT("ui.1e30d919bf9e");
    } finally {
      if ((sequence === loadSequence && selection === globalThis.justVoxelSelectionGeneration.backups) && refreshButton) refreshButton.disabled = false;
    }
  };

  window.addEventListener("justvoxel:server-selection-changed", (event) => {
    if (event.detail?.workspace !== "backups") return;
    ++loadSequence;
    currentURL = "/workspace/backups";
    content?.replaceChildren();
    if (backupsDialog.open) void loadBackups();
  });

  const refreshWorkspace = () => globalThis.justVoxelServerSelection(backupsDialog.querySelector("[data-server-selector-slot]"), true)
    .then(() => loadBackups(currentURL)).catch(error => { if (state) state.textContent = error.message; });
  const workspaceWindow = setupWorkspaceWindow(backupsDialog, { onOpen: refreshWorkspace });

  window.JustVoxelBackupsWorkspace = {
    reload: async (url = "/workspace/backups") => {
      currentURL = url;
      await loadBackups(url);
    },
  };

  backupsOpen.addEventListener("click", () => {
    if (controlCenter) controlCenter.open = false;
    workspaceWindow?.open();
  });
  closeButton?.addEventListener("click", () => workspaceWindow?.close());
  refreshButton?.addEventListener("click", refreshWorkspace);

  const restoreBackupsWhenRoleKnown = () => {
    if (!readWorkspaceWindowState("backups").open) return;
    if (document.body.classList.contains("role-administrator")) {
      workspaceWindow?.open();
      return;
    }
    if (!document.body.classList.contains("role-pending")) return;
    const observer = new MutationObserver(() => {
      if (document.body.classList.contains("role-pending")) return;
      observer.disconnect();
      if (document.body.classList.contains("role-administrator")) workspaceWindow?.open();
    });
    observer.observe(document.body, { attributes: true, attributeFilter: ["class"] });
  };
  restoreBackupsWhenRoleKnown();
}


const migrationOpen = document.querySelector("[data-migration-open]");
document.addEventListener("click", (event) => {
  const open = event.target.closest("[data-recovery-open-confirmation]");
  if (open) {
    open.closest("[data-recovery-cleanup-form]")?.querySelector("[data-recovery-confirmation-dialog]")?.showModal();
    return;
  }
  const cancel = event.target.closest("[data-recovery-cancel]");
  if (cancel) { cancel.closest("dialog")?.close(); return; }
  const confirm = event.target.closest("[data-recovery-confirm]");
  if (confirm) {
    const form = confirm.closest("[data-recovery-cleanup-form]");
    if (!form) return;
    form.querySelector("[data-recovery-confirmed]").value = "yes";
    confirm.closest("dialog")?.close();
    form.requestSubmit();
  }
});

const migrationDialog = document.querySelector("[data-migration-workspace-dialog]");
if (migrationOpen && migrationDialog) {
  const closeButton = migrationDialog.querySelector("[data-migration-close]");
  const refreshButton = migrationDialog.querySelector("[data-migration-refresh]");
  const state = migrationDialog.querySelector("[data-migration-state]");
  const content = migrationDialog.querySelector("[data-migration-workspace-content]");
  const tabs = Array.from(migrationDialog.querySelectorAll("[data-migration-tab]"));
  const recoveryTab = tabs.find((button) => button.dataset.migrationTab === "recovery");
  globalThis.justVoxelServerSelection(migrationDialog.querySelector("[data-server-selector-slot]")).catch((error) => { if (state) state.textContent = error.message; });
  const tabURLs = {
    export: "/workspace/migration/export",
    import: "/workspace/migration/import",
    switch: "/workspace/migration/switch",
    recovery: "/workspace/migration/recovery",
  };
  const migrationDestination = new URLSearchParams(window.location.search);
  const migrationRequested = migrationDestination.get("workspace") === "migration";
  const initialMigrationTab = migrationRequested ? migrationDestination.get("tab") : "";
  let currentTab = Object.hasOwn(tabURLs, initialMigrationTab) ? initialMigrationTab : "export";
  let currentURL = tabURLs[currentTab];
  let loadSequence = 0;
  let migrationSourceSMBPassword = "";
  let migrationExportSMBPassword = "";
  let migrationLoadController = null;
  let migrationSubmitController = null;

  const updateMigrationLocation = (tab = "", operation = "") => {
    const destination = new URL(globalThis.justVoxelSelectedURL("/?workspace=migration", "migration"), window.location.href);
    if (operation) destination.searchParams.set("operation", operation);
    else if (Object.hasOwn(tabURLs, tab)) destination.searchParams.set("tab", tab);
    const nextURL = destination.pathname + destination.search;
    if (window.location.pathname + window.location.search !== nextURL) {
      window.history.replaceState(window.history.state, "", nextURL);
    }
  };

  const setMigrationBusy = (message = "", kind = "busy") => {
    if (state) {
      state.textContent = message;
      state.classList.remove("is-busy", "is-error", "is-info");
      if (message) state.classList.add(kind === "error" ? "is-error" : kind === "info" ? "is-info" : "is-busy");
    }
    migrationDialog.classList.toggle("is-busy", Boolean(message) && kind === "busy");
  };

  const abortMigrationRequests = () => {
    migrationLoadController?.abort();
    migrationSubmitController?.abort();
    migrationLoadController = null;
    migrationSubmitController = null;
    setMigrationBusy("");
  };

  const requestMigrationSMBPassword = (purpose) => new Promise((resolve) => {
    const dialog = document.createElement("dialog");
    dialog.className = "system-action-dialog";
    dialog.setAttribute("aria-label", jvT("ui.c144fd7c3b42"));
    const shell = document.createElement("div");
    shell.className = "system-action-dialog-content";
    const title = document.createElement("h2");
    title.textContent = jvT("ui.ae34ce1c7e57");
    const copy = document.createElement("p");
    copy.textContent = purpose === "import"
      ? jvT("ui.cbbbd4e8cf9a")
      : jvT("ui.e07626c4be0b");
    const label = document.createElement("label");
    label.textContent = jvT("ui.ae34ce1c7e57");
    const input = document.createElement("input");
    input.type = "password";
    input.autocomplete = "off";
    input.required = true;
    label.appendChild(input);
    const actions = document.createElement("div");
    actions.className = "action-row";
    const cancel = document.createElement("button");
    cancel.type = "button";
    cancel.className = "secondary";
    cancel.textContent = jvT("ui.19766ed6ccb2");
    const use = document.createElement("button");
    use.type = "button";
    use.textContent = jvT("ui.31fbef162594");
    actions.append(cancel, use);
    shell.append(title, copy, label, actions);
    dialog.appendChild(shell);
    document.body.appendChild(dialog);
    initializeSharedWebUIControls(dialog);

    const finish = (value) => {
      window.removeEventListener("justvoxel:server-selection-changed", detach);
      input.value = "";
      if (dialog.open) dialog.close();
      dialog.remove();
      resolve(value);
    };
    const detach = (event) => { if (event.detail?.workspace === "migration") finish(null); };
    window.addEventListener("justvoxel:server-selection-changed", detach);
    cancel.addEventListener("click", () => finish(null), { once: true });
    dialog.addEventListener("cancel", (event) => {
      event.preventDefault();
      finish(null);
    }, { once: true });
    use.addEventListener("click", () => {
      if (!input.value) {
        input.focus();
        return;
      }
      finish(input.value);
    }, { once: true });
    input.addEventListener("keydown", (event) => {
      if (event.key === "Enter") {
        event.preventDefault();
        use.click();
      }
    });
    dialog.showModal();
    input.focus();
  });

  const loadMigrationScript = async (src, ready) => {
    if (ready()) return;
    let script = document.querySelector('script[src="' + src + '"]');
    await new Promise((resolve, reject) => {
      if (ready()) {
        resolve();
        return;
      }
      if (!script) {
        script = document.createElement("script");
        script.src = src;
        script.defer = true;
        document.head.appendChild(script);
      }
      script.addEventListener("load", resolve, { once: true });
      script.addEventListener("error", reject, { once: true });
    });
  };

  const ensureMigrationAssets = async () => {
    await loadMigrationScript("/static/server-migration-export.js", () => Boolean(window.JustVoxelServerMigrationExport?.init));
    await loadMigrationScript("/static/server-migration-import.js", () => Boolean(window.JustVoxelServerMigrationImport?.init));
    await loadMigrationScript("/static/server-migration-operation.js", () => Boolean(window.JustVoxelServerMigrationOperation?.init));
    await loadMigrationScript("/static/server-migration-switch.js", () => Boolean(window.JustVoxelServerMigrationSwitch?.init));
  };

  const syncMigrationTabs = () => {
    tabs.forEach((button) => {
      button.setAttribute("aria-selected", button.dataset.migrationTab === currentTab ? "true" : "false");
    });
  };

  const syncMigrationRecoveryAvailability = (root) => {
    if (!recoveryTab || !root) return;
    if (!Object.hasOwn(root.dataset, "migrationRecoveryAvailable")) return;
    const available = root.dataset.migrationRecoveryAvailable === "true";
    recoveryTab.hidden = !available && currentTab !== "recovery";
  };

  const inferMigrationTab = (url, root = null) => {
    const pathname = new URL(url, window.location.href).pathname;
    if (pathname.includes("/migration/export")) return "export";
    if (pathname.includes("/migration/import")) return "import";
    if (pathname.includes("/migration/recovery")) return "recovery";
    if (pathname.includes("/migration/switch")) return "switch";
    const operationName = root?.querySelector("#server-migration-operation-name")?.textContent || "";
    if (operationName.includes("Export")) return "export";
    if (operationName.includes("Import")) return "import";
    if (operationName.includes("Recovery")) return "recovery";
    return currentTab;
  };

  const extractMigrationRoot = (markup) => {
    const shell = document.createElement("div");
    shell.innerHTML = markup;
    return shell.querySelector("[data-migration-workspace-root]");
  };

  const handleMigrationAuth = async (response) => {
    if (response.redirected) {
      const target = new URL(response.url);
      if (target.pathname === "/login" || target.pathname === "/password") {
        window.location.assign(target.pathname + target.search);
        return true;
      }
    }
    if (response.status === 401) {
      window.location.assign("/login");
      return true;
    }
    if (response.status === 403) {
      const message = await response.clone().text();
      if (jvC(message).toLowerCase().includes("password change")) {
        window.location.assign("/password");
        return true;
      }
    }
    return false;
  };

  const initializeMigrationRoot = (root) => {
    window.JustVoxelServerMigrationExport?.init(root);
    window.JustVoxelServerMigrationImport?.init(root);
    window.JustVoxelServerMigrationOperation?.init(root);
    window.JustVoxelServerMigrationSwitch?.init(root);
  };

  const renderMigrationMarkup = (markup, responseURL, preserveLocation = false) => {
    const selection = globalThis.justVoxelSelectionGeneration.migration;
    if (!content) throw new Error(jvT("ui.189db47cd63c"));
    const root = extractMigrationRoot(markup);
    if (!root) throw new Error(jvT("ui.e83f2f640a25"));

    const renderedURL = new URL(responseURL, window.location.href);
    currentURL = renderedURL.pathname + renderedURL.search;
    const progress = renderedURL.origin === window.location.origin
      ? renderedURL.pathname.match(/^\/workspace\/migration\/progress\/([A-Za-z0-9_-]+)$/)
      : null;
    if (root.dataset.serverId && !new URLSearchParams(window.location.search).has("server")) {
      const selectedURL = new URL(window.location.href);
      selectedURL.searchParams.set("workspace", "migration");
      selectedURL.searchParams.set("server", root.dataset.serverId);
      window.history.replaceState(window.history.state, "", selectedURL.pathname + selectedURL.search);
    }
    if (progress && !preserveLocation) updateMigrationLocation("", progress[1]);
    currentTab = inferMigrationTab(responseURL, root);
    syncMigrationTabs();
    syncMigrationRecoveryAvailability(root);

    if (currentTab === "import") {
      const sourceEntries = Array.from(root.querySelectorAll('input[type="radio"][name="source_path"]'));
      if (sourceEntries.length === 1) sourceEntries[0].checked = true;
      const roots = Array.from(root.querySelectorAll('input[type="radio"][name="selected_root"]'));
      if (roots.length === 1) roots[0].checked = true;
      if (migrationSourceSMBPassword) {
        root.querySelectorAll("[data-migration-source-password-repeat]").forEach((node) => node.remove());
      }
    }
    content.replaceChildren(root);

    const attachFailedResetRecovery = async () => {
      if (!root.textContent.includes("Full Factory Reset")) return;
      const csrf = root.querySelector('input[name="csrf"]')?.value || "";
      if (!csrf) return;
      try {
        const statusResponse = await fetch("/api/system/workspace/reset/factory/current", {
          method: "GET",
          credentials: "same-origin",
          headers: { Accept: "application/json" },
          cache: "no-store",
        });
        if (await handleMigrationAuth(statusResponse) || !statusResponse.ok) return;
        const status = await statusResponse.json();
        if (selection !== globalThis.justVoxelSelectionGeneration.migration) return;
        const operation = status?.operation;
        if (!operation || operation.operation_type !== "factory_reset" || operation.state !== "needs_attention") return;

        const target = root.querySelector(".notice.error");
        if (!target || root.querySelector("[data-migration-reset-recovery]")) return;
        const recovery = document.createElement("div");
        recovery.className = "notice warning";
        recovery.dataset.migrationResetRecovery = "true";
        recovery.innerHTML = `<strong>${jvH("ui.6b24d3da910b")}</strong><br>${jvH("ui.f03112a21ee8")}`;
        const actions = document.createElement("div");
        actions.className = "action-row";
        const button = document.createElement("button");
        button.type = "button";
        button.className = "secondary";
        button.textContent = jvT("ui.fa4efc5678a7");
        actions.appendChild(button);
        recovery.appendChild(actions);
        target.insertAdjacentElement("afterend", recovery);

        button.addEventListener("click", async () => {
          button.disabled = true;
          setMigrationBusy(jvT("ui.5bf7125616a4"));
          const body = new URLSearchParams({
            csrf,
            operation_id: operation.operation_id,
            keep_current_state: "yes",
          });
          try {
            const response = await fetch("/api/system/workspace/reset/factory/resolve", {
              method: "POST",
              credentials: "same-origin",
              headers: {
                Accept: "application/json",
                "Content-Type": "application/x-www-form-urlencoded",
              },
              body: body.toString(),
              cache: "no-store",
            });
            if (await handleMigrationAuth(response)) return;
            const result = await response.json().catch(() => ({}));
            if (!response.ok) throw new Error(result.error || jvT("ui.7cfc47e3f260"));
            if (selection !== globalThis.justVoxelSelectionGeneration.migration) return;
            setMigrationBusy(jvT("ui.ca7e73d1bf34"), "info");
            await loadMigration(tabURLs[currentTab]);
          } catch (error) {
            if (selection !== globalThis.justVoxelSelectionGeneration.migration) return;
            button.disabled = false;
            setMigrationBusy(error?.message || jvT("ui.7cfc47e3f260"), "error");
          }
        });
      } catch (_) {
        // Leave the original exact blocker visible if recovery status cannot be loaded.
      }
    };
    void attachFailedResetRecovery();

    root.addEventListener("submit", async (event) => {
      const form = event.target.closest("form");
      if (!form || !root.contains(form)) return;
      const action = new URL(form.getAttribute("action") || currentURL, window.location.href);
      if (action.origin !== window.location.origin || !action.pathname.startsWith("/workspace/migration")) return;

      event.preventDefault();
      migrationSubmitController?.abort();
      const submitController = new AbortController();
      migrationSubmitController = submitController;
      const submitter = event.submitter;
      if (submitter) submitter.disabled = true;
      const isReview = action.pathname.endsWith("/review");
      setMigrationBusy(isReview ? jvT("ui.e0fd374cce36") : jvT("ui.e447c6e994b8"));

      try {
        const body = new URLSearchParams();
        new FormData(form).forEach((value, key) => body.append(key, String(value)));
        const sourceKind = body.get("source_kind") || "";
        const exportKind = body.get("kind") || "";
        if (sourceKind === "smb") {
          if (!migrationSourceSMBPassword) {
            const password = await requestMigrationSMBPassword("import");
            if (selection !== globalThis.justVoxelSelectionGeneration.migration) return;
            if (password === null) {
              if (submitter) submitter.disabled = false;
              setMigrationBusy("");
              return;
            }
            migrationSourceSMBPassword = password;
          }
          body.set("source_smb_password", migrationSourceSMBPassword);
        } else if (sourceKind) {
          migrationSourceSMBPassword = "";
          body.delete("source_smb_password");
        }

        if (exportKind === "smb" && action.pathname.endsWith("/export/apply")) {
          if (!migrationExportSMBPassword) {
            const password = await requestMigrationSMBPassword("export");
            if (selection !== globalThis.justVoxelSelectionGeneration.migration) return;
            if (password === null) {
              if (submitter) submitter.disabled = false;
              setMigrationBusy("");
              return;
            }
            migrationExportSMBPassword = password;
          }
          body.set("smb_password", migrationExportSMBPassword);
        }
        form.querySelectorAll('input[name="source_smb_password"],input[name="smb_password"]').forEach((input) => { input.value = ""; });
        if (selection !== globalThis.justVoxelSelectionGeneration.migration) return;
        const migrationActionURL = globalThis.justVoxelSelectedURL(action.pathname + action.search, "migration");
        const sendMigration = () => fetch(migrationActionURL, {
          method: (form.method || "POST").toUpperCase(),
          credentials: "same-origin",
          headers: {
            Accept: action.pathname === "/workspace/migration/import/apply" ? "text/html, application/json" : "text/html",
            "Content-Type": "application/x-www-form-urlencoded",
            "X-JustVoxel-Migration-Fragment": "1",
          },
          body: body.toString(),
          cache: "no-store",
          redirect: "follow",
          signal: submitController.signal,
        });
        const response = await sendMigration();
        if (selection !== globalThis.justVoxelSelectionGeneration.migration) return;
        if (await handleMigrationAuth(response)) return;
        if (response.status === 403) throw new Error(jvT("ui.2d3d0b30bde9"));

        const responseMarkup = await response.text();
        if (selection !== globalThis.justVoxelSelectionGeneration.migration) return;
        if (!response.ok && !responseMarkup.includes("data-migration-workspace-root")) {
          const detail = responseMarkup.replace(/<[^>]*>/g, " ").replace(/\s+/g, " ").trim();
          throw new Error(detail || jvT("ui.23d3fe85833f", {v0: (response.status)}));
        }
        const responseURL = response.url || action.pathname;
        if (action.pathname === "/workspace/migration/recovery/resolve" && new URL(responseURL, window.location.href).pathname === "/workspace/migration") {
          currentTab = "export";
          updateMigrationLocation();
          await loadMigrationEntry();
        } else {
          renderMigrationMarkup(responseMarkup, response.url || action.pathname);
        }
        if ((response.url || "").includes("/migration/progress/")) {
          migrationSourceSMBPassword = "";
          migrationExportSMBPassword = "";
        }
        setMigrationBusy("");
      } catch (error) {
        if (error?.name === "AbortError" || selection !== globalThis.justVoxelSelectionGeneration.migration) return;
        if (submitter && submitter.isConnected) submitter.disabled = false;
        setMigrationBusy(error?.message || jvT("ui.db7aa31acc6c"), "error");
      } finally {
        if (migrationSubmitController === submitController) migrationSubmitController = null;
      }
    });

    root.addEventListener("click", async (event) => {
      const link = event.target.closest("a");
      if (!link || !root.contains(link)) return;
      const href = link.getAttribute("href") || "";
      if (href === "/") {
        event.preventDefault();
        workspaceWindow?.close();
        return;
      }
      const destination = new URL(href, window.location.href);
      if (destination.origin === window.location.origin && destination.pathname === "/" && destination.searchParams.get("workspace") === "migration") {
        event.preventDefault();
        migrationSubmitController?.abort();
        migrationSubmitController = null;
        const operation = destination.searchParams.get("operation");
        const tab = destination.searchParams.get("tab");
        updateMigrationLocation(tab, operation);
        if (operation) await loadMigration("/workspace/migration/progress/" + encodeURIComponent(operation));
        else if (Object.hasOwn(tabURLs, tab)) {
          if (tab === "import") migrationSourceSMBPassword = "";
          currentTab = tab;
          currentURL = tabURLs[tab];
          syncMigrationTabs();
          await loadMigration(currentURL);
        } else await loadMigrationEntry();
        return;
      }
      if (!href.startsWith("/workspace/migration")) return;
      event.preventDefault();
      migrationSubmitController?.abort();
      migrationSubmitController = null;
      if (href === "/workspace/migration") await loadMigrationEntry();
      else {
        if (href === "/workspace/migration/import") migrationSourceSMBPassword = "";
        await loadMigration(href);
      }
    });

    initializeMigrationRoot(root);
  };

  const loadMigration = async (url = currentURL, preserveLocation = false) => {
    if (globalThis.justVoxelSelectionReady) await globalThis.justVoxelSelectionReady;
    const selection = globalThis.justVoxelSelectionGeneration.migration;
    const sequence = ++loadSequence;
    migrationLoadController?.abort();
    const loadController = new AbortController();
    migrationLoadController = loadController;
    setMigrationBusy(jvT("ui.32b3e138ccff"));
    if (refreshButton) refreshButton.disabled = true;
    try {
      await ensureMigrationAssets();
      const response = await fetch(globalThis.justVoxelSelectedURL(url, "migration"), {
        method: "GET",
        credentials: "same-origin",
        headers: { Accept: "text/html", "X-JustVoxel-Migration-Fragment": "1" },
        cache: "no-store",
        redirect: "follow",
        signal: loadController.signal,
      });
      if (sequence !== loadSequence || selection !== globalThis.justVoxelSelectionGeneration.migration) return;
      if (await handleMigrationAuth(response)) return;
      if (response.status === 403) throw new Error(jvT("ui.2d3d0b30bde9"));
      if (!response.ok) throw new Error(jvT("ui.9418a8c7b777"));

      const finalURL = new URL(response.url || url, window.location.href);
      if (finalURL.pathname === "/workspace/migration") {
        await loadMigration(tabURLs[currentTab], preserveLocation);
        return;
      }
      const markup = await response.text();
      if (sequence !== loadSequence || selection !== globalThis.justVoxelSelectionGeneration.migration) return;
      renderMigrationMarkup(markup, finalURL.href, preserveLocation);
      setMigrationBusy("");
    } catch (error) {
      if (error?.name === "AbortError" || sequence !== loadSequence || selection !== globalThis.justVoxelSelectionGeneration.migration) return;
      if (content) content.replaceChildren();
      setMigrationBusy(error?.message || jvT("ui.9418a8c7b777"));
      state?.classList.remove("is-busy");
      migrationDialog.classList.remove("is-busy");
    } finally {
      if (migrationLoadController === loadController) migrationLoadController = null;
      if ((sequence === loadSequence && selection === globalThis.justVoxelSelectionGeneration.migration) && refreshButton) refreshButton.disabled = false;
    }
  };

  const loadMigrationEntry = async () => {
    if (globalThis.justVoxelSelectionReady) await globalThis.justVoxelSelectionReady;
    syncMigrationTabs();
    const selection = globalThis.justVoxelSelectionGeneration.migration;
    const sequence = ++loadSequence;
    migrationLoadController?.abort();
    const loadController = new AbortController();
    migrationLoadController = loadController;
    setMigrationBusy(jvT("ui.32b3e138ccff"));
    if (refreshButton) refreshButton.disabled = true;
    try {
      await ensureMigrationAssets();
      const response = await fetch(globalThis.justVoxelSelectedURL("/workspace/migration", "migration"), {
        method: "GET",
        credentials: "same-origin",
        headers: { Accept: "text/html", "X-JustVoxel-Migration-Fragment": "1" },
        cache: "no-store",
        redirect: "follow",
        signal: loadController.signal,
      });
      if (sequence !== loadSequence || selection !== globalThis.justVoxelSelectionGeneration.migration) return;
      if (await handleMigrationAuth(response)) return;
      if (response.status === 403) throw new Error(jvT("ui.2d3d0b30bde9"));
      if (!response.ok) throw new Error(jvT("ui.9418a8c7b777"));
      const finalURL = new URL(response.url, window.location.href);
      if (sequence !== loadSequence || selection !== globalThis.justVoxelSelectionGeneration.migration) return;

      if (finalURL.pathname !== "/workspace/migration") {
        const markup = await response.text();
        if (sequence !== loadSequence || selection !== globalThis.justVoxelSelectionGeneration.migration) return;
        renderMigrationMarkup(markup, finalURL.href);
        setMigrationBusy("");
        return;
      }
      const entryMarkup = await response.text();
      if (sequence !== loadSequence || selection !== globalThis.justVoxelSelectionGeneration.migration) return;
      const entryRoot = extractMigrationRoot(entryMarkup);
      if (entryRoot) {
        syncMigrationRecoveryAvailability(entryRoot);
        if (entryRoot.dataset.serverId && !new URLSearchParams(window.location.search).has("server")) {
          const selectedURL = new URL(window.location.href);
          selectedURL.searchParams.set("workspace", "migration");
          selectedURL.searchParams.set("server", entryRoot.dataset.serverId);
          window.history.replaceState(window.history.state, "", selectedURL.pathname + selectedURL.search);
        }
      }
      await loadMigration(tabURLs[currentTab]);
    } catch (error) {
      if (error?.name === "AbortError" || sequence !== loadSequence || selection !== globalThis.justVoxelSelectionGeneration.migration) return;
      if (content) content.replaceChildren();
      setMigrationBusy(error?.message || jvT("ui.9418a8c7b777"));
      state?.classList.remove("is-busy");
      migrationDialog.classList.remove("is-busy");
    } finally {
      if (migrationLoadController === loadController) migrationLoadController = null;
      if ((sequence === loadSequence && selection === globalThis.justVoxelSelectionGeneration.migration) && refreshButton) refreshButton.disabled = false;
    }
  };

  window.addEventListener("justvoxel:server-selection-changed", (event) => {
    if (event.detail?.workspace !== "migration") return;
    ++loadSequence;
    migrationLoadController?.abort();
    // Do not abort an already submitted operation; only detach its presentation.
    migrationSubmitController = null;
    migrationSourceSMBPassword = "";
    migrationExportSMBPassword = "";
    currentURL = tabURLs[currentTab];
    content?.replaceChildren();
    setMigrationBusy("");
    if (migrationDialog.open) void loadMigration(currentURL, true);
  });

  const refreshSelection = () => globalThis.justVoxelServerSelection(migrationDialog.querySelector("[data-server-selector-slot]"), true);
  const workspaceWindow = setupWorkspaceWindow(migrationDialog, {
    onOpen: () => refreshSelection().then(() => {
      const destination = new URLSearchParams(window.location.search);
      const migrationRequested = destination.get("workspace") === "migration";
      const requestedMigrationOperation = migrationRequested ? destination.get("operation") : "";
      const requestedMigrationTab = migrationRequested ? destination.get("tab") : "";
      if (requestedMigrationOperation) return loadMigration("/workspace/migration/progress/" + encodeURIComponent(requestedMigrationOperation));
      if (requestedMigrationTab === "recovery") {
        currentTab = requestedMigrationTab;
        return loadMigration(tabURLs[currentTab]);
      }
      if (Object.hasOwn(tabURLs, requestedMigrationTab)) currentTab = requestedMigrationTab;
      return loadMigrationEntry();
    }).catch(error => { if (state) state.textContent = error.message; }),
    onClose: () => {
      abortMigrationRequests();
      migrationSourceSMBPassword = "";
      migrationExportSMBPassword = "";
    },
  });

  migrationOpen.addEventListener("click", () => {
    if (controlCenter) controlCenter.open = false;
    workspaceWindow?.open();
  });
  closeButton?.addEventListener("click", () => workspaceWindow?.close());
  refreshButton?.addEventListener("click", () => refreshSelection().then(() => loadMigration(currentURL)).catch(error => { if (state) state.textContent = error.message; }));
  tabs.forEach((button) => {
    button.addEventListener("click", async () => {
      abortMigrationRequests();
      const nextTab = button.dataset.migrationTab;
      if (currentTab === "import" && nextTab !== "import") migrationSourceSMBPassword = "";
      if (currentTab === "export" && nextTab !== "export") migrationExportSMBPassword = "";
      currentTab = nextTab;
      currentURL = tabURLs[currentTab];
      updateMigrationLocation(currentTab);
      syncMigrationTabs();
      await loadMigration(currentURL);
    });
  });

  const restoreMigrationWhenRoleKnown = () => {
    if (!migrationRequested && !readWorkspaceWindowState("migration").open) return;
    if (document.body.classList.contains("role-administrator")) {
      workspaceWindow?.open();
      return;
    }
    if (!document.body.classList.contains("role-pending")) return;
    const observer = new MutationObserver(() => {
      if (document.body.classList.contains("role-pending")) return;
      observer.disconnect();
      if (document.body.classList.contains("role-administrator")) workspaceWindow?.open();
    });
    observer.observe(document.body, { attributes: true, attributeFilter: ["class"] });
  };
  restoreMigrationWhenRoleKnown();
}

const systemMonitorOpen = document.querySelector("[data-system-monitor-open]");
const systemMonitorDialog = document.querySelector("[data-system-monitor-dialog]");
if (systemMonitorOpen && systemMonitorDialog) {
  const closeButton = systemMonitorDialog.querySelector("[data-system-monitor-close]");
  const profileToggle = systemMonitorDialog.querySelector("[data-system-monitor-profile-toggle]");
  const profilePanel = systemMonitorDialog.querySelector("[data-system-monitor-profile]");
  const resetButton = systemMonitorDialog.querySelector("[data-system-monitor-reset]");
  const saveButton = systemMonitorDialog.querySelector("[data-system-monitor-save]");
  const profileStatus = systemMonitorDialog.querySelector("[data-system-monitor-profile-status]");
  const profileCSRF = document.querySelector("[data-system-monitor-csrf]");
  const processCount = systemMonitorDialog.querySelector("[data-monitor-process-count]");
  const state = systemMonitorDialog.querySelector("[data-system-monitor-state]");
  const error = systemMonitorDialog.querySelector("[data-system-monitor-error]");
  const cards = Array.from(systemMonitorDialog.querySelectorAll("[data-monitor-card]"));
  const profileInputs = Array.from(systemMonitorDialog.querySelectorAll("[data-monitor-profile]"));
  const defaultProfile = {
    system: true, cpu: true, memory: true, load: true, filesystem: true,
    diskio: true, network: true, processes: true, containers: true, sensors: true, alerts: true,
    process_count: 10,
  };
  let profile = { ...defaultProfile };
  let refreshTimer = null;

  const syncProfile = () => {
    profileInputs.forEach((input) => { input.checked = profile[input.dataset.monitorProfile] !== false; });
    if (processCount) processCount.value = String(profile.process_count || 10);
    cards.forEach((card) => { card.hidden = profile[card.dataset.monitorCard] === false; });
  };

  const loadProfile = async () => {
    try {
      const response = await fetch("/api/system-monitor/profile", {
        method: "GET",
        credentials: "same-origin",
        headers: { Accept: "application/json" },
        cache: "no-store",
      });
      if (response.status === 401) {
        window.location.assign("/login");
        return;
      }
      if (!response.ok) throw new Error(jvT("ui.97553b499b4d"));
      profile = { ...defaultProfile, ...(await response.json()) };
      if (profileStatus) profileStatus.textContent = "";
    } catch (_) {
      profile = { ...defaultProfile };
      if (profileStatus) profileStatus.textContent = jvT("ui.eca5c13b4da8");
    }
    syncProfile();
  };

  const saveProfile = async () => {
    if (!saveButton || !profileCSRF) return;
    const body = new URLSearchParams({ csrf: profileCSRF.value });
    [
      "system", "cpu", "memory", "load", "filesystem", "diskio",
      "network", "processes", "containers", "sensors", "alerts",
    ].forEach((key) => body.set(key, profile[key] === false ? "false" : "true"));
    body.set("process_count", String(profile.process_count || 10));

    saveButton.disabled = true;
    if (profileStatus) profileStatus.textContent = jvT("ui.23e39291d613");
    try {
      const response = await fetch("/api/system-monitor/profile", {
        method: "POST",
        credentials: "same-origin",
        headers: { Accept: "application/json", "Content-Type": "application/x-www-form-urlencoded" },
        body: body.toString(),
      });
      if (response.status === 401) {
        window.location.assign("/login");
        return;
      }
      if (!response.ok) throw new Error(jvT("ui.802f2bef7c23"));
      profile = { ...defaultProfile, ...(await response.json()) };
      syncProfile();
      refreshMonitor();
      if (profileStatus) profileStatus.textContent = jvT("ui.b5c120b316c2");
      if (profilePanel) profilePanel.hidden = true;
    } catch (_) {
      if (profileStatus) profileStatus.textContent = jvT("ui.16efcd21d74f");
    } finally {
      saveButton.disabled = false;
    }
  };
  const fixed = (value, digits = 1) => {
    const parsed = Number(value);
    return Number.isFinite(parsed) ? parsed.toFixed(digits) : "—";
  };
  const percent = (value) => {
    const parsed = Number(value);
    return Number.isFinite(parsed) ? parsed.toFixed(1) + "%" : "—";
  };
  const bytes = (value) => {
    let parsed = Number(value);
    if (!Number.isFinite(parsed)) return "—";
    const units = ["B", "KiB", "MiB", "GiB", "TiB"];
    let index = 0;
    while (Math.abs(parsed) >= 1024 && index < units.length - 1) { parsed /= 1024; index += 1; }
    return parsed.toFixed(index === 0 ? 0 : 1) + " " + units[index];
  };
  const rate = (value) => {
    const rendered = bytes(value);
    return rendered === "—" ? rendered : rendered + "/s";
  };
  const setText = (selector, value) => {
    const node = systemMonitorDialog.querySelector(selector);
    if (node) node.textContent = value;
  };
  const renderLines = (selector, entries) => {
    const node = systemMonitorDialog.querySelector(selector);
    if (!node) return;
    node.replaceChildren();
    entries.filter((entry) => entry[1] !== undefined && entry[1] !== null && entry[1] !== "—").forEach(([label, value]) => {
      const row = document.createElement("div");
      const key = document.createElement("span");
      const val = document.createElement("strong");
      key.textContent = label; val.textContent = String(value);
      row.append(key, val); node.appendChild(row);
    });
  };
  const renderTable = (selector, rows, columns) => {
    const node = systemMonitorDialog.querySelector(selector);
    if (!node) return;
    node.replaceChildren();
    if (!Array.isArray(rows) || rows.length === 0) {
      const empty = document.createElement("p");
      empty.className = "muted compact"; empty.textContent = "No data";
      node.appendChild(empty); return;
    }
    const table = document.createElement("table");
    const head = document.createElement("thead");
    const headRow = document.createElement("tr");
    columns.forEach((column) => { const th = document.createElement("th"); th.textContent = jvS(column.label); headRow.appendChild(th); });
    head.appendChild(headRow);
    const body = document.createElement("tbody");
    rows.forEach((row) => {
      const tr = document.createElement("tr");
      columns.forEach((column) => {
        const td = document.createElement("td");
        const value = column.render(row);
        td.textContent = value === undefined || value === null || value === "" ? "—" : String(value);
        tr.appendChild(td);
      });
      body.appendChild(tr);
    });
    table.append(head, body); node.appendChild(table);
  };
  const renderMonitor = (data) => {
    const system = data.system || {}, uptime = data.uptime || {}, cpu = data.cpu || {};
    const mem = data.mem || {}, swap = data.memswap || {}, load = data.load || {};
    setText("[data-monitor-system-host]", "JustVoxel");
    renderLines("[data-monitor-system-lines]", [
      [jvT("ui.2db533557011"), system.hostname],
      [jvT("ui.8a716159a685"), system.os_version],
      [jvT("ui.d63ab4711473"), typeof uptime === "string" ? uptime : uptime.value],
    ]);
    setText("[data-monitor-cpu-total]", percent(cpu.total));
    renderLines("[data-monitor-cpu-lines]", [[jvT("ui.7bff6181add2"), cpu.cpucore], [jvT("ui.b512d97e7cbf"), percent(cpu.user)], [jvT("ui.6725e7bbcd28"), percent(cpu.system)], ["I/O wait", percent(cpu.iowait)], [jvT("ui.ab0171ca0494"), percent(cpu.idle)]]);
    setText("[data-monitor-memory-main]", percent(mem.percent));
    renderLines("[data-monitor-memory-lines]", [[jvT("ui.ae7d8dfac9ff"), bytes(mem.used)], [jvT("ui.e674447337e8"), bytes(mem.available)], [jvT("ui.c9b3c38247f7"), bytes(mem.total)], [jvT("ui.6ec282d40a8a"), swap.total ? bytes(swap.used) + " / " + bytes(swap.total) : jvT("ui.dccafe55abe3")]]);
    setText("[data-monitor-load-main]", fixed(load.min1) + " / " + fixed(load.min5) + " / " + fixed(load.min15));
    renderLines("[data-monitor-load-lines]", [["1 minute", fixed(load.min1)], ["5 minutes", fixed(load.min5)], ["15 minutes", fixed(load.min15)]]);
    renderTable("[data-monitor-filesystem]", Array.isArray(data.fs) ? data.fs : [], [
      {label:jvT("ui.d5605821c44b"), render:(row)=>row.mnt_point || row.mountpoint || row.device_name},
      {label:jvT("ui.ae7d8dfac9ff"), render:(row)=>percent(row.percent)}, {label:jvT("ui.f411a1fb6275"), render:(row)=>bytes(row.free)}, {label:jvT("ui.1af851907331"), render:(row)=>bytes(row.size)}
    ]);
    renderTable("[data-monitor-diskio]", Array.isArray(data.diskio) ? data.diskio : [], [
      {label:jvT("ui.340e0cf3bfa8"), render:(row)=>row.disk_name || row.name}, {label:jvT("ui.9b9a8d05a7ec"), render:(row)=>rate(row.read_bytes_rate_per_sec)}, {label:jvT("ui.3f00927a7193"), render:(row)=>rate(row.write_bytes_rate_per_sec)}
    ]);
    renderTable("[data-monitor-network]", Array.isArray(data.network) ? data.network : [], [
      {label:jvT("ui.c26b3ed4ce70"), render:(row)=>row.interface_name || row.name}, {label:"RX", render:(row)=>rate(row.bytes_recv_rate_per_sec)}, {label:"TX", render:(row)=>rate(row.bytes_sent_rate_per_sec)}
    ]);
    const processLimit = [5, 10, 15].includes(Number(profile.process_count)) ? Number(profile.process_count) : 10;
    const processesCard = systemMonitorDialog.querySelector('[data-monitor-card="processes"]');
    if (processesCard) processesCard.dataset.processCount = String(processLimit);
    const processes = (Array.isArray(data.processlist) ? data.processlist : [])
      .slice().sort((a,b)=>Number(b.cpu_percent || 0)-Number(a.cpu_percent || 0));
    renderTable("[data-monitor-processes]", processes.slice(0, processLimit), [
      {label:jvT("ui.e083bd83e9d3"), render:(row)=>row.name}, {label:"CPU", render:(row)=>percent(row.cpu_percent)}, {label:jvT("ui.c3963aedaac6"), render:(row)=>percent(row.memory_percent)}
    ]);
    renderTable("[data-monitor-containers]", Array.isArray(data.containers) ? data.containers : [], [
      {label:jvT("ui.0c7823e8c759"), render:(row)=>row.name}, {label:jvT("ui.920e413c7d41"), render:(row)=>row.status},
      {label:"CPU", render:(row)=>percent(row.cpu_percent ?? (row.cpu && row.cpu.total))},
      {label:jvT("ui.c3963aedaac6"), render:(row)=>bytes(row.memory_usage ?? (row.memory && row.memory.usage))},
      {label:jvT("ui.674b0ed54bf7"), render:(row)=>bytes(row.memory_limit ?? (row.memory && row.memory.limit))}
    ]);
    renderTable("[data-monitor-sensors]", Array.isArray(data.sensors) ? data.sensors : [], [
      {label:jvT("ui.162cc64ca1e4"), render:(row)=>row.label || row.name},
      {label:jvT("ui.8e37953d23da"), render:(row)=>row.value === undefined || row.value === null ? "—" : String(row.value) + (row.unit || "")}
    ]);
    const alerts = (Array.isArray(data.alert) ? data.alert : []).filter((row) => row && (row.end === undefined || row.end === null || Number(row.end) < 0));
    const alertsCard = systemMonitorDialog.querySelector('[data-monitor-card="alerts"]');
    if (alertsCard) alertsCard.hidden = profile.alerts === false || alerts.length === 0;
    if (alerts.length > 0) {
      renderTable("[data-monitor-alerts]", alerts, [
        {label:jvT("ui.a3b50c476732"), render:(row)=>row.state},
        {label:jvT("ui.baaddf70fb5d"), render:(row)=>row.type},
        {label:jvT("ui.2f77668a9dfb"), render:(row)=>row.global_msg || row.desc || jvT("ui.c56c69f95ff5")}
      ]);
    }
    if (state) state.textContent = jvT("ui.9bc551fa1148");
    if (error) error.hidden = true;
  };
  let monitorRequest = 0;
  const refreshMonitor = async () => {
    const request = ++monitorRequest;
    try {
      const response = await fetch("/api/system-monitor", {method:"GET", credentials:"same-origin", headers:{Accept:"application/json"}, cache:"no-store"});
      if (response.status === 401) { window.location.assign("/login"); return; }
      if (!response.ok) throw new Error(jvT("ui.c0988d33f576"));
      const data = await response.json();
      if (request === monitorRequest) { lastMonitorData = data; renderMonitor(data); }
    } catch (_) {
      if (state) state.textContent = jvT("ui.1fb26c33bad7");
      if (error) { error.textContent = jvT("ui.9b6afc6eb4b3"); error.hidden = false; }
    }
  };
  const stopRefresh = () => { if (refreshTimer) window.clearInterval(refreshTimer); refreshTimer = null; };
  const startRefresh = async () => {
    if (profilePanel) profilePanel.hidden = true;
    await loadProfile();
    refreshMonitor();
    stopRefresh();
    refreshTimer = window.setInterval(refreshMonitor, 2000);
  };

  const workspaceWindow = setupWorkspaceWindow(systemMonitorDialog, {
    onOpen: startRefresh,
    onClose: stopRefresh,
  });

  systemMonitorOpen.addEventListener("click", () => {
    if (controlCenter) controlCenter.open = false;
    workspaceWindow?.open();
  });
  if (closeButton) closeButton.addEventListener("click", () => workspaceWindow?.close());

  if (profileToggle && profilePanel) profileToggle.addEventListener("click", () => {
    profilePanel.hidden = !profilePanel.hidden;
    if (profileStatus) profileStatus.textContent = "";
  });
  document.addEventListener("pointerdown", (event) => {
    if (profilePanel?.hidden || !systemMonitorDialog.open) return;
    if (profilePanel.contains(event.target) || profileToggle?.contains(event.target)) return;
    profilePanel.hidden = true;
  });
  profileInputs.forEach((input) => input.addEventListener("change", () => {
    profile[input.dataset.monitorProfile] = input.checked;
    syncProfile();
    if (profileStatus) profileStatus.textContent = jvT("ui.22b3467c1ba9");
  }));
  if (processCount) processCount.addEventListener("change", () => {
    profile.process_count = Number(processCount.value);
    if (lastMonitorData) renderMonitor(lastMonitorData);
    if (profileStatus) profileStatus.textContent = jvT("ui.22b3467c1ba9");
  });
  if (resetButton) resetButton.addEventListener("click", () => {
    profile = { ...defaultProfile };
    syncProfile();
    refreshMonitor();
    if (profileStatus) profileStatus.textContent = jvT("ui.22b3467c1ba9");
  });
  if (saveButton) saveButton.addEventListener("click", saveProfile);

  const monitorRestoreRoleAllowed = () => !document.body.classList.contains("role-pending") &&
    ["role-administrator", "role-operator", "role-viewer"].some((role) => document.body.classList.contains(role));
  const restoreWhenRoleKnown = () => {
    if (!readWorkspaceWindowState("system-monitor").open) return;
    if (monitorRestoreRoleAllowed()) {
      workspaceWindow?.open();
      return;
    }
    if (!document.body.classList.contains("role-pending")) return;
    const observer = new MutationObserver(() => {
      if (document.body.classList.contains("role-pending")) return;
      observer.disconnect();
      if (monitorRestoreRoleAllowed()) workspaceWindow?.open();
    });
    observer.observe(document.body, { attributes: true, attributeFilter: ["class"] });
  };
  restoreWhenRoleKnown();
}


const minecraftOpen = document.querySelector("[data-minecraft-open]");
globalThis.justVoxelOpenDeleteServer = ({name, description, csrf, request, isCurrent, dismissOnDeleted = false, onError = () => {}, onDeleted = result => window.location.assign(result.redirect)}) => {
    const selection = globalThis.justVoxelSelectionGeneration.minecraft;
    const dialog = document.createElement("dialog");
    dialog.className = "minecraft-delete-dialog";
    dialog.setAttribute("aria-labelledby", "minecraft-delete-title");
    dialog.innerHTML = `
      <form class="minecraft-delete-card">
        <div class="minecraft-delete-heading"><div><p class="eyebrow">${jvH("ui.3744cc428389")}</p><h2 id="minecraft-delete-title">${jvH("ui.7e9464a23e43")}</h2></div><button type="button" class="secondary" data-delete-close aria-label="${jvH("ui.7d9eb7acb13e")}">×</button></div>
        <p data-delete-description></p>
        <div class="minecraft-delete-review"><dl><div><dt>${jvH("ui.aef7de28d529")}</dt><dd data-delete-name></dd></div><div><dt>${jvH("ui.481e2c0e53b9")}</dt><dd>Deleted</dd></div><div><dt>${jvH("ui.3334fee2c77f")}</dt><dd>Preserved/history</dd></div></dl></div>
        <div class="destructive-confirmation" data-destructive-confirmation data-destructive-idle="Slide to confirm destructive action">
          <p class="minecraft-delete-selected">${jvH("ui.b3bee314c6c2")}<strong data-delete-selected></strong></p>
          <div class="destructive-confirm-slider" data-destructive-slider-shell>
            <span class="destructive-confirm-slider-text" data-destructive-slider-text>${jvH("ui.7f9334299170")}</span>
            <span class="destructive-confirm-slider-thumb" aria-hidden="true">›</span>
            <input type="range" min="0" max="100" value="0" data-destructive-slider aria-label="${jvH("ui.7f9334299170")}">
          </div>
          <label class="destructive-confirm-toggle-row" data-destructive-toggle-row hidden><span class="destructive-confirm-toggle-callout">${jvH("ui.eebdd24a77d9")}</span><input type="checkbox" class="destructive-confirm-toggle" data-destructive-toggle aria-label="${jvH("ui.e5d40b67ccca")}"></label>
        </div>
        <p class="notice error" data-delete-error role="status" hidden></p>
        <div class="action-row"><button type="button" class="secondary" data-delete-cancel>${jvH("ui.19766ed6ccb2")}</button><button type="submit" class="danger" data-destructive-submit disabled>${jvH("ui.a6e53d6b90d2")}</button></div>
      </form>`;
    dialog.querySelector("[data-delete-description]").textContent = description;
    dialog.querySelector("[data-delete-name]").textContent = name;
    dialog.querySelector("[data-delete-selected]").textContent = `${name} will be deleted`;
    const form = dialog.querySelector("form");
    const apply = dialog.querySelector("[data-destructive-submit]");
    const slider = dialog.querySelector("[data-destructive-slider]");
    const toggle = dialog.querySelector("[data-destructive-toggle]");
    const feedback = dialog.querySelector("[data-delete-error]");
    let submitting = false;
    const cancel = () => { if (!submitting) dialog.close(); };
    const detach = (event) => { if (event.detail?.workspace === "minecraft") dialog.close(); };
    dialog.querySelector("[data-delete-close]").addEventListener("click", cancel);
    dialog.querySelector("[data-delete-cancel]").addEventListener("click", cancel);
    dialog.addEventListener("cancel", (event) => { event.preventDefault(); cancel(); });
    dialog.addEventListener("close", () => {
      window.removeEventListener("justvoxel:server-selection-changed", detach);
      dialog.remove();
    });
    window.addEventListener("justvoxel:server-selection-changed", detach);
    form.addEventListener("submit", async (event) => {
      event.preventDefault();
      if (submitting || Number(slider.value) < 100 || !toggle.checked) return;
      if (selection !== globalThis.justVoxelSelectionGeneration.minecraft || !isCurrent()) { dialog.close(); return; }
      submitting = true;
      apply.disabled = true;
      slider.disabled = true;
      toggle.disabled = true;
      try {
        const result = await request("/api/minecraft/workspace/delete", {
          method: "POST", body: new URLSearchParams({csrf, confirm_name: name}),
        });
        if (result) {
          await onDeleted(result);
          if (dismissOnDeleted) dialog.close();
        }
      } catch (error) {
        if (error.staleSelection || selection !== globalThis.justVoxelSelectionGeneration.minecraft) return;
        feedback.textContent = error.message;
        feedback.hidden = false;
        onError(error);
      } finally {
        submitting = false;
        slider.disabled = false;
        toggle.disabled = false;
        dialog.querySelector("[data-destructive-confirmation]")._justVoxelDestructiveSync?.();
      }
    });
    document.body.appendChild(dialog);
    dialog.showModal();
  };


const minecraftDialog = document.querySelector("[data-minecraft-workspace-dialog]");
// RCON state lives only in this page's memory and is replaced on server selection.
function newMinecraftRconState() {
  return {history: [], cursor: 0, draft: "", transcript: [], executed: false, transcriptCleared: false, commands: null, partial: false, busy: false};
}
// Keep discovery evidence intact; only this derived catalog is cleaned for display.
function minecraftCommandCatalog(commands) {
  const entries = commands.map(entry => {
    const description = (entry.description || "").trim();
    return {...entry, description: /^a mojang provided command\.$/i.test(description) ? "" : description};
  }).filter(entry => {
    const name = entry.name.toLowerCase();
    const description = entry.description.toLowerCase();
    const indexTopic = (name === "aliases" && description === "lists command aliases") || description === `all commands for ${name}`;
    return !indexTopic && name !== "stop" && name !== "minecraft:stop";
  });
  const bareNames = new Set(entries.filter(entry => !entry.name.includes(":")).map(entry => entry.name.toLowerCase()));
  const pluginNames = new Set();
  for (const entry of entries) {
    const [namespace, name] = entry.name.toLowerCase().split(":");
    if (name && !["minecraft", "bukkit"].includes(namespace)) pluginNames.add(name);
  }
  return entries.filter(entry => {
    const name = entry.name.toLowerCase();
    return !name.includes(":") || !bareNames.has(name.slice(name.indexOf(":") + 1));
  }).map(entry => ({...entry, group: minecraftCommandGroup(entry.name, pluginNames)}));
}
function minecraftCommandGroup(name, pluginNames) {
  const normalized = name.toLowerCase();
  const [namespace, localName] = normalized.split(":");
  const command = localName || normalized;
  // Keep unique plugin commands out of the core player/gameplay groups.
  const pluginNamespace = localName && !["minecraft", "bukkit"].includes(namespace);
  if (!pluginNamespace) {
    if (["list", "whitelist", "op", "deop", "ban", "ban-ip", "banlist", "pardon", "pardon-ip", "kick", "msg", "tell", "w", "say", "team", "teammsg"].includes(command)) return jvT("ui.ec7d651bf756");
    if (["weather", "time", "gamemode", "gamerule", "difficulty", "give", "tp", "teleport", "seed", "summon", "effect", "enchant", "experience", "xp", "setblock", "fill", "fillbiome", "worldborder", "setworldspawn", "spawnpoint", "locate", "loot", "recipe", "advancement", "forceload", "place", "playsound", "stopsound", "particle", "title", "bossbar", "attribute", "damage", "ride", "spreadplayers"].includes(command)) return jvT("ui.fabc90d29be4");
  }
  if (["help", "version", "ver", "about", "plugins", "pl", "reload", "save-all", "save-on", "save-off", "datapack", "tps", "mspt", "timings", "ping", "uptime", "ram", "perf", "jfr", "debug", "restart"].includes(command)) return jvT("ui.d45e70e92c95");
  if (pluginNamespace || pluginNames.has(command)) return jvT("ui.9514b7ff4860");
  return jvT("ui.b4fbc52a5a15");
}
function createMinecraftRconView({model, running, request, isCurrent}) {
  const node = (tag, text, className = "") => {
    const el = document.createElement(tag);
    el.className = className;
    if (text !== undefined) el.textContent = text;
    return el;
  };
  const root = node("section", undefined, "minecraft-rcon");
  const toolbar = node("div", undefined, "minecraft-rcon-toolbar");
  const consoleButton = node("button", jvT("ui.29a40861bafe"), "secondary");
  const commandsButton = node("button", jvT("ui.b269dc4e81a5"), "secondary");
  consoleButton.type = commandsButton.type = "button";
  toolbar.append(consoleButton, commandsButton);
  const status = node("p", running ? "" : jvT("ui.a58f91e13269"), "muted compact");
  status.setAttribute("role", "status");
  const upper = node("div", undefined, "minecraft-rcon-upper");
  const transcript = node("pre", undefined, "minecraft-rcon-transcript");
  transcript.setAttribute("aria-label", jvT("ui.2ab40dd11bcd"));
  const catalog = node("div", undefined, "minecraft-rcon-commands");
  const controls = node("div", undefined, "minecraft-rcon-toolbar");
  const search = node("input");
  search.type = "search"; search.placeholder = jvT("ui.00ff7398c308");
  search.setAttribute("aria-label", jvT("ui.00ff7398c308"));
  const refresh = node("button", jvT("ui.a3340efc9959"), "secondary");
  refresh.type = "button";
  const discoveryStatus = node("p", "", "muted compact");
  const list = node("div", undefined, "minecraft-rcon-command-list");
  controls.append(search, refresh);
  catalog.append(controls, discoveryStatus, list);
  upper.append(transcript, catalog);
  const form = node("form", undefined, "minecraft-rcon-lane");
  const input = node("input");
  input.type = "text"; input.maxLength = 1024; input.autocomplete = "off";
  input.spellcheck = false; input.placeholder = jvT("ui.36ef4b40d05a");
  input.setAttribute("aria-label", jvT("ui.d7511e08ef72"));
  const clear = node("button", jvT("ui.83b12c2216ef"), "secondary"); clear.type = "button";
  const send = node("button", jvT("ui.f6f4688ff23d")); send.type = "submit";
  form.append(input, clear, send);
  root.append(toolbar, status, upper, form);
  let view = "console", discovering = false;
  const updateDisabled = () => {
    input.disabled = !running || model.busy;
    send.disabled = !running || model.busy || discovering;
    refresh.disabled = !running || discovering || model.busy;
  };
  const paintTranscript = () => {
    transcript.classList.toggle("muted", !model.executed);
    transcript.textContent = model.executed || model.transcriptCleared ? model.transcript.join("\n\n") :
      jvT("ui.bd93ed9b920c") + "\n\n# " + jvT("ui.e759ae396ffa") + "\nlist" + "\n\n# " + jvT("ui.781370a8530e") + "\nwhitelist list" + "\n\n# " + jvT("ui.9b6dee597c5c") + "\nsay Hello everyone!" + "\n\n# " + jvT("ui.868dbc92cb93") + "\nweather clear" + "\n\n# " + jvT("ui.d365494c6640") + "\nversion";
    transcript.scrollTop = transcript.scrollHeight;
  };
  const clearTranscript = () => {
    model.transcript = [];
    model.transcriptCleared = true;
    paintTranscript();
  };
  clear.addEventListener("click", clearTranscript);
  model.onChange = () => { updateDisabled(); paintTranscript(); };
  const show = (next) => {
    view = next;
    transcript.hidden = next !== "console";
    catalog.hidden = next !== "commands";
    consoleButton.setAttribute("aria-pressed", String(next === "console"));
    commandsButton.setAttribute("aria-pressed", String(next === "commands"));
    if (next === "console") paintTranscript();
  };
  const groupOpen = new Map();
  let searching = false;
  const paintCommands = () => {
    list.replaceChildren();
    const needle = search.value.trim().toLowerCase();
    if (searching && !needle) groupOpen.clear();
    searching = !!needle;
    const entries = minecraftCommandCatalog(model.commands || []).filter(entry => `${entry.name} ${entry.usage || ""} ${entry.description}`.toLowerCase().includes(needle));
    for (const group of [jvT("ui.ec7d651bf756"), jvT("ui.fabc90d29be4"), jvT("ui.d45e70e92c95"), jvT("ui.9514b7ff4860"), jvT("ui.b4fbc52a5a15")]) {
      const members = entries.filter(entry => entry.group === group);
      if (!members.length) continue;
      const section = node("details", undefined, "minecraft-rcon-group");
      section.open = searching || (groupOpen.get(group) ?? group === jvT("ui.ec7d651bf756"));
      section.addEventListener("toggle", () => {
        if (!searching && Array.from(list.children).includes(section)) groupOpen.set(group, section.open);
      });
      const summary = node("summary", group);
      summary.append(node("span", ` (${members.length})`, "minecraft-rcon-group-count"));
      section.append(summary);
      list.append(section);
      for (const entry of members) {
        const row = node("div", undefined, "minecraft-rcon-command");
        const choose = node("button", entry.name, "secondary"); choose.type = "button";
        choose.addEventListener("click", () => {
          if (!isCurrent()) return;
          // Place only the discovered name in the lane; placeholders are explanatory.
          input.value = entry.name;
          input.focus();
        });
        const explanation = [entry.description, entry.usage].filter(Boolean).join(" · ");
        row.append(choose, node("span", explanation, "muted"));
        section.append(row);
      }
    }
    if (!entries.length) list.append(node("p", model.commands?.length ? jvT("ui.4ea824450aec") : jvT("ui.86f0defdf40b"), "muted"));
  };
  const discover = async () => {
    if (!running || discovering || model.busy || !isCurrent()) return;
    discovering = true; updateDisabled(); discoveryStatus.textContent = jvT("ui.c5970e1e83fb");
    try {
      const result = await request({discover: "yes"});
      if (!isCurrent()) return;
      model.commands = result.commands || []; model.partial = !!result.partial;
      discoveryStatus.textContent = model.partial ? jvT("ui.38d4ea385790") : jvT("ui.1282c228fe9f");
      paintCommands();
    } catch (error) {
      if (isCurrent() && !error.staleSelection) { discoveryStatus.textContent = error.message; paintCommands(); }
    } finally { discovering = false; updateDisabled(); }
  };
  consoleButton.addEventListener("click", () => show("console"));
  commandsButton.addEventListener("click", () => {
    show("commands");
    if (model.commands === null && running) void discover();
    else {
      discoveryStatus.textContent = model.partial ? jvT("ui.38d4ea385790") : model.commands === null ? "" : jvT("ui.1282c228fe9f");
      paintCommands();
    }
  });
  search.addEventListener("input", paintCommands);
  refresh.addEventListener("click", discover);
  form.addEventListener("submit", async event => {
    event.preventDefault();
    if (!running || model.busy || discovering || !isCurrent() || !input.value.trim()) return;
    const command = input.value.trim();
    show("console"); model.busy = true; updateDisabled(); status.textContent = jvT("ui.b8ed5279e897");
    model.history.push(command);
    if (model.history.length > 100) model.history.shift();
    model.cursor = model.history.length; model.draft = ""; input.value = "";
    try {
      const result = await request({command});
      if (!isCurrent()) return;
      model.executed = true;
      model.transcript.push(`> ${command}\n${result.output || "Command completed."}`);
      // Keep browser memory bounded even during a long session.
      while (model.transcript.length > 100 || model.transcript.join("\n\n").length > 65536) model.transcript.shift();
      paintTranscript(); status.textContent = "";
    } catch (error) { if (isCurrent() && !error.staleSelection) status.textContent = error.message; }
    finally {
      model.busy = false; model.onChange?.();
      if (isCurrent()) input.focus();
    }
  });
  input.addEventListener("keydown", event => {
    if (event.key === "Enter" && !event.isComposing) { event.preventDefault(); form.requestSubmit(); return; }
    if ((event.key === "ArrowUp" || event.key === "ArrowDown") && model.history.length) {
      event.preventDefault();
      if (model.cursor === model.history.length) model.draft = input.value;
      model.cursor = Math.max(0, Math.min(model.history.length, model.cursor + (event.key === "ArrowUp" ? -1 : 1)));
      input.value = model.cursor === model.history.length ? model.draft : model.history[model.cursor];
    }
  });
  root.addEventListener("keydown", event => {
    if (event.ctrlKey && event.key.toLowerCase() === "l") {
      event.preventDefault(); clearTranscript();
    }
  });
  paintTranscript(); show(view); updateDisabled();
  return root;
}

if (minecraftOpen && minecraftDialog) {
  const closeButton = minecraftDialog.querySelector("[data-minecraft-close]");
  const refreshButton = minecraftDialog.querySelector("[data-minecraft-refresh]");
  const state = minecraftDialog.querySelector("[data-minecraft-state]");
  const content = minecraftDialog.querySelector("[data-minecraft-workspace-content]");
  const tabs = Array.from(minecraftDialog.querySelectorAll("[data-minecraft-tab]"));
  const csrf = document.querySelector("[data-minecraft-workspace-csrf]")?.value || "";
  const settingsTabs = new Set(["memory", "gameplay", "crossplay"]);
  globalThis.justVoxelServerSelection(minecraftDialog.querySelector("[data-server-selector-slot]")).catch((error)=>{if(state)state.textContent=error.message;});
  let currentTab = "overview";
  let loadSequence = 0;
  let rconModel = newMinecraftRconState();

  const messageNode = (value, className = "muted compact") => {
    const node = document.createElement("p");
    node.className = className;
    node.textContent = value;
    return node;
  };

  const noticeNode = (value, kind = "success") => {
    const node = document.createElement("div");
    node.className = "notice " + kind;
    node.textContent = value;
    return node;
  };

  const requestWorkspaceJSON = async (url, options = {}, pluginRecovery = null) => {
    if (url === "/api/dashboard-status") url = "/api/minecraft/workspace/snapshot";
    const selection = globalThis.justVoxelSelectionGeneration.minecraft;
    const boundURL = options.method && options.method !== "GET" ? globalThis.justVoxelSelectedURL(url, "minecraft") : null;
    if (globalThis.justVoxelSelectionReady) await globalThis.justVoxelSelectionReady;
    if (selection !== globalThis.justVoxelSelectionGeneration.minecraft) throw globalThis.justVoxelStaleSelection();
    const response = await fetch(boundURL || globalThis.justVoxelSelectedURL(url, "minecraft"), {
      credentials: "same-origin",
      cache: "no-store",
      ...options,
      headers: {
        Accept: "application/json",
        ...(options.headers || {}),
      },
    });
    let payload = {};
    try {
      payload = await response.json();
    } catch (_) {
      payload = {};
    }
    if (selection !== globalThis.justVoxelSelectionGeneration.minecraft) throw globalThis.justVoxelStaleSelection();
    if (response.status === 401) {
      window.location.assign("/login");
      return null;
    }
    if (response.status === 403 && String(jvC(payload.error || "")).toLowerCase().includes("password")) {
      window.location.assign("/password");
      return null;
    }
    if (pluginRecovery && globalThis.justVoxelPluginRuntimeRecovery.matches(response, payload, pluginRecovery.id, pluginRecovery.operationID)) {
      return {recovery: payload};
    }
    if (!response.ok) {
      throw new Error(payload.error || (pluginRecovery && payload.message) || jvT("ui.4d5161e70e99"));
    }
    return payload;
  };

  const openDeleteDialog = (name, description, sequence) => globalThis.justVoxelOpenDeleteServer({
    name, description, csrf, request: requestWorkspaceJSON,
    isCurrent: () => sequence === loadSequence,
    onError: error => { if (state) state.textContent = error.message; },
  });

  const renderOverview = async (sequence, message = "") => {
    const selection = globalThis.justVoxelSelectionGeneration.minecraft;
    const snapshot = await requestWorkspaceJSON("/api/dashboard-status");
    if (!snapshot || sequence !== loadSequence || !content) return;

    const minecraft = snapshot?.status?.minecraft || {};
    const players = snapshot?.players || {};
    const root = document.createElement("div");
    root.className = "minecraft-overview-compact";
    if (message) root.appendChild(noticeNode(message));

    const grid = document.createElement("div");
    grid.className = "minecraft-overview-grid";
    [
      [jvT("ui.920e413c7d41"), minecraft.state || "—"],
      [jvT("ui.dd167905de0d"), minecraft.version || "—"],
      [jvT("ui.84e12ac655dd"), minecraft.configured ? (players.max ?? minecraft.max_players ?? "—") : jvT("ui.dd1841d29502")],
      [jvT("ui.0d21bd52022c"), minecraft.configured ? (players.online ?? 0) : "—"],
    ].forEach(([label, value]) => {
      const card = document.createElement("article");
      card.className = "minecraft-overview-card";
      const labelNode = document.createElement("span");
      labelNode.textContent = label;
      const valueNode = document.createElement("strong");
      valueNode.textContent = value;
      card.append(labelNode, valueNode);
      grid.appendChild(card);
    });
    root.appendChild(grid);
    const playerLine = document.createElement("div");
    playerLine.className = "minecraft-overview-playerline";
    const playerLabel = document.createElement("strong");
    playerLabel.className = "minecraft-summary-label";
    playerLabel.textContent = jvT("ui.0d21bd52022c");
    playerLine.appendChild(playerLabel);
    if (!minecraft.configured || players.state === "not_configured") playerLine.appendChild(messageNode(jvT("ui.bd7bbac1db61")));
    else if (players.state === "stopped") playerLine.appendChild(messageNode(jvT("ui.04333b2f80ea")));
    else if (players.state === "unavailable") playerLine.appendChild(messageNode(players.error || jvT("ui.7dfac59b23ae")));
    else if ((players.online ?? 0) === 0) playerLine.appendChild(messageNode(jvT("ui.c705f28891e8")));
    else if (Array.isArray(players.names) && players.names.length > 0) {
      const list = document.createElement("div");
      list.className = "player-list";
      players.names.forEach((name) => {
        const chip = document.createElement("span");
        chip.className = "player-chip";
        chip.textContent = name;
        list.appendChild(chip);
      });
      playerLine.appendChild(list);
    } else {
      playerLine.appendChild(messageNode(`${players.online ?? 0} players online.`));
    }
    root.appendChild(playerLine);

    if (document.body.classList.contains("role-administrator")) {
      const [context, payload] = await Promise.all([
        requestWorkspaceJSON("/api/minecraft/workspace/selection"),
        requestWorkspaceJSON("/api/minecraft/workspace/settings"),
      ]);
      if (!context || !payload || sequence !== loadSequence) return;
      const name = context.selected.name;
      const settings = payload.minecraft || {};
      const lane = document.createElement("div");
      lane.className = "minecraft-overview-lifecycle";
      lane.innerHTML = `<label class="minecraft-overview-boot-card"><span>${jvH("ui.06b0ab106202")}</span><input class="date-time-switch" type="checkbox" role="switch" name="server_running"></label><label class="minecraft-overview-boot-card"><span>${jvH("ui.7f369a5b5b05")}</span><input class="date-time-switch" type="checkbox" role="switch" name="start_on_boot"></label>`;
      const running = lane.querySelector('[name="server_running"]');
      const boot = lane.querySelector('[name="start_on_boot"]');
      const runtimeState = String(minecraft.state || "").toLowerCase();
      running.checked = runtimeState === "running";
      running.disabled = !minecraft.configured || !["running", "stopped"].includes(runtimeState);
      boot.checked = Boolean(settings.network_member || settings.start_on_boot);
      if (settings.network_member) boot.closest('label').querySelector('span').textContent = jvT("ui.be81bb8c89a0");
      boot.disabled = Boolean(settings.network_member) || !payload.configured || !["running", "stopped"].includes(runtimeState);
      const selectedID = context.selected.id;
      const selectedURL = path => path + (path.includes("?") ? "&" : "?") + new URLSearchParams({server: selectedID});
      const selectedRequest = (path, body, operationID = null) => requestWorkspaceJSON(
        selectedURL(path), {method: "POST", body}, operationID === null ? null : {id: selectedID, operationID}
      );
      const isCurrent = expectedSequence => currentTab === "overview" && expectedSequence === loadSequence &&
        selection === globalThis.justVoxelSelectionGeneration.minecraft;
      const requiresCondition = plan => plan.compatibility_blocked || Object.entries(plan).some(([key, value]) => key.endsWith("_required") && value === true);
      const refusalReason = plan => plan.error || plan.message || (plan.warnings || []).join(" ") || jvT("ui.1952254b901b");
      let pending = false;
      const change = async (control, action) => {
        const requested = control.checked;
        control.checked = !requested;
        if (pending || control.disabled || !isCurrent(sequence)) return;
        pending = true;
        running.disabled = boot.disabled = true;
        if (state) state.textContent = jvT("ui.3329a9bb48b9");
        let failure, recovery;
        try { recovery = await action(requested); }
        catch (error) {
          if (error.staleSelection || selection !== globalThis.justVoxelSelectionGeneration.minecraft) return;
          failure = error;
        }
        if (!isCurrent(sequence)) return;
        // Both success and refusal reload authoritative values.
        const reloadSequence = await loadCurrentTab();
        if (!isCurrent(reloadSequence)) return;
        if (failure && state) state.textContent = failure.message;
        if (recovery) globalThis.justVoxelPluginRuntimeRecovery.open({
          recovery, id: selectedID, csrf, selected: selectedURL,
          request: (path, {body}, operationID) => selectedRequest(path, body, operationID),
          isCurrent: () => isCurrent(reloadSequence),
          refresh: () => loadCurrentTab(),
        });
      };
      running.addEventListener("change", () => change(running, async requested => {
        const result = await selectedRequest("/api/minecraft/servers/lifecycle", new URLSearchParams({csrf, action: requested ? "start" : "stop"}), "");
        if (result?.recovery) return result.recovery;
        if (result && !result.ok) throw new Error(result.error || jvT("ui.bd44764996e3"));
      }));
      boot.addEventListener("change", () => change(boot, async requested => {
        const params = new URLSearchParams({csrf, tab: "overview", start_on_boot: String(requested)});
        const plan = await selectedRequest("/api/minecraft/workspace/settings/plan", params);
        if (!plan || sequence !== loadSequence) return;
        if (requiresCondition(plan)) throw new Error(refusalReason(plan));
        if (!plan.review) throw new Error(jvT("ui.9f99dea1f3c0"));
        params.set("review", plan.review);
        const result = await selectedRequest("/api/minecraft/workspace/settings/apply", params);
        if (result && !result.applied) throw new Error(refusalReason(result));
      }));
      root.appendChild(lane);

      const details = document.createElement("form");
      details.className = "panel details minecraft-workspace-section minecraft-overview-details";
      details.innerHTML = `<div class="minecraft-native-grid"><label>${jvH("ui.21babfed6c38")}<input name="display_name" required maxlength="64"></label><label>${jvH("ui.f6f45fa586e5")}<input name="motd" required></label></div><div class="minecraft-native-actions"><button type="submit">${jvH("ui.1509f561f241")}</button></div>`;
      details.elements.display_name.value = name;
      details.elements.motd.value = settings.motd || "";
      details.addEventListener("submit", async (event) => {
        const selection = globalThis.justVoxelSelectionGeneration.minecraft;
        event.preventDefault();
        const save = details.querySelector("button");
        const proposedName = details.elements.display_name.value;
        const motd = details.elements.motd.value.trim();
        if (!motd) { if (state) state.textContent = jvT("ui.0e596941556e"); return; }
        save.disabled = true;
        const reloadAfterError = async (error) => {
          if (error.staleSelection || selection !== globalThis.justVoxelSelectionGeneration.minecraft) return;
          // Either operation may already have succeeded. Always show authoritative state.
          let remainingError = error.message;
          try {
            const authoritative = await requestWorkspaceJSON("/api/minecraft/workspace/selection");
            if (authoritative) document.querySelectorAll("[data-selected-server] select option").forEach((option) => {
              const server = authoritative.servers.find((entry) => entry.id === option.value);
              if (server) option.textContent = server.name;
            });
          } catch (refreshError) {
            if (refreshError.staleSelection || selection !== globalThis.justVoxelSelectionGeneration.minecraft) return;
            remainingError += " " + refreshError.message;
          }
          await loadCurrentTab();
          if (selection === globalThis.justVoxelSelectionGeneration.minecraft && state) state.textContent = remainingError;
        };
        const finishSave = async () => {
          if (selection !== globalThis.justVoxelSelectionGeneration.minecraft || sequence !== loadSequence) throw globalThis.justVoxelStaleSelection();
          if (proposedName !== name) {
            await requestWorkspaceJSON("/api/minecraft/workspace/rename", {
              method: "POST", body: new URLSearchParams({csrf, display_name: proposedName}),
            });
          }
          window.location.reload();
        };
        try {
          const current = await requestWorkspaceJSON("/api/minecraft/workspace/settings");
          if (!current || sequence !== loadSequence) return;
          if (motd !== current.minecraft.motd) {
            const params = new URLSearchParams({csrf, tab: "gameplay", motd,
              max_players: String(current.minecraft.max_players), game_mode: current.minecraft.game_mode});
            const plan = await requestWorkspaceJSON("/api/minecraft/workspace/settings/plan", {method: "POST", body: params});
            if (!plan || sequence !== loadSequence) return;
            const reviewOptions = {form: details, onApplied: finishSave, onError: reloadAfterError};
            if (plan.confirmation_required || plan.restart_required || plan.memory_restart_required) {
              renderSettingsReview(root, plan, params, reviewOptions);
              return;
            }
            if (plan.review) params.set("review", plan.review);
            const result = await requestWorkspaceJSON("/api/minecraft/workspace/settings/apply", {method: "POST", body: params});
            if (!result) return;
            if (result.confirmation_required && !result.applied) {
              renderSettingsReview(root, result, params, reviewOptions);
              return;
            }
            if (!result.applied) throw new Error(result.error || jvT("ui.59f7401b3446"));
          }
          await finishSave();
        } catch (error) {
          await reloadAfterError(error);
        } finally {
          save.disabled = false;
        }
      });
      root.appendChild(details);

      const danger = document.createElement("section");
      danger.className = "panel details minecraft-workspace-section minecraft-overview-delete";
      const title = document.createElement("h3");
      title.textContent = jvT("ui.3468c78743f0", {v0: (name)});
      const description = jvT("ui.e9d6a58ae67a");
      const actions = document.createElement("div");
      actions.className = "minecraft-native-actions";
      const remove = document.createElement("button");
      remove.type = "button";
      remove.className = "danger";
      remove.textContent = jvT("ui.056e6a5f6594");
      remove.addEventListener("click", () => openDeleteDialog(name, description, sequence));
      actions.appendChild(remove);
      danger.append(title, messageNode(description), actions);
      root.appendChild(danger);
    }

    content.replaceChildren(root);
  };

  const formatRemainingMemory = (mib) => {
    if (mib >= 1024) return `${(mib / 1024).toFixed(mib % 1024 === 0 ? 0 : 1)} GB`;
    return `${Math.max(0, Math.round(mib))} MB`;
  };

  const initNativeMemoryControls = (root, defaults) => {
    const gameMemory = root.querySelector('[name="java_memory"]');
    const maxMemory = root.querySelector('[name="container_memory"]');
    const maxPlayers = Number(root.dataset.maxPlayers || 10);
    const memoryStatus = root.querySelector("[data-minecraft-memory-status]");
    const buttons = Array.from(root.querySelectorAll("[data-minecraft-memory-preset]"));
    const totalMiB = Number(defaults?.system_memory_mib || 0);
    const minimumReserveMiB = Number(defaults?.system_reserve_minimum_mib || 1024);
    const recommendedReserveMiB = Number(defaults?.system_reserve_recommended_mib || 1536);
    let activePreset = "";

    const parseMemoryMiB = (value) => {
      const match = String(value || "").trim().match(/^([1-9][0-9]*)([mMgG])$/);
      if (!match) return 0;
      const amount = Number(match[1]);
      return match[2].toUpperCase() === "G" ? amount * 1024 : amount;
    };
    const memoryValue = (mib) => mib % 1024 === 0 ? `${mib / 1024}G` : `${mib}M`;
    const markPreset = (name) => {
      activePreset = name;
      buttons.forEach((button) => button.classList.toggle("is-selected", button.dataset.minecraftMemoryPreset === name));
    };
    const updateStatus = () => {
      if (!memoryStatus || !maxMemory) return;
      const maximumMiB = parseMemoryMiB(maxMemory.value);
      memoryStatus.classList.remove("warning", "danger");
      if (!totalMiB) {
        memoryStatus.textContent = jvT("ui.51da904725b5");
        memoryStatus.classList.add("warning");
        return;
      }
      if (!maximumMiB) {
        memoryStatus.textContent = jvT("ui.e9efdcb72877");
        memoryStatus.classList.add("warning");
        return;
      }
      const remaining = totalMiB - maximumMiB;
      if (remaining < minimumReserveMiB) {
        memoryStatus.textContent = jvT("ui.711b8607d515", {v0: (formatRemainingMemory(remaining))});
        memoryStatus.classList.add("danger");
      } else if (remaining < recommendedReserveMiB) {
        memoryStatus.textContent = jvT("ui.fc59932c53bf", {v0: (formatRemainingMemory(remaining))});
        memoryStatus.classList.add("warning");
      } else {
        memoryStatus.textContent = jvT("ui.e2fd4a3064c9", {v0: (formatRemainingMemory(remaining))});
      }
    };
    const applyPreset = (name) => {
      if (name === "custom") {
        markPreset("custom");
        gameMemory?.focus();
        updateStatus();
        return;
      }
      if (!gameMemory || !maxMemory) return;
      // Recommended tiers are capped by actual MemTotal.
      const totalGiB = totalMiB > 0 ? Math.round(totalMiB / 1024) : 8;
      const baseHeap = totalGiB >= 16 ? 6 : totalGiB >= 8 ? 4 : totalGiB >= 6 ? 3 : totalGiB >= 4 ? 2 : 1;
      const baseMaximum = totalGiB >= 16 ? 8 : totalGiB >= 8 ? 6 : totalGiB >= 6 ? 4 : totalGiB >= 4 ? 3 : 2;
      let heapGiB = name === "light" ? Math.max(2, baseHeap - 2) : name === "high" ? Math.min(14, baseHeap + 2) : baseHeap;
      let heapMiB = heapGiB * 1024;
      let maximumMiB = (name === "recommended" ? baseMaximum : heapGiB + (heapGiB >= 4 ? 2 : 1)) * 1024;
      if (totalMiB > 0) {
        const allowed = Math.floor(totalMiB - Math.max(1024, minimumReserveMiB));
        if (allowed < 1024) return;
        const limit = name === "high" && allowed >= 2048 ? Math.floor(allowed / 1024) * 1024 : allowed;
        maximumMiB = Math.min(maximumMiB, limit);
        if (heapMiB > maximumMiB - 512) heapMiB = maximumMiB >= 2048 ? maximumMiB - 1024 : Math.floor(maximumMiB / 2);
      }
      gameMemory.value = memoryValue(heapMiB);
      maxMemory.value = memoryValue(maximumMiB);
      markPreset(name);
      updateStatus();
    };

    buttons.forEach((button) => button.addEventListener("click", () => applyPreset(button.dataset.minecraftMemoryPreset)));
    [gameMemory, maxMemory].forEach((input) => input?.addEventListener("input", () => {
      if (activePreset && activePreset !== "custom") markPreset("custom");
      updateStatus();
    }));
    updateStatus();
  };

  const buildSettingsForm = (tab, payload, message = "") => {
    const minecraft = payload?.minecraft || {};
    const defaults = payload?.defaults || {};
    const root = document.createElement("div");
    root.className = "minecraft-native-settings";
    if (message) root.appendChild(noticeNode(message));

    if (!payload?.configured) {
      root.appendChild(noticeNode(jvT("ui.c1a3fcc4188f"), "warning"));
      return root;
    }

    const section = document.createElement("section");
    section.className = "panel details minecraft-workspace-section";
    const form = document.createElement("form");
    form.className = "minecraft-native-form";
    form.dataset.minecraftSettingsTab = tab;

    if (tab === "memory") {
      section.innerHTML = `
        <div class="section-heading"><div><p class="eyebrow">${jvH("ui.c3963aedaac6")}</p><h2>${jvH("ui.5c125b6e770b")}</h2></div></div>
        <p class="muted compact">${jvH("ui.32a912c94bb1")}</p>
        <div class="minecraft-native-presets" role="group" aria-label="${jvH("ui.99f64d108845")}">
          <button type="button" class="secondary" data-minecraft-memory-preset="light">${jvH("ui.dbcd5e7bb7a0")}</button>
          <button type="button" class="secondary" data-minecraft-memory-preset="recommended">${jvH("ui.d70604e84304")}</button>
          <button type="button" class="secondary" data-minecraft-memory-preset="high">${jvH("ui.5e644c364e57")}</button>
          <button type="button" class="secondary" data-minecraft-memory-preset="custom">${jvH("ui.494ca78f7374")}</button>
        </div>
        <div class="minecraft-native-grid">
          <label>${jvH("ui.038f7b4cf610")}<input name="java_memory" required autocomplete="off"><span class="minecraft-native-help">${jvH("ui.7e80d90bb21d")}</span></label>
          <label>${jvH("ui.dfd505296613")}<input name="container_memory" required autocomplete="off"><span class="minecraft-native-help">${jvH("ui.cae92cc1e542")}</span></label>
        </div>
        <div class="minecraft-native-memory-status" data-minecraft-memory-status></div>`;
      section.dataset.maxPlayers = String(minecraft.max_players || defaults.max_players || 10);
      section.querySelector('[name="java_memory"]').value = minecraft.java_memory || "";
      section.querySelector('[name="container_memory"]').value = minecraft.container_memory || "";
      form.appendChild(section);
      initNativeMemoryControls(section, defaults);
    } else if (tab === "gameplay") {
      section.innerHTML = `
        <div class="section-heading"><div><p class="eyebrow">${jvH("ui.aef7de28d529")}</p><h2>${jvH("ui.31bcb8940fff")}</h2></div></div>
        <div class="minecraft-native-grid">
          <label>${jvH("ui.e1f268f4f175")}<input name="max_players" type="number" min="1" step="1" required></label>
          <input name="motd" type="hidden">
          <label>${jvH("ui.ede76ceab6f1")}<select name="game_mode" required>
            <option value="survival">${jvH("ui.8302496de0ac")}</option>
            <option value="creative">${jvH("ui.2c7e63313045")}</option>
            <option value="adventure">${jvH("ui.9a3c16424750")}</option>
            <option value="spectator">${jvH("ui.5ccda7f08fd1")}</option>
          </select></label>
        </div>`;
      section.querySelector('[name="max_players"]').value = String(minecraft.max_players || 10);
      section.querySelector('[name="motd"]').value = minecraft.motd || "";
      section.querySelector('[name="game_mode"]').value = minecraft.game_mode || "survival";
      form.appendChild(section);
    } else if (tab === "crossplay") {
      section.innerHTML = `
        <div class="section-heading"><div><p class="eyebrow">${jvH("ui.1744b96470b5")}</p><h2>${jvH("ui.d2317470b6b7")}</h2></div></div>
        <div class="minecraft-native-grid">
          <label>${jvH("ui.4d830c2e53c4")}<input name="java_port" type="number" min="1" max="65535" required></label>
          <label>${jvH("ui.60abc420b980")}<input name="bedrock_port" type="number" min="1" max="65535" required></label>
        </div>
        <label class="minecraft-native-toggle system-ups-shutdown-switch"><span>${jvH("ui.dc9b45c28c00")}</span><input name="bedrock_enabled" type="checkbox"></label>
        <p class="muted compact">${jvH("ui.edb77020bccc")}</p>
        <div class="minecraft-crossplay-status"><span data-crossplay-compatibility class="muted">${jvH("ui.97a94e9a52fc")}</span></div>`;
      section.querySelector('[name="java_port"]').value = String(minecraft.java_port || 25565);
      section.querySelector('[name="bedrock_port"]').value = String(minecraft.bedrock_port || 19132);
      section.querySelector('[name="bedrock_enabled"]').checked = Boolean(minecraft.bedrock_enabled);
      section.querySelector('[name="bedrock_enabled"]').disabled = !["paper", "purpur", "spigot", "pufferfish", "leaf"].includes(minecraft.server_type);
      if (!["paper", "purpur", "spigot", "pufferfish", "leaf"].includes(minecraft.server_type)) section.querySelector('[data-crossplay-compatibility]').textContent = jvT("ui.e01b053efa2e");
      requestWorkspaceJSON("/api/version/workspace/status").then((status) => {
        const node = section.querySelector("[data-crossplay-compatibility]");
        if (!node || !status) return;
        const installed = status.installed || jvT("ui.b764cdc0eab7");
        const toggle = section.querySelector('[name="bedrock_enabled"]');
        toggle.disabled = !status.managed_crossplay_available;
        if (!status.managed_crossplay_available) { node.textContent = jvT("ui.e01b053efa2e"); return; }
        const shortcut = document.createElement("span");
        shortcut.className = "minecraft-crossplay-status";
        shortcut.innerHTML = `<span class="muted">· open</span><button type="button" class="secondary" data-open-version>${jvH("ui.dd167905de0d")}</button>`;
        shortcut.querySelector('[data-open-version]').addEventListener("click", () => document.querySelector("[data-version-open]")?.click());
        node.parentElement.appendChild(shortcut);
        if (!status.crossplay_enabled) node.textContent = jvT("ui.02c54115f1e0", {v0: (installed)});
        else if (status.crossplay_compatible) node.textContent = jvT("ui.5e3c0c33d4bd", {v0: (installed), v1: (status.geyser_supported_version)});
        else node.textContent = status.reason || jvT("ui.09edf1cbb93f");
      }).catch(() => { const node = section.querySelector("[data-crossplay-compatibility]"); if (node && ["paper", "purpur", "spigot", "pufferfish", "leaf"].includes(minecraft.server_type)) node.textContent = jvT("ui.09edf1cbb93f"); });
      form.appendChild(section);
    }

    const actions = document.createElement("div");
    actions.className = "minecraft-native-actions";
    const reviewButton = document.createElement("button");
    reviewButton.type = "submit";
    reviewButton.textContent = jvT("ui.6a47708f4e14");
    const note = document.createElement("span");
    note.className = "muted";
    note.textContent = jvT("ui.46ee57962a56");
    actions.append(note, reviewButton);
    form.appendChild(actions);
    root.appendChild(form);
    return root;
  };

  const settingsParams = (form) => {
    const params = new URLSearchParams();
    new FormData(form).forEach((value, key) => params.append(key, String(value)));
    params.set("csrf", csrf);
    params.set("tab", form.dataset.minecraftSettingsTab || "");
    return params;
  };

  const minecraftSettingsReviewValue = (change, value) => {
    if (change.field !== "bedrock_enabled") return value;
    switch (String(value).trim().toLowerCase()) {
      case "yes":
      case "true":
      case "on":
        return jvT("ui.92c1cdfdf4cb");
      case "no":
      case "false":
      case "off":
        return jvT("ui.75081b593d15");
      default:
        return value;
    }
  };

  const renderSettingsReview = (root, plan, params, options = {}) => {
    root.querySelector("[data-minecraft-native-review]")?.remove();
    const settingsForm = options.form || root.querySelector("[data-minecraft-settings-tab]");
    if (settingsForm) settingsForm.hidden = false;
    const review = document.createElement("section");
    review.className = "panel details minecraft-native-review";
    review.dataset.minecraftNativeReview = "true";

    const heading = document.createElement("div");
    heading.className = "section-heading";
    const headingText = document.createElement("div");
    const eyebrow = document.createElement("p");
    eyebrow.className = "eyebrow";
    eyebrow.textContent = jvT("ui.aff0766a5290");
    const title = document.createElement("h2");
    title.textContent = jvT("ui.486f49f03bf7");
    headingText.append(eyebrow, title);
    heading.appendChild(headingText);
    if (plan.restart_required) {
      const badge = document.createElement("span");
      badge.className = "badge";
      badge.textContent = jvT("ui.b46871152667");
      heading.appendChild(badge);
    }
    review.appendChild(heading);

    (plan.warnings || []).forEach((warning) => review.appendChild(noticeNode(warning, "warning")));

    const changes = Array.isArray(plan.changes) ? plan.changes : [];
    if (changes.length === 0) {
      review.appendChild(messageNode(jvT("ui.2c4ea6323eaf")));
      root.appendChild(review);
      return;
    }

    if (settingsForm) settingsForm.hidden = true;
    const changeList = document.createElement("div");
    changeList.className = "minecraft-native-change-list";
    changes.forEach((change) => {
      const row = document.createElement("div");
      row.className = "minecraft-native-change-row";
      const label = document.createElement("div");
      const strong = document.createElement("strong");
      strong.textContent = change.label || change.field || jvT("ui.818cd8d91aea");
      label.appendChild(strong);
      if (change.restart_required) label.appendChild(messageNode(jvT("ui.2ea1a1d35390"), "muted"));
      const values = document.createElement("div");
      const before = document.createElement("span");
      before.className = "minecraft-native-change-value";
      before.textContent = minecraftSettingsReviewValue(change, change.before || "—");
      const arrow = document.createElement("span");
      arrow.textContent = "→";
      const after = document.createElement("span");
      after.className = "minecraft-native-change-value";
      after.textContent = minecraftSettingsReviewValue(change, change.after || "—");
      values.append(before, arrow, after);
      row.append(label, values);
      changeList.appendChild(row);
    });
    review.appendChild(changeList);

    if (Number.isFinite(Number(plan.memory_remaining_mib)) && Number(plan.memory_remaining_mib) > 0) {
      review.appendChild(messageNode(jvT("ui.72f71ac62c23", {v0: (plan.memory_remaining_mib)})));
    }

    let playerConfirmation = null;
    if (plan.confirmation_required) {
      const label = document.createElement("label");
      label.className = "minecraft-native-toggle system-ups-shutdown-switch";
      const text = document.createElement("span");
      text.textContent = jvT("ui.1af7241ff4ec", {v0: (plan.server_name || "Minecraft")});
      playerConfirmation = document.createElement("input");
      playerConfirmation.type = "checkbox";
      playerConfirmation.name = "confirm_players";
      playerConfirmation.setAttribute("role", "switch");
      label.append(text, playerConfirmation);
      review.appendChild(label);
    } else if (!plan.reviewed_settings && plan.memory_restart_required) {
      review.appendChild(noticeNode(jvT("ui.6dd251904175"), "warning"));
    } else if (!plan.reviewed_settings && plan.restart_required) {
      review.appendChild(noticeNode(jvT("ui.c4b1864f0f7a"), "warning"));
    }

    const actions = document.createElement("div");
    actions.className = "minecraft-native-actions";
    const cancel = document.createElement("button");
    cancel.type = "button";
    cancel.className = "secondary";
    cancel.textContent = jvT("ui.87a8ca7f6bd8");
    cancel.addEventListener("click", () => {
      review.remove();
      if (settingsForm) settingsForm.hidden = false;
    });
    const apply = document.createElement("button");
    apply.type = "button";
    apply.textContent = jvT("ui.85045ccc0567");
    let applying = false;
    const updateApply = () => { apply.disabled = Boolean(applying || plan.apply_blocked || (playerConfirmation && !playerConfirmation.checked)); };
    playerConfirmation?.addEventListener("change", updateApply);
    updateApply();
    if (plan.apply_blocked) {
      const retry = document.createElement("button");
      retry.type = "button";
      retry.className = "secondary";
      retry.textContent = jvT("ui.d4ea61c79c9b");
      retry.addEventListener("click", async () => {
        retry.disabled = true;
        try {
          const refreshed = await requestWorkspaceJSON("/api/minecraft/workspace/settings/plan", {method: "POST", body: params});
          if (refreshed) renderSettingsReview(root, refreshed, params, options);
        } catch (error) {
          if (!error.staleSelection) review.appendChild(noticeNode(error.message || jvT("ui.157f117393c0"), "warning"));
        } finally { retry.disabled = false; }
      });
      actions.appendChild(retry);
    }
    actions.append(cancel, apply);
    review.appendChild(actions);
    root.prepend(review);

    apply.addEventListener("click", async () => {
      if (apply.disabled) return;
      const selection = globalThis.justVoxelSelectionGeneration.minecraft;
      applying = true;
      if (playerConfirmation) playerConfirmation.disabled = true;
      apply.disabled = true;
      cancel.disabled = true;
      if (state) state.textContent = plan.memory_restart_required ? jvT("ui.7938d0e41498") : jvT("ui.3329a9bb48b9");
      try {
        const applyParams = new URLSearchParams(params);
        if (plan.review) applyParams.set("review", plan.review);
        if (playerConfirmation?.checked) applyParams.set("confirm_players", "yes");
        const result = await requestWorkspaceJSON("/api/minecraft/workspace/settings/apply", {
          method: "POST",
          headers: { "Content-Type": "application/x-www-form-urlencoded" },
          body: applyParams.toString(),
        });
        if (!result) return;
        if (!result.applied && (result.confirmation_required || result.apply_blocked)) {
          if (state) state.textContent = "";
          renderSettingsReview(root, result, params, options);
          return;
        }
        if (!result.applied) throw new Error(result.error || jvT("ui.59f7401b3446"));
        if (options.onApplied) { await options.onApplied(); return; }
        let savedMessage = result.message || jvT("ui.ec76c2a9a1e6");
        if (result.restarted) savedMessage = jvT("ui.3cf07307881b");
        else if (result.restart_deferred) savedMessage = jvT("ui.e61fb97a16c3");
        await loadCurrentTab(savedMessage);
      } catch (error) {
        if (error.staleSelection || selection !== globalThis.justVoxelSelectionGeneration.minecraft) return;
        if (options.onError) { await options.onError(error); return; }
        review.appendChild(noticeNode(error?.message || jvT("ui.96736db71e0a"), "warning"));
        if (state) state.textContent = "";
        applying = false;
        if (playerConfirmation) playerConfirmation.disabled = false;
        updateApply();
        cancel.disabled = false;
      }
    });
  };

  const renderSettingsTab = async (sequence, message = "") => {
    const payload = await requestWorkspaceJSON("/api/minecraft/workspace/settings");
    if (!payload || sequence !== loadSequence || !content) return;
    const root = buildSettingsForm(currentTab, payload, message);
    content.replaceChildren(root);
    const form = root.querySelector("[data-minecraft-settings-tab]");
    form?.addEventListener("submit", async (event) => {
      const selection = globalThis.justVoxelSelectionGeneration.minecraft;
      event.preventDefault();
      const submitter = event.submitter;
      if (submitter) submitter.disabled = true;
      if (state) state.textContent = jvT("ui.fcc53388888f");
      try {
        const params = settingsParams(form);
        const plan = await requestWorkspaceJSON("/api/minecraft/workspace/settings/plan", {
          method: "POST",
          headers: { "Content-Type": "application/x-www-form-urlencoded" },
          body: params.toString(),
        });
        if (!plan) return;
        renderSettingsReview(root, plan, params);
        if (state) state.textContent = "";
      } catch (error) {
        if (error.staleSelection || selection !== globalThis.justVoxelSelectionGeneration.minecraft) return;
        if (state) state.textContent = error?.message || jvT("ui.6081772cc7a8");
      } finally {
        if (submitter) submitter.disabled = false;
      }
    });
  };

  const renderPlayers = async (sequence, message = "") => {
    const [snapshot, payload] = await Promise.all([
      requestWorkspaceJSON("/api/dashboard-status"),
      requestWorkspaceJSON("/api/minecraft/workspace/whitelist"),
    ]);
    if (!snapshot || !payload || sequence !== loadSequence || !content) return;
    const root = document.createElement("div");
    root.className = "minecraft-native-operations";
    if (message) root.appendChild(noticeNode(message));

    const minecraft = snapshot?.status?.minecraft || {};
    const players = snapshot?.players || {};
    const controls = document.createElement("section");
    controls.className = "minecraft-player-controls";
    controls.appendChild(messageNode(jvT("ui.40c1289fadcd", {v0: (players.online ?? 0), v1: (players.max ?? minecraft.max_players ?? "—")})));
    if (!minecraft.configured || players.state === "not_configured") controls.appendChild(messageNode(jvT("ui.bd7bbac1db61")));
    else if (players.state === "unavailable") controls.appendChild(messageNode(players.error || jvT("ui.7dfac59b23ae")));
    const form = document.createElement("form");
    form.className = "minecraft-native-form";
    form.innerHTML = `
      <div class="minecraft-player-add">
        <label class="minecraft-player-platform">${jvH("ui.c78ffe195710")}<select name="platform"><option value="java">Java</option><option value="bedrock">Bedrock</option></select></label>
        <label class="minecraft-player-input">${jvH("ui.69aaf45efe73")}<input name="name" required maxlength="64" autocomplete="off"></label>
        <button type="submit">${jvH("ui.ef207ba9378a")}</button>
      </div>
      <p class="muted compact">${jvH("ui.087edf48a988")}<br>${jvH("ui.499128c660df")}</p>`;
    controls.appendChild(form);
    root.appendChild(controls);

    const section = document.createElement("section");
    section.className = "minecraft-whitelist-section";
    section.innerHTML = `<div class="section-heading"><h2>${jvH("ui.8559afa62d46")}</h2><div class="minecraft-whitelist-state"></div></div><div class="minecraft-whitelist-list" data-minecraft-native-whitelist></div>`;
    const enabled = payload.whitelist_enabled;
    const status = document.createElement("span");
    status.className = "badge";
    status.textContent = enabled === true ? jvT("ui.92c1cdfdf4cb") : enabled === false ? jvT("ui.75081b593d15") : jvT("ui.ca1844969742");
    const header = section.querySelector(".minecraft-whitelist-state");
    header.appendChild(status);
    if (payload.can_toggle && typeof enabled === "boolean") {
      const toggle = document.createElement("button");
      toggle.type = "button";
      toggle.className = "minecraft-whitelist-switch";
      toggle.setAttribute("role", "switch");
      toggle.setAttribute("aria-label", jvT("ui.8559afa62d46"));
      toggle.setAttribute("aria-checked", String(enabled));
      toggle.addEventListener("click", async () => {
        const selection = globalThis.justVoxelSelectionGeneration.minecraft;
        if (enabled && !window.confirm(jvT("ui.d7d23ddf450d"))) return;
        toggle.disabled = true;
        try {
          const result = await requestWorkspaceJSON("/api/minecraft/workspace/whitelist/state", {
            method: "POST",
            headers: { "Content-Type": "application/x-www-form-urlencoded" },
            body: new URLSearchParams({ csrf, enabled: String(!enabled) }).toString(),
          });
          if (result) await loadCurrentTab(result.message || jvT("ui.f3ac5dffc52f"));
        } catch (error) {
          if (error.staleSelection || selection !== globalThis.justVoxelSelectionGeneration.minecraft) return;
          const message = error?.message || jvT("ui.fa3b35c1d480");
          // Reload authoritative state after recovery; never optimistically flip.
          try { await loadCurrentTab(); } catch (_) { /* Keep the last authoritative state visible. */ }
          if (state) state.textContent = message;
        } finally {
          toggle.disabled = false;
        }
      });
      header.appendChild(toggle);
    }

    const changeWhitelist = async (params, submitter) => {
      const selection = globalThis.justVoxelSelectionGeneration.minecraft;
      if (submitter) submitter.disabled = true;
      if (state) state.textContent = jvT("ui.6509e178ad5a");
      try {
        params.set("csrf", csrf);
        const result = await requestWorkspaceJSON("/api/minecraft/workspace/whitelist", {
          method: "POST",
          headers: { "Content-Type": "application/x-www-form-urlencoded" },
          body: params.toString(),
        });
        if (!result) return;
        await loadCurrentTab(result.message || jvT("ui.f3ac5dffc52f"));
      } catch (error) {
        if (error.staleSelection || selection !== globalThis.justVoxelSelectionGeneration.minecraft) return;
        if (state) state.textContent = error?.message || jvT("ui.85d6efa05283");
      } finally {
        if (submitter) submitter.disabled = false;
      }
    };

    const list = section.querySelector("[data-minecraft-native-whitelist]");
    if (!payload.parsed) {
      list.appendChild(noticeNode(jvT("ui.a2379aee5d94"), "warning"));
      const diagnostic = document.createElement("pre");
      diagnostic.className = "log-box";
      diagnostic.textContent = payload.output || jvT("ui.49ca2f2698af");
      list.appendChild(diagnostic);
    } else if (!payload.entries?.length) {
      list.appendChild(messageNode(jvT("ui.00528beb1c2c")));
    } else {
      const table = document.createElement("table");
      table.className = "minecraft-whitelist-table";
      table.innerHTML = `<thead><tr><th scope="col">${jvH("ui.c78ffe195710")}</th><th scope="col">${jvH("ui.64aee8c6cbd0")}</th><th scope="col">${jvH("ui.920e413c7d41")}</th><th scope="col">Actions</th></tr></thead>`;
      const body = document.createElement("tbody");
      const onlineIdentities = new Set(Array.isArray(players.names) ? players.names : []);
      payload.entries.forEach((entry) => {
        const row = document.createElement("tr");
        row.className = "minecraft-whitelist-row";
        row.dataset.playerIdentity = entry.identity;
        const remove = document.createElement("button");
        remove.type = "button";
        remove.className = "secondary";
        remove.textContent = jvT("ui.c3812fc4acb8");
        remove.setAttribute("aria-label", jvT("ui.2a972b9c2175", {v0: (entry.display), v1: (entry.platform === "bedrock" ? "Bedrock" : "Java")}));
        remove.addEventListener("click", () => changeWhitelist(new URLSearchParams({
          platform: entry.platform,
          action: "remove",
          name: entry.platform === "bedrock" ? entry.display : entry.identity,
        }), remove));
        [entry.platform === "bedrock" ? "Bedrock" : "Java", entry.display,
          onlineIdentities.has(entry.identity) ? jvT("ui.0d21bd52022c") : jvT("ui.a1794783aab7")].forEach((value) => {
          const cell = document.createElement("td");
          cell.textContent = value;
          row.appendChild(cell);
        });
        const actions = document.createElement("td");
        actions.appendChild(remove);
        row.appendChild(actions);
        body.appendChild(row);
      });
      table.appendChild(body);
      list.appendChild(table);
    }
    root.appendChild(section);
    content.replaceChildren(root);
    form.addEventListener("submit", (event) => {
      event.preventDefault();
      const params = new URLSearchParams();
      new FormData(form).forEach((value, key) => params.append(key, String(value)));
      params.set("action", "add");
      changeWhitelist(params, event.submitter);
    });
  };

  const renderLogs = async (sequence, message = "") => {
    const payload = await requestWorkspaceJSON("/api/minecraft/workspace/logs");
    if (!payload || sequence !== loadSequence || !content) return;
    const root = document.createElement("div");
    root.className = "minecraft-recent-logs-view";
    if (message) root.appendChild(noticeNode(message));
    const section = document.createElement("section");
    section.className = "panel details minecraft-workspace-section";
    const heading = document.createElement("div");
    heading.className = "section-heading";
    const headingText = document.createElement("div");
    const eyebrow = document.createElement("p");
    eyebrow.className = "eyebrow";
    eyebrow.textContent = jvT("ui.268f14bbfe11");
    const title = document.createElement("h2");
    title.textContent = jvT("ui.8397ad9d9848");
    headingText.append(eyebrow, title);
    heading.appendChild(headingText);
    const pre = document.createElement("pre");
    pre.className = "log-box";
    pre.textContent = Array.isArray(payload.lines) && payload.lines.length ? payload.lines.join("\n") : jvT("ui.b61eab1c58ef");
    section.append(heading, pre);
    root.appendChild(section);
    content.replaceChildren(root);
  };

  const renderRcon = async (sequence) => {
    if (!document.body.classList.contains("role-administrator")) throw new Error(jvT("ui.2d3d0b30bde9"));
    const generation = globalThis.justVoxelSelectionGeneration.minecraft;
    const snapshot = await requestWorkspaceJSON("/api/dashboard-status");
    if (!snapshot || sequence !== loadSequence || !content) return;
    const selectedURL = globalThis.justVoxelSelectedURL("/api/minecraft/workspace/console", "minecraft");
    const id = new URL(selectedURL, window.location.origin).searchParams.get("server");
    const isCurrent = () => generation === globalThis.justVoxelSelectionGeneration.minecraft;
    content.replaceChildren(createMinecraftRconView({
      model: rconModel, running: snapshot.status?.minecraft?.state === "running", isCurrent,
      request: values => {
        if (!isCurrent()) return Promise.reject(globalThis.justVoxelStaleSelection());
        return requestWorkspaceJSON(selectedURL, {method: "POST", body: new URLSearchParams({csrf, server_id: id, ...values})});
      },
    }));
  };

  async function loadCurrentTab(message = "") {
    if (globalThis.justVoxelSelectionReady) await globalThis.justVoxelSelectionReady;
    const selection = globalThis.justVoxelSelectionGeneration.minecraft;
    const sequence = ++loadSequence;
    if (state) state.textContent = jvT("ui.ba3bbbe10d8b");
    if (refreshButton) refreshButton.disabled = true;
    if (content) content.replaceChildren();
    try {
      if (settingsTabs.has(currentTab)) await renderSettingsTab(sequence, message);
      else if (currentTab === "players") await renderPlayers(sequence, message);
      else if (currentTab === "logs") await renderLogs(sequence, message);
      else if (currentTab === "rcon") await renderRcon(sequence);
      else await renderOverview(sequence, message);
      if (sequence === loadSequence && state?.textContent === jvT("ui.ba3bbbe10d8b")) state.textContent = "";
    } catch (error) {
      if (error.staleSelection || selection !== globalThis.justVoxelSelectionGeneration.minecraft) return;
      if (sequence !== loadSequence) return;
      if (content) content.replaceChildren();
      if (state) state.textContent = error?.message || jvT("ui.e0d15c678c4b");
    } finally {
      if (sequence === loadSequence && refreshButton) refreshButton.disabled = false;
    }
    return sequence;
  }

  const selectTab = (tab) => {
    currentTab = tab;
    tabs.forEach((button) => button.setAttribute("aria-selected", button.dataset.minecraftTab === tab ? "true" : "false"));
    loadCurrentTab();
  };

  window.addEventListener("justvoxel:server-selection-changed", (event) => {
    if (event.detail?.workspace !== "minecraft") return;
    rconModel = newMinecraftRconState();
    ++loadSequence;
    content?.replaceChildren();
    if (event.detail.inventoryEmpty && globalThis.document?.querySelector("#dashboard[data-dashboard-status]")?.dataset.selectedMinecraftId === "" &&
        !new URLSearchParams(globalThis.location?.search || "").has("server")) {
      currentTab = "overview";
      tabs.forEach((button) => button.setAttribute("aria-selected", button.dataset.minecraftTab === currentTab ? "true" : "false"));
      content?.appendChild(messageNode(jvT("ui.be9166f41a41")));
      if (state) state.textContent = jvT("ui.3807733d20d3");
      return;
    }
    if (minecraftDialog.open) void loadCurrentTab();
  });
  const refreshWorkspace = () => globalThis.justVoxelServerSelection(minecraftDialog.querySelector("[data-server-selector-slot]"), true)
    .then(() => loadCurrentTab()).catch(error => { if (state) state.textContent = error.message; });
  const workspaceWindow = setupWorkspaceWindow(minecraftDialog, { onOpen: refreshWorkspace });

  minecraftOpen.addEventListener("click", () => {
    if (controlCenter) controlCenter.open = false;
    workspaceWindow?.open();
  });
  closeButton?.addEventListener("click", () => workspaceWindow?.close());
  refreshButton?.addEventListener("click", refreshWorkspace);
  tabs.forEach((button) => button.addEventListener("click", () => selectTab(button.dataset.minecraftTab)));

  if (readWorkspaceWindowState("minecraft").open) workspaceWindow?.open();
}


function validateTerminalAppearance(value) {
  return {
    fontSize: ["", "12", "14", "16", "18", "20", "22"].includes(value?.fontSize) ? value.fontSize : "",
    background: ["", "black", "dark-gray"].includes(value?.background) ? value.background : "",
  };
}

function readTerminalAppearance() {
  try {
    return validateTerminalAppearance(JSON.parse(localStorage.getItem("justvoxel.terminal.appearance")));
  } catch {
    return validateTerminalAppearance(null);
  }
}

function terminalAppearanceURL() {
  const appearance = readTerminalAppearance();
  const query = new URLSearchParams();
  if (appearance.fontSize) query.set("fontSize", appearance.fontSize);
  if (appearance.background) {
    const background = appearance.background === "black" ? "#000000" : "#202124";
    query.set("theme", JSON.stringify({background}));
  }
  return query.size ? "/terminal/?" + query.toString() : "/terminal/";
}

function setupTerminalAppearanceSettings(workspace, settings, terminal) {
  const fontSize = settings.querySelector("[data-terminal-font-size]");
  const background = settings.querySelector("[data-terminal-background]");
  const error = settings.querySelector("[data-terminal-settings-error]");
  workspace.querySelector("[data-terminal-settings-toggle]").addEventListener("click", () => {
    const appearance = readTerminalAppearance();
    fontSize.value = appearance.fontSize;
    background.value = appearance.background;
    error.hidden = true;
    settings.showModal();
  });
  settings.querySelector("[data-terminal-settings-reset]").addEventListener("click", () => {
    fontSize.value = "";
    background.value = "";
    error.hidden = true;
  });
  settings.querySelector("[data-terminal-settings-apply]").addEventListener("click", () => {
    const appearance = validateTerminalAppearance({fontSize: fontSize.value, background: background.value});
    try {
      if (appearance.fontSize || appearance.background) {
        localStorage.setItem("justvoxel.terminal.appearance", JSON.stringify(appearance));
      } else {
        localStorage.removeItem("justvoxel.terminal.appearance");
      }
    } catch {
      error.textContent = jvT("ui.2ef05da6deda");
      error.hidden = false;
      return;
    }
    settings.close();
    void terminal.show(true);
  });
}

// Terminal owns its on-demand lifecycle independently of System.
function createTerminalWorkspace(dialog, csrf) {
  const frame = dialog.querySelector('[data-terminal-frame]');
  const state = dialog.querySelector('[data-terminal-state]');
  const retry = dialog.querySelector('[data-terminal-retry]');
  let generation = 0, pending = Promise.resolve(), touched = false;
  const request = async (action) => {
    const response = await fetch('/api/system/workspace/terminal' + (action ? '/' + action : ''), {
      method: action ? 'POST' : "GET", credentials: 'same-origin', cache: 'no-store',
      ...(action ? {body: new URLSearchParams({csrf})} : {}),
    });
    if (!response.ok) throw new Error(jvT("ui.f59fd322d314"));
    return response.json();
  };
  const mutate = action => {
    const result = pending.then(() => request(action));
    pending = result.catch(() => {});
    return result;
  };
  const clearFrame = () => { frame.hidden = true; frame.removeAttribute('src'); };
  const show = async (refresh = false) => {
    if (!refresh && frame.getAttribute('src')) return;
    const ticket = ++generation;
    touched = true;
    clearFrame();
    state.textContent = jvT("ui.d62477f0d7d2");
    retry.hidden = true;
    const current = () => ticket === generation && dialog.open;
    try {
      // A refresh disconnects the old shell and waits for its managed stop
      // before reconnecting; close/reopen uses the same serialized queue.
      if (refresh) await mutate('stop');
      await pending;
      if (!current()) return;
      let status = await request();
      if (!current()) return;
      if (!status.ready) await mutate('start');
      const deadline = Date.now() + 30000;
      while (!status.ready) {
        if (!current()) return;
        if (Date.now() >= deadline) throw new Error(jvT("ui.c145a020ecf5"));
        await new Promise(resolve => setTimeout(resolve, 1000));
        if (!current()) return;
        status = await request();
        if (!status.ready && status.state === 'failed') throw new Error(jvT("ui.c145a020ecf5"));
      }
      if (!current()) return;
      state.textContent = '';
      frame.src = terminalAppearanceURL();
      frame.hidden = false;
    } catch (error) {
      if (current()) { clearFrame(); state.textContent = error.message; retry.hidden = false; }
    }
  };
  retry.addEventListener('click', () => void show(true));
  return {
    show,
    close: () => {
      ++generation;
      clearFrame();
      if (touched) {
        touched = false;
        void mutate('stop').catch(() => { state.textContent = jvT("ui.72318018d103"); });
      }
    },
  };
}

const terminalDialog = document.querySelector("[data-terminal-workspace-dialog]");
if (terminalDialog) {
  const terminal = createTerminalWorkspace(terminalDialog, document.querySelector("[data-system-workspace-csrf]")?.value || "");
  setupTerminalAppearanceSettings(terminalDialog, document.querySelector("[data-terminal-settings-dialog]"), terminal);
  const workspace = setupWorkspaceWindow(terminalDialog, {
    onOpen: () => { void terminal.show(); },
    // A queued native close event may arrive after a rapid reopen.
    onClose: () => { if (!terminalDialog.open) terminal.close(); },
  });
  document.querySelector("[data-terminal-open]")?.addEventListener("click", () => {
    if (!document.body.classList.contains("primary-administrator")) return;
    if (controlCenter) controlCenter.open = false;
    workspace.open();
  });
  terminalDialog.querySelector("[data-terminal-close]").addEventListener("click", () => {
    terminal.close();
    workspace.close();
  });
  terminalDialog.querySelector("[data-terminal-refresh]").addEventListener("click", () => void terminal.show(true));
  const restoreTerminal = () => {
    if (readWorkspaceWindowState("terminal").open && document.body.classList.contains("primary-administrator")) workspace.open();
  };
  if (document.body.classList.contains("role-pending")) {
    const observer = new MutationObserver(() => {
      if (document.body.classList.contains("role-pending")) return;
      observer.disconnect();
      restoreTerminal();
    });
    observer.observe(document.body, { attributes: true, attributeFilter: ["class"] });
  } else restoreTerminal();
}

const systemWorkspaceOpen = document.querySelector("[data-system-open]");
const systemWorkspaceDialog = document.querySelector("[data-system-workspace-dialog]");
if (systemWorkspaceOpen && systemWorkspaceDialog) {
  const closeButton = systemWorkspaceDialog.querySelector("[data-system-close]");
  const refreshButton = systemWorkspaceDialog.querySelector("[data-system-refresh]");
  const state = systemWorkspaceDialog.querySelector("[data-system-state]");
  const content = systemWorkspaceDialog.querySelector("[data-system-workspace-content]");
  const tabs = Array.from(systemWorkspaceDialog.querySelectorAll("[data-system-tab]"));
  const allowedSystemTabs = new Set(["health", "history", "users", "security", "logs", "date-time", "wallpaper", "reset", "about", "ups"]);
  const administratorTabs = new Set(["health", "users", "security", "logs", "date-time", "reset"]);
  const upsTabButton = systemWorkspaceDialog.querySelector("[data-system-ups-tab]");
  const upsCSRF = document.querySelector("[data-system-ups-csrf]");
  const systemWorkspaceCSRF = document.querySelector("[data-system-workspace-csrf]")?.value || "";
  let identity = null;
  let upsAvailable = false;
  let securityPane = "authentication";
  let currentTab = "health";
  let initialized = false;
  let loadSequence = 0;
  let resetPollTimer = 0;
  let activeFactoryResetOperationID = "";

  const showFactoryResetComplete = () => {
    window.JustVoxelLiveConsole.close();
    const secure = window.location.protocol === "https:" ? "; Secure" : "";
    document.cookie = "jv_factory_reset_complete=1; Path=/factory-reset-complete; Max-Age=120; SameSite=Strict" + secure;
    window.location.replace("/factory-reset-complete");
  };

  const resetLiveConsoleUnauthorized = (operation) => {
    // Preserve the existing factory-reset session invalidation transition.
    if (operation.operation_type === "factory_reset" && operation.operation_id === activeFactoryResetOperationID) {
      showFactoryResetComplete();
      return;
    }
    window.location.assign("/login");
  };

  const systemHandleAuth = (response) => {
    if (response.redirected) {
      const target = new URL(response.url);
      if (target.pathname === "/login" || target.pathname === "/password") {
        window.location.assign(target.pathname + target.search);
        return true;
      }
    }
    if (response.status === 401) {
      window.location.assign("/login");
      return true;
    }
    return false;
  };

  const systemFetchPage = async (url, options = {}) => {
    const response = await fetch(url, {
      method: options.method || "GET",
      credentials: "same-origin",
      headers: options.headers || { Accept: "text/html" },
      body: options.body,
      cache: "no-store",
      redirect: "follow",
    });
    if (systemHandleAuth(response)) return null;
    return response;
  };

  const systemNamespaceIDs = (root, prefix) => {
    const replacements = new Map();
    root.querySelectorAll("[id]").forEach((element) => {
      const oldID = element.id;
      const newID = prefix + oldID;
      replacements.set(oldID, newID);
      element.id = newID;
    });
    for (const attribute of ["for", "aria-controls", "aria-labelledby", "aria-describedby"]) {
      root.querySelectorAll(`[${attribute}]`).forEach((element) => {
        const tokens = String(element.getAttribute(attribute) || "").split(/\s+/).filter(Boolean);
        element.setAttribute(attribute, tokens.map((token) => replacements.get(token) || token).join(" "));
      });
    }
  };

  const systemReplaceText = (root, replacements) => {
    const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
    const nodes = [];
    while (walker.nextNode()) nodes.push(walker.currentNode);
    nodes.forEach((node) => {
      let value = node.nodeValue || "";
      replacements.forEach(([before, after]) => {
        value = value.split(before).join(after);
      });
      node.nodeValue = value;
    });
  };

  const loadIdentity = async () => {
    if (identity) return identity;
    const response = await fetch("/api/session-info", {
      method: "GET",
      credentials: "same-origin",
      headers: { Accept: "application/json" },
      cache: "no-store",
    });
    if (systemHandleAuth(response)) return null;
    if (!response.ok) throw new Error(jvT("ui.98a9fb5b71d1"));
    const result = await response.json();
    const role = String(result.role || "").toLowerCase();
    const username = String(result.username || "");
    if (!username || !["administrator", "operator", "viewer"].includes(role)) {
      throw new Error(jvT("ui.cfbc63955e94"));
    }
    identity = { username, role };
    document.body.classList.remove("role-pending");
    document.body.classList.toggle("primary-administrator", username === "voxel" && role === "administrator");
    document.body.classList.add(`role-${role}`);
    return identity;
  };

  const syncSystemTabs = () => {
    tabs.forEach((button) => {
      button.setAttribute("aria-selected", button.dataset.systemTab === currentTab ? "true" : "false");
    });
  };

  const systemFetchJSON = async (url, options = {}, factoryResetPollOperationID = "") => {
    const response = await fetch(url, {
      credentials: "same-origin",
      cache: "no-store",
      ...options,
      headers: {
        Accept: "application/json",
        ...(options.headers || {}),
      },
    });
    if (response.status === 401) {
      if (factoryResetPollOperationID && factoryResetPollOperationID === activeFactoryResetOperationID &&
          url === "/api/system/workspace/reset/operations/" + factoryResetPollOperationID) {
        showFactoryResetComplete();
        return null;
      }
      window.location.assign("/login");
      return null;
    }
    let payload = {};
    try {
      payload = await response.json();
    } catch (_) {
      payload = {};
    }
    if (response.status === 403 && String(jvC(payload.error || "")).toLowerCase().includes("password change")) {
      window.location.assign("/password");
      return null;
    }
    if (!response.ok) {
      throw new Error(payload.error || jvT("ui.74d3afec7da4"));
    }
    return payload;
  };

  const systemNotice = (message, kind = "success") => {
    const node = document.createElement("div");
    node.className = "notice " + kind;
    node.textContent = message;
    return node;
  };

  const systemPanel = (eyebrowText, titleText) => {
    const panel = document.createElement("section");
    panel.className = "panel details";
    const heading = document.createElement("div");
    heading.className = "section-heading";
    const wrap = document.createElement("div");
    const eyebrow = document.createElement("p");
    eyebrow.className = "eyebrow";
    eyebrow.textContent = eyebrowText;
    const title = document.createElement("h2");
    title.textContent = titleText;
    wrap.append(eyebrow, title);
    heading.appendChild(wrap);
    panel.appendChild(heading);
    return { panel, heading };
  };

  const cleanSystemHealthOutput = (output) => {
    return output
      // Terminal control strings: OSC (including hyperlinks), DCS, SOS, PM and APC.
      .replace(/(?:\u001b\]|\u009d)[\s\S]*?(?:\u0007|\u001b\\|\u009c)/g, "")
      .replace(/(?:\u001b[PX^_]|[\u0090\u0098\u009e\u009f])[\s\S]*?(?:\u001b\\|\u009c)/g, "")
      // CSI includes colors, cursor movement and erase commands; then other escapes.
      .replace(/(?:\u001b\[|\u009b)[0-?]*[ -/]*[@-~]/g, "")
      .replace(/\u001b[ -/]*[0-~]/g, "")
      // Keep tabs, line feeds and carriage returns for the original layout.
      .replace(/[\u0000-\u0008\u000b\u000c\u000e-\u001f\u007f-\u009f]/g, "")
      .replace(/§x(?:§[0-9a-f]){6}|§[0-9a-fk-orx]/gi, "");
  };

  const renderHealth = async (sequence) => {
    if (sequence !== loadSequence || !content) return;
    const root = document.createElement("div");
    root.className = "system-health-view";
    const built = systemPanel(jvT("ui.982649422d26"), jvT("ui.ba6fa824219e"));
    const pre = document.createElement("pre");
    pre.className = "log-box";
    pre.textContent = jvT("ui.b8539cb6201c");
    built.panel.appendChild(pre);
    root.appendChild(built.panel);
    content.replaceChildren(root);
    const payload = await systemFetchJSON("/api/system/workspace/health");
    if (!payload || sequence !== loadSequence) return;
    const result = payload.result || {};
    pre.textContent = cleanSystemHealthOutput(result.output || jvT("ui.55511275fa2f"));
    root.replaceChildren(systemNotice(
      result.ok ? jvT("ui.fff1eb0f6838") : jvT("ui.d438addf7329"),
      result.ok ? "success" : "error"
    ), built.panel);
  };

  const systemSplitter = (label, move) => {
    const handle = document.createElement("div");
    handle.className = "system-logs-splitter";
    handle.setAttribute("role", "separator");
    handle.setAttribute("aria-label", label);
    handle.setAttribute("aria-orientation", "vertical");
    handle.tabIndex = 0;
    handle.addEventListener("pointerdown", (event) => {
      if (event.button !== 0) return;
      event.preventDefault();
      handle.setPointerCapture(event.pointerId);
      let previous = event.clientX;
      const onMove = (next) => { const delta = next.clientX - previous; previous = next.clientX; move(delta); };
      const onEnd = () => { handle.removeEventListener("pointermove", onMove); handle.removeEventListener("pointerup", onEnd); handle.removeEventListener("pointercancel", onEnd); };
      handle.addEventListener("pointermove", onMove);
      handle.addEventListener("pointerup", onEnd);
      handle.addEventListener("pointercancel", onEnd);
    });
    handle.addEventListener("keydown", (event) => {
      if (event.key === "ArrowLeft" || event.key === "ArrowRight") {
        event.preventDefault();
        move(event.key === "ArrowLeft" ? -16 : 16);
      }
    });
    return handle;
  };

  const renderHistory = async (sequence, message = "") => {
    const payload = await systemFetchJSON("/api/system/workspace/history");
    if (!payload || sequence !== loadSequence || !content) return;
    const root = document.createElement("div");
    root.className = "system-history-view";
    if (message) root.appendChild(systemNotice(message));

    if (payload.role === "administrator") {
      const notificationsPanel = systemPanel(jvT("ui.adf69e75cc7d"), jvT("ui.4e2fa50f7e95")).panel;
      const notifications = Array.isArray(payload.notifications) ? payload.notifications : [];
      if (!notifications.length) {
        const empty = document.createElement("p");
        empty.className = "muted";
        empty.textContent = jvT("ui.64d568230a63");
        notificationsPanel.appendChild(empty);
      } else {
        notifications.forEach((item) => {
          const row = document.createElement("div");
          row.className = "event-row";
          const info = document.createElement("div");
          const title = document.createElement("strong");
          title.textContent = item.title || jvT("ui.7d31b8331399");
          const messageNode = document.createElement("span");
          messageNode.textContent = item.message || "";
          const date = document.createElement("span");
          date.textContent = item.created_at || "";
          info.append(title, messageNode, date);
          const button = document.createElement("button");
          button.type = "button";
          button.className = "secondary";
          button.textContent = jvT("ui.c8f193b315c8");
          button.addEventListener("click", async () => {
            button.disabled = true;
            try {
              const body = new URLSearchParams({ csrf: systemWorkspaceCSRF });
              const result = await systemFetchJSON("/api/system/workspace/history/notifications/" + item.id + "/resolve", {
                method: "POST",
                headers: { "Content-Type": "application/x-www-form-urlencoded" },
                body: body.toString(),
              });
              if (result) await renderHistory(++loadSequence, result.message || jvT("ui.4f376daea4c4"));
            } catch (error) {
              if (state) state.textContent = error?.message || jvT("ui.a63ba84b3034");
              button.disabled = false;
            }
          });
          row.append(info, button);
          notificationsPanel.appendChild(row);
        });
      }
      root.appendChild(notificationsPanel);

      const auditPanel = systemPanel(jvT("ui.14d104cb8f68"), jvT("ui.e76b1ee02686")).panel;
      const events = Array.isArray(payload.audit) ? payload.audit : [];
      if (!events.length) {
        const empty = document.createElement("p");
        empty.className = "muted";
        empty.textContent = jvT("ui.bc542d8b313b");
        auditPanel.appendChild(empty);
      } else {
        const list = document.createElement("div");
        list.className = "event-list";
        events.forEach((event) => {
          const row = document.createElement("div");
          row.className = "event-row";
          const info = document.createElement("div");
          const action = document.createElement("strong");
          action.textContent = event.action || jvT("ui.64cff1319d2f");
          const meta = document.createElement("span");
          meta.textContent = [event.occurred_at, event.actor_username, event.actor_role].filter(Boolean).join(" · ");
          info.append(action, meta);
          if (event.target) {
            const target = document.createElement("span");
            target.textContent = jvT("ui.30eaddeafeae") + event.target;
            info.appendChild(target);
          }
          if (event.context) {
            const context = document.createElement("span");
            context.textContent = event.context;
            info.appendChild(context);
          }
          const badge = document.createElement("span");
          badge.className = "badge";
          badge.textContent = event.success ? jvT("ui.c88a0b907419") : jvT("ui.031a8f0f659d");
          row.append(info, badge);
          list.appendChild(row);
        });
        auditPanel.appendChild(list);
      }
      let notificationWidth = 300;
      const divider = systemSplitter(jvT("ui.164de9de6729"), (delta) => {
        notificationWidth += delta;
        setHistoryWidth();
      });
      const setHistoryWidth = () => {
        notificationWidth = Math.max(240, Math.min(notificationWidth, (root.clientWidth || 800) - 270));
        root.style.setProperty("--history-notifications-width", notificationWidth + "px");
        divider.setAttribute("aria-valuemin", "240");
        divider.setAttribute("aria-valuemax", String(Math.max(240, root.clientWidth - 270)));
        divider.setAttribute("aria-valuenow", String(Math.round(notificationWidth)));
      };
      root.append(divider, auditPanel);
      const historyResizeObserver = new ResizeObserver(() => {
        if (!root.isConnected) { historyResizeObserver.disconnect(); return; }
        setHistoryWidth();
      });
      historyResizeObserver.observe(root);
    } else {
      const activityPanel = systemPanel(jvT("ui.38da1505ca83"), jvT("ui.0724c603152c")).panel;
      const events = Array.isArray(payload.events) ? payload.events : [];
      if (!events.length) {
        const empty = document.createElement("p");
        empty.className = "muted";
        empty.textContent = jvT("ui.ebc026afee98");
        activityPanel.appendChild(empty);
      } else {
        const list = document.createElement("div");
        list.className = "event-list";
        events.forEach((event) => {
          const row = document.createElement("div");
          row.className = "event-row";
          const info = document.createElement("div");
          const action = document.createElement("strong");
          action.textContent = event.action || jvT("ui.64cff1319d2f");
          const date = document.createElement("span");
          date.textContent = event.occurred_at || "";
          info.append(action, date);
          const badge = document.createElement("span");
          badge.className = "badge";
          badge.textContent = event.success ? jvT("ui.c88a0b907419") : jvT("ui.031a8f0f659d");
          row.append(info, badge);
          list.appendChild(row);
        });
        activityPanel.appendChild(list);
      }
      root.appendChild(activityPanel);
    }

    content.replaceChildren(root);
  };

  const systemPostForm = async (url, fields = {}) => {
    const body = new URLSearchParams({ csrf: systemWorkspaceCSRF });
    Object.entries(fields).forEach(([key, value]) => body.set(key, String(value ?? "")));
    return systemFetchJSON(url, {
      method: "POST",
      headers: { "Content-Type": "application/x-www-form-urlencoded" },
      body: body.toString(),
    });
  };

  const systemLogDestination = new URLSearchParams(window.location.search);
  let requestedMigrationLog = systemLogDestination.get("workspace") === "system" && systemLogDestination.get("tab") === "logs" && ["migration", "plugins"].includes(systemLogDestination.get("log_group"))
    ? { group: systemLogDestination.get("log_group") === "plugins" ? jvT("ui.9514b7ff4860") : jvT("ui.cc149190687e"), id: systemLogDestination.get("log_id") || "" } : null;

  let failedUnitsSelected = false;
  let supportLogsSelected = false;
  let networkLogsSelected = false;

  const renderLogs = async (sequence) => {
    const payload = await systemFetchJSON("/api/system/workspace/logs");
    if (!payload || sequence !== loadSequence || !content) return;
    const groups = ["Setup", "Migration", "Restore", "Plugins", "Reset", "Network", "Update", "Support", "Failed units"];
    const logs = Array.isArray(payload.logs) ? payload.logs : [];
    const failedUnits = Array.isArray(payload.failed_units) ? payload.failed_units : [];
    const root = document.createElement("div");
    root.className = "system-logs-layout";
    root.dataset.systemLogsLayout = "";
    const saved = (() => { try { return JSON.parse(localStorage.getItem("justvoxel.system.logs.columns") || "{}") || {}; } catch { return {}; } })();
    let fileWidth = Number.isFinite(saved.files) ? saved.files : 280;
    const setWidths = () => {
      const width = root.clientWidth || 900;
      fileWidth = Math.max(160, Math.min(fileWidth, width - 220 - 10));
      root.style.setProperty("--logs-files-width", fileWidth + "px");
      handle.setAttribute("aria-valuemin", "160");
      handle.setAttribute("aria-valuemax", String(Math.max(160, width - 230)));
      handle.setAttribute("aria-valuenow", String(Math.round(fileWidth)));
    };
    const makeColumn = (name, className) => {
      const column = document.createElement("section");
      column.className = className;
      const heading = document.createElement("h3");
      heading.textContent = name;
      column.appendChild(heading);
      return column;
    };
    const groupColumn = document.createElement("nav");
    groupColumn.className = "system-logs-groups";
    groupColumn.setAttribute("aria-label", jvT("ui.cdff6afac6a7"));
    const fileColumn = makeColumn(jvT("ui.abc7e9892806"), "system-logs-files");
    const generateSupport = document.createElement("button");
    generateSupport.type = "button";
    generateSupport.textContent = jvT("ui.d904faacbc97");
    const supportProgress = document.createElement("p");
    supportProgress.className = "muted";
    supportProgress.setAttribute("role", "status");
    supportProgress.hidden = true;
    const fileList = document.createElement("div");
    fileList.className = "system-logs-list";
    const generateNetwork = document.createElement("button");
    generateNetwork.type = "button";
    generateNetwork.textContent = jvT("ui.a82a3e50dee4");
    const networkProgress = document.createElement("p");
    networkProgress.className = "muted";
    networkProgress.setAttribute("role", "status");
    fileColumn.append(generateSupport, supportProgress, generateNetwork, networkProgress, fileList);
    const reader = makeColumn(jvT("ui.678bfa6af48b"), "system-logs-reader");
    const viewer = document.createElement("div");
    viewer.className = "system-logs-viewer";
    const download = document.createElement("a");
    download.className = "button-link secondary";
    download.textContent = jvT("ui.7b8c8adf4591");
    download.hidden = true;
    const documentText = document.createElement("pre");
    documentText.textContent = jvT("ui.9bd8a2a4f30f");
    viewer.append(download, documentText);
    reader.appendChild(viewer);
    let selectedGroup = requestedMigrationLog?.group || (failedUnitsSelected ? "Failed units" : supportLogsSelected ? "Support" : networkLogsSelected ? "Network" : "Setup");
    let selectedFile = "";
    let requestedEntry = requestedMigrationLog ? logs.find((entry) => entry.category === requestedMigrationLog.group.toLowerCase() && entry.log_id === requestedMigrationLog.id) : null;
    requestedMigrationLog = null;
    const renderFiles = () => {
      fileList.replaceChildren();
      viewer.scrollTop = 0;
      download.hidden = true;
      const live = selectedGroup === "Failed units";
      generateSupport.hidden = selectedGroup !== "Support";
      generateNetwork.hidden = selectedGroup !== "Network";
      networkProgress.hidden = selectedGroup !== "Network" || !networkProgress.textContent;
      fileList.className = selectedGroup === "Network" || selectedGroup === "Support"
        ? "system-logs-list system-diagnostic-report-files" : "system-logs-list";
      supportProgress.hidden = selectedGroup !== "Support" || !supportProgress.textContent;
      fileColumn.children[0].textContent = live ? "Units" : jvT("ui.abc7e9892806");
      documentText.textContent = live ? jvT("ui.39e498ec4cfd") : jvT("ui.9bd8a2a4f30f");
      const entries = live ? failedUnits : logs.filter((entry) => entry.category === selectedGroup.toLowerCase());
      if (!entries.length) {
        const empty = document.createElement("p");
        empty.className = "muted";
        empty.textContent = live ? (payload.failed_units_error ? jvT("ui.138f547a3261") : jvT("ui.d0789dca609b")) : selectedGroup === "Support" ? jvT("ui.a5e44902bd11") : jvT("ui.05f8bb36f881") + selectedGroup.toLowerCase() + " logs available.";
        fileList.appendChild(empty);
        return;
      }
      entries.forEach((entry) => {
        const identity = live ? entry.unit : entry.log_id;
        const button = document.createElement("button");
        button.type = "button";
        button.className = "system-log-entry secondary";
        button.setAttribute("aria-pressed", selectedFile === identity ? "true" : "false");
        const title = document.createElement("strong");
        title.textContent = live ? entry.unit : entry.log_id + ".log";
        const meta = document.createElement("span");
        meta.textContent = live ? [entry.description, entry.active_state, entry.sub_state].filter(Boolean).join(" · ") : [entry.operation_type?.replaceAll("_", " ") || "", entry.state || "", formatDiagnosticLogTimestamp(entry.timestamp)].filter(Boolean).join(" · ");
        button.append(title, meta);
        button.addEventListener("click", async () => {
          selectedFile = identity;
          const clickedGroup = selectedGroup;
          fileList.querySelectorAll("button").forEach((item) => item.setAttribute("aria-pressed", item === button ? "true" : "false"));
          const url = live ? "/api/system/workspace/logs/failed/" + encodeURIComponent(entry.unit) : "/api/system/workspace/logs/" + encodeURIComponent(entry.category) + "/" + encodeURIComponent(entry.log_id);
          documentText.textContent = jvT("ui.2de3b91747f9");
          download.hidden = true;
          try {
            const response = await fetch(url, { credentials: "same-origin", cache: "no-store", headers: { Accept: "text/plain" } });
            if (systemHandleAuth(response)) return;
            if (!response.ok) throw new Error(live ? jvT("ui.c651952bfc8b") : jvT("ui.bde5c83597fc"));
            const data = await response.text();
            if (sequence !== loadSequence || selectedFile !== identity || selectedGroup !== clickedGroup) return;
            documentText.textContent = data;
            download.href = url + "?download=1";
            download.hidden = false;
          } catch (error) {
            if (sequence !== loadSequence || selectedFile !== identity || selectedGroup !== clickedGroup) return;
            documentText.textContent = live ? jvT("ui.c651952bfc8b") : error?.message || jvT("ui.bde5c83597fc");
          }
        });
        fileList.appendChild(button);
        if (entry === requestedEntry) {
          requestedEntry = null;
          button.click();
        }
      });
    };
    generateNetwork.addEventListener("click", async () => {
      generateNetwork.disabled = true;
      networkProgress.textContent = jvT("ui.ad2465a85a74");
      networkProgress.hidden = false;
      try {
        const result = await systemPostForm("/api/system/workspace/logs/network");
        if (!result?.ok || result.log?.category !== "network") throw new Error(jvT("ui.e7e942073afa"));
        if (sequence !== loadSequence || !root.isConnected) return;
        logs.unshift(result.log);
        requestedEntry = result.log;
        networkProgress.textContent = jvT("ui.1c7a3e62d4a0");
        if (selectedGroup === "Network") renderFiles();
      } catch (_) {
        if (sequence !== loadSequence || !root.isConnected) return;
        networkProgress.textContent = jvT("ui.c661ccd39e59");
      } finally {
        generateNetwork.disabled = false;
        networkProgress.hidden = selectedGroup !== "Network";
      }
    });
    generateSupport.addEventListener("click", async () => {
      generateSupport.disabled = true;
      supportProgress.textContent = jvT("ui.967fc5f02f43");
      supportProgress.hidden = false;
      try {
        const result = await systemPostForm("/api/system/workspace/logs/support");
        if (!result?.ok || !result.log || result.log.category !== "support") throw new Error(jvT("ui.1354a3af863e"));
        if (sequence !== loadSequence || !root.isConnected) return;
        logs.unshift(result.log);
        requestedEntry = result.log;
        supportProgress.textContent = jvT("ui.9cb9fb4ec5fd");
        if (selectedGroup === "Support") renderFiles();
      } catch {
        if (sequence !== loadSequence || !root.isConnected) return;
        supportProgress.textContent = jvT("ui.5d986f471fad");
      } finally {
        generateSupport.disabled = false;
        supportProgress.hidden = selectedGroup !== "Support";
      }
    });
    groups.forEach((group) => {
      const button = document.createElement("button");
      button.type = "button";
      button.className = "system-log-group secondary";
      button.textContent = jvS(group);
      button.setAttribute("aria-pressed", group === selectedGroup ? "true" : "false");
      button.addEventListener("click", () => {
        selectedGroup = group;
        failedUnitsSelected = group === "Failed units";
        supportLogsSelected = group === "Support";
        networkLogsSelected = group === "Network";
        selectedFile = "";
        groupColumn.querySelectorAll("button").forEach((item) => item.setAttribute("aria-pressed", item === button ? "true" : "false"));
        renderFiles();
      });
      groupColumn.appendChild(button);
    });
    const handle = systemSplitter(jvT("ui.738167cda6d8"), (delta) => {
      fileWidth += delta;
      setWidths();
      try { localStorage.setItem("justvoxel.system.logs.columns", JSON.stringify({ files: fileWidth })); } catch {}
    });
    handle.dataset.logsSplitter = "files";
    root.append(groupColumn, fileColumn, handle, reader);
    content.replaceChildren(root);
    setWidths();
    const logsResizeObserver = new ResizeObserver(() => {
      if (!root.isConnected) { logsResizeObserver.disconnect(); return; }
      setWidths();
    });
    logsResizeObserver.observe(root);
    renderFiles();
  };

  const renderUsers = async (sequence, message = "") => {
    if (!document.querySelector('link[href="/static/storage-browser.css"]')) {
      const stylesheet = document.createElement("link");
      stylesheet.rel = "stylesheet";
      stylesheet.href = "/static/storage-browser.css";
      document.head.appendChild(stylesheet);
    }
    const payload = await systemFetchJSON("/api/system/workspace/users");
    if (!payload || sequence !== loadSequence || !content) return;
    const root = document.createElement("div");
    root.className = "system-users-view";
    if (message) root.appendChild(systemNotice(message));

    const primary = systemPanel(jvT("ui.41d555d7fceb"), payload.primary_administrator || "voxel").panel;
    const primaryText = document.createElement("p");
    primaryText.className = "muted compact";
    primaryText.textContent = jvT("ui.7a55b26e98f0");
    primary.appendChild(primaryText);
    root.appendChild(primary);

    const create = systemPanel(jvT("ui.ca0b591569ff"), jvT("ui.f06da128c342")).panel;
    const createForm = document.createElement("form");
    createForm.className = "admin-form-grid";
    createForm.innerHTML =
      `<label>${jvH("ui.e3b89e9d33f8")}<input name="username" required maxlength="64" autocomplete="off"></label>` +
      `<label>Role<select name="role" required><option value="operator">Operator</option><option value="viewer">${jvH("ui.678bfa6af48b")}</option></select></label>` +
      `<label>${jvH("ui.e7cf3ef4f17c")}<input type="password" name="password" required autocomplete="new-password"></label>` +
      `<label>${jvH("ui.5ac265f396a2")}<input type="password" name="confirm_password" required autocomplete="new-password"></label>` +
      `<div class="form-actions"><button type="submit">${jvH("ui.f06da128c342")}</button></div>`;
    createForm.querySelectorAll('input[type="password"]').forEach((input) => {
      input.minLength = Number(payload.minimum_password_len || 8);
    });
    createForm.addEventListener("submit", async (event) => {
      event.preventDefault();
      const submit = event.submitter;
      if (submit) submit.disabled = true;
      const data = Object.fromEntries(new FormData(createForm).entries());
      try {
        const result = await systemPostForm("/api/system/workspace/users/create", data);
        if (result) await renderUsers(++loadSequence, result.message || jvT("ui.8b71b6651c39"));
      } catch (error) {
        if (state) state.textContent = error?.message || jvT("ui.969cadf867fb");
        if (submit) submit.disabled = false;
      }
    });
    create.appendChild(createForm);
    root.appendChild(create);

    const listPanel = systemPanel(jvT("ui.e6d207fbe7a8"), jvT("ui.5583497579f3")).panel;
    const users = Array.isArray(payload.users) ? payload.users : [];
    if (!users.length) {
      const empty = document.createElement("p");
      empty.className = "muted compact";
      empty.textContent = jvT("ui.a1b459a55030");
      listPanel.appendChild(empty);
    } else {
      const list = document.createElement("div");
      list.className = "user-list";
      const table = document.createElement("table");
      table.className = "system-users-table";
      const head = document.createElement("thead");
      head.innerHTML = `<tr><th scope="col">${jvH("ui.e3b89e9d33f8")}</th><th scope="col">Role</th><th scope="col">${jvH("ui.920e413c7d41")}</th><th scope="col">${jvH("ui.6b983a81e5e8")}</th><th scope="col">${jvH("ui.838557924a5c")}</th><th scope="col">Actions</th></tr>`;
      const rows = document.createElement("tbody");
      table.append(head, rows);
      list.appendChild(table);
      users.forEach((user, index) => {
        const row = document.createElement("tr");
        for (const value of [user.username || jvT("ui.b512d97e7cbf"), user.role === "operator" ? "Operator" : jvT("ui.678bfa6af48b"), user.enabled ? jvT("ui.92c1cdfdf4cb") : jvT("ui.75081b593d15"),
          `${user.restart_used ?? 0} / ${user.restart_limit ?? 0}`, `${user.backup_used ?? 0} / ${user.backup_limit ?? 0}`]) {
          const cell = document.createElement("td");
          cell.textContent = value;
          row.appendChild(cell);
        }
        const cell = document.createElement("td");
        const menu = document.createElement("details");
        menu.className = "storage-action-menu system-user-menu";
        const trigger = document.createElement("summary");
        trigger.textContent = jvT("ui.5a23444828db");
        trigger.setAttribute("aria-label", jvT("ui.d1f3b7c8d7b5") + user.username);
        const choices = document.createElement("div");
        choices.className = "storage-action-menu-popover";
        // Use the browser top layer so the compact scrolling table cannot clip the menu.
        choices.setAttribute("popover", "auto");
        menu.addEventListener("toggle", () => {
          if (!menu.isConnected) return;
          if (menu.open) {
            const rect = trigger.getBoundingClientRect();
            choices.style.right = Math.max(8, window.innerWidth - rect.right) + "px";
            choices.style.top = Math.max(8, Math.min(rect.bottom + 4, window.innerHeight - 300)) + "px";
            choices.showPopover();
          } else if (choices.matches(":popover-open")) choices.hidePopover();
        });
        choices.addEventListener("toggle", (event) => {
          if (event.newState === "closed") {
            menu.open = false;
            if (choices.contains(document.activeElement)) trigger.focus();
          }
        });
        const openAction = (title, fields, action, submitText, dangerous = false) => {
          menu.open = false;
          if (choices.matches(":popover-open")) choices.hidePopover();
          const dialog = document.createElement("dialog");
          dialog.className = "system-action-dialog system-user-dialog";
          const wrap = document.createElement("div");
          wrap.className = "system-action-dialog-content";
          const heading = document.createElement("h2");
          heading.id = "system-user-action-" + index;
          heading.textContent = title + " · " + user.username;
          dialog.setAttribute("aria-labelledby", heading.id);
          const form = document.createElement("form");
          form.innerHTML = fields;
          form.querySelectorAll('input[type="password"]').forEach((input) => { input.minLength = Number(payload.minimum_password_len || 8); });
          const error = document.createElement("p");
          error.className = "notice error";
          error.hidden = true;
          error.setAttribute("role", "alert");
          const actions = document.createElement("div");
          actions.className = "action-row";
          const cancel = document.createElement("button");
          cancel.type = "button";
          cancel.className = "secondary";
          cancel.textContent = jvT("ui.19766ed6ccb2");
          cancel.addEventListener("click", () => dialog.close());
          const submit = document.createElement("button");
          submit.type = "submit";
          submit.textContent = submitText;
          if (dangerous) {
            submit.className = "danger";
            submit.dataset.destructiveSubmit = "";
            submit.disabled = true;
            const confirmation = document.createElement("div");
            confirmation.className = "destructive-confirmation";
            confirmation.dataset.destructiveConfirmation = "";
            confirmation.innerHTML = `<div class="destructive-confirm-slider" data-destructive-slider-shell><span class="destructive-confirm-slider-text" data-destructive-slider-text>${jvH("ui.656bae9bfca6")}</span><span class="destructive-confirm-slider-thumb" aria-hidden="true">&gt;</span><input type="range" min="0" max="100" step="1" value="0" data-destructive-slider aria-label="${jvH("ui.958924d8ce3a")}"></div><label class="destructive-confirm-toggle-row" data-destructive-toggle-row hidden><span class="destructive-confirm-toggle-callout">${jvH("ui.8ba7a99174e0")}</span><input class="destructive-confirm-toggle" type="checkbox" data-destructive-toggle aria-label="${jvH("ui.0114e3d64460")}"></label>`;
            form.appendChild(confirmation);
          }
          actions.append(cancel, submit);
          form.append(error, actions);
          form.addEventListener("submit", async (event) => {
            event.preventDefault();
            if (submit.disabled || (dangerous && (!form.querySelector("[data-destructive-toggle]").checked || Number(form.querySelector("[data-destructive-slider]").value) < 100))) return;
            submit.disabled = true;
            cancel.disabled = true;
            const preventClose = (event) => event.preventDefault();
            dialog.addEventListener("cancel", preventClose);
            try {
              const fields = Object.fromEntries(new FormData(form).entries());
              const result = await systemPostForm("/api/system/workspace/users/" + user.id + "/" + action, fields);
              if (result) {
                dialog.close();
                await renderUsers(++loadSequence, result.message || jvT("ui.6107d3a04c7b"));
              }
            } catch (failure) {
              error.textContent = failure?.message || jvT("ui.09d234f0346f");
              error.hidden = false;
            } finally {
              cancel.disabled = false;
              dialog.removeEventListener("cancel", preventClose);
              if (dangerous) form.querySelector("[data-destructive-confirmation]")._justVoxelDestructiveSync();
              else submit.disabled = false;
            }
          });
          wrap.append(heading, form);
          dialog.appendChild(wrap);
          document.body.appendChild(dialog);
          initializeSharedWebUIControls(dialog);
          dialog.addEventListener("close", () => { dialog.remove(); trigger.focus(); }, { once: true });
          dialog.showModal();
        };
        const addAction = (title, fields, action, submitText, dangerous = false, disabled = false) => {
          const button = document.createElement("button");
          button.type = "button";
          button.textContent = title;
          button.disabled = disabled;
          if (dangerous) button.className = "storage-action-danger";
          button.addEventListener("click", () => openAction(title, fields, action, submitText, dangerous));
          choices.appendChild(button);
        };
        addAction(jvT("ui.a43a8d8cfd29"), '<label>Role<select name="role"><option value="operator"' + (user.role === "operator" ? ' selected' : '') + '>Operator</option><option value="viewer"' + (user.role !== "operator" ? ' selected' : '') + '>Viewer</option></select></label>', "role", jvT("ui.c7523b152e8b"));
        addAction(jvT("ui.a683c212b7ef"), `<p>${jvH("ui.6bf4d1978e20")}</p>`, "restart-allowance/reset", jvT("ui.37769ec42cac"), false, Number(user.restart_used || 0) === 0);
        addAction(jvT("ui.3ad28e785fb3"), `<p>${jvH("ui.2abbefb55ee4")}</p>`, "backup-allowance/reset", jvT("ui.37769ec42cac"), false, Number(user.backup_used || 0) === 0);
        addAction(user.enabled ? jvT("ui.9f024ada2282") : jvT("ui.1965185f3488"), '<input type="hidden" name="enabled" value="' + String(!user.enabled) + '"><p>' + (user.enabled ? jvT("ui.29ee543445e6") : jvT("ui.67022675441b")) + '</p>', "enabled", user.enabled ? jvT("ui.9f024ada2282") : jvT("ui.1965185f3488"), Boolean(user.enabled));
        addAction(jvT("ui.3f9c991f63a9"), `<label>${jvH("ui.3dd9df4441fb")}<input type="password" name="password" required autocomplete="new-password"></label><label>${jvH("ui.5ac265f396a2")}<input type="password" name="confirm_password" required autocomplete="new-password"></label>`, "password", jvT("ui.3f9c991f63a9"));
        addAction(jvT("ui.a2e20a335700"), `<p>${jvH("ui.0eb640efdff4")}</p>`, "delete", jvT("ui.a2e20a335700"), true);
        menu.append(trigger, choices);
        cell.appendChild(menu);
        row.appendChild(cell);
        rows.appendChild(row);
      });
      listPanel.appendChild(list);
    }
    root.appendChild(listPanel);
    content.replaceChildren(root);
  };

  const renderSecurity = async (sequence, message = "") => {
    const payload = await systemFetchJSON("/api/system/workspace/security");
    if (!payload || sequence !== loadSequence || !content) return;
    const root = document.createElement("div");
    root.className = "system-security-view";
    if (message) root.appendChild(systemNotice(message));

    const switcher = document.createElement("div");
    switcher.className = "system-security-switcher";
    switcher.setAttribute("role", "tablist");
    ["authentication", "password"].forEach((kind) => {
      const button = document.createElement("button");
      button.type = "button";
      button.dataset.systemSecurityChoice = kind;
      button.textContent = kind === "authentication" ? jvT("ui.66880d2d8216") : jvT("ui.e7cf3ef4f17c");
      button.setAttribute("aria-selected", securityPane === kind ? "true" : "false");
      switcher.appendChild(button);
    });
    const securityHeading = document.createElement("div");
    securityHeading.className = "system-security-heading";
    const securityLabel = document.createElement("p");
    securityLabel.className = "eyebrow";
    securityLabel.textContent = jvT("ui.1d85c11e1444");
    const securityTitle = document.createElement("h2");
    securityTitle.textContent = jvT("ui.8f6fb4eb7f42");
    securityHeading.append(securityLabel, securityTitle, switcher);
    root.appendChild(securityHeading);

    const authPane = document.createElement("section");
    authPane.className = "system-security-pane";
    authPane.dataset.systemSecurityPane = "authentication";
    const authHeading = document.createElement("div");
    authHeading.className = "system-security-pane-heading";
    const authTitle = document.createElement("h3");
    authTitle.textContent = jvT("ui.66880d2d8216");
    const authDescription = document.createElement("p");
    authDescription.className = "muted compact";
    authDescription.textContent = jvT("ui.d9daa98b51f6");
    const authLabel = document.createElement("p");
    authLabel.className = "eyebrow";
    authLabel.textContent = jvT("ui.2bb30a7e4603");
    authHeading.append(authLabel, authTitle, authDescription);
    authPane.appendChild(authHeading);

    const authStatus = document.createElement("div");
    authStatus.className = "system-security-status";
    const admin = document.createElement("span");
    admin.textContent = jvT("ui.7cd369e12222") + (payload.username || "voxel");
    const mode = document.createElement("span");
    mode.textContent = jvT("ui.6d155e134de0") + (payload.mode === "system" ? jvT("ui.a87b81d3be0d") : jvT("ui.1df936df289f"));
    authStatus.append(admin, mode);
    authPane.appendChild(authStatus);

    const authForm = document.createElement("form");
    authForm.className = "system-security-form";
    if (payload.mode === "system") {
      authForm.innerHTML =
        '<input type="hidden" name="mode" value="separate">' +
        `<label>${jvH("ui.a99e117fb426")}<input type="password" name="system_password" autocomplete="current-password" required></label>` +
        `<label>${jvH("ui.344ad9fcdb09")}<input type="password" name="new_web_password" autocomplete="new-password" required></label>` +
        `<label>${jvH("ui.1db54d10c829")}<input type="password" name="confirm_web_password" autocomplete="new-password" required></label>` +
        `<p class="muted compact">${jvH("ui.9b90a0542675")}</p>` +
        `<button type="submit">${jvH("ui.9036a3beaa8e")}</button>`;
      authForm.querySelectorAll('input[name="new_web_password"],input[name="confirm_web_password"]').forEach((input) => {
        input.minLength = Number(payload.minimum_password_len || 8);
      });
    } else {
      authForm.innerHTML =
        '<input type="hidden" name="mode" value="system">' +
        `<label>${jvH("ui.b88e3dc0276e")}<input type="password" name="system_password" autocomplete="current-password" required></label>` +
        `<p class="muted compact">${jvH("ui.eaf5cde5b815")}</p>` +
        `<button type="submit">${jvH("ui.95bb800209fc")}</button>`;
    }
    authForm.addEventListener("submit", async (event) => {
      event.preventDefault();
      const submit = event.submitter;
      if (submit) submit.disabled = true;
      try {
        const data = Object.fromEntries(new FormData(authForm).entries());
        const result = await systemPostForm("/api/system/workspace/security/authentication", data);
        if (!result) return;
        if (result.reauthenticate) {
          window.location.assign("/login?message=auth-mode-changed");
          return;
        }
        await renderSecurity(++loadSequence, result.message || jvT("ui.db5611ae62d9"));
      } catch (error) {
        if (state) state.textContent = error?.message || jvT("ui.f5adad2c4d22");
        if (submit) submit.disabled = false;
      }
    });
    authPane.appendChild(authForm);
    root.appendChild(authPane);

    const passwordPane = document.createElement("section");
    passwordPane.className = "system-security-pane";
    passwordPane.dataset.systemSecurityPane = "password";
    const passwordHeading = document.createElement("div");
    passwordHeading.className = "system-security-pane-heading";
    const passwordTitle = document.createElement("h3");
    passwordTitle.textContent = jvT("ui.e7cf3ef4f17c");
    const passwordDescription = document.createElement("p");
    passwordDescription.className = "muted compact";
    passwordDescription.textContent = jvT("ui.f954e2d8e432");
    const passwordLabel = document.createElement("p");
    passwordLabel.className = "eyebrow";
    passwordLabel.textContent = jvT("ui.7da69e0ebd60");
    passwordHeading.append(passwordLabel, passwordTitle, passwordDescription);
    passwordPane.appendChild(passwordHeading);

    const passwordForm = document.createElement("form");
    passwordForm.className = "system-security-form";
    passwordForm.innerHTML =
      `<label>${jvH("ui.72ed2bd767ce")}<input type="password" name="current_password" autocomplete="current-password" required></label>` +
      `<label>${jvH("ui.3dd9df4441fb")}<input type="password" name="new_password" autocomplete="new-password" required></label>` +
      `<label>${jvH("ui.bf000421aeb3")}<input type="password" name="confirm_password" autocomplete="new-password" required></label>` +
      `<p class="muted compact">${jvH("ui.0d33e45397d7")}` + String(payload.minimum_password_len || 8) + ' characters and follow the host password policy.</p>' +
      `<button type="submit">${jvH("ui.3f9c991f63a9")}</button>`;
    passwordForm.querySelectorAll('input[name="new_password"],input[name="confirm_password"]').forEach((input) => {
      input.minLength = Number(payload.minimum_password_len || 8);
    });
    passwordForm.addEventListener("submit", async (event) => {
      event.preventDefault();
      const submit = event.submitter;
      if (submit) submit.disabled = true;
      try {
        const data = Object.fromEntries(new FormData(passwordForm).entries());
        const result = await systemPostForm("/api/system/workspace/security/password", data);
        if (!result) return;
        if (result.reauthenticate) {
          window.location.assign("/login?message=password-changed");
          return;
        }
      } catch (error) {
        if (state) state.textContent = error?.message || jvT("ui.c1683634fe84");
        if (submit) submit.disabled = false;
      }
    });
    passwordPane.appendChild(passwordForm);
    root.appendChild(passwordPane);

    const showPane = (kind) => {
      securityPane = kind;
      root.querySelectorAll("[data-system-security-choice]").forEach((button) => {
        button.setAttribute("aria-selected", button.dataset.systemSecurityChoice === kind ? "true" : "false");
      });
      root.querySelectorAll("[data-system-security-pane]").forEach((pane) => {
        pane.hidden = pane.dataset.systemSecurityPane !== kind;
      });
    };
    root.querySelectorAll("[data-system-security-choice]").forEach((button) => {
      button.addEventListener("click", () => showPane(button.dataset.systemSecurityChoice));
    });
    content.replaceChildren(root);
    showPane(securityPane);
  };

  const clearResetPoll = () => {
    if (resetPollTimer) window.clearTimeout(resetPollTimer);
    resetPollTimer = 0;
  };

  const systemResetList = (title, items, kind = "") => {
    const box = document.createElement("section");
    box.className = "system-reset-impact " + kind;
    const heading = document.createElement("h4");
    heading.textContent = title;
    const list = document.createElement("ul");
    items.forEach((item) => {
      const row = document.createElement("li");
      row.textContent = item;
      list.appendChild(row);
    });
    box.append(heading, list);
    return box;
  };

  const systemResetOperationLabel = (operation) => {
    if (!operation) return jvT("ui.a92f0449a9f7");
    if (operation.state === "queued") return jvT("ui.661ff40a07e0");
    if (operation.state === "validating") return jvT("ui.5a1a167e3d90");
    if (operation.state === "running") return jvT("ui.2740f15dd768");
    if (operation.state === "verifying") return jvT("ui.a7d4a9f301ed");
    if (operation.state === "succeeded") return jvT("ui.143b270a3260");
    if (operation.state === "needs_attention") return jvT("ui.c1ebc7817870");
    if (operation.state === "resolved") return jvT("ui.5be3c2c8354e");
    if (operation.state === "rolled_back") return jvT("ui.71592236d852");
    return jvT("ui.a92f0449a9f7");
  };

  const renderResetOperation = (operation) => {
    clearResetPoll();
    if (!content || !operation) return;
    window.JustVoxelLiveConsole.updateState(operation.operation_id, operation.state, operation.stage);
    if (operation.operation_type === "factory_reset" && operation.state === "succeeded") {
      showFactoryResetComplete();
      return;
    }
    activeFactoryResetOperationID = operation.operation_type === "factory_reset" &&
      !["needs_attention", "resolved", "rolled_back"].includes(operation.state)
      ? operation.operation_id : "";
    if (operation.operation_type === "minecraft_reset" && operation.state === "succeeded") {
      ++loadSequence;
      window.JustVoxelLiveConsole.close();
      workspaceWindow?.close();
      if (!operation.instance_id) {
        const target = new URL(window.location.href);
        target.searchParams.delete("server");
        window.history.replaceState(window.history.state, "", target);
        globalThis.justVoxelSelectionReady = null;
        for (const workspace of serverWorkspaces) {
          delete workspaceServerSelections[workspace];
          writeWorkspaceWindowState(workspace, {selectedServer: ""});
          ++globalThis.justVoxelSelectionGeneration[workspace];
        }
        const dashboard = document.querySelector("#dashboard[data-dashboard-status]");
        if (dashboard) dashboard.dataset.selectedMinecraftId = "";
        document.querySelectorAll("[data-server-selector-slot]").forEach((slot) => {
          slot.replaceChildren();
          slot.dataset.inventoryGeneration = String(globalThis.justVoxelSelectionGeneration[slot.dataset.serverWorkspace]);
          delete slot.dataset.selectionLoading;
          void globalThis.justVoxelServerSelection(slot);
        });
        for (const workspace of serverWorkspaces) {
          window.dispatchEvent(new CustomEvent("justvoxel:server-selection-changed", { detail: { serverId: "", workspace, inventoryEmpty: true } }));
        }
      }
      window.dispatchEvent(new Event("justvoxel:minecraft-reset-complete"));
      return;
    }
    const root = document.createElement("div");
    root.className = "system-reset-view";

    const built = systemPanel(
      operation.operation_type === "factory_reset" ? jvT("ui.20b6891d2ddb") : jvT("ui.979e2ada0bd5"),
      systemResetOperationLabel(operation)
    );
    built.panel.classList.add("system-reset-progress");
    const badge = document.createElement("span");
    badge.className = "badge";
    badge.textContent = systemResetOperationLabel(operation);
    built.heading.appendChild(badge);

    const status = document.createElement("p");
    status.className = "system-reset-progress-status";
    status.textContent = operation.status || jvT("ui.7cd4cd6b5a1f");
    const stage = document.createElement("p");
    stage.className = "muted compact";
    stage.textContent = "Stage: " + String(operation.stage || "working").replaceAll("_", " ");
    built.panel.append(status, stage);
    const liveConsole = document.createElement("button");
    liveConsole.type = "button";
    liveConsole.className = "secondary";
    liveConsole.textContent = jvT("ui.7cac54a8cfc3");
    liveConsole.addEventListener("click", () => window.JustVoxelLiveConsole.open({
      kind: operation.operation_type,
      operationID: operation.operation_id,
      state: operation.state, stage: operation.stage,
      onUnauthorized: () => resetLiveConsoleUnauthorized(operation),
    }));
    if (operation.operation_type !== "factory_reset") built.panel.appendChild(liveConsole);

    if (operation.operation_type === "factory_reset") {
      const note = document.createElement("div");
      note.className = "notice warning";
      note.textContent = jvT("ui.be18d8f28f2d");
      built.panel.appendChild(note);
      const consoleActions = document.createElement("div");
      consoleActions.className = "action-row";
      consoleActions.appendChild(liveConsole);
      built.panel.appendChild(consoleActions);
      if (operation.state === "needs_attention") {
        const recovery = document.createElement("div");
        recovery.className = "notice warning";
        recovery.innerHTML = `<strong>${jvH("ui.86a736100f8c")}</strong><br>${jvH("ui.07bf9377a4c7")}`;
        built.panel.appendChild(recovery);

        const actions = document.createElement("div");
        actions.className = "system-reset-actions";
        const keep = document.createElement("button");
        keep.type = "button";
        keep.className = "secondary";
        keep.textContent = jvT("ui.bf52e7deba92");
        keep.addEventListener("click", async () => {
          if (!window.confirm(jvT("ui.6d795dedffb9"))) return;
          keep.disabled = true;
          retry.disabled = true;
          if (state) state.textContent = jvT("ui.71be1fccb60a");
          try {
            const result = await systemPostForm("/api/system/workspace/reset/factory/resolve", {
              operation_id: operation.operation_id,
              keep_current_state: "yes",
            });
            if (result?.operation) renderResetOperation(result.operation);
            else await renderReset(++loadSequence);
          } catch (error) {
            if (state) state.textContent = error?.message || jvT("ui.7cfc47e3f260");
            keep.disabled = false;
            retry.disabled = false;
          }
        });

        const retry = document.createElement("button");
        retry.type = "button";
        retry.className = "danger";
        retry.textContent = jvT("ui.adc684dd72fc");
        retry.addEventListener("click", () => planReset("factory"));
        actions.append(keep, retry);
        built.panel.appendChild(actions);
      }
    } else if (operation.state === "needs_attention") {
      const recovery = document.createElement("div");
      recovery.className = "notice warning";
      recovery.innerHTML = operation.instance_id
        ? `<strong>${jvH("ui.b17d34c8a367")}</strong><br>${jvH("ui.17474bcfbcd2")}`
        : `<strong>${jvH("ui.4034dd9654dc")}</strong><br>${jvH("ui.a1800c07f600")}`;
      built.panel.appendChild(recovery);

      const actions = document.createElement("div");
      actions.className = "system-reset-actions";
      const retry = document.createElement("button");
      retry.type = "button";
      retry.className = "danger";
      retry.textContent = operation.instance_id ? jvT("ui.6f93a2571e2c") : jvT("ui.658afd0803c0");
      retry.addEventListener("click", () => planReset("minecraft", true));
      actions.appendChild(retry);
      built.panel.appendChild(actions);
    }

    root.appendChild(built.panel);
    content.replaceChildren(root);

    const terminal = ["succeeded", "needs_attention", "resolved", "rolled_back"].includes(operation.state);
    if (terminal) {
      if (state) state.textContent = operation.status || systemResetOperationLabel(operation);
      return;
    }

    resetPollTimer = window.setTimeout(async () => {
      if (currentTab !== "reset" || !content?.isConnected) return;
      try {
        const operationURL = "/api/system/workspace/reset/operations/" + operation.operation_id;
        const payload = await systemFetchJSON(operation.instance_id ? operationURL + "?server=" + encodeURIComponent(operation.instance_id) : operationURL, {},
          operation.operation_type === "factory_reset" ? operation.operation_id : "");
        if (payload?.operation) renderResetOperation(payload.operation);
      } catch (error) {
        if (state) state.textContent = error?.message || jvT("ui.1823a62fdb5c");
        resetPollTimer = window.setTimeout(() => renderReset(++loadSequence), 2500);
      }
    }, 1800);
  };

  const resetCurrentOperation = async () => {
    const factory = await systemFetchJSON("/api/system/workspace/reset/factory/current");
    if (factory?.operation) return factory.operation;
    const minecraft = await systemFetchJSON("/api/system/workspace/reset/minecraft/current");
    return minecraft?.operation || null;
  };

  const resetImpactFromPlan = (mode, plan) => {
    const remove = [];
    const keep = [];

    if (mode === "minecraft" && plan.instance_id) {
      remove.push(jvT("ui.07f376ec2908"));
      if (plan.data_action === "delete") remove.push(jvT("ui.abf24b64455b"));
      else keep.push(jvT("ui.3a8217bf821e"));
      if (plan.plugins_action === "remove_managed_jars" || plan.plugins_action === "delete_with_internal_data") remove.push(jvT("ui.24315f4740a8"));
      if (plan.data_action === "preserve") keep.push(jvT("ui.b107008010b1"));
      keep.push(jvT("ui.d25e93ba058d"));
      keep.push(jvT("ui.c2e406c7e1cf"));
      keep.push(jvT("ui.44fa8600908d"));
      keep.push(jvT("ui.27f9efecdccc"));
    } else if (mode === "minecraft") {
      remove.push(jvT("ui.e5094f800296"));
      remove.push(jvT("ui.7aab4edb3e3c"));
      remove.push(jvT("ui.b66192cbf9e0"));
      keep.push(jvT("ui.f60cea039966"));
      keep.push(jvT("ui.76ad39a83c43"));
      keep.push(jvT("ui.1616c5a17af3"));
      keep.push(jvT("ui.5e9ae5e6318e"));
      keep.push(jvT("ui.81bbea99d4fa"));
    } else {
      remove.push(jvT("ui.07f376ec2908"));
      if (plan.data_action === "delete") remove.push(jvT("ui.abf24b64455b"));
      else if (plan.data_action === "preserve") keep.push(jvT("ui.3a8217bf821e"));
      if (plan.backup_action === "delete") remove.push(jvT("ui.6c50abe6706e"));
      else if (plan.backup_action === "preserve") keep.push(jvT("ui.fdbf31f6f3fe"));
      remove.push(jvT("ui.31cdff1c5446"));
      remove.push(jvT("ui.b4d4c139dc36"));
      remove.push(jvT("ui.6c0e6c6ade8d"));
      remove.push(jvT("ui.c416ea72b8a6"));
      keep.push(jvT("ui.27f9efecdccc"));
      remove.push(jvT("ui.37f46b894b6b"));
      keep.push(jvT("ui.ce7188063e70"));
      keep.push(jvT("ui.d26be76b1f69"));
    }
    return { remove, keep };
  };

  const renderResetPlan = (mode, plan, requireSystemPassword = false) => {
    clearResetPoll();
    if (!content) return;
    const root = document.createElement("div");
    root.className = "system-reset-view";
    const title = mode === "factory" ? jvT("ui.20b6891d2ddb") : jvT("ui.979e2ada0bd5");
    const built = systemPanel(jvT("ui.0dcd15832c6f"), title);
    built.panel.classList.add("system-reset-review");

    const description = document.createElement("p");
    description.className = "muted compact";
    description.textContent = mode === "factory"
      ? jvT("ui.abaa34fb73a4")
      : plan.instance_id ? jvT("ui.c2633366024a")
      : jvT("ui.4f77c4766406");
    built.panel.appendChild(description);

    if (mode === "minecraft" && Array.isArray(plan.participants)) {
      built.panel.appendChild(systemResetList(jvT("ui.e48ae12029e2") + plan.participants.length,
        plan.participants.map((participant) => participant.display_name + " — " +
          ({ running: jvT("ui.f4ccae29e1bb"), stopped: jvT("ui.1a4f630ac1b6"), failed: jvT("ui.031a8f0f659d"), retiring: jvT("ui.9f2c59a113db") }[participant.state] || participant.state))));
    }
    const impact = resetImpactFromPlan(mode, plan);
    const impactGrid = document.createElement("div");
    impactGrid.className = "system-reset-impact-grid";
    impactGrid.append(
      systemResetList(jvT("ui.620c483f98f8"), impact.remove, "danger"),
      systemResetList(jvT("ui.6dfb4a89d46b"), impact.keep, "safe")
    );
    built.panel.appendChild(impactGrid);

    if (Array.isArray(plan.warnings) && plan.warnings.length) {
      const warning = document.createElement("div");
      warning.className = "notice warning";
      warning.textContent = plan.warnings.join(" ");
      built.panel.appendChild(warning);
    }

    const online = Number(plan.players_online || 0);
    let playersConfirm = null;
    if (online > 0) {
      const players = document.createElement("div");
      players.className = "notice warning";
      const names = Array.isArray(plan.players) && plan.players.length ? " (" + plan.players.join(", ") + ")" : "";
      players.textContent = String(online) + " player(s) are online" + names + ". Reset will disconnect them.";
      built.panel.appendChild(players);

      const playerLabel = document.createElement("label");
      playerLabel.className = "system-reset-check";
      playersConfirm = document.createElement("input");
      playersConfirm.type = "checkbox";
      const copy = document.createElement("span");
      copy.textContent = jvT("ui.6ec1430cca33");
      playerLabel.append(playersConfirm, copy);
      built.panel.appendChild(playerLabel);
    }

    let password = null;
    if (mode === "factory" && requireSystemPassword) {
      const passwordLabel = document.createElement("label");
      passwordLabel.className = "system-reset-password";
      passwordLabel.textContent = jvT("ui.b46bd6419d35");
      password = document.createElement("input");
      password.type = "password";
      password.autocomplete = "current-password";
      password.required = true;
      passwordLabel.appendChild(password);
      const help = document.createElement("small");
      help.className = "muted";
      help.textContent = jvT("ui.1c3074505e65");
      passwordLabel.appendChild(help);
      built.panel.appendChild(passwordLabel);
    }

    const arm = document.createElement("div");
    arm.className = "destructive-confirmation";
    const sliderShell = document.createElement("div");
    sliderShell.className = "destructive-confirm-slider";
    const armedText = document.createElement("span");
    armedText.className = "destructive-confirm-slider-text";
    armedText.textContent = jvT("ui.68184139e694");
    const thumb = document.createElement("span");
    thumb.className = "destructive-confirm-slider-thumb";
    thumb.setAttribute("aria-hidden", "true");
    thumb.textContent = jvT("ui.62b67e1f685b");
    const slider = document.createElement("input");
    slider.type = "range";
    slider.min = "0";
    slider.max = "100";
    slider.step = "1";
    slider.value = "0";
    slider.setAttribute("aria-label", jvT("ui.68184139e694"));
    sliderShell.append(armedText, thumb, slider);

    const finalLabel = document.createElement("label");
    finalLabel.className = "destructive-confirm-toggle-row";
    finalLabel.hidden = true;
    const finalCopy = document.createElement("span");
    finalCopy.className = "destructive-confirm-toggle-callout";
    finalCopy.textContent = jvT("ui.8ba7a99174e0");
    const finalConfirm = document.createElement("input");
    finalConfirm.className = "destructive-confirm-toggle";
    finalConfirm.type = "checkbox";
    finalLabel.append(finalCopy, finalConfirm);
    arm.append(sliderShell, finalLabel);
    built.panel.appendChild(arm);

    const actions = document.createElement("div");
    actions.className = "system-reset-actions";
    const cancel = document.createElement("button");
    cancel.type = "button";
    cancel.className = "secondary";
    cancel.textContent = jvT("ui.19766ed6ccb2");
    const apply = document.createElement("button");
    apply.type = "button";
    apply.className = "danger";
    apply.textContent = mode === "factory" ? jvT("ui.20b6891d2ddb") : jvT("ui.979e2ada0bd5");
    apply.disabled = true;
    actions.append(cancel, apply);
    built.panel.appendChild(actions);

    const syncApply = () => {
      const progress = Math.max(0, Math.min(100, Number(slider.value || 0)));
      sliderShell.style.setProperty("--confirm-progress", String(progress / 100));
      const armed = progress >= 100;
      sliderShell.classList.toggle("is-armed", armed);
      armedText.textContent = armed ? jvT("ui.4b9e808b8a8f") : jvT("ui.68184139e694");
      finalLabel.hidden = !armed;
      if (!armed) finalConfirm.checked = false;
      const playersOK = !playersConfirm || playersConfirm.checked;
      const passwordOK = !password || password.value.length > 0;
      apply.disabled = !(armed && finalConfirm.checked && playersOK && passwordOK);
    };

    slider.addEventListener("input", syncApply);
    finalConfirm.addEventListener("change", syncApply);
    playersConfirm?.addEventListener("change", syncApply);
    password?.addEventListener("input", syncApply);
    cancel.addEventListener("click", () => renderResetChoices());

    apply.addEventListener("click", async () => {
      apply.disabled = true;
      cancel.disabled = true;
      slider.disabled = true;
      finalConfirm.disabled = true;
      if (playersConfirm) playersConfirm.disabled = true;
      if (password) password.disabled = true;
      if (state) state.textContent = jvT("ui.0dbd144fff0f");
      try {
        const fields = {
          plan_fingerprint: plan.plan_fingerprint,
          confirm_players: online > 0 ? "yes" : "no",
        };
        if (password) fields.system_password = password.value;
        const endpoint = mode === "factory"
          ? "/api/system/workspace/reset/factory/apply"
          : "/api/system/workspace/reset/minecraft/apply";
        const result = await systemPostForm(endpoint, fields);
        if (password) password.value = "";
        if (!result?.operation) throw new Error(jvT("ui.91e689223f70"));
        renderResetOperation(result.operation);
      } catch (error) {
        if (password) password.value = "";
        if (state) state.textContent = error?.message || jvT("ui.31c5cbfa01f2");
        cancel.disabled = false;
        slider.disabled = false;
        if (playersConfirm) playersConfirm.disabled = false;
        if (password) password.disabled = false;
        syncApply();
      }
    });

    root.appendChild(built.panel);
    content.replaceChildren(root);
    syncApply();
  };

  const planReset = async (mode, retry = false) => {
    if (state) state.textContent = jvT("ui.80c65200649f");
    try {
      if (mode === "minecraft" && !retry) {
        const current = await systemFetchJSON("/api/system/workspace/reset/minecraft/current");
        if (!current) return;
        if (current.operation) {
          renderResetOperation(current.operation);
          return;
        }
      }
      const endpoint = mode === "factory"
        ? "/api/system/workspace/reset/factory/plan"
        : "/api/system/workspace/reset/minecraft/plan";
      const plan = await systemPostForm(endpoint);
      if (!plan) return;
      let requireSystemPassword = false;
      if (mode === "factory") {
        const security = await systemFetchJSON("/api/system/workspace/security");
        if (!security) return;
        requireSystemPassword = security.mode === "separate";
      }
      renderResetPlan(mode, plan, requireSystemPassword);
      if (state) state.textContent = "";
    } catch (error) {
      if (state) state.textContent = error?.message || jvT("ui.fc7fc36b9b26");
    }
  };

  const renderResetChoices = () => {
    clearResetPoll();
    activeFactoryResetOperationID = "";
    if (!content) return;
    const root = document.createElement("div");
    root.className = "system-reset-view";

    const intro = document.createElement("div");
    intro.className = "system-reset-intro";
    const title = document.createElement("h3");
    title.textContent = jvT("ui.6b9e567376a6");
    const copy = document.createElement("p");
    copy.className = "muted compact";
    copy.textContent = jvT("ui.c1378564b9e6");
    intro.append(title, copy);
    root.appendChild(intro);

    const choices = document.createElement("div");
    choices.className = "system-reset-choices";

    const minecraft = document.createElement("button");
    minecraft.type = "button";
    minecraft.className = "system-reset-choice";
    minecraft.innerHTML =
      `<strong>${jvH("ui.979e2ada0bd5")}</strong>` +
      `<span>${jvH("ui.201aef0a5b69")}</span>` +
      '<small>Keeps backups, login/password, WebUI users, and storage layout.</small>';
    minecraft.addEventListener("click", () => planReset("minecraft"));

    const factory = document.createElement("button");
    factory.type = "button";
    factory.className = "system-reset-choice danger";
    factory.innerHTML =
      `<strong>${jvH("ui.20b6891d2ddb")}</strong>` +
      `<span>${jvH("ui.c4e157eca063")}</span>` +
      '<small>Deletes local Minecraft, local backups, WebUI users/history, and resets authentication. External/network storage stays untouched.</small>';
    factory.addEventListener("click", () => planReset("factory"));

    choices.append(minecraft, factory);
    root.appendChild(choices);
    content.replaceChildren(root);
  };

  const renderReset = async (sequence) => {
    clearResetPoll();
    const operation = await resetCurrentOperation();
    if (sequence !== loadSequence || currentTab !== "reset") return;
    if (operation) {
      renderResetOperation(operation);
      return;
    }
    renderResetChoices();
  };

  const renderAbout = async (sequence) => {
    const payload = await systemFetchJSON("/api/system/workspace/about");
    if (!payload || sequence !== loadSequence || !content) return;
    const root = document.createElement("div");
    root.className = "system-about-view";
    const variant = String(payload.variant || "").trim().toLowerCase();
    const variantName = ({ vm: "VM", hws: "HWS", "justvoxel-vm": "VM", "justvoxel-hws": "HWS" })[variant];
    const built = systemPanel(jvT("ui.bc75ba011e41"), variantName ? jvT("ui.446778e3c2f4", {v0: (variantName)}) : "JustVoxel");
    const list = document.createElement("dl");
    [
      ["JustVoxel", payload.justvoxel || "—"],
      ["WebUI", payload.webui || "—"],
      [jvT("ui.60e425d8c33f"), payload.management_api || "—"],
      [jvT("ui.0e570ca6fabe"), payload.commit || "—"],
      [jvT("ui.abc50e334be4"), (payload.username || "—") + " (" + (payload.role || "—") + ")"],
    ].forEach(([label, value]) => {
      const row = document.createElement("div");
      const dt = document.createElement("dt");
      dt.textContent = label;
      const dd = document.createElement("dd");
      dd.textContent = value;
      row.append(dt, dd);
      list.appendChild(row);
    });
    built.panel.appendChild(list);
    const externalLink = (text, url) => {
      const link = document.createElement("a");
      link.href = url;
      link.target = "_blank";
      link.rel = "noopener noreferrer";
      link.textContent = text;
      return link;
    };
    const project = document.createElement("div");
    project.className = "about-project";
    const projectHeading = document.createElement("h3");
    projectHeading.textContent = jvT("ui.985959785319");
    const projectList = document.createElement("dl");
    [
      [jvT("ui.13d6ff07b8a5"), jvT("ui.0e5efa875595"), "https://github.com/home-server-project/justvoxel"],
      [jvT("ui.d764d42592fc"), jvT("ui.299150b6bf15"), "https://github.com/home-server-project"],
      [jvT("ui.103ba8ff257f"), jvT("ui.635ae219ad13"), "https://github.com/home-server-project/justvoxel/issues"],
    ].forEach(([label, text, url]) => {
      const row = document.createElement("div");
      const dt = document.createElement("dt");
      dt.textContent = label;
      const dd = document.createElement("dd");
      dd.appendChild(externalLink(text, url));
      row.append(dt, dd);
      projectList.appendChild(row);
    });
    project.append(projectHeading, projectList);
    built.panel.appendChild(project);
    const appendLinkSection = (title, rows) => {
      const section = document.createElement("div");
      section.className = "about-project";
      const heading = document.createElement("h3");
      heading.textContent = title;
      const links = document.createElement("dl");
      rows.forEach(([label, entries]) => {
        const row = document.createElement("div");
        const dt = document.createElement("dt");
        dt.textContent = label;
        const dd = document.createElement("dd");
        entries.forEach(([text, url], index) => {
          if (index) dd.append(" · ");
          dd.appendChild(externalLink(text, url));
        });
        row.append(dt, dd);
        links.appendChild(row);
      });
      section.append(heading, links);
      built.panel.appendChild(section);
    };
    appendLinkSection(jvT("ui.e1ec9dbb6512"), [
      ["CLI", [
        ["btop", "https://github.com/aristocratos/btop"],
        ["Superfile", "https://github.com/yorukot/superfile"],
        ["just", "https://github.com/casey/just"],
        ["gum", "https://github.com/charmbracelet/gum"],
        ["fzf", "https://github.com/junegunn/fzf"],
        [jvT("ui.63f1711dd65a"), "https://github.com/ublue-os/packages/tree/main/packages/ublue-os-just"],
      ]],
      ["WebUI", [
        ["ttyd", "https://github.com/tsl0922/ttyd"],
        [jvT("ui.33b8a7901c02"), "https://github.com/gtsteffaniak/filebrowser"],
        ["Glances", "https://github.com/nicolargo/glances"],
      ]],
    ]);
    appendLinkSection(jvT("ui.66354a1d3225"), [
      ["VPN / Mesh", [
        ["Tailscale", "https://tailscale.com/"],
        ["NetBird", "https://netbird.io/"],
        [jvT("ui.efbb9e0df755"), "https://nordvpn.com/meshnet/"],
      ]],
      [jvT("ui.2d55598fd54b"), [
        ["Playit.gg", "https://playit.gg/"],
      ]],
    ]);
    const disclaimer = document.createElement("p");
    disclaimer.className = "disclaimer";
    disclaimer.textContent = jvT("ui.fffb0ebaa589");
    root.append(built.panel, disclaimer);
    content.replaceChildren(root);
  };

  const systemUPSStateLabel = (value) => {
    const labels = {
      online: jvT("ui.0d21bd52022c"),
      on_battery: jvT("ui.51d530448a96"),
      low_battery: jvT("ui.a894de3c08a2"),
      bypass: jvT("ui.bf26c2ac087d"),
      unknown: jvT("ui.b764cdc0eab7"),
    };
    return labels[value] || jvT("ui.b764cdc0eab7");
  };

  const systemUPSNumber = (value, suffix = "") => {
    if (value === null || value === undefined || Number.isNaN(Number(value))) return "—";
    const number = Number(value);
    const digits = Math.abs(number % 1) > 0.001 ? 1 : 0;
    return number.toFixed(digits) + suffix;
  };

  const systemUPSRuntime = (seconds) => {
    if (seconds === null || seconds === undefined || Number(seconds) < 0) return "—";
    const total = Math.round(Number(seconds));
    const hours = Math.floor(total / 3600);
    const minutes = Math.floor((total % 3600) / 60);
    if (hours > 0) return hours + "h " + String(minutes).padStart(2, "0") + "m";
    return minutes + "m";
  };

  const systemUPSMetric = (label, value) => {
    const item = document.createElement("div");
    item.className = "system-ups-metric";
    const name = document.createElement("span");
    name.textContent = label;
    const data = document.createElement("strong");
    data.textContent = value;
    item.append(name, data);
    return item;
  };

  const openUPSOptions = (snapshot, selectedMode) => {
    if (!snapshot.available) return;
    const dialog = document.createElement("dialog");
    dialog.className = "system-action-dialog";
    dialog.setAttribute("aria-label", jvT("ui.37cdc3330e3e"));
    const shell = document.createElement("div");
    shell.className = "system-action-dialog-content";
    const title = document.createElement("h2");
    title.textContent = jvT("ui.37cdc3330e3e");
    const settingField = (label, name, type, value) => {
      const field = document.createElement("label");
      field.textContent = label;
      const input = document.createElement("input");
      input.name = name;
      input.type = type;
      input.autocomplete = "off";
      if (type !== "password") input.value = value || "";
      field.appendChild(input);
      return { field, input };
    };
    const shutdownSection = document.createElement("section");
    shutdownSection.className = "system-ups-option-section";
    const shutdownTitle = document.createElement("h3");
    shutdownTitle.textContent = jvT("ui.82ea48e89c88");
    const shutdownToggle = settingField(jvT("ui.5f94bd70e4d9"), "enabled", "checkbox", "");
    shutdownToggle.field.classList.add("system-ups-shutdown-switch");
    shutdownToggle.input.setAttribute("role", "switch");
    shutdownToggle.input.checked = Boolean(snapshot.protection_enabled);
    const delay = settingField(jvT("ui.edcf437d2614"), "delay_seconds", "number", String(snapshot.shutdown_delay_seconds || 120));
    delay.input.min = "1";
    delay.input.max = "86400";
    const monitorUser = settingField(jvT("ui.3367ff7ddea7"), "monitor_username", "text", "");
    const monitorPassword = settingField(jvT("ui.ffd8e01ef6b3"), "monitor_password", "password", "");
    const shutdownNote = document.createElement("p");
    shutdownNote.textContent = snapshot.monitor_credentials_configured ? jvT("ui.8c04e3286a8e") : jvT("ui.97c480300d14");
    const shutdownState = document.createElement("p");
    shutdownState.textContent = jvT("ui.9051c4ee2df8") + (snapshot.monitor_service_active ? "active" : "inactive") + " · NUT mode: " + (snapshot.nut_mode || "none");
    const shutdownSave = document.createElement("button");
    shutdownSave.type = "button";
    shutdownSave.className = "secondary";
    shutdownSave.textContent = jvT("ui.b0617f4f2c48");
    shutdownSection.append(shutdownTitle, shutdownToggle.field, delay.field, monitorUser.field, monitorPassword.field, shutdownNote, shutdownState, shutdownSave);

    const sharingSection = document.createElement("section");
    sharingSection.className = "system-ups-option-section";
    const sharingAvailable = selectedMode === "local";
    const sharingTitle = document.createElement("h3");
    sharingTitle.textContent = jvT("ui.30fbb1558130");
    const sharingToggle = settingField(jvT("ui.99a40b9820c7"), "enabled", "checkbox", "");
    sharingToggle.field.classList.add("system-ups-shutdown-switch");
    sharingToggle.input.setAttribute("role", "switch");
    sharingToggle.input.checked = Boolean(snapshot.sharing_enabled);
    const listenAddress = settingField(jvT("ui.edc8737a7436"), "listen_address", "text", snapshot.sharing_listen_address);
    const listenPort = settingField(jvT("ui.72e9a59f5a1d"), "listen_port", "number", String(snapshot.sharing_listen_port || 3493));
    listenPort.input.min = "1";
    listenPort.input.max = "65535";
    const clientUser = settingField(jvT("ui.bfdba669e35c"), "client_username", "text", "");
    const clientPassword = settingField(jvT("ui.bfd9632986e9"), "client_password", "password", "");
    const sharingNote = document.createElement("p");
    sharingNote.textContent = snapshot.sharing_credentials_configured ? jvT("ui.8c04e3286a8e") : jvT("ui.97c480300d14");
    const sharingState = document.createElement("p");
    sharingState.textContent = jvT("ui.41adb152a3da") + (snapshot.server_service_active ? "active" : "inactive") + " · Driver service: " + (snapshot.driver_service_active ? "active" : "inactive");
    const sharingSave = document.createElement("button");
    sharingSave.type = "button";
    sharingSave.className = "secondary";
    sharingSave.textContent = jvT("ui.8b48365fe420");
    sharingSection.append(sharingTitle, sharingToggle.field, listenAddress.field, listenPort.field, clientUser.field, clientPassword.field, sharingNote, sharingState, sharingSave);
    if (!sharingAvailable) {
      sharingToggle.input.disabled = true;
      sharingSave.disabled = true;
    }

    const syncOptionFields = (toggle, fields) => {
      fields.forEach(({ input }) => { input.disabled = toggle.input.disabled || !toggle.input.checked; });
    };
    shutdownToggle.input.addEventListener("change", () => syncOptionFields(shutdownToggle, [delay, monitorUser, monitorPassword]));
    sharingToggle.input.addEventListener("change", () => syncOptionFields(sharingToggle, [listenAddress, listenPort, clientUser, clientPassword]));
    syncOptionFields(shutdownToggle, [delay, monitorUser, monitorPassword]);
    syncOptionFields(sharingToggle, [listenAddress, listenPort, clientUser, clientPassword]);

    const section = document.createElement("h3");
    section.textContent = jvT("ui.d5b59156cd37");
    const copy = document.createElement("p");
    copy.textContent = jvT("ui.4b7b5eec1fa3");
    const error = document.createElement("p");
    error.className = "system-action-dialog-error";
    error.hidden = true;
    const actions = document.createElement("div");
    actions.className = "action-row";
    const cancel = document.createElement("button");
    cancel.type = "button";
    cancel.className = "secondary";
    cancel.textContent = jvT("ui.19766ed6ccb2");
    const forget = document.createElement("button");
    forget.type = "button";
    forget.className = "danger";
    forget.textContent = jvT("ui.e48602c3b6ec");
    actions.append(cancel, forget);
    const optionColumns = document.createElement("div");
    optionColumns.className = "system-ups-option-columns";
    optionColumns.append(shutdownSection, sharingSection);
    shell.append(title, optionColumns, section, copy, error, actions);
    dialog.appendChild(shell);
    document.body.appendChild(dialog);

    const close = () => {
      if (dialog.open) dialog.close();
      dialog.remove();
    };
    const saveOptions = async (path, toggle, fields, button, fallback) => {
      error.hidden = true;
      button.disabled = true;
      try {
        const body = new URLSearchParams({ csrf: upsCSRF?.value || "", enabled: String(toggle.input.checked) });
        if (toggle.input.checked) fields.forEach(({ input }) => body.set(input.name, input.value));
        const response = await systemFetchPage(path, {
          method: "POST",
          headers: { Accept: "application/json", "Content-Type": "application/x-www-form-urlencoded" },
          body: body.toString(),
        });
        if (!response) return;
        const result = await response.json().catch(() => ({}));
        if (!response.ok) throw new Error(result.error || fallback);
        close();
        upsAvailable = Boolean(result.available);
        if (upsTabButton) upsTabButton.hidden = !upsAvailable;
        renderUPSMarkup(result, await loadIdentity());
        if (state) state.textContent = "";
      } catch (failure) {
        error.textContent = failure?.message || fallback;
        error.hidden = false;
      } finally {
        button.disabled = false;
      }
    };
    shutdownSave.addEventListener("click", () => saveOptions("/api/ups/shutdown", shutdownToggle, [delay, monitorUser, monitorPassword], shutdownSave, jvT("ui.8b744d028a63")));
    sharingSave.addEventListener("click", () => saveOptions("/api/ups/sharing", sharingToggle, [listenAddress, listenPort, clientUser, clientPassword], sharingSave, jvT("ui.ea29e8a14d06")));
    cancel.addEventListener("click", close);
    dialog.addEventListener("cancel", (event) => {
      event.preventDefault();
      if (!forget.disabled) close();
    });
    let confirmed = false;
    forget.addEventListener("click", async () => {
      if (!confirmed) {
        confirmed = true;
        title.textContent = jvT("ui.49e234b02d72");
        copy.textContent = jvT("ui.bcd73e408099");
        forget.textContent = jvT("ui.bb0f9ad6dc33");
        forget.focus();
        return;
      }
      forget.disabled = true;
      cancel.disabled = true;
      error.hidden = true;
      try {
        const body = new URLSearchParams({ csrf: upsCSRF?.value || "" });
        const response = await systemFetchPage("/api/ups/source/forget", {
          method: "POST",
          headers: { Accept: "application/json", "Content-Type": "application/x-www-form-urlencoded" },
          body: body.toString(),
        });
        if (!response) return;
        const result = await response.json().catch(() => ({}));
        if (!response.ok) throw new Error(result.error || jvT("ui.58c4983a2560"));
        close();
        upsAvailable = Boolean(result.available);
        if (upsTabButton) upsTabButton.hidden = !upsAvailable;
        renderUPSMarkup(result, await loadIdentity());
        if (state) state.textContent = "";
      } catch (failure) {
        error.textContent = failure?.message || jvT("ui.58c4983a2560");
        error.hidden = false;
      } finally {
        forget.disabled = false;
        cancel.disabled = false;
      }
    });
    dialog.showModal();
  };

  const renderUPSMarkup = (snapshot, user) => {
    if (!content) return;
    const root = document.createElement("div");
    root.className = "system-ups-overview";

    const sourceBar = document.createElement("div");
    sourceBar.className = "system-ups-source";
    const sourceLabel = document.createElement("span");
    sourceLabel.textContent = jvT("ui.0e570ca6fabe");
    const sourceValue = document.createElement("strong");
    sourceValue.textContent = snapshot.source?.mode === "remote"
      ? (snapshot.source.host + ":" + snapshot.source.port)
      : jvT("ui.1b8548de762c");
    sourceBar.append(sourceLabel, sourceValue);
    root.appendChild(sourceBar);

    if (snapshot.message) {
      const notice = document.createElement("div");
      notice.className = "notice system-ups-notice";
      notice.textContent = snapshot.message;
      root.appendChild(notice);
    }

    if (Array.isArray(snapshot.devices) && snapshot.devices.length > 0) {
      const grid = document.createElement("div");
      grid.className = "system-ups-device-grid";
      snapshot.devices.forEach((device) => {
        const card = document.createElement("section");
        card.className = "system-ups-card";

        const heading = document.createElement("div");
        heading.className = "system-ups-card-heading";
        const titleWrap = document.createElement("div");
        const title = document.createElement("h3");
        title.textContent = device.display_name || device.name || "UPS";
        const technical = document.createElement("small");
        technical.textContent = device.name || "";
        titleWrap.append(title, technical);
        const status = document.createElement("span");
        status.className = "system-ups-status system-ups-status-" + String(device.state || "unknown");
        status.textContent = systemUPSStateLabel(device.state);
        heading.append(titleWrap, status);

        const metrics = document.createElement("div");
        metrics.className = "system-ups-metrics";
        metrics.append(
          systemUPSMetric(jvT("ui.dfcb7c1619d5"), systemUPSNumber(device.battery_charge, "%")),
          systemUPSMetric(jvT("ui.109311589787"), systemUPSRuntime(device.battery_runtime_seconds)),
          systemUPSMetric(jvT("ui.8a6bdb6b18da"), systemUPSNumber(device.load_percent, "%")),
          systemUPSMetric(jvT("ui.36ecb4f86691"), systemUPSNumber(device.input_voltage, " V")),
          systemUPSMetric(jvT("ui.b2439bcb8dee"), systemUPSNumber(device.output_voltage, " V")),
          systemUPSMetric(jvT("ui.2c8fcd20a5e2"), systemUPSNumber(device.input_frequency, " Hz")),
          systemUPSMetric(jvT("ui.1f6efbfdd9ce"), systemUPSNumber(device.output_frequency, " Hz")),
          systemUPSMetric(jvT("ui.39d6c91b57ce"), systemUPSNumber(device.battery_voltage, " V")),
          systemUPSMetric(jvT("ui.b958ce8b871a"), systemUPSNumber(device.temperature, " °C"))
        );
        card.append(heading, metrics);
        grid.appendChild(card);
      });
      root.appendChild(grid);
    }

    if (snapshot.available && user?.role === "administrator") {
      const panel = document.createElement("section");
      panel.className = "system-ups-source-panel";
      const heading = document.createElement("h3");
      heading.textContent = jvT("ui.eac87dc0ea26");
      const help = document.createElement("p");
      help.textContent = jvT("ui.1e9fd473e150");

      const form = document.createElement("form");
      form.method = "post";
      form.action = "/api/ups/source";
      form.className = "system-ups-source-form";

      const csrf = document.createElement("input");
      csrf.type = "hidden";
      csrf.name = "csrf";
      csrf.value = upsCSRF?.value || "";

      const modeLabel = document.createElement("label");
      modeLabel.textContent = jvT("ui.0e570ca6fabe");
      const mode = document.createElement("select");
      mode.name = "mode";
      const localOption = document.createElement("option");
      localOption.value = "local";
      localOption.textContent = jvT("ui.7db10d4467d7");
      const remoteOption = document.createElement("option");
      remoteOption.value = "remote";
      remoteOption.textContent = jvT("ui.77b3d2250b31");
      mode.append(localOption, remoteOption);
      mode.value = snapshot.source?.mode === "remote" ? "remote" : "local";
      modeLabel.appendChild(mode);

      const hostLabel = document.createElement("label");
      hostLabel.textContent = jvT("ui.87cfa1ecec4b");
      const host = document.createElement("input");
      host.name = "host";
      host.type = "text";
      host.autocomplete = "off";
      host.placeholder = jvT("ui.8137545135e4");
      host.value = snapshot.source?.host || "";
      hostLabel.appendChild(host);

      const portLabel = document.createElement("label");
      portLabel.textContent = jvT("ui.72e9a59f5a1d");
      const port = document.createElement("input");
      port.name = "port";
      port.type = "number";
      port.min = "1";
      port.max = "65535";
      port.value = String(snapshot.source?.port || 3493);
      portLabel.appendChild(port);

      const nameLabel = document.createElement("label");
      nameLabel.textContent = jvT("ui.107e7b866610");
      const name = document.createElement("input");
      name.name = "ups_name";
      name.type = "text";
      name.autocomplete = "off";
      name.value = snapshot.source?.ups_name || "";
      nameLabel.appendChild(name);

      const driverLabel = document.createElement("label");
      driverLabel.textContent = jvT("ui.fa59b1e09a98");
      const driver = document.createElement("input");
      driver.name = "driver";
      driver.type = "text";
      driver.autocomplete = "off";
      driver.value = snapshot.source?.driver || "";
      driverLabel.appendChild(driver);

      const deviceLabel = document.createElement("label");
      deviceLabel.textContent = jvT("ui.64ee297b558e");
      const device = document.createElement("input");
      device.name = "device_port";
      device.type = "text";
      device.autocomplete = "off";
      device.value = snapshot.source?.device_port || "";
      deviceLabel.appendChild(device);

      const save = document.createElement("button");
      save.type = "submit";
      save.className = "button primary";
      save.textContent = jvT("ui.4b75a25f8609");

      const actions = document.createElement("div");
      actions.className = "system-ups-source-actions";
      const options = document.createElement("button");
      options.type = "button";
      options.className = "secondary";
      options.textContent = jvT("ui.d0db8b5e364b");
      options.addEventListener("click", () => openUPSOptions(snapshot, mode.value));
      actions.append(options, save);
      const finalRow = document.createElement("div");
      finalRow.className = "system-ups-source-final-row";
      finalRow.append(nameLabel, actions);

      const syncRemoteFields = () => {
        const remote = mode.value === "remote";
        host.disabled = !remote;
        port.disabled = !remote;
        driver.disabled = remote;
        device.disabled = remote;
        hostLabel.hidden = !remote;
        portLabel.hidden = !remote;
        driverLabel.hidden = remote;
        deviceLabel.hidden = remote;
        host.required = remote;
      };
      mode.addEventListener("change", syncRemoteFields);
      syncRemoteFields();

      form.append(csrf, modeLabel, hostLabel, portLabel, driverLabel, deviceLabel, finalRow);
      panel.append(heading, help, form);
      root.appendChild(panel);
    }

    content.replaceChildren(root);
  };

  const fetchUPSStatus = async () => {
    const response = await fetch("/api/ups", {
      method: "GET",
      credentials: "same-origin",
      headers: { Accept: "application/json" },
      cache: "no-store",
    });
    if (systemHandleAuth(response)) return null;
    if (!response.ok) throw new Error(jvT("ui.c262d0b07856"));
    return response.json();
  };

  const refreshUPSCapability = async () => {
    const snapshot = await fetchUPSStatus();
    if (!snapshot) return null;
    upsAvailable = Boolean(snapshot.available);
    if (upsTabButton) upsTabButton.hidden = !upsAvailable;
    if (!upsAvailable && currentTab === "ups") {
      currentTab = identity?.role === "administrator" ? "health" : "history";
      syncSystemTabs();
    }
    return snapshot;
  };

  const loadUPS = async (sequence) => {
    const snapshot = await fetchUPSStatus();
    if (!snapshot || sequence !== loadSequence) return;
    upsAvailable = Boolean(snapshot.available);
    if (upsTabButton) upsTabButton.hidden = !upsAvailable;
    if (!upsAvailable) throw new Error(jvT("ui.f30c7422f23e"));
    renderUPSMarkup(snapshot, await loadIdentity());
  };

  const submitUPSSource = async (form) => {
    const body = new URLSearchParams();
    new FormData(form).forEach((value, key) => body.append(key, String(value)));
    const submitter = form.querySelector('button[type="submit"]');
    if (submitter) submitter.disabled = true;
    if (state) state.textContent = jvT("ui.6c5d7ea5a0a6");
    try {
      const response = await systemFetchPage("/api/ups/source", {
        method: "POST",
        headers: {
          Accept: "application/json",
          "Content-Type": "application/x-www-form-urlencoded",
        },
        body: body.toString(),
      });
      if (!response) return;
      const result = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(result.error || jvT("ui.8d6f6ead6906"));
      upsAvailable = Boolean(result.available);
      if (upsTabButton) upsTabButton.hidden = !upsAvailable;
      renderUPSMarkup(result, await loadIdentity());
      if (state) state.textContent = "";
    } catch (error) {
      if (state) state.textContent = error?.message || jvT("ui.8d6f6ead6906");
    } finally {
      if (submitter && submitter.isConnected) submitter.disabled = false;
    }
  };

  const settingsPanels = new Map(Array.from(document.querySelectorAll("[data-system-settings-panel]"), (panel) =>
    [panel.dataset.systemSettingsPanel, { panel, home: panel.parentNode }]));
  const parkSettingsPanels = () => {
    settingsPanels.forEach(({ panel, home }) => {
      panel.hidden = true;
      home.appendChild(panel);
    });
  };

  const loadCurrentSystemTab = async () => {
    if (window.JustVoxelWallpaper?.canLeave() === false) return;
    window.JustVoxelWallpaper?.leave();
    parkSettingsPanels();
    const sequence = ++loadSequence;
    if (state) state.textContent = jvT("ui.ba3bbbe10d8b");
    if (refreshButton) refreshButton.disabled = true;
    if (content) content.replaceChildren();

    try {
      const user = await loadIdentity();
      if (!user) return;
      if (sequence !== loadSequence) return;
      if (administratorTabs.has(currentTab) && user.role !== "administrator") {
        currentTab = "history";
        syncSystemTabs();
      }
      if (currentTab === "health") await renderHealth(sequence);
      else if (currentTab === "history") await renderHistory(sequence);
      else if (currentTab === "users") await renderUsers(sequence);
      else if (currentTab === "security") await renderSecurity(sequence);
      else if (currentTab === "logs") await renderLogs(sequence);
      else if (settingsPanels.has(currentTab)) {
        const { panel } = settingsPanels.get(currentTab);
        if (sequence !== loadSequence) return;
        panel.hidden = false;
        content.appendChild(panel);
        if (currentTab === "date-time") await window.JustVoxelDateTime?.refresh();
        else window.JustVoxelWallpaper?.prepare();
      }
      else if (currentTab === "reset") await renderReset(sequence);
      else if (currentTab === "ups") await loadUPS(sequence);
      else await renderAbout(sequence);
      if (sequence === loadSequence && state) state.textContent = "";
    } catch (error) {
      if (sequence !== loadSequence) return;
      if (content) content.replaceChildren();
      if (state) state.textContent = error?.message || jvT("ui.6b27df281793");
    } finally {
      if (sequence === loadSequence && refreshButton) refreshButton.disabled = false;
    }
  };

  const selectSystemTab = async (tab) => {
    if (window.JustVoxelWallpaper?.canLeave() === false) return;
    const user = await loadIdentity();
    if (!user) return;
    if (administratorTabs.has(tab) && user.role !== "administrator") return;
    if (tab === "ups" && !upsAvailable) return;
    if (currentTab === "reset" && tab !== "reset") clearResetPoll();
    failedUnitsSelected = false;
    currentTab = tab;
    syncSystemTabs();
    loadCurrentSystemTab();
  };

  content?.addEventListener("submit", async (event) => {
    const form = event.target.closest("form");
    if (!form || !content.contains(form)) return;
    const action = new URL(form.getAttribute("action") || window.location.href, window.location.href);
    if (action.pathname !== "/api/ups/source") return;
    event.preventDefault();
    await submitUPSSource(form);
  });

  const workspaceWindow = setupWorkspaceWindow(systemWorkspaceDialog, {
    onClose: () => { ++loadSequence; failedUnitsSelected = false; window.JustVoxelWallpaper?.leave(); clearResetPoll(); },
    onOpen: async () => {
      try {
        const user = await loadIdentity();
        if (!user) return;
        await refreshUPSCapability();
        if (!initialized) {
          initialized = true;
          const target = new URLSearchParams(window.location.search);
          const requested = target.get("workspace") === "system" ? target.get("tab") : "";
          currentTab = user.role === "administrator" ? "health" : "history";
          if (allowedSystemTabs.has(requested) && (requested !== "ups" || upsAvailable) &&
              (!administratorTabs.has(requested) || user.role === "administrator")) currentTab = requested;
          syncSystemTabs();
        }
        loadCurrentSystemTab();
      } catch (error) {
        if (state) state.textContent = error?.message || jvT("ui.6b27df281793");
      }
    },
  });

  window.JustVoxelSystem = {
    openPluginLogs: async (id) => {
      if (!/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/.test(id)) return;
      requestedMigrationLog = { group: jvT("ui.9514b7ff4860"), id };
      await window.JustVoxelSystem.openTab("logs");
    },
    openTab: async (tab) => {
      if (!allowedSystemTabs.has(tab) || (tab === "ups" && !upsAvailable)) return;
      if (window.JustVoxelWallpaper?.canLeave() === false) return;
      const user = await loadIdentity();
      if (!user || administratorTabs.has(tab) && user.role !== "administrator") return;
      initialized = true;
      currentTab = tab;
      syncSystemTabs();
      if (controlCenter) controlCenter.open = false;
      workspaceWindow?.open();
    },
  };

  systemWorkspaceOpen.addEventListener("click", () => {
    if (controlCenter) controlCenter.open = false;
    workspaceWindow?.open();
  });
  closeButton?.addEventListener("click", () => {
    if (window.JustVoxelWallpaper?.canLeave() === false) return;
    window.JustVoxelWallpaper?.leave();
    clearResetPoll();
    workspaceWindow?.close();
  });
  refreshButton?.addEventListener("click", () => loadCurrentSystemTab());
  tabs.forEach((button) => button.addEventListener("click", () => selectSystemTab(button.dataset.systemTab)));

  if (readWorkspaceWindowState("system").open) workspaceWindow?.open();
}

// Plugin workspace. All local inventory and catalog access stays
// behind the Administrator-only WebUI and Management Agent routes.
const pluginsDialog = document.querySelector("[data-plugins-workspace-dialog]");
const pluginsOpen = document.querySelector("[data-plugins-open]");
if (pluginsDialog && pluginsOpen) {
  const form = pluginsDialog.querySelector("[data-plugins-search]");
  const results = pluginsDialog.querySelector("[data-plugins-results]");
  const state = pluginsDialog.querySelector("[data-plugins-state]");
  globalThis.justVoxelServerSelection(pluginsDialog.querySelector("[data-server-selector-slot]")).catch(error => { if (state) state.textContent = error.message; });
  const pagination = pluginsDialog.querySelector("[data-plugins-pagination]");
  const previous = pluginsDialog.querySelector("[data-plugins-previous]");
  const next = pluginsDialog.querySelector("[data-plugins-next]");
  const pageLabel = pluginsDialog.querySelector("[data-plugins-page]");
  let totalPages = 1, discoverServerKey = null;
  let providerServerKey = null, acceptedServer = null;
  const contentPlural = () => acceptedServer?.content_kind === "mods" ? "mods" : acceptedServer?.content_kind === "plugins" ? "plugins" : "plugins or mods";
  const contentSingular = () => acceptedServer?.content_kind === "mods" ? "mod" : "plugin";
  let pageLimit = 8, capacityTimer = null;
  const firstCursor = () => ({ modrinth_offset: 0, hangar_offset: 0, curseforge_offset: 0, provider_turn: 0, hangar_first: false });
  let pageCursors = [firstCursor()], pageIndex = 0;
  const resetCursors = () => { pageCursors = [firstCursor()]; pageIndex = 0; cursor = firstCursor(); nextCursor = null; totalPages = 1; pageLabel.textContent = jvT("ui.d6e19ce6ca28"); };
  let tab = "discover", cursor = firstCursor(), nextCursor = null, generation = 0, controller = null;
  const text = (tag, value, className = "") => {
    const node = document.createElement(tag);
    node.textContent = value;
    node.className = className;
    return node;
  };
  const pluginBadge = (value, semanticClass = "") => text("span", value, `plugins-badge ${semanticClass}`.trim());
  const installedStatus = () => pluginBadge(jvT("ui.f8b32f4e92bd"), "plugins-badge-installed plugins-overlay-installed");
  const identityTile = (name, projectID, iconURL, source = "Modrinth") => {
    const tile = text("span", Array.from(name || "?").slice(0, 2).join("").toUpperCase(), "plugins-icon-tile");
    tile.setAttribute("aria-hidden", "true");
    tile.dataset.tone = String(Array.from(name || "?").reduce((sum, char) => sum + char.codePointAt(0), 0) & 3);
    // Search logos get one metadata retry; absent search logos start with metadata.
    if (["Modrinth", "Hangar", "CurseForge"].includes(source) && typeof projectID === "string" && /^[A-Za-z0-9]{1,64}$/.test(projectID)) {
      const image = document.createElement("img");
      image.alt = "";
      image.decoding = "async";
      image.hidden = true;
      const query = new URLSearchParams({ source, project_id: projectID });
      if (source === "Modrinth" && typeof iconURL === "string" && iconURL) query.set("icon_url", iconURL);
      let metadataAttempt = !query.has("icon_url");
      let finished = false;
      image.addEventListener("load", () => { finished = true; tile.classList.add("has-logo"); image.hidden = false; }, { once: true });
      const requestImage = () => { image.src = `/api/plugins/workspace/icon?${query}`; };
      image.addEventListener("error", () => {
        if (finished) return;
        image.hidden = true;
        tile.classList.remove("has-logo");
        if (!metadataAttempt) {
          metadataAttempt = true;
          query.delete("icon_url");
          requestImage();
        } else {
          finished = true;
          image.remove();
        }
      });
      requestImage();
      tile.append(image);
    }
    return tile;
  };
  const overlayHeader = (name, description, badges, projectID, action = null, iconURL, source = "Modrinth") => {
    const header = document.createElement("header");
    header.className = "plugins-overlay-header";
    const identity = document.createElement("div");
    identity.className = "plugins-overlay-identity";
    identity.append(identityTile(name, projectID, iconURL, source), text("h3", name));
    if (action) identity.append(action);
    header.append(identity);
    if (description) header.append(text("p", description, "plugins-overlay-description"));
    const meta = document.createElement("div");
    meta.className = "plugins-card-badges";
    badges.filter((badge) => Array.isArray(badge) ? badge[0] : badge).forEach((badge) => meta.append(Array.isArray(badge) ? pluginBadge(...badge) : pluginBadge(badge)));
    header.append(meta);
    panelBody.append(header);
  };
  const overlaySection = (label, area = panelBody) => {
    const section = document.createElement("section");
    section.className = "plugins-overlay-section";
    section.append(text("h4", label));
    area.append(section);
    return section;
  };
  const overlayActions = (...buttons) => {
    const footer = document.createElement("footer");
    footer.className = "plugins-overlay-actions";
    footer.append(...buttons);
    panelBody.append(footer);
    return footer;
  };
  const interactiveCard = (card, name, activate, projectID, iconURL, source = "Modrinth") => {
    card.tabIndex = 0;
    card.setAttribute("role", "button");
    card.setAttribute("aria-label", jvT("ui.863533cb89d4", {v0: (name)}));
    card.setAttribute("aria-haspopup", "dialog");
    card.setAttribute("aria-controls", "plugins-overlay");
    card.setAttribute("aria-pressed", "false");
    const open = () => {
      results.querySelectorAll(".plugins-result-card").forEach((node) => {
        node.classList.toggle("is-selected", node === card);
        node.setAttribute("aria-pressed", String(node === card));
      });
      card.focus();
      activate();
    };
    card.addEventListener("click", open);
    card.addEventListener("keydown", (event) => {
      if (event.key === "Enter" || event.key === " ") {
        event.preventDefault();
        open();
      }
    });
    const heading = card.querySelector("h3");
    const header = document.createElement("div");
    header.className = "plugins-card-heading";
    card.insertBefore(header, heading);
    const copy = document.createElement("div");
    copy.className = "plugins-card-copy";
    copy.append(heading);
    const description = card.querySelector(".plugins-card-description, .plugins-card-release");
    if (description) copy.append(description);
    header.append(identityTile(name, projectID, iconURL, source), copy);
  };
  const addInfo = (card, name, renderInfo, projectID, iconURL, source = "Modrinth") => {
    interactiveCard(card, name, () => {
      stopPolling();
      showPanel(jvT("ui.86d2b0959a05", {v0: (name)}), true);
      const infoActions = renderInfo();
      const actions = Array.isArray(infoActions) ? infoActions : [];
      overlayActions(...actions, panelAction(jvT("ui.7d9eb7acb13e"), closePanel));
    }, projectID, iconURL, source);
  };
  const infoRows = (container) => {
    const rows = document.createElement("dl");
    rows.className = "plugins-info-rows";
    container.append(rows);
    return (label, value) => {
      const row = document.createElement("div");
      row.className = "plugins-info-row";
      row.append(text("dt", label));
      const cell = document.createElement("dd");
      if ([jvT("ui.e511470b21a9"), jvT("ui.6e7fdd059d0f")].includes(label)) cell.className = "plugins-identity-metadata";
      if (typeof value === "string") cell.textContent = value;
      else cell.append(value);
      row.append(cell);
      rows.append(row);
    };
  };
  const dependencyGroups = (dependencies) => {
    const groups = document.createElement("div");
    groups.className = "plugins-dependency-groups";
    for (const [type, label] of [["required", jvT("ui.4850b174b713")], ["optional", jvT("ui.59be71333c96")], ["incompatible", jvT("ui.8777e3a9e3ec")], ["other", jvT("ui.3b115742f0d2")]]) {
      const entries = dependencies.filter((dependency) => type === "other" ? !["required", "optional", "incompatible"].includes(dependency.dependency_type) : dependency.dependency_type === type);
      if (!entries.length) continue;
      const row = document.createElement("p");
      row.className = "plugins-publisher-dependency-row";
      row.append(text("strong", `${label} publisher dependencies`));
      const values = entries.map((dependency) => {
        const identity = dependency.name || dependency.title || dependency.file_name || dependency.project_id || jvT("ui.5b8aa0bf3c59");
        return dependency.version_id ? `${identity} (release ${dependency.version_id})` : identity;
      });
      row.append(text("span", values.join(" / "), "plugins-dependency-ids"));
      groups.append(row);
    }
    if (!dependencies.length) groups.append(text("span", jvT("ui.177fe76745f5")));
    return groups;
  };
  const renderProject = (project, server) => {
    const card = document.createElement("article");
    card.className = "plugins-result-card";
    card.append(text("h3", project.title), text("p", project.description, "plugins-card-description"));
    const version = project.version;
    const badges = document.createElement("div");
    badges.className = "plugins-card-badges";
    const platform = server.server_software || server.server_type;
    const installedState = project.installed_state === "installed" ? "Present" : project.installed_state === "enabled" ? jvT("ui.92c1cdfdf4cb") : project.installed_state === "disabled" ? jvT("ui.75081b593d15") : jvT("ui.c1ebc7817870");
    const identityBadges = [[version?.version_number, "plugins-badge-version"], [platform, "plugins-badge-server"], [server.minecraft_version && `MC ${server.minecraft_version}`, "plugins-badge-minecraft"]];
    if (project.installed) {
      identityBadges.push([jvT("ui.f8b32f4e92bd"), "plugins-badge-installed"], [installedState, ["enabled", "installed"].includes(project.installed_state) ? "plugins-badge-enabled" : project.installed_state === "disabled" ? "plugins-badge-disabled" : "plugins-badge-attention"]);
    }
    identityBadges.filter(([value]) => value).forEach((badge) => badges.append(pluginBadge(...badge)));
    if (!version || project.metadata_error) card.append(text("p", jvT("ui.8ca66b8237f6"), "plugins-card-meta"));
    card.append(badges);
    const dependencies = version?.dependencies || [];
    if (dependencies.length) {
      const counts = new Map();
      dependencies.forEach((dependency) => {
        const kind = ["required", "optional", "incompatible", "embedded"].includes(dependency.dependency_type) ? dependency.dependency_type : "unspecified";
        counts.set(kind, (counts.get(kind) || 0) + 1);
      });
      card.append(text("p", Array.from(counts, ([kind, count]) => `${count} ${kind}`).join(" · "), "plugins-card-meta"));
    }
    const renderInfo = () => {
      overlayHeader(project.title, project.description, identityBadges, iconProjectID, project.installed && !managed ? installedStatus() : null, project.icon_url || "", project.source);
      const about = infoRows(overlaySection(jvT("ui.63e615c21158")));
      about(jvT("ui.e020e3c67bd0"), version?.version_number || jvT("ui.ca1844969742"));
      about(project.source === "Hangar" ? "Channel" : jvT("ui.0f9bdaf90d25"), version?.version_type || jvT("ui.d83b9ff3c07a"));
      about(jvT("ui.2ef42ec21d93"), version?.date_published || jvT("ui.d83b9ff3c07a"));
      about(jvT("ui.0e570ca6fabe"), project.source);
      about(jvT("ui.e511470b21a9"), project.project_id);
      about(jvT("ui.6e7fdd059d0f"), version?.id || jvT("ui.ca1844969742"));
      const compatibility = infoRows(overlaySection(jvT("ui.b37ddf1f6e0f")));
      compatibility(jvT("ui.6e7d50e84f47"), project.compatibility);
      compatibility(jvT("ui.8b8ee2755acb"), platform || jvT("ui.b764cdc0eab7"));
      compatibility(jvT("ui.de8921369774"), server.minecraft_version || jvT("ui.b764cdc0eab7"));
      compatibility(jvT("ui.67635dab0dde"), (version?.loaders || []).join(" / ") || jvT("ui.d83b9ff3c07a"));
      const versions = [...new Set(version?.game_versions || [])];
      compatibility(jvT("ui.aeb9d7cea04c"), versions.join(" / ") || jvT("ui.d83b9ff3c07a"));
      overlaySection(jvT("ui.2e41b118eb20")).append(dependencyGroups(dependencies));
      if (project.installed) {
        const management = infoRows(overlaySection(jvT("ui.e51248773b9a")));
        management(jvT("ui.c7d3acc826a8"), jvT("ui.272e549bca6c"));
        management(jvT("ui.a3b50c476732"), installedState);
        overlaySection(jvT("ui.1e6bf669df09")).append(text("p", jvT("ui.3ac22f0a39de")));
      }
      if (project.metadata_error || !version) panelBody.append(text("p", project.metadata_error || jvT("ui.5d5a17c6fcee"), "plugins-card-meta"));
    };
    const iconProjectID = ["Modrinth", "Hangar", "CurseForge"].includes(project.source) ? project.project_id : undefined;
    const managed = project.source === "Modrinth" && ["geyser", "floodgate", "viaversion"].includes(project.slug);
    if (managed) badges.append(text("span", jvT("ui.9c83434eb959"), "plugins-badge"));
    const installable = ["Modrinth", "Hangar", "CurseForge"].includes(project.source) && !managed && !project.installed && (project.install_available || (server.content_kind === "mods" || project.source === "CurseForge") && project.review_available) && project.compatible && server.supported && server.minecraft_version && version?.id;
    let reviewRejection = "";
    const readOnlyReview = (reason) => {
      reviewRejection = reason;
      showPanel(project.title, true);
      renderInfo();
      if (reason) overlaySection(jvT("ui.c3fc54aa5390")).append(text("p", reason));
      overlayActions(panelAction(jvT("ui.7d9eb7acb13e"), closePanel));
    };
    if (installable) {
      interactiveCard(card, project.title, () => reviewRejection ? readOnlyReview(reviewRejection) : reviewPlugin(project.project_id, version.id, project, readOnlyReview), iconProjectID, project.icon_url || "", project.source);
    } else {
      addInfo(card, project.title, () => {
        renderInfo();
        if (managed) infoRows(overlaySection(jvT("ui.e51248773b9a")))(jvT("ui.920e413c7d41"), jvT("ui.9c83434eb959"));
      }, iconProjectID, project.icon_url || "", project.source);
    }
    // Publisher data uses textContent; artifact URLs never reach this browser.
    return card;
  };
  const renderInstalled = (plugin) => {
    const card = document.createElement("article");
    card.className = "plugins-result-card";
    const name = plugin.name || plugin.filename;
    const projectID = plugin.project_id || plugin.icon_project_id || undefined;
    const management = plugin.management === "justvoxel" ? jvT("ui.9c83434eb959") : plugin.management === "plugins" ? jvT("ui.272e549bca6c") : jvT("ui.c5e24354b2e8");
    card.append(text("h3", name), text("p", jvT("ui.6c75fb0f78f7", {v0: (plugin.version || jvT("ui.b764cdc0eab7"))}), "plugins-card-release"));
    const badges = document.createElement("div");
    badges.className = "plugins-card-badges";
    badges.append(text("span", management, "plugins-badge plugins-management"));
    const managed = acceptedServer?.content_kind === "plugins" && plugin.management === "plugins" && ["enabled", "disabled"].includes(plugin.state);
    const managedMod = acceptedServer?.content_kind === "mods" && plugin.management === "plugins" && plugin.state === "installed";
    const stateLabel = managed ? (plugin.state === "disabled" ? jvT("ui.75081b593d15") : jvT("ui.92c1cdfdf4cb")) : "";
    const stateClass = plugin.state === "disabled" ? "plugins-badge-disabled" : "plugins-badge-enabled";
    if (plugin.management === "plugins") badges.append(pluginBadge(jvT("ui.f8b32f4e92bd"), "plugins-badge-installed"));
    if (stateLabel) badges.append(pluginBadge(stateLabel, `plugins-state ${stateClass}`));
    card.append(badges);
    const attention = plugin.update_status?.startsWith("Attention:");
    if (attention) badges.append(pluginBadge(jvT("ui.c1ebc7817870"), "plugins-badge-attention"));
    const updateAvailable = plugin.update_status?.startsWith(jvT("ui.ff8b555d818f"));
    const updateStatus = updateAvailable ? plugin.update_status : attention ? plugin.update_status.replace(/; manual \/ unmanaged$/, "") : plugin.management === "justvoxel" ? jvT("ui.6e074e25be46") : plugin.management === "plugins" ? jvT("ui.3ac22f0a39de") : jvT("ui.b764cdc0eab7");
    addInfo(card, name, () => {
      overlayHeader(name, "", [[plugin.version || jvT("ui.e83ee049ebbd"), "plugins-badge-version"], management, [plugin.management === "plugins" && jvT("ui.f8b32f4e92bd"), "plugins-badge-installed"], [stateLabel, stateClass], [attention && jvT("ui.c1ebc7817870"), "plugins-badge-attention"]], projectID, plugin.management === "plugins" ? installedStatus() : null, undefined, plugin.source || "Modrinth");
      overlaySection("JAR").append(text("p", plugin.filename));
      overlaySection(jvT("ui.9eddf573cb50")).append(text("p", plugin.metadata_status || jvT("ui.b764cdc0eab7")));
      overlaySection(jvT("ui.e51248773b9a")).append(text("p", managedMod ? jvT("ui.ab11c60f3925") : acceptedServer?.content_kind === "mods" ? jvT("ui.28a63e5e85a9") : plugin.management === "justvoxel" ? jvT("ui.95dba138f6de") : plugin.management === "plugins" ? jvT("ui.2bbacb1b5b49") : jvT("ui.f2218e03ac7c")));
      overlaySection(jvT("ui.1e6bf669df09")).append(text("p", updateStatus));
      if (plugin.project_id || plugin.version_id) {
        const identity = infoRows(overlaySection(jvT("ui.c05702e16b20")));
        if (plugin.project_id) identity(jvT("ui.e511470b21a9"), plugin.project_id);
        if (plugin.version_id) identity(jvT("ui.6e7fdd059d0f"), plugin.version_id);
      }
      if (managedMod && !attention) return [pluginAction(jvT("ui.c3812fc4acb8"), () => reviewManagedPlugin(plugin, "remove"), "plugins-remove-action")];
      if (managed && !attention) {
        const action = plugin.state === "disabled" ? "enable" : "disable";
        return [pluginAction(action === "enable" ? "Enable" : jvT("ui.b7e3e4aa4257"), () => reviewManagedPlugin(plugin, action), "plugins-state-action"), pluginAction(jvT("ui.c3812fc4acb8"), () => reviewManagedPlugin(plugin, "remove"), "plugins-remove-action")];
      }
    }, projectID, undefined, plugin.source || "Modrinth");
    let summary = plugin.metadata_status === "Metadata read" ? jvT("ui.c6cae98c2564") : jvT("ui.602e4ec87412");
    if (updateAvailable) summary = plugin.update_status;
    if (attention) summary = jvT("ui.c1ebc7817870");
    const status = text("p", summary, "plugins-inventory-status");
    if (attention) status.classList.add("is-attention");
    card.append(status);
    return card;
  };
  // Review, progress, Info and completion reuse a single bounded overlay.
  const panel = document.createElement("section");
  panel.className = "plugins-install-panel";
  panel.id = "plugins-overlay";
  panel.tabIndex = -1;
  panel.hidden = true;
  panel.setAttribute("role", "dialog");
  panel.setAttribute("aria-label", `${contentSingular() === "mod" ? "Mod" : "Plugin"} installation`);
  panel.setAttribute("aria-modal", "true");
  const panelBody = document.createElement("div");
  panelBody.className = "plugins-install-content";
  panel.append(panelBody);
  pluginsDialog.append(panel);
  let installGeneration = 0, pollTimer = null, installController = null, returnFocus = null;
  let operationIdentity = null;
  let operationViewID = null;
  let searchTimer = null, inventoryTimer = null;
  const clearInventoryTimer = () => { clearTimeout(inventoryTimer); inventoryTimer = null; };
  const pluginOperationSettled = (operation) => !operation || ["succeeded", "rolled_back", "resolved"].includes(operation.state);
  let overlayDismissible = false;
  const stopPolling = () => { clearTimeout(pollTimer); pollTimer = null; installController?.abort(); ++installGeneration; };
  const workspaceShell = pluginsDialog.querySelector(".plugins-workspace-shell");
  const closePanel = (restoreFocus = true) => {
    if (panel.dataset.curseforgeBusy === "true") return;
    if (panel.classList.contains("plugins-curseforge-panel")) {
      panel.querySelectorAll("input").forEach(input => { input.value = ""; });
      panelBody.replaceChildren();
      panel.classList.remove("plugins-curseforge-panel");
    }
    stopPolling();
    panel.hidden = true;
    results.querySelectorAll(".plugins-result-card").forEach((node) => {
      node.classList.remove("is-selected");
      node.setAttribute("aria-pressed", "false");
    });
    operationViewID = null;
    overlayDismissible = false;
    workspaceShell.inert = false;
    if (!restoreFocus) return;
    if (returnFocus?.isConnected) returnFocus.focus();
    else pluginsDialog.querySelector('[data-plugins-tab][aria-selected="true"]')?.focus();
  };
  const showPanel = (label = `${contentSingular() === "mod" ? "Mod" : "Plugin"} installation`, dismissible = false, preserveIdentity = false) => {
    overlayDismissible = dismissible;
    const focusedLabel = panel.contains(document.activeElement) ? document.activeElement.textContent : null;
    if (panel.hidden) returnFocus = document.activeElement;
    panel.setAttribute("aria-label", label);
    panel.hidden = false;
    workspaceShell.inert = true;
    if (preserveIdentity) {
      // Retain the identity header (including its image) for this operation.
      Array.from(panelBody.children).slice(1).forEach((node) => node.remove());
    } else {
      operationViewID = null;
      panelBody.replaceChildren();
    }
    // Preserve keyboard position when authoritative operation polling rerenders.
    queueMicrotask(() => {
      if (panel.hidden) return;
      const target = focusedLabel && Array.from(panel.querySelectorAll("button,summary")).find((node) => node.textContent === focusedLabel);
      (target || panel).focus();
    });
  };
  const panelAction = (label, action, secondary = false) => {
    const button = text("button", label, secondary ? "secondary" : "");
    button.type = "button";
    button.addEventListener("click", action);
    return button;
  };
  const pluginAction = (label, action, style) => {
    const button = panelAction(label, action);
    button.classList.add(style);
    return button;
  };
  const curseForgeSwitch = pluginsDialog.querySelector("[data-plugins-curseforge]");
  const curseForgeSection = pluginsDialog.querySelector("[data-plugins-curseforge-section]");
  const curseForgeNotice = pluginsDialog.querySelector("[data-plugins-curseforge-notice]");
  const curseForgeInfo = pluginsDialog.querySelector("[data-plugins-curseforge-info]");
  const curseForgeKeyForm = pluginsDialog.querySelector("[data-curseforge-key-form]");
  const curseForgeKey = pluginsDialog.querySelector("[data-curseforge-key]");
  const curseForgeSave = pluginsDialog.querySelector("[data-curseforge-save]");
  const curseForgeFeedback = pluginsDialog.querySelector("[data-curseforge-key-feedback]");
  const curseForgeRemove = pluginsDialog.querySelector("[data-curseforge-remove]");
  let curseForgeConfigured = false, curseForgeNoticeTimer = null, curseForgeRevision = 0;
  const dismissCurseForgeNotice = () => {
    clearTimeout(curseForgeNoticeTimer);
    curseForgeNoticeTimer = null;
    curseForgeNotice.hidden = true;
  };
  const syncCurseForgeInfo = () => {
    curseForgeInfo.hidden = true;
  };
  const applyCurseForgeStatus = (status) => {
    const wasConfigured = curseForgeConfigured;
    curseForgeConfigured = status.configured === true;
    curseForgeSwitch.disabled = !acceptedServer?.providers?.curseforge;
    if (!wasConfigured && curseForgeConfigured && providerServerKey && acceptedServer?.providers?.curseforge) curseForgeSwitch.checked = true;
    if (!curseForgeConfigured) curseForgeSwitch.checked = false;
    else dismissCurseForgeNotice();
    pluginsDialog.querySelector("[data-curseforge-status]").textContent = curseForgeConfigured ? jvT("ui.84aebc69a1bf") : jvT("ui.dd1841d29502");
    pluginsDialog.querySelector("[data-curseforge-mask]").hidden = !curseForgeConfigured;
    curseForgeSave.textContent = curseForgeConfigured ? jvT("ui.f97925f1e14b") : jvT("ui.826590b6b7b2");
    curseForgeRemove.hidden = !curseForgeConfigured;
    syncCurseForgeInfo();
  };
  const curseForgeRequest = async (method = "GET", key) => {
    const options = { method, credentials: "same-origin", cache: "no-store", headers: { Accept: "application/json" } };
    if (method !== "GET") {
      options.body = new URLSearchParams({ csrf: document.querySelector('[data-minecraft-workspace-csrf]')?.value || document.querySelector('input[name="csrf"]')?.value || "" });
      if (method === "PUT") options.body.set("key", key);
    }
    const response = await fetch("/api/plugins/workspace/curseforge-api", options);
    const payload = await response.json();
    if (response.status === 401) window.location.assign("/login");
    if (!response.ok) throw new Error(jvT("ui.ba6d194b5702"));
    return payload;
  };
  const refreshCurseForgeStatus = async () => {
    const revision = ++curseForgeRevision;
    curseForgeSwitch.disabled = true;
    pluginsDialog.querySelector("[data-curseforge-mask]").hidden = true;
    curseForgeRemove.hidden = true;
    curseForgeSave.textContent = jvT("ui.826590b6b7b2");
    pluginsDialog.querySelector("[data-curseforge-status]").textContent = jvT("ui.2b0ed71bd135");
    try {
      const status = await curseForgeRequest();
      if (revision === curseForgeRevision) applyCurseForgeStatus(status);
    } catch {
      if (revision !== curseForgeRevision) return;
      curseForgeConfigured = false;
      curseForgeSwitch.checked = false;
      syncCurseForgeInfo();
      pluginsDialog.querySelector("[data-curseforge-status]").textContent = jvT("ui.0f2a38aac6eb");
    }
  };
  curseForgeSwitch.addEventListener("change", () => {
    if (curseForgeSwitch.checked && !curseForgeConfigured) {
      curseForgeSwitch.checked = false;
      dismissCurseForgeNotice();
      curseForgeNotice.hidden = false;
      const anchor = curseForgeSwitch.parentElement.getBoundingClientRect();
      const bounds = pluginsDialog.getBoundingClientRect();
      const width = curseForgeNotice.getBoundingClientRect().width;
      curseForgeNotice.style.left = `${Math.max(bounds.left + 8, Math.min(anchor.left, bounds.right - width - 8)) - anchor.left}px`;
      curseForgeNoticeTimer = setTimeout(dismissCurseForgeNotice, 15000);
    } else dismissCurseForgeNotice();
    syncCurseForgeInfo();
    clearTimeout(searchTimer); resetCursors(); load();
  });
  pluginsDialog.querySelector("[data-curseforge-notice-dismiss]").addEventListener("click", dismissCurseForgeNotice);
  pluginsDialog.addEventListener("keydown", (event) => {
    if (event.key === "Escape" && !curseForgeNotice.hidden) {
      event.preventDefault(); event.stopPropagation(); dismissCurseForgeNotice();
    }
  });
  curseForgeKeyForm.addEventListener("submit", async (event) => {
    event.preventDefault();
    if (curseForgeSave.disabled) return;
    if (!curseForgeKey.value.trim() || /[\u0000-\u001f\u007f-\u009f\u2028\u2029]/u.test(curseForgeKey.value)) {
      curseForgeFeedback.textContent = jvT("ui.f4a7b374e3e2");
      return;
    }
    const request = curseForgeRequest("PUT", curseForgeKey.value);
    curseForgeKey.value = "";
    curseForgeSave.disabled = true;
    ++curseForgeRevision;
    try {
      applyCurseForgeStatus(await request);
      curseForgeFeedback.textContent = "";
    } catch {
      curseForgeFeedback.textContent = jvT("ui.bca0219e87f0");
    } finally { curseForgeSave.disabled = false; }
  });
  pluginsDialog.querySelector("[data-curseforge-notice-configure]").addEventListener("click", () => {
    dismissCurseForgeNotice();
    pluginsDialog.querySelector('[data-plugins-tab="curseforge-api"]').click();
    curseForgeKey.focus();
  });
  curseForgeRemove.addEventListener("click", () => {
    showPanel(jvT("ui.ccec3fcd8c6d"), true);
    panel.classList.add("plugins-curseforge-panel");
    panelBody.append(text("h3", jvT("ui.ccec3fcd8c6d")), text("p", jvT("ui.95c2e5967153")));
    const error = text("p", ""); error.setAttribute("role", "status"); panelBody.append(error);
    const cancel = panelAction(jvT("ui.19766ed6ccb2"), closePanel, true);
    const remove = panelAction(jvT("ui.ccec3fcd8c6d"), async () => {
      remove.disabled = true; cancel.disabled = true; panel.dataset.curseforgeBusy = "true";
      ++curseForgeRevision;
      try {
        applyCurseForgeStatus(await curseForgeRequest("DELETE"));
        curseForgeKey.value = "";
        curseForgeFeedback.textContent = "";
        dismissCurseForgeNotice(); delete panel.dataset.curseforgeBusy; closePanel();
      } catch { error.textContent = jvT("ui.df4e22b98312"); }
      finally { delete panel.dataset.curseforgeBusy; remove.disabled = false; cancel.disabled = false; }
    });
    overlayActions(cancel, remove);
  });
  const returnToInstalled = () => {
    closePanel();
    pluginsDialog.querySelector('[data-plugins-tab="installed"]').click();
  };
  const reviewManagedPlugin = async (plugin, action) => {
    const identity = { name: plugin.name || plugin.filename, source: plugin.source, project_id: plugin.project_id, icon_project_id: plugin.icon_project_id, version: plugin.version };
    stopPolling();
    const current = installGeneration;
    installController = new AbortController();
    showPanel(jvT("ui.620e222fa349"), true);
    panelBody.append(text("p", jvT("ui.a0bef54806de")));
    try {
      if (await blockPluginInstall(current)) return;
      const plan = await pluginRequest("manage/plan", { source: plugin.source || "Modrinth", action, project_id: plugin.project_id }, installController.signal);
      if (current !== installGeneration) return;
      showPanel(jvT("ui.620e222fa349"), true);
      const label = action === "enable" ? "Enable" : action === "disable" ? "Disable" : jvT("ui.c3812fc4acb8");
      overlayHeader(identity.name, "", [[plugin.version, "plugins-badge-version"], [acceptedServer?.content_kind === "mods" ? jvT("ui.f8b32f4e92bd") : plan.state === "disabled" ? jvT("ui.75081b593d15") : jvT("ui.92c1cdfdf4cb"), acceptedServer?.content_kind === "mods" ? "plugins-badge-installed" : plan.state === "disabled" ? "plugins-badge-disabled" : "plugins-badge-enabled"]], identity.project_id || identity.icon_project_id, null, undefined, identity.source);
      const explanation = action === "enable"
        ? jvT("ui.92ab4ccbdd01")
        : action === "disable"
          ? jvT("ui.158bd02f4ba0")
          : acceptedServer?.content_kind === "mods"
            ? jvT("ui.866d80ff46f1")
          : jvT("ui.bb5f5497d78e");
      panelBody.append(text("p", explanation));
      let players = null;
      if (plan.confirmation_required) {
        const row = text("label", "", "plugins-player-confirm");
        players = document.createElement("input"); players.type = "checkbox";
        row.append(players, text("span", jvT("ui.380785e65c6d")));
        panelBody.append(text("p", `${plan.online} player(s) are online${Array.isArray(plan.players) && plan.players.length ? ` (${plan.players.join(", ")})` : ""}. This action will disconnect them.`));
        panelBody.append(row);
      }
      const apply = pluginAction(label, async () => {
        apply.disabled = true;
        try {
          if (await blockPluginInstall(current)) return;
          const payload = await pluginRequest("manage", { source: plan.source || "Modrinth", action: plan.action, project_id: plan.project_id, plan_fingerprint: plan.plan_fingerprint, confirm_players: String(Boolean(players?.checked)) }, installController.signal);
          if (current !== installGeneration) return;
          if (operationIdentity?.operation_id !== payload.operation.operation_id) operationIdentity = { ...identity, operation_id: payload.operation.operation_id };
          if (!showOperation(payload.operation)) pollTimer = setTimeout(() => pollOperation(payload.operation.operation_id, current), 1000);
        } catch (error) { if (error.staleSelection) return; if (current === installGeneration && error.name !== "AbortError") panelError(error, identity); }
      }, action === "remove" ? "plugins-remove-action" : "plugins-state-action");
      let slider = null, finalConfirm = null;
      if (action === "remove") {
        const arm = text("div", "", "destructive-confirmation");
        const shell = text("div", "", "destructive-confirm-slider");
        const armedText = text("span", jvT("ui.8df1bfa87a15"), "destructive-confirm-slider-text");
        const thumb = text("span", jvT("ui.62b67e1f685b"), "destructive-confirm-slider-thumb"); thumb.setAttribute("aria-hidden", "true");
        slider = document.createElement("input"); slider.type = "range"; slider.min = "0"; slider.max = "100"; slider.step = "1"; slider.value = "0"; slider.setAttribute("aria-label", jvT("ui.8df1bfa87a15"));
        shell.append(armedText, thumb, slider);
        const finalLabel = text("label", "", "destructive-confirm-toggle-row"); finalLabel.hidden = true;
        finalConfirm = document.createElement("input"); finalConfirm.type = "checkbox"; finalConfirm.className = "destructive-confirm-toggle";
        finalLabel.append(text("span", jvT("ui.8ba7a99174e0"), "destructive-confirm-toggle-callout"), finalConfirm);
        arm.append(shell, finalLabel); panelBody.append(arm);
        slider.addEventListener("input", () => {
          const progress = Math.max(0, Math.min(100, Number(slider.value || 0)));
          shell.style.setProperty("--confirm-progress", String(progress / 100));
          shell.classList.toggle("is-armed", progress >= 100);
          finalLabel.hidden = progress < 100;
          armedText.textContent = progress >= 100 ? jvT("ui.24d5a6a37cb4") : jvT("ui.8df1bfa87a15");
          if (progress < 100) finalConfirm.checked = false;
          sync();
        });
        finalConfirm.addEventListener("change", () => sync());
      }
      const sync = () => { apply.disabled = Boolean(players && !players.checked) || Boolean(slider && (Number(slider.value) !== 100 || !finalConfirm.checked)); };
      players?.addEventListener("change", sync);
      sync();
      overlayActions(panelAction(jvT("ui.19766ed6ccb2"), returnToInstalled, true), apply);
    } catch (error) { if (error.staleSelection) return; if (current === installGeneration && error.name !== "AbortError") panelError(error, identity); }
  };
  const pluginRequest = async (route, body, signal) => {
    const selection = globalThis.justVoxelSelectionGeneration.plugins;
    const boundURL = body ? globalThis.justVoxelSelectedURL(`/api/plugins/workspace/${route}`, "plugins") : null;
    try {
      if (globalThis.justVoxelSelectionReady) await globalThis.justVoxelSelectionReady;
      if (selection !== globalThis.justVoxelSelectionGeneration.plugins) throw globalThis.justVoxelStaleSelection();
      const options = { credentials: "same-origin", cache: "no-store", headers: { Accept: "application/json" }, signal };
      if (body) {
        options.method = "POST";
        const csrf = document.querySelector('[data-minecraft-workspace-csrf]')?.value || document.querySelector('input[name="csrf"]')?.value || "";
        if (typeof FormData !== "undefined" && body instanceof FormData) {
          options.headers["X-CSRF-Token"] = csrf;
          options.body = body;
        } else {
          options.body = new URLSearchParams({ csrf, ...body });
        }
      }
      const response = await fetch(boundURL || globalThis.justVoxelSelectedURL(`/api/plugins/workspace/${route}`, "plugins"), options);
      const payload = await response.json();
      if (selection !== globalThis.justVoxelSelectionGeneration.plugins) throw globalThis.justVoxelStaleSelection();
      if (response.status === 401) { window.location.assign("/login"); throw new Error(jvT("ui.e30c2ec3ce01")); }
      if (response.status === 403 && String(jvC(payload.error || "")).toLowerCase().includes("password")) { window.location.assign("/password"); throw new Error(jvT("ui.fa29ee25267e")); }
      if (response.status === 409 && (route === "install/acknowledge" || route === "install/restore-runtime") &&
          payload.recovery_required === true && payload.recovery_kind === "plugin_install" &&
          payload.recovery_operation_id === body?.operation_id &&
          (payload.recovery_action === "start" || payload.recovery_action === "stop")) return payload;
      if (!response.ok) {
        const error = new Error(payload.error || payload.message || jvT("ui.13bd544e594d"));
        if (/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/.test(payload.diagnostic_id || "")) error.diagnosticID = payload.diagnostic_id;
        throw error;
      }
      return payload;
    } catch (error) {
      if (selection !== globalThis.justVoxelSelectionGeneration.plugins) throw globalThis.justVoxelStaleSelection();
      throw error;
    }
  };
  const pluginDiagnosticLink = (id) => {
    if (!/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/.test(id || "")) return;
    const row = text("p", "See System -> Logs -> Plugins for diagnostic details. ", "plugins-diagnostic-link");
    if (window.JustVoxelSystem?.openPluginLogs) row.append(panelAction(jvT("ui.2d4f452b773b"), () => {
      window.JustVoxelSystem.openPluginLogs(id);
    }, true));
    panelBody.append(row);
  };
  const panelError = (error, identity = null) => {
    showPanel();
    if (identity) overlayHeader(identity.name, "", [[identity.version, "plugins-badge-version"]], identity.project_id || identity.icon_project_id, null, identity.icon_url, identity.source);
    panelBody.append(text("p", jvT("ui.95e14e786910")), text("p", error.message || jvT("ui.e062b46f3b50")));
    pluginDiagnosticLink(error.diagnosticID);
    overlayActions(panelAction(jvT("ui.11a6767d5674"), closePanel, true));
  };
  const showAddonOperation = (operation) => {
    const terminal = ["succeeded", "rolled_back", "needs_attention", "resolved"].includes(operation.state);
    showPanel(jvT("ui.a5184d8cab4c"), false);
    panelBody.append(text("h3", jvT("ui.a5184d8cab4c")), text("p", operation.status, "plugins-operation-status"));
    if (operation.state === "needs_attention") {
      const feedback = text("p", jvT("ui.6a47d4640c27"));
      const retry = panelAction(jvT("ui.f37b055f32a8"), async () => {
        retry.disabled = true;
        try {
          const payload = await pluginRequest("addons/recover", { operation_id: operation.operation_id }, installController?.signal);
          showAddonOperation(payload.operation);
          load();
        } catch (error) { if (error.staleSelection) return; feedback.textContent = error.message; retry.disabled = false; }
      });
      panelBody.append(feedback);
      overlayActions(retry, panelAction(jvT("ui.7d9eb7acb13e"), closePanel, true));
    } else overlayActions(panelAction(terminal ? "Done" : jvT("ui.7d9eb7acb13e"), () => { closePanel(); load(); }, true));
    return terminal;
  };
  const showOperation = (operation) => {
    if (operation.operation_type === "plugin_addon_install") return showAddonOperation(operation);
    const sameOperation = !panel.hidden && operationViewID === operation.operation_id;
       const lifecycle = operation.operation_type !== "plugin_install";
    showPanel(lifecycle ? jvT("ui.c4f77e15e818") : `${contentSingular() === "mod" ? "Mod" : "Plugin"} installation`, false, sameOperation);
    const success = operation.state === "succeeded";
    const attention = operation.state === "needs_attention";
    const terminal = ["succeeded", "rolled_back", "needs_attention", "resolved"].includes(operation.state);
    const identity = operationIdentity?.operation_id === operation.operation_id ? operationIdentity : null;
    const heading = lifecycle ? (success ? jvT("ui.95826073d886") : attention ? jvT("ui.1e1b96ec8fd9") : operation.state === "rolled_back" ? jvT("ui.e5a5d82830bf") : operation.state === "resolved" ? jvT("ui.af7b87da8dba") : jvT("ui.90af090b1f0d")) : success ? (acceptedServer?.content_kind === "mods" ? jvT("ui.59cafc8e425d") : jvT("ui.1cfe133ba69b")) : attention ? jvT("ui.1e1b96ec8fd9") : operation.state === "rolled_back" ? jvT("ui.b80934d9f58a") : operation.state === "resolved" ? jvT("ui.af7b87da8dba") : contentSingular() === "mod" ? jvT("ui.24e2a1bc9146") : jvT("ui.11447ae67993");
    if (!sameOperation) {
      overlayHeader(identity?.name || heading, "", [[identity?.version, "plugins-badge-version"]], identity?.project_id || identity?.icon_project_id, null, identity?.icon_url, identity?.source);
      operationViewID = operation.operation_id;
    } else if (!identity) panelBody.querySelector(".plugins-overlay-identity h3").textContent = heading;
    if (identity) panelBody.append(text("h3", heading));
    if (!terminal) {
      const stages = lifecycle ? [jvT("ui.ee820467e916"), jvT("ui.838557924a5c"), jvT("ui.c0bf75bd78bf"), jvT("ui.6b983a81e5e8"), jvT("ui.eea2745e2867")] : [jvT("ui.ee820467e916"), jvT("ui.838557924a5c"), jvT("ui.569ca49f4aaf"), jvT("ui.6b983a81e5e8"), jvT("ui.eea2745e2867")];
      const index = ({ queued: 0, validating: 0, downloading: 0, player_check: 0, backup: 1, installing: 2, changing: 2, restarting: 3, verifying: 4 })[operation.stage];
      const sequence = document.createElement("ol");
      sequence.className = "plugins-install-stages";
      stages.forEach((label, position) => {
        const item = text("li", label);
        if (position === index) item.setAttribute("aria-current", "step");
        if (index !== undefined && position < index) item.className = "is-complete";
        sequence.append(item);
      });
      panelBody.append(sequence);
    }
    // The persisted status is the single source of result and final service state.
    const status = text("p", operation.status, "plugins-operation-status");
    status.setAttribute("role", "status");
    panelBody.append(status);
    if (attention || operation.state === "rolled_back" || operation.state === "failed") {
      pluginDiagnosticLink(operation.operation_id);
    }
    if (success) {
      const actions = overlayActions(panelAction(jvT("ui.46f0bdc90838"), () => {
        closePanel();
        pluginsDialog.querySelector('[data-plugins-tab="installed"]').click();
      }, true), panelAction(jvT("ui.11a6767d5674"), () => { closePanel(); load(); }));
      if (lifecycle) actions.lastElementChild.classList.add("plugins-done-action");
    }
    if (attention) {
      panelBody.append(text("p", jvT("ui.90d9a00b7a84"), "plugins-install-warning"));
      const feedback = text("p", "", "plugins-operation-feedback");
      const finishRecovery = (payload) => {
        if (!payload.operation || payload.operation.state !== "resolved") throw new Error(jvT("ui.1f3344e99b38"));
        showOperation(payload.operation);
        load();
      };
      const offerRuntimeRecovery = (recovery) => {
        const start = recovery.recovery_action === "start";
        const confirmPlayers = recovery.confirmation_required === true;
        feedback.textContent = start ? jvT("plugins.recovery.was_running") : jvT("plugins.recovery.was_stopped");
        if (confirmPlayers) {
          feedback.textContent += " " + jvT("plugins.recovery.players", {count: (recovery.online)});
          if (recovery.players?.length) feedback.textContent += " " + recovery.players.join(", ");
        }
        const label = start ? jvT("plugins.recovery.start") :
          confirmPlayers ? jvT("plugins.recovery.confirm_stop") : jvT("plugins.recovery.stop");
        const repair = panelAction(label, async () => {
          if (repair.disabled) return;
          repair.disabled = true;
          feedback.textContent = jvT("ui.76811ca2ebdb");
          const current = installGeneration;
          try {
            const body = {operation_id: operation.operation_id, restore_runtime: "true"};
            if (confirmPlayers) body.confirm_players = "true";
            const payload = await pluginRequest("install/restore-runtime", body, installController?.signal);
            if (current !== installGeneration) return;
            if (payload.recovery_required) offerRuntimeRecovery(payload);
            else finishRecovery(payload);
          } catch (error) {
            if (current !== installGeneration || error.staleSelection || error.name === "AbortError") return;
            feedback.textContent = error.message || jvT("ui.ead88024e125");
            repair.disabled = false;
          }
        });
        actions.replaceChildren(repair, close);
        repair.focus();
      };
      const recheck = panelAction(jvT("ui.137a11a338a1"), async () => {
        if (recheck.disabled) return;
        recheck.disabled = true;
        feedback.textContent = jvT("ui.76811ca2ebdb");
        const current = installGeneration;
        try {
          const payload = await pluginRequest(lifecycle ? "manage/acknowledge" : "install/acknowledge", { operation_id: operation.operation_id }, installController?.signal);
          if (current !== installGeneration) return;
          if (!lifecycle && payload.recovery_required) offerRuntimeRecovery(payload);
          else finishRecovery(payload);
        } catch (error) { if (error.staleSelection) return;
          if (current === installGeneration && error.name !== "AbortError") {
            feedback.textContent = error.message || jvT("ui.ead88024e125");
            recheck.disabled = false;
          }
        }
      });
      const close = panelAction(jvT("ui.7d9eb7acb13e"), closePanel, true);
      const actions = overlayActions(recheck, close);
      panelBody.append(feedback);
    } else if (terminal && !success) overlayActions(panelAction(jvT("ui.11a6767d5674"), () => { closePanel(); load(); }, true));
    else if (!terminal) overlayActions(panelAction(jvT("ui.7d9eb7acb13e"), closePanel, true));
    return terminal;
  };
  const pollOperation = async (id, current) => {
    if (current !== installGeneration) return;
    try {
      const payload = await pluginRequest(`operation?id=${encodeURIComponent(id)}`, null, installController?.signal);
      if (current !== installGeneration) return;
      if (!payload.operation) throw new Error(jvT("ui.6dcd7c9766b2"));
      if (!showOperation(payload.operation)) pollTimer = setTimeout(() => pollOperation(id, current), 1500);
    } catch (error) { if (error.staleSelection) return; if (current === installGeneration && error.name !== "AbortError") panelError(error, operationIdentity?.operation_id === id ? operationIdentity : null); }
  };
  const resumeActiveOperation = (operation, current) => {
    if (!showOperation(operation)) pollTimer = setTimeout(() => pollOperation(operation.operation_id, current), 1000);
  };
  const blockPluginInstall = async (current) => {
    const payload = await pluginRequest("operation", null, installController.signal);
    if (current !== installGeneration) return true;
    if (pluginOperationSettled(payload.operation)) return false;
    resumeActiveOperation(payload.operation, current);
    return true;
  };
  const applyPlugin = async (plan, project, uploads = []) => {
    if (uploads.some(({ input }) => input.files.length !== 1)) return;
    const fields = {
      source: plan.source, project_id: plan.project_id, version_id: plan.version_id, plan_fingerprint: plan.plan_fingerprint,
      confirm_players: String(plan.confirmation_required === true),
    };
    let body = fields;
    if (uploads.length) {
      body = new FormData();
      for (const [key, value] of Object.entries(fields)) body.append(key, value);
      for (const { artifact, input } of uploads) {
        body.append(`jar_${artifact.project_id}_${artifact.version_id}`, input.files[0]);
      }
    }
    // Presentation metadata never supplies identity for an installation request.
    const matchingProject = project?.project_id === plan.project_id && project?.source === plan.source ? project : null;
    const identity = { source: plan.source, project_id: plan.project_id, name: plan.project_name, version: plan.version_number, icon_project_id: matchingProject?.icon_project_id, icon_url: matchingProject?.icon_url };
    stopPolling();
    const current = installGeneration;
    installController = new AbortController();
    showPanel();
    overlayHeader(identity.name, "", [[identity.version, "plugins-badge-version"]], identity.project_id || identity.icon_project_id, null, identity.icon_url, identity.source);
    panelBody.append(text("h3", jvT("ui.6acfdc74eadf")));
    try {
      if (await blockPluginInstall(current)) return;
      const payload = await pluginRequest("install", body, installController.signal);
      if (current !== installGeneration) return;
      if (!payload.operation) throw new Error(jvT("ui.3f04bc49d05a"));
      if (operationIdentity?.operation_id !== payload.operation.operation_id) operationIdentity = { ...identity, operation_id: payload.operation.operation_id };
      if (!showOperation(payload.operation)) pollTimer = setTimeout(() => pollOperation(payload.operation.operation_id, current), 1000);
    } catch (error) { if (error.staleSelection) return; if (current === installGeneration && error.name !== "AbortError") panelError(error, identity); }
  };
  const readableSize = (size) => {
    if (!Number.isFinite(size) || size < 0) return jvT("ui.d7d557fa439a");
    if (size < 1024) return `${size} B`;
    const unit = size >= 1048576 ? "MiB" : "KiB";
    const value = size / (unit === "MiB" ? 1048576 : 1024);
    return `${Number(value.toFixed(1))} ${unit}`;
  };
  const reviewPlugin = async (projectID, versionID, project, readOnlyReview) => {
    if (project?.installed) { readOnlyReview?.(); return; }
    stopPolling();
    const current = installGeneration;
    installController = new AbortController();
    showPanel(`${contentSingular() === "mod" ? "Mod" : "Plugin"} installation`, true);
    panelBody.append(text("h3", jvT("ui.c6724b687a62")));
    overlayActions(panelAction(jvT("ui.7d9eb7acb13e"), closePanel, true));
    try {
      if (await blockPluginInstall(current)) return;
      const plan = await pluginRequest("install/plan", { source: project?.source || "Modrinth", project_id: projectID, version_id: versionID }, installController.signal);
      if (current !== installGeneration) return;
      if (plan.install_available === false) { readOnlyReview?.(plan.compatibility); return; }
      showPanel(`${contentSingular() === "mod" ? "Mod" : "Plugin"} installation`, true);
      const uploads = [];
      let install;
      overlayHeader(plan.project_name, plan.description || project?.description, [[plan.version_number, "plugins-badge-version"], [plan.server.server_software || plan.server.server_type, "plugins-badge-server"], [plan.server.minecraft_version && `MC ${plan.server.minecraft_version}`, "plugins-badge-minecraft"]], plan.project_id, install = panelAction(jvT("ui.569ca49f4aaf"), () => applyPlugin(plan, project, uploads)), project?.project_id === plan.project_id && project?.source === plan.source ? project.icon_url || "" : undefined, plan.source);
      const selectedVersion = project?.version?.id === plan.version_id ? project.version : null;
      const about = infoRows(overlaySection(jvT("ui.63e615c21158")));
      about(jvT("ui.e020e3c67bd0"), plan.version_number);
      about(plan.source === "Hangar" ? "Channel" : jvT("ui.0f9bdaf90d25"), selectedVersion?.version_type || jvT("ui.d83b9ff3c07a"));
      about(jvT("ui.2ef42ec21d93"), selectedVersion?.date_published || jvT("ui.d83b9ff3c07a"));
      about(jvT("ui.0e570ca6fabe"), plan.source);
      about(jvT("ui.e511470b21a9"), plan.project_id);
      about(jvT("ui.6e7fdd059d0f"), plan.version_id);
      const compatibility = infoRows(overlaySection(jvT("ui.b37ddf1f6e0f")));
      compatibility(jvT("ui.6e7d50e84f47"), plan.compatibility);
      compatibility(jvT("ui.8b8ee2755acb"), plan.server.server_software);
      compatibility(jvT("ui.de8921369774"), plan.server.minecraft_version);
      compatibility(jvT("ui.67635dab0dde"), (selectedVersion?.loaders || []).join(" / ") || jvT("ui.d83b9ff3c07a"));
      const versions = [...new Set(selectedVersion?.game_versions || [])];
      compatibility(jvT("ui.aeb9d7cea04c"), versions.join(" / ") || jvT("ui.d83b9ff3c07a"));
      const dependencies = overlaySection(jvT("ui.2e41b118eb20"));
      const required = plan.artifacts.filter((artifact) => !artifact.requested);
      const dependencyList = document.createElement("ul");
      dependencyList.className = "plugins-overlay-list";
      required.forEach((artifact) => {
        const item = text("li", `${artifact.project_name} · ${artifact.version_number}`);
        item.append(text("span", artifact.satisfied === "Already provided by the server / JustVoxel runtime" ? artifact.satisfied : artifact.satisfied ? jvT("ui.509ad4b7bd9b") : jvT("ui.3992f514b366"), "plugins-list-secondary"));
        item.className = "plugins-file-row";
        dependencyList.append(item);
      });
      dependencies.append(dependencyList);
      const informational = plan.display_dependencies || [];
      if (informational.length) dependencies.append(dependencyGroups(informational));
      if (!required.length && !informational.length) dependencies.append(text("p", jvT("ui.3d6eb94111f6")));
      const files = document.createElement("ul");
      files.className = "plugins-overlay-list";
      plan.artifacts.filter((artifact) => !artifact.satisfied).forEach((artifact) => {
        const item = text("li", artifact.filename, "plugins-file-row");
        item.append(text("span", readableSize(artifact.size), "plugins-list-secondary"));
        files.append(item);
      });
      overlaySection(jvT("ui.448167b72151")).append(files);
      const missing = plan.artifacts.filter((artifact) => artifact.upload_required && !artifact.satisfied);
      if (missing.length) {
        const section = overlaySection("JARs to provide");
        section.classList.add("plugins-addon-results");
        const updateInstall = () => {
          install.disabled = uploads.some(({ input }) => input.files.length !== 1);
        };
        for (const artifact of missing) {
          const label = text("label", `${artifact.project_name ? artifact.project_name + " · " : ""}${artifact.filename} · ${artifact.size} B`);
          const input = document.createElement("input");
          input.type = "file";
          input.accept = ".jar";
          input.required = true;
          input.addEventListener("change", updateInstall);
          label.append(input);
          section.append(text("p", "CurseForge cannot provide this JAR to JustVoxel automatically. Download the selected file from CurseForge and provide the exact JAR below."), label);
          uploads.push({ artifact, input });
        }
        updateInstall();
      }
      const installation = infoRows(overlaySection(jvT("ui.c3fc54aa5390")));
      installation(jvT("ui.3f70f2563600"), plan.installation_location);
      installation(jvT("ui.bbd4b6a86bc6"), jvT("ui.5c385efbd5ad"));
      const restart = infoRows(overlaySection(jvT("ui.6b983a81e5e8")));
      restart(jvT("ui.4850b174b713"), plan.restart_required ? jvT("ui.85a39ab345d6") : jvT("ui.1ea442a134b2"));
      restart(jvT("ui.7140f4f19dec"), acceptedServer?.content_kind === "mods" ? jvT("ui.0a7912f3c03c") : jvT("ui.e8940d0956bd"));
      restart(jvT("ui.943026a9371d"), jvT("ui.cca6ca9c651d"));
      if (plan.confirmation_required) {
        const section = overlaySection(jvT("ui.84e12ac655dd"));
        const players = infoRows(section);
        players(jvT("ui.0d21bd52022c"), String(plan.online));
        if (plan.players?.length) players(jvT("ui.c3b29e6eef2c"), plan.players.join(", "));
        players(jvT("ui.569ca49f4aaf"), jvT("ui.32e10a4d5759"));
      }
      panelBody.querySelector("button")?.focus();
    } catch (error) { if (error.staleSelection) return;
      if (current !== installGeneration || error.name === "AbortError") return;
      if (error.message === "This release is not a direct Paper/Bukkit plugin and cannot be installed into the plugins directory." && readOnlyReview) readOnlyReview(jvT("ui.1be43675b9e0"));
      else if (error.message === "BlueMap native add-on belongs in Plugins -> Add-ons -> BlueMap Add-ons. Select its local file there." && readOnlyReview) readOnlyReview(error.message);
      else if (error.message === "This loader mod cannot be installed into a plugin server. Select a compatible mod-loader server." && readOnlyReview) readOnlyReview(error.message);
      else panelError(error, project ? { name: project.title, source: project.source, project_id: project.project_id, icon_project_id: project.icon_project_id, icon_url: project.icon_url } : null);
    }
  };
  const resumePluginInstall = async () => {
    const requestedGeneration = installGeneration;
    try {
      const payload = await pluginRequest("operation");
      if (!pluginsDialog.open || pluginOperationSettled(payload.operation) || requestedGeneration !== installGeneration || !panel.hidden) return;
      stopPolling();
      installController = new AbortController();
      const current = installGeneration;
      if (!showOperation(payload.operation)) pollTimer = setTimeout(() => pollOperation(payload.operation.operation_id, current), 1000);
    } catch (error) { if (error.staleSelection) return; state.textContent = error.message || jvT("ui.729215ea003e"); }
  };
  panel.addEventListener("click", (event) => {
    if (event.target === panel && overlayDismissible) closePanel();
  });
  panel.addEventListener("keydown", (event) => {
    if (event.key === "Escape") { event.preventDefault(); event.stopPropagation(); closePanel(); }
    if (event.key === "Tab") {
      const buttons = Array.from(panel.querySelectorAll(panel.classList.contains("plugins-curseforge-panel") ? "button:not(:disabled),input:not(:disabled),summary" : "button:not(:disabled),input[type=file]:not(:disabled),summary"));
      const first = buttons[0], last = buttons.at(-1);
      if (document.activeElement === panel) { event.preventDefault(); (event.shiftKey ? last : first)?.focus(); }
      else if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last?.focus(); }
      else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first?.focus(); }
    }
  });
  const pluginInventorySafe = pluginOperationSettled;
  const recheckBlockedInventory = async (current) => {
    inventoryTimer = null;
    if (!pluginsDialog.open || tab === "discover" || current !== generation) return;
    try {
      const payload = await pluginRequest("operation", null, controller.signal);
      if (!pluginsDialog.open || tab === "discover" || current !== generation) return;
      if (pluginInventorySafe(payload.operation)) { load(); return; }
    } catch (error) { if (error.staleSelection) return;
      if (current !== generation || error.name === "AbortError") return;
      state.textContent = error.message || jvT("ui.729215ea003e");
    }
    inventoryTimer = setTimeout(() => recheckBlockedInventory(current), 1500);
  };
  const discoverQuery = (position) => new URLSearchParams({ q: form.elements.q.value, compatible_only: String(form.elements.compatible_only.checked), release_only: String(form.elements.release_only.checked), modrinth: String(providerServerKey !== null && form.elements.modrinth.checked), hangar: String(providerServerKey !== null && form.elements.hangar.checked), curseforge: String(providerServerKey !== null && curseForgeConfigured && curseForgeSwitch.checked), modrinth_offset: String(position.modrinth_offset), hangar_offset: String(position.hangar_offset), curseforge_offset: String(position.curseforge_offset), provider_turn: String(position.provider_turn), hangar_first: String(position.hangar_first), limit: String(pageLimit) });
  // Compare authoritative context before accepting cursor history from any response.
  const acceptDiscoverContext = (server) => {
    const key = JSON.stringify([server.instance_id, server.server_type, server.minecraft_version, server.content_kind, server.providers]);
    const initialize = providerServerKey !== key;
    acceptedServer = server;
    discoverServerKey = key;
    form.elements.q.placeholder = server.content_kind === "mods" ? jvT("ui.b186b72af3a9") : server.content_kind === "plugins" ? jvT("ui.df08b7498d9a") : jvT("ui.2d3a28cab149");
    form.elements.q.setAttribute("aria-label", form.elements.q.placeholder);
    for (const name of ["modrinth", "hangar", "curseforge"]) {
      const control = form.elements[name];
      const supported = server.supported && server.providers?.[name] === true;
      control.disabled = !supported;
      if (initialize) control.checked = supported && (name !== "curseforge" || curseForgeConfigured);
      if (!supported || name === "curseforge" && !curseForgeConfigured) control.checked = false;
    }
    providerServerKey = key;
    if (initialize) resetCursors();
    return !initialize;
  };

  const addonRequest = async (route, kind, file, plan = null, confirmPlayers = false) => {
    const selection = globalThis.justVoxelSelectionGeneration.plugins;
    try {
      const body = new FormData();
      body.append("type", kind);
      body.append("file", file, file.name);
      if (plan) {
        body.append("plan_fingerprint", plan.plan_fingerprint);
        body.append("confirm_players", String(confirmPlayers));
      }
      const response = await fetch(globalThis.justVoxelSelectedURL(`/api/plugins/workspace/addons/${route}`, "plugins"), {
        method: "POST", credentials: "same-origin", cache: "no-store", body,
        signal: installController?.signal,
        headers: { Accept: "application/json", "X-CSRF-Token": document.querySelector('[data-minecraft-workspace-csrf]')?.value || document.querySelector('input[name="csrf"]')?.value || "" },
      });
      const payload = await response.json();
      if (selection !== globalThis.justVoxelSelectionGeneration.plugins) throw globalThis.justVoxelStaleSelection();
      if (response.status === 401) { window.location.assign("/login"); throw new Error(jvT("ui.e30c2ec3ce01")); }
      if (!response.ok) throw new Error(payload.error || jvT("ui.792e51bdfbba"));
      return payload;
    } catch (error) {
      if (selection !== globalThis.justVoxelSelectionGeneration.plugins) throw globalThis.justVoxelStaleSelection();
      throw error;
    }
  };
  const reviewAddon = async (section, file) => {
    stopPolling();
    const current = installGeneration;
    installController = new AbortController();
    showPanel(jvT("ui.c6cb217ab502"), true);
    panelBody.append(text("p", jvT("ui.0b8d6f6cbd46")));
    try {
      if (await blockPluginInstall(current)) return;
      const plan = await addonRequest("plan", section.kind, file);
      if (current !== installGeneration) return;
      showPanel(jvT("ui.c6cb217ab502"), true);
      overlayHeader(plan.display_name || plan.filename, plan.validation, [[section.name, ""]]);
      const rows = infoRows(overlaySection(jvT("ui.c3fc54aa5390")));
      rows(jvT("ui.baaddf70fb5d"), section.name);
      rows(jvT("ui.50009ce1da4d"), plan.filename);
      rows(jvT("ui.1af851907331"), readableSize(plan.size));
      rows(jvT("ui.293d404a500f"), plan.destination);
      rows(jvT("ui.b46871152667"), plan.restart_required ? jvT("ui.4c44901d7781") : jvT("ui.d8177815c19a"));
      if (plan.restart_required) rows(jvT("ui.19d74ce378ee"), jvT("ui.ac7aae0a4bea"));
      let players = null;
      if (plan.confirmation_required) {
        const label = document.createElement("label");
        players = document.createElement("input");
        players.type = "checkbox";
        label.append(players, document.createTextNode(jvT("ui.d90364e2ca98", {v0: (plan.online), v1: (plan.players?.length ? ": " + plan.players.join(", ") : "")})));
        overlaySection(jvT("ui.84e12ac655dd")).append(label);
      }
      const apply = panelAction(jvT("ui.569ca49f4aaf"), async () => {
        apply.disabled = true;
        try {
          // The exact selected File is sent again; the Agent revalidates everything.
          const payload = await addonRequest("install", section.kind, file, plan, Boolean(players?.checked));
          if (current !== installGeneration) return;
          if (!payload.operation) throw new Error(jvT("ui.011bf983e35e"));
          if (!showOperation(payload.operation)) pollTimer = setTimeout(() => pollOperation(payload.operation.operation_id, current), 1000);
        } catch (error) { if (error.staleSelection) return; if (current === installGeneration && error.name !== "AbortError") panelError(error); }
      });
      apply.disabled = Boolean(players && !players.checked);
      players?.addEventListener("change", () => { apply.disabled = !players.checked; });
      overlayActions(panelAction(jvT("ui.19766ed6ccb2"), closePanel, true), apply);
    } catch (error) { if (error.staleSelection) return; if (current === installGeneration && error.name !== "AbortError") panelError(error); }
  };
  const renderAddonSection = (section) => {
    const card = document.createElement("article");
    card.className = "plugins-result-card";
    card.append(text("h3", section.name));
    const badges = document.createElement("div");
    badges.className = "plugins-card-badges";
    badges.append(pluginBadge(section.host.available ? jvT("ui.a943e849a946") : section.host.status === "disabled" ? jvT("ui.b05fd9b8ff81") : jvT("ui.0d27c1cf319c"), section.host.available ? "plugins-badge-installed" : section.host.status === "disabled" ? "plugins-badge-disabled" : "plugins-badge-attention"));
    card.append(badges, text("p", section.host.message, "plugins-card-description"), text("p", section.destination, "plugins-card-release"));
    for (const item of section.files) {
      const content = document.createElement("section");
      content.className = "plugins-overlay-section";
      content.append(text("h4", item.display_name || item.filename), text("p", item.filename));
      const managed = document.createElement("div");
      managed.className = "plugins-card-badges";
      managed.append(pluginBadge(section.name), pluginBadge(item.state === "installed" ? jvT("ui.f8b32f4e92bd") : jvT("ui.c1ebc7817870"), item.state === "installed" ? "plugins-badge-installed" : "plugins-badge-attention"), pluginBadge(jvT("ui.e015cb247578")));
      content.append(managed, text("p", item.target_relative_path), text("p", jvT("ui.3ac22f0a39de")));
      card.append(content);
    }
    const picker = document.createElement("input");
    picker.type = "file";
    picker.accept = section.accept;
    picker.setAttribute("aria-label", jvT("ui.23eff7e5c6f5", {v0: (section.name)}));
    picker.disabled = !section.host.available;
    const choose = document.createElement("label");
    choose.append(document.createTextNode(jvT("ui.b09c73d9d3f8")), picker);
    const review = panelAction(jvT("ui.db401d9a4366"), () => { if (picker.files?.length === 1) reviewAddon(section, picker.files[0]); });
    review.disabled = true;
    picker.addEventListener("change", () => { review.disabled = !section.host.available || picker.files?.length !== 1; });
    card.append(choose, review);
    return card;
  };
  const load = async () => {
    curseForgeSection.hidden = tab !== "curseforge-api";
    results.hidden = tab === "curseforge-api";
    syncCurseForgeInfo();
    if (tab === "curseforge-api") {
      clearInventoryTimer(); ++generation; controller?.abort();
      state.textContent = ""; form.hidden = true; pagination.hidden = true;
      return;
    }
    if (globalThis.justVoxelSelectionReady) await globalThis.justVoxelSelectionReady;
    if (tab === "curseforge-api") return;
    if (!document.body.classList.contains("role-administrator")) return;
    clearInventoryTimer();
    const current = ++generation;
    controller?.abort();
    controller = new AbortController();
    state.textContent = jvT("ui.e1d147ffcc21", {v0: (contentPlural())});
    results.replaceChildren();
    results.classList.toggle("plugins-addon-results", tab === "addons");
    pagination.hidden = tab !== "discover";
    previous.disabled = true;
    next.disabled = true;
    form.hidden = tab !== "discover";
    const query = discoverQuery(cursor);
    try {
      if (tab !== "discover") {
        const installation = await pluginRequest("operation", null, controller.signal);
        if (current !== generation) return;
        if (!pluginInventorySafe(installation.operation)) {
          resumeActiveOperation(installation.operation, installGeneration);
          state.textContent = jvT("ui.59c503a7dc50");
          inventoryTimer = setTimeout(() => recheckBlockedInventory(current), 1500);
          return;
        }
      }
      const response = await fetch(globalThis.justVoxelSelectedURL(`/api/plugins/workspace/${tab}${tab === "discover" ? "?" + query : ""}`, "plugins"), { credentials: "same-origin", cache: "no-store", headers: { Accept: "application/json" }, signal: controller.signal });
      const payload = await response.json();
      if (current !== generation) return;
      if (response.status === 401) { window.location.assign("/login"); return; }
      if (response.status === 403 && String(jvC(payload.error || "")).toLowerCase().includes("password")) { window.location.assign("/password"); return; }
      if (!response.ok) throw new Error(payload.error || jvT("ui.3e588ca3f342"));
      const server = payload.server;
      const contextReady = acceptDiscoverContext(server);
      form.querySelectorAll("input,button").forEach((control) => { if (control === curseForgeSwitch || curseForgeNotice.contains(control)) return; control.disabled = !server.supported; });
      if (!server.supported) { state.textContent = server.message; return; }
      acceptDiscoverContext(server);
      if (tab === "discover" && !contextReady) { load(); return; }
      if (tab === "addons") {
        if (server.content_kind !== "plugins") { state.textContent = jvT("ui.75db37c5b1a1"); return true; }
        payload.sections.forEach((section) => results.append(renderAddonSection(section)));
        state.textContent = jvT("ui.5c6bda48718f");
        return true;
      }
      const items = tab === "discover" ? payload.projects : payload.plugins;
      items.forEach((item) => results.append(tab === "discover" ? renderProject(item, server) : renderInstalled(item)));
      state.textContent = tab === "discover" && !payload.modrinth && !payload.hangar && !payload.curseforge ? jvT("ui.8298decb59a3", {v0: (contentPlural())}) : server.message || (items.length ? "" : tab === "discover" ? jvT("ui.6f1468c7d9be", {v0: (contentPlural())}) : jvT("ui.71bcf5ac0f29", {v0: (contentSingular())}));
      if (tab === "discover") {
        cursor = payload.cursor;
        nextCursor = payload.next_cursor;
        pagination.hidden = false;
        previous.disabled = pageIndex === 0;
        const currentPage = pageIndex + 1;
        totalPages = Math.max(1, Math.ceil(payload.total / pageLimit));
        pageCursors[pageIndex] = cursor;
        if (nextCursor) pageCursors[pageIndex + 1] = nextCursor;
        else pageCursors.length = pageIndex + 1;
        next.disabled = !nextCursor;
        pageLabel.textContent = jvT("ui.477c22095a63", {v0: (currentPage), v1: (totalPages)});
      }
      return true;
    } catch (error) { if (error.staleSelection) return;
      if (current === generation && error.name !== "AbortError") state.textContent = error.message || jvT("ui.3e588ca3f342");
    } finally {
      if (current === generation) {
        if (tab === "discover") {
          previous.disabled = pageIndex === 0;
          next.disabled = !nextCursor;
        }
      }
    }
  };
  const discoverCapacity = () => {
    const geometry = results.getBoundingClientRect();
    if (geometry.width <= 0 || geometry.height <= 0) return pageLimit;
    const style = getComputedStyle(results);
    const columns = style.gridTemplateColumns.split(" ").filter(Boolean).length;
    const cardHeight = parseFloat(style.getPropertyValue("--plugins-card-height"));
    const gap = parseFloat(style.rowGap) || 0;
    const rows = Math.max(1, Math.floor((results.clientHeight + gap) / (cardHeight + gap)));
    return Math.max(1, Math.min(24, columns * rows));
  };
  const updateCapacity = (reload = true) => {
    if (!pluginsDialog.open || tab !== "discover") return;
    const newLimit = discoverCapacity();
    if (newLimit === pageLimit) return;
    pageLimit = newLimit;
    resetCursors();
    if (reload) load();
  };
  const capacityObserver = new ResizeObserver(() => {
    clearTimeout(capacityTimer);
    capacityTimer = setTimeout(() => updateCapacity(), 200);
  });
  capacityObserver.observe(results);
  window.addEventListener("justvoxel:server-selection-changed", (event) => {
    if (event.detail?.workspace !== "plugins") return;
    clearTimeout(capacityTimer);
    clearTimeout(searchTimer);
    clearInventoryTimer();
    ++generation;
    controller?.abort();
    // An in-flight POST still belongs to its original server.
    installController = null;
    closePanel(false);
    panelBody.replaceChildren();
    operationIdentity = null;
    resetCursors();
    discoverServerKey = null;
    providerServerKey = null; acceptedServer = null;
    for (const name of ["modrinth", "hangar", "curseforge"]) { form.elements[name].checked = false; form.elements[name].disabled = true; }
    results.replaceChildren();
    if (pluginsDialog.open) { load(); resumePluginInstall(); }
  });
  const refreshWorkspace = () => globalThis.justVoxelServerSelection(pluginsDialog.querySelector("[data-server-selector-slot]"), true)
    .then(async () => { await refreshCurseForgeStatus(); resetCursors(); load(); }).catch(error => { if (state) state.textContent = error.message; });
  const workspaceWindow = setupWorkspaceWindow(pluginsDialog, { onOpen: () => { updateCapacity(false); refreshWorkspace(); resumePluginInstall(); }, onClose: () => { providerServerKey = null; acceptedServer = null; curseForgeKey.value = ""; dismissCurseForgeNotice(); curseForgeSwitch.checked = false; syncCurseForgeInfo(); clearTimeout(capacityTimer); clearTimeout(searchTimer); clearInventoryTimer(); ++generation; controller?.abort(); resetCursors(); closePanel(); } });
  pluginsOpen.addEventListener("click", () => {
    if (!document.body.classList.contains("role-administrator")) return;
    const controlCenter = document.querySelector("[data-control-center]");
    if (controlCenter) controlCenter.open = false;
    workspaceWindow?.open();
  });
  pluginsDialog.querySelector("[data-plugins-close]").addEventListener("click", () => workspaceWindow?.close());
  pluginsDialog.querySelector("[data-plugins-refresh]").addEventListener("click", refreshWorkspace);
  pluginsDialog.querySelectorAll("[data-plugins-tab]").forEach((button) => button.addEventListener("click", () => {
    clearTimeout(searchTimer);
    clearInventoryTimer();
    tab = button.dataset.pluginsTab;
    dismissCurseForgeNotice();
    resetCursors();
    pluginsDialog.querySelectorAll("[data-plugins-tab]").forEach((node) => node.setAttribute("aria-selected", String(node === button)));
    updateCapacity(false);
    load();
  }));
  form.elements.q.addEventListener("input", () => {
    clearTimeout(searchTimer);
    ++generation;
    controller?.abort();
    resetCursors();
    previous.disabled = true;
    next.disabled = true;
    searchTimer = setTimeout(() => load(), 250);
  });
  form.addEventListener("submit", (event) => { event.preventDefault(); clearTimeout(searchTimer); resetCursors(); load(); });
  form.querySelectorAll('input[name="compatible_only"],input[name="release_only"],input[name="modrinth"],input[name="hangar"]').forEach((control) => control.addEventListener("change", () => { clearTimeout(searchTimer); resetCursors(); load(); }));
  previous.addEventListener("click", () => { if (pageIndex > 0) { cursor = pageCursors[--pageIndex]; load(); } });
  next.addEventListener("click", () => {
    if (!nextCursor) return;
    pageCursors[pageIndex + 1] = nextCursor;
    cursor = pageCursors[++pageIndex];
    load();
  });
  // Role discovery is asynchronous. Restore only after Administrator identity
  // has been established, matching the role-gated launcher.
  const restore = () => {
    if (document.body.classList.contains("role-pending")) return false;
    if (document.body.classList.contains("role-administrator") && readWorkspaceWindowState("plugins").open) workspaceWindow?.open();
    return true;
  };
  if (!restore()) {
    const observer = new MutationObserver(() => { if (restore()) observer.disconnect(); });
    observer.observe(document.body, { attributes: true, attributeFilter: ["class"] });
  }
}

const minecraftServersDialog = document.querySelector("[data-minecraft-servers-workspace-dialog]");
const minecraftServersOpen = document.querySelector("[data-minecraft-servers-open]");
if (minecraftServersDialog && minecraftServersOpen) {
  const content = minecraftServersDialog.querySelector("[data-minecraft-servers-content]");
  const state = minecraftServersDialog.querySelector("[data-minecraft-servers-state]");
  const actionsSlot = minecraftServersDialog.querySelector("[data-minecraft-servers-actions]");
  let sequence = 0;
  const load = async () => {
    const current = ++sequence;
    const query = new URLSearchParams();
    const destination = new URLSearchParams(window.location.search);
    for (const key of ["created", "deleted"]) if (destination.get(key) === "yes") query.set(key, "yes");
    actionsSlot.replaceChildren();
    state.textContent = jvT("ui.70cede2476a8");
    try {
      const response = await fetch("/workspace/minecraft-servers?" + query, {credentials:"same-origin", cache:"no-store"});
      if (!response.ok || response.redirected) throw new Error(jvT("ui.c3ba26d72a06"));
      const html = await response.text();
      if (current !== sequence) return;
      const range = document.createRange();
      range.selectNodeContents(content);
      const fragment = range.createContextualFragment(html);
      content.replaceChildren(fragment);
      content.justVoxelRefreshServers = load;
      globalThis.justVoxelMinecraftServers(content);
      const actions = content.querySelector(".minecraft-server-arrange-toolbar");
      if (actions) actionsSlot.replaceChildren(actions);
      state.textContent = "";
    } catch (error) {
      if (current !== sequence) return;
      content.replaceChildren();
      state.textContent = error.message;
    }
  };
  const workspaceWindow = setupWorkspaceWindow(minecraftServersDialog, {onOpen: load});
  minecraftServersOpen.addEventListener("click", () => {
    if (controlCenter) controlCenter.open = false;
    workspaceWindow?.open();
  });
  minecraftServersDialog.querySelector("[data-minecraft-servers-close]").addEventListener("click", () => workspaceWindow?.close());
  minecraftServersDialog.querySelector("[data-minecraft-servers-refresh]").addEventListener("click", load);
  const restore = () => {
    if (document.body.classList.contains("role-pending")) return false;
    if (document.body.classList.contains("role-administrator") && readWorkspaceWindowState("minecraft-servers").open) workspaceWindow?.open();
    return true;
  };
  if (!restore()) {
    const observer = new MutationObserver(() => { if (restore()) observer.disconnect(); });
    observer.observe(document.body, {attributes:true, attributeFilter:["class"]});
  }
}

// Both Minecraft Network launchers share one appliance workspace.
const minecraftNetworkDialog = document.querySelector("[data-minecraft-network-workspace-dialog]");
if (minecraftNetworkDialog) {
  const content = minecraftNetworkDialog.querySelector("[data-minecraft-network-content]");
  const state = minecraftNetworkDialog.querySelector("[data-minecraft-network-state]");
  let sequence = 0;
  let saving = false;
  let configureForm = null;
  let baseline = "";
  let originalFields = [];
  let runtimeTimer = null;
  let runtimeRequest = null;
  let runtimeGeneration = 0;
  const stopRuntimePolling = () => {
    runtimeGeneration++;
    clearTimeout(runtimeTimer);
    runtimeTimer = null;
    runtimeRequest?.abort();
    runtimeRequest = null;
  };
  const stateBadge = (value) => {
    const states = {
      running: [jvT("ui.f4ccae29e1bb"), "good"], stopped: [jvT("ui.1a4f630ac1b6"), "danger"],
      failed: [jvT("ui.031a8f0f659d"), "danger"], starting: [jvT("ui.aeed4d26bb5f"), "warning"],
      stopping: [jvT("ui.a71ee1d4ac5c"), "warning"],
    };
    return Object.hasOwn(states, value) ? states[value] : [jvT("ui.ca1844969742"), "muted"];
  };
  const scheduleRuntimePolling = () => {
    if (!minecraftNetworkDialog.open || !minecraftNetworkDialog.isConnected) return;
    clearTimeout(runtimeTimer);
    runtimeTimer = setTimeout(pollRuntime, 5000);
  };
  const pollRuntime = async () => {
    if (!minecraftNetworkDialog.open || !minecraftNetworkDialog.isConnected) {
      stopRuntimePolling();
      return;
    }
    if (saving) { scheduleRuntimePolling(); return; }
    const generation = runtimeGeneration;
    const current = sequence;
    const request = new AbortController();
    runtimeRequest = request;
    const timeout = setTimeout(() => request.abort(), 10000);
    try {
      const response = await fetch("/workspace/minecraft-network?runtime_only=true", {
        credentials: "same-origin", cache: "no-store", signal: request.signal,
      });
      if (!response.ok || response.redirected) throw new Error(jvT("ui.180119be630b"));
      const status = await response.json();
      if (generation !== runtimeGeneration || current !== sequence || saving || !minecraftNetworkDialog.open || !minecraftNetworkDialog.isConnected) return;
      const states = new Map((status.servers || []).map(server => [server.id, server.state]));
      // Only badges change. Draft inputs, membership, notices and the baseline
      // remain exactly as they were before this runtime observation.
      for (const badge of content.querySelectorAll("[data-network-server-state]")) {
        const [label, style] = stateBadge(states.get(badge.dataset.networkServerState));
        badge.textContent = label;
        badge.className = `badge minecraft-network-state ${style}`;
      }
    } catch (_) {
      // A failed poll cannot establish that a previously running server is stopped.
      if (generation === runtimeGeneration && current === sequence && !saving) {
        for (const badge of content.querySelectorAll("[data-network-server-state]")) {
          badge.textContent = jvT("ui.ca1844969742");
          badge.className = "badge minecraft-network-state muted";
        }
      }
    } finally {
      clearTimeout(timeout);
      if (generation === runtimeGeneration) {
        runtimeRequest = null;
        scheduleRuntimePolling();
      }
    }
  };
  const configuration = () => {
    if (!configureForm) return "";
    const data = new FormData(configureForm);
    return JSON.stringify({
      enabled: data.get("enabled") === "on",
      java_port: Number(data.get("java_port")),
      bedrock_port: Number(data.get("bedrock_port")),
      members: data.getAll("members").sort(),
      welcome_message: data.get("welcome_message") || "",
      whitelist_enforced: data.get("whitelist_enforced") === "true",
    });
  };
  const isDirty = () => configureForm !== null && configuration() !== baseline;
  const syncDraft = () => {
    if (!configureForm) return;
    const enforced = configureForm.elements.namedItem("whitelist_enforced").value === "true";
    content.querySelector("[data-network-whitelist-switch]").setAttribute("aria-checked", String(enforced));
    content.querySelector("[data-network-whitelist-badge]").textContent = enforced ? jvT("ui.92c1cdfdf4cb") : jvT("ui.75081b593d15");
    content.querySelector("[data-network-whitelist-heading]").textContent = enforced ? "Network Whitelist: Enforced" : jvT("ui.388e35b34211");
    const dirty = isDirty();
    configureForm.querySelector("[data-minecraft-network-cancel]").disabled = !dirty;
    configureForm.querySelector('[value="configure"]').disabled = !dirty || configureForm.dataset.recoveryRequired === "true";
  };
  const discardDraft = () => {
    for (const {field, value, checked} of originalFields) {
      field.value = value;
      if (typeof checked === "boolean") field.checked = checked;
    }
    syncDraft();
  };
  const establishBaseline = () => {
    configureForm = content.querySelector("#minecraft-network-configure");
    originalFields = configureForm ? Array.from(configureForm.elements, field => ({field, value: field.value, checked: field.checked})) : [];
    baseline = configuration();
    syncDraft();
  };
  content.addEventListener("input", syncDraft);
  content.addEventListener("change", syncDraft);
  content.addEventListener("reset", (event) => {
    if (event.target !== configureForm) return;
    event.preventDefault();
    discardDraft();
  });
  content.addEventListener("click", (event) => {
    if (event.target.closest("[data-minecraft-network-cancel]")) discardDraft();
    if (event.target.closest("[data-network-whitelist-switch]") && configureForm) {
      const input = configureForm.elements.namedItem("whitelist_enforced");
      input.value = input.value === "true" ? "false" : "true";
      syncDraft();
    }
  });
  const load = async (body = null) => {
    if (saving) return;
    stopRuntimePolling();
    const current = ++sequence;
    saving = body !== null;
    content.inert = saving;
    state.classList.toggle("notice", saving);
    state.classList.toggle("warning", saving);
    if (saving) content.querySelectorAll(".notice.success").forEach(notice => notice.remove());
    state.textContent = saving ? jvT("ui.b702684f4c43") : jvT("ui.2e3ea2889ba4");
    try {
      // POST returns freshly read authoritative status, including recovery state.
      const response = await fetch("/workspace/minecraft-network", {
        method: body === null ? "GET" : "POST",
        credentials: "same-origin", cache: "no-store",
        ...(body === null ? {} : { body }),
      });
      if (!response.ok || response.redirected) throw new Error(jvT("ui.384ca342d9ed"));
      const html = await response.text();
      if (current !== sequence) return;
      const range = document.createRange();
      range.selectNodeContents(content);
      content.replaceChildren(range.createContextualFragment(html));
      establishBaseline();
      state.textContent = "";
      state.classList.remove("notice", "warning");
      if (body && ["configure", "recover"].includes(body.get("action")) && content.querySelector(".notice.success")) {
        const servers = document.querySelector("[data-minecraft-servers-workspace-dialog]");
        if (servers?.open) servers.querySelector("[data-minecraft-servers-refresh]")?.click();
      }
    } catch (error) {
      if (current === sequence) {
        state.classList.remove("notice", "warning");
        state.textContent = error.message;
      }
    } finally {
      if (current === sequence) {
        saving = false;
        content.inert = false;
        scheduleRuntimePolling();
      }
    }
  };
  const closeNetwork = () => { stopRuntimePolling(); discardDraft(); };
  const workspaceWindow = setupWorkspaceWindow(minecraftNetworkDialog, { onOpen: () => load(), onClose: closeNetwork });
  minecraftNetworkDialog.addEventListener("cancel", closeNetwork);
  minecraftNetworkDialog.addEventListener("close", closeNetwork);
  // Delegation also covers the dynamically loaded Minecraft Servers action bar.
  document.addEventListener("click", (event) => {
    const launcher = event.target.closest("[data-minecraft-network-open]");
    if (!launcher || !document.body.classList.contains("role-administrator")) return;
    event.preventDefault();
    const destination = new URL(window.location.href);
    destination.searchParams.set("workspace", "minecraft-network");
    destination.searchParams.delete("tab");
    destination.searchParams.delete("operation");
    window.history.replaceState(window.history.state, "", destination);
    if (controlCenter) controlCenter.open = false;
    workspaceWindow?.open();
  });
  minecraftNetworkDialog.querySelector("[data-minecraft-network-close]").addEventListener("click", () => workspaceWindow?.close());
  minecraftNetworkDialog.querySelector("[data-minecraft-network-refresh]").addEventListener("click", () => load());
  content.addEventListener("submit", (event) => {
    const form = event.target;
    if (!(form instanceof HTMLFormElement)) return;
    event.preventDefault();
    if (form === configureForm && (!isDirty() || configureForm.dataset.recoveryRequired === "true")) return;
    const body = new URLSearchParams(new FormData(form));
    if (form === configureForm) body.set("action", "configure");
    if (event.submitter?.name) body.set(event.submitter.name, event.submitter.value);
    load(body);
  });
  const restore = () => {
    if (document.body.classList.contains("role-pending")) return false;
    if (document.body.classList.contains("role-administrator") && readWorkspaceWindowState("minecraft-network").open) workspaceWindow?.open();
    return true;
  };
  if (!restore()) {
    const observer = new MutationObserver(() => { if (restore()) observer.disconnect(); });
    observer.observe(document.body, { attributes: true, attributeFilter: ["class"] });
  }
}

// Bookmarked standalone pages land on the matching dashboard workspace.
if (dashboard) {
  const destination = new URLSearchParams(window.location.search);
  let workspace = destination.get("workspace");
  if (workspace === "minecraft" && window.location.hash === "#manual-backup") workspace = "backups";
  const launchers = {
    "minecraft-servers": "[data-minecraft-servers-open]",
    "minecraft-network": "[data-minecraft-network-open]",
    minecraft: "[data-minecraft-open]",
    system: "[data-system-open]",
    storage: "[data-storage-open]",
    backups: "[data-backups-open]",
    migration: "[data-migration-open]",
    version: "[data-version-open]",
    plugins: "[data-plugins-open]",
  };
  if (workspace === "version" && ["switch", "software"].includes(destination.get("tab"))) {
    const migrated = new URL(window.location.href);
    migrated.searchParams.set("workspace", "migration");
    migrated.searchParams.set("tab", "switch");
    window.location.assign(migrated.pathname + migrated.search);
  } else if (Object.hasOwn(launchers, workspace)) {
    const launch = () => {
      if (document.body.classList.contains("role-pending")) return false;
      const button = document.querySelector(launchers[workspace]);
      if (!button || button.classList.contains("nav-admin-only") && !document.body.classList.contains("role-administrator")) return true;
      button.click();
      let tab = destination.get("tab") || "";
      if (workspace === "minecraft") {
        if (window.location.hash === "#whitelist") tab = "players";
        if (window.location.hash === "#minecraft-logs") tab = "logs";
      }
      const allowedTabs = {
        minecraft: ["overview", "logs", "memory", "gameplay", "players", "crossplay", "rcon"],
        version: ["container", "minecraft"],
        plugins: ["discover", "installed", "addons"],
        system: ["health", "history", "users", "security", "logs", "date-time", "wallpaper", "reset", "ups", "about"],
        migration: ["export", "import", "switch"],
      };
      if (workspace !== "migration" && allowedTabs[workspace]?.includes(tab)) {
        const tabButton = document.querySelector(`[data-${workspace}-tab="${tab}"]`);
        const isAdmin = document.body.classList.contains("role-administrator");
        const isOperator = document.body.classList.contains("role-operator");
        if (tabButton && (!tabButton.classList.contains("nav-admin-only") || isAdmin) &&
            (!tabButton.classList.contains("nav-operator-plus") || isAdmin || isOperator)) tabButton.click();
      }
      return true;
    };
    if (!launch()) {
      const observer = new MutationObserver(() => {
        if (!launch()) return;
        observer.disconnect();
      });
      observer.observe(document.body, { attributes: true, attributeFilter: ["class"] });
    }
  }
}

// Standalone selected overview remains a usable Delete retry surface even if
// an interrupted deletion has already removed runtime configuration.
document.querySelectorAll("[data-selected-presentation]").forEach((form)=>{
 const button=form.querySelector("button"),input=form.querySelector('[name="confirm_name"]');
 if(input)input.addEventListener("input",()=>{button.disabled=input.value!==form.dataset.deleteName;});
 form.addEventListener("submit",async(event)=>{event.preventDefault();if(input && input.value!==form.dataset.deleteName)return;button.disabled=true;
 const status=document.querySelector("[data-selected-action-status]");
 try{const response=await fetch(form.action,{method:"POST",credentials:"same-origin",body:new URLSearchParams(new FormData(form))});const result=await response.json();if(!response.ok || !result.ok)throw new Error(result.error || jvT("ui.a3416dd07623"));if(input)window.location.assign(result.redirect);else window.location.reload();}
 catch(error){if(status)status.textContent=error.message;button.disabled=input ? input.value!==form.dataset.deleteName : false;}
 });
});

// Files owns only its lifecycle; shared window close/drag/resize stays unchanged.
const filesDialog = document.querySelector("[data-files-dialog]");
if (filesDialog) {
  const frame = filesDialog.querySelector("[data-files-frame]");
  const state = filesDialog.querySelector("[data-files-state]");
  const retry = filesDialog.querySelector("[data-files-retry]");
  const settings = document.querySelector("[data-files-settings-dialog]");
  const settingsToggle = filesDialog.querySelector("[data-files-settings-toggle]");
  const settingsStatus = settings.querySelector("[data-files-settings-status]");
  const modes = [...settings.querySelectorAll("[data-files-mode]")];
  let mode = "on-demand";
  let generation = 0;
  let pending = Promise.resolve();
  const request = async (action, availability) => {
    const response = await fetch(action ? `/api/files/${action}` : "/api/files", {
      method: action ? "POST" : "GET", credentials: "same-origin", cache: "no-store",
      ...(action ? { body: new URLSearchParams({ csrf: filesDialog.querySelector("[data-files-csrf]").value, availability: availability || "" }) } : {}),
    });
    if (!response.ok) throw new Error(jvT("ui.32541cb0a6d8"));
    return response.json();
  };
  // Serialize lifecycle and setting requests, including rapid close/reopen.
  const mutate = (action, availability) => {
    const result = pending.then(() => request(action, availability));
    pending = result.catch(() => {});
    return result;
  };
  const selectMode = value => {
    mode = value;
    modes.forEach(input => { input.checked = input.value === mode; });
  };
  const displayError = message => {
    frame.hidden = true;
    frame.removeAttribute("src");
    state.textContent = message;
    retry.hidden = false;
  };
  let retryRefresh = false;
  const openFiles = async (refresh = false) => {
    retryRefresh = refresh;
    const ticket = ++generation;
    frame.hidden = true;
    frame.removeAttribute("src");
    retry.hidden = true;
    state.textContent = refresh ? jvT("ui.cad62bc53b77") : jvT("ui.1fcebcdd5073");
    try {
      await pending;
      if (ticket !== generation || !filesDialog.open) return;
      if (refresh) await mutate("refresh");
      if (ticket !== generation || !filesDialog.open) return;
      let status = await request();
      if (ticket !== generation || !filesDialog.open) return;
      selectMode(status.availability);
      if (!status.ready && !refresh) await mutate("start");
      // Poll only during this explicit open/refresh, at a bounded two-second interval.
      const deadline = Date.now() + 180000;
      while (!status.ready) {
        if (ticket !== generation || !filesDialog.open) return;
        if (Date.now() >= deadline) throw new Error(jvT("ui.6df217d4f82f"));
        await new Promise(resolve => setTimeout(resolve, 2000));
        if (ticket !== generation || !filesDialog.open) return;
        status = await request();
        if (!status.ready && status.state === "failed") throw new Error(jvT("ui.6df217d4f82f"));
      }
      if (ticket !== generation || !filesDialog.open) return;
      state.textContent = "";
      frame.src = "/files/";
      frame.hidden = false;
    } catch (error) {
      if (ticket === generation && filesDialog.open) displayError(error.message);
    }
  };
  const workspace = setupWorkspaceWindow(filesDialog, {
    onOpen: () => { void openFiles(); },
    onClose: () => {
      ++generation;
      frame.hidden = true;
      frame.removeAttribute("src");
      // Agent rereads the authoritative mode. It stops only in on-demand mode,
      // even if another Administrator changed availability in another tab.
      void mutate("stop").catch(() => {
        state.textContent = jvT("ui.2ba753608707");
        settingsStatus.textContent = state.textContent;
      });
    },
  });
  document.querySelector("[data-files-open]")?.addEventListener("click", () => {
    if (!document.body.classList.contains("primary-administrator")) return;
    if (controlCenter) controlCenter.open = false;
    workspace.open();
  });
  filesDialog.querySelector("[data-files-close]").addEventListener("click", () => workspace.close());
  filesDialog.querySelector("[data-files-refresh]").addEventListener("click", () => {
    void openFiles(true);
  });
  retry.addEventListener("click", () => void openFiles(retryRefresh));
  settings.querySelector("[data-files-settings-dismiss]").addEventListener("click", () => settings.close());
  settingsToggle.addEventListener("click", async () => {
    settingsStatus.textContent = "";
    modes.forEach(control => { control.disabled = true; });
    settings.showModal();
    try {
      await pending;
      selectMode((await request()).availability);
    } catch (error) { settingsStatus.textContent = error.message; }
    finally { modes.forEach(control => { control.disabled = false; }); }
  });
  modes.forEach(input => input.addEventListener("change", async () => {
    const previous = mode;
    selectMode(input.value);
    modes.forEach(control => { control.disabled = true; });
    settingsStatus.textContent = jvT("ui.23e39291d613");
    try {
      await mutate("availability", input.value);
      settingsStatus.textContent = jvT("ui.fca046ccd586");
    } catch (error) {
      // A systemd failure may follow a durable setting save. Read it back
      // instead of inventing a second local authoritative preference.
      try { selectMode((await request()).availability); }
      catch { selectMode(previous); }
      settingsStatus.textContent = jvT("ui.6e507aba2cd0");
    } finally { modes.forEach(control => { control.disabled = false; }); }
  }));
  // A selected radio remains active; clicking it again safely reapplies a
  // preference after an earlier service-control failure.
  modes.forEach(input => input.addEventListener("click", () => {
    if (input.checked && input.value === mode && settingsStatus.textContent.includes("could not be applied")) input.dispatchEvent(new Event("change"));
  }));
  const restoreFiles = () => {
    if (readWorkspaceWindowState("files").open && document.body.classList.contains("primary-administrator")) workspace.open();
  };
  if (document.body.classList.contains("role-pending")) {
    const observer = new MutationObserver(() => {
      if (document.body.classList.contains("role-pending")) return;
      observer.disconnect();
      restoreFiles();
    });
    observer.observe(document.body, { attributes: true, attributeFilter: ["class"] });
  } else restoreFiles();
}
