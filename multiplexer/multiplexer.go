package multiplexer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/alorle/iptv-manager/circuitbreaker"
	"github.com/alorle/iptv-manager/config"
	"github.com/alorle/iptv-manager/logging"
	"github.com/alorle/iptv-manager/metrics"
)

// Config holds multiplexer configuration
type Config struct {
	// BufferSize is the size of the buffer for each client in bytes
	BufferSize int
	// ReadTimeout for reading from upstream
	ReadTimeout time.Duration
	// WriteTimeout for writing to clients
	WriteTimeout time.Duration
	// ResilienceConfig holds reconnection and circuit breaker settings
	ResilienceConfig *config.ResilienceConfig
	// ResilienceLogger for resilience events
	ResilienceLogger *logging.Logger
}

// DefaultConfig returns default multiplexer configuration
func DefaultConfig() Config {
	return Config{
		BufferSize:       1024 * 1024, // 1MB per client
		ReadTimeout:      30 * time.Second,
		WriteTimeout:     10 * time.Second,
		ResilienceConfig: config.DefaultResilienceConfig(),
	}
}

// Client represents a connected client receiving a stream
type Client struct {
	ID       string
	Writer   http.ResponseWriter
	Flusher  http.Flusher
	Done     chan struct{}
	Err      error
	mu       sync.Mutex
	closed   bool
	buffer   chan []byte
	lastSend time.Time
}

// NewClient creates a new client
func NewClient(id string, w http.ResponseWriter, bufferSize int) (*Client, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, errors.New("streaming not supported")
	}

	return &Client{
		ID:       id,
		Writer:   w,
		Flusher:  flusher,
		Done:     make(chan struct{}),
		buffer:   make(chan []byte, bufferSize/(4*1024)), // Assume ~4KB chunks
		lastSend: time.Now(),
	}, nil
}

// Send sends data to the client through the buffer
func (c *Client) Send(data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return errors.New("client closed")
	}

	// Make a copy of the data
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)

	select {
	case c.buffer <- dataCopy:
		return nil
	default:
		// Buffer full - client too slow
		return errors.New("client buffer full")
	}
}

// Close closes the client
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.closed {
		c.closed = true
		close(c.Done)
		close(c.buffer)
	}
}

// Stream represents an active stream from upstream
type Stream struct {
	ContentID      string
	Clients        map[string]*Client
	mu             sync.RWMutex
	cancel         context.CancelFunc
	upstream       io.ReadCloser
	done           chan struct{}
	started        bool
	startedOnce    sync.Once
	upstreamURL    string
	circuitBreaker circuitbreaker.CircuitBreaker
	httpClient     *http.Client
	ringBuffer     *RingBuffer
	reconnecting   bool
	reconnectMu    sync.RWMutex
	resLogger      *logging.Logger
	reconnectStart time.Time
	mux            *Multiplexer
}

// NewStream creates a new stream with ring buffer
func NewStream(contentID string, cb circuitbreaker.CircuitBreaker, httpClient *http.Client, bufferSize int, mux *Multiplexer) *Stream {
	return &Stream{
		ContentID:      contentID,
		Clients:        make(map[string]*Client),
		done:           make(chan struct{}),
		circuitBreaker: cb,
		httpClient:     httpClient,
		ringBuffer:     NewRingBuffer(bufferSize),
		reconnecting:   false,
		mux:            mux,
	}
}

// AddClient adds a client to the stream
func (s *Stream) AddClient(client *Client) {
	s.mu.Lock()
	s.Clients[client.ID] = client
	log.Printf("Stream %s: Added client %s (total clients: %d)", s.ContentID, client.ID, len(s.Clients))
	s.mu.Unlock()

	// Update metrics through multiplexer
	if s.mux != nil {
		s.mux.mu.RLock()
		s.mux.updateGlobalMetrics()
		s.mux.mu.RUnlock()
	}
}

// RemoveClient removes a client from the stream
func (s *Stream) RemoveClient(clientID string) int {
	s.mu.Lock()
	if client, exists := s.Clients[clientID]; exists {
		client.Close()
		delete(s.Clients, clientID)
		log.Printf("Stream %s: Removed client %s (remaining clients: %d)", s.ContentID, clientID, len(s.Clients))
	}
	remaining := len(s.Clients)
	s.mu.Unlock()

	// Update metrics through multiplexer
	if s.mux != nil {
		s.mux.mu.RLock()
		s.mux.updateGlobalMetrics()
		s.mux.mu.RUnlock()
	}

	return remaining
}

// ClientCount returns the number of connected clients
func (s *Stream) ClientCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.Clients)
}

// IsReconnecting returns true if the stream is currently reconnecting
func (s *Stream) IsReconnecting() bool {
	s.reconnectMu.RLock()
	defer s.reconnectMu.RUnlock()
	return s.reconnecting
}

