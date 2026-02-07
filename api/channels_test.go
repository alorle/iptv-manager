package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alorle/iptv-manager/cache"
	"github.com/alorle/iptv-manager/fetcher"
	"github.com/alorle/iptv-manager/logging"
	"github.com/alorle/iptv-manager/overrides"
)

const (
	testChannelContentID = "1234567890abcdef1234567890abcdef12345678"

	// Mock M3U content for testing
	mockM3U = `#EXTM3U
#EXTINF:-1 tvg-id="test1" tvg-name="Test Channel 1" tvg-logo="http://logo1.png" group-title="Sports",Test Channel 1
acestream://1234567890abcdef1234567890abcdef12345678
#EXTINF:-1 tvg-id="test2" tvg-name="Test Channel 2" tvg-logo="http://logo2.png" group-title="Movies",Test Channel 2
acestream://abcdef1234567890abcdef1234567890abcdef12
`
)

func setupTestHandler(t *testing.T) (*ChannelsHandler, func()) {
	// Create test logger
	logger := logging.New(logging.INFO, "[test]")

	// Create temporary directory for overrides
	tmpDir := t.TempDir()
	overridesPath := filepath.Join(tmpDir, "overrides.yaml")

	// Create overrides manager
	mgr, err := overrides.NewManager(overridesPath, logger)
	if err != nil {
		t.Fatalf("Failed to create overrides manager: %v", err)
	}

	// Create mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-mpegurl")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockM3U))
	}))

	// Create fetcher with temp cache
	cacheDir := filepath.Join(tmpDir, "cache")
	storage, err := cache.NewFileStorage(cacheDir)
	if err != nil {
		t.Fatalf("Failed to create cache storage: %v", err)
	}

	fetch := fetcher.New(5*time.Second, storage, 1*time.Hour, logger)

	// Create handler
	handler := NewChannelsHandler(fetch, mgr, logger, server.URL, server.URL)

	cleanup := func() {
		server.Close()
		_ = os.RemoveAll(tmpDir)
	}

	return handler, cleanup
}

func TestDeleteOverride_Success(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	contentID := testChannelContentID

	// First, create an override using PATCH
	patchBody := UpdateChannelRequest{
		Enabled: boolPtr(false),
		TvgName: stringPtr("Modified Channel"),
	}
	patchJSON, _ := json.Marshal(patchBody)
	patchReq := httptest.NewRequest(http.MethodPatch, "/api/channels/"+contentID, bytes.NewReader(patchJSON))
	patchW := httptest.NewRecorder()
	handler.ServeHTTP(patchW, patchReq)

	if patchW.Code != http.StatusOK {
		t.Fatalf("Failed to create override: status %d", patchW.Code)
	}

	// Verify override was created
	if override := handler.overridesMgr.Get(contentID); override == nil {
		t.Fatal("Override was not created")
	}

	// Now delete the override
	req := httptest.NewRequest(http.MethodDelete, "/api/channels/"+contentID+"/override", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify response body
	var stream streamData
	if err := json.NewDecoder(w.Body).Decode(&stream); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Check that stream is returned in original state
	if stream.HasOverride {
		t.Error("Expected HasOverride to be false")
	}
	if stream.ContentID != contentID {
		t.Errorf("Expected content_id %s, got %s", contentID, stream.ContentID)
	}
	if stream.Name != "Test Channel 1" {
		t.Errorf("Expected original name 'Test Channel 1', got %s", stream.Name)
	}
	if !stream.Enabled {
		t.Error("Expected stream to be enabled (original state)")
	}

	// Verify override was deleted from manager
	if override := handler.overridesMgr.Get(contentID); override != nil {
		t.Error("Override should have been deleted")
	}
}

func TestDeleteOverride_NoOverride(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	contentID := testChannelContentID

	// Try to delete override that doesn't exist
	req := httptest.NewRequest(http.MethodDelete, "/api/channels/"+contentID+"/override", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should return 404
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}

	// Check JSON error response
	var errResp struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}
	if errResp.Error != "No override found for this content_id" {
		t.Errorf("Expected 'No override found' error, got: %s", errResp.Error)
	}
}

func TestDeleteOverride_InvalidID(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	tests := []struct {
		name       string
		id         string
		expectCode int
		expectBody string
	}{
		{
			name:       "too short",
			id:         "1234567890abcdef",
			expectCode: http.StatusBadRequest,
			expectBody: "Invalid content_id: must be 40 hexadecimal characters",
		},
		{
			name:       "too long",
			id:         "1234567890abcdef1234567890abcdef123456789",
			expectCode: http.StatusBadRequest,
			expectBody: "Invalid content_id: must be 40 hexadecimal characters",
		},
		{
			name:       "non-hex characters",
			id:         "1234567890abcdef1234567890abcdef1234567g",
			expectCode: http.StatusBadRequest,
			expectBody: "Invalid content_id: must be 40 hexadecimal characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/api/channels/"+tt.id+"/override", nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.expectCode {
				t.Errorf("Expected status %d, got %d", tt.expectCode, w.Code)
			}

			// Check JSON error response
			var errResp struct {
				Error string `json:"error"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
				t.Fatalf("Failed to parse JSON response: %v", err)
			}
			if errResp.Error != tt.expectBody {
				t.Errorf("Expected error '%s', got '%s'", tt.expectBody, errResp.Error)
			}
		})
	}
}

func TestDeleteOverride_ChannelNotFound(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	// Valid ID but not in sources
	contentID := "ffffffffffffffffffffffffffffffffffffffff"

	// First create an override for this non-existent channel
	// (This simulates an orphaned override)
	override := overrides.ChannelOverride{
		Enabled: boolPtr(false),
	}
	_ = handler.overridesMgr.Set(contentID, override)

	// Try to delete it
	req := httptest.NewRequest(http.MethodDelete, "/api/channels/"+contentID+"/override", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should return 404 because channel doesn't exist in sources
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}

	// Check JSON error response
	var errResp struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}
	if errResp.Error != "Stream not found" {
		t.Errorf("Expected 'Stream not found' error, got: %s", errResp.Error)
	}
}

func TestDeleteOverride_MethodNotAllowed(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	contentID := testChannelContentID

	// Try POST (not allowed)
	req := httptest.NewRequest(http.MethodPost, "/api/channels/"+contentID+"/override", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

// Helper functions
func boolPtr(b bool) *bool {
	return &b
}

func stringPtr(s string) *string {
	return &s
}
