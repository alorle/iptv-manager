# Ralph Wiggum Setup Checklist

## Files Created

- [x] `scripts/validate-go.sh` - Go validation script
- [x] `scripts/validate-react.sh` - React validation script
- [x] `scripts/validate-all.sh` - Combined validation
- [x] `scripts/ralph-loop.sh` - Main Ralph loop
- [x] `scripts/ralph-prompt.md` - Refactoring instructions
- [x] `.golangci.yml` - Go linter configuration
- [x] `RALPH_SETUP.md` - Documentation
- [x] `SETUP_CHECKLIST.md` - This file

## Scripts Executable

- [ ] `chmod +x scripts/validate-go.sh`
- [ ] `chmod +x scripts/validate-react.sh`
- [ ] `chmod +x scripts/validate-all.sh`
- [ ] `chmod +x scripts/ralph-loop.sh`

## Tools Installed

### Go

- [ ] `go version` works
- [ ] `golangci-lint version` works (or staticcheck)
- [ ] `go test ./...` runs

### React

- [ ] `node --version` works (18+)
- [ ] `npm --version` works
- [ ] `cd web-ui && npm install` succeeds
- [ ] ESLint installed in web-ui
- [ ] Prettier installed in web-ui

## Validation Tests

- [ ] `./scripts/validate-go.sh` runs (may fail initially)
- [ ] `./scripts/validate-react.sh` runs (may fail initially)
- [ ] `./scripts/validate-all.sh` runs (may fail initially)

## Git Setup

- [x] Repository has git initialized
- [ ] `.gitignore` includes `coverage.out`, `node_modules/`, etc.
- [ ] Can commit changes

## Ready to Start

Once all checkboxes are ticked, you're ready to:

```bash
./scripts/ralph-loop.sh
```

Or run manual iterations with:

```bash
./scripts/validate-all.sh
# Review issues
# Make changes
# Commit
# Repeat
```
