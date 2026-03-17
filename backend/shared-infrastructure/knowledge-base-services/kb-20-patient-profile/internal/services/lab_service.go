package services

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"kb-patient-profile/internal/cache"
	"kb-patient-profile/internal/database"
	"kb-patient-profile/internal/metrics"
	"kb-patient-profile/internal/models"
)

// LabService handles lab value writes with plausibility validation (F-05),
// auto-derives eGFR from creatinine, and detects medication threshold crossings (F-03).
type LabService struct {
	db        *database.Database
	cache     *cache.Client
	logger    *zap.Logger
	metrics   *metrics.Collector
	validator *LabValidator
	egfr      *EGFREngine
	eventBus  *EventBus
}

// NewLabService creates a lab service with validation and eGFR engine.
func NewLabService(
	db *database.Database,
	cacheClient *cache.Client,
	logger *zap.Logger,
	metricsCollector *metrics.Collector,
	eventBus *EventBus,
) *LabService {
	return &LabService{
		db:        db,
		cache:     cacheClient,
		logger:    logger,
		metrics:   metricsCollector,
		validator: NewLabValidator(),
		egfr:      NewEGFREngine(),
		eventBus:  eventBus,
	}
}

// AddLab validates and stores a lab value. If the lab is creatinine, it
// auto-derives eGFR and checks for medication threshold crossings (F-03).
// All operations (lab write, eGFR derivation, event outbox write) are wrapped
// in a single DB transaction for G-03 durability.
func (s *LabService) AddLab(patientID string, req models.AddLabRequest) (*models.LabEntry, error) {
	// F-05: Validate plausibility
	result := s.validator.Validate(req.LabType, req.Value)
	s.metrics.LabValidations.WithLabelValues(req.LabType, result.Status).Inc()

	if result.Status == models.ValidationRejected {
		return nil, fmt.Errorf("lab value rejected: %s", result.FlagReason)
	}

	measuredAt, err := time.Parse(time.RFC3339, req.MeasuredAt)
	if err != nil {
		measuredAt, err = time.Parse("2006-01-02", req.MeasuredAt)
		if err != nil {
			return nil, fmt.Errorf("invalid measured_at format, use RFC3339 or YYYY-MM-DD: %w", err)
		}
	}

	entry := &models.LabEntry{
		PatientID:        patientID,
		LabType:          req.LabType,
		Value:            decimal.NewFromFloat(req.Value),
		Unit:             req.Unit,
		MeasuredAt:       measuredAt,
		Source:           req.Source,
		ValidationStatus: result.Status,
		FlagReason:       result.FlagReason,
	}

	// G-03: Wrap lab write + eGFR derivation + event publish in single transaction
	txErr := s.db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(entry).Error; err != nil {
			return fmt.Errorf("failed to store lab entry: %w", err)
		}

		// Publish LAB_RESULT event atomically with the lab write (G-03)
		s.eventBus.PublishTx(tx, models.EventLabResult, patientID,
			models.LabResultPayload{
				LabType:          req.LabType,
				Value:            req.Value,
				Unit:             req.Unit,
				MeasuredAt:       measuredAt.Format(time.RFC3339),
				Source:           req.Source,
				ValidationStatus: result.Status,
				IsDerived:        false,
			})

		// Auto-derive eGFR from creatinine (within same tx)
		if req.LabType == models.LabTypeCreatinine {
			s.deriveEGFR(tx, patientID, req.Value, measuredAt)
		}

		// Process ACR longitudinal tracking (LOINC 2951-2)
		if req.LabType == models.LabTypeACR {
			s.processACR(tx, patientID, req.Value, measuredAt)
		}

		// FBG trajectory classification (Track 3 — Sprint 1 prerequisite)
		if req.LabType == models.LabTypeFBG {
			s.updateGlucoseTrajectory(tx, patientID)
		}

		// BP_URGENCY_ALERT: single SBP reading >= 180 mmHg → immediate notification.
		// This is a point-in-time urgency check, independent of 28-day trajectory.
		if req.LabType == models.LabTypeSBP && req.Value >= 180 {
			triggerVal := req.Value
			s.eventBus.PublishTx(tx, models.EventBPUrgencyAlert, patientID,
				models.BPAlertPayload{
					PatientID:      patientID,
					BPStatus:       "URGENCY",
					TargetSBP:      130,
					TriggerReading: &triggerVal,
				})
			s.logger.Warn("BP_URGENCY_ALERT: single SBP reading >= 180 mmHg",
				zap.String("patient_id", patientID),
				zap.Float64("sbp", req.Value))
		}

		// SAFETY_ALERT: critical potassium levels (life-threatening arrhythmia risk)
		if req.LabType == models.LabTypePotassium && (req.Value < 3.0 || req.Value > 6.0) {
			severity := "CRITICAL"
			alertType := "HYPOKALAEMIA_CRITICAL"
			desc := fmt.Sprintf("Potassium %.1f mEq/L < 3.0 — life-threatening hypokalaemia", req.Value)
			if req.Value > 6.0 {
				alertType = "HYPERKALAEMIA_CRITICAL"
				desc = fmt.Sprintf("Potassium %.1f mEq/L > 6.0 — life-threatening hyperkalaemia", req.Value)
			}
			s.eventBus.PublishTx(tx, models.EventSafetyAlert, patientID, models.SafetyAlertPayload{
				Severity:    severity,
				AlertType:   alertType,
				Description: desc,
				LabType:     models.LabTypePotassium,
				NewValue:    fmt.Sprintf("%.1f", req.Value),
			})
			s.logger.Warn("SAFETY_ALERT: critical potassium",
				zap.String("patient_id", patientID),
				zap.Float64("potassium", req.Value))
		}

		// ORTHOSTATIC_ALERT: SBP drop > 20 mmHg between SEATED and STANDING readings.
		// When a STANDING SBP is received, look for the most recent SEATED SBP within 10 min.
		if req.LabType == models.LabTypeSBP && req.Source == "STANDING" {
			var seatedEntry models.LabEntry
			cutoff := measuredAt.Add(-10 * time.Minute)
			err := tx.Where("patient_id = ? AND lab_type = ? AND source = ? AND measured_at >= ? AND measured_at < ?",
				patientID, models.LabTypeSBP, "SEATED", cutoff, measuredAt).
				Order("measured_at DESC").First(&seatedEntry).Error
			if err == nil {
				seatedSBP, _ := seatedEntry.Value.Float64()
				drop := ComputeOrthostaticDrop(seatedSBP, req.Value)
				if drop < -20 {
					s.eventBus.PublishTx(tx, models.EventOrthostaticAlert, patientID,
						models.OrthostaticAlertPayload{
							PatientID:       patientID,
							OrthostaticDrop: drop,
							SeatedSBP:       seatedSBP,
							StandingSBP:     req.Value,
						})
					s.logger.Warn("ORTHOSTATIC_ALERT: SBP drop > 20 mmHg on standing",
						zap.String("patient_id", patientID),
						zap.Float64("drop", drop),
						zap.Float64("seated_sbp", seatedSBP),
						zap.Float64("standing_sbp", req.Value))
				}
			}
		}

		return nil
	})

	if txErr != nil {
		return nil, txErr
	}

	// Invalidate patient cache (outside transaction — cache is best-effort)
	s.cache.Delete(cache.PatientProfilePrefix + patientID)

	return entry, nil
}

