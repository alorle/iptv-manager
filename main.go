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

	"github.com/alorle/iptv-manager/cache"
	"github.com/alorle/iptv-manager/fetcher"
	"github.com/alorle/iptv-manager/multiplexer"
	"github.com/alorle/iptv-manager/pidmanager"
	"github.com/alorle/iptv-manager/rewriter"
)

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

func loadConfig() (*Config, error) {
	cfg := &Config{
		HTTPAddress:            os.Getenv("HTTP_ADDRESS"),
		HTTPPort:               os.Getenv("HTTP_PORT"),
		AcestreamPlayerBaseURL: os.Getenv("ACESTREAM_PLAYER_BASE_URL"),
		AcestreamEngineURL:     os.Getenv("ACESTREAM_ENGINE_URL"),
		CacheDir:               os.Getenv("CACHE_DIR"),
	}

	// Set defaults
	if cfg.HTTPAddress == "" {
		cfg.HTTPAddress = "127.0.0.1"
	}
	if cfg.HTTPPort == "" {
		cfg.HTTPPort = "8080"
	}
	if cfg.AcestreamPlayerBaseURL == "" {
		cfg.AcestreamPlayerBaseURL = "http://127.0.0.1:6878/ace/getstream"
	}
	if cfg.AcestreamEngineURL == "" {
		cfg.AcestreamEngineURL = "http://127.0.0.1:6878"
	}

	// Parse STREAM_BUFFER_SIZE (default 1MB)
	bufferSizeStr := os.Getenv("STREAM_BUFFER_SIZE")
	if bufferSizeStr == "" {
		cfg.StreamBufferSize = 1024 * 1024 // 1MB default
	} else {
		bufferSize, err := strconv.Atoi(bufferSizeStr)
		if err != nil {
			return nil, fmt.Errorf("invalid STREAM_BUFFER_SIZE: %w", err)
		}
		if bufferSize <= 0 {
			return nil, fmt.Errorf("STREAM_BUFFER_SIZE must be positive")
		}
		cfg.StreamBufferSize = bufferSize
	}

	// Parse USE_MULTIPLEXING (default true)
	useMultiplexingStr := os.Getenv("USE_MULTIPLEXING")
	if useMultiplexingStr == "" || useMultiplexingStr == "true" || useMultiplexingStr == "1" {
		cfg.UseMultiplexing = true
	} else {
		cfg.UseMultiplexing = false
	}

	// Parse PROXY_READ_TIMEOUT (default 5s)
	readTimeoutStr := os.Getenv("PROXY_READ_TIMEOUT")
	if readTimeoutStr == "" {
		cfg.ProxyReadTimeout = 5 * time.Second
	} else {
		readTimeout, err := time.ParseDuration(readTimeoutStr)
		if err != nil {
			return nil, fmt.Errorf("invalid PROXY_READ_TIMEOUT: %w", err)
		}
		if readTimeout <= 0 {
			return nil, fmt.Errorf("PROXY_READ_TIMEOUT must be positive")
		}
		cfg.ProxyReadTimeout = readTimeout
	}

	// Parse PROXY_WRITE_TIMEOUT (default 10s)
	writeTimeoutStr := os.Getenv("PROXY_WRITE_TIMEOUT")
	if writeTimeoutStr == "" {
		cfg.ProxyWriteTimeout = 10 * time.Second
	} else {
		writeTimeout, err := time.ParseDuration(writeTimeoutStr)
		if err != nil {
			return nil, fmt.Errorf("invalid PROXY_WRITE_TIMEOUT: %w", err)
		}
		if writeTimeout <= 0 {
			return nil, fmt.Errorf("PROXY_WRITE_TIMEOUT must be positive")
		}
		cfg.ProxyWriteTimeout = writeTimeout
	}

	// Validate and set CACHE_DIR
	if cfg.CacheDir == "" {
		return nil, fmt.Errorf("CACHE_DIR environment variable is required")
	}
	// Ensure cache directory is an absolute path
	if !filepath.IsAbs(cfg.CacheDir) {
		absPath, err := filepath.Abs(cfg.CacheDir)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve absolute path for CACHE_DIR: %w", err)
		}
		cfg.CacheDir = absPath
	}

	// Parse CACHE_TTL
	cacheTTLStr := os.Getenv("CACHE_TTL")
	if cacheTTLStr == "" {
		return nil, fmt.Errorf("CACHE_TTL environment variable is required")
	}
	ttl, err := time.ParseDuration(cacheTTLStr)
	if err != nil {
		return nil, fmt.Errorf("invalid CACHE_TTL format (expected duration like '1h', '30m'): %w", err)
	}
	if ttl <= 0 {
		return nil, fmt.Errorf("CACHE_TTL must be positive, got: %s", cacheTTLStr)
	}
	cfg.CacheTTL = ttl

	return cfg, nil
}

