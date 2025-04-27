package exec

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
)

// RunReturnAll executes a command and returns its exit code, stdout, stderr, and any error.
//
// Params:
//   - command: the command string to execute
//   - timeout: timeout in seconds (0 for no timeout)
//
// Returns: (exitCode, stdout, stderr, err)
//   - exitCode：命令的退出码（0 表示成功，非零表示命令失败或超时等）
//   - stdout：标准输出
//   - stderr：标准错误
//   - err：执行过程中遇到的错误（如管道创建失败、命令启动失败、超时等）。若命令以非零退出码结束，err 为 nil
//   - 管道创建或命令启动失败时，返回退出码 125 和具体的 error
//   - 超时情况下，返回退出码 124 和 err = "command timed out"
//   - 命令以非零退出码结束时，返回该退出码，err 为 nil
//   - 其他未预期的错误通过 err 返回，退出码为 125
func RunReturnAll(command string, timeout int) (int, string, string, error) {
	if command == "" {
		return 125, "", "", fmt.Errorf("start command failed: empty command")
	}

	baseCtx := context.Background()
	ctx := baseCtx
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(baseCtx, time.Duration(timeout)*time.Second)
		defer cancel()
	}

	// 创建命令并支持进程组终止
	// cmd := exec.CommandContext(ctx, "/bin/bash", "-c", command)
	// 在 bash 中启用严格模式，并在 ERR/EXIT 时触发清理（例如终止整个进程组）
	cmd := exec.CommandContext(ctx, "/bin/bash", "-c",
		"set -euo pipefail; "+
			"trap 'echo \"[cleanup] bash exit code $? at line $LINENO\" >&2; "+
			"kill -TERM -$$' ERR EXIT; "+
			command)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// 获取输出管道
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return 125, "", "", fmt.Errorf("capture stdout pipe failed: %v", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return 125, "", "", fmt.Errorf("capture stderr pipe failed: %v", err)
	}

	// 使用环形缓冲区捕获输出
	const maxOutputSize = 10 * 1024 * 1024 // 10MB
	stdoutBuf := NewCircularBuffer(maxOutputSize)
	stderrBuf := NewCircularBuffer(maxOutputSize)

	// 同时将输出写入 os.Stdout/os.Stderr 和环形缓冲区
	stdoutWriter := io.MultiWriter(os.Stdout, stdoutBuf)
	stderrWriter := io.MultiWriter(os.Stderr, stderrBuf)

	// 异步捕获输出
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		_, err := io.Copy(stdoutWriter, stdoutPipe)
		if err != nil && !errors.Is(err, os.ErrClosed) {
			logrus.Errorf("copy stdout failed: %v", err)
		}
	}()
	go func() {
		defer wg.Done()
		_, err := io.Copy(stderrWriter, stderrPipe)
		if err != nil && !errors.Is(err, os.ErrClosed) {
			logrus.Errorf("copy stderr failed: %v", err)
		}
	}()

	// 超时后终止进程组
	if timeout > 0 {
		go func() {
			<-ctx.Done()
			if ctx.Err() == context.DeadlineExceeded && cmd.Process != nil {
				syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			}
		}()
	}

	// 启动命令
	if err := cmd.Start(); err != nil {
		return 125, "", "", fmt.Errorf("start command failed: %v", err)
	}

	// 等待命令结束
	waitErr := cmd.Wait()
	// 确保输出复制完成
	wg.Wait()

	// 获取缓冲区中的数据
	stdoutBytes := stdoutBuf.Bytes()
	stderrBytes := stderrBuf.Bytes()

	if waitErr == nil {
		return 0, string(stdoutBytes), string(stderrBytes), nil
	}

	// waitErr != nil, 处理退出码和错误
	var exitCode int
	var retErr error
	if ctx.Err() == context.DeadlineExceeded {
		exitCode = 124
		retErr = fmt.Errorf("command timed out")
	} else if exitErr, ok := waitErr.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
		// 处理信号终止
		if exitCode == -1 {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				if status.Signaled() {
					exitCode = 128 + int(status.Signal())
				}
			}
		}
		// 命令以非零退出码结束，不是错误
		retErr = nil
	} else {
		exitCode = 125
		retErr = waitErr
	}
	return exitCode, string(stdoutBytes), string(stderrBytes), retErr
}

// RunReturnExitCode ...
func RunReturnExitCode(command string, timeout int) (int, error) {
	code, stdout, stderr, err := RunReturnAll(command, timeout)
	fmt.Printf("exec command:%s\n stdout:\n%s\n", command, stdout)
	fmt.Fprintf(os.Stderr, "exec command: %s\n stderr:\n%s\n", command, stderr)
	return code, err
}

// RunReturnStdout ...
func RunReturnStdout(command string, timeout int) (string, error) {
	code, stdout, stderr, err := RunReturnAll(command, timeout)
	if code != 0 {
		fmt.Fprintf(os.Stderr, "exec command:%s\nexit-code=%d\n", command, code)
		// stdout = ""
	}
	fmt.Fprintf(os.Stderr, "exec command:\n%s\n%s\n", command, stderr)

	// remove leading/tail space
	return strings.TrimSpace(stdout), err
}

// RunWithRetries ...
func RunWithRetries(cmd string, numRetries int, timeout int) int {
	delay := 10 * time.Second
	var code int
	for i := 0; i < numRetries; i++ {
		code, _ = RunReturnExitCode(cmd, timeout)
		if code == 0 {
			return code
		}
		fmt.Printf("num-of-retries:%d,cmd=%s\n", i+1, cmd)
		time.Sleep(delay)
		delay *= 2
		timeout *= 2
	}
	return code
}

// CircularBuffer 实现固定大小的环形缓冲区
type CircularBuffer struct {
	buf    []byte
	size   int
	offset int
	full   bool
}

// NewCircularBuffer 创建一个新的环形缓冲区
func NewCircularBuffer(size int) *CircularBuffer {
	return &CircularBuffer{
		buf:  make([]byte, size),
		size: size,
	}
}

// Write 写入数据到环形缓冲区，超出部分覆盖最早数据
func (c *CircularBuffer) Write(p []byte) (n int, err error) {
	n = len(p)
	for len(p) > 0 {
		chunk := len(p)
		remaining := c.size - c.offset
		if chunk > remaining {
			chunk = remaining
		}
		copy(c.buf[c.offset:], p[:chunk])
		c.offset = (c.offset + chunk) % c.size
		if c.offset == 0 {
			c.full = true
		}
		p = p[chunk:]
	}
	return n, nil
}

// Bytes 返回缓冲区中的最新数据
func (c *CircularBuffer) Bytes() []byte {
	if !c.full {
		return c.buf[:c.offset]
	}
	// 重构缓冲区，返回最新的 10MB 数据
	result := make([]byte, c.size)
	copy(result, c.buf[c.offset:])
	copy(result[c.size-c.offset:], c.buf[:c.offset])
	return result
}
