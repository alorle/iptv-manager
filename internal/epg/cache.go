package epg

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"
)

// Cache provides thread-safe caching of EPG channels with automatic refresh
type Cache struct {
	client    *Client
	channels  []EPGChannel
	fetchedAt time.Time
	ttl       time.Duration
	mu        sync.RWMutex
}

// NewCache creates a new EPG cache with the given TTL
func NewCache(client *Client, ttl time.Duration) *Cache {
	return &Cache{
		client:   client,
		channels: []EPGChannel{},
		ttl:      ttl,
	}
}

// InitialFetch performs an initial fetch of EPG data (non-blocking)
func (c *Cache) InitialFetch(ctx context.Context) {
	log.Println("Fetching initial EPG data...")
	if err := c.Refresh(ctx); err != nil {
		log.Printf("Warning: Failed to fetch initial EPG data: %v", err)
		log.Println("Channel creation will be disabled until EPG is available")
	} else {
		log.Printf("Successfully loaded %d channels from EPG", len(c.channels))
	}
}

// StartBackgroundRefresh starts a goroutine that periodically refreshes the cache
func (c *Cache) StartBackgroundRefresh(ctx context.Context) {
	ticker := time.NewTicker(c.ttl)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Println("Stopping EPG background refresh")
				return
			case <-ticker.C:
				log.Println("Refreshing EPG cache...")
				if err := c.Refresh(ctx); err != nil {
					log.Printf("Error refreshing EPG cache: %v", err)
				} else {
					log.Printf("EPG cache refreshed: %d channels", len(c.channels))
				}
			}
		}
	}()
}

// Refresh fetches new EPG data and updates the cache
func (c *Cache) Refresh(ctx context.Context) error {
	channels, err := c.client.Fetch(ctx)
	if err != nil {
		return err
	}

	c.mu.Lock()
	c.channels = channels
	c.fetchedAt = time.Now()
	c.mu.Unlock()

	return nil
}

// GetAll returns all cached EPG channels
func (c *Cache) GetAll() []EPGChannel {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]EPGChannel, len(c.channels))
	copy(result, c.channels)
	return result
}

// Search returns channels matching the query (case-insensitive, searches ID and Name)
func (c *Cache) Search(query string) []EPGChannel {
	if query == "" {
		return c.GetAll()
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	query = strings.ToLower(query)
	var results []EPGChannel

	for _, ch := range c.channels {
		if strings.Contains(strings.ToLower(ch.ID), query) ||
			strings.Contains(strings.ToLower(ch.Name), query) {
			results = append(results, ch)
		}
	}

	return results
}

// FindByID returns a channel by its ID, or nil if not found
func (c *Cache) FindByID(id string) *EPGChannel {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, ch := range c.channels {
		if ch.ID == id {
			return &ch
		}
	}

	return nil
}

// IsAvailable returns true if EPG data has been successfully loaded
func (c *Cache) IsAvailable() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.channels) > 0
}

// LastFetchTime returns when the cache was last updated
func (c *Cache) LastFetchTime() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.fetchedAt
}
