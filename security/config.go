package security

import (
	"net/url"
	"os"
	"strconv"
	"time"
)

// AuthConfig 是创建 Authenticator 所需的配置子集。
type AuthConfig struct {
	Mode  string // noop | jwt | oauth2 | external
	Token string // 客户端注入的 JWT token

	// JWT
	JWTPublicKeyFile string
	JWTAlgorithm     string
	JWTIssuer        string
	JWTJWKSURL       string
	JWTJWKSRefresh   time.Duration

	// OAuth2
	OAuth2IntrospectionURL string
	OAuth2ClientID         string
	OAuth2ClientSecret     string

	// External
	ExternalAuthURL     string
	ExternalAuthTimeout time.Duration
}

// AuthzConfig 是创建 Authorizer 所需的配置子集。
type AuthzConfig struct {
	Mode string // noop | rbac | external

	// External
	ExternalAuthZURL     string
	ExternalAuthZTimeout time.Duration
}

// BillConfig 是创建 BillingService 所需的配置子集。
type BillConfig struct {
	Mode        string // noop | pg | kafka | external
	BatchSize   int
	FlushInterval time.Duration
	KafkaBrokers     string
	KafkaTopic       string
}

// SecurityConfig 包含所有安全相关配置。
type SecurityConfig struct {
	Enabled   bool
	PluginDir string

	// TLS
	GRPCTLSEnabled bool
	GRPCCertFile   string
	GRPCKeyFile    string
	GRPCCAFile     string

	// PostgreSQL
	PGSSLMode string
	PGCertFile string
	PGKeyFile  string
	PGCAFile   string
	PGPassword string

	// Auth
	AuthMode  string // noop | jwt | oauth2 | external
	AuthToken string // 客户端注入的 token

	// JWT
	JWTPublicKeyFile string
	JWTAlgorithm     string
	JWTIssuer        string
	JWTJWKSURL       string
	JWTJWKSRefresh   time.Duration

	// OAuth2
	OAuth2IntrospectionURL string
	OAuth2ClientID         string
	OAuth2ClientSecret     string

	// External Auth
	ExternalAuthURL     string
	ExternalAuthTimeout time.Duration

	// AuthZ
	AuthZMode string // noop | rbac | external

	// External AuthZ
	ExternalAuthZURL     string
	ExternalAuthZTimeout time.Duration

	// Billing
	BillingMode        string // noop | pg | kafka | external
	BillingBatchSize   int
	BillingFlushInterval time.Duration
	KafkaBrokers       string
	KafkaTopic         string
}

// AuthCfg 从 SecurityConfig 提取 AuthConfig。
func (c *SecurityConfig) AuthCfg() AuthConfig {
	return AuthConfig{
		Mode:                   c.AuthMode,
		Token:                  c.AuthToken,
		JWTPublicKeyFile:       c.JWTPublicKeyFile,
		JWTAlgorithm:           c.JWTAlgorithm,
		JWTIssuer:              c.JWTIssuer,
		JWTJWKSURL:             c.JWTJWKSURL,
		JWTJWKSRefresh:         c.JWTJWKSRefresh,
		OAuth2IntrospectionURL: c.OAuth2IntrospectionURL,
		OAuth2ClientID:         c.OAuth2ClientID,
		OAuth2ClientSecret:     c.OAuth2ClientSecret,
		ExternalAuthURL:        c.ExternalAuthURL,
		ExternalAuthTimeout:    c.ExternalAuthTimeout,
	}
}

// AuthzCfg 从 SecurityConfig 提取 AuthzConfig。
func (c *SecurityConfig) AuthzCfg() AuthzConfig {
	return AuthzConfig{
		Mode:                 c.AuthZMode,
		ExternalAuthZURL:     c.ExternalAuthZURL,
		ExternalAuthZTimeout: c.ExternalAuthZTimeout,
	}
}

// BillCfg 从 SecurityConfig 提取 BillConfig。
func (c *SecurityConfig) BillCfg() BillConfig {
	return BillConfig{
		Mode:          c.BillingMode,
		BatchSize:     c.BillingBatchSize,
		FlushInterval: c.BillingFlushInterval,
		KafkaBrokers:  c.KafkaBrokers,
		KafkaTopic:    c.KafkaTopic,
	}
}

