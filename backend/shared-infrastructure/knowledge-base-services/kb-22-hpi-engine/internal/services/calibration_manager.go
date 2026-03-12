package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"kb-22-hpi-engine/internal/cache"
	"kb-22-hpi-engine/internal/database"
	"kb-22-hpi-engine/internal/metrics"
	"kb-22-hpi-engine/internal/models"
)

// CalibrationManager implements concordance tracking and LR calibration
// for the HPI engine (Gaps D01, E03). It processes clinician adjudication
// feedback against DifferentialSnapshots to compute calibration metrics
// per node, stratum, and CKD substage.
type CalibrationManager struct {
	db      *database.Database
	cache   *cache.CacheClient
	log     *zap.Logger
	metrics *metrics.Collector
}

// NewCalibrationManager creates a new CalibrationManager.
func NewCalibrationManager(
	db *database.Database,
	cacheClient *cache.CacheClient,
	log *zap.Logger,
	m *metrics.Collector,
) *CalibrationManager {
	return &CalibrationManager{
		db:      db,
		cache:   cacheClient,
		log:     log,
		metrics: m,
	}
}

// SubmitFeedback creates a CalibrationRecord from a DifferentialSnapshot
// and the clinician's confirmed diagnosis. This is the core adjudication
// entry point called by POST /calibration/feedback.
//
// Steps:
//  1. Load the DifferentialSnapshot by snapshot_id
//  2. Load the associated HPISession to get node_id, stratum, ckd_substage
//  3. Parse the ranked differentials to extract engine top-1 and top-3
//  4. Compute concordance (top-1 and top-3 match)
//  5. Load session answers for per-question LR estimation (Tier 2)
//  6. Persist the CalibrationRecord
//  7. Update the DifferentialSnapshot with adjudication result
//  8. Invalidate cached calibration status
//  9. Update Prometheus concordance gauge
func (cm *CalibrationManager) SubmitFeedback(
	ctx context.Context,
	feedback models.AdjudicationFeedback,
) (*models.CalibrationRecord, error) {
	// Step 1: load snapshot
	var snapshot models.DifferentialSnapshot
	if err := cm.db.DB.WithContext(ctx).
		Where("snapshot_id = ?", feedback.SnapshotID).
		First(&snapshot).Error; err != nil {
		return nil, fmt.Errorf("snapshot not found: %w", err)
	}

	// Step 2: load session for metadata
	var session models.HPISession
	if err := cm.db.DB.WithContext(ctx).
		Where("session_id = ?", snapshot.SessionID).
		First(&session).Error; err != nil {
		return nil, fmt.Errorf("session not found for snapshot: %w", err)
	}

	// Step 3: parse ranked differentials
	var rankedDiffs []models.DifferentialEntry
	if err := json.Unmarshal(snapshot.RankedDifferentials, &rankedDiffs); err != nil {
		return nil, fmt.Errorf("failed to parse ranked differentials: %w", err)
	}

	if len(rankedDiffs) == 0 {
		return nil, fmt.Errorf("snapshot has no ranked differentials")
	}

	engineTop1 := rankedDiffs[0].DifferentialID
	top3Limit := 3
	if len(rankedDiffs) < top3Limit {
		top3Limit = len(rankedDiffs)
	}
	engineTop3 := make([]string, top3Limit)
	for i := 0; i < top3Limit; i++ {
		engineTop3[i] = rankedDiffs[i].DifferentialID
	}

	// Step 4: compute concordance
	concordantTop1 := engineTop1 == feedback.ConfirmedDiagnosis
	concordantTop3 := false
	for _, diffID := range engineTop3 {
		if diffID == feedback.ConfirmedDiagnosis {
			concordantTop3 = true
			break
		}
	}

	// Step 5: load session answers for Tier 2 per-question LR estimation
	var answers []models.SessionAnswer
	cm.db.DB.WithContext(ctx).
		Where("session_id = ?", snapshot.SessionID).
		Order("answered_at ASC").
		Find(&answers)

	questionAnswers := make(map[string]string, len(answers))
	for _, a := range answers {
		questionAnswers[a.QuestionID] = a.AnswerValue
	}

	qaJSON, _ := json.Marshal(questionAnswers)

	// Step 6: create calibration record
	record := models.CalibrationRecord{
		RecordID:           uuid.New(),
		SnapshotID:         feedback.SnapshotID,
		NodeID:             session.NodeID,
		StratumLabel:       session.StratumLabel,
		CKDSubstage:        session.CKDSubstage,
		ConfirmedDiagnosis: feedback.ConfirmedDiagnosis,
		EngineTop1:         engineTop1,
		EngineTop3:         engineTop3,
		ConcordantTop1:     concordantTop1,
		ConcordantTop3:     concordantTop3,
		QuestionAnswers:    qaJSON,
		AdjudicatedAt:      time.Now(),
	}

	if err := cm.db.DB.WithContext(ctx).Create(&record).Error; err != nil {
		return nil, fmt.Errorf("failed to create calibration record: %w", err)
	}

	// Step 7: update snapshot with adjudication
	cm.db.DB.WithContext(ctx).
		Model(&snapshot).
		Updates(map[string]interface{}{
			"clinician_adjudication": feedback.ConfirmedDiagnosis,
			"concordant":            concordantTop1,
		})

	// Step 8: invalidate cached calibration status
	cacheKey := session.NodeID
	if err := cm.cache.Delete(cache.CalibrationPrefix + cacheKey); err != nil {
		cm.log.Warn("failed to invalidate calibration cache",
			zap.String("node_id", session.NodeID),
			zap.Error(err),
		)
	}

	// Step 9: update Prometheus gauge
	cm.metrics.CalibrationConcordance.WithLabelValues(session.NodeID, session.StratumLabel).Set(
		boolToFloat(concordantTop1),
	)

	cm.log.Info("calibration feedback recorded",
		zap.String("record_id", record.RecordID.String()),
		zap.String("node_id", session.NodeID),
		zap.String("stratum", session.StratumLabel),
		zap.String("confirmed", feedback.ConfirmedDiagnosis),
		zap.String("engine_top1", engineTop1),
		zap.Bool("concordant_top1", concordantTop1),
		zap.Bool("concordant_top3", concordantTop3),
	)

	return &record, nil
}

