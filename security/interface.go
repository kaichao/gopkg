// Package security 定义安全框架的核心接口和数据模型。
package security

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"google.golang.org/grpc/metadata"
)

// ErrPermissionDenied 是授权失败的哨兵错误。
var ErrPermissionDenied = errors.New("permission denied")

// Principal 代表一次认证通过后的主体身份。
type Principal struct {
	ID              string            // 用户唯一标识（JWT sub claim）
	Username        string            // 人类可读用户名
	Roles           []string          // RBAC 角色列表
	AllowedClusters []string          // 可访问的集群（"*" = 全部）
	Attrs           map[string]string // 扩展属性
	ExpiresAt       time.Time         // 会话过期时间
}

// Identity 代表一次认证通过后的主体身份，由各应用定义具体字段。
//
// Principal 结构体实现本接口，保持向后兼容；
// 应用也可以自定义实现以携带专属属性。
type Identity interface {
	Subject() string                // 唯一标识（JWT sub claim）
	Name() string                   // 显示名
	RoleList() []string             // RBAC 角色列表（避免与 Principal.Roles 字段冲突）
	Attr(key string) (string, bool) // 扩展属性，如 "allowed_clusters"、"project_id"
}

// Principal 实现 Identity 接口。
func (p *Principal) Subject() string  { return p.ID }
func (p *Principal) Name() string     { return p.Username }
func (p *Principal) RoleList() []string { return p.Roles }
func (p *Principal) Attr(key string) (string, bool) {
	v, ok := p.Attrs[key]
	return v, ok
}

// Authenticator 身份认证接口。
//
// 从 gRPC metadata 中提取凭证并验证，成功返回 Principal，失败返回 error。
// controld 会将 error 转换为 gRPC Unauthenticated 状态码。
type Authenticator interface {
	Authenticate(ctx context.Context, md metadata.MD) (*Principal, error)
}

// TokenAuthenticator 是协议无关的认证接口。
//
// 接收原始 token 字符串（不含 "Bearer " 前缀），
// 验证成功返回 Identity，失败返回 error。
// HTTP 中间件和 gRPC 拦截器均可使用。
type TokenAuthenticator interface {
	AuthenticateToken(ctx context.Context, token string) (Identity, error)
}

// Authorizer 访问控制接口。
//
// 判断 principal 是否有权对 resource 执行 action。
// 允许返回 nil，拒绝返回 error（controld 转换为 PermissionDenied）。
type Authorizer interface {
	Authorize(ctx context.Context, p *Principal, resource, action, resourceID string) error
}

// PermissionStore 定义角色到权限的映射规则，由各应用实现。
//
// 通用 RBAC 引擎通过此接口查询一组角色对某个资源的操作权限，
// 不关心角色名称和资源类型的具体含义。
// 应用可实现为内存映射、数据库查询、配置文件或外部服务调用。
type PermissionStore interface {
	CheckPermission(ctx context.Context, roles []string, resource, action, resourceID string) (bool, error)
}

// UsageRecord 记录一次资源使用事件，用于记账。
type UsageRecord struct {
	UserID     int
	AppID      int
	Cluster    string
	TaskID     int64
	SlotID     int
	Resource   string
	Action     string
	CPUSeconds float64
	IOBytes    int64
	StartedAt  time.Time
	FinishedAt time.Time
}

// BillingService 记账接口。
//
// Record 异步记录一条使用事件，不得阻塞调用方。实现应采用异步批量写入模式。
type BillingService interface {
	Record(ctx context.Context, r *UsageRecord) error
}

// KeyStore 查询 API Key 对应的用户身份。由各应用实现（查 t_api_key 表）。
type KeyStore interface {
	LookupKey(ctx context.Context, keyHash string) (Identity, error)
}

// TokenBlacklist 检查 JWT 是否已被撤销（jti 在黑名单中）。
type TokenBlacklist interface {
	IsBlacklisted(ctx context.Context, jti string) (bool, error)
}

// AuditEntry 是一条审计记录。
type AuditEntry struct {
	UserID    string
	Action    string
	Resource  string
	Detail    json.RawMessage // t_audit_log.detail 为 JSONB 类型
	Timestamp time.Time
}

// AuditStore 持久化审计记录。由各应用实现（写 t_audit_log 表）。
type AuditStore interface {
	Record(ctx context.Context, entry AuditEntry) error
}
