package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/alorle/iptv-manager/domain"
	"github.com/alorle/iptv-manager/overrides"
)

// OverridesHandler handles CRUD operations for channel overrides
type OverridesHandler struct {
	overridesMgr *overrides.Manager
	epgCache     TVGValidator
}

// NewOverridesHandler creates a new handler for the overrides API
func NewOverridesHandler(overridesMgr *overrides.Manager, epgCache TVGValidator) *OverridesHandler {
	return &OverridesHandler{
		overridesMgr: overridesMgr,
		epgCache:     epgCache,
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
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// /api/overrides/bulk - bulk update operations
	if path == "bulk" {
		if r.Method == http.MethodPatch {
			h.handleBulkUpdate(w, r)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// /api/overrides/:acestreamId - single override operations
	acestreamID := path

	// Validate acestream ID format
	if !domain.IsValidAcestreamID(acestreamID) {
		http.Error(w, "Invalid acestream_id: must be 40 hexadecimal characters", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.handleGet(w, r, acestreamID)
	case http.MethodPut:
		h.handlePut(w, r, acestreamID)
	case http.MethodDelete:
		h.handleDelete(w, r, acestreamID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// OverrideResponse represents the response for a single override
type OverrideResponse struct {
	AcestreamID string                     `json:"acestream_id"`
	Override    *overrides.ChannelOverride `json:"override"`
}

// handleGet handles GET /api/overrides/:acestreamId
func (h *OverridesHandler) handleGet(w http.ResponseWriter, _ *http.Request, acestreamID string) {
	override := h.overridesMgr.Get(acestreamID)
	if override == nil {
		http.Error(w, "Override not found", http.StatusNotFound)
		return
	}

	response := OverrideResponse{
		AcestreamID: acestreamID,
		Override:    override,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode override response: %v", err)
	}
}

// ValidationError represents a validation error response
type ValidationError struct {
	Error       string   `json:"error"`
	Field       string   `json:"field,omitempty"`
	Message     string   `json:"message"`
	Suggestions []string `json:"suggestions,omitempty"`
}

// handlePut handles PUT /api/overrides/:acestreamId
func (h *OverridesHandler) handlePut(w http.ResponseWriter, r *http.Request, acestreamID string) {
	// Check for force parameter
	force := r.URL.Query().Get("force") == "true"

	// Parse request body
	var override overrides.ChannelOverride
	if err := json.NewDecoder(r.Body).Decode(&override); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate TVG-ID if it's being set and EPG cache is available
	if !force && h.epgCache != nil && override.TvgID != nil {
		tvgID := strings.TrimSpace(*override.TvgID)

		// Empty TVG-ID is valid (means "no EPG")
		if tvgID != "" && !h.epgCache.IsValid(tvgID) {
			// Get suggestions for invalid TVG-ID
			suggestions := h.getSuggestions(tvgID, 10)

			validationErr := ValidationError{
				Error:       "validation_error",
				Field:       "tvg_id",
				Message:     "TVG-ID not found in EPG data",
				Suggestions: suggestions,
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			if err := json.NewEncoder(w).Encode(validationErr); err != nil {
				log.Printf("Failed to encode validation error: %v", err)
			}
			return
		}
	}

	// Save the override
	if err := h.overridesMgr.Set(acestreamID, override); err != nil {
		log.Printf("Failed to save override for %s: %v", acestreamID, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	log.Printf("Created/updated override for %s", acestreamID)

	// Return the created/updated override
	response := OverrideResponse{
		AcestreamID: acestreamID,
		Override:    &override,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode override response: %v", err)
	}
}

// handleDelete handles DELETE /api/overrides/:acestreamId
func (h *OverridesHandler) handleDelete(w http.ResponseWriter, _ *http.Request, acestreamID string) {
	// Check if override exists
	if h.overridesMgr.Get(acestreamID) == nil {
		http.Error(w, "Override not found", http.StatusNotFound)
		return
	}

	// Delete the override
	if err := h.overridesMgr.Delete(acestreamID); err != nil {
		log.Printf("Failed to delete override for %s: %v", acestreamID, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	log.Printf("Deleted override for %s", acestreamID)

	w.WriteHeader(http.StatusNoContent)
}

// handleList handles GET /api/overrides
func (h *OverridesHandler) handleList(w http.ResponseWriter, _ *http.Request) {
	allOverrides := h.overridesMgr.List()

	// Convert map to array of responses
	var responses []OverrideResponse
	for acestreamID, override := range allOverrides {
		// Create a copy of the override for the response
		overrideCopy := override
		responses = append(responses, OverrideResponse{
			AcestreamID: acestreamID,
			Override:    &overrideCopy,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(responses); err != nil {
		log.Printf("Failed to encode overrides list: %v", err)
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
	AcestreamIDs []string    `json:"acestream_ids"`
	Field        string      `json:"field"`
	Value        interface{} `json:"value"`
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
	if len(req.AcestreamIDs) == 0 || req.Field == "" {
		return false
	}

	for _, id := range req.AcestreamIDs {
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
func respondWithValidationError(w http.ResponseWriter, field, message string, suggestions []string) {
	validationErr := ValidationError{
		Error:       "validation_error",
		Field:       field,
		Message:     message,
		Suggestions: suggestions,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	if err := json.NewEncoder(w).Encode(validationErr); err != nil {
		log.Printf("Failed to encode validation error: %v", err)
	}
}

// respondWithBulkResult sends a bulk update result response
func respondWithBulkResult(w http.ResponseWriter, result *overrides.BulkUpdateResult) {
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
		log.Printf("Failed to encode bulk update response: %v", err)
	}
}

func (h *OverridesHandler) handleBulkUpdate(w http.ResponseWriter, r *http.Request) {
	force := r.URL.Query().Get("force") == "true"
	atomic := r.URL.Query().Get("atomic") != "false"

	var req BulkUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if !validateBulkRequest(&req) {
		http.Error(w, "acestream_ids cannot be empty and must be valid; field cannot be empty", http.StatusBadRequest)
		return
	}

	// Validate TVG-ID if updating that field (unless force=true)
	if !force && req.Field == "tvg_id" {
		if strVal, ok := req.Value.(string); ok {
			tvgID := strings.TrimSpace(strVal)
			if valid, suggestions := h.validateTVGID(tvgID); !valid {
				respondWithValidationError(w, "tvg_id", "TVG-ID not found in EPG data", suggestions)
				return
			}
		}
	}

	result, err := h.overridesMgr.BulkUpdate(req.AcestreamIDs, req.Field, req.Value, atomic)
	if err != nil {
		log.Printf("Bulk update failed: %v", err)
		if result != nil {
			w.WriteHeader(http.StatusInternalServerError)
			respondWithBulkResult(w, result)
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	log.Printf("Bulk update completed: %d updated, %d failed", result.Updated, result.Failed)
	respondWithBulkResult(w, result)
}
