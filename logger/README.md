# Logger Package

A comprehensive logging library for Go with structured error logging, async output, log rotation, and sensitive data filtering.

## Overview

The `logger` package provides a complete logging solution with:

- **Structured Error Logging**: Specialized functions for `TracedError` with full error chain support
- **Sensitive Data Filtering**: Automatic filtering of passwords, tokens, keys, and secrets
- **Async Logging**: Non-blocking log output for high-performance applications
- **Log Rotation**: Automatic file rotation based on size and time
- **Multiple Output Formats**: Text and JSON formats
- **Environment Configuration**: Easy configuration via environment variables
- **Test Helpers**: Built-in utilities for testing log output

## Quick Start

```go
import (
    "github.com/kaichao/gopkg/logger"
    "github.com/sirupsen/logrus"
)

// Create a logger
cfg := &logger.Config{
    Level:  "info",
    Format: "json",
    Output: "stdout",
}
log, _ := logger.NewLogger(cfg)

// Log a message
log.Info("Application started")

// Log with fields
log.WithField("user_id", 123).Info("User logged in")

// Log an error
err := errors.New("database connection failed")
log.WithError(err).Error("Database error")
```

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

## Logger Struct

The `Logger` struct provides a complete logging interface with support for structured logging, async output, and log rotation.

### Creating a Logger

```go
cfg := &logger.Config{
    Level:  "info",
    Format: "json",
    Output: "stdout",
}

// Create logger
log, err := logger.NewLogger(cfg)
if err != nil {
    panic("failed to create logger: " + err.Error())
}

// Or create with panic on error
log := logger.NewOrMust(cfg)
```

### From Configuration File

```go
// Load from JSON config file
log, err := logger.NewLoggerFromConfigFile("config/logger.json")
```

### Configuration Options

| Option | Description | Default | Environment Variable |
|--------|-------------|---------|---------------------|
| `Level` | Log level (trace, debug, info, warn, error, fatal) | info | `LOG_LEVEL` |
| `Format` | Output format (text, json) | json | `LOG_FORMAT` |
| `Output` | Output target (stdout, stderr, file) | stdout | `LOG_OUTPUT` |
| `FilePath` | Log file path (when Output=file) | app.log | `LOG_FILE_PATH` |
| `MaxSize` | Max file size in MB before rotation | 100 | `LOG_MAX_SIZE` |
| `MaxAge` | Max days to keep old log files | 7 | `LOG_MAX_AGE` |
| `MaxBackups` | Max number of old log files to keep | 5 | `LOG_MAX_BACKUPS` |
| `AsyncEnabled` | Enable asynchronous logging | false | `LOG_ASYNC_ENABLED` |
| `BufferSize` | Async buffer size | 1000 | `LOG_BUFFER_SIZE` |
| `DisableCaller` | Disable caller reporting | false | `LOG_DISABLE_CALLER` |

### Logger Methods

The `Logger` struct provides comprehensive logging methods:

#### Basic Logging
```go
// Standard log levels
log.Trace("Trace message")
log.Debug("Debug message")
log.Info("Info message")
log.Warn("Warning message")
log.Error("Error message")
log.Fatal("Fatal message")

// Formatted logging
log.Tracef("User %d logged in", userID)
log.Debugf("Processing %s", filename)
log.Infof("Request completed in %v", duration)
log.Warnf("High memory usage: %d MB", memUsage)
log.Errorf("Failed to connect: %v", err)
log.Fatalf("Cannot start: %v", err)
```

#### Structured Logging
```go
// With single field
log.WithField("user_id", 123).Info("User logged in")

// With multiple fields
log.WithFields(logrus.Fields{
    "user_id": 123,
    "action": "login",
    "ip": "192.168.1.1",
}).Info("User activity")

// With error
log.WithError(err).Error("Operation failed")

// Chained context
log.WithField("request_id", "abc-123").
    WithField("user_id", 456).
    Info("Processing request")
```

