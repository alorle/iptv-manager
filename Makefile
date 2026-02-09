BINARY := iptv-manager
GO_CMD := ./cmd/iptv-manager

.PHONY: build dev ui-deps ui-build test lint verify clean

## Production build: frontend + backend in one binary
build: ui-build
	go build -o $(BINARY) $(GO_CMD)

## Development: run Vite dev server + air (Go hot reload) in parallel
dev:
	@echo "Starting Vite dev server and air..."
	@npm run dev -w ui & air; wait

## Frontend
ui-deps:
	npm ci

ui-build: ui-deps
	npm run build -w ui

## Go checks
test:
	go test ./... -tags dev -race -count=1

lint:
	golangci-lint run ./...

verify: lint test build
	@echo "All checks passed."

## Cleanup
clean:
	rm -rf ui/dist ui/node_modules node_modules $(BINARY) tmp
