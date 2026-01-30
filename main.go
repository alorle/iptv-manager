package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/alorle/iptv-manager/cache"
	"github.com/alorle/iptv-manager/fetcher"
	"github.com/alorle/iptv-manager/rewriter"
)

type Config struct {
	HTTPAddress            string
	HTTPPort               string
	AcestreamPlayerBaseURL string
	CacheDir               string
	CacheTTL               time.Duration
}

func loadConfig() (*Config, error) {
	cfg := &Config{
		HTTPAddress:            os.Getenv("HTTP_ADDRESS"),
		HTTPPort:               os.Getenv("HTTP_PORT"),
		AcestreamPlayerBaseURL: os.Getenv("ACESTREAM_PLAYER_BASE_URL"),
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
	fmt.Printf("cacheDir: %v\n", cfg.CacheDir)
	fmt.Printf("cacheTTL: %v\n", cfg.CacheTTL)

	// Initialize cache storage
	storage, err := cache.NewFileStorage(cfg.CacheDir)
	if err != nil {
		log.Fatalf("Failed to initialize cache storage: %v", err)
	}

	// Initialize fetcher with 30 second timeout
	fetch := fetcher.New(30*time.Second, storage, cfg.CacheTTL)

	// Initialize rewriter
	rw := rewriter.New(cfg.AcestreamPlayerBaseURL)

	handler := http.NewServeMux()

	handler.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
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

	s := &http.Server{
		Handler: handler,
		Addr:    fmt.Sprintf("%s:%s", cfg.HTTPAddress, cfg.HTTPPort),
	}

	if err := s.ListenAndServe(); err != nil {
		fmt.Printf("Error starting server: %v\n", err)
		return
	}
}
