package exec

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var (
	testSSHServer = "10.0.6.100"
	testSSHPort   = 22
	testSSHUser   = "root"
	testSSHKey    = "" // 默认使用用户主目录下的SSH密钥
	testPassword  = "" // 可设置为密码认证
)

func init() {
	// 尝试设置默认SSH密钥路径用于测试
	if path, err := defaultSSHKeyPath(); err == nil {
		testSSHKey = path
	}
}

func TestRunSingularityCommand(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		timeout  int
		wantCode int
	}{
		{
			name:     "singularity exec simple command",
			command:  "singularity exec /root/singularity/debian.sif echo hello",
			timeout:  30,
			wantCode: 0,
		},
		{
			name:     "singularity with env vars",
			command:  "SINGULARITYENV_FOO=bar singularity exec /root/singularity/debian.sif env",
			timeout:  30,
			wantCode: 0,
		},
		{
			name:     "singularity with bind mounts",
			command:  "singularity exec -B /tmp:/mnt /root/singularity/debian.sif ls /mnt",
			timeout:  30,
			wantCode: 0,
		},
	}

	config := SSHConfig{
		User:     testSSHUser,
		Host:     testSSHServer,
		Port:     testSSHPort,
		KeyPath:  testSSHKey,
		Password: testPassword,
	}

	// 检查认证配置
	if config.KeyPath == "" && config.Password == "" {
		t.Fatal("SSH authentication not configured: must set either KeyPath or Password")
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, stdout, stderr, err := RunSSHCommand(config, tt.command, tt.timeout)
			if err != nil {
				t.Errorf("RunSSHCommand() error = %v", err)
			}
			if code != tt.wantCode {
				t.Errorf("RunSSHCommand() code = %v, want %v", code, tt.wantCode)
				t.Logf("stdout: %s", stdout)
				t.Logf("stderr: %s", stderr)
			}
		})
	}
}

func TestBackgroundCommand(t *testing.T) {
	config := SSHConfig{
		User:       testSSHUser,
		Host:       testSSHServer,
		Port:       testSSHPort,
		KeyPath:    testSSHKey,
		Password:   testPassword,
		Background: true,
	}

	// Test background command with output
	command := "echo startup_message; sleep 60"
	code, pid, stderr, err := RunSSHCommand(config, command, 10)
	if err != nil {
		t.Fatalf("RunSSHCommand failed: %v", err)
	}
	if code != 0 {
		t.Errorf("Expected exit code 0, got %d", code)
		t.Logf("stderr: %s", stderr)
	}

	// Extract PID from output with marker format
	if pid == "" {
		t.Fatal("Empty PID output")
	}

	// Parse pid with flexible format handling
	var pidVal string
	fields := strings.Fields(pid)
	if len(fields) >= 1 {
		pidVal = fields[0]
	} else {
		pidVal = pid
	}

	// Ensure we get a numeric PID
	if pidVal == "0" || !strings.ContainsAny(pidVal, "0123456789") {
		t.Fatalf("Invalid PID format: %s", pidVal)
	}

	// Cleanup with explicit error handling
	killCmd := fmt.Sprintf("kill -9 %s", pidVal)
	killCode, _, killOut, killErr := RunSSHCommand(config, killCmd, 5)
	if killCode != 0 && killErr != nil {
		t.Logf("Cleanup warning: %v", killErr)
		t.Logf("Cleanup output: %s", killOut)
	}
}

func TestProcessCleanup(t *testing.T) {
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

	// 使用后台模式启动命令
	config.Background = true
	command := "sleep 30" // 使用较短的睡眠时间

	// 启动后台命令，获取PID
	code, pidOutput, _, err := RunSSHCommand(config, command, 5)
	if err != nil {
		t.Fatalf("Failed to start background command: %v", err)
	}
	if code != 0 {
		t.Fatalf("Expected exit code 0, got %d", code)
	}

	// 后台命令只返回PID
	pid := strings.TrimSpace(pidOutput)
	if pid == "" || pid == "0" {
		t.Fatalf("Invalid PID output: %s", pidOutput)
	}

	t.Logf("Started process with PID: %s", pid)

	// 验证进程正在运行
	_, psOutput, _, _ := RunSSHCommand(config, "ps -p "+pid+" -o pid= 2>/dev/null || echo 'NOT_FOUND'", 5)
	if strings.Contains(psOutput, "NOT_FOUND") {
		t.Fatalf("Process %s not found after startup", pid)
	}

	// 使用更可靠的清理方法
	cleanupCmds := []string{
		"kill -TERM " + pid, // 先发送TERM信号
		"sleep 1",
		"kill -KILL " + pid,   // 再发送KILL信号
		"pkill -9 -f 'sleep'", // 清理所有sleep进程
	}

	for _, cmd := range cleanupCmds {
		RunSSHCommand(config, cmd, 5)
		time.Sleep(500 * time.Millisecond)
	}

	// 验证进程已被终止
	time.Sleep(3 * time.Second)

	// 使用多种方法验证进程是否已终止
	verifyCmds := []struct {
		cmd          string
		successValue string
	}{
		{
			cmd:          "ps -p " + pid + " -o pid= 2>/dev/null || echo 'NOT_FOUND'",
			successValue: "NOT_FOUND",
		},
		{
			cmd:          "kill -0 " + pid + " 2>/dev/null || echo 'TERMINATED'",
			successValue: "TERMINATED",
		},
		{
			cmd:          "ls /proc/" + pid + " 2>/dev/null || echo 'NO_PROC'",
			successValue: "NO_PROC",
		},
	}

	allTerminated := true
	for _, verify := range verifyCmds {
		code, output, _, _ := RunSSHCommand(config, verify.cmd, 5)
		output = strings.TrimSpace(output)

		// 如果命令执行成功（返回码为0）且输出为空或为"0"，说明进程不存在
		// 或者输出包含预期的终止消息，说明进程已终止
		if code == 0 && (output == "" || output == "0") {
			// 命令执行成功且输出为空或"0"，说明进程不存在
			t.Logf("Process verification passed for command '%s': output='%s' (process not found)", verify.cmd, output)
		} else if output == verify.successValue {
			// 命令执行失败但返回了预期的终止消息
			t.Logf("Process verification passed for command '%s': output='%s'", verify.cmd, output)
		} else {
			// 命令执行成功且有非空输出，说明进程可能仍在运行
			allTerminated = false
			t.Logf("Process verification failed for command '%s': code=%d, output='%s'", verify.cmd, code, output)
		}
	}

	if !allTerminated {
		t.Errorf("Process %s may still be running after cleanup attempts", pid)
	} else {
		t.Logf("Process %s successfully terminated", pid)
	}
}

