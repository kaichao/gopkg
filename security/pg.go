package security

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ── PostgreSQL 默认实现 ──────────────────────────────────────

// pgTokenBlacklist 通过 t_token_blacklist 表实现 TokenBlacklist 接口。
type pgTokenBlacklist struct {
	pool *pgxpool.Pool
}

var _ TokenBlacklist = (*pgTokenBlacklist)(nil)

// IsBlacklisted 检查 jti 是否在 t_token_blacklist 中且未过期。
func (b *pgTokenBlacklist) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	var exists bool
	err := b.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM t_token_blacklist WHERE jti = $1 AND expires_at > now())`,
		jti,
	).Scan(&exists)
	return exists, err
}

// pgKeyStore 通过 t_api_key + t_user + t_role_binding 表实现 KeyStore 接口。
type pgKeyStore struct {
	pool *pgxpool.Pool
}

var _ KeyStore = (*pgKeyStore)(nil)

// LookupKey 通过 key_hash 查找 API Key 对应的用户身份。
func (s *pgKeyStore) LookupKey(ctx context.Context, keyHash string) (Identity, error) {
	var userID, username, status string
	err := s.pool.QueryRow(ctx,
		`SELECT u.id, u.name, u.status
		 FROM t_user u
		 JOIN t_api_key k ON k.user_id = u.id
		 WHERE k.key_hash = $1 AND k.expires_at > now()`,
		keyHash,
	).Scan(&userID, &username, &status)
	if err != nil {
		return nil, err
	}
	if status != "ACTIVE" {
		return nil, ErrPermissionDenied
	}

	// 加载角色
	roles := loadRoles(ctx, s.pool, userID)

	return &Principal{
		ID:       userID,
		Username: username,
		Roles:    roles,
	}, nil
}

// pgAuditStore 通过 t_audit_log 表实现 AuditStore 接口。
type pgAuditStore struct {
	pool *pgxpool.Pool
}

var _ AuditStore = (*pgAuditStore)(nil)

// Record 写入一条审计日志。
func (s *pgAuditStore) Record(ctx context.Context, entry AuditEntry) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO t_audit_log (user_id, action, resource, detail, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		entry.UserID, entry.Action, entry.Resource, entry.Detail, entry.Timestamp,
	)
	return err
}

// ── Options ────────────────────────────────────────────────────

// WithDB 注入 pgxpool.Pool，自动设置黑名单、API Key 和审计日志的默认 PG 实现。
func WithDB(pool *pgxpool.Pool) Option {
	return func(o *Options) {
		if pool != nil {
			o.Blacklist = &pgTokenBlacklist{pool: pool}
			o.KeyStore = &pgKeyStore{pool: pool}
			o.AuditStore = &pgAuditStore{pool: pool}
		}
	}
}

// WithPermissionStore 设置权限存储，自动创建 StoreAuthorizer。
func WithPermissionStore(store PermissionStore) Option {
	return func(o *Options) {
		o.Authz = NewStoreAuthorizer(store)
	}
}

// WithDBModule 注入 pgxpool.Pool 到 gRPC SecurityModule（同 WithDB，用于 gRPC 路径）。
func WithDBModule(pool *pgxpool.Pool) ModuleOption {
	return func(o *moduleOpts) {
		if pool != nil {
			o.blacklist = &pgTokenBlacklist{pool: pool}
			o.auditStore = &pgAuditStore{pool: pool}
		}
	}
}

// WithPermissionStoreModule 设置权限存储到 gRPC SecurityModule。
func WithPermissionStoreModule(store PermissionStore) ModuleOption {
	return func(o *moduleOpts) {
		o.authz = NewStoreAuthorizer(store)
	}
}

// ── 辅助 ─────────────────────────────────────────────────────

// RevokeToken 将 token 加入黑名单。
// jti 和 exp 可从 ParseClaims(tokenString) 获取。
func RevokeToken(ctx context.Context, pool *pgxpool.Pool, jti, userID string, expiresAt time.Time) error {
	_, err := pool.Exec(ctx,
		`INSERT INTO t_token_blacklist (jti, user_id, expires_at) VALUES ($1, $2, $3)`,
		jti, userID, expiresAt,
	)
	return err
}

// loadRoles 从 t_role_binding 加载用户角色列表。
func loadRoles(ctx context.Context, pool *pgxpool.Pool, userID string) []string {
	rows, err := pool.Query(ctx,
		`SELECT role FROM t_role_binding WHERE user_id = $1`, userID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var r string
		if err := rows.Scan(&r); err != nil {
			continue
		}
		roles = append(roles, r)
	}
	return roles
}
