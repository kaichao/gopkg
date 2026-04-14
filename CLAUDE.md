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
The `logger` package provides structured logging specifically designed for `TracedError`:

#### **Recommended: LogError()** ⭐
- **Automatic decision making** based on log level
- DEBUG/TRACE → detailed logging (LogTracedError)
- INFO/WARN/ERROR → simple logging (SimpleLog)
- **Environment variable override**: `LOG_ERROR_VERBOSE`
  - `LOG_ERROR_VERBOSE=true` forces detailed logging
  - `LOG_ERROR_VERBOSE=false` forces simple logging
  - Unset = auto mode (default)
- **Best practice** for most use cases

#### Other Functions:
- `LogTracedError()` - Detailed error chain logging with full context
- `SimpleLog()` - Production-safe logging with sensitive data filtering
- `IsSensitiveKey()` - Detects sensitive field names (password, token, secret, credit, key)
- Default logger support with `SetDefaultLogger()`
- Test helpers: `NewTestEntry()`, `NewJSONTestEntry()`

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