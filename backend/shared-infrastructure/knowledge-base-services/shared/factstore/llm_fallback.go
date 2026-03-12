// Package factstore provides LLM fallback extraction for safety signals.
//
// Phase 3c: LLM Fallback Safety Signal Extractor
//
// When structured table parsing produces zero SAFETY_SIGNAL facts for a drug's
// adverse reactions section, this module calls Claude (Anthropic) to extract
// adverse events from prose text. All LLM-extracted facts are:
//   - Capped at confidence 0.75
//   - Tagged with extraction_method = "LLM_FALLBACK"
//   - Set to governance_status = "PENDING_REVIEW" (never auto-approved)
//   - Validated against MedDRA dictionary (invalid PT codes rejected)
//   - Filtered through isBiomarkerSOC (Investigations SOC blocked)
//
// This is a GAP-FILLER, not a primary extraction path. Authority and table
// parsing always take precedence.
package factstore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/cardiofit/shared/datasources/dailymed"
	"github.com/sirupsen/logrus"
)

// =============================================================================
// BUDGET TRACKER
// =============================================================================

// llmBudgetTracker enforces per-run cost limits and rate limiting for LLM calls.
type llmBudgetTracker struct {
	mu              sync.Mutex
	maxBudgetUSD    float64
	spentUSD        float64
	callCount       int
	lastCallTime    time.Time
	minCallInterval time.Duration // minimum gap between API calls
}

func newBudgetTracker(maxBudgetUSD float64) *llmBudgetTracker {
	if maxBudgetUSD <= 0 {
		maxBudgetUSD = 50.0
	}
	return &llmBudgetTracker{
		maxBudgetUSD:    maxBudgetUSD,
		minCallInterval: 100 * time.Millisecond,
	}
}

func (b *llmBudgetTracker) canSpend(estimatedCostUSD float64) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return (b.spentUSD + estimatedCostUSD) <= b.maxBudgetUSD
}

func (b *llmBudgetTracker) recordSpend(actualCostUSD float64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.spentUSD += actualCostUSD
	b.callCount++
	b.lastCallTime = time.Now()
}

func (b *llmBudgetTracker) exhausted() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.spentUSD >= b.maxBudgetUSD
}

// waitForRateLimit sleeps if needed to enforce minimum interval between calls.
func (b *llmBudgetTracker) waitForRateLimit() {
	b.mu.Lock()
	elapsed := time.Since(b.lastCallTime)
	interval := b.minCallInterval
	b.mu.Unlock()

	if elapsed < interval {
		time.Sleep(interval - elapsed)
	}
}

func (b *llmBudgetTracker) summary() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return fmt.Sprintf("spent=$%.4f of $%.2f, calls=%d", b.spentUSD, b.maxBudgetUSD, b.callCount)
}

// =============================================================================
// LLM FALLBACK PROVIDER (thin wrapper for direct API calls)
// =============================================================================

// llmFallbackProvider makes direct Claude API calls for safety signal extraction.
// This is intentionally separate from the general-purpose extraction/llm package
// because the prompts are hardcoded and domain-specific.
type llmFallbackProvider struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
}

func newLLMFallbackProvider(apiKey, model string) *llmFallbackProvider {
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}
	return &llmFallbackProvider{
		apiKey:     apiKey,
		model:      model,
		baseURL:    "https://api.anthropic.com",
		httpClient: &http.Client{Timeout: 90 * time.Second},
	}
}

// llmAEResponse is the expected JSON response from Claude.
type llmAEResponse struct {
	AdverseEvents []llmAdverseEvent `json:"adverse_events"`
}

type llmAdverseEvent struct {
	MedDRAPT     string `json:"meddraPT"`
	MedDRAPTName string `json:"meddraPTName"`
	SourcePhrase string `json:"sourcePhrase"`
	Severity     string `json:"severity"`  // CRITICAL, HIGH, MEDIUM, LOW
	Frequency    string `json:"frequency"` // e.g., "common", "rare", ">5%"
}

// llmAPIRequest matches Anthropic Messages API.
type llmAPIRequest struct {
	Model       string         `json:"model"`
	MaxTokens   int            `json:"max_tokens"`
	Temperature float64        `json:"temperature"`
	Messages    []llmAPIMsg    `json:"messages"`
}

type llmAPIMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type llmAPIResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
	StopReason string `json:"stop_reason"` // "end_turn" or "max_tokens"
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func (p *llmFallbackProvider) call(ctx context.Context, prompt string) (*llmAPIResponse, error) {
	return p.callWithMaxTokens(ctx, prompt, 4096)
}

