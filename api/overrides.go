package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/alorle/iptv-manager/domain"
	"github.com/alorle/iptv-manager/logging"
	"github.com/alorle/iptv-manager/overrides"
)

// OverridesHandler handles CRUD operations for channel overrides
type OverridesHandler struct {
	overridesMgr overrides.Interface
	epgCache     TVGValidator
	logger       *logging.Logger
}

// NewOverridesHandler creates a new handler for the overrides API
func NewOverridesHandler(overridesMgr overrides.Interface, epgCache TVGValidator, logger *logging.Logger) *OverridesHandler {
	return &OverridesHandler{
		overridesMgr: overridesMgr,
		epgCache:     epgCache,
		logger:       logger,
	}
}

// ServeHTTP handles all override-related requests
func (h *OverridesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract path after /api/overrides
	path := strings.TrimPrefix(r.URL.Path, "/api/overrides")
	path = strings.Trim(path, "/")

	// Route based on path and method
	if path == "" {
		// /api/overrides - list all overrides (GET)
		if r.Method == http.MethodGet {
			h.handleList(w, r)
			return
		}
		logging.WriteJSONError(w, h.logger, "Method not allowed", http.StatusMethodNotAllowed, map[string]interface{}{
			"method": r.Method,
			"path":   r.URL.Path,
		})
		return
	}

	// /api/overrides/bulk - bulk update operations
	if path == "bulk" {
		if r.Method == http.MethodPatch {
			h.handleBulkUpdate(w, r)
			return
		}
		logging.WriteJSONError(w, h.logger, "Method not allowed", http.StatusMethodNotAllowed, map[string]interface{}{
			"method": r.Method,
			"path":   r.URL.Path,
		})
		return
	}

	// /api/overrides/:contentId - single override operations
	contentID := path

	// Validate content ID format
	if !domain.IsValidAcestreamID(contentID) {
		logging.WriteJSONError(w, h.logger, "Invalid content_id: must be 40 hexadecimal characters", http.StatusBadRequest, map[string]interface{}{
			"method":     r.Method,
			"path":       r.URL.Path,
			"content_id": contentID,
		})
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.handleGet(w, r, contentID)
	case http.MethodPut:
		h.handlePut(w, r, contentID)
	case http.MethodDelete:
		h.handleDelete(w, r, contentID)
	default:
		logging.WriteJSONError(w, h.logger, "Method not allowed", http.StatusMethodNotAllowed, map[string]interface{}{
			"method":     r.Method,
			"path":       r.URL.Path,
			"content_id": contentID,
		})
	}
}

// OverrideResponse represents the response for a single override
type OverrideResponse struct {
	ContentID string                     `json:"acestream_id"`
	Override  *overrides.ChannelOverride `json:"override"`
}

// handleGet handles GET /api/overrides/:contentId
func (h *OverridesHandler) handleGet(w http.ResponseWriter, r *http.Request, contentID string) {
	override := h.overridesMgr.Get(contentID)
	if override == nil {
		logging.WriteJSONError(w, h.logger, "Override not found", http.StatusNotFound, map[string]interface{}{
			"method":     r.Method,
			"path":       r.URL.Path,
			"content_id": contentID,
		})
		return
	}

	response := OverrideResponse{
		ContentID: contentID,
		Override:  override,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Warn("Failed to encode override response", map[string]interface{}{
			"method":     r.Method,
			"path":       r.URL.Path,
			"content_id": contentID,
			"error":      err.Error(),
		})
	}
}

// ValidationError represents a validation error response
type ValidationError struct {
	Error       string   `json:"error"`
	Field       string   `json:"field,omitempty"`
	Message     string   `json:"message"`
	Suggestions []string `json:"suggestions,omitempty"`
}

