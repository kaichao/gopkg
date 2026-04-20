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

	// 测试 WithField
	logger1 := logger.WithField("key", "value")
	if logger1 == nil {
		t.Fatal("WithField returned nil")
	}

	// 测试 WithFields
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

	// 测试设置级别
	if err := logger.SetLevel("debug"); err != nil {
		t.Errorf("Failed to set level: %v", err)
	}

	if logger.GetLevel() != "debug" {
		t.Errorf("Expected level 'debug', got '%s'", logger.GetLevel())
	}

	// 测试无效级别
	if err := logger.SetLevel("invalid"); err == nil {
		t.Error("Expected error for invalid level")
	}
}

func TestConfigLoadFromEnv(t *testing.T) {
	// 设置环境变量
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
	// 测试有效配置
	cfg := &Config{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("Valid config should not return error: %v", err)
	}

	// 测试无效级别
	cfg.Level = "invalid"
	if err := cfg.Validate(); err == nil {
		t.Error("Invalid level should return error")
	}

	// 测试无效格式
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

	// 验证 JSON 格式
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		t.Errorf("Invalid JSON: %v", err)
	}
}

func TestGlobalLogger(t *testing.T) {
	// 测试获取全局日志实例
	logger := Global()
	if logger == nil {
		t.Fatal("Global logger is nil")
	}

	// 测试初始化全局日志
	cfg := &Config{
		Level:  "debug",
		Format: "json",
		Output: "stdout",
	}

	if err := InitGlobal(cfg); err != nil {
		t.Errorf("Failed to init global logger: %v", err)
	}

	// 验证全局日志已更新
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

	// 添加字段
	logger1 := logger.WithField("key", "value")

	// 复制
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

	// WARN 级别应该启用 WARN、ERROR、FATAL
	if !logger.IsLevelEnabled(logrus.WarnLevel) {
		t.Error("WARN level should be enabled")
	}
	if !logger.IsLevelEnabled(logrus.ErrorLevel) {
		t.Error("ERROR level should be enabled")
	}
	if !logger.IsLevelEnabled(logrus.FatalLevel) {
		t.Error("FATAL level should be enabled")
	}

	// 但 DEBUG、INFO 应该禁用
	if logger.IsLevelEnabled(logrus.DebugLevel) {
		t.Error("DEBUG level should be disabled")
	}
	if logger.IsLevelEnabled(logrus.InfoLevel) {
		t.Error("INFO level should be disabled")
	}
}

func TestRotatedWriter(t *testing.T) {
	// 创建临时目录
	dir, err := os.MkdirTemp("", "logtest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	filePath := filepath.Join(dir, "test.log")

	// 创建轮转写入器
	writer, err := NewRotatedWriter(filePath, 1, 7, 3)
	if err != nil {
		t.Fatalf("Failed to create rotated writer: %v", err)
	}
	defer writer.Close()

	// 写入数据
	data := []byte("test log entry\n")
	if n, err := writer.Write(data); err != nil {
		t.Errorf("Write failed: %v", err)
	} else if n != len(data) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(data), n)
	}

	// 同步
	if err := writer.Sync(); err != nil {
		t.Errorf("Sync failed: %v", err)
	}
}

func TestAsyncWriter(t *testing.T) {
	var buf bytes.Buffer

	// 创建异步写入器
	asyncWriter := NewAsyncWriter(&buf, 100, 10)
	asyncWriter.SetFormatter(&logrus.JSONFormatter{DisableTimestamp: true})

	// 启动异步写入
	asyncWriter.Start()

	// 创建测试日志条目
	entry := &logrus.Entry{
		Message: "test message",
		Level:   logrus.InfoLevel,
	}

	// 写入日志
	if err := asyncWriter.WriteEntry(entry); err != nil {
		t.Errorf("WriteEntry failed: %v", err)
	}

	// 停止异步写入器，这会等待所有日志写入完成
	asyncWriter.Stop()

	// 验证日志内容
	if buf.Len() == 0 {
		t.Error("No log written to buffer")
	}
}
