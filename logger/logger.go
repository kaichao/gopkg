package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/kaichao/gopkg/errors"
	"github.com/sirupsen/logrus"
)

// Logger 封装 logrus.Logger 的统一日志接口
type Logger struct {
	*logrus.Logger
	mu            sync.Mutex
	config        *Config
	entry         *logrus.Entry
	fields        logrus.Fields
	asyncWriter   *AsyncWriter
	rotatedWriter *RotatedWriter
}

// GlobalLogger 全局日志实例
var (
	defaultLogger *Logger
	once          sync.Once
)

// NewLogger 创建新的日志实例
func NewLogger(cfg *Config) (*Logger, error) {
	if err := cfg.Validate(); err != nil {
		return nil, errors.WrapE(err, "validate logger config")
	}

	logrusLogger := logrus.New()
	var asyncWriter *AsyncWriter
	var rotatedWriter *RotatedWriter
	var outputWriter io.Writer

	// 设置日志级别
	level, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logrusLogger.SetLevel(level)

	// 设置日志格式
	var formatter logrus.Formatter
	switch strings.ToLower(cfg.Format) {
	case "text":
		formatter = &logrus.TextFormatter{
			FullTimestamp: true,
		}
	case "json":
		formatter = &logrus.JSONFormatter{
			DisableTimestamp: false,
		}
	default:
		formatter = &logrus.JSONFormatter{}
	}
	logrusLogger.SetFormatter(formatter)

	// 创建基础输出写入器
	switch strings.ToLower(cfg.Output) {
	case "stdout":
		outputWriter = os.Stdout
	case "stderr":
		outputWriter = os.Stderr
	case "file":
		var err error
		rotatedWriter, err = NewRotatedWriter(cfg.FilePath, cfg.MaxSize, cfg.MaxAge, cfg.MaxBackups)
		if err != nil {
			return nil, errors.WrapE(err, "create rotated writer", "file_path", cfg.FilePath)
		}
		outputWriter = rotatedWriter
	default:
		outputWriter = os.Stdout
	}

	// 如果启用异步日志，包装输出写入器
	if cfg.AsyncEnabled {
		asyncWriter = NewAsyncWriter(outputWriter, cfg.BufferSize, 100) // batchSize 固定为 100
		asyncWriter.SetFormatter(formatter)
		asyncWriter.Start()
		// 使用特殊的 AsyncWriterAdapter 来适配 logrus 的输出
		outputWriter = &AsyncWriterAdapter{asyncWriter}
	}

	// 设置日志输出
	logrusLogger.SetOutput(outputWriter)

	// 设置其他选项
	if !cfg.DisableCaller {
		logrusLogger.SetReportCaller(true)
	}

	logger := &Logger{
		Logger:        logrusLogger,
		config:        cfg,
		entry:         logrus.NewEntry(logrusLogger),
		fields:        make(logrus.Fields),
		asyncWriter:   asyncWriter,
		rotatedWriter: rotatedWriter,
	}

	return logger, nil
}

// NewOrMust 创建日志实例，如果出错则panic
func NewOrMust(cfg *Config) *Logger {
	logger, err := NewLogger(cfg)
	if err != nil {
		panic("logger: " + err.Error())
	}
	return logger
}

// NewLoggerFromConfigFile 从配置文件创建日志实例
func NewLoggerFromConfigFile(filename string) (*Logger, error) {
	cfg := &Config{}

	// 尝试从环境变量加载，文件作为备选
	if filename != "" {
		if data, err := os.ReadFile(filename); err == nil {
			if err := json.Unmarshal(data, cfg); err != nil {
				return nil, errors.WrapE(err, "parse config file", "filename", filename)
			}
		}
	}

	return NewLogger(cfg)
}

// WithField 添加单个字段到日志上下文
func (l *Logger) WithField(key string, value interface{}) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	newFields := make(logrus.Fields)
	for k, v := range l.fields {
		newFields[k] = v
	}
	newFields[key] = value

	return &Logger{
		Logger: l.Logger,
		config: l.config,
		entry:  l.entry.WithField(key, value),
		fields: newFields,
	}
}

// WithFields 添加多个字段到日志上下文
func (l *Logger) WithFields(fields logrus.Fields) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	newFields := make(logrus.Fields)
	for k, v := range l.fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}

	return &Logger{
		Logger: l.Logger,
		config: l.config,
		entry:  l.entry.WithFields(fields),
		fields: newFields,
	}
}

