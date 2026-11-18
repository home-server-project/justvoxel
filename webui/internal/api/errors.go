package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type ResponseError struct {
	StatusCode     int
	Message        string
	DiagnosticID   string
	PluginRecovery *PluginRuntimeRecovery
}

func (e *ResponseError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("management API HTTP %d", e.StatusCode)
}

func ErrorMessage(err error) (string, bool) {
	var responseErr *ResponseError
	if !errors.As(err, &responseErr) || strings.TrimSpace(responseErr.Message) == "" {
		return "", false
	}
	return responseErr.Message, true
}

func readResponseError(resp *http.Response) error {
	const errorLimit = 4096
	limit := int64(errorLimit)
	if resp.StatusCode == http.StatusConflict {
		// A player confirmation can contain the bounded RCON list with JSON escaping.
		limit = 64 << 10
	}
	limited, _ := io.ReadAll(io.LimitReader(resp.Body, limit))
	if resp.StatusCode == http.StatusConflict {
		var recovery PluginRuntimeRecovery
		if json.Unmarshal(limited, &recovery) == nil && recovery.RecoveryRequired && recovery.RecoveryKind == "plugin_install" &&
			minecraftSelectedIDPattern.MatchString(recovery.ID) && persistentOperationIDPattern.MatchString(recovery.RecoveryOperationID) &&
			(recovery.RecoveryAction == "start" || recovery.RecoveryAction == "stop") && strings.TrimSpace(recovery.Message) != "" &&
			(!recovery.ConfirmationRequired || (recovery.RecoveryAction == "stop" && recovery.Online > 0)) {
			return &ResponseError{StatusCode: resp.StatusCode, Message: recovery.Message, PluginRecovery: &recovery}
		}
	}
	var payload struct {
		Error        string `json:"error"`
		DiagnosticID string `json:"diagnostic_id"`
	}
	if json.Unmarshal(limited[:min(len(limited), errorLimit)], &payload) == nil && strings.TrimSpace(payload.Error) != "" {
		id := ""
		if persistentOperationIDPattern.MatchString(payload.DiagnosticID) {
			id = payload.DiagnosticID
		}
		return &ResponseError{StatusCode: resp.StatusCode, Message: payload.Error, DiagnosticID: id}
	}
	return &ResponseError{StatusCode: resp.StatusCode}
}
