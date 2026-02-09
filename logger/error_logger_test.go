package logger_test

import (
	"bytes"
	"testing"

	"github.com/kaichao/gopkg/errors"
	"github.com/kaichao/gopkg/logger"
	"github.com/sirupsen/logrus"
)

func TestLogTracedError(t *testing.T) {
	entry, buf := logger.NewTestEntry()

	// Create a traced error with context
	err := errors.New("test error").
		WithContext("key1", "value1").
		WithContext("key2", 123)

	// Log the error
	logger.LogTracedError(err, entry)

	output := buf.String()

	// Check that key parts are logged
	if !contains(output, "test error") {
		t.Error("Error message should be logged")
	}
	if !contains(output, "key1") || !contains(output, "value1") {
		t.Error("Context should be logged")
	}
	if !contains(output, "error_level") {
		t.Error("Error level should be logged")
	}
}

func TestLogTracedErrorWithCode(t *testing.T) {
	entry, buf := logger.NewTestEntry()

	// Create a traced error with code
	err := errors.New("test error", 1001).
		WithContext("key", "value")

	// Log the error
	logger.LogTracedError(err, entry)

	output := buf.String()

	// Check that error code is logged (as number)
	if !contains(output, "1001") {
		t.Error("Error code should be logged")
	}
	if !contains(output, "error_code") {
		t.Error("error_code field should be present")
	}
}

func TestLogTracedErrorChain(t *testing.T) {
	entry, buf := logger.NewTestEntry()
	entry.Logger.SetLevel(logrus.DebugLevel) // Enable debug logging for chain

	// Create error chain
	original := errors.New("original error")
	wrapped := errors.Wrap(original, "wrapped error")

	// Log the error chain
	logger.LogTracedError(wrapped, entry)

	output := buf.String()

	// Check that both errors are logged
	if !contains(output, "wrapped error") {
		t.Error("Outer error should be logged")
	}
	if !contains(output, "original error") {
		t.Error("Inner error should be logged (in debug)")
	}
	if !contains(output, "Caused by") {
		t.Error("Cause chain should be indicated")
	}
}

func TestSimpleLog(t *testing.T) {
	entry, buf := logger.NewJSONTestEntry()

	// Create a traced error with context
	err := errors.New("test error").
		WithContext("key1", "value1").
		WithContext("password", "secret123") // Sensitive data

	// Log the error
	logger.SimpleLog(err, entry)

	output := buf.String()

	// Check that non-sensitive context is logged
	if !contains(output, "key1") || !contains(output, "value1") {
		t.Error("Non-sensitive context should be logged")
	}

	// Check that sensitive context is filtered
	if contains(output, "secret123") {
		t.Error("Sensitive data should be filtered")
	}
	if contains(output, "password") {
		t.Error("Sensitive key names should be filtered")
	}
}

func TestSimpleLogWithCode(t *testing.T) {
	entry, buf := logger.NewTestEntry()

	// Create a traced error with code
	err := errors.New("test error", 1001)

	// Log the error
	logger.SimpleLog(err, entry)

	output := buf.String()

	// Check that error code is logged (as number)
	if !contains(output, "1001") {
		t.Error("Error code should be logged")
	}
}

func TestLogLevelParameter(t *testing.T) {
	entry, buf := logger.NewTestEntry()
	entry.Logger.SetLevel(logrus.WarnLevel) // Only warn and above

	err := errors.New("test error")

	// Log with Warn level
	logger.LogTracedError(err, entry, logrus.WarnLevel)

	output := buf.String()

	if !contains(output, "test error") {
		t.Error("Error should be logged at Warn level")
	}

	// Clear buffer
	buf.Reset()

	// Log with Debug level (should not appear due to logger level)
	logger.LogTracedError(err, entry, logrus.DebugLevel)

	output = buf.String()
	if output != "" {
		t.Error("Debug level error should not be logged when logger level is Warn")
	}
}

