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
				return true // if not int type, treat as invalid
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
				return true // if not string type, treat as invalid
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
			// For boolean types, false is a valid value (zero value), so always return true meaning "zero value is valid"
			// Specific check is handled in getValueInternal via isFlagSetInCmd
			_, ok := v.(bool)
			return !ok // if not bool type, treat as invalid
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
			return time.ParseDuration(envValue)
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
				return true // if not related type, treat as invalid
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
				return true // if not int/int64 type, treat as invalid
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
				return true // if not float64 type, treat as invalid
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

// GetStringSlice retrieves a string slice parameter with priority: command line > environment variable > default value
func GetStringSlice(cmd *cobra.Command, name string, opts ...Option) ([]string, error) {
	value, err := getValueInternal(
		cmd,
		name,
		opts,
		func(flags *pflag.FlagSet) (interface{}, error) {
			return flags.GetStringSlice(name)
		},
		func(envValue string) (interface{}, error) {
			// Environment variable uses comma-separated strings
			return strings.Split(envValue, ","), nil
		},
		[]string(nil),
		func(v interface{}) bool {
			sliceValue, ok := v.([]string)
			if !ok {
				return true // if not []string type, treat as invalid
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
	isZeroValid func(interface{}) bool, // Check if value is zero (for boolean types, zero value false may be valid)
) (interface{}, error) {
	// Parse options
	opt := &options{}
	for _, o := range opts {
		o(opt)
	}
	if opt.envKey == "" {
		opt.envKey = getEnvKey(name)
	}

	// 1. Try to get from command line parameters
	value, found, err := tryGetFlag(cmd, name, flagGetter)
	if err != nil {
		return zeroValue, err
	}
	if found {
		// Only use if flag is explicitly set or value is non-zero (boolean types need special handling)
		if isFlagSetInCmd(cmd, name) || !isZeroValid(value) {
			if opt.validator != nil {
				if err := opt.validator(value); err != nil {
					return zeroValue, err
				}
			}
			return value, nil
		}
	}

	// 2. Try to get from environment variable
	if envValue := os.Getenv(opt.envKey); envValue != "" {
		if value, err := envParser(envValue); err == nil {
			if opt.validator != nil {
				if err := opt.validator(value); err != nil {
					return zeroValue, err
				}
			}
			return value, nil
		}
	}

	// 3. Check required parameter
	if opt.required {
		return zeroValue, errors.E(fmt.Sprintf("required parameter '%s' not provided", name))
	}

	// 4. Return default value
	if opt.defaultVal != nil {
		// Check if default value type matches expected type
		if isTypeCompatible(opt.defaultVal, zeroValue) {
			return opt.defaultVal, nil
		}
		// Type mismatch: silently ignore and continue as per original behavior
	}

	// 5. Try dynamic default value function
	if opt.defaultValFunc != nil {
		if value, err := opt.defaultValFunc(); err == nil {
			// Check if dynamic default value type matches expected type
			if isTypeCompatible(value, zeroValue) {
				return value, nil
			}
			// Type mismatch: silently ignore and continue as per original behavior
		}
	}

	// 6. Return zero value
	return zeroValue, nil
}

// getEnvKey automatically derives environment variable name from parameter name
// For example: app-id → APP_ID, cluster-name → CLUSTER_NAME
func getEnvKey(name string) string {
	// Convert command line parameter name to environment variable name
	// app-id → APP_ID, app-id2 → APP_ID2
	envKey := strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
	return envKey
}

// tryGetFlag attempts to get value from command flags (tries local flags first, then persistent flags)
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
	// Check if set elsewhere (handle errors)
	if flag := cmd.Flag(name); flag != nil && flag.Changed {
		// Local flag is set but retrieval failed, return error
		if _, err := getter(cmd.Flags()); err != nil {
			return nil, false, err
		}
	}
	if flag := cmd.PersistentFlags().Lookup(name); flag != nil && flag.Changed {
		// Persistent flag is set but retrieval failed, return error
		if _, err := getter(cmd.PersistentFlags()); err != nil {
			return nil, false, err
		}
	}
	return nil, false, nil
}

// isFlagSetInCmd checks if command line flag is set (checks both local and persistent flags)
func isFlagSetInCmd(cmd *cobra.Command, name string) bool {
	if flag := cmd.Flag(name); flag != nil {
		return flag.Changed
	}
	if flag := cmd.PersistentFlags().Lookup(name); flag != nil {
		return flag.Changed
	}
	return false
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
