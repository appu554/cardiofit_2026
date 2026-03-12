// Package factstore provides the Integration Pipeline connecting LAYER 4, 5, and 6.
//
// This implements the complete flow:
// SPL Extraction → Derived Facts → Governance → Activated Facts
//
// DESIGN PRINCIPLE: "Parse once, extract to multiple KBs"
// - Source documents are parsed once
// - Facts are extracted to multiple target KBs
// - Governance engine auto-approves/rejects based on confidence
// - Human escalation for edge cases
//
// Phase 3 Implementation - Integration Pipeline
package factstore

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/sirupsen/logrus"

	"github.com/cardiofit/shared/datasources"
	"github.com/cardiofit/shared/datasources/dailymed"
	"github.com/cardiofit/shared/datasources/kdigo"
	"github.com/cardiofit/shared/governance/routing"
	"github.com/cardiofit/shared/terminology"
)

// =============================================================================
// JSON SANITIZATION - Fix invalid JSON characters for PostgreSQL
// =============================================================================

// sanitizeForJSON removes characters that cause PostgreSQL JSON parse errors:
// - NULL bytes (\x00)
// - Control characters (except \t, \n, \r)
// - Invalid UTF-8 sequences
func sanitizeForJSON(s string) string {
	// Ensure valid UTF-8
	if !utf8.ValidString(s) {
		s = strings.ToValidUTF8(s, "")
	}

	// Remove NULL bytes and other control characters
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		// Allow tab, newline, carriage return; skip other control characters
		if r == '\t' || r == '\n' || r == '\r' || r >= 0x20 {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// sanitizeExtractedTables creates a sanitized copy of tables for JSON storage
func sanitizeExtractedTables(tables []*dailymed.ExtractedTable) []*dailymed.ExtractedTable {
	if len(tables) == 0 {
		return tables
	}

	sanitized := make([]*dailymed.ExtractedTable, len(tables))
	for i, t := range tables {
		// Create a copy with sanitized strings
		sanitized[i] = &dailymed.ExtractedTable{
			TableID:       t.TableID,
			TableType:     t.TableType,
			Confidence:    t.Confidence,
			Caption:       sanitizeForJSON(t.Caption),
			TargetKBs:     t.TargetKBs,
			Priority:      t.Priority,
			SourceSection: sanitizeForJSON(t.SourceSection),
			SourceLOINC:   t.SourceLOINC,
			RawHTML:       sanitizeForJSON(t.RawHTML),
		}

		// Sanitize headers
		if len(t.Headers) > 0 {
			sanitized[i].Headers = make([]string, len(t.Headers))
			for j, h := range t.Headers {
				sanitized[i].Headers[j] = sanitizeForJSON(h)
			}
		}

		// Sanitize rows
		if len(t.Rows) > 0 {
			sanitized[i].Rows = make([][]string, len(t.Rows))
			for j, row := range t.Rows {
				sanitized[i].Rows[j] = make([]string, len(row))
				for k, cell := range row {
					sanitized[i].Rows[j][k] = sanitizeForJSON(cell)
				}
			}
		}
	}
	return sanitized
}

// =============================================================================
// PIPELINE CONFIGURATION
// =============================================================================

// PipelineConfig configures the integration pipeline
type PipelineConfig struct {
	// Processing settings
	BatchSize           int           `json:"batchSize"`
	ProcessingInterval  time.Duration `json:"processingInterval"`
	MaxConcurrentDocs   int           `json:"maxConcurrentDocs"`

	// Governance thresholds
	AutoApproveThreshold float64 `json:"autoApproveThreshold"` // 2.0 = DISABLED (all facts → PENDING_REVIEW for pharmacist review)
	ReviewThreshold      float64 `json:"reviewThreshold"`      // ≥0.65

	// LLM consensus settings
	LLMConsensusRequired int     `json:"llmConsensusRequired"` // 2-of-3 minimum
	LLMMinConfidence     float64 `json:"llmMinConfidence"`     // 0.70 minimum

	// Escalation settings
	EscalateCriticalSafety bool `json:"escalateCriticalSafety"`
	EscalateNoConsensus    bool `json:"escalateNoConsensus"`

	// LLM Fallback (Phase 3c)
	LLMBudgetUSD      float64 `json:"llmBudgetUsd"`      // per-run budget cap (default $50)
	LLMConfidenceCeil float64 `json:"llmConfidenceCeil"`  // hard ceiling (default 0.75)
}

// DefaultPipelineConfig returns sensible defaults
func DefaultPipelineConfig() PipelineConfig {
	return PipelineConfig{
		BatchSize:              50,
		ProcessingInterval:     5 * time.Minute,
		MaxConcurrentDocs:      5,
		AutoApproveThreshold:   2.0, // DISABLED — all facts route to PENDING_REVIEW for pharmacist review. No auto-approval.
		ReviewThreshold:        0.65,
		LLMConsensusRequired:   2,
		LLMMinConfidence:       0.70,
		EscalateCriticalSafety: true,
		EscalateNoConsensus:    true,
	}
}

// =============================================================================
// INTEGRATION PIPELINE
// =============================================================================

// Pipeline manages the integration flow from extraction to governance
type Pipeline struct {
	mu     sync.RWMutex
	config PipelineConfig
	log    *logrus.Entry

	// Components
	repo            *Repository
	sectionRouter   *dailymed.SectionRouter
	authorityRouter *routing.AuthorityRouter

	// Terminology normalization (Phase 3 - Issue 1 FK Fix)
	// Validates and corrects RxCUIs before storage to prevent FK constraint failures.
	// Example: SPL says Lithium=5521 (wrong), normalizer corrects to 6448 (correct).
	drugNormalizer terminology.DrugNormalizer

	// Phase 3 Issues 2+3 Fix: MedDRA adverse event normalization.
	// Replaces regex-based noise filtering with deterministic MedDRA dictionary lookup.
	// Provides FAERS-compatible MedDRA PT codes for pharmacovigilance integration.
	aeNormalizer terminology.AdverseEventNormalizer

	// Phase 3c: LLM fallback for prose-only adverse reaction sections.
	// When table parsing produces zero SAFETY_SIGNAL facts, Claude extracts AEs from prose.
	// All LLM facts are capped at 0.75 confidence and set to PENDING_REVIEW.
	llmProvider *llmFallbackProvider // nil = LLM fallback disabled
	llmBudget   *llmBudgetTracker   // per-run spend + rate limiter

	// Phase 3d: KDIGO organ impairment enrichment (CPIC removed — it's pharmacogenomics, not OI)
	kdigoClient            *kdigo.Client // nil = KDIGO enrichment disabled
	includeOrganImpairment bool

	// P2: DDI grammar extractor for deterministic interaction extraction from prose.
	// Runs on LOINC 34073-7 (Drug Interactions) PlainText before LLM fallback.
	ddiGrammar *DDIGrammar

	// Fix 3: MedDRA prose scanner — finds ALL MedDRA terms in free text.
	// Runs AFTER table parsing and DDI grammar, BEFORE LLM fallback.
	// This is the SCANNER (finds terms in prose) vs the aeNormalizer VALIDATOR (checks known terms).
	proseScanner *MedDRAProseScanner // nil = prose scanning disabled (no MedDRA loaded)

	// Running state
	running      bool
	stopChan     chan struct{}
	seenFactKeys map[string]bool // Dedup: tracks canonical keys seen in this pipeline run

	// P3: Transient skip reason tracking — populated by extractFromTables, read by completeness checker.
	// Reset at the start of each ProcessSPLDocument call.
	currentSkipReasons map[string]int

	// Metrics
	metrics PipelineMetrics
}

// PipelineMetrics tracks pipeline performance
type PipelineMetrics struct {
	DocumentsProcessed  int64         `json:"documentsProcessed"`
	SectionsProcessed   int64         `json:"sectionsProcessed"`
	FactsExtracted      int64         `json:"factsExtracted"`
	FactsAutoApproved   int64         `json:"factsAutoApproved"`
	FactsQueued         int64         `json:"factsQueued"`
	FactsRejected       int64         `json:"factsRejected"`
	EscalationsCreated  int64         `json:"escalationsCreated"`
	AuthorityLookups    int64         `json:"authorityLookups"`
	LLMExtractions      int64         `json:"llmExtractions"`
	GrammarExtractions  int64         `json:"grammarExtractions"`
	AverageConfidence   float64       `json:"averageConfidence"`
	LastProcessedAt     time.Time     `json:"lastProcessedAt"`
	ProcessingDuration  time.Duration `json:"processingDuration"`
}

// NewPipeline creates a new integration pipeline.
//
// The drugNormalizer parameter is optional (can be nil for backward compatibility).
// When provided, it validates and corrects RxCUIs before storage, fixing FK constraint
// failures where SPL documents contain wrong/outdated RxCUI values.
//
// The aeNormalizer parameter is optional (can be nil for backward compatibility).
// When provided, it replaces regex-based noise filtering with MedDRA dictionary lookup
// and adds FAERS-compatible MedDRA PT codes to safety signal facts.
func NewPipeline(
	config PipelineConfig,
	repo *Repository,
	sectionRouter *dailymed.SectionRouter,
	authorityRouter *routing.AuthorityRouter,
	log *logrus.Entry,
	drugNormalizer terminology.DrugNormalizer,
	aeNormalizer terminology.AdverseEventNormalizer,
	proseScanner *MedDRAProseScanner,
) *Pipeline {
	return &Pipeline{
		config:          config,
		repo:            repo,
		sectionRouter:   sectionRouter,
		authorityRouter: authorityRouter,
		drugNormalizer:  drugNormalizer,
		aeNormalizer:    aeNormalizer,
		proseScanner:    proseScanner,
		ddiGrammar:      NewDDIGrammar(),
		log:             log.WithField("component", "factstore-pipeline"),
		stopChan:        make(chan struct{}),
	}
}

// =============================================================================
// DOCUMENT PROCESSING
// =============================================================================

// ProcessSPLDocument processes a complete SPL document through the pipeline.
// The knownDrugName parameter provides the validated drug name from Phase B scope selection,
// used as a fallback when extractDrugNameFromTitle fails on garbage SPL titles.
func (p *Pipeline) ProcessSPLDocument(ctx context.Context, splDoc *dailymed.SPLDocument, rxcui string, knownDrugName string) (*ProcessingResult, error) {
	startTime := time.Now()
	result := &ProcessingResult{
		StartedAt: startTime,
	}

	p.log.WithFields(logrus.Fields{
		"setId":    splDoc.SetID.Root,
		"drugName": splDoc.Title,
		"rxcui":    rxcui,
	}).Info("Processing SPL document")

	// Step 1: Create source document record
	sourceDoc, err := p.createSourceDocument(ctx, splDoc, rxcui, knownDrugName)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to create source document: %v", err)
		return result, err
	}
	result.SourceDocumentID = sourceDoc.ID

	// Step 2: Route and process sections
	routedSections := p.sectionRouter.RouteDocument(splDoc)
	result.TotalSections = len(routedSections)

	p.log.WithField("sections", len(routedSections)).Debug("Routed sections")

	// P3: Collect all extracted facts for completeness checking at the end
	var allExtractedFacts []*DerivedFact
	var totalSourceRows int
	var processedSectionCodes []string
	p.currentSkipReasons = make(map[string]int) // Reset per document

	// Step 3: Process each routed section
	for _, routed := range routedSections {
		if len(routed.TargetKBs) == 0 {
			continue // Skip sections with no target KBs
		}

		// Create source section record
		sourceSection, err := p.createSourceSection(ctx, sourceDoc.ID, routed)
		if err != nil {
			p.log.WithError(err).WithField("sectionCode", routed.Section.Code.Code).Warn("Failed to create section")
			result.Errors = append(result.Errors, err.Error())
			continue
		}
		result.SectionsProcessed++

		// Step 4: Extract facts based on routing decision
		facts, err := p.extractFactsFromSection(ctx, sourceDoc, sourceSection, routed, rxcui)
		if err != nil {
			p.log.WithError(err).WithField("sectionCode", routed.Section.Code.Code).Warn("Failed to extract facts")
			result.Errors = append(result.Errors, err.Error())
			continue
		}

		// P3: Track source table rows and section codes for completeness report
		for _, tbl := range routed.ExtractedTables {
			totalSourceRows += len(tbl.Rows)
		}
		processedSectionCodes = append(processedSectionCodes, routed.Section.Code.Code)

		// Step 5: Store derived facts and process governance
		for _, fact := range facts {
			// P6.1: Assign volatility contract — enriches FactData with stability metadata.
			enrichFactWithStability(fact)

			// P6.2: VERSION CHAIN — check if an active fact with the same canonical
			// key already exists. If so, the new extraction supersedes the old one.
			// This creates a linked chain: old → new, enabling audit trail and rollback.
			if existingID, lookupErr := p.repo.FindActiveByFactKey(ctx, fact.FactKey); lookupErr != nil {
				p.log.WithError(lookupErr).Debug("Version chain lookup failed — proceeding without supersede")
			} else if existingID != "" && existingID != fact.ID {
				fact.Supersedes = existingID
			}

			if err := p.repo.CreateDerivedFact(ctx, fact); err != nil {
				p.log.WithError(err).Warn("Failed to store fact")
				result.Errors = append(result.Errors, err.Error())
				continue
			}

			// P6.2: Complete the version chain — mark old fact as SUPERSEDED
			if fact.Supersedes != "" {
				if supersedeErr := p.repo.SupersedeFact(ctx, fact.Supersedes, fact.ID); supersedeErr != nil {
					p.log.WithError(supersedeErr).Warn("Failed to supersede old fact — chain incomplete")
				} else {
					p.log.WithFields(logrus.Fields{
						"newFactID": fact.ID[:16],
						"oldFactID": fact.Supersedes[:16],
						"factType":  fact.FactType,
					}).Debug("Version chain: new fact supersedes old")
				}
			}

			allExtractedFacts = append(allExtractedFacts, fact)
			result.FactsExtracted++

			// Step 6: Apply governance
			govResult, err := p.applyGovernance(ctx, fact)
			if err != nil {
				p.log.WithError(err).Warn("Failed to apply governance")
				continue
			}

			switch govResult.Status {
			case "APPROVED":
				result.FactsApproved++
			case "PENDING_REVIEW":
				result.FactsQueued++
			case "REJECTED":
				result.FactsRejected++
			}
		}
	}

	// Phase 3d: KDIGO organ impairment enrichment (PDF-extracted or MCP-RAG, always PENDING_REVIEW)
	// NOTE: CPIC removed from OI path — CPIC provides pharmacogenomics (gene-drug), not organ impairment (eGFR thresholds)
	if p.kdigoClient != nil {
		kdigoFacts, kdigoErr := p.enrichOrganImpairmentFromKDIGO(ctx, sourceDoc)
		if kdigoErr != nil {
			p.log.WithError(kdigoErr).WithField("drug", sourceDoc.DrugName).Debug("KDIGO organ impairment enrichment returned no results")
		} else {
			for _, fact := range kdigoFacts {
				if err := p.repo.CreateDerivedFact(ctx, fact); err != nil {
					p.log.WithError(err).Warn("Failed to store KDIGO organ impairment fact")
					result.Errors = append(result.Errors, err.Error())
					continue
				}
				result.FactsExtracted++
				// KDIGO PDF facts are FORCED to PENDING_REVIEW — skip normal governance
				result.FactsQueued++
			}
		}
	}

	// Update source document status
	status := "COMPLETED"
	if len(result.Errors) > 0 {
		status = "COMPLETED_WITH_ERRORS"
	}
	_ = p.repo.UpdateSourceDocumentStatus(ctx, sourceDoc.ID, status, strings.Join(result.Errors, "; "))

	result.CompletedAt = time.Now()
	result.Duration = result.CompletedAt.Sub(startTime)

	// P3: Completeness check — per-drug quality report with all extracted facts.
	// P3.2: Now persisted to completeness_reports table (previously discarded).
	cc := NewCompletenessChecker(p.log)
	completenessReport := cc.Check(sourceDoc.DrugName, rxcui, allExtractedFacts, totalSourceRows, processedSectionCodes, p.currentSkipReasons)
	if completenessReport != nil {
		if saveErr := p.repo.SaveCompletenessReport(ctx, completenessReport); saveErr != nil {
			p.log.WithError(saveErr).Warn("Failed to persist completeness report (non-fatal)")
		}
	}

	// Update metrics
	p.updateMetrics(result)

	p.log.WithFields(logrus.Fields{
		"setId":          splDoc.SetID.Root,
		"facts":          result.FactsExtracted,
		"approved":       result.FactsApproved,
		"queued":         result.FactsQueued,
		"rejected":       result.FactsRejected,
		"duration":       result.Duration,
	}).Info("SPL document processed")

	return result, nil
}

// ProcessingResult contains the outcome of document processing
type ProcessingResult struct {
	SourceDocumentID  string        `json:"sourceDocumentId"`
	StartedAt         time.Time     `json:"startedAt"`
	CompletedAt       time.Time     `json:"completedAt"`
	Duration          time.Duration `json:"duration"`
	TotalSections     int           `json:"totalSections"`
	SectionsProcessed int           `json:"sectionsProcessed"`
	FactsExtracted    int           `json:"factsExtracted"`
	FactsApproved     int           `json:"factsApproved"`
	FactsQueued       int           `json:"factsQueued"`
	FactsRejected     int           `json:"factsRejected"`
	Errors            []string      `json:"errors,omitempty"`
	Error             string        `json:"error,omitempty"`
}

// =============================================================================
// SOURCE DOCUMENT CREATION
// =============================================================================

func (p *Pipeline) createSourceDocument(ctx context.Context, splDoc *dailymed.SPLDocument, rxcui string, knownDrugName string) (*SourceDocument, error) {
	// Compute content hash for change detection
	contentHash := computeHash(splDoc.Title + splDoc.SetID.Root + fmt.Sprintf("%d", splDoc.VersionNumber.Value))

	// Use Phase B validated drug name if available, otherwise extract from SPL title
	drugName := knownDrugName
	if drugName == "" {
		drugName = extractDrugNameFromTitle(splDoc.Title)
	}
	canonicalRxCUI := rxcui

	// Phase 3 Issue 1 Fix: Validate and correct RxCUI via RxNav-in-a-Box.
	// SPL XML often contains wrong/outdated RxCUIs that don't match drug_master,
	// causing FK constraint failures when projecting to clinical_facts.
	// Example: SPL says Lithium=5521 (hydroxychloroquine!), correct is 6448.
	if p.drugNormalizer != nil {
		normalized, err := p.drugNormalizer.ValidateAndNormalize(ctx, rxcui, drugName)
		if err != nil {
			// Log warning but don't fail - use original values as fallback
			p.log.WithError(err).WithFields(logrus.Fields{
				"rxcui":     rxcui,
				"drug_name": drugName,
			}).Warn("Drug normalization failed, using original values")
		} else {
			// Use validated/corrected values
			drugName = normalized.CanonicalName
			canonicalRxCUI = normalized.CanonicalRxCUI

			if normalized.WasCorrected {
				p.log.WithFields(logrus.Fields{
					"original_rxcui":  normalized.OriginalRxCUI,
					"canonical_rxcui": normalized.CanonicalRxCUI,
					"drug_name":       normalized.CanonicalName,
					"confidence":      normalized.Confidence,
				}).Info("RxCUI CORRECTED - FK constraint issue prevented")
			} else {
				p.log.WithFields(logrus.Fields{
					"rxcui":     canonicalRxCUI,
					"drug_name": drugName,
				}).Debug("RxCUI validated successfully")
			}
		}
	}

	doc := &SourceDocument{
		SourceType:       "FDA_SPL",
		DocumentID:       splDoc.SetID.Root,
		VersionNumber:    fmt.Sprintf("%d", splDoc.VersionNumber.Value),
		RawContentHash:   contentHash,
		FetchedAt:        time.Now(),
		DrugName:         drugName,
		RxCUI:            canonicalRxCUI, // Now validated/corrected!
		ProcessingStatus: "PROCESSING",
	}

	if err := p.repo.CreateSourceDocument(ctx, doc); err != nil {
		return nil, err
	}

	return doc, nil
}

// extractDrugNameFromTitle extracts a clean drug name from SPL title
// SPL titles can be verbose like:
// "METFORMIN HYDROCHLORIDE TABLET, EXTENDED RELEASE [MANUFACTURER NAME]"
// This extracts just the drug name portion
func extractDrugNameFromTitle(title string) string {
	// If empty, return empty
	if title == "" {
		return ""
	}

	// Try to extract up to first "[" (manufacturer info)
	if idx := strings.Index(title, "["); idx > 0 {
		title = strings.TrimSpace(title[:idx])
	}

	// Try to extract up to first comma (dosage form info)
	if idx := strings.Index(title, ","); idx > 0 {
		title = strings.TrimSpace(title[:idx])
	}

	// Handle "HIGHLIGHTS" prefix that some SPL docs have
	if strings.HasPrefix(title, "These highlights") || strings.HasPrefix(title, "HIGHLIGHTS") {
		// Try to find the actual drug name - look for UPPERCASE DRUG NAME pattern
		// Format: "...METFORMIN HYDROCHLORIDE TABLETS..."
		words := strings.Fields(title)
		var drugWords []string
		inDrugName := false
		for _, word := range words {
			isUpperWord := strings.ToUpper(word) == word && len(word) > 2
			// Skip common non-drug words
			if isUpperWord && !isCommonWord(word) {
				inDrugName = true
				drugWords = append(drugWords, word)
			} else if inDrugName && isUpperWord {
				drugWords = append(drugWords, word)
			} else if inDrugName && !isUpperWord {
				break // End of drug name
			}
		}
		if len(drugWords) > 0 {
			title = strings.Join(drugWords, " ")
		}
	}

	// Truncate to 200 chars max (leave room for DB VARCHAR(255))
	if len(title) > 200 {
		title = title[:200]
	}

	return title
}

// isCommonWord returns true if word is a common non-drug word
func isCommonWord(word string) bool {
	commonWords := map[string]bool{
		"THESE": true, "HIGHLIGHTS": true, "DO": true, "NOT": true,
		"INCLUDE": true, "ALL": true, "THE": true, "INFORMATION": true,
		"NEEDED": true, "TO": true, "USE": true, "SAFELY": true,
		"AND": true, "EFFECTIVELY": true, "SEE": true, "FULL": true,
		"PRESCRIBING": true, "FOR": true, "INITIAL": true, "U.S.": true,
		"APPROVAL": true, "TABLETS": true, "TABLET": true, "CAPSULE": true,
		"CAPSULES": true, "INJECTION": true, "SOLUTION": true, "ORAL": true,
		"EXTENDED-RELEASE": true, "IMMEDIATE-RELEASE": true,
	}
	return commonWords[word]
}

// =============================================================================
// SOURCE SECTION CREATION
// =============================================================================

func (p *Pipeline) createSourceSection(ctx context.Context, docID string, routed *dailymed.RoutedSection) (*SourceSection, error) {
	// Serialize parsed tables to JSON with sanitization for PostgreSQL compatibility
	// SPL documents often contain control characters that break PostgreSQL's JSON parser
	var parsedTables json.RawMessage
	if len(routed.ExtractedTables) > 0 {
		// Sanitize tables to remove NULL bytes and invalid UTF-8
		sanitized := sanitizeExtractedTables(routed.ExtractedTables)
		marshaled, marshalErr := json.Marshal(sanitized)
		if marshalErr != nil {
			p.log.WithError(marshalErr).Debug("Failed to marshal extracted tables, using empty array")
			parsedTables = json.RawMessage(`[]`)
		} else {
			// PostgreSQL jsonb rejects \u0000 (Unicode NULL) even in valid JSON.
			cleaned := bytes.ReplaceAll(marshaled, []byte(`\u0000`), []byte{})
			// Validate the result is still valid JSON after cleaning
			if json.Valid(cleaned) {
				parsedTables = cleaned
			} else {
				p.log.Debug("Cleaned JSON invalid for jsonb, using empty array")
				parsedTables = json.RawMessage(`[]`)
			}
		}
	}

	// Determine extraction method based on content
	extractionMethod := "NARRATIVE_PARSE"
	if routed.HasTables {
		extractionMethod = "TABLE_PARSE"
	}

	section := &SourceSection{
		SourceDocumentID:     docID,
		SectionCode:          routed.Section.Code.Code,
		SectionName:          sanitizeForJSON(routed.Section.Code.DisplayName),
		TargetKBs:            routed.TargetKBs,
		RawText:              sanitizeForJSON(routed.PlainText),
		RawHTML:              sanitizeForJSON(routed.Section.Text.Content),
		ParsedTables:         parsedTables,
		ExtractionMethod:     extractionMethod,
		ExtractionConfidence: 0.80, // Default for structural parsing
		HasStructuredTables:  routed.HasTables,
		TableCount:           routed.TableCount,
		WordCount:            len(strings.Fields(routed.PlainText)),
	}

	if err := p.repo.CreateSourceSection(ctx, section); err != nil {
		return nil, err
	}

	return section, nil
}

// =============================================================================
// FACT EXTRACTION
// =============================================================================

func (p *Pipeline) extractFactsFromSection(
	ctx context.Context,
	sourceDoc *SourceDocument,
	sourceSection *SourceSection,
	routed *dailymed.RoutedSection,
	rxcui string,
) ([]*DerivedFact, error) {
	var facts []*DerivedFact
	auditStart := time.Now()

	// P5.2: Intercept reproductive safety sections BEFORE the safety block.
	// Pregnancy (34077-8) and Nursing (34080-2) are blocked from SAFETY_SIGNAL
	// extraction but should produce REPRODUCTIVE_SAFETY facts instead.
	if isReproductiveSection(sourceSection.SectionCode) {
		return p.extractReproductiveSafetyFacts(ctx, sourceDoc, sourceSection, routed, rxcui)
	}

	// LOINC Section Gating: Only parse safety signals from appropriate sections.
	// Clinical Studies (34092-7) and Pregnancy (34077-8) sections produce noise like
	// "Fatal familial insomnia" (exclusion criteria) and "Baseline foetal heart rate
	// variability disorder" (obstetric context). Block them from SAFETY_SIGNAL extraction.
	if isSafetyBlockedSection(sourceSection.SectionCode) {
		p.log.WithFields(logrus.Fields{
			"sectionCode": sourceSection.SectionCode,
			"sectionName": sourceSection.SectionName,
			"drug":        sourceDoc.DrugName,
		}).Debug("Section gated: blocked from safety signal extraction")

		// LAB EXTRACTION: DISABLED — parked pending data quality review.
		// Code preserved in llm_fallback.go for future re-enablement.
		// if p.llmProvider != nil && isLabEligibleSection(sourceSection.SectionCode) && !p.llmBudget.exhausted() { ... }

		// ORGAN_IMPAIRMENT: SPL route permanently removed.
		// SPL labels are descriptive ("Use with caution in renal impairment"), not normative.
		// Organ impairment sourced exclusively from guideline authorities (CPIC, KDIGO).

		return facts, nil
	}

	// Get routing decision from authority router
	decision := p.authorityRouter.RouteByLOINC(sourceSection.SectionCode, routed.PlainText)

	p.log.WithFields(logrus.Fields{
		"sectionCode":      sourceSection.SectionCode,
		"primaryAuthority": decision.PrimaryAuthority,
		"extractionMethod": decision.ExtractionMethod,
		"llmPolicy":        decision.LLMPolicy,
	}).Debug("Routing decision")

	// AUTHORITY PATH: If authority exists, use it (LLM NEVER for definitive sources)
	if decision.PrimaryAuthority != "" && decision.ExtractionMethod == "AUTHORITY_LOOKUP" {
		authorityFacts, err := p.extractFromAuthority(ctx, rxcui, decision)
		if err == nil && len(authorityFacts) > 0 {
			for _, af := range authorityFacts {
				fact := p.convertAuthorityFact(sourceDoc, sourceSection, af, routed.TargetKBs)
				facts = append(facts, fact)
			}

			// Log successful authority extraction
			p.logExtraction(ctx, sourceDoc.ID, sourceSection.ID, "AUTHORITY_LOOKUP", auditStart, true, 1.0, nil)
			p.mu.Lock()
			p.metrics.AuthorityLookups++
			p.mu.Unlock()

			return facts, nil
		}
		// Authority lookup failed or returned no results, fall through to table parsing
	}

	// TABLE PARSE PATH: If section has tables, parse them
	if routed.HasTables && len(routed.ExtractedTables) > 0 {
		tableFacts := p.extractFromTables(sourceDoc, sourceSection, routed, sourceDoc.DrugName)

		// DEDUP GATE: Skip facts whose canonical key was already seen in this pipeline run.
		// This collapses duplicates like Apixaban "Death" appearing in multiple sections
		// (Boxed Warning, Adverse Reactions, Clinical Studies). The canonical key is
		// hash(rxcui|factType|MedDRAPT) so same drug+condition = same key regardless of section.
		dedupedFacts := make([]*DerivedFact, 0, len(tableFacts))
		for _, f := range tableFacts {
			p.mu.Lock()
			if p.seenFactKeys == nil {
				p.seenFactKeys = make(map[string]bool)
			}
			if p.seenFactKeys[f.FactKey] {
				p.mu.Unlock()
				p.log.WithFields(map[string]interface{}{
					"factType": f.FactType,
					"factKey":  f.FactKey[:16],
				}).Debug("Dedup: skipping duplicate fact_key")
				continue
			}
			p.seenFactKeys[f.FactKey] = true
			p.mu.Unlock()
			dedupedFacts = append(dedupedFacts, f)
		}
		facts = append(facts, dedupedFacts...)

		if len(dedupedFacts) > 0 {
			p.logExtraction(ctx, sourceDoc.ID, sourceSection.ID, "TABLE_PARSE", auditStart, true, 0.90, nil)
		}
	}

	// P2.2: DDI GRAMMAR EXTRACTION — deterministic prose patterns for Drug Interactions.
	// Runs AFTER table parsing (tables may have PK data) and BEFORE LLM fallback.
	// Grammar patterns match standard FDA DDI prose language and produce STRUCTURED_PARSE facts.
	if sourceSection.SectionCode == "34073-7" && len(routed.PlainText) >= 100 && p.ddiGrammar != nil {
		grammarMatches := p.ddiGrammar.ExtractFromProse(routed.PlainText, sourceDoc.DrugName)
		if len(grammarMatches) > 0 {
			grammarContents := p.ddiGrammar.ToInteractionContents(grammarMatches, sourceDoc.DrugName)

			for _, content := range grammarContents {
				canonicalKey := generateCanonicalKey(sourceDoc.RxCUI, "INTERACTION", content)
				// Dedup against table-parsed interactions
				p.mu.Lock()
				if p.seenFactKeys == nil {
					p.seenFactKeys = make(map[string]bool)
				}
				if p.seenFactKeys[canonicalKey] {
					p.mu.Unlock()
					continue
				}
				p.seenFactKeys[canonicalKey] = true
				p.mu.Unlock()

				factData, _ := json.Marshal(content)
				evidenceSpans, _ := json.Marshal([]string{fmt.Sprintf("Grammar: %s", content.SourcePhrase)})

				fact := &DerivedFact{
					SourceDocumentID:     sourceDoc.ID,
					SourceSectionID:      sourceSection.ID,
					TargetKB:             "KB-5",
					FactType:             "INTERACTION",
					FactKey:              canonicalKey,
					FactData:             factData,
					ExtractionMethod:     "STRUCTURED_PARSE",
					ExtractionConfidence: 0.80, // Grammar: high confidence but not table-level
					EvidenceSpans:        evidenceSpans,
					ConsensusAchieved:    true,
					GovernanceStatus:     "DRAFT",
				}
				facts = append(facts, fact)
			}

			p.mu.Lock()
			p.metrics.GrammarExtractions += int64(len(grammarContents))
			p.mu.Unlock()

			p.log.WithFields(logrus.Fields{
				"drug":             sourceDoc.DrugName,
				"grammarMatches":   len(grammarMatches),
				"grammarFacts":     len(grammarContents),
				"totalFactsSoFar":  len(facts),
			}).Info("DDI grammar extracted interactions from prose")
		}
	}

	// FIX 3: MEDDRA PROSE SCANNER — deterministic AE extraction from free text.
	// Runs AFTER table parsing and DDI grammar, BEFORE LLM fallback.
	// This is the SCANNER (finds terms in prose) vs the aeNormalizer VALIDATOR (checks known terms).
	// Eligible sections: Adverse Reactions, Boxed Warning, Warnings, Precautions, W&P.
	// NOT Drug Interactions (34073-7) — that section has DDI grammar above.
	if p.proseScanner != nil &&
		sourceSection.SectionCode != "34073-7" &&
		len(routed.PlainText) >= 200 &&
		isProseScannableSection(sourceSection.SectionCode) {

		scanResult := p.proseScanner.ScanText(routed.PlainText)
		if scanResult != nil && len(scanResult.Matches) > 0 {
			parser := NewContentParser()
			signalType := parser.loincToSignalType(sourceSection.SectionCode)

			targetKB := "KB-4"
			if len(routed.TargetKBs) > 0 {
				targetKB = routed.TargetKBs[0]
			}

			var proseFacts int
			for _, match := range scanResult.Matches {
				content := KBSafetySignalContent{
					SignalType:     signalType,
					Severity:       "MEDIUM", // Default; prose doesn't carry severity markers
					ConditionName:  match.PTName,
					MedDRAPT:       match.PTCode,
					MedDRAName:     match.PTName,
					MedDRALLT:      match.LLTCode,
					MedDRASOC:      match.SOCCode,
					MedDRASOCName:  match.SOCName,
					Frequency:      match.Frequency,
					FrequencyBand:  match.FrequencyBand,
					TermConfidence: 1.0, // Exact MedDRA dictionary match
				}

				canonicalKey := generateCanonicalKey(sourceDoc.RxCUI, "SAFETY_SIGNAL", content)

				// Dedup against table-parsed facts (tables take priority)
				p.mu.Lock()
				if p.seenFactKeys == nil {
					p.seenFactKeys = make(map[string]bool)
				}
				if p.seenFactKeys[canonicalKey] {
					p.mu.Unlock()
					continue
				}
				p.seenFactKeys[canonicalKey] = true
				p.mu.Unlock()

				factData, _ := json.Marshal(content)
				evidenceSpans, _ := json.Marshal([]string{
					fmt.Sprintf("Prose scanner: matched '%s' → MedDRA PT %s (%s)",
						match.MatchedText, match.PTCode, match.PTName),
				})

				fact := &DerivedFact{
					SourceDocumentID:     sourceDoc.ID,
					SourceSectionID:      sourceSection.ID,
					TargetKB:             targetKB,
					FactType:             "SAFETY_SIGNAL",
					FactKey:              canonicalKey,
					FactData:             factData,
					ExtractionMethod:     "STRUCTURED_PARSE",
					ExtractionConfidence: 0.85, // Dictionary match: high confidence, below table-parse 0.90
					EvidenceSpans:        evidenceSpans,
					ConsensusAchieved:    true,
					GovernanceStatus:     "DRAFT",
				}
				facts = append(facts, fact)
				proseFacts++
			}

			p.log.WithFields(logrus.Fields{
				"drug":          sourceDoc.DrugName,
				"sectionCode":   sourceSection.SectionCode,
				"proseMatches":  len(scanResult.Matches),
				"negatedSkipped": scanResult.NegatedCount,
				"proseFacts":     proseFacts,
				"totalFacts":     len(facts),
			}).Info("MedDRA prose scanner extracted AEs from free text")
		}
	}

	// AUDIT LOG: Note when tables existed but produced 0 AE facts
	// This is NOT always a parser bug — tables may be PK/demographic, not AE tables
	if len(facts) == 0 && routed.HasTables && len(routed.ExtractedTables) > 0 {
		p.log.WithFields(logrus.Fields{
			"drug":        sourceDoc.DrugName,
			"sectionCode": sourceSection.SectionCode,
			"tableCount":  len(routed.ExtractedTables),
		}).Info("Tables existed but produced 0 AE facts — attempting LLM prose extraction")
	}

	// LLM FALLBACK: Extract safety signals from prose when no facts produced
	if len(facts) == 0 &&
		p.llmProvider != nil &&
		p.isLLMEligibleSection(sourceSection.SectionCode) &&
		len(routed.PlainText) >= 500 &&
		!p.llmBudget.exhausted() {

		var llmFacts []*DerivedFact
		var llmErr error
		if sourceSection.SectionCode == "34073-7" {
			llmFacts, llmErr = p.extractInteractionsViaLLM(ctx, sourceDoc, sourceSection, routed)
		} else {
			llmFacts, llmErr = p.extractSafetySignalsViaLLM(ctx, sourceDoc, sourceSection, routed)
		}
		if llmErr != nil {
			p.log.WithError(llmErr).WithField("drug", sourceDoc.DrugName).Warn("LLM fallback failed, continuing without")
		} else {
			for _, f := range llmFacts {
				p.mu.Lock()
				if p.seenFactKeys == nil {
					p.seenFactKeys = make(map[string]bool)
				}
				if !p.seenFactKeys[f.FactKey] {
					p.seenFactKeys[f.FactKey] = true
					p.mu.Unlock()
					facts = append(facts, f)
				} else {
					p.mu.Unlock()
				}
			}
			p.mu.Lock()
			p.metrics.LLMExtractions += int64(len(llmFacts))
			p.mu.Unlock()
		}
	}

	// P5.3: LAB_REFERENCE ADDITIVE PASS — prose grammar for lab monitoring instructions.
	// Runs on Warnings, Dosage, W&P, Boxed Warning sections. These sections primarily
	// produce SAFETY_SIGNAL facts from tables, but also contain narrative monitoring
	// instructions ("Monitor serum creatinine periodically") that should become
	// LAB_REFERENCE facts. This pass is ADDITIVE — it doesn't replace SAFETY_SIGNALs.
	if isLabMonitoringSection(sourceSection.SectionCode) && len(routed.PlainText) >= 100 {
		parser := NewContentParser()
		labContents := parser.extractLabMonitoringFromProse(routed.PlainText)
		for _, content := range labContents {
			canonicalKey := generateCanonicalKey(rxcui, "LAB_REFERENCE", content)

			// Dedup against previously seen facts
			p.mu.Lock()
			if p.seenFactKeys == nil {
				p.seenFactKeys = make(map[string]bool)
			}
			if p.seenFactKeys[canonicalKey] {
				p.mu.Unlock()
				continue
			}
			p.seenFactKeys[canonicalKey] = true
			p.mu.Unlock()

			factData, _ := json.Marshal(content)
			evidenceSpans, _ := json.Marshal([]string{fmt.Sprintf("Prose: %s section", sourceSection.SectionName)})

			fact := &DerivedFact{
				SourceDocumentID:     sourceDoc.ID,
				SourceSectionID:      sourceSection.ID,
				TargetKB:             "KB-16",
				FactType:             "LAB_REFERENCE",
				FactKey:              canonicalKey,
				FactData:             factData,
				ExtractionMethod:     "STRUCTURED_PARSE",
				ExtractionConfidence: 0.70, // Prose grammar — moderate confidence, routes to review
				EvidenceSpans:        evidenceSpans,
				ConsensusAchieved:    true,
				GovernanceStatus:     "DRAFT",
			}
			facts = append(facts, fact)
		}

		if len(labContents) > 0 {
			p.log.WithFields(logrus.Fields{
				"drug":        sourceDoc.DrugName,
				"section":     sourceSection.SectionName,
				"labFactsNew": len(labContents),
			}).Info("LAB_REFERENCE facts extracted from prose monitoring instructions")
		}
	}

	// P2.4: Signal Merger — cross-method corroboration.
	// When table parsing + grammar extraction both produce facts with the same
	// canonical key, merge them: STRUCTURED_PARSE wins, confidence boosted.
	if len(facts) > 1 {
		merger := NewSignalMerger(p.log)
		mergeResult := merger.Merge(facts)
		facts = mergeResult.Facts
	}

	return facts, nil
}

// extractFromAuthority retrieves facts from authoritative sources
func (p *Pipeline) extractFromAuthority(ctx context.Context, rxcui string, decision *routing.RoutingDecision) ([]datasources.AuthorityFact, error) {
	facts, err := p.authorityRouter.GetFacts(ctx, rxcui, decision.FactType)
	if err != nil {
		return nil, err
	}
	return facts, nil
}

// convertAuthorityFact converts an authority fact to a derived fact
func (p *Pipeline) convertAuthorityFact(
	sourceDoc *SourceDocument,
	sourceSection *SourceSection,
	af datasources.AuthorityFact,
	targetKBs []string,
) *DerivedFact {
	// Determine target KB based on fact type
	targetKB := "KB-4" // Default to safety
	for _, kb := range targetKBs {
		targetKB = kb
		break
	}

	// Convert authority fact content to JSON
	factData, _ := json.Marshal(af.Content)

	// Use References as evidence source
	evidenceSpans, _ := json.Marshal(af.References)

	return &DerivedFact{
		SourceDocumentID:     sourceDoc.ID,
		SourceSectionID:      sourceSection.ID,
		TargetKB:             targetKB,
		FactType:             string(af.FactType),
		FactKey:              fmt.Sprintf("%s:%s:%s", sourceDoc.RxCUI, af.FactType, af.AuthoritySource),
		FactData:             factData,
		ExtractionMethod:     "AUTHORITY",
		ExtractionConfidence: 1.0, // Authority sources are 100% confidence
		EvidenceSpans:        evidenceSpans,
		ConsensusAchieved:    true, // Authority doesn't need consensus
		GovernanceStatus:     "DRAFT",
	}
}

// extractFromTables extracts facts from parsed tables using structured content parsing
// Creates one fact per parsed row (not one per table) to match KB view expectations
func (p *Pipeline) extractFromTables(
	sourceDoc *SourceDocument,
	sourceSection *SourceSection,
	routed *dailymed.RoutedSection,
	indexDrugName string,
) []*DerivedFact {
	var facts []*DerivedFact

	// Initialize content parser for structured extraction
	parser := NewContentParser()

	// Phase 3 Issues 2+3: Wire MedDRA normalizer if available
	if p.aeNormalizer != nil {
		parser.SetAdverseEventNormalizer(p.aeNormalizer)
	} else {
		p.log.Debug("MedDRA AE normalizer not loaded — table parse uses regex noise filter only")
	}

	for _, table := range routed.ExtractedTables {
		// Create fact based on table type
		factType := mapTableTypeToFactType(table.TableType)
		if factType == "" {
			p.currentSkipReasons["unmapped_table_type"] += len(table.Rows)
			continue
		}

		// P5.4: Override fact type for How Supplied section (LOINC 34069-5).
		// Tables from this section should produce FORMULARY facts (NDC codes,
		// packaging), not SAFETY_SIGNAL. The table classifier may not have a
		// specific "HowSupplied" type, so we override based on section LOINC.
		if sourceSection.SectionCode == "34069-5" {
			factType = "FORMULARY"
		}

		// Determine target KB
		targetKB := "KB-1" // Default to drug rules
		if len(table.TargetKBs) > 0 {
			targetKB = table.TargetKBs[0]
		}
		// P5.4: Override target KB for formulary facts
		if factType == "FORMULARY" {
			targetKB = "KB-6"
		}

		loincCode := sourceSection.SectionCode
		parseResult, parseErr := parser.Parse(table, loincCode, factType, indexDrugName)

		// Use Caption as the table identifier
		evidenceSpans, _ := json.Marshal([]string{fmt.Sprintf("Table: %s", table.Caption)})

		if parseErr == nil && parseResult != nil && parseResult.ParsedRows > 0 {
			// P1.5b: Prose frequency annotation pass
			// Enrich SAFETY_SIGNAL facts with frequency data from section narrative.
			// Table parsing extracts frequency from adjacent columns (P0.3), but
			// some drugs state frequencies only in prose ("Headache occurred in 12%").
			if factType == "SAFETY_SIGNAL" {
				if contents, ok := parseResult.Content.([]KBSafetySignalContent); ok && routed.PlainText != "" {
					parseResult.Content = parser.annotateProseFrequency(routed.PlainText, contents)
				}
			}

			// Successfully parsed - create one fact per parsed item (not per table)
			// This ensures KB views get single objects, not arrays
			parsedItems := extractParsedItems(parseResult.Content)

			p.log.WithFields(map[string]interface{}{
				"factType":   factType,
				"parsedRows": parseResult.ParsedRows,
				"totalRows":  parseResult.TotalRows,
				"items":      len(parsedItems),
				"confidence": parseResult.Confidence,
			}).Debug("Content parsed into structured format")

			// P3: Track rows skipped by noise filter / header detection inside Parse()
			if skipped := parseResult.TotalRows - parseResult.ParsedRows; skipped > 0 {
				p.currentSkipReasons["noise_or_header"] += skipped
			}

			for _, item := range parsedItems {
				// GUARDRAIL: Empty content rejection — skip facts with no meaningful data
				if isEmptyContent(item) {
					p.log.WithField("factType", factType).Debug("Skipping empty content fact")
					p.currentSkipReasons["empty_content"]++
					continue
				}

				// GUARDRAIL: Contextual exclusion — filter terms that are valid MedDRA
				// but contextually impossible (surgical procedures, genetic diseases,
				// pregnancy-only terms on non-pregnancy drugs)
				if isContextuallyExcluded(factType, item) {
					p.log.WithFields(map[string]interface{}{
						"factType": factType,
						"item":     fmt.Sprintf("%v", item),
					}).Debug("Skipping contextually excluded term")
					p.currentSkipReasons["contextual_exclusion"]++
					continue
				}

				factData, _ := json.Marshal(item)

				// Generate canonical key for dedup at projection time.
				// Key = rxcui|factType|normalizedCondition so duplicates across
				// multiple trial tables (ARISTOTLE, AMPLIFY, etc.) are collapsed.
				canonicalKey := generateCanonicalKey(sourceDoc.RxCUI, factType, item)

				// P5.1: Cap ORGAN_IMPAIRMENT confidence at 0.70 so SPL-sourced
				// organ impairment always routes to PENDING_REVIEW (never auto-approved).
				confidence := parseResult.Confidence
				if factType == "ORGAN_IMPAIRMENT" && confidence > 0.70 {
					confidence = 0.70
				}

				// P5.4: FORMULARY facts from SPL use "SPL_PRODUCT" to distinguish from
				// CMS Medicare Part D ETL data which uses "CMS_FORMULARY". KB-6 consumer
				// dispatches based on ExtractionMethod to handle the two data shapes.
				extractionMethod := "STRUCTURED_PARSE"
				if factType == "FORMULARY" {
					extractionMethod = "SPL_PRODUCT"
				}

				fact := &DerivedFact{
					SourceDocumentID:     sourceDoc.ID,
					SourceSectionID:      sourceSection.ID,
					TargetKB:             targetKB,
					FactType:             factType,
					FactKey:              canonicalKey,
					FactData:             factData,
					ExtractionMethod:     extractionMethod,
					ExtractionConfidence: confidence,
					EvidenceSpans:        evidenceSpans,
					ConsensusAchieved:    true,
					GovernanceStatus:     "DRAFT",
				}
				facts = append(facts, fact)
			}
		} else {
			p.currentSkipReasons["parse_failure"] += len(table.Rows)
			// QUALITY GATE: Do NOT store raw table fallbacks as derived facts.
			// When parsedRows == 0 or parsing fails, the table contains no extractable
			// clinical content (e.g., efficacy tables misrouted as ORGAN_IMPAIRMENT,
			// or tables with no recognizable adverse event terms). Storing these creates
			// empty-shell facts that waste governance reviewer time and pollute KB views.
			//
			// Previously this created TABLE_RAW facts with halved confidence, which led
			// to 33 empty ORGAN_IMPAIRMENT shells and 4 empty-condition safety signals.
			if parseErr != nil {
				p.log.WithError(parseErr).WithFields(map[string]interface{}{
					"factType": factType,
					"caption":  table.Caption,
				}).Debug("Content parsing failed — table dropped (no raw fallback)")
			} else {
				p.log.WithFields(map[string]interface{}{
					"factType": factType,
					"caption":  table.Caption,
				}).Debug("No structured content extracted — table dropped (no raw fallback)")
			}
		}
	}

	return facts
}

// enrichOrganImpairmentFromKDIGO queries the KDIGO PDF-extracted rules for
// renal/hepatic dosing and converts them into DerivedFacts.
//
// Safety invariants:
//   - ExtractionMethod: "PDF_TABLE_EXTRACT" (never "AUTHORITY_LOOKUP")
//   - Confidence: ≤ 0.75 (enforced in kdigo.Client)
//   - ConsensusAchieved: false (NEVER true for PDF extraction)
//   - GovernanceStatus: "PENDING_REVIEW" (FORCED — ignores auto-approve threshold)
func (p *Pipeline) enrichOrganImpairmentFromKDIGO(ctx context.Context, sourceDoc *SourceDocument) ([]*DerivedFact, error) {
	kdigoFacts := p.kdigoClient.GetOrganImpairmentFacts(sourceDoc.RxCUI, sourceDoc.DrugName)
	if len(kdigoFacts) == 0 {
		return nil, nil
	}

	var facts []*DerivedFact
	for _, af := range kdigoFacts {
		factData, _ := json.Marshal(af.Content)
		canonicalKey := generateCanonicalKey(sourceDoc.RxCUI, "ORGAN_IMPAIRMENT", af.Content)

		p.mu.Lock()
		if p.seenFactKeys == nil {
			p.seenFactKeys = make(map[string]bool)
		}
		if p.seenFactKeys[canonicalKey] {
			p.mu.Unlock()
			continue
		}
		p.seenFactKeys[canonicalKey] = true
		p.mu.Unlock()

		fact := &DerivedFact{
			SourceDocumentID:     sourceDoc.ID,
			TargetKB:             "KB-1",
			FactType:             "ORGAN_IMPAIRMENT",
			FactKey:              canonicalKey,
			FactData:             factData,
			ExtractionMethod:     "PDF_TABLE_EXTRACT",
			ExtractionConfidence: af.Confidence,
			ConsensusAchieved:    false,              // NEVER true for PDF extraction
			GovernanceStatus:     "PENDING_REVIEW",   // FORCED — ignores auto-approve threshold
		}
		facts = append(facts, fact)
	}

	if len(facts) > 0 {
		p.log.WithFields(logrus.Fields{
			"drug":  sourceDoc.DrugName,
			"count": len(facts),
		}).Info("KDIGO organ impairment facts enriched (all PENDING_REVIEW)")
	}

	return facts, nil
}

// extractParsedItems converts parsed content (which may be a slice) into individual items
func extractParsedItems(content interface{}) []interface{} {
	if content == nil {
		return nil
	}

	// Use reflection to handle different slice types
	switch v := content.(type) {
	case []KBOrganImpairmentContent:
		items := make([]interface{}, len(v))
		for i, item := range v {
			items[i] = item
		}
		return items
	case []KBSafetySignalContent:
		items := make([]interface{}, len(v))
		for i, item := range v {
			items[i] = item
		}
		return items
	case []KBInteractionContent:
		items := make([]interface{}, len(v))
		for i, item := range v {
			items[i] = item
		}
		return items
	case []KBReproductiveSafetyContent:
		items := make([]interface{}, len(v))
		for i, item := range v {
			items[i] = item
		}
		return items
	case []KBFormularyContent:
		items := make([]interface{}, len(v))
		for i, item := range v {
			items[i] = item
		}
		return items
	case []KBLabReferenceContent:
		items := make([]interface{}, len(v))
		for i, item := range v {
			items[i] = item
		}
		return items
	default:
		// Single item or unknown type - wrap in slice
		return []interface{}{content}
	}
}

// createGapFillFact creates a fact for LLM gap-filling (requires review)
func (p *Pipeline) createGapFillFact(
	sourceDoc *SourceDocument,
	sourceSection *SourceSection,
	routed *dailymed.RoutedSection,
) *DerivedFact {
	targetKB := "KB-4" // Default to safety
	if len(routed.TargetKBs) > 0 {
		targetKB = routed.TargetKBs[0]
	}

	// Create placeholder fact data
	factData, _ := json.Marshal(map[string]interface{}{
		"sectionCode":    sourceSection.SectionCode,
		"sectionName":    sourceSection.SectionName,
		"contentPreview": truncateString(routed.PlainText, 500),
		"needsLLM":       true,
	})

	return &DerivedFact{
		SourceDocumentID:     sourceDoc.ID,
		SourceSectionID:      sourceSection.ID,
		TargetKB:             targetKB,
		FactType:             "GAP_FILL_NEEDED",
		FactKey:              fmt.Sprintf("%s:gap:%s", sourceDoc.RxCUI, sourceSection.SectionCode),
		FactData:             factData,
		ExtractionMethod:     "LLM_GAP",
		ExtractionConfidence: 0.50, // Low confidence for gap-fill
		ConsensusAchieved:    false,
		GovernanceStatus:     "DRAFT",
	}
}

// =============================================================================
// GOVERNANCE APPLICATION
// =============================================================================

// GovernanceOutcome contains the result of governance processing
type GovernanceOutcome struct {
	FactID  string `json:"factId"`
	Status  string `json:"status"`
	Notes   string `json:"notes,omitempty"`
}

func (p *Pipeline) applyGovernance(ctx context.Context, fact *DerivedFact) (*GovernanceOutcome, error) {
	outcome := &GovernanceOutcome{
		FactID: fact.ID,
	}

	confidence := fact.ExtractionConfidence

	// Apply confidence-based governance rules
	switch {
	case confidence >= p.config.AutoApproveThreshold:
		// Auto-approve: High confidence facts (≥0.85)
		outcome.Status = "APPROVED"
		outcome.Notes = fmt.Sprintf("Auto-approved: confidence %.2f >= %.2f", confidence, p.config.AutoApproveThreshold)

		if err := p.repo.UpdateFactGovernanceStatus(ctx, fact.ID, "APPROVED", "SYSTEM", outcome.Notes); err != nil {
			return nil, err
		}

	case confidence >= p.config.ReviewThreshold:
		// Queue for review: Medium confidence (0.65-0.84)
		outcome.Status = "PENDING_REVIEW"
		outcome.Notes = fmt.Sprintf("Queued for review: confidence %.2f in [%.2f, %.2f)", confidence, p.config.ReviewThreshold, p.config.AutoApproveThreshold)

		if err := p.repo.UpdateFactGovernanceStatus(ctx, fact.ID, "PENDING_REVIEW", "", outcome.Notes); err != nil {
			return nil, err
		}

		// Create escalation entry
		escalation := &EscalationEntry{
			DerivedFactID:    fact.ID,
			SourceDocumentID: fact.SourceDocumentID,
			EscalationReason: "LOW_CONFIDENCE",
			Priority:         calculatePriority(confidence, fact.FactType),
		}
		_ = p.repo.CreateEscalation(ctx, escalation)

	default:
		// Auto-reject: Low confidence (<0.65)
		outcome.Status = "REJECTED"
		outcome.Notes = fmt.Sprintf("Auto-rejected: confidence %.2f < %.2f", confidence, p.config.ReviewThreshold)

		if err := p.repo.UpdateFactGovernanceStatus(ctx, fact.ID, "REJECTED", "SYSTEM", outcome.Notes); err != nil {
			return nil, err
		}
	}

	// Special handling for critical safety facts
	if p.config.EscalateCriticalSafety && isCriticalSafetyFact(fact.FactType) && outcome.Status != "APPROVED" {
		escalation := &EscalationEntry{
			DerivedFactID:    fact.ID,
			SourceDocumentID: fact.SourceDocumentID,
			EscalationReason: "CRITICAL_SAFETY",
			Priority:         "CRITICAL",
		}
		_ = p.repo.CreateEscalation(ctx, escalation)
	}

	// Handle consensus not achieved
	if p.config.EscalateNoConsensus && !fact.ConsensusAchieved && fact.ExtractionMethod == "LLM_CONSENSUS" {
		escalation := &EscalationEntry{
			DerivedFactID:    fact.ID,
			SourceDocumentID: fact.SourceDocumentID,
			EscalationReason: "CONSENSUS_NOT_ACHIEVED",
			Priority:         "HIGH",
		}
		_ = p.repo.CreateEscalation(ctx, escalation)
	}

	return outcome, nil
}

// =============================================================================
// AUDIT LOGGING
// =============================================================================

func (p *Pipeline) logExtraction(
	ctx context.Context,
	docID string,
	sectionID string,
	method string,
	startTime time.Time,
	success bool,
	confidence float64,
	err error,
) {
	endTime := time.Now()
	entry := &ExtractionAuditEntry{
		SourceDocumentID:      docID,
		SourceSectionID:       sectionID,
		ExtractionMethod:      method,
		ExtractionStartedAt:   startTime,
		ExtractionCompletedAt: &endTime,
		ExtractionDurationMs:  int(endTime.Sub(startTime).Milliseconds()),
		Success:               success,
		ConfidenceScore:       confidence,
	}

	if err != nil {
		entry.ErrorMessage = err.Error()
	}

	_ = p.repo.LogExtraction(ctx, entry)
}

// =============================================================================
// METRICS
// =============================================================================

func (p *Pipeline) updateMetrics(result *ProcessingResult) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.metrics.DocumentsProcessed++
	p.metrics.SectionsProcessed += int64(result.SectionsProcessed)
	p.metrics.FactsExtracted += int64(result.FactsExtracted)
	p.metrics.FactsAutoApproved += int64(result.FactsApproved)
	p.metrics.FactsQueued += int64(result.FactsQueued)
	p.metrics.FactsRejected += int64(result.FactsRejected)
	p.metrics.LastProcessedAt = time.Now()
	p.metrics.ProcessingDuration = result.Duration
}

