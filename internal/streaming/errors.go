package streaming

import (
	"context"
	"errors"
	"io"
	"strings"
	"syscall"
)

// IsClientDisconnectError reports whether err indicates the remote client
// closed the connection. These errors are expected during normal IPTV usage
// (e.g. channel switching) and should not be logged at ERROR level.
func IsClientDisconnectError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, context.Canceled) {
		return true
	}

	if errors.Is(err, io.ErrClosedPipe) {
		return true
	}

	if errors.Is(err, syscall.ECONNRESET) {
		return true
	}

	if errors.Is(err, syscall.EPIPE) {
		return true
	}

	errStr := err.Error()
	return strings.Contains(errStr, "connection reset by peer") ||
		strings.Contains(errStr, "broken pipe")
}
