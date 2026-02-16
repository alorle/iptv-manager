package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/alorle/iptv-manager/internal/probe"
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
		service := NewPlaylistService(streamRepo, &mockProbeRepository{}, 24*time.Hour)

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
		service := NewPlaylistService(streamRepo, &mockProbeRepository{}, 24*time.Hour)

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
		service := NewPlaylistService(streamRepo, &mockProbeRepository{}, 24*time.Hour)

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
		service := NewPlaylistService(streamRepo, &mockProbeRepository{}, 24*time.Hour)

		m3u, err := service.GenerateM3U(context.Background(), "example.com:9000")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Check that the custom host is used
		if !strings.Contains(m3u, "http://example.com:9000/ace/getstream?id=xyz789") {
			t.Error("M3U playlist should use the provided host in stream URLs")
		}
	})

	t.Run("sorts streams by quality score within channel group", func(t *testing.T) {
		now := time.Now()
		good, _ := stream.NewStream("hash_good", "SameChannel")
		poor, _ := stream.NewStream("hash_poor", "SameChannel")
		streamRepo := &mockStreamRepository{
			findAllFunc: func(ctx context.Context) ([]stream.Stream, error) {
				return []stream.Stream{poor, good}, nil
			},
		}
		probeRepo := &mockProbeRepository{
			findByInfoHashSinceFunc: func(ctx context.Context, infoHash string, since time.Time) ([]probe.Result, error) {
				if infoHash == "hash_good" {
					return []probe.Result{
						probe.ReconstructResult(infoHash, now, true, time.Second, 20, 200000, "dl", ""),
						probe.ReconstructResult(infoHash, now.Add(-30*time.Minute), true, time.Second, 20, 200000, "dl", ""),
					}, nil
				}
				return []probe.Result{
					probe.ReconstructResult(infoHash, now, true, 5*time.Second, 3, 30000, "dl", ""),
					probe.ReconstructResult(infoHash, now.Add(-30*time.Minute), false, 0, 0, 0, "", "timeout"),
				}, nil
			},
		}
		service := NewPlaylistService(streamRepo, probeRepo, 24*time.Hour)

		m3u, err := service.GenerateM3U(context.Background(), "localhost:8080")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		goodIdx := strings.Index(m3u, "hash_good")
		poorIdx := strings.Index(m3u, "hash_poor")
		if goodIdx < 0 || poorIdx < 0 {
			t.Fatal("both streams should appear in the playlist")
		}
		if goodIdx >= poorIdx {
			t.Error("higher quality stream (hash_good) should appear before lower quality (hash_poor)")
		}
	})

	t.Run("streams without probe data sort after scored streams", func(t *testing.T) {
		now := time.Now()
		scored, _ := stream.NewStream("hash_scored", "Chan")
		unscored, _ := stream.NewStream("hash_unscored", "Chan")
		streamRepo := &mockStreamRepository{
			findAllFunc: func(ctx context.Context) ([]stream.Stream, error) {
				return []stream.Stream{unscored, scored}, nil
			},
		}
		probeRepo := &mockProbeRepository{
			findByInfoHashSinceFunc: func(ctx context.Context, infoHash string, since time.Time) ([]probe.Result, error) {
				if infoHash == "hash_scored" {
					return []probe.Result{
						probe.ReconstructResult(infoHash, now, true, time.Second, 10, 100000, "dl", ""),
					}, nil
				}
				return []probe.Result{}, nil
			},
		}
		service := NewPlaylistService(streamRepo, probeRepo, 24*time.Hour)

		m3u, err := service.GenerateM3U(context.Background(), "localhost:8080")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		scoredIdx := strings.Index(m3u, "hash_scored")
		unscoredIdx := strings.Index(m3u, "hash_unscored")
		if scoredIdx < 0 || unscoredIdx < 0 {
			t.Fatal("both streams should appear in the playlist")
		}
		if scoredIdx >= unscoredIdx {
			t.Error("scored stream should appear before unscored stream")
		}
	})

	t.Run("degrades to infohash sort when no probe data exists", func(t *testing.T) {
		s1, _ := stream.NewStream("zzz999", "Chan")
		s2, _ := stream.NewStream("aaa111", "Chan")
		streamRepo := &mockStreamRepository{
			findAllFunc: func(ctx context.Context) ([]stream.Stream, error) {
				return []stream.Stream{s1, s2}, nil
			},
		}
		service := NewPlaylistService(streamRepo, &mockProbeRepository{}, 24*time.Hour)

		m3u, err := service.GenerateM3U(context.Background(), "localhost:8080")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		idx1 := strings.Index(m3u, "aaa111")
		idx2 := strings.Index(m3u, "zzz999")
		if idx1 >= idx2 {
			t.Error("when no probe data exists, streams should sort by infohash ascending")
		}
	})

	t.Run("probeRepo error degrades gracefully", func(t *testing.T) {
		s1, _ := stream.NewStream("abc123", "Channel1")
		streamRepo := &mockStreamRepository{
			findAllFunc: func(ctx context.Context) ([]stream.Stream, error) {
				return []stream.Stream{s1}, nil
			},
		}
		probeRepo := &mockProbeRepository{
			findByInfoHashSinceFunc: func(ctx context.Context, infoHash string, since time.Time) ([]probe.Result, error) {
				return nil, errors.New("db error")
			},
		}
		service := NewPlaylistService(streamRepo, probeRepo, 24*time.Hour)

		m3u, err := service.GenerateM3U(context.Background(), "localhost:8080")
		if err != nil {
			t.Fatalf("expected no error despite probeRepo failure, got %v", err)
		}

		if !strings.Contains(m3u, "abc123") {
			t.Error("stream should still appear in playlist despite probe error")
		}
	})
}
