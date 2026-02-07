package fetcher

// MockFetcher is a mock implementation of the Fetcher interface for testing
type MockFetcher struct {
	FetchWithCacheFunc         func(url string) ([]byte, bool, bool, error)
	FetchWithCacheFallbackFunc func(url string) ([]byte, bool, error)
	IsExpiredFunc              func(url string) (bool, error)
}

// FetchWithCache implements Interface.FetchWithCache
func (m *MockFetcher) FetchWithCache(url string) ([]byte, bool, bool, error) {
	if m.FetchWithCacheFunc != nil {
		return m.FetchWithCacheFunc(url)
	}
	return nil, false, false, nil
}

// FetchWithCacheFallback implements Interface.FetchWithCacheFallback
func (m *MockFetcher) FetchWithCacheFallback(url string) ([]byte, bool, error) {
	if m.FetchWithCacheFallbackFunc != nil {
		return m.FetchWithCacheFallbackFunc(url)
	}
	return nil, false, nil
}

// IsExpired implements Interface.IsExpired
func (m *MockFetcher) IsExpired(url string) (bool, error) {
	if m.IsExpiredFunc != nil {
		return m.IsExpiredFunc(url)
	}
	return false, nil
}
