package models

import (
	"time"

	"github.com/google/uuid"
)

// TransitionState tracks the lifecycle of a care transition.
type TransitionState string

const (
	TransitionActive                TransitionState = "ACTIVE"
	TransitionCompletedSuccessful   TransitionState = "COMPLETED_SUCCESSFUL"
	TransitionCompletedReadmitted   TransitionState = "COMPLETED_READMITTED"
	TransitionCompletedDeteriorated TransitionState = "COMPLETED_DETERIORATED"
	TransitionCompletedDisengaged   TransitionState = "COMPLETED_DISENGAGED"
)

// ReconciliationStatus tracks medication reconciliation progress.
type ReconciliationStatus string

const (
	ReconciliationPending       ReconciliationStatus = "PENDING"
	ReconciliationReconciled    ReconciliationStatus = "RECONCILED"
	ReconciliationDiscrepancies ReconciliationStatus = "DISCREPANCIES_FOUND"
)

// CareTransition tracks a single post-discharge transition episode.
type CareTransition struct {
	ID                           uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PatientID                    string     `gorm:"size:100;index;not null" json:"patient_id"`
	DischargeDate                time.Time  `gorm:"not null" json:"discharge_date"`
	DetectedAt                   time.Time  `gorm:"not null" json:"detected_at"`
	DischargeSource              string     `gorm:"size:30;not null" json:"discharge_source"`
	FacilityName                 string     `gorm:"size:200" json:"facility_name"`
	FacilityType                 string     `gorm:"size:30" json:"facility_type"`
	PrimaryDiagnosis             string     `gorm:"size:200" json:"primary_diagnosis,omitempty"`
	LengthOfStayDays             int        `json:"length_of_stay_days"`
	DischargeDisposition         string     `gorm:"size:30" json:"discharge_disposition"`
	TransitionState              string     `gorm:"size:30;not null;default:'ACTIVE'" json:"transition_state"`
	HeightenedSurveillanceActive bool       `gorm:"default:true" json:"heightened_surveillance_active"`
	ReconciliationStatus         string     `gorm:"size:30;default:'PENDING'" json:"reconciliation_status"`
	TransitionEndDate            *time.Time `json:"transition_end_date,omitempty"`
	SourceConfidence             string     `gorm:"size:10;default:'HIGH'" json:"source_confidence"`
	WindowDays                   int        `gorm:"default:30" json:"window_days"`
	CreatedAt                    time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt                    time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

func (CareTransition) TableName() string { return "care_transitions" }

// DischargeMedication represents one medication on the discharge regimen.
type DischargeMedication struct {
	ID                   uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TransitionID         uuid.UUID `gorm:"type:uuid;index;not null" json:"transition_id"`
	DrugName             string    `gorm:"size:100;not null" json:"drug_name"`
	DrugClass            string    `gorm:"size:50" json:"drug_class"`
	DoseMg               float64   `json:"dose_mg"`
	Frequency            string    `gorm:"size:30" json:"frequency"`
	ReconciliationStatus string    `gorm:"size:20;not null" json:"reconciliation_status"`
	PreAdmissionDrugName string    `gorm:"size:100" json:"pre_admission_drug_name,omitempty"`
	ChangeReason         string    `gorm:"size:200" json:"change_reason,omitempty"`
	ClinicalRiskLevel    string    `gorm:"size:10" json:"clinical_risk_level"`
	FormularyStatus      string    `gorm:"size:30" json:"formulary_status,omitempty"`
	SupplyGapRisk        string    `gorm:"size:10" json:"supply_gap_risk,omitempty"`
	CreatedAt            time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (DischargeMedication) TableName() string { return "discharge_medications" }

// TransitionMilestone is a scheduled review point in the transition.
type TransitionMilestone struct {
	ID               uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TransitionID     uuid.UUID  `gorm:"type:uuid;index;not null" json:"transition_id"`
	MilestoneType    string     `gorm:"size:40;not null" json:"milestone_type"`
	ScheduledFor     time.Time  `gorm:"not null" json:"scheduled_for"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
	CompletionStatus string     `gorm:"size:30;default:'SCHEDULED'" json:"completion_status"`
	CardsGenerated   int        `gorm:"default:0" json:"cards_generated"`
	Notes            string     `gorm:"type:text" json:"notes,omitempty"`
	CreatedAt        time.Time  `gorm:"autoCreateTime" json:"created_at"`
}

func (TransitionMilestone) TableName() string { return "transition_milestones" }

// TransitionOutcome is the final assessment at transition exit.
type TransitionOutcome struct {
	ID                             uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TransitionID                   uuid.UUID  `gorm:"type:uuid;uniqueIndex;not null" json:"transition_id"`
	OutcomeCategory                string     `gorm:"size:30;not null" json:"outcome_category"`
	ReadmissionDate                *time.Time `json:"readmission_date,omitempty"`
	ReadmissionReason              string     `gorm:"size:200" json:"readmission_reason,omitempty"`
	FinalPAITier                   string     `gorm:"size:10" json:"final_pai_tier,omitempty"`
	MedicationReconciliationOutcome string    `gorm:"size:30" json:"medication_reconciliation_outcome,omitempty"`
	EngagementMetric               float64    `json:"engagement_metric"`
	EscalationsTriggeredCount      int        `json:"escalations_triggered_count"`
	ComputedAt                     time.Time  `gorm:"not null" json:"computed_at"`
	CreatedAt                      time.Time  `gorm:"autoCreateTime" json:"created_at"`
}

func (TransitionOutcome) TableName() string { return "transition_outcomes" }

// MedicationReconciliationReport is the output of comparing pre-admission vs discharge.
type MedicationReconciliationReport struct {
	TransitionID         uuid.UUID             `json:"transition_id"`
	NewMedications       []DischargeMedication `json:"new_medications"`
	StoppedMedications   []DischargeMedication `json:"stopped_medications"`
	ChangedMedications   []DischargeMedication `json:"changed_medications"`
	ContinuedMedications []DischargeMedication `json:"continued_medications"`
	UnclearMedications   []DischargeMedication `json:"unclear_medications"`
	DiscrepanciesFound   int                   `json:"discrepancies_found"`
	HighRiskChanges      int                   `json:"high_risk_changes"`
	FormularyIssues      int                   `json:"formulary_issues"`
	ReconciliationOutcome string               `json:"reconciliation_outcome"`
}

// Milestone type constants
const (
	MilestoneMedReconciliation48H = "MEDICATION_RECONCILIATION_48H"
	MilestoneFirstFollowup7D      = "FIRST_FOLLOWUP_7D"
	MilestoneMidpointReview14D    = "MIDPOINT_REVIEW_14D"
	MilestoneExitAssessment30D    = "EXIT_ASSESSMENT_30D"
	MilestoneEngagementCheck72H   = "ENGAGEMENT_CHECK_72H"
	MilestoneMedSupplyCheck       = "MEDICATION_SUPPLY_CHECK"
)

// Discharge source constants
const (
	SourceFHIREncounter   = "FHIR_ENCOUNTER"
	SourceManual          = "MANUAL"
	SourcePatientReported = "PATIENT_REPORTED"
	SourceASHAReported    = "ASHA_REPORTED"
)

// Medication reconciliation status for individual drugs
const (
	MedStatusNew         = "NEW"
	MedStatusContinued   = "CONTINUED"
	MedStatusChangedDose = "CHANGED_DOSE"
	MedStatusStopped     = "STOPPED"
	MedStatusUnclear     = "UNCLEAR"
)
