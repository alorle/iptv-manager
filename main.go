package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/alorle/iptv-manager/internal/api"
	middleware "github.com/oapi-codegen/nethttp-middleware"
	"github.com/olivere/vite"
)

//go:embed all:dist
var dist embed.FS

var (
	httpAddress = os.Getenv("HTTP_ADDRESS")
	httpPort    = os.Getenv("HTTP_PORT")
)

func main() {
	var (
		isDev = flag.Bool("dev", false, "run in development mode")
	)
	flag.Parse()

	// Set defaults
	if httpAddress == "" {
		httpAddress = "0.0.0.0"
	}
	if httpPort == "" {
		httpPort = "8080"
	}

	// Configure Vite handler for serving frontend
	c := vite.Config{
		FS:      os.DirFS("."),
		IsDev:   true,
		ViteURL: "http://localhost:5173",
	}
	if !*isDev {
		fs, err := fs.Sub(dist, "dist")
		if err != nil {
			panic(err)
		}
		c = vite.Config{
			FS:    fs,
			IsDev: false,
		}
	}
	viteHandler, err := vite.NewHandler(c)
	if err != nil {
		panic(err)
	}

	// Load OpenAPI spec
	swagger, err := api.GetSwagger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading swagger spec\n: %s", err)
		os.Exit(1)
	}
	swagger.Servers = nil

	// Create API server (no dependencies needed for health check)
	server := api.NewServer()
	h := api.NewStrictHandler(server, nil)
	m := middleware.OapiRequestValidator(swagger)

	// Setup routes
	router := http.NewServeMux()
	router.Handle("/api/", http.StripPrefix("/api", m(api.HandlerFromMux(h, nil))))

	// Main handler with extension-based routing
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			// Serve the index.html file
			ctx := r.Context()
			ctx = vite.MetadataToContext(ctx, vite.Metadata{
				Title: "IPTV Manager",
			})
			viteHandler.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// Get the extension of the requested file
		ext := filepath.Ext(r.URL.Path)

		// If no extension, it's an API route
		if len(ext) == 0 {
			router.ServeHTTP(w, r)
			return
		}

		// Otherwise, serve static assets via Vite
		viteHandler.ServeHTTP(w, r)
	})

	// Start server
	s := &http.Server{
		Handler:           handler,
		Addr:              fmt.Sprintf("%s:%s", httpAddress, httpPort),
		ReadHeaderTimeout: 10 * time.Second,
	}

	fmt.Printf("Starting server on %s:%s\n", httpAddress, httpPort)
	if *isDev {
		fmt.Println("Running in DEVELOPMENT mode (proxying to Vite at http://localhost:5173)")
	} else {
		fmt.Println("Running in PRODUCTION mode (serving embedded assets)")
	}

	if err := s.ListenAndServe(); err != nil {
		fmt.Printf("Error starting server: %v\n", err)
		return
	}
}
