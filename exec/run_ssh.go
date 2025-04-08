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

	"github.com/sirupsen/logrus"
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

// RunSSHCommand executes command via SSH with full lifecycle management
//
// Steps:
// 1. Establish SSH connection with authentication
// 2. Create session and setup I/O pipes
// 3. Start command with process group isolation
// 4. Handle timeout/cancellation signals
// 5. Cleanup resources (connections, temp files)
//
// Edge cases:
// - Network interruptions during execution
// - Malformed remote command parsing
// - Permission denied errors
func RunSSHCommand(config SSHConfig, command string, timeout int) (int, string, string, error) {
	// 处理密钥路径
	var keyPath string
	if config.KeyPath != "" {
		keyPath = config.KeyPath
	} else {
		var err error
		keyPath, err = DefaultSSHKeyPath()
		if err != nil {
			return 125, "", "", err
		}
	}

	// SSH 认证
	var authMethod ssh.AuthMethod
	if keyPath != "" {
		key, err := os.ReadFile(keyPath)
		if err != nil {
			return 125, "", "", fmt.Errorf("read key file failed: %v", err)
		}
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return 125, "", "", fmt.Errorf("parse private key failed: %v", err)
		}
		authMethod = ssh.PublicKeys(signer)
	} else if config.Password != "" {
		authMethod = ssh.Password(config.Password)
	} else {
		return 125, "", "", fmt.Errorf("no authentication method provided")
	}

	// SSH 客户端配置
	clientConfig := &ssh.ClientConfig{
		User:            config.User,
		Auth:            []ssh.AuthMethod{authMethod},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// 超时控制
	baseCtx := context.Background()
	ctx := baseCtx
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(baseCtx, time.Duration(timeout)*time.Second)
		defer cancel()
	}

	// 连接 SSH
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", config.Host, config.Port), clientConfig)
	if err != nil {
		return 125, "", "", fmt.Errorf("ssh dial failed: %w", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return 125, "", "", fmt.Errorf("ssh: create session failed: %w", err)
	}
	defer session.Close()

	// ✅ 生成唯一标识符（便于测试 pgrep）
	marker := fmt.Sprintf("MARKER_%d", time.Now().UnixNano())
	cmdWithMarker := fmt.Sprintf("%s; # %s", command, marker)

	// ✅ 包装命令：setsid + 记录 PID
	// Embed the marker in an environment variable or argument that is likely to show up in process listings.
	wrappedCmd := fmt.Sprintf("bash -c 'export MARKER=%s; setsid bash -c \"%s\" & echo $! > /tmp/ssh_cmd_pid_%d; wait'",
		marker, strings.ReplaceAll(cmdWithMarker, "\"", "\\\""), os.Getpid())

	// 捕获输出
	stdoutPipe, _ := session.StdoutPipe()
	stderrPipe, _ := session.StderrPipe()
	var stdoutBuf, stderrBuf bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); _, _ = io.Copy(&stdoutBuf, stdoutPipe) }()
	go func() { defer wg.Done(); _, _ = io.Copy(&stderrBuf, stderrPipe) }()

	// 启动命令
	if err := session.Start(wrappedCmd); err != nil {
		return 125, "", "", fmt.Errorf("start command failed: %v", err)
	}

	pidFile := fmt.Sprintf("/tmp/ssh_cmd_pid_%d", os.Getpid())

	// ✅ 超时清理逻辑
	done := make(chan struct{})
	var waitErr error
	go func() {
		waitErr = session.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			// ✅ 主动关闭会话连接
			_ = session.Signal(ssh.SIGKILL)
			_ = session.Close()

			// ✅ 远程 kill
			cleanupRemoteProcess(clientConfig, config, pidFile)

			// ✅ 等待输出收集完毕
			wg.Wait()
			cleanupPidFile(clientConfig, config, pidFile)
			return 124, stdoutBuf.String(), stderrBuf.String(), fmt.Errorf("command timed out")
		}
	case <-done:
		// 命令执行完毕
	}
	wg.Wait()

	// 处理退出码
	var exitCode int
	if waitErr != nil {
		if exitErr, ok := waitErr.(*ssh.ExitError); ok {
			exitCode = exitErr.ExitStatus()
		} else {
			exitCode = 125
			err = waitErr
		}
	} else {
		exitCode = 0
	}

	// 清理 PID 文件
	cleanupPidFile(clientConfig, config, pidFile)

	return exitCode, stdoutBuf.String(), stderrBuf.String(), err
}

// cleanupRemoteProcess kills the remote process group based on PID file.
func cleanupRemoteProcess(clientConfig *ssh.ClientConfig, config SSHConfig, pidFile string) {
	client, err := retryDial(config, clientConfig, 3) // 传入 clientConfig
	if err != nil {
		logrus.Errorf("cleanup: ssh dial failed: %v", err)
		return
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		logrus.Errorf("cleanup: create session failed: %v", err)
		return
	}
	defer session.Close()

	cleanupCmd := fmt.Sprintf(`
        if [ -f %s ]; then
            pgid=$(cat %s | xargs -I{} ps -o pgid= {} | grep -o '[0-9]*');
            kill -TERM -$pgid 2>/dev/null || kill -KILL -$pgid 2>/dev/null;
        fi`, pidFile, pidFile)

	if err := session.Run(cleanupCmd); err != nil {
		logrus.Errorf("cleanup: failed to kill process group: %v", err)
	}
}

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

// cleanupPidFile removes the temporary PID file.
func cleanupPidFile(clientConfig *ssh.ClientConfig, config SSHConfig, pidFile string) {
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", config.Host, config.Port), clientConfig)
	if err != nil {
		logrus.Errorf("cleanup pid file: ssh dial failed: %v", err)
		return
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		logrus.Errorf("cleanup pid file: create session failed: %v", err)
		return
	}
	defer session.Close()

	rmCmd := fmt.Sprintf("rm -f %s", pidFile)
	if err := session.Run(rmCmd); err != nil {
		logrus.Errorf("cleanup pid file: failed to remove %s: %v", pidFile, err)
	}
}
