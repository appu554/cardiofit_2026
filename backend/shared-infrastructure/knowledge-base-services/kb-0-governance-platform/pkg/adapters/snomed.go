// Package adapters provides SNOMED CT (Systematized Nomenclature of Medicine) adapter.
package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"kb-0-governance-platform/internal/models"
)

// =============================================================================
// SNOMED CT ADAPTER (GLOBAL)
// =============================================================================

// SNOMEDCTAdapter ingests clinical terminology from SNOMED CT.
// Used by KB-7 (Terminology), KB-2 (Clinical Context), KB-3 (Guidelines).
type SNOMEDCTAdapter struct {
	*BaseAdapter
	baseURL    string
	httpClient *http.Client
	edition    string // International, US, UK, AU, etc.
	version    string
}

// NewSNOMEDCTAdapter creates a new SNOMED CT adapter.
func NewSNOMEDCTAdapter(edition string) *SNOMEDCTAdapter {
	return &SNOMEDCTAdapter{
		BaseAdapter: NewBaseAdapter(
			"SNOMED_CT",
			models.AuthoritySNOMED,
			[]models.KB{models.KB7, models.KB2, models.KB3},
		),
		baseURL: "https://snowstorm.ihtsdotools.org",
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		edition: edition,
		version: "MAIN", // Use MAIN for latest
	}
}

// FetchUpdates retrieves SNOMED concepts updated since the given timestamp.
func (a *SNOMEDCTAdapter) FetchUpdates(ctx context.Context, since time.Time) ([]RawItem, error) {
	// SNOMED CT Browser API - fetch concepts by effective time
	url := fmt.Sprintf("%s/%s/concepts?ecl=*&effectiveTime=%s",
		a.baseURL, a.edition, since.Format("20060102"))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var items []RawItem
	// Parse response
	return items, nil
}

