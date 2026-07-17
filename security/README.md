# security

[![Go Reference](https://pkg.go.dev/badge/github.com/kaichao/gopkg/security.svg)](https://pkg.go.dev/github.com/kaichao/gopkg/security)

协议无关的安全框架，支持 gRPC 和 HTTP。提供全套密码学引擎、中间件、接口定义和默认 PostgreSQL 实现。

## 特性

- **双协议支持**：gRPC 走 `SecurityModule` + 拦截器，HTTP 走 `SecurityHandler` + 中间件
- **协议无关认证**：`TokenAuthenticator` 接口接收 token 字符串，不绑定 gRPC metadata
- **应用定义角色模型**：`PermissionStore` 接口由各应用实现，框架不硬编码角色名和资源名
- **内置密码学引擎**：JWTVerifier（Ed25519 验签）、JWTSigner（签发）、APIKeyVerifier（SHA-256）
- **默认 PG 实现**：pgTokenBlacklist、pgKeyStore、pgAuditStore，通过 `WithDB` 一行注入
- **零代码接入**：`FilePermissionStore`（YAML 驱动）适合静态角色场景，无需编写 Go 代码
- **Options 模式**：显式构造，不依赖全局注册表
- **双向适配器**：`AuthenticatorFromToken` / `TokenFromAuthenticator`
- **环境变量无前缀**：`SECURITY_ENABLED`、`JWT_PUBLIC_KEY_FILE`（非 `EXASTORE_` 或 `SCALEBOX_`）
- **默认 noop 模式**：未启用安全时注入匿名 admin，所有请求直接放行

## 安装

```bash
go get github.com/kaichao/gopkg/security
```

## 快速开始

### HTTP 服务（推荐）

```go
import "github.com/kaichao/gopkg/security"

// 一行初始化
handler := security.NewHandler(
    security.WithDB(pool),                            // 注入 PG 默认实现
    security.WithPermissionStore(&myPermStore{}),      // 应用定义角色映射
    security.WithSkipPaths("GET /health", "POST /api/auth/login"),
)

mux.Handle("/api/", handler.Middleware(apiMux))
```

### gRPC 服务

```go
mod := security.NewModule(
    security.WithDBModule(pool),
    security.WithPermissionStoreModule(&myPermStore{}),
    security.WithMethodMap(map[string]security.MethodMapping{
        "/app.Service/GetTask": {Resource: "task", Action: "read"},
    }),
)

grpc.NewServer(
    grpc.UnaryInterceptor(mod.UnaryInterceptor()),
    grpc.StreamInterceptor(mod.StreamInterceptor()),
)
```

### 零代码方案（YAML 驱动）

```yaml
# /etc/app/security.yaml
roles:
  admin:
    actions: ["*"]
  operator:
    resources: ["tape", "drive"]
    actions: ["read", "update", "create"]
  viewer:
    actions: ["read"]
```

```go
store, _ := security.NewFilePermissionStore("/etc/app/security.yaml")
handler := security.NewHandler(
    security.WithDB(pool),
    security.WithPermissionStore(store),
)
```

## 核心接口

| 接口 | 说明 |
|------|------|
| `Identity` | 认证后的主体身份（Subject, Name, Roles, Attrs） |
| `TokenAuthenticator` | 协议无关认证（token → Identity） |
| `Authenticator` | gRPC 认证（metadata → Principal） |
| `Authorizer` | 访问控制（Principal → allow/deny） |
| `PermissionStore` | 角色→权限映射，由各应用实现 |
| `KeyStore` | API Key 查询（key hash → Identity） |
| `TokenBlacklist` | Token 撤销检查（jti → blacklisted） |
| `AuditStore` | 审计日志持久化 |

## 内置引擎

| 引擎 | 说明 |
|------|------|
| `JWTVerifier` | Ed25519 验签 + 内存缓存 + 黑名单集成 |
| `JWTSigner` | EdDSA JWT 签发 |
| `APIKeyVerifier` | SHA-256 API Key 验证 |
| `StoreAuthorizer` | PermissionStore → Authorizer 包装 |
| `AuditRecorder` | HTTP 中间件，异步记录 POST/PUT/DELETE |
| `FilePermissionStore` | YAML 驱动，零代码方案 |

## 默认 PG 实现（WithDB 注入）

| 实现 | 表 | 接口 |
|------|----|------|
| `pgTokenBlacklist` | t_token_blacklist | TokenBlacklist |
| `pgKeyStore` | t_api_key + t_user + t_role_binding | KeyStore |
| `pgAuditStore` | t_audit_log | AuditStore |

## 配置

所有环境变量无应用前缀：

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `SECURITY_ENABLED` | `false` | 安全总开关 |
| `JWT_PUBLIC_KEY_FILE` | — | Ed25519 验签公钥文件 |
| `JWT_ISSUER` | — | 期望签发者（空=不校验） |
| `JWT_JWKS_URL` | — | JWKS 端点 |
| `JWT_JWKS_REFRESH` | `3600s` | JWKS 刷新间隔 |
| `JWT_PRIVATE_KEY_FILE` | — | 签发私钥（可选，本地签发用） |
| `TOKEN_SERVICE_URL` | — | Token Service 地址（可选） |
| `SERVICE_KEY` | — | 调用 Token Service 凭证 |
| `TOKEN_SERVICE_TIMEOUT` | `5s` | Token Service 超时 |

## middleware 执行流程

### HTTP

```
请求 → skipPath 检查 → extractToken → AuthenticateToken
     → mapRoute（精确匹配 → 模式匹配 → 路径推断）
     → Authorize → handler
```

- auth 为 nil → 自动注入匿名 admin（noop 模式）
- authz 为 nil → 跳过授权

### gRPC

```
请求 → skipMethod 检查 → metadata 提取 → Authenticate
     → mapMethod → Authorize → handler
     → BillingService.Record（异步）
```

## 编写自定义实现

### PermissionStore（唯一应用定制点）

```go
type MyPermissionStore struct{}

func (s *MyPermissionStore) CheckPermission(ctx context.Context,
    roles []string, resource, action, resourceID string) (bool, error) {
    for _, role := range roles {
        switch role {
        case "admin":
            return true, nil
        case "operator":
            return action == "read" || action == "update", nil
        case "viewer":
            return action == "read", nil
        }
    }
    return false, nil
}
```

### 自定义认证器

```go
type MyAuth struct{}

func (a *MyAuth) AuthenticateToken(ctx context.Context, token string) (security.Identity, error) {
    // 自定义验证逻辑
}

handler := security.NewHandler(
    security.WithAuth(myAuth),
    security.WithAuthorizer(myAuthz),
)
```

也可通过 `SecurityHandlerConfig` 显式构造（向后兼容）。

## 适配器

```go
// 旧 Authenticator → 新 TokenAuthenticator
ta := security.TokenFromAuthenticator(myOldAuth)

// 新 TokenAuthenticator → 旧 Authenticator
a := security.AuthenticatorFromToken(myNewAuth)
```

## 文档

- [Package Documentation](https://pkg.go.dev/github.com/kaichao/gopkg/security)
- [doc.go](./doc.go) — 架构概述
- [CLAUDE.md](./CLAUDE.md) — AI 编码指南

## License

MIT License
