// Package errors provides enhanced error handling for Go with tracing, context, and error codes.
//
// Features:
// - Error tracing with stack information
// - Context key-value pairs for structured error data
// - Integer error codes for programmatic handling
// - Error chain support (compatible with standard errors package)
// - Flexible error creation with E() function
//
// Basic usage:
//
//	// Create a simple traced error
//	err := errors.New("file not found")
//
//	// Error with code
//	err := errors.New("database connection failed", 1001)
//
//	// Error with context
//	err := errors.New("validation failed").
//	    WithContext("field", "email").
//	    WithContext("value", "invalid@example.com")
//
//	// Flexible creation with E()
//	err := errors.E("validation failed", "field", "email", "code", 400)
//
//	// Wrap existing errors
//	wrapped := errors.Wrap(originalErr, "operation failed")
//
//	// Error chain inspection
//	if errors.Is(wrapped, sql.ErrNoRows) {
//	    // Handle specific error type
//	}
//
// For examples, see the examples/ directory.
package errors
