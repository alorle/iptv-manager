package aceproxy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name           string
		config         *Config
		envEngineURL   string
		envTimeout     string
		expectedURL    string
		expectedTimeout time.Duration
	}{
		{
			name:           "default configuration",
			config:         nil,
			expectedURL:    defaultEngineURL,
			expectedTimeout: defaultTimeout,
		},
		{
			name: "custom configuration",
			config: &Config{
				EngineURL: "http://custom:8080",
				Timeout:   15 * time.Second,
			},
			expectedURL:    "http://custom:8080",
			expectedTimeout: 15 * time.Second,
		},
		{
			name:           "environment variable URL",
			config:         &Config{},
			envEngineURL:   "http://env-test:9999",
			expectedURL:    "http://env-test:9999",
			expectedTimeout: defaultTimeout,
		},
		{
			name:           "environment variable timeout",
			config:         &Config{},
			envTimeout:     "45",
			expectedURL:    defaultEngineURL,
			expectedTimeout: 45 * time.Second,
		},
		{
			name: "config overrides environment",
			config: &Config{
				EngineURL: "http://config:7777",
				Timeout:   10 * time.Second,
			},
			envEngineURL:   "http://env:8888",
			envTimeout:     "60",
			expectedURL:    "http://config:7777",
			expectedTimeout: 10 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment variables
			if tt.envEngineURL != "" {
				os.Setenv(envEngineURL, tt.envEngineURL)
				defer os.Unsetenv(envEngineURL)
			}
			if tt.envTimeout != "" {
				os.Setenv(envTimeout, tt.envTimeout)
				defer os.Unsetenv(envTimeout)
			}

			client := NewClient(tt.config)

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
