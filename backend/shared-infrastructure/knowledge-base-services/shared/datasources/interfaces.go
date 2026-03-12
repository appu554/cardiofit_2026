// Package datasources defines interfaces for all external data sources.
// These interfaces enable dependency injection and testability while
// ensuring consistent behavior across all KB services.
//
// DESIGN PRINCIPLE: "Program to interfaces, not implementations"
// All data source clients implement these interfaces, making them swappable.
package datasources

import (
	"context"
	"time"
)

// =============================================================================
// BASE DATA SOURCE INTERFACE
// =============================================================================

// DataSource is the base interface that all data sources must implement
type DataSource interface {
	// Name returns the unique identifier for this data source
	Name() string

	// HealthCheck verifies the data source is available and responding
	HealthCheck(ctx context.Context) error

	// Close releases any resources held by the data source
	Close() error
}

// =============================================================================
// RXNAV / RXCLASS INTERFACES
// =============================================================================

// RxNavClient provides access to the NLM RxNav REST API
// API Docs: https://lhncbc.nlm.nih.gov/RxNav/APIs/RxNormAPIs.html
type RxNavClient interface {
	DataSource

	// ─────────────────────────────────────────────────────────────────────────
	// DRUG LOOKUP
	// ─────────────────────────────────────────────────────────────────────────

	// GetRxCUIByName finds the RxCUI for a drug name
	GetRxCUIByName(ctx context.Context, drugName string) (string, error)

	// GetRxCUIByNDC finds the RxCUI for a National Drug Code
	GetRxCUIByNDC(ctx context.Context, ndc string) (string, error)

	// GetDrugByRxCUI retrieves drug details by RxCUI
	GetDrugByRxCUI(ctx context.Context, rxcui string) (*RxNormDrug, error)

	// SearchDrugs searches for drugs matching a query
	SearchDrugs(ctx context.Context, query string, limit int) ([]RxNormDrug, error)

	// ─────────────────────────────────────────────────────────────────────────
	// RELATIONSHIPS
	// ─────────────────────────────────────────────────────────────────────────

	// GetRelatedByType finds related concepts by relationship type
	GetRelatedByType(ctx context.Context, rxcui string, relType RxNormRelationType) ([]RxNormConcept, error)

	// GetAllRelated retrieves all relationships for a drug
	GetAllRelated(ctx context.Context, rxcui string) (*RxNormRelationships, error)

	// GetIngredients returns the active ingredients for a drug product
	GetIngredients(ctx context.Context, rxcui string) ([]RxNormConcept, error)

	// ─────────────────────────────────────────────────────────────────────────
	// NDC OPERATIONS
	// ─────────────────────────────────────────────────────────────────────────

	// GetNDCsByRxCUI returns all NDCs associated with an RxCUI
	GetNDCsByRxCUI(ctx context.Context, rxcui string) ([]string, error)

	// GetNDCProperties retrieves properties for a specific NDC
	GetNDCProperties(ctx context.Context, ndc string) (*NDCProperties, error)

	// ─────────────────────────────────────────────────────────────────────────
	// INTERACTIONS
	// ─────────────────────────────────────────────────────────────────────────

	// GetInteractions retrieves drug-drug interactions for an RxCUI
	GetInteractions(ctx context.Context, rxcui string) ([]DrugInteraction, error)

	// GetInteractionsBetween checks for interactions between multiple drugs
	GetInteractionsBetween(ctx context.Context, rxcuis []string) ([]DrugInteraction, error)

	// ─────────────────────────────────────────────────────────────────────────
	// BATCH OPERATIONS
	// ─────────────────────────────────────────────────────────────────────────

	// BatchGetDrugs retrieves multiple drugs by RxCUI
	BatchGetDrugs(ctx context.Context, rxcuis []string) (map[string]*RxNormDrug, error)
}

