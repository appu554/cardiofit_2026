// Package extraction provides condition-action generation from normalized tables.
//
// Phase 3b.5.4: Condition-Action Generator
// Key Principle: Extract computable IF/THEN pairs from normalized tables
// without LLM involvement. Tables contain structured clinical types.
//
// Pipeline: NormalizedTable → []GeneratedRule (Condition + Action pairs)
package extraction

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/cardiofit/shared/types"
)

// =============================================================================
// CONDITION-ACTION GENERATOR
// =============================================================================

// ConditionActionGenerator creates Condition/Action pairs from normalized tables
type ConditionActionGenerator struct {
	rangePattern       *regexp.Regexp
	operatorPattern    *regexp.Regexp
	lessThanPattern    *regexp.Regexp
	greaterThanPattern *regexp.Regexp
}

// NewConditionActionGenerator creates a generator with extraction patterns
func NewConditionActionGenerator() *ConditionActionGenerator {
	return &ConditionActionGenerator{
		// Matches: "30-60", "30 - 60", "30 to 60", "30–60"
		rangePattern: regexp.MustCompile(`^\s*(\d+(?:\.\d+)?)\s*(?:-|to|–)\s*(\d+(?:\.\d+)?)\s*$`),
		// Matches: "< 30", "> 60", "<= 15", ">= 90", "≤30", "≥60"
		operatorPattern:    regexp.MustCompile(`^\s*([<>]=?|[≤≥])\s*(\d+(?:\.\d+)?)\s*$`),
		lessThanPattern:    regexp.MustCompile(`(?i)less\s+than\s+(\d+(?:\.\d+)?)`),
		greaterThanPattern: regexp.MustCompile(`(?i)(greater|more)\s+than\s+(\d+(?:\.\d+)?)`),
	}
}

// =============================================================================
// GENERATED RULE STRUCTURES
// =============================================================================

// GeneratedRule represents a rule extracted from a table row
type GeneratedRule struct {
	Condition      types.Condition `json:"condition"`
	Action         types.Action    `json:"action"`
	RowIndex       int             `json:"row_index"`
	SourceCells    []string        `json:"source_cells"`     // Original cell texts
	Confidence     float64         `json:"confidence"`
	ExtractionNote string          `json:"extraction_note,omitempty"`
}

// GenerationResult contains the outcome of rule generation
type GenerationResult struct {
	Rules          []GeneratedRule `json:"rules"`
	SkippedRows    []SkippedRow    `json:"skipped_rows,omitempty"`
	TotalRows      int             `json:"total_rows"`
	GeneratedCount int             `json:"generated_count"`
	SkippedCount   int             `json:"skipped_count"`
}

// SkippedRow records a row that couldn't be converted to a rule
type SkippedRow struct {
	RowIndex int    `json:"row_index"`
	Reason   string `json:"reason"`
	Cells    []string `json:"cells"`
}

// =============================================================================
// RULE GENERATION
// =============================================================================

// GenerateFromTable extracts IF/THEN rules from a normalized table
func (g *ConditionActionGenerator) GenerateFromTable(table *NormalizedTable) (*GenerationResult, error) {
	if !table.Translatable {
		return nil, fmt.Errorf("table is not translatable: %s", table.UntranslatableReason)
	}

	result := &GenerationResult{
		TotalRows: len(table.Rows),
	}

	// Get condition and action column indices
	conditionCols := table.GetConditionColumns()
	actionCols := table.GetActionColumns()

	if len(conditionCols) == 0 {
		return nil, fmt.Errorf("no condition columns found")
	}
	if len(actionCols) == 0 {
		return nil, fmt.Errorf("no action columns found")
	}

	// Generate rule for each row
	for _, row := range table.Rows {
		// Collect source cells for audit trail
		var sourceCells []string
		for _, cell := range row.Cells {
			sourceCells = append(sourceCells, cell.OriginalText)
		}

		// Extract condition from the row
		condition, condConf, condNote := g.extractCondition(row, table.NormalizedCols, conditionCols)
		if condition == nil {
			result.SkippedRows = append(result.SkippedRows, SkippedRow{
				RowIndex: row.Index,
				Reason:   "CONDITION_EXTRACTION_FAILED: " + condNote,
				Cells:    sourceCells,
			})
			result.SkippedCount++
			continue
		}

		// Extract action from the row
		action, actConf, actNote := g.extractAction(row, table.NormalizedCols, actionCols)
		if action == nil {
			result.SkippedRows = append(result.SkippedRows, SkippedRow{
				RowIndex: row.Index,
				Reason:   "ACTION_EXTRACTION_FAILED: " + actNote,
				Cells:    sourceCells,
			})
			result.SkippedCount++
			continue
		}

		// Create generated rule
		rule := GeneratedRule{
			Condition:   *condition,
			Action:      *action,
			RowIndex:    row.Index,
			SourceCells: sourceCells,
			Confidence:  (condConf + actConf) / 2,
		}

		if condNote != "" || actNote != "" {
			rule.ExtractionNote = strings.TrimSpace(condNote + "; " + actNote)
		}

		result.Rules = append(result.Rules, rule)
		result.GeneratedCount++
	}

	return result, nil
}

