package param

import (
	"os"
	"testing"
	"time"

	"github.com/kaichao/gopkg/errors"
	"github.com/spf13/cobra"
)

func TestWithDefaultFunc(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.PersistentFlags().String("test-string", "", "Test string flag")
	cmd.PersistentFlags().Int("test-int", 0, "Test int flag")
	cmd.PersistentFlags().Bool("test-bool", false, "Test bool flag")
	cmd.PersistentFlags().Duration("test-duration", 0, "Test duration flag")
	cmd.PersistentFlags().Int64("test-int64", 0, "Test int64 flag")
	cmd.PersistentFlags().Float64("test-float64", 0, "Test float64 flag")
	cmd.PersistentFlags().StringSlice("test-slice", nil, "Test slice flag")

	t.Run("String dynamic default", func(t *testing.T) {
		got, err := GetString(cmd, "test-string", WithDefaultFunc(func() (interface{}, error) {
			return "dynamic-default", nil
		}))
		if err != nil {
			t.Errorf("GetString() error = %v", err)
			return
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
			t.Errorf("GetInt() error = %v", err)
			return
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
			t.Errorf("GetBool() error = %v", err)
			return
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
			t.Errorf("GetDuration() error = %v", err)
			return
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
			t.Errorf("GetInt64() error = %v", err)
			return
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
			t.Errorf("GetFloat64() error = %v", err)
			return
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
			t.Errorf("GetStringSlice() error = %v", err)
			return
		}
		if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
			t.Errorf("GetStringSlice() = %v, want [a b c]", got)
		}
	})

	t.Run("Priority: command line > environment > static default > dynamic default", func(t *testing.T) {
		// Clean up environment after test
		os.Setenv("TEST_INT", "25")
		defer os.Unsetenv("TEST_INT")

		// Set command line flag - use PersistentFlags for consistency with GetInt
		cmd.PersistentFlags().Set("test-int", "30")

		got, err := GetInt(cmd, "test-int",
			WithDefault(20),
			WithDefaultFunc(func() (interface{}, error) {
				return 42, nil
			}),
		)
		if err != nil {
			t.Errorf("GetInt() error = %v", err)
			return
		}
		if got != 30 {
			t.Errorf("GetInt() = %v, want 30 (command line should win)", got)
		}
	})

	t.Run("Environment variable priority", func(t *testing.T) {
		os.Setenv("TEST_INT2", "99")
		defer os.Unsetenv("TEST_INT2")

		got, err := GetInt(cmd, "test-int2",
			WithDefault(50),
			WithDefaultFunc(func() (interface{}, error) {
				return 75, nil
			}),
		)
		if err != nil {
			t.Errorf("GetInt() error = %v", err)
			return
		}
		if got != 99 {
			t.Errorf("GetInt() = %v, want 99 (environment should win)", got)
		}
	})

	t.Run("Static default priority", func(t *testing.T) {
		got, err := GetInt(cmd, "test-int3",
			WithDefault(88),
			WithDefaultFunc(func() (interface{}, error) {
				return 77, nil
			}),
		)
		if err != nil {
			t.Errorf("GetInt() error = %v", err)
			return
		}
		if got != 88 {
			t.Errorf("GetInt() = %v, want 88 (static default should win)", got)
		}
	})

	t.Run("Dynamic default function error", func(t *testing.T) {
		got, err := GetString(cmd, "test-error", WithDefaultFunc(func() (interface{}, error) {
			return "", errors.E("test error")
		}))
		if err != nil {
			// Error from default function should not bubble up
			t.Errorf("GetString() should not return error from default func, got %v", err)
		}
		if got != "" {
			t.Errorf("GetString() = %v, want empty string when default func errors", got)
		}
	})

	t.Run("Dynamic default function wrong type", func(t *testing.T) {
		got, err := GetInt(cmd, "test-wrong-type", WithDefaultFunc(func() (interface{}, error) {
			return "not an int", nil // Wrong type
		}))
		if err != nil {
			t.Errorf("GetInt() error = %v", err)
		}
		if got != 0 {
			t.Errorf("GetInt() = %v, want 0 when default func returns wrong type", got)
		}
	})
}
