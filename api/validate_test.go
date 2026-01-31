package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alorle/iptv-manager/epg"
)

// mockEPGCache is a mock implementation of EPGCache for testing
type mockEPGCache struct {
	channels map[string]epg.ChannelInfo
}

func (m *mockEPGCache) IsValid(tvgID string) bool {
	_, exists := m.channels[tvgID]
	return exists
}

func (m *mockEPGCache) Search(query string, maxResults int) []epg.ChannelInfo {
	var results []epg.ChannelInfo
	for _, ch := range m.channels {
		if len(results) >= maxResults {
			break
		}
		results = append(results, ch)
	}
	return results
}

func (m *mockEPGCache) Count() int {
	return len(m.channels)
}

func newMockEPGCache() *mockEPGCache {
	return &mockEPGCache{
		channels: map[string]epg.ChannelInfo{
			"La1.TV":      {ID: "La1.TV", DisplayName: "La 1"},
			"La2.TV":      {ID: "La2.TV", DisplayName: "La 2"},
			"Antena3.TV":  {ID: "Antena3.TV", DisplayName: "Antena 3"},
			"Cuatro.TV":   {ID: "Cuatro.TV", DisplayName: "Cuatro"},
			"Telecinco.TV": {ID: "Telecinco.TV", DisplayName: "Telecinco"},
			"LaSexta.TV":  {ID: "LaSexta.TV", DisplayName: "La Sexta"},
		},
	}
}

func TestValidateHandler_ValidTvgID(t *testing.T) {
	cache := newMockEPGCache()
	handler := NewValidateHandler(cache)

	reqBody := ValidateRequest{TvgID: "La1.TV"}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/validate/tvg-id", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp ValidateResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !resp.Valid {
		t.Error("Expected valid=true for valid TVG-ID")
	}

	if len(resp.Suggestions) > 0 {
		t.Error("Expected no suggestions for valid TVG-ID")
	}
}

func TestValidateHandler_InvalidTvgID(t *testing.T) {
	cache := newMockEPGCache()
	handler := NewValidateHandler(cache)

	reqBody := ValidateRequest{TvgID: "InvalidChannel.TV"}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/validate/tvg-id", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp ValidateResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Valid {
		t.Error("Expected valid=false for invalid TVG-ID")
	}

	if len(resp.Suggestions) == 0 {
		t.Error("Expected suggestions for invalid TVG-ID")
	}

	if len(resp.Suggestions) > 10 {
		t.Errorf("Expected at most 10 suggestions, got %d", len(resp.Suggestions))
	}
}

func TestValidateHandler_EmptyTvgID(t *testing.T) {
	cache := newMockEPGCache()
	handler := NewValidateHandler(cache)

	reqBody := ValidateRequest{TvgID: ""}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/validate/tvg-id", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp ValidateResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !resp.Valid {
		t.Error("Expected valid=true for empty TVG-ID (means no EPG)")
	}

	if len(resp.Suggestions) > 0 {
		t.Error("Expected no suggestions for empty TVG-ID")
	}
}

func TestValidateHandler_WhitespaceTvgID(t *testing.T) {
	cache := newMockEPGCache()
	handler := NewValidateHandler(cache)

	reqBody := ValidateRequest{TvgID: "   "}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/validate/tvg-id", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp ValidateResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !resp.Valid {
		t.Error("Expected valid=true for whitespace TVG-ID (means no EPG)")
	}
}

func TestValidateHandler_MethodNotAllowed(t *testing.T) {
	cache := newMockEPGCache()
	handler := NewValidateHandler(cache)

	req := httptest.NewRequest(http.MethodGet, "/api/validate/tvg-id", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestValidateHandler_InvalidJSON(t *testing.T) {
	cache := newMockEPGCache()
	handler := NewValidateHandler(cache)

	req := httptest.NewRequest(http.MethodPost, "/api/validate/tvg-id", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestGetSuggestions_PrefixMatch(t *testing.T) {
	cache := newMockEPGCache()
	handler := NewValidateHandler(cache)

	suggestions := handler.getSuggestions("La", 10)

	if len(suggestions) == 0 {
		t.Error("Expected suggestions for prefix 'La'")
	}

	// Check that suggestions are relevant
	found := false
	for _, s := range suggestions {
		if s == "La1.TV" || s == "La2.TV" || s == "LaSexta.TV" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected at least one channel starting with 'La'")
	}
}

func TestGetSuggestions_MaxResults(t *testing.T) {
	cache := newMockEPGCache()
	handler := NewValidateHandler(cache)

	suggestions := handler.getSuggestions("TV", 3)

	if len(suggestions) > 3 {
		t.Errorf("Expected at most 3 suggestions, got %d", len(suggestions))
	}
}
