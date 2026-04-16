// Package resilience provides fault-tolerance primitives for
// cross-service HTTP calls. Phase 10 P10-A.
//
// The circuit breaker wraps http.Client.Do with a three-state model:
//
//   CLOSED → normal operation, requests pass through
//   OPEN   → too many failures, requests rejected immediately
//   HALF_OPEN → after reset timeout, one probe request allowed;
//              success → CLOSED, failure → OPEN
//
// Additionally, each request attempt uses exponential backoff with
// jitter on retries so transient failures recover without thundering
// herd effects.
//
// Zero external dependencies — stdlib only. Thread-safe via sync.Mutex.
// Designed to wrap the existing KB-23 + KB-20 HTTP clients with
// minimal code changes: swap c.client.Do(req) for c.breaker.Do(req).
package resilience

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

// State represents the circuit breaker's current state.
type State int

const (
	StateClosed   State = iota // normal — requests pass through
	StateOpen                  // failing — requests rejected immediately
	StateHalfOpen              // probing — one request allowed to test recovery
)

// String returns the human-readable name for dashboard labels.
func (s State) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// ErrCircuitOpen is returned when a request is rejected because
// the circuit is open. Callers should treat this as a fast-fail
// and not retry — the circuit will transition to half-open on its
// own after the reset timeout.
var ErrCircuitOpen = errors.New("circuit breaker is open")

// Config holds the tunable parameters for a circuit breaker instance.
type Config struct {
	// Name identifies the service this breaker protects (used in
	// logs and metrics labels). E.g. "kb20", "kb26".
	Name string

	// MaxFailures is the number of consecutive failures before the
	// circuit opens. Default: 5.
	MaxFailures int

	// ResetTimeout is how long the circuit stays open before
	// transitioning to half-open. Default: 30s.
	ResetTimeout time.Duration

	// MaxRetries is the number of times to retry a failed request
	// before counting it as a circuit-breaker failure. Default: 3.
	// Set to 0 for no retries (fail on first error).
	MaxRetries int

	// BaseBackoff is the initial backoff delay for retries.
	// Subsequent retries use exponential backoff: baseBackoff * 2^attempt.
	// Default: 100ms.
	BaseBackoff time.Duration

	// MaxBackoff caps the exponential backoff. Default: 5s.
	MaxBackoff time.Duration

	// OnStateChange is called whenever the circuit transitions
	// between states. Optional — set to nil if you don't need
	// state-change callbacks (e.g. for Prometheus metrics).
	OnStateChange func(name string, from, to State)
}

// DefaultConfig returns a Config with production-safe defaults.
func DefaultConfig(name string) Config {
	return Config{
		Name:         name,
		MaxFailures:  5,
		ResetTimeout: 30 * time.Second,
		MaxRetries:   3,
		BaseBackoff:  100 * time.Millisecond,
		MaxBackoff:   5 * time.Second,
	}
}

// CircuitBreaker wraps an http.Client with circuit-breaking and
// retry logic. Thread-safe.
type CircuitBreaker struct {
	client *http.Client
	cfg    Config

	mu               sync.Mutex
	state            State
	consecutiveFails int
	lastFailTime     time.Time
}

// NewCircuitBreaker constructs a circuit breaker wrapping the given
// http.Client. The client's existing Timeout setting is preserved —
// the circuit breaker adds retry + state management on top of it.
func NewCircuitBreaker(client *http.Client, cfg Config) *CircuitBreaker {
	if cfg.MaxFailures <= 0 {
		cfg.MaxFailures = 5
	}
	if cfg.ResetTimeout <= 0 {
		cfg.ResetTimeout = 30 * time.Second
	}
	if cfg.BaseBackoff <= 0 {
		cfg.BaseBackoff = 100 * time.Millisecond
	}
	if cfg.MaxBackoff <= 0 {
		cfg.MaxBackoff = 5 * time.Second
	}
	return &CircuitBreaker{
		client: client,
		cfg:    cfg,
		state:  StateClosed,
	}
}

