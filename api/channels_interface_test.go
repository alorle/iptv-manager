package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alorle/iptv-manager/fetcher"
	"github.com/alorle/iptv-manager/overrides"
)

// TestChannelsHandlerWithMocks demonstrates using interface-based mocks for testing
func TestChannelsHandlerWithMocks(t *testing.T) {
	t.Parallel()

	// Create mock fetcher
	mockFetcher := &fetcher.MockFetcher{
		FetchWithCacheFunc: func(url string) ([]byte, bool, bool, error) {
			// Return a simple M3U playlist for testing
			content := []byte(`#EXTM3U
#EXTINF:-1 tvg-id="test1" tvg-name="Test Channel" tvg-logo="" group-title="Test",Test Channel
acestream://1234567890abcdef1234567890abcdef12345678
`)
			return content, false, false, nil
		},
	}

	// Create mock overrides manager
	mockOverrides := &overrides.MockManager{
		ListFunc: func() map[string]overrides.ChannelOverride {
			return make(map[string]overrides.ChannelOverride)
		},
	}

	// Create handler with mocks
	handler := NewChannelsHandler(mockFetcher, mockOverrides, "test-url")

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/api/channels", nil)
	w := httptest.NewRecorder()

	// Execute request
	handler.ServeHTTP(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}
}

// TestChannelsHandlerWithFailedFetch demonstrates testing error conditions with mocks
func TestChannelsHandlerWithFailedFetch(t *testing.T) {
	t.Parallel()

	// Create mock fetcher that fails
	mockFetcher := &fetcher.MockFetcher{
		FetchWithCacheFunc: func(url string) ([]byte, bool, bool, error) {
			return nil, false, false, http.ErrAbortHandler
		},
	}

	// Create mock overrides manager
	mockOverrides := &overrides.MockManager{
		ListFunc: func() map[string]overrides.ChannelOverride {
			return make(map[string]overrides.ChannelOverride)
		},
	}

	// Create handler with mocks
	handler := NewChannelsHandler(mockFetcher, mockOverrides, "test-url")

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/api/channels", nil)
	w := httptest.NewRecorder()

	// Execute request
	handler.ServeHTTP(w, req)

	// Verify we get Bad Gateway when all sources fail
	if w.Code != http.StatusBadGateway {
		t.Errorf("Expected status BadGateway, got %d", w.Code)
	}
}
