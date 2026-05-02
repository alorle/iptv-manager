package streaming

import (
	"context"
	"errors"
	"fmt"
	"io"
	"syscall"
	"testing"
)

func TestIsClientDisconnectError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"context.Canceled", context.Canceled, true},
		{"wrapped context.Canceled", fmt.Errorf("stream failed: %w", context.Canceled), true},
		{"io.ErrClosedPipe", io.ErrClosedPipe, true},
		{"ECONNRESET", syscall.ECONNRESET, true},
		{"wrapped ECONNRESET", fmt.Errorf("write tcp: %w", syscall.ECONNRESET), true},
		{"EPIPE", syscall.EPIPE, true},
		{"connection reset by peer string", errors.New("write tcp 10.0.0.1:8080->10.0.0.2:1234: connection reset by peer"), true},
		{"broken pipe string", errors.New("write: broken pipe"), true},
		{"unrelated error", errors.New("stream failed after 3 attempts"), false},
		{"context.DeadlineExceeded", context.DeadlineExceeded, false},
		{"engine unavailable", errors.New("acestream engine unavailable"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsClientDisconnectError(tt.err); got != tt.want {
				t.Errorf("IsClientDisconnectError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
