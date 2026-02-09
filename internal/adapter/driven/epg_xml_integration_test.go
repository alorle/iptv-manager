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

func TestEPGXMLFetcher_FetchEPG_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Run("fetch from actual EPG source", func(t *testing.T) {
		fetcher := NewEPGXMLFetcher("", nil) // Use default URL

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		channels, err := fetcher.FetchEPG(ctx)
		if err != nil {
			t.Fatalf("failed to fetch EPG: %v", err)
		}

		if len(channels) == 0 {
			t.Error("expected at least one channel, got none")
		}

		// Verify first channel has required fields
		if len(channels) > 0 {
			ch := channels[0]
			if ch.ID() == "" {
				t.Error("expected channel to have ID")
			}
			if ch.Name() == "" {
				t.Error("expected channel to have name")
			}
			if ch.EPGID() == "" {
				t.Error("expected channel to have EPG ID")
			}

			t.Logf("Successfully fetched %d channels", len(channels))
			t.Logf("First channel: ID=%q, Name=%q, Logo=%q",
				ch.ID(), ch.Name(), ch.Logo())
		}
	})

	t.Run("parse fixture XML file", func(t *testing.T) {
		// Load fixture file
		fixturePath := filepath.Join("testdata", "epg_sample.xml")
		xmlData, err := os.ReadFile(fixturePath)
		if err != nil {
			t.Fatalf("failed to read fixture file: %v", err)
		}

		// Create test HTTP server serving the fixture
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(xmlData)
		}))
		defer server.Close()

		// Create fetcher pointing to test server
		fetcher := NewEPGXMLFetcher(server.URL, nil)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		channels, err := fetcher.FetchEPG(ctx)
		if err != nil {
			t.Fatalf("failed to fetch EPG from fixture: %v", err)
		}

		// Verify expected channels from fixture
		if len(channels) != 4 {
			t.Fatalf("expected 4 channels from fixture, got %d", len(channels))
		}

		// Verify first channel (HBO)
		hbo := channels[0]
		if hbo.ID() != "hbo.channel" {
			t.Errorf("expected HBO channel ID 'hbo.channel', got %q", hbo.ID())
		}
		if hbo.Name() != "HBO" {
			t.Errorf("expected HBO channel name 'HBO', got %q", hbo.Name())
		}
		if hbo.Logo() != "https://example.com/logos/hbo.png" {
			t.Errorf("expected HBO logo URL, got %q", hbo.Logo())
		}
		if hbo.EPGID() != "hbo.channel" {
			t.Errorf("expected HBO EPG ID to match channel ID, got %q", hbo.EPGID())
		}

		// Verify second channel (ESPN)
		espn := channels[1]
		if espn.ID() != "espn.channel" {
			t.Errorf("expected ESPN channel ID 'espn.channel', got %q", espn.ID())
		}
		if espn.Name() != "ESPN" {
			t.Errorf("expected ESPN channel name 'ESPN', got %q", espn.Name())
		}

		// Verify third channel (CNN - no logo)
		cnn := channels[2]
		if cnn.ID() != "cnn.channel" {
			t.Errorf("expected CNN channel ID 'cnn.channel', got %q", cnn.ID())
		}
		if cnn.Name() != "CNN International" {
			t.Errorf("expected CNN channel name 'CNN International', got %q", cnn.Name())
		}
		if cnn.Logo() != "" {
			t.Errorf("expected CNN to have no logo, got %q", cnn.Logo())
		}

		// Verify fourth channel (minimal - no display-name)
		minimal := channels[3]
		if minimal.ID() != "minimal.channel" {
			t.Errorf("expected minimal channel ID 'minimal.channel', got %q", minimal.ID())
		}
		if minimal.Name() != "minimal.channel" {
			t.Errorf("expected minimal channel name to fall back to ID, got %q", minimal.Name())
		}

		t.Logf("Successfully parsed %d channels from fixture", len(channels))
	})

	t.Run("handle malformed XML fixture", func(t *testing.T) {
		// Load malformed fixture file
		fixturePath := filepath.Join("testdata", "epg_malformed.xml")
		xmlData, err := os.ReadFile(fixturePath)
		if err != nil {
			t.Fatalf("failed to read malformed fixture file: %v", err)
		}

		// Create test HTTP server serving the malformed fixture
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(xmlData)
		}))
		defer server.Close()

		// Create fetcher pointing to test server
		fetcher := NewEPGXMLFetcher(server.URL, nil)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err = fetcher.FetchEPG(ctx)
		if err == nil {
			t.Fatal("expected error when parsing malformed XML, got nil")
		}

		t.Logf("Correctly rejected malformed XML: %v", err)
	})
}
