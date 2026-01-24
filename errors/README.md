# Errors Package

Enhanced error handling for Go with tracing, context, and error codes.

## Overview

The `errors` package provides a comprehensive error handling system with:
- Error tracing with stack information
- Context key-value pairs
- Integer error codes
- Error chain support
- Standard `errors` package compatibility

## Core Types

### TracedError
```go
type TracedError struct {
    Message   string         // Error message
    Code      int            // Error code (default -1)
    Location  string         // File:line:function
    Timestamp time.Time      // When it happened
    Context   map[string]any // Context information
    Cause     *TracedError   // Underlying cause
}
```

## Basic Usage

### Creating Errors
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
```

### E() Function - Flexible Error Creation
```go
// Simple error
err := errors.E("validation failed")

// Error with context
err := errors.E("validation failed", "field", "email", "value", "invalid@")

// Error with code
err := errors.E(404, "not found")

// Error with code and context
err := errors.E(400, "validation failed", "field", "email")
```

### Wrapping Errors
```go
original := errors.New("original error")

// Simple wrapping
wrapped := errors.Wrap(original, "operation failed")

// Flexible wrapping with WrapE
wrapped := errors.WrapE(original, "query failed")
wrapped := errors.WrapE(original, "query failed", "table", "users")
wrapped := errors.WrapE(original, 500, "server error")
wrapped := errors.WrapE(original, 404, "not found", "resource", "/api/users")
```

## Helper Functions

### Must and MustValue
```go
// Panic on error
errors.Must(someOperation())

// Get value or panic
value := errors.MustValue(someOperation())
```

### Error Code Utilities
```go
// Check error code
if errors.IsCode(err, 404) {
    // Handle 404 error
}

// Get error code
code := errors.GetCode(err)
```

## Error Chains

### Creating Chains
```go
root := errors.New("root cause")
middle := errors.Wrap(root, "middle error")
top := errors.Wrap(middle, "top error")
```

### Working with Chains
```go
// Get full chain
chain := err.GetFullChain()
for i, errInChain := range chain {
    fmt.Printf("%d: %s\n", i, errInChain.Message)
}

// Format error chain
formatted := err.Format()
fmt.Print(formatted)
```

## Standard Compatibility

`TracedError` implements standard Go error interfaces:

```go
// errors.Is support
if errors.Is(err, targetError) {
    // Error matches
}

// errors.As support
var tracedErr *errors.TracedError
if errors.As(err, &tracedErr) {
    // Error is a TracedError
}

// errors.Unwrap support
cause := errors.Unwrap(err)
```

## Best Practices

1. **Use error codes** for programmatic error handling
2. **Add context** to errors for better debugging
3. **Wrap errors** to preserve error chains
4. **Use E() and WrapE()** for concise error creation
5. **Check error codes** with IsCode() instead of string matching

## Examples

See `example_test.go` for comprehensive usage examples.