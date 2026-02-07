package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/alorle/iptv-manager/config"
	"github.com/alorle/iptv-manager/fetcher"
	"github.com/alorle/iptv-manager/logging"
	"github.com/alorle/iptv-manager/overrides"
	"github.com/alorle/iptv-manager/playlist"
	"github.com/alorle/iptv-manager/rewriter"
)

// PlaylistDependencies holds the dependencies needed by playlist handlers
type PlaylistDependencies struct {
	Logger       *logging.Logger
	Fetcher      fetcher.Interface
	OverridesMgr overrides.Interface
	Rewriter     rewriter.Interface
}

// CreateUnifiedPlaylistHandler creates an HTTP handler for the unified playlist endpoint
func CreateUnifiedPlaylistHandler(cfg *config.Config, deps PlaylistDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			logging.WriteJSONError(w, deps.Logger, "Method not allowed", http.StatusMethodNotAllowed, map[string]interface{}{
				"method": r.Method,
				"path":   r.URL.Path,
			})
			return
		}

		deps.Logger.Info("Unified playlist request", map[string]interface{}{
			"method": r.Method,
			"path":   r.URL.Path,
		})

		baseURL := GetBaseURL(r)

		// Fetch all configured sources
		sources := make([]playlist.Source, 0, len(cfg.Playlists))
		allFailed := true

		for _, pl := range cfg.Playlists {
			content, fromCache, stale, err := deps.Fetcher.FetchWithCache(pl.URL)
			sources = append(sources, playlist.Source{
				Name:      pl.Name,
				Content:   content,
				Err:       err,
				FromCache: fromCache,
				Stale:     stale,
			})
			if err == nil {
				allFailed = false
			}
		}

		// Check if all sources failed
		if allFailed {
			var errMsgs []string
			for _, src := range sources {
				if src.Err != nil {
					errMsgs = append(errMsgs, fmt.Sprintf("%s=%v", src.Name, src.Err))
				}
			}
			deps.Logger.Error("Failed to fetch unified playlist - all sources failed", map[string]interface{}{
				"errors": strings.Join(errMsgs, ", "),
			})
			logging.WriteJSONError(w, deps.Logger, "Bad Gateway", http.StatusBadGateway, map[string]interface{}{
				"path": r.URL.Path,
			})
			return
		}

		// Merge playlist sources
		mergedContent := playlist.MergeSources(deps.Logger, sources)

		// Clean up orphaned overrides
		playlist.CleanOrphanedOverrides(deps.Logger, deps.OverridesMgr, sources)

		// Process playlist through full pipeline
		rewrittenContent := playlist.Process(deps.OverridesMgr, deps.Rewriter, mergedContent, baseURL)

		// Send response
		w.Header().Set("Content-Type", "audio/x-mpegurl")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(rewrittenContent); err != nil {
			deps.Logger.Warn("Failed to write unified playlist response", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}
}

// CreatePlaylistHandler creates an HTTP handler for serving a single playlist source
func CreatePlaylistHandler(deps PlaylistDependencies, sourceURL string, sourceName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			logging.WriteJSONError(w, deps.Logger, "Method not allowed", http.StatusMethodNotAllowed, map[string]interface{}{
				"method": r.Method,
				"path":   r.URL.Path,
			})
			return
		}

		baseURL := GetBaseURL(r)

		// Fetch with cache fallback
		content, fromCache, stale, err := deps.Fetcher.FetchWithCache(sourceURL)
		if err != nil {
			deps.Logger.Error("Failed to fetch playlist", map[string]interface{}{
				"source": sourceName,
				"error":  err.Error(),
			})
			logging.WriteJSONError(w, deps.Logger, "Bad Gateway", http.StatusBadGateway, map[string]interface{}{
				"source": sourceName,
				"path":   r.URL.Path,
			})
			return
		}

		// Log cache status
		if fromCache {
			if stale {
				deps.Logger.Info("Serving stale cache for playlist", map[string]interface{}{
					"source": sourceName,
				})
			} else {
				deps.Logger.Info("Serving fresh cache for playlist", map[string]interface{}{
					"source": sourceName,
				})
			}
		} else {
			deps.Logger.Info("Serving fresh content for playlist", map[string]interface{}{
				"source": sourceName,
			})
		}

		// Apply channel overrides
		content = rewriter.ApplyOverrides(content, deps.OverridesMgr)

		// Rewrite acestream:// URLs
		rewrittenContent := deps.Rewriter.RewriteM3U(content, baseURL)

		// Set content type
		w.Header().Set("Content-Type", "audio/x-mpegurl")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(rewrittenContent); err != nil {
			deps.Logger.Warn("Failed to write playlist response", map[string]interface{}{
				"source": sourceName,
				"error":  err.Error(),
			})
		}
	}
}
