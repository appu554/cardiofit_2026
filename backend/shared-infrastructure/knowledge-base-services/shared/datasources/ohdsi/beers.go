// Package ohdsi provides a client for OHDSI-based geriatric prescribing criteria.
// This includes AGS Beers Criteria and STOPP/START criteria for potentially
// inappropriate medications (PIMs) in older adults.
//
// Data Sources:
// - AGS Beers Criteria: American Geriatrics Society guidelines
// - STOPP/START: Screening Tool of Older Persons' Prescriptions
// - OHDSI Athena: Standardized vocabulary for concept mapping
//
// AUTHORITY LEVEL: DEFINITIVE (LLM = NEVER)
// Beers and STOPP criteria are peer-reviewed, evidence-based guidelines.
// LLM extraction is NEVER permitted for these facts.
package ohdsi

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// LOCAL TYPE ALIASES (avoid import cycles with datasources package)
// =============================================================================

// AuthorityLevel indicates the trustworthiness of the source
type AuthorityLevel string

const (
	AuthorityDefinitive AuthorityLevel = "DEFINITIVE"
	AuthorityPrimary    AuthorityLevel = "PRIMARY"
	AuthoritySecondary  AuthorityLevel = "SECONDARY"
)

// LLMPolicy defines when LLM extraction is permitted
type LLMPolicy string

const (
	LLMNever         LLMPolicy = "NEVER"
	LLMGapFillOnly   LLMPolicy = "GAP_FILL_ONLY"
	LLMWithConsensus LLMPolicy = "WITH_CONSENSUS"
)

// FactType defines the types of clinical facts
type FactType string

const (
	FactTypeGeriatricPIM FactType = "GERIATRIC_PIM" // Potentially Inappropriate Medication
)

// AuthorityFact represents a clinical fact from an authoritative source
type AuthorityFact struct {
	ID               string      `json:"id"`
	AuthoritySource  string      `json:"authority_source"`
	FactType         FactType    `json:"fact_type"`
	DrugRxCUI        string      `json:"rxcui,omitempty"`
	DrugName         string      `json:"drug_name"`
	GenericName      string      `json:"generic_name,omitempty"`
	FactValue        interface{} `json:"fact_value"`
	RiskLevel        string      `json:"risk_level,omitempty"`
	ActionRequired   string      `json:"action_required,omitempty"`
	Recommendations  []string    `json:"recommendations,omitempty"`
	EvidenceLevel    string      `json:"evidence_level,omitempty"`
	References       []string    `json:"references,omitempty"`
	ExtractionMethod string      `json:"extraction_method"`
	Confidence       float64     `json:"confidence"`
	FetchedAt        time.Time   `json:"fetched_at"`
	SourceVersion    string      `json:"source_version,omitempty"`
	SourceURL        string      `json:"source_url,omitempty"`
}

// AuthorityMetadata contains information about an authority source
type AuthorityMetadata struct {
	Name            string         `json:"name"`
	FullName        string         `json:"full_name"`
	URL             string         `json:"url"`
	Description     string         `json:"description"`
	Level           AuthorityLevel `json:"authority_level"`
	LLMPolicy       LLMPolicy      `json:"llm_policy"`
	DataFormat      string         `json:"data_format"`
	UpdateFrequency string         `json:"update_frequency"`
	FactTypes       []FactType     `json:"fact_types"`
	DrugCount       int            `json:"drug_count,omitempty"`
	Version         string         `json:"version,omitempty"`
	LastSync        time.Time      `json:"last_sync,omitempty"`
}

// SyncResult contains the results of a synchronization operation
type SyncResult struct {
	Authority      string    `json:"authority"`
	StartTime      time.Time `json:"start_time"`
	EndTime        time.Time `json:"end_time"`
	TotalRecords   int       `json:"total_records"`
	NewRecords     int       `json:"new_records"`
	UpdatedRecords int       `json:"updated_records"`
	DeletedRecords int       `json:"deleted_records"`
	ErrorCount     int       `json:"error_count"`
	Errors         []string  `json:"errors,omitempty"`
	SourceVersion  string    `json:"source_version,omitempty"`
	Success        bool      `json:"success"`
}

// =============================================================================
// BEERS/STOPP DATA MODELS
// =============================================================================

// CriteriaSource identifies the source of geriatric criteria
type CriteriaSource string

const (
	CriteriaBeers      CriteriaSource = "BEERS"       // AGS Beers Criteria
	CriteriaSTOPP      CriteriaSource = "STOPP"       // Screening Tool of Older Persons' Prescriptions
	CriteriaSTART      CriteriaSource = "START"       // Screening Tool to Alert to Right Treatment
	CriteriaPRISCUS    CriteriaSource = "PRISCUS"     // German PIM list
	CriteriaFORTA      CriteriaSource = "FORTA"       // Fit for the Aged
)

