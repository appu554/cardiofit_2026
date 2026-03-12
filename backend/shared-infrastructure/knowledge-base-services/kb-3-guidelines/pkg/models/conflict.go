package models

import "time"

// ConflictType categorizes the type of conflict between guidelines
type ConflictType string

const (
	ConflictTargetDifference     ConflictType = "target_difference"
	ConflictEvidenceDisagreement ConflictType = "evidence_disagreement"
	ConflictTreatmentPreference  ConflictType = "treatment_preference"
	ConflictDirectContradiction  ConflictType = "direct_contradiction"
)

// ConflictStatus represents the status of a conflict
type ConflictStatus string

const (
	ConflictDetected ConflictStatus = "detected"
	ConflictReviewed ConflictStatus = "reviewed"
	ConflictResolved ConflictStatus = "resolved"
	ConflictIgnored  ConflictStatus = "ignored"
)

// Conflict represents a detected conflict between guidelines
type Conflict struct {
	ConflictID      string                 `json:"conflict_id"`
	Guideline1ID    string                 `json:"guideline1_id"`
	Guideline2ID    string                 `json:"guideline2_id"`
	Recommendation1 map[string]interface{} `json:"recommendation1"`
	Recommendation2 map[string]interface{} `json:"recommendation2"`
	Type            ConflictType           `json:"type"`
	Severity        Severity               `json:"severity"`
	Domain          string                 `json:"domain"`
	Status          ConflictStatus         `json:"status"`
	Description     string                 `json:"description,omitempty"`
	DetectedAt      time.Time              `json:"detected_at"`
	ResolvedAt      *time.Time             `json:"resolved_at,omitempty"`
}

// Resolution represents a conflict resolution decision
type Resolution struct {
	Applicable           bool   `json:"applicable"`
	WinningGuideline     string `json:"winning_guideline,omitempty"`
	Action               any    `json:"action,omitempty"`
	Rationale            string `json:"rationale,omitempty"`
	SafetyOverride       bool   `json:"safety_override,omitempty"`
	OverrideID           string `json:"override_id,omitempty"`
	RequiresManualReview bool   `json:"requires_manual_review,omitempty"`
	RuleUsed             string `json:"rule_used,omitempty"`
	ConflictID           string `json:"conflict_id,omitempty"`
}

// ResolutionRule represents a tier in the 5-tier resolution system
type ResolutionRule string

const (
	RuleSafetyOverride    ResolutionRule = "safety_override"
	RuleRegionalPreference ResolutionRule = "regional_preference"
	RuleEvidenceStrength  ResolutionRule = "evidence_strength"
	RulePublicationRecency ResolutionRule = "publication_recency"
	RuleConservativeDefault ResolutionRule = "conservative_default"
)

// PatientContext provides patient-specific context for conflict resolution
type PatientContext struct {
	PatientID        string                 `json:"patient_id"`
	Age              int                    `json:"age"`
	Sex              string                 `json:"sex"`
	Region           string                 `json:"region,omitempty"`
	PregnancyStatus  string                 `json:"pregnancy_status,omitempty"`
	Labs             map[string]float64     `json:"labs"`
	ActiveConditions []string               `json:"active_conditions"`
	Medications      []string               `json:"medications"`
	Allergies        []string               `json:"allergies"`
	Comorbidities    []string               `json:"comorbidities"`
	RiskFactors      map[string]interface{} `json:"risk_factors"`
	InsuranceCoverage string               `json:"insurance_coverage,omitempty"`
	CareSetting      string                 `json:"care_setting,omitempty"`
}

// ConflictStatistics provides analytics on conflicts
type ConflictStatistics struct {
	TotalConflicts      int                    `json:"total_conflicts"`
	ByType              map[ConflictType]int   `json:"by_type"`
	BySeverity          map[Severity]int       `json:"by_severity"`
	ByDomain            map[string]int         `json:"by_domain"`
	ResolutionRate      float64                `json:"resolution_rate"`
	AverageResolutionTime time.Duration        `json:"average_resolution_time"`
	Period              string                 `json:"period"`
}

// ConflictDetectionRequest for API
type ConflictDetectionRequest struct {
	GuidelineIDs []string        `json:"guideline_ids"`
	Context      *PatientContext `json:"context,omitempty"`
}

// ConflictResolutionRequest for API
type ConflictResolutionRequest struct {
	ConflictID string          `json:"conflict_id"`
	Conflict   *Conflict       `json:"conflict,omitempty"`
	Context    *PatientContext `json:"context,omitempty"`
	Manual     bool            `json:"manual,omitempty"`
	Rationale  string          `json:"rationale,omitempty"`
}

// ResolveConflictRequest is an alias for ConflictResolutionRequest
type ResolveConflictRequest = ConflictResolutionRequest
