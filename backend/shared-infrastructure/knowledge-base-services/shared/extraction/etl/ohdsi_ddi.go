// Package etl provides Extract-Transform-Load pipelines for clinical data sources.
// This file implements the OHDSI Athena DDI ETL for coverage expansion.
//
// DESIGN PRINCIPLE: "DDI ≠ NLP problem"
// OHDSI Athena provides structured vocabulary data including drug interactions.
// This is pure ETL from standardized vocabulary files - NO LLM involved.
//
// ROLE: Coverage Expansion
// ONC High-Priority covers ~100 critical pairs. OHDSI Athena provides broader
// coverage from standardized clinical vocabularies (RxNorm, ATC, etc.)
package etl

import (
	"bufio"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cardiofit/shared/evidence"
)

// =============================================================================
// OHDSI ATHENA DATA STRUCTURES
// =============================================================================

// AthenaInteractionType categorizes OHDSI drug interaction relationships
type AthenaInteractionType string

const (
	// AthenaInteracts is the standard "Interacts with" relationship
	AthenaInteracts AthenaInteractionType = "interacts_with"

	// AthenaContraindicated is "Is contraindicated with" relationship
	AthenaContraindicated AthenaInteractionType = "contraindicated"

	// AthenaInhibits is "Inhibits" metabolic relationship
	AthenaInhibits AthenaInteractionType = "inhibits"

	// AthenaInduces is "Induces" metabolic relationship
	AthenaInduces AthenaInteractionType = "induces"

	// AthenaSubstrate is "Is substrate of" relationship
	AthenaSubstrate AthenaInteractionType = "substrate_of"
)

// AthenaSeverity maps OHDSI severity classifications
type AthenaSeverity string

const (
	AthenaSeverityMajor    AthenaSeverity = "MAJOR"
	AthenaSeverityModerate AthenaSeverity = "MODERATE"
	AthenaSeverityMinor    AthenaSeverity = "MINOR"
	AthenaSeverityUnknown  AthenaSeverity = "UNKNOWN"
)

// AthenaConcept represents an OHDSI concept from the CONCEPT table
type AthenaConcept struct {
	// ConceptID is the OHDSI concept identifier
	ConceptID int64 `json:"concept_id"`

	// ConceptName is the human-readable name
	ConceptName string `json:"concept_name"`

	// DomainID identifies the domain (Drug, Condition, etc.)
	DomainID string `json:"domain_id"`

	// VocabularyID identifies the source vocabulary (RxNorm, ATC, etc.)
	VocabularyID string `json:"vocabulary_id"`

	// ConceptClassID is the concept class (Ingredient, Clinical Drug, etc.)
	ConceptClassID string `json:"concept_class_id"`

	// StandardConcept indicates if this is a standard concept (S, C, or null)
	StandardConcept string `json:"standard_concept"`

	// ConceptCode is the code in the source vocabulary (e.g., RxCUI)
	ConceptCode string `json:"concept_code"`

	// ValidStartDate is when the concept became valid
	ValidStartDate time.Time `json:"valid_start_date"`

	// ValidEndDate is when the concept expires
	ValidEndDate time.Time `json:"valid_end_date"`

	// InvalidReason indicates why concept is invalid (if applicable)
	InvalidReason string `json:"invalid_reason,omitempty"`
}

// AthenaRelationship represents a relationship from CONCEPT_RELATIONSHIP table
type AthenaRelationship struct {
	// Concept1ID is the source concept
	Concept1ID int64 `json:"concept_id_1"`

	// Concept2ID is the target concept
	Concept2ID int64 `json:"concept_id_2"`

	// RelationshipID identifies the relationship type
	RelationshipID string `json:"relationship_id"`

	// ValidStartDate is when the relationship became valid
	ValidStartDate time.Time `json:"valid_start_date"`

	// ValidEndDate is when the relationship expires
	ValidEndDate time.Time `json:"valid_end_date"`

	// InvalidReason indicates why relationship is invalid (if applicable)
	InvalidReason string `json:"invalid_reason,omitempty"`
}

