package models

// ConfidenceTier represents the diagnostic confidence level.
type ConfidenceTier string

const (
	TierFirm      ConfidenceTier = "FIRM"
	TierProbable  ConfidenceTier = "PROBABLE"
	TierPossible  ConfidenceTier = "POSSIBLE"
	TierUncertain ConfidenceTier = "UNCERTAIN"
)

// MCUGate is a 4-state gate controlling V-MCU insulin titration.
type MCUGate string

const (
	GateSafe   MCUGate = "SAFE"
	GateModify MCUGate = "MODIFY"
	GatePause  MCUGate = "PAUSE"
	GateHalt   MCUGate = "HALT"
)

// Level returns numeric severity (for most-restrictive comparison).
func (g MCUGate) Level() int {
	switch g {
	case GateSafe:
		return 0
	case GateModify:
		return 1
	case GatePause:
		return 2
	case GateHalt:
		return 3
	default:
		return 0
	}
}

// MostRestrictive returns the more restrictive of two gates.
func MostRestrictive(a, b MCUGate) MCUGate {
	if a.Level() >= b.Level() {
		return a
	}
	return b
}

// RecommendationType enumerates the 9 recommendation types.
type RecommendationType string

const (
	RecInvestigation       RecommendationType = "INVESTIGATION"
	RecReferral            RecommendationType = "REFERRAL"
	RecMonitoring          RecommendationType = "MONITORING"
	RecMedicationHold      RecommendationType = "MEDICATION_HOLD"
	RecMedicationModify    RecommendationType = "MEDICATION_MODIFY"
	RecMedicationContinue  RecommendationType = "MEDICATION_CONTINUE"
	RecLifestyle           RecommendationType = "LIFESTYLE"
	RecSafetyInstruction   RecommendationType = "SAFETY_INSTRUCTION"
	RecMedicationReview    RecommendationType = "MEDICATION_REVIEW"
)

// Urgency represents the urgency level of a recommendation.
type Urgency string

const (
	UrgencyImmediate Urgency = "IMMEDIATE"
	UrgencyUrgent    Urgency = "URGENT"
	UrgencyRoutine   Urgency = "ROUTINE"
	UrgencyScheduled Urgency = "SCHEDULED"
)

// CardSource identifies the origin of a decision card.
type CardSource string

const (
	SourceKB22Session       CardSource = "KB22_SESSION"
	SourceHypoglycaemiaFast CardSource = "HYPOGLYCAEMIA_FAST_PATH"
	SourcePerturbationDecay CardSource = "PERTURBATION_DECAY"
	SourceBehavioralGap     CardSource = "BEHAVIORAL_GAP"
	SourceClinicalSignal    CardSource = "CLINICAL_SIGNAL"
)

// CardStatus represents the lifecycle status of a decision card.
type CardStatus string

const (
	StatusActive               CardStatus = "ACTIVE"
	StatusSuperseded           CardStatus = "SUPERSEDED"
	StatusPendingReaffirmation CardStatus = "PENDING_REAFFIRMATION"
	StatusArchived             CardStatus = "ARCHIVED"
)

// SafetyTier categorises the safety urgency of a card.
type SafetyTier string

const (
	SafetyImmediate SafetyTier = "IMMEDIATE"
	SafetyUrgent    SafetyTier = "URGENT"
	SafetyRoutine   SafetyTier = "ROUTINE"
)

// ObservationReliability (A-05) indicates data quality of input observations.
type ObservationReliability string

const (
	ReliabilityHigh     ObservationReliability = "HIGH"
	ReliabilityModerate ObservationReliability = "MODERATE"
	ReliabilityLow      ObservationReliability = "LOW"
)

// HypoglycaemiaSeverity classifies hypoglycaemia events.
type HypoglycaemiaSeverity string

const (
	HypoMild     HypoglycaemiaSeverity = "MILD"
	HypoModerate HypoglycaemiaSeverity = "MODERATE"
	HypoSevere   HypoglycaemiaSeverity = "SEVERE"
)

// HypoglycaemiaSource identifies the detection source.
type HypoglycaemiaSource string

const (
	HypoSourceCGM            HypoglycaemiaSource = "CGM"
	HypoSourceGlucometer     HypoglycaemiaSource = "GLUCOMETER"
	HypoSourceVMCUDetected   HypoglycaemiaSource = "VMCU_DETECTED"
	HypoSourceVMCUPredicted  HypoglycaemiaSource = "VMCU_PREDICTED"
	HypoSourcePatientReport  HypoglycaemiaSource = "PATIENT_REPORT"
	HypoSourceKB21Behavioral HypoglycaemiaSource = "KB21_BEHAVIORAL"
)

// DeprescribingPhase tracks the state machine phase for antihypertensive
// dose-halving cards (AD-04).
type DeprescribingPhase string

const (
	DeprescribingDoseReduction DeprescribingPhase = "DOSE_REDUCTION"
	DeprescribingMonitoring    DeprescribingPhase = "MONITORING"
	DeprescribingRemoval       DeprescribingPhase = "REMOVAL"
	DeprescribingFailed        DeprescribingPhase = "FAILED"
)

// DeprescribingCardSource identifies cards produced by the deprescribing
// state machine.
const SourceDeprescribing CardSource = "DEPRESCRIBING"

// HaltSource (B-05) indicates whether a halt was measured or predicted.
type HaltSource string

const (
	HaltMeasured  HaltSource = "MEASURED"
	HaltPredicted HaltSource = "PREDICTED"
)

// InterventionType (A-01) classifies treatment interventions.
type InterventionType string

const (
	IntInsulinIncrease InterventionType = "INSULIN_INCREASE"
	IntInsulinDecrease InterventionType = "INSULIN_DECREASE"
	IntDrugHold        InterventionType = "DRUG_HOLD"
	IntDrugStart       InterventionType = "DRUG_START"
	IntDoseAdjust      InterventionType = "DOSE_ADJUST"

	// HTN co-management intervention types (Wave 2)
	IntDrugStop        InterventionType = "DRUG_STOP"
	IntDrugIncrease    InterventionType = "DRUG_INCREASE"
)

// FragmentType classifies summary text fragments by audience.
type FragmentType string

const (
	FragClinician         FragmentType = "CLINICIAN"
	FragPatient           FragmentType = "PATIENT"
	FragSafetyInstruction FragmentType = "SAFETY_INSTRUCTION"
)

// ConditionStatus indicates whether guideline criteria were met for a recommendation.
type ConditionStatus string

const (
	ConditionMet     ConditionStatus = "CRITERIA_MET"
	ConditionPartial ConditionStatus = "CRITERIA_PARTIAL"
	ConditionNotMet  ConditionStatus = "CRITERIA_NOT_MET"
)

// EventType identifies the type of event published to KB-19.
type EventType string

const (
	EventMCUGateChanged           EventType = "MCU_GATE_CHANGED"
	EventSafetyAlert              EventType = "SAFETY_ALERT"
	EventUnacknowledgedUrgentCard EventType = "UNACKNOWLEDGED_URGENT_CARD"
	EventMCUGateReaffirmation     EventType = "MCU_GATE_REAFFIRMATION_NEEDED"
	EventSLABreach                EventType = "SLA_BREACH"
)
