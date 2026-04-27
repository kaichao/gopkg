package param

import (
	"os"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

// setupTestCmd creates a cobra.Command with all supported flag types
func setupTestCmd() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.PersistentFlags().String("test-string", "default-str", "Test string flag")
	cmd.PersistentFlags().Int("test-int", 0, "Test int flag")
	cmd.PersistentFlags().Bool("test-bool", false, "Test bool flag")
	cmd.PersistentFlags().Duration("test-duration", 0, "Test duration flag")
	cmd.PersistentFlags().Int64("test-int64", 0, "Test int64 flag")
	cmd.PersistentFlags().Float64("test-float64", 0, "Test float64 flag")
	cmd.PersistentFlags().StringSlice("test-slice", nil, "Test slice flag")
	cmd.PersistentFlags().String("test-str", "", "Test str flag")
	cmd.PersistentFlags().String("empty-str", "", "Test empty str flag")
	cmd.PersistentFlags().StringSlice("comma-slice", nil, "Test comma slice")
	cmd.PersistentFlags().String("plain-str", "", "Test plain string flag")
	return cmd
}

// ==========================================
// 1. Dynamic Default Function Tests
// ==========================================

func TestWithDefaultFunc(t *testing.T) {
	cmd := setupTestCmd()

	t.Run("String dynamic default", func(t *testing.T) {
		got, err := GetString(cmd, "test-string", WithDefaultFunc(func() (interface{}, error) {
			return "dynamic-default", nil
		}))
		if err != nil {
			t.Fatalf("GetString() error = %v", err)
		}
		if got != "dynamic-default" {
			t.Errorf("GetString() = %v, want dynamic-default", got)
		}
	})

	t.Run("Int dynamic default", func(t *testing.T) {
		got, err := GetInt(cmd, "test-int", WithDefaultFunc(func() (interface{}, error) {
			return 42, nil
		}))
		if err != nil {
			t.Fatalf("GetInt() error = %v", err)
		}
		if got != 42 {
			t.Errorf("GetInt() = %v, want 42", got)
		}
	})

	t.Run("Bool dynamic default", func(t *testing.T) {
		got, err := GetBool(cmd, "test-bool", WithDefaultFunc(func() (interface{}, error) {
			return true, nil
		}))
		if err != nil {
			t.Fatalf("GetBool() error = %v", err)
		}
		if got != true {
			t.Errorf("GetBool() = %v, want true", got)
		}
	})

	t.Run("Duration dynamic default", func(t *testing.T) {
		got, err := GetDuration(cmd, "test-duration", WithDefaultFunc(func() (interface{}, error) {
			return 30 * time.Second, nil
		}))
		if err != nil {
			t.Fatalf("GetDuration() error = %v", err)
		}
		if got != 30*time.Second {
			t.Errorf("GetDuration() = %v, want 30s", got)
		}
	})

	t.Run("Int64 dynamic default", func(t *testing.T) {
		got, err := GetInt64(cmd, "test-int64", WithDefaultFunc(func() (interface{}, error) {
			return int64(1000), nil
		}))
		if err != nil {
			t.Fatalf("GetInt64() error = %v", err)
		}
		if got != 1000 {
			t.Errorf("GetInt64() = %v, want 1000", got)
		}
	})

	t.Run("Float64 dynamic default", func(t *testing.T) {
		got, err := GetFloat64(cmd, "test-float64", WithDefaultFunc(func() (interface{}, error) {
			return 3.14, nil
		}))
		if err != nil {
			t.Fatalf("GetFloat64() error = %v", err)
		}
		if got != 3.14 {
			t.Errorf("GetFloat64() = %v, want 3.14", got)
		}
	})

	t.Run("StringSlice dynamic default", func(t *testing.T) {
		got, err := GetStringSlice(cmd, "test-slice", WithDefaultFunc(func() (interface{}, error) {
			return []string{"a", "b", "c"}, nil
		}))
		if err != nil {
			t.Fatalf("GetStringSlice() error = %v", err)
		}
		if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
			t.Errorf("GetStringSlice() = %v, want [a b c]", got)
		}
	})
}