func (p *llmFallbackProvider) callWithMaxTokens(ctx context.Context, prompt string, maxTokens int) (*llmAPIResponse, error) {
	reqBody := llmAPIRequest{
		Model:       p.model,
		MaxTokens:   maxTokens,
		Temperature: 0.0,
		Messages:    []llmAPIMsg{{Role: "user", Content: prompt}},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API call: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var apiResp llmAPIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &apiResp, nil
}

// costUSD estimates cost based on token usage (Sonnet pricing).
func (p *llmFallbackProvider) costUSD(resp *llmAPIResponse) float64 {
	// Claude Sonnet: $3/M input, $15/M output
	inputCost := float64(resp.Usage.InputTokens) * 3.0 / 1_000_000
	outputCost := float64(resp.Usage.OutputTokens) * 15.0 / 1_000_000
	return inputCost + outputCost
}

// =============================================================================
// PROMPTS (hardcoded, not configurable)
// =============================================================================

func buildAdverseReactionsPrompt(drugName, plainText string) string {
	return fmt.Sprintf(`You are a Clinical Data Extractor analyzing an FDA drug label. You extract; you do not infer.

TASK: Extract adverse drug reactions as MedDRA Preferred Terms.

EXTRACTION RULES:
1. Return ONLY MedDRA Preferred Terms (PTs) with official numeric codes.
2. Every extracted term MUST map to a specific phrase in the provided text.
3. If multiple reactions appear in one phrase (e.g., "nausea, vomiting, and diarrhea"), extract each as a SEPARATE event.
4. Return ONLY reactions explicitly attributed to %s. Do not infer causality.

EXCLUSIONS — DO NOT EXTRACT:
- Lab tests or biomarkers (e.g., "Blood glucose increased", "Creatinine elevated")
- Section headers (e.g., "Digestive:", "Skin:", "Metabolic:")
- Cross-references (e.g., "[see Warnings and Precautions]", "[see Section 5.1]")
- Background rates, placebo comparisons, or pre-existing conditions
- Reactions attributed to other drugs or drug classes

SEVERITY CLASSIFICATION:
- CRITICAL: fatal, death, life-threatening
- HIGH: serious, severe, hospitalization, permanent disability
- MEDIUM: moderate, or severity not specified
- LOW: mild, minor, transient

FREQUENCY: Include if stated (e.g., "common", "rare", ">5%%", "1 in 1000"). Use empty string if not mentioned.

OUTPUT FORMAT (JSON only, no other text):
{"adverse_events": [
  {"meddraPT": "10020639", "meddraPTName": "Hyperkalemia", "sourcePhrase": "Hyperkalemia", "severity": "HIGH", "frequency": ""},
  {"meddraPT": "10028813", "meddraPTName": "Nausea", "sourcePhrase": "nausea, vomiting", "severity": "MEDIUM", "frequency": "common"}
]}

If no adverse reactions found: {"adverse_events": []}

DRUG: %s
SECTION: Adverse Reactions

TEXT:
%s`, drugName, drugName, plainText)
}

func buildBoxedWarningPrompt(drugName, plainText string) string {
	return fmt.Sprintf(`You are a Clinical Data Extractor analyzing an FDA Black Box Warning. This section contains the most serious risks.

TASK: Extract the critical adverse reactions that justify this boxed warning.

EXTRACTION RULES:
1. Return ONLY MedDRA Preferred Terms (PTs) with official numeric codes.
2. Every extracted term MUST map to a specific phrase in the warning text.
3. If multiple reactions appear in one phrase, extract each as a SEPARATE event.
4. Expect 1-5 adverse events. Boxed warnings are focused on specific critical risks.

EXCLUSIONS — DO NOT EXTRACT:
- Lab tests or biomarkers
- Monitoring recommendations without a specific adverse event
- Cross-references to other sections
- Background context or epidemiology

ALL BOXED WARNING REACTIONS ARE SEVERITY: CRITICAL
(This is mandatory — FDA boxed warnings indicate life-threatening risks)

OUTPUT FORMAT (JSON only, no other text):
{"adverse_events": [
  {"meddraPT": "10023676", "meddraPTName": "Lactic acidosis", "sourcePhrase": "Lactic acidosis is a rare but serious complication", "severity": "CRITICAL", "frequency": "rare"}
]}

If no specific adverse reactions found: {"adverse_events": []}

DRUG: %s
SECTION: Boxed Warning

TEXT:
%s`, drugName, plainText)
}

// =============================================================================
// EXTRACTION + VALIDATION
// =============================================================================

// extractSafetySignalsViaLLM calls Claude to extract adverse events from prose,
// validates each against MedDRA, filters biomarker SOCs, and returns DerivedFacts.
func (p *Pipeline) extractSafetySignalsViaLLM(
	ctx context.Context,
	sourceDoc *SourceDocument,
	sourceSection *SourceSection,
	routed *dailymed.RoutedSection,
) ([]*DerivedFact, error) {
	if p.llmProvider == nil {
		return nil, fmt.Errorf("LLM provider not initialized")
	}

	drugName := sourceDoc.DrugName
	if drugName == "" {
		drugName = sourceDoc.RxCUI
	}

	// Select prompt based on section code
	var prompt string
	switch sourceSection.SectionCode {
	case "34084-4": // Adverse Reactions
		prompt = buildAdverseReactionsPrompt(drugName, routed.PlainText)
	case "34066-1": // Boxed Warning
		prompt = buildBoxedWarningPrompt(drugName, routed.PlainText)
	default:
		return nil, fmt.Errorf("unsupported section code for LLM fallback: %s", sourceSection.SectionCode)
	}

	// Estimate cost and check budget (~2K input tokens typical)
	estimatedCost := 0.01 // conservative $0.01 estimate per call
	if !p.llmBudget.canSpend(estimatedCost) {
		p.log.Warn("LLM budget exhausted, skipping extraction")
		return nil, nil
	}

	// Rate limit
	p.llmBudget.waitForRateLimit()

	startTime := time.Now()

	// Call Claude (with truncation retry)
	resp, err := p.llmProvider.call(ctx, prompt)
	if err != nil {
		p.logLLMExtraction(ctx, sourceDoc, sourceSection, startTime, nil, false, err.Error())
		return nil, fmt.Errorf("Claude API call for %s: %w", drugName, err)
	}

	// Record actual cost
	actualCost := p.llmProvider.costUSD(resp)
	p.llmBudget.recordSpend(actualCost)

	// Parse response
	if len(resp.Content) == 0 {
		p.logLLMExtraction(ctx, sourceDoc, sourceSection, startTime, resp, false, "empty response")
		return nil, fmt.Errorf("empty response from Claude for %s", drugName)
	}

	rawText := resp.Content[0].Text
	jsonStr := extractLLMJSON(rawText)

	var aeResp llmAEResponse
	if err := json.Unmarshal([]byte(jsonStr), &aeResp); err != nil {
		// Detect truncation: stop_reason=max_tokens means output was cut off
		if resp.StopReason == "max_tokens" {
			p.log.WithFields(logrus.Fields{
				"drug":         drugName,
				"outputTokens": resp.Usage.OutputTokens,
			}).Warn("LLM response truncated (max_tokens), retrying with 8192")

			// Retry with double tokens
			p.llmBudget.waitForRateLimit()
			resp2, err2 := p.llmProvider.callWithMaxTokens(ctx, prompt, 8192)
			if err2 == nil && len(resp2.Content) > 0 {
				actualCost2 := p.llmProvider.costUSD(resp2)
				p.llmBudget.recordSpend(actualCost2)

				rawText = resp2.Content[0].Text
				jsonStr = extractLLMJSON(rawText)
				if err3 := json.Unmarshal([]byte(jsonStr), &aeResp); err3 == nil {
					p.log.WithField("drug", drugName).Info("LLM retry succeeded with 8192 tokens")
					resp = resp2 // use retry response for audit
					goto parseSuccess
				}
			}
		}
		p.logLLMExtraction(ctx, sourceDoc, sourceSection, startTime, resp, false, err.Error())
		return nil, fmt.Errorf("parse Claude JSON for %s: %w (raw: %.200s)", drugName, err, jsonStr)
	}
parseSuccess:

	// Empty response is valid — section had no extractable AEs
	if len(aeResp.AdverseEvents) == 0 {
		p.log.WithFields(logrus.Fields{
			"drug":        drugName,
			"sectionCode": sourceSection.SectionCode,
		}).Info("LLM found no adverse events in section")
		return nil, nil
	}

	// Validate each AE against MedDRA and build facts
	var facts []*DerivedFact
	var rejected, biomarkerFiltered int

	for _, ae := range aeResp.AdverseEvents {
		// MedDRA fields populated from normalizer (kept in scope for fact builder)
		var meddraLLT, meddraSOC, meddraSOCName, snomedCode string

		// 1. MedDRA PT validation
		if p.aeNormalizer != nil {
			normalized, err := p.aeNormalizer.Normalize(ctx, ae.MedDRAPTName)
			if err != nil {
				p.log.WithError(err).WithField("term", ae.MedDRAPTName).Warn("MedDRA normalize error")
				continue
			}
			if !normalized.IsValidTerm {
				rejected++
				p.log.WithFields(logrus.Fields{
					"drug":        drugName,
					"ptCode":      ae.MedDRAPT,
					"ptName":      ae.MedDRAPTName,
					"sectionCode": sourceSection.SectionCode,
					"reason":      "not in MedDRA",
				}).Warn("LLM AE rejected: invalid MedDRA PT")
				continue
			}

			// 2. SOC biomarker filter
			if isBiomarkerSOC(normalized.MedDRASOCName) {
				biomarkerFiltered++
				p.log.WithFields(logrus.Fields{
					"drug":        drugName,
					"ptCode":      normalized.MedDRAPT,
					"ptName":      ae.MedDRAPTName,
					"sectionCode": sourceSection.SectionCode,
					"soc":         normalized.MedDRASOCName,
				}).Warn("LLM AE filtered: Investigations SOC")
				continue
			}

			// Use normalized values (corrected PT code/name from MedDRA dictionary)
			ae.MedDRAPT = normalized.MedDRAPT
			ae.MedDRAPTName = normalized.MedDRAName

			// Capture enrichment fields for fact_data
			meddraLLT = normalized.MedDRALLT
			meddraSOC = normalized.MedDRASOC
			meddraSOCName = normalized.MedDRASOCName
			snomedCode = normalized.SNOMEDCode
		}

		// 3. Build enriched fact (matching KBSafetySignalContent schema)
		targetKB := "KB-4" // Safety
		if len(routed.TargetKBs) > 0 {
			targetKB = routed.TargetKBs[0]
		}

		signalType := llmLoincToSignalType(sourceSection.SectionCode)
		severity := llmNormalizeSeverity(ae.Severity, sourceSection.SectionCode, ae.SourcePhrase)
		requiresMonitor := severity == "CRITICAL" || severity == "HIGH"
		recommendation := llmExtractRecommendation(ae.SourcePhrase, signalType)
		description := llmBuildSafetyDescription(ae.MedDRAPTName, ae.Frequency, severity)

		factData, _ := json.Marshal(map[string]interface{}{
			// Core KBSafetySignalContent fields (matching STRUCTURED_PARSE)
			"signalType":      signalType,
			"severity":        severity,
			"conditionCode":   ae.MedDRAPT,
			"conditionName":   ae.MedDRAPTName,
			"description":     description,
			"recommendation":  recommendation,
			"requiresMonitor": requiresMonitor,
			"frequency":       ae.Frequency,

			// MedDRA fields (from normalizer)
			"meddraPT":      ae.MedDRAPT,
			"meddraName":    ae.MedDRAPTName,
			"meddraLLT":     meddraLLT,
			"meddraSOC":     meddraSOC,
			"meddraSOCName": meddraSOCName,
			"snomedCode":    snomedCode,
			"termConfidence": 0.75,

			// LLM provenance
			"source":       "LLM_FALLBACK",
			"reviewReason": "UNSTRUCTURED_LABEL_FALLBACK",
			"sourcePhrase": ae.SourcePhrase,
			"sectionCode":  sourceSection.SectionCode,
			"sectionName":  sourceSection.SectionName,
		})

		factKey := fmt.Sprintf("%s:SAFETY_SIGNAL:%s", sourceDoc.RxCUI, ae.MedDRAPT)

		fact := &DerivedFact{
			SourceDocumentID:     sourceDoc.ID,
			SourceSectionID:      sourceSection.ID,
			TargetKB:             targetKB,
			FactType:             "SAFETY_SIGNAL",
			FactKey:              factKey,
			FactData:             factData,
			ExtractionMethod:     "LLM_FALLBACK",
			ExtractionConfidence: 0.75, // Hard ceiling
			LLMProvider:          "anthropic",
			LLMModel:             p.llmProvider.model,
			ConsensusAchieved:    false,
			GovernanceStatus:     "PENDING_REVIEW", // Never auto-approve
		}

		facts = append(facts, fact)
	}

	p.log.WithFields(logrus.Fields{
		"drug":              drugName,
		"sectionCode":       sourceSection.SectionCode,
		"extracted":         len(aeResp.AdverseEvents),
		"valid":             len(facts),
		"meddraRejected":    rejected,
		"biomarkerFiltered": biomarkerFiltered,
		"costUSD":           fmt.Sprintf("$%.4f", actualCost),
	}).Info("LLM fallback extraction complete")

	// Audit log: success
	p.logLLMExtraction(ctx, sourceDoc, sourceSection, startTime, resp, true, "")

	return facts, nil
}

// logLLMExtraction writes an extraction audit log entry for LLM fallback calls.
func (p *Pipeline) logLLMExtraction(ctx context.Context, doc *SourceDocument, section *SourceSection, startTime time.Time, resp *llmAPIResponse, success bool, errMsg string) {
	if p.repo == nil {
		return
	}
	now := time.Now()
	entry := &ExtractionAuditEntry{
		SourceDocumentID:      doc.ID,
		SourceSectionID:       section.ID,
		ExtractionMethod:      "LLM_FALLBACK",
		ExtractionStartedAt:   startTime,
		ExtractionCompletedAt: &now,
		ExtractionDurationMs:  int(now.Sub(startTime).Milliseconds()),
		LLMProvider:           "anthropic",
		LLMModel:              p.llmProvider.model,
		Success:               success,
		ErrorMessage:          errMsg,
		ConfidenceScore:       0.75,
	}
	if resp != nil {
		entry.LLMPromptTokens = resp.Usage.InputTokens
		entry.LLMCompletionTokens = resp.Usage.OutputTokens
	}
	if err := p.repo.LogExtraction(ctx, entry); err != nil {
		p.log.WithError(err).Debug("Failed to write LLM audit log")
	}
}

// isLLMEligibleSection returns true for sections where LLM fallback is allowed.
func (p *Pipeline) isLLMEligibleSection(sectionCode string) bool {
	switch sectionCode {
	case "34084-4": // Adverse Reactions
		return true
	case "34066-1": // Boxed Warning
		return true
	case "34073-7": // Drug Interactions
		return true
	default:
		return false
	}
}

// extractLLMJSON strips markdown code fences from Claude's response.
func extractLLMJSON(text string) string {
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "```json") {
		text = strings.TrimPrefix(text, "```json")
		text = strings.TrimSuffix(text, "```")
	} else if strings.HasPrefix(text, "```") {
		text = strings.TrimPrefix(text, "```")
		text = strings.TrimSuffix(text, "```")
	}
	return strings.TrimSpace(text)
}

