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

type mockActiveStreamChecker struct {
	isStreamActiveFunc func(infoHash string) bool
}

func (m *mockActiveStreamChecker) IsStreamActive(infoHash string) bool {
	if m.isStreamActiveFunc != nil {
		return m.isStreamActiveFunc(infoHash)
	}
	return false
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func newTestProbeService(probeRepo driven.ProbeRepository, streamRepo driven.StreamRepository, engine driven.AceStreamEngine) *ProbeService {
	return NewProbeService(probeRepo, streamRepo, engine, newTestLogger(), 30*time.Second, 24*time.Hour, nil, 0, 0)
}

func TestProbeService_ProbeAllStreams(t *testing.T) {
	t.Run("probes all streams sequentially", func(t *testing.T) {
		s1, _ := stream.NewStream("hash1", "Channel1", "")
		s2, _ := stream.NewStream("hash2", "Channel2", "")

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

		svc := newTestProbeService(probeRepo, streamRepo, engine)

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
		s1, _ := stream.NewStream("hash1", "Channel1", "")
		s2, _ := stream.NewStream("hash2", "Channel2", "")

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

		svc := newTestProbeService(probeRepo, streamRepo, engine)

		err := svc.ProbeAllStreams(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(savedResults) != 2 {
			t.Fatalf("expected 2 saved results, got %d", len(savedResults))
		}
		if savedResults[0].Available() {
			t.Error("hash1 probe should not be available")
		}
		if !savedResults[1].Available() {
			t.Error("hash2 probe should be available")
		}
	})

	t.Run("stops on context cancellation", func(t *testing.T) {
		s1, _ := stream.NewStream("hash1", "Channel1", "")
		s2, _ := stream.NewStream("hash2", "Channel2", "")

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
		svc := newTestProbeService(probeRepo, streamRepo, engine)

		err := svc.ProbeAllStreams(ctx)
		if err == nil {
			t.Fatal("expected error on cancelled context")
		}
	})
}

func TestProbeService_ZombieDetection(t *testing.T) {
	t.Run("marks stream with 0 peers and 0 speed as unavailable", func(t *testing.T) {
		var savedResult probe.Result

		engine := &mockAceStreamEngine{
			startStreamFunc: func(ctx context.Context, infoHash, pid string) (string, error) {
				return "http://localhost/stream", nil
			},
			getStatsFunc: func(ctx context.Context, pid string) (driven.StreamStats, error) {
				return driven.StreamStats{Peers: 0, SpeedDown: 0, Status: "prebuf"}, nil
			},
		}

		probeRepo := &mockProbeRepository{
			saveFunc: func(ctx context.Context, r probe.Result) error {
				savedResult = r
				return nil
			},
		}

		svc := newTestProbeService(probeRepo, &mockStreamRepository{}, engine)

		result, err := svc.ProbeStream(context.Background(), "zombiehash")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Available() {
			t.Error("zombie stream should not be available")
		}
		if savedResult.ErrorMessage() == "" {
			t.Error("expected error message for zombie stream")
		}
	})

	t.Run("stream with peers but 0 speed is still available", func(t *testing.T) {
		engine := &mockAceStreamEngine{
			startStreamFunc: func(ctx context.Context, infoHash, pid string) (string, error) {
				return "http://localhost/stream", nil
			},
			getStatsFunc: func(ctx context.Context, pid string) (driven.StreamStats, error) {
				return driven.StreamStats{Peers: 5, SpeedDown: 0, Status: "dl"}, nil
			},
		}

		probeRepo := &mockProbeRepository{}
		svc := newTestProbeService(probeRepo, &mockStreamRepository{}, engine)

		result, err := svc.ProbeStream(context.Background(), "somehash")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Available() {
			t.Error("stream with peers should be available")
		}
	})

	t.Run("stream with 0 peers but has speed is still available", func(t *testing.T) {
		engine := &mockAceStreamEngine{
			startStreamFunc: func(ctx context.Context, infoHash, pid string) (string, error) {
				return "http://localhost/stream", nil
			},
			getStatsFunc: func(ctx context.Context, pid string) (driven.StreamStats, error) {
				return driven.StreamStats{Peers: 0, SpeedDown: 100000, Status: "dl"}, nil
			},
		}

		probeRepo := &mockProbeRepository{}
		svc := newTestProbeService(probeRepo, &mockStreamRepository{}, engine)

		result, err := svc.ProbeStream(context.Background(), "somehash")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Available() {
			t.Error("stream with download speed should be available")
		}
	})
}

func TestProbeService_SkipsActiveStreams(t *testing.T) {
	s1, _ := stream.NewStream("active-hash", "Channel1", "")
	s2, _ := stream.NewStream("inactive-hash", "Channel2", "")

	var probedHashes []string

	streamRepo := &mockStreamRepository{
		findAllFunc: func(ctx context.Context) ([]stream.Stream, error) {
			return []stream.Stream{s1, s2}, nil
		},
	}

	engine := &mockAceStreamEngine{
		startStreamFunc: func(ctx context.Context, infoHash, pid string) (string, error) {
			probedHashes = append(probedHashes, infoHash)
			return "http://localhost/stream", nil
		},
		getStatsFunc: func(ctx context.Context, pid string) (driven.StreamStats, error) {
			return driven.StreamStats{Peers: 10, SpeedDown: 100000, Status: "dl"}, nil
		},
	}

	checker := &mockActiveStreamChecker{
		isStreamActiveFunc: func(infoHash string) bool {
			return infoHash == "active-hash"
		},
	}

	svc := NewProbeService(&mockProbeRepository{}, streamRepo, engine, newTestLogger(), 30*time.Second, 24*time.Hour, checker, 0, 0)

	err := svc.ProbeAllStreams(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(probedHashes) != 1 {
		t.Fatalf("expected 1 probe, got %d: %v", len(probedHashes), probedHashes)
	}
	if probedHashes[0] != "inactive-hash" {
		t.Errorf("expected inactive-hash to be probed, got %q", probedHashes[0])
	}
}

func TestProbeService_CircuitBreaker(t *testing.T) {
	t.Run("trips after consecutive engine failures", func(t *testing.T) {
		streams := make([]stream.Stream, 10)
		for i := range streams {
			s, _ := stream.NewStream("hash"+string(rune('a'+i)), "Channel", "")
			streams[i] = s
		}

		var probeCount int
		streamRepo := &mockStreamRepository{
			findAllFunc: func(ctx context.Context) ([]stream.Stream, error) {
				return streams, nil
			},
		}

		engine := &mockAceStreamEngine{
			startStreamFunc: func(ctx context.Context, infoHash, pid string) (string, error) {
				probeCount++
				return "", errors.New("engine down")
			},
		}

		svc := NewProbeService(&mockProbeRepository{}, streamRepo, engine, newTestLogger(), 30*time.Second, 24*time.Hour, nil, 0, 3)

		_ = svc.ProbeAllStreams(context.Background())

		if probeCount != 3 {
			t.Errorf("expected circuit breaker to trip after 3 probes, got %d", probeCount)
		}
	})

	t.Run("resets counter on successful probe", func(t *testing.T) {
		streams := make([]stream.Stream, 7)
		for i := range streams {
			s, _ := stream.NewStream("hash"+string(rune('a'+i)), "Channel", "")
			streams[i] = s
		}

		var callIndex int
		streamRepo := &mockStreamRepository{
			findAllFunc: func(ctx context.Context) ([]stream.Stream, error) {
				return streams, nil
			},
		}

		engine := &mockAceStreamEngine{
			startStreamFunc: func(ctx context.Context, infoHash, pid string) (string, error) {
				callIndex++
				// Fail for 1st, 2nd; succeed 3rd; fail 4th, 5th; succeed 6th; fail 7th
				if callIndex == 3 || callIndex == 6 {
					return "http://localhost/stream", nil
				}
				return "", errors.New("engine down")
			},
			getStatsFunc: func(ctx context.Context, pid string) (driven.StreamStats, error) {
				return driven.StreamStats{Peers: 10, SpeedDown: 100000, Status: "dl"}, nil
			},
		}

		svc := NewProbeService(&mockProbeRepository{}, streamRepo, engine, newTestLogger(), 30*time.Second, 24*time.Hour, nil, 0, 3)

		_ = svc.ProbeAllStreams(context.Background())

		// All 7 should be attempted because successes reset the counter
		if callIndex != 7 {
			t.Errorf("expected all 7 streams probed (counter resets on success), got %d", callIndex)
		}
	})
}

func TestProbeService_Throttle(t *testing.T) {
	s1, _ := stream.NewStream("hash1", "Channel1", "")
	s2, _ := stream.NewStream("hash2", "Channel2", "")
	s3, _ := stream.NewStream("hash3", "Channel3", "")

	var timestamps []time.Time

	streamRepo := &mockStreamRepository{
		findAllFunc: func(ctx context.Context) ([]stream.Stream, error) {
			return []stream.Stream{s1, s2, s3}, nil
		},
	}

	engine := &mockAceStreamEngine{
		startStreamFunc: func(ctx context.Context, infoHash, pid string) (string, error) {
			timestamps = append(timestamps, time.Now())
			return "http://localhost/stream", nil
		},
		getStatsFunc: func(ctx context.Context, pid string) (driven.StreamStats, error) {
			return driven.StreamStats{Peers: 10, SpeedDown: 100000, Status: "dl"}, nil
		},
	}

	svc := NewProbeService(&mockProbeRepository{}, streamRepo, engine, newTestLogger(), 30*time.Second, 24*time.Hour, nil, 50*time.Millisecond, 0)

	err := svc.ProbeAllStreams(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(timestamps) != 3 {
		t.Fatalf("expected 3 probes, got %d", len(timestamps))
	}

	for i := 1; i < len(timestamps); i++ {
		gap := timestamps[i].Sub(timestamps[i-1])
		if gap < 40*time.Millisecond {
			t.Errorf("gap between probe %d and %d was %v, expected >= 50ms", i-1, i, gap)
		}
	}
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

	svc := newTestProbeService(probeRepo, &mockStreamRepository{}, &mockAceStreamEngine{})

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

	svc := newTestProbeService(probeRepo, &mockStreamRepository{}, &mockAceStreamEngine{})

	_, err := svc.GetMetrics(context.Background(), "abc123")
	if !errors.Is(err, probe.ErrNoProbeData) {
		t.Errorf("expected ErrNoProbeData, got %v", err)
	}
}

func TestProbeService_GetQualityScores(t *testing.T) {
	now := time.Now()

	s1, _ := stream.NewStream("hash1", "Channel1", "")
	s2, _ := stream.NewStream("hash2", "Channel1", "")

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

	svc := newTestProbeService(probeRepo, streamRepo, &mockAceStreamEngine{})

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

	svc := newTestProbeService(&mockProbeRepository{}, streamRepo, &mockAceStreamEngine{})

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
	svc := NewProbeService(probeRepo, &mockStreamRepository{}, &mockAceStreamEngine{}, newTestLogger(), 30*time.Second, window, nil, 0, 0)

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
