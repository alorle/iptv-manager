package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/alorle/iptv-manager/api"
	"github.com/alorle/iptv-manager/cache"
	"github.com/alorle/iptv-manager/config"
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

const (
	elcanoIPFSURL = "https://ipfs.io/ipns/k51qzi5uqu5di462t7j4vu4akwfhvtjhy88qbupktvoacqfqe9uforjvhyi4wr/hashes_acestream.m3u"
	neweraIPFSURL = "https://ipfs.io/ipns/k2k4r8oqlcjxsritt5mczkcn4mmvcmymbqw7113fz2flkrerfwfps004/data/listas/lista_fuera_iptv.m3u"
)

// Config holds the application configuration loaded from environment variables
type Config struct {
	HTTPAddress            string
	HTTPPort               string
	AcestreamPlayerBaseURL string
	AcestreamEngineURL     string
	CacheDir               string
	CacheTTL               time.Duration
	StreamBufferSize       int
	UseMultiplexing        bool
	ProxyReadTimeout       time.Duration
	ProxyWriteTimeout      time.Duration
	ProxyBufferSize        int
}

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

// isValidContentID validates that a content ID is exactly 40 hexadecimal characters
func isValidContentID(id string) bool {
	if len(id) != 40 {
		return false
	}
	for _, c := range id {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
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

// getEnvWithDefault returns the environment variable value or a default if not set
func getEnvWithDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

// parseIntEnv parses an integer environment variable with a default value
// Returns an error if the value is invalid or not positive
func parseIntEnv(key string, defaultValue int) (int, error) {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultValue, nil
	}

	val, err := strconv.Atoi(valStr)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}

	if val <= 0 {
		return 0, fmt.Errorf("%s must be positive", key)
	}

	return val, nil
}

// parseBoolEnv parses a boolean environment variable with a default value
// Accepts: "true", "1" for true; anything else for false
func parseBoolEnv(key string, defaultValue bool) bool {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultValue
	}
	return valStr == "true" || valStr == "1"
}

// parseDurationEnv parses a duration environment variable with a default value
// Returns an error if the value is invalid or not positive
func parseDurationEnv(key string, defaultValue time.Duration) (time.Duration, error) {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultValue, nil
	}

	val, err := time.ParseDuration(valStr)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}

	if val <= 0 {
		return 0, fmt.Errorf("%s must be positive", key)
	}

	return val, nil
}

// parseRequiredDuration parses a required duration environment variable
// Returns an error if the variable is not set, invalid, or not positive
func parseRequiredDuration(key string) (time.Duration, error) {
	valStr := os.Getenv(key)
	if valStr == "" {
		return 0, fmt.Errorf("%s environment variable is required", key)
	}

	val, err := time.ParseDuration(valStr)
	if err != nil {
		return 0, fmt.Errorf("invalid %s format (expected duration like '1h', '30m'): %w", key, err)
	}

	if val <= 0 {
		return 0, fmt.Errorf("%s must be positive, got: %s", key, valStr)
	}

	return val, nil
}

// validateCacheDir validates and normalizes the cache directory path
func validateCacheDir(dir string) (string, error) {
	if dir == "" {
		return "", fmt.Errorf("CACHE_DIR environment variable is required")
	}

	// Ensure cache directory is an absolute path
	if !filepath.IsAbs(dir) {
		absPath, err := filepath.Abs(dir)
		if err != nil {
			return "", fmt.Errorf("failed to resolve absolute path for CACHE_DIR: %w", err)
		}
		return absPath, nil
	}

	return dir, nil
}

func loadConfig() (*Config, error) {
	cfg := &Config{
		HTTPAddress:            getEnvWithDefault("HTTP_ADDRESS", "127.0.0.1"),
		HTTPPort:               getEnvWithDefault("HTTP_PORT", "8080"),
		AcestreamPlayerBaseURL: getEnvWithDefault("ACESTREAM_PLAYER_BASE_URL", "http://127.0.0.1:6878/ace/getstream"),
		AcestreamEngineURL:     getEnvWithDefault("ACESTREAM_ENGINE_URL", "http://127.0.0.1:6878"),
		UseMultiplexing:        parseBoolEnv("USE_MULTIPLEXING", true),
	}

	var err error

	// Parse integer values
	if cfg.StreamBufferSize, err = parseIntEnv("STREAM_BUFFER_SIZE", 1024*1024); err != nil {
		return nil, err
	}
	if cfg.ProxyBufferSize, err = parseIntEnv("PROXY_BUFFER_SIZE", 4194304); err != nil {
		return nil, err
	}

	// Parse duration values
	if cfg.ProxyReadTimeout, err = parseDurationEnv("PROXY_READ_TIMEOUT", 5*time.Second); err != nil {
		return nil, err
	}
	if cfg.ProxyWriteTimeout, err = parseDurationEnv("PROXY_WRITE_TIMEOUT", 10*time.Second); err != nil {
		return nil, err
	}

	// Parse required values
	if cfg.CacheDir, err = validateCacheDir(os.Getenv("CACHE_DIR")); err != nil {
		return nil, err
	}
	if cfg.CacheTTL, err = parseRequiredDuration("CACHE_TTL"); err != nil {
		return nil, err
	}

	return cfg, nil
}

