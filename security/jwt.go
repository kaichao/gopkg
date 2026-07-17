// Package security — JWT EdDSA/Ed25519 验签器
//
// 协议无关、应用无关的通用 JWT 验证引擎。
// 实现 TokenAuthenticator 接口，可直接用于 gRPC 和 HTTP 两种路径。
package security

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// ── JWT Claims ──────────────────────────────────────────────

// JWTClaims 是 JWT 令牌的 claims 结构体。
type JWTClaims struct {
	Subject         string            `json:"sub"`
	Issuer          string            `json:"iss"`
	Exp             time.Time         `json:"exp"`
	Iat             time.Time         `json:"iat"`
	Nbf             time.Time         `json:"nbf"`
	JTI             string            `json:"jti"`
	Username        string            `json:"username"`
	Roles           []string          `json:"roles"`
	AllowedClusters []string          `json:"allowed_clusters"`
	Attrs           map[string]string `json:"attrs"`
}

// UnmarshalJSON 处理 JWT 数字时间戳（Unix timestamp → time.Time）。
func (c *JWTClaims) UnmarshalJSON(data []byte) error {
	type raw JWTClaims
	type alias struct {
		*raw
		Exp json.Number `json:"exp"`
		Iat json.Number `json:"iat"`
		Nbf json.Number `json:"nbf"`
	}
	a := &alias{raw: (*raw)(c)}
	if err := json.Unmarshal(data, a); err != nil {
		return err
	}
	if v, _ := a.Exp.Int64(); v > 0 {
		c.Exp = time.Unix(v, 0)
	}
	if v, _ := a.Iat.Int64(); v > 0 {
		c.Iat = time.Unix(v, 0)
	}
	if v, _ := a.Nbf.Int64(); v > 0 {
		c.Nbf = time.Unix(v, 0)
	}
	return nil
}

// ToPrincipal 将 claims 转换为 Principal（实现 Identity 接口）。
func (c *JWTClaims) ToPrincipal() *Principal {
	allowed := c.AllowedClusters
	if len(allowed) == 0 {
		allowed = []string{"*"}
	}
	return &Principal{
		ID:              c.Subject,
		Username:        c.Username,
		Roles:           c.Roles,
		AllowedClusters: allowed,
		Attrs:           c.Attrs,
		ExpiresAt:       c.Exp,
	}
}

// ── JWT Verifier ────────────────────────────────────────────

// JWTVerifierConfig 是 JWT 验签器的配置。
type JWTVerifierConfig struct {
	PublicKeyFile string        // Ed25519 公钥文件路径
	JWKSURL       string        // JWKS 端点 URL（与 PublicKeyFile 二选一）
	JWKSRefresh   time.Duration // JWKS 刷新间隔（默认 1h）
	Issuer        string        // 期望的签发者（空 = 不校验）
}

// JWTVerifier 实现 TokenAuthenticator 接口，使用 EdDSA/Ed25519 验签。
//
// 支持两种密钥来源：
//   - 本地文件：直接读取 32 字节 Ed25519 公钥
//   - JWKS 端点：定期从远端拉取公钥集合，按 kid 匹配
//
// Token 验证结果缓存在内存中，缓存有效期对齐 token exp。
// 若 Blacklist 不为 nil，验签后检查 jti 是否在黑名单中。
type JWTVerifier struct {
	publicKey ed25519.PublicKey
	issuer    string
	keyFunc   func(kid string) (ed25519.PublicKey, error)
	blacklist TokenBlacklist
	cache     sync.Map // token SHA-256 hex → *cachedToken
}

// SetBlacklist 设置 token 黑名单（logout/refresh 时使用）。
func (v *JWTVerifier) SetBlacklist(bl TokenBlacklist) {
	v.blacklist = bl
}

type cachedToken struct {
	p         *Principal
	expiresAt time.Time
}

var _ TokenAuthenticator = (*JWTVerifier)(nil)

