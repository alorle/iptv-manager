package epg

import (
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// Channel represents a channel element in the EPG XML
type Channel struct {
	ID          string `xml:"id,attr"`
	DisplayName string `xml:"display-name"`
}

// TV represents the root element of the EPG XML
type TV struct {
	XMLName  xml.Name  `xml:"tv"`
	Channels []Channel `xml:"channel"`
}

// ChannelInfo holds information about a channel
type ChannelInfo struct {
	ID          string
	DisplayName string
}

// Cache provides fast TVG-ID validation and search functionality
type Cache struct {
	channels    map[string]ChannelInfo // Map of channel ID -> ChannelInfo for O(1) lookup
	channelList []ChannelInfo          // List of all channels for search/suggestion
	epgURL      string
}

// New creates a new Cache and initializes it by fetching and parsing the EPG XML
func New(epgURL string, timeout time.Duration) (*Cache, error) {
	if epgURL == "" {
		epgURL = "https://raw.githubusercontent.com/davidmuma/EPG_dobleM/master/guiatv.xml"
	}

	cache := &Cache{
		channels: make(map[string]ChannelInfo),
		epgURL:   epgURL,
	}

	if err := cache.fetch(timeout); err != nil {
		return nil, fmt.Errorf("failed to initialize EPG cache: %w", err)
	}

	return cache, nil
}

// fetch downloads and parses the EPG XML
func (c *Cache) fetch(timeout time.Duration) error {
	client := &http.Client{
		Timeout: timeout,
	}

	resp, err := client.Get(c.epgURL)
	if err != nil {
		return fmt.Errorf("failed to fetch EPG XML from %s: %w", c.epgURL, err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Printf("warning: failed to close EPG response body: %v", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch EPG XML: status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read EPG XML response: %w", err)
	}

	var tv TV
	if err := xml.Unmarshal(body, &tv); err != nil {
		return fmt.Errorf("failed to parse EPG XML: %w", err)
	}

	// Build the channel map and list
	for _, ch := range tv.Channels {
		info := ChannelInfo{
			ID:          ch.ID,
			DisplayName: ch.DisplayName,
		}
		c.channels[ch.ID] = info
		c.channelList = append(c.channelList, info)
	}

	return nil
}

// IsValid checks if a TVG-ID exists in the EPG
func (c *Cache) IsValid(tvgID string) bool {
	_, exists := c.channels[tvgID]
	return exists
}

// GetChannelInfo retrieves channel information by TVG-ID
func (c *Cache) GetChannelInfo(tvgID string) (ChannelInfo, bool) {
	info, exists := c.channels[tvgID]
	return info, exists
}

// Search returns channels matching a partial TVG-ID or display name (case-insensitive)
// Returns up to maxResults channels
func (c *Cache) Search(query string, maxResults int) []ChannelInfo {
	if query == "" {
		return nil
	}

	query = strings.ToLower(query)
	var results []ChannelInfo

	for _, ch := range c.channelList {
		// Check if query matches ID or display name (case-insensitive)
		if strings.Contains(strings.ToLower(ch.ID), query) ||
			strings.Contains(strings.ToLower(ch.DisplayName), query) {
			results = append(results, ch)
			if len(results) >= maxResults {
				break
			}
		}
	}

	return results
}

// Count returns the total number of channels in the cache
func (c *Cache) Count() int {
	return len(c.channels)
}
