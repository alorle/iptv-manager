package application

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"sync"
	"testing"
	"time"
)

func TestStreamBroadcaster_Write(t *testing.T) {
	t.Run("broadcasts to multiple subscribers", func(t *testing.T) {
		b := newStreamBroadcaster("test-hash", slog.Default())

		var buf1, buf2 bytes.Buffer
		done1 := make(chan error, 1)
		done2 := make(chan error, 1)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() { done1 <- b.Subscribe(ctx, "pid-1", &buf1, 10*time.Second) }()
		go func() { done2 <- b.Subscribe(ctx, "pid-2", &buf2, 10*time.Second) }()

		// Give subscribers time to register
		time.Sleep(50 * time.Millisecond)

		data := []byte("hello broadcast")
		n, err := b.Write(data)
		if err != nil {
			t.Fatalf("Write returned error: %v", err)
		}
		if n != len(data) {
			t.Fatalf("Write returned %d, want %d", n, len(data))
		}

		// Close to end the subscribers
		b.Close()

		<-done1
		<-done2

		if !bytes.Equal(buf1.Bytes(), data) {
			t.Errorf("subscriber 1 got %q, want %q", buf1.String(), string(data))
		}
		if !bytes.Equal(buf2.Bytes(), data) {
			t.Errorf("subscriber 2 got %q, want %q", buf2.String(), string(data))
		}
	})

	t.Run("write to closed broadcaster returns error", func(t *testing.T) {
		b := newStreamBroadcaster("test-hash", slog.Default())
		b.Close()

		_, err := b.Write([]byte("data"))
		if err == nil {
			t.Fatal("expected error writing to closed broadcaster")
		}
	})

	t.Run("write with no subscribers succeeds", func(t *testing.T) {
		b := newStreamBroadcaster("test-hash", slog.Default())

		n, err := b.Write([]byte("nobody listening"))
		if err != nil {
			t.Fatalf("Write returned error: %v", err)
		}
		if n != len("nobody listening") {
			t.Fatalf("Write returned %d, want %d", n, len("nobody listening"))
		}
	})
}

func TestStreamBroadcaster_Close(t *testing.T) {
	t.Run("close stops all subscribers", func(t *testing.T) {
		b := newStreamBroadcaster("test-hash", slog.Default())

		var buf bytes.Buffer
		done := make(chan error, 1)

		go func() { done <- b.Subscribe(context.Background(), "pid-1", &buf, 10*time.Second) }()

		time.Sleep(50 * time.Millisecond)
		b.Close()

		err := <-done
		if err != nil {
			t.Fatalf("Subscribe returned error: %v", err)
		}
	})

	t.Run("double close is safe", func(t *testing.T) {
		b := newStreamBroadcaster("test-hash", slog.Default())
		b.Close()
		b.Close() // should not panic
	})
}

func TestStreamBroadcaster_Subscribe(t *testing.T) {
	t.Run("subscribe after close returns immediately", func(t *testing.T) {
		b := newStreamBroadcaster("test-hash", slog.Default())
		b.Close()

		var buf bytes.Buffer
		err := b.Subscribe(context.Background(), "pid-1", &buf, 10*time.Second)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})

	t.Run("context cancellation stops subscriber", func(t *testing.T) {
		b := newStreamBroadcaster("test-hash", slog.Default())

		ctx, cancel := context.WithCancel(context.Background())
		var buf bytes.Buffer
		done := make(chan error, 1)

		go func() { done <- b.Subscribe(ctx, "pid-1", &buf, 10*time.Second) }()

		time.Sleep(50 * time.Millisecond)
		cancel()

		err := <-done
		if err != context.Canceled {
			t.Fatalf("expected context.Canceled, got %v", err)
		}

		b.Close()
	})

	t.Run("multiple chunks delivered in order", func(t *testing.T) {
		b := newStreamBroadcaster("test-hash", slog.Default())

		var buf bytes.Buffer
		done := make(chan error, 1)

		go func() { done <- b.Subscribe(context.Background(), "pid-1", &buf, 10*time.Second) }()

		time.Sleep(50 * time.Millisecond)

		chunks := []string{"chunk1-", "chunk2-", "chunk3"}
		for _, c := range chunks {
			if _, err := b.Write([]byte(c)); err != nil {
				t.Fatalf("Write error: %v", err)
			}
		}

		b.Close()
		<-done

		expected := "chunk1-chunk2-chunk3"
		if buf.String() != expected {
			t.Errorf("got %q, want %q", buf.String(), expected)
		}
	})
}

func TestStreamBroadcaster_SlowClient(t *testing.T) {
	t.Run("slow client is dropped when buffer fills", func(t *testing.T) {
		b := newStreamBroadcaster("test-hash", slog.Default())

		// Subscribe a client that never reads
		slowClient := &broadcastClient{
			chunks: make(chan []byte, broadcastBufferSize),
			pid:    "slow-pid",
		}
		b.mu.Lock()
		b.clients["slow-pid"] = slowClient
		b.mu.Unlock()

		// Fill the buffer
		for i := 0; i < broadcastBufferSize; i++ {
			if _, err := b.Write([]byte("x")); err != nil {
				t.Fatalf("Write %d failed: %v", i, err)
			}
		}

		// Next write should drop the slow client
		if _, err := b.Write([]byte("overflow")); err != nil {
			t.Fatalf("overflow Write failed: %v", err)
		}

		b.mu.Lock()
		_, exists := b.clients["slow-pid"]
		b.mu.Unlock()

		if exists {
			t.Error("slow client should have been dropped")
		}
	})
}

func TestStreamBroadcaster_ConcurrentAccess(t *testing.T) {
	t.Run("concurrent subscribe unsubscribe and write", func(t *testing.T) {
		b := newStreamBroadcaster("test-hash", slog.Default())

		var wg sync.WaitGroup
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		// Writers
		for i := 0; i < 3; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					select {
					case <-ctx.Done():
						return
					default:
						_, _ = b.Write([]byte("data"))
					}
				}
			}()
		}

		// Subscribers that come and go
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				var buf bytes.Buffer
				subCtx, subCancel := context.WithTimeout(ctx, 100*time.Millisecond)
				defer subCancel()
				_ = b.Subscribe(subCtx, fmt.Sprintf("pid-%d", id), &buf, 10*time.Second)
			}(i)
		}

		wg.Wait()
		b.Close()
	})
}
