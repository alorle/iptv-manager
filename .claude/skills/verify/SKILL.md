---
name: verify
description: Run the full Go validation checklist (format, vet, lint, test, build)
disable-model-invocation: true
allowed-tools: Bash(gofmt *), Bash(goimports *), Bash(go vet *), Bash(go test *), Bash(go build *), Bash(golangci-lint *), Bash(go install *)
---

Run the full verification checklist. Fix any issues found before reporting results.

Ensure direnv environment is loaded before running commands.

## Steps

1. **Format** — run `gofmt -l .` and `goimports -l .`. Both must produce no output. If they do, run `gofmt -w .` and `goimports -w .` to fix, then re-check.

2. **Vet** — run `go vet ./...`. Must pass with zero issues.

3. **Lint** — run `golangci-lint run ./...`. Must pass with zero issues. If `golangci-lint` is not installed, run `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest` first.

4. **Test** — run `go test ./... -race -count=1`. All tests must pass.

5. **Build** — run `go build ./...`. Must compile cleanly.

If `goimports` is not installed, run `go install golang.org/x/tools/cmd/goimports@latest` first.

Report which steps passed and which failed. If any step fails, fix the issue and re-run that step.