// printConfig outputs the configuration to stdout
func printConfig(cfg *Config, resCfg *config.ResilienceConfig) {
	fmt.Printf("httpAddress: %v\n", cfg.HTTPAddress)
	fmt.Printf("httpPort: %v\n", cfg.HTTPPort)
	fmt.Printf("acestreamPlayerBaseUrl: %v\n", cfg.AcestreamPlayerBaseURL)
	fmt.Printf("acestreamEngineUrl: %v\n", cfg.AcestreamEngineURL)
	fmt.Printf("cacheDir: %v\n", cfg.CacheDir)
	fmt.Printf("cacheTTL: %v\n", cfg.CacheTTL)
	fmt.Printf("streamBufferSize: %v bytes\n", cfg.StreamBufferSize)
	fmt.Printf("useMultiplexing: %v\n", cfg.UseMultiplexing)
	fmt.Printf("proxyReadTimeout: %v\n", cfg.ProxyReadTimeout)
	fmt.Printf("proxyWriteTimeout: %v\n", cfg.ProxyWriteTimeout)
	fmt.Printf("proxyBufferSize: %v bytes\n", cfg.ProxyBufferSize)
	fmt.Printf("logLevel: %v\n", resCfg.LogLevel)
}

// initDependencies initializes all application components
func initDependencies(cfg *Config, resCfg *config.ResilienceConfig, resLogger *logging.Logger) (*dependencies, error) {
	// Initialize cache storage
	storage, err := cache.NewFileStorage(cfg.CacheDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cache storage: %w", err)
	}

	// Initialize overrides manager
	overridesPath := filepath.Join(cfg.CacheDir, "overrides.yaml")
	overridesMgr, err := overrides.NewManager(overridesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize overrides manager: %w", err)
	}
	log.Printf("Loaded %d channel overrides from %s", len(overridesMgr.List()), overridesPath)

	// Initialize EPG cache
	epgURL := os.Getenv("EPG_URL")
	if epgURL == "" {
		epgURL = "https://raw.githubusercontent.com/davidmuma/EPG_dobleM/master/guiatv.xml"
	}
	epgCache, err := epg.New(epgURL, 30*time.Second)
	if err != nil {
		log.Printf("Warning: Failed to initialize EPG cache: %v", err)
		log.Printf("TVG-ID validation will not be available")
	} else {
		log.Printf("EPG cache initialized with %d channels", epgCache.Count())
	}

	// Initialize fetcher with 30 second timeout
	fetch := fetcher.New(30*time.Second, storage, cfg.CacheTTL)

	// Initialize rewriter
	rw := rewriter.New()

	// Initialize multiplexer
	muxCfg := multiplexer.Config{
		BufferSize:       cfg.StreamBufferSize,
		ReadTimeout:      30 * time.Second,
		WriteTimeout:     10 * time.Second,
		ResilienceConfig: resCfg,
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
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// Load resilience configuration
	resCfg, err := config.LoadFromEnv()
	if err != nil {
		log.Fatalf("Failed to load resilience configuration: %v", err)
	}

	// Create resilience logger
	logLevel := logging.ParseLogLevel(resCfg.LogLevel)
	resLogger := logging.New(logLevel, "[resilience]")

	// Print configuration
	printConfig(cfg, resCfg)

	// Initialize all dependencies
	deps, err := initDependencies(cfg, resCfg, resLogger)
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
		Addr:         fmt.Sprintf("%s:%s", cfg.HTTPAddress, cfg.HTTPPort),
		ReadTimeout:  cfg.ProxyReadTimeout,
		WriteTimeout: cfg.ProxyWriteTimeout,
		ErrorLog:     log.Default(),
	}

	if err := s.ListenAndServe(); err != nil {
		fmt.Printf("Error starting server: %v\n", err)
		return
	}
}

