package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alorle/iptv-manager/cache"
	"github.com/alorle/iptv-manager/fetcher"
	"github.com/alorle/iptv-manager/rewriter"
)

const (
	mockM3UContent = `#EXTM3U
#EXTINF:-1 tvg-id="test1" tvg-name="Test Channel 1" tvg-logo="http://example.com/logo1.png" group-title="Sports",Test Channel 1
acestream://0000111122223333444455556666777788889999
#EXTINF:-1 tvg-id="test2" tvg-name="Test Channel 2" group-title="Movies",Test Channel 2
http://example.com/stream.m3u8
#EXTINF:-1 tvg-id="test3" tvg-name="Test Channel 3",Test Channel 3
acestream://aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
`
)

// setupTestEnvironment creates a temporary directory and returns cleanup function
func setupTestEnvironment(t *testing.T) (string, func()) {
	t.Helper()
	tempDir := t.TempDir()
	return tempDir, func() {}
}

// createMockIPFSServer creates an HTTP server that simulates IPFS behavior
func createMockIPFSServer(t *testing.T, shouldFail bool, content string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if shouldFail {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("IPFS node unavailable"))
			return
		}
		w.Header().Set("Content-Type", "audio/x-mpegurl")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(content))
	}))
}

// TestIntegration_FreshFetchFromMockIPFS tests fetching fresh content from mock IPFS server
func TestIntegration_FreshFetchFromMockIPFS(t *testing.T) {
	// Setup test environment
	cacheDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Create mock IPFS server
	mockServer := createMockIPFSServer(t, false, mockM3UContent)
	defer mockServer.Close()

	// Initialize components
	storage, err := cache.NewFileStorage(cacheDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	fetch := fetcher.New(5*time.Second, storage, 1*time.Hour)
	rw := rewriter.New("http://127.0.0.1:6878/ace/getstream")

	// Create test server
	mux := http.NewServeMux()
	mux.HandleFunc("/playlists/test.m3u", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		content, fromCache, stale, err := fetch.FetchWithCache(mockServer.URL)
		if err != nil {
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
			return
		}

		if fromCache {
			t.Error("Expected fresh fetch, but got cached content")
		}
		if stale {
			t.Error("Expected fresh content, but got stale cache")
		}

		rewrittenContent := rw.RewriteM3U(content)
		w.Header().Set("Content-Type", "audio/x-mpegurl")
		w.WriteHeader(http.StatusOK)
		w.Write(rewrittenContent)
	})

	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	// Make request
	resp, err := http.Get(testServer.URL + "/playlists/test.m3u")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Verify HTTP status code
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Verify Content-Type header
	contentType := resp.Header.Get("Content-Type")
	if contentType != "audio/x-mpegurl" {
		t.Errorf("Expected Content-Type 'audio/x-mpegurl', got '%s'", contentType)
	}

	// Read and verify response body
	body := make([]byte, 4096)
	n, _ := resp.Body.Read(body)
	bodyStr := string(body[:n])

	// Verify URL rewriting occurred
	if !strings.Contains(bodyStr, "http://127.0.0.1:6878/ace/getstream?id=0000111122223333444455556666777788889999") {
		t.Error("Expected first acestream URL to be rewritten")
	}
	if !strings.Contains(bodyStr, "http://127.0.0.1:6878/ace/getstream?id=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa") {
		t.Error("Expected second acestream URL to be rewritten")
	}
	if !strings.Contains(bodyStr, "http://example.com/stream.m3u8") {
		t.Error("Expected HTTP URL to be preserved")
	}
	if !strings.Contains(bodyStr, "#EXTINF") {
		t.Error("Expected metadata to be preserved")
	}
}

