package errors_test

import (
	"database/sql"
	"errors"
	"testing"

	gopkgerrors "github.com/kaichao/gopkg/errors"
)

func TestIsWithStandardError(t *testing.T) {
	// Test that errors.Is works with standard errors like sql.ErrNoRows
	err := gopkgerrors.Wrap(sql.ErrNoRows, "query failed")

	// This should work because Unwrap() returns sql.ErrNoRows
	if !errors.Is(err, sql.ErrNoRows) {
		t.Error("errors.Is should find sql.ErrNoRows in the error chain")
	}

	// Test with a non-wrapped error
	err2 := gopkgerrors.New("not found")
	if errors.Is(err2, sql.ErrNoRows) {
		t.Error("errors.Is should not match unrelated errors")
	}
}

func TestAsWithTracedError(t *testing.T) {
	// Test that errors.As works with TracedError
	original := gopkgerrors.New("test error")
	wrapped := gopkgerrors.Wrap(original, "wrapped")

	var target *gopkgerrors.TracedError
	if !errors.As(wrapped, &target) {
		t.Error("errors.As should convert to TracedError")
	}

	if target != wrapped {
		t.Error("errors.As should set target to the wrapped error")
	}

	// Test with error interface - errors.As expects a pointer to a type that implements error
	// We can't use *error directly, but we can test with a concrete type
	var errTarget *gopkgerrors.TracedError
	if !errors.As(wrapped, &errTarget) {
		t.Error("errors.As should convert to TracedError")
	}

	if errTarget != wrapped {
		t.Error("errors.As should set target correctly")
	}
}

func TestUnwrapChain(t *testing.T) {
	// Test complex unwrap chain with mixed error types
	stdErr := errors.New("standard error")
	tracedErr := gopkgerrors.New("traced error")

	// Create a chain: wrapped1 -> stdErr
	wrapped1 := gopkgerrors.Wrap(stdErr, "wrapped std error")

	// Create another chain: wrapped2 -> tracedErr
	wrapped2 := gopkgerrors.Wrap(tracedErr, "wrapped traced error")

	// Test unwrapping
	if unwrapped := wrapped1.Unwrap(); unwrapped != stdErr {
		t.Errorf("Unwrap should return stdErr, got %v", unwrapped)
	}

	if unwrapped := wrapped2.Unwrap(); unwrapped != tracedErr {
		t.Errorf("Unwrap should return tracedErr, got %v", unwrapped)
	}

	// Test errors.Is through the chain
	if !errors.Is(wrapped1, stdErr) {
		t.Error("errors.Is should find stdErr in chain")
	}

	if !errors.Is(wrapped2, tracedErr) {
		t.Error("errors.Is should find tracedErr in chain")
	}
}

func TestIsInstanceEquality(t *testing.T) {
	// Test that Is() returns true for the same instance
	err := gopkgerrors.New("test error")
	if !err.Is(err) {
		t.Error("Is should return true for same instance")
	}

	// Test with different instances - should NOT match (pointer equality only)
	err2 := gopkgerrors.New("test error")
	if err.Is(err2) {
		t.Error("Is should return false for different instances")
	}

	// Test with different code
	err3 := gopkgerrors.New("test error", 1001)
	err4 := gopkgerrors.New("test error", 1002)
	if err3.Is(err4) {
		t.Error("Is should return false for different instances with different codes")
	}
}

func TestAsTypeSafety(t *testing.T) {
	// Test that As handles nil targets safely
	err := gopkgerrors.New("test error")

	// These should not panic
	if err.As(nil) {
		t.Error("As should return false for nil target")
	}

	// Test with nil pointer target - this should work because &nilPtr is not nil
	var nilPtr *gopkgerrors.TracedError
	if !err.As(&nilPtr) {
		t.Error("As should accept a pointer to nil pointer")
	}
	if nilPtr != err {
		t.Error("As should set the nil pointer to the error")
	}

	// Test with invalid target type
	var invalid int
	if err.As(&invalid) {
		t.Error("As should return false for invalid target type")
	}
}
