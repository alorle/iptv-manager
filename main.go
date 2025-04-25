package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"

	middleware "github.com/oapi-codegen/nethttp-middleware"
	"github.com/rs/cors"

	"github.com/alorle/iptv-manager/internal/api"
	"github.com/alorle/iptv-manager/internal/handlers"
	internalJson "github.com/alorle/iptv-manager/internal/memory"
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
	fmt.Printf("acestreamURL: %v\n", acestreamURL)

	swagger, err := api.GetSwagger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading swagger spec\n: %s", err)
		os.Exit(1)
	}
	swagger.Servers = nil

	channels, err := loadChannels(streamsFile)
	if err != nil {
		fmt.Printf("Error creating JSONChannelRepository: %v\n", err)
		return
	}

	channelRepo, err := internalJson.NewInMemoryChannelsRepository(channels)
	if err != nil {
		fmt.Printf("Error creating JSONChannelRepository: %v\n", err)
		return
	}
	channelUseCase := usecase.NewChannelsUseCase(channelRepo)

	m := middleware.OapiRequestValidator(swagger)

	router := http.NewServeMux()

	server := api.NewServer(channelUseCase)
	h := api.NewStrictHandler(server, nil)

	router.Handle("/playlist.m3u", handlers.NewPlaylistHandler(channelUseCase, acestreamURL, epgUrl))
	router.Handle("/api/", http.StripPrefix("/api", m(api.HandlerFromMux(h, nil))))
	router.Handle("/api/documentation.json", handlers.NewDocumentationHandler(swagger))

	s := &http.Server{
		Handler: cors.AllowAll().Handler(router),
		Addr:    fmt.Sprintf("%s:%s", httpAddress, httpPort),
	}

	if err := s.ListenAndServe(); err != nil {
		fmt.Printf("Error starting server: %v\n", err)
		return
	}
}
