package models

import (
	"github.com/google/uuid"
)

// ReasoningStep records a single question that contributed meaningful
// information gain (|IG| > 0.01) during the Bayesian update loop.
// The ordered array of these steps forms the reasoning chain emitted
// on session completion for CTL Panel 4 transparency.
type ReasoningStep struct {
	StepNumber      int     `json:"step_number"`
	QuestionID      string  `json:"question_id"`
	QuestionText    string  `json:"question_text"`
	Answer          string  `json:"answer"`
	InformationGain float64 `json:"information_gain"`
	TopDifferential string  `json:"top_differential"`
	TopPosterior    float64 `json:"top_posterior"`
}

// OtherBucketDiffID is the reserved differential ID for the G15 implicit
// 'Other' bucket. Must not collide with any authored differential ID.
const OtherBucketDiffID = "_OTHER"

// G15 thresholds for the 'Other' bucket differential.
const (
	// OtherIncompleteThreshold triggers DIFFERENTIAL_INCOMPLETE flag.
	OtherIncompleteThreshold = 0.30
	// OtherEscalationThreshold triggers soft escalation to senior clinician.
	OtherEscalationThreshold = 0.45
)

// DifferentialEntry represents a single diagnosis in the ranked differential.
type DifferentialEntry struct {
	DifferentialID       string   `json:"differential_id"`
	Label                string   `json:"label"`
	PosteriorProbability float64  `json:"posterior_probability"`
	LogOdds              float64  `json:"log_odds"`
	IsOtherBucket        bool     `json:"is_other_bucket,omitempty"`
	Flags                []string `json:"flags,omitempty"`
}

// DifferentialSnapshot is the permanent record saved on session completion.
// Consumed by KB-23 (Decision Cards) and CalibrationManager.
type DifferentialSnapshot struct {
	SnapshotID uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"snapshot_id"`
	SessionID  uuid.UUID `gorm:"type:uuid;uniqueIndex;not null" json:"session_id"`

	// Ranked differentials sorted by posterior probability descending
	RankedDifferentials JSONB `gorm:"type:jsonb;not null" json:"ranked_differentials"`

	// All SafetyFlag records fired during session
	SafetyFlags JSONB `gorm:"type:jsonb;default:'[]'" json:"safety_flags"`

	// Convenience fields
	TopDiagnosis  string  `gorm:"type:varchar(128);not null" json:"top_diagnosis"`
	TopPosterior  float64 `gorm:"type:float8;not null" json:"top_posterior"`

	ConvergenceReached      bool `gorm:"type:bool;default:false" json:"convergence_reached"`
	QuestionsToConvergence  *int `gorm:"type:int" json:"questions_to_convergence,omitempty"`

	// N-01: KB-3 guideline references for DISHA audit trail
	GuidelinePriorRefs StringArray `gorm:"type:text[]" json:"guideline_prior_refs,omitempty"`

	// CTL Panel 4: Reasoning chain from Bayesian update loop
	ReasoningChain JSONB `gorm:"type:jsonb" json:"reasoning_chain,omitempty"`

	// Calibration fields (set by POST /calibration/feedback)
	ClinicianAdjudication *string `gorm:"type:varchar(128)" json:"clinician_adjudication,omitempty"`
	Concordant            *bool   `gorm:"type:bool" json:"concordant,omitempty"`
}

func (DifferentialSnapshot) TableName() string { return "differential_snapshots" }
