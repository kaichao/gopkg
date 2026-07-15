// Package security 定义可插拔安全框架的接口、注册表和缺省实现。
//
// 概述：
// security 包提供认证（Authentication）、授权（Authorization）、记账（Billing）
// 三个核心安全接口，同时支持 gRPC 和 HTTP 两种协议。
// 具体实现（如 JWT、OAuth2、RBAC）可以 Go plugin (.so) 形式通过 LoadPlugins()
// 动态加载注册，也可以作为普通库显式构造使用。
//
// 架构：
//
//	                  ┌────────────────────┐
//	                  │    SecurityConfig   │
//	                  │  (环境变量加载)      │
//	                  └────────┬───────────┘
//	                           │
//	              ┌────────────┼────────────┬──────────────┐
//	              ▼            ▼            ▼              ▼
//	        AuthConfig   AuthzConfig   BillConfig   TokenAuthConfig
//	              │            │            │              │
//	              ▼            ▼            ▼              ▼
//	       NewAuthenticator  NewAuthorizer NewBillingService NewTokenAuthenticator
//	              │            │            │              │
//	              └────────────┼────────────┘              │
//	                           ▼                           ▼
//	                     SecurityModule              SecurityHandler
//	                    (gRPC 拦截器)              (HTTP 中间件)
//
// 插件加载流程（gRPC 路径，保留）：
//
//	LoadPlugins(enabled, dir)
//	  → 遍历 dir/*.so
//	  → plugin.Open() 触发 init()
//	  → 插件调用 RegisterAuthenticator / RegisterAuthorizer / RegisterBillingService
//	  → 工厂函数注册到全局注册表
//	  → NewModule() 按 mode 名称查找工厂并创建实例
//
// 显式构造（HTTP 路径，推荐）：
//
//	// 直接构造认证器，不依赖全局注册表和插件机制
//	verifier := jwt.NewVerifier(jwt.Config{...})
//	handler := security.NewSecurityHandler(security.SecurityHandlerConfig{
//	    Auth:     verifier,
//	    Authz:    &security.NoopAuthorizer{},
//	    RouteMap: map[string]security.MethodMapping{
//	        "GET /api/tapes":         {Resource: "tape", Action: "read"},
//	        "POST /api/tapes/import": {Resource: "tape", Action: "create"},
//	    },
//	    SkipPaths: []string{"GET /login", "GET /health"},
//	})
//	mux.Handle("/api/", handler.Middleware(apiMux))
//
// 核心接口：
//
//   - Identity：认证后的主体身份接口，Principal 结构体实现此接口
//   - TokenAuthenticator：协议无关的认证接口（替代绑定 metadata.MD 的 Authenticator）
//   - Authenticator：原有 gRPC 认证接口，保留向后兼容
//   - Authorizer：访问控制接口（原有，不变）
//   - PermissionStore：角色→权限映射规则接口，由各应用实现，供通用 RBAC 引擎调用
//   - BillingService：记账接口（原有，不变）
//
// 适配器：
//
//   - AuthenticatorFromToken：将 TokenAuthenticator 包装为 Authenticator（metadata → token → 认证）
//   - TokenFromAuthenticator：将 Authenticator 包装为 TokenAuthenticator（token → metadata → 认证）
//
// 环境变量：
// 本包设计为独立项目，环境变量不使用项目特有前缀：
//
//	SECURITY_ENABLED        — 总开关（false 时 NewModule 返回 nil）
//	SECURITY_PLUGIN_DIR     — .so 插件目录
//	GRPC_TLS_ENABLED        — gRPC TLS 开关
//	AUTH_MODE               — 认证模式（noop / jwt / oauth2 / external）
//	AUTHZ_MODE              — 授权模式（noop / rbac / external）
//	BILLING_MODE            — 记账模式（noop / pg / kafka / external）
//	PG_SSLMODE              — PostgreSQL TLS 模式
//
// 自定义实现：
// 实现 TokenAuthenticator / Authorizer / PermissionStore 接口，
// 通过 SecurityHandlerConfig 显式传入。也可实现 Authenticator / Authorizer /
// BillingService 接口，编译为 .so 插件，在 init() 中调用 Register* 函数注册（向后兼容）。
//
// 未启用安全时，NewModule 返回 nil —— 调用方应据此跳过拦截器注册。
package security