// WithError 添加错误到日志上下文
func (l *Logger) WithError(err error) *Logger {
	return l.WithField("error", err)
}

// Trace 记录追踪级别日志
func (l *Logger) Trace(args ...interface{}) {
	l.entry.Trace(args...)
}

// Debug 记录调试级别日志
func (l *Logger) Debug(args ...interface{}) {
	l.entry.Debug(args...)
}

// Info 记录信息级别日志
func (l *Logger) Info(args ...interface{}) {
	l.entry.Info(args...)
}

// Warn 记录警告级别日志
func (l *Logger) Warn(args ...interface{}) {
	l.entry.Warn(args...)
}

// Error 记录错误级别日志
func (l *Logger) Error(args ...interface{}) {
	l.entry.Error(args...)
}

// Fatal 记录致命错误级别日志
func (l *Logger) Fatal(args ...interface{}) {
	l.entry.Fatal(args...)
}

// Tracef 记录格式化追踪级别日志
func (l *Logger) Tracef(format string, args ...interface{}) {
	l.entry.Tracef(format, args...)
}

// Debugf 记录格式化调试级别日志
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.entry.Debugf(format, args...)
}

// Infof 记录格式化信息级别日志
func (l *Logger) Infof(format string, args ...interface{}) {
	l.entry.Infof(format, args...)
}

// Warnf 记录格式化警告级别日志
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.entry.Warnf(format, args...)
}

// Errorf 记录格式化错误级别日志
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.entry.Errorf(format, args...)
}

// Fatalf 记录格式化致命错误级别日志
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.entry.Fatalf(format, args...)
}

// SetLevel 动态设置日志级别
func (l *Logger) SetLevel(level string) error {
	logrusLevel, err := logrus.ParseLevel(level)
	if err != nil {
		return errors.WrapE(err, "parse log level", "level", level)
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	l.Logger.SetLevel(logrusLevel)
	l.config.Level = level
	return nil
}

// GetConfig 获取当前配置
func (l *Logger) GetConfig() *Config {
	return l.config
}

// GetLevel 获取当前日志级别
func (l *Logger) GetLevel() string {
	return l.Logger.GetLevel().String()
}

// IsLevelEnabled 检查指定日志级别是否启用
func (l *Logger) IsLevelEnabled(level logrus.Level) bool {
	return l.Logger.IsLevelEnabled(level)
}

// Global 获取全局日志实例
func Global() *Logger {
	once.Do(func() {
		if defaultLogger == nil {
			// 创建默认配置
			cfg := &Config{
				Level:  "info",
				Format: "json",
				Output: "stdout",
			}
			if logger, err := NewLogger(cfg); err == nil {
				defaultLogger = logger
			}
		}
	})
	return defaultLogger
}

// InitGlobal 初始化全局日志实例
func InitGlobal(cfg *Config) error {
	logger, err := NewLogger(cfg)
	if err != nil {
		return err
	}
	defaultLogger = logger
	return nil
}

// Sync 确保所有日志都已写入（用于异步日志）
func (l *Logger) Sync() error {
	// 同步日志输出（如果有缓冲）
	if syncer, ok := l.Logger.Out.(interface{ Sync() error }); ok {
		return syncer.Sync()
	}
	return nil
}

// NewEntry 创建新的日志条目
func (l *Logger) NewEntry() *logrus.Entry {
	return l.entry
}

// Copy 创建日志实例的副本
func (l *Logger) Copy() *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	newFields := make(logrus.Fields)
	for k, v := range l.fields {
		newFields[k] = v
	}

	return &Logger{
		Logger: l.Logger,
		config: l.config,
		entry:  l.entry,
		fields: newFields,
	}
}

// Close 关闭日志器，释放资源
func (l *Logger) Close() error {
	var errs []error

	if l.asyncWriter != nil {
		if err := l.asyncWriter.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if l.rotatedWriter != nil {
		if err := l.rotatedWriter.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errors.New("failed to close logger: " + fmt.Sprintf("%v", errs))
	}
	return nil
}

// String 返回日志配置信息
func (l *Logger) String() string {
	return fmt.Sprintf("Logger(level=%s, format=%s, output=%s)",
		l.config.Level, l.config.Format, l.config.Output)
}
