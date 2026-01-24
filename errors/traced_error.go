package errors

import (
	"fmt"
	"runtime"
	"strings"
	"time"
)

// TracedError 追踪错误
type TracedError struct {
	Message   string         // Error message
	Code      int            // Error code for programmatic handling (optional, default -1)
	Location  string         // Where it happened (file:line:function)
	Timestamp time.Time      // When it happened
	Context   map[string]any // Context information
	Cause     *TracedError   // Underlying cause
}

// New creates a new traced error
// Usage: New("message") or New("message", code) where code is int
func New(msg string, code ...int) *TracedError {
	pc, file, line, _ := runtime.Caller(1)
	fn := runtime.FuncForPC(pc)

	err := &TracedError{
		Message:   msg,
		Code:      -1, // Default code
		Location:  fmt.Sprintf("%s:%d:%s", file, line, fn.Name()),
		Timestamp: time.Now(),
		Context:   make(map[string]any),
	}

	// Set code if provided
	if len(code) > 0 {
		err.Code = code[0]
	}

	return err
}

// Wrap wraps an existing error with context
func Wrap(err error, msg string) *TracedError {
	if err == nil {
		return nil
	}

	pc, file, line, _ := runtime.Caller(1)
	fn := runtime.FuncForPC(pc)

	tracedErr := &TracedError{
		Message:   fmt.Sprintf("%s: %v", msg, err),
		Location:  fmt.Sprintf("%s:%d:%s", file, line, fn.Name()),
		Timestamp: time.Now(),
		Context:   make(map[string]any),
	}

	// If it's already a TracedError, preserve the chain
	if te, ok := err.(*TracedError); ok {
		tracedErr.Cause = te
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
func (e *TracedError) Error() string {
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
	if e.Cause != nil {
		sb.WriteString("\nCaused by:\n")
		sb.WriteString(e.Cause.Format())
	}

	return sb.String()
}

// GetFullChain returns the complete error chain
func (e *TracedError) GetFullChain() []*TracedError {
	chain := []*TracedError{e}

	current := e
	for current.Cause != nil {
		chain = append(chain, current.Cause)
		current = current.Cause
	}

	return chain
}

// Unwrap returns the underlying cause (for errors.Is/As)
func (e *TracedError) Unwrap() error {
	if e.Cause != nil {
		return e.Cause
	}
	return nil
}

// Is checks if this error matches a target (for errors.Is)
func (e *TracedError) Is(target error) bool {
	if target == nil {
		return e == nil
	}

	// Check if target is a TracedError
	if te, ok := target.(*TracedError); ok {
		// Match by code if both have codes (and codes are not default -1)
		if e.Code != -1 && te.Code != -1 {
			return e.Code == te.Code
		}
		// Otherwise match by message
		return e.Message == te.Message
	}

	// Support standard error comparison
	return e.Message == target.Error()
}

// As checks if this error can be converted to target (for errors.As)
func (e *TracedError) As(target any) bool {
	if te, ok := target.(**TracedError); ok {
		*te = e
		return true
	}
	return false
}
