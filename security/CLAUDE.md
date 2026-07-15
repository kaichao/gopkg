# CLAUDE.md - pkg/security

可插拔安全框架的接口定义、注册表、缺省实现、gRPC 拦截器与 HTTP 中间件。本包设计为独立开源项目，不依赖 scalebox 特有逻辑。

## 文件职责

| 文件 | 职责 |
|------|------|
| `doc.go` | Package 级别文档，架构图与使用说明 |
| `interface.go` | `Principal`、`Identity`、`Authenticator`、`TokenAuthenticator`、`Authorizer`、`PermissionStore`、`BillingService`、`UsageRecord` 类型定义 |
| `registry.go` | 工厂类型定义 + `Register*` / `New*` / `AvailableModes` 函数（含 TokenAuthenticator 注册表） |
| `adapter.go` | `AuthenticatorFromToken` / `TokenFromAuthenticator` 双向适配器 |
| `noop.go` | 缺省空实现（匿名 admin、全部放行、不记账） |
| `plugin.go` | `LoadPlugins(enabled, dir)` 遍历目录加载 .so 文件 |
| `config.go` | `SecurityConfig` + 子配置结构体 + `LoadConfig()` 环境变量读取 + `BuildPGConnectionString()` |
| `middleware.go` | `SecurityModule` + `UnaryServerInterceptor` + `StreamServerInterceptor`（gRPC） |
| `middleware_http.go` | `SecurityHandler` + HTTP 中间件 + `IdentityFromContext` + 路径模式匹配（HTTP） |

## 核心类型

### Principal（保留，实现 Identity 接口）
```go
type Principal struct {
    ID              string            // 用户唯一标识
    Username        string            // 人类可读用户名
    Roles           []string          // RBAC 角色列表
    AllowedClusters []string          // 可访问集群（"*" = 全部）
    Attrs           map[string]string // 扩展属性
    ExpiresAt       time.Time         // 会话过期时间
}
```

### Identity（新增）
```go
type Identity interface {
    Subject() string                // 唯一标识
    Name() string                   // 显示名
    RoleList() []string             // 角色列表（避免与 Principal.Roles 字段冲突）
    Attr(key string) (string, bool) // 扩展属性
}
```
Principal 实现 Identity 接口，*Principal 可直接当 Identity 使用。

### TokenAuthenticator（新增）
```go
type TokenAuthenticator interface {
    AuthenticateToken(ctx context.Context, token string) (Identity, error)
}
```
协议无关的认证接口，HTTP 中间件和 gRPC 拦截器均可使用。

### PermissionStore（新增）
```go
type PermissionStore interface {
    CheckPermission(ctx context.Context, roles []string, resource, action, resourceID string) (bool, error)
}
```
由各应用实现角色→权限映射规则，通用 RBAC 引擎通过此接口查询。

### SecurityHandler（新增）
```go
type SecurityHandler struct { /* 内部持有 TokenAuthenticator、Authorizer、routeMap、skipSet */ }
```
HTTP 安全中间件，与 SecurityModule（gRPC）平级。

### MethodMapping
```go
type MethodMapping struct {
    Resource   string  // 资源名（如 "app", "cluster", "task"）
    Action     string  // 操作名（如 "read", "write", "execute"）
    ResourceID string  // 可选，资源 ID（如 "42" 表示 app-42），用于细粒度授权
}
```

### SecurityModule
```go
type SecurityModule struct { /* 内部持有 authenticator, authorizer, billing, methodMap */ }
```

### SecurityConfig
包含所有安全相关配置：Enabled、PluginDir、gRPC TLS、PostgreSQL TLS、Auth、AuthZ、Billing。

## 关键函数

### 配置
- `LoadConfig() SecurityConfig` — 从环境变量加载完整配置
- `(*SecurityConfig).AuthCfg() AuthConfig` — 提取认证子配置
- `(*SecurityConfig).AuthzCfg() AuthzConfig` — 提取授权子配置
- `(*SecurityConfig).BillCfg() BillConfig` — 提取记账子配置
- `(*SecurityConfig).BuildPGConnectionString(base string) string` — 为 PG 连接串追加 TLS 参数