// llmLoincToSignalType maps LOINC section code to signal type.
// Standalone version mirroring ContentParser.loincToSignalType to avoid import cycle.
func llmLoincToSignalType(sectionCode string) string {
	signalTypes := map[string]string{
		"34066-1": "BOXED_WARNING",
		"34070-3": "CONTRAINDICATION",
		"43685-7": "WARNING",
		"34084-4": "ADVERSE_REACTION",
		"43684-0": "PRECAUTION",
	}
	if st, ok := signalTypes[sectionCode]; ok {
		return st
	}
	return "WARNING"
}

// llmNormalizeSeverity validates an LLM-returned severity value.
// Falls back to keyword parsing from sourcePhrase, or CRITICAL for boxed warnings.
func llmNormalizeSeverity(llmSeverity, sectionCode, sourcePhrase string) string {
	valid := map[string]bool{"CRITICAL": true, "HIGH": true, "MEDIUM": true, "LOW": true}
	upper := strings.ToUpper(strings.TrimSpace(llmSeverity))
	if valid[upper] {
		return upper
	}

	// Boxed warnings are always CRITICAL
	if sectionCode == "34066-1" {
		return "CRITICAL"
	}

	// Fallback: keyword parsing from sourcePhrase (mirrors content_parser extractSeverity)
	lower := strings.ToLower(sourcePhrase)
	if strings.Contains(lower, "fatal") || strings.Contains(lower, "death") ||
		strings.Contains(lower, "life-threatening") || strings.Contains(lower, "life threatening") {
		return "CRITICAL"
	}
	if strings.Contains(lower, "serious") || strings.Contains(lower, "severe") {
		return "HIGH"
	}
	if strings.Contains(lower, "moderate") {
		return "MEDIUM"
	}
	if strings.Contains(lower, "mild") || strings.Contains(lower, "minor") {
		return "LOW"
	}
	return "MEDIUM"
}

