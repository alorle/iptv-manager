package driven

import (
	"context"
	"errors"
	"fmt"
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"response":{"playback_url":"http://example.com/stream","stat_url":"http://example.com/stat","command_url":"http://example.com/cmd"}}`))
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	adapter := NewAceStreamHTTPAdapter(server.URL, logger)
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
		_, _ = w.Write([]byte(`{"response":{"playback_url":"http://example.com/stream","stat_url":"http://example.com/stat/abc/def","command_url":"http://example.com/cmd/abc/def"}}`))
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

	// Verify session was stored
	adapter.sessionsMu.RLock()
	session, ok := adapter.sessions["test-pid"]
	adapter.sessionsMu.RUnlock()

	if !ok {
		t.Fatal("expected session to be stored for test-pid")
	}
	if session.statURL != "http://example.com/stat/abc/def" {
		t.Errorf("expected stat URL 'http://example.com/stat/abc/def', got '%s'", session.statURL)
	}
	if session.commandURL != "http://example.com/cmd/abc/def" {
		t.Errorf("expected command URL 'http://example.com/cmd/abc/def', got '%s'", session.commandURL)
	}
}

func TestAceStreamHTTPAdapter_GetStats_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/ace/getstream", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// stat_url and command_url will be overridden below
		_, _ = w.Write([]byte(`{"response":{"playback_url":"http://example.com/stream","stat_url":"STAT_URL_PLACEHOLDER","command_url":"http://example.com/cmd"}}`))
	})
	mux.HandleFunc("/ace/stat/abc/def", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"response":{"status":"dl","peers":5,"speed_down":1000,"speed_up":500,"downloaded":10000,"uploaded":5000},"error":null}`))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	// Re-register with correct stat_url pointing to the test server
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	adapter := NewAceStreamHTTPAdapter(server.URL, logger)

	// Manually register a session with the correct stat URL
	adapter.sessionsMu.Lock()
	adapter.sessions["test-pid"] = engineSession{
		statURL:    server.URL + "/ace/stat/abc/def",
		commandURL: server.URL + "/ace/cmd/abc/def",
	}
	adapter.sessionsMu.Unlock()

	ctx := context.Background()
	stats, err := adapter.GetStats(ctx, "test-pid")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stats.Status != "dl" {
		t.Errorf("expected status 'dl', got '%s'", stats.Status)
	}
	if stats.Peers != 5 {
		t.Errorf("expected peers 5, got %d", stats.Peers)
	}
	if stats.SpeedDown != 1000 {
		t.Errorf("expected speed_down 1000, got %d", stats.SpeedDown)
	}
}

func TestAceStreamHTTPAdapter_GetStats_Timeout(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/ace/stat/abc/def", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"response":{"status":"active","peers":5},"error":null}`))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	adapter := NewAceStreamHTTPAdapter(server.URL, logger)
	adapter.getStatsTimeout = 500 * time.Millisecond

	// Register a session
	adapter.sessionsMu.Lock()
	adapter.sessions["test-pid"] = engineSession{
		statURL:    server.URL + "/ace/stat/abc/def",
		commandURL: server.URL + "/ace/cmd/abc/def",
	}
	adapter.sessionsMu.Unlock()

	ctx := context.Background()
	_, err := adapter.GetStats(ctx, "test-pid")

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}

	if !strings.Contains(err.Error(), "timed out") && !isTimeoutError(err) {
		t.Errorf("expected timeout error, got: %v", err)
	}
}

func TestAceStreamHTTPAdapter_GetStats_NoSession(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	adapter := NewAceStreamHTTPAdapter("http://localhost:6878", logger)

	ctx := context.Background()
	_, err := adapter.GetStats(ctx, "nonexistent-pid")

	if err == nil {
		t.Fatal("expected error for missing session, got nil")
	}

	if !strings.Contains(err.Error(), "no active session") {
		t.Errorf("expected 'no active session' error, got: %v", err)
	}
}

