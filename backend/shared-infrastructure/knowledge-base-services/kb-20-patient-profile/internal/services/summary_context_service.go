package services

import (
	"fmt"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"kb-patient-profile/internal/models"
)

// SummaryContext is the JSON envelope returned by KB-20's
// GET /patient/:id/summary-context endpoint. The field names and JSON
// tags are deliberately matched to KB-23's services.PatientContext
// struct (defined in kb-23-decision-cards/internal/services/mcu_gate_manager.go)
// so the KB-23 HTTP client can deserialize this payload directly.
//
// Phase 8 P8-1: this is the missing endpoint that gated every Phase 7
// card-generation code path in production. KB23Client.FetchSummaryContext
// has been calling GET /patient/:id/summary-context since Phase 6, but
// no handler ever backed the route — every consumer silently 404'd and
// the card pipeline produced nothing for real patients. This service
// closes the loop.
//
// Coupling note (Option α per the Phase 8 kickoff review): this struct
// is a deliberate mirror of KB-23's PatientContext, not a shared type.
// Matches the existing convention used by KB20RenalStatus and
// KB20InterventionTimeline in the KB-23 client — manual sync between
// the two sides, caught by the P8-1 integration test that asserts
// field-for-field compatibility.
type SummaryContext struct {
	PatientID              string   `json:"patient_id"`
	Stratum                string   `json:"stratum"`
	Medications            []string `json:"medications"`
	EGFRValue              float64  `json:"egfr_value"`
	LatestHbA1c            float64  `json:"latest_hba1c"`
	LatestFBG              float64  `json:"latest_fbg"`
	IsAcuteIll             bool     `json:"is_acute_illness"`
	HasRecentTransfusion   bool     `json:"has_recent_transfusion"`
	HasRecentHypoglycaemia bool     `json:"has_recent_hypoglycaemia"`
	WeightKg               float64  `json:"weight_kg"`
}

// SummaryContextService assembles the cross-cutting patient snapshot
// that KB-23's card generation pipeline needs. Queries existing tables
// and services — it does not write anything, does not mutate state,
// does not invoke external services. Pure read-path aggregation.
//
// Phase 8 P8-1.
type SummaryContextService struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewSummaryContextService wires the dependencies.
func NewSummaryContextService(db *gorm.DB, logger *zap.Logger) *SummaryContextService {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &SummaryContextService{db: db, logger: logger}
}

// BuildContext assembles a SummaryContext for the given patient.
// Returns (nil, gorm.ErrRecordNotFound) when the patient profile
// does not exist — the handler maps this to 404. Partial data
// (e.g., no recent HbA1c, no active medications) populates the
// corresponding fields with zero values — the KB-23 consumers treat
// zero as "no signal" and degrade gracefully.
//
// Field population strategy:
//   - PatientID/Stratum/WeightKg/EGFRValue: PatientProfile columns
//   - Medications: distinct drug_class from active medication_states
//   - LatestHbA1c/LatestFBG: most recent accepted lab_entries value
//   - IsAcuteIll/HasRecentTransfusion/HasRecentHypoglycaemia: false
//     for now. These are confounder flags that the Phase 7 MCU gate
//     manager reads but KB-20 does not yet track — a Phase 8 follow-up
//     (P8-2 or later) will populate them from recent safety_events
//     and priority-events audit trails. Defaulting to false biases
//     toward surfacing cards, not suppressing them — safer than the
//     alternative.
func (s *SummaryContextService) BuildContext(patientID string) (*SummaryContext, error) {
	if patientID == "" {
		return nil, fmt.Errorf("summary context: empty patient id")
	}

	var profile models.PatientProfile
	if err := s.db.Where("patient_id = ? AND active = ?", patientID, true).
		First(&profile).Error; err != nil {
		return nil, err
	}

	ctx := &SummaryContext{
		PatientID:              profile.PatientID,
		Stratum:                profile.CVRiskCategory,
		WeightKg:               profile.WeightKg,
		IsAcuteIll:             false,
		HasRecentTransfusion:   false,
		HasRecentHypoglycaemia: false,
	}
	if profile.EGFR != nil {
		ctx.EGFRValue = *profile.EGFR
	}

	// Active medications — distinct drug classes from the medication
	// state table. Uses Pluck + Distinct so a patient on metformin +
	// metformin XR produces one METFORMIN entry, not two.
	var medClasses []string
	if err := s.db.Table("medication_states").
		Where("patient_id = ? AND is_active = ?", patientID, true).
		Distinct("drug_class").
		Pluck("drug_class", &medClasses).Error; err != nil {
		s.logger.Warn("failed to fetch active medications for summary context",
			zap.String("patient_id", patientID),
			zap.Error(err))
		medClasses = []string{}
	}
	// Drop empty strings that may have slipped in via a malformed sync.
	cleaned := make([]string, 0, len(medClasses))
	for _, m := range medClasses {
		if m != "" {
			cleaned = append(cleaned, m)
		}
	}
	ctx.Medications = cleaned

	// Latest HbA1c and FBG — most recent accepted lab value of each type.
	// Uses a single query per lab type because lab_entries may contain
	// both accepted and flagged readings and we want the most recent
	// clinically-usable one.
	ctx.LatestHbA1c = s.latestLabValue(patientID, models.LabTypeHbA1c)
	ctx.LatestFBG = s.latestLabValue(patientID, models.LabTypeFBG)

	return ctx, nil
}

// latestLabValue returns the most recent accepted lab reading of the
// given type for a patient, or 0 if none exists. 0 is the KB-23
// convention for "no signal" — the MCU gate manager checks
// `ctx.LatestHbA1c > 0` before reading the value.
func (s *SummaryContextService) latestLabValue(patientID, labType string) float64 {
	var entry models.LabEntry
	err := s.db.
		Where("patient_id = ? AND lab_type = ? AND validation_status = ?",
			patientID, labType, models.ValidationAccepted).
		Order("measured_at DESC").
		First(&entry).Error
	if err != nil {
		return 0
	}
	val, _ := entry.Value.Float64()
	return val
}
