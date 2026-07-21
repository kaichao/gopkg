// store.go — 安全管理接口定义与数据类型
//
// 在 gopkg/security 已有的读侧接口（TokenBlacklist、KeyStore、AuditStore）之上，
// 新增管理侧 CRUD 接口。各接口均有默认 PG 实现，应用可通过 Options 注入自定义实现。

package security

import (
	"context"
	"encoding/json"
	"time"
)

// ── 数据类型 ──────────────────────────────────────────────────

// User 表示用户记录。
type User struct {
	ID           int        // SERIAL
	Name         string     // 登录名（唯一）
	Email        *string    // 可选
	DisplayName  *string    // 可选
	PasswordHash *string    // bcrypt 哈希
	Status       string     // ACTIVE / DISABLED
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// UserFilter 用户列表筛选条件。
type UserFilter struct {
	Status *string
	Limit  *int
}

// UserPatch 用户字段增量更新，仅更新非 nil 字段。
type UserPatch struct {
	Name         *string
	Email        *string
	DisplayName  *string
	PasswordHash *string
	Status       *string
}

// RoleBinding 角色绑定记录。
type RoleBinding struct {
	ID        int              // SERIAL
	UserID    int              // → t_user.id
	Role      string
	Scope     json.RawMessage  // JSONB 数组，nil = 全局
	CreatedAt time.Time
}

// ScopeItem 是 scope JSONB 数组中的单个约束项。
type ScopeItem struct {
	Type string `json:"type"` // "cluster" / "app" / "project"
	ID   string `json:"id"`
}

// APIKey 表示一条 API 密钥记录。
type APIKey struct {
	ID         int        // SERIAL
	UserID     int        // → t_user.id
	KeyHash    string
	KeyPrefix  string
	Name       string
	ScopeType  *string    // "project" / "app" / nil
	ScopeID    *string    // 对应的实体 ID / nil
	ExpiresAt  *time.Time
	LastUsedAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// ── 管理接口 ──────────────────────────────────────────────────

// UserStore 用户 CRUD。
type UserStore interface {
	// CreateUser 创建用户，status 默认 ACTIVE。
	CreateUser(ctx context.Context, name, email, displayName, passwordHash string) (*User, error)

	// FindUserByID 按主键查找。
	FindUserByID(ctx context.Context, id int) (*User, error)

	// FindUserByName 按登录名查找。
	FindUserByName(ctx context.Context, name string) (*User, error)

	// ListUsers 分页查询用户列表。
	ListUsers(ctx context.Context, filter UserFilter) ([]*User, error)

	// UpdateUser 增量更新用户字段（仅更新非 nil 字段）。
	UpdateUser(ctx context.Context, id int, patch UserPatch) error

	// DeleteUser 删除用户。
	DeleteUser(ctx context.Context, id int) error

	// SetPassword 修改密码哈希。
	SetPassword(ctx context.Context, id int, passwordHash string) error

	// SetStatus 修改用户状态。
	SetStatus(ctx context.Context, id int, status string) error
}

// RoleStore 角色绑定 CRUD。
type RoleStore interface {
	// BindRole 为用户绑定角色。scope 为 nil 时表示全局角色。
	BindRole(ctx context.Context, userID int, role string, scope json.RawMessage) (*RoleBinding, error)

	// UnbindRole 按绑定 ID 移除角色。
	UnbindRole(ctx context.Context, bindingID int) error

	// ListRoleBindings 列出用户的所有角色绑定（含 scope）。
	ListRoleBindings(ctx context.Context, userID int) ([]*RoleBinding, error)

	// GetRoles 仅返回角色名列表（供内部授权检查使用，无网络开销）。
	GetRoles(ctx context.Context, userID int) ([]string, error)
}

// APIKeyManager API 密钥 CRUD。
// 与 KeyStore（读侧 LookupKey）分离：KeyStore 用于认证中间件，APIKeyManager 用于管理面。
type APIKeyManager interface {
	// CreateKey 创建 API 密钥。scopeType/scopeID 为 nil 时不限定范围。
	CreateKey(ctx context.Context, userID int, name, keyHash, keyPrefix string,
		scopeType, scopeID *string, expiresAt *time.Time) (*APIKey, error)

	// ListKeys 列出用户的所有 API 密钥（不含 key_hash，仅元数据）。
	ListKeys(ctx context.Context, userID int) ([]*APIKey, error)

	// RevokeKey 按 ID 删除密钥。
	RevokeKey(ctx context.Context, id int) error
}
