// Package eba_register provides the persistence boundary for Ethics-Based
// Auditing (EBA) findings recorded by the ethics-monitoring service.
//
// Phase 3 Task 1 ships only the interface and two in-process implementations
// (LogOnlyRegister, InMemoryRegister). A Postgres-backed implementation is
// planned for a follow-up task and will be wired against migration
// 045_eba_register.sql (already authored under migrations/).
//
// VisibilityClass: AD
package eba_register

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Finding is a single EBA finding row destined for the eba_register table.
type Finding struct {
	ID          uuid.UUID
	FindingType string
	Severity    int // 1..5
	Description string
	Status      string // defaults to "open" when empty
	DetectedAt  time.Time
	ClosedAt    *time.Time
}

// Register is the append-only persistence boundary for EBA findings.
type Register interface {
	Append(ctx context.Context, f Finding) error
}

// LogOnlyRegister is a no-op Register that emits the finding to the standard
// logger. Intended for very-early bootstrap where Postgres is not yet wired.
type LogOnlyRegister struct{}

// Append logs f and returns nil.
func (LogOnlyRegister) Append(_ context.Context, f Finding) error {
	log.Printf("eba_register: finding type=%s severity=%d desc=%q", f.FindingType, f.Severity, f.Description)
	return nil
}

// InMemoryRegister is a thread-safe in-memory Register, intended for tests.
type InMemoryRegister struct {
	mu       sync.RWMutex
	findings []Finding
}

// NewInMemoryRegister returns an empty InMemoryRegister.
func NewInMemoryRegister() *InMemoryRegister { return &InMemoryRegister{} }

// Append stores f. If f.ID is zero a new UUID is generated. If f.DetectedAt is
// zero it is set to time.Now().UTC(). If f.Status is empty it defaults to
// "open".
func (r *InMemoryRegister) Append(_ context.Context, f Finding) error {
	if f.ID == uuid.Nil {
		f.ID = uuid.New()
	}
	if f.DetectedAt.IsZero() {
		f.DetectedAt = time.Now().UTC()
	}
	if f.Status == "" {
		f.Status = "open"
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.findings = append(r.findings, f)
	return nil
}

// List returns a snapshot copy of all stored findings.
func (r *InMemoryRegister) List() []Finding {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Finding, len(r.findings))
	copy(out, r.findings)
	return out
}