func main() {
	// Load and validate configuration
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// Print configuration
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

	// Initialize cache storage
	storage, err := cache.NewFileStorage(cfg.CacheDir)
	if err != nil {
		log.Fatalf("Failed to initialize cache storage: %v", err)
	}

	// Initialize fetcher with 30 second timeout
	fetch := fetcher.New(30*time.Second, storage, cfg.CacheTTL)

	// Initialize rewriter - use local stream endpoint if multiplexing is enabled
	var playerURL string
	if cfg.UseMultiplexing {
		playerURL = fmt.Sprintf("http://%s:%s/stream", cfg.HTTPAddress, cfg.HTTPPort)
	} else {
		playerURL = cfg.AcestreamPlayerBaseURL
	}
	rw := rewriter.New(playerURL)

	// Initialize multiplexer
	muxCfg := multiplexer.Config{
		BufferSize:   cfg.StreamBufferSize,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	mux := multiplexer.New(muxCfg)

	// Initialize PID manager
	pidMgr := pidmanager.NewManager()

	handler := http.NewServeMux()

	handler.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Stream proxy endpoint with multiplexing
	handler.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
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

		// Extract client identifier
		clientInfo := pidmanager.ExtractClientIdentifier(r)
		pid := pidMgr.GetOrCreatePID(contentID, clientInfo)
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
		if err := mux.ServeStream(r.Context(), w, contentID, upstreamURL, clientID); err != nil {
			log.Printf("Failed to serve stream for contentID=%s: %v", contentID, err)
			// Check if it's a connection error to Engine
			if strings.Contains(err.Error(), "connect") || strings.Contains(err.Error(), "upstream") {
				http.Error(w, "Bad Gateway: cannot connect to Engine", http.StatusBadGateway)
				return
			}
		}

		// Release PID when client disconnects
		if err := pidMgr.ReleasePID(pid); err != nil {
			log.Printf("Failed to release PID %d: %v", pid, err)
		}

		// Cleanup disconnected sessions periodically
		if cleaned := pidMgr.CleanupDisconnected(); cleaned > 0 {
			log.Printf("Cleaned up %d disconnected sessions", cleaned)
		}
	})

	// Elcano playlist endpoint
	handler.HandleFunc("/playlists/elcano.m3u", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		sourceURL := "https://ipfs.io/ipns/k51qzi5uqu5di462t7j4vu4akwfhvtjhy88qbupktvoacqfqe9uforjvhyi4wr/hashes_acestream.m3u"

		// Fetch with cache fallback
		content, fromCache, stale, err := fetch.FetchWithCache(sourceURL)
		if err != nil {
			log.Printf("Failed to fetch elcano playlist: %v", err)
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
			return
		}

		// Log cache status
		if fromCache {
			if stale {
				log.Printf("Serving stale cache for elcano playlist")
			} else {
				log.Printf("Serving fresh cache for elcano playlist")
			}
		} else {
			log.Printf("Serving fresh content for elcano playlist")
		}

		// Rewrite acestream:// URLs
		rewrittenContent := rw.RewriteM3U(content)

		// Set content type
		w.Header().Set("Content-Type", "audio/x-mpegurl")
		w.WriteHeader(http.StatusOK)
		w.Write(rewrittenContent)
	})

	// NewEra playlist endpoint
	handler.HandleFunc("/playlists/newera.m3u", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		sourceURL := "https://ipfs.io/ipns/k2k4r8oqlcjxsritt5mczkcn4mmvcmymbqw7113fz2flkrerfwfps004/data/listas/lista_fuera_iptv.m3u"

		// Fetch with cache fallback
		content, fromCache, stale, err := fetch.FetchWithCache(sourceURL)
		if err != nil {
			log.Printf("Failed to fetch newera playlist: %v", err)
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
			return
		}

		// Log cache status
		if fromCache {
			if stale {
				log.Printf("Serving stale cache for newera playlist")
			} else {
				log.Printf("Serving fresh cache for newera playlist")
			}
		} else {
			log.Printf("Serving fresh content for newera playlist")
		}

		// Rewrite acestream:// URLs
		rewrittenContent := rw.RewriteM3U(content)

		// Set content type
		w.Header().Set("Content-Type", "audio/x-mpegurl")
		w.WriteHeader(http.StatusOK)
		w.Write(rewrittenContent)
	})

	// Unified playlist endpoint - merges all sources
	handler.HandleFunc("/playlist.m3u", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		elcanoURL := "https://ipfs.io/ipns/k51qzi5uqu5di462t7j4vu4akwfhvtjhy88qbupktvoacqfqe9uforjvhyi4wr/hashes_acestream.m3u"
		neweraURL := "https://ipfs.io/ipns/k2k4r8oqlcjxsritt5mczkcn4mmvcmymbqw7113fz2flkrerfwfps004/data/listas/lista_fuera_iptv.m3u"

		// Fetch both sources
		elcanoContent, elcanoFromCache, elcanoStale, elcanoErr := fetch.FetchWithCache(elcanoURL)
		neweraContent, neweraFromCache, neweraStale, neweraErr := fetch.FetchWithCache(neweraURL)

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

		// Apply deduplication by acestream ID (US-003)
		mergedBytes := []byte(mergedContent.String())
		deduplicatedContent := rewriter.DeduplicateStreams(mergedBytes)

		// Apply alphabetical sorting by display name (US-004)
		sortedContent := rewriter.SortStreamsByName(deduplicatedContent)

		// Rewrite acestream:// URLs and remove logos (US-005)
		rewrittenContent := rw.RewriteM3U(sortedContent)

		// Set content type
		w.Header().Set("Content-Type", "audio/x-mpegurl")
		w.WriteHeader(http.StatusOK)
		w.Write(rewrittenContent)
	})

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
