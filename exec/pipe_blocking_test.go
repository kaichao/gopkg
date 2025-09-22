package exec

import (
	"strings"
	"testing"
	"time"
)

// TestPipeBlockingFix 专门测试管道阻塞问题的修复
func TestPipeBlockingFix(t *testing.T) {
	config := SSHConfig{
		User:     testSSHUser,
		Host:     testSSHServer,
		Port:     testSSHPort,
		KeyPath:  testSSHKey,
		Password: testPassword,
	}

	// 检查认证配置
	if config.KeyPath == "" && config.Password == "" {
		t.Skip("SSH authentication not configured: must set either KeyPath or Password")
	}

	tests := []struct {
		name        string
		command     string
		timeout     int
		background  bool
		expectCode  int
		description string
	}{
		{
			name:        "non-background command with large output and timeout",
			command:     "for i in {1..1000}; do echo 'test output line $i'; done; sleep 10",
			timeout:     2,
			background:  false,
			expectCode:  124, // 超时退出码
			description: "测试非后台命令在大输出情况下超时时的管道阻塞问题",
		},
		{
			name:        "background command with timeout during startup",
			command:     "sleep 60", // 长时间运行的命令
			timeout:     2,
			background:  true,
			expectCode:  0, // 后台命令启动应该成功
			description: "测试后台命令启动过程中的超时处理",
		},
		{
			name:        "command with stderr output and timeout",
			command:     "for i in {1..100}; do echo 'error output $i' >&2; done; sleep 10",
			timeout:     2,
			background:  false,
			expectCode:  124,
			description: "测试stderr输出在超时情况下的管道处理",
		},
		{
			name:        "mixed stdout/stderr with timeout",
			command:     "for i in {1..50}; do echo 'stdout $i'; echo 'stderr $i' >&2; done; sleep 10",
			timeout:     2,
			background:  false,
			expectCode:  124,
			description: "测试混合stdout/stderr输出在超时情况下的管道处理",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)

			config.Background = tt.background
			start := time.Now()

			code, stdout, stderr, err := RunSSHCommand(config, tt.command, tt.timeout)
			duration := time.Since(start)

			// 验证超时时间控制
			if tt.expectCode == 124 {
				if duration > time.Duration(tt.timeout+1)*time.Second {
					t.Errorf("Timeout handling took too long: %v (expected < %ds)", duration, tt.timeout+1)
				}
			}

			// 验证退出码
			if code != tt.expectCode {
				t.Errorf("Expected exit code %d, got %d", tt.expectCode, code)
			}

			// 验证错误信息
			if tt.expectCode == 124 {
				if err == nil || !strings.Contains(err.Error(), "timed out") {
					t.Errorf("Expected timeout error, got: %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}

			// 验证输出处理（不应该有管道阻塞）
			if tt.background {
				// 后台命令应该返回PID
				if stdout == "" || stdout == "0" {
					t.Errorf("Background command should return PID, got: %s", stdout)
				}
			} else {
				// 非后台命令应该有部分输出（即使超时）
				if tt.expectCode == 124 {
					// 超时情况下，应该已经捕获了部分输出
					if len(stdout) == 0 && len(stderr) == 0 {
						t.Log("Warning: No output captured during timeout (this might indicate pipe blocking)")
					}
				}
			}

			t.Logf("Test completed in %v, code=%d, stdout_len=%d, stderr_len=%d",
				duration, code, len(stdout), len(stderr))
		})
	}

	// 清理测试过程中可能创建的进程
	cleanupCmd := "pkill -9 -f 'sleep'; pkill -9 -f 'MARKER_'"
	RunSSHCommand(config, cleanupCmd, 5)
}

// TestResourceCleanupOnTimeout 专门测试超时时的资源清理
func TestResourceCleanupOnTimeout(t *testing.T) {
	config := SSHConfig{
		User:     testSSHUser,
		Host:     testSSHServer,
		Port:     testSSHPort,
		KeyPath:  testSSHKey,
		Password: testPassword,
	}

	if config.KeyPath == "" && config.Password == "" {
		t.Skip("SSH authentication not configured: must set either KeyPath or Password")
	}

	// 测试1: 非后台命令超时资源清理
	t.Run("non-background timeout cleanup", func(t *testing.T) {
		cmd := "sleep 30"
		code, _, _, err := RunSSHCommand(config, cmd, 2)

		if code != 124 {
			t.Errorf("Expected timeout code 124, got %d", code)
		}
		if err == nil || !strings.Contains(err.Error(), "timed out") {
			t.Errorf("Expected timeout error, got %v", err)
		}

		// 验证没有残留的sleep进程
		verifyCmd := "ps aux | grep '[s]leep 30' | wc -l"
		verifyCode, verifyOut, _, _ := RunSSHCommand(config, verifyCmd, 5)

		if verifyCode == 0 {
			count := strings.TrimSpace(verifyOut)
			if count != "0" {
				t.Errorf("Found %s residual sleep processes after timeout", count)
			}
		}
	})

	// 测试2: 后台命令启动超时资源清理
	t.Run("background timeout cleanup", func(t *testing.T) {
		bgConfig := config
		bgConfig.Background = true

		cmd := "sleep 60"
		code, pid, _, err := RunSSHCommand(bgConfig, cmd, 2)

		// 后台命令启动应该成功
		if code != 0 {
			t.Errorf("Expected background startup code 0, got %d", code)
		}
		if err != nil {
			t.Errorf("Unexpected error during background startup: %v", err)
		}

		// 清理后台进程
		if pid != "" && pid != "0" {
			killCmd := "kill -9 " + pid
			RunSSHCommand(config, killCmd, 5)
		}
	})
}
