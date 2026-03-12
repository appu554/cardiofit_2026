// Package engine implements the core governance evaluation logic.
// The governance engine is deterministic and reproducible - the same input
// always produces the same output. Every decision is fully explainable with
// complete evidence trails for medico-legal compliance.
package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"kb-18-governance-engine/pkg/programs"
	"kb-18-governance-engine/pkg/types"
)

// =============================================================================
// GOVERNANCE ENGINE
// =============================================================================

// GovernanceEngine evaluates clinical governance rules
type GovernanceEngine struct {
	programStore *programs.ProgramStore
	logger       *logrus.Entry
	stats        *EngineStats
	statsMu      sync.RWMutex

	// Configuration
	version          string
	decisionValidityMinutes int
}

// EngineStats tracks engine statistics
type EngineStats struct {
	TotalEvaluations    int64
	TotalViolations     int64
	TotalBlocked        int64
	TotalAllowed        int64
	ProgramsEvaluated   int64
	RulesEvaluated      int64
	ByProgram           map[string]int64
	BySeverity          map[string]int64
	ByCategory          map[string]int64
	AvgEvaluationTime   time.Duration
	LastEvaluationTime  time.Time
	totalEvaluationTime time.Duration // internal: for calculating average
	Since               time.Time
}

// NewGovernanceEngine creates a new governance engine
// Logger is optional - if nil, a default logger will be created
func NewGovernanceEngine(programStore *programs.ProgramStore, optionalLogger ...*logrus.Entry) *GovernanceEngine {
	var logger *logrus.Entry
	if len(optionalLogger) > 0 && optionalLogger[0] != nil {
		logger = optionalLogger[0].WithField("component", "governance-engine")
	} else {
		logger = logrus.WithField("component", "governance-engine")
	}

	return &GovernanceEngine{
		programStore: programStore,
		logger:       logger,
		version:      "1.0.0",
		decisionValidityMinutes: 60,
		stats: &EngineStats{
			ByProgram:  make(map[string]int64),
			BySeverity: make(map[string]int64),
			ByCategory: make(map[string]int64),
			Since:      time.Now(),
		},
	}
}

// =============================================================================
// MAIN EVALUATION ENTRY POINT
// =============================================================================

// Evaluate performs governance evaluation for a request
func (e *GovernanceEngine) Evaluate(ctx context.Context, req *types.EvaluationRequest) (*types.EvaluationResponse, error) {
	startTime := time.Now()

	// Generate request ID if not provided
	if req.RequestID == "" {
		req.RequestID = types.NewUUID()
	}
	if req.Timestamp.IsZero() {
		req.Timestamp = time.Now()
	}

	e.logger.WithFields(logrus.Fields{
		"request_id":  req.RequestID,
		"patient_id":  req.PatientID,
		"eval_type":   req.EvaluationType,
		"requestor":   req.RequestorID,
	}).Info("Starting governance evaluation")

	// Match applicable programs
	matchedPrograms := e.matchPrograms(req)
	if len(matchedPrograms) == 0 {
		e.logger.Debug("No programs matched - evaluation approved by default")
		return e.buildApprovedResponse(req, nil, []string{}), nil
	}

	e.logger.WithField("programs", getProgramCodes(matchedPrograms)).Debug("Matched programs")

	// Evaluate rules from all matched programs
	allViolations := []types.Violation{}
	ruleResults := []types.RuleResult{}
	programCodes := []string{}

	for _, program := range matchedPrograms {
		programCodes = append(programCodes, program.Code)
		violations, results := e.evaluateProgramRules(ctx, program, req)
		allViolations = append(allViolations, violations...)
		ruleResults = append(ruleResults, results...)
	}

	// Build and return response
	response := e.buildResponse(req, allViolations, ruleResults, programCodes)

	// Calculate evaluation duration
	evalDuration := time.Since(startTime)

	// Update statistics
	e.updateStats(response, programCodes, evalDuration)

	e.logger.WithFields(logrus.Fields{
		"request_id":   req.RequestID,
		"outcome":      response.Outcome,
		"violations":   len(response.Violations),
		"severity":     response.HighestSeverity,
		"duration_ms":  evalDuration.Milliseconds(),
	}).Info("Governance evaluation complete")

	return response, nil
}

