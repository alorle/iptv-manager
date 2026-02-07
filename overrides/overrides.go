package overrides

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/alorle/iptv-manager/logging"
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

// Config is a map from acestream content ID to channel override configuration.
// The key is the acestream content ID (hash), and the value contains the override settings.
type Config map[string]ChannelOverride

// Load reads the overrides configuration from a YAML file.
// If the file does not exist, it returns an empty configuration (not an error).
// Returns an error if the file exists but cannot be read or has invalid format.
func Load(path string) (*Config, error) {
	// Check if file exists
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// File doesn't exist - return empty config, not an error
			config := make(Config)
			return &config, nil
		}
		// Other errors (permissions, etc.)
		return nil, fmt.Errorf("failed to read overrides file: %w", err)
	}

	// Parse YAML
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse overrides file: %w", err)
	}

	// Handle nil map case
	if config == nil {
		config = make(Config)
	}

	return &config, nil
}

// Save writes the overrides configuration to a YAML file.
// It creates the directory structure if it doesn't exist.
// Returns an error if the file cannot be written (permissions, etc.).
func (c *Config) Save(path string) error {
	if c == nil {
		return fmt.Errorf("cannot save nil Config")
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

// Manager provides thread-safe in-memory management of channel overrides
// with automatic persistence to disk.
type Manager struct {
	mu     sync.RWMutex
	config Config
	path   string
	logger *logging.Logger
}

// NewManager creates a new OverridesManager and loads the configuration
// from the specified path. If the file doesn't exist, it starts with an
// empty configuration.
func NewManager(path string, logger *logging.Logger) (Interface, error) {
	config, err := Load(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load overrides: %w", err)
	}

	return &Manager{
		config: *config,
		path:   path,
		logger: logger,
	}, nil
}

// Get retrieves the override configuration for a specific content ID.
// Returns nil if no override exists for the given ID.
func (m *Manager) Get(contentID string) *ChannelOverride {
	m.mu.RLock()
	defer m.mu.RUnlock()

	override, exists := m.config[contentID]
	if !exists {
		return nil
	}

	// Return a copy to prevent external modifications
	return &override
}

// Set updates or creates an override for a specific content ID
// and immediately persists the changes to disk.
func (m *Manager) Set(contentID string, override ChannelOverride) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.config[contentID] = override

	if err := m.config.Save(m.path); err != nil {
		return fmt.Errorf("failed to save overrides: %w", err)
	}

	return nil
}

// Delete removes an override for a specific content ID
// and immediately persists the changes to disk.
func (m *Manager) Delete(contentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.config, contentID)

	if err := m.config.Save(m.path); err != nil {
		return fmt.Errorf("failed to save overrides: %w", err)
	}

	return nil
}

// List returns a copy of all current overrides.
// The returned map is a snapshot and modifications won't affect
// the manager's internal state.
func (m *Manager) List() map[string]ChannelOverride {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Create a copy to prevent external modifications
	result := make(map[string]ChannelOverride, len(m.config))
	for id, override := range m.config {
		result[id] = override
	}

	return result
}

// CleanOrphans removes overrides for content IDs that are not in the provided validIDs list.
// This function should only be called when fresh data is available from upstream sources
// to avoid accidentally deleting overrides during cache fallback scenarios.
// Returns the number of orphaned overrides that were deleted, or an error if the operation fails.
func (m *Manager) CleanOrphans(validIDs []string) (int, error) {
	// Build a set of valid IDs for O(1) lookup
	validSet := make(map[string]bool, len(validIDs))
	for _, id := range validIDs {
		validSet[id] = true
	}

	// Get all current overrides
	allOverrides := m.List()

	// Identify orphaned IDs
	var orphanedIDs []string
	for overrideID := range allOverrides {
		if !validSet[overrideID] {
			orphanedIDs = append(orphanedIDs, overrideID)
		}
	}

	// Delete orphaned overrides one by one
	deletedCount := 0
	for _, id := range orphanedIDs {
		if err := m.Delete(id); err != nil {
			// Log error but continue with other deletions
			return deletedCount, fmt.Errorf("failed to delete orphaned override %s: %w", id, err)
		}
		m.logger.Info("Cleaned up orphaned override", map[string]interface{}{
			"content_id": id,
		})
		deletedCount++
	}

	return deletedCount, nil
}

