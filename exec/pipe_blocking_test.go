package exec

import (
	"strings"
	"testing"
	"time"
)

// TestPipeBlockingFix specifically tests the fix for pipe blocking issues
func TestPipeBlockingFix(t *testing.T) {
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
			expectCode:  124, // timeout exit code
			description: "test pipe blocking issue for non-background commands with large output on timeout",
		},
		{
			name:        "background command with timeout during startup",
			command:     "sleep 60", // long-running command
			timeout:     2,
			background:  true,
			expectCode:  0, // background command startup should succeed
			description: "test timeout handling during background command startup",
		},
		{
			name:        "command with stderr output and timeout",
			command:     "for i in {1..100}; do echo 'error output $i' >&2; done; sleep 10",
			timeout:     2,
			background:  false,
			expectCode:  124,
			description: "test stderr output pipe handling on timeout",
		},
		{
			name:        "mixed stdout/stderr with timeout",
			command:     "for i in {1..50}; do echo 'stdout $i'; echo 'stderr $i' >&2; done; sleep 10",
			timeout:     2,
			background:  false,
			expectCode:  124,
			description: "test mixed stdout/stderr output pipe handling on timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)

			config.Background = tt.background
			start := time.Now()

			code, stdout, stderr, err := RunSSHCommand(config, tt.command, tt.timeout)
			duration := time.Since(start)

			// Verify timeout duration control
			if tt.expectCode == 124 {
				if duration > time.Duration(tt.timeout+1)*time.Second {
					t.Errorf("Timeout handling took too long: %v (expected < %ds)", duration, tt.timeout+1)
				}
			}

			// Verify exit code
			if code != tt.expectCode {
				t.Errorf("Expected exit code %d, got %d", tt.expectCode, code)
			}

			// Verify error message
			if tt.expectCode == 124 {
				if err == nil || !strings.Contains(err.Error(), "timed out") {
					t.Errorf("Expected timeout error, got: %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}

			// Verify output handling (should not have pipe blocking)
			if tt.background {
				// Background command should return PID
				if stdout == "" || stdout == "0" {
					t.Errorf("Background command should return PID, got: %s", stdout)
				}
			} else {
				// Non-background commands should have partial output (even on timeout)
				if tt.expectCode == 124 {
					// On timeout, some output should have been captured
					if len(stdout) == 0 && len(stderr) == 0 {
						t.Log("Warning: No output captured during timeout (this might indicate pipe blocking)")
					}
				}
			}

			t.Logf("Test completed in %v, code=%d, stdout_len=%d, stderr_len=%d",
				duration, code, len(stdout), len(stderr))
		})
	}

	// Clean up processes that may have been created during testing
	cleanupCmd := "pkill -9 -f 'sleep'; pkill -9 -f 'MARKER_'"
	RunSSHCommand(config, cleanupCmd, 5)
}

// TestResourceCleanupOnTimeout specifically tests resource cleanup on timeout
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

	// Test 1: non-background command timeout resource cleanup
	t.Run("non-background timeout cleanup", func(t *testing.T) {
		cmd := "sleep 30"
		code, _, _, err := RunSSHCommand(config, cmd, 2)

		if code != 124 {
			t.Errorf("Expected timeout code 124, got %d", code)
		}
		if err == nil || !strings.Contains(err.Error(), "timed out") {
			t.Errorf("Expected timeout error, got %v", err)
		}

		// Verify no residual sleep processes
		verifyCmd := "ps aux | grep '[s]leep 30' | wc -l"
		verifyCode, verifyOut, _, _ := RunSSHCommand(config, verifyCmd, 5)

		if verifyCode == 0 {
			count := strings.TrimSpace(verifyOut)
			if count != "0" {
				t.Errorf("Found %s residual sleep processes after timeout", count)
			}
		}
	})

	// Test 2: background command startup timeout resource cleanup
	t.Run("background timeout cleanup", func(t *testing.T) {
		bgConfig := config
		bgConfig.Background = true

		cmd := "sleep 60"
		code, pid, _, err := RunSSHCommand(bgConfig, cmd, 2)

		// Background command startup should succeed
		if code != 0 {
			t.Errorf("Expected background startup code 0, got %d", code)
		}
		if err != nil {
			t.Errorf("Unexpected error during background startup: %v", err)
		}

		// Clean up background process
		if pid != "" && pid != "0" {
			killCmd := "kill -9 " + pid
			RunSSHCommand(config, killCmd, 5)
		}
	})
}