// =============================================================================
// PROGRAM MATCHING
// =============================================================================

// matchPrograms finds all programs that apply to the request
func (e *GovernanceEngine) matchPrograms(req *types.EvaluationRequest) []*programs.Program {
	allPrograms := e.programStore.GetActivePrograms()
	matched := []*programs.Program{}

	for _, program := range allPrograms {
		if e.programMatches(program, req) {
			matched = append(matched, program)
		}
	}

	return matched
}

// programMatches checks if a program's activation criteria are met
func (e *GovernanceEngine) programMatches(program *programs.Program, req *types.EvaluationRequest) bool {
	criteria := program.ActivationCriteria

	// Check registry membership
	if len(criteria.RequiresRegistry) > 0 {
		if !e.hasAnyRegistry(req.PatientContext, criteria.RequiresRegistry) {
			return false
		}
	}

	// Check diagnosis
	if len(criteria.RequiresDiagnosis) > 0 {
		if !e.hasAnyDiagnosis(req.PatientContext, criteria.RequiresDiagnosis) {
			return false
		}
	}

	// Check medication/drug class for the order
	if len(criteria.RequiresMedication) > 0 && req.Order != nil {
		if !containsString(criteria.RequiresMedication, req.Order.MedicationCode) {
			return false
		}
	}

	if len(criteria.RequiresDrugClass) > 0 && req.Order != nil {
		if !containsString(criteria.RequiresDrugClass, req.Order.DrugClass) {
			return false
		}
	}

	// Check demographics
	if criteria.Demographics != nil {
		if !e.demographicsMatch(req.PatientContext, criteria.Demographics) {
			return false
		}
	}

	return true
}

// hasAnyRegistry checks if patient is in any of the specified registries
func (e *GovernanceEngine) hasAnyRegistry(ctx *types.PatientContext, registries []string) bool {
	if ctx == nil {
		return false
	}
	for _, membership := range ctx.RegistryMemberships {
		if membership.Status == "ACTIVE" && containsString(registries, membership.RegistryCode) {
			return true
		}
	}
	return false
}

// hasAnyDiagnosis checks if patient has any of the specified diagnoses
func (e *GovernanceEngine) hasAnyDiagnosis(ctx *types.PatientContext, diagnoses []string) bool {
	if ctx == nil {
		return false
	}
	for _, dx := range ctx.ActiveDiagnoses {
		if containsString(diagnoses, dx.Code) {
			return true
		}
	}
	return false
}

// demographicsMatch checks if patient demographics match criteria
func (e *GovernanceEngine) demographicsMatch(ctx *types.PatientContext, demo *programs.DemographicCriteria) bool {
	if ctx == nil {
		return false
	}

	// Check age range
	if demo.MinAge != nil && ctx.Age < *demo.MinAge {
		return false
	}
	if demo.MaxAge != nil && ctx.Age > *demo.MaxAge {
		return false
	}

	// Check sex
	if demo.Sex != "" && ctx.Sex != demo.Sex {
		return false
	}

	// Check pregnancy
	if demo.IsPregnant != nil && ctx.IsPregnant != *demo.IsPregnant {
		return false
	}

	// Check lactating
	if demo.IsLactating != nil && ctx.IsLactating != *demo.IsLactating {
		return false
	}

	return true
}

// =============================================================================
// RULE EVALUATION
// =============================================================================

// evaluateProgramRules evaluates all rules in a program
func (e *GovernanceEngine) evaluateProgramRules(ctx context.Context, program *programs.Program, req *types.EvaluationRequest) ([]types.Violation, []types.RuleResult) {
	violations := []types.Violation{}
	results := []types.RuleResult{}

	// Sort rules by priority (higher first)
	rules := make([]programs.Rule, len(program.Rules))
	copy(rules, program.Rules)
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Priority > rules[j].Priority
	})

	for _, rule := range rules {
		if !rule.IsActive {
			continue
		}

		result := types.RuleResult{
			RuleID:       rule.ID,
			RuleName:     rule.Name,
			WasEvaluated: true,
		}

		triggered, conditionResults := e.evaluateRule(rule, req)
		result.WasTriggered = triggered
		result.OutputDecision = fmt.Sprintf("triggered=%v", triggered)

		if triggered {
			violation := e.createViolation(program, rule, conditionResults)
			violations = append(violations, violation)
		}

		results = append(results, result)
	}

	return violations, results
}

