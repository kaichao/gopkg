package exec

// No init() function is defined here to avoid global side effects.
//
// The exec package no longer modifies logrus global configuration on import.
// If you need to set log level or caller reporting, configure logrus explicitly:
//
//	import "github.com/sirupsen/logrus"
//
//	logrus.SetLevel(logrus.DebugLevel)
//	logrus.SetReportCaller(true)