// RxClassClient provides access to the NLM RxClass API
// API Docs: https://lhncbc.nlm.nih.gov/RxNav/APIs/RxClassAPIs.html
type RxClassClient interface {
	DataSource

	// ─────────────────────────────────────────────────────────────────────────
	// CLASSIFICATION QUERIES
	// ─────────────────────────────────────────────────────────────────────────

	// GetClassByRxCUI returns drug classes for an RxCUI
	GetClassByRxCUI(ctx context.Context, rxcui string, classTypes []RxClassType) ([]DrugClass, error)

	// GetDrugsByClass returns all drugs in a class
	GetDrugsByClass(ctx context.Context, classID string, classType RxClassType) ([]RxNormConcept, error)

	// GetClassMembers returns all members of a drug class
	GetClassMembers(ctx context.Context, classID string, classType RxClassType) ([]ClassMember, error)

	// ─────────────────────────────────────────────────────────────────────────
	// RELATIONSHIP QUERIES (MED-RT)
	// ─────────────────────────────────────────────────────────────────────────

	// GetContraindications returns contraindicated conditions for a drug
	GetContraindications(ctx context.Context, rxcui string) ([]MedicalCondition, error)

	// GetIndications returns conditions the drug may treat
	GetIndications(ctx context.Context, rxcui string) ([]MedicalCondition, error)

	// GetPhysiologicEffects returns physiologic effects of a drug
	GetPhysiologicEffects(ctx context.Context, rxcui string) ([]PhysiologicEffect, error)

	// GetMechanismOfAction returns the drug's mechanism of action
	GetMechanismOfAction(ctx context.Context, rxcui string) ([]MechanismOfAction, error)

	// ─────────────────────────────────────────────────────────────────────────
	// RENAL-SPECIFIC QUERIES
	// ─────────────────────────────────────────────────────────────────────────

	// IsRenallyExcreted checks if a drug is primarily renally excreted
	IsRenallyExcreted(ctx context.Context, rxcui string) (bool, error)

	// HasRenalDoseAdjustment checks if drug requires renal dose adjustment
	HasRenalDoseAdjustment(ctx context.Context, rxcui string) (bool, error)

	// GetRenalRelatedClasses returns renal-related drug classes
	GetRenalRelatedClasses(ctx context.Context, rxcui string) ([]DrugClass, error)
}

// =============================================================================
// RXNAV DATA MODELS
// =============================================================================

// RxNormDrug represents a drug from RxNorm
type RxNormDrug struct {
	RxCUI        string   `json:"rxcui"`
	Name         string   `json:"name"`
	Synonym      string   `json:"synonym,omitempty"`
	TTY          string   `json:"tty"` // Term Type (IN, MIN, SCD, SBD, etc.)
	Language     string   `json:"language,omitempty"`
	Suppress     string   `json:"suppress,omitempty"`
	BrandNames   []string `json:"brandNames,omitempty"`
	GenericName  string   `json:"genericName,omitempty"`
	DoseForm     string   `json:"doseForm,omitempty"`
	Strength     string   `json:"strength,omitempty"`
	Ingredients  []string `json:"ingredients,omitempty"`
}

// RxNormConcept represents a concept from RxNorm
type RxNormConcept struct {
	RxCUI string `json:"rxcui"`
	Name  string `json:"name"`
	TTY   string `json:"tty"`
}

// RxNormRelationType defines relationship types in RxNorm
type RxNormRelationType string

const (
	RelTypeIngredient   RxNormRelationType = "ingredient_of"
	RelTypeTradename    RxNormRelationType = "tradename_of"
	RelTypeDoseForm     RxNormRelationType = "dose_form_of"
	RelTypeContains     RxNormRelationType = "contains"
	RelTypeIsA          RxNormRelationType = "isa"
	RelTypeConstitutes  RxNormRelationType = "constitutes"
	RelTypeQuantifiedBy RxNormRelationType = "quantified_form_of"
)

// RxNormRelationships contains all relationships for a drug
type RxNormRelationships struct {
	RxCUI         string          `json:"rxcui"`
	Ingredients   []RxNormConcept `json:"ingredients"`
	BrandNames    []RxNormConcept `json:"brandNames"`
	DoseForms     []RxNormConcept `json:"doseForms"`
	Components    []RxNormConcept `json:"components"`
	RelatedDrugs  []RxNormConcept `json:"relatedDrugs"`
}

