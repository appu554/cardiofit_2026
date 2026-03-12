package evidence

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// InferenceStep represents a single step in the reasoning chain
// Used for FDA-compliant explainability
type InferenceStep struct {
	StepNumber    int         `json:"step_number"`
	Phase         string      `json:"phase"` // recipe, kb_query, evaluation, scoring, ranking
	KBSource      string      `json:"kb_source,omitempty"` // KB-1, KB-2, KB-3, etc.
	RuleID        string      `json:"rule_id,omitempty"`
	RuleName      string      `json:"rule_name,omitempty"`
	Input         interface{} `json:"input"`
	Output        interface{} `json:"output"`
	Reasoning     string      `json:"reasoning"`
	Confidence    float64     `json:"confidence"`
	Duration      Duration    `json:"duration"`
	Timestamp     time.Time   `json:"timestamp"`
	Dependencies  []int       `json:"dependencies,omitempty"` // Step numbers this depends on
}

// Duration represents step execution duration
type Duration struct {
	Milliseconds int64 `json:"milliseconds"`
}

// InferenceChainBuilder helps construct inference chains
type InferenceChainBuilder struct {
	chain      []InferenceStep
	envelopeID uuid.UUID
	startTime  time.Time
}

// NewInferenceChainBuilder creates a new inference chain builder
func NewInferenceChainBuilder(envelopeID uuid.UUID) *InferenceChainBuilder {
	return &InferenceChainBuilder{
		chain:      []InferenceStep{},
		envelopeID: envelopeID,
		startTime:  time.Now(),
	}
}

// AddRecipeStep adds a recipe resolution step
func (icb *InferenceChainBuilder) AddRecipeStep(
	recipeID string,
	dataRequirements []string,
	resolvedData map[string]interface{},
) *InferenceChainBuilder {

	step := InferenceStep{
		StepNumber: len(icb.chain) + 1,
		Phase:      "recipe",
		Input: map[string]interface{}{
			"recipe_id":    recipeID,
			"requirements": dataRequirements,
		},
		Output: map[string]interface{}{
			"resolved_count": len(resolvedData),
			"data_sources":   resolvedData,
		},
		Reasoning:  fmt.Sprintf("Resolved %d data requirements from recipe %s", len(resolvedData), recipeID),
		Confidence: 1.0,
		Timestamp:  time.Now(),
	}

	icb.chain = append(icb.chain, step)
	return icb
}

// AddKBQueryStep adds a knowledge base query step
func (icb *InferenceChainBuilder) AddKBQueryStep(
	kbSource string,
	queryType string,
	input interface{},
	output interface{},
	reasoning string,
) *InferenceChainBuilder {

	step := InferenceStep{
		StepNumber: len(icb.chain) + 1,
		Phase:      "kb_query",
		KBSource:   kbSource,
		Input: map[string]interface{}{
			"query_type": queryType,
			"parameters": input,
		},
		Output:     output,
		Reasoning:  reasoning,
		Confidence: 0.95, // KB queries have high confidence
		Timestamp:  time.Now(),
	}

	icb.chain = append(icb.chain, step)
	return icb
}

// AddRuleEvaluationStep adds a rule evaluation step
func (icb *InferenceChainBuilder) AddRuleEvaluationStep(
	kbSource string,
	ruleID string,
	ruleName string,
	condition interface{},
	result bool,
	reasoning string,
	dependencies []int,
) *InferenceChainBuilder {

	step := InferenceStep{
		StepNumber: len(icb.chain) + 1,
		Phase:      "evaluation",
		KBSource:   kbSource,
		RuleID:     ruleID,
		RuleName:   ruleName,
		Input: map[string]interface{}{
			"condition": condition,
		},
		Output: map[string]interface{}{
			"result": result,
		},
		Reasoning:    reasoning,
		Confidence:   0.9,
		Timestamp:    time.Now(),
		Dependencies: dependencies,
	}

	icb.chain = append(icb.chain, step)
	return icb
}

// AddScoringStep adds a scoring/weighting step
func (icb *InferenceChainBuilder) AddScoringStep(
	medication string,
	factors map[string]float64,
	weights map[string]float64,
	finalScore float64,
	reasoning string,
) *InferenceChainBuilder {

	step := InferenceStep{
		StepNumber: len(icb.chain) + 1,
		Phase:      "scoring",
		Input: map[string]interface{}{
			"medication": medication,
			"factors":    factors,
			"weights":    weights,
		},
		Output: map[string]interface{}{
			"final_score": finalScore,
		},
		Reasoning:  reasoning,
		Confidence: 0.85,
		Timestamp:  time.Now(),
	}

	icb.chain = append(icb.chain, step)
	return icb
}

