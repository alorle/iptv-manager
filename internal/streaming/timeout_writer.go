package streaming

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

var (
	// ErrWriteTimeout indicates a write operation timed out.
	ErrWriteTimeout = errors.New("write timeout")
)

// TimeoutWriter wraps an io.Writer and enforces a write timeout.
// If a write operation takes longer than the configured timeout,
// it returns ErrWriteTimeout and the writer is considered slow.
type TimeoutWriter struct {
	dst          io.Writer
	timeout      time.Duration
	logger       *slog.Logger
	infoHash     string
	pid          string
	bytesWritten int64
}

// NewTimeoutWriter creates a new timeout-aware writer.
func NewTimeoutWriter(dst io.Writer, timeout time.Duration, logger *slog.Logger, infoHash, pid string) *TimeoutWriter {
	return &TimeoutWriter{
		dst:      dst,
		timeout:  timeout,
		logger:   logger,
		infoHash: infoHash,
		pid:      pid,
	}
}

// Write writes data to the underlying writer with a timeout.
// If the write doesn't complete within the timeout, it returns ErrWriteTimeout.
func (tw *TimeoutWriter) Write(p []byte) (n int, err error) {
	// For HTTP response writers, we need to set a write deadline
	// Since http.ResponseWriter doesn't support deadlines directly,
	// we use http.ResponseController for HTTP/2+ or rely on connection timeouts
	if rw, ok := tw.dst.(http.ResponseWriter); ok {
		// Try to use ResponseController for deadline support (Go 1.20+)
		rc := http.NewResponseController(rw)
		if tw.timeout > 0 {
			deadline := time.Now().Add(tw.timeout)
			if err := rc.SetWriteDeadline(deadline); err != nil {
				// If SetWriteDeadline fails, we can't enforce the timeout
				// Log and continue with the write
				tw.logger.Debug("failed to set write deadline",
					"infohash", tw.infoHash,
					"pid", tw.pid,
					"error", err)
			}
		}
	}

	// Perform the write
	n, err = tw.dst.Write(p)
	tw.bytesWritten += int64(n)

	// Check for timeout-related errors
	if err != nil {
		// Check if this is a timeout error
		if IsTimeoutError(err) {
			tw.logger.Warn("slow client detected - write timeout",
				"infohash", tw.infoHash,
				"pid", tw.pid,
				"timeout", tw.timeout,
				"bytes_written", tw.bytesWritten,
				"error", err)
			return n, fmt.Errorf("%w: %v", ErrWriteTimeout, err)
		}
	}

	return n, err
}

// BytesWritten returns the total number of bytes written successfully.
func (tw *TimeoutWriter) BytesWritten() int64 {
	return tw.bytesWritten
}

// IsTimeoutError checks if an error is a timeout error (context deadline,
// network timeout, or timeout-related error strings).
func IsTimeoutError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	type timeoutError interface {
		Timeout() bool
	}
	if te, ok := err.(timeoutError); ok && te.Timeout() {
		return true
	}

	errStr := err.Error()
	return strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "deadline")
}
