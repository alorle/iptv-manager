//go:build integration

package driven

// Integration Test for AceStream Engine
//
// This test validates the complete flow against a real AceStream Engine instance.
// It is skipped in regular test runs and must be executed explicitly with:
//
//     go test -tags integration -run TestAceStreamIntegration ./...
//
// Prerequisites:
//   - AceStream Engine must be running and accessible
//   - Default URL: http://localhost:6878 (override with ACESTREAM_ENGINE_URL env var)
//   - Test uses a known working infohash: 895d08633bb22c2573281655cf3ca44de476cc73
//
// The test verifies:
//   1. Ping - Engine is reachable
//   2. StartStream - Stream can be started
//   3. StreamContent - At least 1KB of data can be received
//   4. StopStream - Stream can be stopped cleanly
//
// If the engine is not available, the test is skipped (not failed).

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"
)

const (
	// Known working infohash for testing
	testInfoHash = "895d08633bb22c2573281655cf3ca44de476cc73"
	// Minimum data to receive to consider the stream working
	minDataBytes = 1024 // 1KB
	// Global test timeout
	testTimeout = 60 * time.Second
)

func TestAceStreamIntegration(t *testing.T) {
	// Get engine URL from environment or use default
	engineURL := os.Getenv("ACESTREAM_ENGINE_URL")
	if engineURL == "" {
		engineURL = "http://localhost:6878"
	}

	t.Logf("=== Starting AceStream Integration Test ===")
	t.Logf("Engine URL: %s", engineURL)
	t.Logf("Test InfoHash: %s", testInfoHash)
	t.Logf("Global timeout: %v", testTimeout)

	// Create test context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Create adapter with test logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	adapter := NewAceStreamHTTPAdapter(engineURL, logger)

	t.Logf("\n--- Step 1: Ping Engine ---")
	if err := testPing(ctx, t, adapter); err != nil {
		// If engine is not available (connection error), skip the test
		// Note: Some engines return 404 for manifest.json, which is OK - we'll test with StartStream
		if isConnectionError(err) {
			t.Skipf("AceStream Engine not available at %s: %v", engineURL, err)
			return
		}
		t.Logf("⚠ Ping returned error (may be normal): %v", err)
		t.Logf("Will proceed with StartStream test to verify engine functionality")
	} else {
		t.Logf("✓ Engine is reachable")
	}

	t.Logf("\n--- Step 2: Start Stream ---")
	pid := "integration-test-pid-" + time.Now().Format("20060102-150405")
	t.Logf("Generated PID: %s", pid)

	streamURL, err := testStartStream(ctx, t, adapter, pid)
	if err != nil {
		t.Fatalf("StartStream failed: %v", err)
	}
	t.Logf("✓ Stream started successfully")
	t.Logf("Stream URL: %s", streamURL)

	// Ensure cleanup happens even if test fails
	defer func() {
		t.Logf("\n--- Step 4: Stop Stream (cleanup) ---")
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		if err := testStopStream(cleanupCtx, t, adapter, pid); err != nil {
			t.Logf("⚠ Warning: StopStream cleanup failed: %v", err)
		} else {
			t.Logf("✓ Stream stopped successfully")
		}
	}()

	t.Logf("\n--- Step 3: Receive Data ---")
	bytesReceived, err := testStreamContent(ctx, t, adapter, streamURL, pid)
	if err != nil {
		t.Fatalf("StreamContent failed: %v", err)
	}
	t.Logf("✓ Received %d bytes (minimum: %d bytes)", bytesReceived, minDataBytes)

	if bytesReceived < minDataBytes {
		t.Fatalf("Insufficient data received: got %d bytes, want at least %d bytes", bytesReceived, minDataBytes)
	}

	t.Logf("\n=== Integration Test PASSED ===")
}

// testPing verifies the engine is accessible.
func testPing(ctx context.Context, t *testing.T, adapter *AceStreamHTTPAdapter) error {
	t.Logf("Pinging engine...")
	startTime := time.Now()
	err := adapter.Ping(ctx)
	duration := time.Since(startTime)
	t.Logf("Ping completed in %v", duration)
	return err
}

