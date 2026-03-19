package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// --- Gamification (BCE v2.0 E2) ---

// PatientStreak tracks consecutive days of target behavior completion.
// Streaks activate only for patients where T-06 posterior > 0.15 or phenotype = REWARD_RESPONSIVE.
type PatientStreak struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PatientID string    `gorm:"uniqueIndex:idx_streak_patient_behavior;not null" json:"patient_id"`
	Behavior  string    `gorm:"type:varchar(50);uniqueIndex:idx_streak_patient_behavior;not null" json:"behavior"` // "WALK_AFTER_LUNCH", "MEDICATION_TAKEN", "PROTEIN_TARGET"

	CurrentStreak int       `gorm:"default:0" json:"current_streak"`
	LongestStreak int       `gorm:"default:0" json:"longest_streak"`
	LastActiveDay time.Time `gorm:"type:date" json:"last_active_day"`

	// Pause tracking (illness-streak pause per spec §4.1 governance rule)
	Paused       bool       `gorm:"default:false" json:"paused"`
	PausedAt     *time.Time `json:"paused_at,omitempty"`
	PauseReason  string     `gorm:"type:varchar(30)" json:"pause_reason,omitempty"` // "ILLNESS", "TRAVEL", "FESTIVAL"

	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (s *PatientStreak) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

// PatientMilestone records significant achievements.
type PatientMilestone struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PatientID string    `gorm:"index;not null" json:"patient_id"`

	MilestoneType string `gorm:"type:varchar(50);not null" json:"milestone_type"` // "FIRST_WEEK", "PROTEIN_2W", "STEPS_5K", "WAIST_IMPROVEMENT"
	Title         string `gorm:"type:varchar(200);not null" json:"title"`
	Description   string `gorm:"type:text" json:"description"`

	AchievedAt time.Time `gorm:"not null" json:"achieved_at"`
	Celebrated bool      `gorm:"default:false" json:"celebrated"` // whether a celebration nudge was sent

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (m *PatientMilestone) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}

// WeeklyChallenge represents a 7-day challenge with a specific achievable target.
type WeeklyChallenge struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PatientID string    `gorm:"index;not null" json:"patient_id"`

	ChallengeName string `gorm:"type:varchar(200);not null" json:"challenge_name"`
	TargetDays    int    `gorm:"not null;default:5" json:"target_days"` // e.g., "5 out of 7"
	ActualDays    int    `gorm:"default:0" json:"actual_days"`
	Completed     bool   `gorm:"default:false" json:"completed"`

	WeekStart time.Time `gorm:"type:date;not null" json:"week_start"`
	WeekEnd   time.Time `gorm:"type:date;not null" json:"week_end"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (ch *WeeklyChallenge) BeforeCreate(tx *gorm.DB) error {
	if ch.ID == uuid.Nil {
		ch.ID = uuid.New()
	}
	return nil
}

// --- Population Learning (BCE v2.0 E3) ---

// PopulationPrior stores aggregate-derived Alpha/Beta priors per phenotype per technique.
type PopulationPrior struct {
	ID        uuid.UUID          `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Phenotype ColdStartPhenotype `gorm:"type:varchar(30);uniqueIndex:idx_pop_prior_pheno_tech;not null" json:"phenotype"`
	Technique TechniqueID        `gorm:"type:varchar(10);uniqueIndex:idx_pop_prior_pheno_tech;not null" json:"technique"`

	Alpha      float64 `gorm:"type:decimal(8,4);not null" json:"alpha"`
	Beta       float64 `gorm:"type:decimal(8,4);not null" json:"beta"`
	SampleSize int     `gorm:"default:0" json:"sample_size"` // number of patients contributing

	Version   int       `gorm:"default:1" json:"version"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (p *PopulationPrior) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

// PriorCalibrationLog records each population learning cycle for audit.
type PriorCalibrationLog struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	RunAt     time.Time `gorm:"not null" json:"run_at"`

	TotalPatients       int     `gorm:"default:0" json:"total_patients"`
	EligiblePatients    int     `gorm:"default:0" json:"eligible_patients"` // patients with >= 8 deliveries
	AccuracyImprovement float64 `gorm:"type:decimal(5,4)" json:"accuracy_improvement"`
	Adopted             bool    `gorm:"default:false" json:"adopted"` // whether priors were updated

	Details   string    `gorm:"type:text" json:"details,omitempty"` // JSON details
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (l *PriorCalibrationLog) BeforeCreate(tx *gorm.DB) error {
	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}
	return nil
}

// --- Timing Optimization (BCE v2.0 E4) ---

// TimingSlot represents a candidate delivery time-of-day.
type TimingSlot string

const (
	Slot7AM  TimingSlot = "07:00"
	Slot9AM  TimingSlot = "09:00"
	Slot12PM TimingSlot = "12:00"
	Slot1PM  TimingSlot = "13:00"
	Slot5PM  TimingSlot = "17:00"
	Slot7PM  TimingSlot = "19:00"
	Slot9PM  TimingSlot = "21:00"
)

// AllTimingSlots returns the 7 candidate delivery slots.
func AllTimingSlots() []TimingSlot {
	return []TimingSlot{Slot7AM, Slot9AM, Slot12PM, Slot1PM, Slot5PM, Slot7PM, Slot9PM}
}

// PatientTimingProfile tracks per-patient response rates for each delivery time slot.
// Uses Thompson Sampling: Beta(α, β) per arm (slot).
type PatientTimingProfile struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PatientID string     `gorm:"uniqueIndex:idx_timing_patient_slot;not null" json:"patient_id"`
	Slot      TimingSlot `gorm:"type:varchar(10);uniqueIndex:idx_timing_patient_slot;not null" json:"slot"`

	Alpha      float64 `gorm:"type:decimal(8,4);not null;default:1.0" json:"alpha"`
	Beta       float64 `gorm:"type:decimal(8,4);not null;default:1.0" json:"beta"`
	Deliveries int     `gorm:"default:0" json:"deliveries"`
	Responses  int     `gorm:"default:0" json:"responses"` // response within 30 min

	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (p *PatientTimingProfile) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}
