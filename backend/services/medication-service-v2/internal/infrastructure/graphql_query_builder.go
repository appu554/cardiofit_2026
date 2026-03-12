package infrastructure

import (
	"fmt"
	"strings"
)

// GraphQLQueryBuilder provides efficient GraphQL query construction for Apollo Federation
type GraphQLQueryBuilder struct {
	fragments map[string]string
}

// NewGraphQLQueryBuilder creates a new query builder with common fragments
func NewGraphQLQueryBuilder() *GraphQLQueryBuilder {
	builder := &GraphQLQueryBuilder{
		fragments: make(map[string]string),
	}
	
	// Define reusable fragments
	builder.initializeFragments()
	
	return builder
}

// initializeFragments sets up common GraphQL fragments for reuse
func (b *GraphQLQueryBuilder) initializeFragments() {
	// Base dose fragment
	b.fragments["baseDoseFields"] = `
		fragment baseDoseFields on BaseDose {
			unit
			starting
			maxDaily
			minDaily
			frequency
			loading
			maintenance
			maxSingle
		}
	`

	// Dose adjustment fragment
	b.fragments["doseAdjustmentFields"] = `
		fragment doseAdjustmentFields on DoseAdjustment {
			adjustmentId
			adjustType
			description
			conditionExpression
			formulaExpression
			multiplier
			additiveMg
			maxDoseMg
			minDoseMg
			contraindicated
		}
	`

	// Titration step fragment
	b.fragments["titrationStepFields"] = `
		fragment titrationStepFields on TitrationStep {
			stepNumber
			afterDays
			actionType
			actionValue
			maxStep
			monitoringRequired
		}
	`

	// Population rule fragment
	b.fragments["populationRuleFields"] = `
		fragment populationRuleFields on PopulationRule {
			populationId
			populationType
			ageMin
			ageMax
			weightMin
			weightMax
			formula
			safetyLimits
			contraindicated
		}
	`

	// Safety verification fragment
	b.fragments["safetyVerificationFields"] = `
		fragment safetyVerificationFields on SafetyVerification {
			contraindications
			warnings
			precautions
			labMonitoring
		}
	`

	// Provenance info fragment
	b.fragments["provenanceFields"] = `
		fragment provenanceFields on ProvenanceInfo {
			authors
			approvals
			kb3Refs
			kb4Refs
			sourceFile
			effectiveFrom
			lastModifiedBy
		}
	`

	// Complete dosing rule fragment
	b.fragments["dosingRuleFields"] = `
		fragment dosingRuleFields on DosingRule {
			drugCode
			version
			drugName
			contentSHA
			signatureValid
			active
			regions
			baseDose {
				...baseDoseFields
			}
			adjustments {
				...doseAdjustmentFields
			}
			titrationSchedule {
				...titrationStepFields
			}
			populationRules {
				...populationRuleFields
			}
			safetyVerification {
				...safetyVerificationFields
			}
			createdAt
			createdBy
			provenance {
				...provenanceFields
			}
		}
	`

	// Recommended dose fragment
	b.fragments["recommendedDoseFields"] = `
		fragment recommendedDoseFields on RecommendedDose {
			amountMg
			frequency
			route
			duration
			specialInstructions
		}
	`

	// Safety alert fragment
	b.fragments["safetyAlertFields"] = `
		fragment safetyAlertFields on SafetyAlert {
			alertType
			severity
			message
			actionRequired
		}
	`

	// Calculation metadata fragment
	b.fragments["calculationMetadataFields"] = `
		fragment calculationMetadataFields on CalculationMetadata {
			basedOnRuleVersion
			appliedAdjustments
			calculatedAt
			cacheHit
			responseTimeMs
		}
	`

	// Dosing recommendation fragment
	b.fragments["dosingRecommendationFields"] = `
		fragment dosingRecommendationFields on DosingRecommendation {
			drugCode
			version
			applicableAdjustments {
				...doseAdjustmentFields
			}
			recommendedDose {
				...recommendedDoseFields
			}
			safetyAlerts {
				...safetyAlertFields
			}
			monitoringRequirements
			calculationMetadata {
				...calculationMetadataFields
			}
		}
	`

	// Clinical guideline fragments
	b.fragments["recommendationFields"] = `
		fragment recommendationFields on Recommendation {
			recommendationId
			text
			strength
			quality
			categories
			conditions
			evidence
		}
	`

	b.fragments["guidelineMetadataFields"] = `
		fragment guidelineMetadataFields on GuidelineMetadata {
			authors
			reviewers
			doi
			pubmedId
			lastUpdated
			nextReview
			tags
			sourceUrl
		}
	`

	b.fragments["clinicalGuidelineFields"] = `
		fragment clinicalGuidelineFields on ClinicalGuideline {
			guidelineId
			title
			version
			organization
			publicationDate
			recommendations {
				...recommendationFields
			}
			evidenceLevel
			categories
			drugClasses
			conditions
			metadata {
				...guidelineMetadataFields
			}
		}
	`
}

