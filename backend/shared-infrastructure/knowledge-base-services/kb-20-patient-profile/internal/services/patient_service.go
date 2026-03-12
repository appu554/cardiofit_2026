package services

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"kb-patient-profile/internal/cache"
	"kb-patient-profile/internal/database"
	"kb-patient-profile/internal/models"
)

// antihypertensiveClasses lists the drug classes considered antihypertensive
// for resistant HTN classification. Diuretic is included here and also checked
// separately to satisfy the "at least one diuretic" criterion.
var antihypertensiveClasses = map[string]bool{
	models.DrugClassACEInhibitor: true,
	models.DrugClassARB:          true,
	models.DrugClassCCB:          true,
	models.DrugClassBetaBlocker:  true,
	models.DrugClassDiuretic:     true,
}

// MinResistantHTNWeeks is the minimum sustained duration (in weeks) of
// above-target BP required before classifying resistant hypertension.
const MinResistantHTNWeeks = 12

// MinResistantHTNDrugClasses is the minimum number of distinct antihypertensive
// drug classes at optimised doses required for resistant HTN classification.
const MinResistantHTNDrugClasses = 3

// MinAdherenceScore is the minimum adherence score (from KB-21) required.
// Below this threshold, non-adherence is the more likely explanation for
// uncontrolled BP, so the resistant HTN label should not be applied.
const MinAdherenceScore = 0.85

// PatientService handles patient profile CRUD and comorbidity derivation.
type PatientService struct {
	db       *database.Database
	cache    *cache.Client
	logger   *zap.Logger
	eventBus *EventBus
}

// NewPatientService creates a patient service.
func NewPatientService(db *database.Database, cacheClient *cache.Client, logger *zap.Logger) *PatientService {
	return &PatientService{db: db, cache: cacheClient, logger: logger}
}

// SetEventBus attaches the event bus to the patient service. This is called
// after construction to avoid circular dependency during service wiring.
func (s *PatientService) SetEventBus(eb *EventBus) {
	s.eventBus = eb
}

// Create stores a new patient profile.
func (s *PatientService) Create(profile *models.PatientProfile) error {
	if err := s.db.DB.Create(profile).Error; err != nil {
		return fmt.Errorf("failed to create patient profile: %w", err)
	}
	s.cache.Delete(cache.PatientProfilePrefix + profile.PatientID)
	return nil
}

// GetByID retrieves a patient profile by patient_id.
func (s *PatientService) GetByID(patientID string) (*models.PatientProfile, error) {
	// Try cache first
	var profile models.PatientProfile
	cacheKey := cache.PatientProfilePrefix + patientID
	if err := s.cache.Get(cacheKey, &profile); err == nil {
		return &profile, nil
	}

	if err := s.db.DB.Where("patient_id = ? AND active = true", patientID).First(&profile).Error; err != nil {
		return nil, fmt.Errorf("patient not found: %w", err)
	}

	s.cache.Set(cacheKey, &profile, cache.DefaultProfileTTL)
	return &profile, nil
}

// Update modifies a patient profile.
func (s *PatientService) Update(patientID string, updates map[string]interface{}) error {
	result := s.db.DB.Model(&models.PatientProfile{}).Where("patient_id = ?", patientID).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to update patient: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("patient not found: %s", patientID)
	}
	s.cache.Delete(cache.PatientProfilePrefix + patientID)
	return nil
}

