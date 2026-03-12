// Package models provides the unified governance types for all Knowledge Bases.
// KB-0 is the shared infrastructure for ingestion, approval workflows, and compliance.
package models

import (
	"context"
	"time"
)

// =============================================================================
// ENUMS
// =============================================================================

// KB represents a Knowledge Base identifier.
type KB string

const (
	KB1  KB = "KB-1"  // Drug Dosing Rules
	KB2  KB = "KB-2"  // Clinical Context
	KB3  KB = "KB-3"  // Temporal Logic
	KB4  KB = "KB-4"  // Patient Safety
	KB5  KB = "KB-5"  // Drug Interactions
	KB6  KB = "KB-6"  // Formulary
	KB7  KB = "KB-7"  // Terminology
	KB8  KB = "KB-8"  // Calculators
	KB9  KB = "KB-9"  // Care Gaps
	KB10 KB = "KB-10" // Rules Engine
	KB11 KB = "KB-11" // Population Health
	KB12 KB = "KB-12" // Order Sets
	KB13 KB = "KB-13" // Quality Measures
	KB14 KB = "KB-14" // Care Navigator
	KB15 KB = "KB-15" // Evidence Engine
	KB16 KB = "KB-16" // Lab Interpretation
	KB17 KB = "KB-17" // Population Registry
	KB18 KB = "KB-18" // Governance Engine
	KB19 KB = "KB-19" // Protocol Orchestrator
)

// KnowledgeType represents the type of knowledge item.
type KnowledgeType string

const (
	TypeDosingRule      KnowledgeType = "DOSING_RULE"
	TypeSafetyAlert     KnowledgeType = "SAFETY_ALERT"
	TypeInteraction     KnowledgeType = "INTERACTION"
	TypeFormularyEntry  KnowledgeType = "FORMULARY_ENTRY"
	TypeQualityMeasure  KnowledgeType = "QUALITY_MEASURE"
	TypeCareGap         KnowledgeType = "CARE_GAP"
	TypeOrderSet        KnowledgeType = "ORDER_SET"
	TypeProtocol        KnowledgeType = "PROTOCOL"
	TypeGuideline       KnowledgeType = "GUIDELINE"
	TypeLabRange        KnowledgeType = "LAB_RANGE"
	TypeCalculator      KnowledgeType = "CALCULATOR"
	TypeTerminology     KnowledgeType = "TERMINOLOGY"
	TypeValueSet        KnowledgeType = "VALUE_SET"
	TypeCQLLibrary      KnowledgeType = "CQL_LIBRARY"
)

// Authority represents a regulatory or clinical authority.
type Authority string

const (
	AuthorityFDA        Authority = "FDA"
	AuthorityTGA        Authority = "TGA"
	AuthorityCDSCO      Authority = "CDSCO"
	AuthorityEMA        Authority = "EMA"
	AuthorityMHRA       Authority = "MHRA"
	AuthorityNICE       Authority = "NICE"
	AuthorityCMS        Authority = "CMS"
	AuthorityNCQA       Authority = "NCQA"       // HEDIS
	AuthorityWHO        Authority = "WHO"
	AuthorityIDSA       Authority = "IDSA"
	AuthorityACCP       Authority = "ACCP"
	AuthorityACC_AHA    Authority = "ACC_AHA"
	AuthorityNLM        Authority = "NLM"        // RxNorm, SNOMED
	AuthoritySNOMED     Authority = "SNOMED"
	AuthorityLOINC       Authority = "LOINC"
	AuthorityRegenstrief Authority = "REGENSTRIEF" // LOINC parent organization
	AuthorityLexicomp    Authority = "LEXICOMP"
	AuthorityMicromedex  Authority = "MICROMEDEX"
	AuthorityHL7         Authority = "HL7"         // HL7 FHIR standards
	AuthorityInternal    Authority = "INTERNAL"

	// DDI-specific authorities (Three-Layer Authority Model)
	// Layer 2: Pharmacology Authorities
	AuthorityDrugBank     Authority = "DRUGBANK"      // Drug database with CYP/transporter data
	AuthorityPharmGKB     Authority = "PHARMGKB"      // Pharmacogenomics Knowledge Base
	AuthorityCredibleMeds Authority = "CREDIBLEMEDS"  // QT prolongation risk classification
	AuthorityUWDDI        Authority = "UW_DDI"        // University of Washington DDI Database
	AuthorityFlockhart    Authority = "FLOCKHART_CYP" // Indiana University Flockhart CYP Table

	// Layer 3: Clinical Practice Authorities
	AuthorityStockley Authority = "STOCKLEY" // Stockley's Drug Interactions reference
	AuthorityAMH      Authority = "AMH"      // Australian Medicines Handbook
	AuthorityBNF      Authority = "BNF"      // British National Formulary
)

