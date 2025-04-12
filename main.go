package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/alorle/iptv-manager/internal/handlers"
	"github.com/alorle/iptv-manager/internal/json"
	"github.com/alorle/iptv-manager/internal/usecase"
)

var (
	httpAddress   = os.Getenv("HTTP_ADDRESS")
	httpPort      = os.Getenv("HTTP_PORT")
	streamsFile   = os.Getenv("STREAMS_FILE")
	acestreamBase = os.Getenv("ACESTREAM_URL")
	epgUrl        = os.Getenv("EPG_URL")
)

func main() {
	flag.Parse()

	if acestreamBase == "" {
		acestreamBase = "http://127.0.0.1:6878/ace/getstream"
	}

	acestreamURL, err := url.Parse(acestreamBase)
	if err != nil {
		fmt.Printf("Error parsing ACESTREAM_URL: %v\n", err)
		return
	}

	channelRepo := json.NewJSONChannelRepository(streamsFile)
	channelUseCase := usecase.NewChannelUseCase(channelRepo)

	http.HandleFunc("/m3u8", handlers.NewM3U8Handler(channelUseCase, acestreamURL, epgUrl).HandleM3U8)

	if err := http.ListenAndServe(fmt.Sprintf("%s:%s", httpAddress, httpPort), nil); err != nil {
		fmt.Printf("Error starting server: %v\n", err)
		return
	}
}
