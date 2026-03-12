// client.go — KDIGO JSON rules loader and drug rule lookup.
//
// V2 Architecture:
// Loads pre-extracted rules from kdigo_draft_rules.json (produced offline by
// the MCP-RAG atomiser), NOT by parsing PDFs at runtime.
//
// The atomiser uses Claude 3.5 Sonnet + LlamaParse to extract rules from
// KDIGO guideline PDFs, including heatmaps and visual tables that regex
// cannot handle.
//
// Safety: All KDIGO facts use ExtractionMethod "MCP_RAG_EXTRACT" and
// ConfidenceBand ≤ 0.75. They are NEVER auto-approved.
package kdigo

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// Maximum confidence for MCP-RAG extracted rules.
// Higher confidence (0.75) for heatmap+prose, lower (0.55) for ambiguous.
const maxMCPRAGConfidence = 0.75

// Client provides access to KDIGO guideline-extracted organ impairment rules.
type Client struct {
	rulesPath string
	rules     []OrganImpairmentRule
	log       *logrus.Entry
}

// RulesFile is the JSON structure produced by the MCP-RAG atomiser.
type RulesFile struct {
	Rules             []OrganImpairmentRule `json:"rules"`
	ExtractionDate    string                `json:"extraction_date,omitempty"`
	SourcePDF         string                `json:"source_pdf,omitempty"`
	TotalPagesScanned int                   `json:"total_pages_scanned,omitempty"`
	ExtractorVersion  string                `json:"extractor_version,omitempty"`
}

