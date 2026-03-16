package services

import (
	"fmt"
	"math"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"kb-patient-profile/internal/cache"
	"kb-patient-profile/internal/database"
	"kb-patient-profile/internal/models"
)

// FactStore projection cache configuration.
const (
	ChannelBCachePrefix = "kb20:chb:"
	ChannelCCachePrefix = "kb20:chc:"
	ProjectionCacheTTL  = 2 * time.Minute
)

// ProjectionService builds typed projections for V-MCU Channel B and Channel C
// from KB-20's existing lab and medication data. No new database tables —
// it queries LabEntry, MedicationState, PatientProfile, and BPTrajectory.
// LOINC code ownership is delegated to KB-7 via the LOINCRegistry.
type ProjectionService struct {
	db            *database.Database
	cache         *cache.Client
	logger        *zap.Logger
	loincRegistry *LOINCRegistry
}

// NewProjectionService creates a projection service with LOINC registry integration.
func NewProjectionService(db *database.Database, cacheClient *cache.Client, logger *zap.Logger, loincReg *LOINCRegistry) *ProjectionService {
	return &ProjectionService{
		db:            db,
		cache:         cacheClient,
		logger:        logger,
		loincRegistry: loincReg,
	}
}

// GetChannelBProjection returns the typed Channel B input data for a patient.
// Checks Redis cache first (TTL 2min), then builds from database.
func (s *ProjectionService) GetChannelBProjection(patientID string) (*models.ChannelBProjection, error) {
	cacheKey := ChannelBCachePrefix + patientID

	// Try cache first
	var cached models.ChannelBProjection
	if err := s.cache.Get(cacheKey, &cached); err == nil {
		return &cached, nil
	}

	// Build from database
	projection, err := s.buildChannelBProjection(patientID)
	if err != nil {
		return nil, fmt.Errorf("failed to build Channel B projection: %w", err)
	}

	// Cache the result
	if cacheErr := s.cache.Set(cacheKey, projection, ProjectionCacheTTL); cacheErr != nil {
		s.logger.Warn("Failed to cache Channel B projection",
			zap.String("patient_id", patientID),
			zap.Error(cacheErr))
	}

	return projection, nil
}

// GetChannelCProjection returns the typed Channel C input data for a patient.
// Checks Redis cache first (TTL 2min), then builds from database.
func (s *ProjectionService) GetChannelCProjection(patientID string) (*models.ChannelCProjection, error) {
	cacheKey := ChannelCCachePrefix + patientID

	// Try cache first
	var cached models.ChannelCProjection
	if err := s.cache.Get(cacheKey, &cached); err == nil {
		return &cached, nil
	}

	// Build from database
	projection, err := s.buildChannelCProjection(patientID)
	if err != nil {
		return nil, fmt.Errorf("failed to build Channel C projection: %w", err)
	}

	// Cache the result
	if cacheErr := s.cache.Set(cacheKey, projection, ProjectionCacheTTL); cacheErr != nil {
		s.logger.Warn("Failed to cache Channel C projection",
			zap.String("patient_id", patientID),
			zap.Error(cacheErr))
	}

	return projection, nil
}

// InvalidateProjectionCache removes both Channel B and Channel C cache entries
// for a patient. Called when labs or medications change.
func (s *ProjectionService) InvalidateProjectionCache(patientID string) {
	s.cache.Delete(ChannelBCachePrefix + patientID)
	s.cache.Delete(ChannelCCachePrefix + patientID)
}

// ──────────────────────────────────────────────────────────────────────────
// Channel B projection builder
// ──────────────────────────────────────────────────────────────────────────

