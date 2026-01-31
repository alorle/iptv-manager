package aceproxy

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

const (
	defaultEngineURL     = "http://localhost:6878"
	defaultTimeout       = 30 * time.Second
	getStreamEndpoint    = "/ace/getstream"
	envEngineURL         = "ACESTREAM_ENGINE_URL"
	envTimeout           = "ACESTREAM_TIMEOUT"
)

// Client represents an Ace Stream Engine HTTP client
type Client struct {
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
}

// Config holds configuration for the Ace Stream Engine client
type Config struct {
	EngineURL string
	Timeout   time.Duration
}

// NewClient creates a new Ace Stream Engine client with the provided configuration
func NewClient(cfg *Config) *Client {
	if cfg == nil {
		cfg = &Config{}
	}

	// Use configured URL or environment variable or default
	engineURL := cfg.EngineURL
	if engineURL == "" {
		engineURL = os.Getenv(envEngineURL)
	}
	if engineURL == "" {
		engineURL = defaultEngineURL
	}

	// Use configured timeout or environment variable or default
	timeout := cfg.Timeout
	if timeout == 0 {
		if envTimeoutStr := os.Getenv(envTimeout); envTimeoutStr != "" {
			if envTimeout, err := strconv.Atoi(envTimeoutStr); err == nil {
				timeout = time.Duration(envTimeout) * time.Second
			}
		}
	}
	if timeout == 0 {
		timeout = defaultTimeout
	}

	return &Client{
		baseURL: engineURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		timeout: timeout,
	}
}

// GetStreamParams holds parameters for the GetStream request
type GetStreamParams struct {
	ContentID string // id parameter - Ace Stream content ID
	ProductID string // pid parameter - Product ID
}

// GetStreamResponse represents the response from the GetStream endpoint
type GetStreamResponse struct {
	StreamURL  string
	StatusCode int
}

// GetStream requests a stream URL from the Ace Stream Engine for the given content ID
func (c *Client) GetStream(ctx context.Context, params GetStreamParams) (*GetStreamResponse, error) {
	if params.ContentID == "" {
		return nil, fmt.Errorf("content ID is required")
	}

	// Build request URL with query parameters
	reqURL, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid engine URL: %w", err)
	}

	reqURL.Path = getStreamEndpoint
	query := reqURL.Query()
	query.Set("id", params.ContentID)
	if params.ProductID != "" {
		query.Set("pid", params.ProductID)
	}
	reqURL.RawQuery = query.Encode()

	// Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ace Stream Engine at %s: %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	// Check for successful response
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Ace Stream Engine returned status %d: %s", resp.StatusCode, resp.Status)
	}

	// The stream URL is typically returned in the response body or as a redirect
	// For now, return the final URL after any redirects
	streamURL := resp.Request.URL.String()

	return &GetStreamResponse{
		StreamURL:  streamURL,
		StatusCode: resp.StatusCode,
	}, nil
}

// Close closes the client and releases resources
func (c *Client) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}
