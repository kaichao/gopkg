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
    "github.com/kaichao/gopkg/errors"
    "github.com/kaichao/gopkg/logger"
)

// Create logger
cfg := &logger.Config{
    Level:  "info",
    Format: "json",
    Output: "stdout",
}
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

```go
logger.LogError(err, entry)
```

### LogTracedError
Detailed logging with full error chains.

```go
logger.LogTracedError(err, entry)
```

### SimpleLog
Production-safe logging with sensitive data filtering.

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
| LOG_ASYNC_ENABLED | false | Enable async logging |

## Examples

See `examples/` directory for complete examples.

## Documentation

Run `go doc github.com/kaichao/gopkg/logger` for API documentation.