// NewJWTVerifier 创建 JWT 验签器。
// PublicKeyFile 和 JWKSURL 至少需要配置一个。
func NewJWTVerifier(cfg JWTVerifierConfig) (*JWTVerifier, error) {
	v := &JWTVerifier{
		issuer: cfg.Issuer,
	}

	// 本地公钥文件
	if cfg.PublicKeyFile != "" {
		keyBytes, err := os.ReadFile(cfg.PublicKeyFile)
		if err != nil {
			return nil, fmt.Errorf("read jwt public key file: %w", err)
		}
		if len(keyBytes) != ed25519.PublicKeySize {
			return nil, fmt.Errorf("invalid EdDSA public key size: %d, want %d", len(keyBytes), ed25519.PublicKeySize)
		}
		v.publicKey = ed25519.PublicKey(keyBytes)
		return v, nil
	}

	// JWKS 端点
	if cfg.JWKSURL != "" {
		refresh := cfg.JWKSRefresh
		if refresh <= 0 {
			refresh = 3600 * time.Second
		}
		v.keyFunc = newJWKSFetcher(cfg.JWKSURL, refresh)
		return v, nil
	}

	return nil, fmt.Errorf("jwt verifier requires either PublicKeyFile or JWKSURL")
}

// AuthenticateToken 实现 TokenAuthenticator 接口。
func (v *JWTVerifier) AuthenticateToken(ctx context.Context, token string) (Identity, error) {
	return v.verifyAndCache(ctx, token)
}

// ── 内部：验签 + 缓存 ──────────────────────────────────────

func (v *JWTVerifier) verifyAndCache(ctx context.Context, token string) (*Principal, error) {
	hash := tokenHash(token)
	if val, ok := v.cache.Load(hash); ok {
		ct := val.(*cachedToken)
		if time.Now().Before(ct.expiresAt) {
			return ct.p, nil
		}
		v.cache.Delete(hash)
	}

	p, err := v.verifyAndParse(ctx, token)
	if err != nil {
		return nil, err
	}

	ttl := time.Until(p.ExpiresAt)
	if ttl > 0 {
		v.cache.Store(hash, &cachedToken{p: p, expiresAt: p.ExpiresAt})
	}

	return p, nil
}

func (v *JWTVerifier) verifyAndParse(ctx context.Context, tokenStr string) (*Principal, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid jwt format")
	}

	message := parts[0] + "." + parts[1]
	sig, err := b64URLDecode(parts[2])
	if err != nil {
		return nil, fmt.Errorf("decode signature: %w", err)
	}

	// 获取公钥
	var pub ed25519.PublicKey
	if v.keyFunc != nil {
		headerJSON, _ := b64URLDecode(parts[0])
		var h struct {
			Alg string `json:"alg"`
			Kid string `json:"kid"`
		}
		json.Unmarshal(headerJSON, &h)
		pub, err = v.keyFunc(h.Kid)
		if err != nil {
			return nil, fmt.Errorf("fetch key: %w", err)
		}
	} else {
		pub = v.publicKey
	}

	if !ed25519.Verify(pub, []byte(message), sig) {
		return nil, fmt.Errorf("ed25519 signature verification failed")
	}

	// 解析 claims
	claimsJSON, err := b64URLDecode(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode claims: %w", err)
	}
	var claims JWTClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, fmt.Errorf("parse claims: %w", err)
	}

	// 时间校验
	now := time.Now()
	if !claims.Exp.IsZero() && now.After(claims.Exp) {
		return nil, fmt.Errorf("token expired")
	}
	if !claims.Nbf.IsZero() && now.Before(claims.Nbf) {
		return nil, fmt.Errorf("token not yet valid")
	}

	// issuer 校验
	if v.issuer != "" && claims.Issuer != v.issuer {
		return nil, fmt.Errorf("issuer mismatch: want %s, got %s", v.issuer, claims.Issuer)
	}

	// 黑名单检查（logout/refresh 时加入的 jti）
	if v.blacklist != nil && claims.JTI != "" {
		blocked, err := v.blacklist.IsBlacklisted(ctx, claims.JTI)
		if err != nil {
			return nil, fmt.Errorf("blacklist check: %w", err)
		}
		if blocked {
			return nil, fmt.Errorf("token has been revoked")
		}
	}

	return claims.ToPrincipal(), nil
}

