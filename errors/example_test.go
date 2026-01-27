package errors_test

import (
	"database/sql"
	stderrors "errors"
	"fmt"

	"github.com/kaichao/gopkg/errors"
)

// ====== 常用函数示例 ======

func ExampleNew() {
	// 创建基本错误
	err := errors.New("file not found")
	fmt.Println(err.Error())
	// Output: file not found
}

func ExampleNew_withCode() {
	// 创建带错误码的错误
	err := errors.New("database connection failed", 1001)
	fmt.Printf("Code: %d, Message: %s\n", err.Code, err.Message)
	// Output: Code: 1001, Message: database connection failed
}

func ExampleE() {
	// 简单错误
	err1 := errors.E("validation failed")
	fmt.Println(err1.Error())

	// 带上下文的错误
	err2 := errors.E("validation failed", "field", "email", "value", "invalid@")
	fmt.Println(err2.Error())

	// 带错误码和上下文的错误
	err3 := errors.E(400, "validation failed", "field", "email")
	fmt.Println(err3.Error())
}

func ExampleWrap() {
	// 包装错误
	original := errors.New("original error")
	wrapped := errors.Wrap(original, "operation failed")

	fmt.Println(wrapped.Error())
	// Output: operation failed: original error
}

func ExampleWrap_standardError() {
	// 包装标准错误（如 sql.ErrNoRows）
	wrapped := errors.Wrap(sql.ErrNoRows, "query failed")

	// 使用标准库的 errors.Is 检查底层错误
	if stderrors.Is(wrapped, sql.ErrNoRows) {
		fmt.Println("Found sql.ErrNoRows in error chain")
	}

	// Output: Found sql.ErrNoRows in error chain
}

func ExampleTracedError_WithContext() {
	// 添加上下文信息
	err := errors.New("processing failed").
		WithContext("user_id", 12345).
		WithContext("attempt", 3).
		WithContext("timestamp", "2024-01-24T10:00:00Z")

	fmt.Println(err.Error())
	// Output: processing failed
}

func ExampleGetCode() {
	// 获取错误码
	err1 := errors.New("not found", 404)
	err2 := errors.New("generic error")

	fmt.Println(errors.GetCode(err1))
	fmt.Println(errors.GetCode(err2))
	// Output:
	// 404
	// -1
}

// ====== errors.Is 和 errors.As 示例 ======

func Example_errorsIs() {
	// 使用标准库的 errors.Is 检查错误链

	// 包装标准错误
	wrappedErr := errors.Wrap(sql.ErrNoRows, "database query failed")

	// 检查错误链中是否包含 sql.ErrNoRows
	if stderrors.Is(wrappedErr, sql.ErrNoRows) {
		fmt.Println("Error chain contains sql.ErrNoRows")
	}

	// 检查 TracedError 实例
	err1 := errors.New("not found", 404)
	err2 := errors.New("not found", 404)

	if stderrors.Is(err1, err2) {
		fmt.Println("Errors match by code")
	}

	// Output:
	// Error chain contains sql.ErrNoRows
	// Errors match by code
}

func Example_errorsAs() {
	// 使用标准库的 errors.As 转换错误类型

	// 创建错误链
	original := errors.New("original error")
	wrapped := errors.Wrap(original, "wrapped error")

	// 转换为 TracedError
	var tracedErr *errors.TracedError
	if stderrors.As(wrapped, &tracedErr) {
		fmt.Printf("Successfully converted to TracedError: %s\n", tracedErr.Message)
	}

	// Output: Successfully converted to TracedError: wrapped error: original error
}

func ExampleTracedError_Unwrap() {
	// 使用 Unwrap 遍历错误链

	// 创建错误链
	root := errors.New("root error")
	middle := errors.Wrap(root, "middle error")
	top := errors.Wrap(middle, "top error")

	// 手动遍历错误链
	current := top
	for current != nil {
		fmt.Println(current.Message)
		if unwrapped := stderrors.Unwrap(current); unwrapped != nil {
			if te, ok := unwrapped.(*errors.TracedError); ok {
				current = te
			} else {
				current = nil
			}
		} else {
			current = nil
		}
	}

	// Output:
	// top error: middle error: root error
	// middle error: root error
	// root error
}

func ExampleTracedError_Cause() {
	// 使用 Cause() 方法获取底层 TracedError

	// 创建 TracedError 链
	root := errors.New("root error")
	middle := errors.Wrap(root, "middle error")

	// 获取底层原因
	if cause := middle.Cause(); cause != nil {
		fmt.Printf("Cause: %s\n", cause.Message)
	}

	// 包装标准错误
	wrappedStd := errors.Wrap(sql.ErrNoRows, "query failed")
	if cause := wrappedStd.Cause(); cause == nil {
		fmt.Println("No TracedError cause for standard error")
	}

	// Output:
	// Cause: root error
	// No TracedError cause for standard error
}

// ====== 其他常用函数示例 ======

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

func ExampleTracedError_Format() {
	original := errors.New("original error").
		WithContext("key1", "value1")

	wrapped := errors.Wrap(original, "wrapped error").
		WithContext("key2", "value2")

	fmt.Print(wrapped.Format())
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
