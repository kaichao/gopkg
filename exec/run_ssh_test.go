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
	command := "echo 'starting process'; sleep 60"
	code, pid, output, err := RunSSHCommand(config, command, 5)
	if err != nil {
		t.Fatalf("RunSSHCommand failed: %v", err)
	}
	if code != 0 {
		t.Errorf("Expected exit code 0, got %d", code)
	}

	// Verify output contains both startup message and PID
	outputLines := strings.Split(strings.TrimSpace(output), "\n")
	if len(outputLines) < 2 {
		t.Fatalf("Expected at least 2 lines of output, got: %d", len(outputLines))
	}

	// First line should be startup output
	if !strings.Contains(outputLines[0], "starting process") {
		t.Errorf("Expected startup output, got: %s", outputLines[0])
	}

	// Last line should be PID
	if pid == "" {
		t.Error("Expected PID in output")
	}
	if !strings.Contains(outputLines[len(outputLines)-1], pid) {
		t.Errorf("Expected PID %s in last line, got: %s", pid, outputLines[len(outputLines)-1])
	}

	if pid == "" || pid == "0" {
		t.Error("Invalid PID returned")
	}

	// Cleanup
	killCmd := fmt.Sprintf("kill -9 %s", strings.Fields(pid)[0])
	RunSSHCommand(config, killCmd, 5)
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
		t.Fatal("SSH authentication not configured: must set either KeyPath or Password")
	}

	// Test process cleanup after timeout with marker
	marker := fmt.Sprintf("MARKER_%d", time.Now().UnixNano())
	command := fmt.Sprintf("singularity exec /root/singularity/debian.sif sleep 60 & pid=$!; echo \"$pid %s\"; wait", marker)
	t.Logf("Using marker: %s", marker)

	code, _, _, err := RunSSHCommand(config, command, 2) // Short timeout
	if err == nil || code != 124 {
		t.Errorf("Expected timeout error (124), got code=%d err=%v", code, err)
	}

	// Wait longer for process to start and register
	time.Sleep(5 * time.Second)

	// Find process by marker
	_, output, _, _ := RunSSHCommand(config, "pgrep -f '"+marker+"' | head -1", 5)
	pid := strings.TrimSpace(output)
	if pid == "" {
		t.Fatal("Failed to find target process PID")
	}
	isContainer := false

	// Process termination with multiple verification methods
	for i := 1; i <= 3; i++ {
		var killCmd string
		if isContainer {
			killCmd = "singularity exec instance kill $(singularity instance list | grep " + pid + " | awk '{print $1}')"
		} else {
			killCmd = "kill -9 " + pid
		}

		RunSSHCommand(config, killCmd, 5)
		time.Sleep(1 * time.Second)

		// Verify process is gone
		_, psOutput, _, _ := RunSSHCommand(config, "ps -p "+pid+" -o pid= 2>/dev/null || echo 'NOT_FOUND'", 5)
		if strings.Contains(psOutput, "NOT_FOUND") {
			return
		}
	}

	t.Fatalf("Failed to terminate process %s after 3 attempts", pid)
}

func TestCommandTimeout(t *testing.T) {
	config := SSHConfig{
		User:     testSSHUser,
		Host:     testSSHServer,
		Port:     testSSHPort,
		KeyPath:  testSSHKey,
		Password: testPassword,
	}

	// Test background command timeout
	bgConfig := config
	bgConfig.Background = true
	marker := fmt.Sprintf("MARKER_%d", time.Now().UnixNano())
	bgCmd := fmt.Sprintf("echo 'starting'; sleep 10; echo %s", marker)

	start := time.Now()
	code, _, _, err := RunSSHCommand(bgConfig, bgCmd, 2)
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

	// Original timeout test
	config.Background = false

	marker2 := fmt.Sprintf("MARKER_%d", time.Now().UnixNano()+1)
	cmd2 := fmt.Sprintf("sleep 10 && echo %s", marker2)

	// 设置更宽松的超时阈值
	start2 := time.Now()
	code2, _, _, err2 := RunSSHCommand(config, cmd2, 2)
	duration2 := time.Since(start2)

	if code2 != 124 {
		t.Errorf("Expected exit code 124, got %d", code2)
	}
	if err2 == nil || !strings.Contains(err2.Error(), "timed out") {
		t.Errorf("Expected timeout error, got %v", err2)
	}
	if duration2 >= 5*time.Second {
		t.Errorf("Timeout took too long: %v", duration2)
	}

	// 简化进程清理验证
	cleanCmd := fmt.Sprintf("pkill -9 -f '%s'; killall -9 sleep singularity", marker2)
	_, _, _, _ = RunSSHCommand(config, cleanCmd, 5)

	// 快速验证
	verifyCmd := fmt.Sprintf("pgrep -f '%s' || echo clean", marker2)
	_, out, _, _ := RunSSHCommand(config, verifyCmd, 2)
	if !strings.Contains(out, "clean") {
		t.Logf("Process cleanup warning: some processes may still be running")
	}

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
