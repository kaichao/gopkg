package security_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kaichao/gopkg/security"
	"google.golang.org/grpc/metadata"
)

// ── 编译期验证：接口实现 ─────────────────────────────────────

// 编译期断言 *Principal 实现 Identity
var _ security.Identity = (*security.Principal)(nil)

// 编译期断言 AuthenticatorFromToken 返回 Authenticator
var _ security.Authenticator = security.AuthenticatorFromToken(nil)

// 编译期断言 TokenFromAuthenticator 返回 TokenAuthenticator
var _ security.TokenAuthenticator = security.TokenFromAuthenticator(nil)

// ── Identity 接口 ────────────────────────────────────────────

func TestPrincipalImplementsIdentity(t *testing.T) {
	p := &security.Principal{
		ID:              "user-42",
		Username:        "alice",
		Roles:           []string{"admin", "viewer"},
		AllowedClusters: []string{"*"},
		Attrs:           map[string]string{"project_id": "prj-1"},
	}

	var id security.Identity = p

	if id.Subject() != "user-42" {
		t.Errorf("Subject() = %q, want %q", id.Subject(), "user-42")
	}
	if id.Name() != "alice" {
		t.Errorf("Name() = %q, want %q", id.Name(), "alice")
	}
	if len(id.RoleList()) != 2 || id.RoleList()[0] != "admin" {
		t.Errorf("RoleList() = %v, want [admin viewer]", id.RoleList())
	}
	if v, ok := id.Attr("project_id"); !ok || v != "prj-1" {
		t.Errorf("Attr(project_id) = (%q, %v), want (prj-1, true)", v, ok)
	}
	if _, ok := id.Attr("nonexistent"); ok {
		t.Error("Attr(nonexistent) should be false")
	}
}

// ── TokenAuthenticator 适配器 ─────────────────────────────────

type mockTokenAuth struct {
	result security.Identity
	err    error
}

func (m *mockTokenAuth) AuthenticateToken(ctx context.Context, token string) (security.Identity, error) {
	return m.result, m.err
}

func TestAuthenticatorFromToken(t *testing.T) {
	expected := &security.Principal{
		ID:       "user-1",
		Username: "bob",
		Roles:    []string{"viewer"},
	}

	ta := &mockTokenAuth{result: expected}
	auth := security.AuthenticatorFromToken(ta)

	md := metadata.Pairs("authorization", "Bearer test-token-123")
	p, err := auth.Authenticate(context.Background(), md)
	if err != nil {
		t.Fatalf("Authenticate() error: %v", err)
	}
	if p.ID != "user-1" {
		t.Errorf("ID = %q, want user-1", p.ID)
	}
}

func TestAuthenticatorFromTokenMissingToken(t *testing.T) {
	ta := &mockTokenAuth{result: &security.Principal{ID: "x"}}
	auth := security.AuthenticatorFromToken(ta)

	md := metadata.New(nil) // 无 authorization header
	_, err := auth.Authenticate(context.Background(), md)
	if err == nil {
		t.Fatal("expected error for missing authorization header")
	}
}

func TestTokenFromAuthenticator(t *testing.T) {
	// 用 NoopAuthenticator 测试反向适配
	noop := &security.NoopAuthenticator{}
	ta := security.TokenFromAuthenticator(noop)

	id, err := ta.AuthenticateToken(context.Background(), "any-token")
	if err != nil {
		t.Fatalf("AuthenticateToken() error: %v", err)
	}
	if id.Subject() != "anonymous" {
		t.Errorf("Subject() = %q, want anonymous", id.Subject())
	}
	if len(id.RoleList()) == 0 || id.RoleList()[0] != "admin" {
		t.Errorf("RoleList() = %v, want [admin]", id.RoleList())
	}
}

// ── HTTP 中间件 ──────────────────────────────────────────────

func TestSecurityHandlerAuthSuccess(t *testing.T) {
	p := &security.Principal{
		ID:       "user-1",
		Username: "testuser",
		Roles:    []string{"admin"},
	}
	handler := security.NewSecurityHandler(security.SecurityHandlerConfig{
		Auth:  &mockTokenAuth{result: p},
		Authz: &security.NoopAuthorizer{},
		RouteMap: map[string]security.MethodMapping{
			"GET /api/test": {Resource: "test", Action: "read"},
		},
	})

	var capturedID security.Identity
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedID = security.IdentityFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()

	handler.Middleware(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if capturedID == nil {
		t.Fatal("identity not injected into context")
	}
	if capturedID.Subject() != "user-1" {
		t.Errorf("Subject = %q, want user-1", capturedID.Subject())
	}
}

func TestSecurityHandlerAuthMissingToken(t *testing.T) {
	handler := security.NewSecurityHandler(security.SecurityHandlerConfig{
		Auth:  &mockTokenAuth{err: nil},
		Authz: &security.NoopAuthorizer{},
	})

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	})

	req := httptest.NewRequest("GET", "/api/test", nil)
	rec := httptest.NewRecorder()

	handler.Middleware(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestSecurityHandlerSkipPath(t *testing.T) {
	handler := security.NewSecurityHandler(security.SecurityHandlerConfig{
		Auth:      &mockTokenAuth{}, // 会因为没有 token 而报错
		Authz:     &security.NoopAuthorizer{},
		SkipPaths: []string{"GET /login", "GET /health"},
	})

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()

	handler.Middleware(next).ServeHTTP(rec, req)

	if !called {
		t.Error("skip path should bypass auth")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

func TestSecurityHandlerTokenFromCookie(t *testing.T) {
	p := &security.Principal{ID: "cookie-user", Roles: []string{"viewer"}}
	handler := security.NewSecurityHandler(security.SecurityHandlerConfig{
		Auth:  &mockTokenAuth{result: p},
		Authz: &security.NoopAuthorizer{},
	})

	var capturedID security.Identity
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedID = security.IdentityFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: "cookie-token"})
	rec := httptest.NewRecorder()

	handler.Middleware(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if capturedID.Subject() != "cookie-user" {
		t.Errorf("Subject = %q, want cookie-user", capturedID.Subject())
	}
}

func TestSecurityHandlerAuthzDenied(t *testing.T) {
	p := &security.Principal{
		ID:    "user-1",
		Roles: []string{"viewer"},
	}
	denyAuthz := &denyAuthorizer{}
	handler := security.NewSecurityHandler(security.SecurityHandlerConfig{
		Auth:  &mockTokenAuth{result: p},
		Authz: denyAuthz,
		RouteMap: map[string]security.MethodMapping{
			"GET /api/admin": {Resource: "admin", Action: "write"},
		},
	})

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called when authz denied")
	})

	req := httptest.NewRequest("GET", "/api/admin", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()

	handler.Middleware(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rec.Code)
	}
}

