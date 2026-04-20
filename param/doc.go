// Package param provides unified command line parameter management for Go with Cobra.
//
// Design Goals:
// 1. Unified parameter retrieval interface supporting multiple data types
// 2. Priority: command line arguments > environment variables > static default values > dynamic default functions
// 3. Dynamic default value functions (from databases, config files, etc.)
// 4. Simplify command implementation, reduce boilerplate code
// 5. Parameter validation and required parameter checking
//
// Core Features:
// - Type Safety: Supports int, string, bool, time.Duration, int64, float64, []string
// - Automatic Environment Variable Name Derivation: Parameter names automatically converted to uppercase with underscores
// - Dynamic Default Values: Runtime-computed defaults via WithDefaultFunc
// - Parameter Validation: Custom validation logic via WithValidator
// - Required Parameters: Mark parameters as required via WithRequired
//
// Usage Examples:
//
//	import "github.com/spf13/cobra"
//	import "github.com/kaichao/gopkg/param"
//
//	// Basic usage: get integer parameter with automatic environment variable name derivation
//	appID, err := param.GetInt(cmd, "app-id")
//	if err != nil {
//	    return err
//	}
//
//	// With options: specify environment variable name, default value, and required flag
//	cluster, err := param.GetString(cmd, "cluster",
//	    param.WithEnvKey("MY_CLUSTER"),
//	    param.WithDefault("default-cluster"),
//	    param.WithRequired(),
//	)
//	if err != nil {
//	    return err
//	}
//
//	// Using dynamic default value function
//	import "time"
//	timeout, err := param.GetDuration(cmd, "timeout",
//	    param.WithDefaultFunc(func() (interface{}, error) {
//	        // Get default from config or database
//	        return 30 * time.Second, nil
//	    }),
//	)
//	if err != nil {
//	    return err
//	}
//
//	// Using parameter validation
//	import "github.com/kaichao/gopkg/errors"
//	port, err := param.GetInt(cmd, "port",
//	    param.WithValidator(func(v interface{}) error {
//	        port := v.(int)
//	        if port < 1 || port > 65535 {
//	            return errors.E("port must be between 1 and 65535")
//	        }
//	        return nil
//	    }),
//	)
//	if err != nil {
//	    return err
//	}
//
// Available Functions:
//
//	GetInt(cmd *cobra.Command, name string, opts ...Option) (int, error)
//	GetString(cmd *cobra.Command, name string, opts ...Option) (string, error)
//	GetBool(cmd *cobra.Command, name string, opts ...Option) (bool, error)
//	GetDuration(cmd *cobra.Command, name string, opts ...Option) (time.Duration, error)
//	GetInt64(cmd *cobra.Command, name string, opts ...Option) (int64, error)
//	GetFloat64(cmd *cobra.Command, name string, opts ...Option) (float64, error)
//	GetStringSlice(cmd *cobra.Command, name string, opts ...Option) ([]string, error)
//
// Available Options:
//
//	WithEnvKey(key string) Option        // Specify environment variable key
//	WithDefault(val interface{}) Option  // Specify static default value
//	WithDefaultFunc(f DefaultValueFunc) Option  // Specify dynamic default value function
//	WithRequired() Option                // Mark parameter as required
//	WithValidator(v func(interface{}) error) Option  // Add custom validator
//
// Environment Variable Name Derivation:
//
//	Parameter names with "-" are replaced with "_" and converted to uppercase
//	For example: app-id -> APP_ID, cluster-name -> CLUSTER_NAME
//
// Notes:
// - If neither command line argument nor environment variable is set, the default value (static or dynamic) is used
// - If a parameter is marked as required (WithRequired) but not provided, an error is returned
// - Dynamic default functions are only called when no other source provides a value
// - Validators are executed on values from all sources
package param