// TestIntegration_CacheHit tests serving content from fresh cache
func TestIntegration_CacheHit(t *testing.T) {
	// Setup test environment
	cacheDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	requestCount := 0
	// Create mock IPFS server that counts requests
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "audio/x-mpegurl")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockM3UContent))
	}))
	defer mockServer.Close()

	// Initialize components
	storage, err := cache.NewFileStorage(cacheDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	fetch := fetcher.New(5*time.Second, storage, 1*time.Hour)
	rw := rewriter.New("http://127.0.0.1:6878/ace/getstream")

	// Create test server
	mux := http.NewServeMux()
	mux.HandleFunc("/playlists/test.m3u", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		content, fromCache, stale, err := fetch.FetchWithCache(mockServer.URL)
		if err != nil {
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
			return
		}

		rewrittenContent := rw.RewriteM3U(content)
		w.Header().Set("Content-Type", "audio/x-mpegurl")
		w.WriteHeader(http.StatusOK)
		w.Write(rewrittenContent)

		// Store cache status for verification
		w.Header().Set("X-From-Cache", fmt.Sprintf("%v", fromCache))
		w.Header().Set("X-Stale", fmt.Sprintf("%v", stale))
	})

	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	// First request - should fetch from IPFS
	resp1, err := http.Get(testServer.URL + "/playlists/test.m3u")
	if err != nil {
		t.Fatalf("Failed to make first request: %v", err)
	}
	resp1.Body.Close()

	if requestCount != 1 {
		t.Errorf("Expected 1 IPFS request after first fetch, got %d", requestCount)
	}

	// Second request - should serve from cache
	resp2, err := http.Get(testServer.URL + "/playlists/test.m3u")
	if err != nil {
		t.Fatalf("Failed to make second request: %v", err)
	}
	defer resp2.Body.Close()

	// Verify no additional IPFS requests were made
	if requestCount != 1 {
		t.Errorf("Expected 1 IPFS request after cache hit, got %d", requestCount)
	}

	// Verify HTTP status code
	if resp2.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp2.StatusCode)
	}

	// Verify Content-Type header
	contentType := resp2.Header.Get("Content-Type")
	if contentType != "audio/x-mpegurl" {
		t.Errorf("Expected Content-Type 'audio/x-mpegurl', got '%s'", contentType)
	}

	// Read response body
	body := make([]byte, 4096)
	n, _ := resp2.Body.Read(body)
	bodyStr := string(body[:n])

	// Verify URL rewriting still works
	if !strings.Contains(bodyStr, "http://127.0.0.1:6878/ace/getstream?id=0000111122223333444455556666777788889999") {
		t.Error("Expected acestream URL to be rewritten from cache")
	}
}

// TestIntegration_ExpiredCacheRefresh tests cache refresh when TTL expires
func TestIntegration_ExpiredCacheRefresh(t *testing.T) {
	// Setup test environment
	cacheDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	requestCount := 0
	// Create mock IPFS server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "audio/x-mpegurl")
		w.WriteHeader(http.StatusOK)
		// Send different content on second request
		if requestCount == 1 {
			w.Write([]byte(mockM3UContent))
		} else {
			w.Write([]byte("#EXTM3U\n#EXTINF:-1,Updated Channel\nacestream://bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\n"))
		}
	}))
	defer mockServer.Close()

	// Initialize components with very short TTL
	storage, err := cache.NewFileStorage(cacheDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	fetch := fetcher.New(5*time.Second, storage, 100*time.Millisecond) // 100ms TTL
	rw := rewriter.New("http://127.0.0.1:6878/ace/getstream")

	// Create test server
	mux := http.NewServeMux()
	mux.HandleFunc("/playlists/test.m3u", func(w http.ResponseWriter, r *http.Request) {
		content, _, _, err := fetch.FetchWithCache(mockServer.URL)
		if err != nil {
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
			return
		}

		rewrittenContent := rw.RewriteM3U(content)
		w.Header().Set("Content-Type", "audio/x-mpegurl")
		w.WriteHeader(http.StatusOK)
		w.Write(rewrittenContent)
	})

	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	// First request - should fetch from IPFS
	resp1, err := http.Get(testServer.URL + "/playlists/test.m3u")
	if err != nil {
		t.Fatalf("Failed to make first request: %v", err)
	}
	body1 := make([]byte, 4096)
	n1, _ := resp1.Body.Read(body1)
	resp1.Body.Close()

	if requestCount != 1 {
		t.Errorf("Expected 1 IPFS request, got %d", requestCount)
	}

	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)

	// Second request - should refresh cache
	resp2, err := http.Get(testServer.URL + "/playlists/test.m3u")
	if err != nil {
		t.Fatalf("Failed to make second request: %v", err)
	}
	body2 := make([]byte, 4096)
	n2, _ := resp2.Body.Read(body2)
	resp2.Body.Close()

	if requestCount != 2 {
		t.Errorf("Expected 2 IPFS requests after cache expiration, got %d", requestCount)
	}

	// Verify HTTP status code
	if resp2.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp2.StatusCode)
	}

	// Verify Content-Type header
	contentType := resp2.Header.Get("Content-Type")
	if contentType != "audio/x-mpegurl" {
		t.Errorf("Expected Content-Type 'audio/x-mpegurl', got '%s'", contentType)
	}

	// Verify content changed (cache was refreshed)
	bodyStr1 := string(body1[:n1])
	bodyStr2 := string(body2[:n2])

	if !strings.Contains(bodyStr2, "Updated Channel") {
		t.Error("Expected refreshed content with 'Updated Channel'")
	}
	if !strings.Contains(bodyStr2, "http://127.0.0.1:6878/ace/getstream?id=bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb") {
		t.Error("Expected refreshed content with rewritten URL")
	}
	if bodyStr1 == bodyStr2 {
		t.Error("Expected different content after cache refresh")
	}
}

