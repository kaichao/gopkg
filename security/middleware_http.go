package security

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
)

// ── Context key ──────────────────────────────────────────────

type identityKeyType struct{}

var identityKey = identityKeyType{}

// IdentityFromContext 从 HTTP context 中取出认证后的 Identity。
func IdentityFromContext(ctx context.Context) Identity {
	if id, ok := ctx.Value(identityKey).(Identity); ok {
		return id
	}
	return nil
}

// PrincipalFromHTTPContext 从 HTTP context 中取出 *Principal。
// 若 Identity 不是 *Principal，返回 nil。
func PrincipalFromHTTPContext(ctx context.Context) *Principal {
	id := IdentityFromContext(ctx)
	if id == nil {
		return nil
	}
	p, _ := id.(*Principal)
	return p
}

// ── SecurityHandler ───────────────────────────────────────────

// SecurityHandler 是 HTTP 版本的认证+授权中间件，与 SecurityModule（gRPC）平级。
type SecurityHandler struct {
	auth     TokenAuthenticator
	authz    Authorizer
	routeMap map[string]MethodMapping
	skipSet  map[string]bool // METHOD /path → 跳过认证
	log      *logrus.Entry
}

// SecurityHandlerConfig 构造 SecurityHandler 所需的参数。
type SecurityHandlerConfig struct {
	Auth     TokenAuthenticator
	Authz    Authorizer
	RouteMap map[string]MethodMapping
	SkipPaths []string // "METHOD /path" 格式，跳过认证（如登录、健康检查、静态文件）
}

// NewSecurityHandler 显式构造 HTTP 安全中间件，不依赖全局注册表。
func NewSecurityHandler(cfg SecurityHandlerConfig) *SecurityHandler {
	skipSet := make(map[string]bool, len(cfg.SkipPaths))
	for _, p := range cfg.SkipPaths {
		skipSet[p] = true
	}
	return &SecurityHandler{
		auth:     cfg.Auth,
		authz:    cfg.Authz,
		routeMap: cfg.RouteMap,
		skipSet:  skipSet,
		log:      logrus.WithField("component", "security-http"),
	}
}

// Middleware 返回标准 http.Handler 中间件。
func (h *SecurityHandler) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methodPath := r.Method + " " + r.URL.Path

		// 跳过认证的路径
		if h.skipSet[methodPath] {
			next.ServeHTTP(w, r)
			return
		}

		// 1. 认证（auth 为 nil 时使用匿名身份，等价于 noop 模式）
		identity, err := h.authenticate(r)
		if err != nil {
			h.log.WithError(err).WithField("path", methodPath).Debug("auth failed")
			http.Error(w, `{"error":"authentication failed: `+err.Error()+`"}`, http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), identityKey, identity)
		r = r.WithContext(ctx)

		// 2. 授权
		resource, action, resourceID := h.mapRoute(methodPath)
		if h.authz != nil {
			principal := identityToPrincipal(identity)
			if err := h.authz.Authorize(ctx, principal, resource, action, resourceID); err != nil {
				h.log.WithError(err).WithFields(logrus.Fields{
					"path":     methodPath,
					"resource": resource,
					"action":   action,
				}).Debug("authz denied")
				http.Error(w, `{"error":"permission denied: `+err.Error()+`"}`, http.StatusForbidden)
				return
			}
		}

		// 3. 执行 handler
		next.ServeHTTP(w, r)
	})
}

// ── 内部方法 ─────────────────────────────────────────────────

func (h *SecurityHandler) authenticate(r *http.Request) (Identity, error) {
	// 未配置认证器：返回匿名身份，等价于 noop 模式
	if h.auth == nil {
		return &Principal{
			ID:              "anonymous",
			Username:        "anonymous",
			Roles:           []string{"admin"},
			AllowedClusters: []string{"*"},
		}, nil
	}

	token := extractToken(r)
	if token == "" {
		return nil, fmt.Errorf("missing authorization header or cookie")
	}
	return h.auth.AuthenticateToken(r.Context(), token)
}

// mapRoute 查找 HTTP method+path 的映射，未找到时按路径推断。
func (h *SecurityHandler) mapRoute(methodPath string) (resource, action, resourceID string) {
	// 精确匹配
	if entry, ok := h.routeMap[methodPath]; ok {
		return entry.Resource, entry.Action, entry.ResourceID
	}

	// 模式匹配（支持 {name} 通配符）
	method, path := splitMethodPath(methodPath)
	if method != "" {
		for pattern, entry := range h.routeMap {
			pm, pp := splitMethodPath(pattern)
			if pm == method && matchPattern(pp, path) {
				return entry.Resource, entry.Action, entry.ResourceID
			}
		}
	}

	// 未找到映射：按路径最后一段推断 resource
	if path != "" {
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) > 0 {
			return strings.ToLower(parts[0]), "execute", ""
		}
	}
	return "unknown", "execute", ""
}

// ── Token 提取 ────────────────────────────────────────────────

func extractToken(r *http.Request) string {
	// Authorization header
	if ah := r.Header.Get("Authorization"); ah != "" {
		if strings.HasPrefix(ah, "Bearer ") {
			return strings.TrimPrefix(ah, "Bearer ")
		}
		return ah
	}

	// Cookie
	if cookie, err := r.Cookie("access_token"); err == nil && cookie.Value != "" {
		return cookie.Value
	}

	return ""
}

// ── Identity → Principal 转换 ─────────────────────────────────

func identityToPrincipal(id Identity) *Principal {
	if p, ok := id.(*Principal); ok {
		return p
	}
	// 非 Principal 的 Identity：构造临时 Principal
	allowed := []string{"*"}
	if v, ok := id.Attr("allowed_clusters"); ok {
		allowed = strings.Split(v, ",")
	}
	attrs := map[string]string{}
	// 遍历常见 key 填充 attrs（无法枚举所有 key，仅做尽力而为的转换）
	for _, k := range []string{"allowed_clusters", "project_id", "email"} {
		if v, ok := id.Attr(k); ok {
			attrs[k] = v
		}
	}
	return &Principal{
		ID:              id.Subject(),
		Username:        id.Name(),
		Roles:           id.RoleList(),
		AllowedClusters: allowed,
		Attrs:           attrs,
	}
}

// ── 路径模式匹配 ──────────────────────────────────────────────

func splitMethodPath(s string) (method, path string) {
	parts := strings.SplitN(s, " ", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", s
}

// matchPattern 检查实际路径是否匹配包含 {name} 通配符的模式。
// 例：模式 "/api/tapes/{barcode}" 匹配实际路径 "/api/tapes/ABC123"。
func matchPattern(pattern, actual string) bool {
	pSegs := strings.Split(strings.Trim(pattern, "/"), "/")
	aSegs := strings.Split(strings.Trim(actual, "/"), "/")

	if len(pSegs) != len(aSegs) {
		return false
	}

	for i, ps := range pSegs {
		if strings.HasPrefix(ps, "{") && strings.HasSuffix(ps, "}") {
			continue // 通配符段匹配任意值
		}
		if ps != aSegs[i] {
			return false
		}
	}
	return true
}
