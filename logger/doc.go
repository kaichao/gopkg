// Package logger provides structured logging with error tracing, async output, and sensitive data filtering.
//
// Key features:
// - Structured logging (JSON/text formats)
// - Error tracing with gopkg/errors integration
// - Async logging with buffering
// - Log rotation (size/time based)
// - Sensitive data filtering
// - Environment-based configuration
//
// Basic usage:
//
//	// Create logger
//	cfg := &logger.Config{
//		Level:  "info",
//		Format: "json",
//		Output: "stdout",
//	}
//	log := logger.NewOrMust(cfg)
//	defer log.Close()
//
//	// Log with fields
//	log.WithField("user_id", 123).Info("User logged in")
//
//	// Log error
//	err := errors.New("database error")
//	logger.LogError(err, logrus.NewEntry(log))
//
// For examples, see the examples/ directory.
package logger
