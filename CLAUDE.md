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

## Important Architectural Patterns

### Strict Server Interface

The API uses oapi-codegen's "strict server" pattern. This means:
- Handlers receive typed request objects, not raw `http.Request`
- Handlers return typed response objects, not write to `http.ResponseWriter`
- Request parsing and response serialization are handled by generated code
- All validation is declarative in `openapi.yaml`

When implementing API endpoints in `internal/api/channels.go`, work with domain types only.

### Extension-Based Routing

The routing logic in `main.go` uses file extensions to determine handler:
- No extension or special paths (`/api/*`, `*.m3u`) → API router
- Has extension (`.js`, `.css`, images) → Vite handler
- Special case: `/` and `/index.html` always go to Vite handler

When adding new API endpoints, avoid file extensions in the path.

### M3U Flattening Logic

The `playlist.go` handler implements critical business logic:
- Tracks title occurrences with a map to handle duplicate channel names
- Generates incremental numbering (#2, #3, etc.) for multiple streams of the same channel
- Preserves EPG metadata (guideId, logo, groupTitle) across all streams
- Each stream gets a unique URL with Acestream parameters

This flattening is necessary because M3U format is flat (one entry per stream) but our data model is hierarchical (channels contain streams).

### UUID Generation Strategy

UUIDs are generated at load time (in `config.go`), not stored in JSON:
- Keeps `streams.json` human-editable (no need to manually create UUIDs)
- Provides type safety in API (OpenAPI spec requires UUID format)
- Each application restart generates new UUIDs (acceptable for in-memory data)
- Stream UUIDs inherit their parent channel's ID for referential integrity

### Dependency Injection Flow

Dependencies are wired in `main.go`:
```go
// 1. Load data from JSON
channels := loadChannels(streamsFile)

// 2. Create repository with data
repo := internalJson.NewInMemoryChannelsRepository(channels)

// 3. Create use case with repository
useCase := usecase.NewChannelsUseCase(repo)

// 4. Inject use case into handlers
server := api.NewServer(useCase)
playlistHandler := handlers.NewPlaylistHandler(useCase, ...)
```

This makes the system testable and allows swapping implementations (e.g., adding a database repository).

## Common Development Tasks

### Updating Dependencies

```bash
# Update Go dependencies
go get -u ./...
go mod tidy
go generate ./internal/api/server.go  # Regenerate after oapi-codegen updates
go build -o ./.tmp/main .

# Update npm dependencies
npm update
npm outdated  # Check for major version updates
npm install <package>@latest  # Install major updates
npx openapi-typescript openapi.yaml -o src/lib/api/v1.d.ts  # Regenerate types
npm run build
```

Note: If you encounter Go dependency conflicts (especially with `yaml-jsonpath` and `go-yit`), you may need to pin `go-yit` to a compatible version.

### Adding a New API Endpoint

1. Update `openapi.yaml` with new endpoint definition
2. Regenerate backend code: `go generate ./internal/api/server.go`
3. Implement handler in `internal/api/*.go` (you'll get compile errors until you do)
4. Regenerate frontend types: `npx openapi-typescript openapi.yaml -o src/lib/api/v1.d.ts`
5. Update frontend components to use new endpoint

The strict server interface ensures you can't forget to implement the endpoint—you'll get a compile error.

### Adding a Custom Handler (Outside OpenAPI)

For endpoints that don't fit OpenAPI (like `/playlist.m3u`):
1. Create handler in `internal/handlers/`
2. Register in `main.go` router before the Vite handler
3. Ensure the path doesn't conflict with extension-based routing logic

### Modifying the Data Model

1. Update domain entities in `internal/channel.go`
2. Update JSON structs in `config.go` if storage format changes
3. Update `openapi.yaml` to reflect new API schema
4. Regenerate both backend and frontend code
5. Update repository implementations in `internal/memory/`

## Troubleshooting

### Air doesn't restart after Go file changes
- Check `.air.toml` excluded directories
- Ensure you're editing files with `.go` extension
- Air doesn't watch `src/` (frontend) or `node_modules/`

### Vite dev server not found
- Ensure `npm run dev` is running on port 5173
- Backend must be started with `-dev` flag or via `air`
- Check `vite.config.ts` doesn't override default port

### ESLint errors in `.direnv/` directory
- The `eslint.config.js` should ignore `.direnv` (it contains Go module cache)
- Pattern: `{ ignores: ['dist', '.direnv'] }`

### TypeScript types don't match API response
- Regenerate types: `npx openapi-typescript openapi.yaml -o src/lib/api/v1.d.ts`
- Ensure `openapi.yaml` matches backend implementation
- Check if you forgot to regenerate after updating OpenAPI spec

### Go generate fails with YAML errors
- Usually a dependency version conflict (yaml.v3 vs yaml.v4)
- Check `go mod graph | grep yaml` to debug
- May need to pin `go-yit` to compatible version

## File Organization Reference

**Do not edit (generated)**:
- `internal/api/api.gen.go` - Generated from OpenAPI spec
- `src/lib/api/v1.d.ts` - Generated TypeScript types
- `dist/` - Built frontend assets

**Edit these for API changes**:
- `openapi.yaml` - Single source of truth for API contract
- `internal/api/channels.go` - API endpoint implementations
- `internal/handlers/*.go` - Custom (non-OpenAPI) handlers

**Edit these for business logic**:
- `internal/channel.go` - Domain entities (Channel, Stream)
- `internal/repository.go` - Repository interfaces
- `internal/usecase/*.go` - Application business rules
- `internal/memory/*.go` - In-memory data access

**Edit these for frontend**:
- `src/components/` - React components
- `src/lib/api/client.ts` - API client setup
- `src/App.tsx` - Application root

**Infrastructure**:
- `main.go` - Application entry point, routing, dependency injection
- `config.go` - JSON loading and parsing
- `.air.toml` - Live reload configuration
- `vite.config.ts` - Vite build configuration
- `Containerfile` - Container build (multi-stage with frontend + backend)
