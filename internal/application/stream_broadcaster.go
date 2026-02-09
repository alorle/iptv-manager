package application

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/alorle/iptv-manager/internal/streaming"
)

const broadcastBufferSize = 128

// broadcastClient represents a single subscriber to a broadcast stream.
type broadcastClient struct {
	chunks chan []byte
	pid    string
}

// streamBroadcaster reads from a single engine stream and distributes data
// to multiple subscribers. It implements io.Writer so it can be used as the
// destination for engine.StreamContent.
type streamBroadcaster struct {
	mu       sync.Mutex
	clients  map[string]*broadcastClient
	closed   bool
	err      error // error that caused the broadcaster to close
	logger   *slog.Logger
	infoHash string
}

func newStreamBroadcaster(infoHash string, logger *slog.Logger) *streamBroadcaster {
	return &streamBroadcaster{
		clients:  make(map[string]*broadcastClient),
		logger:   logger,
		infoHash: infoHash,
	}
}

// Write implements io.Writer. It copies data to all subscribed clients' channels.
// Slow clients whose channel buffers are full are dropped.
func (b *streamBroadcaster) Write(p []byte) (int, error) {
	data := make([]byte, len(p))
	copy(data, p)

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return 0, io.ErrClosedPipe
	}

	for pid, client := range b.clients {
		select {
		case client.chunks <- data:
		default:
			b.logger.Warn("dropping slow client from broadcast",
				"infohash", b.infoHash,
				"pid", pid)
			close(client.chunks)
			delete(b.clients, pid)
		}
	}

	return len(p), nil
}

// Close signals all subscribers that the stream has ended by closing their channels.
func (b *streamBroadcaster) Close() {
	b.CloseWithError(nil)
}

// CloseWithError signals all subscribers that the stream has ended with the
// given error. Subscribers will receive this error from Subscribe.
func (b *streamBroadcaster) CloseWithError(err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return
	}
	b.closed = true
	b.err = err

	for _, client := range b.clients {
		close(client.chunks)
	}
}

// Subscribe registers a new client and blocks until the stream ends, the context
// is cancelled, or a write error occurs. Each received chunk is written to dst.
func (b *streamBroadcaster) Subscribe(ctx context.Context, pid string, dst io.Writer, writeTimeout time.Duration) error {
	client := &broadcastClient{
		chunks: make(chan []byte, broadcastBufferSize),
		pid:    pid,
	}

	b.mu.Lock()
	if b.closed {
		err := b.err
		b.mu.Unlock()
		return err
	}
	b.clients[pid] = client
	b.mu.Unlock()

	defer b.unsubscribe(pid)

	tw := streaming.NewTimeoutWriter(dst, writeTimeout, b.logger, b.infoHash, pid)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case data, ok := <-client.chunks:
			if !ok {
				b.mu.Lock()
				err := b.err
				b.mu.Unlock()
				return err
			}
			if _, err := tw.Write(data); err != nil {
				return err
			}
			if f, ok := dst.(http.Flusher); ok {
				f.Flush()
			}
		}
	}
}

// unsubscribe removes a client from the broadcast. It does not close the
// client's channel — that is handled by Write (slow client) or Close (stream end).
func (b *streamBroadcaster) unsubscribe(pid string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.clients, pid)
}
