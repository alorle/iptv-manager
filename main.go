package main

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/alorle/iptv-manager/api"
	"github.com/alorle/iptv-manager/cache"
	"github.com/alorle/iptv-manager/config"
	"github.com/alorle/iptv-manager/domain"
	"github.com/alorle/iptv-manager/epg"
	"github.com/alorle/iptv-manager/fetcher"
	"github.com/alorle/iptv-manager/logging"
	"github.com/alorle/iptv-manager/metrics"
	"github.com/alorle/iptv-manager/multiplexer"
	"github.com/alorle/iptv-manager/overrides"
	"github.com/alorle/iptv-manager/pidmanager"
	"github.com/alorle/iptv-manager/rewriter"
	"github.com/alorle/iptv-manager/ui"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// dependencies holds all initialized application components
type dependencies struct {
	storage      *cache.FileStorage
	overridesMgr *overrides.Manager
	epgCache     *epg.Cache
	fetch        *fetcher.Fetcher
	rewriter     *rewriter.Rewriter
	multiplexer  *multiplexer.Multiplexer
	pidMgr       *pidmanager.Manager
}


// getBaseURL returns the scheme and authority (scheme://host) from an HTTP request
func getBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	return fmt.Sprintf("%s://%s", scheme, r.Host)
}

// printConfig outputs the configuration to stdout
func printConfig(cfg *config.Config) {
	fmt.Printf("httpAddress: %v\n", cfg.HTTP.Address)
	fmt.Printf("httpPort: %v\n", cfg.HTTP.Port)
	fmt.Printf("acestreamPlayerBaseUrl: %v\n", cfg.Acestream.PlayerBaseURL)
	fmt.Printf("acestreamEngineUrl: %v\n", cfg.Acestream.EngineURL)
	fmt.Printf("cacheDir: %v\n", cfg.Cache.Dir)
	fmt.Printf("cacheTTL: %v\n", cfg.Cache.TTL)
	fmt.Printf("streamBufferSize: %v bytes\n", cfg.Stream.BufferSize)
	fmt.Printf("useMultiplexing: %v\n", cfg.Stream.UseMultiplexing)
	fmt.Printf("proxyReadTimeout: %v\n", cfg.Proxy.ReadTimeout)
	fmt.Printf("proxyWriteTimeout: %v\n", cfg.Proxy.WriteTimeout)
	fmt.Printf("proxyBufferSize: %v bytes\n", cfg.Proxy.BufferSize)
	fmt.Printf("playlistSources: %d\n", len(cfg.Playlists))
	for _, pl := range cfg.Playlists {
		fmt.Printf("  - %s: %s\n", pl.Name, pl.URL)
	}
	fmt.Printf("epgUrl: %v\n", cfg.EPG.URL)
	fmt.Printf("logLevel: %v\n", cfg.Resilience.LogLevel)
}

// initDependencies initializes all application components
func initDependencies(cfg *config.Config, resLogger *logging.Logger) (*dependencies, error) {
	// Initialize cache storage
	storage, err := cache.NewFileStorage(cfg.Cache.Dir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cache storage: %w", err)
	}

	// Initialize overrides manager
	overridesPath := filepath.Join(cfg.Cache.Dir, "overrides.yaml")
	overridesMgr, err := overrides.NewManager(overridesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize overrides manager: %w", err)
	}
	log.Printf("Loaded %d channel overrides from %s", len(overridesMgr.List()), overridesPath)

	// Initialize EPG cache
	epgCache, err := epg.New(cfg.EPG.URL, 30*time.Second)
	if err != nil {
		log.Printf("Warning: Failed to initialize EPG cache: %v", err)
		log.Printf("TVG-ID validation will not be available")
	} else {
		log.Printf("EPG cache initialized with %d channels", epgCache.Count())
	}

	// Initialize fetcher with 30 second timeout
	fetch := fetcher.New(30*time.Second, storage, cfg.Cache.TTL)

	// Initialize rewriter
	rw := rewriter.New()

	// Initialize multiplexer
	muxCfg := multiplexer.Config{
		BufferSize:       cfg.Stream.BufferSize,
		ReadTimeout:      30 * time.Second,
		WriteTimeout:     10 * time.Second,
		ResilienceConfig: &cfg.Resilience,
		ResilienceLogger: resLogger,
	}
	mux := multiplexer.New(muxCfg)

	// Initialize PID manager
	pidMgr := pidmanager.NewManager()

	return &dependencies{
		storage:      storage,
		overridesMgr: overridesMgr,
		epgCache:     epgCache,
		fetch:        fetch,
		rewriter:     rw,
		multiplexer:  mux,
		pidMgr:       pidMgr,
	}, nil
}

