package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/alorle/iptv-manager/config"
	"github.com/alorle/iptv-manager/domain"
	"github.com/alorle/iptv-manager/multiplexer"
	"github.com/alorle/iptv-manager/pidmanager"
)

// StreamDependencies holds the dependencies needed by stream handlers
type StreamDependencies struct {
	Multiplexer *multiplexer.Multiplexer
	PidMgr      *pidmanager.Manager
}

// CreateStreamHandler creates the HTTP handler for streaming endpoints
func CreateStreamHandler(cfg *config.Config, deps StreamDependencies) http.HandlerFunc {
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
		if !domain.IsValidAcestreamID(contentID) {
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
		pid := deps.PidMgr.GetOrCreatePID(contentID, clientInfo)
		clientID := fmt.Sprintf("%s-%d", clientInfo.IP, pid)

		log.Printf("Stream request: contentID=%s, clientID=%s, pid=%d", contentID, clientID, pid)

		// Build upstream URL with PID and optional transcode parameters
		upstreamURL := fmt.Sprintf("%s/ace/getstream?id=%s&pid=%d", cfg.Acestream.EngineURL, contentID, pid)

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
		if err := deps.Multiplexer.ServeStream(r.Context(), w, contentID, upstreamURL, clientID); err != nil {
			log.Printf("Failed to serve stream for contentID=%s: %v", contentID, err)
			// Check if it's a connection error to Engine
			if strings.Contains(err.Error(), "connect") || strings.Contains(err.Error(), "upstream") {
				http.Error(w, "Bad Gateway: cannot connect to Engine", http.StatusBadGateway)
				return
			}
		}

		// Release PID when client disconnects
		if err := deps.PidMgr.ReleasePID(pid); err != nil {
			log.Printf("Failed to release PID %d: %v", pid, err)
		}

		// Cleanup disconnected sessions periodically
		if cleaned := deps.PidMgr.CleanupDisconnected(); cleaned > 0 {
			log.Printf("Cleaned up %d disconnected sessions", cleaned)
		}
	}
}