// ==========================================
// 2. Priority Tests
// ==========================================

func TestPriority(t *testing.T) {
	cmd := setupTestCmd()

	t.Run("Priority: command line > environment > static default > dynamic default", func(t *testing.T) {
		os.Setenv("TEST_INT", "25")
		defer os.Unsetenv("TEST_INT")

		// Set command line flag
		cmd.PersistentFlags().Set("test-int", "30")

		got, err := GetInt(cmd, "test-int",
			WithDefault(20),
			WithDefaultFunc(func() (interface{}, error) {
				return 42, nil
			}),
		)
		if err != nil {
			t.Fatalf("GetInt() error = %v", err)
		}
		if got != 30 {
			t.Errorf("GetInt() = %v, want 30 (command line should win)", got)
		}
	})

	t.Run("Environment variable priority over defaults", func(t *testing.T) {
		os.Setenv("TEST_INT2", "99")
		defer os.Unsetenv("TEST_INT2")

		got, err := GetInt(cmd, "test-int2",
			WithDefault(50),
			WithDefaultFunc(func() (interface{}, error) {
				return 75, nil
			}),
		)
		if err != nil {
			t.Fatalf("GetInt() error = %v", err)
		}
		if got != 99 {
			t.Errorf("GetInt() = %v, want 99 (environment should win)", got)
		}
	})

	t.Run("Static default priority over dynamic default", func(t *testing.T) {
		got, err := GetInt(cmd, "test-int3",
			WithDefault(88),
			WithDefaultFunc(func() (interface{}, error) {
				return 77, nil
			}),
		)
		if err != nil {
			t.Fatalf("GetInt() error = %v", err)
		}
		if got != 88 {
			t.Errorf("GetInt() = %v, want 88 (static default should win)", got)
		}
	})

	t.Run("Dynamic default used when no other source available", func(t *testing.T) {
		got, err := GetInt(cmd, "test-int4",
			WithDefaultFunc(func() (interface{}, error) {
				return 42, nil
			}),
		)
		if err != nil {
			t.Fatalf("GetInt() error = %v", err)
		}
		if got != 42 {
			t.Errorf("GetInt() = %v, want 42 (dynamic default should be used)", got)
		}
	})
}

// ==========================================
// 3. Bool flag with explicit false value
// ==========================================

func TestBoolExplicitFalse(t *testing.T) {
	t.Run("Bool with explicit --flag=false from command line", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.PersistentFlags().Bool("verbose", true, "Verbose mode")

		// Set the flag to false explicitly
		cmd.PersistentFlags().Set("verbose", "false")

		got, err := GetBool(cmd, "verbose")
		if err != nil {
			t.Fatalf("GetBool() error = %v", err)
		}
		if got != false {
			t.Errorf("GetBool() = %v, want false (--verbose=false should be respected)", got)
		}
	})

	t.Run("Bool required + explicit false should be accepted", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.PersistentFlags().Bool("enabled", true, "Enabled flag")

		// Set to false explicitly
		cmd.PersistentFlags().Set("enabled", "false")

		got, err := GetBool(cmd, "enabled", WithRequired())
		if err != nil {
			t.Fatalf("GetBool() with required should accept explicit false, got error: %v", err)
		}
		if got != false {
			t.Errorf("GetBool() = %v, want false", got)
		}
	})

	t.Run("Bool from environment variable false", func(t *testing.T) {
		os.Setenv("MY_BOOL", "false")
		defer os.Unsetenv("MY_BOOL")

		cmd := &cobra.Command{}
		cmd.PersistentFlags().Bool("my-bool", true, "Test bool")

		got, err := GetBool(cmd, "my-bool", WithEnvKey("MY_BOOL"))
		if err != nil {
			t.Fatalf("GetBool() error = %v", err)
		}
		if got != false {
			t.Errorf("GetBool() = %v, want false", got)
		}
	})
}

// ==========================================
// 4. Empty Environment Variable Tests
// ==========================================