// Jurisdiction represents where rules apply.
type Jurisdiction string

const (
	JurisdictionUS     Jurisdiction = "US"
	JurisdictionAU     Jurisdiction = "AU"
	JurisdictionIN     Jurisdiction = "IN"
	JurisdictionUK     Jurisdiction = "UK"
	JurisdictionEU     Jurisdiction = "EU"
	JurisdictionGlobal Jurisdiction = "GLOBAL"
)

// RiskLevel represents the risk classification.
type RiskLevel string

const (
	RiskHigh   RiskLevel = "HIGH"
	RiskMedium RiskLevel = "MEDIUM"
	RiskLow    RiskLevel = "LOW"
)

// WorkflowTemplate represents the approval workflow pattern.
type WorkflowTemplate string

const (
	TemplateClinicalHigh WorkflowTemplate = "CLINICAL_HIGH" // KB-1, KB-4, KB-5, KB-12, KB-19
	TemplateQualityMed   WorkflowTemplate = "QUALITY_MED"   // KB-6, KB-8, KB-9, KB-13, KB-15, KB-16
	TemplateInfraLow     WorkflowTemplate = "INFRA_LOW"     // KB-2, KB-3, KB-7, KB-10, KB-11, KB-14, KB-17, KB-18
)

// ItemState represents the approval state of a knowledge item.
type ItemState string

const (
	StateDraft           ItemState = "DRAFT"
	StatePrimaryReview   ItemState = "PRIMARY_REVIEW"
	StateSecondaryReview ItemState = "SECONDARY_REVIEW"
	StateReviewed        ItemState = "REVIEWED"
	StateDirectorApproval ItemState = "DIRECTOR_APPROVAL"
	StateCMOApproval     ItemState = "CMO_APPROVAL"
	StateApproved        ItemState = "APPROVED"
	StateActive          ItemState = "ACTIVE"
	StateHold            ItemState = "HOLD"
	StateRetired         ItemState = "RETIRED"
	StateRejected        ItemState = "REJECTED"
	StateRevise          ItemState = "REVISE"
	StateAutoValidation  ItemState = "AUTO_VALIDATION"
	StateLeadApproval    ItemState = "LEAD_APPROVAL"
	StateEmergencyActive ItemState = "EMERGENCY_ACTIVE"
)

// IsUsable returns true if the item can be used in production.
func (s ItemState) IsUsable() bool {
	return s == StateActive || s == StateEmergencyActive
}

// =============================================================================
// KNOWLEDGE ITEM (Universal Schema)
// =============================================================================

// KnowledgeItem represents any governed piece of knowledge across all KBs.
type KnowledgeItem struct {
	// Identity
	ID   string        `json:"id"`   // Unique ID (e.g., "kb1:warfarin:us:2025.1")
	KB   KB            `json:"kb"`   // Which Knowledge Base
	Type KnowledgeType `json:"type"` // Type of knowledge
	
	// Human-readable metadata
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	
	// Content reference
	ContentRef  string `json:"contentRef"`  // Pointer to KB-specific content
	ContentHash string `json:"contentHash"` // SHA256 for integrity
	
	// Source attribution
	Source SourceAttribution `json:"source"`
	
	// Classification
	RiskLevel        RiskLevel        `json:"riskLevel"`
	WorkflowTemplate WorkflowTemplate `json:"workflowTemplate"`
	RequiresDualReview bool           `json:"requiresDualReview"`
	
	// Additional risk flags (for high-risk items)
	RiskFlags RiskFlags `json:"riskFlags,omitempty"`
	
	// State
	State   ItemState `json:"state"`
	Version string    `json:"version"`
	
	// Governance trail
	Governance GovernanceTrail `json:"governance"`
	
	// Timestamps
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	ActiveAt  *time.Time `json:"activeAt,omitempty"`
	RetiredAt *time.Time `json:"retiredAt,omitempty"`
}

// SourceAttribution captures where the knowledge came from.
type SourceAttribution struct {
	Authority     Authority    `json:"authority"`
	Document      string       `json:"document"`
	Section       string       `json:"section,omitempty"`
	URL           string       `json:"url,omitempty"`
	Jurisdiction  Jurisdiction `json:"jurisdiction"`
	EffectiveDate string       `json:"effectiveDate,omitempty"`
	ExpirationDate string      `json:"expirationDate,omitempty"`
}

// RiskFlags captures additional risk classification.
type RiskFlags struct {
	HighAlertDrug     bool `json:"highAlertDrug,omitempty"`
	NarrowTherapeutic bool `json:"narrowTherapeutic,omitempty"`
	BlackBoxWarning   bool `json:"blackBoxWarning,omitempty"`
	ControlledSubstance bool `json:"controlledSubstance,omitempty"`
	Chemotherapy      bool `json:"chemotherapy,omitempty"`
	Pediatric         bool `json:"pediatric,omitempty"`
	Pregnancy         bool `json:"pregnancy,omitempty"`
}

