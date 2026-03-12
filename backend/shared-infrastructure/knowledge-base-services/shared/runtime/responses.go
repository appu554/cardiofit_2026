// Package runtime provides KB-19 runtime response types and contracts.
// These models define the API response structures for clinical safety checks.
//
// DESIGN PRINCIPLE: "Absence ≠ Safety"
// A "no interaction found" response MUST distinguish between:
// - Checked all sources, no interaction found (HIGH confidence SAFE)
// - Only checked some sources (MEDIUM confidence SAFE)
// - Drug not recognized in database (LOW confidence UNKNOWN)
//
// This file implements the Coverage Metadata refinement from Phase 1 review.
package runtime

import "time"

// =============================================================================
// DDI CHECK RESULT WITH TIERED CACHING
// =============================================================================
// Supports the Tiered Caching Strategy for DDI scale:
// - Tier 0 (Memory/Hot): ONC High-Priority (~1,200 pairs), <1ms latency
// - Tier 1 (DB/Warm): OHDSI Athena (~200K pairs), 5-20ms latency

// CacheTier indicates which cache layer served the result
type CacheTier int

const (
	// CacheTierHot is Tier 0 - in-memory cache for high-priority interactions (ONC)
	CacheTierHot CacheTier = 0

	// CacheTierWarm is Tier 1 - database cache for broader coverage (OHDSI)
	CacheTierWarm CacheTier = 1

	// CacheTierCold is Tier 2 - source lookup required (real-time API or file)
	CacheTierCold CacheTier = 2
)

// DDICheckResult is the response for a drug-drug interaction check
// This struct captures both the clinical result and operational metadata
type DDICheckResult struct {
	// ─────────────────────────────────────────────────────────────────────────
	// CLINICAL RESULT
	// ─────────────────────────────────────────────────────────────────────────

	// HasInteraction indicates if an interaction was found
	HasInteraction bool `json:"hasInteraction"`

	// Severity is the interaction severity (if found): CRITICAL, MAJOR, MODERATE, MINOR
	Severity string `json:"severity,omitempty"`

	// ClinicalEffect describes the clinical outcome (if interaction found)
	ClinicalEffect string `json:"clinicalEffect,omitempty"`

	// Management is the recommended management strategy (if interaction found)
	Management string `json:"management,omitempty"`

	// ─────────────────────────────────────────────────────────────────────────
	// SOURCE INFORMATION
	// ─────────────────────────────────────────────────────────────────────────

	// Source indicates which data source provided the result: "ONC", "OHDSI", "MEDRT"
	Source string `json:"source,omitempty"`

	// SourceAuthority is the authority ranking of the source (lower = more authoritative)
	SourceAuthority int `json:"sourceAuthority,omitempty"`

	// ─────────────────────────────────────────────────────────────────────────
	// CACHE & PERFORMANCE
	// ─────────────────────────────────────────────────────────────────────────

	// CacheTier indicates which cache layer served this result (0=hot, 1=warm, 2=cold)
	CacheTier CacheTier `json:"cacheTier"`

	// LatencyMs is the lookup time in milliseconds
	LatencyMs int64 `json:"latencyMs"`

	// CacheHit indicates if this was served from cache vs computed
	CacheHit bool `json:"cacheHit"`

	// ─────────────────────────────────────────────────────────────────────────
	// DIRECTIONALITY (Perpetrator vs Victim)
	// ─────────────────────────────────────────────────────────────────────────

	// IsBidirectional indicates if both drugs contribute equally
	IsBidirectional bool `json:"isBidirectional"`

	// AffectedDrugRxCUI is the drug requiring adjustment (if unidirectional)
	AffectedDrugRxCUI string `json:"affectedDrugRxcui,omitempty"`

	// Mechanism provides structured mechanism: "CYP3A4_INHIBITION", "QT_ADDITIVE", etc.
	Mechanism string `json:"mechanism,omitempty"`
}

// =============================================================================
// SAFETY CHECK RESPONSE WITH COVERAGE METADATA
// =============================================================================
// Critical refinement: Coverage metadata explains WHAT was checked and
// what was NOT checked, allowing clinicians to interpret "no finding" correctly.

// SafetyCheckStatus indicates the overall safety assessment
type SafetyCheckStatus string

const (
	// StatusSafe indicates no safety concerns were found
	StatusSafe SafetyCheckStatus = "SAFE"

	// StatusAlert indicates a safety concern was detected
	StatusAlert SafetyCheckStatus = "ALERT"

	// StatusUnknown indicates insufficient data to determine safety
	StatusUnknown SafetyCheckStatus = "UNKNOWN"

	// StatusError indicates a system error during the check
	StatusError SafetyCheckStatus = "ERROR"
)

