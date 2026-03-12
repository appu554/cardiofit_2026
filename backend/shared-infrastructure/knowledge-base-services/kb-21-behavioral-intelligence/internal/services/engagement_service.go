package services

import (
	"fmt"
	"time"

	"kb-21-behavioral-intelligence/internal/models"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// EngagementService manages patient engagement profiles, phenotyping, and loop trust.
type EngagementService struct {
	db                         *gorm.DB
	logger                     *zap.Logger
	trustEngine                *TrustEngine
	phenotypeEngine            *PhenotypeEngine
	preGatewayDefaultAdherence float64 // G-04
}

func NewEngagementService(db *gorm.DB, logger *zap.Logger, preGatewayDefault float64) *EngagementService {
	return &EngagementService{
		db:                         db,
		logger:                     logger,
		trustEngine:                NewTrustEngine(),
		phenotypeEngine:            NewPhenotypeEngine(),
		preGatewayDefaultAdherence: preGatewayDefault,
	}
}

// GetEngagementProfile returns the current engagement profile for a patient.
func (s *EngagementService) GetEngagementProfile(patientID string) (*models.EngagementProfile, error) {
	var profile models.EngagementProfile
	err := s.db.Where("patient_id = ?", patientID).First(&profile).Error
	if err == gorm.ErrRecordNotFound {
		return s.initProfile(patientID)
	}
	return &profile, err
}

// GetLoopTrust returns the loop trust response for V-MCU (Finding F-01).
// G-02 fix: returns per-drug-class adherence so V-MCU can compute gain_factor per class.
// G-04 fix: returns PRE_GATEWAY_DEFAULT_ADHERENCE when no interaction data exists.
func (s *EngagementService) GetLoopTrust(patientID string) (*models.LoopTrustResponse, error) {
	profile, err := s.GetEngagementProfile(patientID)
	if err != nil {
		return nil, err
	}

	var adherenceStates []models.AdherenceState
	s.db.Where("patient_id = ?", patientID).Find(&adherenceStates)

	// G-04: Determine adherence source. If no adherence states exist, the WhatsApp
	// gateway has not connected yet. Use pre-gateway default so V-MCU doesn't
	// bottom out gain_factor at 0.25 for every patient indefinitely.
	adherenceSource := models.AdherenceSourceObserved
	var score30d, score7d float64

	// G-02: Build per-class adherence map
	perClass := make(map[string]models.DrugClassAdherence)

	if len(adherenceStates) > 0 {
		var sum30, sum7 float64
		for _, a := range adherenceStates {
			sum30 += a.AdherenceScore
			sum7 += a.AdherenceScore7d

			perClass[a.DrugClass] = models.DrugClassAdherence{
				DrugClass:   a.DrugClass,
				Score7d:     a.AdherenceScore7d,
				Score30d:    a.AdherenceScore,
				Trend:       a.AdherenceTrend,
				DataQuality: a.DataQuality,
				IsFDC:       a.IsFDC,
				Source:      models.AdherenceSourceObserved,
			}
		}
		score30d = sum30 / float64(len(adherenceStates))
		score7d = sum7 / float64(len(adherenceStates))
	} else {
		// G-04: No interaction data — apply pre-gateway default
		adherenceSource = models.AdherenceSourcePreGateway
		score30d = s.preGatewayDefaultAdherence
		score7d = s.preGatewayDefaultAdherence

		s.logger.Info("Using PRE_GATEWAY_DEFAULT_ADHERENCE — no interaction data for patient",
			zap.String("patient_id", patientID),
			zap.Float64("default_value", s.preGatewayDefaultAdherence),
		)
	}

	recommendation := s.trustEngine.RecommendAuthority(profile.LoopTrustScore)

	return &models.LoopTrustResponse{
		PatientID:      patientID,
		LoopTrustScore: profile.LoopTrustScore,
		Components: models.LoopTrustComponents{
			AdherenceScore:    score30d,
			DataQualityWeight: profile.DataQualityWeight,
			PhenotypeWeight:   profile.PhenotypeWeight,
			TemporalStability: profile.TemporalStability,
		},
		Phenotype:         profile.Phenotype,
		AdherenceScore7d:  score7d,
		AdherenceScore30d: score30d,
		Recommendation:    recommendation,
		PerClassAdherence: perClass,
		AdherenceSource:   adherenceSource,
	}, nil
}

// RecomputeProfile recalculates the engagement profile from current state.
// Called periodically (every PhenotypeEvalIntervalHours) or after significant events.
func (s *EngagementService) RecomputeProfile(patientID string) error {
	profile, err := s.GetEngagementProfile(patientID)
	if err != nil {
		return err
	}

	now := time.Now().UTC()

	// Count recent interactions
	var count7d, count30d int64
	s.db.Model(&models.InteractionEvent{}).
		Where("patient_id = ? AND timestamp > ?", patientID, now.AddDate(0, 0, -7)).
		Count(&count7d)
	s.db.Model(&models.InteractionEvent{}).
		Where("patient_id = ? AND timestamp > ?", patientID, now.AddDate(0, 0, -30)).
		Count(&count30d)

	profile.InteractionsLast7d = int(count7d)
	profile.InteractionsLast30d = int(count30d)

	// Total interactions
	var totalCount int64
	s.db.Model(&models.InteractionEvent{}).
		Where("patient_id = ?", patientID).Count(&totalCount)
	profile.TotalInteractions = int(totalCount)

	// Last interaction
	var lastEvent models.InteractionEvent
	if err := s.db.Where("patient_id = ?", patientID).
		Order("timestamp DESC").First(&lastEvent).Error; err == nil {
		profile.LastInteractionAt = &lastEvent.Timestamp
		profile.DaysSinceLastInteraction = int(now.Sub(lastEvent.Timestamp).Hours() / 24)
	}

	// Average response latency
	var avgLatency struct{ Avg float64 }
	s.db.Model(&models.InteractionEvent{}).
		Select("COALESCE(AVG(response_latency_ms), 0) as avg").
		Where("patient_id = ? AND response_latency_ms > 0 AND timestamp > ?", patientID, now.AddDate(0, 0, -30)).
		Scan(&avgLatency)
	profile.AvgResponseLatencyMs = int64(avgLatency.Avg)

	// Compute engagement score (normalized composite)
	profile.EngagementScore = s.computeEngagementScore(profile)

	// Get aggregated adherence for trust computation
	var adherenceStates []models.AdherenceState
	s.db.Where("patient_id = ?", patientID).Find(&adherenceStates)

	var avgAdherence float64
	var dominantQuality models.DataQuality = "LOW"
	var dominantTrend models.AdherenceTrend = models.TrendStable

	if len(adherenceStates) > 0 {
		var sum float64
		for _, a := range adherenceStates {
			sum += a.AdherenceScore
			if a.DataQuality == "HIGH" || (a.DataQuality == "MODERATE" && dominantQuality == "LOW") {
				dominantQuality = a.DataQuality
			}
			if a.AdherenceTrend == models.TrendDeclining || a.AdherenceTrend == models.TrendCritical {
				dominantTrend = a.AdherenceTrend
			}
		}
		avgAdherence = sum / float64(len(adherenceStates))
	}

	// Classify phenotype
	newPhenotype := s.phenotypeEngine.Classify(avgAdherence, dominantTrend, profile.DaysSinceLastInteraction)
	if newPhenotype != profile.Phenotype {
		profile.PreviousPhenotype = profile.Phenotype
		profile.Phenotype = newPhenotype
		profile.PhenotypeSince = now
	}

	// Compute loop trust score (Finding F-01)
	profile.DataQualityWeight = s.trustEngine.DataQualityWeight(dominantQuality)
	profile.PhenotypeWeight = s.trustEngine.PhenotypeWeight(profile.Phenotype)
	profile.TemporalStability = s.trustEngine.TemporalStability(dominantTrend)
	profile.LoopTrustScore = s.trustEngine.ComputeLoopTrust(
		avgAdherence,
		profile.DataQualityWeight,
		profile.PhenotypeWeight,
		profile.TemporalStability,
	)

	// Decay risk
	profile.DecayRiskScore = s.computeDecayRisk(profile)

	return s.db.Save(profile).Error
}

// --- Internal helpers ---

func (s *EngagementService) initProfile(patientID string) (*models.EngagementProfile, error) {
	profile := &models.EngagementProfile{
		ID:               uuid.New(),
		PatientID:        patientID,
		EngagementScore:  0,
		Phenotype:        models.PhenotypeSteady,
		PhenotypeSince:   time.Now().UTC(),
		LoopTrustScore:   0,
		OnboardingStatus: "NOT_STARTED",
	}
	if err := s.db.Create(profile).Error; err != nil {
		return nil, fmt.Errorf("failed to init engagement profile: %w", err)
	}
	return profile, nil
}

// computeEngagementScore produces a normalised 0.0–1.0 engagement score.
// Weighted: 40% interaction frequency, 30% response quality, 20% consistency, 10% latency.
func (s *EngagementService) computeEngagementScore(profile *models.EngagementProfile) float64 {
	// Interaction frequency (normalised to expected 1/day = 30/month)
	freqScore := clamp(float64(profile.InteractionsLast30d)/30.0, 0, 1)

	// Response quality (approximated by responded/total ratio from adherence)
	var totalCheckins, respondedCheckins int64
	s.db.Model(&models.InteractionEvent{}).
		Where("patient_id = ? AND timestamp > ?", profile.PatientID, time.Now().UTC().AddDate(0, 0, -30)).
		Count(&totalCheckins)
	s.db.Model(&models.InteractionEvent{}).
		Where("patient_id = ? AND timestamp > ? AND response_quality IN ('HIGH', 'MODERATE')",
			profile.PatientID, time.Now().UTC().AddDate(0, 0, -30)).
		Count(&respondedCheckins)

	qualityScore := 0.0
	if totalCheckins > 0 {
		qualityScore = float64(respondedCheckins) / float64(totalCheckins)
	}

	// Consistency (7d/30d ratio — ideally ~0.23 if evenly distributed)
	consistencyScore := 0.0
	if profile.InteractionsLast30d > 0 {
		ratio := float64(profile.InteractionsLast7d) / float64(profile.InteractionsLast30d)
		// Ideal ratio is ~0.233 (7/30). Score 1.0 at that ratio, penalise deviation.
		consistencyScore = clamp(1.0-4.0*absFloat(ratio-0.233), 0, 1)
	}

	// Latency score (lower is better; <5s = 1.0, >60s = 0.0)
	latencyScore := clamp(1.0-float64(profile.AvgResponseLatencyMs)/(60*1000), 0, 1)

	return 0.40*freqScore + 0.30*qualityScore + 0.20*consistencyScore + 0.10*latencyScore
}

func (s *EngagementService) computeDecayRisk(profile *models.EngagementProfile) float64 {
	risk := 0.0

	// Days since last interaction
	switch {
	case profile.DaysSinceLastInteraction > 21:
		risk += 0.50
	case profile.DaysSinceLastInteraction > 14:
		risk += 0.35
	case profile.DaysSinceLastInteraction > 7:
		risk += 0.20
	case profile.DaysSinceLastInteraction > 3:
		risk += 0.10
	}

	// Declining engagement
	if profile.EngagementScore < 0.30 {
		risk += 0.25
	} else if profile.EngagementScore < 0.50 {
		risk += 0.15
	}

	// Phenotype risk
	switch profile.Phenotype {
	case models.PhenotypeDeclining:
		risk += 0.20
	case models.PhenotypeSporadic:
		risk += 0.10
	}

	return clamp(risk, 0, 1)
}

func clamp(val, min, max float64) float64 {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

// ──────────────────────────────────────────────────
// Salt Sensitivity Assessment (Amendment 12, Wave 3.3)
// ──────────────────────────────────────────────────

// AssessSaltSensitivity evaluates dietary sodium from Tier-1 question responses.
// Tier-1 dietary questions are India-specific, targeting high-sodium dietary patterns
// common in South Asian diets (pickles/papads, post-cooking salt, processed foods).
//
// Scoring:
//
//	Q1 Pickles/papads frequency: DAILY=3, WEEKLY=2, RARELY=1, NEVER=0
//	Q2 Post-cooking salt addition: ALWAYS=3, SOMETIMES=1, NEVER=0
//	Q3 Processed food frequency: DAILY=3, WEEKLY=2, RARELY=1, NEVER=0
//
// Total score 0-9:
//
//	0-2 → LOW sodium, reduction potential 0.1
//	3-5 → MODERATE sodium, reduction potential 0.4
//	6-9 → HIGH sodium, reduction potential 0.7
func (s *EngagementService) AssessSaltSensitivity(patientID string, responses models.SaltQuestionResponses) (*models.SaltSensitivityProfile, error) {
	score := scoreSaltQuestion(responses.PicklesPapadsFrequency, map[string]int{
		"DAILY": 3, "WEEKLY": 2, "RARELY": 1, "NEVER": 0,
	})
	score += scoreSaltQuestion(responses.PostCookingSalt, map[string]int{
		"ALWAYS": 3, "SOMETIMES": 1, "NEVER": 0,
	})
	score += scoreSaltQuestion(responses.ProcessedFoodFrequency, map[string]int{
		"DAILY": 3, "WEEKLY": 2, "RARELY": 1, "NEVER": 0,
	})

	var estimate models.DietarySodiumEstimate
	var reductionPotential float64

	switch {
	case score <= 2:
		estimate = models.SodiumLow
		reductionPotential = 0.1
	case score <= 5:
		estimate = models.SodiumModerate
		reductionPotential = 0.4
	default:
		estimate = models.SodiumHigh
		reductionPotential = 0.7
	}

	now := time.Now().UTC()
	profile := &models.SaltSensitivityProfile{
		PatientID:              patientID,
		DietarySodiumEstimate:  estimate,
		SaltReductionPotential: reductionPotential,
		PicklesPapadsFrequency: responses.PicklesPapadsFrequency,
		PostCookingSalt:        responses.PostCookingSalt,
		ProcessedFoodFrequency: responses.ProcessedFoodFrequency,
		AssessedAt:             now,
	}

	// Upsert: create or update based on patient_id primary key
	result := s.db.Where("patient_id = ?", patientID).First(&models.SaltSensitivityProfile{})
	if result.Error == gorm.ErrRecordNotFound {
		if err := s.db.Create(profile).Error; err != nil {
			return nil, fmt.Errorf("failed to create salt sensitivity profile: %w", err)
		}
	} else {
		if err := s.db.Model(&models.SaltSensitivityProfile{}).
			Where("patient_id = ?", patientID).
			Updates(profile).Error; err != nil {
			return nil, fmt.Errorf("failed to update salt sensitivity profile: %w", err)
		}
	}

	s.logger.Info("Salt sensitivity assessed",
		zap.String("patient_id", patientID),
		zap.String("estimate", string(estimate)),
		zap.Float64("reduction_potential", reductionPotential),
		zap.Int("raw_score", score),
	)

	return profile, nil
}

// GetSaltSensitivity returns the current salt sensitivity profile for a patient.
func (s *EngagementService) GetSaltSensitivity(patientID string) (*models.SaltSensitivityProfile, error) {
	var profile models.SaltSensitivityProfile
	if err := s.db.Where("patient_id = ?", patientID).First(&profile).Error; err != nil {
		return nil, err
	}
	return &profile, nil
}

// scoreSaltQuestion maps a response string to its numeric score using the provided lookup.
// Returns 0 for unrecognised values (defensive default).
func scoreSaltQuestion(response string, lookup map[string]int) int {
	if v, ok := lookup[response]; ok {
		return v
	}
	return 0
}