// GetFullProfile returns the complete patient state including labs and medications.
func (s *PatientService) GetFullProfile(patientID string) (*models.PatientProfileResponse, error) {
	profile, err := s.GetByID(patientID)
	if err != nil {
		return nil, err
	}

	var labs []models.LabEntry
	s.db.DB.Where("patient_id = ? AND validation_status != 'REJECTED'", patientID).
		Order("measured_at DESC").Find(&labs)

	var medications []models.MedicationState
	s.db.DB.Where("patient_id = ? AND is_active = true", patientID).Find(&medications)

	// Find latest eGFR
	var latestEGFR *float64
	var ckdSubstage string
	for _, lab := range labs {
		if lab.LabType == models.LabTypeEGFR {
			val, _ := lab.Value.Float64()
			latestEGFR = &val
			engine := NewEGFREngine()
			ckdSubstage = engine.CKDStageFromEGFR(val)
			break
		}
	}

	return &models.PatientProfileResponse{
		Profile:     *profile,
		Labs:        labs,
		Medications: medications,
		LatestEGFR:  latestEGFR,
		CKDSubstage: ckdSubstage,
	}, nil
}

// DeriveComorbidities auto-derives comorbidities from lab values.
func (s *PatientService) DeriveComorbidities(patientID string, labs []models.LabEntry) []string {
	var comorbidities []string

	for _, lab := range labs {
		val, _ := lab.Value.Float64()
		switch lab.LabType {
		case models.LabTypeHbA1c:
			if val > 7.0 {
				comorbidities = append(comorbidities, "UNCONTROLLED_DM")
			}
		case models.LabTypeSBP:
			if val > 140 {
				comorbidities = append(comorbidities, "UNCONTROLLED_HTN")
			}
		case models.LabTypeTotalCholesterol:
			if val > 200 {
				comorbidities = append(comorbidities, "DYSLIPIDEMIA")
			}
		}
	}

	return comorbidities
}

// ResistantHTNInput bundles the data required by DetectResistantHTN.
// Callers (typically a pipeline or API handler) gather these values from
// the relevant services before invoking the detector.
type ResistantHTNInput struct {
	// BPTrajectory is the most recent 28-day BP analysis for the patient.
	BPTrajectory *models.BPTrajectory

	// ActiveMedications is the list of currently active medications.
	ActiveMedications []models.MedicationState

	// AdherenceScore is the composite adherence score from KB-21 (0.0–1.0).
	// A nil value means adherence data is unavailable; detection is skipped.
	AdherenceScore *float64

	// WeeksAboveTarget is the number of consecutive weeks the patient's BP
	// has remained above the guideline target. Computed externally from the
	// longitudinal SBP record.
	WeeksAboveTarget int
}

