// Priority demonstration for param package
// Shows the priority order: command line > environment > static default > dynamic default
package main

import (
	"fmt"
	"os"

	"github.com/kaichao/gopkg/param"
	"github.com/spf13/cobra"
)

func main() {
	// Create a command with port flag
	cmd := &cobra.Command{}
	cmd.Flags().Int("port", 0, "Server port")

	// Test priority order

	// Case 1: Command line has highest priority
	fmt.Println("=== Case 1: Command line value (should be 8080) ===")
	os.Setenv("PORT", "3000")
	defer os.Unsetenv("PORT")

	cmd.Flags().Set("port", "8080")
	port, err := param.GetInt(cmd, "port",
		param.WithEnvKey("PORT"),
		param.WithDefault(80),
		param.WithDefaultFunc(func() (interface{}, error) {
			return 9000, nil
		}),
	)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Port: %d (from command line)\n\n", port)

	// Case 2: Environment variable when command line not set
	fmt.Println("=== Case 2: Environment variable (should be 3000) ===")
	cmd2 := &cobra.Command{}
	cmd2.Flags().Int("port", 0, "Server port")

	port2, err := param.GetInt(cmd2, "port",
		param.WithEnvKey("PORT"),
		param.WithDefault(80),
		param.WithDefaultFunc(func() (interface{}, error) {
			return 9000, nil
		}),
	)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Port: %d (from environment)\n\n", port2)

	// Case 3: Static default when neither command line nor environment set
	fmt.Println("=== Case 3: Static default (should be 80) ===")
	os.Unsetenv("PORT")
	cmd3 := &cobra.Command{}
	cmd3.Flags().Int("port", 0, "Server port")

	port3, err := param.GetInt(cmd3, "port",
		param.WithEnvKey("PORT"),
		param.WithDefault(80),
		param.WithDefaultFunc(func() (interface{}, error) {
			return 9000, nil
		}),
	)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Port: %d (from static default)\n\n", port3)

	// Case 4: Dynamic default function when all else fails
	fmt.Println("=== Case 4: Dynamic default function (should be 9000) ===")
	cmd4 := &cobra.Command{}
	cmd4.Flags().Int("port", 0, "Server port")

	port4, err := param.GetInt(cmd4, "port",
		param.WithEnvKey("PORT"),
		param.WithDefaultFunc(func() (interface{}, error) {
			return 9000, nil
		}),
	)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Port: %d (from dynamic default function)\n", port4)
}
