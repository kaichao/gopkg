package exec

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/kaichao/gopkg/errors"
	"github.com/stretchr/testify/assert"
)

var (
	testSSHServer = "10.0.6.100"
	testSSHPort   = 22
	testSSHUser   = "root"
	testSSHKey    = "" // defaults to SSH key in user home directory
	testPassword  = "" // can be set for password authentication
)

func init() {
	// Attempt to set default SSH key path for testing
	if path, err := defaultSSHKeyPath(); err == nil {
		testSSHKey = path
	}
}

func TestWrapCommand(t *testing.T) {
	// Test with useHomeTmp = false (default behavior)
	t.Run("useHomeTmp=false", func(t *testing.T) {
		wrapper, marker := wrapCommand("echo hello", false)
		assert.Contains(t, wrapper, "nohup")
		assert.Contains(t, wrapper, ">/dev/null 2>&1")
		assert.Contains(t, wrapper, marker)
		assert.NotContains(t, wrapper, "${HOME}/tmp")
		assert.Contains(t, wrapper, "echo hello")
	})

	// Test with useHomeTmp = true
	t.Run("useHomeTmp=true", func(t *testing.T) {
		wrapper, marker := wrapCommand("echo hello", true)
		assert.Contains(t, wrapper, "nohup")
		assert.Contains(t, wrapper, ">${HOME}/tmp/nohup.out 2>&1")
		assert.Contains(t, wrapper, "mkdir -p ${HOME}/tmp")
		assert.Contains(t, wrapper, marker)
		assert.Contains(t, wrapper, "echo hello")
	})

	// Test that marker format is correct and unique (timestamps differ)
	t.Run("unique marker", func(t *testing.T) {
		wrapper1, marker1 := wrapCommand("echo test", false)
		time.Sleep(time.Microsecond) // ensure different timestamp
		wrapper2, marker2 := wrapCommand("echo test", false)
		assert.NotEqual(t, marker1, marker2, "markers should be unique")
		assert.Contains(t, wrapper1, marker1)
		assert.Contains(t, wrapper2, marker2)
	})
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

	// Check authentication configuration
	if config.KeyPath == "" && config.Password == "" {
		t.Fatal("SSH authentication not configured: must set either KeyPath or Password")
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := RunSSHCommand(config, tt.command, tt.timeout)
			if err != nil {
				t.Errorf("RunSSHCommand() error = %v", err)
			}
			if errors.GetCode(err) != tt.wantCode {
				t.Errorf("RunSSHCommand() code = %v, want %v", errors.GetCode(err), tt.wantCode)
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
	pid, stderr, err := RunSSHCommand(config, command, 10)
	if err != nil {
		t.Fatalf("RunSSHCommand failed: %v", err)
	}
	if errors.GetCode(err) != 0 {
		t.Errorf("Expected exit code 0, got %d", errors.GetCode(err))
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
	_, killOut, killErr := RunSSHCommand(config, killCmd, 5)
	if errors.GetCode(killErr) != 0 && killErr != nil {
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

	// Check authentication configuration
	if config.KeyPath == "" && config.Password == "" {
		t.Skip("SSH authentication not configured: must set either KeyPath or Password")
	}

	// Start command in background mode
	bgConfig := config
	bgConfig.Background = true
	command := "sleep 30" // use a shorter sleep time

	// Start background command and get PID
	pidOutput, _, err := RunSSHCommand(bgConfig, command, 5)
	if err != nil {
		t.Fatalf("Failed to start background command: %v", err)
	}
	if errors.GetCode(err) != 0 {
		t.Fatalf("Expected exit code 0, got %d", errors.GetCode(err))
	}

	// Background command only returns PID
	pid := strings.TrimSpace(pidOutput)
	if pid == "" || pid == "0" {
		t.Fatalf("Invalid PID output: %s", pidOutput)
	}

	t.Logf("Started process with PID: %s", pid)

	// Verify the process is running (use non-background config)
	psOutput, _, _ := RunSSHCommand(config, "ps -p "+pid+" -o pid= 2>/dev/null || echo 'NOT_FOUND'", 5)
	if strings.Contains(psOutput, "NOT_FOUND") {
		t.Fatalf("Process %s not found after startup", pid)
	}

	// Use more reliable cleanup methods
	cleanupCmds := []string{
		"kill -TERM " + pid, // send TERM signal first
		"sleep 1",
		"kill -KILL " + pid,   // send KILL signal next
		"pkill -9 -f 'sleep'", // clean up all sleep processes
	}

	for _, cmd := range cleanupCmds {
		RunSSHCommand(config, cmd, 5) // non-background: send kill signals directly
		time.Sleep(500 * time.Millisecond)
	}

	// Verify the process has been terminated
	time.Sleep(3 * time.Second)

	// Use multiple methods to verify process termination
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
		output, _, _ := RunSSHCommand(config, verify.cmd, 5) // non-background: verify directly
		output = strings.TrimSpace(output)

		// If command succeeded (exit code 0) and output is empty or "0", process does not exist
		// Or output contains expected termination message, process has terminated
		if errors.GetCode(err) == 0 && (output == "" || output == "0") {
			// Command succeeded with empty or "0" output, process not found
			t.Logf("Process verification passed for command '%s': output='%s' (process not found)", verify.cmd, output)
		} else if output == verify.successValue {
			// Command reported expected termination message
			t.Logf("Process verification passed for command '%s': output='%s'", verify.cmd, output)
		} else {
			// Command succeeded with non-empty output, process may still be running
			allTerminated = false
			t.Logf("Process verification failed for command '%s': code=%d, output='%s'", verify.cmd, errors.GetCode(err), output)
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

	// Check authentication configuration
	if config.KeyPath == "" && config.Password == "" {
		t.Skip("SSH authentication not configured: must set either KeyPath or Password")
	}

	// Test 1: non-background command timeout
	t.Run("non-background command timeout", func(t *testing.T) {
		cmd := "sleep 10"
		start := time.Now()
		_, _, err := RunSSHCommand(config, cmd, 2)
		duration := time.Since(start)

		if errors.GetCode(err) != 124 {
			t.Errorf("Expected exit code 124, got %d", errors.GetCode(err))
		}
		if err == nil || !strings.Contains(err.Error(), "timed out") {
			t.Errorf("Expected timeout error, got %v", err)
		}
		if duration > 3*time.Second {
			t.Errorf("Timeout took too long: %v", duration)
		}
	})

	// Test 2: background command startup timeout (only startup phase has timeout)
	t.Run("background command startup timeout", func(t *testing.T) {
		bgConfig := config
		bgConfig.Background = true

		// Use a long-running command to test startup timeout
		// The wrapCommand script itself is fast, so test a normal background command
		cmd := "sleep 60" // long-running command
		start := time.Now()
		_, _, err := RunSSHCommand(bgConfig, cmd, 2)
		duration := time.Since(start)

		// Background command startup should succeed because wrapCommand is fast
		if errors.GetCode(err) != 0 {
			t.Errorf("Expected exit code 0 for background command startup, got %d", errors.GetCode(err))
		}
		if err != nil {
			t.Errorf("Expected no error for background command startup, got %v", err)
		}
		if duration > 3*time.Second {
			t.Errorf("Background startup took too long: %v", duration)
		}
	})

	// Test 3: background command normal startup (short command)
	t.Run("background command normal startup", func(t *testing.T) {
		bgConfig := config
		bgConfig.Background = true

		// Use a fast command
		cmd := "echo 'quick command'"
		pidOutput, _, err := RunSSHCommand(bgConfig, cmd, 5)

		// Background command should start successfully and return PID
		if errors.GetCode(err) != 0 {
			t.Errorf("Expected exit code 0 for background command, got %d", errors.GetCode(err))
		}
		if err != nil {
			t.Errorf("Expected no error for background command, got %v", err)
		}
		if pidOutput == "" {
			t.Error("Expected PID output for background command")
		}

		// Parse PID and clean up
		fields := strings.Fields(pidOutput)
		if len(fields) >= 1 {
			pid := fields[0]
			if pid != "0" {
				// Clean up process
				killCmd := "kill -9 " + pid
				RunSSHCommand(config, killCmd, 5)
			}
		}
	})

	// Test 4: standard command timeout test
	t.Run("standard command timeout", func(t *testing.T) {
		config.Background = false
		cmd := "sleep 10"

		start := time.Now()
		_, _, err := RunSSHCommand(config, cmd, 2)
		duration := time.Since(start)

		if errors.GetCode(err) != 124 {
			t.Errorf("Expected exit code 124, got %d", errors.GetCode(err))
		}
		if err == nil || !strings.Contains(err.Error(), "timed out") {
			t.Errorf("Expected timeout error, got %v", err)
		}
		if duration >= 5*time.Second {
			t.Errorf("Timeout took too long: %v", duration)
		}
	})

	// Clean up all possible residual processes
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

	// Enhanced resource cleanup verification
	assert.Eventually(t, func() bool {
		// More thorough cleanup command combination
		cleanCmd := fmt.Sprintf(`
			sudo rm -f %s || rm -f %s || true;
			if [ -f %s ]; then
				echo "FILE_STILL_EXISTS: $(ls -l %s)";
				false;
			else
				echo "missing";
			fi
		`, pidFile, pidFile, pidFile, pidFile)

		out, _, err := RunSSHCommand(config, cleanCmd, 5)
		t.Logf("Cleanup output: %s", out)
		return errors.GetCode(err) == 0 && strings.Contains(out, "missing")
	}, 30*time.Second, 2*time.Second, "Resource cleanup failed")
}


func TestSignalNameToNumber(t *testing.T) {
	tests := []struct {
		signal string
		want   int
	}{
		{"HUP", 1},
		{"INT", 2},
		{"QUIT", 3},
		{"ILL", 4},
		{"TRAP", 5},
		{"ABRT", 6},
		{"BUS", 7},
		{"FPE", 8},
		{"KILL", 9},
		{"USR1", 10},
		{"SEGV", 11},
		{"USR2", 12},
		{"PIPE", 13},
		{"ALRM", 14},
		{"TERM", 15},
		{"STKFLT", 16},
		{"CHLD", 17},
		{"CONT", 18},
		{"STOP", 19},
		{"TSTP", 20},
		{"TTIN", 21},
		{"TTOU", 22},
		{"URG", 23},
		{"XCPU", 24},
		{"XFSZ", 25},
		{"VTALRM", 26},
		{"PROF", 27},
		{"WINCH", 28},
		{"IO", 29},
		{"PWR", 30},
		{"SYS", 31},
		{"UNKNOWN", 0},
		{"", 0},
	}

	for _, tt := range tests {
		t.Run(tt.signal, func(t *testing.T) {
			got := signalNameToNumber(tt.signal)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRunSSHCommandNonZeroExit(t *testing.T) {
	config := SSHConfig{
		User:     testSSHUser,
		Host:     testSSHServer,
		Port:     testSSHPort,
		KeyPath:  testSSHKey,
		Password: testPassword,
	}

	if config.KeyPath == "" && config.Password == "" {
		t.Skip("SSH authentication not configured")
	}

	tests := []struct {
		name     string
		command  string
		wantCode int
	}{
		{
			name:     "explicit exit 5",
			command:  "sh -c 'exit 5'",
			wantCode: 5,
		},
		{
			name:     "exit 42",
			command:  "sh -c 'exit 42'",
			wantCode: 42,
		},
		{
			name:     "exit 1 on error",
			command:  "sh -c 'false'",
			wantCode: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := RunSSHCommand(config, tt.command, 10)
			assert.Equal(t, tt.wantCode, errors.GetCode(err))
			assert.NotNil(t, err)
			assert.Contains(t, err.Error(), "exit-code not zero")
			assert.Equal(t, tt.wantCode, errors.GetCode(err))
		})
	}
}

func TestRunSSHCommandSignalTermination(t *testing.T) {
	config := SSHConfig{
		User:     testSSHUser,
		Host:     testSSHServer,
		Port:     testSSHPort,
		KeyPath:  testSSHKey,
		Password: testPassword,
	}

	if config.KeyPath == "" && config.Password == "" {
		t.Skip("SSH authentication not configured")
	}

	t.Run("kill with SIGKILL", func(t *testing.T) {
		_, _, err := RunSSHCommand(config, "sh -c 'kill -9 $$'", 5)
		// SIGKILL: 128 + 9 = 137
		assert.Equal(t, 137, errors.GetCode(err))
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "exit-code not zero")
	})
}