// evaluateRule evaluates a single rule's conditions
func (e *GovernanceEngine) evaluateRule(rule programs.Rule, req *types.EvaluationRequest) (bool, []types.ConditionResult) {
	conditionResults := []types.ConditionResult{}
	metCount := 0

	for _, condition := range rule.Conditions {
		result := e.evaluateCondition(condition, req)
		conditionResults = append(conditionResults, result)
		if result.WasMet {
			metCount++
		}
	}

	// Apply condition logic
	var triggered bool
	switch rule.ConditionLogic {
	case "OR":
		triggered = metCount > 0
	case "AND":
		fallthrough
	default:
		triggered = metCount == len(rule.Conditions)
	}

	return triggered, conditionResults
}

// evaluateCondition evaluates a single condition
func (e *GovernanceEngine) evaluateCondition(cond programs.Condition, req *types.EvaluationRequest) types.ConditionResult {
	result := types.ConditionResult{
		ConditionType: string(cond.Type),
		Expression:    fmt.Sprintf("%s %s %v", cond.Field, cond.Operator, cond.Value),
	}

	switch cond.Type {
	case programs.ConditionTypeDrugClass:
		result.WasMet, result.ActualValue = e.evalDrugClassCondition(cond, req)
	case programs.ConditionTypeMedication:
		result.WasMet, result.ActualValue = e.evalMedicationCondition(cond, req)
	case programs.ConditionTypePregnancy:
		result.WasMet, result.ActualValue = e.evalPregnancyCondition(cond, req)
	case programs.ConditionTypeDemographic:
		result.WasMet, result.ActualValue = e.evalDemographicCondition(cond, req)
	case programs.ConditionTypeRenal:
		result.WasMet, result.ActualValue = e.evalRenalCondition(cond, req)
	case programs.ConditionTypeHepatic:
		result.WasMet, result.ActualValue = e.evalHepaticCondition(cond, req)
	case programs.ConditionTypeLabValue:
		result.WasMet, result.ActualValue = e.evalLabCondition(cond, req)
	case programs.ConditionTypeVitalSign:
		result.WasMet, result.ActualValue = e.evalVitalCondition(cond, req)
	case programs.ConditionTypeDiagnosis:
		result.WasMet, result.ActualValue = e.evalDiagnosisCondition(cond, req)
	case programs.ConditionTypeRegistry:
		result.WasMet, result.ActualValue = e.evalRegistryCondition(cond, req)
	case programs.ConditionTypeDose:
		result.WasMet, result.ActualValue = e.evalDoseCondition(cond, req)
	default:
		e.logger.WithField("condition_type", cond.Type).Warn("Unknown condition type")
		result.WasMet = false
	}

	result.ExpectedValue = fmt.Sprintf("%v", cond.Value)
	return result
}

// =============================================================================
// CONDITION EVALUATORS
// =============================================================================

func (e *GovernanceEngine) evalDrugClassCondition(cond programs.Condition, req *types.EvaluationRequest) (bool, string) {
	if req.Order == nil {
		return false, "no_order"
	}

	drugClass := req.Order.DrugClass

	switch cond.Operator {
	case "EQUALS":
		expected, ok := cond.Value.(string)
		return ok && drugClass == expected, drugClass
	case "IN":
		if values, ok := cond.Value.([]string); ok {
			return containsString(values, drugClass), drugClass
		}
		if values, ok := cond.Value.([]interface{}); ok {
			return containsInterface(values, drugClass), drugClass
		}
	case "NOT_IN":
		if values, ok := cond.Value.([]string); ok {
			return !containsString(values, drugClass), drugClass
		}
	}

	return false, drugClass
}

