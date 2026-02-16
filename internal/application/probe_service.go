package application

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/alorle/iptv-manager/internal/port/driven"
	"github.com/alorle/iptv-manager/internal/probe"
)

// StreamQuality pairs a stream's infohash with its quality score and metrics.
type StreamQuality struct {
	InfoHash string
	Score    float64
	Metrics  probe.Metrics
}

// ProbeService orchestrates stream health probing and quality metrics computation.
type ProbeService struct {
	probeRepo    driven.ProbeRepository
	streamRepo   driven.StreamRepository
	engine       driven.AceStreamEngine
	logger       *slog.Logger
	probeTimeout time.Duration
	window       time.Duration
}

// NewProbeService creates a new ProbeService.
func NewProbeService(
	probeRepo driven.ProbeRepository,
	streamRepo driven.StreamRepository,
	engine driven.AceStreamEngine,
	logger *slog.Logger,
	probeTimeout time.Duration,
	window time.Duration,
) *ProbeService {
	return &ProbeService{
		probeRepo:    probeRepo,
		streamRepo:   streamRepo,
		engine:       engine,
		logger:       logger,
		probeTimeout: probeTimeout,
		window:       window,
	}
}

// ProbeAllStreams runs a health-check probe on every known stream sequentially.
// It continues probing remaining streams even if individual probes fail.
func (s *ProbeService) ProbeAllStreams(ctx context.Context) error {
	streams, err := s.streamRepo.FindAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch streams: %w", err)
	}

	s.logger.Info("starting probe cycle", "stream_count", len(streams))

	var probed, failed int
	for _, st := range streams {
		if ctx.Err() != nil {
			s.logger.Info("probe cycle interrupted", "probed", probed, "failed", failed)
			return ctx.Err()
		}

		_, err := s.probeStream(ctx, st.InfoHash())
		if err != nil {
			failed++
			s.logger.Warn("probe failed",
				"infohash", st.InfoHash(),
				"channel", st.ChannelName(),
				"error", err,
			)
		} else {
			probed++
		}
	}

	s.logger.Info("probe cycle completed", "probed", probed, "failed", failed)

	if err := s.Cleanup(ctx); err != nil {
		s.logger.Error("probe cleanup failed", "error", err)
	}

	return nil
}

// probeStream executes a single health-check probe for the given stream.
func (s *ProbeService) probeStream(ctx context.Context, infoHash string) (probe.Result, error) {
	pid := fmt.Sprintf("probe-%d", time.Now().UnixNano())

	probeCtx, cancel := context.WithTimeout(ctx, s.probeTimeout)
	defer cancel()

	startTime := time.Now()
	streamURL, err := s.engine.StartStream(probeCtx, infoHash, pid)
	if err != nil {
		// Stream is unavailable — record a failed probe
		result, resultErr := probe.NewResult(
			infoHash, time.Now(), false, 0, 0, 0, "", err.Error(),
		)
		if resultErr != nil {
			return probe.Result{}, fmt.Errorf("failed to create probe result: %w", resultErr)
		}
		if saveErr := s.probeRepo.Save(ctx, result); saveErr != nil {
			return probe.Result{}, fmt.Errorf("failed to save probe result: %w", saveErr)
		}
		return result, nil
	}
	_ = streamURL

	startupLatency := time.Since(startTime)

	// Get stats from the engine
	stats, statsErr := s.engine.GetStats(probeCtx, pid)

	// Always stop the stream, regardless of stats outcome
	if stopErr := s.engine.StopStream(ctx, pid); stopErr != nil {
		s.logger.Warn("failed to stop probe stream",
			"infohash", infoHash,
			"pid", pid,
			"error", stopErr,
		)
	}

	if statsErr != nil {
		// Stream started but stats failed — record partial probe
		result, resultErr := probe.NewResult(
			infoHash, time.Now(), true, startupLatency, 0, 0, "",
			fmt.Sprintf("stats error: %s", statsErr),
		)
		if resultErr != nil {
			return probe.Result{}, fmt.Errorf("failed to create probe result: %w", resultErr)
		}
		if saveErr := s.probeRepo.Save(ctx, result); saveErr != nil {
			return probe.Result{}, fmt.Errorf("failed to save probe result: %w", saveErr)
		}
		return result, nil
	}

	result, resultErr := probe.NewResult(
		infoHash, time.Now(), true, startupLatency,
		stats.Peers, stats.SpeedDown, stats.Status, "",
	)
	if resultErr != nil {
		return probe.Result{}, fmt.Errorf("failed to create probe result: %w", resultErr)
	}

	if saveErr := s.probeRepo.Save(ctx, result); saveErr != nil {
		return probe.Result{}, fmt.Errorf("failed to save probe result: %w", saveErr)
	}

	s.logger.Debug("probe completed",
		"infohash", infoHash,
		"available", true,
		"peers", stats.Peers,
		"speed_down", stats.SpeedDown,
		"startup_latency", startupLatency,
	)

	return result, nil
}

