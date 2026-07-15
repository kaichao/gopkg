package security

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc/metadata"
)

// ── TokenAuthenticator → Authenticator ────────────────────────

// tokenAuthAdapter 将 TokenAuthenticator 包装为 Authenticator。
type tokenAuthAdapter struct {
	ta TokenAuthenticator
}

// AuthenticatorFromToken 将 TokenAuthenticator 适配为 Authenticator。
//
// 适配逻辑：从 gRPC metadata 提取 Bearer token → 调用 TokenAuthenticator →
// 将返回的 Identity 断言为 *Principal。
// 如果 Identity 不是 *Principal，返回错误。
func AuthenticatorFromToken(ta TokenAuthenticator) Authenticator {
	return &tokenAuthAdapter{ta: ta}
}

func (a *tokenAuthAdapter) Authenticate(ctx context.Context, md metadata.MD) (*Principal, error) {
	token := extractBearerFromMD(md)
	if token == "" {
		return nil, fmt.Errorf("missing authorization header")
	}

	id, err := a.ta.AuthenticateToken(ctx, token)
	if err != nil {
		return nil, err
	}

	p, ok := id.(*Principal)
	if !ok {
		return nil, fmt.Errorf("token authenticator returned non-Principal identity, use Identity path instead")
	}
	return p, nil
}

func extractBearerFromMD(md metadata.MD) string {
	vals := md.Get("authorization")
	if len(vals) == 0 {
		return ""
	}
	t := vals[0]
	if strings.HasPrefix(t, "Bearer ") {
		return strings.TrimPrefix(t, "Bearer ")
	}
	return t
}

// ── Authenticator → TokenAuthenticator ────────────────────────

// authToTokenAdapter 将 Authenticator 包装为 TokenAuthenticator。
type authToTokenAdapter struct {
	a Authenticator
}

// TokenFromAuthenticator 将 Authenticator 适配为 TokenAuthenticator。
//
// 适配逻辑：构造 gRPC metadata 携带 token → 调用 Authenticator →
// 返回的 *Principal 直接作为 Identity（Principal 已实现 Identity 接口）。
func TokenFromAuthenticator(a Authenticator) TokenAuthenticator {
	return &authToTokenAdapter{a: a}
}

func (a *authToTokenAdapter) AuthenticateToken(ctx context.Context, token string) (Identity, error) {
	md := metadata.Pairs("authorization", "Bearer "+token)
	p, err := a.a.Authenticate(ctx, md)
	if err != nil {
		return nil, err
	}
	return p, nil
}