// GovernanceTrail captures the full approval chain.
type GovernanceTrail struct {
	CreatedBy   string     `json:"createdBy"`
	CreatedAt   time.Time  `json:"createdAt"`
	Reviews     []Review   `json:"reviews,omitempty"`
	Approval    *Approval  `json:"approval,omitempty"`
	ActivatedAt *time.Time `json:"activatedAt,omitempty"`
	ActivatedBy string     `json:"activatedBy,omitempty"`
	RetiredAt   *time.Time `json:"retiredAt,omitempty"`
	RetiredBy   string     `json:"retiredBy,omitempty"`
	SupersededBy string    `json:"supersededBy,omitempty"`
}

// Review represents a single review action.
type Review struct {
	ID           string           `json:"id"`
	ReviewType   string           `json:"reviewType"` // PRIMARY, SECONDARY, SPECIALIST
	ReviewerID   string           `json:"reviewerId"`
	ReviewerName string           `json:"reviewerName"`
	Credentials  string           `json:"credentials,omitempty"`
	ReviewedAt   time.Time        `json:"reviewedAt"`
	Decision     string           `json:"decision"` // ACCEPT, REJECT, REVISE
	Checklist    *ReviewChecklist `json:"checklist,omitempty"`
	Notes        string           `json:"notes,omitempty"`
}

// ReviewChecklist represents the verification checklist.
type ReviewChecklist struct {
	Items []ChecklistItem `json:"items"`
}

// ChecklistItem represents a single checklist item.
type ChecklistItem struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Required bool   `json:"required"`
	Verified bool   `json:"verified"`
	Notes    string `json:"notes,omitempty"`
}

// Approval represents the final approval action.
type Approval struct {
	ApproverID     string    `json:"approverId"`
	ApproverName   string    `json:"approverName"`
	ApproverRole   string    `json:"approverRole"` // CMO, DIRECTOR, LEAD
	Credentials    string    `json:"credentials,omitempty"`
	ApprovedAt     time.Time `json:"approvedAt"`
	Decision       string    `json:"decision"` // APPROVE, REJECT, HOLD
	Notes          string    `json:"notes,omitempty"`
	Attestations   map[string]bool `json:"attestations,omitempty"`
}

// =============================================================================
// KB REGISTRY
// =============================================================================

// KBConfig represents configuration for a Knowledge Base.
type KBConfig struct {
	ID               KB               `json:"id"`
	Name             string           `json:"name"`
	Description      string           `json:"description"`
	RiskLevel        RiskLevel        `json:"riskLevel"`
	WorkflowTemplate WorkflowTemplate `json:"workflowTemplate"`
	KnowledgeTypes   []KnowledgeType  `json:"knowledgeTypes"`
	Authorities      []Authority      `json:"authorities"`
	Jurisdictions    []Jurisdiction   `json:"jurisdictions"`
	RequiresDualReview bool           `json:"requiresDualReview"`
	ReviewerRoles    []string         `json:"reviewerRoles"`
	ApproverRole     string           `json:"approverRole"`
}

