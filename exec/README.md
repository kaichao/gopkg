# exec

English | [中文](README.zh.md)

## Table of Contents
1. [Features](#features)
2. [Use Cases](#use-cases)
3. [Installation](#installation)
4. [Quick Start](#quick-start)
5. [API Reference](#api-reference)
6. [Security Considerations](#security-considerations)
7. [Best Practices](#best-practices)
8. [Advanced Usage](#advanced-usage)
9. [FAQ](#faq)
10. [Testing](#testing)

## Features

Cross-environment command execution toolkit with:

- **Unified Interface**: Same API for local, remote SSH and container execution
- **Full Output Capture**: Synchronously captures stdout, stderr and exit code
- **Flexible Timeout**: Supports both command-level and connection-level timeouts
- **Multiple Auth Methods**: SSH supports key, password and agent forwarding
- **Process Management**: Background process and process group support

## Use Cases

### Batch Server Operations
```go
// Execute maintenance commands across servers
for _, host := range servers {
    config.Host = host
    exec.RunSSHCommand(config, "apt update && apt upgrade -y", 300)
}
```

### CI/CD Pipelines
```go
// Post-deployment verification
if code, out, _ := exec.RunReturnAll("curl -sSf http://localhost:8080/health", 10); code != 0 {
    log.Fatal("Service health check failed")
}
```

### Container Management
```go
// Run diagnostic commands in containers
exec.RunSSHCommand(config, "singularity exec app.sif df -h", 30)
```

## Installation

```sh
go get github.com/kaichao/gopkg/exec
```

## Quick Start

### Local Execution
```go
code, stdout, stderr, err := exec.RunReturnAll("ls -l /tmp", 10)
if err != nil {
    log.Printf("Execution failed: %v\nOutput: %s\nError: %s", err, stdout, stderr)
}
```

### SSH Remote Execution
```go
config := exec.SSHConfig{
    User:    "admin",
    Host:    "10.0.0.1", 
    KeyPath: "/home/user/.ssh/id_rsa",
}

// Execute and capture full output
code, out, errOut, err := exec.RunSSHCommand(config, "docker ps -a", 30)
```

### Container Execution
```go
// Run commands in Singularity container
cmd := "singularity exec /images/debian.sif apt-get update"
RunSSHCommand(config, cmd, 60)
```

## API Reference

### Core Methods
```go
// Local execution
func RunReturnAll(command string, timeout int) (code int, stdout string, stderr string, err error)

// SSH execution 
func RunSSHCommand(config SSHConfig, command string, timeout int) (code int, stdout string, stderr string, err error)
```

### SSHConfig Struct
```go
type SSHConfig struct {
    User       string // Required
    Host       string // Required
    Port       int    // Default 22
    KeyPath    string // Preferred over password
    Password   string 
    Timeout    int    // Connection timeout (seconds)
    Background bool   // Run command in background
}
```

## Security Considerations

1. **Authentication Security**
   - Set SSH private key permissions to 600
   - Avoid hardcoding passwords in code

2. **Command Injection Protection**
   ```go
   // Unsafe
   cmd := fmt.Sprintf("ls %s", userInput)
   
   // Safe approach
   cmd := fmt.Sprintf("ls %s", filepath.Clean(userInput))
   ```

3. **Logging**
   - Record metadata for critical operations (user, command, timestamp)
   - Avoid logging sensitive output

## Best Practices

### Connection Reuse
```go
var client *ssh.Client

func getClient(config SSHConfig) (*ssh.Client, error) {
    if client == nil {
        // Initialize connection...
    }
    return client, nil
}
```

### Error Handling
```go
// Check specific error types
if errors.Is(err, exec.ErrTimeout) {
    // Handle timeout
}
```

### Resource Cleanup
```go
defer func() {
    if cmd.Process != nil {
        cmd.Process.Kill()
    }
}()
```

## Advanced Usage

### Signal Handling
```go
// Terminate entire process group
syscall.Kill(-pid, syscall.SIGTERM)
```

### Background Processes
```go
// Run command in background mode
config := exec.SSHConfig{
    Host: "10.0.0.1",
    User: "admin",
    Background: true, // Enable background execution
    // ... other config
}

// Returns PID immediately while process continues running
_, pid, _, _ := exec.RunSSHCommand(config, "long-running-command", 0)
```

## FAQ

### Connection Timeouts
- Check network firewall settings
- Increase connection timeout:
  ```go
  config.Timeout = 30 // seconds
  ```

### Output Truncation
- Use buffers or temp files for large outputs
- Set reasonable execution timeouts

## Testing

1. Prepare test Singularity image:
```sh
mkdir -p ~/singularity
docker save debian:12-slim -o debian.tar
singularity build ~/singularity/debian.sif docker-archive://debian.tar
```

2. Run unit tests:
```sh
cd exec && go test -v