// RecommendationType indicates the type of recommendation
type RecommendationType string

const (
	RecommendAvoid           RecommendationType = "AVOID"             // Generally avoid
	RecommendAvoidInDisease  RecommendationType = "AVOID_IN_DISEASE"  // Avoid in specific conditions
	RecommendAvoidCombination RecommendationType = "AVOID_COMBINATION" // Avoid drug combinations
	RecommendUseCaution      RecommendationType = "USE_CAUTION"       // Use with caution
	RecommendDoseAdjust      RecommendationType = "DOSE_ADJUST"       // Dose adjustment needed
	RecommendConsider        RecommendationType = "CONSIDER"          // Consider for appropriate patients (START)
)

// EvidenceQuality indicates the quality of evidence
type EvidenceQuality string

const (
	EvidenceHigh     EvidenceQuality = "HIGH"      // RCTs, systematic reviews
	EvidenceModerate EvidenceQuality = "MODERATE"  // Observational studies
	EvidenceLow      EvidenceQuality = "LOW"       // Case reports, expert opinion
)

// RecommendationStrength indicates strength of recommendation
type RecommendationStrength string

const (
	StrengthStrong RecommendationStrength = "STRONG"
	StrengthWeak   RecommendationStrength = "WEAK"
)

// PIMEntry represents a Potentially Inappropriate Medication entry
type PIMEntry struct {
	// Identification
	ID             string         `json:"id"`
	Source         CriteriaSource `json:"source"`          // BEERS, STOPP, etc.
	Category       string         `json:"category"`        // e.g., "Anticholinergics", "Benzodiazepines"
	Version        string         `json:"version"`         // e.g., "2023"

	// Drug identification
	DrugName       string   `json:"drug_name"`
	DrugClass      string   `json:"drug_class,omitempty"`    // Therapeutic class
	RxCUIs         []string `json:"rxcuis,omitempty"`        // Associated RxCUI codes
	ATCCodes       []string `json:"atc_codes,omitempty"`     // ATC classification codes
	ConceptIDs     []int64  `json:"concept_ids,omitempty"`   // OHDSI concept IDs

	// Recommendation details
	Recommendation     RecommendationType     `json:"recommendation"`
	Rationale          string                 `json:"rationale"`
	ClinicalConditions []string               `json:"clinical_conditions,omitempty"` // Conditions where PIM applies
	AgeThreshold       int                    `json:"age_threshold,omitempty"`       // Usually 65
	Exceptions         []string               `json:"exceptions,omitempty"`          // When it may be appropriate

	// Evidence
	EvidenceQuality    EvidenceQuality        `json:"evidence_quality"`
	Strength           RecommendationStrength `json:"strength"`
	References         []string               `json:"references,omitempty"` // PMIDs

	// Clinical guidance
	AlternativeDrugs   []string `json:"alternative_drugs,omitempty"`
	MonitoringRequired []string `json:"monitoring_required,omitempty"`
	MaxDose            string   `json:"max_dose,omitempty"`
	MaxDuration        string   `json:"max_duration,omitempty"`
}

// DrugDiseaseInteraction represents a drug-disease interaction from Beers Table 3
type DrugDiseaseInteraction struct {
	ID              string         `json:"id"`
	Source          CriteriaSource `json:"source"`
	Disease         string         `json:"disease"`        // e.g., "Heart Failure", "Dementia"
	DiseaseICD10    []string       `json:"disease_icd10,omitempty"`
	DrugOrClass     string         `json:"drug_or_class"`
	RxCUIs          []string       `json:"rxcuis,omitempty"`
	Recommendation  string         `json:"recommendation"`
	Rationale       string         `json:"rationale"`
	EvidenceQuality EvidenceQuality `json:"evidence_quality"`
	Strength        RecommendationStrength `json:"strength"`
}

// DrugDrugInteraction represents a drug-drug interaction from Beers Table 4
type DrugDrugInteraction struct {
	ID              string         `json:"id"`
	Source          CriteriaSource `json:"source"`
	Drug1           string         `json:"drug1"`
	Drug1RxCUIs     []string       `json:"drug1_rxcuis,omitempty"`
	Drug2           string         `json:"drug2"`
	Drug2RxCUIs     []string       `json:"drug2_rxcuis,omitempty"`
	InteractionType string         `json:"interaction_type"` // e.g., "ACEi + K+ sparing diuretic"
	ClinicalEffect  string         `json:"clinical_effect"`
	Recommendation  string         `json:"recommendation"`
	EvidenceQuality EvidenceQuality `json:"evidence_quality"`
}

// =============================================================================
// BEERS CATEGORIES (2023 AGS Beers Criteria)
// =============================================================================

// BeersCategory represents a category from the Beers Criteria
type BeersCategory string

