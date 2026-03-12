package semantic

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// ReasoningEngine provides clinical reasoning capabilities over the KB-7 ontology
type ReasoningEngine struct {
	graphdb    *GraphDBClient
	logger     *logrus.Logger
	reasoner   string // "HermiT", "Pellet", "GraphDB-OWL"
	ruleEngine *ClinicalRuleEngine
}

// ClinicalRuleEngine handles clinical safety and policy rules
type ClinicalRuleEngine struct {
	rules map[string]ClinicalRule
}

// ClinicalRule represents a clinical reasoning rule
type ClinicalRule interface {
	Evaluate(ctx context.Context, facts map[string]interface{}) (*RuleResult, error)
	GetID() string
	GetDescription() string
	GetPriority() int
}

// RuleResult contains the result of rule evaluation
type RuleResult struct {
	RuleID      string                 `json:"rule_id"`
	Triggered   bool                   `json:"triggered"`
	Confidence  float64                `json:"confidence"`
	Evidence    []string               `json:"evidence"`
	Conclusions []string               `json:"conclusions"`
	Warnings    []string               `json:"warnings"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// InferenceResult contains reasoning results
type InferenceResult struct {
	NewTriples      []TripleData           `json:"new_triples"`
	Inconsistencies []string               `json:"inconsistencies"`
	Warnings        []string               `json:"warnings"`
	ExecutionTime   time.Duration          `json:"execution_time"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// ClinicalContext provides context for reasoning
type ClinicalContext struct {
	PatientID        string                 `json:"patient_id"`
	ClinicalDomain   string                 `json:"clinical_domain"`
	SafetyLevel      string                 `json:"safety_level"`
	RegulatoryFlags  map[string]bool        `json:"regulatory_flags"`
	AdditionalFacts  map[string]interface{} `json:"additional_facts"`
}

// NewReasoningEngine creates a new semantic reasoning engine
func NewReasoningEngine(graphdb *GraphDBClient, logger *logrus.Logger) *ReasoningEngine {
	ruleEngine := &ClinicalRuleEngine{
		rules: make(map[string]ClinicalRule),
	}

	// Register built-in clinical rules
	ruleEngine.RegisterDefaultRules()

	return &ReasoningEngine{
		graphdb:    graphdb,
		logger:     logger,
		reasoner:   "GraphDB-OWL", // Default to GraphDB's built-in OWL reasoner
		ruleEngine: ruleEngine,
	}
}

// PerformInference executes semantic reasoning over the knowledge base
func (r *ReasoningEngine) PerformInference(ctx context.Context, context *ClinicalContext) (*InferenceResult, error) {
	startTime := time.Now()

	r.logger.WithFields(logrus.Fields{
		"reasoner":        r.reasoner,
		"clinical_domain": context.ClinicalDomain,
		"safety_level":    context.SafetyLevel,
	}).Info("Starting semantic inference")

	result := &InferenceResult{
		NewTriples:      []TripleData{},
		Inconsistencies: []string{},
		Warnings:        []string{},
		Metadata:        make(map[string]interface{}),
	}

	// Step 1: Perform OWL reasoning
	owlResults, err := r.performOWLReasoning(ctx, context)
	if err != nil {
		return nil, fmt.Errorf("OWL reasoning failed: %w", err)
	}

	result.NewTriples = append(result.NewTriples, owlResults.NewTriples...)
	result.Inconsistencies = append(result.Inconsistencies, owlResults.Inconsistencies...)

	// Step 2: Apply clinical rules
	ruleResults, err := r.applyClinicalRules(ctx, context)
	if err != nil {
		r.logger.WithError(err).Warn("Clinical rule evaluation had errors")
		result.Warnings = append(result.Warnings, fmt.Sprintf("Rule evaluation errors: %v", err))
	}

	// Process rule results
	for _, ruleResult := range ruleResults {
		if ruleResult.Triggered {
			// Convert rule conclusions to RDF triples
			triples := r.convertConclusionsToTriples(ruleResult, context)
			result.NewTriples = append(result.NewTriples, triples...)

			// Add warnings
			result.Warnings = append(result.Warnings, ruleResult.Warnings...)
		}
	}

	// Step 3: Validate new inferences
	validationWarnings := r.validateInferences(ctx, result.NewTriples, context)
	result.Warnings = append(result.Warnings, validationWarnings...)

	// Calculate execution time
	result.ExecutionTime = time.Since(startTime)

	// Add metadata
	result.Metadata = map[string]interface{}{
		"reasoner":         r.reasoner,
		"clinical_domain":  context.ClinicalDomain,
		"rules_evaluated":  len(r.ruleEngine.rules),
		"inference_count":  len(result.NewTriples),
		"completion_time":  time.Now().Format(time.RFC3339),
	}

	r.logger.WithFields(logrus.Fields{
		"new_triples":      len(result.NewTriples),
		"inconsistencies":  len(result.Inconsistencies),
		"warnings":         len(result.Warnings),
		"execution_time":   result.ExecutionTime,
	}).Info("Semantic inference completed")

	return result, nil
}

// performOWLReasoning executes OWL 2 RL reasoning using GraphDB
func (r *ReasoningEngine) performOWLReasoning(ctx context.Context, context *ClinicalContext) (*InferenceResult, error) {
	result := &InferenceResult{
		NewTriples:      []TripleData{},
		Inconsistencies: []string{},
		Warnings:        []string{},
	}

	// Query for materialized inferences from GraphDB's reasoner
	query := &SPARQLQuery{
		Query: `
			PREFIX kb7: <http://cardiofit.ai/kb7/ontology#>
			PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
			PREFIX owl: <http://www.w3.org/2002/07/owl#>

			SELECT ?subject ?predicate ?object WHERE {
				?subject ?predicate ?object .
				# Filter for inferred triples only
				FILTER NOT EXISTS {
					?subject ?predicate ?object .
					GRAPH ?g { ?subject ?predicate ?object }
					FILTER(?g != <http://cardiofit.ai/kb7/graph/inferred>)
				}
			}
			LIMIT 1000
		`,
	}

	sparqlResults, err := r.graphdb.ExecuteSPARQL(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("executing OWL reasoning query: %w", err)
	}

	// Convert SPARQL results to TripleData
	for _, binding := range sparqlResults.Results.Bindings {
		subject, ok1 := binding["subject"]
		predicate, ok2 := binding["predicate"]
		object, ok3 := binding["object"]

		if ok1 && ok2 && ok3 {
			triple := TripleData{
				Subject:   subject.Value,
				Predicate: predicate.Value,
				Object:    object.Value,
				Context:   "http://cardiofit.ai/kb7/graph/inferred",
			}
			result.NewTriples = append(result.NewTriples, triple)
		}
	}

	// Check for inconsistencies
	inconsistencyQuery := &SPARQLQuery{
		Query: `
			PREFIX owl: <http://www.w3.org/2002/07/owl#>

			ASK {
				?x a owl:Nothing .
			}
		`,
	}

	inconsistencyResults, err := r.graphdb.ExecuteSPARQL(ctx, inconsistencyQuery)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Could not check for inconsistencies: %v", err))
	} else if len(inconsistencyResults.Results.Bindings) > 0 {
		result.Inconsistencies = append(result.Inconsistencies, "Ontology contains logical inconsistencies")
	}

	return result, nil
}

