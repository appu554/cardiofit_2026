package services

import (
	"fmt"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"kb-patient-profile/internal/models"
)

// CKMRecomputationService reassembles a patient's CKMClassifierInput from
// the current PatientProfile + recent lab entries, runs ClassifyCKMStage,
// and — if the stage differs from the patient's persisted CKMStageV2 —
// hands the result off to CKMTransitionPublisher.
//
// Phase 7 P7-B: this service resolves the upstream gap flagged in the
// Phase 6 retrospective, where CKMTransitionPublisher.PublishStageTransition
// existed but had no production caller. The FHIR sync worker now invokes
// this service whenever a CKM-relevant observation (LVEF, NT_PROBNP, CAC)
// lands, so stage transitions fire reactively on the real clinical data
// path — not just via manual HTTP test calls.
//
// Scope note: the classifier assembly below is deliberately minimal.
// It covers the four most clinically-significant inputs (LVEF-driven
// 4c, CAC-driven 4a, eGFR-driven CKD staging, diabetes/HTN comorbidities),
// which is sufficient to detect every transition a new LVEF/NT_PROBNP/
// CAC observation can cause. A richer assembly that incorporates ASCVD
// events, PREVENT scores, LDL, and NYHA class is a Phase 8 follow-up
// once the FHIR Condition sync lands.
type CKMRecomputationService struct {
	db        *gorm.DB
	publisher *CKMTransitionPublisher
	logger    *zap.Logger
}

// NewCKMRecomputationService wires the dependencies. Both db and publisher
// are required — a nil publisher would silently drop every transition.
func NewCKMRecomputationService(db *gorm.DB, publisher *CKMTransitionPublisher, logger *zap.Logger) *CKMRecomputationService {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &CKMRecomputationService{db: db, publisher: publisher, logger: logger}
}

// RecomputeAndPublish reads the patient profile + recent labs, assembles a
// CKMClassifierInput, runs ClassifyCKMStage, and publishes a transition
// event if the computed stage differs from the persisted one. Returns
// (transitioned, error). Callers should treat transitioned=false on a
// successful (nil error) return as the normal steady-state — it just
// means the new observation did not move the patient across a staging
// boundary.
//
// triggeringEventID identifies the observation/event that caused the
// recomputation and is carried on the resulting CKM_STAGE_TRANSITION
// event payload so downstream consumers can trace cause-and-effect.
func (s *CKMRecomputationService) RecomputeAndPublish(patientID, triggeringEventID string) (bool, error) {
	if patientID == "" {
		return false, nil
	}
	if s.db == nil || s.publisher == nil {
		s.logger.Warn("CKM recomputation skipped: db or publisher not wired",
			zap.String("patient_id", patientID))
		return false, nil
	}

	var profile models.PatientProfile
	if err := s.db.Where("patient_id = ?", patientID).First(&profile).Error; err != nil {
		return false, fmt.Errorf("fetch patient profile for CKM recomputation: %w", err)
	}

	input := s.buildClassifierInput(&profile)
	result := ClassifyCKMStage(input)

	transitioned, err := s.publisher.PublishStageTransition(
		patientID,
		string(result.Stage),
		result.StagingRationale,
		string(result.Metadata.HFClassification),
		"OBSERVATION",
		triggeringEventID,
	)
	if err != nil {
		return false, fmt.Errorf("publish CKM transition: %w", err)
	}

	if transitioned {
		s.logger.Info("CKM stage transition detected by recomputation",
			zap.String("patient_id", patientID),
			zap.String("from_stage", profile.CKMStageV2),
			zap.String("to_stage", string(result.Stage)),
			zap.String("triggering_event", triggeringEventID))
	} else {
		s.logger.Debug("CKM recomputation completed without transition",
			zap.String("patient_id", patientID),
			zap.String("stage", string(result.Stage)))
	}
	return transitioned, nil
}

// buildClassifierInput maps a PatientProfile + recent lab entries into
// the CKMClassifierInput struct consumed by ClassifyCKMStage. Kept as
// a method on the service so tests can inject a stub db and verify the
// field mapping in isolation.
func (s *CKMRecomputationService) buildClassifierInput(profile *models.PatientProfile) CKMClassifierInput {
	hasHF := s.hasHeartFailureFromComorbidities(profile.Comorbidities)
	// CKMStageV2 already carrying "4c" also implies HF — this prevents an
	// existing 4c patient from regressing to 4a just because the recent
	// LVEF reading was missing when the recomputation ran.
	if profile.CKMStageV2 == string(models.CKMStageV2_4c) {
		hasHF = true
	}

	latestLVEF := s.latestLabValue(profile.PatientID, models.LabTypeLVEF)
	latestNTproBNP := s.latestLabValue(profile.PatientID, models.LabTypeNTproBNP)
	latestCAC := s.latestLabValue(profile.PatientID, models.LabTypeCACScore)

	// If the patient has no HF coded but a recent LVEF is severely reduced
	// (≤40%), we flag HF presence for the classifier so it can produce a
	// Stage 4c result. This is intentionally conservative — a single low
	// LVEF reading is a strong enough signal to escalate staging; the
	// clinician still has the final word via DecisionCard review.
	if !hasHF && latestLVEF != nil && *latestLVEF <= 40 {
		hasHF = true
	}

	egfr := 0.0
	if profile.EGFR != nil {
		egfr = *profile.EGFR
	}

	return CKMClassifierInput{
		Age:             profile.Age,
		Sex:             profile.Sex,
		BMI:             profile.BMI,
		HasDiabetes:     profile.DMType == "T1DM" || profile.DMType == "T2DM",
		HasHTN:          profile.HTNStatus == "CONFIRMED",
		HbA1c:           profile.HbA1c,
		EGFR:            egfr,
		ACR:             profile.UACR,
		HasHeartFailure: hasHF,
		LVEF:            latestLVEF,
		NTproBNP:        latestNTproBNP,
		CACScore:        latestCAC,
	}
}

// hasHeartFailureFromComorbidities uses the same set of HF comorbidity
// labels as StratumEngine + TargetEngine — "HF", "HEART_FAILURE",
// "HFrEF", "HFpEF", "HFmrEF" — so classifier input assembly stays
// consistent with the rest of KB-20.
func (s *CKMRecomputationService) hasHeartFailureFromComorbidities(comorbidities []string) bool {
	for _, c := range comorbidities {
		switch c {
		case "HF", "HEART_FAILURE", "HFrEF", "HFpEF", "HFmrEF":
			return true
		}
	}
	return false
}

// latestLabValue returns the most recent accepted lab reading of the
// given type for a patient, or nil if none exists. Used to feed the
// classifier with current LVEF / NT-proBNP / CAC score values.
func (s *CKMRecomputationService) latestLabValue(patientID, labType string) *float64 {
	if s.db == nil {
		return nil
	}
	var entry models.LabEntry
	err := s.db.
		Where("patient_id = ? AND lab_type = ? AND validation_status = ?",
			patientID, labType, models.ValidationAccepted).
		Order("measured_at DESC").
		First(&entry).Error
	if err != nil {
		return nil
	}
	val, _ := entry.Value.Float64()
	return &val
}
