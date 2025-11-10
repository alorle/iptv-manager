# Tooling Setup Guide

This project now has comprehensive tooling configured for code quality, consistency, and developer experience.

## Quick Start

### 1. Install Required Tools

```bash
# Install Go tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install golang.org/x/tools/cmd/goimports@latest

# Install npm dependencies (includes Lefthook)
npm install

# Setup git hooks
npx lefthook install
```

Or use the convenience command:

```bash
make setup
```

### 2. Install VSCode Extensions

Open VSCode and install the recommended extensions. You should see a notification, or run:

```
Extensions: Show Recommended Extensions
```

Recommended extensions:

- Go (golang.go)
- Prettier (esbenp.prettier-vscode)
- ESLint (dbaeumer.vscode-eslint)
- Tailwind CSS IntelliSense (bradlc.vscode-tailwindcss)
- YAML (redhat.vscode-yaml)
- REST Client (humao.rest-client)
- EditorConfig (editorconfig.editorconfig)

## What's New

### Configuration Files Created

**Quality & Formatting:**

- `.prettierrc` - Prettier configuration (consistent code formatting)
- `.prettierignore` - Files to exclude from Prettier
- `.golangci.yml` - Go linting configuration (15+ linters enabled)
- `.editorconfig` - Cross-editor consistency (tabs for Go, spaces for TS/JS)

**VSCode:**

- `.vscode/extensions.json` - Recommended extensions
- `.vscode/launch.json` - Debug configurations
- `.vscode/tasks.json` - Common development tasks
- `.vscode/settings.json` - Enhanced with Go and TypeScript optimizations

**Git Hooks:**

- `.lefthook.yml` - Pre-commit and pre-push hooks

**CI/CD:**

- `.github/workflows/ci.yaml` - Automated quality checks on PRs

**Convenience:**

- `Makefile` - Common commands (run `make help`)

### New npm Scripts

```bash
npm run lint          # Lint frontend code
npm run lint:fix      # Lint and auto-fix frontend code
npm run format        # Format all code
npm run format:check  # Check if code is formatted
npm run typecheck     # Type check TypeScript
npm run ci            # Run all CI checks locally
```

### New Make Commands

```bash
make help             # Show all available commands
make install          # Install all dependencies
make lint             # Run all linters
make lint-fix         # Run linters with auto-fix
make format           # Format all code
make format-check     # Check if code is formatted
make test             # Run all tests
make ci               # Run all CI checks locally
```

## How It Works

### Automatic Quality Checks

**On Save (VSCode):**

- Frontend files automatically formatted with Prettier
- Go files automatically formatted with goimports
- ESLint issues auto-fixed
- Imports organized automatically

**On Commit (Git Hooks):**

- Staged frontend files are linted and formatted
- Staged Go files are formatted and linted
- Fixed files are automatically staged
- Commit fails if there are unfixable issues

**On Push (Git Hooks):**

- TypeScript type checking
- Frontend build verification
- Backend tests
- Backend build verification

**On Pull Request (CI/CD):**

- All linting checks (frontend + backend)
- Format checks
- Type checking
- Build verification
- Test execution
- OpenAPI spec validation

### Debugging

Use VSCode's Run & Debug panel (Cmd+Shift+D) to:

1. **Launch Backend (Dev Mode)** - Debug Go backend with Vite proxy
2. **Attach to Frontend (Chrome)** - Debug React app in Chrome
3. **Full Stack Debug** - Debug both simultaneously

### Tasks

Use VSCode's Tasks (Cmd+Shift+P → "Tasks: Run Task") to:

- Generate API code (backend or frontend)
- Build (frontend, backend, or both)
- Format all code
- Lint (frontend or backend)
- Type check
- Run tests

## Troubleshooting

### "golangci-lint: command not found"

Install it:

```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

Make sure `$GOPATH/bin` is in your PATH.

### "goimports: command not found"

Install it:

```bash
go install golang.org/x/tools/cmd/goimports@latest
```

### Git hooks not running

Install Lefthook:

```bash
npx lefthook install
```

### VSCode not formatting on save

1. Check that you've installed the recommended extensions
2. Reload VSCode (Cmd+Shift+P → "Developer: Reload Window")
3. Check the status bar shows "Prettier" as the formatter

### ESLint errors in VSCode

The project now uses stricter linting rules. Common issues:

1. **Type checking errors**: Make sure TypeScript can find types:

   ```bash
   npm run typecheck
   ```

2. **Import order**: ESLint now enforces import order. Auto-fix will handle this:

   ```bash
   npm run lint:fix
   ```

3. **Accessibility**: jsx-a11y plugin catches accessibility issues. Examples:
   - Images need alt text
   - Buttons need accessible names
   - Interactive elements need proper roles

### CI/CD failing

Run checks locally before pushing:

```bash
make ci
```

This runs the same checks as CI/CD.

## Best Practices

1. **Let the tools help you**: VSCode is configured to auto-fix on save
2. **Commit often**: Pre-commit hooks catch issues early
3. **Run `make ci` before pushing**: Catches issues before CI/CD
4. **Use the Makefile**: Simpler than remembering all commands
5. **Don't edit generated files**: api.gen.go and v1.d.ts are regenerated from OpenAPI spec

## Configuration Philosophy

- **Prettier**: Handles all formatting (no bike-shedding)
- **ESLint**: Catches bugs and enforces best practices
- **golangci-lint**: Comprehensive Go code quality
- **TypeScript**: Strict mode for type safety
- **EditorConfig**: Ensures tabs for Go, spaces for everything else
- **Lefthook**: Fast, reliable git hooks

All tools are configured to work together without conflicts.
