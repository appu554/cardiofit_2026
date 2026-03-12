// Package lactmed provides ingestion for the LactMed database.
// LactMed is NIH's database of drugs and lactation with safety information for breastfeeding.
//
// Phase 3b.4: LactMed XML Ingestion
// Authority Level: DEFINITIVE (LLM = ❌ NEVER)
//
// Data Source: https://www.ncbi.nlm.nih.gov/books/NBK501922/
// LactMed is freely available from NLM/NCBI
package lactmed

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
// LACTMED DATABASE
// =============================================================================

// LactMedDB represents the full LactMed database
type LactMedDB struct {
	Drugs        map[string]*DrugLactation
	LastUpdated  time.Time
	TotalDrugs   int
	DownloadURL  string
}

// DrugLactation contains breastfeeding safety data from LactMed
type DrugLactation struct {
	// Identification
	DrugName      string   `xml:"drugName" json:"drug_name"`
	Synonyms      []string `xml:"synonyms>synonym" json:"synonyms,omitempty"`
	RxCUI         string   `xml:"rxcui" json:"rxcui,omitempty"`
	DrugBankID    string   `xml:"drugBankId" json:"drugbank_id,omitempty"`
	CASNumber     string   `xml:"casNumber" json:"cas_number,omitempty"`

	// Relative Infant Dose (RID) - Key Safety Metric
	// RID < 10% is generally considered acceptable
	RIDPercent    *float64 `xml:"ridPercent" json:"rid_percent,omitempty"`
	RIDRange      string   `xml:"ridRange" json:"rid_range,omitempty"` // "0.1-2%"
	RIDCategory   string   `xml:"ridCategory" json:"rid_category"`      // "LOW", "MODERATE", "HIGH"

	// Safety Classification
	SafetyCategory        string `xml:"safetyCategory" json:"safety_category"`
	AAPRecommendation     string `xml:"aapRecommendation" json:"aap_recommendation"` // American Academy of Pediatrics
	WHORecommendation     string `xml:"whoRecommendation" json:"who_recommendation"`
	CompatibilityStatus   string `xml:"compatibilityStatus" json:"compatibility_status"`

	// Pharmacokinetics Relevant to Lactation
	ExcretedInMilk        bool     `xml:"excretedInMilk" json:"excreted_in_milk"`
	OralBioavailability   string   `xml:"oralBioavailability" json:"oral_bioavailability"`
	MilkPlasmaRatio       *float64 `xml:"milkPlasmaRatio" json:"milk_plasma_ratio,omitempty"` // M/P ratio
	ProteinBinding        string   `xml:"proteinBinding" json:"protein_binding"`
	HalfLife              string   `xml:"halfLife" json:"half_life"`
	MolecularWeight       float64  `xml:"molecularWeight" json:"molecular_weight,omitempty"`

	// Clinical Effects
	InfantEffects         []string `xml:"infantEffects>effect" json:"infant_effects,omitempty"`
	MaternalEffects       []string `xml:"maternalEffects>effect" json:"maternal_effects,omitempty"`
	MilkSupplyEffect      string   `xml:"milkSupplyEffect" json:"milk_supply_effect"` // "Increases", "Decreases", "No effect"

	// Risk Factors
	PrematureInfantRisk   bool     `xml:"prematureInfantRisk" json:"premature_infant_risk"`
	NewbornRisk           bool     `xml:"newbornRisk" json:"newborn_risk"`
	SpecialPopulations    []string `xml:"specialPopulations>population" json:"special_populations,omitempty"`

	// Recommendations
	MonitoringRequired    []string `xml:"monitoring>item" json:"monitoring_required,omitempty"`
	Alternatives          []string `xml:"alternatives>drug" json:"alternatives,omitempty"`
	TimingAdvice          string   `xml:"timingAdvice" json:"timing_advice"` // e.g., "Nurse before dose"
	MaxDuration           string   `xml:"maxDuration" json:"max_duration,omitempty"`

	// Drug Levels in Milk
	MilkLevels            string   `xml:"milkLevels" json:"milk_levels,omitempty"`
	InfantDoseEstimate    string   `xml:"infantDoseEstimate" json:"infant_dose_estimate,omitempty"`

	// References
	References            []Reference `xml:"references>reference" json:"references,omitempty"`
	LactMedURL            string      `xml:"lactmedUrl" json:"lactmed_url"`

	// Metadata
	LastUpdated           string `xml:"lastUpdated" json:"last_updated"`
	Version               string `xml:"version" json:"version"`
}

