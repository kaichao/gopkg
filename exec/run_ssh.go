package exec

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHConfig defines SSH connection parameters.
type SSHConfig struct {
	User       string
	Host       string
	Port       int
	KeyPath    string // Path to private key file, empty for default (~/.ssh/id_rsa)
	Password   string // Optional, if using password auth
	Background bool   // If true, run command in background and return PID
}

// DefaultSSHKeyPath returns the default SSH key path (~/.ssh/id_rsa) if it exists.
func DefaultSSHKeyPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir failed: %v", err)
	}
	keyPath := filepath.Join(homeDir, ".ssh", "id_rsa")
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		return "", fmt.Errorf("default key path %s does not exist", keyPath)
	}
	return keyPath, nil
}

// getAuthMethod returns the appropriate SSH authentication method based on config
func getAuthMethod(config SSHConfig) (ssh.AuthMethod, error) {
	if config.KeyPath != "" {
		key, err := os.ReadFile(config.KeyPath)
		if err != nil {
			return nil, fmt.Errorf("read key file failed: %v", err)
		}
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("parse private key failed: %v", err)
		}
		return ssh.PublicKeys(signer), nil
	}
	if config.Password != "" {
		return ssh.Password(config.Password), nil
	}
	return nil, fmt.Errorf("no authentication method provided")
}

// createSSHClient creates a new SSH client with timeout handling
func createSSHClient(config SSHConfig, timeout int) (*ssh.Client, context.Context, context.CancelFunc, error) {
	authMethod, err := getAuthMethod(config)
	if err != nil {
		return nil, nil, nil, err
	}

	clientConfig := &ssh.ClientConfig{
		User:            config.User,
		Auth:            []ssh.AuthMethod{authMethod},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	ctx := context.Background()
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", config.Host, config.Port), clientConfig)
	if err != nil {
		if cancel != nil {
			cancel()
		}
		return nil, nil, nil, fmt.Errorf("ssh dial failed: %w", err)
	}
	return client, ctx, cancel, nil
}

// wrapCommand creates a wrapped command for background execution
func wrapCommand(command string) (string, string) {
	marker := fmt.Sprintf("MARKER_%d", time.Now().UnixNano())
	// Simplified and reliable background execution
	wrapped := `
		# Enhanced background execution with process tracking
		tmp_script=$(mktemp)
		chmod +x $tmp_script
		cat > $tmp_script <<'EOF'
#!/bin/bash
# Prevent process termination
trap '' HUP INT TERM
# Execute command in subshell
(` + command + `) &
real_pid=$!
echo "DEBUG: Real PID: $real_pid" >&2
ps -fp $real_pid >&2
echo $real_pid > /tmp/real_pid
wait $real_pid
EOF
		nohup $tmp_script >/tmp/nohup.out 2>&1 &
		pid=$!
		disown $pid
		echo "DEBUG: Wrapper PID: $pid" >&2
		sleep 1
		if [ -f /tmp/real_pid ]; then
			pid=$(cat /tmp/real_pid)
			echo "DEBUG: Using real PID: $pid" >&2
		fi
		
		# Enhanced process verification
		sleep 1
		# Check process in multiple ways
		if ps -p $pid >/dev/null 2>&1 || \
		   [ -d /proc/$pid ] || \
		   pgrep -P $pid >/dev/null 2>&1; then
			# Output clean results (skip debug info)
			grep -v '^DEBUG:' /tmp/nohup.out || true
			# Then output PID info
			echo "$pid"
			echo "$pid ` + marker + `"
			# Additional verification
			echo "Process verified by:" >&2
			ps -fp $pid >&2 || true
			[ -d /proc/$pid ] && echo "/proc/$pid exists" >&2 || true
		else
			echo "Process failed to start"
			cat /tmp/nohup.out
			echo "0"
			echo "0 ` + marker + `"
		fi
		
		# Cleanup
		rm -f $tmp_script /tmp/nohup.out
		exit 0`
	return wrapped, marker
}

// captureOutput captures stdout/stderr from session
func captureOutput(session *ssh.Session) (*bytes.Buffer, *bytes.Buffer, *sync.WaitGroup) {
	stdoutPipe, _ := session.StdoutPipe()
	stderrPipe, _ := session.StderrPipe()
	var stdoutBuf, stderrBuf bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); _, _ = io.Copy(&stdoutBuf, stdoutPipe) }()
	go func() { defer wg.Done(); _, _ = io.Copy(&stderrBuf, stderrPipe) }()
	return &stdoutBuf, &stderrBuf, &wg
}

