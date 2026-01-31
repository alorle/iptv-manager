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
	"github.com/alorle/iptv-manager/overrides"
)

// Mock M3U content for testing
const mockM3U = `#EXTM3U
#EXTINF:-1 tvg-id="test1" tvg-name="Test Channel 1" tvg-logo="http://logo1.png" group-title="Sports",Test Channel 1
acestream://1234567890abcdef1234567890abcdef12345678
#EXTINF:-1 tvg-id="test2" tvg-name="Test Channel 2" tvg-logo="http://logo2.png" group-title="Movies",Test Channel 2
acestream://abcdef1234567890abcdef1234567890abcdef12
`

func setupTestHandler(t *testing.T) (*ChannelsHandler, *httptest.Server, func()) {
	// Create temporary directory for overrides
	tmpDir := t.TempDir()
	overridesPath := filepath.Join(tmpDir, "overrides.yaml")

	// Create overrides manager
	mgr, err := overrides.NewManager(overridesPath)
	if err != nil {
		t.Fatalf("Failed to create overrides manager: %v", err)
	}

	// Create mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-mpegurl")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockM3U))
	}))

	// Create fetcher with temp cache
	cacheDir := filepath.Join(tmpDir, "cache")
	storage, err := cache.NewFileStorage(cacheDir)
	if err != nil {
		t.Fatalf("Failed to create cache storage: %v", err)
	}

	fetch := fetcher.New(5*time.Second, storage, 1*time.Hour)

	// Create handler
	handler := NewChannelsHandler(fetch, mgr, server.URL, server.URL)

	cleanup := func() {
		server.Close()
		os.RemoveAll(tmpDir)
	}

	return handler, server, cleanup
}

func TestDeleteOverride_Success(t *testing.T) {
	handler, _, cleanup := setupTestHandler(t)
	defer cleanup()

	acestreamID := "1234567890abcdef1234567890abcdef12345678"

	// First, create an override using PATCH
	patchBody := UpdateChannelRequest{
		Enabled: boolPtr(false),
		TvgName: stringPtr("Modified Channel"),
	}
	patchJSON, _ := json.Marshal(patchBody)
	patchReq := httptest.NewRequest(http.MethodPatch, "/api/channels/"+acestreamID, bytes.NewReader(patchJSON))
	patchW := httptest.NewRecorder()
	handler.ServeHTTP(patchW, patchReq)

	if patchW.Code != http.StatusOK {
		t.Fatalf("Failed to create override: status %d", patchW.Code)
	}

	// Verify override was created
	if override := handler.overridesMgr.Get(acestreamID); override == nil {
		t.Fatal("Override was not created")
	}

	// Now delete the override
	req := httptest.NewRequest(http.MethodDelete, "/api/channels/"+acestreamID+"/override", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify response body
	var channel Channel
	if err := json.NewDecoder(w.Body).Decode(&channel); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Check that channel is returned in original state
	if channel.HasOverride {
		t.Error("Expected HasOverride to be false")
	}
	if channel.AcestreamID != acestreamID {
		t.Errorf("Expected acestream_id %s, got %s", acestreamID, channel.AcestreamID)
	}
	if channel.Name != "Test Channel 1" {
		t.Errorf("Expected original name 'Test Channel 1', got %s", channel.Name)
	}
	if !channel.Enabled {
		t.Error("Expected channel to be enabled (original state)")
	}

	// Verify override was deleted from manager
	if override := handler.overridesMgr.Get(acestreamID); override != nil {
		t.Error("Override should have been deleted")
	}
}

func TestDeleteOverride_NoOverride(t *testing.T) {
	handler, _, cleanup := setupTestHandler(t)
	defer cleanup()

	acestreamID := "1234567890abcdef1234567890abcdef12345678"

	// Try to delete override that doesn't exist
	req := httptest.NewRequest(http.MethodDelete, "/api/channels/"+acestreamID+"/override", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should return 404
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}

	body := w.Body.String()
	if body != "No override found for this acestream_id\n" {
		t.Errorf("Expected 'No override found' error, got: %s", body)
	}
}

func TestDeleteOverride_InvalidID(t *testing.T) {
	handler, _, cleanup := setupTestHandler(t)
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
			expectBody: "Invalid acestream_id: must be 40 characters",
		},
		{
			name:       "too long",
			id:         "1234567890abcdef1234567890abcdef123456789",
			expectCode: http.StatusBadRequest,
			expectBody: "Invalid acestream_id: must be 40 characters",
		},
		{
			name:       "non-hex characters",
			id:         "1234567890abcdef1234567890abcdef1234567g",
			expectCode: http.StatusBadRequest,
			expectBody: "Invalid acestream_id: must be hexadecimal",
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

			body := w.Body.String()
			if body != tt.expectBody+"\n" {
				t.Errorf("Expected body '%s', got '%s'", tt.expectBody, body)
			}
		})
	}
}

func TestDeleteOverride_ChannelNotFound(t *testing.T) {
	handler, _, cleanup := setupTestHandler(t)
	defer cleanup()

	// Valid ID but not in sources
	acestreamID := "ffffffffffffffffffffffffffffffffffffffff"

	// First create an override for this non-existent channel
	// (This simulates an orphaned override)
	override := overrides.ChannelOverride{
		Enabled: boolPtr(false),
	}
	handler.overridesMgr.Set(acestreamID, override)

	// Try to delete it
	req := httptest.NewRequest(http.MethodDelete, "/api/channels/"+acestreamID+"/override", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should return 404 because channel doesn't exist in sources
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}

	body := w.Body.String()
	if body != "Channel not found\n" {
		t.Errorf("Expected 'Channel not found' error, got: %s", body)
	}
}

func TestDeleteOverride_MethodNotAllowed(t *testing.T) {
	handler, _, cleanup := setupTestHandler(t)
	defer cleanup()

	acestreamID := "1234567890abcdef1234567890abcdef12345678"

	// Try POST (not allowed)
	req := httptest.NewRequest(http.MethodPost, "/api/channels/"+acestreamID+"/override", nil)
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
