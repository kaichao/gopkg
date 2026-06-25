# CLAUDE.md - pkg/security

可插拔安全框架的接口定义、注册表、缺省实现与 gRPC 集成层。本包设计为独立开源项目，不依赖 scalebox 特有逻辑。

## 文件职责

| 文件 | 职责 |
|------|------|
| `doc.go` | Package 级别文档，架构图与使用说明 |
| `interface.go` | `Principal`、`Authenticator`、`Authorizer`、`BillingService`、`UsageRecord` 类型定义 |
| `registry.go` | 工厂类型定义 + `Register*` / `New*` / `AvailableModes` 函数 |
| `noop.go` | 缺省空实现（匿名 admin、全部放行、不记账） |
| `plugin.go` | `LoadPlugins(enabled, dir)` 遍历目录加载 .so 文件 |
| `config.go` | `SecurityConfig` + 子配置结构体 + `LoadConfig()` 环境变量读取 + `BuildPGConnectionString()` |
| `middleware.go` | `SecurityModule` + `UnaryServerInterceptor` + `StreamServerInterceptor` |

## 核心类型

### Principal
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

### 模块
- `NewModule(cfg SecurityConfig, methodMap map[string]MethodMapping) (*SecurityModule, error)` — cfg.Enabled==false 时返回 nil; methodMap 为 nil 时自动创建空 map

### 拦截器
- `(*SecurityModule).UnaryInterceptor() grpc.UnaryServerInterceptor` — 认证→授权→执行→记账
- `(*SecurityModule).StreamInterceptor() grpc.StreamServerInterceptor` — 同流程，包装 ServerStream 注入 Principal

### 插件
- `LoadPlugins(enabled bool, dir string) error` — enabled==false 或 dir=="" 直接返回；目录不存在静默跳过

### 查询
- `PrincipalFromContext(ctx context.Context) *Principal` — 从 context 取出 Principal
- `AvailableModes() (auths, authzs, bills []string)` — 已注册模式列表，用于调试

### 注册（供插件 init() 调用）
- `RegisterAuthenticator(name string, fn AuthenticatorFactory)` — 重复注册会 panic
- `RegisterAuthorizer(name string, fn AuthorizerFactory)` — 重复注册会 panic
- `RegisterBillingService(name string, fn BillingServiceFactory)` — 重复注册会 panic

### 工厂
- `NewAuthenticator(mode string, cfg AuthConfig) (Authenticator, error)` — mode 空或 "noop" 返回 NoopAuthenticator
- `NewAuthorizer(mode string, cfg AuthzConfig) (Authorizer, error)` — mode 空或 "noop" 返回 NoopAuthorizer
- `NewBillingService(mode string, cfg BillConfig) (BillingService, error)` — mode 空或 "noop" 返回 NoopBillingService

## 架构模式

**接口 + 注册表 + 插件**：

```
外部 .so 插件 init()
  → security.RegisterAuthenticator("jwt", factory)
  → registry 填充
  → NewModule() 按 mode 创建实例
  → gRPC interceptor 调用
```

## 拦截器执行流程

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

## 独立项目设计

1. **环境变量无前缀**：`SECURITY_ENABLED`、`AUTH_MODE`（非 `SCALEBOX_*`）
2. **方法映射由调用方注入**：`NewModule(cfg, methodMap)` — gRPC 方法到 resource/action 的映射由上层传入
3. **Principal 使用通用 claims**：不假设特定 JWT 结构
4. **Authenticator 接口依赖 `google.golang.org/grpc/metadata.MD`**：包内 middleware.go 同时依赖 gRPC，因此拆包前解耦 metadata 类型无实际收益

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

## 外部依赖

- `google.golang.org/grpc` — gRPC interceptor、metadata、status codes（middleware.go、noop.go、interface.go）
- `github.com/sirupsen/logrus` — 结构化日志（plugin.go、middleware.go）

## 与其他包的关系

本包不依赖 gopkg 内部任何子包。是独立的叶子包。