// DetectResistantHTN evaluates whether a patient meets the clinical definition
// of resistant hypertension and, if so, publishes a RESISTANT_HTN_DETECTED
// event via the transactional outbox.
//
// Resistant hypertension criteria (ESH/ESC 2023, AHA 2017):
//  1. BP above target on the most recent trajectory (BPPattern != CONTROLLED)
//  2. 3 or more distinct antihypertensive drug classes at optimised doses
//  3. At least one of those classes is a diuretic
//  4. Adherence >= 0.85 (to exclude pseudo-resistance from non-adherence)
//  5. Sustained for >= 12 consecutive weeks above target
//
// Returns (detected bool, err error). A false return with nil error means
// the patient does not currently meet the criteria.
func (s *PatientService) DetectResistantHTN(patientID string, input ResistantHTNInput) (bool, error) {
	// --- Guard: event bus must be wired ---
	if s.eventBus == nil {
		return false, fmt.Errorf("event bus not configured on PatientService")
	}

	// --- Criterion 1: BP above target ---
	if input.BPTrajectory == nil {
		s.logger.Debug("Resistant HTN check skipped: no BP trajectory available",
			zap.String("patient_id", patientID))
		return false, nil
	}
	if input.BPTrajectory.Pattern == models.BPPatternControlled ||
		input.BPTrajectory.Pattern == models.BPPatternUnknown {
		s.logger.Debug("Resistant HTN check: BP is controlled or unknown",
			zap.String("patient_id", patientID),
			zap.String("pattern", string(input.BPTrajectory.Pattern)))
		return false, nil
	}

	// --- Criterion 2 & 3: 3+ antihypertensive classes including a diuretic ---
	activeHTNClasses := make(map[string]bool)
	var diureticClass string
	for _, med := range input.ActiveMedications {
		for _, dc := range med.EffectiveDrugClasses() {
			if antihypertensiveClasses[dc] {
				activeHTNClasses[dc] = true
			}
			if dc == models.DrugClassDiuretic {
				diureticClass = med.DrugName
			}
		}
	}

	if len(activeHTNClasses) < MinResistantHTNDrugClasses {
		s.logger.Debug("Resistant HTN check: insufficient antihypertensive classes",
			zap.String("patient_id", patientID),
			zap.Int("classes_found", len(activeHTNClasses)),
			zap.Int("required", MinResistantHTNDrugClasses))
		return false, nil
	}

	if diureticClass == "" {
		s.logger.Debug("Resistant HTN check: no diuretic in active medications",
			zap.String("patient_id", patientID))
		return false, nil
	}

	// --- Criterion 4: Adherence >= 0.85 ---
	if input.AdherenceScore == nil {
		s.logger.Debug("Resistant HTN check skipped: adherence data unavailable",
			zap.String("patient_id", patientID))
		return false, nil
	}
	if *input.AdherenceScore < MinAdherenceScore {
		s.logger.Debug("Resistant HTN check: adherence below threshold (pseudo-resistance likely)",
			zap.String("patient_id", patientID),
			zap.Float64("adherence", *input.AdherenceScore),
			zap.Float64("threshold", MinAdherenceScore))
		return false, nil
	}

	// --- Criterion 5: Sustained >= 12 weeks ---
	if input.WeeksAboveTarget < MinResistantHTNWeeks {
		s.logger.Debug("Resistant HTN check: duration below threshold",
			zap.String("patient_id", patientID),
			zap.Int("weeks", input.WeeksAboveTarget),
			zap.Int("required", MinResistantHTNWeeks))
		return false, nil
	}

	// --- All criteria met: emit event ---
	var classList []string
	for dc := range activeHTNClasses {
		classList = append(classList, dc)
	}

	payload := models.ResistantHTNDetectedPayload{
		PatientID:         patientID,
		ActiveDrugClasses: classList,
		DiureticClass:     diureticClass,
		AdherenceScore:    *input.AdherenceScore,
		WeeksAboveTarget:  input.WeeksAboveTarget,
		BPStatus:          string(input.BPTrajectory.Pattern),
		DetectedAt:        time.Now().UTC(),
	}

	s.eventBus.Publish(models.EventResistantHTNDetected, patientID, payload)

	s.logger.Info("Resistant hypertension detected",
		zap.String("patient_id", patientID),
		zap.Strings("drug_classes", classList),
		zap.String("diuretic", diureticClass),
		zap.Float64("adherence", *input.AdherenceScore),
		zap.Int("weeks_above_target", input.WeeksAboveTarget),
		zap.String("bp_pattern", string(input.BPTrajectory.Pattern)))

	return true, nil
}

// ---------------------------------------------------------------------------
// AD-01: Adherence Pre-Condition Gate for Antihypertensive Deprescribing
// ---------------------------------------------------------------------------

// HTNDeprescribingAdherenceThreshold is the minimum 16-week antihypertensive
// adherence score required to enter DEPRESCRIBING_MODE.
const HTNDeprescribingAdherenceThreshold = 0.85

// HTNDeprescribingReviewThreshold is the adherence floor below which a
// medication review card is generated instead of deprescribing entry.
const HTNDeprescribingReviewThreshold = 0.70

