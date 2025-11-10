# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

IPTV Manager is a self-hosted web application for managing Acestream streams with EPG integration. The application enables users to organize streaming sources, match them with EPG metadata, and generate M3U playlists compatible with Jellyfin and similar IPTV clients.

**Current State**: Minimal MVP on `redesign` branch with health check endpoint. See [docs/PRD.md](docs/PRD.md) for comprehensive feature roadmap moving to stream-centric architecture.

**Tech Stack**: Go backend + React 19 frontend, both served from single binary with embedded static assets.

## Architecture

### Backend (Go)

Clean Architecture with minimal complexity:

- **API Layer** ([internal/api/](internal/api/)): HTTP handlers generated from [openapi.yaml](openapi.yaml)
  - [api.gen.go](internal/api/api.gen.go): Generated using oapi-codegen with strict server interface (DO NOT EDIT)
  - [health.go](internal/api/health.go): Health check endpoint implementation
  - [server.go](internal/api/server.go): Server setup and interface compliance
  - [cfg.yaml](internal/api/cfg.yaml): Code generation config
- **Domain Layer** ([internal/](internal/)): Domain entities and repository interfaces
  - [channel.go](internal/channel.go): Stream and Channel entities (stream-centric model)
  - [repository.go](internal/repository.go): Repository interfaces

### Frontend (React + TypeScript)

- Built with Vite, React 19, TanStack Query
- **Type-safe API client**: Uses `openapi-react-query` with `openapi-fetch`
  - TypeScript types generated from OpenAPI spec in [src/lib/api/v1.d.ts](src/lib/api/v1.d.ts) (DO NOT EDIT)
  - API client configured in [src/lib/api/client.ts](src/lib/api/client.ts)
- Tailwind CSS v4 via `@tailwindcss/vite`
- Path alias: `@/` → `./src/`

### Dual-Mode Server

The Go application in [main.go](main.go) serves both API and frontend:

- **Development** (`-dev` flag or via `air`): Proxies to Vite dev server at `http://localhost:5173`
- **Production**: Serves embedded static files from `dist/` via `//go:embed all:dist`
- **Routing**: Extension-based (no extension = API, has extension = static asset)

## Development Commands

### Full Stack Development Workflow

**Normal development** (two terminals):

```bash
# Terminal 1: Vite dev server
npm run dev

# Terminal 2: Go server with live reload
air
```

This is the recommended approach - changes to both frontend and backend will auto-reload.

### Backend Only

```bash
# Build backend
go build -o ./.tmp/main .

# Run manually in dev mode
go run . -dev

# Generate API code from OpenAPI spec
go generate ./internal/api/server.go
```

### Frontend Only

```bash
# Build frontend for production
npm run build

# Type-check only
tsc -b

# Lint and format
npm run lint
npm run format

# Run tests
npm run test
npm run test:watch
npm run test:ui
npm run test:coverage

# Preview production build
npm run preview

# Generate TypeScript types from OpenAPI spec
npm run generate:api
```

## Configuration

Environment variables (see [.env.example](.env.example)):

- `HTTP_ADDRESS`: Server bind address (default: `0.0.0.0`)
- `HTTP_PORT`: Server port (default: `8080`)

## Tooling & Quality Assurance

### Code Quality Tools

**Linting:**

- **Frontend**: ESLint with TypeScript, React Hooks, accessibility (jsx-a11y), and import rules
  - Config: [eslint.config.js](eslint.config.js)
  - Run: `npm run lint` or `npm run lint:fix`
- **Backend**: golangci-lint with 15+ linters (gofmt, govet, errcheck, staticcheck, gosec, revive, etc.)
  - Config: [.golangci.yml](.golangci.yml)
  - Run: `golangci-lint run` or `make lint`

**Formatting:**

- **Frontend**: Prettier with consistent rules (semi, no trailing commas, 100 char width)
  - Config: [.prettierrc](.prettierrc), [.prettierignore](.prettierignore)
  - Run: `npm run format` or `npm run format:check`
- **Backend**: gofmt + goimports (automatic via VSCode on save)
  - Run: `make format`
