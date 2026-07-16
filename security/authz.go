package security

import (
	"context"
)

// StoreAuthorizer 是通用 RBAC 授权器，将 PermissionStore 包装为 Authorizer 接口。
//
// 各应用实现自己的 PermissionStore（定义角色→资源映射），
// 通过此授权器接入 gRPC 拦截器或 HTTP 中间件。
//
// 用法：
//
//	store := &MyPermissionStore{}
//	authz := security.NewStoreAuthorizer(store)
//	handler := security.NewSecurityHandler(security.SecurityHandlerConfig{
//	    Authz: authz,
//	    ...
//	})
type StoreAuthorizer struct {
	store PermissionStore
}

// NewStoreAuthorizer 创建通用 RBAC 授权器。
func NewStoreAuthorizer(store PermissionStore) Authorizer {
	return &StoreAuthorizer{store: store}
}

// Authorize 实现 Authorizer 接口：查询 PermissionStore，拒绝时返回 ErrPermissionDenied。
func (a *StoreAuthorizer) Authorize(ctx context.Context, p *Principal, resource, action, resourceID string) error {
	allowed, err := a.store.CheckPermission(ctx, p.Roles, resource, action, resourceID)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrPermissionDenied
	}
	return nil
}
