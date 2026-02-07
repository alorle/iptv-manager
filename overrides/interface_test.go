package overrides

import (
	"testing"

	"github.com/alorle/iptv-manager/logging"
)

// TestManagerImplementsInterface ensures Manager implements Interface
func TestManagerImplementsInterface(t *testing.T) {
	t.Parallel()

	// Create a temporary test directory
	tempDir := t.TempDir()
	logger := logging.New(logging.INFO, "[test]")
	mgr, err := NewManager(tempDir+"/test-overrides.yaml", logger)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	var _ Interface = mgr
}

// TestMockManagerImplementsInterface ensures MockManager implements Interface
func TestMockManagerImplementsInterface(t *testing.T) {
	t.Parallel()

	var _ Interface = &MockManager{}
}

// TestMockManagerGet tests the mock Get implementation
func TestMockManagerGet(t *testing.T) {
	t.Parallel()

	testID := "1234567890abcdef1234567890abcdef12345678"
	enabled := true
	tvgID := "test-channel"

	mock := &MockManager{
		GetFunc: func(contentID string) *ChannelOverride {
			if contentID == testID {
				return &ChannelOverride{
					Enabled: &enabled,
					TvgID:   &tvgID,
				}
			}
			return nil
		},
	}

	// Test existing override
	override := mock.Get(testID)
	if override == nil {
		t.Fatal("Expected override to exist")
	}
	if override.Enabled == nil || !*override.Enabled {
		t.Error("Expected Enabled to be true")
	}
	if override.TvgID == nil || *override.TvgID != tvgID {
		t.Errorf("Expected TvgID to be %s, got %v", tvgID, override.TvgID)
	}

	// Test non-existing override
	override = mock.Get("nonexistent")
	if override != nil {
		t.Error("Expected override to be nil for non-existent ID")
	}
}
