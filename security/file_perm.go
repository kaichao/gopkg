package security

import (
	"context"
	"os"
	"sync"

	yaml "gopkg.in/yaml.v3"
)

// ── YAML 驱动权限存储 ─────────────────────────────────────────

// FilePermissionStore 从 YAML 文件加载角色→资源映射规则。
// 实现 PermissionStore 接口，适合静态角色场景，应用无需编写 Go 代码。
//
// YAML 格式：
//
//	roles:
//	  admin:
//	    actions: ["*"]
//	  tape-operator:
//	    resources: ["tape", "drive", "library"]
//	    actions: ["read", "update", "create", "execute"]
//	  viewer:
//	    actions: ["read"]
type FilePermissionStore struct {
	mu     sync.RWMutex
	config permConfig
}

type permConfig struct {
	Roles map[string]roleDef `yaml:"roles"`
}

type roleDef struct {
	Resources []string `yaml:"resources"` // 空=不限资源
	Actions   []string `yaml:"actions"`   // ["*"]=不限操作
}

var _ PermissionStore = (*FilePermissionStore)(nil)

// NewFilePermissionStore 从 YAML 文件创建权限存储。
func NewFilePermissionStore(path string) (*FilePermissionStore, error) {
	s := &FilePermissionStore{}
	if err := s.load(path); err != nil {
		return nil, err
	}
	return s, nil
}

// CheckPermission 判断给定角色是否有权执行操作。
func (s *FilePermissionStore) CheckPermission(
	ctx context.Context,
	roles []string,
	resource, action, resourceID string,
) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, role := range roles {
		if s.match(role, resource, action) {
			return true, nil
		}
	}
	return false, nil
}

// Reload 重新加载权限配置文件（支持热更新）。
func (s *FilePermissionStore) Reload(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.load(path)
}

// ── 内部 ─────────────────────────────────────────────────────

func (s *FilePermissionStore) load(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var cfg permConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return err
	}
	s.config = cfg
	return nil
}

func (s *FilePermissionStore) match(role, resource, action string) bool {
	def, ok := s.config.Roles[role]
	if !ok {
		return false
	}

	// 检查 action
	if !matchAny(def.Actions, action) {
		return false
	}

	// 检查 resource（空列表 = 不限制资源）
	if len(def.Resources) == 0 {
		return true
	}
	return matchAny(def.Resources, resource)
}

func matchAny(allowed []string, value string) bool {
	for _, a := range allowed {
		if a == "*" || a == value {
			return true
		}
	}
	return false
}
