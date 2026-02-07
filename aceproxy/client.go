package aceproxy

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/alorle/iptv-manager/circuitbreaker"
	"github.com/alorle/iptv-manager/logging"
)

const (
	defaultEngineURL    = "http://localhost:6878"
	defaultTimeout      = 30 * time.Second
	getStreamEndpoint   = "/ace/getstream"
	healthCheckEndpoint = "/webui/api/service"
	envEngineURL        = "ACESTREAM_ENGINE_URL"
	envTimeout          = "ACESTREAM_TIMEOUT"
)

// HealthStatus represents the result of a health check
type HealthStatus struct {
	Healthy   bool
	Timestamp time.Time
	Error     error
}

// Client represents an Ace Stream Engine HTTP client
type Client struct {
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration

	// Health check fields
	healthCheckInterval time.Duration
	healthCheckCancel   context.CancelFunc
	healthCheckDone     chan struct{}
	lastHealthStatus    HealthStatus
	healthMu            sync.RWMutex
	circuitBreaker      circuitbreaker.CircuitBreaker
	logger              *log.Logger
	resLogger           *logging.Logger
}

// Config holds configuration for the Ace Stream Engine client
type Config struct {
	EngineURL           string
	Timeout             time.Duration
	HealthCheckInterval time.Duration // 0 to disable health checks
	CircuitBreaker      circuitbreaker.CircuitBreaker
	Logger              *log.Logger
	ResilienceLogger    *logging.Logger // Logger for resilience events
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

	// Setup logger
	logger := cfg.Logger
	if logger == nil {
		logger = log.New(os.Stdout, "[aceproxy] ", log.LstdFlags)
	}

	client := &Client{
		baseURL: engineURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		timeout:             timeout,
		healthCheckInterval: cfg.HealthCheckInterval,
		healthCheckDone:     make(chan struct{}),
		circuitBreaker:      cfg.CircuitBreaker,
		resLogger:           cfg.ResilienceLogger,
		logger:              logger,
		lastHealthStatus: HealthStatus{
			Healthy:   true, // Assume healthy initially
			Timestamp: time.Now(),
		},
	}

	// Start health check goroutine if interval is configured
	if client.healthCheckInterval > 0 {
		ctx, cancel := context.WithCancel(context.Background())
		client.healthCheckCancel = cancel
		go client.runHealthChecks(ctx)
	}

	return client
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
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			c.logger.Printf("warning: failed to close response body: %v", closeErr)
		}
	}()

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

// HealthCheck performs a single health check against the Ace Stream Engine
func (c *Client) HealthCheck(ctx context.Context) error {
	// Build health check URL
	reqURL, err := url.Parse(c.baseURL)
	if err != nil {
		return fmt.Errorf("invalid engine URL: %w", err)
	}

	reqURL.Path = healthCheckEndpoint
	query := reqURL.Query()
	query.Set("method", "get_version")
	reqURL.RawQuery = query.Encode()

	// Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			c.logger.Printf("warning: failed to close response body: %v", closeErr)
		}
	}()

	// Check for successful response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned status %d: %s", resp.StatusCode, resp.Status)
	}

	return nil
}

// runHealthChecks is the background goroutine that performs periodic health checks
func (c *Client) runHealthChecks(ctx context.Context) {
	ticker := time.NewTicker(c.healthCheckInterval)
	defer ticker.Stop()
	defer close(c.healthCheckDone)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.performHealthCheck(ctx)
		}
	}
}

// performHealthCheck executes a single health check and updates the status
func (c *Client) performHealthCheck(ctx context.Context) {
	// Create a timeout context for the health check
	checkCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	err := c.HealthCheck(checkCtx)

	c.healthMu.Lock()
	c.lastHealthStatus = HealthStatus{
		Healthy:   err == nil,
		Timestamp: time.Now(),
		Error:     err,
	}
	c.healthMu.Unlock()

	if err != nil {
		// Log warning for failed health check using resilience logger
		if c.resLogger != nil {
			c.resLogger.LogHealthCheckFailed(err)
		} else {
			c.logger.Printf("WARN: Health check failed: %v", err)
		}

		// If circuit breaker is configured, record the failure
		if c.circuitBreaker != nil {
			// Use Execute with a function that returns the error
			// This increments the circuit breaker's failure counter
			_ = c.circuitBreaker.Execute(func() error {
				return err
			})
		}
	} else {
		// Log debug for successful health check
		c.logger.Printf("DEBUG: Health check succeeded")
	}
}

// GetHealthStatus returns the last recorded health check status
func (c *Client) GetHealthStatus() HealthStatus {
	c.healthMu.RLock()
	defer c.healthMu.RUnlock()
	return c.lastHealthStatus
}

// Close closes the client and releases resources
func (c *Client) Close() error {
	// Stop health check goroutine if running
	if c.healthCheckCancel != nil {
		c.healthCheckCancel()
		<-c.healthCheckDone // Wait for goroutine to finish
	}

	c.httpClient.CloseIdleConnections()
	return nil
}