// GetMetrics returns current pipeline metrics
func (p *Pipeline) GetMetrics() PipelineMetrics {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.metrics
}

// =============================================================================
// BATCH PROCESSING
// =============================================================================

// ProcessPendingDocuments processes documents in PENDING status
func (p *Pipeline) ProcessPendingDocuments(ctx context.Context) error {
	// This would query source_documents with status=PENDING
	// and process each one through ProcessSPLDocument
	// Implementation depends on how documents are queued
	return nil
}

// Start begins the background processing loop
func (p *Pipeline) Start(ctx context.Context) error {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return fmt.Errorf("pipeline already running")
	}
	p.running = true
	p.mu.Unlock()

	p.log.Info("Starting factstore pipeline")

	go p.processingLoop(ctx)
	return nil
}

// Stop stops the pipeline
func (p *Pipeline) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return
	}

	close(p.stopChan)
	p.running = false
	p.log.Info("Factstore pipeline stopped")
}

func (p *Pipeline) processingLoop(ctx context.Context) {
	ticker := time.NewTicker(p.config.ProcessingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.stopChan:
			return
		case <-ticker.C:
			if err := p.ProcessPendingDocuments(ctx); err != nil {
				p.log.WithError(err).Error("Error processing pending documents")
			}
		}
	}
}

