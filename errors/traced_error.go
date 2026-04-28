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
func New(msg string, args ...int) *TracedError {
	skip := 1 // Default skip for direct calls
	var code int = -1

	// Parse arguments
	if len(args) == 1 {
		code = args[0]
	} else if len(args) >= 2 {
		code = args[0]
		skip = args[1] + 1 // Add 1 because we're already in New function
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

	// Build the wrapped message, handling empty msg prefix gracefully
	var combinedMsg string
	if msg == "" {
		combinedMsg = err.Error()
	} else {
		combinedMsg = fmt.Sprintf("%s: %s", msg, err)
	}

	tracedErr := &TracedError{
		Message:   combinedMsg,
		Code:      -1, // Default code for wrapped errors
		Location:  fmt.Sprintf("%s:%d:%s", file, line, fn.Name()),
		Timestamp: time.Now(),
		Context:   make(map[string]any),
		cause:     err, // Save the original error
	}

	return tracedErr
}

// WithContext adds context information to the error.
// Note: This mutates the error in place and is intended for use during
// error construction (typically chained right after New/Wrap).
// For concurrent use, create errors in a single goroutine and pass
// them immutably to other goroutines.
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

// Format implements fmt.Formatter for different output formats.
// Supported verbs:
//
//	%v  - error message only (same as Error())
//	%+v - full details including location, timestamp, context, and cause chain
//	%#v - Go-syntax representation showing all struct fields
//	%s  - error message only
//	%q  - quoted error message
func (e *TracedError) Format(f fmt.State, verb rune) {
	switch verb {
	case 'v':
		if f.Flag('+') {
			// %+v: output full details
			fmt.Fprint(f, e.Detailed())
		} else if f.Flag('#') {
			// %#v: Go-syntax representation
			fmt.Fprint(f, e.GoString())
		} else {
			// %v: output message only
			fmt.Fprint(f, e.Message)
		}
	case 's':
		fmt.Fprint(f, e.Message)
	case 'q':
		fmt.Fprintf(f, "%q", e.Message)
	default:
		fmt.Fprint(f, e.Message)
	}
}

// GoString returns a Go-syntax representation of the TracedError.
// This implements the fmt.GoStringer interface and is used by %#v.
func (e *TracedError) GoString() string {
	var sb strings.Builder

	sb.WriteString("&errors.TracedError{")

	fmt.Fprintf(&sb, "Message: %q", e.Message)

	if e.Code != -1 {
		fmt.Fprintf(&sb, ", Code: %d", e.Code)
	}

	if e.Location != "" {
		fmt.Fprintf(&sb, ", Location: %q", e.Location)
	}

	if len(e.Context) > 0 {
		sb.WriteString(", Context: map[string]any{")
		first := true
		for k, v := range e.Context {
			if !first {
				sb.WriteString(", ")
			}
			fmt.Fprintf(&sb, "%q: %#v", k, v)
			first = false
		}
		sb.WriteString("}")
	}

	if e.cause != nil {
		fmt.Fprintf(&sb, ", cause: %#v", e.cause)
	}

	sb.WriteString("}")
	return sb.String()
}

// Detailed returns a formatted error chain with full details.
// This is useful for debugging and logging purposes.
func (e *TracedError) Detailed() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Error: %s\n", e.Message))
	sb.WriteString(fmt.Sprintf("Location: %s\n", e.Location))
	sb.WriteString(fmt.Sprintf("Time: %s\n", e.Timestamp.Format("2006-01-02 15:04:05")))

	if e.Code != -1 {
		sb.WriteString(fmt.Sprintf("Code: %d\n", e.Code))
	}

	if len(e.Context) > 0 {
		sb.WriteString("Context:\n")
		for k, v := range e.Context {
			sb.WriteString(fmt.Sprintf("  %s: %v\n", k, v))
		}
	}

	// Recursively print cause chain using Unwrap() for consistency
	cause := e.Unwrap()
	if cause != nil {
		sb.WriteString("\nCaused by:\n")
		if te, ok := cause.(*TracedError); ok {
			sb.WriteString(te.Detailed())
		} else {
			// Print non-TracedError cause
			sb.WriteString(fmt.Sprintf("  %v\n", cause))
		}
	}

	return sb.String()
}

// GetFullChain returns the complete error chain.
// Unlike Cause(), this follows the full Unwrap() chain including non-TracedError nodes.
// Non-TracedError nodes are wrapped in a TracedError to include them in the chain.
func (e *TracedError) GetFullChain() []*TracedError {
	chain := []*TracedError{e}

	current := e
	for {
		cause := current.Unwrap()
		if cause == nil {
			break
		}

		te, ok := cause.(*TracedError)
		if !ok {
			// Wrap non-TracedError nodes so the chain includes them
			te = &TracedError{
				Message:   cause.Error(),
				Code:      -1,
				Location:  "",
				Timestamp: time.Now(),
				Context:   make(map[string]any),
			}
		}
		chain = append(chain, te)
		current = te
	}

	return chain
}

// Cause returns the underlying TracedError cause (for backward compatibility)
func (e *TracedError) Cause() *TracedError {
	cause := e.Unwrap()
	if cause == nil {
		return nil
	}
	if te, ok := cause.(*TracedError); ok {
		return te
	}
	return nil
}

// Unwrap returns the underlying cause (for errors.Is/As)
func (e *TracedError) Unwrap() error {
	return e.cause
}

// Is checks if this error matches a target (for errors.Is).
// Only returns true for the exact same instance (pointer equality).
func (e *TracedError) Is(target error) bool {
	te, ok := target.(*TracedError)
	if !ok {
		return false
	}
	return e == te
}

// As checks if this error can be converted to target (for errors.As)
func (e *TracedError) As(target any) bool {
	if target == nil {
		return false
	}

	// Handle **TracedError targets
	if t, ok := target.(**TracedError); ok {
		if t == nil {
			return false
		}
		*t = e
		return true
	}

	// For all other target types, return false and let errors.As walk the Unwrap chain
	return false
}
