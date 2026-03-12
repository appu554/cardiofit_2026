// Package drugbank provides a client for accessing DrugBank pharmacological data.
// DrugBank is a comprehensive database containing detailed drug, drug-target,
// drug-action, and drug-drug interaction information.
//
// Data Source: https://go.drugbank.com/
// License: Academic/Commercial license required for full access
//
// AUTHORITY LEVEL: PRIMARY (LLM = GAP_FILL_ONLY)
// DrugBank provides structured PK parameters and DDI data, making it a primary
// authority for pharmacokinetic facts. LLM extraction is only permitted when
// DrugBank has no data for a specific drug.
package drugbank

import (
	"context"
	"database/sql"
	"encoding/json"
	"encoding/xml"
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
	FactTypePKParameters           FactType = "PK_PARAMETERS"
	FactTypeProteinBinding         FactType = "PROTEIN_BINDING"
	FactTypeDrugInteraction        FactType = "DRUG_INTERACTION"
	FactTypeCYPInteraction         FactType = "CYP_INTERACTION"
	FactTypeTransporterInteraction FactType = "TRANSPORTER_INTERACTION"
	FactTypeHepaticDosing          FactType = "HEPATIC_DOSING"
	FactTypeRenalDosing            FactType = "RENAL_DOSING"
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
	Authority     string    `json:"authority"`
	StartTime     time.Time `json:"start_time"`
	EndTime       time.Time `json:"end_time"`
	TotalRecords  int       `json:"total_records"`
	NewRecords    int       `json:"new_records"`
	UpdatedRecords int      `json:"updated_records"`
	DeletedRecords int      `json:"deleted_records"`
	ErrorCount    int       `json:"error_count"`
	Errors        []string  `json:"errors,omitempty"`
	SourceVersion string    `json:"source_version,omitempty"`
	Success       bool      `json:"success"`
}

// =============================================================================
// DRUGBANK DATA MODELS
// =============================================================================

// Drug represents a drug entry from DrugBank
type Drug struct {
	DrugBankID      string   `xml:"drugbank-id" json:"drugbank_id"`
	Name            string   `xml:"name" json:"name"`
	Type            string   `xml:"type,attr" json:"type"` // small molecule, biotech
	CASNumber       string   `xml:"cas-number" json:"cas_number,omitempty"`
	UNII            string   `xml:"unii" json:"unii,omitempty"`
	Description     string   `xml:"description" json:"description,omitempty"`
	State           string   `xml:"state" json:"state,omitempty"` // solid, liquid, gas

	// External identifiers
	RxCUI           string   `json:"rxcui,omitempty"`
	PubChemCID      string   `json:"pubchem_cid,omitempty"`
	ChEMBLID        string   `json:"chembl_id,omitempty"`

	// Pharmacokinetics
	Pharmacokinetics *PKParameters `xml:"pharmacokinetics" json:"pharmacokinetics,omitempty"`

	// Interactions
	DrugInteractions []DrugInteraction `xml:"drug-interactions>drug-interaction" json:"drug_interactions,omitempty"`

	// Enzymes and transporters
	Enzymes         []Enzyme     `xml:"enzymes>enzyme" json:"enzymes,omitempty"`
	Transporters    []Transporter `xml:"transporters>transporter" json:"transporters,omitempty"`
}

