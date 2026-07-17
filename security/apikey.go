package security

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// APIKeyVerifier 实现 TokenAuthenticator，从 token 中提取 API Key 并验证。
//
// 用法：
//
//	store := &MyKeyStore{db: pool}
//	verifier := security.NewAPIKeyVerifier(store, "exastore_key_")
//	id, err := verifier.AuthenticateToken(ctx, "exastore_key_a1b2c3...")
type APIKeyVerifier struct {
	store     KeyStore
	keyPrefix string // 区分不同应用的 key 前缀（如 "exastore_key_"）
}

// NewAPIKeyVerifier 创建 API Key 验证器。
// keyPrefix 用于识别和剥离 key 前缀，空字符串表示无前缀。
func NewAPIKeyVerifier(store KeyStore, keyPrefix string) *APIKeyVerifier {
	return &APIKeyVerifier{store: store, keyPrefix: keyPrefix}
}

var _ TokenAuthenticator = (*APIKeyVerifier)(nil)

// AuthenticateToken 实现 TokenAuthenticator 接口。
func (v *APIKeyVerifier) AuthenticateToken(ctx context.Context, token string) (Identity, error) {
	key := token
	if v.keyPrefix != "" {
		if !strings.HasPrefix(token, v.keyPrefix) {
			return nil, fmt.Errorf("invalid api key format")
		}
		key = strings.TrimPrefix(token, v.keyPrefix)
	}

	if key == "" {
		return nil, fmt.Errorf("empty api key")
	}

	hash := keyHash(key)
	return v.store.LookupKey(ctx, hash)
}

// GenerateAPIKey 生成一个随机 API Key 及其 SHA-256 哈希。
// 返回 (原始 key, 哈希值)。原始 key 仅在此时可见，之后只能通过哈希验证。
func GenerateAPIKey(prefix string) (rawKey, hash string) {
	raw := make([]byte, 32)
	for i := range raw {
		raw[i] = keyChars[seededRand.Intn(len(keyChars))]
	}
	rawKey = prefix + string(raw)
	hash = keyHash(rawKey)
	return
}

func keyHash(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

// ── 随机字符生成 ────────────────────────────────────────────

var keyChars = []byte("abcdefghijklmnopqrstuvwxyz0123456789")

var seededRand = &seededRandSource{seed: time.Now().UnixNano()}

type seededRandSource struct {
	seed int64
}

func (s *seededRandSource) Intn(n int) int {
	s.seed = (s.seed*1103515245 + 12345) & 0x7fffffff
	return int(s.seed) % n
}
