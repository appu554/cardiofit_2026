// Package rules provides the rule translator for converting tables to computable rules.
//
// Phase 3b.5.5: Rule Translator Orchestrator
// Key Principle: Coordinate the full pipeline without LLM involvement:
//   ExtractedTable → NormalizedTable → Condition/Action → DraftRule
//
// This is the central orchestrator that transforms classified SPL tables
// into canonical, computable clinical decision rules with full provenance.
package rules

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/datasources/dailymed"
	"github.com/cardiofit/shared/extraction"
	"github.com/cardiofit/shared/types"
)

// =============================================================================
// RULE TRANSLATOR
// =============================================================================

// RuleTranslator orchestrates the table → rule pipeline
type RuleTranslator struct {
	tableNormalizer    *extraction.TableNormalizer
	conditionActionGen *extraction.ConditionActionGenerator
	fingerprintReg     FingerprintRegistry
	untranslatableQ    UntranslatableQueue
}

// FingerprintRegistry interface for semantic deduplication
type FingerprintRegistry interface {
	Exists(ctx context.Context, hash string) (bool, error)
	Register(ctx context.Context, rule types.FingerprintableRule) error
	GetRuleByFingerprint(ctx context.Context, hash string) (*uuid.UUID, error)
}

// UntranslatableQueue interface for human review queue
type UntranslatableQueue interface {
	Enqueue(ctx context.Context, entry *UntranslatableEntry) error
}

// NewRuleTranslator creates a configured translator
func NewRuleTranslator(fingerprintReg FingerprintRegistry, untranslatableQ UntranslatableQueue) *RuleTranslator {
	return &RuleTranslator{
		tableNormalizer:    extraction.NewTableNormalizer(),
		conditionActionGen: extraction.NewConditionActionGenerator(),
		fingerprintReg:     fingerprintReg,
		untranslatableQ:    untranslatableQ,
	}
}

// NewRuleTranslatorSimple creates a translator without external dependencies
// Useful for testing or standalone operation
func NewRuleTranslatorSimple() *RuleTranslator {
	return &RuleTranslator{
		tableNormalizer:    extraction.NewTableNormalizer(),
		conditionActionGen: extraction.NewConditionActionGenerator(),
	}
}

// =============================================================================
// TRANSLATION RESULT
// =============================================================================

// TranslationResult contains the outcome of rule translation
type TranslationResult struct {
	Rules               []*DraftRule          `json:"rules"`
	UntranslatableTables []UntranslatableEntry `json:"untranslatable_tables,omitempty"`
	Stats               TranslationStats      `json:"stats"`
	Errors              []TranslationError    `json:"errors,omitempty"`
}

// TranslationStats provides metrics on the translation
type TranslationStats struct {
	TablesProcessed      int     `json:"tables_processed"`
	TablesTranslated     int     `json:"tables_translated"`
	TablesUntranslatable int     `json:"tables_untranslatable"`
	TablesSkipped        int     `json:"tables_skipped"`
	RulesGenerated       int     `json:"rules_generated"`
	DuplicatesSkipped    int     `json:"duplicates_skipped"`
	RowsProcessed        int     `json:"rows_processed"`
	RowsSkipped          int     `json:"rows_skipped"`
	AverageConfidence    float64 `json:"average_confidence"`
	ProcessingTimeMs     int64   `json:"processing_time_ms"`
}

// TranslationError records an error during translation
type TranslationError struct {
	TableID string `json:"table_id"`
	Phase   string `json:"phase"`
	Error   string `json:"error"`
}

// =============================================================================
// MAIN TRANSLATION METHODS
// =============================================================================

