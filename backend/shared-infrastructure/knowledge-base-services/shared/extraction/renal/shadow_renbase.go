// Package renal provides the Shadow Renbase Extractor for renal dosing.
//
// Phase 3c.2: Shadow Renbase Extractor
// Authority Level: SPL-DERIVED (FDA labels are authoritative source)
//
// This extractor builds renal dosing guidance from FDA Structured Product Labels.
// It replaces expensive commercial databases like Renbase with freely available FDA data.
//
// EXTRACTION HIERARCHY (Navigation Rules):
// 1. Tabular data: Parse tables directly (HIGH confidence)
// 2. Structured patterns: Use regex for common patterns (MEDIUM confidence)
// 3. LLM consensus: When prose requires interpretation (2-of-3 required)
// 4. Human review: When LLMs disagree (CRITICAL facts only)
//
// KEY PRINCIPLE: "Table exists → PARSE, don't interpret"
package renal

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/cardiofit/shared/datasources/dailymed"
	"github.com/cardiofit/shared/extraction/consensus"
	"github.com/cardiofit/shared/extraction/llm"
	"github.com/cardiofit/shared/extraction/spl"
)

// =============================================================================
// SHADOW RENBASE EXTRACTOR
// =============================================================================

// ShadowRenbaseExtractor builds renal dosing from FDA SPL labels
type ShadowRenbaseExtractor struct {
	splFetcher       *dailymed.SPLFetcher
	rxnavResolver    dailymed.RxNavSPLResolver
	loincParser      *spl.LOINCSectionParser
	tabularHarvester *spl.TabularHarvester
	consensusEngine  *consensus.Engine
	factStore        FactStoreWriter
}

// FactStoreWriter interface for storing extracted facts
type FactStoreWriter interface {
	StoreFact(ctx context.Context, fact interface{}) error
}

// Config contains extractor configuration
type Config struct {
	// SPLFetcher is the DailyMed SPL fetcher (required)
	SPLFetcher *dailymed.SPLFetcher

	// RxNavResolver resolves RxCUI to SPL SetID (required for RxCUI lookups)
	RxNavResolver dailymed.RxNavSPLResolver

	// LOINCSectionParser is the LOINC section parser (required)
	LOINCSectionParser *spl.LOINCSectionParser

	// TabularHarvester is the table extractor (required)
	TabularHarvester *spl.TabularHarvester

	// ConsensusEngine is the LLM consensus engine (required)
	ConsensusEngine *consensus.Engine

	// FactStore is the fact storage backend (optional)
	FactStore FactStoreWriter
}

// NewShadowRenbaseExtractor creates a new renal dosing extractor
func NewShadowRenbaseExtractor(config Config) *ShadowRenbaseExtractor {
	return &ShadowRenbaseExtractor{
		splFetcher:       config.SPLFetcher,
		rxnavResolver:    config.RxNavResolver,
		loincParser:      config.LOINCSectionParser,
		tabularHarvester: config.TabularHarvester,
		consensusEngine:  config.ConsensusEngine,
		factStore:        config.FactStore,
	}
}

// =============================================================================
// RENAL DOSE ADJUSTMENT
// =============================================================================

