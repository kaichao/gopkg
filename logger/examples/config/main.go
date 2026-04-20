// Logger configuration examples
package main

import (
	"fmt"
	"os"

	"github.com/kaichao/gopkg/logger"
)

func main() {
	fmt.Println("=== Logger Configuration Examples ===")

	// 1. Basic configuration
	fmt.Println("\n1. Basic Configuration:")
	cfg1 := &logger.Config{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	}
	log1, err := logger.NewLogger(cfg1)
	if err != nil {
		fmt.Printf("Failed to create logger: %v\n", err)
		return
	}
	log1.Info("Logger created with basic config")
	log1.Close()

	// 2. File output with rotation
	fmt.Println("\n2. File Output with Rotation:")
	cfg2 := &logger.Config{
		Level:      "debug",
		Format:     "text",
		Output:     "file",
		FilePath:   "test.log",
		MaxSize:    10, // 10MB
		MaxAge:     7,  // 7 days
		MaxBackups: 5,
	}
	log2 := logger.NewOrMust(cfg2)
	defer log2.Close()
	log2.Debug("Debug message to file")
	log2.Info("Info message to file")

	// 3. Async logging
	fmt.Println("\n3. Async Logging:")
	cfg3 := &logger.Config{
		Level:        "info",
		Format:       "json",
		Output:       "stdout",
		AsyncEnabled: true,
		BufferSize:   1000,
	}
	log3 := logger.NewOrMust(cfg3)
	defer log3.Close()
	for i := 0; i < 10; i++ {
		log3.WithField("iteration", i).Info("Async log message")
	}

	// 4. Environment-based configuration
	fmt.Println("\n4. Environment Variables (simulated):")
	// In real usage, set these env vars:
	// LOG_LEVEL=debug LOG_FORMAT=text LOG_OUTPUT=stdout
	cfg4 := logger.LoadConfig()
	log4 := logger.NewOrMust(cfg4)
	defer log4.Close()
	log4.Info("Logger configured from environment")

	// 5. Global logger
	fmt.Println("\n5. Global Logger:")
	logger.InitGlobal(cfg1)
	globalLog := logger.Global()
	globalLog.Info("Using global logger")

	fmt.Println("\nAll configuration examples completed.")

	// Clean up
	os.Remove("test.log")
}
