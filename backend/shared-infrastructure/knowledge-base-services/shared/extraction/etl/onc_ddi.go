// Package etl provides Extract-Transform-Load utilities for structured data sources.
// These loaders handle data that requires NO LLM - deterministic ETL only.
//
// DESIGN PRINCIPLE: "DDI ≠ NLP problem. Interaction pairs come from structured sources."
// This loader processes the ONC High-Priority Drug Interactions dataset.
package etl

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/cardiofit/shared/factstore"
	"github.com/sirupsen/logrus"
)

// =============================================================================
// ONC HIGH-PRIORITY DDI DATASET
// =============================================================================
// Source: ONC (Office of the National Coordinator for Health IT)
// URL: https://www.healthit.gov/topic/safety/high-priority-drug-drug-interaction-clinical-decision-support
// Coverage: ~1,200 high-risk drug pairs curated by federal agencies
// Format: CSV/JSON structured data with RxNorm CUIs

// ONCInteraction represents a single interaction from the ONC dataset
type ONCInteraction struct {
	// Drug 1
	Drug1RxCUI string `json:"drug_1_rxcui" csv:"Drug1_RXCUI"`
	Drug1Name  string `json:"drug_1_name" csv:"Drug1_Name"`

	// Drug 2
	Drug2RxCUI string `json:"drug_2_rxcui" csv:"Drug2_RXCUI"`
	Drug2Name  string `json:"drug_2_name" csv:"Drug2_Name"`

	// Interaction Details
	Severity       ONCDDISeverity `json:"severity" csv:"Severity"`
	ClinicalEffect string         `json:"clinical_effect" csv:"Clinical_Effect"`
	Management     string         `json:"management" csv:"Management"`

	// Evidence
	EvidenceLevel  string `json:"evidence_level" csv:"Evidence_Level"`
	Documentation  string `json:"documentation" csv:"Documentation"`
	ClinicalSource string `json:"clinical_source" csv:"Clinical_Source"`

	// Metadata
	ONCPairID   string `json:"onc_pair_id" csv:"ONC_Pair_ID"`
	LastUpdated string `json:"last_updated" csv:"Last_Updated"`
	Source      string `json:"source"` // Always "ONC_HIGH_PRIORITY"

	// Flags
	Authoritative bool `json:"authoritative"` // Always true for ONC
}

// ONCDDISeverity represents the severity level from ONC
type ONCDDISeverity string

const (
	ONCDDISeverityContraindicated ONCDDISeverity = "CONTRAINDICATED"
	ONCDDISeverityHigh            ONCDDISeverity = "HIGH"
	ONCDDISeverityModerate        ONCDDISeverity = "MODERATE"
)

// MapToFactStoreSeverity converts ONC severity to fact store severity
func (s ONCDDISeverity) MapToFactStoreSeverity() string {
	switch s {
	case ONCDDISeverityContraindicated:
		return "CRITICAL"
	case ONCDDISeverityHigh:
		return "MAJOR"
	case ONCDDISeverityModerate:
		return "MODERATE"
	default:
		return "MODERATE"
	}
}

// =============================================================================
// ONC DDI LOADER
// =============================================================================

// ONCDDILoader loads the ONC High-Priority DDI Dataset
type ONCDDILoader struct {
	factStore  FactStoreWriter
	drugMaster DrugMasterLookup
	log        *logrus.Entry
	config     ONCLoaderConfig
}

// ONCLoaderConfig holds configuration for the ONC loader
type ONCLoaderConfig struct {
	// Source URL or file path
	SourcePath string `json:"sourcePath"`

	// Whether to auto-activate (recommended for ONC as it's authoritative)
	AutoActivate bool `json:"autoActivate"`

	// Batch size for processing
	BatchSize int `json:"batchSize"`

	// Whether to validate RxCUIs against drug master
	ValidateRxCUIs bool `json:"validateRxCUIs"`

	// Skip invalid pairs instead of failing
	SkipInvalid bool `json:"skipInvalid"`
}

