package services

import (
	"sync"
	"time"

	"kb-23-decision-cards/internal/models"
)

// InertiaVerdictHistory is a narrow abstraction over per-patient
// inertia-verdict persistence. Phase 7 P7-D needs the previous
// week's verdict to apply stability dampening — the interface stays
// small so both the in-memory implementation shipped now and the
// future PostgreSQL-backed implementation (with inertia_verdict_history
// table + migration) can satisfy it without changing callers.
type InertiaVerdictHistory interface {
	// SaveVerdict persists the given report for a patient, keyed on the
	// Monday of the week the batch ran. Subsequent calls with the same
	// (patient, week_start) upsert.
	SaveVerdict(patientID string, weekStart time.Time, report models.PatientInertiaReport) error

	// FetchLatest returns the most recently saved verdict for a patient,
	// or (zero, false) if the patient has no prior entry.
	FetchLatest(patientID string) (models.PatientInertiaReport, time.Time, bool)
}

// inMemoryInertiaHistory is a concurrent-safe in-memory implementation
// of InertiaVerdictHistory. Phase 7 P7-D ships with this instead of a
// PostgreSQL-backed store — the orchestrator's dampening check works
// against the previous run's verdict regardless of persistence layer,
// so the abstraction proof is complete without a new migration.
//
// The P7-D plan originally called for an inertia_verdict_history table
// + SQL migration + GORM repository. That work is deferred to Phase 8:
// the in-memory store resets on service restart, which for a weekly
// batch means the first run after a deployment does not apply
// dampening. That's an acceptable trade-off for the activation milestone
// — it biases toward more cards, not fewer, so no clinical signal is
// lost. Upgrading to a persistent store is additive (new implementation
// of this interface) and does not require changing the orchestrator or
// batch code.
type inMemoryInertiaHistory struct {
	mu      sync.RWMutex
	entries map[string]inertiaHistoryEntry
}

type inertiaHistoryEntry struct {
	report    models.PatientInertiaReport
	weekStart time.Time
}

// NewInMemoryInertiaHistory constructs an in-memory verdict history.
func NewInMemoryInertiaHistory() InertiaVerdictHistory {
	return &inMemoryInertiaHistory{
		entries: make(map[string]inertiaHistoryEntry),
	}
}

// SaveVerdict implements InertiaVerdictHistory — overwrites any
// existing entry for the patient (upsert on patient_id).
func (h *inMemoryInertiaHistory) SaveVerdict(patientID string, weekStart time.Time, report models.PatientInertiaReport) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.entries[patientID] = inertiaHistoryEntry{report: report, weekStart: weekStart}
	return nil
}

// FetchLatest implements InertiaVerdictHistory.
func (h *inMemoryInertiaHistory) FetchLatest(patientID string) (models.PatientInertiaReport, time.Time, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	entry, ok := h.entries[patientID]
	if !ok {
		return models.PatientInertiaReport{}, time.Time{}, false
	}
	return entry.report, entry.weekStart, true
}
