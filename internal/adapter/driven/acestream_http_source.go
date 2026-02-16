package driven

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/alorle/iptv-manager/internal/port/driven"
)

const (
	// Source identifiers
	SourceNewEra = "new-era"
	SourceElcano = "elcano"

	// HTTP client timeout for fetching hash lists
	defaultFetchTimeout = 30 * time.Second
)

// Source URLs for Acestream hash lists
var sourceURLs = map[string]string{
	SourceNewEra: "https://ipfs.io/ipns/k2k4r8oqlcjxsritt5mczkcn4mmvcmymbqw7113fz2flkrerfwfps004/data/listas/lista_fuera_iptv.m3u",
	SourceElcano: "https://ipfs.io/ipns/k51qzi5uqu5di462t7j4vu4akwfhvtjhy88qbupktvoacqfqe9uforjvhyi4wr/hashes.json",
}

// AcestreamHTTPSource implements the AcestreamSource port by fetching hash lists
// from HTTP endpoints (NEW ERA and Elcano.top).
type AcestreamHTTPSource struct {
	httpClient *http.Client
}

// NewAcestreamHTTPSource creates a new HTTP-based Acestream source adapter.
func NewAcestreamHTTPSource() *AcestreamHTTPSource {
	return &AcestreamHTTPSource{
		httpClient: &http.Client{
			Timeout: defaultFetchTimeout,
		},
	}
}

// FetchHashes retrieves Acestream hashes from the specified source.
// Supported sources: "new-era", "elcano".
func (s *AcestreamHTTPSource) FetchHashes(ctx context.Context, source string) (map[string][]string, error) {
	url, ok := sourceURLs[source]
	if !ok {
		return nil, fmt.Errorf("unknown source: %s", source)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for %s: %w", source, err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %s: %w", source, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d from %s", resp.StatusCode, source)
	}

	switch source {
	case SourceNewEra:
		return s.parseNewEra(resp.Body)
	case SourceElcano:
		return s.parseElcano(resp.Body)
	default:
		return nil, fmt.Errorf("no parser for source: %s", source)
	}
}

// parseNewEra parses the NEW ERA M3U playlist format.
// Format: #EXTINF lines with tvg-id attribute, followed by acestream:// URLs.
// Groups hashes by tvg-id (which matches EPG channel IDs).
func (s *AcestreamHTTPSource) parseNewEra(r io.Reader) (map[string][]string, error) {
	result := make(map[string][]string)
	scanner := bufio.NewScanner(r)
	// Increase scanner buffer for long #EXTINF lines with logos
	scanner.Buffer(make([]byte, 0, 64*1024), 256*1024)

	var currentTVGID string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "#EXTINF:") {
			currentTVGID = extractTVGID(line)
			continue
		}

		// acestream:// URL line following an #EXTINF
		if currentTVGID != "" && strings.HasPrefix(line, "acestream://") {
			hash := strings.TrimPrefix(line, "acestream://")
			if hash != "" {
				result[currentTVGID] = append(result[currentTVGID], hash)
			}
			currentTVGID = ""
			continue
		}

		// Any other line resets state (e.g. #EXTGRP, #EXTM3U, blank lines)
		if !strings.HasPrefix(line, "#") {
			currentTVGID = ""
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to parse NEW ERA M3U: %w", err)
	}

	return result, nil
}

// extractTVGID extracts the tvg-id attribute value from an #EXTINF line.
// Returns empty string if tvg-id is not found or empty.
func extractTVGID(line string) string {
	const marker = `tvg-id="`
	idx := strings.Index(line, marker)
	if idx < 0 {
		return ""
	}
	start := idx + len(marker)
	end := strings.Index(line[start:], `"`)
	if end < 0 {
		return ""
	}
	return strings.TrimSpace(line[start : start+end])
}

// elcanoResponse represents the root JSON object from the Elcano source.
type elcanoResponse struct {
	Hashes []elcanoEntry `json:"hashes"`
}

// elcanoEntry represents a single hash entry in the Elcano JSON format.
type elcanoEntry struct {
	Title string `json:"title"`
	Hash  string `json:"hash"`
	TVGID string `json:"tvg_id"`
}

// parseElcano parses the Elcano JSON format.
// Format: {"generated": "...", "count": N, "hashes": [{"title": "...", "hash": "...", "tvg_id": "...", ...}]}
// Groups hashes by tvg_id (which matches EPG channel IDs) for direct matching.
func (s *AcestreamHTTPSource) parseElcano(r io.Reader) (map[string][]string, error) {
	var resp elcanoResponse

	if err := json.NewDecoder(r).Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to parse Elcano JSON: %w", err)
	}

	result := make(map[string][]string)
	for _, entry := range resp.Hashes {
		if entry.Hash == "" {
			continue
		}
		// Use tvg_id as the channel key since it directly matches EPG channel IDs.
		// Fall back to title if tvg_id is empty.
		key := entry.TVGID
		if key == "" {
			key = entry.Title
		}
		if key != "" {
			result[key] = append(result[key], entry.Hash)
		}
	}

	return result, nil
}

// Ensure AcestreamHTTPSource implements the driven.AcestreamSource interface
var _ driven.AcestreamSource = (*AcestreamHTTPSource)(nil)