// NDCProperties contains properties for an NDC
type NDCProperties struct {
	NDC              string    `json:"ndc"`
	RxCUI            string    `json:"rxcui"`
	PackagingNDC     string    `json:"packagingNdc,omitempty"`
	Labeler          string    `json:"labeler,omitempty"`
	Status           string    `json:"status,omitempty"`
	StartMarketDate  time.Time `json:"startMarketDate,omitempty"`
	EndMarketDate    time.Time `json:"endMarketDate,omitempty"`
}

// DrugInteraction represents a drug-drug interaction
type DrugInteraction struct {
	Drug1RxCUI  string `json:"drug1Rxcui"`
	Drug1Name   string `json:"drug1Name"`
	Drug2RxCUI  string `json:"drug2Rxcui"`
	Drug2Name   string `json:"drug2Name"`
	Severity    string `json:"severity"` // minor, moderate, major
	Description string `json:"description"`
	Source      string `json:"source"`
}

// =============================================================================
// RXCLASS DATA MODELS
// =============================================================================

// RxClassType defines class system types
type RxClassType string

const (
	ClassTypeATC      RxClassType = "ATC"       // WHO ATC Classification
	ClassTypeEPC      RxClassType = "EPC"       // FDA Established Pharmacologic Class
	ClassTypeMOA      RxClassType = "MOA"       // Mechanism of Action
	ClassTypePE       RxClassType = "PE"        // Physiologic Effect
	ClassTypeTHERAPY  RxClassType = "THERAPY"   // Therapeutic Intent
	ClassTypeDISEASE  RxClassType = "DISEASE"   // Disease/Condition
	ClassTypeCHEM     RxClassType = "CHEM"      // Chemical Structure
	ClassTypeVA       RxClassType = "VA"        // VA Drug Class
	ClassTypeMESHPA   RxClassType = "MESHPA"    // MeSH Pharmacological Action
)

// DrugClass represents a drug classification
type DrugClass struct {
	ClassID     string      `json:"classId"`
	ClassName   string      `json:"className"`
	ClassType   RxClassType `json:"classType"`
	Source      string      `json:"source,omitempty"`
	Description string      `json:"description,omitempty"`
}

// ClassMember represents a member of a drug class
type ClassMember struct {
	RxCUI     string `json:"rxcui"`
	DrugName  string `json:"drugName"`
	MinRxCUI  string `json:"minRxcui,omitempty"` // Minimal ingredient RxCUI
}

// MedicalCondition represents a disease/condition
type MedicalCondition struct {
	ConceptID   string `json:"conceptId"`
	ConceptName string `json:"conceptName"`
	Source      string `json:"source"` // MED-RT, SNOMED, etc.
	ClassType   string `json:"classType"`
}

// PhysiologicEffect represents a physiologic effect
type PhysiologicEffect struct {
	EffectID    string `json:"effectId"`
	EffectName  string `json:"effectName"`
	Source      string `json:"source"`
}

// MechanismOfAction represents a drug's mechanism of action
type MechanismOfAction struct {
	MoaID   string `json:"moaId"`
	MoaName string `json:"moaName"`
	Source  string `json:"source"`
}

// =============================================================================
// DAILYMED / OPENFDA INTERFACES
// =============================================================================

