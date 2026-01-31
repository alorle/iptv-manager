package logging

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"
)

// testError is a simple error implementation for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected LogLevel
	}{
		{"DEBUG", DEBUG},
		{"debug", DEBUG},
		{"INFO", INFO},
		{"info", INFO},
		{"WARN", WARN},
		{"warn", WARN},
		{"ERROR", ERROR},
		{"error", ERROR},
		{"invalid", INFO}, // default to INFO
		{"", INFO},        // default to INFO
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseLogLevel(tt.input)
			if result != tt.expected {
				t.Errorf("ParseLogLevel(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestLogLevelString(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{DEBUG, "DEBUG"},
		{INFO, "INFO"},
		{WARN, "WARN"},
		{ERROR, "ERROR"},
		{LogLevel(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.level.String()
			if result != tt.expected {
				t.Errorf("LogLevel(%d).String() = %q, want %q", tt.level, result, tt.expected)
			}
		})
	}
}

func TestLoggerSetGetLevel(t *testing.T) {
	logger := New(INFO, "test")

	if logger.GetLevel() != INFO {
		t.Errorf("Initial level = %v, want %v", logger.GetLevel(), INFO)
	}

	logger.SetLevel(DEBUG)
	if logger.GetLevel() != DEBUG {
		t.Errorf("After SetLevel(DEBUG), level = %v, want %v", logger.GetLevel(), DEBUG)
	}

	logger.SetLevel(ERROR)
	if logger.GetLevel() != ERROR {
		t.Errorf("After SetLevel(ERROR), level = %v, want %v", logger.GetLevel(), ERROR)
	}
}

func TestLoggerFiltering(t *testing.T) {
	tests := []struct {
		name         string
		logLevel     LogLevel
		logFunc      func(*Logger, *bytes.Buffer)
		shouldAppear bool
	}{
		{
			name:     "DEBUG message with DEBUG level",
			logLevel: DEBUG,
			logFunc: func(l *Logger, buf *bytes.Buffer) {
				l.Debug("test", nil)
			},
			shouldAppear: true,
		},
		{
			name:     "DEBUG message with INFO level",
			logLevel: INFO,
			logFunc: func(l *Logger, buf *bytes.Buffer) {
				l.Debug("test", nil)
			},
			shouldAppear: false,
		},
		{
			name:     "INFO message with INFO level",
			logLevel: INFO,
			logFunc: func(l *Logger, buf *bytes.Buffer) {
				l.Info("test", nil)
			},
			shouldAppear: true,
		},
		{
			name:     "INFO message with WARN level",
			logLevel: WARN,
			logFunc: func(l *Logger, buf *bytes.Buffer) {
				l.Info("test", nil)
			},
			shouldAppear: false,
		},
		{
			name:     "WARN message with WARN level",
			logLevel: WARN,
			logFunc: func(l *Logger, buf *bytes.Buffer) {
				l.Warn("test", nil)
			},
			shouldAppear: true,
		},
		{
			name:     "WARN message with ERROR level",
			logLevel: ERROR,
			logFunc: func(l *Logger, buf *bytes.Buffer) {
				l.Warn("test", nil)
			},
			shouldAppear: false,
		},
		{
			name:     "ERROR message with ERROR level",
			logLevel: ERROR,
			logFunc: func(l *Logger, buf *bytes.Buffer) {
				l.Error("test", nil)
			},
			shouldAppear: true,
		},
		{
			name:     "ERROR message with DEBUG level",
			logLevel: DEBUG,
			logFunc: func(l *Logger, buf *bytes.Buffer) {
				l.Error("test", nil)
			},
			shouldAppear: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := NewWithWriter(tt.logLevel, "", buf)

			tt.logFunc(logger, buf)

			output := buf.String()
			hasOutput := len(output) > 0

			if hasOutput != tt.shouldAppear {
				t.Errorf("Log output presence = %v, want %v. Output: %q", hasOutput, tt.shouldAppear, output)
			}
		})
	}
}

