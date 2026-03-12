package cdss

import (
	"context"
	"fmt"
	"time"

	"kb-7-terminology/internal/models"
	"kb-7-terminology/internal/services"

	"github.com/sirupsen/logrus"
)

// ============================================================================
// CDSS Rule Engine
// ============================================================================
// The Rule Engine evaluates clinical rules against a patient's fact set.
// Rules can include:
// - Value Set membership conditions
// - Lab threshold conditions (e.g., Lactate > 2.0)
// - Compound conditions (A AND B, A OR B)
// - Temporal conditions (e.g., creatinine increased 50% in 48h)
// - Negation conditions (NOT on comfort care)

// ============================================================================
// Rule Types and Conditions
// ============================================================================

// ConditionType defines the type of rule condition
type ConditionType string

const (
	// ConditionTypeValueSet - Fact matches a value set
	ConditionTypeValueSet ConditionType = "value_set"
	// ConditionTypeThreshold - Numeric value meets threshold
	ConditionTypeThreshold ConditionType = "threshold"
	// ConditionTypeCompound - Compound condition (AND/OR)
	ConditionTypeCompound ConditionType = "compound"
	// ConditionTypeTemporal - Temporal condition (change over time)
	ConditionTypeTemporal ConditionType = "temporal"
	// ConditionTypePresent - Fact type is present (any code)
	ConditionTypePresent ConditionType = "present"
	// ConditionTypeAbsent - Fact type is absent (for negation)
	ConditionTypeAbsent ConditionType = "absent"
	// ConditionTypeCount - Count facts matching a value set (e.g., >=2 nephrotoxins)
	ConditionTypeCount ConditionType = "count"
)

// ThresholdOperator defines comparison operators for threshold conditions
type ThresholdOperator string

const (
	OpGreaterThan         ThresholdOperator = ">"
	OpGreaterThanOrEqual  ThresholdOperator = ">="
	OpLessThan            ThresholdOperator = "<"
	OpLessThanOrEqual     ThresholdOperator = "<="
	OpEqual               ThresholdOperator = "=="
	OpNotEqual            ThresholdOperator = "!="
	OpBetween             ThresholdOperator = "between"
	OpOutside             ThresholdOperator = "outside"
)

// CompoundOperator defines logical operators for compound conditions
type CompoundOperator string

const (
	OpAnd CompoundOperator = "AND"
	OpOr  CompoundOperator = "OR"
	OpNot CompoundOperator = "NOT"
)

// RuleCondition represents a single condition in a clinical rule
type RuleCondition struct {
	// Condition type
	Type ConditionType `json:"type"`

	// For VALUE_SET conditions: the value set to match
	ValueSetID string `json:"value_set_id,omitempty"`

	// For THRESHOLD conditions: the threshold to compare
	FactType   models.FactType   `json:"fact_type,omitempty"`
	LoincCode  string            `json:"loinc_code,omitempty"` // Specific lab code
	Operator   ThresholdOperator `json:"operator,omitempty"`
	Value      float64           `json:"value,omitempty"`
	ValueHigh  float64           `json:"value_high,omitempty"` // For BETWEEN/OUTSIDE
	Unit       string            `json:"unit,omitempty"`

	// For COMPOUND conditions: sub-conditions
	SubConditions    []RuleCondition  `json:"sub_conditions,omitempty"`
	CompoundOperator CompoundOperator `json:"compound_operator,omitempty"`

	// For TEMPORAL conditions
	TimeWindowHours int     `json:"time_window_hours,omitempty"`
	ChangePercent   float64 `json:"change_percent,omitempty"` // e.g., 50 for 50% increase
	ChangeDirection string  `json:"change_direction,omitempty"` // "increase", "decrease", "any"

	// For PRESENT/ABSENT conditions
	RequiredFactType models.FactType `json:"required_fact_type,omitempty"`

	// For COUNT conditions - count unique facts matching a value set
	CountOperator  ThresholdOperator `json:"count_operator,omitempty"`  // >=, >, ==, etc.
	CountThreshold int               `json:"count_threshold,omitempty"` // Minimum count required
}

