# logger

Structured logging for Go with error tracing, async output, and sensitive data filtering.

## Features

- Structured logging (JSON/text formats)
- Error tracing with gopkg/errors integration
- Async logging with buffering
- Log rotation (size/time based)
- Sensitive data filtering
- Environment-based configuration

## Quick Start

```go
import (
    "github.com/kaichao/gopkg/logger"
    "github.com/sirupsen/logrus"
)

// Create logger
cfg := &logger.Config{
    Level:  "info",
    Format: "json",
    Output: "stdout",
}
log, err := logger.NewLogger(cfg)
if err != nil {
    panic(err)
}
defer log.Close()

// Or use the simpler constructor (panics on error)
log := logger.NewOrMust(cfg)
defer log.Close()

// Basic logging
log.Info("Application started")

// Structured logging
log.WithField("user_id", 123).Info("User logged in")

// Error logging
err := errors.New("database error")
log.WithError(err).Error("Operation failed")
```

## Error Logging

### LogError (recommended)
Auto-chooses between detailed and simple logging based on log level.
- DEBUG/TRACE → detailed (LogTracedError)
- INFO/WARN/ERROR → simple (SimpleLog)
- Override with `LOG_ERROR_VERBOSE=true` or `LOG_ERROR_VERBOSE=false`

```go
logger.LogError(err, entry)
```

### LogTracedError
Detailed logging with full error chains, includes location, timestamp, and codes. Inner errors logged at Debug level.

```go
logger.LogTracedError(err, entry)
```

### SimpleLog
Production-safe logging with sensitive data filtering (passwords, tokens, API keys, secrets).

```go
logger.SimpleLog(err, entry)
```

## Configuration

Configuration via environment variables or struct:

| Variable | Default | Description |
|----------|---------|-------------|
| LOG_LEVEL | info | Log level (trace, debug, info, warn, error, fatal) |
| LOG_FORMAT | json | Output format (text, json) |
| LOG_OUTPUT | stdout | Output destination (stdout, stderr, file) |
| LOG_FILE_PATH | app.log | Log file path |
| LOG_MAX_SIZE | 100 | Max file size in MB |
| LOG_MAX_AGE | 7 | Max days to keep old logs |
| LOG_MAX_BACKUPS | 5 | Max backup files to retain |
| LOG_ASYNC_ENABLED | false | Enable async logging |
| LOG_ASYNC_BUFFER_SIZE | 1000 | Async buffer capacity |
| LOG_ERROR_VERBOSE | (auto) | true=detailed, false=simple |

## Examples

See `examples/` directory for complete examples.

## Documentation

Run `go doc github.com/kaichao/gopkg/logger` for API documentation.
