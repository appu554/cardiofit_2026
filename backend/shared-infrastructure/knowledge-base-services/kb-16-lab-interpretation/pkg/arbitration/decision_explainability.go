// Package arbitration provides decision explainability for clinical arbitration.
// This module generates human-readable explanations for all arbitration decisions,
// supporting regulatory compliance, clinical trust, and audit requirements.
package arbitration

import (
	"fmt"
	"strings"
)

// =============================================================================
// DECISION EXPLAINABILITY ENGINE
// =============================================================================
// Generates human-readable explanations for arbitration decisions.
// These explanations are designed for:
// - Clinical staff (physicians, pharmacists, nurses)
// - Regulatory auditors
// - System administrators
// - Quality assurance teams

// DecisionExplainer generates human-readable explanations for arbitration decisions.
type DecisionExplainer struct {
	// VerboseMode includes additional technical details
	VerboseMode bool
	// IncludeSourceRefs includes source references in explanations
	IncludeSourceRefs bool
}

// NewDecisionExplainer creates a new explainer with default settings.
func NewDecisionExplainer() *DecisionExplainer {
	return &DecisionExplainer{
		VerboseMode:       false,
		IncludeSourceRefs: true,
	}
}

// ExplainDecision generates a human-readable explanation for an arbitration decision.
func (de *DecisionExplainer) ExplainDecision(decision *ArbitrationDecision, input *ArbitrationInput) *DecisionExplanation {
	explanation := &DecisionExplanation{
		Decision:         decision.Decision,
		PrecedenceRule:   decision.PrecedenceRule,
		Confidence:       decision.Confidence,
		DrugName:         input.DrugName,
		ClinicalIntent:   input.ClinicalIntent,
	}

	// Generate explanation based on decision type
	switch decision.Decision {
	case DecisionAccept:
		explanation.Summary = de.explainAccept(decision, input)
		explanation.ActionRequired = "Proceed with prescription"
		explanation.RiskLevel = "LOW"
	case DecisionBlock:
		explanation.Summary = de.explainBlock(decision, input)
		explanation.ActionRequired = "Cannot proceed - address blocking condition"
		explanation.RiskLevel = "CRITICAL"
	case DecisionOverride:
		explanation.Summary = de.explainOverride(decision, input)
		explanation.ActionRequired = "Dual sign-off required to proceed"
		explanation.RiskLevel = "HIGH"
	case DecisionDefer:
		explanation.Summary = de.explainDefer(decision, input)
		explanation.ActionRequired = "Provide missing information"
		explanation.RiskLevel = "MEDIUM"
	case DecisionEscalate:
		explanation.Summary = de.explainEscalate(decision, input)
		explanation.ActionRequired = "Route to expert review"
		explanation.RiskLevel = "HIGH"
	default:
		explanation.Summary = "Unknown decision type"
		explanation.ActionRequired = "Contact system administrator"
		explanation.RiskLevel = "UNKNOWN"
	}

	// Add conflict details if present
	if len(decision.ConflictsFound) > 0 {
		explanation.ConflictSummary = de.summarizeConflicts(decision.ConflictsFound)
	}

	// Add winning source details
	if decision.WinningSource != nil {
		explanation.WinningSourceType = string(*decision.WinningSource)
	}

	return explanation
}

// =============================================================================
// DECISION-SPECIFIC EXPLANATIONS
// =============================================================================

// explainAccept generates explanation for ACCEPT decisions.
func (de *DecisionExplainer) explainAccept(decision *ArbitrationDecision, input *ArbitrationInput) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("Prescription approved: No conflicts detected for %s.", input.DrugName))

	// Add dosing guidance if available
	if input.ClinicalIntent == "PRESCRIBE" || input.ClinicalIntent == "MODIFY" {
		parts = append(parts, "Standard dosing guidelines apply.")
	}

	// Add confidence
	parts = append(parts, fmt.Sprintf("Confidence: %.0f%%.", decision.Confidence*100))

	// Add any monitoring recommendations
	if de.hasMonitoringRecommendation(decision) {
		parts = append(parts, "Routine monitoring recommended per protocol.")
	}

	return strings.Join(parts, " ")
}

