package api

import (
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"github.com/alorle/iptv-manager/fetcher"
	"github.com/alorle/iptv-manager/overrides"
	"github.com/alorle/iptv-manager/rewriter"
)

// Channel represents a channel with its metadata and override status
type Channel struct {
	AcestreamID string `json:"acestream_id"`
	Name        string `json:"name"`
	TvgID       string `json:"tvg_id"`
	TvgLogo     string `json:"tvg_logo"`
	GroupTitle  string `json:"group_title"`
	Source      string `json:"source"` // "elcano" or "newera"
	Enabled     bool   `json:"enabled"`
	HasOverride bool   `json:"has_override"`
}

// ChannelsHandler handles the GET /api/channels endpoint
type ChannelsHandler struct {
	fetcher      *fetcher.Fetcher
	overridesMgr *overrides.Manager
	elcanoURL    string
	neweraURL    string
}

// NewChannelsHandler creates a new handler for the channels API
func NewChannelsHandler(fetch *fetcher.Fetcher, overridesMgr *overrides.Manager, elcanoURL, neweraURL string) *ChannelsHandler {
	return &ChannelsHandler{
		fetcher:      fetch,
		overridesMgr: overridesMgr,
		elcanoURL:    elcanoURL,
		neweraURL:    neweraURL,
	}
}

// extractMetadata extracts metadata attributes from an EXTINF line
// Returns tvg-id, tvg-logo, and group-title
func extractMetadata(extinf string) (tvgID, tvgLogo, groupTitle string) {
	// Extract tvg-id
	if matches := regexp.MustCompile(`tvg-id="([^"]*)"`).FindStringSubmatch(extinf); len(matches) > 1 {
		tvgID = matches[1]
	}

	// Extract tvg-logo
	if matches := regexp.MustCompile(`tvg-logo="([^"]*)"`).FindStringSubmatch(extinf); len(matches) > 1 {
		tvgLogo = matches[1]
	}

	// Extract group-title
	if matches := regexp.MustCompile(`group-title="([^"]*)"`).FindStringSubmatch(extinf); len(matches) > 1 {
		groupTitle = matches[1]
	}

	return
}

// parseM3UChannels parses M3U content and extracts channel information
func parseM3UChannels(content []byte, source string) []Channel {
	lines := strings.Split(string(content), "\n")
	var channels []Channel

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		// Look for EXTINF metadata lines
		if strings.HasPrefix(line, "#EXTINF:") {
			// Next line should be the URL
			if i+1 < len(lines) {
				metadata := line
				url := strings.TrimSpace(lines[i+1])
				i++ // Skip the URL line in the next iteration

				// Only process acestream URLs
				if strings.HasPrefix(url, "acestream://") {
					aceID := strings.TrimPrefix(url, "acestream://")
					aceID = strings.TrimSpace(aceID)

					// Validate acestream ID (40 hex characters)
					if len(aceID) == 40 {
						// Extract display name
						name := rewriter.ExtractDisplayName(metadata)

						// Extract metadata attributes
						tvgID, tvgLogo, groupTitle := extractMetadata(metadata)

						channels = append(channels, Channel{
							AcestreamID: aceID,
							Name:        name,
							TvgID:       tvgID,
							TvgLogo:     tvgLogo,
							GroupTitle:  groupTitle,
							Source:      source,
							Enabled:     true, // Default, will be updated with overrides
							HasOverride: false,
						})
					}
				}
			}
		}
	}

	return channels
}

// applyOverrides applies override settings to channels and marks which ones have overrides
func applyOverrides(channels []Channel, overridesMgr *overrides.Manager) []Channel {
	allOverrides := overridesMgr.List()

	for i := range channels {
		ch := &channels[i]
		if override, exists := allOverrides[ch.AcestreamID]; exists {
			ch.HasOverride = true

			// Apply overrides if they are set (not nil)
			if override.Enabled != nil {
				ch.Enabled = *override.Enabled
			}
			if override.TvgID != nil {
				ch.TvgID = *override.TvgID
			}
			if override.TvgLogo != nil {
				ch.TvgLogo = *override.TvgLogo
			}
			if override.GroupTitle != nil {
				ch.GroupTitle = *override.GroupTitle
			}
		}
	}

	return channels
}

// deduplicateChannels removes duplicate channels by acestream ID, keeping the first occurrence
func deduplicateChannels(channels []Channel) []Channel {
	seen := make(map[string]bool)
	var result []Channel

	for _, ch := range channels {
		if !seen[ch.AcestreamID] {
			seen[ch.AcestreamID] = true
			result = append(result, ch)
		}
	}

	return result
}

// ServeHTTP handles the GET /api/channels request
func (h *ChannelsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Fetch both sources
	elcanoContent, _, _, elcanoErr := h.fetcher.FetchWithCache(h.elcanoURL)
	neweraContent, _, _, neweraErr := h.fetcher.FetchWithCache(h.neweraURL)

	// Check if both sources failed
	if elcanoErr != nil && neweraErr != nil {
		log.Printf("Failed to fetch channels - both sources failed: elcano=%v, newera=%v", elcanoErr, neweraErr)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}

	// Parse channels from both sources
	var allChannels []Channel

	if elcanoErr == nil {
		elcanoChannels := parseM3UChannels(elcanoContent, "elcano")
		allChannels = append(allChannels, elcanoChannels...)
	} else {
		log.Printf("Skipping elcano source: %v", elcanoErr)
	}

	if neweraErr == nil {
		neweraChannels := parseM3UChannels(neweraContent, "newera")
		allChannels = append(allChannels, neweraChannels...)
	} else {
		log.Printf("Skipping newera source: %v", neweraErr)
	}

	// Deduplicate channels by acestream ID
	allChannels = deduplicateChannels(allChannels)

	// Apply overrides
	allChannels = applyOverrides(allChannels, h.overridesMgr)

	// Sort alphabetically by name (case-insensitive)
	sort.SliceStable(allChannels, func(i, j int) bool {
		return strings.ToLower(allChannels[i].Name) < strings.ToLower(allChannels[j].Name)
	})

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(allChannels); err != nil {
		log.Printf("Failed to encode channels response: %v", err)
	}
}