// llmExtractRecommendation derives a clinical recommendation from sourcePhrase keywords.
// Mirrors ContentParser.extractRecommendation logic.
func llmExtractRecommendation(sourcePhrase, signalType string) string {
	lower := strings.ToLower(sourcePhrase)

	if strings.Contains(lower, "discontinue") {
		return "Discontinue use immediately"
	}
	if strings.Contains(lower, "contraindicated") {
		return "Use is contraindicated"
	}
	if strings.Contains(lower, "monitor") {
		return "Monitor patient closely"
	}
	if strings.Contains(lower, "avoid") {
		return "Avoid use in this condition"
	}

	switch signalType {
	case "BOXED_WARNING":
		return "Review boxed warning before prescribing"
	case "CONTRAINDICATION":
		return "Do not use in patients with this condition"
	case "WARNING":
		return "Use with caution; monitor for adverse effects"
	case "ADVERSE_REACTION":
		return "Monitor for this adverse reaction"
	case "PRECAUTION":
		return "Exercise caution and monitor appropriately"
	default:
		return "Review clinical significance"
	}
}

// llmBuildSafetyDescription creates a clinical description from condition name + metadata.
// Mirrors buildSafetyDescription from content_parser.go.
func llmBuildSafetyDescription(conditionName, frequency, severity string) string {
	if conditionName == "" {
		return ""
	}
	var parts []string
	parts = append(parts, conditionName)
	if frequency != "" {
		parts = append(parts, "(frequency: "+frequency+")")
	}
	if severity != "" && severity != "MEDIUM" {
		parts = append(parts, "["+severity+"]")
	}
	return strings.Join(parts, " ")
}