// GetStatus computes the concordance metrics for a given node, optionally
// filtered by stratum and CKD substage (F-06).
//
// Query parameters:
//   - stratum: filter by stratum_label (empty = all strata)
//   - ckdSubstage: filter by ckd_substage (empty = all substages)
//
// Returns cached results if available (30-minute TTL).
func (cm *CalibrationManager) GetStatus(
	ctx context.Context,
	nodeID string,
	stratum string,
	ckdSubstage string,
) (*models.CalibrationStatus, error) {
	// Build cache key with all filter dimensions
	cacheKey := fmt.Sprintf("%s:%s:%s", nodeID, stratum, ckdSubstage)

	// Try cache first
	var cached models.CalibrationStatus
	if err := cm.cache.GetCalibrationStatus(cacheKey, &cached); err == nil {
		return &cached, nil
	}

	// Query database
	query := cm.db.DB.WithContext(ctx).
		Model(&models.CalibrationRecord{}).
		Where("node_id = ?", nodeID)

	if stratum != "" {
		query = query.Where("stratum_label = ?", stratum)
	}
	if ckdSubstage != "" {
		query = query.Where("ckd_substage = ?", ckdSubstage)
	}

	var totalAdjudicated int64
	var concordantTop1Count int64
	var concordantTop3Count int64

	// Total count
	if err := query.Count(&totalAdjudicated).Error; err != nil {
		return nil, fmt.Errorf("count adjudicated: %w", err)
	}

	// Concordant top-1 count
	if err := query.Where("concordant_top1 = ?", true).Count(&concordantTop1Count).Error; err != nil {
		return nil, fmt.Errorf("count concordant top1: %w", err)
	}

	// Reset query for top-3 (re-apply base filters)
	top3Query := cm.db.DB.WithContext(ctx).
		Model(&models.CalibrationRecord{}).
		Where("node_id = ?", nodeID)
	if stratum != "" {
		top3Query = top3Query.Where("stratum_label = ?", stratum)
	}
	if ckdSubstage != "" {
		top3Query = top3Query.Where("ckd_substage = ?", ckdSubstage)
	}
	if err := top3Query.Where("concordant_top3 = ?", true).Count(&concordantTop3Count).Error; err != nil {
		return nil, fmt.Errorf("count concordant top3: %w", err)
	}

	// Compute rates
	var top1Rate, top3Rate float64
	if totalAdjudicated > 0 {
		top1Rate = float64(concordantTop1Count) / float64(totalAdjudicated)
		top3Rate = float64(concordantTop3Count) / float64(totalAdjudicated)
	}

	// Determine calibration tier based on sample size
	tier := determineCalibrationTier(int(totalAdjudicated))

	status := &models.CalibrationStatus{
		NodeID:           nodeID,
		StratumLabel:     stratum,
		CKDSubstage:      ckdSubstage,
		TotalAdjudicated: int(totalAdjudicated),
		ConcordantTop1:   int(concordantTop1Count),
		ConcordantTop3:   int(concordantTop3Count),
		Top1Rate:         top1Rate,
		Top3Rate:         top3Rate,
		CalibrationTier:  tier,
	}

	// Cache the result
	if err := cm.cache.SetCalibrationStatus(cacheKey, status); err != nil {
		cm.log.Warn("failed to cache calibration status",
			zap.String("node_id", nodeID),
			zap.Error(err),
		)
	}

	cm.log.Debug("calibration status computed",
		zap.String("node_id", nodeID),
		zap.String("stratum", stratum),
		zap.Int64("total", totalAdjudicated),
		zap.Float64("top1_rate", top1Rate),
		zap.Float64("top3_rate", top3Rate),
		zap.String("tier", tier),
	)

	return status, nil
}