// ── JWKS Fetcher ────────────────────────────────────────────

func newJWKSFetcher(url string, refresh time.Duration) func(kid string) (ed25519.PublicKey, error) {
	var (
		keys     = map[string]ed25519.PublicKey{}
		mu       sync.RWMutex
		lastTime time.Time
	)

	return func(kid string) (ed25519.PublicKey, error) {
		mu.RLock()
		if k, ok := keys[kid]; ok && time.Since(lastTime) < refresh {
			mu.RUnlock()
			return k, nil
		}
		mu.RUnlock()

		resp, err := http.Get(url)
		if err != nil {
			return nil, fmt.Errorf("fetch jwks: %w", err)
		}
		defer resp.Body.Close()

		var jwks struct {
			Keys []struct {
				Kid string `json:"kid"`
				X   string `json:"x"`
			} `json:"keys"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
			return nil, fmt.Errorf("decode jwks: %w", err)
		}

		mu.Lock()
		for _, k := range jwks.Keys {
			kb, _ := b64URLDecode(k.X)
			keys[k.Kid] = ed25519.PublicKey(kb)
		}
		lastTime = time.Now()
		mu.Unlock()

		mu.RLock()
		k, ok := keys[kid]
		mu.RUnlock()
		if !ok {
			return nil, fmt.Errorf("kid not found: %s", kid)
		}
		return k, nil
	}
}

// ── Base64URL ───────────────────────────────────────────────

// b64URLDecode 解码 base64url 编码（无 padding）为原始字节。
func b64URLDecode(s string) ([]byte, error) {
	s = strings.Map(func(r rune) rune {
		switch r {
		case '-':
			return '+'
		case '_':
			return '/'
		}
		return r
	}, s)
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}
	return base64.StdEncoding.DecodeString(s)
}

// tokenHash 计算 token 的 SHA-256 哈希（用于缓存键）。
func tokenHash(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// ── JWT Signer ──────────────────────────────────────────────

// JWTSigner 使用 Ed25519 私钥签发 JWT 令牌。
type JWTSigner struct {
	privateKey ed25519.PrivateKey
	issuer     string
}

// JWTSignerConfig 是 JWT 签发器的配置。
type JWTSignerConfig struct {
	PrivateKeyFile string // Ed25519 私钥文件路径
	Issuer         string // 签发者 (iss claim)
}

// NewJWTSigner 创建 JWT 签发器。
func NewJWTSigner(cfg JWTSignerConfig) (*JWTSigner, error) {
	keyBytes, err := os.ReadFile(cfg.PrivateKeyFile)
	if err != nil {
		return nil, fmt.Errorf("read jwt private key file: %w", err)
	}
	if len(keyBytes) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid EdDSA private key size: %d, want %d", len(keyBytes), ed25519.PrivateKeySize)
	}
	return &JWTSigner{
		privateKey: ed25519.PrivateKey(keyBytes),
		issuer:     cfg.Issuer,
	}, nil
}

// Sign 签发 JWT 令牌。
// ttl 为有效期，0 表示永不过期。
func (s *JWTSigner) Sign(subject, username string, roles []string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := JWTClaims{
		Subject:  subject,
		Issuer:   s.issuer,
		Username: username,
		Roles:    roles,
		Iat:      now,
	}
	if ttl > 0 {
		claims.Exp = now.Add(ttl)
	}

	header := `{"alg":"EdDSA","typ":"JWT"}`
	claimBytes, _ := json.Marshal(claims)

	headerB64 := b64URLEncode([]byte(header))
	claimsB64 := b64URLEncode(claimBytes)
	message := headerB64 + "." + claimsB64

	sig := ed25519.Sign(s.privateKey, []byte(message))
	return message + "." + b64URLEncode(sig), nil
}

func b64URLEncode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}