// DailyMedClient provides access to FDA DailyMed SPL documents
// API Docs: https://dailymed.nlm.nih.gov/dailymed/webservices-help/v2/
type DailyMedClient interface {
	DataSource

	// ─────────────────────────────────────────────────────────────────────────
	// SPL RETRIEVAL
	// ─────────────────────────────────────────────────────────────────────────

	// GetSPLBySetID retrieves the SPL document by Set ID
	GetSPLBySetID(ctx context.Context, setID string) (*SPLDocument, error)

	// GetSPLByNDC retrieves the SPL document for an NDC
	GetSPLByNDC(ctx context.Context, ndc string) (*SPLDocument, error)

	// GetLatestSPL retrieves the most recent SPL for a drug name
	GetLatestSPL(ctx context.Context, drugName string) (*SPLDocument, error)

	// ─────────────────────────────────────────────────────────────────────────
	// SECTION EXTRACTION
	// ─────────────────────────────────────────────────────────────────────────

	// GetSPLSection retrieves a specific section by LOINC code
	GetSPLSection(ctx context.Context, setID string, loincCode string) (*SPLSection, error)

	// GetDosageSection retrieves the DOSAGE AND ADMINISTRATION section
	GetDosageSection(ctx context.Context, setID string) (*SPLSection, error)

	// GetRenalSection retrieves renal impairment content
	GetRenalSection(ctx context.Context, setID string) (*SPLSection, error)

	// GetHepaticSection retrieves hepatic impairment content
	GetHepaticSection(ctx context.Context, setID string) (*SPLSection, error)

	// GetWarningsSection retrieves WARNINGS AND PRECAUTIONS
	GetWarningsSection(ctx context.Context, setID string) (*SPLSection, error)

	// GetContraindicationsSection retrieves CONTRAINDICATIONS
	GetContraindicationsSection(ctx context.Context, setID string) (*SPLSection, error)

	// ─────────────────────────────────────────────────────────────────────────
	// SEARCH
	// ─────────────────────────────────────────────────────────────────────────

	// SearchSPLs searches for SPL documents
	SearchSPLs(ctx context.Context, query string, limit int) ([]SPLMetadata, error)

	// ListSPLsForDrug lists all SPL versions for a drug
	ListSPLsForDrug(ctx context.Context, drugName string) ([]SPLMetadata, error)

	// ─────────────────────────────────────────────────────────────────────────
	// BATCH OPERATIONS
	// ─────────────────────────────────────────────────────────────────────────

	// BatchGetSections retrieves sections from multiple SPLs
	BatchGetSections(ctx context.Context, requests []SectionRequest) (map[string]*SPLSection, error)
}

// =============================================================================
// DAILYMED DATA MODELS
// =============================================================================

// SPLDocument represents a Structured Product Label
type SPLDocument struct {
	SetID         string                 `json:"setId"`
	Version       int                    `json:"version"`
	EffectiveTime time.Time              `json:"effectiveTime"`
	Title         string                 `json:"title"`
	DrugName      string                 `json:"drugName"`
	GenericName   string                 `json:"genericName,omitempty"`
	Labeler       string                 `json:"labeler"`
	NDCs          []string               `json:"ndcs,omitempty"`
	RxCUIs        []string               `json:"rxcuis,omitempty"`
	Sections      map[string]*SPLSection `json:"sections"`
	RawXML        []byte                 `json:"-"` // Original XML for reprocessing
}

// SPLSection represents a section within an SPL
type SPLSection struct {
	SetID       string       `json:"setId"`
	LoincCode   string       `json:"loincCode"`
	Title       string       `json:"title"`
	Content     string       `json:"content"`     // Plain text
	HTMLContent string       `json:"htmlContent"` // Formatted HTML
	Subsections []SPLSection `json:"subsections,omitempty"`
}

// SPLMetadata contains metadata about an SPL document
type SPLMetadata struct {
	SetID         string    `json:"setId"`
	Version       int       `json:"version"`
	Title         string    `json:"title"`
	EffectiveTime time.Time `json:"effectiveTime"`
	Published     time.Time `json:"published"`
}

// SectionRequest specifies a section to retrieve
type SectionRequest struct {
	SetID     string `json:"setId"`
	LoincCode string `json:"loincCode"`
}