const (
	// Table 1: PIMs independent of diagnosis/condition
	BeersAnticholinergics          BeersCategory = "ANTICHOLINERGICS"
	BeersAntithrombotics           BeersCategory = "ANTITHROMBOTICS"
	BeersAntiinfectives            BeersCategory = "ANTI_INFECTIVES"
	BeersCardiovascular            BeersCategory = "CARDIOVASCULAR"
	BeersCNS                       BeersCategory = "CNS"
	BeersEndocrine                 BeersCategory = "ENDOCRINE"
	BeersGastrointestinal          BeersCategory = "GASTROINTESTINAL"
	BeersPain                      BeersCategory = "PAIN"
	BeersGenitourinary             BeersCategory = "GENITOURINARY"

	// Table 2: PIMs due to drug-disease/syndrome interactions
	BeersDiseaseDrug               BeersCategory = "DISEASE_DRUG"

	// Table 3: Drugs to use with caution
	BeersUseCaution                BeersCategory = "USE_CAUTION"

	// Table 4: Drug-drug interactions
	BeersDrugDrug                  BeersCategory = "DRUG_DRUG"

	// Table 5: Drugs to avoid or dose reduce with kidney function
	BeersRenalFunction             BeersCategory = "RENAL_FUNCTION"
)

// =============================================================================
// OHDSI CLIENT
// =============================================================================

// Config holds configuration for the OHDSI Beers/STOPP client
type Config struct {
	// AthenaURL is the OHDSI Athena vocabulary API URL
	AthenaURL string

	// DataPath is the path to local Beers/STOPP data files
	DataPath string

	// Database connection for persistent storage
	DB *sql.DB

	// HTTPClient for API requests
	HTTPClient *http.Client

	// BeersVersion specifies which Beers version to use
	BeersVersion string // "2023", "2019", etc.

	// IncludeSTOPP whether to include STOPP/START criteria
	IncludeSTOPP bool

	// CacheTTL for in-memory caching
	CacheTTL time.Duration
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		AthenaURL:    "https://athena.ohdsi.org/api/v1",
		DataPath:     "/data/ohdsi/beers_stopp",
		BeersVersion: "2023",
		IncludeSTOPP: true,
		CacheTTL:     24 * time.Hour,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Client provides access to OHDSI Beers/STOPP criteria
type Client struct {
	config Config

	// PIM entries indexed multiple ways
	pimEntries      []*PIMEntry
	byRxCUI         map[string][]*PIMEntry // RxCUI -> PIM entries
	byDrugName      map[string][]*PIMEntry // Lowercase name -> PIM entries
	byCategory      map[BeersCategory][]*PIMEntry
	byATCCode       map[string][]*PIMEntry

	// Drug-disease interactions
	diseaseInteractions []*DrugDiseaseInteraction

	// Drug-drug interactions
	drugInteractions []*DrugDrugInteraction

	mu       sync.RWMutex
	loaded   bool
	loadedAt time.Time
	version  string
}

// NewClient creates a new OHDSI Beers/STOPP client
func NewClient(config Config) *Client {
	return &Client{
		config:              config,
		pimEntries:         make([]*PIMEntry, 0),
		byRxCUI:            make(map[string][]*PIMEntry),
		byDrugName:         make(map[string][]*PIMEntry),
		byCategory:         make(map[BeersCategory][]*PIMEntry),
		byATCCode:          make(map[string][]*PIMEntry),
		diseaseInteractions: make([]*DrugDiseaseInteraction, 0),
		drugInteractions:    make([]*DrugDrugInteraction, 0),
	}
}

// =============================================================================
// DATASOURCE INTERFACE IMPLEMENTATION
// =============================================================================

// Name returns the data source name
func (c *Client) Name() string {
	return "OHDSI_Beers_STOPP"
}

// HealthCheck verifies the data source is available
func (c *Client) HealthCheck(ctx context.Context) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.loaded {
		return fmt.Errorf("Beers/STOPP criteria not loaded")
	}

	if len(c.pimEntries) == 0 {
		return fmt.Errorf("no PIM entries loaded")
	}

	return nil
}

// Close releases resources
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.pimEntries = make([]*PIMEntry, 0)
	c.byRxCUI = make(map[string][]*PIMEntry)
	c.byDrugName = make(map[string][]*PIMEntry)
	c.byCategory = make(map[BeersCategory][]*PIMEntry)
	c.byATCCode = make(map[string][]*PIMEntry)
	c.loaded = false

	return nil
}

// =============================================================================
// AUTHORITY CLIENT INTERFACE IMPLEMENTATION
// =============================================================================

// GetFacts retrieves all geriatric PIM facts for a drug by RxCUI
func (c *Client) GetFacts(ctx context.Context, rxcui string) ([]AuthorityFact, error) {
	c.mu.RLock()
	entries, exists := c.byRxCUI[rxcui]
	c.mu.RUnlock()

	if !exists || len(entries) == 0 {
		return nil, nil // Not found - not an error (drug may be safe)
	}

	var facts []AuthorityFact
	for _, entry := range entries {
		facts = append(facts, c.pimToFact(entry))
	}

	return facts, nil
}

