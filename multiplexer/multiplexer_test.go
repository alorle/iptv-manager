package multiplexer

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// mockResponseWriter implements http.ResponseWriter and http.Flusher for testing
type mockResponseWriter struct {
	headers    http.Header
	body       []byte
	statusCode int
	mu         sync.Mutex
}

func newMockResponseWriter() *mockResponseWriter {
	return &mockResponseWriter{
		headers: make(http.Header),
	}
}

func (m *mockResponseWriter) Header() http.Header {
	return m.headers
}

func (m *mockResponseWriter) Write(b []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.body = append(m.body, b...)
	return len(b), nil
}

func (m *mockResponseWriter) WriteHeader(statusCode int) {
	m.statusCode = statusCode
}

func (m *mockResponseWriter) Flush() {
	// No-op for mock
}

func (m *mockResponseWriter) Body() []byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.body
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.BufferSize != 1024*1024 {
		t.Errorf("Expected BufferSize 1MB, got %d", cfg.BufferSize)
	}
	if cfg.ReadTimeout != 30*time.Second {
		t.Errorf("Expected ReadTimeout 30s, got %v", cfg.ReadTimeout)
	}
	if cfg.WriteTimeout != 10*time.Second {
		t.Errorf("Expected WriteTimeout 10s, got %v", cfg.WriteTimeout)
	}
}

func TestNewClient(t *testing.T) {
	w := newMockResponseWriter()
	client, err := NewClient("test-client-1", w, 1024*1024)

	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	if client.ID != "test-client-1" {
		t.Errorf("Expected client ID 'test-client-1', got %s", client.ID)
	}

	if client.Writer != w {
		t.Error("Client writer not set correctly")
	}

	if client.Done == nil {
		t.Error("Client Done channel not initialized")
	}

	if client.buffer == nil {
		t.Error("Client buffer not initialized")
	}
}

func TestNewClient_NonFlushableWriter(t *testing.T) {
	// Use a non-flushable writer
	type nonFlushableWriter struct {
		http.ResponseWriter
	}
	w := &nonFlushableWriter{}

	_, err := NewClient("test-client", w, 1024*1024)

	if err == nil {
		t.Error("Expected error for non-flushable writer")
	}

	if !strings.Contains(err.Error(), "streaming not supported") {
		t.Errorf("Expected 'streaming not supported' error, got: %v", err)
	}
}

