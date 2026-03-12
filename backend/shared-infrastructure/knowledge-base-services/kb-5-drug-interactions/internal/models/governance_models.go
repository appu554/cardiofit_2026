package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ============================================================================
// GOVERNANCE ACTION TYPES
// Maps clinical severity to governance workflow actions
// ============================================================================

// GovernanceAction represents the workflow action level for clinical alerts
type GovernanceAction string

const (
	// GovernanceIgnore - No action required, informational only
	GovernanceIgnore GovernanceAction = "ignore"

	// GovernanceNotify - Send notification to relevant parties (non-blocking)
	GovernanceNotify GovernanceAction = "notify"

	// GovernanceWarnAcknowledge - Warning that requires clinician acknowledgment
	GovernanceWarnAcknowledge GovernanceAction = "warn_acknowledge"

	// GovernanceHardBlock - Blocked, cannot override at clinical level
	GovernanceHardBlock GovernanceAction = "hard_block"

	// GovernanceHardBlockOverride - Blocked, but governance/P&T can override
	GovernanceHardBlockOverride GovernanceAction = "hard_block_governance_override"

	// GovernanceMandatoryEscalation - Requires escalation to senior clinician/pharmacist
	GovernanceMandatoryEscalation GovernanceAction = "mandatory_escalation"
)

// GovernanceActionPriority returns the enforcement priority (higher = stricter)
func (ga GovernanceAction) Priority() int {
	priorities := map[GovernanceAction]int{
		GovernanceIgnore:              0,
		GovernanceNotify:              1,
		GovernanceWarnAcknowledge:     2,
		GovernanceMandatoryEscalation: 3,
		GovernanceHardBlockOverride:   4,
		GovernanceHardBlock:           5,
	}
	if p, ok := priorities[ga]; ok {
		return p
	}
	return 0
}

// IsBlocking returns true if the action prevents order completion
func (ga GovernanceAction) IsBlocking() bool {
	return ga == GovernanceHardBlock || ga == GovernanceHardBlockOverride
}

// RequiresAcknowledgment returns true if the action needs clinician response
func (ga GovernanceAction) RequiresAcknowledgment() bool {
	return ga == GovernanceWarnAcknowledge ||
		ga == GovernanceMandatoryEscalation ||
		ga == GovernanceHardBlockOverride
}

// AllowsClinicalOverride returns true if clinical staff can override
func (ga GovernanceAction) AllowsClinicalOverride() bool {
	return ga != GovernanceHardBlock
}

// ============================================================================
// SEVERITY → GOVERNANCE MAPPING POLICY
// Configurable mapping layer for institutional policy enforcement
// ============================================================================

