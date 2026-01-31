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
	TvgName     string `json:"tvg_name"`
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
			if override.TvgName != nil {
				ch.TvgName = *override.TvgName
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

// ServeHTTP handles the GET /api/channels request, PATCH /api/channels/{acestream_id}, and DELETE /api/channels/{acestream_id}/override
func (h *ChannelsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		h.handleList(w, r)
		return
	}

	if r.Method == http.MethodPatch {
		h.handleToggle(w, r)
		return
	}

	if r.Method == http.MethodDelete {
		h.handleDelete(w, r)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// handleList handles the GET /api/channels request
func (h *ChannelsHandler) handleList(w http.ResponseWriter, r *http.Request) {

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

// UpdateChannelRequest represents the request body for updating a channel's metadata
type UpdateChannelRequest struct {
	Enabled    *bool   `json:"enabled,omitempty"`
	TvgID      *string `json:"tvg_id,omitempty"`
	TvgName    *string `json:"tvg_name,omitempty"`
	TvgLogo    *string `json:"tvg_logo,omitempty"`
	GroupTitle *string `json:"group_title,omitempty"`
}

// handleToggle handles the PATCH /api/channels/{acestream_id} request
func (h *ChannelsHandler) handleToggle(w http.ResponseWriter, r *http.Request) {
	// Extract acestream_id from the URL path
	path := strings.TrimPrefix(r.URL.Path, "/api/channels/")
	acestreamID := strings.TrimSpace(path)

	// Validate acestream ID format (40 hex characters)
	if len(acestreamID) != 40 {
		http.Error(w, "Invalid acestream_id: must be 40 characters", http.StatusBadRequest)
		return
	}

	// Validate that it's hexadecimal
	for _, c := range acestreamID {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			http.Error(w, "Invalid acestream_id: must be hexadecimal", http.StatusBadRequest)
			return
		}
	}

	// Parse request body
	var req UpdateChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate tvg_id and tvg_name are not empty if provided
	if req.TvgID != nil && strings.TrimSpace(*req.TvgID) == "" {
		http.Error(w, "tvg_id cannot be empty", http.StatusBadRequest)
		return
	}

	if req.TvgName != nil && strings.TrimSpace(*req.TvgName) == "" {
		http.Error(w, "tvg_name cannot be empty", http.StatusBadRequest)
		return
	}

	// Check if the channel exists in any source
	elcanoContent, _, _, elcanoErr := h.fetcher.FetchWithCache(h.elcanoURL)
	neweraContent, _, _, neweraErr := h.fetcher.FetchWithCache(h.neweraURL)

	// Check if both sources failed
	if elcanoErr != nil && neweraErr != nil {
		log.Printf("Failed to fetch channels - both sources failed: elcano=%v, newera=%v", elcanoErr, neweraErr)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}

	// Check if the acestream_id exists in any source
	channelFound := false

	if elcanoErr == nil {
		elcanoChannels := parseM3UChannels(elcanoContent, "elcano")
		for _, ch := range elcanoChannels {
			if ch.AcestreamID == acestreamID {
				channelFound = true
				break
			}
		}
	}

	if !channelFound && neweraErr == nil {
		neweraChannels := parseM3UChannels(neweraContent, "newera")
		for _, ch := range neweraChannels {
			if ch.AcestreamID == acestreamID {
				channelFound = true
				break
			}
		}
	}

	if !channelFound {
		http.Error(w, "Channel not found", http.StatusNotFound)
		return
	}

	// Get existing override or create a new one
	existingOverride := h.overridesMgr.Get(acestreamID)
	var override overrides.ChannelOverride

	if existingOverride != nil {
		// Copy existing override
		override = *existingOverride
	}

	// Perform partial merge - only update fields that are present in the request
	if req.Enabled != nil {
		override.Enabled = req.Enabled
	}

	if req.TvgID != nil {
		override.TvgID = req.TvgID
	}

	if req.TvgName != nil {
		override.TvgName = req.TvgName
	}

	if req.TvgLogo != nil {
		override.TvgLogo = req.TvgLogo
	}

	if req.GroupTitle != nil {
		override.GroupTitle = req.GroupTitle
	}

	// Save the override to disk immediately
	if err := h.overridesMgr.Set(acestreamID, override); err != nil {
		log.Printf("Failed to save override for %s: %v", acestreamID, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	log.Printf("Updated channel %s with overrides", acestreamID)

	// Return the updated channel
	// Fetch channels again to get the updated state
	var updatedChannel *Channel

	if elcanoErr == nil {
		elcanoChannels := parseM3UChannels(elcanoContent, "elcano")
		for _, ch := range elcanoChannels {
			if ch.AcestreamID == acestreamID {
				updatedChannel = &ch
				break
			}
		}
	}

	if updatedChannel == nil && neweraErr == nil {
		neweraChannels := parseM3UChannels(neweraContent, "newera")
		for _, ch := range neweraChannels {
			if ch.AcestreamID == acestreamID {
				updatedChannel = &ch
				break
			}
		}
	}

	// Apply all overrides to the channel
	if updatedChannel != nil {
		updatedChannel.HasOverride = true

		if override.Enabled != nil {
			updatedChannel.Enabled = *override.Enabled
		}
		if override.TvgID != nil {
			updatedChannel.TvgID = *override.TvgID
		}
		if override.TvgName != nil {
			updatedChannel.TvgName = *override.TvgName
		}
		if override.TvgLogo != nil {
			updatedChannel.TvgLogo = *override.TvgLogo
		}
		if override.GroupTitle != nil {
			updatedChannel.GroupTitle = *override.GroupTitle
		}
	}

	// Return the updated channel as JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(updatedChannel); err != nil {
		log.Printf("Failed to encode channel response: %v", err)
	}
}

// handleDelete handles the DELETE /api/channels/{acestream_id}/override request
func (h *ChannelsHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	// Extract acestream_id from the URL path
	// Expected format: /api/channels/{acestream_id}/override
	path := strings.TrimPrefix(r.URL.Path, "/api/channels/")
	path = strings.TrimSuffix(path, "/override")
	acestreamID := strings.TrimSpace(path)

	// Validate acestream ID format (40 hex characters)
	if len(acestreamID) != 40 {
		http.Error(w, "Invalid acestream_id: must be 40 characters", http.StatusBadRequest)
		return
	}

	// Validate that it's hexadecimal
	for _, c := range acestreamID {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			http.Error(w, "Invalid acestream_id: must be hexadecimal", http.StatusBadRequest)
			return
		}
	}

	// Check if override exists for this acestream_id
	existingOverride := h.overridesMgr.Get(acestreamID)
	if existingOverride == nil {
		http.Error(w, "No override found for this acestream_id", http.StatusNotFound)
		return
	}

	// Check if the channel exists in any source
	elcanoContent, _, _, elcanoErr := h.fetcher.FetchWithCache(h.elcanoURL)
	neweraContent, _, _, neweraErr := h.fetcher.FetchWithCache(h.neweraURL)

	// Check if both sources failed
	if elcanoErr != nil && neweraErr != nil {
		log.Printf("Failed to fetch channels - both sources failed: elcano=%v, newera=%v", elcanoErr, neweraErr)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}

	// Check if the acestream_id exists in any source
	channelFound := false
	var originalChannel *Channel

	if elcanoErr == nil {
		elcanoChannels := parseM3UChannels(elcanoContent, "elcano")
		for _, ch := range elcanoChannels {
			if ch.AcestreamID == acestreamID {
				channelFound = true
				originalChannel = &ch
				break
			}
		}
	}

	if !channelFound && neweraErr == nil {
		neweraChannels := parseM3UChannels(neweraContent, "newera")
		for _, ch := range neweraChannels {
			if ch.AcestreamID == acestreamID {
				channelFound = true
				originalChannel = &ch
				break
			}
		}
	}

	if !channelFound {
		http.Error(w, "Channel not found", http.StatusNotFound)
		return
	}

	// Delete the override
	if err := h.overridesMgr.Delete(acestreamID); err != nil {
		log.Printf("Failed to delete override for %s: %v", acestreamID, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	log.Printf("Deleted override for channel %s", acestreamID)

	// Return the channel in its original state (without override)
	originalChannel.HasOverride = false

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(originalChannel); err != nil {
		log.Printf("Failed to encode channel response: %v", err)
	}
}
