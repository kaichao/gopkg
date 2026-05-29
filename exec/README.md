# exec

[![Go Reference](https://pkg.go.dev/badge/github.com/kaichao/gopkg/exec.svg)](https://pkg.go.dev/github.com/kaichao/gopkg/exec)

`exec` provides cross-environment command execution utilities for local and remote SSH environments.

## Features

- **Unified Interface**: Same API for local and remote SSH execution
- **Full Output Capture**: Synchronously captures stdout, stderr and exit code
- **Flexible Timeout**: Supports both command-level and connection-level timeouts
- **Multiple Auth Methods**: SSH supports key, password and agent forwarding
- **Process Management**: Background process and process group support
- **Circular Buffering**: 10MB output limit with circular buffer for large outputs

## Installation

```bash
go get github.com/kaichao/gopkg/exec
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"

    "github.com/kaichao/gopkg/errors"
    "github.com/kaichao/gopkg/exec"
)

func main() {
    // Local execution — exit code embedded in error
    stdout, stderr, err := exec.RunReturnAll("ls -l /tmp", 10)
    if err != nil {
        log.Printf("Command failed: %v (code=%d)", err, errors.GetCode(err))
    }
    fmt.Println(stdout)
}
```

## API

### Key Functions

```go
// Local execution — exit code embedded in error, use errors.GetCode(err)
func RunReturnAll(command string, timeout int) (stdout string, stderr string, err error)

// SSH execution — exit code embedded in error, use errors.GetCode(err)
func RunSSHCommand(config SSHConfig, command string, timeout int) (stdout string, stderr string, err error)

// Retry wrapper
func RunWithRetries(cmd string, numRetries int, timeout int) (int, error)
```

### SSH Configuration

```go
type SSHConfig struct {
    User       string // Required: SSH username
    Host       string // Required: SSH host
    Port       int    // Optional: SSH port (default: 22)
    KeyPath    string // Optional: Path to SSH private key (prefers id_ed25519 > id_rsa > id_ecdsa)
    Password   string // Optional: SSH password
    Background bool   // Optional: Run command in background mode, returns PID
    UseHomeTmp bool   // Optional: Use ${HOME}/tmp instead of /tmp for temporary files
}
```

## Exit Code Convention

- `0`: Command executed successfully
- `124`: Command timed out
- `125`: Command execution failed (e.g., pipe creation, process start)
- Other non-zero: Command-specific exit code
- `128 + signal`: Command terminated by signal (e.g., SIGKILL = 128+9 = 137)

Exit code is embedded in the returned error. Use `errors.GetCode(err)` to retrieve it.

## SSH Usage

```go
config := exec.SSHConfig{
    Host:    "10.0.0.1",
    User:    "admin",
    KeyPath: "/home/user/.ssh/id_ed25519",
}

// Synchronous
stdout, stderr, err := exec.RunSSHCommand(config, "docker ps -a", 30)

// Background (returns PID)
config.Background = true
pid, _, err := exec.RunSSHCommand(config, "long-running-command", 0)
```

## Documentation

For complete documentation see:
- [Package Documentation](https://pkg.go.dev/github.com/kaichao/gopkg/exec)
- [doc.go](./doc.go) - Detailed API reference
- [examples/](./examples/) - Working examples

## License

MIT License