func (e *GovernanceEngine) evalMedicationCondition(cond programs.Condition, req *types.EvaluationRequest) (bool, string) {
	if req.Order == nil {
		return false, "no_order"
	}

	// Check if patient is on medications with certain class
	if cond.Operator == "HAS_CLASS" {
		classToCheck, ok := cond.Value.(string)
		if !ok {
			return false, ""
		}
		if req.PatientContext != nil {
			for _, med := range req.PatientContext.CurrentMedications {
				if med.DrugClass == classToCheck {
					return true, med.Name
				}
			}
		}
		return false, "not_on_class"
	}

	// Determine which field to check based on condition field
	var valueToCheck string
	switch cond.Field {
	case "medication", "medicationName", "name":
		// Check medication name
		valueToCheck = req.Order.MedicationName
	default:
		// Default to medication code
		valueToCheck = req.Order.MedicationCode
	}

	switch cond.Operator {
	case "EQUALS":
		expected, ok := cond.Value.(string)
		return ok && valueToCheck == expected, valueToCheck
	case "IN":
		if values, ok := cond.Value.([]string); ok {
			return containsString(values, valueToCheck), valueToCheck
		}
	}

	return false, valueToCheck
}

func (e *GovernanceEngine) evalPregnancyCondition(cond programs.Condition, req *types.EvaluationRequest) (bool, string) {
	if req.PatientContext == nil {
		return false, "no_context"
	}

	isPregnant := req.PatientContext.IsPregnant
	actual := fmt.Sprintf("%v", isPregnant)

	switch cond.Operator {
	case "EQUALS":
		expected, ok := cond.Value.(bool)
		return ok && isPregnant == expected, actual
	}

	return false, actual
}

func (e *GovernanceEngine) evalDemographicCondition(cond programs.Condition, req *types.EvaluationRequest) (bool, string) {
	if req.PatientContext == nil {
		return false, "no_context"
	}

	ctx := req.PatientContext

	switch cond.Field {
	case "age":
		return e.evalNumericCondition(cond.Operator, float64(ctx.Age), cond.Value), fmt.Sprintf("%d", ctx.Age)
	case "sex":
		return ctx.Sex == cond.Value, ctx.Sex
	case "ageBand":
		return ctx.GetAgeBand() == cond.Value, ctx.GetAgeBand()
	}

	return false, ""
}

func (e *GovernanceEngine) evalRenalCondition(cond programs.Condition, req *types.EvaluationRequest) (bool, string) {
	if req.PatientContext == nil || req.PatientContext.RenalFunction == nil {
		return false, "no_renal_data"
	}

	renal := req.PatientContext.RenalFunction

	switch cond.Field {
	case "egfr":
		return e.evalNumericCondition(cond.Operator, renal.EGFR, cond.Value), fmt.Sprintf("%.1f", renal.EGFR)
	case "ckdStage":
		return renal.CKDStage == cond.Value, renal.CKDStage
	case "onDialysis":
		expected, ok := cond.Value.(bool)
		return ok && renal.OnDialysis == expected, fmt.Sprintf("%v", renal.OnDialysis)
	}

	return false, ""
}

func (e *GovernanceEngine) evalHepaticCondition(cond programs.Condition, req *types.EvaluationRequest) (bool, string) {
	if req.PatientContext == nil || req.PatientContext.HepaticFunction == nil {
		return false, "no_hepatic_data"
	}

	hepatic := req.PatientContext.HepaticFunction

	switch cond.Field {
	case "childPughClass":
		return hepatic.ChildPughClass == cond.Value, hepatic.ChildPughClass
	case "childPughScore":
		return e.evalNumericCondition(cond.Operator, float64(hepatic.ChildPughScore), cond.Value), fmt.Sprintf("%d", hepatic.ChildPughScore)
	}

	return false, ""
}