// RenalDoseAdjustment represents extracted renal dosing information
type RenalDoseAdjustment struct {
	// ─────────────────────────────────────────────────────────────────────────
	// DRUG IDENTIFICATION
	// ─────────────────────────────────────────────────────────────────────────

	// RxCUI is the RxNorm Concept Unique Identifier
	RxCUI string `json:"rxcui"`

	// DrugName is the drug name
	DrugName string `json:"drugName"`

	// GenericName is the generic drug name
	GenericName string `json:"genericName,omitempty"`

	// ─────────────────────────────────────────────────────────────────────────
	// GFR-BASED DOSING
	// ─────────────────────────────────────────────────────────────────────────

	// HasRenalDosing indicates if renal dosing guidance exists
	HasRenalDosing bool `json:"hasRenalDosing"`

	// GFRBands contains dosing by GFR range
	GFRBands []GFRBand `json:"gfrBands,omitempty"`

	// DialysisGuidance contains dialysis-specific recommendations
	DialysisGuidance *DialysisGuidance `json:"dialysisGuidance,omitempty"`

	// ─────────────────────────────────────────────────────────────────────────
	// EXTRACTION METADATA
	// ─────────────────────────────────────────────────────────────────────────

	// ExtractionType indicates how the data was extracted
	ExtractionType ExtractionType `json:"extractionType"`

	// Confidence is the extraction confidence (0.0-1.0)
	Confidence float64 `json:"confidence"`

	// Citations are quotes from the source text
	Citations []string `json:"citations,omitempty"`

	// ─────────────────────────────────────────────────────────────────────────
	// SOURCE LINEAGE
	// ─────────────────────────────────────────────────────────────────────────

	// SourceSetID is the SPL document SetID
	SourceSetID string `json:"sourceSetId"`

	// SourceSection is the LOINC code of the source section
	SourceSection string `json:"sourceSection"`

	// SourceVersion is the SPL version number
	SourceVersion string `json:"sourceVersion,omitempty"`

	// ExtractedAt is when the extraction occurred
	ExtractedAt time.Time `json:"extractedAt"`

	// ─────────────────────────────────────────────────────────────────────────
	// GOVERNANCE
	// ─────────────────────────────────────────────────────────────────────────

	// NeedsReview indicates if human review is required
	NeedsReview bool `json:"needsReview"`

	// ReviewReason explains why review is needed
	ReviewReason string `json:"reviewReason,omitempty"`
}

// GFRBand represents dosing for a GFR range
type GFRBand struct {
	// MinGFR is the minimum GFR (inclusive), 0 = no minimum
	MinGFR float64 `json:"minGfr"`

	// MaxGFR is the maximum GFR (exclusive), 999 = no maximum
	MaxGFR float64 `json:"maxGfr"`

	// Stage is the CKD stage name (e.g., "Stage 3a", "Stage 4")
	Stage string `json:"stage,omitempty"`

	// Action is the recommended action
	Action DoseAction `json:"action"`

	// RecommendedDose is the dose recommendation (e.g., "50% of normal", "250mg BID")
	RecommendedDose string `json:"recommendedDose,omitempty"`

	// MaxDose is the maximum allowed dose
	MaxDose string `json:"maxDose,omitempty"`

	// Frequency is the dosing frequency
	Frequency string `json:"frequency,omitempty"`

	// Notes are additional instructions
	Notes string `json:"notes,omitempty"`
}

// DoseAction represents the type of dose adjustment
type DoseAction string

const (
	// ActionNoChange means no dose adjustment needed
	ActionNoChange DoseAction = "NO_CHANGE"

	// ActionReduce means reduce the dose
	ActionReduce DoseAction = "REDUCE"

	// ActionExtendInterval means extend the dosing interval
	ActionExtendInterval DoseAction = "EXTEND_INTERVAL"

	// ActionReduceAndExtend means both reduce dose and extend interval
	ActionReduceAndExtend DoseAction = "REDUCE_AND_EXTEND"

	// ActionAvoid means avoid use if possible
	ActionAvoid DoseAction = "AVOID"

	// ActionContraindicated means drug is contraindicated
	ActionContraindicated DoseAction = "CONTRAINDICATED"

	// ActionMonitor means use with increased monitoring
	ActionMonitor DoseAction = "MONITOR"
)

// DialysisGuidance contains dialysis-specific recommendations
type DialysisGuidance struct {
	// Hemodialysis guidance
	Hemodialysis string `json:"hemodialysis,omitempty"`

	// HemodialysisSupplemental dose after dialysis
	HemodialysisSupplemental string `json:"hemodialysisSupplemental,omitempty"`

	// Dialyzability indicates if drug is removed by dialysis
	Dialyzability string `json:"dialyzability,omitempty"` // "High", "Moderate", "Low", "Not dialyzable"

	// PeritonealDialysis guidance
	PeritonealDialysis string `json:"peritonealDialysis,omitempty"`

	// CRRT guidance (Continuous Renal Replacement Therapy)
	CRRT string `json:"crrt,omitempty"`
}

// ExtractionType indicates how the data was extracted
type ExtractionType string

