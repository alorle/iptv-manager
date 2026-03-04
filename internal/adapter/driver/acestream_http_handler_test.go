package driver

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alorle/iptv-manager/internal/application"
)

// mockProxyService is a minimal stand-in for AceStreamProxyService.
// It writes data to the response writer over a configurable duration.
type mockProxyService struct {
	streamDuration time.Duration
	chunkInterval  time.Duration
}

func (m *mockProxyService) StreamToClient(ctx context.Context, infoHash string, w io.Writer) error {
	ticker := time.NewTicker(m.chunkInterval)
	defer ticker.Stop()

	deadline := time.After(m.streamDuration)
	chunk := []byte("data chunk\n")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline:
			return nil
		case <-ticker.C:
			if _, err := w.Write(chunk); err != nil {
				return err
			}
		}
	}
}

func (m *mockProxyService) GetActiveStreams() []application.StreamInfo {
	return nil
}

func TestAceStreamHTTPHandler_LongStream(t *testing.T) {
	mock := &mockProxyService{
		streamDuration: 3 * time.Second,
		chunkInterval:  500 * time.Millisecond,
	}
	logger := slog.Default()
	handler := NewAceStreamHTTPHandler(mock, logger)

	// Create a test server with WriteTimeout: 0 (no global timeout)
	server := httptest.NewServer(handler)
	defer server.Close()

	resp, err := http.Get(server.URL + "/ace/getstream?id=abc123")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading response failed: %v", err)
	}

	if len(body) == 0 {
		t.Fatal("expected data from stream, got empty response")
	}
}
