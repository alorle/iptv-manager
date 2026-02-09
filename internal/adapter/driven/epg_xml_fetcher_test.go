package driven

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewEPGXMLFetcher(t *testing.T) {
	t.Run("with custom URL and client", func(t *testing.T) {
		customURL := "https://example.com/epg.xml"
		customClient := &http.Client{Timeout: 10 * time.Second}

		fetcher := NewEPGXMLFetcher(customURL, customClient)

		if fetcher.url != customURL {
			t.Errorf("expected URL %q, got %q", customURL, fetcher.url)
		}
		if fetcher.client != customClient {
			t.Error("expected custom client to be used")
		}
	})

	t.Run("with empty URL uses default", func(t *testing.T) {
		fetcher := NewEPGXMLFetcher("", nil)

		if fetcher.url != defaultURL {
			t.Errorf("expected default URL %q, got %q", defaultURL, fetcher.url)
		}
	})

	t.Run("with nil client creates default", func(t *testing.T) {
		fetcher := NewEPGXMLFetcher("https://example.com/epg.xml", nil)

		if fetcher.client == nil {
			t.Error("expected default client to be created")
		}
		if fetcher.client.Timeout != defaultTimeout {
			t.Errorf("expected timeout %v, got %v", defaultTimeout, fetcher.client.Timeout)
		}
	})
}