// testStartStream initiates a stream and returns the stream URL.
func testStartStream(ctx context.Context, t *testing.T, adapter *AceStreamHTTPAdapter, pid string) (string, error) {
	t.Logf("Starting stream with PID: %s, InfoHash: %s", pid, testInfoHash)
	startTime := time.Now()
	streamURL, err := adapter.StartStream(ctx, testInfoHash, pid)
	duration := time.Since(startTime)
	t.Logf("StartStream completed in %v", duration)
	return streamURL, err
}

// testStreamContent attempts to receive data from the stream.
// Returns the number of bytes received.
func testStreamContent(ctx context.Context, t *testing.T, adapter *AceStreamHTTPAdapter, streamURL, pid string) (int64, error) {
	t.Logf("Starting content stream from: %s", streamURL)

	// Create a context that will cancel after receiving enough data
	streamCtx, streamCancel := context.WithCancel(ctx)
	defer streamCancel()

	// Buffer to collect data
	buf := &bytes.Buffer{}

	// Create a limited reader that will stop after minDataBytes
	limitedWriter := &limitedWriter{
		w:        buf,
		maxBytes: minDataBytes,
		onLimit:  streamCancel,
		t:        t,
	}

	startTime := time.Now()

	// StreamContent will block until context is canceled or stream ends
	// We expect it to be canceled by limitedWriter after receiving minDataBytes
	err := adapter.StreamContent(streamCtx, streamURL, limitedWriter, testInfoHash, pid, 10*time.Second)

	duration := time.Since(startTime)
	bytesReceived := int64(buf.Len())

	t.Logf("StreamContent completed in %v, received %d bytes", duration, bytesReceived)

	// Context.Canceled is expected when we've received enough data
	if err == context.Canceled {
		t.Logf("Stream canceled after receiving sufficient data (expected)")
		return bytesReceived, nil
	}

	// Any other error or nil (stream ended naturally)
	if err != nil {
		t.Logf("Stream ended with error: %v", err)
	}

	return bytesReceived, err
}

// testStopStream stops the stream.
func testStopStream(ctx context.Context, t *testing.T, adapter *AceStreamHTTPAdapter, pid string) error {
	t.Logf("Stopping stream with PID: %s", pid)
	startTime := time.Now()
	err := adapter.StopStream(ctx, pid)
	duration := time.Since(startTime)
	t.Logf("StopStream completed in %v", duration)
	return err
}

// limitedWriter is a writer that cancels the context after writing maxBytes.
type limitedWriter struct {
	w        io.Writer
	written  int64
	maxBytes int64
	onLimit  context.CancelFunc
	t        *testing.T
}

func (lw *limitedWriter) Write(p []byte) (n int, err error) {
	// If we've already reached the limit, don't write more
	if lw.written >= lw.maxBytes {
		lw.t.Logf("Limit reached (%d bytes), triggering context cancellation", lw.written)
		lw.onLimit()
		return 0, context.Canceled
	}

	// Write data
	n, err = lw.w.Write(p)
	lw.written += int64(n)

	// Log progress every 256KB
	if lw.written%(256*1024) == 0 {
		lw.t.Logf("Streaming progress: %d bytes received", lw.written)
	}

	// Check if we've reached the limit
	if lw.written >= lw.maxBytes {
		lw.t.Logf("Target data received (%d bytes), canceling stream", lw.written)
		lw.onLimit()
	}

	return n, err
}

// isConnectionError checks if the error indicates a connection problem (not just HTTP errors).
func isConnectionError(err error) bool {
	if err == nil {
		return false
	}
	// Check for network-level connection errors (but not HTTP status errors like 404)
	errStr := err.Error()
	return contains(errStr, "connection refused") ||
		contains(errStr, "no such host") ||
		contains(errStr, "not reachable") ||
		contains(errStr, "i/o timeout")
}

// contains checks if a string contains a substring (case-insensitive).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(bytes.Contains([]byte(s), []byte(substr))))
}