func TestClient_Send(t *testing.T) {
	w := newMockResponseWriter()
	client, _ := NewClient("test-client", w, 1024*1024)

	data := []byte("test data")
	err := client.Send(data)

	if err != nil {
		t.Errorf("Send failed: %v", err)
	}

	// Verify data is in buffer
	select {
	case received := <-client.buffer:
		if string(received) != string(data) {
			t.Errorf("Expected data '%s', got '%s'", string(data), string(received))
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for data in buffer")
	}
}

func TestClient_SendAfterClose(t *testing.T) {
	w := newMockResponseWriter()
	client, _ := NewClient("test-client", w, 1024*1024)

	client.Close()

	err := client.Send([]byte("test"))
	if err == nil {
		t.Error("Expected error when sending to closed client")
	}

	if !strings.Contains(err.Error(), "client closed") {
		t.Errorf("Expected 'client closed' error, got: %v", err)
	}
}

func TestClient_Close(t *testing.T) {
	w := newMockResponseWriter()
	client, _ := NewClient("test-client", w, 1024*1024)

	// Close should be idempotent
	client.Close()
	client.Close()

	// Verify Done channel is closed
	select {
	case <-client.Done:
		// Success
	case <-time.After(1 * time.Second):
		t.Error("Done channel not closed")
	}
}

func TestNewStream(t *testing.T) {
	stream := NewStream("test-content-id")

	if stream.ContentID != "test-content-id" {
		t.Errorf("Expected ContentID 'test-content-id', got %s", stream.ContentID)
	}

	if stream.Clients == nil {
		t.Error("Clients map not initialized")
	}

	if stream.done == nil {
		t.Error("done channel not initialized")
	}
}

func TestStream_AddClient(t *testing.T) {
	stream := NewStream("test-content")
	w := newMockResponseWriter()
	client, _ := NewClient("client-1", w, 1024*1024)

	stream.AddClient(client)

	if stream.ClientCount() != 1 {
		t.Errorf("Expected 1 client, got %d", stream.ClientCount())
	}

	if stream.Clients["client-1"] != client {
		t.Error("Client not added to map")
	}
}

func TestStream_RemoveClient(t *testing.T) {
	stream := NewStream("test-content")
	w := newMockResponseWriter()
	client, _ := NewClient("client-1", w, 1024*1024)

	stream.AddClient(client)
	remaining := stream.RemoveClient("client-1")

	if remaining != 0 {
		t.Errorf("Expected 0 remaining clients, got %d", remaining)
	}

	if stream.ClientCount() != 0 {
		t.Errorf("Expected 0 clients after removal, got %d", stream.ClientCount())
	}
}

func TestStream_RemoveClient_NonExistent(t *testing.T) {
	stream := NewStream("test-content")

	remaining := stream.RemoveClient("non-existent")

	if remaining != 0 {
		t.Errorf("Expected 0 clients, got %d", remaining)
	}
}

func TestStream_ClientCount(t *testing.T) {
	stream := NewStream("test-content")
	w1 := newMockResponseWriter()
	w2 := newMockResponseWriter()

	client1, _ := NewClient("client-1", w1, 1024*1024)
	client2, _ := NewClient("client-2", w2, 1024*1024)

	if stream.ClientCount() != 0 {
		t.Error("Expected 0 clients initially")
	}

	stream.AddClient(client1)
	if stream.ClientCount() != 1 {
		t.Errorf("Expected 1 client, got %d", stream.ClientCount())
	}

	stream.AddClient(client2)
	if stream.ClientCount() != 2 {
		t.Errorf("Expected 2 clients, got %d", stream.ClientCount())
	}

	stream.RemoveClient("client-1")
	if stream.ClientCount() != 1 {
		t.Errorf("Expected 1 client after removal, got %d", stream.ClientCount())
	}
}

func TestNew(t *testing.T) {
	cfg := Config{
		BufferSize:   2 * 1024 * 1024,
		ReadTimeout:  20 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	mux := New(cfg)

	if mux == nil {
		t.Fatal("New returned nil")
	}

	if mux.cfg.BufferSize != cfg.BufferSize {
		t.Errorf("Expected BufferSize %d, got %d", cfg.BufferSize, mux.cfg.BufferSize)
	}

	if mux.streams == nil {
		t.Error("streams map not initialized")
	}

	if mux.client == nil {
		t.Error("HTTP client not initialized")
	}
}

func TestMultiplexer_GetOrCreateStream_New(t *testing.T) {
	// Create a test server that streams data
	testData := []byte("streaming test data")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "video/mp2t")
		w.WriteHeader(http.StatusOK)
		w.Write(testData)
	}))
	defer server.Close()

	mux := New(DefaultConfig())
	ctx := context.Background()

	stream, existed, err := mux.GetOrCreateStream(ctx, "test-content", server.URL)

	if err != nil {
		t.Fatalf("GetOrCreateStream failed: %v", err)
	}

	if existed {
		t.Error("Expected new stream, got existing")
	}

	if stream.ContentID != "test-content" {
		t.Errorf("Expected ContentID 'test-content', got %s", stream.ContentID)
	}

	// Verify stream was stored
	mux.mu.RLock()
	storedStream, exists := mux.streams["test-content"]
	mux.mu.RUnlock()

	if !exists {
		t.Error("Stream not stored in multiplexer")
	}

	if storedStream != stream {
		t.Error("Stored stream doesn't match returned stream")
	}

	// Cleanup
	stream.Stop()
}