// setReconnecting sets the reconnection state
func (s *Stream) setReconnecting(state bool) {
	s.reconnectMu.Lock()
	defer s.reconnectMu.Unlock()
	s.reconnecting = state
}

// sendBufferToClient sends buffered data to a client
// Returns true if successful, false if client should be closed
func (s *Stream) sendBufferToClient(client *Client, contentID string) bool {
	bufferedData := s.ringBuffer.PeekAll()
	if len(bufferedData) == 0 {
		return true // No data to send, but client is still valid
	}

	// Send buffered data in chunks to avoid blocking
	chunkSize := 32 * 1024 // 32KB chunks
	for i := 0; i < len(bufferedData); i += chunkSize {
		end := i + chunkSize
		if end > len(bufferedData) {
			end = len(bufferedData)
		}
		chunk := bufferedData[i:end]

		if err := client.Send(chunk); err != nil {
			log.Printf("Stream %s: Failed to send buffered data to client %s: %v", contentID, client.ID, err)
			return false
		}
	}

	log.Printf("Stream %s: Sent %d bytes of buffered data to new client %s", contentID, len(bufferedData), client.ID)
	return true
}

// Start starts reading from upstream and fanning out to clients
func (s *Stream) Start(ctx context.Context, upstream io.ReadCloser, upstreamURL string, cfg Config) {
	s.startedOnce.Do(func() {
		s.upstream = upstream
		s.upstreamURL = upstreamURL
		s.resLogger = cfg.ResilienceLogger
		ctx, cancel := context.WithCancel(ctx)
		s.cancel = cancel
		s.started = true

		// Start fan-out goroutine
		go s.fanOut(ctx, cfg)
	})
}

