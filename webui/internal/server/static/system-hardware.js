(() => {
  const menu = document.querySelector("[data-control-center]");
  const open = document.querySelector("[data-display-open]");
  const dialog = document.querySelector("[data-display-dialog]");
  const indicators = document.querySelector("[data-power-indicators]");
  if (!menu || !open || !dialog || !indicators) return;
  const delay = dialog.querySelector("[data-display-delay]");
  const lid = dialog.querySelector("[data-lid-ignore]");
  const lidOption = dialog.querySelector("[data-lid-option]");
  const message = dialog.querySelector("[data-display-status]");
  const buttons = [...dialog.querySelectorAll("[data-display-action]")];
  let snapshot = null;
  let busy = false;

  const get = async (path) => {
    const response = await fetch(path, { credentials: "same-origin", cache: "no-store", headers: { Accept: "application/json" } });
    if (!response.ok) throw new Error("Hardware status is unavailable.");
    return response.json();
  };
  const batteryIndicator = (label, percentage, state, status) => {
    const item = document.createElement("span");
    item.className = "control-battery-indicator";
    const name = document.createElement("span");
    name.textContent = label;
    const icon = document.createElement("span");
    const allowed = ["online", "on_battery", "low_battery", "bypass", "unknown"];
    icon.className = "control-battery-icon system-ups-status system-ups-status-" + (allowed.includes(state) ? state : "unknown");
    const valid = typeof percentage === "number" && Number.isFinite(percentage) && percentage >= 0 && percentage <= 100;
    icon.textContent = valid ? Math.round(percentage) + "%" : "";
    icon.setAttribute("aria-label", valid ? Math.round(percentage) + "%" : "Battery percentage unavailable");
    item.title = status || state || label;
    item.append(name, icon);
    return item;
  };
  const renderIndicators = (hardware, ups) => {
    indicators.replaceChildren();
    if (hardware.hws !== true) { indicators.hidden = true; return; }
    (hardware.batteries || []).forEach((battery) => {
      const state = battery.status === "Discharging" ? (battery.capacity != null && battery.capacity <= 10 ? "low_battery" : "on_battery") : (["Charging", "Full", "Not charging"].includes(battery.status) ? "online" : "unknown");
      indicators.append(batteryIndicator("Onboard", battery.capacity, state, battery.status));
    });
    // The existing NUT path supplies the configured source, devices and states.
    if (ups && ups.available && ups.monitoring === true && ups.source && ["local", "remote"].includes(ups.source.mode)) {
      (ups.devices || []).forEach((device) => indicators.append(batteryIndicator("UPS", device.battery_charge, device.state, device.display_name)));
    }
    indicators.hidden = indicators.childElementCount === 0;
  };
  const renderControls = () => {
    const hws = snapshot && snapshot.hws === true;
    // Role visibility uses the existing nav-admin-only parent; the API also authorizes.
    open.hidden = !(hws && snapshot.display_available);
    lidOption.hidden = !(hws && snapshot.lid_available);
    lid.checked = !!(hws && snapshot.lid_ignored);
  };
  const refresh = async () => {
    open.hidden = true;
    indicators.hidden = true;
    try {
      snapshot = await get("/api/system-hardware");
      renderControls();
      let ups = null;
      if (snapshot.hws === true) {
        try { ups = await get("/api/ups"); } catch (_) { /* Onboard status remains useful independently. */ }
      }
      renderIndicators(snapshot, ups);
    } catch (_) { snapshot = null; lidOption.hidden = true; }
  };
  const submit = async (action) => {
    if (busy) return;
    const body = new URLSearchParams();
    body.set("csrf", document.querySelector("[data-system-actions-csrf]")?.value || "");
    body.set("action", action);
    if (action === "display-delay") {
      if (!delay.reportValidity()) return;
      body.set("delay_seconds", delay.value);
    }
    busy = true;
    buttons.forEach((button) => { button.disabled = true; });
    lid.disabled = true;
    message.textContent = "Applying…";
    try {
      const response = await fetch("/api/system-actions/display", { method: "POST", credentials: "same-origin", headers: { Accept: "application/json", "Content-Type": "application/x-www-form-urlencoded" }, body: body.toString() });
      if (!response.ok) throw new Error("The display or lid action could not be applied.");
      snapshot = await response.json();
      renderControls();
      message.textContent = action === "display-delay" ? "Display off scheduled in " + delay.value + " seconds." : "Applied.";
    } catch (error) {
      message.textContent = error.message;
      // Read the owned policy again after a failure; never guess its state.
      try { snapshot = await get("/api/system-hardware"); renderControls(); } catch (_) { lid.checked = !!snapshot?.lid_ignored; }
    } finally {
      busy = false;
      buttons.forEach((button) => { button.disabled = false; });
      lid.disabled = false;
    }
  };
  menu.addEventListener("toggle", () => { if (menu.open) refresh(); });
  open.addEventListener("click", () => {
    if (!snapshot?.hws || !snapshot.display_available) return;
    delay.value = "10";
    message.textContent = "";
    renderControls();
    dialog.showModal();
  });
  buttons.forEach((button) => button.addEventListener("click", () => submit(button.dataset.displayAction)));
  lid.addEventListener("change", () => submit(lid.checked ? "lid-enable" : "lid-disable"));
  dialog.querySelector("[data-display-cancel]").addEventListener("click", () => dialog.close());
})();
