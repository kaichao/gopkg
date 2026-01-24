package logger

import (
	"bytes"
	"fmt"
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

	// If it's a traced error, log with full details
	if tracedErr, ok := err.(*errors.TracedError); ok {
		chain := tracedErr.GetFullChain()

		for i, errInChain := range chain {
			fields := logrus.Fields{
				"error_level": i, // 0 = outermost error
				"location":    errInChain.Location,
				"timestamp":   errInChain.Timestamp,
			}

			// Add error code if available (only if not default -1)
			if errInChain.Code != -1 {
				fields["error_code"] = errInChain.Code
			}

			// Add all context fields
			for k, v := range errInChain.Context {
				fields[fmt.Sprintf("ctx_%s", k)] = v
			}

			// Log at appropriate level
			if i == 0 {
				log.WithFields(fields).Log(logLevel, errInChain.Message)
			} else {
				log.WithFields(fields).Debug(fmt.Sprintf("Caused by: %s", errInChain.Message))
			}
		}
	} else {
		// Regular error
		log.WithError(err).Log(logLevel, "Operation failed")
	}
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
