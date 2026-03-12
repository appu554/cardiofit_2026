package federation

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"kb-drug-rules/internal/cache"
	"kb-drug-rules/internal/services"
)

// GraphQLSchema defines the Apollo Federation schema for KB-Drug-Rules
const GraphQLSchema = `
	type DosingRule @key(fields: "drugCode version") {
		drugCode: String!
		version: String!
		drugName: String!
		contentSHA: String!
		signatureValid: Boolean!
		active: Boolean!
		regions: [String!]!
		baseDose: BaseDose!
		adjustments: [DoseAdjustment!]!
		titrationSchedule: [TitrationStep!]!
		populationRules: [PopulationRule!]!
		safetyVerification: SafetyVerification!
		createdAt: String!
		createdBy: String!
		provenance: ProvenanceInfo!
	}

	type BaseDose {
		unit: String!
		starting: Float!
		maxDaily: Float!
		minDaily: Float!
		frequency: String!
		loading: String
		maintenance: String
		maxSingle: Float
	}

	type DoseAdjustment {
		adjustmentId: String!
		adjustType: String!
		description: String!
		conditionExpression: String!
		formulaExpression: String!
		multiplier: Float
		additiveMg: Float
		maxDoseMg: Float
		minDoseMg: Float
		contraindicated: Boolean!
	}

	type TitrationStep {
		stepNumber: Int!
		afterDays: Int!
		actionType: String!
		actionValue: Float
		maxStep: Int
		monitoringRequired: String
	}

	type PopulationRule {
		populationId: String!
		populationType: String!
		ageMin: Int
		ageMax: Int
		weightMin: Float
		weightMax: Float
		formula: String!
		safetyLimits: String
		contraindicated: Boolean!
	}

	type SafetyVerification {
		contraindications: [String!]!
		warnings: [String!]!
		precautions: [String!]!
		labMonitoring: [String!]!
	}

	type ProvenanceInfo {
		authors: [String!]!
		approvals: [String!]!
		kb3Refs: [String!]!
		kb4Refs: [String!]!
		sourceFile: String!
		effectiveFrom: String
		lastModifiedBy: String
	}

	type DosingRuleConnection {
		edges: [DosingRuleEdge!]!
		pageInfo: PageInfo!
		totalCount: Int!
	}

	type DosingRuleEdge {
		node: DosingRule!
		cursor: String!
	}

	type PageInfo {
		hasNextPage: Boolean!
		hasPreviousPage: Boolean!
		startCursor: String
		endCursor: String
	}

	input PatientContextInput {
		weightKg: Float!
		egfr: Float!
		ageYears: Int!
		sex: String!
		pregnant: Boolean
		creatinineCleatance: Float
		dialysisType: String
		extraNumeric: [ExtraNumericInput!]
	}

	input ExtraNumericInput {
		key: String!
		value: Float!
	}

	type DosingRecommendation {
		drugCode: String!
		version: String!
		applicableAdjustments: [DoseAdjustment!]!
		recommendedDose: RecommendedDose!
		safetyAlerts: [SafetyAlert!]!
		monitoringRequirements: [String!]!
		calculationMetadata: CalculationMetadata!
	}

	type RecommendedDose {
		amountMg: Float!
		frequency: String!
		route: String!
		duration: String
		specialInstructions: [String!]!
	}

	type SafetyAlert {
		alertType: String!
		severity: String!
		message: String!
		actionRequired: String
	}

	type CalculationMetadata {
		basedOnRuleVersion: String!
		appliedAdjustments: [String!]!
		calculatedAt: String!
		cacheHit: Boolean!
		responseTimeMs: Float!
	}

	extend type Query {
		# Get dosing rules for a specific drug
		dosingRule(drugCode: String!, version: String, region: String): DosingRule
		
		# Get all dosing rules with pagination
		dosingRules(
			first: Int = 20, 
			after: String, 
			drugCodes: [String!], 
			regions: [String!],
			activeOnly: Boolean = true
		): DosingRuleConnection!
		
		# Calculate dosing recommendation for patient context
		calculateDosing(
			drugCode: String!, 
			patientContext: PatientContextInput!, 
			version: String,
			region: String = "US"
		): DosingRecommendation
		
		# Check if dosing rules are available for a drug
		checkDosingAvailability(drugCode: String!, region: String): Boolean!
		
		# Get rule metadata without full content
		dosingRuleMetadata(drugCode: String!, version: String): ProvenanceInfo
	}

	extend type Mutation {
		# Validate TOML dosing rule content
		validateDosingRule(
			tomlContent: String!, 
			drugCode: String!, 
			regions: [String!]
		): ValidationResult!
		
		# Submit dosing rule for clinical approval (governance workflow)
		submitDosingRuleForApproval(
			drugCode: String!,
			version: String!,
			tomlContent: String!,
			submittedBy: String!,
			clinicalJustification: String!
		): SubmissionResult!
	}

	type ValidationResult {
		valid: Boolean!
		errors: [String!]!
		warnings: [String!]!
		compiledJSON: String
		usedFields: [String!]!
		checksum: String
	}

	type SubmissionResult {
		success: Boolean!
		submissionId: String!
		status: String!
		estimatedReviewTime: String
		reviewers: [String!]!
		message: String
	}
`

