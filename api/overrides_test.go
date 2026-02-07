package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/alorle/iptv-manager/domain"
	"github.com/alorle/iptv-manager/overrides"
)

const (
	testAcestreamID = "0123456789abcdef0123456789abcdef01234567"
	testTvgIDLa1    = "La1.TV"
	testTvgIDLa2    = "La2.TV"
)

// Helper to create a temporary overrides manager for testing
func createTestOverridesManager(t *testing.T) (*overrides.Manager, func()) {
	t.Helper()

	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "overrides-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create overrides manager
	overridesPath := filepath.Join(tmpDir, "overrides.yaml")
	mgr, err := overrides.NewManager(overridesPath)
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create overrides manager: %v", err)
	}

	// Return cleanup function
	cleanup := func() {
		_ = os.RemoveAll(tmpDir)
	}

	return mgr, cleanup
}

func TestOverridesHandler_Get_Success(t *testing.T) {
	mgr, cleanup := createTestOverridesManager(t)
	defer cleanup()

	// Create a test override
	acestreamID := testAcestreamID
	tvgID := testTvgIDLa1
	override := overrides.ChannelOverride{
		TvgID: &tvgID,
	}
	if err := mgr.Set(acestreamID, override); err != nil {
		t.Fatalf("Failed to set override: %v", err)
	}

	handler := NewOverridesHandler(mgr, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/overrides/"+acestreamID, nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp OverrideResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.AcestreamID != acestreamID {
		t.Errorf("Expected acestream_id=%s, got %s", acestreamID, resp.AcestreamID)
	}

	if resp.Override == nil || resp.Override.TvgID == nil || *resp.Override.TvgID != tvgID {
		t.Errorf("Expected tvg_id=%s, got %v", tvgID, resp.Override)
	}
}

