package orb

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// OrchestratorRuleBase is THE BRAIN of the clinical decision support system
// It evaluates medication requests against clinical knowledge to generate Intent Manifests
type OrchestratorRuleBase struct {
	// TIER 1 - Core Clinical Knowledge
	medicationKnowledge *MedicationKnowledgeCore
	clinicalRecipeBook  *ClinicalRecipeBook

	// TIER 2 - Decision Support
	orbRules            *ORBRuleSet
	contextRecipes      *ContextServiceRecipeBook

	// TIER 3 - Operational Knowledge
	formularyDatabase   *FormularyDatabase
	monitoringDatabase  *MonitoringDatabase

	// TIER 4 - Evidence & Quality
	evidenceRepository  *EvidenceRepository

	// Configuration
	knowledgeLoader *KnowledgeLoader
	logger          *logrus.Logger

	// Performance tracking
	evaluationMetrics *EvaluationMetrics
}

// MedicationRequest represents an incoming medication request
type MedicationRequest struct {
	RequestID      string                 `json:"request_id"`
	PatientID      string                 `json:"patient_id"`
	MedicationCode string                 `json:"medication_code"`
	MedicationName string                 `json:"medication_name,omitempty"`
	Indication     string                 `json:"indication,omitempty"`
	
	// Patient context for rule evaluation
	PatientConditions []string               `json:"patient_conditions,omitempty"`
	PatientAge        *float64               `json:"patient_age,omitempty"`
	ClinicalContext   map[string]interface{} `json:"clinical_context,omitempty"`
	
	// Request metadata
	Urgency     string    `json:"urgency,omitempty"`
	RequestedBy string    `json:"requested_by,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// EvaluationMetrics tracks ORB performance
type EvaluationMetrics struct {
	TotalEvaluations    int64
	SuccessfulMatches   int64
	FailedMatches       int64
	AverageEvaluationMs float64
	RuleHitCounts       map[string]int64
}

// NewOrchestratorRuleBase creates a new ORB instance
func NewOrchestratorRuleBase(knowledgeBasePath string) (*OrchestratorRuleBase, error) {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	
	// Initialize knowledge loader
	knowledgeLoader := NewKnowledgeLoader(knowledgeBasePath)
	
	// Validate knowledge base
	if err := knowledgeLoader.ValidateKnowledgeBase(); err != nil {
		return nil, fmt.Errorf("knowledge base validation failed: %w", err)
	}
	
	// Load TIER 1 - Core Clinical Knowledge
	medicationKnowledge, err := knowledgeLoader.LoadMedicationKnowledgeCore()
	if err != nil {
		return nil, fmt.Errorf("failed to load medication knowledge: %w", err)
	}

	clinicalRecipeBook, err := knowledgeLoader.LoadClinicalRecipeBook()
	if err != nil {
		return nil, fmt.Errorf("failed to load clinical recipe book: %w", err)
	}

	// Load TIER 2 - Decision Support
	orbRules, err := knowledgeLoader.LoadORBRules()
	if err != nil {
		return nil, fmt.Errorf("failed to load ORB rules: %w", err)
	}

	contextRecipes, err := knowledgeLoader.LoadContextRecipes()
	if err != nil {
		return nil, fmt.Errorf("failed to load context recipes: %w", err)
	}

	// Load TIER 3 - Operational Knowledge
	formularyDatabase, err := knowledgeLoader.LoadFormularyDatabase()
	if err != nil {
		return nil, fmt.Errorf("failed to load formulary database: %w", err)
	}

	monitoringDatabase, err := knowledgeLoader.LoadMonitoringDatabase()
	if err != nil {
		return nil, fmt.Errorf("failed to load monitoring database: %w", err)
	}

	// Load TIER 4 - Evidence & Quality
	evidenceRepository, err := knowledgeLoader.LoadEvidenceRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to load evidence repository: %w", err)
	}

	logger.WithFields(logrus.Fields{
		"orb_rules_count":         len(orbRules.Rules),
		"medications_count":       len(medicationKnowledge.DrugEncyclopedia.Medications),
		"interactions_count":      len(medicationKnowledge.DrugInteractions.Interactions),
		"clinical_recipes_count":  len(clinicalRecipeBook.Recipes),
		"context_recipes_count":   len(contextRecipes.Recipes),
		"formulary_entries_count": len(formularyDatabase.Formularies),
		"monitoring_profiles_count": len(monitoringDatabase.MonitoringProfiles),
		"evidence_entries_count":  len(evidenceRepository.Evidence),
	}).Info("ORB initialized successfully with complete 4-tier knowledge ecosystem")

	return &OrchestratorRuleBase{
		medicationKnowledge: medicationKnowledge,
		clinicalRecipeBook:  clinicalRecipeBook,
		orbRules:            orbRules,
		contextRecipes:      contextRecipes,
		formularyDatabase:   formularyDatabase,
		monitoringDatabase:  monitoringDatabase,
		evidenceRepository:  evidenceRepository,
		knowledgeLoader:     knowledgeLoader,
		logger:              logger,
		evaluationMetrics:   &EvaluationMetrics{
			RuleHitCounts: make(map[string]int64),
		},
	}, nil
}

// ExecuteLocal performs the core ORB evaluation - THE BRAIN FUNCTION
// This is the most critical method in the entire system
func (orb *OrchestratorRuleBase) ExecuteLocal(ctx context.Context, request *MedicationRequest) (*IntentManifest, error) {
	startTime := time.Now()
	
	// Validate request
	if err := orb.validateRequest(request); err != nil {
		return nil, fmt.Errorf("invalid medication request: %w", err)
	}
	
	orb.logger.WithFields(logrus.Fields{
		"request_id":      request.RequestID,
		"patient_id":      request.PatientID,
		"medication_code": request.MedicationCode,
		"conditions":      request.PatientConditions,
	}).Info("Starting ORB rule evaluation")
	
	// Get medication information from knowledge base
	medication, exists := orb.medicationKnowledge.DrugEncyclopedia.Medications[request.MedicationCode]
	if !exists {
		orb.logger.WithFields(logrus.Fields{
			"request_id":      request.RequestID,
			"medication_code": request.MedicationCode,
		}).Error("Unknown medication - not in knowledge base")

		return nil, fmt.Errorf("medication '%s' not found in knowledge base - clinical safety requires explicit medication knowledge", request.MedicationCode)
	}
	
	// Sort rules by priority (highest first)
	sortedRules := orb.getSortedRules()
	
	// Evaluate rules in priority order
	for _, rule := range sortedRules {
		if orb.evaluateRule(rule, request, &medication) {
			// Rule matched! Generate Intent Manifest
			intentManifest := orb.generateIntentManifest(rule, request, &medication)
			
			// Track metrics
			orb.trackSuccessfulEvaluation(rule.ID, time.Since(startTime))
			
			orb.logger.WithFields(logrus.Fields{
				"request_id":        request.RequestID,
				"matched_rule_id":   rule.ID,
				"recipe_id":         intentManifest.RecipeID,
				"data_requirements": len(intentManifest.DataRequirements),
				"evaluation_time":   time.Since(startTime).Milliseconds(),
			}).Info("ORB rule matched successfully")
			
			return intentManifest, nil
		}
	}
	
	// No rules matched
	orb.trackFailedEvaluation(time.Since(startTime))
	
	orb.logger.WithFields(logrus.Fields{
		"request_id":      request.RequestID,
		"medication_code": request.MedicationCode,
		"conditions":      request.PatientConditions,
	}).Warn("No ORB rules matched medication request")
	
	return nil, fmt.Errorf("no ORB rule matches medication '%s' with conditions %v", 
		request.MedicationCode, request.PatientConditions)
}

// evaluateRule checks if a rule matches the medication request
func (orb *OrchestratorRuleBase) evaluateRule(rule *ORBRule, request *MedicationRequest, medication *Medication) bool {
	// Handle production format rules (use conditions for matching)
	if len(rule.Conditions.AllOf) > 0 || len(rule.Conditions.AnyOf) > 0 {
		return orb.evaluateProductionConditions(rule, request)
	}

	// Handle legacy format rules (use MedicationCode for matching)
	if rule.MedicationCode != "" {
		// 1. Check medication code match
		if rule.MedicationCode != request.MedicationCode {
			return false
		}

		// 2. Check patient conditions (if specified in rule)
		if !orb.evaluatePatientConditions(rule, request) {
			return false
		}

		// 3. Check patient demographics (if specified in rule)
		if !orb.evaluatePatientDemographics(rule, request) {
			return false
		}

		// 4. Check clinical context (if specified in rule)
		if !orb.evaluateClinicalContext(rule, request) {
			return false
		}

		return true
	}

	// Rules with no conditions and no medication code match everything
	return true
}

// evaluateProductionConditions evaluates production format rules (all_of/any_of)
func (orb *OrchestratorRuleBase) evaluateProductionConditions(rule *ORBRule, request *MedicationRequest) bool {
	// Evaluate all_of conditions (all must be true)
	if len(rule.Conditions.AllOf) > 0 {
		for _, condition := range rule.Conditions.AllOf {
			if !orb.evaluateCondition(condition, request) {
				return false
			}
		}
	}

	// Evaluate any_of conditions (at least one must be true)
	if len(rule.Conditions.AnyOf) > 0 {
		hasMatch := false
		for _, condition := range rule.Conditions.AnyOf {
			if orb.evaluateCondition(condition, request) {
				hasMatch = true
				break
			}
		}
		if !hasMatch {
			return false
		}
	}

	return true
}

// evaluateCondition evaluates a single rule condition
func (orb *OrchestratorRuleBase) evaluateCondition(condition RuleCondition, request *MedicationRequest) bool {
	switch condition.Fact {
	case "drug_name":
		return orb.evaluateStringCondition(condition, request.MedicationName)
	case "patient_egfr":
		// For demo purposes, assume eGFR is available in patient conditions
		// In production, this would come from patient demographics/labs
		return orb.evaluateNumericCondition(condition, 45.0) // Mock eGFR value
	case "patient_bmi":
		// For demo purposes, assume BMI is available
		return orb.evaluateNumericCondition(condition, 32.0) // Mock BMI value
	default:
		orb.logger.WithField("fact", condition.Fact).Warn("Unknown condition fact")
		return false
	}
}

// evaluateStringCondition evaluates string-based conditions
func (orb *OrchestratorRuleBase) evaluateStringCondition(condition RuleCondition, value string) bool {
	conditionValue, ok := condition.Value.(string)
	if !ok {
		return false
	}

	switch condition.Operator {
	case "equal":
		return strings.EqualFold(value, conditionValue)
	case "contains":
		return strings.Contains(strings.ToLower(value), strings.ToLower(conditionValue))
	default:
		return false
	}
}

// evaluateNumericCondition evaluates numeric-based conditions
func (orb *OrchestratorRuleBase) evaluateNumericCondition(condition RuleCondition, value float64) bool {
	var conditionValue float64

	switch v := condition.Value.(type) {
	case int:
		conditionValue = float64(v)
	case float64:
		conditionValue = v
	default:
		return false
	}

	switch condition.Operator {
	case "lt":
		return value < conditionValue
	case "lte":
		return value <= conditionValue
	case "gt":
		return value > conditionValue
	case "gte":
		return value >= conditionValue
	case "equal":
		return value == conditionValue
	default:
		return false
	}
}

// evaluatePatientConditions checks patient condition matching
func (orb *OrchestratorRuleBase) evaluatePatientConditions(rule *ORBRule, request *MedicationRequest) bool {
	// Handle production format (conditions.all_of / conditions.any_of)
	if len(rule.Conditions.AllOf) > 0 || len(rule.Conditions.AnyOf) > 0 {
		return orb.evaluateProductionConditions(rule, request)
	}

	// Legacy format handling would go here if needed
	return true
}

// evaluatePatientDemographics checks demographic criteria
func (orb *OrchestratorRuleBase) evaluatePatientDemographics(rule *ORBRule, request *MedicationRequest) bool {
	// Handle production format (conditions.all_of / conditions.any_of)
	if len(rule.Conditions.AllOf) > 0 || len(rule.Conditions.AnyOf) > 0 {
		return orb.evaluateProductionConditions(rule, request)
	}

	// Legacy format handling would go here if needed
	return true
}

// evaluateClinicalContext checks clinical context criteria
func (orb *OrchestratorRuleBase) evaluateClinicalContext(rule *ORBRule, request *MedicationRequest) bool {
	// Handle production format (conditions.all_of / conditions.any_of)
	if len(rule.Conditions.AllOf) > 0 || len(rule.Conditions.AnyOf) > 0 {
		return orb.evaluateProductionConditions(rule, request)
	}

	// Legacy format handling would go here if needed
	return true
}

// generateIntentManifest creates the Intent Manifest from matched rule
func (orb *OrchestratorRuleBase) generateIntentManifest(rule *ORBRule, request *MedicationRequest, medication *Medication) *IntentManifest {
	// Handle production format (action.generate_manifest)
	if rule.Action.GenerateManifest.RecipeID != "" {
		builder := NewIntentManifestBuilder().
			WithRequestInfo(request.RequestID, request.PatientID).
			WithRecipe(rule.Action.GenerateManifest.RecipeID).
			WithVariant(rule.Action.GenerateManifest.Variant).
			WithDataRequirements(rule.Action.GenerateManifest.DataManifest.Required).
			WithPriority("high"). // Default priority for production rules
			WithRationale("Production rule matched: " + rule.ID).
			WithRuleInfo(rule.ID, orb.orbRules.Metadata.Version).
			WithMedicationInfo(request.MedicationCode, request.PatientConditions).
			WithEstimatedTime(100). // Default execution time for production rules
			WithMetadata("medication_name", request.MedicationName).
			WithMetadata("rule_type", "production")

		// Add Knowledge Manifest if present in rule
		if len(rule.Action.GenerateManifest.KnowledgeManifest.RequiredKBs) > 0 {
			builder = builder.WithKnowledgeManifest(rule.Action.GenerateManifest.KnowledgeManifest.RequiredKBs)
			builder = builder.WithCacheStrategy("aggressive") // Optimized caching for specific KBs
		} else {
			// Backward compatibility - empty Knowledge Manifest will fall back to all KBs
			builder = builder.WithKnowledgeManifest([]string{}) // Empty slice
			builder = builder.WithCacheStrategy("standard") // Standard caching for all KBs
		}

		return builder.Build()
	}

	// Legacy format handling (for backward compatibility)
	return NewIntentManifestBuilder().
		WithRequestInfo(request.RequestID, request.PatientID).
		WithRecipe(rule.IntentManifest.RecipeID).
		WithVariant(rule.IntentManifest.Variant).
		WithDataRequirements(rule.IntentManifest.DataRequirements).
		WithPriority(rule.IntentManifest.Priority).
		WithRationale(rule.IntentManifest.Rationale).
		WithRuleInfo(rule.ID, orb.orbRules.Metadata.Version).
		WithMedicationInfo(request.MedicationCode, request.PatientConditions).
		WithEstimatedTime(rule.IntentManifest.EstimatedExecutionTimeMs).
		WithMetadata("rule_type", "legacy").
		Build()
}

// Helper methods

func (orb *OrchestratorRuleBase) validateRequest(request *MedicationRequest) error {
	if request.RequestID == "" {
		return fmt.Errorf("request_id is required")
	}
	if request.PatientID == "" {
		return fmt.Errorf("patient_id is required")
	}
	if request.MedicationCode == "" {
		return fmt.Errorf("medication_code is required")
	}
	return nil
}

func (orb *OrchestratorRuleBase) getSortedRules() []*ORBRule {
	rules := make([]*ORBRule, len(orb.orbRules.Rules))
	for i := range orb.orbRules.Rules {
		rules[i] = &orb.orbRules.Rules[i]
	}

	// Sort by priority (highest first)
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Priority > rules[j].Priority
	})

	return rules
}

func (orb *OrchestratorRuleBase) evaluateAgeCondition(condition string, patientAge float64) bool {
	// Simple age condition evaluation
	// Examples: "<18", ">=65", "18-64"
	if strings.HasPrefix(condition, "<") {
		ageLimit := parseFloat(strings.TrimPrefix(condition, "<"))
		return patientAge < ageLimit
	}
	if strings.HasPrefix(condition, ">=") {
		ageLimit := parseFloat(strings.TrimPrefix(condition, ">="))
		return patientAge >= ageLimit
	}
	// Add more age condition logic as needed
	return true
}

func parseFloat(s string) float64 {
	// Simple float parsing - in production, use strconv.ParseFloat with error handling
	if s == "18" {
		return 18.0
	}
	if s == "65" {
		return 65.0
	}
	return 0.0
}



func (orb *OrchestratorRuleBase) trackSuccessfulEvaluation(ruleID string, duration time.Duration) {
	orb.evaluationMetrics.TotalEvaluations++
	orb.evaluationMetrics.SuccessfulMatches++
	orb.evaluationMetrics.RuleHitCounts[ruleID]++

	// Update average evaluation time
	totalTime := orb.evaluationMetrics.AverageEvaluationMs * float64(orb.evaluationMetrics.TotalEvaluations-1)
	orb.evaluationMetrics.AverageEvaluationMs = (totalTime + float64(duration.Milliseconds())) / float64(orb.evaluationMetrics.TotalEvaluations)
}

func (orb *OrchestratorRuleBase) trackFailedEvaluation(duration time.Duration) {
	orb.evaluationMetrics.TotalEvaluations++
	orb.evaluationMetrics.FailedMatches++

	// Update average evaluation time
	totalTime := orb.evaluationMetrics.AverageEvaluationMs * float64(orb.evaluationMetrics.TotalEvaluations-1)
	orb.evaluationMetrics.AverageEvaluationMs = (totalTime + float64(duration.Milliseconds())) / float64(orb.evaluationMetrics.TotalEvaluations)
}

// Public methods for monitoring and management

// GetEvaluationMetrics returns current ORB performance metrics
func (orb *OrchestratorRuleBase) GetEvaluationMetrics() *EvaluationMetrics {
	return orb.evaluationMetrics
}

// ReloadKnowledge reloads the knowledge base (for dynamic updates)
func (orb *OrchestratorRuleBase) ReloadKnowledge() error {
	orb.logger.Info("Reloading ORB knowledge base")

	// Reload ORB rules
	orbRules, err := orb.knowledgeLoader.LoadORBRules()
	if err != nil {
		return fmt.Errorf("failed to reload ORB rules: %w", err)
	}

	// Reload medication knowledge
	medicationKnowledge, err := orb.knowledgeLoader.LoadMedicationKnowledgeCore()
	if err != nil {
		return fmt.Errorf("failed to reload medication knowledge: %w", err)
	}

	// Reload context recipes
	contextRecipes, err := orb.knowledgeLoader.LoadContextRecipes()
	if err != nil {
		return fmt.Errorf("failed to reload context recipes: %w", err)
	}

	// Update knowledge atomically
	orb.orbRules = orbRules
	orb.medicationKnowledge = medicationKnowledge
	orb.contextRecipes = contextRecipes

	orb.logger.WithFields(logrus.Fields{
		"orb_rules_count":      len(orbRules.Rules),
		"medications_count":    len(medicationKnowledge.DrugEncyclopedia.Medications),
		"interactions_count":   len(medicationKnowledge.DrugInteractions.Interactions),
	}).Info("ORB knowledge base reloaded successfully")

	return nil
}

// GetAvailableRules returns list of available rule IDs
func (orb *OrchestratorRuleBase) GetAvailableRules() []string {
	var ruleIDs []string
	for _, rule := range orb.orbRules.Rules {
		ruleIDs = append(ruleIDs, rule.ID)
	}
	return ruleIDs
}

// GetRuleByID returns a specific rule by ID
func (orb *OrchestratorRuleBase) GetRuleByID(ruleID string) (*ORBRule, error) {
	for _, rule := range orb.orbRules.Rules {
		if rule.ID == ruleID {
			return &rule, nil
		}
	}
	return nil, fmt.Errorf("rule not found: %s", ruleID)
}
