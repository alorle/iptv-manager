package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/alorle/iptv-manager/internal/adapter/driven"
	"github.com/alorle/iptv-manager/internal/adapter/driver"
	"github.com/alorle/iptv-manager/internal/application"
	"go.etcd.io/bbolt"
)

type config struct {
	Port               string
	AceStreamEngineURL string
	DBPath             string
	LogLevel           slog.Level
	StreamWriteTimeout time.Duration
}

func loadConfig() config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	aceStreamURL := os.Getenv("ACESTREAM_ENGINE_URL")
	if aceStreamURL == "" {
		aceStreamURL = "http://localhost:6878"
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "iptv-manager.db"
	}

	logLevel := slog.LevelInfo
	if logLevelStr := os.Getenv("LOG_LEVEL"); logLevelStr != "" {
		switch strings.ToUpper(logLevelStr) {
		case "DEBUG":
			logLevel = slog.LevelDebug
		case "INFO":
			logLevel = slog.LevelInfo
		case "WARN":
			logLevel = slog.LevelWarn
		case "ERROR":
			logLevel = slog.LevelError
		}
	}

	streamWriteTimeout := 10 * time.Second
	if timeoutStr := os.Getenv("STREAM_WRITE_TIMEOUT"); timeoutStr != "" {
		if parsedTimeout, err := time.ParseDuration(timeoutStr); err == nil {
			streamWriteTimeout = parsedTimeout
		}
	}

	return config{
		Port:               port,
		AceStreamEngineURL: aceStreamURL,
		DBPath:             dbPath,
		LogLevel:           logLevel,
		StreamWriteTimeout: streamWriteTimeout,
	}
}

func main() {
	cfg := loadConfig()

	// Create structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	}))
	slog.SetDefault(logger)

	logger.Info("starting iptv-manager",
		"port", cfg.Port,
		"acestream_url", cfg.AceStreamEngineURL,
		"db_path", cfg.DBPath,
		"log_level", cfg.LogLevel.String(),
		"stream_write_timeout", cfg.StreamWriteTimeout,
	)

	// Open BoltDB
	db, err := bbolt.Open(cfg.DBPath, 0600, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("error closing database: %v", err)
		}
	}()

	// Create driven adapters (repositories and external services)
	channelRepo, err := driven.NewChannelBoltDBRepository(db)
	if err != nil {
		log.Fatalf("failed to create channel repository: %v", err)
	}

	streamRepo, err := driven.NewStreamBoltDBRepository(db)
	if err != nil {
		log.Fatalf("failed to create stream repository: %v", err)
	}

	aceStreamEngine := driven.NewAceStreamHTTPAdapter(cfg.AceStreamEngineURL, logger)

	// Create application services
	channelService := application.NewChannelService(channelRepo, streamRepo)
	streamService := application.NewStreamService(streamRepo, channelRepo)
	playlistService := application.NewPlaylistService(streamRepo)
	healthService := application.NewHealthService(channelRepo, aceStreamEngine)
	aceStreamProxyService := application.NewAceStreamProxyService(aceStreamEngine, logger, cfg.StreamWriteTimeout)

	// Create HTTP handlers
	channelHandler := driver.NewChannelHTTPHandler(channelService)
	streamHandler := driver.NewStreamHTTPHandler(streamService)
	playlistHandler := driver.NewPlaylistHTTPHandler(playlistService)
	healthHandler := driver.NewHealthHTTPHandler(healthService)
	aceStreamHandler := driver.NewAceStreamHTTPHandler(aceStreamProxyService, logger)

	// Register API routes
	apiMux := http.NewServeMux()
	apiMux.Handle("/channels", channelHandler)
	apiMux.Handle("/channels/", channelHandler)
	apiMux.Handle("/streams", streamHandler)
	apiMux.Handle("/streams/", streamHandler)
	apiMux.Handle("/health", healthHandler)

	// Root router: API under /api/, streaming routes at root, SPA for everything else
	rootMux := http.NewServeMux()
	rootMux.Handle("/api/", http.StripPrefix("/api", apiMux))
	rootMux.Handle("/playlist.m3u", playlistHandler)
	rootMux.Handle("/ace/", aceStreamHandler)
	rootMux.Handle("/", newSPAHandler())

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      rootMux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("http server listening", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("shutdown signal received, shutting down gracefully")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("server shutdown error", "error", err)
	}

	logger.Info("server stopped")
}
