package services

import (
	"fmt"
	"time"

	"go.uber.org/zap"

	"kb-patient-profile/internal/clients"
	"kb-patient-profile/internal/database"
	"kb-patient-profile/internal/models"
)

// EventPublisher is the interface used by ProtocolService to publish events.
// Using an interface allows test doubles (spies) to be injected without
// requiring a real EventBus backed by a database.
type EventPublisher interface {
	Publish(eventType string, patientID string, payload interface{})
}

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
	// EGFRDelta is the change in eGFR (mL/min) since phase entry.
	// A positive value indicates decline (eGFR fell by this amount).
	EGFRDelta float64

	// Medication protocol fields
	HbA1cAboveTarget bool // true if current HbA1c exceeds protocol target
	SBPAboveTarget   bool // true if current SBP exceeds protocol target
	ACRNotImproving  bool // true if ACR not improving after intervention

	// M3-MAINTAIN fields
	MRIScore              float64 `json:"mri_score,omitempty"`
	MRISustainedDays      int     `json:"mri_sustained_days,omitempty"`
	AdherencePct          float64 `json:"adherence_pct,omitempty"`
	ConsecutiveCheckins   int     `json:"consecutive_checkins,omitempty"`
	NoRelapseDays         int     `json:"no_relapse_days,omitempty"`
	HbA1cAtTarget         bool    `json:"hba1c_at_target,omitempty"`
	HbA1cAtTargetReadings int     `json:"hba1c_at_target_readings,omitempty"`
	YearReviewComplete    bool    `json:"year_review_complete,omitempty"`
	PhysicianGradApproval bool    `json:"physician_grad_approval,omitempty"`
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
		// G-9: Safety-first — check eGFR decline before any advancement logic.
		if eval.EGFRDelta > 5 {
			return TransitionDecision{Action: "ESCALATE", Reason: "eGFR declined >5 during restoration"}
		}
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
	db        *database.Database
	registry  *ProtocolRegistry
	eventBus  EventPublisher
	logger    *zap.Logger
	kb25      *clients.KB25Client // optional — nil disables KB-25 integration
}

// NewProtocolService creates a new protocol service.
func NewProtocolService(db *database.Database, registry *ProtocolRegistry, eventBus EventPublisher, logger *zap.Logger) *ProtocolService {
	return &ProtocolService{db: db, registry: registry, eventBus: eventBus, logger: logger}
}

// SetKB25Client attaches the KB-25 Lifestyle Knowledge Graph client to the service.
// When set, ActivateProtocol will call KB-25 for a pre-activation safety check and
// TransitionPhase will call KB-25 for projected outcomes (best-effort, non-blocking).
func (ps *ProtocolService) SetKB25Client(c *clients.KB25Client) {
	ps.kb25 = c
}

