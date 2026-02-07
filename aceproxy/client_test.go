package aceproxy

import (
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alorle/iptv-manager/circuitbreaker"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name            string
		config          *Config
		envEngineURL    string
		envTimeout      string
		expectedURL     string
		expectedTimeout time.Duration
	}{
		{
			name:            "default configuration",
			config:          nil,
			expectedURL:     defaultEngineURL,
			expectedTimeout: defaultTimeout,
		},
		{
			name: "custom configuration",
			config: &Config{
				EngineURL: "http://custom:8080",
				Timeout:   15 * time.Second,
			},
			expectedURL:     "http://custom:8080",
			expectedTimeout: 15 * time.Second,
		},
		{
			name:            "environment variable URL",
			config:          &Config{},
			envEngineURL:    "http://env-test:9999",
			expectedURL:     "http://env-test:9999",
			expectedTimeout: defaultTimeout,
		},
		{
			name:            "environment variable timeout",
			config:          &Config{},
			envTimeout:      "45",
			expectedURL:     defaultEngineURL,
			expectedTimeout: 45 * time.Second,
		},
		{
			name: "config overrides environment",
			config: &Config{
				EngineURL: "http://config:7777",
				Timeout:   10 * time.Second,
			},
			envEngineURL:    "http://env:8888",
			envTimeout:      "60",
			expectedURL:     "http://config:7777",
			expectedTimeout: 10 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment variables first to ensure clean state
			oldEngineURL := os.Getenv(envEngineURL)
			oldTimeout := os.Getenv(envTimeout)
			os.Unsetenv(envEngineURL)
			os.Unsetenv(envTimeout)
			defer func() {
				if oldEngineURL != "" {
					os.Setenv(envEngineURL, oldEngineURL)
				}
				if oldTimeout != "" {
					os.Setenv(envTimeout, oldTimeout)
				}
			}()

			// Set up test environment variables
			if tt.envEngineURL != "" {
				os.Setenv(envEngineURL, tt.envEngineURL)
			}
			if tt.envTimeout != "" {
				os.Setenv(envTimeout, tt.envTimeout)
			}

			client := NewClient(tt.config)
			defer client.Close()

			if client.baseURL != tt.expectedURL {
				t.Errorf("expected baseURL %s, got %s", tt.expectedURL, client.baseURL)
			}

			if client.timeout != tt.expectedTimeout {
				t.Errorf("expected timeout %v, got %v", tt.expectedTimeout, client.timeout)
			}
		})
	}
}

func TestGetStream(t *testing.T) {
	tests := []struct {
		name           string
		params         GetStreamParams
		serverResponse func(w http.ResponseWriter, r *http.Request)
		expectError    bool
		errorContains  string
		checkResponse  func(t *testing.T, resp *GetStreamResponse)
	}{
		{
			name: "successful request",
			params: GetStreamParams{
				ContentID: "test-content-id",
				ProductID: "test-product-id",
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				// Verify query parameters
				if id := r.URL.Query().Get("id"); id != "test-content-id" {
					t.Errorf("expected id=test-content-id, got %s", id)
				}
				if pid := r.URL.Query().Get("pid"); pid != "test-product-id" {
					t.Errorf("expected pid=test-product-id, got %s", pid)
				}
				w.WriteHeader(http.StatusOK)
			},
			expectError: false,
			checkResponse: func(t *testing.T, resp *GetStreamResponse) {
				if resp.StatusCode != http.StatusOK {
					t.Errorf("expected status 200, got %d", resp.StatusCode)
				}
			},
		},
		{
			name: "request without product ID",
			params: GetStreamParams{
				ContentID: "test-content-id",
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				if id := r.URL.Query().Get("id"); id != "test-content-id" {
					t.Errorf("expected id=test-content-id, got %s", id)
				}
				if pid := r.URL.Query().Get("pid"); pid != "" {
					t.Errorf("expected no pid, got %s", pid)
				}
				w.WriteHeader(http.StatusOK)
			},
			expectError: false,
		},
		{
			name: "missing content ID",
			params: GetStreamParams{
				ProductID: "test-product-id",
			},
			expectError:   true,
			errorContains: "content ID is required",
		},
		{
			name: "engine returns error status",
			params: GetStreamParams{
				ContentID: "test-content-id",
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			expectError:   true,
			errorContains: "returned status 404",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server if server response is provided
			var server *httptest.Server
			var client *Client

			if tt.serverResponse != nil {
				server = httptest.NewServer(http.HandlerFunc(tt.serverResponse))
				defer server.Close()

				client = NewClient(&Config{
					EngineURL: server.URL,
					Timeout:   5 * time.Second,
				})
			} else {
				client = NewClient(&Config{
					EngineURL: "http://localhost:9999",
					Timeout:   1 * time.Second,
				})
			}

			ctx := context.Background()
			resp, err := client.GetStream(ctx, tt.params)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if tt.errorContains != "" && !contains(err.Error(), tt.errorContains) {
					t.Errorf("expected error containing %q, got %q", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if tt.checkResponse != nil && resp != nil {
					tt.checkResponse(t, resp)
				}
			}
		})
	}
}

func TestGetStreamTimeout(t *testing.T) {
	// Create a server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(&Config{
		EngineURL: server.URL,
		Timeout:   500 * time.Millisecond,
	})

	ctx := context.Background()
	_, err := client.GetStream(ctx, GetStreamParams{
		ContentID: "test-content-id",
	})

	if err == nil {
		t.Error("expected timeout error but got none")
	}
}

