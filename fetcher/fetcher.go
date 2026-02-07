package fetcher

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/alorle/iptv-manager/cache"
	"github.com/alorle/iptv-manager/logging"
)

// Fetcher handles fetching M3U content with cache fallback
type Fetcher struct {
	client   *http.Client
	storage  cache.Storage
	cacheTTL time.Duration
	logger   *logging.Logger
}

// New creates a new Fetcher with the specified timeout and cache configuration
func New(timeout time.Duration, storage cache.Storage, cacheTTL time.Duration, logger *logging.Logger) Interface {
	return &Fetcher{
		client: &http.Client{
			Timeout: timeout,
		},
		storage:  storage,
		cacheTTL: cacheTTL,
		logger:   logger,
	}
}

// FetchWithCacheFallback fetches M3U content from the URL with cache fallback
// Returns the content and a boolean indicating whether it was served from cache
func (f *Fetcher) FetchWithCacheFallback(url string) ([]byte, bool, error) {
	cacheKey := cache.DeriveKeyFromURL(url)

	// Attempt to fetch from the source
	f.logger.Info("Attempting to fetch M3U from URL", map[string]interface{}{
		"url": url,
	})

	content, err := f.fetchFromURL(url)
	if err == nil {
		// Success: update cache and return fresh content
		f.logger.Info("Successfully fetched M3U from source", map[string]interface{}{
			"url": url,
		})

		if setErr := f.storage.Set(cacheKey, content); setErr != nil {
			f.logger.Warn("Failed to update cache", map[string]interface{}{
				"url":   url,
				"error": setErr.Error(),
			})
		} else {
			f.logger.Info("Cache updated", map[string]interface{}{
				"url": url,
			})
		}

		return content, false, nil
	}

	// Fetch failed: log the error and try cache fallback
	f.logger.Warn("Failed to fetch M3U from source", map[string]interface{}{
		"url":   url,
		"error": err.Error(),
	})
	f.logger.Info("Attempting cache fallback", map[string]interface{}{
		"url": url,
	})

	// Check if we have cached content (even if expired)
	entry, cacheErr := f.storage.Get(cacheKey)
	if cacheErr != nil {
		// No cache available - return error with 502 indication
		f.logger.Warn("Cache miss - no fallback available", map[string]interface{}{
			"url":   url,
			"error": cacheErr.Error(),
		})
		return nil, false, fmt.Errorf("upstream fetch failed and no cache available: %w", err)
	}

	// Serve stale cache as fallback
	f.logger.Info("Serving stale cache", map[string]interface{}{
		"url":       url,
		"cached_at": entry.Timestamp.Format(time.RFC3339),
	})
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
			f.logger.Warn("Failed to close response body", map[string]interface{}{
				"error": closeErr.Error(),
			})
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
			f.logger.Warn("Error checking cache expiration", map[string]interface{}{
				"url":   url,
				"error": expErr.Error(),
			})
			// Treat as expired and continue to fetch
		} else if !expired {
			// Cache is fresh - serve immediately
			f.logger.Info("Serving fresh cache", map[string]interface{}{
				"url":       url,
				"cached_at": entry.Timestamp.Format(time.RFC3339),
				"age":       time.Since(entry.Timestamp).String(),
			})
			return entry.Content, true, false, nil
		}

		// Cache is expired - log and attempt refresh
		f.logger.Info("Cache expired - attempting refresh", map[string]interface{}{
			"url":       url,
			"cached_at": entry.Timestamp.Format(time.RFC3339),
			"age":       time.Since(entry.Timestamp).String(),
		})
	} else {
		// No cache exists
		f.logger.Info("No cache found", map[string]interface{}{
			"url": url,
		})
	}

	// Step 2: Cache is expired or doesn't exist - attempt to fetch
	f.logger.Info("Attempting to fetch M3U from URL", map[string]interface{}{
		"url": url,
	})

	content, fetchErr := f.fetchFromURL(url)
	if fetchErr == nil {
		// Fetch succeeded - update cache and serve new content
		f.logger.Info("Successfully fetched M3U from source", map[string]interface{}{
			"url": url,
		})

		if setErr := f.storage.Set(cacheKey, content); setErr != nil {
			f.logger.Warn("Failed to update cache", map[string]interface{}{
				"url":   url,
				"error": setErr.Error(),
			})
		} else {
			f.logger.Info("Cache updated", map[string]interface{}{
				"url": url,
			})
		}

		return content, false, false, nil
	}

	// Step 3: Fetch failed - check if we can serve stale cache
	f.logger.Warn("Failed to fetch M3U from source", map[string]interface{}{
		"url":   url,
		"error": fetchErr.Error(),
	})

	if cacheErr != nil {
		// No cache available at all
		f.logger.Warn("No cache available for fallback", map[string]interface{}{
			"url": url,
		})
		return nil, false, false, fmt.Errorf("upstream fetch failed and no cache available: %w", fetchErr)
	}

	// Serve stale cache with warning
	f.logger.Warn("Serving stale cache", map[string]interface{}{
		"url":       url,
		"cached_at": entry.Timestamp.Format(time.RFC3339),
		"age":       time.Since(entry.Timestamp).String(),
	})

	return entry.Content, true, true, nil
}
