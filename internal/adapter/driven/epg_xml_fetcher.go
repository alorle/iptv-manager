package driven

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/alorle/iptv-manager/internal/epg"
)

const (
	defaultTimeout = 30 * time.Second
	defaultURL     = "https://raw.githubusercontent.com/davidmuma/EPG_dobleM/master/guiatv.xml"
)

// EPGXMLFetcher fetches EPG data from an XML source via HTTP.
// It implements the driven.EPGFetcher port.
type EPGXMLFetcher struct {
	url    string
	client *http.Client
}

// NewEPGXMLFetcher creates a new EPG XML fetcher with the given URL.
// If url is empty, it uses the default EPG source.
// If client is nil, it creates a default HTTP client with a 30-second timeout.
func NewEPGXMLFetcher(url string, client *http.Client) *EPGXMLFetcher {
	if url == "" {
		url = defaultURL
	}
	if client == nil {
		client = &http.Client{
			Timeout: defaultTimeout,
		}
	}
	return &EPGXMLFetcher{
		url:    url,
		client: client,
	}
}

// FetchEPG retrieves EPG channel data from the configured XML source.
// It fetches the XML file via HTTP, parses it, and returns domain EPG channels.
// Returns an error if the HTTP request fails, the XML is malformed, or domain validation fails.
func (f *EPGXMLFetcher) FetchEPG(ctx context.Context) ([]epg.Channel, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating HTTP request: %w", err)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching EPG XML: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP status: %d %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	var tv tvXML
	if err := xml.Unmarshal(body, &tv); err != nil {
		return nil, fmt.Errorf("parsing EPG XML: %w", err)
	}

	channels := make([]epg.Channel, 0, len(tv.Channels))
	for _, ch := range tv.Channels {
		// Use the first display-name as the channel name, or fall back to the ID
		name := ch.ID
		if len(ch.DisplayNames) > 0 {
			name = ch.DisplayNames[0]
		}

		// Extract logo URL from icon element
		logo := ""
		if len(ch.Icons) > 0 {
			logo = ch.Icons[0].Src
		}

		// Create domain channel entity
		// Note: This XML format doesn't include category or language, so we leave them empty
		domainChannel, err := epg.NewChannel(ch.ID, name, logo, "", "", ch.ID)
		if err != nil {
			return nil, fmt.Errorf("creating domain channel %q: %w", ch.ID, err)
		}

		channels = append(channels, domainChannel)
	}

	return channels, nil
}

// tvXML represents the root element of the EPG XML file.
type tvXML struct {
	XMLName  xml.Name     `xml:"tv"`
	Channels []channelXML `xml:"channel"`
}

// channelXML represents a channel element in the EPG XML.
type channelXML struct {
	ID           string    `xml:"id,attr"`
	DisplayNames []string  `xml:"display-name"`
	Icons        []iconXML `xml:"icon"`
}

// iconXML represents an icon element with a src attribute.
type iconXML struct {
	Src string `xml:"src,attr"`
}