// PKParameters contains pharmacokinetic data from DrugBank
type PKParameters struct {
	// Absorption
	Bioavailability     string  `xml:"bioavailability" json:"bioavailability,omitempty"`
	BioavailabilityPct  float64 `json:"bioavailability_pct,omitempty"`
	AbsorptionText      string  `xml:"absorption" json:"absorption,omitempty"`
	Tmax                string  `json:"tmax,omitempty"`               // Time to peak concentration
	TmaxHours           float64 `json:"tmax_hours,omitempty"`

	// Distribution
	VolumeOfDistribution string  `xml:"volume-of-distribution" json:"volume_of_distribution,omitempty"`
	VdLiters            float64 `json:"vd_liters,omitempty"`
	ProteinBinding      string  `xml:"protein-binding" json:"protein_binding,omitempty"`
	ProteinBindingPct   float64 `json:"protein_binding_pct,omitempty"`

	// Metabolism
	Metabolism          string   `xml:"metabolism" json:"metabolism,omitempty"`
	MetabolizingEnzymes []string `json:"metabolizing_enzymes,omitempty"`
	ActiveMetabolites   []string `json:"active_metabolites,omitempty"`

	// Elimination
	HalfLife            string  `xml:"half-life" json:"half_life,omitempty"`
	HalfLifeHours       float64 `json:"half_life_hours,omitempty"`
	Clearance           string  `xml:"clearance" json:"clearance,omitempty"`
	ClearanceMLMin      float64 `json:"clearance_ml_min,omitempty"`
	RouteOfElimination  string  `xml:"route-of-elimination" json:"route_of_elimination,omitempty"`
	RenalExcretionPct   float64 `json:"renal_excretion_pct,omitempty"`
	HepaticExcretionPct float64 `json:"hepatic_excretion_pct,omitempty"`
}

// DrugInteraction represents a drug-drug interaction from DrugBank
type DrugInteraction struct {
	DrugBankID    string `xml:"drugbank-id" json:"drugbank_id"`
	Name          string `xml:"name" json:"name"`
	Description   string `xml:"description" json:"description"`

	// Parsed fields
	Severity      string `json:"severity,omitempty"`      // MAJOR, MODERATE, MINOR
	Mechanism     string `json:"mechanism,omitempty"`     // PK, PD, or MIXED
	Effect        string `json:"effect,omitempty"`        // INCREASED_EFFECT, DECREASED_EFFECT, etc.
	Management    string `json:"management,omitempty"`    // Clinical management recommendation
	Documentation string `json:"documentation,omitempty"` // EXCELLENT, GOOD, FAIR, POOR
}

// Enzyme represents an enzyme involved in drug metabolism
type Enzyme struct {
	ID           string   `xml:"id,attr" json:"id"`
	Name         string   `xml:"name" json:"name"`
	UniProtID    string   `xml:"uniprot-id" json:"uniprot_id,omitempty"`
	GeneSymbol   string   `xml:"gene-name" json:"gene_symbol,omitempty"`

	// Actions
	Actions      []string `xml:"actions>action" json:"actions,omitempty"` // substrate, inhibitor, inducer

	// Inhibition/Induction strength
	InhibitionStrength string `json:"inhibition_strength,omitempty"` // STRONG, MODERATE, WEAK
	InductionStrength  string `json:"induction_strength,omitempty"`  // STRONG, MODERATE, WEAK
}

// Transporter represents a drug transporter
type Transporter struct {
	ID         string   `xml:"id,attr" json:"id"`
	Name       string   `xml:"name" json:"name"`
	UniProtID  string   `xml:"uniprot-id" json:"uniprot_id,omitempty"`
	GeneSymbol string   `xml:"gene-name" json:"gene_symbol,omitempty"`
	Actions    []string `xml:"actions>action" json:"actions,omitempty"` // substrate, inhibitor, inducer
}

// =============================================================================
// DRUGBANK CLIENT
// =============================================================================

// Config holds configuration for the DrugBank client
type Config struct {
	// DataPath is the path to the DrugBank XML data file
	DataPath string

	// APIURL is the DrugBank API URL (for licensed API access)
	APIURL string

	// APIKey for authenticated access
	APIKey string

	// Database connection for persistent storage
	DB *sql.DB

	// HTTPClient for API requests
	HTTPClient *http.Client

	// CacheTTL for in-memory caching
	CacheTTL time.Duration
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		DataPath: "/data/drugbank/drugbank_all_full_database.xml",
		CacheTTL: 24 * time.Hour,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Client provides access to DrugBank data
type Client struct {
	config     Config
	drugs      map[string]*Drug // Indexed by DrugBank ID
	byRxCUI    map[string]*Drug // Indexed by RxCUI
	byName     map[string]*Drug // Indexed by lowercase name
	mu         sync.RWMutex
	loaded     bool
	loadedAt   time.Time
	version    string
}

// NewClient creates a new DrugBank client
func NewClient(config Config) *Client {
	return &Client{
		config:  config,
		drugs:   make(map[string]*Drug),
		byRxCUI: make(map[string]*Drug),
		byName:  make(map[string]*Drug),
	}
}

// =============================================================================
// DATASOURCE INTERFACE IMPLEMENTATION
// =============================================================================

// Name returns the data source name
func (c *Client) Name() string {
	return "DrugBank"
}

// HealthCheck verifies the data source is available
func (c *Client) HealthCheck(ctx context.Context) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.loaded {
		return fmt.Errorf("DrugBank data not loaded")
	}

	if len(c.drugs) == 0 {
		return fmt.Errorf("DrugBank has no drugs loaded")
	}

	return nil
}

