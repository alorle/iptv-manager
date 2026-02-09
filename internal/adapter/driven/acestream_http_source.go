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
	SourceNewEra: "https://ipfs.io/ipns/k2k4r8oqlcjxsritt5mczkcn4mmvcmymbqw7113fz2flkrerfwfps004/data/listas/listaplana.txt",
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

// parseNewEra parses the NEW ERA plaintext format.
// Format: each line is "channelName acestream://hash"
func (s *AcestreamHTTPSource) parseNewEra(r io.Reader) (map[string][]string, error) {
	result := make(map[string][]string)
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Split by whitespace to get channel name and hash
		parts := strings.Fields(line)
		if len(parts) < 2 {
			// Skip malformed lines
			continue
		}

		channelName := parts[0]
		hashPart := parts[1]

		// Extract hash from "acestream://hash" or use as-is
		hash := strings.TrimPrefix(hashPart, "acestream://")
		if hash == "" {
			continue
		}

		result[channelName] = append(result[channelName], hash)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to parse NEW ERA format: %w", err)
	}

	return result, nil
}

// ElcanoChannel represents a channel entry in Elcano.top JSON format.
type ElcanoChannel struct {
	Name   string   `json:"name"`
	Hashes []string `json:"hashes"`
}

// parseElcano parses the Elcano.top JSON format.
// Format: JSON array of objects with "name" and "hashes" fields
func (s *AcestreamHTTPSource) parseElcano(r io.Reader) (map[string][]string, error) {
	var channels []ElcanoChannel

	if err := json.NewDecoder(r).Decode(&channels); err != nil {
		return nil, fmt.Errorf("failed to parse Elcano JSON: %w", err)
	}

	result := make(map[string][]string)
	for _, ch := range channels {
		if ch.Name != "" && len(ch.Hashes) > 0 {
			result[ch.Name] = ch.Hashes
		}
	}

	return result, nil
}

// Ensure AcestreamHTTPSource implements the driven.AcestreamSource interface
var _ driven.AcestreamSource = (*AcestreamHTTPSource)(nil)
