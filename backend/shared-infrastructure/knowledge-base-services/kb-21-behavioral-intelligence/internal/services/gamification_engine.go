package services

import (
	"time"

	"kb-21-behavioral-intelligence/internal/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// GamificationEngine manages streaks, milestones, and weekly challenges (E2 §4).
// Activation rule: only for patients where T-06 posterior_mean > 0.15
// OR cold-start phenotype = REWARD_RESPONSIVE (spec §4.1).
type GamificationEngine struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewGamificationEngine(db *gorm.DB, logger *zap.Logger) *GamificationEngine {
	return &GamificationEngine{db: db, logger: logger}
}

// ShouldActivate determines if gamification should be active for this patient.
// Per spec §4.1: T-06 posterior_mean > 0.15 OR phenotype = REWARD_RESPONSIVE.
func (ge *GamificationEngine) ShouldActivate(phenotype models.ColdStartPhenotype, t06PosteriorMean float64) bool {
	if phenotype == models.PhenotypeRewardResponsive {
		return true
	}
	return t06PosteriorMean > 0.15
}

// UpdateStreak records a behavior completion and updates the streak.
// Handles: increment on consecutive day, break on gap, continue on pause.
func (ge *GamificationEngine) UpdateStreak(streak *models.PatientStreak, eventTime time.Time) {
	today := eventTime.Truncate(24 * time.Hour)
	lastDay := streak.LastActiveDay.Truncate(24 * time.Hour)
	daysSince := int(today.Sub(lastDay).Hours() / 24)

	// If paused, streak continues (illness/travel pause per spec governance rule)
	if streak.Paused {
		streak.CurrentStreak++
		streak.Paused = false
		streak.PausedAt = nil
		streak.PauseReason = ""
	} else if daysSince <= 1 {
		// Consecutive day (same day or next day)
		if daysSince == 1 {
			streak.CurrentStreak++
		}
		// daysSince == 0: same day, no increment
	} else {
		// Gap > 1 day: break streak
		streak.CurrentStreak = 1
	}

	// Update longest
	if streak.CurrentStreak > streak.LongestStreak {
		streak.LongestStreak = streak.CurrentStreak
	}

	streak.LastActiveDay = today

	if ge.db != nil {
		ge.db.Save(streak)
	}
}

// PauseStreak pauses a streak due to illness, travel, or festival (spec §4.1 governance).
func (ge *GamificationEngine) PauseStreak(streak *models.PatientStreak, reason string) {
	now := time.Now().UTC()
	streak.Paused = true
	streak.PausedAt = &now
	streak.PauseReason = reason

	if ge.db != nil {
		ge.db.Save(streak)
	}
}

// GetOrCreateStreak loads or initializes a streak for a patient behavior.
func (ge *GamificationEngine) GetOrCreateStreak(patientID, behavior string) (*models.PatientStreak, error) {
	if ge.db == nil {
		return &models.PatientStreak{
			PatientID: patientID,
			Behavior:  behavior,
		}, nil
	}

	var streak models.PatientStreak
	result := ge.db.Where("patient_id = ? AND behavior = ?", patientID, behavior).First(&streak)
	if result.Error == gorm.ErrRecordNotFound {
		streak = models.PatientStreak{
			PatientID:     patientID,
			Behavior:      behavior,
			CurrentStreak: 0,
			LongestStreak: 0,
			LastActiveDay: time.Now().UTC().Truncate(24 * time.Hour),
		}
		if err := ge.db.Create(&streak).Error; err != nil {
			return nil, err
		}
		return &streak, nil
	}
	return &streak, result.Error
}

// GetPatientStreaks returns all streaks for a patient.
func (ge *GamificationEngine) GetPatientStreaks(patientID string) ([]models.PatientStreak, error) {
	if ge.db == nil {
		return nil, nil
	}
	var streaks []models.PatientStreak
	err := ge.db.Where("patient_id = ?", patientID).Find(&streaks).Error
	return streaks, err
}

// DetectMilestones checks if the patient has hit any new milestones.
// cycleDay: current day in correction cycle
// adherenceScore: current 30-day adherence
// existingMilestones: already-achieved milestone types (to avoid duplicates)
func (ge *GamificationEngine) DetectMilestones(patientID string, cycleDay int, adherenceScore float64, existingTypes []string) []models.PatientMilestone {
	existing := map[string]bool{}
	for _, t := range existingTypes {
		existing[t] = true
	}

	now := time.Now().UTC()
	var milestones []models.PatientMilestone

	// First week complete
	if cycleDay >= 7 && !existing["FIRST_WEEK_COMPLETE"] {
		milestones = append(milestones, models.PatientMilestone{
			PatientID:     patientID,
			MilestoneType: "FIRST_WEEK_COMPLETE",
			Title:         "First Week Complete",
			Description:   "You completed your first full week in the program!",
			AchievedAt:    now,
		})
	}

	// Two-week adherence champion
	if cycleDay >= 14 && adherenceScore >= 0.70 && !existing["TWO_WEEK_ADHERENCE"] {
		milestones = append(milestones, models.PatientMilestone{
			PatientID:     patientID,
			MilestoneType: "TWO_WEEK_ADHERENCE",
			Title:         "Two-Week Adherence Champion",
			Description:   "Maintained 70%+ adherence for two full weeks!",
			AchievedAt:    now,
		})
	}

	// One month milestone
	if cycleDay >= 30 && !existing["ONE_MONTH_COMPLETE"] {
		milestones = append(milestones, models.PatientMilestone{
			PatientID:     patientID,
			MilestoneType: "ONE_MONTH_COMPLETE",
			Title:         "One Month Milestone",
			Description:   "One full month of building healthier habits!",
			AchievedAt:    now,
		})
	}

	// Mastery approach
	if cycleDay >= 60 && adherenceScore >= 0.60 && !existing["MASTERY_APPROACH"] {
		milestones = append(milestones, models.PatientMilestone{
			PatientID:     patientID,
			MilestoneType: "MASTERY_APPROACH",
			Title:         "Approaching Mastery",
			Description:   "60 days of consistent effort — habits are becoming automatic!",
			AchievedAt:    now,
		})
	}

	if ge.db != nil {
		for i := range milestones {
			ge.db.Create(&milestones[i])
		}
	}

	return milestones
}

// GetPatientMilestones returns all milestones for a patient.
func (ge *GamificationEngine) GetPatientMilestones(patientID string) ([]models.PatientMilestone, error) {
	if ge.db == nil {
		return nil, nil
	}
	var milestones []models.PatientMilestone
	err := ge.db.Where("patient_id = ?", patientID).Order("achieved_at DESC").Find(&milestones).Error
	return milestones, err
}
