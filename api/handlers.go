package api

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"github.com/alorle/iptv-manager/domain"
	"github.com/alorle/iptv-manager/logging"
	"github.com/alorle/iptv-manager/overrides"
)

// ServeHTTP handles the GET /api/channels request, PATCH /api/channels/{content_id}, and DELETE /api/channels/{content_id}/override
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

	logging.WriteJSONError(w, h.logger, "Method not allowed", http.StatusMethodNotAllowed, map[string]interface{}{
		"method": r.Method,
		"path":   r.URL.Path,
	})
}

// handleList handles the GET /api/channels request
func (h *ChannelsHandler) handleList(w http.ResponseWriter, r *http.Request) {
	// Extract query parameters for filtering
	nameFilter := r.URL.Query().Get("name")
	groupFilter := r.URL.Query().Get("group")

	// Fetch all configured sources
	var allStreams []streamData
	allFailed := true

	for i, url := range h.playlistURLs {
		sourceName := domain.GetSourceName(url, i)
		content, _, _, err := h.fetcher.FetchWithCache(url)

		if err == nil {
			streams := parseM3UStreams(content, sourceName)
			allStreams = append(allStreams, streams...)
			allFailed = false
		} else {
			h.logger.Warn("Skipping source", map[string]interface{}{
				"method": r.Method,
				"path":   r.URL.Path,
				"source": sourceName,
				"error":  err.Error(),
			})
		}
	}

	// Check if all sources failed
	if allFailed {
		logging.WriteJSONError(w, h.logger, "Bad Gateway", http.StatusBadGateway, map[string]interface{}{
			"method": r.Method,
			"path":   r.URL.Path,
			"reason": "all sources failed",
		})
		return
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
		h.logger.Warn("Failed to encode channels response", map[string]interface{}{
			"method": r.Method,
			"path":   r.URL.Path,
			"error":  err.Error(),
		})
	}
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

// fetchSourcesContent fetches content from all configured sources
func (h *ChannelsHandler) fetchSourcesContent() []struct {
	content []byte
	err     error
	name    string
} {
	results := make([]struct {
		content []byte
		err     error
		name    string
	}, len(h.playlistURLs))

	for i, url := range h.playlistURLs {
		content, _, _, err := h.fetcher.FetchWithCache(url)
		results[i] = struct {
			content []byte
			err     error
			name    string
		}{
			content: content,
			err:     err,
			name:    domain.GetSourceName(url, i),
		}
	}

	return results
}

// findStreamByID searches for a stream by acestream ID in the provided sources
func findStreamByID(contentID string, sources []struct {
	content []byte
	err     error
	name    string
}) *streamData {
	for _, source := range sources {
		if source.err == nil {
			streams := parseM3UStreams(source.content, source.name)
			for _, s := range streams {
				if s.ContentID == contentID {
					return &s
				}
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

// handleToggle handles the PATCH /api/channels/{content_id} request
func (h *ChannelsHandler) handleToggle(w http.ResponseWriter, r *http.Request) {
	// Extract and validate content_id
	path := strings.TrimPrefix(r.URL.Path, "/api/channels/")
	contentID := strings.TrimSpace(path)

	if !domain.IsValidAcestreamID(contentID) {
		logging.WriteJSONError(w, h.logger, "Invalid content_id: must be 40 hexadecimal characters", http.StatusBadRequest, map[string]interface{}{
			"method":     r.Method,
			"path":       r.URL.Path,
			"content_id": contentID,
		})
		return
	}

	// Parse and validate request body
	var req UpdateChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logging.WriteJSONError(w, h.logger, "Invalid request body", http.StatusBadRequest, map[string]interface{}{
			"method":     r.Method,
			"path":       r.URL.Path,
			"content_id": contentID,
			"error":      err.Error(),
		})
		return
	}

	if !validateUpdateRequest(&req) {
		logging.WriteJSONError(w, h.logger, "tvg_id and tvg_name cannot be empty", http.StatusBadRequest, map[string]interface{}{
			"method":     r.Method,
			"path":       r.URL.Path,
			"content_id": contentID,
		})
		return
	}

	// Fetch sources
	sources := h.fetchSourcesContent()
	allFailed := true
	for _, src := range sources {
		if src.err == nil {
			allFailed = false
			break
		}
	}

	if allFailed {
		logging.WriteJSONError(w, h.logger, "Bad Gateway", http.StatusBadGateway, map[string]interface{}{
			"method":     r.Method,
			"path":       r.URL.Path,
			"content_id": contentID,
			"reason":     "all sources failed",
		})
		return
	}

	// Verify stream exists
	if stream := findStreamByID(contentID, sources); stream == nil {
		logging.WriteJSONError(w, h.logger, "Stream not found", http.StatusNotFound, map[string]interface{}{
			"method":     r.Method,
			"path":       r.URL.Path,
			"content_id": contentID,
		})
		return
	}

	// Merge and save override
	existingOverride := h.overridesMgr.Get(contentID)
	override := mergeOverrideWithRequest(existingOverride, &req)

	if err := h.overridesMgr.Set(contentID, override); err != nil {
		logging.WriteJSONError(w, h.logger, "Internal Server Error", http.StatusInternalServerError, map[string]interface{}{
			"method":     r.Method,
			"path":       r.URL.Path,
			"content_id": contentID,
			"error":      err.Error(),
		})
		return
	}

	h.logger.Info("Updated stream with overrides", map[string]interface{}{
		"method":     r.Method,
		"path":       r.URL.Path,
		"content_id": contentID,
	})

	// Build and return response
	updatedStream := findStreamByID(contentID, sources)
	if updatedStream != nil {
		applyOverrideToStream(updatedStream, override)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(updatedStream); err != nil {
		h.logger.Warn("Failed to encode stream response", map[string]interface{}{
			"method":     r.Method,
			"path":       r.URL.Path,
			"content_id": contentID,
			"error":      err.Error(),
		})
	}
}

// handleDelete handles the DELETE /api/channels/{content_id}/override request
func (h *ChannelsHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	// Extract and validate content_id
	path := strings.TrimPrefix(r.URL.Path, "/api/channels/")
	path = strings.TrimSuffix(path, "/override")
	contentID := strings.TrimSpace(path)

	if !domain.IsValidAcestreamID(contentID) {
		logging.WriteJSONError(w, h.logger, "Invalid content_id: must be 40 hexadecimal characters", http.StatusBadRequest, map[string]interface{}{
			"method":     r.Method,
			"path":       r.URL.Path,
			"content_id": contentID,
		})
		return
	}

	// Check if override exists
	existingOverride := h.overridesMgr.Get(contentID)
	if existingOverride == nil {
		logging.WriteJSONError(w, h.logger, "No override found for this content_id", http.StatusNotFound, map[string]interface{}{
			"method":     r.Method,
			"path":       r.URL.Path,
			"content_id": contentID,
		})
		return
	}

	// Fetch sources and verify stream exists
	sources := h.fetchSourcesContent()
	allFailed := true
	for _, src := range sources {
		if src.err == nil {
			allFailed = false
			break
		}
	}

	if allFailed {
		logging.WriteJSONError(w, h.logger, "Bad Gateway", http.StatusBadGateway, map[string]interface{}{
			"method":     r.Method,
			"path":       r.URL.Path,
			"content_id": contentID,
			"reason":     "all sources failed",
		})
		return
	}

	originalStream := findStreamByID(contentID, sources)
	if originalStream == nil {
		logging.WriteJSONError(w, h.logger, "Stream not found", http.StatusNotFound, map[string]interface{}{
			"method":     r.Method,
			"path":       r.URL.Path,
			"content_id": contentID,
		})
		return
	}

	// Delete the override
	if err := h.overridesMgr.Delete(contentID); err != nil {
		logging.WriteJSONError(w, h.logger, "Internal Server Error", http.StatusInternalServerError, map[string]interface{}{
			"method":     r.Method,
			"path":       r.URL.Path,
			"content_id": contentID,
			"error":      err.Error(),
		})
		return
	}

	h.logger.Info("Deleted override for stream", map[string]interface{}{
		"method":     r.Method,
		"path":       r.URL.Path,
		"content_id": contentID,
	})

	// Return the stream in its original state
	originalStream.HasOverride = false

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(originalStream); err != nil {
		h.logger.Warn("Failed to encode stream response", map[string]interface{}{
			"method":     r.Method,
			"path":       r.URL.Path,
			"content_id": contentID,
			"error":      err.Error(),
		})
	}
}
