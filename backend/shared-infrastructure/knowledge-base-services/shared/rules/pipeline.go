// Package rules provides the canonical rule generation pipeline.
//
// Phase 3b.5.9: Pipeline Integration
// Key Principle: Wire all Phase 3b.5 components into a unified pipeline
// that connects DailyMed SPL extraction → Table Classification → Rule Generation.
//
// Pipeline: SPLDocument → ClassifiedTables → NormalizedTables → DraftRules → Governance
//
// This is the main orchestrator for converting FDA drug labels into computable rules.
package rules

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/datasources/dailymed"
	"github.com/cardiofit/shared/governance/fingerprint_registry"
)

// =============================================================================
// SPL TYPE HELPERS
// =============================================================================

// splIDToUUID converts an SPLID to a deterministic UUID using SHA1 namespace
func splIDToUUID(id dailymed.SPLID) uuid.UUID {
	// Use a namespace UUID for SPL documents
	namespace := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8") // URL namespace
	return uuid.NewSHA1(namespace, []byte(id.Root+":"+id.Extension))
}

// =============================================================================
// CANONICAL RULE PIPELINE
// =============================================================================

// CanonicalRulePipeline orchestrates the full Phase 3b.5 extraction pipeline
// SPL → Tables → Classifications → Normalized → DraftRules → Fingerprint → Governance
type CanonicalRulePipeline struct {
	// Core components
	fetcher         SPLFetcher
	classifier      TableClassifierInterface
	translator      *RuleTranslator
	fingerprintReg  *fingerprint_registry.Registry
	untranslatableQ *PostgresQueue

	// Configuration
	config PipelineConfig

	// Metrics
	metrics *PipelineMetrics
	mu      sync.RWMutex
}

// SPLFetcher interface for fetching SPL documents
type SPLFetcher interface {
	FetchBySetID(ctx context.Context, setID string) (*dailymed.SPLDocument, error)
	FetchByNDC(ctx context.Context, ndc string) (*dailymed.SPLDocument, error)
}

// TableClassifierInterface abstracts table classification
// Matches the actual dailymed.TableClassifier implementation
type TableClassifierInterface interface {
	// ExtractAndClassifyTables extracts all tables from a section with classification
	ExtractAndClassifyTables(section *dailymed.SPLSection) []*dailymed.ExtractedTable
	// ClassifyTable classifies a single SPL table and returns classification result
	ClassifyTable(table *dailymed.SPLTable) *dailymed.ClassificationResult
}

// PipelineConfig contains pipeline configuration
type PipelineConfig struct {
	// Processing options
	MaxConcurrent    int           `json:"max_concurrent"`
	BatchSize        int           `json:"batch_size"`
	ProcessingTimeout time.Duration `json:"processing_timeout"`

	// Domain routing
	DefaultDomain string            `json:"default_domain"`
	SectionToDomain map[string]string `json:"section_to_domain"` // LOINC → KB domain

	// Quality thresholds
	MinConfidence     float64 `json:"min_confidence"`
	RequireFingerprint bool    `json:"require_fingerprint"`

	// Error handling
	ContinueOnError bool `json:"continue_on_error"`
	MaxErrors       int  `json:"max_errors"`
}

// DefaultPipelineConfig returns sensible defaults
func DefaultPipelineConfig() PipelineConfig {
	return PipelineConfig{
		MaxConcurrent:     5,
		BatchSize:         100,
		ProcessingTimeout: 5 * time.Minute,
		DefaultDomain:     "KB-1",
		SectionToDomain: map[string]string{
			"34068-7": "KB-1",  // Dosage & Administration → KB-1 Drug Rules
			"34070-3": "KB-4",  // Contraindications → KB-4 Patient Safety
			"34073-7": "KB-5",  // Drug Interactions → KB-5 Interactions
			"43685-7": "KB-4",  // Warnings & Precautions → KB-4 Safety
			"34066-1": "KB-4",  // Boxed Warning → KB-4 Safety
		},
		MinConfidence:     0.7,
		RequireFingerprint: true,
		ContinueOnError:   true,
		MaxErrors:         10,
	}
}

// =============================================================================
// PIPELINE CONSTRUCTION
// =============================================================================

