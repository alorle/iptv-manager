package driven

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/alorle/iptv-manager/internal/port/driven"
	"github.com/alorle/iptv-manager/internal/streaming"
)

// AceStreamHTTPAdapter implements the AceStreamEngine port using HTTP calls
// to the AceStream Engine API.
type AceStreamHTTPAdapter struct {
	baseURL    string
	httpClient *http.Client
	logger     *slog.Logger
}

// NewAceStreamHTTPAdapter creates a new HTTP adapter for AceStream Engine.
// baseURL should point to the AceStream Engine HTTP API (e.g., http://localhost:6878).
func NewAceStreamHTTPAdapter(baseURL string, logger *slog.Logger) *AceStreamHTTPAdapter {
	return &AceStreamHTTPAdapter{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// StartStream initiates a stream for the given infohash with a unique PID.
func (a *AceStreamHTTPAdapter) StartStream(ctx context.Context, infoHash, pid string) (string, error) {
	// Build the request URL with parameters
	params := url.Values{}
	params.Set("id", infoHash)
	params.Set("pid", pid)
	params.Set("format", "json")

	reqURL := fmt.Sprintf("%s/ace/getstream?%s", a.baseURL, params.Encode())

	a.logger.Debug("engine request", "method", http.MethodGet, "url", reqURL, "pid", pid)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create start stream request: %w", err)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		a.logger.Warn("engine network error", "error", err, "url", reqURL, "timeout", a.httpClient.Timeout)
		return "", fmt.Errorf("failed to start stream: %w", err)
	}
	defer resp.Body.Close()

	a.logger.Debug("engine response", "status_code", resp.StatusCode, "content_type", resp.Header.Get("Content-Type"), "content_length", resp.Header.Get("Content-Length"))

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)
		if len(bodyStr) > 500 {
			bodyStr = bodyStr[:500]
		}
		a.logger.Error("engine http error", "status_code", resp.StatusCode, "body", bodyStr, "url", reqURL)
		return "", fmt.Errorf("engine returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response to extract stream URL
	var result struct {
		Response struct {
			StreamURL string `json:"playback_url"`
		} `json:"response"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode start stream response: %w", err)
	}

	if result.Response.StreamURL == "" {
		return "", fmt.Errorf("engine did not return a stream URL")
	}

	return result.Response.StreamURL, nil
}

// GetStats retrieves statistics for an active stream identified by its PID.
func (a *AceStreamHTTPAdapter) GetStats(ctx context.Context, pid string) (driven.StreamStats, error) {
	params := url.Values{}
	params.Set("pid", pid)
	params.Set("format", "json")

	reqURL := fmt.Sprintf("%s/ace/stat?%s", a.baseURL, params.Encode())

	a.logger.Debug("engine request", "method", http.MethodGet, "url", reqURL, "pid", pid)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return driven.StreamStats{}, fmt.Errorf("failed to create stats request: %w", err)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		a.logger.Warn("engine network error", "error", err, "url", reqURL, "timeout", a.httpClient.Timeout)
		return driven.StreamStats{}, fmt.Errorf("failed to get stats: %w", err)
	}
	defer resp.Body.Close()

	a.logger.Debug("engine response", "status_code", resp.StatusCode, "content_type", resp.Header.Get("Content-Type"), "content_length", resp.Header.Get("Content-Length"))

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)
		if len(bodyStr) > 500 {
			bodyStr = bodyStr[:500]
		}
		a.logger.Error("engine http error", "status_code", resp.StatusCode, "body", bodyStr, "url", reqURL)
		return driven.StreamStats{}, fmt.Errorf("engine returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse engine response
	var result struct {
		Response struct {
			Status     string `json:"status"`
			Peers      int    `json:"peers"`
			SpeedDown  int64  `json:"speed_down"`
			SpeedUp    int64  `json:"speed_up"`
			Downloaded int64  `json:"downloaded"`
			Uploaded   int64  `json:"uploaded"`
		} `json:"response"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return driven.StreamStats{}, fmt.Errorf("failed to decode stats response: %w", err)
	}

	return driven.StreamStats{
		PID:        pid,
		Status:     result.Response.Status,
		Peers:      result.Response.Peers,
		SpeedDown:  result.Response.SpeedDown,
		SpeedUp:    result.Response.SpeedUp,
		Downloaded: result.Response.Downloaded,
		Uploaded:   result.Response.Uploaded,
	}, nil
}

// StopStream terminates the stream identified by its PID.
func (a *AceStreamHTTPAdapter) StopStream(ctx context.Context, pid string) error {
	params := url.Values{}
	params.Set("pid", pid)

	reqURL := fmt.Sprintf("%s/ace/stop?%s", a.baseURL, params.Encode())

	a.logger.Debug("engine request", "method", http.MethodGet, "url", reqURL, "pid", pid)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create stop stream request: %w", err)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		a.logger.Warn("engine network error", "error", err, "url", reqURL, "timeout", a.httpClient.Timeout)
		return fmt.Errorf("failed to stop stream: %w", err)
	}
	defer resp.Body.Close()

	a.logger.Debug("engine response", "status_code", resp.StatusCode, "content_type", resp.Header.Get("Content-Type"), "content_length", resp.Header.Get("Content-Length"))

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)
		if len(bodyStr) > 500 {
			bodyStr = bodyStr[:500]
		}
		a.logger.Error("engine http error", "status_code", resp.StatusCode, "body", bodyStr, "url", reqURL)
		return fmt.Errorf("engine returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// StreamContent establishes a streaming connection and copies the stream data
// to the provided writer.
func (a *AceStreamHTTPAdapter) StreamContent(ctx context.Context, streamURL string, dst io.Writer, infoHash, pid string, writeTimeout time.Duration) error {
	a.logger.Debug("starting content stream", "stream_url", streamURL, "infohash", infoHash, "pid", pid, "write_timeout", writeTimeout)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, streamURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create stream content request: %w", err)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		a.logger.Warn("engine network error", "error", err, "url", streamURL, "timeout", a.httpClient.Timeout)
		return fmt.Errorf("failed to connect to stream: %w", err)
	}
	defer resp.Body.Close()

	a.logger.Debug("engine response", "status_code", resp.StatusCode, "content_type", resp.Header.Get("Content-Type"), "content_length", resp.Header.Get("Content-Length"))

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyStr := string(bodyBytes)
		if len(bodyStr) > 500 {
			bodyStr = bodyStr[:500]
		}
		a.logger.Error("engine http error", "status_code", resp.StatusCode, "body", bodyStr, "url", streamURL)
		return fmt.Errorf("stream returned status %d", resp.StatusCode)
	}

	// Set appropriate headers from the engine response
	if contentType := resp.Header.Get("Content-Type"); contentType != "" {
		if w, ok := dst.(http.ResponseWriter); ok {
			w.Header().Set("Content-Type", contentType)
		}
	}

	if contentLength := resp.Header.Get("Content-Length"); contentLength != "" {
		if w, ok := dst.(http.ResponseWriter); ok {
			w.Header().Set("Content-Length", contentLength)
		}
	}

	// Wrap the writer with timeout support
	timeoutWriter := streaming.NewTimeoutWriter(dst, writeTimeout, a.logger, infoHash, pid)

	// Stream the content efficiently with timeout protection
	bytesWritten, err := io.Copy(timeoutWriter, resp.Body)

	// Log completion with reason
	if err == nil {
		a.logger.Info("content stream completed", "stream_url", streamURL, "infohash", infoHash, "pid", pid, "bytes_written", bytesWritten, "reason", "EOF")
	} else if err == context.Canceled {
		a.logger.Info("content stream completed", "stream_url", streamURL, "infohash", infoHash, "pid", pid, "bytes_written", bytesWritten, "reason", "canceled")
	} else if err == streaming.ErrWriteTimeout {
		// Slow client detected - this is logged by TimeoutWriter
		a.logger.Info("content stream completed", "stream_url", streamURL, "infohash", infoHash, "pid", pid, "bytes_written", bytesWritten, "reason", "slow_client")
		return err
	} else {
		a.logger.Info("content stream completed", "stream_url", streamURL, "infohash", infoHash, "pid", pid, "bytes_written", bytesWritten, "reason", "error", "error", err)
		return fmt.Errorf("failed to stream content: %w", err)
	}

	return nil
}