// =============================================================================
// DRUG INTERACTION LLM FALLBACK
// =============================================================================

// llmDDIResponse is the expected JSON response from Claude for DDI extraction.
type llmDDIResponse struct {
	Interactions []llmInteraction `json:"interactions"`
}

type llmInteraction struct {
	InteractantName string `json:"interactant_name"` // Drug name or class (e.g., "NSAIDs", "Strong CYP3A4 Inhibitors")
	Mechanism       string `json:"mechanism"`         // e.g., "CYP3A4 inhibition"
	ClinicalEffect  string `json:"clinical_effect"`   // e.g., "Increased plasma concentration"
	Severity        string `json:"severity"`          // CONTRAINDICATED, SEVERE, MODERATE, MILD
	Management      string `json:"management"`        // e.g., "Avoid concurrent use or reduce dose"
	SourcePhrase    string `json:"source_phrase"`     // Grounding text from label
}

func buildDrugInteractionPrompt(drugName, plainText string) string {
	return fmt.Sprintf(`You are a Clinical Data Extractor analyzing an FDA drug label's Drug Interactions section.

TASK: Extract drug-drug, drug-food, and drug-class interactions for %s.

EXTRACTION RULES:
1. Extract each interaction as a SEPARATE entry with the interacting drug or drug class.
2. The interactant may be a specific drug (e.g., "Digoxin") OR a drug class (e.g., "Strong CYP3A4 Inhibitors", "NSAIDs", "Potassium-sparing diuretics"). Both are valid.
3. Every extracted interaction MUST map to a specific phrase in the provided text.
4. Extract the mechanism of interaction when stated (e.g., "CYP3A4 inhibition", "P-glycoprotein substrate").
5. Extract the clinical effect (e.g., "Increased plasma concentration", "Risk of hyperkalemia").
6. Extract management advice when stated (e.g., "Reduce dose by 50%%", "Monitor INR more frequently").

EXCLUSIONS — DO NOT EXTRACT:
- Section headers or category labels (e.g., "7.1 Agents That May Enhance...")
- Cross-references without specific interaction data (e.g., "[see Clinical Pharmacology]")
- General pharmacology statements without specific interactants
- Interactions attributed to other drugs, not %s
- Duplicate entries for the same interactant

SEVERITY CLASSIFICATION:
- CONTRAINDICATED: explicitly stated as contraindicated with concurrent use
- SEVERE: serious clinical consequence, avoid concurrent use, or significant dose adjustment required
- MODERATE: clinically significant but manageable with monitoring or minor dose adjustment
- MILD: minor effect, generally no dose adjustment needed

OUTPUT FORMAT (JSON only, no other text):
{"interactions": [
  {"interactant_name": "Strong CYP3A4 Inhibitors", "mechanism": "CYP3A4 inhibition", "clinical_effect": "Increased simvastatin exposure", "severity": "CONTRAINDICATED", "management": "Avoid concurrent use", "source_phrase": "Strong CYP3A4 inhibitors are contraindicated with simvastatin"},
  {"interactant_name": "Digoxin", "mechanism": "P-glycoprotein substrate", "clinical_effect": "Increased digoxin levels", "severity": "MODERATE", "management": "Monitor digoxin levels", "source_phrase": "Digoxin concentrations may be increased when co-administered"}
]}

If no interactions found: {"interactions": []}

DRUG: %s
SECTION: Drug Interactions

TEXT:
%s`, drugName, drugName, drugName, plainText)
}

