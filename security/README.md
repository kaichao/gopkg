# security

[![Go Reference](https://pkg.go.dev/badge/github.com/kaichao/gopkg/security.svg)](https://pkg.go.dev/github.com/kaichao/gopkg/security)

`security` 定义可插拔安全框架的接口、注册表和 gRPC 拦截器，提供认证（AuthN）、授权（AuthZ）、记账（Billing）三个维度的安全能力。

## 特性

- **接口 + 注册表 + 插件**：核心接口定义在 `interface.go`，具体实现通过 `.so` 插件动态加载
- **三件套缺省实现**：`NoopAuthenticator`（匿名 admin）、`NoopAuthorizer`（全部放行）、`NoopBillingService`（不记账）
- **gRPC 原生拦截器**：`UnaryInterceptor` 和 `StreamInterceptor`，自动完成认证→授权→执行→记账流程
- **TLS 配置统一管理**：gRPC TLS 和 PostgreSQL TLS 配置集中在 `SecurityConfig` 中
- **环境变量驱动**：无代码配置，全部通过环境变量控制
- **独立项目设计**：不依赖 scalebox 特有逻辑，环境变量无业务前缀

## 安装

```bash
go get github.com/kaichao/gopkg/security
```

## 快速开始

### 不使用安全（默认）

```go
cfg := security.LoadConfig()
mod, err := security.NewModule(cfg, nil)
// cfg.Enabled == false → mod == nil，跳过拦截器注册
```

### 启用安全

```go
import "github.com/kaichao/gopkg/security"

// 1. 加载插件（可选，不加载则使用 noop 实现）
security.LoadPlugins(true, "/usr/lib/myapp/plugins")

// 2. 加载配置（从环境变量）
cfg := security.LoadConfig()

// 3. 创建安全模块
mod, err := security.NewModule(cfg, map[string]security.MethodMapping{
    "/myapp.UserService/GetUser":  {"user", "read"},
    "/myapp.UserService/ListUsers": {"user", "list"},
})

// 4. 注册 gRPC 拦截器
s := grpc.NewServer(
    grpc.UnaryInterceptor(mod.UnaryInterceptor()),
    grpc.StreamInterceptor(mod.StreamInterceptor()),
)
```

## 核心接口

### Authenticator — 认证

```go
type Authenticator interface {
    Authenticate(ctx context.Context, md metadata.MD) (*Principal, error)
}
```

从 gRPC metadata 中提取凭证并验证，成功返回 `Principal`，失败返回 error。

### Authorizer — 授权

```go
type Authorizer interface {
    Authorize(ctx context.Context, p *Principal, resource, action string) error
}
```

判断 principal 是否有权对 resource 执行 action。允许返回 nil，拒绝返回 error。

### BillingService — 记账

```go
type BillingService interface {
    Record(ctx context.Context, r *UsageRecord) error
}
```

异步记录资源使用事件，不得阻塞调用方。

### Principal — 主体身份

```go
type Principal struct {
    ID              string
    Username        string
    Roles           []string
    AllowedClusters []string
    Attrs           map[string]string
    ExpiresAt       time.Time
}
```

## 配置

| 环境变量 | 默认值 | 说明 |
|----------|--------|------|
| `SECURITY_ENABLED` | `false` | 总开关，false 时 NewModule 返回 nil |
| `SECURITY_PLUGIN_DIR` | — | .so 插件目录 |
| `GRPC_TLS_ENABLED` | `false` | gRPC TLS 开关 |
| `GRPC_TLS_CERT_FILE` | — | gRPC TLS 证书 |
| `GRPC_TLS_KEY_FILE` | — | gRPC TLS 私钥 |
| `GRPC_TLS_CA_FILE` | — | gRPC TLS CA |
| `AUTH_MODE` | `noop` | 认证模式：noop / jwt / oauth2 / external |
| `AUTHZ_MODE` | `noop` | 授权模式：noop / rbac / external |
| `BILLING_MODE` | `noop` | 记账模式：noop / pg / kafka / external |
| `PG_SSLMODE` | `disable` | PostgreSQL TLS 模式 |
| `PG_SSL_CERT_FILE` | — | PG 客户端证书 |
| `PG_SSL_KEY_FILE` | — | PG 客户端私钥 |
| `PG_SSL_CA_FILE` | — | PG CA 证书 |
| `PG_PASSWORD` | — | PG 连接密码 |

完整配置列表见 `SecurityConfig` 结构体。

## 编写插件

实现 `Authenticator` 接口，编译为 `.so`，在 `init()` 中注册：

```go
package main

import "github.com/kaichao/gopkg/security"

func init() {
    security.RegisterAuthenticator("jwt", func(cfg security.AuthConfig) (security.Authenticator, error) {
        return &JWTAuthenticator{...}, nil
    })
}

type JWTAuthenticator struct { ... }

func (a *JWTAuthenticator) Authenticate(ctx context.Context, md metadata.MD) (*security.Principal, error) {
    // 从 md 提取 token，验证，返回 Principal
}
```

编译：

```bash
go build -buildmode=plugin -o jwt.so jwt.go
```

## 拦截器流程

```
请求到达
  │
  ├─ 1. authenticate(ctx)
  │     └─ 从 metadata 提取凭证 → Authenticator.Authenticate()
  │     └─ 注入 Principal 到 context
  │
  ├─ 2. Authorizer.Authorize(ctx, principal, resource, action)
  │     └─ 拒绝 → PermissionDenied
  │
  ├─ 3. handler(ctx, req)
  │
  └─ 4. BillingService.Record()（异步，不影响响应）
```

## 文档

- [Package Documentation](https://pkg.go.dev/github.com/kaichao/gopkg/security)
- [doc.go](./doc.go) — 架构概述
- [CLAUDE.md](./CLAUDE.md) — AI 编码指南

## License

MIT License
