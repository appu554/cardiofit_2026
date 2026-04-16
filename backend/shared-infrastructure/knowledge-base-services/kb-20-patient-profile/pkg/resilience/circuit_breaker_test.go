package resilience

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// TestCircuitBreaker_ClosedState_PassesThrough verifies that a
// freshly-constructed circuit breaker in CLOSED state passes
// requests through to the underlying http.Client without
// interference.
func TestCircuitBreaker_ClosedState_PassesThrough(t *testing.T) {
	var callCount int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&callCount, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := DefaultConfig("test")
	cfg.MaxRetries = 0 // no retries for this test
	cb := NewCircuitBreaker(server.Client(), cfg)

	req, _ := http.NewRequest(http.MethodGet, server.URL, nil)
	resp, err := cb.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
	if atomic.LoadInt64(&callCount) != 1 {
		t.Errorf("callCount = %d, want 1", callCount)
	}
	if cb.State() != StateClosed {
		t.Errorf("state = %v, want CLOSED", cb.State())
	}
}

// TestCircuitBreaker_OpensAfterMaxFailures verifies that the circuit
// transitions to OPEN after MaxFailures consecutive 500 responses
// (with retries exhausted each time).
func TestCircuitBreaker_OpensAfterMaxFailures(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := DefaultConfig("test")
	cfg.MaxFailures = 3
	cfg.MaxRetries = 0    // no retries — each call counts as one failure
	cfg.ResetTimeout = 1 * time.Hour // don't auto-reset during test

	cb := NewCircuitBreaker(server.Client(), cfg)

	for i := 0; i < 3; i++ {
		req, _ := http.NewRequest(http.MethodGet, server.URL, nil)
		cb.Do(req)
	}

	if cb.State() != StateOpen {
		t.Errorf("state = %v, want OPEN after %d failures", cb.State(), cfg.MaxFailures)
	}
}

// TestCircuitBreaker_OpenRejectsImmediately verifies that an OPEN
// circuit returns ErrCircuitOpen without making any HTTP call.
func TestCircuitBreaker_OpenRejectsImmediately(t *testing.T) {
	var callCount int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&callCount, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := DefaultConfig("test")
	cfg.MaxFailures = 1
	cfg.MaxRetries = 0
	cfg.ResetTimeout = 1 * time.Hour

	cb := NewCircuitBreaker(server.Client(), cfg)

	// Trip the circuit
	req, _ := http.NewRequest(http.MethodGet, server.URL, nil)
	cb.Do(req)

	// Next call should be rejected immediately
	callsBefore := atomic.LoadInt64(&callCount)
	req2, _ := http.NewRequest(http.MethodGet, server.URL, nil)
	_, err := cb.Do(req2)
	if err != ErrCircuitOpen {
		t.Errorf("expected ErrCircuitOpen, got %v", err)
	}
	if atomic.LoadInt64(&callCount) != callsBefore {
		t.Error("open circuit should NOT make an HTTP call")
	}
}