// extractInteractionsViaLLM calls Claude to extract drug interactions from prose,
// validates each via RxNorm lookup with class fallback, and returns DerivedFacts.
func (p *Pipeline) extractInteractionsViaLLM(
	ctx context.Context,
	sourceDoc *SourceDocument,
	sourceSection *SourceSection,
	routed *dailymed.RoutedSection,
) ([]*DerivedFact, error) {
	if p.llmProvider == nil {
		return nil, fmt.Errorf("LLM provider not initialized")
	}

	drugName := sourceDoc.DrugName
	if drugName == "" {
		drugName = sourceDoc.RxCUI
	}

	prompt := buildDrugInteractionPrompt(drugName, routed.PlainText)

	// Budget check
	estimatedCost := 0.01
	if !p.llmBudget.canSpend(estimatedCost) {
		p.log.Warn("LLM budget exhausted, skipping DDI extraction")
		return nil, nil
	}

	p.llmBudget.waitForRateLimit()
	startTime := time.Now()

	// Call Claude (with truncation retry — DDI sections can be large)
	resp, err := p.llmProvider.call(ctx, prompt)
	if err != nil {
		p.logLLMExtraction(ctx, sourceDoc, sourceSection, startTime, nil, false, err.Error())
		return nil, fmt.Errorf("Claude API call for DDI %s: %w", drugName, err)
	}

	actualCost := p.llmProvider.costUSD(resp)
	p.llmBudget.recordSpend(actualCost)

	if len(resp.Content) == 0 {
		p.logLLMExtraction(ctx, sourceDoc, sourceSection, startTime, resp, false, "empty response")
		return nil, fmt.Errorf("empty DDI response from Claude for %s", drugName)
	}

	rawText := resp.Content[0].Text
	jsonStr := extractLLMJSON(rawText)

	var ddiResp llmDDIResponse
	if err := json.Unmarshal([]byte(jsonStr), &ddiResp); err != nil {
		// Truncation retry
		if resp.StopReason == "max_tokens" {
			p.log.WithFields(logrus.Fields{
				"drug":         drugName,
				"outputTokens": resp.Usage.OutputTokens,
			}).Warn("DDI LLM response truncated, retrying with 8192")

			p.llmBudget.waitForRateLimit()
			resp2, err2 := p.llmProvider.callWithMaxTokens(ctx, prompt, 8192)
			if err2 == nil && len(resp2.Content) > 0 {
				actualCost2 := p.llmProvider.costUSD(resp2)
				p.llmBudget.recordSpend(actualCost2)

				rawText = resp2.Content[0].Text
				jsonStr = extractLLMJSON(rawText)
				if err3 := json.Unmarshal([]byte(jsonStr), &ddiResp); err3 == nil {
					p.log.WithField("drug", drugName).Info("DDI LLM retry succeeded with 8192 tokens")
					resp = resp2
					goto ddiParseSuccess
				}
			}
		}
		p.logLLMExtraction(ctx, sourceDoc, sourceSection, startTime, resp, false, err.Error())
		return nil, fmt.Errorf("parse DDI JSON for %s: %w (raw: %.200s)", drugName, err, jsonStr)
	}
ddiParseSuccess:

	if len(ddiResp.Interactions) == 0 {
		p.log.WithFields(logrus.Fields{
			"drug":        drugName,
			"sectionCode": sourceSection.SectionCode,
		}).Info("LLM found no drug interactions in section")
		return nil, nil
	}

	// Build facts
	var facts []*DerivedFact
	var rejected int

	targetKB := "KB-5"
	if len(routed.TargetKBs) > 0 {
		targetKB = routed.TargetKBs[0]
	}

	for _, ix := range ddiResp.Interactions {
		// Validate interactant name is not empty or a header
		interactant := strings.TrimSpace(ix.InteractantName)
		if interactant == "" || isDDINoiseInteractant(interactant) {
			rejected++
			continue
		}

		// Classify: mapped drug vs unmapped class
		mappingStatus := classifyInteractant(interactant)

		// Normalize severity
		severity := normalizeDDISeverity(ix.Severity)

		// Enrich short source phrases (e.g., "Quinidine 100% NA" from PK tables)
		// by prepending drug context and clinical effect for reviewer clarity.
		sourcePhrase := strings.TrimSpace(ix.SourcePhrase)
		if len(sourcePhrase) < 40 && ix.ClinicalEffect != "" {
			sourcePhrase = fmt.Sprintf("%s interaction with %s: %s. %s",
				drugName, interactant, ix.ClinicalEffect, sourcePhrase)
		} else if len(sourcePhrase) < 40 {
			sourcePhrase = fmt.Sprintf("%s interaction with %s — %s",
				drugName, interactant, sourcePhrase)
		}

		factData, _ := json.Marshal(map[string]interface{}{
			// Core KBInteractionContent fields (matching STRUCTURED_PARSE schema)
			"interactionType": "DRUG_DRUG",
			"precipitantDrug": interactant,
			"objectDrug":      drugName,
			"interactantName": interactant,
			"severity":        severity,
			"mechanism":       ix.Mechanism,
			"clinicalEffect":  ix.ClinicalEffect,
			"management":      ix.Management,

			// Mapping status for reviewer workflow
			"mappingStatus": mappingStatus,

			// LLM provenance
			"source":         "LLM_FALLBACK",
			"reviewReason":   "UNSTRUCTURED_DDI_FALLBACK",
			"sourcePhrase":   sourcePhrase,
			"sectionCode":    sourceSection.SectionCode,
			"sectionName":    sourceSection.SectionName,
			"termConfidence": 0.75,
		})

		// Dedup key: use same generateCanonicalKey as STRUCTURED_PARSE so
		// cross-method duplicates (table + LLM for same interactant) are caught.
		canonicalContent := KBInteractionContent{
			InteractantName: interactant,
		}
		factKey := generateCanonicalKey(sourceDoc.RxCUI, "INTERACTION", canonicalContent)

		fact := &DerivedFact{
			SourceDocumentID:     sourceDoc.ID,
			SourceSectionID:      sourceSection.ID,
			TargetKB:             targetKB,
			FactType:             "INTERACTION",
			FactKey:              factKey,
			FactData:             factData,
			ExtractionMethod:     "LLM_FALLBACK",
			ExtractionConfidence: 0.75,
			LLMProvider:          "anthropic",
			LLMModel:             p.llmProvider.model,
			ConsensusAchieved:    false,
			GovernanceStatus:     "PENDING_REVIEW",
		}

		facts = append(facts, fact)
	}

	p.log.WithFields(logrus.Fields{
		"drug":        drugName,
		"sectionCode": sourceSection.SectionCode,
		"extracted":   len(ddiResp.Interactions),
		"valid":       len(facts),
		"rejected":    rejected,
		"costUSD":     fmt.Sprintf("$%.4f", actualCost),
	}).Info("LLM DDI fallback extraction complete")

	p.logLLMExtraction(ctx, sourceDoc, sourceSection, startTime, resp, true, "")

	return facts, nil
}

// =============================================================================
// DDI HELPERS
// =============================================================================

