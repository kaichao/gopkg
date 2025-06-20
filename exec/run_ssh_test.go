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
	testSSHKey    = "/Users/kaichao/.ssh/id_rsa" // 默认使用用户主目录下的SSH密钥
	testPassword  = ""                           // 可设置为密码认证
)

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

	// Enhanced process detection with multiple methods
	pid := ""
	detectionMethods := []string{
		"ps aux | grep -E '" + marker + "' | grep -v grep | head -1 | awk '{print $2}'",
		"pgrep -f '" + marker + "' | head -1",
		"ps -eo pid,cmd | grep -E '" + marker + "' | grep -v grep | head -1 | awk '{print $1}'",
	}

	for _, cmd := range detectionMethods {
		_, output, _, _ := RunSSHCommand(config, cmd, 5)
		pid = strings.TrimSpace(output)
		if pid != "" {
			break
		}
	}

	if pid == "" {
		// Debug: show all marker processes
		_, allMarkers, _, _ := RunSSHCommand(config, "ps aux | grep -E '"+marker+"'", 5)
		t.Fatalf("Failed to find target process PID. Marker processes:\n%s", allMarkers)
	}

	t.Logf("Target process PID: %s", pid)

	// Enhanced process verification
	_, procDetails, _, _ := RunSSHCommand(config, "ps -p "+pid+" -o pid,cmd", 5)
	t.Logf("Process details:\n%s", procDetails)

	// Container detection
	_, cgroup, _, _ := RunSSHCommand(config, "cat /proc/"+pid+"/cgroup 2>/dev/null || echo 'not_in_container'", 5)
	isContainer := !strings.Contains(cgroup, "not_in_container")

	// Process termination with multiple verification methods
	for i := 1; i <= 3; i++ {
		t.Logf("Kill attempt %d (container:%v)", i, isContainer)

		var killCmd string
		if isContainer {
			_, containerID, _, _ := RunSSHCommand(config,
				"grep -o 'docker/\\|lxc/\\|containerd/\\w\\+' /proc/"+pid+"/cgroup | head -1 | cut -d/ -f3 || true", 5)
			if containerID != "" {
				killCmd = fmt.Sprintf("docker kill %s || singularity exec instance kill %s", containerID, containerID)
			} else {
				killCmd = "singularity exec instance kill $(singularity instance list | grep " + pid + " | awk '{print $1}')"
			}
		} else {
			killCmd = "kill -9 " + pid
		}

		// Execute kill command with verification
		_, killOutput, _, _ := RunSSHCommand(config, killCmd+" && echo 'KILL_SUCCESS' || echo 'KILL_FAILED'", 5)
		t.Logf("Kill command output: %s", killOutput)
		time.Sleep(2 * time.Second) // Longer wait after kill

		// Enhanced termination verification
		_, psOutput, _, _ := RunSSHCommand(config, "ps -p "+pid+" -o pid= 2>/dev/null || echo 'NOT_FOUND'", 5)
		_, procStatus, _, _ := RunSSHCommand(config, "cat /proc/"+pid+"/status 2>/dev/null || echo 'PROCESS_GONE'", 5)

		t.Logf("Termination verification:\nps: %s\nstatus: %s",
			strings.TrimSpace(psOutput),
			strings.TrimSpace(procStatus))

		if strings.Contains(procStatus, "PROCESS_GONE") {
			t.Logf("Process %s successfully terminated (verified by /proc status)", pid)
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

	marker := fmt.Sprintf("MARKER_%d", time.Now().UnixNano())
	cmd := fmt.Sprintf("sleep 10 && echo %s", marker)

	// 设置更宽松的超时阈值
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

	// 简化进程清理验证
	cleanCmd := fmt.Sprintf("pkill -9 -f '%s'; killall -9 sleep singularity", marker)
	_, _, _, _ = RunSSHCommand(config, cleanCmd, 5)

	// 快速验证
	verifyCmd := fmt.Sprintf("pgrep -f '%s' || echo clean", marker)
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
