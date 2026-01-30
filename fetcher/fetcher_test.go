package fetcher

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alorle/iptv-manager/cache"
)

// mockStorage is a simple in-memory cache for testing
type mockStorage struct {
	data map[string]*cache.Entry
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		data: make(map[string]*cache.Entry),
	}
}

func (m *mockStorage) Get(key string) (*cache.Entry, error) {
	entry, exists := m.data[key]
	if !exists {
		return nil, fmt.Errorf("cache entry not found")
	}
	return entry, nil
}

func (m *mockStorage) Set(key string, content []byte) error {
	m.data[key] = &cache.Entry{
		Content:   content,
		Timestamp: time.Now(),
	}
	return nil
}

func (m *mockStorage) IsExpired(key string, ttl time.Duration) (bool, error) {
	entry, exists := m.data[key]
	if !exists {
		return true, nil
	}
	return time.Since(entry.Timestamp) > ttl, nil
}

func TestNew(t *testing.T) {
	storage := newMockStorage()
	timeout := 10 * time.Second
	cacheTTL := 1 * time.Hour

	fetcher := New(timeout, storage, cacheTTL)

	if fetcher == nil {
		t.Fatal("Expected fetcher to be non-nil")
	}

	if fetcher.client == nil {
		t.Error("Expected HTTP client to be initialized")
	}

	if fetcher.client.Timeout != timeout {
		t.Errorf("Expected timeout %v, got %v", timeout, fetcher.client.Timeout)
	}

	if fetcher.storage == nil {
		t.Error("Expected storage to be initialized")
	}

	if fetcher.cacheTTL != cacheTTL {
		t.Errorf("Expected cacheTTL %v, got %v", cacheTTL, fetcher.cacheTTL)
	}
}

func TestFetchWithCacheFallback_SuccessfulFetch(t *testing.T) {
	// Create a test server that returns M3U content
	expectedContent := "#EXTM3U\n#EXTINF:-1,Test Channel\nhttp://example.com/stream.m3u8\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expectedContent))
	}))
	defer server.Close()

	storage := newMockStorage()
	fetcher := New(10*time.Second, storage, 1*time.Hour)

	content, fromCache, err := fetcher.FetchWithCacheFallback(server.URL)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if fromCache {
		t.Error("Expected content to be from source, not cache")
	}

	if string(content) != expectedContent {
		t.Errorf("Expected content %q, got %q", expectedContent, string(content))
	}

	// Verify cache was updated
	cacheKey := cache.DeriveKeyFromURL(server.URL)
	entry, err := storage.Get(cacheKey)
	if err != nil {
		t.Fatalf("Expected cache to be updated, got error: %v", err)
	}

	if string(entry.Content) != expectedContent {
		t.Errorf("Expected cached content %q, got %q", expectedContent, string(entry.Content))
	}
}

func TestFetchWithCacheFallback_FetchFailure_CacheFallback(t *testing.T) {
	// Create a test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	storage := newMockStorage()
	fetcher := New(10*time.Second, storage, 1*time.Hour)

	// Pre-populate cache with stale content
	cacheKey := cache.DeriveKeyFromURL(server.URL)
	staleContent := "#EXTM3U\n#EXTINF:-1,Stale Channel\nhttp://example.com/stale.m3u8\n"
	storage.Set(cacheKey, []byte(staleContent))

	content, fromCache, err := fetcher.FetchWithCacheFallback(server.URL)

	if err != nil {
		t.Fatalf("Expected no error (cache fallback), got: %v", err)
	}

	if !fromCache {
		t.Error("Expected content to be from cache")
	}

	if string(content) != staleContent {
		t.Errorf("Expected stale content %q, got %q", staleContent, string(content))
	}
}

func TestFetchWithCacheFallback_FetchFailure_NoCache(t *testing.T) {
	// Create a test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	storage := newMockStorage()
	fetcher := New(10*time.Second, storage, 1*time.Hour)

	content, fromCache, err := fetcher.FetchWithCacheFallback(server.URL)

	if err == nil {
		t.Fatal("Expected error when fetch fails and no cache exists")
	}

	if fromCache {
		t.Error("Expected content not to be from cache")
	}

	if content != nil {
		t.Errorf("Expected nil content, got: %v", content)
	}

	expectedErrMsg := "upstream fetch failed and no cache available"
	if err.Error()[:len(expectedErrMsg)] != expectedErrMsg {
		t.Errorf("Expected error message to start with %q, got: %v", expectedErrMsg, err)
	}
}