// DefaultONCLoaderConfig returns sensible defaults
func DefaultONCLoaderConfig() ONCLoaderConfig {
	return ONCLoaderConfig{
		AutoActivate:   true,  // ONC is authoritative
		BatchSize:      100,
		ValidateRxCUIs: true,
		SkipInvalid:    true,
	}
}

// FactStoreWriter interface for fact persistence
type FactStoreWriter interface {
	SaveFact(ctx context.Context, fact *factstore.Fact) error
	SaveFacts(ctx context.Context, facts []*factstore.Fact) error
}

// DrugMasterLookup interface for drug validation
type DrugMasterLookup interface {
	GetByRxCUI(ctx context.Context, rxcui string) (*DrugInfo, error)
	ExistsRxCUI(ctx context.Context, rxcui string) (bool, error)
}

// DrugInfo minimal drug info for validation
type DrugInfo struct {
	RxCUI    string `json:"rxcui"`
	DrugName string `json:"drugName"`
	Status   string `json:"status"`
}

// NewONCDDILoader creates a new ONC DDI loader
func NewONCDDILoader(factStore FactStoreWriter, drugMaster DrugMasterLookup, config ONCLoaderConfig, log *logrus.Entry) *ONCDDILoader {
	if log == nil {
		log = logrus.NewEntry(logrus.StandardLogger())
	}
	return &ONCDDILoader{
		factStore:  factStore,
		drugMaster: drugMaster,
		config:     config,
		log:        log.WithField("loader", "ONC_DDI"),
	}
}

// LoadResult captures the outcome of a load operation
type LoadResult struct {
	TotalParsed    int           `json:"totalParsed"`
	FactsCreated   int           `json:"factsCreated"`
	FactsSkipped   int           `json:"factsSkipped"`
	ValidationErrs int           `json:"validationErrors"`
	Errors         []string      `json:"errors,omitempty"`
	Duration       time.Duration `json:"duration"`
	Source         string        `json:"source"`

	// ─────────────────────────────────────────────────────────────────────────
	// ENHANCED METRICS (Review Refinement)
	// ─────────────────────────────────────────────────────────────────────────

	// ConflictsFound counts evidence conflicts detected during load
	ConflictsFound int `json:"conflictsFound,omitempty"`

	// DuplicatesSkipped counts duplicate entries skipped
	DuplicatesSkipped int `json:"duplicatesSkipped,omitempty"`

	// DatasetMetadata captures version information for audit trail
	DatasetMetadata *DatasetMetadata `json:"datasetMetadata,omitempty"`

	// SourceFile captures source file checksums for audit trail
	SourceFile *SourceFileMetadata `json:"sourceFile,omitempty"`
}

// DatasetMetadata captures version and provenance information for datasets
// This helps answer: "Which version of the data was this fact derived from?"
type DatasetMetadata struct {
	// Version is the dataset version (e.g., "2024-Q4", "v2.1")
	Version string `json:"version"`

	// ReleaseDate is when this version was released by the source
	ReleaseDate time.Time `json:"releaseDate"`

	// RecordCount is the total number of records in the dataset
	RecordCount int `json:"recordCount"`

	// SourceOrganization identifies who published this dataset
	SourceOrganization string `json:"sourceOrganization"`

	// DownloadURL is where this dataset was obtained
	DownloadURL string `json:"downloadUrl,omitempty"`

	// DownloadedAt is when we downloaded/retrieved this dataset
	DownloadedAt time.Time `json:"downloadedAt"`

	// Notes contains any release notes or changelog info
	Notes string `json:"notes,omitempty"`
}

// SourceFileMetadata captures source file information for audit trail
// This enables answering: "Which exact file version produced this fact?"
type SourceFileMetadata struct {
	// FilePath is the path to the source file
	FilePath string `json:"filePath"`

	// FileSize is the size in bytes
	FileSize int64 `json:"fileSize"`

	// SHA256 is the SHA-256 checksum of the file
	SHA256 string `json:"sha256"`

	// MD5 is the MD5 checksum (for backward compatibility)
	MD5 string `json:"md5,omitempty"`

	// LastModified is the file's last modification time
	LastModified time.Time `json:"lastModified"`

	// DownloadURL is where this file was obtained (if applicable)
	DownloadURL string `json:"downloadUrl,omitempty"`

	// DownloadedAt is when we downloaded this file
	DownloadedAt time.Time `json:"downloadedAt,omitempty"`
}

