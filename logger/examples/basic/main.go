// Basic logging examples
package main

import (
	"errors"

	"github.com/kaichao/gopkg/logger"
)

func main() {
	// Create logger with default configuration
	cfg := &logger.Config{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	}

	log := logger.NewOrMust(cfg)
	defer log.Close()

	// Basic logging
	log.Info("Application started")

	// Log with fields
	log.WithField("user_id", 123).Info("User logged in")

	// Log error
	err := errors.New("database connection failed")
	log.WithError(err).Error("Database operation failed")

	// Multiple fields
	log.WithFields(map[string]interface{}{
		"request_id": "abc-123",
		"method":     "GET",
		"path":       "/api/users",
		"status":     200,
	}).Info("Request completed")

	// Format logging
	log.Infof("Processing %d items", 42)
	log.Errorf("Failed to process request: %v", err)
}
