// Package security 定义可插拔安全框架的接口、注册表和缺省实现。
//
// 具体认证/授权/记账实现（如 JWT、RBAC、OAuth2）以 Go plugin (.so) 形式提供，
// 运行时通过 plugin.Open() 加载并自动注册到本包的工厂注册表中。
package security

import (
	"context"
	"time"

	"google.golang.org/grpc/metadata"
)

// Principal 代表一次认证通过后的主体身份。
type Principal struct {
	ID              string            // 用户唯一标识（JWT sub claim）
	Username        string            // 人类可读用户名
	Roles           []string          // RBAC 角色列表
	AllowedClusters []string          // 可访问的集群（"*" = 全部）
	Attrs           map[string]string // 扩展属性
	ExpiresAt       time.Time         // 会话过期时间
}

// Authenticator 身份认证接口。
//
// 从 gRPC metadata 中提取凭证并验证，成功返回 Principal，失败返回 error。
// controld 会将 error 转换为 gRPC Unauthenticated 状态码。
type Authenticator interface {
	Authenticate(ctx context.Context, md metadata.MD) (*Principal, error)
}

// Authorizer 访问控制接口。
//
// 判断 principal 是否有权对 resource 执行 action。
// 允许返回 nil，拒绝返回 error（controld 转换为 PermissionDenied）。
type Authorizer interface {
	Authorize(ctx context.Context, p *Principal, resource, action, resourceID string) error
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
