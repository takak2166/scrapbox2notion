package logger

import (
	"github.com/sirupsen/logrus"
)

var log = logrus.New()

// Init initializes the logger with the specified level
func Init(level string) error {
	// Set formatter
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// Parse and set log level
	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		return err
	}
	log.SetLevel(lvl)

	return nil
}

// Debug logs a debug message
func Debug(msg string, fields ...map[string]interface{}) {
	if len(fields) > 0 {
		log.WithFields(fields[0]).Debug(msg)
	} else {
		log.Debug(msg)
	}
}

// Info logs an info message
func Info(msg string, fields ...map[string]interface{}) {
	if len(fields) > 0 {
		log.WithFields(fields[0]).Info(msg)
	} else {
		log.Info(msg)
	}
}

// Error logs an error message
func Error(msg string, err error, fields ...map[string]interface{}) {
	if len(fields) > 0 {
		log.WithFields(fields[0]).WithError(err).Error(msg)
	} else {
		log.WithError(err).Error(msg)
	}
}
