package config

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestDefaultResilienceConfig(t *testing.T) {
	cfg := DefaultResilienceConfig()

	// Check reconnection defaults
	if cfg.ReconnectBufferSize != 2*1024*1024 {
		t.Errorf("Expected ReconnectBufferSize=2MB, got %d", cfg.ReconnectBufferSize)
	}
	if cfg.ReconnectMaxBackoff != 30*time.Second {
		t.Errorf("Expected ReconnectMaxBackoff=30s, got %v", cfg.ReconnectMaxBackoff)
	}
	if cfg.ReconnectInitialBackoff != 500*time.Millisecond {
		t.Errorf("Expected ReconnectInitialBackoff=500ms, got %v", cfg.ReconnectInitialBackoff)
	}

	// Check circuit breaker defaults
	if cfg.CBFailureThreshold != 5 {
		t.Errorf("Expected CBFailureThreshold=5, got %d", cfg.CBFailureThreshold)
	}
	if cfg.CBTimeout != 30*time.Second {
		t.Errorf("Expected CBTimeout=30s, got %v", cfg.CBTimeout)
	}
	if cfg.CBHalfOpenRequests != 1 {
		t.Errorf("Expected CBHalfOpenRequests=1, got %d", cfg.CBHalfOpenRequests)
	}

	// Check health check defaults
	if cfg.HealthCheckInterval != 30*time.Second {
		t.Errorf("Expected HealthCheckInterval=30s, got %v", cfg.HealthCheckInterval)
	}

	// Check logging defaults
	if cfg.LogLevel != "INFO" {
		t.Errorf("Expected LogLevel=INFO, got %s", cfg.LogLevel)
	}

	// Validate the default config
	if err := cfg.Validate(); err != nil {
		t.Errorf("Default config should be valid, got error: %v", err)
	}
}

func TestLoadFromEnv_Defaults(t *testing.T) {
	// Clear all env vars
	clearEnvVars()

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv failed: %v", err)
	}

	// Should return default values
	expected := DefaultResilienceConfig()
	if cfg.ReconnectBufferSize != expected.ReconnectBufferSize {
		t.Errorf("Expected default ReconnectBufferSize, got %d", cfg.ReconnectBufferSize)
	}
	if cfg.LogLevel != expected.LogLevel {
		t.Errorf("Expected default LogLevel, got %s", cfg.LogLevel)
	}
}

func TestLoadFromEnv_ValidValues(t *testing.T) {
	clearEnvVars()

	// Set valid environment variables
	_ = os.Setenv("RECONNECT_BUFFER_SIZE", "4MB")
	_ = os.Setenv("RECONNECT_MAX_BACKOFF", "1m")
	_ = os.Setenv("RECONNECT_INITIAL_BACKOFF", "1s")
	_ = os.Setenv("CB_FAILURE_THRESHOLD", "10")
	_ = os.Setenv("CB_TIMEOUT", "45s")
	_ = os.Setenv("CB_HALF_OPEN_REQUESTS", "3")
	_ = os.Setenv("HEALTH_CHECK_INTERVAL", "15s")
	_ = os.Setenv("LOG_LEVEL", "debug")

	defer clearEnvVars()

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv failed: %v", err)
	}

	// Verify parsed values
	if cfg.ReconnectBufferSize != 4*1024*1024 {
		t.Errorf("Expected ReconnectBufferSize=4MB, got %d", cfg.ReconnectBufferSize)
	}
	if cfg.ReconnectMaxBackoff != time.Minute {
		t.Errorf("Expected ReconnectMaxBackoff=1m, got %v", cfg.ReconnectMaxBackoff)
	}
	if cfg.ReconnectInitialBackoff != time.Second {
		t.Errorf("Expected ReconnectInitialBackoff=1s, got %v", cfg.ReconnectInitialBackoff)
	}
	if cfg.CBFailureThreshold != 10 {
		t.Errorf("Expected CBFailureThreshold=10, got %d", cfg.CBFailureThreshold)
	}
	if cfg.CBTimeout != 45*time.Second {
		t.Errorf("Expected CBTimeout=45s, got %v", cfg.CBTimeout)
	}
	if cfg.CBHalfOpenRequests != 3 {
		t.Errorf("Expected CBHalfOpenRequests=3, got %d", cfg.CBHalfOpenRequests)
	}
	if cfg.HealthCheckInterval != 15*time.Second {
		t.Errorf("Expected HealthCheckInterval=15s, got %v", cfg.HealthCheckInterval)
	}
	if cfg.LogLevel != "DEBUG" {
		t.Errorf("Expected LogLevel=DEBUG, got %s", cfg.LogLevel)
	}
}

