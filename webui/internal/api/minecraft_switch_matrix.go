package api

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

const switchMatrixPath = "/v1/minecraft/switch-matrix"

type SwitchMatrixStatus struct {
	Enabled   bool                   `json:"enabled"`
	Operation *SwitchMatrixOperation `json:"operation"`
}

type SwitchMatrixOperation struct {
	Targeted          bool                  `json:"targeted,omitempty"`
	OperationID       string                `json:"operation_id"`
	State             string                `json:"state"`
	ReservationActive bool                  `json:"reservation_active"`
	RecoveryRequired  bool                  `json:"recovery_required"`
	Progress          *SwitchMatrixProgress `json:"progress,omitempty"`
	DiagnosticID      string                `json:"diagnostic_id"`
}

type SwitchMatrixProgress struct {
	ManifestReady   bool   `json:"manifest_ready"`
	Completed       int    `json:"completed"`
	Total           int    `json:"total"`
	Passed          int    `json:"passed"`
	Failed          int    `json:"failed"`
	Unsupported     int    `json:"unsupported"`
	Blocked         int    `json:"blocked"`
	Unfinished      int    `json:"unfinished"`
	CancelRequested bool   `json:"cancel_requested"`
	Source          string `json:"source,omitempty"`
	Target          string `json:"target,omitempty"`
	Action          string `json:"action,omitempty"`
}

func (c *Client) SwitchMatrixStatus(ctx context.Context, session, id string) (SwitchMatrixStatus, error) {
	var out SwitchMatrixStatus
	path := switchMatrixPath
	if id != "" {
		path += "?operation_id=" + url.QueryEscape(id)
	}
	err := c.do(ctx, http.MethodGet, path, session, nil, &out)
	return out, err
}

func (c *Client) StartSwitchMatrix(ctx context.Context, session string) (SwitchMatrixStatus, error) {
	var out SwitchMatrixStatus
	err := c.do(ctx, http.MethodPost, switchMatrixPath+"/start", session, struct{}{}, &out)
	return out, err
}

func (c *Client) StartTargetedSwitchMatrix(ctx context.Context, session string, ids []string) (SwitchMatrixStatus, error) {
	var out SwitchMatrixStatus
	err := c.do(ctx, http.MethodPost, switchMatrixPath+"/start", session, map[string][]string{"case_ids": ids}, &out)
	return out, err
}

func (c *Client) StopSwitchMatrix(ctx context.Context, session, id string) (SwitchMatrixStatus, error) {
	var out SwitchMatrixStatus
	err := c.do(ctx, http.MethodPost, switchMatrixPath+"/stop", session, map[string]string{"operation_id": id}, &out)
	return out, err
}

func (c *Client) RecoverSwitchMatrix(ctx context.Context, session, id string) (SwitchMatrixStatus, error) {
	var out SwitchMatrixStatus
	client := *c.http
	client.Timeout = 36 * time.Minute
	recovery := Client{http: &client}
	err := recovery.do(ctx, http.MethodPost, switchMatrixPath+"/recover", session, map[string]string{"operation_id": id}, &out)
	return out, err
}