// GetFactsByName retrieves PIM facts by drug name
func (c *Client) GetFactsByName(ctx context.Context, drugName string) ([]AuthorityFact, error) {
	c.mu.RLock()
	entries, exists := c.byDrugName[strings.ToLower(drugName)]
	c.mu.RUnlock()

	if !exists || len(entries) == 0 {
		return nil, nil
	}

	var facts []AuthorityFact
	for _, entry := range entries {
		facts = append(facts, c.pimToFact(entry))
	}

	return facts, nil
}

// GetFactByType retrieves a specific fact type for a drug
func (c *Client) GetFactByType(ctx context.Context, rxcui string, factType FactType) (*AuthorityFact, error) {
	if factType != FactTypeGeriatricPIM {
		return nil, nil // Only support GERIATRIC_PIM
	}

	facts, err := c.GetFacts(ctx, rxcui)
	if err != nil || len(facts) == 0 {
		return nil, err
	}

	return &facts[0], nil
}

// Sync synchronizes the local cache with criteria data
func (c *Client) Sync(ctx context.Context) (*SyncResult, error) {
	result := &SyncResult{
		Authority: "OHDSI_Beers_STOPP",
		StartTime: time.Now(),
	}

	// Load Beers criteria
	if err := c.loadBeersCriteria(ctx); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Beers: %v", err))
		result.ErrorCount++
	}

	// Load STOPP/START if configured
	if c.config.IncludeSTOPP {
		if err := c.loadSTOPPCriteria(ctx); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("STOPP: %v", err))
			result.ErrorCount++
		}
	}

	result.EndTime = time.Now()
	result.TotalRecords = len(c.pimEntries)
	result.Success = result.ErrorCount == 0
	result.SourceVersion = c.version

	return result, nil
}

// SyncDelta performs incremental sync (criteria typically updated annually)
func (c *Client) SyncDelta(ctx context.Context, since time.Time) (*SyncResult, error) {
	// Criteria are updated infrequently - perform full sync
	return c.Sync(ctx)
}

// Authority returns metadata about this authority source
func (c *Client) Authority() AuthorityMetadata {
	return AuthorityMetadata{
		Name:        "OHDSI_Beers_STOPP",
		FullName:    "OHDSI Athena - AGS Beers Criteria & STOPP/START",
		URL:         "https://www.americangeriatrics.org/programs/clinical-practice/beers-criteria",
		Description: "Peer-reviewed criteria for potentially inappropriate medications in older adults (65+)",
		Level:       AuthorityDefinitive,
		LLMPolicy:   LLMNever,
		DataFormat:  "CONCEPT_SET_API",
		UpdateFrequency: "TRIENNIAL", // Beers updated every ~3 years
		FactTypes: []FactType{
			FactTypeGeriatricPIM,
		},
		DrugCount: len(c.pimEntries),
		Version:   c.version,
		LastSync:  c.loadedAt,
	}
}

// SupportedFactTypes returns fact types provided by this source
func (c *Client) SupportedFactTypes() []FactType {
	return []FactType{FactTypeGeriatricPIM}
}

// LLMPolicy returns the LLM usage policy
func (c *Client) LLMPolicy() LLMPolicy {
	return LLMNever // Definitive source - LLM never permitted
}

// =============================================================================
// DATA LOADING
// =============================================================================

// loadBeersCriteria loads AGS Beers Criteria
func (c *Client) loadBeersCriteria(ctx context.Context) error {
	// Try loading from JSON file first
	filePath := fmt.Sprintf("%s/beers_%s.json", c.config.DataPath, c.config.BeersVersion)

	file, err := os.Open(filePath)
	if err != nil {
		// If file doesn't exist, load embedded defaults
		return c.loadEmbeddedBeers(ctx)
	}
	defer file.Close()

	return c.parseBeersJSON(ctx, file)
}

// loadSTOPPCriteria loads STOPP/START criteria
func (c *Client) loadSTOPPCriteria(ctx context.Context) error {
	filePath := fmt.Sprintf("%s/stopp_start_v2.json", c.config.DataPath)

	file, err := os.Open(filePath)
	if err != nil {
		// If file doesn't exist, load embedded defaults
		return c.loadEmbeddedSTOPP(ctx)
	}
	defer file.Close()

	return c.parseSTOPPJSON(ctx, file)
}

