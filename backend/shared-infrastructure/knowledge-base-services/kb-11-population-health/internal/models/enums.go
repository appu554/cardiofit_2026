// Package models provides domain models for KB-11 Population Health Engine.
package models

import "strings"

// RiskTier represents the risk stratification level for a patient.
// Per North Star: KB-11 calculates these tiers, KB-19 uses them for decisions.
type RiskTier string

const (
	RiskTierUnscored RiskTier = "UNSCORED"
	RiskTierLow      RiskTier = "LOW"
	RiskTierModerate RiskTier = "MODERATE"
	RiskTierHigh     RiskTier = "HIGH"
	RiskTierVeryHigh RiskTier = "VERY_HIGH"
	RiskTierRising   RiskTier = "RISING"
)

// IsValid returns true if the risk tier is a valid value.
func (r RiskTier) IsValid() bool {
	switch r {
	case RiskTierUnscored, RiskTierLow, RiskTierModerate, RiskTierHigh, RiskTierVeryHigh, RiskTierRising:
		return true
	}
	return false
}

// Priority returns the priority level (higher = more urgent).
func (r RiskTier) Priority() int {
	switch r {
	case RiskTierVeryHigh:
		return 5
	case RiskTierHigh:
		return 4
	case RiskTierRising:
		return 3
	case RiskTierModerate:
		return 2
	case RiskTierLow:
		return 1
	default:
		return 0
	}
}

// CohortType represents the type of cohort.
type CohortType string

const (
	CohortTypeStatic   CohortType = "STATIC"   // Fixed membership, manually maintained
	CohortTypeDynamic  CohortType = "DYNAMIC"  // Rule-based, automatically refreshed
	CohortTypeSnapshot CohortType = "SNAPSHOT" // Point-in-time capture
)

// IsValid returns true if the cohort type is valid.
func (c CohortType) IsValid() bool {
	switch c {
	case CohortTypeStatic, CohortTypeDynamic, CohortTypeSnapshot:
		return true
	}
	return false
}

// SyncSource represents the source of patient data.
type SyncSource string

const (
	SyncSourceFHIR SyncSource = "FHIR" // Google Healthcare API / FHIR Store
	SyncSourceKB17 SyncSource = "KB17" // KB-17 Population Registry
	SyncSourceKB13 SyncSource = "KB13" // KB-13 Care Gaps (for aggregation)
)

// IsValid returns true if the sync source is valid.
func (s SyncSource) IsValid() bool {
	switch s {
	case SyncSourceFHIR, SyncSourceKB17, SyncSourceKB13:
		return true
	}
	return false
}

// SyncStatus represents the status of a synchronization operation.
type SyncStatus string

const (
	SyncStatusPending    SyncStatus = "PENDING"
	SyncStatusInProgress SyncStatus = "IN_PROGRESS"
	SyncStatusSuccess    SyncStatus = "SUCCESS"
	SyncStatusFailed     SyncStatus = "FAILED"
)

// IsValid returns true if the sync status is valid.
func (s SyncStatus) IsValid() bool {
	switch s {
	case SyncStatusPending, SyncStatusInProgress, SyncStatusSuccess, SyncStatusFailed:
		return true
	}
	return false
}

// RiskModelType represents the type of risk model.
type RiskModelType string

const (
	RiskModelHospitalization     RiskModelType = "HOSPITALIZATION"
	RiskModelReadmission         RiskModelType = "READMISSION"
	RiskModelEDUtilization       RiskModelType = "ED_UTILIZATION"
	RiskModelDiabetesProgression RiskModelType = "DIABETES_PROGRESSION"
	RiskModelCHFExacerbation     RiskModelType = "CHF_EXACERBATION"
	RiskModelFrailty             RiskModelType = "FRAILTY"
)

// IsValid returns true if the risk model type is valid.
func (r RiskModelType) IsValid() bool {
	switch r {
	case RiskModelHospitalization, RiskModelReadmission, RiskModelEDUtilization,
		RiskModelDiabetesProgression, RiskModelCHFExacerbation, RiskModelFrailty:
		return true
	}
	return false
}

// Gender represents patient gender.
type Gender string

const (
	GenderMale    Gender = "male"
	GenderFemale  Gender = "female"
	GenderOther   Gender = "other"
	GenderUnknown Gender = "unknown"
)

// IsValid returns true if the gender is valid.
func (g Gender) IsValid() bool {
	lower := strings.ToLower(string(g))
	switch Gender(lower) {
	case GenderMale, GenderFemale, GenderOther, GenderUnknown:
		return true
	}
	return false
}

// InterventionType represents the type of intervention recommended.
type InterventionType string

const (
	InterventionStandardPreventive   InterventionType = "standard_preventive"
	InterventionEnhancedMonitoring   InterventionType = "enhanced_monitoring"
	InterventionCareManagement       InterventionType = "care_management"
	InterventionIntensiveCoordination InterventionType = "intensive_coordination"
)

// IsValid returns true if the intervention type is valid.
func (i InterventionType) IsValid() bool {
	switch i {
	case InterventionStandardPreventive, InterventionEnhancedMonitoring,
		InterventionCareManagement, InterventionIntensiveCoordination:
		return true
	}
	return false
}

// CriteriaOperator represents operators for cohort criteria evaluation.
type CriteriaOperator string

const (
	OpEquals     CriteriaOperator = "eq"
	OpNotEquals  CriteriaOperator = "neq"
	OpGreaterThan CriteriaOperator = "gt"
	OpGreaterEq  CriteriaOperator = "gte"
	OpLessThan   CriteriaOperator = "lt"
	OpLessEq     CriteriaOperator = "lte"
	OpIn         CriteriaOperator = "in"
	OpNotIn      CriteriaOperator = "not_in"
	OpMatches    CriteriaOperator = "matches"
	OpContains   CriteriaOperator = "contains"
)

// IsValid returns true if the operator is valid.
func (o CriteriaOperator) IsValid() bool {
	switch o {
	case OpEquals, OpNotEquals, OpGreaterThan, OpGreaterEq, OpLessThan, OpLessEq,
		OpIn, OpNotIn, OpMatches, OpContains:
		return true
	}
	return false
}
