// pg_store.go — UserStore / RoleStore / APIKeyManager 的 PostgreSQL 默认实现

package security

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	gopkgerrors "github.com/kaichao/gopkg/errors"
)

// ── pgUserStore ───────────────────────────────────────────────

type pgUserStore struct {
	pool *pgxpool.Pool
}

var _ UserStore = (*pgUserStore)(nil)

func (s *pgUserStore) CreateUser(ctx context.Context, name, email, displayName, passwordHash string) (*User, error) {
	u := &User{}
	err := s.pool.QueryRow(ctx,
		`INSERT INTO t_user (name, email, display_name, password_hash, status)
		 VALUES ($1, NULLIF($2,''), NULLIF($3,''), NULLIF($4,''), 'ACTIVE')
		 RETURNING id, name, email, display_name, password_hash, status, created_at, updated_at`,
		name, email, displayName, passwordHash,
	).Scan(&u.ID, &u.Name, &u.Email, &u.DisplayName, &u.PasswordHash, &u.Status, &u.CreatedAt, &u.UpdatedAt)
	return u, wrapErr(err, "UserStore.CreateUser")
}

func (s *pgUserStore) FindUserByID(ctx context.Context, id int) (*User, error) {
	u := &User{}
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, email, display_name, password_hash, status, created_at, updated_at
		 FROM t_user WHERE id = $1`, id,
	).Scan(&u.ID, &u.Name, &u.Email, &u.DisplayName, &u.PasswordHash, &u.Status, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, wrapErr(err, "UserStore.FindByID", "id", id)
	}
	return u, nil
}

func (s *pgUserStore) FindUserByName(ctx context.Context, name string) (*User, error) {
	u := &User{}
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, email, display_name, password_hash, status, created_at, updated_at
		 FROM t_user WHERE name = $1`, name,
	).Scan(&u.ID, &u.Name, &u.Email, &u.DisplayName, &u.PasswordHash, &u.Status, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, wrapErr(err, "UserStore.FindByName", "name", name)
	}
	return u, nil
}

func (s *pgUserStore) ListUsers(ctx context.Context, filter UserFilter) ([]*User, error) {
	query := `SELECT id, name, email, display_name, password_hash, status, created_at, updated_at FROM t_user WHERE 1=1`
	args := []interface{}{}
	idx := 1

	if filter.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", idx)
		args = append(args, *filter.Status)
		idx++
	}
	query += " ORDER BY id"
	if filter.Limit != nil {
		query += fmt.Sprintf(" LIMIT $%d", idx)
		args = append(args, *filter.Limit)
		idx++
	}

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, wrapErr(err, "UserStore.ListUsers")
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		u := &User{}
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.DisplayName, &u.PasswordHash, &u.Status, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, wrapErr(err, "UserStore.ListUsers scan")
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (s *pgUserStore) UpdateUser(ctx context.Context, id int, patch UserPatch) error {
	query := "UPDATE t_user SET updated_at = now()"
	args := []interface{}{}
	idx := 1

	if patch.Name != nil {
		query += fmt.Sprintf(", name = NULLIF($%d,'')", idx)
		args = append(args, *patch.Name)
		idx++
	}
	if patch.Email != nil {
		query += fmt.Sprintf(", email = NULLIF($%d,'')", idx)
		args = append(args, *patch.Email)
		idx++
	}
	if patch.DisplayName != nil {
		query += fmt.Sprintf(", display_name = NULLIF($%d,'')", idx)
		args = append(args, *patch.DisplayName)
		idx++
	}
	if patch.PasswordHash != nil {
		query += fmt.Sprintf(", password_hash = $%d", idx)
		args = append(args, *patch.PasswordHash)
		idx++
	}
	if patch.Status != nil {
		query += fmt.Sprintf(", status = $%d", idx)
		args = append(args, *patch.Status)
		idx++
	}

	query += fmt.Sprintf(" WHERE id = $%d", idx)
	args = append(args, id)

	result, err := s.pool.Exec(ctx, query, args...)
	if err != nil {
		return wrapErr(err, "UserStore.UpdateUser", "id", id)
	}
	if result.RowsAffected() == 0 {
		return wrapErr(sql.ErrNoRows, "UserStore.UpdateUser: no rows", "id", id)
	}
	return nil
}

