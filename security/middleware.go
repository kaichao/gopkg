package security

import (
	"context"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// ── Context key ──────────────────────────────────────────────

type principalKeyType struct{}

var principalKey = principalKeyType{}

// PrincipalFromContext 从 context 中取出认证后的 Principal。
func PrincipalFromContext(ctx context.Context) *Principal {
	if p, ok := ctx.Value(principalKey).(*Principal); ok {
		return p
	}
	return nil
}

// ── SecurityModule ───────────────────────────────────────────

// MethodMapping 定义 gRPC 方法到 (resource, action, resourceID) 的映射。
// ResourceID 可选，用于 app_id 等细粒度授权（空字符串表示不限定）。
type MethodMapping struct {
	Resource   string
	Action     string
	ResourceID string // 可选，如 "42" (app_id)
}

// SecurityModule 组装认证、授权、记账三个组件，提供 gRPC 拦截器。
type SecurityModule struct {
	auth      Authenticator
	authz     Authorizer
	bill      BillingService
	log       *logrus.Entry
	methodMap map[string]MethodMapping
	blacklist TokenBlacklist
}

// ModuleOption 是创建 SecurityModule 的函数选项。
type ModuleOption func(*moduleOpts)

type moduleOpts struct {
	auth       Authenticator
	authz      Authorizer
	bill       BillingService
	methodMap  map[string]MethodMapping
	blacklist  TokenBlacklist
	auditStore AuditStore
}

// WithAuthenticator 设置自定义认证器。
func WithAuthenticator(a Authenticator) ModuleOption {
	return func(o *moduleOpts) { o.auth = a }
}

// WithAuthorizer 设置自定义授权器。
func WithAuthorizer(a Authorizer) ModuleOption {
	return func(o *moduleOpts) { o.authz = a }
}

// WithBillingService 设置记账服务。
func WithBillingService(b BillingService) ModuleOption {
	return func(o *moduleOpts) { o.bill = b }
}

// WithMethodMap 设置 gRPC 方法到资源的映射表。
func WithMethodMap(m map[string]MethodMapping) ModuleOption {
	return func(o *moduleOpts) { o.methodMap = m }
}

// WithBlacklist 设置 Token 黑名单（用于 JWT 验签后的撤销检查）。
func WithBlacklist(bl TokenBlacklist) ModuleOption {
	return func(o *moduleOpts) { o.blacklist = bl }
}

// WithAuditStore 设置审计日志存储。
func WithAuditStore(as AuditStore) ModuleOption {
	return func(o *moduleOpts) { o.auditStore = as }
}


// NewModule 创建 gRPC SecurityModule（Options 模式）。
//
// 用法：
//
//	mod := security.NewModule(
//	    security.WithDB(pool),
//	    security.WithPermissionStore(&myStore{}),
//	    security.WithMethodMap(methodMap),
//	)
func NewModule(opts ...ModuleOption) *SecurityModule {
	o := &moduleOpts{}
	for _, opt := range opts {
		opt(o)
	}
	if o.methodMap == nil {
		o.methodMap = map[string]MethodMapping{}
	}
	if o.auth == nil {
		o.auth = &NoopAuthenticator{}
	}
	if o.authz == nil {
		o.authz = &NoopAuthorizer{}
	}
	if o.bill == nil {
		o.bill = &NoopBillingService{}
	}

	m := &SecurityModule{
		auth:      o.auth,
		authz:     o.authz,
		bill:      o.bill,
		log:       logrus.WithField("component", "security"),
		methodMap: o.methodMap,
		blacklist: o.blacklist,
	}

	return m
}

// NewModuleWith 使用显式传入的组件创建 SecurityModule。
// 适用于直接构造认证/授权器的场景（向后兼容）。
func NewModuleWith(auth Authenticator, authz Authorizer, methodMap map[string]MethodMapping) *SecurityModule {
	if methodMap == nil {
		methodMap = map[string]MethodMapping{}
	}
	return &SecurityModule{
		auth:      auth,
		authz:     authz,
		bill:      &NoopBillingService{},
		log:       logrus.WithField("component", "security"),
		methodMap: methodMap,
	}
}

// UnaryInterceptor 返回 gRPC UnaryServerInterceptor。
func (m *SecurityModule) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		// 跳过系统 RPC（健康检查、反射）
		if skipAuth(info.FullMethod) {
			return handler(ctx, req)
		}

		// 1. 认证
		principal, err := m.authenticate(ctx)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, err.Error())
		}
		ctx = context.WithValue(ctx, principalKey, principal)

		// 2. 授权
		resource, action, resourceID := m.mapMethod(info.FullMethod)
		if err := m.authz.Authorize(ctx, principal, resource, action, resourceID); err != nil {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}

		// 3. 执行 handler
		resp, err = handler(ctx, req)

		// 4. 记账（异步，不影响响应）
		m.recordUsage(principal, resource, action, err)

		return resp, err
	}
}