// Load processes the ONC DDI dataset from the configured source
func (l *ONCDDILoader) Load(ctx context.Context) (*LoadResult, error) {
	start := time.Now()
	result := &LoadResult{
		Source: "ONC_HIGH_PRIORITY",
	}

	l.log.WithField("source", l.config.SourcePath).Info("Starting ONC DDI load")

	// Parse interactions from source
	interactions, err := l.parseSource(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ONC source: %w", err)
	}

	result.TotalParsed = len(interactions)
	l.log.WithField("count", result.TotalParsed).Info("Parsed ONC interactions")

	// Convert to facts in batches
	var factsToSave []*factstore.Fact

	for _, interaction := range interactions {
		fact, err := l.convertToFact(ctx, &interaction)
		if err != nil {
			result.ValidationErrs++
			if l.config.SkipInvalid {
				result.Errors = append(result.Errors, fmt.Sprintf("%s-%s: %v",
					interaction.Drug1RxCUI, interaction.Drug2RxCUI, err))
				continue
			}
			return result, fmt.Errorf("failed to convert interaction: %w", err)
		}

		factsToSave = append(factsToSave, fact)

		// Batch save
		if len(factsToSave) >= l.config.BatchSize {
			if err := l.saveBatch(ctx, factsToSave); err != nil {
				return result, fmt.Errorf("failed to save batch: %w", err)
			}
			result.FactsCreated += len(factsToSave)
			factsToSave = nil
		}
	}

	// Save remaining
	if len(factsToSave) > 0 {
		if err := l.saveBatch(ctx, factsToSave); err != nil {
			return result, fmt.Errorf("failed to save final batch: %w", err)
		}
		result.FactsCreated += len(factsToSave)
	}

	result.FactsSkipped = result.TotalParsed - result.FactsCreated
	result.Duration = time.Since(start)

	l.log.WithFields(logrus.Fields{
		"created":    result.FactsCreated,
		"skipped":    result.FactsSkipped,
		"validation": result.ValidationErrs,
		"duration":   result.Duration,
	}).Info("ONC DDI load complete")

	return result, nil
}

// parseSource reads interactions from file or URL
func (l *ONCDDILoader) parseSource(ctx context.Context) ([]ONCInteraction, error) {
	source := l.config.SourcePath

	// Determine source type
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		return l.parseFromURL(ctx, source)
	}

	// File-based
	if strings.HasSuffix(source, ".json") {
		return l.parseFromJSONFile(source)
	}

	return l.parseFromCSVFile(source)
}

// parseFromURL downloads and parses from URL
func (l *ONCDDILoader) parseFromURL(ctx context.Context, url string) ([]ONCInteraction, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	// Detect content type
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "json") {
		return l.parseJSONReader(resp.Body)
	}

	return l.parseCSVReader(resp.Body)
}

// parseFromJSONFile reads from a JSON file
func (l *ONCDDILoader) parseFromJSONFile(path string) ([]ONCInteraction, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	return l.parseJSONReader(file)
}

// parseFromCSVFile reads from a CSV file
func (l *ONCDDILoader) parseFromCSVFile(path string) ([]ONCInteraction, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	return l.parseCSVReader(file)
}

// parseJSONReader parses JSON format
func (l *ONCDDILoader) parseJSONReader(r io.Reader) ([]ONCInteraction, error) {
	var interactions []ONCInteraction
	decoder := json.NewDecoder(r)
	if err := decoder.Decode(&interactions); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	// Mark all as ONC source and authoritative
	for i := range interactions {
		interactions[i].Source = "ONC_HIGH_PRIORITY"
		interactions[i].Authoritative = true
	}

	return interactions, nil
}

