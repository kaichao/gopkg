package param

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/kaichao/gopkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// GetInt retrieves an integer parameter with priority: command line > environment variable > default value
func GetInt(cmd *cobra.Command, name string, opts ...Option) (int, error) {
	value, err := getValueInternal(
		cmd,
		name,
		opts,
		func(flags *pflag.FlagSet) (interface{}, error) {
			return flags.GetInt(name)
		},
		func(envValue string) (interface{}, error) {
			return strconv.Atoi(envValue)
		},
		0,
		func(v interface{}) bool {
			intValue, ok := v.(int)
			if !ok {
				return true // if not int type, treat as placeholder
			}
			return intValue == 0
		},
	)
	if err != nil {
		return 0, err
	}
	intValue, ok := value.(int)
	if !ok {
		return 0, errors.E(fmt.Sprintf("internal error: unexpected type %T for int parameter", value))
	}
	return intValue, nil
}

// GetString retrieves a string parameter with priority: command line > environment variable > default value
func GetString(cmd *cobra.Command, name string, opts ...Option) (string, error) {
	value, err := getValueInternal(
		cmd,
		name,
		opts,
		func(flags *pflag.FlagSet) (interface{}, error) {
			return flags.GetString(name)
		},
		func(envValue string) (interface{}, error) {
			// For strings, environment variable value is used directly, no conversion needed
			return envValue, nil
		},
		"",
		func(v interface{}) bool {
			strValue, ok := v.(string)
			if !ok {
				return true // if not string type, treat as placeholder
			}
			return strValue == ""
		},
	)
	if err != nil {
		return "", err
	}
	strValue, ok := value.(string)
	if !ok {
		return "", errors.E(fmt.Sprintf("internal error: unexpected type %T for string parameter", value))
	}
	return strValue, nil
}

// GetBool retrieves a boolean parameter with priority: command line > environment variable > default value
func GetBool(cmd *cobra.Command, name string, opts ...Option) (bool, error) {
	value, err := getValueInternal(
		cmd,
		name,
		opts,
		func(flags *pflag.FlagSet) (interface{}, error) {
			return flags.GetBool(name)
		},
		func(envValue string) (interface{}, error) {
			return strconv.ParseBool(envValue)
		},
		false,
		func(v interface{}) bool {
			// For boolean types, false is a valid value (not a placeholder),
			// so always return false meaning "false is explicit, not a placeholder"
			_, ok := v.(bool)
			return !ok // if not bool type, treat as placeholder
		},
	)
	if err != nil {
		return false, err
	}
	boolValue, ok := value.(bool)
	if !ok {
		return false, errors.E(fmt.Sprintf("internal error: unexpected type %T for bool parameter", value))
	}
	return boolValue, nil
}

// GetDuration retrieves a time.Duration parameter with priority: command line > environment variable > default value
func GetDuration(cmd *cobra.Command, name string, opts ...Option) (time.Duration, error) {
	value, err := getValueInternal(
		cmd,
		name,
		opts,
		func(flags *pflag.FlagSet) (interface{}, error) {
			return flags.GetDuration(name)
		},
		func(envValue string) (interface{}, error) {
			// Try standard duration format first (e.g., "30s", "5m")
			if d, err := time.ParseDuration(envValue); err == nil {
				return d, nil
			}
			// Fallback: try parsing as a plain number of seconds (e.g., "30" means 30s)
			if seconds, err := strconv.ParseInt(envValue, 10, 64); err == nil {
				return time.Duration(seconds) * time.Second, nil
			}
			return nil, fmt.Errorf("cannot parse '%s' as duration: try formats like '30s', '5m', '1h' or a plain number of seconds", envValue)
		},
		time.Duration(0),
		func(v interface{}) bool {
			// Handle time.Duration and int64 types (time.Duration is an alias for int64)
			switch val := v.(type) {
			case time.Duration:
				return val == 0
			case int64:
				return val == 0
			case int:
				return val == 0
			default:
				return true // if not related type, treat as placeholder
			}
		},
	)
	if err != nil {
		return 0, err
	}
	// Handle possible time.Duration, int64, or int types
	switch val := value.(type) {
	case time.Duration:
		return val, nil
	case int64:
		return time.Duration(val), nil
	case int:
		return time.Duration(val), nil
	default:
		return 0, errors.E(fmt.Sprintf("internal error: unexpected type %T for time.Duration parameter", value))
	}
}