// SafetyConfidence indicates confidence in the safety assessment
type SafetyConfidence string

const (
	// ConfidenceHigh means all relevant sources were checked successfully
	ConfidenceHigh SafetyConfidence = "HIGH"

	// ConfidenceMedium means some sources were checked but not all
	ConfidenceMedium SafetyConfidence = "MEDIUM"

	// ConfidenceLow means limited sources were available or drug not recognized
	ConfidenceLow SafetyConfidence = "LOW"
)

// CoverageSource represents a data source that was or was not checked
type CoverageSource string

const (
	// CoverageONCChecked - ONC High-Priority DDI database was checked
	CoverageONCChecked CoverageSource = "ONC_CHECKED"

	// CoverageOHDSIChecked - OHDSI Athena vocabulary was checked
	CoverageOHDSIChecked CoverageSource = "OHDSI_CHECKED"

	// CoverageMEDRTChecked - MED-RT signal database was checked
	CoverageMEDRTChecked CoverageSource = "MEDRT_CHECKED"

	// CoverageDrugBankChecked - DrugBank database was checked
	CoverageDrugBankChecked CoverageSource = "DRUGBANK_CHECKED"

	// CoverageFDAChecked - FDA SPL labels were checked
	CoverageFDAChecked CoverageSource = "FDA_SPL_CHECKED"

	// CoverageFormularyChecked - Formulary coverage was checked
	CoverageFormularyChecked CoverageSource = "FORMULARY_CHECKED"

	// CoverageLabRangesChecked - Lab reference ranges were checked
	CoverageLabRangesChecked CoverageSource = "LAB_RANGES_CHECKED"
)

// NotCoveredReason explains why a source was not checked
type NotCoveredReason string

const (
	// NotCoveredTimeout - source timed out during lookup
	NotCoveredTimeout NotCoveredReason = "TIMEOUT"

	// NotCoveredUnavailable - source was unavailable/down
	NotCoveredUnavailable NotCoveredReason = "UNAVAILABLE"

	// NotCoveredNotConfigured - source not configured for this environment
	NotCoveredNotConfigured NotCoveredReason = "NOT_CONFIGURED"

	// NotCoveredDrugNotMapped - drug not mapped in this source
	NotCoveredDrugNotMapped NotCoveredReason = "DRUG_NOT_MAPPED"

	// NotCoveredRateLimited - source rate limited
	NotCoveredRateLimited NotCoveredReason = "RATE_LIMITED"
)

// NotCoveredSource describes a source that was NOT checked and why
type NotCoveredSource struct {
	// Source identifies which source was not checked
	Source string `json:"source"`

	// Reason explains why it was not checked
	Reason NotCoveredReason `json:"reason"`

	// Details provides additional context
	Details string `json:"details,omitempty"`
}

// SafetyCheckResponse is the comprehensive response for a safety check
// This is the KB-19 runtime response contract with coverage metadata
type SafetyCheckResponse struct {
	// ─────────────────────────────────────────────────────────────────────────
	// PRIMARY RESULT
	// ─────────────────────────────────────────────────────────────────────────

	// Status is the overall safety assessment: SAFE, ALERT, UNKNOWN, ERROR
	Status SafetyCheckStatus `json:"status"`

	// Confidence indicates how much to trust this assessment: HIGH, MEDIUM, LOW
	Confidence SafetyConfidence `json:"confidence"`

	// ─────────────────────────────────────────────────────────────────────────
	// COVERAGE METADATA (Critical: "Absence ≠ Safety")
	// ─────────────────────────────────────────────────────────────────────────

	// Coverage lists all sources that WERE checked successfully
	Coverage []CoverageSource `json:"coverage"`

	// NotCovered lists sources that were NOT checked and why
	// This is CRITICAL for clinical interpretation of "no finding"
	NotCovered []NotCoveredSource `json:"notCovered,omitempty"`

	// ─────────────────────────────────────────────────────────────────────────
	// DRUG RECOGNITION
	// ─────────────────────────────────────────────────────────────────────────

	// DrugRecognized indicates if the drug(s) were found in the master database
	// If false, "SAFE" status cannot be trusted - it means "no data exists"
	DrugRecognized bool `json:"drugRecognized"`

	// UnrecognizedDrugs lists RxCUIs that were not found in drug master
	UnrecognizedDrugs []string `json:"unrecognizedDrugs,omitempty"`

	// ─────────────────────────────────────────────────────────────────────────
	// ALERTS (if Status == ALERT)
	// ─────────────────────────────────────────────────────────────────────────

	// Alerts contains the specific safety concerns found
	Alerts []SafetyAlert `json:"alerts,omitempty"`

	// ─────────────────────────────────────────────────────────────────────────
	// METADATA
	// ─────────────────────────────────────────────────────────────────────────

	// CheckedAt is when the safety check was performed
	CheckedAt time.Time `json:"checkedAt"`

	// TotalLatencyMs is the total time for all source lookups
	TotalLatencyMs int64 `json:"totalLatencyMs"`

	// RequestID is a unique identifier for this check (for audit/debugging)
	RequestID string `json:"requestId,omitempty"`
}

