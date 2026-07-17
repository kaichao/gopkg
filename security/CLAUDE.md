# CLAUDE.md — pkg/security

协议无关的安全框架。提供全套密码学引擎（JWT、APIKey）、中间件（gRPC 拦截器、HTTP 中间件）、接口定义和默认 PostgreSQL 实现。本包为独立开源项目，不依赖 scalebox 特有逻辑。

## 文件职责

| 文件 | 职责 |
|------|------|
| `doc.go` | Package 级别文档，架构说明 |
| `interface.go` | 核心接口：Identity、TokenAuthenticator、Authenticator、Authorizer、PermissionStore、KeyStore、TokenBlacklist、AuditStore、BillingService |
| `config.go` | SecurityConfig + JWTConfig + TokenServiceConfig + LoadConfig()（环境变量无前缀） |
| `jwt.go` | JWTVerifier（Ed25519 验签+缓存+黑名单）、JWTSigner（签发）、JWTClaims（UnmarshalJSON/MarshalJSON 双向 Unix↔Time 转换+ToPrincipal）、JWKS 拉取 |
| `apikey.go` | APIKeyVerifier（SHA-256 验 Key）+ GenerateAPIKey |
| `authz.go` | StoreAuthorizer（PermissionStore → Authorizer 包装） |
| `audit.go` | AuditRecorder（HTTP 中间件，异步写审计日志） |
| `adapter.go` | AuthenticatorFromToken / TokenFromAuthenticator 双向适配 |
| `noop.go` | NoopAuthenticator、NoopAuthorizer、NoopBillingService、NoopTokenAuthenticator（匿名 admin） |
| `middleware.go` | SecurityModule + Options 模式 NewModule + gRPC Unary/Stream 拦截器 |
| `middleware_http.go` | SecurityHandler + Options 模式 NewHandler + HTTP 中间件 + IdentityFromContext + 路径匹配 |
| `pg.go` | pgTokenBlacklist、pgKeyStore、pgAuditStore（默认 PG 实现）+ WithDB/WithDBModule/WithPermissionStore/WithPermissionStoreModule |
| `file_perm.go` | FilePermissionStore（YAML 驱动，零代码方案） |
| `jwt_test.go` | JWT 单元测试 |
| `security_test.go` | 集成测试 |

## 核心类型

### Principal

```go
type Principal struct {
    ID              string            // 用户唯一标识（JWT sub）
    Username        string            // 人类可读用户名
    Roles           []string          // RBAC 角色列表
    AllowedClusters []string          // 可访问集群（"*" = 全部）
    Attrs           map[string]string // 扩展属性
    ExpiresAt       time.Time         // 会话过期时间
}
```

实现 Identity 接口，*Principal 可直接当 Identity 使用。

### Identity

```go
type Identity interface {
    Subject() string                // 唯一标识
    Name() string                   // 显示名
    RoleList() []string             // 角色列表
    Attr(key string) (string, bool) // 扩展属性
}
```

### TokenAuthenticator（协议无关）

```go
type TokenAuthenticator interface {
    AuthenticateToken(ctx context.Context, token string) (Identity, error)
}
```

HTTP 中间件和 gRPC 拦截器均可使用。

### PermissionStore

```go
type PermissionStore interface {
    CheckPermission(ctx context.Context, roles []string, resource, action, resourceID string) (bool, error)
}
```

由各应用实现角色→权限映射规则。框架提供 FilePermissionStore（YAML 驱动）作为零代码方案。

### Additional Stores

```go
type KeyStore interface {
    LookupKey(ctx context.Context, keyHash string) (Identity, error)
}
type TokenBlacklist interface {
    IsBlacklisted(ctx context.Context, jti string) (bool, error)
}
type AuditStore interface {
    Record(ctx context.Context, entry AuditEntry) error
}
```

三个接口均有默认 PG 实现（pgKeyStore、pgTokenBlacklist、pgAuditStore），通过 `WithDB` 一行注入。

### MethodMapping

```go
type MethodMapping struct {
    Resource   string
    Action     string
    ResourceID string  // 可选，用于细粒度授权
}
```

### SecurityHandler（HTTP 中间件）

```go
type SecurityHandler struct {
    // 内部持有 TokenAuthenticator, Authorizer, routeMap, skipSet, log
}
```

### SecurityModule（gRPC 拦截器）

```go
type SecurityModule struct {
    // 内部持有 Authenticator, Authorizer, BillingService, methodMap, blacklist, log
}
```

## 关键函数

### HTTP 中间件（推荐）

- `NewHandler(opts ...Option) *SecurityHandler` — Options 模式创建 HTTP 中间件
- `NewSecurityHandler(cfg SecurityHandlerConfig) *SecurityHandler` — 显式构造（向后兼容）
- `(*SecurityHandler).Middleware(next http.Handler) http.Handler` — 返回标准 HTTP 中间件

### gRPC 模块

- `NewModule(opts ...ModuleOption) *SecurityModule` — Options 模式创建 gRPC 模块
- `NewModuleWith(auth Authenticator, authz Authorizer, methodMap map[string]MethodMapping) *SecurityModule` — 显式构造（向后兼容）
- `(*SecurityModule).UnaryInterceptor() grpc.UnaryServerInterceptor`
- `(*SecurityModule).StreamInterceptor() grpc.StreamServerInterceptor`

### Options