// cleanupProcesses performs thorough cleanup of remote processes
func cleanupProcesses(client *ssh.Client, command string, marker string) error {
	cleanupCmds := []string{
		fmt.Sprintf("pkill -9 -f '%s'", command),
		fmt.Sprintf("pkill -9 -f '%s'", marker),
		"pkill -9 -f 'sleep'",
		"pkill -9 -f 'singularity'",
	}

	for _, cmd := range cleanupCmds {
		session, _ := client.NewSession()
		_ = session.Run(cmd)
		session.Close()
		time.Sleep(300 * time.Millisecond)
	}
	return nil
}

// executeCommand runs the command and handles timeout
func executeCommand(session *ssh.Session, ctx context.Context, client *ssh.Client,
	stdoutBuf *bytes.Buffer, stderrBuf *bytes.Buffer, wg *sync.WaitGroup) (int, error) {
	done := make(chan struct{})
	var waitErr error

	go func() {
		waitErr = session.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			_ = session.Signal(ssh.SIGTERM)
			time.Sleep(500 * time.Millisecond)
			_ = session.Signal(ssh.SIGKILL)
			_ = session.Close()
			wg.Wait()
			return 124, fmt.Errorf("command timed out")
		}
	case <-done:
		wg.Wait()
	}

	if waitErr != nil {
		if exitErr, ok := waitErr.(*ssh.ExitError); ok {
			return exitErr.ExitStatus(), nil
		}
		return 125, waitErr
	}
	return 0, nil
}

// RunSSHCommand executes command via SSH with full lifecycle management
func RunSSHCommand(config SSHConfig, command string, timeout int) (int, string, string, error) {
	client, ctx, cancel, err := createSSHClient(config, timeout)
	if err != nil {
		return 125, "", "", err
	}
	defer client.Close()
	if cancel != nil {
		defer cancel()
	}

	session, err := client.NewSession()
	if err != nil {
		return 125, "", "", fmt.Errorf("ssh: create session failed: %w", err)
	}
	defer session.Close()

	var stdoutBuf, stderrBuf *bytes.Buffer
	var wg *sync.WaitGroup
	var exitCode int

	if config.Background {
		wrappedCmd, marker := wrapCommand(command)
		stdoutBuf, stderrBuf, wg = captureOutput(session)

		if err := session.Start(wrappedCmd); err != nil {
			return 125, "", "", fmt.Errorf("start background command failed: %v", err)
		}

		// For background mode, timeout only applies to command startup
		exitCode, err = executeCommand(session, ctx, client, stdoutBuf, stderrBuf, wg)
		if err != nil {
			_ = cleanupProcesses(client, command, marker)
			return exitCode, stdoutBuf.String(), stderrBuf.String(), err
		}

		// Parse stdout for PID marker
		output := stdoutBuf.String()
		lines := strings.Split(strings.TrimSpace(output), "\n")
		if len(lines) == 0 {
			return 125, "", "", fmt.Errorf("empty background command output")
		}

		// Last line should be PID and marker
		pidLine := lines[len(lines)-1]
		fields := strings.Fields(pidLine)
		if len(fields) != 2 || !strings.HasPrefix(fields[1], "MARKER_") {
			return 125, "", "", fmt.Errorf("invalid PID marker format, got: %q", pidLine)
		}

		pidOutput := fields[0]
		startupOutput := ""
		if len(lines) > 1 {
			startupOutput = strings.Join(lines[:len(lines)-1], "\n")
		}

		return 0, pidOutput, startupOutput, nil
	} else {
		stdoutBuf, stderrBuf, wg = captureOutput(session)
		if err := session.Start(command); err != nil {
			return 125, "", "", fmt.Errorf("start command failed: %v", err)
		}
		exitCode, err = executeCommand(session, ctx, client, stdoutBuf, stderrBuf, wg)
		if err != nil {
			return exitCode, stdoutBuf.String(), stderrBuf.String(), err
		}
		return exitCode, stdoutBuf.String(), stderrBuf.String(), nil
	}
}

// retryDial attempts to establish SSH connection with retries
func retryDial(config SSHConfig, clientConfig *ssh.ClientConfig, attempts int) (*ssh.Client, error) {
	var client *ssh.Client
	var err error
	for i := 0; i < attempts; i++ {
		client, err = ssh.Dial("tcp", fmt.Sprintf("%s:%d", config.Host, config.Port), clientConfig)
		if err == nil {
			return client, nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return nil, err
}