func (s *pgUserStore) DeleteUser(ctx context.Context, id int) error {
	result, err := s.pool.Exec(ctx, "DELETE FROM t_user WHERE id = $1", id)
	if err != nil {
		return wrapErr(err, "UserStore.DeleteUser", "id", id)
	}
	if result.RowsAffected() == 0 {
		return wrapErr(sql.ErrNoRows, "UserStore.DeleteUser: no rows", "id", id)
	}
	return nil
}

func (s *pgUserStore) SetPassword(ctx context.Context, id int, passwordHash string) error {
	result, err := s.pool.Exec(ctx,
		"UPDATE t_user SET password_hash = $2, updated_at = now() WHERE id = $1",
		id, passwordHash)
	if err != nil {
		return wrapErr(err, "UserStore.SetPassword", "id", id)
	}
	if result.RowsAffected() == 0 {
		return wrapErr(sql.ErrNoRows, "UserStore.SetPassword: no rows", "id", id)
	}
	return nil
}

func (s *pgUserStore) SetStatus(ctx context.Context, id int, status string) error {
	result, err := s.pool.Exec(ctx,
		"UPDATE t_user SET status = $2, updated_at = now() WHERE id = $1",
		id, status)
	if err != nil {
		return wrapErr(err, "UserStore.SetStatus", "id", id)
	}
	if result.RowsAffected() == 0 {
		return wrapErr(sql.ErrNoRows, "UserStore.SetStatus: no rows", "id", id)
	}
	return nil
}

// ── pgRoleStore ───────────────────────────────────────────────

type pgRoleStore struct {
	pool *pgxpool.Pool
}

var _ RoleStore = (*pgRoleStore)(nil)

func (s *pgRoleStore) BindRole(ctx context.Context, userID int, role string, scope json.RawMessage) (*RoleBinding, error) {
	rb := &RoleBinding{}
	err := s.pool.QueryRow(ctx,
		`INSERT INTO t_role_binding (user_id, role, scope)
		 VALUES ($1, $2, $3)
		 RETURNING id, user_id, role, scope, created_at`,
		userID, role, scope,
	).Scan(&rb.ID, &rb.UserID, &rb.Role, &rb.Scope, &rb.CreatedAt)
	if err != nil {
		return nil, wrapErr(err, "RoleStore.BindRole", "user_id", userID, "role", role)
	}
	return rb, nil
}

func (s *pgRoleStore) UnbindRole(ctx context.Context, bindingID int) error {
	result, err := s.pool.Exec(ctx,
		"DELETE FROM t_role_binding WHERE id = $1", bindingID)
	if err != nil {
		return wrapErr(err, "RoleStore.UnbindRole", "id", bindingID)
	}
	if result.RowsAffected() == 0 {
		return wrapErr(sql.ErrNoRows, "RoleStore.UnbindRole: no rows", "id", bindingID)
	}
	return nil
}

func (s *pgRoleStore) ListRoleBindings(ctx context.Context, userID int) ([]*RoleBinding, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, user_id, role, scope, created_at
		 FROM t_role_binding WHERE user_id = $1 ORDER BY id`, userID)
	if err != nil {
		return nil, wrapErr(err, "RoleStore.ListBindings", "user_id", userID)
	}
	defer rows.Close()

	var bindings []*RoleBinding
	for rows.Next() {
		rb := &RoleBinding{}
		if err := rows.Scan(&rb.ID, &rb.UserID, &rb.Role, &rb.Scope, &rb.CreatedAt); err != nil {
			return nil, wrapErr(err, "RoleStore.ListBindings scan")
		}
		bindings = append(bindings, rb)
	}
	return bindings, rows.Err()
}

func (s *pgRoleStore) GetRoles(ctx context.Context, userID int) ([]string, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT role FROM t_role_binding WHERE user_id = $1`, userID)
	if err != nil {
		return nil, wrapErr(err, "RoleStore.GetRoles", "user_id", userID)
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var r string
		if err := rows.Scan(&r); err != nil {
			return nil, wrapErr(err, "RoleStore.GetRoles scan")
		}
		roles = append(roles, r)
	}
	return roles, rows.Err()
}