const (
	// ExtractionTable means data was parsed from a structured table
	ExtractionTable ExtractionType = "TABLE_PARSE"

	// ExtractionRegex means data was extracted via regex patterns
	ExtractionRegex ExtractionType = "REGEX_PARSE"

	// ExtractionLLM means data was extracted via LLM consensus
	ExtractionLLM ExtractionType = "LLM_CONSENSUS"

	// ExtractionNone means no renal dosing data was found
	ExtractionNone ExtractionType = "NO_DATA"

	// ExtractionHuman means data was manually curated
	ExtractionHuman ExtractionType = "HUMAN_CURATED"
)

// =============================================================================
// EXTRACTION METHODS
// =============================================================================

// Extract retrieves renal dosing for a drug by RxCUI
func (e *ShadowRenbaseExtractor) Extract(ctx context.Context, rxcui string) (*RenalDoseAdjustment, error) {
	// Step 1: Fetch SPL document via RxNav resolver
	splDoc, err := e.splFetcher.FetchByRxCUI(ctx, rxcui, e.rxnavResolver)
	if err != nil {
		return nil, fmt.Errorf("fetching SPL for RxCUI %s: %w", rxcui, err)
	}

	return e.ExtractFromSPL(ctx, rxcui, splDoc)
}

// ExtractFromSPL extracts renal dosing from an SPL document
func (e *ShadowRenbaseExtractor) ExtractFromSPL(ctx context.Context, rxcui string, splDoc *dailymed.SPLDocument) (*RenalDoseAdjustment, error) {
	result := &RenalDoseAdjustment{
		RxCUI:         rxcui,
		SourceSetID:   splDoc.SetID.Extension,
		SourceVersion: strconv.Itoa(splDoc.VersionNumber.Value),
		ExtractedAt:   time.Now(),
	}

	// Step 2: Get Dosage & Administration section (LOINC 34068-7)
	dosageSection := splDoc.GetSection(dailymed.LOINCDosageAdministration)
	if dosageSection == nil {
		// Try Clinical Pharmacology section as fallback (LOINC 34090-1)
		dosageSection = splDoc.GetSection(dailymed.LOINCClinicalPharm)
	}

	if dosageSection == nil {
		result.HasRenalDosing = false
		result.ExtractionType = ExtractionNone
		result.Confidence = 1.0 // High confidence in "no data"
		return result, nil
	}

	result.SourceSection = dosageSection.Code.Code

	// Step 3: Try tabular extraction first (Navigation Rule 2)
	tables := e.tabularHarvester.HarvestAllTables(dosageSection)
	gfrTable := e.findGFRTable(tables)

	if gfrTable != nil {
		// High confidence: data from structured table
		return e.buildFromTable(result, gfrTable, splDoc)
	}

	// Step 4: Try regex patterns for common formats
	regexResult := e.extractWithRegex(dosageSection)
	if regexResult != nil && len(regexResult.GFRBands) > 0 {
		// Medium confidence: data from regex patterns
		regexResult.RxCUI = rxcui
		regexResult.SourceSetID = splDoc.SetID.Extension
		regexResult.SourceSection = dosageSection.Code.Code
		regexResult.ExtractedAt = time.Now()
		return regexResult, nil
	}

	// Step 5: Fall back to LLM consensus (Navigation Rule 3)
	return e.extractWithLLM(ctx, result, splDoc, dosageSection)
}

// =============================================================================
// TABLE-BASED EXTRACTION
// =============================================================================

// findGFRTable finds a GFR dosing table from harvested tables
func (e *ShadowRenbaseExtractor) findGFRTable(tables []spl.HarvestedTable) *spl.HarvestedTable {
	for i, table := range tables {
		if table.Type == spl.TableTypeGFRDose {
			return &tables[i]
		}
	}
	return nil
}