// explainBlock generates explanation for BLOCK decisions.
func (de *DecisionExplainer) explainBlock(decision *ArbitrationDecision, input *ArbitrationInput) string {
	var parts []string

	// Start with decision
	parts = append(parts, fmt.Sprintf("BLOCKED: Cannot proceed with %s.", input.DrugName))

	// Explain based on precedence rule
	switch decision.PrecedenceRule {
	case "P0":
		parts = append(parts, de.explainP0Block(decision, input))
	case "P1":
		parts = append(parts, de.explainP1Block(decision, input))
	case "P2":
		parts = append(parts, "Blocked by higher-level authority guidance.")
	case "P3":
		parts = append(parts, "Authority contraindication supersedes dosing rules.")
	default:
		if decision.ClinicalRationale != "" {
			parts = append(parts, decision.ClinicalRationale)
		}
	}

	// Add required action
	parts = append(parts, "Clinical intervention required before proceeding.")

	return strings.Join(parts, " ")
}

// explainP0Block generates P0 (Physiology Supremacy) explanation.
func (de *DecisionExplainer) explainP0Block(decision *ArbitrationDecision, input *ArbitrationInput) string {
	// P0 blocks are due to CRITICAL/PANIC lab values
	var labDetails string

	for _, lab := range input.LabInterpretations {
		if lab.IsCritical() {
			labDetails = fmt.Sprintf(
				"Lab %s (%.2f %s) is at %s level. This physiological finding supersedes all other rules.",
				lab.LabTest, lab.Value, lab.Unit, lab.Interpretation,
			)
			break
		}
	}

	if labDetails == "" {
		labDetails = "Critical lab value detected - physiology supersedes all rules."
	}

	return labDetails
}

// explainP1Block generates P1 (Regulatory Block) explanation.
func (de *DecisionExplainer) explainP1Block(decision *ArbitrationDecision, input *ArbitrationInput) string {
	// P1 blocks are regulatory (FDA BBW, etc.)
	for _, block := range input.RegulatoryBlocks {
		// All regulatory blocks are active by nature - their presence indicates a block
		if block.Effect == EffectContraindicated {
			return fmt.Sprintf(
				"FDA %s prohibits use in this context. This regulatory requirement cannot be overridden programmatically.",
				block.BlockType,
			)
		}
	}
	return "Regulatory block in effect - cannot be overridden."
}

// explainOverride generates explanation for OVERRIDE decisions.
func (de *DecisionExplainer) explainOverride(decision *ArbitrationDecision, input *ArbitrationInput) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("Override required for %s.", input.DrugName))

	// Explain the conflict
	if len(decision.ConflictsFound) > 0 {
		conflict := decision.ConflictsFound[0]
		parts = append(parts, fmt.Sprintf(
			"Conflict detected: %s vs %s.",
			conflict.SourceAType, conflict.SourceBType,
		))
	}

	// Explain override requirements
	parts = append(parts, "Dual sign-off required to proceed.")
	parts = append(parts, "Override reason must be documented.")

	// Add the precedence rule that determined override
	if decision.PrecedenceRule != "" {
		parts = append(parts, fmt.Sprintf(
			"Resolution rule: %s - %s.",
			decision.PrecedenceRule, de.getPrecedenceRuleDescription(decision.PrecedenceRule),
		))
	}

	return strings.Join(parts, " ")
}

// explainDefer generates explanation for DEFER decisions.
func (de *DecisionExplainer) explainDefer(decision *ArbitrationDecision, input *ArbitrationInput) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("Decision deferred for %s.", input.DrugName))
	parts = append(parts, "Insufficient data to make safe determination.")

	// List what's missing (if known)
	missingData := de.identifyMissingData(input)
	if len(missingData) > 0 {
		parts = append(parts, fmt.Sprintf("Missing: %s.", strings.Join(missingData, ", ")))
	}

	parts = append(parts, "Please provide required information and resubmit.")

	return strings.Join(parts, " ")
}