### gRPC 模块（现有，保留）
- `NewModule(cfg SecurityConfig, methodMap map[string]MethodMapping) (*SecurityModule, error)` — cfg.Enabled==false 时返回 nil

### HTTP 模块（新增）
- `NewSecurityHandler(cfg SecurityHandlerConfig) *SecurityHandler` — 显式构造，不依赖全局注册表
- `(*SecurityHandler).Middleware(next http.Handler) http.Handler` — 返回标准 HTTP 中间件

### 拦截器
- `(*SecurityModule).UnaryInterceptor() grpc.UnaryServerInterceptor` — gRPC 认证→授权→执行→记账
- `(*SecurityModule).StreamInterceptor() grpc.StreamServerInterceptor` — 同流程，包装 ServerStream 注入 Principal

### 适配器（新增）
- `AuthenticatorFromToken(ta TokenAuthenticator) Authenticator` — TokenAuth → gRPC Auth 适配
- `TokenFromAuthenticator(a Authenticator) TokenAuthenticator` — gRPC Auth → TokenAuth 适配

### 插件
- `LoadPlugins(enabled bool, dir string) error` — enabled==false 或 dir=="" 直接返回；目录不存在静默跳过

### 查询
- `PrincipalFromContext(ctx context.Context) *Principal` — 从 gRPC context 取出 Principal
- `IdentityFromContext(ctx context.Context) Identity` — 从 HTTP context 取出 Identity
- `PrincipalFromHTTPContext(ctx context.Context) *Principal` — 从 HTTP context 取出 *Principal
- `AvailableModes() (auths, tokenAuths, authzs, bills []string)` — 已注册模式列表（4 个返回值）

### 注册（供插件 init() 调用）
- `RegisterAuthenticator(name string, fn AuthenticatorFactory)` — 重复注册会 panic
- `RegisterTokenAuthenticator(name string, fn TokenAuthenticatorFactory)` — 重复注册会 panic
- `RegisterAuthorizer(name string, fn AuthorizerFactory)` — 重复注册会 panic
- `RegisterBillingService(name string, fn BillingServiceFactory)` — 重复注册会 panic

### 工厂
- `NewAuthenticator(mode string, cfg AuthConfig) (Authenticator, error)` — mode 空或 "noop" 返回 NoopAuthenticator
- `NewTokenAuthenticator(mode string, cfg AuthConfig) (TokenAuthenticator, error)` — mode 空或 "noop" 返回 nil
- `NewAuthorizer(mode string, cfg AuthzConfig) (Authorizer, error)` — mode 空或 "noop" 返回 NoopAuthorizer
- `NewBillingService(mode string, cfg BillConfig) (BillingService, error)` — mode 空或 "noop" 返回 NoopBillingService

## 架构模式

**插件模式（gRPC，保留）：**

```
外部 .so 插件 init()
  → security.RegisterAuthenticator("jwt", factory)
  → registry 填充
  → NewModule() 按 mode 创建实例
  → gRPC interceptor 调用
```

**显式构造模式（HTTP，推荐新项目使用）：**

```
应用构造 TokenAuthenticator 实例
  → security.NewSecurityHandler(cfg)
  → handler.Middleware(apiMux)
  → HTTP 中间件：提取 token → 认证 → 注入 Identity → 授权 → handler
```

## gRPC 拦截器执行流程

```
1. skipAuth(fullMethod)
   └─ 健康检查/反射 → 直接放行

2. authenticate(ctx)
   └─ metadata.FromIncomingContext → Authenticator.Authenticate()
   └─ Principal 注入 context

3. mapMethod(fullMethod) → resource, action, resourceID
   └─ Authorizer.Authorize(ctx, principal, resource, action, resourceID)
   └─ 拒绝 → gRPC PermissionDenied

4. handler(ctx, req)

5. recordUsage()（异步，不阻塞响应）
   └─ BillingService.Record()
```

