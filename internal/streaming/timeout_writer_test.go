package streaming

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTimeoutWriter_Write(t *testing.T) {
	t.Run("successful write to buffer", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.Default()
		tw := NewTimeoutWriter(&buf, 1*time.Second, logger, "test-hash", "test-pid")

		data := []byte("test data")
		n, err := tw.Write(data)

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if n != len(data) {
			t.Errorf("expected %d bytes written, got %d", len(data), n)
		}
		if !bytes.Equal(buf.Bytes(), data) {
			t.Errorf("expected buffer to contain %q, got %q", string(data), buf.String())
		}
		if tw.BytesWritten() != int64(len(data)) {
			t.Errorf("expected bytes written %d, got %d", len(data), tw.BytesWritten())
		}
	})

	t.Run("write with zero timeout", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.Default()
		tw := NewTimeoutWriter(&buf, 0, logger, "test-hash", "test-pid")

		data := []byte("test data")
		n, err := tw.Write(data)

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if n != len(data) {
			t.Errorf("expected %d bytes written, got %d", len(data), n)
		}
	})

	t.Run("write to http.ResponseWriter", func(t *testing.T) {
		rec := httptest.NewRecorder()
		logger := slog.Default()
		tw := NewTimeoutWriter(rec, 100*time.Millisecond, logger, "test-hash", "test-pid")

		data := []byte("test data")
		n, err := tw.Write(data)

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if n != len(data) {
			t.Errorf("expected %d bytes written, got %d", len(data), n)
		}
		if !bytes.Equal(rec.Body.Bytes(), data) {
			t.Errorf("expected body to contain %q, got %q", string(data), rec.Body.String())
		}
	})

	t.Run("tracks bytes written across multiple writes", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.Default()
		tw := NewTimeoutWriter(&buf, 1*time.Second, logger, "test-hash", "test-pid")

		data1 := []byte("first ")
		data2 := []byte("second")

		n1, err := tw.Write(data1)
		if err != nil {
			t.Fatalf("first write failed: %v", err)
		}

		n2, err := tw.Write(data2)
		if err != nil {
			t.Fatalf("second write failed: %v", err)
		}

		expectedTotal := int64(n1 + n2)
		if tw.BytesWritten() != expectedTotal {
			t.Errorf("expected bytes written %d, got %d", expectedTotal, tw.BytesWritten())
		}

		expectedContent := "first second"
		if buf.String() != expectedContent {
			t.Errorf("expected buffer to contain %q, got %q", expectedContent, buf.String())
		}
	})
}

func TestIsTimeoutError(t *testing.T) {
	t.Run("recognizes nil as not timeout", func(t *testing.T) {
		if isTimeoutError(nil) {
			t.Error("expected nil to not be recognized as timeout error")
		}
	})

	t.Run("recognizes timeout error interface", func(t *testing.T) {
		err := &mockTimeoutError{timeout: true}
		if !isTimeoutError(err) {
			t.Error("expected timeout error to be recognized")
		}
	})

	t.Run("recognizes non-timeout error interface", func(t *testing.T) {
		err := &mockTimeoutError{timeout: false}
		if isTimeoutError(err) {
			t.Error("expected non-timeout error to not be recognized")
		}
	})

	t.Run("recognizes timeout in error string", func(t *testing.T) {
		testCases := []struct {
			name     string
			err      error
			expected bool
		}{
			{"contains 'timeout'", errors.New("connection timeout"), true},
			{"contains 'deadline'", errors.New("deadline exceeded"), true},
			{"contains 'i/o timeout'", errors.New("i/o timeout"), true},
			{"no timeout words", errors.New("connection refused"), false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := isTimeoutError(tc.err)
				if result != tc.expected {
					t.Errorf("expected %v for error %q, got %v", tc.expected, tc.err, result)
				}
			})
		}
	})
}

// mockTimeoutError implements the timeout error interface for testing.
type mockTimeoutError struct {
	timeout bool
}

func (e *mockTimeoutError) Error() string {
	if e.timeout {
		return "mock error with timeout condition"
	}
	return "mock error"
}

func (e *mockTimeoutError) Timeout() bool {
	return e.timeout
}

func (e *mockTimeoutError) Temporary() bool {
	return false
}

// slowWriter simulates a slow writer that takes time to write.
type slowWriter struct {
	delay time.Duration
}

func (w *slowWriter) Write(p []byte) (n int, err error) {
	time.Sleep(w.delay)
	return len(p), nil
}

// errorWriter always returns an error on write.
type errorWriter struct {
	err error
}

func (w *errorWriter) Write(p []byte) (n int, err error) {
	return 0, w.err
}

func TestTimeoutWriter_SlowWrite(t *testing.T) {
	t.Run("handles slow writer gracefully", func(t *testing.T) {
		// This test verifies that TimeoutWriter doesn't block indefinitely
		// Note: Actual timeout enforcement depends on the underlying writer
		// supporting deadlines (like http.ResponseWriter with ResponseController)
		sw := &slowWriter{delay: 50 * time.Millisecond}
		logger := slog.Default()
		tw := NewTimeoutWriter(sw, 100*time.Millisecond, logger, "test-hash", "test-pid")

		data := []byte("test data")
		n, err := tw.Write(data)

		if err != nil {
			t.Fatalf("expected no error for write within timeout, got %v", err)
		}
		if n != len(data) {
			t.Errorf("expected %d bytes written, got %d", len(data), n)
		}
	})
}

func TestTimeoutWriter_ErrorHandling(t *testing.T) {
	t.Run("propagates non-timeout errors", func(t *testing.T) {
		expectedErr := errors.New("connection refused")
		ew := &errorWriter{err: expectedErr}
		logger := slog.Default()
		tw := NewTimeoutWriter(ew, 1*time.Second, logger, "test-hash", "test-pid")

		data := []byte("test data")
		_, err := tw.Write(data)

		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("wraps timeout errors", func(t *testing.T) {
		timeoutErr := &mockTimeoutError{timeout: true}
		ew := &errorWriter{err: timeoutErr}
		logger := slog.Default()
		tw := NewTimeoutWriter(ew, 1*time.Second, logger, "test-hash", "test-pid")

		data := []byte("test data")
		_, err := tw.Write(data)

		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrWriteTimeout) {
			t.Errorf("expected ErrWriteTimeout, got %v", err)
		}
	})
}