// fanOut reads from upstream and sends to all clients
func (s *Stream) fanOut(ctx context.Context, cfg Config) {
	defer func() {
		s.mu.Lock()
		if s.upstream != nil {
			if closeErr := s.upstream.Close(); closeErr != nil {
				log.Printf("Stream %s: warning: failed to close upstream: %v", s.ContentID, closeErr)
			}
		}
		// Close all clients
		for _, client := range s.Clients {
			client.Close()
		}
		s.mu.Unlock()
		close(s.done)
		log.Printf("Stream %s: Fan-out stopped", s.ContentID)
	}()

	buffer := make([]byte, 32*1024) // 32KB read buffer
	attemptNumber := 0

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Read from upstream
		n, err := s.upstream.Read(buffer)
		if err != nil {
			// Check if we should reconnect
			shouldReconnect := err != io.EOF && ctx.Err() == nil && s.ClientCount() > 0

			if !shouldReconnect {
				if err != io.EOF {
					log.Printf("Stream %s: Error reading from upstream: %v", s.ContentID, err)
				}
				return
			}

			// Attempt reconnection with exponential backoff
			log.Printf("Stream %s: Upstream connection lost: %v", s.ContentID, err)

			// Record upstream error
			metrics.RecordUpstreamError(s.ContentID, "connection_lost")

			// Mark as reconnecting
			s.setReconnecting(true)
			s.reconnectStart = time.Now()
			log.Printf("Stream %s: Entering reconnection mode - clients will use buffer", s.ContentID)

			// Close current upstream connection
			if s.upstream != nil {
				if closeErr := s.upstream.Close(); closeErr != nil {
					log.Printf("Stream %s: warning: failed to close upstream: %v", s.ContentID, closeErr)
				}
				s.upstream = nil
			}

			// Reconnection loop
			backoff := cfg.ResilienceConfig.ReconnectInitialBackoff
			attemptNumber = 1

			for {
				// Check if we should stop reconnecting
				if ctx.Err() != nil || s.ClientCount() == 0 {
					reason := "context cancelled"
					if s.ClientCount() == 0 {
						reason = "no clients remaining"
					}
					log.Printf("Stream %s: Stopping reconnection - no clients or context cancelled", s.ContentID)
					if s.resLogger != nil {
						s.resLogger.LogReconnectFailed(s.ContentID, reason, attemptNumber)
					}
					s.setReconnecting(false)
					return
				}

				// Check circuit breaker state before attempting reconnection
				cbState := s.circuitBreaker.State()
				if cbState == circuitbreaker.StateOpen {
					log.Printf("Stream %s: Circuit breaker is OPEN, skipping reconnection attempt %d", s.ContentID, attemptNumber)

					// Wait for circuit breaker timeout before checking again
					select {
					case <-time.After(cfg.ResilienceConfig.CBTimeout):
						continue
					case <-ctx.Done():
						s.setReconnecting(false)
						return
					}
				}

				// Log reconnection attempt
				log.Printf("Stream %s: Reconnection attempt #%d (backoff: %v, buffer available: %d bytes)",
					s.ContentID, attemptNumber, backoff, s.ringBuffer.Available())
				if s.resLogger != nil {
					s.resLogger.LogReconnectAttempt(s.ContentID, attemptNumber, backoff)
				}

				// Wait for backoff duration
				select {
				case <-time.After(backoff):
				case <-ctx.Done():
					s.setReconnecting(false)
					return
				}

				// Attempt to reconnect through circuit breaker
				var newUpstream io.ReadCloser
				reconnectErr := s.circuitBreaker.Execute(func() error {
					req, err := http.NewRequestWithContext(ctx, "GET", s.upstreamURL, nil)
					if err != nil {
						return fmt.Errorf("failed to create request: %w", err)
					}

					resp, err := s.httpClient.Do(req)
					if err != nil {
						return fmt.Errorf("failed to connect to upstream: %w", err)
					}

					if resp.StatusCode != http.StatusOK {
						if closeErr := resp.Body.Close(); closeErr != nil {
							log.Printf("Stream %s: warning: failed to close response body: %v", s.ContentID, closeErr)
						}
						return fmt.Errorf("upstream returned status %d", resp.StatusCode)
					}

					newUpstream = resp.Body
					return nil
				})

				if reconnectErr != nil {
					log.Printf("Stream %s: Reconnection attempt #%d failed: %v", s.ContentID, attemptNumber, reconnectErr)

					// Calculate next backoff (exponential)
					backoff = backoff * 2
					if backoff > cfg.ResilienceConfig.ReconnectMaxBackoff {
						backoff = cfg.ResilienceConfig.ReconnectMaxBackoff
					}

					attemptNumber++
					continue
				}

				// Reconnection successful
				downtime := time.Since(s.reconnectStart)
				log.Printf("Stream %s: Reconnection attempt #%d succeeded - resuming normal streaming", s.ContentID, attemptNumber)
				if s.resLogger != nil {
					s.resLogger.LogReconnectSuccess(s.ContentID, downtime)
				}

				// Record reconnection metric
				metrics.RecordUpstreamReconnection(s.ContentID)

				s.mu.Lock()
				s.upstream = newUpstream
				s.mu.Unlock()

				// Mark as no longer reconnecting
				s.setReconnecting(false)

				// Reset attempt counter and break out of reconnection loop
				attemptNumber = 0
				break
			}
		}

		if n > 0 {
			data := buffer[:n]

			// Write to ring buffer for resilience
			s.ringBuffer.Write(data)

			// Send to all clients
			s.mu.RLock()
			clients := make([]*Client, 0, len(s.Clients))
			for _, client := range s.Clients {
				clients = append(clients, client)
			}
			s.mu.RUnlock()

			// Send to each client concurrently
			var wg sync.WaitGroup
			for _, client := range clients {
				wg.Add(1)
				go func(c *Client) {
					defer wg.Done()
					if err := c.Send(data); err != nil {
						log.Printf("Stream %s: Failed to send to client %s: %v", s.ContentID, c.ID, err)
						c.Close()
					}
				}(client)
			}
			wg.Wait()
		}
	}
}

// Stop stops the stream
func (s *Stream) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	<-s.done
}

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
	// This ensures the upstream reading loop is not cancelled when a client disconnects
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

// ServeStream handles a client connection for a stream
func (m *Multiplexer) ServeStream(ctx context.Context, w http.ResponseWriter, contentID string, upstreamURL string, clientID string) error {
	// Disable write deadline for streaming - this is essential for long-running video streams
	// The http.Server.WriteTimeout would otherwise kill the connection
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

	// Set headers for streaming BEFORE adding client to stream
	// This ensures we're ready to receive data
	w.Header().Set("Content-Type", "video/mp2t")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	// Start client writer goroutine BEFORE adding client to stream
	// This ensures the writer is ready to process data when it arrives
	writerReady := make(chan struct{})
	go func() {
		close(writerReady) // Signal that the writer goroutine has started
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

	// Wait for writer goroutine to be ready
	<-writerReady

	// Now add client to stream - it's ready to receive data
	stream.AddClient(client)

	// If stream is reconnecting, send buffered data to new client
	if stream.IsReconnecting() {
		log.Printf("Stream %s: New client %s joining during reconnection - sending buffered data", contentID, clientID)
		if !stream.sendBufferToClient(client, contentID) {
			// Failed to send buffer, client will be removed below
			log.Printf("Stream %s: Failed to send buffer to new client %s", contentID, clientID)
		}
	}

	// Wait for client to disconnect
	select {
	case <-client.Done:
	case <-ctx.Done():
	}

	// Remove client from stream
	remainingClients := stream.RemoveClient(clientID)

	// If no clients left, remove the stream
	if remainingClients == 0 {
		m.RemoveStream(contentID)
	}

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