// explainEscalate generates explanation for ESCALATE decisions.
func (de *DecisionExplainer) explainEscalate(decision *ArbitrationDecision, input *ArbitrationInput) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("Escalated for expert review: %s.", input.DrugName))

	// Explain why escalation is needed
	if len(decision.ConflictsFound) > 0 {
		if len(decision.ConflictsFound) > 1 {
			parts = append(parts, fmt.Sprintf(
				"Multiple conflicts detected (%d) requiring expert arbitration.",
				len(decision.ConflictsFound),
			))
		} else {
			conflict := decision.ConflictsFound[0]
			parts = append(parts, fmt.Sprintf(
				"Conflicting guidance: %s (says %s) vs %s (says %s).",
				conflict.SourceAType, conflict.SourceAEffect,
				conflict.SourceBType, conflict.SourceBEffect,
			))
		}
	}

	// Check if lab abnormality contributed
	for _, lab := range input.LabInterpretations {
		if lab.Interpretation == "HIGH" || lab.Interpretation == "LOW" || lab.Interpretation == "ABNORMAL" {
			parts = append(parts, fmt.Sprintf(
				"Lab abnormality noted: %s is %s.",
				lab.LabTest, lab.Interpretation,
			))
			break
		}
	}

	parts = append(parts, "Route to clinical pharmacist or supervising physician.")

	return strings.Join(parts, " ")
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// hasMonitoringRecommendation checks if monitoring is recommended.
func (de *DecisionExplainer) hasMonitoringRecommendation(decision *ArbitrationDecision) bool {
	// Check if any winning assertion includes monitoring
	if decision.WinningAssertion != nil {
		// Type assertion to check for monitoring effect
		switch v := decision.WinningAssertion.(type) {
		case AuthorityFactAssertion:
			return v.Effect == EffectMonitor
		case CanonicalRuleAssertion:
			return v.Effect == EffectMonitor
		case LabInterpretationAssertion:
			return v.Effect == EffectMonitor
		}
	}
	return false
}

// summarizeConflicts creates a human-readable conflict summary.
func (de *DecisionExplainer) summarizeConflicts(conflicts []Conflict) string {
	if len(conflicts) == 0 {
		return "No conflicts detected"
	}

	var summaries []string
	for i, c := range conflicts {
		if i >= 3 { // Limit to first 3 for readability
			summaries = append(summaries, fmt.Sprintf("...and %d more conflicts", len(conflicts)-3))
			break
		}
		summaries = append(summaries, fmt.Sprintf(
			"%s vs %s (%s severity)",
			c.SourceAType, c.SourceBType, c.Severity,
		))
	}

	return strings.Join(summaries, "; ")
}

// getPrecedenceRuleDescription returns human-readable description of precedence rule.
func (de *DecisionExplainer) getPrecedenceRuleDescription(rule string) string {
	descriptions := map[string]string{
		"P0": "Physiology Supremacy - Critical/Panic labs supersede all rules",
		"P1": "Regulatory Block - FDA requirements cannot be overridden",
		"P2": "Authority Hierarchy - Higher evidence level wins",
		"P3": "Authority over Rule - Curated sources supersede extracted rules",
		"P4": "Lab Critical Escalation - Abnormal labs require review",
		"P5": "Provenance Consensus - More sources agreeing wins",
		"P6": "Local Policy Limits - Site policies cannot override authorities",
		"P7": "Restrictive Wins Ties - Fail-safe default",
	}

	if desc, ok := descriptions[rule]; ok {
		return desc
	}
	return "Standard resolution"
}

// identifyMissingData identifies what data is missing for a deferred decision.
func (de *DecisionExplainer) identifyMissingData(input *ArbitrationInput) []string {
	var missing []string

	// Check patient context
	if input.PatientContext == nil {
		missing = append(missing, "Patient demographics")
	} else {
		if input.PatientContext.EGFR == nil && input.PatientContext.CKDStage == nil {
			missing = append(missing, "Renal function (eGFR/CrCl)")
		}
		if input.PatientContext.Age == 0 {
			missing = append(missing, "Patient age")
		}
		if input.PatientContext.Gender == "" {
			missing = append(missing, "Patient sex")
		}
	}

	// Check lab data
	if len(input.LabInterpretations) == 0 {
		missing = append(missing, "Recent laboratory results")
	}

	return missing
}

