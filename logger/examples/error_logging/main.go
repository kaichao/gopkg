// Error logging examples
package main

import (
	"fmt"

	"github.com/kaichao/gopkg/errors"
	"github.com/kaichao/gopkg/logger"
	"github.com/sirupsen/logrus"
)

func main() {
	// Setup logger
	log := logrus.New()
	log.SetFormatter(&logrus.TextFormatter{DisableTimestamp: true})
	entry := logrus.NewEntry(log)

	fmt.Println("=== Basic Error Logging ===")

	// LogTracedError - detailed error logging
	err1 := errors.New("file not found").
		WithContext("filename", "data.txt").
		WithContext("user", "john_doe")
	logger.LogTracedError(err1, entry)

	// LogTracedError with custom level
	err2 := errors.New("permission denied", 403).
		WithContext("resource", "/api/data")
	logger.LogTracedError(err2, entry, logrus.WarnLevel)

	// SimpleLog - filtered sensitive data
	err3 := errors.New("authentication failed").
		WithContext("username", "admin").
		WithContext("password", "secret123") // filtered
	logger.SimpleLog(err3, entry)

	// LogError - auto decision (detailed for debug, simple for info+)
	log.SetLevel(logrus.DebugLevel)
	err4 := errors.New("debug error")
	logger.LogError(err4, entry) // detailed (debug level)

	log.SetLevel(logrus.InfoLevel)
	logger.LogError(err4, entry) // simple (info level)

	fmt.Println("=== Error Chain ===")
	// Error chain
	root := errors.New("database error").WithContext("query", "SELECT * FROM users")
	wrapped := errors.Wrap(root, "query failed")
	logger.LogTracedError(wrapped, entry)

	fmt.Println("All examples completed. Check log output above.")
}