// KBRegistry holds configuration for all Knowledge Bases.
var KBRegistry = map[KB]KBConfig{
	KB1: {
		ID:               KB1,
		Name:             "Drug Dosing Rules",
		Description:      "Comprehensive drug dosing calculation and validation",
		RiskLevel:        RiskHigh,
		WorkflowTemplate: TemplateClinicalHigh,
		KnowledgeTypes:   []KnowledgeType{TypeDosingRule},
		Authorities:      []Authority{AuthorityFDA, AuthorityTGA, AuthorityCDSCO, AuthorityEMA},
		Jurisdictions:    []Jurisdiction{JurisdictionUS, JurisdictionAU, JurisdictionIN, JurisdictionEU},
		RequiresDualReview: true,
		ReviewerRoles:    []string{"pharmacist"},
		ApproverRole:     "cmo",
	},
	KB4: {
		ID:               KB4,
		Name:             "Patient Safety Alerts",
		Description:      "Drug safety alerts, contraindications, and warnings",
		RiskLevel:        RiskHigh,
		WorkflowTemplate: TemplateClinicalHigh,
		KnowledgeTypes:   []KnowledgeType{TypeSafetyAlert},
		Authorities:      []Authority{AuthorityFDA, AuthorityTGA, AuthorityCDSCO},
		Jurisdictions:    []Jurisdiction{JurisdictionUS, JurisdictionAU, JurisdictionIN},
		RequiresDualReview: true,
		ReviewerRoles:    []string{"pharmacist"},
		ApproverRole:     "cmo",
	},
	KB5: {
		ID:               KB5,
		Name:             "Drug Interactions",
		Description:      "Drug-drug, drug-food, drug-condition interactions",
		RiskLevel:        RiskHigh,
		WorkflowTemplate: TemplateClinicalHigh,
		KnowledgeTypes:   []KnowledgeType{TypeInteraction},
		// Three-Layer Authority Model:
		// Layer 1 (Regulatory): FDA, TGA, CDSCO, EMA
		// Layer 2 (Pharmacology): DrugBank, PharmGKB, CredibleMeds, UW DDI, Flockhart CYP
		// Layer 3 (Clinical): Lexicomp, Micromedex, Stockley, AMH, BNF
		Authorities: []Authority{
			// Layer 1: Regulatory
			AuthorityFDA, AuthorityTGA, AuthorityCDSCO, AuthorityEMA,
			// Layer 2: Pharmacology
			AuthorityDrugBank, AuthorityPharmGKB, AuthorityCredibleMeds, AuthorityUWDDI, AuthorityFlockhart,
			// Layer 3: Clinical Practice
			AuthorityLexicomp, AuthorityMicromedex, AuthorityStockley, AuthorityAMH, AuthorityBNF,
		},
		Jurisdictions:      []Jurisdiction{JurisdictionGlobal, JurisdictionUS, JurisdictionAU, JurisdictionIN, JurisdictionUK, JurisdictionEU},
		RequiresDualReview: true,
		ReviewerRoles:      []string{"pharmacist"},
		ApproverRole:       "cmo",
	},
	KB6: {
		ID:               KB6,
		Name:             "Formulary Management",
		Description:      "Hospital formulary, NLEM, PBS listings",
		RiskLevel:        RiskMedium,
		WorkflowTemplate: TemplateQualityMed,
		KnowledgeTypes:   []KnowledgeType{TypeFormularyEntry},
		Authorities:      []Authority{AuthorityInternal, AuthorityFDA, AuthorityTGA},
		Jurisdictions:    []Jurisdiction{JurisdictionUS, JurisdictionAU, JurisdictionIN},
		RequiresDualReview: false,
		ReviewerRoles:    []string{"pharmacist"},
		ApproverRole:     "pt_chair",
	},
	KB7: {
		ID:               KB7,
		Name:             "Terminology Service",
		Description:      "RxNorm, SNOMED CT, ICD-10, LOINC terminology",
		RiskLevel:        RiskLow,
		WorkflowTemplate: TemplateInfraLow,
		KnowledgeTypes:   []KnowledgeType{TypeTerminology, TypeValueSet},
		Authorities:      []Authority{AuthorityNLM, AuthoritySNOMED, AuthorityLOINC},
		Jurisdictions:    []Jurisdiction{JurisdictionGlobal, JurisdictionUS, JurisdictionAU},
		RequiresDualReview: false,
		ReviewerRoles:    []string{"terminology_manager"},
		ApproverRole:     "terminology_manager",
	},
	KB8: {
		ID:               KB8,
		Name:             "Clinical Calculators",
		Description:      "Clinical scores and calculators (CHA2DS2-VASc, MELD, etc.)",
		RiskLevel:        RiskMedium,
		WorkflowTemplate: TemplateQualityMed,
		KnowledgeTypes:   []KnowledgeType{TypeCalculator},
		Authorities:      []Authority{AuthorityInternal, AuthorityACC_AHA},
		Jurisdictions:    []Jurisdiction{JurisdictionGlobal},
		RequiresDualReview: false,
		ReviewerRoles:    []string{"physician"},
		ApproverRole:     "clinical_lead",
	},
	KB9: {
		ID:               KB9,
		Name:             "Care Gaps",
		Description:      "Preventive care and chronic disease management gaps",
		RiskLevel:        RiskMedium,
		WorkflowTemplate: TemplateQualityMed,
		KnowledgeTypes:   []KnowledgeType{TypeCareGap, TypeQualityMeasure},
		Authorities:      []Authority{AuthorityCMS, AuthorityNCQA},
		Jurisdictions:    []Jurisdiction{JurisdictionUS},
		RequiresDualReview: false,
		ReviewerRoles:    []string{"quality_analyst"},
		ApproverRole:     "quality_director",
	},
	KB12: {
		ID:               KB12,
		Name:             "Order Sets",
		Description:      "Clinical order set templates",
		RiskLevel:        RiskHigh,
		WorkflowTemplate: TemplateClinicalHigh,
		KnowledgeTypes:   []KnowledgeType{TypeOrderSet},
		Authorities:      []Authority{AuthorityInternal, AuthorityIDSA, AuthorityACC_AHA},
		Jurisdictions:    []Jurisdiction{JurisdictionUS, JurisdictionAU, JurisdictionIN},
		RequiresDualReview: true,
		ReviewerRoles:    []string{"physician", "pharmacist"},
		ApproverRole:     "cmo",
	},
	KB13: {
		ID:               KB13,
		Name:             "Quality Measures",
		Description:      "CMS eCQM and HEDIS quality measures",
		RiskLevel:        RiskMedium,
		WorkflowTemplate: TemplateQualityMed,
		KnowledgeTypes:   []KnowledgeType{TypeQualityMeasure, TypeCQLLibrary},
		Authorities:      []Authority{AuthorityCMS, AuthorityNCQA},
		Jurisdictions:    []Jurisdiction{JurisdictionUS},
		RequiresDualReview: false,
		ReviewerRoles:    []string{"quality_analyst"},
		ApproverRole:     "quality_director",
	},
	KB15: {
		ID:               KB15,
		Name:             "Evidence Engine",
		Description:      "Clinical evidence and literature references",
		RiskLevel:        RiskMedium,
		WorkflowTemplate: TemplateQualityMed,
		KnowledgeTypes:   []KnowledgeType{TypeGuideline},
		Authorities:      []Authority{AuthorityIDSA, AuthorityACC_AHA, AuthorityNICE},
		Jurisdictions:    []Jurisdiction{JurisdictionGlobal},
		RequiresDualReview: false,
		ReviewerRoles:    []string{"physician"},
		ApproverRole:     "clinical_lead",
	},
	KB16: {
		ID:               KB16,
		Name:             "Lab Interpretation",
		Description:      "Laboratory reference ranges and interpretations",
		RiskLevel:        RiskMedium,
		WorkflowTemplate: TemplateQualityMed,
		KnowledgeTypes:   []KnowledgeType{TypeLabRange},
		Authorities:      []Authority{AuthorityLOINC, AuthorityInternal},
		Jurisdictions:    []Jurisdiction{JurisdictionUS, JurisdictionAU, JurisdictionIN},
		RequiresDualReview: false,
		ReviewerRoles:    []string{"pathologist"},
		ApproverRole:     "lab_director",
	},
	KB19: {
		ID:               KB19,
		Name:             "Protocol Orchestrator",
		Description:      "Clinical protocol and guideline orchestration",
		RiskLevel:        RiskHigh,
		WorkflowTemplate: TemplateClinicalHigh,
		KnowledgeTypes:   []KnowledgeType{TypeProtocol, TypeGuideline, TypeCQLLibrary},
		Authorities:      []Authority{AuthorityIDSA, AuthorityACC_AHA, AuthorityACCP, AuthorityNICE},
		Jurisdictions:    []Jurisdiction{JurisdictionUS, JurisdictionAU, JurisdictionIN, JurisdictionUK},
		RequiresDualReview: true,
		ReviewerRoles:    []string{"specialist", "pharmacist"},
		ApproverRole:     "cmo",
	},
	KB2: {
		ID:               KB2,
		Name:             "Clinical Context",
		Description:      "Patient clinical context and history aggregation",
		RiskLevel:        RiskLow,
		WorkflowTemplate: TemplateInfraLow,
		KnowledgeTypes:   []KnowledgeType{TypeTerminology, TypeValueSet},
		Authorities:      []Authority{AuthorityHL7, AuthoritySNOMED, AuthorityInternal},
		Jurisdictions:    []Jurisdiction{JurisdictionGlobal},
		RequiresDualReview: false,
		ReviewerRoles:    []string{"tech_lead"},
		ApproverRole:     "tech_lead",
	},
	KB3: {
		ID:               KB3,
		Name:             "Temporal Logic",
		Description:      "Time-based clinical logic and scheduling rules",
		RiskLevel:        RiskLow,
		WorkflowTemplate: TemplateInfraLow,
		KnowledgeTypes:   []KnowledgeType{TypeCQLLibrary},
		Authorities:      []Authority{AuthorityInternal, AuthorityHL7},
		Jurisdictions:    []Jurisdiction{JurisdictionGlobal},
		RequiresDualReview: false,
		ReviewerRoles:    []string{"tech_lead"},
		ApproverRole:     "tech_lead",
	},
	KB10: {
		ID:               KB10,
		Name:             "Rules Engine",
		Description:      "Clinical decision support rules and inference engine",
		RiskLevel:        RiskLow,
		WorkflowTemplate: TemplateInfraLow,
		KnowledgeTypes:   []KnowledgeType{TypeCQLLibrary, TypeProtocol},
		Authorities:      []Authority{AuthorityHL7, AuthorityInternal},
		Jurisdictions:    []Jurisdiction{JurisdictionGlobal},
		RequiresDualReview: false,
		ReviewerRoles:    []string{"tech_lead", "clinical_informaticist"},
		ApproverRole:     "tech_lead",
	},
	KB11: {
		ID:               KB11,
		Name:             "Population Health",
		Description:      "Population health analytics and cohort definitions",
		RiskLevel:        RiskLow,
		WorkflowTemplate: TemplateInfraLow,
		KnowledgeTypes:   []KnowledgeType{TypeCQLLibrary, TypeQualityMeasure},
		Authorities:      []Authority{AuthorityCMS, AuthorityNCQA, AuthorityInternal},
		Jurisdictions:    []Jurisdiction{JurisdictionUS, JurisdictionGlobal},
		RequiresDualReview: false,
		ReviewerRoles:    []string{"analytics_lead", "quality_analyst"},
		ApproverRole:     "analytics_lead",
	},
	KB14: {
		ID:               KB14,
		Name:             "Care Navigator",
		Description:      "Patient care pathway navigation and workflow guidance",
		RiskLevel:        RiskLow,
		WorkflowTemplate: TemplateInfraLow,
		KnowledgeTypes:   []KnowledgeType{TypeProtocol, TypeGuideline},
		Authorities:      []Authority{AuthorityInternal, AuthorityACC_AHA, AuthorityIDSA},
		Jurisdictions:    []Jurisdiction{JurisdictionUS, JurisdictionAU, JurisdictionIN},
		RequiresDualReview: false,
		ReviewerRoles:    []string{"clinical_informaticist"},
		ApproverRole:     "clinical_lead",
	},
	KB17: {
		ID:               KB17,
		Name:             "Population Registry",
		Description:      "Patient population registries and cohort tracking",
		RiskLevel:        RiskLow,
		WorkflowTemplate: TemplateInfraLow,
		KnowledgeTypes:   []KnowledgeType{TypeValueSet, TypeCQLLibrary},
		Authorities:      []Authority{AuthorityInternal, AuthorityCMS, AuthorityNCQA},
		Jurisdictions:    []Jurisdiction{JurisdictionUS, JurisdictionGlobal},
		RequiresDualReview: false,
		ReviewerRoles:    []string{"analytics_lead"},
		ApproverRole:     "analytics_lead",
	},
	KB18: {
		ID:               KB18,
		Name:             "Governance Engine",
		Description:      "Knowledge governance workflow and compliance engine",
		RiskLevel:        RiskLow,
		WorkflowTemplate: TemplateInfraLow,
		KnowledgeTypes:   []KnowledgeType{TypeProtocol},
		Authorities:      []Authority{AuthorityInternal},
		Jurisdictions:    []Jurisdiction{JurisdictionGlobal},
		RequiresDualReview: false,
		ReviewerRoles:    []string{"tech_lead"},
		ApproverRole:     "tech_lead",
	},
}