// NewCanonicalRulePipeline creates a fully configured pipeline
func NewCanonicalRulePipeline(db *sql.DB, fetcher SPLFetcher, config PipelineConfig) *CanonicalRulePipeline {
	// Initialize fingerprint registry
	fingerprintReg := fingerprint_registry.NewRegistry(db)

	// Initialize untranslatable queue
	untranslatableQ := NewPostgresQueue(db)

	// Create rule translator with dependencies
	translator := NewRuleTranslator(fingerprintReg, untranslatableQ)

	return &CanonicalRulePipeline{
		fetcher:         fetcher,
		classifier:      dailymed.NewTableClassifier(),
		translator:      translator,
		fingerprintReg:  fingerprintReg,
		untranslatableQ: untranslatableQ,
		config:          config,
		metrics:         &PipelineMetrics{},
	}
}

// NewCanonicalRulePipelineSimple creates a pipeline without database dependencies
// Useful for testing or standalone operation
func NewCanonicalRulePipelineSimple(fetcher SPLFetcher) *CanonicalRulePipeline {
	return &CanonicalRulePipeline{
		fetcher:    fetcher,
		classifier: dailymed.NewTableClassifier(),
		translator: NewRuleTranslatorSimple(),
		config:     DefaultPipelineConfig(),
		metrics:    &PipelineMetrics{},
	}
}

// =============================================================================
// PIPELINE RESULT
// =============================================================================

// PipelineResult contains the complete outcome of pipeline processing
type PipelineResult struct {
	// Input tracking
	SetID       string    `json:"set_id"`
	DocumentID  string    `json:"document_id"` // SPLID extension value
	ProcessedAt time.Time `json:"processed_at"`

	// Extracted rules
	Rules []*DraftRule `json:"rules"`

	// Untranslatable tables (sent to human review)
	UntranslatableTables []UntranslatableEntry `json:"untranslatable_tables,omitempty"`

	// Statistics
	Stats PipelineStats `json:"stats"`

	// Errors (if ContinueOnError is true)
	Errors []PipelineError `json:"errors,omitempty"`

	// Processing metadata
	ProcessingTimeMs int64 `json:"processing_time_ms"`
}

// PipelineStats aggregates pipeline statistics
type PipelineStats struct {
	SectionsProcessed     int     `json:"sections_processed"`
	TablesExtracted       int     `json:"tables_extracted"`
	TablesClassified      int     `json:"tables_classified"`
	TablesTranslated      int     `json:"tables_translated"`
	TablesUntranslatable  int     `json:"tables_untranslatable"`
	TablesSkipped         int     `json:"tables_skipped"`
	RulesGenerated        int     `json:"rules_generated"`
	RulesDeduplicated     int     `json:"rules_deduplicated"`
	AverageConfidence     float64 `json:"average_confidence"`
}

// PipelineError records an error during pipeline processing
type PipelineError struct {
	Phase   string `json:"phase"`
	Section string `json:"section,omitempty"`
	TableID string `json:"table_id,omitempty"`
	Error   string `json:"error"`
}

// =============================================================================
// MAIN PROCESSING METHODS
// =============================================================================

// ProcessDocument runs the full pipeline on a single SPL document
func (p *CanonicalRulePipeline) ProcessDocument(ctx context.Context, setID string) (*PipelineResult, error) {
	startTime := time.Now()

	result := &PipelineResult{
		SetID:       setID,
		ProcessedAt: startTime,
	}

	// Step 1: Fetch SPL document
	doc, err := p.fetcher.FetchBySetID(ctx, setID)
	if err != nil {
		return nil, fmt.Errorf("fetching SPL document %s: %w", setID, err)
	}

	result.DocumentID = doc.ID.Extension

	// Step 2: Process each section
	errorCount := 0
	for _, section := range doc.Sections {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		// Determine target domain from LOINC section code
		domain := p.getDomainForSection(section.Code.Code)

		// Process section
		sectionResult, err := p.processSection(ctx, doc, &section, domain)
		if err != nil {
			if p.config.ContinueOnError && errorCount < p.config.MaxErrors {
				result.Errors = append(result.Errors, PipelineError{
					Phase:   "section_processing",
					Section: section.Code.Code,
					Error:   err.Error(),
				})
				errorCount++
				continue
			}
			return result, fmt.Errorf("processing section %s: %w", section.Code, err)
		}

		// Merge results
		result.Stats.SectionsProcessed++
		result.Rules = append(result.Rules, sectionResult.Rules...)
		result.UntranslatableTables = append(result.UntranslatableTables, sectionResult.UntranslatableTables...)

		// Accumulate stats
		result.Stats.TablesExtracted += sectionResult.Stats.TablesProcessed
		result.Stats.TablesTranslated += sectionResult.Stats.TablesTranslated
		result.Stats.TablesUntranslatable += sectionResult.Stats.TablesUntranslatable
		result.Stats.TablesSkipped += sectionResult.Stats.TablesSkipped
		result.Stats.RulesGenerated += sectionResult.Stats.RulesGenerated
		result.Stats.RulesDeduplicated += sectionResult.Stats.DuplicatesSkipped
	}

	// Calculate average confidence
	if result.Stats.RulesGenerated > 0 {
		var totalConfidence float64
		for _, rule := range result.Rules {
			totalConfidence += rule.Provenance.Confidence
		}
		result.Stats.AverageConfidence = totalConfidence / float64(result.Stats.RulesGenerated)
	}

	result.ProcessingTimeMs = time.Since(startTime).Milliseconds()

	// Update metrics
	p.updateMetrics(result)

	return result, nil
}

