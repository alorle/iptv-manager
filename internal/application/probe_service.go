package application

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/alorle/iptv-manager/internal/port/driven"
	"github.com/alorle/iptv-manager/internal/probe"
)

var errEngineFailure = errors.New("engine failure during probe")

// StreamQuality pairs a stream's infohash with its quality score and metrics.
type StreamQuality struct {
	InfoHash string
	Score    float64
	Metrics  probe.Metrics
}

// ProbeService orchestrates stream health probing and quality metrics computation.
type ProbeService struct {
	probeRepo              driven.ProbeRepository
	streamRepo             driven.StreamRepository
	engine                 driven.AceStreamEngine
	activeChecker          driven.ActiveStreamChecker
	logger                 *slog.Logger
	probeTimeout           time.Duration
	window                 time.Duration
	probeDelay             time.Duration
	maxConsecutiveFailures int
}

// NewProbeService creates a new ProbeService.
func NewProbeService(
	probeRepo driven.ProbeRepository,
	streamRepo driven.StreamRepository,
	engine driven.AceStreamEngine,
	logger *slog.Logger,
	probeTimeout time.Duration,
	window time.Duration,
	activeChecker driven.ActiveStreamChecker,
	probeDelay time.Duration,
	maxConsecutiveFailures int,
) *ProbeService {
	return &ProbeService{
		probeRepo:              probeRepo,
		streamRepo:             streamRepo,
		engine:                 engine,
		activeChecker:          activeChecker,
		logger:                 logger,
		probeTimeout:           probeTimeout,
		window:                 window,
		probeDelay:             probeDelay,
		maxConsecutiveFailures: maxConsecutiveFailures,
	}
}