// applyClinicalRules applies clinical safety and policy rules
func (r *ReasoningEngine) applyClinicalRules(ctx context.Context, context *ClinicalContext) ([]*RuleResult, error) {
	var results []*RuleResult

	// Gather facts for rule evaluation
	facts := r.gatherClinicalFacts(ctx, context)

	// Apply each rule
	for _, rule := range r.ruleEngine.rules {
		ruleResult, err := rule.Evaluate(ctx, facts)
		if err != nil {
			r.logger.WithFields(logrus.Fields{
				"rule_id": rule.GetID(),
				"error":   err,
			}).Error("Rule evaluation failed")
			continue
		}

		results = append(results, ruleResult)

		r.logger.WithFields(logrus.Fields{
			"rule_id":    rule.GetID(),
			"triggered":  ruleResult.Triggered,
			"confidence": ruleResult.Confidence,
		}).Debug("Rule evaluated")
	}

	return results, nil
}

// gatherClinicalFacts collects relevant facts for rule evaluation
func (r *ReasoningEngine) gatherClinicalFacts(ctx context.Context, context *ClinicalContext) map[string]interface{} {
	facts := make(map[string]interface{})

	// Add context facts
	facts["patient_id"] = context.PatientID
	facts["clinical_domain"] = context.ClinicalDomain
	facts["safety_level"] = context.SafetyLevel
	facts["regulatory_flags"] = context.RegulatoryFlags

	// Add additional facts
	for key, value := range context.AdditionalFacts {
		facts[key] = value
	}

	// Query for relevant clinical concepts
	if context.ClinicalDomain != "" {
		domainFacts := r.queryDomainFacts(ctx, context.ClinicalDomain)
		for key, value := range domainFacts {
			facts[key] = value
		}
	}

	return facts
}