// buildFromTable constructs renal dosing from a parsed table
func (e *ShadowRenbaseExtractor) buildFromTable(result *RenalDoseAdjustment, table *spl.HarvestedTable, splDoc *dailymed.SPLDocument) (*RenalDoseAdjustment, error) {
	result.HasRenalDosing = true
	result.ExtractionType = ExtractionTable
	result.Confidence = 0.95 // High confidence for tabular data
	result.GFRBands = make([]GFRBand, 0)

	for _, row := range table.Rows {
		if row.Parsed != nil {
			band := GFRBand{
				Action: e.parseAction(row.Parsed.Action),
			}

			if row.Parsed.GFRMin != nil {
				band.MinGFR = *row.Parsed.GFRMin
			}
			if row.Parsed.GFRMax != nil {
				band.MaxGFR = *row.Parsed.GFRMax
			} else {
				band.MaxGFR = 999 // No upper limit
			}

			if row.Parsed.DoseAdjust != "" {
				band.RecommendedDose = row.Parsed.DoseAdjust
			}
			if row.Parsed.MaxDose != "" {
				band.MaxDose = row.Parsed.MaxDose
			}
			if row.Parsed.Frequency != "" {
				band.Frequency = row.Parsed.Frequency
			}
			if row.Parsed.Notes != "" {
				band.Notes = row.Parsed.Notes
			}

			// Add citation from row condition
			result.Citations = append(result.Citations, row.Condition)

			result.GFRBands = append(result.GFRBands, band)
		}
	}

	// Assign CKD stages to bands
	e.assignCKDStages(result.GFRBands)

	return result, nil
}

// parseAction converts a string action to DoseAction
func (e *ShadowRenbaseExtractor) parseAction(action string) DoseAction {
	switch strings.ToUpper(action) {
	case "NO_CHANGE", "NO CHANGE", "NO ADJUSTMENT":
		return ActionNoChange
	case "REDUCE", "DECREASE":
		return ActionReduce
	case "EXTEND_INTERVAL", "EXTEND INTERVAL":
		return ActionExtendInterval
	case "AVOID":
		return ActionAvoid
	case "CONTRAINDICATED":
		return ActionContraindicated
	case "MONITOR":
		return ActionMonitor
	default:
		return ActionReduce // Default to reduce if unspecified
	}
}

// assignCKDStages assigns CKD stage names to GFR bands
func (e *ShadowRenbaseExtractor) assignCKDStages(bands []GFRBand) {
	for i := range bands {
		bands[i].Stage = getCKDStage(bands[i].MinGFR, bands[i].MaxGFR)
	}
}

// getCKDStage returns the CKD stage for a GFR range
func getCKDStage(min, max float64) string {
	// Use midpoint for classification
	mid := (min + max) / 2
	if max >= 999 {
		mid = min + 15 // Assume reasonable range above min
	}

	switch {
	case mid >= 90:
		return "Stage 1 (Normal)"
	case mid >= 60:
		return "Stage 2 (Mild)"
	case mid >= 45:
		return "Stage 3a (Mild-Moderate)"
	case mid >= 30:
		return "Stage 3b (Moderate-Severe)"
	case mid >= 15:
		return "Stage 4 (Severe)"
	default:
		return "Stage 5 (ESRD)"
	}
}

// =============================================================================
// REGEX-BASED EXTRACTION
// =============================================================================

// Common patterns for renal dosing in drug labels
var (
	// Pattern: "CrCl 30-60 mL/min: reduce dose by 50%"
	gfrDosePattern = regexp.MustCompile(`(?i)(CrCl|eGFR|GFR|creatinine\s+clearance)\s*[<>≤≥]?\s*(\d+)(?:\s*[-–to]+\s*(\d+))?\s*(mL/min)?[:\s]+(.{10,100}?)(?:\.|;|$)`)

	// Pattern: "In patients with renal impairment (CrCl < 30 mL/min), reduce dose"
	renalImpairmentPattern = regexp.MustCompile(`(?i)renal\s+impairment[^.]*(?:CrCl|eGFR|GFR)\s*[<>≤≥]\s*(\d+)[^.]*\.`)

	// Pattern: "Contraindicated in severe renal impairment"
	contraindicatedPattern = regexp.MustCompile(`(?i)contraindicated[^.]*(?:severe\s+)?renal[^.]*\.`)

	// Pattern: "No dosage adjustment required"
	noAdjustmentPattern = regexp.MustCompile(`(?i)no\s+(?:dosage|dose)?\s*adjustment\s+(?:is\s+)?(?:required|necessary|needed)`)

	// Pattern for dialysis: "Supplemental dose after hemodialysis"
	dialysisPattern = regexp.MustCompile(`(?i)(hemodialysis|dialysis|HD)[^.]*(?:supplement|dose|not\s+removed)[^.]*\.`)
)

