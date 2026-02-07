package multiplexer

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/alorle/iptv-manager/circuitbreaker"
	"github.com/alorle/iptv-manager/metrics"
)

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

			// Attempt reconnection
			attemptNumber = 1
			if !s.attemptReconnection(ctx, cfg, &attemptNumber) {
				s.setReconnecting(false)
				return
			}

			// Mark as no longer reconnecting
			s.setReconnecting(false)
		}

		if n > 0 {
			data := buffer[:n]
			s.distributeData(data)
		}
	}
}

// shouldStopReconnecting checks if reconnection should stop due to context cancellation or no clients
func (s *Stream) shouldStopReconnecting(ctx context.Context, attemptNumber int) bool {
	if ctx.Err() == nil && s.ClientCount() > 0 {
		return false
	}

	reason := "context canceled"
	if s.ClientCount() == 0 {
		reason = "no clients remaining"
	}

	log.Printf("Stream %s: Stopping reconnection - no clients or context canceled", s.ContentID)
	if s.resLogger != nil {
		s.resLogger.LogReconnectFailed(s.ContentID, reason, attemptNumber)
	}

	return true
}

// waitForCircuitBreaker waits for the circuit breaker to close if it's open
// Returns false if context is canceled while waiting
func (s *Stream) waitForCircuitBreaker(ctx context.Context, cfg Config, attemptNumber int) bool {
	if s.circuitBreaker.State() != circuitbreaker.StateOpen {
		return true
	}

	log.Printf("Stream %s: Circuit breaker is OPEN, skipping reconnection attempt %d", s.ContentID, attemptNumber)

	// Wait for circuit breaker timeout before checking again
	select {
	case <-time.After(cfg.ResilienceConfig.CBTimeout):
		return true
	case <-ctx.Done():
		return false
	}
}

// connectToUpstream attempts to establish a new connection to the upstream server
func (s *Stream) connectToUpstream(ctx context.Context) (io.ReadCloser, error) {
	var newUpstream io.ReadCloser

	err := s.circuitBreaker.Execute(func() error {
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

	return newUpstream, err
}

// handleReconnectionSuccess processes a successful reconnection
func (s *Stream) handleReconnectionSuccess(newUpstream io.ReadCloser, attemptNumber int) {
	downtime := time.Since(s.reconnectStart)
	log.Printf("Stream %s: Reconnection attempt #%d succeeded - resuming normal streaming", s.ContentID, attemptNumber)

	if s.resLogger != nil {
		s.resLogger.LogReconnectSuccess(s.ContentID, downtime)
	}

	metrics.RecordUpstreamReconnection(s.ContentID)

	s.mu.Lock()
	s.upstream = newUpstream
	s.mu.Unlock()
}

// calculateNextBackoff calculates the next backoff duration using exponential backoff
func calculateNextBackoff(current, max time.Duration) time.Duration {
	next := current * 2
	if next > max {
		return max
	}
	return next
}

// attemptReconnection attempts to reconnect to the upstream server
// Returns true if reconnection succeeded, false if it should stop trying
func (s *Stream) attemptReconnection(ctx context.Context, cfg Config, attemptNumber *int) bool {
	backoff := cfg.ResilienceConfig.ReconnectInitialBackoff

	for {
		// Check if we should stop reconnecting
		if s.shouldStopReconnecting(ctx, *attemptNumber) {
			return false
		}

		// Check circuit breaker state and wait if needed
		if !s.waitForCircuitBreaker(ctx, cfg, *attemptNumber) {
			return false
		}

		// Log reconnection attempt
		log.Printf("Stream %s: Reconnection attempt #%d (backoff: %v, buffer available: %d bytes)",
			s.ContentID, *attemptNumber, backoff, s.ringBuffer.Available())
		if s.resLogger != nil {
			s.resLogger.LogReconnectAttempt(s.ContentID, *attemptNumber, backoff)
		}

		// Wait for backoff duration
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return false
		}

		// Attempt to reconnect
		newUpstream, reconnectErr := s.connectToUpstream(ctx)
		if reconnectErr != nil {
			log.Printf("Stream %s: Reconnection attempt #%d failed: %v", s.ContentID, *attemptNumber, reconnectErr)
			backoff = calculateNextBackoff(backoff, cfg.ResilienceConfig.ReconnectMaxBackoff)
			*attemptNumber++
			continue
		}

		// Reconnection successful
		s.handleReconnectionSuccess(newUpstream, *attemptNumber)
		return true
	}
}