// parseCSVReader parses CSV format
func (l *ONCDDILoader) parseCSVReader(r io.Reader) ([]ONCInteraction, error) {
	reader := csv.NewReader(bufio.NewReader(r))
	reader.FieldsPerRecord = -1 // Allow variable fields

	// Read header
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV header: %w", err)
	}

	// Build column index map
	colIndex := make(map[string]int)
	for i, col := range header {
		colIndex[strings.TrimSpace(col)] = i
	}

	var interactions []ONCInteraction

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			l.log.WithError(err).Warn("Error reading CSV record, skipping")
			continue
		}

		interaction := l.recordToInteraction(record, colIndex)
		interactions = append(interactions, interaction)
	}

	return interactions, nil
}

// recordToInteraction converts a CSV record to ONCInteraction
func (l *ONCDDILoader) recordToInteraction(record []string, colIndex map[string]int) ONCInteraction {
	getField := func(name string) string {
		if idx, ok := colIndex[name]; ok && idx < len(record) {
			return strings.TrimSpace(record[idx])
		}
		return ""
	}

	return ONCInteraction{
		Drug1RxCUI:     getField("Drug1_RXCUI"),
		Drug1Name:      getField("Drug1_Name"),
		Drug2RxCUI:     getField("Drug2_RXCUI"),
		Drug2Name:      getField("Drug2_Name"),
		Severity:       ONCDDISeverity(getField("Severity")),
		ClinicalEffect: getField("Clinical_Effect"),
		Management:     getField("Management"),
		EvidenceLevel:  getField("Evidence_Level"),
		Documentation:  getField("Documentation"),
		ClinicalSource: getField("Clinical_Source"),
		ONCPairID:      getField("ONC_Pair_ID"),
		LastUpdated:    getField("Last_Updated"),
		Source:         "ONC_HIGH_PRIORITY",
		Authoritative:  true,
	}
}

// convertToFact converts an ONC interaction to a Fact
func (l *ONCDDILoader) convertToFact(ctx context.Context, interaction *ONCInteraction) (*factstore.Fact, error) {
	// Validate RxCUIs if configured
	if l.config.ValidateRxCUIs && l.drugMaster != nil {
		if exists, _ := l.drugMaster.ExistsRxCUI(ctx, interaction.Drug1RxCUI); !exists {
			return nil, fmt.Errorf("drug1 RxCUI not found: %s", interaction.Drug1RxCUI)
		}
		if exists, _ := l.drugMaster.ExistsRxCUI(ctx, interaction.Drug2RxCUI); !exists {
			return nil, fmt.Errorf("drug2 RxCUI not found: %s", interaction.Drug2RxCUI)
		}
	}

	// Create fact
	fact := factstore.NewFact(
		factstore.FactTypeInteraction,
		interaction.Drug1RxCUI, // Primary drug
		interaction.Drug1Name,
	)

	// Set interaction content
	content := factstore.InteractionContent{
		InteractionType:  "DRUG_DRUG",
		InteractantRxCUI: interaction.Drug2RxCUI,
		InteractantName:  interaction.Drug2Name,
		Severity:         interaction.Severity.MapToFactStoreSeverity(),
		ClinicalEffect:   interaction.ClinicalEffect,
		Management:       interaction.Management,
		EvidenceLevel:    "HIGH", // ONC is high-evidence
		Source:           "ONC_HIGH_PRIORITY",
	}

	if err := fact.SetContent(content); err != nil {
		return nil, fmt.Errorf("failed to set content: %w", err)
	}

	// Set confidence - ONC is authoritative, so HIGH confidence
	fact.Confidence = factstore.FactConfidence{
		Overall:           0.95,
		SourceQuality:     1.0, // Government source
		ExtractionCertainty: 1.0, // Structured data, no extraction uncertainty
		Band:              factstore.ConfidenceHigh,
		SourceDiversity:   1,
		HumanVerified:     true, // ONC data is human-curated
	}

	// Set stability - DDI data is relatively stable
	fact.Stability = factstore.FactStability{
		Volatility:           factstore.VolatilityStable,
		ExpectedHalfLife:     factstore.HalfLifeMonths,
		AutoRevalidationDays: 180, // Revalidate every 6 months
		ChangeTriggers:       []string{"ONC_DATASET_UPDATE", "FDA_SAFETY_ALERT"},
		LastRefreshedAt:      time.Now(),
		RefreshSource:        "ONC_HIGH_PRIORITY_ETL",
	}

	// Auto-activate if configured (recommended for ONC)
	if l.config.AutoActivate {
		fact.Status = factstore.StatusActive
	}

	// Set provenance
	fact.ExtractorID = "ONC_DDI_ETL"
	fact.ExtractorVersion = "1.0"
	fact.SourceURL = "https://www.healthit.gov/topic/safety/high-priority-drug-drug-interaction"
	fact.SourceVersion = interaction.LastUpdated
	fact.RegulatoryBody = "ONC/HHS"

	return fact, nil
}

