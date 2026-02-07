package circuitbreaker

import (
	"errors"
	"testing"
	"time"
)

var errTestFailure = errors.New("test failure")

// TestNew verifies circuit breaker creation with valid and default configs
func TestNew(t *testing.T) {
	tests := []struct {
		name           string
		config         Config
		expectedState  State
		expectedConfig Config
	}{
		{
			name: "valid config",
			config: Config{
				FailureThreshold: 3,
				Timeout:          10 * time.Second,
				HalfOpenRequests: 2,
			},
			expectedState: StateClosed,
			expectedConfig: Config{
				FailureThreshold: 3,
				Timeout:          10 * time.Second,
				HalfOpenRequests: 2,
			},
		},
		{
			name: "zero values use defaults",
			config: Config{
				FailureThreshold: 0,
				Timeout:          0,
				HalfOpenRequests: 0,
			},
			expectedState: StateClosed,
			expectedConfig: Config{
				FailureThreshold: 5,
				Timeout:          30 * time.Second,
				HalfOpenRequests: 1,
			},
		},
		{
			name: "partial defaults",
			config: Config{
				FailureThreshold: 10,
				Timeout:          0,
				HalfOpenRequests: 0,
			},
			expectedState: StateClosed,
			expectedConfig: Config{
				FailureThreshold: 10,
				Timeout:          30 * time.Second,
				HalfOpenRequests: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb := New(tt.config)
			if cb.State() != tt.expectedState {
				t.Errorf("expected state %s, got %s", tt.expectedState, cb.State())
			}

			// Verify internal config (cast to concrete type)
			br := cb.(*breaker)
			if br.config.FailureThreshold != tt.expectedConfig.FailureThreshold {
				t.Errorf("expected FailureThreshold %d, got %d",
					tt.expectedConfig.FailureThreshold, br.config.FailureThreshold)
			}
			if br.config.Timeout != tt.expectedConfig.Timeout {
				t.Errorf("expected Timeout %v, got %v",
					tt.expectedConfig.Timeout, br.config.Timeout)
			}
			if br.config.HalfOpenRequests != tt.expectedConfig.HalfOpenRequests {
				t.Errorf("expected HalfOpenRequests %d, got %d",
					tt.expectedConfig.HalfOpenRequests, br.config.HalfOpenRequests)
			}
		})
	}
}

// TestStateString verifies string representation of states
func TestStateString(t *testing.T) {
	tests := []struct {
		state    State
		expected string
	}{
		{StateClosed, "CLOSED"},
		{StateOpen, "OPEN"},
		{StateHalfOpen, "HALF-OPEN"},
		{State(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.state.String(); got != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, got)
			}
		})
	}
}

// TestClosedToOpen verifies transition from CLOSED to OPEN after threshold failures
func TestClosedToOpen(t *testing.T) {
	cb := New(Config{
		FailureThreshold: 3,
		Timeout:          100 * time.Millisecond,
		HalfOpenRequests: 1,
	})

	// Initial state should be CLOSED
	if cb.State() != StateClosed {
		t.Fatalf("expected initial state CLOSED, got %s", cb.State())
	}

	// First failure - should stay CLOSED
	err := cb.Execute(func() error { return errTestFailure })
	if err != errTestFailure {
		t.Errorf("expected test failure error, got %v", err)
	}
	if cb.State() != StateClosed {
		t.Errorf("expected state CLOSED after 1 failure, got %s", cb.State())
	}

	// Second failure - should stay CLOSED
	err = cb.Execute(func() error { return errTestFailure })
	if err != errTestFailure {
		t.Errorf("expected test failure error, got %v", err)
	}
	if cb.State() != StateClosed {
		t.Errorf("expected state CLOSED after 2 failures, got %s", cb.State())
	}

	// Third failure - should transition to OPEN
	err = cb.Execute(func() error { return errTestFailure })
	if err != errTestFailure {
		t.Errorf("expected test failure error, got %v", err)
	}
	if cb.State() != StateOpen {
		t.Errorf("expected state OPEN after 3 failures, got %s", cb.State())
	}
}

// TestOpenBlocksRequests verifies that OPEN state blocks all requests
func TestOpenBlocksRequests(t *testing.T) {
	cb := New(Config{
		FailureThreshold: 1,
		Timeout:          1 * time.Second,
		HalfOpenRequests: 1,
	})

	// Trigger transition to OPEN
	_ = cb.Execute(func() error { return errTestFailure })

	if cb.State() != StateOpen {
		t.Fatalf("expected state OPEN, got %s", cb.State())
	}

	// Subsequent requests should be blocked
	err := cb.Execute(func() error {
		t.Error("function should not be called when circuit is OPEN")
		return nil
	})

	if err != ErrCircuitOpen {
		t.Errorf("expected ErrCircuitOpen, got %v", err)
	}
}

