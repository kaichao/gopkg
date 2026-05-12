package main

import (
	"fmt"
	"log"

	"github.com/kaichao/gopkg/errors"
	"github.com/kaichao/gopkg/exec"
)

func main() {
	// Example 1: Local execution with full output
	fmt.Println("=== Example 1: Local command execution ===")
	stdout, stderr, err := exec.RunReturnAll("ls -l /tmp", 10)
	if err != nil {
		log.Printf("Execution failed: %v\nOutput: %s\nError: %s", err, stdout, stderr)
	} else {
		fmt.Printf("Exit code: %d\n", errors.GetCode(err))
		fmt.Printf("Stdout: %s\n", stdout[:min(100, len(stdout))])
	}

	// Example 2: Get exit code from error
	fmt.Println("\n=== Example 2: Get exit code from error ===")
	stdout, stderr, err = exec.RunReturnAll("echo 'Hello World'", 5)
	if err != nil {
		log.Printf("Error: %v (code=%d)", err, errors.GetCode(err))
	} else {
		fmt.Printf("Exit code: %d\n", errors.GetCode(err))
		fmt.Printf("Stdout: %s\nStderr: %s\n", stdout, stderr)
	}

	// Example 3: Get stdout only
	fmt.Println("\n=== Example 3: Get stdout only ===")
	stdout, _, err = exec.RunReturnAll("date", 5)
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("Current date: %s\n", stdout)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
