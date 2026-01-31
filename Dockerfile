# Build stage for frontend
FROM node:20-alpine AS frontend-builder

WORKDIR /app/web-ui

# Copy package files
COPY web-ui/package*.json ./

# Install dependencies
RUN npm ci

# Copy frontend source
COPY web-ui/ ./

# Build frontend
RUN npm run build

# Build stage for Go binary
FROM golang:1.25-alpine AS go-builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Copy built frontend from previous stage
COPY --from=frontend-builder /app/web-ui/dist ./ui/dist

# Build Go binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o iptv-manager .

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=go-builder /app/iptv-manager .

# Expose port
EXPOSE 8080

# Set environment variables with defaults
ENV HTTP_ADDRESS=0.0.0.0 \
    HTTP_PORT=8080 \
    CACHE_DIR=/cache \
    CACHE_TTL=1h

# Create cache directory
RUN mkdir -p /cache

# Run the binary
CMD ["./iptv-manager"]