// ClinicalRule represents a clinical decision support rule
type ClinicalRule struct {
	// Rule identification
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`

	// Clinical classification
	Domain    models.ClinicalDomain    `json:"domain"`
	Severity  models.CDSSAlertSeverity `json:"severity"`
	Category  string                   `json:"category"` // diagnosis, treatment, monitoring, prevention

	// Rule conditions (all must be true for rule to fire)
	Conditions []RuleCondition `json:"conditions"`

	// Exclusion conditions (if ANY are true, rule should NOT fire)
	// Used for clinical safety - e.g., don't recommend anticoagulation if active bleeding
	Exclusions []RuleCondition `json:"exclusions,omitempty"`

	// Alert generation
	AlertTitle          string   `json:"alert_title"`
	AlertDescription    string   `json:"alert_description"`
	Recommendations     []string `json:"recommendations"`
	GuidelineReferences []string `json:"guideline_references,omitempty"`

	// Rule metadata
	Enabled    bool       `json:"enabled"`
	Priority   int        `json:"priority"` // Lower = higher priority
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	Author     string     `json:"author,omitempty"`
}

// FiredRule represents a rule that has been triggered
type FiredRule struct {
	Rule          *ClinicalRule         `json:"rule"`
	FiredAt       time.Time             `json:"fired_at"`
	MatchingFacts []models.ClinicalFact `json:"matching_facts"`
	Evidence      []RuleEvidence        `json:"evidence"`
}

// RuleEvidence represents evidence that contributed to a rule firing
type RuleEvidence struct {
	ConditionType string  `json:"condition_type"`
	Description   string  `json:"description"`
	FactID        string  `json:"fact_id,omitempty"`
	FactType      string  `json:"fact_type,omitempty"`      // condition, observation, medication, etc.
	Code          string  `json:"code,omitempty"`
	System        string  `json:"system,omitempty"`         // SNOMED, LOINC, etc.
	Display       string  `json:"display,omitempty"`
	ValueSetID    string  `json:"value_set_id,omitempty"`
	ValueSetName  string  `json:"value_set_name,omitempty"` // Human-readable value set name
	MatchType     string  `json:"match_type,omitempty"`     // exact, subsumption, expansion
	NumericValue  float64 `json:"numeric_value,omitempty"`
	Unit          string  `json:"unit,omitempty"`           // mmol/L, mg/dL, etc.
	Threshold     float64 `json:"threshold,omitempty"`
	Operator      string  `json:"operator,omitempty"`
}

// ============================================================================
// Rule Engine Interface
// ============================================================================

// RuleEngine evaluates clinical rules against patient facts
type RuleEngine interface {
	// EvaluateRules evaluates all enabled rules against a fact set
	EvaluateRules(ctx context.Context, factSet *models.PatientFactSet, evaluationResults []models.EvaluationResult) ([]FiredRule, error)

	// EvaluateRule evaluates a single rule against a fact set
	EvaluateRule(ctx context.Context, rule *ClinicalRule, factSet *models.PatientFactSet, evaluationResults []models.EvaluationResult) (*FiredRule, bool, error)

	// LoadRules loads rules from configuration
	LoadRules(ctx context.Context) error

	// GetRules returns all loaded rules
	GetRules() []ClinicalRule

	// GetRulesByDomain returns rules for a specific clinical domain
	GetRulesByDomain(domain models.ClinicalDomain) []ClinicalRule
}

// ============================================================================
// Rule Engine Implementation
// ============================================================================

type ruleEngineImpl struct {
	rules          []ClinicalRule
	ruleManager    services.RuleManager
	ruleRepository RuleRepository // Database repository for persistent rules
	logger         *logrus.Logger
}

// NewRuleEngine creates a new RuleEngine instance
func NewRuleEngine(ruleManager services.RuleManager, logger *logrus.Logger) RuleEngine {
	engine := &ruleEngineImpl{
		ruleManager: ruleManager,
		logger:      logger,
	}
	// Load default clinical rules (in-memory only, no DB)
	engine.loadDefaultRules()
	return engine
}

// NewRuleEngineWithRepository creates a new RuleEngine with database repository support
// This enables persistent rule storage with fallback to in-memory defaults
func NewRuleEngineWithRepository(ruleManager services.RuleManager, ruleRepository RuleRepository, logger *logrus.Logger) RuleEngine {
	engine := &ruleEngineImpl{
		ruleManager:    ruleManager,
		ruleRepository: ruleRepository,
		logger:         logger,
	}
	// Load rules from database with fallback to in-memory defaults
	ctx := context.Background()
	if err := engine.LoadRules(ctx); err != nil {
		logger.WithError(err).Warn("Failed to load rules from database, using in-memory defaults")
		engine.loadDefaultRules()
	}
	return engine
}

// loadDefaultRules loads the default clinical rule set
func (re *ruleEngineImpl) loadDefaultRules() {
	re.rules = GetDefaultClinicalRules()
	re.logger.WithField("rule_count", len(re.rules)).Info("Loaded default clinical rules")
}

// LoadRules loads rules from database (if repository available) with fallback to in-memory defaults
func (re *ruleEngineImpl) LoadRules(ctx context.Context) error {
	// If no repository configured, use in-memory defaults
	if re.ruleRepository == nil {
		re.loadDefaultRules()
		return nil
	}

	// Check if clinical_rules table exists
	tableExists, err := re.ruleRepository.TableExists(ctx)
	if err != nil {
		re.logger.WithError(err).Warn("Failed to check clinical_rules table existence, using in-memory defaults")
		re.loadDefaultRules()
		return nil
	}

	if !tableExists {
		re.logger.Info("clinical_rules table doesn't exist yet, using in-memory defaults")
		re.loadDefaultRules()
		return nil
	}

	// Check rule count in database
	count, err := re.ruleRepository.RuleCount(ctx)
	if err != nil {
		re.logger.WithError(err).Warn("Failed to count rules in database, using in-memory defaults")
		re.loadDefaultRules()
		return nil
	}

	// If database is empty, use in-memory defaults
	if count == 0 {
		re.logger.Info("No rules in database, using in-memory defaults (use POST /v1/rules/seed to populate)")
		re.loadDefaultRules()
		return nil
	}

	// Load rules from database
	dbRules, err := re.ruleRepository.GetAllEnabledRules(ctx)
	if err != nil {
		re.logger.WithError(err).Warn("Failed to load rules from database, using in-memory defaults")
		re.loadDefaultRules()
		return nil
	}

	re.rules = dbRules
	re.logger.WithField("rule_count", len(re.rules)).Info("Loaded clinical rules from database")
	return nil
}

// GetRules returns all loaded rules
func (re *ruleEngineImpl) GetRules() []ClinicalRule {
	return re.rules
}

// GetRulesByDomain returns rules for a specific clinical domain
func (re *ruleEngineImpl) GetRulesByDomain(domain models.ClinicalDomain) []ClinicalRule {
	var domainRules []ClinicalRule
	for _, rule := range re.rules {
		if rule.Domain == domain && rule.Enabled {
			domainRules = append(domainRules, rule)
		}
	}
	return domainRules
}

// EvaluateRules evaluates all enabled rules against a fact set
func (re *ruleEngineImpl) EvaluateRules(ctx context.Context, factSet *models.PatientFactSet, evaluationResults []models.EvaluationResult) ([]FiredRule, error) {
	var firedRules []FiredRule

	// Build lookup maps for efficient access
	vsMatchLookup := buildValueSetMatchLookup(evaluationResults)
	labLookup := buildLabValueLookup(factSet)

	for i := range re.rules {
		rule := &re.rules[i]
		if !rule.Enabled {
			continue
		}

		// Check expiration
		if rule.ExpiresAt != nil && time.Now().After(*rule.ExpiresAt) {
			continue
		}

		firedRule, fired, err := re.evaluateRuleInternal(ctx, rule, factSet, vsMatchLookup, labLookup)
		if err != nil {
			re.logger.WithError(err).WithField("rule_id", rule.ID).Warn("Error evaluating rule")
			continue
		}

		if fired {
			firedRules = append(firedRules, *firedRule)
		}
	}

	// Sort by priority (lower = higher priority)
	sortFiredRulesByPriority(firedRules)

	return firedRules, nil
}

// EvaluateRule evaluates a single rule against a fact set
func (re *ruleEngineImpl) EvaluateRule(ctx context.Context, rule *ClinicalRule, factSet *models.PatientFactSet, evaluationResults []models.EvaluationResult) (*FiredRule, bool, error) {
	vsMatchLookup := buildValueSetMatchLookup(evaluationResults)
	labLookup := buildLabValueLookup(factSet)
	return re.evaluateRuleInternal(ctx, rule, factSet, vsMatchLookup, labLookup)
}

// evaluateRuleInternal evaluates a rule with pre-built lookup maps
func (re *ruleEngineImpl) evaluateRuleInternal(
	ctx context.Context,
	rule *ClinicalRule,
	factSet *models.PatientFactSet,
	vsMatchLookup map[string][]models.EvaluationResult,
	labLookup map[string][]models.ClinicalFact,
) (*FiredRule, bool, error) {

	firedRule := &FiredRule{
		Rule:    rule,
		FiredAt: time.Now(),
	}

	// Evaluate all conditions (AND logic by default)
	allConditionsMet := true
	for _, condition := range rule.Conditions {
		met, evidence, matchingFacts := re.evaluateCondition(ctx, condition, factSet, vsMatchLookup, labLookup)
		if !met {
			allConditionsMet = false
			break
		}
		firedRule.Evidence = append(firedRule.Evidence, evidence...)
		firedRule.MatchingFacts = append(firedRule.MatchingFacts, matchingFacts...)
	}

	if !allConditionsMet {
		return nil, false, nil
	}

	// Check exclusion conditions (if ANY exclusion is met, rule should NOT fire)
	// This is for clinical safety - e.g., don't recommend anticoagulation if active bleeding
	if len(rule.Exclusions) > 0 {
		for _, exclusion := range rule.Exclusions {
			met, _, _ := re.evaluateCondition(ctx, exclusion, factSet, vsMatchLookup, labLookup)
			if met {
				// Exclusion condition was met, so rule should NOT fire
				re.logger.WithFields(map[string]interface{}{
					"rule_id":   rule.ID,
					"exclusion": exclusion.ValueSetID,
				}).Debug("Rule excluded due to matching exclusion condition")
				return nil, false, nil
			}
		}
	}

	return firedRule, true, nil
}

// evaluateCondition evaluates a single condition
func (re *ruleEngineImpl) evaluateCondition(
	ctx context.Context,
	condition RuleCondition,
	factSet *models.PatientFactSet,
	vsMatchLookup map[string][]models.EvaluationResult,
	labLookup map[string][]models.ClinicalFact,
) (bool, []RuleEvidence, []models.ClinicalFact) {

	switch condition.Type {
	case ConditionTypeValueSet:
		return re.evaluateValueSetCondition(condition, vsMatchLookup)

	case ConditionTypeThreshold:
		return re.evaluateThresholdCondition(condition, factSet, labLookup)

	case ConditionTypeCompound:
		return re.evaluateCompoundCondition(ctx, condition, factSet, vsMatchLookup, labLookup)

	case ConditionTypePresent:
		return re.evaluatePresentCondition(condition, factSet)

	case ConditionTypeAbsent:
		return re.evaluateAbsentCondition(condition, factSet)

	case ConditionTypeTemporal:
		return re.evaluateTemporalCondition(condition, factSet, labLookup)

	case ConditionTypeCount:
		return re.evaluateCountCondition(condition, vsMatchLookup)

	default:
		re.logger.WithField("condition_type", condition.Type).Warn("Unknown condition type")
		return false, nil, nil
	}
}

// evaluateValueSetCondition checks if any fact matches a value set
func (re *ruleEngineImpl) evaluateValueSetCondition(condition RuleCondition, vsMatchLookup map[string][]models.EvaluationResult) (bool, []RuleEvidence, []models.ClinicalFact) {
	results, ok := vsMatchLookup[condition.ValueSetID]
	if !ok || len(results) == 0 {
		return false, nil, nil
	}

	var evidence []RuleEvidence
	var matchingFacts []models.ClinicalFact

	for _, result := range results {
		// Extract match type and value set name from matched value sets if available
		matchType := "exact"
		valueSetName := condition.ValueSetID
		if len(result.MatchedValueSets) > 0 {
			for _, vsMatch := range result.MatchedValueSets {
				if vsMatch.ValueSetID == condition.ValueSetID {
					matchType = string(vsMatch.MatchType)
					if vsMatch.ValueSetName != "" {
						valueSetName = vsMatch.ValueSetName
					}
					break
				}
			}
		}

		evidence = append(evidence, RuleEvidence{
			ConditionType: "value_set_match",
			Description:   fmt.Sprintf("Code %s matches value set %s", result.Code, condition.ValueSetID),
			FactID:        result.FactID,
			FactType:      string(result.FactType), // Populate FactType
			Code:          result.Code,
			System:        result.System,           // Populate System
			Display:       result.Display,
			ValueSetID:    condition.ValueSetID,
			ValueSetName:  valueSetName,            // Populate ValueSetName
			MatchType:     matchType,               // Populate MatchType
		})

		matchingFacts = append(matchingFacts, models.ClinicalFact{
			ID:       result.FactID,
			FactType: result.FactType,
			Code:     result.Code,
			System:   result.System,
			Display:  result.Display,
		})
	}

	return true, evidence, matchingFacts
}

// evaluateThresholdCondition checks if a lab value meets a threshold
func (re *ruleEngineImpl) evaluateThresholdCondition(condition RuleCondition, factSet *models.PatientFactSet, labLookup map[string][]models.ClinicalFact) (bool, []RuleEvidence, []models.ClinicalFact) {
	// Find matching labs by LOINC code
	labs, ok := labLookup[condition.LoincCode]
	if !ok || len(labs) == 0 {
		// Also try looking in all observations
		for _, obs := range factSet.Observations {
			if obs.Code == condition.LoincCode && obs.NumericValue != nil {
				labs = append(labs, obs)
			}
		}
	}

	if len(labs) == 0 {
		return false, nil, nil
	}

	var evidence []RuleEvidence
	var matchingFacts []models.ClinicalFact

	for _, lab := range labs {
		if lab.NumericValue == nil {
			continue
		}

		value := *lab.NumericValue
		met := false

		switch condition.Operator {
		case OpGreaterThan:
			met = value > condition.Value
		case OpGreaterThanOrEqual:
			met = value >= condition.Value
		case OpLessThan:
			met = value < condition.Value
		case OpLessThanOrEqual:
			met = value <= condition.Value
		case OpEqual:
			met = value == condition.Value
		case OpNotEqual:
			met = value != condition.Value
		case OpBetween:
			met = value >= condition.Value && value <= condition.ValueHigh
		case OpOutside:
			met = value < condition.Value || value > condition.ValueHigh
		}

		if met {
			evidence = append(evidence, RuleEvidence{
				ConditionType: "threshold",
				Description:   fmt.Sprintf("%s = %.2f %s %s %.2f", lab.Display, value, condition.Operator, string(condition.Operator), condition.Value),
				FactID:        lab.ID,
				FactType:      string(lab.FactType), // Populate FactType (typically "observation")
				Code:          lab.Code,
				System:        lab.System,           // Populate System (typically LOINC)
				Display:       lab.Display,
				NumericValue:  value,
				Unit:          lab.Unit,             // Populate Unit (e.g., mmol/L, mg/dL)
				Threshold:     condition.Value,
				Operator:      string(condition.Operator),
			})
			matchingFacts = append(matchingFacts, lab)
			return true, evidence, matchingFacts
		}
	}

	return false, nil, nil
}

// evaluateCompoundCondition evaluates a compound (AND/OR/NOT) condition
func (re *ruleEngineImpl) evaluateCompoundCondition(
	ctx context.Context,
	condition RuleCondition,
	factSet *models.PatientFactSet,
	vsMatchLookup map[string][]models.EvaluationResult,
	labLookup map[string][]models.ClinicalFact,
) (bool, []RuleEvidence, []models.ClinicalFact) {

	var allEvidence []RuleEvidence
	var allMatchingFacts []models.ClinicalFact

	switch condition.CompoundOperator {
	case OpAnd:
		// All sub-conditions must be true
		for _, subCondition := range condition.SubConditions {
			met, evidence, facts := re.evaluateCondition(ctx, subCondition, factSet, vsMatchLookup, labLookup)
			if !met {
				return false, nil, nil
			}
			allEvidence = append(allEvidence, evidence...)
			allMatchingFacts = append(allMatchingFacts, facts...)
		}
		return true, allEvidence, allMatchingFacts

	case OpOr:
		// At least one sub-condition must be true
		for _, subCondition := range condition.SubConditions {
			met, evidence, facts := re.evaluateCondition(ctx, subCondition, factSet, vsMatchLookup, labLookup)
			if met {
				return true, evidence, facts
			}
		}
		return false, nil, nil

	case OpNot:
		// The sub-condition must be false
		if len(condition.SubConditions) > 0 {
			met, _, _ := re.evaluateCondition(ctx, condition.SubConditions[0], factSet, vsMatchLookup, labLookup)
			if !met {
				return true, []RuleEvidence{{
					ConditionType: "negation",
					Description:   "Required condition is absent",
				}}, nil
			}
		}
		return false, nil, nil
	}

	return false, nil, nil
}

// evaluatePresentCondition checks if a fact type is present
func (re *ruleEngineImpl) evaluatePresentCondition(condition RuleCondition, factSet *models.PatientFactSet) (bool, []RuleEvidence, []models.ClinicalFact) {
	var count int
	var matchingFacts []models.ClinicalFact

	switch condition.RequiredFactType {
	case models.FactTypeCondition:
		count = len(factSet.Conditions)
		for _, f := range factSet.Conditions {
			matchingFacts = append(matchingFacts, f)
		}
	case models.FactTypeObservation, models.FactTypeLab, models.FactTypeVitalSign:
		count = len(factSet.Observations)
		for _, f := range factSet.Observations {
			matchingFacts = append(matchingFacts, f)
		}
	case models.FactTypeMedication:
		count = len(factSet.Medications)
		for _, f := range factSet.Medications {
			matchingFacts = append(matchingFacts, f)
		}
	case models.FactTypeProcedure:
		count = len(factSet.Procedures)
		for _, f := range factSet.Procedures {
			matchingFacts = append(matchingFacts, f)
		}
	case models.FactTypeAllergy:
		count = len(factSet.Allergies)
		for _, f := range factSet.Allergies {
			matchingFacts = append(matchingFacts, f)
		}
	}

	if count > 0 {
		return true, []RuleEvidence{{
			ConditionType: "presence",
			Description:   fmt.Sprintf("Found %d %s facts", count, condition.RequiredFactType),
		}}, matchingFacts
	}

	return false, nil, nil
}

// evaluateAbsentCondition checks if a fact type is absent
func (re *ruleEngineImpl) evaluateAbsentCondition(condition RuleCondition, factSet *models.PatientFactSet) (bool, []RuleEvidence, []models.ClinicalFact) {
	present, _, _ := re.evaluatePresentCondition(condition, factSet)
	if !present {
		return true, []RuleEvidence{{
			ConditionType: "absence",
			Description:   fmt.Sprintf("No %s facts found", condition.RequiredFactType),
		}}, nil
	}
	return false, nil, nil
}

// evaluateTemporalCondition checks for changes over time
func (re *ruleEngineImpl) evaluateTemporalCondition(condition RuleCondition, factSet *models.PatientFactSet, labLookup map[string][]models.ClinicalFact) (bool, []RuleEvidence, []models.ClinicalFact) {
	labs, ok := labLookup[condition.LoincCode]
	if !ok || len(labs) < 2 {
		return false, nil, nil
	}

	// Sort labs by time
	sortLabsByTime(labs)

	// Calculate time window
	windowDuration := time.Duration(condition.TimeWindowHours) * time.Hour
	cutoffTime := time.Now().Add(-windowDuration)

	// Find labs within window
	var windowLabs []models.ClinicalFact
	for _, lab := range labs {
		if lab.EffectiveDateTime != nil && lab.EffectiveDateTime.After(cutoffTime) {
			windowLabs = append(windowLabs, lab)
		}
	}

	if len(windowLabs) < 2 {
		return false, nil, nil
	}

	// Calculate change
	oldest := windowLabs[0]
	newest := windowLabs[len(windowLabs)-1]

	if oldest.NumericValue == nil || newest.NumericValue == nil {
		return false, nil, nil
	}

	oldVal := *oldest.NumericValue
	newVal := *newest.NumericValue

	if oldVal == 0 {
		return false, nil, nil
	}

	changePercent := ((newVal - oldVal) / oldVal) * 100.0

	// Check if change meets criteria
	met := false
	switch condition.ChangeDirection {
	case "increase":
		met = changePercent >= condition.ChangePercent
	case "decrease":
		met = changePercent <= -condition.ChangePercent
	case "any":
		met = abs(changePercent) >= condition.ChangePercent
	}

	if met {
		return true, []RuleEvidence{{
			ConditionType: "temporal_change",
			Description:   fmt.Sprintf("%s changed %.1f%% (%.2f → %.2f) in %dh", newest.Display, changePercent, oldVal, newVal, condition.TimeWindowHours),
			FactID:        newest.ID,
			Code:          newest.Code,
			NumericValue:  changePercent,
		}}, []models.ClinicalFact{oldest, newest}
	}

	return false, nil, nil
}

// evaluateCountCondition counts unique facts matching a value set
// Used for rules like "patient has >=2 nephrotoxic medications"
func (re *ruleEngineImpl) evaluateCountCondition(condition RuleCondition, vsMatchLookup map[string][]models.EvaluationResult) (bool, []RuleEvidence, []models.ClinicalFact) {
	results, ok := vsMatchLookup[condition.ValueSetID]
	if !ok || len(results) == 0 {
		return false, nil, nil
	}

	// Count UNIQUE facts (by FactID) to avoid double-counting
	// A single medication might match multiple value sets, but we only count it once
	seenFactIDs := make(map[string]bool)
	var uniqueFacts []models.EvaluationResult

	for _, result := range results {
		if !seenFactIDs[result.FactID] {
			seenFactIDs[result.FactID] = true
			uniqueFacts = append(uniqueFacts, result)
		}
	}

	count := len(uniqueFacts)

	// Evaluate count against threshold using the specified operator
	met := false
	switch condition.CountOperator {
	case OpGreaterThan:
		met = count > condition.CountThreshold
	case OpGreaterThanOrEqual:
		met = count >= condition.CountThreshold
	case OpLessThan:
		met = count < condition.CountThreshold
	case OpLessThanOrEqual:
		met = count <= condition.CountThreshold
	case OpEqual:
		met = count == condition.CountThreshold
	case OpNotEqual:
		met = count != condition.CountThreshold
	default:
		// Default to >= for backward compatibility
		met = count >= condition.CountThreshold
	}

	if !met {
		return false, nil, nil
	}

	// Build evidence and matching facts
	var evidence []RuleEvidence
	var matchingFacts []models.ClinicalFact

	for _, result := range uniqueFacts {

		// Get match details
		matchType := "exact"
		valueSetName := condition.ValueSetID
		if len(result.MatchedValueSets) > 0 {
			for _, vsMatch := range result.MatchedValueSets {
				if vsMatch.ValueSetID == condition.ValueSetID {
					matchType = string(vsMatch.MatchType)
					if vsMatch.ValueSetName != "" {
						valueSetName = vsMatch.ValueSetName
					}
					break
				}
			}
		}

		evidence = append(evidence, RuleEvidence{
			ConditionType: "count_match",
			Description:   fmt.Sprintf("Matched %s in %s (count: %d/%d)", result.Display, valueSetName, count, condition.CountThreshold),
			FactID:        result.FactID,
			FactType:      string(result.FactType),
			Code:          result.Code,
			System:        result.System,
			Display:       result.Display,
			ValueSetID:    condition.ValueSetID,
			ValueSetName:  valueSetName,
			MatchType:     matchType,
		})

		matchingFacts = append(matchingFacts, models.ClinicalFact{
			ID:       result.FactID,
			FactType: result.FactType,
			Code:     result.Code,
			System:   result.System,
			Display:  result.Display,
		})
	}

	re.logger.WithFields(logrus.Fields{
		"value_set_id": condition.ValueSetID,
		"count":        count,
		"threshold":    condition.CountThreshold,
		"operator":     condition.CountOperator,
		"matched":      true,
	}).Debug("COUNT condition evaluated")

	return true, evidence, matchingFacts
}

// ============================================================================
// Helper Functions
// ============================================================================

// buildValueSetMatchLookup creates a lookup map from value set ID to matching results
func buildValueSetMatchLookup(results []models.EvaluationResult) map[string][]models.EvaluationResult {
	lookup := make(map[string][]models.EvaluationResult)
	for _, result := range results {
		if !result.Matched {
			continue
		}
		for _, vsMatch := range result.MatchedValueSets {
			lookup[vsMatch.ValueSetID] = append(lookup[vsMatch.ValueSetID], result)
		}
	}
	return lookup
}

// buildLabValueLookup creates a lookup map from LOINC code to lab facts
func buildLabValueLookup(factSet *models.PatientFactSet) map[string][]models.ClinicalFact {
	lookup := make(map[string][]models.ClinicalFact)
	for _, obs := range factSet.Observations {
		if obs.FactType == models.FactTypeLab || obs.FactType == models.FactTypeObservation {
			lookup[obs.Code] = append(lookup[obs.Code], obs)
		}
	}
	return lookup
}

// sortFiredRulesByPriority sorts fired rules by their priority (lower = higher priority)
func sortFiredRulesByPriority(rules []FiredRule) {
	for i := 0; i < len(rules)-1; i++ {
		for j := i + 1; j < len(rules); j++ {
			if rules[j].Rule.Priority < rules[i].Rule.Priority {
				rules[i], rules[j] = rules[j], rules[i]
			}
		}
	}
}

// sortLabsByTime sorts labs by effective time
func sortLabsByTime(labs []models.ClinicalFact) {
	for i := 0; i < len(labs)-1; i++ {
		for j := i + 1; j < len(labs); j++ {
			if labs[j].EffectiveDateTime != nil && labs[i].EffectiveDateTime != nil {
				if labs[j].EffectiveDateTime.Before(*labs[i].EffectiveDateTime) {
					labs[i], labs[j] = labs[j], labs[i]
				}
			}
		}
	}
}

// abs returns the absolute value of a float64
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// ============================================================================
// Default Clinical Rules
// ============================================================================

// GetDefaultClinicalRules returns the default set of clinical rules
func GetDefaultClinicalRules() []ClinicalRule {
	now := time.Now()
	return []ClinicalRule{
		// Sepsis with Elevated Lactate (Compound Rule)
		{
			ID:          "sepsis-lactate-elevated",
			Name:        "Sepsis with Elevated Lactate",
			Description: "Alert for sepsis diagnosis with lactate > 2.0 mmol/L",
			Version:     "1.0",
			Domain:      models.DomainSepsis,
			Severity:    models.SeverityCritical,
			Category:    "diagnosis",
			Conditions: []RuleCondition{
				{
					Type:             ConditionTypeCompound,
					CompoundOperator: OpAnd,
					SubConditions: []RuleCondition{
						{Type: ConditionTypeValueSet, ValueSetID: "SepsisDiagnosis"},
						{Type: ConditionTypeThreshold, LoincCode: "2524-7", Operator: OpGreaterThan, Value: 2.0, Unit: "mmol/L"}, // Lactate
					},
				},
			},
			AlertTitle:       "CRITICAL: Sepsis with Elevated Lactate",
			AlertDescription: "Patient has sepsis diagnosis with lactate > 2.0 mmol/L indicating tissue hypoperfusion",
			Recommendations: []string{
				"Initiate Sepsis Hour-1 Bundle immediately",
				"Obtain blood cultures before antibiotics",
				"Administer broad-spectrum antibiotics within 1 hour",
				"Begin 30 mL/kg crystalloid resuscitation if hypotensive",
				"Repeat lactate in 2-4 hours",
			},
			GuidelineReferences: []string{"Surviving Sepsis Campaign 2021"},
			Enabled:             true,
			Priority:            1,
			CreatedAt:           now,
			UpdatedAt:           now,
		},

		// Simple Sepsis Alert (Value Set only)
		{
			ID:          "sepsis-diagnosis",
			Name:        "Sepsis Diagnosis Detected",
			Description: "Alert for any sepsis diagnosis",
			Version:     "1.0",
			Domain:      models.DomainSepsis,
			Severity:    models.SeverityCritical,
			Category:    "diagnosis",
			Conditions: []RuleCondition{
				{Type: ConditionTypeValueSet, ValueSetID: "SepsisDiagnosis"},
			},
			AlertTitle:       "Sepsis Indicator Detected",
			AlertDescription: "Patient has an active sepsis diagnosis",
			Recommendations: []string{
				"Consider Sepsis-3 criteria evaluation",
				"Obtain lactate level if not recent",
				"Assess for source of infection",
				"Review antibiotic coverage",
			},
			Enabled:   true,
			Priority:  2,
			CreatedAt: now,
			UpdatedAt: now,
		},

		// Septic Shock (Sepsis-3 Criteria)
		// Septic shock = sepsis + hypotension requiring vasopressors + lactate > 2 mmol/L
		{
			ID:          "septic-shock",
			Name:        "Septic Shock Detected",
			Description: "Alert for septic shock: sepsis with hypotension/vasopressors and lactate > 2.0 mmol/L",
			Version:     "1.0",
			Domain:      models.DomainSepsis,
			Severity:    models.SeverityCritical,
			Category:    "diagnosis",
			Conditions: []RuleCondition{
				{
					Type:             ConditionTypeCompound,
					CompoundOperator: OpAnd,
					SubConditions: []RuleCondition{
						// Sepsis diagnosis
						{Type: ConditionTypeValueSet, ValueSetID: "SepsisDiagnosis"},
						// Hypotension OR Vasopressor use
						{
							Type:             ConditionTypeCompound,
							CompoundOperator: OpOr,
							SubConditions: []RuleCondition{
								{Type: ConditionTypeThreshold, LoincCode: "8480-6", Operator: OpLessThan, Value: 90.0, Unit: "mmHg"}, // Systolic BP < 90
								{Type: ConditionTypeValueSet, ValueSetID: "Vasopressors"}, // Vasopressor medication
							},
						},
						// Elevated lactate (> 2.0 mmol/L)
						{Type: ConditionTypeThreshold, LoincCode: "2524-7", Operator: OpGreaterThan, Value: 2.0, Unit: "mmol/L"}, // Lactate
					},
				},
			},
			AlertTitle:       "🚨 SEPTIC SHOCK - IMMEDIATE ACTION REQUIRED",
			AlertDescription: "Patient meets Sepsis-3 criteria for septic shock: sepsis with hypotension/vasopressor requirement and lactate > 2.0 mmol/L",
			Recommendations: []string{
				"ACTIVATE SEPTIC SHOCK PROTOCOL IMMEDIATELY",
				"Continue vasopressors to maintain MAP ≥ 65 mmHg",
				"Complete 30 mL/kg crystalloid bolus if not done",
				"Ensure broad-spectrum antibiotics given within 1 hour of recognition",
				"Consider ICU admission for hemodynamic monitoring",
				"Re-measure lactate every 2 hours until trending down",
				"Consider central venous access for vasopressor infusion",
				"Consider arterial line for continuous BP monitoring",
				"Evaluate for source control (drainage, debridement)",
				"Consider corticosteroids if refractory to fluids and vasopressors",
			},
			GuidelineReferences: []string{
				"Surviving Sepsis Campaign 2021",
				"Sepsis-3 Definitions (JAMA 2016)",
			},
			Enabled:   true,
			Priority:  0, // Highest priority - life-threatening condition
			CreatedAt: now,
			UpdatedAt: now,
		},

		// AKI with Creatinine Threshold
		{
			ID:          "aki-creatinine-elevated",
			Name:        "AKI with Elevated Creatinine",
			Description: "Alert for AKI diagnosis with creatinine > 2.0 mg/dL",
			Version:     "1.0",
			Domain:      models.DomainRenal,
			Severity:    models.SeverityHigh,
			Category:    "diagnosis",
			Conditions: []RuleCondition{
				{
					Type:             ConditionTypeCompound,
					CompoundOperator: OpAnd,
					SubConditions: []RuleCondition{
						{Type: ConditionTypeValueSet, ValueSetID: "AUAKIConditions"},
						{Type: ConditionTypeThreshold, LoincCode: "2160-0", Operator: OpGreaterThan, Value: 2.0, Unit: "mg/dL"}, // Creatinine
					},
				},
			},
			AlertTitle:       "AKI with Elevated Creatinine",
			AlertDescription: "Patient has AKI diagnosis with creatinine > 2.0 mg/dL",
			Recommendations: []string{
				"Review and hold nephrotoxic medications",
				"Monitor urine output (target > 0.5 mL/kg/hr)",
				"Consider renal replacement therapy if indicated",
				"Order renal panel in 6-12 hours",
				"Consider nephrology consultation",
			},
			Enabled:   true,
			Priority:  3,
			CreatedAt: now,
			UpdatedAt: now,
		},

		// AKI with Creatinine Rise (Temporal Rule)
		{
			ID:          "aki-creatinine-rise",
			Name:        "AKI - Rapid Creatinine Rise",
			Description: "Alert for 50% creatinine increase within 48 hours",
			Version:     "1.0",
			Domain:      models.DomainRenal,
			Severity:    models.SeverityHigh,
			Category:    "monitoring",
			Conditions: []RuleCondition{
				{
					Type:            ConditionTypeTemporal,
					LoincCode:       "2160-0", // Creatinine
					TimeWindowHours: 48,
					ChangePercent:   50.0,
					ChangeDirection: "increase",
				},
			},
			AlertTitle:       "Rapid Creatinine Rise Detected",
			AlertDescription: "Creatinine increased ≥50% within 48 hours, concerning for AKI progression",
			Recommendations: []string{
				"Evaluate for AKI using KDIGO criteria",
				"Review all nephrotoxic medications",
				"Assess volume status and urine output",
				"Consider renal ultrasound to rule out obstruction",
			},
			Enabled:   true,
			Priority:  3,
			CreatedAt: now,
			UpdatedAt: now,
		},

		// Hypoglycemia Alert
		{
			ID:          "hypoglycemia-critical",
			Name:        "Critical Hypoglycemia",
			Description: "Alert for glucose < 70 mg/dL in diabetic patient",
			Version:     "1.0",
			Domain:      models.DomainMetabolic,
			Severity:    models.SeverityCritical,
			Category:    "monitoring",
			Conditions: []RuleCondition{
				{
					Type:             ConditionTypeCompound,
					CompoundOperator: OpAnd,
					SubConditions: []RuleCondition{
						{Type: ConditionTypeValueSet, ValueSetID: "DiabetesMellitus"},
						{Type: ConditionTypeThreshold, LoincCode: "2339-0", Operator: OpLessThan, Value: 70.0, Unit: "mg/dL"}, // Glucose
					},
				},
			},
			AlertTitle:       "CRITICAL: Hypoglycemia in Diabetic Patient",
			AlertDescription: "Blood glucose < 70 mg/dL in patient with diabetes",
			Recommendations: []string{
				"Administer 15-20g fast-acting carbohydrate if conscious",
				"Give 25mL D50W IV if unconscious or NPO",
				"Recheck glucose in 15 minutes",
				"Review insulin and oral hypoglycemic dosing",
				"Consider reducing basal insulin if recurrent",
			},
			Enabled:   true,
			Priority:  1,
			CreatedAt: now,
			UpdatedAt: now,
		},

		// Heart Failure with Elevated BNP
		{
			ID:          "hf-elevated-bnp",
			Name:        "Heart Failure with Elevated BNP",
			Description: "Alert for heart failure with BNP > 400 pg/mL",
			Version:     "1.0",
			Domain:      models.DomainCardiac,
			Severity:    models.SeverityHigh,
			Category:    "diagnosis",
			Conditions: []RuleCondition{
				{
					Type:             ConditionTypeCompound,
					CompoundOperator: OpAnd,
					SubConditions: []RuleCondition{
						{Type: ConditionTypeValueSet, ValueSetID: "HeartFailure"},
						{Type: ConditionTypeThreshold, LoincCode: "30934-4", Operator: OpGreaterThan, Value: 400.0, Unit: "pg/mL"}, // BNP
					},
				},
			},
			AlertTitle:       "Heart Failure with Elevated BNP",
			AlertDescription: "Patient has heart failure with BNP > 400 pg/mL suggesting decompensation",
			Recommendations: []string{
				"Assess volume status and weight",
				"Consider diuretic adjustment",
				"Review GDMT optimization",
				"Obtain echocardiogram if not recent",
				"Consider cardiology consultation",
			},
			Enabled:   true,
			Priority:  4,
			CreatedAt: now,
			UpdatedAt: now,
		},

		// Respiratory Failure with Low SpO2
		{
			ID:          "resp-failure-hypoxia",
			Name:        "Respiratory Failure with Hypoxia",
			Description: "Alert for respiratory failure with SpO2 < 90%",
			Version:     "1.0",
			Domain:      models.DomainRespiratory,
			Severity:    models.SeverityCritical,
			Category:    "diagnosis",
			Conditions: []RuleCondition{
				{
					Type:             ConditionTypeCompound,
					CompoundOperator: OpAnd,
					SubConditions: []RuleCondition{
						{Type: ConditionTypeValueSet, ValueSetID: "RespiratoryFailure"},
						{Type: ConditionTypeThreshold, LoincCode: "2708-6", Operator: OpLessThan, Value: 90.0, Unit: "%"}, // SpO2
					},
				},
			},
			AlertTitle:       "CRITICAL: Respiratory Failure with Hypoxia",
			AlertDescription: "Patient has respiratory failure with SpO2 < 90%",
			Recommendations: []string{
				"Increase supplemental oxygen",
				"Consider non-invasive ventilation (BiPAP/CPAP)",
				"Obtain arterial blood gas",
				"Prepare for possible intubation",
				"Notify respiratory therapy",
			},
			Enabled:   true,
			Priority:  1,
			CreatedAt: now,
			UpdatedAt: now,
		},

		// Simple value set alerts for each domain
		{
			ID:          "simple-aki",
			Name:        "Acute Kidney Injury Detected",
			Description: "Alert for any AKI diagnosis",
			Version:     "1.0",
			Domain:      models.DomainRenal,
			Severity:    models.SeverityHigh,
			Category:    "diagnosis",
			Conditions: []RuleCondition{
				{Type: ConditionTypeValueSet, ValueSetID: "AUAKIConditions"},
			},
			AlertTitle:       "Acute Kidney Injury Indicator",
			AlertDescription: "Patient has indicators of acute kidney injury",
			Recommendations: []string{
				"Review nephrotoxic medications",
				"Monitor urine output",
				"Order renal function panel",
			},
			Enabled:   true,
			Priority:  5,
			CreatedAt: now,
			UpdatedAt: now,
		},

		{
			ID:          "simple-hf",
			Name:        "Heart Failure Detected",
			Description: "Alert for any heart failure diagnosis",
			Version:     "1.0",
			Domain:      models.DomainCardiac,
			Severity:    models.SeverityHigh,
			Category:    "diagnosis",
			Conditions: []RuleCondition{
				{Type: ConditionTypeValueSet, ValueSetID: "HeartFailure"},
			},
			AlertTitle:       "Heart Failure Indicator",
			AlertDescription: "Patient has heart failure indicators",
			Recommendations: []string{
				"Assess volume status",
				"Review diuretic therapy",
				"Ensure guideline-directed medical therapy",
			},
			Enabled:   true,
			Priority:  5,
			CreatedAt: now,
			UpdatedAt: now,
		},

		{
			ID:          "simple-diabetes",
			Name:        "Diabetes Mellitus Detected",
			Description: "Alert for diabetes mellitus",
			Version:     "1.0",
			Domain:      models.DomainMetabolic,
			Severity:    models.SeverityModerate,
			Category:    "diagnosis",
			Conditions: []RuleCondition{
				{Type: ConditionTypeValueSet, ValueSetID: "DiabetesMellitus"},
			},
			AlertTitle:       "Diabetes Mellitus",
			AlertDescription: "Patient has diabetes mellitus",
			Recommendations: []string{
				"Monitor glucose regularly",
				"Review HbA1c if not recent",
				"Screen for complications",
			},
			Enabled:   true,
			Priority:  6,
			CreatedAt: now,
			UpdatedAt: now,
		},

		// ============================================================================
		// AFib Anticoagulation Rules (CHA₂DS₂-VASc)
		// ============================================================================

		// AFib Anticoagulation Assessment - checks if patient needs anticoagulation
		{
			ID:          "afib-anticoagulation-assessment",
			Name:        "AFib Anticoagulation Assessment Required",
			Description: "Alert for AFib patient who may need anticoagulation therapy",
			Version:     "1.0",
			Domain:      models.DomainCardiac,
			Severity:    models.SeverityHigh,
			Category:    "anticoagulation",
			Conditions: []RuleCondition{
				{
					Type:             ConditionTypeCompound,
					CompoundOperator: OpAnd,
					SubConditions: []RuleCondition{
						{Type: ConditionTypeValueSet, ValueSetID: "AtrialFibrillation"},
						// INR < 2.0 (subtherapeutic or no anticoagulation)
						{Type: ConditionTypeThreshold, LoincCode: LOINCINR, Operator: OpLessThan, Value: 2.0, Unit: "ratio"},
					},
				},
			},
			Exclusions: []RuleCondition{
				{Type: ConditionTypeValueSet, ValueSetID: "ActiveBleeding"},
			},
			AlertTitle:       "HIGH: AFib Patient May Need Anticoagulation",
			AlertDescription: "Patient has atrial fibrillation without apparent therapeutic anticoagulation (INR < 2.0)",
			Recommendations: []string{
				"Calculate CHA₂DS₂-VASc score for stroke risk assessment",
				"If CHA₂DS₂-VASc ≥2 (men) or ≥3 (women): Anticoagulation recommended",
				"Consider DOAC (apixaban, rivaroxaban, edoxaban, dabigatran) over warfarin",
				"If warfarin preferred: Target INR 2.0-3.0",
				"Assess bleeding risk using HAS-BLED score",
				"Document anticoagulation decision and rationale",
			},
			GuidelineReferences: []string{
				"2020 ESC Guidelines for AFib",
				"2019 AHA/ACC/HRS Focused Update on AFib",
			},
			Enabled:   true,
			Priority:  2,
			CreatedAt: now,
			UpdatedAt: now,
		},

		// AFib with Multiple CHA₂DS₂-VASc Risk Factors
		{
			ID:          "afib-stroke-risk-factors",
			Name:        "AFib with Stroke Risk Factors",
			Description: "Alert for AFib with multiple CHA₂DS₂-VASc risk factors indicating high stroke risk",
			Version:     "1.0",
			Domain:      models.DomainCardiac,
			Severity:    models.SeverityHigh,
			Category:    "stroke-prevention",
			Conditions: []RuleCondition{
				{
					Type:             ConditionTypeCompound,
					CompoundOperator: OpAnd,
					SubConditions: []RuleCondition{
						{Type: ConditionTypeValueSet, ValueSetID: "AtrialFibrillation"},
						// At least one major risk factor (each worth +1 or +2 in CHA₂DS₂-VASc)
						{
							Type:             ConditionTypeCompound,
							CompoundOperator: OpOr,
							SubConditions: []RuleCondition{
								{Type: ConditionTypeValueSet, ValueSetID: "Hypertension"},        // +1
								{Type: ConditionTypeValueSet, ValueSetID: "DiabetesMellitus"},    // +1
								{Type: ConditionTypeValueSet, ValueSetID: "HeartFailure"},        // +1
								{Type: ConditionTypeValueSet, ValueSetID: "IschemicStroke"},      // +2 (Stroke/TIA)
								{Type: ConditionTypeValueSet, ValueSetID: "VascularDisease"},     // +1
							},
						},
					},
				},
			},
			AlertTitle:       "HIGH: AFib with Stroke Risk Factors",
			AlertDescription: "Patient has AFib with CHA₂DS₂-VASc risk factor(s) - elevated stroke risk",
			Recommendations: []string{
				"CHA₂DS₂-VASc score likely ≥1 - evaluate need for anticoagulation",
				"If CHA₂DS₂-VASc ≥2: Anticoagulation strongly recommended",
				"Preferred: DOAC over warfarin (unless mechanical valve or moderate-severe mitral stenosis)",
				"Assess renal function for DOAC dosing adjustments",
				"Review for bleeding contraindications before initiating therapy",
			},
			GuidelineReferences: []string{
				"2020 ESC Guidelines for AFib",
				"CHA₂DS₂-VASc Score (Lip et al.)",
			},
			Enabled:   true,
			Priority:  2,
			CreatedAt: now,
			UpdatedAt: now,
		},

		// Subtherapeutic INR in AFib Patient on Warfarin
		{
			ID:          "afib-subtherapeutic-inr",
			Name:        "Subtherapeutic INR in AFib Patient",
			Description: "Alert for AFib patient with INR below therapeutic range",
			Version:     "1.0",
			Domain:      models.DomainCardiac,
			Severity:    models.SeverityHigh,
			Category:    "anticoagulation-monitoring",
			Conditions: []RuleCondition{
				{
					Type:             ConditionTypeCompound,
					CompoundOperator: OpAnd,
					SubConditions: []RuleCondition{
						{Type: ConditionTypeValueSet, ValueSetID: "AtrialFibrillation"},
						{Type: ConditionTypeValueSet, ValueSetID: "Anticoagulants"}, // On anticoagulant
						// INR < 2.0 (below therapeutic range for AFib)
						{Type: ConditionTypeThreshold, LoincCode: LOINCINR, Operator: OpLessThan, Value: 2.0, Unit: "ratio"},
					},
				},
			},
			AlertTitle:       "WARNING: Subtherapeutic INR in AFib Patient on Warfarin",
			AlertDescription: "INR is below therapeutic range (2.0-3.0) - increased stroke risk",
			Recommendations: []string{
				"Review warfarin adherence with patient",
				"Check for drug-drug and drug-food interactions",
				"Consider warfarin dose adjustment",
				"Consider bridging with LMWH if very high stroke risk",
				"Evaluate switch to DOAC for more stable anticoagulation",
				"Recheck INR in 3-7 days after dose adjustment",
			},
			GuidelineReferences: []string{
				"2020 ESC Guidelines for AFib",
				"CHEST Guidelines for Antithrombotic Therapy",
			},
			Enabled:   true,
			Priority:  2,
			CreatedAt: now,
			UpdatedAt: now,
		},

		// ========================================================================
		// RENAL SAFETY RULES (6)
		// ========================================================================

		// 1. NSAID + CKD Risk
		{
			ID:          "nsaid-ckd-risk",
			Name:        "NSAID Use in CKD Patient",
			Description: "Alert for NSAID use in patient with chronic kidney disease - increased AKI risk",
			Version:     "1.0",
			Domain:      models.DomainRenal,
			Severity:    models.SeverityHigh,
			Category:    "medication-safety",
			Conditions: []RuleCondition{
				{
					Type:             ConditionTypeCompound,
					CompoundOperator: OpAnd,
					SubConditions: []RuleCondition{
						{Type: ConditionTypeValueSet, ValueSetID: "NSAIDs"},
						{Type: ConditionTypeValueSet, ValueSetID: "CKDStages"},
					},
				},
			},
			AlertTitle:       "HIGH: NSAID Use in CKD Patient",
			AlertDescription: "Patient with chronic kidney disease is taking NSAIDs - increased risk of acute kidney injury",
			Recommendations: []string{
				"Consider alternative analgesics (acetaminophen, topical NSAIDs)",
				"If NSAID necessary: use lowest effective dose for shortest duration",
				"Monitor renal function weekly while on NSAIDs",
				"Ensure adequate hydration",
				"Counsel patient on NSAID risks in CKD",
			},
			GuidelineReferences: []string{"KDIGO CKD Guidelines 2024"},
			Enabled:             true,
			Priority:            3,
			CreatedAt:           now,
			UpdatedAt:           now,
		},

		// 2. Triple Whammy AKI Risk (NSAID + ACE-I/ARB + Diuretic)
		{
			ID:          "triple-whammy-risk",
			Name:        "Triple Whammy AKI Risk",
			Description: "Alert for concurrent use of NSAID, ACE-I/ARB, and diuretic - high AKI risk",
			Version:     "1.0",
			Domain:      models.DomainRenal,
			Severity:    models.SeverityCritical,
			Category:    "medication-safety",
			Conditions: []RuleCondition{
				{
					Type:             ConditionTypeCompound,
					CompoundOperator: OpAnd,
					SubConditions: []RuleCondition{
						{Type: ConditionTypeValueSet, ValueSetID: "NSAIDs"},
						{Type: ConditionTypeValueSet, ValueSetID: "ACEInhibitorsARBs"},
						{Type: ConditionTypeValueSet, ValueSetID: "Diuretics"},
					},
				},
			},
			AlertTitle:       "🚨 CRITICAL: Triple Whammy Combination Detected",
			AlertDescription: "Patient is on NSAID + ACE-I/ARB + Diuretic combination - very high acute kidney injury risk",
			Recommendations: []string{
				"URGENT: Discontinue NSAID if possible",
				"Consider alternative analgesic (acetaminophen, topical agents, opioids)",
				"If NSAID essential: monitor creatinine every 3-5 days",
				"Counsel patient to avoid OTC NSAIDs (ibuprofen, naproxen)",
				"Ensure adequate hydration - avoid volume depletion",
				"Consider holding ACE-I/ARB temporarily if intercurrent illness",
				"Monitor potassium closely",
			},
			GuidelineReferences: []string{
				"KDIGO AKI Guidelines 2024",
				"Triple Whammy AKI Risk (BMJ 2013)",
			},
			Enabled:   true,
			Priority:  1,
			CreatedAt: now,
			UpdatedAt: now,
		},

		// 3. Aminoglycoside + CKD Risk
		{
			ID:          "aminoglycoside-ckd-risk",
			Name:        "Aminoglycoside Use in CKD",
			Description: "Alert for aminoglycoside use in patient with CKD - nephrotoxicity risk",
			Version:     "1.0",
			Domain:      models.DomainRenal,
			Severity:    models.SeverityHigh,
			Category:    "medication-safety",
			Conditions: []RuleCondition{
				{
					Type:             ConditionTypeCompound,
					CompoundOperator: OpAnd,
					SubConditions: []RuleCondition{
						{Type: ConditionTypeValueSet, ValueSetID: "Aminoglycosides"},
						{Type: ConditionTypeValueSet, ValueSetID: "CKDStages"},
					},
				},
			},
			AlertTitle:       "HIGH: Aminoglycoside Use in CKD Patient",
			AlertDescription: "Aminoglycoside antibiotic in patient with chronic kidney disease - nephrotoxicity risk",
			Recommendations: []string{
				"Consider alternative antibiotic if possible",
				"If aminoglycoside necessary: use extended-interval dosing",
				"Adjust dose based on renal function (CrCl or eGFR)",
				"Monitor trough levels (target <1 mg/L for gentamicin)",
				"Monitor peak levels if indicated",
				"Check creatinine every 2-3 days during therapy",
				"Limit duration to <7 days if possible",
				"Monitor for ototoxicity (hearing, vestibular function)",
			},
			GuidelineReferences: []string{
				"KDIGO CKD Guidelines 2024",
				"Aminoglycoside Dosing Guidelines (IDSA)",
			},
			Enabled:   true,
			Priority:  3,
			CreatedAt: now,
			UpdatedAt: now,
		},

		// 4. Metformin + CKD Contraindication
		{
			ID:          "metformin-ckd-contraindication",
			Name:        "Metformin Contraindicated in Advanced CKD",
			Description: "Alert for metformin use in CKD stage 4-5 - lactic acidosis risk",
			Version:     "1.0",
			Domain:      models.DomainMetabolic,
			Severity:    models.SeverityCritical,
			Category:    "medication-safety",
			Conditions: []RuleCondition{
				{
					Type:             ConditionTypeCompound,
					CompoundOperator: OpAnd,
					SubConditions: []RuleCondition{
						{Type: ConditionTypeValueSet, ValueSetID: "DiabetesMellitus"},
						{Type: ConditionTypeThreshold, LoincCode: LOINCCreatinine, Operator: OpGreaterThan, Value: 2.0, Unit: "mg/dL"},
					},
				},
			},
			AlertTitle:       "🚨 CRITICAL: Metformin May Be Contraindicated",
			AlertDescription: "Elevated creatinine (>2.0 mg/dL) in diabetic patient - metformin contraindication",
			Recommendations: []string{
				"Check if patient is on metformin - DISCONTINUE if eGFR <30 mL/min/1.73m²",
				"Calculate eGFR to assess renal function",
				"If eGFR 30-45: Reduce metformin dose to 500-1000mg daily maximum",
				"If eGFR <30: STOP metformin - lactic acidosis risk",
				"Consider alternative diabetic agents (DPP-4 inhibitors, insulin)",
				"Monitor for lactic acidosis symptoms if metformin recently used",
			},
			GuidelineReferences: []string{
				"ADA Standards of Care 2024",
				"FDA Metformin Safety Communication",
			},
			Enabled:   true,
			Priority:  1,
			CreatedAt: now,
			UpdatedAt: now,
		},

		// 5. SGLT2 Inhibitor + CKD Monitoring
		{
			ID:          "sglt2-ckd-monitoring",
			Name:        "SGLT2 Inhibitor Renal Monitoring",
			Description: "Alert for SGLT2 inhibitor use with renal impairment - monitoring required",
			Version:     "1.0",
			Domain:      models.DomainMetabolic,
			Severity:    models.SeverityModerate,
			Category:    "medication-monitoring",
			Conditions: []RuleCondition{
				{
					Type:             ConditionTypeCompound,
					CompoundOperator: OpAnd,
					SubConditions: []RuleCondition{
						{Type: ConditionTypeValueSet, ValueSetID: "DiabetesMellitus"},
						{Type: ConditionTypeValueSet, ValueSetID: "CKDStages"},
					},
				},
			},
			AlertTitle:       "SGLT2 Inhibitor Renal Monitoring",
			AlertDescription: "Diabetic patient with CKD - consider SGLT2 inhibitor for renal protection",
			Recommendations: []string{
				"SGLT2 inhibitors (empagliflozin, dapagliflozin, canagliflozin) show renal benefit in CKD",
				"Can use if eGFR ≥20 mL/min/1.73m² for kidney protection",
				"Monitor eGFR - expect initial 5-10% decline (acceptable and reversible)",
				"Assess volume status - risk of volume depletion",
				"Monitor for genital mycotic infections",
				"Educate on euglycemic DKA risk (rare but serious)",
			},
			GuidelineReferences: []string{
				"KDIGO CKD-Diabetes Guidelines 2022",
				"ADA Standards of Care 2024",
			},
			Enabled:   true,
			Priority:  5,
			CreatedAt: now,
			UpdatedAt: now,
		},

		// 6. ACE-I/ARB + Hyperkalemia Risk
		{
			ID:          "ace-arb-hyperkalemia-risk",
			Name:        "ACE-I/ARB Hyperkalemia Risk in CKD",
			Description: "Alert for ACE-I/ARB use in CKD with elevated potassium",
			Version:     "1.0",
			Domain:      models.DomainRenal,
			Severity:    models.SeverityHigh,
			Category:    "medication-safety",
			Conditions: []RuleCondition{
				{
					Type:             ConditionTypeCompound,
					CompoundOperator: OpAnd,
					SubConditions: []RuleCondition{
						{Type: ConditionTypeValueSet, ValueSetID: "ACEInhibitorsARBs"},
						{Type: ConditionTypeValueSet, ValueSetID: "CKDStages"},
						{Type: ConditionTypeThreshold, LoincCode: LOINCPotassium, Operator: OpGreaterThan, Value: 5.0, Unit: "mmol/L"},
					},
				},
			},
			AlertTitle:       "HIGH: Hyperkalemia Risk with ACE-I/ARB in CKD",
			AlertDescription: "Patient on ACE-I/ARB with CKD and elevated potassium - hyperkalemia risk",
			Recommendations: []string{
				"Repeat potassium to confirm (hemolysis can falsely elevate)",
				"If K+ >5.5: Consider holding ACE-I/ARB temporarily",
				"If K+ 5.0-5.5: Continue ACE-I/ARB with close monitoring",
				"Review other K+-raising medications (K+ supplements spironolactone, NSAIDs)",
				"Consider low potassium diet counseling",
				"Consider patiromer or sodium zirconium cyclosilicate if recurrent hyperkalemia",
				"Recheck potassium in 3-7 days",
			},
			GuidelineReferences: []string{
				"KDIGO CKD Guidelines 2024",
				"Hyperkalemia Management (KDIGO 2024)",
			},
			Enabled:   true,
			Priority:  3,
			CreatedAt: now,
			UpdatedAt: now,
		},

		// ========================================================================
		// CARDIAC / ANTICOAGULATION RULES (5)
		// ========================================================================

		// 7. CHA₂DS₂-VASc Anticoagulation (Enhanced)
		{
			ID:          "chadsvasc-anticoagulation",
			Name:        "CHA₂DS₂-VASc High Stroke Risk",
			Description: "AFib patient with multiple CHA₂DS₂-VASc risk factors - anticoagulation indicated",
			Version:     "1.0",
			Domain:      models.DomainCardiac,
			Severity:    models.SeverityCritical,
			Category:    "stroke-prevention",
			Conditions: []RuleCondition{
				{
					Type:             ConditionTypeCompound,
					CompoundOperator: OpAnd,
					SubConditions: []RuleCondition{
						{Type: ConditionTypeValueSet, ValueSetID: "AtrialFibrillation"},
						// At least 2 risk factors (high stroke risk)
						{
							Type:             ConditionTypeCompound,
							CompoundOperator: OpOr,
							SubConditions: []RuleCondition{
								// Prior stroke/TIA (2 points - automatic anticoagulation indication)
								{Type: ConditionTypeValueSet, ValueSetID: "IschemicStroke"},
								// Or combination of risk factors
								{
									Type:             ConditionTypeCompound,
									CompoundOperator: OpAnd,
									SubConditions: []RuleCondition{
										{Type: ConditionTypeValueSet, ValueSetID: "Hypertension"},
										{Type: ConditionTypeValueSet, ValueSetID: "DiabetesMellitus"},
									},
								},
								{
									Type:             ConditionTypeCompound,
									CompoundOperator: OpAnd,
									SubConditions: []RuleCondition{
										{Type: ConditionTypeValueSet, ValueSetID: "HeartFailure"},
										{Type: ConditionTypeValueSet, ValueSetID: "VascularDisease"},
									},
								},
							},
						},
					},
				},
			},
			Exclusions: []RuleCondition{
				{Type: ConditionTypeValueSet, ValueSetID: "ActiveBleeding"},
			},
			AlertTitle:       "🚨 HIGH: AFib with High Stroke Risk - Anticoagulation Required",
			AlertDescription: "Patient has AFib with CHA₂DS₂-VASc ≥2 - anticoagulation strongly indicated",
			Recommendations: []string{
				"CHA₂DS₂-VASc score ≥2: Anticoagulation STRONGLY RECOMMENDED",
				"Preferred: DOAC (apixaban, rivaroxaban, edoxaban, dabigatran)",
				"If mechanical valve or moderate-severe mitral stenosis: Warfarin (target INR 2.0-3.0)",
				"Assess bleeding risk with HAS-BLED score (≥3 = high bleeding risk)",
				"Check renal function for DOAC dosing (adjust if CrCl 15-50 mL/min)",
				"Consider gasteroprotection (PPI) if age >75 or prior GI bleed",
				"Annual stroke risk ~4-6% without anticoagulation",
			},
			GuidelineReferences: []string{
				"2020 ESC Guidelines for AFib",
				"2023 ACC Expert Consensus on AFib",
			},
			Enabled:   true,
			Priority:  1,
			CreatedAt: now,
			UpdatedAt: now,
		},

		// 8. AFib Rate Control Assessment
		{
			ID:          "afib-rate-control-assessment",
			Name:        "AFib Rate Control Assessment",
			Description: "AFib patient - assess rate control and rhythm management",
			Version:     "1.0",
			Domain:      models.DomainCardiac,
			Severity:    models.SeverityModerate,
			Category:    "rhythm-management",
			Conditions: []RuleCondition{
				{Type: ConditionTypeValueSet, ValueSetID: "AtrialFibrillation"},
			},
			AlertTitle:       "AFib Detected - Rate/Rhythm Assessment",
			AlertDescription: "Patient has atrial fibrillation - ensure appropriate rate/rhythm control",
			Recommendations: []string{
				"Assess heart rate control (target: resting HR 60-100 bpm)",
				"If inadequate rate control: consider beta-blocker (metoprolol, bisoprolol) or CCB (diltiazem)",
				"If newly diagnosed AFib: consider rhythm control (cardioversion + antiarrhythmic)",
				"Rhythm control preferred if: age <65, first episode, symptomatic, HFrEF",
				"If persistent AFib >12 months: rate control usually preferred",
				"Ensure anticoagulation addressed (CHA₂DS₂-VASc score)",
			},
			GuidelineReferences: []string{
				"2020 ESC Guidelines for AFib",
				"EAST-AFNET 4 Trial (Early Rhythm Control)",
			},
			Enabled:   true,
			Priority:  5,
			CreatedAt: now,
			UpdatedAt: now,
		},

		// 9. Warfarin Drug Interaction
		{
			ID:          "warfarin-drug-interaction",
			Name:        "Warfarin Drug Interaction Risk",
			Description: "Alert for warfarin use with potential drug interactions",
			Version:     "1.0",
			Domain:      models.DomainCardiac,
			Severity:    models.SeverityHigh,
			Category:    "drug-interaction",
			Conditions: []RuleCondition{
				{
					Type:             ConditionTypeCompound,
					CompoundOperator: OpAnd,
					SubConditions: []RuleCondition{
						{Type: ConditionTypeValueSet, ValueSetID: "WarfarinTherapy"},
						// Common interacting medications
						{
							Type:             ConditionTypeCompound,
							CompoundOperator: OpOr,
							SubConditions: []RuleCondition{
								{Type: ConditionTypeValueSet, ValueSetID: "Aminoglycosides"},
								{Type: ConditionTypeValueSet, ValueSetID: "BroadSpectrumAntibiotics"},
								{Type: ConditionTypeValueSet, ValueSetID: "NSAIDs"},
							},
						},
					},
				},
			},
			AlertTitle:       "WARNING: Warfarin Drug Interaction Risk",
			AlertDescription: "Patient on warfarin with medication that may interact - bleeding or clotting risk",
			Recommendations: []string{
				"Check INR within 3-5 days of starting/stopping interacting medication",
				"NSAIDs: increase bleeding risk even with therapeutic INR - avoid if possible",
				"Antibiotics: many increase warfarin effect (especially metronidazole, ciprofloxacin, macrolides)",
				"Consider DOAC as alternative if frequent drug interactions",
				"Counsel patient to report new medications immediately",
				"Provide warfarin interaction wallet card",
			},
			GuidelineReferences: []string{
				"CHEST Guidelines for Antithrombotic Therapy",
				"Warfarin Drug Interactions (Micromedex)",
			},
			Enabled:   true,
			Priority:  3,
			CreatedAt: now,
			UpdatedAt: now,
		},

		// 10. Anticoagulation + Bleeding Risk
		{
			ID:          "anticoagulation-bleeding-risk",
			Name:        "Anticoagulation with Active Bleeding",
			Description: "CRITICAL: Patient on anticoagulation with active bleeding",
			Version:     "1.0",
			Domain:      models.DomainCardiac,
			Severity:    models.SeverityCritical,
			Category:    "bleeding-risk",
			Conditions: []RuleCondition{
				{
					Type:             ConditionTypeCompound,
					CompoundOperator: OpAnd,
					SubConditions: []RuleCondition{
						{Type: ConditionTypeValueSet, ValueSetID: "Anticoagulants"},
						{Type: ConditionTypeValueSet, ValueSetID: "ActiveBleeding"},
					},
				},
			},
			AlertTitle:       "🚨 CRITICAL: Anticoagulation with Active Bleeding",
			AlertDescription: "Patient on anticoagulation has active bleeding - immediate action required",
			Recommendations: []string{
				"HOLD anticoagulant immediately",
				"Assess bleeding severity and site",
				"If major bleeding: activate massive transfusion protocol if needed",
				"If warfarin: Give vitamin K 10mg IV + 4-factor PCC (Kcentra)",
				"If DOAC: Consider reversal agent (idarucizumab for dabigatran, andexanet alfa for Xa inhibitors)",
				"Check hemoglobin, platelet count, coagulation studies",
				"Consider transfusion if Hb <7 g/dL (or <8 g/dL if cardiac disease)",
				"Consult hematology for reversal guidance",
			},
			GuidelineReferences: []string{
				"2018 AHA/ASA Stroke Guidelines",
				"Anticoagulation Reversal Guidelines (CHEST 2022)",
			},
			Enabled:   true,
			Priority:  0, // Highest priority
			CreatedAt: now,
			UpdatedAt: now,
		},

		// 11. Dual Antiplatelet + Bleeding Risk
		{
			ID:          "dual-antiplatelet-bleeding-risk",
			Name:        "Dual Antiplatelet Therapy Duration",
			Description: "Alert for prolonged dual antiplatelet therapy - bleeding risk assessment",
			Version:     "1.0",
			Domain:      models.DomainCardiac,
			Severity:    models.SeverityModerate,
			Category:    "bleeding-risk",
			Conditions: []RuleCondition{
				{Type: ConditionTypeValueSet, ValueSetID: "IschemicStroke"},
			},
			AlertTitle:       "Dual Antiplatelet Therapy Assessment",
			AlertDescription: "Patient may be on DAPT - assess duration and bleeding risk",
			Recommendations: []string{
				"After PCI with stent: DAPT duration depends on stent type and indication",
				"BMS: minimum 1 month DAPT",
				"DES for stable CAD: minimum 6 months DAPT",
				"DES for ACS: minimum 12 months DAPT",
				"If high bleeding risk (HAS-BLED ≥3): consider shortening DAPT to 3 months",
				"After 12 months: transition to single antiplatelet (usually aspirin or clopidogrel)",
				"Assess for gastroprotection (PPI) if DAPT continues",
			},
			GuidelineReferences: []string{
				"2021 ACC/AHA/SCAI PCI Guidelines",
				"2020 ESC Guidelines for ACS",
			},
			Enabled:   true,
			Priority:  5,
			CreatedAt: now,
			UpdatedAt: now,
		},

		// ========================================================================
		// SEPSIS RULES (2)
		// ========================================================================

		// 12. Sepsis Lactate Elevation (Enhanced)
		{
			ID:          "sepsis-lactate-elevation",
			Name:        "Sepsis with Severe Lactate Elevation",
			Description: "Alert for sepsis with lactate >4.0 mmol/L - septic shock likely",
			Version:     "1.0",
			Domain:      models.DomainSepsis,
			Severity:    models.SeverityCritical,
			Category:    "diagnosis",
			Conditions: []RuleCondition{
				{
					Type:             ConditionTypeCompound,
					CompoundOperator: OpAnd,
					SubConditions: []RuleCondition{
						{Type: ConditionTypeValueSet, ValueSetID: "SepsisDiagnosis"},
						{Type: ConditionTypeThreshold, LoincCode: LOINCLactate, Operator: OpGreaterThan, Value: 4.0, Unit: "mmol/L"},
					},
				},
			},
			AlertTitle:       "🚨 CRITICAL: Sepsis with Severe Lactate Elevation",
			AlertDescription: "Lactate >4.0 mmol/L with sepsis - high mortality risk",
			Recommendations: []string{
				"ACTIVATE SEPTIC SHOCK PROTOCOL - CODE SEPSIS",
				"Complete Sepsis Hour-1 Bundle IMMEDIATELY:",
				"  1. Measure lactate (DONE - recheck in 2-4h)",
				"  2. Obtain blood cultures BEFORE antibiotics",
				"  3. Administer broad-spectrum antibiotics within 1 hour",
				"  4. Begin 30 mL/kg IV crystalloid bolus for hypotension/lactate ≥4",
				"  5. Apply vasopressors if hypotensive during/after fluids (MAP ≥65)",
				"ICU admission for hemodynamic monitoring",
				"Consider arterial line + central access",
				"Mortality risk ~30-40% with lactate >4 mmol/L",
			},
			GuidelineReferences: []string{
				"Surviving Sepsis Campaign 2021",
				"Sepsis-3 Definitions",
			},
			Enabled:   true,
			Priority:  0,
			CreatedAt: now,
			UpdatedAt: now,
		},

		// 13. Sepsis Antibiotic Timing
		{
			ID:          "sepsis-antibiotic-timing",
			Name:        "Sepsis - Antibiotic Administration Urgency",
			Description: "Alert for sepsis requiring urgent antibiotic administration",
			Version:     "1.0",
			Domain:      models.DomainSepsis,
			Severity:    models.SeverityCritical,
			Category:    "treatment",
			Conditions: []RuleCondition{
				{Type: ConditionTypeValueSet, ValueSetID: "SepsisDiagnosis"},
			},
			AlertTitle:       "URGENT: Sepsis - Antibiotics Required Within 1 Hour",
			AlertDescription: "Sepsis detected - empiric antibiotics should be administered within 1 hour",
			Recommendations: []string{
				"Obtain blood cultures BEFORE antibiotics (do not delay antibiotics >45 min)",
				"Administer empiric broad-spectrum antibiotics within 1 hour of recognition",
				"Suggested regimens:",
				"  - Community-acquired: Ceftriaxone 2g IV + Azithromycin 500mg IV",
				"  - Hospital-acquired: Pip-Tazo 4.5g IV or Meropenem 1g IV",
				"  - Neutropenic: Cefepime 2g IV + Vancomycin 15-20mg/kg IV",
				"Adjust based on suspected source and local antibiogram",
				"De-escalate based on culture results at 48-72 hours",
			},
			GuidelineReferences: []string{
				"Surviving Sepsis Campaign 2021",
				"Hour-1 Bundle (NEJM 2017)",
			},
			Enabled:   true,
			Priority:  1,
			CreatedAt: now,
			UpdatedAt: now,
		},

		// ========================================================================
		// METABOLIC RULES (2)
		// ========================================================================

		// 14. Hypoglycemia Risk in Elderly
		{
			ID:          "hypoglycemia-risk-elderly",
			Name:        "Hypoglycemia Risk in Elderly Diabetic",
			Description: "Alert for elderly diabetic patient - increased hypoglycemia risk",
			Version:     "1.0",
			Domain:      models.DomainMetabolic,
			Severity:    models.SeverityModerate,
			Category:    "medication-safety",
			Conditions: []RuleCondition{
				{
					Type:             ConditionTypeCompound,
					CompoundOperator: OpAnd,
					SubConditions: []RuleCondition{
						{Type: ConditionTypeValueSet, ValueSetID: "DiabetesMellitus"},
						// Note: Age would need to be added as a fact type or condition
						{Type: ConditionTypeThreshold, LoincCode: LOINCGlucose, Operator: OpLessThan, Value: 100.0, Unit: "mg/dL"},
					},
				},
			},
			AlertTitle:       "Hypoglycemia Risk Assessment Required",
			AlertDescription: "Diabetic patient with glucose trends toward hypoglycemia range",
			Recommendations: []string{
				"Assess for hypoglycemia risk factors: age >75, CKD, cognitive impairment, living alone",
				"Review diabetic medications - consider de-intensification if frequent lows",
				"If on sulfonylurea (glyburide, glipizide): consider switching to DPP-4 or GLP-1 RA",
				"If on intensive insulin: consider relaxing HbA1c target to <8% if high-risk",
				"Provide hypoglycemia action plan and glucagon prescription",
				"Consider continuous glucose monitor (CGM) if recurrent hypoglycemia",
			},
			GuidelineReferences: []string{
				"ADA Standards of Care 2024",
				"Endocrine Society Hypoglycemia Guidelines",
			},
			Enabled:   true,
			Priority:  4,
			CreatedAt: now,
			UpdatedAt: now,
		},

		// 15. DKA Risk with SGLT2 Inhibitor
		{
			ID:          "dka-risk-sglt2",
			Name:        "Euglycemic DKA Risk with SGLT2 Inhibitor",
			Description: "Alert for SGLT2 inhibitor use - euglycemic DKA risk education",
			Version:     "1.0",
			Domain:      models.DomainMetabolic,
			Severity:    models.SeverityModerate,
			Category:    "medication-safety",
			Conditions: []RuleCondition{
				{Type: ConditionTypeValueSet, ValueSetID: "DiabetesMellitus"},
			},
			AlertTitle:       "SGLT2 Inhibitor: DKA Risk Counseling Required",
			AlertDescription: "Patient on/candidate for SGLT2 inhibitor - educate on euglycemic DKA risk",
			Recommendations: []string{
				"SGLT2 inhibitors (empagliflozin, dapagliflozin, canagliflozin) carry small DKA risk",
				"Euglycemic DKA: ketoacidosis with glucose <250 mg/dL (can be missed)",
				"Risk factors: insulin deficiency, reduced food intake, illness, surgery, alcohol",
				"Counsel patient to HOLD SGLT2i during:",
				"  - Prolonged fasting or illness with reduced PO intake",
				"  - Surgery (hold 3 days before)",
				"  - Severe illness or DKA symptoms",
				"DKA symptoms: nausea, vomiting, abdominal pain, dyspnea, confusion",
				"If DKA suspected: check BMP, venous pH, beta-hydroxybutyrate",
			},
			GuidelineReferences: []string{
				"FDA SGLT2 Inhibitor Safety Communication",
				"ADA Standards of Care 2024",
			},
			Enabled:   true,
			Priority:  5,
			CreatedAt: now,
			UpdatedAt: now,
		},

		// ========================================================================
		// HEART FAILURE RULES (3)
		// ========================================================================

		// 16. Heart Failure - ACE-I/ARB Therapy
		{
			ID:          "hf-ace-arb-therapy",
			Name:        "Heart Failure - ACE-I/ARB/ARNI Recommended",
			Description: "Alert for heart failure patient not on ACE-I/ARB/ARNI - GDMT optimization",
			Version:     "1.0",
			Domain:      models.DomainCardiac,
			Severity:    models.SeverityHigh,
			Category:    "guideline-directed-therapy",
			Conditions: []RuleCondition{
				{
					Type:             ConditionTypeCompound,
					CompoundOperator: OpAnd,
					SubConditions: []RuleCondition{
						{Type: ConditionTypeValueSet, ValueSetID: "HeartFailure"},
						// NOT on ACE-I/ARB (absence of medication)
						{
							Type:             ConditionTypeCompound,
							CompoundOperator: OpNot,
							SubConditions: []RuleCondition{
								{Type: ConditionTypeValueSet, ValueSetID: "ACEInhibitorsARBs"},
							},
						},
					},
				},
			},
			AlertTitle:       "HIGH: Heart Failure - GDMT Optimization Needed",
			AlertDescription: "HFrEF patient not on ACE-I/ARB/ARNI - guideline-directed therapy indicated",
			Recommendations: []string{
				"HFrEF (EF ≤40%): ACE-I/ARB/ARNI is Class I recommendation (mortality benefit)",
				"Preferred: ARNI (sacubitril/valsartan) - superior to ACE-I/ARB",
				"If ARNI not tolerated/available: ACE-I (lisinopril, enalapril) or ARB (losartan, valsartan)",
				"Start low, titrate to target doses:",
				"  - Sacubitril/valsartan: target 97/103mg BID",
				"  - Enalapril: target 10-20mg BID",
				"  - Lisinopril: target 20-40mg daily",
				"Monitor K+ and creatinine 1-2 weeks after initiation/titration",
				"If hyperkalemia or worsening renal function: consider dose adjustment vs. continuation",
			},
			GuidelineReferences: []string{
				"2022 AHA/ACC/HFSA Heart Failure Guidelines",
				"PARADIGM-HF Trial (ARNI vs. ACE-I)",
			},
			Enabled:   true,
			Priority:  3,
			CreatedAt: now,
			UpdatedAt: now,
		},

		// 17. Heart Failure - Beta-Blocker Therapy
		{
			ID:          "hf-beta-blocker-therapy",
			Name:        "Heart Failure - Beta-Blocker Recommended",
			Description: "Alert for HFrEF patient not on evidence-based beta-blocker",
			Version:     "1.0",
			Domain:      models.DomainCardiac,
			Severity:    models.SeverityHigh,
			Category:    "guideline-directed-therapy",
			Conditions: []RuleCondition{
				{Type: ConditionTypeValueSet, ValueSetID: "HeartFailure"},
			},
			AlertTitle:       "HIGH: HFrEF - Beta-Blocker Therapy Recommended",
			AlertDescription: "Heart failure patient may need evidence-based beta-blocker for mortality benefit",
			Recommendations: []string{
				"HFrEF: Beta-blocker is Class I recommendation (reduces mortality by ~35%)",
				"Only 3 beta-blockers proven beneficial in HF:",
				"  1. Carvedilol (target: 25mg BID if <85kg, 50mg BID if ≥85kg)",
				"  2. Metoprolol succinate (target: 200mg daily)",
				"  3. Bisoprolol (target: 10mg daily)",
				"Start low dose after patient euvolemic (not during decompensation)",
				"Titrate slowly every 2 weeks as tolerated",
				"Monitor heart rate (target: 50-70 bpm) and blood pressure",
				"Do not discontinue for asymptomatic bradycardia if HR >50 bpm",
			},
			GuidelineReferences: []string{
				"2022 AHA/ACC/HFSA Heart Failure Guidelines",
				"CIBIS-II, COPERNICUS, MERIT-HF Trials",
			},
			Enabled:   true,
			Priority:  3,
			CreatedAt: now,
			UpdatedAt: now,
		},

		// 18. Heart Failure - Fluid Overload
		{
			ID:          "hf-fluid-overload",
			Name:        "Heart Failure Decompensation - Fluid Overload",
			Description: "Alert for heart failure with signs of volume overload",
			Version:     "1.0",
			Domain:      models.DomainCardiac,
			Severity:    models.SeverityCritical,
			Category:    "decompensation",
			Conditions: []RuleCondition{
				{
					Type:             ConditionTypeCompound,
					CompoundOperator: OpAnd,
					SubConditions: []RuleCondition{
						{Type: ConditionTypeValueSet, ValueSetID: "HeartFailure"},
						{Type: ConditionTypeThreshold, LoincCode: LOINCBNP, Operator: OpGreaterThan, Value: 1000.0, Unit: "pg/mL"},
					},
				},
			},
			AlertTitle:       "🚨 CRITICAL: Heart Failure Decompensation Likely",
			AlertDescription: "HF patient with severely elevated BNP (>1000 pg/mL) - volume overload likely",
			Recommendations: []string{
				"Assess for volume overload: JVD, pulmonary rales, peripheral edema, weight gain",
				"Obtain chest X-ray to assess for pulmonary edema",
				"If volume overload confirmed:",
				"  - Increase loop diuretic dose (IV if severe)",
				"  - Furosemide: start 40mg IV, can uptitrate to 80-160mg IV",
				 "  - Monitor strict I/Os, daily weights (target: net negative 1-2L/day)",
				"  - Fluid restrict to <2L/day",
				"  - Sodium restrict to <2g/day",
				"Consider ICU if respiratory distress or refractory to diuretics",
				"Recheck BNP/NT-proBNP after diuresis to assess response",
			},
			GuidelineReferences: []string{
				"2022 AHA/ACC/HFSA Heart Failure Guidelines",
				"DOSE Trial (Diuretic Strategies)",
			},
			Enabled:   true,
			Priority:  1,
			CreatedAt: now,
			UpdatedAt: now,
		},

		// ============================================================================
		// INR Supratherapeutic - Anticoagulant + Critically High INR
		// ============================================================================
		{
			ID:          "inr-supratherapeutic",
			Name:        "Supratherapeutic INR with Anticoagulation",
			Description: "Alert for critically elevated INR in patients on anticoagulation therapy",
			Version:     "1.0",
			Domain:      models.DomainHematologic,
			Severity:    models.SeverityCritical,
			Category:    "bleeding-risk",
			Conditions: []RuleCondition{
				{
					Type:             ConditionTypeCompound,
					CompoundOperator: OpAnd,
					SubConditions: []RuleCondition{
						{Type: ConditionTypeValueSet, ValueSetID: "Anticoagulants"},
						{Type: ConditionTypeThreshold, LoincCode: LOINCINR, Operator: OpGreaterThan, Value: 4.0, Unit: ""},
					},
				},
			},
			AlertTitle:       "🚨 CRITICAL: Supratherapeutic INR - High Bleeding Risk",
			AlertDescription: "Patient on anticoagulation with INR > 4.0 - significant bleeding risk",
			Recommendations: []string{
				"IMMEDIATE ACTIONS:",
				"  - Hold anticoagulant therapy",
				"  - Assess for active bleeding (GI, urinary, intracranial symptoms)",
				"  - Check for drug interactions (antibiotics, antifungals, NSAIDs)",
				"REVERSAL OPTIONS (based on bleeding status):",
				"  - No bleeding: Hold warfarin, recheck INR in 24-48 hours",
				"  - INR 4.5-10, no bleeding: Consider Vitamin K 1-2.5mg PO",
				"  - INR >10, no bleeding: Vitamin K 2.5-5mg PO",
				"  - Active bleeding: Vitamin K 10mg IV + 4-factor PCC (25-50 units/kg)",
				"  - Life-threatening bleeding: 4-factor PCC + Vitamin K 10mg IV STAT",
				"Monitor for signs of bleeding: bruising, hematuria, melena, hematemesis",
				"Investigate cause: dietary changes, new medications, illness",
			},
			GuidelineReferences: []string{
				"CHEST Guidelines 2021 - Management of Anticoagulation",
				"ACC Expert Consensus on Anticoagulation Reversal",
			},
			Enabled:   true,
			Priority:  1,
			CreatedAt: now,
			UpdatedAt: now,
		},

		// ============================================================================
		// Severe Hyperglycemia - Diabetes + Critically High Glucose
		// ============================================================================
		{
			ID:          "severe-hyperglycemia",
			Name:        "Severe Hyperglycemia in Diabetic Patient",
			Description: "Alert for critically elevated blood glucose suggesting diabetic emergency",
			Version:     "1.0",
			Domain:      models.DomainMetabolic,
			Severity:    models.SeverityCritical,
			Category:    "diabetic-emergency",
			Conditions: []RuleCondition{
				{
					Type:             ConditionTypeCompound,
					CompoundOperator: OpAnd,
					SubConditions: []RuleCondition{
						{Type: ConditionTypeValueSet, ValueSetID: "DiabetesMellitus"},
						{Type: ConditionTypeThreshold, LoincCode: LOINCGlucose, Operator: OpGreaterThan, Value: 400.0, Unit: "mg/dL"},
					},
				},
			},
			AlertTitle:       "🚨 CRITICAL: Severe Hyperglycemia - Possible DKA/HHS",
			AlertDescription: "Diabetic patient with glucose > 400 mg/dL - evaluate for diabetic ketoacidosis or hyperosmolar state",
			Recommendations: []string{
				"IMMEDIATE EVALUATION:",
				"  - Check basic metabolic panel (BMP) for anion gap, bicarbonate, potassium",
				"  - Check serum ketones (beta-hydroxybutyrate)",
				"  - Check serum osmolality",
				"  - Assess mental status and hydration",
				"IF DKA SUSPECTED (glucose >250, pH <7.3, bicarb <18, ketones+):",
				"  - IV fluid resuscitation: NS 1-1.5L/hr initially",
				"  - Insulin drip: 0.1 units/kg/hr after initial fluid bolus",
				"  - Potassium replacement when K <5.2 mEq/L",
				"  - Monitor glucose hourly, BMP q2-4h",
				"IF HHS SUSPECTED (glucose >600, osmolality >320, no ketosis):",
				"  - Aggressive IV fluid resuscitation (may need 6-10L deficit)",
				"  - Insulin drip at lower rate (0.05-0.1 units/kg/hr)",
				"  - Correct sodium based on corrected sodium formula",
				"Identify precipitating cause: infection, MI, medication non-compliance",
			},
			GuidelineReferences: []string{
				"ADA Standards of Care 2024 - Hyperglycemic Crises",
				"Joint British Diabetes Societies - DKA Guidelines",
			},
			Enabled:   true,
			Priority:  1,
			CreatedAt: now,
			UpdatedAt: now,
		},

		// ============================================================================
		// AKI Risk - Multiple Nephrotoxic Medications (uses COUNT condition)
		// ============================================================================
		{
			ID:          "aki-multiple-nephrotoxins",
			Name:        "AKI Risk - Multiple Nephrotoxic Medications",
			Description: "Alert for patients receiving 2 or more nephrotoxic medications simultaneously",
			Version:     "1.0",
			Domain:      models.DomainRenal,
			Severity:    models.SeverityHigh,
			Category:    "nephrotoxicity",
			Conditions: []RuleCondition{
				{
					Type:           ConditionTypeCount,
					ValueSetID:     "NephrotoxicMedications",
					CountOperator:  OpGreaterThanOrEqual,
					CountThreshold: 2,
				},
			},
			AlertTitle:       "⚠️ HIGH RISK: Multiple Nephrotoxic Medications",
			AlertDescription: "Patient receiving ≥2 nephrotoxic medications - elevated risk for acute kidney injury",
			Recommendations: []string{
				"NEPHROTOXIN BURDEN ASSESSMENT:",
				"  - Review all nephrotoxic medications: aminoglycosides, NSAIDs, contrast, ACEi/ARBs, diuretics",
				"  - Evaluate necessity of each nephrotoxic agent",
				"  - Consider alternatives with less nephrotoxic potential",
				"RENAL PROTECTION MEASURES:",
				"  - Ensure adequate hydration (IV fluids if needed)",
				"  - Avoid additional nephrotoxins (IV contrast, new NSAIDs)",
				"  - Hold ACEi/ARB if volume depleted or acute illness",
				"MONITORING:",
				"  - Check baseline creatinine and eGFR",
				"  - Repeat renal function in 24-48 hours",
				"  - If aminoglycosides: monitor drug levels, limit duration to <5 days",
				"  - Daily creatinine while on multiple nephrotoxins",
				"HIGH-RISK COMBINATIONS to avoid:",
				"  - Triple whammy: ACEi/ARB + Diuretic + NSAID",
				"  - Aminoglycoside + Loop diuretic",
				"  - NSAID + ACEi/ARB in dehydrated patient",
			},
			GuidelineReferences: []string{
				"KDIGO AKI Guidelines 2012",
				"Nephrotoxin Stewardship Programs - Best Practices",
			},
			Enabled:   true,
			Priority:  2,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
}

// ============================================================================
// LOINC Code Constants for Common Labs
// ============================================================================

// Common LOINC codes used in clinical rules
const (
	LOINCLactate     = "2524-7"  // Lactate [Moles/volume] in Blood
	LOINCCreatinine  = "2160-0"  // Creatinine [Mass/volume] in Serum or Plasma
	LOINCGlucose     = "2339-0"  // Glucose [Mass/volume] in Blood
	LOINCBNP         = "30934-4" // Natriuretic peptide B [Mass/volume] in Serum or Plasma
	LOINCSpO2        = "2708-6"  // Oxygen saturation in Arterial blood by Pulse oximetry
	LOINCHemoglobin  = "718-7"   // Hemoglobin [Mass/volume] in Blood
	LOINCPotassium   = "2823-3"  // Potassium [Moles/volume] in Serum or Plasma
	LOINCSodium      = "2951-2"  // Sodium [Moles/volume] in Serum or Plasma
	LOINCWBC         = "6690-2"  // Leukocytes [#/volume] in Blood
	LOINCPlatelets   = "777-3"   // Platelets [#/volume] in Blood
	LOINCPT          = "5902-2"  // Prothrombin time (PT)
	LOINCINR         = "6301-6"  // INR in Platelet poor plasma
	LOINCTroponinI   = "10839-9" // Troponin I.cardiac [Mass/volume] in Serum or Plasma
	LOINCProCalcitonin = "33959-8" // Procalcitonin [Mass/volume] in Serum or Plasma
)

// GetLOINCDisplayName returns the display name for a LOINC code
func GetLOINCDisplayName(code string) string {
	names := map[string]string{
		LOINCLactate:       "Lactate",
		LOINCCreatinine:    "Creatinine",
		LOINCGlucose:       "Glucose",
		LOINCBNP:           "BNP",
		LOINCSpO2:          "SpO2",
		LOINCHemoglobin:    "Hemoglobin",
		LOINCPotassium:     "Potassium",
		LOINCSodium:        "Sodium",
		LOINCWBC:           "WBC",
		LOINCPlatelets:     "Platelets",
		LOINCPT:            "PT",
		LOINCINR:           "INR",
		LOINCTroponinI:     "Troponin I",
		LOINCProCalcitonin: "Procalcitonin",
	}
	if name, ok := names[code]; ok {
		return name
	}
	return code
}
