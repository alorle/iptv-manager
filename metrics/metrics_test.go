package metrics

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func TestMetricsEndpoint(t *testing.T) {
	// Initialize metrics - including vector metrics to ensure they appear
	SetStreamsActive(0)
	SetClientsConnected(0)
	RecordUpstreamReconnection("init")
	RecordUpstreamError("init", "init")
	SetCircuitBreakerState("init", "CLOSED")
	RecordCircuitBreakerTrip("init")
	RecordHealthCheckFailure()

	// Create a test server with the Prometheus handler
	handler := promhttp.Handler()
	server := httptest.NewServer(handler)
	defer server.Close()

	// Make a request to the /metrics endpoint
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to get metrics: %v", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			t.Errorf("failed to close response body: %v", closeErr)
		}
	}()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	output := string(body)

	// Check for expected metrics
	expectedMetrics := []string{
		"iptv_streams_active",
		"iptv_clients_connected",
		"iptv_upstream_reconnections_total",
		"iptv_upstream_errors_total",
		"iptv_circuit_breaker_state",
		"iptv_circuit_breaker_trips_total",
		"iptv_health_check_failures_total",
	}

	for _, metric := range expectedMetrics {
		if !strings.Contains(output, metric) {
			t.Errorf("Expected metric %s not found in output", metric)
		}
	}
}

func TestMetricsValues(t *testing.T) {
	// Set some test values
	SetStreamsActive(3)
	SetClientsConnected(10)
	RecordUpstreamReconnection("test-content-123")
	RecordUpstreamError("test-content-123", "connection_lost")
	SetCircuitBreakerState("test-content-123", "CLOSED")
	RecordCircuitBreakerTrip("test-content-456")
	RecordHealthCheckFailure()

	// Create a test server
	handler := promhttp.Handler()
	server := httptest.NewServer(handler)
	defer server.Close()

	// Make a request
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to get metrics: %v", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			t.Errorf("failed to close response body: %v", closeErr)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	output := string(body)

	// Check that values are present
	tests := []struct {
		name     string
		contains string
	}{
		{"streams_active", "iptv_streams_active 3"},
		{"clients_connected", "iptv_clients_connected 10"},
		{"reconnections", "iptv_upstream_reconnections_total"},
		{"errors", "iptv_upstream_errors_total"},
		{"cb_state", "iptv_circuit_breaker_state"},
		{"cb_trips", "iptv_circuit_breaker_trips_total"},
		{"health_check", "iptv_health_check_failures_total"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(output, tt.contains) {
				t.Errorf("Expected to find %s in output", tt.contains)
			}
		})
	}
}

func TestCircuitBreakerStateValues(t *testing.T) {
	tests := []struct {
		state string
		value string
	}{
		{"CLOSED", "0"},
		{"OPEN", "1"},
		{"HALF-OPEN", "2"},
	}

	for _, tt := range tests {
		t.Run(tt.state, func(t *testing.T) {
			// Set the state
			SetCircuitBreakerState("test-cb", tt.state)

			// Create test server
			handler := promhttp.Handler()
			server := httptest.NewServer(handler)
			defer server.Close()

			// Get metrics
			resp, err := http.Get(server.URL)
			if err != nil {
				t.Fatalf("Failed to get metrics: %v", err)
			}
			defer func() {
				if closeErr := resp.Body.Close(); closeErr != nil {
					t.Errorf("failed to close response body: %v", closeErr)
				}
			}()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response body: %v", err)
			}

			output := string(body)

			// Check for the expected value
			expectedLine := `iptv_circuit_breaker_state{content_id="test-cb"} ` + tt.value
			if !strings.Contains(output, expectedLine) {
				t.Errorf("Expected to find %s in output for state %s", expectedLine, tt.state)
			}
		})
	}
}

func TestMetricsLabels(t *testing.T) {
	// Test that metrics with labels work correctly
	RecordUpstreamReconnection("content-1")
	RecordUpstreamReconnection("content-2")
	RecordUpstreamError("content-1", "timeout")
	RecordUpstreamError("content-1", "connection_lost")

	// Create test server
	handler := promhttp.Handler()
	server := httptest.NewServer(handler)
	defer server.Close()

	// Get metrics
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to get metrics: %v", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			t.Errorf("failed to close response body: %v", closeErr)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	output := string(body)

	// Check that labels are present
	expectedLabels := []string{
		`content_id="content-1"`,
		`content_id="content-2"`,
		`error_type="timeout"`,
		`error_type="connection_lost"`,
	}

	for _, label := range expectedLabels {
		if !strings.Contains(output, label) {
			t.Errorf("Expected to find label %s in output", label)
		}
	}
}