// EvaluateHTNDeprescribingEligibility checks whether a patient may enter
// DEPRESCRIBING_MODE for antihypertensives based on the 16-week adherence
// score provided by KB-21.
//
// Decision logic:
//   - AD-09: eGFR < 30 (CKD Stage 4/5) → HARD BLOCK (never deprescribe)
//   - adherenceScore >= 0.85 → eligible (deprescribing may proceed)
//   - adherenceScore >= 0.70 → not eligible; generate MEDICATION_REVIEW card
//   - adherenceScore <  0.70 → not eligible; no action (adherence too low)
func (s *PatientService) EvaluateHTNDeprescribingEligibility(
	ctx context.Context,
	patientID string,
	adherenceScore float64,
) (eligible bool, alternativeAction string) {
	// ── AD-09: CKD Stage 4/5 hard block — never deprescribe antihypertensives ──
	// Near-dialysis kidney (eGFR < 30) has critically narrow BP window;
	// J-curve floor is already at 110 mmHg. Any drug removal risks renal
	// perfusion collapse below the J-curve nadir. This is an absolute
	// contraindication regardless of adherence or BP stability.
	// Reference: KDIGO 2021 CKD Management, Stage 4/5 safety exclusion.
	var latestEGFRLab models.LabEntry
	if err := s.db.DB.WithContext(ctx).
		Where("patient_id = ? AND lab_type = ? AND validation_status != 'REJECTED'",
			patientID, models.LabTypeEGFR).
		Order("measured_at DESC").
		First(&latestEGFRLab).Error; err == nil {
		eGFR, _ := latestEGFRLab.Value.Float64()
		if eGFR < 30 {
			s.logger.Warn("AD-09: CKD Stage 4/5 hard block — antihypertensive deprescribing contraindicated",
				zap.String("patient_id", patientID),
				zap.Float64("egfr", eGFR))
			return false, "CKD_STAGE_4_BLOCK"
		}
	}

	if adherenceScore >= HTNDeprescribingAdherenceThreshold {
		s.logger.Info("AD-01: patient eligible for HTN deprescribing",
			zap.String("patient_id", patientID),
			zap.Float64("adherence_score", adherenceScore))
		return true, ""
	}
	if adherenceScore >= HTNDeprescribingReviewThreshold {
		s.logger.Info("AD-01: adherence borderline — medication review recommended",
			zap.String("patient_id", patientID),
			zap.Float64("adherence_score", adherenceScore))
		return false, "MEDICATION_REVIEW"
	}
	s.logger.Info("AD-01: adherence too low for deprescribing consideration",
		zap.String("patient_id", patientID),
		zap.Float64("adherence_score", adherenceScore))
	return false, "NO_ACTION"
}

// ---------------------------------------------------------------------------
// AD-03: SGLT2i Buffer Check
// ---------------------------------------------------------------------------

// SGLT2iBufferWeeks is the number of weeks after SGLT2i discontinuation
// during which the vasodilatory BP buffer may still be waning.
const SGLT2iBufferWeeks = 6

// CheckSGLT2iBuffer returns true if the patient's SGLT2i was stopped within
// the last 6 weeks, meaning BP may rise as the vasodilatory buffer is lost.
// Antihypertensive deprescribing should be blocked during this window.
func (s *PatientService) CheckSGLT2iBuffer(
	ctx context.Context,
	patientID string,
) (blocked bool, reason string) {
	var meds []models.MedicationState
	if err := s.db.DB.WithContext(ctx).
		Where("patient_id = ? AND drug_class = ? AND is_active = false AND end_date IS NOT NULL",
			patientID, models.DrugClassSGLT2I).
		Order("end_date DESC").
		Limit(1).
		Find(&meds).Error; err != nil {
		s.logger.Error("AD-03: failed to query SGLT2i history",
			zap.String("patient_id", patientID),
			zap.Error(err))
		// Fail-safe: do not block if we cannot determine status.
		return false, ""
	}

	if len(meds) == 0 {
		return false, ""
	}

	if meds[0].EndDate == nil {
		return false, ""
	}

	weeksSinceStopped := int(time.Since(*meds[0].EndDate).Hours() / (24 * 7))
	if weeksSinceStopped < SGLT2iBufferWeeks {
		s.logger.Warn("AD-03: SGLT2i recently stopped — deprescribing blocked",
			zap.String("patient_id", patientID),
			zap.String("drug_name", meds[0].DrugName),
			zap.Int("weeks_since_stopped", weeksSinceStopped))
		return true, "SGLT2I_RECENTLY_STOPPED_BP_BUFFER_LOST"
	}

	return false, ""
}

