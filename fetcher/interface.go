package fetcher

// Interface defines the contract for fetching M3U content with caching
type Interface interface {
	// FetchWithCache fetches M3U content with cache-first strategy
	// Returns: content, fromCache, stale, error
	FetchWithCache(url string) ([]byte, bool, bool, error)

	// FetchWithCacheFallback fetches M3U content from the URL with cache fallback
	// Returns: content, fromCache, error
	FetchWithCacheFallback(url string) ([]byte, bool, error)

	// IsExpired checks if the cached content for the URL is expired
	IsExpired(url string) (bool, error)
}
