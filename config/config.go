package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// PlaylistSource represents a single M3U playlist source
type PlaylistSource struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
}

// Config holds the complete application configuration
type Config struct {
	// HTTP server settings
	HTTP struct {
		Address string `yaml:"address"`
		Port    string `yaml:"port"`
	} `yaml:"http"`

	// Acestream Engine settings
	Acestream struct {
		EngineURL     string `yaml:"engine_url"`
		PlayerBaseURL string `yaml:"player_base_url"`
	} `yaml:"acestream"`

	// Cache settings
	Cache struct {
		Dir string        `yaml:"dir"`
		TTL time.Duration `yaml:"ttl"`
	} `yaml:"cache"`

	// Proxy settings
	Proxy struct {
		ReadTimeout  time.Duration `yaml:"read_timeout"`
		WriteTimeout time.Duration `yaml:"write_timeout"`
		BufferSize   int           `yaml:"buffer_size"`
	} `yaml:"proxy"`

	// Stream settings
	Stream struct {
		BufferSize      int  `yaml:"buffer_size"`
		UseMultiplexing bool `yaml:"use_multiplexing"`
	} `yaml:"stream"`

	// Playlist sources
	Playlists []PlaylistSource `yaml:"playlists"`

	// EPG URL
	EPG struct {
		URL string `yaml:"url"`
	} `yaml:"epg"`

	// Resilience settings (embedded)
	Resilience ResilienceConfig `yaml:"resilience"`
}

// Validate performs validation on the configuration
func (c *Config) Validate() error {
	var errors []string

	// Validate HTTP settings
	if c.HTTP.Address == "" {
		errors = append(errors, "HTTP address is required")
	}
	if c.HTTP.Port == "" {
		errors = append(errors, "HTTP port is required")
	}

	// Validate Acestream settings
	if c.Acestream.EngineURL == "" {
		errors = append(errors, "Acestream engine URL is required")
	}
	if c.Acestream.PlayerBaseURL == "" {
		errors = append(errors, "Acestream player base URL is required")
	}

	// Validate cache settings
	if c.Cache.Dir == "" {
		errors = append(errors, "Cache directory is required")
	}
	if c.Cache.TTL <= 0 {
		errors = append(errors, "Cache TTL must be positive")
	}

	// Validate proxy settings
	if c.Proxy.ReadTimeout <= 0 {
		errors = append(errors, "Proxy read timeout must be positive")
	}
	if c.Proxy.WriteTimeout <= 0 {
		errors = append(errors, "Proxy write timeout must be positive")
	}
	if c.Proxy.BufferSize <= 0 {
		errors = append(errors, "Proxy buffer size must be positive")
	}

	// Validate stream settings
	if c.Stream.BufferSize <= 0 {
		errors = append(errors, "Stream buffer size must be positive")
	}

	// Validate playlists
	if len(c.Playlists) == 0 {
		errors = append(errors, "At least one playlist source is required")
	}
	for i, pl := range c.Playlists {
		if pl.Name == "" {
			errors = append(errors, fmt.Sprintf("Playlist %d: name is required", i))
		}
		if pl.URL == "" {
			errors = append(errors, fmt.Sprintf("Playlist %d (%s): URL is required", i, pl.Name))
		}
	}

	// Validate EPG URL
	if c.EPG.URL == "" {
		errors = append(errors, "EPG URL is required")
	}

	// Validate resilience config
	if err := c.Resilience.Validate(); err != nil {
		errors = append(errors, fmt.Sprintf("Resilience config: %v", err))
	}

	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil
}

// Default returns a Config with sensible default values
func Default() *Config {
	cfg := &Config{}

	// HTTP defaults
	cfg.HTTP.Address = "127.0.0.1"
	cfg.HTTP.Port = "8080"

	// Acestream defaults
	cfg.Acestream.EngineURL = "http://127.0.0.1:6878"
	cfg.Acestream.PlayerBaseURL = "http://127.0.0.1:6878/ace/getstream"

	// Cache defaults
	cfg.Cache.Dir = "" // Required, no default
	cfg.Cache.TTL = 0  // Required, no default

	// Proxy defaults
	cfg.Proxy.ReadTimeout = 5 * time.Second
	cfg.Proxy.WriteTimeout = 10 * time.Second
	cfg.Proxy.BufferSize = 4 * 1024 * 1024 // 4MB

	// Stream defaults
	cfg.Stream.BufferSize = 1024 * 1024 // 1MB
	cfg.Stream.UseMultiplexing = true

	// Default playlist sources
	cfg.Playlists = []PlaylistSource{
		{
			Name: "elcano",
			URL:  "https://ipfs.io/ipns/k51qzi5uqu5di462t7j4vu4akwfhvtjhy88qbupktvoacqfqe9uforjvhyi4wr/hashes_acestream.m3u",
		},
		{
			Name: "newera",
			URL:  "https://ipfs.io/ipns/k2k4r8oqlcjxsritt5mczkcn4mmvcmymbqw7113fz2flkrerfwfps004/data/listas/lista_fuera_iptv.m3u",
		},
	}

	// EPG default
	cfg.EPG.URL = "https://raw.githubusercontent.com/davidmuma/EPG_dobleM/master/guiatv.xml"

	// Resilience defaults
	cfg.Resilience = *DefaultResilienceConfig()

	return cfg
}

// LoadFromFile loads configuration from a YAML file
func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := Default()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return cfg, nil
}

