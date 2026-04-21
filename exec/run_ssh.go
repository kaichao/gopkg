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

	"github.com/kaichao/gopkg/errors"
	"golang.org/x/crypto/ssh"
)

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
		return 125, "", "", errors.WrapE(err, "ssh: create session failed")
	}
	defer func() {
		if session != nil {
			session.Close()
		}
	}()

	var stdoutBuf, stderrBuf *bytes.Buffer
	var wg *sync.WaitGroup
	var exitCode int

	if config.Background {
		wrappedCmd, marker := wrapCommand(command, config.UseHomeTmp)
		stdoutBuf, stderrBuf, wg = captureOutput(ctx, session)

		if err := session.Start(wrappedCmd); err != nil {
			// Clean up any processes that may have started
			_ = cleanupProcesses(client, command, marker)
			return 125, "", "", errors.WrapE(err, "start background command failed")
		}

		// For background mode, timeout only applies to command startup
		select {
		case <-ctx.Done():
			// Timeout during startup - clean up remote processes
			_ = cleanupProcesses(client, command, marker)
			return 124, "", "", errors.E("background command timed out during startup")
		default:
			// Continue to read PID marker
		}

		// Wait for background command to write PID marker
		wg.Wait()
		if stdoutBuf.Len() == 0 {
			_ = cleanupProcesses(client, command, marker)
			return 125, "", "", errors.E("empty background command output")
		}
		pidLine := strings.TrimSpace(stdoutBuf.String())
		lines := strings.Split(pidLine, "\n")
		if len(lines) == 0 {
			_ = cleanupProcesses(client, command, marker)
			return 125, "", "", errors.E("empty background command output")
		}
		pidLine = lines[0]
		fields := strings.Fields(pidLine)
		if len(fields) != 2 || !strings.HasPrefix(fields[1], "MARKER_") {
			_ = cleanupProcesses(client, command, marker)
			return 125, "", "", errors.E("invalid PID marker format, got: %q", pidLine)
		}
		// Return PID as stdout, zero exit code
		return 0, fields[1], "", nil
	}

	// Normal synchronous command execution
	stdoutBuf, stderrBuf, wg = captureOutput(ctx, session)
	if err := session.Start(command); err != nil {
		return 125, "", "", errors.WrapE(err, "start command failed")
	}

	// Wait for command completion with timeout
	done := make(chan error, 1)
	go func() {
		done <- session.Wait()
	}()

	select {
	case waitErr := <-done:
		wg.Wait()
		if waitErr != nil {
			if exitErr, ok := waitErr.(*ssh.ExitError); ok {
				exitCode = exitErr.ExitStatus()
			} else {
				exitCode = 125
				return exitCode, "", "", waitErr
			}
		}
		return exitCode, stdoutBuf.String(), stderrBuf.String(), nil
	case <-ctx.Done():
		// Timeout or cancellation
		if ctx.Err() == context.DeadlineExceeded {
			// Send SIGKILL to remote process and ensure cleanup
			_ = session.Signal(ssh.SIGKILL)
			_ = session.Close()
			wg.Wait() // Wait for output buffers to be fully read
			return 124, "", "", errors.E("command timed out")
		}
		return 125, "", "", ctx.Err()
	}
}

// createSSHClient creates SSH client and context for timeout management
func createSSHClient(config SSHConfig, timeout int) (*ssh.Client, context.Context, context.CancelFunc, error) {
	if config.Host == "" {
		return nil, nil, nil, errors.E("empty host in SSH config")
	}
	if config.User == "" {
		return nil, nil, nil, errors.E("empty user in SSH config")
	}
	if config.Port == 0 {
		config.Port = 22
	}

	authMethod, err := getAuthMethod(config)
	if err != nil {
		return nil, nil, nil, err
	}

	clientConfig := &ssh.ClientConfig{
		User:            config.User,
		Auth:            []ssh.AuthMethod{authMethod},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	ctx := context.Background()
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	}

	// Establish SSH connection with retry
	client, err := sshDialWithRetry(ctx, config, clientConfig, 3)
	if err != nil {
		// Clean up cancel function to avoid context leak
		if cancel != nil {
			cancel()
		}
		return nil, nil, nil, errors.WrapE(err, "ssh dial failed")
	}

	return client, ctx, cancel, nil
}

// wrapCommand wraps command with PID marker for background execution
func wrapCommand(command string, useHomeTmp bool) (string, string) {
	marker := fmt.Sprintf("MARKER_%d", time.Now().UnixNano())
	tmpDir := "/tmp"
	if useHomeTmp {
		if home, err := os.UserHomeDir(); err == nil {
			tmpDir = filepath.Join(home, "tmp")
		}
	}
	// Create wrapper script that:
	// 1. Uses temp directory based on useHomeTmp
	// 2. Creates a temporary script file
	// 3. Executes command in background with proper process management
	// 4. Returns actual PID and marker for validation
	wrapper := fmt.Sprintf(`
		tmp_dir="%s"
		if %t; then
			mkdir -p ${HOME}/tmp
			tmp_dir="${HOME}/tmp"
		fi
		tmp_script=$(mktemp -p "$tmp_dir")
		chmod +x "$tmp_script"
		
		cat > "$tmp_script" <<'EOF'
#!/bin/bash
# MARKER - used by tests to locate the process
(%s) &
real_pid=$!
echo "$real_pid" "$marker"
echo "$real_pid $marker" > "$tmp_dir/real_pid"
wait $real_pid
EOF

		nohup "$tmp_script" >/dev/null 2>&1 &
		wrapper_pid=$!
		disown $wrapper_pid
		sleep 0.5

		if [ -f "$tmp_dir/real_pid" ]; then
			real_pid=$(cat "$tmp_dir/real_pid")
		else
			real_pid=$wrapper_pid
		fi

		sleep 0.5
		if ps -p "$real_pid" >/dev/null 2>&1 || [ -d "/proc/$real_pid" ]; then
			echo "$real_pid"
			echo "$real_pid %s"
		else
			echo "0"
			echo "0 %s"
		fi

		rm -f "$tmp_script" "$tmp_dir/real_pid" 2>/dev/null || true
	`, tmpDir, useHomeTmp, command, marker, marker)
	return wrapper, marker
}

