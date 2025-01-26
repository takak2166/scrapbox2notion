package logger

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestInit(t *testing.T) {
	tests := []struct {
		name        string
		level       string
		expectError bool
	}{
		{
			name:        "Debug level",
			level:       "debug",
			expectError: false,
		},
		{
			name:        "Info level",
			level:       "info",
			expectError: false,
		},
		{
			name:        "Error level",
			level:       "error",
			expectError: false,
		},
		{
			name:        "Invalid level",
			level:       "invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Init(tt.level)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestLogging(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
		DisableColors:    true,
	})

	tests := []struct {
		name          string
		logFunc       func(string, ...map[string]interface{})
		message       string
		fields        map[string]interface{}
		expectedLevel string
	}{
		{
			name:          "Debug message",
			logFunc:       Debug,
			message:       "Debug test",
			expectedLevel: "debug",
		},
		{
			name:          "Info message",
			logFunc:       Info,
			message:       "Info test",
			expectedLevel: "info",
		},
		{
			name:    "Debug with fields",
			logFunc: Debug,
			message: "Debug with fields",
			fields: map[string]interface{}{
				"key": "value",
			},
			expectedLevel: "debug",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			log.SetLevel(logrus.DebugLevel)

			if tt.fields != nil {
				tt.logFunc(tt.message, tt.fields)
			} else {
				tt.logFunc(tt.message)
			}

			output := buf.String()
			if !strings.Contains(output, "level="+tt.expectedLevel) {
				t.Errorf("Expected log level %s, got %s", tt.expectedLevel, output)
			}
			if !strings.Contains(output, tt.message) {
				t.Errorf("Expected message %s, got %s", tt.message, output)
			}
			if tt.fields != nil {
				for k, v := range tt.fields {
					if !strings.Contains(output, k+"="+v.(string)) {
						t.Errorf("Expected field %s=%v in output: %s", k, v, output)
					}
				}
			}
		})
	}
}

func TestError(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
		DisableColors:    true,
	})

	testMessage := "Error test"
	testError := errors.New("test error")
	testFields := map[string]interface{}{
		"key": "value",
	}

	// Test error logging without fields
	Error(testMessage, testError)
	output := buf.String()
	if !strings.Contains(output, "level=error") {
		t.Error("Expected error level")
	}
	if !strings.Contains(output, testMessage) {
		t.Error("Expected error message")
	}
	if !strings.Contains(output, testError.Error()) {
		t.Error("Expected error details")
	}

	// Test error logging with fields
	buf.Reset()
	Error(testMessage, testError, testFields)
	output = buf.String()
	if !strings.Contains(output, "key=value") {
		t.Error("Expected error with fields")
	}
	if !strings.Contains(output, testError.Error()) {
		t.Error("Expected error details with fields")
	}
}