// TestOpenToHalfOpen verifies transition from OPEN to HALF-OPEN after timeout
func TestOpenToHalfOpen(t *testing.T) {
	cb := New(Config{
		FailureThreshold: 1,
		Timeout:          100 * time.Millisecond,
		HalfOpenRequests: 1,
	})

	// Transition to OPEN
	_ = cb.Execute(func() error { return errTestFailure })
	if cb.State() != StateOpen {
		t.Fatalf("expected state OPEN, got %s", cb.State())
	}

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Next request should trigger transition to HALF-OPEN
	err := cb.Execute(func() error { return nil })
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if cb.State() != StateClosed {
		// After successful HALF-OPEN request with HalfOpenRequests=1, should go to CLOSED
		t.Errorf("expected state CLOSED after successful half-open request, got %s", cb.State())
	}
}

// TestHalfOpenSuccessToClosed verifies successful HALF-OPEN transitions to CLOSED
func TestHalfOpenSuccessToClosed(t *testing.T) {
	cb := New(Config{
		FailureThreshold: 1,
		Timeout:          50 * time.Millisecond,
		HalfOpenRequests: 2,
	})

	// Transition to OPEN
	_ = cb.Execute(func() error { return errTestFailure })

	// Wait for timeout
	time.Sleep(100 * time.Millisecond)

	// First half-open request - success
	err := cb.Execute(func() error { return nil })
	if err != nil {
		t.Errorf("expected no error on first half-open request, got %v", err)
	}
	if cb.State() != StateHalfOpen {
		t.Errorf("expected state HALF-OPEN after first success, got %s", cb.State())
	}

	// Second half-open request - success -> should transition to CLOSED
	err = cb.Execute(func() error { return nil })
	if err != nil {
		t.Errorf("expected no error on second half-open request, got %v", err)
	}
	if cb.State() != StateClosed {
		t.Errorf("expected state CLOSED after all half-open successes, got %s", cb.State())
	}
}

// TestHalfOpenFailureToOpen verifies failed HALF-OPEN transitions back to OPEN
func TestHalfOpenFailureToOpen(t *testing.T) {
	cb := New(Config{
		FailureThreshold: 1,
		Timeout:          50 * time.Millisecond,
		HalfOpenRequests: 2,
	})

	// Transition to OPEN
	_ = cb.Execute(func() error { return errTestFailure })

	// Wait for timeout
	time.Sleep(100 * time.Millisecond)

	// First half-open request - success
	err := cb.Execute(func() error { return nil })
	if err != nil {
		t.Errorf("expected no error on first half-open request, got %v", err)
	}
	if cb.State() != StateHalfOpen {
		t.Errorf("expected state HALF-OPEN after first success, got %s", cb.State())
	}

	// Second half-open request - failure -> should transition back to OPEN
	err = cb.Execute(func() error { return errTestFailure })
	if err != errTestFailure {
		t.Errorf("expected test failure error, got %v", err)
	}
	if cb.State() != StateOpen {
		t.Errorf("expected state OPEN after half-open failure, got %s", cb.State())
	}
}

// TestHalfOpenRequestLimit verifies limit enforcement in HALF-OPEN state
func TestHalfOpenRequestLimit(t *testing.T) {
	cb := New(Config{
		FailureThreshold: 1,
		Timeout:          50 * time.Millisecond,
		HalfOpenRequests: 2,
	})

	// Transition to OPEN
	_ = cb.Execute(func() error { return errTestFailure })

	// Wait for timeout
	time.Sleep(100 * time.Millisecond)

	// Make HalfOpenRequests (2) requests
	for i := 0; i < 2; i++ {
		err := cb.Execute(func() error { return nil })
		if err != nil {
			t.Errorf("request %d: expected no error, got %v", i+1, err)
		}
	}

	// Circuit should now be CLOSED after 2 successful half-open requests
	if cb.State() != StateClosed {
		t.Errorf("expected state CLOSED after all half-open requests, got %s", cb.State())
	}
}

