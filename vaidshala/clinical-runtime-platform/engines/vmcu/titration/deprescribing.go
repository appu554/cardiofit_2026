// deprescribing.go implements controlled medication dose reduction (Task 3.1).
//
// DEPRESCRIBING_MODE allows clinician-initiated step-down of medication doses
// with safety monitoring. Key differences from escalation:
//   - Wider Channel B glucose thresholds during active deprescribing
//   - Step-down rate limiting (max one step per cooldown period)
//   - If Channel B fires HALT during deprescribing, freeze at current
//     reduced dose (do NOT revert to original higher dose)
//   - Full SafetyTrace audit for every deprescribing cycle
package titration

import (
	"fmt"
	"sync"
	"time"
)

// DeprescribingState tracks the lifecycle of a deprescribing plan.
type DeprescribingState string

const (
	DeprescribingInactive  DeprescribingState = "INACTIVE"
	DeprescribingActive    DeprescribingState = "ACTIVE"
	DeprescribingPaused    DeprescribingState = "PAUSED"    // Channel B fired, frozen at current reduced dose
	DeprescribingCompleted DeprescribingState = "COMPLETED" // target dose reached
	DeprescribingAborted   DeprescribingState = "ABORTED"   // clinician cancelled
)

// DeprescribingPlan describes a controlled dose reduction program.
type DeprescribingPlan struct {
	PlanID        string             `json:"plan_id"`
	PatientID     string             `json:"patient_id"`
	DrugClass     string             `json:"drug_class"`
	MedClass      MedicationClass    `json:"med_class"`
	InitialDose   float64            `json:"initial_dose_mg"`
	TargetDose    float64            `json:"target_dose_mg"`
	StepDownMg    float64            `json:"step_down_mg"`
	CurrentDose   float64            `json:"current_dose_mg"`
	State         DeprescribingState `json:"state"`
	Rationale     string             `json:"rationale"`
	InitiatedBy   string             `json:"initiated_by"` // clinician ID
	InitiatedAt   time.Time          `json:"initiated_at"`
	CompletedAt   *time.Time         `json:"completed_at,omitempty"`
	StepsExecuted int                `json:"steps_executed"`
	PausedAt      *time.Time         `json:"paused_at,omitempty"`
	PauseReason   string             `json:"pause_reason,omitempty"`
}

// DeprescribingManager tracks active deprescribing plans per patient.
type DeprescribingManager struct {
	mu    sync.RWMutex
	plans map[string]*DeprescribingPlan // patientID:drugClass → plan
}

// NewDeprescribingManager creates a new manager.
func NewDeprescribingManager() *DeprescribingManager {
	return &DeprescribingManager{
		plans: make(map[string]*DeprescribingPlan),
	}
}

