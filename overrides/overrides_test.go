package overrides

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_NonExistentFile(t *testing.T) {
	// Load from non-existent file should return empty config, not error
	config, err := Load("/tmp/nonexistent-test-file.yaml")
	if err != nil {
		t.Fatalf("Load() should not return error for non-existent file, got: %v", err)
	}
	if config == nil {
		t.Fatal("Load() returned nil config")
	}
	if len(*config) != 0 {
		t.Errorf("Load() should return empty config, got %d items", len(*config))
	}
}

func TestLoad_EmptyFile(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "empty.yaml")

	// Create empty file
	if err := os.WriteFile(tmpFile, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Load empty file
	config, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	if config == nil {
		t.Fatal("Load() returned nil config")
	}
	if len(*config) != 0 {
		t.Errorf("Load() should return empty config, got %d items", len(*config))
	}
}

func TestLoad_ValidFile(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "valid.yaml")

	// Create valid YAML file
	enabled := true
	tvgID := "test-channel"
	yamlContent := `abc123def456:
  enabled: true
  tvg_id: test-channel
xyz789:
  enabled: false
`
	if err := os.WriteFile(tmpFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Load file
	config, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	if config == nil {
		t.Fatal("Load() returned nil config")
	}
	if len(*config) != 2 {
		t.Errorf("Expected 2 items in config, got %d", len(*config))
	}

	// Verify first entry
	override1, ok := (*config)["abc123def456"]
	if !ok {
		t.Error("Expected entry 'abc123def456' not found")
	}
	if override1.Enabled == nil || *override1.Enabled != enabled {
		t.Errorf("Expected Enabled=true, got %v", override1.Enabled)
	}
	if override1.TvgID == nil || *override1.TvgID != tvgID {
		t.Errorf("Expected TvgID='test-channel', got %v", override1.TvgID)
	}

	// Verify second entry
	override2, ok := (*config)["xyz789"]
	if !ok {
		t.Error("Expected entry 'xyz789' not found")
	}
	if override2.Enabled == nil || *override2.Enabled != false {
		t.Errorf("Expected Enabled=false, got %v", override2.Enabled)
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "invalid.yaml")

	// Create invalid YAML file
	invalidYAML := `this is not: valid: yaml: content`
	if err := os.WriteFile(tmpFile, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Load should return error
	config, err := Load(tmpFile)
	if err == nil {
		t.Error("Load() should return error for invalid YAML")
	}
	if config != nil {
		t.Error("Load() should return nil config on error")
	}
}

func TestLoad_PermissionError(t *testing.T) {
	// Skip on Windows as file permissions work differently
	if os.Getenv("GOOS") == "windows" {
		t.Skip("Skipping permission test on Windows")
	}

	// Create temporary directory
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "noperms.yaml")

	// Create file with no read permissions
	if err := os.WriteFile(tmpFile, []byte("test: data"), 0000); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Chmod(tmpFile, 0644) // Cleanup

	// Load should return error
	config, err := Load(tmpFile)
	if err == nil {
		t.Error("Load() should return error for file without read permissions")
	}
	if config != nil {
		t.Error("Load() should return nil config on error")
	}
}

func TestSave_NilConfig(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "nil.yaml")

	var nilConfig *OverridesConfig
	err := nilConfig.Save(tmpFile)
	if err == nil {
		t.Error("Save() should return error for nil config")
	}
}

func TestSave_EmptyConfig(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "empty.yaml")

	config := make(OverridesConfig)
	err := config.Save(tmpFile)
	if err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Verify file was created
	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read saved file: %v", err)
	}

	// Empty map should serialize to {}
	expected := "{}\n"
	if string(data) != expected {
		t.Errorf("Expected '%s', got '%s'", expected, string(data))
	}
}

