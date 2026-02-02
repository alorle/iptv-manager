# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

IPTV Manager is a Go service that proxies M3U playlists from IPFS sources, rewrites acestream:// URLs to player-compatible format, and provides stream multiplexing so multiple clients can share a single upstream connection to the Acestream Engine.

## Development Commands

### Backend (Go)

```bash
# Run with hot reload (requires air)
air

# Run directly
go run .

# Run tests
go test ./...

# Run specific test
go test ./multiplexer -run TestServeStream

# Vet
go vet ./...
```

### Frontend (React/Vite in web-ui/)

```bash
cd web-ui

# Development server
npm run dev

# Build for production
npm run build

# Run tests
npm test

# Lint and format check
npm run lint
npm run format
```

### Container

```bash
podman build -f Containerfile -t iptv-manager .
```

## Architecture

```
                    ┌────────────────────────────────────────┐
                    │              main.go                   │
                    │  HTTP server, routes, config loading   │
                    └───────────────────┬────────────────────┘
                                        │
    ┌───────────────┬───────────────────┼────────────────────┬───────────────┐
    ▼               ▼                   ▼                    ▼               ▼
┌─────────┐   ┌──────────┐       ┌────────────┐      ┌───────────┐   ┌──────────┐
│ fetcher │   │ rewriter │       │ multiplexer│      │    api    │   │    ui    │
│         │   │          │       │            │      │           │   │ (embed)  │
│ HTTP +  │   │ M3U URL  │       │ Stream fan-│      │ REST API  │   │ React SPA│
│ cache   │   │ rewrite  │       │ out to     │      │ channels  │   │ served   │
│ fallback│   │ dedup    │       │ clients    │      │ overrides │   │ via      │
│         │   │ sort     │       │            │      │           │   │ go:embed │
└────┬────┘   └──────────┘       └─────┬──────┘      └─────┬─────┘   └──────────┘
     │                                 │                   │
     ▼                                 ▼                   ▼
┌─────────┐                    ┌───────────────┐    ┌───────────┐
│  cache  │                    │circuitbreaker │    │ overrides │
│         │                    │ + ringbuffer  │    │           │
│ File    │                    │               │    │ YAML      │
│ storage │                    │ Resilience    │    │ persist   │
│ + TTL   │                    │ for upstream  │    │           │
└─────────┘                    └───────────────┘    └───────────┘
```

### Key Modules

| Module | Purpose |
|--------|---------|
| `multiplexer/` | Core streaming: single upstream connection serves multiple clients. Uses ring buffer during reconnection to prevent data loss. |
| `circuitbreaker/` | Prevents cascade failures when Acestream Engine is unavailable. Exponential backoff on reconnection. |
| `rewriter/` | M3U processing pipeline: URL rewriting, stream deduplication by acestream ID, alphabetical sorting by group-title then name. |
| `overrides/` | Channel metadata customization (tvg-id, tvg-name, group-title, enabled). Persisted to `$CACHE_DIR/overrides.yaml`. |
| `fetcher/` | HTTP client with cache fallback - serves stale cache when upstream (IPFS) is unreachable. |
| `pidmanager/` | Manages unique PIDs per client connection for Acestream session tracking. |
| `ui/` | Embeds built React frontend (`web-ui/dist`) into the binary via `go:embed`. |

### M3U Processing Pipeline

Unified playlist (`/playlist.m3u`) processing order:
1. Fetch from IPFS sources (elcano + newera)
2. Merge content
3. Apply channel overrides (disable/metadata changes)
4. Deduplicate by acestream ID
5. Sort by group-title, then display name
6. Rewrite acestream:// URLs to `/stream?id=...`

### Stream Multiplexing Flow

1. Client requests `/stream?id={contentID}`
2. Multiplexer checks for existing stream or creates new upstream connection
3. Data read from upstream is written to ring buffer (resilience) and fanned out to all clients
4. On upstream disconnect: circuit breaker controls reconnection with exponential backoff
5. During reconnection: new clients receive buffered data from ring buffer
6. When last client disconnects, stream is closed

## Environment Variables

Required:
- `CACHE_DIR`: Directory for playlist cache and overrides
- `CACHE_TTL`: Cache time-to-live (e.g., `1h`)

Optional (with defaults):
- `HTTP_ADDRESS`: `127.0.0.1`
- `HTTP_PORT`: `8080`
- `ACESTREAM_ENGINE_URL`: `http://127.0.0.1:6878`
- `PROXY_READ_TIMEOUT`: `5s`
- `PROXY_WRITE_TIMEOUT`: `10s`
- `PROXY_BUFFER_SIZE`: `4194304` (4MB)

## Testing Patterns

- Tests are colocated with implementation (`*_test.go`)
- Use `t.Parallel()` for concurrent test execution where appropriate
- Multiplexer tests use mock HTTP servers for upstream simulation
- Table-driven tests are preferred for multiple scenarios

## Manual E2E Testing with Browser MCPs

For manual end-to-end testing, use the chrome-devtools or playwright MCP tools to interact with the running application.

### Setup

1. Start the backend with frontend:
```bash
# Terminal 1: Build frontend and run backend
cd web-ui && npm run build && cd .. && air
```

2. The app will be available at `http://localhost:8080`

### Using Browser MCPs

**Navigate to the app:**
```
mcp__playwright__browser_navigate or mcp__chrome-devtools__navigate_page
URL: http://localhost:8080
```

**Take snapshots for inspection:**
```
mcp__playwright__browser_snapshot or mcp__chrome-devtools__take_snapshot
```

**Interact with elements** (use refs/uids from snapshot):
```
mcp__playwright__browser_click or mcp__chrome-devtools__click
mcp__playwright__browser_type or mcp__chrome-devtools__fill
```

### Key Test Scenarios

| Endpoint | What to verify |
|----------|----------------|
| `/` | React UI loads, channel list renders |
| `/health` | Returns "OK" |
| `/playlist.m3u` | Returns M3U with rewritten URLs (`/stream?id=...`) |
| `/api/channels` | JSON list of channels with override info |
| `/metrics` | Prometheus metrics (streams_active, clients_connected) |

### Testing Channel Overrides via API

```bash
# List channels
curl http://localhost:8080/api/channels

# Disable a channel
curl -X PATCH http://localhost:8080/api/channels/{acestream_id} \
  -H "Content-Type: application/json" \
  -d '{"enabled": false}'

# Verify channel is filtered from playlist
curl http://localhost:8080/playlist.m3u | grep {acestream_id}  # should be empty
```
