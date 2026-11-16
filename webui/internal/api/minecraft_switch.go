package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"
)

type SwitchContent struct {
	Name          string `json:"name"`
	Filename      string `json:"filename"`
	Compatibility string `json:"compatibility"`
	Reason        string `json:"reason"`
}
type MinecraftSwitchPlan struct {
	InstanceID  string          `json:"instance_id"`
	Current     string          `json:"current"`
	Target      string          `json:"target"`
	Version     string          `json:"version"`
	Convert     bool            `json:"convert"`
	Clone       bool            `json:"clone"`
	Reason      string          `json:"reason"`
	Content     []SwitchContent `json:"content"`
	Warnings    []string        `json:"warnings"`
	Fingerprint string          `json:"plan_fingerprint"`
}
type MinecraftSwitchRequest struct {
	Target      string `json:"target"`
	Action      string `json:"action"`
	Fingerprint string `json:"plan_fingerprint"`
	Confirm     bool   `json:"confirm"`
	OperationID string `json:"operation_id,omitempty"`
}
type MinecraftSwitchOperation struct {
	PersistentOperationResponse
	CloneID string `json:"clone_id,omitempty"`
}

func (c *Client) minecraftSwitch(ctx context.Context, session, id, action string, body any, out any) error {
	path := "http://unix/v1/minecraft/instances/" + url.PathEscape(id) + "/switch/" + action
	method := http.MethodGet
	var data []byte
	if body != nil {
		var err error
		data, err = json.Marshal(body)
		if err != nil {
			return err
		}
		method = http.MethodPost
	}
	req, err := http.NewRequestWithContext(ctx, method, path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+session)
	req.Header.Set("Content-Type", "application/json")
	client := *c.http
	client.Timeout = 95 * time.Second
	if action == "recover" {
		client.Timeout = 31 * time.Minute
	}
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusUnauthorized {
		return ErrUnauthorized
	}
	if response.StatusCode == http.StatusForbidden {
		return ErrPasswordChangeRequired
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return readResponseError(response)
	}
	return json.NewDecoder(response.Body).Decode(out)
}
func (c *Client) MinecraftInstanceSwitchPlan(ctx context.Context, session, id, target string) (MinecraftSwitchPlan, error) {
	var out MinecraftSwitchPlan
	err := c.minecraftSwitch(ctx, session, id, "plan?target="+url.QueryEscape(target), nil, &out)
	return out, err
}
func (c *Client) MinecraftInstanceSwitchApply(ctx context.Context, session, id string, request MinecraftSwitchRequest) (MinecraftSwitchOperation, error) {
	var out MinecraftSwitchOperation
	err := c.minecraftSwitch(ctx, session, id, "apply", request, &out)
	return out, err
}
func (c *Client) MinecraftInstanceSwitchOperation(ctx context.Context, session, id, operation string) (MinecraftSwitchOperation, error) {
	var out MinecraftSwitchOperation
	err := c.minecraftSwitch(ctx, session, id, "operation?operation_id="+url.QueryEscape(operation), nil, &out)
	return out, err
}
func (c *Client) MinecraftInstanceSwitchRecover(ctx context.Context, session, id, operation string) (MinecraftSwitchOperation, error) {
	var out MinecraftSwitchOperation
	err := c.minecraftSwitch(ctx, session, id, "recover", map[string]any{"operation_id": operation, "confirm": true}, &out)
	return out, err
}
