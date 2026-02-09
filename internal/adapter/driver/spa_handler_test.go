package driver

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func newTestSPAHandler() *SPAHandler {
	fsys := fstest.MapFS{
		"index.html":              {Data: []byte("<html><body>SPA</body></html>")},
		"assets/main.abc123.js":   {Data: []byte("console.log('app')")},
		"assets/style.def456.css": {Data: []byte("body{margin:0}")},
		"favicon.ico":             {Data: []byte("icon")},
	}
	return NewSPAHandler(fsys)
}

func TestSPAHandler_ServesIndexHTML(t *testing.T) {
	handler := newTestSPAHandler()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if got := rec.Body.String(); got != "<html><body>SPA</body></html>" {
		t.Errorf("unexpected body: %s", got)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-cache" {
		t.Errorf("expected no-cache for index.html, got %q", got)
	}
}

func TestSPAHandler_ServesStaticAsset(t *testing.T) {
	handler := newTestSPAHandler()
	req := httptest.NewRequest(http.MethodGet, "/assets/main.abc123.js", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if got := rec.Body.String(); got != "console.log('app')" {
		t.Errorf("unexpected body: %s", got)
	}
	if got := rec.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Errorf("expected long cache for assets, got %q", got)
	}
}

func TestSPAHandler_FallsBackToIndexHTML(t *testing.T) {
	handler := newTestSPAHandler()
	req := httptest.NewRequest(http.MethodGet, "/some/unknown/route", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if got := rec.Body.String(); got != "<html><body>SPA</body></html>" {
		t.Errorf("expected index.html content, got: %s", got)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-cache" {
		t.Errorf("expected no-cache for SPA fallback, got %q", got)
	}
}

func TestSPAHandler_ServesFavicon(t *testing.T) {
	handler := newTestSPAHandler()
	req := httptest.NewRequest(http.MethodGet, "/favicon.ico", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if got := rec.Body.String(); got != "icon" {
		t.Errorf("unexpected body: %s", got)
	}
}