func TestFetchWithCacheFallback_NetworkTimeout(t *testing.T) {
	// Create a test server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("#EXTM3U\n"))
	}))
	defer server.Close()

	storage := newMockStorage()
	// Set very short timeout to trigger timeout error
	fetcher := New(50*time.Millisecond, storage, 1*time.Hour)

	// Pre-populate cache
	cacheKey := cache.DeriveKeyFromURL(server.URL)
	cachedContent := "#EXTM3U\n#EXTINF:-1,Cached Channel\nhttp://example.com/cached.m3u8\n"
	storage.Set(cacheKey, []byte(cachedContent))

	content, fromCache, err := fetcher.FetchWithCacheFallback(server.URL)

	// Should fallback to cache due to timeout
	if err != nil {
		t.Fatalf("Expected no error (cache fallback), got: %v", err)
	}

	if !fromCache {
		t.Error("Expected content to be from cache due to timeout")
	}

	if string(content) != cachedContent {
		t.Errorf("Expected cached content %q, got %q", cachedContent, string(content))
	}
}

func TestFetchWithCacheFallback_Non200StatusCode(t *testing.T) {
	testCases := []struct {
		name       string
		statusCode int
	}{
		{"404 Not Found", http.StatusNotFound},
		{"500 Internal Server Error", http.StatusInternalServerError},
		{"502 Bad Gateway", http.StatusBadGateway},
		{"503 Service Unavailable", http.StatusServiceUnavailable},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
			}))
			defer server.Close()

			storage := newMockStorage()
			fetcher := New(10*time.Second, storage, 1*time.Hour)

			// Pre-populate cache
			cacheKey := cache.DeriveKeyFromURL(server.URL)
			cachedContent := "#EXTM3U\n#EXTINF:-1,Cached Channel\nhttp://example.com/cached.m3u8\n"
			storage.Set(cacheKey, []byte(cachedContent))

			content, fromCache, err := fetcher.FetchWithCacheFallback(server.URL)

			// Should fallback to cache
			if err != nil {
				t.Fatalf("Expected no error (cache fallback), got: %v", err)
			}

			if !fromCache {
				t.Error("Expected content to be from cache")
			}

			if string(content) != cachedContent {
				t.Errorf("Expected cached content %q, got %q", cachedContent, string(content))
			}
		})
	}
}

func TestIsExpired(t *testing.T) {
	storage := newMockStorage()
	fetcher := New(10*time.Second, storage, 1*time.Hour)

	testURL := "http://example.com/test.m3u"
	cacheKey := cache.DeriveKeyFromURL(testURL)

	// Test with no cache entry
	expired, err := fetcher.IsExpired(testURL)
	if err != nil {
		t.Errorf("Expected no error for missing cache, got: %v", err)
	}
	if !expired {
		t.Error("Expected missing cache to be considered expired")
	}

	// Add fresh cache entry
	storage.Set(cacheKey, []byte("#EXTM3U\n"))

	expired, err = fetcher.IsExpired(testURL)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if expired {
		t.Error("Expected fresh cache to not be expired")
	}

	// Manually set old timestamp to simulate expiration
	storage.data[cacheKey].Timestamp = time.Now().Add(-2 * time.Hour)

	expired, err = fetcher.IsExpired(testURL)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if !expired {
		t.Error("Expected old cache to be expired")
	}
}

func TestFetchFromURL_LargeContent(t *testing.T) {
	// Create a large M3U playlist
	largeContent := "#EXTM3U\n"
	for i := 0; i < 10000; i++ {
		largeContent += fmt.Sprintf("#EXTINF:-1,Channel %d\nhttp://example.com/stream%d.m3u8\n", i, i)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(largeContent))
	}))
	defer server.Close()

	storage := newMockStorage()
	fetcher := New(10*time.Second, storage, 1*time.Hour)

	content, fromCache, err := fetcher.FetchWithCacheFallback(server.URL)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if fromCache {
		t.Error("Expected content to be from source, not cache")
	}

	if string(content) != largeContent {
		t.Errorf("Content length mismatch: expected %d, got %d", len(largeContent), len(content))
	}
}
