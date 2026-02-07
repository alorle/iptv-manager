package handlers

import (
	"fmt"
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
	// Create application logger (INFO level by default, or use resilience log level)
	logLevel := logging.ParseLogLevel(cfg.Resilience.LogLevel)
	appLogger := logging.New(logLevel, "[app]")

	// Create resilience logger
	resLogger := logging.New(logLevel, "[resilience]")

	// Initialize cache storage
	storage, err := cache.NewFileStorage(cfg.Cache.Dir)
	if err != nil {
		return Dependencies{}, fmt.Errorf("failed to initialize cache storage: %w", err)
	}

	// Initialize overrides manager
	overridesPath := filepath.Join(cfg.Cache.Dir, "overrides.yaml")
	overridesMgr, err := overrides.NewManager(overridesPath, appLogger)
	if err != nil {
		return Dependencies{}, fmt.Errorf("failed to initialize overrides manager: %w", err)
	}
	appLogger.Info("Loaded channel overrides", map[string]interface{}{
		"count": len(overridesMgr.List()),
		"path":  overridesPath,
	})

	// Initialize EPG cache
	epgCache, err := epg.New(cfg.EPG.URL, 30*time.Second, appLogger)
	if err != nil {
		appLogger.Warn("Failed to initialize EPG cache", map[string]interface{}{
			"error": err.Error(),
		})
		appLogger.Warn("TVG-ID validation will not be available", nil)
	} else {
		appLogger.Info("EPG cache initialized", map[string]interface{}{
			"channels": epgCache.Count(),
		})
	}

	return Dependencies{
		Logger:       appLogger,
		Fetcher:      fetcher.New(30*time.Second, storage, cfg.Cache.TTL, appLogger),
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
