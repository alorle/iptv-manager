package epg

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client fetches EPG data from a URL
type Client struct {
	httpClient *http.Client
	epgURL     string
}

// NewClient creates a new EPG client
func NewClient(epgURL string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		epgURL: epgURL,
	}
}

// Fetch retrieves and parses EPG data from the configured URL
func (c *Client) Fetch(ctx context.Context) ([]EPGChannel, error) {
	if c.epgURL == "" {
		return nil, fmt.Errorf("EPG URL not configured")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.epgURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch EPG: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("EPG server returned status %d: %s", resp.StatusCode, string(body))
	}

	// Detect if the response is gzip-compressed
	reader := io.Reader(resp.Body)

	// Check Content-Encoding header
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzipReader.Close()
		reader = gzipReader
	} else if strings.HasSuffix(strings.ToLower(c.epgURL), ".gz") ||
	           strings.HasSuffix(strings.ToLower(c.epgURL), ".xml.gz") {
		// URL ends with .gz, assume it's gzip-compressed
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzipReader.Close()
		reader = gzipReader
	}

	channels, err := ParseXMLTV(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse EPG: %w", err)
	}

	return channels, nil
}