func TestOverridesHandler_Get_NotFound(t *testing.T) {
	mgr, cleanup := createTestOverridesManager(t)
	defer cleanup()

	handler := NewOverridesHandler(mgr, nil)

	acestreamID := testAcestreamID
	req := httptest.NewRequest(http.MethodGet, "/api/overrides/"+acestreamID, nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestOverridesHandler_Get_InvalidID(t *testing.T) {
	mgr, cleanup := createTestOverridesManager(t)
	defer cleanup()

	handler := NewOverridesHandler(mgr, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/overrides/invalid", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestOverridesHandler_Put_Create(t *testing.T) {
	mgr, cleanup := createTestOverridesManager(t)
	defer cleanup()

	handler := NewOverridesHandler(mgr, nil)

	acestreamID := testAcestreamID
	tvgID := testTvgIDLa1
	override := overrides.ChannelOverride{
		TvgID: &tvgID,
	}

	body, _ := json.Marshal(override)
	req := httptest.NewRequest(http.MethodPut, "/api/overrides/"+acestreamID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify it was saved
	saved := mgr.Get(acestreamID)
	if saved == nil || saved.TvgID == nil || *saved.TvgID != tvgID {
		t.Errorf("Override was not saved correctly")
	}
}

func TestOverridesHandler_Put_Update(t *testing.T) {
	mgr, cleanup := createTestOverridesManager(t)
	defer cleanup()

	acestreamID := testAcestreamID
	tvgID1 := testTvgIDLa1
	override1 := overrides.ChannelOverride{
		TvgID: &tvgID1,
	}
	if err := mgr.Set(acestreamID, override1); err != nil {
		t.Fatalf("Failed to set initial override: %v", err)
	}

	handler := NewOverridesHandler(mgr, nil)

	tvgID2 := testTvgIDLa2
	override2 := overrides.ChannelOverride{
		TvgID: &tvgID2,
	}

	body, _ := json.Marshal(override2)
	req := httptest.NewRequest(http.MethodPut, "/api/overrides/"+acestreamID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify it was updated
	saved := mgr.Get(acestreamID)
	if saved == nil || saved.TvgID == nil || *saved.TvgID != tvgID2 {
		t.Errorf("Override was not updated correctly")
	}
}

func TestOverridesHandler_Put_WithValidation_Valid(t *testing.T) {
	mgr, cleanup := createTestOverridesManager(t)
	defer cleanup()

	cache := newMockEPGCache()
	handler := NewOverridesHandler(mgr, cache)

	acestreamID := testAcestreamID
	tvgID := testTvgIDLa1
	override := overrides.ChannelOverride{
		TvgID: &tvgID,
	}

	body, _ := json.Marshal(override)
	req := httptest.NewRequest(http.MethodPut, "/api/overrides/"+acestreamID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d (body: %s)", w.Code, w.Body.String())
	}
}

func TestOverridesHandler_Put_WithValidation_Invalid(t *testing.T) {
	mgr, cleanup := createTestOverridesManager(t)
	defer cleanup()

	cache := newMockEPGCache()
	handler := NewOverridesHandler(mgr, cache)

	acestreamID := testAcestreamID
	tvgID := "InvalidChannel.TV"
	override := overrides.ChannelOverride{
		TvgID: &tvgID,
	}

	body, _ := json.Marshal(override)
	req := httptest.NewRequest(http.MethodPut, "/api/overrides/"+acestreamID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	var validationErr ValidationError
	if err := json.NewDecoder(w.Body).Decode(&validationErr); err != nil {
		t.Fatalf("Failed to decode validation error: %v", err)
	}

	if validationErr.Error != "validation_error" {
		t.Errorf("Expected error=validation_error, got %s", validationErr.Error)
	}

	if validationErr.Field != "tvg_id" {
		t.Errorf("Expected field=tvg_id, got %s", validationErr.Field)
	}

	if len(validationErr.Suggestions) == 0 {
		t.Error("Expected suggestions for invalid TVG-ID")
	}
}

func TestOverridesHandler_Put_WithValidation_Force(t *testing.T) {
	mgr, cleanup := createTestOverridesManager(t)
	defer cleanup()

	cache := newMockEPGCache()
	handler := NewOverridesHandler(mgr, cache)

	acestreamID := testAcestreamID
	tvgID := "InvalidChannel.TV"
	override := overrides.ChannelOverride{
		TvgID: &tvgID,
	}

	body, _ := json.Marshal(override)
	req := httptest.NewRequest(http.MethodPut, "/api/overrides/"+acestreamID+"?force=true", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 with force=true, got %d", w.Code)
	}

	// Verify it was saved despite invalid TVG-ID
	saved := mgr.Get(acestreamID)
	if saved == nil || saved.TvgID == nil || *saved.TvgID != tvgID {
		t.Errorf("Override was not saved with force=true")
	}
}

func TestOverridesHandler_Put_EmptyTvgID(t *testing.T) {
	mgr, cleanup := createTestOverridesManager(t)
	defer cleanup()

	cache := newMockEPGCache()
	handler := NewOverridesHandler(mgr, cache)

	acestreamID := testAcestreamID
	tvgID := ""
	override := overrides.ChannelOverride{
		TvgID: &tvgID,
	}

	body, _ := json.Marshal(override)
	req := httptest.NewRequest(http.MethodPut, "/api/overrides/"+acestreamID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for empty TVG-ID, got %d", w.Code)
	}
}

func TestOverridesHandler_Put_AllAttributes(t *testing.T) {
	mgr, cleanup := createTestOverridesManager(t)
	defer cleanup()

	handler := NewOverridesHandler(mgr, nil)

	acestreamID := testAcestreamID
	enabled := false
	tvgID := testTvgIDLa1
	tvgName := "La 1"
	tvgLogo := "http://example.com/logo.png"
	groupTitle := "Spain"

	override := overrides.ChannelOverride{
		Enabled:    &enabled,
		TvgID:      &tvgID,
		TvgName:    &tvgName,
		TvgLogo:    &tvgLogo,
		GroupTitle: &groupTitle,
	}

	body, _ := json.Marshal(override)
	req := httptest.NewRequest(http.MethodPut, "/api/overrides/"+acestreamID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify all attributes were saved
	saved := mgr.Get(acestreamID)
	if saved == nil {
		t.Fatal("Override was not saved")
	}

	if saved.Enabled == nil || *saved.Enabled != enabled {
		t.Error("Enabled was not saved correctly")
	}
	if saved.TvgID == nil || *saved.TvgID != tvgID {
		t.Error("TvgID was not saved correctly")
	}
	if saved.TvgName == nil || *saved.TvgName != tvgName {
		t.Error("TvgName was not saved correctly")
	}
	if saved.TvgLogo == nil || *saved.TvgLogo != tvgLogo {
		t.Error("TvgLogo was not saved correctly")
	}
	if saved.GroupTitle == nil || *saved.GroupTitle != groupTitle {
		t.Error("GroupTitle was not saved correctly")
	}
}

func TestOverridesHandler_Put_InvalidJSON(t *testing.T) {
	mgr, cleanup := createTestOverridesManager(t)
	defer cleanup()

	handler := NewOverridesHandler(mgr, nil)

	acestreamID := testAcestreamID
	req := httptest.NewRequest(http.MethodPut, "/api/overrides/"+acestreamID, bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestOverridesHandler_Delete_Success(t *testing.T) {
	mgr, cleanup := createTestOverridesManager(t)
	defer cleanup()

	// Create a test override
	acestreamID := testAcestreamID
	tvgID := testTvgIDLa1
	override := overrides.ChannelOverride{
		TvgID: &tvgID,
	}
	if err := mgr.Set(acestreamID, override); err != nil {
		t.Fatalf("Failed to set override: %v", err)
	}

	handler := NewOverridesHandler(mgr, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/overrides/"+acestreamID, nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", w.Code)
	}

	// Verify it was deleted
	if mgr.Get(acestreamID) != nil {
		t.Error("Override was not deleted")
	}
}

func TestOverridesHandler_Delete_NotFound(t *testing.T) {
	mgr, cleanup := createTestOverridesManager(t)
	defer cleanup()

	handler := NewOverridesHandler(mgr, nil)

	acestreamID := testAcestreamID
	req := httptest.NewRequest(http.MethodDelete, "/api/overrides/"+acestreamID, nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestOverridesHandler_List_Empty(t *testing.T) {
	mgr, cleanup := createTestOverridesManager(t)
	defer cleanup()

	handler := NewOverridesHandler(mgr, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/overrides", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp []OverrideResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(resp) != 0 {
		t.Errorf("Expected empty list, got %d items", len(resp))
	}
}

func TestOverridesHandler_List_WithData(t *testing.T) {
	mgr, cleanup := createTestOverridesManager(t)
	defer cleanup()

	// Create test overrides
	acestreamID1 := "0123456789abcdef0123456789abcdef01234567"
	acestreamID2 := "fedcba9876543210fedcba9876543210fedcba98"

	tvgID1 := testTvgIDLa1
	tvgID2 := testTvgIDLa2

	override1 := overrides.ChannelOverride{TvgID: &tvgID1}
	override2 := overrides.ChannelOverride{TvgID: &tvgID2}

	if err := mgr.Set(acestreamID1, override1); err != nil {
		t.Fatalf("Failed to set override1: %v", err)
	}
	if err := mgr.Set(acestreamID2, override2); err != nil {
		t.Fatalf("Failed to set override2: %v", err)
	}

	handler := NewOverridesHandler(mgr, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/overrides", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp []OverrideResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(resp) != 2 {
		t.Errorf("Expected 2 items, got %d", len(resp))
	}

	// Verify both overrides are in the list
	foundIDs := make(map[string]bool)
	for _, item := range resp {
		foundIDs[item.AcestreamID] = true
	}

	if !foundIDs[acestreamID1] {
		t.Error("Expected to find acestreamID1 in list")
	}
	if !foundIDs[acestreamID2] {
		t.Error("Expected to find acestreamID2 in list")
	}
}

func TestOverridesHandler_List_MethodNotAllowed(t *testing.T) {
	mgr, cleanup := createTestOverridesManager(t)
	defer cleanup()

	handler := NewOverridesHandler(mgr, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/overrides", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestValidateAcestreamID(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		expected bool
	}{
		{"Valid lowercase", "0123456789abcdef0123456789abcdef01234567", true},
		{"Valid uppercase", "0123456789ABCDEF0123456789ABCDEF01234567", true},
		{"Valid mixed case", "0123456789AbCdEf0123456789aBcDeF01234567", true},
		{"Too short", "0123456789abcdef", false},
		{"Too long", "0123456789abcdef0123456789abcdef0123456789", false},
		{"Invalid characters", "0123456789abcdef0123456789abcdef0123456g", false},
		{"Empty", "", false},
		{"Special characters", "0123456789abcdef0123456789abcdef0123456!", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := domain.IsValidAcestreamID(tt.id)
			if result != tt.expected {
				t.Errorf("IsValidAcestreamID(%q) = %v, expected %v", tt.id, result, tt.expected)
			}
		})
	}
}