// AthenaInteraction is the processed drug interaction from OHDSI data
type AthenaInteraction struct {
	// Drug1ConceptID is the OHDSI concept ID for drug 1
	Drug1ConceptID int64 `json:"drug_1_concept_id"`

	// Drug1RxCUI is the RxNorm CUI for drug 1 (if available)
	Drug1RxCUI string `json:"drug_1_rxcui,omitempty"`

	// Drug1Name is the drug name
	Drug1Name string `json:"drug_1_name"`

	// Drug2ConceptID is the OHDSI concept ID for drug 2
	Drug2ConceptID int64 `json:"drug_2_concept_id"`

	// Drug2RxCUI is the RxNorm CUI for drug 2 (if available)
	Drug2RxCUI string `json:"drug_2_rxcui,omitempty"`

	// Drug2Name is the drug name
	Drug2Name string `json:"drug_2_name"`

	// InteractionType classifies the interaction
	InteractionType AthenaInteractionType `json:"interaction_type"`

	// Severity is the classified severity level
	Severity AthenaSeverity `json:"severity"`

	// RelationshipID is the original OHDSI relationship
	RelationshipID string `json:"relationship_id"`

	// Source vocabulary information
	SourceVocabulary string `json:"source_vocabulary"`

	// Authoritative indicates if from authoritative source
	Authoritative bool `json:"authoritative"`
}

// =============================================================================
// OHDSI DDI LOADER
// =============================================================================

// OHDSIDDILoaderConfig configures the OHDSI DDI loader
type OHDSIDDILoaderConfig struct {
	// ConceptFilePath is the path to CONCEPT.csv from Athena download
	ConceptFilePath string

	// RelationshipFilePath is the path to CONCEPT_RELATIONSHIP.csv
	RelationshipFilePath string

	// DrugStrengthFilePath is optional path to DRUG_STRENGTH.csv for dosing info
	DrugStrengthFilePath string

	// OnlyStandardConcepts filters to standard concepts only
	OnlyStandardConcepts bool

	// IncludeExpired includes expired/invalid concepts
	IncludeExpired bool

	// InteractionRelationships lists relationship_ids to consider as interactions
	InteractionRelationships []string

	// MinimumValidDate filters concepts valid after this date
	MinimumValidDate time.Time
}

// DefaultInteractionRelationships returns the default OHDSI relationship IDs for drug interactions
func DefaultInteractionRelationships() []string {
	return []string{
		"Interacts with",           // Standard drug-drug interaction
		"Is contraindicated with",  // Contraindication relationship
		"Inhibits",                 // Metabolic inhibition
		"Induces",                  // Metabolic induction
		"Is substrate of",          // CYP450 substrate relationships
		"Has pharmacokinetic with", // PK interactions
		"Has pharmacodynamic with", // PD interactions
	}
}

// OHDSIDDILoader loads drug interaction data from OHDSI Athena vocabulary files
type OHDSIDDILoader struct {
	config OHDSIDDILoaderConfig

	// ─────────────────────────────────────────────────────────────────────────
	// CONCURRENCY SAFETY (Review Refinement)
	// ─────────────────────────────────────────────────────────────────────────
	// Mutex protects concurrent access to the index maps
	// Use RLock for read operations, Lock for write operations
	mu sync.RWMutex

	// Loaded concept index (concept_id -> concept)
	concepts map[int64]*AthenaConcept

	// RxNorm concept index (rxcui -> concept_id)
	rxcuiIndex map[string]int64

	// Drug domain concepts only
	drugConcepts map[int64]*AthenaConcept
}