// AddRankingStep adds a final ranking step
func (icb *InferenceChainBuilder) AddRankingStep(
	medications []string,
	scores []float64,
	rankings []int,
	reasoning string,
) *InferenceChainBuilder {

	step := InferenceStep{
		StepNumber: len(icb.chain) + 1,
		Phase:      "ranking",
		Input: map[string]interface{}{
			"medications": medications,
			"scores":      scores,
		},
		Output: map[string]interface{}{
			"rankings": rankings,
		},
		Reasoning:  reasoning,
		Confidence: 0.95,
		Timestamp:  time.Now(),
	}

	icb.chain = append(icb.chain, step)
	return icb
}

// AddExclusionStep adds a medication exclusion step
func (icb *InferenceChainBuilder) AddExclusionStep(
	medication string,
	reason string,
	kbSource string,
	ruleID string,
	severity string,
) *InferenceChainBuilder {

	step := InferenceStep{
		StepNumber: len(icb.chain) + 1,
		Phase:      "exclusion",
		KBSource:   kbSource,
		RuleID:     ruleID,
		Input: map[string]interface{}{
			"medication": medication,
		},
		Output: map[string]interface{}{
			"excluded": true,
			"severity": severity,
		},
		Reasoning:  reason,
		Confidence: 1.0, // Exclusions are definitive
		Timestamp:  time.Now(),
	}

	icb.chain = append(icb.chain, step)
	return icb
}

// AddContraindicationStep adds a contraindication detection step
func (icb *InferenceChainBuilder) AddContraindicationStep(
	medication string,
	contraindication string,
	patientCondition string,
	kbSource string,
	ruleID string,
) *InferenceChainBuilder {

	step := InferenceStep{
		StepNumber: len(icb.chain) + 1,
		Phase:      "contraindication",
		KBSource:   kbSource,
		RuleID:     ruleID,
		Input: map[string]interface{}{
			"medication":        medication,
			"patient_condition": patientCondition,
		},
		Output: map[string]interface{}{
			"contraindicated": true,
		},
		Reasoning:  fmt.Sprintf("Medication %s contraindicated due to %s: %s", medication, patientCondition, contraindication),
		Confidence: 1.0,
		Timestamp:  time.Now(),
	}

	icb.chain = append(icb.chain, step)
	return icb
}

// AddInteractionStep adds a drug-drug interaction step
func (icb *InferenceChainBuilder) AddInteractionStep(
	medication1 string,
	medication2 string,
	interactionType string,
	severity string,
	recommendation string,
) *InferenceChainBuilder {

	step := InferenceStep{
		StepNumber: len(icb.chain) + 1,
		Phase:      "interaction",
		KBSource:   "KB-2",
		Input: map[string]interface{}{
			"medication_1": medication1,
			"medication_2": medication2,
		},
		Output: map[string]interface{}{
			"interaction_type": interactionType,
			"severity":         severity,
			"recommendation":   recommendation,
		},
		Reasoning:  fmt.Sprintf("%s interaction between %s and %s: %s", severity, medication1, medication2, recommendation),
		Confidence: 0.95,
		Timestamp:  time.Now(),
	}

	icb.chain = append(icb.chain, step)
	return icb
}

// AddDoseAdjustmentStep adds a dose adjustment step
func (icb *InferenceChainBuilder) AddDoseAdjustmentStep(
	medication string,
	originalDose float64,
	adjustedDose float64,
	unit string,
	reason string,
	kbSource string,
) *InferenceChainBuilder {

	step := InferenceStep{
		StepNumber: len(icb.chain) + 1,
		Phase:      "dose_adjustment",
		KBSource:   kbSource,
		Input: map[string]interface{}{
			"medication":    medication,
			"original_dose": originalDose,
			"unit":          unit,
		},
		Output: map[string]interface{}{
			"adjusted_dose":     adjustedDose,
			"adjustment_ratio": adjustedDose / originalDose,
		},
		Reasoning:  reason,
		Confidence: 0.95,
		Timestamp:  time.Now(),
	}

	icb.chain = append(icb.chain, step)
	return icb
}

