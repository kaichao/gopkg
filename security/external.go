package security

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"google.golang.org/grpc/metadata"
)

// ── External Authenticator ─────────────────────────────────

type extAuthenticator struct {
	url    string
	client *http.Client
}

// NewExternalAuthenticatorWithURL 创建外部认证器（指定 URL 和超时）。
func NewExternalAuthenticatorWithURL(authURL string, timeout time.Duration) (Authenticator, error) {
	if authURL == "" {
		return nil, fmt.Errorf("external auth requires auth URL")
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &extAuthenticator{
		url:    authURL,
		client: &http.Client{Timeout: timeout},
	}, nil
}

func (a *extAuthenticator) Authenticate(ctx context.Context, md metadata.MD) (*Principal, error) {
	token := ""
	if vals := md.Get("authorization"); len(vals) > 0 {
		token = vals[0]
	}

	reqBody, _ := json.Marshal(map[string]string{"token": token})
	req, _ := http.NewRequestWithContext(ctx, "POST", a.url, bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("external auth: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Allowed   bool                `json:"allowed"`
		Principal *Principal `json:"principal,omitempty"`
		Reason    string              `json:"reason,omitempty"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if !result.Allowed {
		return nil, fmt.Errorf("external auth denied: %s", result.Reason)
	}
	return result.Principal, nil
}

// ── External Authorizer ────────────────────────────────────

type extAuthorizer struct {
	url    string
	client *http.Client
}

// NewExternalAuthorizerWithURL 创建外部授权器（指定 URL 和超时）。
func NewExternalAuthorizerWithURL(authzURL string, timeout time.Duration) (Authorizer, error) {
	if authzURL == "" {
		return nil, fmt.Errorf("external authz requires authz URL")
	}
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	return &extAuthorizer{
		url:    authzURL,
		client: &http.Client{Timeout: timeout},
	}, nil
}

func (a *extAuthorizer) Authorize(ctx context.Context, p *Principal, resource, action, resourceID string) error {
	reqBody, _ := json.Marshal(map[string]interface{}{
		"principal": p,
		"resource":  resource,
		"action":    action,
	})
	req, _ := http.NewRequestWithContext(ctx, "POST", a.url, bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("external authz: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Allowed bool   `json:"allowed"`
		Reason  string `json:"reason,omitempty"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if !result.Allowed {
		return fmt.Errorf("external authz denied: %s", result.Reason)
	}
	return nil
}