// extractWithRegex attempts to extract renal dosing using regex patterns
func (e *ShadowRenbaseExtractor) extractWithRegex(section *dailymed.SPLSection) *RenalDoseAdjustment {
	content := section.GetRawText()
	result := &RenalDoseAdjustment{
		HasRenalDosing: false,
		ExtractionType: ExtractionRegex,
		Confidence:     0.75, // Medium confidence for regex
		GFRBands:       make([]GFRBand, 0),
		Citations:      make([]string, 0),
	}

	// Check for "no adjustment needed"
	if noAdjustmentPattern.MatchString(content) {
		result.HasRenalDosing = true
		result.GFRBands = append(result.GFRBands, GFRBand{
			MinGFR: 0,
			MaxGFR: 999,
			Action: ActionNoChange,
			Notes:  "No dosage adjustment required for renal impairment",
		})
		return result
	}

	// Check for contraindication
	if contraindicatedPattern.MatchString(content) {
		result.HasRenalDosing = true
		result.GFRBands = append(result.GFRBands, GFRBand{
			MinGFR: 0,
			MaxGFR: 30, // Assume severe = <30
			Action: ActionContraindicated,
			Stage:  "Stage 4-5 (Severe)",
		})
		match := contraindicatedPattern.FindString(content)
		result.Citations = append(result.Citations, match)
		return result
	}

	// Extract GFR-based dosing
	matches := gfrDosePattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) >= 6 {
			band := GFRBand{}

			// Parse GFR values
			if gfr1, err := strconv.ParseFloat(match[2], 64); err == nil {
				if match[3] != "" {
					// Range: "30-60"
					band.MinGFR = gfr1
					if gfr2, err := strconv.ParseFloat(match[3], 64); err == nil {
						band.MaxGFR = gfr2
					}
				} else {
					// Single value with operator
					band.MaxGFR = gfr1
					band.MinGFR = 0
				}
			}

			// Parse recommendation
			recommendation := strings.TrimSpace(match[5])
			band.Action = e.inferAction(recommendation)
			band.RecommendedDose = recommendation
			band.Stage = getCKDStage(band.MinGFR, band.MaxGFR)

			result.GFRBands = append(result.GFRBands, band)
			result.Citations = append(result.Citations, match[0])
			result.HasRenalDosing = true
		}
	}

	// Check for dialysis guidance
	dialysisMatches := dialysisPattern.FindAllString(content, -1)
	if len(dialysisMatches) > 0 {
		result.DialysisGuidance = &DialysisGuidance{
			Hemodialysis: strings.Join(dialysisMatches, "; "),
		}
		result.Citations = append(result.Citations, dialysisMatches...)
	}

	if result.HasRenalDosing {
		return result
	}

	return nil
}

// inferAction infers the dose action from recommendation text
func (e *ShadowRenbaseExtractor) inferAction(text string) DoseAction {
	lower := strings.ToLower(text)

	switch {
	case strings.Contains(lower, "contraindicated"):
		return ActionContraindicated
	case strings.Contains(lower, "avoid"):
		return ActionAvoid
	case strings.Contains(lower, "not recommended"):
		return ActionAvoid
	case strings.Contains(lower, "reduce") || strings.Contains(lower, "decrease"):
		return ActionReduce
	case strings.Contains(lower, "extend") || strings.Contains(lower, "interval"):
		return ActionExtendInterval
	case strings.Contains(lower, "monitor"):
		return ActionMonitor
	case strings.Contains(lower, "no adjustment") || strings.Contains(lower, "no change"):
		return ActionNoChange
	default:
		return ActionReduce
	}
}

// =============================================================================
// LLM-BASED EXTRACTION
// =============================================================================