func TestLoggerPrefix(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewWithWriter(INFO, "[test-prefix]", buf)

	logger.Info("test message", nil)

	output := buf.String()
	if !strings.Contains(output, "[test-prefix]") {
		t.Errorf("Output missing prefix: %q", output)
	}
}

func TestLoggerFields(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewWithWriter(INFO, "", buf)

	fields := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
		"key3": true,
	}

	logger.Info("test message", fields)

	output := buf.String()

	if !strings.Contains(output, "test message") {
		t.Errorf("Output missing message: %q", output)
	}

	// Check that all fields appear in output
	for k := range fields {
		expected := k + "="
		if !strings.Contains(output, expected) {
			t.Errorf("Output missing field %q: %q", k, output)
		}
	}
}

func TestLogReconnectAttempt(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewWithWriter(INFO, "", buf)

	logger.LogReconnectAttempt("test-content-123", 3, 2*time.Second)

	output := buf.String()

	// Verify log level is INFO
	if !strings.Contains(output, "INFO") {
		t.Errorf("Output missing INFO level: %q", output)
	}

	// Verify message
	if !strings.Contains(output, "Reconnection attempt") {
		t.Errorf("Output missing message: %q", output)
	}

	// Verify fields
	expectedFields := []string{
		"event=reconnect_attempt",
		"contentID=test-content-123",
		"attempt=3",
		"backoff=2s",
		"timestamp=",
	}

	for _, field := range expectedFields {
		if !strings.Contains(output, field) {
			t.Errorf("Output missing field %q: %q", field, output)
		}
	}
}

func TestLogReconnectSuccess(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewWithWriter(INFO, "", buf)

	logger.LogReconnectSuccess("test-content-123", 5*time.Second)

	output := buf.String()

	// Verify log level is INFO
	if !strings.Contains(output, "INFO") {
		t.Errorf("Output missing INFO level: %q", output)
	}

	// Verify message
	if !strings.Contains(output, "Reconnection successful") {
		t.Errorf("Output missing message: %q", output)
	}

	// Verify fields
	expectedFields := []string{
		"event=reconnect_success",
		"contentID=test-content-123",
		"downtime=5s",
		"timestamp=",
	}

	for _, field := range expectedFields {
		if !strings.Contains(output, field) {
			t.Errorf("Output missing field %q: %q", field, output)
		}
	}
}

func TestLogReconnectFailed(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewWithWriter(INFO, "", buf)

	logger.LogReconnectFailed("test-content-123", "max retries exceeded", 10)

	output := buf.String()

	// Verify log level is ERROR
	if !strings.Contains(output, "ERROR") {
		t.Errorf("Output missing ERROR level: %q", output)
	}

	// Verify message
	if !strings.Contains(output, "Reconnection failed") {
		t.Errorf("Output missing message: %q", output)
	}

	// Verify fields
	expectedFields := []string{
		"event=reconnect_failed",
		"contentID=test-content-123",
		"reason=max retries exceeded",
		"attempts=10",
		"timestamp=",
	}

	for _, field := range expectedFields {
		if !strings.Contains(output, field) {
			t.Errorf("Output missing field %q: %q", field, output)
		}
	}
}

func TestLogCircuitBreakerChange(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewWithWriter(INFO, "", buf)

	logger.LogCircuitBreakerChange("CLOSED", "OPEN", "test-content-123")

	output := buf.String()

	// Verify log level is WARN
	if !strings.Contains(output, "WARN") {
		t.Errorf("Output missing WARN level: %q", output)
	}

	// Verify message
	if !strings.Contains(output, "Circuit breaker state changed") {
		t.Errorf("Output missing message: %q", output)
	}

	// Verify fields
	expectedFields := []string{
		"event=circuit_breaker_change",
		"oldState=CLOSED",
		"newState=OPEN",
		"contentID=test-content-123",
		"timestamp=",
	}

	for _, field := range expectedFields {
		if !strings.Contains(output, field) {
			t.Errorf("Output missing field %q: %q", field, output)
		}
	}
}

