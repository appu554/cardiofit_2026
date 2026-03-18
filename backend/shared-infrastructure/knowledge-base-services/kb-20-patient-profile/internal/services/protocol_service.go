package services

import (
	"fmt"
	"time"

	"go.uber.org/zap"

	"kb-patient-profile/internal/database"
	"kb-patient-profile/internal/models"
)

// TransitionEvaluation holds the inputs for phase transition evaluation.
type TransitionEvaluation struct {
	ProtocolID           string
	CurrentPhase         string
	DaysInPhase          int
	ProteinAdherence     float64
	ExerciseAdherence    float64
	MealQualityScore     float64
	MealQualityImproving bool
	SafetyFlags          bool
	FBGWorsening         bool
	WeightLossKg         float64
	BMI                  float64
}

// TransitionDecision is the output of a phase transition evaluation.
type TransitionDecision struct {
	Action    string `json:"action"`
	NextPhase string `json:"next_phase,omitempty"`
	Reason    string `json:"reason"`
}

// EvaluatePRPTransition applies M3-PRP phase transition rules (Spec Section 4.2).
func EvaluatePRPTransition(eval TransitionEvaluation) TransitionDecision {
	if eval.SafetyFlags {
		return TransitionDecision{Action: "ABORT", Reason: "safety flag triggered (LS-01 eGFR check)"}
	}

	switch eval.CurrentPhase {
	case "STABILIZATION":
		if eval.DaysInPhase >= 14 && eval.ProteinAdherence >= 0.60 {
			return TransitionDecision{Action: "ADVANCE", NextPhase: "RESTORATION", Reason: "day >= 14 and protein adherence >= 60%"}
		}
		if eval.DaysInPhase >= 21 {
			return TransitionDecision{Action: "ESCALATE", Reason: "Phase 1 extended to max 21 days, adherence still < 60%"}
		}
		return TransitionDecision{Action: "HOLD", Reason: fmt.Sprintf("day %d, adherence %.0f%% — extend Phase 1", eval.DaysInPhase, eval.ProteinAdherence*100)}

	case "RESTORATION":
		if eval.DaysInPhase >= 28 && eval.ProteinAdherence >= 0.50 && eval.ExerciseAdherence >= 0.50 {
			return TransitionDecision{Action: "ADVANCE", NextPhase: "OPTIMIZATION", Reason: "day >= 42 with adequate adherence"}
		}
		if eval.FBGWorsening {
			return TransitionDecision{Action: "ESCALATE", Reason: "FBG worsening despite adherence — trigger KB-25 Comparator"}
		}
		if eval.DaysInPhase >= 42 {
			return TransitionDecision{Action: "HOLD", Reason: "extended Phase 2 — adherence below threshold"}
		}
		return TransitionDecision{Action: "HOLD", Reason: "Phase 2 in progress"}

	case "OPTIMIZATION":
		if eval.DaysInPhase >= 42 && eval.ProteinAdherence >= 0.70 {
			return TransitionDecision{Action: "ADVANCE", NextPhase: "GRADUATED", Reason: "12-week cycle complete with sustained adherence"}
		}
		if eval.DaysInPhase >= 63 && eval.FBGWorsening {
			return TransitionDecision{Action: "ESCALATE", Reason: "trajectory RED at Day 63 — trigger V-MCU medication review"}
		}
		return TransitionDecision{Action: "HOLD", Reason: "Phase 3 in progress"}
	}

	return TransitionDecision{Action: "HOLD", Reason: "unknown phase"}
}

