// Advanced error handling examples
package main

import (
	"errors"
	"fmt"

	pkgerrors "github.com/kaichao/gopkg/errors"
)

func main() {
	fmt.Println("=== Advanced Error Handling ===")

	// 1. Error chain with mixed types
	fmt.Println("\n1. Mixed Error Chain:")
	stdErr := errors.New("standard library error")
	mixedChain := pkgerrors.Wrap(stdErr, "operation failed").
		WithContext("file", "data.txt").
		WithContext("user", "admin")
	fmt.Printf("Mixed chain: %v\n", mixedChain)

	// Check if it contains the std error
	if errors.Is(mixedChain, stdErr) {
		fmt.Println("✓ Contains original std error")
	}

	// 2. errors.As usage
	fmt.Println("\n2. errors.As Usage:")
	deepChain := pkgerrors.Wrap(
		pkgerrors.Wrap(
			pkgerrors.New("root error", 1000),
			"mid-level",
		),
		"top-level",
	)

	var tracedErr *pkgerrors.TracedError
	if errors.As(deepChain, &tracedErr) {
		fmt.Printf("✓ Converted to TracedError: %v (code=%d)\n", tracedErr.Message, tracedErr.Code)
	}

	// 3. Manual chain traversal
	fmt.Println("\n3. Chain Traversal:")
	fmt.Println("Full chain:")
	var current error = deepChain
	for i := 1; current != nil; i++ {
		fmt.Printf("  %d. %v\n", i, current)
		if unwrapped := errors.Unwrap(current); unwrapped != nil {
			current = unwrapped
		} else {
			break
		}
	}

	// 4. GetCode utility
	fmt.Println("\n4. Error Code Utilities:")
	noCodeErr := pkgerrors.New("no code")
	withCodeErr := pkgerrors.New("with code", 404)

	fmt.Printf("No code error: code=%d\n", pkgerrors.GetCode(noCodeErr))
	fmt.Printf("With code error: code=%d\n", pkgerrors.GetCode(withCodeErr))

	// 5. Format output
	fmt.Println("\n5. Formatted Error Output:")
	detailedErr := pkgerrors.New("detailed error", 500).
		WithContext("request_id", "abc-123").
		WithContext("user_id", 456).
		WithContext("path", "/api/users")
	fmt.Println(detailedErr.Format())

	// 6. Full chain retrieval
	fmt.Println("\n6. Full Chain Retrieval:")
	chain := pkgerrors.New("root", 1)
	chain = pkgerrors.Wrap(chain, "level2")
	chain = pkgerrors.Wrap(chain, "level3")

	fullChain := chain.GetFullChain()
	fmt.Printf("Chain has %d levels:\n", len(fullChain))
	for i, err := range fullChain {
		fmt.Printf("  %d. %s\n", i+1, err.Message)
	}

	fmt.Println("\nAll advanced examples completed.")
}