// TestIntegration_IPFSFailureWithStaleCacheFallback tests serving stale cache when IPFS fails
func TestIntegration_IPFSFailureWithStaleCacheFallback(t *testing.T) {
	// Setup test environment
	cacheDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	ipfsAvailable := true
	// Create mock IPFS server that can be toggled
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !ipfsAvailable {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("IPFS node unavailable"))
			return
		}
		w.Header().Set("Content-Type", "audio/x-mpegurl")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockM3UContent))
	}))
	defer mockServer.Close()

	// Initialize components with short TTL
	storage, err := cache.NewFileStorage(cacheDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	fetch := fetcher.New(5*time.Second, storage, 100*time.Millisecond)
	rw := rewriter.New("http://127.0.0.1:6878/ace/getstream")

	// Create test server
	mux := http.NewServeMux()
	mux.HandleFunc("/playlists/test.m3u", func(w http.ResponseWriter, r *http.Request) {
		content, fromCache, stale, err := fetch.FetchWithCache(mockServer.URL)
		if err != nil {
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
			return
		}

		rewrittenContent := rw.RewriteM3U(content)
		w.Header().Set("Content-Type", "audio/x-mpegurl")
		w.Header().Set("X-From-Cache", fmt.Sprintf("%v", fromCache))
		w.Header().Set("X-Stale", fmt.Sprintf("%v", stale))
		w.WriteHeader(http.StatusOK)
		w.Write(rewrittenContent)
	})

	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	// First request - populate cache
	resp1, err := http.Get(testServer.URL + "/playlists/test.m3u")
	if err != nil {
		t.Fatalf("Failed to make first request: %v", err)
	}
	resp1.Body.Close()

	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)

	// Make IPFS unavailable
	ipfsAvailable = false

	// Second request - should serve stale cache
	resp2, err := http.Get(testServer.URL + "/playlists/test.m3u")
	if err != nil {
		t.Fatalf("Failed to make second request: %v", err)
	}
	defer resp2.Body.Close()

	// Verify HTTP status code (should still be OK)
	if resp2.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp2.StatusCode)
	}

	// Verify Content-Type header
	contentType := resp2.Header.Get("Content-Type")
	if contentType != "audio/x-mpegurl" {
		t.Errorf("Expected Content-Type 'audio/x-mpegurl', got '%s'", contentType)
	}

	// Read response body
	body := make([]byte, 4096)
	n, _ := resp2.Body.Read(body)
	bodyStr := string(body[:n])

	// Verify we got stale cache content (original content should be present)
	if !strings.Contains(bodyStr, "http://127.0.0.1:6878/ace/getstream?id=0000111122223333444455556666777788889999") {
		t.Error("Expected stale cache content with rewritten URL")
	}
	if !strings.Contains(bodyStr, "Test Channel 1") {
		t.Error("Expected stale cache content with original metadata")
	}
}