func TestLoadFromEnv_InvalidBufferSize(t *testing.T) {
	clearEnvVars()

	tests := []struct {
		name  string
		value string
	}{
		{"invalid format", "not-a-number"},
		{"negative value", "-1MB"},
		{"zero value", "0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = os.Setenv("RECONNECT_BUFFER_SIZE", tt.value)
			defer func() { _ = os.Unsetenv("RECONNECT_BUFFER_SIZE") }()

			_, err := LoadFromEnv()
			if err == nil {
				t.Errorf("Expected error for RECONNECT_BUFFER_SIZE=%s, got nil", tt.value)
			}
			if !strings.Contains(err.Error(), "RECONNECT_BUFFER_SIZE") {
				t.Errorf("Error should mention RECONNECT_BUFFER_SIZE, got: %v", err)
			}
		})
	}
}

func TestLoadFromEnv_InvalidDuration(t *testing.T) {
	clearEnvVars()

	tests := []struct {
		name   string
		envVar string
		value  string
	}{
		{"invalid max backoff format", "RECONNECT_MAX_BACKOFF", "invalid"},
		{"negative max backoff", "RECONNECT_MAX_BACKOFF", "-10s"},
		{"zero max backoff", "RECONNECT_MAX_BACKOFF", "0s"},
		{"invalid initial backoff", "RECONNECT_INITIAL_BACKOFF", "not-a-duration"},
		{"invalid cb timeout", "CB_TIMEOUT", "xyz"},
		{"invalid health check", "HEALTH_CHECK_INTERVAL", "bad-format"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = os.Setenv(tt.envVar, tt.value)
			defer func() { _ = os.Unsetenv(tt.envVar) }()

			_, err := LoadFromEnv()
			if err == nil {
				t.Errorf("Expected error for %s=%s, got nil", tt.envVar, tt.value)
			}
			if !strings.Contains(err.Error(), tt.envVar) {
				t.Errorf("Error should mention %s, got: %v", tt.envVar, err)
			}
		})
	}
}

func TestLoadFromEnv_InvalidIntegers(t *testing.T) {
	clearEnvVars()

	tests := []struct {
		name   string
		envVar string
		value  string
	}{
		{"invalid threshold format", "CB_FAILURE_THRESHOLD", "not-a-number"},
		{"negative threshold", "CB_FAILURE_THRESHOLD", "-5"},
		{"zero threshold", "CB_FAILURE_THRESHOLD", "0"},
		{"invalid half open", "CB_HALF_OPEN_REQUESTS", "abc"},
		{"zero half open", "CB_HALF_OPEN_REQUESTS", "0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = os.Setenv(tt.envVar, tt.value)
			defer func() { _ = os.Unsetenv(tt.envVar) }()

			_, err := LoadFromEnv()
			if err == nil {
				t.Errorf("Expected error for %s=%s, got nil", tt.envVar, tt.value)
			}
			if !strings.Contains(err.Error(), tt.envVar) {
				t.Errorf("Error should mention %s, got: %v", tt.envVar, err)
			}
		})
	}
}

func TestLoadFromEnv_InvalidLogLevel(t *testing.T) {
	clearEnvVars()

	_ = os.Setenv("LOG_LEVEL", "INVALID")
	defer func() { _ = os.Unsetenv("LOG_LEVEL") }()

	_, err := LoadFromEnv()
	if err == nil {
		t.Fatal("Expected error for invalid LOG_LEVEL, got nil")
	}
	if !strings.Contains(err.Error(), "LOG_LEVEL") {
		t.Errorf("Error should mention LOG_LEVEL, got: %v", err)
	}
}

func TestLoadFromEnv_LogLevelCaseInsensitive(t *testing.T) {
	clearEnvVars()

	tests := []string{"debug", "DEBUG", "Debug", "DeBuG"}

	for _, level := range tests {
		t.Run(level, func(t *testing.T) {
			_ = os.Setenv("LOG_LEVEL", level)
			defer func() { _ = os.Unsetenv("LOG_LEVEL") }()

			cfg, err := LoadFromEnv()
			if err != nil {
				t.Fatalf("LoadFromEnv failed for LOG_LEVEL=%s: %v", level, err)
			}
			if cfg.LogLevel != "DEBUG" {
				t.Errorf("Expected LogLevel=DEBUG, got %s", cfg.LogLevel)
			}
		})
	}
}

