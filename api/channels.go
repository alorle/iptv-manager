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

// Stream represents a single stream within a channel
type Stream struct {
	AcestreamID string `json:"acestream_id"`
	Name        string `json:"name"`
	TvgName     string `json:"tvg_name"`
	Source      string `json:"source"` // "elcano" or "newera"
	Enabled     bool   `json:"enabled"`
	HasOverride bool   `json:"has_override"`
}

// Channel represents a channel with its metadata and array of streams
type Channel struct {
	Name       string   `json:"name"`
	TvgID      string   `json:"tvg_id"`
	TvgLogo    string   `json:"tvg_logo"`
	GroupTitle string   `json:"group_title"`
	Streams    []Stream `json:"streams"`
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

// streamData holds raw parsed stream data before grouping
type streamData struct {
	AcestreamID string
	Name        string
	TvgID       string
	TvgName     string
	TvgLogo     string
	GroupTitle  string
	Source      string
	Enabled     bool
	HasOverride bool
}

// parseM3UStreams parses M3U content and extracts stream information
func parseM3UStreams(content []byte, source string) []streamData {
	lines := strings.Split(string(content), "\n")
	var streams []streamData

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

						streams = append(streams, streamData{
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

	return streams
}

// isValidTvgID checks if a tvg-id is valid (not empty or whitespace-only)
func isValidTvgID(tvgID string) bool {
	return strings.TrimSpace(tvgID) != ""
}

// groupStreamsByTvgID groups streams by their tvg-id
// Streams with empty/whitespace tvg-id are returned as individual channels
func groupStreamsByTvgID(streams []streamData) []Channel {
	var channels []Channel

	// Group streams by valid tvg-id
	grouped := make(map[string][]streamData)
	var ungrouped []streamData

	for _, stream := range streams {
		if isValidTvgID(stream.TvgID) {
			grouped[stream.TvgID] = append(grouped[stream.TvgID], stream)
		} else {
			ungrouped = append(ungrouped, stream)
		}
	}

	// Create channels from grouped streams
	for tvgID, streamList := range grouped {
		// Use first stream's metadata for the channel
		first := streamList[0]

		var streamObjs []Stream
		for _, s := range streamList {
			streamObjs = append(streamObjs, Stream{
				AcestreamID: s.AcestreamID,
				Name:        s.Name,
				TvgName:     s.TvgName,
				Source:      s.Source,
				Enabled:     s.Enabled,
				HasOverride: s.HasOverride,
			})
		}

		channels = append(channels, Channel{
			Name:       first.Name,
			TvgID:      tvgID,
			TvgLogo:    first.TvgLogo,
			GroupTitle: first.GroupTitle,
			Streams:    streamObjs,
		})
	}

	// Create individual channels for ungrouped streams
	for _, stream := range ungrouped {
		channels = append(channels, Channel{
			Name:       stream.Name,
			TvgID:      stream.TvgID,
			TvgLogo:    stream.TvgLogo,
			GroupTitle: stream.GroupTitle,
			Streams: []Stream{
				{
					AcestreamID: stream.AcestreamID,
					Name:        stream.Name,
					TvgName:     stream.TvgName,
					Source:      stream.Source,
					Enabled:     stream.Enabled,
					HasOverride: stream.HasOverride,
				},
			},
		})
	}

	return channels
}

// applyOverrides applies override settings to channels and marks which ones have overrides
func applyOverrides(channels []Channel, overridesMgr *overrides.Manager) []Channel {
	allOverrides := overridesMgr.List()

	for i := range channels {
		ch := &channels[i]

		// Apply overrides to each stream in the channel
		for j := range ch.Streams {
			stream := &ch.Streams[j]
			if override, exists := allOverrides[stream.AcestreamID]; exists {
				stream.HasOverride = true

				// Apply overrides if they are set (not nil)
				if override.Enabled != nil {
					stream.Enabled = *override.Enabled
				}
				if override.TvgName != nil {
					stream.TvgName = *override.TvgName
				}
			}
		}

		// Apply channel-level overrides from the first stream if it has overrides
		if len(ch.Streams) > 0 {
			firstStream := ch.Streams[0]
			if override, exists := allOverrides[firstStream.AcestreamID]; exists {
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
	}

	return channels
}

// filterChannels filters channels by name and/or group (case-insensitive substring match)
func filterChannels(channels []Channel, nameFilter, groupFilter string) []Channel {
	var result []Channel

	nameLower := strings.ToLower(nameFilter)
	groupLower := strings.ToLower(groupFilter)

	for _, ch := range channels {
		nameMatches := nameFilter == "" || strings.Contains(strings.ToLower(ch.Name), nameLower)
		groupMatches := groupFilter == "" || strings.Contains(strings.ToLower(ch.GroupTitle), groupLower)

		if nameMatches && groupMatches {
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
	// Extract query parameters for filtering
	nameFilter := r.URL.Query().Get("name")
	groupFilter := r.URL.Query().Get("group")

	// Fetch both sources
	elcanoContent, _, _, elcanoErr := h.fetcher.FetchWithCache(h.elcanoURL)
	neweraContent, _, _, neweraErr := h.fetcher.FetchWithCache(h.neweraURL)

	// Check if both sources failed
	if elcanoErr != nil && neweraErr != nil {
		log.Printf("Failed to fetch channels - both sources failed: elcano=%v, newera=%v", elcanoErr, neweraErr)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}

	// Parse streams from both sources
	var allStreams []streamData

	if elcanoErr == nil {
		elcanoStreams := parseM3UStreams(elcanoContent, "elcano")
		allStreams = append(allStreams, elcanoStreams...)
	} else {
		log.Printf("Skipping elcano source: %v", elcanoErr)
	}

	if neweraErr == nil {
		neweraStreams := parseM3UStreams(neweraContent, "newera")
		allStreams = append(allStreams, neweraStreams...)
	} else {
		log.Printf("Skipping newera source: %v", neweraErr)
	}

	// Group streams by tvg-id into channels
	allChannels := groupStreamsByTvgID(allStreams)

	// Apply overrides
	allChannels = applyOverrides(allChannels, h.overridesMgr)

	// Apply filters if provided
	if nameFilter != "" || groupFilter != "" {
		allChannels = filterChannels(allChannels, nameFilter, groupFilter)
	}

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

// validateAcestreamID validates that an acestream ID is 40 hexadecimal characters
func validateAcestreamID(acestreamID string) bool {
	if len(acestreamID) != 40 {
		return false
	}
	for _, c := range acestreamID {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// validateUpdateRequest validates the update channel request fields
func validateUpdateRequest(req *UpdateChannelRequest) bool {
	if req.TvgID != nil && strings.TrimSpace(*req.TvgID) == "" {
		return false
	}
	if req.TvgName != nil && strings.TrimSpace(*req.TvgName) == "" {
		return false
	}
	return true
}

// fetchSourcesContent fetches content from both elcano and newera sources
func (h *ChannelsHandler) fetchSourcesContent() (elcanoContent, neweraContent []byte, elcanoErr, neweraErr error) {
	elcanoContent, _, _, elcanoErr = h.fetcher.FetchWithCache(h.elcanoURL)
	neweraContent, _, _, neweraErr = h.fetcher.FetchWithCache(h.neweraURL)
	return
}

// findStreamByID searches for a stream by acestream ID in the provided sources
func findStreamByID(acestreamID string, elcanoContent, neweraContent []byte, elcanoErr, neweraErr error) *streamData {
	if elcanoErr == nil {
		elcanoStreams := parseM3UStreams(elcanoContent, "elcano")
		for _, s := range elcanoStreams {
			if s.AcestreamID == acestreamID {
				return &s
			}
		}
	}

	if neweraErr == nil {
		neweraStreams := parseM3UStreams(neweraContent, "newera")
		for _, s := range neweraStreams {
			if s.AcestreamID == acestreamID {
				return &s
			}
		}
	}

	return nil
}

// mergeOverrideWithRequest merges the request fields into the existing override
func mergeOverrideWithRequest(existing *overrides.ChannelOverride, req *UpdateChannelRequest) overrides.ChannelOverride {
	var override overrides.ChannelOverride
	if existing != nil {
		override = *existing
	}

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

	return override
}

// applyOverrideToStream applies override settings to a stream
func applyOverrideToStream(stream *streamData, override overrides.ChannelOverride) {
	stream.HasOverride = true

	if override.Enabled != nil {
		stream.Enabled = *override.Enabled
	}
	if override.TvgID != nil {
		stream.TvgID = *override.TvgID
	}
	if override.TvgName != nil {
		stream.TvgName = *override.TvgName
	}
	if override.TvgLogo != nil {
		stream.TvgLogo = *override.TvgLogo
	}
	if override.GroupTitle != nil {
		stream.GroupTitle = *override.GroupTitle
	}
}

// handleToggle handles the PATCH /api/channels/{acestream_id} request
func (h *ChannelsHandler) handleToggle(w http.ResponseWriter, r *http.Request) {
	// Extract and validate acestream_id
	path := strings.TrimPrefix(r.URL.Path, "/api/channels/")
	acestreamID := strings.TrimSpace(path)

	if !validateAcestreamID(acestreamID) {
		http.Error(w, "Invalid acestream_id: must be 40 hexadecimal characters", http.StatusBadRequest)
		return
	}

	// Parse and validate request body
	var req UpdateChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if !validateUpdateRequest(&req) {
		http.Error(w, "tvg_id and tvg_name cannot be empty", http.StatusBadRequest)
		return
	}

	// Fetch sources
	elcanoContent, neweraContent, elcanoErr, neweraErr := h.fetchSourcesContent()
	if elcanoErr != nil && neweraErr != nil {
		log.Printf("Failed to fetch channels - both sources failed: elcano=%v, newera=%v", elcanoErr, neweraErr)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}

	// Verify stream exists
	if stream := findStreamByID(acestreamID, elcanoContent, neweraContent, elcanoErr, neweraErr); stream == nil {
		http.Error(w, "Stream not found", http.StatusNotFound)
		return
	}

	// Merge and save override
	existingOverride := h.overridesMgr.Get(acestreamID)
	override := mergeOverrideWithRequest(existingOverride, &req)

	if err := h.overridesMgr.Set(acestreamID, override); err != nil {
		log.Printf("Failed to save override for %s: %v", acestreamID, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	log.Printf("Updated stream %s with overrides", acestreamID)

	// Build and return response
	updatedStream := findStreamByID(acestreamID, elcanoContent, neweraContent, elcanoErr, neweraErr)
	if updatedStream != nil {
		applyOverrideToStream(updatedStream, override)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(updatedStream); err != nil {
		log.Printf("Failed to encode stream response: %v", err)
	}
}

// handleDelete handles the DELETE /api/channels/{acestream_id}/override request
func (h *ChannelsHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	// Extract and validate acestream_id
	path := strings.TrimPrefix(r.URL.Path, "/api/channels/")
	path = strings.TrimSuffix(path, "/override")
	acestreamID := strings.TrimSpace(path)

	if !validateAcestreamID(acestreamID) {
		http.Error(w, "Invalid acestream_id: must be 40 hexadecimal characters", http.StatusBadRequest)
		return
	}

	// Check if override exists
	existingOverride := h.overridesMgr.Get(acestreamID)
	if existingOverride == nil {
		http.Error(w, "No override found for this acestream_id", http.StatusNotFound)
		return
	}

	// Fetch sources and verify stream exists
	elcanoContent, neweraContent, elcanoErr, neweraErr := h.fetchSourcesContent()
	if elcanoErr != nil && neweraErr != nil {
		log.Printf("Failed to fetch channels - both sources failed: elcano=%v, newera=%v", elcanoErr, neweraErr)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}

	originalStream := findStreamByID(acestreamID, elcanoContent, neweraContent, elcanoErr, neweraErr)
	if originalStream == nil {
		http.Error(w, "Stream not found", http.StatusNotFound)
		return
	}

	// Delete the override
	if err := h.overridesMgr.Delete(acestreamID); err != nil {
		log.Printf("Failed to delete override for %s: %v", acestreamID, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	log.Printf("Deleted override for stream %s", acestreamID)

	// Return the stream in its original state
	originalStream.HasOverride = false

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(originalStream); err != nil {
		log.Printf("Failed to encode stream response: %v", err)
	}
}
