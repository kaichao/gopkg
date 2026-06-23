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
├── misc/           # Miscellaneous small utilities (stdin, etc.)
├── param/          # Unified command line parameter management for Cobra
├── pgbulk/         # PostgreSQL bulk operations (COPY, INSERT, UPDATE)
├── security/       # Pluggable security framework (AuthN/AuthZ/Billing) with gRPC interceptors
└── self/           # Runtime introspection utilities (goroutine/thread/process ID)
```

Each sub-package has its own `CLAUDE.md` with package-specific details.

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
go test ./self/...
go test ./misc/...
go test ./security/...
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
- `google.golang.org/grpc` - gRPC framework for security interceptors

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
- `security` + `grpc`: Pluggable AuthN/AuthZ/Billing via gRPC interceptors

## Testing

Each package has comprehensive test coverage:
- Example tests demonstrate usage patterns
- Test helpers in logger package for consistent test output
- Integration tests for PostgreSQL operations (pgbulk)
- SSH command execution tests (exec)
