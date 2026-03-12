package federation

import (
	"context"
	"fmt"
)

// GraphQL types for Apollo Federation integration
// These types correspond to the GraphQL schema defined in schema.go

// Ensure context import is used
var _ = context.Background

// DosingRuleType represents a dosing rule in GraphQL
type DosingRuleType struct {
	DrugCode           string                     `json:"drugCode"`
	Version            string                     `json:"version"`
	DrugName           string                     `json:"drugName"`
	ContentSHA         string                     `json:"contentSHA"`
	SignatureValid     bool                       `json:"signatureValid"`
	Active             bool                       `json:"active"`
	Regions            []string                   `json:"regions"`
	BaseDose           *BaseDoseType              `json:"baseDose"`
	Adjustments        []*DoseAdjustmentType      `json:"adjustments"`
	TitrationSchedule  []*TitrationStepType       `json:"titrationSchedule"`
	PopulationRules    []*PopulationRuleType      `json:"populationRules"`
	SafetyVerification *SafetyVerificationType    `json:"safetyVerification"`
	CreatedAt          string                     `json:"createdAt"`
	CreatedBy          string                     `json:"createdBy"`
	Provenance         *ProvenanceInfoType        `json:"provenance"`
}

// BaseDoseType represents base dosing information
type BaseDoseType struct {
	Unit        string  `json:"unit"`
	Starting    float64 `json:"starting"`
	MaxDaily    float64 `json:"maxDaily"`
	MinDaily    float64 `json:"minDaily"`
	Frequency   string  `json:"frequency"`
	Loading     *string `json:"loading"`
	Maintenance *string `json:"maintenance"`
	MaxSingle   *float64 `json:"maxSingle"`
}

// DoseAdjustmentType represents a dose adjustment rule
type DoseAdjustmentType struct {
	AdjustmentID        string   `json:"adjustmentId"`
	AdjustType          string   `json:"adjustType"`
	Description         string   `json:"description"`
	ConditionExpression string   `json:"conditionExpression"`
	FormulaExpression   string   `json:"formulaExpression"`
	Multiplier          *float64 `json:"multiplier"`
	AdditiveMg          *float64 `json:"additiveMg"`
	MaxDoseMg           *float64 `json:"maxDoseMg"`
	MinDoseMg           *float64 `json:"minDoseMg"`
	Contraindicated     bool     `json:"contraindicated"`
}

// TitrationStepType represents a titration step
type TitrationStepType struct {
	StepNumber         int32   `json:"stepNumber"`
	AfterDays          int32   `json:"afterDays"`
	ActionType         string  `json:"actionType"`
	ActionValue        *float64 `json:"actionValue"`
	MaxStep            *int32  `json:"maxStep"`
	MonitoringRequired *string `json:"monitoringRequired"`
}

// PopulationRuleType represents population-specific dosing
type PopulationRuleType struct {
	PopulationID   string   `json:"populationId"`
	PopulationType string   `json:"populationType"`
	AgeMin         *int32   `json:"ageMin"`
	AgeMax         *int32   `json:"ageMax"`
	WeightMin      *float64 `json:"weightMin"`
	WeightMax      *float64 `json:"weightMax"`
	Formula        string   `json:"formula"`
	SafetyLimits   *string  `json:"safetyLimits"`
	Contraindicated bool    `json:"contraindicated"`
}

// SafetyVerificationType represents safety verification information
type SafetyVerificationType struct {
	Contraindications []string `json:"contraindications"`
	Warnings          []string `json:"warnings"`
	Precautions       []string `json:"precautions"`
	LabMonitoring     []string `json:"labMonitoring"`
}

// ProvenanceInfoType represents rule provenance and governance information
type ProvenanceInfoType struct {
	Authors          []string `json:"authors"`
	Approvals        []string `json:"approvals"`
	KB3Refs          []string `json:"kb3Refs"`
	KB4Refs          []string `json:"kb4Refs"`
	SourceFile       string   `json:"sourceFile"`
	EffectiveFrom    *string  `json:"effectiveFrom"`
	LastModifiedBy   string   `json:"lastModifiedBy"`
}

// Connection types for pagination

// DosingRuleConnectionType represents paginated dosing rules
type DosingRuleConnectionType struct {
	Edges      []*DosingRuleEdgeType `json:"edges"`
	PageInfo   *PageInfoType         `json:"pageInfo"`
	TotalCount int32                 `json:"totalCount"`
}

// DosingRuleEdgeType represents an edge in the dosing rules connection
type DosingRuleEdgeType struct {
	Node   *DosingRuleType `json:"node"`
	Cursor string          `json:"cursor"`
}

// PageInfoType represents pagination information
type PageInfoType struct {
	HasNextPage     bool    `json:"hasNextPage"`
	HasPreviousPage bool    `json:"hasPreviousPage"`
	StartCursor     *string `json:"startCursor"`
	EndCursor       *string `json:"endCursor"`
}