// =============================================================================
// HELPERS
// =============================================================================

func computeHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// generateCanonicalKey produces a deterministic dedup key for a fact.
// Facts with the same canonical key across different trial tables (e.g., ARISTOTLE vs AMPLIFY)
// are considered duplicates. At projection time (Phase I), only the highest-confidence
// instance is projected to clinical_facts.
//
// Key structure: hash(rxcui|factType|normalizedIdentifier)
//   - SAFETY_SIGNAL: identifier = MedDRA PT code (or conditionName if no PT)
//   - INTERACTION: identifier = interacting drug + effect
//   - ORGAN_IMPAIRMENT: identifier = organ + threshold
//   - Others: identifier = description
func generateCanonicalKey(rxcui, factType string, item interface{}) string {
	identifier := ""
	switch v := item.(type) {
	case KBSafetySignalContent:
		// Prefer MedDRA PT code (deterministic), fall back to normalized name
		if v.MedDRAPT != "" {
			identifier = v.MedDRAPT
		} else if v.MedDRAName != "" {
			identifier = strings.ToLower(v.MedDRAName)
		} else {
			identifier = strings.ToLower(v.ConditionName)
		}
	case KBInteractionContent:
		// Use only interactantName for dedup (not clinicalEffect) so that
		// STRUCTURED_PARSE and LLM_FALLBACK facts for the same drug pair
		// produce the same canonical key and are properly deduplicated.
		identifier = strings.ToLower(strings.TrimSpace(v.InteractantName))
	case KBOrganImpairmentContent:
		identifier = strings.ToLower(v.Organ + "|" + v.ImpairmentLevel)
	case KBReproductiveSafetyContent:
		identifier = strings.ToLower(v.Category + "|" + v.RiskLevel)
	case KBLabReferenceContent:
		identifier = strings.ToLower(v.LabName + "|" + v.Unit)
	case KBFormularyContent:
		identifier = strings.ToLower(v.NDCCode + "|" + v.PackageForm)
	default:
		// Fallback: marshal to JSON and hash
		data, _ := json.Marshal(item)
		identifier = string(data)
	}

	raw := fmt.Sprintf("%s|%s|%s", rxcui, factType, identifier)
	hash := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(hash[:16]) // 32-char hex, enough for dedup
}

