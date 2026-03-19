package services

import (
	"time"

	"kb-21-behavioral-intelligence/internal/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// PhaseMultipliers defines technique weight multipliers per motivation phase (spec Table 9).
// During RECOVERY, T-11 (Recovery Protocol) is exclusive for 3 days.
var PhaseMultipliers = map[models.MotivationPhase]map[models.TechniqueID]float64{
	models.PhaseInitiation: {
		models.TechMicroCommitment:        1.5,
		models.TechHabitStacking:          0.8,
		models.TechLossAversion:           0.3, // too early for loss framing
		models.TechSocialNorms:            0.3, // no baseline to compare
		models.TechMicroEducation:         1.3,
		models.TechProgressVisualization:  0.5,
		models.TechEnvironmentRestructure: 0.8,
		models.TechImplementIntention:     0.5,
		models.TechCostAwareSubstitution:  1.2,
		models.TechFamilyInclusion:        0.8,
		models.TechRecoveryProtocol:       0.1,
		models.TechKinshipTone:            1.2,
	},
	models.PhaseExploration: {
		models.TechMicroCommitment:        0.8,
		models.TechHabitStacking:          1.5, // bridge behaviors
		models.TechLossAversion:           0.5,
		models.TechSocialNorms:            0.8,
		models.TechMicroEducation:         1.0,
		models.TechProgressVisualization:  1.3, // show early FBG improvement
		models.TechEnvironmentRestructure: 1.3,
		models.TechImplementIntention:     1.0,
		models.TechCostAwareSubstitution:  1.0,
		models.TechFamilyInclusion:        1.0,
		models.TechRecoveryProtocol:       0.1,
		models.TechKinshipTone:            1.0,
	},
	models.PhaseConsolidation: {
		models.TechMicroCommitment:        0.5,
		models.TechHabitStacking:          1.0,
		models.TechLossAversion:           1.0,
		models.TechSocialNorms:            1.5, // peer comparison now meaningful
		models.TechMicroEducation:         0.8,
		models.TechProgressVisualization:  1.2,
		models.TechEnvironmentRestructure: 0.8,
		models.TechImplementIntention:     1.5, // for anticipated disruptions
		models.TechCostAwareSubstitution:  0.8,
		models.TechFamilyInclusion:        1.0,
		models.TechRecoveryProtocol:       0.1,
		models.TechKinshipTone:            0.8,
	},
	models.PhaseMastery: {
		models.TechMicroCommitment:        0.3,
		models.TechHabitStacking:          0.5,
		models.TechLossAversion:           1.5, // "don't lose your gains" — now appropriate
		models.TechSocialNorms:            1.0,
		models.TechMicroEducation:         0.5,
		models.TechProgressVisualization:  1.5, // 12-week trajectory
		models.TechEnvironmentRestructure: 0.3,
		models.TechImplementIntention:     1.0,
		models.TechCostAwareSubstitution:  0.5,
		models.TechFamilyInclusion:        1.0,
		models.TechRecoveryProtocol:       0.1,
		models.TechKinshipTone:            1.0,
	},
	models.PhaseRecovery: {
		// T-11 exclusive for 3-day recovery window. All others suppressed.
		models.TechMicroCommitment:        0.1,
		models.TechHabitStacking:          0.1,
		models.TechLossAversion:           0.1,
		models.TechSocialNorms:            0.1,
		models.TechMicroEducation:         0.1,
		models.TechProgressVisualization:  0.1,
		models.TechEnvironmentRestructure: 0.1,
		models.TechImplementIntention:     0.1,
		models.TechCostAwareSubstitution:  0.1,
		models.TechFamilyInclusion:        0.1,
		models.TechRecoveryProtocol:       3.0, // dominant
		models.TechKinshipTone:            0.5, // warm tone still allowed
	},
}

// PhaseEngine manages motivation phase state and transitions (BCE v2.0 Enhancement 5).
type PhaseEngine struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewPhaseEngine(db *gorm.DB, logger *zap.Logger) *PhaseEngine {
	return &PhaseEngine{db: db, logger: logger}
}

// DeterminePhase returns the motivation phase for a given cycle day (spec Table 9).
func (pe *PhaseEngine) DeterminePhase(cycleDay int) models.MotivationPhase {
	switch {
	case cycleDay <= 14:
		return models.PhaseInitiation
	case cycleDay <= 35:
		return models.PhaseExploration
	case cycleDay <= 60:
		return models.PhaseConsolidation
	default:
		return models.PhaseMastery
	}
}

// ShouldEnterRecovery determines if the patient needs recovery phase.
// Triggers: adherence drop below 0.40 + declining trend, OR 7+ days inactive.
func (pe *PhaseEngine) ShouldEnterRecovery(adherenceScore float64, trend models.AdherenceTrend, daysInactive int) bool {
	if daysInactive >= 7 {
		return true
	}
	if adherenceScore < 0.40 && (trend == models.TrendDeclining || trend == models.TrendCritical) {
		return true
	}
	return false
}

// ShouldExitRecovery determines if the patient can leave recovery.
// Exit condition: adherence returns to >= 0.50 for 7 consecutive days.
func (pe *PhaseEngine) ShouldExitRecovery(adherenceScore7d float64) bool {
	return adherenceScore7d >= 0.50
}

// GetMultipliers returns phase-specific technique multipliers.
func (pe *PhaseEngine) GetMultipliers(phase models.MotivationPhase) map[models.TechniqueID]float64 {
	if mults, ok := PhaseMultipliers[phase]; ok {
		return mults
	}
	return nil // no multipliers = equal weighting
}

// GetOrCreatePhase loads or initializes the patient's motivation phase.
func (pe *PhaseEngine) GetOrCreatePhase(patientID string) (*models.PatientMotivationPhase, error) {
	if pe.db == nil {
		return &models.PatientMotivationPhase{
			PatientID:      patientID,
			Phase:          models.PhaseInitiation,
			PhaseStartedAt: time.Now().UTC(),
			CycleDay:       1,
			CycleDayStart:  1,
		}, nil
	}

	var phase models.PatientMotivationPhase
	result := pe.db.Where("patient_id = ?", patientID).First(&phase)
	if result.Error == gorm.ErrRecordNotFound {
		phase = models.PatientMotivationPhase{
			PatientID:      patientID,
			Phase:          models.PhaseInitiation,
			PhaseStartedAt: time.Now().UTC(),
			CycleDay:       1,
			CycleDayStart:  1,
		}
		if err := pe.db.Create(&phase).Error; err != nil {
			return nil, err
		}
		return &phase, nil
	}
	if result.Error != nil {
		return nil, result.Error
	}

	// Update cycle day based on elapsed time
	elapsed := int(time.Since(phase.PhaseStartedAt).Hours()/24) + phase.CycleDayStart
	if elapsed != phase.CycleDay {
		phase.CycleDay = elapsed
		pe.db.Save(&phase)
	}

	return &phase, nil
}

// TransitionPhase moves the patient to a new phase, recording the transition.
func (pe *PhaseEngine) TransitionPhase(phase *models.PatientMotivationPhase, newPhase models.MotivationPhase) {
	now := time.Now().UTC()
	phase.PreviousPhase = phase.Phase
	phase.TransitionedAt = &now

	if newPhase == models.PhaseRecovery {
		phase.PreRecoveryPhase = phase.Phase
		phase.RecoveryCount++
	}

	phase.Phase = newPhase
	phase.PhaseStartedAt = now
	phase.CycleDayStart = phase.CycleDay

	if pe.db != nil {
		pe.db.Save(phase)
	}
}