- **Cross-editor**: EditorConfig ensures tabs for Go, spaces for TS/JS/JSON/YAML
  - Config: [.editorconfig](.editorconfig)

**Type Checking:**

- TypeScript with strict mode enabled
- Run: `npm run typecheck` or `npx tsc -b`

**Testing:**

- **Frontend**: Vitest with React Testing Library
  - Config: [vitest.config.ts](vitest.config.ts), [src/test/setup.ts](src/test/setup.ts)
  - Test utilities: [src/test/utils.tsx](src/test/utils.tsx) (custom render with providers)
  - Run: `npm run test` (single run), `npm run test:watch` (watch mode)
  - UI: `npm run test:ui` (opens Vitest UI)
  - Coverage: `npm run test:coverage`
  - Test files are co-located with source files (e.g., `Component.test.tsx` next to `Component.tsx`)
- **Backend**: Go standard testing package
  - Run: `go test ./...` or `make test-backend`

### VSCode Integration

Recommended extensions ([.vscode/extensions.json](.vscode/extensions.json)):

- golang.go (Go support with gopls)
- esbenp.prettier-vscode (Prettier formatting)
- dbaeumer.vscode-eslint (ESLint integration)
- bradlc.vscode-tailwindcss (Tailwind IntelliSense)
- redhat.vscode-yaml (OpenAPI spec editing)
- humao.rest-client (API testing)
- editorconfig.editorconfig (EditorConfig support)

**Debugging** ([.vscode/launch.json](.vscode/launch.json)):

- "Launch Backend (Dev Mode)" - Start backend with `-dev` flag
- "Attach to Frontend (Chrome)" - Debug React app in Chrome
- "Full Stack Debug" - Debug both simultaneously

**Tasks** ([.vscode/tasks.json](.vscode/tasks.json)):

- Generate API code (backend + frontend)
- Build (frontend, backend, or both)
- Format all code
- Lint (frontend + backend)
- Type check
- Run tests

**Settings** ([.vscode/settings.json](.vscode/settings.json)):

- Format on save with Prettier (frontend) and goimports (backend)
- Auto-fix ESLint issues on save
- gopls configuration with advanced analyses and hints
- Tailwind IntelliSense for `cn()` utility

### Git Hooks (Lefthook)

Pre-commit hooks automatically run on `git commit` ([.lefthook.yml](.lefthook.yml)):

- Lint and format staged frontend files
- Format and lint staged Go files
- Stage fixed files automatically

Pre-push hooks run before `git push`:

- TypeScript type checking
- Frontend tests
- Frontend build
- Backend tests
- Backend build

**Setup**: Run `npx lefthook install` (or `npm install` with prepare script)

### Makefile Commands

Common tasks ([Makefile](Makefile)):

```bash
make help                    # Show all available commands
make install                 # Install Go + npm dependencies
make dev-backend             # Start backend with air (live reload)
make dev-frontend            # Start Vite dev server
make build                   # Build both frontend and backend
make test                    # Run all tests (frontend + backend)
make test-frontend           # Run frontend tests only
make test-frontend-watch     # Run frontend tests in watch mode
make test-frontend-ui        # Run frontend tests with UI
make test-frontend-coverage  # Run frontend tests with coverage
make test-backend            # Run backend tests only
make test-coverage           # Run all tests with coverage
make lint                    # Run all linters (frontend + backend)
make format                  # Format all code
make generate                # Generate code from OpenAPI spec
make clean                   # Clean build artifacts
make ci                      # Run all CI checks locally
```

### CI/CD Pipeline

GitHub Actions workflow ([.github/workflows/ci.yaml](.github/workflows/ci.yaml)) runs on PRs and main/redesign branches:

**Frontend checks:**

- ESLint
- Prettier format check
- TypeScript type checking
- Vitest tests
- Build verification

**Backend checks:**

- Go build
- Go tests with race detector
- golangci-lint

**OpenAPI validation:**

- Validates OpenAPI spec syntax

All checks must pass before merging.

## Code Generation Workflow

**Critical**: This project uses OpenAPI as the single source of truth. Always follow this sequence:

1. **Define the API contract**: Edit [openapi.yaml](openapi.yaml)
2. **Generate backend code**: `go generate ./internal/api/server.go`
3. **Implement handlers**: Add implementation in [internal/api/](internal/api/) (you'll get compile errors until you do)
4. **Generate frontend types**: `npm run generate:api`
5. **Update frontend**: Use the `$api` client from [src/lib/api/client.ts](src/lib/api/client.ts) in React components

**Tools used**:

- Backend: `oapi-codegen` (managed as Go tool dependency in [go.mod](go.mod))
- Frontend: `openapi-typescript` (generates types), `openapi-react-query` + `openapi-fetch` (type-safe hooks and client)

**Never edit generated files**: [internal/api/api.gen.go](internal/api/api.gen.go) and [src/lib/api/v1.d.ts](src/lib/api/v1.d.ts) are regenerated from spec.

## Using the API Client

The frontend uses `openapi-react-query` for type-safe API calls:

```tsx
import { $api } from "@/lib/api/client";

function MyComponent() {
  // Type-safe query with auto-complete for paths and parameters
  const { data, isLoading, isError } = $api.useQuery(
    "get",
    "/health",
    {},
    {
      refetchInterval: 5000,
    }
  );

  // data is fully typed based on OpenAPI schema
  if (data) {
    console.log(data.status, data.version, data.timestamp);
  }
}
```

For mutations:

```tsx
const { mutate } = $api.useMutation("post", "/streams");
mutate({
  body: { name: "My Stream", url: "..." },
});
```

## Key Architectural Patterns

### Strict Server Interface

The API uses oapi-codegen's "strict server" pattern:

- Handlers receive **typed request objects**, not raw `http.Request`
- Handlers return **typed response objects**, not write to `http.ResponseWriter`
- Validation is **declarative in OpenAPI spec**, not in code
- You'll get **compile errors** if you forget to implement an endpoint

Example from [server.go](internal/api/server.go):

```go
// Server must implement StrictServerInterface
var _ StrictServerInterface = (*Server)(nil)
```

### Extension-Based Routing

The main handler in [main.go](main.go):78-100 routes requests:

- `/` or `/index.html` → Vite handler (always)
- No extension (e.g., `/api/health`) → API router
- Has extension (e.g., `/app.js`, `/logo.png`) → Vite handler

**Implication**: Don't use file extensions in API paths (e.g., `/api/data.json` would route to Vite, not API).

### Dependency Injection

Dependencies are wired in [main.go](main.go):68-75:

```go
server := api.NewServer()  // Add dependencies here as constructor args
h := api.NewStrictHandler(server, nil)
m := middleware.OapiRequestValidator(swagger)
```

Currently no dependencies (health check only), but pattern is ready for repositories/use cases.

## Updating Dependencies

```bash
# Go dependencies
go get -u ./...
go mod tidy
go generate ./internal/api/server.go  # Regenerate after oapi-codegen updates
go build -o ./.tmp/main .

# npm dependencies
npm update
npm outdated  # Check for major updates
npm install <package>@latest
npx openapi-typescript openapi.yaml -o src/lib/api/v1.d.ts  # Regenerate types
npm run build
```

## Common Issues

### Air doesn't restart

- Check [.air.toml](.air.toml):10 - excludes `src/`, `node_modules/`, `dist/`
- Air only watches `.go` files

### Vite dev server not found

- Ensure `npm run dev` is running on port 5173
- Backend must be started with `-dev` flag or via `air` (see [.air.toml](.air.toml):6)

### Type mismatch between frontend and backend

- Regenerate types: `npx openapi-typescript openapi.yaml -o src/lib/api/v1.d.ts`
- Verify [openapi.yaml](openapi.yaml) matches backend implementation

### YAML version conflicts

- Check `go mod graph | grep yaml`
- Common issue: `yaml.v3` vs `yaml.v4` conflicts
- May need to pin `go-yit` version

## File Organization

**Generated (never edit)**:

- [internal/api/api.gen.go](internal/api/api.gen.go)
- [src/lib/api/v1.d.ts](src/lib/api/v1.d.ts)
- `dist/`

**API contract (edit first)**:

- [openapi.yaml](openapi.yaml)

**Backend implementation**:

- [main.go](main.go) - Entry point, routing, dependency injection
- [internal/api/\*.go](internal/api/) - API handlers (except api.gen.go)
- [internal/\*.go](internal/) - Domain entities, repositories

**Frontend**:

- [src/App.tsx](src/App.tsx) - Root component
- [src/components/](src/components/) - React components
- [src/lib/utils.ts](src/lib/utils.ts) - Utilities

**Configuration**:

- [.air.toml](.air.toml) - Live reload config (backend)
- [vite.config.ts](vite.config.ts) - Vite build config (frontend)
- [package.json](package.json) - npm scripts and dependencies
- [go.mod](go.mod) - Go modules (note `tool` directive for oapi-codegen)
- [Containerfile](Containerfile) - Multi-stage Docker build

## Current Features

- **Health Check API**: `GET /api/health` returns `{"status": "healthy", "version": "1.0.0", "timestamp": "..."}`
- **Health Check UI**: Real-time status with 5s polling, dark mode support

## Next Steps (Roadmap)

See [docs/PRD.md](docs/PRD.md) for full feature requirements. Key upcoming features:

1. Stream management (CRUD operations)
2. Bulk stream import with parsing (text/JSON formats)
3. EPG integration and fuzzy matching
4. M3U playlist generation
5. Drag-and-drop stream reordering

The Clean Architecture foundation is ready - add layers (use cases, repositories, domain logic) as features grow.

## Testing Best Practices

### Frontend Testing Patterns

**1. Test File Organization**

Co-locate test files with source files:

```
src/
├── components/
│   ├── Health.tsx
│   └── Health.test.tsx
├── lib/
│   ├── utils.ts
│   └── utils.test.ts
```

**2. Test Utilities**

Use custom render from [src/test/utils.tsx](src/test/utils.tsx) for components that need providers:

```tsx
import { renderWithProviders, screen, userEvent } from "@/test/utils";

// Automatically wraps with QueryClientProvider
renderWithProviders(<MyComponent />);
```

**3. Mocking API Calls**

For components using `$api.useQuery` or `$api.useMutation`, mock the API client:

```tsx
import { vi } from "vitest";
import { $api } from "@/lib/api/client";

vi.mock("@/lib/api/client", () => ({
  $api: {
    useQuery: vi.fn(),
    useMutation: vi.fn(),
  },
}));

// In test
vi.mocked($api.useQuery).mockReturnValue({
  data: mockData,
  isLoading: false,
  isError: false,
} as any);
```

**4. Testing Hooks with Side Effects**

For hooks that use localStorage or other browser APIs, mock them in `beforeEach`:

```tsx
beforeEach(() => {
  localStorage.clear();
  Object.defineProperty(window, "matchMedia", {
    writable: true,
    value: vi.fn().mockImplementation((query) => ({
      matches: false,
      media: query,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    })),
  });
});
```

**5. Test Coverage Guidelines**

- **Utilities**: Aim for >90% coverage (pure functions are easy to test)
- **UI Components**: Focus on user interactions and accessibility
- **API-integrated components**: Test loading, error, and success states
- **Custom hooks**: Test state changes and side effects

**6. What to Test**

✅ **DO test:**

- Component renders correctly
- User interactions (clicks, typing, form submission)
- Conditional rendering based on props/state
- Accessibility attributes (aria-labels, roles)
- Integration with TanStack Query (loading, error, success states)

❌ **DON'T test:**

- Implementation details (internal state variable names)
- Third-party library internals
- Styling (unless critical for functionality)
- Generated code ([src/lib/api/v1.d.ts](src/lib/api/v1.d.ts), [internal/api/api.gen.go](internal/api/api.gen.go))

**7. Running Tests During Development**

```bash
# Watch mode for TDD workflow
npm run test:watch

# UI mode for interactive debugging
npm run test:ui

# Single run (used in CI)
npm run test

# With coverage
npm run test:coverage
```

### Backend Testing

Go tests follow standard Go testing practices:

- Unit tests for domain logic
- Integration tests for repositories
- Handler tests using httptest

Run with: `make test-backend` or `go test ./...`