// TestIntegration_URLRewritingOutput tests that URL rewriting produces correct output
func TestIntegration_URLRewritingOutput(t *testing.T) {
	// Setup test environment
	cacheDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Create mock IPFS server
	mockServer := createMockIPFSServer(t, false, mockM3UContent)
	defer mockServer.Close()

	// Initialize components
	storage, err := cache.NewFileStorage(cacheDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	fetch := fetcher.New(5*time.Second, storage, 1*time.Hour)
	rw := rewriter.New("http://127.0.0.1:6878/ace/getstream")

	// Create test server
	mux := http.NewServeMux()
	mux.HandleFunc("/playlists/test.m3u", func(w http.ResponseWriter, r *http.Request) {
		content, _, _, err := fetch.FetchWithCache(mockServer.URL)
		if err != nil {
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
			return
		}

		rewrittenContent := rw.RewriteM3U(content)
		w.Header().Set("Content-Type", "audio/x-mpegurl")
		w.WriteHeader(http.StatusOK)
		w.Write(rewrittenContent)
	})

	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	// Make request
	resp, err := http.Get(testServer.URL + "/playlists/test.m3u")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Read response body
	body := make([]byte, 4096)
	n, _ := resp.Body.Read(body)
	bodyStr := string(body[:n])

	// Verify all acestream URLs are rewritten correctly
	expectedRewrites := []struct {
		original  string
		rewritten string
	}{
		{
			original:  "acestream://0000111122223333444455556666777788889999",
			rewritten: "http://127.0.0.1:6878/ace/getstream?id=0000111122223333444455556666777788889999",
		},
		{
			original:  "acestream://aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			rewritten: "http://127.0.0.1:6878/ace/getstream?id=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		},
	}

	for _, test := range expectedRewrites {
		if strings.Contains(bodyStr, test.original) {
			t.Errorf("Expected original URL '%s' to be rewritten", test.original)
		}
		if !strings.Contains(bodyStr, test.rewritten) {
			t.Errorf("Expected rewritten URL '%s' to be present", test.rewritten)
		}
	}

	// Verify HTTP URLs are preserved
	if !strings.Contains(bodyStr, "http://example.com/stream.m3u8") {
		t.Error("Expected HTTP URL to be preserved unchanged")
	}

	// Verify metadata is preserved
	metadataChecks := []string{
		"#EXTM3U",
		"#EXTINF:-1 tvg-id=\"test1\"",
		"tvg-name=\"Test Channel 1\"",
		"tvg-logo=\"http://example.com/logo1.png\"",
		"group-title=\"Sports\"",
		"Test Channel 1",
	}

	for _, check := range metadataChecks {
		if !strings.Contains(bodyStr, check) {
			t.Errorf("Expected metadata '%s' to be preserved", check)
		}
	}
}

// TestIntegration_ContentTypeHeaders tests that Content-Type headers are correct
func TestIntegration_ContentTypeHeaders(t *testing.T) {
	// Setup test environment
	cacheDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Create mock IPFS server
	mockServer := createMockIPFSServer(t, false, mockM3UContent)
	defer mockServer.Close()

	// Initialize components
	storage, err := cache.NewFileStorage(cacheDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	fetch := fetcher.New(5*time.Second, storage, 1*time.Hour)
	rw := rewriter.New("http://127.0.0.1:6878/ace/getstream")

	// Create test server
	mux := http.NewServeMux()
	mux.HandleFunc("/playlists/test.m3u", func(w http.ResponseWriter, r *http.Request) {
		content, _, _, err := fetch.FetchWithCache(mockServer.URL)
		if err != nil {
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
			return
		}

		rewrittenContent := rw.RewriteM3U(content)
		w.Header().Set("Content-Type", "audio/x-mpegurl")
		w.WriteHeader(http.StatusOK)
		w.Write(rewrittenContent)
	})

	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	// Make request
	resp, err := http.Get(testServer.URL + "/playlists/test.m3u")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Verify Content-Type header
	contentType := resp.Header.Get("Content-Type")
	expectedContentType := "audio/x-mpegurl"

	if contentType != expectedContentType {
		t.Errorf("Expected Content-Type '%s', got '%s'", expectedContentType, contentType)
	}

	// Verify Content-Type is present and not empty
	if contentType == "" {
		t.Error("Content-Type header is missing")
	}
}

// TestIntegration_HTTPStatusCodes tests HTTP status codes for all scenarios
func TestIntegration_HTTPStatusCodes(t *testing.T) {
	tests := []struct {
		name               string
		ipfsFails          bool
		cacheExists        bool
		expectedStatusCode int
		description        string
	}{
		{
			name:               "Success - Fresh fetch",
			ipfsFails:          false,
			cacheExists:        false,
			expectedStatusCode: http.StatusOK,
			description:        "Should return 200 OK when IPFS fetch succeeds",
		},
		{
			name:               "Success - Cache hit",
			ipfsFails:          false,
			cacheExists:        true,
			expectedStatusCode: http.StatusOK,
			description:        "Should return 200 OK when serving from cache",
		},
		{
			name:               "Success - Stale cache fallback",
			ipfsFails:          true,
			cacheExists:        true,
			expectedStatusCode: http.StatusOK,
			description:        "Should return 200 OK when serving stale cache",
		},
		{
			name:               "Error - IPFS failure with no cache",
			ipfsFails:          true,
			cacheExists:        false,
			expectedStatusCode: http.StatusBadGateway,
			description:        "Should return 502 Bad Gateway when IPFS fails and no cache exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test environment
			cacheDir, cleanup := setupTestEnvironment(t)
			defer cleanup()

			// Create mock IPFS server
			mockServer := createMockIPFSServer(t, tt.ipfsFails, mockM3UContent)
			defer mockServer.Close()

			// Initialize components
			storage, err := cache.NewFileStorage(cacheDir)
			if err != nil {
				t.Fatalf("Failed to create storage: %v", err)
			}

			// Pre-populate cache if needed
			if tt.cacheExists {
				cacheKey := fmt.Sprintf("%s", mockServer.URL)
				storage.Set(cacheKey, []byte(mockM3UContent))
			}

			fetch := fetcher.New(5*time.Second, storage, 1*time.Hour)
			rw := rewriter.New("http://127.0.0.1:6878/ace/getstream")

			// Create test server
			mux := http.NewServeMux()
			mux.HandleFunc("/playlists/test.m3u", func(w http.ResponseWriter, r *http.Request) {
				content, _, _, err := fetch.FetchWithCache(mockServer.URL)
				if err != nil {
					http.Error(w, "Bad Gateway", http.StatusBadGateway)
					return
				}

				rewrittenContent := rw.RewriteM3U(content)
				w.Header().Set("Content-Type", "audio/x-mpegurl")
				w.WriteHeader(http.StatusOK)
				w.Write(rewrittenContent)
			})

			testServer := httptest.NewServer(mux)
			defer testServer.Close()

			// Make request
			resp, err := http.Get(testServer.URL + "/playlists/test.m3u")
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}
			defer resp.Body.Close()

			// Verify status code
			if resp.StatusCode != tt.expectedStatusCode {
				t.Errorf("%s: Expected status code %d, got %d", tt.description, tt.expectedStatusCode, resp.StatusCode)
			}
		})
	}
}

// TestIntegration_MethodNotAllowed tests that non-GET methods return 405
func TestIntegration_MethodNotAllowed(t *testing.T) {
	// Setup test environment
	cacheDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	mockServer := createMockIPFSServer(t, false, mockM3UContent)
	defer mockServer.Close()

	storage, err := cache.NewFileStorage(cacheDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	fetch := fetcher.New(5*time.Second, storage, 1*time.Hour)
	rw := rewriter.New("http://127.0.0.1:6878/ace/getstream")

	mux := http.NewServeMux()
	mux.HandleFunc("/playlists/test.m3u", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		content, _, _, err := fetch.FetchWithCache(mockServer.URL)
		if err != nil {
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
			return
		}

		rewrittenContent := rw.RewriteM3U(content)
		w.Header().Set("Content-Type", "audio/x-mpegurl")
		w.WriteHeader(http.StatusOK)
		w.Write(rewrittenContent)
	})

	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	// Test POST method
	resp, err := http.Post(testServer.URL+"/playlists/test.m3u", "text/plain", strings.NewReader("test"))
	if err != nil {
		t.Fatalf("Failed to make POST request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status code %d for POST, got %d", http.StatusMethodNotAllowed, resp.StatusCode)
	}
}

// TestIntegration_RealEndpoints tests the actual elcano and newera endpoints
func TestIntegration_RealEndpoints(t *testing.T) {
	// Setup test environment
	cacheDir := t.TempDir()

	// Set environment variables
	os.Setenv("CACHE_DIR", cacheDir)
	os.Setenv("CACHE_TTL", "1h")
	os.Setenv("HTTP_ADDRESS", "127.0.0.1")
	os.Setenv("HTTP_PORT", "0") // Use random port
	os.Setenv("ACESTREAM_PLAYER_BASE_URL", "http://127.0.0.1:6878/ace/getstream")

	defer func() {
		os.Unsetenv("CACHE_DIR")
		os.Unsetenv("CACHE_TTL")
		os.Unsetenv("HTTP_ADDRESS")
		os.Unsetenv("HTTP_PORT")
		os.Unsetenv("ACESTREAM_PLAYER_BASE_URL")
	}()

	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Ensure cache directory is valid
	if !filepath.IsAbs(cfg.CacheDir) {
		t.Fatalf("Cache directory must be absolute path, got: %s", cfg.CacheDir)
	}

	// Initialize components
	storage, err := cache.NewFileStorage(cfg.CacheDir)
	if err != nil {
		t.Fatalf("Failed to initialize storage: %v", err)
	}

	fetch := fetcher.New(30*time.Second, storage, cfg.CacheTTL)
	rw := rewriter.New(cfg.AcestreamPlayerBaseURL)

	// Create mock IPFS server
	mockServer := createMockIPFSServer(t, false, mockM3UContent)
	defer mockServer.Close()

	// Create test server with real endpoint handlers
	mux := http.NewServeMux()

	// Health endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Elcano endpoint (modified to use mock server)
	mux.HandleFunc("/playlists/elcano.m3u", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		content, _, _, err := fetch.FetchWithCache(mockServer.URL)
		if err != nil {
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
			return
		}

		rewrittenContent := rw.RewriteM3U(content)
		w.Header().Set("Content-Type", "audio/x-mpegurl")
		w.WriteHeader(http.StatusOK)
		w.Write(rewrittenContent)
	})

	// NewEra endpoint (modified to use mock server)
	mux.HandleFunc("/playlists/newera.m3u", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		content, _, _, err := fetch.FetchWithCache(mockServer.URL)
		if err != nil {
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
			return
		}

		rewrittenContent := rw.RewriteM3U(content)
		w.Header().Set("Content-Type", "audio/x-mpegurl")
		w.WriteHeader(http.StatusOK)
		w.Write(rewrittenContent)
	})

	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	// Test health endpoint
	t.Run("Health endpoint", func(t *testing.T) {
		resp, err := http.Get(testServer.URL + "/health")
		if err != nil {
			t.Fatalf("Failed to request health endpoint: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
		}

		body := make([]byte, 10)
		n, _ := resp.Body.Read(body)
		if string(body[:n]) != "OK" {
			t.Errorf("Expected body 'OK', got '%s'", string(body[:n]))
		}
	})

	// Test elcano endpoint
	t.Run("Elcano endpoint", func(t *testing.T) {
		resp, err := http.Get(testServer.URL + "/playlists/elcano.m3u")
		if err != nil {
			t.Fatalf("Failed to request elcano endpoint: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
		}

		contentType := resp.Header.Get("Content-Type")
		if contentType != "audio/x-mpegurl" {
			t.Errorf("Expected Content-Type 'audio/x-mpegurl', got '%s'", contentType)
		}

		body := make([]byte, 4096)
		n, _ := resp.Body.Read(body)
		bodyStr := string(body[:n])

		if !strings.Contains(bodyStr, "http://127.0.0.1:6878/ace/getstream?id=") {
			t.Error("Expected rewritten acestream URLs")
		}
	})

	// Test newera endpoint
	t.Run("NewEra endpoint", func(t *testing.T) {
		resp, err := http.Get(testServer.URL + "/playlists/newera.m3u")
		if err != nil {
			t.Fatalf("Failed to request newera endpoint: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
		}

		contentType := resp.Header.Get("Content-Type")
		if contentType != "audio/x-mpegurl" {
			t.Errorf("Expected Content-Type 'audio/x-mpegurl', got '%s'", contentType)
		}

		body := make([]byte, 4096)
		n, _ := resp.Body.Read(body)
		bodyStr := string(body[:n])

		if !strings.Contains(bodyStr, "http://127.0.0.1:6878/ace/getstream?id=") {
			t.Error("Expected rewritten acestream URLs")
		}
	})
}
