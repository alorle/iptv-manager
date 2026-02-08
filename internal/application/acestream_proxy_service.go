package application

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/alorle/iptv-manager/internal/port/driven"
)

var (
	// ErrEngineUnavailable indicates the AceStream engine is not available.
	ErrEngineUnavailable = errors.New("acestream engine unavailable")
	// ErrStreamNotActive indicates the requested stream is not active.
	ErrStreamNotActive = errors.New("stream not active")
	// ErrInvalidInfoHash indicates the infohash is invalid or empty.
	ErrInvalidInfoHash = errors.New("invalid infohash")
)

// AceStreamProxyService manages multiplexed AceStream connections.
// Multiple clients can connect to the same infohash, each receiving a unique PID.
// The service manages session lifecycle and cleanup.
type AceStreamProxyService struct {
	engine   driven.AceStreamEngine
	sessions *sessionRegistry
	mu       sync.Mutex
	pidGen   *pidGenerator
	logger   *slog.Logger
}

// NewAceStreamProxyService creates a new proxy service with the given engine.
func NewAceStreamProxyService(engine driven.AceStreamEngine, logger *slog.Logger) *AceStreamProxyService {
	return &AceStreamProxyService{
		engine:   engine,
		sessions: newSessionRegistry(),
		pidGen:   newPIDGenerator(),
		logger:   logger,
	}
}

// StreamToClient initiates a stream for the given infohash and streams content
// to the provided writer. Returns when the stream ends or an error occurs.
func (s *AceStreamProxyService) StreamToClient(ctx context.Context, infoHash string, dst io.Writer) error {
	if infoHash == "" {
		return ErrInvalidInfoHash
	}

	// Generate unique PID for this client
	pid := s.pidGen.Generate()

	s.logger.Info("client connecting to stream", "infohash", infoHash, "pid", pid)

	// Register the client session
	session, isNew, err := s.sessions.AddClient(infoHash, pid)
	if err != nil {
		s.logger.Error("failed to register client", "infohash", infoHash, "pid", pid, "error", err)
		return fmt.Errorf("failed to register client: %w", err)
	}

	// Ensure cleanup on exit
	defer s.cleanupClient(infoHash, pid)

	// If this is a new session, start the stream with the engine
	if isNew {
		s.logger.Info("starting new stream session", "infohash", infoHash, "pid", pid)
		if err := s.startEngineStream(ctx, session); err != nil {
			s.logger.Error("failed to start engine stream", "infohash", infoHash, "pid", pid, "error", err)
			s.sessions.RemoveClient(infoHash, pid)
			return fmt.Errorf("failed to start engine stream: %w", err)
		}
	} else {
		s.logger.Debug("joining existing stream session", "infohash", infoHash, "pid", pid)
		// Wait for the stream to be ready if another client is starting it
		if err := s.waitForStreamReady(ctx, session); err != nil {
			s.logger.Error("stream not ready", "infohash", infoHash, "pid", pid, "error", err)
			s.sessions.RemoveClient(infoHash, pid)
			return err
		}
	}

	// Stream content to the client with reconnection support
	return s.streamWithReconnection(ctx, session, pid, dst)
}

// startEngineStream initiates the stream with the AceStream engine.
func (s *AceStreamProxyService) startEngineStream(ctx context.Context, session *streamSession) error {
	// Use the first PID in the session to start the stream
	firstPID := session.GetFirstPID()
	if firstPID == "" {
		return fmt.Errorf("no PID available to start stream")
	}

	streamURL, err := s.engine.StartStream(ctx, session.InfoHash(), firstPID)
	if err != nil {
		session.SetError(err)
		return err
	}

	s.logger.Info("stream started", "infohash", session.InfoHash(), "pid", firstPID, "stream_url", streamURL)
	session.SetStreamURL(streamURL)
	session.MarkReady()
	return nil
}

// waitForStreamReady waits for the stream to become ready.
func (s *AceStreamProxyService) waitForStreamReady(ctx context.Context, session *streamSession) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	timeout := time.After(30 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("timeout waiting for stream to be ready")
		case <-ticker.C:
			if session.IsReady() {
				return nil
			}
			if err := session.GetError(); err != nil {
				return err
			}
		}
	}
}

// streamWithReconnection streams content with automatic reconnection on failure.
func (s *AceStreamProxyService) streamWithReconnection(ctx context.Context, session *streamSession, pid string, dst io.Writer) error {
	const maxRetries = 3
	retryDelay := 2 * time.Second

	for attempt := 0; attempt < maxRetries; attempt++ {
		streamURL := session.GetStreamURL()
		if streamURL == "" {
			return fmt.Errorf("stream URL not available")
		}

		err := s.engine.StreamContent(ctx, streamURL, dst)
		if err == nil || err == context.Canceled {
			return err
		}

		s.logger.Warn("stream content error", "infohash", session.InfoHash(), "pid", pid, "attempt", attempt+1, "error", err)

		// Check if we should retry
		if attempt < maxRetries-1 {
			// Try to restart the stream
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(retryDelay):
				s.logger.Info("retrying stream", "infohash", session.InfoHash(), "pid", pid, "attempt", attempt+2)
				// Attempt to restart the stream
				if restartErr := s.restartStream(ctx, session, pid); restartErr != nil {
					s.logger.Error("stream restart failed", "infohash", session.InfoHash(), "pid", pid, "error", restartErr)
					return fmt.Errorf("stream failed and could not restart: %w", err)
				}
				retryDelay *= 2 // Exponential backoff
			}
		} else {
			return fmt.Errorf("stream failed after %d attempts: %w", maxRetries, err)
		}
	}

	return fmt.Errorf("stream failed after maximum retries")
}