// SPL Section LOINC codes
const (
	LoincDosageAdministration   = "34068-7"
	LoincUseSpecificPopulations = "43684-0"
	LoincRenalImpairment        = "42232-9"
	LoincHepaticImpairment      = "42229-5"
	LoincWarningsPrecautions    = "43685-7"
	LoincContraindications      = "34070-3"
	LoincBoxedWarning           = "34066-1"
	LoincDrugInteractions       = "34073-7"
	LoincPregnancy              = "42228-7"
	LoincNursing                = "34080-2"
	LoincPediatric              = "34081-0"
	LoincGeriatric              = "34082-8"
)

// =============================================================================
// LLM CLIENT INTERFACE
// =============================================================================

// LLMClient provides access to LLM services for extraction
type LLMClient interface {
	DataSource

	// ─────────────────────────────────────────────────────────────────────────
	// TEXT EXTRACTION
	// ─────────────────────────────────────────────────────────────────────────

	// Extract runs extraction using a prompt template
	Extract(ctx context.Context, prompt string, content string) (string, error)

	// ExtractWithSchema extracts structured data matching a schema
	ExtractWithSchema(ctx context.Context, prompt string, content string, schema interface{}) error

	// ExtractBatch processes multiple extraction requests
	ExtractBatch(ctx context.Context, requests []ExtractionRequest) ([]ExtractionResponse, error)

	// ─────────────────────────────────────────────────────────────────────────
	// UTILITIES
	// ─────────────────────────────────────────────────────────────────────────

	// GetTokenCount estimates token count for text
	GetTokenCount(ctx context.Context, text string) (int, error)

	// GetModel returns the model being used
	GetModel() string

	// GetMaxTokens returns the maximum tokens supported
	GetMaxTokens() int
}

// ExtractionRequest represents an extraction request
type ExtractionRequest struct {
	ID      string      `json:"id"`
	Prompt  string      `json:"prompt"`
	Content string      `json:"content"`
	Schema  interface{} `json:"schema,omitempty"`
}

// ExtractionResponse represents an extraction response
type ExtractionResponse struct {
	ID          string  `json:"id"`
	Result      string  `json:"result"`
	TokensUsed  int     `json:"tokensUsed"`
	Confidence  float64 `json:"confidence,omitempty"`
	Error       string  `json:"error,omitempty"`
}

// =============================================================================
// CACHE INTERFACE
// =============================================================================

// Cache provides caching capabilities for data sources
type Cache interface {
	// Get retrieves a value from cache
	Get(ctx context.Context, key string) ([]byte, error)

	// Set stores a value in cache with TTL
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error

	// Delete removes a value from cache
	Delete(ctx context.Context, key string) error

	// Exists checks if a key exists
	Exists(ctx context.Context, key string) (bool, error)

	// Clear removes all cached values
	Clear(ctx context.Context) error

	// Close closes the cache connection
	Close() error
}

// =============================================================================
// OHDSI ATHENA INTERFACE (for cross-validation)
// =============================================================================

// OHDSIClient provides access to OHDSI Athena vocabulary
type OHDSIClient interface {
	DataSource

	// ValidateRxNormConcept validates an RxNorm concept
	ValidateRxNormConcept(ctx context.Context, rxcui string) (*ConceptValidation, error)

	// GetContraindicationTable retrieves contraindications from OHDSI
	GetContraindicationTable(ctx context.Context, rxcui string) ([]OHDSIContraindication, error)

	// CrossValidateFact validates a fact against OHDSI data
	CrossValidateFact(ctx context.Context, factType string, rxcui string, content interface{}) (*ValidationResult, error)
}

// ConceptValidation contains validation results
type ConceptValidation struct {
	RxCUI       string `json:"rxcui"`
	Valid       bool   `json:"valid"`
	ConceptName string `json:"conceptName,omitempty"`
	DomainID    string `json:"domainId,omitempty"`
	Message     string `json:"message,omitempty"`
}

// OHDSIContraindication represents a contraindication from OHDSI
type OHDSIContraindication struct {
	DrugConceptID     int    `json:"drugConceptId"`
	DrugName          string `json:"drugName"`
	ConditionID       int    `json:"conditionId"`
	ConditionName     string `json:"conditionName"`
	Source            string `json:"source"`
}

