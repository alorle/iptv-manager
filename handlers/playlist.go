package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/alorle/iptv-manager/config"
	"github.com/alorle/iptv-manager/fetcher"
	"github.com/alorle/iptv-manager/overrides"
	"github.com/alorle/iptv-manager/playlist"
	"github.com/alorle/iptv-manager/rewriter"
)

// PlaylistDependencies holds the dependencies needed by playlist handlers
type PlaylistDependencies struct {
	Fetcher      fetcher.Interface
	OverridesMgr overrides.Interface
	Rewriter     rewriter.Interface
}

// CreateUnifiedPlaylistHandler creates an HTTP handler for the unified playlist endpoint
func CreateUnifiedPlaylistHandler(cfg *config.Config, deps PlaylistDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

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
			log.Printf("Failed to fetch unified playlist - all sources failed: %s", strings.Join(errMsgs, ", "))
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
			return
		}

		// Merge playlist sources
		mergedContent := playlist.MergeSources(sources)

		// Clean up orphaned overrides
		playlist.CleanOrphanedOverrides(deps.OverridesMgr, sources)

		// Process playlist through full pipeline
		rewrittenContent := playlist.Process(deps.OverridesMgr, deps.Rewriter, mergedContent, baseURL)

		// Send response
		w.Header().Set("Content-Type", "audio/x-mpegurl")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(rewrittenContent); err != nil {
			log.Printf("Error writing unified playlist: %v", err)
		}
	}
}

// CreatePlaylistHandler creates an HTTP handler for serving a single playlist source
func CreatePlaylistHandler(deps PlaylistDependencies, sourceURL string, sourceName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		baseURL := GetBaseURL(r)

		// Fetch with cache fallback
		content, fromCache, stale, err := deps.Fetcher.FetchWithCache(sourceURL)
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
		content = rewriter.ApplyOverrides(content, deps.OverridesMgr)

		// Rewrite acestream:// URLs
		rewrittenContent := deps.Rewriter.RewriteM3U(content, baseURL)

		// Set content type
		w.Header().Set("Content-Type", "audio/x-mpegurl")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(rewrittenContent); err != nil {
			log.Printf("Error writing %s playlist: %v", sourceName, err)
		}
	}
}