func TestSave_ValidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "valid.yaml")

	// Create config
	enabled := true
	disabled := false
	tvgID := "test-channel"
	tvgName := "Test Channel"
	config := OverridesConfig{
		"abc123": {
			Enabled: &enabled,
			TvgID:   &tvgID,
			TvgName: &tvgName,
		},
		"xyz789": {
			Enabled: &disabled,
		},
	}

	// Save config
	err := config.Save(tmpFile)
	if err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Load it back
	loaded, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify loaded config matches original
	if len(*loaded) != 2 {
		t.Errorf("Expected 2 items, got %d", len(*loaded))
	}

	override1 := (*loaded)["abc123"]
	if override1.Enabled == nil || *override1.Enabled != enabled {
		t.Errorf("Expected Enabled=true, got %v", override1.Enabled)
	}
	if override1.TvgID == nil || *override1.TvgID != tvgID {
		t.Errorf("Expected TvgID='test-channel', got %v", override1.TvgID)
	}
	if override1.TvgName == nil || *override1.TvgName != tvgName {
		t.Errorf("Expected TvgName='Test Channel', got %v", override1.TvgName)
	}

	override2 := (*loaded)["xyz789"]
	if override2.Enabled == nil || *override2.Enabled != disabled {
		t.Errorf("Expected Enabled=false, got %v", override2.Enabled)
	}
}

func TestSave_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	// Use a nested path that doesn't exist
	tmpFile := filepath.Join(tmpDir, "subdir", "nested", "config.yaml")

	config := make(OverridesConfig)
	err := config.Save(tmpFile)
	if err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Error("Save() should create directory structure")
	}
}

func TestSave_InvalidPath(t *testing.T) {
	// Try to save to an invalid path (subdirectory of a file)
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "file.txt")

	// Create a regular file
	if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Try to save to a path that uses the file as a directory
	invalidPath := filepath.Join(tmpFile, "subdir", "config.yaml")
	config := make(OverridesConfig)
	err := config.Save(invalidPath)
	if err == nil {
		t.Error("Save() should return error for invalid path")
	}
}

func TestRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "roundtrip.yaml")

	// Create complex config with all field types
	enabled := true
	disabled := false
	tvgID := "channel-1"
	tvgName := "Channel One"
	tvgLogo := "https://example.com/logo.png"
	groupTitle := "Sports"
	emptyString := ""

	original := OverridesConfig{
		"id1": {
			Enabled:    &enabled,
			TvgID:      &tvgID,
			TvgName:    &tvgName,
			TvgLogo:    &tvgLogo,
			GroupTitle: &groupTitle,
		},
		"id2": {
			Enabled: &disabled,
		},
		"id3": {
			TvgID: &emptyString, // Explicitly empty string
		},
		"id4": {
			// All nil fields
		},
	}

	// Save
	if err := original.Save(tmpFile); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Load
	loaded, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Compare
	if len(*loaded) != len(original) {
		t.Fatalf("Expected %d items, got %d", len(original), len(*loaded))
	}

	for id, origOverride := range original {
		loadedOverride, ok := (*loaded)[id]
		if !ok {
			t.Errorf("Missing override for id '%s'", id)
			continue
		}

		// Compare each field
		if !boolPtrEqual(origOverride.Enabled, loadedOverride.Enabled) {
			t.Errorf("id '%s': Enabled mismatch", id)
		}
		if !stringPtrEqual(origOverride.TvgID, loadedOverride.TvgID) {
			t.Errorf("id '%s': TvgID mismatch", id)
		}
		if !stringPtrEqual(origOverride.TvgName, loadedOverride.TvgName) {
			t.Errorf("id '%s': TvgName mismatch", id)
		}
		if !stringPtrEqual(origOverride.TvgLogo, loadedOverride.TvgLogo) {
			t.Errorf("id '%s': TvgLogo mismatch", id)
		}
		if !stringPtrEqual(origOverride.GroupTitle, loadedOverride.GroupTitle) {
			t.Errorf("id '%s': GroupTitle mismatch", id)
		}
	}
}

// Helper functions for pointer comparison
func boolPtrEqual(a, b *bool) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func stringPtrEqual(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}