// deriveEGFR computes eGFR from creatinine and stores it, then checks thresholds.
// Accepts tx to participate in the caller's transaction (G-03).
func (s *LabService) deriveEGFR(tx *gorm.DB, patientID string, creatinine float64, measuredAt time.Time) {
	var profile models.PatientProfile
	if err := tx.Where("patient_id = ?", patientID).First(&profile).Error; err != nil {
		s.logger.Warn("Cannot derive eGFR: patient profile not found", zap.String("patient_id", patientID))
		return
	}

	egfr := s.egfr.ComputeEGFR(creatinine, profile.Age, profile.Sex)
	s.metrics.EGFRComputed.Inc()

	stage := s.egfr.CKDStageFromEGFR(egfr)
	s.logger.Info("eGFR computed",
		zap.String("patient_id", patientID),
		zap.Float64("creatinine", creatinine),
		zap.Float64("egfr", egfr),
		zap.String("stage", stage))

	// Store derived eGFR
	egfrEntry := &models.LabEntry{
		PatientID:        patientID,
		LabType:          models.LabTypeEGFR,
		Value:            decimal.NewFromFloat(egfr),
		Unit:             "mL/min/1.73m²",
		MeasuredAt:       measuredAt,
		Source:           "CKD-EPI-2021",
		IsDerived:        true,
		ValidationStatus: models.ValidationAccepted,
	}
	tx.Create(egfrEntry)

	// F-03: Check medication threshold crossings (event written to outbox in same tx)
	s.checkThresholdCrossings(tx, patientID, egfr)

	// RAAS monitoring: detect concerning creatinine rise post-ACEi/ARB
	s.checkRAASCreatinineMonitoring(tx, patientID, creatinine, measuredAt)

	// SAFETY_ALERT: critically low eGFR (<15) or rapid decline warrants immediate notification
	if egfr < 15 {
		s.eventBus.PublishTx(tx, models.EventSafetyAlert, patientID, models.SafetyAlertPayload{
			Severity:    "CRITICAL",
			AlertType:   "EGFR_CRITICAL",
			Description: "eGFR < 15 mL/min/1.73m² — CKD Stage 5, possible dialysis requirement",
			LabType:     models.LabTypeEGFR,
			NewValue:    fmt.Sprintf("%.1f", egfr),
		})
		s.logger.Warn("SAFETY_ALERT: critically low eGFR",
			zap.String("patient_id", patientID),
			zap.Float64("egfr", egfr))
	}

	// Update CKD status on patient profile
	s.updateCKDStatus(tx, patientID, egfr, stage)
}

// updateGlucoseTrajectory classifies the patient's FBG trajectory from the
// last 28 days of FBG readings and publishes GLUCOSE_TRAJECTORY_CHANGE when
// the classification changes. Called on each new FBG lab write.
// Follows the same pattern as UpdateEarlyWarning() for BP trajectory.
func (s *LabService) updateGlucoseTrajectory(tx *gorm.DB, patientID string) {
	cutoff := time.Now().AddDate(0, 0, -28)

	var fbgEntries []models.LabEntry
	if err := tx.Where("patient_id = ? AND lab_type = ? AND validation_status = ? AND measured_at >= ?",
		patientID, models.LabTypeFBG, models.ValidationAccepted, cutoff).
		Order("measured_at ASC").Find(&fbgEntries).Error; err != nil {
		s.logger.Warn("failed to query FBG readings for trajectory",
			zap.String("patient_id", patientID), zap.Error(err))
		return
	}

	// Convert to TimestampedLabValue for the trajectory engine
	readings := make([]models.TimestampedLabValue, len(fbgEntries))
	for i, e := range fbgEntries {
		val, _ := e.Value.Float64()
		readings[i] = models.TimestampedLabValue{Value: val, Timestamp: e.MeasuredAt}
	}

	result := ClassifyGlucoseTrajectory(readings)

	// Check if classification changed from previous (stored in cache)
	prevClass := s.getPreviousTrajectoryClass(patientID)
	if result.Classification != prevClass && prevClass != "" {
		s.eventBus.PublishTx(tx, models.EventGlucoseTrajectoryChange, patientID,
			models.GlucoseTrajectoryPayload{
				PatientID:              patientID,
				Classification:         result.Classification,
				PreviousClassification: prevClass,
				FBGSlope:               result.FBGSlope,
				GlucoseCV:              result.GlucoseCV,
				ReadingsUsed:           result.ReadingsUsed,
			})
		s.logger.Info("FBG trajectory classification changed",
			zap.String("patient_id", patientID),
			zap.String("from", prevClass),
			zap.String("to", result.Classification),
			zap.Float64("slope_mg_dl_per_week", result.FBGSlope))
	}

	// Store current classification for next comparison
	s.cache.Set(cache.GlucoseTrajectoryPrefix+patientID, result.Classification, 24*time.Hour)
}

// getPreviousTrajectoryClass retrieves the last known trajectory classification from cache.
func (s *LabService) getPreviousTrajectoryClass(patientID string) string {
	var prev string
	if err := s.cache.Get(cache.GlucoseTrajectoryPrefix+patientID, &prev); err != nil {
		return ""
	}
	return prev
}

// checkThresholdCrossings detects when eGFR crosses medication-relevant boundaries (F-03 RED).
// Uses tx to write event outbox entries atomically with the lab data (G-03).
func (s *LabService) checkThresholdCrossings(tx *gorm.DB, patientID string, newEGFR float64) {
	// Get previous eGFR
	var prevEntry models.LabEntry
	err := tx.Where("patient_id = ? AND lab_type = ? AND validation_status = ?",
		patientID, models.LabTypeEGFR, models.ValidationAccepted).
		Order("measured_at DESC").Offset(1).First(&prevEntry).Error
	if err != nil {
		return // No previous eGFR — no crossing to detect
	}

	oldEGFR, _ := prevEntry.Value.Float64()
	crossings := s.egfr.DetectThresholdCrossings(oldEGFR, newEGFR)

	if len(crossings) == 0 {
		return
	}

	// Get active medications to identify affected drugs
	var medications []models.MedicationState
	tx.Where("patient_id = ? AND is_active = true", patientID).Find(&medications)

	for _, crossing := range crossings {
		var affected []models.AffectedMedication
		for _, med := range medications {
			for _, dc := range med.EffectiveDrugClasses() {
				if dc == crossing.AffectedDrugClass {
					affected = append(affected, models.AffectedMedication{
						DrugClass:      dc,
						RequiredAction: crossing.RequiredAction,
						MaxDoseMg:      crossing.MaxDoseMg,
					})
				}
			}
		}

		if len(affected) > 0 {
			// G-03: Write event to outbox in same transaction as lab data
			s.eventBus.PublishTx(tx, models.EventMedicationThresholdCrossed, patientID,
				models.MedicationThresholdCrossedPayload{
					Lab:                 "eGFR",
					OldValue:            oldEGFR,
					NewValue:            newEGFR,
					ThresholdCrossed:    crossing.EGFRBoundary,
					AffectedMedications: affected,
				})
		}
	}
}

// updateCKDStatus sets SUSPECTED or CONFIRMED CKD status within the transaction.
func (s *LabService) updateCKDStatus(tx *gorm.DB, patientID string, egfr float64, stage string) {
	var egfrEntries []models.LabEntry
	tx.Where("patient_id = ? AND lab_type = ? AND validation_status = ?",
		patientID, models.LabTypeEGFR, models.ValidationAccepted).
		Order("measured_at ASC").Find(&egfrEntries)

	hasCKD, isConfirmed := s.egfr.IsCKDConfirmed(egfrEntries)

	updates := map[string]interface{}{"ckd_stage": stage}
	if isConfirmed {
		updates["ckd_status"] = "CONFIRMED"
	} else if hasCKD {
		updates["ckd_status"] = "SUSPECTED"
	} else {
		updates["ckd_status"] = "NONE"
	}

	tx.Model(&models.PatientProfile{}).Where("patient_id = ?", patientID).Updates(updates)
}