func main() {
	// Load and validate configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// Create resilience logger
	logLevel := logging.ParseLogLevel(cfg.Resilience.LogLevel)
	resLogger := logging.New(logLevel, "[resilience]")

	// Print configuration
	printConfig(cfg)

	// Initialize all dependencies
	deps, err := initDependencies(cfg, resLogger)
	if err != nil {
		log.Fatalf("Failed to initialize dependencies: %v", err)
	}

	// Initialize metrics to ensure they appear in /metrics output
	metrics.SetStreamsActive(0)
	metrics.SetClientsConnected(0)

	// Setup HTTP handlers
	handler := setupHandlers(cfg, deps)

	s := &http.Server{
		Handler:      handler,
		Addr:         fmt.Sprintf("%s:%s", cfg.HTTP.Address, cfg.HTTP.Port),
		ReadTimeout:  cfg.Proxy.ReadTimeout,
		WriteTimeout: cfg.Proxy.WriteTimeout,
		ErrorLog:     log.Default(),
	}

	if err := s.ListenAndServe(); err != nil {
		fmt.Printf("Error starting server: %v\n", err)
		return
	}
}

// logCacheStatus logs the cache status for a fetched source
func logCacheStatus(sourceName string, fromCache, stale bool) {
	if !fromCache {
		log.Printf("Using fresh content for %s in unified playlist", sourceName)
		return
	}

	if stale {
		log.Printf("Using stale cache for %s in unified playlist", sourceName)
	} else {
		log.Printf("Using fresh cache for %s in unified playlist", sourceName)
	}
}

// stripM3UHeader removes the #EXTM3U header from playlist content if present
func stripM3UHeader(content []byte) string {
	str := string(content)
	if strings.HasPrefix(str, "#EXTM3U") {
		str = strings.TrimPrefix(str, "#EXTM3U")
		str = strings.TrimLeft(str, "\n")
	}
	return str
}

// playlistSource represents a fetched playlist source
type playlistSource struct {
	name      string
	content   []byte
	err       error
	fromCache bool
	stale     bool
}

// mergePlaylistSources merges multiple playlist sources into a single M3U
func mergePlaylistSources(sources []playlistSource) string {
	var merged strings.Builder
	merged.WriteString("#EXTM3U\n")

	for _, source := range sources {
		if source.err != nil {
			log.Printf("Skipping %s source in unified playlist: %v", source.name, source.err)
			continue
		}

		logCacheStatus(source.name, source.fromCache, source.stale)

		// Add newline separator if we're appending to existing content
		if merged.Len() > len("#EXTM3U\n") {
			merged.WriteString("\n")
		}
		merged.WriteString(stripM3UHeader(source.content))
	}

	return merged.String()
}

// cleanOrphanedOverrides removes overrides for channels no longer in the playlists
func cleanOrphanedOverrides(deps *dependencies, sources []struct {
	content []byte
	err     error
	stale   bool
}) {
	// Only clean if we have fresh data from at least one source
	hasFreshData := false
	for _, source := range sources {
		if source.err == nil && !source.stale {
			hasFreshData = true
			break
		}
	}

	if !hasFreshData {
		log.Printf("Skipping orphan cleanup - using only stale cache data")
		return
	}

	// Collect all valid acestream IDs from successful fetches
	var validIDs []string
	for _, source := range sources {
		if source.err == nil {
			ids := rewriter.ExtractAcestreamIDs(source.content)
			validIDs = append(validIDs, ids...)
		}
	}

	// Clean orphaned overrides
	deletedCount, err := deps.overridesMgr.CleanOrphans(validIDs)
	if err != nil {
		log.Printf("WARNING: Failed to clean orphaned overrides: %v", err)
	} else if deletedCount > 0 {
		log.Printf("Cleaned up %d orphaned override(s)", deletedCount)
	}
}

