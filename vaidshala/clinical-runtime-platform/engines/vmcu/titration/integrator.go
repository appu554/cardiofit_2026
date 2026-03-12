// integrator.go implements Commitment 1: Integrator freeze()/resume() (Phase 8.1).
//
// When the arbiter returns PAUSE or HALT, the integrator freezes.
// On resume, it restarts from the frozen dose value — no drift accumulation.
package titration

import (
	"sync"
	"time"
)

// IntegratorState tracks the freeze/resume state for a patient's titration.
type IntegratorState string

const (
	IntegratorActive IntegratorState = "ACTIVE"
	IntegratorFrozen IntegratorState = "FROZEN"
)

// Integrator tracks titration state with freeze/resume semantics.
// When frozen, the dose is locked at the pre-freeze value.
// On resume, titration restarts from that frozen value.
type Integrator struct {
	mu          sync.Mutex
	state       IntegratorState
	frozenDose  float64
	frozenAt    time.Time
	resumedAt   *time.Time
	pauseReason string
}

// NewIntegrator creates an active integrator with the given starting dose.
func NewIntegrator(currentDose float64) *Integrator {
	return &Integrator{
		state:      IntegratorActive,
		frozenDose: currentDose,
	}
}

// Freeze locks the integrator at the current dose.
// Called when arbiter returns PAUSE or HALT.
func (ig *Integrator) Freeze(currentDose float64, reason string) {
	ig.mu.Lock()
	defer ig.mu.Unlock()
	ig.state = IntegratorFrozen
	ig.frozenDose = currentDose
	ig.frozenAt = time.Now()
	ig.resumedAt = nil
	ig.pauseReason = reason
}

// Resume unlocks the integrator, returning to the frozen dose.
// The caller should use FrozenDose() as the starting point for the next cycle.
func (ig *Integrator) Resume() {
	ig.mu.Lock()
	defer ig.mu.Unlock()
	ig.state = IntegratorActive
	now := time.Now()
	ig.resumedAt = &now
}

// IsFrozen returns true if the integrator is currently frozen.
func (ig *Integrator) IsFrozen() bool {
	ig.mu.Lock()
	defer ig.mu.Unlock()
	return ig.state == IntegratorFrozen
}

// FrozenDose returns the dose at which the integrator was frozen.
func (ig *Integrator) FrozenDose() float64 {
	ig.mu.Lock()
	defer ig.mu.Unlock()
	return ig.frozenDose
}

// FrozenDuration returns how long the integrator has been frozen.
// Returns 0 if not frozen.
func (ig *Integrator) FrozenDuration() time.Duration {
	ig.mu.Lock()
	defer ig.mu.Unlock()
	if ig.state != IntegratorFrozen {
		return 0
	}
	return time.Since(ig.frozenAt)
}

// PauseHours returns the number of hours the integrator was paused.
// Uses resumedAt if available, otherwise current time.
func (ig *Integrator) PauseHours() float64 {
	ig.mu.Lock()
	defer ig.mu.Unlock()
	if ig.frozenAt.IsZero() {
		return 0
	}
	end := time.Now()
	if ig.resumedAt != nil {
		end = *ig.resumedAt
	}
	return end.Sub(ig.frozenAt).Hours()
}

// State returns the current integrator state.
func (ig *Integrator) State() IntegratorState {
	ig.mu.Lock()
	defer ig.mu.Unlock()
	return ig.state
}
