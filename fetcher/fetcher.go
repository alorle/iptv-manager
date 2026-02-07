package fetcher

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/alorle/iptv-manager/cache"
)

// Fetcher handles fetching M3U content with cache fallback
type Fetcher struct {
	client   *http.Client
	storage  cache.Storage
	cacheTTL time.Duration
}

// New creates a new Fetcher with the specified timeout and cache configuration
func New(timeout time.Duration, storage cache.Storage, cacheTTL time.Duration) Interface {
	return &Fetcher{
		client: &http.Client{
			Timeout: timeout,
		},
		storage:  storage,
		cacheTTL: cacheTTL,
	}
}

// FetchWithCacheFallback fetches M3U content from the URL with cache fallback
// Returns the content and a boolean indicating whether it was served from cache
func (f *Fetcher) FetchWithCacheFallback(url string) ([]byte, bool, error) {
	cacheKey := cache.DeriveKeyFromURL(url)

	// Attempt to fetch from the source
	log.Printf("Attempting to fetch M3U from: %s", url)

	content, err := f.fetchFromURL(url)
	if err == nil {
		// Success: update cache and return fresh content
		log.Printf("Successfully fetched M3U from source: %s", url)

		if setErr := f.storage.Set(cacheKey, content); setErr != nil {
			log.Printf("Warning: Failed to update cache for %s: %v", url, setErr)
		} else {
			log.Printf("Cache updated for: %s", url)
		}

		return content, false, nil
	}

	// Fetch failed: log the error and try cache fallback
	log.Printf("Failed to fetch M3U from source: %v", err)
	log.Printf("Attempting cache fallback for: %s", url)

	// Check if we have cached content (even if expired)
	entry, cacheErr := f.storage.Get(cacheKey)
	if cacheErr != nil {
		// No cache available - return error with 502 indication
		log.Printf("Cache miss for %s: %v", url, cacheErr)
		return nil, false, fmt.Errorf("upstream fetch failed and no cache available: %w", err)
	}

	// Serve stale cache as fallback
	log.Printf("Serving stale cache for: %s (cached at: %s)", url, entry.Timestamp.Format(time.RFC3339))
	return entry.Content, true, nil
}

// fetchFromURL performs the actual HTTP fetch with timeout
func (f *Fetcher) fetchFromURL(url string) ([]byte, error) {
	resp, err := f.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Printf("warning: failed to close response body: %v", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request returned status %d: %s", resp.StatusCode, resp.Status)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return content, nil
}

// IsExpired checks if the cached content for the URL is expired
func (f *Fetcher) IsExpired(url string) (bool, error) {
	cacheKey := cache.DeriveKeyFromURL(url)
	return f.storage.IsExpired(cacheKey, f.cacheTTL)
}

// FetchWithCache fetches M3U content with cache-first strategy
// Checks cache freshness first, only fetches if expired
// Returns the content, whether it was from cache, and whether cache was stale
func (f *Fetcher) FetchWithCache(url string) ([]byte, bool, bool, error) {
	cacheKey := cache.DeriveKeyFromURL(url)

	// Step 1: Check if cached content exists
	entry, cacheErr := f.storage.Get(cacheKey)

	if cacheErr == nil {
		// Cache exists - check if it's fresh
		expired, expErr := f.storage.IsExpired(cacheKey, f.cacheTTL)
		if expErr != nil {
			log.Printf("Error checking cache expiration for %s: %v", url, expErr)
			// Treat as expired and continue to fetch
		} else if !expired {
			// Cache is fresh - serve immediately
			log.Printf("Serving fresh cache for: %s (cached at: %s, age: %s)",
				url, entry.Timestamp.Format(time.RFC3339), time.Since(entry.Timestamp))
			return entry.Content, true, false, nil
		}

		// Cache is expired - log and attempt refresh
		log.Printf("Cache expired for: %s (cached at: %s, age: %s)",
			url, entry.Timestamp.Format(time.RFC3339), time.Since(entry.Timestamp))
	} else {
		// No cache exists
		log.Printf("No cache found for: %s", url)
	}

	// Step 2: Cache is expired or doesn't exist - attempt to fetch
	log.Printf("Attempting to fetch M3U from: %s", url)

	content, fetchErr := f.fetchFromURL(url)
	if fetchErr == nil {
		// Fetch succeeded - update cache and serve new content
		log.Printf("Successfully fetched M3U from source: %s", url)

		if setErr := f.storage.Set(cacheKey, content); setErr != nil {
			log.Printf("Warning: Failed to update cache for %s: %v", url, setErr)
		} else {
			log.Printf("Cache updated for: %s", url)
		}

		return content, false, false, nil
	}

	// Step 3: Fetch failed - check if we can serve stale cache
	log.Printf("Failed to fetch M3U from source: %v", fetchErr)

	if cacheErr != nil {
		// No cache available at all
		log.Printf("No cache available for fallback: %s", url)
		return nil, false, false, fmt.Errorf("upstream fetch failed and no cache available: %w", fetchErr)
	}

	// Serve stale cache with warning
	log.Printf("WARNING: Serving stale cache for: %s (cached at: %s, age: %s)",
		url, entry.Timestamp.Format(time.RFC3339), time.Since(entry.Timestamp))

	return entry.Content, true, true, nil
}