// NewOHDSIDDILoader creates a new OHDSI DDI loader
func NewOHDSIDDILoader(config OHDSIDDILoaderConfig) *OHDSIDDILoader {
	if len(config.InteractionRelationships) == 0 {
		config.InteractionRelationships = DefaultInteractionRelationships()
	}
	return &OHDSIDDILoader{
		config:       config,
		concepts:     make(map[int64]*AthenaConcept),
		rxcuiIndex:   make(map[string]int64),
		drugConcepts: make(map[int64]*AthenaConcept),
	}
}

// OHDSILoadResult contains the results of loading OHDSI DDI data
type OHDSILoadResult struct {
	// Interactions loaded
	Interactions []*AthenaInteraction

	// Statistics
	TotalConceptsLoaded   int
	DrugConceptsLoaded    int
	RelationshipsLoaded   int
	InteractionsExtracted int
	RxCUIMappings         int

	// Processing metadata
	LoadedAt     time.Time
	LoadDuration time.Duration

	// Errors encountered (non-fatal)
	Warnings []string
}

// Load performs the full ETL: load concepts, build index, extract interactions
func (l *OHDSIDDILoader) Load(ctx context.Context) (*OHDSILoadResult, error) {
	startTime := time.Now()
	result := &OHDSILoadResult{
		Interactions: make([]*AthenaInteraction, 0),
		Warnings:     make([]string, 0),
		LoadedAt:     startTime,
	}

	// Step 1: Load concepts
	if err := l.loadConcepts(ctx, result); err != nil {
		return nil, fmt.Errorf("failed to load concepts: %w", err)
	}

	// Step 2: Build RxCUI index
	l.buildRxCUIIndex(result)

	// Step 3: Load relationships and extract interactions
	if err := l.loadRelationships(ctx, result); err != nil {
		return nil, fmt.Errorf("failed to load relationships: %w", err)
	}

	result.LoadDuration = time.Since(startTime)
	return result, nil
}

// loadConcepts loads the CONCEPT.csv file
func (l *OHDSIDDILoader) loadConcepts(ctx context.Context, result *OHDSILoadResult) error {
	file, err := os.Open(l.config.ConceptFilePath)
	if err != nil {
		return fmt.Errorf("failed to open concept file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(bufio.NewReader(file))
	reader.Comma = '\t' // Athena files are tab-separated
	reader.LazyQuotes = true

	// Read header
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("failed to read header: %w", err)
	}

	// Build column index
	colIndex := make(map[string]int)
	for i, col := range header {
		colIndex[strings.ToLower(col)] = i
	}

	// Required columns
	requiredCols := []string{"concept_id", "concept_name", "domain_id", "vocabulary_id", "concept_class_id", "concept_code"}
	for _, col := range requiredCols {
		if _, ok := colIndex[col]; !ok {
			return fmt.Errorf("missing required column: %s", col)
		}
	}

	// Read concepts
	lineNum := 1
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("line %d: %v", lineNum, err))
			lineNum++
			continue
		}

		concept, err := l.parseConcept(record, colIndex)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("line %d: %v", lineNum, err))
			lineNum++
			continue
		}

		// Apply filters
		if l.config.OnlyStandardConcepts && concept.StandardConcept != "S" {
			lineNum++
			continue
		}

		if !l.config.IncludeExpired && concept.InvalidReason != "" {
			lineNum++
			continue
		}

		if !l.config.MinimumValidDate.IsZero() && concept.ValidEndDate.Before(l.config.MinimumValidDate) {
			lineNum++
			continue
		}

		// Store concept
		l.concepts[concept.ConceptID] = concept
		result.TotalConceptsLoaded++

		// Index drug domain concepts
		if concept.DomainID == "Drug" {
			l.drugConcepts[concept.ConceptID] = concept
			result.DrugConceptsLoaded++
		}

		lineNum++
	}

	return nil
}

