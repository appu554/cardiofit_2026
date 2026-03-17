package services

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"kb-21-behavioral-intelligence/internal/models"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	adherenceThreshold = 0.70 // Finding F-06: below this, drug-related CMs scale proportionally
)

// AdherenceService computes and manages per-drug adherence state.
type AdherenceService struct {
	db               *gorm.DB
	logger           *zap.Logger
	kb1Client        KB1Client         // FDC decomposition via KB-1; nil-safe (uses NoOp)
	festivalCalendar *FestivalCalendar // Festival Mode calendar; nil-safe (no adjustment when nil)
}

// NewAdherenceService creates an adherence service without KB-1 integration.
// FDC decomposition relies on pre-existing FDCComponents data in the adherence state.
func NewAdherenceService(db *gorm.DB, logger *zap.Logger) *AdherenceService {
	return &AdherenceService{db: db, logger: logger, kb1Client: &NoOpKB1Client{}}
}

// NewAdherenceServiceWithKB1 creates an adherence service with KB-1 FDC integration.
// When a medication is checked for adherence, KB-1 is consulted to decompose FDCs
// into their constituent drug classes so a single FDC pill counts as adherence
// to ALL constituent classes (e.g., telmisartan+amlodipine = ARB + CCB).
func NewAdherenceServiceWithKB1(db *gorm.DB, logger *zap.Logger, kb1 KB1Client) *AdherenceService {
	if kb1 == nil {
		kb1 = &NoOpKB1Client{}
	}
	return &AdherenceService{db: db, logger: logger, kb1Client: kb1}
}

// SetKB1Client allows setting the KB-1 client after construction (for late binding).
func (s *AdherenceService) SetKB1Client(kb1 KB1Client) {
	if kb1 != nil {
		s.kb1Client = kb1
	}
}

// SetFestivalCalendar configures the festival calendar for adherence suppression.
// When set, RecomputeAdherence() adjusts scores for patients who have consented
// to festival-mode adaptation (EngagementProfile.ConsentForFestivalAdapt = true).
func (s *AdherenceService) SetFestivalCalendar(fc *FestivalCalendar) {
	s.festivalCalendar = fc
}

// RecordInteraction persists a patient interaction event and triggers adherence recomputation.
func (s *AdherenceService) RecordInteraction(req models.RecordInteractionRequest) (*models.InteractionEvent, error) {
	event := &models.InteractionEvent{
		ID:                   uuid.New(),
		PatientID:            req.PatientID,
		Channel:              req.Channel,
		Type:                 req.Type,
		QuestionID:           req.QuestionID,
		ResponseValue:        req.ResponseValue,
		ResponseQuality:      req.ResponseQuality,
		ResponseLatencyMs:    req.ResponseLatencyMs,
		DrugClass:            req.DrugClass,
		MedicationID:         req.MedicationID,
		EveningMealConfirmed: req.EveningMealConfirmed,
		FastingToday:         req.FastingToday,
		SessionID:            req.SessionID,
		LanguageCode:         req.LanguageCode,
		Timestamp:            time.Now().UTC(),
	}

	if event.LanguageCode == "" {
		event.LanguageCode = "hi"
	}

	if err := s.db.Create(event).Error; err != nil {
		return nil, fmt.Errorf("failed to record interaction: %w", err)
	}

	// If this is a medication confirmation, recompute adherence for the drug class
	if req.Type == models.InteractionMedConfirm && req.DrugClass != "" {
		go func() {
			if err := s.RecomputeAdherence(req.PatientID, req.DrugClass); err != nil {
				s.logger.Error("async adherence recomputation failed",
					zap.String("patient_id", req.PatientID),
					zap.String("drug_class", req.DrugClass),
					zap.Error(err),
				)
			}
		}()
	}

	// Record dietary signal if present (Finding F-05)
	if req.EveningMealConfirmed != nil || req.FastingToday != nil {
		go s.recordDietarySignal(req)
	}

	return event, nil
}