| Option | 适用 | 说明 |
|--------|:--:|------|
| `WithDB(pool)` | HTTP | 注入 pgxpool，自动设置 Blacklist/KeyStore/AuditStore |
| `WithDBModule(pool)` | gRPC | 同上，用于 gRPC 路径 |
| `WithPermissionStore(store)` | HTTP | 设置权限存储，自动创建 StoreAuthorizer |
| `WithPermissionStoreModule(store)` | gRPC | 同上，用于 gRPC 路径 |
| `WithSkipPaths(paths...)` | HTTP | 跳过认证的路径 |
| `WithRouteMap(m)` | HTTP | HTTP 路由映射 |
| `WithSigner(signer)` | HTTP | JWT 签发器 |
| `WithAuthenticator(a)` | gRPC | 自定义认证器 |
| `WithAuthorizer(a)` | gRPC | 自定义授权器 |
| `WithMethodMap(m)` | gRPC | gRPC 方法映射 |
| `WithBlacklist(bl)` | gRPC | Token 黑名单 |
| `WithAuditStore(as)` | gRPC | 审计存储 |

### 引擎

- `NewJWTVerifier(cfg JWTVerifierConfig) (*JWTVerifier, error)` — 创建 Ed25519 验签器
- `NewJWTSigner(cfg JWTSignerConfig) (*JWTSigner, error)` — 创建 EdDSA 签发器
- `(*JWTSigner).Sign(subject, username string, roles []string, ttl time.Duration) (string, error)`
- `(*JWTVerifier).SetBlacklist(bl TokenBlacklist)`
- `NewAPIKeyVerifier(store KeyStore, keyPrefix string) *APIKeyVerifier`
- `NewStoreAuthorizer(store PermissionStore) Authorizer`
- `NewAuditRecorder(store AuditStore) *AuditRecorder`
- `(*AuditRecorder).Middleware(next http.Handler) http.Handler`
- `NewFilePermissionStore(path string) (*FilePermissionStore, error)` — YAML 驱动
- `GenerateAPIKey(prefix string) (rawKey, hash string)` — 生成 API Key 对

### 适配器

- `AuthenticatorFromToken(ta TokenAuthenticator) Authenticator`
- `TokenFromAuthenticator(a Authenticator) TokenAuthenticator`

### 配置

- `LoadConfig() SecurityConfig` — 从环境变量加载（无前缀）

### 上下文查询

- `IdentityFromContext(ctx) Identity` — HTTP context
- `PrincipalFromHTTPContext(ctx) *Principal` — HTTP context
- `PrincipalFromContext(ctx) *Principal` — gRPC context

### 工具

- `Record(ctx, store, userID, action, resource, detail)` — 手动记录审计日志

## 架构模式

**Options 模式（推荐）：**

```go
// HTTP 路径
handler := security.NewHandler(
    security.WithDB(pool),
    security.WithPermissionStore(&ExastorePermissionStore{}),
    security.WithSkipPaths("GET /health", "POST /api/auth/login"),
)

// gRPC 路径
mod := security.NewModule(
    security.WithDBModule(pool),
    security.WithPermissionStoreModule(&ScaleboxPermissionStore{}),
    security.WithMethodMap(methodMap),
)
```

## HTTP 中间件执行流程

```
1. 检查 skipPaths → 在列表中 → 直接放行

2. 认证：extractToken(r) → AuthenticateToken(ctx, token)
   └─ auth==nil → 注入匿名 admin（noop 模式）
   └─ Identity 注入 context

3. 授权：mapRoute(methodPath) → Authorizer.Authorize(ctx, principal, resource, action, resourceID)
   └─ 拒绝 → HTTP 403

4. 执行业务 handler
```

## gRPC 拦截器执行流程

```
1. 检查 skipMethods（健康检查/反射）→ 放行

2. 认证：metadata → Authenticator.Authenticate(ctx, md)
   └─ Principal 注入 context

3. 授权：mapMethod(fullMethod) → Authorizer.Authorize()
   └─ 拒绝 → PermissionDenied

4. 执行业务 handler

5. 记账（异步）
```

## 环境变量（无前缀）

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `SECURITY_ENABLED` | `false` | 安全总开关 |
| `JWT_PUBLIC_KEY_FILE` | — | Ed25519 验签公钥 |
| `JWT_ISSUER` | — | 期望签发者（空=不校验） |
| `JWT_JWKS_URL` | — | JWKS 端点 |
| `JWT_JWKS_REFRESH` | `3600s` | JWKS 刷新间隔 |
| `JWT_PRIVATE_KEY_FILE` | — | 签发私钥（可选，本地签发用） |
| `TOKEN_SERVICE_URL` | — | Token Service 地址（可选） |
| `SERVICE_KEY` | — | Token Service 调用凭证 |
| `TOKEN_SERVICE_TIMEOUT` | `5s` | Token Service 超时 |

## 外部依赖

- `google.golang.org/grpc` — gRPC interceptor、metadata、status codes
- `github.com/sirupsen/logrus` — 结构化日志
- `github.com/jackc/pgx/v5` — PG 默认实现
- `gopkg.in/yaml.v3` — YAML 权限文件解析
- `golang.org/x/crypto` — Ed25519 签名验证
- `net/http` — HTTP 标准库

本包不依赖 gopkg 内部任何子包，是独立的叶子包。