// NewClient loads pre-extracted rules from a JSON file produced by the
// MCP-RAG atomiser. This is NOT runtime PDF parsing — the atomiser runs
// offline as a batch job.
//
// The rulesPath should point to kdigo_draft_rules.json or similar.
func NewClient(rulesPath string) (*Client, error) {
	log := logrus.WithField("component", "kdigo")

	// Check if file exists
	info, err := os.Stat(rulesPath)
	if err != nil {
		return nil, fmt.Errorf("KDIGO rules file %q: %w", rulesPath, err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("KDIGO rules path %q is a directory, expected JSON file", rulesPath)
	}

	// Read and parse JSON
	data, err := os.ReadFile(rulesPath)
	if err != nil {
		return nil, fmt.Errorf("read KDIGO rules file: %w", err)
	}

	var rulesFile RulesFile
	if err := json.Unmarshal(data, &rulesFile); err != nil {
		return nil, fmt.Errorf("parse KDIGO rules JSON: %w", err)
	}

	// Deduplicate rules by composite key
	rules := deduplicateRules(rulesFile.Rules)

	// Validate and enforce safety constraints
	for i := range rules {
		// Enforce confidence ceiling based on ConfidenceBand
		if rules[i].ConfidenceBand > maxMCPRAGConfidence {
			rules[i].ConfidenceBand = maxMCPRAGConfidence
		}
		if rules[i].Confidence > maxMCPRAGConfidence {
			rules[i].Confidence = maxMCPRAGConfidence
		}

		// Set source to KDIGO
		rules[i].EvidenceSource = "KDIGO"

		// Default scope if not set
		if rules[i].RuleScope == "" {
			rules[i].RuleScope = "BOTH"
		}
	}

	log.WithFields(logrus.Fields{
		"rules_file":        rulesPath,
		"rules_total":       len(rules),
		"extractor_version": rulesFile.ExtractorVersion,
		"extraction_date":   rulesFile.ExtractionDate,
	}).Info("KDIGO client initialized from MCP-RAG extracted rules")

	return &Client{
		rulesPath: rulesPath,
		rules:     rules,
		log:       log,
	}, nil
}

// deduplicateRules removes duplicate rules based on composite key.
// Same rule may be extracted from multiple pages or PDF sections.
func deduplicateRules(rules []OrganImpairmentRule) []OrganImpairmentRule {
	seen := make(map[string]bool)
	var result []OrganImpairmentRule

	for _, rule := range rules {
		// Composite key: drug + organ + metric + operator + threshold + action
		key := fmt.Sprintf("%s|%s|%s|%s|%.2f|%s",
			strings.ToLower(rule.DrugName),
			rule.OrganSystem,
			rule.ImpairmentMetric,
			rule.ThresholdOp,
			rule.ThresholdValue,
			rule.ActionType)

		if !seen[key] {
			seen[key] = true
			result = append(result, rule)
		}
	}

	return result
}

// GetRulesForDrug returns all rules matching the given drug name.
func (c *Client) GetRulesForDrug(drugName string) []OrganImpairmentRule {
	var matched []OrganImpairmentRule
	for _, rule := range c.rules {
		if matchesDrug(rule.DrugName, drugName) {
			matched = append(matched, rule)
		}
	}
	return matched
}

// GetOrganImpairmentFacts converts matched rules to AuthorityFact format
// for pipeline storage.
//
// Safety invariants enforced:
//   - ExtractionMethod: "MCP_RAG_EXTRACT" (LLM-based, not regex)
//   - Confidence: capped at 0.75 (maxMCPRAGConfidence)
//   - All facts intended for PENDING_REVIEW governance (enforced in pipeline)
func (c *Client) GetOrganImpairmentFacts(rxcui, drugName string) []AuthorityFact {
	rules := c.GetRulesForDrug(drugName)
	if len(rules) == 0 {
		return nil
	}

	var facts []AuthorityFact
	for _, rule := range rules {
		// Populate RxCUI from pipeline context
		rule.DrugRxCUI = rxcui

		// Use ConfidenceBand if set, otherwise default to Confidence
		confidence := rule.ConfidenceBand
		if confidence == 0 {
			confidence = rule.Confidence
		}
		if confidence == 0 {
			confidence = 0.65 // Default for prose-only extraction
		}
		if confidence > maxMCPRAGConfidence {
			confidence = maxMCPRAGConfidence
		}

		fact := AuthorityFact{
			ID:               fmt.Sprintf("kdigo-oi-%s-%s-%s-%.0f", rxcui, rule.OrganSystem, rule.ImpairmentMetric, rule.ThresholdValue),
			AuthoritySource:  "KDIGO",
			FactType:         FactTypeOrganImpairment,
			RxCUI:            rxcui,
			DrugName:         drugName,
			Content:          rule,
			RiskLevel:        riskLevelFromAction(rule.ActionType),
			ActionRequired:   rule.ActionType,
			Recommendations:  []string{rule.ActionDetail},
			EvidenceLevel:    rule.EvidenceLevel,
			ExtractionMethod: "MCP_RAG_EXTRACT", // V2: LLM-based extraction
			Confidence:       confidence,
			FetchedAt:        time.Now(),
		}
		facts = append(facts, fact)
	}

	if len(facts) > 0 {
		c.log.WithFields(logrus.Fields{
			"drug":  drugName,
			"count": len(facts),
		}).Info("KDIGO organ impairment facts matched")
	}

	return facts
}

// matchesDrug checks if the rule's drug name matches the pipeline drug.
// Uses case-insensitive comparison and handles common variations
// (e.g., "metformin" matches "metformin hydrochloride").
// False positives are acceptable — caught by mandatory PENDING_REVIEW governance.
func matchesDrug(ruleDrugName, pipelineDrugName string) bool {
	a := strings.ToLower(strings.TrimSpace(ruleDrugName))
	b := strings.ToLower(strings.TrimSpace(pipelineDrugName))
	if a == "" || b == "" {
		return false
	}
	return a == b || strings.Contains(b, a) || strings.Contains(a, b)
}

// riskLevelFromAction maps action type to risk level.
func riskLevelFromAction(actionType string) string {
	switch actionType {
	case "CONTRAINDICATED":
		return "CRITICAL"
	case "AVOID":
		return "HIGH"
	case "DOSE_REDUCE", "REDUCE_DOSE":
		return "HIGH"
	case "MONITOR":
		return "MODERATE"
	case "USE_WITH_CAUTION":
		return "LOW"
	default:
		return "MODERATE"
	}
}