// RecomputeAdherence recalculates both 30-day and 7-day adherence scores
// for a patient/drug_class pair from raw interaction events.
func (s *AdherenceService) RecomputeAdherence(patientID, drugClass string) error {
	now := time.Now().UTC()

	// Compute 30-day recency-weighted score
	score30d, stats30d, err := s.computeWindowedScore(patientID, drugClass, now.AddDate(0, 0, -30), now, true)
	if err != nil {
		return fmt.Errorf("30-day computation failed: %w", err)
	}

	// Compute 7-day unweighted score (Finding F-08)
	score7d, _, err := s.computeWindowedScore(patientID, drugClass, now.AddDate(0, 0, -7), now, false)
	if err != nil {
		return fmt.Errorf("7-day computation failed: %w", err)
	}

	// Determine data quality
	dataQuality := classifyDataQuality(stats30d.respondedCheckIns, stats30d.totalCheckIns)

	// Compute trend via linear regression on weekly adherence buckets
	trend, slope := s.computeTrend(patientID, drugClass)

	// Festival Mode suppression: adjust adherence score for culturally expected dietary deviations.
	// Only applied when: (1) festival calendar is configured, (2) patient consented to festival adaptation.
	if s.festivalCalendar != nil {
		var profile models.EngagementProfile
		if err := s.db.Where("patient_id = ?", patientID).First(&profile).Error; err == nil {
			if profile.ConsentForFestivalAdapt && profile.Region != "" {
				windowStart := now.AddDate(0, 0, -30)
				score30d = s.festivalCalendar.AdjustAdherenceScore(score30d, windowStart, now, profile.Region)

				window7dStart := now.AddDate(0, 0, -7)
				score7d = s.festivalCalendar.AdjustAdherenceScore(score7d, window7dStart, now, profile.Region)

				s.logger.Debug("Festival Mode: adherence scores adjusted",
					zap.String("patient_id", patientID),
					zap.String("region", profile.Region))
			}
		}
	}

	// Upsert adherence state
	state := models.AdherenceState{
		PatientID:         patientID,
		DrugClass:         drugClass,
		AdherenceScore:    score30d,
		AdherenceScore7d:  score7d,
		DataQuality:       dataQuality,
		AdherenceTrend:    trend,
		TrendSlopePerWeek: slope,
		TotalCheckIns:     stats30d.totalCheckIns,
		RespondedCheckIns: stats30d.respondedCheckIns,
		ConfirmedDoses:    stats30d.confirmedDoses,
		MissedDoses:       stats30d.missedDoses,
		LastConfirmedAt:   stats30d.lastConfirmedAt,
		LastMissedAt:      stats30d.lastMissedAt,
		WindowStart:       now.AddDate(0, 0, -30),
		WindowEnd:         now,
	}

	result := s.db.Where("patient_id = ? AND drug_class = ?", patientID, drugClass).
		Assign(state).
		FirstOrCreate(&state)

	if result.Error != nil {
		return fmt.Errorf("failed to upsert adherence state: %w", result.Error)
	}

	// FDC projection (Finding F-07): if this is an FDC, project scores to components
	if state.IsFDC && state.FDCComponents != "" {
		s.projectFDCScores(patientID, state)
	}

	return nil
}

// GetAdherence returns the current adherence state for a patient across all drug classes.
func (s *AdherenceService) GetAdherence(patientID string) ([]models.AdherenceState, error) {
	var states []models.AdherenceState
	err := s.db.Where("patient_id = ?", patientID).Find(&states).Error
	return states, err
}

// GetAdherenceWeights returns adherence-adjusted weights for KB-22 (Finding F-06).
// Formula: adjusted_weight = min(1.0, adherence_score / adherence_threshold)
func (s *AdherenceService) GetAdherenceWeights(patientID string) (*models.AdherenceWeightsResponse, error) {
	states, err := s.GetAdherence(patientID)
	if err != nil {
		return nil, err
	}

	weights := make(map[string]models.AdherenceWeight)
	for _, state := range states {
		adjustedWeight := math.Min(1.0, state.AdherenceScore/adherenceThreshold)
		weights[state.DrugClass] = models.AdherenceWeight{
			DrugClass:      state.DrugClass,
			AdherenceScore: state.AdherenceScore,
			AdjustedWeight: adjustedWeight,
			DataQuality:    state.DataQuality,
			IsFDC:          state.IsFDC,
		}

		// For FDC drugs, also emit weights for each component drug class
		if state.IsFDC && state.FDCComponents != "" {
			var components []string
			if err := json.Unmarshal([]byte(state.FDCComponents), &components); err == nil {
				for _, comp := range components {
					comp = strings.TrimSpace(comp)
					if _, exists := weights[comp]; !exists {
						weights[comp] = models.AdherenceWeight{
							DrugClass:      comp,
							AdherenceScore: state.AdherenceScore,
							AdjustedWeight: adjustedWeight,
							DataQuality:    state.DataQuality,
							IsFDC:          true,
						}
					}
				}
			}
		}
	}

	return &models.AdherenceWeightsResponse{
		PatientID: patientID,
		Weights:   weights,
	}, nil
}