func (e *GovernanceEngine) evalLabCondition(cond programs.Condition, req *types.EvaluationRequest) (bool, string) {
	if req.PatientContext == nil {
		return false, "no_context"
	}

	// Handle MISSING operator
	if cond.Operator == "MISSING" {
		windowHours := cond.LabWindow
		if windowHours == 0 {
			windowHours = 24 // Default 24 hours
		}
		cutoff := time.Now().Add(-time.Duration(windowHours) * time.Hour)

		for _, lab := range req.PatientContext.RecentLabs {
			if lab.Code == cond.LabCode && lab.Timestamp.After(cutoff) {
				return false, fmt.Sprintf("%.2f", lab.Value)
			}
		}
		return true, "no_recent_lab"
	}

	// Find most recent lab matching code
	var latestLab *types.LabResult
	for i := range req.PatientContext.RecentLabs {
		lab := &req.PatientContext.RecentLabs[i]
		if lab.Code == cond.LabCode {
			if latestLab == nil || lab.Timestamp.After(latestLab.Timestamp) {
				latestLab = lab
			}
		}
	}

	if latestLab == nil {
		return false, "no_lab_found"
	}

	met := e.evalNumericCondition(cond.Operator, latestLab.Value, cond.Value)
	return met, fmt.Sprintf("%.2f", latestLab.Value)
}

func (e *GovernanceEngine) evalVitalCondition(cond programs.Condition, req *types.EvaluationRequest) (bool, string) {
	if req.PatientContext == nil || req.PatientContext.Vitals == nil {
		return false, "no_vitals"
	}

	vitals := req.PatientContext.Vitals

	var value float64
	switch cond.Field {
	case "systolicBp":
		value = float64(vitals.SystolicBP)
	case "diastolicBp":
		value = float64(vitals.DiastolicBP)
	case "heartRate":
		value = float64(vitals.HeartRate)
	case "temperature":
		value = vitals.Temperature
	case "spo2":
		value = float64(vitals.SpO2)
	default:
		return false, "unknown_vital"
	}

	met := e.evalNumericCondition(cond.Operator, value, cond.Value)
	return met, fmt.Sprintf("%.1f", value)
}

func (e *GovernanceEngine) evalDiagnosisCondition(cond programs.Condition, req *types.EvaluationRequest) (bool, string) {
	if req.PatientContext == nil {
		return false, "no_context"
	}

	var diagnoses []string
	for _, dx := range req.PatientContext.ActiveDiagnoses {
		diagnoses = append(diagnoses, dx.Code)
	}

	switch cond.Operator {
	case "IN":
		if values, ok := cond.Value.([]string); ok {
			for _, dx := range diagnoses {
				if containsString(values, dx) {
					return true, dx
				}
			}
		}
	case "HAS":
		expected, ok := cond.Value.(string)
		return ok && containsString(diagnoses, expected), strings.Join(diagnoses, ",")
	}

	return false, strings.Join(diagnoses, ",")
}

func (e *GovernanceEngine) evalRegistryCondition(cond programs.Condition, req *types.EvaluationRequest) (bool, string) {
	if req.PatientContext == nil {
		return false, "no_context"
	}

	var activeRegistries []string
	for _, reg := range req.PatientContext.RegistryMemberships {
		if reg.Status == "ACTIVE" {
			activeRegistries = append(activeRegistries, reg.RegistryCode)
		}
	}

	switch cond.Operator {
	case "IN":
		if values, ok := cond.Value.([]string); ok {
			for _, reg := range activeRegistries {
				if containsString(values, reg) {
					return true, reg
				}
			}
		}
		if values, ok := cond.Value.([]interface{}); ok {
			for _, reg := range activeRegistries {
				if containsInterface(values, reg) {
					return true, reg
				}
			}
		}
	}

	return false, strings.Join(activeRegistries, ",")
}

func (e *GovernanceEngine) evalDoseCondition(cond programs.Condition, req *types.EvaluationRequest) (bool, string) {
	if req.Order == nil {
		return false, "no_order"
	}

	// For MME calculations, we would need additional context
	// For now, use the order dose directly
	dose := req.Order.Dose

	met := e.evalNumericCondition(cond.Operator, dose, cond.Value)
	return met, fmt.Sprintf("%.2f", dose)
}

// evalNumericCondition evaluates numeric comparisons
func (e *GovernanceEngine) evalNumericCondition(operator string, actual float64, expected interface{}) bool {
	switch operator {
	case "GT":
		exp, ok := toFloat64(expected)
		return ok && actual > exp
	case "GTE":
		exp, ok := toFloat64(expected)
		return ok && actual >= exp
	case "LT":
		exp, ok := toFloat64(expected)
		return ok && actual < exp
	case "LTE":
		exp, ok := toFloat64(expected)
		return ok && actual <= exp
	case "EQUALS":
		exp, ok := toFloat64(expected)
		return ok && actual == exp
	case "BETWEEN":
		if arr, ok := expected.([]float64); ok && len(arr) == 2 {
			return actual >= arr[0] && actual <= arr[1]
		}
		if arr, ok := expected.([]interface{}); ok && len(arr) == 2 {
			low, ok1 := toFloat64(arr[0])
			high, ok2 := toFloat64(arr[1])
			return ok1 && ok2 && actual >= low && actual <= high
		}
	}
	return false
}

