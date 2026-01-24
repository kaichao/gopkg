package errors_test

import (
	"fmt"

	"github.com/kaichao/gopkg/errors"
)

func ExampleNew() {
	err := errors.New("file not found")
	fmt.Println(err.Error())
	// Output: file not found
}

func ExampleNew_withCode() {
	err := errors.New("database connection failed", 1001)
	fmt.Printf("Code: %d, Message: %s\n", err.Code, err.Message)
	// Output: Code: 1001, Message: database connection failed
}

func ExampleE() {
	// Simple error
	err1 := errors.E("validation failed")
	fmt.Println(err1.Error())

	// Error with context
	err2 := errors.E("validation failed", "field", "email", "value", "invalid@")
	fmt.Println(err2.Error())

	// Error with code and context
	err3 := errors.E(400, "validation failed", "field", "email")
	fmt.Println(err3.Error())
}

func ExampleWrap() {
	original := errors.New("original error")
	wrapped := errors.Wrap(original, "operation failed")

	fmt.Println(wrapped.Error())
	// Output: operation failed: original error
}

func ExampleWrapE() {
	original := errors.New("database error")

	// Simple wrapping
	wrapped1 := errors.WrapE(original, "query failed")
	fmt.Println(wrapped1.Error())

	// Wrapping with context
	wrapped2 := errors.WrapE(original, "query failed", "table", "users", "query_id", 123)
	fmt.Println(wrapped2.Error())

	// Wrapping with code
	wrapped3 := errors.WrapE(original, 500, "server error")
	fmt.Println(wrapped3.Error())

	// Wrapping with code and context
	wrapped4 := errors.WrapE(original, 404, "not found", "resource", "/api/users", "user_id", 12345)
	fmt.Println(wrapped4.Error())
}

func ExampleTracedError_WithContext() {
	err := errors.New("processing failed").
		WithContext("user_id", 12345).
		WithContext("attempt", 3).
		WithContext("timestamp", "2024-01-24T10:00:00Z")

	fmt.Println(err.Error())
	// Output: processing failed
}

func ExampleTracedError_Format() {
	original := errors.New("original error").
		WithContext("key1", "value1")

	wrapped := errors.Wrap(original, "wrapped error").
		WithContext("key2", "value2")

	fmt.Print(wrapped.Format())
}

func ExampleIsCode() {
	err := errors.New("not found", 404)

	if errors.IsCode(err, 404) {
		fmt.Println("Error is 404")
	}

	if !errors.IsCode(err, 500) {
		fmt.Println("Error is not 500")
	}
	// Output:
	// Error is 404
	// Error is not 500
}

func ExampleGetCode() {
	err1 := errors.New("not found", 404)
	err2 := errors.New("generic error")

	fmt.Println(errors.GetCode(err1))
	fmt.Println(errors.GetCode(err2))
	// Output:
	// 404
	// -1
}

func ExampleMust() {
	// This would panic if there's an error
	// Must(someOperation())

	// Safe usage with defer
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic:", r)
		}
	}()

	// Simulating an error
	err := errors.New("something went wrong")
	errors.Must(err) // This will panic
}

func ExampleMustValue() {
	// Simulating a function that returns a value and error
	getValue := func() (int, error) {
		return 42, nil
	}

	value := errors.MustValue(getValue())
	fmt.Println("Value:", value)

	// Output: Value: 42
}

func ExampleTracedError_GetFullChain() {
	// Create an error chain
	rootErr := errors.New("root cause").
		WithContext("root_key", "root_value")

	middleErr := errors.Wrap(rootErr, "middle error").
		WithContext("middle_key", "middle_value")

	topErr := errors.Wrap(middleErr, "top error").
		WithContext("top_key", "top_value")

	// Print the chain
	// Wrap returns *TracedError, so we can directly use it
	chain := topErr.GetFullChain()
	fmt.Printf("Error chain length: %d\n", len(chain))

	for i, err := range chain {
		fmt.Printf("%d: %s\n", i, err.Message)
	}

	// Output:
	// Error chain length: 3
	// 0: top error: middle error: root cause
	// 1: middle error: root cause
	// 2: root cause
}

func ExampleTracedError_interface() {
	// TracedError works with standard error handling
	var err error = errors.New("traced error")

	// Note: The standard errors.Is and errors.As functions from Go's standard library
	// work with TracedError because it implements Unwrap() and As() methods

	// For demonstration, we show how to check error type
	if te, ok := err.(*errors.TracedError); ok {
		fmt.Println("Error is a TracedError")
		fmt.Printf("Message: %s\n", te.Message)
	}

	// Output:
	// Error is a TracedError
	// Message: traced error
}