// parseConcept parses a CSV record into an AthenaConcept
func (l *OHDSIDDILoader) parseConcept(record []string, colIndex map[string]int) (*AthenaConcept, error) {
	conceptID, err := strconv.ParseInt(record[colIndex["concept_id"]], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid concept_id: %v", err)
	}

	concept := &AthenaConcept{
		ConceptID:      conceptID,
		ConceptName:    record[colIndex["concept_name"]],
		DomainID:       record[colIndex["domain_id"]],
		VocabularyID:   record[colIndex["vocabulary_id"]],
		ConceptClassID: record[colIndex["concept_class_id"]],
		ConceptCode:    record[colIndex["concept_code"]],
	}

	// Optional columns
	if idx, ok := colIndex["standard_concept"]; ok && idx < len(record) {
		concept.StandardConcept = record[idx]
	}

	if idx, ok := colIndex["invalid_reason"]; ok && idx < len(record) {
		concept.InvalidReason = record[idx]
	}

	// Parse dates
	if idx, ok := colIndex["valid_start_date"]; ok && idx < len(record) {
		if t, err := time.Parse("2006-01-02", record[idx]); err == nil {
			concept.ValidStartDate = t
		}
	}

	if idx, ok := colIndex["valid_end_date"]; ok && idx < len(record) {
		if t, err := time.Parse("2006-01-02", record[idx]); err == nil {
			concept.ValidEndDate = t
		}
	}

	return concept, nil
}

// buildRxCUIIndex creates an index from RxCUI to concept_id for RxNorm concepts
func (l *OHDSIDDILoader) buildRxCUIIndex(result *OHDSILoadResult) {
	for conceptID, concept := range l.drugConcepts {
		// RxNorm concepts have vocabulary_id = "RxNorm" and concept_code = RxCUI
		if concept.VocabularyID == "RxNorm" && concept.ConceptCode != "" {
			l.rxcuiIndex[concept.ConceptCode] = conceptID
			result.RxCUIMappings++
		}
	}
}

// loadRelationships loads CONCEPT_RELATIONSHIP.csv and extracts interactions
func (l *OHDSIDDILoader) loadRelationships(ctx context.Context, result *OHDSILoadResult) error {
	file, err := os.Open(l.config.RelationshipFilePath)
	if err != nil {
		return fmt.Errorf("failed to open relationship file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(bufio.NewReader(file))
	reader.Comma = '\t'
	reader.LazyQuotes = true

	// Read header
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("failed to read header: %w", err)
	}

	// Build column index
	colIndex := make(map[string]int)
	for i, col := range header {
		colIndex[strings.ToLower(col)] = i
	}

	// Build relationship set for fast lookup
	interactionRels := make(map[string]bool)
	for _, rel := range l.config.InteractionRelationships {
		interactionRels[rel] = true
	}

	// Read relationships
	lineNum := 1
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			lineNum++
			continue
		}

		result.RelationshipsLoaded++

		// Parse relationship
		rel, err := l.parseRelationship(record, colIndex)
		if err != nil {
			lineNum++
			continue
		}

		// Check if this is an interaction relationship
		if !interactionRels[rel.RelationshipID] {
			lineNum++
			continue
		}

		// Both concepts must be drugs
		drug1, ok1 := l.drugConcepts[rel.Concept1ID]
		drug2, ok2 := l.drugConcepts[rel.Concept2ID]
		if !ok1 || !ok2 {
			lineNum++
			continue
		}

		// Create interaction record
		interaction := l.createInteraction(drug1, drug2, rel)
		result.Interactions = append(result.Interactions, interaction)
		result.InteractionsExtracted++

		lineNum++
	}

	return nil
}

