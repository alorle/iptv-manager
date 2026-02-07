package handlers

import (
	"fmt"
	"net/http"
)

// GetBaseURL returns the scheme and authority (scheme://host) from an HTTP request
func GetBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	return fmt.Sprintf("%s://%s", scheme, r.Host)
}
