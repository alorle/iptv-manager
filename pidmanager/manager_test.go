package pidmanager

import (
	"net/http"
	"testing"
)

func TestNewManager(t *testing.T) {
	m := NewManager()

	if m == nil {
		t.Fatal("NewManager returned nil")
	}

	if m.nextPID != 1 {
		t.Errorf("Expected nextPID to be 1, got %d", m.nextPID)
	}

	if m.sessions == nil {
		t.Error("sessions map not initialized")
	}

	if m.pidToSession == nil {
		t.Error("pidToSession map not initialized")
	}
}

func TestGetOrCreatePID_NewSession(t *testing.T) {
	m := NewManager()
	streamID := "test-stream-123"
	clientID := ClientIdentifier{
		IP:        "192.168.1.100",
		UserAgent: "VLC/3.0.18",
	}

	pid := m.GetOrCreatePID(streamID, clientID)

	if pid != 1 {
		t.Errorf("Expected first PID to be 1, got %d", pid)
	}

	if m.GetActiveSessions() != 1 {
		t.Errorf("Expected 1 active session, got %d", m.GetActiveSessions())
	}
}

func TestGetOrCreatePID_ReuseExisting(t *testing.T) {
	m := NewManager()
	streamID := "test-stream-123"
	clientID := ClientIdentifier{
		IP:        "192.168.1.100",
		UserAgent: "VLC/3.0.18",
	}

	// First call creates new PID
	pid1 := m.GetOrCreatePID(streamID, clientID)

	// Second call with same client and stream should return same PID
	pid2 := m.GetOrCreatePID(streamID, clientID)

	if pid1 != pid2 {
		t.Errorf("Expected same PID for reconnecting client, got %d and %d", pid1, pid2)
	}

	if m.GetActiveSessions() != 1 {
		t.Errorf("Expected 1 active session after reuse, got %d", m.GetActiveSessions())
	}
}

func TestGetOrCreatePID_DifferentClients(t *testing.T) {
	m := NewManager()
	streamID := "test-stream-123"

	client1 := ClientIdentifier{IP: "192.168.1.100", UserAgent: "VLC/3.0.18"}
	client2 := ClientIdentifier{IP: "192.168.1.101", UserAgent: "VLC/3.0.18"}

	pid1 := m.GetOrCreatePID(streamID, client1)
	pid2 := m.GetOrCreatePID(streamID, client2)

	if pid1 == pid2 {
		t.Error("Expected different PIDs for different clients")
	}

	if m.GetActiveSessions() != 2 {
		t.Errorf("Expected 2 active sessions, got %d", m.GetActiveSessions())
	}
}

func TestGetOrCreatePID_DifferentStreams(t *testing.T) {
	m := NewManager()
	clientID := ClientIdentifier{
		IP:        "192.168.1.100",
		UserAgent: "VLC/3.0.18",
	}

	stream1 := "stream-123"
	stream2 := "stream-456"

	pid1 := m.GetOrCreatePID(stream1, clientID)
	pid2 := m.GetOrCreatePID(stream2, clientID)

	if pid1 == pid2 {
		t.Error("Expected different PIDs for different streams")
	}

	if m.GetActiveSessions() != 2 {
		t.Errorf("Expected 2 active sessions, got %d", m.GetActiveSessions())
	}
}

func TestGetOrCreatePID_DifferentUserAgents(t *testing.T) {
	m := NewManager()
	streamID := "test-stream-123"

	client1 := ClientIdentifier{IP: "192.168.1.100", UserAgent: "VLC/3.0.18"}
	client2 := ClientIdentifier{IP: "192.168.1.100", UserAgent: "Kodi/19.0"}

	pid1 := m.GetOrCreatePID(streamID, client1)
	pid2 := m.GetOrCreatePID(streamID, client2)

	if pid1 == pid2 {
		t.Error("Expected different PIDs for different user agents")
	}

	if m.GetActiveSessions() != 2 {
		t.Errorf("Expected 2 active sessions, got %d", m.GetActiveSessions())
	}
}

