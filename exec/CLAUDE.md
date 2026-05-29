# CLAUDE.md

## exec Package

Cross-environment command execution (local/SSH) with unified interface.

### Key Functions

```go
// Local execution — exit code embedded in error
func RunReturnAll(command string, timeout int) (stdout string, stderr string, err error)

// SSH execution — exit code embedded in error
func RunSSHCommand(config SSHConfig, command string, timeout int) (stdout string, stderr string, err error)

// Retry wrapper
func RunWithRetries(cmd string, numRetries int, timeout int) (int, error)
```

**Important:** Exit code is no longer a separate return value. Use `errors.GetCode(err)` to retrieve it.

### SSHConfig
```go
type SSHConfig struct {
    User       string // Required
    Host       string // Required
    Port       int    // Default: 22
    KeyPath    string // Prefers id_ed25519 > id_rsa > id_ecdsa
    Password   string
    Background bool   // Run in background, returns PID
    UseHomeTmp bool   // Use ${HOME}/tmp instead of /tmp
}
```

### Exit Code Convention
- `0` — Success
- `124` — Timeout
- `125` — Execution failure (pipe, process start, etc.)
- Other — Command-specific
- `128 + signal` — Signal termination (e.g., SIGKILL = 137)

### Output Handling
- 10MB circular buffer for stdout/stderr
- SSH DEBUG lines are filtered from output
- SSH background mode returns PID as stdout

### Usage Examples
```go
// Local
stdout, stderr, err := exec.RunReturnAll("ls -l /tmp", 10)
if err != nil {
    log.Printf("failed: %v, code=%d", err, errors.GetCode(err))
}

// SSH
config := exec.SSHConfig{Host: "10.0.0.1", User: "admin", KeyPath: "/path/to/key"}
stdout, stderr, err = exec.RunSSHCommand(config, "docker ps -a", 30)
```