// ProcessBatch processes multiple SPL documents
func (p *CanonicalRulePipeline) ProcessBatch(ctx context.Context, setIDs []string) ([]*PipelineResult, error) {
	results := make([]*PipelineResult, 0, len(setIDs))

	// Process in batches
	for i := 0; i < len(setIDs); i += p.config.BatchSize {
		end := i + p.config.BatchSize
		if end > len(setIDs) {
			end = len(setIDs)
		}

		batch := setIDs[i:end]

		// Process batch concurrently
		batchResults, err := p.processBatchConcurrent(ctx, batch)
		if err != nil {
			return results, fmt.Errorf("processing batch %d-%d: %w", i, end, err)
		}

		results = append(results, batchResults...)
	}

	return results, nil
}

// processBatchConcurrent processes a batch with controlled concurrency
func (p *CanonicalRulePipeline) processBatchConcurrent(ctx context.Context, setIDs []string) ([]*PipelineResult, error) {
	results := make([]*PipelineResult, len(setIDs))
	errors := make([]error, len(setIDs))

	sem := make(chan struct{}, p.config.MaxConcurrent)
	var wg sync.WaitGroup

	for i, setID := range setIDs {
		wg.Add(1)
		go func(idx int, id string) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			// Create timeout context
			procCtx, cancel := context.WithTimeout(ctx, p.config.ProcessingTimeout)
			defer cancel()

			// Process document
			result, err := p.ProcessDocument(procCtx, id)
			results[idx] = result
			errors[idx] = err
		}(i, setID)
	}

	wg.Wait()

	// Collect first error
	for _, err := range errors {
		if err != nil && !p.config.ContinueOnError {
			return results, err
		}
	}

	return results, nil
}

// =============================================================================
// SECTION PROCESSING
// =============================================================================

// processSection extracts and translates tables from a single section
func (p *CanonicalRulePipeline) processSection(
	ctx context.Context,
	doc *dailymed.SPLDocument,
	section *dailymed.SPLSection,
	domain string,
) (*TranslationResult, error) {
	// Step 1: Extract and classify tables
	tables := p.classifier.ExtractAndClassifyTables(section)

	// Convert to pointer slice
	tablePtrs := make([]*dailymed.ExtractedTable, len(tables))
	for i := range tables {
		tablePtrs[i] = tables[i]
	}

	// Step 2: Build provenance (convert SPL types to expected types)
	sectionUUID := splIDToUUID(section.ID)
	provenance := Provenance{
		SourceDocumentID: splIDToUUID(doc.ID),
		SourceSectionID:  &sectionUUID,
		SourceType:       "FDA_SPL",
		DocumentID:       doc.SetID.Extension,
		SectionCode:      section.Code.Code,
		SectionName:      section.Code.DisplayName,
		ExtractionMethod: "TABLE_PARSE",
		ExtractedAt:      time.Now(),
	}

	// Step 3: Translate tables to rules
	result, err := p.translator.TranslateExtractedTables(ctx, tablePtrs, provenance, domain)
	if err != nil {
		return nil, fmt.Errorf("translating tables: %w", err)
	}

	return result, nil
}

// getDomainForSection maps LOINC section codes to KB domains
func (p *CanonicalRulePipeline) getDomainForSection(loincCode string) string {
	if domain, exists := p.config.SectionToDomain[loincCode]; exists {
		return domain
	}
	return p.config.DefaultDomain
}

// =============================================================================
// METRICS
// =============================================================================

// PipelineMetrics tracks cumulative pipeline statistics
type PipelineMetrics struct {
	DocumentsProcessed    int64     `json:"documents_processed"`
	TotalRulesGenerated   int64     `json:"total_rules_generated"`
	TotalDeduplicated     int64     `json:"total_deduplicated"`
	TotalUntranslatable   int64     `json:"total_untranslatable"`
	TotalErrors           int64     `json:"total_errors"`
	AverageProcessingMs   float64   `json:"average_processing_ms"`
	LastProcessedAt       time.Time `json:"last_processed_at"`

	mu sync.Mutex
}

