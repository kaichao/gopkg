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

// MethodMapping 定义 gRPC 方法到 (resource, action) 的映射。
type MethodMapping struct {
	Resource string
	Action   string
}

// SecurityModule 组装认证、授权、记账三个组件，提供 gRPC 拦截器。
type SecurityModule struct {
	auth      Authenticator
	authz     Authorizer
	bill      BillingService
	log       *logrus.Entry
	methodMap map[string]MethodMapping
}

// NewModule 根据配置创建 SecurityModule。
// cfg.Enabled==false 时返回 nil（不注册拦截器）。
// methodMap 是 gRPC FullMethod → MethodMapping 的映射表，由调用方注入。
func NewModule(cfg SecurityConfig, methodMap map[string]MethodMapping) (*SecurityModule, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	authenticator, err := NewAuthenticator(cfg.AuthMode, cfg.AuthCfg())
	if err != nil {
		return nil, err
	}

	authorizer, err := NewAuthorizer(cfg.AuthZMode, cfg.AuthzCfg())
	if err != nil {
		return nil, err
	}

	billing, err := NewBillingService(cfg.BillingMode, cfg.BillCfg())
	if err != nil {
		return nil, err
	}

	if methodMap == nil {
		methodMap = map[string]MethodMapping{}
	}

	return &SecurityModule{
		auth:      authenticator,
		authz:     authorizer,
		bill:      billing,
		log:       logrus.WithField("component", "security"),
		methodMap: methodMap,
	}, nil
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
		resource, action := m.mapMethod(info.FullMethod)
		if err := m.authz.Authorize(ctx, principal, resource, action); err != nil {
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
		resource, action := m.mapMethod(info.FullMethod)
		if err := m.authz.Authorize(ctx, principal, resource, action); err != nil {
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
func (m *SecurityModule) mapMethod(fullMethod string) (resource, action string) {
	if entry, ok := m.methodMap[fullMethod]; ok {
		return entry.Resource, entry.Action
	}
	// 未知方法按 RPC 路径最后一段推断 resource
	parts := strings.Split(fullMethod, "/")
	if len(parts) > 0 {
		return strings.ToLower(parts[len(parts)-1]), "execute"
	}
	return "unknown", "execute"
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
