// Basic error handling examples
package main

import (
	"database/sql"
	"errors"
	"fmt"

	pkgerrors "github.com/kaichao/gopkg/errors"
)

func main() {
	fmt.Println("=== Basic Error Handling ===")

	// 1. Create simple errors
	fmt.Println("\n1. Simple Errors:")
	err1 := pkgerrors.New("file not found")
	fmt.Printf("Simple error: %v\n", err1)

	// 2. Error with code
	fmt.Println("\n2. Error with Code:")
	err2 := pkgerrors.New("database connection failed", 1001)
	fmt.Printf("Error with code: %v (code=%d)\n", err2, err2.Code)

	// 3. Error with context
	fmt.Println("\n3. Error with Context:")
	err3 := pkgerrors.New("validation failed").
		WithContext("field", "email").
		WithContext("value", "invalid@example.com")
	fmt.Printf("Error with context: %v\n", err3)

	// 4. Flexible E() function
	fmt.Println("\n4. E() Function:")
	err4 := pkgerrors.E("validation failed", "field", "email", "code", 400)
	fmt.Printf("E() created: %v\n", err4)

	// 5. Wrap errors
	fmt.Println("\n5. Error Wrapping:")
	original := pkgerrors.New("original error")
	wrapped := pkgerrors.Wrap(original, "operation failed")
	fmt.Printf("Wrapped error: %v\n", wrapped)

	// 6. Wrap standard errors
	fmt.Println("\n6. Wrap Standard Errors:")
	wrappedStd := pkgerrors.Wrap(sql.ErrNoRows, "query failed")
	fmt.Printf("Wrapped std error: %v\n", wrappedStd)

	// 7. Error chain inspection
	fmt.Println("\n7. Error Chain Inspection:")
	if errors.Is(wrappedStd, sql.ErrNoRows) {
		fmt.Println("✓ Error chain contains sql.ErrNoRows")
	}

	fmt.Println("\nAll basic examples completed.")
}