func TestMultiplexer_GetOrCreateStream_Existing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Send minimal data and let connection close naturally
		w.Write([]byte("test"))
	}))
	defer server.Close()

	mux := New(DefaultConfig())
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Create first stream
	stream1, existed1, err := mux.GetOrCreateStream(ctx, "test-content", server.URL)
	if err != nil {
		t.Fatalf("First GetOrCreateStream failed: %v", err)
	}
	if existed1 {
		t.Error("First stream should be new")
	}

	// Small delay to ensure stream is started
	time.Sleep(50 * time.Millisecond)

	// Try to create same stream again
	stream2, existed2, err := mux.GetOrCreateStream(ctx, "test-content", server.URL)
	if err != nil {
		t.Fatalf("Second GetOrCreateStream failed: %v", err)
	}

	if !existed2 {
		t.Error("Second stream should be existing")
	}

	if stream1 != stream2 {
		t.Error("Expected same stream instance")
	}

	// Cleanup
	stream1.Stop()
}

func TestMultiplexer_GetOrCreateStream_UpstreamError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	mux := New(DefaultConfig())
	ctx := context.Background()

	_, _, err := mux.GetOrCreateStream(ctx, "test-content", server.URL)

	if err == nil {
		t.Error("Expected error for upstream error response")
	}

	if !strings.Contains(err.Error(), "upstream returned status 500") {
		t.Errorf("Expected upstream status error, got: %v", err)
	}
}

func TestMultiplexer_GetOrCreateStream_InvalidURL(t *testing.T) {
	mux := New(DefaultConfig())
	ctx := context.Background()

	_, _, err := mux.GetOrCreateStream(ctx, "test-content", "http://localhost:99999/invalid")

	if err == nil {
		t.Error("Expected error for invalid upstream URL")
	}
}

func TestMultiplexer_RemoveStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	mux := New(DefaultConfig())
	ctx := context.Background()

	stream, _, err := mux.GetOrCreateStream(ctx, "test-content", server.URL)
	if err != nil {
		t.Fatalf("GetOrCreateStream failed: %v", err)
	}

	// Remove stream (has no clients)
	mux.RemoveStream("test-content")

	// Verify stream was removed
	mux.mu.RLock()
	_, exists := mux.streams["test-content"]
	mux.mu.RUnlock()

	if exists {
		t.Error("Stream should be removed")
	}

	// Cleanup
	stream.Stop()
}

func TestMultiplexer_RemoveStream_WithClients(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test"))
	}))
	defer server.Close()

	mux := New(DefaultConfig())
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	stream, _, err := mux.GetOrCreateStream(ctx, "test-content", server.URL)
	if err != nil {
		t.Fatalf("GetOrCreateStream failed: %v", err)
	}

	// Add a client
	w := newMockResponseWriter()
	client, _ := NewClient("client-1", w, mux.cfg.BufferSize)
	stream.AddClient(client)

	// Try to remove stream (has clients)
	mux.RemoveStream("test-content")

	// Verify stream was NOT removed
	mux.mu.RLock()
	_, exists := mux.streams["test-content"]
	mux.mu.RUnlock()

	if !exists {
		t.Error("Stream with clients should not be removed")
	}

	// Cleanup
	stream.RemoveClient("client-1")
	stream.Stop()
}

func TestMultiplexer_Stats(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test"))
	}))
	defer server.Close()

	mux := New(DefaultConfig())
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Initially no streams
	stats := mux.Stats()
	if stats["total_streams"] != 0 {
		t.Errorf("Expected 0 streams, got %v", stats["total_streams"])
	}

	// Create streams
	stream1, _, _ := mux.GetOrCreateStream(ctx, "content-1", server.URL)
	w1 := newMockResponseWriter()
	client1, _ := NewClient("client-1", w1, mux.cfg.BufferSize)
	stream1.AddClient(client1)

	stream2, _, _ := mux.GetOrCreateStream(ctx, "content-2", server.URL)
	w2 := newMockResponseWriter()
	client2, _ := NewClient("client-2", w2, mux.cfg.BufferSize)
	stream2.AddClient(client2)

	stats = mux.Stats()
	if stats["total_streams"] != 2 {
		t.Errorf("Expected 2 streams, got %v", stats["total_streams"])
	}

	streams := stats["streams"].([]map[string]interface{})
	if len(streams) != 2 {
		t.Errorf("Expected 2 stream entries, got %d", len(streams))
	}

	// Cleanup
	stream1.RemoveClient("client-1")
	stream2.RemoveClient("client-2")
	stream1.Stop()
	stream2.Stop()
}