// =============================================================================
// FDC-AWARE DRUG CLASS EXPANSION (Item 3)
// =============================================================================

// MedicationEntry represents a single medication in a patient's active list.
type MedicationEntry struct {
	MedicationID string
	DrugName     string
	DrugClass    string
	DoseMg       float64
}

// expandFDCToClasses takes a medication list and expands any FDCs into their
// constituent drug classes using KB-1. This ensures that a single FDC pill
// counts as adherence to multiple drug classes.
//
// For each medication:
//  1. Check KB-1 if it is an FDC
//  2. If yes, expand into component drug classes
//  3. If no, use the medication's own drug class
//  4. Mark FDC components with IsFDC=true
func (s *AdherenceService) expandFDCToClasses(medications []MedicationEntry) []ExpandedDrugClass {
	var expanded []ExpandedDrugClass
	seen := make(map[string]bool) // prevent duplicate drug classes

	for _, med := range medications {
		// Try KB-1 FDC decomposition
		fdcMapping, err := s.kb1Client.GetFDCComponents(med.DrugName)
		if err != nil {
			s.logger.Warn("KB-1 FDC lookup error, falling back to direct class",
				zap.String("drug_name", med.DrugName),
				zap.Error(err),
			)
			// Fall through to use direct drug class
			fdcMapping = nil
		}

		if fdcMapping != nil && len(fdcMapping.Components) > 0 {
			// FDC detected: expand into component drug classes
			for _, comp := range fdcMapping.Components {
				key := comp.DrugClass + ":" + med.MedicationID
				if seen[key] {
					continue
				}
				seen[key] = true
				expanded = append(expanded, ExpandedDrugClass{
					DrugClass:    comp.DrugClass,
					DrugName:     comp.DrugName,
					IsFDC:        true,
					FDCParent:    fdcMapping.FDCName,
					MedicationID: med.MedicationID,
				})
			}
		} else {
			// Not an FDC: use the medication's own drug class
			key := med.DrugClass + ":" + med.MedicationID
			if !seen[key] {
				seen[key] = true
				expanded = append(expanded, ExpandedDrugClass{
					DrugClass:    med.DrugClass,
					DrugName:     med.DrugName,
					IsFDC:        false,
					MedicationID: med.MedicationID,
				})
			}
		}
	}

	return expanded
}