func (s *ProjectionService) buildChannelBProjection(patientID string) (*models.ChannelBProjection, error) {
	proj := &models.ChannelBProjection{
		PatientID:   patientID,
		ProjectedAt: time.Now().UTC(),
	}

	// 1. Fetch patient profile (for CKD stage, season, BP pattern)
	var profile models.PatientProfile
	if err := s.db.DB.Where("patient_id = ?", patientID).First(&profile).Error; err != nil {
		return nil, fmt.Errorf("patient not found: %w", err)
	}
	proj.CKDStage = profile.CKDStage
	proj.Season = profile.Season

	// 2. Fetch latest labs by type (current values)
	// Glucose: exclude PATIENT_REPORTED source for reliability (G-03)
	proj.GlucoseCurrent, proj.GlucoseTimestamp, proj.GlucoseSource = s.latestLabValueFilteredSource(patientID, models.LabTypeFBG, "PATIENT_REPORTED")
	proj.CreatinineCurrent, _ = s.latestLabValue(patientID, models.LabTypeCreatinine)
	proj.PotassiumCurrent, _ = s.latestLabValue(patientID, models.LabTypePotassium)
	proj.SBPCurrent, _ = s.latestLabValue(patientID, models.LabTypeSBP)
	proj.DBPCurrent, _ = s.latestLabValue(patientID, models.LabTypeDBP)
	proj.EGFRCurrent, _ = s.latestLabValue(patientID, models.LabTypeEGFR)
	proj.HbA1cCurrent, _ = s.latestLabValue(patientID, models.LabTypeHbA1c)
	proj.SodiumCurrent, _ = s.latestLabValue(patientID, models.LabTypeSodium)
	proj.HeartRateCurrent, _ = s.latestLabValue(patientID, "HEART_RATE")

	// Weight — prefer lab-sourced WEIGHT, fall back to profile
	if weightVal, _ := s.latestLabValue(patientID, "WEIGHT"); weightVal != nil {
		proj.WeightKgCurrent = weightVal
	} else if profile.WeightKg > 0 {
		proj.WeightKgCurrent = float64Ptr(profile.WeightKg)
	}

	// 3. Historical values for delta computation (calibrated windows per plan spec)
	proj.Creatinine48hAgo = s.labValueAtOffsetWithWindow(patientID, models.LabTypeCreatinine, 48*time.Hour, 4*time.Hour)  // ±4h for AKI detection (KDIGO 48h criterion)
	proj.EGFRPrior48h = s.labValueAtOffsetWithWindow(patientID, models.LabTypeEGFR, 48*time.Hour, 12*time.Hour)          // ±12h acceptable for eGFR
	proj.HbA1cPrior30d = s.labValueAtOffsetWithWindow(patientID, models.LabTypeHbA1c, 30*24*time.Hour, 5*24*time.Hour)   // 25-35d window (±5d)
	proj.Weight72hAgo = s.labValueAtOffsetWithWindow(patientID, "WEIGHT", 72*time.Hour, 4*time.Hour)                     // ±4h for weight

	// 4. Staleness — per-lab-type with StaleDays and IsStale pre-computed
	now := time.Now().UTC()
	proj.Staleness = models.StalenessInfo{
		Labs: map[string]models.LabStaleness{
			models.LabTypeEGFR:       s.computeLabStaleness(patientID, models.LabTypeEGFR, models.StalenessThresholdEGFR, now),
			models.LabTypeHbA1c:      s.computeLabStaleness(patientID, models.LabTypeHbA1c, models.StalenessThresholdHbA1c, now),
			models.LabTypeCreatinine: s.computeLabStaleness(patientID, models.LabTypeCreatinine, models.StalenessThresholdCreatinine, now),
			models.LabTypePotassium:  s.computeLabStaleness(patientID, models.LabTypePotassium, models.StalenessThresholdPotassium, now),
		},
	}

	// 5. Medication flags
	activeMeds, err := s.getActiveMedications(patientID)
	if err != nil {
		s.logger.Warn("Failed to fetch medications for Channel B projection",
			zap.String("patient_id", patientID),
			zap.Error(err))
	} else {
		proj.OnRAASAgent = hasDrugClass(activeMeds, models.DrugClassACEInhibitor) || hasDrugClass(activeMeds, models.DrugClassARB)
		proj.BetaBlockerActive = hasDrugClass(activeMeds, models.DrugClassBetaBlocker)
		proj.ThiazideActive = hasDrugClass(activeMeds, models.DrugClassDiuretic)
		proj.FinerenoneActive = hasDrugClass(activeMeds, models.DrugClassFinerenone) // B-21

		// Beta-blocker dose change in last 7 days
		proj.BetaBlockerDoseChangeIn7d = s.hasMedChangeInWindow(patientID, models.DrugClassBetaBlocker, 7*24*time.Hour)
	}

	// 6. BP trajectory context
	var bpTraj models.BPTrajectory
	if err := s.db.DB.Where("patient_id = ?", patientID).First(&bpTraj).Error; err == nil {
		proj.BPPattern = string(bpTraj.Pattern)
		proj.MeasurementUncertainty = bpTraj.MeasurementUncertainty

		// J-curve SBP lower limit from eGFR stratification
		proj.SBPLowerLimit = jCurveSBPFloor(profile.CKDStage)
	}

	// 7. Glucose readings (last 3 for trend, excluding PATIENT_REPORTED)
	proj.GlucoseReadings = s.recentLabValuesFilteredSource(patientID, models.LabTypeFBG, "PATIENT_REPORTED", 3)

	// 8. eGFR slope (from trajectory)
	if proj.EGFRCurrent != nil {
		proj.EGFRSlope = s.computeEGFRSlope(patientID)
	}

	// 9. RAAS creatinine tolerance (PG-14)
	proj.CreatinineRiseExplained = s.isCreatinineRiseExplainedByRAAS(patientID, activeMeds)

	// 10. Glucose CV30d from FBGTracking (B-20: CV% > 36% → PAUSE)
	var fbgTracking models.FBGTracking
	if err := s.db.DB.Where("patient_id = ?", patientID).First(&fbgTracking).Error; err == nil {
		if fbgTracking.CV30d > 0 {
			proj.GlucoseCV30d = float64Ptr(fbgTracking.CV30d)
		}
	}

	return proj, nil
}