// =============================================================================
// INGESTION ADAPTER INTERFACE
// =============================================================================

// UpdateInfo represents information about an available update.
type UpdateInfo struct {
	SourceID     string    `json:"sourceId"`
	Authority    Authority `json:"authority"`
	DocumentName string    `json:"documentName"`
	Version      string    `json:"version"`
	UpdatedAt    time.Time `json:"updatedAt"`
	ChangeType   string    `json:"changeType"` // NEW, UPDATE, RETIRE
}

// RawContent represents raw content from a source.
type RawContent struct {
	SourceID    string            `json:"sourceId"`
	Authority   Authority         `json:"authority"`
	RawData     []byte            `json:"rawData"`
	ContentType string            `json:"contentType"` // application/xml, application/pdf, etc.
	Metadata    map[string]string `json:"metadata"`
	FetchedAt   time.Time         `json:"fetchedAt"`
}

// ValidationError represents a validation error.
type ValidationError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// IngestionAdapter defines the interface for source adapters.
type IngestionAdapter interface {
	// Metadata
	GetName() string
	GetAuthority() Authority
	GetSupportedKBs() []KB
	GetSupportedTypes() []KnowledgeType
	
	// Discovery
	CheckForUpdates(ctx context.Context, since time.Time) ([]UpdateInfo, error)
	
	// Fetching
	Fetch(ctx context.Context, sourceID string) (*RawContent, error)
	FetchAll(ctx context.Context) ([]*RawContent, error)
	
	// Parsing
	Parse(ctx context.Context, raw *RawContent) ([]map[string]interface{}, error)
	
	// Transformation (KB-specific)
	Transform(ctx context.Context, parsed map[string]interface{}, targetKB KB) (*KnowledgeItem, error)
	
	// Validation
	Validate(ctx context.Context, item *KnowledgeItem) ([]ValidationError, error)
}

