package application

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alorle/iptv-manager/internal/port/driven"
)

func TestAceStreamProxyService_StreamToClient(t *testing.T) {
	t.Run("successfully streams to client", func(t *testing.T) {
		streamContent := []byte("mock video stream content")
		mockEngine := &mockAceStreamEngine{
			startStreamFunc: func(ctx context.Context, infoHash, pid string) (string, error) {
				if infoHash != "test-infohash" {
					t.Errorf("expected infohash 'test-infohash', got %q", infoHash)
				}
				return "http://localhost:6878/stream/test", nil
			},
			streamContentFunc: func(ctx context.Context, streamURL string, dst io.Writer, infoHash, pid string, writeTimeout time.Duration) error {
				if streamURL != "http://localhost:6878/stream/test" {
					t.Errorf("expected stream URL 'http://localhost:6878/stream/test', got %q", streamURL)
				}
				_, err := dst.Write(streamContent)
				return err
			},
			stopStreamFunc: func(ctx context.Context, pid string) error {
				return nil
			},
		}

		service := NewAceStreamProxyService(mockEngine, slog.Default(), 10*time.Second)
		var buf bytes.Buffer

		err := service.StreamToClient(context.Background(), "test-infohash", &buf)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if !bytes.Equal(buf.Bytes(), streamContent) {
			t.Errorf("expected content %q, got %q", string(streamContent), buf.String())
		}

		// Verify cleanup happened
		activeStreams := service.GetActiveStreams()
		if len(activeStreams) != 0 {
			t.Errorf("expected 0 active streams after completion, got %d", len(activeStreams))
		}
	})

	t.Run("returns error for empty infohash", func(t *testing.T) {
		mockEngine := &mockAceStreamEngine{}
		service := NewAceStreamProxyService(mockEngine, slog.Default(), 10*time.Second)
		var buf bytes.Buffer

		err := service.StreamToClient(context.Background(), "", &buf)
		if !errors.Is(err, ErrInvalidInfoHash) {
			t.Errorf("expected ErrInvalidInfoHash, got %v", err)
		}
	})

	t.Run("returns error when engine fails to start stream", func(t *testing.T) {
		expectedErr := errors.New("engine connection failed")
		mockEngine := &mockAceStreamEngine{
			startStreamFunc: func(ctx context.Context, infoHash, pid string) (string, error) {
				return "", expectedErr
			},
		}

		service := NewAceStreamProxyService(mockEngine, slog.Default(), 10*time.Second)
		var buf bytes.Buffer

		err := service.StreamToClient(context.Background(), "test-infohash", &buf)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "failed to start engine stream") {
			t.Errorf("expected error about engine stream, got %v", err)
		}
	})

	t.Run("multiple clients share the same stream session", func(t *testing.T) {
		var streamStartCount int
		var mu sync.Mutex
		streamContent := []byte("shared stream content")
		blockStream := make(chan struct{})

		mockEngine := &mockAceStreamEngine{
			startStreamFunc: func(ctx context.Context, infoHash, pid string) (string, error) {
				mu.Lock()
				streamStartCount++
				mu.Unlock()
				return "http://localhost:6878/stream/test", nil
			},
			streamContentFunc: func(ctx context.Context, streamURL string, dst io.Writer, infoHash, pid string, writeTimeout time.Duration) error {
				<-blockStream // Block until we're ready to finish
				_, err := dst.Write(streamContent)
				return err
			},
			stopStreamFunc: func(ctx context.Context, pid string) error {
				return nil
			},
		}

		service := NewAceStreamProxyService(mockEngine, slog.Default(), 10*time.Second)

		// Start first client
		var buf1 bytes.Buffer
		done1 := make(chan error, 1)
		go func() {
			done1 <- service.StreamToClient(context.Background(), "shared-infohash", &buf1)
		}()

		// Give first client time to initialize
		time.Sleep(100 * time.Millisecond)

		// Verify stream was started only once
		mu.Lock()
		count1 := streamStartCount
		mu.Unlock()
		if count1 != 1 {
			t.Errorf("expected stream to start once, got %d times", count1)
		}

		// Verify one active stream
		activeStreams := service.GetActiveStreams()
		if len(activeStreams) != 1 {
			t.Fatalf("expected 1 active stream, got %d", len(activeStreams))
		}
		if activeStreams[0].ClientCount != 1 {
			t.Errorf("expected 1 client, got %d", activeStreams[0].ClientCount)
		}

		// Start second client with same infohash
		var buf2 bytes.Buffer
		done2 := make(chan error, 1)
		go func() {
			done2 <- service.StreamToClient(context.Background(), "shared-infohash", &buf2)
		}()

		// Give second client time to join
		time.Sleep(100 * time.Millisecond)

		// Verify stream was not started again
		mu.Lock()
		count2 := streamStartCount
		mu.Unlock()
		if count2 != 1 {
			t.Errorf("expected stream to start once total, got %d times", count2)
		}

		// Verify two clients on one stream
		activeStreams = service.GetActiveStreams()
		if len(activeStreams) != 1 {
			t.Fatalf("expected 1 active stream, got %d", len(activeStreams))
		}
		if activeStreams[0].ClientCount != 2 {
			t.Errorf("expected 2 clients, got %d", activeStreams[0].ClientCount)
		}

		// Unblock streams to complete
		close(blockStream)

		// Wait for both clients to complete
		<-done1
		<-done2

		// Verify both received the same content
		if !bytes.Equal(buf1.Bytes(), streamContent) {
			t.Errorf("client 1 got wrong content")
		}
		if !bytes.Equal(buf2.Bytes(), streamContent) {
			t.Errorf("client 2 got wrong content")
		}
	})

	t.Run("last client disconnect stops the stream", func(t *testing.T) {
		stopCalled := false
		mockEngine := &mockAceStreamEngine{
			startStreamFunc: func(ctx context.Context, infoHash, pid string) (string, error) {
				return "http://localhost:6878/stream/test", nil
			},
			streamContentFunc: func(ctx context.Context, streamURL string, dst io.Writer, infoHash, pid string, writeTimeout time.Duration) error {
				_, err := dst.Write([]byte("content"))
				return err
			},
			stopStreamFunc: func(ctx context.Context, pid string) error {
				stopCalled = true
				return nil
			},
		}

		service := NewAceStreamProxyService(mockEngine, slog.Default(), 10*time.Second)
		var buf bytes.Buffer

		err := service.StreamToClient(context.Background(), "test-infohash", &buf)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Give cleanup time to execute
		time.Sleep(50 * time.Millisecond)

		if !stopCalled {
			t.Error("expected StopStream to be called when last client disconnects")
		}

		activeStreams := service.GetActiveStreams()
		if len(activeStreams) != 0 {
			t.Errorf("expected 0 active streams after last client disconnect, got %d", len(activeStreams))
		}
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		mockEngine := &mockAceStreamEngine{
			startStreamFunc: func(ctx context.Context, infoHash, pid string) (string, error) {
				return "http://localhost:6878/stream/test", nil
			},
			streamContentFunc: func(ctx context.Context, streamURL string, dst io.Writer, infoHash, pid string, writeTimeout time.Duration) error {
				<-ctx.Done()
				return ctx.Err()
			},
			stopStreamFunc: func(ctx context.Context, pid string) error {
				return nil
			},
		}

		service := NewAceStreamProxyService(mockEngine, slog.Default(), 10*time.Second)
		ctx, cancel := context.WithCancel(context.Background())
		var buf bytes.Buffer

		// Cancel immediately after starting
		go func() {
			time.Sleep(10 * time.Millisecond)
			cancel()
		}()

		err := service.StreamToClient(ctx, "test-infohash", &buf)
		if err == nil {
			t.Fatal("expected error due to context cancellation, got nil")
		}
		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	})
}

