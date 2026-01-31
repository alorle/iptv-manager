# Ace Stream Engine Client

This package provides a Go client for communicating with the Ace Stream Engine HTTP API.

## Features

- HTTP client for Ace Stream Engine API
- Support for `/ace/getstream` endpoint with `id` and `pid` parameters
- Configurable engine URL via environment variable `ACESTREAM_ENGINE_URL`
- Configurable connection timeout (default 30s)
- Proper error handling for connection failures
- Context support for request cancellation

## Usage

```go
package main

import (
    "context"
    "fmt"
    "log"

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

### `GetStream(ctx context.Context, params GetStreamParams) (*GetStreamResponse, error)`

Requests a stream URL from the Ace Stream Engine for the given content ID.

Parameters:
- `ctx`: Context for request cancellation
- `params.ContentID`: Ace Stream content ID (required)
- `params.ProductID`: Product ID (optional)

Returns:
- `GetStreamResponse` with the stream URL and status code
- Error if the request fails

### `Close() error`

Closes the client and releases resources.