func TestReleasePID(t *testing.T) {
	m := NewManager()
	streamID := "test-stream-123"
	clientID := ClientIdentifier{
		IP:        "192.168.1.100",
		UserAgent: "VLC/3.0.18",
	}

	pid := m.GetOrCreatePID(streamID, clientID)

	// Release the PID
	err := m.ReleasePID(pid)
	if err != nil {
		t.Errorf("Unexpected error releasing PID: %v", err)
	}

	// Verify session is marked as disconnected
	session, err := m.GetSession(pid)
	if err != nil {
		t.Errorf("Session should still exist after release: %v", err)
	}

	if session.Connected {
		t.Error("Session should be marked as disconnected")
	}
}

func TestReleasePID_InvalidPID(t *testing.T) {
	m := NewManager()

	err := m.ReleasePID(999)
	if err == nil {
		t.Error("Expected error when releasing non-existent PID")
	}
}

func TestCleanupDisconnected(t *testing.T) {
	m := NewManager()
	streamID := "test-stream-123"

	// Create multiple sessions
	client1 := ClientIdentifier{IP: "192.168.1.100", UserAgent: "VLC/3.0.18"}
	client2 := ClientIdentifier{IP: "192.168.1.101", UserAgent: "VLC/3.0.18"}
	client3 := ClientIdentifier{IP: "192.168.1.102", UserAgent: "VLC/3.0.18"}

	pid1 := m.GetOrCreatePID(streamID, client1)
	pid2 := m.GetOrCreatePID(streamID, client2)
	pid3 := m.GetOrCreatePID(streamID, client3)

	// Disconnect two sessions
	_ = m.ReleasePID(pid1)
	_ = m.ReleasePID(pid2)

	// Cleanup
	cleaned := m.CleanupDisconnected()

	if cleaned != 2 {
		t.Errorf("Expected 2 sessions cleaned, got %d", cleaned)
	}

	// Verify only one session remains
	if m.GetActiveSessions() != 1 {
		t.Errorf("Expected 1 active session after cleanup, got %d", m.GetActiveSessions())
	}

	// Verify the remaining session is the connected one
	session, err := m.GetSession(pid3)
	if err != nil {
		t.Errorf("Active session should still exist: %v", err)
	}
	if !session.Connected {
		t.Error("Remaining session should be connected")
	}

	// Verify cleaned sessions are gone
	_, err = m.GetSession(pid1)
	if err == nil {
		t.Error("Cleaned session should not exist")
	}
}

func TestGetSession(t *testing.T) {
	m := NewManager()
	streamID := "test-stream-123"
	clientID := ClientIdentifier{
		IP:        "192.168.1.100",
		UserAgent: "VLC/3.0.18",
	}

	pid := m.GetOrCreatePID(streamID, clientID)

	session, err := m.GetSession(pid)
	if err != nil {
		t.Errorf("Unexpected error getting session: %v", err)
	}

	if session.PID != pid {
		t.Errorf("Expected PID %d, got %d", pid, session.PID)
	}

	if session.StreamID != streamID {
		t.Errorf("Expected stream ID %s, got %s", streamID, session.StreamID)
	}

	if session.ClientID.IP != clientID.IP {
		t.Errorf("Expected IP %s, got %s", clientID.IP, session.ClientID.IP)
	}

	if session.ClientID.UserAgent != clientID.UserAgent {
		t.Errorf("Expected User-Agent %s, got %s", clientID.UserAgent, session.ClientID.UserAgent)
	}
}

func TestGetSession_NotFound(t *testing.T) {
	m := NewManager()

	_, err := m.GetSession(999)
	if err == nil {
		t.Error("Expected error when getting non-existent session")
	}
}

func TestGetActiveSessions(t *testing.T) {
	m := NewManager()
	streamID := "test-stream-123"

	if m.GetActiveSessions() != 0 {
		t.Error("Expected 0 active sessions initially")
	}

	// Create sessions
	client1 := ClientIdentifier{IP: "192.168.1.100", UserAgent: "VLC/3.0.18"}
	client2 := ClientIdentifier{IP: "192.168.1.101", UserAgent: "VLC/3.0.18"}

	pid1 := m.GetOrCreatePID(streamID, client1)
	m.GetOrCreatePID(streamID, client2)

	if m.GetActiveSessions() != 2 {
		t.Errorf("Expected 2 active sessions, got %d", m.GetActiveSessions())
	}

	// Release one
	_ = m.ReleasePID(pid1)

	if m.GetActiveSessions() != 1 {
		t.Errorf("Expected 1 active session after release, got %d", m.GetActiveSessions())
	}
}

