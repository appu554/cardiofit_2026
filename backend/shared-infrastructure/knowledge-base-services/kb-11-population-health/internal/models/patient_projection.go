// Package models provides domain models for KB-11 Population Health Engine.
//
// CRITICAL: KB-11 is a "Population Intelligence Layer" - NOT a patient registry.
// North Star: "KB-11 answers population-level questions, NOT patient-level decisions."
//
// Data Ownership Rules:
// - Patient demographics: CONSUMED from FHIR Store/KB-17 (NOT authoritative)
// - Risk assessments: OWNED by KB-11, GOVERNED by KB-18
// - Care gaps: CONSUMED from KB-13 (aggregated counts only)
// - Attribution data: OWNED by KB-11 (PCP assignments, practice attribution)
package models

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// PatientProjection represents a denormalized view of patient data for population analytics.
// This is NOT the source of truth - data is synced from FHIR Store and KB-17 Registry.
// KB-11 CONSUMES patient data, it does NOT own it.
type PatientProjection struct {
	ID uuid.UUID `json:"id" db:"id"`

	// External references (source of truth is elsewhere)
	FHIRID      string     `json:"fhir_id" db:"fhir_id"`           // From FHIR Store (authoritative)
	KB17PatientID *uuid.UUID `json:"kb17_patient_id,omitempty" db:"kb17_patient_id"` // From KB-17 Registry
	MRN         *string    `json:"mrn,omitempty" db:"mrn"`         // Medical Record Number

	// Cached demographics (synced from upstream - NOT authoritative)
	FirstName   *string    `json:"first_name,omitempty" db:"first_name"`
	LastName    *string    `json:"last_name,omitempty" db:"last_name"`
	DateOfBirth *time.Time `json:"date_of_birth,omitempty" db:"date_of_birth"`
	Gender      *Gender    `json:"gender,omitempty" db:"gender"`

	// Attribution overlay (KB-11 OWNS this)
	AttributedPCP      *string    `json:"attributed_pcp,omitempty" db:"attributed_pcp"`
	AttributedPractice *string    `json:"attributed_practice,omitempty" db:"attributed_practice"`
	AttributionDate    *time.Time `json:"attribution_date,omitempty" db:"attribution_date"`

	// Computed fields (KB-11 OWNS these - calculated locally)
	CurrentRiskTier  RiskTier `json:"current_risk_tier" db:"current_risk_tier"`
	LatestRiskScore  *float64 `json:"latest_risk_score,omitempty" db:"latest_risk_score"`

	// Aggregated from KB-13 (NOT source of truth - cached count only)
	CareGapCount int `json:"care_gap_count" db:"care_gap_count"`

	// Sync metadata
	LastSyncedAt time.Time  `json:"last_synced_at" db:"last_synced_at"`
	SyncSource   SyncSource `json:"sync_source" db:"sync_source"`
	SyncVersion  int        `json:"sync_version" db:"sync_version"`

	// Audit fields
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// RiskAssessment represents a risk score calculation for a patient.
// KB-11 OWNS this data, but it is GOVERNED by KB-18.
// Every calculation must be deterministic and auditable.
type RiskAssessment struct {
	ID            uuid.UUID `json:"id" db:"id"`
	PatientFHIRID string    `json:"patient_fhir_id" db:"patient_fhir_id"`

	// Model governance (KB-18 integration)
	ModelName    string `json:"model_name" db:"model_name"`
	ModelVersion string `json:"model_version" db:"model_version"`

	// Score data
	Score               float64            `json:"score" db:"score"`
	RiskTier            RiskTier           `json:"risk_tier" db:"risk_tier"`
	ContributingFactors map[string]float64 `json:"contributing_factors,omitempty" db:"contributing_factors"`

	// Determinism guarantee (CRITICAL for enterprise review)
	InputHash       string `json:"input_hash" db:"input_hash"`             // SHA-256 of input data
	CalculationHash string `json:"calculation_hash" db:"calculation_hash"` // SHA-256 of score computation

	// Governance emission (KB-18 reference)
	GovernanceEventID *uuid.UUID `json:"governance_event_id,omitempty" db:"governance_event_id"`

	// Validity period
	CalculatedAt time.Time  `json:"calculated_at" db:"calculated_at"`
	ValidUntil   *time.Time `json:"valid_until,omitempty" db:"valid_until"`
}

// RiskAssessmentHistory maintains the audit trail of all risk calculations.
type RiskAssessmentHistory struct {
	ID                  uuid.UUID          `json:"id" db:"id"`
	AssessmentID        uuid.UUID          `json:"assessment_id" db:"assessment_id"`
	PatientFHIRID       string             `json:"patient_fhir_id" db:"patient_fhir_id"`
	ModelName           string             `json:"model_name" db:"model_name"`
	ModelVersion        string             `json:"model_version" db:"model_version"`
	Score               float64            `json:"score" db:"score"`
	RiskTier            RiskTier           `json:"risk_tier" db:"risk_tier"`
	ContributingFactors map[string]float64 `json:"contributing_factors,omitempty" db:"contributing_factors"`
	InputHash           string             `json:"input_hash" db:"input_hash"`
	CalculationHash     string             `json:"calculation_hash" db:"calculation_hash"`
	GovernanceEventID   *uuid.UUID         `json:"governance_event_id,omitempty" db:"governance_event_id"`
	CalculatedAt        time.Time          `json:"calculated_at" db:"calculated_at"`
	ArchivedAt          time.Time          `json:"archived_at" db:"archived_at"`
}

// SyncStatusRecord tracks synchronization with upstream sources.
type SyncStatusRecord struct {
	ID                uuid.UUID   `json:"id" db:"id"`
	Source            SyncSource  `json:"source" db:"source"`
	LastSyncStarted   *time.Time  `json:"last_sync_started,omitempty" db:"last_sync_started"`
	LastSyncCompleted *time.Time  `json:"last_sync_completed,omitempty" db:"last_sync_completed"`
	LastSyncStatus    SyncStatus  `json:"last_sync_status" db:"last_sync_status"`
	RecordsSynced     int         `json:"records_synced" db:"records_synced"`
	ErrorMessage      *string     `json:"error_message,omitempty" db:"error_message"`
	CreatedAt         time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time   `json:"updated_at" db:"updated_at"`
}

// RiskInput represents the input data for risk calculation.
// Used for determinism verification via hashing.
type RiskInput struct {
	PatientFHIRID string                 `json:"patient_fhir_id"`
	ModelName     string                 `json:"model_name"`
	ModelVersion  string                 `json:"model_version"`
	Demographics  map[string]interface{} `json:"demographics"`
	Conditions    []string               `json:"conditions"`
	Medications   []string               `json:"medications"`
	LabValues     map[string]float64     `json:"lab_values"`
	Timestamp     time.Time              `json:"timestamp"`
}

// Hash generates a deterministic SHA-256 hash of the input.
func (ri *RiskInput) Hash() string {
	data, _ := json.Marshal(ri)
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

// RiskOutput represents the output of a risk calculation.
type RiskOutput struct {
	Score               float64            `json:"score"`
	RiskTier            RiskTier           `json:"risk_tier"`
	ContributingFactors map[string]float64 `json:"contributing_factors"`
	Confidence          float64            `json:"confidence"`
}

// Hash generates a deterministic SHA-256 hash of the output.
func (ro *RiskOutput) Hash() string {
	data, _ := json.Marshal(ro)
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

// PopulationMetrics represents aggregated population-level analytics.
// This is what KB-11 is designed to answer - population questions, not individual.
type PopulationMetrics struct {
	TotalPatients       int                    `json:"total_patients"`
	RiskDistribution    map[RiskTier]int       `json:"risk_distribution"`
	CareGapDistribution map[string]int         `json:"care_gap_distribution"`
	AverageRiskScore    float64                `json:"average_risk_score"`
	HighRiskPercentage  float64                `json:"high_risk_percentage"`
	RisingRiskCount     int                    `json:"rising_risk_count"`
	ByPractice          map[string]int         `json:"by_practice,omitempty"`
	ByPCP               map[string]int         `json:"by_pcp,omitempty"`
	CalculatedAt        time.Time              `json:"calculated_at"`
}

// CohortMembership represents patient membership in a cohort.
type CohortMembership struct {
	CohortID   uuid.UUID `json:"cohort_id"`
	PatientID  uuid.UUID `json:"patient_id"`
	JoinedAt   time.Time `json:"joined_at"`
	RemovedAt  *time.Time `json:"removed_at,omitempty"`
	IsActive   bool       `json:"is_active"`
}

// NewPatientProjection creates a new patient projection from FHIR data.
func NewPatientProjection(fhirID string, source SyncSource) *PatientProjection {
	now := time.Now()
	return &PatientProjection{
		ID:              uuid.New(),
		FHIRID:          fhirID,
		CurrentRiskTier: RiskTierUnscored,
		CareGapCount:    0,
		LastSyncedAt:    now,
		SyncSource:      source,
		SyncVersion:     1,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

// NewRiskAssessment creates a new risk assessment with determinism tracking.
func NewRiskAssessment(patientFHIRID, modelName, modelVersion string, input *RiskInput, output *RiskOutput) *RiskAssessment {
	now := time.Now()
	return &RiskAssessment{
		ID:                  uuid.New(),
		PatientFHIRID:       patientFHIRID,
		ModelName:           modelName,
		ModelVersion:        modelVersion,
		Score:               output.Score,
		RiskTier:            output.RiskTier,
		ContributingFactors: output.ContributingFactors,
		InputHash:           input.Hash(),
		CalculationHash:     output.Hash(),
		CalculatedAt:        now,
	}
}

// ToHistory converts a risk assessment to a history record.
func (ra *RiskAssessment) ToHistory() *RiskAssessmentHistory {
	return &RiskAssessmentHistory{
		ID:                  uuid.New(),
		AssessmentID:        ra.ID,
		PatientFHIRID:       ra.PatientFHIRID,
		ModelName:           ra.ModelName,
		ModelVersion:        ra.ModelVersion,
		Score:               ra.Score,
		RiskTier:            ra.RiskTier,
		ContributingFactors: ra.ContributingFactors,
		InputHash:           ra.InputHash,
		CalculationHash:     ra.CalculationHash,
		GovernanceEventID:   ra.GovernanceEventID,
		CalculatedAt:        ra.CalculatedAt,
		ArchivedAt:          time.Now(),
	}
}

// Age calculates the patient's age based on date of birth.
func (pp *PatientProjection) Age() *int {
	if pp.DateOfBirth == nil {
		return nil
	}
	now := time.Now()
	age := now.Year() - pp.DateOfBirth.Year()
	if now.YearDay() < pp.DateOfBirth.YearDay() {
		age--
	}
	return &age
}

// FullName returns the patient's full name.
func (pp *PatientProjection) FullName() string {
	first := ""
	last := ""
	if pp.FirstName != nil {
		first = *pp.FirstName
	}
	if pp.LastName != nil {
		last = *pp.LastName
	}
	if first == "" && last == "" {
		return "Unknown"
	}
	if first == "" {
		return last
	}
	if last == "" {
		return first
	}
	return first + " " + last
}

// IsHighRisk returns true if the patient is in a high-risk tier.
func (pp *PatientProjection) IsHighRisk() bool {
	return pp.CurrentRiskTier == RiskTierHigh ||
		pp.CurrentRiskTier == RiskTierVeryHigh ||
		pp.CurrentRiskTier == RiskTierRising
}

// NeedsSyncUpdate returns true if the projection is stale and needs refresh.
func (pp *PatientProjection) NeedsSyncUpdate(maxAge time.Duration) bool {
	return time.Since(pp.LastSyncedAt) > maxAge
}
