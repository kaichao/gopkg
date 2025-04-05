# exec

[中文](README.zh.md) | English

Cross-environment command execution toolkit for Go, supporting both local and SSH remote execution with full output capture.

## Features

- **Unified Interface**: Consistent API for local and remote execution
- **Output Capture**: Simultaneous stream handling for stdout/stderr
- **Timeout Control**: Configurable execution timeout with process cleanup
- **SSH Support**: Key-based and password authentication

## Use Cases

- Bulk operations across server clusters
- CI/CD pipeline task execution
- Distributed system monitoring
- Batch log collection/analysis

## Installation
```sh
go get github.com/kaichao/gopkg/exec
```

## Quick Start
### Local Execution
```go
code, stdout, stderr, err := exec.ExecCommandReturnAll("ls -l", 10) // 10s timeout
fmt.Printf("Exit: %d\nOutput:\n%s\nError:\n%s", code, stdout, stderr)
```

### SSH Execution
```go
config := exec.SSHConfig{
    User:    "admin",
    Host:    "192.168.1.100",
    Port:    22,
    KeyPath: "/path/to/private_key",
}

code, out, errOut, err := exec.ExecSSHCommand(config, "docker ps", 30) // 30s timeout
```

## API Reference
### ExecCommandReturnAll
```go
func ExecCommandReturnAll(command string, timeout int) (int, string, string, error)
```

### ExecSSHCommand
```go
func ExecSSHCommand(config SSHConfig, command string, timeout int) (int, string, string, error)
```

### SSHConfig
```go
type SSHConfig struct {
    User     string // SSH username
    Host     string // Server IP/hostname
    Port     int    // SSH port (default: 22)
    KeyPath  string // Path to private key (default: ~/.ssh/id_rsa)
    Password string // Password authentication
}
```

## Advanced Usage
### Custom Signal Handling
```go
// Terminate process group on SIGINT
signal.Notify(sigChan, syscall.SIGINT)
go func() {
    <-sigChan
    syscall.Kill(-pid, syscall.SIGTERM)
}()
```