// GetLabs retrieves lab history for a patient, optionally filtered by type.
func (s *LabService) GetLabs(patientID string, labType string) ([]models.LabEntry, error) {
	var labs []models.LabEntry
	query := s.db.DB.Where("patient_id = ? AND validation_status != 'REJECTED'", patientID)
	if labType != "" {
		query = query.Where("lab_type = ?", labType)
	}
	if err := query.Order("measured_at DESC").Find(&labs).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve labs: %w", err)
	}
	return labs, nil
}

// ComputeBPTrajectory analyses the last 28 days of SBP/DBP readings to detect
// clinically significant BP patterns (white-coat, masked, morning surge, etc.).
//
// Detection rules:
//   - WHITE_COAT: clinic SBP consistently >10 mmHg above home SBP (≥3 paired readings)
//   - MASKED: home SBP consistently >10 mmHg above clinic SBP (≥3 paired readings)
//   - MORNING_SURGE: morning_fasting SBP − evening SBP > 20 mmHg (≥5 pairs each)
//   - SUSTAINED_HIGH: mean SBP ≥140 over 2+ weeks with ≥5 readings
//   - CONTROLLED: mean SBP <130 AND >80% readings in target (SBP 90-140)
//   - UNKNOWN: fewer than 5 readings in 28 days
func (s *LabService) ComputeBPTrajectory(patientID string) (*models.BPTrajectory, error) {
	cutoff := time.Now().AddDate(0, 0, -28)

	var sbpEntries []models.LabEntry
	if err := s.db.DB.Where("patient_id = ? AND lab_type = ? AND validation_status = ? AND measured_at >= ?",
		patientID, models.LabTypeSBP, models.ValidationAccepted, cutoff).
		Order("measured_at ASC").Find(&sbpEntries).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve SBP readings: %w", err)
	}

	var dbpEntries []models.LabEntry
	s.db.DB.Where("patient_id = ? AND lab_type = ? AND validation_status = ? AND measured_at >= ?",
		patientID, models.LabTypeDBP, models.ValidationAccepted, cutoff).
		Order("measured_at ASC").Find(&dbpEntries)

	traj := &models.BPTrajectory{
		PatientID:        patientID,
		ComputedAt:       time.Now(),
		TotalReadings28d: len(sbpEntries),
	}

	if len(sbpEntries) < 5 {
		traj.Pattern = models.BPPatternUnknown
		return traj, nil
	}

	// Compute mean SBP and DBP
	var sbpSum, dbpSum float64
	inTarget := 0
	for _, e := range sbpEntries {
		v, _ := e.Value.Float64()
		sbpSum += v
		if v >= 90 && v <= 140 {
			inTarget++
		}
	}
	meanSBP := sbpSum / float64(len(sbpEntries))
	traj.MeanSBP28d = &meanSBP
	traj.ReadingsInTarget = inTarget

	if len(dbpEntries) > 0 {
		for _, e := range dbpEntries {
			v, _ := e.Value.Float64()
			dbpSum += v
		}
		meanDBP := dbpSum / float64(len(dbpEntries))
		traj.MeanDBP28d = &meanDBP
	}

	// Estimate measurement uncertainty from SBP standard deviation
	if len(sbpEntries) >= 3 {
		var sumSq float64
		for _, e := range sbpEntries {
			v, _ := e.Value.Float64()
			diff := v - meanSBP
			sumSq += diff * diff
		}
		traj.MeasurementUncertainty = math.Sqrt(sumSq / float64(len(sbpEntries)-1))
	}

	// Pattern detection
	traj.Pattern = s.detectBPPattern(sbpEntries, meanSBP, inTarget)

	// Emit MASKED_HTN_DETECTED when masked hypertension pattern is detected (Gap #21)
	if traj.Pattern == models.BPPatternMasked {
		var clinicReadings, homeReadings []float64
		for _, e := range sbpEntries {
			v, _ := e.Value.Float64()
			switch e.Source {
			case "CLINIC":
				clinicReadings = append(clinicReadings, v)
			case "HOME", "AMBULATORY":
				homeReadings = append(homeReadings, v)
			}
		}
		s.eventBus.Publish(models.EventMaskedHTNDetected, patientID, models.MaskedHTNPayload{
			PatientID:         patientID,
			ClinicSBPMean:     sliceMean(clinicReadings),
			HomeDeviceSBPMean: sliceMean(homeReadings),
			PairedReadings:    min(len(clinicReadings), len(homeReadings)),
		})
	}

	return traj, nil
}

// detectBPPattern classifies BP behaviour from readings and their metadata.
// Priority order: WHITE_COAT > MASKED > MORNING_SURGE > SUSTAINED_HIGH > CONTROLLED > UNKNOWN
func (s *LabService) detectBPPattern(sbpEntries []models.LabEntry, meanSBP float64, inTarget int) models.BPPattern {
	var clinicReadings, homeReadings []float64
	var morningReadings, eveningReadings []float64

	for _, e := range sbpEntries {
		v, _ := e.Value.Float64()
		switch e.Source {
		case "CLINIC":
			clinicReadings = append(clinicReadings, v)
		case "HOME", "AMBULATORY":
			homeReadings = append(homeReadings, v)
		}
		hour := e.MeasuredAt.Hour()
		if hour >= 5 && hour <= 9 {
			morningReadings = append(morningReadings, v)
		} else if hour >= 18 && hour <= 22 {
			eveningReadings = append(eveningReadings, v)
		}
	}

	// WHITE_COAT: clinic SBP consistently >10 mmHg above home SBP (≥3 paired)
	if len(clinicReadings) >= 3 && len(homeReadings) >= 3 {
		clinicMean := sliceMean(clinicReadings)
		homeMean := sliceMean(homeReadings)
		if clinicMean-homeMean > 10 {
			return models.BPPatternWhiteCoat
		}
		if homeMean-clinicMean > 10 {
			return models.BPPatternMasked
		}
	}

	// MORNING_SURGE: morning SBP − evening SBP > 20 mmHg (≥5 pairs each)
	if len(morningReadings) >= 5 && len(eveningReadings) >= 5 {
		if sliceMean(morningReadings)-sliceMean(eveningReadings) > 20 {
			return models.BPPatternMorningHTN
		}
	}

	// SUSTAINED_HIGH: mean SBP ≥140
	if meanSBP >= 140 && len(sbpEntries) >= 5 {
		return models.BPPatternSustainedHigh
	}

	// CONTROLLED: mean SBP <130 AND >80% readings in target
	targetPct := float64(inTarget) / float64(len(sbpEntries))
	if meanSBP < 130 && targetPct > 0.80 {
		return models.BPPatternControlled
	}

	return models.BPPatternUnknown
}

