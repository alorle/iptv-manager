package probe

import "math"

// Metrics holds aggregated quality metrics derived from probe history.
type Metrics struct {
	infoHash          string
	totalProbes       int
	successfulProbes  int
	uptimeRatio       float64
	avgPeerCount      float64
	avgDownloadSpeed  float64
	speedStdDev       float64
	failureRate       float64
	avgStartupLatency float64 // milliseconds
}

// NewMetrics computes aggregated metrics from a slice of probe results.
// Returns ErrNoProbeData if results is empty.
func NewMetrics(infoHash string, results []Result) (Metrics, error) {
	if len(results) == 0 {
		return Metrics{}, ErrNoProbeData
	}

	total := len(results)
	successful := 0
	var totalPeers int
	var totalSpeed int64
	var totalLatency float64
	var speeds []float64

	for _, r := range results {
		if r.Available() {
			successful++
			totalPeers += r.PeerCount()
			totalSpeed += r.DownloadSpeed()
			totalLatency += float64(r.StartupLatency().Milliseconds())
			speeds = append(speeds, float64(r.DownloadSpeed()))
		}
	}

	uptimeRatio := float64(successful) / float64(total)
	failureRate := 1.0 - uptimeRatio

	var avgPeers, avgSpeed, avgLatency, stdDev float64
	if successful > 0 {
		avgPeers = float64(totalPeers) / float64(successful)
		avgSpeed = float64(totalSpeed) / float64(successful)
		avgLatency = totalLatency / float64(successful)

		var sumSquaredDiff float64
		for _, s := range speeds {
			diff := s - avgSpeed
			sumSquaredDiff += diff * diff
		}
		stdDev = math.Sqrt(sumSquaredDiff / float64(len(speeds)))
	}

	return Metrics{
		infoHash:          infoHash,
		totalProbes:       total,
		successfulProbes:  successful,
		uptimeRatio:       uptimeRatio,
		avgPeerCount:      avgPeers,
		avgDownloadSpeed:  avgSpeed,
		speedStdDev:       stdDev,
		failureRate:       failureRate,
		avgStartupLatency: avgLatency,
	}, nil
}

func (m Metrics) InfoHash() string           { return m.infoHash }
func (m Metrics) TotalProbes() int           { return m.totalProbes }
func (m Metrics) SuccessfulProbes() int      { return m.successfulProbes }
func (m Metrics) UptimeRatio() float64       { return m.uptimeRatio }
func (m Metrics) AvgPeerCount() float64      { return m.avgPeerCount }
func (m Metrics) AvgDownloadSpeed() float64  { return m.avgDownloadSpeed }
func (m Metrics) SpeedStdDev() float64       { return m.speedStdDev }
func (m Metrics) FailureRate() float64       { return m.failureRate }
func (m Metrics) AvgStartupLatency() float64 { return m.avgStartupLatency }