// queryDomainFacts queries for domain-specific clinical facts
func (r *ReasoningEngine) queryDomainFacts(ctx context.Context, domain string) map[string]interface{} {
	facts := make(map[string]interface{})

	// Query for medication-specific facts
	if strings.Contains(strings.ToLower(domain), "medication") {
		query := &SPARQLQuery{
			Query: `
				PREFIX kb7: <http://cardiofit.ai/kb7/ontology#>

				SELECT ?drug ?interaction ?severity WHERE {
					?drug a kb7:MedicationConcept ;
						kb7:hasInteraction ?interaction .
					?interaction kb7:severity ?severity .
				}
				LIMIT 100
			`,
		}

		results, err := r.graphdb.ExecuteSPARQL(ctx, query)
		if err == nil {
			var interactions []map[string]string
			for _, binding := range results.Results.Bindings {
				interaction := map[string]string{
					"drug":        binding["drug"].Value,
					"interaction": binding["interaction"].Value,
					"severity":    binding["severity"].Value,
				}
				interactions = append(interactions, interaction)
			}
			facts["drug_interactions"] = interactions
		}
	}

	return facts
}

// convertConclusionsToTriples converts rule conclusions to RDF triples
func (r *ReasoningEngine) convertConclusionsToTriples(result *RuleResult, context *ClinicalContext) []TripleData {
	var triples []TripleData

	// Generate rule application triple
	ruleURI := fmt.Sprintf("http://cardiofit.ai/kb7/rule-application/%s_%d", result.RuleID, time.Now().Unix())

	triples = append(triples, TripleData{
		Subject:   ruleURI,
		Predicate: "http://www.w3.org/1999/02/22-rdf-syntax-ns#type",
		Object:    "http://cardiofit.ai/kb7/ontology#RuleApplication",
		Context:   "http://cardiofit.ai/kb7/graph/inferred",
	})

	triples = append(triples, TripleData{
		Subject:   ruleURI,
		Predicate: "http://cardiofit.ai/kb7/ontology#appliedRule",
		Object:    fmt.Sprintf("http://cardiofit.ai/kb7/rule/%s", result.RuleID),
		Context:   "http://cardiofit.ai/kb7/graph/inferred",
	})

	triples = append(triples, TripleData{
		Subject:   ruleURI,
		Predicate: "http://cardiofit.ai/kb7/ontology#confidence",
		Object:    fmt.Sprintf("%.3f", result.Confidence),
		Context:   "http://cardiofit.ai/kb7/graph/inferred",
	})

	// Add conclusions as triples
	for i, conclusion := range result.Conclusions {
		conclusionURI := fmt.Sprintf("%s/conclusion/%d", ruleURI, i)

		triples = append(triples, TripleData{
			Subject:   conclusionURI,
			Predicate: "http://www.w3.org/2000/01/rdf-schema#comment",
			Object:    conclusion,
			Context:   "http://cardiofit.ai/kb7/graph/inferred",
		})
	}

	return triples
}

