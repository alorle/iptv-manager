package driver

import (
	"errors"
	"net/http"
	"strings"

	"github.com/alorle/iptv-manager/internal/application"
	"github.com/alorle/iptv-manager/internal/probe"
)

// ProbeHTTPHandler handles HTTP requests for probe results and quality metrics.
type ProbeHTTPHandler struct {
	service *application.ProbeService
}

// NewProbeHTTPHandler creates a new HTTP handler for probes.
func NewProbeHTTPHandler(service *application.ProbeService) *ProbeHTTPHandler {
	return &ProbeHTTPHandler{service: service}
}

// probeResultResponse represents a probe result in JSON format.
type probeResultResponse struct {
	InfoHash       string `json:"info_hash"`
	Timestamp      string `json:"timestamp"`
	Available      bool   `json:"available"`
	StartupLatency int64  `json:"startup_latency_ms"`
	PeerCount      int    `json:"peer_count"`
	DownloadSpeed  int64  `json:"download_speed"`
	Status         string `json:"status"`
	ErrorMessage   string `json:"error_message,omitempty"`
}

// metricsResponse represents aggregated metrics in JSON format.
type metricsResponse struct {
	InfoHash          string  `json:"info_hash"`
	TotalProbes       int     `json:"total_probes"`
	SuccessfulProbes  int     `json:"successful_probes"`
	UptimeRatio       float64 `json:"uptime_ratio"`
	AvgPeerCount      float64 `json:"avg_peer_count"`
	AvgDownloadSpeed  float64 `json:"avg_download_speed"`
	SpeedStdDev       float64 `json:"speed_std_dev"`
	FailureRate       float64 `json:"failure_rate"`
	AvgStartupLatency float64 `json:"avg_startup_latency_ms"`
}

// qualityResponse represents a stream's quality score in JSON format.
type qualityResponse struct {
	InfoHash string          `json:"info_hash"`
	Score    float64         `json:"score"`
	Metrics  metricsResponse `json:"metrics"`
}

// ServeHTTP routes the request based on path prefix.
func (h *ProbeHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// POST /probes/run - trigger immediate probe cycle
	if r.Method == http.MethodPost && path == "/probes/run" {
		h.handleRun(w, r)
		return
	}

	// Routes under /probes/{infoHash}
	if strings.HasPrefix(path, "/probes/") {
		remaining := strings.TrimPrefix(path, "/probes/")

		// GET /probes/{infoHash}/metrics
		if r.Method == http.MethodGet && strings.HasSuffix(remaining, "/metrics") {
			infoHash := strings.TrimSuffix(remaining, "/metrics")
			h.handleMetrics(w, r, infoHash)
			return
		}

		// GET /probes/{infoHash} — probe history
		if r.Method == http.MethodGet && remaining != "" {
			h.handleHistory(w, r, remaining)
			return
		}
	}

	// GET /quality/{channelName}
	if r.Method == http.MethodGet && strings.HasPrefix(path, "/quality/") {
		channelName := strings.TrimPrefix(path, "/quality/")
		if channelName != "" {
			h.handleQuality(w, r, channelName)
			return
		}
	}

	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}

// handleHistory handles GET /probes/{infoHash}
func (h *ProbeHTTPHandler) handleHistory(w http.ResponseWriter, r *http.Request, infoHash string) {
	results, err := h.service.GetProbeHistory(r.Context(), infoHash)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	response := make([]probeResultResponse, len(results))
	for i, res := range results {
		response[i] = toProbeResultResponse(res)
	}

	writeJSON(w, http.StatusOK, response)
}

// handleMetrics handles GET /probes/{infoHash}/metrics
func (h *ProbeHTTPHandler) handleMetrics(w http.ResponseWriter, r *http.Request, infoHash string) {
	m, err := h.service.GetMetrics(r.Context(), infoHash)
	if err != nil {
		if errors.Is(err, probe.ErrNoProbeData) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, toMetricsResponse(m))
}

// handleQuality handles GET /quality/{channelName}
func (h *ProbeHTTPHandler) handleQuality(w http.ResponseWriter, r *http.Request, channelName string) {
	scores, err := h.service.GetQualityScores(r.Context(), channelName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	response := make([]qualityResponse, len(scores))
	for i, sq := range scores {
		response[i] = qualityResponse{
			InfoHash: sq.InfoHash,
			Score:    sq.Score,
			Metrics:  toMetricsResponse(sq.Metrics),
		}
	}

	writeJSON(w, http.StatusOK, response)
}

// handleRun handles POST /probes/run
func (h *ProbeHTTPHandler) handleRun(w http.ResponseWriter, r *http.Request) {
	if err := h.service.ProbeAllStreams(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "probe cycle failed: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "completed"})
}

func toProbeResultResponse(r probe.Result) probeResultResponse {
	return probeResultResponse{
		InfoHash:       r.InfoHash(),
		Timestamp:      r.Timestamp().Format("2006-01-02T15:04:05Z07:00"),
		Available:      r.Available(),
		StartupLatency: r.StartupLatency().Milliseconds(),
		PeerCount:      r.PeerCount(),
		DownloadSpeed:  r.DownloadSpeed(),
		Status:         r.Status(),
		ErrorMessage:   r.ErrorMessage(),
	}
}

func toMetricsResponse(m probe.Metrics) metricsResponse {
	return metricsResponse{
		InfoHash:          m.InfoHash(),
		TotalProbes:       m.TotalProbes(),
		SuccessfulProbes:  m.SuccessfulProbes(),
		UptimeRatio:       m.UptimeRatio(),
		AvgPeerCount:      m.AvgPeerCount(),
		AvgDownloadSpeed:  m.AvgDownloadSpeed(),
		SpeedStdDev:       m.SpeedStdDev(),
		FailureRate:       m.FailureRate(),
		AvgStartupLatency: m.AvgStartupLatency(),
	}
}
