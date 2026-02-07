package handlers

import (
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/alorle/iptv-manager/cache"
	"github.com/alorle/iptv-manager/config"
	"github.com/alorle/iptv-manager/epg"
	"github.com/alorle/iptv-manager/fetcher"
	"github.com/alorle/iptv-manager/logging"
	"github.com/alorle/iptv-manager/multiplexer"
	"github.com/alorle/iptv-manager/overrides"
	"github.com/alorle/iptv-manager/pidmanager"
	"github.com/alorle/iptv-manager/rewriter"
)

// InitDependencies initializes all application components
func InitDependencies(cfg *config.Config) (Dependencies, error) {
	// Create resilience logger
	logLevel := logging.ParseLogLevel(cfg.Resilience.LogLevel)
	resLogger := logging.New(logLevel, "[resilience]")

	// Initialize cache storage
	storage, err := cache.NewFileStorage(cfg.Cache.Dir)
	if err != nil {
		return Dependencies{}, fmt.Errorf("failed to initialize cache storage: %w", err)
	}

	// Initialize overrides manager
	overridesPath := filepath.Join(cfg.Cache.Dir, "overrides.yaml")
	overridesMgr, err := overrides.NewManager(overridesPath)
	if err != nil {
		return Dependencies{}, fmt.Errorf("failed to initialize overrides manager: %w", err)
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

	return Dependencies{
		Fetcher:      fetcher.New(30*time.Second, storage, cfg.Cache.TTL),
		OverridesMgr: overridesMgr,
		EPGCache:     epgCache,
		Rewriter:     rewriter.New(),
		Multiplexer: multiplexer.New(multiplexer.Config{
			BufferSize:       cfg.Stream.BufferSize,
			ReadTimeout:      30 * time.Second,
			WriteTimeout:     10 * time.Second,
			ResilienceConfig: &cfg.Resilience,
			ResilienceLogger: resLogger,
		}),
		PidMgr: pidmanager.NewManager(),
	}, nil
}
