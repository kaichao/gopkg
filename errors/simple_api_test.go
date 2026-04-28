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
	// Test error with code (int)
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

func TestEWithInt32Code(t *testing.T) {
	// Test error with int32 code (should be auto-converted to int)
	var code int32 = 1001
	err := errors.E(code, "int32 error", "key1", "value1")
	if err == nil {
		t.Fatal("E should return non-nil error with int32 code")
	}

	te, ok := err.(*errors.TracedError)
	if !ok {
		t.Fatal("E should return TracedError")
	}

	if te.Code != 1001 {
		t.Errorf("Expected code 1001, got %d", te.Code)
	}

	if te.Message != "int32 error" {
		t.Errorf("Expected message 'int32 error', got %q", te.Message)
	}

	if len(te.Context) != 1 {
		t.Errorf("Expected 1 context item, got %d", len(te.Context))
	}
}

func TestEWithInt64Code(t *testing.T) {
	// Test error with int64 code (should be auto-converted to int)
	var code int64 = 2002
	err := errors.E(code, "int64 error")
	if err == nil {
		t.Fatal("E should return non-nil error with int64 code")
	}

	te, ok := err.(*errors.TracedError)
	if !ok {
		t.Fatal("E should return TracedError")
	}

	if te.Code != 2002 {
		t.Errorf("Expected code 2002, got %d", te.Code)
	}

	if te.Message != "int64 error" {
		t.Errorf("Expected message 'int64 error', got %q", te.Message)
	}
}

func TestEWithUintCode(t *testing.T) {
	// Test error with uint code (should be auto-converted to int)
	var code uint = 3003
	err := errors.E(code, "uint error")
	if err == nil {
		t.Fatal("E should return non-nil error with uint code")
	}

	te, ok := err.(*errors.TracedError)
	if !ok {
		t.Fatal("E should return TracedError")
	}

	if te.Code != 3003 {
		t.Errorf("Expected code 3003, got %d", te.Code)
	}

	if te.Message != "uint error" {
		t.Errorf("Expected message 'uint error', got %q", te.Message)
	}
}

func TestEWithInt8Code(t *testing.T) {
	// Test error with int8 code (should be auto-converted to int)
	var code int8 = 127
	err := errors.E(code, "int8 error")
	if err == nil {
		t.Fatal("E should return non-nil error with int8 code")
	}

	te, ok := err.(*errors.TracedError)
	if !ok {
		t.Fatal("E should return TracedError")
	}

	if te.Code != 127 {
		t.Errorf("Expected code 127, got %d", te.Code)
	}

	if te.Message != "int8 error" {
		t.Errorf("Expected message 'int8 error', got %q", te.Message)
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

	if te.Cause() != original {
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
