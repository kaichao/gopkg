package exec

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/kaichao/gopkg/errors"
	"github.com/sirupsen/logrus"
)

// RunReturnAll executes a command and returns its exit code, stdout, stderr, and any error.
//
// Params:
//   - command: the command string to execute
//   - timeout: timeout in seconds (0 for no timeout)
//
// Returns: (exitCode, stdout, stderr, err)
//   - exitCode: command exit code (0 for success, non-zero for command failure or timeout)
//   - stdout: standard output
//   - stderr: standard error
//   - err: error encountered during execution (e.g., pipe creation failure, command start failure, timeout). If command exits with non-zero exit code, err is nil
//   - If pipe creation or command start fails, returns exit code 125 and specific error
//   - In timeout case, returns exit code 124 and err = "command timed out"
//   - If command ends with non-zero exit code, returns that exit code and err = nil
//   - Other unexpected errors returned via err with exit code 125
func RunReturnAll(command string, timeout int) (int, string, string, error) {
	if command == "" {
		return 125, "", "", errors.E(125, "start command failed: empty command")
	}

	baseCtx := context.Background()
	ctx := baseCtx
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(baseCtx, time.Duration(timeout)*time.Second)
		defer cancel()
	}

	// Create command with process group support
	// Enable strict mode in bash and clean up only child processes on EXIT while preserving original exit code
	// Note: Add "|| true" to pkill to avoid failure (no child processes) interrupting trap
	bashCmd := command
	if os.Getenv("STRICT_BASH_MODE") == "yes" {
		bashCmd = `
			set -euo pipefail
			trap 'rc=$?; echo "[cleanup] bash exit rc=$rc" >&2; pkill -TERM -P $$ || true; exit $rc' EXIT
		` + command
	}
	cmd := exec.CommandContext(ctx, "/bin/bash", "-c", bashCmd)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Get output pipes
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return 125, "", "", errors.WrapE(err, 125, "capture stdout pipe failed")
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return 125, "", "", errors.WrapE(err, 125, "capture stderr pipe failed")
	}

	// Use circular buffer to capture output
	const maxOutputSize = 10 * 1024 * 1024 // 10MB
	stdoutBuf := NewCircularBuffer(maxOutputSize)
	stderrBuf := NewCircularBuffer(maxOutputSize)

	// Capture output asynchronously
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		_, err := io.Copy(stdoutBuf, stdoutPipe)
		if err != nil && !errors.Is(err, os.ErrClosed) {
			logrus.Errorf("copy stdout failed: %v", err)
		}
	}()
	go func() {
		defer wg.Done()
		_, err := io.Copy(stderrBuf, stderrPipe)
		if err != nil && !errors.Is(err, os.ErrClosed) {
			logrus.Errorf("copy stderr failed: %v", err)
		}
	}()

	// Start command
	if err := cmd.Start(); err != nil {
		return 125, "", "", errors.WrapE(err, 125, "start command failed")
	}

	// Terminate process group after timeout
	if timeout > 0 {
		go func() {
			<-ctx.Done()
			if ctx.Err() == context.DeadlineExceeded && cmd.Process != nil {
				syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			}
		}()
	}

	// Wait for command to finish
	waitErr := cmd.Wait()
	// Ensure output copying is complete
	wg.Wait()

	// Get data from buffers
	stdoutBytes := stdoutBuf.Bytes()
	stderrBytes := stderrBuf.Bytes()

	if waitErr == nil {
		return 0, string(stdoutBytes), string(stderrBytes), nil
	}

	// waitErr != nil, handle exit code and errors
	var exitCode int
	var retErr error
	if ctx.Err() == context.DeadlineExceeded {
		exitCode = 124
		retErr = errors.E(124, "command timed out")
	} else if exitErr, ok := waitErr.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
		// Handle signal termination
		if exitCode == -1 {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				if status.Signaled() {
					exitCode = 128 + int(status.Signal())
				}
			}
		}
		// Command ended with non-zero exit code, not an error
		retErr = nil
	} else {
		exitCode = 125
		retErr = errors.WrapE(waitErr, 125, "unexpected command error")
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

// RunWithRetries executes a command up to numRetries times until success.
// Returns 0 on success, or the last exit code if all retries are exhausted.
// An error is returned if RunReturnExitCode encounters a non-exit-code error (e.g., timeout).
func RunWithRetries(cmd string, numRetries int, timeout int) (int, error) {
	delay := 10 * time.Second
	var lastCode int
	for i := 0; i < numRetries; i++ {
		code, err := RunReturnExitCode(cmd, timeout)
		if err != nil {
			return code, err
		}
		if code == 0 {
			return code, nil
		}
		lastCode = code
		fmt.Printf("num-of-retries:%d,cmd=%s\n", i+1, cmd)
		time.Sleep(delay)
		delay *= 2
		timeout *= 2
	}
	return lastCode, nil
}

// CircularBuffer implements a fixed-size circular buffer, safe for concurrent use.
type CircularBuffer struct {
	mu     sync.RWMutex
	buf    []byte
	size   int
	offset int
	full   bool
}

// NewCircularBuffer creates a new circular buffer
func NewCircularBuffer(size int) *CircularBuffer {
	return &CircularBuffer{
		buf:  make([]byte, size),
		size: size,
	}
}

// Write writes data to the circular buffer, overwriting oldest data when exceeding capacity
func (c *CircularBuffer) Write(p []byte) (n int, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
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

// Bytes returns the latest data from the buffer
func (c *CircularBuffer) Bytes() []byte {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if !c.full {
		return c.buf[:c.offset]
	}
	// Reconstruct buffer to return the latest 10MB of data
	result := make([]byte, c.size)
	copy(result, c.buf[c.offset:])
	copy(result[c.size-c.offset:], c.buf[:c.offset])
	return result
}