// ──────────────────────────────────────────────────────────────────────────
// Channel C projection builder
// ──────────────────────────────────────────────────────────────────────────

func (s *ProjectionService) buildChannelCProjection(patientID string) (*models.ChannelCProjection, error) {
	proj := &models.ChannelCProjection{
		PatientID:   patientID,
		ProjectedAt: time.Now().UTC(),
	}

	// 1. Fetch patient profile
	var profile models.PatientProfile
	if err := s.db.DB.Where("patient_id = ?", patientID).First(&profile).Error; err != nil {
		return nil, fmt.Errorf("patient not found: %w", err)
	}

	// 2. eGFR
	if val, _ := s.latestLabValue(patientID, models.LabTypeEGFR); val != nil {
		proj.EGFR = *val
	}

	// 3. Active medications (drug class list)
	activeMeds, err := s.getActiveMedications(patientID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch medications: %w", err)
	}
	proj.ActiveMedications = effectiveDrugClasses(activeMeds)

	// 4. Numeric values for threshold comparisons
	if val, _ := s.latestLabValue(patientID, models.LabTypePotassium); val != nil {
		proj.PotassiumCurrent = *val
	}
	if val, _ := s.latestLabValue(patientID, models.LabTypeSBP); val != nil {
		proj.SBPCurrent = *val
	}
	if val, _ := s.latestLabValue(patientID, models.LabTypeSodium); val != nil {
		proj.SodiumCurrent = *val
	}

	// 5. Compute HTN composite booleans

	// PG-08: ACEi/ARB + K+ ≥ 5.5 + declining eGFR (slope < −3.0 mL/min/year = SLOW_DECLINE)
	onACEiARB := hasDrugClass(activeMeds, models.DrugClassACEInhibitor) || hasDrugClass(activeMeds, models.DrugClassARB)
	eGFRSlope := s.computeEGFRSlope(patientID)
	decliningEGFR := eGFRSlope != nil && *eGFRSlope < -3.0
	proj.ACEiARBHyperKDecliningEGFR = onACEiARB && proj.PotassiumCurrent >= 5.5 && decliningEGFR

	// PG-09: Beta-blocker + active insulin
	proj.BetaBlockerInsulinActive = hasDrugClass(activeMeds, models.DrugClassBetaBlocker) &&
		hasDrugClass(activeMeds, models.DrugClassInsulin)

	// PG-10: Resistant HTN (already detected and stored in BP trajectory)
	var bpTraj models.BPTrajectory
	if err := s.db.DB.Where("patient_id = ?", patientID).First(&bpTraj).Error; err == nil {
		proj.ResistantHTNDetected = bpTraj.Pattern == models.BPPatternResistant
	}

	// PG-11: Thiazide + Na+ < 132 (KDIGO threshold)
	proj.ThiazideHyponatraemia = hasDrugClass(activeMeds, models.DrugClassDiuretic) && proj.SodiumCurrent < 132

	// PG-13: CCB + SBP < 110 + recent dose increase
	ccbActive := hasDrugClass(activeMeds, models.DrugClassCCB)
	recentCCBChange := s.hasMedChangeInWindow(patientID, models.DrugClassCCB, 7*24*time.Hour)
	proj.CCBExcessiveResponse = ccbActive && proj.SBPCurrent < 110 && recentCCBChange

	// PG-14: RAAS creatinine tolerance
	proj.RAASCreatinineTolerant = s.isCreatinineRiseExplainedByRAAS(patientID, activeMeds)

	// Creatinine rise percentage
	proj.CreatinineRisePct = s.computeCreatinineRisePct(patientID)

	// AD-09: CKD Stage 4 deprescribing block (eGFR < 30)
	proj.CKDStage4DeprescribingBlocked = proj.EGFR > 0 && proj.EGFR < 30

	// ── Sprint parity fields ──

	// PG-17: ACR-RAAS escalation
	var acrTracking models.ACRTracking
	if err := s.db.DB.Where("patient_id = ?", patientID).First(&acrTracking).Error; err == nil {
		proj.ACRCategory = acrTracking.Category
		proj.ACRA3NoRAAS = acrTracking.Category == models.ACRCategoryA3 && !onACEiARB
		proj.ACRA2NoRAAS = acrTracking.Category == models.ACRCategoryA2 && !onACEiARB
	}

	// PG-18: Finerenone eligibility (FIDELIO-DKD 2020 criteria)
	// eGFR 25-75 + ACR A2/A3 + already on maximal ACEi/ARB
	onFinerenone := hasDrugClass(activeMeds, models.DrugClassFinerenone)
	acrEligible := proj.ACRCategory == models.ACRCategoryA2 || proj.ACRCategory == models.ACRCategoryA3
	proj.FinerenoneEligible = !onFinerenone && onACEiARB && acrEligible &&
		proj.EGFR >= 25 && proj.EGFR <= 75

	// PG-19: Finerenone K+ monitoring (FIGARO-DKD 2021)
	proj.FinerenoneKMonitoring = onFinerenone && proj.PotassiumCurrent >= 5.0

	return proj, nil
}