// Close releases resources
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.drugs = make(map[string]*Drug)
	c.byRxCUI = make(map[string]*Drug)
	c.byName = make(map[string]*Drug)
	c.loaded = false

	return nil
}

// =============================================================================
// AUTHORITY CLIENT INTERFACE IMPLEMENTATION
// =============================================================================

// GetFacts retrieves all facts for a drug by RxCUI
func (c *Client) GetFacts(ctx context.Context, rxcui string) ([]AuthorityFact, error) {
	c.mu.RLock()
	drug, exists := c.byRxCUI[rxcui]
	c.mu.RUnlock()

	if !exists {
		return nil, nil // Not found - not an error
	}

	return c.drugToFacts(drug), nil
}

// GetFactsByName retrieves facts by drug name
func (c *Client) GetFactsByName(ctx context.Context, drugName string) ([]AuthorityFact, error) {
	c.mu.RLock()
	drug, exists := c.byName[strings.ToLower(drugName)]
	c.mu.RUnlock()

	if !exists {
		return nil, nil // Not found - not an error
	}

	return c.drugToFacts(drug), nil
}

// GetFactByType retrieves a specific fact type for a drug
func (c *Client) GetFactByType(ctx context.Context, rxcui string, factType FactType) (*AuthorityFact, error) {
	facts, err := c.GetFacts(ctx, rxcui)
	if err != nil {
		return nil, err
	}

	for _, fact := range facts {
		if fact.FactType == factType {
			return &fact, nil
		}
	}

	return nil, nil // Not found
}

// Sync synchronizes the local cache with DrugBank
func (c *Client) Sync(ctx context.Context) (*SyncResult, error) {
	result := &SyncResult{
		Authority: "DrugBank",
		StartTime: time.Now(),
	}

	// Try to load from XML file
	if c.config.DataPath != "" {
		if err := c.loadFromXML(ctx, c.config.DataPath); err != nil {
			result.Errors = append(result.Errors, err.Error())
			result.ErrorCount++
		}
	}

	result.EndTime = time.Now()
	result.TotalRecords = len(c.drugs)
	result.Success = result.ErrorCount == 0
	result.SourceVersion = c.version

	return result, nil
}

// SyncDelta performs incremental sync (DrugBank typically uses full sync)
func (c *Client) SyncDelta(ctx context.Context, since time.Time) (*SyncResult, error) {
	// DrugBank doesn't support delta sync - perform full sync
	return c.Sync(ctx)
}

// Authority returns metadata about DrugBank
func (c *Client) Authority() AuthorityMetadata {
	return AuthorityMetadata{
		Name:            "DrugBank",
		FullName:        "DrugBank Online Database",
		URL:             "https://go.drugbank.com/",
		Description:     "Comprehensive drug database with PK parameters, DDI, enzyme/transporter data",
		Level:           AuthorityPrimary,
		LLMPolicy:       LLMGapFillOnly,
		DataFormat:      "XML_DOWNLOAD",
		UpdateFrequency: "QUARTERLY",
		FactTypes: []FactType{
			FactTypePKParameters,
			FactTypeProteinBinding,
			FactTypeDrugInteraction,
			FactTypeCYPInteraction,
			FactTypeTransporterInteraction,
		},
		DrugCount: len(c.drugs),
		Version:   c.version,
		LastSync:  c.loadedAt,
	}
}

