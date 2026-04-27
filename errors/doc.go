// Package errors provides enhanced error handling for Go with tracing, context, and error codes.
//
// # Features
//
//   - Error tracing with file:line:function location information
//   - Stack trace capture (via callers()) when creating/wrapping errors
//   - Context key-value pairs for structured error data
//   - Integer error codes for programmatic handling
//   - Error chain support (compatible with standard errors.Is/As)
//   - Flexible error creation with E() and WrapE() shorthand
//   - fmt.Formatter support: %v for message, %+v for full details
//
// # Basic Usage
//
// Create simple traced errors:
//
//	err := errors.New("file not found")
//
// Errors with codes:
//
//	err := errors.New("database connection failed", 1001)
//
// Errors with context:
//
//	err := errors.New("validation failed").
//	    WithContext("field", "email").
//	    WithContext("value", "invalid@example.com")
//
// Flexible creation with E():
//
//	err := errors.E("validation failed", "field", "email", "code", 400)
//
// Wrap existing errors:
//
//	wrapped := errors.Wrap(originalErr, "operation failed")
//
// # Error Chain Inspection
//
// Compatible with the standard library's errors package:
//
//	if errors.Is(wrapped, sql.ErrNoRows) {
//	    // Handle specific error type
//	}
//
//	var target *errors.TracedError
//	if errors.As(wrapped, &target) {
//	    // Use target
//	}
//
// Extract the root cause:
//
//	root := errors.Cause(wrapped)
//
// # Formatting
//
// The TracedError type implements fmt.Formatter:
//
//	fmt.Printf("%v", err)   // Error message only
//	fmt.Printf("%+v", err)  // Full details with location, timestamp, context, and cause chain
//
// # Thread Safety
//
// TracedError instances should be created and configured (via WithContext
// chaining) within a single goroutine. Once constructed, they can safely
// be passed to and used by multiple goroutines for read-only access
// (e.g., comparing with errors.Is, printing with fmt.Printf).
//
// For examples, see the examples/ directory.
package errors