// isEmptyContent returns true if the parsed item has no meaningful clinical data.
// Empty facts waste governance reviewer time and pollute clinical_facts.
func isEmptyContent(item interface{}) bool {
	switch v := item.(type) {
	case KBSafetySignalContent:
		return v.ConditionName == "" && v.Description == ""
	case KBInteractionContent:
		return v.InteractantName == "" && v.ClinicalEffect == ""
	case KBOrganImpairmentContent:
		return v.Organ == "" && v.Action == ""
	case KBReproductiveSafetyContent:
		return v.Category == "" && v.RiskLevel == ""
	case KBLabReferenceContent:
		return v.LabName == ""
	case KBFormularyContent:
		return v.NDCCode == "" && v.PackageForm == ""
	default:
		return false
	}
}

// isContextuallyExcluded filters terms that are valid MedDRA entries but contextually
// impossible as adverse drug reactions. These fall into three categories:
//
//  1. Surgical procedures: "Eventration repair", "Mastectomy" — procedures, not reactions
//  2. Genetic diseases: "Fatal familial insomnia" — inherited, not drug-caused
//  3. Pregnancy-only terms on non-pregnancy drugs — obstetric context misclassified
//
// This is a defense-in-depth layer: section gating blocks most noise, but some terms
// leak through from Adverse Reactions tables that list background conditions.
func isContextuallyExcluded(factType string, item interface{}) bool {
	if factType != "SAFETY_SIGNAL" {
		return false
	}
	ss, ok := item.(KBSafetySignalContent)
	if !ok {
		return false
	}
	name := strings.ToLower(ss.ConditionName)
	if name == "" {
		return false
	}

	// Surgical procedures — cannot be adverse drug reactions
	surgicalTerms := []string{
		"repair", "mastectomy", "transplant", "amputation",
		"arthroplasty", "colectomy", "gastrectomy", "nephrectomy",
		"thyroidectomy", "appendectomy", "cholecystectomy",
	}
	for _, term := range surgicalTerms {
		if strings.Contains(name, term) {
			return true
		}
	}

	// Genetic/inherited diseases — not drug-caused
	geneticTerms := []string{
		"fatal familial insomnia", "huntington", "cystic fibrosis",
		"sickle cell", "down syndrome", "turner syndrome",
		"marfan syndrome", "ehlers-danlos",
	}
	for _, term := range geneticTerms {
		if strings.Contains(name, term) {
			return true
		}
	}

	// Pregnancy-specific obstetric terms misclassified as AE
	obstetricTerms := []string{
		"foetal heart rate", "fetal heart rate", "baseline variability",
		"uterine contraction", "labour", "labor induction",
	}
	for _, term := range obstetricTerms {
		if strings.Contains(name, term) {
			return true
		}
	}

	return false
}