func TestEmptyEnvVar(t *testing.T) {
	t.Run("Empty env var for string should be accepted", func(t *testing.T) {
		os.Setenv("EMPTY_STR", "")
		defer os.Unsetenv("EMPTY_STR")

		cmd := &cobra.Command{}
		cmd.PersistentFlags().String("empty-str", "default", "Test empty string")

		got, err := GetString(cmd, "empty-str", WithEnvKey("EMPTY_STR"))
		if err != nil {
			t.Fatalf("GetString() error = %v", err)
		}
		if got != "" {
			t.Errorf("GetString() = %q, want empty string (empty env var should be accepted)", got)
		}
	})

	t.Run("Unset env var should fall through to default", func(t *testing.T) {
		os.Unsetenv("UNSET_VAR")

		cmd := &cobra.Command{}
		cmd.PersistentFlags().String("unset-str", "fallback-default", "Test unset string")

		got, err := GetString(cmd, "unset-str", WithEnvKey("UNSET_VAR"))
		if err != nil {
			t.Fatalf("GetString() error = %v", err)
		}
		if got != "fallback-default" {
			t.Errorf("GetString() = %q, want fallback-default", got)
		}
	})
}

// ==========================================
// 5. Required Parameter Tests
// ==========================================

func TestRequired(t *testing.T) {
	t.Run("Required parameter not provided should return error", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.PersistentFlags().String("api-key", "", "API key")

		_, err := GetString(cmd, "api-key", WithRequired())
		if err == nil {
			t.Errorf("GetString() should return error for missing required parameter")
		}
	})

	t.Run("Required parameter provided via flag should pass", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.PersistentFlags().String("api-key", "", "API key")
		cmd.PersistentFlags().Set("api-key", "my-secret-key")

		got, err := GetString(cmd, "api-key", WithRequired())
		if err != nil {
			t.Fatalf("GetString() error = %v", err)
		}
		if got != "my-secret-key" {
			t.Errorf("GetString() = %v, want my-secret-key", got)
		}
	})

	t.Run("Required parameter provided via env should pass", func(t *testing.T) {
		os.Setenv("DB_HOST", "db.example.com")
		defer os.Unsetenv("DB_HOST")

		cmd := &cobra.Command{}
		cmd.PersistentFlags().String("db-host", "", "DB host")

		got, err := GetString(cmd, "db-host", WithRequired())
		if err != nil {
			t.Fatalf("GetString() error = %v", err)
		}
		if got != "db.example.com" {
			t.Errorf("GetString() = %v, want db.example.com", got)
		}
	})

	t.Run("Required parameter error message contains helpful info", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.PersistentFlags().String("my-param", "", "Test param")

		_, err := GetString(cmd, "my-param", WithRequired())
		if err == nil {
			t.Fatal("expected error")
		}
		errMsg := err.Error()
		if !contains(errMsg, "--my-param") {
			t.Errorf("Error message should contain flag name, got: %s", errMsg)
		}
		if !contains(errMsg, "MY_PARAM") {
			t.Errorf("Error message should contain env var name, got: %s", errMsg)
		}
	})
}

// ==========================================
// 6. Validator Tests
// ==========================================