// =============================================================================
// OUTPUT STRUCTURES
// =============================================================================

// DecisionExplanation contains the human-readable explanation of a decision.
type DecisionExplanation struct {
	// Core decision info
	Decision         DecisionType `json:"decision"`
	PrecedenceRule   string       `json:"precedence_rule"`
	Confidence       float64      `json:"confidence"`
	DrugName         string       `json:"drug_name"`
	ClinicalIntent   string       `json:"clinical_intent"`

	// Human-readable summary
	Summary          string `json:"summary"`
	ActionRequired   string `json:"action_required"`
	RiskLevel        string `json:"risk_level"` // LOW, MEDIUM, HIGH, CRITICAL

	// Conflict summary
	ConflictSummary  string `json:"conflict_summary,omitempty"`
	WinningSourceType string `json:"winning_source_type,omitempty"`
}

// ForClinician returns a clinician-friendly version of the explanation.
func (e *DecisionExplanation) ForClinician() string {
	return fmt.Sprintf(
		"[%s] %s\n\nAction: %s\nRisk Level: %s",
		e.Decision, e.Summary, e.ActionRequired, e.RiskLevel,
	)
}

// ForAudit returns an audit-ready version of the explanation.
func (e *DecisionExplanation) ForAudit() string {
	var parts []string

	parts = append(parts, fmt.Sprintf("Decision: %s", e.Decision))
	parts = append(parts, fmt.Sprintf("Drug: %s", e.DrugName))
	parts = append(parts, fmt.Sprintf("Intent: %s", e.ClinicalIntent))
	parts = append(parts, fmt.Sprintf("Precedence Rule: %s", e.PrecedenceRule))
	parts = append(parts, fmt.Sprintf("Confidence: %.2f", e.Confidence))
	parts = append(parts, fmt.Sprintf("Summary: %s", e.Summary))

	if e.ConflictSummary != "" {
		parts = append(parts, fmt.Sprintf("Conflicts: %s", e.ConflictSummary))
	}
	if e.WinningSourceType != "" {
		parts = append(parts, fmt.Sprintf("Winning Source: %s", e.WinningSourceType))
	}

	return strings.Join(parts, "\n")
}

// =============================================================================
// TEMPLATE-BASED EXPLANATIONS FOR COMMON SCENARIOS
// =============================================================================

// ExplainTemplate provides pre-defined templates for common decision scenarios.
type ExplainTemplate struct {
	Scenario     string
	Decision     DecisionType
	Template     string
	Placeholders map[string]string
}

// GetCommonTemplates returns templates for common clinical scenarios.
func GetCommonTemplates() []ExplainTemplate {
	return []ExplainTemplate{
		{
			Scenario: "Benign Drug Accept",
			Decision: DecisionAccept,
			Template: "Prescription approved: No conflicts detected for {{drug}} {{dose}}. Confidence: {{confidence}}%.",
		},
		{
			Scenario: "P0 Physiology Block",
			Decision: DecisionBlock,
			Template: "BLOCKED because lab {{lab_name}} ({{lab_value}} {{lab_unit}}) exceeded {{context}} critical threshold ({{authority}}). No dosing rule may override this physiological finding.",
		},
		{
			Scenario: "P1 Regulatory Block",
			Decision: DecisionBlock,
			Template: "BLOCKED by FDA {{block_type}} for {{drug}}. This regulatory requirement cannot be overridden programmatically. Contact prescriber for alternative.",
		},
		{
			Scenario: "Override Required",
			Decision: DecisionOverride,
			Template: "Override requires dual sign-off because this action contradicts {{precedence_rule}} guidance. Documenting override reason is mandatory.",
		},
		{
			Scenario: "Defer for Data",
			Decision: DecisionDefer,
			Template: "Decision deferred: Insufficient data. Missing: {{missing_data}}. Please provide and resubmit.",
		},
		{
			Scenario: "Escalate for Review",
			Decision: DecisionEscalate,
			Template: "Escalated due to conflicting authority guidance ({{source_a}} vs {{source_b}}) in presence of {{lab_status}}. Routing to expert review.",
		},
	}
}
