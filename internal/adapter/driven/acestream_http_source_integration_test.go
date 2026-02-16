package driven

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAcestreamHTTPSource_FetchHashes_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Run("parse NEW ERA M3U fixture file", func(t *testing.T) {
		// Load fixture file
		fixturePath := filepath.Join("testdata", "acestream_newera.txt")
		fixtureData, err := os.ReadFile(fixturePath)
		if err != nil {
			t.Fatalf("failed to read NEW ERA fixture file: %v", err)
		}

		// Create test HTTP server serving the fixture
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "audio/x-mpegurl")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(fixtureData)
		}))
		defer server.Close()

		// Temporarily override source URL for testing
		originalURL := sourceURLs[SourceNewEra]
		sourceURLs[SourceNewEra] = server.URL
		defer func() { sourceURLs[SourceNewEra] = originalURL }()

		// Create source and fetch hashes
		source := NewAcestreamHTTPSource()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		hashes, err := source.FetchHashes(ctx, SourceNewEra)
		if err != nil {
			t.Fatalf("failed to fetch NEW ERA hashes from fixture: %v", err)
		}

		// Fixture has: HBO HD (1), ESPN HD (1), CNN HD (1), Discovery HD (1), DAZN 1 HD (2)
		// Entry with empty tvg-id is skipped
		expectedChannels := 5
		if len(hashes) != expectedChannels {
			t.Errorf("expected %d channels, got %d: %v", expectedChannels, len(hashes), hashes)
		}

		// Verify HBO HD channel (keyed by tvg-id, not display name)
		hboHashes, ok := hashes["HBO HD"]
		if !ok {
			t.Fatal("expected 'HBO HD' channel in results")
		}
		if len(hboHashes) != 1 {
			t.Errorf("expected 1 hash for HBO HD, got %d", len(hboHashes))
		}
		if hboHashes[0] != "0123456789abcdef0123456789abcdef01234567" {
			t.Errorf("unexpected HBO hash: %q", hboHashes[0])
		}

		// Verify ESPN HD channel
		espnHashes, ok := hashes["ESPN HD"]
		if !ok {
			t.Fatal("expected 'ESPN HD' channel in results")
		}
		if len(espnHashes) != 1 {
			t.Errorf("expected 1 hash for ESPN HD, got %d", len(espnHashes))
		}

		// Verify DAZN 1 HD has 2 hashes (two entries with same tvg-id)
		daznHashes, ok := hashes["DAZN 1 HD"]
		if !ok {
			t.Fatal("expected 'DAZN 1 HD' channel in results")
		}
		if len(daznHashes) != 2 {
			t.Errorf("expected 2 hashes for DAZN 1 HD, got %d", len(daznHashes))
		}

		// Verify entry with empty tvg-id was skipped
		if _, ok := hashes[""]; ok {
			t.Error("entry with empty tvg-id should have been skipped")
		}

		t.Logf("Successfully parsed %d channels from NEW ERA M3U fixture", len(hashes))
	})

	t.Run("parse Elcano fixture file", func(t *testing.T) {
		// Load fixture file
		fixturePath := filepath.Join("testdata", "acestream_elcano.json")
		fixtureData, err := os.ReadFile(fixturePath)
		if err != nil {
			t.Fatalf("failed to read Elcano fixture file: %v", err)
		}

		// Create test HTTP server serving the fixture
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(fixtureData)
		}))
		defer server.Close()

		// Temporarily override source URL for testing
		originalURL := sourceURLs[SourceElcano]
		sourceURLs[SourceElcano] = server.URL
		defer func() { sourceURLs[SourceElcano] = originalURL }()

		// Create source and fetch hashes
		source := NewAcestreamHTTPSource()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		hashes, err := source.FetchHashes(ctx, SourceElcano)
		if err != nil {
			t.Fatalf("failed to fetch Elcano hashes from fixture: %v", err)
		}

		// Fixture has: HBO HD (2 hashes), ESPN HD (1 hash), CNN HD (empty hash → skipped),
		// "No TVG ID Channel" (no tvg_id → falls back to title)
		expectedChannels := 3
		if len(hashes) != expectedChannels {
			t.Errorf("expected %d channels, got %d: %v", expectedChannels, len(hashes), hashes)
		}

		// Verify HBO HD channel (grouped by tvg_id) with 2 hashes
		hboHashes, ok := hashes["HBO HD"]
		if !ok {
			t.Fatal("expected 'HBO HD' channel in results")
		}
		if len(hboHashes) != 2 {
			t.Errorf("expected 2 hashes for HBO HD, got %d", len(hboHashes))
		}
		if hboHashes[0] != "0123456789abcdef0123456789abcdef01234567" {
			t.Errorf("unexpected HBO hash[0]: %q", hboHashes[0])
		}
		if hboHashes[1] != "9999999999999999999999999999999999999999" {
			t.Errorf("unexpected HBO hash[1]: %q", hboHashes[1])
		}

		// Verify ESPN HD channel
		espnHashes, ok := hashes["ESPN HD"]
		if !ok {
			t.Fatal("expected 'ESPN HD' channel in results")
		}
		if len(espnHashes) != 1 {
			t.Errorf("expected 1 hash for ESPN HD, got %d", len(espnHashes))
		}

		// Verify CNN HD with empty hash was skipped
		if _, ok := hashes["CNN HD"]; ok {
			t.Error("CNN HD entry with empty hash should have been skipped")
		}

		// Verify entry with no tvg_id fell back to title
		noTvgHashes, ok := hashes["No TVG ID Channel"]
		if !ok {
			t.Fatal("expected 'No TVG ID Channel' in results (fallback to title)")
		}
		if len(noTvgHashes) != 1 {
			t.Errorf("expected 1 hash for 'No TVG ID Channel', got %d", len(noTvgHashes))
		}

		t.Logf("Successfully parsed %d channels from Elcano fixture", len(hashes))
	})

	t.Run("handle HTTP errors", func(t *testing.T) {
		// Create test HTTP server that returns 404
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		// Temporarily override source URL for testing
		originalURL := sourceURLs[SourceNewEra]
		sourceURLs[SourceNewEra] = server.URL
		defer func() { sourceURLs[SourceNewEra] = originalURL }()

		// Create source and attempt to fetch hashes
		source := NewAcestreamHTTPSource()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := source.FetchHashes(ctx, SourceNewEra)
		if err == nil {
			t.Fatal("expected error for HTTP 404, got nil")
		}

		t.Logf("Correctly handled HTTP error: %v", err)
	})

	t.Run("handle malformed JSON for Elcano", func(t *testing.T) {
		// Create test HTTP server serving invalid JSON
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"invalid": "json"`))
		}))
		defer server.Close()

		// Temporarily override source URL for testing
		originalURL := sourceURLs[SourceElcano]
		sourceURLs[SourceElcano] = server.URL
		defer func() { sourceURLs[SourceElcano] = originalURL }()

		// Create source and attempt to fetch hashes
		source := NewAcestreamHTTPSource()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := source.FetchHashes(ctx, SourceElcano)
		if err == nil {
			t.Fatal("expected error for malformed JSON, got nil")
		}

		t.Logf("Correctly rejected malformed JSON: %v", err)
	})

	t.Run("handle context timeout", func(t *testing.T) {
		// Create test HTTP server that delays response
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		// Temporarily override source URL for testing
		originalURL := sourceURLs[SourceNewEra]
		sourceURLs[SourceNewEra] = server.URL
		defer func() { sourceURLs[SourceNewEra] = originalURL }()

		// Create source and attempt to fetch with short timeout
		source := NewAcestreamHTTPSource()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		_, err := source.FetchHashes(ctx, SourceNewEra)
		if err == nil {
			t.Fatal("expected timeout error, got nil")
		}

		t.Logf("Correctly handled context timeout: %v", err)
	})

	t.Run("handle context cancellation", func(t *testing.T) {
		// Create test HTTP server that delays response
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		// Temporarily override source URL for testing
		originalURL := sourceURLs[SourceElcano]
		sourceURLs[SourceElcano] = server.URL
		defer func() { sourceURLs[SourceElcano] = originalURL }()

		// Create source and attempt to fetch with cancelled context
		source := NewAcestreamHTTPSource()

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := source.FetchHashes(ctx, SourceElcano)
		if err == nil {
			t.Fatal("expected cancellation error, got nil")
		}

		t.Logf("Correctly handled context cancellation: %v", err)
	})

	t.Run("handle unknown source", func(t *testing.T) {
		source := NewAcestreamHTTPSource()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := source.FetchHashes(ctx, "unknown-source")
		if err == nil {
			t.Fatal("expected error for unknown source, got nil")
		}

		t.Logf("Correctly rejected unknown source: %v", err)
	})

	t.Run("parse NEW ERA M3U with empty lines and comments", func(t *testing.T) {
		// Create test HTTP server serving M3U data with extra lines
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "audio/x-mpegurl")
			w.WriteHeader(http.StatusOK)
			data := "#EXTM3U\n\n#EXTGRP: group-title=\"TEST\"\n\n#EXTINF:-1 tvg-id=\"HBO HD\" group-title=\"TEST\", HBO\nacestream://0123456789abcdef0123456789abcdef01234567\n\n#EXTINF:-1 tvg-id=\"ESPN HD\" group-title=\"TEST\", ESPN\nacestream://fedcba9876543210fedcba9876543210fedcba98\n"
			_, _ = w.Write([]byte(data))
		}))
		defer server.Close()

		// Temporarily override source URL for testing
		originalURL := sourceURLs[SourceNewEra]
		sourceURLs[SourceNewEra] = server.URL
		defer func() { sourceURLs[SourceNewEra] = originalURL }()

		// Create source and fetch hashes
		source := NewAcestreamHTTPSource()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		hashes, err := source.FetchHashes(ctx, SourceNewEra)
		if err != nil {
			t.Fatalf("failed to fetch NEW ERA hashes: %v", err)
		}

		if len(hashes) != 2 {
			t.Errorf("expected 2 channels, got %d", len(hashes))
		}

		t.Logf("Successfully parsed %d channels from M3U with extra lines", len(hashes))
	})

	t.Run("parse Elcano with empty hashes array", func(t *testing.T) {
		// Create test HTTP server serving JSON with empty hashes array
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"generated": "2026-01-01T00:00:00Z", "count": 0, "hashes": []}`))
		}))
		defer server.Close()

		// Temporarily override source URL for testing
		originalURL := sourceURLs[SourceElcano]
		sourceURLs[SourceElcano] = server.URL
		defer func() { sourceURLs[SourceElcano] = originalURL }()

		// Create source and fetch hashes
		source := NewAcestreamHTTPSource()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		hashes, err := source.FetchHashes(ctx, SourceElcano)
		if err != nil {
			t.Fatalf("failed to fetch Elcano hashes: %v", err)
		}

		// Empty array should result in no channels
		if len(hashes) != 0 {
			t.Errorf("expected 0 channels from empty array, got %d", len(hashes))
		}

		t.Logf("Successfully handled empty hashes array")
	})
}
