# CLAUDE.md — iptv-manager

Go backend for managing IPTV playlists, channels and EPG data.

- **Module**: `github.com/alorle/iptv-manager`
- **Go version**: 1.25+ (see `go.mod`)
- **Environment**: managed with **direnv** (`.envrc` → `layout go`). Run `direnv allow .` before any Go command.

## Architecture — Hexagonal / Ports & Adapters

IMPORTANT: This project follows hexagonal architecture. All code changes MUST respect these rules.

### Package layout

```
internal/
├── <entity>/        # Domain layer: pure business types, value objects, domain errors.
│                    # One package per bounded context / aggregate (e.g. channel/, playlist/).
│                    # NO imports from application, port, or adapter. NO framework deps.
├── application/     # Use cases / services. Depends ONLY on domain packages + port interfaces.
├── port/
│   ├── driven/      # Interfaces the app NEEDS (repos, external services)
│   └── driver/      # Interfaces the app EXPOSES (HTTP, CLI, gRPC)
└── adapter/
    ├── driven/      # Concrete implementations: DB repos, HTTP clients, etc.
    └── driver/      # Concrete implementations: HTTP server, CLI, etc.
cmd/
└── <binary>/        # main.go — wiring ONLY (instantiate adapters, inject into services)
```

Domain packages live directly under `internal/` — there is NO `internal/domain/` folder. Each bounded context gets its own package (e.g. `internal/channel/`, `internal/playlist/`).

### Dependency rules

- Allowed: `adapter → port → application → internal/<entity>`
- FORBIDDEN: domain packages importing `application`, `port`, or `adapter`
- Domain packages are pure: no framework deps, no `context.Context` in domain types
- Ports are Go interfaces defined close to the consumer
- Accept interfaces, return structs
- Wiring in `main.go` only — no global state, no `init()` side effects
- No `utils`, `helpers`, or `common` packages — name packages after what they provide
- Domain-specific error types and sentinels live in their domain package under `internal/`

## Verification

IMPORTANT: YOU MUST run verification after every code change. Do NOT mark a task as done unless ALL checks pass.

Run `/verify` to execute the full checklist (format, vet, lint, test, build). Fix issues before reporting completion.

## Workflow

1. Read and understand existing code before modifying it. Verify the change fits the architecture.
2. After every code change, run `/verify`.
3. Never introduce imports that violate the dependency direction rule.
4. Prefer the standard library. Vet new dependencies carefully; run `go mod tidy` after changes.