// SupportedFactTypes returns fact types provided by DrugBank
func (c *Client) SupportedFactTypes() []FactType {
	return []FactType{
		FactTypePKParameters,
		FactTypeProteinBinding,
		FactTypeDrugInteraction,
		FactTypeCYPInteraction,
		FactTypeTransporterInteraction,
	}
}

// LLMPolicy returns the LLM usage policy for DrugBank
func (c *Client) LLMPolicy() LLMPolicy {
	return LLMGapFillOnly
}

// =============================================================================
// DATA LOADING
// =============================================================================

// loadFromXML loads DrugBank data from XML file
func (c *Client) loadFromXML(ctx context.Context, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("opening DrugBank XML: %w", err)
	}
	defer file.Close()

	return c.parseXML(ctx, file)
}

// parseXML parses DrugBank XML and builds indices
func (c *Client) parseXML(ctx context.Context, reader io.Reader) error {
	decoder := xml.NewDecoder(reader)

	c.mu.Lock()
	defer c.mu.Unlock()

	// Clear existing data
	c.drugs = make(map[string]*Drug)
	c.byRxCUI = make(map[string]*Drug)
	c.byName = make(map[string]*Drug)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("parsing XML: %w", err)
		}

		switch elem := token.(type) {
		case xml.StartElement:
			if elem.Name.Local == "drug" {
				var drug Drug
				if err := decoder.DecodeElement(&drug, &elem); err != nil {
					continue // Skip malformed entries
				}

				// Index the drug
				c.drugs[drug.DrugBankID] = &drug
				c.byName[strings.ToLower(drug.Name)] = &drug

				if drug.RxCUI != "" {
					c.byRxCUI[drug.RxCUI] = &drug
				}
			}
		}
	}

	c.loaded = true
	c.loadedAt = time.Now()

	return nil
}