func TestAceStreamProxyService_Reconnection(t *testing.T) {
	t.Run("retries on stream failure", func(t *testing.T) {
		attemptCount := 0
		mockEngine := &mockAceStreamEngine{
			startStreamFunc: func(ctx context.Context, infoHash, pid string) (string, error) {
				return "http://localhost:6878/stream/test", nil
			},
			streamContentFunc: func(ctx context.Context, streamURL string, dst io.Writer, infoHash, pid string, writeTimeout time.Duration) error {
				attemptCount++
				if attemptCount == 1 {
					// Fail first attempt
					return errors.New("network error")
				}
				// Succeed on retry
				_, err := dst.Write([]byte("content after retry"))
				return err
			},
			stopStreamFunc: func(ctx context.Context, pid string) error {
				return nil
			},
		}

		service := NewAceStreamProxyService(mockEngine, slog.Default(), 10*time.Second)
		var buf bytes.Buffer

		err := service.StreamToClient(context.Background(), "test-infohash", &buf)
		if err != nil {
			t.Fatalf("expected no error after retry, got %v", err)
		}

		if attemptCount < 2 {
			t.Errorf("expected at least 2 attempts, got %d", attemptCount)
		}

		if buf.String() != "content after retry" {
			t.Errorf("expected 'content after retry', got %q", buf.String())
		}
	})

	t.Run("fails after max retries", func(t *testing.T) {
		mockEngine := &mockAceStreamEngine{
			startStreamFunc: func(ctx context.Context, infoHash, pid string) (string, error) {
				return "http://localhost:6878/stream/test", nil
			},
			streamContentFunc: func(ctx context.Context, streamURL string, dst io.Writer, infoHash, pid string, writeTimeout time.Duration) error {
				return errors.New("persistent error")
			},
			stopStreamFunc: func(ctx context.Context, pid string) error {
				return nil
			},
		}

		service := NewAceStreamProxyService(mockEngine, slog.Default(), 10*time.Second)
		var buf bytes.Buffer

		err := service.StreamToClient(context.Background(), "test-infohash", &buf)
		if err == nil {
			t.Fatal("expected error after max retries, got nil")
		}
		if !strings.Contains(err.Error(), "failed after") {
			t.Errorf("expected error about max retries, got %v", err)
		}
	})
}