// GetInt64 retrieves an int64 parameter with priority: command line > environment variable > default value
func GetInt64(cmd *cobra.Command, name string, opts ...Option) (int64, error) {
	value, err := getValueInternal(
		cmd,
		name,
		opts,
		func(flags *pflag.FlagSet) (interface{}, error) {
			return flags.GetInt64(name)
		},
		func(envValue string) (interface{}, error) {
			return strconv.ParseInt(envValue, 10, 64)
		},
		int64(0),
		func(v interface{}) bool {
			// Handle int and int64 types
			switch val := v.(type) {
			case int64:
				return val == 0
			case int:
				return val == 0
			default:
				return true // if not int/int64 type, treat as placeholder
			}
		},
	)
	if err != nil {
		return 0, err
	}
	// Handle possible int or int64 types
	switch val := value.(type) {
	case int64:
		return val, nil
	case int:
		return int64(val), nil
	default:
		return 0, errors.E(fmt.Sprintf("internal error: unexpected type %T for int64 parameter", value))
	}
}

// GetFloat64 retrieves a float64 parameter with priority: command line > environment variable > default value
func GetFloat64(cmd *cobra.Command, name string, opts ...Option) (float64, error) {
	value, err := getValueInternal(
		cmd,
		name,
		opts,
		func(flags *pflag.FlagSet) (interface{}, error) {
			return flags.GetFloat64(name)
		},
		func(envValue string) (interface{}, error) {
			return strconv.ParseFloat(envValue, 64)
		},
		0.0,
		func(v interface{}) bool {
			floatValue, ok := v.(float64)
			if !ok {
				return true // if not float64 type, treat as placeholder
			}
			return floatValue == 0.0
		},
	)
	if err != nil {
		return 0, err
	}
	floatValue, ok := value.(float64)
	if !ok {
		return 0, errors.E(fmt.Sprintf("internal error: unexpected type %T for float64 parameter", value))
	}
	return floatValue, nil
}

// preParseSeparator extracts a custom separator from options for GetStringSlice
func preParseSeparator(opts []Option) string {
	opt := &options{}
	for _, o := range opts {
		o(opt)
	}
	if opt.separator != "" {
		return opt.separator
	}
	return ","
}

// GetStringSlice retrieves a string slice parameter with priority: command line > environment variable > default value
func GetStringSlice(cmd *cobra.Command, name string, opts ...Option) ([]string, error) {
	sep := preParseSeparator(opts)

	value, err := getValueInternal(
		cmd,
		name,
		opts,
		func(flags *pflag.FlagSet) (interface{}, error) {
			return flags.GetStringSlice(name)
		},
		func(envValue string) (interface{}, error) {
			// Use custom separator if provided, default to comma
			if envValue == "" {
				return []string{}, nil
			}
			return strings.Split(envValue, sep), nil
		},
		[]string(nil),
		func(v interface{}) bool {
			sliceValue, ok := v.([]string)
			if !ok {
				return true // if not []string type, treat as placeholder
			}
			return len(sliceValue) == 0
		},
	)
	if err != nil {
		return nil, err
	}
	sliceValue, ok := value.([]string)
	if !ok {
		return nil, errors.E(fmt.Sprintf("internal error: unexpected type %T for []string parameter", value))
	}
	return sliceValue, nil
}

// getValueInternal common value retrieval logic
func getValueInternal(
	cmd *cobra.Command,
	name string,
	opts []Option,
	flagGetter func(*pflag.FlagSet) (interface{}, error),
	envParser func(string) (interface{}, error),
	zeroValue interface{},
	isZeroPlaceholder func(interface{}) bool, // returns true if value is a zero placeholder (not explicitly set)
) (interface{}, error) {
	// Parse options
	opt := &options{}
	for _, o := range opts {
		o(opt)
	}
	if opt.envKey == "" {
		opt.envKey = getEnvKey(name)
	}
	if opt.separator == "" {
		opt.separator = "," // default separator for string slice
	}

	// 1. Try to get from command line parameters (only when explicitly set by user)
	value, found, err := tryGetFlag(cmd, name, flagGetter)
	if err != nil {
		return zeroValue, err
	}
	if found {
		if opt.validator != nil {
			if err := opt.validator(value); err != nil {
				return zeroValue, errors.E(fmt.Sprintf("flag '%s' validation failed: %v", name, err))
			}
		}
		return value, nil
	}

	// 2. Try to get from environment variable
	if envValue, envSet := os.LookupEnv(opt.envKey); envSet {
		// Accept empty string as a valid value
		parsedValue, err := envParser(envValue)
		if err != nil {
			return zeroValue, errors.E(fmt.Sprintf("environment variable '%s'='%s' parse error: %v", opt.envKey, envValue, err))
		}
		if opt.validator != nil {
			if err := opt.validator(parsedValue); err != nil {
				return zeroValue, errors.E(fmt.Sprintf("environment variable '%s' validation failed: %v", opt.envKey, err))
			}
		}
		return parsedValue, nil
	}

	// 3. Check required parameter
	if opt.required {
		return zeroValue, errors.E(fmt.Sprintf("required parameter '%s' not provided (set via --%s flag or %s environment variable)", name, name, opt.envKey))
	}

	// 4. Return default value
	if opt.defaultVal != nil {
		// Check if default value type matches expected type
		if isTypeCompatible(opt.defaultVal, zeroValue) {
			if opt.validator != nil {
				if err := opt.validator(opt.defaultVal); err != nil {
					return zeroValue, errors.E(fmt.Sprintf("default value for '%s' validation failed: %v", name, err))
				}
			}
			return opt.defaultVal, nil
		}
		// Type mismatch: warn but continue
		fmt.Printf("param: warning: default value type %T does not match expected type for parameter '%s', ignoring\n", opt.defaultVal, name)
	}

	// 5. Try dynamic default value function
	if opt.defaultValFunc != nil {
		if value, err := opt.defaultValFunc(); err == nil {
			// Check if dynamic default value type matches expected type
			if isTypeCompatible(value, zeroValue) {
				if opt.validator != nil {
					if err := opt.validator(value); err != nil {
						return zeroValue, errors.E(fmt.Sprintf("dynamic default value for '%s' validation failed: %v", name, err))
					}
				}
				return value, nil
			}
			// Type mismatch: warn but continue
			fmt.Printf("param: warning: dynamic default function returned type %T for parameter '%s', does not match expected type, ignoring\n", value, name)
		} else {
			// Dynamic default function error: propagate it since we have no other fallback
			return zeroValue, errors.E(fmt.Sprintf("dynamic default function for '%s' failed: %v", name, err))
		}
	}

	// 6. Try flag's built-in default value (as a fallback when no other default provided)
	if flagValue, found, err := tryFlagDefault(cmd, name, flagGetter, isZeroPlaceholder); found && err == nil {
		if opt.validator != nil {
			if err := opt.validator(flagValue); err != nil {
				return zeroValue, errors.E(fmt.Sprintf("flag '%s' default value validation failed: %v", name, err))
			}
		}
		return flagValue, nil
	}

	// 7. Return zero value
	return zeroValue, nil
}

