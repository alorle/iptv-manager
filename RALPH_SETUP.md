# Ralph Wiggum Clean Architecture Setup

This repository is configured for automated technical debt resolution using the Ralph Wiggum technique.

## What is Ralph Wiggum?

Ralph Wiggum is an infinite loop technique that repeatedly feeds the same refactoring prompt to an AI coding agent until the codebase meets all quality criteria. Progress persists in files and git history, not in the AI's context window.

## Structure

- **Go Backend** (root): Clean Architecture with SOLID principles, idiomatic Go patterns
- **React Frontend** (web-ui/): TypeScript + React with Logic/UI separation using custom hooks

## Quick Start

### 1. Run Validation

Test current state:

```bash
./scripts/validate-all.sh
```

### 2. Start Ralph Loop

```bash
./scripts/ralph-loop.sh
```

This will continuously refactor until the completion promise is found.

### 3. Manual Iteration

For a single refactoring pass:

```bash
# Read the prompt
cat scripts/ralph-prompt.md

# Apply changes
# ... make your changes ...

# Validate
./scripts/validate-all.sh

# Commit with proper format
git commit -m "refactor(go/user): extract repository interface

- Apply Dependency Inversion Principle
- Enable easier testing"
```

## Validation Scripts

| Script | Purpose |
|--------|---------|
| `scripts/validate-go.sh` | Go backend validation (format, vet, lint, tests) |
| `scripts/validate-react.sh` | React frontend validation (ESLint, TypeScript, tests, build) |
| `scripts/validate-all.sh` | Full repository validation |
| `scripts/ralph-loop.sh` | Infinite refactoring loop |
| `scripts/ralph-prompt.md` | AI refactoring instructions |

## Architecture Goals

### Go Backend

- Feature-based package structure
- Interface-driven design
- Constructor injection
- Repository pattern
- No circular dependencies
- 50%+ test coverage

### React Frontend

- Custom hooks for logic
- Presentational components
- Service layer for APIs
- No business logic in JSX
- Clean, readable components

## Completion Criteria

The loop completes when:

- All validations pass
- No TODO/FIXME/HACK comments
- All SOLID principles applied
- Clean Architecture enforced
- Code is idiomatic and maintainable

## Tools Required

### Go

- Go 1.21+ (current: 1.25.6)
- golangci-lint (recommended)
- staticcheck (alternative)

### React

- Node.js 18+
- npm
- ESLint (included)
- Prettier (included)

## Manual Installation

Install tools if missing:

```bash
# Go tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install honnef.co/go/tools/cmd/staticcheck@latest

# React tools (in web-ui/)
cd web-ui
npm install
```

## Git Commit Convention

```
refactor(component): brief description

- Why this change (principle violated)
- What was changed
- Impact/benefit
```

## Support

If validation fails, review the output for specific issues and refactor incrementally.