// ValidationResult contains cross-validation results
type ValidationResult struct {
	Valid       bool     `json:"valid"`
	Confidence  float64  `json:"confidence"`
	Discrepancies []string `json:"discrepancies,omitempty"`
	Source      string   `json:"source"`
}

// =============================================================================
// AUTHORITY CLIENT INTERFACE (Phase 3b)
// =============================================================================
// All ground truth authority sources (CPIC, CredibleMeds, LiverTox, LactMed, etc.)
// implement this common interface for unified fact retrieval and governance.
//
// DESIGN PRINCIPLE: "When authoritative sources exist, we ROUTE — we do NOT EXTRACT"
// LLM extraction should be avoided for facts where curated, peer-reviewed authorities exist.

// AuthorityClient defines the common interface for all ground truth authority sources
type AuthorityClient interface {
	DataSource

	// ─────────────────────────────────────────────────────────────────────────
	// FACT RETRIEVAL
	// ─────────────────────────────────────────────────────────────────────────

	// GetFacts retrieves all facts for a drug by RxCUI
	GetFacts(ctx context.Context, rxcui string) ([]AuthorityFact, error)

	// GetFactsByName retrieves facts by drug name (generic or brand)
	GetFactsByName(ctx context.Context, drugName string) ([]AuthorityFact, error)

	// GetFactByType retrieves a specific fact type for a drug
	GetFactByType(ctx context.Context, rxcui string, factType FactType) (*AuthorityFact, error)

	// ─────────────────────────────────────────────────────────────────────────
	// SYNCHRONIZATION
	// ─────────────────────────────────────────────────────────────────────────

	// Sync synchronizes the local cache with the authority source
	Sync(ctx context.Context) (*SyncResult, error)

	// SyncDelta performs incremental sync since last update
	SyncDelta(ctx context.Context, since time.Time) (*SyncResult, error)

	// ─────────────────────────────────────────────────────────────────────────
	// METADATA
	// ─────────────────────────────────────────────────────────────────────────

	// Authority returns metadata about this authority source
	Authority() AuthorityMetadata

	// SupportedFactTypes returns the fact types this authority provides
	SupportedFactTypes() []FactType

	// LLMPolicy returns the LLM usage policy for this authority
	LLMPolicy() LLMPolicy
}

// =============================================================================
// AUTHORITY DATA MODELS
// =============================================================================

// AuthorityFact represents a clinical fact from an authoritative source
type AuthorityFact struct {
	// Identification
	ID              string    `json:"id"`
	AuthoritySource string    `json:"authority_source"` // "CPIC", "CredibleMeds", "LiverTox", etc.
	FactType        FactType  `json:"fact_type"`

	// Drug identification
	RxCUI           string    `json:"rxcui,omitempty"`
	DrugName        string    `json:"drug_name"`
	GenericName     string    `json:"generic_name,omitempty"`

	// Fact content (polymorphic based on FactType)
	Content         interface{} `json:"content"`

	// Clinical interpretation
	RiskLevel       string    `json:"risk_level,omitempty"`       // "HIGH", "MODERATE", "LOW", "NONE"
	ActionRequired  string    `json:"action_required,omitempty"`  // "AVOID", "MONITOR", "CAUTION", "NONE"
	Recommendations []string  `json:"recommendations,omitempty"`

	// Evidence and provenance
	EvidenceLevel   string    `json:"evidence_level,omitempty"`   // "A", "B", "C", "D" or similar
	References      []string  `json:"references,omitempty"`       // PMIDs or URLs

	// Extraction metadata
	ExtractionMethod string   `json:"extraction_method"` // Always "AUTHORITY_LOOKUP" for this interface
	Confidence       float64  `json:"confidence"`        // 1.0 for authority sources

	// Audit
	FetchedAt       time.Time `json:"fetched_at"`
	SourceVersion   string    `json:"source_version,omitempty"`
	SourceURL       string    `json:"source_url,omitempty"`
}

// FactType defines the types of clinical facts
type FactType string