// getEnvKey automatically derives environment variable name from parameter name
// For example: app-id -> APP_ID, cluster-name -> CLUSTER_NAME
func getEnvKey(name string) string {
	// Convert command line parameter name to environment variable name
	transformed := strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
	return transformed
}

// tryGetFlag attempts to get value from command flags (tries local flags first, then persistent flags).
// Only returns found=true when the flag is explicitly set by the user.
func tryGetFlag(cmd *cobra.Command, name string, getter func(*pflag.FlagSet) (interface{}, error)) (interface{}, bool, error) {
	// First try local flags
	if flag := cmd.Flag(name); flag != nil && flag.Changed {
		if value, err := getter(cmd.Flags()); err == nil {
			return value, true, nil
		}
	}
	// Then try persistent flags
	if flag := cmd.PersistentFlags().Lookup(name); flag != nil && flag.Changed {
		if value, err := getter(cmd.PersistentFlags()); err == nil {
			return value, true, nil
		}
	}
	return nil, false, nil
}

// tryFlagDefault attempts to get the default value from a flag if the flag exists.
// This is used as a fallback when no user-provided value, env var, or WithDefault is available.
func tryFlagDefault(cmd *cobra.Command, name string, getter func(*pflag.FlagSet) (interface{}, error), isZeroPlaceholder func(interface{}) bool) (interface{}, bool, error) {
	// First try local flags
	if cmd.Flag(name) != nil {
		if value, err := getter(cmd.Flags()); err == nil {
			if !isZeroPlaceholder(value) {
				return value, true, nil
			}
		}
	}
	// Then try persistent flags
	if cmd.PersistentFlags().Lookup(name) != nil {
		if value, err := getter(cmd.PersistentFlags()); err == nil {
			if !isZeroPlaceholder(value) {
				return value, true, nil
			}
		}
	}
	return nil, false, nil
}

// isTypeCompatible checks if two values are type compatible
func isTypeCompatible(value, zeroValue interface{}) bool {
	// Use reflection to check type compatibility
	switch zeroValue.(type) {
	case int:
		_, ok := value.(int)
		return ok
	case int64:
		_, ok := value.(int64)
		if ok {
			return true
		}
		// Allow int assignment to int64
		_, ok2 := value.(int)
		return ok2
	case float64:
		_, ok := value.(float64)
		return ok
	case string:
		_, ok := value.(string)
		return ok
	case bool:
		_, ok := value.(bool)
		return ok
	case time.Duration:
		_, ok := value.(time.Duration)
		if ok {
			return true
		}
		// Allow int assignment to time.Duration (since time.Duration is an alias for int64)
		_, ok2 := value.(int)
		return ok2
	case []string:
		_, ok := value.([]string)
		return ok
	default:
		// For other types, use type assertion check
		return value == nil || fmt.Sprintf("%T", value) == fmt.Sprintf("%T", zeroValue)
	}
}
