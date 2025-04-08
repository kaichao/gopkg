package exec_test

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/kaichao/gopkg/exec"
	"github.com/stretchr/testify/assert"
)

func TestRunSSH(t *testing.T) {
	config := exec.SSHConfig{
		User: "scalebox",
		Host: "10.255.128.1",
		Port: 22,
		// KeyPath 留空，使用默认 ~/.ssh/id_rsa
	}
	exitCode, stdout, stderr, err := exec.RunSSHCommand(config, "sleepa 10", 5)
	if err != nil {
		fmt.Printf("Error: %v, ExitCode: %d\n", err, exitCode)
	} else {
		fmt.Printf("ExitCode: %d, Stdout: %s, Stderr: %s\n", exitCode, stdout, stderr)
	}
}

func TestRunSSHCommand(t *testing.T) {
	baseConfig := exec.SSHConfig{
		Host: "h11",
		Port: 22,
		User: os.Getenv("USER"),
	}

	// (1) 修复密钥认证测试
	t.Run("key-based auth", func(t *testing.T) {
		config := baseConfig
		config.KeyPath = os.ExpandEnv("$HOME/.ssh/id_rsa")

		code, out, errOut, err := exec.RunSSHCommand(config, "/usr/bin/printf 'SSH_OK'", 3)

		t.Logf("ExitCode: %d", code)
		t.Logf("Stdout: %q", out)
		t.Logf("Stderr: %q", errOut)
		t.Logf("Err: %v", err)

		assert.Equal(t, 0, code)
		assert.Equal(t, "SSH_OK", out)
		assert.Nil(t, err)
	})

	// (2) 增强超时测试
	t.Run("command timeout with process cleanup", func(t *testing.T) {
		config := baseConfig
		config.KeyPath = os.ExpandEnv("$HOME/.ssh/id_rsa")

		// 生成唯一标记
		marker := fmt.Sprintf("MARKER_%d", time.Now().UnixNano())
		cmd := fmt.Sprintf("sleep 10 && echo %s", marker)

		// 执行测试
		start := time.Now()
		code, _, _, err := exec.RunSSHCommand(config, cmd, 2)
		duration := time.Since(start)

		// 验证超时处理
		assert.Equal(t, 124, code)
		assert.ErrorContains(t, err, "timed out")
		assert.True(t, duration < 3*time.Second, "actual duration: %v", duration)

		// 验证进程清理
		assert.Eventually(t, func() bool {
			_, out, _, _ := exec.RunSSHCommand(config,
				fmt.Sprintf("pgrep -f '%s' || echo clean", marker),
				2,
			)
			return strings.Contains(out, "clean")
		}, 3*time.Second, 500*time.Millisecond)
	})

	// (3) 可靠的文件清理验证
	t.Run("resource cleanup verification", func(t *testing.T) {
		config := baseConfig
		config.KeyPath = os.ExpandEnv("$HOME/.ssh/id_rsa")

		uniqueID := fmt.Sprintf("%d", time.Now().UnixNano())
		pidFile := fmt.Sprintf("/tmp/ssh_test_%s", uniqueID)

		// 使用原子操作创建文件
		testCmd := fmt.Sprintf(
			"tmp=$(mktemp) && echo $$ > $tmp && mv $tmp %s && sleep 30",
			pidFile,
		)

		go exec.RunSSHCommand(config, testCmd, 1)

		// 验证文件被清理
		assert.Eventually(t, func() bool {
			code, out, _, _ := exec.RunSSHCommand(config,
				fmt.Sprintf("test -f %s && echo exists || echo missing", pidFile),
				2,
			)
			return code == 0 && strings.Contains(out, "missing")
		}, 5*time.Second, 1*time.Second)
	})
}