// Resolver implements GraphQL resolvers for the KB-Drug-Rules federation schema
type Resolver struct {
	db      *gorm.DB
	cache   cache.KB1CacheInterface
	logger  *logrus.Logger
	service services.RuleService
}

// NewResolver creates a new GraphQL resolver for federation
func NewResolver(db *gorm.DB, cacheClient cache.KB1CacheInterface, logger *logrus.Logger) *Resolver {
	// Create a cache adapter for the rule service
	adapter := cache.NewKB1ClientAdapter(cacheClient)
	return &Resolver{
		db:      db,
		cache:   cacheClient,
		logger:  logger,
		service: services.NewRuleService(db, adapter, logger),
	}
}

// DosingRule resolver methods

func (r *Resolver) DosingRule(ctx context.Context, args struct {
	DrugCode string
	Version  *string
	Region   *string
}) (*DosingRuleType, error) {
	start := time.Now()
	defer func() {
		r.logger.WithFields(logrus.Fields{
			"resolver":    "dosingRule",
			"drug_code":   args.DrugCode,
			"duration_ms": time.Since(start).Milliseconds(),
		}).Debug("GraphQL dosingRule resolver")
	}()

	// Query from active_dosing_rules materialized view for performance
	var result struct {
		RuleID          string          `json:"rule_id"`
		DrugCode        string          `json:"drug_code"`
		DrugName        string          `json:"drug_name"`
		SemanticVersion string          `json:"semantic_version"`
		CompiledJSON    json.RawMessage `json:"compiled_json"`
		Checksum        string          `json:"checksum"`
		Adjustments     json.RawMessage `json:"adjustments"`
		TitrationSchedule json.RawMessage `json:"titration_schedule"`
		PopulationRules json.RawMessage `json:"population_rules"`
		Provenance      json.RawMessage `json:"provenance"`
		CreatedAt       time.Time       `json:"created_at"`
	}

	query := `
		SELECT rule_id, drug_code, drug_name, semantic_version, compiled_json, 
		       checksum, adjustments, titration_schedule, population_rules, 
		       provenance, created_at
		FROM active_dosing_rules 
		WHERE drug_code = ?
	`
	
	queryArgs := []interface{}{args.DrugCode}
	
	if args.Version != nil {
		query += " AND semantic_version = ?"
		queryArgs = append(queryArgs, *args.Version)
	}
	
	query += " ORDER BY semantic_version DESC LIMIT 1"

	if err := r.db.Raw(query, queryArgs...).Scan(&result).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // Not found
		}
		r.logger.WithError(err).Error("Failed to query dosing rule")
		return nil, fmt.Errorf("database query failed: %w", err)
	}

	return r.buildDosingRuleType(&result)
}