// TranslateExtractedTables converts extracted tables to DraftRules
// This is the main entry point for the translation pipeline
func (t *RuleTranslator) TranslateExtractedTables(
	ctx context.Context,
	tables []*dailymed.ExtractedTable,
	provenance Provenance,
	domain string,
) (*TranslationResult, error) {
	startTime := time.Now()

	result := &TranslationResult{
		Stats: TranslationStats{},
	}

	var totalConfidence float64

	for _, table := range tables {
		result.Stats.TablesProcessed++

		// Skip non-clinical tables
		if t.shouldSkipTable(table) {
			result.Stats.TablesSkipped++
			continue
		}

		// Translate single table
		tableResult, err := t.translateSingleTable(ctx, table, provenance, domain)
		if err != nil {
			result.Errors = append(result.Errors, TranslationError{
				TableID: table.TableID,
				Phase:   "translation",
				Error:   err.Error(),
			})
			continue
		}

		// Merge results
		if tableResult.Translated {
			result.Stats.TablesTranslated++
			result.Rules = append(result.Rules, tableResult.Rules...)
			result.Stats.RulesGenerated += len(tableResult.Rules)
			result.Stats.DuplicatesSkipped += tableResult.DuplicatesSkipped
			result.Stats.RowsProcessed += tableResult.RowsProcessed
			result.Stats.RowsSkipped += tableResult.RowsSkipped
			totalConfidence += tableResult.TotalConfidence
		} else {
			result.Stats.TablesUntranslatable++
			result.UntranslatableTables = append(result.UntranslatableTables, *tableResult.UntranslatableEntry)

			// Queue for human review if queue is available
			if t.untranslatableQ != nil {
				if err := t.untranslatableQ.Enqueue(ctx, tableResult.UntranslatableEntry); err != nil {
					result.Errors = append(result.Errors, TranslationError{
						TableID: table.TableID,
						Phase:   "queue_untranslatable",
						Error:   err.Error(),
					})
				}
			}
		}
	}

	// Calculate average confidence
	if result.Stats.RulesGenerated > 0 {
		result.Stats.AverageConfidence = totalConfidence / float64(result.Stats.RulesGenerated)
	}

	result.Stats.ProcessingTimeMs = time.Since(startTime).Milliseconds()

	return result, nil
}

// =============================================================================
// SINGLE TABLE TRANSLATION
// =============================================================================

// singleTableResult contains the result of translating one table
type singleTableResult struct {
	Translated         bool
	Rules              []*DraftRule
	UntranslatableEntry *UntranslatableEntry
	DuplicatesSkipped  int
	RowsProcessed      int
	RowsSkipped        int
	TotalConfidence    float64
}

// translateSingleTable processes one table through the pipeline
func (t *RuleTranslator) translateSingleTable(
	ctx context.Context,
	table *dailymed.ExtractedTable,
	provenance Provenance,
	domain string,
) (*singleTableResult, error) {
	result := &singleTableResult{}

	// Step 1: Normalize the table
	normalized, err := t.tableNormalizer.Normalize(table.TableID, table.Headers, table.Rows)
	if err != nil {
		return nil, fmt.Errorf("normalizing table %s: %w", table.TableID, err)
	}

	// Step 2: Check if translatable
	if !normalized.Translatable {
		result.Translated = false
		result.UntranslatableEntry = &UntranslatableEntry{
			ID:               uuid.New(),
			TableID:          table.TableID,
			Headers:          table.Headers,
			RowCount:         len(table.Rows),
			Reason:           normalized.UntranslatableReason,
			SourceDocumentID: provenance.SourceDocumentID,
			SourceSectionID:  provenance.SourceSectionID,
			SourceInfo:       fmt.Sprintf("%s/%s", provenance.DocumentID, provenance.SectionCode),
			TableType:        string(table.TableType),
			Status:           StatusPending,
			SLADeadline:      time.Now().Add(72 * time.Hour),
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}
		return result, nil
	}

	result.Translated = true

	// Step 3: Generate condition/action pairs
	genResult, err := t.conditionActionGen.GenerateFromTable(normalized)
	if err != nil {
		return nil, fmt.Errorf("generating rules from table %s: %w", table.TableID, err)
	}

	result.RowsProcessed = genResult.TotalRows
	result.RowsSkipped = genResult.SkippedCount

	// Step 4: Convert to DraftRules with fingerprinting
	for _, generated := range genResult.Rules {
		// Create provenance with table-specific info
		ruleProvenance := provenance
		ruleProvenance.TableID = table.TableID
		ruleProvenance.EvidenceSpan = fmt.Sprintf("Row %d: %v", generated.RowIndex, generated.SourceCells)
		ruleProvenance.Confidence = generated.Confidence
		ruleProvenance.ExtractedAt = time.Now()

		// Create the draft rule
		draftRule := NewDraftRule(
			domain,
			t.mapTableTypeToRuleType(table.TableType),
			generated.Condition,
			generated.Action,
			ruleProvenance,
		)

		// Check for duplicates via fingerprint
		if t.fingerprintReg != nil {
			exists, err := t.fingerprintReg.Exists(ctx, draftRule.SemanticFingerprint.Hash)
			if err != nil {
				return nil, fmt.Errorf("checking fingerprint: %w", err)
			}
			if exists {
				result.DuplicatesSkipped++
				continue
			}

			// Register new fingerprint
			if err := t.fingerprintReg.Register(ctx, draftRule); err != nil {
				return nil, fmt.Errorf("registering fingerprint: %w", err)
			}
		}

		result.Rules = append(result.Rules, draftRule)
		result.TotalConfidence += generated.Confidence
	}

	return result, nil
}

