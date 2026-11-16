function initServerMigrationOperation(root = document) {
  const panel = root.querySelector("[data-server-migration-operation]");
  if (!panel || panel.dataset.serverMigrationOperationInitialized === "true") return;
  panel.dataset.serverMigrationOperationInitialized = "true";

  const statusURL = panel.dataset.statusUrl;
  const statusText = root.querySelector("#server-migration-operation-status");
  const stateBadge = root.querySelector("#server-migration-operation-state");
  const stageText = root.querySelector("#server-migration-operation-stage");
  const nameText = root.querySelector("#server-migration-operation-name");
  const reconnectNote = root.querySelector("#server-migration-reconnect-note");
  const successNote = root.querySelector("#server-migration-operation-success");
  const rollbackNote = root.querySelector("#server-migration-operation-rollback");
  const attentionNote = root.querySelector("#server-migration-operation-attention");
  const backLink = root.querySelector("#server-migration-back-link");
  const dashboardLink = root.querySelector("#server-migration-dashboard-link");
  const recoveryLink = root.querySelector("#server-migration-recovery-link");
  const exportLogDetails = root.querySelector("#server-migration-export-log-details");
  let failures = 0;
  let finished = false;

  const operationName = (type) => {
    if (type === "migration_export") return jvT("ui.d1f081749243");
    if (type === "migration_import") return jvT("ui.ffe624a69618");
    if (type === "migration_recovery") return jvT("ui.504ca73ba883");
    return jvT("ui.0f8e0d8c6039");
  };

  const stateLabel = (state) => {
    if (state === "queued") return jvT("ui.661ff40a07e0");
    if (state === "validating") return jvT("ui.5a1a167e3d90");
    if (state === "running") return jvT("ui.f4ccae29e1bb");
    if (state === "verifying") return jvT("ui.a7d4a9f301ed");
    if (state === "failed") return jvT("ui.d90064dbdb11");
    if (state === "rolling_back") return jvT("ui.8b9e964b606b");
    if (state === "succeeded") return jvT("ui.6d9a6f97a5fd");
    if (state === "rolled_back") return jvT("ui.71592236d852");
    if (state === "needs_attention") return jvT("ui.c1ebc7817870");
    if (state === "resolved") return jvT("ui.5be3c2c8354e");
    return jvT("ui.a92f0449a9f7");
  };

  const friendlyToken = (value) => {
    if (!value) return jvT("ui.a92f0449a9f7");
    const text = value.replaceAll("_", " ");
    return text.charAt(0).toUpperCase() + text.slice(1);
  };

  const stageLabel = (type, stage) => {
    if (stage === "queued") return jvT("ui.f5e5f81fd256");
    if (stage === "completed") return jvT("ui.143b270a3260");
    if (stage === "player_recheck") return jvT("ui.ab2d5bb1e79d");
    if (stage === "minecraft_stop") return jvT("ui.995beb2af122");
    if (["invalid_execution_plan", "interrupted"].includes(stage)) return jvT("ui.1e1b96ec8fd9");
    if (type === "migration_export") {
      if (stage === "export_preflight") return jvT("ui.60117bda3d60");
      if (stage === "archive_integrity" || stage === "integrity_verification") return jvT("ui.78c93c0ea236");
      if (stage === "export_failed") return jvT("ui.a689980fba1b");
      if (stage === "rollback") return jvT("ui.d0d1b11758c9");
      if (stage === "export_rolled_back") return jvT("ui.39aec9840b2d");
      if (["export_needs_attention", "export_backend_interrupted", "export_backend_incomplete"].includes(stage)) return jvT("ui.1e1b96ec8fd9");
    }
    if (type === "migration_import") {
      if (stage === "recovery_handoff") return jvT("ui.8c5a07566067");
      if (stage === "import_preflight") return jvT("ui.1db8eecfbe9b");
      if (stage === "import_storage") return jvT("ui.1b553da67100");
      if (stage === "import_execute") return jvT("ui.44c03884fea1");
      if (stage === "import_verify") return jvT("ui.70c4b99c7c87");
      if (stage === "import_failed") return jvT("ui.44d2868f2a28");
      if (stage === "rollback") return jvT("ui.f6e0f42acf8e");
      if (stage === "import_rolled_back") return jvT("ui.3d9d723dfd19");
      if (["import_needs_attention", "import_backend_interrupted", "import_backend_incomplete"].includes(stage)) return jvT("ui.1e1b96ec8fd9");
    }
    if (type === "migration_recovery") {
      if (stage === "recovery_preflight") return jvT("ui.8e649c951902");
      if (stage === "recovery_finalize") return jvT("ui.665aaa8d5d59");
      if (stage === "recovery_verify") return jvT("ui.0cc638ce0ff6");
      if (["recovery_backend_failed", "recovery_backend_invalid", "recovery_backend_incomplete"].includes(stage)) return jvT("ui.1e1b96ec8fd9");
    }
    return friendlyToken(stage);
  };

  const supportedType = (type) => ["migration_export", "migration_import", "migration_recovery"].includes(type);
  const terminalState = (state) => ["succeeded", "rolled_back", "needs_attention", "resolved"].includes(state);

  const render = (operation) => {
    if (!operation || !supportedType(operation.operation_type)) return;
    failures = 0;
    if (reconnectNote) reconnectNote.hidden = true;
    if (nameText) nameText.textContent = operationName(operation.operation_type);
    if (statusText) statusText.textContent = operation.status || jvT("ui.1d593de3d211");
    if (stateBadge) stateBadge.textContent = stateLabel(operation.state);
    if (stageText) stageText.textContent = stageLabel(operation.operation_type, operation.stage);

    const succeeded = operation.state === "succeeded";
    const rolledBack = operation.state === "rolled_back";
    const needsAttention = operation.state === "needs_attention";
    const resolved = operation.state === "resolved";
    if (successNote) successNote.hidden = !succeeded;
    if (rollbackNote) rollbackNote.hidden = !rolledBack;
    if (attentionNote) attentionNote.hidden = !needsAttention;
    if (dashboardLink) {
      dashboardLink.hidden = !succeeded;
      if (operation.operation_type === "migration_export") {
        dashboardLink.textContent = jvT("ui.11a6767d5674");
        dashboardLink.href = "/?workspace=migration&tab=export&server=" + encodeURIComponent(panel.closest("[data-server-id]").dataset.serverId);
      }
    }
    if (exportLogDetails) exportLogDetails.hidden = !(operation.operation_type === "migration_export" && ["succeeded", "rolled_back", "failed", "needs_attention"].includes(operation.state));
    if (backLink) backLink.hidden = !(succeeded || rolledBack || resolved);
    if (recoveryLink) recoveryLink.hidden = !(needsAttention && ["migration_import", "migration_export", "migration_recovery"].includes(operation.operation_type));
    if (terminalState(operation.state)) finished = true;
  };

  const poll = async () => {
    if (finished || !panel.isConnected) return;
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
    if (!finished && panel.isConnected) window.setTimeout(poll, 2500);
  };

  poll();
}

window.JustVoxelServerMigrationOperation = { init: initServerMigrationOperation };
