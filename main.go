package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/alorle/iptv-manager/config"
	"github.com/alorle/iptv-manager/handlers"
	"github.com/alorle/iptv-manager/metrics"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// Print configuration
	cfg.Print()

	// Initialize dependencies
	deps, err := handlers.InitDependencies(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize dependencies: %v", err)
	}

	// Initialize metrics
	metrics.SetStreamsActive(0)
	metrics.SetClientsConnected(0)

	// Setup routes
	handler := handlers.SetupRoutes(cfg, deps)

	// Start server
	s := &http.Server{
		Handler:      handler,
		Addr:         fmt.Sprintf("%s:%s", cfg.HTTP.Address, cfg.HTTP.Port),
		ReadTimeout:  cfg.Proxy.ReadTimeout,
		WriteTimeout: cfg.Proxy.WriteTimeout,
		ErrorLog:     log.Default(),
	}

	if err := s.ListenAndServe(); err != nil {
		fmt.Printf("Error starting server: %v\n", err)
	}
}