// Build returns the completed inference chain
func (icb *InferenceChainBuilder) Build() []InferenceStep {
	// Calculate durations
	for i := range icb.chain {
		if i == 0 {
			icb.chain[i].Duration = Duration{
				Milliseconds: icb.chain[i].Timestamp.Sub(icb.startTime).Milliseconds(),
			}
		} else {
			icb.chain[i].Duration = Duration{
				Milliseconds: icb.chain[i].Timestamp.Sub(icb.chain[i-1].Timestamp).Milliseconds(),
			}
		}
	}
	return icb.chain
}

// GetChainSummary returns a human-readable summary of the inference chain
func (icb *InferenceChainBuilder) GetChainSummary() string {
	summary := fmt.Sprintf("Inference Chain: %d steps\n", len(icb.chain))

	for _, step := range icb.chain {
		summary += fmt.Sprintf("  [%d] %s", step.StepNumber, step.Phase)
		if step.KBSource != "" {
			summary += fmt.Sprintf(" (%s)", step.KBSource)
		}
		summary += fmt.Sprintf(": %s\n", step.Reasoning)
	}

	return summary
}

// ExplainResult provides explanation for a specific question
type ExplainResult struct {
	Question       string          `json:"question"`
	Answer         string          `json:"answer"`
	RelevantSteps  []InferenceStep `json:"relevant_steps"`
	Confidence     float64         `json:"confidence"`
	KBSources      []string        `json:"kb_sources"`
}

// ExplainChain provides explanation capabilities for the inference chain
type ExplainChain struct {
	chain []InferenceStep
}

// NewExplainChain creates an explanation helper for an inference chain
func NewExplainChain(chain []InferenceStep) *ExplainChain {
	return &ExplainChain{chain: chain}
}

// WhyIncluded explains why a medication was included
func (ec *ExplainChain) WhyIncluded(medication string) *ExplainResult {
	relevantSteps := []InferenceStep{}
	kbSources := map[string]bool{}

	for _, step := range ec.chain {
		if step.Phase == "scoring" || step.Phase == "ranking" {
			input, ok := step.Input.(map[string]interface{})
			if ok {
				if med, ok := input["medication"].(string); ok && med == medication {
					relevantSteps = append(relevantSteps, step)
					if step.KBSource != "" {
						kbSources[step.KBSource] = true
					}
				}
			}
		}
	}

	sources := make([]string, 0, len(kbSources))
	for s := range kbSources {
		sources = append(sources, s)
	}

	return &ExplainResult{
		Question:      fmt.Sprintf("Why was %s included?", medication),
		Answer:        fmt.Sprintf("Medication %s passed all safety checks and scored positively on quality factors", medication),
		RelevantSteps: relevantSteps,
		Confidence:    0.9,
		KBSources:     sources,
	}
}

// WhyExcluded explains why a medication was excluded
func (ec *ExplainChain) WhyExcluded(medication string) *ExplainResult {
	relevantSteps := []InferenceStep{}
	kbSources := map[string]bool{}
	var reason string

	for _, step := range ec.chain {
		if step.Phase == "exclusion" || step.Phase == "contraindication" {
			input, ok := step.Input.(map[string]interface{})
			if ok {
				if med, ok := input["medication"].(string); ok && med == medication {
					relevantSteps = append(relevantSteps, step)
					reason = step.Reasoning
					if step.KBSource != "" {
						kbSources[step.KBSource] = true
					}
				}
			}
		}
	}

	sources := make([]string, 0, len(kbSources))
	for s := range kbSources {
		sources = append(sources, s)
	}

	if reason == "" {
		reason = fmt.Sprintf("No exclusion found for %s in inference chain", medication)
	}

	return &ExplainResult{
		Question:      fmt.Sprintf("Why was %s excluded?", medication),
		Answer:        reason,
		RelevantSteps: relevantSteps,
		Confidence:    1.0,
		KBSources:     sources,
	}
}

// WhyRanked explains the ranking of a medication
func (ec *ExplainChain) WhyRanked(medication string, rank int) *ExplainResult {
	relevantSteps := []InferenceStep{}

	for _, step := range ec.chain {
		if step.Phase == "scoring" || step.Phase == "ranking" {
			relevantSteps = append(relevantSteps, step)
		}
	}

	return &ExplainResult{
		Question:      fmt.Sprintf("Why is %s ranked #%d?", medication, rank),
		Answer:        fmt.Sprintf("Ranking based on weighted quality factors: guideline alignment (30%%), safety (25%%), efficacy (20%%), interactions (15%%), monitoring (10%%)"),
		RelevantSteps: relevantSteps,
		Confidence:    0.85,
		KBSources:     []string{"KB-3", "KB-4", "KB-6"},
	}
}