func TestAceStreamHTTPAdapter_StopStream_Success(t *testing.T) {
	var receivedMethod string
	mux := http.NewServeMux()
	mux.HandleFunc("/ace/cmd/abc/def", func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.URL.Query().Get("method")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"response":"ok","error":null}`))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	adapter := NewAceStreamHTTPAdapter(server.URL, logger)

	// Register a session
	adapter.sessionsMu.Lock()
	adapter.sessions["test-pid"] = engineSession{
		statURL:    server.URL + "/ace/stat/abc/def",
		commandURL: server.URL + "/ace/cmd/abc/def",
	}
	adapter.sessionsMu.Unlock()

	ctx := context.Background()
	err := adapter.StopStream(ctx, "test-pid")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedMethod != "stop" {
		t.Errorf("expected method=stop, got '%s'", receivedMethod)
	}

	// Verify session was removed
	adapter.sessionsMu.RLock()
	_, ok := adapter.sessions["test-pid"]
	adapter.sessionsMu.RUnlock()

	if ok {
		t.Error("expected session to be removed after StopStream")
	}
}

func TestAceStreamHTTPAdapter_StopStream_Timeout(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/ace/cmd/abc/def", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	adapter := NewAceStreamHTTPAdapter(server.URL, logger)
	adapter.stopStreamTimeout = 500 * time.Millisecond

	// Register a session
	adapter.sessionsMu.Lock()
	adapter.sessions["test-pid"] = engineSession{
		statURL:    server.URL + "/ace/stat/abc/def",
		commandURL: server.URL + "/ace/cmd/abc/def",
	}
	adapter.sessionsMu.Unlock()

	ctx := context.Background()
	err := adapter.StopStream(ctx, "test-pid")

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}

	if !strings.Contains(err.Error(), "timed out") && !isTimeoutError(err) {
		t.Errorf("expected timeout error, got: %v", err)
	}
}

func TestAceStreamHTTPAdapter_StopStream_NoSession(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	adapter := NewAceStreamHTTPAdapter("http://localhost:6878", logger)

	ctx := context.Background()
	err := adapter.StopStream(ctx, "nonexistent-pid")

	// StopStream with no session should not return an error (best-effort)
	if err != nil {
		t.Fatalf("expected nil error for missing session, got: %v", err)
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "video/mp2t")
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
	os.Setenv("ACESTREAM_START_TIMEOUT", "2s")
	defer os.Unsetenv("ACESTREAM_START_TIMEOUT")

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	adapter := NewAceStreamHTTPAdapter("http://localhost:6878", logger)

	if adapter.startStreamTimeout != 2*time.Second {
		t.Errorf("expected startStreamTimeout to be 2s, got %v", adapter.startStreamTimeout)
	}
}

func TestAceStreamHTTPAdapter_StartStreamTimeout_InvalidEnv(t *testing.T) {
	os.Setenv("ACESTREAM_START_TIMEOUT", "invalid")
	defer os.Unsetenv("ACESTREAM_START_TIMEOUT")

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	adapter := NewAceStreamHTTPAdapter("http://localhost:6878", logger)

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
			expected: false,
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

func TestAceStreamHTTPAdapter_StartStream_StoresSession(t *testing.T) {
	// Full integration test: StartStream → GetStats → StopStream
	mux := http.NewServeMux()
	mux.HandleFunc("/ace/getstream", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		statURL := fmt.Sprintf("http://%s/ace/stat/hash123/client456", r.Host)
		cmdURL := fmt.Sprintf("http://%s/ace/cmd/hash123/client456", r.Host)
		_, _ = fmt.Fprintf(w, `{"response":{"playback_url":"http://%s/ace/r/hash123/client456","stat_url":"%s","command_url":"%s"},"error":null}`, r.Host, statURL, cmdURL)
	})
	mux.HandleFunc("/ace/stat/hash123/client456", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"response":{"status":"dl","peers":10,"speed_down":50000,"speed_up":1000,"downloaded":100000,"uploaded":5000},"error":null}`))
	})
	mux.HandleFunc("/ace/cmd/hash123/client456", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("method") != "stop" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"response":"ok","error":null}`))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	adapter := NewAceStreamHTTPAdapter(server.URL, logger)

	ctx := context.Background()

	// Step 1: Start stream
	streamURL, err := adapter.StartStream(ctx, "hash123", "pid-1")
	if err != nil {
		t.Fatalf("StartStream: unexpected error: %v", err)
	}
	if !strings.Contains(streamURL, "/ace/r/hash123/client456") {
		t.Errorf("unexpected stream URL: %s", streamURL)
	}

	// Step 2: Get stats using the same PID
	stats, err := adapter.GetStats(ctx, "pid-1")
	if err != nil {
		t.Fatalf("GetStats: unexpected error: %v", err)
	}
	if stats.Peers != 10 {
		t.Errorf("expected 10 peers, got %d", stats.Peers)
	}
	if stats.Status != "dl" {
		t.Errorf("expected status 'dl', got '%s'", stats.Status)
	}

	// Step 3: Stop stream using the same PID
	err = adapter.StopStream(ctx, "pid-1")
	if err != nil {
		t.Fatalf("StopStream: unexpected error: %v", err)
	}

	// Step 4: Verify session was cleaned up
	adapter.sessionsMu.RLock()
	_, ok := adapter.sessions["pid-1"]
	adapter.sessionsMu.RUnlock()
	if ok {
		t.Error("expected session to be removed after StopStream")
	}
}
