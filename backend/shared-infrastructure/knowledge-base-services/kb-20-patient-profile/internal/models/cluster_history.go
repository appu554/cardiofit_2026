package models

import (
	"time"

	"github.com/google/uuid"
)

// ClusterAssignmentRecord captures the raw HDBSCAN output for one patient in one batch run,
// plus the stability-processed assignment that was actually applied.
type ClusterAssignmentRecord struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PatientID string    `gorm:"size:100;index;not null"                        json:"patient_id"`
	RunID     string    `gorm:"size:100;index;not null"                        json:"run_id"`
	RunDate   time.Time `gorm:"not null"                                       json:"run_date"`

	// Raw HDBSCAN output
	RawClusterLabel   string  `gorm:"size:30;not null"         json:"raw_cluster_label"`
	MembershipProb    float64 `gorm:"type:decimal(5,4)"        json:"membership_prob"`
	DistToCentroid    float64 `gorm:"type:decimal(10,4)"       json:"dist_to_centroid"`
	DistToNearest     float64 `gorm:"type:decimal(10,4)"       json:"dist_to_nearest"`
	SeparabilityRatio float64 `gorm:"type:decimal(8,4)"        json:"separability_ratio"` // dist_nearest / dist_own
	IsNoise           bool    `gorm:"not null;default:false"   json:"is_noise"`

	// Stability-processed outcome
	StableCluster  string `gorm:"size:30"          json:"stable_cluster"`
	WasOverridden  bool   `gorm:"not null;default:false" json:"was_overridden"`
	OverrideReason string `gorm:"size:200"         json:"override_reason,omitempty"`

	CreatedAt time.Time `gorm:"not null;autoCreateTime" json:"created_at"`
}

// TableName implements schema.Tabler for GORM.
func (ClusterAssignmentRecord) TableName() string {
	return "cluster_assignment_records"
}

// ClusterTransitionRecord is written whenever the patient's stable cluster label changes.
// It captures both the mechanical transition type and the clinical classification.
type ClusterTransitionRecord struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PatientID      string    `gorm:"size:100;index;not null"                        json:"patient_id"`
	TransitionDate time.Time `gorm:"not null"                                       json:"transition_date"`

	PreviousCluster string `gorm:"size:30;not null" json:"previous_cluster"`
	NewCluster      string `gorm:"size:30;not null" json:"new_cluster"`

	// TransitionType: GENUINE | FLAP_DAMPENED | OVERRIDE | INITIAL
	TransitionType string `gorm:"size:30;not null" json:"transition_type"`

	// Classification: GENUINE_TRANSITION | PROBABLE_FLAP | UNCERTAIN | DWELL_HELD
	Classification string `gorm:"size:30;not null" json:"classification"`

	DwellDaysInPrevious int     `gorm:"not null;default:0"    json:"dwell_days_in_previous"`
	ConfidenceInNew     float64 `gorm:"type:decimal(5,4)"     json:"confidence_in_new"`

	// Cross-domain context at the time of transition
	TriggerEvent                 string `gorm:"size:100"        json:"trigger_event,omitempty"`
	DominantDomainDriver         string `gorm:"size:100"        json:"dominant_domain_driver,omitempty"`
	MHRICategoryAtTransition     string `gorm:"size:50"         json:"mhri_category_at_transition,omitempty"`
	EngagementStatusAtTransition string `gorm:"size:50"         json:"engagement_status_at_transition,omitempty"`
	DataModalityChange           bool   `gorm:"not null;default:false" json:"data_modality_change"`

	CreatedAt time.Time `gorm:"not null;autoCreateTime" json:"created_at"`
}

// TableName implements schema.Tabler for GORM.
func (ClusterTransitionRecord) TableName() string {
	return "cluster_transition_records"
}

// PatientClusterState is the in-memory stability state for a patient.
// It is NOT persisted directly; it is reconstructed from ClusterAssignmentRecords
// and ClusterTransitionRecords by the stability engine.
type PatientClusterState struct {
	PatientID            string
	CurrentStableCluster string
	StableSince          time.Time
	DwellDays            int
	Confidence           float64

	// Pending: a raw assignment that has not yet cleared the dwell gate
	PendingRawCluster *string
	PendingSince      *time.Time

	// Flap tracking
	FlapCount  int
	IsFlapping bool
	FlapPair   []string // the two cluster labels oscillating

	// Most recent transitions for context
	TransitionHistory []ClusterTransitionRecord
}

// StabilityDecision is the output of one stability-engine evaluation cycle for a patient.
type StabilityDecision struct {
	PatientID         string
	RawClusterLabel   string
	StableClusterLabel string // may equal RawClusterLabel or may be the held label

	// Decision: ACCEPT | HOLD_DWELL | HOLD_FLAP | OVERRIDE_EVENT
	Decision   string
	Reason     string
	Confidence float64

	TransitionType string `json:",omitempty"`
	TriggerEvent   string `json:",omitempty"`
	DomainDriver   string `json:",omitempty"`
}

// OverrideEvent is a clinical event that causes the stability engine to bypass the
// standard dwell gate and accept a cluster transition immediately.
type OverrideEvent struct {
	EventType  string
	EventDate  time.Time
	Domain     string `json:",omitempty"`
	Detail     string `json:",omitempty"`
}
