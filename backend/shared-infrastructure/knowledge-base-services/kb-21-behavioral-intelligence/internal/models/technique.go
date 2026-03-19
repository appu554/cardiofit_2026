package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TechniqueID identifies one of the 12 coaching techniques.
type TechniqueID string

const (
	TechMicroCommitment        TechniqueID = "T-01" // Small, achievable daily goals
	TechHabitStacking          TechniqueID = "T-02" // Attach new behavior to existing habit
	TechLossAversion           TechniqueID = "T-03" // Frame as avoiding loss of progress
	TechSocialNorms            TechniqueID = "T-04" // Peer/district comparison
	TechMicroEducation         TechniqueID = "T-05" // Brief educational content
	TechProgressVisualization  TechniqueID = "T-06" // Show data-driven progress
	TechEnvironmentRestructure TechniqueID = "T-07" // Modify physical environment cues
	TechImplementIntention     TechniqueID = "T-08" // If-then planning for disruptions
	TechCostAwareSubstitution  TechniqueID = "T-09" // Affordable alternative suggestions
	TechFamilyInclusion        TechniqueID = "T-10" // Involve family in coaching
	TechRecoveryProtocol       TechniqueID = "T-11" // Post-disruption re-engagement
	TechKinshipTone            TechniqueID = "T-12" // Culturally warm, elder-respectful tone
)

// AllTechniques returns the canonical list of 12 techniques.
func AllTechniques() []TechniqueID {
	return []TechniqueID{
		TechMicroCommitment, TechHabitStacking, TechLossAversion,
		TechSocialNorms, TechMicroEducation, TechProgressVisualization,
		TechEnvironmentRestructure, TechImplementIntention, TechCostAwareSubstitution,
		TechFamilyInclusion, TechRecoveryProtocol, TechKinshipTone,
	}
}

// MotivationPhase represents where the patient is in the correction cycle (E5).
type MotivationPhase string

const (
	PhaseInitiation    MotivationPhase = "INITIATION"    // Days 1-14
	PhaseExploration   MotivationPhase = "EXPLORATION"   // Days 15-35
	PhaseConsolidation MotivationPhase = "CONSOLIDATION" // Days 36-60
	PhaseMastery       MotivationPhase = "MASTERY"       // Days 61-84
	PhaseRecovery      MotivationPhase = "RECOVERY"      // After any disruption
)

// TechniqueEffectiveness tracks Bayesian posterior per patient per technique.
// Alpha and Beta are the Beta distribution parameters.
// PosteriorMean = Alpha / (Alpha + Beta).
type TechniqueEffectiveness struct {
	ID        uuid.UUID   `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PatientID string      `gorm:"uniqueIndex:idx_tech_patient_tech;not null" json:"patient_id"`
	Technique TechniqueID `gorm:"type:varchar(10);uniqueIndex:idx_tech_patient_tech;not null" json:"technique"`

	// Beta distribution parameters (Thompson Sampling)
	Alpha float64 `gorm:"type:decimal(8,4);not null;default:1.0" json:"alpha"`
	Beta  float64 `gorm:"type:decimal(8,4);not null;default:1.0" json:"beta"`

	// Derived metrics (updated on each posterior update)
	PosteriorMean float64    `gorm:"type:decimal(5,4);not null;default:0.5" json:"posterior_mean"`
	Deliveries    int        `gorm:"default:0" json:"deliveries"`
	Successes     int        `gorm:"default:0" json:"successes"` // adherence improved within 7d observation window
	LastDelivered *time.Time `json:"last_delivered,omitempty"`

	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (t *TechniqueEffectiveness) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}

// PatientMotivationPhase tracks the patient's current motivation phase (E5).
type PatientMotivationPhase struct {
	PatientID      string          `gorm:"primaryKey" json:"patient_id"`
	Phase          MotivationPhase `gorm:"type:varchar(20);not null;default:'INITIATION'" json:"phase"`
	PhaseStartedAt time.Time       `gorm:"not null" json:"phase_started_at"`
	CycleDayStart  int             `gorm:"not null;default:1" json:"cycle_day_start"` // day number when this phase began
	CycleDay       int             `gorm:"not null;default:1" json:"cycle_day"`       // current day in correction cycle

	// Phase transition tracking
	PreviousPhase  MotivationPhase `gorm:"type:varchar(20)" json:"previous_phase,omitempty"`
	TransitionedAt *time.Time      `json:"transitioned_at,omitempty"`

	// Recovery tracking
	PreRecoveryPhase MotivationPhase `gorm:"type:varchar(20)" json:"pre_recovery_phase,omitempty"` // phase before disruption
	RecoveryCount    int             `gorm:"default:0" json:"recovery_count"`

	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// IntakeProfile captures cold-start signals from Tier-1 onboarding.
// Used by BCE v2.0 E1 (Cold-Start Profiling) to assign behavioral phenotype priors.
// Stored here as the foundation; cold-start assignment logic is E1 scope.
type IntakeProfile struct {
	PatientID string `gorm:"primaryKey" json:"patient_id"`

	// Cold-start signals (spec Table 3)
	AgeBand              string  `gorm:"type:varchar(10)" json:"age_band"`                       // "30-45", "45-60", "60+"
	EducationLevel       string  `gorm:"type:varchar(20)" json:"education_level"`                // "LOW", "MODERATE", "HIGH"
	SmartphoneLiteracy   string  `gorm:"type:varchar(20)" json:"smartphone_literacy"`            // "LOW", "MODERATE", "HIGH"
	SelfEfficacy         float64 `gorm:"type:decimal(3,2);default:0.5" json:"self_efficacy"`     // 0.0-1.0 from intake question
	FamilyStructure      string  `gorm:"type:varchar(20)" json:"family_structure"`               // "JOINT", "NUCLEAR", "ALONE"
	EmploymentStatus     string  `gorm:"type:varchar(20)" json:"employment_status"`              // "WORKING", "RETIRED", "HOME"
	PriorProgramSuccess  *bool   `gorm:"type:boolean" json:"prior_program_success"`              // nil=never tried, true=succeeded, false=failed
	FirstResponseLatency int64   `gorm:"default:0" json:"first_response_latency_ms"`            // from first check-in

	CollectedAt time.Time `gorm:"not null" json:"collected_at"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
}