func TestIsSensitiveKey(t *testing.T) {
	tests := []struct {
		key      string
		expected bool
	}{
		{"password", true},
		{"PASSWORD", true},
		{"user_password", true},
		{"password_hash", true},
		{"token", true},
		{"api_token", true},
		{"secret", true},
		{"secret_key", true},
		{"credit", true},
		{"credit_card", true},
		{"key", true},
		{"api_key", true},
		{"secret_key", true},
		{"private_key", true},
		{"username", false},
		{"email", false},
		{"name", false},
		{"age", false},
		{"key1", false},         // Should not match because it's just "key1"
		{"key_name", false},     // Should not match because it's not a sensitive pattern
		{"customer_key", false}, // Should not match
	}

	for _, test := range tests {
		result := logger.IsSensitiveKey(test.key)
		if result != test.expected {
			t.Errorf("isSensitiveKey(%q) = %v, expected %v", test.key, result, test.expected)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || contains(s[1:], substr)))
}

func TestLogTracedErrorDefault(t *testing.T) {
	// Setup a test logger to capture output
	var buf bytes.Buffer
	testLogger := logrus.New()
	testLogger.SetOutput(&buf)
	testLogger.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})

	// Set as default logger
	logger.SetDefaultLogger(testLogger)

	// Create a traced error with context
	err := errors.New("test error default").
		WithContext("key1", "value1").
		WithContext("key2", 123)

	// Log the error using default logger
	logger.LogTracedErrorDefault(err)

	output := buf.String()

	// Check that key parts are logged
	if !contains(output, "test error default") {
		t.Error("Error message should be logged")
	}
	if !contains(output, "key1") || !contains(output, "value1") {
		t.Error("Context should be logged")
	}
	if !contains(output, "error_level") {
		t.Error("Error level should be logged")
	}
}

func TestSimpleLogDefault(t *testing.T) {
	// Setup a test logger to capture output
	var buf bytes.Buffer
	testLogger := logrus.New()
	testLogger.SetOutput(&buf)
	testLogger.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})

	// Set as default logger
	logger.SetDefaultLogger(testLogger)

	// Create a traced error with context
	err := errors.New("test error default").
		WithContext("key1", "value1").
		WithContext("password", "secret123") // Sensitive data

	// Log the error using default logger
	logger.SimpleLogDefault(err)

	output := buf.String()

	// Check that non-sensitive context is logged
	if !contains(output, "key1") || !contains(output, "value1") {
		t.Error("Non-sensitive context should be logged")
	}

	// Check that sensitive context is filtered
	if contains(output, "secret123") {
		t.Error("Sensitive data should be filtered")
	}
	if contains(output, "password") {
		t.Error("Sensitive key names should be filtered")
	}
}

func TestLogTracedErrorDefaultWithLevel(t *testing.T) {
	// Setup a test logger to capture output
	var buf bytes.Buffer
	testLogger := logrus.New()
	testLogger.SetOutput(&buf)
	testLogger.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})
	testLogger.SetLevel(logrus.WarnLevel) // Only warn and above

	// Set as default logger
	logger.SetDefaultLogger(testLogger)

	err := errors.New("test error")

	// Log with Warn level
	logger.LogTracedErrorDefault(err, logrus.WarnLevel)

	output := buf.String()

	if !contains(output, "test error") {
		t.Error("Error should be logged at Warn level")
	}

	// Clear buffer
	buf.Reset()

	// Log with Debug level (should not appear due to logger level)
	logger.LogTracedErrorDefault(err, logrus.DebugLevel)

	output = buf.String()
	if output != "" {
		t.Error("Debug level error should not be logged when logger level is Warn")
	}
}

func TestSimpleLogDefaultWithLevel(t *testing.T) {
	// Setup a test logger to capture output
	var buf bytes.Buffer
	testLogger := logrus.New()
	testLogger.SetOutput(&buf)
	testLogger.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})
	testLogger.SetLevel(logrus.WarnLevel) // Only warn and above

	// Set as default logger
	logger.SetDefaultLogger(testLogger)

	err := errors.New("test error")

	// Log with Warn level
	logger.SimpleLogDefault(err, logrus.WarnLevel)

	output := buf.String()

	if !contains(output, "test error") {
		t.Error("Error should be logged at Warn level")
	}

	// Clear buffer
	buf.Reset()

	// Log with Debug level (should not appear due to logger level)
	logger.SimpleLogDefault(err, logrus.DebugLevel)

	output = buf.String()
	if output != "" {
		t.Error("Debug level error should not be logged when logger level is Warn")
	}
}