// BuildDosingRuleQuery builds a query for retrieving a single dosing rule
func (b *GraphQLQueryBuilder) BuildDosingRuleQuery() string {
	query := `
		query GetDosingRule($drugCode: String!, $version: String, $region: String) {
			dosingRule(drugCode: $drugCode, version: $version, region: $region) {
				...dosingRuleFields
			}
		}
	`
	
	return b.assembleQueryWithFragments(query, []string{
		"dosingRuleFields",
		"baseDoseFields", 
		"doseAdjustmentFields",
		"titrationStepFields",
		"populationRuleFields",
		"safetyVerificationFields",
		"provenanceFields",
	})
}

// BuildBatchDosingRulesQuery builds a query for retrieving multiple dosing rules
func (b *GraphQLQueryBuilder) BuildBatchDosingRulesQuery() string {
	query := `
		query GetBatchDosingRules($drugCodes: [String!], $regions: [String!], $first: Int, $after: String) {
			dosingRules(drugCodes: $drugCodes, regions: $regions, activeOnly: true, first: $first, after: $after) {
				edges {
					node {
						...dosingRuleFields
					}
					cursor
				}
				pageInfo {
					hasNextPage
					hasPreviousPage
					startCursor
					endCursor
				}
				totalCount
			}
		}
	`

	return b.assembleQueryWithFragments(query, []string{
		"dosingRuleFields",
		"baseDoseFields",
		"doseAdjustmentFields", 
		"titrationStepFields",
		"populationRuleFields",
		"safetyVerificationFields",
		"provenanceFields",
	})
}

// BuildCalculateDosingQuery builds a query for calculating dosing recommendations
func (b *GraphQLQueryBuilder) BuildCalculateDosingQuery() string {
	query := `
		query CalculateDosing($drugCode: String!, $patientContext: PatientContextInput!, $version: String, $region: String) {
			calculateDosing(drugCode: $drugCode, patientContext: $patientContext, version: $version, region: $region) {
				...dosingRecommendationFields
			}
		}
	`

	return b.assembleQueryWithFragments(query, []string{
		"dosingRecommendationFields",
		"doseAdjustmentFields",
		"recommendedDoseFields",
		"safetyAlertFields",
		"calculationMetadataFields",
	})
}

// BuildClinicalGuidelinesQuery builds a query for retrieving clinical guidelines
func (b *GraphQLQueryBuilder) BuildClinicalGuidelinesQuery() string {
	query := `
		query GetClinicalGuidelines($drugClass: String, $condition: String, $first: Int, $categories: [String!]) {
			clinicalGuidelines(drugClass: $drugClass, condition: $condition, first: $first, categories: $categories) {
				...clinicalGuidelineFields
			}
		}
	`

	return b.assembleQueryWithFragments(query, []string{
		"clinicalGuidelineFields",
		"recommendationFields",
		"guidelineMetadataFields",
	})
}

// BuildAvailabilityCheckQuery builds a simple availability check query
func (b *GraphQLQueryBuilder) BuildAvailabilityCheckQuery() string {
	return `
		query CheckDosingAvailability($drugCode: String!, $region: String) {
			checkDosingAvailability(drugCode: $drugCode, region: $region)
		}
	`
}

// BuildHealthCheckQuery builds a simple health check query
func (b *GraphQLQueryBuilder) BuildHealthCheckQuery() string {
	return `
		query HealthCheck {
			serviceInfo {
				name
				version
				status
			}
		}
	`
}

// BuildValidateRuleQuery builds a mutation for validating TOML dosing rules
func (b *GraphQLQueryBuilder) BuildValidateRuleQuery() string {
	return `
		mutation ValidateDosingRule($tomlContent: String!, $drugCode: String!, $regions: [String!]!) {
			validateDosingRule(tomlContent: $tomlContent, drugCode: $drugCode, regions: $regions) {
				valid
				errors
				warnings
				compiledJSON
				usedFields
				checksum
			}
		}
	`
}