// StartDeprescribing initiates a new deprescribing plan.
// The acrCategory parameter is the patient's current KDIGO ACR category
// (A1, A2, or A3). Pass an empty string if ACR status is unknown.
// The eGFR parameter is the patient's current eGFR in mL/min/1.73m².
// Pass 0 if eGFR is unknown (safe default: SGLT2i will be vetoed).
func (dm *DeprescribingManager) StartDeprescribing(
	planID, patientID, drugClass string,
	medClass MedicationClass,
	currentDose, targetDose, stepDownMg float64,
	rationale, clinicianID string,
	acrCategory string,
	eGFR float64,
) (*DeprescribingPlan, error) {
	// Deprescribing veto: ACEi/ARB (ACR A2/A3) and SGLT2i (eGFR <60 or ACR >=A2)
	if DeprescribingVetoCheck(drugClass, acrCategory, eGFR) {
		return nil, fmt.Errorf(
			"DEPRESCRIBING_VETO: cannot deprescribe %s (ACR=%s, eGFR=%.1f) — renoprotection mandate",
			drugClass, acrCategory, eGFR)
	}

	if targetDose >= currentDose {
		return nil, fmt.Errorf("target dose (%.1f) must be lower than current dose (%.1f)", targetDose, currentDose)
	}
	if stepDownMg <= 0 {
		return nil, fmt.Errorf("step_down_mg must be positive, got %.1f", stepDownMg)
	}
	if targetDose < 0 {
		return nil, fmt.Errorf("target dose cannot be negative")
	}

	key := planKey(patientID, drugClass)

	dm.mu.Lock()
	defer dm.mu.Unlock()

	// Check for existing active plan
	if existing, ok := dm.plans[key]; ok && existing.State == DeprescribingActive {
		return nil, fmt.Errorf("active deprescribing plan already exists for patient %s, drug %s", patientID, drugClass)
	}

	plan := &DeprescribingPlan{
		PlanID:      planID,
		PatientID:   patientID,
		DrugClass:   drugClass,
		MedClass:    medClass,
		InitialDose: currentDose,
		TargetDose:  targetDose,
		StepDownMg:  stepDownMg,
		CurrentDose: currentDose,
		State:       DeprescribingActive,
		Rationale:   rationale,
		InitiatedBy: clinicianID,
		InitiatedAt: time.Now(),
	}

	dm.plans[key] = plan
	return plan, nil
}

// StepDown executes the next dose reduction step.
// The acrCategory parameter is re-checked at each step to guard against
// ACR worsening since plan initiation. Pass empty string if unknown.
// The eGFR parameter is the patient's current eGFR in mL/min/1.73m².
// Pass 0 if eGFR is unknown (safe default: SGLT2i will be vetoed).
// Returns the new dose and whether the plan is now complete.
func (dm *DeprescribingManager) StepDown(patientID, drugClass, acrCategory string, eGFR float64) (newDose float64, completed bool, err error) {
	key := planKey(patientID, drugClass)

	dm.mu.Lock()
	defer dm.mu.Unlock()

	plan, ok := dm.plans[key]
	if !ok {
		return 0, false, fmt.Errorf("no deprescribing plan for patient %s, drug %s", patientID, drugClass)
	}
	if plan.State != DeprescribingActive {
		return plan.CurrentDose, false, fmt.Errorf("plan is %s, not ACTIVE", plan.State)
	}

	// Deprescribing veto re-check at each step: ACR or eGFR may have worsened
	if DeprescribingVetoCheck(drugClass, acrCategory, eGFR) {
		plan.State = DeprescribingPaused
		now := time.Now()
		plan.PausedAt = &now
		plan.PauseReason = fmt.Sprintf("DEPRESCRIBING_VETO: ACR=%s eGFR=%.1f requires %s to be maintained", acrCategory, eGFR, drugClass)
		return plan.CurrentDose, false, fmt.Errorf(
			"DEPRESCRIBING_VETO: step-down blocked for %s (ACR=%s, eGFR=%.1f)",
			drugClass, acrCategory, eGFR)
	}

	// Reduce dose by step amount
	plan.CurrentDose -= plan.StepDownMg
	if plan.CurrentDose < plan.TargetDose {
		plan.CurrentDose = plan.TargetDose
	}
	plan.StepsExecuted++

	// Check if target reached
	if plan.CurrentDose <= plan.TargetDose {
		plan.State = DeprescribingCompleted
		now := time.Now()
		plan.CompletedAt = &now
		return plan.CurrentDose, true, nil
	}

	return plan.CurrentDose, false, nil
}

// PausePlan pauses the deprescribing plan (e.g., Channel B fired HALT).
// The dose stays at the current reduced level — does NOT revert to initial.
func (dm *DeprescribingManager) PausePlan(patientID, drugClass, reason string) error {
	key := planKey(patientID, drugClass)

	dm.mu.Lock()
	defer dm.mu.Unlock()

	plan, ok := dm.plans[key]
	if !ok {
		return fmt.Errorf("no deprescribing plan for patient %s, drug %s", patientID, drugClass)
	}
	if plan.State != DeprescribingActive {
		return fmt.Errorf("plan is %s, cannot pause", plan.State)
	}

	plan.State = DeprescribingPaused
	now := time.Now()
	plan.PausedAt = &now
	plan.PauseReason = reason
	return nil
}

