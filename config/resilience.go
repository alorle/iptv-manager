package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// ResilienceConfig centralizes all resilience-related configuration
type ResilienceConfig struct {
	// Reconnection settings
	ReconnectBufferSize     int           // Buffer size for reconnection data
	ReconnectMaxBackoff     time.Duration // Maximum backoff duration between retries
	ReconnectInitialBackoff time.Duration // Initial backoff duration

	// Circuit breaker settings
	CBFailureThreshold int           // Number of failures before opening circuit
	CBTimeout          time.Duration // Timeout before attempting to close circuit
	CBHalfOpenRequests int           // Number of requests allowed in half-open state

	// Health check settings
	HealthCheckInterval time.Duration // Interval between health checks

	// Logging settings
	LogLevel string // Log level: DEBUG, INFO, WARN, ERROR
}

// DefaultResilienceConfig returns a ResilienceConfig with sensible defaults
func DefaultResilienceConfig() *ResilienceConfig {
	return &ResilienceConfig{
		// Reconnection defaults
		ReconnectBufferSize:     2 * 1024 * 1024, // 2MB
		ReconnectMaxBackoff:     30 * time.Second,
		ReconnectInitialBackoff: 500 * time.Millisecond,

		// Circuit breaker defaults
		CBFailureThreshold: 5,
		CBTimeout:          30 * time.Second,
		CBHalfOpenRequests: 1,

		// Health check defaults
		HealthCheckInterval: 30 * time.Second,

		// Logging defaults
		LogLevel: "INFO",
	}
}

