// Package ethics_log provides a parallel audit log substrate for ethical decision events,
// per Ethical Architecture Guidelines §14.2. Every entry records an ethical event
// (concern, review, pattern, incident) linked to a decision_id from ethical_decision_metadata.
//
// VisibilityClass: AD (audit-defensible)
package ethics_log

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// EntryType classifies the kind of ethical event being logged.
type EntryType string

const (
	// EntryTypeDecision records a primary algorithmic decision event.
	EntryTypeDecision EntryType = "decision"
	// EntryTypeConcernFlagged records a flagged ethical concern.
	EntryTypeConcernFlagged EntryType = "concern_flagged"
	// EntryTypeReviewRequested records a request for human/ERM review.
	EntryTypeReviewRequested EntryType = "review_requested"
	// EntryTypePatternDetected records detection of a systematic ethical pattern.
	EntryTypePatternDetected EntryType = "pattern_detected"
	// EntryTypeIncident records a confirmed ethical incident.
	EntryTypeIncident EntryType = "incident"
)

// IsValidEntryType returns true when s is one of the five canonical EntryType values.
func IsValidEntryType(s string) bool {
	switch EntryType(s) {
	case EntryTypeDecision, EntryTypeConcernFlagged, EntryTypeReviewRequested,
		EntryTypePatternDetected, EntryTypeIncident:
		return true
	default:
		return false
	}
}

// Status represents the lifecycle state of an ethics log entry.
type Status string

const (
	// StatusOpen is the initial state for a new entry.
	StatusOpen Status = "open"
	// StatusInvestigating indicates the entry is under active review.
	StatusInvestigating Status = "investigating"
	// StatusRemediated indicates a corrective action has been taken.
	StatusRemediated Status = "remediated"
	// StatusVerified indicates the remediation has been verified effective.
	StatusVerified Status = "verified"
	// StatusClosed indicates the entry is fully resolved and closed.
	StatusClosed Status = "closed"
)

// IsValidStatus returns true when s is one of the five canonical Status values.
func IsValidStatus(s string) bool {
	switch Status(s) {
	case StatusOpen, StatusInvestigating, StatusRemediated, StatusVerified, StatusClosed:
		return true
	default:
		return false
	}
}

// Entry is a single record in the ethics audit log, linked to a decision in
// ethical_decision_metadata via DecisionID.
//
// VisibilityClass: AD (audit-defensible)
type Entry struct {
	// ID is the unique identifier for this log entry.
	ID uuid.UUID
	// DecisionID references ethical_decision_metadata(decision_id).
	DecisionID uuid.UUID
	// EntryType classifies the ethical event.
	EntryType EntryType
	// Severity is a 1..5 scale where 5 is most severe.
	Severity int
	// Description is a human-readable account of the event.
	Description string
	// Reviewer is an optional identifier of the reviewing party.
	Reviewer *string
	// ReviewOutcome is an optional summary of the review decision.
	ReviewOutcome *string
	// RemediationActions lists actions taken to address the entry.
	RemediationActions []string
	// Status is the current lifecycle state of this entry.
	Status Status
	// CreatedAt is the UTC time the entry was first written.
	CreatedAt time.Time
	// UpdatedAt is the UTC time the entry was last modified.
	UpdatedAt time.Time
}

// Store is the persistence boundary for Entry records.
type Store interface {
	Append(ctx context.Context, e Entry) error
	List(ctx context.Context) ([]Entry, error)
}

// Logger writes ethics log entries via the injected Store.
type Logger struct{ store Store }

// NewLogger returns a Logger backed by s.
func NewLogger(s Store) *Logger { return &Logger{store: s} }

// Append persists e. If e.ID is zero, Append generates a new UUID.
// If e.CreatedAt or e.UpdatedAt are zero, they are set to time.Now().UTC().
// If e.Status is empty, it defaults to StatusOpen.
func (l *Logger) Append(ctx context.Context, e Entry) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	now := time.Now().UTC()
	if e.CreatedAt.IsZero() {
		e.CreatedAt = now
	}
	if e.UpdatedAt.IsZero() {
		e.UpdatedAt = now
	}
	if e.Status == "" {
		e.Status = StatusOpen
	}
	return l.store.Append(ctx, e)
}

// InMemoryStore is a thread-safe in-memory implementation of Store, intended
// for testing and development use only.
type InMemoryStore struct {
	mu      sync.RWMutex
	entries []Entry
}

// NewInMemoryStore returns an empty InMemoryStore.
func NewInMemoryStore() *InMemoryStore { return &InMemoryStore{} }

// Append adds e to the store under a write lock.
func (s *InMemoryStore) Append(_ context.Context, e Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = append(s.entries, e)
	return nil
}

// List returns a snapshot copy of all entries under a read lock.
func (s *InMemoryStore) List(_ context.Context) ([]Entry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Entry, len(s.entries))
	copy(out, s.entries)
	return out, nil
}
