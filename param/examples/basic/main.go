// Basic usage examples for param package
package main

import (
	"fmt"

	"github.com/kaichao/gopkg/param"
	"github.com/spf13/cobra"
)

func main() {
	// Create a command with flags
	cmd := &cobra.Command{}
	cmd.Flags().String("name", "", "Your name")
	cmd.Flags().Int("age", 0, "Your age")
	cmd.Flags().Bool("verbose", false, "Enable verbose output")
	cmd.Flags().Duration("timeout", 0, "Operation timeout")
	cmd.Flags().StringSlice("tags", nil, "List of tags")

	// Simulate command line arguments
	cmd.Flags().Set("name", "Alice")
	cmd.Flags().Set("age", "30")
	cmd.Flags().Set("verbose", "true")
	cmd.Flags().Set("timeout", "5s")
	cmd.Flags().Set("tags", "go,cli,example")

	// Get parameters with basic usage
	name, err := param.GetString(cmd, "name")
	if err != nil {
		panic(err)
	}

	age, err := param.GetInt(cmd, "age")
	if err != nil {
		panic(err)
	}

	verbose, err := param.GetBool(cmd, "verbose")
	if err != nil {
		panic(err)
	}

	timeout, err := param.GetDuration(cmd, "timeout")
	if err != nil {
		panic(err)
	}

	tags, err := param.GetStringSlice(cmd, "tags")
	if err != nil {
		panic(err)
	}

	fmt.Printf("Name: %s\n", name)
	fmt.Printf("Age: %d\n", age)
	fmt.Printf("Verbose: %v\n", verbose)
	fmt.Printf("Timeout: %v\n", timeout)
	fmt.Printf("Tags: %v\n", tags)
}