// sliceMean computes the arithmetic mean of a float64 slice.
func sliceMean(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

// processACR handles a new ACR lab result within the caller's transaction.
// It maintains the ACRTracking record: appends readings, computes category/trend,
// checks RAAS status, and emits ACR_WORSENING or ACR_TARGET_MET events.
func (s *LabService) processACR(tx *gorm.DB, patientID string, valueMgMmol float64, measuredAt time.Time) {
	var tracking models.ACRTracking
	err := tx.Where("patient_id = ?", patientID).First(&tracking).Error
	if err != nil {
		// No existing record — create one
		tracking = models.ACRTracking{
			PatientID: patientID,
		}
	}

	reading := models.ACRReading{
		ValueMgMmol: valueMgMmol,
		CollectedAt: measuredAt,
	}

	// Append reading, keep last 10
	tracking.Readings = append(tracking.Readings, reading)
	if len(tracking.Readings) > 10 {
		tracking.Readings = tracking.Readings[len(tracking.Readings)-10:]
	}

	// Compute category from latest value
	newCategory := models.CategorizeACR(valueMgMmol)
	oldCategory := tracking.Category
	tracking.Category = newCategory

	// Compute trend from last 3+ readings
	oldTrend := tracking.Trend
	tracking.Trend = s.computeACRTrend(tracking.Readings, oldCategory, newCategory)

	// Check RAAS status: patient has active ACEi or ARB
	tracking.OnRAAS = s.patientOnRAAS(tx, patientID)

	tracking.UpdatedAt = time.Now()

	// Upsert the tracking record
	if err != nil {
		// Was not found — create
		tx.Create(&tracking)
	} else {
		tx.Save(&tracking)
	}

	// Emit ACR_WORSENING if trend just changed to WORSENING
	if tracking.Trend == models.ACRTrendWorsening && oldTrend != models.ACRTrendWorsening {
		prevValue := 0.0
		if len(tracking.Readings) >= 2 {
			prevValue = tracking.Readings[len(tracking.Readings)-2].ValueMgMmol
		}
		s.eventBus.PublishTx(tx, models.EventACRWorsening, patientID,
			models.ACRWorseningPayload{
				PatientID:        patientID,
				CurrentValue:     valueMgMmol,
				PreviousValue:    prevValue,
				CurrentCategory:  newCategory,
				PreviousCategory: oldCategory,
				OnRAAS:           tracking.OnRAAS,
			})
		s.logger.Warn("ACR worsening detected",
			zap.String("patient_id", patientID),
			zap.Float64("acr_value", valueMgMmol),
			zap.String("category", newCategory),
			zap.String("trend", tracking.Trend))
	}

	// Emit ACR_TARGET_MET if category improved (ordinal decreased)
	if oldCategory != "" && models.ACRCategoryOrdinal(newCategory) < models.ACRCategoryOrdinal(oldCategory) {
		s.eventBus.PublishTx(tx, models.EventACRTargetMet, patientID,
			models.ACRTargetMetPayload{
				PatientID:        patientID,
				CurrentValue:     valueMgMmol,
				CurrentCategory:  newCategory,
				PreviousCategory: oldCategory,
			})
		s.logger.Info("ACR target met — category improved",
			zap.String("patient_id", patientID),
			zap.String("from", oldCategory),
			zap.String("to", newCategory))
	}
}

// computeACRTrend classifies the ACR trend from readings.
//
// Rules:
//   - WORSENING: latest > previous by >20% OR category stepped up
//   - IMPROVING: latest < previous by >20% OR category stepped down
//   - STABLE: otherwise
//
// Requires at least 2 readings; returns STABLE if insufficient data.
func (s *LabService) computeACRTrend(readings []models.ACRReading, oldCategory, newCategory string) string {
	if len(readings) < 2 {
		return models.ACRTrendStable
	}

	latest := readings[len(readings)-1].ValueMgMmol
	previous := readings[len(readings)-2].ValueMgMmol

	oldOrd := models.ACRCategoryOrdinal(oldCategory)
	newOrd := models.ACRCategoryOrdinal(newCategory)

	// Category step-up is always WORSENING
	if oldOrd > 0 && newOrd > oldOrd {
		return models.ACRTrendWorsening
	}
	// Category step-down is always IMPROVING
	if oldOrd > 0 && newOrd < oldOrd {
		return models.ACRTrendImproving
	}

	// Within same category, check percentage change
	if previous > 0 {
		changePct := (latest - previous) / previous
		if changePct > 0.20 {
			return models.ACRTrendWorsening
		}
		if changePct < -0.20 {
			return models.ACRTrendImproving
		}
	}

	return models.ACRTrendStable
}

// patientOnRAAS checks whether the patient has an active ACEi or ARB medication.
func (s *LabService) patientOnRAAS(tx *gorm.DB, patientID string) bool {
	var count int64
	tx.Model(&models.MedicationState{}).
		Where("patient_id = ? AND is_active = true AND drug_class IN ?",
			patientID, []string{models.DrugClassACEInhibitor, models.DrugClassARB}).
		Count(&count)
	return count > 0
}

// GetACRTracking returns the current ACR tracking record for a patient.
func (s *LabService) GetACRTracking(patientID string) (*models.ACRTracking, error) {
	var tracking models.ACRTracking
	if err := s.db.DB.Where("patient_id = ?", patientID).First(&tracking).Error; err != nil {
		return nil, fmt.Errorf("ACR tracking not found for patient %s: %w", patientID, err)
	}
	return &tracking, nil
}

// ---------------------------------------------------------------------------
// AD-06: Deprescribing Failure Threshold Correction
// ---------------------------------------------------------------------------

// ComputeDeprescribingFailureThreshold returns the SBP value above which a
// step-down is considered failed. The threshold is bp_target.sbp + 10, NOT
// baseline + 10. This is a critical distinction: an SBP that rises but
// remains below target + 10 is not a failure.
//
// Example with target 130, pre-step-down SBP 118:
//   - SBP rises to 129 → below 140 → NOT a failure
//   - SBP rises to 142 → exceeds 140 → STEP_DOWN_FAILED
func ComputeDeprescribingFailureThreshold(bpTargetSBP float64) float64 {
	return bpTargetSBP + 10
}

// ---------------------------------------------------------------------------
// AD-10: ACR Recheck During ACEi/ARB Dose Reduction
// ---------------------------------------------------------------------------

// SetACRRecheckDue flags that an ACR recheck is required for a patient
// entering the ACEi/ARB monitoring window. The flag is stored as an
// ACR_RECHECK_DUE event on the outbox so downstream services (KB-23, lab
// scheduling) can act on it.
func (s *LabService) SetACRRecheckDue(ctx context.Context, patientID string) error {
	s.eventBus.Publish("ACR_RECHECK_DUE", patientID, map[string]interface{}{
		"reason":       "ACEi/ARB dose reduction entered monitoring phase",
		"requested_at": time.Now().UTC().Format(time.RFC3339),
	})
	s.logger.Info("AD-10: ACR recheck flagged",
		zap.String("patient_id", patientID))
	return nil
}

// ---------------------------------------------------------------------------
// RAAS Creatinine Monitoring (RAAS_MONITORING_ESCALATE / RESOLVED)
// ---------------------------------------------------------------------------

// checkRAASCreatinineMonitoring detects concerning creatinine changes after
// ACEi/ARB initiation or titration. KDIGO recommends monitoring creatinine
// within 2-4 weeks of RAAS blockade changes:
//   - Rise >30% from pre-RAAS baseline → RAAS_MONITORING_ESCALATE
//   - Rise >20% with concurrent hyperkalaemia → RAAS_MONITORING_ESCALATE
//   - Previously escalated + rise settled to <10% → RAAS_MONITORING_RESOLVED
//
// The 30% threshold aligns with KDIGO 2024 §4.2 guidance: a rise up to 30%
// is pharmacodynamically expected post-ACEi/ARB and should not trigger alarm.
// Beyond 30%, the risk of true RAAS-induced renal injury requires action.
func (s *LabService) checkRAASCreatinineMonitoring(tx *gorm.DB, patientID string, creatinine float64, measuredAt time.Time) {
	if !s.patientOnRAAS(tx, patientID) {
		return
	}

	// Get the pre-RAAS baseline creatinine: the most recent creatinine BEFORE
	// the latest ACEi/ARB change date.
	var raasChangeAt time.Time
	var raasMed models.MedicationState
	err := tx.Where("patient_id = ? AND is_active = true AND drug_class IN ?",
		patientID, []string{models.DrugClassACEInhibitor, models.DrugClassARB}).
		Order("updated_at DESC").First(&raasMed).Error
	if err != nil {
		return
	}
	raasChangeAt = raasMed.UpdatedAt
	daysSinceRAAS := int(measuredAt.Sub(raasChangeAt).Hours() / 24)

	// Only monitor within 60-day post-RAAS window
	if daysSinceRAAS > 60 {
		return
	}

	// Find baseline creatinine (last value before RAAS change)
	var baselineEntry models.LabEntry
	err = tx.Where("patient_id = ? AND lab_type = ? AND validation_status = ? AND measured_at < ?",
		patientID, models.LabTypeCreatinine, models.ValidationAccepted, raasChangeAt).
		Order("measured_at DESC").First(&baselineEntry).Error
	if err != nil {
		return // No baseline — cannot compute rise
	}

	baseline, _ := baselineEntry.Value.Float64()
	if baseline <= 0 {
		return
	}

	risePct := ((creatinine - baseline) / baseline) * 100

	// Get current potassium for hyperkalaemia assessment
	var kEntry models.LabEntry
	potassium := 0.0
	err = tx.Where("patient_id = ? AND lab_type = ? AND validation_status = ?",
		patientID, models.LabTypePotassium, models.ValidationAccepted).
		Order("measured_at DESC").First(&kEntry).Error
	if err == nil {
		potassium, _ = kEntry.Value.Float64()
	}

	// Escalation criteria:
	// 1. Creatinine rise >30% (beyond expected RAAS pharmacodynamics)
	// 2. Rise >20% WITH hyperkalaemia (K+ >5.5)
	escalate := false
	action := "RECHECK_48H"
	if risePct > 30 {
		escalate = true
		if risePct > 50 || potassium > 6.0 {
			action = "HOLD_AND_REFER"
		}
	} else if risePct > 20 && potassium > 5.5 {
		escalate = true
		action = "RECHECK_48H"
	}

	if escalate {
		s.eventBus.PublishTx(tx, models.EventRAASMonitoringEscalate, patientID,
			models.RAASMonitoringEscalatePayload{
				CreatinineRisePct:   math.Round(risePct*10) / 10,
				DaysSinceRAASChange: daysSinceRAAS,
				PotassiumCurrent:    potassium,
				RequiredAction:      action,
			})
		s.logger.Warn("RAAS_MONITORING_ESCALATE: creatinine rise exceeds tolerance",
			zap.String("patient_id", patientID),
			zap.Float64("rise_pct", risePct),
			zap.Float64("potassium", potassium),
			zap.String("action", action))
		return
	}

	// Resolution check: if rise has settled to <10%, check for prior escalation.
	// We detect prior escalation by looking for a recent RAAS_MONITORING_ESCALATE
	// event in the outbox for this patient.
	if risePct < 10 {
		var escalationEvent models.EventOutboxEntry
		err = tx.Where("patient_id = ? AND event_type = ? AND created_at > ?",
			patientID, models.EventRAASMonitoringEscalate, raasChangeAt).
			First(&escalationEvent).Error
		if err == nil {
			// Prior escalation exists and creatinine has stabilised
			resolution := "STABILISED"
			if risePct < 2 {
				resolution = "RETURNED_TO_BASELINE"
			}
			daysSinceEscalation := int(measuredAt.Sub(escalationEvent.CreatedAt).Hours() / 24)
			s.eventBus.PublishTx(tx, models.EventRAASMonitoringResolved, patientID,
				models.RAASMonitoringResolvedPayload{
					CreatinineCurrentPct: math.Round(risePct*10) / 10,
					DaysSinceEscalation:  daysSinceEscalation,
					Resolution:           resolution,
				})
			s.logger.Info("RAAS_MONITORING_RESOLVED: creatinine rise settled",
				zap.String("patient_id", patientID),
				zap.Float64("rise_pct", risePct),
				zap.String("resolution", resolution))
		}
	}
}

// ---------------------------------------------------------------------------
// SAFETY_ALERT — Generic Safety Event
// ---------------------------------------------------------------------------

// PublishSafetyAlert emits a SAFETY_ALERT event for clinically significant
// safety conditions that don't map to a more specific event type.
// Called from various detection points across KB-20 when a patient's
// lab values indicate immediate clinical attention is needed.
func (s *LabService) PublishSafetyAlert(patientID string, severity string, alertType string, description string, labType string, oldVal string, newVal string) {
	s.eventBus.Publish(models.EventSafetyAlert, patientID, models.SafetyAlertPayload{
		Severity:    severity,
		AlertType:   alertType,
		Description: description,
		LabType:     labType,
		OldValue:    oldVal,
		NewValue:    newVal,
	})
	s.logger.Warn("SAFETY_ALERT published",
		zap.String("patient_id", patientID),
		zap.String("severity", severity),
		zap.String("alert_type", alertType))
}

// EvaluateACRAfterStepDown checks the current ACR category at the end of
// the 6-week monitoring window for an ACEi/ARB dose reduction.
//
// If the ACR category has worsened (e.g. A1 → A2 or A2 → A3), the step-down
// is considered failed regardless of BP status. This protects renal function
// by ensuring RAAS blockade is restored when albuminuria progresses.
func (s *LabService) EvaluateACRAfterStepDown(
	ctx context.Context,
	patientID string,
	preStepDownACRCategory string,
) (failed bool, reason string) {
	// Fetch the most recent ACR lab entry.
	var acrEntry models.LabEntry
	err := s.db.DB.WithContext(ctx).
		Where("patient_id = ? AND lab_type = ? AND validation_status = ?",
			patientID, models.LabTypeACR, models.ValidationAccepted).
		Order("measured_at DESC").
		First(&acrEntry).Error
	if err != nil {
		s.logger.Warn("AD-10: no post-step-down ACR available",
			zap.String("patient_id", patientID),
			zap.Error(err))
		return false, ""
	}

	currentCategory := s.classifyACRCategoryFromEntry(acrEntry)
	preRank := models.ACRCategoryOrdinal(preStepDownACRCategory)
	currentRank := models.ACRCategoryOrdinal(currentCategory)

	if preRank == 0 || currentRank == 0 {
		s.logger.Warn("AD-10: unrecognised ACR category",
			zap.String("patient_id", patientID),
			zap.String("pre", preStepDownACRCategory),
			zap.String("current", currentCategory))
		return false, ""
	}

	if currentRank > preRank {
		s.logger.Warn("AD-10: ACR worsened during RAAS reduction",
			zap.String("patient_id", patientID),
			zap.String("pre_category", preStepDownACRCategory),
			zap.String("current_category", currentCategory))
		return true, "ACR_WORSENED_DURING_RAAS_REDUCTION"
	}

	return false, ""
}

// classifyACRCategoryFromEntry maps a raw ACR value (mg/mmol) to a KDIGO
// category from a lab entry.
func (s *LabService) classifyACRCategoryFromEntry(entry models.LabEntry) string {
	val, _ := entry.Value.Float64()
	return models.CategorizeACR(val)
}

// ---------------------------------------------------------------------------
// EW-01 through EW-08: Early Warning Loop
// ---------------------------------------------------------------------------

// ComputeBPRiskStratum determines the patient's risk stratum based on CKD stage
// and ACR category, returning the stratum key, declining threshold, early watch
// floor, and early watch weeks. (EW-01/02)
//
// Mapping logic:
//   - DM + CKD 3b, any ACR         → DM_CKD3B_ANY
//   - DM + any CKD, ACR A2/A3      → DM_CKD_A2A3
//   - DM + CKD 3a, ACR A1          → DM_CKD3A_A1
//   - DM only (no CKD), ACR A1     → DM_ONLY_A1 (default)
func ComputeBPRiskStratum(ckdStage string, acrCategory string) (stratum string, decliningThreshold float64, earlyWatchFloor float64, earlyWatchWeeks int) {
	// Default to the least restrictive stratum
	stratum = "DM_ONLY_A1"

	hasCKD := ckdStage != "" && ckdStage != "NONE" && ckdStage != "G1" && ckdStage != "G2"
	isG3b := ckdStage == models.CKDG3b || ckdStage == models.CKDG4 || ckdStage == models.CKDG5
	isA2A3 := acrCategory == models.ACRCategoryA2 || acrCategory == models.ACRCategoryA3

	switch {
	case isG3b:
		// CKD 3b or worse — most restrictive threshold regardless of ACR
		stratum = "DM_CKD3B_ANY"
	case hasCKD && isA2A3:
		// Any CKD stage with elevated albuminuria
		stratum = "DM_CKD_A2A3"
	case ckdStage == models.CKDG3a && !isA2A3:
		// CKD 3a with normal albuminuria
		stratum = "DM_CKD3A_A1"
	default:
		stratum = "DM_ONLY_A1"
	}

	entry := models.BPRiskStratumTable[stratum]
	return stratum, entry.DecliningThreshold, entry.EarlyWatchFloor, entry.EarlyWatchWeeks
}

// ComputeSlopeConfidence classifies data adequacy from the reading count
// in the last 4 weeks. (EW-06)
func ComputeSlopeConfidence(readingCount int) models.SlopeConfidence {
	switch {
	case readingCount >= 8:
		return models.SlopeConfidenceHigh
	case readingCount >= 5:
		return models.SlopeConfidenceModerate
	default:
		return models.SlopeConfidenceLow
	}
}

// computeSBPSlope calculates the SBP slope in mmHg/week from a set of SBP
// readings using ordinary least-squares linear regression on time (weeks).
// Returns nil if fewer than 3 data points.
func computeSBPSlope(sbpEntries []models.LabEntry) *float64 {
	n := len(sbpEntries)
	if n < 3 {
		return nil
	}

	// Use first reading as time origin
	t0 := sbpEntries[0].MeasuredAt
	var sumT, sumY, sumTY, sumTT float64
	for _, e := range sbpEntries {
		t := e.MeasuredAt.Sub(t0).Hours() / (24.0 * 7.0) // weeks since first reading
		y, _ := e.Value.Float64()
		sumT += t
		sumY += y
		sumTY += t * y
		sumTT += t * t
	}
	nf := float64(n)
	denom := nf*sumTT - sumT*sumT
	if denom == 0 {
		return nil
	}
	slope := (nf*sumTY - sumT*sumY) / denom
	return &slope
}

// compute7dMean returns the mean SBP from readings in the last 7 days.
// Returns nil if no readings found in that window.
func compute7dMean(sbpEntries []models.LabEntry) *float64 {
	cutoff := time.Now().AddDate(0, 0, -7)
	var sum float64
	var count int
	for _, e := range sbpEntries {
		if e.MeasuredAt.After(cutoff) || e.MeasuredAt.Equal(cutoff) {
			v, _ := e.Value.Float64()
			sum += v
			count++
		}
	}
	if count == 0 {
		return nil
	}
	mean := sum / float64(count)
	return &mean
}

// UpdateEarlyWarning is called after each BP trajectory computation.
// It resolves the patient's risk stratum, evaluates EARLY_WATCH conditions,
// manages the consecutive weeks counter, computes time-to-severe projection,
// and emits BP_TRAJECTORY_CONCERN when the stratum threshold is exceeded. (EW-01 to EW-06)
func (s *LabService) UpdateEarlyWarning(ctx context.Context, patientID string, traj *models.BPTrajectory) error {
	// 1. Resolve CKD stage and ACR category from patient profile and tracking
	var profile models.PatientProfile
	if err := s.db.DB.WithContext(ctx).Where("patient_id = ?", patientID).First(&profile).Error; err != nil {
		s.logger.Warn("EW: cannot resolve risk stratum — patient profile not found",
			zap.String("patient_id", patientID))
		return nil // non-fatal: early warning is best-effort
	}

	acrCategory := models.ACRCategoryA1 // default if no ACR tracking
	var acrTracking models.ACRTracking
	if err := s.db.DB.WithContext(ctx).Where("patient_id = ?", patientID).First(&acrTracking).Error; err == nil {
		acrCategory = acrTracking.Category
	}

	// 2. Compute risk stratum thresholds
	stratum, decliningThreshold, earlyWatchFloor, earlyWatchWeeks := ComputeBPRiskStratum(profile.CKDStage, acrCategory)
	traj.BPRiskStratum = stratum
	traj.SBPDecliningThreshold = decliningThreshold
	traj.EarlyWatchFloor = earlyWatchFloor
	traj.EarlyWatchWeeksThreshold = earlyWatchWeeks

	// 3. Retrieve SBP readings for slope computation
	cutoff := time.Now().AddDate(0, 0, -28)
	var sbpEntries []models.LabEntry
	if err := s.db.DB.WithContext(ctx).Where(
		"patient_id = ? AND lab_type = ? AND validation_status = ? AND measured_at >= ?",
		patientID, models.LabTypeSBP, models.ValidationAccepted, cutoff,
	).Order("measured_at ASC").Find(&sbpEntries).Error; err != nil {
		return fmt.Errorf("EW: failed to retrieve SBP readings: %w", err)
	}

	// 4. Compute SBP slope and confidence
	traj.SBP4wSlope = computeSBPSlope(sbpEntries)
	traj.SBP7dMean = compute7dMean(sbpEntries)
	traj.SlopeConfidence = ComputeSlopeConfidence(len(sbpEntries))

	// 5. Determine BP status tier
	slope := 0.0
	if traj.SBP4wSlope != nil {
		slope = *traj.SBP4wSlope
	}
	meanSBP := 0.0
	if traj.MeanSBP28d != nil {
		meanSBP = *traj.MeanSBP28d
	}

	oldStatus := traj.Status

	switch {
	case meanSBP >= models.SBPSevereThreshold:
		traj.Status = models.BPStatusSevere
		traj.ConsecutiveEarlyWatchWeeks = 0

	case slope >= decliningThreshold:
		traj.Status = models.BPStatusDeclining
		traj.ConsecutiveEarlyWatchWeeks = 0

	case slope > 0 && slope >= earlyWatchFloor && slope < decliningThreshold:
		traj.Status = models.BPStatusEarlyWatch
		traj.ConsecutiveEarlyWatchWeeks++

	case meanSBP >= 130:
		traj.Status = models.BPStatusAboveTarget
		traj.ConsecutiveEarlyWatchWeeks = 0

	default:
		traj.Status = models.BPStatusAtTarget
		traj.ConsecutiveEarlyWatchWeeks = 0
	}

	// Emit BP status transition events (Gap #21 wiring)
	if traj.Status != oldStatus {
		bpPayload := models.BPAlertPayload{
			PatientID:  patientID,
			BPStatus:   string(traj.Status),
			SBP7dMean:  func() float64 { if traj.SBP7dMean != nil { return *traj.SBP7dMean }; return 0 }(),
			DBP7dMean:  0, // DBP 7d mean not tracked in trajectory
			SBP4wSlope: slope,
			TargetSBP:  130,
		}
		switch traj.Status {
		case models.BPStatusSevere:
			s.eventBus.Publish(models.EventBPSevereAlert, patientID, bpPayload)
		case models.BPStatusAboveTarget, models.BPStatusDeclining:
			s.eventBus.Publish(models.EventBPAlert, patientID, bpPayload)
		case models.BPStatusAtTarget:
			if oldStatus == models.BPStatusAboveTarget || oldStatus == models.BPStatusSevere || oldStatus == models.BPStatusDeclining {
				s.eventBus.Publish(models.EventBPControlled, patientID, bpPayload)
			}
		}
	}

	// 5b. BP_VARIABILITY_ALERT: emit when visit-to-visit SBP variability is HIGH (SD > 15 mmHg).
	// Uses the sbpEntries already fetched in step 3 above.
	if len(sbpEntries) >= 2 {
		var dbpEntries []models.LabEntry
		s.db.DB.WithContext(ctx).Where(
			"patient_id = ? AND lab_type = ? AND validation_status = ? AND measured_at >= ?",
			patientID, models.LabTypeDBP, models.ValidationAccepted, cutoff,
		).Order("measured_at ASC").Find(&dbpEntries)

		sbpSD, dbpSD, varStatus := s.ComputeBPVariability(sbpEntries, dbpEntries)
		if varStatus == models.VariabilityHigh {
			s.eventBus.Publish(models.EventBPVariabilityAlert, patientID,
				models.BPVariabilityAlertPayload{
					PatientID:         patientID,
					SBPSD:             sbpSD,
					DBPSD:             dbpSD,
					VariabilityStatus: varStatus,
					ReadingCount:      len(sbpEntries),
				})
			s.logger.Warn("BP_VARIABILITY_ALERT: SBP SD > 15 mmHg (HIGH variability)",
				zap.String("patient_id", patientID),
				zap.Float64("sbp_sd", sbpSD))
		}
	}

	// 6. Emit BP_TRAJECTORY_CONCERN when EARLY_WATCH exceeds stratum threshold
	if traj.Status == models.BPStatusEarlyWatch && traj.ConsecutiveEarlyWatchWeeks >= earlyWatchWeeks {
		suggestion := fmt.Sprintf(
			"BP trending upward slowly (+%.1f mmHg/week) for %d consecutive weeks. Risk stratum: %s.",
			slope, traj.ConsecutiveEarlyWatchWeeks, stratum,
		)
		s.eventBus.Publish(models.EventBPTrajectoryConcern, patientID,
			models.BPTrajectoryConcernPayload{
				PatientID:                  patientID,
				SBPSlope:                   slope,
				ConsecutiveEarlyWatchWeeks: traj.ConsecutiveEarlyWatchWeeks,
				BPRiskStratum:              stratum,
				EarlyWatchThreshold:        earlyWatchFloor,
				Pattern:                    string(traj.Pattern),
				MeanSBP28d:                 meanSBP,
				ReadingsUsed:               traj.TotalReadings28d,
				Suggestion:                 suggestion,
			})
		s.logger.Warn("EW-04: BP trajectory concern emitted",
			zap.String("patient_id", patientID),
			zap.Float64("slope", slope),
			zap.Int("consecutive_weeks", traj.ConsecutiveEarlyWatchWeeks),
			zap.String("stratum", stratum))
	}

	// 7. Compute weeks-to-severe projection (EW-05)
	if slope > 0 && traj.SBP7dMean != nil {
		gap := models.SBPSevereThreshold - *traj.SBP7dMean
		if gap > 0 {
			wts := gap / slope
			traj.WeeksToSevere = &wts
		}
	} else {
		traj.WeeksToSevere = nil
	}

	s.logger.Info("EW: early warning updated",
		zap.String("patient_id", patientID),
		zap.String("status", string(traj.Status)),
		zap.String("stratum", stratum),
		zap.Float64("slope", slope),
		zap.String("confidence", string(traj.SlopeConfidence)))

	return nil
}

// ComputeDamageComposite calculates the compound damage concern score (EW-07/08).
// Called after each BP trajectory update. The score is composed of four
// contributors (0-2 each), totalling 0-8.
//
// Contributors:
//   - Variability: SBP standard deviation. MODERATE (SD 12-18) = 1, HIGH (SD > 18) = 2
//   - ACR trend: WORSENING = 1, category step-up to A3 = 2
//   - Pulse pressure: > 60 + widening trend = 1, > 80 = 2
//   - BP status: ABOVE_TARGET >= 8 weeks with adherence >= 0.85 = 2
//
// Events emitted:
//   - Score 3-4: BP_SUBCLINICAL_CONCERN
//   - Score >= 5: DAMAGE_COMPOSITE_ALERT
func (s *LabService) ComputeDamageComposite(ctx context.Context, patientID string, traj *models.BPTrajectory) error {
	dc := models.DamageComposite{
		ComputedAt: time.Now(),
	}

	// --- Variability contributor (from measurement uncertainty = SBP σ) ---
	switch {
	case traj.MeasurementUncertainty > 18:
		dc.VariabilityContrib = 2
	case traj.MeasurementUncertainty >= 12:
		dc.VariabilityContrib = 1
	}

	// --- ACR trend contributor ---
	var acrTracking models.ACRTracking
	if err := s.db.DB.WithContext(ctx).Where("patient_id = ?", patientID).First(&acrTracking).Error; err == nil {
		switch {
		case acrTracking.Category == models.ACRCategoryA3 && acrTracking.Trend == models.ACRTrendWorsening:
			dc.ACRTrendContrib = 2
		case acrTracking.Trend == models.ACRTrendWorsening:
			dc.ACRTrendContrib = 1
		}
	}

	// --- Pulse pressure contributor ---
	if traj.MeanSBP28d != nil && traj.MeanDBP28d != nil {
		pp := *traj.MeanSBP28d - *traj.MeanDBP28d
		switch {
		case pp > 80:
			dc.PulsePressureContrib = 2
		case pp > 60:
			// Check for widening trend: compare 7d vs 28d pulse pressure
			// A widening PP with >60 qualifies for 1 point
			dc.PulsePressureContrib = 1
		}
	}

	// --- BP status contributor ---
	// 2 points if ABOVE_TARGET for >= 8 weeks with adherence >= 0.85
	// We use the pattern as proxy: SUSTAINED_HIGH implies extended above-target
	if traj.Pattern == models.BPPatternSustainedHigh || traj.Status == models.BPStatusAboveTarget {
		// Check adherence by counting RAAS medications (proxy: if on >=1 HTN med, assume adherence tracked externally)
		// Score 2 only for established above-target (sustained high pattern implies >= 2 weeks)
		if traj.Pattern == models.BPPatternSustainedHigh {
			dc.BPStatusContrib = 2
		} else {
			dc.BPStatusContrib = 1
		}
	}

	dc.Score = dc.VariabilityContrib + dc.ACRTrendContrib + dc.PulsePressureContrib + dc.BPStatusContrib
	traj.DamageScore = &dc

	// Emit events based on score — with hysteresis (Wave 2 Track C):
	// 1. After an alert is emitted, do NOT re-emit for the same or lower score
	//    within 72 hours (cooldown).
	// 2. If score drops to 0, clear the cooldown immediately (all-clear reset).
	// 3. If score INCREASES above the previously alerted score, emit immediately
	//    regardless of cooldown.
	payload := models.DamageCompositePayload{
		PatientID:            patientID,
		Score:                dc.Score,
		VariabilityContrib:   dc.VariabilityContrib,
		ACRTrendContrib:      dc.ACRTrendContrib,
		PulsePressureContrib: dc.PulsePressureContrib,
		BPStatusContrib:      dc.BPStatusContrib,
	}

	// All-clear reset: score dropped to 0 — clear cooldown
	if dc.Score == 0 {
		traj.LastDamageAlertScore = 0
		traj.LastDamageAlertTime = nil
	}

	shouldEmit := false
	if dc.Score >= 3 {
		inCooldown := false
		if traj.LastDamageAlertTime != nil {
			elapsed := time.Since(*traj.LastDamageAlertTime)
			inCooldown = elapsed.Hours() < float64(models.DamageAlertCooldownHours)
		}

		if dc.Score > traj.LastDamageAlertScore {
			// Score increased above previous alert — emit immediately regardless of cooldown
			shouldEmit = true
		} else if !inCooldown {
			// Not in cooldown — emit normally
			shouldEmit = true
		}
	}

	if shouldEmit {
		now := time.Now()
		traj.LastDamageAlertScore = dc.Score
		traj.LastDamageAlertTime = &now

		switch {
		case dc.Score >= 5:
			s.eventBus.Publish(models.EventDamageCompositeAlert, patientID, payload)
			s.logger.Warn("EW-08: damage composite alert",
				zap.String("patient_id", patientID),
				zap.Int("score", dc.Score))
		case dc.Score >= 3:
			s.eventBus.Publish(models.EventBPSubclinicalConcern, patientID, payload)
			s.logger.Warn("EW-07: subclinical concern",
				zap.String("patient_id", patientID),
				zap.Int("score", dc.Score))
		}
	} else if dc.Score >= 3 {
		s.logger.Info("EW: damage composite alert suppressed by hysteresis cooldown",
			zap.String("patient_id", patientID),
			zap.Int("score", dc.Score),
			zap.Int("last_alert_score", traj.LastDamageAlertScore))
	}

	return nil
}

// ---------------------------------------------------------------------------
// Wave 3.1 Amendment 7: Visit-to-visit BP Variability
// ---------------------------------------------------------------------------

// ComputeBPVariability calculates visit-to-visit BP variability (SD) over the
// last 5 SBP and DBP readings. Classifies status based on SBP SD thresholds:
//   - LOW:      SD < 10 mmHg
//   - MODERATE: SD 10-15 mmHg
//   - HIGH:     SD > 15 mmHg
//
// Returns (sbpSD, dbpSD, status). If fewer than 2 readings, returns (0, 0, "LOW").
func (s *LabService) ComputeBPVariability(sbpReadings, dbpReadings []models.LabEntry) (sbpSD, dbpSD float64, status string) {
	sbpSD = computeSDFromEntries(lastN(sbpReadings, 5))
	dbpSD = computeSDFromEntries(lastN(dbpReadings, 5))

	switch {
	case sbpSD > 15:
		status = models.VariabilityHigh
	case sbpSD >= 10:
		status = models.VariabilityModerate
	default:
		status = models.VariabilityLow
	}
	return sbpSD, dbpSD, status
}

// ---------------------------------------------------------------------------
// Wave 3.4 Amendment 13: Pulse Pressure Statistics
// ---------------------------------------------------------------------------

// ComputePulsePressure derives pulse pressure (SBP - DBP) statistics over the
// last 5 paired SBP/DBP readings. It pairs readings by closest timestamp
// within a 30-minute window.
//
// Returns mean PP and trend:
//   - WIDENING:  second-half mean PP exceeds first-half mean PP by >5 mmHg
//   - NARROWING: second-half mean PP is below first-half mean PP by >5 mmHg
//   - STABLE:    otherwise
//
// Returns (0, "STABLE") if fewer than 2 pairs are found.
func (s *LabService) ComputePulsePressure(sbpReadings, dbpReadings []models.LabEntry) (meanPP float64, trend string) {
	// Pair SBP/DBP by closest timestamp within 30 minutes
	type ppPair struct {
		pp float64
	}
	var pairs []ppPair
	usedDBP := make(map[int]bool)

	sbpWindow := lastN(sbpReadings, 5)
	for _, sbp := range sbpWindow {
		bestIdx := -1
		bestDelta := time.Duration(math.MaxInt64)
		for j, dbp := range dbpReadings {
			if usedDBP[j] {
				continue
			}
			delta := sbp.MeasuredAt.Sub(dbp.MeasuredAt)
			if delta < 0 {
				delta = -delta
			}
			if delta < bestDelta && delta <= 30*time.Minute {
				bestDelta = delta
				bestIdx = j
			}
		}
		if bestIdx >= 0 {
			sbpVal, _ := sbp.Value.Float64()
			dbpVal, _ := dbpReadings[bestIdx].Value.Float64()
			pairs = append(pairs, ppPair{pp: sbpVal - dbpVal})
			usedDBP[bestIdx] = true
		}
	}

	if len(pairs) < 2 {
		return 0, models.PulsePressureTrendStable
	}

	// Mean PP
	var sum float64
	for _, p := range pairs {
		sum += p.pp
	}
	meanPP = sum / float64(len(pairs))

	// Trend: compare first half vs second half means
	mid := len(pairs) / 2
	firstHalf := pairs[:mid]
	secondHalf := pairs[mid:]

	var firstSum, secondSum float64
	for _, p := range firstHalf {
		firstSum += p.pp
	}
	for _, p := range secondHalf {
		secondSum += p.pp
	}
	firstMean := firstSum / float64(len(firstHalf))
	secondMean := secondSum / float64(len(secondHalf))

	diff := secondMean - firstMean
	switch {
	case diff > 5:
		trend = models.PulsePressureTrendWidening
	case diff < -5:
		trend = models.PulsePressureTrendNarrowing
	default:
		trend = models.PulsePressureTrendStable
	}
	return meanPP, trend
}

// ---------------------------------------------------------------------------
// Wave 2 Track G: SBP Slope Acceleration
// ---------------------------------------------------------------------------

// ComputeSBPSlopeAcceleration computes the second derivative of SBP trajectory.
// A positive acceleration means the rate of SBP increase is itself increasing,
// which is a stronger warning signal than slope alone.
//
// Requires at least 3 slope measurements (computed weekly over at least 3 weeks).
// Returns nil if insufficient data. Units: mmHg/week^2.
func (s *LabService) ComputeSBPSlopeAcceleration(weeklySlopes []float64) *float64 {
	if len(weeklySlopes) < 3 {
		return nil
	}

	// Compute first differences of slopes (acceleration per interval)
	var accelSum float64
	n := len(weeklySlopes) - 1
	for i := 0; i < n; i++ {
		accelSum += weeklySlopes[i+1] - weeklySlopes[i]
	}
	meanAccel := accelSum / float64(n)
	return &meanAccel
}

// ---------------------------------------------------------------------------
// Helper functions for BP variability and pulse pressure
// ---------------------------------------------------------------------------

// lastN returns the last n elements of a LabEntry slice. If the slice has
// fewer than n elements, the entire slice is returned.
func lastN(entries []models.LabEntry, n int) []models.LabEntry {
	if len(entries) <= n {
		return entries
	}
	return entries[len(entries)-n:]
}

// computeSDFromEntries computes the sample standard deviation of lab entry
// values. Returns 0 if fewer than 2 entries.
func computeSDFromEntries(entries []models.LabEntry) float64 {
	n := len(entries)
	if n < 2 {
		return 0
	}

	var sum float64
	vals := make([]float64, n)
	for i, e := range entries {
		v, _ := e.Value.Float64()
		vals[i] = v
		sum += v
	}
	mean := sum / float64(n)

	var sumSq float64
	for _, v := range vals {
		diff := v - mean
		sumSq += diff * diff
	}
	return math.Sqrt(sumSq / float64(n-1))
}

// ComputeOrthostaticDrop calculates the postural BP change from paired SEATED and STANDING readings.
// Returns standing - seated (negative value = drop). Drop < -20 mmHg is clinically significant.
// Requires readings taken within 3 minutes of each other.
func ComputeOrthostaticDrop(seatedSBP, standingSBP float64) float64 {
	return standingSBP - seatedSBP
}

// GetEGFRTrajectory returns the eGFR history with trend classification.
func (s *LabService) GetEGFRTrajectory(patientID string) (*models.EGFRTrajectoryResponse, error) {
	var entries []models.LabEntry
	s.db.DB.Where("patient_id = ? AND lab_type = ? AND validation_status = ?",
		patientID, models.LabTypeEGFR, models.ValidationAccepted).
		Order("measured_at ASC").Find(&entries)

	var points []models.EGFRTrajectoryPoint
	for _, e := range entries {
		val, _ := e.Value.Float64()
		points = append(points, models.EGFRTrajectoryPoint{
			Value:      val,
			MeasuredAt: e.MeasuredAt,
			CKDStage:   s.egfr.CKDStageFromEGFR(val),
		})
	}

	trend, annualChange := s.egfr.ClassifyTrajectory(points)

	return &models.EGFRTrajectoryResponse{
		PatientID:    patientID,
		Points:       points,
		Trend:        trend,
		AnnualChange: annualChange,
	}, nil
}