#### Dynamic Configuration
```go
// Change log level
if err := log.SetLevel("debug"); err != nil {
    log.Error("Failed to set log level")
}

// Get current configuration
config := log.GetConfig()
fmt.Printf("Current level: %s\n", log.GetLevel())

// Check if level is enabled
if log.IsLevelEnabled(logrus.DebugLevel) {
    log.Debug("Debug logging is enabled")
}
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

### Global Logger Functions
For simpler usage, the package provides global logger functions that don't require creating a logger instance.

```go
// Initialize global logger (optional)
func InitGlobal(cfg *Config) error

// Get global logger
func Global() *Logger

// Global logging functions
func LogTracedErrorDefault(err error, level ...logrus.Level)
func SimpleLogDefault(err error, level ...logrus.Level)
```

**Features:**
- No need to manage logger instances
- Thread-safe global logger
- Easy to use in simple applications or libraries

**Usage:**
```go
import (
    "github.com/kaichao/gopkg/errors"
    "github.com/kaichao/gopkg/logger"
)

// Initialize global logger (once at app startup)
cfg := &logger.Config{
    Level:  "info",
    Format: "json",
}
logger.InitGlobal(cfg)

// Get global logger
log := logger.Global()

// Use global logger
log.Info("Application started")

// Or use default error logging functions
err := errors.New("file not found")
logger.LogTracedErrorDefault(err)
```

**Default Configuration:**
If `InitGlobal` is not called, a default logger is automatically created with:
- Level: `info`
- Format: `json`
- Output: `stdout`

### AsyncWriter
Asynchronous log writer for high-performance applications.

```go
type AsyncWriter struct {
    // Async configuration
    BufferSize int  // Channel buffer size (default: 1000)
    BatchSize  int  // Batch write size (default: 100)
}
```

**Features:**
- Non-blocking log writes
- Buffered channel with configurable size
- Batch processing for efficiency
- Graceful degradation when buffer is full
- Automatic flushing on shutdown

**Usage:**
```go
// Enable async logging in config
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

### RotatedWriter
Automatic log file rotation based on size and time.

```go
type RotatedWriter struct {
    // Rotation configuration
    MaxSize    int // Maximum file size in MB (default: 100)
    MaxAge     int // Maximum days to keep (default: 7)
    MaxBackups int // Maximum backup files (default: 5)
}
```

**Features:**
- Size-based rotation
- Time-based rotation (daily)
- Automatic cleanup of old files
- Configurable retention policy
- Thread-safe operations

**Usage:**
```go
// Configure file output with rotation
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

log.Info("This is logged to rotating file")
```

### Logger Management
```go
// Create a copy of logger (for independent context)
logCopy := log.Copy()

// Add fields to copy without affecting original
logCopy.WithField("request_id", "abc-123")

// Close logger and release resources
if err := log.Close(); err != nil {
    log.Error("Failed to close logger")
}

// Sync pending logs (for async writers)
if err := log.Sync(); err != nil {
    log.Error("Failed to sync logs")
}
```

## Sensitive Data Filtering

## Sensitive Data Filtering

The logger includes built-in protection against accidentally logging sensitive information.

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

**Usage:**
```go
// Sensitive data is automatically filtered in SimpleLog
err := errors.New("auth failed").
    WithContext("username", "john_doe").
    WithContext("password", "secret123").  // This field will be filtered
    WithContext("api_key", "sk-123456")   // This field will be filtered

logger.SimpleLog(err, entry)  // Password and API key won't appear in logs
```

