//go:build !dev

package main

import (
	"io/fs"
	"log"
	"net/http"

	"github.com/alorle/iptv-manager/internal/adapter/driver"
	"github.com/alorle/iptv-manager/ui"
)

func newSPAHandler() http.Handler {
	distFS, err := fs.Sub(ui.DistFS, "dist")
	if err != nil {
		log.Fatalf("failed to create sub filesystem for SPA: %v", err)
	}
	return driver.NewSPAHandler(distFS)
}