// parseBeersJSON parses Beers criteria from JSON
func (c *Client) parseBeersJSON(ctx context.Context, reader io.Reader) error {
	var entries []*PIMEntry
	if err := json.NewDecoder(reader).Decode(&entries); err != nil {
		return fmt.Errorf("parsing Beers JSON: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for _, entry := range entries {
		entry.Source = CriteriaBeers
		c.indexEntry(entry)
	}

	c.loaded = true
	c.loadedAt = time.Now()
	c.version = c.config.BeersVersion

	return nil
}

// parseSTOPPJSON parses STOPP/START criteria from JSON
func (c *Client) parseSTOPPJSON(ctx context.Context, reader io.Reader) error {
	var entries []*PIMEntry
	if err := json.NewDecoder(reader).Decode(&entries); err != nil {
		return fmt.Errorf("parsing STOPP JSON: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for _, entry := range entries {
		c.indexEntry(entry)
	}

	return nil
}

// loadEmbeddedBeers loads embedded Beers criteria (common high-risk PIMs)
func (c *Client) loadEmbeddedBeers(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Load core Beers 2023 criteria (most common/important PIMs)
	embeddedPIMs := c.getEmbeddedBeers2023()

	for _, entry := range embeddedPIMs {
		c.indexEntry(entry)
	}

	c.loaded = true
	c.loadedAt = time.Now()
	c.version = "2023-embedded"

	return nil
}

// loadEmbeddedSTOPP loads embedded STOPP criteria
func (c *Client) loadEmbeddedSTOPP(ctx context.Context) error {
	// STOPP criteria are complementary - can be added similarly
	return nil
}

// indexEntry adds a PIM entry to all indices
func (c *Client) indexEntry(entry *PIMEntry) {
	c.pimEntries = append(c.pimEntries, entry)

	// Index by RxCUI
	for _, rxcui := range entry.RxCUIs {
		c.byRxCUI[rxcui] = append(c.byRxCUI[rxcui], entry)
	}

	// Index by drug name (lowercase)
	nameLower := strings.ToLower(entry.DrugName)
	c.byDrugName[nameLower] = append(c.byDrugName[nameLower], entry)

	// Index by category
	category := c.categoryFromString(entry.Category)
	c.byCategory[category] = append(c.byCategory[category], entry)

	// Index by ATC code
	for _, atc := range entry.ATCCodes {
		c.byATCCode[atc] = append(c.byATCCode[atc], entry)
	}
}

// categoryFromString converts category string to BeersCategory
func (c *Client) categoryFromString(cat string) BeersCategory {
	switch strings.ToUpper(cat) {
	case "ANTICHOLINERGICS":
		return BeersAnticholinergics
	case "CNS", "CENTRAL NERVOUS SYSTEM":
		return BeersCNS
	case "CARDIOVASCULAR":
		return BeersCardiovascular
	case "PAIN":
		return BeersPain
	case "ENDOCRINE":
		return BeersEndocrine
	case "GASTROINTESTINAL", "GI":
		return BeersGastrointestinal
	default:
		return BeersCNS // Default to CNS for unrecognized
	}
}

// =============================================================================
// EMBEDDED BEERS 2023 CRITERIA (Core High-Risk PIMs)
// =============================================================================

// getEmbeddedBeers2023 returns embedded high-priority Beers criteria
func (c *Client) getEmbeddedBeers2023() []*PIMEntry {
	return []*PIMEntry{
		// CNS - Benzodiazepines
		{
			ID:             "beers-2023-bzd-001",
			Source:         CriteriaBeers,
			Category:       "CNS",
			Version:        "2023",
			DrugName:       "Benzodiazepines",
			DrugClass:      "Benzodiazepine",
			RxCUIs:         []string{"596", "2598", "4501", "17767"}, // alprazolam, diazepam, lorazepam, etc.
			ATCCodes:       []string{"N05BA", "N05CD"},
			Recommendation: RecommendAvoid,
			Rationale:      "Older adults have increased sensitivity to benzodiazepines and decreased metabolism. All benzodiazepines increase risk of cognitive impairment, delirium, falls, fractures, and motor vehicle crashes in older adults.",
			AgeThreshold:   65,
			Exceptions:     []string{"Seizure disorders", "Severe generalized anxiety disorder", "Alcohol withdrawal", "Benzodiazepine withdrawal", "Periprocedural anesthesia"},
			EvidenceQuality: EvidenceHigh,
			Strength:       StrengthStrong,
			AlternativeDrugs: []string{"SSRIs", "SNRIs", "Buspirone", "CBT for insomnia"},
			References:     []string{"PMID:30693238"},
		},

		// CNS - Non-benzodiazepine hypnotics ("Z-drugs")
		{
			ID:             "beers-2023-zdrug-001",
			Source:         CriteriaBeers,
			Category:       "CNS",
			Version:        "2023",
			DrugName:       "Non-benzodiazepine hypnotics",
			DrugClass:      "Z-Drug",
			RxCUIs:         []string{"39993", "103971"}, // zolpidem, eszopiclone
			ATCCodes:       []string{"N05CF"},
			Recommendation: RecommendAvoid,
			Rationale:      "Nonbenzodiazepine, benzodiazepine receptor agonist hypnotics have adverse events similar to those of benzodiazepines in older adults. Minimal improvement in sleep latency and duration.",
			AgeThreshold:   65,
			EvidenceQuality: EvidenceModerate,
			Strength:       StrengthStrong,
			AlternativeDrugs: []string{"CBT-I", "Melatonin", "Sleep hygiene"},
			References:     []string{"PMID:30693238"},
		},

		// Anticholinergics - First-generation antihistamines
		{
			ID:             "beers-2023-antihist-001",
			Source:         CriteriaBeers,
			Category:       "ANTICHOLINERGICS",
			Version:        "2023",
			DrugName:       "First-generation antihistamines",
			DrugClass:      "First-generation antihistamine",
			RxCUIs:         []string{"3498", "1117"}, // diphenhydramine, chlorpheniramine
			ATCCodes:       []string{"R06AA", "R06AB"},
			Recommendation: RecommendAvoid,
			Rationale:      "Highly anticholinergic; clearance reduced with advanced age. Risk of confusion, dry mouth, constipation, and other anticholinergic effects and toxicity. Use of diphenhydramine in special situations such as acute severe allergic reaction may be appropriate.",
			AgeThreshold:   65,
			EvidenceQuality: EvidenceHigh,
			Strength:       StrengthStrong,
			AlternativeDrugs: []string{"Loratadine", "Cetirizine", "Fexofenadine"},
			References:     []string{"PMID:30693238"},
		},

		// Pain - NSAIDs
		{
			ID:             "beers-2023-nsaid-001",
			Source:         CriteriaBeers,
			Category:       "PAIN",
			Version:        "2023",
			DrugName:       "NSAIDs (non-selective)",
			DrugClass:      "NSAID",
			RxCUIs:         []string{"5640", "41493", "6714"}, // ibuprofen, naproxen, ketorolac
			ATCCodes:       []string{"M01A"},
			Recommendation: RecommendAvoid,
			Rationale:      "Increased risk of GI bleeding/peptic ulcer disease in high-risk groups, including those aged >75 or taking oral or parenteral corticosteroids, anticoagulants, or antiplatelet agents. Use of proton pump inhibitor or misoprostol reduces but does not eliminate risk. Upper GI ulcers, gross bleeding, or perforation caused by NSAIDs occur in approximately 1% of patients treated for 3-6 months and in about 2-4% of patients treated for 1 year.",
			AgeThreshold:   65,
			Exceptions:     []string{"Short-term use (<1 week) with gastroprotection"},
			EvidenceQuality: EvidenceHigh,
			Strength:       StrengthStrong,
			AlternativeDrugs: []string{"Acetaminophen", "Topical NSAIDs", "Duloxetine"},
			MaxDuration:    "7 days",
			References:     []string{"PMID:30693238"},
		},

		// Cardiovascular - Digoxin
		{
			ID:             "beers-2023-digoxin-001",
			Source:         CriteriaBeers,
			Category:       "CARDIOVASCULAR",
			Version:        "2023",
			DrugName:       "Digoxin",
			DrugClass:      "Cardiac glycoside",
			RxCUIs:         []string{"3407"},
			ATCCodes:       []string{"C01AA05"},
			Recommendation: RecommendUseCaution,
			Rationale:      "Use in atrial fibrillation: should not be used as first-line agent, as there are more effective alternatives. Decreased renal clearance of digoxin may lead to increased risk of toxic effects; further dose reduction may be necessary in those with Stage 4 or 5 CKD.",
			AgeThreshold:   65,
			EvidenceQuality: EvidenceModerate,
			Strength:       StrengthStrong,
			MaxDose:        "0.125 mg/day",
			MonitoringRequired: []string{"Serum digoxin level", "Renal function", "Potassium"},
			References:     []string{"PMID:30693238"},
		},

		// Endocrine - Sulfonylureas (long-acting)
		{
			ID:             "beers-2023-sulfa-001",
			Source:         CriteriaBeers,
			Category:       "ENDOCRINE",
			Version:        "2023",
			DrugName:       "Sulfonylureas (long-acting)",
			DrugClass:      "Sulfonylurea",
			RxCUIs:         []string{"4815", "4821"}, // glyburide, chlorpropamide
			ATCCodes:       []string{"A10BB01", "A10BB02"},
			Recommendation: RecommendAvoid,
			Rationale:      "Chlorpropamide: prolonged half-life in older adults; can cause prolonged hypoglycemia; causes SIADH. Glyburide: higher risk of severe prolonged hypoglycemia in older adults.",
			AgeThreshold:   65,
			EvidenceQuality: EvidenceHigh,
			Strength:       StrengthStrong,
			AlternativeDrugs: []string{"Metformin", "DPP-4 inhibitors", "GLP-1 agonists", "SGLT2 inhibitors"},
			References:     []string{"PMID:30693238"},
		},

		// Anticholinergics - Antispasmodics
		{
			ID:             "beers-2023-antispas-001",
			Source:         CriteriaBeers,
			Category:       "ANTICHOLINERGICS",
			Version:        "2023",
			DrugName:       "Antispasmodics",
			DrugClass:      "Antispasmodic",
			RxCUIs:         []string{"3356", "5476", "6809"}, // dicyclomine, hyoscyamine, scopolamine
			ATCCodes:       []string{"A03B"},
			Recommendation: RecommendAvoid,
			Rationale:      "Highly anticholinergic, uncertain effectiveness.",
			AgeThreshold:   65,
			EvidenceQuality: EvidenceModerate,
			Strength:       StrengthStrong,
			References:     []string{"PMID:30693238"},
		},

		// CNS - Antipsychotics
		{
			ID:             "beers-2023-antipsy-001",
			Source:         CriteriaBeers,
			Category:       "CNS",
			Version:        "2023",
			DrugName:       "Antipsychotics (first and second generation)",
			DrugClass:      "Antipsychotic",
			RxCUIs:         []string{"5093", "89013", "61381"}, // haloperidol, risperidone, olanzapine
			ATCCodes:       []string{"N05A"},
			Recommendation: RecommendAvoid,
			Rationale:      "Increased risk of cerebrovascular accident (stroke) and greater rate of cognitive decline and mortality in persons with dementia. Avoid antipsychotics for behavioral problems associated with dementia or delirium unless nonpharmacological options have failed or are not possible and the older adult is threatening substantial harm to self or others.",
			AgeThreshold:   65,
			ClinicalConditions: []string{"Dementia"},
			EvidenceQuality: EvidenceHigh,
			Strength:       StrengthStrong,
			References:     []string{"PMID:30693238"},
		},

		// GI - Proton pump inhibitors (long-term)
		{
			ID:             "beers-2023-ppi-001",
			Source:         CriteriaBeers,
			Category:       "GASTROINTESTINAL",
			Version:        "2023",
			DrugName:       "Proton pump inhibitors",
			DrugClass:      "PPI",
			RxCUIs:         []string{"7646", "283742", "40790"}, // omeprazole, pantoprazole, esomeprazole
			ATCCodes:       []string{"A02BC"},
			Recommendation: RecommendUseCaution,
			Rationale:      "Risk of Clostridioides difficile infection, bone loss, and fractures. Avoid scheduled use for >8 weeks unless for high-risk patients (e.g., oral corticosteroids or chronic NSAID use), erosive esophagitis, Barrett's esophagus, pathological hypersecretory condition, or demonstrated need for maintenance treatment.",
			AgeThreshold:   65,
			EvidenceQuality: EvidenceHigh,
			Strength:       StrengthStrong,
			MaxDuration:    "8 weeks",
			Exceptions:     []string{"Barrett's esophagus", "Severe esophagitis", "H. pylori treatment", "High-risk NSAID users"},
			References:     []string{"PMID:30693238"},
		},

		// Genitourinary - Alpha-blockers
		{
			ID:             "beers-2023-alpha-001",
			Source:         CriteriaBeers,
			Category:       "GENITOURINARY",
			Version:        "2023",
			DrugName:       "Alpha-blockers for hypertension",
			DrugClass:      "Alpha-blocker",
			RxCUIs:         []string{"49276", "8629"}, // doxazosin, prazosin
			ATCCodes:       []string{"C02CA"},
			Recommendation: RecommendAvoid,
			Rationale:      "High risk of orthostatic hypotension and associated harms, especially in older adults. Not recommended as routine treatment for hypertension; alternative agents have superior risk/benefit profile.",
			AgeThreshold:   65,
			EvidenceQuality: EvidenceModerate,
			Strength:       StrengthStrong,
			AlternativeDrugs: []string{"ACE inhibitors", "ARBs", "CCBs", "Thiazide diuretics"},
			References:     []string{"PMID:30693238"},
		},
	}
}

// =============================================================================
// FACT CONVERSION
// =============================================================================

// pimToFact converts a PIMEntry to AuthorityFact
func (c *Client) pimToFact(entry *PIMEntry) AuthorityFact {
	riskLevel := "HIGH"
	actionRequired := "AVOID"

	switch entry.Recommendation {
	case RecommendUseCaution:
		riskLevel = "MODERATE"
		actionRequired = "CAUTION"
	case RecommendDoseAdjust:
		riskLevel = "MODERATE"
		actionRequired = "DOSE_ADJUST"
	case RecommendConsider:
		riskLevel = "LOW"
		actionRequired = "CONSIDER"
	}

	var rxcui string
	if len(entry.RxCUIs) > 0 {
		rxcui = entry.RxCUIs[0]
	}

	return AuthorityFact{
		ID:              uuid.New().String(),
		AuthoritySource: string(entry.Source),
		FactType:        FactTypeGeriatricPIM,
		DrugRxCUI:       rxcui,
		DrugName:        entry.DrugName,
		FactValue: map[string]interface{}{
			"category":           entry.Category,
			"drug_class":         entry.DrugClass,
			"recommendation":     entry.Recommendation,
			"rationale":          entry.Rationale,
			"age_threshold":      entry.AgeThreshold,
			"conditions":         entry.ClinicalConditions,
			"exceptions":         entry.Exceptions,
			"alternative_drugs":  entry.AlternativeDrugs,
			"max_dose":           entry.MaxDose,
			"max_duration":       entry.MaxDuration,
			"monitoring":         entry.MonitoringRequired,
		},
		RiskLevel:        riskLevel,
		ActionRequired:   actionRequired,
		Recommendations:  entry.AlternativeDrugs,
		EvidenceLevel:    string(entry.EvidenceQuality),
		References:       entry.References,
		ExtractionMethod: "AUTHORITY_LOOKUP",
		Confidence:       1.0,
		FetchedAt:        time.Now(),
		SourceVersion:    entry.Version,
		SourceURL:        "https://www.americangeriatrics.org/programs/clinical-practice/beers-criteria",
	}
}

// =============================================================================
// QUERY METHODS
// =============================================================================

// IsPIM checks if a drug is a potentially inappropriate medication
func (c *Client) IsPIM(ctx context.Context, rxcui string) (bool, error) {
	c.mu.RLock()
	entries, exists := c.byRxCUI[rxcui]
	c.mu.RUnlock()

	return exists && len(entries) > 0, nil
}

// GetPIMDetails retrieves detailed PIM information
func (c *Client) GetPIMDetails(ctx context.Context, rxcui string) ([]*PIMEntry, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entries, exists := c.byRxCUI[rxcui]
	if !exists {
		return nil, nil
	}

	// Return a copy to prevent modification
	result := make([]*PIMEntry, len(entries))
	copy(result, entries)

	return result, nil
}

// GetPIMsByCategory retrieves all PIMs in a category
func (c *Client) GetPIMsByCategory(ctx context.Context, category BeersCategory) ([]*PIMEntry, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entries, exists := c.byCategory[category]
	if !exists {
		return nil, nil
	}

	result := make([]*PIMEntry, len(entries))
	copy(result, entries)

	return result, nil
}

// GetAlternatives retrieves safer alternative drugs for a PIM
func (c *Client) GetAlternatives(ctx context.Context, rxcui string) ([]string, error) {
	entries, err := c.GetPIMDetails(ctx, rxcui)
	if err != nil || len(entries) == 0 {
		return nil, err
	}

	// Collect all alternatives from all matching entries
	alternativesMap := make(map[string]bool)
	for _, entry := range entries {
		for _, alt := range entry.AlternativeDrugs {
			alternativesMap[alt] = true
		}
	}

	alternatives := make([]string, 0, len(alternativesMap))
	for alt := range alternativesMap {
		alternatives = append(alternatives, alt)
	}

	return alternatives, nil
}

// CheckDrugDiseaseInteraction checks for drug-disease interactions
func (c *Client) CheckDrugDiseaseInteraction(ctx context.Context, rxcui string, diseaseICD10 string) (*DrugDiseaseInteraction, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, interaction := range c.diseaseInteractions {
		// Check if drug matches
		drugMatch := false
		for _, rxc := range interaction.RxCUIs {
			if rxc == rxcui {
				drugMatch = true
				break
			}
		}

		if !drugMatch {
			continue
		}

		// Check if disease matches
		for _, icd := range interaction.DiseaseICD10 {
			if strings.HasPrefix(diseaseICD10, icd) {
				return interaction, nil
			}
		}
	}

	return nil, nil
}

// GetAllPIMEntries returns all loaded PIM entries
func (c *Client) GetAllPIMEntries(ctx context.Context) []*PIMEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]*PIMEntry, len(c.pimEntries))
	copy(result, c.pimEntries)

	return result
}

// LoadFromJSON loads criteria from a JSON file
func (c *Client) LoadFromJSON(ctx context.Context, reader io.Reader) error {
	var entries []*PIMEntry
	if err := json.NewDecoder(reader).Decode(&entries); err != nil {
		return fmt.Errorf("parsing JSON: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Clear and reload
	c.pimEntries = make([]*PIMEntry, 0)
	c.byRxCUI = make(map[string][]*PIMEntry)
	c.byDrugName = make(map[string][]*PIMEntry)
	c.byCategory = make(map[BeersCategory][]*PIMEntry)
	c.byATCCode = make(map[string][]*PIMEntry)

	for _, entry := range entries {
		c.indexEntry(entry)
	}

	c.loaded = true
	c.loadedAt = time.Now()

	return nil
}
