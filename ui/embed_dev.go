//go:build dev

package ui

import "embed"

// DistFS is an empty filesystem in development mode.
// The actual frontend is served by the Vite dev server via reverse proxy.
var DistFS embed.FS