func TestAceStreamProxyService_GetActiveStreams(t *testing.T) {
	t.Run("returns active streams info", func(t *testing.T) {
		blockChan := make(chan struct{})
		mockEngine := &mockAceStreamEngine{
			startStreamFunc: func(ctx context.Context, infoHash, pid string) (string, error) {
				return "http://localhost:6878/stream/" + infoHash, nil
			},
			streamContentFunc: func(ctx context.Context, streamURL string, dst io.Writer, infoHash, pid string, writeTimeout time.Duration) error {
				<-blockChan // Block until we're done testing
				return nil
			},
			stopStreamFunc: func(ctx context.Context, pid string) error {
				return nil
			},
		}

		service := NewAceStreamProxyService(mockEngine, slog.Default(), 10*time.Second)

		// Start two clients on different infohashes
		go func() {
			var buf bytes.Buffer
			_ = service.StreamToClient(context.Background(), "infohash-1", &buf)
		}()
		go func() {
			var buf bytes.Buffer
			_ = service.StreamToClient(context.Background(), "infohash-2", &buf)
		}()

		// Give clients time to start
		time.Sleep(100 * time.Millisecond)

		activeStreams := service.GetActiveStreams()
		if len(activeStreams) != 2 {
			t.Fatalf("expected 2 active streams, got %d", len(activeStreams))
		}

		// Verify each stream has one client
		for _, info := range activeStreams {
			if info.ClientCount != 1 {
				t.Errorf("expected 1 client per stream, got %d", info.ClientCount)
			}
			if len(info.PIDs) != 1 {
				t.Errorf("expected 1 PID per stream, got %d", len(info.PIDs))
			}
		}

		// Unblock the streams
		close(blockChan)
	})
}

// mockAceStreamEngine is a mock implementation of the AceStreamEngine port for testing.
type mockAceStreamEngine struct {
	startStreamFunc   func(ctx context.Context, infoHash, pid string) (streamURL string, err error)
	getStatsFunc      func(ctx context.Context, pid string) (stats driven.StreamStats, err error)
	stopStreamFunc    func(ctx context.Context, pid string) error
	streamContentFunc func(ctx context.Context, streamURL string, dst io.Writer, infoHash, pid string, writeTimeout time.Duration) error
	pingFunc          func(ctx context.Context) error
}

func (m *mockAceStreamEngine) StartStream(ctx context.Context, infoHash, pid string) (string, error) {
	if m.startStreamFunc != nil {
		return m.startStreamFunc(ctx, infoHash, pid)
	}
	return "http://localhost:6878/stream/mock", nil
}

func (m *mockAceStreamEngine) GetStats(ctx context.Context, pid string) (driven.StreamStats, error) {
	if m.getStatsFunc != nil {
		return m.getStatsFunc(ctx, pid)
	}
	return driven.StreamStats{}, nil
}

func (m *mockAceStreamEngine) StopStream(ctx context.Context, pid string) error {
	if m.stopStreamFunc != nil {
		return m.stopStreamFunc(ctx, pid)
	}
	return nil
}

func (m *mockAceStreamEngine) StreamContent(ctx context.Context, streamURL string, dst io.Writer, infoHash, pid string, writeTimeout time.Duration) error {
	if m.streamContentFunc != nil {
		return m.streamContentFunc(ctx, streamURL, dst, infoHash, pid, writeTimeout)
	}
	return nil
}

func (m *mockAceStreamEngine) Ping(ctx context.Context) error {
	if m.pingFunc != nil {
		return m.pingFunc(ctx)
	}
	return nil
}
