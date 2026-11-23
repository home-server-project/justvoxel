document.addEventListener("DOMContentLoaded", () => {
  const applyForm = document.querySelector("[data-setup-apply]");
  if (applyForm) {
    const addingServer = applyForm.dataset.addServer === "true";
    const eulaDialog = document.querySelector("[data-setup-eula-dialog]");
    const passwordDialog = document.querySelector("[data-setup-password-dialog]");
    const password = passwordDialog?.querySelector("[data-setup-password]");
    const configure = document.querySelector("[data-setup-configure]");
    let passwordConfirmed = false;
    let submitting = false;

    configure?.addEventListener("click", (event) => {
      if (applyForm.dataset.smbRequired === "true" && !passwordConfirmed) {
        event.preventDefault();
        if (passwordDialog && !passwordDialog.open) passwordDialog.showModal();
      } else if (applyForm.dataset.eulaAccepted !== "true") {
        event.preventDefault();
        if (eulaDialog && !eulaDialog.open) eulaDialog.showModal();
      }
    });
    passwordDialog?.querySelector("[data-setup-password-cancel]")?.addEventListener("click", () => passwordDialog.close());
    passwordDialog?.addEventListener("close", () => {
      if (!passwordConfirmed && password) password.value = "";
    });
    const continueWithPassword = () => {
      if (!password?.reportValidity()) return;
      passwordConfirmed = true;
      passwordDialog.close();
      if (applyForm.dataset.eulaAccepted !== "true" && eulaDialog && !eulaDialog.open) eulaDialog.showModal();
      else applyForm.requestSubmit(configure);
    };
    passwordDialog?.querySelector("[data-setup-password-continue]")?.addEventListener("click", continueWithPassword);
    password?.addEventListener("keydown", (event) => {
      if (event.key !== "Enter" || event.isComposing) return;
      event.preventDefault();
      continueWithPassword();
    });
    eulaDialog?.querySelectorAll("[data-setup-eula-cancel], [data-setup-eula-decline]").forEach((button) => {
      button.addEventListener("click", () => eulaDialog.close());
    });
    eulaDialog?.addEventListener("close", () => {
      if (!submitting) {
        passwordConfirmed = false;
        if (password) password.value = "";
      }
    });
    applyForm.addEventListener("submit", async (event) => {
      if (applyForm.dataset.smbRequired === "true" && !passwordConfirmed) {
        event.preventDefault();
        if (passwordDialog && !passwordDialog.open) passwordDialog.showModal();
        return;
      }
      if (applyForm.dataset.eulaAccepted !== "true" && event.submitter?.dataset.setupEulaAccept === undefined) {
        event.preventDefault();
        if (eulaDialog && !eulaDialog.open) eulaDialog.showModal();
        return;
      }
      event.preventDefault();
      if (submitting) return;
      // Keep the exact reviewed submission, including credentials, only in memory.
      const body = new URLSearchParams();
      new FormData(applyForm).forEach((value, key) => body.append(key, String(value)));
      if (event.submitter?.name) body.set(event.submitter.name, event.submitter.value);
      submitting = true;
      eulaDialog?.close();
      if (configure) {
        configure.disabled = true;
        configure.textContent = addingServer ? jvT("ui.583ba8d35f24") : jvT("ui.696abab0be63");
      }
      let submissionMessage = jvT("ui.ee4e2378bd55");
      try {
        const submit = () => fetch(applyForm.action, {
          method: "POST", credentials: "same-origin", cache: "no-store",
          headers: {Accept: "text/html, application/json", "Content-Type": "application/x-www-form-urlencoded"},
          body: body.toString(),
        });
        const response = addingServer ? await submit() : await resumeReviewedIdentitySubmission(submit, body.get("csrf") || "");
        if (response.redirected) {
          window.location.assign(response.url);
          return;
        }
        const markup = await response.text();
        const page = new DOMParser().parseFromString(markup, "text/html");
        const message = page.querySelector(".notice.error")?.textContent.trim();
        submissionMessage = message || submissionMessage;
        throw new Error(submissionMessage);
      } catch (error) {
        submitting = false;
        if (configure) {
          configure.disabled = false;
          configure.textContent = addingServer ? jvT("ui.7ac52b00d920") : jvT("ui.fef1e933ec92");
        }
        const feedback = document.createElement("p");
        feedback.setAttribute("role", "alert");
        feedback.textContent = submissionMessage;
        const review = document.createElement("a");
        review.href = addingServer ? "/minecraft/servers/add/review" : "/setup/review";
        review.textContent = jvT("ui.72ff222f7f89");
        applyForm.append(feedback, review);
      } finally {
        body.delete("smb_password");
      }
    });
  }

  const panel = document.querySelector("[data-setup-operation]");
  if (!panel) return;

  const statusURL = panel.dataset.statusUrl;
  const statusText = document.getElementById("setup-operation-status");
  const stateBadge = document.getElementById("setup-operation-state");
  const stageText = document.getElementById("setup-operation-stage");
  const reconnectNote = document.getElementById("setup-reconnect-note");
  const successNote = document.getElementById("setup-operation-success");
  const rollbackNote = document.getElementById("setup-operation-rollback");
  const attentionNote = document.getElementById("setup-operation-attention");
  const dashboardLink = document.getElementById("setup-dashboard-link");
  const reviewLink = document.getElementById("setup-review-link");
  const startOverForm = document.getElementById("setup-start-over-form");
  const recoverForm = document.getElementById("setup-recover-form");
  const disabledStartOver = document.getElementById("setup-start-over-disabled");
  const diagnosticLogLink = document.getElementById("setup-diagnostic-log-link");
  const diagnosticLogView = document.getElementById("setup-diagnostic-log-view");
  const stages = Array.from(panel.querySelectorAll("[data-setup-stage]"));
  const verifyNote = document.getElementById("setup-minecraft-verify-note");
  const elapsed = document.getElementById("setup-operation-elapsed");

  let consoleState = panel.dataset.initialState;
  let consoleStage = panel.dataset.initialStage;
  diagnosticLogView?.addEventListener("click", () => window.JustVoxelLiveConsole.open({
    kind: "setup", operationID: panel.dataset.operationId, state: consoleState, stage: consoleStage,
  }));
  dashboardLink?.addEventListener("click", (event) => {
    window.JustVoxelLiveConsole.close();
    if (window.name === "justvoxel-add-server" && consoleState === "succeeded") {
      const opener = window.opener;
      try {
        if (!opener || opener.closed || opener.location.origin !== window.location.origin) return;
      } catch {
        return; // A cross-origin opener leaves the Dashboard link in control.
      }
      event.preventDefault();
      try {
        opener.location.assign("/?workspace=minecraft-servers");
        opener.focus();
        window.close();
      } catch {
        window.location.assign(dashboardLink.href);
      }
    }
  });

  let failures = 0;
  let finished = false;
  let elapsedInterval = null;
  let startedAt = NaN;
  let finalElapsed = null;

  const stateLabel = (state) => {
    if (state === "queued") return jvT("ui.661ff40a07e0");
    if (state === "validating") return jvT("ui.5a1a167e3d90");
    if (state === "running") return jvT("ui.f852556e2329");
    if (state === "verifying") return jvT("ui.a7d4a9f301ed");
    if (state === "failed") return jvT("ui.d90064dbdb11");
    if (state === "rolling_back") return jvT("ui.8b9e964b606b");
    if (state === "rolled_back") return jvT("ui.71592236d852");
    if (state === "needs_attention") return jvT("ui.c1ebc7817870");
    if (state === "succeeded") return jvT("ui.143b270a3260");
    return state || jvT("ui.a92f0449a9f7");
  };

  const stageLabel = (stage) => {
    if (stage === "queued") return jvT("ui.f5e5f81fd256");
    if (stage === "storage_preflight") return jvT("ui.b5deea41f994");
    if (stage === "storage_snapshot") return jvT("ui.a4177ad95738");
    if (stage === "storage_verified") return jvT("ui.53e417e293d7");
    if (stage === "runtime_preflight") return jvT("ui.87e72934b376");
    if (stage === "runtime_config") return jvT("ui.d01ed84b6ec2");
    if (stage === "minecraft_verify") return jvT("ui.eb7466fbf37e");
    if (stage === "final_validation") return jvT("ui.3fa519aa4bf7");
    if (stage === "completed") return jvT("ui.143b270a3260");
    if (stage === "setup_failed") return jvT("ui.b62ffd61bf8c");
    if (stage === "runtime_rollback") return jvT("ui.d353e569c4b7");
    if (stage === "storage_rollback") return jvT("ui.ae7cbca04533");
    if (stage === "runtime_recovery") return jvT("ui.20da9d1e8eb0");
    if (stage === "storage_recovery") return jvT("ui.c79560178cdb");
    if (stage === "setup_rolled_back") return jvT("ui.71592236d852");
    if (stage === "interrupted") return jvT("ui.132d124d6bb3");
    return jvT("ui.a92f0449a9f7");
  };

  const terminalState = (state) => ["succeeded", "rolled_back", "needs_attention"].includes(state);

  const visibleStage = (stage) => {
    if (["storage_preflight", "storage_snapshot", "storage_verified"].includes(stage)) return 0;
    if (["runtime_preflight", "runtime_config"].includes(stage)) return 1;
    if (stage === "minecraft_verify") return 2;
    if (stage === "final_validation") return 3;
    if (stage === "completed") return 4;
    return -1;
  };

  const renderStages = (operation) => {
    const succeeded = operation.state === "succeeded";
    const recovering = ["failed", "rolling_back", "rolled_back", "needs_attention"].includes(operation.state);
    const current = visibleStage(operation.stage);
    stages.forEach((item, index) => {
      const complete = succeeded || (!recovering && current >= 0 && index < current);
      const active = !succeeded && !recovering && index === current;
      item.classList.toggle("is-complete", complete);
      item.classList.toggle("is-current", active);
      item.classList.toggle("is-pending", !complete && !active && !recovering);
      item.classList.toggle("is-rolled-back", operation.state === "rolled_back");
      if (active) item.setAttribute("aria-current", "step");
      else item.removeAttribute("aria-current");
    });
    if (verifyNote) verifyNote.hidden = operation.stage !== "minecraft_verify" || recovering || succeeded;
  };

  const renderElapsed = () => {
    if (!elapsed) return;
    if (!Number.isFinite(startedAt)) {
      elapsed.hidden = true;
      return;
    }
    const seconds = finalElapsed ?? Math.max(0, Math.floor((Date.now() - startedAt) / 1000));
    const minutes = Math.floor(seconds / 60);
    const remainder = seconds % 60;
    const formatted = minutes === 0 ? `${seconds} sec` : remainder === 0 ? `${minutes} min` : jvT("ui.b58375451be0", {v0: (minutes), v1: (String(remainder).padStart(2, "0"))});
    elapsed.textContent = jvT("ui.663e41d8f18f", {v0: (formatted)});
    elapsed.hidden = false;
  };

  const render = (operation) => {
    if (!operation) return;
    consoleState = operation.state;
    consoleStage = operation.stage;
    window.JustVoxelLiveConsole.updateState(panel.dataset.operationId, operation.state, operation.stage);
    failures = 0;
    if (reconnectNote) reconnectNote.hidden = true;
    if (statusText) statusText.textContent = operation.status || jvT("ui.654b6d52bf7a");
    if (stateBadge) stateBadge.textContent = stateLabel(operation.state);
    if (stageText) stageText.textContent = operation.state === "rolled_back" ? jvT("ui.71592236d852") : stageLabel(operation.stage);
    renderStages(operation);
    const parsedStart = Date.parse(operation.started_at || "");
    if (Number.isFinite(parsedStart)) startedAt = parsedStart;
    if (terminalState(operation.state)) {
      const finishedAt = Date.parse(operation.finished_at || "");
      finalElapsed = Number.isFinite(startedAt) ? Math.max(0, Math.floor(((Number.isFinite(finishedAt) ? finishedAt : Date.now()) - startedAt) / 1000)) : null;
      if (elapsedInterval !== null) { window.clearInterval(elapsedInterval); elapsedInterval = null; }
    } else if (elapsedInterval === null) {
      elapsedInterval = window.setInterval(renderElapsed, 1000);
    }
    renderElapsed();

    const succeeded = operation.state === "succeeded";
    const rolledBack = operation.state === "rolled_back";
    const needsAttention = operation.state === "needs_attention";
    if (successNote) successNote.hidden = !succeeded;
    if (rollbackNote) rollbackNote.hidden = !rolledBack;
    if (attentionNote) attentionNote.hidden = !needsAttention;
    if (dashboardLink) dashboardLink.hidden = !terminalState(operation.state);
    if (reviewLink) reviewLink.hidden = !rolledBack;
    if (startOverForm) startOverForm.hidden = !rolledBack;
    if (recoverForm) recoverForm.hidden = !(needsAttention && ["storage_rollback", "runtime_rollback"].includes(operation.stage));
    if (disabledStartOver) disabledStartOver.hidden = !needsAttention;
    if (diagnosticLogLink) diagnosticLogLink.hidden = !terminalState(operation.state);

    if (terminalState(operation.state)) {
      finished = true;
      panel.classList.add("is-finished");
      panel.classList.toggle("is-rolled-back", rolledBack);
    }
  };

  render({
    state: panel.dataset.initialState,
    stage: panel.dataset.initialStage,
    started_at: panel.dataset.startedAt,
    finished_at: panel.dataset.finishedAt,
    status: statusText?.textContent,
  });

  const poll = async () => {
    if (finished) return;
    try {
      const response = await fetch(statusURL, {
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
      if (!response.ok) throw new Error(jvT("ui.180119be630b"));
      const payload = await response.json();
      render(payload.operation);
    } catch (_) {
      failures += 1;
      if (failures >= 2 && reconnectNote) reconnectNote.hidden = false;
    }
    if (!finished) window.setTimeout(poll, 2500);
  };

  poll();
});
