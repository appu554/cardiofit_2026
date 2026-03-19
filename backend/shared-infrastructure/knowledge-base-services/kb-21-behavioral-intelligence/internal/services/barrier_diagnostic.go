package services

import (
	"kb-21-behavioral-intelligence/internal/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// BarrierSignals aggregates input signals for barrier diagnosis.
type BarrierSignals struct {
	// From adherence data
	MissedDosesLast7d    int
	ConfirmedDosesLast7d int
	ResponseLatencyAvg   int64 // ms

	// Self-reported (from interaction events)
	SelfReportedBarrier models.BarrierCode

	// From engagement profile
	DaysSinceLastInteraction int
	Phenotype                models.BehavioralPhenotype

	// Context
	IsFasting      bool
	HasFamilyLink  bool
	DrugClassCount int
}

// DiagnosedBarrier is a barrier detection with confidence and recommended technique.
type DiagnosedBarrier struct {
	Barrier    models.BarrierCode
	Confidence float64
	Method     string // SELF_REPORT, PATTERN_ANALYSIS
}

// BarrierTechniqueMap links barriers to their primary intervention technique.
var BarrierTechniqueMap = map[models.BarrierCode]models.TechniqueID{
	models.BarrierForgetfulness: models.TechHabitStacking,         // T-02: attach to existing routine
	models.BarrierSideEffects:   models.TechMicroEducation,        // T-05: explain what to expect
	models.BarrierCost:          models.TechCostAwareSubstitution, // T-09: affordable alternatives
	models.BarrierCultural:      models.TechKinshipTone,           // T-12: culturally sensitive framing
	models.BarrierFasting:       models.TechImplementIntention,    // T-08: plan around fasting
	models.BarrierKnowledge:     models.TechMicroEducation,        // T-05: educational content
	models.BarrierAccess:        models.TechCostAwareSubstitution, // T-09: supply alternatives
	models.BarrierPolypharmacy:  models.TechMicroCommitment,       // T-01: simplify to one behavior
}

// BarrierDiagnostic detects adherence barriers from behavioral signals.
type BarrierDiagnostic struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewBarrierDiagnostic creates a new BarrierDiagnostic. db and logger may be nil.
func NewBarrierDiagnostic(db *gorm.DB, logger *zap.Logger) *BarrierDiagnostic {
	return &BarrierDiagnostic{db: db, logger: logger}
}

// Diagnose detects barriers from aggregated signals.
func (bd *BarrierDiagnostic) Diagnose(signals BarrierSignals) []DiagnosedBarrier {
	var barriers []DiagnosedBarrier

	// Self-reported barrier (highest confidence)
	if signals.SelfReportedBarrier != "" {
		barriers = append(barriers, DiagnosedBarrier{
			Barrier:    signals.SelfReportedBarrier,
			Confidence: 0.95,
			Method:     "SELF_REPORT",
		})
	}

	// Pattern: frequent misses with fast response → forgetfulness (not disengagement)
	totalDoses := signals.MissedDosesLast7d + signals.ConfirmedDosesLast7d
	if totalDoses > 0 {
		missRate := float64(signals.MissedDosesLast7d) / float64(totalDoses)
		if missRate >= 0.50 && signals.ResponseLatencyAvg > 0 && signals.ResponseLatencyAvg < 600000 {
			barriers = append(barriers, DiagnosedBarrier{
				Barrier:    models.BarrierForgetfulness,
				Confidence: 0.70,
				Method:     "PATTERN_ANALYSIS",
			})
		}
	}

	// Pattern: fasting detected → fasting barrier
	if signals.IsFasting {
		barriers = append(barriers, DiagnosedBarrier{
			Barrier:    models.BarrierFasting,
			Confidence: 0.85,
			Method:     "PATTERN_ANALYSIS",
		})
	}

	// Pattern: polypharmacy (3+ drug classes)
	if signals.DrugClassCount >= 3 {
		barriers = append(barriers, DiagnosedBarrier{
			Barrier:    models.BarrierPolypharmacy,
			Confidence: 0.60,
			Method:     "PATTERN_ANALYSIS",
		})
	}

	return barriers
}

// RecommendTechnique returns the primary technique for addressing a barrier.
func (bd *BarrierDiagnostic) RecommendTechnique(barrier models.BarrierCode) models.TechniqueID {
	if tech, ok := BarrierTechniqueMap[barrier]; ok {
		return tech
	}
	return models.TechMicroCommitment // safe default
}
