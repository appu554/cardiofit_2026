package services

import (
	"fmt"
	"math"
	"time"

	"kb-21-behavioral-intelligence/internal/models"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// OutcomeCorrelationPublisher is satisfied by events.Publisher.
// Defined here to avoid an import cycle (services ↔ events).
type OutcomeCorrelationPublisher interface {
	PublishOutcomeCorrelation(corr models.OutcomeCorrelation) error
}

// CorrelationService computes OutcomeCorrelation — the behavioral-clinical feedback loop.
// This is the entity from Finding F-04 (Gap 5) that enables pharmacological vs. behavioral
// differential diagnosis:
//
//   - CONCORDANT:    adherence↑ + outcome↑ → treatment working → celebrate + reinforce
//   - DISCORDANT:    adherence↑ + outcome flat → pharmacological issue → escalate medication
//   - BEHAVIORAL_GAP: adherence↓ + outcome↓ → fix behavior first → do NOT intensify treatment
//
// Data flow: KB-20 LAB_RESULT events → KB-21 correlation computation → V-MCU consumption
type CorrelationService struct {
	db               *gorm.DB
	logger           *zap.Logger
	minEventsForCorr int
	safetyClient     *SafetyClient                  // G-01: KB-23 direct fast-path
	publisher        OutcomeCorrelationPublisher    // Gap #23: publish OUTCOME_CORRELATION to KB-19
}

func NewCorrelationService(db *gorm.DB, logger *zap.Logger, minEvents int, safetyClient *SafetyClient, publisher OutcomeCorrelationPublisher) *CorrelationService {
	return &CorrelationService{
		db:               db,
		logger:           logger,
		minEventsForCorr: minEvents,
		safetyClient:     safetyClient,
		publisher:        publisher,
	}
}

// LabResultEvent represents an inbound lab result from KB-20's LAB_RESULT event bus.
type LabResultEvent struct {
	PatientID   string    `json:"patient_id"`
	LabType     string    `json:"lab_type"`     // HBA1C, FBG, BP_SYSTOLIC, EGFR
	Value       float64   `json:"value"`
	Unit        string    `json:"unit"`
	CollectedAt time.Time `json:"collected_at"`
}

// OnLabResult is called when KB-20 publishes a LAB_RESULT event.
// It recomputes the OutcomeCorrelation for the 90-day period ending at this lab result.
func (s *CorrelationService) OnLabResult(event LabResultEvent) error {
	if event.LabType != "HBA1C" && event.LabType != "FBG" {
		return nil // Only HbA1c and FBG trigger correlation recomputation
	}

	patientID := event.PatientID
	periodEnd := event.CollectedAt
	periodStart := periodEnd.AddDate(0, -3, 0) // 90-day correlation window

	// Gather behavioral data for the period
	var adherenceStates []models.AdherenceState
	if err := s.db.Where("patient_id = ?", patientID).Find(&adherenceStates).Error; err != nil {
		return fmt.Errorf("failed to get adherence states: %w", err)
	}

	if len(adherenceStates) == 0 {
		return nil // No adherence data yet
	}

	// Compute mean adherence across drug classes
	var sumAdh float64
	for _, a := range adherenceStates {
		sumAdh += a.AdherenceScore
	}
	meanAdherence := sumAdh / float64(len(adherenceStates))

	// Determine dominant trend and phenotype
	trend := s.dominantTrend(adherenceStates)

	var profile models.EngagementProfile
	if err := s.db.Where("patient_id = ?", patientID).First(&profile).Error; err != nil {
		return fmt.Errorf("failed to get engagement profile: %w", err)
	}

	// Find the previous HbA1c for delta computation
	var prevCorrelation models.OutcomeCorrelation
	hasPrev := s.db.Where("patient_id = ? AND hba1c_end IS NOT NULL", patientID).
		Order("period_end DESC").First(&prevCorrelation).Error == nil

	// Build the correlation record
	corr := models.OutcomeCorrelation{
		ID:                 uuid.New(),
		PatientID:          patientID,
		PeriodStart:        periodStart,
		PeriodEnd:          periodEnd,
		MeanAdherenceScore: meanAdherence,
		AdherenceTrend:     trend,
		DominantPhenotype:  profile.Phenotype,
		ComputedAt:         time.Now().UTC(),
	}

	// Populate clinical values based on lab type
	switch event.LabType {
	case "HBA1C":
		val := event.Value
		corr.HbA1cEnd = &val
		if hasPrev && prevCorrelation.HbA1cEnd != nil {
			corr.HbA1cStart = prevCorrelation.HbA1cEnd
			delta := val - *prevCorrelation.HbA1cEnd
			corr.HbA1cDelta = &delta
		}
	case "FBG":
		val := event.Value
		corr.FBGMean = &val
		corr.FBGTrend = s.classifyLabTrend(patientID, "FBG", periodStart, periodEnd)
	}

	// Count interaction events in the period for confidence
	var eventCount int64
	s.db.Model(&models.InteractionEvent{}).
		Where("patient_id = ? AND timestamp BETWEEN ? AND ?", patientID, periodStart, periodEnd).
		Count(&eventCount)

	// Classify treatment response
	corr.TreatmentResponseClass = s.classifyResponse(meanAdherence, trend, corr.HbA1cDelta, int(eventCount))
	corr.CorrelationStrength = s.computeCorrelationStrength(meanAdherence, corr.HbA1cDelta)
	corr.ConfidenceLevel = s.classifyConfidence(int(eventCount))

	// Celebration eligibility (Gap 5 Q3)
	corr.CelebrationEligible = s.isCelebrationEligible(corr)
	if corr.CelebrationEligible {
		corr.CelebrationMessage = s.generateCelebrationMessage(corr)
	}

	if err := s.db.Create(&corr).Error; err != nil {
		return fmt.Errorf("failed to save outcome correlation: %w", err)
	}

	s.logger.Info("OutcomeCorrelation computed",
		zap.String("patient_id", patientID),
		zap.String("response_class", string(corr.TreatmentResponseClass)),
		zap.Float64("mean_adherence", meanAdherence),
		zap.Bool("celebration_eligible", corr.CelebrationEligible),
	)

	// Gap #23: Publish OUTCOME_CORRELATION to KB-19 for protocol-level awareness.
	// V-MCU uses treatment_response_class to decide: escalate (CONCORDANT), hold (DISCORDANT),
	// or behavioural intervention (BEHAVIORAL_GAP).
	if s.publisher != nil {
		if err := s.publisher.PublishOutcomeCorrelation(corr); err != nil {
			s.logger.Error("failed to publish OUTCOME_CORRELATION event",
				zap.String("patient_id", patientID),
				zap.Error(err),
			)
		}
	}

	// G-01: Alert KB-23 directly for BEHAVIORAL_GAP and DISCORDANT classifications.
	// This is the two-hop fast-path (KB-21 → KB-23) that ensures V-MCU does not
	// escalate insulin on a patient where the evidence says: do not escalate.
	if s.safetyClient != nil {
		if err := s.safetyClient.AlertBehavioralGap(corr); err != nil {
			s.logger.Error("G-01: failed to alert KB-23 for behavioral gap — will retry via event bus",
				zap.String("patient_id", patientID),
				zap.String("response_class", string(corr.TreatmentResponseClass)),
				zap.Error(err),
			)
			// Don't fail the correlation save — log and continue
		}
	}

	return nil
}

// GetLatestCorrelation returns the most recent OutcomeCorrelation for a patient.
func (s *CorrelationService) GetLatestCorrelation(patientID string) (*models.OutcomeCorrelation, error) {
	var corr models.OutcomeCorrelation
	err := s.db.Where("patient_id = ?", patientID).
		Order("period_end DESC").First(&corr).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &corr, err
}

// GetCorrelationHistory returns all outcome correlations for a patient.
func (s *CorrelationService) GetCorrelationHistory(patientID string) ([]models.OutcomeCorrelation, error) {
	var corrs []models.OutcomeCorrelation
	err := s.db.Where("patient_id = ?", patientID).
		Order("period_end DESC").Find(&corrs).Error
	return corrs, err
}

// --- Internal classification logic ---

// classifyResponse implements the three-loop differential diagnosis from Gap 5.
func (s *CorrelationService) classifyResponse(
	meanAdherence float64,
	trend models.AdherenceTrend,
	hba1cDelta *float64,
	eventCount int,
) models.TreatmentResponseClass {
	if eventCount < s.minEventsForCorr {
		return models.ResponseInsufficient
	}

	highAdherence := meanAdherence >= 0.70
	improving := false
	if hba1cDelta != nil {
		improving = *hba1cDelta < -0.3 // HbA1c drop > 0.3% = meaningful improvement
	}

	switch {
	case highAdherence && improving:
		return models.ResponseConcordant
	case highAdherence && !improving:
		return models.ResponseDiscordant
	case !highAdherence:
		return models.ResponseBehavioral
	default:
		return models.ResponseInsufficient
	}
}

func (s *CorrelationService) computeCorrelationStrength(adherence float64, hba1cDelta *float64) float64 {
	if hba1cDelta == nil {
		return 0
	}
	// Simple correlation proxy: how strongly does adherence predict outcome direction?
	// High adherence + large negative delta = strong positive correlation
	// High adherence + no delta = weak correlation
	delta := math.Abs(*hba1cDelta)
	strength := clamp(adherence*delta/2.0, 0, 1)
	return strength
}

func (s *CorrelationService) classifyConfidence(eventCount int) string {
	switch {
	case eventCount >= 30:
		return "HIGH"
	case eventCount >= 15:
		return "MODERATE"
	default:
		return "LOW"
	}
}

func (s *CorrelationService) isCelebrationEligible(corr models.OutcomeCorrelation) bool {
	return corr.TreatmentResponseClass == models.ResponseConcordant &&
		corr.MeanAdherenceScore >= 0.75 &&
		corr.HbA1cDelta != nil && *corr.HbA1cDelta < -0.3
}

func (s *CorrelationService) generateCelebrationMessage(corr models.OutcomeCorrelation) string {
	if corr.HbA1cDelta == nil {
		return ""
	}
	delta := math.Abs(*corr.HbA1cDelta)
	// Outcome-linked celebration per Gap 5 Q3:
	// "Your blood sugar is the best it's been in 3 months — your consistency is paying off."
	return fmt.Sprintf(
		"Aapka HbA1c %.1f%% kam hua hai pichhle 3 mahine mein. Aapki niyamitata kaam kar rahi hai! "+
			"(Your HbA1c dropped by %.1f%% in the last 3 months. Your consistency is paying off!)",
		delta, delta,
	)
}

func (s *CorrelationService) dominantTrend(states []models.AdherenceState) models.AdherenceTrend {
	for _, a := range states {
		if a.AdherenceTrend == models.TrendCritical {
			return models.TrendCritical
		}
	}
	for _, a := range states {
		if a.AdherenceTrend == models.TrendDeclining {
			return models.TrendDeclining
		}
	}
	for _, a := range states {
		if a.AdherenceTrend == models.TrendImproving {
			return models.TrendImproving
		}
	}
	return models.TrendStable
}

func (s *CorrelationService) classifyLabTrend(patientID, labType string, start, end time.Time) string {
	// Simplified: check if the latest value is better than the period mean
	// Full implementation would query KB-20 directly
	return "STABLE"
}
