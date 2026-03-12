// cooldown.go implements Commitment 6: Dose cooldown (Phase 8.6).
//
// Minimum intervals between dose changes:
//   Basal insulin:       48 hours (A-03)
//   Rapid-acting insulin: 6 hours (A-03)
//
// If a dose change was applied less than the cooldown period ago,
// the titration engine blocks the proposed change.
package titration

import "time"

// MedicationClass categorizes medications for cooldown rules.
type MedicationClass string

const (
	MedClassBasalInsulin  MedicationClass = "BASAL_INSULIN"
	MedClassRapidInsulin  MedicationClass = "RAPID_INSULIN"
	MedClassOralAgent     MedicationClass = "ORAL_AGENT"
)

// CooldownConfig holds cooldown durations per medication class.
type CooldownConfig struct {
	BasalInsulinHours int // default: 48
	RapidInsulinHours int // default: 6
	OralAgentHours    int // default: 24
}

// DefaultCooldownConfig returns production-safe cooldown defaults.
func DefaultCooldownConfig() CooldownConfig {
	return CooldownConfig{
		BasalInsulinHours: 48,
		RapidInsulinHours: 6,
		OralAgentHours:    24,
	}
}

// DoseEvent records a dose change for cooldown tracking.
type DoseEvent struct {
	PatientID string
	MedClass  MedicationClass
	AppliedAt time.Time
	DoseDelta float64
}

// CooldownTracker enforces minimum intervals between dose changes.
type CooldownTracker struct {
	cfg        CooldownConfig
	lastChange map[string]map[MedicationClass]time.Time // patientID → medClass → timestamp
}

// NewCooldownTracker creates a tracker with the given config.
func NewCooldownTracker(cfg CooldownConfig) *CooldownTracker {
	return &CooldownTracker{
		cfg:        cfg,
		lastChange: make(map[string]map[MedicationClass]time.Time),
	}
}

// IsOnCooldown returns true if the patient's medication class is still
// within the cooldown period from the last dose change.
func (ct *CooldownTracker) IsOnCooldown(patientID string, medClass MedicationClass) bool {
	patient, ok := ct.lastChange[patientID]
	if !ok {
		return false
	}
	lastTime, ok := patient[medClass]
	if !ok {
		return false
	}
	cooldown := ct.cooldownDuration(medClass)
	return time.Since(lastTime) < cooldown
}

// RemainingCooldown returns how much time remains in the cooldown period.
// Returns 0 if not on cooldown.
func (ct *CooldownTracker) RemainingCooldown(patientID string, medClass MedicationClass) time.Duration {
	patient, ok := ct.lastChange[patientID]
	if !ok {
		return 0
	}
	lastTime, ok := patient[medClass]
	if !ok {
		return 0
	}
	cooldown := ct.cooldownDuration(medClass)
	remaining := cooldown - time.Since(lastTime)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// RecordDoseChange records a dose change event, starting the cooldown timer.
func (ct *CooldownTracker) RecordDoseChange(event DoseEvent) {
	if _, ok := ct.lastChange[event.PatientID]; !ok {
		ct.lastChange[event.PatientID] = make(map[MedicationClass]time.Time)
	}
	ct.lastChange[event.PatientID][event.MedClass] = event.AppliedAt
}

func (ct *CooldownTracker) cooldownDuration(medClass MedicationClass) time.Duration {
	switch medClass {
	case MedClassBasalInsulin:
		return time.Duration(ct.cfg.BasalInsulinHours) * time.Hour
	case MedClassRapidInsulin:
		return time.Duration(ct.cfg.RapidInsulinHours) * time.Hour
	case MedClassOralAgent:
		return time.Duration(ct.cfg.OralAgentHours) * time.Hour
	default:
		return 24 * time.Hour // safe default
	}
}