// =============================================================================
// CONDITION EXTRACTION
// =============================================================================

// extractCondition builds a Condition from row cells
func (g *ConditionActionGenerator) extractCondition(row NormalizedRow, cols []NormalizedColumn, conditionCols []NormalizedColumn) (*types.Condition, float64, string) {
	// Try each condition column until we find a parseable one
	for _, col := range conditionCols {
		if col.Index >= len(row.Cells) {
			continue
		}

		cell := row.Cells[col.Index]
		if cell.IsEmpty {
			continue
		}

		// Use normalized value if available
		if cell.Normalized != nil {
			condition := g.normalizedValueToCondition(cell.Normalized, col)
			if condition != nil {
				return condition, cell.Normalized.Confidence, ""
			}
		}

		// Try to parse the original text
		condition := g.parseConditionText(cell.OriginalText, col.NormalizedName, col.Unit)
		if condition != nil {
			return condition, 0.7, "parsed from text"
		}
	}

	return nil, 0, "no parseable condition found"
}

// normalizedValueToCondition converts a NormalizedValue to a types.Condition
func (g *ConditionActionGenerator) normalizedValueToCondition(nv *NormalizedValue, col NormalizedColumn) *types.Condition {
	if nv == nil {
		return nil
	}

	// Handle range values
	if nv.IsRange() {
		return &types.Condition{
			Variable: nv.Variable,
			Operator: types.OpBetween,
			MinValue: nv.MinValue,
			MaxValue: nv.MaxValue,
			Unit:     nv.Unit,
		}
	}

	// Handle single numeric value
	if nv.NumericValue != nil {
		// Default to >= for single values (common in GFR tables)
		return &types.Condition{
			Variable: nv.Variable,
			Operator: types.OpGreaterOrEqual,
			Value:    nv.NumericValue,
			Unit:     nv.Unit,
		}
	}

	// Handle string value (e.g., Child-Pugh class)
	if nv.StringValue != nil {
		return &types.Condition{
			Variable:    nv.Variable,
			Operator:    types.OpEquals,
			StringValue: nv.StringValue,
		}
	}

	return nil
}

