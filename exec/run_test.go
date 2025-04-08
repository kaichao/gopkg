package exec_test

import (
	"sync"
	"testing"
	"time"

	"github.com/kaichao/gopkg/exec"
	"github.com/stretchr/testify/assert"
)

func TestRunReturnAll(t *testing.T) {
	// 1. 成功执行命令
	t.Run("successful command execution", func(t *testing.T) {
		code, out, _, err := exec.RunReturnAll("echo 'hello world'", 2)
		assert.Equal(t, 0, code)
		assert.Contains(t, out, "hello world")
		assert.Nil(t, err)
	})

	// 2. 命令超时
	t.Run("command timeout", func(t *testing.T) {
		start := time.Now()
		code, _, _, err := exec.RunReturnAll("sleep 5", 1)
		duration := time.Since(start)

		assert.Equal(t, 124, code)
		assert.ErrorContains(t, err, "command timed out")
		assert.True(t, duration < 2*time.Second)
	})

	// 3. 无效命令 (bash-level error)
	t.Run("invalid command (bash-level error)", func(t *testing.T) {
		code, _, _, err := exec.RunReturnAll("invalid_command_xyz", 0)
		assert.Equal(t, 127, code) // Bash 返回 127 表示命令未找到
		assert.Nil(t, err)         // 视为正常退出
	})

	// 4. 显式非零退出码
	t.Run("explicit non-zero exit code", func(t *testing.T) {
		code, _, _, err := exec.RunReturnAll("sh -c 'exit 5'", 2)
		assert.Equal(t, 5, code) // 确定返回码
		assert.Nil(t, err)
	})

	// 5. 命令启动失败 (空命令)
	t.Run("command_start_failure", func(t *testing.T) {
		code, _, _, err := exec.RunReturnAll("", 2)
		assert.Equal(t, 125, code, "expected exit code 125 for empty command")
		assert.NotNil(t, err, "expected a non-nil error")
		assert.Contains(t, err.Error(), "empty command", "error should indicate empty command")
		// Avoid further dereferencing of err without nil check
	})

	// 6. 信号终止
	t.Run("signal termination", func(t *testing.T) {
		code, _, _, err := exec.RunReturnAll("sh -c 'kill -9 $$'", 2) // 使用 $$ 获取当前 shell 的 PID
		assert.Equal(t, 137, code)                                    // SIGKILL 对应退出码 128 + 9 = 137
		assert.Nil(t, err)
	})

	// 7. 多线程安全
	t.Run("concurrent execution", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				code, _, _, err := exec.RunReturnAll("echo 'concurrent test'", 2)
				assert.Equal(t, 0, code)
				assert.Nil(t, err)
			}()
		}
		wg.Wait()
	})

	// 8. 特殊字符命令
	t.Run("special characters in command", func(t *testing.T) {
		code, out, _, err := exec.RunReturnAll("echo 'special chars: !@#$%^&*()'", 2)
		assert.Equal(t, 0, code)
		assert.Contains(t, out, "special chars: !@#$%^&*()")
		assert.Nil(t, err)
	})

	// 9. 超时无输出
	t.Run("timeout with no output", func(t *testing.T) {
		start := time.Now()
		code, out, _, err := exec.RunReturnAll("sleep 5", 1)
		duration := time.Since(start)

		assert.Equal(t, 124, code)
		assert.Empty(t, out)
		assert.ErrorContains(t, err, "command timed out")
		assert.True(t, duration < 2*time.Second)
	})

	// 10. stderr 输出
	t.Run("command with stderr output", func(t *testing.T) {
		code, _, errOut, err := exec.RunReturnAll("sh -c 'echo error >&2'", 2)
		assert.Equal(t, 0, code)
		assert.Contains(t, errOut, "error")
		assert.Nil(t, err)
	})

	// 11. 空命令
	t.Run("empty command", func(t *testing.T) {
		code, _, _, err := exec.RunReturnAll("", 2)
		assert.Equal(t, 125, code)
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "empty command") // Match current behavior
	})

	// 12. 超大输出
	t.Run("large output", func(t *testing.T) {
		// 生成刚好 10MB 的输出（base64 编码后约为 13.33MB，但环形缓冲区会截断）
		// 确保命令在 30 秒内完成，避免触发超时
		code, out, _, err := exec.RunReturnAll("dd if=/dev/zero bs=1M count=8 | base64", 30)
		assert.Equal(t, 0, code)
		assert.True(t, len(out) <= 10*1024*1024) // 验证输出被正确截断
		assert.Nil(t, err)
	})
}
