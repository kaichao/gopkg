package logger

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestNewLogger(t *testing.T) {
	cfg := &Config{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	}

	logger, err := NewLogger(cfg)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	if logger == nil {
		t.Fatal("Logger is nil")
	}

	if logger.GetLevel() != "info" {
		t.Errorf("Expected level 'info', got '%s'", logger.GetLevel())
	}
}

func TestLoggerWithFields(t *testing.T) {
	cfg := &Config{
		Level:  "debug",
		Format: "json",
		Output: "stdout",
	}

	logger, err := NewLogger(cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Test WithField
	logger1 := logger.WithField("key", "value")
	if logger1 == nil {
		t.Fatal("WithField returned nil")
	}

	// Test WithFields
	logger2 := logger.WithFields(logrus.Fields{
		"field1": "value1",
		"field2": "value2",
	})
	if logger2 == nil {
		t.Fatal("WithFields returned nil")
	}
}

func TestLoggerSetLevel(t *testing.T) {
	cfg := &Config{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	}

	logger, err := NewLogger(cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Test setting level
	if err := logger.SetLevel("debug"); err != nil {
		t.Errorf("Failed to set level: %v", err)
	}

	if logger.GetLevel() != "debug" {
		t.Errorf("Expected level 'debug', got '%s'", logger.GetLevel())
	}

	// Test invalid level
	if err := logger.SetLevel("invalid"); err == nil {
		t.Error("Expected error for invalid level")
	}
}

func TestConfigLoadFromEnv(t *testing.T) {
	// Set environment variables
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("LOG_FORMAT", "text")
	defer func() {
		os.Unsetenv("LOG_LEVEL")
		os.Unsetenv("LOG_FORMAT")
	}()

	cfg := LoadConfig()
	if cfg.Level != "debug" {
		t.Errorf("Expected level 'debug', got '%s'", cfg.Level)
	}

	if cfg.Format != "text" {
		t.Errorf("Expected format 'text', got '%s'", cfg.Format)
	}
}

func TestConfigValidate(t *testing.T) {
	// Test valid configuration
	cfg := &Config{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("Valid config should not return error: %v", err)
	}

	// Test invalid level
	cfg.Level = "invalid"
	if err := cfg.Validate(); err == nil {
		t.Error("Invalid level should return error")
	}

	// Test invalid format
	cfg.Level = "info"
	cfg.Format = "invalid"
	if err := cfg.Validate(); err == nil {
		t.Error("Invalid format should return error")
	}
}

func TestConfigToJSON(t *testing.T) {
	cfg := &Config{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	}

	jsonStr := cfg.ToJSON()
	if jsonStr == "" {
		t.Error("ToJSON returned empty string")
	}

	// Validate JSON format
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		t.Errorf("Invalid JSON: %v", err)
	}
}

func TestGlobalLogger(t *testing.T) {
	// Test getting global logger instance
	logger := Global()
	if logger == nil {
		t.Fatal("Global logger is nil")
	}

	// Test initializing global logger
	cfg := &Config{
		Level:  "debug",
		Format: "json",
		Output: "stdout",
	}

	if err := InitGlobal(cfg); err != nil {
		t.Errorf("Failed to init global logger: %v", err)
	}

	// Verify global logger was updated
	newLogger := Global()
	if newLogger.GetLevel() != "debug" {
		t.Errorf("Expected global logger level 'debug', got '%s'", newLogger.GetLevel())
	}
}

func TestLoggerCopy(t *testing.T) {
	cfg := &Config{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	}

	logger, err := NewLogger(cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Add field
	logger1 := logger.WithField("key", "value")

	// Copy
	logger2 := logger1.Copy()
	if logger2 == nil {
		t.Fatal("Copy returned nil")
	}
}

func TestLoggerIsLevelEnabled(t *testing.T) {
	cfg := &Config{
		Level:  "warn",
		Format: "json",
		Output: "stdout",
	}

	logger, err := NewLogger(cfg)
	if err != nil {
		t.Fatal(err)
	}

	// WARN level should enable WARN, ERROR, FATAL
	if !logger.IsLevelEnabled(logrus.WarnLevel) {
		t.Error("WARN level should be enabled")
	}
	if !logger.IsLevelEnabled(logrus.ErrorLevel) {
		t.Error("ERROR level should be enabled")
	}
	if !logger.IsLevelEnabled(logrus.FatalLevel) {
		t.Error("FATAL level should be enabled")
	}

	// But DEBUG, INFO should be disabled
	if logger.IsLevelEnabled(logrus.DebugLevel) {
		t.Error("DEBUG level should be disabled")
	}
	if logger.IsLevelEnabled(logrus.InfoLevel) {
		t.Error("INFO level should be disabled")
	}
}

func TestRotatedWriter(t *testing.T) {
	// Create temporary directory
	dir, err := os.MkdirTemp("", "logtest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	filePath := filepath.Join(dir, "test.log")

	// Create rotated writer
	writer, err := NewRotatedWriter(filePath, 1, 7, 3)
	if err != nil {
		t.Fatalf("Failed to create rotated writer: %v", err)
	}
	defer writer.Close()

	// Write data
	data := []byte("test log entry\n")
	if n, err := writer.Write(data); err != nil {
		t.Errorf("Write failed: %v", err)
	} else if n != len(data) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(data), n)
	}

	// Sync
	if err := writer.Sync(); err != nil {
		t.Errorf("Sync failed: %v", err)
	}
}

func TestAsyncWriter(t *testing.T) {
	var buf bytes.Buffer

	// Create async writer
	asyncWriter := NewAsyncWriter(&buf, 100, 10)
	asyncWriter.SetFormatter(&logrus.JSONFormatter{DisableTimestamp: true})

	// Start async writing
	asyncWriter.Start()

	// Create test log entry
	entry := &logrus.Entry{
		Message: "test message",
		Level:   logrus.InfoLevel,
	}

	// Write log
	if err := asyncWriter.WriteEntry(entry); err != nil {
		t.Errorf("WriteEntry failed: %v", err)
	}

	// Stop async writer, which waits for all logs to complete
	asyncWriter.Stop()

	// Verify log content
	if buf.Len() == 0 {
		t.Error("No log written to buffer")
	}
}
