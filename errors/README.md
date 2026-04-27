# errors

Enhanced error handling for Go with tracing, context, and error codes.

## Features

- Error tracing with file:line:function location information
- Context key-value pairs for structured error data
- Integer error codes for programmatic handling
- Error chain support (compatible with standard errors.Is/As)
- Flexible error creation with E() and WrapE() shorthand
- `fmt.Formatter` support: `%v` for message, `%+v` for full details
- Root cause extraction with `Cause()`
- Stack trace capture when creating/wrapping errors

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

// Wrap existing errors
wrapped := errors.Wrap(originalErr, "operation failed")

// Error chain inspection
if errors.Is(wrapped, sql.ErrNoRows) {
    // Handle specific error type
}
```

## Core Types

### TracedError
```go
type TracedError struct {
    Message   string         // Error message
    Code      int            // Error code (default -1)
    Location  string         // File:line:function
    Timestamp time.Time      // When it happened
    Context   map[string]any // Context information
}
```

## Functions

### New
Create a new traced error.
```go
errors.New("message")
errors.New("message", code)
```

### E
Flexible error creation with context.
```go
errors.E("message")
errors.E("message", "key", value)
errors.E(code, "message", "key", value)
```

### Wrap
Wrap an existing error.
```go
errors.Wrap(err, "wrapping message")
```

### WrapE
Wrap with flexible syntax.
```go
errors.WrapE(err, "message")
errors.WrapE(err, "message", "key", value)
errors.WrapE(err, code, "message")
```

### Cause
Extract the root cause from an error chain.
```go
errors.Cause(err)
```

### GetCode
Retrieve the error code from a TracedError, returning -1 if not found.
```go
code := errors.GetCode(err)
```

### Is/As
Compatible with standard errors package.
```go
errors.Is(err, target)
errors.As(err, &target)
```

### Formatting
TracedError implements fmt.Formatter for verbose output:
```go
fmt.Printf("%v", err)   // Error message only
fmt.Printf("%+v", err)  // Full details with location, timestamp, context, and cause chain
```

## Examples

See `examples/` directory for complete examples.

## Documentation

Run `go doc github.com/kaichao/gopkg/errors` for API documentation.