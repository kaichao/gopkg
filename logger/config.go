package logger

import (
	"encoding/json"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/kaichao/gopkg/errors"
)

// Config defines logging configuration structure
type Config struct {
	// Log level: trace, debug, info, warn, error, fatal
	Level string `json:"level" env:"LOG_LEVEL" default:"info"`

	// Log format: text, json
	Format string `json:"format" env:"LOG_FORMAT" default:"json"`

	// Output destination: stdout, stderr, file
	Output string `json:"output" env:"LOG_OUTPUT" default:"stdout"`

	// Log file path
	FilePath string `json:"file_path" env:"LOG_FILE_PATH" default:"app.log"`

	// Log rotation configuration
	MaxSize    int `json:"max_size" env:"LOG_MAX_SIZE" default:"100"`     // MB
	MaxAge     int `json:"max_age" env:"LOG_MAX_AGE" default:"7"`         // days
	MaxBackups int `json:"max_backups" env:"LOG_MAX_BACKUPS" default:"5"` // count

	// Async logging configuration
	AsyncEnabled bool `json:"async_enabled" env:"LOG_ASYNC_ENABLED" default:"false"`
	BufferSize   int  `json:"buffer_size" env:"LOG_BUFFER_SIZE" default:"1000"`

	// Performance optimization
	DisableCaller     bool `json:"disable_caller" env:"LOG_DISABLE_CALLER" default:"false"`
	DisableStacktrace bool `json:"disable_stacktrace" env:"LOG_DISABLE_STACKTRACE" default:"false"`
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	cfg := &Config{}

	// Use reflection to populate config from environment
	loadFromEnv(cfg)

	// Apply default values
	if cfg.Level == "" {
		cfg.Level = "info"
	}
	if cfg.Format == "" {
		cfg.Format = "json"
	}
	if cfg.Output == "" {
		cfg.Output = "stdout"
	}
	if cfg.FilePath == "" {
		cfg.FilePath = "app.log"
	}
	if cfg.MaxSize <= 0 {
		cfg.MaxSize = 100
	}
	if cfg.MaxAge <= 0 {
		cfg.MaxAge = 7
	}
	if cfg.MaxBackups <= 0 {
		cfg.MaxBackups = 5
	}
	if cfg.BufferSize <= 0 {
		cfg.BufferSize = 1000
	}

	return cfg
}

// loadFromEnv loads configuration from environment variables using reflection
func loadFromEnv(cfg *Config) {
	val := reflect.ValueOf(cfg).Elem()
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// Get env tag
		envTag := fieldType.Tag.Get("env")
		if envTag == "" {
			continue
		}

		// Get default tag
		defaultTag := fieldType.Tag.Get("default")

		// Try to get value from environment
		if envValue, exists := os.LookupEnv(envTag); exists {
			// Parse value based on field type
			switch field.Type().String() {
			case "string":
				field.SetString(envValue)
			case "int":
				if intValue, err := strconv.Atoi(envValue); err == nil {
					field.SetInt(int64(intValue))
				}
			case "bool":
				if boolValue, err := strconv.ParseBool(envValue); err == nil {
					field.SetBool(boolValue)
				}
			}
		} else if defaultTag != "" {
			// Use default value
			switch field.Type().String() {
			case "string":
				field.SetString(defaultTag)
			case "int":
				if intValue, err := strconv.Atoi(defaultTag); err == nil {
					field.SetInt(int64(intValue))
				}
			case "bool":
				if boolValue, err := strconv.ParseBool(defaultTag); err == nil {
					field.SetBool(boolValue)
				}
			}
		}
	}
}

// ToJSON converts config to JSON string
func (c *Config) ToJSON() string {
	data, _ := json.MarshalIndent(c, "", "  ")
	return string(data)
}

// Validate validates configuration
func (c *Config) Validate() error {
	// Validate log level
	validLevels := map[string]bool{
		"trace": true,
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
		"fatal": true,
	}
	if !validLevels[strings.ToLower(c.Level)] {
		return errors.E("invalid log level: %s", c.Level)
	}

	// Validate log format
	validFormats := map[string]bool{
		"text": true,
		"json": true,
	}
	if !validFormats[strings.ToLower(c.Format)] {
		return errors.E("invalid log format: %s", c.Format)
	}

	// Validate output destination
	validOutputs := map[string]bool{
		"stdout": true,
		"stderr": true,
		"file":   true,
	}
	if !validOutputs[strings.ToLower(c.Output)] {
		return errors.E("invalid output destination: %s", c.Output)
	}

	// Validate async logging configuration
	if c.AsyncEnabled && c.BufferSize <= 0 {
		return errors.E("buffer size must be positive when async logging is enabled")
	}

	// Validate rotation configuration (if using file output)
	if strings.ToLower(c.Output) == "file" {
		if c.MaxSize <= 0 {
			return errors.E("max size must be positive for file output")
		}
		if c.MaxAge <= 0 {
			return errors.E("max age must be positive for file output")
		}
		if c.MaxBackups <= 0 {
			return errors.E("max backups must be positive for file output")
		}
	}

	return nil
}
