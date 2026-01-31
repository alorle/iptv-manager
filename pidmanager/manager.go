package pidmanager

import (
	"fmt"
	"net/http"
	"sync"
)

// ClientIdentifier uniquely identifies a client based on IP and User-Agent
type ClientIdentifier struct {
	IP        string
	UserAgent string
}

// Session represents an active client-stream session
type Session struct {
	ClientID  ClientIdentifier
	StreamID  string
	PID       int
	Connected bool
}

// Manager handles PID generation and session management
type Manager struct {
	mu           sync.RWMutex
	nextPID      int
	sessions     map[string]*Session // key: streamID-clientIP-userAgent
	pidToSession map[int]*Session    // reverse lookup
}

// NewManager creates a new PID manager
func NewManager() *Manager {
	return &Manager{
		nextPID:      1,
		sessions:     make(map[string]*Session),
		pidToSession: make(map[int]*Session),
	}
}

// sessionKey generates a unique key for a client-stream combination
func (m *Manager) sessionKey(streamID string, clientID ClientIdentifier) string {
	return fmt.Sprintf("%s-%s-%s", streamID, clientID.IP, clientID.UserAgent)
}

// GetOrCreatePID returns an existing PID for a reconnecting client or creates a new one
func (m *Manager) GetOrCreatePID(streamID string, clientID ClientIdentifier) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.sessionKey(streamID, clientID)

	// Check if session exists
	if session, exists := m.sessions[key]; exists {
		// Reuse existing PID
		session.Connected = true
		return session.PID
	}

	// Generate new PID
	pid := m.nextPID
	m.nextPID++

	// Create new session
	session := &Session{
		ClientID:  clientID,
		StreamID:  streamID,
		PID:       pid,
		Connected: true,
	}

	m.sessions[key] = session
	m.pidToSession[pid] = session

	return pid
}

// ReleasePID marks a PID as disconnected
func (m *Manager) ReleasePID(pid int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.pidToSession[pid]
	if !exists {
		return fmt.Errorf("PID %d not found", pid)
	}

	session.Connected = false
	return nil
}

// CleanupDisconnected removes disconnected sessions
func (m *Manager) CleanupDisconnected() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	cleaned := 0
	for key, session := range m.sessions {
		if !session.Connected {
			delete(m.sessions, key)
			delete(m.pidToSession, session.PID)
			cleaned++
		}
	}

	return cleaned
}

// GetSession returns session information for a given PID
func (m *Manager) GetSession(pid int) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.pidToSession[pid]
	if !exists {
		return nil, fmt.Errorf("PID %d not found", pid)
	}

	return session, nil
}

// GetActiveSessions returns the number of active sessions
func (m *Manager) GetActiveSessions() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	active := 0
	for _, session := range m.sessions {
		if session.Connected {
			active++
		}
	}

	return active
}

// ExtractClientIdentifier extracts client information from HTTP request
func ExtractClientIdentifier(r *http.Request) ClientIdentifier {
	// Get real IP, considering proxy headers
	ip := r.Header.Get("X-Real-IP")
	if ip == "" {
		ip = r.Header.Get("X-Forwarded-For")
	}
	if ip == "" {
		ip = r.RemoteAddr
	}

	userAgent := r.Header.Get("User-Agent")

	return ClientIdentifier{
		IP:        ip,
		UserAgent: userAgent,
	}
}