func TestValidator(t *testing.T) {
	t.Run("Validator passes for valid value", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.PersistentFlags().Int("port", 0, "Server port")
		cmd.PersistentFlags().Set("port", "8080")

		got, err := GetInt(cmd, "port", WithValidator(func(v interface{}) error {
			return nil // always pass
		}))
		if err != nil {
			t.Fatalf("GetInt() error = %v", err)
		}
		if got != 8080 {
			t.Errorf("GetInt() = %v, want 8080", got)
		}
	})

	t.Run("Validator fails for invalid value from command line", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.PersistentFlags().Int("port", 0, "Server port")
		cmd.PersistentFlags().Set("port", "99999")

		_, err := GetInt(cmd, "port", WithValidator(func(v interface{}) error {
			p := v.(int)
			if p > 65535 {
				return &testError{"port too large"}
			}
			return nil
		}))
		if err == nil {
			t.Errorf("GetInt() should return error for invalid port")
		}
	})

	t.Run("Validator fails for invalid value from environment", func(t *testing.T) {
		os.Setenv("PORT", "99999")
		defer os.Unsetenv("PORT")

		cmd := &cobra.Command{}
		cmd.PersistentFlags().Int("port", 0, "Server port")

		_, err := GetInt(cmd, "port", WithValidator(func(v interface{}) error {
			p := v.(int)
			if p > 65535 {
				return &testError{"port too large"}
			}
			return nil
		}))
		if err == nil {
			t.Errorf("GetInt() should return error for invalid port from env")
		}
	})

	t.Run("Validator runs on default value", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.PersistentFlags().Int("port", 0, "Server port")

		_, err := GetInt(cmd, "port",
			WithDefault(99999),
			WithValidator(func(v interface{}) error {
				p := v.(int)
				if p > 65535 {
					return &testError{"port too large"}
				}
				return nil
			}),
		)
		if err == nil {
			t.Errorf("GetInt() should return error when default value fails validation")
		}
	})

	t.Run("Validator runs on dynamic default value", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.PersistentFlags().Int("port", 0, "Server port")

		_, err := GetInt(cmd, "port",
			WithDefaultFunc(func() (interface{}, error) {
				return 99999, nil
			}),
			WithValidator(func(v interface{}) error {
				p := v.(int)
				if p > 65535 {
					return &testError{"port too large"}
				}
				return nil
			}),
		)
		if err == nil {
			t.Errorf("GetInt() should return error when dynamic default fails validation")
		}
	})
}

// testError implements error for test validators
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

// ==========================================
// 7. Duration Parsing Tests
// ==========================================

func TestDurationParsing(t *testing.T) {
	t.Run("Duration from env with standard format '30s'", func(t *testing.T) {
		os.Setenv("TIMEOUT", "30s")
		defer os.Unsetenv("TIMEOUT")

		cmd := &cobra.Command{}
		cmd.PersistentFlags().Duration("timeout", 0, "Timeout")

		got, err := GetDuration(cmd, "timeout", WithEnvKey("TIMEOUT"))
		if err != nil {
			t.Fatalf("GetDuration() error = %v", err)
		}
		if got != 30*time.Second {
			t.Errorf("GetDuration() = %v, want 30s", got)
		}
	})

	t.Run("Duration from env with plain number (seconds)", func(t *testing.T) {
		os.Setenv("TIMEOUT", "60")
		defer os.Unsetenv("TIMEOUT")

		cmd := &cobra.Command{}
		cmd.PersistentFlags().Duration("timeout", 0, "Timeout")

		got, err := GetDuration(cmd, "timeout", WithEnvKey("TIMEOUT"))
		if err != nil {
			t.Fatalf("GetDuration() error = %v", err)
		}
		if got != 60*time.Second {
			t.Errorf("GetDuration() = %v, want 1m0s", got)
		}
	})

	t.Run("Duration from env with invalid format returns error", func(t *testing.T) {
		os.Setenv("TIMEOUT", "invalid")
		defer os.Unsetenv("TIMEOUT")

		cmd := &cobra.Command{}
		cmd.PersistentFlags().Duration("timeout", 0, "Timeout")

		_, err := GetDuration(cmd, "timeout", WithEnvKey("TIMEOUT"))
		if err == nil {
			t.Errorf("GetDuration() should return error for invalid env value")
		}
	})
}

// ==========================================
// 8. String Slice With Separator Tests
// ==========================================