// parseConditionText parses condition from raw text
func (g *ConditionActionGenerator) parseConditionText(text, variable, unit string) *types.Condition {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	// Try range pattern: "30-60", "30 to 60"
	if matches := g.rangePattern.FindStringSubmatch(text); len(matches) == 3 {
		min, _ := strconv.ParseFloat(matches[1], 64)
		max, _ := strconv.ParseFloat(matches[2], 64)
		return &types.Condition{
			Variable: variable,
			Operator: types.OpBetween,
			MinValue: &min,
			MaxValue: &max,
			Unit:     unit,
		}
	}

	// Try operator pattern: "< 30", "> 60", "≤15"
	if matches := g.operatorPattern.FindStringSubmatch(text); len(matches) == 3 {
		value, _ := strconv.ParseFloat(matches[2], 64)
		op := g.parseOperatorSymbol(matches[1])
		return &types.Condition{
			Variable: variable,
			Operator: op,
			Value:    &value,
			Unit:     unit,
		}
	}

	// Try "less than X" pattern
	if matches := g.lessThanPattern.FindStringSubmatch(text); len(matches) == 2 {
		value, _ := strconv.ParseFloat(matches[1], 64)
		return &types.Condition{
			Variable: variable,
			Operator: types.OpLessThan,
			Value:    &value,
			Unit:     unit,
		}
	}

	// Try "greater than X" pattern
	if matches := g.greaterThanPattern.FindStringSubmatch(text); len(matches) == 3 {
		value, _ := strconv.ParseFloat(matches[2], 64)
		return &types.Condition{
			Variable: variable,
			Operator: types.OpGreaterThan,
			Value:    &value,
			Unit:     unit,
		}
	}

	// Try Child-Pugh patterns
	lower := strings.ToLower(text)
	if strings.Contains(lower, "child-pugh") || strings.Contains(lower, "child pugh") ||
		strings.Contains(lower, "hepatic") || strings.Contains(lower, "liver") {
		childPugh := g.extractChildPughClass(text)
		if childPugh != "" {
			return &types.Condition{
				Variable:    "hepatic.child_pugh",
				Operator:    types.OpEquals,
				StringValue: &childPugh,
			}
		}
	}

	// Try to extract just a number (common in GFR tables)
	if numMatch := regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*$`).FindStringSubmatch(text); len(numMatch) == 2 {
		value, _ := strconv.ParseFloat(numMatch[1], 64)
		// Single number typically represents lower bound
		return &types.Condition{
			Variable: variable,
			Operator: types.OpGreaterOrEqual,
			Value:    &value,
			Unit:     unit,
		}
	}

	return nil
}

// parseOperatorSymbol converts operator string to types.Operator
func (g *ConditionActionGenerator) parseOperatorSymbol(op string) types.Operator {
	switch op {
	case "<":
		return types.OpLessThan
	case ">":
		return types.OpGreaterThan
	case "<=", "≤":
		return types.OpLessOrEqual
	case ">=", "≥":
		return types.OpGreaterOrEqual
	default:
		return types.OpEquals
	}
}

// extractChildPughClass extracts Child-Pugh class from text
func (g *ConditionActionGenerator) extractChildPughClass(text string) string {
	lower := strings.ToLower(text)

	// Direct class mentions
	classPattern := regexp.MustCompile(`(?i)(?:child[-\s]?pugh\s*)?(?:class\s*)?([abc])`)
	if matches := classPattern.FindStringSubmatch(text); len(matches) == 2 {
		return strings.ToUpper(matches[1])
	}

	// Severity-based
	if strings.Contains(lower, "mild") {
		return "A"
	}
	if strings.Contains(lower, "moderate") {
		return "B"
	}
	if strings.Contains(lower, "severe") {
		return "C"
	}

	return ""
}

// =============================================================================
// ACTION EXTRACTION
// =============================================================================

// extractAction builds an Action from row cells
func (g *ConditionActionGenerator) extractAction(row NormalizedRow, cols []NormalizedColumn, actionCols []NormalizedColumn) (*types.Action, float64, string) {
	// Collect all action cell texts for comprehensive parsing
	var actionTexts []string
	var bestConfidence float64

	for _, col := range actionCols {
		if col.Index >= len(row.Cells) {
			continue
		}

		cell := row.Cells[col.Index]
		if cell.IsEmpty {
			continue
		}

		actionTexts = append(actionTexts, cell.OriginalText)

		if cell.Normalized != nil && cell.Normalized.Confidence > bestConfidence {
			bestConfidence = cell.Normalized.Confidence
		}
	}

	if len(actionTexts) == 0 {
		return nil, 0, "no action text found"
	}

	// Combine all action texts for parsing
	combinedText := strings.Join(actionTexts, " ")

	// Parse the action
	action := g.parseActionText(combinedText)
	if action != nil {
		return action, 0.8, ""
	}

	return nil, 0, "could not parse action from: " + combinedText
}

// parseActionText converts action text to an Action
func (g *ConditionActionGenerator) parseActionText(text string) *types.Action {
	lower := strings.ToLower(text)
	text = strings.TrimSpace(text)

	// Check for contraindication keywords first (highest severity)
	if g.isContraindicated(lower) {
		return &types.Action{
			Effect:   types.EffectContraindicated,
			Message:  text,
			Severity: types.SeverityCritical,
		}
	}

	// Check for avoid keywords
	if g.isAvoid(lower) {
		return &types.Action{
			Effect:   types.EffectAvoid,
			Message:  text,
			Severity: types.SeverityHigh,
		}
	}

	// Check for no change / normal dose
	if g.isNoChange(lower) {
		return &types.Action{
			Effect:   types.EffectNoChange,
			Message:  text,
			Severity: types.SeverityInfo,
		}
	}

	// Check for monitoring requirement
	if g.isMonitoring(lower) {
		return &types.Action{
			Effect:   types.EffectMonitor,
			Message:  text,
			Severity: types.SeverityModerate,
		}
	}

	// Check for use with caution
	if g.isUseWithCaution(lower) {
		return &types.Action{
			Effect:   types.EffectUseWithCaution,
			Message:  text,
			Severity: types.SeverityModerate,
		}
	}

	// Try to parse dose adjustment
	if adjustment := g.parseDoseAdjustment(text); adjustment != nil {
		return &types.Action{
			Effect:     types.EffectDoseAdjust,
			Adjustment: adjustment,
			Message:    text,
			Severity:   types.SeverityHigh,
		}
	}

	// Default: treat as dose adjustment with message only
	return &types.Action{
		Effect:   types.EffectDoseAdjust,
		Message:  text,
		Severity: types.SeverityModerate,
	}
}

// =============================================================================
// ACTION KEYWORD DETECTION
// =============================================================================

func (g *ConditionActionGenerator) isContraindicated(text string) bool {
	keywords := []string{
		"contraindicated", "do not use", "should not be used",
		"must not be used", "prohibited", "never use",
		"absolute contraindication", "not recommended for use",
	}
	for _, kw := range keywords {
		if strings.Contains(text, kw) {
			return true
		}
	}
	return false
}

func (g *ConditionActionGenerator) isAvoid(text string) bool {
	keywords := []string{
		"avoid", "not recommended", "should be avoided",
		"should not be administered", "use is not recommended",
	}
	for _, kw := range keywords {
		if strings.Contains(text, kw) {
			return true
		}
	}
	return false
}

func (g *ConditionActionGenerator) isNoChange(text string) bool {
	keywords := []string{
		"no adjustment", "no dose adjustment", "no change",
		"normal dose", "no dosage adjustment", "standard dose",
		"no modification", "usual dose", "no reduction",
	}
	for _, kw := range keywords {
		if strings.Contains(text, kw) {
			return true
		}
	}
	return false
}

func (g *ConditionActionGenerator) isMonitoring(text string) bool {
	keywords := []string{
		"monitor", "monitoring", "close monitoring",
		"careful monitoring", "frequent monitoring",
		"monitor closely", "observe",
	}
	for _, kw := range keywords {
		if strings.Contains(text, kw) {
			return true
		}
	}
	return false
}

func (g *ConditionActionGenerator) isUseWithCaution(text string) bool {
	keywords := []string{
		"use with caution", "caution", "with caution",
		"careful use", "cautious use", "use cautiously",
	}
	for _, kw := range keywords {
		if strings.Contains(text, kw) {
			return true
		}
	}
	return false
}

// =============================================================================
// DOSE ADJUSTMENT PARSING
// =============================================================================

// parseDoseAdjustment extracts specific dosing modifications from text
func (g *ConditionActionGenerator) parseDoseAdjustment(text string) *types.DoseAdjustment {
	lower := strings.ToLower(text)

	// Try percentage reduction: "50%", "reduce by 50%", "50% of normal"
	if pctMatch := regexp.MustCompile(`(\d+(?:\.\d+)?)\s*%`).FindStringSubmatch(text); len(pctMatch) == 2 {
		pct, _ := strconv.ParseFloat(pctMatch[1], 64)

		// Check if this is "reduce BY X%" or "X% of normal"
		if strings.Contains(lower, "reduce") || strings.Contains(lower, "decrease") {
			// "Reduce by 50%" means give 50% of normal
			return &types.DoseAdjustment{
				Type:       types.AdjustmentPercentage,
				Percentage: &pct,
			}
		}
		// "50% of normal dose" means give 50%
		return &types.DoseAdjustment{
			Type:       types.AdjustmentPercentage,
			Percentage: &pct,
		}
	}

	// Try specific dose: "250 mg", "500mg BID"
	if doseMatch := regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s*(mg|mcg|g|μg|µg)\s*(daily|bid|tid|qid|once|twice)?`).FindStringSubmatch(text); len(doseMatch) >= 3 {
		doseStr := strings.TrimSpace(doseMatch[0])
		return &types.DoseAdjustment{
			Type:         types.AdjustmentAbsolute,
			AbsoluteDose: &doseStr,
		}
	}

	// Try maximum dose: "maximum 500mg", "max dose 1g"
	if maxMatch := regexp.MustCompile(`(?i)(?:maximum|max(?:imum)?(?:\s+dose)?)\s*[:\s]*(\d+(?:\.\d+)?)\s*(mg|mcg|g)`).FindStringSubmatch(text); len(maxMatch) >= 3 {
		maxStr := maxMatch[1] + " " + maxMatch[2]
		return &types.DoseAdjustment{
			Type:    types.AdjustmentMaxDose,
			MaxDose: &maxStr,
		}
	}

	// Try interval: "every 48 hours", "q48h"
	if intMatch := regexp.MustCompile(`(?i)(?:every|q)\s*(\d+)\s*(hours?|h)`).FindStringSubmatch(text); len(intMatch) >= 2 {
		intStr := "Every " + intMatch[1] + " hours"
		return &types.DoseAdjustment{
			Type:     types.AdjustmentInterval,
			Interval: &intStr,
		}
	}

	// Try frequency change: "once daily", "every other day"
	if strings.Contains(lower, "once daily") || strings.Contains(lower, "daily") {
		freq := "daily"
		return &types.DoseAdjustment{
			Type:      types.AdjustmentFrequency,
			Frequency: &freq,
		}
	}
	if strings.Contains(lower, "every other day") || strings.Contains(lower, "alternate day") {
		freq := "every other day"
		return &types.DoseAdjustment{
			Type:      types.AdjustmentFrequency,
			Frequency: &freq,
		}
	}

	// Try half dose: "half the dose", "halve the dose"
	if strings.Contains(lower, "half") || strings.Contains(lower, "halve") {
		pct := 50.0
		return &types.DoseAdjustment{
			Type:       types.AdjustmentPercentage,
			Percentage: &pct,
		}
	}

	return nil
}
