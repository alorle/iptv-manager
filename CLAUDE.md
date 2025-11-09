# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

IPTV Manager is a dual-stack application (Go backend + React frontend) that manages Acestream channels and generates M3U playlists for IPTV clients. The application reads channel configurations from `streams.json` and provides both an API and a web interface for channel management.

## Architecture

### Backend (Go)

The backend follows Clean Architecture principles with clear separation of concerns:

- **Domain Layer** (`internal/`): Core business entities and repository interfaces
  - `Channel` struct: TV channel entity with EPG metadata (title, guideId, logo, groupTitle)
  - `Stream` struct: Access method for a channel (acestream_id, quality, tags, networkCaching)
  - One channel can have multiple streams
  - `Stream.FullTitle()`: Generates formatted title with quality and tags
  - `Stream.GetStreamURL()`: Constructs Acestream URL
  - `ChannelRepository` interface: Defines data access contract

- **Use Case Layer** (`internal/usecase/`): Application business rules
  - `GetChannelUseCase`: Orchestrates channel retrieval logic

- **Repository Layer** (`internal/memory/`): Data access implementations
  - In-memory repository that loads channels from JSON file at startup

- **API Layer** (`internal/api/`): HTTP handlers generated from OpenAPI spec
  - Generated using oapi-codegen with strict server interface
  - Code generation config in `internal/api/cfg.yaml`
  - Returns nested structure: channels with embedded streams array

- **Handlers** (`internal/handlers/`): Custom HTTP handlers not part of OpenAPI spec
  - `playlist.go`: Generates M3U playlist by flattening channels→streams with incremental numbering
  - `documentation.go`: Serves OpenAPI spec

- **M3U Package** (`internal/m3u/`): M3U playlist encoding with TVG tags support

### Frontend (React + TypeScript)

- Built with Vite and React 19
- Uses TanStack Query for data fetching and caching
- TypeScript types generated from OpenAPI spec (stored in `src/lib/api/v1.d.ts`)
- Components in `src/components/` follow a modular structure

### Dual-Mode Server

The Go application serves both API and frontend:
- In development mode (`-dev` flag): Proxies to Vite dev server at `http://localhost:5173`
- In production: Serves embedded static files from `dist/` directory
- Uses `github.com/olivere/vite` package for Vite integration

## Development Commands

### Backend

```bash
# Run with live reload (Air)
air

# Build backend
go build -o ./.tmp/main .

# Run manually in dev mode
go run . -dev

# Generate API code from OpenAPI spec
go generate ./internal/api/server.go
```

### Frontend

```bash
# Start Vite dev server
npm run dev

# Build frontend for production
npm run build

# Type-check
tsc -b

# Lint
npm run lint

# Preview production build
npm run preview

# Generate TypeScript types from OpenAPI spec
npx openapi-typescript openapi.yaml -o src/lib/api/v1.d.ts
```

### Full Stack Development

For active development, run both simultaneously:
1. Terminal 1: `npm run dev` (Vite dev server on port 5173)
2. Terminal 2: `air` (Go server with live reload on configured port)

## Configuration

Environment variables (see `.env.example`):
- `HTTP_ADDRESS`: Server bind address (default: `0.0.0.0`)
- `HTTP_PORT`: Server port (default: `8080`)
- `STREAMS_FILE`: Path to channels JSON file (default: `streams.json`)
- `ACESTREAM_URL`: Acestream engine URL (default: `http://127.0.0.1:6878/ace/getstream`)
- `EPG_URL`: Optional EPG (Electronic Program Guide) URL

## Data Format

Channels are defined in `streams.json` with nested structure:
```json
{
  "channels": [
    {
      "title": "Channel Name",
      "guideId": "EPG ID",
      "logo": "http://example.com/logo.png",
      "groupTitle": "Category",
      "streams": [
        {
          "acestream_id": "hex_hash",
          "quality": "FHD",
          "tags": ["TAG1", "TAG2"],
          "networkCaching": 10000
        }
      ]
    }
  ]
}
```

**Important Business Logic:**
- A **Channel** represents a TV channel with EPG metadata (for program guides)
- A **Stream** represents a way to access that channel (multiple streams per channel for different qualities/sources)
- The `/playlist.m3u` endpoint flattens this structure: one M3U entry per stream, with incremental numbering
  - First stream: "Channel Name [FHD] [TAG1]"
  - Second stream: "Channel Name (#2) [FHD] [TAG2]"
  - Nth stream: "Channel Name (#N) [quality] [tags]"

## Code Generation

This project uses code generation for API contracts:

1. **Backend**: OpenAPI → Go server code
   - Trigger: `go generate ./internal/api/server.go`
   - Source: `openapi.yaml`
   - Output: `internal/api/api.gen.go`
   - Tool: oapi-codegen (managed as Go tool dependency)

2. **Frontend**: OpenAPI → TypeScript types
   - Trigger: `npx openapi-typescript openapi.yaml -o src/lib/api/v1.d.ts`
   - Source: `openapi.yaml`
   - Output: `src/lib/api/v1.d.ts`

When modifying the API, update `openapi.yaml` first, then regenerate both backend and frontend code.

## Routing

- `/` - Frontend application (served by Vite handler)
- `/api/channels` - REST API endpoints (OpenAPI spec)
- `/api/documentation.json` - OpenAPI spec JSON
- `/playlist.m3u` - M3U playlist generator

The main handler in `main.go` routes requests based on path and file extension to either the API router or Vite handler.

## Key Dependencies

### Backend
- `oapi-codegen`: OpenAPI code generation
- `kin-openapi`: OpenAPI spec parsing
- `nethttp-middleware`: Request validation middleware
- `olivere/vite`: Vite integration for Go

### Frontend
- `@tanstack/react-query`: Data fetching and state management
- `openapi-fetch` + `openapi-react-query`: Type-safe API client from OpenAPI spec
- `react-player`: Video player component