// handlePut handles PUT /api/overrides/:contentId
func (h *OverridesHandler) handlePut(w http.ResponseWriter, r *http.Request, contentID string) {
	// Check for force parameter
	force := r.URL.Query().Get("force") == "true"

	// Parse request body
	var override overrides.ChannelOverride
	if err := json.NewDecoder(r.Body).Decode(&override); err != nil {
		logging.WriteJSONError(w, h.logger, "Invalid request body", http.StatusBadRequest, map[string]interface{}{
			"method":     r.Method,
			"path":       r.URL.Path,
			"content_id": contentID,
			"error":      err.Error(),
		})
		return
	}

	// Validate TVG-ID if it's being set and EPG cache is available
	if !force && h.epgCache != nil && override.TvgID != nil {
		tvgID := strings.TrimSpace(*override.TvgID)

		// Empty TVG-ID is valid (means "no EPG")
		if tvgID != "" && !h.epgCache.IsValid(tvgID) {
			// Get suggestions for invalid TVG-ID
			suggestions := h.getSuggestions(tvgID, 10)

			h.logger.Warn("TVG-ID validation failed", map[string]interface{}{
				"method":     r.Method,
				"path":       r.URL.Path,
				"content_id": contentID,
				"tvg_id":     tvgID,
			})

			validationErr := ValidationError{
				Error:       "validation_error",
				Field:       "tvg_id",
				Message:     "TVG-ID not found in EPG data",
				Suggestions: suggestions,
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			if err := json.NewEncoder(w).Encode(validationErr); err != nil {
				h.logger.Warn("Failed to encode validation error", map[string]interface{}{
					"method":     r.Method,
					"path":       r.URL.Path,
					"content_id": contentID,
					"error":      err.Error(),
				})
			}
			return
		}
	}

	// Save the override
	if err := h.overridesMgr.Set(contentID, override); err != nil {
		logging.WriteJSONError(w, h.logger, "Internal Server Error", http.StatusInternalServerError, map[string]interface{}{
			"method":     r.Method,
			"path":       r.URL.Path,
			"content_id": contentID,
			"error":      err.Error(),
		})
		return
	}

	h.logger.Info("Created/updated override", map[string]interface{}{
		"method":     r.Method,
		"path":       r.URL.Path,
		"content_id": contentID,
	})

	// Return the created/updated override
	response := OverrideResponse{
		ContentID: contentID,
		Override:  &override,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Warn("Failed to encode override response", map[string]interface{}{
			"method":     r.Method,
			"path":       r.URL.Path,
			"content_id": contentID,
			"error":      err.Error(),
		})
	}
}

// handleDelete handles DELETE /api/overrides/:contentId
func (h *OverridesHandler) handleDelete(w http.ResponseWriter, r *http.Request, contentID string) {
	// Check if override exists
	if h.overridesMgr.Get(contentID) == nil {
		logging.WriteJSONError(w, h.logger, "Override not found", http.StatusNotFound, map[string]interface{}{
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

	h.logger.Info("Deleted override", map[string]interface{}{
		"method":     r.Method,
		"path":       r.URL.Path,
		"content_id": contentID,
	})

	w.WriteHeader(http.StatusNoContent)
}

// handleList handles GET /api/overrides
func (h *OverridesHandler) handleList(w http.ResponseWriter, r *http.Request) {
	allOverrides := h.overridesMgr.List()

	// Convert map to array of responses
	var responses []OverrideResponse
	for contentID, override := range allOverrides {
		// Create a copy of the override for the response
		overrideCopy := override
		responses = append(responses, OverrideResponse{
			ContentID: contentID,
			Override:  &overrideCopy,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(responses); err != nil {
		h.logger.Warn("Failed to encode overrides list", map[string]interface{}{
			"method": r.Method,
			"path":   r.URL.Path,
			"error":  err.Error(),
		})
	}
}

// getSuggestions returns up to maxResults closest matches for a TVG-ID
func (h *OverridesHandler) getSuggestions(tvgID string, maxResults int) []string {
	if h.epgCache == nil {
		return nil
	}

	matches := h.epgCache.Search(tvgID, maxResults)
	var suggestions []string
	for _, match := range matches {
		suggestions = append(suggestions, match.ID)
	}
	return suggestions
}

// BulkUpdateRequest represents a request to update multiple overrides
type BulkUpdateRequest struct {
	ContentIDs []string    `json:"acestream_ids"` // Note: keep JSON tag as acestream_ids for API compatibility
	Field      string      `json:"field"`
	Value      interface{} `json:"value"`
}

// BulkUpdateResponse represents the response for a bulk update operation
type BulkUpdateResponse struct {
	Updated int                         `json:"updated"`
	Failed  int                         `json:"failed"`
	Errors  []overrides.BulkUpdateError `json:"errors,omitempty"`
}

// handleBulkUpdate handles PATCH /api/overrides/bulk
// validateBulkRequest validates the bulk update request
func validateBulkRequest(req *BulkUpdateRequest) bool {
	if len(req.ContentIDs) == 0 || req.Field == "" {
		return false
	}

	for _, id := range req.ContentIDs {
		if !domain.IsValidAcestreamID(id) {
			return false
		}
	}

	return true
}

// validateTVGID validates a TVG-ID against EPG cache if available
func (h *OverridesHandler) validateTVGID(tvgID string) (bool, []string) {
	if tvgID == "" || h.epgCache == nil {
		return true, nil
	}

	if h.epgCache.IsValid(tvgID) {
		return true, nil
	}

	suggestions := h.getSuggestions(tvgID, 10)
	return false, suggestions
}

// respondWithValidationError sends a validation error response
func (h *OverridesHandler) respondWithValidationError(w http.ResponseWriter, r *http.Request, field, message string, suggestions []string) {
	validationErr := ValidationError{
		Error:       "validation_error",
		Field:       field,
		Message:     message,
		Suggestions: suggestions,
	}

	h.logger.Warn("Validation error", map[string]interface{}{
		"method": r.Method,
		"path":   r.URL.Path,
		"field":  field,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	if err := json.NewEncoder(w).Encode(validationErr); err != nil {
		h.logger.Warn("Failed to encode validation error", map[string]interface{}{
			"method": r.Method,
			"path":   r.URL.Path,
			"error":  err.Error(),
		})
	}
}

// respondWithBulkResult sends a bulk update result response
func (h *OverridesHandler) respondWithBulkResult(w http.ResponseWriter, r *http.Request, result *overrides.BulkUpdateResult) {
	response := BulkUpdateResponse{
		Updated: result.Updated,
		Failed:  result.Failed,
		Errors:  result.Errors,
	}

	w.Header().Set("Content-Type", "application/json")
	if result.Failed > 0 {
		w.WriteHeader(http.StatusMultiStatus)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Warn("Failed to encode bulk update response", map[string]interface{}{
			"method": r.Method,
			"path":   r.URL.Path,
			"error":  err.Error(),
		})
	}
}

func (h *OverridesHandler) handleBulkUpdate(w http.ResponseWriter, r *http.Request) {
	force := r.URL.Query().Get("force") == "true"
	atomic := r.URL.Query().Get("atomic") != "false"

	var req BulkUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logging.WriteJSONError(w, h.logger, "Invalid request body", http.StatusBadRequest, map[string]interface{}{
			"method": r.Method,
			"path":   r.URL.Path,
			"error":  err.Error(),
		})
		return
	}

	if !validateBulkRequest(&req) {
		logging.WriteJSONError(w, h.logger, "acestream_ids cannot be empty and must be valid; field cannot be empty", http.StatusBadRequest, map[string]interface{}{
			"method": r.Method,
			"path":   r.URL.Path,
		})
		return
	}

	// Validate TVG-ID if updating that field (unless force=true)
	if !force && req.Field == "tvg_id" {
		if strVal, ok := req.Value.(string); ok {
			tvgID := strings.TrimSpace(strVal)
			if valid, suggestions := h.validateTVGID(tvgID); !valid {
				h.respondWithValidationError(w, r, "tvg_id", "TVG-ID not found in EPG data", suggestions)
				return
			}
		}
	}

	result, err := h.overridesMgr.BulkUpdate(req.ContentIDs, req.Field, req.Value, atomic)
	if err != nil {
		if result != nil {
			h.logger.Error("Bulk update partially failed", map[string]interface{}{
				"method":  r.Method,
				"path":    r.URL.Path,
				"updated": result.Updated,
				"failed":  result.Failed,
				"error":   err.Error(),
			})
			w.WriteHeader(http.StatusInternalServerError)
			h.respondWithBulkResult(w, r, result)
			return
		}
		logging.WriteJSONError(w, h.logger, "Internal Server Error", http.StatusInternalServerError, map[string]interface{}{
			"method": r.Method,
			"path":   r.URL.Path,
			"error":  err.Error(),
		})
		return
	}

	h.logger.Info("Bulk update completed", map[string]interface{}{
		"method":  r.Method,
		"path":    r.URL.Path,
		"updated": result.Updated,
		"failed":  result.Failed,
	})
	h.respondWithBulkResult(w, r, result)
}
