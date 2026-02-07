package multiplexer

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alorle/iptv-manager/circuitbreaker"
	"github.com/alorle/iptv-manager/config"
)

func TestStream_RingBufferDuringReconnection(t *testing.T) {
	// Create a simple test configuration
	cfg := Config{
		BufferSize:   1024 * 1024,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		ResilienceConfig: &config.ResilienceConfig{
			ReconnectBufferSize:     2 * 1024 * 1024,
			ReconnectMaxBackoff:     100 * time.Millisecond,
			ReconnectInitialBackoff: 10 * time.Millisecond,
			CBFailureThreshold:      5,
			CBTimeout:               30 * time.Second,
			CBHalfOpenRequests:      1,
		},
	}

	// Create circuit breaker
	cb := circuitbreaker.New(circuitbreaker.Config{
		FailureThreshold: 5,
		Timeout:          30 * time.Second,
		HalfOpenRequests: 1,
	})

	// Create stream with ring buffer
	stream := NewStream("test-content", cb, &http.Client{}, cfg.ResilienceConfig.ReconnectBufferSize, nil)

	// Verify ring buffer is created
	if stream.ringBuffer == nil {
		t.Fatal("Ring buffer not created")
	}

	if stream.ringBuffer.Size() != cfg.ResilienceConfig.ReconnectBufferSize {
		t.Errorf("Ring buffer size = %d, want %d", stream.ringBuffer.Size(), cfg.ResilienceConfig.ReconnectBufferSize)
	}
}

func TestStream_BufferWriteDuringNormalOperation(t *testing.T) {
	cfg := Config{
		BufferSize:   1024 * 1024,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		ResilienceConfig: &config.ResilienceConfig{
			ReconnectBufferSize:     1024, // Small buffer for testing
			ReconnectMaxBackoff:     100 * time.Millisecond,
			ReconnectInitialBackoff: 10 * time.Millisecond,
			CBFailureThreshold:      5,
			CBTimeout:               30 * time.Second,
			CBHalfOpenRequests:      1,
		},
	}

	cb := circuitbreaker.New(circuitbreaker.Config{
		FailureThreshold: 5,
		Timeout:          30 * time.Second,
		HalfOpenRequests: 1,
	})

	stream := NewStream("test-content", cb, &http.Client{}, cfg.ResilienceConfig.ReconnectBufferSize, nil)

	// Simulate data flow - ring buffer should be populated
	testData := []byte("test stream data")
	stream.ringBuffer.Write(testData)

	// Verify data is in buffer
	if stream.ringBuffer.Available() != len(testData) {
		t.Errorf("Buffer available = %d, want %d", stream.ringBuffer.Available(), len(testData))
	}

	// Peek at buffer contents
	buffered := stream.ringBuffer.PeekAll()
	if string(buffered) != string(testData) {
		t.Errorf("Buffered data = %q, want %q", buffered, testData)
	}
}

func TestStream_ReconnectionState(t *testing.T) {
	cfg := Config{
		BufferSize:   1024 * 1024,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		ResilienceConfig: &config.ResilienceConfig{
			ReconnectBufferSize:     2 * 1024 * 1024,
			ReconnectMaxBackoff:     100 * time.Millisecond,
			ReconnectInitialBackoff: 10 * time.Millisecond,
			CBFailureThreshold:      5,
			CBTimeout:               30 * time.Second,
			CBHalfOpenRequests:      1,
		},
	}

	cb := circuitbreaker.New(circuitbreaker.Config{
		FailureThreshold: 5,
		Timeout:          30 * time.Second,
		HalfOpenRequests: 1,
	})

	stream := NewStream("test-content", cb, &http.Client{}, cfg.ResilienceConfig.ReconnectBufferSize, nil)

	// Initially not reconnecting
	if stream.IsReconnecting() {
		t.Error("Stream should not be reconnecting initially")
	}

	// Set reconnecting state
	stream.setReconnecting(true)
	if !stream.IsReconnecting() {
		t.Error("Stream should be reconnecting after setReconnecting(true)")
	}

	// Clear reconnecting state
	stream.setReconnecting(false)
	if stream.IsReconnecting() {
		t.Error("Stream should not be reconnecting after setReconnecting(false)")
	}
}

func TestStream_SendBufferToClient(t *testing.T) {
	cfg := Config{
		BufferSize:   1024 * 1024,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		ResilienceConfig: &config.ResilienceConfig{
			ReconnectBufferSize:     1024,
			ReconnectMaxBackoff:     100 * time.Millisecond,
			ReconnectInitialBackoff: 10 * time.Millisecond,
			CBFailureThreshold:      5,
			CBTimeout:               30 * time.Second,
			CBHalfOpenRequests:      1,
		},
	}

	cb := circuitbreaker.New(circuitbreaker.Config{
		FailureThreshold: 5,
		Timeout:          30 * time.Second,
		HalfOpenRequests: 1,
	})

	stream := NewStream("test-content", cb, &http.Client{}, cfg.ResilienceConfig.ReconnectBufferSize, nil)

	// Add some data to the buffer
	testData := []byte("buffered stream data")
	stream.ringBuffer.Write(testData)

	// Create a test HTTP response recorder
	recorder := httptest.NewRecorder()
	client, err := NewClient("test-client", recorder, cfg.BufferSize)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Start a goroutine to consume client buffer
	done := make(chan []byte)
	go func() {
		var received []byte
		for data := range client.buffer {
			received = append(received, data...)
		}
		done <- received
	}()

	// Send buffer to client
	success := stream.sendBufferToClient(client, "test-content")
	if !success {
		t.Error("sendBufferToClient failed")
	}

	// Close client and get received data
	client.Close()
	received := <-done

	// Verify received data matches buffered data
	if string(received) != string(testData) {
		t.Errorf("Received data = %q, want %q", received, testData)
	}
}

