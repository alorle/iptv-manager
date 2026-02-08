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

	a.logger.Debug("requesting stream from engine", "infohash", infoHash, "pid", pid, "url", reqURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create start stream request: %w", err)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		a.logger.Error("failed to connect to acestream engine", "infohash", infoHash, "pid", pid, "error", err)
		return "", fmt.Errorf("failed to start stream: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		a.logger.Error("acestream engine returned error", "infohash", infoHash, "pid", pid, "status", resp.StatusCode, "body", string(body))
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

	a.logger.Info("acestream engine returned stream url", "infohash", infoHash, "pid", pid, "stream_url", result.Response.StreamURL)

	return result.Response.StreamURL, nil
}

// GetStats retrieves statistics for an active stream identified by its PID.
func (a *AceStreamHTTPAdapter) GetStats(ctx context.Context, pid string) (driven.StreamStats, error) {
	params := url.Values{}
	params.Set("pid", pid)
	params.Set("format", "json")

	reqURL := fmt.Sprintf("%s/ace/stat?%s", a.baseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return driven.StreamStats{}, fmt.Errorf("failed to create stats request: %w", err)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return driven.StreamStats{}, fmt.Errorf("failed to get stats: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
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

	a.logger.Debug("stopping stream on engine", "pid", pid, "url", reqURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create stop stream request: %w", err)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		a.logger.Error("failed to stop stream on engine", "pid", pid, "error", err)
		return fmt.Errorf("failed to stop stream: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		a.logger.Error("engine returned error on stop", "pid", pid, "status", resp.StatusCode, "body", string(body))
		return fmt.Errorf("engine returned status %d: %s", resp.StatusCode, string(body))
	}

	a.logger.Info("stream stopped on engine", "pid", pid)

	return nil
}

// StreamContent establishes a streaming connection and copies the stream data
// to the provided writer.
func (a *AceStreamHTTPAdapter) StreamContent(ctx context.Context, streamURL string, dst io.Writer) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, streamURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create stream content request: %w", err)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to stream: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
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

	// Stream the content efficiently
	_, err = io.Copy(dst, resp.Body)
	if err != nil && err != context.Canceled {
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

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return EngineStats{}, fmt.Errorf("failed to create engine stats request: %w", err)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return EngineStats{}, fmt.Errorf("failed to get engine stats: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
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

	a.logger.Debug("pinging acestream engine", "url", reqURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create ping request: %w", err)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		a.logger.Error("acestream engine not reachable", "url", reqURL, "error", err)
		return fmt.Errorf("acestream engine not reachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		a.logger.Error("acestream engine returned error on ping", "url", reqURL, "status", resp.StatusCode)
		return fmt.Errorf("acestream engine returned status %d", resp.StatusCode)
	}

	a.logger.Debug("acestream engine is healthy", "url", reqURL)

	return nil
}
