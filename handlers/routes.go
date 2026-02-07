package handlers

import (
	"fmt"
	"net/http"

	"github.com/alorle/iptv-manager/api"
	"github.com/alorle/iptv-manager/config"
	"github.com/alorle/iptv-manager/epg"
	"github.com/alorle/iptv-manager/fetcher"
	"github.com/alorle/iptv-manager/logging"
	"github.com/alorle/iptv-manager/multiplexer"
	"github.com/alorle/iptv-manager/overrides"
	"github.com/alorle/iptv-manager/pidmanager"
	"github.com/alorle/iptv-manager/rewriter"
	"github.com/alorle/iptv-manager/ui"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Dependencies holds all the dependencies needed by the handlers
type Dependencies struct {
	Logger       *logging.Logger
	Fetcher      fetcher.Interface
	OverridesMgr overrides.Interface
	EPGCache     *epg.Cache
	Rewriter     rewriter.Interface
	Multiplexer  *multiplexer.Multiplexer
	PidMgr       *pidmanager.Manager
}

// SetupRoutes configures all HTTP routes and handlers
func SetupRoutes(cfg *config.Config, deps Dependencies) http.Handler {
	handler := http.NewServeMux()

	handler.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			deps.Logger.Warn("Failed to write health response", map[string]interface{}{
				"error": err.Error(),
			})
		}
	})

	// Prometheus metrics endpoint
	handler.Handle("/metrics", promhttp.Handler())

	// Stream handler - shared by /stream and /ace/getstream
	streamDeps := StreamDependencies{
		Logger:      deps.Logger,
		Multiplexer: deps.Multiplexer,
		PidMgr:      deps.PidMgr,
	}
	streamHandler := CreateStreamHandler(cfg, streamDeps)
	handler.HandleFunc("/stream", streamHandler)
	handler.HandleFunc("/ace/getstream", streamHandler)

	// Individual playlist endpoints - one for each configured source
	playlistDeps := PlaylistDependencies{
		Logger:       deps.Logger,
		Fetcher:      deps.Fetcher,
		OverridesMgr: deps.OverridesMgr,
		Rewriter:     deps.Rewriter,
	}
	for _, pl := range cfg.Playlists {
		path := fmt.Sprintf("/playlists/%s.m3u", pl.Name)
		handler.HandleFunc(path, CreatePlaylistHandler(playlistDeps, pl.URL, pl.Name))
	}

	// Unified playlist endpoint - merges all sources
	handler.HandleFunc("/playlist.m3u", CreateUnifiedPlaylistHandler(cfg, playlistDeps))

	// API endpoints for channels - pass all playlist URLs
	playlistURLs := make([]string, len(cfg.Playlists))
	for i, pl := range cfg.Playlists {
		playlistURLs[i] = pl.URL
	}
	channelsHandler := api.NewChannelsHandler(deps.Fetcher, deps.OverridesMgr, deps.Logger, playlistURLs...)
	// Handle both /api/channels and /api/channels/{id}
	handler.Handle("/api/channels", channelsHandler)
	handler.Handle("/api/channels/", channelsHandler)

	// API endpoint for TVG-ID validation
	if deps.EPGCache != nil {
		validateHandler := api.NewValidateHandler(deps.EPGCache, deps.Logger)
		handler.Handle("/api/validate/tvg-id", validateHandler)
	}

	// API endpoints for overrides CRUD
	overridesHandler := api.NewOverridesHandler(deps.OverridesMgr, deps.EPGCache, deps.Logger)
	handler.Handle("/api/overrides", overridesHandler)
	handler.Handle("/api/overrides/", overridesHandler)

	// Mount embedded UI at /ui/ path
	handler.Handle("/", ui.Handler("/"))

	return handler
}
