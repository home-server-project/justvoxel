package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Only public presentation fields leave the WebUI. Journal evidence and backend
// diagnostic strings are deliberately not part of this contract.
type selectedRestoreProgress struct {
	ID               string `json:"id"`
	State            string `json:"state"`
	Stage            string `json:"stage"`
	RecoveryRequired bool   `json:"recovery_required"`
	StateLabel       string `json:"state_label"`
	StageLabel       string `json:"stage_label"`
	Status           string `json:"status"`
	Complete         bool   `json:"complete"`
	RolledBack       bool   `json:"rolled_back"`
	Attention        bool   `json:"attention"`
	Terminal         bool   `json:"terminal"`
}

func (p selectedRestoreProgress) Busy() bool {
	return p.State != "idle" && !p.Complete && !p.RolledBack
}

func restoreProgressLabel(state string) string {
	switch state {
	case "idle":
		return uiText("ui.5fa7aac5375c")
	case "accepted", "validating":
		return "Validating"
	case "staging", "ready_to_apply":
		return uiText("ui.1c84ade9fb65")
	case "shutdown":
		return uiText("ui.995beb2af122")
	case "preserving_original", "replacement_activated", "original_preserved":
		return uiText("ui.6cdbd4f77314")
	case "runtime_verification":
		return uiText("ui.724382940a57")
	case "rollback_required", "recovering":
		return uiText("ui.b62ffd61bf8c")
	case "original_restored":
		return uiText("ui.109857563af5")
	case "rollback_verification":
		return uiText("ui.8b9e964b606b")
	case "rolled_back":
		return uiText("ui.71592236d852")
	case "completed":
		return "Complete"
	case "recovery_required", "interrupted", "failed":
		return uiText("ui.c1ebc7817870")
	default:
		return uiText("ui.f9daad8f5a86")
	}
}

func presentSelectedRestore(p selectedRestoreProgress) selectedRestoreProgress {
	p.StateLabel = restoreProgressLabel(p.State)
	p.StageLabel = restoreProgressLabel(p.Stage)
	if p.Stage == "" {
		p.StageLabel = p.StateLabel
	}
	p.Complete = p.State == "completed" && !p.RecoveryRequired
	p.RolledBack = p.State == "rolled_back" && !p.RecoveryRequired
	p.Attention = p.RecoveryRequired || p.State == "interrupted" || p.State == "failed"
	p.Terminal = p.Complete || p.RolledBack || p.Attention || p.State == "idle"
	p.Status = uiText("ui.6d934ee1803d")
	// Active Recovery can still have a recovery_required journal stage while
	// the Agent explicitly reports recovery_required=false. Keep polling.
	if p.State == "recovery_required" && !p.RecoveryRequired {
		p.StateLabel = uiText("ui.b62ffd61bf8c")
		if p.Stage == "recovery_required" || p.Stage == "" {
			p.StageLabel = uiText("ui.b62ffd61bf8c")
		}
	}
	switch {
	case p.Attention:
		p.StateLabel = uiText("ui.c1ebc7817870")
		p.Status = uiText("ui.f4231c2da286")
	case p.Complete:
		p.Status = uiText("ui.0c192101a9ac")
	case p.RolledBack:
		p.Status = uiText("ui.a5cc653350d4")
	case p.State == "idle":
		p.Status = uiText("ui.e720806d820c")
	}
	return p
}

func readSelectedRestore(ctx context.Context, client selectedBackupsAPI, session, id string) (selectedRestoreProgress, error) {
	raw, err := client.MinecraftInstanceBackups(ctx, session, id, "restore/status", nil)
	if err != nil {
		return selectedRestoreProgress{}, err
	}
	var p selectedRestoreProgress
	// Decode only authoritative backend fields, never backend-supplied labels.
	var status struct {
		ID               string `json:"id"`
		State            string `json:"state"`
		Stage            string `json:"stage"`
		RecoveryRequired bool   `json:"recovery_required"`
	}
	if json.Unmarshal(raw, &status) != nil || status.ID != id {
		return p, fmt.Errorf("%s", uiText("ui.0dc768ae2272"))
	}
	p.ID, p.State, p.Stage, p.RecoveryRequired = status.ID, status.State, status.Stage, status.RecoveryRequired
	return presentSelectedRestore(p), nil
}

func (a *App) selectedRestoreStatus(w http.ResponseWriter, r *http.Request) {
	session, _, ok := a.minecraftWorkspaceIdentity(w, r, "administrator")
	if !ok {
		return
	}
	selected, ok := a.selectedMinecraft(w, r, session)
	if !ok {
		return
	}
	client, ok := a.api.(selectedBackupsAPI)
	if !ok {
		writeMinecraftWorkspaceError(w, 503, ui(w, "ui.367bc30f5104"))
		return
	}
	progress, err := readSelectedRestore(r.Context(), client, session, selected.ID)
	if err != nil {
		a.writeMinecraftWorkspaceAPIError(w, err, ui(w, "ui.afadfb0b5986"))
		return
	}
	writeMinecraftWorkspaceJSON(w, 200, progress)
}