func TestLoadFromEnv_BackoffRelationship(t *testing.T) {
	clearEnvVars()

	// Initial backoff > max backoff should fail
	_ = os.Setenv("RECONNECT_INITIAL_BACKOFF", "1m")
	_ = os.Setenv("RECONNECT_MAX_BACKOFF", "30s")
	defer clearEnvVars()

	_, err := LoadFromEnv()
	if err == nil {
		t.Fatal("Expected error when RECONNECT_INITIAL_BACKOFF > RECONNECT_MAX_BACKOFF")
	}
	if !strings.Contains(err.Error(), "RECONNECT_INITIAL_BACKOFF") {
		t.Errorf("Error should mention backoff relationship, got: %v", err)
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	cfg := DefaultResilienceConfig()
	if err := cfg.Validate(); err != nil {
		t.Errorf("Default config should be valid, got: %v", err)
	}
}

func TestValidate_InvalidValues(t *testing.T) {
	tests := []struct {
		name   string
		modify func(*ResilienceConfig)
	}{
		{
			name: "zero buffer size",
			modify: func(c *ResilienceConfig) {
				c.ReconnectBufferSize = 0
			},
		},
		{
			name: "negative buffer size",
			modify: func(c *ResilienceConfig) {
				c.ReconnectBufferSize = -1
			},
		},
		{
			name: "zero max backoff",
			modify: func(c *ResilienceConfig) {
				c.ReconnectMaxBackoff = 0
			},
		},
		{
			name: "zero initial backoff",
			modify: func(c *ResilienceConfig) {
				c.ReconnectInitialBackoff = 0
			},
		},
		{
			name: "initial > max backoff",
			modify: func(c *ResilienceConfig) {
				c.ReconnectInitialBackoff = 1 * time.Minute
				c.ReconnectMaxBackoff = 30 * time.Second
			},
		},
		{
			name: "zero failure threshold",
			modify: func(c *ResilienceConfig) {
				c.CBFailureThreshold = 0
			},
		},
		{
			name: "zero cb timeout",
			modify: func(c *ResilienceConfig) {
				c.CBTimeout = 0
			},
		},
		{
			name: "zero half open requests",
			modify: func(c *ResilienceConfig) {
				c.CBHalfOpenRequests = 0
			},
		},
		{
			name: "zero health check interval",
			modify: func(c *ResilienceConfig) {
				c.HealthCheckInterval = 0
			},
		},
		{
			name: "invalid log level",
			modify: func(c *ResilienceConfig) {
				c.LogLevel = "INVALID"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultResilienceConfig()
			tt.modify(cfg)

			err := cfg.Validate()
			if err == nil {
				t.Errorf("Expected validation error for %s", tt.name)
			}
		})
	}
}

func TestParseByteSize(t *testing.T) {
	tests := []struct {
		input    string
		expected int
		wantErr  bool
	}{
		// Valid cases
		{"1024", 1024, false},
		{"1KB", 1024, false},
		{"1MB", 1024 * 1024, false},
		{"2MB", 2 * 1024 * 1024, false},
		{"1GB", 1024 * 1024 * 1024, false},
		{"1.5MB", int(1.5 * 1024 * 1024), false},
		{"0.5GB", int(0.5 * 1024 * 1024 * 1024), false},
		{"  2MB  ", 2 * 1024 * 1024, false}, // with whitespace
		{"2mb", 2 * 1024 * 1024, false},     // lowercase

		// Invalid cases
		{"", 0, true},
		{"invalid", 0, true},
		{"1.5.5MB", 0, true},
		{"MB", 0, true},
		{"-1MB", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parseByteSize(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for input %q, got nil", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for input %q: %v", tt.input, err)
				}
				if result != tt.expected {
					t.Errorf("Expected %d, got %d for input %q", tt.expected, result, tt.input)
				}
			}
		})
	}
}

// Helper function to clear all resilience-related env vars
func clearEnvVars() {
	_ = os.Unsetenv("RECONNECT_BUFFER_SIZE")
	_ = os.Unsetenv("RECONNECT_MAX_BACKOFF")
	_ = os.Unsetenv("RECONNECT_INITIAL_BACKOFF")
	_ = os.Unsetenv("CB_FAILURE_THRESHOLD")
	_ = os.Unsetenv("CB_TIMEOUT")
	_ = os.Unsetenv("CB_HALF_OPEN_REQUESTS")
	_ = os.Unsetenv("HEALTH_CHECK_INTERVAL")
	_ = os.Unsetenv("LOG_LEVEL")
}
