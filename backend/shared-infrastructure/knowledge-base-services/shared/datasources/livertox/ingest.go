// Package livertox provides ingestion for the LiverTox database.
// LiverTox is NIH's comprehensive database of drug-induced liver injury (DILI).
//
// Phase 3b.3: LiverTox XML Ingestion
// Authority Level: DEFINITIVE (LLM = ❌ NEVER)
//
// Data Source: https://www.ncbi.nlm.nih.gov/books/NBK547852/
// LiverTox is freely available from NLM/NCBI
package livertox

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// =============================================================================
// LIVERTOX DATABASE
// =============================================================================

// LiverToxDB represents the full LiverTox database
type LiverToxDB struct {
	Drugs        map[string]*DrugHepatotoxicity
	LastUpdated  time.Time
	TotalDrugs   int
	DownloadURL  string
}

// DrugHepatotoxicity contains hepatotoxicity data for a drug
type DrugHepatotoxicity struct {
	// Identification
	DrugName    string   `xml:"drugName" json:"drug_name"`
	Synonyms    []string `xml:"synonyms>synonym" json:"synonyms,omitempty"`
	PubChemCID  string   `xml:"pubChemCid" json:"pubchem_cid,omitempty"`
	RxCUI       string   `xml:"rxcui" json:"rxcui,omitempty"`
	DrugBankID  string   `xml:"drugBankId" json:"drugbank_id,omitempty"`
	ATCCodes    []string `xml:"atcCodes>code" json:"atc_codes,omitempty"`

	// Likelihood Score (A-E)
	LikelihoodScore    string `xml:"likelihoodScore" json:"likelihood_score"`
	LikelihoodCategory string `xml:"likelihoodCategory" json:"likelihood_category"`

	// Clinical Presentation
	LatencyRange    string `xml:"latencyRange" json:"latency_range"`       // "1-8 weeks"
	Pattern         string `xml:"pattern" json:"pattern"`                   // "Hepatocellular", "Cholestatic", "Mixed"
	Severity        string `xml:"severity" json:"severity"`                 // "Mild", "Moderate", "Severe"
	Chronicity      string `xml:"chronicity" json:"chronicity"`             // "Acute", "Chronic"
	RecoveryTime    string `xml:"recoveryTime" json:"recovery_time"`

	// Mechanism and Features
	Mechanism             string   `xml:"mechanism" json:"mechanism"`
	ImmunologicFeatures   bool     `xml:"immunologicFeatures" json:"immunologic_features"`
	Autoimmune            bool     `xml:"autoimmune" json:"autoimmune"`
	IsDoseDependent       bool     `xml:"isDoseDependent" json:"is_dose_dependent"`
	IsIdiosyncratic       bool     `xml:"isIdiosyncratic" json:"is_idiosyncratic"`
	HasBlackBoxWarning    bool     `xml:"hasBlackBoxWarning" json:"has_black_box_warning"`
	CrossReactivity       []string `xml:"crossReactivity>drug" json:"cross_reactivity,omitempty"`

	// Clinical Details
	HistologicalFindings  string   `xml:"histologicalFindings" json:"histological_findings"`
	CommonSymptoms        []string `xml:"symptoms>symptom" json:"common_symptoms,omitempty"`
	RiskFactors           []string `xml:"riskFactors>factor" json:"risk_factors,omitempty"`
	MonitoringRecommended []string `xml:"monitoring>item" json:"monitoring_recommended,omitempty"`

	// Case Statistics
	CasesReported     int     `xml:"casesReported" json:"cases_reported"`
	FatalCases        int     `xml:"fatalCases" json:"fatal_cases"`
	TransplantCases   int     `xml:"transplantCases" json:"transplant_cases"`
	IncidenceRate     string  `xml:"incidenceRate" json:"incidence_rate"`

	// References
	References     []Reference `xml:"references>reference" json:"references,omitempty"`
	ProductLabel   string      `xml:"productLabel" json:"product_label,omitempty"`
	LiverToxURL    string      `xml:"liverToxUrl" json:"livertox_url"`

	// Metadata
	LastUpdated    string `xml:"lastUpdated" json:"last_updated"`
	Version        string `xml:"version" json:"version"`
}