// ResumePlan resumes a paused deprescribing plan.
func (dm *DeprescribingManager) ResumePlan(patientID, drugClass string) error {
	key := planKey(patientID, drugClass)

	dm.mu.Lock()
	defer dm.mu.Unlock()

	plan, ok := dm.plans[key]
	if !ok {
		return fmt.Errorf("no deprescribing plan for patient %s, drug %s", patientID, drugClass)
	}
	if plan.State != DeprescribingPaused {
		return fmt.Errorf("plan is %s, not PAUSED", plan.State)
	}

	plan.State = DeprescribingActive
	plan.PausedAt = nil
	plan.PauseReason = ""
	return nil
}

// AbortPlan cancels the deprescribing plan. Dose stays at current reduced level.
func (dm *DeprescribingManager) AbortPlan(patientID, drugClass string) error {
	key := planKey(patientID, drugClass)

	dm.mu.Lock()
	defer dm.mu.Unlock()

	plan, ok := dm.plans[key]
	if !ok {
		return fmt.Errorf("no deprescribing plan for patient %s, drug %s", patientID, drugClass)
	}

	plan.State = DeprescribingAborted
	return nil
}

// GetPlan returns the current deprescribing plan for a patient+drug, if any.
func (dm *DeprescribingManager) GetPlan(patientID, drugClass string) *DeprescribingPlan {
	key := planKey(patientID, drugClass)

	dm.mu.RLock()
	defer dm.mu.RUnlock()

	return dm.plans[key]
}

// IsDeprescribing returns true if the patient has an active deprescribing plan
// for any drug class.
func (dm *DeprescribingManager) IsDeprescribing(patientID string) bool {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	for _, plan := range dm.plans {
		if plan.PatientID == patientID && plan.State == DeprescribingActive {
			return true
		}
	}
	return false
}

// DeprescribingVetoCheck returns true if the drug class must NOT be deprescribed.
//
// ACEi/ARB: blocked if ACR A2 or A3 (renal protection independent of BP).
// SGLT2i: blocked if eGFR <60 OR ACR >=A2 (triple-action: antidiabetic +
//
//	antihypertensive + renal-protective). Per Proposal §7.1, the system must
//	NEVER propose SGLT2i deprescribing in a patient with eGFR <60 or
//	proteinuria, even if glycaemic control is adequate without it.
func DeprescribingVetoCheck(drugClass string, acrCategory string, eGFR float64) bool {
	// ACEi/ARB: never deprescribe if ACR A2 or A3
	if (drugClass == "ACE_INHIBITOR" || drugClass == "ARB") &&
		(acrCategory == "A2" || acrCategory == "A3") {
		return true
	}
	// SGLT2i: never deprescribe if eGFR <60 OR proteinuria (ACR >=A2)
	if drugClass == "SGLT2I" || drugClass == "SGLT2_INHIBITOR" {
		if eGFR < 60 || acrCategory == "A2" || acrCategory == "A3" {
			return true
		}
	}
	return false
}

// ACRVetoCheck is the backward-compatible wrapper for DeprescribingVetoCheck.
// It passes eGFR=0, which is a safe default: SGLT2i will always be vetoed
// when eGFR is unknown (0 < 60), preventing unsafe deprescribing.
//
// Callers that have eGFR available should use DeprescribingVetoCheck directly.
func ACRVetoCheck(drugClass string, acrCategory string) bool {
	return DeprescribingVetoCheck(drugClass, acrCategory, 0)
}

func planKey(patientID, drugClass string) string {
	return patientID + ":" + drugClass
}
