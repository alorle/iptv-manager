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

// envParser is a helper for parsing environment variables with validation
type envParser struct {
	errors []string
}

// parseDuration parses a duration environment variable, ensuring it's positive
func (p *envParser) parseDuration(envName string, target *time.Duration) {
	val := os.Getenv(envName)
	if val == "" {
		return
	}

	duration, err := time.ParseDuration(val)
	if err != nil {
		p.errors = append(p.errors, fmt.Sprintf("%s: invalid duration format (use '30s', '1m', etc.)", envName))
		return
	}

	if duration <= 0 {
		p.errors = append(p.errors, fmt.Sprintf("%s must be positive", envName))
		return
	}

	*target = duration
}

// parseInt parses an integer environment variable, ensuring it's positive
func (p *envParser) parseInt(envName string, target *int) {
	val := os.Getenv(envName)
	if val == "" {
		return
	}

	intVal, err := strconv.Atoi(val)
	if err != nil {
		p.errors = append(p.errors, fmt.Sprintf("%s: must be a valid integer", envName))
		return
	}

	if intVal <= 0 {
		p.errors = append(p.errors, fmt.Sprintf("%s must be positive", envName))
		return
	}

	*target = intVal
}

// parseByteSize parses a byte size environment variable, ensuring it's positive
func (p *envParser) parseByteSize(envName string, target *int) {
	val := os.Getenv(envName)
	if val == "" {
		return
	}

	size, err := parseByteSize(val)
	if err != nil {
		p.errors = append(p.errors, fmt.Sprintf("%s: %v", envName, err))
		return
	}

	if size <= 0 {
		p.errors = append(p.errors, fmt.Sprintf("%s must be positive", envName))
		return
	}

	*target = size
}

// parseEnum parses an enum environment variable from a set of valid values
func (p *envParser) parseEnum(envName string, target *string, validValues map[string]bool) {
	val := os.Getenv(envName)
	if val == "" {
		return
	}

	normalized := strings.ToUpper(val)
	if !validValues[normalized] {
		// Build list of valid values for error message
		var validList []string
		for k := range validValues {
			validList = append(validList, k)
		}
		p.errors = append(p.errors, fmt.Sprintf("%s must be one of: %s", envName, strings.Join(validList, ", ")))
		return
	}

	*target = normalized
}

// LoadFromEnv loads resilience configuration from environment variables
// and returns an error if any value is invalid
func LoadFromEnv() (*ResilienceConfig, error) {
	cfg := DefaultResilienceConfig()
	parser := &envParser{}

	// Parse all environment variables
	parser.parseByteSize("RECONNECT_BUFFER_SIZE", &cfg.ReconnectBufferSize)
	parser.parseDuration("RECONNECT_MAX_BACKOFF", &cfg.ReconnectMaxBackoff)
	parser.parseDuration("RECONNECT_INITIAL_BACKOFF", &cfg.ReconnectInitialBackoff)
	parser.parseInt("CB_FAILURE_THRESHOLD", &cfg.CBFailureThreshold)
	parser.parseDuration("CB_TIMEOUT", &cfg.CBTimeout)
	parser.parseInt("CB_HALF_OPEN_REQUESTS", &cfg.CBHalfOpenRequests)
	parser.parseDuration("HEALTH_CHECK_INTERVAL", &cfg.HealthCheckInterval)
	parser.parseEnum("LOG_LEVEL", &cfg.LogLevel, map[string]bool{
		"DEBUG": true,
		"INFO":  true,
		"WARN":  true,
		"ERROR": true,
	})

	// Validate relationships between values
	if cfg.ReconnectInitialBackoff > cfg.ReconnectMaxBackoff {
		parser.errors = append(parser.errors, "RECONNECT_INITIAL_BACKOFF must be less than or equal to RECONNECT_MAX_BACKOFF")
	}

	if len(parser.errors) > 0 {
		return nil, fmt.Errorf("configuration validation failed:\n  - %s", strings.Join(parser.errors, "\n  - "))
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