func TestLogCircuitBreakerChangeWithoutContentID(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewWithWriter(INFO, "", buf)

	logger.LogCircuitBreakerChange("HALF-OPEN", "CLOSED", "")

	output := buf.String()

	// Verify log level is WARN
	if !strings.Contains(output, "WARN") {
		t.Errorf("Output missing WARN level: %q", output)
	}

	// Verify fields
	expectedFields := []string{
		"event=circuit_breaker_change",
		"oldState=HALF-OPEN",
		"newState=CLOSED",
		"timestamp=",
	}

	for _, field := range expectedFields {
		if !strings.Contains(output, field) {
			t.Errorf("Output missing field %q: %q", field, output)
		}
	}

	// Verify contentID is NOT present
	if strings.Contains(output, "contentID=") {
		t.Errorf("Output should not contain contentID when empty: %q", output)
	}
}

func TestLogHealthCheckFailed(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewWithWriter(INFO, "", buf)

	testErr := &testError{msg: "connection timeout"}
	logger.LogHealthCheckFailed(testErr)

	output := buf.String()

	// Verify log level is WARN
	if !strings.Contains(output, "WARN") {
		t.Errorf("Output missing WARN level: %q", output)
	}

	// Verify message
	if !strings.Contains(output, "Health check failed") {
		t.Errorf("Output missing message: %q", output)
	}

	// Verify fields
	expectedFields := []string{
		"event=health_check_failed",
		"error=",
		"timestamp=",
	}

	for _, field := range expectedFields {
		if !strings.Contains(output, field) {
			t.Errorf("Output missing field %q: %q", field, output)
		}
	}
}

func TestLogLevelFiltering(t *testing.T) {
	tests := []struct {
		name      string
		level     LogLevel
		logFunc   func(*Logger)
		shouldLog bool
	}{
		{
			name:  "ReconnectAttempt with INFO level",
			level: INFO,
			logFunc: func(l *Logger) {
				l.LogReconnectAttempt("test", 1, time.Second)
			},
			shouldLog: true,
		},
		{
			name:  "ReconnectAttempt with WARN level",
			level: WARN,
			logFunc: func(l *Logger) {
				l.LogReconnectAttempt("test", 1, time.Second)
			},
			shouldLog: false,
		},
		{
			name:  "ReconnectFailed with ERROR level",
			level: ERROR,
			logFunc: func(l *Logger) {
				l.LogReconnectFailed("test", "reason", 5)
			},
			shouldLog: true,
		},
		{
			name:  "CircuitBreakerChange with WARN level",
			level: WARN,
			logFunc: func(l *Logger) {
				l.LogCircuitBreakerChange("CLOSED", "OPEN", "test")
			},
			shouldLog: true,
		},
		{
			name:  "CircuitBreakerChange with ERROR level",
			level: ERROR,
			logFunc: func(l *Logger) {
				l.LogCircuitBreakerChange("CLOSED", "OPEN", "test")
			},
			shouldLog: false,
		},
		{
			name:  "HealthCheckFailed with WARN level",
			level: WARN,
			logFunc: func(l *Logger) {
				l.LogHealthCheckFailed(fmt.Errorf("test error"))
			},
			shouldLog: true,
		},
		{
			name:  "HealthCheckFailed with ERROR level",
			level: ERROR,
			logFunc: func(l *Logger) {
				l.LogHealthCheckFailed(fmt.Errorf("test error"))
			},
			shouldLog: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := NewWithWriter(tt.level, "", buf)

			tt.logFunc(logger)

			output := buf.String()
			hasOutput := len(output) > 0

			if hasOutput != tt.shouldLog {
				t.Errorf("Log output presence = %v, want %v. Output: %q", hasOutput, tt.shouldLog, output)
			}
		})
	}
}