// =============================================================================
// VIOLATION & RESPONSE BUILDING
// =============================================================================

// createViolation creates a violation from a triggered rule
func (e *GovernanceEngine) createViolation(program *programs.Program, rule programs.Rule, conditions []types.ConditionResult) types.Violation {
	return types.Violation{
		ID:               types.NewViolationID(),
		RuleID:           rule.ID,
		RuleName:         rule.Name,
		ProgramCode:      program.Code,
		Category:         rule.Category,
		Severity:         rule.Severity,
		EnforcementLevel: rule.EnforcementLevel,
		Description:      rule.Description,
		ClinicalRisk:     rule.ClinicalRisk,
		EvidenceLevel:    rule.EvidenceLevel,
		References:       rule.References,
		CanOverride:      rule.EnforcementLevel.CanOverride(),
		RequiresAck:      rule.EnforcementLevel.RequiresAcknowledgment(),
		ConditionsMet:    conditions,
	}
}

// buildResponse constructs the evaluation response
func (e *GovernanceEngine) buildResponse(req *types.EvaluationRequest, violations []types.Violation, ruleResults []types.RuleResult, programCodes []string) *types.EvaluationResponse {
	response := &types.EvaluationResponse{
		RequestID:         req.RequestID,
		ProgramsEvaluated: programCodes,
		EvaluatedAt:       time.Now(),
		ExpiresAt:         time.Now().Add(time.Duration(e.decisionValidityMinutes) * time.Minute),
	}

	if len(violations) == 0 {
		response.Outcome = types.OutcomeApproved
		response.IsApproved = true
		response.HasViolations = false
	} else {
		response.HasViolations = true
		response.Violations = violations

		// Determine outcome based on highest enforcement level
		highestEnforcement := e.getHighestEnforcement(violations)
		highestSeverity := e.getHighestSeverity(violations)
		response.HighestSeverity = highestSeverity

		switch {
		case highestEnforcement == types.EnforcementHardBlock:
			response.Outcome = types.OutcomeBlocked
			response.IsApproved = false
		case highestEnforcement == types.EnforcementMandatoryEscalation:
			response.Outcome = types.OutcomeEscalated
			response.IsApproved = false
		case highestEnforcement == types.EnforcementHardBlockWithOverride:
			response.Outcome = types.OutcomePendingOverride
			response.IsApproved = false
		case highestEnforcement == types.EnforcementWarnAcknowledge:
			response.Outcome = types.OutcomePendingAck
			response.IsApproved = false
		default:
			response.Outcome = types.OutcomeApprovedWithWarns
			response.IsApproved = true
		}

		// Collect recommendations
		response.Recommendations = e.collectRecommendations(violations, programCodes)

		// Set next steps
		response.NextSteps = e.determineNextSteps(response.Outcome, violations)
	}

	// Set accountable parties
	response.AccountableParties = e.getAccountableParties(programCodes)

	// Build evidence trail
	response.EvidenceTrail = e.buildEvidenceTrail(req, response, ruleResults)

	return response
}

// buildApprovedResponse creates an approved response with no violations
func (e *GovernanceEngine) buildApprovedResponse(req *types.EvaluationRequest, ruleResults []types.RuleResult, programCodes []string) *types.EvaluationResponse {
	response := &types.EvaluationResponse{
		RequestID:         req.RequestID,
		Outcome:           types.OutcomeApproved,
		IsApproved:        true,
		HasViolations:     false,
		ProgramsEvaluated: programCodes,
		EvaluatedAt:       time.Now(),
		ExpiresAt:         time.Now().Add(time.Duration(e.decisionValidityMinutes) * time.Minute),
	}

	response.EvidenceTrail = e.buildEvidenceTrail(req, response, ruleResults)

	return response
}