// GetMetrics computes aggregated metrics for a stream within the rolling window.
func (s *ProbeService) GetMetrics(ctx context.Context, infoHash string) (probe.Metrics, error) {
	since := time.Now().Add(-s.window)
	results, err := s.probeRepo.FindByInfoHashSince(ctx, infoHash, since)
	if err != nil {
		return probe.Metrics{}, fmt.Errorf("failed to fetch probe results: %w", err)
	}
	return probe.NewMetrics(infoHash, results)
}

// GetQualityScores returns quality scores for all streams of a channel,
// sorted by score descending (best first).
func (s *ProbeService) GetQualityScores(ctx context.Context, channelName string) ([]StreamQuality, error) {
	streams, err := s.streamRepo.FindByChannelName(ctx, channelName)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch streams: %w", err)
	}

	if len(streams) == 0 {
		return []StreamQuality{}, nil
	}

	// Compute metrics for each stream
	type entry struct {
		infoHash string
		metrics  probe.Metrics
	}
	var entries []entry

	for _, st := range streams {
		m, err := s.GetMetrics(ctx, st.InfoHash())
		if err != nil {
			// No probe data for this stream yet — skip it
			continue
		}
		entries = append(entries, entry{infoHash: st.InfoHash(), metrics: m})
	}

	if len(entries) == 0 {
		return []StreamQuality{}, nil
	}

	// Find normalization ceilings
	var maxSpeed, maxPeers float64
	for _, e := range entries {
		if e.metrics.AvgDownloadSpeed() > maxSpeed {
			maxSpeed = e.metrics.AvgDownloadSpeed()
		}
		if e.metrics.AvgPeerCount() > maxPeers {
			maxPeers = e.metrics.AvgPeerCount()
		}
	}

	// Compute scores
	qualities := make([]StreamQuality, 0, len(entries))
	for _, e := range entries {
		score := probe.ComputeQualityScore(e.metrics, maxSpeed, maxPeers)
		qualities = append(qualities, StreamQuality{
			InfoHash: e.infoHash,
			Score:    score,
			Metrics:  e.metrics,
		})
	}

	// Sort by score descending
	slices.SortFunc(qualities, func(a, b StreamQuality) int {
		return cmp.Compare(b.Score, a.Score)
	})

	return qualities, nil
}

// GetProbeHistory returns raw probe results for a stream within the rolling window.
func (s *ProbeService) GetProbeHistory(ctx context.Context, infoHash string) ([]probe.Result, error) {
	since := time.Now().Add(-s.window)
	return s.probeRepo.FindByInfoHashSince(ctx, infoHash, since)
}

// Cleanup removes probe data older than twice the rolling window.
func (s *ProbeService) Cleanup(ctx context.Context) error {
	cutoff := time.Now().Add(-s.window * 2)
	return s.probeRepo.DeleteBefore(ctx, cutoff)
}
