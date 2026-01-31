package overrides

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ChannelOverride defines optional overrides for a channel's metadata.
// All fields are pointers to distinguish between "not configured" (nil)
// and "set to empty value" (pointer to empty string or false).
type ChannelOverride struct {
	// Enabled controls whether the channel is enabled or disabled.
	// nil means use default behavior, otherwise use the specified value.
	Enabled *bool `yaml:"enabled,omitempty"`

	// TvgID overrides the tvg-id attribute for the channel.
	// nil means no override, empty string means explicitly set to empty.
	TvgID *string `yaml:"tvg_id,omitempty"`

	// TvgName overrides the tvg-name attribute for the channel.
	// nil means no override, empty string means explicitly set to empty.
	TvgName *string `yaml:"tvg_name,omitempty"`

	// TvgLogo overrides the tvg-logo attribute for the channel.
	// nil means no override, empty string means explicitly set to empty.
	TvgLogo *string `yaml:"tvg_logo,omitempty"`

	// GroupTitle overrides the group-title attribute for the channel.
	// nil means no override, empty string means explicitly set to empty.
	GroupTitle *string `yaml:"group_title,omitempty"`
}

// OverridesConfig is a map from acestream content ID to channel override configuration.
// The key is the acestream content ID (hash), and the value contains the override settings.
type OverridesConfig map[string]ChannelOverride

// Load reads the overrides configuration from a YAML file.
// If the file does not exist, it returns an empty configuration (not an error).
// Returns an error if the file exists but cannot be read or has invalid format.
func Load(path string) (*OverridesConfig, error) {
	// Check if file exists
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// File doesn't exist - return empty config, not an error
			config := make(OverridesConfig)
			return &config, nil
		}
		// Other errors (permissions, etc.)
		return nil, fmt.Errorf("failed to read overrides file: %w", err)
	}

	// Parse YAML
	var config OverridesConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse overrides file: %w", err)
	}

	// Handle nil map case
	if config == nil {
		config = make(OverridesConfig)
	}

	return &config, nil
}

// Save writes the overrides configuration to a YAML file.
// It creates the directory structure if it doesn't exist.
// Returns an error if the file cannot be written (permissions, etc.).
func (c *OverridesConfig) Save(path string) error {
	if c == nil {
		return fmt.Errorf("cannot save nil OverridesConfig")
	}

	// Marshal to YAML
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal overrides to YAML: %w", err)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file with restricted permissions (0644)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write overrides file: %w", err)
	}

	return nil
}
