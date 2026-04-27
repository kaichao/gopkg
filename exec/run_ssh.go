package exec

import (
	"bufio"
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
		return 125, "", "", errors.WrapE(err, 125, "ssh: create session failed")
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
			return 125, "", "", errors.WrapE(err, 125, "start background command failed")
		}

		// Use a dedicated startup context (10s) for background commands.
		// The user's timeout context is for the real command lifetime, not
		// for the wrapper script that just starts it in the background.
		startupCtx, startupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer startupCancel()

		// Wait for the wrapper script to finish (it starts the real command in background,
		// prints PID, and exits). This ensures SSH pipes are closed so captureOutput
		// goroutines can read all data and exit.
		waitDone := make(chan error, 1)
		go func() {
			waitDone <- session.Wait()
		}()

		select {
		case <-waitDone:
			// Wrapper script finished normally, output should be available
		case <-startupCtx.Done():
			// Timeout during startup - clean up remote processes
			_ = cleanupProcesses(client, command, marker)
			return 124, "", "", errors.E(124, "background command timed out during startup")
		}

		// Wait for output goroutines to finish reading all data
		wg.Wait()
		if stdoutBuf.Len() == 0 {
			_ = cleanupProcesses(client, command, marker)
			return 125, "", "", errors.E(125, "empty background command output")
		}
		pidLine := strings.TrimSpace(stdoutBuf.String())
		lines := strings.Split(pidLine, "\n")
		if len(lines) == 0 {
			_ = cleanupProcesses(client, command, marker)
			return 125, "", "", errors.E(125, "empty background command output")
		}

		// Find the line with "PID MARKER_xxx" format (the second line from wrapCommand)
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) == 2 && strings.HasPrefix(fields[1], "MARKER_") {
				// Return the actual PID as stdout, zero exit code
				return 0, fields[0], "", nil
			}
		}

		_ = cleanupProcesses(client, command, marker)
		return 125, "", "", errors.E(125, fmt.Sprintf("invalid PID marker format, got: %q", pidLine))
	}

	// Normal synchronous command execution
	stdoutBuf, stderrBuf, wg = captureOutput(ctx, session)
	if err := session.Start(command); err != nil {
		return 125, "", "", errors.WrapE(err, 125, "start command failed")
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
				return exitCode, "", "", errors.WrapE(waitErr, 125, "unexpected command error")
			}
		}
		return exitCode, stdoutBuf.String(), stderrBuf.String(), nil
	case <-ctx.Done():
		// Timeout or cancellation
		if ctx.Err() == context.DeadlineExceeded {
			// Send SIGKILL to remote process
			_ = session.Signal(ssh.SIGKILL)
			// Wait briefly for the remote process to die and pipes to drain,
			// so we can capture any buffered output before closing the session.
			select {
			case <-done:
				// Command exited due to signal
			case <-time.After(500 * time.Millisecond):
				// Grace period for pipe draining
			}
			wg.Wait() // Wait for output goroutines to finish reading
			stdout := stdoutBuf.String()
			stderr := stderrBuf.String()
			_ = session.Close()
			return 124, stdout, stderr, errors.E(124, "command timed out")
		}
		return 125, "", "", errors.WrapE(ctx.Err(), 125, "context cancelled")
	}
}

// createSSHClient creates SSH client and context for timeout management
func createSSHClient(config SSHConfig, timeout int) (*ssh.Client, context.Context, context.CancelFunc, error) {
	if config.Host == "" {
		return nil, nil, nil, errors.E(125, "empty host in SSH config")
	}
	if config.User == "" {
		return nil, nil, nil, errors.E(125, "empty user in SSH config")
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
		return nil, nil, nil, errors.WrapE(err, 125, "ssh dial failed")
	}

	return client, ctx, cancel, nil
}

// wrapCommand wraps command with PID marker for background execution.
// Uses `nohup` directly in the SSH command line (no temp script, no
// heredoc, no quoting issues). The command backgrounds the real command
// with nohup, then immediately echoes its PID and exit code.
// session.Wait() returns immediately after the echo.
// If useHomeTmp is true, output is redirected to ${HOME}/tmp/nohup.out
// instead of /dev/null, allowing debugging of background command output.
func wrapCommand(command string, useHomeTmp bool) (string, string) {
	marker := fmt.Sprintf("MARKER_%d", time.Now().UnixNano())
	var wrapper string
	if useHomeTmp {
		wrapper = fmt.Sprintf(
			"mkdir -p ${HOME}/tmp; nohup %s >${HOME}/tmp/nohup.out 2>&1 & echo \"$! %s\"",
			command, marker)
	} else {
		wrapper = fmt.Sprintf(
			"nohup %s >/dev/null 2>&1 & echo \"$! %s\"",
			command, marker)
	}
	return wrapper, marker
}

// captureOutput captures stdout and stderr from SSH session with DEBUG line filtering.
// Reading is done via bufio.Scanner which handles line boundaries and properly
// terminates when the pipe is closed (on session end/cancel).
func captureOutput(ctx context.Context, session *ssh.Session) (*bytes.Buffer, *bytes.Buffer, *sync.WaitGroup) {
	stdoutPipe, err := session.StdoutPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "capture stdout pipe failed: %v\n", err)
		stdoutPipe = nil
	}
	stderrPipe, err := session.StderrPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "capture stderr pipe failed: %v\n", err)
		stderrPipe = nil
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(2)

	copyData := func(dest *bytes.Buffer, src io.Reader) {
		defer wg.Done()
		scanner := bufio.NewScanner(src)
		for scanner.Scan() {
			line := scanner.Bytes()
			// Skip debug lines to reduce noise
			if !bytes.Contains(line, []byte("DEBUG:")) {
				dest.Write(line)
				dest.WriteByte('\n')
			}
		}
	}

	if stdoutPipe != nil {
		go copyData(&stdoutBuf, stdoutPipe)
	} else {
		wg.Done()
	}
	if stderrPipe != nil {
		go copyData(&stderrBuf, stderrPipe)
	} else {
		wg.Done()
	}

	return &stdoutBuf, &stderrBuf, &wg
}

// defaultSSHKeyPath returns the default SSH key path, preferring id_ed25519 over id_rsa.
func defaultSSHKeyPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", errors.WrapE(err, 125, "get home dir failed")
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

	return "", fmt.Errorf("no default SSH key found in %s/.ssh/ (tried: id_ed25519, id_rsa, id_ecdsa)", homeDir)
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
			return nil, errors.WrapE(err, 125, "read key file failed")
		}
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, errors.WrapE(err, 125, "parse private key failed")
		}
		return ssh.PublicKeys(signer), nil
	}
	if config.Password != "" {
		return ssh.Password(config.Password), nil
	}
	return nil, errors.E(125, "no authentication method provided")
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

// cleanupProcesses cleans up remote processes and temporary files
// associated with a specific command and marker on the remote host.
// Uses marker-specific pkill to avoid killing unrelated processes.
func cleanupProcesses(client *ssh.Client, command string, marker string) error {
	if client == nil {
		return nil
	}

	// Only kill processes matching the marker or the wrapper script
	cleanupCmds := []string{
		fmt.Sprintf("pkill -9 -f '%s' 2>/dev/null || true", marker),
		fmt.Sprintf("rm -f /tmp/real_pid ${HOME}/tmp/real_pid 2>/dev/null || true"),
	}

	if command != "" {
		cleanupCmds = append([]string{fmt.Sprintf("pkill -9 -f '%s' 2>/dev/null || true", command)}, cleanupCmds...)
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