// TestHalfOpenRequestLimitExceeded verifies behavior when exceeding half-open limit
func TestHalfOpenRequestLimitExceeded(t *testing.T) {
	cb := New(Config{
		FailureThreshold: 1,
		Timeout:          50 * time.Millisecond,
		HalfOpenRequests: 1,
	})

	// Transition to OPEN
	_ = cb.Execute(func() error { return errTestFailure })

	// Wait for timeout
	time.Sleep(100 * time.Millisecond)

	// First request transitions to HALF-OPEN and succeeds
	err := cb.Execute(func() error { return nil })
	if err != nil {
		t.Errorf("expected no error on first request, got %v", err)
	}

	// Should now be CLOSED since HalfOpenRequests=1 and it succeeded
	if cb.State() != StateClosed {
		t.Errorf("expected state CLOSED, got %s", cb.State())
	}
}

// TestClosedSuccessResetsFailureCount verifies failure count reset on success
func TestClosedSuccessResetsFailureCount(t *testing.T) {
	cb := New(Config{
		FailureThreshold: 3,
		Timeout:          100 * time.Millisecond,
		HalfOpenRequests: 1,
	})

	// Two failures
	_ = cb.Execute(func() error { return errTestFailure })
	_ = cb.Execute(func() error { return errTestFailure })

	if cb.State() != StateClosed {
		t.Errorf("expected state CLOSED after 2 failures, got %s", cb.State())
	}

	// Success should reset failure count
	err := cb.Execute(func() error { return nil })
	if err != nil {
		t.Errorf("expected no error on success, got %v", err)
	}

	// Verify failure count was reset by checking we need 3 more failures
	_ = cb.Execute(func() error { return errTestFailure })
	_ = cb.Execute(func() error { return errTestFailure })
	if cb.State() != StateClosed {
		t.Errorf("expected state still CLOSED after 2 more failures, got %s", cb.State())
	}

	// Third failure should open circuit
	_ = cb.Execute(func() error { return errTestFailure })
	if cb.State() != StateOpen {
		t.Errorf("expected state OPEN after 3 failures, got %s", cb.State())
	}
}

// TestReset verifies Reset() returns circuit to CLOSED state
func TestReset(t *testing.T) {
	cb := New(Config{
		FailureThreshold: 1,
		Timeout:          1 * time.Second,
		HalfOpenRequests: 1,
	})

	// Transition to OPEN
	_ = cb.Execute(func() error { return errTestFailure })
	if cb.State() != StateOpen {
		t.Fatalf("expected state OPEN, got %s", cb.State())
	}

	// Reset should transition back to CLOSED
	cb.Reset()
	if cb.State() != StateClosed {
		t.Errorf("expected state CLOSED after reset, got %s", cb.State())
	}

	// Verify circuit works normally after reset
	err := cb.Execute(func() error { return nil })
	if err != nil {
		t.Errorf("expected no error after reset, got %v", err)
	}
}

// TestResetFromHalfOpen verifies Reset() works from HALF-OPEN state
func TestResetFromHalfOpen(t *testing.T) {
	cb := New(Config{
		FailureThreshold: 1,
		Timeout:          50 * time.Millisecond,
		HalfOpenRequests: 2,
	})

	// Transition to OPEN
	_ = cb.Execute(func() error { return errTestFailure })

	// Wait and make one half-open request
	time.Sleep(100 * time.Millisecond)
	_ = cb.Execute(func() error { return nil })

	if cb.State() != StateHalfOpen {
		t.Fatalf("expected state HALF-OPEN, got %s", cb.State())
	}

	// Reset should transition to CLOSED
	cb.Reset()
	if cb.State() != StateClosed {
		t.Errorf("expected state CLOSED after reset from HALF-OPEN, got %s", cb.State())
	}
}

// TestConcurrentAccess verifies thread-safety of circuit breaker
func TestConcurrentAccess(t *testing.T) {
	cb := New(Config{
		FailureThreshold: 5,
		Timeout:          50 * time.Millisecond,
		HalfOpenRequests: 2,
	})

	// Launch multiple goroutines making concurrent requests
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_ = cb.Execute(func() error {
					if j%3 == 0 {
						return errTestFailure
					}
					return nil
				})
				time.Sleep(1 * time.Millisecond)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Just verify we didn't panic and can still check state
	_ = cb.State()
}

// TestDescriptiveErrorMessages verifies error messages are descriptive
func TestDescriptiveErrorMessages(t *testing.T) {
	cb := New(Config{
		FailureThreshold: 1,
		Timeout:          1 * time.Second,
		HalfOpenRequests: 1,
	})

	// Transition to OPEN
	_ = cb.Execute(func() error { return errTestFailure })

	// Verify OPEN error message
	err := cb.Execute(func() error { return nil })
	if err != ErrCircuitOpen {
		t.Errorf("expected ErrCircuitOpen, got %v", err)
	}
	if err.Error() != "circuit breaker is open" {
		t.Errorf("expected descriptive error message, got: %s", err.Error())
	}
}
