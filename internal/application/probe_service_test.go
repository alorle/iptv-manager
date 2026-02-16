package application

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/alorle/iptv-manager/internal/port/driven"
	"github.com/alorle/iptv-manager/internal/probe"
	"github.com/alorle/iptv-manager/internal/stream"
)

// mockProbeRepository implements driven.ProbeRepository for testing.
type mockProbeRepository struct {
	saveFunc                func(ctx context.Context, r probe.Result) error
	findByInfoHashFunc      func(ctx context.Context, infoHash string) ([]probe.Result, error)
	findByInfoHashSinceFunc func(ctx context.Context, infoHash string, since time.Time) ([]probe.Result, error)
	deleteBeforeFunc        func(ctx context.Context, before time.Time) error
}

func (m *mockProbeRepository) Save(ctx context.Context, r probe.Result) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, r)
	}
	return nil
}

func (m *mockProbeRepository) FindByInfoHash(ctx context.Context, infoHash string) ([]probe.Result, error) {
	if m.findByInfoHashFunc != nil {
		return m.findByInfoHashFunc(ctx, infoHash)
	}
	return []probe.Result{}, nil
}

func (m *mockProbeRepository) FindByInfoHashSince(ctx context.Context, infoHash string, since time.Time) ([]probe.Result, error) {
	if m.findByInfoHashSinceFunc != nil {
		return m.findByInfoHashSinceFunc(ctx, infoHash, since)
	}
	return []probe.Result{}, nil
}