// Reference represents a literature reference
type Reference struct {
	PMID       string `xml:"pmid" json:"pmid"`
	Title      string `xml:"title" json:"title"`
	Authors    string `xml:"authors" json:"authors"`
	Journal    string `xml:"journal" json:"journal"`
	Year       string `xml:"year" json:"year"`
}

// =============================================================================
// SAFETY LEVELS
// =============================================================================

// LactationSafetyLevel represents breastfeeding safety classification
type LactationSafetyLevel string

const (
	// LactationSafe - RID <2%, no documented concerns
	// Generally considered compatible with breastfeeding
	LactationSafe LactationSafetyLevel = "SAFE"

	// LactationProbablySafe - RID 2-10%, monitor infant
	// Compatible with breastfeeding with monitoring
	LactationProbablySafe LactationSafetyLevel = "PROBABLY_SAFE"

	// LactationUseWithCaution - RID 10-25% or documented risks
	// Use with caution, monitor infant closely
	LactationUseWithCaution LactationSafetyLevel = "USE_CAUTION"

	// LactationAvoid - RID >25% or contraindicated
	// Should be avoided during breastfeeding
	LactationAvoid LactationSafetyLevel = "AVOID"

	// LactationContraindicated - Absolutely contraindicated
	// Do not use while breastfeeding
	LactationContraindicated LactationSafetyLevel = "CONTRAINDICATED"

	// LactationUnknown - Insufficient data
	LactationUnknown LactationSafetyLevel = "UNKNOWN"
)

// =============================================================================
// SAFETY METHODS
// =============================================================================

// GetSafetyLevel calculates breastfeeding safety from RID
func (d *DrugLactation) GetSafetyLevel() LactationSafetyLevel {
	// First check explicit safety category
	switch strings.ToLower(d.SafetyCategory) {
	case "contraindicated":
		return LactationContraindicated
	case "avoid":
		return LactationAvoid
	case "compatible":
		return LactationSafe
	}

	// Calculate from RID if available
	if d.RIDPercent != nil {
		rid := *d.RIDPercent
		switch {
		case rid < 2:
			return LactationSafe
		case rid < 10:
			return LactationProbablySafe
		case rid < 25:
			return LactationUseWithCaution
		default:
			return LactationAvoid
		}
	}

	// Fall back to RID category
	switch strings.ToUpper(d.RIDCategory) {
	case "LOW":
		return LactationProbablySafe
	case "MODERATE":
		return LactationUseWithCaution
	case "HIGH":
		return LactationAvoid
	}

	return LactationUnknown
}

// IsCompatible returns true if drug is generally compatible with breastfeeding
func (d *DrugLactation) IsCompatible() bool {
	level := d.GetSafetyLevel()
	return level == LactationSafe || level == LactationProbablySafe
}

// RequiresMonitoring returns true if infant monitoring is recommended
func (d *DrugLactation) RequiresMonitoring() bool {
	return len(d.MonitoringRequired) > 0 ||
		d.GetSafetyLevel() == LactationUseWithCaution ||
		d.PrematureInfantRisk ||
		d.NewbornRisk
}

// GetRiskNumeric returns a numeric risk level for sorting (0-5)
func (d *DrugLactation) GetRiskNumeric() int {
	switch d.GetSafetyLevel() {
	case LactationContraindicated:
		return 5
	case LactationAvoid:
		return 4
	case LactationUseWithCaution:
		return 3
	case LactationProbablySafe:
		return 2
	case LactationSafe:
		return 1
	default:
		return 0
	}
}

// GetRIDDescription returns a human-readable RID description
func (d *DrugLactation) GetRIDDescription() string {
	if d.RIDPercent != nil {
		rid := *d.RIDPercent
		switch {
		case rid < 2:
			return fmt.Sprintf("Very Low (%.1f%%) - Generally acceptable", rid)
		case rid < 10:
			return fmt.Sprintf("Low (%.1f%%) - Usually acceptable", rid)
		case rid < 25:
			return fmt.Sprintf("Moderate (%.1f%%) - Use with caution", rid)
		default:
			return fmt.Sprintf("High (%.1f%%) - Consider alternatives", rid)
		}
	}
	if d.RIDRange != "" {
		return d.RIDRange
	}
	return "Unknown"
}