const (
	// Safety facts
	FactTypeLactationSafety    FactType = "LACTATION_SAFETY"
	FactTypeHepatotoxicity     FactType = "HEPATOTOXICITY"
	FactTypeQTProlongation     FactType = "QT_PROLONGATION"
	FactTypeGeriatricPIM       FactType = "GERIATRIC_PIM"        // Potentially Inappropriate Medication

	// Dosing facts
	FactTypeRenalDosing        FactType = "RENAL_DOSING"
	FactTypeHepaticDosing      FactType = "HEPATIC_DOSING"
	FactTypePharmacogenomics   FactType = "PHARMACOGENOMICS"

	// Interaction facts
	FactTypeDrugInteraction    FactType = "DRUG_INTERACTION"
	FactTypeCYPInteraction     FactType = "CYP_INTERACTION"
	FactTypeTransporterInteraction FactType = "TRANSPORTER_INTERACTION"

	// Pharmacokinetic facts
	FactTypePKParameters       FactType = "PK_PARAMETERS"
	FactTypeProteinBinding     FactType = "PROTEIN_BINDING"
)

// AuthorityMetadata contains information about an authority source
type AuthorityMetadata struct {
	Name            string    `json:"name"`             // "CPIC", "CredibleMeds", etc.
	FullName        string    `json:"full_name"`        // "Clinical Pharmacogenetics Implementation Consortium"
	URL             string    `json:"url"`              // "https://cpicpgx.org"
	Description     string    `json:"description"`

	// Authority characteristics
	AuthorityLevel  AuthorityLevel `json:"authority_level"`
	DataFormat      string    `json:"data_format"`      // "REST_API", "XML_DOWNLOAD", "CSV"
	UpdateFrequency string    `json:"update_frequency"` // "DAILY", "WEEKLY", "MONTHLY", "QUARTERLY"

	// Coverage
	FactTypes       []FactType `json:"fact_types"`
	DrugCount       int       `json:"drug_count,omitempty"`

	// Version info
	Version         string    `json:"version,omitempty"`
	LastSync        time.Time `json:"last_sync,omitempty"`
}

// AuthorityLevel indicates the trustworthiness of the source
type AuthorityLevel string

const (
	// AuthorityDefinitive - Peer-reviewed, curated sources. LLM = NEVER
	// Examples: CPIC, CredibleMeds, LiverTox, LactMed
	AuthorityDefinitive AuthorityLevel = "DEFINITIVE"

	// AuthorityPrimary - Official sources, structured data. LLM = RARELY
	// Examples: FDA DailyMed (structured tables), DrugBank
	AuthorityPrimary AuthorityLevel = "PRIMARY"

	// AuthoritySecondary - Aggregated/derived sources. LLM = WITH_CONSENSUS
	// Examples: Parsed SPL prose
	AuthoritySecondary AuthorityLevel = "SECONDARY"
)

// LLMPolicy defines when LLM extraction is permitted for an authority
type LLMPolicy string

const (
	// LLMNever - LLM must never be used for this authority's fact types
	LLMNever LLMPolicy = "NEVER"

	// LLMGapFillOnly - LLM can fill gaps where authority has no data
	LLMGapFillOnly LLMPolicy = "GAP_FILL_ONLY"

	// LLMWithConsensus - LLM requires 2-of-3 consensus before accepting
	LLMWithConsensus LLMPolicy = "WITH_CONSENSUS"
)

// SyncResult contains the results of a synchronization operation
type SyncResult struct {
	Authority       string    `json:"authority"`
	StartTime       time.Time `json:"start_time"`
	EndTime         time.Time `json:"end_time"`

	// Counts
	TotalFacts      int       `json:"total_facts"`
	NewFacts        int       `json:"new_facts"`
	UpdatedFacts    int       `json:"updated_facts"`
	DeletedFacts    int       `json:"deleted_facts"`
	ErrorCount      int       `json:"error_count"`

	// Details
	Errors          []string  `json:"errors,omitempty"`
	SourceVersion   string    `json:"source_version,omitempty"`

	// Provenance (Gap 1 Fix: Dataset Provenance Locking)
	Provenance      *DataProvenance `json:"provenance,omitempty"`

	Success         bool      `json:"success"`
}

