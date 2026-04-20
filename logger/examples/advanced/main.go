// Advanced logging examples
package main

import (
	"fmt"

	"github.com/kaichao/gopkg/errors"
	"github.com/kaichao/gopkg/logger"
	"github.com/sirupsen/logrus"
)

func main() {
	fmt.Println("=== Advanced Logging Examples ===")

	// 1. Error chain logging
	fmt.Println("\n1. Error Chain:")
	log1 := logrus.New()
	log1.SetLevel(logrus.DebugLevel)
	entry1 := logrus.NewEntry(log1)

	rootErr := errors.New("database error").WithContext("query", "SELECT * FROM users")
	wrappedErr := errors.Wrap(rootErr, "query failed")
	logger.LogTracedError(wrappedErr, entry1)

	// 2. Mixed error chain (standard + TracedError)
	fmt.Println("\n2. Mixed Error Chain:")
	stdErr := fmt.Errorf("standard error: file not found")
	mixedErr := errors.Wrap(stdErr, "operation failed").WithContext("file", "data.txt")
	logger.LogTracedError(mixedErr, entry1)

	// 3. Development vs Production logging
	fmt.Println("\n3. Development vs Production:")
	log2 := logrus.New()
	entry2 := logrus.NewEntry(log2)

	devErr := errors.New("auth failed").
		WithContext("username", "admin").
		WithContext("password", "secret123")

	// Development: detailed logging
	log2.SetLevel(logrus.DebugLevel)
	fmt.Println("Development (debug level):")
	logger.LogTracedError(devErr, entry2)

	// Production: filtered logging
	log2.SetLevel(logrus.InfoLevel)
	fmt.Println("Production (info level):")
	logger.SimpleLog(devErr, entry2)

	// 4. LogError with automatic decision
	fmt.Println("\n4. LogError Auto Decision:")
	log3 := logrus.New()
	entry3 := logrus.NewEntry(log3)

	autoErr := errors.New("processing error")

	log3.SetLevel(logrus.DebugLevel)
	fmt.Println("Debug level -> detailed:")
	logger.LogError(autoErr, entry3)

	log3.SetLevel(logrus.InfoLevel)
	fmt.Println("Info level -> simple:")
	logger.LogError(autoErr, entry3)

	// 5. Environment variable override
	fmt.Println("\n5. Environment Override (simulated):")
	// In real usage, set LOG_ERROR_VERBOSE=true/false
	err := errors.New("error with context").WithContext("data", "sensitive")
	logger.LogError(err, entry3) // Uses auto decision based on level

	fmt.Println("\nAll advanced examples completed.")
}
