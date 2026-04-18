# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

`gopkg` is a Go utility library powering [scalebox](https://github.com/kaichao/scalebox). It provides several specialized sub-packages for common infrastructure tasks.

## Project Structure

```
gopkg/
├── asyncbatch/     # Asynchronous batch processor
├── dbcache/        # Database caching layer
├── errors/         # Enhanced error handling with tracing
├── exec/           # Cross-environment command executor
├── logger/         # Structured logging for traced errors
└── pgbulk/         # PostgreSQL bulk operations
```

## Key Packages

### errors Package
The `errors` package provides comprehensive error handling with:
- **TracedError** type with stack traces, error codes, timestamps, and context
- Error chain support with `Wrap()`, `WrapE()` functions
- Standard Go error compatibility (`errors.Is`, `errors.As`, `errors.Unwrap`)
- Convenient `E()` function for flexible error creation

Key files:
- `traced_error.go` - Core TracedError type and methods
- `simple_api.go` - Convenience functions (E, WrapE, Must, etc.)

### logger Package
The `logger` package provides a comprehensive logging solution with structured error logging, async output, log rotation, and sensitive data filtering.

#### **Logger Struct (Recommended for New Code)**
The primary interface is the `Logger` struct which provides a complete logging solution:

```go
cfg := &logger.Config{
    Level:        "info",              // Log level
    Format:       "json",              // Output format (text, json)
    Output:       "stdout",            // Output target (stdout, stderr, file)
    FilePath:     "app.log",           // Log file path
    MaxSize:      100,                 // Max file size in MB
    MaxAge:       7,                   // Max days to keep old logs
    MaxBackups:   5,                   // Max backup files
    AsyncEnabled: false,               // Enable async logging
    BufferSize:   1000,                // Async buffer size
}
log, err := logger.NewLogger(cfg)
```

**Key Features:**
- **Async Logging**: Non-blocking log writes with configurable buffer
- **Log Rotation**: Automatic rotation based on size and time
- **Environment Configuration**: Easy setup via environment variables
- **Multiple Output Formats**: Text and JSON support
- **Thread Safety**: Safe for concurrent use

**Methods:**
```go
// Standard logging
log.Trace("message")
log.Debug("message")
log.Info("message")
log.Warn("message")
log.Error("message")

// Structured logging
log.WithField("key", "value").Info("message")
log.WithFields(logrus.Fields{...}).Info("message")
log.WithError(err).Error("error occurred")

// Dynamic configuration
log.SetLevel("debug")
config := log.GetConfig()
```

#### **Error Logging Functions (For TracedError Integration)**

##### **LogError()** ⭐
- **Automatic decision making** based on log level
- DEBUG/TRACE → detailed logging (LogTracedError)
- INFO/WARN/ERROR → simple logging (SimpleLog)
- **Environment variable override**: `LOG_ERROR_VERBOSE`
  - `LOG_ERROR_VERBOSE=true` forces detailed logging
  - `LOG_ERROR_VERBOSE=false` forces simple logging
  - Unset = auto mode (default)
- **Best practice** for most use cases

##### **LogTracedError()**
- Detailed error chain logging with full context
- Supports both TracedError and standard Go errors in chains
- Includes location, timestamp, and error codes
- Inner errors logged at Debug level
- Use for development debugging

##### **SimpleLog()**
- Production-safe logging with sensitive data filtering
- Filters passwords, tokens, API keys, and secrets
- Suitable for production environments

##### **Global Functions**
- `InitGlobal(cfg *Config)` - Initialize global logger
- `Global() *Logger` - Get global logger instance
- `LogTracedErrorDefault(err error, level ...logrus.Level)` - Log using global logger
- `SimpleLogDefault(err error, level ...logrus.Level)` - Simple log using global logger

#### **Configuration**
```go
// Via Config struct
cfg := &logger.Config{
    Level:  "info",
    Format: "json",
    Output: "stdout",
}
log, _ := logger.NewLogger(cfg)

// Via environment variables
// LOG_LEVEL=debug LOG_FORMAT=json LOG_OUTPUT=file go run main.go
cfg := logger.LoadConfig()
log, _ := logger.NewLogger(cfg)

// From JSON file
log, _ := logger.NewLoggerFromConfigFile("config/logger.json")
```

#### **Async Logging**
```go
cfg := &logger.Config{
    Level:        "info",
    Format:       "json",
    Output:       "stdout",
    AsyncEnabled: true,
    BufferSize:   2000,
}
log, _ := logger.NewLogger(cfg)
defer log.Close() // Ensure all logs are flushed

log.Info("This is logged asynchronously")
```

#### **Log Rotation**
```go
cfg := &logger.Config{
    Level:      "info",
    Format:     "json",
    Output:     "file",
    FilePath:   "/var/log/myapp/app.log",
    MaxSize:    100,  // 100MB
    MaxAge:     30,   // 30 days
    MaxBackups: 10,   // Keep 10 backups
}
log, _ := logger.NewLogger(cfg)
defer log.Close()
```

#### **Test Helpers**
```go
entry, buf := logger.NewTestEntry()      // Text formatter
entry, buf := logger.NewJSONTestEntry()  // JSON formatter
```

#### **Sensitive Data Filtering**
The `IsSensitiveKey()` function detects and filters:
- Passwords: `password`, `user_password`, `password_hash`
- Tokens: `token`, `api_token`, `access_token`
- Secrets: `secret`, `secret_key`, `api_secret`
- Credit info: `credit`, `credit_card`
- Keys: `key`, `api_key`, `private_key` (specific patterns only)

## Development Commands

### Build
```bash
go build ./...
```

### Test
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./errors/...
go test ./logger/...
go test ./pgbulk/...
go test ./asyncbatch/...
go test ./dbcache/...
go test ./exec/...
```

### Lint
```bash
# Run golangci-lint if available
golangci-lint run

# Or use go vet
go vet ./...
```

## Dependencies

Key external dependencies:
- `github.com/jackc/pgx/v5` - PostgreSQL driver
- `github.com/lib/pq` - PostgreSQL driver (legacy)
- `github.com/patrickmn/go-cache` - In-memory cache
- `github.com/sirupsen/logrus` - Structured logging
- `github.com/stretchr/testify` - Testing utilities
- `golang.org/x/crypto` - SSH support for exec package

## Architecture Notes

### Error Handling Pattern
The codebase uses a consistent error handling pattern:
1. Create errors using `errors.New()` or `errors.E()` with optional codes and context
2. Wrap errors with additional context using `errors.Wrap()` or `errors.WrapE()`
3. Log errors using `logger.LogTracedError()` (development) or `logger.SimpleLog()` (production)
4. Use standard Go error utilities (`errors.Is`, `errors.As`) for error checking

### Error Chain Support
The `TracedError` type supports mixed error chains:
- Can wrap standard Go errors (`error` interface)
- Can wrap other `TracedError` instances
- `LogTracedError()` handles both types in the chain correctly

### Sensitive Data Filtering
The logger package automatically filters sensitive context fields:
- Passwords, tokens, secrets, credit card info
- API keys and private keys
- Custom patterns can be added to `IsSensitiveKey()`

## Recent Changes

The repository has recent commits focused on:
- Enhanced error chain printing for mixed error types (fix)
- Simplified API with default logger support (feat)
- Package-level Is, As, and Unwrap functions (feat)
- Error handling enhancements and Location tracking fixes

## Testing

Each package has comprehensive test coverage:
- Example tests demonstrate usage patterns
- Test helpers in logger package for consistent test output
- Integration tests for PostgreSQL operations (pgbulk)
- SSH command execution tests (exec)

## Usage Examples

### Error Creation and Logging
```go
import (
    "github.com/kaichao/gopkg/errors"
    "github.com/kaichao/gopkg/logger"
)

// Create error with code and context
err := errors.E(404, "user not found", "user_id", 123)

// Log with full details (development)
logger.LogTracedError(err, logEntry)

// Log safely (production)
logger.SimpleLog(err, logEntry)
```

### Error Wrapping
```go
if err := db.QueryRow(ctx, query).Scan(&result); err != nil {
    return errors.WrapE(err, "database query failed", "query", query)
}