func TestGetStreamContextCancellation(t *testing.T) {
	// Create a server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(&Config{
		EngineURL: server.URL,
		Timeout:   10 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	_, err := client.GetStream(ctx, GetStreamParams{
		ContentID: "test-content-id",
	})

	if err == nil {
		t.Error("expected context cancellation error but got none")
	}
}

func TestClientClose(t *testing.T) {
	client := NewClient(nil)
	err := client.Close()
	if err != nil {
		t.Errorf("unexpected error closing client: %v", err)
	}
}

func TestHealthCheck(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		expectError    bool
		errorContains  string
	}{
		{
			name: "successful health check",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				// Verify the health check endpoint and method parameter
				if r.URL.Path != healthCheckEndpoint {
					t.Errorf("expected path %s, got %s", healthCheckEndpoint, r.URL.Path)
				}
				if method := r.URL.Query().Get("method"); method != "get_version" {
					t.Errorf("expected method=get_version, got %s", method)
				}
				w.WriteHeader(http.StatusOK)
			},
			expectError: false,
		},
		{
			name: "health check returns error status",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
			},
			expectError:   true,
			errorContains: "health check returned status 503",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			client := NewClient(&Config{
				EngineURL: server.URL,
				Timeout:   5 * time.Second,
			})
			defer client.Close()

			ctx := context.Background()
			err := client.HealthCheck(ctx)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				} else if tt.errorContains != "" && !contains(err.Error(), tt.errorContains) {
					t.Errorf("expected error containing %q, got %q", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestHealthCheckTimeout(t *testing.T) {
	// Create a server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(&Config{
		EngineURL: server.URL,
		Timeout:   500 * time.Millisecond,
	})
	defer client.Close()

	ctx := context.Background()
	err := client.HealthCheck(ctx)

	if err == nil {
		t.Error("expected timeout error but got none")
	}
}