// isSafetyBlockedSection returns true for LOINC sections that should NOT produce
// SAFETY_SIGNAL facts. These sections contain tables with clinical trial endpoints,
// exclusion criteria, or pregnancy-specific data that get misclassified as adverse events.
//
// Examples of noise from blocked sections:
//   - 34092-7 (Clinical Studies): "Fatal familial insomnia" = exclusion criterion, not AE
//   - 34077-8 (Pregnancy): "Baseline foetal heart rate variability disorder" = obstetric context
//   - 34076-0 (Patient Info): Lay-language summaries, not structured AE data
//
// Whitelisted sections (parsed for safety signals):
//   - 34084-4 (Adverse Reactions) — primary source
//   - 34066-1 (Boxed Warning) — critical safety data
//   - 34067-9 (Warnings and Precautions) — important safety context
//   - 43685-7 (Warnings and Precautions - new format)
func isSafetyBlockedSection(loincCode string) bool {
	// Whitelist approach: only allow known-safe sections
	allowed := map[string]bool{
		"34084-4": true, // Adverse Reactions
		"34066-1": true, // Boxed Warning
		"34067-9": true, // Warnings and Precautions
		"43685-7": true, // Warnings and Precautions (alternate)
		"34068-7": true, // Dosage and Administration (for ORGAN_IMPAIRMENT)
		"34073-7": true, // Drug Interactions
		"34069-5": true, // How Supplied
		"42229-5": true, // SPL Unclassified / Hepatic Impairment subsection
		"43684-0": true, // Use in Specific Populations (renal/hepatic impairment parent)
		"42232-0": true, // Renal Impairment (subsection of Use in Specific Populations)
	}
	// If section is in the whitelist, it's NOT blocked
	if allowed[loincCode] {
		return false
	}
	// Explicitly blocked sections (for clarity in logs)
	blocked := map[string]bool{
		"34092-7": true, // Clinical Studies — exclusion criteria noise
		"34077-8": true, // Pregnancy — obstetric context misclassification
		"34076-0": true, // Patient Information — lay summaries
		"34090-1": true, // Clinical Pharmacology — PK data, not AEs
	}
	if blocked[loincCode] {
		return true
	}
	// Unknown sections: allow by default (conservative — don't lose data)
	return false
}