// BuildSubmitRuleQuery builds a mutation for submitting dosing rules for approval
func (b *GraphQLQueryBuilder) BuildSubmitRuleQuery() string {
	return `
		mutation SubmitDosingRuleForApproval(
			$drugCode: String!,
			$version: String!,
			$tomlContent: String!,
			$submittedBy: String!,
			$clinicalJustification: String!
		) {
			submitDosingRuleForApproval(
				drugCode: $drugCode,
				version: $version,
				tomlContent: $tomlContent,
				submittedBy: $submittedBy,
				clinicalJustification: $clinicalJustification
			) {
				success
				submissionId
				status
				estimatedReviewTime
				reviewers
				message
			}
		}
	`
}

// BuildComprehensiveClinicalQuery builds a complex query for comprehensive clinical intelligence
func (b *GraphQLQueryBuilder) BuildComprehensiveClinicalQuery() string {
	query := `
		query GetComprehensiveClinicalIntelligence(
			$drugCode: String!,
			$patientContext: PatientContextInput,
			$version: String,
			$region: String,
			$guidelineLimit: Int
		) {
			# Dosing information
			dosingRule(drugCode: $drugCode, version: $version, region: $region) {
				...dosingRuleFields
			}
			
			# Patient-specific dosing recommendation if context provided
			calculateDosing(drugCode: $drugCode, patientContext: $patientContext, version: $version, region: $region) @skip(if: false) {
				...dosingRecommendationFields
			}
			
			# Check availability
			checkDosingAvailability(drugCode: $drugCode, region: $region)
			
			# Related clinical guidelines
			clinicalGuidelines(first: $guidelineLimit, categories: ["dosing", "safety", "monitoring"]) {
				...clinicalGuidelineFields
			}
			
			# Get metadata if available
			dosingRuleMetadata(drugCode: $drugCode, version: $version) {
				...provenanceFields
			}
		}
	`

	return b.assembleQueryWithFragments(query, []string{
		"dosingRuleFields",
		"baseDoseFields",
		"doseAdjustmentFields",
		"titrationStepFields", 
		"populationRuleFields",
		"safetyVerificationFields",
		"provenanceFields",
		"dosingRecommendationFields",
		"recommendedDoseFields",
		"safetyAlertFields",
		"calculationMetadataFields",
		"clinicalGuidelineFields",
		"recommendationFields",
		"guidelineMetadataFields",
	})
}

// BuildDrugInteractionQuery builds a query for drug interactions (future KB5 integration)
func (b *GraphQLQueryBuilder) BuildDrugInteractionQuery() string {
	return `
		query GetDrugInteractions($drugCode: String!, $otherDrugs: [String!], $severity: String) {
			drugInteractions(drugCode: $drugCode, otherDrugs: $otherDrugs, minSeverity: $severity) {
				interactionId
				drugA {
					code
					name
				}
				drugB {
					code  
					name
				}
				severity
				mechanism
				clinicalEffect
				management
				evidence {
					level
					studies
					references
				}
				lastUpdated
			}
		}
	`
}

// BuildPatientSafetyQuery builds a query for patient safety checks (future KB4 integration)  
func (b *GraphQLQueryBuilder) BuildPatientSafetyQuery() string {
	return `
		query CheckPatientSafety(
			$drugCode: String!,
			$patientContext: PatientContextInput!,
			$checkTypes: [String!]
		) {
			patientSafetyCheck(
				drugCode: $drugCode, 
				patientContext: $patientContext,
				checkTypes: $checkTypes
			) {
				safetyAlerts {
					alertType
					severity
					message
					category
					actionRequired
					contraindicated
				}
				riskScore
				riskFactors
				recommendedMonitoring
				labValues {
					parameter
					current
					target
					units
					priority
				}
				approvedForPatient
				requiresApproval
				metadata {
					rulesApplied
					calculatedAt
					version
				}
			}
		}
	`
}

// assembleQueryWithFragments combines a query with its required fragments
func (b *GraphQLQueryBuilder) assembleQueryWithFragments(query string, fragmentNames []string) string {
	var fragments []string
	
	// Collect unique fragments and their dependencies
	seen := make(map[string]bool)
	
	for _, fragName := range fragmentNames {
		if fragment, exists := b.fragments[fragName]; exists && !seen[fragName] {
			fragments = append(fragments, fragment)
			seen[fragName] = true
		}
	}
	
	if len(fragments) == 0 {
		return query
	}
	
	return query + "\n\n" + strings.Join(fragments, "\n\n")
}