// captureOutput captures stdout and stderr from SSH session with DEBUG line filtering
func captureOutput(ctx context.Context, session *ssh.Session) (*bytes.Buffer, *bytes.Buffer, *sync.WaitGroup) {
	stdoutPipe, _ := session.StdoutPipe()
	stderrPipe, _ := session.StderrPipe()

	var stdoutBuf, stderrBuf bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(2)

	copyData := func(dest *bytes.Buffer, src io.Reader) {
		defer wg.Done()
		buf := make([]byte, 32*1024)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				n, err := src.Read(buf)
				if n > 0 {
					// Skip debug lines to reduce noise
					if !bytes.Contains(buf[:n], []byte("DEBUG:")) {
						dest.Write(buf[:n])
					}
				}
				if err != nil {
					return
				}
			}
		}
	}

	go copyData(&stdoutBuf, stdoutPipe)
	go copyData(&stderrBuf, stderrPipe)

	return &stdoutBuf, &stderrBuf, &wg
}

// defaultSSHKeyPath returns the default SSH key path, preferring id_ed25519 over id_rsa.
func defaultSSHKeyPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", errors.WrapE(err, "get home dir failed")
	}

	// Try common SSH key locations in order of preference
	keyPaths := []string{
		filepath.Join(homeDir, ".ssh", "id_ed25519"), // Preferred: more secure and faster
		filepath.Join(homeDir, ".ssh", "id_rsa"),     // Fallback: traditional RSA key
		filepath.Join(homeDir, ".ssh", "id_ecdsa"),   // Alternative: ECDSA key
	}

	for _, keyPath := range keyPaths {
		if _, err := os.Stat(keyPath); err == nil {
			return keyPath, nil
		}
	}

	return "", errors.E("no default SSH key found in %s/.ssh/ (tried: id_ed25519, id_rsa, id_ecdsa)", homeDir)
}

// getAuthMethod returns the appropriate SSH authentication method based on config
func getAuthMethod(config SSHConfig) (ssh.AuthMethod, error) {
	if config.KeyPath == "" {
		// Use default key path when KeyPath is explicitly empty
		keyPath, err := defaultSSHKeyPath()
		if err == nil {
			config.KeyPath = keyPath
		}
	}
	if config.KeyPath != "" {
		key, err := os.ReadFile(config.KeyPath)
		if err != nil {
			return nil, errors.WrapE(err, "read key file failed")
		}
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, errors.WrapE(err, "parse private key failed")
		}
		return ssh.PublicKeys(signer), nil
	}
	if config.Password != "" {
		return ssh.Password(config.Password), nil
	}
	return nil, errors.E("no authentication method provided")
}

// sshDialWithRetry establishes SSH connection with retry logic
func sshDialWithRetry(ctx context.Context, config SSHConfig, clientConfig *ssh.ClientConfig, attempts int) (*ssh.Client, error) {
	var client *ssh.Client
	var err error

	for i := 0; i < attempts; i++ {
		client, err = ssh.Dial("tcp", fmt.Sprintf("%s:%d", config.Host, config.Port), clientConfig)
		if err == nil {
			return client, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(500 * time.Millisecond):
			// Continue to next attempt
		}
	}
	return nil, err
}

// cleanupProcesses cleans up remote processes and temporary files associated with a command
func cleanupProcesses(client *ssh.Client, command string, marker string) error {
	if client == nil {
		return nil
	}

	cleanupCmds := []string{
		fmt.Sprintf("pkill -9 -f '%s' 2>/dev/null || true", command),
		fmt.Sprintf("pkill -9 -f '%s' 2>/dev/null || true", marker),
		"pkill -9 -f 'sleep' 2>/dev/null || true",
		"pkill -9 -f 'singularity' 2>/dev/null || true",
		// Clean up temporary files
		fmt.Sprintf("rm -rf /tmp/ssh_wrapper_* 2>/dev/null || true"),
		fmt.Sprintf("rm -rf ${HOME}/tmp/ssh_wrapper_* 2>/dev/null || true"),
	}

	for _, cmd := range cleanupCmds {
		session, err := client.NewSession()
		if err != nil {
			continue
		}
		_ = session.Run(cmd)
		session.Close()
		time.Sleep(100 * time.Millisecond)
	}
	return nil
}

// SSHConfig defines SSH connection parameters
type SSHConfig struct {
	User       string
	Host       string
	Port       int
	KeyPath    string // Path to private key file, empty for default (~/.ssh/id_rsa)
	Password   string // Optional, if using password auth
	Background bool   // If true, run command in background and return PID
	UseHomeTmp bool   // If true, use ${HOME}/tmp instead of /tmp for temporary files
}