// ──────────────────────────────────────────────────────────────────────────
// Database query helpers
// ──────────────────────────────────────────────────────────────────────────

// latestLabValue returns the most recent accepted lab value of the given type.
func (s *ProjectionService) latestLabValue(patientID, labType string) (*float64, error) {
	var entry models.LabEntry
	err := s.db.DB.Where("patient_id = ? AND lab_type = ? AND validation_status = ?",
		patientID, labType, models.ValidationAccepted).
		Order("measured_at DESC").
		First(&entry).Error
	if err != nil {
		return nil, err
	}
	v, _ := entry.Value.Float64()
	return &v, nil
}

// latestLabValueWithTime returns the most recent lab value and its timestamp.
func (s *ProjectionService) latestLabValueWithTime(patientID, labType string) (*float64, *time.Time) {
	var entry models.LabEntry
	err := s.db.DB.Where("patient_id = ? AND lab_type = ? AND validation_status = ?",
		patientID, labType, models.ValidationAccepted).
		Order("measured_at DESC").
		First(&entry).Error
	if err != nil {
		return nil, nil
	}
	v, _ := entry.Value.Float64()
	return &v, &entry.MeasuredAt
}

// latestLabTimestamp returns the most recent measurement time for a lab type.
func (s *ProjectionService) latestLabTimestamp(patientID, labType string) *time.Time {
	var entry models.LabEntry
	err := s.db.DB.Where("patient_id = ? AND lab_type = ? AND validation_status = ?",
		patientID, labType, models.ValidationAccepted).
		Order("measured_at DESC").
		First(&entry).Error
	if err != nil {
		return nil
	}
	return &entry.MeasuredAt
}