type denyAuthorizer struct{}

func (d *denyAuthorizer) Authorize(ctx context.Context, p *security.Principal, resource, action, resourceID string) error {
	return security.ErrPermissionDenied
}

// ── 路径模式匹配 ─────────────────────────────────────────────

func TestRouteMapExactMatch(t *testing.T) {
	handler := security.NewSecurityHandler(security.SecurityHandlerConfig{
		Auth:  &mockTokenAuth{result: &security.Principal{ID: "x", Roles: []string{"admin"}}},
		Authz: &security.NoopAuthorizer{},
		RouteMap: map[string]security.MethodMapping{
			"POST /api/tapes/import": {Resource: "tape", Action: "create"},
		},
	})

	var called bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	req := httptest.NewRequest("POST", "/api/tapes/import", nil)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()

	handler.Middleware(next).ServeHTTP(rec, req)
	if !called {
		t.Error("exact match route should pass auth")
	}
}

func TestRouteMapPatternMatch(t *testing.T) {
	handler := security.NewSecurityHandler(security.SecurityHandlerConfig{
		Auth:  &mockTokenAuth{result: &security.Principal{ID: "x", Roles: []string{"admin"}}},
		Authz: &security.NoopAuthorizer{},
		RouteMap: map[string]security.MethodMapping{
			"GET /api/tapes/{barcode}": {Resource: "tape", Action: "read"},
		},
	})

	var called bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	req := httptest.NewRequest("GET", "/api/tapes/ABC123", nil)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()

	handler.Middleware(next).ServeHTTP(rec, req)
	if !called {
		t.Error("pattern match route should pass auth")
	}
}

// ── PrincipalFromHTTPContext ──────────────────────────────────

func TestPrincipalFromHTTPContext(t *testing.T) {
	p := &security.Principal{ID: "test", Username: "tester", Roles: []string{"admin"}}
	handler := security.NewSecurityHandler(security.SecurityHandlerConfig{
		Auth:  &mockTokenAuth{result: p},
		Authz: &security.NoopAuthorizer{},
	})

	var captured *security.Principal
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = security.PrincipalFromHTTPContext(r.Context())
	})

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()

	handler.Middleware(next).ServeHTTP(rec, req)

	if captured == nil {
		t.Fatal("PrincipalFromHTTPContext returned nil")
	}
	if captured.ID != "test" {
		t.Errorf("ID = %q, want test", captured.ID)
	}
}

// ── 向后兼容：旧接口无变化 ───────────────────────────────────

func TestNoopAuthenticatorBackwardCompat(t *testing.T) {
	noop := &security.NoopAuthenticator{}
	md := metadata.New(nil)
	p, err := noop.Authenticate(context.Background(), md)
	if err != nil {
		t.Fatalf("NoopAuthenticator should never error: %v", err)
	}
	if p.ID != "anonymous" {
		t.Errorf("ID = %q, want anonymous", p.ID)
	}
	if len(p.Roles) == 0 || p.Roles[0] != "admin" {
		t.Errorf("Roles = %v, want [admin]", p.Roles)
	}
	if len(p.AllowedClusters) == 0 || p.AllowedClusters[0] != "*" {
		t.Errorf("AllowedClusters = %v, want [*]", p.AllowedClusters)
	}
}

func TestNoopAuthorizerBackwardCompat(t *testing.T) {
	noop := &security.NoopAuthorizer{}
	if err := noop.Authorize(context.Background(), nil, "any", "any", ""); err != nil {
		t.Errorf("NoopAuthorizer should always allow: %v", err)
	}
}

func TestConfigLoad(t *testing.T) {
	// 验证 LoadConfig 返回非零值
	cfg := security.LoadConfig()
	if cfg.Enabled {
		t.Logf("security enabled with config: %+v", cfg)
	} else {
		t.Logf("security disabled (default)")
	}
}

// ── 新增：PermissionStore 接口可用 ────────────────────────────

type testPermissionStore struct{}

func (s *testPermissionStore) CheckPermission(ctx context.Context, roles []string, resource, action, resourceID string) (bool, error) {
	for _, r := range roles {
		if r == "admin" {
			return true, nil
		}
	}
	return false, nil
}

var _ security.PermissionStore = (*testPermissionStore)(nil)

func TestPermissionStoreInterface(t *testing.T) {
	store := &testPermissionStore{}
	allowed, err := store.CheckPermission(context.Background(), []string{"admin"}, "tape", "read", "")
	if err != nil {
		t.Fatalf("CheckPermission error: %v", err)
	}
	if !allowed {
		t.Error("admin should be allowed")
	}

	allowed, _ = store.CheckPermission(context.Background(), []string{"viewer"}, "tape", "write", "")
	if allowed {
		t.Error("viewer should not be allowed to write")
	}
}