// BulkUpdateError represents an error that occurred during a bulk update operation
type BulkUpdateError struct {
	ContentID string
	Error     string
}

// BulkUpdateResult contains the result of a bulk update operation
type BulkUpdateResult struct {
	Updated int
	Failed  int
	Errors  []BulkUpdateError
}

// BulkUpdate updates a specific field across multiple content IDs.
// If atomic is true, all updates succeed or none (rollback on any error).
// If atomic is false, partial updates are allowed (best effort).
// Returns a BulkUpdateResult with counts and any errors encountered.
func (m *Manager) BulkUpdate(contentIDs []string, field string, value interface{}, atomic bool) (*BulkUpdateResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := &BulkUpdateResult{
		Errors: []BulkUpdateError{},
	}

	// Validate field name
	validFields := map[string]bool{
		"enabled":     true,
		"tvg_id":      true,
		"tvg_name":    true,
		"tvg_logo":    true,
		"group_title": true,
	}

	if !validFields[field] {
		return nil, fmt.Errorf("invalid field: %s", field)
	}

	// Store original state for rollback if atomic
	var originalState map[string]ChannelOverride
	if atomic {
		originalState = make(map[string]ChannelOverride, len(contentIDs))
		for _, id := range contentIDs {
			if override, exists := m.config[id]; exists {
				originalState[id] = override
			}
		}
	}

	// Apply updates
	for _, id := range contentIDs {
		// Get existing override or create new one
		override, exists := m.config[id]
		if !exists {
			override = ChannelOverride{}
		}

		// Update the specified field
		if err := updateOverrideField(&override, field, value); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, BulkUpdateError{
				ContentID: id,
				Error:     err.Error(),
			})

			if atomic {
				// Rollback all changes
				m.config = make(Config, len(originalState))
				for origID, origOverride := range originalState {
					m.config[origID] = origOverride
				}
				return result, fmt.Errorf("atomic operation failed, rolled back all changes")
			}
			continue
		}

		m.config[id] = override
		result.Updated++
	}

	// Persist changes if any updates succeeded
	if result.Updated > 0 {
		if err := m.config.Save(m.path); err != nil {
			// If atomic and save fails, rollback
			if atomic {
				m.config = make(Config, len(originalState))
				for origID, origOverride := range originalState {
					m.config[origID] = origOverride
				}
				return result, fmt.Errorf("failed to save overrides: %w", err)
			}
			return result, fmt.Errorf("failed to save overrides: %w", err)
		}
	}

	return result, nil
}

// updateOverrideField updates a specific field in a ChannelOverride struct
func updateOverrideField(override *ChannelOverride, field string, value interface{}) error {
	switch field {
	case "enabled":
		boolVal, ok := value.(bool)
		if !ok {
			return fmt.Errorf("enabled must be a boolean")
		}
		override.Enabled = &boolVal

	case "tvg_id":
		strVal, ok := value.(string)
		if !ok {
			return fmt.Errorf("tvg_id must be a string")
		}
		override.TvgID = &strVal

	case "tvg_name":
		strVal, ok := value.(string)
		if !ok {
			return fmt.Errorf("tvg_name must be a string")
		}
		override.TvgName = &strVal

	case "tvg_logo":
		strVal, ok := value.(string)
		if !ok {
			return fmt.Errorf("tvg_logo must be a string")
		}
		override.TvgLogo = &strVal

	case "group_title":
		strVal, ok := value.(string)
		if !ok {
			return fmt.Errorf("group_title must be a string")
		}
		override.GroupTitle = &strVal

	default:
		return fmt.Errorf("unknown field: %s", field)
	}

	return nil
}