// LoadFromJSON loads DrugBank data from pre-processed JSON
func (c *Client) LoadFromJSON(ctx context.Context, reader io.Reader) error {
	var drugs []*Drug
	if err := json.NewDecoder(reader).Decode(&drugs); err != nil {
		return fmt.Errorf("parsing JSON: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.drugs = make(map[string]*Drug)
	c.byRxCUI = make(map[string]*Drug)
	c.byName = make(map[string]*Drug)

	for _, drug := range drugs {
		c.drugs[drug.DrugBankID] = drug
		c.byName[strings.ToLower(drug.Name)] = drug
		if drug.RxCUI != "" {
			c.byRxCUI[drug.RxCUI] = drug
		}
	}

	c.loaded = true
	c.loadedAt = time.Now()

	return nil
}

// =============================================================================
// FACT CONVERSION
// =============================================================================

// drugToFacts converts a Drug to AuthorityFacts
func (c *Client) drugToFacts(drug *Drug) []AuthorityFact {
	var facts []AuthorityFact
	now := time.Now()

	// PK Parameters fact
	if drug.Pharmacokinetics != nil {
		facts = append(facts, AuthorityFact{
			ID:               uuid.New().String(),
			AuthoritySource:  "DrugBank",
			FactType:         FactTypePKParameters,
			DrugRxCUI:        drug.RxCUI,
			DrugName:         drug.Name,
			FactValue:        drug.Pharmacokinetics,
			ExtractionMethod: "AUTHORITY_LOOKUP",
			Confidence:       1.0,
			FetchedAt:        now,
			SourceURL:        fmt.Sprintf("https://go.drugbank.com/drugs/%s", drug.DrugBankID),
		})

		// Separate protein binding fact if available
		if drug.Pharmacokinetics.ProteinBinding != "" {
			facts = append(facts, AuthorityFact{
				ID:               uuid.New().String(),
				AuthoritySource:  "DrugBank",
				FactType:         FactTypeProteinBinding,
				DrugRxCUI:        drug.RxCUI,
				DrugName:         drug.Name,
				FactValue:        drug.Pharmacokinetics.ProteinBindingPct,
				EvidenceLevel:    c.getProteinBindingEvidence(drug.Pharmacokinetics.ProteinBindingPct),
				ExtractionMethod: "AUTHORITY_LOOKUP",
				Confidence:       1.0,
				FetchedAt:        now,
				SourceURL:        fmt.Sprintf("https://go.drugbank.com/drugs/%s", drug.DrugBankID),
			})
		}
	}

	// Drug interactions
	for _, ddi := range drug.DrugInteractions {
		fact := c.ddiToFact(drug, &ddi)
		facts = append(facts, fact)
	}

	// CYP interactions from enzymes
	for _, enzyme := range drug.Enzymes {
		if strings.HasPrefix(enzyme.GeneSymbol, "CYP") {
			fact := c.enzymeToFact(drug, &enzyme)
			facts = append(facts, fact)
		}
	}

	// Transporter interactions
	for _, transporter := range drug.Transporters {
		fact := c.transporterToFact(drug, &transporter)
		facts = append(facts, fact)
	}

	return facts
}

// ddiToFact converts a DrugInteraction to AuthorityFact
func (c *Client) ddiToFact(drug *Drug, ddi *DrugInteraction) AuthorityFact {
	riskLevel := "MODERATE"
	actionRequired := "MONITOR"

	switch ddi.Severity {
	case "MAJOR":
		riskLevel = "HIGH"
		actionRequired = "AVOID"
	case "MINOR":
		riskLevel = "LOW"
		actionRequired = "NONE"
	}

	return AuthorityFact{
		ID:               uuid.New().String(),
		AuthoritySource:  "DrugBank",
		FactType:         FactTypeDrugInteraction,
		DrugRxCUI:        drug.RxCUI,
		DrugName:         drug.Name,
		FactValue: map[string]interface{}{
			"interacting_drug": ddi.Name,
			"description":      ddi.Description,
			"mechanism":        ddi.Mechanism,
			"effect":           ddi.Effect,
		},
		RiskLevel:        riskLevel,
		ActionRequired:   actionRequired,
		Recommendations:  []string{ddi.Management},
		ExtractionMethod: "AUTHORITY_LOOKUP",
		Confidence:       1.0,
		FetchedAt:        time.Now(),
		SourceURL:        fmt.Sprintf("https://go.drugbank.com/drugs/%s", drug.DrugBankID),
	}
}

// enzymeToFact converts an Enzyme to AuthorityFact
func (c *Client) enzymeToFact(drug *Drug, enzyme *Enzyme) AuthorityFact {
	riskLevel := "LOW"
	actionRequired := "NONE"

	// Determine risk based on inhibition/induction strength
	if enzyme.InhibitionStrength == "STRONG" || enzyme.InductionStrength == "STRONG" {
		riskLevel = "HIGH"
		actionRequired = "CAUTION"
	} else if enzyme.InhibitionStrength == "MODERATE" || enzyme.InductionStrength == "MODERATE" {
		riskLevel = "MODERATE"
		actionRequired = "MONITOR"
	}

	return AuthorityFact{
		ID:              uuid.New().String(),
		AuthoritySource: "DrugBank",
		FactType:        FactTypeCYPInteraction,
		DrugRxCUI:       drug.RxCUI,
		DrugName:        drug.Name,
		FactValue: map[string]interface{}{
			"enzyme":              enzyme.GeneSymbol,
			"actions":             enzyme.Actions,
			"inhibition_strength": enzyme.InhibitionStrength,
			"induction_strength":  enzyme.InductionStrength,
		},
		RiskLevel:        riskLevel,
		ActionRequired:   actionRequired,
		ExtractionMethod: "AUTHORITY_LOOKUP",
		Confidence:       1.0,
		FetchedAt:        time.Now(),
		SourceURL:        fmt.Sprintf("https://go.drugbank.com/drugs/%s", drug.DrugBankID),
	}
}

// transporterToFact converts a Transporter to AuthorityFact
func (c *Client) transporterToFact(drug *Drug, transporter *Transporter) AuthorityFact {
	return AuthorityFact{
		ID:              uuid.New().String(),
		AuthoritySource: "DrugBank",
		FactType:        FactTypeTransporterInteraction,
		DrugRxCUI:       drug.RxCUI,
		DrugName:        drug.Name,
		FactValue: map[string]interface{}{
			"transporter": transporter.GeneSymbol,
			"name":        transporter.Name,
			"actions":     transporter.Actions,
		},
		ExtractionMethod: "AUTHORITY_LOOKUP",
		Confidence:       1.0,
		FetchedAt:        time.Now(),
		SourceURL:        fmt.Sprintf("https://go.drugbank.com/drugs/%s", drug.DrugBankID),
	}
}

// getProteinBindingEvidence returns evidence level based on binding percentage
func (c *Client) getProteinBindingEvidence(pct float64) string {
	if pct > 90 {
		return "HIGH_BINDING" // Clinically significant
	} else if pct > 70 {
		return "MODERATE_BINDING"
	}
	return "LOW_BINDING"
}

// =============================================================================
// QUERY METHODS
// =============================================================================

// GetDrug retrieves a drug by DrugBank ID
func (c *Client) GetDrug(ctx context.Context, drugBankID string) (*Drug, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	drug, exists := c.drugs[drugBankID]
	if !exists {
		return nil, nil
	}

	return drug, nil
}

// GetDrugByRxCUI retrieves a drug by RxCUI
func (c *Client) GetDrugByRxCUI(ctx context.Context, rxcui string) (*Drug, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	drug, exists := c.byRxCUI[rxcui]
	if !exists {
		return nil, nil
	}

	return drug, nil
}

// GetPKParameters retrieves PK parameters for a drug
func (c *Client) GetPKParameters(ctx context.Context, rxcui string) (*PKParameters, error) {
	drug, err := c.GetDrugByRxCUI(ctx, rxcui)
	if err != nil {
		return nil, err
	}
	if drug == nil {
		return nil, nil
	}

	return drug.Pharmacokinetics, nil
}

// GetDrugInteractions retrieves all DDIs for a drug
func (c *Client) GetDrugInteractions(ctx context.Context, rxcui string) ([]DrugInteraction, error) {
	drug, err := c.GetDrugByRxCUI(ctx, rxcui)
	if err != nil {
		return nil, err
	}
	if drug == nil {
		return nil, nil
	}

	return drug.DrugInteractions, nil
}

// GetCYPProfile retrieves CYP enzyme profile for a drug
func (c *Client) GetCYPProfile(ctx context.Context, rxcui string) ([]Enzyme, error) {
	drug, err := c.GetDrugByRxCUI(ctx, rxcui)
	if err != nil {
		return nil, err
	}
	if drug == nil {
		return nil, nil
	}

	var cypEnzymes []Enzyme
	for _, enzyme := range drug.Enzymes {
		if strings.HasPrefix(enzyme.GeneSymbol, "CYP") {
			cypEnzymes = append(cypEnzymes, enzyme)
		}
	}

	return cypEnzymes, nil
}

// IsStrongCYPInhibitor checks if drug is a strong CYP inhibitor
func (c *Client) IsStrongCYPInhibitor(ctx context.Context, rxcui string, cypGene string) (bool, error) {
	drug, err := c.GetDrugByRxCUI(ctx, rxcui)
	if err != nil {
		return false, err
	}
	if drug == nil {
		return false, nil
	}

	for _, enzyme := range drug.Enzymes {
		if enzyme.GeneSymbol == cypGene && enzyme.InhibitionStrength == "STRONG" {
			for _, action := range enzyme.Actions {
				if action == "inhibitor" {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

// GetTransporters retrieves transporter profile for a drug
func (c *Client) GetTransporters(ctx context.Context, rxcui string) ([]Transporter, error) {
	drug, err := c.GetDrugByRxCUI(ctx, rxcui)
	if err != nil {
		return nil, err
	}
	if drug == nil {
		return nil, nil
	}

	return drug.Transporters, nil
}
