package driven

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestAceStreamHTTPAdapter_StartStream_Timeout(t *testing.T) {
	// Create a test server that delays response beyond timeout
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // Delay longer than our test timeout
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"response":{"playback_url":"http://example.com/stream"}}`))
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	adapter := NewAceStreamHTTPAdapter(server.URL, logger)
	// Override the default timeout for testing
	adapter.startStreamTimeout = 500 * time.Millisecond

	ctx := context.Background()
	_, err := adapter.StartStream(ctx, "test-hash", "test-pid")

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}

	if !strings.Contains(err.Error(), "timed out") && !isTimeoutError(err) {
		t.Errorf("expected timeout error, got: %v", err)
	}
}

func TestAceStreamHTTPAdapter_StartStream_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"response":{"playback_url":"http://example.com/stream"}}`))
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	adapter := NewAceStreamHTTPAdapter(server.URL, logger)

	ctx := context.Background()
	streamURL, err := adapter.StartStream(ctx, "test-hash", "test-pid")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if streamURL != "http://example.com/stream" {
		t.Errorf("expected streamURL 'http://example.com/stream', got '%s'", streamURL)
	}
}

func TestAceStreamHTTPAdapter_GetStats_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"response":{"status":"active","peers":5}}`))
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	adapter := NewAceStreamHTTPAdapter(server.URL, logger)
	adapter.getStatsTimeout = 500 * time.Millisecond

	ctx := context.Background()
	_, err := adapter.GetStats(ctx, "test-pid")

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}

	if !strings.Contains(err.Error(), "timed out") && !isTimeoutError(err) {
		t.Errorf("expected timeout error, got: %v", err)
	}
}

func TestAceStreamHTTPAdapter_GetStats_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"response":{"status":"active","peers":5,"speed_down":1000,"speed_up":500,"downloaded":10000,"uploaded":5000}}`))
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	adapter := NewAceStreamHTTPAdapter(server.URL, logger)

	ctx := context.Background()
	stats, err := adapter.GetStats(ctx, "test-pid")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stats.Status != "active" {
		t.Errorf("expected status 'active', got '%s'", stats.Status)
	}
	if stats.Peers != 5 {
		t.Errorf("expected peers 5, got %d", stats.Peers)
	}
}

func TestAceStreamHTTPAdapter_StopStream_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	adapter := NewAceStreamHTTPAdapter(server.URL, logger)
	adapter.stopStreamTimeout = 500 * time.Millisecond

	ctx := context.Background()
	err := adapter.StopStream(ctx, "test-pid")

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}

	if !strings.Contains(err.Error(), "timed out") && !isTimeoutError(err) {
		t.Errorf("expected timeout error, got: %v", err)
	}
}

func TestAceStreamHTTPAdapter_StopStream_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	adapter := NewAceStreamHTTPAdapter(server.URL, logger)

	ctx := context.Background()
	err := adapter.StopStream(ctx, "test-pid")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAceStreamHTTPAdapter_Ping_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"version":"3.0"}`))
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	adapter := NewAceStreamHTTPAdapter(server.URL, logger)
	adapter.pingTimeout = 500 * time.Millisecond

	ctx := context.Background()
	err := adapter.Ping(ctx)

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}

	if !strings.Contains(err.Error(), "timed out") && !isTimeoutError(err) {
		t.Errorf("expected timeout error, got: %v", err)
	}
}

func TestAceStreamHTTPAdapter_Ping_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"version":"3.0"}`))
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	adapter := NewAceStreamHTTPAdapter(server.URL, logger)

	ctx := context.Background()
	err := adapter.Ping(ctx)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAceStreamHTTPAdapter_StreamContent_NoTimeout(t *testing.T) {
	// Stream should not timeout even if it takes longer than typical operation timeouts
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "video/mp2t")
		// Simulate streaming by writing data slowly
		for i := 0; i < 5; i++ {
			_, _ = w.Write([]byte("data"))
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			time.Sleep(200 * time.Millisecond)
		}
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	adapter := NewAceStreamHTTPAdapter(server.URL, logger)

	ctx := context.Background()
	var buf strings.Builder
	err := adapter.StreamContent(ctx, server.URL, &buf, "test-hash", "test-pid", 5*time.Second)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if buf.String() != "datadatadatadatadata" {
		t.Errorf("expected 'datadatadatadatadata', got '%s'", buf.String())
	}
}

func TestAceStreamHTTPAdapter_StartStreamTimeout_FromEnv(t *testing.T) {
	// Set environment variable
	os.Setenv("ACESTREAM_START_TIMEOUT", "2s")
	defer os.Unsetenv("ACESTREAM_START_TIMEOUT")

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	adapter := NewAceStreamHTTPAdapter("http://localhost:6878", logger)

	if adapter.startStreamTimeout != 2*time.Second {
		t.Errorf("expected startStreamTimeout to be 2s, got %v", adapter.startStreamTimeout)
	}
}

func TestAceStreamHTTPAdapter_StartStreamTimeout_InvalidEnv(t *testing.T) {
	// Set invalid environment variable
	os.Setenv("ACESTREAM_START_TIMEOUT", "invalid")
	defer os.Unsetenv("ACESTREAM_START_TIMEOUT")

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	adapter := NewAceStreamHTTPAdapter("http://localhost:6878", logger)

	// Should fall back to default
	if adapter.startStreamTimeout != defaultStartStreamTimeout {
		t.Errorf("expected startStreamTimeout to be default (%v), got %v", defaultStartStreamTimeout, adapter.startStreamTimeout)
	}
}

func TestIsTimeoutError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "context deadline exceeded",
			err:      context.DeadlineExceeded,
			expected: true,
		},
		{
			name:     "wrapped context deadline exceeded",
			err:      errors.New("wrapped: " + context.DeadlineExceeded.Error()),
			expected: false, // wrapped string doesn't match errors.Is
		},
		{
			name:     "non-timeout error",
			err:      errors.New("some other error"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isTimeoutError(tt.err)
			if result != tt.expected {
				t.Errorf("expected %v, got %v for error: %v", tt.expected, result, tt.err)
			}
		})
	}
}

func TestAceStreamHTTPAdapter_DefaultTimeouts(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	adapter := NewAceStreamHTTPAdapter("http://localhost:6878", logger)

	if adapter.startStreamTimeout != defaultStartStreamTimeout {
		t.Errorf("expected startStreamTimeout to be %v, got %v", defaultStartStreamTimeout, adapter.startStreamTimeout)
	}
	if adapter.getStatsTimeout != defaultGetStatsTimeout {
		t.Errorf("expected getStatsTimeout to be %v, got %v", defaultGetStatsTimeout, adapter.getStatsTimeout)
	}
	if adapter.stopStreamTimeout != defaultStopStreamTimeout {
		t.Errorf("expected stopStreamTimeout to be %v, got %v", defaultStopStreamTimeout, adapter.stopStreamTimeout)
	}
	if adapter.pingTimeout != defaultPingTimeout {
		t.Errorf("expected pingTimeout to be %v, got %v", defaultPingTimeout, adapter.pingTimeout)
	}
}