func (r *Resolver) DosingRules(ctx context.Context, args struct {
	First      *int32
	After      *string
	DrugCodes  *[]string
	Regions    *[]string
	ActiveOnly *bool
}) (*DosingRuleConnectionType, error) {
	start := time.Now()
	defer func() {
		r.logger.WithFields(logrus.Fields{
			"resolver":    "dosingRules",
			"duration_ms": time.Since(start).Milliseconds(),
		}).Debug("GraphQL dosingRules resolver")
	}()

	// Build query with filters
	query := "SELECT * FROM active_dosing_rules WHERE 1=1"
	var queryArgs []interface{}
	argIndex := 1

	if args.DrugCodes != nil && len(*args.DrugCodes) > 0 {
		placeholders := make([]string, len(*args.DrugCodes))
		for i, drugCode := range *args.DrugCodes {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			queryArgs = append(queryArgs, drugCode)
			argIndex++
		}
		query += fmt.Sprintf(" AND drug_code IN (%s)", strings.Join(placeholders, ","))
	}

	// Add pagination
	limit := int32(20)
	if args.First != nil {
		limit = *args.First
	}
	
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d", argIndex)
	queryArgs = append(queryArgs, limit)

	var results []struct {
		RuleID          string          `json:"rule_id"`
		DrugCode        string          `json:"drug_code"`
		DrugName        string          `json:"drug_name"`
		SemanticVersion string          `json:"semantic_version"`
		CompiledJSON    json.RawMessage `json:"compiled_json"`
		Checksum        string          `json:"checksum"`
		Adjustments     json.RawMessage `json:"adjustments"`
		TitrationSchedule json.RawMessage `json:"titration_schedule"`
		PopulationRules json.RawMessage `json:"population_rules"`
		Provenance      json.RawMessage `json:"provenance"`
		CreatedAt       time.Time       `json:"created_at"`
	}

	if err := r.db.Raw(query, queryArgs...).Scan(&results).Error; err != nil {
		r.logger.WithError(err).Error("Failed to query dosing rules")
		return nil, fmt.Errorf("database query failed: %w", err)
	}

	// Build connection response
	edges := make([]*DosingRuleEdgeType, len(results))
	for i, result := range results {
		dosingRule, err := r.buildDosingRuleType(&result)
		if err != nil {
			r.logger.WithError(err).Error("Failed to build dosing rule type")
			continue
		}
		
		edges[i] = &DosingRuleEdgeType{
			Node:   dosingRule,
			Cursor: fmt.Sprintf("cursor_%d", i),
		}
	}

	return &DosingRuleConnectionType{
		Edges: edges,
		PageInfo: &PageInfoType{
			HasNextPage:     len(results) == int(limit),
			HasPreviousPage: args.After != nil,
		},
		TotalCount: int32(len(results)),
	}, nil
}