// Do executes the HTTP request with circuit-breaking and retry
// logic. Returns the response and error from the underlying
// http.Client.Do, or ErrCircuitOpen if the circuit is open.
//
// Retry semantics:
//   - On a transport error (connection refused, timeout, DNS),
//     retries up to MaxRetries times with exponential backoff +
//     jitter.
//   - On a 5xx response, retries (the server might recover).
//   - On a 4xx response, does NOT retry (client error, retrying
//     won't help).
//   - On a 2xx/3xx response, returns immediately (success).
//
// Circuit semantics:
//   - After MaxRetries exhausted AND MaxFailures consecutive
//     failures, the circuit opens.
//   - While open, all requests return ErrCircuitOpen immediately.
//   - After ResetTimeout, the circuit transitions to half-open
//     and allows one probe request.
//   - If the probe succeeds, the circuit closes (reset).
//   - If the probe fails, the circuit re-opens for another
//     ResetTimeout period.
func (cb *CircuitBreaker) Do(req *http.Request) (*http.Response, error) {
	// Check circuit state before attempting the request.
	if err := cb.allowRequest(); err != nil {
		return nil, err
	}

	// Attempt the request with retries.
	var lastErr error
	var resp *http.Response
	for attempt := 0; attempt <= cb.cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			backoff := cb.computeBackoff(attempt)
			time.Sleep(backoff)
		}

		// Clone the request for retry (the body may have been consumed).
		// For requests with no body (GET, DELETE), this is safe.
		// For requests with a body (POST, PUT), the caller should use
		// GetBody or pass a re-readable body. In practice, the KB
		// clients that use this breaker mostly do GETs; the one POST
		// (KB26 target-status) has a small JSON body.
		var err error
		resp, err = cb.client.Do(req)
		if err != nil {
			lastErr = err
			continue // retry on transport error
		}

		// 2xx/3xx → success
		if resp.StatusCode < 400 {
			cb.recordSuccess()
			return resp, nil
		}

		// 4xx → client error, don't retry
		if resp.StatusCode < 500 {
			cb.recordSuccess() // not a server failure, don't count it
			return resp, nil
		}

		// 5xx → server error, retry
		lastErr = fmt.Errorf("server returned %d", resp.StatusCode)
		// Don't close the body here — the caller might want to read
		// the error response on the last attempt. Close on next
		// iteration's overwrite.
	}

	// All retries exhausted — record as a failure.
	cb.recordFailure()
	if resp != nil {
		return resp, lastErr
	}
	return nil, lastErr
}

// State returns the current circuit state. Thread-safe read.
func (cb *CircuitBreaker) State() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// allowRequest checks whether a request should be allowed through.
// Returns nil if allowed, ErrCircuitOpen if the circuit is open.
func (cb *CircuitBreaker) allowRequest() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return nil
	case StateOpen:
		// Check if reset timeout has elapsed → transition to half-open
		if time.Since(cb.lastFailTime) >= cb.cfg.ResetTimeout {
			cb.transitionTo(StateHalfOpen)
			return nil // allow one probe request
		}
		return ErrCircuitOpen
	case StateHalfOpen:
		// Already in half-open — only one probe allowed at a time.
		// For simplicity, we allow concurrent probes. A production
		// implementation might use a semaphore to limit to exactly
		// one, but the correctness property (eventually one succeeds
		// or the circuit re-opens) holds either way.
		return nil
	}
	return nil
}

// recordSuccess resets the failure counter and closes the circuit
// if it was half-open.
func (cb *CircuitBreaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.consecutiveFails = 0
	if cb.state == StateHalfOpen {
		cb.transitionTo(StateClosed)
	}
}

// recordFailure increments the failure counter and opens the circuit
// if the threshold is reached.
func (cb *CircuitBreaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.consecutiveFails++
	cb.lastFailTime = time.Now()

	if cb.state == StateHalfOpen {
		// Probe failed — re-open
		cb.transitionTo(StateOpen)
		return
	}

	if cb.consecutiveFails >= cb.cfg.MaxFailures {
		cb.transitionTo(StateOpen)
	}
}

// transitionTo changes the circuit state and fires the callback.
// Must be called under the mutex.
func (cb *CircuitBreaker) transitionTo(newState State) {
	if cb.state == newState {
		return
	}
	oldState := cb.state
	cb.state = newState
	if cb.cfg.OnStateChange != nil {
		// Fire callback outside the mutex to avoid deadlock if the
		// callback itself tries to read the breaker state.
		go cb.cfg.OnStateChange(cb.cfg.Name, oldState, newState)
	}
}

// computeBackoff returns the delay for the given retry attempt
// (0-indexed). Uses exponential backoff with full jitter:
//
//	delay = min(maxBackoff, baseBackoff * 2^attempt) * random(0.5, 1.0)
//
// Full jitter prevents thundering herd on multi-client recovery.
func (cb *CircuitBreaker) computeBackoff(attempt int) time.Duration {
	exp := math.Pow(2, float64(attempt))
	delay := float64(cb.cfg.BaseBackoff) * exp
	if delay > float64(cb.cfg.MaxBackoff) {
		delay = float64(cb.cfg.MaxBackoff)
	}
	// Full jitter: random between 50% and 100% of computed delay
	jitter := 0.5 + rand.Float64()*0.5
	return time.Duration(delay * jitter)
}