// LoadFromEnv loads resilience configuration from environment variables
// and returns an error if any value is invalid
func LoadFromEnv() (*ResilienceConfig, error) {
	cfg := DefaultResilienceConfig()
	var errors []string

	// Parse RECONNECT_BUFFER_SIZE
	if val := os.Getenv("RECONNECT_BUFFER_SIZE"); val != "" {
		size, err := parseByteSize(val)
		if err != nil {
			errors = append(errors, fmt.Sprintf("RECONNECT_BUFFER_SIZE: %v", err))
		} else if size <= 0 {
			errors = append(errors, "RECONNECT_BUFFER_SIZE must be positive")
		} else {
			cfg.ReconnectBufferSize = size
		}
	}

	// Parse RECONNECT_MAX_BACKOFF
	if val := os.Getenv("RECONNECT_MAX_BACKOFF"); val != "" {
		duration, err := time.ParseDuration(val)
		if err != nil {
			errors = append(errors, fmt.Sprintf("RECONNECT_MAX_BACKOFF: invalid duration format (use '30s', '1m', etc.)"))
		} else if duration <= 0 {
			errors = append(errors, "RECONNECT_MAX_BACKOFF must be positive")
		} else {
			cfg.ReconnectMaxBackoff = duration
		}
	}

	// Parse RECONNECT_INITIAL_BACKOFF
	if val := os.Getenv("RECONNECT_INITIAL_BACKOFF"); val != "" {
		duration, err := time.ParseDuration(val)
		if err != nil {
			errors = append(errors, fmt.Sprintf("RECONNECT_INITIAL_BACKOFF: invalid duration format (use '500ms', '1s', etc.)"))
		} else if duration <= 0 {
			errors = append(errors, "RECONNECT_INITIAL_BACKOFF must be positive")
		} else {
			cfg.ReconnectInitialBackoff = duration
		}
	}

	// Parse CB_FAILURE_THRESHOLD
	if val := os.Getenv("CB_FAILURE_THRESHOLD"); val != "" {
		threshold, err := strconv.Atoi(val)
		if err != nil {
			errors = append(errors, fmt.Sprintf("CB_FAILURE_THRESHOLD: must be a valid integer"))
		} else if threshold <= 0 {
			errors = append(errors, "CB_FAILURE_THRESHOLD must be positive")
		} else {
			cfg.CBFailureThreshold = threshold
		}
	}

	// Parse CB_TIMEOUT
	if val := os.Getenv("CB_TIMEOUT"); val != "" {
		duration, err := time.ParseDuration(val)
		if err != nil {
			errors = append(errors, fmt.Sprintf("CB_TIMEOUT: invalid duration format (use '30s', '1m', etc.)"))
		} else if duration <= 0 {
			errors = append(errors, "CB_TIMEOUT must be positive")
		} else {
			cfg.CBTimeout = duration
		}
	}

	// Parse CB_HALF_OPEN_REQUESTS
	if val := os.Getenv("CB_HALF_OPEN_REQUESTS"); val != "" {
		requests, err := strconv.Atoi(val)
		if err != nil {
			errors = append(errors, fmt.Sprintf("CB_HALF_OPEN_REQUESTS: must be a valid integer"))
		} else if requests <= 0 {
			errors = append(errors, "CB_HALF_OPEN_REQUESTS must be positive")
		} else {
			cfg.CBHalfOpenRequests = requests
		}
	}

	// Parse HEALTH_CHECK_INTERVAL
	if val := os.Getenv("HEALTH_CHECK_INTERVAL"); val != "" {
		duration, err := time.ParseDuration(val)
		if err != nil {
			errors = append(errors, fmt.Sprintf("HEALTH_CHECK_INTERVAL: invalid duration format (use '30s', '1m', etc.)"))
		} else if duration <= 0 {
			errors = append(errors, "HEALTH_CHECK_INTERVAL must be positive")
		} else {
			cfg.HealthCheckInterval = duration
		}
	}

	// Parse LOG_LEVEL
	if val := os.Getenv("LOG_LEVEL"); val != "" {
		level := strings.ToUpper(val)
		validLevels := map[string]bool{
			"DEBUG": true,
			"INFO":  true,
			"WARN":  true,
			"ERROR": true,
		}
		if !validLevels[level] {
			errors = append(errors, "LOG_LEVEL must be one of: DEBUG, INFO, WARN, ERROR")
		} else {
			cfg.LogLevel = level
		}
	}

	// Validate relationships between values
	if cfg.ReconnectInitialBackoff > cfg.ReconnectMaxBackoff {
		errors = append(errors, "RECONNECT_INITIAL_BACKOFF must be less than or equal to RECONNECT_MAX_BACKOFF")
	}

	if len(errors) > 0 {
		return nil, fmt.Errorf("configuration validation failed:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return cfg, nil
}

// Validate performs additional validation on the configuration
func (c *ResilienceConfig) Validate() error {
	var errors []string

	if c.ReconnectBufferSize <= 0 {
		errors = append(errors, "ReconnectBufferSize must be positive")
	}

	if c.ReconnectMaxBackoff <= 0 {
		errors = append(errors, "ReconnectMaxBackoff must be positive")
	}

	if c.ReconnectInitialBackoff <= 0 {
		errors = append(errors, "ReconnectInitialBackoff must be positive")
	}

	if c.ReconnectInitialBackoff > c.ReconnectMaxBackoff {
		errors = append(errors, "ReconnectInitialBackoff must be <= ReconnectMaxBackoff")
	}

	if c.CBFailureThreshold <= 0 {
		errors = append(errors, "CBFailureThreshold must be positive")
	}

	if c.CBTimeout <= 0 {
		errors = append(errors, "CBTimeout must be positive")
	}

	if c.CBHalfOpenRequests <= 0 {
		errors = append(errors, "CBHalfOpenRequests must be positive")
	}

	if c.HealthCheckInterval <= 0 {
		errors = append(errors, "HealthCheckInterval must be positive")
	}

	validLevels := map[string]bool{
		"DEBUG": true,
		"INFO":  true,
		"WARN":  true,
		"ERROR": true,
	}
	if !validLevels[c.LogLevel] {
		errors = append(errors, "LogLevel must be one of: DEBUG, INFO, WARN, ERROR")
	}

	if len(errors) > 0 {
		return fmt.Errorf("invalid configuration:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil
}

// parseByteSize parses a byte size string (e.g., "2MB", "1024", "1.5GB")
// Supports: bytes (no suffix), KB, MB, GB
func parseByteSize(s string) (int, error) {
	s = strings.TrimSpace(strings.ToUpper(s))

	// Try to parse as plain integer first
	if val, err := strconv.Atoi(s); err == nil {
		return val, nil
	}

	// Parse with suffix - check longer suffixes first to avoid "B" matching "MB"
	suffixes := []struct {
		suffix     string
		multiplier int
	}{
		{"GB", 1024 * 1024 * 1024},
		{"MB", 1024 * 1024},
		{"KB", 1024},
		{"B", 1},
	}

	for _, item := range suffixes {
		if strings.HasSuffix(s, item.suffix) {
			numStr := strings.TrimSuffix(s, item.suffix)
			numStr = strings.TrimSpace(numStr)

			// Try integer first
			if val, err := strconv.Atoi(numStr); err == nil {
				if val < 0 {
					return 0, fmt.Errorf("negative values are not allowed")
				}
				return val * item.multiplier, nil
			}

			// Try float
			if val, err := strconv.ParseFloat(numStr, 64); err == nil {
				if val < 0 {
					return 0, fmt.Errorf("negative values are not allowed")
				}
				return int(val * float64(item.multiplier)), nil
			}

			return 0, fmt.Errorf("invalid numeric value: %s", numStr)
		}
	}

	return 0, fmt.Errorf("invalid byte size format (use '2MB', '1024', '1.5GB', etc.)")
}
