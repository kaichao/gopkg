# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

`gopkg` is a Go utility library powering [scalebox](https://github.com/kaichao/scalebox). It provides several specialized sub-packages for common infrastructure tasks.

## Project Structure

```
gopkg/
├── asyncbatch/     # Asynchronous batch processor with dynamic flow control
├── dbcache/        # Database caching layer with SQL template support
├── errors/         # Enhanced error handling with tracing and context
├── exec/           # Cross-environment command executor (local/SSH)
├── logger/         # Structured logging with error tracing and rotation
├── param/          # Unified command line parameter management for Cobra
└── pgbulk/         # PostgreSQL bulk operations (COPY, INSERT, UPDATE)
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

### param Package
The `param` package provides unified command line parameter management for Cobra with:

#### **Design Goals**
1. Unified parameter retrieval interface supporting multiple data types
2. Priority: command line arguments > environment variables > static default values > dynamic default functions
3. Dynamic default value functions (from databases, config files, etc.)
4. Simplify command implementation, reduce boilerplate code
5. Parameter validation and required parameter checking

#### **Core Features**
- **Type Safety**: Supports `int`, `string`, `bool`, `time.Duration`, `int64`, `float64`, `[]string`
- **Automatic Environment Variable Name Derivation**: Parameter names automatically converted to uppercase with underscores
- **Dynamic Default Values**: Runtime-computed defaults via `WithDefaultFunc`
- **Parameter Validation**: Custom validation logic via `WithValidator`
- **Required Parameters**: Mark parameters as required via `WithRequired`

### asyncbatch Package
Generic batch processor for asynchronous task processing with dynamic flow control and parallel execution.

**Key Features:**
- **Generic Support**: Type-safe processing for any task type
- **Flexible Configuration**: Configure parameters via `With...` functions
- **Dynamic Batching**: Adjusts batch triggering based on task count and timing
- **Parallel Processing**: Multiple workers for concurrent batch processing
- **Graceful Shutdown**: Safely processes remaining tasks before exiting

**Configuration Parameters:**
- `maxSize` (default 1000): Maximum tasks per batch
- `lowerRatio` (default 0.1): Minimum ratio for underfilled batches
- `fixedWait` (default 5ms): Wait time for initial task checks
- `underfilledWait` (default 20ms): Wait time for underfilled batches
- `numWorkers` (default 1): Number of parallel workers (1-8)

### pgbulk Package
Lightweight PostgreSQL bulk operations package for high-performance data operations.

**Features:**
- **Batch Processing**: Automatically chunks large datasets into optimal batches
- **SQL Templates**: Reusable templates with dynamic placeholders
- **Full CRUD Support**: `INSERT`, `UPDATE`, and `INSERT...RETURNING` operations
- **PG-Compatible**: Respects PostgreSQL's parameter limits
- **Error Handling**: Enhanced error tracing with `github.com/kaichao/gopkg/errors`

**Key Functions:**
- `Copy()`: Bulk insert using PostgreSQL's COPY command
- `Insert()`: Insert data with optional ON CONFLICT clause
- `InsertReturningID()`: Insert data and return IDs of inserted rows
- `Update()`: Bulk update with error tracking

### dbcache Package
Generic database caching layer with SQL template support and automatic cache population.

**Features:**
- **SQL Templating**: Parameterized query support with $1, $2 placeholders
- **Automatic Caching**: Transparent cache population on cache misses
- **Type Safety**: Generics support for any data type
- **Cache Control**: Configurable expiration and cleanup intervals
- **Custom Loaders**: Optional custom loader functions for complex data loading

### exec Package
Cross-environment command execution utilities for local and remote SSH environments.

**Features:**
- **Unified Interface**: Same API for local and remote SSH execution
- **Full Output Capture**: Synchronously captures stdout, stderr and exit code
- **Flexible Timeout**: Supports both command-level and connection-level timeouts
- **Multiple Auth Methods**: SSH supports key, password and agent forwarding
- **Process Management**: Background process and process group support
- **Circular Buffering**: 10MB output limit with circular buffer for large outputs

**Exit Code Convention:**
- `0`: Command executed successfully
- `124`: Command timed out
- `125`: Command execution failed
- Other non-zero: Command-specific exit code
- `128 + signal`: Command terminated by signal

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
go test ./param/...
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
- `github.com/spf13/cobra` - Command line interface
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

### Package Integration Patterns
- `errors` + `logger`: Use `LogTracedError()` for detailed error chain logging
- `pgbulk` + `errors`: All bulk operations return enhanced traced errors
- `param` + `cobra`: Simplifies command parameter handling with validation
- `dbcache` + `go-cache`: Provides automatic database result caching

## Recent Changes

The repository has recent commits focused on:
- Enhanced error chain printing for mixed error types (fix)
- Simplified API with default logger support (feat)
- Package-level Is, As, and Unwrap functions (feat)
- Error handling enhancements and Location tracking fixes
- Added param package for unified command line parameter management with Cobra (feat)
- Comprehensive documentation improvements across all packages

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
```

### Async Batch Processing
```go
bp, _ := asyncbatch.NewBatchProcessor[int](
    func(batch []int) {
        fmt.Printf("Processing batch: %v\n", batch)
    },
    asyncbatch.WithMaxSize(100),
    asyncbatch.WithNumWorkers(2),
)
defer bp.Shutdown()

for i := 0; i < 500; i++ {
    bp.Add(i)
}
```

### PostgreSQL Bulk Operations
```go
import (
    "context"
    "github.com/jackc/pgx/v5"
    "github.com/kaichao/gopkg/pgbulk"
)

conn, _ := pgx.Connect(context.Background(), "postgres://user:password@localhost/dbname")

// Bulk insert with ID returning
ids, _ := pgbulk.InsertReturningID(conn, "INSERT INTO products (name, price)", [][]interface{}{
    {"Product A", 99.99},
    {"Product B", 149.99},
})
```

### Database Caching
```go
import (
    "database/sql"
    "time"
    "github.com/kaichao/gopkg/dbcache"
)

db, _ := sql.Open("postgres", "postgres://user:password@localhost/dbname")

emailCache := dbcache.New[string](
    db,
    "SELECT email FROM users WHERE id = $1",
    5*time.Minute,  // Cache expiration
    10*time.Minute, // Cleanup interval
    nil,            // Use default SQL loader
)

email, _ := emailCache.Get(123)
```

### Command Execution
```go
import "github.com/kaichao/gopkg/exec"

// Local execution
code, stdout, stderr, err := exec.RunReturnAll("ls -l /tmp", 10)

// SSH execution
sshConfig := exec.SSHConfig{
    Host: "10.0.0.1",
    User: "admin",
    KeyPath: "/path/to/key",
}
code, stdout, stderr, err := exec.RunSSHCommand(sshConfig, "ps aux", 30)
```

### Parameter Management with Cobra
```go
import (
    "github.com/spf13/cobra"
    "github.com/kaichao/gopkg/param"
)

var rootCmd = &cobra.Command{
    RunE: func(cmd *cobra.Command, args []string) error {
        // Get integer parameter with automatic env var derivation
        appID, err := param.GetInt(cmd, "app-id")
        if err != nil {
            return err
        }

        // Get required string parameter
        cluster, err := param.GetString(cmd, "cluster",
            param.WithRequired(),
            param.WithDefault("default-cluster"),
        )
        if err != nil {
            return err
        }

        // Use parameters...
        return nil
    },
}
```