package logger

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"

	"reflect"

	"github.com/kaichao/gopkg/errors"
)

// Config 日志配置结构
type Config struct {
	// 日志级别: trace, debug, info, warn, error, fatal
	Level string `json:"level" env:"LOG_LEVEL" default:"info"`

	// 日志格式: text, json
	Format string `json:"format" env:"LOG_FORMAT" default:"json"`

	// 输出目标: stdout, stderr, file
	Output string `json:"output" env:"LOG_OUTPUT" default:"stdout"`

	// 日志文件路径
	FilePath string `json:"file_path" env:"LOG_FILE_PATH" default:"app.log"`

	// 日志轮转配置
	MaxSize    int `json:"max_size" env:"LOG_MAX_SIZE" default:"100"`     // MB
	MaxAge     int `json:"max_age" env:"LOG_MAX_AGE" default:"7"`         // days
	MaxBackups int `json:"max_backups" env:"LOG_MAX_BACKUPS" default:"5"` // count

	// 异步日志配置
	AsyncEnabled bool `json:"async_enabled" env:"LOG_ASYNC_ENABLED" default:"false"`
	BufferSize   int  `json:"buffer_size" env:"LOG_BUFFER_SIZE" default:"1000"`

	// 性能优化
	DisableCaller     bool `json:"disable_caller" env:"LOG_DISABLE_CALLER" default:"false"`
	DisableStacktrace bool `json:"disable_stacktrace" env:"LOG_DISABLE_STACKTRACE" default:"false"`
}

// LoadConfig 从环境变量加载配置
func LoadConfig() *Config {
	cfg := &Config{}

	// 使用反射从环境变量填充配置
	loadFromEnv(cfg)

	// 应用默认值
	if cfg.Level == "" {
		cfg.Level = "info"
	}
	if cfg.Format == "" {
		cfg.Format = "json"
	}
	if cfg.Output == "" {
		cfg.Output = "stdout"
	}

	return cfg
}

// loadFromEnv 从环境变量加载配置
func loadFromEnv(cfg *Config) {
	t := reflect.TypeOf(cfg).Elem()
	v := reflect.ValueOf(cfg).Elem()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		envTag := field.Tag.Get("env")
		defaultTag := field.Tag.Get("default")

		if envTag != "" {
			if envValue, exists := os.LookupEnv(envTag); exists {
				// 根据字段类型解析值
				switch field.Type.String() {
				case "string":
					v.Field(i).SetString(envValue)
				case "int":
					if intValue, err := strconv.Atoi(envValue); err == nil {
						v.Field(i).SetInt(int64(intValue))
					}
				case "bool":
					if boolValue, err := strconv.ParseBool(envValue); err == nil {
						v.Field(i).SetBool(boolValue)
					}
				}
			} else if defaultTag != "" {
				// 使用默认值
				switch field.Type.String() {
				case "string":
					v.Field(i).SetString(defaultTag)
				case "int":
					if intValue, err := strconv.Atoi(defaultTag); err == nil {
						v.Field(i).SetInt(int64(intValue))
					}
				case "bool":
					if boolValue, err := strconv.ParseBool(defaultTag); err == nil {
						v.Field(i).SetBool(boolValue)
					}
				}
			}
		}
	}
}

// ToJSON 将配置转换为JSON字符串
func (c *Config) ToJSON() string {
	bytes, _ := json.MarshalIndent(c, "", "  ")
	return string(bytes)
}

// Validate 验证配置有效性
func (c *Config) Validate() error {
	validLevels := map[string]bool{"trace": true, "debug": true, "info": true, "warn": true, "error": true, "fatal": true}
	if !validLevels[strings.ToLower(c.Level)] {
		return errors.New("invalid log level: " + c.Level)
	}

	validFormats := map[string]bool{"text": true, "json": true}
	if !validFormats[strings.ToLower(c.Format)] {
		return errors.New("invalid log format: " + c.Format)
	}

	// 验证异步日志配置
	if c.AsyncEnabled && c.BufferSize <= 0 {
		return errors.New("buffer size must be positive when async enabled")
	}

	// 验证轮转配置（如果使用文件输出）
	if strings.ToLower(c.Output) == "file" {
		if c.MaxSize <= 0 {
			return errors.New("max size must be positive for file output")
		}
		if c.MaxAge <= 0 {
			return errors.New("max age must be positive for file output")
		}
		if c.MaxBackups < 0 {
			return errors.New("max backups cannot be negative")
		}
	}

	return nil
}