// Load loads configuration from a file (if provided) and applies environment variable overrides
func Load() (*Config, error) {
	// Get config file path from flag or environment variable
	configPath := os.Getenv("CONFIG_FILE")
	if configPath == "" {
		configPath = "config.yaml"
	}

	var cfg *Config

	// Try to load from file if it exists
	if _, err := os.Stat(configPath); err == nil {
		cfg, err = LoadFromFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load config from %s: %w", configPath, err)
		}
	} else {
		// File doesn't exist, use defaults
		cfg = Default()
	}

	// Apply environment variable overrides
	if err := applyEnvOverrides(cfg); err != nil {
		return nil, fmt.Errorf("failed to apply environment overrides: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// applyEnvOverrides applies environment variable overrides to the configuration
func applyEnvOverrides(cfg *Config) error {
	// HTTP settings
	if val := os.Getenv("HTTP_ADDRESS"); val != "" {
		cfg.HTTP.Address = val
	}
	if val := os.Getenv("HTTP_PORT"); val != "" {
		cfg.HTTP.Port = val
	}

	// Acestream settings
	if val := os.Getenv("ACESTREAM_ENGINE_URL"); val != "" {
		cfg.Acestream.EngineURL = val
	}
	if val := os.Getenv("ACESTREAM_PLAYER_BASE_URL"); val != "" {
		cfg.Acestream.PlayerBaseURL = val
	}

	// Cache settings
	if val := os.Getenv("CACHE_DIR"); val != "" {
		// Normalize to absolute path
		absPath, err := validateCacheDir(val)
		if err != nil {
			return err
		}
		cfg.Cache.Dir = absPath
	}
	if val := os.Getenv("CACHE_TTL"); val != "" {
		duration, err := time.ParseDuration(val)
		if err != nil {
			return fmt.Errorf("invalid CACHE_TTL format (expected duration like '1h', '30m'): %w", err)
		}
		if duration <= 0 {
			return fmt.Errorf("CACHE_TTL must be positive, got: %s", val)
		}
		cfg.Cache.TTL = duration
	}

	// Proxy settings
	if val := os.Getenv("PROXY_READ_TIMEOUT"); val != "" {
		duration, err := time.ParseDuration(val)
		if err != nil {
			return fmt.Errorf("invalid PROXY_READ_TIMEOUT: %w", err)
		}
		if duration <= 0 {
			return fmt.Errorf("PROXY_READ_TIMEOUT must be positive")
		}
		cfg.Proxy.ReadTimeout = duration
	}
	if val := os.Getenv("PROXY_WRITE_TIMEOUT"); val != "" {
		duration, err := time.ParseDuration(val)
		if err != nil {
			return fmt.Errorf("invalid PROXY_WRITE_TIMEOUT: %w", err)
		}
		if duration <= 0 {
			return fmt.Errorf("PROXY_WRITE_TIMEOUT must be positive")
		}
		cfg.Proxy.WriteTimeout = duration
	}
	if val := os.Getenv("PROXY_BUFFER_SIZE"); val != "" {
		size, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("invalid PROXY_BUFFER_SIZE: %w", err)
		}
		if size <= 0 {
			return fmt.Errorf("PROXY_BUFFER_SIZE must be positive")
		}
		cfg.Proxy.BufferSize = size
	}

	// Stream settings
	if val := os.Getenv("STREAM_BUFFER_SIZE"); val != "" {
		size, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("invalid STREAM_BUFFER_SIZE: %w", err)
		}
		if size <= 0 {
			return fmt.Errorf("STREAM_BUFFER_SIZE must be positive")
		}
		cfg.Stream.BufferSize = size
	}
	if val := os.Getenv("USE_MULTIPLEXING"); val != "" {
		cfg.Stream.UseMultiplexing = val == "true" || val == "1"
	}

	// EPG URL
	if val := os.Getenv("EPG_URL"); val != "" {
		cfg.EPG.URL = val
	}

	// Resilience settings (use existing LoadFromEnv logic)
	resCfg, err := LoadFromEnv()
	if err != nil {
		return fmt.Errorf("failed to load resilience config: %w", err)
	}
	cfg.Resilience = *resCfg

	return nil
}

// validateCacheDir validates and normalizes the cache directory path
func validateCacheDir(dir string) (string, error) {
	if dir == "" {
		return "", fmt.Errorf("cache directory cannot be empty")
	}

	// Ensure cache directory is an absolute path
	if !filepath.IsAbs(dir) {
		absPath, err := filepath.Abs(dir)
		if err != nil {
			return "", fmt.Errorf("failed to resolve absolute path for cache dir: %w", err)
		}
		return absPath, nil
	}

	return dir, nil
}

// Print outputs the configuration to stdout
func (c *Config) Print() {
	fmt.Printf("httpAddress: %v\n", c.HTTP.Address)
	fmt.Printf("httpPort: %v\n", c.HTTP.Port)
	fmt.Printf("acestreamPlayerBaseUrl: %v\n", c.Acestream.PlayerBaseURL)
	fmt.Printf("acestreamEngineUrl: %v\n", c.Acestream.EngineURL)
	fmt.Printf("cacheDir: %v\n", c.Cache.Dir)
	fmt.Printf("cacheTTL: %v\n", c.Cache.TTL)
	fmt.Printf("streamBufferSize: %v bytes\n", c.Stream.BufferSize)
	fmt.Printf("useMultiplexing: %v\n", c.Stream.UseMultiplexing)
	fmt.Printf("proxyReadTimeout: %v\n", c.Proxy.ReadTimeout)
	fmt.Printf("proxyWriteTimeout: %v\n", c.Proxy.WriteTimeout)
	fmt.Printf("proxyBufferSize: %v bytes\n", c.Proxy.BufferSize)
	fmt.Printf("playlistSources: %d\n", len(c.Playlists))
	for _, pl := range c.Playlists {
		fmt.Printf("  - %s: %s\n", pl.Name, pl.URL)
	}
	fmt.Printf("epgUrl: %v\n", c.EPG.URL)
	fmt.Printf("logLevel: %v\n", c.Resilience.LogLevel)
}
