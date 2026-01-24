package errors_test

import (
	"testing"

	"github.com/kaichao/gopkg/errors"
)

func TestE(t *testing.T) {
	// Test basic error
	err := errors.E("test error")
	if err == nil {
		t.Fatal("E should return non-nil error")
	}

	// Test error with context
	err = errors.E("test error", "key1", "value1", "key2", 123)
	if err == nil {
		t.Fatal("E should return non-nil error with context")
	}

	te, ok := err.(*errors.TracedError)
	if !ok {
		t.Fatal("E should return TracedError")
	}

	if len(te.Context) != 2 {
		t.Errorf("Expected 2 context items, got %d", len(te.Context))
	}
}

func TestEWithCode(t *testing.T) {
	// Test error with code
	err := errors.E(404, "not found", "key1", "value1")
	if err == nil {
		t.Fatal("E should return non-nil error with code")
	}

	te, ok := err.(*errors.TracedError)
	if !ok {
		t.Fatal("E should return TracedError")
	}

	if te.Code != 404 {
		t.Errorf("Expected code 404, got %d", te.Code)
	}

	if te.Message != "not found" {
		t.Errorf("Expected message 'not found', got %q", te.Message)
	}

	if len(te.Context) != 1 {
		t.Errorf("Expected 1 context item, got %d", len(te.Context))
	}
}

func TestWrapE(t *testing.T) {
	original := errors.New("original error")

	// Test simple wrapping
	wrapped := errors.WrapE(original, "wrapped error")
	if wrapped == nil {
		t.Fatal("WrapE should return non-nil error")
	}

	// Test wrapping with context
	wrapped = errors.WrapE(original, "wrapped error", "key1", "value1")
	if wrapped == nil {
		t.Fatal("WrapE should return non-nil error with context")
	}

	te, ok := wrapped.(*errors.TracedError)
	if !ok {
		t.Fatal("WrapE should return TracedError")
	}

	if te.Cause != original {
		t.Error("WrapE should preserve cause chain")
	}

	if len(te.Context) != 1 {
		t.Errorf("Expected 1 context item, got %d", len(te.Context))
	}
}

func TestWrapEWithCode(t *testing.T) {
	original := errors.New("original error")

	// Test wrapping with code
	wrapped := errors.WrapE(original, 404, "not found")
	if wrapped == nil {
		t.Fatal("WrapE should return non-nil error with code")
	}

	te, ok := wrapped.(*errors.TracedError)
	if !ok {
		t.Fatal("WrapE should return TracedError")
	}

	if te.Code != 404 {
		t.Errorf("Expected code 404, got %d", te.Code)
	}

	if te.Message != "not found: original error" {
		t.Errorf("Expected message 'not found: original error', got %q", te.Message)
	}

	// Test wrapping with code and context
	wrapped = errors.WrapE(original, 500, "server error", "key1", "value1")
	if wrapped == nil {
		t.Fatal("WrapE should return non-nil error with code and context")
	}

	te, ok = wrapped.(*errors.TracedError)
	if !ok {
		t.Fatal("WrapE should return TracedError")
	}

	if te.Code != 500 {
		t.Errorf("Expected code 500, got %d", te.Code)
	}

	if len(te.Context) != 1 {
		t.Errorf("Expected 1 context item, got %d", len(te.Context))
	}
}

func TestMust(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Must should panic on non-nil error")
		}
	}()

	err := errors.New("test error")
	errors.Must(err)
}

func TestMustNoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Error("Must should not panic on nil error")
		}
	}()

	errors.Must(nil)
}

func TestMustValue(t *testing.T) {
	// Test with nil error
	value := errors.MustValue(42, nil)
	if value != 42 {
		t.Errorf("Expected value 42, got %v", value)
	}

	// Test with error (should panic)
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustValue should panic on non-nil error")
		}
	}()

	err := errors.New("test error")
	errors.MustValue(42, err)
}

func TestIsCode(t *testing.T) {
	// Test with coded error
	err := errors.New("test error", 1001)
	if !errors.IsCode(err, 1001) {
		t.Error("IsCode should return true for matching code")
	}

	if errors.IsCode(err, 1002) {
		t.Error("IsCode should return false for non-matching code")
	}

	// Test with non-coded error (default code -1)
	err2 := errors.New("test error")
	if errors.IsCode(err2, 1001) {
		t.Error("IsCode should return false for errors without code")
	}

	// Test with nil error
	if errors.IsCode(nil, 1001) {
		t.Error("IsCode should return false for nil error")
	}
}

func TestGetCode(t *testing.T) {
	// Test with coded error
	err := errors.New("test error", 1001)
	code := errors.GetCode(err)
	if code != 1001 {
		t.Errorf("Expected code 1001, got %d", code)
	}

	// Test with non-coded error (default -1)
	err2 := errors.New("test error")
	code = errors.GetCode(err2)
	if code != -1 {
		t.Errorf("Expected code -1, got %d", code)
	}

	// Test with nil error
	code = errors.GetCode(nil)
	if code != -1 {
		t.Errorf("Expected code -1 for nil error, got %d", code)
	}
}
