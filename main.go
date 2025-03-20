package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/alorle/iptv-manager/pkg/m3u"
	"github.com/google/uuid"
)

var httpAddress = os.Getenv("HTTP_ADDRESS")
var httpPort = os.Getenv("HTTP_PORT")
var streamsFile = os.Getenv("STREAMS_FILE")
var acestreamUrl *url.URL

type AcestreamStream struct {
	Title          string `json:"title"`
	GroupTitle     string `json:"group_title"`
	ID             string `json:"acestream_id"`
	NetworkCaching uint64 `json:"network_caching"`
}

type StreamsConfig struct {
	Streams []AcestreamStream `json:"streams"`
}

var acestreamStreams []AcestreamStream

func loadStreams() error {
	data, err := os.ReadFile(streamsFile)
	if err != nil {
		return fmt.Errorf("error reading %s: %v", streamsFile, err)
	}

	var config StreamsConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("error parsing %s: %v", streamsFile, err)
	}

	acestreamStreams = config.Streams
	return nil
}

func (s *AcestreamStream) GetURL(u *url.URL) string {
	q := u.Query()
	q.Set("id", s.ID)
	q.Set("pid", uuid.New().String())
	q.Set("network-caching", fmt.Sprintf("%d", s.NetworkCaching))
	u.RawQuery = q.Encode()

	return u.String()
}

func handleM3u8(w http.ResponseWriter, req *http.Request) {
	playlist := m3u.NewPlaylist()

	for _, stream := range acestreamStreams {
		playlist.AppendItem(&m3u.PlaylistItem{
			SeqId:    1,
			Title:    stream.Title,
			URI:      stream.GetURL(acestreamUrl),
			Duration: -1,
			TVGTags: &m3u.TVGTags{
				ID:         fmt.Sprintf("ace_%s", stream.ID),
				Name:       stream.Title,
				GroupTitle: stream.GroupTitle,
			},
		})
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/x-mpegURL")
	playlist.Write(w)
}

func main() {
	var err error

	flag.Parse()

	acestreamBaseUrl := os.Getenv("ACESTREAM_URL")
	if acestreamBaseUrl == "" {
		acestreamBaseUrl = "http://127.0.0.1:6878/ace/getstream"
	}

	acestreamUrl, err = url.Parse(acestreamBaseUrl)
	if err != nil {
		fmt.Printf("Error parsing ACESTREAM_URL: %v\n", err)
		return
	}

	if err := loadStreams(); err != nil {
		fmt.Printf("Error loading streams: %v\n", err)
		return
	}

	http.HandleFunc("/m3u8", handleM3u8)

	if err := http.ListenAndServe(fmt.Sprintf("%s:%s", httpAddress, httpPort), nil); err != nil {
		fmt.Printf("Error starting server: %v\n", err)
		return
	}
}
