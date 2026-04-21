package logger

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/kaichao/gopkg/errors"

	"github.com/sirupsen/logrus"
)

// LogTracedError logs traced errors with full context
// level specifies the log level for the outermost error (inner errors are logged as Debug)
func LogTracedError(err error, log *logrus.Entry, level ...logrus.Level) {
	if err == nil {
		return
	}

	logLevel := logrus.ErrorLevel
	if len(level) > 0 {
		logLevel = level[0]
	}

	// Collect full error chain using Unwrap()
	chain := collectErrorChain(err)

	for i, errInChain := range chain {
		// Check if it's a TracedError
		if tracedErr, ok := errInChain.(*errors.TracedError); ok {
			fields := logrus.Fields{
				"error_level": i, // 0 = outermost error
				"error_type":  fmt.Sprintf("%T", errInChain),
				"location":    tracedErr.Location,
				"timestamp":   tracedErr.Timestamp,
			}

			// Add error code if available (only if not default -1)
			if tracedErr.Code != -1 {
				fields["error_code"] = tracedErr.Code
			}

			// Add all context fields
			for k, v := range tracedErr.Context {
				fields[fmt.Sprintf("ctx_%s", k)] = v
			}

			// Prepare message preserving original formatting
			msg := tracedErr.Message
			// For non-zero levels (inner errors), add "Caused by:" prefix
			if i > 0 {
				log.WithFields(fields).Debug(fmt.Sprintf("Caused by: %s", msg))
			} else {
				log.WithFields(fields).Log(logLevel, msg)
			}
		} else {
			// Regular error in the chain
			fields := logrus.Fields{
				"error_level": i,
				"error_type":  fmt.Sprintf("%T", errInChain),
			}

			if i == 0 {
				log.WithFields(fields).Log(logLevel, errInChain.Error())
			} else {
				log.WithFields(fields).Debug(fmt.Sprintf("Caused by: %s", errInChain.Error()))
			}
		}
	}
}

// collectErrorChain collects the full error chain using Unwrap()
func collectErrorChain(err error) []error {
	var chain []error

	current := err
	for current != nil {
		chain = append(chain, current)

		// Try to unwrap the error
		if unwrapper, ok := current.(interface{ Unwrap() error }); ok {
			current = unwrapper.Unwrap()
		} else {
			// No more errors in the chain
			break
		}
	}

	return chain
}

// SimpleLog is a simplified version for production
// level specifies the log level (defaults to ErrorLevel)
func SimpleLog(err error, log *logrus.Entry, level ...logrus.Level) {
	if err == nil {
		return
	}

	logLevel := logrus.ErrorLevel
	if len(level) > 0 {
		logLevel = level[0]
	}

	if tracedErr, ok := err.(*errors.TracedError); ok {
		fields := logrus.Fields{
			"location": tracedErr.Location,
		}

		// Add error code if available (only if not default -1)
		if tracedErr.Code != -1 {
			fields["error_code"] = tracedErr.Code
		}

		// Add non-sensitive context as individual fields
		for k, v := range tracedErr.Context {
			if !IsSensitiveKey(k) {
				fields[k] = v
			}
		}

		log.WithFields(fields).Log(logLevel, tracedErr.Message)
	} else {
		log.WithError(err).Log(logLevel, "Operation failed")
	}
}

// IsSensitiveKey ...
func IsSensitiveKey(key string) bool {
	key = strings.ToLower(key)
	sensitive := []string{"password", "token", "secret", "credit"}

	// Check for exact matches or common patterns
	for _, s := range sensitive {
		if key == s || strings.Contains(key, "_"+s) || strings.Contains(key, s+"_") {
			return true
		}
	}

	// Special case for "key" - only match if it's part of common sensitive patterns
	if key == "key" || key == "api_key" || key == "secret_key" || key == "private_key" {
		return true
	}

	return false
}

// NewTestEntry creates a logrus.Entry for testing purposes
// It captures output in a buffer and disables timestamps for consistent test output
func NewTestEntry() (*logrus.Entry, *bytes.Buffer) {
	var buf bytes.Buffer
	log := logrus.New()
	log.SetOutput(&buf)
	log.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})
	return logrus.NewEntry(log), &buf
}

// NewJSONTestEntry creates a logrus.Entry with JSON formatter for testing
func NewJSONTestEntry() (*logrus.Entry, *bytes.Buffer) {
	var buf bytes.Buffer
	log := logrus.New()
	log.SetOutput(&buf)
	log.SetFormatter(&logrus.JSONFormatter{
		DisableTimestamp: true,
	})
	return logrus.NewEntry(log), &buf
}

// SetDefaultLogger sets the package-level default logger
// This should be called early in the application initialization
// DEPRECATED: Use logger.InitGlobal() instead
func SetDefaultLogger(logrusLogger *logrus.Logger) {
	if logrusLogger == nil {
		return
	}

	// Create a new Logger that wraps the provided logrus.Logger
	wrappedLogger := &Logger{
		Logger: logrusLogger,
		config: &Config{
			Level:  logrusLogger.GetLevel().String(),
			Format: "text", // Assume text for backward compatibility
			Output: "stdout",
		},
		entry:  logrus.NewEntry(logrusLogger),
		fields: make(logrus.Fields),
	}

	// Set as global logger
	defaultLogger = wrappedLogger
}

// getDefaultEntry returns a logrus.Entry using the global logger
func getDefaultEntry() *logrus.Entry {
	// Use the global logger from logger package
	logger := Global()
	if logger != nil {
		return logger.NewEntry()
	}

	// Fallback to a basic logger if global logger is not available
	fallbackLogger := logrus.New()
	fallbackLogger.SetOutput(os.Stderr)
	fallbackLogger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	return logrus.NewEntry(fallbackLogger)
}

// LogTracedErrorDefault logs traced errors using the global logger
// level specifies the log level for the outermost error (inner errors are logged as Debug)
func LogTracedErrorDefault(err error, level ...logrus.Level) {
	LogTracedError(err, getDefaultEntry(), level...)
}

// SimpleLogDefault logs errors using the global logger with sensitive data filtering
// level specifies the log level (defaults to ErrorLevel)
func SimpleLogDefault(err error, level ...logrus.Level) {
	SimpleLog(err, getDefaultEntry(), level...)
}

// LogError automatically decides between LogTracedError and SimpleLog based on log level
// DEBUG and TRACE levels use LogTracedError (detailed output)
// INFO, WARN, ERROR levels use SimpleLog (concise output)
//
// Environment variable LOG_ERROR_VERBOSE can override this behavior:
//
//	LOG_ERROR_VERBOSE=true   - forces detailed logging for all levels
//	LOG_ERROR_VERBOSE=false  - forces simple logging for all levels
//	LOG_ERROR_VERBOSE not set - auto mode (default)
func LogError(err error, log *logrus.Entry, level ...logrus.Level) {
	if err == nil {
		return
	}

	logLevel := logrus.ErrorLevel
	if len(level) > 0 {
		logLevel = level[0]
	}

	// Default behavior: detailed for DEBUG/TRACE, simple for INFO/WARN/ERROR
	useDetailed := logLevel <= logrus.DebugLevel

	// Check environment variable override
	envValue := os.Getenv("LOG_ERROR_VERBOSE")
	if envValue != "" {
		// Only override if explicitly set
		if envValue == "true" {
			useDetailed = true
		} else if envValue == "false" {
			useDetailed = false
		}
		// Other values are ignored, keeping auto behavior
	}

	// Execute logging based on decision
	if useDetailed {
		LogTracedError(err, log, logLevel)
	} else {
		SimpleLog(err, log, logLevel)
	}
}
