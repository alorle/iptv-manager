package application

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/alorle/iptv-manager/internal/stream"
)

func TestPlaylistService_GenerateM3U(t *testing.T) {
	t.Run("generates M3U playlist with streams successfully", func(t *testing.T) {
		st1, _ := stream.NewStream("abc123", "Channel1")
		st2, _ := stream.NewStream("def456", "Channel2")
		expectedStreams := []stream.Stream{st1, st2}

		streamRepo := &mockStreamRepository{
			findAllFunc: func(ctx context.Context) ([]stream.Stream, error) {
				return expectedStreams, nil
			},
		}
		service := NewPlaylistService(streamRepo)

		m3u, err := service.GenerateM3U(context.Background(), "localhost:8080")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Check header
		if !strings.HasPrefix(m3u, "#EXTM3U\n") {
			t.Error("M3U playlist should start with #EXTM3U header")
		}

		// Check first stream entry
		if !strings.Contains(m3u, `#EXTINF:-1 tvg-id="Channel1",Channel1 - abc123`) {
			t.Error("M3U playlist should contain first stream metadata")
		}
		if !strings.Contains(m3u, "http://localhost:8080/ace/getstream?id=abc123") {
			t.Error("M3U playlist should contain first stream URL")
		}

		// Check second stream entry
		if !strings.Contains(m3u, `#EXTINF:-1 tvg-id="Channel2",Channel2 - def456`) {
			t.Error("M3U playlist should contain second stream metadata")
		}
		if !strings.Contains(m3u, "http://localhost:8080/ace/getstream?id=def456") {
			t.Error("M3U playlist should contain second stream URL")
		}
	})

	t.Run("generates empty M3U playlist when no streams exist", func(t *testing.T) {
		streamRepo := &mockStreamRepository{
			findAllFunc: func(ctx context.Context) ([]stream.Stream, error) {
				return []stream.Stream{}, nil
			},
		}
		service := NewPlaylistService(streamRepo)

		m3u, err := service.GenerateM3U(context.Background(), "localhost:8080")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Should only contain header
		if m3u != "#EXTM3U\n" {
			t.Errorf("expected only #EXTM3U header, got %q", m3u)
		}
	})

	t.Run("returns error when stream repository fails", func(t *testing.T) {
		expectedError := errors.New("repository error")
		streamRepo := &mockStreamRepository{
			findAllFunc: func(ctx context.Context) ([]stream.Stream, error) {
				return nil, expectedError
			},
		}
		service := NewPlaylistService(streamRepo)

		_, err := service.GenerateM3U(context.Background(), "localhost:8080")
		if !errors.Is(err, expectedError) {
			t.Errorf("expected repository error, got %v", err)
		}
	})

	t.Run("uses correct host in stream URLs", func(t *testing.T) {
		st1, _ := stream.NewStream("xyz789", "TestChannel")
		expectedStreams := []stream.Stream{st1}

		streamRepo := &mockStreamRepository{
			findAllFunc: func(ctx context.Context) ([]stream.Stream, error) {
				return expectedStreams, nil
			},
		}
		service := NewPlaylistService(streamRepo)

		m3u, err := service.GenerateM3U(context.Background(), "example.com:9000")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Check that the custom host is used
		if !strings.Contains(m3u, "http://example.com:9000/ace/getstream?id=xyz789") {
			t.Error("M3U playlist should use the provided host in stream URLs")
		}
	})
}
