// Advanced usage examples for param package
// Shows validation, required parameters, and complex dynamic defaults
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/kaichao/gopkg/errors"
	"github.com/kaichao/gopkg/param"
	"github.com/spf13/cobra"
)

func main() {
	// Example 1: Validation
	fmt.Println("=== Example 1: Parameter validation ===")
	cmd1 := &cobra.Command{}
	cmd1.Flags().Int("port", 0, "Server port")

	// This should work
	cmd1.Flags().Set("port", "8080")
	port, err := param.GetInt(cmd1, "port",
		param.WithValidator(func(v interface{}) error {
			p := v.(int)
			if p < 1 || p > 65535 {
				return errors.E(fmt.Sprintf("port must be between 1 and 65535, got %d", p))
			}
			return nil
		}),
	)
	if err != nil {
		fmt.Printf("Validation error: %v\n", err)
	} else {
		fmt.Printf("Valid port: %d\n\n", port)
	}

	// Example 2: Required parameter
	fmt.Println("=== Example 2: Required parameter ===")
	cmd2 := &cobra.Command{}
	cmd2.Flags().String("api-key", "", "API key")

	// This should fail
	_, err = param.GetString(cmd2, "api-key", param.WithRequired())
	if err != nil {
		fmt.Printf("Expected error for missing required parameter: %v\n\n", err)
	}

	// Example 3: Dynamic default from environment
	fmt.Println("=== Example 3: Dynamic default from environment ===")
	os.Setenv("DEFAULT_TIMEOUT", "30s")
	defer os.Unsetenv("DEFAULT_TIMEOUT")

	cmd3 := &cobra.Command{}
	cmd3.Flags().Duration("timeout", 0, "Request timeout")

	timeout, err := param.GetDuration(cmd3, "timeout",
		param.WithDefaultFunc(func() (interface{}, error) {
			// Read from environment variable as fallback
			if val := os.Getenv("DEFAULT_TIMEOUT"); val != "" {
				return time.ParseDuration(val)
			}
			return 10 * time.Second, nil
		}),
	)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Timeout: %v (from dynamic default function)\n\n", timeout)

	// Example 4: Custom environment variable key
	fmt.Println("=== Example 4: Custom environment variable key ===")
	os.Setenv("APP_DB_HOST", "localhost")
	defer os.Unsetenv("APP_DB_HOST")

	cmd4 := &cobra.Command{}
	cmd4.Flags().String("db-host", "", "Database host")

	dbHost, err := param.GetString(cmd4, "db-host",
		param.WithEnvKey("APP_DB_HOST"),
		param.WithDefault("127.0.0.1"),
	)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Database host: %s (from environment variable APP_DB_HOST)\n\n", dbHost)

	// Example 5: String slice with validation
	fmt.Println("=== Example 5: String slice with validation ===")
	cmd5 := &cobra.Command{}
	cmd5.Flags().StringSlice("roles", nil, "User roles")

	cmd5.Flags().Set("roles", "admin,user")
	roles, err := param.GetStringSlice(cmd5, "roles",
		param.WithValidator(func(v interface{}) error {
			roles := v.([]string)
			if len(roles) == 0 {
				return errors.E("at least one role is required")
			}
			for _, role := range roles {
				if role == "" {
					return errors.E("role cannot be empty")
				}
			}
			return nil
		}),
	)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Roles: %v\n", roles)
}
