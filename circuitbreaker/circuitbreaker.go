package circuitbreaker

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/alorle/iptv-manager/logging"
)

// State represents the current state of the circuit breaker
type State int

const (
	// StateClosed means the circuit is operating normally
	StateClosed State = iota
	// StateOpen means the circuit is blocking all requests
	StateOpen
	// StateHalfOpen means the circuit is testing if it can close
	StateHalfOpen
)

// String returns the string representation of a State
func (s State) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF-OPEN"
	default:
		return "UNKNOWN"
	}
}

// Config contains the configuration for a circuit breaker
type Config struct {
	FailureThreshold   int           // Number of consecutive failures before opening
	Timeout            time.Duration // How long to wait in OPEN before transitioning to HALF-OPEN
	HalfOpenRequests   int           // Number of test requests allowed in HALF-OPEN state
	Logger             *logging.Logger // Logger for state changes (optional)
	ContentID          string        // Content ID for logging context (optional)
}

// CircuitBreaker defines the interface for circuit breaker functionality
type CircuitBreaker interface {
	// Execute runs the given function if the circuit allows it
	Execute(func() error) error
	// State returns the current state of the circuit breaker
	State() State
	// Reset resets the circuit breaker to CLOSED state
	Reset()
}

var (
	// ErrCircuitOpen is returned when the circuit breaker is in OPEN state
	ErrCircuitOpen = errors.New("circuit breaker is open")
	// ErrHalfOpenLimitReached is returned when too many requests are made in HALF-OPEN state
	ErrHalfOpenLimitReached = errors.New("circuit breaker half-open request limit reached")
)

// breaker is the concrete implementation of CircuitBreaker
type breaker struct {
	config Config
	mu     sync.RWMutex

	state            State
	failureCount     int
	halfOpenRequests int
	halfOpenSuccesses int
	openedAt         time.Time
	logger           *logging.Logger
	contentID        string
}

// New creates a new circuit breaker with the given configuration
func New(cfg Config) CircuitBreaker {
	if cfg.FailureThreshold <= 0 {
		cfg.FailureThreshold = 5
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.HalfOpenRequests <= 0 {
		cfg.HalfOpenRequests = 1
	}

	return &breaker{
		config:    cfg,
		state:     StateClosed,
		logger:    cfg.Logger,
		contentID: cfg.ContentID,
	}
}

// Execute runs the given function if the circuit allows it
func (b *breaker) Execute(fn func() error) error {
	b.mu.Lock()

	// Check if we should transition from OPEN to HALF-OPEN
	if b.state == StateOpen && time.Since(b.openedAt) >= b.config.Timeout {
		b.transitionTo(StateHalfOpen)
	}

	currentState := b.state

	// Handle state-specific logic
	switch currentState {
	case StateOpen:
		b.mu.Unlock()
		return ErrCircuitOpen

	case StateHalfOpen:
		if b.halfOpenRequests >= b.config.HalfOpenRequests {
			b.mu.Unlock()
			return ErrHalfOpenLimitReached
		}
		b.halfOpenRequests++
		b.mu.Unlock()

		// Execute the function
		err := fn()

		b.mu.Lock()
		if err != nil {
			// Failure in HALF-OPEN -> back to OPEN
			b.transitionTo(StateOpen)
			b.mu.Unlock()
			return err
		}

		// Success in HALF-OPEN
		b.halfOpenSuccesses++
		if b.halfOpenSuccesses >= b.config.HalfOpenRequests {
			// All test requests succeeded -> CLOSED
			b.transitionTo(StateClosed)
		}
		b.mu.Unlock()
		return nil

	case StateClosed:
		b.mu.Unlock()

		// Execute the function
		err := fn()

		b.mu.Lock()
		if err != nil {
			b.failureCount++
			if b.failureCount >= b.config.FailureThreshold {
				b.transitionTo(StateOpen)
			}
			b.mu.Unlock()
			return err
		}

		// Success -> reset failure count
		b.failureCount = 0
		b.mu.Unlock()
		return nil

	default:
		b.mu.Unlock()
		return fmt.Errorf("unknown circuit breaker state: %d", currentState)
	}
}

// State returns the current state of the circuit breaker
func (b *breaker) State() State {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.state
}

// Reset resets the circuit breaker to CLOSED state
func (b *breaker) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.transitionTo(StateClosed)
}

// transitionTo changes the circuit breaker state
// Must be called with lock held
func (b *breaker) transitionTo(newState State) {
	if b.state == newState {
		return
	}

	oldState := b.state
	b.state = newState

	// Log state change
	if b.logger != nil {
		b.logger.LogCircuitBreakerChange(oldState.String(), newState.String(), b.contentID)
	}

	switch newState {
	case StateClosed:
		b.failureCount = 0
		b.halfOpenRequests = 0
		b.halfOpenSuccesses = 0
		b.openedAt = time.Time{}

	case StateOpen:
		b.openedAt = time.Now()
		b.halfOpenRequests = 0
		b.halfOpenSuccesses = 0

	case StateHalfOpen:
		b.halfOpenRequests = 0
		b.halfOpenSuccesses = 0
	}
}
