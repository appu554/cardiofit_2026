package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ProtocolState tracks a single patient's active protocol and current phase.
// One row per patient per active protocol.
type ProtocolState struct {
	ID             uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	PatientID      string         `gorm:"index;not null" json:"patient_id"`
	ProtocolID     string         `gorm:"index;not null" json:"protocol_id"`     // M3-PRP | M3-VFRP
	CurrentPhase   string         `gorm:"not null" json:"current_phase"`          // BASELINE | STABILIZATION | RESTORATION | OPTIMIZATION (PRP) or METABOLIC_STABILIZATION | FAT_MOBILIZATION | SUSTAINED_REDUCTION (VFRP)
	Status         string         `gorm:"not null;default:'ACTIVE'" json:"status"` // ACTIVE | GRADUATED | ESCALATED | ABORTED | EXTENDED
	PhaseStartDate time.Time      `gorm:"not null" json:"phase_start_date"`
	ProtocolStart  time.Time      `gorm:"not null" json:"protocol_start_date"`
	PhaseExtended  bool           `gorm:"default:false" json:"phase_extended"`
	Trajectory     string         `gorm:"default:'GREEN'" json:"trajectory"`       // GREEN | YELLOW | RED
	CycleNumber    int            `gorm:"default:1" json:"cycle_number"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

// ProtocolMetrics stores protocol-specific adherence metrics per state snapshot.
type ProtocolMetrics struct {
	ID                   uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	ProtocolStateID      uuid.UUID `gorm:"type:uuid;index;not null" json:"protocol_state_id"`
	ProteinAdherencePct  float64   `json:"protein_adherence_pct,omitempty"`
	ExerciseAdherencePct float64   `json:"exercise_adherence_pct,omitempty"`
	MealQualityScore     float64   `json:"meal_quality_score,omitempty"`
	DailySteps           int       `json:"daily_steps,omitempty"`
	MeasuredAt           time.Time `json:"measured_at"`
}

// Valid phase transitions per protocol.
var prpPhaseOrder = []string{"BASELINE", "STABILIZATION", "RESTORATION", "OPTIMIZATION", "GRADUATED"}
var vfrpPhaseOrder = []string{"BASELINE", "METABOLIC_STABILIZATION", "FAT_MOBILIZATION", "SUSTAINED_REDUCTION", "GRADUATED"}

// CanTransition checks if a phase transition is valid (must be sequential).
func (p *ProtocolState) CanTransition(nextPhase string) bool {
	order := p.phaseOrder()
	currentIdx := -1
	nextIdx := -1
	for i, phase := range order {
		if phase == p.CurrentPhase {
			currentIdx = i
		}
		if phase == nextPhase {
			nextIdx = i
		}
	}
	// Must advance exactly one phase forward
	return currentIdx >= 0 && nextIdx == currentIdx+1
}

// DaysInPhase returns the number of days the patient has been in the current phase.
func (p *ProtocolState) DaysInPhase() int {
	return int(time.Since(p.PhaseStartDate).Hours() / 24)
}

// DaysSinceStart returns total days since protocol activation.
func (p *ProtocolState) DaysSinceStart() int {
	return int(time.Since(p.ProtocolStart).Hours() / 24)
}

func (p *ProtocolState) phaseOrder() []string {
	switch p.ProtocolID {
	case "M3-PRP":
		return prpPhaseOrder
	case "M3-VFRP":
		return vfrpPhaseOrder
	default:
		return nil
	}
}
