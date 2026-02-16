package probe

// ComputeQualityScore calculates a composite quality score from metrics.
// Returns a value in [0.0, 1.0] where 1.0 is the best possible quality.
//
// Formula:
//
//	score = (uptime_ratio * 0.50) +
//	        (normalized_speed * 0.20) +
//	        (normalized_peers * 0.15) +
//	        (speed_stability * 0.10) +
//	        (startup_latency_score * 0.05)
//
// maxSpeed and maxPeers are normalization ceilings provided by the caller
// (typically the maximum observed across all streams being compared).
func ComputeQualityScore(m Metrics, maxSpeed float64, maxPeers float64) float64 {
	uptimeComponent := m.UptimeRatio() * 0.50

	var normalizedSpeed float64
	if maxSpeed > 0 {
		normalizedSpeed = clamp(m.AvgDownloadSpeed()/maxSpeed, 0, 1)
	}
	speedComponent := normalizedSpeed * 0.20

	var normalizedPeers float64
	if maxPeers > 0 {
		normalizedPeers = clamp(m.AvgPeerCount()/maxPeers, 0, 1)
	}
	peersComponent := normalizedPeers * 0.15

	var speedStability float64
	if m.AvgDownloadSpeed() > 0 {
		speedStability = clamp(1.0-(m.SpeedStdDev()/m.AvgDownloadSpeed()), 0, 1)
	}
	stabilityComponent := speedStability * 0.10

	const maxLatencyMs = 10000.0
	var latencyScore float64
	if m.AvgStartupLatency() > 0 {
		latencyScore = clamp(1.0-(m.AvgStartupLatency()/maxLatencyMs), 0, 1)
	} else if m.SuccessfulProbes() > 0 {
		latencyScore = 1.0
	}
	latencyComponent := latencyScore * 0.05

	return uptimeComponent + speedComponent + peersComponent +
		stabilityComponent + latencyComponent
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