// labValueAtOffset returns a lab value measured approximately `offset` ago.
// Looks for the closest measurement within ±12h of the target time.
func (s *ProjectionService) labValueAtOffset(patientID, labType string, offset time.Duration) *float64 {
	return s.labValueAtOffsetWithWindow(patientID, labType, offset, 12*time.Hour)
}

// labValueAtOffsetWithWindow returns a lab value measured approximately `offset` ago,
// within a configurable ±window around the target time. Tighter windows improve
// clinical accuracy (e.g., ±4h for creatinine 48h aligns with KDIGO AKI criteria).
func (s *ProjectionService) labValueAtOffsetWithWindow(patientID, labType string, offset, window time.Duration) *float64 {
	target := time.Now().UTC().Add(-offset)
	windowStart := target.Add(-window)
	windowEnd := target.Add(window)

	var entry models.LabEntry
	err := s.db.DB.Where("patient_id = ? AND lab_type = ? AND validation_status = ? AND measured_at BETWEEN ? AND ?",
		patientID, labType, models.ValidationAccepted, windowStart, windowEnd).
		Order("measured_at DESC").
		First(&entry).Error
	if err != nil {
		return nil
	}
	v, _ := entry.Value.Float64()
	return &v
}

// recentLabValues returns the N most recent lab values of a given type.
func (s *ProjectionService) recentLabValues(patientID, labType string, limit int) []models.TimestampedLabValue {
	var entries []models.LabEntry
	s.db.DB.Where("patient_id = ? AND lab_type = ? AND validation_status = ?",
		patientID, labType, models.ValidationAccepted).
		Order("measured_at DESC").
		Limit(limit).
		Find(&entries)

	result := make([]models.TimestampedLabValue, 0, len(entries))
	for _, e := range entries {
		v, _ := e.Value.Float64()
		result = append(result, models.TimestampedLabValue{
			Value:     v,
			Timestamp: e.MeasuredAt,
		})
	}
	return result
}

// getActiveMedications returns all active medications for a patient.
func (s *ProjectionService) getActiveMedications(patientID string) ([]models.MedicationState, error) {
	var meds []models.MedicationState
	err := s.db.DB.Where("patient_id = ? AND is_active = true", patientID).Find(&meds).Error
	return meds, err
}

// hasMedChangeInWindow checks if a medication of the given drug class had a
// dose change (via updated_at) within the specified time window.
func (s *ProjectionService) hasMedChangeInWindow(patientID, drugClass string, window time.Duration) bool {
	cutoff := time.Now().UTC().Add(-window)
	var count int64
	s.db.DB.Model(&models.MedicationState{}).
		Where("patient_id = ? AND drug_class = ? AND is_active = true AND updated_at > ?",
			patientID, drugClass, cutoff).
		Count(&count)
	return count > 0
}

