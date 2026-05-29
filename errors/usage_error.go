package errors

// UsageError signals that the error is caused by incorrect usage (e.g., missing
// required flags) and the caller should display usage/help information.
type UsageError struct {
	msg string
}

// Error implements the error interface.
func (e *UsageError) Error() string {
	return e.msg
}

// NewUsage creates a UsageError with the given message.
func NewUsage(msg string) *UsageError {
	return &UsageError{msg: msg}
}
