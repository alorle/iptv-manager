package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/alorle/iptv-manager/internal/api"
	"github.com/alorle/iptv-manager/internal/handlers"
	internalJson "github.com/alorle/iptv-manager/internal/memory"
	"github.com/alorle/iptv-manager/internal/usecase"
	middleware "github.com/oapi-codegen/nethttp-middleware"
	"github.com/olivere/vite"
)

//go:embed all:dist
var dist embed.FS

var (
	httpAddress   = os.Getenv("HTTP_ADDRESS")
	httpPort      = os.Getenv("HTTP_PORT")
	streamsFile   = os.Getenv("STREAMS_FILE")
	acestreamBase = os.Getenv("ACESTREAM_URL")
	epgUrl        = os.Getenv("EPG_URL")
)

func main() {
	var (
		isDev = flag.Bool("dev", false, "run in development mode")
	)
	flag.Parse()

	c := vite.Config{
		FS:      os.DirFS("."),
		IsDev:   true,
		ViteURL: "http://localhost:5173",
	}
	if !*isDev {
		if fs, err := fs.Sub(dist, "dist"); err != nil {
			panic(err)
		} else {
			c = vite.Config{
				FS:    fs,
				IsDev: false,
			}
		}
	}
	viteHandler, err := vite.NewHandler(c)
	if err != nil {
		panic(err)
	}

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

	if streamsFile == "" {
		streamsFile = "streams.json"
	}

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

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			// Server the index.html file.
			ctx := r.Context()
			ctx = vite.MetadataToContext(ctx, vite.Metadata{
				Title: "Hello, Vite!",
			})
			ctx = vite.ScriptsToContext(ctx, `<script>console.log('Hello, nice to meet you in the console!')</script>`)
			viteHandler.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// Get the extension of the requested file
		ext := filepath.Ext(r.URL.Path)

		if len(ext) == 0 || r.URL.Path == "/api/documentation.json" || ext == ".m3u" {
			router.ServeHTTP(w, r)
			return
		}

		viteHandler.ServeHTTP(w, r)
	})

	if httpAddress == "" {
		httpAddress = "0.0.0.0"
	}
	if httpPort == "" {
		httpPort = "8080"
	}

	s := &http.Server{
		Handler: handler,
		Addr:    fmt.Sprintf("%s:%s", httpAddress, httpPort),
	}

	if err := s.ListenAndServe(); err != nil {
		fmt.Printf("Error starting server: %v\n", err)
		return
	}
}
