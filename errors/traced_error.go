package errors

import (
	"fmt"
	"runtime"
	"strings"
	"time"
)

// TracedError represents an error with tracing information.
type TracedError struct {
	Message   string         // Error message
	Code      int            // Error code for programmatic handling (optional, default -1)
	Location  string         // Where it happened (file:line:function)
	Timestamp time.Time      // When it happened
	Context   map[string]any // Context information
	cause     error          // Underlying cause (private, can be any error type)
}

// New creates a new traced error
// Usage:
//
//	New("message")                    // Simple error
//	New("message", code)              // Error with code
//	New("message", code, skip)        // Error with code and custom skip
//	New("message", skip)              // Error with custom skip (no code)
func New(msg string, args ...any) *TracedError {
	skip := 1 // Default skip for direct calls
	var code int = -1

	// Parse arguments
	if len(args) == 1 {
		// Single argument: could be code or skip
		if c, ok := args[0].(int); ok {
			code = c
		}
	} else if len(args) == 2 {
		// Two arguments: first is code, second is skip
		if c, ok := args[0].(int); ok {
			code = c
		}
		if s, ok := args[1].(int); ok {
			skip = s + 1 // Add 1 because we're already in New function
		}
	}

	pc, file, line, _ := runtime.Caller(skip)
	fn := runtime.FuncForPC(pc)

	err := &TracedError{
		Message:   msg,
		Code:      code,
		Location:  fmt.Sprintf("%s:%d:%s", file, line, fn.Name()),
		Timestamp: time.Now(),
		Context:   make(map[string]any),
	}

	return err
}

// Wrap wraps an existing error with context
// The optional skip parameter specifies how many stack frames to skip
func Wrap(err error, msg string, skip ...int) *TracedError {
	if err == nil {
		return nil
	}

	skipCount := 1 // Default skip for direct calls
	if len(skip) > 0 {
		skipCount = skip[0] + 1 // Add 1 because we're already in Wrap function
	}

	pc, file, line, _ := runtime.Caller(skipCount)
	fn := runtime.FuncForPC(pc)

	tracedErr := &TracedError{
		Message:   fmt.Sprintf("%s: %s", msg, err),
		Code:      -1, // Default code for wrapped errors
		Location:  fmt.Sprintf("%s:%d:%s", file, line, fn.Name()),
		Timestamp: time.Now(),
		Context:   make(map[string]any),
		cause:     err, // Save the original error
	}

	return tracedErr
}

// WithContext adds context information to the error
func (e *TracedError) WithContext(key string, value any) *TracedError {
	if e.Context == nil {
		e.Context = make(map[string]any)
	}
	e.Context[key] = value
	return e
}

// Error implements the error interface
// Returns the error message, preserving original formatting including newlines
func (e *TracedError) Error() string {
	// Preserve the original message formatting including newlines and whitespace
	// Don't trim or normalize whitespace
	return e.Message
}

// Format returns a formatted error chain
func (e *TracedError) Format() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Error: %s\n", e.Message))
	sb.WriteString(fmt.Sprintf("Location: %s\n", e.Location))
	sb.WriteString(fmt.Sprintf("Time: %s\n", e.Timestamp.Format("2006-01-02 15:04:05")))

	if len(e.Context) > 0 {
		sb.WriteString("Context:\n")
		for k, v := range e.Context {
			sb.WriteString(fmt.Sprintf("  %s: %v\n", k, v))
		}
	}

	// Recursively print cause chain
	if cause := e.Cause(); cause != nil {
		sb.WriteString("\nCaused by:\n")
		sb.WriteString(cause.Format())
	} else if e.cause != nil {
		// Print non-TracedError cause
		sb.WriteString("\nCaused by:\n")
		sb.WriteString(fmt.Sprintf("  %v\n", e.cause))
	}

	return sb.String()
}

// GetFullChain returns the complete error chain
func (e *TracedError) GetFullChain() []*TracedError {
	chain := []*TracedError{e}

	current := e
	for {
		cause := current.Cause()
		if cause == nil {
			break
		}
		chain = append(chain, cause)
		current = cause
	}

	return chain
}

// Cause returns the underlying TracedError cause (for backward compatibility)
func (e *TracedError) Cause() *TracedError {
	if e.cause == nil {
		return nil
	}
	if te, ok := e.cause.(*TracedError); ok {
		return te
	}
	return nil
}

// Unwrap returns the underlying cause (for errors.Is/As)
func (e *TracedError) Unwrap() error {
	return e.cause
}

// Is checks if this error matches a target (for errors.Is)
func (e *TracedError) Is(target error) bool {
	if target == nil {
		return e == nil
	}

	// Check if it's the same instance
	if te, ok := target.(*TracedError); ok && e == te {
		return true
	}

	// Check if target is a TracedError for semantic equality
	if te, ok := target.(*TracedError); ok {
		// Match by code if both have codes (and codes are not default -1)
		if e.Code != -1 && te.Code != -1 {
			return e.Code == te.Code
		}
		// Otherwise match by message
		return e.Message == te.Message
	}

	// Don't handle other specific error types - let the standard library handle them
	return false
}

// As checks if this error can be converted to target (for errors.As)
func (e *TracedError) As(target any) bool {
	if target == nil {
		return false
	}

	// Use type assertion with reflection for safety
	switch t := target.(type) {
	case **TracedError:
		if t == nil {
			return false
		}
		*t = e
		return true
	case *error:
		if t == nil {
			return false
		}
		*t = e
		return true
	}

	return false
}
