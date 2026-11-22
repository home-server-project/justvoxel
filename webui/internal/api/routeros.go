package api

import (
	"context"
	"net/http"
	"time"
)

const routerOSPath = "/v1/admin/network/router/mikrotik"

type RouterOSSelector struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

type RouterOSCredentials struct {
	Endpoint string `json:"endpoint"`
	Username string `json:"username"`
	Password string `json:"password"`
	CAPEM    string `json:"ca_pem,omitempty"`
}

type RouterOSConfiguration struct {
	RouterOSCredentials
	Target  string           `json:"target"`
	Ingress RouterOSSelector `json:"ingress"`
}

type RouterMinecraftEndpoint struct {
	ID        string   `json:"id,omitempty"`
	Name      string   `json:"name,omitempty"`
	Network   bool     `json:"network,omitempty"`
	Edition   string   `json:"edition"`
	Protocols []string `json:"protocols"`
	Port      int      `json:"port"`
	LastPort  int      `json:"last_port"`
	State     string   `json:"state"`
}

type RouterOSCAMetadata struct {
	CommonName  string    `json:"common_name,omitempty"`
	Fingerprint string    `json:"fingerprint"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type RouterOSStatus struct {
	Configured         bool                      `json:"configured"`
	CanConfigure       bool                      `json:"can_configure,omitempty"`
	Status             string                    `json:"status"`
	Diagnostic         string                    `json:"diagnostic,omitempty"`
	TrustedCA          *RouterOSCAMetadata       `json:"trusted_ca,omitempty"`
	Identity           string                    `json:"identity,omitempty"`
	Version            string                    `json:"version,omitempty"`
	Board              string                    `json:"board,omitempty"`
	Endpoint           string                    `json:"endpoint,omitempty"`
	Username           string                    `json:"username,omitempty"`
	Target             string                    `json:"target,omitempty"`
	Ingress            *RouterOSSelector         `json:"ingress,omitempty"`
	RecommendedTarget  string                    `json:"recommended_target,omitempty"`
	RecommendedIngress *RouterOSSelector         `json:"recommended_ingress,omitempty"`
	IngressChoices     []RouterOSSelector        `json:"ingress_choices,omitempty"`
	TargetChoices      []string                  `json:"target_choices,omitempty"`
	Endpoints          []RouterMinecraftEndpoint `json:"endpoints,omitempty"`
	InventoryError     bool                      `json:"inventory_error,omitempty"`
	ServerName         string                    `json:"server_name,omitempty"`
	NetworkMember      bool                      `json:"network_member,omitempty"`
}

func (c *Client) RouterOSStatus(ctx context.Context, session, id string) (RouterOSStatus, error) {
	path := routerOSPath
	if id != "" {
		if !minecraftSelectedIDPattern.MatchString(id) {
			return RouterOSStatus{}, &ResponseError{StatusCode: http.StatusBadRequest, Message: "invalid_request"}
		}
		path += "/instances/" + id
	}
	var out RouterOSStatus
	err := c.do(ctx, http.MethodGet, path, session, nil, &out)
	return out, err
}

func (c *Client) RouterOSTest(ctx context.Context, session string, input RouterOSCredentials) (RouterOSStatus, error) {
	var out RouterOSStatus
	err := c.do(ctx, http.MethodPost, routerOSPath+"/test", session, input, &out)
	return out, err
}

func (c *Client) RouterOSSave(ctx context.Context, session string, input RouterOSConfiguration) error {
	return c.do(ctx, http.MethodPut, routerOSPath, session, input, nil)
}

func (c *Client) RouterOSForget(ctx context.Context, session string) error {
	return c.do(ctx, http.MethodDelete, routerOSPath, session, nil, nil)
}

func (c *Client) RouterOSPreviewCA(ctx context.Context, session, caPEM string) (RouterOSCAMetadata, error) {
	var out RouterOSCAMetadata
	err := c.do(ctx, http.MethodPost, routerOSPath+"/ca", session, struct {
		CAPEM string `json:"ca_pem"`
	}{caPEM}, &out)
	return out, err
}

func (c *Client) RouterOSReplaceCA(ctx context.Context, session, caPEM string) error {
	return c.do(ctx, http.MethodPut, routerOSPath+"/ca", session, struct {
		CAPEM string `json:"ca_pem"`
	}{CAPEM: caPEM}, nil)
}

func (c *Client) RouterOSRemoveCA(ctx context.Context, session string) error {
	return c.do(ctx, http.MethodDelete, routerOSPath+"/ca", session, nil, nil)
}