// Reference represents a literature reference
type Reference struct {
	PMID        string `xml:"pmid" json:"pmid"`
	Title       string `xml:"title" json:"title"`
	Authors     string `xml:"authors" json:"authors"`
	Journal     string `xml:"journal" json:"journal"`
	Year        string `xml:"year" json:"year"`
	Annotation  string `xml:"annotation" json:"annotation,omitempty"`
}

// =============================================================================
// RISK LEVELS
// =============================================================================

// HepatotoxicityRisk severity levels
type HepatotoxicityRisk string

const (
	// HepatoRiskHigh - Likelihood A-B (Well-known/Likely cause)
	HepatoRiskHigh HepatotoxicityRisk = "HIGH"

	// HepatoRiskModerate - Likelihood C (Probable cause)
	HepatoRiskModerate HepatotoxicityRisk = "MODERATE"

	// HepatoRiskLow - Likelihood D (Possible cause)
	HepatoRiskLow HepatotoxicityRisk = "LOW"

	// HepatoRiskUnlikely - Likelihood E (Unlikely cause)
	HepatoRiskUnlikely HepatotoxicityRisk = "UNLIKELY"

	// HepatoRiskUnknown - Not in database or unclassified
	HepatoRiskUnknown HepatotoxicityRisk = "UNKNOWN"
)

// =============================================================================
// LIKELIHOOD SCORE MAPPING
// =============================================================================

// Likelihood scores as defined by LiverTox:
// A = Well-known cause (>50 published cases with convincing evidence)
// B = Likely cause (at least 1 convincing case report, 12+ published cases)
// C = Probable cause (at least 1 convincing case report, 4-11 published cases)
// D = Possible cause (suspicious case reports, 1-3 published cases)
// E = Unlikely cause (no convincing evidence)

// GetRiskLevel converts LiverTox likelihood score to risk level
func (d *DrugHepatotoxicity) GetRiskLevel() HepatotoxicityRisk {
	switch strings.ToUpper(d.LikelihoodScore) {
	case "A", "B":
		return HepatoRiskHigh
	case "C":
		return HepatoRiskModerate
	case "D":
		return HepatoRiskLow
	case "E":
		return HepatoRiskUnlikely
	default:
		return HepatoRiskUnknown
	}
}

// GetRiskNumeric returns a numeric risk level for sorting (0-4)
func (d *DrugHepatotoxicity) GetRiskNumeric() int {
	switch d.GetRiskLevel() {
	case HepatoRiskHigh:
		return 4
	case HepatoRiskModerate:
		return 3
	case HepatoRiskLow:
		return 2
	case HepatoRiskUnlikely:
		return 1
	default:
		return 0
	}
}

// RequiresLFTMonitoring returns true if liver function test monitoring is recommended
func (d *DrugHepatotoxicity) RequiresLFTMonitoring() bool {
	risk := d.GetRiskLevel()
	return risk == HepatoRiskHigh || risk == HepatoRiskModerate || d.HasBlackBoxWarning
}

// IsSignificantRisk returns true if drug has clinically significant hepatotoxicity risk
func (d *DrugHepatotoxicity) IsSignificantRisk() bool {
	return d.GetRiskLevel() == HepatoRiskHigh || d.GetRiskLevel() == HepatoRiskModerate
}

// GetPattern returns the liver injury pattern
func (d *DrugHepatotoxicity) GetInjuryPattern() string {
	if d.Pattern == "" {
		return "Unknown"
	}
	return d.Pattern
}

// =============================================================================
// INGESTOR
// =============================================================================

// Ingestor handles LiverTox database updates
type Ingestor struct {
	downloadURL string
	httpClient  *http.Client
	factStore   FactStoreWriter
}

// FactStoreWriter interface for storing facts
type FactStoreWriter interface {
	StoreFact(ctx context.Context, fact interface{}) error
}

// Config for the ingestor
type IngestorConfig struct {
	DownloadURL string
	Timeout     time.Duration
}

// DefaultIngestorConfig returns default configuration
func DefaultIngestorConfig() IngestorConfig {
	return IngestorConfig{
		DownloadURL: "https://www.ncbi.nlm.nih.gov/books/NBK547852/xml/", // Placeholder
		Timeout:     5 * time.Minute,
	}
}

// NewIngestor creates a new LiverTox ingestor
func NewIngestor(config IngestorConfig, factStore FactStoreWriter) *Ingestor {
	return &Ingestor{
		downloadURL: config.DownloadURL,
		httpClient:  &http.Client{Timeout: config.Timeout},
		factStore:   factStore,
	}
}