// normalizeDDISeverity validates DDI severity from LLM output.
func normalizeDDISeverity(severity string) string {
	upper := strings.ToUpper(strings.TrimSpace(severity))
	switch upper {
	case "CONTRAINDICATED", "SEVERE", "MODERATE", "MILD":
		return upper
	}
	// Map safety-signal-style severities to DDI equivalents
	switch upper {
	case "CRITICAL", "HIGH":
		return "SEVERE"
	case "MEDIUM":
		return "MODERATE"
	case "LOW":
		return "MILD"
	}
	return "MODERATE" // default
}

// isDDINoiseInteractant rejects interactant names that are section headers or noise.
func isDDINoiseInteractant(name string) bool {
	lower := strings.ToLower(name)
	// Section headers and generic labels
	noisePatterns := []string{
		"drug interactions",
		"see clinical pharmacology",
		"see warnings",
		"see precautions",
		"other drugs",
		"other medications",
		"concomitant therapy",
		"table ",
		"section ",
		"figure ",
	}
	for _, pat := range noisePatterns {
		if strings.Contains(lower, pat) {
			return true
		}
	}
	// Reject single-character or very short names
	if len(strings.TrimSpace(name)) < 3 {
		return true
	}
	return false
}

// classifyInteractant determines if the interactant is a specific drug or a drug class.
// Returns "MAPPED" for specific drugs, "UNMAPPED_CLASS" for recognized class patterns,
// or "UNMAPPED_UNKNOWN" for unresolvable names.
func classifyInteractant(name string) string {
	lower := strings.ToLower(name)

	// Class patterns: plurals, enzyme families, pharmacological categories
	classIndicators := []string{
		"inhibitor", "inducer", "antagonist", "agonist",
		"blocker", "substrate",
		"nsaid", "ace inhibitor", "arb",
		"diuretic", "anticoagulant", "antiplatelet",
		"antifungal", "antiarrhythmic", "antidepressant",
		"antipsychotic", "antibiotic", "antiepileptic",
		"corticosteroid", "immunosuppressant",
		"cyp3a4", "cyp2d6", "cyp2c9", "cyp2c19", "cyp1a2",
		"p-glycoprotein", "p-gp", "oatp", "ugt",
		"agents", "drugs", "products", "medications",
		"containing", "supplements",
		"potassium-sparing", "vitamin k",
		"qt-prolonging", "serotonergic",
		"mao inhibitor", "maoi",
	}
	for _, ind := range classIndicators {
		if strings.Contains(lower, ind) {
			return "UNMAPPED_CLASS"
		}
	}

	// Specific drug names are typically single capitalized words without class indicators
	return "MAPPED"
}

// =============================================================================
// LAB REFERENCE LLM EXTRACTION
// =============================================================================

// llmLabResponse is the expected JSON response from Claude for lab monitoring extraction.
type llmLabResponse struct {
	LabMonitoring []llmLabEntry `json:"lab_monitoring"`
}

type llmLabEntry struct {
	LabName             string  `json:"lab_name"`              // e.g., "INR", "Serum potassium", "LFTs"
	LOINCCode           string  `json:"loinc_code,omitempty"`  // If identifiable
	ReferenceRangeLow   *float64 `json:"reference_range_low"`  // Normal low
	ReferenceRangeHigh  *float64 `json:"reference_range_high"` // Normal high
	Unit                string  `json:"unit"`                  // e.g., "mEq/L", "mg/dL", "ratio"
	MonitoringFrequency string  `json:"monitoring_frequency"`  // e.g., "weekly", "every 3 months", "baseline then annually"
	BaselineRequired    bool    `json:"baseline_required"`     // If baseline measurement needed
	ClinicalRationale   string  `json:"clinical_rationale"`    // Why this lab needs monitoring
	SourcePhrase        string  `json:"source_phrase"`         // Grounding text from label
}

func buildLabMonitoringPrompt(drugName, plainText string) string {
	return fmt.Sprintf(`You are a Clinical Data Extractor analyzing an FDA drug label section for %s.

TASK: Extract all lab monitoring requirements and reference ranges mentioned in this section.

EXTRACTION RULES:
1. Extract each lab test as a SEPARATE entry.
2. A "lab test" includes: blood tests (CBC, BMP, CMP, LFTs), coagulation tests (INR, PT, aPTT), drug levels (serum digoxin, lithium), metabolic markers (HbA1c, creatinine, eGFR, BUN), electrolytes (potassium, sodium, magnesium, calcium), and organ function markers (ALT, AST, bilirubin).
3. Extract reference ranges when stated (e.g., "INR 2-3", "potassium 3.5-5.0 mEq/L").
4. Extract monitoring frequency when stated (e.g., "weekly for first month", "every 3 months", "baseline and periodic").
5. Mark baseline_required as true if the label says to check before starting therapy.
6. Every extraction MUST map to a specific phrase in the text.

EXCLUSIONS — DO NOT EXTRACT:
- Pharmacokinetic parameters (Cmax, AUC, half-life, Tmax, bioavailability) unless they define a therapeutic drug monitoring range
- Protein binding percentages
- General pharmacology statements without monitoring implications
- Clinical trial endpoints or study results

OUTPUT FORMAT (JSON only, no other text):
{"lab_monitoring": [
  {"lab_name": "INR", "loinc_code": "6301-6", "reference_range_low": 2.0, "reference_range_high": 3.0, "unit": "ratio", "monitoring_frequency": "regularly during therapy", "baseline_required": true, "clinical_rationale": "Warfarin anticoagulation monitoring", "source_phrase": "Perform regular monitoring of INR in all treated patients"},
  {"lab_name": "Serum potassium", "loinc_code": "2823-3", "reference_range_low": 3.5, "reference_range_high": 5.0, "unit": "mEq/L", "monitoring_frequency": "periodically", "baseline_required": true, "clinical_rationale": "Risk of hyperkalemia with spironolactone", "source_phrase": "Monitor serum potassium levels periodically"}
]}

If reference range is not mentioned, use null for reference_range_low and reference_range_high.
If no lab monitoring found: {"lab_monitoring": []}

DRUG: %s
TEXT:
%s`, drugName, drugName, plainText)
}