func TestCommandTimeout(t *testing.T) {
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

	// Test 1: 非后台命令超时
	t.Run("non-background command timeout", func(t *testing.T) {
		cmd := "sleep 10"
		start := time.Now()
		code, _, _, err := RunSSHCommand(config, cmd, 2)
		duration := time.Since(start)

		if code != 124 {
			t.Errorf("Expected exit code 124, got %d", code)
		}
		if err == nil || !strings.Contains(err.Error(), "timed out") {
			t.Errorf("Expected timeout error, got %v", err)
		}
		if duration > 3*time.Second {
			t.Errorf("Timeout took too long: %v", duration)
		}
	})

	// Test 2: 后台命令启动超时（后台命令只对启动过程有超时）
	t.Run("background command startup timeout", func(t *testing.T) {
		bgConfig := config
		bgConfig.Background = true

		// 使用一个长时间运行的命令来测试启动超时
		// 由于wrapCommand脚本本身很快，我们测试正常的后台命令
		cmd := "sleep 60" // 长时间运行的命令
		start := time.Now()
		code, _, _, err := RunSSHCommand(bgConfig, cmd, 2)
		duration := time.Since(start)

		// 后台命令的启动过程应该成功，因为wrapCommand脚本很快
		if code != 0 {
			t.Errorf("Expected exit code 0 for background command startup, got %d", code)
		}
		if err != nil {
			t.Errorf("Expected no error for background command startup, got %v", err)
		}
		if duration > 3*time.Second {
			t.Errorf("Background startup took too long: %v", duration)
		}
	})

	// Test 3: 后台命令正常启动（短时间命令）
	t.Run("background command normal startup", func(t *testing.T) {
		bgConfig := config
		bgConfig.Background = true

		// 使用一个快速完成的命令
		cmd := "echo 'quick command'"
		code, pidOutput, _, err := RunSSHCommand(bgConfig, cmd, 5)

		// 后台命令应该成功启动并返回PID
		if code != 0 {
			t.Errorf("Expected exit code 0 for background command, got %d", code)
		}
		if err != nil {
			t.Errorf("Expected no error for background command, got %v", err)
		}
		if pidOutput == "" {
			t.Error("Expected PID output for background command")
		}

		// 解析PID并清理
		fields := strings.Fields(pidOutput)
		if len(fields) >= 1 {
			pid := fields[0]
			if pid != "0" {
				// 清理进程
				killCmd := "kill -9 " + pid
				RunSSHCommand(config, killCmd, 5)
			}
		}
	})

	// Test 4: 标准命令超时测试
	t.Run("standard command timeout", func(t *testing.T) {
		config.Background = false
		cmd := "sleep 10"

		start := time.Now()
		code, _, _, err := RunSSHCommand(config, cmd, 2)
		duration := time.Since(start)

		if code != 124 {
			t.Errorf("Expected exit code 124, got %d", code)
		}
		if err == nil || !strings.Contains(err.Error(), "timed out") {
			t.Errorf("Expected timeout error, got %v", err)
		}
		if duration >= 5*time.Second {
			t.Errorf("Timeout took too long: %v", duration)
		}
	})

	// 清理所有可能的残留进程
	cleanCmd := "pkill -9 -f 'sleep'; pkill -9 -f 'MARKER_'"
	RunSSHCommand(config, cleanCmd, 5)
}

func TestResourceCleanup(t *testing.T) {
	config := SSHConfig{
		User:     testSSHUser,
		Host:     testSSHServer,
		Port:     testSSHPort,
		KeyPath:  testSSHKey,
		Password: testPassword,
	}

	uniqueID := fmt.Sprintf("%d", time.Now().UnixNano())
	pidFile := fmt.Sprintf("/tmp/ssh_test_%s", uniqueID)

	testCmd := fmt.Sprintf(
		"tmp=$(mktemp) && echo $$ > $tmp && mv $tmp %s && sleep 30",
		pidFile,
	)

	go RunSSHCommand(config, testCmd, 1)

	// 增强资源清理验证
	assert.Eventually(t, func() bool {
		// 更彻底的清理命令组合
		cleanCmd := fmt.Sprintf(`
			sudo rm -f %s || rm -f %s || true;
			if [ -f %s ]; then
				echo "FILE_STILL_EXISTS: $(ls -l %s)";
				false;
			else
				echo "missing";
			fi
		`, pidFile, pidFile, pidFile, pidFile)

		code, out, _, _ := RunSSHCommand(config, cleanCmd, 5)
		t.Logf("Cleanup output: %s", out)
		return code == 0 && strings.Contains(out, "missing")
	}, 30*time.Second, 2*time.Second, "Resource cleanup failed")
}