// IngestFull downloads and processes the complete LiverTox database
func (i *Ingestor) IngestFull(ctx context.Context) (*LiverToxDB, error) {
	// Download LiverTox data
	resp, err := http.Get(i.downloadURL)
	if err != nil {
		return nil, fmt.Errorf("downloading LiverTox: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	return i.ParseXML(ctx, resp.Body)
}

// ParseXML parses LiverTox XML data
func (i *Ingestor) ParseXML(ctx context.Context, reader io.Reader) (*LiverToxDB, error) {
	db := &LiverToxDB{
		Drugs:       make(map[string]*DrugHepatotoxicity),
		LastUpdated: time.Now(),
		DownloadURL: i.downloadURL,
	}

	// Parse XML structure
	var xmlData struct {
		Drugs []DrugHepatotoxicity `xml:"drug"`
	}

	if err := xml.NewDecoder(reader).Decode(&xmlData); err != nil {
		return nil, fmt.Errorf("parsing LiverTox XML: %w", err)
	}

	// Index by drug name
	for idx := range xmlData.Drugs {
		drug := &xmlData.Drugs[idx]
		normalizedName := strings.ToLower(drug.DrugName)
		db.Drugs[normalizedName] = drug

		// Store as fact if factStore is available
		if i.factStore != nil {
			if err := i.factStore.StoreFact(ctx, drug); err != nil {
				return nil, fmt.Errorf("storing fact for %s: %w", drug.DrugName, err)
			}
		}
	}

	db.TotalDrugs = len(db.Drugs)
	return db, nil
}

// =============================================================================
// DATABASE QUERIES
// =============================================================================

// GetDrug retrieves hepatotoxicity data for a drug
func (db *LiverToxDB) GetDrug(drugName string) (*DrugHepatotoxicity, bool) {
	drug, found := db.Drugs[strings.ToLower(drugName)]
	return drug, found
}

// GetHighRiskDrugs returns all drugs with high hepatotoxicity risk
func (db *LiverToxDB) GetHighRiskDrugs() []*DrugHepatotoxicity {
	var drugs []*DrugHepatotoxicity
	for _, drug := range db.Drugs {
		if drug.GetRiskLevel() == HepatoRiskHigh {
			drugs = append(drugs, drug)
		}
	}
	return drugs
}

// GetDrugsWithBlackBoxWarning returns all drugs with black box warnings for hepatotoxicity
func (db *LiverToxDB) GetDrugsWithBlackBoxWarning() []*DrugHepatotoxicity {
	var drugs []*DrugHepatotoxicity
	for _, drug := range db.Drugs {
		if drug.HasBlackBoxWarning {
			drugs = append(drugs, drug)
		}
	}
	return drugs
}

// GetDrugsByPattern returns drugs by injury pattern
func (db *LiverToxDB) GetDrugsByPattern(pattern string) []*DrugHepatotoxicity {
	var drugs []*DrugHepatotoxicity
	pattern = strings.ToLower(pattern)
	for _, drug := range db.Drugs {
		if strings.ToLower(drug.Pattern) == pattern {
			drugs = append(drugs, drug)
		}
	}
	return drugs
}

// GetDrugsByRxCUI retrieves drug by RxNorm ID
func (db *LiverToxDB) GetDrugByRxCUI(rxcui string) (*DrugHepatotoxicity, bool) {
	for _, drug := range db.Drugs {
		if drug.RxCUI == rxcui {
			return drug, true
		}
	}
	return nil, false
}

// =============================================================================
// CLINICAL HELPERS
// =============================================================================

// HepatotoxicityAlert represents a clinical alert for hepatotoxicity
type HepatotoxicityAlert struct {
	DrugName          string             `json:"drug_name"`
	RiskLevel         HepatotoxicityRisk `json:"risk_level"`
	LikelihoodScore   string             `json:"likelihood_score"`
	Pattern           string             `json:"pattern"`
	HasBlackBoxWarning bool              `json:"has_black_box_warning"`
	Recommendations   []string           `json:"recommendations"`
	MonitoringAdvice  []string           `json:"monitoring_advice"`
}

// GenerateAlert creates a clinical alert for a drug
func (d *DrugHepatotoxicity) GenerateAlert() *HepatotoxicityAlert {
	alert := &HepatotoxicityAlert{
		DrugName:          d.DrugName,
		RiskLevel:         d.GetRiskLevel(),
		LikelihoodScore:   d.LikelihoodScore,
		Pattern:           d.Pattern,
		HasBlackBoxWarning: d.HasBlackBoxWarning,
		MonitoringAdvice:  d.MonitoringRecommended,
	}

	// Generate recommendations based on risk
	switch d.GetRiskLevel() {
	case HepatoRiskHigh:
		alert.Recommendations = []string{
			"Obtain baseline liver function tests before initiating therapy",
			"Monitor LFTs regularly during treatment",
			"Educate patient about signs/symptoms of liver injury",
			"Discontinue if ALT > 3x ULN with symptoms or ALT > 5x ULN",
		}
	case HepatoRiskModerate:
		alert.Recommendations = []string{
			"Consider baseline LFTs before initiating therapy",
			"Monitor LFTs periodically",
			"Advise patient to report symptoms of liver injury",
		}
	case HepatoRiskLow:
		alert.Recommendations = []string{
			"Routine LFT monitoring not required",
			"Advise patient to report unexplained symptoms",
		}
	}

	return alert
}

// =============================================================================
// AUTHORITY CLIENT INTERFACE IMPLEMENTATION (Phase 3b)
// =============================================================================

// LiverToxClient wraps the database and ingestor for AuthorityClient interface
type LiverToxClient struct {
	db       *LiverToxDB
	ingestor *Ingestor
}

// NewLiverToxClient creates a new LiverTox authority client
func NewLiverToxClient(config IngestorConfig) *LiverToxClient {
	return &LiverToxClient{
		ingestor: NewIngestor(config, nil),
	}
}

// Name returns the unique identifier for this data source
func (c *LiverToxClient) Name() string {
	return "LiverTox"
}

// HealthCheck verifies the data source is available
func (c *LiverToxClient) HealthCheck(ctx context.Context) error {
	if c.db == nil {
		return fmt.Errorf("LiverTox database not loaded")
	}
	return nil
}

// Close releases any resources
func (c *LiverToxClient) Close() error {
	return nil
}

// GetFacts retrieves hepatotoxicity facts for a drug by RxCUI
func (c *LiverToxClient) GetFacts(ctx context.Context, rxcui string) ([]AuthorityFact, error) {
	if c.db == nil {
		return nil, fmt.Errorf("LiverTox database not loaded")
	}

	drug, found := c.db.GetDrugByRxCUI(rxcui)
	if !found {
		return nil, nil
	}

	fact := c.drugToFact(drug, rxcui)
	return []AuthorityFact{fact}, nil
}

// GetFactsByName retrieves facts by drug name
func (c *LiverToxClient) GetFactsByName(ctx context.Context, drugName string) ([]AuthorityFact, error) {
	if c.db == nil {
		return nil, fmt.Errorf("LiverTox database not loaded")
	}

	drug, found := c.db.GetDrug(drugName)
	if !found {
		return nil, nil
	}

	fact := c.drugToFact(drug, "")
	return []AuthorityFact{fact}, nil
}

// GetFactByType retrieves a specific fact type for a drug
func (c *LiverToxClient) GetFactByType(ctx context.Context, rxcui string, factType FactType) (*AuthorityFact, error) {
	if factType != FactTypeHepatotoxicity {
		return nil, fmt.Errorf("LiverTox only provides %s facts", FactTypeHepatotoxicity)
	}

	facts, err := c.GetFacts(ctx, rxcui)
	if err != nil {
		return nil, err
	}

	if len(facts) == 0 {
		return nil, nil
	}

	return &facts[0], nil
}

// Sync synchronizes with LiverTox (full database download)
func (c *LiverToxClient) Sync(ctx context.Context) (*SyncResult, error) {
	startTime := time.Now()

	db, err := c.ingestor.IngestFull(ctx)
	if err != nil {
		return &SyncResult{
			Authority:  "LiverTox",
			StartTime:  startTime,
			EndTime:    time.Now(),
			ErrorCount: 1,
			Errors:     []string{err.Error()},
			Success:    false,
		}, err
	}

	c.db = db

	return &SyncResult{
		Authority:  "LiverTox",
		StartTime:  startTime,
		EndTime:    time.Now(),
		TotalFacts: db.TotalDrugs,
		NewFacts:   db.TotalDrugs,
		Success:    true,
	}, nil
}

// SyncDelta performs incremental sync
func (c *LiverToxClient) SyncDelta(ctx context.Context, since time.Time) (*SyncResult, error) {
	return c.Sync(ctx) // LiverTox doesn't support delta
}

// Authority returns metadata about LiverTox
func (c *LiverToxClient) Authority() AuthorityMetadata {
	return AuthorityMetadata{
		Name:            "LiverTox",
		FullName:        "LiverTox: Clinical and Research Information on Drug-Induced Liver Injury",
		URL:             "https://www.ncbi.nlm.nih.gov/books/NBK547852/",
		Description:     "NIH database of drug-induced liver injury with likelihood scores and clinical patterns",
		AuthorityLevel:  AuthorityDefinitive,
		DataFormat:      "XML_DOWNLOAD",
		UpdateFrequency: "QUARTERLY",
		FactTypes:       []FactType{FactTypeHepatotoxicity},
	}
}

// SupportedFactTypes returns the fact types provided by LiverTox
func (c *LiverToxClient) SupportedFactTypes() []FactType {
	return []FactType{FactTypeHepatotoxicity}
}

// LLMPolicy returns the LLM policy for LiverTox
func (c *LiverToxClient) LLMPolicy() LLMPolicy {
	return LLMNever
}

// drugToFact converts DrugHepatotoxicity to AuthorityFact
func (c *LiverToxClient) drugToFact(drug *DrugHepatotoxicity, rxcui string) AuthorityFact {
	alert := drug.GenerateAlert()

	fact := AuthorityFact{
		ID:               fmt.Sprintf("livertox-%s", drug.DrugName),
		AuthoritySource:  "LiverTox",
		FactType:         FactTypeHepatotoxicity,
		RxCUI:            rxcui,
		DrugName:         drug.DrugName,
		Content:          drug,
		EvidenceLevel:    drug.LikelihoodScore,
		ExtractionMethod: "AUTHORITY_LOOKUP",
		Confidence:       1.0,
		FetchedAt:        time.Now(),
		SourceURL:        drug.LiverToxURL,
		Recommendations:  alert.Recommendations,
	}

	// Map risk level
	switch drug.GetRiskLevel() {
	case HepatoRiskHigh:
		fact.RiskLevel = "HIGH"
		fact.ActionRequired = "MONITOR"
	case HepatoRiskModerate:
		fact.RiskLevel = "MODERATE"
		fact.ActionRequired = "CAUTION"
	case HepatoRiskLow:
		fact.RiskLevel = "LOW"
		fact.ActionRequired = "AWARENESS"
	default:
		fact.RiskLevel = "UNKNOWN"
	}

	// Add references
	for _, ref := range drug.References {
		if ref.PMID != "" {
			fact.References = append(fact.References, ref.PMID)
		}
	}

	return fact
}

// =============================================================================
// LOCAL TYPE ALIASES
// =============================================================================

type AuthorityFact struct {
	ID               string      `json:"id"`
	AuthoritySource  string      `json:"authority_source"`
	FactType         FactType    `json:"fact_type"`
	RxCUI            string      `json:"rxcui,omitempty"`
	DrugName         string      `json:"drug_name"`
	GenericName      string      `json:"generic_name,omitempty"`
	Content          interface{} `json:"content"`
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

type FactType string

const (
	FactTypeHepatotoxicity FactType = "HEPATOTOXICITY"
)

type AuthorityMetadata struct {
	Name            string         `json:"name"`
	FullName        string         `json:"full_name"`
	URL             string         `json:"url"`
	Description     string         `json:"description"`
	AuthorityLevel  AuthorityLevel `json:"authority_level"`
	DataFormat      string         `json:"data_format"`
	UpdateFrequency string         `json:"update_frequency"`
	FactTypes       []FactType     `json:"fact_types"`
}

type AuthorityLevel string

const (
	AuthorityDefinitive AuthorityLevel = "DEFINITIVE"
)

type LLMPolicy string

const (
	LLMNever LLMPolicy = "NEVER"
)

type SyncResult struct {
	Authority    string    `json:"authority"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	TotalFacts   int       `json:"total_facts"`
	NewFacts     int       `json:"new_facts"`
	UpdatedFacts int       `json:"updated_facts"`
	DeletedFacts int       `json:"deleted_facts"`
	ErrorCount   int       `json:"error_count"`
	Errors       []string  `json:"errors,omitempty"`
	Success      bool      `json:"success"`
}
