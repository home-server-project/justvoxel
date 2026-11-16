(() => {
  if (!document.querySelector('link[href="/static/network-workspace.css"]')) {
    const stylesheet = document.createElement("link");
    stylesheet.rel = "stylesheet";
    stylesheet.href = "/static/network-workspace.css";
    document.head.appendChild(stylesheet);
  }

  const openButton = document.querySelector("[data-network-open]");
  const dialog = document.querySelector("[data-network-workspace-dialog]");
  if (!openButton || !dialog || typeof setupWorkspaceWindow !== "function") return;

  const closeButton = dialog.querySelector("[data-network-close]");
  const refreshButton = dialog.querySelector("[data-network-refresh]");
  const state = dialog.querySelector("[data-network-state]");
  const content = dialog.querySelector("[data-network-content]");
  const csrf = document.querySelector("[data-network-csrf]")?.value || "";
  const controlCenter = document.querySelector("[data-control-center]");
  const checkpointStorageKey = "justvoxel-network-checkpoint";

  let selectedTab = "Overview";
  let remoteProviders = [];
  let remoteError = "";
  let playitSetup = { state: "idle" };
  let playitPopup = null;
  let playitClaimOpened = false;
  let remoteBusy = false;
  let netbirdLoginURL = "";
  let tailscaleLoginURL = "";
  const validTailscaleLoginURL = (value) => {
    if (typeof value !== "string" || value.length > 2048 || /\s/.test(value)) return false;
    try {
      const url = new URL(value);
      return url.protocol === "https:" && url.host === "login.tailscale.com" &&
        url.pathname.startsWith("/a/") && url.pathname.length > 3 && !url.username && !url.password;
    } catch (_) { return false; }
  };
  const providerDashboards = { tailscale: "https://console.tailscale.com/admin/", netbird: "https://app.netbird.io/" };
  const pendingDashboards = new Set();
  const providerProgress = new Map();
  const validNetbirdLoginURL = (value) => {
    if (typeof value !== "string" || value.length > 2048 || /\s/.test(value) || /[<>"\x27]/.test(value)) return false;
    try {
      const url = new URL(value);
      const host = url.hostname.toLowerCase();
      return url.protocol === "https:" && Boolean(host) && !url.username && !url.password &&
        host !== "tailscale.com" && !host.endsWith(".tailscale.com");
    } catch (_) { return false; }
  };
  // Reuse the existing Playit setup appearance for VPN authentication.
  // Keep document.write static; set all presentation values through the DOM.
  const renderRemoteLoginPopup = (popup, id, failed = false) => {
    if (!popup || popup.closed) return;
    const providerName = id === "tailscale" ? "Tailscale" : "NetBird";
    const title = failed ? jvT("ui.e92b39d45904") : jvT("ui.389f202d8a75", {v0: (providerName)});
    const description = failed
      ? jvT("ui.c0ab9afe3c5c")
      : jvT("ui.6cc09ed67201", {v0: (providerName)});
    popup.document.open();
    popup.document.write(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title></title>
<style>
:root{color-scheme:dark;font-family:system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;color:#eef3f5;background:#0b1117}
*{box-sizing:border-box}body{margin:0;min-height:100vh;display:flex;flex-direction:column;background:#0b1117}
header{padding:18px 24px;border-bottom:1px solid #26343e;background:#0d151d;font-size:.75rem;font-weight:650;letter-spacing:.14em;color:#c8d0d6}
header .separator{margin:0 10px;color:#667782}header .brand,.eyebrow{color:#78c99d}
main{flex:1;display:grid;place-items:center;padding:32px 20px}.setup-card{width:100%;max-width:580px;padding:36px;border:1px solid #294c40;border-radius:18px;background:#111c24}
h1{margin:0 0 14px;font-size:clamp(1.4rem,4vw,1.8rem);line-height:1.25}p{font-size:.9rem;line-height:1.65;color:#ced8df}.muted{color:#9aa6b2}
.claim-panel{margin-top:28px;padding:20px;border:1px solid #273b49;border-radius:12px;background:#0d1822}.claim-heading{font-size:.82rem}
</style>
<link rel="stylesheet" href="/static/playit-setup.css">
</head>
<body>
<header aria-label=""><span>JUSTVOXEL</span><span class="separator" aria-hidden="true">·</span><span class="brand"></span></header>
<main>
<section class="setup-card" aria-labelledby="setup-title">
<p class="eyebrow">REMOTE ACCESS SETUP</p>
<h1 id="setup-title"></h1>
<p></p>
<p class="muted"></p>
<div class="claim-panel" role="status">
<h2 class="claim-heading"><span class="pulse-dot" aria-hidden="true"></span>Secure login link</h2>
<p></p>
</div>
</section>
</main>
</body>
</html>`);
    popup.document.close();
    popup.document.documentElement?.setAttribute("lang", globalThis.justVoxelLanguage?.locale || "en");
    popup.document.title = providerName + " setup";
    popup.document.querySelector("header").setAttribute("aria-label", jvT("ui.f2d34f9a3604"));
    popup.document.querySelector("header .brand").textContent = providerName;
    popup.document.querySelector(".setup-card").classList.toggle("failure", failed);
    popup.document.querySelector("#setup-title").textContent = title;
    popup.document.querySelector("#setup-title + p").textContent = description;
    popup.document.querySelector(".muted").hidden = failed;
    popup.document.querySelector(".claim-panel").hidden = failed;
    popup.document.querySelector(".muted").textContent = `This page will open ${providerName} automatically when ready.`;
    popup.document.querySelector(".claim-heading").lastChild.textContent = "Secure login link";
    popup.document.querySelector(".claim-panel p").textContent = "Waiting for the authentication link.";
  };
  const openRemoteLoginPopup = (id) => {
    let popup = null;
    try {
      // Reserve the tab during the click to avoid browser popup blockers.
      popup = window.open("about:blank", "_blank");
      if (popup) {
        try { renderRemoteLoginPopup(popup, id); }
        finally { popup.opener = null; }
      }
    } catch (_) {
      try { if (popup && !popup.closed) popup.close(); } catch (_) {}
      popup = null;
    }
    return popup;
  };
  const failRemoteLoginPopup = (popup, id) => {
    try { renderRemoteLoginPopup(popup, id, true); }
    catch (_) { try { if (popup && !popup.closed) popup.close(); } catch (_) {} }
  };
  const openActivatedDashboards = () => {
    for (const id of pendingDashboards) {
      const provider = remoteProviders.find((item) => item.id === id);
      if (!provider || provider.status_unavailable || providerTransitional(provider)) continue;
      if (!provider.service_active) {
        pendingDashboards.delete(id);
        providerProgress.set(id, jvT("ui.08952af4b5a2"));
        continue;
      }
      pendingDashboards.delete(id);
      let popup = null;
      try {
        // Open the actual destination only after the service reports running.
        popup = window.open(providerDashboards[id], "_blank");
        if (popup) popup.opener = null;
      } catch (_) { /* The fixed dashboard link remains available. */ }
      providerProgress.set(id, popup
        ? jvT("ui.107d2ab8647d")
        : jvT("ui.cb885910e721"));
    }
  };
  let remotePollSequence = 0;
  const providerTransitional = (provider) => ["activating", "deactivating", "reloading", "refreshing"].includes(provider.service_state);
  const setupRunning = () => ["starting", "waiting"].includes(playitSetup.state);
  const providerControls = (provider) => {
    const blocked = !provider.installed || remoteBusy || provider.service_state === "deactivating";
    return {
      activateDisabled: blocked || (providerTransitional(provider) && !(provider.id === "netbird" && provider.service_state === "activating")) || (["netbird", "tailscale"].includes(provider.id) ? provider.connected : provider.service_active) || (provider.id === "playit" && setupRunning()),
      deactivateDisabled: blocked || (provider.id === "nordvpn" ? !provider.service_active : (!(provider.id === "playit" && setupRunning()) && !provider.service_active && !provider.service_enabled && provider.service_state === "inactive"))
    };
  };
  const effectiveTabs = (snapshot) => (snapshot?.workspace_tabs || ["Overview"])
    .filter((tab) => tab !== "Wi-Fi" || (snapshot?.devices || []).some((device) => device.kind === "wifi"));
  const connectivitySummary = (value) => ({
    full: jvT("ui.efc751eb6e64"), limited: jvT("ui.4ead7680959f"), portal: jvT("ui.8c36e202e707"),
    none: jvT("ui.c4184bd7daac")
  }[value] || jvT("ui.676c3cd833ba"));
  const validPlayitClaim = (url) => /^https:\/\/playit\.gg\/claim\/[0-9a-fA-F]{10}(?![\s\S])$/.test(url || "");
  const renderPlayitPopup = (failed = false) => {
    if (!playitPopup || playitPopup.closed) return;
    playitPopup.document.open();
    if (failed) {
      playitPopup.document.write(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title></title>
<style>
:root{color-scheme:dark;font-family:system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;color:#eef3f5;background:#0b1117}
*{box-sizing:border-box}body{margin:0;min-height:100vh;display:flex;flex-direction:column;background:#0b1117}
header{padding:18px 24px;border-bottom:1px solid #26343e;background:#0d151d;font-size:.75rem;font-weight:650;letter-spacing:.14em;color:#c8d0d6}
header .separator{margin:0 10px;color:#667782}header .brand,.eyebrow{color:#78c99d}
main{flex:1;display:grid;place-items:center;padding:32px 20px}.setup-card{width:100%;max-width:580px;padding:36px;border:1px solid #294c40;border-radius:18px;background:#111c24}
h1{margin:0 0 14px;font-size:clamp(1.4rem,4vw,1.8rem);line-height:1.25}p{font-size:.9rem;line-height:1.65;color:#ced8df}.muted{color:#9aa6b2}
.claim-panel{margin-top:28px;padding:20px;border:1px solid #273b49;border-radius:12px;background:#0d1822}.claim-heading{font-size:.82rem}
</style>
<link rel="stylesheet" href="/static/playit-setup.css">
</head>
<body>
<header aria-label=""><span>JUSTVOXEL</span><span class="separator" aria-hidden="true">·</span><span class="brand">PLAYIT</span></header>
<main>
<section class="setup-card failure" aria-labelledby="setup-title">
<p class="eyebrow">PLAYIT SETUP</p>
<h1 id="setup-title"></h1>
<p></p>
</section>
</main>
</body>
</html>`);
    } else {
      playitPopup.document.write(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title></title>
<style>
:root{color-scheme:dark;font-family:system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;color:#eef3f5;background:#0b1117}
*{box-sizing:border-box}body{margin:0;min-height:100vh;display:flex;flex-direction:column;background:#0b1117}
header{padding:18px 24px;border-bottom:1px solid #26343e;background:#0d151d;font-size:.75rem;font-weight:650;letter-spacing:.14em;color:#c8d0d6}
header .separator{margin:0 10px;color:#667782}header .brand,.eyebrow{color:#78c99d}
main{flex:1;display:grid;place-items:center;padding:32px 20px}.setup-card{width:100%;max-width:580px;padding:36px;border:1px solid #294c40;border-radius:18px;background:#111c24}
h1{margin:0 0 14px;font-size:clamp(1.4rem,4vw,1.8rem);line-height:1.25}p{font-size:.9rem;line-height:1.65;color:#ced8df}.muted{color:#9aa6b2}
.claim-panel{margin-top:28px;padding:20px;border:1px solid #273b49;border-radius:12px;background:#0d1822}.claim-heading{font-size:.82rem}
</style>
<link rel="stylesheet" href="/static/playit-setup.css">
</head>
<body>
<header aria-label=""><span>JUSTVOXEL</span><span class="separator" aria-hidden="true">·</span><span class="brand">PLAYIT</span></header>
<main>
<section class="setup-card" aria-labelledby="setup-title">
<p class="eyebrow">PLAYIT SETUP</p>
<h1 id="setup-title"></h1>
<p></p>
<p class="muted"></p>
<div class="claim-panel" role="status">
<h2 class="claim-heading"><span class="pulse-dot" aria-hidden="true"></span> </h2>
<p></p>
</div>
</section>
</main>
</body>
</html>`);
    }
    playitPopup.document.close();
    playitPopup.document.documentElement?.setAttribute("lang", globalThis.justVoxelLanguage?.locale || "en");
    playitPopup.document.title = jvT("ui.3ff2b156e51f");
    playitPopup.document.querySelector("header").setAttribute("aria-label", jvT("ui.83f793a8f365"));
    playitPopup.document.querySelector("#setup-title").textContent = failed ? jvT("ui.999b774e4661") : jvT("ui.cab7909a1e25");
    playitPopup.document.querySelector("#setup-title + p").textContent = failed ? jvT("ui.ec8f0dee88c4") : jvT("ui.3a3a95633b6d");
    if (!failed) {
      playitPopup.document.querySelector(".muted").textContent = jvT("ui.b8b8db33c0a2");
      playitPopup.document.querySelector(".claim-heading").lastChild.textContent = jvT("ui.2ccf167b4c86");
      playitPopup.document.querySelector(".claim-panel p").textContent = jvT("ui.b8613b69d448");
    }
  };
  const failPlayitPopup = () => {
    try { renderPlayitPopup(true); } catch (_) { /* Popup may be unavailable. */ }
    // Keep the fixed failure page open, outside subsequent popup cleanup.
    playitPopup = null;
  };
  const closePlayitPopup = () => {
    try { playitPopup?.close(); } catch (_) { /* Browser may have closed it already. */ }
    playitPopup = null;
  };
  const updatePlayitPopup = () => {
    if (playitSetup.state === "waiting" && validPlayitClaim(playitSetup.claim_url) && playitPopup) {
      try {
        if (!playitPopup.closed) {
          playitPopup.location.replace(playitSetup.claim_url);
          playitClaimOpened = true;
        }
      } catch (_) { closePlayitPopup(); /* Offer the validated fallback link. */ }
      playitPopup = null;
    } else if (playitSetup.state === "failed") {
      failPlayitPopup();
    } else if (["idle", "complete"].includes(playitSetup.state)) {
      closePlayitPopup();
    }
    if (playitSetup.state === "failed") remoteError = jvT("ui.70af0fcfb7f3");
  };
  const isAdministrator = () => (currentSnapshot?.workspace_tabs || []).includes("Ethernet");
  let loadSequence = 0;
  let currentSnapshot = null;
  let currentNetworks = new Map();
  let currentCheckpoint = null;
  let connectionDraft = null;
  let checkpointTimer = null;
  let recoveryTimer = null;

  const escapePath = (value) => encodeURIComponent(String(value || ""));

  const label = (value) => String(value || "unknown").replaceAll("-", " ");

  const signalBars = (strength) => {
    const value = Number(strength || 0);
    if (value >= 75) return jvT("ui.0dc8ee555c28");
    if (value >= 50) return jvT("ui.c939327ca16d");
    if (value >= 30) return jvT("ui.f7b19afcde96");
    if (value > 0) return jvT("ui.8d6cea2517ff");
    return jvT("ui.b764cdc0eab7");
  };

  const firstAddress = (config, includePrefix = true) => {
    const address = config?.addresses?.[0];
    if (!address?.address) return jvT("ui.13075c233611");
    if (!includePrefix) return address.address.split("/")[0];
    return address.prefix !== undefined ? address.address + "/" + address.prefix : address.address;
  };

  const clearRecoveryTimer = () => {
    if (recoveryTimer) window.clearTimeout(recoveryTimer);
    recoveryTimer = null;
  };

  const scheduleRecovery = () => {
    clearRecoveryTimer();
    recoveryTimer = window.setTimeout(() => loadNetwork(), 3000);
  };

  const requestJSON = async (url, options = {}) => {
    const response = await fetch(url, {
      method: options.method || "GET",
      credentials: "same-origin",
      headers: options.headers || { Accept: "application/json" },
      body: options.body,
      cache: "no-store",
    });
    if (response.redirected) {
      const target = new URL(response.url);
      if (target.pathname === "/login" || target.pathname === "/password") {
        window.location.assign(target.pathname + target.search);
        return null;
      }
    }
    if (response.status === 401) {
      window.location.assign("/login");
      return null;
    }
    const result = await response.json().catch(() => ({}));
    if (response.status === 403 && jvC(result.error) === "password change required") {
      window.location.assign("/password");
      return null;
    }
    if (!response.ok) {
      const error = new Error(result.error || jvT("ui.cd80c924a70a"));
      error.status = response.status;
      throw error;
    }
    return result;
  };

  const postForm = (url, values = {}) => {
    const body = new URLSearchParams({ csrf });
    Object.entries(values).forEach(([key, value]) => {
      if (Array.isArray(value)) {
        value.forEach((item) => body.append(key, String(item)));
      } else if (value !== undefined && value !== null) {
        body.set(key, String(value));
      }
    });
    return requestJSON(url, {
      method: "POST",
      headers: {
        Accept: "application/json",
        "Content-Type": "application/x-www-form-urlencoded",
      },
      body: body.toString(),
    });
  };

  const row = (name, value) => {
    const item = document.createElement("div");
    item.className = "network-detail-row";
    const key = document.createElement("span");
    key.textContent = name;
    const val = document.createElement("strong");
    val.textContent = value || "—";
    item.append(key, val);
    return item;
  };

  const badge = (text, tone = "") => {
    const item = document.createElement("span");
    item.className = "badge network-badge" + (tone ? " " + tone : "");
    item.textContent = text;
    return item;
  };

  const internetStatus = (snapshot) => {
    const status = badge(
      connectivitySummary(snapshot.connectivity),
      snapshot.connectivity === "full" ? "good" : ["none", "limited", "portal"].includes(snapshot.connectivity) ? "danger" : "muted"
    );
    status.className += " network-internet-status";
    return status;
  };

  const actionButton = (text, datasetName, datasetValue, kind = "secondary") => {
    const button = document.createElement("button");
    button.type = "button";
    button.textContent = text;
    button.className = kind + " nav-admin-only network-action-button";
    button.dataset[datasetName] = datasetValue;
    return button;
  };

  const readStoredCheckpointID = () => {
    try {
      return window.sessionStorage.getItem(checkpointStorageKey) || "";
    } catch (_) {
      return "";
    }
  };

  const storeCheckpoint = (checkpoint) => {
    if (!checkpoint?.id) return;
    try {
      window.sessionStorage.setItem(checkpointStorageKey, checkpoint.id);
    } catch (_) {
      // Automatic NetworkManager rollback still protects connectivity.
    }
  };

  const clearStoredCheckpoint = () => {
    try {
      window.sessionStorage.removeItem(checkpointStorageKey);
    } catch (_) {
      // Nothing else is required.
    }
  };

  const createCheckpoint = async (interfaces) => {
    const unique = [...new Set((interfaces || []).filter(Boolean))];
    if (!unique.length) throw new Error(jvT("ui.e7842722d566"));
    const checkpoint = await postForm("/api/network/checkpoints", {
      interface: unique,
      rollback_timeout_seconds: 90,
    });
    if (!checkpoint?.id) throw new Error(jvT("ui.1af13b327171"));
    storeCheckpoint(checkpoint);
    return checkpoint;
  };

  const rollbackStoredCheckpoint = async (id) => {
    if (!id) return null;
    try {
      return await postForm("/api/network/checkpoints/" + escapePath(id) + "/rollback");
    } catch (error) {
      if (error?.status === 404) return null;
      throw error;
    }
  };

  const runCheckpointedMutation = async (interfaces, mutate) => {
    const checkpoint = await createCheckpoint(interfaces);
    try {
      const result = await mutate(checkpoint.id);
      if (result?.checkpoint?.id) storeCheckpoint(result.checkpoint);
      return result;
    } catch (error) {
      try {
        await rollbackStoredCheckpoint(checkpoint.id);
        clearStoredCheckpoint();
      } catch (_) {
        // Keep the stored ID. NetworkManager still has the automatic timeout.
      }
      throw error;
    }
  };

  const sectionHeading = (eyebrowText, titleText) => {
    const heading = document.createElement("div");
    heading.className = "section-heading";
    const copy = document.createElement("div");
    const eyebrow = document.createElement("p");
    eyebrow.className = "eyebrow";
    eyebrow.textContent = eyebrowText;
    const title = document.createElement("h2");
    title.textContent = titleText;
    copy.append(eyebrow, title);
    heading.appendChild(copy);
    return heading;
  };

  // Bound variable-length Wi-Fi lists without hiding networks or adding a scroller.
  const paginateList = (list, pageSize) => {
    const items = [...list.children];
    if (items.length <= pageSize) return null;
    let page = 0;
    const pages = Math.ceil(items.length / pageSize);
    const controls = document.createElement("div");
    controls.className = "network-list-pages";
    const previous = document.createElement("button");
    const next = document.createElement("button");
    const status = document.createElement("span");
    status.setAttribute("aria-live", "polite");
    previous.type = next.type = "button";
    previous.className = next.className = "secondary network-action-button";
    previous.textContent = jvT("ui.a57b08a480b8");
    next.textContent = jvT("ui.1ff57a29d7c9");
    const update = () => {
      items.forEach((item, index) => { item.hidden = Math.floor(index / pageSize) !== page; });
      previous.disabled = page === 0;
      next.disabled = page === pages - 1;
      status.textContent = jvT("ui.6076934f99bd") + (page + 1) + " of " + pages;
    };
    previous.addEventListener("click", () => { page--; update(); });
    next.addEventListener("click", () => { page++; update(); });
    controls.append(previous, status, next);
    update();
    return controls;
  };

  const renderConnectivity = (snapshot) => {
    const section = document.createElement("section");
    section.className = "panel details network-overview";

    const heading = document.createElement("div");
    heading.className = "section-heading";

    const status = document.createElement("div");
    status.className = "network-overview-status";
    status.append(
      internetStatus(snapshot),
      badge(snapshot.networking_enabled ? jvT("ui.e7c27b2f3aef") : jvT("ui.4bb993ee83ff"), snapshot.networking_enabled ? "good" : "danger")
    );

    const wifiDevices = (snapshot.devices || []).filter((device) => device.kind === "wifi");
    if (wifiDevices.length) {
      status.appendChild(
        badge(snapshot.wireless_enabled ? jvT("ui.0f5eddd7f886") : jvT("ui.ba356882b016"), snapshot.wireless_enabled ? "" : "muted")
      );
    } else {
      status.appendChild(badge(jvT("ui.a328fef833c4"), "muted"));
    }

    heading.appendChild(status);

    const subtitle = document.createElement("p");
    subtitle.className = "state-text network-overview-version";
    subtitle.textContent = jvT("ui.b1771a332842") + (snapshot.version || "unknown version");

    section.append(heading, subtitle);
    return section;
  };

  const renderOverview = (snapshot) => {
    const section = document.createElement("section");
    section.className = "panel details network-overview";
    const heading = internetStatus(snapshot);
    const devices = (snapshot.devices || []).filter((device) => ["ethernet", "wifi"].includes(device.kind));
    const primary = devices.find((device) => device.state === "activated" && device.ipv4?.gateway && device.ipv4?.addresses?.length)
      || devices.find((device) => device.state === "activated" && device.ipv4?.addresses?.length);
    const ethernet = devices.find((device) => device.kind === "ethernet" && device.state === "activated")
      || devices.find((device) => device.kind === "ethernet");
    section.append(heading,
      row("IPv4", primary ? firstAddress(primary.ipv4, false) : jvT("ui.ca1844969742")),
      row("Ethernet", ethernet ? ethernet.interface + " · " + (ethernet.state === "activated" && ethernet.carrier !== false ? jvT("ui.22965568d22a") : jvT("ui.04dfac3671b4")) : jvT("ui.ca1844969742")));
    for (const [id, name] of [["tailscale", "Tailscale"], ["netbird", "NetBird"], ["playit", "Playit"]]) {
      const provider = remoteProviders.find((item) => item.id === id);
      section.appendChild(row(name, provider?.connected && provider.ip ? provider.ip.split("/")[0] : provider?.summary || jvT("ui.ca1844969742")));
    }
    return section;
  };

  const renderCheckpoint = (checkpoint) => {
    const section = document.createElement("section");
    section.className = "network-checkpoint-banner";
    section.dataset.networkCheckpointBanner = checkpoint.id;

    const text = document.createElement("div");
    const title = document.createElement("strong");
    title.textContent = jvT("ui.0e32df3f9764");
    const detail = document.createElement("span");
    detail.dataset.networkCheckpointCountdown = checkpoint.expires_at || "";
    text.append(title, detail);

    const actions = document.createElement("div");
    const keep = document.createElement("button");
    keep.type = "button";
    keep.dataset.networkCheckpointKeep = checkpoint.id;
    keep.textContent = jvT("ui.0e8ac53f7659");
    const revert = document.createElement("button");
    revert.type = "button";
    revert.className = "secondary";
    revert.dataset.networkCheckpointRevert = checkpoint.id;
    revert.textContent = jvT("ui.9cf62b1f1c72");
    actions.append(revert, keep);

    section.append(text, actions);
    return section;
  };

  const updateCheckpointCountdown = () => {
    const element = content.querySelector("[data-network-checkpoint-countdown]");
    if (!element) return;
    const expires = Date.parse(element.dataset.networkCheckpointCountdown || "");
    if (!Number.isFinite(expires)) {
      element.textContent = jvT("ui.c8c0983ef060");
      return;
    }
    const seconds = Math.max(0, Math.ceil((expires - Date.now()) / 1000));
    element.textContent = seconds > 0
      ? jvT("ui.39dbca2fc797") + seconds + " seconds."
      : jvT("ui.dd3ca3c47cb5");
    if (seconds === 0) {
      clearStoredCheckpoint();
      currentCheckpoint = null;
      if (checkpointTimer) window.clearInterval(checkpointTimer);
      checkpointTimer = null;
      scheduleRecovery();
    }
  };

  const startCheckpointCountdown = () => {
    if (checkpointTimer) window.clearInterval(checkpointTimer);
    updateCheckpointCountdown();
    checkpointTimer = window.setInterval(updateCheckpointCountdown, 1000);
  };

  const renderDevice = (device, snapshot = null) => {
    const card = document.createElement("article");
    card.className = "panel network-device-card";

    const heading = document.createElement("div");
    heading.className = "network-device-heading";
    const titleWrap = document.createElement("div");
    const kind = document.createElement("span");
    kind.className = "eyebrow network-device-kind";
    kind.textContent = device.kind === "wifi" ? "Wi-Fi" : device.kind === "ethernet" ? "Ethernet" : jvT("ui.1744b96470b5");
    const title = document.createElement("strong");
    title.textContent = device.interface || jvT("ui.f9781840b7a4");
    titleWrap.append(kind, title);

    const stateBadge = badge(label(device.state), device.state === "activated" ? "good" : device.state === "failed" ? "danger" : "");
    if (device.kind === "ethernet" && snapshot) {
      const status = document.createElement("div");
      status.className = "network-inline-actions";
      status.append(stateBadge, internetStatus(snapshot));
      heading.append(titleWrap, status);
    } else {
      heading.append(titleWrap, stateBadge);
    }
    card.appendChild(heading);

    const summary = document.createElement("div");
    summary.className = "network-device-summary";
    if (device.kind === "wifi") {
      const ssid = device.wireless?.ssid || jvT("ui.0303e1824670");
      summary.append(
        row(jvT("ui.1744b96470b5"), ssid),
        row(jvT("ui.1e9806e4227b"), device.wireless?.ssid ? signalBars(device.wireless?.signal) + " · " + (device.wireless?.signal || 0) + "%" : "—"),
        row("IPv4", firstAddress(device.ipv4, false))
      );
    } else {
      summary.append(
        row(jvT("ui.a6a32dbc5618"), device.carrier === false ? jvT("ui.bc9c4d065b35") : device.speed_mbps ? device.speed_mbps + " Mbps" : jvT("ui.22965568d22a")),
        row("IPv4", firstAddress(device.ipv4, false)),
        row(jvT("ui.d696a35bdd18"), device.active_connection?.id || jvT("ui.ee3e198aa826")),
        row(jvT("ui.41ed52921661"), device.ipv4?.gateway || "—"),
        row("DNS", device.ipv4?.dns?.join(", ") || "—"),
        row("MTU", device.mtu ? String(device.mtu) : "—")
      );
    }

    const details = document.createElement("details");
    details.className = "network-device-details";
    const detailsSummary = document.createElement("summary");
    detailsSummary.textContent = jvT("ui.51bfffbcdbcf");
    const body = document.createElement("div");
    body.className = "network-detail-grid";
    body.append(
      row(jvT("ui.8f2de600bf94"), device.managed ? jvT("ui.85a39ab345d6") : jvT("ui.1ea442a134b2")),
      row(jvT("ui.8036352a05b7"), device.hardware_address || "—"),
      row("MTU", device.mtu ? String(device.mtu) : "—"),
      row(jvT("ui.41ed52921661"), device.ipv4?.gateway || "—"),
      row("DNS", device.ipv4?.dns?.join(", ") || "—"),
      row("IPv6", firstAddress(device.ipv6)),
      row(device.kind === "ethernet" ? jvT("ui.a33a8f572209") : jvT("ui.a04e87255298"), device.active_connection ? (device.active_connection.autoconnect ? "On" : jvT("ui.ca7981b46ecf")) : "—"),
      row(device.kind === "ethernet" ? jvT("ui.469e3c7987e4") : jvT("ui.9fdc9f555c8a"), device.active_connection?.ipv4_method || "—"),
      row("IPv6 mode", device.active_connection?.ipv6_method || "—")
    );
    details.append(detailsSummary, body);
    card.append(summary, details);
    return card;
  };

  const supportedNewNetwork = (network) =>
    ["open", "owe", "wpa-personal", "wpa3-personal"].includes(network.security);

  const renderWiFiNetworks = (device, response) => {
    const section = document.createElement("section");
    section.className = "panel network-wifi-section";

    const heading = document.createElement("div");
    heading.className = "network-section-heading";
    const headingText = document.createElement("div");
    const title = document.createElement("h3");
    title.textContent = jvT("ui.f2e9d2c6727e");
    const subtitle = document.createElement("span");
    subtitle.textContent = device.interface;
    headingText.append(title, subtitle);

    const headingActions = document.createElement("div");
    headingActions.className = "network-inline-actions";
    const toggle = actionButton(
      currentSnapshot.wireless_enabled ? jvT("ui.815fc0039346") : jvT("ui.3cbfdf6e372f"),
      "networkWifiRadio", currentSnapshot.wireless_enabled ? "off" : "on"
    );
    const other = actionButton(jvT("ui.366763cbc9fd"), "networkWifiOther", device.interface, "secondary");
    const scan = document.createElement("button");
    scan.type = "button";
    scan.className = "primary network-scan-button";
    scan.dataset.networkScan = device.interface;
    scan.textContent = jvT("ui.ea1c510d1ddb");
    headingActions.append(toggle, other, scan);
    heading.append(headingText, headingActions);
    section.appendChild(heading);

    const list = document.createElement("div");
    list.className = "network-wifi-list";
    const networks = response?.networks || [];
    if (networks.length === 0) {
      const empty = document.createElement("p");
      empty.className = "network-empty";
      empty.textContent = jvT("ui.441c0756b589");
      list.appendChild(empty);
    } else {
      networks.forEach((network) => {
        const item = document.createElement("div");
        item.className = "network-wifi-row" + (network.active ? " active" : "");
        const main = document.createElement("div");
        const name = document.createElement("strong");
        name.textContent = network.hidden ? jvT("ui.366763cbc9fd") : network.ssid || jvT("ui.2f23a756dea1");
        const meta = document.createElement("span");
        const parts = [signalBars(network.strength), String(network.strength || 0) + "%", label(network.security)];
        if (network.known) parts.push(jvT("ui.b5c120b316c2"));
        meta.textContent = parts.join(" · ");
        main.append(name, meta);
        item.appendChild(main);

        const actions = document.createElement("div");
        actions.className = "network-wifi-row-actions";
        if (network.active) {
          actions.append(badge(jvT("ui.22965568d22a"), "good"), actionButton(jvT("ui.acfc5be785a9"), "networkWifiDisconnect", device.interface, "danger"));
        } else if (network.profile_uuid) {
          const connect = actionButton(jvT("ui.1a2303ede074"), "networkWifiSavedConnect", device.interface);
          connect.dataset.profileUuid = network.profile_uuid;
          actions.appendChild(connect);
        } else if (!network.hidden && supportedNewNetwork(network)) {
          const join = actionButton(jvT("ui.fd30fe681b4b"), "networkWifiJoin", device.interface);
          join.dataset.ssid = network.ssid || "";
          join.dataset.bssid = network.bssid || "";
          join.dataset.keyManagement = network.key_management || "";
          join.dataset.security = network.security || "";
          actions.appendChild(join);
        } else {
          const unavailable = document.createElement("span");
          unavailable.className = "network-wifi-unavailable";
          unavailable.textContent = network.security === "enterprise" ? jvT("ui.e4f3d1d1205d") : jvT("ui.54324658e2eb");
          actions.appendChild(unavailable);
        }
        item.appendChild(actions);
        list.appendChild(item);
      });
    }
    section.appendChild(list);
    return section;
  };

  const activeProfileUUIDs = (snapshot) => new Set(
    (snapshot.devices || [])
      .map((device) => device.active_connection?.uuid)
      .filter(Boolean)
  );

  const defaultWiFiInterface = (snapshot, profile) => {
    const devices = (snapshot.devices || []).filter((device) => device.kind === "wifi");
    if (!devices.length) return "";
    if (profile.interface_name && devices.some((device) => device.interface === profile.interface_name)) {
      return profile.interface_name;
    }
    return devices[0].interface;
  };

  const renderSavedWiFi = (snapshot) => {
    const profiles = (snapshot.profiles || []).filter((profile) => profile.type === "802-11-wireless");
    if (profiles.length === 0) return null;
    const active = activeProfileUUIDs(snapshot);
    const details = document.createElement("details");
    details.className = "panel network-saved";
    const summary = document.createElement("summary");
    summary.textContent = jvT("ui.c2aac85d761d") + profiles.length + ")";
    const list = document.createElement("div");
    list.className = "network-saved-list";
    profiles.forEach((profile) => {
      const item = document.createElement("div");
      item.className = "network-saved-row";
      const info = document.createElement("div");
      const name = document.createElement("strong");
      name.textContent = profile.ssid || profile.id || jvT("ui.b2e35fe02f63");
      const meta = document.createElement("span");
      meta.textContent = (profile.autoconnect ? jvT("ui.a04e87255298") : jvT("ui.b0b9fe24ffa9")) + " · " + label(profile.key_management || "open");
      info.append(name, meta);
      item.appendChild(info);

      const actions = document.createElement("div");
      actions.className = "network-wifi-row-actions nav-admin-only";
      if (active.has(profile.uuid)) {
        actions.appendChild(badge(jvT("ui.22965568d22a"), "good"));
      } else {
        const interfaceName = defaultWiFiInterface(snapshot, profile);
        if (interfaceName) {
          const connect = actionButton(jvT("ui.1a2303ede074"), "networkWifiSavedConnect", interfaceName);
          connect.dataset.profileUuid = profile.uuid;
          actions.appendChild(connect);
        }
        const forget = actionButton(jvT("ui.a6bd489d9550"), "networkWifiForget", profile.uuid, "secondary");
        actions.appendChild(forget);
      }
      item.appendChild(actions);
      list.appendChild(item);
    });
    details.append(summary, list);
    const pages = paginateList(list, 3);
    if (pages) details.appendChild(pages);
    return details;
  };

  const renderConnectPanel = () => {
    if (!connectionDraft) return null;
    const panel = document.createElement("section");
    panel.className = "panel network-connect-panel nav-admin-only";
    panel.dataset.networkConnectPanel = "true";

    const heading = document.createElement("div");
    const title = document.createElement("strong");
    title.textContent = connectionDraft.hidden ? jvT("ui.fbeebc16df15") : jvT("ui.bb23428b6599") + connectionDraft.ssid;
    const subtitle = document.createElement("span");
    subtitle.textContent = connectionDraft.interface;
    heading.append(title, subtitle);
    panel.appendChild(heading);

    const form = document.createElement("form");
    form.dataset.networkConnectForm = "true";
    form.dataset.interface = connectionDraft.interface;
    form.dataset.bssid = connectionDraft.bssid || "";
    form.dataset.hidden = connectionDraft.hidden ? "true" : "false";

    if (connectionDraft.hidden) {
      const ssidLabel = document.createElement("label");
      ssidLabel.textContent = jvT("ui.b97af0128946");
      const ssid = document.createElement("input");
      ssid.name = "ssid";
      ssid.required = true;
      ssid.maxLength = 32;
      ssid.autocomplete = "off";
      ssidLabel.appendChild(ssid);
      form.appendChild(ssidLabel);

      const securityLabel = document.createElement("label");
      securityLabel.textContent = jvT("ui.8f6fb4eb7f42");
      const security = document.createElement("select");
      security.name = "key_management";
      [
        ["", jvT("ui.ed077f3d8125")],
        ["wpa-psk", "WPA/WPA2 Personal"],
        ["sae", "WPA3 Personal"],
        ["owe", jvT("ui.3e104a888955")],
      ].forEach(([value, text]) => {
        const option = document.createElement("option");
        option.value = value;
        option.textContent = text;
        security.appendChild(option);
      });
      securityLabel.appendChild(security);
      form.appendChild(securityLabel);
    } else {
      const ssid = document.createElement("input");
      ssid.type = "hidden";
      ssid.name = "ssid";
      ssid.value = connectionDraft.ssid;
      form.appendChild(ssid);
      const keyManagement = document.createElement("input");
      keyManagement.type = "hidden";
      keyManagement.name = "key_management";
      keyManagement.value = connectionDraft.keyManagement || "";
      form.appendChild(keyManagement);
    }

    const passwordLabel = document.createElement("label");
    passwordLabel.className = "network-connect-password";
    passwordLabel.textContent = jvT("ui.e7cf3ef4f17c");
    const password = document.createElement("input");
    password.type = "password";
    password.name = "password";
    password.autocomplete = "new-password";
    password.spellcheck = false;
    passwordLabel.appendChild(password);
    form.appendChild(passwordLabel);

    const hint = document.createElement("small");
    hint.className = "network-connect-hint";
    hint.textContent = jvT("ui.b9a44f748b66");
    form.appendChild(hint);

    const actions = document.createElement("div");
    actions.className = "network-connect-actions";
    const cancel = document.createElement("button");
    cancel.type = "button";
    cancel.className = "secondary";
    cancel.dataset.networkConnectCancel = "true";
    cancel.textContent = jvT("ui.19766ed6ccb2");
    const connect = document.createElement("button");
    connect.type = "submit";
    connect.textContent = jvT("ui.1a2303ede074");
    actions.append(cancel, connect);
    form.appendChild(actions);
    panel.appendChild(form);

    const updatePasswordVisibility = () => {
      const key = form.elements.key_management?.value || connectionDraft.keyManagement || "";
      const needsPassword = key === "wpa-psk" || key === "sae";
      passwordLabel.hidden = !needsPassword;
      password.required = needsPassword;
      if (!needsPassword) password.value = "";
    };
    form.elements.key_management?.addEventListener("change", updatePasswordVisibility);
    updatePasswordVisibility();

    return panel;
  };

  const renderEthernetForm = (device) => {
    const form = document.createElement("form");
    form.className = "settings-form network-ethernet-form";
    form.dataset.networkEthernetForm = device.interface;
    form.dataset.profileUuid = device.active_connection.uuid;
    const field = (name, title, value, type = "text") => {
      const wrap = document.createElement("label");
      wrap.textContent = title;
      const input = document.createElement("input");
      input.name = name;
      input.type = type;
      input.value = value ?? "";
      if (type === "checkbox") {
        input.checked = Boolean(value);
        wrap.className = "toggle-line";
        wrap.appendChild(input);
      }
      if (type !== "checkbox") wrap.appendChild(input);
      form.appendChild(wrap);
      return input;
    };
    const methodLabel = document.createElement("label");
    methodLabel.textContent = jvT("ui.9fdc9f555c8a");
    const method = document.createElement("select");
    method.name = "method";
    [["auto", jvT("ui.df7c8f177d2e")], ["manual", jvT("ui.b0b9fe24ffa9")]].forEach(([value, text]) => {
      const option = document.createElement("option");
      option.value = value;
      option.textContent = text;
      method.appendChild(option);
    });
    method.value = device.active_connection.ipv4_method === "manual" ? "manual" : "auto";
    methodLabel.appendChild(method);
    form.appendChild(methodLabel);
    const address = field("address", "IPv4 address", device.ipv4?.addresses?.[0]?.address);
    const prefix = field("prefix", jvT("ui.9b7eecc8b23d"), device.ipv4?.addresses?.[0]?.prefix ?? 24, "number");
    prefix.min = "0"; prefix.max = "32";
    const gateway = field("gateway", jvT("ui.9a3176cdc431"), device.ipv4?.gateway);
    field("dns", jvT("ui.a0d86937a4d2"), (device.ipv4?.dns || []).join(", "));
    const mtu = field("mtu", jvT("ui.a77024ba1df5"), device.mtu || 0, "number");
    mtu.min = "0"; mtu.max = "9000";
    const autoconnect = field("autoconnect", jvT("ui.bc4b00e5e098"), device.active_connection.autoconnect, "checkbox");
    const updateMode = () => {
      [address, prefix, gateway].forEach((input) => { input.disabled = method.value === "auto"; });
      address.required = method.value === "manual";
      prefix.required = method.value === "manual";
    };
    method.addEventListener("change", updateMode);
    updateMode();
    const hint = document.createElement("p");
    hint.className = "state-text";
    hint.textContent = jvT("ui.c8fa1cb5a4eb");
    const apply = document.createElement("button");
    apply.type = "submit";
    apply.textContent = jvT("ui.31e392d1c037");
    apply.disabled = Boolean(currentCheckpoint);
    const actions = document.createElement("div");
    actions.className = "network-form-actions";
    actions.appendChild(apply);
    const information = document.createElement("div");
    information.className = "network-form-information";
    information.append(hint, autoconnect.parentElement);
    form.append(information, actions);
    return form;
  };

  const renderTroubleshoot = (snapshot) => {
    const section = document.createElement("section");
    section.className = "panel network-section network-troubleshoot";
    const heading = document.createElement("div");
    heading.className = "network-health-heading";
    const title = document.createElement("p");
    title.className = "eyebrow";
    title.textContent = jvT("ui.d30f3eece6eb");
    const status = internetStatus(snapshot);
    heading.append(title, status);
    if (["none", "limited", "portal"].includes(snapshot.connectivity)) {
      heading.appendChild(actionButton(jvT("ui.5fd2684fde6b"), "networkConnectivityCheck", "true"));
    }
    section.appendChild(heading);
    const grid = document.createElement("div");
    grid.className = "network-troubleshoot-devices";
    if (!snapshot.networking_enabled) section.appendChild(row(jvT("ui.39b36d38d6eb"), jvT("ui.cd204d443cd2")));
    const devices = (snapshot.devices || []).filter((device) => ["ethernet", "wifi"].includes(device.kind));
    if (!devices.length) section.appendChild(row(jvT("ui.39b36d38d6eb"), jvT("ui.b285d70453d6")));
    devices.forEach((device) => {
      const card = document.createElement("article");
      card.className = "panel network-device-card";
      const summary = document.createElement("div");
      summary.className = "network-health-summary";
      summary.append(
        row(jvT("ui.c26b3ed4ce70"), device.interface), row(jvT("ui.a3b50c476732"), label(device.state)),
        row(jvT("ui.a6a32dbc5618"), device.carrier === false ? jvT("ui.388105a792f1") : device.carrier === true ? jvT("ui.14494dd61bac") : jvT("ui.adadface0166")),
        row("IPv4", firstAddress(device.ipv4, false)),
        row(jvT("ui.063ef3af2949"), device.ipv4?.gateway || jvT("ui.6be36ca49ee8")),
        row(jvT("ui.a384b75cca98"), device.ipv4?.dns?.join(", ") || jvT("ui.6be36ca49ee8")),
        row(jvT("ui.5608b89a8112"), device.active_connection?.id || jvT("ui.6be36ca49ee8"))
      );
      card.appendChild(summary);
      if (device.active_connection && device.managed) {
        const repair = actionButton(jvT("ui.83b2f2e0db6b"), "networkReconnect", device.interface);
        repair.dataset.profileUuid = device.active_connection.uuid;
        repair.disabled = Boolean(currentCheckpoint);
        const actions = document.createElement("div");
        actions.className = "network-inline-actions";
        const explanation = document.createElement("p");
        explanation.className = "state-text";
        explanation.textContent = jvT("ui.eee1d2bb2ac7");
        actions.classList.add("network-reconnect-information");
        actions.append(explanation, repair);
        card.appendChild(actions);
      }
      grid.appendChild(card);
    });
    section.appendChild(grid);
    return section;
  };

  // Account actions reuse setup state and polling, offering the validated link
  // for an explicit click. Initial Activate retains its existing popup behavior.
  const startPlayitAccountSetup = (relink) => {
    closePlayitPopup();
    playitClaimOpened = false;
    return runRemoteAction(async () => {
      try {
        playitSetup = await postForm(relink ? "/api/network/remote-access/playit/relink" : "/api/network/remote-access/playit/setup");
        updatePlayitPopup();
      } catch (_) {
        throw new Error(relink ? jvT("ui.a4d81f64042c") : jvT("ui.79af4d3a3060"));
      }
    });
  };
  let playitAccountDialog = null;
  const showPlayitAccountDialog = () => {
    if (playitAccountDialog?.open || remoteBusy || setupRunning()) return;
    const popup = document.createElement("dialog");
    playitAccountDialog = popup;
    popup.className = "system-action-dialog";
    const body = document.createElement("div");
    body.className = "system-action-dialog-content";
    const title = document.createElement("h2");
    title.id = "playit-account-title";
    title.textContent = jvT("ui.0b2d3d5d7041");
    popup.setAttribute("aria-labelledby", title.id);
    const message = document.createElement("p");
    message.textContent = jvT("ui.67ad6c402ff2");
    const actions = document.createElement("div"); actions.className = "action-row";
    const cancel = document.createElement("button");
    cancel.type = "button"; cancel.className = "secondary"; cancel.textContent = jvT("ui.19766ed6ccb2");
    cancel.addEventListener("click", () => popup.close());
    const relink = document.createElement("button");
    relink.type = "button"; relink.textContent = jvT("ui.beaddb19f0ef");
    relink.addEventListener("click", () => {
      if (remoteBusy || setupRunning()) return;
      popup.close();
      startPlayitAccountSetup(true);
    });
    actions.append(cancel, relink);
    body.append(title, message, actions); popup.appendChild(body);
    popup.addEventListener("close", () => { popup.remove(); playitAccountDialog = null; });
    document.body.appendChild(popup); popup.showModal();
  };

  const validNordAccessToken = (value) =>
    typeof value === "string" && value.length > 0 && value.length <= 4096 && !/[^\x21-\x7e]/.test(value);
  let nordDialog = null;
  const showNordAccountDialog = () => {
    if (nordDialog?.open || remoteBusy) return;
    const popup = document.createElement("dialog");
    nordDialog = popup;
    popup.className = "system-action-dialog";
    const body = document.createElement("div");
    body.className = "system-action-dialog-content";
    const title = document.createElement("h2");
    title.id = "nord-login-title";
    title.textContent = jvT("ui.09eb206a54f3");
    popup.setAttribute("aria-labelledby", title.id);
    const message = document.createElement("p");
    message.textContent = jvT("ui.44487688c2b1");
    const details = document.createElement("div");
    const existing = document.createElement("p"); existing.textContent = jvT("ui.2c47eff53d0b");
    const dashboard = document.createElement("a");
    dashboard.href = "https://my.nordaccount.com/dashboard/nordvpn/";
    dashboard.target = "_blank"; dashboard.rel = "noopener noreferrer";
    dashboard.className = "button-link"; dashboard.textContent = "Open NordVPN dashboard";
    const newAccount = document.createElement("p"); newAccount.textContent = jvT("ui.842fffba1546");
    const referral = document.createElement("a");
    referral.href = "https://refer-nordvpn.com/pZGMpGTrBrO";
    referral.target = "_blank"; referral.rel = "noopener noreferrer";
    referral.className = "button-link"; referral.textContent = "Create NordVPN account";
    const heading = document.createElement("h3"); heading.textContent = jvT("ui.5d6fd1f1b37a");
    const instructions = document.createElement("ol");
    for (const text of [
      jvT("ui.30abb1c510b5"),
      jvT("ui.cca9199c9a20"),
      jvT("ui.8113b4f1ba8d"),
      jvT("ui.48fb704b886d"),
      jvT("ui.7281848cc133"),
      jvT("ui.29d65fd1755d")
    ]) {
      const step = document.createElement("li"); step.textContent = text; instructions.appendChild(step);
    }
    const note = document.createElement("small");
    note.className = "muted";
    note.textContent = jvT("ui.e114463706d5");
    const label = document.createElement("label"); label.textContent = jvT("ui.9a911b08feb9");
    const input = document.createElement("input");
    input.type = "password"; input.maxLength = 4096;
    input.autocomplete = "off"; input.spellcheck = false; input.autocapitalize = "off";
    label.appendChild(input);
    details.append(existing, dashboard, newAccount, referral, heading, instructions, note, label);
    const status = document.createElement("p"); status.setAttribute("role", "status");
    const actions = document.createElement("div"); actions.className = "action-row";
    let submitting = false;
    const cancel = document.createElement("button");
    cancel.type = "button"; cancel.className = "secondary"; cancel.textContent = jvT("ui.19766ed6ccb2");
    cancel.addEventListener("click", () => {
      if (submitting) return;
      input.value = "";
      popup.close();
    });
    const login = document.createElement("button"); login.type = "button"; login.textContent = jvT("ui.f3455b1b00f5");
    login.addEventListener("click", async () => {
      if (remoteBusy) return;
      if (!validNordAccessToken(input.value)) {
        input.value = "";
        status.textContent = jvT("ui.3347fdbbfc43");
        return;
      }
      let token = input.value;
      input.value = "";
      submitting = true;
      input.disabled = true;
      cancel.disabled = true;
      login.disabled = true;
      login.textContent = jvT("ui.27b80374e115");
      remoteBusy = true;
      renderNetwork();
      try {
        await postForm("/api/network/remote-access/nordvpn/login", { token: token });
        if (popup.open) popup.close();
        await refreshRemoteAccess();
      } catch (error) {
        const nordLoginFailureMessages = new Set([
          jvT("ui.bea0b17a3742"),
          jvT("ui.6bd3689fa539")
        ]);
        if (popup.open) {
          status.textContent = nordLoginFailureMessages.has(error?.message)
            ? error.message
            : jvT("ui.bea0b17a3742");
        }
      } finally {
        token = "";
        input.value = "";
        submitting = false;
        input.disabled = false;
        cancel.disabled = false;
        login.disabled = false;
        login.textContent = jvT("ui.f3455b1b00f5");
        remoteBusy = false;
        renderNetwork();
      }
    });
    actions.append(cancel, login);
    body.append(title, message, details, status, actions);
    popup.appendChild(body);
    popup.addEventListener("close", () => {
      input.value = "";
      details.replaceChildren();
      popup.remove();
      if (nordDialog === popup) nordDialog = null;
    });
    document.body.appendChild(popup); popup.showModal();
  };

  let nordLoginChoice = null;
  const showNordLoginChoice = () => {
    if (nordLoginChoice?.open || remoteBusy) return;
    const popup = document.createElement("dialog");
    nordLoginChoice = popup;
    popup.className = "system-action-dialog";
    const body = document.createElement("div"); body.className = "system-action-dialog-content";
    const title = document.createElement("h2"); title.id = "nord-login-choice-title";
    title.textContent = jvT("ui.4250b2dbec33");
    popup.setAttribute("aria-labelledby", title.id);
    const message = document.createElement("p");
    message.textContent = jvT("ui.70dd45762de1");
    const actions = document.createElement("div"); actions.className = "action-row";
    const cancel = document.createElement("button");
    cancel.type = "button"; cancel.className = "secondary"; cancel.textContent = jvT("ui.19766ed6ccb2");
    cancel.addEventListener("click", () => popup.close());
    const update = document.createElement("button");
    update.type = "button"; update.className = "secondary"; update.textContent = "Update Token";
    update.addEventListener("click", () => { popup.close(); showNordAccountDialog(); });
    const current = document.createElement("button");
    current.type = "button"; current.textContent = jvT("ui.e80ef082465d");
    current.addEventListener("click", async () => {
      if (remoteBusy) return;
      popup.close();
      let loginRequired = false;
      await runRemoteAction(async () => {
        const result = await postForm("/api/network/remote-access/nordvpn", { action: "activate" });
        loginRequired = Boolean(result?.login_required);
      });
      if (loginRequired) showNordAccountDialog();
    });
    actions.append(cancel, update, current);
    body.append(title, message, actions); popup.appendChild(body);
    popup.addEventListener("close", () => {
      popup.remove();
      if (nordLoginChoice === popup) nordLoginChoice = null;
    });
    document.body.appendChild(popup); popup.showModal();
  };

  const providerIcons = new Map([
    ["tailscale", "/static/provider-icons/tailscale.svg"],
    ["netbird", "/static/provider-icons/netbird.svg"],
    ["playit", "/static/provider-icons/playit.svg"],
    ["nordvpn", "/static/provider-icons/nordvpn.svg"],
  ]);

  const renderRemoteAccess = () => {
    const section = document.createElement("section");
    section.className = "panel network-section";
    section.appendChild(sectionHeading(jvT("ui.66354a1d3225"), jvT("ui.996c32b35f21")));
    const grid = document.createElement("div");
    grid.className = "network-provider-grid";
    const hint = document.createElement("p");
    hint.className = "state-text";
    hint.textContent = remoteError || jvT("ui.26992440ff9a");
    section.appendChild(hint);
    remoteProviders.forEach((provider) => {
      const card = document.createElement("article");
      card.className = "panel network-device-card network-provider-card";
      const title = document.createElement("h3");
      title.textContent = provider.display_name;
      const iconPath = providerIcons.get(provider.id);
      if (iconPath) {
        const icon = document.createElement("img");
        icon.className = "network-provider-icon";
        icon.src = iconPath;
        icon.alt = "";
        icon.setAttribute("aria-hidden", "true");
        title.prepend(icon);
      }
      card.append(title,
        row(jvT("ui.9b3289a385a5"), provider.status_unavailable ? jvT("ui.b764cdc0eab7") : provider.installed ? jvT("ui.f8b32f4e92bd") : jvT("ui.ca1844969742")),
        row(jvT("ui.af06465d485d"), provider.status_unavailable ? jvT("ui.b764cdc0eab7") : provider.service_enabled ? jvT("ui.85a39ab345d6") : jvT("ui.1ea442a134b2")),
        row(jvT("ui.d677190e0a99"), providerTransitional(provider) ? provider.service_state[0].toUpperCase() + provider.service_state.slice(1) + "…" : provider.service_active ? jvT("ui.f4ccae29e1bb") : label(provider.service_state).replace(/^./, (c) => c.toUpperCase())),
        row(provider.id === "nordvpn" ? "Account" : jvT("ui.b332c3492d5e"), provider.id === "nordvpn" ? provider.account || jvT("ui.b764cdc0eab7") : provider.status_unavailable ? jvT("ui.b764cdc0eab7") : provider.configured ? jvT("ui.84aebc69a1bf") : jvT("ui.dd1841d29502"))
      );
      if (provider.id === "nordvpn") {
        card.appendChild(row(jvT("ui.3a9f6b579cd4"), provider.meshnet || jvT("ui.b764cdc0eab7")));
        if (provider.nord_name) card.appendChild(row(jvT("ui.29c5504827ca"), provider.nord_name));
        if (provider.ip) card.appendChild(row(jvT("ui.7ce7f3a81eb0"), provider.ip));
      }
      if (provider.id === "netbird" && provider.connected && provider.ip) card.appendChild(row(jvT("ui.52513d9a9d28"), provider.ip.split("/")[0]));
      if (provider.id === "tailscale") {
        card.appendChild(row(jvT("ui.639a40e82b9a"), provider.summary || jvT("ui.b764cdc0eab7")));
        if (provider.connected && provider.ip) card.appendChild(row(jvT("ui.a452ee8ba9a0"), provider.ip.split("/")[0]));
      }
      if (provider.id === "playit") {
        card.appendChild(row(jvT("ui.2e65f2b5ee02"), provider.summary === "Running" ? jvT("ui.ca1844969742") : provider.summary || jvT("ui.ca1844969742")));
        (provider.tunnels || []).forEach((tunnel) => {
          card.appendChild(row(tunnel.is_disabled ? jvT("ui.0401b21b94d9") : jvT("ui.6f213980234a"), tunnel.display_address));
          card.appendChild(row(jvT("ui.16eb689dc945"), tunnel.destination));
        });
      }
      if (providerProgress.has(provider.id)) {
        const progress = document.createElement("p");
        progress.className = "state-text";
        progress.setAttribute("role", "status");
        progress.textContent = providerProgress.get(provider.id);
        card.appendChild(progress);
      }
      const needsSetup = provider.id === "playit" && provider.installed && (!provider.configured || setupRunning());
      const actions = document.createElement("div");
      actions.className = "network-inline-actions";
      if (provider.id === "playit") actions.className += " network-playit-lifecycle";
      if (provider.id === "nordvpn") actions.className += " network-nord-lifecycle";
      if (needsSetup) {
        if (setupRunning()) card.appendChild(row(jvT("ui.7013af4c42fe"), playitSetup.state === "waiting" ? jvT("ui.10c5739b95bd") : jvT("ui.5d1fa38bcf0d")));
        if (playitSetup.state === "failed") card.appendChild(row(jvT("ui.7013af4c42fe"), jvT("ui.089c1f975bf6")));
      }
      const controls = providerControls(provider);
      const activate = actionButton(jvT("ui.24433c70eba5"), "networkProvider", provider.id, "primary");
      activate.dataset.action = "activate";
      activate.disabled = controls.activateDisabled;
      const deactivate = actionButton(jvT("ui.fb1e6fa55327"), "networkProvider", provider.id);
      deactivate.dataset.action = "deactivate";
      deactivate.disabled = controls.deactivateDisabled;
      actions.appendChild(deactivate);
      if (provider.id === "playit") {
        const rightActions = document.createElement("div");
        rightActions.className = "network-playit-actions";
        const account = actionButton(jvT("ui.7e1b0d5641f2"), "networkProvider", provider.id);
        account.dataset.action = "account";
        account.disabled = !provider.installed || remoteBusy || setupRunning();
        rightActions.append(account, activate);
        actions.appendChild(rightActions);
      }
      if (needsSetup && playitSetup.state === "waiting" && !playitClaimOpened && validPlayitClaim(playitSetup.claim_url)) {
        const claim = document.createElement("a");
        claim.href = playitSetup.claim_url;
        claim.target = "_blank";
        claim.rel = "noopener noreferrer";
        claim.className = "button-link primary network-action-button";
        claim.textContent = jvT("ui.c2f09f26c012");
        actions.appendChild(claim);
      } else if (provider.id === "nordvpn") {
        const rightActions = document.createElement("div");
        rightActions.className = "network-nord-actions";
        const token = actionButton(jvT("ui.d2089be67295"), "networkProvider", provider.id);
        token.dataset.action = "token";
        token.disabled = !provider.installed || remoteBusy;
        rightActions.append(token, activate);
        actions.appendChild(rightActions);
      } else if (provider.id !== "playit") {
        actions.appendChild(activate);
      }
      if (provider.id === "tailscale" && !provider.connected && validTailscaleLoginURL(tailscaleLoginURL)) {
        const login = document.createElement("a");
        login.href = tailscaleLoginURL;
        login.target = "_blank";
        login.rel = "noopener noreferrer";
        login.className = "button-link primary network-action-button";
        login.textContent = jvT("ui.2146b7fe17c8");
        actions.appendChild(login);
      }
      if (provider.id === "netbird" && !provider.connected && validNetbirdLoginURL(netbirdLoginURL)) {
        const login = document.createElement("a");
        login.href = netbirdLoginURL;
        login.target = "_blank";
        login.rel = "noopener noreferrer";
        login.className = "button-link primary network-action-button";
        login.textContent = jvT("ui.4426a088b7b0");
        actions.appendChild(login);
      }
      const dashboard = document.createElement("a");
      // Fixed destinations only, even if a malformed status payload is received.
      dashboard.href = { tailscale: "https://console.tailscale.com/admin/", netbird: "https://app.netbird.io/", playit: "https://playit.gg/account/", nordvpn: "https://my.nordaccount.com/" }[provider.id];
      dashboard.target = "_blank";
      dashboard.rel = "noopener noreferrer";
      dashboard.className = "button-link warning network-action-button";
      dashboard.textContent = { tailscale: jvT("ui.4691212b71d0"), netbird: jvT("ui.92a48aa76b62"), playit: jvT("ui.a661f3935b88"), nordvpn: jvT("ui.044dd47673b3") }[provider.id];
      actions.appendChild(dashboard);
      if (activate.parentElement === actions) actions.appendChild(activate);
      card.appendChild(actions);
      grid.appendChild(card);
    });
    section.appendChild(grid);
    return section;
  };

  const renderNetwork = (checkpoint = currentCheckpoint) => {
    if (!currentSnapshot) return;
    currentCheckpoint = checkpoint || null;
    const root = document.createElement("div");
    root.className = "network-workspace-view";
    const tabs = effectiveTabs(currentSnapshot);
    if (!tabs.includes(selectedTab)) selectedTab = "Overview";
    const tablist = document.createElement("nav");
    tablist.className = "system-workspace-tabs network-tabs";
    tablist.setAttribute("role", "tablist");
    tablist.setAttribute("aria-label", jvT("ui.2a6073b40239"));
    tabs.forEach((name) => {
      const button = document.createElement("button");
      button.type = "button";
      button.dataset.networkTab = name;
      button.textContent = jvS(name);
      button.setAttribute("role", "tab");
      button.setAttribute("aria-selected", String(name === selectedTab));
      button.tabIndex = name === selectedTab ? 0 : -1;
      button.setAttribute("aria-controls", "network-tab-panel");
      button.id = "network-tab-" + name.replaceAll(" ", "-");
      tablist.appendChild(button);
    });
    root.append(tablist, state);
    if (currentCheckpoint && isAdministrator()) root.appendChild(renderCheckpoint(currentCheckpoint));
    const panel = document.createElement("div");
    panel.id = "network-tab-panel";
    panel.dataset.networkView = selectedTab;
    panel.setAttribute("role", "tabpanel");
    panel.setAttribute("aria-labelledby", "network-tab-" + selectedTab.replaceAll(" ", "-"));
    const devices = (currentSnapshot.devices || []).filter((device) => ["ethernet", "wifi"].includes(device.kind));
    if (selectedTab === "Overview") {
      panel.appendChild(renderOverview(currentSnapshot));
    } else if (selectedTab === "Ethernet") {
      devices.filter((device) => device.kind === "ethernet").forEach((device) => {
        const card = renderDevice(device, currentSnapshot);
        if (currentCheckpoint?.interfaces?.includes(device.interface)) card.appendChild(row(jvT("ui.aff0766a5290"), jvT("ui.4376411d3c2e")));
        else if (device.active_connection && device.state === "activated") card.appendChild(renderEthernetForm(device));
        else card.appendChild(row(jvT("ui.b332c3492d5e"), jvT("ui.30e7963b582f")));
        panel.appendChild(card);
      });
      if (!devices.some((device) => device.kind === "ethernet")) panel.appendChild(row("Ethernet", jvT("ui.ea4aaf4c3791")));
    } else if (selectedTab === "Wi-Fi") {
      panel.appendChild(renderConnectivity(currentSnapshot));
      const layout = document.createElement("div");
      layout.className = "network-wifi-layout";
      const settings = document.createElement("div");
      settings.className = "network-wifi-settings";
      const nearby = document.createElement("div");
      nearby.className = "network-wifi-nearby";
      devices.filter((device) => device.kind === "wifi").forEach((device) => {
        settings.appendChild(renderDevice(device));
        nearby.appendChild(renderWiFiNetworks(device, currentNetworks.get(device.interface)));
      });
      const saved = renderSavedWiFi(currentSnapshot);
      if (saved) settings.appendChild(saved);
      const connectPanel = renderConnectPanel();
      if (connectPanel) settings.appendChild(connectPanel);
      layout.append(settings, nearby);
      panel.appendChild(layout);
    } else if (selectedTab === "Troubleshoot") {
      panel.appendChild(renderTroubleshoot(currentSnapshot));
    } else if (selectedTab === "Remote Access") {
      panel.appendChild(renderRemoteAccess());
    }
    root.appendChild(panel);
    content.replaceChildren(root);
    if (currentCheckpoint) startCheckpointCountdown();
  };

  const resumeCheckpoint = async () => {
    const id = readStoredCheckpointID();
    if (!id) return null;
    try {
      return await requestJSON("/api/network/checkpoints/" + escapePath(id));
    } catch (error) {
      if (error?.status === 404) {
        clearStoredCheckpoint();
        return null;
      }
      throw error;
    }
  };

  const loadNetwork = async () => {
    const sequence = ++loadSequence;
    clearRecoveryTimer();
    refreshButton.disabled = true;
    state.textContent = jvT("ui.a62ed9454c60");
    try {
      const snapshot = await requestJSON("/api/network");
      if (!snapshot || sequence !== loadSequence) return;

      currentSnapshot = snapshot;
      currentCheckpoint = isAdministrator() ? (await resumeCheckpoint() || snapshot.pending_checkpoints?.[0] || null) : null;
      if (currentCheckpoint) storeCheckpoint(currentCheckpoint);
      if (sequence !== loadSequence) return;
      currentNetworks = new Map();
      const wifiDevices = (snapshot.devices || []).filter((device) => device.kind === "wifi");
      for (const device of isAdministrator() ? wifiDevices : []) {
        const networks = await requestJSON("/api/network/wifi/" + escapePath(device.interface) + "/networks");
        if (sequence !== loadSequence) return;
        currentNetworks.set(device.interface, networks);
      }

      remoteProviders = [
        { id: "tailscale", display_name: "Tailscale", status_unavailable: true },
        { id: "netbird", display_name: "NetBird", status_unavailable: true },
        { id: "playit", display_name: "Playit.gg", status_unavailable: true },
        { id: "nordvpn", display_name: jvT("ui.efbb9e0df755"), status_unavailable: true },
      ];
      remoteError = "";
      try {
        const remote = await requestJSON("/api/network/remote-access");
        if (sequence !== loadSequence) return;
        remoteProviders = (remote?.providers || []).filter((provider) => ["tailscale", "netbird", "playit", "nordvpn"].includes(provider.id));
        if (isAdministrator() && selectedTab === "Remote Access") {
          playitSetup = await requestJSON("/api/network/remote-access/playit/setup") || { state: "idle" };
        }
      } catch (error) { remoteError = error?.message || jvT("ui.d6b5cd96e853"); }
      if (sequence !== loadSequence) return;
      updatePlayitPopup();
      openActivatedDashboards();
      renderNetwork();
      state.textContent = "";
      if (remoteProviders.some(providerTransitional) || setupRunning()) pollRemoteAccess();
    } catch (error) {
      if (sequence !== loadSequence) return;
      state.textContent = error?.message || jvT("ui.e9e5872f8df9");
      if (readStoredCheckpointID()) {
        state.textContent += jvT("ui.1ad0f8c0f24a");
        scheduleRecovery();
      }
    } finally {
      if (sequence === loadSequence) refreshButton.disabled = false;
    }
  };

  const requestScan = async (interfaceName, button) => {
    button.disabled = true;
    button.textContent = jvT("ui.38d96da6e8b2");
    try {
      await postForm("/api/network/wifi/" + escapePath(interfaceName) + "/scan");
      window.setTimeout(loadNetwork, 900);
    } catch (error) {
      state.textContent = error?.message || jvT("ui.8b311001f6ea");
    } finally {
      button.disabled = false;
      button.textContent = jvT("ui.ea1c510d1ddb");
    }
  };

  const setRadio = async (enabled, button) => {
    button.disabled = true;
    state.textContent = enabled ? jvT("ui.fb62ab30dd6e") : jvT("ui.8fcf1e7523d7");
    try {
      if (enabled) {
        await postForm("/api/network/wifi/radio", { enabled: true });
        await loadNetwork();
        return;
      }
      const interfaces = (currentSnapshot?.devices || [])
        .filter((device) => device.kind === "wifi")
        .map((device) => device.interface);
      await runCheckpointedMutation(interfaces, (checkpointID) =>
        postForm("/api/network/wifi/radio", { enabled: false, checkpoint_id: checkpointID })
      );
      await loadNetwork();
    } catch (error) {
      state.textContent = error?.message || jvT("ui.bbe1d8a1c316");
      if (readStoredCheckpointID()) scheduleRecovery();
    } finally {
      button.disabled = false;
    }
  };

  const connectSaved = async (interfaceName, profileUUID, button) => {
    button.disabled = true;
    state.textContent = jvT("ui.3da4bd0d247d");
    try {
      await runCheckpointedMutation([interfaceName], (checkpointID) =>
        postForm("/api/network/wifi/" + escapePath(interfaceName) + "/connect", {
          checkpoint_id: checkpointID,
          profile_uuid: profileUUID,
        })
      );
      await loadNetwork();
    } catch (error) {
      state.textContent = error?.message || jvT("ui.72747c73c861");
      if (readStoredCheckpointID()) scheduleRecovery();
    } finally {
      button.disabled = false;
    }
  };

  const disconnectWiFi = async (interfaceName, button) => {
    button.disabled = true;
    state.textContent = jvT("ui.f7d90587893d");
    try {
      await runCheckpointedMutation([interfaceName], (checkpointID) =>
        postForm("/api/network/wifi/" + escapePath(interfaceName) + "/disconnect", {
          checkpoint_id: checkpointID,
        })
      );
      await loadNetwork();
    } catch (error) {
      state.textContent = error?.message || jvT("ui.05d16938890e");
      if (readStoredCheckpointID()) scheduleRecovery();
    } finally {
      button.disabled = false;
    }
  };

  const forgetWiFi = async (profileUUID, button) => {
    button.disabled = true;
    state.textContent = jvT("ui.c5d2d8af4452");
    try {
      await postForm("/api/network/wifi/profiles/" + escapePath(profileUUID) + "/forget");
      await loadNetwork();
    } catch (error) {
      state.textContent = error?.message || jvT("ui.342d17f0088a");
    } finally {
      button.disabled = false;
    }
  };

  const submitNewWiFi = async (form) => {
    const button = form.querySelector('button[type="submit"]');
    if (button) button.disabled = true;
    const interfaceName = form.dataset.interface || "";
    const passwordInput = form.elements.password;
    const password = passwordInput?.value || "";
    if (passwordInput) passwordInput.value = "";

    state.textContent = jvT("ui.3da4bd0d247d");
    try {
      await runCheckpointedMutation([interfaceName], (checkpointID) =>
        postForm("/api/network/wifi/" + escapePath(interfaceName) + "/connect", {
          checkpoint_id: checkpointID,
          ssid: form.elements.ssid?.value || "",
          bssid: form.dataset.bssid || "",
          key_management: form.elements.key_management?.value || connectionDraft?.keyManagement || "",
          password,
          hidden: form.dataset.hidden === "true",
        })
      );
      connectionDraft = null;
      await loadNetwork();
    } catch (error) {
      state.textContent = error?.message || jvT("ui.0d6c96df3093");
      if (readStoredCheckpointID()) scheduleRecovery();
      if (button) button.disabled = false;
    }
  };

  const confirmCheckpoint = async (id, button) => {
    button.disabled = true;
    state.textContent = jvT("ui.ebf818aa4414");
    try {
      await postForm("/api/network/checkpoints/" + escapePath(id) + "/confirm");
      clearStoredCheckpoint();
      currentCheckpoint = null;
      await loadNetwork();
    } catch (error) {
      // A persistence failure can occur after successful checkpoint acceptance.
      // Recover terminal state without hiding the failure message.
      try {
        const pending = await resumeCheckpoint();
        if (!pending) { currentCheckpoint = null; renderNetwork(); }
      } catch (_) { /* Connectivity safety remains owned by NetworkManager. */ }
      state.textContent = error?.message || jvT("ui.0237be5d159e");
      button.disabled = false;
    }
  };

  const revertCheckpoint = async (id, button) => {
    button.disabled = true;
    state.textContent = jvT("ui.f81c97b0cbd9");
    try {
      await rollbackStoredCheckpoint(id);
      clearStoredCheckpoint();
      currentCheckpoint = null;
      connectionDraft = null;
      scheduleRecovery();
    } catch (error) {
      state.textContent = error?.message || jvT("ui.e6b11c7adcf9");
      button.disabled = false;
    }
  };

  const runConfigurationAction = async (button, action) => {
    button.disabled = true;
    try { await action(); await loadNetwork(); }
    catch (error) {
      state.textContent = error?.message || jvT("ui.75dcb9e07392");
      if (readStoredCheckpointID()) scheduleRecovery();
    } finally { button.disabled = false; }
  };

  const refreshRemoteAccess = async () => {
    if (selectedTab === "Remote Access") playitSetup = await requestJSON("/api/network/remote-access/playit/setup") || { state: "idle" };
    const remote = await requestJSON("/api/network/remote-access");
    remoteProviders = (remote?.providers || []).filter((provider) => ["tailscale", "netbird", "playit", "nordvpn"].includes(provider.id));
    updatePlayitPopup();
    openActivatedDashboards();
    if (remoteProviders.some((provider) => provider.id === "netbird" && provider.connected)) {
      netbirdLoginURL = "";
      providerProgress.delete("netbird");
    }
    if (remoteProviders.some((provider) => provider.id === "tailscale" && provider.connected)) {
      tailscaleLoginURL = "";
      providerProgress.delete("tailscale");
    }
    renderNetwork();
  };

  // Each refresh/action starts a finite observation window. Closing the workspace
  // ends observation; the agent still owns the setup timeout.
  const pollRemoteAccess = async () => {
    const sequence = ++remotePollSequence;
    const deadline = Date.now() + (setupRunning() ? 610000 : (providerProgress.has("netbird") || providerProgress.has("tailscale")) ? 120000 : 20000);
    try {
      while (sequence === remotePollSequence && dialog.open && isAdministrator() && Date.now() < deadline &&
             (remoteProviders.some(providerTransitional) || setupRunning() || providerProgress.has("netbird") || providerProgress.has("tailscale"))) {
        await new Promise((resolve) => setTimeout(resolve, setupRunning() || providerProgress.has("netbird") || providerProgress.has("tailscale") ? 2000 : 750));
        if (sequence !== remotePollSequence || !dialog.open || !isAdministrator()) return;
        await refreshRemoteAccess();
      }
      if (sequence === remotePollSequence) {
        if (providerProgress.has("netbird")) providerProgress.set("netbird", jvT("ui.69f389548f99"));
        if (providerProgress.has("tailscale")) providerProgress.set("tailscale", jvT("ui.555fd8bc2259"));
        pendingDashboards.forEach((id) => providerProgress.set(id, jvT("ui.647444b35c30")));
        pendingDashboards.clear();
        renderNetwork();
      }
    } catch (error) {
      failPlayitPopup();
      remoteError = error?.message || jvT("ui.2a14bb279de6");
      renderNetwork();
    }
  };

  const runRemoteAction = async (action) => {
    if (remoteBusy) return;
    remoteBusy = true;
    remoteError = "";
    ++remotePollSequence;
    renderNetwork();
    try {
      await action();
      await refreshRemoteAccess();
      pollRemoteAccess();
    } catch (error) {
      failPlayitPopup();
      remoteError = error?.message || jvT("ui.00b9c29d129c");
      try { await refreshRemoteAccess(); pollRemoteAccess(); } catch (_) { /* Keep the reported failure visible. */ }
    } finally {
      remoteBusy = false;
      renderNetwork();
    }
  };

  const submitEthernet = (form) => {
    const values = Object.fromEntries(new FormData(form));
    if (values.method === "auto") { values.address = ""; values.prefix = 0; values.gateway = ""; }
    values.autoconnect = form.elements.autoconnect.checked;
    values.profile_uuid = form.dataset.profileUuid;
    values.mtu = values.mtu || 0;
    const interfaceName = form.dataset.networkEthernetForm;
    return runConfigurationAction(form.querySelector('button[type="submit"]'), () =>
      runCheckpointedMutation([interfaceName], (checkpointID) =>
        postForm("/api/network/ethernet/" + escapePath(interfaceName), { ...values, checkpoint_id: checkpointID })
      )
    );
  };

  content.addEventListener("click", (event) => {
    const tab = event.target.closest("[data-network-tab]");
    if (tab) {
      selectedTab = tab.dataset.networkTab;
      connectionDraft = null;
      renderNetwork();
      if (selectedTab === "Remote Access") refreshRemoteAccess().then(pollRemoteAccess).catch((error) => { state.textContent = error?.message || jvT("ui.d6b5cd96e853"); });
      content.querySelector('[aria-selected="true"]')?.focus();
      return;
    }
    if (!isAdministrator()) return;
    const provider = event.target.closest("[data-network-provider]");
    if (provider) {
      if (provider.disabled || remoteBusy) return;
      const id = provider.dataset.networkProvider;
      if (id === "playit" && provider.dataset.action === "account") {
        const current = remoteProviders.find((item) => item.id === id);
        if (!current?.installed || setupRunning()) return;
        if (current.configured) showPlayitAccountDialog();
        else startPlayitAccountSetup(false);
        return;
      }
      const needsSetup = id === "playit" && provider.dataset.action === "activate" &&
        !remoteProviders.find((item) => item.id === id)?.configured;
      if (needsSetup) {
        closePlayitPopup();
        playitClaimOpened = false;
        // Open during the click; sever the opener before navigating the validated claim.
        try {
          playitPopup = window.open("about:blank", "_blank");
          if (playitPopup) {
            if (playitPopup.document.documentElement) {
              playitPopup.document.documentElement.style.backgroundColor = "#0b1117";
              playitPopup.document.documentElement.style.colorScheme = "dark";
            }
            try { renderPlayitPopup(); }
            finally { playitPopup.opener = null; }
          }
        } catch (_) { closePlayitPopup(); }
        runRemoteAction(async () => {
          // The existing setup endpoint enables/starts the fixed service before setup.
          playitSetup = await postForm("/api/network/remote-access/playit/setup");
          updatePlayitPopup();
        });
      } else if (id === "nordvpn" && provider.dataset.action === "token") {
        showNordAccountDialog();
      } else if (id === "nordvpn" && provider.dataset.action === "activate") {
        showNordLoginChoice();
      } else if (id === "tailscale" && provider.dataset.action === "activate") {
        const loginTab = openRemoteLoginPopup(id);
        providerProgress.set(id, jvT("ui.29e09fc7a949"));
        runRemoteAction(async () => {
          try {
            const result = await postForm("/api/network/remote-access/tailscale", { action: "activate" });
            if (result?.login_url && !validTailscaleLoginURL(result.login_url)) throw new Error(jvT("ui.d53bb6301580"));
            tailscaleLoginURL = result?.login_url || "";
            if (tailscaleLoginURL) {
              providerProgress.set(id, jvT("ui.86667cf2364e"));
              if (loginTab && !loginTab.closed) {
                try { loginTab.location.replace(tailscaleLoginURL); }
                catch (_) { failRemoteLoginPopup(loginTab, id); /* Keep the login link in the card. */ }
              }
            } else {
              try { if (loginTab && !loginTab.closed) loginTab.close(); } catch (_) {}
              providerProgress.set(id, jvT("ui.69eef3f45ce2"));
            }
          } catch (error) {
            failRemoteLoginPopup(loginTab, id);
            providerProgress.delete(id);
            throw error;
          }
        });
      } else if (id === "netbird" && provider.dataset.action === "activate") {
        const loginTab = openRemoteLoginPopup(id);
        providerProgress.set(id, jvT("ui.161479722141"));
        runRemoteAction(async () => {
          try {
            const result = await postForm("/api/network/remote-access/netbird", { action: "activate" });
            if (result?.login_url && !validNetbirdLoginURL(result.login_url)) throw new Error(jvT("ui.981f67cede6f"));
            netbirdLoginURL = result?.login_url || "";
            if (netbirdLoginURL) {
              providerProgress.set(id, jvT("ui.f32f7fd2b4b2"));
              if (loginTab && !loginTab.closed) {
                try { loginTab.location.replace(netbirdLoginURL); }
                catch (_) { failRemoteLoginPopup(loginTab, id); /* Keep the login link in the card. */ }
              }
            } else {
              // NetBird may already have a saved login and connect without a URL.
              try { if (loginTab && !loginTab.closed) loginTab.close(); } catch (_) {}
              providerProgress.set(id, jvT("ui.d7d7b58a77d7"));
            }
          } catch (error) {
            failRemoteLoginPopup(loginTab, id);
            providerProgress.delete(id);
            throw error;
          }
        });
      } else {
        if (id === "playit" && provider.dataset.action === "deactivate") closePlayitPopup();
        const action = provider.dataset.action;
        if (action === "activate" && providerDashboards[id]) {
          providerProgress.set(id, jvT("ui.e7785e94e602"));
        } else {
          pendingDashboards.delete(id);
          providerProgress.delete(id);
        }
        runRemoteAction(async () => {
          try {
            await postForm("/api/network/remote-access/" + escapePath(id), { action });
            if (action === "activate" && providerDashboards[id]) pendingDashboards.add(id);
          } catch (error) {
            providerProgress.delete(id);
            throw error;
          }
        });
      }
      return;
    }
    const check = event.target.closest("[data-network-connectivity-check]");
    if (check) { runConfigurationAction(check, () => postForm("/api/network/connectivity-check")); return; }
    const repair = event.target.closest("[data-network-reconnect]");
    if (repair) {
      runConfigurationAction(repair, () => runCheckpointedMutation([repair.dataset.networkReconnect], (checkpointID) =>
        postForm("/api/network/reconnect/" + escapePath(repair.dataset.networkReconnect), { profile_uuid: repair.dataset.profileUuid, checkpoint_id: checkpointID })
      ));
      return;
    }
    const scan = event.target.closest("[data-network-scan]");
    if (scan) {
      requestScan(scan.dataset.networkScan, scan);
      return;
    }

    const radio = event.target.closest("[data-network-wifi-radio]");
    if (radio) {
      setRadio(radio.dataset.networkWifiRadio === "on", radio);
      return;
    }

    const saved = event.target.closest("[data-network-wifi-saved-connect]");
    if (saved) {
      connectSaved(saved.dataset.networkWifiSavedConnect, saved.dataset.profileUuid, saved);
      return;
    }

    const disconnect = event.target.closest("[data-network-wifi-disconnect]");
    if (disconnect) {
      disconnectWiFi(disconnect.dataset.networkWifiDisconnect, disconnect);
      return;
    }

    const forget = event.target.closest("[data-network-wifi-forget]");
    if (forget) {
      forgetWiFi(forget.dataset.networkWifiForget, forget);
      return;
    }

    const join = event.target.closest("[data-network-wifi-join]");
    if (join) {
      connectionDraft = {
        interface: join.dataset.networkWifiJoin,
        ssid: join.dataset.ssid || "",
        bssid: join.dataset.bssid || "",
        keyManagement: join.dataset.keyManagement || "",
        security: join.dataset.security || "",
        hidden: false,
      };
      renderNetwork();
      content.querySelector("[data-network-connect-panel] input")?.focus({ preventScroll: true });
      return;
    }

    const other = event.target.closest("[data-network-wifi-other]");
    if (other) {
      connectionDraft = {
        interface: other.dataset.networkWifiOther,
        ssid: "",
        bssid: "",
        keyManagement: "",
        hidden: true,
      };
      renderNetwork();
      content.querySelector("[data-network-connect-panel] input")?.focus({ preventScroll: true });
      return;
    }

    if (event.target.closest("[data-network-connect-cancel]")) {
      connectionDraft = null;
      renderNetwork();
      return;
    }

    const keep = event.target.closest("[data-network-checkpoint-keep]");
    if (keep) {
      confirmCheckpoint(keep.dataset.networkCheckpointKeep, keep);
      return;
    }

    const revert = event.target.closest("[data-network-checkpoint-revert]");
    if (revert) {
      revertCheckpoint(revert.dataset.networkCheckpointRevert, revert);
    }
  });

  content.addEventListener("keydown", (event) => {
    const tab = event.target.closest("[data-network-tab]");
    if (!tab || !["ArrowLeft", "ArrowRight", "Home", "End"].includes(event.key)) return;
    event.preventDefault();
    const tabs = effectiveTabs(currentSnapshot);
    let index = tabs.indexOf(selectedTab);
    if (event.key === "Home") index = 0;
    else if (event.key === "End") index = tabs.length - 1;
    else index = (index + (event.key === "ArrowRight" ? 1 : -1) + tabs.length) % tabs.length;
    selectedTab = tabs[index];
    connectionDraft = null;
    renderNetwork();
    if (selectedTab === "Remote Access") refreshRemoteAccess().then(pollRemoteAccess).catch((error) => { state.textContent = error?.message || jvT("ui.d6b5cd96e853"); });
    content.querySelector('[aria-selected="true"]')?.focus();
  });

  content.addEventListener("submit", (event) => {
    if (!isAdministrator()) { event.preventDefault(); return; }
    const ethernet = event.target.closest("[data-network-ethernet-form]");
    if (ethernet) { event.preventDefault(); submitEthernet(ethernet); return; }
    const form = event.target.closest("[data-network-connect-form]");
    if (!form) return;
    event.preventDefault();
    submitNewWiFi(form);
  });

  const workspace = setupWorkspaceWindow(dialog, { onOpen: loadNetwork });
  openButton.addEventListener("click", () => {
    if (controlCenter) controlCenter.open = false;
    workspace?.open();
  });
  closeButton?.addEventListener("click", () => workspace?.close());
  refreshButton?.addEventListener("click", loadNetwork);

  if (readWorkspaceWindowState("network").open) workspace?.open();
})();