// TestCircuitBreaker_HalfOpenProbeSuccess verifies the recovery
// path: after ResetTimeout, the circuit transitions to HALF_OPEN,
// allows one probe, and on success transitions back to CLOSED.
func TestCircuitBreaker_HalfOpenProbeSuccess(t *testing.T) {
	var shouldFail int64 = 1
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.LoadInt64(&shouldFail) == 1 {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	cfg := DefaultConfig("test")
	cfg.MaxFailures = 1
	cfg.MaxRetries = 0
	cfg.ResetTimeout = 50 * time.Millisecond // short for testing

	cb := NewCircuitBreaker(server.Client(), cfg)

	// Trip the circuit
	req, _ := http.NewRequest(http.MethodGet, server.URL, nil)
	cb.Do(req)

	if cb.State() != StateOpen {
		t.Fatalf("expected OPEN, got %v", cb.State())
	}

	// Wait for reset timeout
	time.Sleep(100 * time.Millisecond)

	// Server recovers
	atomic.StoreInt64(&shouldFail, 0)

	// Probe request should succeed and close the circuit
	req2, _ := http.NewRequest(http.MethodGet, server.URL, nil)
	resp, err := cb.Do(req2)
	if err != nil {
		t.Fatalf("probe request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("probe StatusCode = %d, want 200", resp.StatusCode)
	}

	// Give the state-change callback goroutine a moment to fire
	time.Sleep(10 * time.Millisecond)

	if cb.State() != StateClosed {
		t.Errorf("state = %v, want CLOSED after successful probe", cb.State())
	}
}

// TestCircuitBreaker_HalfOpenProbeFailure verifies that a failed
// probe in HALF_OPEN re-opens the circuit.
func TestCircuitBreaker_HalfOpenProbeFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := DefaultConfig("test")
	cfg.MaxFailures = 1
	cfg.MaxRetries = 0
	cfg.ResetTimeout = 50 * time.Millisecond

	cb := NewCircuitBreaker(server.Client(), cfg)

	// Trip the circuit
	req, _ := http.NewRequest(http.MethodGet, server.URL, nil)
	cb.Do(req)

	// Wait for reset timeout → half-open
	time.Sleep(100 * time.Millisecond)

	// Probe request still fails → re-open
	req2, _ := http.NewRequest(http.MethodGet, server.URL, nil)
	cb.Do(req2)

	// Give the state-change callback goroutine a moment
	time.Sleep(10 * time.Millisecond)

	if cb.State() != StateOpen {
		t.Errorf("state = %v, want OPEN after failed probe", cb.State())
	}
}

// TestCircuitBreaker_RetriesOnServerError verifies that a 500
// response triggers retries up to MaxRetries before counting as
// a single failure.
func TestCircuitBreaker_RetriesOnServerError(t *testing.T) {
	var callCount int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&callCount, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := DefaultConfig("test")
	cfg.MaxFailures = 5
	cfg.MaxRetries = 2   // 1 initial + 2 retries = 3 total calls
	cfg.BaseBackoff = 1 * time.Millisecond // fast for testing

	cb := NewCircuitBreaker(server.Client(), cfg)

	req, _ := http.NewRequest(http.MethodGet, server.URL, nil)
	cb.Do(req)

	if atomic.LoadInt64(&callCount) != 3 {
		t.Errorf("callCount = %d, want 3 (1 initial + 2 retries)", callCount)
	}
}

// TestCircuitBreaker_NoRetryOn4xx verifies that a 4xx response does
// NOT trigger retries — it's a client error, retrying won't help.
func TestCircuitBreaker_NoRetryOn4xx(t *testing.T) {
	var callCount int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&callCount, 1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cfg := DefaultConfig("test")
	cfg.MaxRetries = 3 // would retry 3 times if it were a 5xx

	cb := NewCircuitBreaker(server.Client(), cfg)

	req, _ := http.NewRequest(http.MethodGet, server.URL, nil)
	resp, _ := cb.Do(req)

	if atomic.LoadInt64(&callCount) != 1 {
		t.Errorf("callCount = %d, want 1 (no retry on 4xx)", callCount)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode = %d, want 404", resp.StatusCode)
	}
}

// TestCircuitBreaker_SuccessResetsFailureCount verifies that a
// successful request resets the consecutive failure counter so the
// circuit doesn't open on intermittent errors.
func TestCircuitBreaker_SuccessResetsFailureCount(t *testing.T) {
	var callIdx int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idx := atomic.AddInt64(&callIdx, 1)
		// Alternate: fail, fail, succeed, fail, fail — should NOT open
		// a circuit with MaxFailures=3 because the success resets the
		// consecutive counter.
		if idx == 3 {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	cfg := DefaultConfig("test")
	cfg.MaxFailures = 3
	cfg.MaxRetries = 0

	cb := NewCircuitBreaker(server.Client(), cfg)

	for i := 0; i < 5; i++ {
		req, _ := http.NewRequest(http.MethodGet, server.URL, nil)
		cb.Do(req)
	}

	// The circuit should NOT be open because the success at call 3
	// reset the consecutive failure counter. Calls 4-5 are only
	// 2 consecutive failures (below MaxFailures=3).
	if cb.State() != StateClosed {
		t.Errorf("state = %v, want CLOSED (success at call 3 should reset counter)", cb.State())
	}
}

// TestCircuitBreaker_StateChangeCallback verifies that the
// OnStateChange callback fires on every transition.
func TestCircuitBreaker_StateChangeCallback(t *testing.T) {
	var transitions []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := DefaultConfig("test-cb")
	cfg.MaxFailures = 1
	cfg.MaxRetries = 0
	cfg.ResetTimeout = 50 * time.Millisecond
	cfg.OnStateChange = func(name string, from, to State) {
		transitions = append(transitions, fmt.Sprintf("%s: %s→%s", name, from, to))
	}

	cb := NewCircuitBreaker(server.Client(), cfg)

	// Trip the circuit: CLOSED → OPEN
	req, _ := http.NewRequest(http.MethodGet, server.URL, nil)
	cb.Do(req)

	// Wait for reset timeout: OPEN → HALF_OPEN
	time.Sleep(100 * time.Millisecond)

	// Probe (fails): HALF_OPEN → OPEN
	req2, _ := http.NewRequest(http.MethodGet, server.URL, nil)
	cb.Do(req2)

	// Give async callbacks time to fire
	time.Sleep(50 * time.Millisecond)

	if len(transitions) < 2 {
		t.Errorf("expected at least 2 transitions, got %d: %v", len(transitions), transitions)
	}
}

// TestCircuitBreaker_ComputeBackoff verifies that backoff increases
// exponentially and is capped at MaxBackoff.
func TestCircuitBreaker_ComputeBackoff(t *testing.T) {
	cfg := DefaultConfig("test")
	cfg.BaseBackoff = 100 * time.Millisecond
	cfg.MaxBackoff = 2 * time.Second
	cb := NewCircuitBreaker(&http.Client{}, cfg)

	// Attempt 1: ~200ms (100ms * 2^1 * jitter)
	b1 := cb.computeBackoff(1)
	if b1 < 100*time.Millisecond || b1 > 250*time.Millisecond {
		t.Errorf("backoff(1) = %v, expected ~100-250ms", b1)
	}

	// Attempt 5: should be capped at MaxBackoff (2s)
	b5 := cb.computeBackoff(5)
	if b5 > 2*time.Second+100*time.Millisecond {
		t.Errorf("backoff(5) = %v, expected <= MaxBackoff 2s + jitter", b5)
	}
}

// TestState_String covers the String method for dashboard label
// correctness.
func TestState_String(t *testing.T) {
	if StateClosed.String() != "CLOSED" {
		t.Errorf("StateClosed.String() = %q", StateClosed.String())
	}
	if StateOpen.String() != "OPEN" {
		t.Errorf("StateOpen.String() = %q", StateOpen.String())
	}
	if StateHalfOpen.String() != "HALF_OPEN" {
		t.Errorf("StateHalfOpen.String() = %q", StateHalfOpen.String())
	}
}
