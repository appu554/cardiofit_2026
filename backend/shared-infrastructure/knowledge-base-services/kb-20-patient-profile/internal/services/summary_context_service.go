package services

import (
	"context"
	"fmt"
	"time"

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
// Phase 8 P8-1 (commit a7a099c3) shipped the first 10 fields and
// fixed the missing-endpoint bug. Phase 8 P8-2 (this commit) extends
// the wire contract with demographics, CKM stage V2 + substage
// metadata, potassium, engagement status, and CGM status — the full
// field set the Phase 7 retrospective called out as missing.
//
// Coupling note (Option α per the Phase 8 kickoff review): this struct
// is a deliberate mirror of KB-23's PatientContext, not a shared type.
// Matches the existing convention used by KB20RenalStatus and
// KB20InterventionTimeline in the KB-23 client — manual sync between
// the two sides, caught by the P8-1 integration test that asserts
// field-for-field compatibility. Drift on either side fails CI.
type SummaryContext struct {
	// ── P8-1 core fields (10) ──
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

	// ── P8-2 Demographics ──
	Age int     `json:"age,omitempty"`
	Sex string  `json:"sex,omitempty"`
	BMI float64 `json:"bmi,omitempty"`

	// ── P8-2 CKM stage + substage metadata ──
	CKMStageV2          string           `json:"ckm_stage_v2,omitempty"`
	CKMSubstageMetadata *CKMSubstageWire `json:"ckm_substage_metadata,omitempty"`

	// ── P8-2 Extended labs ──
	LatestPotassium float64 `json:"latest_potassium,omitempty"`

	// ── P8-2 Engagement / adherence ──
	EngagementComposite *float64 `json:"engagement_composite,omitempty"`
	EngagementStatus    string   `json:"engagement_status,omitempty"`

	// ── P8-2 CGM status (cross-service fetch from KB-26) ──
	HasCGM           bool       `json:"has_cgm,omitempty"`
	LatestCGMTIR     *float64   `json:"latest_cgm_tir,omitempty"`
	LatestCGMGRIZone string     `json:"latest_cgm_gri_zone,omitempty"`
	CGMReportAt      *time.Time `json:"cgm_report_at,omitempty"`

	// ── V4-7 Phenotype stability ──
	PhenotypeCluster string `json:"phenotype_cluster,omitempty"`

	// ── PAI attention dimension data sources ──
	// LastClinicianContactAt is the most recent timestamp when a clinician
	// interacted with this patient's record (protocol state update,
	// medication change, or lab review). Used by PAI's attention
	// dimension to compute DaysSinceLastClinician. Nil if no interaction
	// found — PAI treats nil as "no data, don't penalize" rather than
	// "never contacted, maximum urgency."
	LastClinicianContactAt *time.Time `json:"last_clinician_contact_at,omitempty"`
}

// CKMSubstageWire is the wire shape for the CKM substage metadata
// JSONB blob on the PatientProfile row. Mirrors a subset of
// models.SubstageMetadata — only the fields downstream card
// generation actually reads, matching the Option α field-by-field
// coupling rule. Full fidelity is available via the direct
// CKMSubstageMetadata JSONB query on KB-20 side if a caller
// needs more detail in the future.
type CKMSubstageWire struct {
	HFClassification string   `json:"hf_type,omitempty"`
	LVEFPercent      *float64 `json:"lvef_pct,omitempty"`
	NYHAClass        string   `json:"nyha_class,omitempty"`
	NTproBNP         *float64 `json:"nt_probnp,omitempty"`
	BNP              *float64 `json:"bnp,omitempty"`
	HFEtiology       string   `json:"hf_etiology,omitempty"`
	CACScore         *float64 `json:"cac_score,omitempty"`
	CIMTPercentile   *int     `json:"cimt_percentile,omitempty"`
	HasLVH           bool     `json:"has_lvh,omitempty"`
}

// CGMStatusFetcher is the narrow dependency the SummaryContextService
// uses to fetch the latest CGM period report from KB-26 without
// creating a hard dependency on the KB-26 package. Tests can inject a
// stub fetcher; production wires a real HTTP client to KB-26's
// P7-E Milestone 2 cgm-latest endpoint.
type CGMStatusFetcher interface {
	FetchLatestCGMStatus(ctx context.Context, patientID string) (*CGMStatusSnapshot, error)
}

// CGMStatusSnapshot is the narrow projection the summary-context
// service needs from KB-26's CGMPeriodReport. Kept minimal so the
// cross-service fetch carries only what the KB-23 consumer reads.
type CGMStatusSnapshot struct {
	HasCGM     bool
	TIRPct     float64
	GRIZone    string
	ReportedAt time.Time
}

// SummaryContextService assembles the cross-cutting patient snapshot
// that KB-23's card generation pipeline needs. Queries existing tables
// and services — it does not write anything, does not mutate state.
// Phase 8 P8-2: adds one cross-service fetch (KB-26 cgm-latest) for
// CGM status. Phase 8 P8-5: adds safety_events query for confounder
// flag population (IsAcuteIll, HasRecentTransfusion,
// HasRecentHypoglycaemia).
type SummaryContextService struct {
	db             *gorm.DB
	cgmFetcher     CGMStatusFetcher
	safetyRecorder *SafetyEventRecorder
	logger         *zap.Logger
}

// NewSummaryContextService wires the dependencies. cgmFetcher and
// safetyRecorder are both optional — pass nil in tests that don't
// need them, or when the dependency is not yet reachable. When nil,
// the corresponding fields on SummaryContext stay at their zero
// values and KB-23 consumers degrade cleanly (HasCGM=false,
// confounder flags all false, which biases toward surfacing cards
// rather than suppressing them).
func NewSummaryContextService(
	db *gorm.DB,
	cgmFetcher CGMStatusFetcher,
	safetyRecorder *SafetyEventRecorder,
	logger *zap.Logger,
) *SummaryContextService {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &SummaryContextService{
		db:             db,
		cgmFetcher:     cgmFetcher,
		safetyRecorder: safetyRecorder,
		logger:         logger,
	}
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
func (s *SummaryContextService) BuildContext(ctx context.Context, patientID string) (*SummaryContext, error) {
	if patientID == "" {
		return nil, fmt.Errorf("summary context: empty patient id")
	}

	var profile models.PatientProfile
	if err := s.db.Where("patient_id = ? AND active = ?", patientID, true).
		First(&profile).Error; err != nil {
		return nil, err
	}

	// P8-5: derive confounder flags from the safety_events audit
	// trail. A nil recorder produces all-false (same as the P8-2
	// default), and a recorder-level failure also produces all-
	// false — defensive so a DB hiccup on the safety table cannot
	// break the entire summary-context endpoint. The conservative
	// direction here is deliberate: all-false biases toward
	// surfacing cards rather than suppressing them, which is the
	// safer clinical direction when the confounder signal is
	// unavailable.
	var isAcuteIll, hasRecentTransfusion, hasRecentHypoglycaemia bool
	if s.safetyRecorder != nil {
		isAcuteIll, hasRecentTransfusion, hasRecentHypoglycaemia =
			s.safetyRecorder.ConfounderFlags(patientID, time.Now().UTC())
	}

	result := &SummaryContext{
		// P8-1 core
		PatientID:              profile.PatientID,
		Stratum:                profile.CVRiskCategory,
		WeightKg:               profile.WeightKg,
		IsAcuteIll:             isAcuteIll,
		HasRecentTransfusion:   hasRecentTransfusion,
		HasRecentHypoglycaemia: hasRecentHypoglycaemia,

		// P8-2 demographics — straight column reads
		Age: profile.Age,
		Sex: profile.Sex,
		BMI: profile.BMI,

		// P8-2 CKM stage — string column, JSONB metadata converted below
		CKMStageV2: profile.CKMStageV2,

		// P8-2 engagement — optional fields on the profile, passed through
		// as pointers when present
		EngagementComposite: profile.EngagementComposite,
		EngagementStatus:    profile.EngagementStatus,

		// V4-7: stable phenotype cluster from the stability engine
		PhenotypeCluster: profile.PhenotypeCluster,
	}
	if profile.EGFR != nil {
		result.EGFRValue = *profile.EGFR
	}

	// PAI: last clinician contact — most recent medication change or
	// protocol state update as proxy for clinician interaction.
	var lastContact time.Time
	if err := s.db.Raw(`
		SELECT COALESCE(MAX(t), '0001-01-01') FROM (
			SELECT MAX(updated_at) AS t FROM medication_states WHERE patient_id = ?
			UNION ALL
			SELECT MAX(updated_at) AS t FROM protocol_states WHERE patient_id = ?
		) sub`, patientID, patientID).Scan(&lastContact).Error; err == nil {
		if !lastContact.IsZero() {
			result.LastClinicianContactAt = &lastContact
		}
	}

	// P8-2 CKM substage metadata: convert the Go-side JSONB struct
	// into the narrower wire shape. Only populate if the profile
	// actually carries metadata (non-nil pointer) — CKM Stage 0-3
	// patients have no substage detail and KB-23 treats nil as
	// "no substage available."
	if profile.CKMSubstageMetadata != nil {
		meta := profile.CKMSubstageMetadata
		result.CKMSubstageMetadata = &CKMSubstageWire{
			HFClassification: string(meta.HFClassification),
			LVEFPercent:      meta.LVEFPercent,
			NYHAClass:        meta.NYHAClass,
			NTproBNP:         meta.NTproBNP,
			BNP:              meta.BNP,
			HFEtiology:       meta.HFEtiology,
			CACScore:         meta.CACScore,
			CIMTPercentile:   meta.CIMTPercentile,
			HasLVH:           meta.HasLVH,
		}
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
	result.Medications = cleaned

	// Latest HbA1c and FBG — most recent accepted lab value of each type.
	// Uses a single query per lab type because lab_entries may contain
	// both accepted and flagged readings and we want the most recent
	// clinically-usable one.
	result.LatestHbA1c = s.latestLabValue(patientID, models.LabTypeHbA1c)
	result.LatestFBG = s.latestLabValue(patientID, models.LabTypeFBG)

	// P8-2: Latest Potassium. Prefers the lab_entries row (freshest
	// accepted reading) over the PatientProfile.Potassium cached
	// column because the lab table is authoritative and the profile
	// column may lag FHIR sync by minutes.
	result.LatestPotassium = s.latestLabValue(patientID, models.LabTypePotassium)
	if result.LatestPotassium == 0 && profile.Potassium != nil {
		result.LatestPotassium = *profile.Potassium
	}

	// P8-2: CGM status via cross-service fetch to KB-26. The fetcher
	// is optional — when nil (tests without CGM infra, or KB-26 not
	// reachable), CGM fields stay at zero values and KB-23 consumers
	// fall back to the HbA1c glycaemic path. A fetch error is logged
	// but not fatal: it's preferable to return a slightly-degraded
	// SummaryContext than to 500 the entire card pipeline because
	// KB-26 is temporarily unreachable.
	if s.cgmFetcher != nil {
		snap, cgmErr := s.cgmFetcher.FetchLatestCGMStatus(ctx, patientID)
		if cgmErr != nil {
			s.logger.Debug("summary context: CGM fetch failed, falling back to HasCGM=false",
				zap.String("patient_id", patientID),
				zap.Error(cgmErr))
		} else if snap != nil && snap.HasCGM {
			result.HasCGM = true
			tir := snap.TIRPct
			result.LatestCGMTIR = &tir
			result.LatestCGMGRIZone = snap.GRIZone
			reportAt := snap.ReportedAt
			result.CGMReportAt = &reportAt
		}
	}

	// P8-5: Confounder flags (IsAcuteIll, HasRecentTransfusion,
	// HasRecentHypoglycaemia) are populated from the safety_events
	// audit trail via safetyRecorder.ConfounderFlags above. The
	// table is fed by lab_service safety event publish paths and
	// other clinical triggers — see safety_event_recorder.go for
	// the clinical window definitions (7d / 90d / 30d).

	return result, nil
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
