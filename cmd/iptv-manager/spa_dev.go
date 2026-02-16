//go:build dev

package main

import (
	"net/http"
	"os"

	"github.com/alorle/iptv-manager/internal/adapter/driver"
)

func newSPAHandler() http.Handler {
	target := os.Getenv("VITE_DEV_URL")
	if target == "" {
		target = "http://localhost:5173"
	}
	return driver.NewSPADevProxy(target)
}
