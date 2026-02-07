package ui

import (
	"embed"
	"io"
	"io/fs"
	"log"
	"mime"
	"net/http"
	"path"
	"strings"
)

//go:embed all:dist
var distFS embed.FS

// Handler returns an HTTP handler that serves the embedded UI
func Handler(mountPath string) http.Handler {
	// Strip the "dist" prefix from the embedded filesystem
	fsys, err := fs.Sub(distFS, "dist")
	if err != nil {
		log.Fatalf("Failed to create sub filesystem: %v", err)
	}

	return &uiHandler{
		mountPath: strings.TrimSuffix(mountPath, "/"),
		fs:        fsys,
	}
}

type uiHandler struct {
	mountPath string
	fs        fs.FS
}

func (h *uiHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Only allow GET and HEAD methods
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Remove mount path prefix from URL path
	urlPath := strings.TrimPrefix(r.URL.Path, h.mountPath)
	if urlPath == "" || urlPath == "/" {
		urlPath = "/index.html"
	}

	// Clean the path to prevent directory traversal
	urlPath = path.Clean(urlPath)
	urlPath = strings.TrimPrefix(urlPath, "/")

	// Try to open the requested file
	file, err := h.fs.Open(urlPath)
	if err != nil {
		// For SPA routing: if file doesn't exist and it's not an asset request,
		// serve index.html to let React Router handle it
		if !strings.Contains(urlPath, ".") {
			urlPath = "index.html"
			file, err = h.fs.Open(urlPath)
			if err != nil {
				http.Error(w, "Not found", http.StatusNotFound)
				return
			}
		} else {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("Warning: Failed to close file %s: %v", urlPath, err)
		}
	}()

	// Get file info for content length and modification time
	stat, err := file.Stat()
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Set Content-Type based on file extension
	ext := path.Ext(urlPath)
	if ext != "" {
		if contentType := mime.TypeByExtension(ext); contentType != "" {
			w.Header().Set("Content-Type", contentType)
		}
	}

	// For HTML files, disable caching to ensure SPA updates are picked up
	if ext == ".html" {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	} else {
		// For assets (JS, CSS, images), enable caching
		w.Header().Set("Cache-Control", "public, max-age=31536000")
	}

	// Serve the file content
	http.ServeContent(w, r, urlPath, stat.ModTime(), file.(io.ReadSeeker))
}
