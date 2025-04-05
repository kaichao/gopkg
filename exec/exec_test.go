package exec_test

import (
	"testing"
	"time"

	"github.com/kaichao/gopkg/exec"
	"github.com/stretchr/testify/assert"
)

func TestExecCommandReturnAll(t *testing.T) {
	t.Run("successful command execution", func(t *testing.T) {
		code, out, _, err := exec.ExecCommandReturnAll("echo 'hello world'", 2)
		assert.Equal(t, 0, code)
		assert.Contains(t, out, "hello world")
		assert.Nil(t, err)
	})

	t.Run("command timeout", func(t *testing.T) {
		start := time.Now()
		code, _, _, err := exec.ExecCommandReturnAll("sleep 5", 1)
		duration := time.Since(start)

		assert.Equal(t, 124, code)
		assert.ErrorContains(t, err, "command timed out")
		assert.True(t, duration < 2*time.Second)
	})

	t.Run("invalid command (bash-level error)", func(t *testing.T) {
		code, _, _, err := exec.ExecCommandReturnAll("invalid_command_xyz", 0)
		assert.Equal(t, 127, code) // Bash返回127表示命令未找到
		assert.Nil(t, err)         // 视为正常退出
	})

	t.Run("explicit non-zero exit code", func(t *testing.T) {
		code, _, _, err := exec.ExecCommandReturnAll("sh -c 'exit 5'", 2)
		assert.Equal(t, 5, code) // 确定返回码
		assert.Nil(t, err)
	})

	t.Run("concurrent execution", func(t *testing.T) {
		// ... 原有并发测试代码 ...
	})
}