func TestPeriodicHealthChecks(t *testing.T) {
	var requestCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == healthCheckEndpoint {
			requestCount.Add(1)
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	// Discard logs during test
	logger := log.New(os.NewFile(0, os.DevNull), "", 0)

	client := NewClient(&Config{
		EngineURL:           server.URL,
		Timeout:             5 * time.Second,
		HealthCheckInterval: 100 * time.Millisecond,
		Logger:              logger,
	})
	defer client.Close()

	// Wait for at least 3 health checks
	time.Sleep(350 * time.Millisecond)

	count := requestCount.Load()
	if count < 3 {
		t.Errorf("expected at least 3 health checks, got %d", count)
	}
}

func TestHealthCheckDisabled(t *testing.T) {
	var requestCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == healthCheckEndpoint {
			requestCount.Add(1)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create client with health checks disabled (interval = 0)
	client := NewClient(&Config{
		EngineURL:           server.URL,
		Timeout:             5 * time.Second,
		HealthCheckInterval: 0, // Disabled
	})
	defer client.Close()

	// Wait a bit
	time.Sleep(200 * time.Millisecond)

	count := requestCount.Load()
	if count != 0 {
		t.Errorf("expected no health checks, got %d", count)
	}
}

func TestHealthCheckIntegrationWithCircuitBreaker(t *testing.T) {
	var requestCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == healthCheckEndpoint {
			count := requestCount.Add(1)
			// Fail first 3 requests, then succeed
			if count <= 3 {
				w.WriteHeader(http.StatusServiceUnavailable)
			} else {
				w.WriteHeader(http.StatusOK)
			}
		}
	}))
	defer server.Close()

	cb := circuitbreaker.New(circuitbreaker.Config{
		FailureThreshold: 3,
		Timeout:          1 * time.Second,
		HalfOpenRequests: 1,
	})

	// Discard logs during test
	logger := log.New(os.NewFile(0, os.DevNull), "", 0)

	client := NewClient(&Config{
		EngineURL:           server.URL,
		Timeout:             5 * time.Second,
		HealthCheckInterval: 100 * time.Millisecond,
		CircuitBreaker:      cb,
		Logger:              logger,
	})
	defer client.Close()

	// Wait for health checks to run
	time.Sleep(500 * time.Millisecond)

	// Circuit breaker should be open due to failures
	if cb.State() != circuitbreaker.StateOpen {
		t.Errorf("expected circuit breaker to be OPEN, got %s", cb.State())
	}

	count := requestCount.Load()
	if count < 3 {
		t.Errorf("expected at least 3 health checks, got %d", count)
	}
}

func TestGetHealthStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == healthCheckEndpoint {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	// Discard logs during test
	logger := log.New(os.NewFile(0, os.DevNull), "", 0)

	client := NewClient(&Config{
		EngineURL:           server.URL,
		Timeout:             5 * time.Second,
		HealthCheckInterval: 100 * time.Millisecond,
		Logger:              logger,
	})
	defer client.Close()

	// Initial status should be healthy
	status := client.GetHealthStatus()
	if !status.Healthy {
		t.Error("expected initial status to be healthy")
	}

	// Wait for a health check to run
	time.Sleep(150 * time.Millisecond)

	// Status should still be healthy
	status = client.GetHealthStatus()
	if !status.Healthy {
		t.Error("expected status to remain healthy")
	}

	// Timestamp should be recent
	if time.Since(status.Timestamp) > 200*time.Millisecond {
		t.Errorf("expected recent timestamp, got %v", status.Timestamp)
	}
}

func TestHealthCheckFailureUpdatesStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == healthCheckEndpoint {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	}))
	defer server.Close()

	// Discard logs during test
	logger := log.New(os.NewFile(0, os.DevNull), "", 0)

	client := NewClient(&Config{
		EngineURL:           server.URL,
		Timeout:             5 * time.Second,
		HealthCheckInterval: 100 * time.Millisecond,
		Logger:              logger,
	})
	defer client.Close()

	// Wait for a health check to run
	time.Sleep(150 * time.Millisecond)

	// Status should be unhealthy
	status := client.GetHealthStatus()
	if status.Healthy {
		t.Error("expected status to be unhealthy")
	}
	if status.Error == nil {
		t.Error("expected error to be set")
	}
}

func TestClientCloseStopsHealthChecks(t *testing.T) {
	var requestCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == healthCheckEndpoint {
			requestCount.Add(1)
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	// Discard logs during test
	logger := log.New(os.NewFile(0, os.DevNull), "", 0)

	client := NewClient(&Config{
		EngineURL:           server.URL,
		Timeout:             5 * time.Second,
		HealthCheckInterval: 100 * time.Millisecond,
		Logger:              logger,
	})

	// Wait for a few health checks
	time.Sleep(250 * time.Millisecond)
	countBefore := requestCount.Load()

	// Close the client
	client.Close()

	// Wait a bit more
	time.Sleep(250 * time.Millisecond)
	countAfter := requestCount.Load()

	// Count should not have increased significantly after close
	if countAfter > countBefore+1 {
		t.Errorf("expected health checks to stop after close, before=%d after=%d", countBefore, countAfter)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
