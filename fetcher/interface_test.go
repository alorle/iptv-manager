package fetcher

import (
	"errors"
	"testing"
	"time"

	"github.com/alorle/iptv-manager/cache"
	"github.com/alorle/iptv-manager/logging"
)

// TestFetcherImplementsInterface ensures Fetcher implements Interface
func TestFetcherImplementsInterface(t *testing.T) {
	t.Parallel()

	mockStorage := &cache.MockStorage{}
	logger := logging.New(logging.INFO, "[test]")
	var _ Interface = New(30*time.Second, mockStorage, 1*time.Hour, logger)
}

// TestMockFetcherImplementsInterface ensures MockFetcher implements Interface
func TestMockFetcherImplementsInterface(t *testing.T) {
	t.Parallel()

	var _ Interface = &MockFetcher{}
}

// TestMockFetcherFetchWithCache tests the mock implementation
func TestMockFetcherFetchWithCache(t *testing.T) {
	t.Parallel()

	expectedContent := []byte("test content")
	expectedError := errors.New("test error")

	mock := &MockFetcher{
		FetchWithCacheFunc: func(url string) ([]byte, bool, bool, error) {
			if url == "error-url" {
				return nil, false, false, expectedError
			}
			return expectedContent, true, false, nil
		},
	}

	// Test success case
	content, fromCache, stale, err := mock.FetchWithCache("test-url")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if string(content) != string(expectedContent) {
		t.Errorf("Expected content %s, got %s", expectedContent, content)
	}
	if !fromCache {
		t.Error("Expected fromCache to be true")
	}
	if stale {
		t.Error("Expected stale to be false")
	}

	// Test error case
	_, _, _, err = mock.FetchWithCache("error-url")
	if err != expectedError {
		t.Errorf("Expected error %v, got %v", expectedError, err)
	}
}