func TestStringSliceSeparator(t *testing.T) {
	t.Run("StringSlice from env with default comma separator", func(t *testing.T) {
		os.Setenv("TAGS", "a,b,c")
		defer os.Unsetenv("TAGS")

		cmd := &cobra.Command{}
		cmd.PersistentFlags().StringSlice("tags", nil, "Tags")

		got, err := GetStringSlice(cmd, "tags", WithEnvKey("TAGS"))
		if err != nil {
			t.Fatalf("GetStringSlice() error = %v", err)
		}
		if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
			t.Errorf("GetStringSlice() = %v, want [a b c]", got)
		}
	})

	t.Run("StringSlice from env with custom separator", func(t *testing.T) {
		os.Setenv("ITEMS", "x|y|z")
		defer os.Unsetenv("ITEMS")

		cmd := &cobra.Command{}
		cmd.PersistentFlags().StringSlice("items", nil, "Items")

		got, err := GetStringSlice(cmd, "items", WithEnvKey("ITEMS"), WithSeparator("|"))
		if err != nil {
			t.Fatalf("GetStringSlice() error = %v", err)
		}
		if len(got) != 3 || got[0] != "x" || got[1] != "y" || got[2] != "z" {
			t.Errorf("GetStringSlice() = %v, want [x y z]", got)
		}
	})

	t.Run("StringSlice from env with empty string", func(t *testing.T) {
		os.Setenv("EMPTY_LIST", "")
		defer os.Unsetenv("EMPTY_LIST")

		cmd := &cobra.Command{}
		cmd.PersistentFlags().StringSlice("empty-list", nil, "Empty list")

		got, err := GetStringSlice(cmd, "empty-list", WithEnvKey("EMPTY_LIST"))
		if err != nil {
			t.Fatalf("GetStringSlice() error = %v", err)
		}
		if len(got) != 0 {
			t.Errorf("GetStringSlice() = %v, want []", got)
		}
	})
}

// ==========================================
// 9. WithEnvKey Custom Env Var Tests
// ==========================================

func TestWithEnvKey(t *testing.T) {
	t.Run("Custom env key overrides automatic derivation", func(t *testing.T) {
		os.Setenv("CUSTOM_DB_HOST", "my-db.example.com")
		defer os.Unsetenv("CUSTOM_DB_HOST")

		cmd := &cobra.Command{}
		cmd.PersistentFlags().String("db-host", "", "DB host")

		got, err := GetString(cmd, "db-host", WithEnvKey("CUSTOM_DB_HOST"))
		if err != nil {
			t.Fatalf("GetString() error = %v", err)
		}
		if got != "my-db.example.com" {
			t.Errorf("GetString() = %v, want my-db.example.com", got)
		}
	})

	t.Run("Automatic env key derivation (app-id -> APP_ID)", func(t *testing.T) {
		os.Setenv("APP_ID", "my-app-123")
		defer os.Unsetenv("APP_ID")

		cmd := &cobra.Command{}
		cmd.PersistentFlags().String("app-id", "", "App ID")

		got, err := GetString(cmd, "app-id")
		if err != nil {
			t.Fatalf("GetString() error = %v", err)
		}
		if got != "my-app-123" {
			t.Errorf("GetString() = %v, want my-app-123", got)
		}
	})
}

// ==========================================
// 10. Dynamic Default Function Error Tests
// ==========================================

func TestDynamicDefaultFuncError(t *testing.T) {
	t.Run("Dynamic default function error should propagate", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.PersistentFlags().String("test-key", "", "Test key")

		_, err := GetString(cmd, "test-key", WithDefaultFunc(func() (interface{}, error) {
			return "", &testError{"db connection failed"}
		}))
		if err == nil {
			t.Errorf("GetString() should return error when default func fails")
		}
	})

	t.Run("Dynamic default function wrong type silently ignored", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.PersistentFlags().Int("test-num", 0, "Test num")

		got, err := GetInt(cmd, "test-num", WithDefaultFunc(func() (interface{}, error) {
			return "not an int", nil // Wrong type
		}))
		if err != nil {
			t.Fatalf("GetInt() error = %v", err)
		}
		if got != 0 {
			t.Errorf("GetInt() = %v, want 0 when default func returns wrong type", got)
		}
	})
}

// ==========================================
// 11. Type Compatibility Tests
// ==========================================