// =============================================================================
// TABLE TYPE MAPPING
// =============================================================================

// shouldSkipTable determines if a table should be skipped
func (t *RuleTranslator) shouldSkipTable(table *dailymed.ExtractedTable) bool {
	// Skip unknown tables
	if table.TableType == dailymed.TableTypeUnknown {
		return true
	}

	// Skip adverse events tables (not rules)
	if table.TableType == dailymed.TableTypeAdverseEvents {
		return true
	}

	// Skip PK parameter tables (informational, not rules)
	if table.TableType == dailymed.TableTypePK {
		return true
	}

	return false
}

// mapTableTypeToRuleType converts table classification to rule type
func (t *RuleTranslator) mapTableTypeToRuleType(tableType dailymed.TableType) RuleType {
	switch tableType {
	case dailymed.TableTypeGFRDosing, dailymed.TableTypeHepaticDosing, dailymed.TableTypeDosing:
		return RuleTypeDosing
	case dailymed.TableTypeDDI:
		return RuleTypeInteraction
	case dailymed.TableTypeContraindications:
		return RuleTypeContraindication
	default:
		return RuleTypeDosing // Default
	}
}

// =============================================================================
// CONVENIENCE METHODS
// =============================================================================

// TranslateFromClassifier translates tables from the TableClassifier output
func (t *RuleTranslator) TranslateFromClassifier(
	ctx context.Context,
	section *dailymed.SPLSection,
	classifier *dailymed.TableClassifier,
	provenance Provenance,
	domain string,
) (*TranslationResult, error) {
	// Extract and classify tables
	extractedTables := classifier.ExtractAndClassifyTables(section)

	// Convert to pointer slice
	tables := make([]*dailymed.ExtractedTable, len(extractedTables))
	for i := range extractedTables {
		tables[i] = extractedTables[i]
	}

	return t.TranslateExtractedTables(ctx, tables, provenance, domain)
}

// TranslateSingleExtractedTable is a convenience method for translating one table
func (t *RuleTranslator) TranslateSingleExtractedTable(
	ctx context.Context,
	table *dailymed.ExtractedTable,
	sourceDocID uuid.UUID,
	sourceSectionID *uuid.UUID,
	sourceType string,
	documentID string,
	sectionCode string,
	domain string,
) (*TranslationResult, error) {
	provenance := Provenance{
		SourceDocumentID: sourceDocID,
		SourceSectionID:  sourceSectionID,
		SourceType:       sourceType,
		DocumentID:       documentID,
		SectionCode:      sectionCode,
		ExtractionMethod: "TABLE_PARSE",
	}

	return t.TranslateExtractedTables(ctx, []*dailymed.ExtractedTable{table}, provenance, domain)
}

// =============================================================================
// BATCH PROCESSING
// =============================================================================

// BatchTranslationRequest represents a batch of tables to translate
type BatchTranslationRequest struct {
	Tables     []*dailymed.ExtractedTable
	Provenance Provenance
	Domain     string
}

// TranslateBatch processes multiple translation requests
func (t *RuleTranslator) TranslateBatch(
	ctx context.Context,
	requests []BatchTranslationRequest,
) ([]*TranslationResult, error) {
	var results []*TranslationResult

	for _, req := range requests {
		result, err := t.TranslateExtractedTables(ctx, req.Tables, req.Provenance, req.Domain)
		if err != nil {
			return nil, fmt.Errorf("batch translation failed: %w", err)
		}
		results = append(results, result)
	}

	return results, nil
}

// =============================================================================
// STATISTICS AGGREGATION
// =============================================================================

// AggregateStats combines stats from multiple translation results
func AggregateStats(results []*TranslationResult) TranslationStats {
	var agg TranslationStats
	var totalConfidenceSum float64

	for _, r := range results {
		agg.TablesProcessed += r.Stats.TablesProcessed
		agg.TablesTranslated += r.Stats.TablesTranslated
		agg.TablesUntranslatable += r.Stats.TablesUntranslatable
		agg.TablesSkipped += r.Stats.TablesSkipped
		agg.RulesGenerated += r.Stats.RulesGenerated
		agg.DuplicatesSkipped += r.Stats.DuplicatesSkipped
		agg.RowsProcessed += r.Stats.RowsProcessed
		agg.RowsSkipped += r.Stats.RowsSkipped
		agg.ProcessingTimeMs += r.Stats.ProcessingTimeMs
		totalConfidenceSum += r.Stats.AverageConfidence * float64(r.Stats.RulesGenerated)
	}

	if agg.RulesGenerated > 0 {
		agg.AverageConfidence = totalConfidenceSum / float64(agg.RulesGenerated)
	}

	return agg
}