// ImportGolden bulk-imports golden dataset cases for synthetic concordance
// measurement. Each case is converted to a CalibrationRecord with
// pre-computed concordance values.
//
// Returns the number of successfully imported cases.
func (cm *CalibrationManager) ImportGolden(
	ctx context.Context,
	dataset models.GoldenDatasetImport,
) (int, error) {
	imported := 0

	for _, gc := range dataset.Cases {
		// Compute concordance
		concordantTop1 := gc.EngineTop1 == gc.ConfirmedDiagnosis
		concordantTop3 := false
		for _, diffID := range gc.EngineTop3 {
			if diffID == gc.ConfirmedDiagnosis {
				concordantTop3 = true
				break
			}
		}

		qaJSON, err := json.Marshal(gc.QuestionAnswers)
		if err != nil {
			cm.log.Warn("failed to marshal golden case answers, skipping",
				zap.String("node_id", gc.NodeID),
				zap.Error(err),
			)
			continue
		}

		record := models.CalibrationRecord{
			RecordID:           uuid.New(),
			SnapshotID:         uuid.New(), // synthetic snapshot ID for golden cases
			NodeID:             gc.NodeID,
			StratumLabel:       gc.StratumLabel,
			CKDSubstage:        gc.CKDSubstage,
			ConfirmedDiagnosis: gc.ConfirmedDiagnosis,
			EngineTop1:         gc.EngineTop1,
			EngineTop3:         gc.EngineTop3,
			ConcordantTop1:     concordantTop1,
			ConcordantTop3:     concordantTop3,
			QuestionAnswers:    qaJSON,
			AdjudicatedAt:      time.Now(),
		}

		if err := cm.db.DB.WithContext(ctx).Create(&record).Error; err != nil {
			cm.log.Warn("failed to import golden case",
				zap.String("node_id", gc.NodeID),
				zap.Error(err),
			)
			continue
		}

		imported++
	}

	// Invalidate all calibration caches after bulk import
	if err := cm.cache.DeletePattern(cache.CalibrationPrefix + "*"); err != nil {
		cm.log.Warn("failed to invalidate calibration caches after import",
			zap.Error(err),
		)
	}

	cm.log.Info("golden dataset import complete",
		zap.Int("imported", imported),
		zap.Int("total", len(dataset.Cases)),
	)

	return imported, nil
}

// determineCalibrationTier returns the calibration tier based on the number
// of adjudicated cases. Tiers determine which LR source takes precedence:
//   - EXPERT_PANEL:   < 30 cases  — Tier A: rely on expert-authored LRs (Month 0-6)
//   - BLENDED:        30-199 cases — Tier B: blend expert and empirical LRs (Month 6-18)
//   - DATA_DRIVEN:    200+ cases  — Tier C: empirical LRs via logistic regression (Month 18+)
func determineCalibrationTier(totalAdjudicated int) string {
	switch {
	case totalAdjudicated >= 200:
		return "DATA_DRIVEN"
	case totalAdjudicated >= 30:
		return "BLENDED"
	default:
		return "EXPERT_PANEL"
	}
}

// boolToFloat converts a boolean to a float64 for Prometheus gauge updates.
func boolToFloat(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}
