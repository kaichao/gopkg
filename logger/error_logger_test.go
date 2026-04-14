package logger_test

import (
	"bytes"
	"fmt"
	"os"
	"strings"
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

func TestLogTracedErrorMixedChain(t *testing.T) {
	entry, buf := logger.NewTestEntry()
	entry.Logger.SetLevel(logrus.DebugLevel) // Enable debug logging for chain

	// Create mixed error chain with standard error at root
	stdErr := fmt.Errorf("standard library error")
	wrapped := errors.Wrap(stdErr, "wrapped error")

	// Log the error chain
	logger.LogTracedError(wrapped, entry)

	output := buf.String()

	// Check that both errors are logged
	if !contains(output, "wrapped error") {
		t.Error("Outer error should be logged")
	}
	if !contains(output, "standard library error") {
		t.Error("Standard error at root should be logged (in debug)")
	}
	if !contains(output, "Caused by") {
		t.Error("Cause chain should be indicated")
	}

	// Count log lines (should be 2: error + debug)
	lines := strings.Count(output, "\n")
	if lines != 2 {
		t.Errorf("Expected 2 log lines, got %d", lines)
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

func TestLogErrorAutoMode(t *testing.T) {
	// Test auto mode behavior (no environment variable set)

	// Test 1: Debug level should use detailed logging
	entry, buf := logger.NewTestEntry()
	entry.Logger.SetLevel(logrus.DebugLevel)
	err := errors.New("debug error")
	logger.LogError(err, entry, logrus.DebugLevel)
	output := buf.String()
	if !contains(output, "debug error") {
		t.Error("Debug error should be logged")
	}
	if !contains(output, "location") {
		t.Error("Debug level should use detailed logging with location")
	}

	// Test 2: Info level should use simple logging
	entry2, buf2 := logger.NewJSONTestEntry()
	err2 := errors.New("info error")
	logger.LogError(err2, entry2, logrus.InfoLevel)
	output2 := buf2.String()
	if !contains(output2, "info error") {
		t.Error("Info error should be logged")
	}
}

func TestLogErrorVerboseOverride(t *testing.T) {
	// Set environment variable to force verbose logging
	os.Setenv("LOG_ERROR_VERBOSE", "true")
	defer os.Unsetenv("LOG_ERROR_VERBOSE")

	entry, buf := logger.NewTestEntry()
	err := errors.New("error with verbose override")
	logger.LogError(err, entry, logrus.InfoLevel) // Info level but should use detailed

	output := buf.String()
	if !contains(output, "location") {
		t.Error("Verbose override should force detailed logging")
	}
}

func TestLogErrorSimpleOverride(t *testing.T) {
	// Set environment variable to force simple logging
	os.Setenv("LOG_ERROR_VERBOSE", "false")
	defer os.Unsetenv("LOG_ERROR_VERBOSE")

	entry, buf := logger.NewTestEntry() // Use text formatter for easier testing
	err := errors.New("error with simple override")
	logger.LogError(err, entry, logrus.DebugLevel) // Debug level but should use simple

	output := buf.String()
	if !contains(output, "error with simple override") {
		t.Error("Simple override should still log the error")
	}
}

func TestLogErrorInvalidEnvValue(t *testing.T) {
	// Set invalid environment variable value (should be ignored)
	os.Setenv("LOG_ERROR_VERBOSE", "invalid_value")
	defer os.Unsetenv("LOG_ERROR_VERBOSE")

	// Should fall back to auto mode
	entry, buf := logger.NewTestEntry()
	entry.Logger.SetLevel(logrus.DebugLevel)
	err := errors.New("error with invalid env")
	logger.LogError(err, entry, logrus.DebugLevel) // Debug → detailed

	output := buf.String()
	if !contains(output, "error with invalid env") {
		t.Error("Invalid env value should still log the error")
	}
	// In auto mode with Debug level, should use detailed logging
	if !strings.Contains(output, "error_level") {
		t.Error("Invalid env value should fall back to auto mode (detailed for debug)")
	}
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