// validateInferences validates new inferences for consistency and safety
func (r *ReasoningEngine) validateInferences(ctx context.Context, triples []TripleData, context *ClinicalContext) []string {
	var warnings []string

	// Check for high-risk inferences
	for _, triple := range triples {
		if strings.Contains(triple.Predicate, "hasInteraction") {
			warnings = append(warnings, fmt.Sprintf("New drug interaction inferred: %s", triple.Subject))
		}

		if strings.Contains(triple.Predicate, "contraindicated") {
			warnings = append(warnings, fmt.Sprintf("New contraindication inferred: %s", triple.Subject))
		}
	}

	// Validate against clinical safety rules
	if context.SafetyLevel == "critical" && len(triples) > 10 {
		warnings = append(warnings, "Large number of inferences in critical safety context")
	}

	return warnings
}

// Clinical Rule Implementations

// RegisterDefaultRules registers built-in clinical rules
func (c *ClinicalRuleEngine) RegisterDefaultRules() {
	c.rules["drug-interaction-safety"] = &DrugInteractionSafetyRule{}
	c.rules["medication-mapping-safety"] = &MedicationMappingSafetyRule{}
	c.rules["australian-regulatory-compliance"] = &AustralianRegulatoryRule{}
	c.rules["clinical-review-requirement"] = &ClinicalReviewRequirementRule{}
}

// DrugInteractionSafetyRule checks for drug interaction safety
type DrugInteractionSafetyRule struct{}

func (r *DrugInteractionSafetyRule) GetID() string         { return "drug-interaction-safety" }
func (r *DrugInteractionSafetyRule) GetDescription() string { return "Identifies critical drug interactions requiring clinical review" }
func (r *DrugInteractionSafetyRule) GetPriority() int      { return 100 }

func (r *DrugInteractionSafetyRule) Evaluate(ctx context.Context, facts map[string]interface{}) (*RuleResult, error) {
	result := &RuleResult{
		RuleID:      r.GetID(),
		Triggered:   false,
		Confidence:  0.0,
		Evidence:    []string{},
		Conclusions: []string{},
		Warnings:    []string{},
		Metadata:    make(map[string]interface{}),
	}

	// Check for drug interactions in facts
	if interactions, ok := facts["drug_interactions"].([]map[string]string); ok {
		criticalCount := 0
		for _, interaction := range interactions {
			if interaction["severity"] == "critical" || interaction["severity"] == "severe" {
				criticalCount++
				result.Evidence = append(result.Evidence, fmt.Sprintf("Critical interaction: %s", interaction["interaction"]))
			}
		}

		if criticalCount > 0 {
			result.Triggered = true
			result.Confidence = 0.95
			result.Conclusions = append(result.Conclusions, fmt.Sprintf("Found %d critical drug interactions requiring review", criticalCount))
			result.Warnings = append(result.Warnings, "Critical drug interactions detected - manual clinical review required")
		}
	}

	return result, nil
}

// MedicationMappingSafetyRule validates medication mapping safety
type MedicationMappingSafetyRule struct{}

func (r *MedicationMappingSafetyRule) GetID() string         { return "medication-mapping-safety" }
func (r *MedicationMappingSafetyRule) GetDescription() string { return "Validates safety of medication terminology mappings" }
func (r *MedicationMappingSafetyRule) GetPriority() int      { return 90 }