// isProseScannableSection returns true for sections where the MedDRA prose scanner should run.
// These are sections that contain adverse event descriptions in narrative/prose form.
// Drug Interactions (34073-7) is excluded — it has its own DDI grammar extractor.
func isProseScannableSection(loincCode string) bool {
	switch loincCode {
	case "34084-4": // Adverse Reactions — primary AE section
		return true
	case "34066-1": // Boxed Warning — critical safety prose
		return true
	case "34067-9": // Warnings and Precautions
		return true
	case "43685-7": // Warnings and Precautions (alternate LOINC)
		return true
	case "43684-0": // Precautions
		return true
	default:
		return false
	}
}

// P5.2: isReproductiveSection checks if a LOINC code is a pregnancy/lactation section.
func isReproductiveSection(loincCode string) bool {
	return loincCode == "34077-8" || // Pregnancy
		loincCode == "34080-2" // Nursing Mothers
}

// P5.2: extractReproductiveSafetyFacts extracts REPRODUCTIVE_SAFETY facts from
// pregnancy/lactation sections using prose-based parsing.
func (p *Pipeline) extractReproductiveSafetyFacts(
	ctx context.Context,
	sourceDoc *SourceDocument,
	sourceSection *SourceSection,
	routed *dailymed.RoutedSection,
	rxcui string,
) ([]*DerivedFact, error) {
	var facts []*DerivedFact

	parser := NewContentParser()
	content := parser.ParseReproductiveSafetyFromProse(routed.PlainText, sourceSection.SectionCode)
	if content == nil {
		return facts, nil
	}

	factData, _ := json.Marshal(content)
	canonicalKey := generateCanonicalKey(rxcui, "REPRODUCTIVE_SAFETY", *content)
	evidenceSpans, _ := json.Marshal([]string{fmt.Sprintf("Section: %s", sourceSection.SectionName)})

	fact := &DerivedFact{
		SourceDocumentID:     sourceDoc.ID,
		SourceSectionID:      sourceSection.ID,
		TargetKB:             "KB-4",
		FactType:             "REPRODUCTIVE_SAFETY",
		FactKey:              canonicalKey,
		FactData:             factData,
		ExtractionMethod:     "STRUCTURED_PARSE",
		ExtractionConfidence: 0.75, // Prose extraction — moderate confidence
		EvidenceSpans:        evidenceSpans,
		ConsensusAchieved:    true,
		GovernanceStatus:     "DRAFT",
	}
	facts = append(facts, fact)

	p.log.WithFields(logrus.Fields{
		"drug":      sourceDoc.DrugName,
		"category":  content.Category,
		"riskLevel": content.RiskLevel,
	}).Info("Extracted REPRODUCTIVE_SAFETY from prose")

	return facts, nil
}

