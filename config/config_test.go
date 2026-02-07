package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	// Verify defaults
	if cfg.HTTP.Address != "127.0.0.1" {
		t.Errorf("Expected HTTP.Address to be 127.0.0.1, got %s", cfg.HTTP.Address)
	}
	if cfg.HTTP.Port != "8080" {
		t.Errorf("Expected HTTP.Port to be 8080, got %s", cfg.HTTP.Port)
	}
	if cfg.Acestream.EngineURL != "http://127.0.0.1:6878" {
		t.Errorf("Expected Acestream.EngineURL to be http://127.0.0.1:6878, got %s", cfg.Acestream.EngineURL)
	}
	if cfg.Proxy.ReadTimeout != 5*time.Second {
		t.Errorf("Expected Proxy.ReadTimeout to be 5s, got %v", cfg.Proxy.ReadTimeout)
	}
	if cfg.Proxy.WriteTimeout != 10*time.Second {
		t.Errorf("Expected Proxy.WriteTimeout to be 10s, got %v", cfg.Proxy.WriteTimeout)
	}
	if cfg.Proxy.BufferSize != 4*1024*1024 {
		t.Errorf("Expected Proxy.BufferSize to be 4MB, got %d", cfg.Proxy.BufferSize)
	}
	if cfg.Stream.BufferSize != 1024*1024 {
		t.Errorf("Expected Stream.BufferSize to be 1MB, got %d", cfg.Stream.BufferSize)
	}
	if !cfg.Stream.UseMultiplexing {
		t.Error("Expected Stream.UseMultiplexing to be true")
	}
	if len(cfg.Playlists) != 2 {
		t.Errorf("Expected 2 default playlists, got %d", len(cfg.Playlists))
	}
	if cfg.EPG.URL == "" {
		t.Error("Expected EPG.URL to have a default value")
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr bool
	}{
		{
			name: "valid config",
			mutate: func(cfg *Config) {
				cfg.Cache.Dir = "/tmp/test"
				cfg.Cache.TTL = time.Hour
			},
			wantErr: false,
		},
		{
			name: "missing cache dir",
			mutate: func(cfg *Config) {
				cfg.Cache.Dir = ""
				cfg.Cache.TTL = time.Hour
			},
			wantErr: true,
		},
		{
			name: "missing cache TTL",
			mutate: func(cfg *Config) {
				cfg.Cache.Dir = "/tmp/test"
				cfg.Cache.TTL = 0
			},
			wantErr: true,
		},
		{
			name: "negative cache TTL",
			mutate: func(cfg *Config) {
				cfg.Cache.Dir = "/tmp/test"
				cfg.Cache.TTL = -time.Hour
			},
			wantErr: true,
		},
		{
			name: "empty playlist sources",
			mutate: func(cfg *Config) {
				cfg.Cache.Dir = "/tmp/test"
				cfg.Cache.TTL = time.Hour
				cfg.Playlists = []PlaylistSource{}
			},
			wantErr: true,
		},
		{
			name: "playlist with empty name",
			mutate: func(cfg *Config) {
				cfg.Cache.Dir = "/tmp/test"
				cfg.Cache.TTL = time.Hour
				cfg.Playlists = []PlaylistSource{
					{Name: "", URL: "http://example.com"},
				}
			},
			wantErr: true,
		},
		{
			name: "playlist with empty URL",
			mutate: func(cfg *Config) {
				cfg.Cache.Dir = "/tmp/test"
				cfg.Cache.TTL = time.Hour
				cfg.Playlists = []PlaylistSource{
					{Name: "test", URL: ""},
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			tt.mutate(cfg)
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadFromFile(t *testing.T) {
	// Create temp dir
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Write test config
	configContent := `http:
  address: "0.0.0.0"
  port: "9090"
acestream:
  engine_url: "http://engine:6878"
  player_base_url: "http://engine:6878/ace/getstream"
cache:
  dir: "/tmp/cache"
  ttl: "2h"
proxy:
  read_timeout: "10s"
  write_timeout: "20s"
  buffer_size: 8388608
stream:
  buffer_size: 2097152
  use_multiplexing: false
playlists:
  - name: "test"
    url: "http://example.com/playlist.m3u"
epg:
  url: "http://example.com/epg.xml"
resilience:
  reconnect_buffer_size: 4194304
  reconnect_max_backoff: "60s"
  reconnect_initial_backoff: "1s"
  cb_failure_threshold: 3
  cb_timeout: "20s"
  cb_half_open_requests: 2
  health_check_interval: "15s"
  log_level: "DEBUG"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Load config
	cfg, err := LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("LoadFromFile() error = %v", err)
	}

	// Verify loaded values
	if cfg.HTTP.Address != "0.0.0.0" {
		t.Errorf("Expected HTTP.Address to be 0.0.0.0, got %s", cfg.HTTP.Address)
	}
	if cfg.HTTP.Port != "9090" {
		t.Errorf("Expected HTTP.Port to be 9090, got %s", cfg.HTTP.Port)
	}
	if cfg.Cache.Dir != "/tmp/cache" {
		t.Errorf("Expected Cache.Dir to be /tmp/cache, got %s", cfg.Cache.Dir)
	}
	if cfg.Cache.TTL != 2*time.Hour {
		t.Errorf("Expected Cache.TTL to be 2h, got %v", cfg.Cache.TTL)
	}
	if cfg.Stream.UseMultiplexing {
		t.Error("Expected Stream.UseMultiplexing to be false")
	}
	if len(cfg.Playlists) != 1 {
		t.Errorf("Expected 1 playlist, got %d", len(cfg.Playlists))
	}
	if cfg.Playlists[0].Name != "test" {
		t.Errorf("Expected playlist name to be test, got %s", cfg.Playlists[0].Name)
	}
	if cfg.Resilience.LogLevel != "DEBUG" {
		t.Errorf("Expected log level to be DEBUG, got %s", cfg.Resilience.LogLevel)
	}
}

func TestApplyEnvOverrides(t *testing.T) {
	// Set environment variables
	envVars := map[string]string{
		"HTTP_ADDRESS":             "192.168.1.1",
		"HTTP_PORT":                "9999",
		"CACHE_DIR":                "/custom/cache",
		"CACHE_TTL":                "3h",
		"PROXY_READ_TIMEOUT":       "15s",
		"PROXY_WRITE_TIMEOUT":      "25s",
		"PROXY_BUFFER_SIZE":        "16777216",
		"STREAM_BUFFER_SIZE":       "4194304",
		"USE_MULTIPLEXING":         "false",
		"EPG_URL":                  "http://custom-epg.com/guide.xml",
		"ACESTREAM_ENGINE_URL":     "http://custom-engine:6878",
		"ACESTREAM_PLAYER_BASE_URL": "http://custom-player:6878/ace/getstream",
	}

	// Set env vars
	for k, v := range envVars {
		if err := os.Setenv(k, v); err != nil {
			t.Fatalf("Failed to set env var %s: %v", k, err)
		}
	}

	// Clean up after test
	defer func() {
		for k := range envVars {
			os.Unsetenv(k)
		}
	}()

	// Create config and apply overrides
	cfg := Default()
	if err := applyEnvOverrides(cfg); err != nil {
		t.Fatalf("applyEnvOverrides() error = %v", err)
	}

	// Verify overrides
	if cfg.HTTP.Address != "192.168.1.1" {
		t.Errorf("Expected HTTP.Address to be 192.168.1.1, got %s", cfg.HTTP.Address)
	}
	if cfg.HTTP.Port != "9999" {
		t.Errorf("Expected HTTP.Port to be 9999, got %s", cfg.HTTP.Port)
	}
	if cfg.Cache.TTL != 3*time.Hour {
		t.Errorf("Expected Cache.TTL to be 3h, got %v", cfg.Cache.TTL)
	}
	if cfg.Proxy.ReadTimeout != 15*time.Second {
		t.Errorf("Expected Proxy.ReadTimeout to be 15s, got %v", cfg.Proxy.ReadTimeout)
	}
	if cfg.Stream.BufferSize != 4194304 {
		t.Errorf("Expected Stream.BufferSize to be 4194304, got %d", cfg.Stream.BufferSize)
	}
	if cfg.Stream.UseMultiplexing {
		t.Error("Expected Stream.UseMultiplexing to be false")
	}
	if cfg.EPG.URL != "http://custom-epg.com/guide.xml" {
		t.Errorf("Expected EPG.URL to be custom, got %s", cfg.EPG.URL)
	}
}

func TestValidateCacheDir(t *testing.T) {
	tests := []struct {
		name    string
		dir     string
		wantErr bool
	}{
		{
			name:    "empty dir",
			dir:     "",
			wantErr: true,
		},
		{
			name:    "absolute path",
			dir:     "/tmp/cache",
			wantErr: false,
		},
		{
			name:    "relative path",
			dir:     "cache",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validateCacheDir(tt.dir)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateCacheDir() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadWithMissingFile(t *testing.T) {
	// Save original CONFIG_FILE value
	originalConfigFile := os.Getenv("CONFIG_FILE")
	defer func() {
		if originalConfigFile != "" {
			os.Setenv("CONFIG_FILE", originalConfigFile)
		} else {
			os.Unsetenv("CONFIG_FILE")
		}
	}()

	// Set CONFIG_FILE to non-existent file
	os.Setenv("CONFIG_FILE", "/non/existent/config.yaml")

	// Set required env vars
	os.Setenv("CACHE_DIR", "/tmp/test")
	os.Setenv("CACHE_TTL", "1h")
	defer func() {
		os.Unsetenv("CACHE_DIR")
		os.Unsetenv("CACHE_TTL")
	}()

	// Load should use defaults when file doesn't exist
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() should not error when config file is missing: %v", err)
	}

	// Verify we got default values with env overrides
	if cfg.HTTP.Address != "127.0.0.1" {
		t.Errorf("Expected default HTTP.Address, got %s", cfg.HTTP.Address)
	}
}