// parseRelationship parses a CSV record into an AthenaRelationship
func (l *OHDSIDDILoader) parseRelationship(record []string, colIndex map[string]int) (*AthenaRelationship, error) {
	concept1ID, err := strconv.ParseInt(record[colIndex["concept_id_1"]], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid concept_id_1: %v", err)
	}

	concept2ID, err := strconv.ParseInt(record[colIndex["concept_id_2"]], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid concept_id_2: %v", err)
	}

	rel := &AthenaRelationship{
		Concept1ID:     concept1ID,
		Concept2ID:     concept2ID,
		RelationshipID: record[colIndex["relationship_id"]],
	}

	// Optional columns
	if idx, ok := colIndex["invalid_reason"]; ok && idx < len(record) {
		rel.InvalidReason = record[idx]
	}

	// Parse dates
	if idx, ok := colIndex["valid_start_date"]; ok && idx < len(record) {
		if t, err := time.Parse("2006-01-02", record[idx]); err == nil {
			rel.ValidStartDate = t
		}
	}

	if idx, ok := colIndex["valid_end_date"]; ok && idx < len(record) {
		if t, err := time.Parse("2006-01-02", record[idx]); err == nil {
			rel.ValidEndDate = t
		}
	}

	return rel, nil
}

// createInteraction creates an AthenaInteraction from two concepts and their relationship
func (l *OHDSIDDILoader) createInteraction(drug1, drug2 *AthenaConcept, rel *AthenaRelationship) *AthenaInteraction {
	interaction := &AthenaInteraction{
		Drug1ConceptID:   drug1.ConceptID,
		Drug1Name:        drug1.ConceptName,
		Drug2ConceptID:   drug2.ConceptID,
		Drug2Name:        drug2.ConceptName,
		InteractionType:  l.mapRelationshipToType(rel.RelationshipID),
		Severity:         AthenaSeverityUnknown, // OHDSI doesn't have severity in standard vocab
		RelationshipID:   rel.RelationshipID,
		SourceVocabulary: drug1.VocabularyID,
		Authoritative:    l.isAuthoritativeSource(drug1.VocabularyID),
	}

	// Add RxCUI if available
	if drug1.VocabularyID == "RxNorm" {
		interaction.Drug1RxCUI = drug1.ConceptCode
	}
	if drug2.VocabularyID == "RxNorm" {
		interaction.Drug2RxCUI = drug2.ConceptCode
	}

	return interaction
}

// mapRelationshipToType maps OHDSI relationship_id to our interaction type
func (l *OHDSIDDILoader) mapRelationshipToType(relationshipID string) AthenaInteractionType {
	switch relationshipID {
	case "Interacts with":
		return AthenaInteracts
	case "Is contraindicated with":
		return AthenaContraindicated
	case "Inhibits":
		return AthenaInhibits
	case "Induces":
		return AthenaInduces
	case "Is substrate of":
		return AthenaSubstrate
	default:
		return AthenaInteracts
	}
}

// isAuthoritativeSource determines if a vocabulary is considered authoritative
func (l *OHDSIDDILoader) isAuthoritativeSource(vocabularyID string) bool {
	authoritative := map[string]bool{
		"RxNorm":     true,
		"RxNorm Extension": true,
		"ATC":        true,
		"NDC":        true,
		"SNOMED":     true,
	}
	return authoritative[vocabularyID]
}

// =============================================================================
// EVIDENCE UNIT CONVERSION
// =============================================================================