## HTTP 中间件执行流程

```
1. skipPath(methodPath)
   └─ 在 SkipPaths 中 → 直接放行

2. authenticate(r)
   └─ Authorization header 或 access_token cookie → TokenAuthenticator.AuthenticateToken()
   └─ Identity 注入 context（IdentityFromContext / PrincipalFromHTTPContext 取回）

3. mapRoute(methodPath) → resource, action, resourceID
   └─ 精确匹配 → 模式匹配（{barcode} 通配符）→ 路径推断
   └─ Authorizer.Authorize(ctx, principal, resource, action, resourceID)
   └─ 拒绝 → HTTP 403

4. handler(w, r)
```

## 独立项目设计

1. **环境变量无前缀**：`SECURITY_ENABLED`、`AUTH_MODE`（非 `SCALEBOX_*`）
2. **双协议支持**：gRPC 走 SecurityModule + 拦截器，HTTP 走 SecurityHandler + 中间件
3. **方法/路径映射由调用方注入**：gRPC 传入 methodMap，HTTP 传入 routeMap
4. **角色模型由应用定义**：PermissionStore 接口推给应用实现，框架不硬编码角色名和资源名
5. **Principal 使用通用 claims**：不假设特定 JWT 结构，通过 Identity.Attr() 扩展属性

## 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `SECURITY_ENABLED` | `false` | 总开关，false 时 NewModule 返回 nil |
| `SECURITY_PLUGIN_DIR` | — | .so 插件目录 |
| `GRPC_TLS_ENABLED` | `false` | gRPC TLS 开关 |
| `AUTH_MODE` | `noop` | noop / jwt / oauth2 / external |
| `AUTHZ_MODE` | `noop` | noop / rbac / external |
| `BILLING_MODE` | `noop` | noop / pg / kafka / external |
| `PG_SSLMODE` | `disable` | PostgreSQL TLS |

完整列表见 `config.go` 中的 `LoadConfig()`。

## 使用示例

### 集成到 gRPC 服务

```go
// 在 controld 中：
cfg := security.LoadConfig()
mod, err := security.NewModule(cfg, map[string]security.MethodMapping{
    "/scalebox.ControlService/GetTaskList": {"app", "read"},
    "/scalebox.ControlService/GetSlotList": {"cluster", "read"},
})

if mod != nil {
    s := grpc.NewServer(
        grpc.UnaryInterceptor(mod.UnaryInterceptor()),
        grpc.StreamInterceptor(mod.StreamInterceptor()),
    )
}
```

### 插件加载

```go
// controld / agent / actuator 启动时：
security.LoadPlugins(cfg.Enabled, cfg.PluginDir)
// .so 存在 → plugin.Open() → init() → Register* → registry 填充
// .so 不存在 → 静默跳过，registry 只有 noop
```

### 集成到 HTTP 服务（新增）

```go
// 显式构造，不依赖插件和全局注册表：
verifier := jwt.NewVerifier(jwt.Config{
    PublicKeyFile: "/etc/myapp/jwt/public.pem",
    Issuer:        "myapp",
})
handler := security.NewSecurityHandler(security.SecurityHandlerConfig{
    Auth:  verifier,
    Authz: &security.NoopAuthorizer{},
    RouteMap: map[string]security.MethodMapping{
        "POST /api/tapes/import":    {Resource: "tape", Action: "create"},
        "GET  /api/tapes/{barcode}": {Resource: "tape", Action: "read"},
    },
    SkipPaths: []string{"GET /login", "GET /health"},
})
mux.Handle("/api/", handler.Middleware(apiMux))
```

## 外部依赖

- `google.golang.org/grpc` — gRPC interceptor、metadata、status codes（middleware.go、interface.go、adapter.go）
- `net/http` — HTTP 标准库（middleware_http.go）
- `github.com/sirupsen/logrus` — 结构化日志（plugin.go、middleware.go、middleware_http.go）

## 与其他包的关系

本包不依赖 gopkg 内部任何子包。是独立的叶子包。
