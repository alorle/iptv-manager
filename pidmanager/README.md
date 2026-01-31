# PID Manager

The PID Manager package provides unique player ID (PID) generation and session management for client-stream combinations in the IPTV Manager application.

## Overview

The Ace Stream Engine requires unique PIDs to track individual playback sessions. This package ensures that:
- Each client-stream combination gets a unique PID
- PIDs are reused when the same client reconnects to the same stream
- PIDs are properly released when clients disconnect
- Session cleanup can be performed to remove stale sessions

## Client Identification

Clients are identified by a combination of:
- **IP Address**: Extracted from `X-Real-IP`, `X-Forwarded-For`, or `RemoteAddr` headers
- **User-Agent**: The client's User-Agent string

This combination ensures that different devices/players from the same IP get separate PIDs, while allowing the same device to reconnect with the same PID.

## Usage

### Creating a Manager

```go
import "github.com/alorle/iptv-manager/pidmanager"

manager := pidmanager.NewManager()
```

### Getting or Creating a PID

```go
// Extract client information from HTTP request
clientID := pidmanager.ExtractClientIdentifier(r)

// Get or create PID for this client-stream combination
streamID := "ace123456789abcdef"
pid := manager.GetOrCreatePID(streamID, clientID)
```

### Releasing a PID

When a client disconnects:

```go
err := manager.ReleasePID(pid)
if err != nil {
    log.Printf("Error releasing PID: %v", err)
}
```

### Cleaning Up Disconnected Sessions

Periodically remove disconnected sessions:

```go
cleaned := manager.CleanupDisconnected()
log.Printf("Cleaned up %d disconnected sessions", cleaned)
```

### Getting Session Information

```go
session, err := manager.GetSession(pid)
if err != nil {
    log.Printf("Session not found: %v", err)
    return
}

fmt.Printf("Client: %s (%s)\n", session.ClientID.IP, session.ClientID.UserAgent)
fmt.Printf("Stream: %s\n", session.StreamID)
fmt.Printf("Connected: %v\n", session.Connected)
```

### Monitoring Active Sessions

```go
active := manager.GetActiveSessions()
fmt.Printf("Active sessions: %d\n", active)
```

## PID Generation

PIDs are generated as sequential integers starting from 1. The manager maintains internal state to ensure:
- No duplicate PIDs are issued
- PIDs are reused for reconnecting clients
- Thread-safe operations with concurrent access

## Thread Safety

All operations on the Manager are thread-safe and can be called concurrently from multiple goroutines.

## Example Integration

```go
// Initialize manager
pidManager := pidmanager.NewManager()

// In your HTTP handler
func streamHandler(w http.ResponseWriter, r *http.Request) {
    streamID := r.URL.Query().Get("id")
    clientID := pidmanager.ExtractClientIdentifier(r)

    // Get PID for this client-stream combination
    pid := pidManager.GetOrCreatePID(streamID, clientID)

    // Use PID when requesting stream from Ace Stream Engine
    params := aceproxy.GetStreamParams{
        ContentID: streamID,
        ProductID: strconv.Itoa(pid),
    }

    resp, err := aceClient.GetStream(r.Context(), params)
    // ... handle stream response

    // When client disconnects (use defer or error handling)
    defer func() {
        if err := pidManager.ReleasePID(pid); err != nil {
            log.Printf("Error releasing PID %d: %v", pid, err)
        }
    }()
}

// Periodic cleanup (run in background)
go func() {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()

    for range ticker.C {
        cleaned := pidManager.CleanupDisconnected()
        if cleaned > 0 {
            log.Printf("Cleaned up %d disconnected sessions", cleaned)
        }
    }
}()
```

## Session Lifecycle

1. **Client connects**: `GetOrCreatePID()` is called
   - If client-stream combination exists: reuse PID
   - If new combination: generate new PID
2. **Client streaming**: Session remains active with `Connected = true`
3. **Client disconnects**: `ReleasePID()` marks session as `Connected = false`
4. **Cleanup**: `CleanupDisconnected()` removes disconnected sessions

## Design Decisions

- **Incremental PIDs**: Simple, predictable, and compatible with Ace Stream Engine
- **IP + User-Agent**: Balances uniqueness with reconnection detection
- **Lazy Cleanup**: Sessions remain after disconnect for potential reconnection
- **Thread-safe maps**: Uses mutex to protect concurrent access