// processPlaylist applies the full M3U processing pipeline: overrides, dedup, sort, rewrite
func processPlaylist(deps *dependencies, content string, baseURL string) []byte {
	contentBytes := []byte(content)

	// Apply channel overrides BEFORE deduplication and sorting
	overridden := rewriter.ApplyOverrides(contentBytes, deps.overridesMgr)

	// Apply deduplication by acestream ID
	deduplicated := rewriter.DeduplicateStreams(overridden)

	// Apply alphabetical sorting by display name
	sorted := rewriter.SortStreamsByName(deduplicated)

	// Rewrite acestream:// URLs and remove logos
	return deps.rewriter.RewriteM3U(sorted, baseURL)
}

// createUnifiedPlaylistHandler creates an HTTP handler for the unified playlist endpoint
func createUnifiedPlaylistHandler(cfg *config.Config, deps *dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		baseURL := getBaseURL(r)

		// Fetch all configured sources
		sources := make([]playlistSource, 0, len(cfg.Playlists))
		allFailed := true

		for _, pl := range cfg.Playlists {
			content, fromCache, stale, err := deps.fetch.FetchWithCache(pl.URL)
			sources = append(sources, playlistSource{
				name:      pl.Name,
				content:   content,
				err:       err,
				fromCache: fromCache,
				stale:     stale,
			})
			if err == nil {
				allFailed = false
			}
		}

		// Check if all sources failed
		if allFailed {
			var errMsgs []string
			for _, src := range sources {
				if src.err != nil {
					errMsgs = append(errMsgs, fmt.Sprintf("%s=%v", src.name, src.err))
				}
			}
			log.Printf("Failed to fetch unified playlist - all sources failed: %s", strings.Join(errMsgs, ", "))
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
			return
		}

		// Merge playlist sources
		mergedContent := mergePlaylistSources(sources)

		// Clean up orphaned overrides
		cleanupSources := make([]struct {
			content []byte
			err     error
			stale   bool
		}, len(sources))
		for i, src := range sources {
			cleanupSources[i] = struct {
				content []byte
				err     error
				stale   bool
			}{src.content, src.err, src.stale}
		}
		cleanOrphanedOverrides(deps, cleanupSources)

		// Process playlist through full pipeline
		rewrittenContent := processPlaylist(deps, mergedContent, baseURL)

		// Send response
		w.Header().Set("Content-Type", "audio/x-mpegurl")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(rewrittenContent); err != nil {
			log.Printf("Error writing unified playlist: %v", err)
		}
	}
}

// createPlaylistHandler creates an HTTP handler for serving a single playlist source
func createPlaylistHandler(deps *dependencies, sourceURL string, sourceName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		baseURL := getBaseURL(r)

		// Fetch with cache fallback
		content, fromCache, stale, err := deps.fetch.FetchWithCache(sourceURL)
		if err != nil {
			log.Printf("Failed to fetch %s playlist: %v", sourceName, err)
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
			return
		}

		// Log cache status
		if fromCache {
			if stale {
				log.Printf("Serving stale cache for %s playlist", sourceName)
			} else {
				log.Printf("Serving fresh cache for %s playlist", sourceName)
			}
		} else {
			log.Printf("Serving fresh content for %s playlist", sourceName)
		}

		// Apply channel overrides
		content = rewriter.ApplyOverrides(content, deps.overridesMgr)

		// Rewrite acestream:// URLs
		rewrittenContent := deps.rewriter.RewriteM3U(content, baseURL)

		// Set content type
		w.Header().Set("Content-Type", "audio/x-mpegurl")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(rewrittenContent); err != nil {
			log.Printf("Error writing %s playlist: %v", sourceName, err)
		}
	}
}