func TestExtractClientIdentifier(t *testing.T) {
	tests := []struct {
		name          string
		remoteAddr    string
		xRealIP       string
		xForwardedFor string
		userAgent     string
		expectedIP    string
		expectedUA    string
	}{
		{
			name:       "Direct connection",
			remoteAddr: "192.168.1.100:12345",
			userAgent:  "VLC/3.0.18",
			expectedIP: "192.168.1.100:12345",
			expectedUA: "VLC/3.0.18",
		},
		{
			name:       "With X-Real-IP",
			remoteAddr: "127.0.0.1:12345",
			xRealIP:    "192.168.1.100",
			userAgent:  "VLC/3.0.18",
			expectedIP: "192.168.1.100",
			expectedUA: "VLC/3.0.18",
		},
		{
			name:          "With X-Forwarded-For",
			remoteAddr:    "127.0.0.1:12345",
			xForwardedFor: "192.168.1.100",
			userAgent:     "VLC/3.0.18",
			expectedIP:    "192.168.1.100",
			expectedUA:    "VLC/3.0.18",
		},
		{
			name:          "X-Real-IP takes priority over X-Forwarded-For",
			remoteAddr:    "127.0.0.1:12345",
			xRealIP:       "192.168.1.100",
			xForwardedFor: "10.0.0.1",
			userAgent:     "VLC/3.0.18",
			expectedIP:    "192.168.1.100",
			expectedUA:    "VLC/3.0.18",
		},
		{
			name:       "Empty User-Agent",
			remoteAddr: "192.168.1.100:12345",
			userAgent:  "",
			expectedIP: "192.168.1.100:12345",
			expectedUA: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "http://example.com", nil)
			if err != nil {
				t.Fatal(err)
			}

			req.RemoteAddr = tt.remoteAddr
			if tt.xRealIP != "" {
				req.Header.Set("X-Real-IP", tt.xRealIP)
			}
			if tt.xForwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tt.xForwardedFor)
			}
			if tt.userAgent != "" {
				req.Header.Set("User-Agent", tt.userAgent)
			}

			clientID := ExtractClientIdentifier(req)

			if clientID.IP != tt.expectedIP {
				t.Errorf("Expected IP %s, got %s", tt.expectedIP, clientID.IP)
			}

			if clientID.UserAgent != tt.expectedUA {
				t.Errorf("Expected User-Agent %s, got %s", tt.expectedUA, clientID.UserAgent)
			}
		})
	}
}

func TestConcurrency(t *testing.T) {
	m := NewManager()
	streamID := "test-stream-123"

	// Create sessions concurrently
	done := make(chan bool)
	for i := 0; i < 100; i++ {
		go func() {
			clientID := ClientIdentifier{
				IP:        "192.168.1.100",
				UserAgent: "VLC/3.0.18",
			}
			pid := m.GetOrCreatePID(streamID, clientID)
			if pid <= 0 {
				t.Errorf("Invalid PID: %d", pid)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}

	// All requests from same client should get same PID
	if m.GetActiveSessions() != 1 {
		t.Errorf("Expected 1 active session, got %d", m.GetActiveSessions())
	}
}

func TestPIDIncrement(t *testing.T) {
	m := NewManager()
	streamID := "test-stream-123"

	// Create 10 different sessions
	pids := make([]int, 10)
	for i := 0; i < 10; i++ {
		clientID := ClientIdentifier{
			IP:        "192.168.1." + string(rune(100+i)),
			UserAgent: "VLC/3.0.18",
		}
		pids[i] = m.GetOrCreatePID(streamID, clientID)
	}

	// Verify PIDs are sequential
	for i := 0; i < 10; i++ {
		if pids[i] != i+1 {
			t.Errorf("Expected PID %d, got %d", i+1, pids[i])
		}
	}
}