// updateMetrics updates cumulative metrics
func (p *CanonicalRulePipeline) updateMetrics(result *PipelineResult) {
	p.metrics.mu.Lock()
	defer p.metrics.mu.Unlock()

	p.metrics.DocumentsProcessed++
	p.metrics.TotalRulesGenerated += int64(result.Stats.RulesGenerated)
	p.metrics.TotalDeduplicated += int64(result.Stats.RulesDeduplicated)
	p.metrics.TotalUntranslatable += int64(result.Stats.TablesUntranslatable)
	p.metrics.TotalErrors += int64(len(result.Errors))

	// Running average for processing time
	n := float64(p.metrics.DocumentsProcessed)
	p.metrics.AverageProcessingMs = ((n-1)*p.metrics.AverageProcessingMs + float64(result.ProcessingTimeMs)) / n

	p.metrics.LastProcessedAt = time.Now()
}

// GetMetrics returns current pipeline metrics
func (p *CanonicalRulePipeline) GetMetrics() PipelineMetrics {
	p.metrics.mu.Lock()
	defer p.metrics.mu.Unlock()

	return *p.metrics
}

// ResetMetrics resets cumulative metrics
func (p *CanonicalRulePipeline) ResetMetrics() {
	p.metrics.mu.Lock()
	defer p.metrics.mu.Unlock()

	p.metrics = &PipelineMetrics{}
}

// =============================================================================
// CONVENIENCE METHODS
// =============================================================================

// ProcessByNDC processes an SPL document found by NDC code
func (p *CanonicalRulePipeline) ProcessByNDC(ctx context.Context, ndc string) (*PipelineResult, error) {
	doc, err := p.fetcher.FetchByNDC(ctx, ndc)
	if err != nil {
		return nil, fmt.Errorf("fetching SPL by NDC %s: %w", ndc, err)
	}

	return p.ProcessDocument(ctx, doc.SetID.Extension)
}

// GetFingerprintRegistry returns the fingerprint registry for direct access
func (p *CanonicalRulePipeline) GetFingerprintRegistry() *fingerprint_registry.Registry {
	return p.fingerprintReg
}

// GetUntranslatableQueue returns the untranslatable queue for direct access
func (p *CanonicalRulePipeline) GetUntranslatableQueue() *PostgresQueue {
	return p.untranslatableQ
}

// =============================================================================
// BATCH RESULT AGGREGATION
// =============================================================================

// AggregateBatchResults combines results from multiple documents
func AggregateBatchResults(results []*PipelineResult) *BatchAggregation {
	agg := &BatchAggregation{
		ProcessedAt: time.Now(),
	}

	for _, r := range results {
		if r == nil {
			continue
		}

		agg.DocumentsProcessed++
		agg.TotalRules += len(r.Rules)
		agg.TotalUntranslatable += len(r.UntranslatableTables)
		agg.TotalErrors += len(r.Errors)
		agg.TotalProcessingMs += r.ProcessingTimeMs

		agg.Stats.SectionsProcessed += r.Stats.SectionsProcessed
		agg.Stats.TablesExtracted += r.Stats.TablesExtracted
		agg.Stats.TablesTranslated += r.Stats.TablesTranslated
		agg.Stats.TablesUntranslatable += r.Stats.TablesUntranslatable
		agg.Stats.TablesSkipped += r.Stats.TablesSkipped
		agg.Stats.RulesGenerated += r.Stats.RulesGenerated
		agg.Stats.RulesDeduplicated += r.Stats.RulesDeduplicated
	}

	if agg.DocumentsProcessed > 0 {
		agg.AverageProcessingMs = float64(agg.TotalProcessingMs) / float64(agg.DocumentsProcessed)
	}

	return agg
}

// BatchAggregation holds aggregated batch processing results
type BatchAggregation struct {
	ProcessedAt         time.Time     `json:"processed_at"`
	DocumentsProcessed  int           `json:"documents_processed"`
	TotalRules          int           `json:"total_rules"`
	TotalUntranslatable int           `json:"total_untranslatable"`
	TotalErrors         int           `json:"total_errors"`
	TotalProcessingMs   int64         `json:"total_processing_ms"`
	AverageProcessingMs float64       `json:"average_processing_ms"`
	Stats               PipelineStats `json:"stats"`
}