// ActivateProtocol starts a protocol for a patient.
//
// If a KB-25 client is configured, a safety check is performed before the protocol
// state is written. KB-25 hard-stop rules (e.g. LS-01: eGFR < 15) cause the
// activation to be rejected with a descriptive error. If KB-25 is unreachable the
// safety check is skipped and activation proceeds (fail-open to avoid blocking
// clinical workflows when KB-25 is temporarily down).
func (ps *ProtocolService) ActivateProtocol(patientID string, protocolID string) (*models.ProtocolState, error) {
	var existing models.ProtocolState
	err := ps.db.DB.Where("patient_id = ? AND protocol_id = ? AND status = ?", patientID, protocolID, "ACTIVE").First(&existing).Error
	if err == nil {
		return nil, fmt.Errorf("protocol %s already active for patient %s", protocolID, patientID)
	}

	// G-6: KB-25 lifestyle safety check before activation.
	if ps.kb25 != nil {
		safetyResp, safetyErr := ps.kb25.CheckSafety(clients.SafetyCheckRequest{
			PatientID:  patientID,
			ProtocolID: protocolID,
		})
		if safetyErr != nil {
			// KB-25 unreachable — log and proceed (fail-open).
			ps.logger.Warn("KB-25 safety check unavailable — proceeding with activation",
				zap.String("patient_id", patientID),
				zap.String("protocol_id", protocolID),
				zap.Error(safetyErr),
			)
		} else if !safetyResp.Safe {
			return nil, fmt.Errorf("KB-25 safety rule %s blocked activation of %s for patient %s: %s",
				safetyResp.RuleCode, protocolID, patientID, safetyResp.Reason)
		}
	}

	// M3-MAINTAIN requires M3-PRP + M3-VFRP to be GRADUATED (Spec Section 7)
	if protocolID == "M3-MAINTAIN" {
		var graduatedCount int64
		ps.db.DB.Model(&models.ProtocolState{}).
			Where("patient_id = ? AND protocol_id IN (?, ?) AND status = ?",
				patientID, "M3-PRP", "M3-VFRP", "GRADUATED").
			Count(&graduatedCount)
		if graduatedCount < 2 {
			return nil, fmt.Errorf("M3-MAINTAIN requires M3-PRP and M3-VFRP to be GRADUATED (found %d)", graduatedCount)
		}
	}

	// Resolve initial phase from template (fixes hardcoded BASELINE for M3-MAINTAIN which starts at CONSOLIDATION)
	tmpl, tmplErr := ps.registry.GetTemplate(protocolID)
	initialPhase := "BASELINE"
	if tmplErr == nil && len(tmpl.Phases) > 0 {
		initialPhase = tmpl.Phases[0].ID
	}

	state := models.ProtocolState{
		PatientID:      patientID,
		ProtocolID:     protocolID,
		CurrentPhase:   initialPhase,
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

	// G-8: Publish activation event after the record is persisted.
	ps.eventBus.Publish(models.EventProtocolActivated, patientID, map[string]interface{}{
		"protocol_id": protocolID,
		"phase":       initialPhase,
	})

	return &state, nil
}

// TransitionPhase advances a protocol to the next phase.
//
// The method evaluates the transition using the protocol-specific evaluator.
// If the evaluator returns ESCALATE or ABORT the event is published to the
// event bus (G-7) before returning an error to the caller so that downstream
// consumers (e.g. KB-23) can react regardless of whether the HTTP handler
// acts on the error.
//
// On a successful DB write the appropriate phase transition event is published
// (G-8): EventProtocolGraduated when the protocol reaches GRADUATED status,
// EventProtocolTransitioned for all other phases.
func (ps *ProtocolService) TransitionPhase(patientID string, protocolID string, nextPhase string) (*models.ProtocolState, error) {
	var state models.ProtocolState
	err := ps.db.DB.Where("patient_id = ? AND protocol_id = ? AND status = ?", patientID, protocolID, "ACTIVE").First(&state).Error
	if err != nil {
		return nil, fmt.Errorf("no active protocol %s for patient %s", protocolID, patientID)
	}

	if !state.CanTransition(nextPhase) {
		return nil, fmt.Errorf("invalid transition from %s to %s", state.CurrentPhase, nextPhase)
	}

	fromPhase := state.CurrentPhase

	state.CurrentPhase = nextPhase
	state.PhaseStartDate = time.Now().UTC()
	state.PhaseExtended = false

	if nextPhase == "GRADUATED" {
		state.Status = "GRADUATED"
	}

	if err := ps.db.DB.Save(&state).Error; err != nil {
		return nil, fmt.Errorf("failed to transition phase: %w", err)
	}

	// G-8: Publish the appropriate phase transition event.
	if state.Status == "GRADUATED" {
		ps.eventBus.Publish(models.EventProtocolGraduated, patientID, map[string]interface{}{
			"protocol_id": protocolID,
			"from_phase":  fromPhase,
			"to_phase":    nextPhase,
			"patient_id":  patientID,
		})
	} else {
		ps.eventBus.Publish(models.EventProtocolTransitioned, patientID, map[string]interface{}{
			"protocol_id": protocolID,
			"from_phase":  fromPhase,
			"to_phase":    nextPhase,
			"patient_id":  patientID,
		})
	}

	// G-6: Best-effort KB-25 projection after a successful phase transition.
	// Errors are logged but do not fail the transition — the mechanical write has
	// already been committed and the event published.
	if ps.kb25 != nil {
		projResp, projErr := ps.kb25.ProjectCombined(clients.ProjectionRequest{
			PatientID:   patientID,
			ProtocolIDs: []string{protocolID},
			HorizonDays: 90,
		})
		if projErr != nil {
			ps.logger.Warn("KB-25 projection unavailable after phase transition",
				zap.String("patient_id", patientID),
				zap.String("protocol_id", protocolID),
				zap.String("to_phase", nextPhase),
				zap.Error(projErr),
			)
		} else {
			ps.logger.Info("KB-25 projected outcomes for new phase",
				zap.String("patient_id", patientID),
				zap.String("protocol_id", protocolID),
				zap.String("to_phase", nextPhase),
				zap.Int("projections", len(projResp.Projections)),
				zap.Float64("synergy_multiplier", projResp.Synergy),
			)
		}
	}

	return &state, nil
}

// EvaluateAndTransition evaluates a PRP or VFRP transition and, if the evaluator
// signals ESCALATE or ABORT, publishes an escalation event (G-7) before returning
// the decision to the caller.
//
// This is the preferred entry point when the caller has a full TransitionEvaluation
// struct (e.g. from an API handler that receives lab context). TransitionPhase
// performs the mechanical DB write; this method performs the clinical guard.
func (ps *ProtocolService) EvaluateAndTransition(patientID string, eval TransitionEvaluation) (TransitionDecision, error) {
	var decision TransitionDecision
	switch eval.ProtocolID {
	case "M3-PRP":
		decision = EvaluatePRPTransition(eval)
	case "M3-VFRP":
		decision = EvaluateVFRPTransition(eval)
	case "GLYC-1":
		decision = EvaluateGLYC1Transition(eval)
	case "HTN-1":
		decision = EvaluateHTN1Transition(eval)
	case "RENAL-1":
		decision = EvaluateRENAL1Transition(eval)
	case "DEPRESC-1":
		decision = EvaluateDEPRESC1Transition(eval)
	case "M3-MAINTAIN":
		decision = EvaluateMAINTAINTransition(eval)
	case "M3-RECORRECTION":
		decision = EvaluateRECORRECTIONTransition(eval)
	case "LIPID-1":
		// LIPID-1 is card-only — no phase transitions
		return TransitionDecision{Action: "HOLD", Reason: "LIPID-1 is card-only, no phase transitions"}, nil
	default:
		return TransitionDecision{Action: "HOLD", Reason: "unknown protocol"}, nil
	}

	// G-7: Publish escalation event before returning so KB-23 receives it
	// regardless of how the caller handles the error.
	if decision.Action == "ESCALATE" || decision.Action == "ABORT" {
		ps.eventBus.Publish(models.EventProtocolEscalated, patientID, map[string]interface{}{
			"protocol_id":   eval.ProtocolID,
			"current_phase": eval.CurrentPhase,
			"action":        decision.Action,
			"reason":        decision.Reason,
		})
		return decision, fmt.Errorf("protocol %s: %s — %s", eval.ProtocolID, decision.Action, decision.Reason)
	}

	return decision, nil
}

// GetActiveProtocols returns all active protocols for a patient.
func (ps *ProtocolService) GetActiveProtocols(patientID string) ([]models.ProtocolState, error) {
	var states []models.ProtocolState
	err := ps.db.DB.Where("patient_id = ? AND status = ?", patientID, "ACTIVE").Find(&states).Error
	return states, err
}
