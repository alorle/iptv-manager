package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

// LogLevel represents the severity level of a log message
type LogLevel int

// Log level constants define the severity hierarchy for filtering log output
const (
	DEBUG LogLevel = iota // DEBUG is the lowest severity level for detailed diagnostics
	INFO                  // INFO is for general informational messages
	WARN                  // WARN is for warning messages that don't prevent operation
	ERROR                 // ERROR is the highest severity for error conditions
)

func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ParseLogLevel converts a string to a LogLevel
func ParseLogLevel(s string) LogLevel {
	switch strings.ToUpper(s) {
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARN":
		return WARN
	case "ERROR":
		return ERROR
	default:
		return INFO
	}
}

// Logger provides structured logging with configurable levels
type Logger struct {
	mu     sync.Mutex
	level  LogLevel
	logger *log.Logger
	prefix string
}

// New creates a new Logger with the specified level
func New(level LogLevel, prefix string) *Logger {
	return &Logger{
		level:  level,
		logger: log.New(os.Stderr, "", log.LstdFlags),
		prefix: prefix,
	}
}

// NewWithWriter creates a new Logger with custom output writer
func NewWithWriter(level LogLevel, prefix string, w io.Writer) *Logger {
	return &Logger{
		level:  level,
		logger: log.New(w, "", log.LstdFlags),
		prefix: prefix,
	}
}

// SetLevel changes the log level
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// GetLevel returns the current log level
func (l *Logger) GetLevel() LogLevel {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.level
}

// shouldLog checks if a message at the given level should be logged
func (l *Logger) shouldLog(level LogLevel) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return level >= l.level
}

// log writes a log message with the given level and fields
func (l *Logger) log(level LogLevel, msg string, fields map[string]interface{}) {
	if !l.shouldLog(level) {
		return
	}

	// Build structured log message
	var sb strings.Builder

	// Add prefix if set
	if l.prefix != "" {
		sb.WriteString(l.prefix)
		sb.WriteString(" ")
	}

	// Add level
	sb.WriteString(level.String())
	sb.WriteString(": ")

	// Add message
	sb.WriteString(msg)

	// Add fields
	if len(fields) > 0 {
		sb.WriteString(" |")
		for k, v := range fields {
			sb.WriteString(fmt.Sprintf(" %s=%v", k, v))
		}
	}

	l.logger.Println(sb.String())
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, fields map[string]interface{}) {
	l.log(DEBUG, msg, fields)
}

// Info logs an info message
func (l *Logger) Info(msg string, fields map[string]interface{}) {
	l.log(INFO, msg, fields)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, fields map[string]interface{}) {
	l.log(WARN, msg, fields)
}

// Error logs an error message
func (l *Logger) Error(msg string, fields map[string]interface{}) {
	l.log(ERROR, msg, fields)
}

// ResilienceEvent represents a type of resilience-related event
type ResilienceEvent string

// Resilience event constants identify specific types of recovery and failure events
const (
	EventReconnectAttempt     ResilienceEvent = "reconnect_attempt"      // EventReconnectAttempt indicates a reconnection is being attempted
	EventReconnectSuccess     ResilienceEvent = "reconnect_success"      // EventReconnectSuccess indicates successful reconnection
	EventReconnectFailed      ResilienceEvent = "reconnect_failed"       // EventReconnectFailed indicates reconnection failure
	EventCircuitBreakerChange ResilienceEvent = "circuit_breaker_change" // EventCircuitBreakerChange indicates circuit breaker state transition
	EventHealthCheckFailed    ResilienceEvent = "health_check_failed"    // EventHealthCheckFailed indicates health check failure
)

// LogReconnectAttempt logs a reconnection attempt (INFO level)
func (l *Logger) LogReconnectAttempt(contentID string, attempt int, backoff time.Duration) {
	l.Info("Reconnection attempt", map[string]interface{}{
		"event":     EventReconnectAttempt,
		"contentID": contentID,
		"attempt":   attempt,
		"backoff":   backoff.String(),
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// LogReconnectSuccess logs a successful reconnection (INFO level)
func (l *Logger) LogReconnectSuccess(contentID string, downtime time.Duration) {
	l.Info("Reconnection successful", map[string]interface{}{
		"event":     EventReconnectSuccess,
		"contentID": contentID,
		"downtime":  downtime.String(),
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// LogReconnectFailed logs a definitively failed reconnection (ERROR level)
func (l *Logger) LogReconnectFailed(contentID string, reason string, attempts int) {
	l.Error("Reconnection failed", map[string]interface{}{
		"event":     EventReconnectFailed,
		"contentID": contentID,
		"reason":    reason,
		"attempts":  attempts,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// LogCircuitBreakerChange logs a circuit breaker state change (WARN level)
func (l *Logger) LogCircuitBreakerChange(oldState, newState string, contentID string) {
	fields := map[string]interface{}{
		"event":     EventCircuitBreakerChange,
		"oldState":  oldState,
		"newState":  newState,
		"timestamp": time.Now().Format(time.RFC3339),
	}
	if contentID != "" {
		fields["contentID"] = contentID
	}
	l.Warn("Circuit breaker state changed", fields)
}

// LogHealthCheckFailed logs a failed health check (WARN level)
func (l *Logger) LogHealthCheckFailed(err error) {
	l.Warn("Health check failed", map[string]interface{}{
		"event":     EventHealthCheckFailed,
		"error":     err.Error(),
		"timestamp": time.Now().Format(time.RFC3339),
	})
}
