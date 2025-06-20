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
	User     string
	Host     string
	Port     int
	KeyPath  string // Path to private key file, empty for default (~/.ssh/id_rsa)
	Password string // Optional, if using password auth
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

// wrapCommand creates a wrapped command with process tracking
func wrapCommand(command string) (string, string) {
	marker := fmt.Sprintf("MARKER_%d", time.Now().UnixNano())
	wrapped := fmt.Sprintf(`bash -c '
		startup_output=$( (%s) 2>&1 )
		echo "$startup_output"
		nohup bash -c "%s" >/dev/null 2>&1 &
		pid=$!
		disown $pid
		echo "$pid %s"
	'`, command, strings.ReplaceAll(command, "\"", "\\\""), marker)
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

	wrappedCmd, marker := wrapCommand(command)
	stdoutBuf, stderrBuf, wg := captureOutput(session)

	if err := session.Start(wrappedCmd); err != nil {
		return 125, "", "", fmt.Errorf("start command failed: %v", err)
	}

	exitCode, err := executeCommand(session, ctx, client, stdoutBuf, stderrBuf, wg)
	if err != nil {
		_ = cleanupProcesses(client, command, marker)
		return exitCode, stdoutBuf.String(), stderrBuf.String(), err
	}

	return exitCode, stdoutBuf.String(), stderrBuf.String(), nil
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