// computeEGFRSlope returns the eGFR slope (mL/min/1.73m² per year) from the
// last 4 eGFR measurements. Returns nil if insufficient data.
func (s *ProjectionService) computeEGFRSlope(patientID string) *float64 {
	var entries []models.LabEntry
	s.db.DB.Where("patient_id = ? AND lab_type = ? AND validation_status = ?",
		patientID, models.LabTypeEGFR, models.ValidationAccepted).
		Order("measured_at DESC").
		Limit(4).
		Find(&entries)

	if len(entries) < 2 {
		return nil
	}

	// Simple linear regression: slope = Σ(xi-x̄)(yi-ȳ) / Σ(xi-x̄)²
	// x = time in years from first measurement, y = eGFR value
	earliest := entries[len(entries)-1].MeasuredAt
	n := float64(len(entries))
	var sumX, sumY, sumXY, sumX2 float64

	for _, e := range entries {
		x := e.MeasuredAt.Sub(earliest).Hours() / (24 * 365.25) // years
		y, _ := e.Value.Float64()
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	denominator := n*sumX2 - sumX*sumX
	if math.Abs(denominator) < 1e-10 {
		return nil
	}

	slope := (n*sumXY - sumX*sumY) / denominator
	return &slope
}

// computeCreatinineRisePct returns the % rise in creatinine from the reading
// 14 days ago to the latest reading. Returns 0 if insufficient data.
func (s *ProjectionService) computeCreatinineRisePct(patientID string) float64 {
	current, _ := s.latestLabValue(patientID, models.LabTypeCreatinine)
	baseline := s.labValueAtOffset(patientID, models.LabTypeCreatinine, 14*24*time.Hour)

	if current == nil || baseline == nil || *baseline == 0 {
		return 0
	}

	return ((*current - *baseline) / *baseline) * 100
}

// isCreatinineRiseExplainedByRAAS determines PG-14: whether a creatinine rise
// is within expected RAAS blockade pharmacodynamics. Criteria:
// - Patient is on ACEi/ARB
// - ACEi/ARB was started or uptitrated within 14 days
// - Creatinine rise 10-30% (below 10% is not RAAS-related, above 30% exceeds tolerance)
// - K+ < 5.5 mEq/L
// - No oliguria reported (oliguria overrides tolerance regardless of rise %)
func (s *ProjectionService) isCreatinineRiseExplainedByRAAS(patientID string, meds []models.MedicationState) bool {
	onACEiARB := hasDrugClass(meds, models.DrugClassACEInhibitor) || hasDrugClass(meds, models.DrugClassARB)
	if !onACEiARB {
		return false
	}

	recentRAASChange := s.hasMedChangeInWindow(patientID, models.DrugClassACEInhibitor, 14*24*time.Hour) ||
		s.hasMedChangeInWindow(patientID, models.DrugClassARB, 14*24*time.Hour)
	if !recentRAASChange {
		return false
	}

	risePct := s.computeCreatinineRisePct(patientID)
	// Rise must be in the 10-30% RAAS pharmacodynamic window
	if risePct < 10 || risePct >= 30 {
		return false
	}

	potassium, _ := s.latestLabValue(patientID, models.LabTypePotassium)
	if potassium != nil && *potassium >= 5.5 {
		return false
	}

	// Oliguria overrides tolerance — if oliguria is reported, creatinine rise
	// is NOT explained by RAAS even within the 10-30% window
	if s.isOliguriaReported(patientID) {
		return false
	}

	return true
}

// isOliguriaReported checks if the patient has an active oliguria flag.
// Oliguria overrides RAAS creatinine tolerance (PG-14 safety override).
func (s *ProjectionService) isOliguriaReported(patientID string) bool {
	var profile models.PatientProfile
	if err := s.db.DB.Where("patient_id = ?", patientID).First(&profile).Error; err != nil {
		return false
	}
	for _, c := range profile.Comorbidities {
		if c == "OLIGURIA" {
			return true
		}
	}
	return false
}

// computeLabStaleness builds a LabStaleness entry for a given lab type.
func (s *ProjectionService) computeLabStaleness(patientID, labType string, thresholdDays int, now time.Time) models.LabStaleness {
	ts := s.latestLabTimestamp(patientID, labType)
	if ts == nil {
		return models.LabStaleness{
			LastMeasuredAt: nil,
			StaleDays:      0,
			IsStale:        true, // never measured = stale
		}
	}
	staleDays := int(now.Sub(*ts).Hours() / 24)
	return models.LabStaleness{
		LastMeasuredAt: ts,
		StaleDays:      staleDays,
		IsStale:        staleDays >= thresholdDays,
	}
}

// latestLabValueFilteredSource returns the most recent accepted lab value,
// excluding entries from the given source. Used to filter PATIENT_REPORTED glucose.
func (s *ProjectionService) latestLabValueFilteredSource(patientID, labType, excludeSource string) (*float64, *time.Time, string) {
	var entry models.LabEntry
	err := s.db.DB.Where("patient_id = ? AND lab_type = ? AND validation_status = ? AND (source IS NULL OR source != ?)",
		patientID, labType, models.ValidationAccepted, excludeSource).
		Order("measured_at DESC").
		First(&entry).Error
	if err != nil {
		return nil, nil, ""
	}
	v, _ := entry.Value.Float64()
	return &v, &entry.MeasuredAt, entry.Source
}

// recentLabValuesFilteredSource returns the N most recent lab values, excluding a source.
func (s *ProjectionService) recentLabValuesFilteredSource(patientID, labType, excludeSource string, limit int) []models.TimestampedLabValue {
	var entries []models.LabEntry
	s.db.DB.Where("patient_id = ? AND lab_type = ? AND validation_status = ? AND (source IS NULL OR source != ?)",
		patientID, labType, models.ValidationAccepted, excludeSource).
		Order("measured_at DESC").
		Limit(limit).
		Find(&entries)

	result := make([]models.TimestampedLabValue, 0, len(entries))
	for _, e := range entries {
		v, _ := e.Value.Float64()
		result = append(result, models.TimestampedLabValue{
			Value:     v,
			Timestamp: e.MeasuredAt,
		})
	}
	return result
}

// ──────────────────────────────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────────────────────────────

// hasDrugClass checks if any medication in the list matches the given drug class,
// including FDC component decomposition.
func hasDrugClass(meds []models.MedicationState, drugClass string) bool {
	for _, m := range meds {
		for _, cls := range m.EffectiveDrugClasses() {
			if cls == drugClass {
				return true
			}
		}
	}
	return false
}

// effectiveDrugClasses returns all unique drug classes from a medication list,
// decomposing FDCs into their component classes.
func effectiveDrugClasses(meds []models.MedicationState) []string {
	seen := make(map[string]bool)
	for _, m := range meds {
		for _, cls := range m.EffectiveDrugClasses() {
			seen[cls] = true
		}
	}
	result := make([]string, 0, len(seen))
	for cls := range seen {
		result = append(result, cls)
	}
	return result
}

// jCurveSBPFloor returns the eGFR-stratified minimum SBP floor.
// Below this floor, BP lowering risks renal hypoperfusion.
func jCurveSBPFloor(ckdStage string) *float64 {
	var floor float64
	switch ckdStage {
	case "3a":
		floor = 120
	case "3b":
		floor = 125
	case "4":
		floor = 130
	case "5":
		floor = 135
	default:
		return nil // no J-curve floor for stages 1-2 or non-CKD
	}
	return &floor
}

func float64Ptr(v float64) *float64 {
	return &v
}

// Compile-time assertion that decimal.Decimal has Float64 method.
var _ = decimal.Decimal.Float64