func (r *Resolver) CalculateDosing(ctx context.Context, args struct {
	DrugCode       string
	PatientContext PatientContextInputType
	Version        *string
	Region         *string
}) (*DosingRecommendationType, error) {
	start := time.Now()
	
	// This would integrate with the gRPC service for actual calculation
	// For now, return a placeholder that shows the integration pattern
	
	region := "US"
	if args.Region != nil {
		region = *args.Region
	}

	// Get dosing rule
	dosingRule, err := r.DosingRule(ctx, struct {
		DrugCode string
		Version  *string
		Region   *string
	}{
		DrugCode: args.DrugCode,
		Version:  args.Version,
		Region:   &region,
	})
	if err != nil {
		return nil, err
	}
	if dosingRule == nil {
		return nil, fmt.Errorf("no dosing rules found for drug code: %s", args.DrugCode)
	}

	// Build recommendation (simplified - would use actual calculation engine)
	recommendation := &DosingRecommendationType{
		DrugCode: args.DrugCode,
		Version:  dosingRule.Version,
		ApplicableAdjustments: dosingRule.Adjustments,
		RecommendedDose: &RecommendedDoseType{
			AmountMg:  dosingRule.BaseDose.Starting,
			Frequency: dosingRule.BaseDose.Frequency,
			Route:     "oral",
			SpecialInstructions: []string{"Take with food"},
		},
		SafetyAlerts: []*SafetyAlertType{},
		MonitoringRequirements: []string{"Monitor renal function"},
		CalculationMetadata: &CalculationMetadataType{
			BasedOnRuleVersion: dosingRule.Version,
			AppliedAdjustments: []string{},
			CalculatedAt:       time.Now().Format(time.RFC3339),
			CacheHit:          false,
			ResponseTimeMs:    float64(time.Since(start).Milliseconds()),
		},
	}

	return recommendation, nil
}

func (r *Resolver) CheckDosingAvailability(ctx context.Context, args struct {
	DrugCode string
	Region   *string
}) (bool, error) {
	var count int64
	query := "SELECT COUNT(*) FROM active_dosing_rules WHERE drug_code = ?"
	
	if err := r.db.Raw(query, args.DrugCode).Scan(&count).Error; err != nil {
		r.logger.WithError(err).Error("Failed to check dosing availability")
		return false, err
	}
	
	return count > 0, nil
}

func (r *Resolver) DosingRuleMetadata(ctx context.Context, args struct {
	DrugCode string
	Version  string
}) (*ProvenanceInfoType, error) {
	var result struct {
		Provenance  json.RawMessage `json:"provenance"`
		CreatedBy   string          `json:"created_by"`
		SourceFile  string          `json:"source_file"`
		CreatedAt   time.Time       `json:"created_at"`
	}

	query := `
		SELECT provenance, created_by, 
		       COALESCE(source_file, drug_code || '_' || semantic_version || '.toml') as source_file,
		       created_at
		FROM dosing_rules 
		WHERE drug_code = ? AND semantic_version = ?
	`

	if err := r.db.Raw(query, args.DrugCode, args.Version).Scan(&result).Error; err != nil {
		return nil, err
	}

	// Parse provenance JSON
	var provenance map[string]interface{}
	if err := json.Unmarshal(result.Provenance, &provenance); err != nil {
		r.logger.WithError(err).Error("Failed to parse provenance")
		return nil, err
	}

	return &ProvenanceInfoType{
		SourceFile:       result.SourceFile,
		Authors:          stringArrayFromInterface(provenance["authors"]),
		Approvals:        stringArrayFromInterface(provenance["approvals"]),
		KB3Refs:          stringArrayFromInterface(provenance["kb3_refs"]),
		KB4Refs:          stringArrayFromInterface(provenance["kb4_refs"]),
		LastModifiedBy:   result.CreatedBy,
	}, nil
}

// Mutation resolvers

func (r *Resolver) ValidateDosingRule(ctx context.Context, args struct {
	TOMLContent string
	DrugCode    string
	Regions     []string
}) (*ValidationResultType, error) {
	start := time.Now()
	
	// Use TOML compiler for validation
	// TODO: Implement actual TOML compilation and validation

	checksumVal := "placeholder_checksum"
	result := &ValidationResultType{
		Valid:      true,
		Errors:     []string{},
		Warnings:   []string{},
		UsedFields: []string{"weight_kg", "age_years", "egfr"},
		Checksum:   &checksumVal,
	}
	
	r.logger.WithFields(logrus.Fields{
		"resolver":    "validateDosingRule",
		"drug_code":   args.DrugCode,
		"duration_ms": time.Since(start).Milliseconds(),
	}).Info("Dosing rule validation completed")

	return result, nil
}