// extractWithLLM uses LLM consensus for complex prose extraction
func (e *ShadowRenbaseExtractor) extractWithLLM(ctx context.Context, result *RenalDoseAdjustment, splDoc *dailymed.SPLDocument, section *dailymed.SPLSection) (*RenalDoseAdjustment, error) {
	// Build extraction request
	req := &llm.ExtractionRequest{
		FactType:   llm.FactTypeRenalDoseAdjust,
		SourceText: section.GetRawText(),
		SourceType: llm.SourceTypeSPLSection,
		SourceLOINC: section.Code.Code,
		Schema:     llm.RenalDoseSchema,
		DrugContext: &llm.DrugContext{
			RxCUI:    result.RxCUI,
			DrugName: splDoc.Title,
		},
		RequireCitations: true,
		StrictSchema:     true,
		Temperature:      0.0, // Deterministic
		MaxRetries:       2,
		RequestedAt:      time.Now(),
	}

	// Run consensus extraction
	consensusResult, err := e.consensusEngine.Extract(ctx, req)
	if err != nil {
		// If consensus fails, mark for human review
		result.HasRenalDosing = false
		result.ExtractionType = ExtractionNone
		result.Confidence = 0.0
		result.NeedsReview = true
		result.ReviewReason = fmt.Sprintf("Consensus extraction failed: %v", err)
		return result, nil
	}

	// Check if consensus was achieved
	if !consensusResult.Achieved {
		// Navigation Rule 3: LLMs disagree → HUMAN first
		result.HasRenalDosing = false
		result.ExtractionType = ExtractionLLM
		result.Confidence = consensusResult.MaxConfidence
		result.NeedsReview = true
		result.ReviewReason = fmt.Sprintf("LLM consensus not achieved (%d/%d agreed). Disagreements: %v",
			consensusResult.AgreementCount, consensusResult.TotalProviders,
			summarizeDisagreements(consensusResult.Disagreements))
		return result, nil
	}

	// Consensus achieved - parse the winning value
	result.ExtractionType = ExtractionLLM
	result.Confidence = consensusResult.Confidence

	// Parse the consensus result
	if err := e.parseConsensusResult(result, consensusResult); err != nil {
		result.NeedsReview = true
		result.ReviewReason = fmt.Sprintf("Failed to parse consensus result: %v", err)
		return result, nil
	}

	// Add citations from provider results
	for _, provResult := range consensusResult.ProviderResults {
		for _, cit := range provResult.Citations {
			result.Citations = append(result.Citations, cit.QuotedText)
		}
	}

	return result, nil
}

// parseConsensusResult extracts structured data from consensus result
func (e *ShadowRenbaseExtractor) parseConsensusResult(result *RenalDoseAdjustment, consensusResult *consensus.Result) error {
	// Convert winning value to JSON
	jsonData, err := json.Marshal(consensusResult.WinningValue)
	if err != nil {
		return fmt.Errorf("marshaling consensus value: %w", err)
	}

	// Parse into our schema
	var extracted struct {
		HasRenalDosing bool `json:"hasRenalDosing"`
		GFRBands       []struct {
			MinGFR          float64 `json:"minGFR"`
			MaxGFR          float64 `json:"maxGFR"`
			Action          string  `json:"action"`
			RecommendedDose string  `json:"recommendedDose"`
			MaxDose         string  `json:"maxDose"`
			Frequency       string  `json:"frequency"`
		} `json:"gfrBands"`
		DialysisGuidance *struct {
			Hemodialysis       string `json:"hemodialysis"`
			PeritonealDialysis string `json:"peritonealDialysis"`
			CRRT               string `json:"crrt"`
		} `json:"dialysisGuidance"`
	}

	if err := json.Unmarshal(jsonData, &extracted); err != nil {
		return fmt.Errorf("parsing consensus JSON: %w", err)
	}

	result.HasRenalDosing = extracted.HasRenalDosing

	// Convert GFR bands
	for _, band := range extracted.GFRBands {
		result.GFRBands = append(result.GFRBands, GFRBand{
			MinGFR:          band.MinGFR,
			MaxGFR:          band.MaxGFR,
			Action:          e.parseAction(band.Action),
			RecommendedDose: band.RecommendedDose,
			MaxDose:         band.MaxDose,
			Frequency:       band.Frequency,
			Stage:           getCKDStage(band.MinGFR, band.MaxGFR),
		})
	}

	// Convert dialysis guidance
	if extracted.DialysisGuidance != nil {
		result.DialysisGuidance = &DialysisGuidance{
			Hemodialysis:       extracted.DialysisGuidance.Hemodialysis,
			PeritonealDialysis: extracted.DialysisGuidance.PeritonealDialysis,
			CRRT:               extracted.DialysisGuidance.CRRT,
		}
	}

	return nil
}