// ProbeAllStreams runs a health-check probe on every known stream sequentially.
// It skips streams that are actively being watched, throttles between probes,
// and trips a circuit breaker after consecutive engine failures.
func (s *ProbeService) ProbeAllStreams(ctx context.Context) error {
	streams, err := s.streamRepo.FindAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch streams: %w", err)
	}

	s.logger.Info("starting probe cycle", "stream_count", len(streams), "probe_delay", s.probeDelay)

	var probed, failed, skipped, consecutiveFailures int
	for i, st := range streams {
		if ctx.Err() != nil {
			s.logger.Info("probe cycle interrupted", "probed", probed, "failed", failed, "skipped", skipped)
			return ctx.Err()
		}

		if s.activeChecker != nil && s.activeChecker.IsStreamActive(st.InfoHash()) {
			s.logger.Debug("skipping probe for active stream", "infohash", st.InfoHash(), "channel", st.ChannelName())
			skipped++
			continue
		}

		_, err := s.probeStream(ctx, st.InfoHash())
		if err != nil {
			failed++
			if errors.Is(err, errEngineFailure) {
				consecutiveFailures++
			} else {
				consecutiveFailures = 0
			}
			if s.maxConsecutiveFailures > 0 && consecutiveFailures >= s.maxConsecutiveFailures {
				s.logger.Error("circuit breaker tripped: engine appears unhealthy, aborting probe cycle",
					"consecutive_failures", consecutiveFailures,
					"probed", probed,
					"failed", failed,
					"skipped", skipped,
				)
				break
			}
			s.logger.Warn("probe failed",
				"infohash", st.InfoHash(),
				"channel", st.ChannelName(),
				"error", err,
			)
		} else {
			probed++
			consecutiveFailures = 0
		}

		if s.probeDelay > 0 && i < len(streams)-1 {
			select {
			case <-ctx.Done():
				s.logger.Info("probe cycle interrupted during throttle", "probed", probed, "failed", failed, "skipped", skipped)
				return ctx.Err()
			case <-time.After(s.probeDelay):
			}
		}
	}

	s.logger.Info("probe cycle completed", "probed", probed, "failed", failed, "skipped", skipped)

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
	_, err := s.engine.StartStream(probeCtx, infoHash, pid)
	if err != nil {
		result, resultErr := probe.NewResult(
			infoHash, time.Now(), false, 0, 0, 0, "", err.Error(),
		)
		if resultErr != nil {
			return probe.Result{}, fmt.Errorf("failed to create probe result: %w", resultErr)
		}
		if saveErr := s.probeRepo.Save(ctx, result); saveErr != nil {
			return probe.Result{}, fmt.Errorf("failed to save probe result: %w", saveErr)
		}
		return result, errEngineFailure
	}

	startupLatency := time.Since(startTime)

	stats, statsErr := s.engine.GetStats(probeCtx, pid)

	if stopErr := s.engine.StopStream(ctx, pid); stopErr != nil {
		s.logger.Warn("failed to stop probe stream",
			"infohash", infoHash,
			"pid", pid,
			"error", stopErr,
		)
	}

	if statsErr != nil {
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

	// Zombie detection: engine returned a URL but no peers and no data flowing
	isZombie := stats.Peers == 0 && stats.SpeedDown == 0
	available := !isZombie
	errorMsg := ""
	if isZombie {
		errorMsg = "zombie stream: 0 peers and 0 download speed"
	}

	result, resultErr := probe.NewResult(
		infoHash, time.Now(), available, startupLatency,
		stats.Peers, stats.SpeedDown, stats.Status, errorMsg,
	)
	if resultErr != nil {
		return probe.Result{}, fmt.Errorf("failed to create probe result: %w", resultErr)
	}

	if saveErr := s.probeRepo.Save(ctx, result); saveErr != nil {
		return probe.Result{}, fmt.Errorf("failed to save probe result: %w", saveErr)
	}

	s.logger.Debug("probe completed",
		"infohash", infoHash,
		"available", available,
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

// ProbeStream runs an immediate health-check probe for a single stream and returns the result.
func (s *ProbeService) ProbeStream(ctx context.Context, infoHash string) (probe.Result, error) {
	return s.probeStream(ctx, infoHash)
}

// ChannelHealth holds the health summary for a single channel.
type ChannelHealth struct {
	ChannelName string
	BestScore   float64
	StreamCount int
	InfoHashes  []string
	LastProbe   *probe.Result
}

// GetChannelHealth computes the health summary for a single channel.
func (s *ProbeService) GetChannelHealth(ctx context.Context, channelName string) (ChannelHealth, error) {
	streams, err := s.streamRepo.FindByChannelName(ctx, channelName)
	if err != nil {
		return ChannelHealth{}, fmt.Errorf("failed to fetch streams: %w", err)
	}

	health := ChannelHealth{
		ChannelName: channelName,
		StreamCount: len(streams),
		InfoHashes:  make([]string, len(streams)),
	}
	for i, st := range streams {
		health.InfoHashes[i] = st.InfoHash()
	}

	if len(streams) == 0 {
		return health, nil
	}

	scores, err := s.GetQualityScores(ctx, channelName)
	if err != nil {
		return ChannelHealth{}, fmt.Errorf("failed to get quality scores: %w", err)
	}
	if len(scores) > 0 {
		health.BestScore = scores[0].Score
	}

	since := time.Now().Add(-s.window)
	for _, st := range streams {
		results, err := s.probeRepo.FindByInfoHashSince(ctx, st.InfoHash(), since)
		if err != nil || len(results) == 0 {
			continue
		}
		latest := results[0]
		if health.LastProbe == nil || latest.Timestamp().After(health.LastProbe.Timestamp()) {
			r := latest
			health.LastProbe = &r
		}
	}

	return health, nil
}

// Cleanup removes probe data older than twice the rolling window.
func (s *ProbeService) Cleanup(ctx context.Context) error {
	cutoff := time.Now().Add(-s.window * 2)
	return s.probeRepo.DeleteBefore(ctx, cutoff)
}
