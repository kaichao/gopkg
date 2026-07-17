package security

import (
	"os"
	"strconv"
	"time"
)

// ── JWT 配置 ──────────────────────────────────────────────────

// JWTConfig 是 JWT 验签器的配置。
type JWTConfig struct {
	PublicKeyFile  string        // Ed25519 验签公钥路径
	Issuer         string        // 期望签发者（空=不校验）
	JWKSURL        string        // JWKS 端点（替代公钥文件）
	JWKSRefresh    time.Duration // JWKS 刷新间隔（默认 1h）
	PrivateKeyFile string        // 可选：本地签发私钥路径
}

// ── Token Service 配置 ────────────────────────────────────────

// TokenServiceConfig 是远程 Token Service 的客户端配置。
type TokenServiceConfig struct {
	URL     string        // Token Service 地址
	Key     string        // SERVICE_KEY（调用凭证）
	Timeout time.Duration // 请求超时（默认 5s）
}

// ── 安全总配置 ────────────────────────────────────────────────

// SecurityConfig 是应用安全配置。
// 所有环境变量无应用前缀，多个应用可共用同一套配置。
type SecurityConfig struct {
	Enabled      bool
	JWT          JWTConfig
	TokenService TokenServiceConfig
}

// LoadConfig 从环境变量加载安全配置（无前缀）。
//
//	SECURITY_ENABLED            — 安全总开关（默认 false）
//	JWT_PUBLIC_KEY_FILE         — 验签公钥文件路径
//	JWT_JWKS_URL                — JWKS 端点
//	JWT_JWKS_REFRESH            — JWKS 刷新间隔（默认 3600s）
//	JWT_ISSUER                  — 期望签发者
//	JWT_PRIVATE_KEY_FILE        — 签发私钥文件路径（可选，本地签发用）
//	TOKEN_SERVICE_URL           — Token Service 地址（可选，远程签发用）
//	SERVICE_KEY                 — 调用 Token Service 的凭证
//	TOKEN_SERVICE_TIMEOUT       — Token Service 超时（默认 5s）
func LoadConfig() SecurityConfig {
	return SecurityConfig{
		Enabled: getEnvBool("SECURITY_ENABLED", false),
		JWT: JWTConfig{
			PublicKeyFile:  getEnvString("JWT_PUBLIC_KEY_FILE", ""),
			Issuer:         getEnvString("JWT_ISSUER", ""),
			JWKSURL:        getEnvString("JWT_JWKS_URL", ""),
			JWKSRefresh:    getEnvDuration("JWT_JWKS_REFRESH", 3600*time.Second),
			PrivateKeyFile: getEnvString("JWT_PRIVATE_KEY_FILE", ""),
		},
		TokenService: TokenServiceConfig{
			URL:     getEnvString("TOKEN_SERVICE_URL", ""),
			Key:     getEnvString("SERVICE_KEY", ""),
			Timeout: getEnvDuration("TOKEN_SERVICE_TIMEOUT", 5*time.Second),
		},
	}
}

// ── 环境变量辅助 ──────────────────────────────────────────────

func getEnvString(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return defaultValue
}
