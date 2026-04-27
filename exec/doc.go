// Package exec provides cross-environment command execution utilities.
//
// Overview:
// The exec package offers a unified interface for executing commands in various
// environments including local execution and remote SSH execution. It provides
// full output capture, flexible timeout management, and robust error handling.
//
// Core Features:
// - Local command execution with timeout and output capture
// - SSH remote command execution with authentication support
// - Circular buffering for large output (10MB limit)
// - Process group management for proper termination
// - Background process support for SSH commands
//
// Usage Examples:
//
// Local Execution:
//
//	import "github.com/kaichao/gopkg/exec"
//
//	code, stdout, stderr, err := exec.RunReturnAll("ls -l /tmp", 10)
//	if err != nil {
//		log.Printf("Execution failed: %v\nOutput: %s\nError: %s", err, stdout, stderr)
//	}
//
// SSH Remote Execution:
//
//	config := exec.SSHConfig{
//		User:    "admin",
//		Host:    "10.0.0.1",
//		KeyPath: "/home/user/.ssh/id_rsa",
//	}
//
//	code, out, errOut, err := exec.RunSSHCommand(config, "docker ps -a", 30)
//
// Key Functions:
//
//	RunReturnAll(command string, timeout int) (code int, stdout string, stderr string, err error)
//	RunReturnExitCode(command string, timeout int) (code int, error)
//	RunReturnStdout(command string, timeout int) (string, error)
//	RunWithRetries(cmd string, numRetries int, timeout int) (int, error)
//	RunSSHCommand(config SSHConfig, command string, timeout int) (code int, stdout string, stderr string, err error)
//
// Exit Code Convention:
//   - 0: Command executed successfully
//   - 124: Command timed out
//   - 125: Command execution failed (e.g., pipe creation, process start)
//   - Other non-zero: Command-specific exit code
//   - 128 + signal: Command terminated by signal (e.g., SIGKILL = 128+9 = 137)
//
// SSH Configuration:
//
//	type SSHConfig struct {
//		User       string // Required: SSH username
//		Host       string // Required: SSH host
//		Port       int    // Optional: SSH port (default: 22)
//		KeyPath    string // Optional: Path to SSH private key (preferred over password)
//		Password   string // Optional: SSH password
//		Background bool   // Optional: Run command in background mode
//		UseHomeTmp bool   // Optional: Use ${HOME}/tmp instead of /tmp
//	}
//
// Output Handling:
// - Standard output and error are captured using circular buffers (10MB limit)
// - Output is not automatically printed to os.Stdout/os.Stderr; it is returned to the caller
// - Background SSH commands return PID instead of output
//
// Error Handling:
// - All functions return consistent error types following gopkg/errors conventions
// - Timeouts are distinguished from other execution errors
// - Non-zero exit codes are not considered errors (err is nil)
//
// Security Considerations:
// - SSH private keys should have 600 permissions
// - Avoid hardcoding passwords in source code
// - Validate and sanitize command inputs to prevent injection
//
// Dependencies:
// - golang.org/x/crypto/ssh for SSH functionality
// - github.com/sirupsen/logrus for structured logging (logging only, no global init side effects)
package exec
