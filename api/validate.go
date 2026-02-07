package api

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strings"

	"github.com/alorle/iptv-manager/epg"
)

// ValidateRequest represents the request body for validating a TVG-ID
type ValidateRequest struct {
	TvgID string `json:"tvg_id"`
}

// ValidateResponse represents the response for TVG-ID validation
type ValidateResponse struct {
	Valid       bool     `json:"valid"`
	Suggestions []string `json:"suggestions,omitempty"`
}

// TVGValidator defines the interface for TVG-ID validation and channel search
type TVGValidator interface {
	IsValid(tvgID string) bool
	Search(query string, maxResults int) []epg.ChannelInfo
}

// ValidateHandler handles the POST /api/validate/tvg-id endpoint
type ValidateHandler struct {
	epgCache TVGValidator
}

// NewValidateHandler creates a new handler for the validation API
func NewValidateHandler(epgCache TVGValidator) *ValidateHandler {
	return &ValidateHandler{
		epgCache: epgCache,
	}
}

// ServeHTTP handles the POST /api/validate/tvg-id request
func (h *ValidateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var req ValidateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Empty/null TVG-ID is valid (means "no EPG")
	if strings.TrimSpace(req.TvgID) == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(ValidateResponse{Valid: true})
		return
	}

	// Check if TVG-ID exists in EPG cache
	valid := h.epgCache.IsValid(req.TvgID)

	response := ValidateResponse{
		Valid: valid,
	}

	// If invalid, provide suggestions
	if !valid {
		suggestions := h.getSuggestions(req.TvgID, 10)
		response.Suggestions = suggestions
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode validation response: %v", err)
	}
}

// getSuggestions returns up to maxResults closest matches for a TVG-ID
// Suggestions are sorted by relevance (exact prefix match first, then fuzzy)
func (h *ValidateHandler) getSuggestions(tvgID string, maxResults int) []string {
	type suggestion struct {
		id       string
		priority int // Lower is better (0 = exact prefix, 1 = case-insensitive prefix, 2 = fuzzy)
	}

	var suggestions []suggestion
	tvgIDLower := strings.ToLower(tvgID)

	// Use Search to get potential matches
	matches := h.epgCache.Search(tvgID, maxResults*3) // Get more than needed for better sorting

	for _, match := range matches {
		idLower := strings.ToLower(match.ID)
		nameLower := strings.ToLower(match.DisplayName)

		priority := 2 // Default: fuzzy match

		// Exact prefix match (case-sensitive)
		if strings.HasPrefix(match.ID, tvgID) {
			priority = 0
		} else if strings.HasPrefix(idLower, tvgIDLower) {
			// Case-insensitive prefix match
			priority = 1
		} else if strings.HasPrefix(nameLower, tvgIDLower) {
			// Prefix match on display name
			priority = 1
		}

		suggestions = append(suggestions, suggestion{
			id:       match.ID,
			priority: priority,
		})
	}

	// Sort by priority (lower is better)
	sort.SliceStable(suggestions, func(i, j int) bool {
		return suggestions[i].priority < suggestions[j].priority
	})

	// Extract IDs and limit to maxResults
	var result []string
	for i := 0; i < len(suggestions) && i < maxResults; i++ {
		result = append(result, suggestions[i].id)
	}

	return result
}
