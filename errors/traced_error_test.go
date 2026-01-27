package errors_test

import (
	"testing"

	"github.com/kaichao/gopkg/errors"
)

func TestNew(t *testing.T) {
	err := errors.New("test error")
	if err == nil {
		t.Fatal("New should return non-nil error")
	}

	if err.Message != "test error" {
		t.Errorf("Expected message 'test error', got %q", err.Message)
	}

	if err.Location == "" {
		t.Error("Location should be set")
	}

	if err.Timestamp.IsZero() {
		t.Error("Timestamp should be set")
	}
}

func TestNewWithCode(t *testing.T) {
	err := errors.New("test error", 1001)
	if err == nil {
		t.Fatal("New should return non-nil error with code")
	}

	if err.Message != "test error" {
		t.Errorf("Expected message 'test error', got %q", err.Message)
	}

	if err.Code != 1001 {
		t.Errorf("Expected code 1001, got %d", err.Code)
	}
}

func TestWrap(t *testing.T) {
	original := errors.New("original error")
	wrapped := errors.Wrap(original, "wrapped error")

	if wrapped == nil {
		t.Fatal("Wrap should return non-nil error")
	}

	if wrapped.Cause() != original {
		t.Error("Wrap should preserve cause chain")
	}

	if wrapped.Message != "wrapped error: original error" {
		t.Errorf("Unexpected wrapped message: %q", wrapped.Message)
	}
}

func TestWithContext(t *testing.T) {
	err := errors.New("test error").
		WithContext("key1", "value1").
		WithContext("key2", 123)

	if len(err.Context) != 2 {
		t.Errorf("Expected 2 context items, got %d", len(err.Context))
	}

	if err.Context["key1"] != "value1" {
		t.Errorf("Expected key1='value1', got %v", err.Context["key1"])
	}

	if err.Context["key2"] != 123 {
		t.Errorf("Expected key2=123, got %v", err.Context["key2"])
	}
}

func TestErrorInterface(t *testing.T) {
	err := errors.New("test error")
	errStr := err.Error()

	if errStr != "test error" {
		t.Errorf("Error() should return message, got %q", errStr)
	}
}

func TestFormat(t *testing.T) {
	err := errors.New("test error").
		WithContext("key", "value")

	formatted := err.Format()
	if formatted == "" {
		t.Error("Format should return non-empty string")
	}

	// Check that key parts are present
	if !contains(formatted, "Error: test error") {
		t.Error("Format should contain error message")
	}
	if !contains(formatted, "Location:") {
		t.Error("Format should contain location")
	}
	if !contains(formatted, "Context:") {
		t.Error("Format should contain context")
	}
}

func TestGetFullChain(t *testing.T) {
	original := errors.New("original")
	wrapped := errors.Wrap(original, "wrapped")
	doubleWrapped := errors.Wrap(wrapped, "double wrapped")

	chain := doubleWrapped.GetFullChain()
	if len(chain) != 3 {
		t.Errorf("Expected chain length 3, got %d", len(chain))
	}

	if chain[0].Message != "double wrapped: wrapped: original" {
		t.Errorf("Unexpected chain[0] message: %q", chain[0].Message)
	}
	if chain[1].Message != "wrapped: original" {
		t.Errorf("Unexpected chain[1] message: %q", chain[1].Message)
	}
	if chain[2].Message != "original" {
		t.Errorf("Unexpected chain[2] message: %q", chain[2].Message)
	}
}

func TestUnwrap(t *testing.T) {
	original := errors.New("original")
	wrapped := errors.Wrap(original, "wrapped")

	unwrapped := wrapped.Unwrap()
	if unwrapped != original {
		t.Error("Unwrap should return the cause")
	}

	// Test nil cause
	if original.Unwrap() != nil {
		t.Error("Unwrap should return nil for errors without cause")
	}
}

func TestIs(t *testing.T) {
	err1 := errors.New("test error")
	err2 := errors.New("test error")
	err3 := errors.New("different error")

	if !err1.Is(err2) {
		t.Error("Is should return true for errors with same message")
	}

	if err1.Is(err3) {
		t.Error("Is should return false for errors with different messages")
	}

	// Test with error codes
	errCode1 := errors.New("error", 1001)
	errCode2 := errors.New("error", 1001)
	errCode3 := errors.New("error", 1002)

	if !errCode1.Is(errCode2) {
		t.Error("Is should return true for errors with same code")
	}

	if errCode1.Is(errCode3) {
		t.Error("Is should return false for errors with different codes")
	}

	// Test instance equality
	if !err1.Is(err1) {
		t.Error("Is should return true for same instance")
	}
}

func TestAs(t *testing.T) {
	err := errors.New("test error")
	var target *errors.TracedError

	if !err.As(&target) {
		t.Error("As should convert to TracedError")
	}

	if target != err {
		t.Error("As should set target to the error")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || contains(s[1:], substr)))
}
