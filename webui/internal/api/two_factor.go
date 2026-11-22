package api

import (
	"context"
	"net/http"
)

type TwoFactorStatus struct {
	TwoFactorEnabled bool   `json:"two_factor_enabled"`
	TwoFactorSource  string `json:"two_factor_source,omitempty"`
}

type TwoFactorRequest struct {
	UserID     *int64 `json:"user_id,omitempty"`
	Enrollment string `json:"enrollment,omitempty"`
	Password   string `json:"password,omitempty"`
	Code       string `json:"code,omitempty"`
	Recovery   bool   `json:"recovery,omitempty"`
}

type TwoFactorResponse struct {
	OK            bool     `json:"ok,omitempty"`
	Enrollment    string   `json:"enrollment,omitempty"`
	SetupKey      string   `json:"setup_key,omitempty"`
	QRPNG         []byte   `json:"qr_png,omitempty"`
	RecoveryCodes []string `json:"recovery_codes,omitempty"`
}

func (c *Client) VerifyTwoFactor(ctx context.Context, challenge, code string, recovery bool) (LoginResponse, error) {
	var result LoginResponse
	err := c.do(ctx, http.MethodPost, "/v1/auth/login/two-factor", challenge, TwoFactorRequest{Code: code, Recovery: recovery}, &result)
	return result, err
}

func (c *Client) TwoFactor(ctx context.Context, session, action string, request TwoFactorRequest) (TwoFactorResponse, error) {
	var result TwoFactorResponse
	err := c.do(ctx, http.MethodPost, "/v1/auth/two-factor/"+action, session, request, &result)
	return result, err
}
