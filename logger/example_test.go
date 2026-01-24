package logger_test

import (
	"bytes"
	"fmt"

	"github.com/kaichao/gopkg/errors"
	"github.com/kaichao/gopkg/logger"
	"github.com/sirupsen/logrus"
)

func ExampleLogTracedError() {
	// Setup logger
	var buf bytes.Buffer
	log := logrus.New()
	log.SetOutput(&buf)
	log.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})

	entry := logrus.NewEntry(log)

	// Create a traced error
	err := errors.New("file not found").
		WithContext("filename", "data.txt").
		WithContext("user", "john_doe")

	// Log the error
	logger.LogTracedError(err, entry)

	fmt.Println("Error logged with full context")
	// Check buf.String() for the actual log output
}

func ExampleLogTracedError_withLevel() {
	var buf bytes.Buffer
	log := logrus.New()
	log.SetOutput(&buf)
	log.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})

	entry := logrus.NewEntry(log)

	err := errors.New("permission denied", 403).
		WithContext("resource", "/api/data").
		WithContext("user_id", 12345)

	// Log with Warn level instead of default Error level
	logger.LogTracedError(err, entry, logrus.WarnLevel)

	fmt.Println("Error logged at Warn level")
}

func ExampleSimpleLog() {
	var buf bytes.Buffer
	log := logrus.New()
	log.SetOutput(&buf)
	log.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})

	entry := logrus.NewEntry(log)

	// Create error with sensitive data
	err := errors.New("authentication failed").
		WithContext("username", "john_doe").
		WithContext("password", "secret123"). // This will be filtered
		WithContext("attempt", 3)

	// SimpleLog filters sensitive data
	logger.SimpleLog(err, entry)

	fmt.Println("Error logged with sensitive data filtered")
}

func ExampleSimpleLog_withLevel() {
	var buf bytes.Buffer
	log := logrus.New()
	log.SetOutput(&buf)
	log.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})

	entry := logrus.NewEntry(log)

	err := errors.New("validation error", 400).
		WithContext("field", "email").
		WithContext("value", "invalid@example")

	// Log at Info level
	logger.SimpleLog(err, entry, logrus.InfoLevel)

	fmt.Println("Validation error logged at Info level")
}

func ExampleLogTracedError_errorChain() {
	var buf bytes.Buffer
	log := logrus.New()
	log.SetOutput(&buf)
	log.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})
	log.SetLevel(logrus.DebugLevel) // Enable debug to see full chain

	entry := logrus.NewEntry(log)

	// Create error chain
	rootErr := errors.New("database connection failed").
		WithContext("host", "db.example.com").
		WithContext("port", 5432)

	middleErr := errors.Wrap(rootErr, "query execution failed").
		WithContext("query", "SELECT * FROM users")

	topErr := errors.Wrap(middleErr, "user data fetch failed").
		WithContext("user_id", 12345)

	// LogTracedError will show the full chain
	logger.LogTracedError(topErr, entry)

	fmt.Println("Full error chain logged")
}

func ExampleLogTracedError_productionVsDevelopment() {
	// In development: use LogTracedError for detailed debugging
	var devBuf bytes.Buffer
	devLog := logrus.New()
	devLog.SetOutput(&devBuf)
	devLog.SetFormatter(&logrus.TextFormatter{DisableTimestamp: true})
	devLog.SetLevel(logrus.DebugLevel)

	devEntry := logrus.NewEntry(devLog)

	// In production: use SimpleLog for security and brevity
	var prodBuf bytes.Buffer
	prodLog := logrus.New()
	prodLog.SetOutput(&prodBuf)
	prodLog.SetFormatter(&logrus.JSONFormatter{})
	prodLog.SetLevel(logrus.WarnLevel)

	prodEntry := logrus.NewEntry(prodLog)

	// Same error
	err := errors.New("api request failed").
		WithContext("endpoint", "/api/users").
		WithContext("api_key", "sk_live_123456"). // Sensitive!
		WithContext("status_code", 500)

	// Development logging (detailed)
	logger.LogTracedError(err, devEntry, logrus.ErrorLevel)
	fmt.Println("Development log (detailed):", len(devBuf.String()), "bytes")

	// Production logging (filtered, secure)
	logger.SimpleLog(err, prodEntry, logrus.ErrorLevel)
	fmt.Println("Production log (filtered):", len(prodBuf.String()), "bytes")

	// In production, sensitive data like api_key is filtered out
}