// GetFragment returns a specific fragment by name
func (b *GraphQLQueryBuilder) GetFragment(name string) (string, bool) {
	fragment, exists := b.fragments[name]
	return fragment, exists
}

// AddCustomFragment adds a custom fragment to the builder
func (b *GraphQLQueryBuilder) AddCustomFragment(name, fragment string) {
	b.fragments[name] = fragment
}

// BuildOptimizedBatchQuery builds an optimized query for multiple drugs with selective fields
func (b *GraphQLQueryBuilder) BuildOptimizedBatchQuery(fields []string) string {
	// Build field selection based on requirements
	fieldSelection := b.buildFieldSelection(fields)
	
	query := fmt.Sprintf(`
		query GetOptimizedBatchData($drugCodes: [String!]!, $region: String, $limit: Int) {
			dosingRules(drugCodes: $drugCodes, regions: [$region], activeOnly: true, first: $limit) {
				edges {
					node {
						%s
					}
				}
				totalCount
			}
		}
	`, fieldSelection)

	return query
}

// buildFieldSelection creates optimized field selection based on requirements
func (b *GraphQLQueryBuilder) buildFieldSelection(fields []string) string {
	fieldMap := map[string]string{
		"basic": `
			drugCode
			version
			drugName
			active
		`,
		"dosing": `
			drugCode
			version
			drugName  
			baseDose {
				unit
				starting
				maxDaily
				frequency
			}
		`,
		"safety": `
			drugCode
			version
			safetyVerification {
				contraindications
				warnings
				precautions
			}
		`,
		"complete": `
			...dosingRuleFields
		`,
	}

	if len(fields) == 0 || (len(fields) == 1 && fields[0] == "complete") {
		return fieldMap["complete"]
	}

	var selectedFields []string
	for _, field := range fields {
		if fieldDef, exists := fieldMap[field]; exists {
			selectedFields = append(selectedFields, fieldDef)
		}
	}

	if len(selectedFields) == 0 {
		return fieldMap["basic"]
	}

	return strings.Join(selectedFields, "\n")
}

// ValidateQuery performs basic GraphQL query validation
func (b *GraphQLQueryBuilder) ValidateQuery(query string) error {
	// Basic validation checks
	if strings.TrimSpace(query) == "" {
		return fmt.Errorf("query cannot be empty")
	}

	if !strings.Contains(query, "query") && !strings.Contains(query, "mutation") && !strings.Contains(query, "subscription") {
		return fmt.Errorf("query must contain a valid GraphQL operation")
	}

	// Check for balanced braces
	openBraces := strings.Count(query, "{")
	closeBraces := strings.Count(query, "}")
	if openBraces != closeBraces {
		return fmt.Errorf("unbalanced braces in query: %d open, %d close", openBraces, closeBraces)
	}

	return nil
}

// GetQueryComplexity estimates query complexity for rate limiting
func (b *GraphQLQueryBuilder) GetQueryComplexity(query string) int {
	// Simple complexity calculation based on depth and field count
	complexity := 0
	
	// Count nested levels (approximate)
	maxDepth := 0
	currentDepth := 0
	
	for _, char := range query {
		switch char {
		case '{':
			currentDepth++
			if currentDepth > maxDepth {
				maxDepth = currentDepth
			}
		case '}':
			currentDepth--
		}
	}
	
	// Count field selections (lines with field names)
	lines := strings.Split(query, "\n")
	fieldCount := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) > 0 && !strings.HasPrefix(trimmed, "#") && 
		   !strings.Contains(trimmed, "query") && !strings.Contains(trimmed, "mutation") &&
		   !strings.Contains(trimmed, "fragment") && !strings.Contains(trimmed, "}") {
			fieldCount++
		}
	}
	
	complexity = maxDepth * fieldCount
	
	// Apply multipliers for expensive operations
	if strings.Contains(query, "calculateDosing") {
		complexity *= 3 // Calculation queries are more expensive
	}
	if strings.Contains(query, "dosingRules") && strings.Contains(query, "first:") {
		complexity *= 2 // Batch queries are more expensive
	}
	
	return complexity
}