func TestStream_FanOut(t *testing.T) {
	// Create a test upstream that sends data
	testData := []byte("test stream data chunk")
	pr, pw := io.Pipe()

	// Create stream and clients
	stream := NewStream("test-content")
	w1 := newMockResponseWriter()
	w2 := newMockResponseWriter()
	client1, _ := NewClient("client-1", w1, 1024*1024)
	client2, _ := NewClient("client-2", w2, 1024*1024)

	stream.AddClient(client1)
	stream.AddClient(client2)

	// Start the stream
	ctx := context.Background()
	stream.Start(ctx, pr, DefaultConfig())

	// Give stream time to start
	time.Sleep(50 * time.Millisecond)

	// Write data to upstream
	pw.Write(testData)

	// Verify both clients received the data
	select {
	case data1 := <-client1.buffer:
		if string(data1) != string(testData) {
			t.Errorf("Client 1 received wrong data: %s", string(data1))
		}
	case <-time.After(1 * time.Second):
		t.Error("Client 1 timeout")
	}

	select {
	case data2 := <-client2.buffer:
		if string(data2) != string(testData) {
			t.Errorf("Client 2 received wrong data: %s", string(data2))
		}
	case <-time.After(1 * time.Second):
		t.Error("Client 2 timeout")
	}

	// Cleanup
	pw.Close()
	stream.Stop()
}

func TestStream_FanOut_ClientRemovalDuringStream(t *testing.T) {
	testData := []byte("test data")
	pr, pw := io.Pipe()

	stream := NewStream("test-content")
	w := newMockResponseWriter()
	client, _ := NewClient("client-1", w, 1024*1024)
	stream.AddClient(client)

	ctx := context.Background()
	stream.Start(ctx, pr, DefaultConfig())

	time.Sleep(50 * time.Millisecond)

	// Write some data
	pw.Write(testData)

	// Verify client received data
	select {
	case <-client.buffer:
		// Success
	case <-time.After(1 * time.Second):
		t.Error("Client timeout")
	}

	// Remove client during stream
	stream.RemoveClient("client-1")

	// Write more data - should not panic
	pw.Write(testData)

	// Cleanup
	pw.Close()
	stream.Stop()
}

func TestMultiplexer_ConcurrentStreams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Stream some data
		for i := 0; i < 5; i++ {
			w.Write([]byte("data"))
			time.Sleep(10 * time.Millisecond)
		}
	}))
	defer server.Close()

	mux := New(DefaultConfig())
	ctx := context.Background()

	// Create multiple streams concurrently
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			contentID := string(rune('A' + id))
			stream, _, err := mux.GetOrCreateStream(ctx, contentID, server.URL)
			if err != nil {
				t.Errorf("GetOrCreateStream failed for %s: %v", contentID, err)
				return
			}
			stream.Stop()
		}(i)
	}

	wg.Wait()

	stats := mux.Stats()
	totalStreams := stats["total_streams"].(int)
	if totalStreams != 5 {
		t.Errorf("Expected 5 streams, got %d", totalStreams)
	}
}

func TestClient_SendBufferFull(t *testing.T) {
	w := newMockResponseWriter()
	// Create client with very small buffer
	client, _ := NewClient("test-client", w, 1)

	// Fill the buffer
	for i := 0; i < 100; i++ {
		err := client.Send([]byte("data"))
		if err != nil {
			// Expected - buffer should fill up
			if !strings.Contains(err.Error(), "client buffer full") {
				t.Errorf("Expected 'client buffer full' error, got: %v", err)
			}
			return
		}
	}

	t.Error("Expected buffer full error but none occurred")
}

func TestStream_Stop(t *testing.T) {
	pr, pw := io.Pipe()
	stream := NewStream("test-content")

	ctx := context.Background()
	stream.Start(ctx, pr, DefaultConfig())

	// Stop should complete without hanging
	done := make(chan bool)
	go func() {
		pw.Close()
		stream.Stop()
		done <- true
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Error("Stream stop timed out")
	}
}
