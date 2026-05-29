# errors

Enhanced error handling for Go with tracing, context, and error codes.

## Features

- Error tracing with file:line:function location information
- Context key-value pairs for structured error data
- Integer error codes for programmatic handling (default: 1)
- Error chain support (compatible with standard errors.Is/As)
- Flexible error creation with `E()` and `WrapE()` shorthand
- `fmt.Formatter` support: `%v` for message, `%+v` for full details, `%#v` for Go-syntax
- Root cause extraction with `Cause()`
- `UsageError` type for CLI usage errors

## Quick Start

```go
import "github.com/kaichao/gopkg/errors"

// Simple error
err := errors.New("file not found")

// Error with code
err := errors.New("database connection failed", 1001)

// Error with context
err := errors.New("validation failed").
    WithContext("field", "email").
    WithContext("value", "invalid@example.com")

// Flexible creation with E()
err := errors.E("validation failed", "field", "email", "code", 400)

// With error code via E()
err := errors.E(404, "user not found", "user_id", 123)

// Wrap existing errors
wrapped := errors.Wrap(originalErr, "operation failed")

// Wrap with flexible syntax
wrapped := errors.WrapE(originalErr, 500, "internal error", "service", "auth")

// Error chain inspection
if errors.Is(wrapped, sql.ErrNoRows) {
    // Handle specific error type
}

// Get root cause
root := errors.Cause(wrapped)

// Extract error code (0=nil, -1=not TracedError)
code := errors.GetCode(err)

// Usage error (signals CLI should show help)
usageErr := errors.NewUsage("missing required flag")
```

## Core Types

### TracedError
```go
type TracedError struct {
    Message   string         // Error message
    Code      int            // Error code (default 1)
    Location  string         // File:line:function
    Timestamp time.Time      // When it happened
    Context   map[string]any // Context information
}
```

### UsageError
```go
type UsageError struct { ... }
func NewUsage(msg string) *UsageError  // Signals incorrect CLI usage
```

## Functions

### Creation
| Function | Description |
|----------|-------------|
| `New(msg, args...int)` | Create traced error, optional code |
| `E(args...any)` | Flexible: `E("msg")`, `E(code, "msg")`, `E("msg", "k", v)`, `E(code, "msg", "k", v)` |
| `NewUsage(msg)` | Create usage error for CLI help display |

### Wrapping
| Function | Description |
|----------|-------------|
| `Wrap(err, msg, skip...int)` | Wrap error with message |
| `WrapE(err, args...any)` | Flexible wrap like `E()` |

### Inspection
| Function | Description |
|----------|-------------|
| `GetCode(err)` | 0 if nil, error code if TracedError, -1 otherwise |
| `Cause(err)` | Root cause of error chain |
| `Is(err, target)` | Wraps stdlib `errors.Is` |
| `As(err, target)` | Wraps stdlib `errors.As` |
| `Unwrap(err)` | Wraps stdlib `errors.Unwrap` |

### Other
| Function | Description |
|----------|-------------|
| `Must(err)` | Panics if err != nil |
| `MustValue[T](val, err)` | Returns val or panics |

## Formatting

```go
fmt.Printf("%v", err)   // Error message only
fmt.Printf("%+v", err)  // Full details: location, timestamp, context, cause chain
fmt.Printf("%#v", err)  // Go-syntax representation of all struct fields
```

## Examples

See `examples/` directory for complete examples.

## Documentation

Run `go doc github.com/kaichao/gopkg/errors` for API documentation.