// =============================================================================
// PROVENANCE TRACKING (Phase 3b Production Hardening)
// =============================================================================
// These structures ensure regulatory audit defensibility by tracking:
// - WHICH version of data was loaded
// - WHEN it was downloaded
// - WHAT transformations were applied
// - HOW to verify integrity (checksums)

// DataProvenance tracks the origin and transformation of authority data
type DataProvenance struct {
	// Source identification
	SourceURL       string    `json:"source_url"`
	SourceVersion   string    `json:"source_version"`
	ReleaseDate     string    `json:"release_date,omitempty"`

	// Download metadata
	DownloadedAt    time.Time `json:"downloaded_at"`
	DownloadedBy    string    `json:"downloaded_by"`

	// File integrity
	Checksum        string    `json:"checksum"`          // SHA-256
	ChecksumAlgo    string    `json:"checksum_algo"`     // "SHA256"
	FileSize        int64     `json:"file_size_bytes"`
	RecordCount     int       `json:"record_count"`

	// Transformation tracking (Gap 2 Fix: Versioned Transformations)
	TransformScript string    `json:"transform_script,omitempty"` // e.g., "transform/lactmed_v2026_01.go"
	TransformVersion string   `json:"transform_version,omitempty"` // e.g., "1.0.0"
	TransformHash   string    `json:"transform_hash,omitempty"`    // SHA-256 of transform script

	// Audit chain
	PreviousChecksum string   `json:"previous_checksum,omitempty"` // For delta tracking
	ProvenanceChain  []string `json:"provenance_chain,omitempty"`  // History of changes
}

// AuthorityCapabilities represents the runtime status of authority sources
// (Gap 3 Fix: Capability Flags Exposure)
type AuthorityCapabilities struct {
	// Authority availability
	Authorities map[string]AuthorityStatus `json:"authorities"`

	// Fact type coverage
	FactTypeCoverage map[FactType]bool `json:"fact_type_coverage"`

	// Overall status
	CoverageLevel   string `json:"coverage_level"`   // "FULL", "PARTIAL", "MINIMAL"
	CoverageWarning string `json:"coverage_warning,omitempty"`

	// Timestamps
	LastUpdated     time.Time `json:"last_updated"`
	NextSyncDue     time.Time `json:"next_sync_due,omitempty"`
}

// AuthorityStatus represents the status of a single authority source
type AuthorityStatus struct {
	Name            string    `json:"name"`
	Available       bool      `json:"available"`
	Healthy         bool      `json:"healthy"`
	LastSync        time.Time `json:"last_sync,omitempty"`
	FactCount       int       `json:"fact_count"`
	SourceVersion   string    `json:"source_version,omitempty"`
	Checksum        string    `json:"checksum,omitempty"`
	LLMPolicy       LLMPolicy `json:"llm_policy"`
	AuthorityLevel  AuthorityLevel `json:"authority_level"`
}

// ManifestValidationResult contains the result of manifest validation
type ManifestValidationResult struct {
	Valid           bool      `json:"valid"`
	ValidatedAt     time.Time `json:"validated_at"`
	ValidatedBy     string    `json:"validated_by"`

	// Per-authority validation
	AuthorityResults map[string]AuthorityValidationResult `json:"authority_results"`

	// Errors
	Errors          []string  `json:"errors,omitempty"`
	Warnings        []string  `json:"warnings,omitempty"`
}

// AuthorityValidationResult contains validation results for a single authority
type AuthorityValidationResult struct {
	Authority       string `json:"authority"`
	Valid           bool   `json:"valid"`
	ChecksumMatch   bool   `json:"checksum_match"`
	VersionMatch    bool   `json:"version_match"`
	RecordCountMatch bool  `json:"record_count_match"`
	TransformValid  bool   `json:"transform_valid"`
	Errors          []string `json:"errors,omitempty"`
}
