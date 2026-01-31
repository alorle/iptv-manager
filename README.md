# IPTV Manager

Centralizes IPTV stream management with acestream URL rewriting, caching, and multiplexing.

## Overview

IPTV Manager is a Go service that proxies M3U playlists from IPFS sources, rewrites acestream:// URLs to player-compatible format, and caches content for reliability. It also provides stream multiplexing so multiple clients can share a single upstream connection to the Acestream Engine.

### Dependencies

- **Upstream**: Acestream Engine (required for stream playback)
- **Downstream**: Media players (VLC, Kodi, etc.)

## Local Development Setup

### Prerequisites

- Go 1.25+
- [Air](https://github.com/air-verse/air) for hot reload
- Acestream Engine running locally (or accessible via network)
- direnv (optional, for automatic env loading)

### Environment Variables

Copy `.env.example` to `.env` and configure:

| Variable               | Description                  | Default                 |
| ---------------------- | ---------------------------- | ----------------------- |
| `HTTP_ADDRESS`         | Listen address               | `127.0.0.1`             |
| `HTTP_PORT`            | Listen port                  | `8080`                  |
| `ACESTREAM_ENGINE_URL` | Acestream Engine URL         | `http://127.0.0.1:6878` |
| `CACHE_DIR`            | Directory for playlist cache | Required                |
| `CACHE_TTL`            | Cache time-to-live           | `1h`                    |
| `PROXY_READ_TIMEOUT`   | Stream read timeout          | `5s`                    |
| `PROXY_WRITE_TIMEOUT`  | Stream write timeout         | `10s`                   |
| `PROXY_BUFFER_SIZE`    | Stream buffer size (bytes)   | `4194304` (4MB)         |

### Running Locally

```bash
# Copy and configure environment
cp .env.example .env

# Run with hot reload
air

# Or run directly
go run .
```

### Running Tests

```bash
go test ./...
go vet ./...
```

## Architecture

```
┌─────────────┐     ┌──────────────────┐     ┌─────────────────┐
│   Client    │────▶│   IPTV Manager   │────▶│ Acestream Engine│
│ (VLC, Kodi) │     │                  │     │                 │
└─────────────┘     └──────────────────┘     └─────────────────┘
                            │
                    ┌───────┴───────┐
                    │  IPFS Source  │
                    │  (Playlists)  │
                    └───────────────┘
```

### Key Modules

| Path           | Purpose                                  |
| -------------- | ---------------------------------------- |
| `main.go`      | HTTP server and route handlers           |
| `cache/`       | File-based playlist caching with TTL     |
| `fetcher/`     | HTTP client with cache fallback          |
| `rewriter/`    | Acestream URL rewriting in M3U files     |
| `multiplexer/` | Stream multiplexing for multiple clients |
| `pidmanager/`  | PID management for Acestream sessions    |
| `overrides/`   | Channel override management              |
| `api/`         | REST API for channel management          |
| `aceproxy/`    | Acestream Engine client                  |

## API Endpoints

### Playlists

| Endpoint                    | Description                                 |
| --------------------------- | ------------------------------------------- |
| `GET /playlist.m3u`         | Unified playlist (rewritten acestream URLs) |
| `GET /playlists/elcano.m3u` | Elcano playlist (rewritten acestream URLs)  |
| `GET /playlists/newera.m3u` | NewEra playlist (rewritten acestream URLs)  |

### Streaming

| Endpoint                            | Description                      |
| ----------------------------------- | -------------------------------- |
| `GET /stream?id={contentID}`        | Stream proxy with multiplexing   |
| `GET /ace/getstream?id={contentID}` | Acexy-compatible stream endpoint |

Optional stream parameters: `transcode_audio`, `transcode_mp3`, `transcode_ac3`

### Channel Management

| Endpoint                                       | Description                          |
| ---------------------------------------------- | ------------------------------------ |
| `GET /health`                                  | Health check                         |
| `GET /api/channels`                            | List all channels with override info |
| `PATCH /api/channels/{acestream_id}`           | Create or update channel override    |
| `DELETE /api/channels/{acestream_id}/override` | Delete channel override              |

## Channel Overrides

Channel overrides allow you to customize channel metadata or disable channels from appearing in playlists. Overrides are persisted in `$CACHE_DIR/overrides.yaml` and applied automatically when playlists are generated.

### Override Fields

| Field         | Type    | Description                              |
| ------------- | ------- | ---------------------------------------- |
| `enabled`     | boolean | Set to `false` to hide channel from M3U  |
| `tvg_id`      | string  | EPG identifier                           |
| `tvg_name`    | string  | Display name in EPG                      |
| `tvg_logo`    | string  | Logo URL                                 |
| `group_title` | string  | Category/group for channel organization  |

### API Usage Examples

**List all channels:**

```bash
curl http://localhost:8080/api/channels
```

Response includes `has_override: true` for channels with active overrides.

**Disable a channel:**

```bash
curl -X PATCH http://localhost:8080/api/channels/{acestream_id} \
  -H "Content-Type: application/json" \
  -d '{"enabled": false}'
```

**Update channel metadata:**

```bash
curl -X PATCH http://localhost:8080/api/channels/{acestream_id} \
  -H "Content-Type: application/json" \
  -d '{"tvg_name": "Custom Name", "group_title": "Sports"}'
```

**Remove an override (restore original metadata):**

```bash
curl -X DELETE http://localhost:8080/api/channels/{acestream_id}/override
```

### Behavior Notes

- Only provided fields are updated; omitted fields retain their current values
- Disabled channels are completely filtered out from M3U playlists
- Orphaned overrides (for channels no longer in upstream sources) are automatically cleaned up when fresh data is fetched
- Overrides are thread-safe and persisted immediately to disk

## Deployment

### Container Build

```bash
podman build -f Containerfile -t iptv-manager .
podman run -p 8080:8080 -e CACHE_DIR=/cache -v ./cache:/cache iptv-manager
```

## Runbooks

### Clear Playlist Cache

```bash
rm -rf $CACHE_DIR/*.cache $CACHE_DIR/*.meta
```

### Check Acestream Engine Connection

```bash
curl http://127.0.0.1:6878/webui/api/service?method=get_version
```

### View Active Streams

Check multiplexer logs for active stream connections and client counts.

## Troubleshooting

### 502 Bad Gateway on Stream

**Symptom**: Stream endpoint returns 502
**Cause**: Cannot connect to Acestream Engine
**Fix**: Verify `ACESTREAM_ENGINE_URL` is correct and Engine is running

### Stale Playlist Content

**Symptom**: Playlist shows outdated channels
**Cause**: IPFS source unreachable, serving cached version
**Fix**: Check IPFS connectivity; clear cache to force refresh on next request
