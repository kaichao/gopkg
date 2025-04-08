package exec_test

import (
	"fmt"
	"os"
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
	// 需要真实SSH环境的测试用例标记为需要联网
	if os.Getenv("NETWORK_TESTS") != "1" {
		t.Skip("Skipping network-dependent tests")
	}

	baseConfig := exec.SSHConfig{
		Host: "localhost",
		Port: 22,
		User: os.Getenv("USER"),
	}

	t.Run("key-based authentication", func(t *testing.T) {
		config := baseConfig
		config.KeyPath = os.ExpandEnv("$HOME/.ssh/id_rsa")

		code, out, _, err := exec.RunSSHCommand(config, "echo 'ssh success'", 5)
		assert.Equal(t, 0, code)
		assert.Contains(t, out, "ssh success")
		assert.Nil(t, err)
	})

	t.Run("password authentication", func(t *testing.T) {
		config := baseConfig
		config.Password = "your_password" // 替换为测试密码

		code, out, _, err := exec.RunSSHCommand(config, "whoami", 5)
		assert.Equal(t, 0, code)
		assert.Contains(t, out, config.User)
		assert.Nil(t, err)
	})

	t.Run("invalid credentials", func(t *testing.T) {
		config := baseConfig
		config.KeyPath = "/invalid/key/path"

		code, _, _, err := exec.RunSSHCommand(config, "echo test", 2)
		assert.Equal(t, 125, code)
		assert.ErrorContains(t, err, "ssh dial failed")
	})

	t.Run("remote command timeout", func(t *testing.T) {
		config := baseConfig
		config.KeyPath = os.ExpandEnv("$HOME/.ssh/id_rsa")

		start := time.Now()
		code, _, _, err := exec.RunSSHCommand(config, "sleep 10", 2)
		duration := time.Since(start)

		assert.Equal(t, 124, code)
		assert.ErrorContains(t, err, "timed out")
		assert.True(t, duration < 3*time.Second, "should timeout within 3 seconds")
	})

	t.Run("cleanup verification", func(t *testing.T) {
		config := baseConfig
		config.KeyPath = os.ExpandEnv("$HOME/.ssh/id_rsa")

		// 生成唯一PID文件
		uniqueID := time.Now().UnixNano()
		testCmd := fmt.Sprintf("sleep 10; echo $! > /tmp/ssh_test_%d", uniqueID)

		// 启动超时命令
		go exec.RunSSHCommand(config, testCmd, 1)

		// 等待清理完成
		time.Sleep(2 * time.Second)

		// 验证PID文件是否被清理
		checkCmd := fmt.Sprintf("test -f /tmp/ssh_test_%d && echo exists || echo missing", uniqueID)
		code, out, _, _ := exec.RunSSHCommand(config, checkCmd, 2)
		assert.Equal(t, 0, code)
		assert.Contains(t, out, "missing")
	})
}