// EngineStats contains overall engine statistics.
type EngineStats struct {
	ActiveStreams int
	TotalPeers    int
	UploadSpeed   int64
	DownloadSpeed int64
}

// GetEngineStats retrieves overall engine statistics.
// This is useful for monitoring and debugging.
func (a *AceStreamHTTPAdapter) GetEngineStats(ctx context.Context) (EngineStats, error) {
	reqURL := fmt.Sprintf("%s/ace/manifest.json", a.baseURL)

	a.logger.Debug("engine request", "method", http.MethodGet, "url", reqURL, "pid", "")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return EngineStats{}, fmt.Errorf("failed to create engine stats request: %w", err)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		a.logger.Warn("engine network error", "error", err, "url", reqURL, "timeout", a.httpClient.Timeout)
		return EngineStats{}, fmt.Errorf("failed to get engine stats: %w", err)
	}
	defer resp.Body.Close()

	a.logger.Debug("engine response", "status_code", resp.StatusCode, "content_type", resp.Header.Get("Content-Type"), "content_length", resp.Header.Get("Content-Length"))

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)
		if len(bodyStr) > 500 {
			bodyStr = bodyStr[:500]
		}
		a.logger.Error("engine http error", "status_code", resp.StatusCode, "body", bodyStr, "url", reqURL)
		return EngineStats{}, fmt.Errorf("engine returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Version string `json:"version"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return EngineStats{}, fmt.Errorf("failed to decode engine stats: %w", err)
	}

	// For now, return basic stats. More detailed stats would require
	// querying individual streams.
	return EngineStats{}, nil
}

// SetHTTPClient allows replacing the default HTTP client.
// Useful for testing with custom transports or timeouts.
func (a *AceStreamHTTPAdapter) SetHTTPClient(client *http.Client) {
	a.httpClient = client
}

// Ping checks if the AceStream engine is accessible and operational.
func (a *AceStreamHTTPAdapter) Ping(ctx context.Context) error {
	// Try to access the manifest endpoint as a health check
	reqURL := fmt.Sprintf("%s/ace/manifest.json", a.baseURL)

	a.logger.Debug("engine request", "method", http.MethodGet, "url", reqURL, "pid", "")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create ping request: %w", err)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		a.logger.Warn("engine network error", "error", err, "url", reqURL, "timeout", a.httpClient.Timeout)
		return fmt.Errorf("acestream engine not reachable: %w", err)
	}
	defer resp.Body.Close()

	a.logger.Debug("engine response", "status_code", resp.StatusCode, "content_type", resp.Header.Get("Content-Type"), "content_length", resp.Header.Get("Content-Length"))

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyStr := string(bodyBytes)
		if len(bodyStr) > 500 {
			bodyStr = bodyStr[:500]
		}
		a.logger.Error("engine http error", "status_code", resp.StatusCode, "body", bodyStr, "url", reqURL)
		return fmt.Errorf("acestream engine returned status %d", resp.StatusCode)
	}

	return nil
}