// Transform converts raw SNOMED RF2 content to a KnowledgeItem.
func (a *SNOMEDCTAdapter) Transform(ctx context.Context, raw RawItem) (*models.KnowledgeItem, error) {
	concept, err := a.parseConcept(raw.RawData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SNOMED concept: %w", err)
	}

	item := &models.KnowledgeItem{
		ID:      fmt.Sprintf("kb7:snomed:%s", concept.ConceptID),
		Type:    models.TypeTerminology,
		KB:      models.KB7,
		Version: concept.EffectiveTime,
		Name:    concept.FSN,
		Description: concept.PreferredTerm,
		Source: models.SourceAttribution{
			Authority:    models.AuthoritySNOMED,
			Document:     "SNOMED CT International Edition",
			Section:      concept.Hierarchy,
			Jurisdiction: models.JurisdictionGlobal,
			URL:          fmt.Sprintf("https://browser.ihtsdotools.org/?perspective=full&conceptId1=%s", concept.ConceptID),
		},
		ContentRef:  fmt.Sprintf("snomed:concept:%s", concept.ConceptID),
		ContentHash: "",
		State:       models.StateDraft,
		RiskLevel:   models.RiskLow,
		WorkflowTemplate: models.TemplateInfraLow,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	return item, nil
}

// Validate performs SNOMED-specific validation.
func (a *SNOMEDCTAdapter) Validate(ctx context.Context, item *models.KnowledgeItem) error {
	if item.Source.Authority != models.AuthoritySNOMED {
		return fmt.Errorf("invalid authority: expected SNOMED, got %s", item.Source.Authority)
	}

	// Additional SNOMED-specific validation:
	// - Valid SNOMED CT ID format (numeric)
	// - Concept is active
	// - Has at least one description

	return nil
}

// =============================================================================
// SNOMED CONCEPT LOOKUP
// =============================================================================

// LookupConcept retrieves a single SNOMED concept by ID.
func (a *SNOMEDCTAdapter) LookupConcept(ctx context.Context, conceptID string) (*SNOMEDConcept, error) {
	url := fmt.Sprintf("%s/%s/concepts/%s", a.baseURL, a.edition, conceptID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return a.parseConcept(body)
}

// SearchConcepts searches for SNOMED concepts by term.
func (a *SNOMEDCTAdapter) SearchConcepts(ctx context.Context, term string, limit int) ([]*SNOMEDConcept, error) {
	searchURL := fmt.Sprintf("%s/%s/concepts?term=%s&limit=%d&activeFilter=true",
		a.baseURL, a.edition, url.QueryEscape(term), limit)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var result SNOMEDSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Items, nil
}

// GetDescendants retrieves descendant concepts using ECL query.
func (a *SNOMEDCTAdapter) GetDescendants(ctx context.Context, ancestorID string, limit int) ([]*SNOMEDConcept, error) {
	ecl := fmt.Sprintf("<< %s", ancestorID) // ECL for descendants
	eclURL := fmt.Sprintf("%s/%s/concepts?ecl=%s&limit=%d",
		a.baseURL, a.edition, url.QueryEscape(ecl), limit)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, eclURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var result SNOMEDSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Items, nil
}

// =============================================================================
// SNOMED STRUCTURES
// =============================================================================

// SNOMEDConcept represents a SNOMED CT concept.
type SNOMEDConcept struct {
	ConceptID         string              `json:"conceptId"`
	Active            bool                `json:"active"`
	EffectiveTime     string              `json:"effectiveTime"`
	ModuleID          string              `json:"moduleId"`
	DefinitionStatus  string              `json:"definitionStatus"`
	FSN               string              `json:"fsn"`               // Fully Specified Name
	PreferredTerm     string              `json:"pt"`                // Preferred Term
	Hierarchy         string              `json:"hierarchy,omitempty"`
	Descriptions      []SNOMEDDescription `json:"descriptions,omitempty"`
	Relationships     []SNOMEDRelationship `json:"relationships,omitempty"`
	InactivationInfo  *InactivationInfo   `json:"inactivationIndicator,omitempty"`
}

// SNOMEDDescription represents a concept description.
type SNOMEDDescription struct {
	DescriptionID string `json:"descriptionId"`
	Active        bool   `json:"active"`
	Term          string `json:"term"`
	TypeID        string `json:"typeId"`
	AcceptabilityMap map[string]string `json:"acceptabilityMap"`
}

// SNOMEDRelationship represents a concept relationship.
type SNOMEDRelationship struct {
	RelationshipID string        `json:"relationshipId"`
	Active         bool          `json:"active"`
	TypeID         string        `json:"typeId"`
	DestinationID  string        `json:"destinationId"`
	Target         *SNOMEDConcept `json:"target,omitempty"`
	CharacteristicTypeID string  `json:"characteristicTypeId"`
}

// InactivationInfo contains inactivation details for retired concepts.
type InactivationInfo struct {
	InactivationIndicator string   `json:"inactivationIndicator"`
	AssociationTargets    []string `json:"associationTargets"`
}

// SNOMEDSearchResult represents search results.
type SNOMEDSearchResult struct {
	Items      []*SNOMEDConcept `json:"items"`
	Total      int              `json:"total"`
	Limit      int              `json:"limit"`
	Offset     int              `json:"offset"`
	SearchAfter string          `json:"searchAfter"`
}

// parseConcept parses SNOMED concept JSON.
func (a *SNOMEDCTAdapter) parseConcept(data []byte) (*SNOMEDConcept, error) {
	var concept SNOMEDConcept
	if err := json.Unmarshal(data, &concept); err != nil {
		return nil, fmt.Errorf("failed to parse concept: %w", err)
	}
	return &concept, nil
}

// =============================================================================
// SNOMED HIERARCHIES (TOP-LEVEL CONCEPTS)
// =============================================================================

// SNOMED CT Top-Level Hierarchies
const (
	SNOMEDClinicalFinding      = "404684003" // Clinical finding
	SNOMEDProcedure            = "71388002"  // Procedure
	SNOMEDObservableEntity     = "363787002" // Observable entity
	SNOMEDBodyStructure        = "123037004" // Body structure
	SNOMEDOrganism             = "410607006" // Organism
	SNOMEDSubstance            = "105590001" // Substance
	SNOMEDPharmaceuticalProduct = "373873005" // Pharmaceutical / biologic product
	SNOMEDSpecimen             = "123038009" // Specimen
	SNOMEDSituation            = "243796009" // Situation with explicit context
	SNOMEDEvent                = "272379006" // Event
	SNOMEDEnvironment          = "308916002" // Environment
	SNOMEDSocialContext        = "48176007"  // Social context
	SNOMEDQualifierValue       = "362981000" // Qualifier value
	SNOMEDPhysicalObject       = "260787004" // Physical object
	SNOMEDPhysicalForce        = "78621006"  // Physical force
	SNOMEDLinkageAssertion     = "106237007" // Linkage concept
	SNOMEDRecordArtifact       = "419891008" // Record artifact
)

// HierarchyInfo contains information about a SNOMED hierarchy.
type HierarchyInfo struct {
	ConceptID   string
	Name        string
	Description string
}

// GetHierarchies returns the top-level SNOMED CT hierarchies.
func GetHierarchies() []HierarchyInfo {
	return []HierarchyInfo{
		{SNOMEDClinicalFinding, "Clinical finding", "Diseases, disorders, and clinical observations"},
		{SNOMEDProcedure, "Procedure", "Actions performed in healthcare"},
		{SNOMEDObservableEntity, "Observable entity", "Things that can be measured or observed"},
		{SNOMEDBodyStructure, "Body structure", "Anatomical structures"},
		{SNOMEDOrganism, "Organism", "Living organisms including pathogens"},
		{SNOMEDSubstance, "Substance", "Chemical substances"},
		{SNOMEDPharmaceuticalProduct, "Pharmaceutical product", "Drugs and medications"},
		{SNOMEDSpecimen, "Specimen", "Biological specimens"},
		{SNOMEDSituation, "Situation", "Clinical situations with context"},
		{SNOMEDEvent, "Event", "Occurrences and incidents"},
	}
}

// =============================================================================
// EXPRESSION CONSTRAINT LANGUAGE (ECL) HELPERS
// =============================================================================

// ECLQuery builds common ECL queries for SNOMED CT.
type ECLQuery struct {
	builder strings.Builder
}

// NewECLQuery creates a new ECL query builder.
func NewECLQuery() *ECLQuery {
	return &ECLQuery{}
}

// Concept adds a concept ID to the query.
func (q *ECLQuery) Concept(conceptID string) *ECLQuery {
	q.builder.WriteString(conceptID)
	return q
}

// Descendants adds descendant constraint (<< conceptID).
func (q *ECLQuery) Descendants(conceptID string) *ECLQuery {
	q.builder.WriteString("<< ")
	q.builder.WriteString(conceptID)
	return q
}

// Children adds child constraint (< conceptID).
func (q *ECLQuery) Children(conceptID string) *ECLQuery {
	q.builder.WriteString("< ")
	q.builder.WriteString(conceptID)
	return q
}

// Ancestors adds ancestor constraint (>> conceptID).
func (q *ECLQuery) Ancestors(conceptID string) *ECLQuery {
	q.builder.WriteString(">> ")
	q.builder.WriteString(conceptID)
	return q
}

// Parents adds parent constraint (> conceptID).
func (q *ECLQuery) Parents(conceptID string) *ECLQuery {
	q.builder.WriteString("> ")
	q.builder.WriteString(conceptID)
	return q
}

// And adds conjunction operator.
func (q *ECLQuery) And() *ECLQuery {
	q.builder.WriteString(" AND ")
	return q
}

// Or adds disjunction operator.
func (q *ECLQuery) Or() *ECLQuery {
	q.builder.WriteString(" OR ")
	return q
}

// Build returns the ECL query string.
func (q *ECLQuery) Build() string {
	return q.builder.String()
}

// =============================================================================
// MAPPING SUPPORT
// =============================================================================

// MapToICD10 retrieves ICD-10 mappings for a SNOMED concept.
func (a *SNOMEDCTAdapter) MapToICD10(ctx context.Context, conceptID string) ([]CodeMapping, error) {
	// SNOMED CT to ICD-10 map set
	mapURL := fmt.Sprintf("%s/%s/members?referencedComponentId=%s&referenceSet=447562003",
		a.baseURL, a.edition, conceptID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, mapURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	// Parse mapping results
	var mappings []CodeMapping
	// In production: parse reference set members
	return mappings, nil
}

// CodeMapping represents a code system mapping.
type CodeMapping struct {
	SourceCode       string `json:"source_code"`
	SourceSystem     string `json:"source_system"`
	TargetCode       string `json:"target_code"`
	TargetSystem     string `json:"target_system"`
	MapGroup         int    `json:"map_group"`
	MapPriority      int    `json:"map_priority"`
	MapRule          string `json:"map_rule,omitempty"`
	MapAdvice        string `json:"map_advice,omitempty"`
	CorrelationID    string `json:"correlation_id,omitempty"`
}