// ── pgAPIKeyManager ───────────────────────────────────────────

type pgAPIKeyManager struct {
	pool *pgxpool.Pool
}

var _ APIKeyManager = (*pgAPIKeyManager)(nil)

func (s *pgAPIKeyManager) CreateKey(ctx context.Context, userID int, name, keyHash, keyPrefix string,
	scopeType, scopeID *string, expiresAt *time.Time) (*APIKey, error) {

	k := &APIKey{}
	err := s.pool.QueryRow(ctx,
		`INSERT INTO t_api_key (user_id, key_hash, key_prefix, name, scope_type, scope_id, expires_at)
		 VALUES ($1, $2, NULLIF($3,''), $4, $5, $6, $7)
		 RETURNING id, user_id, key_hash, key_prefix, name, scope_type, scope_id,
		           expires_at, last_used_at, created_at, updated_at`,
		userID, keyHash, keyPrefix, name, scopeType, scopeID, expiresAt,
	).Scan(&k.ID, &k.UserID, &k.KeyHash, &k.KeyPrefix, &k.Name,
		&k.ScopeType, &k.ScopeID, &k.ExpiresAt, &k.LastUsedAt, &k.CreatedAt, &k.UpdatedAt)
	if err != nil {
		return nil, wrapErr(err, "APIKeyManager.CreateKey", "user_id", userID)
	}
	return k, nil
}

func (s *pgAPIKeyManager) ListKeys(ctx context.Context, userID int) ([]*APIKey, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, user_id, key_prefix, name, scope_type, scope_id,
		        expires_at, last_used_at, created_at, updated_at
		 FROM t_api_key WHERE user_id = $1 ORDER BY id`, userID)
	if err != nil {
		return nil, wrapErr(err, "APIKeyManager.ListKeys", "user_id", userID)
	}
	defer rows.Close()

	var keys []*APIKey
	for rows.Next() {
		k := &APIKey{UserID: userID}
		if err := rows.Scan(&k.ID, &k.UserID, &k.KeyPrefix, &k.Name,
			&k.ScopeType, &k.ScopeID, &k.ExpiresAt, &k.LastUsedAt, &k.CreatedAt, &k.UpdatedAt); err != nil {
			return nil, wrapErr(err, "APIKeyManager.ListKeys scan")
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

func (s *pgAPIKeyManager) RevokeKey(ctx context.Context, id int) error {
	result, err := s.pool.Exec(ctx, "DELETE FROM t_api_key WHERE id = $1", id)
	if err != nil {
		return wrapErr(err, "APIKeyManager.RevokeKey", "id", id)
	}
	if result.RowsAffected() == 0 {
		return wrapErr(sql.ErrNoRows, "APIKeyManager.RevokeKey: no rows", "id", id)
	}
	return nil
}

// ── 构造函数 ──────────────────────────────────────────────────

// NewPGUserStore 创建 UserStore 的 PostgreSQL 默认实现。
func NewPGUserStore(pool *pgxpool.Pool) UserStore {
	return &pgUserStore{pool: pool}
}

// NewPGRoleStore 创建 RoleStore 的 PostgreSQL 默认实现。
func NewPGRoleStore(pool *pgxpool.Pool) RoleStore {
	return &pgRoleStore{pool: pool}
}

// NewPGAPIKeyManager 创建 APIKeyManager 的 PostgreSQL 默认实现。
func NewPGAPIKeyManager(pool *pgxpool.Pool) APIKeyManager {
	return &pgAPIKeyManager{pool: pool}
}

// ── 辅助 ──────────────────────────────────────────────────────

func wrapErr(err error, args ...any) error {
	if err == nil {
		return nil
	}
	return gopkgerrors.WrapE(err, args...)
}
