# Logger Package

Structured logging for traced errors with sensitive data filtering.

## Overview

The `logger` package provides specialized logging functions for working with `TracedError` objects from the `errors` package. It offers:

- Detailed error chain logging
- Automatic sensitive data filtering
- Production-safe logging
- Test helper functions

## Core Functions

### LogError (Recommended)
Automatically chooses between detailed and simple logging based on log level.

```go
func LogError(err error, log *logrus.Entry, level ...logrus.Level)
```

**Features:**
- **Automatic decision making**: DEBUG/TRACE → detailed, INFO/WARN/ERROR → simple
- **Environment variable override**: 
  - `LOG_ERROR_VERBOSE=true` forces detailed logging
  - `LOG_ERROR_VERBOSE=false` forces simple logging
  - Unset = auto mode (default)
- **Same interface** as other logging functions
- **Best practice** for most use cases

### LogTracedError
Logs traced errors with full context and error chains.

```go
func LogTracedError(err error, log *logrus.Entry, level ...logrus.Level)
```

**Features:**
- Logs complete error chain (outermost error + all causes)
- **Includes both TracedError and standard Go errors in the chain**
- Includes location, timestamp, and context for each TracedError in chain
- Standard errors are logged with basic information (error message only)
- Inner errors are logged at Debug level
- Supports custom log levels
- **Use for development debugging or when full details are needed**

**Usage:**
```go
import (
    "fmt"
    
    "github.com/kaichao/gopkg/errors"
    "github.com/kaichao/gopkg/logger"
    "github.com/sirupsen/logrus"
)

// Create pure TracedError chain
root := errors.New("database error").WithContext("query", "SELECT * FROM users")
wrapped := errors.Wrap(root, "query failed")

// Setup logger
log := logrus.New()
entry := logrus.NewEntry(log)

// Log with full details
logger.LogTracedError(wrapped, entry)

// Log with custom level
logger.LogTracedError(wrapped, entry, logrus.WarnLevel)

// Create mixed error chain (TracedError + standard errors)
stdErr := fmt.Errorf("standard library: file not found")
mixedWrapped := errors.Wrap(stdErr, "operation failed").WithContext("filename", "data.txt")

// LogTracedError will show both errors in the chain
logger.LogTracedError(mixedWrapped, entry)
```

### SimpleLog
Production-safe logging with sensitive data filtering.

```go
func SimpleLog(err error, log *logrus.Entry, level ...logrus.Level)
```

**Features:**
- Filters sensitive information (passwords, tokens, keys, etc.)
- Logs only non-sensitive context
- Suitable for production environments
- Supports custom log levels

**Usage:**
```go
// Create error with sensitive data
err := errors.New("auth failed").
    WithContext("username", "john_doe").
    WithContext("password", "secret123").  // Will be filtered
    WithContext("attempt", 3)

// Safe logging (password won't appear in logs)
logger.SimpleLog(err, entry)
```

### Default Logger Functions (Simplified API)
For simpler usage, the package provides default logger functions that don't require passing a `logrus.Entry`.

```go
// Set the default logger (optional, called automatically with sensible defaults)
func SetDefaultLogger(logger *logrus.Logger)

// Log using default logger
func LogTracedErrorDefault(err error, level ...logrus.Level)
func SimpleLogDefault(err error, level ...logrus.Level)
```

**Features:**
- No need to create or pass `logrus.Entry`
- Uses package-level default logger
- Same functionality as regular functions
- Backward compatible

**Usage:**
```go
import (
    "github.com/kaichao/gopkg/errors"
    "github.com/kaichao/gopkg/logger"
)

// Optionally configure default logger (once at app startup)
logger.SetDefaultLogger(myCustomLogger)

// Create error
err := errors.New("file not found").
    WithContext("filename", "data.txt")

// Simple logging with default logger
logger.LogTracedErrorDefault(err)

// Or with custom level
logger.SimpleLogDefault(err, logrus.WarnLevel)
```

**Default Configuration:**
If `SetDefaultLogger` is not called, a default logger is automatically created with:
- Output: `os.Stderr`
- Formatter: `logrus.TextFormatter` with full timestamps
- Level: `logrus.InfoLevel`

## Sensitive Data Filtering

### IsSensitiveKey
Checks if a key contains sensitive information.

```go
func IsSensitiveKey(key string) bool
```

**Filtered patterns:**
- `password`, `user_password`, `password_hash`
- `token`, `api_token`, `access_token`
- `secret`, `secret_key`, `api_secret`
- `credit`, `credit_card`, `credit_number`
- `key`, `api_key`, `private_key` (only specific patterns)

**Examples:**
- `password` → filtered
- `api_key` → filtered
- `key1` → NOT filtered
- `customer_key` → NOT filtered

## Test Helper Functions

### NewTestEntry
Creates a logrus.Entry for testing with text formatter.

```go
func NewTestEntry() (*logrus.Entry, *bytes.Buffer)
```

### NewJSONTestEntry
Creates a logrus.Entry for testing with JSON formatter.

```go
func NewJSONTestEntry() (*logrus.Entry, *bytes.Buffer)
```

**Usage in tests:**
```go
func TestLogging(t *testing.T) {
    // Get entry and buffer
    entry, buf := logger.NewTestEntry()
    
    // Create and log error
    err := errors.New("test error")
    logger.LogTracedError(err, entry)
    
    // Check output
    output := buf.String()
    if !strings.Contains(output, "test error") {
        t.Error("Error should be logged")
    }
}
```

## Development vs Production

### Development Environment
Use `LogTracedError` for detailed debugging:
- Full error chains
- All context information
- Complete stack traces

### Production Environment
Use `SimpleLog` for security and performance:
- Sensitive data filtered
- Concise output
- Safe for external logging systems

**Example:**
```go
func handleError(err error, log *logrus.Entry, isProduction bool) {
    if isProduction {
        logger.SimpleLog(err, log, logrus.ErrorLevel)
    } else {
        logger.LogTracedError(err, log, logrus.ErrorLevel)
    }
}
```

## Integration with Errors Package

The logger package is designed to work seamlessly with the `errors` package:

```go
import (
    "github.com/kaichao/goscalebox/internal/errors"
    "github.com/kaichao/goscalebox/internal/logger"
)

// Create traced error
err := errors.E(404, "not found", "resource", "/api/users")

// Log it
logger.LogTracedError(err, logEntry)
```

## Best Practices

1. **Use `SimpleLog` in production** to avoid logging sensitive data
2. **Use `LogTracedError` in development** for complete debugging
3. **Add meaningful context** to errors for better logs
4. **Use test helpers** for consistent test output
5. **Set appropriate log levels** based on error severity

## Examples

See `example_test.go` for comprehensive usage examples including development vs production scenarios.