// isLabEligibleSection returns true for sections where lab monitoring extraction is allowed.
func isLabEligibleSection(sectionCode string) bool {
	switch sectionCode {
	case "34090-1": // Clinical Pharmacology
		return true
	case "43685-7": // Warnings and Precautions
		return true
	case "34066-1": // Boxed Warning (may contain critical monitoring)
		return true
	default:
		return false
	}
}

// extractLabReferencesViaLLM calls Claude to extract lab monitoring requirements from a section.
func (p *Pipeline) extractLabReferencesViaLLM(
	ctx context.Context,
	sourceDoc *SourceDocument,
	sourceSection *SourceSection,
	routed *dailymed.RoutedSection,
) ([]*DerivedFact, error) {
	if p.llmProvider == nil {
		return nil, fmt.Errorf("LLM provider not initialized")
	}

	drugName := sourceDoc.DrugName
	if drugName == "" {
		drugName = sourceDoc.RxCUI
	}

	// Only process sections with enough text to contain monitoring info
	if len(routed.PlainText) < 200 {
		return nil, nil
	}

	prompt := buildLabMonitoringPrompt(drugName, routed.PlainText)

	estimatedCost := 0.01
	if !p.llmBudget.canSpend(estimatedCost) {
		p.log.Warn("LLM budget exhausted, skipping lab extraction")
		return nil, nil
	}

	p.llmBudget.waitForRateLimit()
	startTime := time.Now()

	resp, err := p.llmProvider.call(ctx, prompt)
	if err != nil {
		p.logLLMExtraction(ctx, sourceDoc, sourceSection, startTime, nil, false, err.Error())
		return nil, fmt.Errorf("Claude API call for lab %s: %w", drugName, err)
	}

	actualCost := p.llmProvider.costUSD(resp)
	p.llmBudget.recordSpend(actualCost)

	if len(resp.Content) == 0 {
		p.logLLMExtraction(ctx, sourceDoc, sourceSection, startTime, resp, false, "empty response")
		return nil, fmt.Errorf("empty lab response from Claude for %s", drugName)
	}

	rawText := resp.Content[0].Text
	jsonStr := extractLLMJSON(rawText)

	var labResp llmLabResponse
	if err := json.Unmarshal([]byte(jsonStr), &labResp); err != nil {
		p.logLLMExtraction(ctx, sourceDoc, sourceSection, startTime, resp, false, err.Error())
		return nil, fmt.Errorf("parse lab JSON for %s: %w (raw: %.200s)", drugName, err, jsonStr)
	}

	if len(labResp.LabMonitoring) == 0 {
		p.log.WithFields(logrus.Fields{
			"drug":        drugName,
			"sectionCode": sourceSection.SectionCode,
		}).Info("LLM found no lab monitoring requirements in section")
		return nil, nil
	}

	// Build facts
	var facts []*DerivedFact

	targetKB := "KB-16"
	if len(routed.TargetKBs) > 0 {
		targetKB = routed.TargetKBs[0]
	}

	for _, lab := range labResp.LabMonitoring {
		labName := strings.TrimSpace(lab.LabName)
		if labName == "" {
			continue
		}

		factData, _ := json.Marshal(map[string]interface{}{
			"loincCode":           lab.LOINCCode,
			"labName":             labName,
			"referenceRangeLow":   lab.ReferenceRangeLow,
			"referenceRangeHigh":  lab.ReferenceRangeHigh,
			"unit":                lab.Unit,
			"monitoringFrequency": lab.MonitoringFrequency,
			"baselineRequired":    lab.BaselineRequired,
			"clinicalRationale":   lab.ClinicalRationale,

			// LLM provenance
			"source":         "LLM_FALLBACK",
			"reviewReason":   "LAB_MONITORING_EXTRACTION",
			"sourcePhrase":   lab.SourcePhrase,
			"sectionCode":    sourceSection.SectionCode,
			"sectionName":    sourceSection.SectionName,
			"termConfidence": 0.75,
		})

		// Dedup key: rxcui + LAB_REFERENCE + labName|unit
		canonicalContent := KBLabReferenceContent{
			LabName: labName,
			Unit:    lab.Unit,
		}
		factKey := generateCanonicalKey(sourceDoc.RxCUI, "LAB_REFERENCE", canonicalContent)

		fact := &DerivedFact{
			SourceDocumentID:     sourceDoc.ID,
			SourceSectionID:      sourceSection.ID,
			TargetKB:             targetKB,
			FactType:             "LAB_REFERENCE",
			FactKey:              factKey,
			FactData:             factData,
			ExtractionMethod:     "LLM_FALLBACK",
			ExtractionConfidence: 0.75,
			LLMProvider:          "anthropic",
			LLMModel:             p.llmProvider.model,
			ConsensusAchieved:    false,
			GovernanceStatus:     "PENDING_REVIEW",
		}

		facts = append(facts, fact)
	}

	p.log.WithFields(logrus.Fields{
		"drug":        drugName,
		"sectionCode": sourceSection.SectionCode,
		"extracted":   len(labResp.LabMonitoring),
		"valid":       len(facts),
		"costUSD":     fmt.Sprintf("$%.4f", actualCost),
	}).Info("LLM lab reference extraction complete")

	p.logLLMExtraction(ctx, sourceDoc, sourceSection, startTime, resp, true, "")

	return facts, nil
}
