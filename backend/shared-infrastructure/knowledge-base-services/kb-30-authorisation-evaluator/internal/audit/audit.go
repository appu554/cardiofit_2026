// Package audit hosts the regulator-facing query API for the kb-30
// Authorisation evaluator (Layer 3 v2 doc Part 4.5.4).
//
// Every authorisation evaluation is recorded as an EvaluationRecord. The
// Service exposes four sample queries:
//
//	Q1: For a resident over a date range, list every evaluation + decision.
//	Q2: For a credential ID, list every evaluation that relied on it.
//	Q3: For a jurisdiction + medication schedule, list evaluations in the
//	    last N days.
//	Q4: For a specific evaluation, surface the chain of authority back to
//	    the legislative reference.
//
// MVP storage is in-process. Production wiring will project these records
// onto the EvidenceTrace graph via kb-20.
package audit

import (
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"

	"kb-authorisation-evaluator/internal/dsl"
	"kb-authorisation-evaluator/internal/evaluator"
)

// EvaluationRecord captures one authorisation evaluation.
type EvaluationRecord struct {
	ID            uuid.UUID         `json:"id"`
	Query         evaluator.Query   `json:"query"`
	Result        evaluator.Result  `json:"result"`
	EvaluatedAt   time.Time         `json:"evaluated_at"`
	CredentialIDs []uuid.UUID       `json:"credential_ids,omitempty"`
}

// Service is the regulator query layer.
type Service struct {
	mu      sync.RWMutex
	records []EvaluationRecord
}

// NewService returns an empty in-memory audit service.
func NewService() *Service {
	return &Service{records: make([]EvaluationRecord, 0, 64)}
}

// Record appends an evaluation record.
func (s *Service) Record(r EvaluationRecord) {
	s.mu.Lock()
	s.records = append(s.records, r)
	s.mu.Unlock()
}

// All returns a copy of the underlying slice (test helper).
func (s *Service) All() []EvaluationRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]EvaluationRecord, len(s.records))
	copy(out, s.records)
	return out
}

// QueryByResident — Q1. Returns evaluations for residentRef whose
// EvaluatedAt falls in [from, to]. If from/to are zero, that bound is open.
func (s *Service) QueryByResident(residentRef uuid.UUID, from, to time.Time) []EvaluationRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []EvaluationRecord
	for _, r := range s.records {
		if r.Query.ResidentRef != residentRef {
			continue
		}
		if !from.IsZero() && r.EvaluatedAt.Before(from) {
			continue
		}
		if !to.IsZero() && r.EvaluatedAt.After(to) {
			continue
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].EvaluatedAt.Before(out[j].EvaluatedAt) })
	return out
}

// QueryByCredential — Q2. Returns evaluations whose CredentialIDs slice
// includes credID.
func (s *Service) QueryByCredential(credID uuid.UUID) []EvaluationRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []EvaluationRecord
	for _, r := range s.records {
		for _, c := range r.CredentialIDs {
			if c == credID {
				out = append(out, r)
				break
			}
		}
	}
	return out
}

// QueryByJurisdictionSchedule — Q3. Returns evaluations for the given
// jurisdiction + medication_schedule in the last `since` window.
func (s *Service) QueryByJurisdictionSchedule(jurisdiction, schedule string, since time.Duration) []EvaluationRecord {
	cutoff := time.Now().Add(-since)
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []EvaluationRecord
	for _, r := range s.records {
		if r.Query.Jurisdiction != jurisdiction {
			continue
		}
		if schedule != "" && r.Query.MedicationSchedule != schedule {
			continue
		}
		if r.EvaluatedAt.Before(cutoff) {
			continue
		}
		out = append(out, r)
	}
	return out
}

// AuthorisationChain — Q4. Surfaces the full chain from a single evaluation
// back to the legislative reference + rule + conditions evaluated.
type AuthorisationChain struct {
	Record               EvaluationRecord  `json:"record"`
	RuleID               string            `json:"rule_id,omitempty"`
	RuleVersion          int               `json:"rule_version,omitempty"`
	LegislativeReference string            `json:"legislative_reference,omitempty"`
	Decision             dsl.Decision      `json:"decision"`
	ConditionTrail       []evaluator.ConditionResult `json:"condition_trail,omitempty"`
}

// QueryAuthorisationChain — Q4.
func (s *Service) QueryAuthorisationChain(evalID uuid.UUID) (*AuthorisationChain, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, r := range s.records {
		if r.ID == evalID {
			return &AuthorisationChain{
				Record:               r,
				RuleID:               r.Result.RuleID,
				RuleVersion:          r.Result.RuleVersion,
				LegislativeReference: r.Result.LegislativeReference,
				Decision:             r.Result.Decision,
				ConditionTrail:       r.Result.Conditions,
			}, true
		}
	}
	return nil, false
}