func mapTableTypeToFactType(tableType dailymed.TableType) string {
	// Map table types to fact types that match KB view expectations
	// KB views use: ORGAN_IMPAIRMENT, SAFETY_SIGNAL, INTERACTION, REPRODUCTIVE_SAFETY, FORMULARY, LAB_REFERENCE
	switch tableType {
	case dailymed.TableTypeGFRDosing, dailymed.TableTypeHepaticDosing:
		// P5.1: Re-enabled with confidence capped at 0.70 → always routes to PENDING_REVIEW.
		// SPL labels are descriptive ("Use with caution"), not normative. CPIC/KDIGO are
		// authoritative, but SPL provides baseline extraction when authorities are absent.
		return "ORGAN_IMPAIRMENT"
	case dailymed.TableTypeDosing:
		// General dosing tables (efficacy, PK, trial results) are NOT organ impairment.
		// They contain HbA1c outcomes, body weight changes, maintenance doses — none of
		// which have GFR/CrCl thresholds. Dropping them prevents empty-shell facts.
		return "" // Explicitly drop — not extractable by any current parser
	case dailymed.TableTypeAdverseEvents, dailymed.TableTypeContraindications:
		return "SAFETY_SIGNAL" // Matches kb4_safety_signals view
	case dailymed.TableTypeDDI:
		return "INTERACTION" // Matches kb5_interactions view
	case dailymed.TableTypePK:
		return "" // PK tables disabled — LAB_REFERENCE extraction parked
	case dailymed.TableTypeEfficacy:
		return "" // Efficacy/outcome tables produce NO facts — prevents LIFE/RENAAL/CAPRIE contamination
	default:
		return "SAFETY_SIGNAL" // Default to safety signal for unknown types
	}
}

