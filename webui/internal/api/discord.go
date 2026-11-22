package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

func (c *Client) DiscordAdmin(ctx context.Context, session string, body json.RawMessage) (json.RawMessage, error) {
	var out json.RawMessage
	method := http.MethodGet
	if len(body) > 0 {
		method = http.MethodPost
	}
	caller := *c
	client := *c.http
	client.Timeout = 45 * time.Second
	caller.http = &client
	err := caller.do(ctx, method, "/v1/admin/discord", session, body, &out)
	return out, err
}