// SeverityGovernanceMapping defines the mapping from clinical severity to governance action
type SeverityGovernanceMapping struct {
	ID                 uuid.UUID         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PolicyName         string            `gorm:"size:100;not null;uniqueIndex" json:"policy_name"`
	InstitutionID      string            `gorm:"size:100;index" json:"institution_id,omitempty"`

	// Severity → Governance mappings
	ContraindicatedAction GovernanceAction `gorm:"not null;default:'hard_block'" json:"contraindicated_action"`
	MajorAction           GovernanceAction `gorm:"not null;default:'warn_acknowledge'" json:"major_action"`
	ModerateAction        GovernanceAction `gorm:"not null;default:'notify'" json:"moderate_action"`
	MinorAction           GovernanceAction `gorm:"not null;default:'ignore'" json:"minor_action"`
	UnknownAction         GovernanceAction `gorm:"not null;default:'notify'" json:"unknown_action"`

	// Context modifiers
	PediatricEscalation   bool `gorm:"default:true" json:"pediatric_escalation"`
	GeriatricEscalation   bool `gorm:"default:true" json:"geriatric_escalation"`
	RenalImpairmentUpgrade bool `gorm:"default:true" json:"renal_impairment_upgrade"`
	HepaticImpairmentUpgrade bool `gorm:"default:true" json:"hepatic_impairment_upgrade"`

	// Policy metadata
	EffectiveDate time.Time  `gorm:"not null;default:now()" json:"effective_date"`
	ExpiryDate    *time.Time `json:"expiry_date,omitempty"`
	Active        bool       `gorm:"default:true;index" json:"active"`
	ApprovedBy    string     `gorm:"size:100" json:"approved_by,omitempty"`
	ApprovedAt    *time.Time `json:"approved_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (SeverityGovernanceMapping) TableName() string {
	return "ddi_severity_governance_mappings"
}

// MapSeverityToAction translates clinical severity to governance action
func (sgm *SeverityGovernanceMapping) MapSeverityToAction(severity DDISeverity) GovernanceAction {
	switch severity {
	case SeverityContraindicated:
		return sgm.ContraindicatedAction
	case SeverityMajor:
		return sgm.MajorAction
	case SeverityModerate:
		return sgm.ModerateAction
	case SeverityMinor:
		return sgm.MinorAction
	default:
		return sgm.UnknownAction
	}
}

// ============================================================================
// ATTRIBUTION + EVIDENCE LABELS
// Provides provenance, traceability, and medico-legal documentation
// ============================================================================

// ClinicalSource represents the origin of clinical knowledge
type ClinicalSource string

const (
	SourceFDALabel        ClinicalSource = "FDA_LABEL"
	SourceDrugBank        ClinicalSource = "DRUGBANK"
	SourceRxNorm          ClinicalSource = "RXNORM"
	SourceCPIC            ClinicalSource = "CPIC"           // Clinical Pharmacogenetics Implementation Consortium
	SourcePharmGKB        ClinicalSource = "PHARMGKB"
	SourceMicromedex      ClinicalSource = "MICROMEDEX"
	SourceLexicomp        ClinicalSource = "LEXICOMP"
	SourceUpToDate        ClinicalSource = "UPTODATE"
	SourceClinicalTrials  ClinicalSource = "CLINICAL_TRIALS"
	SourcePeerReviewed    ClinicalSource = "PEER_REVIEWED"
	SourceExpertConsensus ClinicalSource = "EXPERT_CONSENSUS"
	SourceInstitutional   ClinicalSource = "INSTITUTIONAL"
	SourceUnknown         ClinicalSource = "UNKNOWN"

	// Additional DDI-specific clinical sources (Three-Layer Authority Model)
	// Layer 1: Regulatory Authorities
	SourceTGA   ClinicalSource = "TGA"   // Therapeutic Goods Administration (Australia)
	SourceCDSCO ClinicalSource = "CDSCO" // Central Drugs Standard Control Organization (India)
	SourceEMA   ClinicalSource = "EMA"   // European Medicines Agency

	// Layer 2: Pharmacology Authorities
	SourceCredibleMeds ClinicalSource = "CREDIBLEMEDS"  // CredibleMeds QT Drug Lists
	SourceUWDDI        ClinicalSource = "UW_DDI"        // University of Washington DDI Database
	SourceFlockhart    ClinicalSource = "FLOCKHART_CYP" // Indiana University Flockhart CYP Table

	// Layer 3: Clinical Practice Authorities
	SourceStockley ClinicalSource = "STOCKLEY" // Stockley's Drug Interactions
	SourceAMH      ClinicalSource = "AMH"      // Australian Medicines Handbook
	SourceBNF      ClinicalSource = "BNF"      // British National Formulary
	SourceACCP     ClinicalSource = "ACCP"     // American College of Clinical Pharmacy
)

// EvidenceStrength represents the quality/reliability of evidence
type EvidenceStrength string

const (
	EvidenceStrengthHigh       EvidenceStrength = "HIGH"        // RCTs, meta-analyses
	EvidenceStrengthModerate   EvidenceStrength = "MODERATE"    // Observational studies
	EvidenceStrengthLow        EvidenceStrength = "LOW"         // Case reports, mechanistic
	EvidenceStrengthVeryLow    EvidenceStrength = "VERY_LOW"    // Expert opinion, theoretical
	EvidenceStrengthUngraded   EvidenceStrength = "UNGRADED"    // Not formally assessed
)

// GovernanceRelevance indicates applicability to governance/compliance
type GovernanceRelevance string

const (
	GovernanceRelevanceRegulatory   GovernanceRelevance = "REGULATORY"    // FDA, TGA, EMA mandated
	GovernanceRelevanceLegal        GovernanceRelevance = "LEGAL"         // Medico-legal significance
	GovernanceRelevanceAccreditation GovernanceRelevance = "ACCREDITATION" // JCI, NABH standards
	GovernanceRelevanceFormulary    GovernanceRelevance = "FORMULARY"     // P&T Committee decisions
	GovernanceRelevanceProtocol     GovernanceRelevance = "PROTOCOL"      // Clinical protocol requirement
	GovernanceRelevanceAdvisory     GovernanceRelevance = "ADVISORY"      // Best practice recommendation
	GovernanceRelevanceInformational GovernanceRelevance = "INFORMATIONAL" // Educational only
)

// ProgramFlag represents special handling requirements for programs
type ProgramFlag string

const (
	ProgramFlagOncology       ProgramFlag = "ONCOLOGY"
	ProgramFlagPediatric      ProgramFlag = "PEDIATRIC"
	ProgramFlagGeriatric      ProgramFlag = "GERIATRIC"
	ProgramFlagPregnancy      ProgramFlag = "PREGNANCY"
	ProgramFlagRenalDosing    ProgramFlag = "RENAL_DOSING"
	ProgramFlagHepaticDosing  ProgramFlag = "HEPATIC_DOSING"
	ProgramFlagPGx            ProgramFlag = "PHARMACOGENOMICS"
	ProgramFlagControlled     ProgramFlag = "CONTROLLED_SUBSTANCE"
	ProgramFlagHighAlert      ProgramFlag = "HIGH_ALERT"
	ProgramFlagLASA           ProgramFlag = "LASA" // Look-Alike Sound-Alike
	ProgramFlagNarrowTI       ProgramFlag = "NARROW_TI" // Narrow Therapeutic Index
	ProgramFlagBlackBox       ProgramFlag = "BLACK_BOX_WARNING"
	ProgramFlagREMS           ProgramFlag = "REMS" // Risk Evaluation and Mitigation Strategy
)

// ============================================================================
// DDI GOVERNANCE - THREE-LAYER AUTHORITY MODEL
// Per-interaction provenance tracking for medico-legal defensibility
// ============================================================================

// DDIGovernance provides per-interaction provenance tracking
// Implements the three-layer authority model for medico-legal defensibility:
//   - Layer 1: REGULATORY AUTHORITY - "What is LEGALLY prohibited?"
//   - Layer 2: PHARMACOLOGY AUTHORITY - "WHY does this interaction happen?"
//   - Layer 3: CLINICAL PRACTICE AUTHORITY - "HOW do clinicians manage this?"
type DDIGovernance struct {
	// Layer 1: REGULATORY AUTHORITY - "What is LEGALLY prohibited?"
	// Sources: FDA DailyMed (Section 7), TGA Product Information, CDSCO, EMA SmPC
	RegulatoryAuthority    ClinicalSource `gorm:"size:50" json:"regulatory_authority"`               // FDA_LABEL, TGA, CDSCO, EMA
	RegulatoryDocument     string         `gorm:"size:500" json:"regulatory_document,omitempty"`     // "Warfarin Sodium Tablets FDA Label Section 7"
	RegulatoryURL          string         `gorm:"type:text" json:"regulatory_url,omitempty"`         // Link to authoritative source
	RegulatoryJurisdiction string         `gorm:"size:20" json:"regulatory_jurisdiction,omitempty"`  // US, AU, IN, EU, UK

	// Layer 2: PHARMACOLOGY AUTHORITY - "WHY does this interaction happen?"
	// Sources: DrugBank, Flockhart CYP Table, UW DDI Database, CredibleMeds, PharmGKB
	PharmacologyAuthority ClinicalSource `gorm:"size:50" json:"pharmacology_authority"`            // DRUGBANK, FLOCKHART_CYP, PHARMGKB
	MechanismEvidence     string         `gorm:"size:500" json:"mechanism_evidence,omitempty"`     // "CYP2C9 inhibition, Ki = 7.1 μM"
	TransporterData       string         `gorm:"size:200" json:"transporter_data,omitempty"`       // "P-gp substrate", "OATP1B1 inhibitor"
	CYPPathway            string         `gorm:"size:100" json:"cyp_pathway,omitempty"`            // "CYP3A4, CYP2C9"
	QTRiskCategory        string         `gorm:"size:50" json:"qt_risk_category,omitempty"`        // CredibleMeds categories: Known, Possible, Conditional

	// Layer 3: CLINICAL PRACTICE AUTHORITY - "HOW do clinicians manage this?"
	// Sources: Lexicomp, Micromedex, AMH, BNF, Stockley, ACCP Guidelines
	ClinicalAuthority ClinicalSource `gorm:"size:50" json:"clinical_authority"`              // LEXICOMP, MICROMEDEX, AMH, BNF
	SeveritySource    string         `gorm:"size:200" json:"severity_source,omitempty"`      // "Lexicomp Drug Interactions 2024"
	ManagementSource  string         `gorm:"size:200" json:"management_source,omitempty"`    // "ACCP Antithrombotic Guidelines 2024"

	// Quality & Review Metadata
	EvidenceGrade    string     `gorm:"size:10" json:"evidence_grade,omitempty"`    // A, B, C, D per ACCP grading
	LastReviewedDate *time.Time `json:"last_reviewed_date,omitempty"`
	NextReviewDue    *time.Time `json:"next_review_due,omitempty"`
	ReviewedBy       string     `gorm:"size:100" json:"reviewed_by,omitempty"`
}

// HasRegulatorySource returns true if regulatory authority is documented
func (g DDIGovernance) HasRegulatorySource() bool {
	return g.RegulatoryAuthority != "" && g.RegulatoryAuthority != SourceUnknown
}

// HasPharmacologySource returns true if pharmacology authority is documented
func (g DDIGovernance) HasPharmacologySource() bool {
	return g.PharmacologyAuthority != "" && g.PharmacologyAuthority != SourceUnknown
}

// HasClinicalSource returns true if clinical practice authority is documented
func (g DDIGovernance) HasClinicalSource() bool {
	return g.ClinicalAuthority != "" && g.ClinicalAuthority != SourceUnknown
}

// IsFullyAttributed returns true if all three layers have documented sources
func (g DDIGovernance) IsFullyAttributed() bool {
	return g.HasRegulatorySource() && g.HasPharmacologySource() && g.HasClinicalSource()
}

// GetProvenanceGaps returns which layers are missing attribution
func (g DDIGovernance) GetProvenanceGaps() []string {
	var gaps []string
	if !g.HasRegulatorySource() {
		gaps = append(gaps, "REGULATORY")
	}
	if !g.HasPharmacologySource() {
		gaps = append(gaps, "PHARMACOLOGY")
	}
	if !g.HasClinicalSource() {
		gaps = append(gaps, "CLINICAL")
	}
	return gaps
}

// AttributionEnvelope provides complete provenance for clinical decisions
type AttributionEnvelope struct {
	// Dataset provenance
	DatasetVersion     string    `json:"dataset_version"`
	DatasetDate        time.Time `json:"dataset_date"`
	HarmonizationID    string    `json:"harmonization_id,omitempty"`

	// Clinical sources
	ClinicalSources    []ClinicalSource `json:"clinical_sources"`
	PrimaryCitation    string           `json:"primary_citation,omitempty"`
	SecondaryCitations []string         `json:"secondary_citations,omitempty"`

	// Evidence grading
	EvidenceStrength   EvidenceStrength `json:"evidence_strength"`
	EvidenceLevel      EvidenceLevel    `json:"evidence_level"` // Reuse existing model
	ConfidenceScore    *decimal.Decimal `json:"confidence_score,omitempty"`

	// Governance metadata
	GovernanceRelevance []GovernanceRelevance `json:"governance_relevance"`
	GovernanceAction    GovernanceAction      `json:"governance_action"`
	PolicyReference     string                `json:"policy_reference,omitempty"`

	// Program applicability
	ProgramFlags       []ProgramFlag `json:"program_flags,omitempty"`

	// Audit trail
	LastReviewedAt     *time.Time `json:"last_reviewed_at,omitempty"`
	LastReviewedBy     string     `json:"last_reviewed_by,omitempty"`
	NextReviewDue      *time.Time `json:"next_review_due,omitempty"`

	// Override tracking
	InstitutionalOverride bool   `json:"institutional_override"`
	OverrideReference     string `json:"override_reference,omitempty"`
	OverrideApprover      string `json:"override_approver,omitempty"`
}

// ============================================================================
// ENHANCED RESPONSE MODELS WITH GOVERNANCE + ATTRIBUTION
// ============================================================================

// GovernedInteractionResult extends EnhancedInteractionResult with governance
type GovernedInteractionResult struct {
	EnhancedInteractionResult

	// Governance layer
	GovernanceAction    GovernanceAction    `json:"governance_action"`
	ActionDescription   string              `json:"action_description"`
	RequiresOverride    bool                `json:"requires_override"`
	OverrideLevel       string              `json:"override_level,omitempty"` // "clinical", "pharmacy", "governance"
	EscalationRequired  bool                `json:"escalation_required"`
	EscalationTarget    string              `json:"escalation_target,omitempty"`

	// Attribution envelope
	Attribution         AttributionEnvelope `json:"attribution"`

	// Medico-legal fields
	LegalDisclaimer     string   `json:"legal_disclaimer,omitempty"`
	DocumentationRequired bool   `json:"documentation_required"`
	AuditTrailID        string   `json:"audit_trail_id,omitempty"`
}

// GovernedInteractionCheckResponse wraps response with governance metadata
type GovernedInteractionCheckResponse struct {
	// Core response
	TransactionID      string                      `json:"transaction_id,omitempty"`
	DatasetVersion     string                      `json:"dataset_version"`

	// Governed interactions
	Interactions       []GovernedInteractionResult `json:"interactions"`

	// Summary with governance
	Summary            GovernedInteractionSummary  `json:"summary"`

	// Policy information
	PolicyApplied      string             `json:"policy_applied"`
	PolicyVersion      string             `json:"policy_version"`
	InstitutionID      string             `json:"institution_id,omitempty"`

	// Attribution header
	Attribution        ResponseAttribution `json:"attribution"`

	// Standard fields
	Recommendations    []string                    `json:"recommendations"`
	AlternativeDrugs   map[string]DrugAlternatives `json:"alternative_drugs,omitempty"`
	MonitoringPlan     []MonitoringRecommendation  `json:"monitoring_plan,omitempty"`
	CheckTimestamp     time.Time                   `json:"check_timestamp"`
	CacheHit           bool                        `json:"cache_hit,omitempty"`
}

// GovernedInteractionSummary extends summary with governance counts
type GovernedInteractionSummary struct {
	EnhancedInteractionSummary

	// Governance action counts
	HardBlockCount           int `json:"hard_block_count"`
	RequiresAcknowledgment   int `json:"requires_acknowledgment_count"`
	EscalationRequiredCount  int `json:"escalation_required_count"`

	// Highest governance action
	HighestGovernanceAction  GovernanceAction `json:"highest_governance_action"`

	// Blocking status
	IsOrderBlocked           bool   `json:"is_order_blocked"`
	BlockingReason           string `json:"blocking_reason,omitempty"`

	// Documentation requirements
	DocumentationRequired    bool     `json:"documentation_required"`
	RequiredDocumentation    []string `json:"required_documentation,omitempty"`
}

// ResponseAttribution provides top-level provenance for the entire response
type ResponseAttribution struct {
	// Service identification
	ServiceName        string    `json:"service_name"`
	ServiceVersion     string    `json:"service_version"`
	APIVersion         string    `json:"api_version"`

	// Dataset provenance
	DatasetVersion     string    `json:"dataset_version"`
	DatasetDate        time.Time `json:"dataset_date"`
	DatasetSource      string    `json:"dataset_source"`

	// Processing metadata
	ProcessedAt        time.Time `json:"processed_at"`
	ProcessingTimeMs   float64   `json:"processing_time_ms"`
	EnginesUsed        []string  `json:"engines_used"`

	// Evidence summary
	EvidenceSources    []ClinicalSource `json:"evidence_sources"`
	AverageConfidence  *decimal.Decimal `json:"average_confidence,omitempty"`

	// Governance policy
	GovernancePolicy   string    `json:"governance_policy"`
	PolicyEffectiveDate time.Time `json:"policy_effective_date"`

	// Regulatory compliance
	ComplianceFlags    []string  `json:"compliance_flags,omitempty"`

	// Audit
	AuditTrailID       string    `json:"audit_trail_id"`
	RequestHash        string    `json:"request_hash,omitempty"`
}

// ============================================================================
// DEFAULT GOVERNANCE POLICY
// Used when no institutional policy is configured
// ============================================================================

// DefaultGovernancePolicy returns the standard severity-to-action mapping
func DefaultGovernancePolicy() *SeverityGovernanceMapping {
	return &SeverityGovernanceMapping{
		PolicyName:            "default_clinical_safety",
		ContraindicatedAction: GovernanceHardBlock,
		MajorAction:           GovernanceWarnAcknowledge,
		ModerateAction:        GovernanceNotify,
		MinorAction:           GovernanceIgnore,
		UnknownAction:         GovernanceNotify,
		PediatricEscalation:   true,
		GeriatricEscalation:   true,
		RenalImpairmentUpgrade: true,
		HepaticImpairmentUpgrade: true,
		Active:                true,
	}
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// GetGovernanceActionDescription returns human-readable action description
func GetGovernanceActionDescription(action GovernanceAction) string {
	descriptions := map[GovernanceAction]string{
		GovernanceIgnore:              "No action required - informational only",
		GovernanceNotify:              "Notification sent to relevant parties",
		GovernanceWarnAcknowledge:     "Warning displayed - clinician acknowledgment required before proceeding",
		GovernanceHardBlock:           "Order blocked - cannot proceed under any circumstances",
		GovernanceHardBlockOverride:   "Order blocked - requires governance/P&T committee override",
		GovernanceMandatoryEscalation: "Mandatory escalation to senior clinician/pharmacist required",
	}
	if desc, ok := descriptions[action]; ok {
		return desc
	}
	return "Unknown governance action"
}

// GetOverrideLevel returns who can override a given governance action
func GetOverrideLevel(action GovernanceAction) string {
	levels := map[GovernanceAction]string{
		GovernanceIgnore:              "none",
		GovernanceNotify:              "none",
		GovernanceWarnAcknowledge:     "clinical",
		GovernanceHardBlock:           "not_overridable",
		GovernanceHardBlockOverride:   "governance",
		GovernanceMandatoryEscalation: "senior_clinical",
	}
	if level, ok := levels[action]; ok {
		return level
	}
	return "unknown"
}

// BuildAttributionEnvelope creates an attribution envelope from interaction data
func BuildAttributionEnvelope(
	datasetVersion string,
	sources []ClinicalSource,
	evidenceLevel EvidenceLevel,
	confidence *decimal.Decimal,
	governanceAction GovernanceAction,
	programFlags []ProgramFlag,
) AttributionEnvelope {
	return AttributionEnvelope{
		DatasetVersion:      datasetVersion,
		DatasetDate:         time.Now(), // Should be from dataset metadata
		ClinicalSources:     sources,
		EvidenceLevel:       evidenceLevel,
		ConfidenceScore:     confidence,
		GovernanceAction:    governanceAction,
		ProgramFlags:        programFlags,
		EvidenceStrength:    mapEvidenceLevelToStrength(evidenceLevel),
		GovernanceRelevance: []GovernanceRelevance{GovernanceRelevanceAdvisory},
	}
}

// mapEvidenceLevelToStrength converts EvidenceLevel to EvidenceStrength
func mapEvidenceLevelToStrength(level EvidenceLevel) EvidenceStrength {
	mapping := map[EvidenceLevel]EvidenceStrength{
		EvidenceLevelA:             EvidenceStrengthHigh,
		EvidenceLevelB:             EvidenceStrengthModerate,
		EvidenceLevelC:             EvidenceStrengthLow,
		EvidenceLevelD:             EvidenceStrengthVeryLow,
		EvidenceLevelExpertOpinion: EvidenceStrengthVeryLow,
		EvidenceLevelUnknown:       EvidenceStrengthUngraded,
	}
	if strength, ok := mapping[level]; ok {
		return strength
	}
	return EvidenceStrengthUngraded
}