func calculatePriority(confidence float64, factType string) string {
	// Critical safety facts always get high priority
	if isCriticalSafetyFact(factType) {
		return "HIGH"
	}

	// Priority based on confidence
	if confidence >= 0.80 {
		return "NORMAL"
	} else if confidence >= 0.70 {
		return "NORMAL"
	}
	return "LOW"
}

// =============================================================================
// P6.1: FACT STABILITY ASSIGNMENT
// Enriches FactData with volatility contract metadata at creation time.
// The _stability key tells downstream consumers how long to trust this fact
// and what triggers re-extraction (FDA label update, new study, etc.).
// =============================================================================

// enrichFactWithStability adds a _stability JSON key to the fact's FactData.
// This embeds volatility/half-life/revalidation metadata alongside the clinical content.
func enrichFactWithStability(fact *DerivedFact) {
	factType := FactType(fact.FactType)
	stability := DefaultStability(factType)

	// Build stability metadata
	stabilityMeta := map[string]interface{}{
		"volatility":           string(stability.Volatility),
		"expectedHalfLife":     string(stability.ExpectedHalfLife),
		"autoRevalidationDays": stability.AutoRevalidationDays,
		"changeTriggers":       stability.ChangeTriggers,
		"lastRefreshedAt":      stability.LastRefreshedAt.Format(time.RFC3339),
		"refreshSource":        stability.RefreshSource,
	}
	if stability.StaleAfter != nil {
		stabilityMeta["staleAfter"] = stability.StaleAfter.Format(time.RFC3339)
	}

	// Merge into existing FactData
	var factDataMap map[string]interface{}
	if err := json.Unmarshal(fact.FactData, &factDataMap); err != nil {
		// FactData might be a JSON array (e.g., safety signals) — wrap it
		factDataMap = map[string]interface{}{
			"content": json.RawMessage(fact.FactData),
		}
	}
	factDataMap["_stability"] = stabilityMeta

	if enriched, err := json.Marshal(factDataMap); err == nil {
		fact.FactData = enriched
	}
}

func isCriticalSafetyFact(factType string) bool {
	criticalTypes := map[string]bool{
		"BLACK_BOX_WARNING":   true,
		"CONTRAINDICATION":    true,
		"QT_PROLONGATION":     true,
		"HEPATOTOXICITY":      true,
		"SERIOUS_ADVERSE":     true,
		"RENAL_DOSE_ADJUST":   true,
		"HEPATIC_DOSE_ADJUST": true,
	}
	return criticalTypes[factType]
}