// enrichAdherenceWithFDC checks adherence states for any medications that might
// be FDCs not yet decomposed, and creates projected adherence records for their
// constituent drug classes using KB-1. This is called during
// GetAntihypertensiveAdherence to ensure FDC-taking patients get credit for
// all constituent classes even if the FDC was not originally registered with
// component information.
func (s *AdherenceService) enrichAdherenceWithFDC(patientID string, states []models.AdherenceState) []models.AdherenceState {
	// For each state where IsFDC is false and FDCComponents is empty,
	// check KB-1 to see if it is actually an FDC
	var enriched []models.AdherenceState
	enriched = append(enriched, states...)
	existingClasses := make(map[string]bool)
	for _, st := range states {
		existingClasses[st.DrugClass] = true
	}

	for _, state := range states {
		if state.IsFDC || state.FDCComponents != "" {
			continue // already decomposed
		}

		// Construct a drug name from the drug class for KB-1 lookup
		// (KB-1 FDC registry uses product names, not classes)
		drugName := state.DrugClass
		if state.MedicationID != "" {
			drugName = state.MedicationID
		}

		fdcMapping, err := s.kb1Client.GetFDCComponents(drugName)
		if err != nil {
			s.logger.Debug("KB-1 FDC enrichment lookup failed",
				zap.String("drug_class", state.DrugClass),
				zap.Error(err),
			)
			continue
		}

		if fdcMapping == nil || len(fdcMapping.Components) == 0 {
			continue
		}

		// This is an FDC — create projected adherence for missing component classes
		var componentClasses []string
		for _, comp := range fdcMapping.Components {
			componentClasses = append(componentClasses, comp.DrugClass)
		}
		componentsJSON, _ := json.Marshal(componentClasses)

		// Update the original state to mark it as FDC
		s.db.Model(&models.AdherenceState{}).
			Where("patient_id = ? AND drug_class = ?", patientID, state.DrugClass).
			Updates(map[string]interface{}{
				"is_fdc":         true,
				"fdc_components": string(componentsJSON),
			})

		// Project adherence to any missing component classes
		for _, comp := range fdcMapping.Components {
			if existingClasses[comp.DrugClass] {
				continue
			}
			existingClasses[comp.DrugClass] = true

			projected := models.AdherenceState{
				PatientID:         patientID,
				DrugClass:         comp.DrugClass,
				MedicationID:      state.MedicationID,
				IsFDC:             true,
				FDCComponents:     string(componentsJSON),
				AdherenceScore:    state.AdherenceScore,
				AdherenceScore7d:  state.AdherenceScore7d,
				DataQuality:       state.DataQuality,
				AdherenceTrend:    state.AdherenceTrend,
				TrendSlopePerWeek: state.TrendSlopePerWeek,
				TotalCheckIns:     state.TotalCheckIns,
				RespondedCheckIns: state.RespondedCheckIns,
				ConfirmedDoses:    state.ConfirmedDoses,
				MissedDoses:       state.MissedDoses,
				LastConfirmedAt:   state.LastConfirmedAt,
				LastMissedAt:      state.LastMissedAt,
				WindowStart:       state.WindowStart,
				WindowEnd:         state.WindowEnd,
				PrimaryBarrier:    state.PrimaryBarrier,
			}

			s.db.Where("patient_id = ? AND drug_class = ?", patientID, comp.DrugClass).
				Assign(projected).
				FirstOrCreate(&projected)

			enriched = append(enriched, projected)
		}
	}

	return enriched
}

// --- Internal computation helpers ---

type adherenceStats struct {
	totalCheckIns     int
	respondedCheckIns int
	confirmedDoses    int
	missedDoses       int
	lastConfirmedAt   *time.Time
	lastMissedAt      *time.Time
}

// computeWindowedScore calculates adherence over a time window.
// If recencyWeighted=true, more recent events carry higher weight (for 30-day).
// If recencyWeighted=false, all events are equally weighted (for 7-day, per F-08).
func (s *AdherenceService) computeWindowedScore(
	patientID, drugClass string,
	windowStart, windowEnd time.Time,
	recencyWeighted bool,
) (float64, adherenceStats, error) {
	var events []models.InteractionEvent
	err := s.db.Where(
		"patient_id = ? AND drug_class = ? AND type = ? AND timestamp BETWEEN ? AND ?",
		patientID, drugClass, models.InteractionMedConfirm, windowStart, windowEnd,
	).Order("timestamp ASC").Find(&events).Error

	if err != nil {
		return 0, adherenceStats{}, err
	}

	stats := adherenceStats{totalCheckIns: len(events)}
	if len(events) == 0 {
		return 0, stats, nil
	}

	var weightedSum, weightTotal float64
	windowDays := windowEnd.Sub(windowStart).Hours() / 24.0

	for _, event := range events {
		stats.respondedCheckIns++
		isConfirmed := strings.ToLower(event.ResponseValue) == "yes" ||
			strings.ToLower(event.ResponseValue) == "haan" ||
			event.ResponseValue == "1"

		var score float64
		if isConfirmed {
			score = 1.0
			stats.confirmedDoses++
			t := event.Timestamp
			stats.lastConfirmedAt = &t
		} else {
			score = 0.0
			stats.missedDoses++
			t := event.Timestamp
			stats.lastMissedAt = &t
		}

		if recencyWeighted {
			// Recency weight: events closer to windowEnd get higher weight
			daysSinceStart := event.Timestamp.Sub(windowStart).Hours() / 24.0
			weight := 0.5 + 0.5*(daysSinceStart/windowDays) // range [0.5, 1.0]
			weightedSum += score * weight
			weightTotal += weight
		} else {
			weightedSum += score
			weightTotal += 1.0
		}
	}

	if weightTotal == 0 {
		return 0, stats, nil
	}

	return weightedSum / weightTotal, stats, nil
}

