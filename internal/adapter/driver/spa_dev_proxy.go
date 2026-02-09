package driver

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

// SPADevProxy forwards all requests to a Vite development server,
// enabling hot module replacement during development.
type SPADevProxy struct {
	proxy *httputil.ReverseProxy
}

// NewSPADevProxy creates a new reverse proxy that forwards requests to the
// given target URL (e.g. "http://localhost:5173").
func NewSPADevProxy(target string) *SPADevProxy {
	targetURL, err := url.Parse(target)
	if err != nil {
		panic("spa dev proxy: invalid target URL: " + err.Error())
	}

	return &SPADevProxy{
		proxy: httputil.NewSingleHostReverseProxy(targetURL),
	}
}

// ServeHTTP forwards the request to the Vite development server.
func (h *SPADevProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.proxy.ServeHTTP(w, r)
}