func TestTypeCompatibility(t *testing.T) {
	t.Run("int default for GetInt64 should work", func(t *testing.T) {
		cmd := setupTestCmd()

		got, err := GetInt64(cmd, "test-int64", WithDefault(100))
		if err != nil {
			t.Fatalf("GetInt64() error = %v", err)
		}
		if got != 100 {
			t.Errorf("GetInt64() = %v, want 100", got)
		}
	})

	t.Run("int default for GetDuration should work as nanoseconds", func(t *testing.T) {
		cmd := setupTestCmd()

		got, err := GetDuration(cmd, "test-duration", WithDefault(30))
		if err != nil {
			t.Fatalf("GetDuration() error = %v", err)
		}
		if got != 30*time.Nanosecond {
			t.Errorf("GetDuration() = %v, want 30ns (int default for Duration is nanoseconds in Go)", got)
		}
	})
}

// ==========================================
// 12. Zero Value Detection Tests
// ==========================================

func TestZeroValues(t *testing.T) {
	t.Run("GetInt with zero from default func falls through", func(t *testing.T) {
		cmd := setupTestCmd()

		got, err := GetInt(cmd, "test-int", WithDefaultFunc(func() (interface{}, error) {
			return 0, nil
		}))
		if err != nil {
			t.Fatalf("GetInt() error = %v", err)
		}
		if got != 0 {
			t.Errorf("GetInt() = %v, want 0 (zero from default func is allowed)", got)
		}
	})

	t.Run("Bool true from dynamic default is accepted", func(t *testing.T) {
		cmd := setupTestCmd()

		got, err := GetBool(cmd, "test-bool", WithDefaultFunc(func() (interface{}, error) {
			return true, nil
		}))
		if err != nil {
			t.Fatalf("GetBool() error = %v", err)
		}
		if got != true {
			t.Errorf("GetBool() = %v, want true", got)
		}
	})
}

// ==========================================
// 13. Value From Different Flag Types (local/persistent)
// ==========================================

func TestFlagTypes(t *testing.T) {
	t.Run("Get value from local flag", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.Flags().String("local-flag", "", "Local flag")

		cmd.Flags().Set("local-flag", "from-local")

		got, err := GetString(cmd, "local-flag")
		if err != nil {
			t.Fatalf("GetString() error = %v", err)
		}
		if got != "from-local" {
			t.Errorf("GetString() = %v, want from-local", got)
		}
	})

	t.Run("Get value from persistent flag", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.PersistentFlags().String("persistent-flag", "", "Persistent flag")

		cmd.PersistentFlags().Set("persistent-flag", "from-persistent")

		got, err := GetString(cmd, "persistent-flag")
		if err != nil {
			t.Fatalf("GetString() error = %v", err)
		}
		if got != "from-persistent" {
			t.Errorf("GetString() = %v, want from-persistent", got)
		}
	})
}

// ==========================================
// 14. Required + Validator Combined
// ==========================================

func TestRequiredAndValidator(t *testing.T) {
	t.Run("Required with validator, valid value should pass", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.PersistentFlags().Int("port", 0, "Server port")
		cmd.PersistentFlags().Set("port", "8080")

		got, err := GetInt(cmd, "port",
			WithRequired(),
			WithValidator(func(v interface{}) error {
				p := v.(int)
				if p < 1 || p > 65535 {
					return &testError{"invalid port range"}
				}
				return nil
			}),
		)
		if err != nil {
			t.Fatalf("GetInt() error = %v", err)
		}
		if got != 8080 {
			t.Errorf("GetInt() = %v, want 8080", got)
		}
	})

	t.Run("Required with validator, invalid value should fail", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.PersistentFlags().Int("port", 0, "Server port")
		cmd.PersistentFlags().Set("port", "0")

		_, err := GetInt(cmd, "port",
			WithRequired(),
			WithValidator(func(v interface{}) error {
				p := v.(int)
				if p < 1 {
					return &testError{"port must be positive"}
				}
				return nil
			}),
		)
		if err == nil {
			t.Errorf("GetInt() should return error for port=0 with positive validator")
		}
	})
}