// computeTrend uses 4 weekly buckets to determine adherence direction.
func (s *AdherenceService) computeTrend(patientID, drugClass string) (models.AdherenceTrend, float64) {
	now := time.Now().UTC()
	weeklyScores := make([]float64, 4)

	for i := 0; i < 4; i++ {
		weekEnd := now.AddDate(0, 0, -7*i)
		weekStart := weekEnd.AddDate(0, 0, -7)
		score, _, err := s.computeWindowedScore(patientID, drugClass, weekStart, weekEnd, false)
		if err != nil {
			continue
		}
		weeklyScores[3-i] = score // oldest first
	}

	// Simple linear regression slope
	n := float64(len(weeklyScores))
	var sumX, sumY, sumXY, sumX2 float64
	for i, y := range weeklyScores {
		x := float64(i)
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	denominator := n*sumX2 - sumX*sumX
	if denominator == 0 {
		return models.TrendStable, 0
	}
	slope := (n*sumXY - sumX*sumY) / denominator

	switch {
	case slope > 0.05:
		return models.TrendImproving, slope
	case slope < -0.10:
		return models.TrendCritical, slope
	case slope < -0.03:
		return models.TrendDeclining, slope
	default:
		return models.TrendStable, slope
	}
}

// projectFDCScores creates projected adherence records for each FDC component (Finding F-07).
func (s *AdherenceService) projectFDCScores(patientID string, fdcState models.AdherenceState) {
	var components []string
	if err := json.Unmarshal([]byte(fdcState.FDCComponents), &components); err != nil {
		s.logger.Warn("failed to parse FDC components", zap.Error(err))
		return
	}

	for _, comp := range components {
		comp = strings.TrimSpace(comp)
		projected := models.AdherenceState{
			PatientID:         patientID,
			DrugClass:         comp,
			MedicationID:      fdcState.MedicationID,
			IsFDC:             true,
			FDCComponents:     fdcState.FDCComponents,
			AdherenceScore:    fdcState.AdherenceScore,
			AdherenceScore7d:  fdcState.AdherenceScore7d,
			DataQuality:       fdcState.DataQuality,
			AdherenceTrend:    fdcState.AdherenceTrend,
			TrendSlopePerWeek: fdcState.TrendSlopePerWeek,
			TotalCheckIns:     fdcState.TotalCheckIns,
			RespondedCheckIns: fdcState.RespondedCheckIns,
			ConfirmedDoses:    fdcState.ConfirmedDoses,
			MissedDoses:       fdcState.MissedDoses,
			LastConfirmedAt:   fdcState.LastConfirmedAt,
			LastMissedAt:      fdcState.LastMissedAt,
			WindowStart:       fdcState.WindowStart,
			WindowEnd:         fdcState.WindowEnd,
		}

		s.db.Where("patient_id = ? AND drug_class = ?", patientID, comp).
			Assign(projected).
			FirstOrCreate(&projected)
	}
}

func (s *AdherenceService) recordDietarySignal(req models.RecordInteractionRequest) {
	signal := models.DietarySignal{
		ID:                   uuid.New(),
		PatientID:            req.PatientID,
		Date:                 time.Now().UTC().Truncate(24 * time.Hour),
		EveningMealConfirmed: req.EveningMealConfirmed != nil && *req.EveningMealConfirmed,
		FastingToday:         req.FastingToday != nil && *req.FastingToday,
		Source:               "SELF_REPORT",
	}

	if err := s.db.Create(&signal).Error; err != nil {
		s.logger.Error("failed to record dietary signal", zap.Error(err))
	}
}

// ──────────────────────────────────────────────────
// Antihypertensive Adherence (Amendment 4, Wave 2)
// ──────────────────────────────────────────────────

// GetAntihypertensiveAdherence computes aggregate adherence across all active
// HTN drug classes for a patient. Returns per-class breakdown and a
// quality-weighted aggregate score.
//
// FDC-aware: if a telmisartan+amlodipine FDC is present, both ARB and CCB
// component rows share the FDC's adherence score (already projected by
// projectFDCScores in RecomputeAdherence).
func (s *AdherenceService) GetAntihypertensiveAdherence(patientID string) (*models.AntihypertensiveAdherenceResponse, error) {
	// Fetch all adherence states for HTN drug classes
	var states []models.AdherenceState
	err := s.db.Where("patient_id = ? AND drug_class IN ?", patientID, models.HTNDrugClasses).
		Find(&states).Error
	if err != nil {
		return nil, fmt.Errorf("failed to query HTN adherence: %w", err)
	}

	if len(states) == 0 {
		return &models.AntihypertensiveAdherenceResponse{
			PatientID:            patientID,
			PrimaryReason:        models.ReasonUnknown,
			PerClassAdherence:    map[string]models.HTNClassAdherence{},
			Source:               models.AdherenceSourcePreGateway,
			DietarySodiumEstimate: "UNKNOWN",
		}, nil
	}

	// FDC enrichment: check KB-1 for any medications that might be FDCs
	// not yet decomposed into constituent drug classes. This ensures a patient
	// taking one FDC pill gets adherence credit for ALL constituent classes.
	states = s.enrichAdherenceWithFDC(patientID, states)

	perClass := make(map[string]models.HTNClassAdherence, len(states))
	var weightedSum30d, weightedSum7d, weightTotal float64
	var primaryReason models.AdherenceReason = models.ReasonUnknown
	var latestBarrierTime time.Time

	for _, state := range states {
		dqWeight := dataQualityWeight(state.DataQuality)

		perClass[state.DrugClass] = models.HTNClassAdherence{
			DrugClass:      state.DrugClass,
			Score30d:       state.AdherenceScore,
			Score7d:        state.AdherenceScore7d,
			Trend:          state.AdherenceTrend,
			DataQuality:    state.DataQuality,
			IsFDC:          state.IsFDC,
			PrimaryBarrier: state.PrimaryBarrier,
		}

		weightedSum30d += state.AdherenceScore * dqWeight
		weightedSum7d += state.AdherenceScore7d * dqWeight
		weightTotal += dqWeight

		// Track most recent barrier for primary reason classification
		if state.PrimaryBarrier != "" && state.UpdatedAt.After(latestBarrierTime) {
			latestBarrierTime = state.UpdatedAt
			primaryReason = barrierToReason(state.PrimaryBarrier)
		}
	}

	aggregate30d := 0.0
	aggregate7d := 0.0
	if weightTotal > 0 {
		aggregate30d = weightedSum30d / weightTotal
		aggregate7d = weightedSum7d / weightTotal
	}

	// Compute aggregate trend from individual trends
	aggregateTrend := s.computeAggregateTrend(states)

	// Estimate dietary sodium from recent dietary signals
	sodiumEstimate, saltPotential := s.estimateDietarySodium(patientID)

	resp := &models.AntihypertensiveAdherenceResponse{
		PatientID:              patientID,
		AggregateScore:         aggregate30d,
		AggregateScore7d:       aggregate7d,
		AggregateTrend:         aggregateTrend,
		PrimaryReason:          primaryReason,
		PerClassAdherence:      perClass,
		ActiveHTNDrugClasses:   len(states),
		DietarySodiumEstimate:  sodiumEstimate,
		SaltReductionPotential: saltPotential,
		Source:                 models.AdherenceSourceObserved,
	}

	// Persist aggregate state for caching / cohort analytics
	if err := s.upsertHTNAdherenceState(patientID, resp); err != nil {
		s.logger.Warn("failed to persist HTN adherence state", zap.Error(err))
	}

	return resp, nil
}

// EvaluateHTNAdherenceGate determines the card behaviour based on the
// aggregate antihypertensive adherence score and primary reason.
//
// Decision matrix (from Amendment 4):
//
//	Adherence >= 0.85           → STANDARD_ESCALATION
//	Adherence 0.60-0.84        → ADHERENCE_LEAD
//	Adherence < 0.60           → ADHERENCE_INTERVENTION
//	Reason == SIDE_EFFECT (any) → SIDE_EFFECT_HPI (overrides above)
func (s *AdherenceService) EvaluateHTNAdherenceGate(patientID string) (models.HTNAdherenceGateAction, *models.AntihypertensiveAdherenceResponse, error) {
	resp, err := s.GetAntihypertensiveAdherence(patientID)
	if err != nil {
		return models.GateStandardEscalation, nil, err
	}

	// Side effect override: if any HTN drug class has SIDE_EFFECTS barrier, route to HPI
	for _, cls := range resp.PerClassAdherence {
		if cls.PrimaryBarrier == models.BarrierSideEffects {
			return models.GateSideEffectHPI, resp, nil
		}
	}

	switch {
	case resp.AggregateScore >= 0.85:
		return models.GateStandardEscalation, resp, nil
	case resp.AggregateScore >= 0.60:
		return models.GateAdherenceLead, resp, nil
	default:
		return models.GateAdherenceIntervention, resp, nil
	}
}

// upsertHTNAdherenceState persists the aggregate HTN adherence to the database.
func (s *AdherenceService) upsertHTNAdherenceState(patientID string, resp *models.AntihypertensiveAdherenceResponse) error {
	perClassJSON, err := json.Marshal(resp.PerClassAdherence)
	if err != nil {
		return fmt.Errorf("marshal per-class adherence: %w", err)
	}

	state := models.AntihypertensiveAdherenceState{
		PatientID:              patientID,
		PerClassAdherence:      perClassJSON,
		AggregateScore:         resp.AggregateScore,
		AggregateScore7d:       resp.AggregateScore7d,
		AggregateTrend:         resp.AggregateTrend,
		PrimaryReason:          resp.PrimaryReason,
		DietarySodiumEstimate:  resp.DietarySodiumEstimate,
		SaltReductionPotential: resp.SaltReductionPotential,
		ActiveHTNDrugClasses:   resp.ActiveHTNDrugClasses,
	}

	result := s.db.Where("patient_id = ?", patientID).
		Assign(state).
		FirstOrCreate(&state)

	return result.Error
}

// computeAggregateTrend derives the worst trend across all HTN drug classes.
func (s *AdherenceService) computeAggregateTrend(states []models.AdherenceState) models.AdherenceTrend {
	worst := models.TrendStable
	for _, state := range states {
		if trendSeverity(state.AdherenceTrend) > trendSeverity(worst) {
			worst = state.AdherenceTrend
		}
	}
	return worst
}

func trendSeverity(t models.AdherenceTrend) int {
	switch t {
	case models.TrendImproving:
		return 0
	case models.TrendStable:
		return 1
	case models.TrendDeclining:
		return 2
	case models.TrendCritical:
		return 3
	default:
		return 1
	}
}

// estimateDietarySodium estimates sodium intake from recent dietary signals.
// Circle 1 approximation: uses meal regularity and carb category as proxy.
func (s *AdherenceService) estimateDietarySodium(patientID string) (string, float64) {
	var signals []models.DietarySignal
	cutoff := time.Now().UTC().AddDate(0, 0, -14)
	err := s.db.Where("patient_id = ? AND date >= ?", patientID, cutoff).
		Order("date DESC").
		Limit(14).
		Find(&signals).Error

	if err != nil || len(signals) == 0 {
		return "UNKNOWN", 0.0
	}

	highCarbCount := 0
	for _, sig := range signals {
		if strings.EqualFold(sig.CarbEstimateCategory, "HIGH") {
			highCarbCount++
		}
	}

	// Heuristic: high carb correlates with high sodium in Indian diet
	ratio := float64(highCarbCount) / float64(len(signals))
	switch {
	case ratio >= 0.60:
		return "HIGH", 0.70 // high salt reduction potential
	case ratio >= 0.30:
		return "MODERATE", 0.40
	default:
		return "LOW", 0.15
	}
}

// dataQualityWeight returns a numeric weight for quality-weighted aggregation.
func dataQualityWeight(dq models.DataQuality) float64 {
	switch dq {
	case models.DataQualityHigh:
		return 1.0
	case models.DataQualityModerate:
		return 0.7
	case models.DataQualityLow:
		return 0.4
	default:
		return 0.4
	}
}

// barrierToReason maps BarrierCode to AdherenceReason for HTN routing.
func barrierToReason(b models.BarrierCode) models.AdherenceReason {
	switch b {
	case models.BarrierCost:
		return models.ReasonCost
	case models.BarrierSideEffects:
		return models.ReasonSideEffect
	case models.BarrierForgetfulness:
		return models.ReasonForgot
	case models.BarrierAccess:
		return models.ReasonSupply
	default:
		return models.ReasonUnknown
	}
}

func classifyDataQuality(responded, total int) models.DataQuality {
	if total == 0 {
		return models.DataQuality("LOW")
	}
	rate := float64(responded) / float64(total)
	switch {
	case rate >= 0.80:
		return models.DataQuality("HIGH")
	case rate >= 0.50:
		return models.DataQuality("MODERATE")
	default:
		return models.DataQuality("LOW")
	}
}
