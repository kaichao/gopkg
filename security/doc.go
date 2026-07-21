// Package security 提供协议无关的安全框架，支持 gRPC 和 HTTP。
//
// 架构：
// gopkg/security 包含全套密码学引擎（JWT、APIKey）、中间件（gRPC 拦截器、HTTP 中间件）、
// 接口定义（PermissionStore 等）和默认 PostgreSQL 实现。
// 应用通过 NewHandler() / NewModule() Options 模式一行接入，仅需定义 PermissionStore。
//
// 核心接口：
//   - Identity：认证后的主体身份，Principal 实现
//   - TokenAuthenticator：协议无关认证（token → Identity）
//   - Authenticator：gRPC 认证（metadata → Principal）
//   - Authorizer：访问控制
//   - PermissionStore：角色→权限映射，由各应用实现
//   - TokenBlacklist：Token 撤销检查
//   - KeyStore：API Key 查询
//   - AuditStore：审计日志持久化
//   - BillingService：记账
//
// 内置引擎（全开源）：
//   - JWTVerifier + JWTSigner：Ed25519 JWT 验签+签发
//   - APIKeyVerifier：SHA-256 API Key 验证
//   - StoreAuthorizer：PermissionStore → Authorizer 包装
//   - AuditRecorder：HTTP 审计中间件
//   - FilePermissionStore：YAML 驱动权限（零代码方案）
//
// 默认 PG 实现（读侧，通过 WithDB 一行注入）：
//   - pgTokenBlacklist：查 t_token_blacklist
//   - pgKeyStore：查 t_api_key + t_user + t_role_binding
//   - pgAuditStore：写 t_audit_log
//
// 管理侧 PG 实现（通过构造函数独立创建）：
//   - pgUserStore：t_user CRUD（NewPGUserStore）
//   - pgRoleStore：t_role_binding CRUD + scope JSONB（NewPGRoleStore）
//   - pgAPIKeyManager：t_api_key CRUD + scope 限定（NewPGAPIKeyManager）
//
// 统一 schema 见 schema.sql。
//
// Options 模式（推荐）：
//
//	// HTTP 路径
//	handler := security.NewHandler(
//	    security.WithDB(pool),
//	    security.WithPermissionStore(&myStore{}),
//	    security.WithSkipPaths("GET /health", "POST /api/auth/login"),
//	)
//
//	// gRPC 路径
//	mod := security.NewModule(
//	    security.WithDBModule(pool),
//	    security.WithPermissionStoreModule(&myStore{}),
//	    security.WithMethodMap(methodMap),
//	)
//
// 环境变量（无前缀）：
//
//	SECURITY_ENABLED         — 总开关（false 时 noop 模式）
//	JWT_PUBLIC_KEY_FILE      — Ed25519 验签公钥
//	JWT_ISSUER               — 期望签发者
//	JWT_PRIVATE_KEY_FILE     — 可选：本地签发私钥
//	TOKEN_SERVICE_URL        — 可选：远程 Token Service 地址
//	SERVICE_KEY              — 调用 Token Service 的凭证
//
// 未启用安全时，NewHandler 注入匿名 admin（noop 模式），所有请求直接放行。
package security