func TestEPGXMLFetcher_FetchEPG(t *testing.T) {
	t.Run("successful fetch with valid XML", func(t *testing.T) {
		xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<tv generator-info-name="test">
	<channel id="channel-1">
		<display-name>Channel One</display-name>
		<display-name>Ch1</display-name>
		<icon src="https://example.com/logo1.png" />
	</channel>
	<channel id="channel-2">
		<display-name>Channel Two</display-name>
		<icon src="https://example.com/logo2.png" />
	</channel>
</tv>`

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET request, got %s", r.Method)
			}
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(xmlData))
		}))
		defer server.Close()

		fetcher := NewEPGXMLFetcher(server.URL, nil)
		channels, err := fetcher.FetchEPG(context.Background())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(channels) != 2 {
			t.Fatalf("expected 2 channels, got %d", len(channels))
		}

		// Verify first channel
		if channels[0].ID() != "channel-1" {
			t.Errorf("expected channel ID 'channel-1', got %q", channels[0].ID())
		}
		if channels[0].Name() != "Channel One" {
			t.Errorf("expected channel name 'Channel One', got %q", channels[0].Name())
		}
		if channels[0].Logo() != "https://example.com/logo1.png" {
			t.Errorf("expected logo URL, got %q", channels[0].Logo())
		}
		if channels[0].EPGID() != "channel-1" {
			t.Errorf("expected EPG ID 'channel-1', got %q", channels[0].EPGID())
		}
		if channels[0].Category() != "" {
			t.Errorf("expected empty category, got %q", channels[0].Category())
		}
		if channels[0].Language() != "" {
			t.Errorf("expected empty language, got %q", channels[0].Language())
		}

		// Verify second channel
		if channels[1].ID() != "channel-2" {
			t.Errorf("expected channel ID 'channel-2', got %q", channels[1].ID())
		}
		if channels[1].Name() != "Channel Two" {
			t.Errorf("expected channel name 'Channel Two', got %q", channels[1].Name())
		}
	})

	t.Run("channel with no display-name uses ID as name", func(t *testing.T) {
		xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<tv>
	<channel id="channel-fallback">
		<icon src="https://example.com/logo.png" />
	</channel>
</tv>`

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(xmlData))
		}))
		defer server.Close()

		fetcher := NewEPGXMLFetcher(server.URL, nil)
		channels, err := fetcher.FetchEPG(context.Background())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(channels) != 1 {
			t.Fatalf("expected 1 channel, got %d", len(channels))
		}

		if channels[0].Name() != "channel-fallback" {
			t.Errorf("expected channel name to fall back to ID, got %q", channels[0].Name())
		}
	})

	t.Run("channel with no icon has empty logo", func(t *testing.T) {
		xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<tv>
	<channel id="no-logo">
		<display-name>No Logo Channel</display-name>
	</channel>
</tv>`

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(xmlData))
		}))
		defer server.Close()

		fetcher := NewEPGXMLFetcher(server.URL, nil)
		channels, err := fetcher.FetchEPG(context.Background())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if channels[0].Logo() != "" {
			t.Errorf("expected empty logo, got %q", channels[0].Logo())
		}
	})

	t.Run("HTTP 404 error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		fetcher := NewEPGXMLFetcher(server.URL, nil)
		_, err := fetcher.FetchEPG(context.Background())

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if !strings.Contains(err.Error(), "unexpected HTTP status") {
			t.Errorf("expected HTTP status error, got: %v", err)
		}
		if !strings.Contains(err.Error(), "404") {
			t.Errorf("expected 404 in error message, got: %v", err)
		}
	})

	t.Run("HTTP 500 error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		fetcher := NewEPGXMLFetcher(server.URL, nil)
		_, err := fetcher.FetchEPG(context.Background())

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if !strings.Contains(err.Error(), "unexpected HTTP status") {
			t.Errorf("expected HTTP status error, got: %v", err)
		}
		if !strings.Contains(err.Error(), "500") {
			t.Errorf("expected 500 in error message, got: %v", err)
		}
	})

	t.Run("malformed XML", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("<invalid><xml"))
		}))
		defer server.Close()

		fetcher := NewEPGXMLFetcher(server.URL, nil)
		_, err := fetcher.FetchEPG(context.Background())

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if !strings.Contains(err.Error(), "parsing EPG XML") {
			t.Errorf("expected XML parsing error, got: %v", err)
		}
	})

	t.Run("context timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		fetcher := NewEPGXMLFetcher(server.URL, nil)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		_, err := fetcher.FetchEPG(ctx)

		if err == nil {
			t.Fatal("expected timeout error, got nil")
		}

		if !errors.Is(err, context.DeadlineExceeded) && !strings.Contains(err.Error(), "context deadline exceeded") {
			t.Errorf("expected context deadline error, got: %v", err)
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		fetcher := NewEPGXMLFetcher(server.URL, nil)
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := fetcher.FetchEPG(ctx)

		if err == nil {
			t.Fatal("expected cancellation error, got nil")
		}

		if !errors.Is(err, context.Canceled) && !strings.Contains(err.Error(), "context canceled") {
			t.Errorf("expected context canceled error, got: %v", err)
		}
	})

	t.Run("invalid URL", func(t *testing.T) {
		fetcher := NewEPGXMLFetcher("://invalid-url", nil)
		_, err := fetcher.FetchEPG(context.Background())

		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("empty channel ID fails domain validation", func(t *testing.T) {
		xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<tv>
	<channel id="">
		<display-name>Empty ID Channel</display-name>
	</channel>
</tv>`

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(xmlData))
		}))
		defer server.Close()

		fetcher := NewEPGXMLFetcher(server.URL, nil)
		_, err := fetcher.FetchEPG(context.Background())

		if err == nil {
			t.Fatal("expected domain validation error, got nil")
		}

		if !strings.Contains(err.Error(), "creating domain channel") {
			t.Errorf("expected domain channel creation error, got: %v", err)
		}
	})

	t.Run("whitespace-only channel ID fails domain validation", func(t *testing.T) {
		xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<tv>
	<channel id="   ">
		<display-name>Whitespace ID Channel</display-name>
	</channel>
</tv>`

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(xmlData))
		}))
		defer server.Close()

		fetcher := NewEPGXMLFetcher(server.URL, nil)
		_, err := fetcher.FetchEPG(context.Background())

		if err == nil {
			t.Fatal("expected domain validation error, got nil")
		}
	})

	t.Run("empty response body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(""))
		}))
		defer server.Close()

		fetcher := NewEPGXMLFetcher(server.URL, nil)
		_, err := fetcher.FetchEPG(context.Background())

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if !strings.Contains(err.Error(), "parsing EPG XML") {
			t.Errorf("expected XML parsing error, got: %v", err)
		}
	})

	t.Run("no channels in XML", func(t *testing.T) {
		xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<tv generator-info-name="test">
</tv>`

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(xmlData))
		}))
		defer server.Close()

		fetcher := NewEPGXMLFetcher(server.URL, nil)
		channels, err := fetcher.FetchEPG(context.Background())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(channels) != 0 {
			t.Errorf("expected empty channel list, got %d channels", len(channels))
		}
	})
}