// getHighestEnforcement returns the most severe enforcement level
func (e *GovernanceEngine) getHighestEnforcement(violations []types.Violation) types.EnforcementLevel {
	highest := types.EnforcementIgnore
	for _, v := range violations {
		if v.EnforcementLevel.Priority() > highest.Priority() {
			highest = v.EnforcementLevel
		}
	}
	return highest
}

// getHighestSeverity returns the most severe severity level
func (e *GovernanceEngine) getHighestSeverity(violations []types.Violation) types.Severity {
	highest := types.SeverityInfo
	for _, v := range violations {
		if v.Severity.Priority() > highest.Priority() {
			highest = v.Severity
		}
	}
	return highest
}

// collectRecommendations gathers all unique recommendations
func (e *GovernanceEngine) collectRecommendations(violations []types.Violation, programCodes []string) []types.Recommendation {
	seen := make(map[string]bool)
	recs := []types.Recommendation{}

	for _, v := range violations {
		// Get recommendations from rule
		for _, program := range e.programStore.GetActivePrograms() {
			if v.ProgramCode != program.Code {
				continue
			}
			for _, rule := range program.Rules {
				if rule.ID != v.RuleID {
					continue
				}
				for _, rec := range rule.Recommendations {
					key := rec.Type + ":" + rec.Title
					if !seen[key] {
						seen[key] = true
						recs = append(recs, rec)
					}
				}
			}
		}
	}

	return recs
}

// determineNextSteps determines what actions are needed
func (e *GovernanceEngine) determineNextSteps(outcome types.Outcome, violations []types.Violation) []string {
	steps := []string{}

	switch outcome {
	case types.OutcomeBlocked:
		steps = append(steps, "Order cannot proceed - modify order or select alternative")
	case types.OutcomeEscalated:
		steps = append(steps, "Immediate escalation required - notify supervisor")
	case types.OutcomePendingOverride:
		steps = append(steps, "Override required - submit override request with clinical justification")
	case types.OutcomePendingAck:
		steps = append(steps, "Acknowledgment required - review warnings and acknowledge to proceed")
	case types.OutcomeApprovedWithWarns:
		steps = append(steps, "Proceed with caution - warnings documented in evidence trail")
	}

	return steps
}

// getAccountableParties returns accountability chain for evaluated programs
func (e *GovernanceEngine) getAccountableParties(programCodes []string) []types.AccountableParty {
	seen := make(map[string]bool)
	parties := []types.AccountableParty{}
	order := 1

	for _, code := range programCodes {
		program, ok := e.programStore.GetProgram(code)
		if !ok {
			continue
		}
		for _, role := range program.AccountabilityChain {
			if !seen[role] {
				seen[role] = true
				parties = append(parties, types.AccountableParty{
					Role:            role,
					Accountability:  getAccountabilityDescription(role),
					EscalationOrder: order,
				})
				order++
			}
		}
	}

	return parties
}

// buildEvidenceTrail creates an immutable evidence trail
func (e *GovernanceEngine) buildEvidenceTrail(req *types.EvaluationRequest, response *types.EvaluationResponse, ruleResults []types.RuleResult) *types.EvidenceTrail {
	trail := &types.EvidenceTrail{
		TrailID:           types.NewTrailID(),
		Timestamp:         time.Now(),
		ProgramsEvaluated: response.ProgramsEvaluated,
		RulesApplied:      ruleResults,
		FinalDecision:     response.Outcome,
		DecisionRationale: fmt.Sprintf("%s due to %d violation(s). Highest severity: %s",
			response.Outcome, len(response.Violations), response.HighestSeverity),
		RequestedBy:       req.RequestorID,
		EvaluatedBy:       fmt.Sprintf("KB18-GOV-ENGINE-v%s", e.version),
		IsImmutable:       true,
	}

	// Capture patient snapshot
	if req.PatientContext != nil {
		snapshot, _ := json.Marshal(req.PatientContext)
		trail.PatientSnapshot = snapshot
	}

	// Capture order snapshot
	if req.Order != nil {
		snapshot, _ := json.Marshal(req.Order)
		trail.OrderSnapshot = snapshot
	}

	// Generate cryptographic hash
	trail.Hash = trail.GenerateHash()

	return trail
}

