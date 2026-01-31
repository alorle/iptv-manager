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
)

// Config holds multiplexer configuration
type Config struct {
	// BufferSize is the size of the buffer for each client in bytes
	BufferSize int
	// ReadTimeout for reading from upstream
	ReadTimeout time.Duration
	// WriteTimeout for writing to clients
	WriteTimeout time.Duration
}

// DefaultConfig returns default multiplexer configuration
func DefaultConfig() Config {
	return Config{
		BufferSize:   1024 * 1024, // 1MB per client
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 10 * time.Second,
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
	ContentID   string
	Clients     map[string]*Client
	mu          sync.RWMutex
	cancel      context.CancelFunc
	upstream    io.ReadCloser
	done        chan struct{}
	started     bool
	startedOnce sync.Once
}

// NewStream creates a new stream
func NewStream(contentID string) *Stream {
	return &Stream{
		ContentID: contentID,
		Clients:   make(map[string]*Client),
		done:      make(chan struct{}),
	}
}

// AddClient adds a client to the stream
func (s *Stream) AddClient(client *Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Clients[client.ID] = client
	log.Printf("Stream %s: Added client %s (total clients: %d)", s.ContentID, client.ID, len(s.Clients))
}

// RemoveClient removes a client from the stream
func (s *Stream) RemoveClient(clientID string) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	if client, exists := s.Clients[clientID]; exists {
		client.Close()
		delete(s.Clients, clientID)
		log.Printf("Stream %s: Removed client %s (remaining clients: %d)", s.ContentID, clientID, len(s.Clients))
	}

	return len(s.Clients)
}

// ClientCount returns the number of connected clients
func (s *Stream) ClientCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.Clients)
}

// Start starts reading from upstream and fanning out to clients
func (s *Stream) Start(ctx context.Context, upstream io.ReadCloser, cfg Config) {
	s.startedOnce.Do(func() {
		s.upstream = upstream
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
			s.upstream.Close()
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

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Read from upstream
		n, err := s.upstream.Read(buffer)
		if err != nil {
			if err != io.EOF {
				log.Printf("Stream %s: Error reading from upstream: %v", s.ContentID, err)
			}
			return
		}

		if n > 0 {
			data := buffer[:n]

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
}

// New creates a new multiplexer
func New(cfg Config) *Multiplexer {
	return &Multiplexer{
		streams: make(map[string]*Stream),
		cfg:     cfg,
		client: &http.Client{
			Timeout: 0, // No timeout for streaming
		},
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

	// Create new stream
	stream := NewStream(contentID)

	// Start upstream connection
	req, err := http.NewRequestWithContext(ctx, "GET", upstreamURL, nil)
	if err != nil {
		return nil, false, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("failed to connect to upstream: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, false, fmt.Errorf("upstream returned status %d", resp.StatusCode)
	}

	log.Printf("Multiplexer: Created new stream for content ID %s", contentID)

	// Start the stream
	stream.Start(ctx, resp.Body, m.cfg)

	// Store the stream
	m.streams[contentID] = stream

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
		}
	}
}

// ServeStream handles a client connection for a stream
func (m *Multiplexer) ServeStream(ctx context.Context, w http.ResponseWriter, contentID string, upstreamURL string, clientID string) error {
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

	// Add client to stream
	stream.AddClient(client)

	// Set headers for streaming
	w.Header().Set("Content-Type", "video/mp2t")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	// Start client writer goroutine
	go func() {
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