// createUnifiedPlaylistHandler creates an HTTP handler for the unified playlist endpoint
func createUnifiedPlaylistHandler(deps *dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		baseURL := getBaseURL(r)

		// Fetch both sources
		elcanoContent, elcanoFromCache, elcanoStale, elcanoErr := deps.fetch.FetchWithCache(elcanoIPFSURL)
		neweraContent, neweraFromCache, neweraStale, neweraErr := deps.fetch.FetchWithCache(neweraIPFSURL)

		// Check if both sources failed
		if elcanoErr != nil && neweraErr != nil {
			log.Printf("Failed to fetch unified playlist - both sources failed: elcano=%v, newera=%v", elcanoErr, neweraErr)
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
			return
		}

		// Build merged content starting with M3U header
		var mergedContent strings.Builder
		mergedContent.WriteString("#EXTM3U\n")

		// Add elcano content if available
		if elcanoErr == nil {
			// Log cache status for elcano
			if elcanoFromCache {
				if elcanoStale {
					log.Printf("Using stale cache for elcano in unified playlist")
				} else {
					log.Printf("Using fresh cache for elcano in unified playlist")
				}
			} else {
				log.Printf("Using fresh content for elcano in unified playlist")
			}

			// Parse and add elcano streams (skip header line)
			elcanoStr := string(elcanoContent)
			if strings.HasPrefix(elcanoStr, "#EXTM3U") {
				elcanoStr = strings.TrimPrefix(elcanoStr, "#EXTM3U")
				elcanoStr = strings.TrimLeft(elcanoStr, "\n")
			}
			mergedContent.WriteString(elcanoStr)
		} else {
			log.Printf("Skipping elcano source in unified playlist: %v", elcanoErr)
		}

		// Add newera content if available
		if neweraErr == nil {
			// Log cache status for newera
			if neweraFromCache {
				if neweraStale {
					log.Printf("Using stale cache for newera in unified playlist")
				} else {
					log.Printf("Using fresh cache for newera in unified playlist")
				}
			} else {
				log.Printf("Using fresh content for newera in unified playlist")
			}

			// Parse and add newera streams (skip header line)
			neweraStr := string(neweraContent)
			if strings.HasPrefix(neweraStr, "#EXTM3U") {
				neweraStr = strings.TrimPrefix(neweraStr, "#EXTM3U")
				neweraStr = strings.TrimLeft(neweraStr, "\n")
			}
			if mergedContent.Len() > len("#EXTM3U\n") {
				mergedContent.WriteString("\n")
			}
			mergedContent.WriteString(neweraStr)
		} else {
			log.Printf("Skipping newera source in unified playlist: %v", neweraErr)
		}

		// Clean up orphaned overrides if we have fresh data from at least one source
		hasFreshData := (elcanoErr == nil && !elcanoStale) || (neweraErr == nil && !neweraStale)
		if hasFreshData {
			// Collect all valid acestream IDs from both sources
			var validIDs []string
			if elcanoErr == nil {
				elcanoIDs := rewriter.ExtractAcestreamIDs(elcanoContent)
				validIDs = append(validIDs, elcanoIDs...)
			}
			if neweraErr == nil {
				neweraIDs := rewriter.ExtractAcestreamIDs(neweraContent)
				validIDs = append(validIDs, neweraIDs...)
			}

			// Clean orphaned overrides
			if deletedCount, err := deps.overridesMgr.CleanOrphans(validIDs); err != nil {
				log.Printf("WARNING: Failed to clean orphaned overrides: %v", err)
			} else if deletedCount > 0 {
				log.Printf("Cleaned up %d orphaned override(s)", deletedCount)
			}
		} else {
			log.Printf("Skipping orphan cleanup - using only stale cache data")
		}

		// Apply channel overrides BEFORE deduplication and sorting
		mergedBytes := []byte(mergedContent.String())
		overriddenContent := rewriter.ApplyOverrides(mergedBytes, deps.overridesMgr)

		// Apply deduplication by acestream ID
		deduplicatedContent := rewriter.DeduplicateStreams(overriddenContent)

		// Apply alphabetical sorting by display name
		sortedContent := rewriter.SortStreamsByName(deduplicatedContent)

		// Rewrite acestream:// URLs and remove logos
		rewrittenContent := deps.rewriter.RewriteM3U(sortedContent, baseURL)

		// Set content type
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
func createStreamHandler(cfg *Config, deps *dependencies) http.HandlerFunc {
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
		if !isValidContentID(contentID) {
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
		upstreamURL := fmt.Sprintf("%s/ace/getstream?id=%s&pid=%d", cfg.AcestreamEngineURL, contentID, pid)

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
func setupHandlers(cfg *Config, deps *dependencies) http.Handler {
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

	// Elcano playlist endpoint
	handler.HandleFunc("/playlists/elcano.m3u", createPlaylistHandler(deps, elcanoIPFSURL, "elcano"))

	// NewEra playlist endpoint
	handler.HandleFunc("/playlists/newera.m3u", createPlaylistHandler(deps, neweraIPFSURL, "newera"))

	// Unified playlist endpoint - merges all sources
	handler.HandleFunc("/playlist.m3u", createUnifiedPlaylistHandler(deps))

	// API endpoints for channels
	elcanoURL := elcanoIPFSURL
	neweraURL := neweraIPFSURL
	channelsHandler := api.NewChannelsHandler(deps.fetch, deps.overridesMgr, elcanoURL, neweraURL)
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