// summarizeDisagreements creates a summary of disagreements for review
func summarizeDisagreements(disagreements []consensus.Disagreement) string {
	if len(disagreements) == 0 {
		return "no specific disagreements recorded"
	}

	var summary []string
	for _, d := range disagreements {
		summary = append(summary, fmt.Sprintf("%s: %v vs %v (%s)",
			d.Field, d.Value1, d.Value2, d.Severity))
	}
	return strings.Join(summary, "; ")
}

// =============================================================================
// BATCH EXTRACTION
// =============================================================================

// BatchExtractResult contains results for multiple drugs
type BatchExtractResult struct {
	Results    map[string]*RenalDoseAdjustment `json:"results"`
	Errors     map[string]string               `json:"errors"`
	TotalCount int                             `json:"totalCount"`
	SuccessCount int                           `json:"successCount"`
	FailedCount  int                           `json:"failedCount"`
	Duration   time.Duration                   `json:"duration"`
}

// BatchExtract extracts renal dosing for multiple drugs
func (e *ShadowRenbaseExtractor) BatchExtract(ctx context.Context, rxcuis []string) *BatchExtractResult {
	startTime := time.Now()

	result := &BatchExtractResult{
		Results:    make(map[string]*RenalDoseAdjustment),
		Errors:     make(map[string]string),
		TotalCount: len(rxcuis),
	}

	for _, rxcui := range rxcuis {
		select {
		case <-ctx.Done():
			result.Errors[rxcui] = "context cancelled"
			result.FailedCount++
			continue
		default:
		}

		extraction, err := e.Extract(ctx, rxcui)
		if err != nil {
			result.Errors[rxcui] = err.Error()
			result.FailedCount++
			continue
		}

		result.Results[rxcui] = extraction
		result.SuccessCount++

		// Store if fact store is available
		if e.factStore != nil {
			if err := e.factStore.StoreFact(ctx, extraction); err != nil {
				// Log but don't fail the extraction
				result.Errors[rxcui] = fmt.Sprintf("storage failed: %v", err)
			}
		}
	}

	result.Duration = time.Since(startTime)
	return result
}

// =============================================================================
// HELPER METHODS
// =============================================================================

// GetExtractionMethod returns the extraction method for auditing
func (r *RenalDoseAdjustment) GetExtractionMethod() string {
	return string(r.ExtractionType)
}

// IsHighConfidence returns true if extraction confidence is >= 0.85
func (r *RenalDoseAdjustment) IsHighConfidence() bool {
	return r.Confidence >= 0.85
}

// RequiresHumanReview returns true if human review is needed
func (r *RenalDoseAdjustment) RequiresHumanReview() bool {
	return r.NeedsReview || r.Confidence < 0.65
}

// GetCriticalBands returns GFR bands that require dose changes
func (r *RenalDoseAdjustment) GetCriticalBands() []GFRBand {
	var critical []GFRBand
	for _, band := range r.GFRBands {
		if band.Action != ActionNoChange && band.Action != ActionMonitor {
			critical = append(critical, band)
		}
	}
	return critical
}

// HasContraindication returns true if drug is contraindicated for any GFR range
func (r *RenalDoseAdjustment) HasContraindication() bool {
	for _, band := range r.GFRBands {
		if band.Action == ActionContraindicated {
			return true
		}
	}
	return false
}

// GetRecommendationForGFR returns the recommendation for a specific GFR value
func (r *RenalDoseAdjustment) GetRecommendationForGFR(gfr float64) *GFRBand {
	for i, band := range r.GFRBands {
		if gfr >= band.MinGFR && gfr < band.MaxGFR {
			return &r.GFRBands[i]
		}
	}
	return nil
}
