package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
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

	return config{
		Port:               port,
		AceStreamEngineURL: aceStreamURL,
		DBPath:             dbPath,
	}
}

func main() {
	cfg := loadConfig()

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

	aceStreamEngine := driven.NewAceStreamHTTPAdapter(cfg.AceStreamEngineURL)

	// Create application services
	channelService := application.NewChannelService(channelRepo, streamRepo)
	streamService := application.NewStreamService(streamRepo, channelRepo)
	playlistService := application.NewPlaylistService(streamRepo)
	healthService := application.NewHealthService(channelRepo, aceStreamEngine)
	aceStreamProxyService := application.NewAceStreamProxyService(aceStreamEngine)

	// Create HTTP handlers
	channelHandler := driver.NewChannelHTTPHandler(channelService)
	streamHandler := driver.NewStreamHTTPHandler(streamService)
	playlistHandler := driver.NewPlaylistHTTPHandler(playlistService)
	healthHandler := driver.NewHealthHTTPHandler(healthService)
	aceStreamHandler := driver.NewAceStreamHTTPHandler(aceStreamProxyService)

	// Register routes
	mux := http.NewServeMux()
	mux.Handle("/channels", channelHandler)
	mux.Handle("/channels/", channelHandler)
	mux.Handle("/streams", streamHandler)
	mux.Handle("/streams/", streamHandler)
	mux.Handle("/playlist.m3u", playlistHandler)
	mux.Handle("/ace/", aceStreamHandler)
	mux.Handle("/health", healthHandler)

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting IPTV Manager on port %s", cfg.Port)
		log.Printf("AceStream Engine URL: %s", cfg.AceStreamEngineURL)
		log.Printf("Database: %s", cfg.DBPath)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}