// =============================================================================
// INGESTOR
// =============================================================================

// Ingestor handles LactMed database updates
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
		DownloadURL: "https://www.ncbi.nlm.nih.gov/books/NBK501922/xml/", // Placeholder
		Timeout:     5 * time.Minute,
	}
}

// NewIngestor creates a new LactMed ingestor
func NewIngestor(config IngestorConfig, factStore FactStoreWriter) *Ingestor {
	return &Ingestor{
		downloadURL: config.DownloadURL,
		httpClient:  &http.Client{Timeout: config.Timeout},
		factStore:   factStore,
	}
}

// IngestFull downloads and processes the complete LactMed database
func (i *Ingestor) IngestFull(ctx context.Context) (*LactMedDB, error) {
	resp, err := http.Get(i.downloadURL)
	if err != nil {
		return nil, fmt.Errorf("downloading LactMed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	return i.ParseXML(ctx, resp.Body)
}

// ParseXML parses LactMed XML data
func (i *Ingestor) ParseXML(ctx context.Context, reader io.Reader) (*LactMedDB, error) {
	db := &LactMedDB{
		Drugs:       make(map[string]*DrugLactation),
		LastUpdated: time.Now(),
		DownloadURL: i.downloadURL,
	}

	var xmlData struct {
		Drugs []DrugLactation `xml:"drug"`
	}

	if err := xml.NewDecoder(reader).Decode(&xmlData); err != nil {
		return nil, fmt.Errorf("parsing LactMed XML: %w", err)
	}

	for idx := range xmlData.Drugs {
		drug := &xmlData.Drugs[idx]
		normalizedName := strings.ToLower(drug.DrugName)
		db.Drugs[normalizedName] = drug

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

// GetDrug retrieves lactation data for a drug
func (db *LactMedDB) GetDrug(drugName string) (*DrugLactation, bool) {
	drug, found := db.Drugs[strings.ToLower(drugName)]
	return drug, found
}

// GetDrugByRxCUI retrieves drug by RxNorm ID
func (db *LactMedDB) GetDrugByRxCUI(rxcui string) (*DrugLactation, bool) {
	for _, drug := range db.Drugs {
		if drug.RxCUI == rxcui {
			return drug, true
		}
	}
	return nil, false
}

// GetSafeDrugs returns all drugs safe for breastfeeding
func (db *LactMedDB) GetSafeDrugs() []*DrugLactation {
	var drugs []*DrugLactation
	for _, drug := range db.Drugs {
		if drug.IsCompatible() {
			drugs = append(drugs, drug)
		}
	}
	return drugs
}

// GetContraindicatedDrugs returns all contraindicated drugs
func (db *LactMedDB) GetContraindicatedDrugs() []*DrugLactation {
	var drugs []*DrugLactation
	for _, drug := range db.Drugs {
		level := drug.GetSafetyLevel()
		if level == LactationContraindicated || level == LactationAvoid {
			drugs = append(drugs, drug)
		}
	}
	return drugs
}

// GetAlternatives returns safer alternatives for a drug
func (db *LactMedDB) GetAlternatives(drugName string) []*DrugLactation {
	drug, found := db.GetDrug(drugName)
	if !found || len(drug.Alternatives) == 0 {
		return nil
	}

	var alternatives []*DrugLactation
	for _, altName := range drug.Alternatives {
		if alt, found := db.GetDrug(altName); found {
			alternatives = append(alternatives, alt)
		}
	}
	return alternatives
}

// =============================================================================
// CLINICAL HELPERS
// =============================================================================

// LactationAlert represents a clinical alert for lactation safety
type LactationAlert struct {
	DrugName            string               `json:"drug_name"`
	SafetyLevel         LactationSafetyLevel `json:"safety_level"`
	RIDPercent          *float64             `json:"rid_percent,omitempty"`
	ExcretedInMilk      bool                 `json:"excreted_in_milk"`
	InfantEffects       []string             `json:"infant_effects,omitempty"`
	Recommendations     []string             `json:"recommendations"`
	Alternatives        []string             `json:"alternatives,omitempty"`
	MonitoringAdvice    []string             `json:"monitoring_advice,omitempty"`
	SpecialConsiderations []string           `json:"special_considerations,omitempty"`
}

// GenerateAlert creates a clinical alert for breastfeeding
func (d *DrugLactation) GenerateAlert() *LactationAlert {
	alert := &LactationAlert{
		DrugName:       d.DrugName,
		SafetyLevel:    d.GetSafetyLevel(),
		RIDPercent:     d.RIDPercent,
		ExcretedInMilk: d.ExcretedInMilk,
		InfantEffects:  d.InfantEffects,
		Alternatives:   d.Alternatives,
		MonitoringAdvice: d.MonitoringRequired,
	}

	// Add special considerations
	if d.PrematureInfantRisk {
		alert.SpecialConsiderations = append(alert.SpecialConsiderations,
			"Increased risk in premature infants - use with extra caution")
	}
	if d.NewbornRisk {
		alert.SpecialConsiderations = append(alert.SpecialConsiderations,
			"Increased risk in newborns (< 1 month) - monitor closely")
	}
	if d.MilkSupplyEffect == "Decreases" {
		alert.SpecialConsiderations = append(alert.SpecialConsiderations,
			"May decrease milk supply")
	}

	// Generate recommendations based on safety level
	switch d.GetSafetyLevel() {
	case LactationSafe:
		alert.Recommendations = []string{
			"Compatible with breastfeeding",
			"No special precautions required",
		}
	case LactationProbablySafe:
		alert.Recommendations = []string{
			"Generally compatible with breastfeeding",
			"Monitor infant for unusual symptoms",
		}
	case LactationUseWithCaution:
		alert.Recommendations = []string{
			"Use with caution during breastfeeding",
			"Monitor infant closely for adverse effects",
			"Consider alternative medications if available",
		}
		if d.TimingAdvice != "" {
			alert.Recommendations = append(alert.Recommendations, d.TimingAdvice)
		}
	case LactationAvoid, LactationContraindicated:
		alert.Recommendations = []string{
			"Avoid use during breastfeeding",
			"Consider alternative medications",
			"If essential, consider temporary cessation of breastfeeding",
		}
	default:
		alert.Recommendations = []string{
			"Limited data on breastfeeding safety",
			"Consult with healthcare provider",
			"Consider alternative medications with more safety data",
		}
	}

	return alert
}

// =============================================================================
// AUTHORITY CLIENT INTERFACE IMPLEMENTATION (Phase 3b)
// =============================================================================

// LactMedClient wraps the database and ingestor for AuthorityClient interface
type LactMedClient struct {
	db       *LactMedDB
	ingestor *Ingestor
}

// NewLactMedClient creates a new LactMed authority client
func NewLactMedClient(config IngestorConfig) *LactMedClient {
	return &LactMedClient{
		ingestor: NewIngestor(config, nil),
	}
}

// Name returns the unique identifier for this data source
func (c *LactMedClient) Name() string {
	return "LactMed"
}

// HealthCheck verifies the data source is available
func (c *LactMedClient) HealthCheck(ctx context.Context) error {
	if c.db == nil {
		return fmt.Errorf("LactMed database not loaded")
	}
	return nil
}

// Close releases any resources
func (c *LactMedClient) Close() error {
	return nil
}

// GetFacts retrieves lactation safety facts for a drug by RxCUI
func (c *LactMedClient) GetFacts(ctx context.Context, rxcui string) ([]AuthorityFact, error) {
	if c.db == nil {
		return nil, fmt.Errorf("LactMed database not loaded")
	}

	drug, found := c.db.GetDrugByRxCUI(rxcui)
	if !found {
		return nil, nil
	}

	fact := c.drugToFact(drug, rxcui)
	return []AuthorityFact{fact}, nil
}

// GetFactsByName retrieves facts by drug name
func (c *LactMedClient) GetFactsByName(ctx context.Context, drugName string) ([]AuthorityFact, error) {
	if c.db == nil {
		return nil, fmt.Errorf("LactMed database not loaded")
	}

	drug, found := c.db.GetDrug(drugName)
	if !found {
		return nil, nil
	}

	fact := c.drugToFact(drug, "")
	return []AuthorityFact{fact}, nil
}

// GetFactByType retrieves a specific fact type for a drug
func (c *LactMedClient) GetFactByType(ctx context.Context, rxcui string, factType FactType) (*AuthorityFact, error) {
	if factType != FactTypeLactationSafety {
		return nil, fmt.Errorf("LactMed only provides %s facts", FactTypeLactationSafety)
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

// Sync synchronizes with LactMed (full database download)
func (c *LactMedClient) Sync(ctx context.Context) (*SyncResult, error) {
	startTime := time.Now()

	db, err := c.ingestor.IngestFull(ctx)
	if err != nil {
		return &SyncResult{
			Authority:  "LactMed",
			StartTime:  startTime,
			EndTime:    time.Now(),
			ErrorCount: 1,
			Errors:     []string{err.Error()},
			Success:    false,
		}, err
	}

	c.db = db

	return &SyncResult{
		Authority:  "LactMed",
		StartTime:  startTime,
		EndTime:    time.Now(),
		TotalFacts: db.TotalDrugs,
		NewFacts:   db.TotalDrugs,
		Success:    true,
	}, nil
}

// SyncDelta performs incremental sync
func (c *LactMedClient) SyncDelta(ctx context.Context, since time.Time) (*SyncResult, error) {
	return c.Sync(ctx) // LactMed doesn't support delta
}

// Authority returns metadata about LactMed
func (c *LactMedClient) Authority() AuthorityMetadata {
	return AuthorityMetadata{
		Name:            "LactMed",
		FullName:        "LactMed: Drugs and Lactation Database",
		URL:             "https://www.ncbi.nlm.nih.gov/books/NBK501922/",
		Description:     "NIH database of drugs and breastfeeding with Relative Infant Dose and safety assessments",
		AuthorityLevel:  AuthorityDefinitive,
		DataFormat:      "XML_DOWNLOAD",
		UpdateFrequency: "MONTHLY",
		FactTypes:       []FactType{FactTypeLactationSafety},
	}
}

// SupportedFactTypes returns the fact types provided by LactMed
func (c *LactMedClient) SupportedFactTypes() []FactType {
	return []FactType{FactTypeLactationSafety}
}

// LLMPolicy returns the LLM policy for LactMed
func (c *LactMedClient) LLMPolicy() LLMPolicy {
	return LLMNever
}

// drugToFact converts DrugLactation to AuthorityFact
func (c *LactMedClient) drugToFact(drug *DrugLactation, rxcui string) AuthorityFact {
	alert := drug.GenerateAlert()

	fact := AuthorityFact{
		ID:               fmt.Sprintf("lactmed-%s", drug.DrugName),
		AuthoritySource:  "LactMed",
		FactType:         FactTypeLactationSafety,
		RxCUI:            rxcui,
		DrugName:         drug.DrugName,
		Content:          drug,
		ExtractionMethod: "AUTHORITY_LOOKUP",
		Confidence:       1.0,
		FetchedAt:        time.Now(),
		SourceURL:        drug.LactMedURL,
		Recommendations:  alert.Recommendations,
	}

	// Add RID description if available
	if drug.RIDPercent != nil {
		fact.EvidenceLevel = drug.GetRIDDescription()
	}

	// Map safety level to risk level and action
	switch drug.GetSafetyLevel() {
	case LactationSafe:
		fact.RiskLevel = "NONE"
		fact.ActionRequired = "NONE"
	case LactationProbablySafe:
		fact.RiskLevel = "LOW"
		fact.ActionRequired = "MONITOR"
	case LactationUseWithCaution:
		fact.RiskLevel = "MODERATE"
		fact.ActionRequired = "CAUTION"
	case LactationAvoid:
		fact.RiskLevel = "HIGH"
		fact.ActionRequired = "AVOID"
	case LactationContraindicated:
		fact.RiskLevel = "CRITICAL"
		fact.ActionRequired = "CONTRAINDICATED"
	default:
		fact.RiskLevel = "UNKNOWN"
		fact.ActionRequired = "CONSULT"
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
	FactTypeLactationSafety FactType = "LACTATION_SAFETY"
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
