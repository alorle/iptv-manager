.PHONY: help dev build test lint format clean generate install

# Default target
.DEFAULT_GOAL := help

help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

install: ## Install all dependencies (Go + npm)
	@echo "Installing Go dependencies..."
	@go mod download
	@echo "Installing npm dependencies..."
	@npm ci
	@echo "✓ All dependencies installed"

dev: ## Start development servers (requires 2 terminals: use 'make dev-backend' and 'make dev-frontend' separately)
	@echo "Use 'make dev-backend' in one terminal and 'make dev-frontend' in another"

dev-backend: ## Start backend with live reload (air)
	@air

dev-frontend: ## Start frontend dev server (Vite)
	@npm run dev

build: ## Build both frontend and backend
	@echo "Building frontend..."
	@npm run build
	@echo "Building backend..."
	@go build -o ./.tmp/main .
	@echo "✓ Build complete"

build-frontend: ## Build frontend only
	@npm run build

build-backend: ## Build backend only
	@go build -o ./.tmp/main .

run: ## Run the production binary
	@./.tmp/main

test: ## Run all tests (frontend + backend)
	@echo "Running frontend tests..."
	@npm run test
	@echo "Running Go tests..."
	@go test -v -race ./...
	@echo "✓ All tests passed"

test-frontend: ## Run frontend tests only
	@npm run test

test-frontend-watch: ## Run frontend tests in watch mode
	@npm run test:watch

test-frontend-ui: ## Run frontend tests with UI
	@npm run test:ui

test-frontend-coverage: ## Run frontend tests with coverage
	@npm run test:coverage

test-backend: ## Run backend tests only
	@go test -v -race ./...

test-coverage: ## Run all tests with coverage report
	@echo "Running frontend tests with coverage..."
	@npm run test:coverage
	@echo "Running backend tests with coverage..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage reports generated"

lint: ## Run all linters
	@echo "Linting frontend..."
	@npm run lint
	@echo "Linting backend..."
	@golangci-lint run --timeout=5m
	@echo "✓ All linters passed"

lint-fix: ## Run linters with auto-fix
	@echo "Fixing frontend..."
	@npm run lint -- --fix
	@echo "Fixing backend..."
	@golangci-lint run --fix --timeout=5m
	@echo "✓ Linting fixed"

format: ## Format all code
	@echo "Formatting frontend..."
	@npm run format
	@echo "Formatting backend..."
	@find . -type f -name "*.go" \
		-not -path "./.direnv/*" \
		-not -path "./node_modules/*" \
		-not -path "./dist/*" \
		-not -path "./.tmp/*" \
		-not -path "./tmp/*" \
		-not -path "./vendor/*" \
		-print0 | xargs -0 -r gofmt -s -w
	@if command -v goimports >/dev/null 2>&1; then \
		find . -type f -name "*.go" \
			-not -path "./.direnv/*" \
			-not -path "./node_modules/*" \
			-not -path "./dist/*" \
			-not -path "./.tmp/*" \
			-not -path "./tmp/*" \
			-not -path "./vendor/*" \
			-print0 | xargs -0 -r goimports -w; \
	else \
		echo "⚠ goimports not found. Run 'make setup' to install it."; \
	fi
	@echo "✓ All code formatted"

format-check: ## Check if code is formatted
	@echo "Checking frontend formatting..."
	@npm run format:check
	@echo "Checking backend formatting..."
	@UNFORMATTED=$$(find . -type f -name "*.go" \
		-not -path "./.direnv/*" \
		-not -path "./node_modules/*" \
		-not -path "./dist/*" \
		-not -path "./.tmp/*" \
		-not -path "./tmp/*" \
		-not -path "./vendor/*" \
		-print0 | xargs -0 gofmt -l); \
	if [ -n "$$UNFORMATTED" ]; then \
		echo "Backend code is not formatted. Run 'make format'"; \
		echo "$$UNFORMATTED"; \
		exit 1; \
	fi
	@echo "✓ All code is properly formatted"

typecheck: ## Type check TypeScript code
	@npx tsc -b

generate: ## Generate all code from OpenAPI spec
	@echo "Generating backend code..."
	@go generate ./internal/api/server.go
	@echo "Generating frontend types..."
	@npm run generate:api
	@echo "✓ Code generation complete"

generate-backend: ## Generate backend code only
	@go generate ./internal/api/server.go

generate-frontend: ## Generate frontend types only
	@npm run generate:api

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	@rm -rf dist/ .tmp/ tmp/ coverage.out coverage.html
	@echo "✓ Clean complete"

clean-all: clean ## Clean everything including dependencies
	@echo "Cleaning dependencies..."
	@rm -rf node_modules/
	@go clean -modcache
	@echo "✓ Everything cleaned"

docker-build: ## Build Docker image
	@docker build -t iptv-manager .

docker-run: ## Run Docker container
	@docker run -p 8080:8080 iptv-manager

ci: format-check lint typecheck test build ## Run all CI checks locally

setup: install ## Setup the project (install dependencies and tools)
	@echo "Installing golangci-lint..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Installing goimports..."
	@go install golang.org/x/tools/cmd/goimports@latest
	@echo "✓ Setup complete"
