package api

import (
	"context"
	"net/http"
)

type OnboardBattery struct {
	Capacity *int   `json:"capacity,omitempty"`
	Status   string `json:"status,omitempty"`
}
type SystemHardwareStatus struct {
	HWS              bool             `json:"hws"`
	DisplayAvailable bool             `json:"display_available"`
	LidAvailable     bool             `json:"lid_available"`
	LidIgnored       bool             `json:"lid_ignored"`
	Batteries        []OnboardBattery `json:"batteries"`
}
type SystemHardwareRequest struct {
	Action       string `json:"action"`
	DelaySeconds int    `json:"delay_seconds"`
}

func (c *Client) SystemHardwareStatus(ctx context.Context, session string) (SystemHardwareStatus, error) {
	var out SystemHardwareStatus
	err := c.do(ctx, http.MethodGet, "/v1/system/hardware", session, nil, &out)
	return out, err
}
func (c *Client) AdminSystemHardwareAction(ctx context.Context, session string, request SystemHardwareRequest) (SystemHardwareStatus, error) {
	var out SystemHardwareStatus
	err := c.do(ctx, http.MethodPost, "/v1/admin/system/display", session, request, &out)
	return out, err
}
