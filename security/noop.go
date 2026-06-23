package security

import (
	"context"
	"time"

	"google.golang.org/grpc/metadata"
)

// NoopAuthenticator 缺省认证实现：直接放行，返回匿名 admin 身份。
type NoopAuthenticator struct{}

func (a *NoopAuthenticator) Authenticate(ctx context.Context, md metadata.MD) (*Principal, error) {
	return &Principal{
		ID:              "anonymous",
		Username:        "anonymous",
		Roles:           []string{"admin"},
		AllowedClusters: []string{"*"},
		ExpiresAt:       time.Now().Add(24 * time.Hour),
	}, nil
}

// NoopAuthorizer 缺省授权实现：全部放行。
type NoopAuthorizer struct{}

func (a *NoopAuthorizer) Authorize(ctx context.Context, p *Principal, resource, action string) error {
	return nil
}

// NoopBillingService 缺省记账实现：不记录任何使用事件。
type NoopBillingService struct{}

func (b *NoopBillingService) Record(ctx context.Context, r *UsageRecord) error {
	return nil
}
