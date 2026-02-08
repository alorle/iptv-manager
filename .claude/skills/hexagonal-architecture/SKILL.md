---
name: hexagonal-architecture
description: Hexagonal architecture and SOLID design principles for Go. Use when designing new packages, creating ports/adapters, refactoring architecture, or reviewing layer boundaries.
user-invocable: false
---

# Hexagonal Architecture — Design References

When making architectural decisions, follow the principles from these references:

## References

- [SOLID Go Design](https://dave.cheney.net/2016/08/20/solid-go-design) — Dave Cheney
- [Introducing Clean Architecture](https://threedots.tech/post/introducing-clean-architecture/) — Three Dots Labs
- [Ready for Changes with Hexagonal Architecture](https://netflixtechblog.com/ready-for-changes-with-hexagonal-architecture-b315ec967749) — Netflix
- [Hexagonal Architecture in Go](https://medium.com/@matiasvarela/hexagonal-architecture-in-go-cfd4e436faa3) — Matías Varela

## Key principles

### Ports & Adapters

- **Ports** are interfaces that define how actors communicate with the core. They belong to the core.
  - **Driver ports**: actions the core provides (use cases). Defined near `application/`.
  - **Driven ports**: actions the core needs from the outside (repos, services). Defined near the consumer.
- **Adapters** transform between external requests and the core's port interfaces.
- Swap any adapter without touching business logic.

### SOLID in Go

- **Single Responsibility**: one reason to change per package.
- **Open/Closed**: extend via embedding and interfaces, don't modify existing types.
- **Liskov Substitution**: "require no more, promise no less" — implementations must honor the full interface contract.
- **Interface Segregation**: depend on the narrowest interface needed (`io.Reader` over `*os.File`).
- **Dependency Inversion**: application depends on port interfaces, never on concrete adapters.

### Clean Architecture layering

- Domain packages live directly under `internal/` — one per bounded context (e.g. `internal/channel/`, `internal/playlist/`). There is NO `internal/domain/` folder.
- Domain is pure: no framework deps, no imports from other layers.
- Application orchestrates domain logic using port interfaces.
- Adapters are infrastructure — databases, HTTP, CLI, messaging.
- Wiring (dependency injection) happens in `main.go` only.
- Keep import graphs acyclic and flat. Push specifics toward `main`.