// ---------------------------------------------------------------------------
// AD-02: Lifestyle Attribution Bonus (Wave 3.6)
// ---------------------------------------------------------------------------

// DefaultDeprescribingEntryWeeks is the standard deprescribing entry window.
const DefaultDeprescribingEntryWeeks = 16

// ReducedDeprescribingEntryWeeks is the shortened entry window when the
// lifestyle attribution bonus applies.
const ReducedDeprescribingEntryWeeks = 12

// ApplyLifestyleAttributionBonus checks if the deprescribing entry window
// can be reduced from 16 weeks to 12 weeks based on dietary improvement.
// Condition: salt_reduction_potential changed HIGH→LOW during the same period AND bp_status AT_TARGET.
func (s *PatientService) ApplyLifestyleAttributionBonus(
	previousSaltReduction float64,
	currentSaltReduction float64,
	bpStatus string,
) int {
	if previousSaltReduction >= 0.6 && currentSaltReduction < 0.3 && bpStatus == "AT_TARGET" {
		s.logger.Info("AD-02: lifestyle attribution bonus applied — entry window reduced",
			zap.Float64("previous_salt_reduction", previousSaltReduction),
			zap.Float64("current_salt_reduction", currentSaltReduction),
			zap.String("bp_status", bpStatus),
			zap.Int("entry_weeks", ReducedDeprescribingEntryWeeks))
		return ReducedDeprescribingEntryWeeks
	}
	return DefaultDeprescribingEntryWeeks
}

// ---------------------------------------------------------------------------
// AD-08: CKD Stage Constraints on Antihypertensive Deprescribing
// ---------------------------------------------------------------------------

// CKDDeprescribingConstraints holds the per-stage rules that modify the
// standard deprescribing protocol for patients with CKD.
type CKDDeprescribingConstraints struct {
	// MonitoringWeeks is the number of weeks to monitor after dose halving
	// (overrides the per-drug-class default when non-zero).
	MonitoringWeeks int

	// ThiazideRemovalAllowed indicates whether Step 2 (full removal) of a
	// thiazide is permitted. When false, removal is conditional on the
	// SBP floor check.
	ThiazideRemovalAllowed bool

	// ThiazideRemovalSBPFloor is the minimum 7-day mean SBP at half-dose
	// required before thiazide removal is allowed. Only relevant when
	// ThiazideRemovalAllowed is false.
	ThiazideRemovalSBPFloor float64
}

// GetCKDDeprescribingConstraints returns deprescribing constraints based on
// the patient's CKD stage.
//
// Stage 3b rules:
//   - Step 1 (dose halving) is allowed for all classes.
//   - Step 2 (full thiazide removal) is blocked unless sbp_7d_mean >= 110
//     at the halved dose.
//   - Monitoring windows are extended to 6 weeks for ALL drug classes.
//
// Stage 3a rules:
//   - Standard hierarchy but extended monitoring (6 weeks for all).
//
// All other stages use the standard per-class monitoring windows.
func GetCKDDeprescribingConstraints(ckdStage string) CKDDeprescribingConstraints {
	switch ckdStage {
	case "3b":
		return CKDDeprescribingConstraints{
			MonitoringWeeks:         6,
			ThiazideRemovalAllowed:  false,
			ThiazideRemovalSBPFloor: 110.0,
		}
	case "3a":
		return CKDDeprescribingConstraints{
			MonitoringWeeks:        6,
			ThiazideRemovalAllowed: true,
		}
	default:
		return CKDDeprescribingConstraints{
			MonitoringWeeks:        0, // use per-class default
			ThiazideRemovalAllowed: true,
		}
	}
}