// SafetyAlert represents a specific safety concern
type SafetyAlert struct {
	// AlertType categorizes the alert: DDI, CONTRAINDICATION, LAB_CRITICAL, etc.
	AlertType string `json:"alertType"`

	// Severity is the clinical severity: CRITICAL, MAJOR, MODERATE, MINOR
	Severity string `json:"severity"`

	// Title is a short description of the alert
	Title string `json:"title"`

	// Description provides clinical details
	Description string `json:"description"`

	// Recommendation is the clinical action to take
	Recommendation string `json:"recommendation"`

	// Source indicates where this alert came from
	Source string `json:"source"`

	// SourceAuthority is the authority ranking of the source
	SourceAuthority int `json:"sourceAuthority"`

	// EvidenceLevel is the quality of evidence: HIGH, MODERATE, LOW, THEORETICAL
	EvidenceLevel string `json:"evidenceLevel"`

	// AffectedDrug is the RxCUI of the drug requiring action (if applicable)
	AffectedDrug string `json:"affectedDrug,omitempty"`
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// NewSafeResponse creates a SAFE response with full coverage
func NewSafeResponse(coverage []CoverageSource) *SafetyCheckResponse {
	return &SafetyCheckResponse{
		Status:         StatusSafe,
		Confidence:     ConfidenceHigh,
		Coverage:       coverage,
		DrugRecognized: true,
		CheckedAt:      time.Now(),
	}
}

// NewAlertResponse creates an ALERT response with safety concerns
func NewAlertResponse(alerts []SafetyAlert, coverage []CoverageSource) *SafetyCheckResponse {
	return &SafetyCheckResponse{
		Status:         StatusAlert,
		Confidence:     ConfidenceHigh,
		Coverage:       coverage,
		DrugRecognized: true,
		Alerts:         alerts,
		CheckedAt:      time.Now(),
	}
}

// NewUnknownResponse creates an UNKNOWN response for unrecognized drugs
func NewUnknownResponse(unrecognizedDrugs []string) *SafetyCheckResponse {
	return &SafetyCheckResponse{
		Status:            StatusUnknown,
		Confidence:        ConfidenceLow,
		Coverage:          []CoverageSource{},
		DrugRecognized:    false,
		UnrecognizedDrugs: unrecognizedDrugs,
		CheckedAt:         time.Now(),
	}
}

// NewPartialCoverageResponse creates a SAFE response with partial coverage
func NewPartialCoverageResponse(coverage []CoverageSource, notCovered []NotCoveredSource) *SafetyCheckResponse {
	return &SafetyCheckResponse{
		Status:         StatusSafe,
		Confidence:     ConfidenceMedium, // Lower confidence due to partial coverage
		Coverage:       coverage,
		NotCovered:     notCovered,
		DrugRecognized: true,
		CheckedAt:      time.Now(),
	}
}

// DetermineConfidence calculates confidence based on coverage completeness
func DetermineConfidence(coverage []CoverageSource, notCovered []NotCoveredSource, drugRecognized bool) SafetyConfidence {
	if !drugRecognized {
		return ConfidenceLow
	}

	// Count critical sources checked
	criticalSources := map[CoverageSource]bool{
		CoverageONCChecked:   false,
		CoverageOHDSIChecked: false,
	}

	for _, src := range coverage {
		if _, ok := criticalSources[src]; ok {
			criticalSources[src] = true
		}
	}

	// If all critical sources checked, HIGH confidence
	allCritical := true
	for _, checked := range criticalSources {
		if !checked {
			allCritical = false
			break
		}
	}

	if allCritical && len(notCovered) == 0 {
		return ConfidenceHigh
	}

	if len(coverage) > 0 {
		return ConfidenceMedium
	}

	return ConfidenceLow
}
