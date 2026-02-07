package multiplexer

import (
	"context"
	"errors"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/alorle/iptv-manager/circuitbreaker"
	"github.com/alorle/iptv-manager/config"
	"github.com/alorle/iptv-manager/logging"
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
	clientCount := len(s.Clients)
	s.mu.Unlock()

	if s.resLogger != nil {
		s.resLogger.Info("Added client to stream", map[string]interface{}{
			"content_id":   s.ContentID,
			"client_id":    client.ID,
			"total_clients": clientCount,
		})
	}

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
		remaining := len(s.Clients)
		s.mu.Unlock()

		if s.resLogger != nil {
			s.resLogger.Info("Removed client from stream", map[string]interface{}{
				"content_id":        s.ContentID,
				"client_id":         clientID,
				"remaining_clients": remaining,
			})
		}
	} else {
		s.mu.Unlock()
	}

	s.mu.RLock()
	remaining := len(s.Clients)
	s.mu.RUnlock()

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
			if s.resLogger != nil {
				s.resLogger.Warn("Failed to send buffered data to client", map[string]interface{}{
					"content_id": contentID,
					"client_id":  client.ID,
					"error":      err.Error(),
				})
			}
			return false
		}
	}

	if s.resLogger != nil {
		s.resLogger.Info("Sent buffered data to new client", map[string]interface{}{
			"content_id": contentID,
			"client_id":  client.ID,
			"bytes":      len(bufferedData),
		})
	}
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

// distributeData sends data to all connected clients concurrently
func (s *Stream) distributeData(data []byte) {
	// Write to ring buffer for resilience
	s.ringBuffer.Write(data)

	// Get snapshot of clients
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
				// Client send errors are expected (slow clients, disconnects)
				// Only log at debug level to avoid spam
				c.Close()
			}
		}(client)
	}
	wg.Wait()
}

// Stop stops the stream
func (s *Stream) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	<-s.done
}