func (r *Resolver) SubmitDosingRuleForApproval(ctx context.Context, args struct {
	DrugCode              string
	Version               string
	TOMLContent           string
	SubmittedBy           string
	ClinicalJustification string
}) (*SubmissionResultType, error) {
	// TODO: Implement governance workflow integration

	estimatedTime := "3-5 business days"
	message := "Submission received and queued for clinical review"
	return &SubmissionResultType{
		Success:             true,
		SubmissionID:        fmt.Sprintf("submission_%s_%s_%d", args.DrugCode, args.Version, time.Now().Unix()),
		Status:              "pending_clinical_review",
		EstimatedReviewTime: &estimatedTime,
		Reviewers:           []string{"Clinical Board", "Pharmacy Committee"},
		Message:             &message,
	}, nil
}

// Helper methods

func (r *Resolver) buildDosingRuleType(result *struct {
	RuleID          string          `json:"rule_id"`
	DrugCode        string          `json:"drug_code"`
	DrugName        string          `json:"drug_name"`
	SemanticVersion string          `json:"semantic_version"`
	CompiledJSON    json.RawMessage `json:"compiled_json"`
	Checksum        string          `json:"checksum"`
	Adjustments     json.RawMessage `json:"adjustments"`
	TitrationSchedule json.RawMessage `json:"titration_schedule"`
	PopulationRules json.RawMessage `json:"population_rules"`
	Provenance      json.RawMessage `json:"provenance"`
	CreatedAt       time.Time       `json:"created_at"`
}) (*DosingRuleType, error) {
	// Parse compiled JSON to extract base dose information
	var compiledRule map[string]interface{}
	if err := json.Unmarshal(result.CompiledJSON, &compiledRule); err != nil {
		return nil, fmt.Errorf("failed to parse compiled JSON: %w", err)
	}

	// Extract base dose
	baseDose := &BaseDoseType{}
	if baseDoseData, ok := compiledRule["base_dose"].(map[string]interface{}); ok {
		baseDose.Unit = getStringFromMap(baseDoseData, "unit_ucum")
		baseDose.Starting = getFloat64FromMap(baseDoseData, "starting_mg")
		baseDose.MaxDaily = getFloat64FromMap(baseDoseData, "max_daily_mg")
		baseDose.MinDaily = getFloat64FromMap(baseDoseData, "min_daily_mg")
		baseDose.Frequency = getStringFromMap(baseDoseData, "frequency_code")
	}

	// Parse adjustments
	var adjustments []*DoseAdjustmentType
	if len(result.Adjustments) > 0 {
		var adjArray []map[string]interface{}
		if err := json.Unmarshal(result.Adjustments, &adjArray); err == nil {
			for _, adj := range adjArray {
				adjustment := &DoseAdjustmentType{
					AdjustmentID:        getStringFromMap(adj, "adj_id"),
					AdjustType:          getStringFromMap(adj, "adjust_type"),
					Description:         fmt.Sprintf("%s adjustment", getStringFromMap(adj, "adjust_type")),
					ConditionExpression: getStringFromMap(adj, "condition_json"),
					FormulaExpression:   getStringFromMap(adj, "formula_json"),
					Contraindicated:     false, // TODO: Extract from condition
				}
				adjustments = append(adjustments, adjustment)
			}
		}
	}

	// Parse titration schedule
	var titrationSteps []*TitrationStepType
	if len(result.TitrationSchedule) > 0 {
		var stepArray []map[string]interface{}
		if err := json.Unmarshal(result.TitrationSchedule, &stepArray); err == nil {
			for _, step := range stepArray {
				titrationStep := &TitrationStepType{
					StepNumber:  int32(getIntFromMap(step, "step_number")),
					AfterDays:   int32(getIntFromMap(step, "after_days")),
					ActionType:  getStringFromMap(step, "action_type"),
					ActionValue: float64Ptr(getFloat64FromMap(step, "action_value")),
					MaxStep:     int32Ptr(getIntFromMap(step, "max_step")),
				}
				titrationSteps = append(titrationSteps, titrationStep)
			}
		}
	}

	// Parse population rules
	var populationRules []*PopulationRuleType
	if len(result.PopulationRules) > 0 {
		var popArray []map[string]interface{}
		if err := json.Unmarshal(result.PopulationRules, &popArray); err == nil {
			for _, pop := range popArray {
				populationRule := &PopulationRuleType{
					PopulationID:   getStringFromMap(pop, "pop_id"),
					PopulationType: getStringFromMap(pop, "population_type"),
					AgeMin:         int32Ptr(getIntFromMap(pop, "age_min")),
					AgeMax:         int32Ptr(getIntFromMap(pop, "age_max")),
					WeightMin:      float64Ptr(getFloat64FromMap(pop, "weight_min")),
					WeightMax:      float64Ptr(getFloat64FromMap(pop, "weight_max")),
					Formula:        getStringFromMap(pop, "formula_json"),
					Contraindicated: false, // TODO: Extract from safety limits
				}
				populationRules = append(populationRules, populationRule)
			}
		}
	}

	// Parse provenance
	var provenance map[string]interface{}
	if err := json.Unmarshal(result.Provenance, &provenance); err != nil {
		provenance = make(map[string]interface{})
	}

	dosingRule := &DosingRuleType{
		DrugCode:         result.DrugCode,
		Version:          result.SemanticVersion,
		DrugName:         result.DrugName,
		ContentSHA:       result.Checksum,
		SignatureValid:   true, // TODO: Get from actual signature validation
		Active:           true,
		Regions:          []string{"US", "EU", "CA", "AU"}, // TODO: Get from actual regions
		BaseDose:         baseDose,
		Adjustments:      adjustments,
		TitrationSchedule: titrationSteps,
		PopulationRules:  populationRules,
		SafetyVerification: &SafetyVerificationType{
			Contraindications: []string{}, // TODO: Extract from compiled JSON
			Warnings:          []string{},
			Precautions:       []string{},
			LabMonitoring:     []string{},
		},
		CreatedAt:        result.CreatedAt.Format(time.RFC3339),
		CreatedBy:        "system", // TODO: Get from actual creator
		Provenance: &ProvenanceInfoType{
			Authors:          stringArrayFromInterface(provenance["authors"]),
			Approvals:        stringArrayFromInterface(provenance["approvals"]),
			KB3Refs:          stringArrayFromInterface(provenance["kb3_refs"]),
			KB4Refs:          stringArrayFromInterface(provenance["kb4_refs"]),
			SourceFile:       fmt.Sprintf("%s_%s.toml", result.DrugCode, result.SemanticVersion),
			LastModifiedBy:   "system",
		},
	}

	return dosingRule, nil
}

// Helper functions

func stringArrayFromInterface(val interface{}) []string {
	if arr, ok := val.([]interface{}); ok {
		result := make([]string, len(arr))
		for i, item := range arr {
			if str, ok := item.(string); ok {
				result[i] = str
			}
		}
		return result
	}
	return []string{}
}

func getStringFromMap(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getFloat64FromMap(m map[string]interface{}, key string) float64 {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case float64:
			return v
		case string:
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				return f
			}
		}
	}
	return 0.0
}

func getIntFromMap(m map[string]interface{}, key string) int {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case int:
			return v
		case int64:
			return int(v)
		case float64:
			return int(v)
		case string:
			if i, err := strconv.Atoi(v); err == nil {
				return i
			}
		}
	}
	return 0
}

func int32Ptr(val int) *int32 {
	if val == 0 {
		return nil
	}
	i32 := int32(val)
	return &i32
}

func float64Ptr(val float64) *float64 {
	if val == 0.0 {
		return nil
	}
	return &val
}