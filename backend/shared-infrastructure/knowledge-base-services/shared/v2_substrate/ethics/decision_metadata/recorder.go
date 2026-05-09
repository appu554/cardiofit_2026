// Package decision_metadata records ethical decision metadata per Guidelines §14.1.
// Every algorithmic decision in the platform attaches metadata: which component,
// decision type, affected subject, principles implicated, ERM outcome,
// contestation enabled flag, and audit trace ref. Queries against this metadata
// power the detection mechanisms described in the Ethical Architecture Guidelines.
//
// VisibilityClass: AD (audit-defensible)
package decision_metadata

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Metadata captures the ethical decision context for a single algorithmic
// decision event. All fields are recorded at decision time.
//
// VisibilityClass: AD (audit-defensible)
type Metadata struct {
	DecisionID           uuid.UUID
	Component            string
	DecisionType         string
	AffectedSubjectID    string
	AffectedSubjectClass string // "resident" / "pharmacist" / "gp" / etc
	PrinciplesImplicated []string
	ERMReviewed          bool
	ERMOutcome           *string
	ContestationEnabled  bool
	AuditTraceRef        uuid.UUID
	Timestamp            time.Time
}

// Store is the persistence boundary for Metadata records.
type Store interface {
	Put(ctx context.Context, m Metadata) error
	Get(ctx context.Context, id uuid.UUID) (*Metadata, error)
	QueryBySubject(ctx context.Context, subjectID string) ([]Metadata, error)
}

// Recorder writes ethical decision metadata via the injected Store.
type Recorder struct{ store Store }

// NewRecorder returns a Recorder backed by s.
func NewRecorder(s Store) *Recorder { return &Recorder{store: s} }

// Record persists m. If m.Timestamp is zero, Record sets it to time.Now().UTC()
// before writing, ensuring every record carries a non-zero timestamp.
func (r *Recorder) Record(ctx context.Context, m Metadata) error {
	if m.Timestamp.IsZero() {
		m.Timestamp = time.Now().UTC()
	}
	return r.store.Put(ctx, m)
}

// IsValidPrinciple returns true when s is one of the seven canonical principle
// identifiers (P1..P7) defined in the Ethical Architecture Guidelines §1.
func IsValidPrinciple(s string) bool {
	switch s {
	case "P1", "P2", "P3", "P4", "P5", "P6", "P7":
		return true
	default:
		return false
	}
}
