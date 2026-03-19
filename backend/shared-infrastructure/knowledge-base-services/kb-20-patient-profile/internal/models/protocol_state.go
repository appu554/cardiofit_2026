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

// Medication protocol phase orders (lifelong protocols have no GRADUATED terminal state).
var glyc1PhaseOrder = []string{"BASELINE", "MONOTHERAPY", "COMBINATION", "OPTIMIZATION"}
var htn1PhaseOrder = []string{"BASELINE", "MONOTHERAPY", "DUAL_THERAPY", "TRIPLE_THERAPY", "RESISTANT_HTN"}
var renal1PhaseOrder = []string{"BASELINE", "RAAS_OPTIMISATION", "SGLT2I_ADDITION", "FINERENONE_ADDITION", "MONITORING"}
var lipid1PhaseOrder = []string{"ASSESSMENT"} // single phase, card-only
var depresc1PhaseOrder = []string{"ASSESSMENT", "STEPDOWN", "MONITORING"}

// M3 lifestyle protocol phase orders.
// M3-MAINTAIN: indefinite maintenance — PARTNERSHIP has no GRADUATED terminal state.
var maintainPhaseOrder = []string{"CONSOLIDATION", "INDEPENDENCE", "STABILITY", "PARTNERSHIP"}

// M3-RECORRECTION: short correction cycle with GRADUATED terminal state.
var recorrectionPhaseOrder = []string{"ASSESSMENT", "CORRECTION", "GRADUATED"}

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
	case "GLYC-1":
		return glyc1PhaseOrder
	case "HTN-1":
		return htn1PhaseOrder
	case "RENAL-1":
		return renal1PhaseOrder
	case "LIPID-1":
		return lipid1PhaseOrder
	case "DEPRESC-1":
		return depresc1PhaseOrder
	case "M3-MAINTAIN":
		return maintainPhaseOrder
	case "M3-RECORRECTION":
		return recorrectionPhaseOrder
	default:
		return nil
	}
}
