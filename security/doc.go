// Package security 定义可插拔安全框架的接口、注册表和缺省实现。
//
// 概述：
// security 包提供认证（Authentication）、授权（Authorization）、记账（Billing）
// 三个核心安全接口。具体实现（如 JWT、OAuth2、RBAC）以 Go plugin (.so) 形式提供，
// 运行时通过 LoadPlugins() 加载并自动注册。
//
// 架构：
//
//	                  ┌────────────────────┐
//	                  │    SecurityConfig   │
//	                  │  (环境变量加载)      │
//	                  └────────┬───────────┘
//	                           │
//	              ┌────────────┼────────────┐
//	              ▼            ▼            ▼
//	        AuthConfig   AuthzConfig   BillConfig
//	              │            │            │
//	              ▼            ▼            ▼
//	       NewAuthenticator  NewAuthorizer  NewBillingService
//	              │            │            │
//	              └────────────┼────────────┘
//	                           ▼
//	                     SecurityModule
//	                    (gRPC 拦截器)
//
// 插件加载流程：
//
//	LoadPlugins(enabled, dir)
//	  → 遍历 dir/*.so
//	  → plugin.Open() 触发 init()
//	  → 插件调用 RegisterAuthenticator / RegisterAuthorizer / RegisterBillingService
//	  → 工厂函数注册到全局注册表
//	  → NewModule() 按 mode 名称查找工厂并创建实例
//
// 基础用法：
//
//	import "github.com/kaichao/gopkg/security"
//
//	// 加载插件
//	security.LoadPlugins(true, "/usr/lib/scalebox/plugins")
//
//	// 加载配置
//	cfg := security.LoadConfig()
//
//	// 创建模块（Enabled=false 时返回 nil）
//	mod, err := security.NewModule(cfg, map[string]security.MethodMapping{
//	    "/myservice.MyService/GetItem": {"item", "read"},
//	    "/myservice.MyService/CreateItem": {"item", "write"},
//	})
//
//	// 注册 gRPC 拦截器
//	if mod != nil {
//	    s := grpc.NewServer(
//	        grpc.UnaryInterceptor(mod.UnaryInterceptor()),
//	        grpc.StreamInterceptor(mod.StreamInterceptor()),
//	    )
//	}
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
// 实现 Authenticator / Authorizer / BillingService 接口，编译为 .so 插件，
// 在 init() 中调用 Register* 函数注册。
//
// 未启用安全时，NewModule 返回 nil —— 调用方应据此跳过拦截器注册。
package security