// =============================================================================
// STATISTICS
// =============================================================================

// updateStats updates engine statistics
func (e *GovernanceEngine) updateStats(response *types.EvaluationResponse, programCodes []string, evalDuration time.Duration) {
	e.statsMu.Lock()
	defer e.statsMu.Unlock()

	e.stats.TotalEvaluations++
	e.stats.TotalViolations += int64(len(response.Violations))
	e.stats.ProgramsEvaluated += int64(len(programCodes))
	e.stats.LastEvaluationTime = time.Now()
	e.stats.totalEvaluationTime += evalDuration

	if e.stats.TotalEvaluations > 0 {
		e.stats.AvgEvaluationTime = e.stats.totalEvaluationTime / time.Duration(e.stats.TotalEvaluations)
	}

	if response.Outcome == types.OutcomeBlocked || response.Outcome == types.OutcomeEscalated {
		e.stats.TotalBlocked++
	} else {
		e.stats.TotalAllowed++
	}

	for _, code := range programCodes {
		e.stats.ByProgram[code]++
	}

	for _, v := range response.Violations {
		e.stats.BySeverity[string(v.Severity)]++
		e.stats.ByCategory[string(v.Category)]++
	}
}

// GetStats returns current engine statistics
func (e *GovernanceEngine) GetStats() *types.EngineStats {
	e.statsMu.RLock()
	defer e.statsMu.RUnlock()

	return &types.EngineStats{
		TotalEvaluations:   e.stats.TotalEvaluations,
		TotalViolations:    e.stats.TotalViolations,
		TotalBlocked:       e.stats.TotalBlocked,
		TotalAllowed:       e.stats.TotalAllowed,
		ProgramsEvaluated:  e.stats.ProgramsEvaluated,
		RulesEvaluated:     e.stats.RulesEvaluated,
		ByProgram:          copyMap(e.stats.ByProgram),
		BySeverity:         copyMap(e.stats.BySeverity),
		ByCategory:         copyMap(e.stats.ByCategory),
		AvgEvaluationTime:  e.stats.AvgEvaluationTime,
		LastEvaluationTime: e.stats.LastEvaluationTime,
		Since:              e.stats.Since,
	}
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func containsInterface(slice []interface{}, s string) bool {
	for _, item := range slice {
		if str, ok := item.(string); ok && str == s {
			return true
		}
	}
	return false
}

func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case int32:
		return float64(val), true
	}
	return 0, false
}

func getProgramCodes(programs []*programs.Program) []string {
	codes := make([]string, len(programs))
	for i, p := range programs {
		codes[i] = p.Code
	}
	return codes
}

func copyMap(m map[string]int64) map[string]int64 {
	copy := make(map[string]int64)
	for k, v := range m {
		copy[k] = v
	}
	return copy
}

func getAccountabilityDescription(role string) string {
	descriptions := map[string]string{
		"PRESCRIBER":            "Order verification and clinical justification",
		"PHARMACIST":            "Dispensing safety check and drug review",
		"NURSE":                 "Administration verification and monitoring",
		"ATTENDING_PHYSICIAN":   "Clinical oversight and escalation review",
		"OB_ATTENDING":          "Obstetric clinical oversight",
		"MFM_SPECIALIST":        "Maternal-fetal medicine consultation",
		"PAIN_SPECIALIST":       "Pain management consultation",
		"ADDICTION_SPECIALIST":  "Addiction medicine consultation",
		"ANTICOAGULATION_CLINIC": "Anticoagulation therapy management",
		"HEMATOLOGIST":          "Hematology consultation",
		"NEPHROLOGIST":          "Nephrology consultation for renal dosing",
		"ENDOCRINOLOGIST":       "Endocrinology consultation",
		"DEPARTMENT_CHIEF":      "Department-level oversight and approval",
		"MEDICAL_DIRECTOR":      "Medical directorship approval",
		"CMO":                   "Chief Medical Officer final authority",
	}
	if desc, ok := descriptions[role]; ok {
		return desc
	}
	return "Clinical accountability"
}