// ToEvidenceUnits converts OHDSI interactions to EvidenceUnits for the Evidence Router
func (l *OHDSIDDILoader) ToEvidenceUnits(interactions []*AthenaInteraction) []*evidence.EvidenceUnit {
	units := make([]*evidence.EvidenceUnit, 0, len(interactions))

	for _, interaction := range interactions {
		unit := evidence.NewEvidenceUnit(evidence.SourceTypeAPI, "https://athena.ohdsi.org")
		unit.EvidenceID = fmt.Sprintf("OHDSI-%d-%d-%s",
			interaction.Drug1ConceptID,
			interaction.Drug2ConceptID,
			interaction.RelationshipID)

		// Set drug reference (prefer drug 1 as primary)
		if interaction.Drug1RxCUI != "" {
			unit.RxCUI = interaction.Drug1RxCUI
		}
		unit.DrugName = interaction.Drug1Name

		// Set clinical domains
		unit.AddClinicalDomain(evidence.DomainInteraction)
		unit.AddClinicalDomain(evidence.DomainSafety)

		// Target KB-5 (Drug Interactions)
		unit.AddKBTarget("KB-5")

		// Set priority based on source
		if interaction.Authoritative {
			unit.Priority = 3
		} else {
			unit.Priority = 5
		}

		// Store interaction data in parsed content
		unit.ParsedContent = interaction
		unit.ContentType = "application/json"

		// Set provenance
		unit.SourceVersion = "athena-v5" // OHDSI vocabulary version
		unit.Jurisdiction = "US"
		unit.RegulatoryBody = "OHDSI"

		// Set quality signals
		unit.ConfidenceFloor = 0.70 // OHDSI is standardized but community-maintained
		if interaction.Authoritative {
			unit.QualityScore = 0.85
		} else {
			unit.QualityScore = 0.75
		}

		// Add metadata
		unit.SourceMetadata = map[string]string{
			"drug_2_rxcui":       interaction.Drug2RxCUI,
			"drug_2_name":        interaction.Drug2Name,
			"interaction_type":   string(interaction.InteractionType),
			"relationship_id":    interaction.RelationshipID,
			"source_vocabulary":  interaction.SourceVocabulary,
			"drug_1_concept_id":  strconv.FormatInt(interaction.Drug1ConceptID, 10),
			"drug_2_concept_id":  strconv.FormatInt(interaction.Drug2ConceptID, 10),
		}

		units = append(units, unit)
	}

	return units
}

// =============================================================================
// LOOKUP METHODS
// =============================================================================

// GetConceptByRxCUI looks up an OHDSI concept by RxCUI (thread-safe)
func (l *OHDSIDDILoader) GetConceptByRxCUI(rxcui string) *AthenaConcept {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if conceptID, ok := l.rxcuiIndex[rxcui]; ok {
		return l.concepts[conceptID]
	}
	return nil
}

// GetConceptByID looks up an OHDSI concept by concept_id (thread-safe)
func (l *OHDSIDDILoader) GetConceptByID(conceptID int64) *AthenaConcept {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.concepts[conceptID]
}

// GetInteractionsForDrug returns all interactions for a given drug
func (l *OHDSIDDILoader) GetInteractionsForDrug(rxcui string, interactions []*AthenaInteraction) []*AthenaInteraction {
	result := make([]*AthenaInteraction, 0)
	for _, interaction := range interactions {
		if interaction.Drug1RxCUI == rxcui || interaction.Drug2RxCUI == rxcui {
			result = append(result, interaction)
		}
	}
	return result
}

// HasInteraction checks if two drugs have an interaction
func (l *OHDSIDDILoader) HasInteraction(rxcui1, rxcui2 string, interactions []*AthenaInteraction) *AthenaInteraction {
	for _, interaction := range interactions {
		if (interaction.Drug1RxCUI == rxcui1 && interaction.Drug2RxCUI == rxcui2) ||
			(interaction.Drug1RxCUI == rxcui2 && interaction.Drug2RxCUI == rxcui1) {
			return interaction
		}
	}
	return nil
}

// =============================================================================
// STATISTICS AND REPORTING
// =============================================================================

// GetStatistics returns statistics about loaded OHDSI data
func (l *OHDSIDDILoader) GetStatistics() map[string]interface{} {
	// Count by vocabulary
	vocabCounts := make(map[string]int)
	for _, concept := range l.drugConcepts {
		vocabCounts[concept.VocabularyID]++
	}

	// Count by concept class
	classCounts := make(map[string]int)
	for _, concept := range l.drugConcepts {
		classCounts[concept.ConceptClassID]++
	}

	return map[string]interface{}{
		"total_concepts":     len(l.concepts),
		"drug_concepts":      len(l.drugConcepts),
		"rxcui_mappings":     len(l.rxcuiIndex),
		"vocab_distribution": vocabCounts,
		"class_distribution": classCounts,
	}
}