## Test Helper Functions

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
```

## Advanced Features

### Development vs Production

#### Development Environment
Use `LogTracedError` for detailed debugging:
- Full error chains
- All context information
- Complete stack traces

#### Production Environment
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

## Code Analysis and Potential Issues

### ✅ Well-Implemented Features
1. **Error Chain Support**: Full support for both TracedError and standard Go errors
2. **Sensitive Data Filtering**: Comprehensive pattern matching for sensitive keys
3. **Async Logging**: Non-blocking with graceful degradation
4. **Log Rotation**: Size and time-based rotation with cleanup
5. **Environment Configuration**: Easy configuration via environment variables
6. **Test Coverage**: Comprehensive tests for all major features
7. **Thread Safety**: Proper use of mutexes for concurrent access

### ⚠️ Potential Issues and Considerations

#### 1. AsyncWriter Channel Overflow
**Issue**: When the async buffer is full, logs are written synchronously, which could cause blocking in high-throughput scenarios.

**Mitigation**: 
- Default buffer size is reasonable (1000 entries)
- Batch processing reduces overhead
- Consider monitoring `Buffered()` vs `Cap()` metrics

#### 2. Error Chain Memory Usage
**Issue**: `collectErrorChain` builds a slice of all errors in the chain, which could be memory-intensive for very deep error chains.

**Mitigation**: 
- Error chains are typically shallow (2-5 errors)
- Memory usage is temporary and freed immediately after logging
- Consider iterative processing for extremely deep chains

#### 3. File Permissions
**Issue**: Log files are created with `0644` permissions, which might be too permissive for sensitive applications.

**Recommendation**: 
```go
// Consider making this configurable
cfg := &logger.Config{
    FilePath:     "/var/log/app.log",
    // Add FileMode option in future
}
```

#### 4. Rotation Race Condition
**Issue**: There's a small window between checking file size and writing where rotation might not occur immediately.

**Impact**: Log files might slightly exceed the configured max size.

**Mitigation**: 
- Check includes `additionalSize` parameter
- Daily rotation provides additional safety net
- Acceptable for most use cases

#### 5. Global Logger Initialization
**Issue**: The `Global()` function uses `sync.Once` which means the first call determines the configuration.

**Best Practice**: 
```go
// Call InitGlobal early in main()
func main() {
    cfg := &logger.Config{Level: "debug"}
    logger.InitGlobal(cfg)
    
    // Now Global() will use this configuration
    log := logger.Global()
}
```

#### 6. Sensitive Key False Positives
**Issue**: The `IsSensitiveKey` function might be too aggressive or not aggressive enough depending on use case.

**Examples**:
- `key_name` is NOT filtered (false negative if it contains a secret)
- `customer_key` is NOT filtered (could be sensitive)
- `key1` is NOT filtered (correct, not a secret key)

**Recommendation**: 
```go
// Consider extending patterns for your use case
func MySensitiveKey(key string) bool {
    if logger.IsSensitiveKey(key) {
        return true
    }
    // Add custom patterns
    customPatterns := []string{"customer_key", "internal_key"}
    for _, pattern := range customPatterns {
        if strings.Contains(strings.ToLower(key), pattern) {
            return true
        }
    }
    return false
}
```

#### 7. Missing Features
1. **No log redaction for values**: Only keys are filtered, not values
2. **No structured field filtering**: Cannot filter specific field values
3. **No hooks/processors**: Cannot add custom processing to log entries
4. **No sampling**: High-volume logs could overwhelm systems
5. **No compression**: Rotated files are not compressed automatically

### Recommendations

#### For Production Deployments
1. **Use async logging** for better performance
2. **Enable file rotation** to prevent disk space issues
3. **Monitor log file sizes** and adjust rotation settings
4. **Review sensitive key patterns** for your specific use case
5. **Consider log sampling** for high-volume applications

#### For Development
1. **Use detailed logging** (`LogTracedError`) for better debugging
2. **Enable debug level** to see full error chains
3. **Use structured fields** for better log analysis
4. **Test log output** using the test helpers

#### Future Enhancements
1. Add support for custom sensitive key patterns
2. Implement log entry processors/hooks
3. Add log sampling support
4. Add automatic compression for rotated files
5. Add support for remote log destinations
6. Implement log filtering rules

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