// restartStream attempts to restart a failed stream.
func (s *AceStreamProxyService) restartStream(ctx context.Context, session *streamSession, pid string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Try to start a new stream with the current PID
	streamURL, err := s.engine.StartStream(ctx, session.InfoHash(), pid)
	if err != nil {
		return err
	}

	session.SetStreamURL(streamURL)
	return nil
}

// cleanupClient removes the client and stops the stream if it's the last one.
func (s *AceStreamProxyService) cleanupClient(infoHash, pid string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.Info("client disconnected", "infohash", infoHash, "pid", pid)

	isLast := s.sessions.RemoveClient(infoHash, pid)

	// If this was the last client, stop the stream
	if isLast {
		s.logger.Info("last client disconnected, stopping stream", "infohash", infoHash, "pid", pid)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Use the PID that was just removed to stop the stream
		if err := s.engine.StopStream(ctx, pid); err != nil {
			s.logger.Error("failed to stop stream", "infohash", infoHash, "pid", pid, "error", err)
		}
	}
}

// GetActiveStreams returns information about all active stream sessions.
func (s *AceStreamProxyService) GetActiveStreams() []StreamInfo {
	return s.sessions.GetAllSessions()
}

// StreamInfo contains information about an active stream session.
type StreamInfo struct {
	InfoHash    string
	ClientCount int
	PIDs        []string
}

// sessionRegistry manages all active stream sessions.
type sessionRegistry struct {
	mu       sync.RWMutex
	sessions map[string]*streamSession // infohash -> session
}

func newSessionRegistry() *sessionRegistry {
	return &sessionRegistry{
		sessions: make(map[string]*streamSession),
	}
}

// AddClient adds a client to a session, creating the session if needed.
// Returns the session, whether it's new, and any error.
func (r *sessionRegistry) AddClient(infoHash, pid string) (*streamSession, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, exists := r.sessions[infoHash]
	if !exists {
		session = newStreamSession(infoHash)
		r.sessions[infoHash] = session
	}

	session.AddPID(pid)
	return session, !exists, nil
}

// RemoveClient removes a client from a session.
// Returns true if this was the last client and the session was removed.
func (r *sessionRegistry) RemoveClient(infoHash, pid string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, exists := r.sessions[infoHash]
	if !exists {
		return false
	}

	session.RemovePID(pid)

	if session.ClientCount() == 0 {
		delete(r.sessions, infoHash)
		return true
	}

	return false
}

// GetAllSessions returns information about all active sessions.
func (r *sessionRegistry) GetAllSessions() []StreamInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]StreamInfo, 0, len(r.sessions))
	for _, session := range r.sessions {
		result = append(result, StreamInfo{
			InfoHash:    session.InfoHash(),
			ClientCount: session.ClientCount(),
			PIDs:        session.GetPIDs(),
		})
	}
	return result
}

// streamSession represents an active stream with multiple clients.
type streamSession struct {
	mu        sync.RWMutex
	infoHash  string
	pids      map[string]struct{}
	streamURL string
	ready     bool
	err       error
}

func newStreamSession(infoHash string) *streamSession {
	return &streamSession{
		infoHash: infoHash,
		pids:     make(map[string]struct{}),
	}
}

func (s *streamSession) InfoHash() string {
	return s.infoHash
}

func (s *streamSession) AddPID(pid string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pids[pid] = struct{}{}
}

func (s *streamSession) RemovePID(pid string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.pids, pid)
}

func (s *streamSession) ClientCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.pids)
}

func (s *streamSession) GetPIDs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]string, 0, len(s.pids))
	for pid := range s.pids {
		result = append(result, pid)
	}
	return result
}

func (s *streamSession) GetFirstPID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for pid := range s.pids {
		return pid
	}
	return ""
}

func (s *streamSession) SetStreamURL(url string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.streamURL = url
}

func (s *streamSession) GetStreamURL() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.streamURL
}

func (s *streamSession) MarkReady() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ready = true
}

func (s *streamSession) IsReady() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ready
}

func (s *streamSession) SetError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.err = err
}

func (s *streamSession) GetError() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.err
}

// pidGenerator generates unique PIDs for clients.
type pidGenerator struct {
	mu      sync.Mutex
	counter uint64
}

func newPIDGenerator() *pidGenerator {
	return &pidGenerator{
		counter: uint64(time.Now().UnixNano()),
	}
}

// Generate creates a unique PID.
func (g *pidGenerator) Generate() string {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.counter++
	return fmt.Sprintf("pid-%d", g.counter)
}
