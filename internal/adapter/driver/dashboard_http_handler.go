package driver

import (
	"net/http"

	"github.com/alorle/iptv-manager/internal/application"
)

// DashboardHTTPHandler serves an aggregated dashboard view combining
// system health, per-channel health, and active streaming sessions.
type DashboardHTTPHandler struct {
	channelService *application.ChannelService
	probeService   *application.ProbeService
	proxyService   *application.AceStreamProxyService
	healthService  *application.HealthService
}

// NewDashboardHTTPHandler creates a new dashboard handler.
func NewDashboardHTTPHandler(
	channelService *application.ChannelService,
	probeService *application.ProbeService,
	proxyService *application.AceStreamProxyService,
	healthService *application.HealthService,
) *DashboardHTTPHandler {
	return &DashboardHTTPHandler{
		channelService: channelService,
		probeService:   probeService,
		proxyService:   proxyService,
		healthService:  healthService,
	}
}

type dashboardResponse struct {
	System   dashboardSystemHealth    `json:"system"`
	Channels []dashboardChannelHealth `json:"channels"`
	Sessions []dashboardSession       `json:"sessions"`
}

type dashboardSystemHealth struct {
	Status          string `json:"status"`
	DB              string `json:"db"`
	AceStreamEngine string `json:"acestream_engine"`
}

type dashboardChannelHealth struct {
	Name        string               `json:"name"`
	Status      string               `json:"status"`
	StreamCount int                  `json:"stream_count"`
	BestScore   float64              `json:"best_score"`
	HealthLevel string               `json:"health_level"`
	LastProbe   *probeResultResponse `json:"last_probe,omitempty"`
	Watching    int                  `json:"watching"`
}

type dashboardSession struct {
	InfoHash    string `json:"info_hash"`
	ClientCount int    `json:"client_count"`
}

// ServeHTTP handles GET /dashboard.
func (h *DashboardHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx := r.Context()

	sysHealth := h.healthService.Check(ctx)

	channels, err := h.channelService.ListChannels(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list channels")
		return
	}

	activeSessions := h.proxyService.GetActiveStreams()
	sessionMap := make(map[string]int, len(activeSessions))
	for _, s := range activeSessions {
		sessionMap[s.InfoHash] = s.ClientCount
	}

	channelResponses := make([]dashboardChannelHealth, 0, len(channels))
	for _, ch := range channels {
		health, err := h.probeService.GetChannelHealth(ctx, ch.Name())
		if err != nil {
			continue
		}

		watching := 0
		for _, ih := range health.InfoHashes {
			watching += sessionMap[ih]
		}

		var lastProbe *probeResultResponse
		if health.LastProbe != nil {
			r := toProbeResultResponse(*health.LastProbe)
			lastProbe = &r
		}

		channelResponses = append(channelResponses, dashboardChannelHealth{
			Name:        ch.Name(),
			Status:      string(ch.Status()),
			StreamCount: health.StreamCount,
			BestScore:   health.BestScore,
			HealthLevel: healthLevel(health.BestScore, health.LastProbe != nil),
			LastProbe:   lastProbe,
			Watching:    watching,
		})
	}

	sessions := make([]dashboardSession, len(activeSessions))
	for i, s := range activeSessions {
		sessions[i] = dashboardSession{
			InfoHash:    s.InfoHash,
			ClientCount: s.ClientCount,
		}
	}

	writeJSON(w, http.StatusOK, dashboardResponse{
		System: dashboardSystemHealth{
			Status:          sysHealth.Status,
			DB:              sysHealth.DB.Status,
			AceStreamEngine: sysHealth.AceStreamEngine.Status,
		},
		Channels: channelResponses,
		Sessions: sessions,
	})
}

func healthLevel(bestScore float64, hasProbes bool) string {
	if !hasProbes {
		return "unknown"
	}
	switch {
	case bestScore >= 0.7:
		return "green"
	case bestScore >= 0.4:
		return "yellow"
	default:
		return "red"
	}
}