// Input types

// PatientContextInputType represents patient context input for GraphQL
type PatientContextInputType struct {
	WeightKg             float64                  `json:"weightKg"`
	EGFR                 float64                  `json:"egfr"`
	AgeYears             int32                    `json:"ageYears"`
	Sex                  string                   `json:"sex"`
	Pregnant             *bool                    `json:"pregnant"`
	CreatinineClearance  *float64                 `json:"creatinineClearance"`
	DialysisType         *string                  `json:"dialysisType"`
	ExtraNumeric         []*ExtraNumericInputType `json:"extraNumeric"`
}

// ExtraNumericInputType represents additional numeric parameters
type ExtraNumericInputType struct {
	Key   string  `json:"key"`
	Value float64 `json:"value"`
}

// Response types for dosing calculations

// DosingRecommendationType represents a complete dosing recommendation
type DosingRecommendationType struct {
	DrugCode               string                     `json:"drugCode"`
	Version                string                     `json:"version"`
	ApplicableAdjustments  []*DoseAdjustmentType      `json:"applicableAdjustments"`
	RecommendedDose        *RecommendedDoseType       `json:"recommendedDose"`
	SafetyAlerts           []*SafetyAlertType         `json:"safetyAlerts"`
	MonitoringRequirements []string                   `json:"monitoringRequirements"`
	CalculationMetadata    *CalculationMetadataType   `json:"calculationMetadata"`
}

// RecommendedDoseType represents the final dose recommendation
type RecommendedDoseType struct {
	AmountMg            float64  `json:"amountMg"`
	Frequency           string   `json:"frequency"`
	Route               string   `json:"route"`
	Duration            *string  `json:"duration"`
	SpecialInstructions []string `json:"specialInstructions"`
}

// SafetyAlertType represents a safety alert
type SafetyAlertType struct {
	AlertType      string  `json:"alertType"`
	Severity       string  `json:"severity"`
	Message        string  `json:"message"`
	ActionRequired *string `json:"actionRequired"`
}

// CalculationMetadataType represents metadata about the dose calculation
type CalculationMetadataType struct {
	BasedOnRuleVersion string   `json:"basedOnRuleVersion"`
	AppliedAdjustments []string `json:"appliedAdjustments"`
	CalculatedAt       string   `json:"calculatedAt"`
	CacheHit           bool     `json:"cacheHit"`
	ResponseTimeMs     float64  `json:"responseTimeMs"`
}

// Validation and submission types

// ValidationResultType represents TOML validation results
type ValidationResultType struct {
	Valid        bool     `json:"valid"`
	Errors       []string `json:"errors"`
	Warnings     []string `json:"warnings"`
	CompiledJSON *string  `json:"compiledJSON"`
	UsedFields   []string `json:"usedFields"`
	Checksum     *string  `json:"checksum"`
}

// SubmissionResultType represents governance submission results
type SubmissionResultType struct {
	Success             bool     `json:"success"`
	SubmissionID        string   `json:"submissionId"`
	Status              string   `json:"status"`
	EstimatedReviewTime *string  `json:"estimatedReviewTime"`
	Reviewers           []string `json:"reviewers"`
	Message             *string  `json:"message"`
}

// Federation key resolvers

// __resolveReference resolves entity references for Apollo Federation
func (r *Resolver) __resolveReference(ctx context.Context, args struct {
	DrugCode string `json:"drugCode"`
	Version  string `json:"version"`
}) (*DosingRuleType, error) {
	return r.DosingRule(ctx, struct {
		DrugCode string
		Version  *string
		Region   *string
	}{
		DrugCode: args.DrugCode,
		Version:  &args.Version,
		Region:   nil,
	})
}

// Federation directive implementations

// Key directive implementation for DosingRule entity
func (d *DosingRuleType) Key() string {
	return fmt.Sprintf("%s:%s", d.DrugCode, d.Version)
}

// EntityResolvers for Apollo Federation
type EntityResolvers struct {
	resolver *Resolver
}

// NewEntityResolvers creates entity resolvers for federation
func NewEntityResolvers(resolver *Resolver) *EntityResolvers {
	return &EntityResolvers{resolver: resolver}
}

// ResolveDosingRule resolves DosingRule entities by key
func (e *EntityResolvers) ResolveDosingRule(ctx context.Context, key map[string]interface{}) (*DosingRuleType, error) {
	drugCode, ok := key["drugCode"].(string)
	if !ok {
		return nil, fmt.Errorf("drugCode not found in entity key")
	}
	
	version, ok := key["version"].(string)
	if !ok {
		return nil, fmt.Errorf("version not found in entity key")
	}

	return e.resolver.DosingRule(ctx, struct {
		DrugCode string
		Version  *string
		Region   *string
	}{
		DrugCode: drugCode,
		Version:  &version,
		Region:   nil,
	})
}