// =============================================================================
// WORKFLOW TEMPLATE DEFINITIONS
// =============================================================================

// WorkflowTemplateDefinition defines a workflow template.
type WorkflowTemplateDefinition struct {
	ID          WorkflowTemplate `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	States      []ItemState      `json:"states"`
	Transitions []StateTransition `json:"transitions"`
	Checklist   []ChecklistTemplate `json:"checklist"`
	SLA         SLAConfig        `json:"sla"`
}

// StateTransition defines an allowed state transition.
type StateTransition struct {
	From        []ItemState `json:"from"`
	To          ItemState   `json:"to"`
	ActorRoles  []string    `json:"actorRoles"`
	Action      string      `json:"action"`
	Condition   string      `json:"condition,omitempty"`
	Requires    []string    `json:"requires,omitempty"`
}

// ChecklistTemplate defines a checklist item template.
type ChecklistTemplate struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Required bool   `json:"required"`
	AppliesWhen string `json:"appliesWhen,omitempty"`
}

// SLAConfig defines SLA targets.
type SLAConfig struct {
	ReviewTarget    time.Duration `json:"reviewTarget"`
	ApprovalTarget  time.Duration `json:"approvalTarget"`
	EscalationAfter time.Duration `json:"escalationAfter"`
}

// WorkflowTemplates holds all workflow template definitions.
var WorkflowTemplates = map[WorkflowTemplate]WorkflowTemplateDefinition{
	TemplateClinicalHigh: {
		ID:          TemplateClinicalHigh,
		Name:        "High-Risk Clinical Content",
		Description: "For drug dosing, safety alerts, interactions, order sets, protocols",
		States: []ItemState{
			StateDraft, StatePrimaryReview, StateSecondaryReview,
			StateCMOApproval, StateApproved, StateActive, StateHold, StateRetired, StateRejected,
		},
		Transitions: []StateTransition{
			{From: []ItemState{StateDraft}, To: StatePrimaryReview, ActorRoles: []string{"pharmacist", "physician"}, Action: "submit_review"},
			{From: []ItemState{StatePrimaryReview}, To: StateSecondaryReview, ActorRoles: []string{"pharmacist", "physician"}, Action: "submit_review", Condition: "requires_dual_review"},
			{From: []ItemState{StatePrimaryReview, StateSecondaryReview}, To: StateCMOApproval, ActorRoles: []string{"system"}, Action: "route_to_approval", Condition: "all_reviews_complete"},
			{From: []ItemState{StateCMOApproval}, To: StateApproved, ActorRoles: []string{"cmo"}, Action: "approve", Requires: []string{"medical_responsibility", "clinical_standards"}},
			{From: []ItemState{StateApproved}, To: StateActive, ActorRoles: []string{"system"}, Action: "activate"},
			{From: []ItemState{StateActive}, To: StateRetired, ActorRoles: []string{"system"}, Action: "retire", Condition: "superseded"},
			{From: []ItemState{StateActive, StateApproved}, To: StateHold, ActorRoles: []string{"cmo"}, Action: "hold"},
			// Rejection transitions - allow rejection at any review or approval stage
			{From: []ItemState{StatePrimaryReview}, To: StateRejected, ActorRoles: []string{"pharmacist", "physician"}, Action: "reject"},
			{From: []ItemState{StateSecondaryReview}, To: StateRejected, ActorRoles: []string{"pharmacist", "physician"}, Action: "reject"},
			{From: []ItemState{StateCMOApproval}, To: StateRejected, ActorRoles: []string{"cmo"}, Action: "reject"},
		},
		Checklist: []ChecklistTemplate{
			{ID: "dose_verification", Label: "Dose verified against regulatory label", Required: true},
			{ID: "renal_adjustment", Label: "Renal adjustments verified", Required: true},
			{ID: "hepatic_adjustment", Label: "Hepatic adjustments verified", Required: true},
			{ID: "pediatric_dosing", Label: "Pediatric dosing verified", Required: false, AppliesWhen: "has_pediatric"},
			{ID: "geriatric_dosing", Label: "Geriatric dosing verified", Required: false, AppliesWhen: "has_geriatric"},
			{ID: "interactions_checked", Label: "Drug interactions reviewed", Required: true},
			{ID: "monitoring_validated", Label: "Monitoring requirements validated", Required: true},
			{ID: "black_box_confirmed", Label: "Black box warning confirmed", Required: false, AppliesWhen: "has_black_box"},
			{ID: "contraindications_verified", Label: "Contraindications verified", Required: true},
		},
		SLA: SLAConfig{
			ReviewTarget:    24 * time.Hour,
			ApprovalTarget:  48 * time.Hour,
			EscalationAfter: 72 * time.Hour,
		},
	},
	TemplateQualityMed: {
		ID:          TemplateQualityMed,
		Name:        "Medium-Risk Quality Content",
		Description: "For formulary, calculators, quality measures, evidence, lab ranges",
		States: []ItemState{
			StateDraft, StateReviewed, StateDirectorApproval, StateApproved, StateActive, StateRetired,
		},
		Transitions: []StateTransition{
			{From: []ItemState{StateDraft}, To: StateReviewed, ActorRoles: []string{"quality_analyst", "specialist", "pharmacist", "pathologist"}, Action: "submit_review"},
			{From: []ItemState{StateReviewed}, To: StateDirectorApproval, ActorRoles: []string{"system"}, Action: "route_to_approval"},
			{From: []ItemState{StateDirectorApproval}, To: StateApproved, ActorRoles: []string{"quality_director", "clinical_lead", "pt_chair", "lab_director"}, Action: "approve"},
			{From: []ItemState{StateApproved}, To: StateActive, ActorRoles: []string{"system"}, Action: "activate"},
		},
		Checklist: []ChecklistTemplate{
			{ID: "content_accuracy", Label: "Content accuracy verified", Required: true},
			{ID: "source_validated", Label: "Source document validated", Required: true},
			{ID: "jurisdiction_appropriate", Label: "Jurisdiction appropriateness confirmed", Required: true},
		},
		SLA: SLAConfig{
			ReviewTarget:    24 * time.Hour,
			ApprovalTarget:  24 * time.Hour,
			EscalationAfter: 48 * time.Hour,
		},
	},
	TemplateInfraLow: {
		ID:          TemplateInfraLow,
		Name:        "Low-Risk Infrastructure Content",
		Description: "For terminology, context, temporal logic, rules engine, analytics",
		States: []ItemState{
			StateDraft, StateAutoValidation, StateLeadApproval, StateActive, StateRetired,
		},
		Transitions: []StateTransition{
			{From: []ItemState{StateDraft}, To: StateAutoValidation, ActorRoles: []string{"system"}, Action: "auto_validate"},
			{From: []ItemState{StateAutoValidation}, To: StateLeadApproval, ActorRoles: []string{"system"}, Action: "route_to_approval", Condition: "validation_passed"},
			{From: []ItemState{StateLeadApproval}, To: StateActive, ActorRoles: []string{"tech_lead", "terminology_manager", "analytics_lead"}, Action: "approve"},
		},
		Checklist: []ChecklistTemplate{
			{ID: "schema_valid", Label: "Schema validation passed", Required: true},
			{ID: "reference_integrity", Label: "Reference integrity verified", Required: true},
			{ID: "regression_tests", Label: "Regression tests passed", Required: true},
		},
		SLA: SLAConfig{
			ReviewTarget:    1 * time.Hour,
			ApprovalTarget:  24 * time.Hour,
			EscalationAfter: 48 * time.Hour,
		},
	},
}

// =============================================================================
// AUDIT ACTIONS
// =============================================================================

// AuditAction represents an auditable action.
type AuditAction string

const (
	AuditItemCreated      AuditAction = "ITEM_CREATED"
	AuditItemIngested     AuditAction = "ITEM_INGESTED"
	AuditItemReviewed     AuditAction = "ITEM_REVIEWED"
	AuditItemApproved     AuditAction = "ITEM_APPROVED"
	AuditItemActivated    AuditAction = "ITEM_ACTIVATED"
	AuditItemRetired      AuditAction = "ITEM_RETIRED"
	AuditItemRejected     AuditAction = "ITEM_REJECTED"
	AuditItemHeld         AuditAction = "ITEM_HELD"
	AuditItemRevised      AuditAction = "ITEM_SENT_TO_REVISE"
	AuditEmergencyOverride AuditAction = "EMERGENCY_OVERRIDE"
	AuditEmergencyExpired AuditAction = "EMERGENCY_EXPIRED"
)

// AuditEntry represents an immutable audit log entry.
type AuditEntry struct {
	ID            string      `json:"id"`
	Timestamp     time.Time   `json:"timestamp"`
	Action        AuditAction `json:"action"`
	
	// Actor
	ActorID       string `json:"actorId"`
	ActorName     string `json:"actorName"`
	ActorRole     string `json:"actorRole"`
	Credentials   string `json:"credentials,omitempty"`
	
	// Item reference
	ItemID        string    `json:"itemId"`
	KB            KB        `json:"kb"`
	ItemVersion   string    `json:"itemVersion"`
	
	// State transition
	PreviousState ItemState `json:"previousState,omitempty"`
	NewState      ItemState `json:"newState"`
	
	// Decision details
	Decision      string          `json:"decision,omitempty"`
	Notes         string          `json:"notes,omitempty"`
	Checklist     *ReviewChecklist `json:"checklist,omitempty"`
	Attestations  map[string]bool `json:"attestations,omitempty"`
	
	// Request metadata
	IPAddress     string `json:"ipAddress,omitempty"`
	SessionID     string `json:"sessionId,omitempty"`
	UserAgent     string `json:"userAgent,omitempty"`
	
	// Content integrity
	ContentHash   string `json:"contentHash,omitempty"`
}
