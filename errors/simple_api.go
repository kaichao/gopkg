package errors

import "fmt"

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

	var msg string
	var code int = -1 // Default code
	var startIdx int

	// Check if first arg is an int (error code)
	if c, ok := args[0].(int); ok {
		code = c
		// Second arg should be message
		if len(args) >= 2 {
			if m, ok := args[1].(string); ok {
				msg = m
				startIdx = 2
			} else {
				// If second arg is not string, treat first arg as message
				msg = fmt.Sprintf("%v", args[0])
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
			msg = fmt.Sprintf("%v", args[0])
		}
		startIdx = 1
	}

	// Create error with code, skip 1 for E function
	err := New(msg, code, 1)

	// Process key-value pairs
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

	var msg string
	var code int = -1 // Default code
	var startIdx int

	// Check if first arg is an int (error code)
	if c, ok := args[0].(int); ok {
		code = c
		// Second arg should be message
		if len(args) >= 2 {
			if m, ok := args[1].(string); ok {
				msg = m
				startIdx = 2
			} else {
				// If second arg is not string, treat first arg as message
				msg = fmt.Sprintf("%v", args[0])
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
			msg = fmt.Sprintf("%v", args[0])
		}
		startIdx = 1
	}

	// Create wrapped error, skip 1 for WrapE function
	wrapped := Wrap(err, msg, 1)
	if code != -1 {
		wrapped.Code = code
	}

	// Process key-value pairs
	for i := startIdx; i < len(args); i += 2 {
		if i+1 >= len(args) {
			break
		}

		key, ok := args[i].(string)
		if !ok {
			continue
		}

		wrapped.WithContext(key, args[i+1])
	}

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
