package multiplexer

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/alorle/iptv-manager/circuitbreaker"
	"github.com/alorle/iptv-manager/metrics"
)

// Multiplexer manages multiple streams
type Multiplexer struct {
	streams map[string]*Stream
	mu      sync.RWMutex
	cfg     Config
	client  *http.Client
	ctx     context.Context // Independent context for upstream connections
}

// updateGlobalMetrics updates global metrics based on all streams
// Must be called with lock held
func (m *Multiplexer) updateGlobalMetrics() {
	totalClients := 0
	for _, stream := range m.streams {
		totalClients += len(stream.Clients)
	}
	metrics.SetClientsConnected(totalClients)
}

// New creates a new multiplexer
func New(cfg Config) *Multiplexer {
	return &Multiplexer{
		streams: make(map[string]*Stream),
		cfg:     cfg,
		client: &http.Client{
			Timeout: 0, // No timeout for streaming
		},
		ctx: context.Background(), // Independent context for upstream connections
	}
}

// GetOrCreateStream gets an existing stream or creates a new one
func (m *Multiplexer) GetOrCreateStream(ctx context.Context, contentID string, upstreamURL string) (*Stream, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if stream already exists
	if stream, exists := m.streams[contentID]; exists {
		log.Printf("Multiplexer: Reusing existing stream for content ID %s", contentID)
		return stream, true, nil
	}

	// Create circuit breaker for this stream
	cbConfig := circuitbreaker.Config{
		FailureThreshold: m.cfg.ResilienceConfig.CBFailureThreshold,
		Timeout:          m.cfg.ResilienceConfig.CBTimeout,
		HalfOpenRequests: m.cfg.ResilienceConfig.CBHalfOpenRequests,
		Logger:           m.cfg.ResilienceLogger,
		ContentID:        contentID,
	}
	cb := circuitbreaker.New(cbConfig)

	// Create new stream with circuit breaker and ring buffer
	stream := NewStream(contentID, cb, m.client, m.cfg.ResilienceConfig.ReconnectBufferSize, m)

	// Start upstream connection using multiplexer's independent context
	// This ensures upstream is not tied to any single client's request context
	req, err := http.NewRequestWithContext(m.ctx, "GET", upstreamURL, nil)
	if err != nil {
		return nil, false, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("failed to connect to upstream: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Printf("Multiplexer: warning: failed to close response body: %v", closeErr)
		}
		return nil, false, fmt.Errorf("upstream returned status %d", resp.StatusCode)
	}

	log.Printf("Multiplexer: Created new stream for content ID %s", contentID)

	// Start the stream with multiplexer's independent context
	// This ensures the upstream reading loop is not canceled when a client disconnects
	stream.Start(m.ctx, resp.Body, upstreamURL, m.cfg)

	// Store the stream
	m.streams[contentID] = stream

	// Update metrics
	metrics.SetStreamsActive(len(m.streams))

	return stream, false, nil
}

// RemoveStream removes a stream if it has no clients
func (m *Multiplexer) RemoveStream(contentID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if stream, exists := m.streams[contentID]; exists {
		if stream.ClientCount() == 0 {
			log.Printf("Multiplexer: Removing stream %s (no clients)", contentID)
			stream.Stop()
			delete(m.streams, contentID)

			// Update metrics
			metrics.SetStreamsActive(len(m.streams))
		}
	}
}

// setupStreamingHeaders sets HTTP headers for streaming responses
func setupStreamingHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "video/mp2t")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
}

// startClientWriter starts a goroutine that writes data from the client buffer to the HTTP response
// Returns a channel that signals when the writer goroutine is ready
func startClientWriter(ctx context.Context, client *Client, contentID, clientID string) <-chan struct{} {
	ready := make(chan struct{})

	go func() {
		close(ready) // Signal that the writer goroutine has started
		for {
			select {
			case data, ok := <-client.buffer:
				if !ok {
					return
				}

				// Write data to client
				if _, err := client.Writer.Write(data); err != nil {
					log.Printf("Stream %s: Client %s write error: %v", contentID, clientID, err)
					client.Close()
					return
				}

				// Flush immediately for streaming
				client.Flusher.Flush()
				client.lastSend = time.Now()

			case <-client.Done:
				return

			case <-ctx.Done():
				return
			}
		}
	}()

	return ready
}

// sendBufferedDataIfReconnecting sends buffered data to a new client if the stream is reconnecting
func sendBufferedDataIfReconnecting(stream *Stream, client *Client, contentID, clientID string) {
	if !stream.IsReconnecting() {
		return
	}

	log.Printf("Stream %s: New client %s joining during reconnection - sending buffered data", contentID, clientID)
	if !stream.sendBufferToClient(client, contentID) {
		log.Printf("Stream %s: Failed to send buffer to new client %s", contentID, clientID)
	}
}

// waitForClientDisconnect blocks until the client disconnects or context is canceled
func waitForClientDisconnect(ctx context.Context, client *Client) {
	select {
	case <-client.Done:
	case <-ctx.Done():
	}
}

// cleanupClient removes the client from the stream and cleans up the stream if no clients remain
func (m *Multiplexer) cleanupClient(stream *Stream, contentID, clientID string) {
	remainingClients := stream.RemoveClient(clientID)
	if remainingClients == 0 {
		m.RemoveStream(contentID)
	}
}

// ServeStream handles an HTTP request for streaming content to a client.
// It manages the client lifecycle, connecting them to a shared stream for the given content ID.
// Multiple clients requesting the same content will share a single upstream connection via multiplexing.
func (m *Multiplexer) ServeStream(ctx context.Context, w http.ResponseWriter, contentID string, upstreamURL string, clientID string) error {
	// Disable write deadline for streaming - this is essential for long-running video streams
	rc := http.NewResponseController(w)
	if err := rc.SetWriteDeadline(time.Time{}); err != nil {
		log.Printf("Warning: Failed to disable write deadline: %v", err)
	}

	// Get or create stream
	stream, existed, err := m.GetOrCreateStream(ctx, contentID, upstreamURL)
	if err != nil {
		return fmt.Errorf("failed to get stream: %w", err)
	}

	// Create client
	client, err := NewClient(clientID, w, m.cfg.BufferSize)
	if err != nil {
		if !existed {
			m.RemoveStream(contentID)
		}
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Set up HTTP streaming response
	setupStreamingHeaders(w)

	// Start client writer goroutine and wait for it to be ready
	writerReady := startClientWriter(ctx, client, contentID, clientID)
	<-writerReady

	// Add client to stream now that it's ready
	stream.AddClient(client)

	// Send buffered data if stream is reconnecting
	sendBufferedDataIfReconnecting(stream, client, contentID, clientID)

	// Wait for client to disconnect
	waitForClientDisconnect(ctx, client)

	// Cleanup
	m.cleanupClient(stream, contentID, clientID)

	return nil
}

// Stats returns multiplexer statistics
func (m *Multiplexer) Stats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["total_streams"] = len(m.streams)

	streams := make([]map[string]interface{}, 0, len(m.streams))
	for contentID, stream := range m.streams {
		streams = append(streams, map[string]interface{}{
			"content_id":   contentID,
			"client_count": stream.ClientCount(),
		})
	}
	stats["streams"] = streams

	return stats
}
