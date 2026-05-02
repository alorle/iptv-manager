package application

import (
	"sync/atomic"
	"time"
)

// streamCounters tracks lifecycle events for stream sessions using atomic counters.
// All counters reset on process restart, which is intentional — the diagnostic
// value lies in comparing values since the last restart.
type streamCounters struct {
	streamsStarted        atomic.Int64
	streamStartFailures   atomic.Int64
	streamsStopped        atomic.Int64
	streamStopFailures    atomic.Int64
	reconnectionAttempts  atomic.Int64
	reconnectionSuccesses atomic.Int64
	clientsServed         atomic.Int64
}

func (c *streamCounters) snapshot() StreamCountersSnapshot {
	return StreamCountersSnapshot{
		StreamsStarted:        c.streamsStarted.Load(),
		StreamStartFailures:   c.streamStartFailures.Load(),
		StreamsStopped:        c.streamsStopped.Load(),
		StreamStopFailures:    c.streamStopFailures.Load(),
		ReconnectionAttempts:  c.reconnectionAttempts.Load(),
		ReconnectionSuccesses: c.reconnectionSuccesses.Load(),
		ClientsServed:         c.clientsServed.Load(),
	}
}

// StreamDiagnostics is the complete diagnostic snapshot returned by Diagnostics().
type StreamDiagnostics struct {
	Uptime   time.Duration          `json:"uptime"`
	Counters StreamCountersSnapshot `json:"counters"`
	Sessions []SessionDiagnostic    `json:"sessions"`
	EngineOK bool                   `json:"engine_healthy"`
}

// StreamCountersSnapshot is a point-in-time read of all lifecycle counters.
type StreamCountersSnapshot struct {
	StreamsStarted        int64 `json:"streams_started"`
	StreamStartFailures   int64 `json:"stream_start_failures"`
	StreamsStopped        int64 `json:"streams_stopped"`
	StreamStopFailures    int64 `json:"stream_stop_failures"`
	ReconnectionAttempts  int64 `json:"reconnection_attempts"`
	ReconnectionSuccesses int64 `json:"reconnection_successes"`
	ClientsServed         int64 `json:"clients_served"`
}

// LeakedSessions returns the number of sessions that were started but never stopped.
func (s StreamCountersSnapshot) LeakedSessions() int64 {
	return s.StreamsStarted - s.StreamsStopped - s.StreamStopFailures
}

// SessionDiagnostic describes the internal state of a single stream session.
type SessionDiagnostic struct {
	InfoHash    string    `json:"info_hash"`
	State       string    `json:"state"`
	StreamURL   string    `json:"stream_url,omitempty"`
	EnginePID   string    `json:"engine_pid,omitempty"`
	Clients     []string  `json:"clients"`
	ClientCount int       `json:"client_count"`
	Error       string    `json:"error,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}