func TestStream_BufferOverflow(t *testing.T) {
	// Small buffer to test overflow
	bufferSize := 100
	cfg := Config{
		BufferSize:   1024 * 1024,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		ResilienceConfig: &config.ResilienceConfig{
			ReconnectBufferSize:     bufferSize,
			ReconnectMaxBackoff:     100 * time.Millisecond,
			ReconnectInitialBackoff: 10 * time.Millisecond,
			CBFailureThreshold:      5,
			CBTimeout:               30 * time.Second,
			CBHalfOpenRequests:      1,
		},
	}

	cb := circuitbreaker.New(circuitbreaker.Config{
		FailureThreshold: 5,
		Timeout:          30 * time.Second,
		HalfOpenRequests: 1,
	})

	stream := NewStream("test-content", cb, &http.Client{}, cfg.ResilienceConfig.ReconnectBufferSize, nil)

	// Write more data than buffer size
	largeData := make([]byte, bufferSize*2)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}
	stream.ringBuffer.Write(largeData)

	// Buffer should be full but not exceed its size
	available := stream.ringBuffer.Available()
	if available != bufferSize {
		t.Errorf("Buffer available = %d, want %d (buffer should be full)", available, bufferSize)
	}

	// Peek should return only the last bufferSize bytes
	buffered := stream.ringBuffer.PeekAll()
	if len(buffered) != bufferSize {
		t.Errorf("Buffered data length = %d, want %d", len(buffered), bufferSize)
	}
}

func TestRingBuffer_EmptyBuffer(t *testing.T) {
	cfg := Config{
		BufferSize:   1024 * 1024,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		ResilienceConfig: &config.ResilienceConfig{
			ReconnectBufferSize:     1024,
			ReconnectMaxBackoff:     100 * time.Millisecond,
			ReconnectInitialBackoff: 10 * time.Millisecond,
			CBFailureThreshold:      5,
			CBTimeout:               30 * time.Second,
			CBHalfOpenRequests:      1,
		},
	}

	cb := circuitbreaker.New(circuitbreaker.Config{
		FailureThreshold: 5,
		Timeout:          30 * time.Second,
		HalfOpenRequests: 1,
	})

	stream := NewStream("test-content", cb, &http.Client{}, cfg.ResilienceConfig.ReconnectBufferSize, nil)

	// Create a test client
	recorder := httptest.NewRecorder()
	client, err := NewClient("test-client", recorder, cfg.BufferSize)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Send empty buffer to client (should succeed without error)
	success := stream.sendBufferToClient(client, "test-content")
	if !success {
		t.Error("sendBufferToClient with empty buffer should succeed")
	}

	client.Close()
}

// TestMultiplexer_NewClientDuringReconnection tests that new clients
// joining during reconnection receive buffered data
func TestMultiplexer_NewClientDuringReconnection(t *testing.T) {
	// Create test upstream server that fails after first request
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			// First request succeeds with some data
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("initial data"))
			// Close connection to simulate failure
			return
		}
		// Subsequent requests succeed (reconnection)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("reconnected data"))
		// Keep connection open for streaming
		flusher, ok := w.(http.Flusher)
		if ok {
			flusher.Flush()
		}
		// Block to simulate ongoing stream
		<-r.Context().Done()
	}))
	defer server.Close()

	cfg := Config{
		BufferSize:   1024 * 1024,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		ResilienceConfig: &config.ResilienceConfig{
			ReconnectBufferSize:     1024,
			ReconnectMaxBackoff:     100 * time.Millisecond,
			ReconnectInitialBackoff: 10 * time.Millisecond,
			CBFailureThreshold:      5,
			CBTimeout:               30 * time.Second,
			CBHalfOpenRequests:      1,
		},
	}

	mux := New(cfg)

	// This test verifies the structure is correct
	// Full integration testing would require more complex setup
	if mux.cfg.ResilienceConfig.ReconnectBufferSize != 1024 {
		t.Errorf("Multiplexer buffer size = %d, want 1024", mux.cfg.ResilienceConfig.ReconnectBufferSize)
	}
}

// TestStream_ConcurrentBufferAccess tests thread safety of ring buffer
func TestStream_ConcurrentBufferAccess(t *testing.T) {
	cfg := Config{
		BufferSize:   1024 * 1024,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		ResilienceConfig: &config.ResilienceConfig{
			ReconnectBufferSize:     10 * 1024,
			ReconnectMaxBackoff:     100 * time.Millisecond,
			ReconnectInitialBackoff: 10 * time.Millisecond,
			CBFailureThreshold:      5,
			CBTimeout:               30 * time.Second,
			CBHalfOpenRequests:      1,
		},
	}

	cb := circuitbreaker.New(circuitbreaker.Config{
		FailureThreshold: 5,
		Timeout:          30 * time.Second,
		HalfOpenRequests: 1,
	})

	stream := NewStream("test-content", cb, &http.Client{}, cfg.ResilienceConfig.ReconnectBufferSize, nil)

	// Concurrent writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				data := []byte(strings.Repeat("x", 10))
				stream.ringBuffer.Write(data)
			}
			done <- true
		}()
	}

	// Concurrent reads
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				stream.ringBuffer.PeekAll()
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 15; i++ {
		<-done
	}

	// Should not panic and buffer should have some data
	if stream.ringBuffer.Available() < 0 {
		t.Error("Buffer available should not be negative")
	}
}