func (m *mockProbeRepository) DeleteBefore(ctx context.Context, before time.Time) error {
	if m.deleteBeforeFunc != nil {
		return m.deleteBeforeFunc(ctx, before)
	}
	return nil
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestProbeService_ProbeAllStreams(t *testing.T) {
	t.Run("probes all streams sequentially", func(t *testing.T) {
		s1, _ := stream.NewStream("hash1", "Channel1")
		s2, _ := stream.NewStream("hash2", "Channel2")

		var probeOrder []string
		var savedResults []probe.Result

		streamRepo := &mockStreamRepository{
			findAllFunc: func(ctx context.Context) ([]stream.Stream, error) {
				return []stream.Stream{s1, s2}, nil
			},
		}

		engine := &mockAceStreamEngine{
			startStreamFunc: func(ctx context.Context, infoHash, pid string) (string, error) {
				probeOrder = append(probeOrder, infoHash)
				return "http://localhost/stream", nil
			},
			getStatsFunc: func(ctx context.Context, pid string) (driven.StreamStats, error) {
				return driven.StreamStats{Peers: 10, SpeedDown: 100000, Status: "dl"}, nil
			},
		}

		probeRepo := &mockProbeRepository{
			saveFunc: func(ctx context.Context, r probe.Result) error {
				savedResults = append(savedResults, r)
				return nil
			},
		}

		svc := NewProbeService(probeRepo, streamRepo, engine, newTestLogger(), 30*time.Second, 24*time.Hour)

		err := svc.ProbeAllStreams(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(probeOrder) != 2 {
			t.Fatalf("expected 2 probes, got %d", len(probeOrder))
		}
		if probeOrder[0] != "hash1" || probeOrder[1] != "hash2" {
			t.Errorf("probe order = %v, want [hash1, hash2]", probeOrder)
		}
		if len(savedResults) != 2 {
			t.Fatalf("expected 2 saved results, got %d", len(savedResults))
		}
		for _, r := range savedResults {
			if !r.Available() {
				t.Errorf("expected all probes to be available")
			}
		}
	})

	t.Run("continues on engine failure", func(t *testing.T) {
		s1, _ := stream.NewStream("hash1", "Channel1")
		s2, _ := stream.NewStream("hash2", "Channel2")

		var savedResults []probe.Result

		streamRepo := &mockStreamRepository{
			findAllFunc: func(ctx context.Context) ([]stream.Stream, error) {
				return []stream.Stream{s1, s2}, nil
			},
		}

		engine := &mockAceStreamEngine{
			startStreamFunc: func(ctx context.Context, infoHash, pid string) (string, error) {
				if infoHash == "hash1" {
					return "", errors.New("engine unavailable")
				}
				return "http://localhost/stream", nil
			},
			getStatsFunc: func(ctx context.Context, pid string) (driven.StreamStats, error) {
				return driven.StreamStats{Peers: 5, SpeedDown: 50000, Status: "dl"}, nil
			},
		}

		probeRepo := &mockProbeRepository{
			saveFunc: func(ctx context.Context, r probe.Result) error {
				savedResults = append(savedResults, r)
				return nil
			},
		}

		svc := NewProbeService(probeRepo, streamRepo, engine, newTestLogger(), 30*time.Second, 24*time.Hour)

		err := svc.ProbeAllStreams(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(savedResults) != 2 {
			t.Fatalf("expected 2 saved results, got %d", len(savedResults))
		}
		// hash1 should be unavailable, hash2 should be available
		if savedResults[0].Available() {
			t.Error("hash1 probe should not be available")
		}
		if !savedResults[1].Available() {
			t.Error("hash2 probe should be available")
		}
	})

	t.Run("stops on context cancellation", func(t *testing.T) {
		s1, _ := stream.NewStream("hash1", "Channel1")
		s2, _ := stream.NewStream("hash2", "Channel2")

		streamRepo := &mockStreamRepository{
			findAllFunc: func(ctx context.Context) ([]stream.Stream, error) {
				return []stream.Stream{s1, s2}, nil
			},
		}

		ctx, cancel := context.WithCancel(context.Background())

		probeCount := 0
		engine := &mockAceStreamEngine{
			startStreamFunc: func(ctx context.Context, infoHash, pid string) (string, error) {
				probeCount++
				cancel() // Cancel after first probe starts
				return "http://localhost/stream", nil
			},
			getStatsFunc: func(ctx context.Context, pid string) (driven.StreamStats, error) {
				return driven.StreamStats{}, nil
			},
		}

		probeRepo := &mockProbeRepository{}
		svc := NewProbeService(probeRepo, streamRepo, engine, newTestLogger(), 30*time.Second, 24*time.Hour)

		err := svc.ProbeAllStreams(ctx)
		if err == nil {
			t.Fatal("expected error on cancelled context")
		}
	})
}

func TestProbeService_GetMetrics(t *testing.T) {
	now := time.Now()

	probeRepo := &mockProbeRepository{
		findByInfoHashSinceFunc: func(ctx context.Context, infoHash string, since time.Time) ([]probe.Result, error) {
			return []probe.Result{
				probe.ReconstructResult(infoHash, now, true, 2*time.Second, 10, 100000, "dl", ""),
				probe.ReconstructResult(infoHash, now.Add(-30*time.Minute), true, 3*time.Second, 20, 200000, "dl", ""),
			}, nil
		},
	}

	svc := NewProbeService(probeRepo, &mockStreamRepository{}, &mockAceStreamEngine{}, newTestLogger(), 30*time.Second, 24*time.Hour)

	m, err := svc.GetMetrics(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if m.TotalProbes() != 2 {
		t.Errorf("TotalProbes() = %d, want 2", m.TotalProbes())
	}
	if m.UptimeRatio() != 1.0 {
		t.Errorf("UptimeRatio() = %f, want 1.0", m.UptimeRatio())
	}
	if m.AvgPeerCount() != 15.0 {
		t.Errorf("AvgPeerCount() = %f, want 15.0", m.AvgPeerCount())
	}
}

func TestProbeService_GetMetrics_NoData(t *testing.T) {
	probeRepo := &mockProbeRepository{
		findByInfoHashSinceFunc: func(ctx context.Context, infoHash string, since time.Time) ([]probe.Result, error) {
			return []probe.Result{}, nil
		},
	}

	svc := NewProbeService(probeRepo, &mockStreamRepository{}, &mockAceStreamEngine{}, newTestLogger(), 30*time.Second, 24*time.Hour)

	_, err := svc.GetMetrics(context.Background(), "abc123")
	if !errors.Is(err, probe.ErrNoProbeData) {
		t.Errorf("expected ErrNoProbeData, got %v", err)
	}
}

func TestProbeService_GetQualityScores(t *testing.T) {
	now := time.Now()

	s1, _ := stream.NewStream("hash1", "Channel1")
	s2, _ := stream.NewStream("hash2", "Channel1")

	streamRepo := &mockStreamRepository{
		findByChannelNameFunc: func(ctx context.Context, channelName string) ([]stream.Stream, error) {
			return []stream.Stream{s1, s2}, nil
		},
	}

	probeRepo := &mockProbeRepository{
		findByInfoHashSinceFunc: func(ctx context.Context, infoHash string, since time.Time) ([]probe.Result, error) {
			if infoHash == "hash1" {
				// Good stream: always up, fast, many peers
				return []probe.Result{
					probe.ReconstructResult(infoHash, now, true, time.Second, 20, 200000, "dl", ""),
					probe.ReconstructResult(infoHash, now.Add(-30*time.Minute), true, time.Second, 20, 200000, "dl", ""),
				}, nil
			}
			// Poor stream: sometimes down, slow
			return []probe.Result{
				probe.ReconstructResult(infoHash, now, true, 5*time.Second, 3, 30000, "dl", ""),
				probe.ReconstructResult(infoHash, now.Add(-30*time.Minute), false, 0, 0, 0, "", "timeout"),
			}, nil
		},
	}

	svc := NewProbeService(probeRepo, streamRepo, &mockAceStreamEngine{}, newTestLogger(), 30*time.Second, 24*time.Hour)

	scores, err := svc.GetQualityScores(context.Background(), "Channel1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(scores) != 2 {
		t.Fatalf("expected 2 scores, got %d", len(scores))
	}

	// hash1 should score higher than hash2
	if scores[0].InfoHash != "hash1" {
		t.Errorf("expected hash1 first (best), got %q", scores[0].InfoHash)
	}
	if scores[1].InfoHash != "hash2" {
		t.Errorf("expected hash2 second, got %q", scores[1].InfoHash)
	}
	if scores[0].Score <= scores[1].Score {
		t.Errorf("hash1 score (%f) should be > hash2 score (%f)", scores[0].Score, scores[1].Score)
	}
}

func TestProbeService_GetQualityScores_NoStreams(t *testing.T) {
	streamRepo := &mockStreamRepository{
		findByChannelNameFunc: func(ctx context.Context, channelName string) ([]stream.Stream, error) {
			return []stream.Stream{}, nil
		},
	}

	svc := NewProbeService(&mockProbeRepository{}, streamRepo, &mockAceStreamEngine{}, newTestLogger(), 30*time.Second, 24*time.Hour)

	scores, err := svc.GetQualityScores(context.Background(), "NoChannel")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(scores) != 0 {
		t.Errorf("expected empty scores, got %d", len(scores))
	}
}

func TestProbeService_Cleanup(t *testing.T) {
	var deletedBefore time.Time

	probeRepo := &mockProbeRepository{
		deleteBeforeFunc: func(ctx context.Context, before time.Time) error {
			deletedBefore = before
			return nil
		},
	}

	window := 24 * time.Hour
	svc := NewProbeService(probeRepo, &mockStreamRepository{}, &mockAceStreamEngine{}, newTestLogger(), 30*time.Second, window)

	before := time.Now()
	err := svc.Cleanup(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Cutoff should be approximately 2 * window ago
	expectedCutoff := before.Add(-window * 2)
	diff := deletedBefore.Sub(expectedCutoff)
	if diff < -time.Second || diff > time.Second {
		t.Errorf("DeleteBefore cutoff = %v, expected ~%v (diff: %v)", deletedBefore, expectedCutoff, diff)
	}
}
