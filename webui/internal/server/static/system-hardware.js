(() => {
  const menu = document.querySelector("[data-control-center]");
  const power = document.querySelector("[data-system-power]");
  const open = document.querySelector("[data-display-open]");
  const dialog = document.querySelector("[data-display-dialog]");
  const indicators = document.querySelector("[data-power-indicators]");
  if (!menu || !power || !open || !dialog || !indicators) return;
  const delay = dialog.querySelector("[data-display-delay]");
  const timer = dialog.querySelector("[data-display-timer]");
  const display = dialog.querySelector("[data-display-command]");
  const lid = dialog.querySelector("[data-lid-ignore]");
  const lidOption = dialog.querySelector("[data-lid-option]");
  const message = dialog.querySelector("[data-display-status]");
  const apply = dialog.querySelector("[data-display-apply]");
  const cancel = dialog.querySelector("[data-display-cancel]");
  let snapshot = null;
  let busy = false;
  let opening = false;
  let refreshRequest = 0;
  let displayDirty = false;
  // Session-only desired command. This is deliberately not physical readback.
  let displaySelection = true;

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
    const charge = document.createElement("span");
    const allowed = ["online", "on_battery", "low_battery", "bypass", "unknown"];
    charge.className = "system-ups-status system-ups-status-" + (allowed.includes(state) ? state : "unknown");
    const valid = typeof percentage === "number" && Number.isFinite(percentage) && percentage >= 0 && percentage <= 100;
    charge.textContent = valid ? Math.round(percentage) + "%" : "";
    item.title = status || state || label;
    item.append(name, charge);
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
  const refresh = async () => {
    const request = ++refreshRequest;
    power.dataset.hardwareSettled = "false";
    open.hidden = true;
    indicators.hidden = true;
    let hardware;
    try {
      hardware = await get("/api/system-hardware");
      if (request !== refreshRequest) return;
      snapshot = hardware;
      open.hidden = !(hardware.hws === true && hardware.display_available);
    } catch (_) {
      if (request === refreshRequest) snapshot = null;
      return;
    } finally {
      if (request === refreshRequest) power.dataset.hardwareSettled = "true";
    }
    let ups = null;
    if (hardware.hws === true) {
      try { ups = await get("/api/ups"); } catch (_) { /* Onboard status remains useful independently. */ }
    }
    if (request === refreshRequest) renderIndicators(hardware, ups);
  };
  const setInputs = () => {
    const available = snapshot?.hws === true && snapshot.display_available;
    timer.disabled = busy || !available;
    delay.disabled = busy || !available || !timer.checked;
    display.disabled = busy || !available;
    lid.disabled = busy || !(snapshot?.hws === true && snapshot.lid_available);
    apply.disabled = busy || !available;
    cancel.disabled = busy;
  };
  const submit = async () => {
    if (busy || !snapshot?.hws || !snapshot.display_available) return;
    if (timer.checked && !delay.reportValidity()) return;
    const actions = [];
    const immediateOff = displayDirty && !display.checked;
    if (displayDirty) actions.push(display.checked ? "display-on" : "display-off");
    if (timer.checked && !immediateOff) actions.push("display-delay");
    if (snapshot.lid_available && lid.checked !== snapshot.lid_ignored) actions.push(lid.checked ? "lid-enable" : "lid-disable");
    busy = true;
    setInputs();
    message.textContent = "";
    try {
      for (const action of actions) {
        const body = new URLSearchParams();
        body.set("csrf", document.querySelector("[data-system-actions-csrf]")?.value || "");
        body.set("action", action);
        if (action === "display-delay") body.set("delay_seconds", delay.value);
        const response = await fetch("/api/system-actions/display", { method: "POST", credentials: "same-origin", headers: { Accept: "application/json", "Content-Type": "application/x-www-form-urlencoded" }, body: body.toString() });
        if (!response.ok) throw new Error("The display or lid action could not be applied.");
        snapshot = await response.json();
        // Clear each successful command before attempting the next one.
        if (action === "display-on" || action === "display-off") {
          displaySelection = action === "display-on";
          displayDirty = false;
          if (action === "display-off") timer.checked = false;
        }
        if (action === "display-delay") timer.checked = false;
      }
      snapshot = await get("/api/system-hardware");
      lid.checked = !!snapshot.lid_ignored;
    } catch (error) {
      message.textContent = error.message;
      // Reconcile owned policy after partial failure; never infer display state.
      try { snapshot = await get("/api/system-hardware"); } catch (_) { /* Keep the last authoritative response. */ }
      lid.checked = !!snapshot?.lid_ignored;
    } finally {
      busy = false;
      setInputs();
    }
  };
  menu.addEventListener("toggle", () => { if (menu.open) refresh(); });
  open.addEventListener("click", async () => {
    if (opening || busy || open.hidden) return;
    opening = true;
    message.textContent = "";
    try {
      snapshot = await get("/api/system-hardware");
      if (!snapshot.hws || !snapshot.display_available) { open.hidden = true; return; }
    } catch (error) {
      snapshot = null;
      message.textContent = error.message;
    } finally {
      opening = false;
    }
    delay.value = "10";
    timer.checked = false;
    display.checked = displaySelection;
    displayDirty = false;
    lidOption.hidden = !(snapshot?.hws === true && snapshot.lid_available);
    lid.checked = !!snapshot?.lid_ignored;
    setInputs();
    dialog.showModal();
  });
  timer.addEventListener("change", setInputs);
  display.addEventListener("change", () => { displayDirty = true; });
  apply.addEventListener("click", submit);
  cancel.addEventListener("click", () => { if (!busy) dialog.close(); });
  dialog.addEventListener("cancel", (event) => { if (busy) event.preventDefault(); });
})();