// BuildPGConnectionString 根据安全配置为 PostgreSQL 连接串追加 TLS 参数。
// base 为原始连接串（如 postgres://user:pass@host:5432/dbname）。
// 安全未启用或 sslmode=disable 时原样返回。
func (c *SecurityConfig) BuildPGConnectionString(base string) string {
	if !c.Enabled || c.PGSSLMode == "" || c.PGSSLMode == "disable" {
		return base
	}

	u, err := url.Parse(base)
	if err != nil {
		return base
	}
	q := u.Query()
	q.Set("sslmode", c.PGSSLMode)
	if c.PGCertFile != "" {
		q.Set("sslcert", c.PGCertFile)
	}
	if c.PGKeyFile != "" {
		q.Set("sslkey", c.PGKeyFile)
	}
	if c.PGCAFile != "" {
		q.Set("sslrootcert", c.PGCAFile)
	}
	if c.PGPassword != "" {
		q.Set("password", c.PGPassword)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// LoadConfig 从环境变量加载安全配置。
func LoadConfig() SecurityConfig {
	return SecurityConfig{
		Enabled:   getEnvBool("SECURITY_ENABLED", false),
		PluginDir: os.Getenv("SECURITY_PLUGIN_DIR"),

		GRPCTLSEnabled: getEnvBool("GRPC_TLS_ENABLED", false),
		GRPCCertFile:   getEnvString("GRPC_TLS_CERT_FILE", "/etc/scalebox/tls/server.crt"),
		GRPCKeyFile:    getEnvString("GRPC_TLS_KEY_FILE", "/etc/scalebox/tls/server.key"),
		GRPCCAFile:     getEnvString("GRPC_TLS_CA_FILE", "/usr/local/etc/ca.crt"),

		PGSSLMode:  getEnvString("PG_SSLMODE", "disable"),
		PGCertFile: os.Getenv("PG_SSL_CERT_FILE"),
		PGKeyFile:  os.Getenv("PG_SSL_KEY_FILE"),
		PGCAFile:   os.Getenv("PG_SSL_CA_FILE"),
		PGPassword: os.Getenv("PG_PASSWORD"),

		AuthMode:  getEnvString("AUTH_MODE", "noop"),
		AuthToken: os.Getenv("AUTH_TOKEN"),

		JWTPublicKeyFile: getEnvString("JWT_PUBLIC_KEY_FILE", "/usr/local/etc/jwt/public.pem"),
		JWTAlgorithm:     getEnvString("JWT_ALGORITHM", "EdDSA"),
		JWTIssuer:        getEnvString("JWT_ISSUER", "gopkg-security"),
		JWTJWKSURL:       os.Getenv("JWT_JWKS_URL"),
		JWTJWKSRefresh:   getEnvDuration("JWT_JWKS_REFRESH_INTERVAL", 3600*time.Second),

		OAuth2IntrospectionURL: os.Getenv("OAUTH2_INTROSPECTION_URL"),
		OAuth2ClientID:         os.Getenv("OAUTH2_CLIENT_ID"),
		OAuth2ClientSecret:     os.Getenv("OAUTH2_CLIENT_SECRET"),

		ExternalAuthURL:     os.Getenv("EXTERNAL_AUTH_URL"),
		ExternalAuthTimeout: getEnvDuration("EXTERNAL_AUTH_TIMEOUT", 5*time.Second),

		AuthZMode: getEnvString("AUTHZ_MODE", "noop"),

		ExternalAuthZURL:     os.Getenv("EXTERNAL_AUTHZ_URL"),
		ExternalAuthZTimeout: getEnvDuration("EXTERNAL_AUTHZ_TIMEOUT", 3*time.Second),

		BillingMode:          getEnvString("BILLING_MODE", "noop"),
		BillingBatchSize:     getEnvInt("BILLING_BATCH_SIZE", 100),
		BillingFlushInterval: getEnvDuration("BILLING_FLUSH_INTERVAL", 5*time.Second),
		KafkaBrokers:         os.Getenv("KAFKA_BROKERS"),
		KafkaTopic:           getEnvString("KAFKA_TOPIC", "security.usage"),
	}
}

// ── 环境变量辅助函数 ──────────────────────────────────────────

func getEnvString(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
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
