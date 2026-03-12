// Package kdigo provides KDIGO (Kidney Disease: Improving Global Outcomes) guideline
// ingestion for organ impairment dosing rules.
//
// OrganImpairmentRule is the canonical computable schema for organ impairment
// dosing adjustments. It encodes threshold logic (e.g., "eGFR < 30 → CONTRAINDICATED")
// rather than descriptive prose.
//
// V2 Architecture:
//   - KDIGO is the SOLE source for organ impairment rules
//   - Extraction via MCP-RAG atomiser (offline, LLM-based)
//   - All rules require pharmacist review (PENDING_REVIEW)
//   - CPIC removed from OI path (CPIC is pharmacogenomics, not organ function)
package kdigo

import "time"

// =============================================================================
// CANONICAL ORGAN IMPAIRMENT RULE SCHEMA
// =============================================================================

// OrganImpairmentRule is the canonical computable schema for renal/hepatic
// dosing rules. KDIGO is the sole source (CPIC removed — it's pharmacogenomics).
// All organ impairment facts MUST use this schema as fact Content.
type OrganImpairmentRule struct {
	DrugRxCUI        string  `json:"drugRxCUI"`
	DrugName         string  `json:"drugName"`
	OrganSystem      string  `json:"organSystem"`       // RENAL | HEPATIC
	ImpairmentMetric string  `json:"impairmentMetric"`  // eGFR | CrCl | ChildPugh
	ThresholdOp      string  `json:"thresholdOp"`       // < | <= | >=
	ThresholdValue   float64 `json:"thresholdValue"`
	ThresholdUnit    string  `json:"thresholdUnit"`     // mL/min/1.73m2 | mL/min | score
	ActionType       string  `json:"actionType"`        // DOSE_REDUCE | AVOID | CONTRAINDICATED | MONITOR | USE_WITH_CAUTION
	ActionDetail     string  `json:"actionDetail"`      // "Reduce dose by 50%"
	AppliesTo        string  `json:"appliesTo"`         // ADULT | PEDIATRIC | ALL
	EvidenceSource   string  `json:"evidenceSource"`    // KDIGO (sole source for OI)
	EvidenceLevel    string  `json:"evidenceLevel"`     // A | B | 1A | 1B | 2C | High | Medium
	GuidelineRef     string  `json:"guidelineRef"`      // "KDIGO 2024 Diabetes in CKD §4.2"
	Confidence       float64 `json:"confidence"`

	// V2 Enhancements: Version pinning, scope, conflict detection
	GuidelineVersion string  `json:"guidelineVersion,omitempty"` // "KDIGO 2024" — version pinning for audit
	RuleScope        string  `json:"ruleScope,omitempty"`        // INITIATION_ONLY | MAINTENANCE | BOTH
	Conflict         bool    `json:"conflict,omitempty"`         // True if sources disagree at same threshold
	ConfidenceBand   float64 `json:"confidenceBand,omitempty"`   // 0.75 (heatmap+prose), 0.65 (prose), 0.55 (ambiguous)

	// PDF extraction provenance
	SourcePage    int    `json:"sourcePage,omitempty"`    // Page number in source PDF
	SourceSnippet string `json:"sourceSnippet,omitempty"` // Raw text snippet for reviewer audit

	// V3 Gap Fixes: Citation linkage for Evidence Registry
	GuidelineDOI     string `json:"guidelineDoi,omitempty"`     // DOI of the guideline (e.g., "10.1016/j.kint.2022.06.008")
	RecommendationID string `json:"recommendationId,omitempty"` // Formal recommendation ID (e.g., "Recommendation 1.3.1")
}

// =============================================================================
// LOCAL TYPE ALIASES (to avoid import cycles)
// =============================================================================
// These mirror types from datasources package for interface implementation.
// Same pattern as cpic, lactmed, drugbank, crediblemeds, etc.

// FactType represents the category of clinical fact.
type FactType string

const (
	FactTypeOrganImpairment FactType = "ORGAN_IMPAIRMENT"
)

// AuthorityFact represents a clinical fact from an authority source.
type AuthorityFact struct {
	ID               string      `json:"id"`
	AuthoritySource  string      `json:"authority_source"`
	FactType         FactType    `json:"fact_type"`
	RxCUI            string      `json:"rxcui,omitempty"`
	DrugName         string      `json:"drug_name"`
	Content          interface{} `json:"content"`
	RiskLevel        string      `json:"risk_level,omitempty"`
	ActionRequired   string      `json:"action_required,omitempty"`
	Recommendations  []string    `json:"recommendations,omitempty"`
	EvidenceLevel    string      `json:"evidence_level,omitempty"`
	ExtractionMethod string      `json:"extraction_method"`
	Confidence       float64     `json:"confidence"`
	FetchedAt        time.Time   `json:"fetched_at"`
}