// saveBatch saves a batch of facts
func (l *ONCDDILoader) saveBatch(ctx context.Context, facts []*factstore.Fact) error {
	if l.factStore == nil {
		l.log.WithField("count", len(facts)).Debug("Dry run - would save facts")
		return nil
	}
	return l.factStore.SaveFacts(ctx, facts)
}

// =============================================================================
// REVERSE INTERACTION CREATION
// =============================================================================
// DDIs are bidirectional - Drug A interacts with Drug B means Drug B interacts with Drug A

// LoadWithReverse loads ONC data and creates bidirectional interaction facts
func (l *ONCDDILoader) LoadWithReverse(ctx context.Context) (*LoadResult, error) {
	start := time.Now()
	result := &LoadResult{
		Source: "ONC_HIGH_PRIORITY",
	}

	l.log.WithField("source", l.config.SourcePath).Info("Starting ONC DDI load with reverse pairs")

	// Parse interactions
	interactions, err := l.parseSource(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ONC source: %w", err)
	}

	result.TotalParsed = len(interactions) * 2 // Forward + reverse
	l.log.WithField("count", len(interactions)).Info("Parsed ONC interactions, will create bidirectional facts")

	var factsToSave []*factstore.Fact

	for _, interaction := range interactions {
		// Forward direction (Drug1 -> Drug2)
		forwardFact, err := l.convertToFact(ctx, &interaction)
		if err != nil {
			result.ValidationErrs++
			if l.config.SkipInvalid {
				continue
			}
			return result, err
		}
		factsToSave = append(factsToSave, forwardFact)

		// Reverse direction (Drug2 -> Drug1)
		reverseInteraction := ONCInteraction{
			Drug1RxCUI:     interaction.Drug2RxCUI,
			Drug1Name:      interaction.Drug2Name,
			Drug2RxCUI:     interaction.Drug1RxCUI,
			Drug2Name:      interaction.Drug1Name,
			Severity:       interaction.Severity,
			ClinicalEffect: interaction.ClinicalEffect,
			Management:     interaction.Management,
			EvidenceLevel:  interaction.EvidenceLevel,
			Documentation:  interaction.Documentation,
			ClinicalSource: interaction.ClinicalSource,
			ONCPairID:      interaction.ONCPairID + "_REV",
			LastUpdated:    interaction.LastUpdated,
			Source:         "ONC_HIGH_PRIORITY",
			Authoritative:  true,
		}

		reverseFact, err := l.convertToFact(ctx, &reverseInteraction)
		if err != nil {
			result.ValidationErrs++
			if l.config.SkipInvalid {
				continue
			}
			return result, err
		}
		factsToSave = append(factsToSave, reverseFact)

		// Batch save
		if len(factsToSave) >= l.config.BatchSize {
			if err := l.saveBatch(ctx, factsToSave); err != nil {
				return result, err
			}
			result.FactsCreated += len(factsToSave)
			factsToSave = nil
		}
	}

	// Save remaining
	if len(factsToSave) > 0 {
		if err := l.saveBatch(ctx, factsToSave); err != nil {
			return result, err
		}
		result.FactsCreated += len(factsToSave)
	}

	result.FactsSkipped = result.TotalParsed - result.FactsCreated
	result.Duration = time.Since(start)

	l.log.WithFields(logrus.Fields{
		"created":        result.FactsCreated,
		"bidirectional":  result.FactsCreated / 2,
		"skipped":        result.FactsSkipped,
		"duration":       result.Duration,
	}).Info("ONC DDI load with reverse pairs complete")

	return result, nil
}