// EvaluateVFRPTransition applies M3-VFRP phase transition rules (Spec Section 5.2).
func EvaluateVFRPTransition(eval TransitionEvaluation) TransitionDecision {
	if eval.SafetyFlags {
		return TransitionDecision{Action: "ABORT", Reason: "safety flag triggered"}
	}

	// VFRP safety guard: excessive weight loss in low-BMI patients
	if eval.WeightLossKg > 3.0 && eval.BMI > 0 && eval.BMI < 24 {
		return TransitionDecision{Action: "ABORT", Reason: "weight loss > 3kg with BMI < 24 — risk of muscle wasting, protocol suspended"}
	}

	switch eval.CurrentPhase {
	case "METABOLIC_STABILIZATION":
		if eval.DaysInPhase >= 14 && eval.MealQualityScore >= 50 {
			return TransitionDecision{Action: "ADVANCE", NextPhase: "FAT_MOBILIZATION", Reason: "day >= 14 and meal adherence >= 50%"}
		}
		if eval.DaysInPhase >= 28 {
			return TransitionDecision{Action: "ESCALATE", Reason: "Phase 1 extended beyond 28 days — meal adherence still < 50%, escalate to clinical review"}
		}
		if eval.DaysInPhase >= 21 {
			return TransitionDecision{Action: "HOLD", Reason: "extended Phase 1 to max — simplify substitutions"}
		}
		return TransitionDecision{Action: "HOLD", Reason: "Phase 1 in progress"}

	case "FAT_MOBILIZATION":
		if eval.DaysInPhase >= 28 && eval.ExerciseAdherence >= 0.50 && eval.MealQualityImproving {
			return TransitionDecision{Action: "ADVANCE", NextPhase: "SUSTAINED_REDUCTION", Reason: "adequate activity + improving meal quality"}
		}
		if eval.FBGWorsening {
			return TransitionDecision{Action: "ESCALATE", Reason: "FBG worsening despite adherence — medication addition"}
		}
		return TransitionDecision{Action: "HOLD", Reason: "Phase 2 in progress"}

	case "SUSTAINED_REDUCTION":
		if eval.DaysInPhase >= 42 {
			return TransitionDecision{Action: "ADVANCE", NextPhase: "GRADUATED", Reason: "12-week cycle complete"}
		}
		if eval.DaysInPhase >= 63 && eval.FBGWorsening {
			return TransitionDecision{Action: "ESCALATE", Reason: "trajectory RED at Day 63 — trigger V-MCU review"}
		}
		return TransitionDecision{Action: "HOLD", Reason: "Phase 3 in progress"}
	}

	return TransitionDecision{Action: "HOLD", Reason: "unknown phase"}
}

// ProtocolService manages protocol lifecycle (activate, transition, evaluate).
type ProtocolService struct {
	db       *database.Database
	registry *ProtocolRegistry
	eventBus *EventBus
	logger   *zap.Logger
}

// NewProtocolService creates a new protocol service.
func NewProtocolService(db *database.Database, registry *ProtocolRegistry, eventBus *EventBus, logger *zap.Logger) *ProtocolService {
	return &ProtocolService{db: db, registry: registry, eventBus: eventBus, logger: logger}
}

// ActivateProtocol starts a protocol for a patient.
func (ps *ProtocolService) ActivateProtocol(patientID string, protocolID string) (*models.ProtocolState, error) {
	var existing models.ProtocolState
	err := ps.db.DB.Where("patient_id = ? AND protocol_id = ? AND status = ?", patientID, protocolID, "ACTIVE").First(&existing).Error
	if err == nil {
		return nil, fmt.Errorf("protocol %s already active for patient %s", protocolID, patientID)
	}

	state := models.ProtocolState{
		PatientID:      patientID,
		ProtocolID:     protocolID,
		CurrentPhase:   "BASELINE",
		Status:         "ACTIVE",
		PhaseStartDate: time.Now().UTC(),
		ProtocolStart:  time.Now().UTC(),
		Trajectory:     "GREEN",
		CycleNumber:    1,
	}

	if err := ps.db.DB.Create(&state).Error; err != nil {
		return nil, fmt.Errorf("failed to activate protocol: %w", err)
	}

	ps.logger.Info("Protocol activated",
		zap.String("patient_id", patientID),
		zap.String("protocol_id", protocolID),
	)

	return &state, nil
}

// TransitionPhase advances a protocol to the next phase.
func (ps *ProtocolService) TransitionPhase(patientID string, protocolID string, nextPhase string) (*models.ProtocolState, error) {
	var state models.ProtocolState
	err := ps.db.DB.Where("patient_id = ? AND protocol_id = ? AND status = ?", patientID, protocolID, "ACTIVE").First(&state).Error
	if err != nil {
		return nil, fmt.Errorf("no active protocol %s for patient %s", protocolID, patientID)
	}

	if !state.CanTransition(nextPhase) {
		return nil, fmt.Errorf("invalid transition from %s to %s", state.CurrentPhase, nextPhase)
	}

	state.CurrentPhase = nextPhase
	state.PhaseStartDate = time.Now().UTC()
	state.PhaseExtended = false

	if nextPhase == "GRADUATED" {
		state.Status = "GRADUATED"
	}

	if err := ps.db.DB.Save(&state).Error; err != nil {
		return nil, fmt.Errorf("failed to transition phase: %w", err)
	}

	return &state, nil
}

// GetActiveProtocols returns all active protocols for a patient.
func (ps *ProtocolService) GetActiveProtocols(patientID string) ([]models.ProtocolState, error) {
	var states []models.ProtocolState
	err := ps.db.DB.Where("patient_id = ? AND status = ?", patientID, "ACTIVE").Find(&states).Error
	return states, err
}