// StreamInterceptor 返回 gRPC StreamServerInterceptor。
func (m *SecurityModule) StreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// 跳过系统 RPC
		if skipAuth(info.FullMethod) {
			return handler(srv, ss)
		}

		ctx := ss.Context()

		// 1. 认证
		principal, err := m.authenticate(ctx)
		if err != nil {
			return status.Error(codes.Unauthenticated, err.Error())
		}
		ctx = context.WithValue(ctx, principalKey, principal)

		// 2. 授权
		resource, action, resourceID := m.mapMethod(info.FullMethod)
		if err := m.authz.Authorize(ctx, principal, resource, action, resourceID); err != nil {
			return status.Error(codes.PermissionDenied, err.Error())
		}

		// 3. 包装 stream 并执行
		wrapped := &wrappedStream{
			ServerStream: ss,
			ctx:          ctx,
		}
		handlerErr := handler(srv, wrapped)

		// 4. 记账
		m.recordUsage(principal, resource, action, handlerErr)

		return handlerErr
	}
}

// wrappedStream 覆盖 Context() 返回注入 Principal 的 ctx。
type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedStream) Context() context.Context {
	return w.ctx
}

// ── 内部方法 ─────────────────────────────────────────────────

func (m *SecurityModule) authenticate(ctx context.Context) (*Principal, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	return m.auth.Authenticate(ctx, md)
}

func (m *SecurityModule) recordUsage(principal *Principal, resource, action string, handlerErr error) {
	statusCode := 0
	if handlerErr != nil {
		statusCode = -1
	}

	r := &UsageRecord{
		Resource:  resource,
		Action:    action,
		StartedAt: time.Now(),
	}
	// 尝试提取 user ID
	if principal != nil {
		r.UserID = parseID(principal.ID)
	}

	if err := m.bill.Record(context.Background(), r); err != nil {
		m.log.WithError(err).WithFields(logrus.Fields{
			"resource":    resource,
			"action":      action,
			"status_code": statusCode,
		}).Warn("billing record failed")
	}
}

// ── Method → Resource 映射 ───────────────────────────────────

// mapMethod 查找 fullMethod 的映射，未找到时按路径推断。
func (m *SecurityModule) mapMethod(fullMethod string) (resource, action, resourceID string) {
	if entry, ok := m.methodMap[fullMethod]; ok {
		return entry.Resource, entry.Action, entry.ResourceID
	}
	// 未知方法按 RPC 路径最后一段推断 resource
	parts := strings.Split(fullMethod, "/")
	if len(parts) > 0 {
		return strings.ToLower(parts[len(parts)-1]), "execute", ""
	}
	return "unknown", "execute", ""
}

// ── 系统 RPC 豁免 ────────────────────────────────────────────

// skipAuth 跳过健康检查和反射等系统 RPC 的认证。
var skipMethods = map[string]bool{
	"/grpc.health.v1.Health/Check":                         true,
	"/grpc.health.v1.Health/Watch":                         true,
	"/grpc.reflection.v1alpha.ServerReflection/ServerReflectionInfo": true,
	"/grpc.reflection.v1.ServerReflection/ServerReflectionInfo":      true,
}

func skipAuth(fullMethod string) bool {
	return skipMethods[fullMethod]
}

// ── 辅助函数 ─────────────────────────────────────────────────

// parseID 尝试从字符串 ID 解析为整数，失败返回 0。
func parseID(id string) int {
	if id == "" || id == "anonymous" {
		return 0
	}
	var n int
	for _, c := range id {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		} else {
			return 0
		}
	}
	return n
}
