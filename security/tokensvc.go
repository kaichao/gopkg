package security

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// TokenServiceClient 是远程 Token Service 的 HTTP 客户端。
// 用于 login handler 调用远程签发 JWT，logout handler 调用远程撤销。
type TokenServiceClient struct {
	url     string
	key     string
	client  *http.Client
}

// NewTokenServiceClient 创建 Token Service 客户端。
func NewTokenServiceClient(url, serviceKey string) *TokenServiceClient {
	return &TokenServiceClient{
		url: url,
		key: serviceKey,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SignResponse 是 Token Service 签发的响应。
type SignResponse struct {
	Token string `json:"token"`
	JTI   string `json:"jti"`
	Exp   int64  `json:"exp"`
}

// Sign 调用 Token Service 签发 JWT。
func (c *TokenServiceClient) Sign(ctx context.Context, sub, username string, roles []string, ttl time.Duration) (*SignResponse, error) {
	body := map[string]interface{}{
		"sub":      sub,
		"username": username,
		"roles":    roles,
	}
	if ttl > 0 {
		body["ttl"] = ttl.String()
	}

	reqBody, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", c.url+"/v1/token/sign", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("token service sign request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.key)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token service sign: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token service sign: %s (%s)", resp.Status, string(body))
	}

	var sr SignResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return nil, fmt.Errorf("token service sign response: %w", err)
	}
	return &sr, nil
}

// Verify 调用 Token Service 验证 JWT。
func (c *TokenServiceClient) Verify(ctx context.Context, token string) (bool, error) {
	reqBody, _ := json.Marshal(map[string]string{"token": token})
	req, err := http.NewRequestWithContext(ctx, "POST", c.url+"/v1/token/verify", bytes.NewReader(reqBody))
	if err != nil {
		return false, fmt.Errorf("token service verify request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.key)

	resp, err := c.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("token service verify: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, nil
	}

	var vr struct {
		Valid bool `json:"valid"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&vr); err != nil {
		return false, fmt.Errorf("token service verify response: %w", err)
	}
	return vr.Valid, nil
}

// Revoke 调用 Token Service 撤销 JWT。
func (c *TokenServiceClient) Revoke(ctx context.Context, token string) error {
	reqBody, _ := json.Marshal(map[string]string{"token": token})
	req, err := http.NewRequestWithContext(ctx, "POST", c.url+"/v1/token/revoke", bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("token service revoke request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.key)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("token service revoke: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token service revoke: %s (%s)", resp.Status, string(body))
	}
	return nil
}