// ==========================================
// 15. Edge Case Tests
// ==========================================

func TestEdgeCases(t *testing.T) {
	t.Run("Multiple options combined: WithEnvKey + WithSeparator", func(t *testing.T) {
		os.Setenv("CUSTOM_ITEMS", "a|b|c")
		defer os.Unsetenv("CUSTOM_ITEMS")

		cmd := &cobra.Command{}
		cmd.PersistentFlags().StringSlice("items", nil, "Items")

		got, err := GetStringSlice(cmd, "items", WithEnvKey("CUSTOM_ITEMS"), WithSeparator("|"))
		if err != nil {
			t.Fatalf("GetStringSlice() error = %v", err)
		}
		if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
			t.Errorf("GetStringSlice() = %v, want [a b c]", got)
		}
	})

	t.Run("Multiple options combined: WithEnvKey + WithDefault + WithValidator", func(t *testing.T) {
		os.Unsetenv("MY_PORT")

		cmd := &cobra.Command{}
		cmd.PersistentFlags().Int("my-port", 0, "Port")

		got, err := GetInt(cmd, "my-port",
			WithEnvKey("MY_PORT"),
			WithDefault(3000),
			WithValidator(func(v interface{}) error {
				p := v.(int)
				if p < 1 || p > 65535 {
					return &testError{"invalid port"}
				}
				return nil
			}),
		)
		if err != nil {
			t.Fatalf("GetInt() error = %v", err)
		}
		if got != 3000 {
			t.Errorf("GetInt() = %v, want 3000", got)
		}
	})

	t.Run("Environment variable with whitespace only", func(t *testing.T) {
		os.Setenv("WHITESPACE_STR", "   ")
		defer os.Unsetenv("WHITESPACE_STR")

		cmd := &cobra.Command{}
		cmd.PersistentFlags().String("whitespace-str", "default", "Test")

		got, err := GetString(cmd, "whitespace-str", WithEnvKey("WHITESPACE_STR"))
		if err != nil {
			t.Fatalf("GetString() error = %v", err)
		}
		if got != "   " {
			t.Errorf("GetString() = %q, want space characters", got)
		}
	})

	t.Run("Bool --flag=false from command line with flag default true", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.PersistentFlags().Bool("enabled", true, "Enabled")

		cmd.PersistentFlags().Set("enabled", "false")

		got, err := GetBool(cmd, "enabled")
		if err != nil {
			t.Fatalf("GetBool() error = %v", err)
		}
		if got != false {
			t.Errorf("GetBool() = %v, want false", got)
		}
	})

	t.Run("Bool from env with 'true' (string)", func(t *testing.T) {
		os.Setenv("BOOL_ENABLED", "true")
		defer os.Unsetenv("BOOL_ENABLED")

		cmd := &cobra.Command{}
		cmd.PersistentFlags().Bool("bool-enabled", false, "Test")

		got, err := GetBool(cmd, "bool-enabled", WithEnvKey("BOOL_ENABLED"))
		if err != nil {
			t.Fatalf("GetBool() error = %v", err)
		}
		if got != true {
			t.Errorf("GetBool() = %v, want true", got)
		}
	})

	t.Run("Int from env with negative value", func(t *testing.T) {
		os.Setenv("NEGATIVE_INT", "-42")
		defer os.Unsetenv("NEGATIVE_INT")

		cmd := &cobra.Command{}
		cmd.PersistentFlags().Int("negative-int", 0, "Test")

		got, err := GetInt(cmd, "negative-int", WithEnvKey("NEGATIVE_INT"))
		if err != nil {
			t.Fatalf("GetInt() error = %v", err)
		}
		if got != -42 {
			t.Errorf("GetInt() = %v, want -42", got)
		}
	})

	t.Run("Float64 from env with integer string", func(t *testing.T) {
		os.Setenv("FLOAT_VAL", "42")
		defer os.Unsetenv("FLOAT_VAL")

		cmd := &cobra.Command{}
		cmd.PersistentFlags().Float64("float-val", 0, "Test")

		got, err := GetFloat64(cmd, "float-val", WithEnvKey("FLOAT_VAL"))
		if err != nil {
			t.Fatalf("GetFloat64() error = %v", err)
		}
		if got != 42.0 {
			t.Errorf("GetFloat64() = %v, want 42.0", got)
		}
	})

	t.Run("Default value wrong type silently ignored (verified by zero value)", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.PersistentFlags().Int("test-int", 0, "Test int")

		got, err := GetInt(cmd, "test-int", WithDefault("not-an-int"))
		if err != nil {
			t.Fatalf("GetInt() error = %v", err)
		}
		if got != 0 {
			t.Errorf("GetInt() = %v, want 0 (wrong type default should be ignored, falling to flag default)", got)
		}
	})

	t.Run("Validator error message contains flag name for command line", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.PersistentFlags().Int("my-port", 0, "Port")
		cmd.PersistentFlags().Set("my-port", "99999")

		_, err := GetInt(cmd, "my-port", WithValidator(func(v interface{}) error {
			return &testError{"invalid value"}
		}))
		if err == nil {
			t.Fatal("expected error")
		}
		errMsg := err.Error()
		if !contains(errMsg, "my-port") {
			t.Errorf("Error message should contain flag name, got: %s", errMsg)
		}
	})

	t.Run("Validator error message contains env key for environment", func(t *testing.T) {
		os.Setenv("MY_ENV_PORT", "99999")
		defer os.Unsetenv("MY_ENV_PORT")

		cmd := &cobra.Command{}
		cmd.PersistentFlags().Int("env-port", 0, "Port")

		_, err := GetInt(cmd, "env-port", WithEnvKey("MY_ENV_PORT"), WithValidator(func(v interface{}) error {
			return &testError{"invalid value"}
		}))
		if err == nil {
			t.Fatal("expected error")
		}
		errMsg := err.Error()
		if !contains(errMsg, "MY_ENV_PORT") {
			t.Errorf("Error message should contain env key, got: %s", errMsg)
		}
	})

	t.Run("WithSeparator combined with WithDefault", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.PersistentFlags().StringSlice("items", nil, "Items")

		got, err := GetStringSlice(cmd, "items",
			WithDefault([]string{"x", "y", "z"}),
			WithSeparator("|"),
		)
		if err != nil {
			t.Fatalf("GetStringSlice() error = %v", err)
		}
		if len(got) != 3 || got[0] != "x" || got[1] != "y" || got[2] != "z" {
			t.Errorf("GetStringSlice() = %v, want [x y z]", got)
		}
	})

	t.Run("Env key automatic derivation with multiple dashes", func(t *testing.T) {
		os.Setenv("MY_APP_DB_HOST", "db-host")
		defer os.Unsetenv("MY_APP_DB_HOST")

		cmd := &cobra.Command{}
		cmd.PersistentFlags().String("my-app-db-host", "", "DB host")

		got, err := GetString(cmd, "my-app-db-host")
		if err != nil {
			t.Fatalf("GetString() error = %v", err)
		}
		if got != "db-host" {
			t.Errorf("GetString() = %v, want db-host", got)
		}
	})

	t.Run("Required parameter with env key override", func(t *testing.T) {
		os.Unsetenv("REQUIRED_KEY")

		cmd := &cobra.Command{}
		cmd.PersistentFlags().String("required-key", "", "Required")

		_, err := GetString(cmd, "required-key", WithRequired(), WithEnvKey("REQUIRED_KEY"))
		if err == nil {
			t.Fatal("expected error for missing required parameter")
		}
		errMsg := err.Error()
		if !contains(errMsg, "--required-key") {
			t.Errorf("Error message should contain --required-key, got: %s", errMsg)
		}
		if !contains(errMsg, "REQUIRED_KEY") {
			t.Errorf("Error message should contain REQUIRED_KEY, got: %s", errMsg)
		}
	})
}

// ==========================================
// Helper functions
// ==========================================

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsStr(s, substr)
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