// createStreamHandler creates the HTTP handler for streaming endpoints
func createStreamHandler(cfg *config.Config, deps *dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Allow GET and HEAD requests (VLC sends HEAD to probe stream before playing)
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Get content ID from query parameter
		contentID := r.URL.Query().Get("id")
		if contentID == "" {
			http.Error(w, "Missing id parameter", http.StatusBadRequest)
			return
		}

		// Validate content ID format (40 hex characters)
		if !domain.IsValidAcestreamID(contentID) {
			http.Error(w, "Invalid id format: must be 40 hexadecimal characters", http.StatusBadRequest)
			return
		}

		// For HEAD requests, just return headers without starting the stream
		if r.Method == http.MethodHead {
			w.Header().Set("Content-Type", "video/mp2t")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")
			w.WriteHeader(http.StatusOK)
			return
		}

		// Extract client identifier
		clientInfo := pidmanager.ExtractClientIdentifier(r)
		pid := deps.pidMgr.GetOrCreatePID(contentID, clientInfo)
		clientID := fmt.Sprintf("%s-%d", clientInfo.IP, pid)

		log.Printf("Stream request: contentID=%s, clientID=%s, pid=%d", contentID, clientID, pid)

		// Build upstream URL with PID and optional transcode parameters
		upstreamURL := fmt.Sprintf("%s/ace/getstream?id=%s&pid=%d", cfg.Acestream.EngineURL, contentID, pid)

		// Add optional transcode parameters
		if transcodeAudio := r.URL.Query().Get("transcode_audio"); transcodeAudio != "" {
			upstreamURL += "&transcode_audio=" + transcodeAudio
		}
		if transcodeMp3 := r.URL.Query().Get("transcode_mp3"); transcodeMp3 != "" {
			upstreamURL += "&transcode_mp3=" + transcodeMp3
		}
		if transcodeAc3 := r.URL.Query().Get("transcode_ac3"); transcodeAc3 != "" {
			upstreamURL += "&transcode_ac3=" + transcodeAc3
		}

		// Serve the stream through multiplexer
		// Note: multiplexer sets Content-Type to video/mp2t automatically
		if err := deps.multiplexer.ServeStream(r.Context(), w, contentID, upstreamURL, clientID); err != nil {
			log.Printf("Failed to serve stream for contentID=%s: %v", contentID, err)
			// Check if it's a connection error to Engine
			if strings.Contains(err.Error(), "connect") || strings.Contains(err.Error(), "upstream") {
				http.Error(w, "Bad Gateway: cannot connect to Engine", http.StatusBadGateway)
				return
			}
		}

		// Release PID when client disconnects
		if err := deps.pidMgr.ReleasePID(pid); err != nil {
			log.Printf("Failed to release PID %d: %v", pid, err)
		}

		// Cleanup disconnected sessions periodically
		if cleaned := deps.pidMgr.CleanupDisconnected(); cleaned > 0 {
			log.Printf("Cleaned up %d disconnected sessions", cleaned)
		}
	}
}

// setupHandlers configures all HTTP routes and handlers
func setupHandlers(cfg *config.Config, deps *dependencies) http.Handler {
	handler := http.NewServeMux()

	handler.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			log.Printf("Error writing health response: %v", err)
		}
	})

	// Prometheus metrics endpoint
	handler.Handle("/metrics", promhttp.Handler())

	// Stream handler - shared by /stream and /ace/getstream
	streamHandler := createStreamHandler(cfg, deps)
	handler.HandleFunc("/stream", streamHandler)
	handler.HandleFunc("/ace/getstream", streamHandler)

	// Individual playlist endpoints - one for each configured source
	for _, pl := range cfg.Playlists {
		path := fmt.Sprintf("/playlists/%s.m3u", pl.Name)
		handler.HandleFunc(path, createPlaylistHandler(deps, pl.URL, pl.Name))
	}

	// Unified playlist endpoint - merges all sources
	handler.HandleFunc("/playlist.m3u", createUnifiedPlaylistHandler(cfg, deps))

	// API endpoints for channels - pass all playlist URLs
	playlistURLs := make([]string, len(cfg.Playlists))
	for i, pl := range cfg.Playlists {
		playlistURLs[i] = pl.URL
	}
	channelsHandler := api.NewChannelsHandler(deps.fetch, deps.overridesMgr, playlistURLs...)
	// Handle both /api/channels and /api/channels/{id}
	handler.Handle("/api/channels", channelsHandler)
	handler.Handle("/api/channels/", channelsHandler)

	// API endpoint for TVG-ID validation
	if deps.epgCache != nil {
		validateHandler := api.NewValidateHandler(deps.epgCache)
		handler.Handle("/api/validate/tvg-id", validateHandler)
	}

	// API endpoints for overrides CRUD
	overridesHandler := api.NewOverridesHandler(deps.overridesMgr, deps.epgCache)
	handler.Handle("/api/overrides", overridesHandler)
	handler.Handle("/api/overrides/", overridesHandler)

	// Mount embedded UI at /ui/ path
	handler.Handle("/", ui.Handler("/"))

	return handler
}
