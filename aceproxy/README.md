# Ace Stream Engine Client

This package provides a Go client for communicating with the Ace Stream Engine HTTP API.

## Features

- HTTP client for Ace Stream Engine API
- Support for `/ace/getstream` endpoint with `id` and `pid` parameters
- Configurable engine URL via environment variable `ACESTREAM_ENGINE_URL`
- Configurable connection timeout (default 30s)
- Proper error handling for connection failures
- Context support for request cancellation
- Periodic health checks with configurable interval
- Health check integration with circuit breaker for failure detection
- Health status monitoring

## Usage

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/alorle/iptv-manager/aceproxy"
)

func main() {
    // Create client with default configuration
    client := aceproxy.NewClient(nil)
    defer client.Close()

    // Or create with custom configuration
    client = aceproxy.NewClient(&aceproxy.Config{
        EngineURL: "http://192.168.1.100:6878",
        Timeout:   30 * time.Second,
    })

    // Get stream URL for content ID
    resp, err := client.GetStream(context.Background(), aceproxy.GetStreamParams{
        ContentID: "94c2fd8fb9bc8f2fc71a2cbe9d4b866f227a0209",
        ProductID: "my-product-id",
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Stream URL: %s\n", resp.StreamURL)
}
```

### Health Checks

```go
package main

import (
    "log"
    "time"

    "github.com/alorle/iptv-manager/aceproxy"
    "github.com/alorle/iptv-manager/circuitbreaker"
)

func main() {
    // Create circuit breaker for health check integration
    cb := circuitbreaker.New(circuitbreaker.Config{
        FailureThreshold:   5,
        Timeout:            30 * time.Second,
        HalfOpenRequests:   1,
    })

    // Create client with health checks enabled
    client := aceproxy.NewClient(&aceproxy.Config{
        EngineURL:           "http://localhost:6878",
        Timeout:             30 * time.Second,
        HealthCheckInterval: 30 * time.Second, // Set to 0 to disable
        CircuitBreaker:      cb,
    })
    defer client.Close()

    // Health checks run automatically in the background
    // You can check the status at any time
    status := client.GetHealthStatus()
    if !status.Healthy {
        log.Printf("Engine is unhealthy: %v", status.Error)
    }
}
```

## Configuration

### Environment Variables

- `ACESTREAM_ENGINE_URL`: Base URL of the Ace Stream Engine (default: `http://localhost:6878`)
- `ACESTREAM_TIMEOUT`: Connection timeout in seconds (default: `30`)

### Example

```bash
export ACESTREAM_ENGINE_URL=http://192.168.1.100:6878
export ACESTREAM_TIMEOUT=60
```

## API Reference

### `NewClient(cfg *Config) *Client`

Creates a new Ace Stream Engine client. If `cfg` is `nil`, uses default configuration or environment variables.

Config fields:
- `EngineURL`: Base URL of the Ace Stream Engine
- `Timeout`: Connection timeout
- `HealthCheckInterval`: Interval between health checks (0 to disable)
- `CircuitBreaker`: Optional circuit breaker for health check failure tracking
- `Logger`: Optional logger for health check messages

### `GetStream(ctx context.Context, params GetStreamParams) (*GetStreamResponse, error)`

Requests a stream URL from the Ace Stream Engine for the given content ID.

Parameters:
- `ctx`: Context for request cancellation
- `params.ContentID`: Ace Stream content ID (required)
- `params.ProductID`: Product ID (optional)

Returns:
- `GetStreamResponse` with the stream URL and status code
- Error if the request fails

### `HealthCheck(ctx context.Context) error`

Performs a single health check against the Ace Stream Engine. Returns an error if the engine is not healthy.

### `GetHealthStatus() HealthStatus`

Returns the last recorded health check status, including:
- `Healthy`: Whether the last health check succeeded
- `Timestamp`: When the last health check was performed
- `Error`: Error from the last health check (if any)

### `Close() error`

Closes the client and releases resources. This stops any running health check goroutines.
