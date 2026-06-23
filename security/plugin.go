package security

import (
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"strings"

	"github.com/sirupsen/logrus"
)

// LoadPlugins 遍历指定目录，加载所有 .so 插件文件。
// .so 中的 init() 会自动调用 Register* 函数填充注册表。
// enabled==false 或 dir=="" 时直接返回 nil。
func LoadPlugins(enabled bool, dir string) error {
	if !enabled || dir == "" {
		return nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 目录不存在，静默跳过
		}
		return fmt.Errorf("read plugin dir %s: %w", dir, err)
	}

	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".so") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		_, err := plugin.Open(path)
		if err != nil {
			return fmt.Errorf("plugin.Open %s: %w", path, err)
		}
		logrus.Infof("loaded security plugin: %s", e.Name())
	}

	return nil
}
