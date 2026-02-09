package driver

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// SPAHandler serves a single-page application from an embedded filesystem.
// It serves static files when they exist and falls back to index.html for
// unknown paths, enabling HTML5 history-based routing on the client side.
type SPAHandler struct {
	fileSystem fs.FS
	fileServer http.Handler
}

// NewSPAHandler creates a new handler that serves the SPA from the given filesystem.
func NewSPAHandler(fsys fs.FS) *SPAHandler {
	return &SPAHandler{
		fileSystem: fsys,
		fileServer: http.FileServerFS(fsys),
	}
}

// ServeHTTP serves a static file if it exists, otherwise serves index.html.
func (h *SPAHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cleanPath := path.Clean(r.URL.Path)
	if cleanPath == "/" {
		cleanPath = "/index.html"
	}

	// Strip leading slash for fs.Open
	filePath := strings.TrimPrefix(cleanPath, "/")

	f, err := h.fileSystem.Open(filePath)
	if err != nil {
		// File not found — serve index.html for SPA fallback
		r.URL.Path = "/"
		h.setCacheHeaders(w, "/index.html")
		h.fileServer.ServeHTTP(w, r)
		return
	}
	f.Close()

	h.setCacheHeaders(w, cleanPath)
	h.fileServer.ServeHTTP(w, r)
}

// setCacheHeaders sets appropriate cache headers based on the file path.
// Vite hashed assets (under /assets/) get long cache; index.html gets no-cache.
func (h *SPAHandler) setCacheHeaders(w http.ResponseWriter, filePath string) {
	if strings.HasPrefix(filePath, "/assets/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else if filePath == "/index.html" {
		w.Header().Set("Cache-Control", "no-cache")
	}
}
