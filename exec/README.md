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
	
	"github.com/kaichao/gopkg/exec"
)

func main() {
	// Local execution
	code, stdout, stderr, err := exec.RunReturnAll("ls -l /tmp", 10)
	if err != nil {
		log.Printf("Execution failed: %v\nOutput: %s\nError: %s", err, stdout, stderr)
	}
}
```

## Documentation

For complete documentation including all API functions, configuration options, and usage examples, see:

- [Package Documentation](https://pkg.go.dev/github.com/kaichao/gopkg/exec)
- [doc.go](./doc.go) - Detailed API reference with examples
- [examples/basic/main.go](./examples/basic/main.go) - Basic usage examples
- [examples/advanced/main.go](./examples/advanced/main.go) - Advanced scenarios including SSH

## Exit Code Convention

- `0`: Command executed successfully
- `124`: Command timed out
- `125`: Command execution failed (e.g., pipe creation, process start)
- Other non-zero: Command-specific exit code
- `128 + signal`: Command terminated by signal (e.g., SIGKILL = 128+9 = 137)

## SSH Configuration

```go
type SSHConfig struct {
	User       string // Required: SSH username
	Host       string // Required: SSH host
	Port       int    // Optional: SSH port (default: 22)
	KeyPath    string // Optional: Path to SSH private key (preferred over password)
	Password   string // Optional: SSH password
	Background bool   // Optional: Run command in background mode
	UseHomeTmp bool   // Optional: Use ${HOME}/tmp instead of /tmp for temporary files
}
```

## Security Considerations

1. **Authentication Security**
   - Set SSH private key permissions to 600
   - Avoid hardcoding passwords in source code

2. **Command Injection Protection**
   ```go
   // Unsafe
   cmd := fmt.Sprintf("ls %s", userInput)
   
   // Safer approach
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