func (r *MedicationMappingSafetyRule) Evaluate(ctx context.Context, facts map[string]interface{}) (*RuleResult, error) {
	result := &RuleResult{
		RuleID:      r.GetID(),
		Triggered:   false,
		Confidence:  0.0,
		Evidence:    []string{},
		Conclusions: []string{},
		Warnings:    []string{},
		Metadata:    make(map[string]interface{}),
	}

	// Check safety level
	if safetyLevel, ok := facts["safety_level"].(string); ok {
		if safetyLevel == "critical" || safetyLevel == "high" {
			result.Triggered = true
			result.Confidence = 0.90
			result.Evidence = append(result.Evidence, fmt.Sprintf("Safety level: %s", safetyLevel))
			result.Conclusions = append(result.Conclusions, "High-risk medication mapping requires clinical review")

			if safetyLevel == "critical" {
				result.Warnings = append(result.Warnings, "Critical safety medication - automated mapping prohibited")
			}
		}
	}

	return result, nil
}

// AustralianRegulatoryRule validates Australian regulatory compliance
type AustralianRegulatoryRule struct{}

func (r *AustralianRegulatoryRule) GetID() string         { return "australian-regulatory-compliance" }
func (r *AustralianRegulatoryRule) GetDescription() string { return "Ensures compliance with Australian healthcare regulations" }
func (r *AustralianRegulatoryRule) GetPriority() int      { return 85 }

func (r *AustralianRegulatoryRule) Evaluate(ctx context.Context, facts map[string]interface{}) (*RuleResult, error) {
	result := &RuleResult{
		RuleID:      r.GetID(),
		Triggered:   false,
		Confidence:  0.0,
		Evidence:    []string{},
		Conclusions: []string{},
		Warnings:    []string{},
		Metadata:    make(map[string]interface{}),
	}

	// Check regulatory flags
	if regFlags, ok := facts["regulatory_flags"].(map[string]bool); ok {
		if regFlags["australian_only"] {
			result.Triggered = true
			result.Confidence = 0.85
			result.Evidence = append(result.Evidence, "Australian healthcare context detected")
			result.Conclusions = append(result.Conclusions, "Apply Australian regulatory compliance rules")

			// Check for required TGA compliance
			if regFlags["requires_tga_review"] {
				result.Warnings = append(result.Warnings, "TGA specialist review required")
			}

			// Check for PBS compliance
			if regFlags["requires_pbs_validation"] {
				result.Warnings = append(result.Warnings, "PBS subsidy validation required")
			}
		}
	}

	return result, nil
}

// ClinicalReviewRequirementRule determines when clinical review is required
type ClinicalReviewRequirementRule struct{}

func (r *ClinicalReviewRequirementRule) GetID() string         { return "clinical-review-requirement" }
func (r *ClinicalReviewRequirementRule) GetDescription() string { return "Determines when clinical specialist review is required" }
func (r *ClinicalReviewRequirementRule) GetPriority() int      { return 80 }

func (r *ClinicalReviewRequirementRule) Evaluate(ctx context.Context, facts map[string]interface{}) (*RuleResult, error) {
	result := &RuleResult{
		RuleID:      r.GetID(),
		Triggered:   false,
		Confidence:  0.0,
		Evidence:    []string{},
		Conclusions: []string{},
		Warnings:    []string{},
		Metadata:    make(map[string]interface{}),
	}

	reviewRequired := false
	confidence := 0.0

	// Check safety level
	if safetyLevel, ok := facts["safety_level"].(string); ok {
		if safetyLevel == "critical" {
			reviewRequired = true
			confidence = 0.95
			result.Evidence = append(result.Evidence, "Critical safety level requires review")
		} else if safetyLevel == "high" {
			reviewRequired = true
			confidence = 0.85
			result.Evidence = append(result.Evidence, "High safety level requires review")
		}
	}

	// Check clinical domain
	if domain, ok := facts["clinical_domain"].(string); ok {
		if strings.Contains(strings.ToLower(domain), "medication") ||
		   strings.Contains(strings.ToLower(domain), "drug") {
			reviewRequired = true
			confidence = max(confidence, 0.80)
			result.Evidence = append(result.Evidence, "Medication domain requires specialist review")
		}
	}

	if reviewRequired {
		result.Triggered = true
		result.Confidence = confidence
		result.Conclusions = append(result.Conclusions, "Clinical specialist review required")
	}

	return result, nil
}

// Utility function for max
func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}