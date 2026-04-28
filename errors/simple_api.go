package errors

import (
	"errors"
	"fmt"
)

// toInt attempts to convert any integer type to int.
// Returns the int value and true if successful, 0 and false otherwise.
func toInt(v any) (int, bool) {
	switch val := v.(type) {
	case int:
		return val, true
	case int8:
		return int(val), true
	case int16:
		return int(val), true
	case int32:
		return int(val), true
	case int64:
		return int(val), true
	case uint:
		return int(val), true
	case uint8:
		return int(val), true
	case uint16:
		return int(val), true
	case uint32:
		return int(val), true
	case uint64:
		return int(val), true
	default:
		return 0, false
	}
}

// parseEArgs extracts message, code, and key-value start index from variadic arguments.
// This is shared by E() and WrapE() to avoid duplicate parsing logic.
// Returns:
//   - msg: the extracted error message
//   - code: the extracted error code (-1 if not specified)
//   - startIdx: the index where key-value pairs begin in args
func parseEArgs(args []any) (msg string, code int, startIdx int) {
	code = -1 // Default code

	if len(args) == 0 {
		return "", -1, 0
	}

	// Check if first arg is an integer type (error code)
	if c, ok := toInt(args[0]); ok {
		code = c
		// Second arg should be message
		if len(args) >= 2 {
			if m, ok := args[1].(string); ok {
				msg = m
				startIdx = 2
			} else {
				// If second arg is not string, treat first arg as message
				msg = fmt.Sprintf("%s", args[0])
				code = -1
				startIdx = 1
			}
		} else {
			// Only code provided, no message
			msg = fmt.Sprintf("Error code: %d", code)
			startIdx = 1
		}
	} else {
		// First arg is message (string or any)
		msg, _ = args[0].(string)
		if msg == "" {
			msg = fmt.Sprintf("%s", args[0])
		}
		startIdx = 1
	}

	return msg, code, startIdx
}

// applyContext sets key-value pairs from args starting at startIdx on the error.
func applyContext(err *TracedError, args []any, startIdx int) {
	for i := startIdx; i < len(args); i += 2 {
		if i+1 >= len(args) {
			break
		}

		key, ok := args[i].(string)
		if !ok {
			continue
		}

		err.WithContext(key, args[i+1])
	}
}

// E is a shorthand for creating errors with context
// Usage:
//
//	errors.E("message")  // Simple error
//	errors.E("message", "key1", value1, "key2", value2)  // With context
//	errors.E(404, "message")  // With error code (int)
//	errors.E(404, "message", "key1", value1)  // With code and context
func E(args ...any) error {
	if len(args) == 0 {
		return nil
	}

	msg, code, startIdx := parseEArgs(args)

	// Create error with code, skip 1 for E function
	err := New(msg, code, 1)

	// Process key-value pairs
	applyContext(err, args, startIdx)

	return err
}

// WrapE is a shorthand for wrapping errors with context and optional error code
// Usage:
//
//	WrapE(err, "message")  // Simple wrapping
//	WrapE(err, "message", "key1", value1, "key2", value2)  // With context
//	WrapE(err, 404, "message")  // With error code (int)
//	WrapE(err, 404, "message", "key1", value1)  // With code and context
func WrapE(err error, args ...any) error {
	if err == nil {
		return nil
	}

	if len(args) == 0 {
		return Wrap(err, "")
	}

	msg, code, startIdx := parseEArgs(args)

	// Create wrapped error, skip 1 for WrapE function
	wrapped := Wrap(err, msg, 1)
	if code != -1 {
		wrapped.Code = code
	}

	// Process key-value pairs
	applyContext(wrapped, args, startIdx)

	return wrapped
}

// Must panics if err is not nil
func Must(err error) {
	if err != nil {
		panic(err)
	}
}

// MustValue returns value if err is nil, otherwise panics
func MustValue[T any](value T, err error) T {
	Must(err)
	return value
}

// GetCode returns the error code if available
func GetCode(err error) int {
	if err == nil {
		return -1
	}

	if te, ok := err.(*TracedError); ok {
		return te.Code
	}

	return -1
}

// Is reports whether any error in err's chain matches target.
// It wraps the standard library's errors.Is function.
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As finds the first error in err's chain that matches target, and if so, sets
// target to that error value and returns true. Otherwise, it returns false.
// It wraps the standard library's errors.As function.
func As(err error, target any) bool {
	return errors.As(err, target)
}

// Unwrap returns the result of calling the Unwrap method on err, if err's
// type contains an Unwrap method returning error.
// Otherwise, Unwrap returns nil.
// It wraps the standard library's errors.Unwrap function.
func Unwrap(err error) error {
	return errors.Unwrap(err)
}

// Cause returns the root underlying cause of an error by walking the entire
// Unwrap chain until it reaches the innermost error (i.e., the error that
// has no Unwrap method or returns nil from Unwrap).
//
// This is useful for extracting the original error from a chain of wrapped
// errors, similar to the Cause pattern from github.com/pkg/errors.
//
// If err is nil, returns nil.
// If err has no cause, returns err itself.
func Cause(err error) error {
	if err == nil {
		return nil
	}
	for {
		cause := errors.Unwrap(err)
		if cause == nil {
			return err
		}
		err = cause
	}
}
