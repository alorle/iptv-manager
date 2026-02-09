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

	t.Run("parse NEW ERA fixture file", func(t *testing.T) {
		// Load fixture file
		fixturePath := filepath.Join("testdata", "acestream_newera.txt")
		fixtureData, err := os.ReadFile(fixturePath)
		if err != nil {
			t.Fatalf("failed to read NEW ERA fixture file: %v", err)
		}

		// Create test HTTP server serving the fixture
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
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

		// Verify expected channels from fixture
		// HBO, ESPN, CNN, Discovery, and AlsoInvalid (valid format, just no acestream:// prefix)
		expectedChannels := 5
		if len(hashes) != expectedChannels {
			t.Errorf("expected %d channels, got %d", expectedChannels, len(hashes))
		}

		// Verify HBO channel
		hboHashes, ok := hashes["HBO"]
		if !ok {
			t.Fatal("expected HBO channel in results")
		}
		if len(hboHashes) != 1 {
			t.Errorf("expected 1 hash for HBO, got %d", len(hboHashes))
		}
		if hboHashes[0] != "0123456789abcdef0123456789abcdef01234567" {
			t.Errorf("unexpected HBO hash: %q", hboHashes[0])
		}

		// Verify ESPN channel
		espnHashes, ok := hashes["ESPN"]
		if !ok {
			t.Fatal("expected ESPN channel in results")
		}
		if len(espnHashes) != 1 {
			t.Errorf("expected 1 hash for ESPN, got %d", len(espnHashes))
		}
		if espnHashes[0] != "fedcba9876543210fedcba9876543210fedcba98" {
			t.Errorf("unexpected ESPN hash: %q", espnHashes[0])
		}

		// Verify CNN channel
		cnnHashes, ok := hashes["CNN"]
		if !ok {
			t.Fatal("expected CNN channel in results")
		}
		if len(cnnHashes) != 1 {
			t.Errorf("expected 1 hash for CNN, got %d", len(cnnHashes))
		}

		// Verify Discovery channel
		discoveryHashes, ok := hashes["Discovery"]
		if !ok {
			t.Fatal("expected Discovery channel in results")
		}
		if len(discoveryHashes) != 1 {
			t.Errorf("expected 1 hash for Discovery, got %d", len(discoveryHashes))
		}

		// Verify invalid lines were skipped
		if _, ok := hashes["InvalidLine"]; ok {
			t.Error("invalid line (only one field) should have been skipped")
		}

		// AlsoInvalid is actually valid - the parser accepts plain hashes without acestream:// prefix
		alsoInvalidHashes, ok := hashes["AlsoInvalid"]
		if !ok {
			t.Fatal("expected AlsoInvalid channel in results (plain hash without prefix is valid)")
		}
		if len(alsoInvalidHashes) != 1 {
			t.Errorf("expected 1 hash for AlsoInvalid, got %d", len(alsoInvalidHashes))
		}
		if alsoInvalidHashes[0] != "noprefix" {
			t.Errorf("unexpected AlsoInvalid hash: %q", alsoInvalidHashes[0])
		}

		t.Logf("Successfully parsed %d channels from NEW ERA fixture", len(hashes))
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

		// Verify expected channels from fixture
		// HBO and ESPN should be present, CNN (empty hashes) and unnamed entry should be skipped
		expectedChannels := 2
		if len(hashes) != expectedChannels {
			t.Errorf("expected %d channels, got %d", expectedChannels, len(hashes))
		}

		// Verify HBO channel with multiple hashes
		hboHashes, ok := hashes["HBO"]
		if !ok {
			t.Fatal("expected HBO channel in results")
		}
		if len(hboHashes) != 2 {
			t.Errorf("expected 2 hashes for HBO, got %d", len(hboHashes))
		}
		if hboHashes[0] != "0123456789abcdef0123456789abcdef01234567" {
			t.Errorf("unexpected HBO hash[0]: %q", hboHashes[0])
		}
		if hboHashes[1] != "9999999999999999999999999999999999999999" {
			t.Errorf("unexpected HBO hash[1]: %q", hboHashes[1])
		}

		// Verify ESPN channel
		espnHashes, ok := hashes["ESPN"]
		if !ok {
			t.Fatal("expected ESPN channel in results")
		}
		if len(espnHashes) != 1 {
			t.Errorf("expected 1 hash for ESPN, got %d", len(espnHashes))
		}
		if espnHashes[0] != "fedcba9876543210fedcba9876543210fedcba98" {
			t.Errorf("unexpected ESPN hash: %q", espnHashes[0])
		}

		// Verify CNN with empty hashes was skipped
		if _, ok := hashes["CNN"]; ok {
			t.Error("CNN channel with empty hashes should have been skipped")
		}

		// Verify unnamed entry was skipped
		if _, ok := hashes[""]; ok {
			t.Error("entry with empty name should have been skipped")
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

	t.Run("parse NEW ERA with empty lines", func(t *testing.T) {
		// Create test HTTP server serving data with empty lines
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			data := "HBO acestream://0123456789abcdef0123456789abcdef01234567\n\n\nESPN acestream://fedcba9876543210fedcba9876543210fedcba98\n"
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

		// Empty lines should be ignored
		if len(hashes) != 2 {
			t.Errorf("expected 2 channels (empty lines ignored), got %d", len(hashes))
		}

		t.Logf("Successfully parsed %d channels, ignoring empty lines", len(hashes))
	})

	t.Run("parse Elcano with empty array", func(t *testing.T) {
		// Create test HTTP server serving empty JSON array
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[]`))
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

		t.Logf("Successfully handled empty JSON array")
	})
}
